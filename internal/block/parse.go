// Package block rozbiera i rysuje diagram blokowy (block): bloki ustawione w siatce kolumn.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package block

import (
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni diagramu blokowego.
const (
	CodeHeader    = "block-header"
	CodeStatement = "block-statement"
	CodeBlock     = "block-block"
	CodeEdge      = "block-edge"
	CodeEmpty     = "block-empty"
	maxItems      = 300
	maxDepth      = 8
)

// Kształty bloków.
const (
	Rect     = "rect"
	Round    = "round"
	Stadium  = "stadium"
	Circle   = "circle"
	Diamond  = "diamond"
	Hexagon  = "hexagon"
	Cylinder = "cylinder"
	Arrow    = "arrow" // strzałka blokowa; kierunek w Direction
	Space    = "space" // puste miejsce w siatce
	Group    = "group" // blok złożony z własną siatką
)

// Item to blok, puste miejsce albo blok złożony.
type Item struct {
	ID        string
	Label     []string
	Shape     string
	Direction string // dla strzałki blokowej: right, left, up, down
	Span      int    // szerokość w kolumnach
	Columns   int    // dla bloku złożonego: liczba kolumn; 0 = wszystkie elementy w jednym rzędzie
	Children  []*Item
	Paint     graph.Paint
	Line      int
}

// Edge to połączenie dwóch bloków.
type Edge struct {
	From, To string
	Label    []string
	Arrow    bool
	Dotted   bool
	Line     int
}

// Diagram to diagram blokowy: korzeń jest blokiem złożonym bez ramki.
type Diagram struct {
	Root     *Item
	Edges    []Edge
	AccTitle string
	AccDescr string
}

const id = `[\p{L}\p{N}_][\p{L}\p{N}_.-]*`

var (
	// Kształty w kolejności dopasowywania: dłuższe ograniczniki przed krótszymi.
	shapes = []struct {
		open, close, shape string
	}{
		{`((`, `))`, Circle}, {`([`, `])`, Stadium}, {`[(`, `)]`, Cylinder}, {`{{`, `}}`, Hexagon},
		{`(`, `)`, Round}, {`[`, `]`, Rect}, {`{`, `}`, Diamond},
	}
	arrow      = syntax.Lazy(`^(` + id + `)<\[(.*)\]>\((right|left|up|down|x|y)(?:\s*,\s*(?:right|left|up|down|x|y))*\)(?::(\d+))?$`)
	plain      = syntax.Lazy(`^(` + id + `)(?::(\d+))?$`)
	edge       = syntax.Lazy(`^(` + id + `)\s*(?:--\s*"([^"]*)"\s*)?(-->|---|-\.->|-\.-)\s*(?:\|([^|]*)\|\s*)?(` + id + `)$`)
	blockStart = syntax.Lazy(`^block(?::(` + id + `))?(?::(\d+))?$`)
	styleLine  = syntax.Lazy(`^(\S+)\s+(.+)$`)
	wholeID    = syntax.Lazy(`^` + id + `$`)
)

// tokens dzieli linię na elementy rozdzielone odstępami poza nawiasami i cudzysłowami.
func tokens(s string) []string {
	var out []string
	depth, quoted, start := 0, false, -1
	for i, c := range s {
		switch {
		case c == '"':
			quoted = !quoted
		case quoted:
		case c == '[' || c == '(' || c == '{':
			depth++
		case c == ']' || c == ')' || c == '}':
			if depth > 0 {
				depth--
			}
		}
		if (c == ' ' || c == '\t') && depth == 0 && !quoted {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

// item rozbiera jeden element linii: blok z kształtem, sam identyfikator, puste miejsce albo
// strzałkę blokową.
func item(tok string, line int) (*Item, error) {
	if m := arrow.FindStringSubmatch(tok); m != nil {
		it := &Item{ID: m[1], Label: textwrap.Split(strings.Trim(m[2], `"`)), Shape: Arrow, Direction: m[3], Span: 1, Line: line}
		if m[4] != "" {
			it.Span, _ = strconv.Atoi(m[4])
		}
		return it, nil
	}
	if m := plain.FindStringSubmatch(tok); m != nil {
		it := &Item{ID: m[1], Label: []string{m[1]}, Shape: Rect, Span: 1, Line: line}
		if m[1] == "space" {
			it.ID, it.Label, it.Shape = "", nil, Space
		}
		if m[2] != "" {
			it.Span, _ = strconv.Atoi(m[2])
		}
		return it, nil
	}
	// Identyfikator, kształt z etykietą i opcjonalna szerokość po dwukropku.
	body, span := tok, 1
	if i := strings.LastIndex(tok, ":"); i > 0 && !strings.ContainsAny(tok[i:], `"])}`) {
		if n, err := strconv.Atoi(tok[i+1:]); err == nil {
			body, span = tok[:i], n
		}
	}
	for _, sh := range shapes {
		at := strings.Index(body, sh.open)
		if at <= 0 || !strings.HasSuffix(body, sh.close) || len(body) < at+len(sh.open)+len(sh.close) {
			continue
		}
		name := body[:at]
		if !wholeID.MatchString(name) {
			continue
		}
		label := textwrap.Split(strings.Trim(strings.TrimSpace(body[at+len(sh.open):len(body)-len(sh.close)]), `"`))
		if len(label) == 0 {
			label = []string{name}
		}
		return &Item{ID: name, Label: label, Shape: sh.shape, Span: span, Line: line}, nil
	}
	return nil, syntax.Errorf(line, CodeStatement, "block cannot be read: %q", tok)
}

// Parse rozbiera źródło diagramu blokowego.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{Root: &Item{Shape: Group, Span: 1}}
	var acc syntax.Accessibility
	seenHeader := false
	stack := []*Item{d.Root}
	byID := map[string]*Item{}
	classes := map[string]graph.Paint{}
	type use struct {
		target, class string
		paint         graph.Paint
		line          int
	}
	var uses []use
	count := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "block-beta" && s != "block" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"block-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		current := stack[len(stack)-1]
		word, rest := s, ""
		if i := strings.IndexAny(s, " \t"); i >= 0 {
			word, rest = s[:i], strings.TrimSpace(s[i+1:])
		}
		switch {
		case word == "columns":
			if rest == "auto" {
				current.Columns = 0
				continue
			}
			v, err := strconv.Atoi(rest)
			if err != nil || v < 1 || v > 24 {
				return nil, syntax.Errorf(line, CodeStatement, "columns needs a number from 1 to 24 or auto, found %q", rest)
			}
			current.Columns = v
			continue
		case s == "end":
			if len(stack) == 1 {
				return nil, syntax.Errorf(line, CodeBlock, "\"end\" without an open block")
			}
			stack = stack[:len(stack)-1]
			continue
		case word == "classDef" || word == "style" || word == "class":
			m := styleLine.FindStringSubmatch(rest)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "%s needs a target and a value", word)
			}
			if word == "class" {
				for _, t := range strings.Split(m[1], ",") {
					uses = append(uses, use{target: strings.TrimSpace(t), class: strings.TrimSpace(m[2]), line: line})
				}
				continue
			}
			paint, err := graph.ParsePaint(m[2], line)
			if err != nil {
				return nil, err
			}
			for _, t := range strings.Split(m[1], ",") {
				if word == "classDef" {
					classes[strings.TrimSpace(t)] = paint
				} else {
					uses = append(uses, use{target: strings.TrimSpace(t), paint: paint, line: line})
				}
			}
			continue
		}
		if m := blockStart.FindStringSubmatch(s); m != nil {
			if len(stack) > maxDepth {
				return nil, syntax.Errorf(line, CodeBlock, "blocks are nested too deeply (limit %d)", maxDepth)
			}
			g := &Item{ID: m[1], Shape: Group, Span: 1, Line: line}
			if m[2] != "" {
				g.Span, _ = strconv.Atoi(m[2])
			}
			if g.ID != "" {
				if byID[g.ID] != nil {
					return nil, syntax.Errorf(line, CodeBlock, "block %q is defined twice", g.ID)
				}
				byID[g.ID] = g
			}
			current.Children = append(current.Children, g)
			stack = append(stack, g)
			continue
		}
		if m := edge.FindStringSubmatch(s); m != nil {
			label := m[2]
			if label == "" {
				label = strings.Trim(strings.TrimSpace(m[4]), `"`)
			}
			d.Edges = append(d.Edges, Edge{From: m[1], To: m[5], Label: textwrap.Split(label), Arrow: strings.HasSuffix(m[3], ">"), Dotted: strings.Contains(m[3], "."), Line: line})
			continue
		}
		for _, tok := range tokens(s) {
			it, err := item(tok, line)
			if err != nil {
				return nil, err
			}
			if it.Span < 1 || it.Span > 24 {
				return nil, syntax.Errorf(line, CodeStatement, "width of %q must be from 1 to 24 columns", tok)
			}
			if count++; count > maxItems {
				return nil, syntax.Errorf(line, CodeStatement, "too many blocks (limit %d)", maxItems)
			}
			if it.ID != "" {
				if byID[it.ID] != nil {
					return nil, syntax.Errorf(line, CodeBlock, "block %q is defined twice", it.ID)
				}
				byID[it.ID] = it
			}
			current.Children = append(current.Children, it)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"block-beta\"")
	}
	if len(stack) > 1 {
		return nil, syntax.Errorf(stack[len(stack)-1].Line, CodeBlock, "block is not closed with \"end\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(d.Root.Children) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the diagram needs at least one block")
	}
	for _, e := range d.Edges {
		for _, end := range []string{e.From, e.To} {
			if byID[end] == nil {
				return nil, syntax.Errorf(e.Line, CodeEdge, "edge refers to unknown block %q", end)
			}
		}
	}
	for _, u := range uses {
		it := byID[u.target]
		if it == nil {
			return nil, syntax.Errorf(u.line, CodeStatement, "style refers to unknown block %q", u.target)
		}
		paint := u.paint
		if u.class != "" {
			var ok bool
			if paint, ok = classes[u.class]; !ok {
				return nil, syntax.Errorf(u.line, CodeStatement, "unknown class %q", u.class)
			}
		}
		if paint.Fill != "" {
			it.Paint.Fill = paint.Fill
		}
		if paint.Stroke != "" {
			it.Paint.Stroke = paint.Stroke
		}
		if paint.Color != "" {
			it.Paint.Color = paint.Color
		}
		if paint.StrokeWidth != 0 {
			it.Paint.StrokeWidth = paint.StrokeWidth
		}
		if paint.Dash != "" {
			it.Paint.Dash = paint.Dash
		}
	}
	d.AccTitle, d.AccDescr = acc.Title, acc.Descr
	return d, nil
}
