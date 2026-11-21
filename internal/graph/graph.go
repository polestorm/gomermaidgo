// Package graph to wspólny model diagramów grafowych: schematu blokowego i diagramu stanów.
// Parsery budują ten model, układ warstwowy go rozmieszcza.
package graph

import (
	"fmt"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Limity chroniące układ przed wejściem, którego nie da się sensownie narysować.
const (
	MaxNodes    = 500
	MaxEdges    = 1000
	MaxNesting  = 10
	CodeLimit   = "graph-limit"
	CodeInvalid = "graph-invalid"
)

// Direction to kierunek układu.
type Direction string

const (
	TopBottom Direction = "TB"
	BottomTop Direction = "BT"
	LeftRight Direction = "LR"
	RightLeft Direction = "RL"
)

// ParseDirection czyta kierunek; „TD” jest synonimem „TB”.
func ParseDirection(s string) (Direction, bool) {
	switch s {
	case "TB", "TD":
		return TopBottom, true
	case "BT":
		return BottomTop, true
	case "LR":
		return LeftRight, true
	case "RL":
		return RightLeft, true
	}
	return "", false
}

// Shape to kształt węzła.
type Shape string

const (
	Rect             Shape = "rect"
	Round            Shape = "round"
	Stadium          Shape = "stadium"
	Subroutine       Shape = "subroutine"
	Cylinder         Shape = "cylinder"
	Circle           Shape = "circle"
	DoubleCircle     Shape = "double-circle"
	Asymmetric       Shape = "asymmetric"
	Rhombus          Shape = "rhombus"
	Hexagon          Shape = "hexagon"
	Parallelogram    Shape = "parallelogram"
	ParallelogramAlt Shape = "parallelogram-alt"
	Trapezoid        Shape = "trapezoid"
	TrapezoidAlt     Shape = "trapezoid-alt"
	StateStart       Shape = "state-start"
	StateEnd         Shape = "state-end"
	Choice           Shape = "choice"
	Fork             Shape = "fork"
	Join             Shape = "join"
	Entity           Shape = "entity" // encja diagramu związków: nagłówek i tabela atrybutów
)

// LineStyle to rodzaj linii krawędzi.
type LineStyle string

const (
	Normal    LineStyle = "normal"
	Thick     LineStyle = "thick"
	Dotted    LineStyle = "dotted"
	Invisible LineStyle = "invisible"
)

// Head to zakończenie krawędzi.
type Head string

const (
	NoHead     Head = "none"
	ArrowHead  Head = "arrow"
	CircleHead Head = "circle"
	CrossHead  Head = "cross"

	// Liczności związku w diagramie związków encji (notacja „kurzej stopki”).
	OnlyOne    Head = "only-one"
	ZeroOrOne  Head = "zero-or-one"
	OneOrMore  Head = "one-or-more"
	ZeroOrMore Head = "zero-or-more"
)

// Node to węzeł. Parent to identyfikator podgrafu albo pusty napis.
type Node struct {
	ID     string
	Label  []string
	Shape  Shape
	Parent string
	Line   int

	// Explicit mówi, że źródło podało kształt albo opis węzła (a nie tylko identyfikator).
	Explicit bool

	Paint Paint // wygląd narzucony przez źródło

	// Table to wiersze tabeli atrybutów encji: typ, nazwa, klucze, komentarz.
	Table [][]string
}

// Edge to krawędź. From i To wskazują węzeł albo podgraf.
type Edge struct {
	From, To  string
	Style     LineStyle
	StartHead Head
	EndHead   Head
	MinLen    int // minimalna liczba warstw między końcami, od 1
	Label     []string
	Line      int
	Paint     Paint // wygląd narzucony przez źródło (linkStyle)
}

// Subgraph to podgraf schematu blokowego albo stan złożony.
type Subgraph struct {
	ID        string
	Title     []string
	Parent    string
	Direction Direction
	Line      int
	Paint     Paint // wygląd narzucony przez źródło

	// Region oznacza region współbieżny stanu złożonego (separator „--”): podgraf podrzędny
	// bez tytułu, rysowany linią przerywaną.
	Region bool
}

// Note to notatka przy węźle (diagram stanów).
type Note struct {
	Target string
	Side   string // "left" albo "right"
	Text   []string
	Line   int
}

// Graph to cały diagram.
type Graph struct {
	Kind      string // "flowchart" albo "state"
	Direction Direction
	Nodes     []*Node
	Edges     []*Edge
	Subgraphs []*Subgraph
	Notes     []*Note
	AccTitle  string // tytuł dostępności (accTitle)
	AccDescr  string // opis dostępności (accDescr)

	classes  map[string]Paint
	uses     []classUse
	linkUses []classUse

	nodeIndex map[string]*Node
	subIndex  map[string]*Subgraph
}

// New tworzy pusty graf.
func New(kind string) *Graph {
	return &Graph{Kind: kind, Direction: TopBottom, nodeIndex: map[string]*Node{}, subIndex: map[string]*Subgraph{}}
}

// Node zwraca węzeł o podanym identyfikatorze albo nil.
func (g *Graph) Node(id string) *Node { return g.nodeIndex[id] }

// Subgraph zwraca podgraf o podanym identyfikatorze albo nil.
func (g *Graph) Subgraph(id string) *Subgraph { return g.subIndex[id] }

// EnsureNode zwraca istniejący węzeł albo tworzy nowy z etykietą równą identyfikatorowi.
// Węzeł bez rodzica, wspomniany wewnątrz podgrafu, staje się jego członkiem.
func (g *Graph) EnsureNode(id, parent string, shape Shape, line int) (*Node, error) {
	if n := g.nodeIndex[id]; n != nil {
		if n.Parent == "" && parent != "" {
			n.Parent = parent
		}
		return n, nil
	}
	if len(g.Nodes) >= MaxNodes {
		return nil, syntax.Errorf(line, CodeLimit, "too many nodes (limit %d)", MaxNodes)
	}
	n := &Node{ID: id, Label: []string{id}, Shape: shape, Parent: parent, Line: line}
	g.Nodes = append(g.Nodes, n)
	g.nodeIndex[id] = n
	return n, nil
}

// AddEdge dopisuje krawędź.
func (g *Graph) AddEdge(e *Edge) error {
	if len(g.Edges) >= MaxEdges {
		return syntax.Errorf(e.Line, CodeLimit, "too many edges (limit %d)", MaxEdges)
	}
	if e.MinLen < 1 {
		e.MinLen = 1
	}
	g.Edges = append(g.Edges, e)
	return nil
}

// AddSubgraph dopisuje podgraf.
func (g *Graph) AddSubgraph(s *Subgraph) error {
	if g.subIndex[s.ID] != nil {
		return syntax.Errorf(s.Line, CodeInvalid, "subgraph %q is already defined", s.ID)
	}
	depth := 1
	for p := s.Parent; p != ""; p = g.subIndex[p].Parent {
		depth++
	}
	if depth > MaxNesting {
		return syntax.Errorf(s.Line, CodeLimit, "subgraphs nested too deeply (limit %d)", MaxNesting)
	}
	g.Subgraphs = append(g.Subgraphs, s)
	g.subIndex[s.ID] = s
	return nil
}

// Finish zamyka budowę grafu: identyfikator użyty i jako węzeł, i jako podgraf oznacza
// podgraf (krawędź prowadzi wtedy do ramki), więc taki węzeł znika. Sprawdza też spójność.
func (g *Graph) Finish() error {
	kept := g.Nodes[:0]
	for _, n := range g.Nodes {
		s := g.subIndex[n.ID]
		if s == nil {
			kept = append(kept, n)
			continue
		}
		if n.Explicit && g.Kind == "flowchart" {
			return syntax.Errorf(n.Line, CodeInvalid, "%q is used both as a node and as a subgraph", n.ID)
		}
		if n.Explicit && len(s.Title) == 1 && s.Title[0] == s.ID {
			s.Title = n.Label // opis stanu podany osobno od definicji stanu złożonego
		}
		delete(g.nodeIndex, n.ID)
	}
	g.Nodes = kept

	exists := func(id string) bool { return g.nodeIndex[id] != nil || g.subIndex[id] != nil }
	inside := func(id, ancestor string) bool {
		p := ""
		if n := g.nodeIndex[id]; n != nil {
			p = n.Parent
		} else if s := g.subIndex[id]; s != nil {
			p = s.Parent
		}
		for ; p != ""; p = g.subIndex[p].Parent {
			if p == ancestor {
				return true
			}
		}
		return false
	}
	for _, n := range g.Nodes {
		if n.Parent != "" && g.subIndex[n.Parent] == nil {
			return syntax.Errorf(n.Line, CodeInvalid, "node %q refers to unknown subgraph %q", n.ID, n.Parent)
		}
	}
	for _, e := range g.Edges {
		if !exists(e.From) || !exists(e.To) {
			return syntax.Errorf(e.Line, CodeInvalid, "edge refers to an unknown node")
		}
		// Krawędź między ramką a jej własną zawartością nie ma sensownego rysunku.
		if g.subIndex[e.From] != nil && (e.From == e.To || inside(e.To, e.From)) ||
			g.subIndex[e.To] != nil && inside(e.From, e.To) {
			return syntax.Errorf(e.Line, CodeInvalid, "edge connects %q with its own contents", pick(g.subIndex[e.From] != nil, e.From, e.To))
		}
	}
	for _, n := range g.Notes {
		if !exists(n.Target) {
			return syntax.Errorf(n.Line, CodeInvalid, "note refers to unknown state %q", n.Target)
		}
	}
	if len(g.Nodes) == 0 && len(g.Subgraphs) == 0 {
		return syntax.Errorf(0, CodeInvalid, "diagram has no nodes")
	}
	return g.resolvePaint()
}

func pick(first bool, a, b string) string {
	if first {
		return a
	}
	return b
}

func quote(lines []string) string { return fmt.Sprintf("%q", showMarks(strings.Join(lines, "\n"))) }

// Dump zapisuje graf jako tekst w stałej kolejności; służy za plik wzorcowy parsera.
func (g *Graph) Dump() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", g.Kind, g.Direction)
	for _, s := range g.Subgraphs {
		fmt.Fprintf(&b, "subgraph %s title=%s", s.ID, quote(s.Title))
		if s.Parent != "" {
			fmt.Fprintf(&b, " in=%s", s.Parent)
		}
		if s.Direction != "" {
			fmt.Fprintf(&b, " dir=%s", s.Direction)
		}
		if s.Region {
			b.WriteString(" region")
		}
		b.WriteString(s.Paint.dump())
		b.WriteByte('\n')
	}
	for _, n := range g.Nodes {
		fmt.Fprintf(&b, "node %s %s %s", n.ID, n.Shape, quote(n.Label))
		if n.Parent != "" {
			fmt.Fprintf(&b, " in=%s", n.Parent)
		}
		b.WriteString(n.Paint.dump())
		b.WriteByte('\n')
		for _, row := range n.Table {
			fmt.Fprintf(&b, "  attribute %q\n", row)
		}
	}
	for _, e := range g.Edges {
		fmt.Fprintf(&b, "edge %s -> %s %s %s/%s len=%d", e.From, e.To, e.Style, e.StartHead, e.EndHead, e.MinLen)
		if len(e.Label) > 0 {
			fmt.Fprintf(&b, " %s", quote(e.Label))
		}
		b.WriteString(e.Paint.dump())
		b.WriteByte('\n')
	}
	for _, n := range g.Notes {
		fmt.Fprintf(&b, "note %s of %s %s\n", n.Side, n.Target, quote(n.Text))
	}
	return b.String()
}

// Texts zwraca wszystkie teksty, które mają trafić na rysunek, linia po linii.
func (g *Graph) Texts() []string {
	var out []string
	add := func(lines []string) {
		for _, s := range lines {
			if s = Plain(s); s != "" {
				out = append(out, s)
			}
		}
	}
	for _, s := range g.Subgraphs {
		add(s.Title)
	}
	for _, n := range g.Nodes {
		add(n.Label)
		for _, row := range n.Table {
			add(row)
		}
	}
	for _, e := range g.Edges {
		add(e.Label)
	}
	for _, n := range g.Notes {
		add(n.Text)
	}
	return out
}
