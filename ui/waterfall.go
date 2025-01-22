package ui

import (
	"image"
	"image/color"
	"log"
	"math"

	ebimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/colornames"
)

type WaterfallWidgets struct {
	Container *widget.Container
	SliceArea *widget.Container
	Slices    map[string]*Slice
	Waterfall *Waterfall
}

type Slice struct {
	Container *widget.Container
	Letter    *widget.Text
	Frequency *widget.Text
	RXAnt     *widget.Text
	TXAnt     *widget.Text
	Mode      *widget.Text
	Data      SliceData
}

type Waterfall struct {
	Widget            *widget.Graphic
	Img               *ebiten.Image
	BackBuffer        *ebiten.Image
	RowBuffer         []byte
	Width             int
	Height            int
	Bins              int
	ScrollPos         int
	DispLow           float64
	DispHigh          float64
	DispLowLatch      float64
	DispHighLatch     float64
	DataLow           float64
	DataHigh          float64
	PrevDataLow       float64
	PrevDataHigh      float64
	SliceBwImg        *ebimage.NineSlice
	SliceMarkImg      *ebimage.NineSlice
	ScrollAccumulator float64
}

func (u *UI) MakeSlice(letter string, pos widget.AnchorLayoutPosition) *Slice {
	s := &Slice{}
	s.Container = u.MakeRoundedRect(colornames.Black, color.NRGBA{}, 4,
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 12, Right: 12, Top: 4, Bottom: 4}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.AnchorLayoutData{
					VerticalPosition:   widget.AnchorLayoutPositionStart,
					HorizontalPosition: pos,
				},
			)),
	)
	row1 := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	letterContainer := u.MakeRoundedRect(colornames.Deepskyblue, color.NRGBA{}, 4)
	s.Letter = widget.NewText(
		widget.TextOpts.Text(letter, u.Font("Roboto-48"), colornames.Darkslategray),
		widget.TextOpts.Insets(widget.Insets{}),
	)
	letterContainer.AddChild(s.Letter)
	row1.AddChild(letterContainer)
	s.RXAnt = u.MakeText("Roboto-24", colornames.Deepskyblue)
	row1.AddChild(s.RXAnt)
	s.TXAnt = u.MakeText("Roboto-24", colornames.Red)
	row1.AddChild(s.TXAnt)
	s.Container.AddChild(row1)

	row2 := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(4),
		)),
	)
	s.Frequency = u.MakeText("Roboto-36", colornames.Seashell)
	row2.AddChild(s.Frequency)
	s.Mode = u.MakeText("Roboto-18", colornames.Lightgray)
	s.Container.AddChild(row2)
	return s
}

func (u *UI) MakeWaterfallPage() {
	wf := &WaterfallWidgets{}
	wf.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(8)),
			widget.GridLayoutOpts.Spacing(0, 4),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.AnchorLayoutData{
					StretchHorizontal: true,
					StretchVertical:   true,
				},
			),
		),
	)
	wf.SliceArea = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	wf.Slices = map[string]*Slice{
		"A": u.MakeSlice("A", widget.AnchorLayoutPositionStart),
		"B": u.MakeSlice("B", widget.AnchorLayoutPositionEnd),
	}
	wf.SliceArea.AddChild(wf.Slices["A"].Container)
	wf.SliceArea.AddChild(wf.Slices["B"].Container)
	wf.Container.AddChild(wf.SliceArea)
	wf.Waterfall = u.MakeWaterfall()
	wf.Container.AddChild(wf.Waterfall.Widget)
	wf.Waterfall.SliceBwImg = ebimage.NewNineSliceColor(colornames.Lightskyblue)
	wf.Waterfall.SliceMarkImg = ebimage.NewNineSliceColor(colornames.Yellow)
	u.Widgets.WaterfallPage = wf
}

func (u *UI) ShowWaterfall() {
	u.Widgets.MainPage.SetPage(u.Widgets.WaterfallPage.Container)
	u.Widgets.TopBar.AudioButton.GetWidget().Visibility = widget.Visibility_Show
	u.state = MainState
}

