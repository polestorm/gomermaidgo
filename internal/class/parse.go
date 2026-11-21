// Package class rozbiera diagram klas (classDiagram) do modelu grafu.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package class

import (
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramu klas.
const (
	CodeHeader      = "class-header"
	CodeUnsupported = "class-unsupported"
	CodeStatement   = "class-statement"
	CodeMember      = "class-member"
	CodeClass       = "class-class"
	maxMembers      = 100
)

// Nazwa klasy: litery, cyfry i podkreślenie, z kropką albo łącznikiem w środku; opcjonalnie
// parametr typu w tyldach (Lista~T~).
const name = `[\p{L}\p{N}_](?:[\p{L}\p{N}_.-]*[\p{L}\p{N}_])?(?:~[^~]+~)?`

var (
	lineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)

	relation = regexp.MustCompile(`^(` + name + `)\s*(?:"([^"]*)")?\s*(<\||\*|o|<|\(\))?(--|\.\.)(\|>|\*|o|>|\(\))?\s*(?:"([^"]*)")?\s*(` + name + `)\s*(?::\s*(.*))?$`)
	declare  = regexp.MustCompile(`^class\s+(` + name + `)\s*(?:\[\s*"([^"]*)"\s*\])?\s*(?::::\s*([\p{L}\p{N}_-]+))?\s*(\{)?\s*(\})?$`)
	colon    = regexp.MustCompile(`^(` + name + `)\s*:\s*(.+)$`)
	annotate = regexp.MustCompile(`^<<\s*([^<>]+?)\s*>>\s*(` + name + `)?$`)
	noteFor  = regexp.MustCompile(`^note\s+for\s+(` + name + `)\s+"([^"]*)"$`)
	cssClass = regexp.MustCompile(`^cssClass\s+"([^"]+)"\s+([\p{L}\p{N}_-]+)$`)
	target   = regexp.MustCompile(`^(\S+)\s+(.+)$`)
	// Klasyfikator składowej: gwiazdka (abstrakcyjna) albo dolar (statyczna), po nawiasie metody
	// albo na końcu tekstu.
	classifier = regexp.MustCompile(`^(.*\))([*$])(.*)$|^(.*?)([*$])$`)
	returns    = regexp.MustCompile(`^(.*\))\s+(\S.*)$`)

	leftHeads  = map[string]graph.Head{"<|": graph.Triangle, "*": graph.DiamondFilled, "o": graph.DiamondHollow, "<": graph.OpenArrow, "": graph.NoHead}
	rightHeads = map[string]graph.Head{"|>": graph.Triangle, "*": graph.DiamondFilled, "o": graph.DiamondHollow, ">": graph.OpenArrow, "": graph.NoHead}
)

type parser struct {
	g    *graph.Graph
	acc  syntax.Accessibility
	open *graph.Node // klasa, której blok składowych jest otwarty
}

// generics zamienia zapis parametrów typu w tyldach na nawiasy ostre: Lista~T~ → Lista<T>.
func generics(s string) string {
	var b strings.Builder
	opened := false
	for _, c := range s {
		if c != '~' {
			b.WriteRune(c)
			continue
		}
		if opened {
			b.WriteByte('>')
		} else {
			b.WriteByte('<')
		}
		opened = !opened
	}
	if opened {
		return s // tylda bez pary zostaje dosłownie
	}
	return b.String()
}

func split(text string) []string {
	parts := lineBreak.Split(strings.TrimSpace(text), -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Parse rozbiera źródło diagramu klas.
func Parse(source string) (*graph.Graph, error) {
	p := &parser{g: graph.New("class")}
	seenHeader := false
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "classDiagram" && s != "classDiagram-v2" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"classDiagram\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if p.open == nil && p.acc.Line(s) {
			continue
		}
		if err := p.statement(s, line); err != nil {
			return nil, err
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"classDiagram\"")
	}
	if p.open != nil {
		return nil, syntax.Errorf(p.open.Line, CodeClass, "member block of %q is not closed with \"}\"", p.open.ID)
	}
	if p.acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	p.g.AccTitle, p.g.AccDescr = p.acc.Title, p.acc.Descr
	if err := p.g.Finish(); err != nil {
		return nil, err
	}
	return p.g, nil
}

// class zwraca klasę o podanej nazwie ze źródła; identyfikatorem jest nazwa bez parametru typu.
func (p *parser) class(written string, line int) (*graph.Node, error) {
	id := written
	if i := strings.Index(written, "~"); i >= 0 {
		id = written[:i]
	}
	n, err := p.g.EnsureNode(id, "", graph.Class, line)
	if err != nil {
		return nil, err
	}
	if len(n.Label) == 0 || (id != written && n.Label[0] == id) {
		n.Label = []string{generics(written)}
	}
	return n, nil
}

