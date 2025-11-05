package ui

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/colornames"
)

type TransmitSettings struct {
	Window          *Window
	MicSelection    *widget.Button
	MicLevelSlider  *widget.Slider
	MicLevelLabel   *widget.Text
	VoxEnableToggle *widget.Button
	VoxLevelSlider  *widget.Slider
	VoxLevelLabel   *widget.Text
	CompanderToggle *widget.Button
	CompanderSlider *widget.Slider
	CompanderLabel  *widget.Text
	ProcessorToggle *widget.Button
	ProcessorSlider *widget.Slider
	ProcessorLabel  *widget.Text
	CarrierSlider   *widget.Slider
	CarrierLabel    *widget.Text
	TunePowerSlider *widget.Slider
	TunePowerLabel  *widget.Text
	RFPowerSlider   *widget.Slider
	RFPowerLabel    *widget.Text
}

func (u *UI) MakeTransmitSettingsWindow() *TransmitSettings {
	ts := &TransmitSettings{}

	// Get cached transmit parameters
	u.mu.RLock()
	params := make(map[string]string)
	for k, v := range u.transmitParams {
		params[k] = v
	}
	u.mu.RUnlock()

	contents := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(12),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			StretchVertical:   true,
			StretchHorizontal: true,
		})),
	)

	// Microphone Selection
	currentMic := params["mic_selection"]
	if currentMic == "" {
		currentMic = "Unknown"
	}
	micRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	micLabel := widget.NewText(
		widget.TextOpts.Text("Microphone", u.Font("Roboto-16"), colornames.White),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.MinSize(120, 0)),
	)
	ts.MicSelection = u.MakeButton("Roboto-16", currentMic, func(args *widget.ButtonClickedEventArgs) {
		// Get mic list from radio
		u.RadioShim.GetMicList(func(mics []string) {
			if len(mics) == 0 {
				return
			}
			// Find current selection from button label
			currentLabel := ts.MicSelection.Text().Label
			var selected interface{} = currentLabel
			for _, mic := range mics {
				if mic == currentLabel {
					selected = mic
					break
				}
			}
			// Show list window
			window := u.MakeListWindow(
				"Select Microphone",
				"Roboto-24",
				"",
				"Roboto-16",
				stringSliceToAny(mics),
				selected,
				func(item any) string {
					return item.(string)
				},
				func(item any, ok bool) {
					if ok && item != nil {
						micName := item.(string)
						ts.MicSelection.Text().Label = micName
						u.RadioShim.SetMicInput(micName)
					}
				},
			)
			u.ShowWindow(window)
		})
	})
	micRow.AddChild(micLabel)
	micRow.AddChild(ts.MicSelection)
	contents.AddChild(micRow)

	// Mic Level
	micInitial := getIntParam(params, "mic_level", 50)
	micLevelRow := u.makeSliderRow("Mic Level", 0, 100, micInitial, func(value int) string {
		return formatPercent(value)
	}, func(value int) {
		u.RadioShim.SetMicLevel(value)
	})
	ts.MicLevelSlider = micLevelRow.slider
	ts.MicLevelLabel = micLevelRow.label
	contents.AddChild(micLevelRow.container)

	// VOX
	voxLevelInitial := getIntParam(params, "vox_level", 50)
	voxRow := u.makeToggleSliderRow("VOX", 0, 100, voxLevelInitial, func(enabled bool) {
		u.RadioShim.SetTransmitParam("vox_enable", boolToInt(enabled))
	}, func(value int) string {
		return formatPercent(value)
	}, func(value int) {
		u.RadioShim.SetTransmitParam("vox_level", value)
	})
	ts.VoxEnableToggle = voxRow.toggle
	ts.VoxLevelSlider = voxRow.slider
	ts.VoxLevelLabel = voxRow.label
	// Set initial toggle state
	if params["vox_enable"] == "1" {
		ts.VoxEnableToggle.SetState(widget.WidgetChecked)
	}
	contents.AddChild(voxRow.container)

	// Compander (DEXP)
	dexpLevelInitial := getIntParam(params, "compander_level", 50)
	dexpRow := u.makeToggleSliderRow("DEXP", 0, 100, dexpLevelInitial, func(enabled bool) {
		u.RadioShim.SetTransmitParam("compander", boolToInt(enabled))
	}, func(value int) string {
		return formatPercent(value)
	}, func(value int) {
		u.RadioShim.SetTransmitParam("compander_level", value)
	})
	ts.CompanderToggle = dexpRow.toggle
	ts.CompanderSlider = dexpRow.slider
	ts.CompanderLabel = dexpRow.label
	// Set initial toggle state
	if params["compander"] == "1" {
		ts.CompanderToggle.SetState(widget.WidgetChecked)
	}
	contents.AddChild(dexpRow.container)

	// Processor (PROC)
	procLevelInitial := getIntParam(params, "speech_processor_level", 0)
	procRow := u.makeToggleSliderRow("PROC", 0, 2, procLevelInitial, func(enabled bool) {
		u.RadioShim.SetTransmitParam("speech_processor_enable", boolToInt(enabled))
	}, func(value int) string {
		return formatProcLevel(value)
	}, func(value int) {
		u.RadioShim.SetTransmitParam("speech_processor_level", value)
	})
	ts.ProcessorToggle = procRow.toggle
	ts.ProcessorSlider = procRow.slider
	ts.ProcessorLabel = procRow.label
	// Set initial toggle state
	if params["speech_processor_enable"] == "1" {
		ts.ProcessorToggle.SetState(widget.WidgetChecked)
	}
	contents.AddChild(procRow.container)

	// AM Carrier
	carrierInitial := getIntParam(params, "am_carrier_level", 50)
	carrierRow := u.makeSliderRow("Carrier", 0, 100, carrierInitial, func(value int) string {
		return formatPercent(value)
	}, func(value int) {
		u.RadioShim.SetAMCarrierLevel(value)
	})
	ts.CarrierSlider = carrierRow.slider
	ts.CarrierLabel = carrierRow.label
	contents.AddChild(carrierRow.container)

	// Tune Power
	tuneInitial := getIntParam(params, "tunepower", 10)
	tuneRow := u.makeSliderRow("TUNE", 0, 100, tuneInitial, func(value int) string {
		return formatPercent(value)
	}, func(value int) {
		u.RadioShim.SetTransmitParam("tunepower", value)
	})
	ts.TunePowerSlider = tuneRow.slider
	ts.TunePowerLabel = tuneRow.label
	contents.AddChild(tuneRow.container)

	// RF Power
	rfPowerInitial := getIntParam(params, "rfpower", 50)
	rfPowerRow := u.makeSliderRow("RF Power", 0, 100, rfPowerInitial, func(value int) string {
		return formatPercent(value)
	}, func(value int) {
		u.RadioShim.SetTransmitParam("rfpower", value)
	})
	ts.RFPowerSlider = rfPowerRow.slider
	ts.RFPowerLabel = rfPowerRow.label
	contents.AddChild(rfPowerRow.container)

	// Close button
	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	buttonRow.AddChild(
		u.MakeButton("Roboto-16", "Close", func(_ *widget.ButtonClickedEventArgs) {
			ts.Window.widget.Close()
		}, widget.WidgetOpts.LayoutData(
			widget.RowLayoutData{Stretch: true},
		)),
	)
	contents.AddChild(buttonRow)

	ts.Window = u.MakeWindow("Transmit Settings", "Roboto-24", contents)
	return ts
}

