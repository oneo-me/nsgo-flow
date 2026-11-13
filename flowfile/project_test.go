package flowfile

import (
	"slices"
	"strings"
	"testing"
)

const app = `title "Shop"

screen signin "Sign in"
  text "Sign in" class="text-2xl font-bold"
  image "A cup" src="https://example.com/cup.png"
  box class="flex-row gap-2"
    icon lock class="size-4"
    text "Remember me"
  end
  svg "<svg viewBox='0 0 10 10'><circle cx='5' cy='5' r='4'/></svg>"

  flow login "Signing in"
    direction right
    node start "Tap Sign in" kind=start
    start -> check "go"

screen home "Home" device=tablet
  box
  end
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
	if s.ID != "signin" || len(s.UI) != 4 || len(s.UI[2].Children) != 2 || s.UI[1].Attr("src") != "https://example.com/cup.png" {
		t.Fatalf("signin: %+v", s)
	}
	if f := s.Flows[0]; f.ID != "login" || f.Title != "Signing in" || f.Direction != Right || len(f.Nodes) != 2 {
		t.Fatalf("login: %+v", f)
	}
	if home := p.Screens[1]; home.Device != "tablet" || len(home.Flows) != 2 || home.UI[0].Kind != "box" {
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
	if !strings.Contains(p.Format(), "  box class=\"flex-row gap-2\"\n    icon \"lock\" class=\"size-4\"\n") {
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
		"screen a\nbox\ntext \"x\"":          "line 3: box has no end",
		"screen a\nend":                      "end closes no",
		"screen a\ntext \"x\" src=y":         "text takes no attribute \"src\"; use class",
		"screen a\ntext x y":                 "quote text with spaces",
		"screen a\nicon nope":                "unknown icon \"nope\"",
		"screen a\nicon cofee-cup":           "such as cup-soda",
		"screen a\nsvg \"<div/>\"":           "svg needs a document",
		"screen a\nsvg \"<svg>\"":            "svg XML syntax error",
		"screen a\nscreen a":                 "a screen \"a\" exists",
		"screen a\nflow f\nscreen b\nflow f": "a flow \"f\" exists",
		"screen a device=watch":              "unknown device",
		"screen a\ntitle \"x\"":              "title belongs before",
	} {
		if _, err := ParseProject(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}

func TestElementPaths(t *testing.T) {
	els, err := ParseUI("text \"a\"\nbox\n  text \"b\"\n  text \"c\"\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	if el, ok := ElementAt(els, "1.1"); !ok || el.Text != "c" {
		t.Fatalf("1.1 = %+v", el)
	}
	if _, ok := ElementAt(els, "0.0"); ok {
		t.Fatal("a text holds an element")
	}
	with, _ := ParseUI("text \"x\"\ntext \"y\"\n")
	out, err := ReplaceElement(els, "1.0", with)
	if err != nil || FormatUI(out) != "text \"a\"\nbox\n  text \"x\"\n  text \"y\"\n  text \"c\"\nend\n" {
		t.Fatalf("%v\n%s", err, FormatUI(out))
	}
	if FormatUI(els) != "text \"a\"\nbox\n  text \"b\"\n  text \"c\"\nend\n" {
		t.Fatal("replacing changed the elements")
	}
	if _, err := ReplaceElement(els, "5", nil); err == nil {
		t.Fatal("replaced a missing element")
	}
}

func TestComments(t *testing.T) {
	p, err := ParseProject("screen home\n  box comment=\"Roomier\" class=\"p-2\"\n    text \"a\"\n  end\nscreen pay\n  icon mail comment=\"Bigger\"\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []Comment{{"home", "0", "Roomier"}, {"pay", "0", "Bigger"}}
	if got := p.Comments(); !slices.Equal(got, want) {
		t.Fatalf("comments %v", got)
	}
	if got := FormatUI(p.Screens[0].UI); got != "box class=\"p-2\" comment=\"Roomier\"\n  text \"a\"\nend\n" {
		t.Fatalf("formatted as\n%s", got)
	}
	els, err := SetComment(p.Screens[0].UI, "0.0", "Bolder")
	if err != nil {
		t.Fatal(err)
	}
	if els, _ = SetComment(els, "0", ""); FormatUI(els) != "box class=\"p-2\"\n  text \"a\" comment=\"Bolder\"\nend\n" {
		t.Fatalf("set as\n%s", FormatUI(els))
	}
	if p.Screens[0].UI[0].Attr("comment") != "Roomier" || p.Screens[0].UI[0].Children[0].Attr("comment") != "" {
		t.Fatal("setting a comment changed the elements")
	}
	if _, err := SetComment(els, "3", "x"); err == nil {
		t.Fatal("commented a missing element")
	}
}

func TestIconsTakeLucideNamesAndAliases(t *testing.T) {
	els, err := ParseUI("icon \"alert-circle\"\nicon \"badge-dollar-sign\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if els[0].Text != "circle-alert" || els[1].Text != "badge-dollar-sign" {
		t.Fatalf("icons read as %q and %q", els[0].Text, els[1].Text)
	}
	for alias, name := range iconAliases {
		if _, ok := IconName(name); !ok {
			t.Errorf("alias %s names no icon %s", alias, name)
		}
	}
	for _, name := range CommonIcons {
		if _, ok := IconName(name); !ok {
			t.Errorf("no common icon %s", name)
		}
	}
}
