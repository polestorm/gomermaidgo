// Package flowchart rozbiera schemat blokowy (flowchart, graph) do modelu grafu.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package flowchart

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni schematu blokowego.
const (
	CodeHeader      = "flowchart-header"
	CodeUnsupported = "flowchart-unsupported"
	CodeStatement   = "flowchart-statement"
	CodeNode        = "flowchart-node"
	CodeEdge        = "flowchart-edge"
	CodeSubgraph    = "flowchart-subgraph"
)

const idPattern = `[\p{L}\p{N}_]+(?:[-.][\p{L}\p{N}_]+)*`

var (
	lineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)
	swimlanes = regexp.MustCompile(`^%%\s*layout:\s*swimlanes\s*$`)
	// Identyfikator: litery, cyfry i podkreślenie; łącznik i kropka tylko między nimi, żeby
	// „A-->B” i „A-.->B” nadal zaczynały krawędź.
	idAt      = regexp.MustCompile(`^` + idPattern)
	classAt   = regexp.MustCompile(`^:::\s*([\p{L}\p{N}_-]+)`)
	styleHead = regexp.MustCompile(`^(\S+)\s+(.+)$`)

	subgraphWithTitle = regexp.MustCompile(`^(` + idPattern + `)\s*\[(.*)\]$`)
	singleID          = regexp.MustCompile(`^` + idPattern + `$`)

	// Krawędzie z tekstem w środku: „-- tekst -->”, „== tekst ==>”, „-. tekst .->”.
	inlineNormal = regexp.MustCompile(`^([<ox])?--\s+(\S.*?)\s*(-{2,})([>ox])?`)
	inlineThick  = regexp.MustCompile(`^([<ox])?==\s+(\S.*?)\s*(={2,})([>ox])?`)
	inlineDotted = regexp.MustCompile(`^([<ox])?-\.\s+(\S.*?)\s*(\.+)-([>ox])?`)
	// Krawędzie bez tekstu.
	plainDotted = regexp.MustCompile(`^([<ox])?-(\.+)-([>ox])?`)
	plainNormal = regexp.MustCompile(`^([<ox])?(-{2,})([>ox])?`)
	plainThick  = regexp.MustCompile(`^([<ox])?(={2,})([>ox])?`)
	invisible   = regexp.MustCompile(`^~{3,}`)

	// Instrukcje Mermaid poza profilem: znane, ale świadomie nieobsługiwane.
	unsupported = []string{"interpolate"}

	// Wyróżnienia w tekście Markdown: pogrubienie i kursywa.
	strong   = regexp.MustCompile(`\*\*([^*]+)\*\*|\b__([^_]+)__\b`)
	emphasis = regexp.MustCompile(`\*([^*]+)\*|\b_([^_]+)_\b`)
)

// shapes: nawias otwierający, zamykający i kształt; dłuższe otwarcia przed krótszymi.
var shapes = []struct {
	open, close string
	shape       graph.Shape
}{
	{"(((", ")))", graph.DoubleCircle},
	{"((", "))", graph.Circle},
	{"([", "])", graph.Stadium},
	{"[[", "]]", graph.Subroutine},
	{"[(", ")]", graph.Cylinder},
	{"[/", "", ""},  // równoległobok albo trapez, zależnie od zamknięcia
	{"[\\", "", ""}, // jak wyżej, odwrócone
	{"{{", "}}", graph.Hexagon},
	{"{", "}", graph.Rhombus},
	{"(", ")", graph.Round},
	{"[", "]", graph.Rect},
	{">", "]", graph.Asymmetric},
}

type parser struct {
	g     *graph.Graph
	stack []string // otwarte podgrafy
	auto  int
	acc   syntax.Accessibility
}

func (p *parser) scope() string {
	if len(p.stack) == 0 {
		return ""
	}
	return p.stack[len(p.stack)-1]
}

