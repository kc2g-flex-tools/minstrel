package midi

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"

	"github.com/kc2g-flex-tools/minstrel/events"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
)

type Config struct {
	Enabled         bool
	Port            string
	VFOControl      byte
	VolControl      byte
	LeftPaddleNote  byte
	RightPaddleNote byte
	PTTNote         byte
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:         false,
		Port:            "",
		VFOControl:      100,
		VolControl:      102,
		LeftPaddleNote:  20,
		RightPaddleNote: 21,
		PTTNote:         31,
	}
}

// Device represents a MIDI input device
type Device struct {
	Name string
}

// ControlEventType identifies the type of control event
type ControlEventType int

const (
	VFOControl ControlEventType = iota
	VolumeControl
)

// ControlEvent represents a MIDI control change event
type ControlEvent struct {
	Type  ControlEventType
	Delta int
}

type MIDI struct {
	mu           sync.RWMutex
	cfg          *Config
	eventBus     *events.Bus
	cancel       context.CancelFunc
	connected    bool
	lastError    string
	currentPort  string
	controlChan  chan ControlEvent
	workerCancel context.CancelFunc
}

func NewMIDI(cfg *Config, eventBus *events.Bus) *MIDI {
	return &MIDI{
		cfg:         cfg,
		eventBus:    eventBus,
		controlChan: make(chan ControlEvent, 100), // Buffered channel for non-blocking sends
	}
}

// GetDevices returns a list of available MIDI input devices
func GetDevices() []Device {
	ports := midi.GetInPorts()
	devices := make([]Device, len(ports))
	for i, port := range ports {
		devices[i] = Device{Name: port.String()}
	}
	return devices
}

// startControlWorker starts a goroutine that processes control events with rate limiting
func (m *MIDI) startControlWorker(ctx context.Context, rs radioshim.Shim) {
	workerCtx, cancel := context.WithCancel(ctx)
	m.workerCancel = cancel

	go func() {
		const rateLimit = 50 * time.Millisecond // Max 20 events per second

		// Accumulators for each control type
		vfoAccumulator := 0
		volumeAccumulator := 0

		timer := time.NewTimer(rateLimit)
		timer.Stop() // Stop initially, will be started on first event
		timerActive := false

		for {
			select {
			case <-workerCtx.Done():
				timer.Stop()
				return

			case event := <-m.controlChan:
				// Accumulate the delta based on event type
				switch event.Type {
				case VFOControl:
					vfoAccumulator += event.Delta
				case VolumeControl:
					volumeAccumulator += event.Delta
				}

				if !timerActive {
					// No pending timer - send immediately and start timer
					m.applyControlEvents(rs, &vfoAccumulator, &volumeAccumulator)
					timer.Reset(rateLimit)
					timerActive = true
				}
				// If timer is active, just accumulate (will be sent when timer fires)

			case <-timer.C:
				// Timer expired - send any accumulated events
				if vfoAccumulator != 0 || volumeAccumulator != 0 {
					m.applyControlEvents(rs, &vfoAccumulator, &volumeAccumulator)
					// Restart timer in case more events come
					timer.Reset(rateLimit)
				} else {
					// No accumulated events - mark timer inactive
					timerActive = false
				}
			}
		}
	}()
}

// applyControlEvents applies accumulated control changes to the radio
func (m *MIDI) applyControlEvents(rs radioshim.Shim, vfoAccumulator, volumeAccumulator *int) {
	slices := rs.GetSlices()

	// Apply VFO changes
	if *vfoAccumulator != 0 {
		for _, slice := range slices {
			if slice.Active {
				rs.TuneSliceStep(slice, *vfoAccumulator)
			}
		}
		*vfoAccumulator = 0
	}

	// Apply volume changes
	if *volumeAccumulator != 0 {
		for _, slice := range slices {
			if slice.Active {
				// Volume uses the full accumulated delta (no multiplier)
				newVolume := slice.Volume + *volumeAccumulator
				// Clamp to valid range (0-100)
				if newVolume < 0 {
					newVolume = 0
				}
				if newVolume > 100 {
					newVolume = 100
				}
				rs.SetSliceVolume(slice.Index, newVolume)
			}
		}
		*volumeAccumulator = 0
	}
}

// Connect attempts to connect to the specified MIDI device
func (m *MIDI) Connect(ctx context.Context, portName string, rs radioshim.Shim) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing worker if any
	if m.workerCancel != nil {
		m.workerCancel()
		m.workerCancel = nil
	}

	// Disconnect existing connection if any
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	if portName == "" {
		m.connected = false
		m.lastError = ""
		m.currentPort = ""
		return nil
	}

	in, err := midi.FindInPort(portName)
	if err != nil {
		m.connected = false
		m.lastError = fmt.Sprintf("Failed to open MIDI port: %v", err)
		m.currentPort = ""
		log.Printf("MIDI connection error: %s", m.lastError)
		return err
	}

	listenCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	// Start the control event worker
	m.startControlWorker(ctx, rs)

	stopListen, err := midi.ListenTo(in, func(msg midi.Message, timestamp int32) {
		var ch, id, val byte
		switch {
		case msg.GetNoteStart(&ch, &id, &val):
			switch id {
			case m.cfg.PTTNote:
				rs.SetPTT(true)
			}
		case msg.GetNoteEnd(&ch, &id):
			switch id {
			case m.cfg.PTTNote:
				rs.SetPTT(false)
			}
		case msg.GetControlChange(&ch, &id, &val):
			switch id {
			case m.cfg.VFOControl:
				delta := int(val) - 64
				// Send to worker channel (non-blocking due to buffer)
				select {
				case m.controlChan <- ControlEvent{Type: VFOControl, Delta: delta}:
				default:
					// Channel full - drop event (should be rare with 100 buffer)
					log.Printf("Warning: MIDI control channel full, dropping VFO event")
				}
			case m.cfg.VolControl:
				delta := int(val) - 64
				// Send to worker channel (non-blocking due to buffer)
				select {
				case m.controlChan <- ControlEvent{Type: VolumeControl, Delta: delta}:
				default:
					// Channel full - drop event (should be rare with 100 buffer)
					log.Printf("Warning: MIDI control channel full, dropping volume event")
				}
			}
		default:
			fmt.Printf("MIDI unknown message: %v\n", msg)
		}
	})
	if err != nil {
		m.connected = false
		m.lastError = fmt.Sprintf("Failed to start MIDI listener: %v", err)
		m.currentPort = ""
		cancel()
		if m.workerCancel != nil {
			m.workerCancel()
			m.workerCancel = nil
		}
		log.Printf("MIDI listener error: %s", m.lastError)
		return err
	}

	m.connected = true
	m.lastError = ""
	m.currentPort = portName
	log.Printf("MIDI connected to %s", portName)

	// Start goroutine to handle disconnection
	go func() {
		<-listenCtx.Done()
		stopListen()
		m.mu.Lock()
		m.connected = false
		m.mu.Unlock()
		log.Println("MIDI disconnected")
	}()

	return nil
}

// Disconnect closes the MIDI connection
func (m *MIDI) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.workerCancel != nil {
		m.workerCancel()
		m.workerCancel = nil
	}

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.connected = false
	m.currentPort = ""
}

// Status returns the current connection status and any error message
func (m *MIDI) Status() (connected bool, port string, errorMsg string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected, m.currentPort, m.lastError
}

// Run is deprecated - use Connect instead
func (m *MIDI) Run(ctx context.Context, rs radioshim.Shim) {
	if m.cfg.Port != "" {
		m.Connect(ctx, m.cfg.Port, rs)
	}
}
