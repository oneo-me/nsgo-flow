package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
	uiflow "github.com/nsgo-dev/nsgo/ui/flow"

	"github.com/oneo-me/nsgo-scenra/agent"
	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/locales"
)

// historyLimit is how many changes undo keeps.
const historyLimit = 200

// The storage keys of the files open in tabs, a path a line, and of the
// one shown, which the next start opens again.
const (
	keyOpenFiles  = "editor.openFiles"
	keyActiveFile = "editor.activeFile"
)

// document is a flow document open in a tab: its content and file, its
// history, where the user looks in it and the conversation about it.
type document struct {
	e *editor
	// id keys the document's tab.
	id  int
	doc *core.Signal[flowfile.Project]
	// path is the file the document came from or was saved to, which
	// saving writes again, "" for a new one; dirty says it changed since.
	path  *core.Signal[string]
	dirty *core.Signal[bool]
	// screen and flow are the IDs of the open screen and of its open
	// flow, "" for none; selected is the node of the flow the user or the
	// assistant selected, and element the path of the design's selected
	// element. design says the screen's design shows instead of the flow.
	screen, flow, selected, element *core.Signal[string]
	design                          *core.Signal[bool]
	// undos and redos are documents before and after changes.
	undos, redos *core.Signal[[]flowfile.Project]
	// group is the key of the last change, whose undo later changes with
	// the same key join.
	group string
	chat  *conversation
}

// editor is the window's open documents, a tab each, and the one shown.
// Every document has a file: a new one is saved before it opens.
type editor struct {
	cx     *core.Scope
	tabs   *core.Signal[[]*document]
	active *core.Signal[*document]
	// blank stands for the document shown while none is open.
	blank  *document
	camera *uiflow.State
	toasts *ui.Toaster
	ids    int
}

func newEditor(cx *core.Scope, toasts *ui.Toaster) *editor {
	e := &editor{cx: cx, tabs: core.NewSignalFunc(cx, []*document(nil), func(a, b []*document) bool { return false }),
		active: core.NewSignal[*document](cx, nil), camera: &uiflow.State{}, toasts: toasts}
	e.blank = e.newDocument(flowfile.Project{}, "")
	e.restore()
	// Another flow shows fitted, once the batch that opened it is over.
	cx.Effect(func() {
		e.cur().flow.Get()
		cx.Post(e.camera.Fit)
	})
	// The files open in tabs open again at the next start.
	cx.Effect(func() {
		var paths []string
		for _, d := range e.tabs.Get() {
			if p := d.path.Get(); p != "" {
				paths = append(paths, p)
			}
		}
		active := e.cur().path.Get()
		core.Untracked(cx, func() bool {
			st := cx.Storage()
			st.SetItem(keyOpenFiles, strings.Join(paths, "\n"))
			st.SetItem(keyActiveFile, active)
			return true
		})
	})
	return e
}

// restore opens the files open in tabs when the app last quit, leaving
// out those that are gone.
func (e *editor) restore() {
	st := e.cx.Storage()
	list, _ := st.GetItem(keyOpenFiles)
	active, _ := st.GetItem(keyActiveFile)
	var shown *document
	for _, path := range strings.Split(list, "\n") {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		p, err := flowfile.ParseProject(string(data))
		if err != nil {
			continue
		}
		d := e.newDocument(p, path)
		e.tabs.Set(append(e.tabs.GetUntracked(), d))
		if shown == nil || path == active {
			shown = d
		}
	}
	if shown != nil {
		e.active.Set(shown)
	}
}

