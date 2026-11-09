// Command nsgo-flow is a flow chart editor with an AI assistant: the chart
// of a flow file on the left, and on the right a conversation with an
// OpenAI-compatible model that draws on it with tools.
package main

import (
	"os"

	"github.com/oneo-me/nsgo"
	"github.com/oneo-me/nsgo/base"
	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/desktop"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/locales"
)

// startPath is the flow file the command line names.
var startPath string

func main() {
	if len(os.Args) > 1 {
		startPath = os.Args[1]
	}
	nsgo.Run(Root, nsgo.WindowSize(1280, 800), nsgo.WindowConstraints(core.WindowConstraints{MinWidth: 760, MinHeight: 480}))
}

// shown is the editor of the window shown last, which tests reach.
var shown *editor

// Root is the app's window: the diagram beside the assistant.
func Root(cx *core.Scope) core.Node {
	cx.SetTheme(ui.DefaultTheme())
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	toasts := ui.NewToaster(cx)
	e := newEditor(cx, toasts)
	shown = e
	settingsOpen := core.NewSignal(cx, false)
	a := newAssistant(cx, e, func() { settingsOpen.Set(true) })
	// A conversation is about the document it began on.
	e.opened = a.clear
	if startPath != "" {
		if err := e.openPath(startPath); err != nil {
			toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.OpenFailed(err.Error())})
		}
	}

	if desktop.IsAvailable(cx) {
		cx.Effect(func() {
			title := e.name()
			if e.dirty.Get() {
				title += " — " + locales.For(cx).Edited()
			}
			desktop.SetTitle(cx, title)
		})
		cx.Effect(func() { desktop.SetApplicationMenu(cx, menu(cx, e, a, settingsOpen)) })
	} else {
		cx.OnKey("mod+o", e.open)
		cx.OnKey("mod+s", e.save)
	}

	chatWidth := core.NewSignal(cx, core.Points(400))
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		ui.ResizablePanelGroup{Style: ui.FillRest, HandleLabel: tr.Assistant(), Panels: []base.ResizablePanel{
			{MinSize: 320, Children: []core.Node{diagramPane(cx, e)}},
			{Size: chatWidth.Get, OnResize: chatWidth.Set, MinSize: 320, MaxSize: 720, Children: []core.Node{assistantPane(cx, a)}},
		}},
		settingsDialog(cx, settingsOpen),
		ui.Toasts{Toaster: toasts, Placement: ui.ToastBottomStart},
	}}
}

// menu is the application menu.
func menu(cx *core.Scope, e *editor, a *assistant, settingsOpen *core.Signal[bool]) []desktop.MenuItem {
	tr := locales.For(cx)
	canUndo, canRedo := len(e.undos.Get()) > 0, len(e.redos.Get()) > 0
	settings := func() { settingsOpen.Set(true) }
	return []desktop.MenuItem{
		{Label: tr.AppName(), Items: []desktop.MenuItem{
			{Role: desktop.MenuAbout}, {Separator: true},
			{Label: tr.Settings(), Shortcut: "mod+,", OnSelect: settings}, {Separator: true},
			{Role: desktop.MenuHide}, {Role: desktop.MenuHideOthers}, {Role: desktop.MenuShowAll}, {Separator: true},
			{Role: desktop.MenuQuit},
		}},
		{Label: tr.FileMenu(), Items: []desktop.MenuItem{
			{Label: tr.NewFlow(), Shortcut: "mod+n", OnSelect: e.newFlow},
			{Label: tr.Open(), Shortcut: "mod+o", OnSelect: e.open},
			{Separator: true},
			{Label: tr.Save(), Shortcut: "mod+s", OnSelect: e.save},
			{Label: tr.SaveAs(), Shortcut: "mod+shift+s", OnSelect: e.saveAs},
			{Separator: true},
			{Label: tr.NewChat(), Shortcut: "mod+shift+n", OnSelect: a.clear},
			{Separator: true},
			{Role: desktop.MenuClose},
		}},
		{Label: tr.EditMenu(), Items: []desktop.MenuItem{
			{Role: desktop.MenuUndo}, {Role: desktop.MenuRedo}, {Separator: true},
			{Role: desktop.MenuCut}, {Role: desktop.MenuCopy}, {Role: desktop.MenuPaste}, {Role: desktop.MenuSelectAll},
			{Separator: true},
			{Label: tr.UndoChange(), Shortcut: "mod+alt+z", Disabled: !canUndo, OnSelect: e.undo},
			{Label: tr.RedoChange(), Shortcut: "mod+alt+shift+z", Disabled: !canRedo, OnSelect: e.redo},
		}},
		{Label: tr.ViewMenu(), Items: []desktop.MenuItem{
			{Label: tr.ZoomIn(), Shortcut: "mod+=", OnSelect: e.camera.ZoomIn},
			{Label: tr.ZoomOut(), Shortcut: "mod+-", OnSelect: e.camera.ZoomOut},
			{Label: tr.Fit(), Shortcut: "mod+0", OnSelect: e.camera.Fit},
			{Separator: true},
			{Label: tr.LeftToRight(), Checkable: true, Checked: e.doc.Get().Direction == "right", OnSelect: e.toggleDirection},
		}},
	}
}
