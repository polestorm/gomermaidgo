// Package wardley rozbiera i rysuje mapę Wardleya: składniki łańcucha wartości ułożone według
// widoczności (oś pionowa) i ewolucji (oś pozioma), zależności, ewolucję i siły.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package wardley

import (
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni mapy Wardleya.
const (
	CodeHeader    = "wardley-header"
	CodeStatement = "wardley-statement"
	maxElements   = 300
)

// Pos to położenie: widoczność (0 na dole, 1 na górze) i ewolucja (0 z lewej, 1 z prawej).
type Pos struct{ Vis, Evo float64 }

// Component to składnik albo kotwica (użytkownik).
type Component struct {
	Name       string
	At         Pos
	Anchor     bool
	LabelDX    float64 // przesunięcie podpisu w pikselach
	LabelDY    float64
	Decorators []string // build, buy, outsource, market, inertia
	Pipeline   string   // nazwa rodzica, gdy składnik leży w potoku
	Line       int
}

// Link to zależność albo przepływ między składnikami.
type Link struct {
	From, To   string
	Dashed     bool
	StartArrow bool
	EndArrow   bool
	Label      string
	Line       int
}

// Evolve to przewidziany ruch składnika na osi ewolucji.
type Evolve struct {
	Name string
	To   float64
	Line int
}

// Trend to przewidziane położenie składnika (x = ewolucja, y = widoczność).
type Trend struct {
	Name string
	At   Pos
	Line int
}

// Text to notatka, adnotacja albo siła w danym położeniu.
type Text struct {
	Kind   string // note, annotation, accelerator, deaccelerator
	Number int    // numer adnotacji
	Text   string
	At     Pos
	Line   int
}

// Stage to etap ewolucji: nazwa (może mieć dwie linie, „A / B”) i prawa granica.
type Stage struct {
	Names []string
	Until float64
}

// Map to cała mapa.
type Map struct {
	Title       string
	Width       float64 // rozmiar z „size”, 0 = domyślny
	Height      float64
	Components  []*Component
	Links       []Link
	Evolves     []Evolve
	Trends      []Trend
	Texts       []Text
	Stages      []Stage
	Annotations *Pos // położenie listy adnotacji
	AccTitle    string
	AccDescr    string
}

// Component zwraca składnik o nazwie albo nil.
func (m *Map) Component(name string) *Component {
	for _, c := range m.Components {
		if c.Name == name {
			return c
		}
	}
	return nil
}

var (
	position   = syntax.Lazy(`^\[\s*(-?[\d.]+)\s*,\s*(-?[\d.]+)\s*\]`)
	single     = syntax.Lazy(`^\[\s*(-?[\d.]+)\s*\]`)
	labelShift = syntax.Lazy(`^label\s*\[\s*(-?[\d.]+)\s*,\s*(-?[\d.]+)\s*\]`)
	decorator  = syntax.Lazy(`^\(\s*(build|buy|outsource|market|inertia)\s*\)`)
	linkOp     = syntax.Lazy(`\s(-\.->|-->|->|\+<>|\+<|\+'[^']*'>|\+>)\s`)
	trendOp    = syntax.Lazy(`^(.+?)\s+-\.-\s+\(\s*(-?[\d.]+)\s*,\s*(-?[\d.]+)\s*\)$`)
	annotation = syntax.Lazy(`^annotation\s+(\d+)\s*,\s*\[\s*(-?[\d.]+)\s*,\s*(-?[\d.]+)\s*\]\s*"([^"]*)"$`)
	sizeRule   = syntax.Lazy(`^size\s*\[\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*\]$`)
)

// Parse rozbiera źródło.
func Parse(source string) (*Map, error) {
	m := &Map{}
	var acc syntax.Accessibility
	seenHeader := false
	pipeline := ""
	count := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(stripComment(raw))
		if s == "" {
			continue
		}
		if !seenHeader {
			if s != "wardley-beta" && s != "wardley" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"wardley-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		if count++; count > maxElements {
			return nil, syntax.Errorf(line, CodeStatement, "too many statements (limit %d)", maxElements)
		}
		if pipeline != "" {
			if s == "}" {
				pipeline = ""
				continue
			}
			if !strings.HasPrefix(s, "component ") {
				return nil, syntax.Errorf(line, CodeStatement, "a pipeline holds only component lines")
			}
			name, rest, err := nameAndRest(strings.TrimSpace(s[10:]), line)
			if err != nil {
				return nil, err
			}
			v := single.FindStringSubmatch(rest)
			if v == nil {
				return nil, syntax.Errorf(line, CodeStatement, "a pipeline component takes one evolution value: [0.5]")
			}
			evo, err := unit(v[1], line)
			if err != nil {
				return nil, err
			}
			parent := m.Component(pipeline)
			if err := m.add(&Component{Name: name, At: Pos{parent.At.Vis, evo}, Pipeline: pipeline, Line: line}, line); err != nil {
				return nil, err
			}
			continue
		}
		word, rest, _ := strings.Cut(s, " ")
		rest = strings.TrimSpace(rest)
		switch word {
		case "title":
			m.Title = unquote(rest)
			continue
		case "size":
			v := sizeRule.FindStringSubmatch(s)
			if v == nil {
				return nil, syntax.Errorf(line, CodeStatement, "write size [width, height]")
			}
			m.Width, _ = strconv.ParseFloat(v[1], 64)
			m.Height, _ = strconv.ParseFloat(v[2], 64)
			continue
		case "component", "anchor":
			c, err := parseComponent(rest, word == "anchor", line)
			if err != nil {
				return nil, err
			}
			if err := m.add(c, line); err != nil {
				return nil, err
			}
			continue
		case "pipeline":
			name := strings.TrimSpace(strings.TrimSuffix(rest, "{"))
			name = unquote(name)
			if !strings.HasSuffix(rest, "{") {
				return nil, syntax.Errorf(line, CodeStatement, "write pipeline Name {")
			}
			if m.Component(name) == nil {
				return nil, syntax.Errorf(line, CodeStatement, "pipeline %q needs a component of that name first", name)
			}
			pipeline = name
			continue
		case "evolve":
			i := strings.LastIndex(rest, " ")
			if i < 0 {
				return nil, syntax.Errorf(line, CodeStatement, "write evolve Name 0.8")
			}
			to, err := unit(rest[i+1:], line)
			if err != nil {
				return nil, err
			}
			m.Evolves = append(m.Evolves, Evolve{unquote(strings.TrimSpace(rest[:i])), to, line})
			continue
		case "evolution":
			for _, part := range strings.Split(rest, "->") {
				part = strings.TrimSpace(part)
				st := Stage{Until: -1}
				if i := strings.LastIndex(part, "@"); i >= 0 {
					v, err := unit(part[i+1:], line)
					if err != nil {
						return nil, err
					}
					st.Until, part = v, strings.TrimSpace(part[:i])
				}
				for _, nm := range strings.Split(part, "/") {
					st.Names = append(st.Names, strings.TrimSpace(nm))
				}
				m.Stages = append(m.Stages, st)
			}
			if len(m.Stages) < 2 {
				return nil, syntax.Errorf(line, CodeStatement, "evolution needs at least two stages")
			}
			continue
		case "note", "accelerator", "deaccelerator":
			text, pos, err := quotedAt(rest, line)
			if err != nil {
				return nil, err
			}
			m.Texts = append(m.Texts, Text{Kind: word, Text: text, At: pos, Line: line})
			continue
		case "annotations":
			v := position.FindStringSubmatch(rest)
			if v == nil {
				return nil, syntax.Errorf(line, CodeStatement, "write annotations [visibility, evolution]")
			}
			p, err := pos(v, line)
			if err != nil {
				return nil, err
			}
			m.Annotations = &p
			continue
		case "annotation":
			v := annotation.FindStringSubmatch(s)
			if v == nil {
				return nil, syntax.Errorf(line, CodeStatement, "write annotation 1,[visibility, evolution] \"text\"")
			}
			num, _ := strconv.Atoi(v[1])
			p, err := pos([]string{"", v[2], v[3]}, line)
			if err != nil {
				return nil, err
			}
			m.Texts = append(m.Texts, Text{Kind: "annotation", Number: num, Text: v[4], At: p, Line: line})
			continue
		}
		if v := trendOp.FindStringSubmatch(s); v != nil {
			x, err := unit(v[2], line)
			if err != nil {
				return nil, err
			}
			y, err := unit(v[3], line)
			if err != nil {
				return nil, err
			}
			m.Trends = append(m.Trends, Trend{unquote(strings.TrimSpace(v[1])), Pos{y, x}, line})
			continue
		}
		if loc := linkOp.FindStringSubmatchIndex(s); loc != nil {
			op := s[loc[2]:loc[3]]
			l := Link{From: unquote(strings.TrimSpace(s[:loc[0]])), Line: line}
			to := strings.TrimSpace(s[loc[1]:])
			if j := semicolon(to); j >= 0 {
				to, l.Label = strings.TrimSpace(to[:j]), strings.TrimSpace(to[j+1:])
			}
			l.To = unquote(to)
			switch {
			case op == "-.->":
				l.Dashed, l.EndArrow = true, true
			case op == "+>":
				l.EndArrow = true
			case op == "+<":
				l.StartArrow = true
			case op == "+<>":
				l.StartArrow, l.EndArrow = true, true
			case strings.HasPrefix(op, "+'"):
				l.EndArrow, l.Label = true, op[2:len(op)-2]
			}
			m.Links = append(m.Links, l)
			continue
		}
		return nil, syntax.Errorf(line, CodeStatement, "unknown statement %q", s)
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"wardley-beta\"")
	}
	if pipeline != "" {
		return nil, syntax.Errorf(0, CodeStatement, "pipeline %q is not closed with \"}\"", pipeline)
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	for _, l := range m.Links {
		for _, end := range []string{l.From, l.To} {
			if m.Component(end) == nil {
				return nil, syntax.Errorf(l.Line, CodeStatement, "%q is not a component", end)
			}
		}
	}
	for _, e := range m.Evolves {
		if m.Component(e.Name) == nil {
			return nil, syntax.Errorf(e.Line, CodeStatement, "evolve: %q is not a component", e.Name)
		}
	}
	for _, t := range m.Trends {
		if m.Component(t.Name) == nil {
			return nil, syntax.Errorf(t.Line, CodeStatement, "trend: %q is not a component", t.Name)
		}
	}
	m.AccTitle, m.AccDescr = acc.Title, acc.Descr
	return m, nil
}

