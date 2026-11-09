package flowfile

import (
	"strings"
	"testing"
)

const app = `title "Shop"

screen signin "Sign in"
  navbar "Sign in" back=true
  input "Email" placeholder="you@example.com"
  row
    checkbox "Remember me" checked=true
    link "Forgot password?"
  end
  button "Sign in" variant=primary

  flow login "Signing in"
    direction right
    node start "Tap Sign in" kind=start
    start -> check "go"

screen home "Home" device=tablet
  tabbar "Home" "Search" "Me" selected=1
  flow browse
    a -> b
  flow buy "Buying"
`

func TestParseProject(t *testing.T) {
	p, err := ParseProject(app)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Shop" || len(p.Screens) != 2 {
		t.Fatalf("project: %+v", p)
	}
	s := p.Screens[0]
	if s.ID != "signin" || len(s.UI) != 4 || len(s.UI[2].Children) != 2 || s.UI[1].Attr("placeholder") != "you@example.com" {
		t.Fatalf("signin: %+v", s)
	}
	if f := s.Flows[0]; f.ID != "login" || f.Title != "Signing in" || f.Direction != Right || len(f.Nodes) != 2 {
		t.Fatalf("login: %+v", f)
	}
	if home := p.Screens[1]; home.Device != "tablet" || len(home.Flows) != 2 || len(home.UI[0].Items) != 3 {
		t.Fatalf("home: %+v", home)
	}
	if s, f := p.FlowIndex("buy"); s != 1 || f != 1 {
		t.Fatalf("buy at %d %d", s, f)
	}
	again, err := ParseProject(p.Format())
	if err != nil {
		t.Fatalf("%v\n%s", err, p.Format())
	}
	if again.Format() != p.Format() {
		t.Fatalf("not stable:\n%s\n---\n%s", p.Format(), again.Format())
	}
	if !strings.Contains(p.Format(), "  row\n    checkbox \"Remember me\" checked=true\n") {
		t.Fatalf("format:\n%s", p.Format())
	}
}

func TestParseProjectReadsAFlowWithoutScreens(t *testing.T) {
	p, err := ParseProject(sample)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Order handling" || len(p.Screens) != 1 || p.Screens[0].ID != "main" || p.Screens[0].Flows[0].ID != "main" {
		t.Fatalf("project: %+v", p)
	}
	if f := p.Screens[0].Flows[0]; f.Direction != Right || len(f.Nodes) != 5 {
		t.Fatalf("flow: %+v", f)
	}
}

func TestParseProjectErrors(t *testing.T) {
	for src, want := range map[string]string{
		"screen a\nwidget":                   "line 2: unknown element",
		"screen a\nrow\nbutton \"x\"":        "line 3: row has no end",
		"screen a\nend":                      "end closes no",
		"screen a\nbutton \"x\" variant=big": "unknown variant",
		"screen a\nbutton \"x\" tone=muted":  "button takes no attribute \"tone\"; use icon, variant, size",
		"screen a\nscreen a":                 "a screen \"a\" exists",
		"screen a\nflow f\nscreen b\nflow f": "a flow \"f\" exists",
		"screen a device=watch":              "unknown device",
		"screen a\ntitle \"x\"":              "title belongs before",
		"screen a\ntabbar \"a\" selected=x":  "not a whole number",
	} {
		if _, err := ParseProject(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}
