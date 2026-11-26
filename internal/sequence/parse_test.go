package sequence

import (
	"errors"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: SEQ-011
func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src string
		line      int
		code      string
	}{
		{"wrong header", "sequence\nA->>B: x", 1, CodeHeader},
		{"no header", "", 0, CodeHeader},
		{"no participants", "sequenceDiagram\n", 0, CodeEmpty},
		{"unknown statement", "sequenceDiagram\nA->>B: x\nfoo bar", 3, CodeStatement},
		{"message without text", "sequenceDiagram\nA->>B", 2, CodeStatement},
		{"links outside profile", "sequenceDiagram\nparticipant A\nlink A: Panel @ https://example.org", 3, CodeUnsupported},
		{"message inside box", "sequenceDiagram\nbox Grupa\nA->>B: x\nend", 3, CodeBlock},
		{"empty box", "sequenceDiagram\nbox Grupa\nend\nA->>B: x", 3, CodeBlock},
		{"unclosed box", "sequenceDiagram\nbox Grupa\nparticipant A", 0, CodeBlock},
		{"create without message", "sequenceDiagram\nA->>B: x\ncreate participant C", 0, CodeParticipant},
		{"create with unrelated message", "sequenceDiagram\nA->>B: x\ncreate participant C\nA->>B: y", 4, CodeParticipant},
		{"create without kind", "sequenceDiagram\nA->>B: x\ncreate C", 3, CodeStatement},
		{"destroy unknown", "sequenceDiagram\nA->>B: x\ndestroy C", 3, CodeParticipant},
		{"use after destroy", "sequenceDiagram\nA->>B: x\ndestroy B\nA-xB: koniec\nA->>B: y", 5, CodeParticipant},
		{"autonumber with text", "sequenceDiagram\nautonumber od dziesięciu\nA->>B: x", 2, CodeStatement},
		{"unclosed accDescr", "sequenceDiagram\naccDescr {\nopis\nA->>B: x", 0, CodeStatement},
		{"unclosed block", "sequenceDiagram\nloop zawsze\nA->>B: x", 0, CodeBlock},
		{"end without block", "sequenceDiagram\nA->>B: x\nend", 3, CodeBlock},
		{"else outside alt", "sequenceDiagram\nloop x\nA->>B: x\nelse y\nend", 4, CodeBlock},
		{"and outside par", "sequenceDiagram\nalt x\nA->>B: x\nand y\nend", 4, CodeBlock},
		{"rect without colour", "sequenceDiagram\nrect czerwony\nA->>B: x\nend", 2, CodeBlock},
		{"deactivate inactive", "sequenceDiagram\nA->>B: x\ndeactivate B", 3, CodeActivation},
		{"minus on inactive sender", "sequenceDiagram\nA->>B: x\nB-->>-A: y", 3, CodeActivation},
		{"activate unknown", "sequenceDiagram\nA->>B: x\nactivate C", 3, CodeActivation},
		{"duplicate participant", "sequenceDiagram\nparticipant A\nparticipant A", 3, CodeParticipant},
		{"note without text", "sequenceDiagram\nNote over A", 2, CodeStatement},
		{"note with bad placement", "sequenceDiagram\nNote under A: x", 2, CodeStatement},
		{"note right of two", "sequenceDiagram\nNote right of A,B: x", 2, CodeStatement},
	}
	for _, c := range cases {
		_, err := Parse(c.src)
		var se *syntax.Error
		if !errors.As(err, &se) {
			t.Errorf("%s: want a syntax error, got %v", c.name, err)
			continue
		}
		if se.Line != c.line || se.Code != c.code {
			t.Errorf("%s: got line %d code %q (%s), want line %d code %q", c.name, se.Line, se.Code, se.Message, c.line, c.code)
		}
	}
}

