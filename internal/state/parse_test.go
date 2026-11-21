package state

import (
	"errors"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: STATE-009
func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src string
		line      int
		code      string
	}{
		{"wrong header", "state\nA --> B", 1, CodeHeader},
		{"empty", "", 0, CodeHeader},
		{"no states", "stateDiagram-v2\n", 0, graph.CodeInvalid},
		{"dangling transition", "stateDiagram-v2\nA -->", 2, CodeStatement},
		{"wrong arrow", "stateDiagram-v2\nA -> B", 2, CodeStatement},
		{"unclosed composite", "stateDiagram-v2\nstate A {\nB --> C", 0, CodeComposite},
		{"stray brace", "stateDiagram-v2\nA --> B\n}", 3, CodeComposite},
		{"bad declaration", "stateDiagram-v2\nstate \"Opis\" jako A", 2, CodeStatement},
		{"bare state keyword", "stateDiagram-v2\nstate A", 2, CodeUnsupported},
		{"unknown stereotype", "stateDiagram-v2\nstate A <<merge>>", 2, CodeStatement},
		{"region outside composite", "stateDiagram-v2\nA --> B\n--\nC", 3, CodeComposite},
		{"undefined class", "stateDiagram-v2\nA:::ważny --> B", 2, graph.CodeInvalid},
		{"bad style value", "stateDiagram-v2\nA --> B\nclassDef x fill:url(#a)", 3, graph.CodeInvalid},
		{"scale outside profile", "stateDiagram-v2\nscale 350 width\nA --> B", 2, CodeUnsupported},
		{"note without side", "stateDiagram-v2\nA --> B\nnote over A : tekst", 3, CodeNote},
		{"unclosed note", "stateDiagram-v2\nA --> B\nnote right of A\ntekst", 3, CodeNote},
		{"note for unknown state", "stateDiagram-v2\nA --> B\nnote right of C : tekst", 3, graph.CodeInvalid},
		{"bad direction", "stateDiagram-v2\ndirection XY\nA", 2, CodeStatement},
		{"duplicate composite", "stateDiagram-v2\nstate A {\nB\n}\nstate A {\nC\n}", 5, graph.CodeInvalid},
		{"transition into own contents", "stateDiagram-v2\nstate A {\nB\n}\nA --> B", 5, graph.CodeInvalid},
	}
	for _, c := range cases {
		_, err := Parse(c.src)
		var se *syntax.Error
		if !errors.As(err, &se) {
			t.Errorf("%s: want a syntax error, got %v", c.name, err)
			continue
		}
		if se.Line != c.line || se.Code != c.code {
			t.Errorf("%s: got line %d code %q (%s), want line %d code %q", c.name, se.Line, se.Code, se.Message, c.line, c.code)
		}
	}
}

// REQ: STATE-003 STATE-005
func TestCompositeDeclaredAfterUse(t *testing.T) {
	// Stan użyty w przejściu, opisany osobno i dopiero potem zdefiniowany jako złożony.
	g, err := Parse("stateDiagram-v2\n[*] --> P\nP : Przegląd dokumentu\nstate P {\n[*] --> A\nA --> [*]\n}\nP --> [*]")
	if err != nil {
		t.Fatal(err)
	}
	if g.Node("P") != nil {
		t.Error("P should be a composite state, not a node")
	}
	if s := g.Subgraph("P"); s == nil || s.Title[0] != "Przegląd dokumentu" {
		t.Errorf("composite P: %+v", s)
	}
	if g.Node("[start]") == nil || g.Node("[start]@P") == nil || g.Node("[end]@P") == nil || g.Node("[end]") == nil {
		t.Error("start and end states should be separate for each scope")
	}
}

// REQ: STATE-012 STATE-013 STATE-014
func TestParseRegionsClassesIdentifiers(t *testing.T) {
	g, err := Parse("stateDiagram-v2\naccTitle: Praca urządzenia\nclassDef alarm fill:#fde,stroke:#b00,stroke-width:2px\n" +
		"[*] --> W-toku\nstate W-toku {\n[*] --> Pomiar.1\nPomiar.1 --> [*]\n--\n[*] --> Zapis:::alarm\n}\nclass W-toku alarm")
	if err != nil {
		t.Fatal(err)
	}
	if g.AccTitle != "Praca urządzenia" {
		t.Errorf("accTitle: %q", g.AccTitle)
	}
	r1, r2 := g.Subgraph("W-toku#1"), g.Subgraph("W-toku#2")
	if r1 == nil || r2 == nil || !r1.Region || r1.Parent != "W-toku" || r2.Parent != "W-toku" {
		t.Fatalf("regions: %+v %+v", r1, r2)
	}
	if n := g.Node("Pomiar.1"); n == nil || n.Parent != "W-toku#1" {
		t.Errorf("first region did not adopt the earlier state: %+v", n)
	}
	if n := g.Node("Zapis"); n == nil || n.Parent != "W-toku#2" || n.Paint.Stroke != "#b00" || n.Paint.StrokeWidth != 2 {
		t.Errorf("state in the second region: %+v", n)
	}
	if g.Node("[start]@W-toku#2") == nil || g.Subgraph("W-toku").Paint.Fill != "#fde" {
		t.Errorf("region start or composite paint missing")
	}
}
