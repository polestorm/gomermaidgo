package class

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: CLS-001 CLS-002 CLS-003
func TestParseClassesAndMembers(t *testing.T) {
	g, err := Parse("%% komentarz\nclassDiagram\nclass Lista~T~ {\n  <<interface>>\n  +int rozmiar\n  -Map~String, T~ indeks$\n  +dodaj(T element) bool\n  +wyczysc()*\n  +utworz()$ Lista~T~\n  +opis() : String\n}\nPusta : +pole\nclass Pusta[\"Z etykietą\"]\n<<service>> Pusta\nclass Sama")
	if err != nil {
		t.Fatal(err)
	}
	l := g.Node("Lista")
	if l == nil || l.Label[0] != "Lista<T>" || l.Annotation != "interface" || l.Shape != graph.Class {
		t.Fatalf("class: %+v", l)
	}
	want := []graph.Member{
		{Text: "+int rozmiar"},
		{Text: "-Map<String, T> indeks", Static: true},
		{Text: "+dodaj(T element) : bool", Method: true},
		{Text: string(graph.MarkItalic) + "+wyczysc()" + string(graph.MarkItalic), Method: true},
		{Text: "+utworz() : Lista<T>", Method: true, Static: true},
		{Text: "+opis() : String", Method: true},
	}
	if len(l.Members) != len(want) {
		t.Fatalf("members: %+v", l.Members)
	}
	for i, m := range l.Members {
		if m != want[i] {
			t.Errorf("member %d: got %+v, want %+v", i, m, want[i])
		}
	}
	if p := g.Node("Pusta"); p.Label[0] != "Z etykietą" || p.Annotation != "service" || len(p.Members) != 1 {
		t.Errorf("class with a label: %+v", p)
	}
	if s := g.Node("Sama"); s == nil || len(s.Members) != 0 {
		t.Errorf("class without members: %+v", s)
	}
}

// REQ: CLS-004 CLS-005
func TestParseRelations(t *testing.T) {
	type rel struct {
		start, end graph.Head
		style      graph.LineStyle
	}
	for src, want := range map[string]rel{
		"A <|-- B": {graph.Triangle, graph.NoHead, graph.Normal},
		"A --|> B": {graph.NoHead, graph.Triangle, graph.Normal},
		"A *-- B":  {graph.DiamondFilled, graph.NoHead, graph.Normal},
		"A --o B":  {graph.NoHead, graph.DiamondHollow, graph.Normal},
		"A --> B":  {graph.NoHead, graph.OpenArrow, graph.Normal},
		"A <--> B": {graph.OpenArrow, graph.OpenArrow, graph.Normal},
		"A -- B":   {graph.NoHead, graph.NoHead, graph.Normal},
		"A ..> B":  {graph.NoHead, graph.OpenArrow, graph.Dotted},
		"A ..|> B": {graph.NoHead, graph.Triangle, graph.Dotted},
		"A <|.. B": {graph.Triangle, graph.NoHead, graph.Dotted},
		"A .. B":   {graph.NoHead, graph.NoHead, graph.Dotted},
		"Foo--Boo": {graph.NoHead, graph.NoHead, graph.Normal},
	} {
		g, err := Parse("classDiagram\n" + src)
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		e := g.Edges[0]
		if (rel{e.StartHead, e.EndHead, e.Style}) != want {
			t.Errorf("%q: got %s/%s %s", src, e.StartHead, e.EndHead, e.Style)
		}
	}
	g, err := Parse("classDiagram\nKlient \"1\" --> \"0..*\" Zamowienie~T~ : składa<br/>wiele")
	if err != nil {
		t.Fatal(err)
	}
	e := g.Edges[0]
	if e.From != "Klient" || e.To != "Zamowienie" || e.StartLabel != "1" || e.EndLabel != "0..*" || strings.Join(e.Label, "|") != "składa|wiele" {
		t.Errorf("relation with cardinalities: %+v", e)
	}
	if g.Node("Zamowienie").Label[0] != "Zamowienie<T>" {
		t.Errorf("generic name from a relation: %q", g.Node("Zamowienie").Label)
	}
}

