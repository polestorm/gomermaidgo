package journey

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: JRN-001 JRN-002 JRN-003
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\njourney\n  title Dzień pracy\n  accDescr: Opis\n  Bez sekcji: 3\n  section Rano\n  Kawa: 5: Ja\n  Dojazd : 1 : Ja , Kot,\n")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Dzień pracy" || d.AccDescr != "Opis" || len(d.Tasks) != 3 {
		t.Fatalf("diagram: %+v", d)
	}
	if a := d.Tasks[0]; a.Name != "Bez sekcji" || a.Score != 3 || len(a.Actors) != 0 || a.Section != "" {
		t.Errorf("task without actors: %+v", a)
	}
	if c := d.Tasks[2]; c.Name != "Dojazd" || c.Score != 1 || strings.Join(c.Actors, "|") != "Ja|Kot" || c.Section != "Rano" || c.Line != 8 {
		t.Errorf("task with actors: %+v", c)
	}
}

// REQ: JRN-004
func TestParseErrors(t *testing.T) {
	many := "journey\n"
	for i := 0; i < 101; i++ {
		many += "krok: 3\n"
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                           {CodeHeader, 0},
		"journeys\nA: 1":             {CodeHeader, 1},
		"journey\nsam tekst":         {CodeStatement, 2},
		"journey\n: 3: Ja":           {CodeStatement, 2},
		"journey\nA: 3: Ja: za dużo": {CodeStatement, 2},
		"journey\nA: 0":              {CodeScore, 2},
		"journey\nA: 6: Ja":          {CodeScore, 2},
		"journey\nA: dobrze":         {CodeScore, 2},
		"journey\nA: 2.5":            {CodeScore, 2},
		"journey\nsection":           {CodeStatement, 2},
		"journey\ntitle Sam tytuł":   {CodeEmpty, 0},
		many:                         {CodeStatement, 102},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: JRN-002 JRN-003
// Znaczniki ocen stoją w kolumnach skali, jeden pod drugim w kolejności kroków; linia przechodzi
// przez wszystkie; wypełnienie znacznika odróżnia ocenę niską, średnią i wysoką bez koloru.
func TestLayout(t *testing.T) {
	d, err := Parse("journey\ntitle T\nsection S\nA: 1: X\nB: 3\nC: 5: X, Y\nD: 5")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var marks, scale []scene.Item
	var line scene.Item
	texts := map[string]int{}
	for _, it := range sc.Items {
		switch it.Role {
		case "journey-mark":
			marks = append(marks, it)
		case "journey-scale":
			scale = append(scale, it)
		case "journey-line":
			line = it
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	if len(marks) != 4 || len(scale) != 5 || len(line.Points) != 4 {
		t.Fatalf("%d marks, %d scale labels, %d line points", len(marks), len(scale), len(line.Points))
	}
	for i, want := range []int{1, 3, 5, 5} {
		if marks[i].X != scale[want-1].X {
			t.Errorf("mark %d stands at %.0f, the scale column %d at %.0f", i, marks[i].X, want, scale[want-1].X)
		}
		if i > 0 && marks[i].Y <= marks[i-1].Y {
			t.Errorf("mark %d is not below the previous one", i)
		}
		if line.Points[i].X != marks[i].X || line.Points[i].Y != marks[i].Y {
			t.Errorf("the line misses mark %d", i)
		}
	}
	if marks[0].Fill == marks[1].Fill || marks[1].Fill == marks[2].Fill || marks[2].Fill != marks[3].Fill {
		t.Errorf("fills should tell low, middle and high scores apart: %s %s %s %s", marks[0].Fill, marks[1].Fill, marks[2].Fill, marks[3].Fill)
	}
	for _, want := range []string{"T", "S", "A", "D", "X", "X, Y"} {
		if texts[want] == 0 {
			t.Errorf("text %q is missing", want)
		}
	}
}
