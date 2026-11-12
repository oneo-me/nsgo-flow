package main

import (
	"strings"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestLongConversationKeepsTheComposer(t *testing.T) {
	long := strings.Repeat("A long line of the reply. ", 20)
	var replies []string
	for range 6 {
		replies = append(replies, deltas(`{"content":"`+strings.Repeat(long+`\n\n`, 6)+`"}`))
	}
	url := model(t, replies...)
	app := nsgotest.Start(t, Root, nsgotest.Size(1280, 800),
		nsgotest.Storage(map[string]string{keyBaseURL: url, keyModel: "m"}))
	newFileAt(t, app, "long.flow")
	for i := range 6 {
		app.Type("label:Message", "Question")
		app.Press("label:Send")
		app.WaitUntil(5*time.Second, func() bool { return len(app.Query("text:Question")) == i+1 && app.Exists("label:Send") })
		app.Settle()
	}
	// The transcript scrolls and the composer stays in the window.
	var bottom float64
	for v := app.Query("id:composer")[0]; v != nil; v = v.Parent {
		bottom += float64(v.Frame.Origin.Y)
	}
	bottom += float64(app.Query("id:composer")[0].Frame.Size.Height)
	if bottom > 800 {
		t.Fatalf("the composer ends at %v, below the window\n%s", bottom, app.Dump())
	}
}
