package ui

import (
	"fmt"
	"os/exec"

	"github.com/ebitenui/ebitenui/widget"
)

type radioProps = map[string]string

func (u *UI) MakeRadiosPage() {
	u.Widgets.Radios = u.MakeList(
		"Roboto-24",
		func(e any) string {
			switch entry := e.(type) {
			case *radioProps:
				radio := *entry
				return fmt.Sprintf("%s (%s)", radio["nickname"], radio["model"])
			case string:
				return entry
			default:
				return "<error>"
			}
		},
		func(event *widget.ListEntrySelectedEventArgs) {
			cb := u.Callbacks.Connect
			if cb == nil {
				return
			}
			switch entry := event.Entry.(type) {
			case *radioProps:
				radio := *entry
				cb(radio["ip"] + ":" + radio["port"])
			case string:
				switch entry {
				case "Exit":
					u.exit = true
				case "Shutdown":
					exec.Command("systemctl", "poweroff").Run()
				default:
					fmt.Printf("not supported yet\n")
				}
			default:
				fmt.Printf("huh?\n")
			}
		},
	)
}

func (u *UI) SetRadios(radios []radioProps) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.radios = radios
	entries := make([]any, len(radios))
	for i := range radios {
		entries[i] = &radios[i]
	}
	entries = append(entries, "Enter IP address...", "Exit", "Shutdown")
	u.Widgets.Radios.SetEntries(entries)
}
