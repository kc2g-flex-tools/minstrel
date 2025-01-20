package ui

import (
	"os/exec"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/colornames"
)

type TopBar struct {
	Container   *widget.Container
	AudioButton *widget.Button
}

func (u *UI) MakeTopBar() *TopBar {
	tb := &TopBar{}
	tb.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Spacing(4),
		)),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(colornames.Black)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, 24),
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
				VerticalPosition:   widget.GridLayoutPositionStart,
			}),
		),
	)
	tb.Container.AddChild(u.MakeButton("Roboto-24", "Exit", func(args *widget.ButtonClickedEventArgs) {
		u.exit = true
	}, widget.WidgetOpts.LayoutData(widget.RowLayoutData{
		Stretch: true,
	})))
	tb.Container.AddChild(u.MakeButton("Roboto-24", "Shutdown", func(args *widget.ButtonClickedEventArgs) {
		cmd := exec.Command("systemctl", "poweroff")
		cmd.Run()
	}, widget.WidgetOpts.LayoutData(widget.RowLayoutData{
		Stretch: true,
	})))
	tb.AudioButton = u.MakeButton("Roboto-24", "Audio", func(args *widget.ButtonClickedEventArgs) {
		u.RadioShim.ToggleAudio()
	}, widget.WidgetOpts.LayoutData(widget.RowLayoutData{
		Stretch: true,
	}))
	tb.AudioButton.GetWidget().Visibility = widget.Visibility_Hide
	tb.Container.AddChild(tb.AudioButton)
	return tb
}
