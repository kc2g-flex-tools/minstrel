package ui

import (
	"context"
	"log"

	"github.com/kc2g-flex-tools/minstrel/midi"
	"github.com/kc2g-flex-tools/minstrel/persistence"
)

// GetMIDIDevices retrieves the list of available MIDI devices
func (u *UI) GetMIDIDevices(callback func([]string)) {
	devices := midi.GetDevices()
	deviceNames := make([]string, len(devices))
	for i, dev := range devices {
		deviceNames[i] = dev.Name
	}
	callback(deviceNames)
}

// ConnectMIDIDevice attempts to connect to a MIDI device and updates UI
func (u *UI) ConnectMIDIDevice(deviceName string, ts *TransmitSettings) {
	if u.MIDIShim == nil {
		u.Defer(func() {
			ts.MIDIStatusLabel.Label = "Status: Error - MIDI not available"
		})
		return
	}

	var err error
	if deviceName == "None" || deviceName == "" {
		u.MIDIShim.Disconnect()
		u.Defer(func() {
			ts.MIDIStatusLabel.Label = "Status: Not connected"
		})
		// Save "None" to clear the persisted MIDI device
		deviceName = ""
	} else {
		if u.RadioShim == nil {
			u.Defer(func() {
				ts.MIDIStatusLabel.Label = "Status: Error - Radio not connected"
			})
			return
		}
		// Get a background context for MIDI connection
		ctx := context.Background()
		err = u.MIDIShim.Connect(ctx, deviceName, u.RadioShim)

		u.Defer(func() {
			if err != nil {
				ts.MIDIStatusLabel.Label = "Status: Error - " + err.Error()
			} else {
				ts.MIDIStatusLabel.Label = "Status: Connected"
			}
		})
	}

	// Save the MIDI device selection to persistent storage
	go func() {
		store, err := persistence.NewSettingsStore()
		if err != nil {
			log.Printf("Failed to create settings store: %v", err)
			return
		}

		settings, err := store.Load()
		if err != nil {
			log.Printf("Failed to load settings: %v", err)
			settings = &persistence.Settings{}
		}

		settings.MIDI.Port = deviceName
		if err := store.Save(settings); err != nil {
			log.Printf("Failed to save MIDI settings: %v", err)
		}
	}()
}

// UpdateMIDIStatus updates the MIDI status display based on current connection state
func (u *UI) UpdateMIDIStatus(ts *TransmitSettings) {
	if u.MIDIShim == nil {
		u.Defer(func() {
			ts.MIDIStatusLabel.Label = "Status: MIDI not available"
			ts.MIDIDeviceButton.Text().Label = "None"
		})
		return
	}

	connected, port, errorMsg := u.MIDIShim.Status()

	u.Defer(func() {
		if connected {
			ts.MIDIStatusLabel.Label = "Status: Connected"
			ts.MIDIDeviceButton.Text().Label = port
			ts.selectedMIDIDevice = port
		} else if errorMsg != "" {
			ts.MIDIStatusLabel.Label = "Status: Error - " + errorMsg
			if port != "" {
				ts.MIDIDeviceButton.Text().Label = port
				ts.selectedMIDIDevice = port
			}
		} else {
			ts.MIDIStatusLabel.Label = "Status: Not connected"
			ts.MIDIDeviceButton.Text().Label = "None"
		}
	})
}
