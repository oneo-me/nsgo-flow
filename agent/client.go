// Package agent talks to an OpenAI-compatible chat completions API and
// gives the model tools that draw on a flow document.
package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// Provider is where requests go: an OpenAI-compatible API.
type Provider struct {
	// BaseURL is the API's root, such as https://api.openai.com/v1.
	BaseURL string
	APIKey  string
	Model   string
}

// Ready reports whether p names enough to send a request.
func (p Provider) Ready() bool {
	return strings.TrimSpace(p.BaseURL) != "" && strings.TrimSpace(p.Model) != ""
}

// Message is one message of the conversation, as the API takes it.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall is a call the model asks for.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Tool is a function the model may call, its parameters a JSON Schema.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

func (t Tool) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"type": "function", "function": map[string]any{
		"name": t.Name, "description": t.Description, "parameters": t.Parameters}})
}

// Delta is what a stream adds to the reply: text, reasoning text from
// models that send it apart, or nothing but its end.
type Delta struct {
	Text, Reasoning string
}

// Complete sends messages and streams the reply, calling onDelta from
// the goroutine it runs on as text arrives. It returns the whole reply,
// with the tool calls the model asks for.
func Complete(ctx context.Context, p Provider, messages []Message, tools []Tool, onDelta func(Delta)) (Message, error) {
	body := map[string]any{"model": p.Model, "messages": messages, "stream": true}
	if len(tools) > 0 {
		body["tools"] = tools
	}
	data, err := json.Marshal(body)
	if err != nil {
		return Message{}, err
	}
	url := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if key := strings.TrimSpace(p.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return Message{}, apiError(resp)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
		// A provider that ignores stream answers in one piece.
		return whole(resp.Body, onDelta)
	}
	return stream(resp.Body, onDelta)
}

// apiError is what a failed response says.
func apiError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	msg := strings.TrimSpace(string(data))
	if json.Unmarshal(data, &e) == nil {
		if e.Error.Message != "" {
			msg = e.Error.Message
		} else if e.Message != "" {
			msg = e.Message
		}
	}
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return fmt.Errorf("%s (HTTP %d)", msg, resp.StatusCode)
}

type choiceMessage struct {
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
	Reasoning        string `json:"reasoning"`
	ToolCalls        []struct {
		Index    int    `json:"index"`
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
}

func whole(r io.Reader, onDelta func(Delta)) (Message, error) {
	var out struct {
		Choices []struct {
			Message choiceMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return Message{}, err
	}
	if len(out.Choices) == 0 {
		return Message{}, errors.New("the reply has no choices")
	}
	m := out.Choices[0].Message
	reply := Message{Role: "assistant", Content: m.Content}
	for _, c := range m.ToolCalls {
		call := ToolCall{ID: c.ID, Type: "function"}
		call.Function.Name, call.Function.Arguments = c.Function.Name, c.Function.Arguments
		reply.ToolCalls = append(reply.ToolCalls, call)
	}
	if reasoning := m.ReasoningContent + m.Reasoning; reasoning != "" {
		onDelta(Delta{Reasoning: reasoning})
	}
	if m.Content != "" {
		onDelta(Delta{Text: m.Content})
	}
	return reply, nil
}

func stream(r io.Reader, onDelta func(Delta)) (Message, error) {
	reply := Message{Role: "assistant"}
	var text strings.Builder
	calls := map[int]*ToolCall{}
	lines := bufio.NewScanner(r)
	lines.Buffer(make([]byte, 64<<10), 16<<20)
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		payload, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		payload = strings.TrimSpace(payload)
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta choiceMessage `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return Message{}, errors.New(chunk.Error.Message)
		}
		for _, c := range chunk.Choices {
			d := c.Delta
			if reasoning := d.ReasoningContent + d.Reasoning; reasoning != "" {
				onDelta(Delta{Reasoning: reasoning})
			}
			if d.Content != "" {
				text.WriteString(d.Content)
				onDelta(Delta{Text: d.Content})
			}
			for _, tc := range d.ToolCalls {
				call := calls[tc.Index]
				if call == nil {
					call = &ToolCall{Type: "function"}
					calls[tc.Index] = call
				}
				if tc.ID != "" {
					call.ID = tc.ID
				}
				call.Function.Name += tc.Function.Name
				call.Function.Arguments += tc.Function.Arguments
			}
		}
	}
	if err := lines.Err(); err != nil {
		return Message{}, err
	}
	reply.Content = text.String()
	indexes := make([]int, 0, len(calls))
	for i := range calls {
		indexes = append(indexes, i)
	}
	sort.Ints(indexes)
	for n, i := range indexes {
		call := *calls[i]
		if call.ID == "" {
			call.ID = fmt.Sprintf("call_%d", n)
		}
		reply.ToolCalls = append(reply.ToolCalls, call)
	}
	return reply, nil
}
