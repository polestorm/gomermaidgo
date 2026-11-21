package sequence

import (
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramu sekwencji.
const (
	CodeHeader        = "sequence-header"
	CodeUnsupported   = "sequence-unsupported"
	CodeStatement     = "sequence-statement"
	CodeBlock         = "sequence-block"
	CodeActivation    = "sequence-activation"
	CodeParticipant   = "sequence-participant"
	CodeEmpty         = "sequence-empty"
	maxParticipants   = 50
	maxEvents         = 2000
	maxBlockNestDepth = 20
)

var (
	lineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)

	// Strzałki w kolejności od najdłuższej, żeby „-->>” nie zostało odczytane jako „-->”.
	arrows = []struct {
		token string
		line  LineStyle
		head  HeadStyle
		both  bool
	}{
		{"<<-->>", Dotted, HeadArrow, true},
		{"<<->>", Solid, HeadArrow, true},
		{"-->>", Dotted, HeadArrow, false},
		{"->>", Solid, HeadArrow, false},
		{"--x", Dotted, HeadCross, false},
		{"--)", Dotted, HeadOpen, false},
		{"-->", Dotted, HeadNone, false},
		{"-x", Solid, HeadCross, false},
		{"-)", Solid, HeadOpen, false},
		{"->", Solid, HeadNone, false},
	}

	// Instrukcje Mermaid poza profilem v1: znane, ale świadomie nieobsługiwane.
	unsupported = []string{"box", "create", "destroy", "link", "links", "properties", "details", "accTitle", "accDescr"}

	blockStarts   = map[string]bool{"loop": true, "alt": true, "opt": true, "par": true, "critical": true, "break": true, "rect": true}
	blockSections = map[string]string{"else": "alt", "and": "par", "option": "critical"}
)

func splitLines(text string) []string {
	parts := lineBreak.Split(text, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// keyword oddziela pierwsze słowo instrukcji od reszty.
func keyword(s string) (word, rest string) {
	i := strings.IndexAny(s, " \t")
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i+1:])
}

type parser struct {
	d      *Diagram
	index  map[string]int
	blocks []string // stos otwartych bloków
}

func (p *parser) participant(id string, line int) (int, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0, syntax.Errorf(line, CodeParticipant, "missing participant name")
	}
	if i, ok := p.index[id]; ok {
		return i, nil
	}
	if len(p.d.Participants) >= maxParticipants {
		return 0, syntax.Errorf(line, CodeParticipant, "too many participants (limit %d)", maxParticipants)
	}
	p.index[id] = len(p.d.Participants)
	p.d.Participants = append(p.d.Participants, &Participant{ID: id, Label: splitLines(id), Line: line})
	return p.index[id], nil
}

func (p *parser) add(e Event) error {
	if len(p.d.Events) >= maxEvents {
		return syntax.Errorf(e.Line, CodeStatement, "too many statements (limit %d)", maxEvents)
	}
	p.d.Events = append(p.d.Events, e)
	return nil
}

// Parse rozbiera źródło diagramu sekwencji. Pierwszą linią treści musi być „sequenceDiagram”.
func Parse(source string) (*Diagram, error) {
	p := &parser{d: &Diagram{}, index: map[string]int{}}
	seenHeader := false
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "sequenceDiagram" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"sequenceDiagram\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if err := p.statement(s, line); err != nil {
			return nil, err
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"sequenceDiagram\"")
	}
	if len(p.blocks) > 0 {
		return nil, syntax.Errorf(0, CodeBlock, "block %q is not closed with \"end\"", p.blocks[len(p.blocks)-1])
	}
	if len(p.d.Participants) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "diagram has no participants")
	}
	if err := checkActivations(p.d); err != nil {
		return nil, err
	}
	return p.d, nil
}

