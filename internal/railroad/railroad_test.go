package railroad

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// show zapisuje wyrażenie w jednej linii.
func show(n *Node) string {
	var kids []string
	for _, c := range n.Children {
		kids = append(kids, show(c))
	}
	switch n.Kind {
	case Terminal:
		return `"` + n.Text + `"`
	case NonTerminal:
		return n.Text
	case Special:
		return "?" + n.Text + "?"
	case Sequence:
		return "seq(" + strings.Join(kids, " ") + ")"
	case Choice:
		return "alt(" + strings.Join(kids, " | ") + ")"
	case Optional:
		return "opt(" + kids[0] + ")"
	case ZeroOrMore:
		return "star(" + kids[0] + ")"
	case OneOrMore:
		if n.Label != "" {
			return "rep[" + n.Label + "](" + kids[0] + ")"
		}
		return "plus(" + kids[0] + ")"
	}
	return "?"
}

func rules(t *testing.T, src string) string {
	t.Helper()
	d, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range d.Rules {
		out = append(out, r.Name+"="+show(r.Expr))
	}
	return strings.Join(out, "; ")
}

// REQ: RAIL-001 RAIL-002
func TestEBNF(t *testing.T) {
	got := rules(t, "railroad-ebnf-beta\ntitle \"T\"\n/* c */ a = b c? d* e+ ( \"x\" | 'y' ) ;\n(* iso *) f ::= g , [ h ] , { i } ;\nj = ? any ? - \"q\"\n  | k ;")
	want := `a=seq(b opt(c) star(d) plus(e) alt("x" | "y")); f=seq(g opt(h) star(i)); j=alt(seq(?any? ?except "q"?) | k)`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// REQ: RAIL-003
func TestABNFAndPEG(t *testing.T) {
	got := rules(t, "railroad-abnf-beta\nd = 4DIGIT \"-\" 2*3x [ \"T\" ] *y 1*z / %x30-39 ; komentarz\nr = 0*1w 3*v ;")
	want := `d=alt(seq(rep[4×](DIGIT) "-" rep[2–3×](x) opt("T") star(y) plus(z)) | "%x30-39"); r=seq(opt(w) rep[≥ 3×](v))`
	if got != want {
		t.Errorf("ABNF got\n%s\nwant\n%s", got, want)
	}
	got = rules(t, "railroad-peg-beta\n# c\nE <- T ((\"+\" / \"-\") T)* ;\nN <- !\"0\" [0-9]+ &. ;")
	want = `E=seq(T star(seq(alt("+" | "-") T))); N=seq(seq(?not followed by? "0") plus("[0-9]") seq(?followed by? ?any character?))`
	if got != want {
		t.Errorf("PEG got\n%s\nwant\n%s", got, want)
	}
}

// REQ: RAIL-004
func TestConstructors(t *testing.T) {
	got := rules(t, "railroad-beta\nk = sequence(terminal(\"W\"), choice(terminal(\"A\"), nonterminal(\"s\")), optional(terminal(\"-\")), zeroOrMore(special(\"x\")), oneOrMore(nonterminal(\"d\"), terminal(\",\")));")
	want := `k=seq("W" alt("A" | s) opt("-") star(?x?) plus(seq(d ",")))`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// REQ: RAIL-005
func TestErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                 {CodeHeader, 0},
		"railroad-bnf-beta\na = b ;":       {CodeHeader, 1},
		"railroad-ebnf-beta\n%% nic":       {CodeEmpty, 0},
		"railroad-ebnf-beta\na = b":        {CodeStatement, 2},
		"railroad-ebnf-beta\n= b ;":        {CodeStatement, 2},
		"railroad-ebnf-beta\na b ;":        {CodeStatement, 2},
		"railroad-ebnf-beta\n\na = \"x ;":  {CodeStatement, 3},
		"railroad-ebnf-beta\na = ( b ;":    {CodeStatement, 2},
		"railroad-ebnf-beta\n(* a = b ;":   {CodeStatement, 2},
		"railroad-peg-beta\na = b ;":       {CodeStatement, 2},
		"railroad-beta\na = loop(\"x\") ;": {CodeStatement, 2},
		"railroad-beta\na = sequence() ;":  {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: RAIL-005
func TestLayout(t *testing.T) {
	d, err := Parse("railroad-ebnf-beta\nlista = el { \",\" el } ;\nel = \"a\" | \"b\" | c ;")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	count := map[string]int{}
	for _, it := range sc.Items {
		count[it.Role]++
	}
	if count["railroad-rule"] != 2 || count["railroad-start"] != 2 || count["railroad-end"] != 2 || count["railroad-terminal"] != 3 || count["railroad-nonterminal"] != 3 {
		t.Errorf("items: %v", count)
	}
	b := sc.Bounds()
	if b.Width() <= 0 || b.Height() <= 0 || strings.Contains(string(sc.SVG()), "NaN") {
		t.Errorf("bounds %+v", b)
	}
}