// newDocument is a document of p, from the file at path, showing its
// first screen.
func (e *editor) newDocument(p flowfile.Project, path string) *document {
	cx := e.cx
	never := func(a, b flowfile.Project) bool { return false }
	stacks := func(a, b []flowfile.Project) bool { return false }
	e.ids++
	d := &document{e: e, id: e.ids, doc: core.NewSignalFunc(cx, p, never),
		path: core.NewSignal(cx, path), dirty: core.NewSignal(cx, false),
		screen: core.NewSignal(cx, ""), flow: core.NewSignal(cx, ""), selected: core.NewSignal(cx, ""), element: core.NewSignal(cx, ""),
		design: core.NewSignal(cx, false),
		undos:  core.NewSignalFunc(cx, []flowfile.Project(nil), stacks), redos: core.NewSignalFunc(cx, []flowfile.Project(nil), stacks),
		chat: newConversation(cx)}
	d.show(agent.Location{})
	return d
}

// cur is the document shown, blank while none is open, which a caller
// reading it follows.
func (e *editor) cur() *document {
	if d := e.active.Get(); d != nil {
		return d
	}
	return e.blank
}

// shown is the document shown, for actions.
func (e *editor) shown() *document { return core.Untracked(e.cx, e.cur) }

// empty says no document is open.
func (e *editor) empty() bool { return len(e.tabs.Get()) == 0 }

// activate shows d's tab.
func (e *editor) activate(d *document) {
	if e.active.GetUntracked() == d {
		return
	}
	e.active.Set(d)
	e.cx.Post(e.camera.Fit)
}

// newFile asks where to save a new flow file and opens it in a tab,
// then runs then, when set, on it.
func (e *editor) newFile(then func(d *document)) {
	tr := locales.For(e.cx)
	d := e.newDocument(flowfile.Project{}, "")
	dir, err := os.MkdirTemp("", "scenra")
	if err != nil {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
		return
	}
	name := tr.Untitled() + ".scenra"
	tmp := filepath.Join(dir, name)
	if err := writeFile(tmp, ""); err != nil {
		os.RemoveAll(dir)
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
		return
	}
	e.cx.SaveFile(core.SaveOptions{Name: name, MIME: "text/plain", Path: tmp}, func(saved bool, path string, err error) {
		os.RemoveAll(dir)
		switch {
		case err != nil:
			e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
		case saved && path != "":
			d.path.Set(path)
			e.cx.Batch(func() {
				e.tabs.Set(append(slices.Clone(e.tabs.GetUntracked()), d))
				e.activate(d)
			})
			if then != nil {
				then(d)
			}
		}
	})
}

// newTab opens a new flow file.
func (e *editor) newTab() { e.newFile(nil) }

// closeTab closes d's tab once its changes are saved or discarded,
// leaving an empty document when it was the last.
func (e *editor) closeTab(d *document) {
	e.keepChanges(d, func() {
		d.chat.stop()
		tabs := slices.Clone(e.tabs.GetUntracked())
		i := slices.Index(tabs, d)
		if i < 0 {
			return
		}
		tabs = slices.Delete(tabs, i, i+1)
		e.cx.Batch(func() {
			e.tabs.Set(tabs)
			switch {
			case len(tabs) == 0:
				e.active.Set(nil)
			case e.active.GetUntracked() == d:
				e.activate(tabs[min(i, len(tabs)-1)])
			}
		})
	})
}

// openDoc opens p from the file at path in a tab: the file's tab when it
// is open already, else a new one.
func (e *editor) openDoc(p flowfile.Project, path string) {
	for _, d := range e.tabs.GetUntracked() {
		if path != "" && d.path.GetUntracked() == path {
			e.activate(d)
			return
		}
	}
	d := e.newDocument(p, path)
	e.cx.Batch(func() {
		e.tabs.Set(append(slices.Clone(e.tabs.GetUntracked()), d))
		e.activate(d)
	})
}

// name is the document's file name, or Untitled.
func (d *document) name() string {
	if p := d.path.Get(); p != "" {
		return filepath.Base(p)
	}
	return locales.For(d.e.cx).Untitled()
}

// openScreen is the open screen, false when none is.
func (d *document) openScreen() (flowfile.Screen, bool) {
	p := d.doc.Get()
	if i := p.ScreenIndex(d.screen.Get()); i >= 0 {
		return p.Screens[i], true
	}
	return flowfile.Screen{}, false
}

