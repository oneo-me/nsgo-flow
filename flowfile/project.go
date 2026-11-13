package flowfile

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

// Devices a screen may be drawn on; an empty device is a phone.
var Devices = []string{"phone", "tablet", "desktop"}

// Project is a flow file's content: its screens in order.
type Project struct {
	Title   string
	Screens []Screen
}

// Screen is one interface of the app and the flows that happen on it.
type Screen struct {
	ID, Title string
	Device    string
	UI        []Element
	Flows     []Flow
}

// Flow is a flow chart of a screen; its Doc's title is the flow's.
type Flow struct {
	ID string
	Doc
}

// Element is a piece of a screen's interface: a box holding others, a
// text, an icon, an image or an SVG drawing, its look given by Tailwind
// CSS classes.
type Element struct {
	Kind string
	// Text is a text's content, an icon's name, an image's description
	// or an SVG's document.
	Text     string
	Attrs    map[string]string
	Children []Element
}

// Attr returns the element's attribute key, "" when unset.
func (el Element) Attr(key string) string { return el.Attrs[key] }

// ElementSpec says what an element kind is and takes.
type ElementSpec struct {
	Kind string
	// Holds says the element holds the lines up to its end.
	Holds bool
	// Attrs are the attributes it takes besides class.
	Attrs []string
	// Doc tells a reader, the model included, what the element is.
	Doc string
}

// Elements are the kinds of element an interface is made of: a few
// primitives that classes shape into anything, rather than ready-made
// controls whose look cannot change.
var Elements = []ElementSpec{
	{Kind: "box", Holds: true, Doc: "a rectangle laying out what it holds, a column unless its class has flex-row: with classes, any bar, panel, card, button, field, pill, circle or line"},
	{Kind: "text", Doc: "its text, which wraps; size, weight and color it with classes"},
	{Kind: "icon", Doc: "the Lucide icon its text names, as high as its font size: size it with size-* and color it with text-*"},
	{Kind: "image", Attrs: []string{"src"}, Doc: "a picture its text describes: the image at the URL src, or a placeholder; size it with classes"},
	{Kind: "svg", Doc: "any drawing, such as a chart, a logo or an illustration: its text is an SVG document, quoted with single quotes inside, sized by its class or else its viewBox"},
}

// commonAttrs are the attributes every element takes: the classes that
// shape it, and a user's comment on it, which asks the assistant for a
// change until the assistant resolves it.
var commonAttrs = []string{"class", "comment"}

// attrOrder is the order Format writes attributes in.
var attrOrder = []string{"src", "class", "comment"}

// Spec returns the spec of the element kind.
func Spec(kind string) (ElementSpec, bool) {
	i := slices.IndexFunc(Elements, func(s ElementSpec) bool { return s.Kind == kind })
	if i < 0 {
		return ElementSpec{}, false
	}
	return Elements[i], true
}

func kinds() string {
	names := make([]string, len(Elements))
	for i, s := range Elements {
		names[i] = s.Kind
	}
	return strings.Join(names, ", ")
}

// uiReader reads element lines, holding the containers still open.
type uiReader struct {
	out  []Element
	open []Element
}

func (r *uiReader) add(el Element) {
	if n := len(r.open); n > 0 {
		r.open[n-1].Children = append(r.open[n-1].Children, el)
	} else {
		r.out = append(r.out, el)
	}
}

// line reads an element or an end, and returns what is wrong with it.
func (r *uiReader) line(toks []token) string {
	head := toks[0]
	if !head.quoted && head.text == "end" {
		if len(r.open) == 0 {
			return "end closes no box"
		}
		el := r.open[len(r.open)-1]
		r.open = r.open[:len(r.open)-1]
		r.add(el)
		return ""
	}
	spec, ok := Spec(head.text)
	if head.quoted || !ok {
		return fmt.Sprintf("unknown element %q; use one of %s", head.text, kinds())
	}
	el := Element{Kind: spec.Kind}
	for _, t := range toks[1:] {
		key, value, isAttr := t.attr()
		switch {
		case !isAttr && el.Text == "":
			el.Text = t.text
		case !isAttr:
			return fmt.Sprintf("unexpected %q; quote text with spaces, and write attributes as key=value", t.text)
		case !slices.Contains(commonAttrs, key) && !slices.Contains(spec.Attrs, key):
			return fmt.Sprintf("%s takes no attribute %q; use %s", spec.Kind, key, strings.Join(slices.Concat(spec.Attrs, commonAttrs), ", "))
		default:
			if el.Attrs == nil {
				el.Attrs = map[string]string{}
			}
			el.Attrs[key] = value
		}
	}
	if msg := checkElement(el); msg != "" {
		return msg
	}
	if el.Kind == "icon" {
		el.Text, _ = IconName(el.Text)
	}
	if spec.Holds {
		r.open = append(r.open, el)
	} else {
		r.add(el)
	}
	return ""
}

