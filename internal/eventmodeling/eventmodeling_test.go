package eventmodeling

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: EVM-001 EVM-002 EVM-003
func TestParse(t *testing.T) {
	m, err := Parse("%% c\neventmodeling\naccTitle: Opis\ntf 01 ui Ekran\ntimeframe 02 command Sklep.Dodaj `json`{ opis: string, ilosc: 2 }\ntf 03 evt Dodano [[D1]]\nrf 04 rmo Lista ->> 02 ->> 03\ndata D1 {\n  klient: 'Jan'\n}")
	if err != nil {
		t.Fatal(err)
	}
	if m.AccTitle != "Opis" || len(m.Frames) != 4 {
		t.Fatalf("model: %+v", m)
	}
	f := m.Frames[1]
	if f.Kind != Command || f.Namespace != "Sklep" || f.Name != "Dodaj" || strings.Join(f.Data, "|") != "opis: string|ilosc: 2" {
		t.Errorf("frame 2: %+v", f)
	}
	if d := m.Frames[2].Data; strings.Join(d, "|") != "klient: Jan" {
		t.Errorf("data block: %q", d)
	}
	if r := m.Frames[3]; !r.Reset || r.Kind != ReadModel || len(r.From) != 2 || r.From[1] != 3 {
		t.Errorf("reset frame: %+v", r)
	}
}

// REQ: EVM-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                      {CodeHeader, 0},
		"eventModeling":                         {CodeHeader, 1},
		"eventmodeling\n%% nic":                 {CodeEmpty, 0},
		"eventmodeling\ntf 01 widget A":         {CodeStatement, 2},
		"eventmodeling\ntf 01 ui A\ntf 1 evt B": {CodeStatement, 3},
		"eventmodeling\ntf 01 ui A ->> 07":      {CodeStatement, 2},
		"eventmodeling\ntf 01 ui A [[X]]":       {CodeStatement, 2},
		"eventmodeling\ntf 01 ui A { a: b":      {CodeStatement, 2},
		"eventmodeling\ndata X {\n a: 1":        {CodeStatement, 2},
		"eventmodeling\nslice 1":                {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: EVM-004
func TestLayout(t *testing.T) {
	m, err := Parse("eventmodeling\ntf 01 ui Ekran\ntf 02 cmd Dodaj\ntf 03 evt Dodano\nrf 04 rmo Lista ->> 03\ntf 05 evt Inne")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var xs, ys []float64
	edges := 0
	for _, it := range sc.Items {
		switch it.Role {
		case "eventmodeling-element":
			xs, ys = append(xs, it.X), append(ys, it.Y)
		case "eventmodeling-edge":
			edges++
		}
	}
	if len(xs) != 5 {
		t.Fatalf("%d elements", len(xs))
	}
	for i := 1; i < len(xs); i++ {
		if xs[i] <= xs[i-1] {
			t.Errorf("frame %d is not right of frame %d", i+1, i)
		}
	}
	// Tory: ekran nad poleceniem, polecenie nad zdarzeniem; model odczytu w torze polecenia.
	if !(ys[0] < ys[1] && ys[1] < ys[2] && ys[3] == ys[1] && ys[4] == ys[2]) {
		t.Errorf("lanes: %v", ys)
	}
	// 1→2, 2→3, 3→4 (jawne; reset bez wywnioskowanego), 4→5: linia i grot każda.
	if edges != 8 {
		t.Errorf("%d edge items, want 8", edges)
	}
}
