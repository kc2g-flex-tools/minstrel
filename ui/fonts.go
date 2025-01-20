package ui

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kc2g-flex-tools/minstrel/assets"
)

func LoadFont(name string) *text.GoTextFaceSource {
	file, err := assets.Assets.Open("fonts/" + name)
	if err != nil {
		log.Fatal(err)
	}
	source, err := text.NewGoTextFaceSource(file)
	if err != nil {
		log.Fatal(err)
	}
	return source
}

type fontspec struct {
	File string
	Size float64
}

var fonts = map[string]fontspec{
	"Roboto-16": fontspec{File: "Roboto-Medium.ttf", Size: 16},
	"Roboto-18": fontspec{File: "Roboto-Medium.ttf", Size: 18},
	"Roboto-24": fontspec{File: "Roboto-Medium.ttf", Size: 24},
	"Roboto-36": fontspec{File: "Roboto-Medium.ttf", Size: 36},
	"Roboto-48": fontspec{File: "Roboto-Medium.ttf", Size: 48},
}

func (u *UI) LoadFonts() {
	u.font = map[string]text.Face{}
	sources := map[string]*text.GoTextFaceSource{}
	for name, spec := range fonts {
		if sources[spec.File] == nil {
			sources[spec.File] = LoadFont(spec.File)
		}
		u.font[name] = &text.GoTextFace{Source: sources[spec.File], Size: spec.Size}
	}
}
