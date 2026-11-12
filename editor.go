package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
	uiflow "github.com/nsgo-dev/nsgo/ui/flow"

	"github.com/oneo-me/nsgo-flow/agent"
	"github.com/oneo-me/nsgo-flow/flowfile"
	"github.com/oneo-me/nsgo-flow/locales"
)

// historyLimit is how many changes undo keeps.
const historyLimit = 200

// editor is the open flow document and what the views share of it.
type editor struct {
	cx  *core.Scope
	doc *core.Signal[flowfile.Project]
	// path is the file the document came from or was saved to, "" for a
	// new one; dirty says it changed since.
	path  *core.Signal[string]
	dirty *core.Signal[bool]
	// screen and flow are the IDs of the open screen and of its open
	// flow, "" for none; selected is the node of the flow the user or the
	// assistant selected. design says the screen's design shows instead
	// of the flow.
	screen, flow, selected *core.Signal[string]
	design                 *core.Signal[bool]
	// undos and redos are documents before and after changes.
	undos, redos *core.Signal[[]flowfile.Project]
	// group is the key of the last change, whose undo later changes with
	// the same key join.
	group  string
	camera *uiflow.State
	toasts *ui.Toaster
	// opened runs when another document replaces the shown one.
	opened func()
}

func newEditor(cx *core.Scope, toasts *ui.Toaster) *editor {
	never := func(a, b flowfile.Project) bool { return false }
	stacks := func(a, b []flowfile.Project) bool { return false }
	e := &editor{cx: cx, doc: core.NewSignalFunc(cx, flowfile.Project{}, never),
		path: core.NewSignal(cx, ""), dirty: core.NewSignal(cx, false),
		screen: core.NewSignal(cx, ""), flow: core.NewSignal(cx, ""), selected: core.NewSignal(cx, ""), design: core.NewSignal(cx, false),
		undos: core.NewSignalFunc(cx, []flowfile.Project(nil), stacks), redos: core.NewSignalFunc(cx, []flowfile.Project(nil), stacks),
		camera: &uiflow.State{}, toasts: toasts}
	// Another flow shows fitted, once the batch that opened it is over.
	cx.Effect(func() {
		e.flow.Get()
		cx.Post(e.camera.Fit)
	})
	return e
}

// name is the document's file name, or Untitled.
func (e *editor) name() string {
	if p := e.path.Get(); p != "" {
		return filepath.Base(p)
	}
	return locales.For(e.cx).Untitled()
}

// openScreen is the open screen, false when none is.
func (e *editor) openScreen() (flowfile.Screen, bool) {
	p := e.doc.Get()
	if i := p.ScreenIndex(e.screen.Get()); i >= 0 {
		return p.Screens[i], true
	}
	return flowfile.Screen{}, false
}

// flowDoc is the flow with id, empty when there is none.
func (e *editor) flowDoc(id string) flowfile.Doc {
	p := e.doc.Get()
	if i, j := p.FlowIndex(id); i >= 0 {
		return p.Screens[i].Flows[j].Doc
	}
	return flowfile.Doc{}
}

// at is where the user is looking.
func (e *editor) at() agent.Location {
	return agent.Location{Screen: e.screen.GetUntracked(), Flow: e.flow.GetUntracked(), Node: e.selected.GetUntracked(),
		Design: e.design.GetUntracked()}
}

// show opens the screen of l on its design or its flow, and selects the
// node, falling back to the first screen when it is gone. A screen
// opened without a flow shows its design, or its first flow while it has
// no design.
func (e *editor) show(l agent.Location) {
	p := e.doc.GetUntracked()
	si := p.ScreenIndex(l.Screen)
	fi, _ := p.FlowIndex(l.Flow)
	chosen := fi >= 0 && (si < 0 || fi == si)
	if chosen {
		si = fi
	} else {
		l.Flow = ""
	}
	if si < 0 && len(p.Screens) > 0 {
		si = 0
	}
	if si < 0 {
		l = agent.Location{}
	} else {
		s := p.Screens[si]
		l.Screen = s.ID
		l.Design = l.Design || !chosen && len(s.UI) > 0
		if l.Flow == "" && len(s.Flows) > 0 {
			l.Flow = s.Flows[0].ID
		}
		l.Design = l.Design || l.Flow == ""
	}
	if l.Node != "" {
		if _, ok := e.flowDoc(l.Flow).Node(l.Node); !ok {
			l.Node = ""
		}
	}
	e.cx.Batch(func() {
		e.screen.Set(l.Screen)
		e.flow.Set(l.Flow)
		e.design.Set(l.Design)
		e.selected.Set(l.Node)
	})
	if l.Node != "" {
		e.cx.Post(func() { e.camera.Reveal(l.Node) })
	}
}

// change replaces the document with an edit of it, which undo reverts.
// Consecutive changes of one nonempty group, such as an assistant's
// turn, undo together.
func (e *editor) change(d flowfile.Project, group string) {
	before := e.doc.GetUntracked()
	joined := group != "" && group == e.group && len(e.undos.GetUntracked()) > 0
	e.group = group
	e.cx.Batch(func() {
		if !joined {
			undos := append(e.undos.GetUntracked(), before)
			if len(undos) > historyLimit {
				undos = undos[len(undos)-historyLimit:]
			}
			e.undos.Set(undos)
		}
		e.redos.Set(nil)
		e.doc.Set(d)
		e.dirty.Set(true)
		e.show(e.at())
	})
}

