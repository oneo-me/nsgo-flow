package agent

import (
	"context"
	"errors"
)

// MaxRounds bounds the requests one turn makes while the model keeps
// calling tools.
const MaxRounds = 24

// ErrTooManyRounds ends a turn whose model kept calling tools.
var ErrTooManyRounds = errors.New("the model kept calling tools; stopped")

// Hooks are how a turn reaches the app. Each runs on the turn's
// goroutine and may block, as Call does until the app ran the tool.
type Hooks struct {
	// Prompt is the system prompt, asked for before each request.
	Prompt func() string
	// Delta streams the reply being written.
	Delta func(Delta)
	// Reply follows each complete reply, before its tools run.
	Reply func(Message)
	// Call runs a tool and returns what the model reads back.
	Call func(ToolCall) string
}

// Run is one turn: it sends history, runs the tools the model calls and
// sends their results back until the model answers without calling one.
// It returns the messages the turn added, those before a failure too.
func Run(ctx context.Context, p Provider, history []Message, tools []Tool, h Hooks) ([]Message, error) {
	var added []Message
	for range MaxRounds {
		messages := append([]Message{{Role: "system", Content: h.Prompt()}}, history...)
		messages = append(messages, added...)
		reply, err := Complete(ctx, p, messages, tools, h.Delta)
		if err != nil {
			if ctx.Err() != nil {
				err = ctx.Err()
			}
			return added, err
		}
		added = append(added, reply)
		h.Reply(reply)
		if len(reply.ToolCalls) == 0 {
			return added, nil
		}
		for _, call := range reply.ToolCalls {
			if ctx.Err() != nil {
				// Every call needs an answer for the history to stay valid.
				added = append(added, Message{Role: "tool", ToolCallID: call.ID, Content: "error: cancelled by the user"})
				continue
			}
			added = append(added, Message{Role: "tool", ToolCallID: call.ID, Content: h.Call(call)})
		}
		if ctx.Err() != nil {
			return added, ctx.Err()
		}
	}
	return added, ErrTooManyRounds
}
