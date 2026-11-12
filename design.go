package main

import (
	"math"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/flowfile"
	"github.com/oneo-me/nsgo-flow/locales"
)

// The design canvas's zoom limits and steps, as the flow chart's.
const (
	designMinZoom   = .1
	designMaxZoom   = 4
	designZoomStep  = 1.25
	designFitMargin = 32
	// designWheelZoom and designPinchZoom are the scroll distances in
	// points that zoom by ×1.5: a wheel's notches, and a touchpad's scroll
	// with Control or Command held.
	designWheelZoom = 120
	designPinchZoom = 48
)

// designCanvas is the camera over a screen's design: the design is laid
// out at its device's size on a plane that the camera moves and scales,
// fitted and centered until the user pans or zooms it.
type designCanvas struct {
	cx                 *core.Scope
	x, y, zoom         *core.AnimatedValue
	cam                struct{ x, y, zoom float64 }
	width, height      float64
	contentW, contentH float64
	moved              bool
	// pan is where the camera was when a drag began, and pinchFrom the
	// pinch's scale at its last change.
	pan       [2]float64
	pinchFrom float64
	run       *core.AnimationRun
}

func newDesignCanvas(cx *core.Scope) *designCanvas {
	c := &designCanvas{cx: cx, x: core.NewAnimatedValue(cx, 0), y: core.NewAnimatedValue(cx, 0), zoom: core.NewAnimatedValue(cx, 1)}
	c.cam.zoom = 1
	return c
}

// move sends the camera to x, y and zoom, at once or easing there.
func (c *designCanvas) move(x, y, zoom float64, animated bool) {
	c.cam.x, c.cam.y, c.cam.zoom = x, y, zoom
	if c.run != nil {
		c.run.Stop()
	}
	if !animated || core.Untracked(c.cx, c.cx.Env().ReduceMotion) {
		c.x.Set(x)
		c.y.Set(y)
		c.zoom.Set(zoom)
		return
	}
	spring := core.SpringConfig{Stiffness: 666, Damping: 71, Mass: 1}
	c.run = c.cx.StartAnimation(core.Parallel(core.Spring(c.x, x, spring), core.Spring(c.y, y, spring), core.Spring(c.zoom, zoom, spring)), nil)
}

// fit shows the whole design centered, no larger than its size.
func (c *designCanvas) fit(animated bool) {
	if c.width == 0 || c.contentW == 0 {
		return
	}
	zoom := math.Max(designMinZoom, math.Min(1, math.Min((c.width-2*designFitMargin)/c.contentW, (c.height-2*designFitMargin)/c.contentH)))
	c.moved = false
	c.move((c.width-c.contentW*zoom)/2, (c.height-c.contentH*zoom)/2, zoom, animated)
}

// zoomAt scales the design around a point of the view.
func (c *designCanvas) zoomAt(x, y, factor float64, animated bool) {
	zoom := math.Max(designMinZoom, math.Min(designMaxZoom, c.cam.zoom*factor))
	gx, gy := (x-c.cam.x)/c.cam.zoom, (y-c.cam.y)/c.cam.zoom
	c.moved = true
	c.move(x-gx*zoom, y-gy*zoom, zoom, animated)
}

func (c *designCanvas) zoomBy(factor float64) { c.zoomAt(c.width/2, c.height/2, factor, true) }

// resize keeps a design the user has not moved fitted, and otherwise
// keeps the view's center where it is.
func (c *designCanvas) resize(b core.Bounds) {
	w, h := float64(b.Size.Width), float64(b.Size.Height)
	if w == c.width && h == c.height {
		return
	}
	dx, dy := (w-c.width)/2, (h-c.height)/2
	first := c.width == 0
	c.width, c.height = w, h
	if first || !c.moved {
		c.fit(false)
		return
	}
	c.move(c.cam.x+dx, c.cam.y+dy, c.cam.zoom, false)
}

// measured takes the design's size, fitting it while the user has not
// moved it.
func (c *designCanvas) measured(b core.Bounds) {
	w, h := float64(b.Size.Width), float64(b.Size.Height)
	if w == c.contentW && h == c.contentH {
		return
	}
	c.contentW, c.contentH = w, h
	if !c.moved {
		c.fit(false)
	}
}

