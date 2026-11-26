package usecase

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: UC-001 UC-002 UC-003
func TestParseElements(t *testing.T) {
	g, err := Parse("%% komentarz\nusecase-beta\ndirection LR\naccTitle: Konto\nCustomer --> Login\nactor Customer\nactor Admin(\"Main administrator\")@{ type: hollow } <<Staff>>:::ext\nsystemBoundary sb[\"Shop\"]@{ type: package }\n  Login(\"Sign in\")\n  Report[Generate report]\nend\n\"Reset password\"\nsb@{ type: rect }\nclassDef ext stroke:#999")
	if err != nil {
		t.Fatal(err)
	}
	if g.Direction != graph.LeftRight || g.AccTitle != "Konto" {
		t.Errorf("direction %q, accTitle %q", g.Direction, g.AccTitle)
	}
	want := map[string]struct {
		shape  graph.Shape
		label  string
		parent string
	}{
		"Customer":       {graph.Actor, "Customer", ""},
		"Admin":          {graph.Actor, "«Staff»|Main administrator", ""},
		"Login":          {graph.UseCase, "Sign in", "sb"},
		"Report":         {graph.Rect, "Generate report", "sb"},
		"Reset_password": {graph.UseCase, "Reset password", ""},
	}
	for id, w := range want {
		n := g.Node(id)
		if n == nil || n.Shape != w.shape || strings.Join(n.Label, "|") != w.label || n.Parent != w.parent {
			t.Errorf("%s: %+v, want %+v", id, n, w)
		}
	}
	if s := g.Subgraph("sb"); s == nil || s.Title[0] != "Shop" {
		t.Errorf("boundary: %+v", s)
	}
}

// REQ: UC-004
func TestParseRelations(t *testing.T) {
	g, err := Parse("usecase-beta\nactor A\nactor B\nA --> X\nX <-- B\nX -- Y\nA --o X\nX o-- B\nA --x Y\nY x-- B\nA -- \"opens form\" ---> Y\nX ..> : include Y\nY ..> : extend X\nA --|> B\nnote for X \"Note\"")
	if err != nil {
		t.Fatal(err)
	}
	type sum struct {
		from, to   string
		start, end graph.Head
		style      graph.LineStyle
		label      string
		min        int
	}
	var got []sum
	for _, e := range g.Edges {
		got = append(got, sum{e.From, e.To, e.StartHead, e.EndHead, e.Style, strings.Join(e.Label, "|"), e.MinLen})
	}
	want := []sum{
		{"A", "X", graph.NoHead, graph.ArrowHead, graph.Normal, "", 1},
		{"X", "B", graph.ArrowHead, graph.NoHead, graph.Normal, "", 1},
		{"X", "Y", graph.NoHead, graph.NoHead, graph.Normal, "", 1},
		{"A", "X", graph.NoHead, graph.CircleHead, graph.Normal, "", 1},
		{"X", "B", graph.CircleHead, graph.NoHead, graph.Normal, "", 1},
		{"A", "Y", graph.NoHead, graph.CrossHead, graph.Normal, "", 1},
		{"Y", "B", graph.CrossHead, graph.NoHead, graph.Normal, "", 1},
		{"A", "Y", graph.NoHead, graph.ArrowHead, graph.Normal, "opens form", 2},
		{"X", "Y", graph.NoHead, graph.OpenArrow, graph.Dotted, "«include»", 1},
		{"Y", "X", graph.NoHead, graph.OpenArrow, graph.Dotted, "«extend»", 1},
		{"A", "B", graph.NoHead, graph.Triangle, graph.Normal, "", 1},
		{"note-1", "X", graph.NoHead, graph.NoHead, graph.Dotted, "", 1},
	}
	if len(got) != len(want) {
		t.Fatalf("%d edges, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("edge %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	if g.Node("X").Shape != graph.UseCase || g.Node("note-1").Shape != graph.NoteBox {
		t.Error("undeclared endpoint is not a use case, or the note is not a note")
	}
}

// REQ: UC-002 UC-005
func TestParseLabelsJSONAndStyles(t *testing.T) {
	g, err := Parse("usecase-beta\nF(\"`**Bold**\nline`\")\nQ(\"Say #quot;hi#quot; #40;now#41;\")\njson P@{\n  \"b\": 1,\n  \"a\": { \"c\": [\"x\", {}] },\n  \"b\": 2,\n  \"n\": null\n}:::data\nactor U\nU e1@--> F\nclassDef data stroke:#36c\nclassDef hot stroke:#c00\nclass e1 hot\nstyle F stroke-width:3px")
	if err != nil {
		t.Fatal(err)
	}
	if f := g.Node("F"); len(f.Label) != 2 || graph.Plain(f.Label[0]) != "Bold" || !graph.HasMarks(f.Label[0]) || f.Label[1] != "line" {
		t.Errorf("markdown label: %q", f.Label)
	}
	if q := g.Node("Q"); q.Label[0] != `Say "hi" (now)` {
		t.Errorf("entity codes: %q", q.Label)
	}
	p := g.Node("P")
	var rows []string
	for _, r := range p.Table {
		rows = append(rows, r[0]+"="+r[1])
	}
	if p.Shape != graph.Entity || strings.Join(rows, " ") != "b=2 a.c[0]=x a.c[1]={} n=null" || p.Paint.Stroke != "#36c" {
		t.Errorf("json: %v %+v", rows, p.Paint)
	}
	if g.Edges[0].Paint.Stroke != "#c00" || g.Node("F").Paint.StrokeWidth != 3 {
		t.Errorf("styles: edge %+v, node %+v", g.Edges[0].Paint, g.Node("F").Paint)
	}
}

// REQ: UC-006
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                     {CodeHeader, 0},
		"usecaseDiagram\nactor A":              {CodeHeader, 1},
		"usecase-beta\nactor A\nactor A":       {CodeStatement, 3},
		"usecase-beta\nactor A@{ color: red }": {CodeStatement, 2},
		"usecase-beta\nsystemBoundary S\nX":    {CodeStatement, 0},
		"usecase-beta\nsystemBoundary S\nsystemBoundary T": {CodeStatement, 3},
		"usecase-beta\nend":                     {CodeStatement, 2},
		"usecase-beta\nA ..> : uses B":          {CodeStatement, 2},
		"usecase-beta\nactor A\nA --|> B":       {CodeStatement, 3},
		"usecase-beta\njson J@{ \"a\": }":       {CodeStatement, 2},
		"usecase-beta\njson J@[1, 2]":           {CodeStatement, 2},
		"usecase-beta\n?? what":                 {CodeStatement, 2},
		"usecase-beta\ndirection UP":            {CodeStatement, 2},
		"usecase-beta\nactor A@{ type: robot }": {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: UC-001 UC-002 UC-006
func TestParseVariantsAndDefaultDirection(t *testing.T) {
	g, err := Parse("usecase-beta\nactor A@{ type: hollow, business: true }\nactor B@{ type: \"awesome\" }\nU(\"Zamów\")@{ business: true }\nV(\"Wydaj\")\nA --> U")
	if err != nil {
		t.Fatal(err)
	}
	if g.Direction != graph.LeftRight {
		t.Errorf("default direction %q, want LR", g.Direction)
	}
	for id, want := range map[string]struct {
		variant  string
		business bool
	}{"A": {"hollow", true}, "B": {"awesome", false}, "U": {"", true}, "V": {"", false}} {
		if n := g.Node(id); n == nil || n.Variant != want.variant || n.Business != want.business {
			t.Errorf("%s: %+v", id, n)
		}
	}
}
