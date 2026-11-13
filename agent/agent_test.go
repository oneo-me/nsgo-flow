package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/oneo-me/nsgo-flow/flowfile"
)

func TestApply(t *testing.T) {
	var p flowfile.Project
	var at Location
	steps := []struct{ tool, args, want string }{
		{ToolAddNodes, `{"nodes":[{"id":"a","label":"Start","kind":"start"},{"id":"b","label":"Work"}]}`, "added 2 nodes"},
		{ToolAddEdges, `{"edges":[{"from":"a","to":"b","label":"go","dashed":true}]}`, "added 1 edges"},
		{ToolUpdateNode, `{"id":"b","rename_to":"work","status":"doing"}`, "updated b"},
		{ToolSetLayout, `{"title":"Plan","direction":"right","project_title":"App"}`, "layout set"},
		{ToolAddScreen, `{"id":"signin","title":"Sign in","ui":"text \"Sign in\"\nbox class=\"flex-row\"\n  icon check class=\"size-4\"\nend"}`, "screen signin has 2 elements and 0 flows"},
		{ToolAddFlow, `{"id":"login","title":"Log in"}`, "flow login is on screen signin"},
		{ToolAddNodes, `{"nodes":[{"id":"tap","label":"Tap Go"}]}`, "added 1 nodes"},
		{ToolSetUI, `{"screen":"main","ui":"text \"Plan\""}`, "screen main has 1 elements"},
	}
	for _, s := range steps {
		r := Apply(p, at, s.tool, s.args)
		if r.Failed || r.Output != s.want {
			t.Fatalf("%s: %q", s.tool, r.Output)
		}
		p = r.Project
		if r.Show.Screen != "" {
			at = r.Show
		}
	}
	want := `title "App"

screen main
  text "Plan"

  flow main "Plan"
    direction right

    node a "Start" kind=start
    node work "Work" status=doing

    a ..> work "go"

screen signin "Sign in"
  text "Sign in"
  box class="flex-row"
    icon "check" class="size-4"
  end

  flow login "Log in"
    node tap "Tap Go"
`
	if got := p.Format(); got != want {
		t.Fatalf("project:\n%s", got)
	}
	if at != (Location{Screen: "main", Design: true}) {
		t.Fatalf("open at %+v", at)
	}
	if r := Apply(p, at, ToolGetProject, ""); r.Output != want || r.Changed {
		t.Fatalf("get_project: %+v", r)
	}
	if r := Apply(p, at, ToolShow, `{"flow":"login","node":"tap"}`); r.Show != (Location{Screen: "signin", Flow: "login", Node: "tap"}) || r.Changed {
		t.Fatalf("show: %+v", r)
	}
	r := Apply(p, at, ToolRemoveNodes, `{"ids":["a","nope"]}`)
	if !r.Failed || len(r.Project.Screens[0].Flows[0].Nodes) != 2 || len(p.Screens[0].Flows[0].Nodes) != 2 {
		t.Fatalf("a failed call changed the project: %+v", r)
	}
	if r := Apply(p, at, ToolSetUI, `{"ui":"widget"}`); !r.Failed || !strings.Contains(r.Output, "unknown element") {
		t.Fatalf("set_ui: %+v", r)
	}
	if r := Apply(p, Location{Screen: "signin"}, ToolAddFlow, `{"id":"login","screen":"main"}`); !r.Failed {
		t.Fatalf("a second flow login: %+v", r)
	}
	r = Apply(p, at, ToolReplaceFlow, `{"flow":"login","text":"x -> y\ny -> z"}`)
	if r.Failed || len(r.Project.Screens[1].Flows[0].Nodes) != 3 || r.Project.Screens[1].Flows[0].Title != "Log in" {
		t.Fatalf("replace: %+v", r)
	}
	r = Apply(p, at, ToolRemoveScreen, `{"id":"main"}`)
	if r.Failed || len(r.Project.Screens) != 1 {
		t.Fatalf("remove_screen: %+v", r)
	}
	for _, tool := range Tools() {
		if _, err := json.Marshal(tool); err != nil {
			t.Fatal(err)
		}
	}
}

// fakeModel is an OpenAI-compatible server that answers with the
// replies in order, streaming each as server-sent events, and records
// the requests.
type fakeModel struct {
	mu       sync.Mutex
	replies  []string
	requests []map[string]any
}

