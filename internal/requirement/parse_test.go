package requirement

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: RQD-001 RQD-002 RQD-003
func TestParse(t *testing.T) {
	g, err := Parse("%% komentarz\nrequirementDiagram\ndirection LR\naccTitle: Wymagania\nclassDef wazne fill:#fde\nrequirement R1 {\n  id: 1\n  text: \"krótki tekst\"\n  risk: HIGH\n  verifymethod: test\n}\nperformanceRequirement \"Drugie wymaganie\":::wazne {\n  id: 2.1\n  text: " + strings.Repeat("słowo ", 20) + "\n}\nelement E {\n  type: \"symulacja\"\n  docref: DOC-1\n}\nE - satisfies -> R1\n\"Drugie wymaganie\" <- contains - R1\nstyle E stroke:#00f")
	if err != nil {
		t.Fatal(err)
	}
	if g.Direction != graph.LeftRight || g.AccTitle != "Wymagania" || len(g.Nodes) != 3 {
		t.Fatalf("graph: %+v", g)
	}
	r1 := g.Node("R1")
	var rows []string
	for _, m := range r1.Members {
		rows = append(rows, m.Text)
	}
	if r1.Annotation != "Requirement" || r1.Shape != graph.Requirement || strings.Join(rows, "|") != "Id: 1|Text: krótki tekst|Risk: High|Verification: Test" {
		t.Errorf("requirement: %q %q", r1.Annotation, rows)
	}
	second := g.Node("Drugie wymaganie")
	if second == nil || second.Annotation != "Performance Requirement" || second.Paint.Fill != "#fde" || len(second.Members) < 4 {
		t.Fatalf("quoted requirement with a long text: %+v", second)
	}
	for _, m := range second.Members {
		if len([]rune(m.Text)) > 50 {
			t.Errorf("long text was not wrapped: %q", m.Text)
		}
	}
	if e := g.Node("E"); e.Annotation != "Element" || e.Members[0].Text != "Type: symulacja" || e.Members[1].Text != "Doc Ref: DOC-1" || e.Paint.Stroke != "#00f" {
		t.Errorf("element: %+v", e)
	}
	a, b := g.Edges[0], g.Edges[1]
	if a.From != "E" || a.To != "R1" || a.Label[0] != "«satisfies»" || a.EndHead != graph.OpenArrow {
		t.Errorf("forward relation: %+v", a)
	}
	if b.From != "R1" || b.To != "Drugie wymaganie" || b.Label[0] != "«contains»" || b.StartHead != graph.CircleHead || b.EndHead != graph.NoHead || b.Style != graph.Normal || g.Edges[0].Style != graph.Dotted {
		t.Errorf("backward contains relation: %+v", b)
	}
}

// REQ: RQD-004
func TestParseErrors(t *testing.T) {
	ok := "requirementDiagram\nrequirement A {\nid: 1\n}\n"
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                         {CodeHeader, 0},
		"requirements\nrequirement A {\n}":         {CodeHeader, 1},
		"requirementDiagram\nwymaganie A {":        {CodeBlock, 2},
		ok + "requirement A {\n}":                  {CodeBlock, 5},
		ok + "requirement B {\nid: 2":              {CodeBlock, 5},
		ok + "requirement B {\nbez dwukropka\n}":   {CodeField, 6},
		ok + "requirement B {\nrisk: ogromne\n}":   {CodeField, 6},
		ok + "requirement B {\nverifymethod: x\n}": {CodeField, 6},
		ok + "requirement B {\ntype: x\n}":         {CodeField, 6},
		ok + "element E {\nid: 3\n}":               {CodeField, 6},
		ok + "A - lubi -> A":                       {CodeRelation, 5},
		ok + "A - traces -> Nieznane":              {CodeRelation, 5},
		ok + "to nie jest instrukcja":              {CodeStatement, 5},
		ok + "direction XY":                        {CodeStatement, 5},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.50q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}
