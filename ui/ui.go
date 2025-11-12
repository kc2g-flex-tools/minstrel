package ui

import (
	"context"
	"image/color"
	"log"
	"math"
	"sync"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kc2g-flex-tools/minstrel/audioshim"
	"github.com/kc2g-flex-tools/minstrel/events"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
)

type State int

const (
	DiscoveryState State = iota
	MainState
)

type widgets struct {
	Root          *widget.Container
	MainPage      *widget.FlipBook
	Radios        *RadiosPage
	WaterfallPage *WaterfallWidgets
}

type Config struct {
	Touch bool    `dialsdesc:"Touchscreen mode" dialsflag:"touch"`
	FPS   int     `dialsdesc:"Framerate" dialsflag:"fps"`
	Kiosk bool    `dialsdesc:"Kiosk mode (fullscreen, I own the machine)" dialsflag:"kiosk"`
	Scale float64 `dialsdesc:"UI Scale factor" dialsflag:"scale"`
}

func DefaultConfig() *Config {
	return &Config{
		FPS:   30,
		Scale: 1.0,
	}
}

type UI struct {
	mu        sync.RWMutex
	update    bool
	exit      bool
	state     State
	font      map[string]text.Face
	Width     int
	Height    int
	radios    []map[string]string
	eui       *ebitenui.UI
	Widgets   widgets
	RadioShim radioshim.Shim
	AudioShim audioshim.Shim
	MIDIShim  interface {
		Connect(ctx context.Context, portName string, rs radioshim.Shim) error
		Disconnect()
		Status() (connected bool, port string, errorMsg string)
	}
	deferred       []func()
	cfg            *Config
	eventBus       *events.Bus
	transmitParams map[string]string
}

func NewUI(cfg *Config, eventBus *events.Bus) *UI {
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0x12, 0x23, 0x34, 0xff})),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true}),
		)),
	)

	u := &UI{
		state: DiscoveryState,
		eui: &ebitenui.UI{
			Container: rootContainer,
		},
		Widgets: widgets{
			Root: rootContainer,
		},
		cfg:            cfg,
		eventBus:       eventBus,
		transmitParams: make(map[string]string),
	}
	u.MakeLayout()
	u.Widgets.Radios = u.MakeRadiosPage()
	u.MakeWaterfallPage()
	u.Widgets.MainPage.SetPage(u.Widgets.Radios.List)

	ebiten.SetTPS(cfg.FPS)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(720, 480)
	ebiten.SetWindowSizeLimits(720, 480, -1, -1)
	ebiten.SetWindowTitle("Minstrel")
	if cfg.Touch {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	}
	if cfg.Kiosk {
		ebiten.SetFullscreen(true)
	}
	return u
}

func (u *UI) MakeLayout() {
	u.Widgets.MainPage = widget.NewFlipBook(widget.FlipBookOpts.Padding(widget.NewInsetsSimple(4)))
	u.Widgets.Root.AddChild(u.Widgets.MainPage)
}

func (u *UI) Update() error {
	if u.exit || inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if u.state == MainState {
		u.Widgets.WaterfallPage.Update(u)
	}
	u.runDeferred()
	u.eui.Update()
	u.update = true
	return nil
}

func (u *UI) runDeferred() {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, cb := range u.deferred {
		cb()
	}
	u.deferred = nil
}

func (u *UI) Draw(screen *ebiten.Image) {
	if !u.update {
		return
	}
	u.update = false
	screen.Clear()

	u.eui.Draw(screen)
}

func (u *UI) Layout(width, height int) (int, int) {
	newWidth, newHeight := float64(width), float64(height)
	scale := max(u.cfg.Scale, newWidth/2048, newHeight/1152)
	newWidth /= scale
	newHeight /= scale
	width, height = int(math.Round(newWidth)), int(math.Round(newHeight))

	if u.Width != width || u.Height != height {
		u.Width = width
		u.Height = height
		log.Printf("layout %d x %d", width, height)
	}
	return width, height
}

// Defer schedules a callback to run on the next UI update cycle.
// This is required when updating UI state from goroutines to ensure
// thread safety with Ebiten's game loop. The deferred callbacks run
// during Update() under the UI mutex lock.
//
// Usage:
//   - Use this for all UI updates from event handlers
//   - Do NOT call UI methods directly from goroutines
//   - Callbacks should be fast and non-blocking
//
// Thread Safety:
//
//	The deferred queue is protected by u.mu and executed serially
//	during runDeferred(), which is called from Update().
func (u *UI) Defer(cb func()) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.deferred = append(u.deferred, cb)
}

// HandleEvents processes events from the event bus and updates the UI
func (u *UI) HandleEvents(eventChan chan events.Event) {
	for event := range eventChan {
		switch e := event.(type) {
		case events.RadiosDiscovered:
			u.Defer(func() {
				u.SetRadios(e.Radios)
			})

		case events.RadioConnected:
			u.Defer(func() {
				u.ShowWaterfall()
			})

		case events.TransmitStateChanged:
			u.Defer(func() {
				state := widget.WidgetUnchecked
				if e.Transmitting {
					state = widget.WidgetChecked
				}
				u.Widgets.WaterfallPage.Controls.MOX.SetState(state)
			})

		case events.VOXStateChanged:
			u.Defer(func() {
				state := widget.WidgetUnchecked
				if e.Enabled {
					state = widget.WidgetChecked
				}
				u.Widgets.WaterfallPage.Controls.VOX.SetState(state)
			})

		case events.TransmitParamsChanged:
			u.Defer(func() {
				// Cache the parameters (already under u.mu)
				for k, v := range e.Params {
					u.transmitParams[k] = v
				}
				// Update the window if it exists
				u.UpdateTransmitSettings(e.Params)
			})

		case events.SlicesUpdated:
			u.Defer(func() {
				u.Widgets.WaterfallPage.UpdateSlices(e.Slices)
			})

		case events.WaterfallDisplayRangeChanged:
			u.Defer(func() {
				wf := u.Widgets.WaterfallPage.Waterfall
				wf.DispLow = e.Low
				wf.DispHigh = e.High
			})

		case events.WaterfallBinsConfigured:
			u.Defer(func() {
				u.Widgets.WaterfallPage.Waterfall.SetBins(e.Width)
			})

		case events.WaterfallDataRangeChanged:
			u.Defer(func() {
				wf := u.Widgets.WaterfallPage.Waterfall
				wf.DataLow = e.Low
				wf.DataHigh = e.High
			})

		case events.WaterfallRowReceived:
			u.Defer(func() {
				u.Widgets.WaterfallPage.Waterfall.AddRow(e.Bins, e.BlackLevel)
			})
		}
	}
}
