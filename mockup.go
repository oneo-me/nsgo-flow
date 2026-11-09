package main

import (
	"strconv"
	"strings"

	"github.com/oneo-me/nsgo/base"
	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-flow/flowfile"
)

// deviceSize is the size a screen of device is laid out at.
func deviceSize(device string) (w, h float32) {
	switch device {
	case "tablet":
		return 820, 1080
	case "desktop":
		return 1200, 760
	}
	return 390, 800
}

// screenView draws a screen's interface with NSGo's controls, room
// wide at most: a sidebar on the left, navigation bars at the top and tab
// bars at the bottom of the rest, which stacks between them. A tablet or
// desktop screen narrower than its device lays itself out anew, at least
// the device's shape, rather than scaling: macOS leaves out what a
// scaled view's frame would have outside the window.
func screenView(cx *core.Scope, s flowfile.Screen, room func() float32) core.Node {
	colors := ui.ColorsOf(cx)
	w, h := deviceSize(s.Device)
	size := func() (float32, float32) {
		if s.Device != "tablet" && s.Device != "desktop" {
			return w, h
		}
		width := max(min(w, room()), w*.6)
		return width, h * width / w
	}
	radius, pad := core.Points(44), core.Points(16)
	switch s.Device {
	case "tablet":
		radius, pad = 28, 24
	case "desktop":
		radius, pad = 12, 24
	}
	var side, top, body, bottom []core.Node
	for _, el := range s.UI {
		switch el.Kind {
		case "sidebar":
			side = append(side, element(cx, el, false))
		case "navbar":
			top = append(top, element(cx, el, false))
		case "tabbar":
			bottom = append(bottom, element(cx, el, false))
		default:
			body = append(body, element(cx, el, false))
		}
	}
	main := append(append(top, core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: core.Pt(0), Overflow: core.Hidden,
		Padding: core.All(pad), Gap: 14}, Children: body}), bottom...)
	content := []core.Node{core.View{Style: core.Style{Direction: core.Row, FlexGrow: 1, MinHeight: core.Pt(0)},
		Children: append(side, core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)}, Children: main})}}
	if s.Device == "" || s.Device == "phone" {
		content = append([]core.Node{statusBar(cx)}, content...)
	}
	return core.View{Style: core.Style{CornerRadius: radius, FlexShrink: 0, BorderCurve: core.BorderCurveContinuous, Overflow: core.Hidden, BorderWidth: 1},
		StyleFunc: func() core.Style {
			w, h := size()
			st := core.Style{Width: core.Pt(core.Points(w)), Height: core.Pt(core.Points(h)), Background: colors.Background(), BorderColor: colors.Border()}
			if s.Device == "tablet" || s.Device == "desktop" {
				// A narrower screen grows to hold what no longer fits.
				st.Height, st.MinHeight = core.Length{}, core.Pt(core.Points(h))
			}
			return st
		},
		Children: content}
}

// statusBar is a phone's clock and indicators.
func statusBar(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 5, Height: core.Pt(44), Padding: core.Symmetric(0, 30)},
		Children: []core.Node{
			ui.Text("9:41", 14, ui.Semibold, colors.Text), ui.Spacer{},
			ui.Icon(lucide.Signal, 14, colors.Text), ui.Icon(lucide.Wifi, 14, colors.Text), ui.Icon(lucide.BatteryFull, 18, colors.Text),
		}}
}

// tone is the color a tone attribute names.
func tone(colors ui.Colors, name string) func() core.Color {
	switch name {
	case "muted":
		return colors.Muted
	case "accent":
		return colors.Accent
	case "info":
		return colors.Info
	case "success":
		return colors.Up
	case "warning":
		return colors.Warn
	case "danger":
		return colors.Danger
	}
	return colors.Text
}

// number reads an element's whole-number attribute, def when unset.
func number(el flowfile.Element, key string, def int) int {
	if n, err := strconv.Atoi(el.Attr(key)); err == nil {
		return n
	}
	return def
}

// selected is the index of an element's selected item.
func selected(el flowfile.Element) int { return max(number(el, "selected", 1), 1) - 1 }

