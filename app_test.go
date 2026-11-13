package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"

	"github.com/oneo-me/nsgo-flow/flowfile"
)

// TestMain gives the platform the main thread under nsgo test -target.
func TestMain(m *testing.M) { nsgotest.Main(m) }

// model answers each request with the next reply, as server-sent events.
func model(t *testing.T, replies ...string) string {
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if len(replies) == 0 {
			http.Error(w, `{"error":{"message":"no more replies"}}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, replies[0])
		replies = replies[1:]
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func deltas(ds ...string) string {
	var b strings.Builder
	for _, d := range ds {
		fmt.Fprintf(&b, "data: {\"choices\":[{\"delta\":%s}]}\n\n", d)
	}
	return b.String() + "data: [DONE]\n\n"
}

// newFileAt answers the next save dialogs with path, in a new folder.
func newFileAt(t *testing.T, app *nsgotest.App, name string) string {
	path := filepath.Join(t.TempDir(), name)
	app.SaveFile = func(core.SaveRequest) string { return path }
	return path
}

func TestAssistantDrawsOnTheChart(t *testing.T) {
	url := model(t,
		deltas(`{"tool_calls":[{"index":0,"id":"c1","function":{"name":"add_nodes","arguments":"{\"nodes\":[{\"id\":\"order\",\"label\":\"Order placed\",\"kind\":\"start\"},{\"id\":\"ship\",\"label\":\"Ship order\"}]}"}},{"index":1,"id":"c2","function":{"name":"add_edges","arguments":"{\"edges\":[{\"from\":\"order\",\"to\":\"ship\"}]}"}}]}`),
		deltas(`{"content":"I drew **two steps**."}`),
	)
	app, e := startEditor(t, nsgotest.Storage(map[string]string{keyBaseURL: url, keyModel: "test-model"}))
	app.Find("text:Make a flow file, or open one, to design an app or chart a flow.")
	// A message with no file open makes one first.
	newFileAt(t, app, "orders.flow")
	app.Type("label:Message", "Chart an order flow")
	app.Press("label:Send")
	app.WaitFor("flow >> label:Ship order", 5*time.Second)
	app.WaitFor("text:two steps", 5*time.Second)
	app.Find("text:Chart an order flow")
	if got := tabNames(app); !slices.Equal(got, []string{"orders.flow"}) {
		t.Fatalf("tabs %v", got)
	}

	// The change undoes as one step per tool call.
	app.Post(e.undo)
	app.WaitForGone("flow >> label:Order placed", time.Second)
	app.Find("text:Ask the assistant to design an app or draw a flow.")
}

func TestAssistantShowsFailuresAndAsksForAProvider(t *testing.T) {
	app, _ := startEditor(t)
	app.Find("text:Set up an OpenAI-compatible provider to start chatting.")
	app.Type("label:Message", "Hello")
	app.Press("label:Send")
	app.WaitFor("id:settings", time.Second)
	app.Type("id:model", "test-model")
	app.Clear("id:base-url")
	app.Type("id:base-url", model(t))
	app.Press("id:settings-done")
	app.WaitForGone("id:settings", 2*time.Second)
	newFileAt(t, app, "hello.flow")
	app.Press("label:Send")
	app.WaitFor("text:no more replies (HTTP 500)", 5*time.Second)
	app.Find("id:retry")
}

// writeFlow writes a flow file named name in a new folder.
func writeFlow(t *testing.T, name, text string) string {
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// startEditor starts the app and returns its editor.
func startEditor(t *testing.T, opts ...nsgotest.Option) (*nsgotest.App, *editor) {
	var e *editor
	app := nsgotest.Start(t, Root, append([]nsgotest.Option{nsgotest.Size(1280, 800)}, opts...)...)
	app.Post(func() { e = shown })
	app.Settle()
	return app, e
}

// tabNames are the names of the open tabs, in order.
func tabNames(app *nsgotest.App) []string {
	var names []string
	for _, v := range app.Query("id:title-bar >> role:tab") {
		names = append(names, v.Props.Label)
	}
	return names
}

func TestOpensFlowFilesInTabs(t *testing.T) {
	plan := writeFlow(t, "plan.flow", "title \"Plan\"\nnode a \"Research\" note=\"Read the docs\"\na -> b\n")
	shop := writeFlow(t, "shop.flow", "screen home \"Home\"\n  text \"Welcome\"\n")
	app, e := startEditor(t)
	open := func(path string) {
		app.Post(func() {
			if err := e.openPath(path); err != nil {
				t.Error(err)
			}
		})
		app.Settle()
	}
	open(plan)
	app.Find("flow >> label:Research")
	app.Find("flow >> text:Read the docs")
	if got := tabNames(app); !slices.Equal(got, []string{"plan.flow"}) {
		t.Fatalf("tabs %v", got)
	}
	open(shop)
	app.WaitFor("design >> text:Welcome", time.Second)
	// A file open already shows its tab again.
	open(plan)
	app.WaitFor("flow >> label:Research", time.Second)
	if got := tabNames(app); !slices.Equal(got, []string{"plan.flow", "shop.flow"}) {
		t.Fatalf("tabs %v", got)
	}
	// The sidebar lists the entries of the document shown.
	app.Press("id:title-bar >> label:shop.flow")
	app.WaitFor("id:design-home", time.Second)
	if app.Exists("id:flow-main") {
		t.Fatal("the sidebar still lists the other document's flow")
	}

	// The files open again at the next start, the one shown too.
	stored, _ := app.Storage().GetItem(keyOpenFiles)
	again, _ := startEditor(t, nsgotest.Storage(map[string]string{keyOpenFiles: stored, keyActiveFile: shop}))
	if got := tabNames(again); !slices.Equal(got, []string{"plan.flow", "shop.flow"}) {
		t.Fatalf("restored tabs %v", got)
	}
	again.Find("design >> text:Welcome")
}

func TestDesignSelectsWhatThePointerIsOver(t *testing.T) {
	shop := writeFlow(t, "shop.flow", "screen home \"Home\"\n"+
		"  box class=\"flex-1 gap-4 p-6\"\n"+
		"    text \"Welcome\" class=\"text-2xl\"\n"+
		"    box class=\"flex-row gap-2 p-2\"\n"+
		"      icon mail class=\"size-5\"\n"+
		"      text \"Mail\"\n"+
		"    end\n"+
		"  end\n")
	app, e := startEditor(t)
	app.Post(func() {
		if err := e.openPath(shop); err != nil {
			t.Error(err)
		}
	})
	app.WaitFor("design >> text:Mail", time.Second)
	// The design fits the pane scaled down, which the layer over it
	// undoes: the pointer is where the camera drew each element.
	pane, design := app.Find("id:design-pointer").Frame.Size, app.Find("id:design").Frame.Size
	zoom := min(1, (pane.Width-2*designFitMargin)/design.Width, (pane.Height-2*designFitMargin)/design.Height)
	if zoom >= 1 {
		t.Fatalf("the design shows unscaled at %v in %v", design, pane)
	}
	at := func(path string) core.Point {
		v := app.Find("id:element-" + path)
		x, y := v.Frame.Size.Width/2, v.Frame.Size.Height/2
		for ; v.Props.TestID != "design"; v = v.Parent {
			x, y = x+v.Frame.Origin.X, y+v.Frame.Origin.Y
		}
		return core.Point{X: (pane.Width-design.Width*zoom)/2 + x*zoom, Y: (pane.Height-design.Height*zoom)/2 + y*zoom}
	}
	app.HoverAt("id:design-pointer", at("0.1.0"))
	app.WaitFor("id:design-hovered", time.Second)
	app.PressAt("id:design-pointer", at("0.1.0"))
	if got := e.cur().element.GetUntracked(); got != "0.1.0" {
		t.Fatalf("pressing the icon selected %q", got)
	}
	app.WaitFor("id:design-selected", time.Second)
	// The comment card opens beside where the pointer pressed, and typing
	// in it keeps the element selected.
	card, pressed := app.Find("id:comment-card").Frame, at("0.1.0")
	if dx, dy := card.Origin.X-pressed.X, card.Origin.Y-pressed.Y; dx < 0 || dx > 20 || dy < 0 || dy > 20 {
		t.Fatalf("the card is at %v, pressed at %v", card.Origin, pressed)
	}
	app.Type("id:comment-input", "Bigger")
	if got := e.cur().element.GetUntracked(); got != "0.1.0" || !app.Exists("id:comment-card") {
		t.Fatalf("typing in the card selected %q", got)
	}
	app.HoverAt("id:design-pointer", at("0.0"))
	app.PressAt("id:design-pointer", at("0.0"))
	if got := e.cur().element.GetUntracked(); got != "0.0" {
		t.Fatalf("pressing the title selected %q", got)
	}
	app.HoverOut()
	if app.Exists("id:design-hovered") {
		t.Fatal("the outline stayed after the pointer left")
	}
	app.PressAt("id:design-pointer", core.Point{X: 4, Y: 4})
	if got := e.cur().element.GetUntracked(); got != "" {
		t.Fatalf("pressing around the design left %q selected", got)
	}
}

func TestSavesWhereTheFileWasSaved(t *testing.T) {
	app, e := startEditor(t)
	// A new file is saved where the user chooses before it opens.
	saveTo := newFileAt(t, app, "chosen.flow")
	app.Press("id:new-file")
	app.Settle()
	if got := tabNames(app); !slices.Equal(got, []string{"chosen.flow"}) {
		t.Fatalf("tabs after making a file %v", got)
	}
	if _, err := os.Stat(saveTo); err != nil {
		t.Fatal(err)
	}
	retitle := func(title string) {
		app.Post(func() {
			d := e.shown()
			p := d.doc.GetUntracked()
			p.Title = title
			d.change(p, "")
		})
		app.Settle()
	}
	// Saving writes the same file without asking where.
	retitle("Second")
	app.Post(e.save)
	app.Settle()
	if len(app.Saves) != 1 {
		t.Fatalf("asked where to save %d times", len(app.Saves))
	}
	if data, _ := os.ReadFile(saveTo); !strings.Contains(string(data), "Second") {
		t.Fatalf("the file was not saved:\n%s", data)
	}

	// An opened file saves where it came from.
	opened := writeFlow(t, "opened.flow", "title \"Opened\"\n")
	app.Post(func() {
		if err := e.openPath(opened); err != nil {
			t.Error(err)
		}
	})
	app.Settle()
	retitle("Changed")
	app.Post(e.save)
	app.Settle()
	if data, _ := os.ReadFile(opened); len(app.Saves) != 1 || !strings.Contains(string(data), "Changed") {
		t.Fatalf("saves %d, file:\n%s", len(app.Saves), data)
	}

	// Canceling the dialog makes no file.
	app.SaveFile = nil
	app.Post(e.newTab)
	app.Settle()
	if got := tabNames(app); len(got) != 2 {
		t.Fatalf("tabs after canceling %v", got)
	}
}

func TestUnsavedChangesAskBeforeTheyGo(t *testing.T) {
	path := writeFlow(t, "plan.flow", "title \"Plan\"\nnode a \"Research\"\n")
	app, e := startEditor(t, nsgotest.PendingAlerts())
	open := func() {
		app.Post(func() {
			if err := e.openPath(path); err != nil {
				t.Error(err)
			}
		})
		app.Settle()
	}
	retitle := func(title string) {
		app.Post(func() {
			d := e.shown()
			p := d.doc.GetUntracked()
			p.Title = title
			d.change(p, "")
		})
		app.Settle()
	}
	closeTab := func() {
		app.Post(e.closeShown)
		app.Settle()
		if alerts := app.Alerts(); len(alerts) != 1 || !strings.Contains(alerts[0].Title, "plan.flow") {
			t.Fatalf("alerts %+v", alerts)
		}
	}

	// Canceling keeps the tab.
	open()
	retitle("Edited plan")
	closeTab()
	app.Answer(2)
	app.Settle()
	if got := tabNames(app); !slices.Equal(got, []string{"plan.flow"}) {
		t.Fatalf("after canceling, the tabs are %v", got)
	}

	// Saving writes the file before the tab closes.
	closeTab()
	app.Answer(0)
	app.Settle()
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "Edited plan") {
		t.Fatalf("the file was not saved:\n%s", data)
	}
	if got := tabNames(app); len(got) != 0 {
		t.Fatalf("after closing the last tab, the tabs are %v", got)
	}

	// Not saving drops the changes.
	open()
	retitle("Dropped")
	closeTab()
	app.Answer(1)
	app.Settle()
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "Dropped") {
		t.Fatal("not saving saved")
	}

	// A clean document goes without asking.
	app.Post(e.closeShown)
	app.Settle()
	if len(app.Alerts()) != 0 {
		t.Fatalf("a clean document asked: %+v", app.Alerts())
	}
}

func TestSettingsTurnOffTheStatistics(t *testing.T) {
	app := nsgotest.Start(t, Root, nsgotest.Size(1280, 800))
	app.Press("label:Open Settings")
	app.WaitFor("id:settings", time.Second)
	app.Press("id:settings >> label:Send usage statistics")
	if v, _ := app.Storage().GetItem(keyStats); v != "off" {
		t.Fatalf("stats = %q, want off", v)
	}
	app.Press("id:settings >> label:Send usage statistics")
	if _, ok := app.Storage().GetItem(keyStats); ok {
		t.Fatal("stats still stored after turning them back on")
	}
}

// recorder is a fake model that answers with replies in turn and keeps
// the requests' bodies.
func recorder(t *testing.T, replies ...string) (url string, bodies func() []string) {
	var mu sync.Mutex
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		data, _ := io.ReadAll(r.Body)
		got = append(got, string(data))
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, replies[0])
		replies = replies[1:]
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() []string { mu.Lock(); defer mu.Unlock(); return slices.Clone(got) }
}

func TestAssistantKnowsTheFilesAndReadsTheFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc checkout() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "app.flow")
	if err := os.WriteFile(path, []byte("title \"App\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := writeFlow(t, "other.flow", "screen pay \"Pay\"\n")
	url, bodies := recorder(t,
		deltas(`{"tool_calls":[{"index":0,"id":"c1","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}},`+
			`{"index":1,"id":"c2","function":{"name":"get_project","arguments":"{\"file\":\"`+other+`\"}"}}]}`),
		deltas(`{"content":"It has a checkout step."}`),
	)
	app, e := startEditor(t, nsgotest.Storage(map[string]string{keyBaseURL: url, keyModel: "test-model"}))
	app.Post(func() {
		e.openPath(other)
		e.openPath(path)
	})
	app.Settle()
	app.Type("label:Message", "Chart the checkout")
	app.Press("label:Send")
	app.WaitFor("text:It has a checkout step.", 5*time.Second)
	got := bodies()
	if !strings.Contains(got[0], `"name":"grep"`) || !strings.Contains(got[0], "in the folder") || !strings.Contains(got[0], "other.flow") {
		t.Fatalf("the first request tells of no folder or other file:\n%s", got[0])
	}
	if !strings.Contains(got[1], `func checkout() {}`) || !strings.Contains(got[1], `screen pay`) {
		t.Fatalf("the file and the other project did not reach the model:\n%s", got[1])
	}
}

func TestAssistantDesignsScreensWithFlows(t *testing.T) {
	ui := `box class=\\\"flex-1 gap-4 p-6 bg-nope\\\"\\ntext \\\"Continue\\\" class=\\\"font-bold\\\"\\nicon mail class=\\\"size-5\\\"\\nend`
	url, bodies := recorder(t,
		deltas(`{"tool_calls":[`+
			`{"index":0,"id":"c1","function":{"name":"add_screen","arguments":"{\"id\":\"signin\",\"title\":\"Sign in\",\"ui\":\"`+ui+`\"}"}},`+
			`{"index":1,"id":"c2","function":{"name":"add_flow","arguments":"{\"id\":\"login\",\"title\":\"Log in\"}"}},`+
			`{"index":2,"id":"c3","function":{"name":"add_nodes","arguments":"{\"nodes\":[{\"id\":\"tap\",\"label\":\"Tap Continue\",\"kind\":\"start\"}]}"}},`+
			`{"index":3,"id":"c4","function":{"name":"add_flow","arguments":"{\"id\":\"reset\",\"title\":\"Reset password\"}"}},`+
			`{"index":4,"id":"c5","function":{"name":"add_nodes","arguments":"{\"nodes\":[{\"id\":\"mail\",\"label\":\"Send reset mail\"}]}"}}]}`),
		deltas(`{"content":"I drew the sign-in screen."}`),
		deltas(`{"content":"Made it bold."}`),
	)
	app, _ := startEditor(t, nsgotest.Size(1440, 900), nsgotest.Storage(map[string]string{keyBaseURL: url, keyModel: "test-model"}))
	newFileAt(t, app, "signin.flow")
	app.Type("label:Message", "Design a sign-in screen")
	app.Press("label:Send")
	app.WaitFor("text:I drew the sign-in screen.", 5*time.Second)
	app.Find("id:screen-signin")
	// The last flow drawn is open; the sidebar opens the other and the
	// design.
	app.Find("flow >> label:Send reset mail")
	app.Press("id:flow-login")
	app.WaitFor("flow >> label:Tap Continue", time.Second)
	app.Press("id:design-signin")
	app.WaitFor("design >> text:Continue", time.Second)
	app.Find("id:design-fit")
	// A class that draws nothing goes back to the model.
	if got := bodies()[1]; !strings.Contains(got, "draw nothing: bg-nope") {
		t.Fatalf("the model was not told of the unknown class:\n%s", got)
	}
	// Pressing an element selects it, which the model hears of.
	app.Press("id:element-0.0")
	app.Type("label:Message", "Make it bold")
	app.Press("label:Send")
	app.WaitFor("text:Made it bold.", 5*time.Second)
	if got := bodies()[2]; !strings.Contains(got, "selected element 0.0") {
		t.Fatalf("the model was not told of the selection:\n%s", got)
	}
}

func TestCommentsAskTheAssistant(t *testing.T) {
	url, bodies := recorder(t,
		deltas(`{"tool_calls":[{"index":0,"id":"c1","function":{"name":"set_element","arguments":"{\"path\":\"0.0\",\"ui\":\"text \\\"Hello there\\\" class=\\\"font-bold\\\"\"}"}},`+
			`{"index":1,"id":"c2","function":{"name":"resolve_comment","arguments":"{\"path\":\"0.0\"}"}}]}`),
		deltas(`{"content":"Made it friendlier."}`),
	)
	path := writeFlow(t, "shop.flow", "screen home \"Home\"\n  box class=\"flex-1 gap-4 p-6\"\n    text \"Welcome\"\n    text \"Mail\" comment=\"Smaller\"\n  end\n")
	app, e := startEditor(t, nsgotest.Storage(map[string]string{keyBaseURL: url, keyModel: "test-model"}))
	app.Post(func() {
		if err := e.openPath(path); err != nil {
			t.Error(err)
		}
	})
	app.WaitFor("design >> text:Welcome", time.Second)
	app.Find("id:comment-pin-0.1")
	app.Find("id:resolve-comments")
	if app.Exists("id:comment-card") {
		t.Fatal("a comment card shows with nothing selected")
	}

	// Commenting on the selected element saves the comment in the file.
	app.Press("id:element-0.0")
	app.WaitFor("id:comment-card", time.Second)
	app.Type("id:comment-input", "Friendlier")
	app.Press("id:comment-save")
	app.WaitFor("id:comment-pin-0.0", time.Second)
	if got := e.shown().doc.GetUntracked().Format(); !strings.Contains(got, `text "Welcome" comment="Friendlier"`) {
		t.Fatalf("the comment was not saved:\n%s", got)
	}
	app.Press("id:comment-remove")
	app.WaitForGone("id:comment-pin-0.0", time.Second)
	app.Type("id:comment-input", "Friendlier")

	// Asking the assistant sends it the comment, which it resolves.
	app.Press("id:comment-ask")
	app.WaitFor("text:Made it friendlier.", 5*time.Second)
	if got := bodies()[0]; !strings.Contains(got, "Resolve my comment on element 0.0: Friendlier") || !strings.Contains(got, `element 0.0: \"Friendlier\"`) {
		t.Fatalf("the model was not told of the comment:\n%s", got)
	}
	got := e.shown().doc.GetUntracked().Format()
	if !strings.Contains(got, `text "Hello there" class="font-bold"`+"\n") || !strings.Contains(got, `comment="Smaller"`) {
		t.Fatalf("the comment was not resolved, or the other one went:\n%s", got)
	}
	app.WaitForGone("id:comment-pin-0.0", time.Second)
	if app.Exists("id:comment-remove") {
		t.Fatal("the card still shows the resolved comment")
	}
}

func must(name string, ok bool) string {
	if !ok {
		return ""
	}
	return name
}

func TestEveryIconNameDraws(t *testing.T) {
	for _, name := range append(slices.Clone(flowfile.CommonIcons), "home", "edit", "alert-circle", "bar-chart-2") {
		if lucideIcons[must(flowfile.IconName(name))] == nil {
			t.Errorf("no icon %q", name)
		}
	}
	for name := range lucideIcons {
		if _, ok := flowfile.IconName(name); !ok {
			t.Errorf("flow files refuse icon %q", name)
		}
	}
}

func TestExamplesDrawEveryClassAndDrawing(t *testing.T) {
	paths, _ := filepath.Glob("examples/*.flow")
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err := flowfile.ParseProject(string(data))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if unknown := unknownClasses(p); len(unknown) > 0 {
			t.Errorf("%s: classes that draw nothing: %v", path, unknown)
		}
		var walk func(els []flowfile.Element)
		walk = func(els []flowfile.Element) {
			for _, el := range els {
				if el.Kind == "svg" {
					if _, err := core.ParseSVG([]byte(el.Text)); err != nil {
						t.Errorf("%s: %v", path, err)
					}
				}
				walk(el.Children)
			}
		}
		for _, s := range p.Screens {
			walk(s.UI)
		}
	}
}