// checkElement says what is wrong with an element's text.
func checkElement(el Element) string {
	switch el.Kind {
	case "icon":
		if _, ok := IconName(el.Text); !ok {
			msg := fmt.Sprintf("unknown icon %q; use a Lucide icon's name", el.Text)
			if similar := similarIcons(el.Text, 12); len(similar) > 0 {
				msg += " such as " + strings.Join(similar, ", ")
			}
			return msg + ", or draw it with svg"
		}
	case "svg":
		if err := checkSVG(el.Text); err != nil {
			return "svg " + err.Error()
		}
	}
	return ""
}

// checkSVG says why doc is not an SVG document.
func checkSVG(doc string) error {
	dec := xml.NewDecoder(strings.NewReader(doc))
	root := ""
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if start, ok := tok.(xml.StartElement); ok && root == "" {
			root = start.Name.Local
		}
	}
	if root != "svg" {
		return errors.New("needs a document such as <svg viewBox='0 0 24 24'>…</svg>")
	}
	return nil
}

// done returns the elements read, or what is still open.
func (r *uiReader) done() ([]Element, string) {
	if n := len(r.open); n > 0 {
		return nil, fmt.Sprintf("%s has no end", r.open[n-1].Kind)
	}
	return r.out, ""
}

// ParseUI reads the lines of an interface: elements, without a screen.
func ParseUI(src string) ([]Element, error) {
	var r uiReader
	if err := eachLine(src, r.line); err != nil {
		return nil, err
	}
	els, msg := r.done()
	if msg != "" {
		return nil, &Error{strings.Count(src, "\n") + 1, msg}
	}
	return els, nil
}

// ParseProject reads a flow file.
func ParseProject(src string) (Project, error) {
	var p Project
	var ui *uiReader
	flow := -1
	last := func() *Screen { return &p.Screens[len(p.Screens)-1] }
	// endUI ends the interface being read, if one is.
	endUI := func() string {
		if ui == nil {
			return ""
		}
		els, msg := ui.done()
		last().UI, ui = els, nil
		return msg
	}
	addScreen := func(s Screen) string {
		if p.ScreenIndex(s.ID) >= 0 {
			return fmt.Sprintf("a screen %q exists", s.ID)
		}
		p.Screens = append(p.Screens, s)
		flow = -1
		return ""
	}
	addFlow := func(f Flow) string {
		if s, _ := p.FlowIndex(f.ID); s >= 0 {
			return fmt.Sprintf("a flow %q exists", f.ID)
		}
		last().Flows = append(last().Flows, f)
		flow = len(last().Flows) - 1
		return ""
	}
	lines := 0
	err := eachLine(src, func(toks []token) string {
		lines++
		head := toks[0]
		word := ""
		if !head.quoted {
			word = head.text
		}
		switch {
		case word == "title" && len(p.Screens) == 0:
			p.Title = joined(toks[1:])
		case word == "title":
			return "title belongs before the first screen; name screens and flows on their own lines"
		case word == "screen":
			if msg := endUI(); msg != "" {
				return msg
			}
			s, msg := screenLine(toks)
			if msg == "" {
				msg = addScreen(s)
			}
			ui = &uiReader{}
			return msg
		case word == "flow":
			if msg := endUI(); msg != "" {
				return msg
			}
			if len(p.Screens) == 0 {
				addScreen(Screen{ID: "main"})
			}
			if len(toks) < 2 || toks[1].quoted || !ValidID(toks[1].text) {
				return "flow needs an ID of letters, digits, _, - or ."
			}
			f := Flow{ID: toks[1].text}
			for _, t := range toks[2:] {
				if _, _, isAttr := t.attr(); isAttr || f.Title != "" {
					return fmt.Sprintf("unexpected %q; a flow line is flow ID \"Title\"", t.text)
				}
				f.Title = t.text
			}
			return addFlow(f)
		case flow >= 0:
			return last().Flows[flow].line(toks)
		case ui != nil:
			return ui.line(toks)
		default:
			// A file without screens is one flow.
			addScreen(Screen{ID: "main"})
			addFlow(Flow{ID: "main"})
			return last().Flows[flow].line(toks)
		}
		return ""
	})
	if err != nil {
		return Project{}, err
	}
	if msg := endUI(); msg != "" {
		return Project{}, &Error{lines, msg}
	}
	for i := range p.Screens {
		for j := range p.Screens[i].Flows {
			p.Screens[i].Flows[j].fillLabels()
		}
	}
	return p, nil
}

