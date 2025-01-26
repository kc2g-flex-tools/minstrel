package ui

import (
	"image"
	"log"
	"math"
	"time"

	ebimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/colornames"
)

type WaterfallWidgets struct {
	Container *widget.Container
	SliceArea *widget.Container
	Slices    map[string]*Slice
	Waterfall *Waterfall
	Controls  *WaterfallControls
}

type Waterfall struct {
	Widget               *widget.Graphic
	Img                  *ebiten.Image
	BackBuffer           *ebiten.Image
	RowBuffer            []byte
	Width                int
	Height               int
	Bins                 int
	ScrollPos            int
	DispLow              float64
	DispHigh             float64
	DispLowLatch         float64
	DispHighLatch        float64
	DataLow              float64
	DataHigh             float64
	PrevDataLow          float64
	PrevDataHigh         float64
	SliceBwImg           *ebimage.NineSlice
	ActiveSliceMarkImg   *ebimage.NineSlice
	InactiveSliceMarkImg *ebimage.NineSlice
	ScrollAccumulator    float64
	Drag                 DragData
	ClickTime            time.Time
}

type DragData struct {
	Active bool
	What   int
	Start  float64
	Aux    float64
}

const tuneStep = 0.0001 // 100Hz. TODO: Configurable.

func (u *UI) MakeWaterfallPage() {
	wf := &WaterfallWidgets{}
	wf.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
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
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(4)),
			widget.GridLayoutOpts.Spacing(8, 0),
		)),
	)
	wf.Slices = map[string]*Slice{
		"A": u.MakeSlice("A"),
		"B": u.MakeSlice("B"),
	}

	wf.Controls = u.MakeWaterfallControls()

	wf.SliceArea.AddChild(
		wf.Slices["A"].Container,
		wf.Controls.Container,
		wf.Slices["B"].Container,
	)

	wf.Container.AddChild(wf.SliceArea)
	wf.Waterfall = u.MakeWaterfall(wf)
	wf.Container.AddChild(wf.Waterfall.Widget)
	wf.Waterfall.SliceBwImg = ebimage.NewNineSliceColor(colornames.Lightskyblue)
	wf.Waterfall.ActiveSliceMarkImg = ebimage.NewNineSliceColor(colornames.Yellow)
	wf.Waterfall.InactiveSliceMarkImg = ebimage.NewNineSliceColor(colornames.Red)
	u.Widgets.WaterfallPage = wf
}

func (u *UI) ShowWaterfall() {
	u.Widgets.MainPage.SetPage(u.Widgets.WaterfallPage.Container)
	u.state = MainState
}

