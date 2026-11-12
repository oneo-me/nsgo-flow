package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/base/textview"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/nsgo-dev/nsgo/ui/chat"

	"github.com/oneo-me/nsgo-flow/agent"
	"github.com/oneo-me/nsgo-flow/locales"
)

// call is a tool call of a round, as the transcript shows it.
type call struct {
	id, name, arguments string
	status              chat.Status
	output              string
}

// round is one reply of the model: what it thought, wrote and called.
type round struct {
	reasoning, text string
	calls           []call
}

// turn is a message of the user's and the replies to it.
type turn struct {
	id     string
	prompt string
	status *core.Signal[chat.Status]
	rounds *core.Signal[[]round]
	err    *core.Signal[string]
	start  time.Time
	end    *core.Signal[time.Time]
}

// update changes the turn's latest round.
func (t *turn) update(fn func(r *round)) {
	t.rounds.Update(func(rs []round) []round {
		rs = slices.Clone(rs)
		if len(rs) == 0 {
			rs = append(rs, round{})
		}
		fn(&rs[len(rs)-1])
		return rs
	})
}

// assistant is the conversation about the open document.
type assistant struct {
	cx *core.Scope
	e  *editor
	// history is what the model has been told, the system prompt aside,
	// which each request writes afresh.
	history      []agent.Message
	turns        *core.Signal[[]*turn]
	draft        *core.Signal[string]
	running      *core.Signal[bool]
	now          *core.Signal[time.Time]
	conversation *chat.ConversationState
	cancel       context.CancelFunc
	ids          int
	// openSettings shows the provider's settings.
	openSettings func()
}

func newAssistant(cx *core.Scope, e *editor, openSettings func()) *assistant {
	a := &assistant{cx: cx, e: e, openSettings: openSettings,
		turns:        core.NewSignalFunc(cx, []*turn(nil), func(a, b []*turn) bool { return false }),
		draft:        core.NewSignal(cx, ""),
		running:      core.NewSignal(cx, false),
		now:          core.NewSignal(cx, cx.Now()),
		conversation: chat.NewConversationState(cx)}
	cx.Cleanup(a.stop)
	return a
}

// clear starts a new conversation.
func (a *assistant) clear() {
	a.stop()
	a.history = nil
	a.turns.Set(nil)
}

func (a *assistant) stop() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}

// send asks the model about the draft.
func (a *assistant) send() {
	text := strings.TrimSpace(a.draft.GetUntracked())
	if text == "" {
		return
	}
	p := core.Untracked(a.cx, func() agent.Provider { return provider(a.cx) })
	if !p.Ready() {
		a.openSettings()
		return
	}
	a.stop()
	a.draft.Set("")
	a.run(p, text)
}

// run starts a turn for prompt on a goroutine, which reaches the
// document through the UI's.
func (a *assistant) run(p agent.Provider, prompt string) {
	cx := a.cx
	a.ids++
	t := &turn{id: fmt.Sprintf("t%d", a.ids), prompt: prompt, start: cx.Now(),
		status: core.NewSignal(cx, chat.Running),
		rounds: core.NewSignalFunc(cx, []round{{}}, func(a, b []round) bool { return false }),
		err:    core.NewSignal(cx, ""), end: core.NewSignal(cx, time.Time{})}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.turns.Update(func(ts []*turn) []*turn { return append(slices.Clone(ts), t) })
	a.running.Set(true)
	a.conversation.JumpToLatest()
	history := append(slices.Clone(a.history), agent.Message{Role: "user", Content: prompt})
	// The folder a turn reads is the one attached when it began.
	dir, name := core.Untracked(cx, func() string { return folder(cx) }), ""
	tools := agent.Tools()
	if dir != "" {
		name = filepath.Base(dir)
		tools = append(tools, agent.FolderTools()...)
	}

	// onUI runs fn on the UI's goroutine and waits for it.
	onUI := func(fn func() string) string {
		answer := make(chan string, 1)
		cx.Post(func() { answer <- fn() })
		select {
		case s := <-answer:
			return s
		case <-ctx.Done():
			return "error: cancelled by the user"
		}
	}
	live := func(fn func()) {
		cx.Post(func() {
			if ctx.Err() == nil {
				fn()
			}
		})
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				live(func() { a.now.Set(cx.Now()) })
			}
		}
	}()
	go func() {
		added, err := agent.Run(ctx, p, history, tools, agent.Hooks{
			Prompt: func() string {
				return onUI(func() string { return agent.SystemPrompt(a.e.doc.GetUntracked(), a.e.at(), name) })
			},
			Delta: func(d agent.Delta) {
				live(func() {
					t.update(func(r *round) { r.reasoning += d.Reasoning; r.text += d.Text })
				})
			},
			Reply: func(m agent.Message) {
				live(func() {
					t.rounds.Update(func(rs []round) []round {
						rs = slices.Clone(rs)
						last := &rs[len(rs)-1]
						last.text = m.Content
						for _, c := range m.ToolCalls {
							last.calls = append(last.calls, call{id: c.ID, name: c.Function.Name, arguments: c.Function.Arguments, status: chat.Running})
						}
						if len(m.ToolCalls) > 0 {
							rs = append(rs, round{})
						}
						return rs
					})
				})
			},
			Call: func(c agent.ToolCall) string {
				if agent.IsFolderTool(c.Function.Name) {
					// Files are read here, off the UI's goroutine.
					out, failed := readFolder(dir, c)
					return onUI(func() string { a.done(t, c.ID, out, failed); return out })
				}
				return onUI(func() string { return a.call(t, c) })
			},
		})
		cx.Post(func() { a.finish(t, cancel, history, added, err) })
	}()
}

