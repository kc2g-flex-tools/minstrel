package midi

import (
	"context"
	"fmt"
	"log"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"

	"github.com/kc2g-flex-tools/minstrel/events"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
)

type Config struct {
	Enabled         bool
	Port            string
	VFOControl      byte
	VolControl      byte
	LeftPaddleNote  byte
	RightPaddleNote byte
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:         false,
		Port:            "XIAO_ESP32S3",
		VFOControl:      100,
		VolControl:      101,
		LeftPaddleNote:  20,
		RightPaddleNote: 21,
	}
}

type MIDI struct {
	cfg      *Config
	eventBus *events.Bus
}

func NewMIDI(cfg *Config, eventBus *events.Bus) *MIDI {
	return &MIDI{
		cfg:      cfg,
		eventBus: eventBus,
	}
}

func (m *MIDI) Run(ctx context.Context, rs radioshim.Shim) {
	if !m.cfg.Enabled {
		return
	}
	defer midi.CloseDriver()
	in, err := midi.FindInPort(m.cfg.Port)
	if err != nil {
		log.Fatalf("opening midi: %s", err)
		return
	}
	cancel, err := midi.ListenTo(in, func(msg midi.Message, timestamp int32) {
		var ch, id, val byte
		switch {
		case msg.GetNoteStart(&ch, &id, &val):
			fmt.Printf("keydown key=%d vel=%d\n", id, val)
		case msg.GetNoteEnd(&ch, &id):
			fmt.Printf("keyup key=%d\n", id)
		case msg.GetControlChange(&ch, &id, &val):
			fmt.Printf("control id=%d val=%d\n", id, val)
			switch {
			case id == 100:
				delta := int(val) - 64
				slices := rs.GetSlices()
				for _, slice := range slices {
					if slice.Active {
						rs.TuneSliceStep(slice, delta)
					}
				}
			}
		default:
			fmt.Printf("unknown midi message: %v\n", msg)
		}
	})
	if err != nil {
		log.Fatalf("starting midi loop: %s", err)
		return
	}
	defer cancel()
	<-ctx.Done()
}
