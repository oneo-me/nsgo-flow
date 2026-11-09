package agent

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oneo-me/nsgo-flow/flowfile"
)

// Tool names; Show is the one that changes no project.
const (
	ToolGetProject     = "get_project"
	ToolAddScreen      = "add_screen"
	ToolSetUI          = "set_ui"
	ToolRemoveScreen   = "remove_screen"
	ToolAddFlow        = "add_flow"
	ToolRemoveFlow     = "remove_flow"
	ToolAddNodes       = "add_nodes"
	ToolUpdateNode     = "update_node"
	ToolRemoveNodes    = "remove_nodes"
	ToolAddEdges       = "add_edges"
	ToolRemoveEdges    = "remove_edges"
	ToolSetLayout      = "set_layout"
	ToolReplaceFlow    = "replace_flow"
	ToolReplaceProject = "replace_project"
	ToolShow           = "show"
)

// Location is what the user sees: a screen, one of its flows, "" for
// none, and a selected node of that flow, or the screen's design when
// Design says.
type Location struct {
	Screen, Flow, Node string
	Design             bool
}

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

// with adds the flow parameter to a flow tool's properties.
func with(props map[string]any) map[string]any {
	props["flow"] = str("ID of the flow to change; the open one when omitted.")
	return props
}

// Tools are the drawing tools the model gets.
func Tools() []Tool {
	ui := str("The screen's interface as element lines, one element a line, such as navbar \"Sign in\" back=true; row, column and card hold the lines up to their end.")
	return []Tool{
		{ToolGetProject, "Return the current flow file's text: every screen with its interface and flows.", object(nil, map[string]any{})},
		{ToolAddScreen, "Add a screen of the app and show it, or change the title, device or interface of the screen with the same ID.",
			object([]string{"id"}, map[string]any{"id": str("Stable short ID: letters, digits, _, - or ., such as sign_in."),
				"title": str("The screen's name, short."), "device": enum("What the screen is drawn on; phone when a new screen omits it.", flowfile.Devices),
				"ui": ui})},
		{ToolSetUI, "Replace a screen's whole interface.",
			object([]string{"ui"}, map[string]any{"screen": str("ID of the screen; the open one when omitted."), "ui": ui})},
		{ToolRemoveScreen, "Remove a screen with its interface and flows.",
			object([]string{"id"}, map[string]any{"id": str("ID of the screen.")})},
		{ToolAddFlow, "Add a flow chart to a screen and show it, such as what happens when a button is pressed, or retitle the flow with the same ID.",
			object([]string{"id"}, map[string]any{"screen": str("ID of the screen; the open one when omitted."),
				"id": str("ID of the flow, unique in the project."), "title": str("The flow's name, short."),
				"direction": enum("down lays the flow out top to bottom, right left to right.", []string{"down", "right"})})},
		{ToolRemoveFlow, "Remove a flow chart.",
			object([]string{"id"}, map[string]any{"id": str("ID of the flow.")})},
		{ToolAddNodes, "Add nodes to a flow, or replace nodes with the same IDs.",
			object([]string{"nodes"}, with(map[string]any{"nodes": array(object([]string{"id", "label"}, nodeProps))}))},
		{ToolUpdateNode, "Change some fields of a node; give rename_to to change its ID, which its edges follow.",
			object([]string{"id"}, with(map[string]any{"id": str("ID of the node to change."), "rename_to": str("A new ID."),
				"label": nodeProps["label"], "note": nodeProps["note"], "kind": nodeProps["kind"],
				"status": enum("New status; an empty string clears it.", append([]string{""}, flowfile.Statuses...))}))},
		{ToolRemoveNodes, "Remove nodes and every edge touching them.",
			object([]string{"ids"}, with(map[string]any{"ids": array(str("A node ID."))}))},
		{ToolAddEdges, "Connect existing nodes, or change the label and dashing of an edge between them.",
			object([]string{"edges"}, with(map[string]any{"edges": array(object([]string{"from", "to"}, edgeProps))}))},
		{ToolRemoveEdges, "Remove edges.",
			object([]string{"edges"}, with(map[string]any{"edges": array(object([]string{"from", "to"},
				map[string]any{"from": edgeProps["from"], "to": edgeProps["to"]}))}))},
		{ToolSetLayout, "Set a flow's title or the direction its edges point, or the project's title.",
			object(nil, with(map[string]any{"title": str("The flow's title."),
				"direction":     enum("down lays the flow out top to bottom, right left to right.", []string{"down", "right"}),
				"project_title": str("The whole project's title, such as the app's name.")}))},
		{ToolReplaceFlow, "Replace a whole flow's nodes and edges with flow text: node and edge lines, without screen or flow lines. Prefer the smaller tools for edits.",
			object([]string{"text"}, with(map[string]any{"text": str("The flow's lines.")}))},
		{ToolReplaceProject, "Replace the whole project with a complete flow file. Use this only to start over or restructure many screens.",
			object([]string{"text"}, map[string]any{"text": str("A complete flow file.")})},
		{ToolShow, "Show a screen's design, or a flow or a node of it, to the user, to point at what you are talking about.",
			object(nil, map[string]any{"screen": str("ID of the screen."), "flow": str("ID of the flow."), "node": str("ID of a node of the flow to select.")})},
	}
}

