// Package er rozbiera diagram związków encji (erDiagram) do modelu grafu.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package er

import (
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramu związków encji.
const (
	CodeHeader      = "er-header"
	CodeUnsupported = "er-unsupported"
	CodeStatement   = "er-statement"
	CodeAttribute   = "er-attribute"
	CodeEntity      = "er-entity"
	maxAttributes   = 100
)

const name = `"[^"]+"|[\p{L}\p{N}_][\p{L}\p{N}_-]*`

var (
	lineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)

	relation  = regexp.MustCompile(`^(` + name + `)\s*(\|o|\|\||\}o|\}\|)(--|\.\.)(o\||\|\||o\{|\|\{)\s*(` + name + `)\s*(?::\s*(.*))?$`)
	entity    = regexp.MustCompile(`^(` + name + `)\s*(?:\[\s*"?([^"\]]*)"?\s*\])?\s*(?::::\s*([\p{L}\p{N}_-]+))?\s*(\{)?\s*(\})?$`)
	attribute = regexp.MustCompile(`^(\S+)\s+(\S+)(?:\s+((?:PK|FK|UK)(?:\s*,\s*(?:PK|FK|UK))*))?(?:\s+"([^"]*)")?$`)
	styleHead = regexp.MustCompile(`^(\S+)\s+(.+)$`)

	heads = map[string]graph.Head{
		"|o": graph.ZeroOrOne, "o|": graph.ZeroOrOne, "||": graph.OnlyOne,
		"}o": graph.ZeroOrMore, "o{": graph.ZeroOrMore, "}|": graph.OneOrMore, "|{": graph.OneOrMore,
	}
)

type parser struct {
	g    *graph.Graph
	acc  syntax.Accessibility
	open *graph.Node // encja, której blok atrybutów jest otwarty
}

func unquote(s string) string { return strings.Trim(s, `"`) }

func split(text string) []string {
	parts := lineBreak.Split(strings.TrimSpace(unquote(strings.TrimSpace(text))), -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Parse rozbiera źródło diagramu związków encji.
func Parse(source string) (*graph.Graph, error) {
	p := &parser{g: graph.New("er")}
	seenHeader := false
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "erDiagram" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"erDiagram\", found %q", s)
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
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"erDiagram\"")
	}
	if p.open != nil {
		return nil, syntax.Errorf(p.open.Line, CodeEntity, "attribute block of %q is not closed with \"}\"", p.open.ID)
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

func (p *parser) entity(id string, line int) (*graph.Node, error) {
	n, err := p.g.EnsureNode(unquote(id), "", graph.Entity, line)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (p *parser) statement(s string, line int) error {
	if p.open != nil {
		if s == "}" {
			p.open = nil
			return nil
		}
		m := attribute.FindStringSubmatch(s)
		if m == nil {
			return syntax.Errorf(line, CodeAttribute, "attribute needs \"type name [PK, FK, UK] [\\\"comment\\\"]\", found %q", s)
		}
		if len(p.open.Table) >= maxAttributes {
			return syntax.Errorf(line, CodeAttribute, "too many attributes in %q (limit %d)", p.open.ID, maxAttributes)
		}
		keys := strings.Join(strings.FieldsFunc(m[3], func(r rune) bool { return r == ',' || r == ' ' }), ", ")
		p.open.Table = append(p.open.Table, []string{m[1], m[2], keys, m[4]})
		return nil
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
		m := styleHead.FindStringSubmatch(rest)
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
	case "class":
		m := styleHead.FindStringSubmatch(rest)
		if m == nil {
			return syntax.Errorf(line, CodeStatement, "class needs entity names and a class name")
		}
		p.g.AssignClass(m[1], m[2], line)
		return nil
	}
	if m := relation.FindStringSubmatch(s); m != nil {
		from, err := p.entity(m[1], line)
		if err != nil {
			return err
		}
		to, err := p.entity(m[5], line)
		if err != nil {
			return err
		}
		e := &graph.Edge{From: from.ID, To: to.ID, Style: graph.Normal, StartHead: heads[m[2]], EndHead: heads[m[4]], Line: line}
		if m[3] == ".." {
			e.Style = graph.Dotted
		}
		if strings.TrimSpace(unquote(strings.TrimSpace(m[6]))) != "" {
			e.Label = split(m[6])
		}
		return p.g.AddEdge(e)
	}
	if m := entity.FindStringSubmatch(s); m != nil {
		n, err := p.entity(m[1], line)
		if err != nil {
			return err
		}
		if alias := strings.TrimSpace(m[2]); alias != "" {
			n.Label = split(alias)
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
	if strings.Contains(s, " to ") || strings.Contains(s, " optionally ") {
		return syntax.Errorf(line, CodeUnsupported, "relationships written in words are outside the supported profile; use symbols such as ||--o{")
	}
	return syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
}