// call runs a tool on the document and marks it done in the transcript.
func (a *assistant) call(t *turn, c agent.ToolCall) string {
	r := agent.Apply(a.e.doc.GetUntracked(), a.e.at(), c.Function.Name, c.Function.Arguments)
	if r.Changed {
		a.e.change(r.Project, t.id)
	}
	if r.Show.Screen != "" {
		a.e.show(r.Show)
	}
	a.done(t, c.ID, r.Output, r.Failed)
	return r.Output
}

// done marks a call done in the transcript.
func (a *assistant) done(t *turn, id, output string, failed bool) {
	t.rounds.Update(func(rs []round) []round {
		rs = slices.Clone(rs)
		for i := range rs {
			rs[i].calls = slices.Clone(rs[i].calls)
			for j := range rs[i].calls {
				if cl := &rs[i].calls[j]; cl.id == id {
					cl.output, cl.status = output, chat.Completed
					if failed {
						cl.status = chat.Failed
					}
				}
			}
		}
		return rs
	})
}

// readFolder runs a folder tool inside dir, which it cannot leave, not
// even through a symbolic link.
func readFolder(dir string, c agent.ToolCall) (string, bool) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "error: " + err.Error(), true
	}
	defer root.Close()
	return agent.ReadFolder(root.FS(), c.Function.Name, c.Function.Arguments)
}

// pickFolder asks for the project folder the assistant may read.
func (a *assistant) pickFolder() {
	a.cx.PickDirectories(core.PickDirectoriesOptions{}, func(dirs []core.Directory, err error) {
		if err != nil {
			a.e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(a.cx).FolderFailed(err.Error())})
			return
		}
		if len(dirs) > 0 && dirs[0].Path != "" {
			setFolder(a.cx, dirs[0].Path)
		}
	})
}

// finish ends a turn, keeping what it added to the history unless it
// failed.
func (a *assistant) finish(t *turn, cancel context.CancelFunc, history, added []agent.Message, err error) {
	cancel()
	cancelled := errors.Is(err, context.Canceled)
	a.cx.Batch(func() {
		// Calls a stop cut short end with the turn.
		t.rounds.Update(func(rs []round) []round {
			rs = slices.Clone(rs)
			for i := range rs {
				rs[i].calls = slices.Clone(rs[i].calls)
				for j := range rs[i].calls {
					if rs[i].calls[j].status == chat.Running {
						rs[i].calls[j].status = chat.Cancelled
					}
				}
			}
			if n := len(rs); n > 1 && rs[n-1].text == "" && rs[n-1].reasoning == "" && len(rs[n-1].calls) == 0 {
				rs = rs[:n-1]
			}
			return rs
		})
		t.end.Set(a.cx.Now())
		switch {
		case err == nil:
			t.status.Set(chat.Completed)
		case cancelled:
			t.status.Set(chat.Cancelled)
		default:
			t.status.Set(chat.Failed)
			t.err.Set(err.Error())
		}
		if err == nil || cancelled {
			a.history = append(history, added...)
		}
		if a.cancel != nil && t == a.last() {
			a.cancel = nil
		}
		if t == a.last() {
			a.running.Set(false)
		}
	})
}

func (a *assistant) last() *turn {
	ts := a.turns.GetUntracked()
	if len(ts) == 0 {
		return nil
	}
	return ts[len(ts)-1]
}

