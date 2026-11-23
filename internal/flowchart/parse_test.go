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
		{"unknown named shape", "flowchart TD\nA@{ shape: cloud }", 2, CodeUnsupported},
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

// REQ: FLOW-018
// Znaczniki <b>, <strong>, <i> i <em> w etykiecie dają pogrubienie i kursywę, także gdy obejmują
// złamanie linii; inne znaczniki zostają tekstem.
func TestParseEmphasisTags(t *testing.T) {
	g, err := Parse("flowchart TD\nA[\"<b>Nazwa</b><br/><i>opis w<br/>dwóch liniach</i>\"] -->|<EM>może</EM>| B[<u>inne</u>]")
	if err != nil {
		t.Fatal(err)
	}
	a := g.Node("A")
	want := []string{"Nazwa", "opis w", "dwóch liniach"}
	if len(a.Label) != 3 {
		t.Fatalf("label lines: %q", a.Label)
	}
	for i, line := range a.Label {
		if graph.Plain(line) != want[i] || !graph.HasMarks(line) || strings.Count(line, string(graph.MarkBold))%2 != 0 || strings.Count(line, string(graph.MarkItalic))%2 != 0 {
			t.Errorf("line %d: %q", i, line)
		}
	}
	if !strings.Contains(a.Label[0], string(graph.MarkBold)) || !strings.Contains(a.Label[2], string(graph.MarkItalic)) {
		t.Errorf("wrong kind of emphasis: %q", a.Label)
	}
	if e := g.Edges[0]; graph.Plain(e.Label[0]) != "może" || !graph.HasMarks(e.Label[0]) {
		t.Errorf("edge label: %q", e.Label)
	}
	if b := g.Node("B"); b.Label[0] != "<u>inne</u>" {
		t.Errorf("other tags should stay as text: %q", b.Label)
	}
}

// REQ: FLOW-022
// Nowy zapis kształtu @{ shape: …, label: … } dla kształtów, które rysujemy, z synonimami nazw.
func TestParseNamedShapes(t *testing.T) {
	g, err := Parse("flowchart TD\nA@{ shape: diam, label: \"Czy **na pewno**?\" } --> B@{ shape: database }\nB --> C@{ label: 'Sam tekst, z przecinkiem' }:::k\nC --> D@{ shape: lean-r, label: \"Wejście\" }\nclassDef k fill:#fde\nD --> E@{shape: dbl-circ}\nE --> F@{ shape: notch-pent, label: \"Dla każdej partii\" }")
	if err != nil {
		t.Fatal(err)
	}
	if a := g.Node("A"); a.Shape != graph.Rhombus || a.Label[0] != "Czy **na pewno**?" {
		t.Errorf("A: %+v", a)
	}
	if b := g.Node("B"); b.Shape != graph.Cylinder || b.Label[0] != "B" {
		t.Errorf("B keeps its id as the label: %+v", b)
	}
	if c := g.Node("C"); c.Shape != graph.Rect || c.Label[0] != "Sam tekst, z przecinkiem" || c.Paint.Fill != "#fde" {
		t.Errorf("C: %+v", c)
	}
	if g.Node("D").Shape != graph.Parallelogram || g.Node("E").Shape != graph.DoubleCircle || len(g.Edges) != 5 || g.Node("F").Shape != graph.LoopLimit {
		t.Errorf("D, E and edges: %+v %+v %d", g.Node("D"), g.Node("E"), len(g.Edges))
	}
	for src, code := range map[string]string{
		"flowchart TD\nA@{ shape: cloud }": CodeUnsupported,
		"flowchart TD\nA@{ shape: rect":    CodeNode,
	} {
		_, err := Parse(src)
		if e, ok := err.(*syntax.Error); !ok || e.Code != code || e.Line != 2 {
			t.Errorf("%q: got %v, want code %s at line 2", src, err, code)
		}
	}
}

