package main

import (
	"strconv"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/locales"
)

// titleBarHeight is the height of the window's own title bar, in which
// the system centers its window controls.
const titleBarHeight core.Points = 40

// titleBar is the window's title bar over the documents: a tab for each,
// as Figma's, and a button that makes another. Its background drags the
// window, and it keeps clear of the window controls.
func titleBar(cx *core.Scope, e *editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	tabs := core.Each(e.tabs.Get, func(d *document) int { return d.id }, func(cx *core.Scope, d func() *document, _ func() int) core.Node {
		return tabView(cx, e, core.Untracked(cx, d))
	})
	return core.View{TestID: "title-bar", Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOn},
		StyleFunc: func() core.Style {
			bar := desktop.TitleBar(cx)
			return core.Style{Direction: core.Row, AlignItems: core.AlignStretch, FlexShrink: 0,
				Height: core.Pt(max(titleBarHeight, bar.Top)), Padding: core.Edges{Left: bar.Left + 4},
				Background: ui.Chrome(cx), Borders: core.Edges{Bottom: 1}, BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignStretch, FlexShrink: 1, MinWidth: core.Pt(0)},
				Children: []core.Node{tabs}},
			core.View{Style: core.Style{JustifyContent: core.JustifyCenter, Padding: core.Symmetric(0, 6)}, Children: []core.Node{
				ui.IconButton{Icon: lucide.Plus, Label: tr.NewFlow(), Size: 26, OnPress: e.newTab, TestID: "new"},
			}},
		}}
}

// tabView is d's tab: a rectangle the bar's height above its line, the
// shown one in the content's color, with its file's name and a button
// that closes it, which a dot of unsaved changes takes the place of
// while the tab is neither shown nor under the pointer.
func tabView(cx *core.Scope, e *editor, d *document) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	id := strconv.Itoa(d.id)
	active := func() bool { return e.active.Get() == d }
	hovered := core.NewSignal(cx, false)
	closable := func() bool { return active() || hovered.Get() }
	return core.View{Role: core.RoleTab, LabelFunc: d.name, Selected: active, NoFocus: true, TestID: "tab-" + id,
		OnPress:      func(core.PressEvent) { e.activate(d) },
		OnMouseEnter: func(core.MouseEvent) { hovered.Set(true) },
		OnMouseLeave: func(core.MouseEvent) { hovered.Set(false) },
		Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, Width: core.Pt(180), MinWidth: core.Pt(96), FlexShrink: 1,
			Padding: core.Edges{Left: 14, Right: 6}, Borders: core.Edges{Left: 1, Right: 1}},
		StyleFunc: func() core.Style {
			if active() {
				return core.Style{Background: colors.Background(), BorderColors: core.EdgeColors{Left: colors.Line(), Right: colors.Line()}}
			}
			// A clear border keeps the tab the shown one's size.
			clear := core.RGBA(0xffffff, 0)
			return core.Style{BorderColors: core.EdgeColors{Left: clear, Right: clear}}
		},
		Children: []core.Node{
			core.Text{ContentFunc: d.name, MaxLines: 1, TestID: "tab-name-" + id,
				Style: ui.TypeBody.Merge(core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)}),
				StyleFunc: func() core.Style {
					if active() {
						return core.Style{Color: colors.Text(), FontWeight: ui.Medium}
					}
					return core.Style{Color: colors.Muted()}
				}},
			core.View{Style: core.Style{Width: core.Pt(20), Height: core.Pt(20), FlexShrink: 0, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
				Children: []core.Node{core.Show{When: closable,
					Then: func(*core.Scope) core.Node {
						return ui.IconButton{Icon: lucide.X, Label: tr.CloseTab(), Size: 20, OnPress: func() { e.closeTab(d) }, TestID: "close-tab-" + id}
					},
					Else: func(*core.Scope) core.Node {
						return core.Show{When: d.dirty.Get, Then: func(*core.Scope) core.Node {
							return core.View{Style: core.Style{Width: core.Pt(7), Height: core.Pt(7), CornerRadius: 3.5},
								StyleFunc: ui.Filled(colors.Muted), Label: tr.Edited()}
						}}
					}}}},
		}}
}
