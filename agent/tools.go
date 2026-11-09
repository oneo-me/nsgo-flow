package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oneo-me/nsgo-flow/flowfile"
)

// Tool names; Focus is the one that changes no document.
const (
	ToolGetFlow     = "get_flow"
	ToolAddNodes    = "add_nodes"
	ToolUpdateNode  = "update_node"
	ToolRemoveNodes = "remove_nodes"
	ToolAddEdges    = "add_edges"
	ToolRemoveEdges = "remove_edges"
	ToolSetLayout   = "set_layout"
	ToolReplaceFlow = "replace_flow"
	ToolFocus       = "focus_node"
)

func object(required []string, props map[string]any) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func enum(desc string, values []string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": values}
}

func array(items map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

var (
	nodeProps = map[string]any{
		"id":     str("Stable short ID: letters, digits, _, - or ., such as check_stock."),
		"label":  str("The text on the box, short."),
		"note":   str("An optional second line with detail."),
		"kind":   enum("What the step is; task when omitted.", flowfile.Kinds),
		"status": enum("Progress of the step, when the chart tracks it; skipped fades the step.", flowfile.Statuses),
	}
	edgeProps = map[string]any{
		"from":   str("ID of the node the edge starts at."),
		"to":     str("ID of the node the edge points to."),
		"label":  str("Optional text on the edge, such as a decision's answer."),
		"dashed": map[string]any{"type": "boolean", "description": "Draw the edge dashed, for optional or indirect links."},
	}
)

// Tools are the drawing tools the model gets.
func Tools() []Tool {
	return []Tool{
		{ToolGetFlow, "Return the current flow file's text.", object(nil, map[string]any{})},
		{ToolAddNodes, "Add nodes to the chart, or replace nodes with the same IDs.",
			object([]string{"nodes"}, map[string]any{"nodes": array(object([]string{"id", "label"}, nodeProps))})},
		{ToolUpdateNode, "Change some fields of a node; give rename_to to change its ID, which its edges follow.",
			object([]string{"id"}, map[string]any{"id": str("ID of the node to change."), "rename_to": str("A new ID."),
				"label": nodeProps["label"], "note": nodeProps["note"], "kind": nodeProps["kind"],
				"status": enum("New status; an empty string clears it.", append([]string{""}, flowfile.Statuses...))})},
		{ToolRemoveNodes, "Remove nodes and every edge touching them.",
			object([]string{"ids"}, map[string]any{"ids": array(str("A node ID."))})},
		{ToolAddEdges, "Connect existing nodes, or change the label and dashing of an edge between them.",
			object([]string{"edges"}, map[string]any{"edges": array(object([]string{"from", "to"}, edgeProps))})},
		{ToolRemoveEdges, "Remove edges.",
			object([]string{"edges"}, map[string]any{"edges": array(object([]string{"from", "to"},
				map[string]any{"from": edgeProps["from"], "to": edgeProps["to"]}))})},
		{ToolSetLayout, "Set the chart's title or the direction its edges point.",
			object(nil, map[string]any{"title": str("The chart's title."),
				"direction": enum("down lays the flow out top to bottom, right left to right.", []string{"down", "right"})})},
		{ToolReplaceFlow, "Replace the whole chart with flow file text. Prefer the smaller tools for edits; use this to start over or restructure.",
			object([]string{"text"}, map[string]any{"text": str("A complete flow file.")})},
		{ToolFocus, "Select a node in the user's view and move the camera to it, to point at what you are talking about.",
			object([]string{"id"}, map[string]any{"id": str("ID of the node to show.")})},
	}
}

// Result is what a tool call did: the document after it, whether it
// changed, the node to focus, and the text the model reads back.
type Result struct {
	Doc     flowfile.Doc
	Changed bool
	Focus   string
	Output  string
	Failed  bool
}

// Apply runs a tool call on doc, which it leaves unchanged.
func Apply(doc flowfile.Doc, name, arguments string) Result {
	d := doc.Clone()
	fail := func(format string, args ...any) Result {
		return Result{Doc: doc, Output: "error: " + fmt.Sprintf(format, args...), Failed: true}
	}
	if strings.TrimSpace(arguments) == "" {
		arguments = "{}"
	}
	decode := func(v any) error { return json.Unmarshal([]byte(arguments), v) }
	ok := func(format string, args ...any) Result {
		return Result{Doc: d, Changed: true, Output: fmt.Sprintf(format, args...)}
	}
	switch name {
	case ToolGetFlow:
		return Result{Doc: doc, Output: doc.Format()}
	case ToolAddNodes:
		var a struct{ Nodes []flowfile.Node }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		for _, n := range a.Nodes {
			if err := d.AddNode(n); err != nil {
				return fail("%v", err)
			}
		}
		return ok("added %d nodes", len(a.Nodes))
	case ToolUpdateNode:
		var a struct {
			ID       string  `json:"id"`
			RenameTo string  `json:"rename_to"`
			Label    *string `json:"label"`
			Note     *string `json:"note"`
			Kind     *string `json:"kind"`
			Status   *string `json:"status"`
		}
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		if err := d.UpdateNode(a.ID, flowfile.NodePatch{Label: a.Label, Note: a.Note, Kind: a.Kind, Status: a.Status, Rename: a.RenameTo}); err != nil {
			return fail("%v", err)
		}
		return ok("updated %s", a.ID)
	case ToolRemoveNodes:
		var a struct{ IDs []string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		for _, id := range a.IDs {
			if err := d.RemoveNode(id); err != nil {
				return fail("%v", err)
			}
		}
		return ok("removed %d nodes", len(a.IDs))
	case ToolAddEdges:
		var a struct{ Edges []flowfile.Edge }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		for _, e := range a.Edges {
			if err := d.AddEdge(e); err != nil {
				return fail("%v", err)
			}
		}
		return ok("added %d edges", len(a.Edges))
	case ToolRemoveEdges:
		var a struct{ Edges []flowfile.Edge }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		for _, e := range a.Edges {
			if err := d.RemoveEdge(e.From, e.To); err != nil {
				return fail("%v", err)
			}
		}
		return ok("removed %d edges", len(a.Edges))
	case ToolSetLayout:
		var a struct {
			Title     *string `json:"title"`
			Direction *string `json:"direction"`
		}
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		if a.Title != nil {
			d.Title = *a.Title
		}
		if a.Direction != nil {
			switch dir := flowfile.Direction(*a.Direction); dir {
			case flowfile.Down, flowfile.Right:
				d.Direction = dir
			default:
				return fail("unknown direction %q; use down or right", *a.Direction)
			}
		}
		return ok("layout set")
	case ToolReplaceFlow:
		var a struct{ Text string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		parsed, err := flowfile.Parse(a.Text)
		if err != nil {
			return fail("%v", err)
		}
		d = parsed
		return ok("replaced the chart: %d nodes, %d edges", len(d.Nodes), len(d.Edges))
	case ToolFocus:
		var a struct{ ID string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		if _, found := doc.Node(a.ID); !found {
			return fail("no node %q", a.ID)
		}
		return Result{Doc: doc, Focus: a.ID, Output: "showing " + a.ID}
	}
	return fail("unknown tool %q", name)
}

// SystemPrompt tells the model its job and shows it the chart as it is.
func SystemPrompt(doc flowfile.Doc, selected string) string {
	var b strings.Builder
	b.WriteString(`You are a flow chart designer working inside a flow editor. The user sees the chart on the left and talks with you on the right.

Discuss the process with the user: ask about steps, decisions, actors and edge cases when they are unclear, suggest improvements, and draw as you agree. Change the chart only with the tools, in small steps, and say briefly what you changed. Keep labels short and put detail in notes. Use kind=start and kind=end for entry and exit points, kind=decision for branches, with the answers as edge labels. Reuse existing IDs; never invent nodes the user did not ask for without saying so. Reply in the user's language.

The chart is stored as a flow file:

` + "```" + `
title "Order handling"
direction down                  # or right
node start "Order placed" kind=start
node check "Check stock" note="Reserve every item" status=done
start -> check
check -> ship "in stock"
check ..> start "retry"         # dashed edge
` + "```" + `

Kinds: ` + strings.Join(flowfile.Kinds, ", ") + `. Statuses: ` + strings.Join(flowfile.Statuses, ", ") + `.

The current chart:

` + "```\n")
	text := doc.Format()
	if strings.TrimSpace(text) == "" {
		text = "(empty)\n"
	}
	b.WriteString(text)
	b.WriteString("```\n")
	if selected != "" {
		fmt.Fprintf(&b, "\nThe user has selected node %q.\n", selected)
	}
	return b.String()
}