// REQ: FLOW-023
func TestParseTightLabelsAndIcons(t *testing.T) {
	g, err := Parse("flowchart TD\nA --tekst--> B\nB ==gruby==> C\nC -.kropki.-> D\nD --o E\nE --x F\nF --opis z o--> G\nG --- H --> I\nI[fa:fa-user Użytkownik] --> J[fa:fa-database]\nJ --> K[Zapis fas:fa-check gotowy]")
	if err != nil {
		t.Fatal(err)
	}
	var labels []string
	for _, e := range g.Edges {
		labels = append(labels, strings.Join(e.Label, " "))
	}
	if got := strings.Join(labels, "|"); got != "tekst|gruby|kropki|||opis z o||||" {
		t.Errorf("edge labels: %q", got)
	}
	if g.Edges[1].Style != graph.Thick || g.Edges[2].Style != graph.Dotted || g.Edges[3].EndHead != graph.CircleHead || g.Edges[4].EndHead != graph.CrossHead {
		t.Errorf("edge kinds: %+v %+v %+v %+v", g.Edges[1], g.Edges[2], g.Edges[3], g.Edges[4])
	}
	if g.Node("I").Label[0] != "Użytkownik" || g.Node("J").Label[0] != "database" || g.Node("K").Label[0] != "Zapis gotowy" {
		t.Errorf("icon tags: %q %q %q", g.Node("I").Label, g.Node("J").Label, g.Node("K").Label)
	}
}

// REQ: FLOW-024
// Wielokropek (`...` albo `…`) jest identyfikatorem węzła, także na końcu krawędzi.
func TestParseEllipsisNodes(t *testing.T) {
	g, err := Parse("flowchart TD\n  ...\nA --> ...\n... --> B\nB --> …\nv1.2 --> v1.3")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"...", "…", "v1.2", "v1.3"} {
		if g.Node(id) == nil || g.Node(id).Label[0] != id {
			t.Errorf("node %q is missing", id)
		}
	}
	if len(g.Edges) != 4 {
		t.Errorf("edges: %d", len(g.Edges))
	}
	// Krawędź kropkowana nadal jest krawędzią, a nie węzłem z kropek.
	g, err = Parse("flowchart TD\nA -.-> B\nA -..-> C")
	if err != nil || len(g.Edges) != 2 || g.Edges[0].Style != graph.Dotted || g.Node("...") != nil {
		t.Errorf("dotted edges: %v %+v", err, g)
	}
}

// REQ: FLOW-026
func TestSwimlaneBeta(t *testing.T) {
	g, err := Parse("swimlane-beta LR\n  subgraph a [Pierwszy]\n    x[X]\n    subgraph box [Ramka]\n      y[Y]\n    end\n  end\n  subgraph b [Drugi]\n    z[Z]\n    x\n  end\n  x --> y --> z")
	if err != nil {
		t.Fatal(err)
	}
	if g.Layout != "swimlanes" || g.Direction != graph.LeftRight || len(g.Subgraphs) != 2 {
		t.Fatalf("layout %q, direction %v, %d subgraphs", g.Layout, g.Direction, len(g.Subgraphs))
	}
	for id, lane := range map[string]string{"x": "a", "y": "a", "z": "b"} {
		if n := g.Node(id); n == nil || n.Parent != lane {
			t.Errorf("node %s should stand in lane %s, got %+v", id, lane, n)
		}
	}
	if _, err := Parse("swimlane-beta\n  subgraph a\n    x\n  end\n  x --> a"); err == nil {
		t.Error("an edge to a lane should be an error")
	}
}

// REQ: FLOW-027
func TestChainContinuesOnNextLine(t *testing.T) {
	g, err := Parse("flowchart TD\n  A --> B & C\n  -->|tak| D\n  --> E")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range g.Edges {
		got = append(got, e.From+e.To)
	}
	if strings.Join(got, " ") != "AB AC BD CD DE" {
		t.Errorf("edges %v", got)
	}
	for _, src := range []string{"flowchart TD\n  --> B", "flowchart TD\n  A --> B\n  style A fill:#fff\n  --> C"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("%q: expected an error", src)
		}
	}
}
