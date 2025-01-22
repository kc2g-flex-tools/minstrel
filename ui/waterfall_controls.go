package ui

import "github.com/ebitenui/ebitenui/widget"

type WaterfallControls struct {
	Container *widget.Container
}

func (u *UI) MakeWaterfallControls() *WaterfallControls {
	wfc := &WaterfallControls{}
	wfc.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		// widget.ContainerOpts.BackgroundImage(ebimage.NewNineSliceColor(colornames.Pink)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.RowLayoutData{
					Position: widget.RowLayoutPositionCenter,
					Stretch:  true,
				},
			),
		),
	)
	return wfc
}