// element draws an element, which in a row shares the row's width when
// it is a wide one. Controls keep their own state, so a design can be
// tried.
func element(cx *core.Scope, el flowfile.Element, inRow bool) core.Node {
	colors := ui.ColorsOf(cx)
	text := func(content string, style core.Style, color func() core.Color) core.Node {
		return core.Text{Content: content, MaxLines: 4, Style: style.Merge(core.Style{FlexShrink: 1}), StyleFunc: ui.Colored(color)}
	}
	grow := core.Style{}
	if inRow {
		grow = core.Style{FlexGrow: 1, FlexBasis: core.Pt(0), MinWidth: core.Pt(0)}
	}
	children := func(row bool) []core.Node {
		out := make([]core.Node, len(el.Children))
		for i, c := range el.Children {
			out[i] = element(cx, c, row)
		}
		return out
	}
	// labeled puts the element's text over control.
	labeled := func(control core.Node) core.Node {
		nodes := []core.Node{}
		if el.Text != "" {
			nodes = append(nodes, text(el.Text, ui.TypeLabel, colors.Text))
		}
		return core.View{Style: core.Style{Gap: 6}.Merge(grow), Children: append(nodes, control)}
	}
	note := func(lines []core.Node) []core.Node {
		if n := el.Attr("note"); n != "" {
			lines = append(lines, text(n, ui.TypeCaption, colors.Muted))
		}
		return lines
	}
	on := core.NewSignal(cx, el.Attr("checked") == "true")
	icon := iconNamed(el.Attr("icon"))
	switch el.Kind {
	case "navbar":
		var leading, trailing []core.Node
		if el.Attr("back") == "true" {
			leading = append(leading, ui.IconButton{Icon: lucide.ChevronLeft, Label: el.Text})
		}
		if a := el.Attr("action"); a != "" {
			trailing = append(trailing, ui.Button{Text: a, Icon: icon, Variant: ui.ButtonGhost, Size: ui.SizeSmall})
		} else if icon != nil {
			trailing = append(trailing, ui.IconButton{Icon: icon, Label: el.Text})
		}
		side := func(nodes []core.Node, end bool) core.Node {
			s := core.Style{Direction: core.Row, Width: core.Pt(96), AlignItems: core.AlignCenter}
			if end {
				s.JustifyContent = core.JustifyEnd
			}
			return core.View{Style: s, Children: nodes}
		}
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Height: core.Pt(48), Padding: core.Symmetric(0, 8),
			Borders: core.Edges{Bottom: 1}},
			StyleFunc: func() core.Style { return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}} },
			Children: []core.Node{side(leading, false), core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter}, Children: []core.Node{
				core.Text{Content: el.Text, MaxLines: 1, Style: ui.TypeTitle, StyleFunc: ui.Colored(colors.Text)}}}, side(trailing, true)}}
	case "heading":
		return text(el.Text, core.Style{FontSize: 24, LineHeight: 30, FontWeight: ui.Bold}, colors.Text)
	case "text":
		return text(el.Text, ui.TypeBody, tone(colors, el.Attr("tone")))
	case "link":
		return core.View{Style: core.Style{Direction: core.Row}, Children: []core.Node{ui.Link{Text: el.Text}}}
	case "button":
		b := ui.Button{Text: el.Text, Icon: icon, Variant: ui.ButtonSecondary}
		switch el.Attr("variant") {
		case "primary":
			b.Variant = ui.ButtonPrimary
		case "outline":
			b.Variant = ui.ButtonOutline
		case "ghost":
			b.Variant = ui.ButtonGhost
		case "danger":
			b.Variant, b.Color = ui.ButtonPrimary, colors.Danger
		}
		switch el.Attr("size") {
		case "small":
			b.Size = ui.SizeSmall
		case "large":
			b.Size = ui.SizeLarge
		}
		if inRow {
			b.Flex = 1
		}
		return b
	case "input":
		in := base.Input{Placeholder: el.Attr("placeholder"), DefaultValue: el.Attr("value"), Label: el.Text}
		switch el.Attr("type") {
		case "password":
			if in.DefaultValue == "" {
				in.DefaultValue = "••••••••"
			}
		case "search":
			if icon == nil {
				icon = lucide.Search
			}
		case "multiline":
			in.Multiline = true
		}
		if icon == nil {
			return labeled(ui.Input{Input: in})
		}
		in.Style = core.Style{FlexGrow: 1, Height: core.Pt(ui.ButtonHeight - 2), Padding: core.Symmetric(0, 2)}
		return labeled(core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4, Padding: core.Edges{Left: 9, Right: 4},
			CornerRadius: ui.RadiusControl, BorderWidth: 1},
			StyleFunc: func() core.Style { return core.Style{Background: colors.Field(), BorderColor: colors.FieldBorder()} },
			Children: []core.Node{ui.Icon(icon, 14, colors.Muted), core.Input{Placeholder: in.Placeholder, DefaultValue: in.DefaultValue, Label: in.Label,
				Style: core.Style{FlexGrow: 1, FontSize: 13, Height: core.Pt(ui.ButtonHeight - 2)},
				StyleFunc: func() core.Style {
					return core.Style{Color: colors.Text(), PlaceholderColor: colors.Faint(), CaretColor: colors.Accent()}
				}}}})
	case "select":
		value := el.Attr("value")
		var options []ui.Option[string]
		if value != "" {
			options = []ui.Option[string]{{Value: value, Label: value}}
		}
		v := core.NewSignal(cx, value)
		return labeled(ui.Select[string]{Options: options, Value: v.Get, OnChange: v.Set, Placeholder: el.Attr("placeholder"), Label: el.Text})
	case "otp":
		return labeled(ui.OtpInput{Length: number(el, "length", 6), Label: el.Text})
	case "tags":
		tags := core.NewSignalFunc(cx, el.Items, func(a, b []string) bool { return false })
		return labeled(ui.TagInput{Tags: tags.Get, OnChange: tags.Set, Label: el.Text})
	case "checkbox":
		return ui.Checkbox{Checked: on.Get, OnChange: on.Set, Text: el.Text}
	case "radio":
		return ui.Radio{Checked: on.Get, OnChange: on.Set, Text: el.Text}
	case "switch":
		row := []core.Node{}
		if icon != nil {
			row = append(row, iconTile(icon, colors.Accent))
		}
		row = append(row, core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 2}, Children: note([]core.Node{text(el.Text, ui.TypeBody, colors.Text)})},
			ui.Switch{Checked: on.Get, OnChange: on.Set, Label: el.Text})
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10}.Merge(grow), Children: row}
	case "slider":
		v := core.NewSignal(cx, float64(number(el, "value", 50)))
		return labeled(ui.Slider{Value: v.Get, OnChange: v.Set, Min: 0, Max: 100, Step: 1, Label: el.Text})
	case "rating":
		v := core.NewSignal(cx, number(el, "value", 0))
		return labeled(ui.Rating{Value: v.Get, OnChange: v.Set, Max: 5, Label: el.Text})
	case "progress":
		value := float32(number(el, "value", 0))
		if el.Attr("type") == "ring" {
			return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}.Merge(grow), Children: []core.Node{
				ui.ProgressRing{Value: func() float32 { return value }, Size: 44, Label: el.Text}, text(el.Text, ui.TypeLabel, colors.Text)}}
		}
		return labeled(ui.Progress{Value: func() float32 { return value }, Label: el.Text})
	case "spinner":
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 8}, Children: []core.Node{
			ui.Spinner{Label: el.Text}, text(el.Text, ui.TypeBody, colors.Muted)}}
	case "segmented", "tabs":
		options := make([]ui.Option[int], len(el.Items))
		for i, item := range el.Items {
			options[i] = ui.Option[int]{Value: i, Label: tabLabel(item)}
			if _, name, ok := strings.Cut(item, ":"); ok {
				options[i].Icon = iconNamed(name)
			}
		}
		v := core.NewSignal(cx, selected(el))
		t := ui.Tabs[int]{Options: options, Value: v.Get, OnChange: v.Set, Equal: el.Kind == "segmented", Label: el.Text}
		if el.Kind == "tabs" {
			t.Variant = ui.TabsUnderline
		}
		return core.View{Style: grow, Children: []core.Node{t}}
	case "tabbar":
		sel := selected(el)
		tabs := make([]core.Node, len(el.Items))
		for i, item := range el.Items {
			color := colors.Muted
			if i == sel {
				color = colors.Accent
			}
			tabs[i] = core.View{Style: core.Style{FlexGrow: 1, FlexBasis: core.Pt(0), AlignItems: core.AlignCenter, Gap: 3}, Children: []core.Node{
				ui.Icon(tabIcon(item), 22, color),
				core.Text{Content: tabLabel(item), MaxLines: 1, Style: core.Style{FontSize: 10.5, FontWeight: ui.Medium}, StyleFunc: ui.Colored(color)}}}
		}
		return core.View{Style: core.Style{Direction: core.Row, Padding: core.Edges{Top: 8, Bottom: 26, Left: 8, Right: 8}, Borders: core.Edges{Top: 1}},
			StyleFunc: func() core.Style {
				return core.Style{Background: colors.Surface(), BorderColors: core.EdgeColors{Top: colors.Line()}}
			},
			Children: tabs}
	case "breadcrumbs":
		return ui.Breadcrumbs{Items: el.Items}
	case "stepper":
		steps := make([]ui.StepperStep, len(el.Items))
		for i, item := range el.Items {
			steps[i] = ui.StepperStep{Label: item}
		}
		current := selected(el)
		return ui.Stepper{Steps: steps, Current: func() int { return current }, Label: el.Text}
	case "pagination":
		page := core.NewSignal(cx, max(number(el, "value", 1), 1))
		total := max(number(el, "total", 10), 1)
		return core.View{Style: core.Style{AlignItems: core.AlignCenter}, Children: []core.Node{
			ui.Pagination{Page: page.Get, TotalPages: func() int { return total }, OnChange: page.Set, Label: el.Text}}}
	case "image":
		nodes := []core.Node{core.View{Style: core.Style{Height: core.Pt(core.Points(number(el, "height", 160))), CornerRadius: ui.RadiusInset + 4,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
			StyleFunc: func() core.Style { return core.Style{Background: colors.Accent().Faded(.1)} },
			Children:  []core.Node{ui.Icon(cmpIcon(icon, lucide.Image), 28, colors.Accent)}}}
		if el.Text != "" {
			nodes = append(nodes, text(el.Text, ui.TypeCaption, colors.Muted))
		}
		return core.View{Style: core.Style{Gap: 6}.Merge(grow), Children: nodes}
	case "avatar":
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}.Merge(grow), Children: []core.Node{
			ui.Avatar{Name: el.Text, Size: 44},
			core.View{Style: core.Style{FlexShrink: 1, Gap: 2}, Children: note([]core.Node{text(el.Text, ui.TypeLabel, colors.Text)})}}}
	case "item":
		row := []core.Node{}
		if icon != nil {
			row = append(row, iconTile(icon, colors.Accent))
		}
		row = append(row, core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 2}, Children: note([]core.Node{text(el.Text, ui.TypeBody, colors.Text)})})
		if t := el.Attr("trailing"); t != "" {
			row = append(row, core.Text{Content: t, MaxLines: 1, Style: ui.TypeBody, StyleFunc: ui.Colored(colors.Muted)})
		}
		row = append(row, ui.Icon(lucide.ChevronRight, 14, colors.Faint))
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, MinHeight: core.Pt(40), Padding: core.Symmetric(4, 0)}.Merge(grow),
			Children: row}
	case "badge":
		variant := ui.BadgeAccent
		switch el.Attr("tone") {
		case "muted":
			variant = ui.BadgeNeutral
		case "info":
			variant = ui.BadgeInfo
		case "success":
			variant = ui.BadgeUp
		case "warning":
			variant = ui.BadgeWarn
		case "danger":
			variant = ui.BadgeDanger
		}
		return core.View{Style: core.Style{Direction: core.Row}, Children: []core.Node{ui.Badge{Text: el.Text, Variant: variant}}}
	case "alert":
		tints := ui.TintsOf(cx)
		color, tint, mark := colors.Info, tints.Info, lucide.Info
		switch el.Attr("tone") {
		case "success":
			color, tint, mark = colors.Up, tints.Up, lucide.CircleCheck
		case "warning":
			color, tint, mark = colors.Warn, tints.Warn, lucide.TriangleAlert
		case "danger":
			color, tint, mark = colors.Danger, tints.Danger, lucide.CircleAlert
		case "accent":
			color, tint, mark = colors.Accent, tints.Accent, lucide.Sparkles
		}
		return core.View{Style: core.Style{Direction: core.Row, Gap: 10, Padding: core.All(12), CornerRadius: ui.RadiusInset, BorderWidth: 1}.Merge(grow),
			StyleFunc: func() core.Style { return core.Style{Background: tint(), BorderColor: color().Faded(.3)} },
			Children: []core.Node{ui.Icon(cmpIcon(icon, mark), 16, color),
				core.View{Style: core.Style{FlexShrink: 1, Gap: 2}, Children: note([]core.Node{text(el.Text, ui.TypeLabel, colors.Text)})}}}
	case "stat":
		head := []core.Node{text(el.Text, ui.TypeCaption, colors.Muted)}
		if icon != nil {
			head = append(head, ui.Spacer{}, ui.Icon(icon, 14, colors.Muted))
		}
		nodes := []core.Node{core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter}, Children: head},
			text(el.Attr("value"), core.Style{FontSize: 24, LineHeight: 30, FontWeight: ui.Semibold}, colors.Text)}
		if n := el.Attr("note"); n != "" {
			color := tone(colors, cmp(el.Attr("tone"), "muted"))
			trend := []core.Node{}
			switch el.Attr("tone") {
			case "success":
				trend = append(trend, ui.Icon(lucide.TrendingUp, 13, color))
			case "danger":
				trend = append(trend, ui.Icon(lucide.TrendingDown, 13, color))
			}
			nodes = append(nodes, core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4},
				Children: append(trend, text(n, ui.TypeCaption, color))})
		}
		return panel(cx, grow, 4, nodes)
	case "chart":
		var values []float64
		top := 0.0
		for _, part := range strings.Split(el.Attr("values"), ",") {
			if v, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil && v >= 0 {
				values = append(values, v)
				top = max(top, v)
			}
		}
		bars := make([]core.Node, len(values))
		for i, v := range values {
			height := core.Points(4)
			if top > 0 {
				height = max(4, core.Points(v/top*112))
			}
			last := i == len(values)-1
			bars[i] = core.View{Style: core.Style{FlexGrow: 1, FlexBasis: core.Pt(0), Height: core.Pt(height), CornerRadius: 4},
				StyleFunc: func() core.Style {
					if last {
						return core.Style{Background: colors.AccentFill()}
					}
					return core.Style{Background: colors.Accent().Faded(.3)}
				}}
		}
		nodes := []core.Node{}
		if el.Text != "" {
			nodes = append(nodes, text(el.Text, ui.TypeLabel, colors.Text))
		}
		nodes = append(nodes, core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignEnd, Gap: 6, Height: core.Pt(112)}, Children: bars})
		return panel(cx, grow, 12, nodes)
	case "skeleton":
		return ui.Skeleton{Lines: number(el, "lines", 3), Label: el.Text}
	case "calendar":
		return core.View{Style: core.Style{AlignItems: core.AlignCenter}.Merge(grow), Children: []core.Node{ui.Calendar{}}}
	case "divider":
		return core.View{Style: core.Style{Height: core.Pt(1)}, StyleFunc: ui.Filled(colors.Line)}
	case "spacer":
		return core.View{Style: core.Style{FlexGrow: 1}}
	case "row":
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifySpaceBetween, Gap: 12}.Merge(grow),
			Children: children(true)}
	case "column":
		return core.View{Style: core.Style{Gap: 12}.Merge(grow), Children: children(false)}
	case "grid":
		columns := max(number(el, "columns", 2), 1)
		cells := children(true)
		var rows []core.Node
		for i := 0; i < len(cells); i += columns {
			row := cells[i:min(i+columns, len(cells))]
			for len(row) < columns {
				row = append(row, core.View{Style: core.Style{FlexGrow: 1, FlexBasis: core.Pt(0)}})
			}
			rows = append(rows, core.View{Style: core.Style{Direction: core.Row, Gap: 12}, Children: row})
		}
		return core.View{Style: core.Style{Gap: 12}.Merge(grow), Children: rows}
	case "card":
		nodes := children(false)
		if el.Text != "" {
			nodes = append([]core.Node{text(el.Text, ui.TypeTitle, colors.Text)}, nodes...)
		}
		return panel(cx, grow, 10, nodes)
	case "section":
		rows := []core.Node{}
		for i, c := range el.Children {
			if i > 0 {
				rows = append(rows, core.View{Style: core.Style{Height: core.Pt(1)}, StyleFunc: ui.Filled(colors.Line)})
			}
			rows = append(rows, core.View{Style: core.Style{Padding: core.Symmetric(8, 0)}, Children: []core.Node{element(cx, c, false)}})
		}
		nodes := []core.Node{}
		if el.Text != "" {
			nodes = append(nodes, core.View{Style: core.Style{Padding: core.Edges{Left: 4}}, Children: []core.Node{
				text(strings.ToUpper(el.Text), core.Style{FontSize: 11, FontWeight: ui.Semibold, LetterSpacing: .4}, colors.Muted)}})
		}
		return core.View{Style: core.Style{Gap: 6}.Merge(grow), Children: append(nodes,
			core.View{Style: core.Style{Padding: core.Symmetric(4, 14), CornerRadius: ui.RadiusInset + 4, BorderWidth: 1},
				StyleFunc: func() core.Style { return core.Style{Background: colors.Surface(), BorderColor: colors.Border()} },
				Children:  rows})}
	case "sidebar":
		return core.View{Style: core.Style{Width: core.Pt(220), FlexShrink: 0, Gap: 2, Padding: core.All(12), Borders: core.Edges{Right: 1}},
			StyleFunc: func() core.Style {
				return core.Style{Background: colors.Surface(), BorderColors: core.EdgeColors{Right: colors.Line()}}
			},
			Children: children(false)}
	case "navitem":
		active := el.Attr("active") == "true"
		b := ui.SidebarButton{Text: el.Text, Icon: icon, Selected: func() bool { return active }}
		t := el.Attr("trailing")
		if t == "" {
			return b
		}
		return core.View{Children: []core.Node{b, core.View{Style: core.Style{Position: core.Absolute, Right: core.Pt(10), Top: core.Pt(0), Bottom: core.Pt(0),
			JustifyContent: core.JustifyCenter, PointerEvents: core.PointerEventsNone}, Children: []core.Node{
			core.Text{Content: t, MaxLines: 1, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)}}}}}
	}
	return core.Group{}
}

