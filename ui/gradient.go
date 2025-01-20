package ui

import (
	"image/color"
	"math"

	"github.com/tinne26/badcolor"
	"golang.org/x/image/colornames"
)

func gradientGet(stops []badcolor.Oklab, i byte) color.Color {
	pos := float64(i) * float64(len(stops)-1) / 255
	floor := math.Floor(pos)
	if floor == pos {
		return stops[int(pos)].RGBA8()
	} else {
		prev := stops[int(floor)]
		next := stops[int(floor)+1]
		lerp := pos - floor
		blended := prev.Interpolate(next, lerp)
		return blended.RGBA8()
	}
}

var waterfallGradient [4 * 256]byte

func init() {
	stops := []color.Color{
		colornames.Darkblue, colornames.Blue, colornames.Green, colornames.Yellow, colornames.Red, colornames.White,
	}
	labStops := make([]badcolor.Oklab, len(stops))
	for i := range stops {
		labStops[i] = badcolor.ToOklab(stops[i])
	}

	for i := 0; i < 256; i++ {
		color := gradientGet(labStops, byte(i))
		r, g, b, _ := color.RGBA()
		waterfallGradient[4*i] = byte(r >> 8)
		waterfallGradient[4*i+1] = byte(g >> 8)
		waterfallGradient[4*i+2] = byte(b >> 8)
		waterfallGradient[4*i+3] = 0xff
	}
}