// wheel pans and zooms as the flow chart does: a touchpad's two fingers
// pan unless Control or Command is held, which zooms, and a wheel zooms.
func (c *designCanvas) wheel(e core.WheelEvent) {
	modified := e.Modifiers.Control || e.Modifiers.Platform
	switch {
	case e.Phase == core.GestureEnded || e.Phase == core.GestureCancelled:
	case e.Touchpad && !modified:
		c.moved = true
		c.move(c.cam.x+float64(e.Delta.X), c.cam.y+float64(e.Delta.Y), c.cam.zoom, false)
	case !e.Touchpad && !modified && e.Delta.Y == 0:
		c.moved = true
		c.move(c.cam.x+float64(e.Delta.X), c.cam.y, c.cam.zoom, false)
	default:
		distance := float64(designWheelZoom)
		if e.Touchpad {
			distance = designPinchZoom
		}
		c.zoomAt(float64(e.Position.X), float64(e.Position.Y), math.Exp(float64(e.Delta.Y)*math.Log(1.5)/distance), false)
	}
}

// designPane is the open screen's design on the canvas, with its zoom
// tools. Pressing an element selects it, and pressing around it clears
// the selection.
func designPane(cx *core.Scope, e *editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fin := ui.FinOf(cx)
	c := newDesignCanvas(cx)
	look := core.NewMemo(cx, func() string {
		s, _ := e.cur().openScreen()
		return s.ID + "\n" + s.Device + "\n" + flowfile.FormatUI(s.UI)
	})
	// Another screen shows fitted.
	screen := core.NewMemo(cx, func() string { s, _ := e.cur().openScreen(); return s.ID })
	cx.Effect(func() {
		screen.Get()
		cx.Post(func() { c.fit(false) })
	})
	sel := selection{
		selected: func() string { return e.cur().element.Get() },
		choose: func(path string) {
			d := e.shown()
			d.element.Set(path)
		},
	}
	picture := core.Dynamic(func(cx *core.Scope) core.Node {
		look.Get()
		s := core.Untracked(cx, func() flowfile.Screen { s, _ := e.cur().openScreen(); return s })
		if len(s.UI) == 0 {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 10}, Children: []core.Node{
				ui.Icon(lucide.PanelsTopLeft, 28, colors.Muted),
				core.Text{Content: tr.DesignEmpty(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(320)},
					StyleFunc: ui.Colored(colors.Muted)}}}
		}
		return core.View{TestID: "design", Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0),
			TransformOrigin: core.ObjectPosition{X: core.PositionAxis{Offset: core.Pt(0)}, Y: core.PositionAxis{Offset: core.Pt(0)}},
			Animated:        &core.AnimatedStyle{TranslateX: c.x, TranslateY: c.y, Scale: c.zoom}},
			OnLayout: c.measured,
			Children: []core.Node{screenView(cx, s, sel)}}
	})
	tool := func(icon func() *core.SVG, label string, press func(), testID string) core.Node {
		return ui.IconButton{Icon: icon, Label: label, OnPress: press, TestID: testID}
	}
	toolbar := core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(12), Right: core.Pt(12), Direction: core.Row, Gap: 2,
		Padding: core.All(4), CornerRadius: ui.RadiusControl + 4, BorderWidth: 1},
		StyleFunc: func() core.Style {
			return fin.Raised().Over(core.Style{Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			tool(lucide.ZoomOut, tr.ZoomOut(), func() { c.zoomBy(1 / designZoomStep) }, "design-zoom-out"),
			tool(lucide.ZoomIn, tr.ZoomIn(), func() { c.zoomBy(designZoomStep) }, "design-zoom-in"),
			tool(lucide.Scan, tr.Fit(), func() { c.fit(true) }, "design-fit"),
		}}
	return core.View{Style: ui.FillRest.Merge(core.Style{Overflow: core.Hidden}), StyleFunc: ui.Filled(colors.Inset), NoFocus: true,
		OnLayout: c.resize,
		OnPress:  func(core.PressEvent) { sel.choose("") },
		OnWheel:  c.wheel,
		OnPan: func(ev core.PanEvent) {
			switch ev.Phase {
			case core.GestureBegan:
				c.moved = true
				c.pan = [2]float64{c.cam.x, c.cam.y}
			case core.GestureChanged:
				c.move(c.pan[0]+float64(ev.Translation.X), c.pan[1]+float64(ev.Translation.Y), c.cam.zoom, false)
			}
		},
		OnPinch: func(ev core.GestureEvent) {
			switch ev.Phase {
			case core.GestureBegan:
				c.pinchFrom = 1
			case core.GestureChanged:
				if c.pinchFrom > 0 && ev.Value > 0 {
					c.zoomAt(float64(ev.Position.X), float64(ev.Position.Y), float64(ev.Value)/c.pinchFrom, false)
					c.pinchFrom = float64(ev.Value)
				}
			}
		},
		Children: []core.Node{picture, toolbar}}
}
