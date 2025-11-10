package ui

import (
	"fmt"
	"image"
	"image/color"

	ebimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type Window struct {
	widget *widget.Window
}

func (u *UI) MakeWindow(title, titleFont string, content *widget.Container, opts ...widget.WindowOpt) *Window {
	titleFace := u.Font(titleFont)

	titleBar := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(ebimage.NewNineSliceColor(color.NRGBA{0xee, 0xee, 0xee, 0xc0})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(&widget.Insets{
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
	wOpts := append(
		[]widget.WindowOpt{
			widget.WindowOpts.TitleBar(titleBar, tbHeight),
			widget.WindowOpts.Contents(contentWrapper),
			widget.WindowOpts.Modal(),
			widget.WindowOpts.MinSize(tbWidth, 0),
		},
		opts...,
	)
	window := widget.NewWindow(wOpts...)
	return &Window{
		widget: window,
	}
}

// MakeNumericEntryWindow creates either a numeric keypad (touch mode) or regular text entry
func (u *UI) MakeNumericEntryWindow(title, titleFont, prompt, mainFont string, fontSize int, cb func(string, bool)) *Window {
	if u.cfg.Touch {
		return u.MakeNumericKeypadWindow(title, titleFont, prompt, fmt.Sprintf("%s-%d", mainFont, fontSize), fmt.Sprintf("Icons-%d", fontSize), cb)
	}
	return u.MakeEntryWindow(title, titleFont, prompt, fmt.Sprintf("%s-%d", mainFont, fontSize), cb)
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

func (u *UI) MakeNumericKeypadWindow(title, titleFont, prompt, mainFont, iconFont string, cb func(string, bool)) *Window {
	var window *Window
	mainFace := u.Font(mainFont)
	var currentInput string

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

	// Display for current input
	display := widget.NewText(
		widget.TextOpts.Text("", mainFace, color.NRGBA{0xee, 0xee, 0xee, 0xff}),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
				Stretch:  true,
			}),
		),
	)
	displayContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(ebimage.NewNineSliceColor(color.NRGBA{0x44, 0x44, 0x44, 0xff})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(8)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch:  true,
			MaxWidth: 600,
		})),
	)
	displayContainer.AddChild(display)
	contents.AddChild(displayContainer)

	// Helper to update display
	updateDisplay := func() {
		if currentInput == "" {
			display.Label = "0"
		} else {
			display.Label = currentInput
		}
	}
	updateDisplay()

	// Create keypad grid
	keypadGrid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			widget.GridLayoutOpts.Spacing(8, 8),
			widget.GridLayoutOpts.Stretch([]bool{true, true, true}, []bool{true, true, true, true}),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true,
		})),
	)

	// Button factory for digit buttons
	makeDigitButton := func(label string) *widget.Button {
		return u.MakeButton(mainFont, label, func(_ *widget.ButtonClickedEventArgs) {
			currentInput += label
			updateDisplay()
		}, widget.WidgetOpts.LayoutData(widget.GridLayoutData{
			MaxHeight: 60,
		}))
	}

	// Add digit buttons 1-9
	for i := 1; i <= 9; i++ {
		digit := string(rune('0' + i))
		keypadGrid.AddChild(makeDigitButton(digit))
	}

	// Bottom row: Backspace, 0, .
	keypadGrid.AddChild(u.MakeButton(iconFont, "\uE14A", func(_ *widget.ButtonClickedEventArgs) {
		if len(currentInput) > 0 {
			currentInput = currentInput[:len(currentInput)-1]
			updateDisplay()
		}
	}, widget.WidgetOpts.LayoutData(widget.GridLayoutData{
		MaxHeight: 60,
	})))

	keypadGrid.AddChild(makeDigitButton("0"))

	keypadGrid.AddChild(makeDigitButton("."))

	contents.AddChild(keypadGrid)

	// OK and Cancel buttons
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
			cb(currentInput, true)
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

		// Skip initial SetSelectedEntry call (not a user action)
		if args.PreviousEntry == nil {
			return
		}

		// Always close the window on any user click
		window.widget.Close()

		// Only trigger callback if selection actually changed
		if args.Entry != args.PreviousEntry {
			cb(args.Entry, true)
		}
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

