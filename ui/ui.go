package ui

import (
	"image/color"
	"log"
	"sync"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

type callbacks struct {
	Connect func(string)
}

type RadioShim interface {
	ToggleAudio(bool)
	ZoomIn()
	ZoomOut()
	FindActiveSlice()
	GetSlices() map[string]SliceData
	TuneSlice(int, float64)
	SetSliceMode(int, string)
	CenterWaterfallAt(float64)
	ActivateSlice(int)
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
	Callbacks callbacks
	RadioShim RadioShim
	deferred  []func()
}

type Config struct {
	Touch      bool `dialsdesc:"Touchscreen mode" dialsflag:"touch"`
	FPS        int  `dialsdesc:"Framerate" dialsflag:"fps"`
	Fullscreen bool `dialsdesc:"Start in fullscreen"`
}

func NewUI(cfg *Config) *UI {
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
	if cfg.Fullscreen {
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
	if u.Width != width || u.Height != height {
		u.Width = width
		u.Height = height
		log.Printf("layout %d x %d", width, height)
	}
	return width, height
}

func (u *UI) Defer(cb func()) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.deferred = append(u.deferred, cb)
}