// text czyści tekst etykiety: zdejmuje cudzysłowy i dzieli na linie.
func text(raw string, line int) ([]string, error) {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s = s[1 : len(s)-1]
	}
	if strings.HasPrefix(s, "`") {
		// Tekst Markdown w odwrotnych apostrofach: pogrubienie i kursywa stają się znacznikami stylu.
		if len(s) < 2 || !strings.HasSuffix(s, "`") {
			return nil, syntax.Errorf(line, CodeNode, "markdown string opened with a backtick is not closed in %q", raw)
		}
		s = strong.ReplaceAllString(s[1:len(s)-1], string(graph.MarkBold)+"$1$2"+string(graph.MarkBold))
		s = emphasis.ReplaceAllString(s, string(graph.MarkItalic)+"$1$2"+string(graph.MarkItalic))
	}
	if strings.Contains(s, `"`) {
		return nil, syntax.Errorf(line, CodeNode, "unexpected quote in text %q", raw)
	}
	parts := lineBreak.Split(s, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts, nil
}

// splitStatements dzieli linię na instrukcje rozdzielone średnikiem poza cudzysłowami i nawiasami.
func splitStatements(line string) []string {
	var out []string
	depth, quoted, start := 0, false, 0
	for i, c := range line {
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
		case c == ';' && depth == 0:
			out = append(out, line[start:i])
			start = i + 1
		}
	}
	return append(out, line[start:])
}

// joinMarkdownLines skleja tekst Markdown rozciągnięty na kilka linii źródła w jedną linię: koniec
// linii wewnątrz "`…`" jest złamaniem tekstu. Puste linie dopisane za sklejoną zachowują numerację
// linii w komunikatach błędów. Tekst niezamknięty zostaje bez zmian i daje błąd w swojej linii.
func joinMarkdownLines(source string) string {
	if !strings.Contains(source, "\"`") {
		return source
	}
	var b strings.Builder
	inside, pending := false, 0
	for i := 0; i < len(source); i++ {
		switch {
		case !inside && strings.HasPrefix(source[i:], "\"`"):
			inside = true
			b.WriteString("\"`")
			i++
		case inside && strings.HasPrefix(source[i:], "`\""):
			inside = false
			b.WriteString("`\"")
			i++
		case inside && source[i] == '\n':
			b.WriteString("<br/>")
			pending++
		case source[i] == '\n':
			b.WriteString(strings.Repeat("\n", pending+1))
			pending = 0
		default:
			b.WriteByte(source[i])
		}
	}
	if inside {
		return source
	}
	return b.String()
}

// Parse rozbiera źródło schematu blokowego.
func Parse(source string) (*graph.Graph, error) {
	p := &parser{g: graph.New("flowchart")}
	seenHeader := false
	source = joinMarkdownLines(strings.ReplaceAll(source, "\r\n", "\n"))
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if swimlanes.MatchString(s) {
			// Dyrektywa w komentarzu: inne narzędzia ją pomijają i rysują zwykłe podgrafy.
			p.g.Layout = "swimlanes"
			continue
		}
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if err := p.header(strings.TrimRight(s, "; "), line); err != nil {
				return nil, err
			}
			seenHeader = true
			continue
		}
		if p.acc.Line(s) {
			continue
		}
		for _, st := range splitStatements(s) {
			st = strings.TrimSpace(st)
			if st == "" {
				continue
			}
			if err := p.statement(st, line); err != nil {
				return nil, err
			}
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"flowchart\" or \"graph\"")
	}
	if len(p.stack) > 0 {
		return nil, syntax.Errorf(0, CodeSubgraph, "subgraph %q is not closed with \"end\"", p.scope())
	}
	if p.acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	p.g.AccTitle, p.g.AccDescr = p.acc.Title, p.acc.Descr
	if err := p.g.Finish(); err != nil {
		return nil, err
	}
	if p.g.Layout == "swimlanes" {
		// Tor jest pasem przez cały rysunek: nie da się go zagnieździć ani do niego poprowadzić krawędzi.
		for _, s := range p.g.Subgraphs {
			if s.Parent != "" {
				return nil, syntax.Errorf(s.Line, CodeSubgraph, "swimlanes do not support nested subgraphs (%q)", s.ID)
			}
		}
		for _, e := range p.g.Edges {
			if p.g.Subgraph(e.From) != nil || p.g.Subgraph(e.To) != nil {
				return nil, syntax.Errorf(e.Line, CodeEdge, "in swimlanes an edge must connect nodes, not a lane")
			}
		}
	}
	return p.g, nil
}

