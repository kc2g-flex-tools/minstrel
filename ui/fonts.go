package ui

import (
	"log"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kc2g-flex-tools/minstrel/assets"
)

type variation struct {
	Tag   text.Tag
	Value float32
}

type feature struct {
	Tag   text.Tag
	Value uint32
}

type fontspec struct {
	Filename   string
	Variations []variation
	Features   []feature
}

var fontFiles = map[string]fontspec{
	"Roboto":                 fontspec{Filename: "Roboto-Variable.ttf"},
	"Roboto-Condensed":       fontspec{Filename: "Roboto-Variable.ttf", Variations: []variation{{Tag: text.MustParseTag("wdth"), Value: 85}}},
	"Roboto-Light":           fontspec{Filename: "Roboto-Variable.ttf", Variations: []variation{{Tag: text.MustParseTag("wght"), Value: 300}}, Features: []feature{{Tag: text.MustParseTag("pnum"), Value: 1}}},
	"Roboto-Condensed-Light": fontspec{Filename: "Roboto-Variable.ttf", Variations: []variation{{Tag: text.MustParseTag("wght"), Value: 300}, {Tag: text.MustParseTag("wdth"), Value: 85}}, Features: []feature{{Tag: text.MustParseTag("pnum"), Value: 1}}},
	"Icons":                  fontspec{Filename: "MaterialSymbolsSharp-Regular.ttf"},
	"Icons-Filled":           fontspec{Filename: "MaterialSymbolsSharp_Filled-Regular.ttf"},
}

var sources = make(map[string]*text.GoTextFaceSource)

func loadFontSource(filename string) *text.GoTextFaceSource {
	if source, ok := sources[filename]; ok {
		return source
	}
	file, err := assets.Assets.Open("fonts/" + filename)
	if err != nil {
		log.Fatal(err)
	}
	source, err := text.NewGoTextFaceSource(file)
	if err != nil {
		log.Fatal(err)
	}
	sources[filename] = source
	return source
}

func loadFont(name string) (*text.GoTextFaceSource, []variation, []feature) {
	spec, ok := fontFiles[name]
	if !ok {
		log.Fatalf("font %q not found", name)
	}

	source := loadFontSource(spec.Filename)
	return source, spec.Variations, spec.Features
}

var fontCache = make(map[string]text.Face)

func (u *UI) Font(name string) text.Face {
	if font, ok := fontCache[name]; ok {
		return font
	}

	idx := strings.LastIndex(name, "-")
	if idx == -1 {
		log.Fatalf("invalid font spec %q: no size", name)
	}
	fontName := name[:idx]
	size, err := strconv.ParseFloat(name[idx+1:], 64)
	if err != nil {
		log.Fatalf("invalid font spec %q: %s parsing size", name, err)
	}

	source, variations, features := loadFont(fontName)
	face := &text.GoTextFace{Source: source, Size: size}
	for _, variation := range variations {
		face.SetVariation(variation.Tag, variation.Value)
	}
	for _, feature := range features {
		face.SetFeature(feature.Tag, feature.Value)
	}
	fontCache[name] = face
	return face
}
