package main

import (
	"slices"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/agent"
	"github.com/oneo-me/nsgo-flow/flowfile"
	"github.com/oneo-me/nsgo-flow/locales"
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

// screenItem is what the list shows of a screen.
type screenItem struct {
	id, title, device string
	flows             int
}

// sidebar lists the screens, the open one selected.
func sidebar(cx *core.Scope, e *editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	items := core.NewMemoFunc(cx, func() []screenItem {
		var items []screenItem
		for _, s := range e.doc.Get().Screens {
			items = append(items, screenItem{s.ID, cmp(s.Title, s.ID), s.Device, len(s.Flows)})
		}
		return items
	}, slices.Equal)
	sections := func() []ui.SidebarSection {
		section := ui.SidebarSection{Title: tr.Screens(),
			Note: core.Show{When: func() bool { return len(items.Get()) == 0 }, Then: func(*core.Scope) core.Node {
				return core.View{Style: core.Style{Padding: core.Symmetric(4, 10)}, Children: []core.Node{
					core.Text{Content: tr.ScreensEmpty(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)}}}
			}}}
		for _, it := range items.Get() {
			row := []core.Node{
				ui.Icon(deviceIcon(it.device), 15, colors.Accent),
				core.Text{Content: it.title, MaxLines: 1, Style: ui.TypeBody.Merge(core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)}),
					StyleFunc: ui.Colored(colors.Text)},
			}
			if it.flows > 0 {
				row = append(row, core.Text{Content: tr.FlowCount(it.flows), MaxLines: 1, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Faint)})
			}
			section.Items = append(section.Items, ui.SidebarItem{ID: it.id, Text: it.title, TestID: "screen-" + it.id,
				Content: core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, FlexGrow: 1, MinWidth: core.Pt(0)}, Children: row}})
		}
		return []ui.SidebarSection{section}
	}
	return core.View{Style: core.Style{Width: core.Pt(sidebarWidth), FlexShrink: 0, Padding: core.Symmetric(8, 0), Borders: core.Edges{Right: 1}},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Surface(), BorderColors: core.EdgeColors{Right: colors.Line()}}
		},
		Children: []core.Node{ui.Sidebar{SectionsFunc: sections, Selected: e.screen.Get, TestID: "screens",
			OnSelect: func(id string) { e.show(agent.Location{Screen: id}) }}}}
}

// designTab is the tab value of the open screen's design; flows' tabs
// are their IDs, which are never empty.
const designTab = ""