// flowDoc is the flow with id, empty when there is none.
func (d *document) flowDoc(id string) flowfile.Doc {
	p := d.doc.Get()
	if i, j := p.FlowIndex(id); i >= 0 {
		return p.Screens[i].Flows[j].Doc
	}
	return flowfile.Doc{}
}

// at is where the user is looking.
func (d *document) at() agent.Location {
	return agent.Location{Screen: d.screen.GetUntracked(), Flow: d.flow.GetUntracked(), Node: d.selected.GetUntracked(),
		Design: d.design.GetUntracked(), Element: d.element.GetUntracked()}
}

// show opens the screen of l on its design or its flow, and selects the
// node, falling back to the first screen when it is gone. A screen
// opened without a flow shows its design, or its first flow while it has
// no design.
func (d *document) show(l agent.Location) {
	p := d.doc.GetUntracked()
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
		if _, ok := d.flowDoc(l.Flow).Node(l.Node); !ok {
			l.Node = ""
		}
	}
	if l.Element != "" {
		if i := p.ScreenIndex(l.Screen); i < 0 || !l.Design {
			l.Element = ""
		} else if _, ok := flowfile.ElementAt(p.Screens[i].UI, l.Element); !ok {
			l.Element = ""
		}
	}
	d.e.cx.Batch(func() {
		d.screen.Set(l.Screen)
		d.flow.Set(l.Flow)
		d.design.Set(l.Design)
		d.selected.Set(l.Node)
		d.element.Set(l.Element)
	})
	if l.Node != "" && d.e.shown() == d {
		d.e.cx.Post(func() { d.e.camera.Reveal(l.Node) })
	}
}

// change replaces the document with an edit of it, which undo reverts.
// Consecutive changes of one nonempty group, such as an assistant's
// turn, undo together.
func (d *document) change(p flowfile.Project, group string) {
	before := d.doc.GetUntracked()
	joined := group != "" && group == d.group && len(d.undos.GetUntracked()) > 0
	d.group = group
	d.e.cx.Batch(func() {
		if !joined {
			undos := append(d.undos.GetUntracked(), before)
			if len(undos) > historyLimit {
				undos = undos[len(undos)-historyLimit:]
			}
			d.undos.Set(undos)
		}
		d.redos.Set(nil)
		d.doc.Set(p)
		d.dirty.Set(true)
		d.show(d.at())
	})
}

// setComment sets the comment on the element at path of the open
// screen, or removes it when text is "", as a change undo reverts.
func (d *document) setComment(path, text string) {
	p := d.doc.GetUntracked()
	i := p.ScreenIndex(d.screen.GetUntracked())
	if i < 0 {
		return
	}
	els, err := flowfile.SetComment(p.Screens[i].UI, path, text)
	if err != nil {
		return
	}
	p.Screens = slices.Clone(p.Screens)
	p.Screens[i].UI = els
	d.change(p, "")
}

func (d *document) undo() { d.step(d.undos, d.redos) }
func (d *document) redo() { d.step(d.redos, d.undos) }

// step moves the document from one history stack to the other.
func (d *document) step(from, to *core.Signal[[]flowfile.Project]) {
	list := from.GetUntracked()
	if len(list) == 0 {
		return
	}
	d.group = ""
	d.e.cx.Batch(func() {
		to.Set(append(to.GetUntracked(), d.doc.GetUntracked()))
		from.Set(list[: len(list)-1 : len(list)-1])
		d.doc.Set(list[len(list)-1])
		d.dirty.Set(true)
		d.show(d.at())
	})
}

func (e *editor) undo() { e.shown().undo() }
func (e *editor) redo() { e.shown().redo() }

// closeShown closes the tab shown.
func (e *editor) closeShown() {
	if !e.empty() {
		e.closeTab(e.shown())
	}
}

