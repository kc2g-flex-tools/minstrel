package ui

import (
	"image/color"
	"strconv"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
	"golang.org/x/image/colornames"
)

type Slice struct {
	Container      *widget.Container
	Letter         *widget.Text
	Frequency      *widget.Text
	RXAnt          *widget.Text
	TXAnt          *widget.Text
	Mode           *widget.Text
	Data           *radioshim.SliceData
	FootprintLeft  float64
	FootprintRight float64
	TuneX          float64
}

func (u *UI) MakeSlice(letter string) *Slice {
	s := &Slice{
		Data: &radioshim.SliceData{},
	}
	s.Container = u.MakeRoundedRect(colornames.Black, color.NRGBA{}, 4,
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
		)),
	)
	display := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: 12, Right: 12, Top: 4, Bottom: 4}),
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
		widget.TextOpts.Text(letter, u.Font("Roboto-Semibold-36"), colornames.Darkslategray),
		widget.TextOpts.Insets(widget.Insets{}),
	)
	letterContainer.AddChild(s.Letter)
	row1.AddChild(letterContainer)
	s.RXAnt = u.MakeText("Roboto-Condensed-24", colornames.Deepskyblue)
	row1.AddChild(s.RXAnt)
	s.TXAnt = u.MakeText("Roboto-Condensed-24", colornames.Red)
	row1.AddChild(s.TXAnt)
	display.AddChild(row1)

	row2 := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(4),
		)),
	)
	s.Frequency = u.MakeText("Roboto-Condensed-Light-32", colornames.Seashell)
	s.Frequency.GetWidget().MouseButtonPressedEvent.AddHandler(func(_ any) {
		u.ShowWindow(
			u.MakeEntryWindow("Enter frequency", "Roboto-24", "", "Roboto-24", func(freqStr string, ok bool) {
				freq, _ := strconv.ParseFloat(freqStr, 64)
				u.RadioShim.TuneSlice(s.Data, freq, false)
			}),
		)
	})
	row2.AddChild(s.Frequency)
	s.Mode = u.MakeText("Roboto-16", colornames.Lightgray, widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})))
	s.Mode.GetWidget().MouseButtonPressedEvent.AddHandler(func(_ any) {
		var modes []any
		for _, mode := range s.Data.Modes {
			modes = append(modes, mode)
		}
		u.ShowWindow(
			u.MakeListWindow(
				"Select mode", "Roboto-16", "", "Roboto-16",
				modes, s.Data.Mode, func(m any) string { return m.(string) },
				func(item any, ok bool) {
					if ok {
						u.RadioShim.SetSliceMode(s.Data.Index, item.(string))
					}
				},
			))
	})
	row2.AddChild(s.Mode)
	display.AddChild(row2)
	s.Container.AddChild(display)

	buttons := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		)),
	)
	buttons.AddChild(u.MakeButton("Icons-16", "\ue5cd", func(*widget.ButtonClickedEventArgs) {}))
	buttons.AddChild(u.MakeButton("Icons-16", "\ue8b8", func(*widget.ButtonClickedEventArgs) {}))
	s.Container.AddChild(buttons)

	return s
}

func (w *WaterfallWidgets) UpdateSlices(u *UI) {
	slices := u.RadioShim.GetSlices()
	for _, letter := range []string{"A", "B"} {
		slice := slices[letter]
		if slice == nil {
			slice = &radioshim.SliceData{}
		}

		widg := w.Slices[letter]
		widg.Data = slice
		if !slice.Present {
			widg.Container.GetWidget().Visibility = widget.Visibility_Hide
			continue
		}
		widg.Container.GetWidget().Visibility = widget.Visibility_Show
		widg.Frequency.Label = slice.FreqFormatted
		widg.Mode.Label = slice.Mode
		widg.RXAnt.Label = slice.RXAnt
		widg.TXAnt.Label = slice.TXAnt
	}
}