type sliderRow struct {
	container *widget.Container
	slider    *widget.Slider
	label     *widget.Text
}

type toggleSliderRow struct {
	container *widget.Container
	toggle    *widget.Button
	slider    *widget.Slider
	label     *widget.Text
}

func (u *UI) makeSliderRow(name string, min, max, initial int, formatter func(int) string, onChange func(int)) sliderRow {
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)

	nameLabel := widget.NewText(
		widget.TextOpts.Text(name, u.Font("Roboto-16"), colornames.White),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.MinSize(120, 0)),
	)

	slider := widget.NewSlider(
		widget.SliderOpts.MinMax(min, max),
		widget.SliderOpts.InitialCurrent(initial),
		widget.SliderOpts.Images(u.SliderTrackImage(), u.SliderHandleImage()),
		widget.SliderOpts.MinHandleSize(20),
		widget.SliderOpts.PageSizeFunc(func() int { return 1 }),
		widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(2)),
		widget.SliderOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
			widget.WidgetOpts.MinSize(200, 20),
		),
	)

	valueLabel := widget.NewText(
		widget.TextOpts.Text(formatter(initial), u.Font("Roboto-16"), colornames.White),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.MinSize(60, 0)),
	)

	slider.ChangedEvent.AddHandler(func(e interface{}) {
		args := e.(*widget.SliderChangedEventArgs)
		valueLabel.Label = formatter(args.Current)
		onChange(args.Current)
	})

	row.AddChild(nameLabel)
	row.AddChild(slider)
	row.AddChild(valueLabel)

	return sliderRow{
		container: row,
		slider:    slider,
		label:     valueLabel,
	}
}