// checkActivations odrzuca dezaktywację uczestnika, który nie jest aktywny, żeby układ
// nie musiał zgadywać, gdzie zaczyna się pasek aktywacji.
func checkActivations(d *Diagram) error {
	active := make([]int, len(d.Participants))
	off := func(i, line int) error {
		if active[i] == 0 {
			return syntax.Errorf(line, CodeActivation, "participant %q is deactivated but not active", d.Participants[i].ID)
		}
		active[i]--
		return nil
	}
	for _, e := range d.Events {
		switch e.Kind {
		case EventActivate:
			active[e.Target]++
		case EventDeactivate:
			if err := off(e.Target, e.Line); err != nil {
				return err
			}
		case EventMessage:
			if e.ActivateTo {
				active[e.To]++
			}
			if e.DeactivateFrom {
				if err := off(e.From, e.Line); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *parser) statement(s string, line int) error {
	word, rest := keyword(s)
	lower := strings.ToLower(word)

	switch {
	case word == "participant" || word == "actor":
		id, label := rest, rest
		if i := strings.Index(rest, " as "); i >= 0 {
			id, label = rest[:i], strings.TrimSpace(rest[i+4:])
		}
		id = strings.TrimSpace(id)
		if _, exists := p.index[id]; exists {
			return syntax.Errorf(line, CodeParticipant, "participant %q is already declared", id)
		}
		i, err := p.participant(id, line)
		if err != nil {
			return err
		}
		p.d.Participants[i].Actor = word == "actor"
		p.d.Participants[i].Label = splitLines(label)
		return nil

	case word == "autonumber":
		if rest != "" {
			return syntax.Errorf(line, CodeUnsupported, "autonumber with start or step is outside the supported profile")
		}
		p.d.Autonumber = true
		return nil

	case word == "title" || word == "title:":
		p.d.Title = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
		return nil

	case word == "activate" || word == "deactivate":
		i, ok := p.index[rest]
		if !ok {
			return syntax.Errorf(line, CodeActivation, "%s refers to unknown participant %q", word, rest)
		}
		kind := EventActivate
		if word == "deactivate" {
			kind = EventDeactivate
		}
		return p.add(Event{Kind: kind, Line: line, Target: i})

	case lower == "note":
		return p.note(rest, line)

	case blockStarts[word]:
		if len(p.blocks) >= maxBlockNestDepth {
			return syntax.Errorf(line, CodeBlock, "blocks nested too deeply (limit %d)", maxBlockNestDepth)
		}
		p.blocks = append(p.blocks, word)
		e := Event{Kind: EventBlockStart, Line: line, Block: word}
		if word == "rect" {
			if !validColor(rest) {
				return syntax.Errorf(line, CodeBlock, "rect needs a colour such as rgb(200, 220, 255), found %q", rest)
			}
			e.Color = normalizeColor(rest)
		} else if rest != "" {
			e.Text = splitLines(rest)
		}
		return p.add(e)

	case blockSections[word] != "":
		want := blockSections[word]
		if len(p.blocks) == 0 || p.blocks[len(p.blocks)-1] != want {
			return syntax.Errorf(line, CodeBlock, "%q is only allowed directly inside %q", word, want)
		}
		e := Event{Kind: EventBlockSection, Line: line, Block: want}
		if rest != "" {
			e.Text = splitLines(rest)
		}
		return p.add(e)

	case s == "end":
		if len(p.blocks) == 0 {
			return syntax.Errorf(line, CodeBlock, "\"end\" without an open block")
		}
		kind := p.blocks[len(p.blocks)-1]
		p.blocks = p.blocks[:len(p.blocks)-1]
		return p.add(Event{Kind: EventBlockEnd, Line: line, Block: kind})
	}

	for _, u := range unsupported {
		if word == u {
			return syntax.Errorf(line, CodeUnsupported, "%q is outside the supported profile", word)
		}
	}
	return p.message(s, line)
}

func (p *parser) note(rest string, line int) error {
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return syntax.Errorf(line, CodeStatement, "note needs \": text\"")
	}
	head, text := strings.TrimSpace(rest[:colon]), strings.TrimSpace(rest[colon+1:])
	e := Event{Kind: EventNote, Line: line, Text: splitLines(text)}
	var names string
	lowerHead := strings.ToLower(head)
	switch {
	case strings.HasPrefix(lowerHead, "right of "):
		e.Placement, names = NoteRightOf, head[len("right of "):]
	case strings.HasPrefix(lowerHead, "left of "):
		e.Placement, names = NoteLeftOf, head[len("left of "):]
	case strings.HasPrefix(lowerHead, "over "):
		e.Placement, names = NoteOver, head[len("over "):]
	default:
		return syntax.Errorf(line, CodeStatement, "note needs \"right of\", \"left of\" or \"over\", found %q", head)
	}
	ids := strings.Split(names, ",")
	if len(ids) > 2 || (len(ids) == 2 && e.Placement != NoteOver) {
		return syntax.Errorf(line, CodeStatement, "note refers to too many participants")
	}
	for k, id := range ids {
		i, err := p.participant(id, line)
		if err != nil {
			return err
		}
		e.Over[k] = i
	}
	if len(ids) == 1 {
		e.Over[1] = e.Over[0]
	}
	if e.Over[0] > e.Over[1] {
		e.Over[0], e.Over[1] = e.Over[1], e.Over[0]
	}
	return p.add(e)
}

func (p *parser) message(s string, line int) error {
	// Szukamy strzałki najbliżej początku linii; przy tym samym początku wygrywa dłuższa.
	pos, which := -1, -1
	for k, a := range arrows {
		i := strings.Index(s, a.token)
		if i < 0 {
			continue
		}
		if pos < 0 || i < pos {
			pos, which = i, k
		}
	}
	if which < 0 {
		return syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
	}
	a := arrows[which]
	from := strings.TrimSpace(s[:pos])
	rest := s[pos+len(a.token):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return syntax.Errorf(line, CodeStatement, "message needs \": text\" after the receiver")
	}
	to, text := strings.TrimSpace(rest[:colon]), strings.TrimSpace(rest[colon+1:])
	e := Event{Kind: EventMessage, Line: line, LineStyle: a.line, Head: a.head, Bidirectional: a.both}
	switch {
	case strings.HasPrefix(to, "+"):
		e.ActivateTo, to = true, strings.TrimSpace(to[1:])
	case strings.HasPrefix(to, "-"):
		e.DeactivateFrom, to = true, strings.TrimSpace(to[1:])
	}
	var err error
	if e.From, err = p.participant(from, line); err != nil {
		return err
	}
	if e.To, err = p.participant(to, line); err != nil {
		return err
	}
	if text != "" {
		e.Text = splitLines(text)
	}
	return p.add(e)
}

var colorPattern = regexp.MustCompile(`^(?:#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6}|rgba?\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*(?:,\s*(?:0|1|0?\.\d+)\s*)?\))$`)

func validColor(s string) bool { return colorPattern.MatchString(strings.TrimSpace(s)) }

// normalizeColor usuwa odstępy, żeby ten sam kolor zawsze był zapisany tak samo.
func normalizeColor(s string) string { return strings.Join(strings.Fields(s), "") }
