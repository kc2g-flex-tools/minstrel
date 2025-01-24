package audio

import (
	"encoding/binary"
	"log"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
	"gopkg.in/hraban/opus.v2"
)

type Audio struct {
	Context  *pulse.Client
	Opus     *opus.Decoder
	Player   *pulse.PlaybackStream
	s16Buf   [512]int16
	f32buf   [4]byte
	cbuf     *CircularBuf[[4]byte]
	cbufSize int
	wakeup   chan struct{}
}

func NewAudio() *Audio {
	audio := &Audio{
		cbuf:     NewCircularBuf[[4]byte](2880),
		cbufSize: 2880, // max cbuf latency: 120ms
		wakeup:   make(chan struct{}),
	}
	pc, err := pulse.NewClient(
		pulse.ClientApplicationName("Minstrel"),
	)
	if err != nil {
		panic(err)
	}
	audio.Context = pc
	audio.Player, err = pc.NewPlayback(
		pulse.NewReader(audio, proto.FormatFloat32LE),
		pulse.PlaybackChannels(proto.ChannelMap{proto.ChannelMono}),
		pulse.PlaybackLatency(50.0/1000),
		pulse.PlaybackSampleRate(24000),
	)
	if err != nil {
		panic(err)
	}

	opus, err := opus.NewDecoder(24000, 1)
	if err != nil {
		panic(err)
	}
	audio.Opus = opus
	return audio
}

func (a *Audio) Decode(data []byte) {
	n, err := a.Opus.Decode(data, a.s16Buf[:])
	if err != nil {
		log.Println(err)
	}
	for i := 0; i < n; i++ {
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
