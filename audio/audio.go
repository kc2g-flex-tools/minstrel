package audio

import (
	"encoding/binary"
	"log"
	"math"
	"sync"

	"github.com/hb9fxq/flexlib-go/vita"
	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
	"github.com/kc2g-flex-tools/flexclient"
	"github.com/kc2g-flex-tools/minstrel/opus"
	"github.com/kc2g-flex-tools/minstrel/types"
	opuslib "gopkg.in/hraban/opus.v2"
)

// VitaOpusPacket represents a VITA packet for Opus audio transmission
type VitaOpusPacket struct {
	header   vita.VitaHeader
	streamID types.StreamID
	classID  vita.VitaClassID
	payload  []byte
}

// TXAudioWriter implements pulse.Writer for TX audio processing
type TXAudioWriter struct {
	audio *Audio
}

func (w *TXAudioWriter) Write(data []byte) (int, error) {
	return w.audio.processTXAudio(data)
}

func (w *TXAudioWriter) Format() byte {
	return proto.FormatFloat32LE
}

type Audio struct {
	Context  *pulse.Client
	Opus     *opuslib.Decoder
	OpusEnc  *opus.Encoder
	player   *pulse.PlaybackStream
	recorder *pulse.RecordStream
	s16Buf   [512]int16
	f32buf   [4]byte
	cbuf     *CircularBuf[[4]byte]
	cbufSize int
	wakeup   chan struct{}

	// TX audio fields
	txMutex    sync.Mutex
	txRunning  bool
	txClient   *flexclient.FlexClient
	txStreamID *types.StreamID
	txPacket   *VitaOpusPacket
	txSeq      uint16
	txWriter   *TXAudioWriter
}

func NewAudio() *Audio {
	audio := &Audio{
		cbuf:     NewCircularBuf[[4]byte](2880),
		cbufSize: 2880, // max cbuf latency: 240ms
		wakeup:   make(chan struct{}),
	}
	pc, err := pulse.NewClient(
		pulse.ClientApplicationName("Minstrel"),
	)
	if err != nil {
		panic(err)
	}
	audio.Context = pc
	audio.player, err = pc.NewPlayback(
		pulse.NewReader(audio, proto.FormatFloat32LE),
		pulse.PlaybackChannels(proto.ChannelMap{proto.ChannelMono}),
		pulse.PlaybackLatency(50.0/1000),
		pulse.PlaybackSampleRate(24000),
	)
	if err != nil {
		panic(err)
	}

	opusDecoder, err := opuslib.NewDecoder(24000, 1)
	if err != nil {
		panic(err)
	}
	audio.Opus = opusDecoder

	// Create Opus encoder for TX
	opusEncoder, err := opus.NewEncoder(24000, 2, opus.ApplicationAudio)
	if err != nil {
		panic(err)
	}
	err = opusEncoder.SetBitrate(70000)
	if err != nil {
		log.Println("Failed to set opus bitrate:", err)
	}
	err = opusEncoder.SetComplexity(1)
	if err != nil {
		log.Println("Failed to set opus complexity:", err)
	}

	audio.OpusEnc = opusEncoder

	// Create TX audio writer
	audio.txWriter = &TXAudioWriter{audio: audio}

	return audio
}

// StartTX starts transmit audio recording and encoding
func (a *Audio) StartTX(client *flexclient.FlexClient, streamID *types.StreamID) {
	a.txMutex.Lock()
	defer a.txMutex.Unlock()

	if a.txRunning {
		return
	}

	a.txClient = client
	a.txStreamID = streamID
	a.txRunning = true
	a.txSeq = 0

	// Create PulseAudio recorder
	var err error
	a.recorder, err = a.Context.NewRecord(
		a.txWriter,
		pulse.RecordStereo,
		pulse.RecordSampleRate(24000),
		pulse.RecordRawOption(func(rs *proto.CreateRecordStream) {
			// Ask pulse to give us exactly 10ms frames (240 samples at 24kHz) * 2ch * 4bytes/sample = 1920 bytes
			// Opus encoding will fail if the frame size isn't exactly a supported size.
			rs.BufferFragSize = 3840
		}),
	)
	if err != nil {
		log.Println("Failed to create recorder:", err)
		a.txRunning = false
		return
	}

	a.recorder.Start()
	log.Println("TX audio started")
}

// StopTX stops transmit audio recording
func (a *Audio) StopTX() {
	a.txMutex.Lock()
	defer a.txMutex.Unlock()

	if !a.txRunning {
		return
	}

	a.txRunning = false
	if a.recorder != nil {
		a.recorder.Stop()
		a.recorder.Close()
		a.recorder = nil
	}
	log.Println("TX audio stopped")
}

