package flowfile

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
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

// Element is a piece of a screen's interface.
type Element struct {
	Kind string
	Text string
	// Items are the labels of an element that takes several, such as a
	// tab bar's.
	Items    []string
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
	// Items says the element takes several labels instead of one text.
	Items bool
	// Attrs are the attributes and the values each takes: nil any
	// text, Number a whole number.
	Attrs map[string][]string
	// Doc tells a reader, the model included, what the element is.
	Doc string
}

// Number is the values of an attribute that takes a whole number.
var Number = []string{"#"}

var (
	bools  = []string{"true", "false"}
	tones  = []string{"muted", "accent", "info", "success", "warning", "danger"}
	toned  = map[string][]string{"tone": tones}
	picked = map[string][]string{"selected": Number}
)

// with adds the icon attribute to attrs.
func with(attrs map[string][]string) map[string][]string {
	attrs["icon"] = Icons
	return attrs
}

// Elements are the kinds of element an interface is made of.
var Elements = []ElementSpec{
	{Kind: "navbar", Attrs: with(map[string][]string{"back": bools, "action": nil}), Doc: "the bar at the top with the screen's title, a back arrow and an action on the right, or the action's icon"},
	{Kind: "heading", Doc: "a large title"},
	{Kind: "text", Attrs: toned, Doc: "a paragraph"},
	{Kind: "link", Doc: "a text link"},
	{Kind: "button", Attrs: with(map[string][]string{"variant": {"primary", "secondary", "outline", "ghost", "danger"}, "size": {"small", "large"}}), Doc: "a button; secondary when no variant is set"},
	{Kind: "input", Attrs: with(map[string][]string{"placeholder": nil, "value": nil, "type": {"text", "password", "search", "multiline"}}), Doc: "a text field under its label"},
	{Kind: "select", Attrs: map[string][]string{"value": nil, "placeholder": nil}, Doc: "a field that picks one of several values, under its label"},
	{Kind: "otp", Attrs: map[string][]string{"length": Number}, Doc: "a one-time code's boxes, under its label"},
	{Kind: "tags", Items: true, Doc: "a field of removable tags"},
	{Kind: "checkbox", Attrs: map[string][]string{"checked": bools}, Doc: "a check box and its label"},
	{Kind: "radio", Attrs: map[string][]string{"checked": bools}, Doc: "a radio button and its label"},
	{Kind: "switch", Attrs: with(map[string][]string{"checked": bools, "note": nil}), Doc: "a setting's label, note and on/off switch"},
	{Kind: "slider", Attrs: map[string][]string{"value": Number}, Doc: "a slider under its label; value runs from 0 to 100"},
	{Kind: "rating", Attrs: map[string][]string{"value": Number}, Doc: "stars, value of 5"},
	{Kind: "progress", Attrs: map[string][]string{"value": Number, "type": {"bar", "ring"}}, Doc: "a progress bar or ring under its label; value runs from 0 to 100"},
	{Kind: "spinner", Doc: "a busy indicator with an optional label"},
	{Kind: "segmented", Items: true, Attrs: picked, Doc: "a segmented control of its labels; selected counts from 1"},
	{Kind: "tabs", Items: true, Attrs: picked, Doc: "tabs with an underline over the content below them; selected counts from 1"},
	{Kind: "tabbar", Items: true, Attrs: picked, Doc: "the tab bar at the bottom of a phone app's screens; selected counts from 1"},
	{Kind: "breadcrumbs", Items: true, Doc: "the path to the page, last item current"},
	{Kind: "stepper", Items: true, Attrs: picked, Doc: "the steps of a process, the selected one current; selected counts from 1"},
	{Kind: "pagination", Attrs: map[string][]string{"value": Number, "total": Number}, Doc: "page value of total pages"},
	{Kind: "image", Attrs: map[string][]string{"height": Number}, Doc: "a placeholder for a picture, with an optional caption"},
	{Kind: "avatar", Attrs: map[string][]string{"note": nil}, Doc: "a person's round picture, name and note"},
	{Kind: "item", Attrs: with(map[string][]string{"note": nil, "trailing": nil}), Doc: "a list row: an icon, a title, a note below it and a value on the right"},
	{Kind: "badge", Attrs: toned, Doc: "a small label in a pill"},
	{Kind: "alert", Attrs: with(map[string][]string{"tone": tones, "note": nil}), Doc: "a callout with a title and a note"},
	{Kind: "stat", Attrs: with(map[string][]string{"value": nil, "note": nil, "tone": tones}), Doc: "a dashboard figure: its label, a large value and a note such as a trend in the tone"},
	{Kind: "chart", Attrs: map[string][]string{"values": nil}, Doc: "a bar chart titled by its text; values are numbers separated by commas"},
	{Kind: "skeleton", Attrs: map[string][]string{"lines": Number}, Doc: "content still loading"},
	{Kind: "calendar", Doc: "a month calendar"},
	{Kind: "divider", Doc: "a line across"},
	{Kind: "spacer", Doc: "empty room that pushes what follows to the end"},
	{Kind: "row", Holds: true, Doc: "lays what it holds out side by side"},
	{Kind: "column", Holds: true, Doc: "stacks what it holds"},
	{Kind: "grid", Holds: true, Attrs: map[string][]string{"columns": Number}, Doc: "lays what it holds out in columns, 2 when unset"},
	{Kind: "card", Holds: true, Doc: "a raised panel with an optional title that stacks what it holds"},
	{Kind: "section", Holds: true, Doc: "a titled group of rows, such as a settings page's items and switches"},
	{Kind: "sidebar", Holds: true, Doc: "a desktop or tablet app's navigation column on the left of the rest of the screen"},
	{Kind: "navitem", Attrs: with(map[string][]string{"active": bools, "trailing": nil}), Doc: "a destination in a sidebar"},
}

