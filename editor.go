package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/ui"
	uiflow "github.com/oneo-me/nsgo/ui/flow"

	"github.com/oneo-me/nsgo-flow/flowfile"
	"github.com/oneo-me/nsgo-flow/locales"
)

// historyLimit is how many changes undo keeps.
const historyLimit = 200

// editor is the open flow document and what the views share of it.
type editor struct {
	cx  *core.Scope
	doc *core.Signal[flowfile.Doc]
	// path is the file the document came from or was saved to, "" for a
	// new one; dirty says it changed since.
	path  *core.Signal[string]
	dirty *core.Signal[bool]
	// selected is the node the user or the assistant selected.
	selected *core.Signal[string]
	// undos and redos are documents before and after changes.
	undos, redos *core.Signal[[]flowfile.Doc]
	// group is the key of the last change, whose undo later changes with
	// the same key join.
	group  string
	camera *uiflow.State
	toasts *ui.Toaster
	// opened runs when another document replaces the shown one.
	opened func()
}

func newEditor(cx *core.Scope, toasts *ui.Toaster) *editor {
	never := func(a, b flowfile.Doc) bool { return false }
	stacks := func(a, b []flowfile.Doc) bool { return false }
	return &editor{cx: cx, doc: core.NewSignalFunc(cx, flowfile.Doc{}, never),
		path: core.NewSignal(cx, ""), dirty: core.NewSignal(cx, false), selected: core.NewSignal(cx, ""),
		undos: core.NewSignalFunc(cx, []flowfile.Doc(nil), stacks), redos: core.NewSignalFunc(cx, []flowfile.Doc(nil), stacks),
		camera: &uiflow.State{}, toasts: toasts}
}

// name is the document's file name, or Untitled.
func (e *editor) name() string {
	if p := e.path.Get(); p != "" {
		return filepath.Base(p)
	}
	return locales.For(e.cx).Untitled()
}

// change replaces the document with an edit of it, which undo reverts.
// Consecutive changes of one nonempty group, such as an assistant's
// turn, undo together.
func (e *editor) change(d flowfile.Doc, group string) {
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
		if s := e.selected.GetUntracked(); s != "" {
			if _, ok := d.Node(s); !ok {
				e.selected.Set("")
			}
		}
	})
}

func (e *editor) undo() { e.step(e.undos, e.redos) }
func (e *editor) redo() { e.step(e.redos, e.undos) }

// step moves the document from one history stack to the other.
func (e *editor) step(from, to *core.Signal[[]flowfile.Doc]) {
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
	})
}

// replace shows another document, forgetting the history.
func (e *editor) replace(d flowfile.Doc, path string) {
	e.cx.Batch(func() {
		e.doc.Set(d)
		e.path.Set(path)
		e.dirty.Set(false)
		e.selected.Set("")
		e.undos.Set(nil)
		e.redos.Set(nil)
	})
	e.group = ""
	if e.opened != nil {
		e.opened()
	}
	e.cx.Post(e.camera.Fit)
}

func (e *editor) newFlow() { e.replace(flowfile.Doc{}, "") }

// open asks for a flow file and shows it.
func (e *editor) open() {
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
	d, err := flowfile.Parse(string(data))
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
	d, err := flowfile.Parse(string(data))
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
func (e *editor) save() {
	path := e.path.GetUntracked()
	if path == "" {
		e.saveAs()
		return
	}
	tr := locales.For(e.cx)
	if err := writeFile(path, e.doc.GetUntracked().Format()); err != nil {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
		return
	}
	e.dirty.Set(false)
	e.toasts.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: tr.Saved()})
}

// saveAs asks where to save the document. The dialog copies a file, so
// the document goes to a temporary one first.
func (e *editor) saveAs() {
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
