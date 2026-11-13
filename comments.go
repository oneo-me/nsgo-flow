package main

import (
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/flowfile"
	"github.com/oneo-me/nsgo-flow/locales"
)

// pinSize is a comment pin's diameter on screen, and cardWidth the
// comment card's width.
const (
	pinSize   = 22
	cardWidth = 320
)

// commentPins mark the commented elements of s with a pin on each one's
// top right corner, its size on screen the same at any zoom.
func commentPins(cx *core.Scope, s flowfile.Screen, sel selection, zoom func() float64) []core.Node {
	colors := ui.ColorsOf(cx)
	var out []core.Node
	for _, c := range (flowfile.Project{Screens: []flowfile.Screen{s}}).Comments() {
		out = append(out, core.View{TestID: "comment-pin-" + c.Path, PointerEvents: core.PointerEventsNone,
			StyleFunc: func() core.Style {
				sel.laidOut.Get()
				b, ok := sel.bounds(c.Path)
				if !ok {
					return core.Style{Display: core.DisplayNone}
				}
				return core.Style{Position: core.Absolute, Left: core.Pt(b.Origin.X + b.Size.Width - pinSize/2), Top: core.Pt(b.Origin.Y - pinSize/2),
					Width: core.Pt(pinSize), Height: core.Pt(pinSize), CornerRadius: pinSize / 2, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
					Background: colors.Accent(), Transform: core.Transform{Scale: float32(1 / zoom())}}
			},
			Children: []core.Node{ui.Icon(lucide.MessageSquare, 12, colors.OnAccent)}})
	}
	return out
}

// commentCard edits the comment on the selected element of the design
// shown, and asks the assistant to resolve it, saving it first. place
// says where a card of a size goes in the design pane. The card moves
// by its position, not a transform, as platforms hit-test views without
// their transforms.
func commentCard(cx *core.Scope, e *editor, ask func(prompt string), place func(core.Size) (core.Point, bool)) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fin := ui.FinOf(cx)
	selected := func() (flowfile.Element, string) {
		d := e.cur()
		s, _ := d.openScreen()
		path := d.element.Get()
		el, _ := flowfile.ElementAt(s.UI, path)
		return el, path
	}
	comment := core.NewMemo(cx, func() string { el, _ := selected(); return el.Attr("comment") })
	draft := core.NewSignal(cx, "")
	// The draft starts from the comment of the element selected, and
	// follows it when it changes, such as when the assistant resolves it.
	cx.Effect(func() {
		e.cur().element.Get()
		c := comment.Get()
		core.Untracked(cx, func() bool { draft.Set(c); return true })
	})
	text := func() string { return strings.TrimSpace(draft.Get()) }
	save := func() {
		if t := core.Untracked(cx, text); t != core.Untracked(cx, comment.Get) {
			e.shown().setComment(e.shown().element.GetUntracked(), t)
		}
	}
	size := core.NewSignal(cx, core.Size{})
	return core.View{TestID: "comment-card",
		Style:    core.Style{Position: core.Absolute, Width: core.Pt(cardWidth), Padding: core.All(12), Gap: 8, CornerRadius: ui.RadiusCard, BorderWidth: 1},
		OnLayout: func(b core.Bounds) { size.Set(b.Size) },
		StyleFunc: func() core.Style {
			at, ok := place(size.Get())
			if !ok {
				return core.Style{Display: core.DisplayNone}
			}
			return fin.Raised().Over(core.Style{Left: core.Pt(at.X), Top: core.Pt(at.Y), Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
				ui.Icon(lucide.MessageSquare, 14, colors.Muted),
				ui.Text(tr.Comment(), 13, ui.Semibold, colors.Text),
				ui.Spacer{},
				core.Text{ContentFunc: func() string { el, path := selected(); return el.Kind + " " + path },
					Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
			}},
			ui.Input{Value: draft.Get, OnChange: draft.Set, OnSubmit: func(string) { save() }, Placeholder: tr.CommentPlaceholder(),
				Label: tr.Comment(), TestID: "comment-input"},
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
				core.Show{When: func() bool { return comment.Get() != "" }, Then: func(*core.Scope) core.Node {
					return ui.IconButton{Icon: lucide.Trash, Label: tr.RemoveComment(), TestID: "comment-remove",
						OnPress: func() { e.shown().setComment(e.shown().element.GetUntracked(), "") }}
				}},
				ui.Spacer{},
				ui.Button{Text: tr.SaveComment(), Variant: ui.ButtonOutline, Size: ui.SizeSmall, TestID: "comment-save",
					Disabled: func() bool { return text() == comment.Get() }, OnPress: save},
				ui.Button{Text: tr.AskAssistant(), Icon: lucide.Sparkles, Variant: ui.ButtonPrimary, Size: ui.SizeSmall, TestID: "comment-ask",
					Disabled: func() bool { return text() == "" },
					OnPress: func() {
						save()
						ask(tr.ResolveComment(e.shown().element.GetUntracked(), core.Untracked(cx, text)))
					}},
			}},
		}}
}