// retry runs a failed turn again in its place.
func (a *assistant) retry(t *turn) {
	p := core.Untracked(a.cx, func() agent.Provider { return provider(a.cx) })
	if !p.Ready() {
		a.openSettings()
		return
	}
	a.stop()
	a.turns.Update(func(ts []*turn) []*turn {
		return slices.DeleteFunc(slices.Clone(ts), func(x *turn) bool { return x == t })
	})
	a.run(p, t.prompt)
}

// toolLook is how the transcript names a tool.
func toolLook(tr locales.Messages, name string) (func() *core.SVG, string) {
	switch name {
	case agent.ToolGetProject:
		return lucide.FileText, tr.ToolGetProject()
	case agent.ToolAddScreen:
		return lucide.SquarePlus, tr.ToolAddScreen()
	case agent.ToolSetUI:
		return lucide.PanelsTopLeft, tr.ToolSetUI()
	case agent.ToolRemoveScreen:
		return lucide.SquareMinus, tr.ToolRemoveScreen()
	case agent.ToolAddFlow:
		return lucide.GitBranchPlus, tr.ToolAddFlow()
	case agent.ToolRemoveFlow:
		return lucide.Trash, tr.ToolRemoveFlow()
	case agent.ToolAddNodes:
		return lucide.Plus, tr.ToolAddNodes()
	case agent.ToolUpdateNode:
		return lucide.Pencil, tr.ToolUpdateNode()
	case agent.ToolRemoveNodes:
		return lucide.Trash, tr.ToolRemoveNodes()
	case agent.ToolAddEdges:
		return lucide.Link, tr.ToolAddEdges()
	case agent.ToolRemoveEdges:
		return lucide.Unlink, tr.ToolRemoveEdges()
	case agent.ToolSetLayout:
		return lucide.LayoutGrid, tr.ToolSetLayout()
	case agent.ToolReplaceFlow:
		return lucide.Workflow, tr.ToolReplaceFlow()
	case agent.ToolReplaceProject:
		return lucide.LayoutTemplate, tr.ToolReplaceProject()
	case agent.ToolShow:
		return lucide.Crosshair, tr.ToolShow()
	case agent.ToolListDir:
		return lucide.FolderOpen, tr.ToolListDir()
	case agent.ToolReadFile:
		return lucide.FileCode, tr.ToolReadFile()
	case agent.ToolGlob:
		return lucide.FileSearch, tr.ToolGlob()
	case agent.ToolGrep:
		return lucide.TextSearch, tr.ToolGrep()
	}
	return lucide.Wrench, name
}

// callDetail summarizes a call's arguments in a line.
func callDetail(name, arguments string) string {
	var a struct {
		ID, Title, Direction, Text string
		Screen, Flow, Node         string
		Path, Pattern              string
		Nodes                      []struct{ ID, Label string }
		IDs                        []string
		Edges                      []struct{ From, To string }
	}
	if json.Unmarshal([]byte(arguments), &a) != nil {
		return ""
	}
	var parts []string
	switch name {
	case agent.ToolAddNodes:
		for _, n := range a.Nodes {
			parts = append(parts, cmp(n.Label, n.ID))
		}
	case agent.ToolRemoveNodes:
		parts = a.IDs
	case agent.ToolAddEdges, agent.ToolRemoveEdges:
		for _, e := range a.Edges {
			parts = append(parts, e.From+" → "+e.To)
		}
	case agent.ToolSetLayout:
		parts = slices.DeleteFunc([]string{a.Title, a.Direction}, func(s string) bool { return s == "" })
	case agent.ToolReplaceFlow, agent.ToolReplaceProject:
		return fmt.Sprintf("%d lines", strings.Count(strings.TrimSpace(a.Text), "\n")+1)
	case agent.ToolAddScreen, agent.ToolAddFlow:
		return cmp(a.Title, a.ID)
	case agent.ToolSetUI:
		return a.Screen
	case agent.ToolShow:
		parts = slices.DeleteFunc([]string{a.Screen, a.Flow, a.Node}, func(s string) bool { return s == "" })
	case agent.ToolListDir, agent.ToolReadFile:
		return cmp(a.Path, ".")
	case agent.ToolGlob, agent.ToolGrep:
		parts = slices.DeleteFunc([]string{a.Pattern, a.Path}, func(s string) bool { return s == "" })
	default:
		return a.ID
	}
	return strings.Join(parts, ", ")
}

func cmp(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// markdown is a reply's text, streaming while live says.
func markdown(md func() string, live func() bool) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		return ui.TextView{MarkdownFunc: md, Options: textview.Options{Streaming: live()}, Selectable: true}
	})
}