func screenLine(toks []token) (Screen, string) {
	if len(toks) < 2 || toks[1].quoted || !ValidID(toks[1].text) {
		return Screen{}, "screen needs an ID of letters, digits, _, - or ."
	}
	s := Screen{ID: toks[1].text}
	for _, t := range toks[2:] {
		key, value, isAttr := t.attr()
		switch {
		case !isAttr && s.Title == "":
			s.Title = t.text
		case !isAttr:
			return Screen{}, fmt.Sprintf("unexpected %q; write attributes as key=value", t.text)
		case key == "device":
			if !slices.Contains(Devices, value) {
				return Screen{}, fmt.Sprintf("unknown device %q; use one of %s", value, strings.Join(Devices, ", "))
			}
			s.Device = value
		default:
			return Screen{}, fmt.Sprintf("unknown attribute %q; a screen takes device", key)
		}
	}
	return s, ""
}

// ScreenIndex returns the index of the screen with id, or -1.
func (p Project) ScreenIndex(id string) int {
	return slices.IndexFunc(p.Screens, func(s Screen) bool { return s.ID == id })
}

// FlowIndex returns the indexes of the screen and of the flow with id
// on it, or -1 and -1.
func (p Project) FlowIndex(id string) (screen, flow int) {
	for i, s := range p.Screens {
		if j := slices.IndexFunc(s.Flows, func(f Flow) bool { return f.ID == id }); j >= 0 {
			return i, j
		}
	}
	return -1, -1
}

// Clone returns a copy of p that shares nothing with it.
func (p Project) Clone() Project {
	p.Screens = slices.Clone(p.Screens)
	for i := range p.Screens {
		s := &p.Screens[i]
		s.UI = cloneElements(s.UI)
		s.Flows = slices.Clone(s.Flows)
		for j := range s.Flows {
			s.Flows[j].Doc = s.Flows[j].Doc.Clone()
		}
	}
	return p
}

func cloneElements(els []Element) []Element {
	els = slices.Clone(els)
	for i := range els {
		els[i].Attrs = maps.Clone(els[i].Attrs)
		els[i].Children = cloneElements(els[i].Children)
	}
	return els
}

// Format writes p as a flow file.
func (p Project) Format() string {
	var b strings.Builder
	if p.Title != "" {
		fmt.Fprintf(&b, "title %s\n", quote(p.Title))
	}
	for _, s := range p.Screens {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "screen %s", s.ID)
		if s.Title != "" {
			fmt.Fprintf(&b, " %s", quote(s.Title))
		}
		if s.Device != "" && s.Device != "phone" {
			fmt.Fprintf(&b, " device=%s", s.Device)
		}
		b.WriteByte('\n')
		formatUI(&b, s.UI, "  ")
		for j, f := range s.Flows {
			if len(s.UI) > 0 || j > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "  flow %s", f.ID)
			if f.Title != "" {
				fmt.Fprintf(&b, " %s", quote(f.Title))
			}
			b.WriteByte('\n')
			f.body(&b, "    ", false)
		}
	}
	return b.String()
}

// FormatUI writes elements as the lines of an interface.
func FormatUI(els []Element) string {
	var b strings.Builder
	formatUI(&b, els, "")
	return b.String()
}

func formatUI(b *strings.Builder, els []Element, indent string) {
	for _, el := range els {
		b.WriteString(indent + el.Kind)
		if el.Text != "" {
			b.WriteString(" " + quote(el.Text))
		}
		for _, key := range attrOrder {
			if value, ok := el.Attrs[key]; ok {
				fmt.Fprintf(b, " %s=%s", key, quote(value))
			}
		}
		b.WriteByte('\n')
		if spec, _ := Spec(el.Kind); spec.Holds {
			formatUI(b, el.Children, indent+"  ")
			b.WriteString(indent + "end\n")
		}
	}
}
