package ui

import (
	"os/exec"

	"github.com/ebitenui/ebitenui/widget"
)

type WaterfallControls struct {
	Container *widget.Container
	Exit      *widget.Button
	Audio     *widget.Button
	ZoomOut   *widget.Button
	ZoomIn    *widget.Button
	Find      *widget.Button
	MOX       *widget.Button
	VOX       *widget.Button
	Settings  *widget.Button
}

func (u *UI) MakeWaterfallControls() *WaterfallControls {
	wfc := &WaterfallControls{}
	wfc.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(4),
			widget.GridLayoutOpts.Spacing(8, 8),
			widget.GridLayoutOpts.Stretch([]bool{false, false, false, false}, []bool{false}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.GridLayoutData{
					MaxWidth:           192,
					HorizontalPosition: widget.GridLayoutPositionCenter,
					VerticalPosition:   widget.GridLayoutPositionEnd,
				},
			),
		),
	)
	if u.cfg.Kiosk {
		wfc.Exit = u.MakeButton("Icons-32", "\ue8ac", func(args *widget.ButtonClickedEventArgs) {
			exec.Command("systemctl", "poweroff").Run()
		})
	} else {
		wfc.Exit = u.MakeButton("Icons-32", "\ue9ba", func(args *widget.ButtonClickedEventArgs) {
			u.exit = true
		})
	}
	wfc.Audio = u.MakeToggleButton("Icons-32", "\ue050", func(args *widget.ButtonChangedEventArgs) {
		u.RadioShim.ToggleAudio(args.State == widget.WidgetChecked)
	})
	wfc.ZoomOut = u.MakeButton("Icons-32", "\ue900", func(args *widget.ButtonClickedEventArgs) {
		u.RadioShim.ZoomOut()
	})
	wfc.ZoomIn = u.MakeButton("Icons-32", "\ue8ff", func(args *widget.ButtonClickedEventArgs) {
		u.RadioShim.ZoomIn()
	})
	wfc.Find = u.MakeButton("Icons-32", "\uf70c", func(args *widget.ButtonClickedEventArgs) {
		u.RadioShim.FindActiveSlice()
	})
	wfc.MOX = u.MakeToggleButton("Roboto-16", "MOX", func(args *widget.ButtonChangedEventArgs) {
		if args.OffsetX == -1 {
			return
		}
		u.RadioShim.SetPTT(args.State == widget.WidgetChecked)
	})
	wfc.VOX = u.MakeToggleButton("Roboto-16", "VOX", func(args *widget.ButtonChangedEventArgs) {
		if args.OffsetX == -1 {
			return
		}
		u.RadioShim.SetVOX(args.State == widget.WidgetChecked)
	})
	wfc.Settings = u.MakeButton("Icons-32", "\ue8b8", func(args *widget.ButtonClickedEventArgs) {
		u.ShowTransmitSettings()
	})

	wfc.Container.AddChild(
		wfc.Exit,
		wfc.Audio,
		wfc.MOX,
		wfc.VOX,
		wfc.ZoomOut,
		wfc.ZoomIn,
		wfc.Find,
		wfc.Settings,
	)

	return wfc
}