func (m *Map) add(c *Component, line int) error {
	if m.Component(c.Name) != nil {
		return syntax.Errorf(line, CodeStatement, "component %q is defined twice", c.Name)
	}
	m.Components = append(m.Components, c)
	return nil
}

// stripComment usuwa komentarz %% z końca linii (poza cudzysłowem).
func stripComment(s string) string {
	quoted := false
	for i := 0; i+1 < len(s); i++ {
		switch {
		case s[i] == '"':
			quoted = !quoted
		case !quoted && s[i] == '%' && s[i+1] == '%':
			return s[:i]
		}
	}
	return s
}

// parseComponent czyta „Nazwa [vis, evo] [label [dx, dy]] [(dekorator)…]”.
func parseComponent(s string, anchor bool, line int) (*Component, error) {
	name, rest, err := nameAndRest(s, line)
	if err != nil {
		return nil, err
	}
	v := position.FindStringSubmatch(rest)
	if v == nil {
		return nil, syntax.Errorf(line, CodeStatement, "component %q needs a position [visibility, evolution]", name)
	}
	p, err := pos(v, line)
	if err != nil {
		return nil, err
	}
	c := &Component{Name: name, At: p, Anchor: anchor, Line: line}
	rest = strings.TrimSpace(rest[len(v[0]):])
	for rest != "" {
		if v := labelShift.FindStringSubmatch(rest); v != nil {
			c.LabelDX, _ = strconv.ParseFloat(v[1], 64)
			c.LabelDY, _ = strconv.ParseFloat(v[2], 64)
			rest = strings.TrimSpace(rest[len(v[0]):])
			continue
		}
		if v := decorator.FindStringSubmatch(rest); v != nil {
			c.Decorators = append(c.Decorators, v[1])
			rest = strings.TrimSpace(rest[len(v[0]):])
			continue
		}
		return nil, syntax.Errorf(line, CodeStatement, "unexpected %q after the position", rest)
	}
	return c, nil
}

