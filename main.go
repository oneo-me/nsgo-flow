// Command nsgo-flow is a desktop app designer with an AI assistant: flow
// files open in tabs, each listing its screens' designs and flow charts on
// the left, and on the right a conversation with an OpenAI-compatible
// model that draws them with tools.
package main

import (
	"slices"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/locales"
)

func main() {
	nsgo.Run(Root, nsgo.WindowSize(1440, 900), nsgo.WindowConstraints(core.WindowConstraints{MinWidth: 1000, MinHeight: 480}))
}

// shown is the editor of the window shown last, which tests reach.
var shown *editor

// Root is the app's window: the open documents' tabs in its title bar
// over the entries of the one shown, the entry open and the assistant.
func Root(cx *core.Scope) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	toasts := ui.NewToaster(cx)
	e := newEditor(cx, toasts)
	shown = e
	settingsOpen := core.NewSignal(cx, false)
	a := newAssistant(cx, e, func() { settingsOpen.Set(true) })
	reportUsage(cx, e, a)
	// Flow files opened from the Finder, or named on the command line.
	desktop.OnOpenFiles(cx, func(paths []string) {
		for _, path := range paths {
			if err := e.openPath(path); err != nil {
				toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.OpenFailed(err.Error())})
			}
		}
	})
	// Closing the window with unsaved changes asks first.
	if w := desktop.CurrentWindow(cx); w != nil {
		w.OnClosing(cx, func(c *desktop.WindowClosing) {
			if slices.ContainsFunc(e.tabs.GetUntracked(), func(d *document) bool { return d.dirty.GetUntracked() }) {
				c.Cancel()
				e.keepAllChanges(w.Close)
			}
		})
	}

	if desktop.IsAvailable(cx) {
		desktop.SetTitleBar(cx, desktop.TitleBarHidden, titleBarHeight)
		cx.Effect(func() {
			d := e.cur()
			title := d.name()
			if e.empty() {
				title = locales.For(cx).AppName()
			} else if d.dirty.Get() {
				title += " — " + locales.For(cx).Edited()
			}
			desktop.SetTitle(cx, title)
		})
		cx.Effect(func() { desktop.SetApplicationMenu(cx, menu(cx, e, a, settingsOpen)) })
	} else {
		cx.OnKey("mod+n", e.newTab)
		cx.OnKey("mod+o", e.open)
		cx.OnKey("mod+s", e.save)
		cx.OnKey("mod+shift+s", e.saveAs)
		cx.OnKey("mod+w", e.closeShown)
	}

	chatWidth := core.NewSignal(cx, core.Points(400))
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		ui.ResizablePanelGroup{Style: ui.FillRest, HandleLabel: tr.Assistant(), Panels: []base.ResizablePanel{
			{MinSize: 640, Children: []core.Node{core.View{Style: ui.FillRest, Children: []core.Node{
				titleBar(cx, e),
				core.View{Style: ui.FillRest.Merge(core.Style{Direction: core.Row}), Children: []core.Node{sidebar(cx, e), workspacePane(cx, e, a)}},
			}}}},
			{Size: chatWidth.Get, OnResize: chatWidth.Set, MinSize: 320, MaxSize: 720, Children: []core.Node{assistantPane(cx, a)}},
		}},
		settingsDialog(cx, settingsOpen),
		ui.Toasts{Toaster: toasts, Placement: ui.ToastBottomStart},
	}}
}

// menu is the application menu.
func menu(cx *core.Scope, e *editor, a *assistant, settingsOpen *core.Signal[bool]) []desktop.MenuItem {
	tr := locales.For(cx)
	d := e.cur()
	canUndo, canRedo := len(d.undos.Get()) > 0, len(d.redos.Get()) > 0
	settings := func() { settingsOpen.Set(true) }
	return []desktop.MenuItem{
		{Label: tr.AppName(), Items: []desktop.MenuItem{
			{Role: desktop.MenuAbout}, {Separator: true},
			{Label: tr.Settings(), Shortcut: "mod+,", OnSelect: settings}, {Separator: true},
			{Role: desktop.MenuHide}, {Role: desktop.MenuHideOthers}, {Role: desktop.MenuShowAll}, {Separator: true},
			{Role: desktop.MenuQuit},
		}},
		{Label: tr.FileMenu(), Items: []desktop.MenuItem{
			{Label: tr.NewFlow(), Shortcut: "mod+n", OnSelect: e.newTab},
			{Label: tr.Open(), Shortcut: "mod+o", OnSelect: e.open},
			{Separator: true},
			{Label: tr.Save(), Shortcut: "mod+s", OnSelect: e.save},
			{Label: tr.SaveAs(), Shortcut: "mod+shift+s", OnSelect: e.saveAs},
			{Separator: true},
			{Label: tr.NewChat(), Shortcut: "mod+shift+n", OnSelect: a.clear},
			{Separator: true},
			{Label: tr.CloseTab(), Shortcut: "mod+w", OnSelect: e.closeShown},
		}},
		{Label: tr.EditMenu(), Items: []desktop.MenuItem{
			{Role: desktop.MenuUndo}, {Role: desktop.MenuRedo}, {Separator: true},
			{Role: desktop.MenuCut}, {Role: desktop.MenuCopy}, {Role: desktop.MenuPaste}, {Role: desktop.MenuSelectAll},
			{Separator: true},
			{Label: tr.UndoChange(), Shortcut: "mod+alt+z", Disabled: !canUndo, OnSelect: e.undo},
			{Label: tr.RedoChange(), Shortcut: "mod+alt+shift+z", Disabled: !canRedo, OnSelect: e.redo},
		}},
		{Label: tr.ViewMenu(), Items: []desktop.MenuItem{
			{Label: tr.ZoomIn(), Shortcut: "mod+=", OnSelect: func() { e.camera.ZoomIn() }},
			{Label: tr.ZoomOut(), Shortcut: "mod+-", OnSelect: func() { e.camera.ZoomOut() }},
			{Label: tr.Fit(), Shortcut: "mod+0", OnSelect: func() { e.camera.Fit() }},
			{Separator: true},
			{Label: tr.LeftToRight(), Checkable: true, Checked: d.flowDoc(d.flow.Get()).Direction == "right", Disabled: d.flow.Get() == "", OnSelect: e.toggleDirection},
		}},
	}
}