// REQ: CLS-006 CLS-007
func TestParseNotesStylesDirection(t *testing.T) {
	g, err := Parse("classDiagram\ndirection RL\naccTitle: Tytuł\nclassDef wazne fill:#fde,stroke:#b00\nclass A:::wazne\nclass B\ncssClass \"B\" wazne\nstyle B stroke-width:3px\nnote for A \"pierwsza<br/>druga\"\nclick A href \"https://example.org\"\nlink B \"https://example.org\"\nA --> A")
	if err != nil {
		t.Fatal(err)
	}
	if g.Direction != graph.RightLeft || g.AccTitle != "Tytuł" {
		t.Errorf("direction or title: %s %q", g.Direction, g.AccTitle)
	}
	if a, b := g.Node("A"), g.Node("B"); a.Paint.Fill != "#fde" || b.Paint.Stroke != "#b00" || b.Paint.StrokeWidth != 3 {
		t.Errorf("styles: %+v %+v", a.Paint, b.Paint)
	}
	if len(g.Notes) != 1 || g.Notes[0].Target != "A" || strings.Join(g.Notes[0].Text, "|") != "pierwsza|druga" {
		t.Errorf("note: %+v", g.Notes)
	}
	if len(g.Edges) != 1 || g.Edges[0].From != g.Edges[0].To {
		t.Errorf("self relation: %+v", g.Edges)
	}
}

// REQ: CLS-008
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                            {CodeHeader, 0},
		"classes\nA":                  {CodeHeader, 1},
		"classDiagram\nnamespace N {": {CodeClass, 2},
		"classDiagram\nnamespace N {\nnamespace M {": {CodeUnsupported, 3},
		"classDiagram\nnamespace bez klamry":         {CodeStatement, 2},
		"classDiagram\nnote bez cudzysłowu":          {CodeStatement, 2},
		"classDiagram\nclass A {\n+x":                {CodeClass, 2},
		"classDiagram\nclass A {\n+x {\n}":           {CodeMember, 3},
		"classDiagram\nclass":                        {CodeClass, 2},
		"classDiagram\nto nie jest instrukcja":       {CodeStatement, 2},
		"classDiagram\ndirection XY":                 {CodeStatement, 2},
		"classDiagram\ncssClass A b":                 {CodeStatement, 2},
		"classDiagram\nclass A\nstyle A fill:%%%":    {"graph-style", 3},
		"classDiagram\naccDescr {\nbez końca":        {CodeStatement, 0},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || (want.code != "graph-style" && e.Code != want.code) || e.Line != want.line {
			t.Errorf("%q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: CLS-009
// Przestrzeń nazw jest ramką wokół swoich klas; blok klasy może stać w jednej linii; notatka bez
// klasy jest osobnym polem; interfejs „lizak” ma kółko przy swoim końcu.
func TestParseNamespacesNotesLollipops(t *testing.T) {
	g, err := Parse("classDiagram\nnamespace Dokumenty {\n  class Procedura {\n    +String numer\n  }\n  class Instrukcja\n}\nclass Pojazd { <<interface>> }\nclass Rejestr { +int licznik }\nnote \"Ogólna<br/>notatka\"\nPodpisywalny ()-- Procedura\nRejestr --() Czytelny")
	if err != nil {
		t.Fatal(err)
	}
	ns := g.Subgraph("namespace Dokumenty")
	if ns == nil || ns.Title[0] != "Dokumenty" || g.Node("Procedura").Parent != ns.ID || g.Node("Instrukcja").Parent != ns.ID || g.Node("Rejestr").Parent != "" {
		t.Errorf("namespace: %+v", ns)
	}
	if p := g.Node("Pojazd"); p.Annotation != "interface" || len(p.Members) != 0 {
		t.Errorf("one-line annotation: %+v", p)
	}
	if r := g.Node("Rejestr"); len(r.Members) != 1 || r.Members[0].Text != "+int licznik" {
		t.Errorf("one-line member: %+v", r)
	}
	note := g.Node("[note] 1")
	if note == nil || note.Shape != "note" || strings.Join(note.Label, "|") != "Ogólna|notatka" {
		t.Errorf("free note: %+v", note)
	}
	if a, b := g.Edges[0], g.Edges[1]; a.StartHead != graph.CircleHead || a.EndHead != graph.NoHead || b.EndHead != graph.CircleHead || b.StartHead != graph.NoHead {
		t.Errorf("lollipops: %+v %+v", a, b)
	}
}

// REQ: CLS-010
// Tekst notatki w cudzysłowach może ciągnąć się przez kilka linii; podziały linii zostają.
func TestParseMultilineNote(t *testing.T) {
	g, err := Parse("classDiagram\n    note \"Pierwsza linia\n    druga linia\n    trzecia.\"\n    A <|-- B")
	if err != nil {
		t.Fatal(err)
	}
	n := g.Node("[note] 1")
	if n == nil || strings.Join(n.Label, "|") != "Pierwsza linia|druga linia|trzecia." || len(g.Edges) != 1 {
		t.Errorf("note: %+v", n)
	}
	if _, err := Parse("classDiagram\nnote \"bez końca\nA <|-- B"); err == nil {
		t.Errorf("an unclosed note should be an error")
	}
}
