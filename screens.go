package main

import (
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/agent"
	"github.com/oneo-me/nsgo-scenra/locales"
)

const sidebarWidth = 220

// deviceIcon is the icon of a screen's device.
func deviceIcon(device string) func() *core.SVG {
	switch device {
	case "tablet":
		return lucide.Tablet
	case "desktop":
		return lucide.Monitor
	}
	return lucide.Smartphone
}

// The sidebar's entries are a screen's design, "design:" and the
// screen's ID, and its flows, "flow:" and the flow's.
const (
	designEntry = "design:"
	flowEntry   = "flow:"
)

// screenItem is what the sidebar shows of a screen.
type screenItem struct {
	id, title, device string
	flows             []flowItem
}

type flowItem struct{ id, title string }

// sidebar lists the entries of the document shown: under each screen,
// its design and its flows, the open one selected.
func sidebar(cx *core.Scope, e *editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	items := core.NewMemoFunc(cx, func() []screenItem {
		var items []screenItem
		for _, s := range e.cur().doc.Get().Screens {
			it := screenItem{id: s.ID, title: cmp(s.Title, s.ID), device: s.Device}
			for _, f := range s.Flows {
				it.flows = append(it.flows, flowItem{f.ID, cmp(f.Title, f.ID)})
			}
			items = append(items, it)
		}
		return items
	}, func(a, b []screenItem) bool {
		return slices.EqualFunc(a, b, func(x, y screenItem) bool {
			return x.id == y.id && x.title == y.title && x.device == y.device && slices.Equal(x.flows, y.flows)
		})
	})
	sections := func() []ui.SidebarSection {
		if len(items.Get()) == 0 {
			return []ui.SidebarSection{{Title: tr.Screens(), Note: core.View{Style: core.Style{Padding: core.Symmetric(4, 10)}, Children: []core.Node{
				core.Text{Content: tr.ScreensEmpty(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)}}}}}
		}
		var sections []ui.SidebarSection
		for _, it := range items.Get() {
			sec := ui.SidebarSection{Title: it.title, Collapsible: true, TestID: "screen-" + it.id,
				Items: []ui.SidebarItem{{ID: designEntry + it.id, Text: tr.Design(), Icon: deviceIcon(it.device), TestID: "design-" + it.id}}}
			for _, f := range it.flows {
				sec.Items = append(sec.Items, ui.SidebarItem{ID: flowEntry + f.id, Text: f.title, Icon: lucide.Workflow, TestID: "flow-" + f.id})
			}
			sections = append(sections, sec)
		}
		return sections
	}
	selected := func() string {
		d := e.cur()
		if d.design.Get() {
			return designEntry + d.screen.Get()
		}
		return flowEntry + d.flow.Get()
	}
	choose := func(id string) {
		d := e.shown()
		if screen, ok := strings.CutPrefix(id, designEntry); ok {
			d.show(agent.Location{Screen: screen, Design: true})
		} else {
			d.show(agent.Location{Flow: strings.TrimPrefix(id, flowEntry)})
		}
	}
	return core.View{Style: core.Style{Width: core.Pt(sidebarWidth), FlexShrink: 0, Padding: core.Symmetric(8, 0), Borders: core.Edges{Right: 1}},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Surface(), BorderColors: core.EdgeColors{Right: colors.Line()}}
		},
		Children: []core.Node{ui.Sidebar{SectionsFunc: sections, Selected: selected, OnSelect: choose, TestID: "screens"}}}
}

// workspacePane is the entry the sidebar opened: a screen's design, or a
// flow chart; or, while no file is open, a way to make or open one.
func workspacePane(cx *core.Scope, e *editor, a *assistant) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	message := func(icon func() *core.SVG, text string, actions ...core.Node) core.Node {
		return core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(32), Gap: 12},
			Children: []core.Node{
				ui.Icon(icon, 32, colors.Muted),
				core.Text{Content: text, Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(360)},
					StyleFunc: ui.Colored(colors.Muted)},
				core.View{Style: core.Style{Direction: core.Row, Gap: 8, Margin: core.Edges{Top: 4}}, Children: actions},
			}}
	}
	// The flow pane stays while entries and documents change, as
	// remounting a chart trips NSGo; the design covers it while it shows.
	body := core.View{Style: ui.FillRest, Children: []core.Node{
		flowPane(cx, e),
		core.Show{When: func() bool { return e.cur().design.Get() }, Then: func(cx *core.Scope) core.Node {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
				Children: []core.Node{designPane(cx, e, func(prompt string) { a.ask(prompt, func() {}) })}}
		}},
	}}
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		core.Show{When: func() bool { return e.empty() || len(e.cur().doc.Get().Screens) == 0 },
			Then: func(*core.Scope) core.Node {
				return core.Show{When: e.empty,
					Then: func(*core.Scope) core.Node {
						return message(lucide.FileStack, tr.NoFiles(),
							ui.Button{Text: tr.NewFlow(), Icon: lucide.FilePlus, Variant: ui.ButtonPrimary, OnPress: e.newTab, TestID: "new-file"},
							ui.Button{Text: tr.Open(), Icon: lucide.FolderOpen, Variant: ui.ButtonOutline, OnPress: e.open, TestID: "open-file"})
					},
					Else: func(*core.Scope) core.Node { return message(lucide.AppWindow, tr.DiagramEmpty()) }}
			},
			Else: func(*core.Scope) core.Node { return body }},
	}}
}
