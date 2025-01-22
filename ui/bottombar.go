package ui

import (
	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/colornames"
)

func (u *UI) MakeBottomBar() {
	u.Widgets.BottomBar.AddChild(
		widget.NewText(widget.TextOpts.Text("Bottom Bar", u.Font("Roboto-16"), colornames.White),
			widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionEnd),
		),
	)
}
