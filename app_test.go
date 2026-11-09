package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oneo-me/nsgo/nsgotest"
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

func TestAssistantDrawsOnTheChart(t *testing.T) {
	url := model(t,
		deltas(`{"tool_calls":[{"index":0,"id":"c1","function":{"name":"add_nodes","arguments":"{\"nodes\":[{\"id\":\"order\",\"label\":\"Order placed\",\"kind\":\"start\"},{\"id\":\"ship\",\"label\":\"Ship order\"}]}"}},{"index":1,"id":"c2","function":{"name":"add_edges","arguments":"{\"edges\":[{\"from\":\"order\",\"to\":\"ship\"}]}"}}]}`),
		deltas(`{"content":"I drew **two steps**."}`),
	)
	app := nsgotest.Start(t, Root, nsgotest.Size(1280, 800),
		nsgotest.Storage(map[string]string{keyBaseURL: url, keyModel: "test-model"}))
	app.Find("text:Ask the assistant to draw a flow, or open a .flow file.")
	app.Type("label:Message", "Chart an order flow")
	app.Press("label:Send")
	app.WaitFor("flow >> label:Ship order", 5*time.Second)
	app.WaitFor("text:two steps", 5*time.Second)
	app.Find("text:Chart an order flow")

	// The change undoes as one step per tool call.
	app.Press("id:undo")
	app.WaitForGone("flow >> label:Order placed", time.Second)
	app.Find("text:Ask the assistant to draw a flow, or open a .flow file.")
}

func TestAssistantShowsFailuresAndAsksForAProvider(t *testing.T) {
	app := nsgotest.Start(t, Root, nsgotest.Size(1280, 800))
	app.Find("text:Set up an OpenAI-compatible provider to start chatting.")
	app.Type("label:Message", "Hello")
	app.Press("label:Send")
	app.WaitFor("id:settings", time.Second)
	app.Type("id:model", "test-model")
	app.Clear("id:base-url")
	app.Type("id:base-url", model(t))
	app.Press("id:settings-done")
	app.WaitForGone("id:settings", 2*time.Second)
	app.Press("label:Send")
	app.WaitFor("text:no more replies (HTTP 500)", 5*time.Second)
	app.Find("id:retry")
}

func TestOpensAFlowFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.flow")
	if err := os.WriteFile(path, []byte("title \"Plan\"\nnode a \"Research\" note=\"Read the docs\"\na -> b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var e *editor
	app := nsgotest.Start(t, Root, nsgotest.Size(1280, 800))
	app.Post(func() { e = shown })
	app.Settle()
	app.Post(func() {
		if err := e.openPath(path); err != nil {
			t.Error(err)
		}
	})
	app.Settle()
	app.Find("flow >> label:Research")
	app.Find("text:Read the docs")
	if got := app.Find("id:doc-title").Text; got != "Plan" {
		t.Fatalf("title %q", got)
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
