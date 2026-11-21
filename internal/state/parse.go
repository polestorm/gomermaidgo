// Package state rozbiera diagram stanów (stateDiagram-v2, stateDiagram) do modelu grafu.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package state

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramu stanów.
const (
	CodeHeader      = "state-header"
	CodeUnsupported = "state-unsupported"
	CodeStatement   = "state-statement"
	CodeComposite   = "state-composite"
	CodeNote        = "state-note"
)

// Identyfikator: litery, cyfry i podkreślenie; łącznik i kropka tylko między nimi.
const idPattern = `[\p{L}\p{N}_]+(?:[-.][\p{L}\p{N}_]+)*`

var (
	lineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)

	transition  = regexp.MustCompile(`^(\[\*\]|` + idPattern + `)\s*-->\s*(\[\*\]|` + idPattern + `)\s*(?::\s*(.*))?$`)
	described   = regexp.MustCompile(`^state\s+"([^"]*)"\s+as\s+(` + idPattern + `)\s*(\{)?$`)
	stereotype  = regexp.MustCompile(`^state\s+(` + idPattern + `)\s+<<(choice|fork|join)>>$`)
	declared    = regexp.MustCompile(`^state\s+(` + idPattern + `)\s*(\{)?$`)
	description = regexp.MustCompile(`^(` + idPattern + `)\s*:\s*(.*)$`)
	bareID      = regexp.MustCompile(`^` + idPattern + `$`)
	noteLine    = regexp.MustCompile(`(?i)^note\s+(left|right)\s+of\s+(` + idPattern + `)\s*(?::\s*(.*))?$`)

	// Instrukcje Mermaid poza profilem: znane, ale świadomie nieobsługiwane.
	unsupported = []string{"scale", "hide"}

	classUse  = regexp.MustCompile(`(` + idPattern + `):::\s*([\p{L}\p{N}_-]+)`)
	styleHead = regexp.MustCompile(`^(\S+)\s+(.+)$`)
)