func (p *parser) header(s string, line int) error {
	fields := strings.Fields(s)
	if len(fields) == 0 || (fields[0] != "flowchart" && fields[0] != "graph") {
		return syntax.Errorf(line, CodeHeader, "expected \"flowchart\" or \"graph\", found %q", s)
	}
	switch len(fields) {
	case 1:
		return nil
	case 2:
		d, ok := graph.ParseDirection(fields[1])
		if !ok {
			return syntax.Errorf(line, CodeHeader, "unknown direction %q (use TB, TD, BT, LR or RL)", fields[1])
		}
		p.g.Direction = d
		return nil
	}
	return syntax.Errorf(line, CodeHeader, "unexpected text after the direction: %q", strings.Join(fields[2:], " "))
}

func (p *parser) statement(s string, line int) error {
	word, rest := s, ""
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		word, rest = s[:i], strings.TrimSpace(s[i+1:])
	}
	switch {
	case s == "end":
		if len(p.stack) == 0 {
			return syntax.Errorf(line, CodeSubgraph, "\"end\" without an open subgraph")
		}
		p.stack = p.stack[:len(p.stack)-1]
		return nil
	case word == "subgraph":
		return p.subgraph(rest, line)
	case word == "direction":
		d, ok := graph.ParseDirection(rest)
		if !ok {
			return syntax.Errorf(line, CodeStatement, "unknown direction %q", rest)
		}
		if len(p.stack) == 0 {
			return syntax.Errorf(line, CodeStatement, "\"direction\" is only allowed inside a subgraph")
		}
		p.g.Subgraph(p.scope()).Direction = d
		return nil
	case word == "click":
		return nil // odnośniki i wywołania nie mają znaczenia na rysunku statycznym
	case word == "classDef" || word == "style" || word == "linkStyle":
		m := styleHead.FindStringSubmatch(rest)
		if m == nil {
			return syntax.Errorf(line, CodeStatement, "%s needs a target and a list of properties", word)
		}
		paint, err := graph.ParsePaint(m[2], line)
		if err != nil {
			return err
		}
		switch word {
		case "classDef":
			p.g.DefineClass(m[1], paint)
		case "style":
			p.g.SetStyle(m[1], paint, line)
		default:
			p.g.SetLinkStyle(m[1], paint, line)
		}
		return nil
	case word == "class":
		m := styleHead.FindStringSubmatch(rest)
		if m == nil {
			return syntax.Errorf(line, CodeStatement, "class needs node ids and a class name")
		}
		p.g.AssignClass(m[1], m[2], line)
		return nil
	}
	for _, u := range unsupported {
		if word == u {
			return syntax.Errorf(line, CodeUnsupported, "%q is outside the supported profile", word)
		}
	}
	return p.chain(s, line)
}

func (p *parser) subgraph(rest string, line int) error {
	var id string
	var title []string
	var err error
	switch m := subgraphWithTitle.FindStringSubmatch(rest); {
	case rest == "":
		return syntax.Errorf(line, CodeSubgraph, "subgraph needs an id or a title")
	case m != nil:
		id = m[1]
		title, err = text(m[2], line)
	case singleID.MatchString(rest):
		id, title = rest, []string{rest}
	default:
		p.auto++
		id = "subgraph#" + strconv.Itoa(p.auto)
		title, err = text(rest, line)
	}
	if err != nil {
		return err
	}
	if err := p.g.AddSubgraph(&graph.Subgraph{ID: id, Title: title, Parent: p.scope(), Line: line}); err != nil {
		return err
	}
	p.stack = append(p.stack, id)
	return nil
}

// scanner czyta jedną instrukcję: łańcuch grup węzłów połączonych krawędziami.
type scanner struct {
	s    string
	pos  int
	line int
}

func (sc *scanner) skipSpace() {
	for sc.pos < len(sc.s) && (sc.s[sc.pos] == ' ' || sc.s[sc.pos] == '\t') {
		sc.pos++
	}
}

func (sc *scanner) rest() string { return sc.s[sc.pos:] }

