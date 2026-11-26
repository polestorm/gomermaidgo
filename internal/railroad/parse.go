// Package railroad rozbiera i rysuje diagramy składni (railroad): reguły gramatyki zapisane w EBNF,
// ABNF, PEG albo konstruktorami, każda jako tor, po którym czyta się poprawne napisy.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package railroad

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramu składni.
const (
	CodeHeader    = "railroad-header"
	CodeStatement = "railroad-statement"
	CodeEmpty     = "railroad-empty"
	maxRules      = 100
	maxNodes      = 2000
	maxDepth      = 40
)

// Kind to rodzaj węzła wyrażenia.
type Kind int

const (
	Terminal Kind = iota
	NonTerminal
	Special
	Sequence
	Choice
	Optional
	ZeroOrMore
	OneOrMore
	Skip // pusta droga (gałąź opcji)
)

// Node to węzeł wyrażenia; Label przy powtórzeniu to podpis pętli (np. „2–4×”).
type Node struct {
	Kind     Kind
	Text     string
	Children []*Node
	Label    string
}

// Rule to reguła: nazwa i wyrażenie.
type Rule struct {
	Name string
	Expr *Node
	Line int
}

// Diagram to tytuł i reguły w kolejności źródła.
type Diagram struct {
	Notation string // ebnf, abnf, peg, ir
	Title    string
	Rules    []Rule
	AccTitle string
	AccDescr string
}

var headers = map[string]string{"railroad-ebnf-beta": "ebnf", "railroad-abnf-beta": "abnf", "railroad-peg-beta": "peg", "railroad-beta": "ir",
	"railroad-ebnf": "ebnf", "railroad-abnf": "abnf", "railroad-peg": "peg", "railroad": "ir"}

// Parse rozbiera źródło.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	source = strings.ReplaceAll(source, "\r\n", "\n")
	lines := strings.Split(source, "\n")
	// Nagłówek, title i dostępność; reszta to treść gramatyki z zachowanymi numerami linii.
	var body []string
	start := 0
	for n, raw := range lines {
		s := strings.TrimSpace(raw)
		if d.Notation == "" {
			if s == "" || strings.HasPrefix(s, "%%") {
				continue
			}
			nt, ok := headers[s]
			if !ok {
				return nil, syntax.Errorf(n+1, CodeHeader, "expected \"railroad-beta\", \"railroad-ebnf-beta\", \"railroad-abnf-beta\" or \"railroad-peg-beta\", found %q", s)
			}
			d.Notation, start = nt, n+1
			continue
		}
		switch {
		case strings.HasPrefix(s, "%%"):
			body = append(body, "")
		case acc.Line(s):
			body = append(body, "")
		case strings.HasPrefix(s, "title ") || s == "title":
			d.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(s, "title")), `"`)
			body = append(body, "")
		default:
			body = append(body, raw)
		}
	}
	if d.Notation == "" {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"railroad-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	toks, err := tokenize(strings.Join(body, "\n"), d.Notation, start+1)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks, notation: d.Notation}
	for !p.done() {
		if len(d.Rules) >= maxRules {
			return nil, syntax.Errorf(p.line(), CodeStatement, "too many rules (limit %d)", maxRules)
		}
		r, err := p.rule()
		if err != nil {
			return nil, err
		}
		d.Rules = append(d.Rules, r)
	}
	if len(d.Rules) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the diagram has no rules")
	}
	d.AccTitle, d.AccDescr = acc.Title, acc.Descr
	return d, nil
}

// token to słowo gramatyki: s napis, i nazwa, n liczba albo wartość ABNF (%x41), p znak.
type token struct {
	kind byte
	text string
	line int
}