func (e *editor) undo() { e.step(e.undos, e.redos) }
func (e *editor) redo() { e.step(e.redos, e.undos) }

// step moves the document from one history stack to the other.
func (e *editor) step(from, to *core.Signal[[]flowfile.Project]) {
	list := from.GetUntracked()
	if len(list) == 0 {
		return
	}
	e.group = ""
	e.cx.Batch(func() {
		to.Set(append(to.GetUntracked(), e.doc.GetUntracked()))
		from.Set(list[: len(list)-1 : len(list)-1])
		e.doc.Set(list[len(list)-1])
		e.dirty.Set(true)
		e.show(e.at())
	})
}

// replace shows another document, forgetting the history.
func (e *editor) replace(d flowfile.Project, path string) {
	flow := e.flow.GetUntracked()
	e.cx.Batch(func() {
		e.doc.Set(d)
		e.path.Set(path)
		e.dirty.Set(false)
		e.undos.Set(nil)
		e.redos.Set(nil)
		e.screen.Set("")
		e.flow.Set("")
		e.show(agent.Location{})
	})
	e.group = ""
	if e.opened != nil {
		e.opened()
	}
	if e.flow.GetUntracked() == flow {
		// Another flow fits as it opens.
		e.cx.Post(e.camera.Fit)
	}
}

// keepChanges asks what becomes of the document's unsaved changes before
// it goes, and runs then once they are saved or discarded; canceling, or
// a save that fails or is canceled, keeps the document.
func (e *editor) keepChanges(then func()) {
	if !e.dirty.GetUntracked() {
		then()
		return
	}
	tr := locales.For(e.cx)
	save, discard := tr.Save(), tr.DontSave()
	base.ShowAlertDialog(e.cx, base.AlertDialog{
		Title: tr.SaveChanges(core.Untracked(e.cx, e.name)), Description: tr.ChangesLost(),
		Actions: []core.AlertAction{{Label: save}, {Label: discard, Role: core.AlertDestructive}, {Label: tr.Cancel(), Role: core.AlertCancel}},
	}, func(label string) {
		switch label {
		case save:
			e.saveThen(then)
		case discard:
			then()
		}
	})
}

func (e *editor) newFlow() { e.keepChanges(func() { e.replace(flowfile.Project{}, "") }) }

// open asks for a flow file and shows it.
func (e *editor) open() { e.keepChanges(e.pick) }

func (e *editor) pick() {
	e.cx.PickFiles(core.PickFilesOptions{Types: []string{".flow", ".txt"}}, func(files []core.File, err error) {
		if err == nil && len(files) == 0 {
			return
		}
		if err == nil {
			err = e.load(files[0])
		}
		if err != nil {
			e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(e.cx).OpenFailed(err.Error())})
		}
	})
}

func (e *editor) load(f core.File) error {
	r, err := f.Open()
	if err != nil {
		return err
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		return err
	}
	d, err := flowfile.ParseProject(string(data))
	if err != nil {
		return err
	}
	path := f.Path
	if f.Temporary {
		path = ""
	}
	e.replace(d, path)
	return nil
}

// openPath shows the flow file at path.
func (e *editor) openPath(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	d, err := flowfile.ParseProject(string(data))
	if err != nil {
		return err
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	e.replace(d, path)
	return nil
}

// save writes the document to its file, or asks where.
func (e *editor) save() { e.saveThen(nil) }

// saveThen saves the document and runs done, when set, once it is saved.
func (e *editor) saveThen(done func()) {
	path := e.path.GetUntracked()
	if path == "" {
		e.saveAsThen(done)
		return
	}
	tr := locales.For(e.cx)
	if err := writeFile(path, e.doc.GetUntracked().Format()); err != nil {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
		return
	}
	e.dirty.Set(false)
	e.toasts.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: tr.Saved()})
	if done != nil {
		done()
	}
}

// saveAs asks where to save the document. The dialog copies a file, so
// the document goes to a temporary one first.
func (e *editor) saveAs() { e.saveAsThen(nil) }

// saveAsThen is saveAs, running done, when set, once the file is saved.
func (e *editor) saveAsThen(done func()) {
	tr := locales.For(e.cx)
	fail := func(err error) {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
	}
	dir, err := os.MkdirTemp("", "nsgo-flow")
	if err != nil {
		fail(err)
		return
	}
	name := strings.TrimSuffix(e.name(), ".flow") + ".flow"
	tmp := filepath.Join(dir, name)
	if err := writeFile(tmp, e.doc.GetUntracked().Format()); err != nil {
		os.RemoveAll(dir)
		fail(err)
		return
	}
	e.cx.SaveFile(core.SaveOptions{Name: name, MIME: "text/plain", Path: tmp}, func(saved bool, err error) {
		os.RemoveAll(dir)
		switch {
		case err != nil:
			fail(err)
		case saved:
			e.dirty.Set(false)
			e.toasts.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: tr.Saved()})
			if done != nil {
				done()
			}
		}
	})
}

// writeFile replaces the file at path with text, leaving the old one
// whole if writing fails.
func writeFile(path, text string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".nsgo-flow-*")
	if err != nil {
		return err
	}
	_, err = tmp.WriteString(text)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}
