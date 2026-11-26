// Package zenuml rozbiera diagram sekwencji zapisany składnią ZenUML (zenuml w Mermaid) do modelu
// diagramu sekwencji, który rysuje pakiet sequence. Składnia według publicznej dokumentacji
// Mermaid, w zakresie profilu z docs/profile.md.
package zenuml

import (
	"strings"

	"github.com/polestorm/gomermaidgo/internal/sequence"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni ZenUML.
const (
	CodeHeader    = "zenuml-header"
	CodeStatement = "zenuml-statement"
	CodeBlock     = "zenuml-block"
	maxEvents     = 1000
	maxDepth      = 30
)

var (
	name        = `[\p{L}_][\p{L}\p{N}_]*`
	participant = syntax.Lazy(`^(?:@(\w+)\s+)?(` + name + `)(?:\s+as\s+(.+))?$`)
	async       = syntax.Lazy(`^(?:(` + name + `)\s*->\s*)?(` + name + `)\s*:\s*(.*)$`)
	// wywołanie: [Typ ][zmienna = ][Od->]Do.metoda(argumenty) albo [zmienna = ]new Do(argumenty)
	call    = syntax.Lazy(`^(?:(?:` + name + `\s+)?(` + name + `)\s*=\s*)?(?:(` + name + `)\s*->\s*)?(` + name + `)\.(` + name + `)\s*(\(.*\))?$`)
	create  = syntax.Lazy(`^(?:(?:` + name + `\s+)?(` + name + `)\s*=\s*)?new\s+(` + name + `)\s*(\(.*\))?$`)
	control = syntax.Lazy(`^(while|for|forEach|foreach|loop|if|opt|par|try)\s*(\((.*)\))?$`)
)

// stmt to instrukcja z ewentualnym blokiem w klamrach i dalszymi gałęziami (else, catch, finally).
type stmt struct {
	text     string
	line     int
	body     []*stmt
	hasBlock bool
	branches []*stmt // else if, else, catch, finally
}

// parser przechodzi listę instrukcji i buduje zdarzenia diagramu sekwencji.
type parser struct {
	d       *sequence.Diagram
	index   map[string]int
	comment []string
	reply   bool // @return albo @reply przed komunikatem asynchronicznym
	starter int  // indeks uczestnika rozpoczynającego (-1: jeszcze go nie ma)
}

// Parse rozbiera źródło do modelu diagramu sekwencji.
func Parse(source string) (*sequence.Diagram, error) {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	var lines []srcLine
	seenHeader := false
	var acc syntax.Accessibility
	title := ""
	for n, raw := range strings.Split(source, "\n") {
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "zenuml" {
				return nil, syntax.Errorf(n+1, CodeHeader, "expected \"zenuml\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		if strings.HasPrefix(s, "title ") && title == "" && len(lines) == 0 {
			title = strings.TrimSpace(s[6:])
			continue
		}
		lines = append(lines, srcLine{s, n + 1})
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"zenuml\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	pos := 0
	body, err := block(lines, &pos, 0)
	if err != nil {
		return nil, err
	}
	if pos < len(lines) {
		return nil, syntax.Errorf(lines[pos].line, CodeBlock, "\"}\" without an opening \"{\"")
	}
	p := &parser{d: &sequence.Diagram{Title: title, AccTitle: acc.Title, AccDescr: acc.Descr}, index: map[string]int{}, starter: -1}
	if err := p.run(body, -1); err != nil {
		return nil, err
	}
	if len(p.d.Participants) == 0 {
		return nil, syntax.Errorf(0, CodeStatement, "the diagram has no participants")
	}
	return p.d, nil
}

type srcLine struct {
	text string
	line int
}

// block czyta instrukcje do zamykającej klamry (albo do końca źródła na najwyższym poziomie).
// Linia kończąca się „{” otwiera blok; „} else {”, „} catch {” i „} finally {” zamykają gałąź
// i otwierają następną.
func block(lines []srcLine, pos *int, depth int) ([]*stmt, error) {
	if depth > maxDepth {
		return nil, syntax.Errorf(lines[*pos-1].line, CodeBlock, "blocks nested too deep (limit %d)", maxDepth)
	}
	var out []*stmt
	for *pos < len(lines) {
		l := lines[*pos]
		s := l.text
		if strings.HasPrefix(s, "}") {
			if depth == 0 {
				return out, nil
			}
			return out, nil // zamknięcie oddaje wywołującemu, który rozpozna dalszą gałąź
		}
		*pos++
		st := &stmt{text: s, line: l.line}
		if strings.HasSuffix(s, "{") {
			st.text, st.hasBlock = strings.TrimSpace(strings.TrimSuffix(s, "{")), true
			body, err := closeBlock(lines, pos, depth, l.line)
			if err != nil {
				return nil, err
			}
			st.body = body
			// Dalsze gałęzie: „} else if(…) {”, „} else {”, „} catch {”, „} finally {”.
			for *pos < len(lines) {
				t := strings.TrimSpace(strings.TrimPrefix(lines[*pos].text, "}"))
				if !strings.HasPrefix(lines[*pos].text, "}") || t == "" || !strings.HasSuffix(t, "{") {
					break
				}
				br := &stmt{text: strings.TrimSpace(strings.TrimSuffix(t, "{")), line: lines[*pos].line, hasBlock: true}
				*pos++
				if br.body, err = closeBlock(lines, pos, depth, br.line); err != nil {
					return nil, err
				}
				st.branches = append(st.branches, br)
			}
			// Sama „}” zamyka ostatnią gałąź.
			if *pos < len(lines) && lines[*pos].text == "}" {
				*pos++
			} else {
				return nil, syntax.Errorf(l.line, CodeBlock, "block opened here is not closed with \"}\"")
			}
		}
		out = append(out, st)
	}
	return out, nil
}

// closeBlock czyta treść bloku do linii zaczynającej się „}”, nie zjadając jej.
func closeBlock(lines []srcLine, pos *int, depth, open int) ([]*stmt, error) {
	body, err := block(lines, pos, depth+1)
	if err != nil {
		return nil, err
	}
	if *pos >= len(lines) {
		return nil, syntax.Errorf(open, CodeBlock, "block opened here is not closed with \"}\"")
	}
	return body, nil
}

// get zwraca indeks uczestnika, dodając go przy pierwszym użyciu.
func (p *parser) get(id string, line int) int {
	if i, ok := p.index[id]; ok {
		return i
	}
	p.d.Participants = append(p.d.Participants, &sequence.Participant{ID: id, Label: []string{id}, Line: line})
	p.index[id] = len(p.d.Participants) - 1
	return p.index[id]
}

// caller zwraca uczestnika, od którego wychodzi wywołanie bez jawnego nadawcy: wywoływanego
// w bieżącym bloku albo, na najwyższym poziomie, postać rozpoczynającą bez nazwy.
func (p *parser) caller(current int, line int) int {
	if current >= 0 {
		return current
	}
	if p.starter < 0 {
		p.d.Participants = append([]*sequence.Participant{{ID: "", Label: []string{""}, Actor: true, Line: line}}, p.d.Participants...)
		for k := range p.index {
			p.index[k]++
		}
		for i := range p.d.Events {
			e := &p.d.Events[i]
			e.From, e.To, e.Target = e.From+1, e.To+1, e.Target+1
			e.Over[0], e.Over[1] = e.Over[0]+1, e.Over[1]+1
			if e.Created > 0 {
				e.Created++
			}
		}
		p.starter = 0
	}
	return p.starter
}

func (p *parser) add(e sequence.Event) error {
	if len(p.d.Events) >= maxEvents {
		return syntax.Errorf(e.Line, CodeStatement, "too many messages (limit %d)", maxEvents)
	}
	p.d.Events = append(p.d.Events, e)
	return nil
}

// note wypisuje oczekujący komentarz „//” jako notatkę nad komunikatem.
func (p *parser) note(from, to, line int) error {
	if p.comment == nil {
		return nil
	}
	text := p.comment
	p.comment = nil
	return p.add(sequence.Event{Kind: sequence.EventNote, Line: line, Placement: sequence.NoteOver, Over: [2]int{from, to}, Text: text})
}

// run przetwarza instrukcje bloku, w którym wywołującym jest current (-1: najwyższy poziom).
// Zwraca błąd albo nil; odpowiedź „return” zapisuje w p przez wynik returned.
func (p *parser) run(body []*stmt, current int) error {
	_, err := p.runBlock(body, current, -1)
	return err
}

// runBlock zwraca, czy w bloku padło „return” (odpowiedź do back).
func (p *parser) runBlock(body []*stmt, current, back int) (bool, error) {
	returned := false
	for _, st := range body {
		s := st.text
		switch {
		case strings.HasPrefix(s, "//"):
			p.comment = append(p.comment, strings.TrimSpace(s[2:]))
			continue
		case s == "@return" || s == "@reply":
			p.reply = true
			continue
		case strings.HasPrefix(s, "return"):
			if current < 0 || back < 0 {
				return false, syntax.Errorf(st.line, CodeStatement, "return outside a synchronous call")
			}
			value := strings.TrimSpace(strings.TrimPrefix(s, "return"))
			if err := p.add(sequence.Event{Kind: sequence.EventMessage, Line: st.line, From: current, To: back, LineStyle: sequence.Dotted, Head: sequence.HeadArrow, Text: []string{value}}); err != nil {
				return false, err
			}
			returned = true
			continue
		}
		if m := control.FindStringSubmatch(s); m != nil && st.hasBlock {
			if err := p.control(st, m[1], m[3], current, back); err != nil {
				return false, err
			}
			continue
		}
		if m := create.FindStringSubmatch(s); m != nil {
			from := p.caller(current, st.line)
			if _, ok := p.index[m[2]]; ok {
				return false, syntax.Errorf(st.line, CodeStatement, "%q already exists and cannot be created again", m[2])
			}
			to := p.get(m[2], st.line)
			p.d.Participants[to].Created = true
			if err := p.note(from, to, st.line); err != nil {
				return false, err
			}
			label := "«create»"
			if args := strings.TrimSpace(m[3]); args != "" && args != "()" {
				label += " " + args
			}
			if err := p.add(sequence.Event{Kind: sequence.EventMessage, Line: st.line, From: from, To: to, Head: sequence.HeadArrow, Created: to + 1, Text: []string{label}}); err != nil {
				return false, err
			}
			if err := p.callBody(st, from, to, m[1], true); err != nil {
				return false, err
			}
			continue
		}
		if m := call.FindStringSubmatch(s); m != nil {
			from := current
			if m[2] != "" {
				from = p.get(m[2], st.line)
			} else {
				from = p.caller(current, st.line)
			}
			to := p.get(m[3], st.line)
			if err := p.note(from, to, st.line); err != nil {
				return false, err
			}
			if err := p.add(sequence.Event{Kind: sequence.EventMessage, Line: st.line, From: from, To: to, Head: sequence.HeadArrow, ActivateTo: true, Text: []string{m[4] + m[5]}}); err != nil {
				return false, err
			}
			if err := p.callBody(st, from, to, m[1], false); err != nil {
				return false, err
			}
			continue
		}
		if st.hasBlock {
			return false, syntax.Errorf(st.line, CodeStatement, "unknown block %q", s)
		}
		if m := async.FindStringSubmatch(s); m != nil {
			// Nadawca najpierw: dodanie postaci rozpoczynającej przesuwa indeksy uczestników.
			var from int
			if m[1] != "" {
				from = p.get(m[1], st.line)
			} else {
				from = p.caller(current, st.line)
			}
			to := p.get(m[2], st.line)
			if err := p.note(from, to, st.line); err != nil {
				return false, err
			}
			e := sequence.Event{Kind: sequence.EventMessage, Line: st.line, From: from, To: to, Head: sequence.HeadOpen, Text: []string{strings.TrimSpace(m[3])}}
			if p.reply {
				e.LineStyle, e.Head, p.reply = sequence.Dotted, sequence.HeadArrow, false
			}
			if err := p.add(e); err != nil {
				return false, err
			}
			continue
		}
		if m := participant.FindStringSubmatch(s); m != nil {
			if _, ok := p.index[m[2]]; ok {
				return false, syntax.Errorf(st.line, CodeStatement, "participant %q is declared after its first use", m[2])
			}
			i := p.get(m[2], st.line)
			pt := p.d.Participants[i]
			if m[3] != "" {
				pt.Label = []string{strings.TrimSpace(m[3])}
			}
			switch {
			case m[1] == "Actor":
				pt.Actor = true
			case m[1] != "":
				pt.Label = append([]string{"«" + m[1] + "»"}, pt.Label...)
			}
			p.comment = nil // komentarz przy uczestniku nie jest rysowany
			continue
		}
		return false, syntax.Errorf(st.line, CodeStatement, "unknown statement %q", s)
	}
	return returned, nil
}

// callBody przetwarza blok wywołania (wywołującym staje się to) i kończy je: odpowiedzią
// z „return” albo odpowiedzią do zmiennej z przypisania, które zamykają aktywację wywołanego,
// a bez odpowiedzi samym końcem aktywacji. Utworzenie obiektu nie ma aktywacji.
func (p *parser) callBody(st *stmt, from, to int, variable string, created bool) error {
	if len(st.branches) > 0 {
		return syntax.Errorf(st.branches[0].line, CodeBlock, "%q cannot follow a call", st.branches[0].text)
	}
	returned := false
	if st.hasBlock {
		var err error
		if returned, err = p.runBlock(st.body, to, from); err != nil {
			return err
		}
	}
	if !returned && variable != "" {
		if err := p.add(sequence.Event{Kind: sequence.EventMessage, Line: st.line, From: to, To: from, LineStyle: sequence.Dotted, Head: sequence.HeadArrow, Text: []string{variable}}); err != nil {
			return err
		}
		returned = true
	}
	if created {
		return nil
	}
	if returned {
		p.d.Events[len(p.d.Events)-1].DeactivateFrom = true
		return nil
	}
	return p.add(sequence.Event{Kind: sequence.EventDeactivate, Line: st.line, Target: to})
}

// control rysuje blok sterujący: pętlę, alternatywę z gałęziami, opcję, równoległość albo
// try/catch/finally.
func (p *parser) control(st *stmt, kind, cond string, current, back int) error {
	block, text := kind, strings.TrimSpace(cond)
	switch kind {
	case "while", "for", "forEach", "foreach", "loop":
		block = "loop"
		if text == "" {
			text = kind
		} else if kind != "loop" && kind != "while" {
			text = kind + " " + text
		}
	case "if":
		block = "alt"
	}
	p.comment = nil
	if err := p.add(sequence.Event{Kind: sequence.EventBlockStart, Line: st.line, Block: block, Text: []string{text}}); err != nil {
		return err
	}
	if _, err := p.runBlock(st.body, current, back); err != nil {
		return err
	}
	for _, br := range st.branches {
		label := br.text
		switch {
		case kind == "if" && strings.HasPrefix(label, "else if"):
			label = strings.TrimSpace(strings.TrimPrefix(label, "else if"))
			label = strings.TrimSuffix(strings.TrimPrefix(label, "("), ")")
		case kind == "if" && label == "else":
		case kind == "try" && (label == "catch" || strings.HasPrefix(label, "catch") || label == "finally"):
			label = strings.TrimSpace(label)
		default:
			return syntax.Errorf(br.line, CodeBlock, "%q cannot follow %s", br.text, kind)
		}
		if err := p.add(sequence.Event{Kind: sequence.EventBlockSection, Line: br.line, Block: block, Text: []string{label}}); err != nil {
			return err
		}
		if _, err := p.runBlock(br.body, current, back); err != nil {
			return err
		}
	}
	return p.add(sequence.Event{Kind: sequence.EventBlockEnd, Line: st.line})
}
