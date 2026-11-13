package main

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	uiflow "github.com/nsgo-dev/nsgo/ui/flow"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/locales"
)

// diagramData maps the flow with id, the open one for "", onto the
// diagram's data. Each node's face reads its node from the document, as
// the view keeps a face while its ID stays.
func diagramData(e *editor, flow string) func() uiflow.Data {
	return func() uiflow.Data {
		d := e.cur().flowDoc(cmp(flow, e.cur().flow.Get()))
		var data uiflow.Data
		for _, n := range d.Nodes {
			data.Nodes = append(data.Nodes, uiflow.Node{ID: n.ID, Label: n.Label, Muted: n.Status == "skipped",
				Content: nodeCard{e: e, flow: flow, id: n.ID}})
		}
		for _, l := range d.Edges {
			data.Edges = append(data.Edges, uiflow.Edge{Source: l.From, Target: l.To, Label: l.Label, Dashed: l.Dashed})
		}
		return data
	}
}

// diagramDirection is the way the flow with id, the open one for "",
// points.
func diagramDirection(e *editor, flow string) func() uiflow.Direction {
	return func() uiflow.Direction {
		if e.cur().flowDoc(cmp(flow, e.cur().flow.Get())).Direction == flowfile.Right {
			return uiflow.LeftToRight
		}
		return uiflow.TopToBottom
	}
}

// nodeCard is a node's face: its kind's icon tile, label and note, and
// its status above them.
type nodeCard struct {
	e        *editor
	flow, id string
}

// Render implements core.Node.
func (c nodeCard) Render(cx *core.Scope) core.Node {
	node := core.NewMemo(cx, func() flowfile.Node {
		n, _ := c.e.cur().flowDoc(cmp(c.flow, c.e.cur().flow.Get())).Node(c.id)
		return n
	})
	return core.Dynamic(func(cx *core.Scope) core.Node { return card(cx, node.Get()) })
}

func card(cx *core.Scope, n flowfile.Node) core.Node {
	colors := ui.ColorsOf(cx)
	tile, icon := colors.Accent, lucide.ListChecks
	switch n.Kind {
	case "start":
		tile, icon = colors.Up, lucide.Play
	case "end":
		tile, icon = colors.Danger, lucide.Flag
	case "decision":
		tile, icon = colors.Warn, lucide.Split
	case "data":
		tile, icon = colors.Info, lucide.Database
	case "note":
		tile, icon = colors.Muted, lucide.StickyNote
	}
	card := uiflow.Card{Title: n.Label, Description: n.Note,
		Leading: core.View{Style: core.Style{Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 8, FlexShrink: 0,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
			StyleFunc: func() core.Style { return core.Style{Background: tile().Faded(.16)} },
			Children:  []core.Node{ui.Icon(icon, 16, tile)}}}
	if n.Note != "" {
		// A note may be long: let it wrap to a few lines.
		card.Body = core.Group{
			core.Text{Content: n.Label, MaxLines: 2, Style: ui.TypeBody.Merge(core.Style{FontWeight: ui.Semibold}), StyleFunc: ui.Colored(colors.Text)},
			core.Text{Content: n.Note, MaxLines: 3, Style: core.Style{FontSize: 11, LineHeight: 15}, StyleFunc: ui.Colored(colors.Muted)},
		}
	}
	if n.Status != "" {
		tr := locales.For(cx)
		color, icon, label := colors.Muted, lucide.CircleDashed, tr.StatusTodo()
		switch n.Status {
		case "doing":
			color, icon, label = colors.Accent, lucide.LoaderCircle, tr.StatusDoing()
			card.Variant = uiflow.CardAccent
		case "done":
			color, icon, label = colors.Up, lucide.CircleCheck, tr.StatusDone()
		case "blocked":
			color, icon, label = colors.Danger, lucide.CircleAlert, tr.StatusBlocked()
		case "skipped":
			icon, label = lucide.CircleSlash, tr.StatusSkipped()
		}
		card.Header = core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4}, Children: []core.Node{
			ui.Icon(icon, 12, color), ui.Text(label, 11, ui.Medium, color)}}
	}
	return card
}

// flowPane is the open flow's chart, with the view's tools over it.
func flowPane(cx *core.Scope, e *editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fin := ui.FinOf(cx)
	full := func() bool { d := e.cur(); return len(d.flowDoc(d.flow.Get()).Nodes) > 0 }
	tool := func(icon func() *core.SVG, label string, enabled func() bool, press func(), testID string) core.Node {
		b := ui.IconButton{Icon: icon, Label: label, OnPress: press, TestID: testID}
		if enabled != nil {
			b.Disabled = func() bool { return !enabled() }
		}
		return b
	}
	toolbar := core.View{Style: core.Style{Direction: core.Row, Gap: 2, Padding: core.All(4), CornerRadius: ui.RadiusControl + 4, BorderWidth: 1},
		StyleFunc: func() core.Style {
			return fin.Raised().Over(core.Style{Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			tool(lucide.ArrowRightLeft, tr.LeftToRight(), full, e.toggleDirection, "direction"),
			tool(lucide.ZoomOut, tr.ZoomOut(), full, func() { e.camera.ZoomOut() }, "zoom-out"),
			tool(lucide.ZoomIn, tr.ZoomIn(), full, func() { e.camera.ZoomIn() }, "zoom-in"),
			tool(lucide.Scan, tr.Fit(), full, func() { e.camera.Fit() }, "fit"),
		}}
	// One view follows the open flow: remounting a chart is costly.
	view := uiflow.View{TestID: "flow", State: e.camera, Data: diagramData(e, ""), Direction: diagramDirection(e, ""),
		LabelFunc: func() string {
			d := e.cur()
			f := d.flowDoc(d.flow.Get())
			return tr.DiagramLabel(len(f.Nodes), len(f.Edges))
		},
		EmptyText: tr.FlowEmpty(), Selected: func() string { return e.cur().selected.Get() }, OnSelect: func(id string) { e.shown().selected.Set(id) }, Overlay: toolbar}
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{view}}
}

// toggleDirection turns the open flow between down and right.
func (e *editor) toggleDirection() {
	d := e.shown()
	p := d.doc.GetUntracked().Clone()
	i, j := p.FlowIndex(d.flow.GetUntracked())
	if i < 0 {
		return
	}
	f := &p.Screens[i].Flows[j]
	if f.Direction == flowfile.Right {
		f.Direction = flowfile.Down
	} else {
		f.Direction = flowfile.Right
	}
	d.change(p, "")
	e.cx.Post(func() { e.camera.Fit() })
}