// panel is a card holding nodes.
func panel(cx *core.Scope, grow core.Style, gap core.Points, nodes []core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{Style: core.Style{Gap: gap, Padding: core.All(16), CornerRadius: ui.RadiusInset + 4, BorderWidth: 1}.Merge(grow),
		StyleFunc: func() core.Style { return core.Style{Background: colors.Surface(), BorderColor: colors.Border()} },
		Children:  nodes}
}

// iconTile is an icon on a tint of its color.
func iconTile(icon func() *core.SVG, color func() core.Color) core.Node {
	return core.View{Style: core.Style{Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 8, FlexShrink: 0,
		AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
		StyleFunc: func() core.Style { return core.Style{Background: color().Faded(.14)} },
		Children:  []core.Node{ui.Icon(icon, 16, color)}}
}

func cmpIcon(a, b func() *core.SVG) func() *core.SVG {
	if a != nil {
		return a
	}
	return b
}

// tabIcon is a tab's icon: the one its label names, such as "Home:house",
// else a guess from the label.
func tabIcon(label string) func() *core.SVG {
	if _, name, ok := strings.Cut(label, ":"); ok {
		return iconNamed(name)
	}
	l := strings.ToLower(label)
	for _, guess := range []struct {
		icon  func() *core.SVG
		words []string
	}{
		{lucide.House, []string{"home", "首页", "主页"}},
		{lucide.Search, []string{"search", "explore", "discover", "搜索", "发现"}},
		{lucide.ShoppingCart, []string{"cart", "shop", "购物"}},
		{lucide.MessageCircle, []string{"message", "chat", "inbox", "消息", "聊天"}},
		{lucide.Heart, []string{"favorite", "like", "收藏", "喜欢"}},
		{lucide.Bell, []string{"notification", "alert", "通知"}},
		{lucide.Package, []string{"order", "订单"}},
		{lucide.Settings, []string{"setting", "设置"}},
		{lucide.User, []string{"profile", "account", "me", "我"}},
	} {
		for _, w := range guess.words {
			if strings.Contains(l, w) {
				return guess.icon
			}
		}
	}
	return lucide.Circle
}

// tabLabel is a tab's label without the icon it names.
func tabLabel(label string) string {
	text, _, _ := strings.Cut(label, ":")
	return text
}