func (wf *WaterfallWidgets) Update(u *UI) {
	wf.UpdateSlices(u)
	wf.Waterfall.Update(u)
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		for _, slice := range wf.Slices {
			if slice.Data.Active {
				u.RadioShim.TuneSlice(slice.Data.Index, slice.Data.Freq-tuneStep)
				break
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		for _, slice := range wf.Slices {
			if slice.Data.Active {
				u.RadioShim.TuneSlice(slice.Data.Index, slice.Data.Freq+tuneStep)
				break
			}
		}
	}
}

func (u *UI) MakeWaterfall(wfw *WaterfallWidgets) *Waterfall {
	wf := &Waterfall{}
	wf.Widget = widget.NewGraphic(
		widget.GraphicOpts.WidgetOpts(
			widget.WidgetOpts.MouseButtonPressedHandler(func(args *widget.WidgetMouseButtonPressedEventArgs) {
				now := time.Now()
				if time.Since(wf.ClickTime) < 200*time.Millisecond {
					freq := wf.DispLowLatch + (float64(args.OffsetX)/float64(wf.Width))*(wf.DispHighLatch-wf.DispLowLatch)
					freq = math.Round(freq/tuneStep) * tuneStep
					for _, slice := range wfw.Slices {
						if slice.Data.Active {
							u.RadioShim.TuneSlice(slice.Data.Index, freq)
							break
						}
					}
				}
				wf.ClickTime = now
				for _, slice := range wfw.Slices {
					if float64(args.OffsetX) >= slice.FootprintLeft && float64(args.OffsetX) <= slice.FootprintRight {
						wf.Drag = DragData{
							Active: true,
							What:   slice.Data.Index,
							Start:  float64(args.OffsetX),
							Aux:    slice.TuneX - float64(args.OffsetX),
						}
						if !slice.Data.Active {
							u.RadioShim.ActivateSlice(slice.Data.Index)
						}
					}
				}
				if !wf.Drag.Active {
					wf.Drag = DragData{
						Active: true,
						What:   -1,
						Start:  float64(args.OffsetX),
						Aux:    (wf.DispLow + wf.DispHigh) / 2,
					}
				}
			}),
			widget.WidgetOpts.CursorMoveHandler(func(args *widget.WidgetCursorMoveEventArgs) {
				if wf.Drag.Active && math.Abs(float64(args.OffsetX)-wf.Drag.Start) >= 4 {
					if wf.Drag.What < 0 {
						delta := wf.Drag.Start - float64(args.OffsetX)
						freq := wf.Drag.Aux + (delta/float64(wf.Width))*(wf.DispHighLatch-wf.DispLowLatch)
						u.RadioShim.CenterWaterfallAt(freq)
					} else {
						newTuneX := float64(args.OffsetX) + wf.Drag.Aux
						freq := wf.DispLowLatch + (newTuneX/float64(wf.Width))*(wf.DispHighLatch-wf.DispLowLatch)
						freq = math.Round(freq/tuneStep) * tuneStep
						u.RadioShim.TuneSlice(wf.Drag.What, freq)
					}
				}
			}),
			widget.WidgetOpts.MouseButtonReleasedHandler(func(args *widget.WidgetMouseButtonReleasedEventArgs) {
				wf.Drag = DragData{}
			}),
		),
	)
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

func (wf *Waterfall) updateSize() {
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
}

func (wf *Waterfall) handleFreqScroll() {
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
				wf.BackBuffer = ebiten.NewImage(wf.Bins, wf.Height)
				wf.BackBuffer.Fill(colornames.Black)
				wf.BackBuffer.DrawImage(
					oldBb, &ebiten.DrawImageOptions{GeoM: geom},
				)
				oldBb.Deallocate()
			}
		}
		wf.PrevDataLow, wf.PrevDataHigh = wf.DataLow, wf.DataHigh
	}
}

func (wf *Waterfall) drawWaterfall() {
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
}

func (wf *Waterfall) drawSliceMarker(slice *Slice) {
	data := slice.Data
	freq := data.Freq
	markerPos := float64(wf.Width) * (freq - wf.DispLowLatch) / (wf.DispHighLatch - wf.DispLowLatch)
	shadeLeft := float64(wf.Width) * (freq + data.FiltLow/1e6 - wf.DispLowLatch) / (wf.DispHighLatch - wf.DispLowLatch)
	shadeRight := float64(wf.Width) * (freq + data.FiltHigh/1e6 - wf.DispLowLatch) / (wf.DispHighLatch - wf.DispLowLatch)

	slice.FootprintLeft = min(markerPos, shadeLeft)
	slice.FootprintRight = max(markerPos, shadeRight)
	slice.TuneX = markerPos

	wf.SliceBwImg.Draw(wf.Widget.Image, 1, wf.Height, func(opts *ebiten.DrawImageOptions) {
		opts.GeoM.Scale(shadeRight-shadeLeft, 1)
		opts.GeoM.Translate(shadeLeft, 0)
		opts.ColorScale.ScaleAlpha(0.3)
	})

	mark := wf.InactiveSliceMarkImg
	if data.Active {
		mark = wf.ActiveSliceMarkImg
	}
	mark.Draw(wf.Widget.Image, 2, wf.Height, func(opts *ebiten.DrawImageOptions) {
		opts.GeoM.Translate(markerPos, 0)
		opts.ColorScale.ScaleAlpha(0.5)
	})
}

func (wf *Waterfall) Update(u *UI) {
	if wf.Bins == 0 {
		return
	}

	wf.updateSize()
	wf.handleFreqScroll()
	wf.drawWaterfall()

	for _, letter := range []string{"A", "B"} {
		slice := u.Widgets.WaterfallPage.Slices[letter]
		if slice.Data.Present {
			wf.drawSliceMarker(slice)
		}
	}

}
