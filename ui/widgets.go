package ui

import (
	"image/color"

	"golang.org/x/image/colornames"

	ebimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (u *UI) MakeButton(fontName string, text string, handler func(*widget.ButtonClickedEventArgs), wopts ...widget.WidgetOpt) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Text(text, u.Font(fontName), &widget.ButtonTextColor{
			Idle:     colornames.White,
			Disabled: colornames.Gray,
			Hover:    colornames.Lightskyblue,
			Pressed:  colornames.Yellow,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(4)),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:         ebimage.NewNineSliceColor(colornames.Dimgray),
			Hover:        ebimage.NewNineSliceColor(colornames.Dimgray),
			Pressed:      ebimage.NewNineSliceColor(colornames.Dimgray),
			PressedHover: ebimage.NewNineSliceColor(colornames.Dimgray),
			Disabled:     ebimage.NewNineSliceColor(colornames.Dimgray),
		}),
		widget.ButtonOpts.ClickedHandler(handler),
		widget.ButtonOpts.WidgetOpts(wopts...),
	)
}

func (u *UI) MakeToggleButton(fontName string, text string, handler func(*widget.ButtonChangedEventArgs), wopts ...widget.WidgetOpt) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Text(text, u.Font(fontName), &widget.ButtonTextColor{
			Idle:     colornames.White,
			Disabled: colornames.Gray,
			Hover:    colornames.Lightskyblue,
			Pressed:  colornames.Yellow,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(4)),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:         ebimage.NewNineSliceColor(colornames.Dimgray),
			Hover:        ebimage.NewNineSliceColor(colornames.Dimgray),
			Pressed:      ebimage.NewNineSliceColor(colornames.Dimgray),
			PressedHover: ebimage.NewNineSliceColor(colornames.Dimgray),
			Disabled:     ebimage.NewNineSliceColor(colornames.Dimgray),
		}),
		widget.ButtonOpts.ToggleMode(),
		widget.ButtonOpts.StateChangedHandler(handler),
		widget.ButtonOpts.WidgetOpts(wopts...),
	)
}

func (u *UI) MakeList(fontName string, labeler func(e any) string, handler func(*widget.ListEntrySelectedEventArgs)) *widget.List {
	return widget.NewList(
		widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				StretchVertical:   true,
				StretchHorizontal: true,
				Padding:           widget.NewInsetsSimple(10),
			}),
		)),
		widget.ListOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
				Idle:     ebimage.NewNineSliceColor(colornames.Dimgray),
				Disabled: ebimage.NewNineSliceColor(colornames.Dimgray),
				Mask:     ebimage.NewNineSliceColor(colornames.Dimgray),
			}),
		),
		widget.ListOpts.SliderOpts(
			// Set the background images/color for the background of the slider track
			widget.SliderOpts.Images(&widget.SliderTrackImage{
				Idle:  ebimage.NewNineSliceColor(colornames.Dimgray),
				Hover: ebimage.NewNineSliceColor(colornames.Dimgray),
			}, sliderImage()),
			widget.SliderOpts.MinHandleSize(5),
			// Set how wide the track should be
			widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(2))),
		// Hide the horizontal slider
		widget.ListOpts.HideHorizontalSlider(),
		// Set the font for the list options
		widget.ListOpts.EntryFontFace(u.Font(fontName)),
		// Set the colors for the list
		widget.ListOpts.EntryColor(&widget.ListEntryColor{
			Selected:           colornames.White,
			SelectedBackground: colornames.Darkcyan,
			Unselected:         colornames.White,
			DisabledSelected:   colornames.Lightgray,
			DisabledUnselected: colornames.Lightgray,
			FocusedBackground:  colornames.Darkcyan,
		}),
		// This required function returns the string displayed in the list
		widget.ListOpts.EntryLabelFunc(labeler),
		// Padding for each entry
		widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(5)),
		// Text position for each entry
		widget.ListOpts.EntryTextPosition(widget.TextPositionStart, widget.TextPositionCenter),
		// This handler defines what function to run when a list item is selected.
		widget.ListOpts.EntrySelectedHandler(handler),
	)
}

func (u *UI) MakeText(fontName string, fgColor color.Color, opts ...widget.TextOpt) *widget.Text {
	opts = append(
		[]widget.TextOpt{
			widget.TextOpts.Text("", u.Font(fontName), fgColor),
		},
		opts...,
	)
	return widget.NewText(opts...)
}

func (u *UI) MakeTextArea(fontName string, fgColor color.Color, bgColor color.Color) *widget.TextArea {
	return widget.NewTextArea(
		widget.TextAreaOpts.FontFace(u.Font(fontName)),
		widget.TextAreaOpts.TextPadding(widget.NewInsetsSimple(4)),
		widget.TextAreaOpts.FontColor(fgColor),
		widget.TextAreaOpts.ScrollContainerOpts(
			widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
				Idle: ebimage.NewNineSliceColor(bgColor),
				Mask: ebimage.NewNineSliceColor(bgColor),
			}),
		),
		widget.TextAreaOpts.SliderOpts(
			widget.SliderOpts.Images(
				&widget.SliderTrackImage{},
				&widget.ButtonImage{},
			),
		),
	)
}

func (u *UI) MakeRoundedRect(fg color.Color, bg color.Color, radius int, opts ...widget.ContainerOpt) *widget.Container {
	img := ebiten.NewImage(2*radius+1, 2*radius+1)
	r := float32(radius)
	img.Fill(bg)
	vector.DrawFilledCircle(img, r, r, r, fg, true)
	nineslice := ebimage.NewNineSliceSimple(img, radius, 1)
	opts = append([]widget.ContainerOpt{
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{
				Left:   radius,
				Right:  radius,
				Top:    0,
				Bottom: 0,
			}),
		)),
		widget.ContainerOpts.BackgroundImage(nineslice)},
		opts...,
	)
	return widget.NewContainer(opts...)
}

func sliderImage() *widget.ButtonImage {
	return &widget.ButtonImage{
		Idle:    ebimage.NewNineSliceColor(colornames.Lightgray),
		Hover:   ebimage.NewNineSliceColor(colornames.Seashell),
		Pressed: ebimage.NewNineSliceColor(colornames.Seashell),
	}
}

func NewNineSliceBorder(innerColor, borderColor color.Color, borderWidthHeight int) *ebimage.NineSlice {
	i := ebiten.NewImage(2*borderWidthHeight+1, 2*borderWidthHeight+1)
	i.Fill(borderColor)
	i.Set(borderWidthHeight, borderWidthHeight, innerColor)
	return ebimage.NewNineSliceSimple(i, borderWidthHeight, 1)
}
