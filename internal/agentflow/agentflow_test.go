package agentflow

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: AGF-001 AGF-002 AGF-003
func TestParseShapesAndEdges(t *testing.T) {
	g, err := Parse("agentflow LR\nq[\"Pytanie\"]@{ shape: input }\nt[\"Zadanie\"]@{ shape: task, model: \"x\" }\ntool[\"Narzędzie\"]@{ shape: tool }\nd[\"Decyzja\"]@{ shape: decision }\nr[\"Dokument\"]@{ shape: refdoc }\na[\"Akcja\"]@{ shape: action }\nc@{ shape: cyl, label: \"Baza\" }\nq --> t --> tool\ntool -.- r\nt -- tak --> d\nd --x a\nc --> a")
	if err != nil {
		t.Fatal(err)
	}
	if g.Direction != graph.LeftRight {
		t.Errorf("direction %q", g.Direction)
	}
	want := map[string]graph.Shape{"q": graph.Parallelogram, "t": graph.Round, "tool": graph.Subroutine, "d": graph.Rhombus, "r": graph.Rect, "a": graph.Hexagon, "c": graph.Cylinder}
	for id, sh := range want {
		if n := g.Node(id); n == nil || n.Shape != sh {
			t.Errorf("%s: %+v, want shape %s", id, n, sh)
		}
	}
	if g.Node("c").Label[0] != "Baza" || g.Node("r").Paint.Dash == "" {
		t.Errorf("label %q, refdoc dash %q", g.Node("c").Label, g.Node("r").Paint.Dash)
	}
	var edges []string
	for _, e := range g.Edges {
		edges = append(edges, e.From+">"+e.To+":"+string(e.Style)+":"+string(e.EndHead)+":"+strings.Join(e.Label, " "))
	}
	if got, want := strings.Join(edges, " "), "q>t:normal:arrow: t>tool:normal:arrow: tool>r:dotted:none: t>d:normal:arrow:tak d>a:normal:cross: c>a:normal:arrow:"; got != want {
		t.Errorf("edges\n%s\nwant\n%s", got, want)
	}
}

// REQ: AGF-004
func TestTranslateContainers(t *testing.T) {
	src := "agentflow\nglobal\n  kb[\"Baza\"]@{ shape: refdoc }\nend\nconnector api[\"API\"]@{ protocol: \"http\" }\nflow f1[\"Przegląd\"]@{ algorithm: \"elk.rectpacking\" }\n  a[\"A\"]@{\n    shape: tool\n    connectorRef: api\n  }\n  flow f2[\"Wewnątrz\"]\n    b[\"B\"]\n  end\nend\nflow p[\"Publikacja\"]\n  x[\"X\"] --> y[\"Y\"]\nend\np@{ view: \"collapsed\" }\na --> b --> x\nb -.- kb"
	fc, err := Translate(src)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(fc, "\n"); n < strings.Count(src, "\n") {
		t.Errorf("line count changed: %d < %d", n, strings.Count(src, "\n"))
	}
	g, err := Parse(src)
	if err != nil {
		t.Fatalf("%v\n%s", err, fc)
	}
	if g.Subgraph("f1") == nil || g.Subgraph("f2") == nil || g.Subgraph("f2").Parent != "f1" || g.Node("a").Parent != "f1" {
		t.Errorf("containers: %+v", g.Subgraphs)
	}
	if g.Node("kb").Parent != "" || g.Node("api").Shape != graph.Stadium {
		t.Errorf("global or connector: %+v %+v", g.Node("kb"), g.Node("api"))
	}
	if g.Subgraph("p") != nil || g.Node("p") == nil || g.Node("x") != nil {
		t.Errorf("collapsed container is not one node")
	}
	var edges []string
	for _, e := range g.Edges {
		edges = append(edges, e.From+">"+e.To)
	}
	if got, want := strings.Join(edges, " "), "a>b b>p b>kb a>api"; got != want {
		t.Errorf("edges %q, want %q", got, want)
	}
}

// REQ: AGF-005
func TestErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                      {CodeHeader, 0},
		"agentflow DOWN":                        {CodeHeader, 1},
		"agentflow\nend":                        {CodeStatement, 2},
		"agentflow\nflow f[\"F\"]\n  a":         {CodeStatement, 0},
		"agentflow\na[\"A\"]@{ label: 'x\"y' }": {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
	// Błąd węzła zgłasza parser schematu blokowego z numerem linii źródła.
	_, err := Parse("agentflow\n\na[\"A\"]@{ shape: cloud }")
	if e, ok := err.(*syntax.Error); !ok || e.Line != 3 {
		t.Errorf("shape outside the profile: %v", err)
	}
}