// member dopisuje klasie składową albo adnotację.
func (p *parser) member(n *graph.Node, text string, line int) error {
	text = strings.TrimSpace(text)
	if m := annotate.FindStringSubmatch(text); m != nil && m[2] == "" {
		n.Annotation = m[1]
		return nil
	}
	if text == "" || strings.ContainsAny(text, "{}") {
		return syntax.Errorf(line, CodeMember, "member of %q cannot be read: %q", n.ID, text)
	}
	if len(n.Members) >= maxMembers {
		return syntax.Errorf(line, CodeMember, "too many members in %q (limit %d)", n.ID, maxMembers)
	}
	m := graph.Member{Method: strings.Contains(text, "(")}
	abstract := false
	if c := classifier.FindStringSubmatch(text); c != nil {
		mark := c[2] + c[5]
		text = strings.TrimSpace(c[1] + c[3] + c[4])
		abstract, m.Static = mark == "*", mark == "$"
	}
	if r := returns.FindStringSubmatch(text); m.Method && r != nil && !strings.HasPrefix(r[2], ":") {
		text = r[1] + " : " + r[2]
	}
	m.Text = generics(text)
	if abstract {
		m.Text = string(graph.MarkItalic) + m.Text + string(graph.MarkItalic)
	}
	n.Members = append(n.Members, m)
	return nil
}

func (p *parser) statement(s string, line int) error {
	if p.open != nil {
		if s == "}" {
			p.open = nil
			return nil
		}
		return p.member(p.open, s, line)
	}
	word, rest := s, ""
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		word, rest = s[:i], strings.TrimSpace(s[i+1:])
	}
	switch word {
	case "direction":
		d, ok := graph.ParseDirection(rest)
		if !ok {
			return syntax.Errorf(line, CodeStatement, "unknown direction %q", rest)
		}
		p.g.Direction = d
		return nil
	case "classDef", "style":
		m := target.FindStringSubmatch(rest)
		if m == nil {
			return syntax.Errorf(line, CodeStatement, "%s needs a target and a list of properties", word)
		}
		paint, err := graph.ParsePaint(m[2], line)
		if err != nil {
			return err
		}
		if word == "classDef" {
			p.g.DefineClass(m[1], paint)
		} else {
			p.g.SetStyle(m[1], paint, line)
		}
		return nil
	case "cssClass":
		m := cssClass.FindStringSubmatch(s)
		if m == nil {
			return syntax.Errorf(line, CodeStatement, "cssClass needs quoted class names and a style name")
		}
		p.g.AssignClass(m[1], m[2], line)
		return nil
	case "click", "link", "callback":
		return nil // rysunek nie ma odsyłaczy
	case "namespace":
		return syntax.Errorf(line, CodeUnsupported, "namespaces are outside the supported profile")
	case "note":
		m := noteFor.FindStringSubmatch(s)
		if m == nil {
			return syntax.Errorf(line, CodeUnsupported, "only notes written as: note for Class \"text\" are supported")
		}
		n, err := p.class(m[1], line)
		if err != nil {
			return err
		}
		p.g.Notes = append(p.g.Notes, &graph.Note{Target: n.ID, Side: "right", Text: split(strings.ReplaceAll(m[2], `\n`, "<br/>")), Line: line})
		return nil
	case "class":
		m := declare.FindStringSubmatch(s)
		if m == nil {
			return syntax.Errorf(line, CodeClass, "class declaration cannot be read: %q", s)
		}
		n, err := p.class(m[1], line)
		if err != nil {
			return err
		}
		if label := strings.TrimSpace(m[2]); label != "" {
			n.Label = split(label)
		}
		if m[3] != "" {
			p.g.AssignClass(n.ID, m[3], line)
		}
		if m[4] != "" && m[5] == "" {
			p.open = n
			p.open.Line = line
		}
		return nil
	}
	if m := annotate.FindStringSubmatch(s); m != nil && m[2] != "" {
		n, err := p.class(m[2], line)
		if err != nil {
			return err
		}
		n.Annotation = m[1]
		return nil
	}
	if m := relation.FindStringSubmatch(s); m != nil {
		if m[3] == "()" || m[5] == "()" {
			return syntax.Errorf(line, CodeUnsupported, "lollipop interfaces are outside the supported profile")
		}
		from, err := p.class(m[1], line)
		if err != nil {
			return err
		}
		to, err := p.class(m[7], line)
		if err != nil {
			return err
		}
		e := &graph.Edge{From: from.ID, To: to.ID, Style: graph.Normal, StartHead: leftHeads[m[3]], EndHead: rightHeads[m[5]],
			StartLabel: strings.TrimSpace(m[2]), EndLabel: strings.TrimSpace(m[6]), Line: line}
		if m[4] == ".." {
			e.Style = graph.Dotted
		}
		if label := strings.TrimSpace(m[8]); label != "" {
			e.Label = split(label)
		}
		return p.g.AddEdge(e)
	}
	if m := colon.FindStringSubmatch(s); m != nil {
		n, err := p.class(m[1], line)
		if err != nil {
			return err
		}
		return p.member(n, m[2], line)
	}
	return syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
}