type SliceData struct {
	Present       bool
	Freq          float64
	FreqFormatted string
	Mode          string
	Modes         []string
	RXAnt         string
	TXAnt         string
	FiltHigh      float64
	FiltLow       float64
}

func (w *WaterfallWidgets) SetSlices(slices map[string]SliceData) {
	for _, letter := range []string{"A", "B"} {
		slice := slices[letter]
		widg := w.Slices[letter]
		widg.Data = slice
		if !slice.Present {
			widg.Container.GetWidget().Visibility = widget.Visibility_Hide_Blocking
			continue
		}
		widg.Container.GetWidget().Visibility = widget.Visibility_Show
		widg.Frequency.Label = slice.FreqFormatted
		widg.Mode.Label = slice.Mode
		widg.RXAnt.Label = slice.RXAnt
		widg.TXAnt.Label = slice.TXAnt
	}
}

func (u *UI) MakeWaterfall() *Waterfall {
	wf := &Waterfall{
		Widget: widget.NewGraphic(),
	}
	return wf
}

func (wf *Waterfall) SetBins(w uint16) {
	if wf.Bins == int(w) {
		return
	}
	wf.Bins = int(w)
	wf.RowBuffer = make([]byte, 4*wf.Bins)
	if wf.Height != 0 {
		if wf.BackBuffer != nil {
			wf.BackBuffer.Deallocate()
		}
		wf.BackBuffer = ebiten.NewImage(wf.Bins, wf.Height)
		wf.BackBuffer.Fill(colornames.Black)
	}
}

func (wf *Waterfall) AddRow(bins []uint16, blackLevel uint32) {
	if wf.BackBuffer == nil {
		return
	}

	wf.ScrollPos -= 1
	if wf.ScrollPos < 0 {
		wf.ScrollPos += wf.Height
	}

	wf.DispLowLatch, wf.DispHighLatch = wf.DispLow, wf.DispHigh

	for n, bin := range bins {
		scaledBin := (int(bin) - int(blackLevel)) / 64
		if scaledBin < 0 {
			scaledBin = 0
		}
		if scaledBin > 255 {
			scaledBin = 255
		}
		copy(wf.RowBuffer[4*n:4*n+4], waterfallGradient[scaledBin*4:scaledBin*4+4])
	}
	wf.BackBuffer.SubImage(image.Rect(0, wf.ScrollPos, wf.Bins, wf.ScrollPos+1)).(*ebiten.Image).WritePixels(wf.RowBuffer)
}

