package events

import (
	"github.com/kc2g-flex-tools/minstrel/radioshim"
)

// Event is a marker interface for all radio events
type Event interface {
	isEvent()
}

// Base implementation for all events
type baseEvent struct{}

func (baseEvent) isEvent() {}

// WaterfallDisplayRangeChanged is fired when pan/zoom changes the display range
type WaterfallDisplayRangeChanged struct {
	baseEvent
	Low  float64
	High float64
}

// WaterfallBinsConfigured is fired when waterfall bin width is set
type WaterfallBinsConfigured struct {
	baseEvent
	Width uint16
}

// WaterfallDataRangeChanged is fired when actual data frequency range changes
type WaterfallDataRangeChanged struct {
	baseEvent
	Low  float64
	High float64
}

// WaterfallRowReceived is fired when new waterfall row is ready
type WaterfallRowReceived struct {
	baseEvent
	Bins       []uint16
	BlackLevel uint32
}

// SlicesUpdated is fired when slice data changes
type SlicesUpdated struct {
	baseEvent
	Slices map[string]*radioshim.SliceData
}

// TransmitStateChanged is fired when TX state changes
type TransmitStateChanged struct {
	baseEvent
	Transmitting bool
}

// StreamEstablished is fired when audio/waterfall streams are created
type StreamEstablished struct {
	baseEvent
	Type     string // "waterfall", "rx_audio", "tx_audio"
	StreamID uint32
}

// Bus provides simple event publish/subscribe
type Bus struct {
	subscribers []chan Event
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe creates a new event channel for receiving events
func (b *Bus) Subscribe(bufferSize int) chan Event {
	ch := make(chan Event, bufferSize)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

// Publish sends an event to all subscribers (non-blocking)
func (b *Bus) Publish(event Event) {
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// Skip slow subscribers - prevents RadioState from blocking
		}
	}
}