// nameAndRest dzieli „Nazwa reszta”: nazwa w cudzysłowie albo tekst do nawiasu kwadratowego.
func nameAndRest(s string, line int) (string, string, error) {
	if strings.HasPrefix(s, `"`) {
		end := strings.Index(s[1:], `"`)
		if end < 0 {
			return "", "", syntax.Errorf(line, CodeStatement, "quoted name is not closed")
		}
		return s[1 : 1+end], strings.TrimSpace(s[2+end:]), nil
	}
	i := strings.Index(s, "[")
	if i <= 0 {
		return "", "", syntax.Errorf(line, CodeStatement, "a name and a position are needed: %q", s)
	}
	return strings.TrimSpace(s[:i]), s[i:], nil
}

// quotedAt czyta „"tekst" [vis, evo]”.
func quotedAt(s string, line int) (string, Pos, error) {
	if !strings.HasPrefix(s, `"`) {
		return "", Pos{}, syntax.Errorf(line, CodeStatement, "the text must be in quotes")
	}
	text, rest, err := nameAndRest(s, line)
	if err != nil {
		return "", Pos{}, err
	}
	v := position.FindStringSubmatch(rest)
	if v == nil || strings.TrimSpace(rest[len(v[0]):]) != "" {
		return "", Pos{}, syntax.Errorf(line, CodeStatement, "a position [visibility, evolution] is needed after the text")
	}
	p, err := pos(v, line)
	return text, p, err
}

func pos(v []string, line int) (Pos, error) {
	a, err := unit(v[1], line)
	if err != nil {
		return Pos{}, err
	}
	b, err := unit(v[2], line)
	return Pos{a, b}, err
}

// unit czyta liczbę z przedziału od 0 do 1.
func unit(s string, line int) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 || v > 1 {
		return 0, syntax.Errorf(line, CodeStatement, "%q is not a number from 0 to 1", s)
	}
	return v, nil
}

// semicolon zwraca położenie pierwszego średnika poza cudzysłowem albo -1.
func semicolon(s string) int {
	quoted := false
	for i, r := range s {
		switch {
		case r == '"':
			quoted = !quoted
		case r == ';' && !quoted:
			return i
		}
	}
	return -1
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
