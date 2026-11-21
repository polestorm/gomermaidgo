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
		{"box outside profile", "sequenceDiagram\nbox Grupa\nparticipant A\nend", 2, CodeUnsupported},
		{"create outside profile", "sequenceDiagram\nA->>B: x\ncreate participant C", 3, CodeUnsupported},
		{"autonumber with start", "sequenceDiagram\nautonumber 10 5\nA->>B: x", 2, CodeUnsupported},
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