func (p *parser) chain(s string, line int) error {
	sc := &scanner{s: s, line: line}
	from, err := p.group(sc)
	if err != nil {
		return err
	}
	for {
		sc.skipSpace()
		if sc.pos >= len(sc.s) {
			return nil
		}
		edge, err := sc.edge()
		if err != nil {
			return err
		}
		to, err := p.group(sc)
		if err != nil {
			return err
		}
		for _, a := range from {
			for _, b := range to {
				e := *edge
				e.From, e.To = a, b
				if err := p.g.AddEdge(&e); err != nil {
					return err
				}
			}
		}
		from = to
	}
}

// group czyta węzły połączone „&”.
func (p *parser) group(sc *scanner) ([]string, error) {
	var ids []string
	for {
		id, err := p.node(sc)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
		sc.skipSpace()
		if !strings.HasPrefix(sc.rest(), "&") {
			return ids, nil
		}
		sc.pos++
	}
}

func (p *parser) node(sc *scanner) (string, error) {
	sc.skipSpace()
	id := idAt.FindString(sc.rest())
	if id == "" {
		if sc.pos >= len(sc.s) {
			return "", syntax.Errorf(sc.line, CodeNode, "expected a node at the end of %q", sc.s)
		}
		return "", syntax.Errorf(sc.line, CodeNode, "expected a node id, found %q", sc.rest())
	}
	sc.pos += len(id)
	n, err := p.g.EnsureNode(id, p.scope(), graph.Rect, sc.line)
	if err != nil {
		return "", err
	}
	rest := sc.rest()
	if strings.HasPrefix(rest, "@{") {
		return "", syntax.Errorf(sc.line, CodeUnsupported, "shape syntax \"@{ }\" is outside the supported profile")
	}
	// Klasa może stać przed kształtem albo za nim: „A:::k[tekst]”, „A[tekst]:::k”.
	class := func() {
		if m := classAt.FindStringSubmatch(sc.rest()); m != nil {
			p.g.AssignClass(id, m[1], sc.line)
			sc.pos += len(m[0])
		}
	}
	class()
	defer class()
	rest = sc.rest()
	for _, sh := range shapes {
		if !strings.HasPrefix(rest, sh.open) {
			continue
		}
		body := rest[len(sh.open):]
		shape, closeAt, closeLen := sh.shape, -1, len(sh.close)
		switch {
		case strings.HasPrefix(strings.TrimLeft(body, " \t"), `"`):
			// Tekst w cudzysłowie może zawierać nawiasy: zamknięcia szukamy za drugim cudzysłowem.
			first := strings.Index(body, `"`)
			second := strings.Index(body[first+1:], `"`)
			if second < 0 {
				return "", syntax.Errorf(sc.line, CodeNode, "unterminated quote in node %q", id)
			}
			after := first + 1 + second + 1
			tail := strings.TrimLeft(body[after:], " \t")
			closeAt = len(body) - len(tail)
			if sh.close == "" {
				shape, closeLen = slanted(sh.open, tail)
			} else if !strings.HasPrefix(tail, sh.close) {
				closeAt = -1
			}
		case sh.close == "":
			a, b := strings.Index(body, "/]"), strings.Index(body, "\\]")
			closeAt = a
			if a < 0 || (b >= 0 && b < a) {
				closeAt = b
			}
			if closeAt >= 0 {
				shape, closeLen = slanted(sh.open, body[closeAt:])
			}
		default:
			closeAt = strings.Index(body, sh.close)
		}
		if closeAt < 0 || shape == "" {
			return "", syntax.Errorf(sc.line, CodeNode, "node %q: shape opened with %q is not closed", id, sh.open)
		}
		label, err := text(body[:closeAt], sc.line)
		if err != nil {
			return "", err
		}
		n.Label, n.Shape, n.Explicit = label, shape, true
		sc.pos += len(sh.open) + closeAt + closeLen
		break
	}
	return id, nil
}

// slanted rozpoznaje równoległobok i trapez po parze nawiasów ukośnych.
func slanted(open, closing string) (graph.Shape, int) {
	forward := strings.HasPrefix(closing, "/]")
	if !forward && !strings.HasPrefix(closing, "\\]") {
		return "", 0
	}
	switch {
	case open == "[/" && forward:
		return graph.Parallelogram, 2
	case open == "[/":
		return graph.Trapezoid, 2
	case forward:
		return graph.TrapezoidAlt, 2
	}
	return graph.ParallelogramAlt, 2
}

