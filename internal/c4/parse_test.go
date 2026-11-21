package c4

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: C4D-001 C4D-002 C4D-003
func TestParse(t *testing.T) {
	g, err := Parse("%% komentarz\nC4Container\ntitle Tytuł diagramu\nPerson(u, \"Użytkownik\", \"Opis, z przecinkiem\")\nSystem_Boundary(b, \"Granica\") {\n  Container(app, \"Aplikacja\", \"Go\", \"Robi rzeczy\", $tags=\"x\")\n  ContainerDb_Ext(db, \"Baza\", \"SQL\")\n  Boundary(w, \"Wewnętrzna\", \"strefa\") {\n    ComponentQueue(q, \"Kolejka\")\n  }\n}\nDeployment_Node(n, \"Serwer\", \"Linux\") {\n  System(s, \"System\")\n}\nRel(u, app, \"Używa\", \"HTTPS\")\nBiRel(app, db, \"Czyta\")\nRel_Back(app, q, \"Odbiera\")\nRelIndex(1, s, app, \"Woła\")\nRel_U(u, b, \"Do granicy\")\nUpdateElementStyle(u, $fontColor=\"red\")")
	if err != nil {
		t.Fatal(err)
	}
	label := func(id string) string { return strings.Join(g.Node(id).Label, "|") }
	bold := string(graph.MarkBold)
	if label("u") != bold+"Użytkownik"+bold+"|«Person»|Opis, z przecinkiem" || g.Node("u").Shape != graph.Stadium {
		t.Errorf("person: %q %s", label("u"), g.Node("u").Shape)
	}
	if label("app") != bold+"Aplikacja"+bold+"|«Container: Go»|Robi rzeczy" || g.Node("app").Parent != "b" {
		t.Errorf("container: %q in %q", label("app"), g.Node("app").Parent)
	}
	if db := g.Node("db"); label("db") != bold+"Baza"+bold+"|«External Container: SQL»" || db.Shape != graph.Cylinder || db.Paint.Dash == "" {
		t.Errorf("external database: %q %+v", label("db"), db)
	}
	if q := g.Node("q"); q.Parent != "w" || q.Shape != graph.Round || label("q") != bold+"Kolejka"+bold+"|«Component»" {
		t.Errorf("queue in a nested boundary: %+v", q)
	}
	title := func(id string) string { return strings.Join(g.Subgraph(id).Title, "|") }
	if title("b") != "Granica|[System]" || title("w") != "Wewnętrzna|[strefa]" || g.Subgraph("w").Parent != "b" || title("n") != "Serwer|[Linux]" {
		t.Errorf("boundaries: %q %q %q", title("b"), title("w"), title("n"))
	}
	frame := g.Subgraph("c4 title")
	if frame == nil || frame.Title[0] != "Tytuł diagramu" || g.Subgraph("b").Parent != "c4 title" || g.Node("u").Parent != "c4 title" || g.AccTitle != "Tytuł diagramu" {
		t.Errorf("title frame: %+v", frame)
	}
	if len(g.Edges) != 5 {
		t.Fatalf("%d edges", len(g.Edges))
	}
	if e := g.Edges[0]; strings.Join(e.Label, "|") != "Używa|[HTTPS]" || e.EndHead != graph.ArrowHead || e.StartHead != graph.NoHead {
		t.Errorf("Rel: %+v", e)
	}
	if e := g.Edges[1]; e.StartHead != graph.ArrowHead || e.EndHead != graph.ArrowHead {
		t.Errorf("BiRel: %+v", e)
	}
	if e := g.Edges[2]; e.StartHead != graph.ArrowHead || e.EndHead != graph.NoHead {
		t.Errorf("Rel_Back: %+v", e)
	}
	if e := g.Edges[3]; e.From != "s" || e.To != "app" || e.Label[0] != "Woła" {
		t.Errorf("RelIndex: %+v", e)
	}
	if e := g.Edges[4]; e.To != "b" {
		t.Errorf("relation to a boundary: %+v", e)
	}
}

// REQ: C4D-004
func TestParseErrors(t *testing.T) {
	head := "C4Context\nPerson(a, \"A\")\n"
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                              {CodeHeader, 0},
		"C4\nPerson(a, \"A\")":                          {CodeHeader, 1},
		head + "to nie jest wywołanie":                  {CodeStatement, 3},
		head + "Robot(r, \"R\")":                        {CodeStatement, 3},
		head + "System(s)":                              {CodeArguments, 3},
		head + "System(s, \"niezamknięty)":              {CodeArguments, 3},
		head + "Person(a, \"Drugi raz\")":               {CodeArguments, 3},
		head + "Rel(a, nieznany, \"x\")":                {CodeArguments, 3},
		head + "Rel(a, a)":                              {CodeArguments, 3},
		head + "Boundary(b, \"B\")":                     {CodeBoundary, 3},
		head + "Boundary(b, \"B\") {\nSystem(s, \"S\")": {CodeBoundary, 3},
		head + "}":                                      {CodeBoundary, 3},
		head + "System(s, \"S\") {":                     {CodeBoundary, 3},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.50q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}
