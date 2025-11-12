package ui

import (
	"image"
	"image/color"
	"log"
	"maps"
	"math"
	"slices"
	"time"

	ebimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/colornames"
)

type WaterfallWidgets struct {
	Container        *widget.Container
	SliceArea        *widget.Container
	Slices           map[string]*Slice
	Waterfall        *Waterfall
	Controls         *WaterfallControls
	TransmitSettings *TransmitSettings
}

const sliceFlagPadding = 4.0

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
	SliceFlagBgRX        *ebiten.Image
	SliceFlagBgTX        *ebiten.Image
	SliceFlagFace        *text.Face
}

type DragData struct {
	Active bool
	What   string
	Start  float64
	Aux    float64
}

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
	wf.Waterfall.initSliceFlagBg(u)
	u.Widgets.WaterfallPage = wf
}

func (u *UI) ShowWaterfall() {
	u.Widgets.MainPage.SetPage(u.Widgets.WaterfallPage.Container)
	u.state = MainState
}

func (wfw *WaterfallWidgets) GetActiveSlice() *Slice {
	for _, slice := range wfw.Slices {
		if slice.Data.Active {
			return slice
		}
	}
	return nil
}

func (wf *WaterfallWidgets) Update(u *UI) {
	wf.Waterfall.Update(u)
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if slice := wf.GetActiveSlice(); slice != nil {
			go u.RadioShim.TuneSliceStep(slice.Data, -1)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if slice := wf.GetActiveSlice(); slice != nil {
			go u.RadioShim.TuneSliceStep(slice.Data, 1)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		u.RadioShim.SetPTT(wf.Controls.MOX.State() != widget.WidgetChecked)
	}

	wf.Container.RequestRelayout()
}

type wfDragDropper struct {
	container *widget.Container
	u         *UI
	wfw       *WaterfallWidgets
}

func (d *wfDragDropper) Create(parent widget.HasWidget) (*widget.Container, interface{}) {
	d.container = widget.NewContainer()
	return d.container, nil
}

func (d *wfDragDropper) Update(canDrop bool, targetWidget widget.HasWidget, dragData interface{}) {
	wf := d.wfw.Waterfall
	xPos := float64(d.container.GetWidget().Rect.Min.X)
	if wf.Drag.Active {
		if wf.Drag.What == "waterfall" {
			delta := wf.Drag.Start - xPos
			freq := wf.Drag.Aux + (delta/float64(wf.Width))*(wf.DispHighLatch-wf.DispLowLatch)
			go d.u.RadioShim.CenterWaterfallAt(freq)
		} else {
			newTuneX := xPos + wf.Drag.Aux
			freq := wf.DispLowLatch + (newTuneX/float64(wf.Width))*(wf.DispHighLatch-wf.DispLowLatch)
			go d.u.RadioShim.TuneSlice(d.wfw.Slices[wf.Drag.What].Data, freq, true)
		}
	} else {
		log.Println("how are we in wfDragDropper.Update with no active drag?")
	}
}

func (d *wfDragDropper) EndDrag(dropped bool, sourceWidget widget.HasWidget, dragData interface{}) {
	d.wfw.Waterfall.Drag = DragData{}
}

func (u *UI) MakeWaterfall(wfw *WaterfallWidgets) *Waterfall {
	wf := &Waterfall{}
	wf.Widget = widget.NewGraphic(
		widget.GraphicOpts.Image(ebiten.NewImage(1, 1)),
		widget.GraphicOpts.WidgetOpts(
			widget.WidgetOpts.EnableDragAndDrop(
				widget.NewDragAndDrop(
					widget.DragAndDropOpts.ContentsCreater(&wfDragDropper{u: u, wfw: wfw}),
					widget.DragAndDropOpts.MinDragStartDistance(4),
				),
			),
			widget.WidgetOpts.MouseButtonPressedHandler(func(args *widget.WidgetMouseButtonPressedEventArgs) {
				clickPt := image.Pt(args.OffsetX, args.OffsetY)

				// Build ordered slice list: active slice first, then inactive slices alphabetically
				var orderedSlices []struct {
					key   string
					slice *Slice
				}
				letters := slices.Sorted(maps.Keys(wfw.Slices))
				for _, letter := range letters {
					slice := wfw.Slices[letter]
					if !slice.Data.Present {
						continue
					}
					if slice.Data.Active {
						// Insert active slice at the front
						orderedSlices = append([]struct {
							key   string
							slice *Slice
						}{{letter, slice}}, orderedSlices...)
					} else {
						orderedSlices = append(orderedSlices, struct {
							key   string
							slice *Slice
						}{letter, slice})
					}
				}

				// Check if click is on arrow buttons in touch mode
				if u.cfg.Touch {
					for _, item := range orderedSlices {
						if clickPt.In(item.slice.TuneDownBounds) {
							go u.RadioShim.TuneSliceStep(item.slice.Data, -1)
							return
						}
						if clickPt.In(item.slice.TuneUpBounds) {
							go u.RadioShim.TuneSliceStep(item.slice.Data, 1)
							return
						}
					}
				}

				now := time.Now()
				if time.Since(wf.ClickTime) < 400*time.Millisecond {
					freq := wf.DispLowLatch + (float64(args.OffsetX)/float64(wf.Width))*(wf.DispHighLatch-wf.DispLowLatch)
					if slice := wfw.GetActiveSlice(); slice != nil {
						go u.RadioShim.TuneSlice(slice.Data, freq, true)
					}
				}
				wf.ClickTime = now
				for _, item := range orderedSlices {
					if float64(args.OffsetX) >= item.slice.FootprintLeft && float64(args.OffsetX) <= item.slice.FootprintRight {
						wf.Drag = DragData{
							Active: true,
							What:   item.key,
							Start:  float64(args.OffsetX),
							Aux:    item.slice.TuneX - float64(args.OffsetX),
						}
						if !item.slice.Data.Active {
							u.RadioShim.ActivateSlice(item.slice.Data.Index)
						}
						break
					}
				}
				if !wf.Drag.Active {
					wf.Drag = DragData{
						Active: true,
						What:   "waterfall",
						Start:  float64(args.OffsetX),
						Aux:    (wf.DispLow + wf.DispHigh) / 2,
					}
				}
			}),
			widget.WidgetOpts.ScrolledHandler(func(args *widget.WidgetScrolledEventArgs) {
				// Scroll up (positive Y) tunes up, scroll down (negative Y) tunes down
				if args.Y != 0 {
					if slice := wfw.GetActiveSlice(); slice != nil {
						go u.RadioShim.TuneSliceStep(slice.Data, int(args.Y))
					}
				}
			}),
		),
	)
	return wf
}

func (wf *Waterfall) initSliceFlagBg(u *UI) {
	// Store font face for reuse
	wf.SliceFlagFace = u.Font("Roboto-Semibold-16")

	// Calculate dimensions based on monospace font
	textWidth, textHeight := text.Measure("M", *wf.SliceFlagFace, 0)

	flagWidth := textWidth + sliceFlagPadding*2
	flagHeight := textHeight + sliceFlagPadding*2

	// Create both RX and TX flag backgrounds
	wf.SliceFlagBgRX = createSliceFlagImage(flagWidth, flagHeight, sliceRXBgColor)
	wf.SliceFlagBgTX = createSliceFlagImage(flagWidth, flagHeight, sliceTXBgColor)
}

func createSliceFlagImage(flagWidth, flagHeight float64, bgColor color.Color) *ebiten.Image {
	radius := float32(2.0)
	img := ebiten.NewImage(int(flagWidth)+1, int(flagHeight)+1)

	flagX := float32(0)
	flagY := float32(0)

	// Four corners
	vector.DrawFilledCircle(img, flagX+radius, flagY+radius, radius, bgColor, true)
	vector.DrawFilledCircle(img, flagX+float32(flagWidth)-radius, flagY+radius, radius, bgColor, true)
	vector.DrawFilledCircle(img, flagX+radius, flagY+float32(flagHeight)-radius, radius, bgColor, true)
	vector.DrawFilledCircle(img, flagX+float32(flagWidth)-radius, flagY+float32(flagHeight)-radius, radius, bgColor, true)

	// Fill the middle rectangles
	vector.DrawFilledRect(img, flagX+radius, flagY, float32(flagWidth)-2*radius, float32(flagHeight), bgColor, true)
	vector.DrawFilledRect(img, flagX, flagY+radius, radius, float32(flagHeight)-2*radius, bgColor, true)
	vector.DrawFilledRect(img, flagX+float32(flagWidth)-radius, flagY+radius, radius, float32(flagHeight)-2*radius, bgColor, true)

	return img
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
			binShift := math.Round(wf.ScrollAccumulator)
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

	// log.Printf("data: (%f - %f) in %d, disp: (%f - %f) in %d, scale: %s\n", wf.DataLow, wf.DataHigh, wf.Bins, wf.DispLowLatch, wf.DispHighLatch, wf.Width, geom.String())
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

func (wf *Waterfall) drawSliceFlag(markerPos float64, letter string, isTX bool, u *UI, slice *Slice) {
	// Position flag at top of waterfall, just to the right of marker
	flagX := markerPos + 2
	flagY := 2.0

	// Select background based on TX status
	flagBg := wf.SliceFlagBgRX
	letterColor := sliceRXTextColor
	if isTX {
		flagBg = wf.SliceFlagBgTX
		letterColor = sliceTXTextColor
	}

	bgWidth := float64(flagBg.Bounds().Dx())
	bgHeight := float64(flagBg.Bounds().Dy())

	// If touch mode is enabled, draw arrow buttons on either side
	if u.cfg.Touch {
		arrowSize := 24.0
		arrowPadding := 2.0

		// Draw left arrow button (tune down)
		leftArrowX := flagX - arrowSize - arrowPadding
		if leftArrowX >= 0 {
			drawArrowButton(wf.Widget.Image, leftArrowX, flagY, arrowSize, bgHeight, true, u)
			// Store bounds for click detection
			slice.TuneDownBounds = image.Rect(
				int(leftArrowX),
				int(flagY),
				int(leftArrowX+arrowSize),
				int(flagY+bgHeight),
			)
		} else {
			slice.TuneDownBounds = image.Rectangle{}
		}

		// Draw right arrow button (tune up)
		rightArrowX := flagX + bgWidth + arrowPadding
		if rightArrowX+arrowSize <= float64(wf.Width) {
			drawArrowButton(wf.Widget.Image, rightArrowX, flagY, arrowSize, bgHeight, false, u)
			// Store bounds for click detection
			slice.TuneUpBounds = image.Rect(
				int(rightArrowX),
				int(flagY),
				int(rightArrowX+arrowSize),
				int(flagY+bgHeight),
			)
		} else {
			slice.TuneUpBounds = image.Rectangle{}
		}
	} else {
		// Clear bounds when not in touch mode
		slice.TuneDownBounds = image.Rectangle{}
		slice.TuneUpBounds = image.Rectangle{}
	}

	// Draw pre-rendered background with transparency
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(flagX, flagY)
	opts.ColorScale.ScaleAlpha(0.9)
	wf.Widget.Image.DrawImage(flagBg, opts)

	// Measure the actual letter width and calculate centering offset
	letterWidth, _ := text.Measure(letter, *wf.SliceFlagFace, 0)
	centerOffsetX := (bgWidth - letterWidth) / 2

	// Draw the letter text centered
	textOpts := &text.DrawOptions{}
	textOpts.GeoM.Translate(flagX+centerOffsetX, flagY+sliceFlagPadding)
	textOpts.ColorScale.ScaleWithColor(letterColor)
	text.Draw(wf.Widget.Image, letter, *wf.SliceFlagFace, textOpts)
}

// drawArrowButton draws a clickable arrow button for tuning
func drawArrowButton(img *ebiten.Image, x, y, width, height float64, isLeft bool, u *UI) {
	// Draw rounded background
	radius := float32(2.0)
	bgColor := color.RGBA{0x40, 0x40, 0x40, 0xe0}

	// Four corners
	vector.DrawFilledCircle(img, float32(x)+radius, float32(y)+radius, radius, bgColor, true)
	vector.DrawFilledCircle(img, float32(x+width)-radius, float32(y)+radius, radius, bgColor, true)
	vector.DrawFilledCircle(img, float32(x)+radius, float32(y+height)-radius, radius, bgColor, true)
	vector.DrawFilledCircle(img, float32(x+width)-radius, float32(y+height)-radius, radius, bgColor, true)

	// Fill the middle rectangles
	vector.DrawFilledRect(img, float32(x)+radius, float32(y), float32(width)-2*radius, float32(height), bgColor, true)
	vector.DrawFilledRect(img, float32(x), float32(y)+radius, radius, float32(height)-2*radius, bgColor, true)
	vector.DrawFilledRect(img, float32(x+width)-radius, float32(y)+radius, radius, float32(height)-2*radius, bgColor, true)

	// Draw arrow icon
	arrowIcon := "\ue5c4" // Left arrow
	if !isLeft {
		arrowIcon = "\ue5c8" // Right arrow
	}

	iconFace := u.Font("Icons-16")
	iconWidth, _ := text.Measure(arrowIcon, *iconFace, 0)
	iconX := x + (width-iconWidth)/2
	iconY := y + sliceFlagPadding

	textOpts := &text.DrawOptions{}
	textOpts.GeoM.Translate(iconX, iconY)
	textOpts.ColorScale.ScaleWithColor(colornames.White)
	text.Draw(img, arrowIcon, *iconFace, textOpts)
}

func (wf *Waterfall) drawSliceMarker(slice *Slice, letter string, u *UI) {
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

	// Draw the flag
	wf.drawSliceFlag(markerPos, letter, data.TX, u, slice)
}

func (wf *Waterfall) Update(u *UI) {
	if wf.Bins == 0 {
		return
	}

	wf.updateSize()
	wf.handleFreqScroll()
	wf.drawWaterfall()

	// Draw slices in reverse letter order to avoid flickering
	letters := slices.Sorted(maps.Keys(u.Widgets.WaterfallPage.Slices))
	slices.Reverse(letters)
	for _, letter := range letters {
		slice := u.Widgets.WaterfallPage.Slices[letter]
		if slice.Data.Present {
			wf.drawSliceMarker(slice, letter, u)
		}
	}

}
