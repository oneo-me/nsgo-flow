// Package flowfile reads and writes flow files: plain text documents that
// describe a directed flow chart line by line.
//
//	# A comment runs to the end of the line.
//	title "Order handling"
//	direction down                 # or right
//
//	node start "Order placed" kind=start
//	node check "Check stock" note="Reserve every item" status=done
//	node ship "Ship" kind=task status=doing
//
//	start -> check
//	check -> ship "in stock"
//	check ..> start "out of stock"  # a dashed edge
//	a -> b -> c                     # a chain of edges
//
// An edge may name nodes no line declares; they take their ID as their
// label. Format writes a document back in this canonical form, without
// the comments.
package flowfile

import (
	"fmt"
	"slices"
	"strings"
)

// Direction is the way edges point.
type Direction string

const (
	Down  Direction = "down"
	Right Direction = "right"
)

// Kinds a node may be; an empty kind is a task.
var Kinds = []string{"task", "start", "end", "decision", "data", "note"}

// Statuses a node may have; an empty status is none, and skipped fades it.
var Statuses = []string{"todo", "doing", "done", "blocked", "skipped"}

// Node is one box of the chart.
type Node struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Note   string `json:"note,omitempty"`
	Kind   string `json:"kind,omitempty"`
	Status string `json:"status,omitempty"`
}

// Edge points from one node to another.
type Edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label,omitempty"`
	Dashed bool   `json:"dashed,omitempty"`
}

// Doc is a flow file's content.
type Doc struct {
	Title     string
	Direction Direction
	Nodes     []Node
	Edges     []Edge
}

// Error is a line Parse could not read.
type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// ValidID reports whether id can name a node: letters, digits, _, - and .
func ValidID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !(r == '_' || r == '-' || r == '.' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 0x7f) {
			return false
		}
	}
	return true
}

// Parse reads a flow file.
func Parse(src string) (Doc, error) {
	var d Doc
	for i, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		n := i + 1
		toks, err := tokenize(line)
		if err != nil {
			return Doc{}, &Error{n, err.Error()}
		}
		if len(toks) == 0 {
			continue
		}
		fail := func(format string, args ...any) (Doc, error) { return Doc{}, &Error{n, fmt.Sprintf(format, args...)} }
		head := toks[0]
		switch {
		case !head.quoted && head.text == "title":
			parts := make([]string, 0, len(toks)-1)
			for _, t := range toks[1:] {
				parts = append(parts, t.text)
			}
			d.Title = strings.Join(parts, " ")
		case !head.quoted && head.text == "direction":
			if len(toks) != 2 {
				return fail("direction takes down or right")
			}
			switch dir := Direction(toks[1].text); dir {
			case Down, Right:
				d.Direction = dir
			default:
				return fail("unknown direction %q; use down or right", toks[1].text)
			}
		case !head.quoted && head.text == "node":
			if len(toks) < 2 || toks[1].quoted || !ValidID(toks[1].text) {
				return fail("node needs an ID of letters, digits, _, - or .")
			}
			node := Node{ID: toks[1].text}
			for _, t := range toks[2:] {
				key, value, ok := t.attr()
				switch {
				case !ok && node.Label == "":
					node.Label = t.text
				case !ok:
					return fail("unexpected %q; write attributes as key=value", t.text)
				case key == "label":
					node.Label = value
				case key == "note":
					node.Note = value
				case key == "kind":
					if !slices.Contains(Kinds, value) {
						return fail("unknown kind %q; use one of %s", value, strings.Join(Kinds, ", "))
					}
					node.Kind = value
				case key == "status":
					if value != "" && !slices.Contains(Statuses, value) {
						return fail("unknown status %q; use one of %s", value, strings.Join(Statuses, ", "))
					}
					node.Status = value
				default:
					return fail("unknown attribute %q", key)
				}
			}
			if i := d.node(node.ID); i >= 0 {
				// A declaration after an edge named the node fills it in.
				d.Nodes[i] = node
			} else {
				d.Nodes = append(d.Nodes, node)
			}
		default:
			if err := d.parseEdges(toks); err != "" {
				return fail("%s", err)
			}
		}
	}
	for i := range d.Nodes {
		if d.Nodes[i].Label == "" {
			d.Nodes[i].Label = d.Nodes[i].ID
		}
	}
	return d, nil
}