// keepChanges asks what becomes of d's unsaved changes before it goes,
// showing its tab, and runs then once they are saved or discarded;
// canceling, or a save that fails or is canceled, keeps the document.
func (e *editor) keepChanges(d *document, then func()) {
	if !d.dirty.GetUntracked() {
		then()
		return
	}
	e.activate(d)
	tr := locales.For(e.cx)
	save, discard := tr.Save(), tr.DontSave()
	base.ShowAlertDialog(e.cx, base.AlertDialog{
		Title: tr.SaveChanges(core.Untracked(e.cx, d.name)), Description: tr.ChangesLost(),
		Actions: []core.AlertAction{{Label: save}, {Label: discard, Role: core.AlertDestructive}, {Label: tr.Cancel(), Role: core.AlertCancel}},
	}, func(label string) {
		switch label {
		case save:
			e.saveThen(d, then)
		case discard:
			then()
		}
	})
}

// keepAllChanges asks about each document's unsaved changes in turn, and
// runs then once every one is saved or discarded.
func (e *editor) keepAllChanges(then func()) { e.keepChangesFrom(0, then) }

func (e *editor) keepChangesFrom(i int, then func()) {
	tabs := e.tabs.GetUntracked()
	for ; i < len(tabs); i++ {
		if d := tabs[i]; d.dirty.GetUntracked() {
			e.keepChanges(d, func() { e.keepChangesFrom(i+1, then) })
			return
		}
	}
	then()
}

// open asks for flow files and shows each in a tab.
func (e *editor) open() {
	e.cx.PickFiles(core.PickFilesOptions{Multiple: true, Types: []string{".scenra", ".txt"}}, func(files []core.File, err error) {
		for _, f := range files {
			if err != nil {
				break
			}
			err = e.load(f)
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
	p, err := flowfile.ParseProject(string(data))
	if err != nil {
		return err
	}
	path := f.Path
	if f.Temporary {
		path = ""
	} else if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	e.openDoc(p, path)
	return nil
}

// openPath shows the flow file at path.
func (e *editor) openPath(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	p, err := flowfile.ParseProject(string(data))
	if err != nil {
		return err
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	e.openDoc(p, path)
	return nil
}

// save writes the document shown to its file.
func (e *editor) save() {
	if !e.empty() {
		e.saveThen(e.shown(), nil)
	}
}

// saveAs asks where to save the document shown.
func (e *editor) saveAs() {
	if !e.empty() {
		e.saveAsThen(e.shown(), nil)
	}
}

// saveThen saves d and runs done, when set, once it is saved.
func (e *editor) saveThen(d *document, done func()) {
	path := d.path.GetUntracked()
	if path == "" {
		e.saveAsThen(d, done)
		return
	}
	tr := locales.For(e.cx)
	if err := writeFile(path, d.doc.GetUntracked().Format()); err != nil {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
		return
	}
	d.dirty.Set(false)
	e.toasts.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: tr.Saved()})
	if done != nil {
		done()
	}
}

// saveAsThen asks where to save d, which saves there from then on, and
// runs done, when set, once it is saved. The dialog copies a file, so
// the document goes to a temporary one first.
func (e *editor) saveAsThen(d *document, done func()) {
	tr := locales.For(e.cx)
	fail := func(err error) {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: tr.SaveFailed(err.Error())})
	}
	dir, err := os.MkdirTemp("", "scenra")
	if err != nil {
		fail(err)
		return
	}
	name := strings.TrimSuffix(core.Untracked(e.cx, d.name), ".scenra") + ".scenra"
	tmp := filepath.Join(dir, name)
	if err := writeFile(tmp, d.doc.GetUntracked().Format()); err != nil {
		os.RemoveAll(dir)
		fail(err)
		return
	}
	e.cx.SaveFile(core.SaveOptions{Name: name, MIME: "text/plain", Path: tmp}, func(saved bool, path string, err error) {
		os.RemoveAll(dir)
		switch {
		case err != nil:
			fail(err)
		case saved:
			e.cx.Batch(func() {
				if path != "" {
					d.path.Set(path)
				}
				d.dirty.Set(false)
			})
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
	tmp, err := os.CreateTemp(filepath.Dir(path), ".scenra-*")
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