// processTXAudio processes incoming audio data for transmission
func (a *Audio) processTXAudio(data []byte) (int, error) {
	if !a.txRunning || a.txClient == nil || a.txStreamID == nil || !a.txStreamID.IsValid() {
		return len(data), nil
	}

	// Encode with Opus
	opusData, err := a.OpusEnc.EncodeFloatRaw(data)
	if err != nil {
		log.Println("Opus encoding error:", err)
		return len(data), nil
	}

	// Send via VITA packet
	a.sendVitaOpusPacket(opusData)

	return len(data), nil
}

// sendVitaOpusPacket sends Opus data as a VITA packet
func (a *Audio) sendVitaOpusPacket(opusData []byte) {
	if a.txPacket == nil {
		a.txPacket = &VitaOpusPacket{
			header: vita.VitaHeader{
				Pkt_type:     vita.ExtDataWithStream,
				C:            true,
				T:            false,
				Tsi:          vita.Other,
				Tsf:          vita.SampleCount,
				Packet_count: 0,
			},
			streamID: *a.txStreamID,
			classID: vita.VitaClassID{
				OUI:                  0x001C2D,
				InformationClassCode: 0x534C,
				PacketClassCode:      0x8005,
			},
		}
	}

	a.txPacket.payload = opusData
	a.txPacket.header.Packet_size = uint16(math.Ceil(float64(len(opusData))/4.0) + 7.0) // 7*4=28 bytes VITA overhead
	a.txPacket.header.Packet_count = a.txSeq
	a.txSeq = (a.txSeq + 1) % 16

	// Convert to bytes and send
	packetBytes := a.txPacket.ToBytes()
	err := a.txClient.SendUdp(packetBytes)
	if err != nil {
		log.Println("Failed to send UDP packet:", err)
	}
}

// ToBytes converts VitaOpusPacket to byte array for transmission
func (p *VitaOpusPacket) ToBytes() []byte {
	buf := make([]byte, 28+len(p.payload)) // 28 bytes header + payload

	// VITA header (simplified)
	headerWord := uint32(p.header.Pkt_type) << 28
	if p.header.C {
		headerWord |= 1 << 27
	}
	if p.header.T {
		headerWord |= 1 << 26
	}
	headerWord |= uint32(p.header.Tsi) << 22
	headerWord |= uint32(p.header.Tsf) << 20
	headerWord |= uint32(p.header.Packet_count) << 16
	headerWord |= uint32(p.header.Packet_size)

	binary.BigEndian.PutUint32(buf[0:4], headerWord)
	binary.BigEndian.PutUint32(buf[4:8], uint32(p.streamID))
	binary.BigEndian.PutUint32(buf[8:12], p.classID.OUI&0x00FFFFFF)
	binary.BigEndian.PutUint32(buf[12:16], uint32(p.classID.InformationClassCode)<<16|uint32(p.classID.PacketClassCode))

	// Timestamps (set to 0 for now)
	binary.BigEndian.PutUint32(buf[16:20], 0) // integer timestamp
	binary.BigEndian.PutUint64(buf[20:28], 0) // fractional timestamp

	// Payload
	copy(buf[28:], p.payload)

	return buf
}

func (a *Audio) Decode(data []byte) {
	n, err := a.Opus.Decode(data, a.s16Buf[:])
	if err != nil {
		log.Println(err)
	}
	for i := range n {
		if a.cbuf.Size() > a.cbufSize-4 {
			log.Println("audio cbuf overflow")
			return
		}
		f32 := float32(a.s16Buf[i]) / 32768
		binary.Append(a.f32buf[:0:4], binary.LittleEndian, f32)
		a.cbuf.Insert(a.f32buf)
	}
	if n > 0 {
		select {
		case a.wakeup <- struct{}{}:
		default:
		}
	}
}

func (a *Audio) Read(dest []byte) (n int, err error) {
	for n < len(dest) {
		chunk, ok := a.cbuf.PopFront()
		if !ok {
			if n > 0 {
				return
			}
			// always return at least one sample. If we can't do that, wait for the buffer to fill.
			<-a.wakeup
			continue
		}
		copy(dest[n:n+4], chunk[:])
		n += 4
	}
	return
}

func (a *Audio) Start() {
	a.cbuf.Clear()
	a.player.Start()
}

func (a *Audio) Pause() {
	a.player.Stop()
}
