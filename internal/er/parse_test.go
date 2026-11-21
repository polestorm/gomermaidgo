package er

import (
	"errors"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: ER-005
func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src string
		line      int
		code      string
	}{
		{"empty", "", 0, CodeHeader},
		{"wrong header", "er\nA ||--o{ B : x", 1, CodeHeader},
		{"no entities", "erDiagram\n", 0, graph.CodeInvalid},
		{"bad cardinality", "erDiagram\nA |--| B : x", 2, CodeStatement},
		{"missing entity", "erDiagram\nA ||--o{ : x", 2, CodeStatement},
		{"words", "erDiagram\nA only one to zero or more B : x", 2, CodeUnsupported},
		{"bad attribute", "erDiagram\nA {\n  samotny\n}", 3, CodeAttribute},
		{"bad key", "erDiagram\nA {\n  int id XK\n}", 3, CodeAttribute},
		{"unclosed block", "erDiagram\nA {\n  int id", 2, CodeEntity},
		{"bad direction", "erDiagram\ndirection XY\nA", 2, CodeStatement},
		{"undefined class", "erDiagram\nA:::brak", 2, graph.CodeInvalid},
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

// REQ: ER-002 ER-003
func TestParseEntitiesAndRelations(t *testing.T) {
	g, err := Parse("erDiagram\nKLIENT ||--o{ ZAMOWIENIE : składa\n\"Pozycja zamówienia\" }|..|| ZAMOWIENIE : \"należy<br/>do\"\nKLIENT[\"Klient firmy\"] {\n  string nip PK, UK \"bez kresek\"\n  string nazwa\n}\nSAMOTNA")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 4 || g.Node("Pozycja zamówienia") == nil || g.Node("SAMOTNA").Shape != graph.Entity {
		t.Fatalf("nodes: %s", g.Dump())
	}
	k := g.Node("KLIENT")
	if k.Label[0] != "Klient firmy" || len(k.Table) != 2 || k.Table[0][2] != "PK, UK" || k.Table[0][3] != "bez kresek" || k.Table[1][0] != "string" {
		t.Errorf("entity: %+v", k)
	}
	e := g.Edges[0]
	if e.StartHead != graph.OnlyOne || e.EndHead != graph.ZeroOrMore || e.Style != graph.Normal || e.Label[0] != "składa" {
		t.Errorf("first relation: %+v", e)
	}
	e = g.Edges[1]
	if e.StartHead != graph.OneOrMore || e.EndHead != graph.OnlyOne || e.Style != graph.Dotted || len(e.Label) != 2 {
		t.Errorf("second relation: %+v", e)
	}
}
