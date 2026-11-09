package flowfile

import (
	"fmt"
	"slices"
	"strings"
)

// The edits change d in place and say what is wrong instead when they
// cannot; d is unchanged then. Clone a document others hold first.

// AddNode adds n, or fills in the node with its ID.
func (d *Doc) AddNode(n Node) error {
	if err := checkNode(n); err != nil {
		return err
	}
	if n.Label == "" {
		n.Label = n.ID
	}
	if i := d.node(n.ID); i >= 0 {
		d.Nodes[i] = n
		return nil
	}
	d.Nodes = append(d.Nodes, n)
	return nil
}

// NodePatch is what UpdateNode changes: each set field.
type NodePatch struct {
	Label, Note, Kind, Status *string
	// Rename gives the node a new ID, which its edges follow.
	Rename string
}

// UpdateNode changes the node with id.
func (d *Doc) UpdateNode(id string, p NodePatch) error {
	i := d.node(id)
	if i < 0 {
		return fmt.Errorf("no node %q", id)
	}
	n := d.Nodes[i]
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = *src
		}
	}
	set(&n.Label, p.Label)
	set(&n.Note, p.Note)
	set(&n.Kind, p.Kind)
	set(&n.Status, p.Status)
	if n.Label == "" {
		n.Label = n.ID
	}
	if p.Rename != "" && p.Rename != id {
		if d.node(p.Rename) >= 0 {
			return fmt.Errorf("a node %q exists", p.Rename)
		}
		n.ID = p.Rename
	}
	if err := checkNode(n); err != nil {
		return err
	}
	d.Nodes[i] = n
	if n.ID != id {
		for j := range d.Edges {
			if d.Edges[j].From == id {
				d.Edges[j].From = n.ID
			}
			if d.Edges[j].To == id {
				d.Edges[j].To = n.ID
			}
		}
	}
	return nil
}

// RemoveNode removes the node with id and its edges.
func (d *Doc) RemoveNode(id string) error {
	i := d.node(id)
	if i < 0 {
		return fmt.Errorf("no node %q", id)
	}
	d.Nodes = slices.Delete(d.Nodes, i, i+1)
	d.Edges = slices.DeleteFunc(d.Edges, func(e Edge) bool { return e.From == id || e.To == id })
	return nil
}

// AddEdge adds e between nodes that exist, or replaces the label and
// dashing of the edge between them.
func (d *Doc) AddEdge(e Edge) error {
	for _, id := range []string{e.From, e.To} {
		if d.node(id) < 0 {
			return fmt.Errorf("no node %q; add it first", id)
		}
	}
	if e.From == e.To {
		return fmt.Errorf("an edge cannot point from %q to itself", e.From)
	}
	if i := d.edge(e.From, e.To); i >= 0 {
		d.Edges[i] = e
		return nil
	}
	d.Edges = append(d.Edges, e)
	return nil
}

// RemoveEdge removes the edge from one node to another.
func (d *Doc) RemoveEdge(from, to string) error {
	i := d.edge(from, to)
	if i < 0 {
		return fmt.Errorf("no edge from %q to %q", from, to)
	}
	d.Edges = slices.Delete(d.Edges, i, i+1)
	return nil
}

func (d *Doc) edge(from, to string) int {
	return slices.IndexFunc(d.Edges, func(e Edge) bool { return e.From == from && e.To == to })
}

func checkNode(n Node) error {
	if !ValidID(n.ID) {
		return fmt.Errorf("%q is not a node ID; use letters, digits, _, - or .", n.ID)
	}
	if n.Kind != "" && !slices.Contains(Kinds, n.Kind) {
		return fmt.Errorf("unknown kind %q; use one of %s", n.Kind, strings.Join(Kinds, ", "))
	}
	if n.Status != "" && !slices.Contains(Statuses, n.Status) {
		return fmt.Errorf("unknown status %q; use one of %s", n.Status, strings.Join(Statuses, ", "))
	}
	return nil
}
