package audio

import (
	"encoding/binary"
	"log"
	"time"

	ebaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/smallnest/ringbuffer"
	"gopkg.in/hraban/opus.v2"
)

type Audio struct {
	Context *ebaudio.Context
	Buffer  *ringbuffer.RingBuffer
	Opus    *opus.Decoder
	Player  *ebaudio.Player
	PCMBuf  [1440]int16
}

func NewAudio() *Audio {
	audio := &Audio{
		Context: ebaudio.NewContext(24000),
		Buffer:  ringbuffer.New(64 * 1024).SetBlocking(true).WithReadTimeout(500 * time.Millisecond),
	}
	opus, err := opus.NewDecoder(24000, 1)
	if err != nil {
		panic(err)
	}
	audio.Opus = opus
	audio.Player, err = audio.Context.NewPlayerF32(audio.Buffer)
	if err != nil {
		panic(err)
	}
	return audio
}

func (a *Audio) Decode(data []byte) {
	n, err := a.Opus.Decode(data, a.PCMBuf[:])
	if err != nil {
		log.Println(err)
	}
	for i := 0; i < n; i++ {
		f32 := float32(a.PCMBuf[i]) / 32768
		binary.Write(a.Buffer, binary.LittleEndian, f32)
		binary.Write(a.Buffer, binary.LittleEndian, f32)
	}
}
