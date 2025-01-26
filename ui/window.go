package ui

import (
	"image"
	"image/color"

	ebimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type Window struct {
	widget *widget.Window
}

func (u *UI) MakeWindow(title, titleFont string, content *widget.Container) *Window {
	titleFace := u.Font(titleFont)

	titleBar := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(ebimage.NewNineSliceColor(color.NRGBA{0xee, 0xee, 0xee, 0xc0})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	titleBar.AddChild(widget.NewText(
		widget.TextOpts.Text(title, titleFace, color.NRGBA{0x44, 0x44, 0x44, 0xff}),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		}))))
	contentWrapper := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(NewNineSliceBorder(color.NRGBA{0x44, 0x44, 0x44, 0xc0}, color.NRGBA{0xee, 0xee, 0xee, 0xc0}, 2)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(8+2)),
		)))
	contentWrapper.AddChild(content)

	window := widget.NewWindow(
		widget.WindowOpts.TitleBar(titleBar, 24),
		widget.WindowOpts.Contents(contentWrapper),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.MinSize(400, 200),
	)
	return &Window{
		widget: window,
	}
}

func (u *UI) MakeEntryWindow(title, titleFont, prompt, mainFont string, cb func(string, bool)) *Window {
	var window *Window
	mainFace := u.Font(mainFont)

	contents := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(16),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			StretchVertical:   true,
			StretchHorizontal: true,
		})),
	)
	if prompt != "" {
		contents.AddChild(widget.NewText(
			widget.TextOpts.Text(prompt, mainFace, color.NRGBA{0xee, 0xee, 0xee, 0xff}),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}))))
	}
	input := widget.NewTextInput(
		widget.TextInputOpts.Face(mainFace),
		widget.TextInputOpts.Color(&widget.TextInputColor{
			Idle:          color.NRGBA{0xee, 0xee, 0xee, 0xff},
			Disabled:      color.NRGBA{0xee, 0xee, 0xee, 0xff},
			Caret:         color.NRGBA{0xee, 0xee, 0xee, 0xff},
			DisabledCaret: color.NRGBA{0xee, 0xee, 0xee, 0xff},
		}),
		widget.TextInputOpts.Image(&widget.TextInputImage{
			Idle:     ebimage.NewNineSliceColor(color.NRGBA{0x44, 0x44, 0x44, 0xff}),
			Disabled: ebimage.NewNineSliceColor(color.NRGBA{0x44, 0x44, 0x44, 0xff}),
		}),
		widget.TextInputOpts.CaretOpts(
			widget.CaretOpts.Color(color.NRGBA{0xee, 0xee, 0xee, 0xff}),
			widget.CaretOpts.Size(mainFace, 2),
		),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			cb(args.InputText, true)
			window.widget.Close()
		}),
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch:  true,
				MaxWidth: 600,
			}),
		),
	)
	contents.AddChild(input)

	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	buttonRow.AddChild(
		u.MakeButton(mainFont, "OK", func(_ *widget.ButtonClickedEventArgs) {
			cb(input.GetText(), true)
			window.widget.Close()
		}, widget.WidgetOpts.LayoutData(
			widget.RowLayoutData{Stretch: true},
		)),
		u.MakeButton(mainFont, "Cancel", func(_ *widget.ButtonClickedEventArgs) {
			cb("", false)
			window.widget.Close()
		}),
	)
	contents.AddChild(buttonRow)
	window = u.MakeWindow(title, titleFont, contents)
	return window
}

func (u *UI) ShowWindow(window *Window) {
	win := window.widget
	x, y := win.Contents.PreferredSize()
	if minSize := win.MinSize; minSize != nil {
		x, y = max(x, minSize.X), max(y, minSize.Y)
	}
	if maxSize := win.MaxSize; maxSize != nil {
		x, y = min(x, maxSize.X), min(y, maxSize.Y)
	}
	winX, winY := ebiten.WindowSize()
	r := image.Rect(0, 0, x, y).Add(image.Point{max((winX-x)/2, 0), max((winY-y)/2, 0)})
	win.SetLocation(r)
	u.eui.AddWindow(win)
	u.Defer(func() {
		u.eui.ChangeFocus(widget.FOCUS_NEXT)
	})
}
