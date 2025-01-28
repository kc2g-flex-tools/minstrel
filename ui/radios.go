package ui

import (
	"fmt"
	"os/exec"

	"github.com/ebitenui/ebitenui/widget"
)

type radioProps = map[string]string

type RadiosPage struct {
	List *widget.List
}

func (u *UI) MakeRadiosPage() *RadiosPage {
	radios := &RadiosPage{}
	radios.List = u.MakeList(
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
	)
	radios.List.EntrySelectedEvent.AddHandler(
		func(e any) {
			event := e.(*widget.ListEntrySelectedEventArgs)
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
				case "Enter IP address...":
					u.ShowWindow(
						u.MakeEntryWindow("Enter IP", "Roboto-24", "Enter an IP[:port] to connect to a radio", "Roboto-24", func(ip string, ok bool) {
							if ok {
								u.Callbacks.Connect(ip)
							}
						}),
					)
				default:
					fmt.Printf("not supported yet\n")
				}
			default:
				fmt.Printf("huh?\n")
			}
		},
	)
	return radios
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
	u.Widgets.Radios.List.SetEntries(entries)
}
