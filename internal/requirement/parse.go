// Package requirement rozbiera diagram wymagań (requirementDiagram) do modelu grafu.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package requirement

import (
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramu wymagań.
const (
	CodeHeader    = "requirement-header"
	CodeStatement = "requirement-statement"
	CodeBlock     = "requirement-block"
	CodeField     = "requirement-field"
	CodeRelation  = "requirement-relation"
)

const name = `"[^"]+"|[\p{L}\p{N}_][\p{L}\p{N}_.-]*`

var (
	block    = regexp.MustCompile(`^(\w+)\s+(` + name + `)\s*(?::::\s*([\p{L}\p{N}_,-]+))?\s*\{$`)
	fieldRow = regexp.MustCompile(`^(\w+)\s*:\s*(.*)$`)
	forward  = regexp.MustCompile(`^(` + name + `)\s*-\s*(\w+)\s*->\s*(` + name + `)$`)
	backward = regexp.MustCompile(`^(` + name + `)\s*<-\s*(\w+)\s*-\s*(` + name + `)$`)
	target   = regexp.MustCompile(`^(\S+)\s+(.+)$`)

	// Rodzaje bloków i ich nazwy na rysunku.
	kinds = map[string]string{
		"requirement": "Requirement", "functionalrequirement": "Functional Requirement", "interfacerequirement": "Interface Requirement",
		"performancerequirement": "Performance Requirement", "physicalrequirement": "Physical Requirement", "designconstraint": "Design Constraint",
		"element": "Element",
	}
	relations = map[string]bool{"contains": true, "copies": true, "derives": true, "satisfies": true, "verifies": true, "refines": true, "traces": true}
	risks     = map[string]string{"low": "Low", "medium": "Medium", "high": "High"}
	methods   = map[string]string{"analysis": "Analysis", "inspection": "Inspection", "test": "Test", "demonstration": "Demonstration"}
)

func unquote(s string) string { return strings.Trim(strings.TrimSpace(s), `"`) }

// wrap łamie długą wartość pola na linie o podobnej długości w znakach; kolejne linie są wcięte.
func wrap(label, value string) []string {
	const limit = 44
	words := strings.Fields(value)
	lines := []string{label + ":"}
	for _, w := range words {
		last := &lines[len(lines)-1]
		if len([]rune(*last))+1+len([]rune(w)) > limit && *last != label+":" {
			lines = append(lines, "  "+w)
			continue
		}
		*last += " " + w
	}
	return lines
}

// Parse rozbiera źródło diagramu wymagań.
func Parse(source string) (*graph.Graph, error) {
	g := graph.New("requirement")
	var acc syntax.Accessibility
	seenHeader := false
	var open *graph.Node
	element := false
	node := func(written string, line int) (*graph.Node, error) {
		n, err := g.EnsureNode(unquote(written), "", graph.Requirement, line)
		if err != nil {
			return nil, err
		}
		if len(n.Label) == 0 {
			n.Label = []string{unquote(written)}
		}
		return n, nil
	}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "requirementDiagram" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"requirementDiagram\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if open != nil {
			if s == "}" {
				open = nil
				continue
			}
			m := fieldRow.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeField, "expected a field written as name: value, found %q", s)
			}
			value := unquote(m[2])
			label := ""
			switch key := strings.ToLower(m[1]); {
			case !element && key == "id":
				label = "Id"
			case !element && key == "text":
				label = "Text"
			case !element && key == "risk":
				if label, value = "Risk", risks[strings.ToLower(value)]; value == "" {
					return nil, syntax.Errorf(line, CodeField, "risk is Low, Medium or High, found %q", unquote(m[2]))
				}
			case !element && key == "verifymethod":
				if label, value = "Verification", methods[strings.ToLower(value)]; value == "" {
					return nil, syntax.Errorf(line, CodeField, "verifymethod is Analysis, Inspection, Test or Demonstration, found %q", unquote(m[2]))
				}
			case element && key == "type":
				label = "Type"
			case element && key == "docref":
				label = "Doc Ref"
			default:
				return nil, syntax.Errorf(line, CodeField, "unknown field %q", m[1])
			}
			for _, text := range wrap(label, value) {
				open.Members = append(open.Members, graph.Member{Text: text})
			}
			continue
		}
		if acc.Line(s) {
			continue
		}
		word, rest := s, ""
		if i := strings.IndexAny(s, " \t"); i >= 0 {
			word, rest = s[:i], strings.TrimSpace(s[i+1:])
		}
		switch word {
		case "direction":
			d, ok := graph.ParseDirection(rest)
			if !ok {
				return nil, syntax.Errorf(line, CodeStatement, "unknown direction %q", rest)
			}
			g.Direction = d
			continue
		case "classDef", "style":
			m := target.FindStringSubmatch(rest)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "%s needs a target and a list of properties", word)
			}
			paint, err := graph.ParsePaint(m[2], line)
			if err != nil {
				return nil, err
			}
			if word == "classDef" {
				g.DefineClass(m[1], paint)
			} else {
				g.SetStyle(m[1], paint, line)
			}
			continue
		case "class":
			m := target.FindStringSubmatch(rest)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "class needs names and a class name")
			}
			g.AssignClass(m[1], m[2], line)
			continue
		}
		if m := block.FindStringSubmatch(s); m != nil {
			kind, ok := kinds[strings.ToLower(m[1])]
			if !ok {
				return nil, syntax.Errorf(line, CodeBlock, "unknown kind of block %q", m[1])
			}
			nd, err := node(m[2], line)
			if err != nil {
				return nil, err
			}
			if nd.Annotation != "" {
				return nil, syntax.Errorf(line, CodeBlock, "%q is defined twice", nd.ID)
			}
			nd.Annotation, nd.Line = kind, line
			for _, class := range strings.Split(m[3], ",") {
				if class != "" {
					g.AssignClass(nd.ID, class, line)
				}
			}
			open, element = nd, kind == "Element"
			continue
		}
		from, relation, to := "", "", ""
		if m := forward.FindStringSubmatch(s); m != nil {
			from, relation, to = m[1], m[2], m[3]
		} else if m := backward.FindStringSubmatch(s); m != nil {
			from, relation, to = m[3], m[2], m[1]
		} else {
			return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
		}
		if !relations[strings.ToLower(relation)] {
			return nil, syntax.Errorf(line, CodeRelation, "unknown relationship %q", relation)
		}
		a, err := node(from, line)
		if err != nil {
			return nil, err
		}
		b, err := node(to, line)
		if err != nil {
			return nil, err
		}
		e := &graph.Edge{From: a.ID, To: b.ID, Style: graph.Normal, StartHead: graph.NoHead, EndHead: graph.OpenArrow,
			Label: []string{"«" + strings.ToLower(relation) + "»"}, Line: line}
		if strings.EqualFold(relation, "contains") {
			// Zawieranie: kółko przy elemencie zawierającym, bez grotu.
			e.StartHead, e.EndHead = graph.CircleHead, graph.NoHead
		}
		if err := g.AddEdge(e); err != nil {
			return nil, err
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"requirementDiagram\"")
	}
	if open != nil {
		return nil, syntax.Errorf(open.Line, CodeBlock, "block of %q is not closed with \"}\"", open.ID)
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	for _, n := range g.Nodes {
		if n.Annotation == "" {
			return nil, syntax.Errorf(n.Line, CodeRelation, "%q is used in a relationship but never defined", n.ID)
		}
	}
	g.AccTitle, g.AccDescr = acc.Title, acc.Descr
	if err := g.Finish(); err != nil {
		return nil, err
	}
	return g, nil
}
