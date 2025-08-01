package main

import (
	"context"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kc2g-flex-tools/flexclient"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sources/env"
	"github.com/vimeo/dials/sources/flag"

	"github.com/kc2g-flex-tools/minstrel/audio"
	"github.com/kc2g-flex-tools/minstrel/ui"
)

type Config struct {
	Station string `dialsdesc:"Station name"`
	Profile string `dialsdesc:"Global profile to load on startup"`
	UI      *ui.Config
}

var config *Config

func defaultConfig() *Config {
	return &Config{
		Station: "Minstrel",
		Profile: "",
		UI:      ui.DefaultConfig(),
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

	u := ui.NewUI(config.UI)
	audio := audio.NewAudio()

	discoveryCtx, discoveryCancel := context.WithCancel(mainCtx)

	discoverChan := make(chan []map[string]string, 1)
	go func() {
		log.Println("start discovery")
		err := flexclient.DiscoverAll(discoveryCtx, 10*time.Second, discoverChan)
		log.Println("finished discovery")
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		for data := range discoverChan {
			u.SetRadios(data)
		}
	}()

	var rs *RadioState

	// TODO: initialize the RadioState before connect, give it control over discovery,
	// and use RadioShim as the callback mechanism, and delete Callbacks
	u.Callbacks.Connect = func(dst string) {
		discoveryCancel()
		fc, err := flexclient.NewFlexClient(dst)
		if err != nil {
			panic(err) // TODO: errors in the UI
		}
		go func() {
			fc.Run()
			log.Fatal("flexclient exited")
		}()
		rs = NewRadioState(fc, u, audio)
		go rs.Run(mainCtx)
		u.ShowWaterfall()
	}

	if err := ebiten.RunGame(u); err != nil {
		log.Fatal(err)
	}
}
