package ui

import (
	"image"

	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type HookContainer struct {
	child      *widget.Container
	updateHook func(*HookContainer)
	renderHook func(*HookContainer, *ebiten.Image)
}

type HookContainerOpt func(c *HookContainer)

type HookContainerOptions struct{}

var HookContainerOpts HookContainerOptions

// UpdateHook sets the container's Update() hook. If it's nil, the child's
// Update() will be called. If it's set, it should call UpdateChild()
// itself on the HookContainer (which is provided as an arg to the hook)
// when and if it wants that to happen.
func (o HookContainerOptions) UpdateHook(hook func(*HookContainer)) HookContainerOpt {
	return func(c *HookContainer) {
		c.updateHook = hook
	}
}

// RenderHook sets the container's Render() hook. If it's nil, the child's
// Render() will be called. If it's set, it should call RenderChild()
// itself on the HookContainer (which is provided as an arg to the hook)
// when and if it wants that to happen.
func (o HookContainerOptions) RenderHook(hook func(*HookContainer, *ebiten.Image)) HookContainerOpt {
	return func(c *HookContainer) {
		c.renderHook = hook
	}
}

// Child sets the HookContainer's child, which must itself be a container.
func (o HookContainerOptions) Child(child *widget.Container) HookContainerOpt {
	return func(c *HookContainer) {
		c.SetChild(child)
	}
}

func NewHookContainer(opt ...HookContainerOpt) *HookContainer {
	c := &HookContainer{}
	for _, opt := range opt {
		opt(c)
	}
	return c
}

func (c *HookContainer) SetChild(child *widget.Container) {
	c.child = child
}

func (c *HookContainer) Update() {
	if c.updateHook != nil {
		c.updateHook(c)
	} else {
		c.UpdateChild()
	}
}

func (c *HookContainer) UpdateChild() {
	c.child.Update()
}

func (c *HookContainer) Render(screen *ebiten.Image) {
	if c.renderHook != nil {
		c.renderHook(c, screen)
	} else {
		c.RenderChild(screen)
	}
}

func (c *HookContainer) RenderChild(screen *ebiten.Image) {
	c.child.Render(screen)
}

// GetWidget implements HasWidget and PreferredSizeLocateableWidget
func (c *HookContainer) GetWidget() *widget.Widget {
	return c.child.GetWidget()
}

// PreferredSize implements PreferredSizer and PreferredSizeLocateableWidget
func (c *HookContainer) PreferredSize() (int, int) {
	return c.child.PreferredSize()
}

// SetLocation implements Locateable and PreferredSizeLocateableWidget
func (c *HookContainer) SetLocation(rect image.Rectangle) {
	c.child.SetLocation(rect)
}

// WidgetAt implements WidgetLocator
func (c *HookContainer) WidgetAt(x, y int) widget.HasWidget {
	return c.child.WidgetAt(x, y)
}

// SetupInputLayer implements InputLayerer
func (c *HookContainer) SetupInputLayer(def input.DeferredSetupInputLayerFunc) {
	c.child.SetupInputLayer(def)
}
