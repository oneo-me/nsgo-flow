package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

// The app counts its use in Umami under a site of its own, as NSGo's
// gallery does: a pageview at start, an event for each message sent and
// heartbeats while it is used, tagged with the platform and carrying a
// random ID kept in the storage, so that one install is one visitor. No
// chart, message or path is sent. Builds of nsgo dev (statsBuild) and test
// binaries report nothing, and the settings turn it off.
const (
	statsURL     = "https://umami.oneo.me/api/send"
	statsWebsite = "0d6c7a62-255c-498f-ad36-c4c57d93ab0c"
	statsHost    = "flow.oneo.me"
)

// Storage keys of the statistics: the install's ID, and keyStats "off"
// when the settings turned them off.
const (
	keyInstall = "installID"
	keyStats   = "stats"
)

// Umami measures a visit from its first event to its last, so while the
// app is used it sends a heartbeat each statsBeat, for statsIdle after the
// chart last changed or a message was last sent: a window left open does
// not count as use.
const (
	statsBeat = time.Minute
	statsIdle = 5 * time.Minute
)

// statsOn reports whether the settings leave the statistics on.
func statsOn(cx *core.Scope) bool {
	v, _ := cx.Storage().GetItem(keyStats)
	return v != "off"
}

// setStatsOn turns the statistics on or off.
func setStatsOn(cx *core.Scope, on bool) {
	if on {
		cx.Storage().RemoveItem(keyStats)
	} else {
		cx.Storage().SetItem(keyStats, "off")
	}
}

// statsPayload is an event of Umami's /api/send: a pageview without a
// name.
type statsPayload struct {
	Website  string `json:"website"`
	Hostname string `json:"hostname"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Language string `json:"language,omitempty"`
	Screen   string `json:"screen,omitempty"`
	Tag      string `json:"tag"`
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
}

// reportUsage sends the pageview, the events and the heartbeats. Events
// are made on the graph's goroutine and sent from another, oldest first;
// one that fails is dropped.
func reportUsage(cx *core.Scope, e *editor, a *assistant) {
	if !statsBuild || testing.Testing() {
		return
	}
	store := cx.Storage()
	id, ok := store.GetItem(keyInstall)
	if !ok {
		b := make([]byte, 16)
		rand.Read(b)
		id = hex.EncodeToString(b)
		store.SetItem(keyInstall, id)
	}
	env := cx.Env()
	agent := statsAgent(env.OS())
	queue := make(chan statsPayload, 32)
	go sendStats(queue, agent)

	send := func(name string) {
		if !statsOn(cx) {
			return
		}
		w := env.Window()
		p := statsPayload{Website: statsWebsite, Hostname: statsHost, URL: "/", Title: "Flow Editor",
			Language: env.Locale(), Screen: fmt.Sprintf("%dx%d", int(w.Width), int(w.Height)),
			Tag: osName(env.OS()), ID: id, Name: name}
		select {
		case queue <- p:
		default:
		}
	}
	since := time.Now()
	send("")
	cx.Effect(func() {
		e.doc.Get()
		core.Untracked(cx, func() bool { since = time.Now(); return true })
	})
	turns := 0
	cx.Effect(func() {
		n := len(a.turns.Get())
		core.Untracked(cx, func() bool {
			if n > turns {
				since = time.Now()
				send("message")
			}
			turns = n
			return true
		})
	})
	var beat func()
	beat = func() {
		if time.Since(since) < statsIdle {
			send("heartbeat")
		}
		cx.AfterFunc(statsBeat, beat)
	}
	cx.AfterFunc(statsBeat, beat)
}

// sendStats posts the queue's events to Umami until it closes.
func sendStats(queue <-chan statsPayload, agent string) {
	client := &http.Client{Timeout: 10 * time.Second}
	for p := range queue {
		body, _ := json.Marshal(map[string]any{"type": "event", "payload": p})
		req, err := http.NewRequest(http.MethodPost, statsURL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if agent != "" {
			req.Header.Set("User-Agent", agent)
		}
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
		}
	}
}

// statsAgent is the User-Agent the app sends, as a browser on the
// platform would say it, for Umami to tell the system and not take it for
// a bot as it does Go's own.
func statsAgent(os core.OS) string {
	platforms := map[core.OS]string{
		core.OSMacOS:   "Macintosh; Intel Mac OS X 10_15_7",
		core.OSWindows: "Windows NT 10.0; Win64; x64",
		core.OSLinux:   "X11; Linux x86_64",
	}
	p, ok := platforms[os]
	if !ok {
		return ""
	}
	return "Mozilla/5.0 (" + p + ") NSGoFlow/1.0"
}

// osName is the platform's name, as the statistics tag it.
func osName(os core.OS) string {
	names := map[core.OS]string{core.OSMacOS: "macOS", core.OSWindows: "Windows", core.OSLinux: "Linux"}
	if n, ok := names[os]; ok {
		return n
	}
	return "unknown"
}
