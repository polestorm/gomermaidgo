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
		{"unclosed markdown string", "flowchart TD\nA[\"`**pogrubienie**\"]", 2, CodeNode},
		{"undefined class", "flowchart TD\nA:::ważne --> B", 2, graph.CodeInvalid},
		{"style without properties", "flowchart TD\nA-->B\nstyle A", 3, CodeStatement},
		{"bad colour", "flowchart TD\nA-->B\nstyle A fill:url(#x)", 3, graph.CodeInvalid},
		{"bad width", "flowchart TD\nA-->B\nstyle A stroke-width:gruba", 3, graph.CodeInvalid},
		{"linkStyle out of range", "flowchart TD\nA-->B\nlinkStyle 3 stroke:red", 3, graph.CodeInvalid},
		{"interpolate outside profile", "flowchart TD\nA-->B\ninterpolate basis", 3, CodeUnsupported},
		{"nested lane", "%% layout: swimlanes\nflowchart TD\nsubgraph A\nsubgraph B\nx\nend\nend", 4, CodeSubgraph},
		{"edge to a lane", "%% layout: swimlanes\nflowchart TD\nsubgraph A\nx\nend\ny --> A", 6, CodeEdge},
		{"new shape syntax", "flowchart TD\nA@{ shape: rect }", 2, CodeUnsupported},
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

// REQ: FLOW-017 FLOW-018
func TestParseStylesAndIdentifiers(t *testing.T) {
	g, err := Parse("flowchart TD\nclassDef default fill:#fff\nclassDef ważne,pilne fill:#fde,stroke:#b00,stroke-width:2px,stroke-dasharray: 5 5,font-weight:bold\n" +
		"krok-1.a:::ważne[\"`Pierwszy **krok**`\"] --> B[Drugi]:::pilne\nB --> C\nsubgraph S\nC\nend\n" +
		"class C ważne\nstyle C fill:rgb(1, 2, 3),color:white\nstyle S fill:#eee\nlinkStyle default stroke:#999\nlinkStyle 1 stroke:red,stroke-width:3px\nclick B href \"https://example.org\"")
	if err != nil {
		t.Fatal(err)
	}
	a := g.Node("krok-1.a")
	if a == nil || graph.Plain(a.Label[0]) != "Pierwszy krok" || !graph.HasMarks(a.Label[0]) || a.Paint != (graph.Paint{Fill: "#fde", Stroke: "#b00", StrokeWidth: 2, Dash: "5 5"}) {
		t.Errorf("node with a hyphen and a dot: %+v", a)
	}
	if c := g.Node("C"); c.Paint.Fill != "rgb(1,2,3)" || c.Paint.Color != "white" || c.Paint.Stroke != "#b00" {
		t.Errorf("direct style should win over the class: %+v", c.Paint)
	}
	if g.Subgraph("S").Paint.Fill != "#eee" || g.Edges[0].Paint.Stroke != "#999" || g.Edges[1].Paint != (graph.Paint{Stroke: "red", StrokeWidth: 3}) {
		t.Errorf("subgraph or link styles: %+v %+v %+v", g.Subgraph("S").Paint, g.Edges[0].Paint, g.Edges[1].Paint)
	}
}

// REQ: FLOW-018
// Tekst Markdown rozciągnięty na kilka linii źródła: koniec linii jest złamaniem tekstu, a numery
// linii w błędach za takim tekstem się nie przesuwają.
func TestParseMultilineMarkdownString(t *testing.T) {
	g, err := Parse("flowchart TD\nA[\"`Pierwsza **linia**\n    druga linia\n  trzecia`\"] --> B\nB --> C")
	if err != nil {
		t.Fatal(err)
	}
	a := g.Node("A")
	if len(a.Label) != 3 || graph.Plain(a.Label[0]) != "Pierwsza linia" || a.Label[1] != "druga linia" || a.Label[2] != "trzecia" {
		t.Errorf("label lines: %q", a.Label)
	}
	if len(g.Edges) != 2 {
		t.Errorf("%d edges, want 2", len(g.Edges))
	}
	_, err = Parse("flowchart TD\nA[\"`raz\ndwa`\"] --> B\nB --> C{{")
	if err == nil || !strings.Contains(err.Error(), "line 4") {
		t.Errorf("error after a multi-line string should point at line 4: %v", err)
	}
	_, err = Parse("flowchart TD\nA[\"`raz\ndwa] --> B")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("unclosed string should be reported at line 2: %v", err)
	}
}