func split(text string) []string {
	parts := lineBreak.Split(text, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

type parser struct {
	g     *graph.Graph
	stack []string // otwarte stany złożone

	note     *graph.Note // notatka wieloliniowa w trakcie czytania
	noteLine int

	acc     syntax.Accessibility
	regions map[string]int // liczba regionów współbieżnych stanu złożonego
}

// region otwiera kolejny region współbieżny stanu złożonego (separator „--”). Przy pierwszym
// separatorze dotychczasowa zawartość stanu staje się regionem pierwszym.
func (p *parser) region(line int) error {
	scope := p.scope()
	if scope == "" {
		return syntax.Errorf(line, CodeComposite, "\"--\" is only allowed inside a composite state")
	}
	owner := scope
	if s := p.g.Subgraph(scope); s.Region {
		owner = s.Parent
		p.stack = p.stack[:len(p.stack)-1]
	} else {
		first := owner + "#1"
		if err := p.g.AddSubgraph(&graph.Subgraph{ID: first, Parent: owner, Region: true, Line: s.Line}); err != nil {
			return err
		}
		for _, n := range p.g.Nodes {
			if n.Parent == owner {
				n.Parent = first
			}
		}
		for _, sub := range p.g.Subgraphs {
			if sub.Parent == owner && sub.ID != first {
				sub.Parent = first
			}
		}
		p.regions[owner] = 1
	}
	p.regions[owner]++
	id := owner + "#" + strconv.Itoa(p.regions[owner])
	if err := p.g.AddSubgraph(&graph.Subgraph{ID: id, Parent: owner, Region: true, Line: line}); err != nil {
		return err
	}
	p.stack = append(p.stack, id)
	return nil
}

func (p *parser) scope() string {
	if len(p.stack) == 0 {
		return ""
	}
	return p.stack[len(p.stack)-1]
}

// Parse rozbiera źródło diagramu stanów.
func Parse(source string) (*graph.Graph, error) {
	p := &parser{g: graph.New("state"), regions: map[string]int{}}
	seenHeader := false
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if p.note != nil {
			// Wewnątrz notatki wieloliniowej każda niepusta linia jest tekstem; komentarze
			// „%%” są pomijane jak wszędzie.
			if strings.EqualFold(s, "end note") {
				p.g.Notes = append(p.g.Notes, p.note)
				p.note = nil
			} else if s != "" && !strings.HasPrefix(s, "%%") {
				p.note.Text = append(p.note.Text, split(s)...)
			}
			continue
		}
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "stateDiagram-v2" && s != "stateDiagram" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"stateDiagram-v2\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if p.acc.Line(s) {
			continue
		}
		if err := p.statement(s, line); err != nil {
			return nil, err
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"stateDiagram-v2\"")
	}
	if p.note != nil {
		return nil, syntax.Errorf(p.noteLine, CodeNote, "note is not closed with \"end note\"")
	}
	if len(p.stack) > 0 {
		return nil, syntax.Errorf(0, CodeComposite, "composite state %q is not closed with \"}\"", p.scope())
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

// endpoint zamienia zapis końca przejścia na identyfikator węzła; „[*]” oznacza stan
// początkowy (jako źródło) albo końcowy (jako cel) bieżącego zakresu.
func (p *parser) endpoint(token string, source bool, line int) (string, error) {
	if token != "[*]" {
		_, err := p.g.EnsureNode(token, p.scope(), graph.Round, line)
		return token, err
	}
	id, shape := "[end]", graph.StateEnd
	if source {
		id, shape = "[start]", graph.StateStart
	}
	if p.scope() != "" {
		id += "@" + p.scope()
	}
	n, err := p.g.EnsureNode(id, p.scope(), shape, line)
	if err != nil {
		return "", err
	}
	n.Label = nil
	return id, nil
}

func (p *parser) open(id string, title []string, line int) error {
	if err := p.g.AddSubgraph(&graph.Subgraph{ID: id, Title: title, Parent: p.scope(), Line: line}); err != nil {
		return err
	}
	p.stack = append(p.stack, id)
	return nil
}

func (p *parser) statement(s string, line int) error {
	word := s
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		word = s[:i]
	}
	switch {
	case s == "}":
		if len(p.stack) == 0 {
			return syntax.Errorf(line, CodeComposite, "\"}\" without an open composite state")
		}
		if p.g.Subgraph(p.scope()).Region {
			p.stack = p.stack[:len(p.stack)-1] // klamra zamyka ostatni region razem ze stanem
		}
		p.stack = p.stack[:len(p.stack)-1]
		return nil
	case s == "--":
		return p.region(line)
	case word == "click":
		return nil // odnośniki nie mają znaczenia na rysunku statycznym
	case word == "classDef" || word == "style":
		m := styleHead.FindStringSubmatch(strings.TrimSpace(s[len(word):]))
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
	case word == "class":
		m := styleHead.FindStringSubmatch(strings.TrimSpace(s[len(word):]))
		if m == nil {
			return syntax.Errorf(line, CodeStatement, "class needs state ids and a class name")
		}
		p.g.AssignClass(m[1], m[2], line)
		return nil
	case word == "direction":
		d, ok := graph.ParseDirection(strings.TrimSpace(s[len(word):]))
		if !ok {
			return syntax.Errorf(line, CodeStatement, "unknown direction in %q", s)
		}
		if p.scope() == "" {
			p.g.Direction = d
		} else {
			p.g.Subgraph(p.scope()).Direction = d
		}
		return nil
	}
	for _, u := range unsupported {
		if word == u {
			return syntax.Errorf(line, CodeUnsupported, "%q is outside the supported profile", word)
		}
	}

	// Skrót „Id:::klasa” może stać przy każdym identyfikatorze stanu; po zapisaniu klasy znika.
	if !strings.EqualFold(word, "note") && strings.Contains(s, ":::") {
		for _, m := range classUse.FindAllStringSubmatch(s, -1) {
			p.g.AssignClass(m[1], m[2], line)
		}
		s = classUse.ReplaceAllString(s, "$1")
	}

	if m := noteLine.FindStringSubmatch(s); m != nil {
		n := &graph.Note{Target: m[2], Side: strings.ToLower(m[1]), Line: line}
		if strings.Contains(s, ":") {
			n.Text = split(m[3])
			p.g.Notes = append(p.g.Notes, n)
		} else {
			p.note, p.noteLine = n, line
		}
		return nil
	}
	if strings.EqualFold(word, "note") {
		return syntax.Errorf(line, CodeNote, "note needs \"left of\" or \"right of\" a state, found %q", s)
	}

	if m := transition.FindStringSubmatch(s); m != nil {
		from, err := p.endpoint(m[1], true, line)
		if err != nil {
			return err
		}
		to, err := p.endpoint(m[2], false, line)
		if err != nil {
			return err
		}
		e := &graph.Edge{From: from, To: to, Style: graph.Normal, StartHead: graph.NoHead, EndHead: graph.ArrowHead, Line: line}
		if strings.TrimSpace(m[3]) != "" {
			e.Label = split(m[3])
		}
		return p.g.AddEdge(e)
	}
	if m := described.FindStringSubmatch(s); m != nil {
		if m[3] != "" {
			return p.open(m[2], split(m[1]), line)
		}
		n, err := p.g.EnsureNode(m[2], p.scope(), graph.Round, line)
		if err != nil {
			return err
		}
		n.Label, n.Explicit = split(m[1]), true
		return nil
	}
	if m := stereotype.FindStringSubmatch(s); m != nil {
		n, err := p.g.EnsureNode(m[1], p.scope(), graph.Round, line)
		if err != nil {
			return err
		}
		n.Shape, n.Label = map[string]graph.Shape{"choice": graph.Choice, "fork": graph.Fork, "join": graph.Join}[m[2]], nil
		return nil
	}
	if m := declared.FindStringSubmatch(s); m != nil {
		if m[2] != "" {
			return p.open(m[1], []string{m[1]}, line)
		}
		// „state Id” bez opisu, stereotypu i klamry nie jest opisane w dokumentacji Mermaid.
		return syntax.Errorf(line, CodeUnsupported, "\"state %s\" needs a description, a stereotype or \"{\"; declare a plain state with its id alone", m[1])
	}
	if word == "state" {
		return syntax.Errorf(line, CodeStatement, "unrecognised state declaration %q", s)
	}
	if m := description.FindStringSubmatch(s); m != nil {
		n, err := p.g.EnsureNode(m[1], p.scope(), graph.Round, line)
		if err != nil {
			return err
		}
		if n.Explicit {
			n.Label = append(n.Label, split(m[2])...)
		} else {
			n.Label, n.Explicit = split(m[2]), true
		}
		return nil
	}
	if bareID.MatchString(s) {
		_, err := p.g.EnsureNode(s, p.scope(), graph.Round, line)
		return err
	}
	return syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
}
