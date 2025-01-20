package ui

import (
	"image"
	"image/color"
	"log"

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
	Frequency *widget.TextArea
	RXAnt     *widget.TextArea
	TXAnt     *widget.TextArea
	Mode      *widget.TextArea
}

type Waterfall struct {
	Widget     *widget.Graphic
	Img        *ebiten.Image
	BackBuffer *ebiten.Image
	RowBuffer  []byte
	Width      int
	Height     int
	Bins       int
	ScrollPos  int
}

func (u *UI) MakeSlice(letter string, pos widget.AnchorLayoutPosition) *Slice {
	s := &Slice{}
	s.Container = u.MakeRoundedRect(colornames.Black, color.NRGBA{}, 4,
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(4)),
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
			widget.RowLayoutOpts.Spacing(4),
		)),
	)
	letterContainer := u.MakeRoundedRect(colornames.Deepskyblue, color.NRGBA{}, 4)
	s.Letter = widget.NewText(
		widget.TextOpts.Text(letter, u.font["Roboto-48"], colornames.Darkslategray),
		widget.TextOpts.Insets(widget.Insets{}),
	)
	letterContainer.AddChild(s.Letter)
	row1.AddChild(letterContainer)
	s.RXAnt = u.MakeTextArea("Roboto-24", colornames.Deepskyblue, colornames.Black)
	row1.AddChild(s.RXAnt)
	s.TXAnt = u.MakeTextArea("Roboto-24", colornames.Red, colornames.Black)
	row1.AddChild(s.TXAnt)
	s.Container.AddChild(row1)

	row2 := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(4),
		)),
	)
	s.Frequency = u.MakeTextArea("Roboto-36", colornames.Seashell, colornames.Black)
	row2.AddChild(s.Frequency)
	s.Mode = u.MakeTextArea("Roboto-18", colornames.Lightgray, colornames.Black)
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
}

func (w *WaterfallWidgets) SetSlices(slices map[string]SliceData) {
	for _, letter := range []string{"A", "B"} {
		slice := slices[letter]
		widg := w.Slices[letter]
		if !slice.Present {
			widg.Container.GetWidget().Visibility = widget.Visibility_Hide_Blocking
			continue
		}
		widg.Container.GetWidget().Visibility = widget.Visibility_Show
		widg.Frequency.SetText(slice.FreqFormatted)
		widg.Mode.SetText(slice.Mode)
		widg.RXAnt.SetText(slice.RXAnt)
		widg.TXAnt.SetText(slice.TXAnt)
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
		wf.BackBuffer = ebiten.NewImage(wf.Bins, wf.Height)
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

func (wf *Waterfall) Update() {
	if wf.Bins == 0 {
		return
	}

	rect := wf.Widget.GetWidget().Rect
	width, height := rect.Dx(), rect.Dy()
	if wf.Width != width || wf.Height != height {
		log.Printf("wf rect %d x %d\n", width, height)
		wf.Width, wf.Height = width, height
		wf.Img = ebiten.NewImage(width, height)
		wf.Widget.Image = wf.Img
		wf.BackBuffer = ebiten.NewImage(wf.Bins, height)
		wf.ScrollPos = height
	}

	geom := &ebiten.GeoM{}
	geom.Scale(float64(wf.Width)/float64(wf.Bins), 1)
	wf.Widget.Image.DrawImage(
		wf.BackBuffer.SubImage(image.Rect(0, wf.ScrollPos, wf.Bins, wf.Height)).(*ebiten.Image),
		&ebiten.DrawImageOptions{GeoM: *geom, Filter: ebiten.FilterLinear},
	)
	if wf.ScrollPos == 0 {
		return
	}
	geom.Translate(0, float64(wf.Height-wf.ScrollPos))
	wf.Widget.Image.DrawImage(
		wf.BackBuffer.SubImage(image.Rect(0, 0, wf.Bins, wf.ScrollPos)).(*ebiten.Image),
		&ebiten.DrawImageOptions{GeoM: *geom, Filter: ebiten.FilterLinear},
	)
}