func (wf *Waterfall) Update(u *UI) {
	if wf.Bins == 0 {
		return
	}

	rect := wf.Widget.GetWidget().Rect
	width, height := rect.Dx(), rect.Dy()
	if wf.Width != width || wf.Height != height {
		log.Printf("wf rect %d x %d\n", width, height)
		if wf.Height != height {
			oldBB := wf.BackBuffer
			// Make a new backbuffer of the correct height.
			wf.BackBuffer = ebiten.NewImage(wf.Bins, height)
			// Fill any increased height with black
			wf.BackBuffer.Fill(colornames.Black)
			if oldBB != nil {
				// Copy the old backbuffer into the new one, moving scrollpos to 0
				// so that if the height is decreasing we keep the newest data.
				wf.BackBuffer.DrawImage(
					oldBB.SubImage(image.Rect(0, wf.ScrollPos, wf.Bins, wf.Height)).(*ebiten.Image),
					nil,
				)
				geom := ebiten.GeoM{}
				geom.Translate(0, float64(wf.Height-wf.ScrollPos))
				wf.BackBuffer.DrawImage(
					oldBB.SubImage(image.Rect(0, 0, wf.Bins, wf.ScrollPos)).(*ebiten.Image),
					&ebiten.DrawImageOptions{GeoM: geom},
				)
				oldBB.Deallocate()
			}
			wf.ScrollPos = 0
		}
		// Update the size and create the new front buffer
		wf.Width, wf.Height = width, height
		wf.Img = ebiten.NewImage(width, height)
		wf.Widget.Image = wf.Img
	}

	if wf.DataLow != wf.PrevDataLow || wf.DataHigh != wf.PrevDataHigh {
		newSpan := wf.DataHigh - wf.DataLow
		oldSpan := wf.PrevDataHigh - wf.PrevDataLow
		if math.Abs(newSpan-oldSpan)/(newSpan+oldSpan) > 0.01 {
			wf.BackBuffer.Fill(colornames.Black)
		} else {
			freqShift := wf.PrevDataLow - wf.DataLow
			wf.ScrollAccumulator += freqShift * float64(wf.Bins) / (wf.DataHigh - wf.DataLow)
			binShift := float64(int(wf.ScrollAccumulator))
			wf.ScrollAccumulator -= binShift
			if binShift != 0 {
				oldBb := wf.BackBuffer
				geom := ebiten.GeoM{}
				geom.Translate(binShift, 0)
				wf.BackBuffer = ebiten.NewImage(wf.Bins, height)
				wf.BackBuffer.Fill(colornames.Black)
				wf.BackBuffer.DrawImage(
					oldBb, &ebiten.DrawImageOptions{GeoM: geom},
				)
				oldBb.Deallocate()
			}
		}
		wf.PrevDataLow, wf.PrevDataHigh = wf.DataLow, wf.DataHigh
	}

	geom := ebiten.GeoM{}
	geom.Scale((wf.DataHigh-wf.DataLow)/float64(wf.Bins), 1)
	geom.Translate(wf.DataLow-wf.DispLowLatch, 0)
	geom.Scale(float64(wf.Width-1)/(wf.DispHighLatch-wf.DispLowLatch), 1)

	// log.Printf("data: (%f - %f) in %d, disp: (%f - %f) in %d, scale: %#v\n", wf.DataLow, wf.DataHigh, wf.Bins, wf.DispLowLatch, wf.DispHighLatch, wf.Width, geom)
	wf.Widget.Image.Clear()
	wf.Widget.Image.DrawImage(
		wf.BackBuffer.SubImage(image.Rect(0, wf.ScrollPos, wf.Bins, wf.Height)).(*ebiten.Image),
		&ebiten.DrawImageOptions{GeoM: geom, Filter: ebiten.FilterLinear},
	)
	if wf.ScrollPos != 0 {
		geom.Translate(0, float64(wf.Height-wf.ScrollPos))
		wf.Widget.Image.DrawImage(
			wf.BackBuffer.SubImage(image.Rect(0, 0, wf.Bins, wf.ScrollPos)).(*ebiten.Image),
			&ebiten.DrawImageOptions{GeoM: geom, Filter: ebiten.FilterLinear},
		)
	}

	for _, letter := range []string{"A", "B"} {
		data := u.Widgets.WaterfallPage.Slices[letter].Data
		if !data.Present {
			continue
		}
		freq := data.Freq
		markerPos := float64(wf.Width) * (freq - wf.DispLowLatch) / (wf.DispHighLatch - wf.DispLowLatch)
		shadeLeft := float64(wf.Width) * (freq + data.FiltLow/1e6 - wf.DispLowLatch) / (wf.DispHighLatch - wf.DispLowLatch)
		shadeRight := float64(wf.Width) * (freq + data.FiltHigh/1e6 - wf.DispLowLatch) / (wf.DispHighLatch - wf.DispLowLatch)

		wf.SliceBwImg.Draw(wf.Widget.Image, 1, wf.Height, func(opts *ebiten.DrawImageOptions) {
			opts.GeoM.Scale(shadeRight-shadeLeft, 1)
			opts.GeoM.Translate(shadeLeft, 0)
			opts.ColorScale.ScaleAlpha(0.3)
		})

		wf.SliceMarkImg.Draw(wf.Widget.Image, 2, wf.Height, func(opts *ebiten.DrawImageOptions) {
			opts.GeoM.Translate(markerPos, 0)
			opts.ColorScale.ScaleAlpha(0.5)
		})
	}
}
