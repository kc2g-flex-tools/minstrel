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
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{
				Left:  4,
				Right: 4,
			}),
		)),
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
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			StretchHorizontal: true,
			StretchVertical:   true,
		})),
	)
	contentWrapper.AddChild(content)

	tbWidth, tbHeight := titleBar.PreferredSize()
	window := widget.NewWindow(
		widget.WindowOpts.TitleBar(titleBar, tbHeight),
		widget.WindowOpts.Contents(contentWrapper),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.MinSize(tbWidth, 0),
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
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
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

func (u *UI) MakeListWindow(title, titleFont, prompt, mainFont string, items []any, selected any, labeler func(any) string, cb func(any, bool)) *Window {
	var window *Window

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
			widget.TextOpts.Text(prompt, u.Font(mainFont), color.NRGBA{0xee, 0xee, 0xee, 0xff}),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}))))
	}

	list := u.MakeList(mainFont, labeler)
	for _, item := range items {
		list.AddEntry(item)
	}
	if selected != nil {
		list.SetSelectedEntry(selected)
	}
	list.EntrySelectedEvent.AddHandler(func(e any) {
		args := e.(*widget.ListEntrySelectedEventArgs)
		cb(args.Entry, true)
		window.widget.Close()
	})

	contents.AddChild(list)
	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	buttonRow.AddChild(
		u.MakeButton(mainFont, "Cancel", func(_ *widget.ButtonClickedEventArgs) {
			cb(nil, false)
			window.widget.Close()
		}),
	)
	contents.AddChild(buttonRow)
	window = u.MakeWindow(title, titleFont, contents)
	return window
}

func (u *UI) ShowWindow(window *Window) {
	win := window.widget
	win.Contents.Update()
	win.TitleBar.Update()
	contentWidth, contentHeight := win.Contents.PreferredSize()
	tbWidth, tbHeight := win.TitleBar.PreferredSize()

	x := max(contentWidth, tbWidth)
	y := contentHeight + tbHeight

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
