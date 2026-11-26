package zenuml

import (
	"fmt"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/sequence"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// events zapisuje zdarzenia w skrócie: komunikat „od>do:tekst”, odpowiedź „od-->do:tekst”,
// bloki i notatki nazwą, deaktywacja „-X”.
func events(d *sequence.Diagram) string {
	name := func(i int) string {
		if d.Participants[i].ID == "" {
			return "*"
		}
		return d.Participants[i].ID
	}
	var out []string
	for _, e := range d.Events {
		switch e.Kind {
		case sequence.EventMessage:
			op := ">"
			if e.LineStyle == sequence.Dotted {
				op = "-->"
			} else if e.Head == sequence.HeadOpen {
				op = "->"
			}
			s := name(e.From) + op + name(e.To) + ":" + strings.Join(e.Text, " ")
			if e.ActivateTo {
				s += "+"
			}
			if e.DeactivateFrom {
				s += "-"
			}
			out = append(out, s)
		case sequence.EventNote:
			out = append(out, "note:"+strings.Join(e.Text, " "))
		case sequence.EventBlockStart:
			out = append(out, e.Block+"["+strings.Join(e.Text, " ")+"]")
		case sequence.EventBlockSection:
			out = append(out, "|"+strings.Join(e.Text, " "))
		case sequence.EventBlockEnd:
			out = append(out, "end")
		case sequence.EventDeactivate:
			out = append(out, "-"+name(e.Target))
		}
	}
	return strings.Join(out, " ")
}

// REQ: ZEN-001 ZEN-002 ZEN-003
func TestParseParticipantsAndAsync(t *testing.T) {
	d, err := Parse("%% c\nzenuml\n  title Demo\n  accTitle: Opis\n  @Actor Alice\n  @Database Db\n  B as Bob\n  // pierwszy\n  Alice->B: Hi\n  @return\n  B->Alice: Hello\n  Db: zapis")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Demo" || d.AccTitle != "Opis" {
		t.Errorf("title %q, accTitle %q", d.Title, d.AccTitle)
	}
	var ps []string
	for _, p := range d.Participants {
		ps = append(ps, fmt.Sprintf("%s=%s/%v%s", p.ID, strings.Join(p.Label, "|"), p.Actor, p.Kind))
	}
	if got, want := strings.Join(ps, " "), "=/true Alice=Alice/true Db=Db/falsedatabase B=Bob/false"; got != want {
		t.Errorf("participants %q, want %q", got, want)
	}
	if got, want := events(d), "note:pierwszy Alice->B:Hi B-->Alice:Hello *->Db:zapis"; got != want {
		t.Errorf("events %q, want %q", got, want)
	}
}

// REQ: ZEN-004
func TestParseSyncReturnsAndCreate(t *testing.T) {
	d, err := Parse("zenuml\nA.m() {\n  B.n(x)\n  r = C.get() {\n    return v\n  }\n  T t = C.put()\n  o = new D(1)\n}\nKlient->A.k()")
	if err != nil {
		t.Fatal(err)
	}
	want := "*>A:m()+ A>B:n(x)+ -B A>C:get()+ C-->A:v- A>C:put()+ C-->A:t- A>D:«create» (1) D-->A:o -A Klient>A:k()+ -A"
	if got := events(d); got != want {
		t.Errorf("events\n%q\nwant\n%q", got, want)
	}
	if !d.Participants[d.Events[7].To].Created || d.Events[7].Created == 0 {
		t.Error("new D does not create a participant")
	}
}

// REQ: ZEN-005
func TestParseControl(t *testing.T) {
	d, err := Parse("zenuml\nwhile(ok) {\n  A->B: x\n}\nforEach(item) {\n  A->B: y\n}\nif(a) {\n  A->B: 1\n} else if(b) {\n  A->B: 2\n} else {\n  A->B: 3\n}\nopt {\n  A->B: o\n}\npar {\n  A->B: p\n}\ntry {\n  A->B: t\n} catch {\n  A->B: c\n} finally {\n  A->B: f\n}")
	if err != nil {
		t.Fatal(err)
	}
	want := "loop[ok] A->B:x end loop[forEach item] A->B:y end alt[a] A->B:1 |b A->B:2 |else A->B:3 end opt[] A->B:o end par[] A->B:p end try[] A->B:t |catch A->B:c |finally A->B:f end"
	if got := events(d); got != want {
		t.Errorf("events\n%q\nwant\n%q", got, want)
	}
}

// REQ: ZEN-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                         {CodeHeader, 0},
		"sequenceDiagram":          {CodeHeader, 1},
		"zenuml\n%% nic":           {CodeStatement, 0},
		"zenuml\nA->B: x\n}":       {CodeBlock, 3},
		"zenuml\nA.m() {\n  B.n()": {CodeBlock, 2},
		"zenuml\nopt {\n  A->B: x\n} else {\n  A->B: y\n}": {CodeBlock, 4},
		"zenuml\nreturn x":                  {CodeStatement, 2},
		"zenuml\nA->B: x\nB":                {CodeStatement, 3},
		"zenuml\nnew A\nnew A":              {CodeStatement, 3},
		"zenuml\nswitch(x) {\n  A->B: y\n}": {CodeStatement, 2},
		"zenuml\nA->>B: x":                  {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}