// MakeDropdownWindow creates a dropdown-style window without a title bar
func (u *UI) MakeDropdownWindow(triggerWidget widget.HasWidget, items []any, selected any, labeler func(any) string, cb func(any, bool)) *Window {
	var window *Window

	// Create list with fixed layout to prevent sizing issues
	list := u.MakeList("Roboto-16", labeler)
	list.GetWidget().LayoutData = widget.AnchorLayoutData{
		StretchHorizontal: true,
		StretchVertical:   true,
	}
	for _, item := range items {
		list.AddEntry(item)
	}
	if selected != nil {
		list.SetSelectedEntry(selected)
	}
	list.EntrySelectedEvent.AddHandler(func(e any) {
		args := e.(*widget.ListEntrySelectedEventArgs)

		// Skip initial SetSelectedEntry call (not a user action)
		if args.PreviousEntry == nil {
			return
		}

		// Always close the window on any user click
		window.widget.Close()

		// Only trigger callback if selection actually changed
		if args.Entry != args.PreviousEntry {
			cb(args.Entry, true)
		}
	})

	// Create contents with just the list (no buttons, no prompt)
	contents := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(NewNineSliceBorder(color.NRGBA{0x44, 0x44, 0x44, 0xc0}, color.NRGBA{0xee, 0xee, 0xee, 0xc0}, 2)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(8+2)),
		)),
	)
	contents.AddChild(list)

	// Create window without title bar
	win := widget.NewWindow(
		widget.WindowOpts.Contents(contents),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
	)

	window = &Window{widget: win}
	return window
}

// ShowDropdownWindow positions and shows a dropdown window relative to a trigger widget
func (u *UI) ShowDropdownWindow(window *Window, triggerWidget widget.HasWidget) {
	win := window.widget

	// Get trigger widget's rectangle
	triggerRect := triggerWidget.GetWidget().Rect

	// Get window size
	winX, winY := ebiten.WindowSize()

	// Calculate max height (3/4 of screen)
	maxHeight := (winY * 3) / 4

	// Validate and get preferred content size
	win.Contents.RequestRelayout()
	win.Contents.Validate()
	contentWidth, contentHeight := win.Contents.PreferredSize()

	// Limit height to maxHeight
	actualHeight := contentHeight
	if actualHeight > maxHeight {
		actualHeight = maxHeight
	}

	// Start with trigger widget width
	dropdownWidth := triggerRect.Dx()

	// Extend right if content needs more width
	if contentWidth > dropdownWidth {
		dropdownWidth = contentWidth
	}

	// Ensure minimum width for readability
	if dropdownWidth < 200 {
		dropdownWidth = 200
	}

	// Calculate X position (align with left edge of trigger, or shift left if would go off screen)
	x := triggerRect.Min.X
	if x+dropdownWidth > winX {
		x = winX - dropdownWidth
		if x < 0 {
			x = 0
			dropdownWidth = winX
		}
	}

	// Calculate Y position (try below first, then above if no space)
	var y int
	if triggerRect.Max.Y+actualHeight <= winY {
		// Position below
		y = triggerRect.Max.Y
	} else if triggerRect.Min.Y-actualHeight >= 0 {
		// Position above
		y = triggerRect.Min.Y - actualHeight
	} else {
		// Not enough space either way, position below and let height be limited
		y = triggerRect.Max.Y
		// Recalculate height to fit remaining space
		remainingSpace := winY - y
		if remainingSpace < actualHeight {
			actualHeight = remainingSpace
		}
	}

	r := image.Rect(x, y, x+dropdownWidth, y+actualHeight)
	win.SetLocation(r)
	u.eui.AddWindow(win)
	u.Defer(func() {
		u.eui.ChangeFocus(widget.FOCUS_NEXT)
	})
}

func (u *UI) ShowWindow(window *Window) {
	win := window.widget
	win.Contents.Validate()
	win.TitleBar.Validate()
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