func (u *UI) makeToggleSliderRow(name string, min, max, initial int, onToggle func(bool), formatter func(int) string, onChange func(int)) toggleSliderRow {
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)

	toggle := u.MakeToggleButton("Roboto-16", name, func(args *widget.ButtonChangedEventArgs) {
		onToggle(args.State == widget.WidgetChecked)
	})
	toggle.GetWidget().LayoutData = widget.RowLayoutData{
		Position: widget.RowLayoutPositionCenter,
	}

	slider := widget.NewSlider(
		widget.SliderOpts.MinMax(min, max),
		widget.SliderOpts.InitialCurrent(initial),
		widget.SliderOpts.Images(u.SliderTrackImage(), u.SliderHandleImage()),
		widget.SliderOpts.MinHandleSize(20),
		widget.SliderOpts.PageSizeFunc(func() int { return 1 }),
		widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(2)),
		widget.SliderOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
			widget.WidgetOpts.MinSize(200, 20),
		),
	)

	valueLabel := widget.NewText(
		widget.TextOpts.Text(formatter(initial), u.Font("Roboto-16"), colornames.White),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.MinSize(60, 0)),
	)

	slider.ChangedEvent.AddHandler(func(e interface{}) {
		args := e.(*widget.SliderChangedEventArgs)
		valueLabel.Label = formatter(args.Current)
		onChange(args.Current)
	})

	row.AddChild(toggle)
	row.AddChild(slider)
	row.AddChild(valueLabel)

	return toggleSliderRow{
		container: row,
		toggle:    toggle,
		slider:    slider,
		label:     valueLabel,
	}
}

func formatPercent(value int) string {
	return fmt.Sprintf("%d%%", value)
}

func formatProcLevel(value int) string {
	switch value {
	case 0:
		return "NORM"
	case 1:
		return "DX"
	case 2:
		return "DX+"
	default:
		return "NORM"
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (u *UI) ShowTransmitSettings() {
	if u.Widgets.WaterfallPage.TransmitSettings == nil {
		u.Widgets.WaterfallPage.TransmitSettings = u.MakeTransmitSettingsWindow()
	}
	u.ShowWindow(u.Widgets.WaterfallPage.TransmitSettings.Window)
}

func (u *UI) UpdateTransmitSettings(params map[string]string) {
	if u.Widgets.WaterfallPage.TransmitSettings == nil {
		return
	}
	ts := u.Widgets.WaterfallPage.TransmitSettings

	// Update mic selection
	if val, ok := params["mic_selection"]; ok {
		if val != "" {
			ts.MicSelection.Text().Label = val
		}
	}

	// Update mic level
	if val, ok := params["mic_level"]; ok {
		if intVal := parseInt(val); intVal >= 0 {
			ts.MicLevelSlider.Current = intVal
			ts.MicLevelLabel.Label = formatPercent(intVal)
		}
	}

	// Update VOX enable and level
	if val, ok := params["vox_enable"]; ok {
		state := widget.WidgetUnchecked
		if val == "1" {
			state = widget.WidgetChecked
		}
		ts.VoxEnableToggle.SetState(state)
	}
	if val, ok := params["vox_level"]; ok {
		if intVal := parseInt(val); intVal >= 0 {
			ts.VoxLevelSlider.Current = intVal
			ts.VoxLevelLabel.Label = formatPercent(intVal)
		}
	}

	// Update compander enable and level
	if val, ok := params["compander"]; ok {
		state := widget.WidgetUnchecked
		if val == "1" {
			state = widget.WidgetChecked
		}
		ts.CompanderToggle.SetState(state)
	}
	if val, ok := params["compander_level"]; ok {
		if intVal := parseInt(val); intVal >= 0 {
			ts.CompanderSlider.Current = intVal
			ts.CompanderLabel.Label = formatPercent(intVal)
		}
	}

	// Update speech processor enable and level
	if val, ok := params["speech_processor_enable"]; ok {
		state := widget.WidgetUnchecked
		if val == "1" {
			state = widget.WidgetChecked
		}
		ts.ProcessorToggle.SetState(state)
	}
	if val, ok := params["speech_processor_level"]; ok {
		if intVal := parseInt(val); intVal >= 0 && intVal <= 2 {
			ts.ProcessorSlider.Current = intVal
			ts.ProcessorLabel.Label = formatProcLevel(intVal)
		}
	}

	// Update AM carrier level
	if val, ok := params["am_carrier_level"]; ok {
		if intVal := parseInt(val); intVal >= 0 {
			ts.CarrierSlider.Current = intVal
			ts.CarrierLabel.Label = formatPercent(intVal)
		}
	}

	// Update tune power
	if val, ok := params["tunepower"]; ok {
		if intVal := parseInt(val); intVal >= 0 {
			ts.TunePowerSlider.Current = intVal
			ts.TunePowerLabel.Label = formatPercent(intVal)
		}
	}

	// Update RF power
	if val, ok := params["rfpower"]; ok {
		if intVal := parseInt(val); intVal >= 0 {
			ts.RFPowerSlider.Current = intVal
			ts.RFPowerLabel.Label = formatPercent(intVal)
		}
	}
}

func parseInt(s string) int {
	result := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			return -1
		}
	}
	return result
}

func getIntParam(params map[string]string, key string, defaultVal int) int {
	if val, ok := params[key]; ok {
		if intVal := parseInt(val); intVal >= 0 {
			return intVal
		}
	}
	return defaultVal
}

func stringSliceToAny(strs []string) []any {
	result := make([]any, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}