// REQ: SEQ-002 SEQ-003 SEQ-005
func TestParseMessage(t *testing.T) {
	d, err := Parse("sequenceDiagram\nparticipant Z as Zatwierdzający\nJan Kowalski-->>+Z: Proszę<br/>o podpis\nZ-x-Jan Kowalski: Odmowa")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Participants) != 2 || d.Participants[1].ID != "Jan Kowalski" || d.Participants[0].Label[0] != "Zatwierdzający" {
		t.Fatalf("participants: %+v %+v", d.Participants[0], d.Participants[1])
	}
	m := d.Events[0]
	if m.From != 1 || m.To != 0 || m.LineStyle != Dotted || m.Head != HeadArrow || !m.ActivateTo || len(m.Text) != 2 || m.Text[1] != "o podpis" {
		t.Errorf("first message: %+v", m)
	}
	m = d.Events[1]
	if m.From != 0 || m.To != 1 || m.LineStyle != Solid || m.Head != HeadCross || !m.DeactivateFrom {
		t.Errorf("second message: %+v", m)
	}
}

// REQ: SEQ-003
func TestParseArrows(t *testing.T) {
	want := map[string]struct {
		line LineStyle
		head HeadStyle
		both bool
	}{
		"->": {Solid, HeadNone, false}, "-->": {Dotted, HeadNone, false},
		"->>": {Solid, HeadArrow, false}, "-->>": {Dotted, HeadArrow, false},
		"-x": {Solid, HeadCross, false}, "--x": {Dotted, HeadCross, false},
		"-)": {Solid, HeadOpen, false}, "--)": {Dotted, HeadOpen, false},
		"<<->>": {Solid, HeadArrow, true}, "<<-->>": {Dotted, HeadArrow, true},
	}
	for arrow, w := range want {
		d, err := Parse("sequenceDiagram\nA" + arrow + "B: t")
		if err != nil {
			t.Errorf("%s: %v", arrow, err)
			continue
		}
		m := d.Events[0]
		if m.LineStyle != w.line || m.Head != w.head || m.Bidirectional != w.both || len(d.Participants) != 2 {
			t.Errorf("%s: got %+v", arrow, m)
		}
	}
}

// REQ: SEQ-014 SEQ-015 SEQ-016
func TestParseGroupsLifetimeAccessibility(t *testing.T) {
	d, err := Parse("sequenceDiagram\naccTitle: Obieg\naccDescr {\n  Pierwsza linia\n  druga linia\n}\nautonumber 10 5\n" +
		"box rgb(230, 240, 255) Dział jakości\nparticipant A\nactor B\nend\nbox Bez koloru\nparticipant C\nend\n" +
		"A->>B: x\ncreate participant D as Zadanie\nB->>D: utwórz\ndestroy D\nD-->>B: gotowe")
	if err != nil {
		t.Fatal(err)
	}
	if d.AccTitle != "Obieg" || d.AccDescr != "Pierwsza linia druga linia" || d.NumberStart != 10 || d.NumberStep != 5 {
		t.Errorf("accessibility or numbering: %+v", d)
	}
	if len(d.Boxes) != 2 || d.Boxes[0] != (Box{Title: "Dział jakości", Color: "rgb(230,240,255)", First: 0, Last: 1}) ||
		d.Boxes[1] != (Box{Title: "Bez koloru", First: 2, Last: 2}) {
		t.Errorf("boxes: %+v", d.Boxes)
	}
	p := d.Participants[3]
	if !p.Created || !p.Destroyed || p.Label[0] != "Zadanie" || d.Events[1].Created != 4 || d.Events[2].Destroyed != 4 {
		t.Errorf("lifetime: %+v %+v %+v", p, d.Events[1], d.Events[2])
	}
}

// REQ: SEQ-002
func TestParticipantTypes(t *testing.T) {
	d, err := Parse("sequenceDiagram\nparticipant A@{ \"type\": \"database\" } as Rejestr\nparticipant B@{ \"type\" : \"actor\" }\nparticipant C@{ \"type\": \"queue\" }\nA->>B: x\nB->>C: y")
	if err != nil {
		t.Fatal(err)
	}
	ps := d.Participants
	if ps[0].Kind != "database" || ps[0].Label[0] != "Rejestr" || !ps[1].Actor || ps[1].Kind != "" || ps[2].Kind != "queue" {
		t.Errorf("participants: %+v %+v %+v", ps[0], ps[1], ps[2])
	}
	for _, src := range []string{"sequenceDiagram\nparticipant A@{ \"type\": \"robot\" }", "sequenceDiagram\nparticipant A@{ \"shape\": \"x\" }"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("%q: expected an error", src)
		}
	}
}
