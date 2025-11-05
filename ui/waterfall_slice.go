package ui

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/colornames"

	"github.com/kc2g-flex-tools/minstrel/pkg/errutil"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
)

type Slice struct {
	Container      *widget.Container
	CreateButton   *widget.Container
	SlicePanel     *widget.Container
	Letter         *widget.Text
	Frequency      *widget.Text
	RXAnt          *widget.Text
	TXAnt          *widget.Text
	Mode           *widget.Text
	Data           *radioshim.SliceData
	FootprintLeft  float64
	FootprintRight float64
	TuneX          float64
	VolumeSlider   *widget.Slider // Volume slider property
}

func (u *UI) MakeSlice(letter string) *Slice {
	s := &Slice{
		Data: &radioshim.SliceData{},
	}
	// Outer container that will hold either the slice panel or create button
	s.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// Create the slice panel with rounded background
	s.SlicePanel = u.MakeRoundedRect(colornames.Black, color.NRGBA{}, 4)

	// Inner container for the horizontal row layout
	innerRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			StretchHorizontal: true,
			StretchVertical:   true,
		})),
	)

	display := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(12)),
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
		widget.TextOpts.Padding(widget.NewInsetsSimple(0)),
	)
	letterContainer.AddChild(s.Letter)
	row1.AddChild(letterContainer)
	s.RXAnt = u.MakeText("Roboto-Condensed-24", colornames.Deepskyblue)
	s.RXAnt.GetWidget().MouseButtonPressedEvent.AddHandler(func(_ any) {
		var antennas []any
		for _, ant := range s.Data.RXAntList {
			antennas = append(antennas, ant)
		}
		u.ShowWindow(
			u.MakeListWindow(
				"Select RX Antenna", "Roboto-16", "", "Roboto-16",
				antennas, s.Data.RXAnt, func(a any) string { return a.(string) },
				func(item any, ok bool) {
					if ok {
						u.RadioShim.SetSliceRXAnt(s.Data.Index, item.(string))
					}
				},
			))
	})
	row1.AddChild(s.RXAnt)
	s.TXAnt = u.MakeText("Roboto-Condensed-24", colornames.Red)
	s.TXAnt.GetWidget().MouseButtonPressedEvent.AddHandler(func(_ any) {
		var antennas []any
		for _, ant := range s.Data.TXAntList {
			antennas = append(antennas, ant)
		}
		u.ShowWindow(
			u.MakeListWindow(
				"Select TX Antenna", "Roboto-16", "", "Roboto-16",
				antennas, s.Data.TXAnt, func(a any) string { return a.(string) },
				func(item any, ok bool) {
					if ok {
						u.RadioShim.SetSliceTXAnt(s.Data.Index, item.(string))
					}
				},
			))
	})

	// Create the volume slider (hidden by default)
	s.VolumeSlider = widget.NewSlider(
		widget.SliderOpts.MinMax(0, 100),
		widget.SliderOpts.InitialCurrent(100-s.Data.Volume),
		widget.SliderOpts.Images(u.SliderTrackImage(), u.SliderHandleImage()),
		widget.SliderOpts.MinHandleSize(5),
		widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(2)),
		widget.SliderOpts.Orientation(widget.DirectionVertical),
		widget.SliderOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(10, 100),
		),
	)
	s.VolumeSlider.ChangedEvent.AddHandler(func(e interface{}) {
		args := e.(*widget.SliderChangedEventArgs)
		u.RadioShim.SetSliceVolume(s.Data.Index, 100-args.Current)
	})

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
			u.MakeNumericEntryWindow("Enter frequency", "Roboto-24", "", "Roboto", 24, func(freqStr string, ok bool) {
				freq := errutil.MustParseFloat(freqStr, "frequency entry")
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
	innerRow.AddChild(display)

	buttons := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		)),
	)
	// Close button
	buttons.AddChild(u.MakeButton("Icons-16", "\ue5cd", func(*widget.ButtonClickedEventArgs) {
		u.RadioShim.RemoveSlice(s.Data.Index)
	}))
	// Slice settings icon
	buttons.AddChild(u.MakeButton("Icons-16", "\ue8b8", func(*widget.ButtonClickedEventArgs) {}))
	// Speaker icon for volume control
	buttons.AddChild(u.MakeButton("Icons-16", "", func(_ *widget.ButtonClickedEventArgs) {
		volContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(10),
			)),
		)
		volContainer.AddChild(s.VolumeSlider)
		window := u.MakeWindow("Slice Volume", "Roboto-24", volContainer, widget.WindowOpts.CloseMode(widget.CLICK_OUT))
		u.ShowWindow(window)
	}))
	innerRow.AddChild(buttons)
	s.SlicePanel.AddChild(innerRow)

	// Create the + button for creating a new slice
	s.CreateButton = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	plusButton := u.MakeButton("Icons-48", "\ue145", func(*widget.ButtonClickedEventArgs) {
		u.RadioShim.CreateSlice()
	})
	s.CreateButton.AddChild(plusButton)

	// Initially show the slice panel (UpdateSlices will adjust visibility)
	s.Container.AddChild(s.SlicePanel)

	return s
}

func (w *WaterfallWidgets) UpdateSlices(slices map[string]*radioshim.SliceData) {
	for _, letter := range []string{"A", "B"} {
		slice := slices[letter]
		if slice == nil {
			slice = &radioshim.SliceData{}
		}

		widg := w.Slices[letter]
		widg.Data = slice
		if !slice.Present {
			// Show create button, hide slice panel
			widg.Container.RemoveChild(widg.SlicePanel)
			widg.Container.RemoveChild(widg.CreateButton)
			widg.Container.AddChild(widg.CreateButton)
			continue
		}
		// Show slice panel, hide create button
		widg.Container.RemoveChild(widg.SlicePanel)
		widg.Container.RemoveChild(widg.CreateButton)
		widg.Container.AddChild(widg.SlicePanel)

		widg.Frequency.Label = slice.FreqFormatted
		widg.Mode.Label = slice.Mode
		widg.RXAnt.Label = slice.RXAnt
		widg.TXAnt.Label = slice.TXAnt
		if widg.VolumeSlider != nil {
			widg.VolumeSlider.Current = 100 - slice.Volume
		}
	}
}