// Result is what a tool call did: the project after it, whether it
// changed, what to show, and the text the model reads back.
type Result struct {
	Project flowfile.Project
	Changed bool
	// Show is where the user should look, zero to stay.
	Show   Location
	Output string
	Failed bool
}

// Apply runs a tool call on p, open at at, and leaves p unchanged.
func Apply(p flowfile.Project, at Location, name, arguments string) Result {
	d := p.Clone()
	fail := func(format string, args ...any) Result {
		return Result{Project: p, Output: "error: " + fmt.Sprintf(format, args...), Failed: true}
	}
	if strings.TrimSpace(arguments) == "" {
		arguments = "{}"
	}
	decode := func(v any) error { return json.Unmarshal([]byte(arguments), v) }
	var show Location
	ok := func(format string, args ...any) Result {
		return Result{Project: d, Changed: true, Show: show, Output: fmt.Sprintf(format, args...)}
	}
	// screen finds the screen with id, the open one when id is "".
	screen := func(id string) (*flowfile.Screen, error) {
		if id == "" {
			id = at.Screen
		}
		if id == "" {
			return nil, fmt.Errorf("no screen is open; add one with %s", ToolAddScreen)
		}
		i := d.ScreenIndex(id)
		if i < 0 {
			return nil, fmt.Errorf("no screen %q", id)
		}
		return &d.Screens[i], nil
	}
	// flow finds the flow with id, the open one when id is "". An empty
	// project gets a first screen and flow to draw on.
	flow := func(id string) (*flowfile.Flow, error) {
		if id == "" && len(d.Screens) == 0 {
			d.Screens = []flowfile.Screen{{ID: "main", Flows: []flowfile.Flow{{ID: "main"}}}}
			id = "main"
		}
		if id == "" {
			id = at.Flow
		}
		if id == "" {
			s, err := screen("")
			if err != nil {
				return nil, err
			}
			if len(s.Flows) == 0 {
				return nil, fmt.Errorf("screen %q has no flow; add one with %s", s.ID, ToolAddFlow)
			}
			id = s.Flows[0].ID
		}
		i, j := d.FlowIndex(id)
		if i < 0 {
			return nil, fmt.Errorf("no flow %q", id)
		}
		show = Location{Screen: d.Screens[i].ID, Flow: id}
		return &d.Screens[i].Flows[j], nil
	}
	// onFlow decodes the arguments, which name a flow, into v and runs
	// edit on that flow.
	onFlow := func(v any, edit func(f *flowfile.Flow) error) error {
		if err := decode(v); err != nil {
			return err
		}
		var a struct{ Flow string }
		decode(&a)
		f, err := flow(a.Flow)
		if err != nil {
			return err
		}
		return edit(f)
	}
	switch name {
	case ToolGetProject:
		text := p.Format()
		if strings.TrimSpace(text) == "" {
			text = "(empty)\n"
		}
		return Result{Project: p, Output: text}
	case ToolAddScreen:
		var a struct {
			ID, Title string
			Device    *string
			UI        *string
		}
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		if !flowfile.ValidID(a.ID) {
			return fail("%q is not a screen ID; use letters, digits, _, - or .", a.ID)
		}
		if a.Device != nil && *a.Device != "" && !slices.Contains(flowfile.Devices, *a.Device) {
			return fail("unknown device %q; use one of %s", *a.Device, strings.Join(flowfile.Devices, ", "))
		}
		i := d.ScreenIndex(a.ID)
		if i < 0 {
			d.Screens = append(d.Screens, flowfile.Screen{ID: a.ID})
			i = len(d.Screens) - 1
		}
		s := &d.Screens[i]
		if a.Title != "" {
			s.Title = a.Title
		}
		if a.Device != nil {
			s.Device = *a.Device
		}
		if a.UI != nil {
			els, err := flowfile.ParseUI(*a.UI)
			if err != nil {
				return fail("ui %v", err)
			}
			s.UI = els
		}
		show = Location{Screen: s.ID, Design: true}
		return ok("screen %s has %d elements and %d flows", s.ID, len(s.UI), len(s.Flows))
	case ToolSetUI:
		var a struct{ Screen, UI string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		s, err := screen(a.Screen)
		if err != nil {
			return fail("%v", err)
		}
		els, err := flowfile.ParseUI(a.UI)
		if err != nil {
			return fail("ui %v", err)
		}
		s.UI = els
		show = Location{Screen: s.ID, Design: true}
		return ok("screen %s has %d elements", s.ID, len(els))
	case ToolRemoveScreen:
		var a struct{ ID string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		i := d.ScreenIndex(a.ID)
		if i < 0 {
			return fail("no screen %q", a.ID)
		}
		d.Screens = slices.Delete(d.Screens, i, i+1)
		return ok("removed screen %s", a.ID)
	case ToolAddFlow:
		var a struct {
			Screen, ID, Title string
			Direction         string
		}
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		if !flowfile.ValidID(a.ID) {
			return fail("%q is not a flow ID; use letters, digits, _, - or .", a.ID)
		}
		if a.Direction != "" && a.Direction != string(flowfile.Down) && a.Direction != string(flowfile.Right) {
			return fail("unknown direction %q; use down or right", a.Direction)
		}
		var f *flowfile.Flow
		if i, j := d.FlowIndex(a.ID); i >= 0 {
			if a.Screen != "" && d.Screens[i].ID != a.Screen {
				return fail("flow %q is on screen %q; flow IDs are unique in the project", a.ID, d.Screens[i].ID)
			}
			f = &d.Screens[i].Flows[j]
			show.Screen = d.Screens[i].ID
		} else {
			if a.Screen == "" && len(d.Screens) == 0 {
				d.Screens = []flowfile.Screen{{ID: "main"}}
				a.Screen = "main"
			}
			s, err := screen(a.Screen)
			if err != nil {
				return fail("%v", err)
			}
			s.Flows = append(s.Flows, flowfile.Flow{ID: a.ID})
			f = &s.Flows[len(s.Flows)-1]
			show.Screen = s.ID
		}
		if a.Title != "" {
			f.Title = a.Title
		}
		if a.Direction != "" {
			f.Direction = flowfile.Direction(a.Direction)
		}
		show.Flow = f.ID
		return ok("flow %s is on screen %s", f.ID, show.Screen)
	case ToolRemoveFlow:
		var a struct{ ID string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		i, j := d.FlowIndex(a.ID)
		if i < 0 {
			return fail("no flow %q", a.ID)
		}
		d.Screens[i].Flows = slices.Delete(d.Screens[i].Flows, j, j+1)
		return ok("removed flow %s", a.ID)
	case ToolAddNodes:
		var a struct{ Nodes []flowfile.Node }
		err := onFlow(&a, func(f *flowfile.Flow) error {
			for _, n := range a.Nodes {
				if err := f.AddNode(n); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fail("%v", err)
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
		err := onFlow(&a, func(f *flowfile.Flow) error {
			return f.UpdateNode(a.ID, flowfile.NodePatch{Label: a.Label, Note: a.Note, Kind: a.Kind, Status: a.Status, Rename: a.RenameTo})
		})
		if err != nil {
			return fail("%v", err)
		}
		return ok("updated %s", a.ID)
	case ToolRemoveNodes:
		var a struct{ IDs []string }
		err := onFlow(&a, func(f *flowfile.Flow) error {
			for _, id := range a.IDs {
				if err := f.RemoveNode(id); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fail("%v", err)
		}
		return ok("removed %d nodes", len(a.IDs))
	case ToolAddEdges:
		var a struct{ Edges []flowfile.Edge }
		err := onFlow(&a, func(f *flowfile.Flow) error {
			for _, e := range a.Edges {
				if err := f.AddEdge(e); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fail("%v", err)
		}
		return ok("added %d edges", len(a.Edges))
	case ToolRemoveEdges:
		var a struct{ Edges []flowfile.Edge }
		err := onFlow(&a, func(f *flowfile.Flow) error {
			for _, e := range a.Edges {
				if err := f.RemoveEdge(e.From, e.To); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fail("%v", err)
		}
		return ok("removed %d edges", len(a.Edges))
	case ToolSetLayout:
		var a struct {
			Flow         string
			Title        *string `json:"title"`
			Direction    *string `json:"direction"`
			ProjectTitle *string `json:"project_title"`
		}
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		if a.ProjectTitle != nil {
			d.Title = *a.ProjectTitle
		}
		if a.Title != nil || a.Direction != nil {
			f, err := flow(a.Flow)
			if err != nil {
				return fail("%v", err)
			}
			if a.Title != nil {
				f.Title = *a.Title
			}
			if a.Direction != nil {
				switch dir := flowfile.Direction(*a.Direction); dir {
				case flowfile.Down, flowfile.Right:
					f.Direction = dir
				default:
					return fail("unknown direction %q; use down or right", *a.Direction)
				}
			}
		}
		return ok("layout set")
	case ToolReplaceFlow:
		var a struct{ Text string }
		var f *flowfile.Flow
		err := onFlow(&a, func(target *flowfile.Flow) error {
			parsed, err := flowfile.Parse(a.Text)
			if err != nil {
				return err
			}
			title := target.Title
			if parsed.Title != "" {
				title = parsed.Title
			}
			target.Doc, target.Title = parsed, title
			f = target
			return nil
		})
		if err != nil {
			return fail("%v", err)
		}
		return ok("replaced flow %s: %d nodes, %d edges", f.ID, len(f.Nodes), len(f.Edges))
	case ToolReplaceProject:
		var a struct{ Text string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		parsed, err := flowfile.ParseProject(a.Text)
		if err != nil {
			return fail("%v", err)
		}
		d = parsed
		if len(d.Screens) > 0 {
			show.Screen = d.Screens[0].ID
			if len(d.Screens[0].Flows) > 0 {
				show.Flow = d.Screens[0].Flows[0].ID
			}
		}
		return ok("replaced the project: %d screens", len(d.Screens))
	case ToolShow:
		var a struct{ Screen, Flow, Node string }
		if err := decode(&a); err != nil {
			return fail("%v", err)
		}
		if a.Flow != "" || a.Node != "" {
			f, err := flow(a.Flow)
			if err != nil {
				return fail("%v", err)
			}
			if a.Screen != "" && a.Screen != show.Screen {
				return fail("flow %q is on screen %q", f.ID, show.Screen)
			}
			if a.Node != "" {
				if _, found := f.Node(a.Node); !found {
					return fail("no node %q in flow %q", a.Node, f.ID)
				}
				show.Node = a.Node
			}
		} else {
			s, err := screen(a.Screen)
			if err != nil {
				return fail("%v", err)
			}
			show = Location{Screen: s.ID, Design: true}
		}
		return Result{Project: p, Show: show, Output: strings.Join(slices.DeleteFunc([]string{"showing", show.Screen, show.Flow, show.Node},
			func(s string) bool { return s == "" }), " ")}
	}
	return fail("unknown tool %q", name)
}

// SystemPrompt tells the model its job and shows it the project as it
// is, where the user is looking, and the name of the project folder it
// may read, "" for none.
func SystemPrompt(p flowfile.Project, at Location, folder string) string {
	var b strings.Builder
	b.WriteString(`You are an app designer working inside a flow editor. The user sees a list of the app's screens on the left, the open screen in the middle with a tab for its design and one for each of its flow charts, and talks with you on the right.

Discuss the app with the user: ask about its screens, what each shows, the steps, decisions, actors and edge cases of what happens on them when they are unclear, suggest improvements, and draw as you agree. Each screen has a design, drawn with the controls of NSGo, a desktop and mobile UI toolkit, and one or more flow charts, such as what happens when the user presses its main button. Make designs look like a finished product: use icons, sections, cards, badges, stats and the right controls rather than plain text, and fill them with realistic content. When the user only wants a flow chart, draw flows and leave the design empty. Change the project only with the tools, in small steps, and say briefly what you changed. Keep labels short and put detail in notes. Use kind=start and kind=end for entry and exit points, kind=decision for branches, with the answers as edge labels. Reuse existing IDs; never invent screens or nodes the user did not ask for without saying so. Reply in the user's language, and write the interface's texts in it too.

The project is stored as a flow file:

` + "```" + `
title "Shop"
screen signin "Sign in"
  navbar "Sign in" back=true
  input "Email" placeholder="you@example.com"
  row
    checkbox "Remember me" checked=true
    link "Forgot password?"
  end
  button "Sign in" variant=primary
  flow login "Signing in"
    node start "Tap Sign in" kind=start
    node check "Check password" kind=decision note="Lock after 5 tries"
    start -> check
    check -> home "valid"
    check ..> start "invalid"   # dashed edge
` + "```" + `

Interface elements, a line each, with their attributes:
`)
	for _, spec := range flowfile.Elements {
		fmt.Fprintf(&b, "- %s: %s", spec.Kind, spec.Doc)
		var attrs []string
		for _, key := range slices.Sorted(maps.Keys(spec.Attrs)) {
			switch values := spec.Attrs[key]; {
			case values == nil:
				attrs = append(attrs, key+"=\"text\"")
			case slices.Equal(values, flowfile.Number):
				attrs = append(attrs, key+"=number")
			default:
				attrs = append(attrs, key+"="+strings.Join(values, "|"))
			}
		}
		if spec.Items {
			attrs = append([]string{`"label" "label" …`}, attrs...)
		}
		if _, ok := spec.Attrs["icon"]; ok {
			attrs = slices.DeleteFunc(attrs, func(a string) bool { return strings.HasPrefix(a, "icon=") })
			attrs = append(attrs, "icon=name")
		}
		if len(attrs) > 0 {
			fmt.Fprintf(&b, " (%s)", strings.Join(attrs, ", "))
		}
		b.WriteByte('\n')
	}
	b.WriteString(`
Icon names: ` + strings.Join(flowfile.Icons, ", ") + `. A tab of tabs, segmented or tabbar names its icon after a colon, such as "Home:house".

Node kinds: ` + strings.Join(flowfile.Kinds, ", ") + `. Statuses: ` + strings.Join(flowfile.Statuses, ", ") + `. Devices: ` + strings.Join(flowfile.Devices, ", ") + `.

The current project:

` + "```\n")
	text := p.Format()
	if strings.TrimSpace(text) == "" {
		text = "(empty)\n"
	}
	b.WriteString(text)
	b.WriteString("```\n")
	if at.Screen != "" {
		fmt.Fprintf(&b, "\nThe user has screen %q open", at.Screen)
		switch {
		case at.Design:
			b.WriteString(" on its design")
		case at.Flow != "":
			fmt.Fprintf(&b, " on flow %q", at.Flow)
		}
		b.WriteString("; tools that take a screen or flow use the open ones when you omit them.\n")
	}
	if at.Node != "" {
		fmt.Fprintf(&b, "The user has selected node %q.\n", at.Node)
	}
	if folder != "" {
		fmt.Fprintf(&b, `
The user attached the project folder %q. When the user asks about how this project works, study it before drawing: list folders, find and search files, and read the code, configuration and documents that matter, then draw what the code actually does, naming the files behind a step in its note when that helps. Paths are relative to the folder's root, and you can only read it. Read only what the question needs, and do not quote long passages back.
`, folder)
	}
	return b.String()
}