// turnView is a turn in the transcript.
func (a *assistant) turnView(cx *core.Scope, t *turn) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	done := func() bool { return t.status.Get() == chat.Completed }
	rounds := t.rounds.Get
	text := func(i int) func() string {
		return func() string {
			if rs := rounds(); i < len(rs) {
				return rs[i].text
			}
			return ""
		}
	}
	steps := func() []chat.Step {
		rs := rounds()
		running := t.status.Get() == chat.Running
		var out []chat.Step
		for i, r := range rs {
			final := done() && i == len(rs)-1
			var activities []chat.Activity
			if r.reasoning != "" {
				reasoning := r.reasoning
				status := chat.Completed
				if running && i == len(rs)-1 && r.text == "" && len(r.calls) == 0 {
					status = chat.Running
				}
				activities = append(activities, chat.Activity{ID: "think", Icon: lucide.Brain, Action: tr.Thinking(), Status: status,
					Content: func(*core.Scope) core.Node {
						return core.Text{Content: reasoning, Selectable: core.SelectableOn, Style: core.Style{FontSize: 12.5, LineHeight: 19}}
					}})
			}
			for _, c := range r.calls {
				icon, action := toolLook(tr, c.name)
				c := c
				activities = append(activities, chat.Activity{ID: c.id, Icon: icon, Action: action, Detail: callDetail(c.name, c.arguments),
					Status: c.status, Mono: true, MaxHeight: 240, Content: func(*core.Scope) core.Node {
						body := strings.TrimSpace(c.arguments)
						if c.output != "" {
							body += "\n\n→ " + c.output
						}
						return core.Text{Content: body, Selectable: core.SelectableOn}
					}})
			}
			step := chat.Step{ID: fmt.Sprintf("r%d", i), Activities: activities}
			if len(r.calls) > 0 {
				step.Title = tr.RanTools(len(r.calls))
			} else if len(activities) > 0 {
				step.Title = tr.Thinking()
			}
			if r.text != "" && !final {
				md, live := text(i), func() bool { return t.status.Get() == chat.Running && i == len(rounds())-1 }
				step.Message = func(*core.Scope) core.Node { return markdown(md, live) }
			}
			if step.Message != nil || len(step.Activities) > 0 {
				out = append(out, step)
			}
		}
		return out
	}
	seconds := func(until time.Time) int { return max(1, int(until.Sub(t.start).Seconds()+0.5)) }
	return chat.Turn{Status: t.status.Get, Steps: steps, Conversation: a.conversation, TestID: "turn-" + t.id,
		Summary: func() string { return tr.Worked(seconds(t.end.Get())) },
		Working: func() string { return tr.Working(seconds(a.now.Get())) },
		Prompt: func(*core.Scope) core.Node {
			return core.Text{Content: t.prompt, Selectable: core.SelectableOn, Style: core.Style{FontSize: 13, LineHeight: 19}}
		},
		Final: func(*core.Scope) core.Node {
			return markdown(func() string { return text(len(rounds()) - 1)() }, func() bool { return false })
		},
		HasFinal: func() bool { return done() && text(len(rounds())-1)() != "" },
		Footer: func(cx *core.Scope) core.Node {
			return core.Group{
				core.Show{When: func() bool { return t.status.Get() == chat.Failed }, Then: func(cx *core.Scope) core.Node {
					return errorRow(cx, t.err.Get, func() { a.retry(t) })
				}},
				core.Show{When: func() bool { return t.status.Get() == chat.Cancelled }, Then: func(cx *core.Scope) core.Node {
					return ui.Text(tr.Stopped(), 12, ui.Regular, colors.Muted)
				}},
			}
		}}
}

// errorRow is what went wrong with a turn and a Retry button.
func errorRow(cx *core.Scope, message func() string, retry func()) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.View{StyleFunc: func() core.Style {
		return core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, Padding: core.All(10), CornerRadius: ui.RadiusInset,
			Background: ui.TintsOf(cx).Danger(), BorderWidth: 1, BorderColor: ui.Alpha(colors.Danger(), 0.32)}
	}, Children: []core.Node{
		ui.Icon(lucide.CircleAlert, 16, colors.Danger),
		core.Text{ContentFunc: func() string {
			if m := message(); m != "" {
				return m
			}
			return tr.Failed()
		}, Selectable: core.SelectableOn, Style: core.Style{FlexGrow: 1, FlexShrink: 1, FontSize: 12.5, LineHeight: 18}, StyleFunc: ui.Colored(colors.Text)},
		ui.Button{Text: tr.Retry(), Variant: ui.ButtonOutline, Size: ui.SizeSmall, OnPress: retry, TestID: "retry"},
	}}
}