// attrOrder is the order Format writes attributes in.
var attrOrder = []string{"back", "action", "icon", "variant", "size", "type", "placeholder", "value", "total", "length", "lines", "columns", "values",
	"checked", "active", "selected", "height", "tone", "note", "trailing"}

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
			return "end closes no row, column or card"
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
		case !isAttr && spec.Items:
			el.Items = append(el.Items, t.text)
		case !isAttr && el.Text == "":
			el.Text = t.text
		case !isAttr:
			return fmt.Sprintf("unexpected %q; write attributes as key=value", t.text)
		default:
			if msg := checkAttr(spec, key, value); msg != "" {
				return msg
			}
			if el.Attrs == nil {
				el.Attrs = map[string]string{}
			}
			el.Attrs[key] = value
		}
	}
	if spec.Holds {
		r.open = append(r.open, el)
	} else {
		r.add(el)
	}
	return ""
}

// done returns the elements read, or what is still open.
func (r *uiReader) done() ([]Element, string) {
	if n := len(r.open); n > 0 {
		return nil, fmt.Sprintf("%s has no end", r.open[n-1].Kind)
	}
	return r.out, ""
}

func checkAttr(spec ElementSpec, key, value string) string {
	values, ok := spec.Attrs[key]
	if !ok {
		var keys []string
		for _, k := range attrOrder {
			if _, ok := spec.Attrs[k]; ok {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			return fmt.Sprintf("%s takes no attributes", spec.Kind)
		}
		return fmt.Sprintf("%s takes no attribute %q; use %s", spec.Kind, key, strings.Join(keys, ", "))
	}
	switch {
	case values == nil:
	case slices.Equal(values, Number):
		if n, err := strconv.Atoi(value); err != nil || n < 0 {
			return fmt.Sprintf("%s=%q is not a whole number", key, value)
		}
	case !slices.Contains(values, value):
		return fmt.Sprintf("unknown %s %q; use one of %s", key, value, strings.Join(values, ", "))
	}
	return ""
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
		els[i].Items = slices.Clone(els[i].Items)
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
		for _, item := range el.Items {
			b.WriteString(" " + quote(item))
		}
		spec, _ := Spec(el.Kind)
		for _, key := range attrOrder {
			value, ok := el.Attrs[key]
			if !ok {
				continue
			}
			if spec.Attrs[key] == nil {
				value = quote(value)
			}
			fmt.Fprintf(b, " %s=%s", key, value)
		}
		b.WriteByte('\n')
		if spec.Holds {
			formatUI(b, el.Children, indent+"  ")
			b.WriteString(indent + "end\n")
		}
	}
}

// Icons are the icon names an element's icon attribute takes, as Lucide
// names them.
var Icons = []string{"house", "search", "shopping-cart", "shopping-bag", "user", "users", "settings", "message-circle", "mail", "heart", "bell", "package", "star", "bookmark", "calendar", "clock", "camera", "image", "map-pin", "phone", "credit-card", "wallet", "gift", "tag", "truck", "file-text", "folder", "download", "upload", "share-2", "plus", "minus", "x", "check", "pencil", "trash", "sliders-horizontal", "list", "grid-2x2", "layout-dashboard", "chart-bar", "chart-line", "chart-pie", "activity", "trending-up", "trending-down", "lock", "key", "log-out", "log-in", "eye", "eye-off", "info", "circle-alert", "triangle-alert", "circle-check", "menu", "ellipsis", "arrow-right", "arrow-left", "chevron-right", "chevron-left", "refresh-cw", "globe", "sun", "moon", "wifi", "music", "play", "pause", "video", "mic", "headphones", "book", "graduation-cap", "briefcase", "building", "store", "coffee", "utensils", "car", "plane", "bike", "dumbbell", "inbox", "send", "archive", "paperclip", "link", "qr-code", "scan", "shield", "zap", "sparkles", "flame", "award", "trophy", "target", "compass", "navigation", "home", "cart", "profile"}