func head(mark string) graph.Head {
	switch mark {
	case "<", ">":
		return graph.ArrowHead
	case "o":
		return graph.CircleHead
	case "x":
		return graph.CrossHead
	}
	return graph.NoHead
}

// edge czyta krawędź z pozycji skanera, razem z etykietą w „|…|”.
func (sc *scanner) edge() (*graph.Edge, error) {
	rest := sc.rest()
	e := &graph.Edge{Line: sc.line, StartHead: graph.NoHead, EndHead: graph.NoHead}
	fail := func() (*graph.Edge, error) {
		return nil, syntax.Errorf(sc.line, CodeEdge, "expected an edge, found %q", rest)
	}
	// length liczy minimalną długość krawędzi z liczby znaków linii: „-->” i „---” to 1.
	length := func(count int, hasHead bool) int {
		if hasHead {
			return count - 1
		}
		return count - 2
	}
	var label string
	switch {
	case inlineNormal.MatchString(rest), inlineThick.MatchString(rest):
		re, style := inlineNormal, graph.Normal
		if !inlineNormal.MatchString(rest) {
			re, style = inlineThick, graph.Thick
		}
		m := re.FindStringSubmatch(rest)
		if m[4] == "" && len(m[3]) < 3 {
			return fail()
		}
		e.Style, e.StartHead, e.EndHead, label = style, head(m[1]), head(m[4]), m[2]
		e.MinLen = length(len(m[3]), m[4] != "")
		sc.pos += len(m[0])
	case inlineDotted.MatchString(rest):
		m := inlineDotted.FindStringSubmatch(rest)
		e.Style, e.StartHead, e.EndHead, label = graph.Dotted, head(m[1]), head(m[4]), m[2]
		e.MinLen = len(m[3])
		sc.pos += len(m[0])
	case plainDotted.MatchString(rest):
		m := plainDotted.FindStringSubmatch(rest)
		e.Style, e.StartHead, e.EndHead, e.MinLen = graph.Dotted, head(m[1]), head(m[3]), len(m[2])
		sc.pos += len(m[0])
	case plainNormal.MatchString(rest), plainThick.MatchString(rest):
		re, style := plainNormal, graph.Normal
		if !plainNormal.MatchString(rest) {
			re, style = plainThick, graph.Thick
		}
		m := re.FindStringSubmatch(rest)
		if m[3] == "" && len(m[2]) < 3 {
			return fail()
		}
		e.Style, e.StartHead, e.EndHead = style, head(m[1]), head(m[3])
		e.MinLen = length(len(m[2]), m[3] != "")
		sc.pos += len(m[0])
	case invisible.MatchString(rest):
		m := invisible.FindString(rest)
		e.Style, e.MinLen = graph.Invisible, len(m)-2
		sc.pos += len(m)
	default:
		return fail()
	}
	if e.StartHead == graph.ArrowHead && e.EndHead != graph.ArrowHead {
		return nil, syntax.Errorf(sc.line, CodeEdge, "an edge starting with \"<\" must end with \">\"")
	}
	sc.skipSpace()
	if strings.HasPrefix(sc.rest(), "|") {
		if label != "" {
			return nil, syntax.Errorf(sc.line, CodeEdge, "edge has two labels")
		}
		body := sc.rest()[1:]
		end := strings.Index(body, "|")
		if trimmed := strings.TrimLeft(body, " \t"); strings.HasPrefix(trimmed, `"`) {
			// Etykieta w cudzysłowie może zawierać kreskę pionową: zamknięcia szukamy za drugim cudzysłowem.
			skip := len(body) - len(trimmed) + 1
			quote := strings.Index(body[skip:], `"`)
			end = -1
			if quote >= 0 {
				if bar := strings.Index(body[skip+quote:], "|"); bar >= 0 {
					end = skip + quote + bar
				}
			}
		}
		if end < 0 {
			return nil, syntax.Errorf(sc.line, CodeEdge, "edge label opened with \"|\" is not closed")
		}
		label = sc.rest()[1 : 1+end]
		sc.pos += end + 2
	}
	if strings.TrimSpace(label) != "" {
		lines, err := text(label, sc.line)
		if err != nil {
			return nil, err
		}
		e.Label = lines
	}
	return e, nil
}