func (f *fakeModel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer key" {
		http.Error(w, `{"error":{"message":"bad request"}}`, http.StatusUnauthorized)
		return
	}
	var body map[string]any
	json.NewDecoder(r.Body).Decode(&body)
	f.mu.Lock()
	f.requests = append(f.requests, body)
	reply := f.replies[0]
	f.replies = f.replies[1:]
	f.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, reply)
}

// sse writes chunks as a stream of deltas.
func sse(deltas ...string) string {
	var b strings.Builder
	for _, d := range deltas {
		fmt.Fprintf(&b, "data: {\"choices\":[{\"delta\":%s}]}\n\n", d)
	}
	return b.String() + "data: [DONE]\n\n"
}

func TestRunCallsToolsUntilTheModelAnswers(t *testing.T) {
	model := &fakeModel{replies: []string{
		sse(`{"content":"Drawing"}`,
			`{"tool_calls":[{"index":0,"id":"c1","function":{"name":"add_nodes","arguments":"{\"nodes\":[{\"id\":\"a\","}}]}`,
			`{"tool_calls":[{"index":0,"function":{"arguments":"\"label\":\"A\"}]}"}}]}`),
		sse(`{"reasoning_content":"hmm"}`, `{"content":"Done."}`),
	}}
	srv := httptest.NewServer(model)
	defer srv.Close()
	var doc flowfile.Project
	var text, reasoning strings.Builder
	added, err := Run(context.Background(), Provider{BaseURL: srv.URL + "/v1/", APIKey: "key", Model: "m"},
		[]Message{{Role: "user", Content: "Draw A"}}, Tools(), Hooks{
			Prompt: func() string { return SystemPrompt(doc, Location{}, Context{}) },
			Delta:  func(d Delta) { text.WriteString(d.Text); reasoning.WriteString(d.Reasoning) },
			Reply:  func(Message) {},
			Call: func(c ToolCall) string {
				r := Apply(doc, Location{}, c.Function.Name, c.Function.Arguments)
				doc = r.Project
				return r.Output
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 3 || added[1].Role != "tool" || added[1].ToolCallID != "c1" || added[2].Content != "Done." {
		t.Fatalf("messages: %+v", added)
	}
	if n, ok := doc.Screens[0].Flows[0].Node("a"); !ok || n.Label != "A" {
		t.Fatalf("the tool did not run: %+v", doc)
	}
	if text.String() != "DrawingDone." || reasoning.String() != "hmm" {
		t.Fatalf("streamed %q, reasoning %q", text.String(), reasoning.String())
	}
	second := model.requests[1]["messages"].([]any)
	if prompt := second[0].(map[string]any)["content"].(string); !strings.Contains(prompt, `node a "A"`) {
		t.Fatalf("the second request's prompt misses the chart:\n%s", prompt)
	}
	if len(second) != 4 {
		t.Fatalf("the second request has %d messages", len(second))
	}
}

func TestCompleteReportsErrors(t *testing.T) {
	srv := httptest.NewServer(&fakeModel{})
	defer srv.Close()
	_, err := Complete(context.Background(), Provider{BaseURL: srv.URL + "/v1", APIKey: "wrong", Model: "m"}, nil, nil, func(Delta) {})
	if err == nil || !strings.Contains(err.Error(), "bad request (HTTP 401)") {
		t.Fatalf("error: %v", err)
	}
}

func TestReadFolder(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod":                  {Data: []byte("module demo\n")},
		"cmd/server/main.go":      {Data: []byte("package main\n\nfunc main() {\n\tserve()\n}\n")},
		"internal/order/order.go": {Data: []byte("package order\n\nfunc Place() {}\nfunc Ship() {}\n")},
		"node_modules/x/index.js": {Data: []byte("func Place() {}\n")},
		".git/HEAD":               {Data: []byte("ref: main\n")},
		"assets/logo.png":         {Data: []byte("\x89PNG\x00\x00")},
	}
	steps := []struct {
		tool, args, want string
		failed           bool
	}{
		{ToolListDir, ``, ".git/\nassets/\ncmd/\ngo.mod  12 B\ninternal/\nnode_modules/\n", false},
		{ToolReadFile, `{"path":"cmd/server/main.go","offset":3,"limit":2}`, "     3\tfunc main() {\n     4\t\tserve()\n… 1 more lines; read on with offset=5\n", false},
		{ToolGlob, `{"pattern":"**/*.go"}`, "cmd/server/main.go\ninternal/order/order.go\n", false},
		{ToolGlob, `{"pattern":"*.go","path":"internal/order"}`, "internal/order/order.go\n", false},
		{ToolGrep, `{"pattern":"func (Place|main)","include":"*.go"}`, "cmd/server/main.go:3: func main() {\ninternal/order/order.go:3: func Place() {}\n", false},
		{ToolGrep, `{"pattern":"PNG"}`, "no matches", false},
		{ToolReadFile, `{"path":"../secret"}`, `error: "../secret" is outside the project`, true},
		{ToolReadFile, `{"path":"assets/logo.png"}`, "error: assets/logo.png is not a text file", true},
		{ToolReadFile, `{"path":"cmd"}`, "error: cmd is a folder; use list_dir", true},
	}
	for _, s := range steps {
		out, failed := ReadFolder(fsys, s.tool, s.args)
		if out != s.want || failed != s.failed {
			t.Errorf("%s %s:\n%q, failed %v", s.tool, s.args, out, failed)
		}
	}
	prompt := SystemPrompt(flowfile.Project{}, Location{}, Context{File: "/w/app.flow", Folder: "/w", Others: []string{"/x/b.flow"}})
	if !strings.Contains(prompt, `in the folder "/w"`) || !strings.Contains(prompt, "/x/b.flow") {
		t.Fatal("the prompt does not mention the folder and the other files")
	}
}

func TestSetElement(t *testing.T) {
	p, _ := flowfile.ParseProject("screen a\n  box\n    text \"old\"\n  end\n")
	at := Location{Screen: "a", Design: true}
	r := Apply(p, at, ToolSetElement, `{"path":"0.0","ui":"text \"new\" class=\"font-bold\""}`)
	if r.Failed || r.Show.Element != "0.0" || !strings.Contains(r.Project.Format(), `text "new" class="font-bold"`) {
		t.Fatalf("set_element: %+v", r)
	}
	if r := Apply(p, at, ToolSetElement, `{"path":"0.0","ui":""}`); r.Failed || strings.Contains(r.Project.Format(), "old") {
		t.Fatalf("removing: %+v", r)
	}
	if r := Apply(p, at, ToolSetElement, `{"path":"3","ui":""}`); !r.Failed {
		t.Fatalf("a missing element: %+v", r)
	}
	prompt := SystemPrompt(p, Location{Screen: "a", Design: true, Element: "0.0"}, Context{})
	if !strings.Contains(prompt, "selected element 0.0") || !strings.Contains(prompt, `text "old"`) {
		t.Fatalf("the prompt does not show the selection:\n%s", prompt)
	}
}

func TestComments(t *testing.T) {
	p, _ := flowfile.ParseProject("screen a\n  box\n    text \"old\" comment=\"Say hello\"\n  end\n")
	at := Location{Screen: "a", Design: true}
	if prompt := SystemPrompt(p, at, Context{}); !strings.Contains(prompt, `screen a, element 0.0: "Say hello"`) {
		t.Fatalf("the prompt does not list the comment:\n%s", prompt)
	}
	// An element rewritten as one keeps its comment.
	r := Apply(p, at, ToolSetElement, `{"path":"0.0","ui":"text \"Hello\""}`)
	if r.Failed || !strings.Contains(r.Project.Format(), `text "Hello" comment="Say hello"`) || strings.Contains(r.Output, "warning") {
		t.Fatalf("set_element: %+v", r)
	}
	// Rewriting the interface without it warns of it.
	if r := Apply(p, at, ToolSetUI, `{"ui":"text \"Hello\""}`); !strings.Contains(r.Output, `warning: this dropped the user's comments "Say hello" on element 0.0 of screen a`) {
		t.Fatalf("set_ui: %q", r.Output)
	}
	r = Apply(r.Project, at, ToolResolveComment, `{"path":"0.0"}`)
	if r.Failed || strings.Contains(r.Project.Format(), "comment") || r.Show.Element != "0.0" {
		t.Fatalf("resolve_comment: %+v", r)
	}
	if r := Apply(r.Project, at, ToolResolveComment, `{"path":"0.0"}`); !r.Failed {
		t.Fatalf("resolved an element without a comment: %+v", r)
	}
}