// tokenize dzieli treść na słowa, pomijając komentarze notacji.
func tokenize(src, notation string, firstLine int) ([]token, error) {
	var out []token
	rs := []rune(src)
	line := firstLine
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '\n':
			line++
			i++
		case unicode.IsSpace(r):
			i++
		case notation == "ebnf" && r == '/' && i+1 < len(rs) && rs[i+1] == '*',
			notation == "ebnf" && r == '(' && i+1 < len(rs) && rs[i+1] == '*':
			end := "*/"
			if r == '(' {
				end = "*)"
			}
			j := i + 2
			for j+1 < len(rs) && string(rs[j:j+2]) != end {
				if rs[j] == '\n' {
					line++
				}
				j++
			}
			if j+1 >= len(rs) {
				return nil, syntax.Errorf(line, CodeStatement, "comment is not closed with %q", end)
			}
			i = j + 2
		case notation == "peg" && r == '#':
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
		case notation == "abnf" && r == ';':
			// Średnik kończy regułę; reszta linii to komentarz.
			out = append(out, token{'p', ";", line})
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
		case r == '"' || r == '\'':
			j := i + 1
			for j < len(rs) && rs[j] != r && rs[j] != '\n' {
				j++
			}
			if j >= len(rs) || rs[j] != r {
				return nil, syntax.Errorf(line, CodeStatement, "string is not closed with %c", r)
			}
			out = append(out, token{'s', string(rs[i+1 : j]), line})
			i = j + 1
		case notation == "ebnf" && r == '?':
			// Sekwencja specjalna ISO: ? opis ?; w zapisie W3C „?” po elemencie to opcja.
			j := i + 1
			for j < len(rs) && rs[j] != '?' && rs[j] != '\n' && rs[j] != ';' {
				j++
			}
			text := strings.TrimSpace(string(rs[i+1 : j]))
			if j < len(rs) && rs[j] == '?' && text != "" && len(out) > 0 && (out[len(out)-1].kind == 'p' && strings.Contains("=,|([{", out[len(out)-1].text)) {
				out = append(out, token{'x', text, line})
				i = j + 1
			} else {
				out = append(out, token{'p', "?", line})
				i++
			}
		case r == '%' && notation == "abnf":
			j := i + 1
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '-' || rs[j] == '.') {
				j++
			}
			out = append(out, token{'n', string(rs[i:j]), line})
			i = j
		case unicode.IsDigit(r):
			j := i
			for j < len(rs) && unicode.IsDigit(rs[j]) {
				j++
			}
			out = append(out, token{'n', string(rs[i:j]), line})
			i = j
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_' || (notation == "abnf" && rs[j] == '-')) {
				j++
			}
			out = append(out, token{'i', string(rs[i:j]), line})
			i = j
		case r == ':' && i+2 < len(rs) && rs[i+1] == ':' && rs[i+2] == '=':
			out = append(out, token{'p', "=", line})
			i += 3
		case r == '<' && i+1 < len(rs) && rs[i+1] == '-':
			out = append(out, token{'p', "<-", line})
			i += 2
		case notation == "peg" && r == '[':
			j := i + 1
			for j < len(rs) && rs[j] != ']' && rs[j] != '\n' {
				j++
			}
			if j >= len(rs) || rs[j] != ']' {
				return nil, syntax.Errorf(line, CodeStatement, "character class is not closed with ]")
			}
			out = append(out, token{'s', string(rs[i : j+1]), line})
			i = j + 1
		default:
			out = append(out, token{'p', string(r), line})
			i++
		}
	}
	return out, nil
}

type parser struct {
	toks     []token
	pos      int
	notation string
	nodes    int
	depth    int
}

func (p *parser) done() bool { return p.pos >= len(p.toks) }

func (p *parser) line() int {
	if p.pos < len(p.toks) {
		return p.toks[p.pos].line
	}
	if len(p.toks) > 0 {
		return p.toks[len(p.toks)-1].line
	}
	return 0
}

func (p *parser) peek(text string) bool {
	return p.pos < len(p.toks) && p.toks[p.pos].kind == 'p' && p.toks[p.pos].text == text
}

func (p *parser) accept(text string) bool {
	if p.peek(text) {
		p.pos++
		return true
	}
	return false
}

func (p *parser) errorf(format string, args ...any) error {
	return syntax.Errorf(p.line(), CodeStatement, format, args...)
}

func (p *parser) node(n *Node) (*Node, error) {
	if p.nodes++; p.nodes > maxNodes {
		return nil, p.errorf("the grammar is too large (limit %d elements)", maxNodes)
	}
	return n, nil
}

// rule czyta „nazwa = wyrażenie ;” (w PEG „Nazwa <- wyrażenie ;”).
func (p *parser) rule() (Rule, error) {
	if p.toks[p.pos].kind != 'i' {
		return Rule{}, p.errorf("a rule starts with its name, found %q", p.toks[p.pos].text)
	}
	r := Rule{Name: p.toks[p.pos].text, Line: p.toks[p.pos].line}
	p.pos++
	assign := "="
	if p.notation == "peg" {
		assign = "<-"
	}
	if !p.accept(assign) {
		return Rule{}, p.errorf("rule %q: expected %q after the name", r.Name, assign)
	}
	var err error
	if p.notation == "ir" {
		r.Expr, err = p.constructor()
	} else {
		r.Expr, err = p.choice()
	}
	if err != nil {
		return Rule{}, err
	}
	if !p.accept(";") {
		if p.done() {
			return Rule{}, syntax.Errorf(p.line(), CodeStatement, "rule %q is not closed with \";\"", r.Name)
		}
		return Rule{}, p.errorf("rule %q: unexpected %q", r.Name, p.toks[p.pos].text)
	}
	return r, nil
}

