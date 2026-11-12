package main

import (
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/flowfile"
)

// deviceSize is the size a screen of device is laid out at.
func deviceSize(device string) (w, h float32) {
	switch device {
	case "tablet":
		return 820, 1080
	case "desktop":
		return 1200, 760
	}
	return 390, 800
}

// selection is which element of a design is selected, by its path,
// and how pressing one selects it.
type selection struct {
	selected func() string
	choose   func(path string)
}

// screenView draws a screen's interface at its device's size: its
// elements stack in the device's frame, which clips what overflows.
func screenView(cx *core.Scope, s flowfile.Screen, sel selection) core.Node {
	colors := ui.ColorsOf(cx)
	w, h := deviceSize(s.Device)
	radius := core.Points(44)
	switch s.Device {
	case "tablet":
		radius = 28
	case "desktop":
		radius = 12
	}
	return core.View{Style: core.Style{Width: core.Pt(core.Points(w)), Height: core.Pt(core.Points(h)), CornerRadius: radius, FlexShrink: 0,
		BorderCurve: core.BorderCurveContinuous, Overflow: core.Hidden, BorderWidth: 1, FontSize: 15},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Background(), Color: colors.Text(), BorderColor: colors.Border()}
		},
		Children: elements(cx, s.UI, nil, sel)}
}

func elements(cx *core.Scope, els []flowfile.Element, parent []int, sel selection) []core.Node {
	out := make([]core.Node, len(els))
	for i, el := range els {
		out[i] = element(cx, el, append(slices.Clone(parent), i), sel)
	}
	return out
}

// element draws the element at path, shaped by its classes, which
// pressing selects and an outline marks while it is selected. An image
// without a picture, or an SVG NSGo cannot read, draws a placeholder.
func element(cx *core.Scope, el flowfile.Element, path []int, sel selection) core.Node {
	colors := ui.ColorsOf(cx)
	class := el.Attr("class")
	id := flowfile.FormatPath(path)
	press := func(core.PressEvent) { sel.choose(id) }
	marked := func() core.Style {
		if sel.selected() != id {
			return core.Style{}
		}
		return core.Style{OutlineWidth: 2, OutlineColor: colors.Accent()}
	}
	testID := "element-" + id
	placeholder := func(icon func() *core.SVG) core.Node {
		return core.View{Class: "h-40 items-center justify-center overflow-hidden bg-muted " + class, Label: el.Text,
			OnPress: press, StyleFunc: marked, TestID: testID,
			Children: []core.Node{core.Icon{SVG: icon(), Class: "size-8 text-muted-foreground"}}}
	}
	switch el.Kind {
	case "box":
		return core.View{Class: class, OnPress: press, StyleFunc: marked, TestID: testID, Children: elements(cx, el.Children, path, sel)}
	case "text":
		return core.Text{Content: el.Text, Class: class, OnPress: press, StyleFunc: marked, TestID: testID}
	case "icon":
		return core.Icon{SVG: iconNamed(el.Text)(), Class: class, OnPress: press, StyleFunc: marked, TestID: testID}
	case "image":
		if src := el.Attr("src"); src != "" {
			return core.Image{Source: core.Src(src), ResizeMode: core.Cover, Class: "h-40 overflow-hidden bg-muted " + class, Label: el.Text,
				OnPress: press, StyleFunc: marked, TestID: testID}
		}
		return placeholder(lucide.Image)
	case "svg":
		doc, err := core.ParseSVG([]byte(el.Text))
		if err != nil {
			return placeholder(lucide.ImageOff)
		}
		return core.SVGView{SVG: doc, Class: class, OnPress: press, StyleFunc: marked, TestID: testID}
	}
	return core.Group{}
}

// classWarning tells the model of the classes of after's interfaces
// that draw nothing, as NSGo knows no such utility, leaving out those
// before had already.
func classWarning(before, after flowfile.Project) string {
	seen := map[string]bool{}
	for _, c := range unknownClasses(before) {
		seen[c] = true
	}
	var fresh []string
	for _, c := range unknownClasses(after) {
		if !seen[c] {
			seen[c] = true
			fresh = append(fresh, c)
		}
	}
	if len(fresh) == 0 {
		return ""
	}
	return "\nwarning: these classes are no Tailwind CSS utility and draw nothing: " + strings.Join(fresh, " ")
}

// unknownClasses are the classes of p's interfaces that draw nothing.
func unknownClasses(p flowfile.Project) []string {
	var out []string
	var walk func(els []flowfile.Element)
	walk = func(els []flowfile.Element) {
		for _, el := range els {
			for _, c := range core.UnknownClasses(el.Attr("class")) {
				if !slices.Contains(out, c) {
					out = append(out, c)
				}
			}
			walk(el.Children)
		}
	}
	for _, s := range p.Screens {
		walk(s.UI)
	}
	return out
}