// workspacePane is the open screen: a header with the file, a tab for
// its design and one for each of its flows, and the open tab.
func workspacePane(cx *core.Scope, e *editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	title := func() string {
		if t := e.doc.Get().Title; t != "" {
			return t
		}
		return e.name()
	}
	header := core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, Height: core.Pt(44), Padding: core.Symmetric(0, 12),
		Borders: core.Edges{Bottom: 1}},
		StyleFunc: func() core.Style { return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}} },
		Children: []core.Node{
			ui.Icon(lucide.Workflow, 16, colors.Muted),
			core.Text{ContentFunc: title, MaxLines: 1, Style: ui.TypeBody.Merge(core.Style{FontWeight: ui.Semibold, FlexShrink: 1, MinWidth: core.Pt(0)}),
				StyleFunc: ui.Colored(colors.Text), TestID: "doc-title"},
			core.Show{When: e.dirty.Get, Then: func(*core.Scope) core.Node {
				return ui.Text(tr.Edited(), 12, ui.Regular, colors.Muted)
			}},
			ui.Spacer{},
			ui.IconButton{Icon: lucide.FilePlus, Label: tr.NewFlow(), OnPress: e.newFlow, TestID: "new"},
			ui.IconButton{Icon: lucide.FolderOpen, Label: tr.Open(), OnPress: e.open, TestID: "open"},
			ui.IconButton{Icon: lucide.Save, Label: tr.Save(), OnPress: e.save, TestID: "save"},
		}}
	type tab struct{ id, title string }
	tabs := core.NewMemoFunc(cx, func() []tab {
		s, _ := e.openScreen()
		var tabs []tab
		for _, f := range s.Flows {
			tabs = append(tabs, tab{f.ID, cmp(f.Title, f.ID)})
		}
		return tabs
	}, slices.Equal)
	tabBar := core.Dynamic(func(cx *core.Scope) core.Node {
		options := []ui.Option[string]{{Value: designTab, Label: tr.Design(), Icon: lucide.PanelsTopLeft, TestID: "tab-design"}}
		for _, t := range tabs.Get() {
			options = append(options, ui.Option[string]{Value: t.id, Label: t.title, Icon: lucide.Workflow, TestID: "flow-tab-" + t.id})
		}
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignEnd, Padding: core.Edges{Left: 12, Right: 12, Top: 6},
			Borders: core.Edges{Bottom: 1}},
			StyleFunc: func() core.Style { return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}} },
			Children: []core.Node{ui.Tabs[string]{Variant: ui.TabsUnderline, Options: options, Label: tr.Flows(),
				Value: func() string {
					if e.design.Get() {
						return designTab
					}
					return e.flow.Get()
				},
				OnChange: func(id string) {
					e.show(agent.Location{Screen: e.screen.GetUntracked(), Flow: cmp(id, e.flow.GetUntracked()), Design: id == designTab})
				}}}}
	})
	empty := core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(32), Gap: 12},
		Children: []core.Node{
			ui.Icon(lucide.AppWindow, 32, colors.Muted),
			core.Text{Content: tr.DiagramEmpty(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(360)},
				StyleFunc: ui.Colored(colors.Muted)},
		}}
	// The flow pane stays while tabs and screens change, as remounting a
	// chart trips NSGo; the design covers it while its tab is open.
	body := core.View{Style: ui.FillRest, Children: []core.Node{
		flowPane(cx, e),
		core.Show{When: e.design.Get, Then: func(cx *core.Scope) core.Node {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
				StyleFunc: ui.Filled(colors.Inset), Children: []core.Node{designPane(cx, e)}}
		}},
	}}
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		header,
		core.Show{When: func() bool { return len(e.doc.Get().Screens) == 0 },
			Then: func(*core.Scope) core.Node { return empty },
			Else: func(*core.Scope) core.Node {
				return core.View{Style: ui.FillRest, Children: []core.Node{tabBar, body}}
			}},
	}}
}

// designPane is the open screen's design on its device, as wide as the
// pane lets it be, up to the device's width.
func designPane(cx *core.Scope, e *editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	look := core.NewMemo(cx, func() string {
		s, _ := e.openScreen()
		return s.ID + "\n" + s.Device + "\n" + flowfile.FormatUI(s.UI)
	})
	width := core.NewSignal(cx, core.Points(0))
	picture := core.Dynamic(func(cx *core.Scope) core.Node {
		look.Get()
		s := core.Untracked(cx, func() flowfile.Screen { s, _ := e.openScreen(); return s })
		if len(s.UI) == 0 {
			return core.View{Style: core.Style{Padding: core.All(48), AlignItems: core.AlignCenter, Gap: 10}, Children: []core.Node{
				ui.Icon(lucide.PanelsTopLeft, 28, colors.Muted),
				core.Text{Content: tr.DesignEmpty(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(320)},
					StyleFunc: ui.Colored(colors.Muted)}}}
		}
		return core.View{TestID: "design", Children: []core.Node{screenView(cx, s, func() float32 { return float32(width.Get()) })}}
	})
	return ui.ScrollView{Style: ui.FillRest, EdgeColor: colors.Inset, ContentStyle: core.Style{Padding: core.All(24)},
		Children: []core.Node{core.View{Style: core.Style{AlignItems: core.AlignCenter},
			OnLayout: func(b core.Bounds) { width.Set(b.Size.Width) }, Children: []core.Node{picture}}}}
}