// folderRow is the project folder the assistant reads, which pressing
// changes, or a button that attaches one.
func folderRow(cx *core.Scope, a *assistant) core.Node {
	tr := locales.For(cx)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		dir := folder(cx)
		if dir == "" {
			return core.View{Style: core.Style{Direction: core.Row}, Children: []core.Node{
				ui.Button{Text: tr.AttachFolder(), Icon: lucide.FolderOpen, Variant: ui.ButtonGhost, Size: ui.SizeSmall,
					OnPress: a.pickFolder, TestID: "attach-folder"},
			}}
		}
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 2}, Children: []core.Node{
			ui.Button{Text: filepath.Base(dir), Label: tr.ChangeFolder(filepath.Base(dir)), Icon: lucide.FolderCode, Variant: ui.ButtonGhost,
				Size: ui.SizeSmall, OnPress: a.pickFolder, TestID: "change-folder"},
			ui.IconButton{Icon: lucide.X, Label: tr.DetachFolder(), Size: 24, OnPress: func() { setFolder(cx, "") }, TestID: "detach-folder"},
		}}
	})
}

// assistantPane is the conversation over the composer.
func assistantPane(cx *core.Scope, a *assistant) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	empty := func() bool { return len(a.turns.Get()) == 0 }
	ready := func() bool { return provider(cx).Ready() }
	header := core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, Height: core.Pt(44), Padding: core.Symmetric(0, 12),
		Borders: core.Edges{Bottom: 1}},
		StyleFunc: func() core.Style { return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}} },
		Children: []core.Node{
			ui.Icon(lucide.Sparkles, 16, colors.Muted),
			ui.Text(tr.Assistant(), 13, ui.Semibold, colors.Text),
			core.Text{ContentFunc: func() string { return provider(cx).Model }, MaxLines: 1,
				Style: ui.TypeCaption.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}), StyleFunc: ui.Colored(colors.Muted)},
			ui.Spacer{},
			ui.IconButton{Icon: lucide.MessageSquarePlus, Label: tr.NewChat(), OnPress: a.clear, TestID: "new-chat"},
			ui.IconButton{Icon: lucide.Settings, Label: tr.Settings(), OnPress: a.openSettings, TestID: "open-settings"},
		}}
	placeholder := core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(32), Gap: 10},
		Children: []core.Node{
			ui.Icon(lucide.Workflow, 32, colors.Muted),
			ui.Text(tr.EmptyTitle(), 15, ui.Semibold, colors.Text),
			core.Text{Content: tr.EmptyDetail(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(320)},
				StyleFunc: ui.Colored(colors.Muted)},
			core.Show{When: func() bool { return !ready() }, Then: func(cx *core.Scope) core.Node {
				return core.View{Style: core.Style{AlignItems: core.AlignCenter, Gap: 10, Margin: core.Edges{Top: 8}}, Children: []core.Node{
					core.Text{Content: tr.NoProvider(), Style: core.Style{FontSize: 12.5, TextAlign: core.TextAlignCenter}, StyleFunc: ui.Colored(colors.Warn)},
					ui.Button{Text: tr.OpenSettings(), Variant: ui.ButtonPrimary, OnPress: a.openSettings, TestID: "setup"},
				}}
			}},
		}}
	transcript := chat.Conversation{State: a.conversation, JumpLabel: tr.JumpToLatest(), TestID: "messages",
		Children: []core.Node{core.Each(a.turns.Get, func(t *turn) string { return t.id }, func(cx *core.Scope, t func() *turn, _ func() int) core.Node {
			return a.turnView(cx, core.Untracked(cx, t))
		})}}
	composer := core.View{Style: core.Style{Gap: 6, Padding: core.Edges{Left: 12, Right: 12, Bottom: 12}}, Children: []core.Node{
		folderRow(cx, a),
		chat.Composer{Draft: a.draft.Get, OnDraftChange: a.draft.Set, Label: tr.Message(), Placeholder: tr.Placeholder(),
			SendLabel: tr.Send(), StopLabel: tr.Stop(), Running: a.running.Get, OnSend: a.send, OnStop: a.stop, State: a.conversation, TestID: "composer"},
	}}
	return core.View{Style: ui.FillRest,
		StyleFunc: ui.Filled(colors.Background),
		Children: []core.Node{
			header,
			core.Show{When: empty, Then: func(*core.Scope) core.Node { return placeholder }, Else: func(*core.Scope) core.Node { return transcript }},
			composer,
		}}
}