// parseEdges reads a chain of edges, a -> b ..> c "label", and returns
// what is wrong with it.
func (d *Doc) parseEdges(toks []token) string {
	label := ""
	if last := toks[len(toks)-1]; last.quoted && len(toks) > 3 {
		label, toks = last.text, toks[:len(toks)-1]
	}
	if len(toks) < 3 || len(toks)%2 == 0 {
		return fmt.Sprintf("unexpected %q; a line is a title, a direction, a node or edges such as a -> b", toks[0].text)
	}
	for i := 0; i+2 < len(toks); i += 2 {
		from, arrow, to := toks[i], toks[i+1], toks[i+2]
		if arrow.quoted || arrow.text != "->" && arrow.text != "..>" {
			return fmt.Sprintf("expected -> or ..> after %q", from.text)
		}
		for _, t := range []token{from, to} {
			if t.quoted || !ValidID(t.text) {
				return fmt.Sprintf("%q is not a node ID", t.text)
			}
			if d.node(t.text) < 0 {
				d.Nodes = append(d.Nodes, Node{ID: t.text})
			}
		}
		e := Edge{From: from.text, To: to.text, Dashed: arrow.text == "..>"}
		if i+3 == len(toks) {
			e.Label = label
		}
		d.Edges = append(d.Edges, e)
	}
	return ""
}

func (d *Doc) node(id string) int {
	return slices.IndexFunc(d.Nodes, func(n Node) bool { return n.ID == id })
}

// Node returns the node with id.
func (d Doc) Node(id string) (Node, bool) {
	if i := d.node(id); i >= 0 {
		return d.Nodes[i], true
	}
	return Node{}, false
}

// Clone returns a copy of d that shares nothing with it.
func (d Doc) Clone() Doc {
	d.Nodes, d.Edges = slices.Clone(d.Nodes), slices.Clone(d.Edges)
	return d
}

// Format writes d as a flow file.
func (d Doc) Format() string {
	var b strings.Builder
	if d.Title != "" {
		fmt.Fprintf(&b, "title %s\n", quote(d.Title))
	}
	if d.Direction != "" && d.Direction != Down {
		fmt.Fprintf(&b, "direction %s\n", d.Direction)
	}
	if b.Len() > 0 && len(d.Nodes) > 0 {
		b.WriteByte('\n')
	}
	for _, n := range d.Nodes {
		fmt.Fprintf(&b, "node %s %s", n.ID, quote(n.Label))
		if n.Kind != "" && n.Kind != "task" {
			fmt.Fprintf(&b, " kind=%s", n.Kind)
		}
		if n.Status != "" {
			fmt.Fprintf(&b, " status=%s", n.Status)
		}
		if n.Note != "" {
			fmt.Fprintf(&b, " note=%s", quote(n.Note))
		}
		b.WriteByte('\n')
	}
	if len(d.Nodes) > 0 && len(d.Edges) > 0 {
		b.WriteByte('\n')
	}
	for _, e := range d.Edges {
		arrow := "->"
		if e.Dashed {
			arrow = "..>"
		}
		fmt.Fprintf(&b, "%s %s %s", e.From, arrow, e.To)
		if e.Label != "" {
			fmt.Fprintf(&b, " %s", quote(e.Label))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// quote writes s as a quoted string.
func quote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}

// token is a word of a line: quoted when it was a quoted string as a
// whole, raw holding a key=value attribute's parts as written.
type token struct {
	text   string
	quoted bool
	// eq is where an unquoted = split the token, -1 when none did.
	eq int
}

// attr splits an attribute token into its key and value.
func (t token) attr() (key, value string, ok bool) {
	if t.quoted || t.eq < 0 {
		return "", "", false
	}
	return t.text[:t.eq], t.text[t.eq+1:], true
}

// tokenize splits a line into words and quoted strings, dropping a
// comment.
func tokenize(line string) ([]token, error) {
	var toks []token
	rs := []rune(line)
	for i := 0; i < len(rs); {
		if rs[i] == ' ' || rs[i] == '\t' {
			i++
			continue
		}
		if rs[i] == '#' {
			break
		}
		var b strings.Builder
		t := token{eq: -1}
		whole := rs[i] == '"'
		for i < len(rs) && rs[i] != ' ' && rs[i] != '\t' {
			switch c := rs[i]; {
			case c == '"':
				i++
				closed := false
				for i < len(rs) {
					c := rs[i]
					i++
					if c == '"' {
						closed = true
						break
					}
					if c == '\\' && i < len(rs) {
						e := rs[i]
						i++
						switch e {
						case 'n':
							b.WriteRune('\n')
						case 't':
							b.WriteRune('\t')
						default:
							b.WriteRune(e)
						}
						continue
					}
					b.WriteRune(c)
				}
				if !closed {
					return nil, fmt.Errorf("unclosed quote")
				}
			case c == '=' && t.eq < 0 && !whole:
				t.eq = b.Len()
				b.WriteRune(c)
				i++
			default:
				b.WriteRune(c)
				i++
			}
		}
		t.text, t.quoted = b.String(), whole
		toks = append(toks, t)
	}
	return toks, nil
}