// choice: sekwencje rozdzielone „|” (EBNF) albo „/” (ABNF, PEG).
func (p *parser) choice() (*Node, error) {
	if p.depth++; p.depth > maxDepth {
		return nil, p.errorf("expression nested too deep (limit %d)", maxDepth)
	}
	defer func() { p.depth-- }()
	sep := "|"
	if p.notation != "ebnf" {
		sep = "/"
	}
	var alts []*Node
	for {
		s, err := p.sequence()
		if err != nil {
			return nil, err
		}
		alts = append(alts, s)
		if !p.accept(sep) {
			break
		}
	}
	if len(alts) == 1 {
		return alts[0], nil
	}
	return p.node(&Node{Kind: Choice, Children: alts})
}

// sequence: elementy jeden za drugim (w EBNF ISO rozdzielone przecinkami), z wyjątkiem „A - B”.
func (p *parser) sequence() (*Node, error) {
	var items []*Node
	for !p.done() && !p.peek(";") && !p.peek(")") && !p.peek("]") && !p.peek("}") && !p.peek("|") && !(p.notation != "ebnf" && p.peek("/")) {
		if len(items) > 0 && p.notation == "ebnf" && p.accept(",") {
			continue
		}
		t, err := p.term()
		if err != nil {
			return nil, err
		}
		if p.notation == "ebnf" && p.accept("-") {
			ex, err := p.term()
			if err != nil {
				return nil, err
			}
			not, err := p.node(&Node{Kind: Special, Text: "except " + describe(ex)})
			if err != nil {
				return nil, err
			}
			t = &Node{Kind: Sequence, Children: []*Node{t, not}}
		}
		items = append(items, t)
	}
	if len(items) == 0 {
		return nil, p.errorf("an expression is expected")
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return p.node(&Node{Kind: Sequence, Children: items})
}

// term: element z przyrostkiem (EBNF W3C, PEG), przedrostkiem powtórzenia (ABNF) albo predykatem (PEG).
func (p *parser) term() (*Node, error) {
	if p.notation == "abnf" {
		min, max, rep := -1, -1, false
		if !p.done() && p.toks[p.pos].kind == 'n' && !strings.HasPrefix(p.toks[p.pos].text, "%") {
			min, _ = strconv.Atoi(p.toks[p.pos].text)
			p.pos++
			rep = true
		}
		if p.accept("*") {
			rep = true
			if min < 0 {
				min = 0
			}
			if !p.done() && p.toks[p.pos].kind == 'n' && !strings.HasPrefix(p.toks[p.pos].text, "%") {
				max, _ = strconv.Atoi(p.toks[p.pos].text)
				p.pos++
			}
		} else if rep {
			max = min
		}
		e, err := p.primary()
		if err != nil || !rep {
			return e, err
		}
		return repeat(e, min, max)
	}
	pred := ""
	if p.notation == "peg" && (p.peek("&") || p.peek("!")) {
		pred = p.toks[p.pos].text
		p.pos++
	}
	e, err := p.primary()
	if err != nil {
		return nil, err
	}
	for p.notation != "abnf" {
		switch {
		case p.accept("?"):
			e = &Node{Kind: Optional, Children: []*Node{e}}
		case p.accept("*"):
			e = &Node{Kind: ZeroOrMore, Children: []*Node{e}}
		case p.accept("+"):
			e = &Node{Kind: OneOrMore, Children: []*Node{e}}
		default:
			if pred != "" {
				label := "followed by"
				if pred == "!" {
					label = "not followed by"
				}
				return &Node{Kind: Sequence, Children: []*Node{{Kind: Special, Text: label}, e}}, nil
			}
			return e, nil
		}
	}
	return e, nil
}

// repeat buduje powtórzenie ABNF: *A, 1*A, n*mA, nA.
func repeat(e *Node, min, max int) (*Node, error) {
	switch {
	case min <= 0 && max < 0:
		return &Node{Kind: ZeroOrMore, Children: []*Node{e}}, nil
	case min == 1 && max < 0:
		return &Node{Kind: OneOrMore, Children: []*Node{e}}, nil
	case min == 0 && max == 1:
		return &Node{Kind: Optional, Children: []*Node{e}}, nil
	case min == 1 && max == 1:
		return e, nil
	}
	label := strconv.Itoa(min) + "–" + strconv.Itoa(max) + "×"
	switch {
	case max < 0:
		label = "≥ " + strconv.Itoa(min) + "×"
	case min == max:
		label = strconv.Itoa(min) + "×"
	}
	n := &Node{Kind: OneOrMore, Children: []*Node{e}, Label: label}
	if min == 0 {
		return &Node{Kind: Optional, Children: []*Node{n}}, nil
	}
	return n, nil
}

// primary: napis, nazwa, wartość ABNF, grupa (), opcja [], powtórzenie {} (EBNF ISO),
// sekwencja specjalna ? … ?, kropka PEG.
func (p *parser) primary() (*Node, error) {
	if p.done() {
		return nil, p.errorf("an expression is expected")
	}
	t := p.toks[p.pos]
	p.pos++
	switch {
	case t.kind == 's' || t.kind == 'n' && strings.HasPrefix(t.text, "%"):
		return p.node(&Node{Kind: Terminal, Text: t.text})
	case t.kind == 'i':
		return p.node(&Node{Kind: NonTerminal, Text: t.text})
	case t.kind == 'x':
		return p.node(&Node{Kind: Special, Text: t.text})
	case t.text == "." && p.notation == "peg":
		return p.node(&Node{Kind: Special, Text: "any character"})
	case t.text == "(" || t.text == "[" || t.text == "{" && p.notation == "ebnf":
		closing := map[string]string{"(": ")", "[": "]", "{": "}"}[t.text]
		e, err := p.choice()
		if err != nil {
			return nil, err
		}
		if !p.accept(closing) {
			return nil, p.errorf("%q is not closed with %q", t.text, closing)
		}
		switch t.text {
		case "[":
			return &Node{Kind: Optional, Children: []*Node{e}}, nil
		case "{":
			return &Node{Kind: ZeroOrMore, Children: []*Node{e}}, nil
		}
		return e, nil
	}
	return nil, syntax.Errorf(t.line, CodeStatement, "unexpected %q", t.text)
}

// constructor czyta zapis railroad-beta: terminal("x"), nonterminal("x"), special("x"),
// sequence(…), choice(…), optional(a), zeroOrMore(a), oneOrMore(a).
func (p *parser) constructor() (*Node, error) {
	if p.depth++; p.depth > maxDepth {
		return nil, p.errorf("expression nested too deep (limit %d)", maxDepth)
	}
	defer func() { p.depth-- }()
	if p.done() || p.toks[p.pos].kind != 'i' {
		return nil, p.errorf("a constructor such as terminal(\"x\") is expected")
	}
	name := p.toks[p.pos].text
	p.pos++
	if !p.accept("(") {
		return nil, p.errorf("%s: \"(\" expected", name)
	}
	switch name {
	case "terminal", "nonterminal", "special":
		if p.done() || p.toks[p.pos].kind != 's' {
			return nil, p.errorf("%s takes a quoted text", name)
		}
		text := p.toks[p.pos].text
		p.pos++
		if !p.accept(")") {
			return nil, p.errorf("%s: \")\" expected", name)
		}
		kind := map[string]Kind{"terminal": Terminal, "nonterminal": NonTerminal, "special": Special}[name]
		return p.node(&Node{Kind: kind, Text: text})
	case "sequence", "choice", "optional", "zeroOrMore", "oneOrMore":
		var args []*Node
		for !p.accept(")") {
			if len(args) > 0 && !p.accept(",") {
				return nil, p.errorf("%s: \",\" or \")\" expected", name)
			}
			a, err := p.constructor()
			if err != nil {
				return nil, err
			}
			args = append(args, a)
		}
		if len(args) == 0 {
			return nil, p.errorf("%s needs at least one argument", name)
		}
		kind := map[string]Kind{"sequence": Sequence, "choice": Choice, "optional": Optional, "zeroOrMore": ZeroOrMore, "oneOrMore": OneOrMore}[name]
		if (kind == Optional || kind == ZeroOrMore || kind == OneOrMore) && len(args) > 1 {
			args = []*Node{{Kind: Sequence, Children: args}}
		}
		return p.node(&Node{Kind: kind, Children: args})
	}
	return nil, p.errorf("unknown constructor %q", name)
}

// describe zwraca krótki opis wyrażenia do podpisu wyjątku.
func describe(n *Node) string {
	switch n.Kind {
	case Terminal:
		if strings.Contains(n.Text, `"`) {
			return "'" + n.Text + "'"
		}
		return `"` + n.Text + `"`
	case NonTerminal, Special:
		return n.Text
	}
	return "…"
}
