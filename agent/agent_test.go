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

	"github.com/oneo-me/nsgo-flow/flowfile"
)

func TestApply(t *testing.T) {
	var d flowfile.Doc
	steps := []struct{ tool, args, want string }{
		{ToolAddNodes, `{"nodes":[{"id":"a","label":"Start","kind":"start"},{"id":"b","label":"Work"}]}`, "added 2 nodes"},
		{ToolAddEdges, `{"edges":[{"from":"a","to":"b","label":"go","dashed":true}]}`, "added 1 edges"},
		{ToolUpdateNode, `{"id":"b","rename_to":"work","status":"doing"}`, "updated b"},
		{ToolSetLayout, `{"title":"Plan","direction":"right"}`, "layout set"},
	}
	for _, s := range steps {
		r := Apply(d, s.tool, s.args)
		if r.Failed || r.Output != s.want {
			t.Fatalf("%s: %q", s.tool, r.Output)
		}
		d = r.Doc
	}
	want := "title \"Plan\"\ndirection right\n\nnode a \"Start\" kind=start\nnode work \"Work\" status=doing\n\na ..> work \"go\"\n"
	if got := d.Format(); got != want {
		t.Fatalf("chart:\n%s", got)
	}
	if r := Apply(d, ToolGetFlow, ""); r.Output != want || r.Changed {
		t.Fatalf("get_flow: %+v", r)
	}
	if r := Apply(d, ToolFocus, `{"id":"work"}`); r.Focus != "work" || r.Changed {
		t.Fatalf("focus: %+v", r)
	}
	r := Apply(d, ToolRemoveNodes, `{"ids":["a","nope"]}`)
	if !r.Failed || len(r.Doc.Nodes) != 2 || len(d.Nodes) != 2 {
		t.Fatalf("a failed call changed the chart: %+v", r)
	}
	r = Apply(d, ToolReplaceFlow, `{"text":"x -> y\ny -> z"}`)
	if r.Failed || len(r.Doc.Nodes) != 3 {
		t.Fatalf("replace: %+v", r)
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
	var doc flowfile.Doc
	var text, reasoning strings.Builder
	added, err := Run(context.Background(), Provider{BaseURL: srv.URL + "/v1/", APIKey: "key", Model: "m"},
		[]Message{{Role: "user", Content: "Draw A"}}, Tools(), Hooks{
			Prompt: func() string { return SystemPrompt(doc, "") },
			Delta:  func(d Delta) { text.WriteString(d.Text); reasoning.WriteString(d.Reasoning) },
			Reply:  func(Message) {},
			Call: func(c ToolCall) string {
				r := Apply(doc, c.Function.Name, c.Function.Arguments)
				doc = r.Doc
				return r.Output
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 3 || added[1].Role != "tool" || added[1].ToolCallID != "c1" || added[2].Content != "Done." {
		t.Fatalf("messages: %+v", added)
	}
	if n, ok := doc.Node("a"); !ok || n.Label != "A" {
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
