package flowchart

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: FLOW-010 FLOW-011
func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src string
		line      int
		code      string
	}{
		{"wrong header", "flow TD\nA-->B", 1, CodeHeader},
		{"empty", "", 0, CodeHeader},
		{"bad direction", "flowchart XY\nA-->B", 1, CodeHeader},
		{"text after direction", "flowchart TD A-->B", 1, CodeHeader},
		{"no nodes", "flowchart TD\n", 0, graph.CodeInvalid},
		{"dangling edge", "flowchart TD\nA -->", 2, CodeNode},
		{"edge without source", "flowchart TD\n--> B", 2, CodeNode},
		{"two dashes", "flowchart TD\nA -- B", 2, CodeEdge},
		{"junk after node", "flowchart TD\nA B", 2, CodeEdge},
		{"unclosed shape", "flowchart TD\nA[tekst", 2, CodeNode},
		{"unclosed quote", "flowchart TD\nA[\"tekst]", 2, CodeNode},
		{"mismatched slant", "flowchart TD\nA[/tekst]", 2, CodeNode},
		{"unclosed label", "flowchart TD\nA -->|tak B", 2, CodeEdge},
		{"two labels", "flowchart TD\nA -- raz -->|dwa| B", 2, CodeEdge},
		{"half bidirectional", "flowchart TD\nA <--- B", 2, CodeEdge},
		{"markdown string", "flowchart TD\nA[\"`**pogrubienie**`\"]", 2, CodeUnsupported},
		{"class shorthand", "flowchart TD\nA:::ważne --> B", 2, CodeUnsupported},
		{"new shape syntax", "flowchart TD\nA@{ shape: rect }", 2, CodeUnsupported},
		{"style", "flowchart TD\nA-->B\nstyle A fill:#f00", 3, CodeUnsupported},
		{"classDef", "flowchart TD\nclassDef x fill:#f00", 2, CodeUnsupported},
		{"click", "flowchart TD\nA-->B\nclick A href \"https://example.com\"", 3, CodeUnsupported},
		{"linkStyle", "flowchart TD\nA-->B\nlinkStyle 0 stroke:red", 3, CodeUnsupported},
		{"end without subgraph", "flowchart TD\nA-->B\nend", 3, CodeSubgraph},
		{"unclosed subgraph", "flowchart TD\nsubgraph S\nA-->B", 0, CodeSubgraph},
		{"subgraph without name", "flowchart TD\nsubgraph\nA\nend", 2, CodeSubgraph},
		{"duplicate subgraph", "flowchart TD\nsubgraph S\nA\nend\nsubgraph S\nB\nend", 5, graph.CodeInvalid},
		{"direction outside subgraph", "flowchart TD\ndirection LR\nA", 2, CodeStatement},
		{"node and subgraph", "flowchart TD\nS[Węzeł] --> A\nsubgraph S\nB\nend", 2, graph.CodeInvalid},
		{"edge into own contents", "flowchart TD\nsubgraph S\nA\nend\nS --> A", 5, graph.CodeInvalid},
		{"edge from own contents", "flowchart TD\nsubgraph S\nA\nend\nA --> S", 5, graph.CodeInvalid},
		{"subgraph to itself", "flowchart TD\nsubgraph S\nA\nend\nS --> S", 5, graph.CodeInvalid},
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

// REQ: FLOW-011
func TestLimits(t *testing.T) {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for i := 0; i <= graph.MaxNodes; i++ {
		fmt.Fprintf(&b, "n%d\n", i)
	}
	var se *syntax.Error
	if _, err := Parse(b.String()); !errors.As(err, &se) || se.Code != graph.CodeLimit {
		t.Errorf("node limit: got %v", err)
	}
	b.Reset()
	b.WriteString("flowchart TD\n")
	for i := 0; i <= graph.MaxNesting; i++ {
		fmt.Fprintf(&b, "subgraph s%d\n", i)
	}
	if _, err := Parse(b.String()); !errors.As(err, &se) || se.Code != graph.CodeLimit {
		t.Errorf("nesting limit: got %v", err)
	}
}

// REQ: FLOW-002 FLOW-007
func TestRedefinitionAndMembership(t *testing.T) {
	g, err := Parse("flowchart TD\nA --> B\nA[Pierwsza]\nsubgraph S\nB --> C\nsubgraph T\nC\nD\nend\nend\nA(Druga)")
	if err != nil {
		t.Fatal(err)
	}
	if a := g.Node("A"); a.Shape != graph.Round || a.Label[0] != "Druga" || a.Parent != "" {
		t.Errorf("A: %+v", a)
	}
	// B wystąpił najpierw poza podgrafem, potem w S: należy do S. C wystąpił w S, potem w T: zostaje w S.
	if g.Node("B").Parent != "S" || g.Node("C").Parent != "S" || g.Node("D").Parent != "T" {
		t.Errorf("membership: B=%q C=%q D=%q", g.Node("B").Parent, g.Node("C").Parent, g.Node("D").Parent)
	}
}

// REQ: FLOW-003
func TestCompactEdges(t *testing.T) {
	// Krawędzie bez odstępów oraz znane pułapki: litera o albo x tuż za linią jest zakończeniem.
	g, err := Parse("flowchart TD\nA-->B\nB---C\nC-.->D\nD==>E\nE--oF\nF--xG\nG --- oH\nH-->|t|I")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"B arrow", "C none", "D arrow", "E arrow", "F circle", "G cross", "oH none", "I arrow"}
	for i, e := range g.Edges {
		if got := e.To + " " + string(e.EndHead); got != want[i] {
			t.Errorf("edge %d: got %q, want %q", i, got, want[i])
		}
	}
}
