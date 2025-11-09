package ui

import (
	"log"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kc2g-flex-tools/minstrel/assets"
	"github.com/kc2g-flex-tools/minstrel/errutil"
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
	"Roboto":                 {Filename: "Roboto-Variable.ttf"},
	"Roboto-Semibold":        {Filename: "Roboto-Variable.ttf", Variations: []variation{{Tag: text.MustParseTag("wght"), Value: 600}}},
	"Roboto-Condensed":       {Filename: "Roboto-Variable.ttf", Variations: []variation{{Tag: text.MustParseTag("wdth"), Value: 87.5}}},
	"Roboto-Light":           {Filename: "Roboto-Variable.ttf", Variations: []variation{{Tag: text.MustParseTag("wght"), Value: 300}}, Features: []feature{{Tag: text.MustParseTag("pnum"), Value: 0}}},
	"Roboto-Condensed-Light": {Filename: "Roboto-Variable.ttf", Variations: []variation{{Tag: text.MustParseTag("wght"), Value: 300}, {Tag: text.MustParseTag("wdth"), Value: 87.5}}, Features: []feature{{Tag: text.MustParseTag("pnum"), Value: 0}}},
	"Icons":                  {Filename: "MaterialSymbolsSharp-Regular.ttf"},
}

var sources sync.Map // map[string]*text.GoTextFaceSource

func loadFontSource(filename string) *text.GoTextFaceSource {
	if cached, ok := sources.Load(filename); ok {
		return cached.(*text.GoTextFaceSource)
	}
	file, err := assets.Assets.Open("fonts/" + filename)
	if err != nil {
		log.Fatal(err)
	}
	source, err := text.NewGoTextFaceSource(file)
	if err != nil {
		log.Fatal(err)
	}
	sources.Store(filename, source)
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

var fontCache sync.Map // map[string]*text.Face

func (u *UI) Font(name string) *text.Face {
	if cached, ok := fontCache.Load(name); ok {
		return cached.(*text.Face)
	}

	idx := strings.LastIndex(name, "-")
	if idx == -1 {
		log.Fatalf("invalid font spec %q: no size", name)
	}
	fontName := name[:idx]
	size := errutil.MustParseFloat(name[idx+1:], "font spec "+name)
	if size == 0 {
		log.Fatalf("invalid font spec %q: size must be non-zero", name)
	}

	source, variations, features := loadFont(fontName)
	face := &text.GoTextFace{Source: source, Size: size}
	for _, variation := range variations {
		face.SetVariation(variation.Tag, variation.Value)
	}
	for _, feature := range features {
		face.SetFeature(feature.Tag, feature.Value)
	}
	var faceInterface text.Face = face
	fontCache.Store(name, &faceInterface)
	return &faceInterface
}
