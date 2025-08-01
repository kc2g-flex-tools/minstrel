package main

import (
	"context"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sources/env"
	"github.com/vimeo/dials/sources/flag"

	"github.com/kc2g-flex-tools/minstrel/audio"
	"github.com/kc2g-flex-tools/minstrel/events"
	"github.com/kc2g-flex-tools/minstrel/midi"
	"github.com/kc2g-flex-tools/minstrel/ui"
)

type Config struct {
	Station string `dialsdesc:"Station name"`
	Profile string `dialsdesc:"Global profile to load on startup"`
	UI      *ui.Config
	MIDI    *midi.Config
}

var config *Config

func defaultConfig() *Config {
	stationName := "Minstrel"
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		stationName = hostname
	}

	return &Config{
		Station: stationName,
		Profile: "",
		UI:      ui.DefaultConfig(),
		MIDI:    midi.DefaultConfig(),
	}
}

func main() {
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	config = defaultConfig()
	flagSrc, err := flag.NewCmdLineSet(flag.DefaultFlagNameConfig(), config)
	if err != nil {
		panic(err)
	}
	d, err := dials.Config(mainCtx, config, &env.Source{}, flagSrc)
	if err != nil {
		panic(err)
	}
	config = d.View()

	// Create event bus
	eventBus := events.NewBus()

	// Create audio context
	audioCtx := audio.NewAudio()
	midiCtx := midi.NewMIDI(config.MIDI, eventBus)

	// Create RadioState before UI - it now owns discovery
	rs := NewRadioState(audioCtx, midiCtx, eventBus, config.Station, config.Profile)

	// Create UI with event bus
	u := ui.NewUI(config.UI, eventBus)
	u.RadioShim = rs

	// Start UI event handler
	go u.HandleEvents(eventBus.Subscribe(100))

	// Start RadioState event handler for connection requests
	go func() {
		eventChan := eventBus.Subscribe(10)
		for event := range eventChan {
			switch e := event.(type) {
			case events.RadioSelected:
				if err := rs.ConnectToRadio(mainCtx, e.Address); err != nil {
					log.Println("Connection error:", err)
					// TODO: show error in UI
				}
			}
		}
	}()

	// Start radio discovery
	rs.StartDiscovery(mainCtx)

	if err := ebiten.RunGame(u); err != nil {
		log.Fatal(err)
	}
}
