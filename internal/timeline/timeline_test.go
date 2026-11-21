package timeline

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: TML-001 TML-002 TML-003
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\ntimeline LR\n  title Tytuł osi\n  accTitle: Dostępny\n  2024 : pierwsze : drugie<br/>w dwóch liniach\n       : trzecie\n  section Później\n  2025\n  2026 : jedno :  : ")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Tytuł osi" || d.AccTitle != "Dostępny" || len(d.Periods) != 3 {
		t.Fatalf("diagram: %+v", d)
	}
	p := d.Periods[0]
	if p.Label[0] != "2024" || len(p.Events) != 3 || strings.Join(p.Events[1], "|") != "drugie|w dwóch liniach" || p.Section != "" {
		t.Errorf("first period: %+v", p)
	}
	if q := d.Periods[1]; q.Section != "Później" || len(q.Events) != 0 {
		t.Errorf("period without events: %+v", q)
	}
	if len(d.Periods[2].Events) != 1 {
		t.Errorf("empty events should be skipped: %+v", d.Periods[2])
	}
}

// REQ: TML-005
func TestParseErrors(t *testing.T) {
	many := "timeline\n"
	for i := 0; i < 101; i++ {
		many += "okres\n"
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                    {CodeHeader, 0},
		"timelines\n2024 : x":                 {CodeHeader, 1},
		"timeline\n : bez okresu":             {CodeStatement, 2},
		"timeline\n2024 : a\nsection S\n : b": {CodeStatement, 4},
		"timeline\nsection":                   {CodeStatement, 2},
		"timeline\n<br/> : a":                 {CodeStatement, 2},
		"timeline\ntitle Sam tytuł":           {CodeEmpty, 0},
		"timeline\n2024" + strings.Repeat(" : e", 21): {CodeStatement, 2},
		many: {CodeStatement, 102},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: TML-002 TML-003 TML-004
// Okresy stoją w kolumnie o wspólnej szerokości, jeden pod drugim; zdarzenia mieszczą się
// w szerokości strony; żadne dwa pola się nie nakładają; każdy tekst jest na rysunku.
func TestLayout(t *testing.T) {
	src := "timeline\ntitle Oś\nsection Pierwsza\nA : " + strings.Repeat("zdarzenie na osi : ", 7) + "ostatnie\nBardzo długa nazwa okresu która się złamie : x\nsection Druga\nC"
	d, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []Options{{Measure: font.ApproxTextWidth, FontSize: 16}, {Measure: font.ApproxTextWidth, FontSize: 16, Colors: true, Shadows: true}} {
		sc := Layout(d, o)
		var boxes []scene.Item
		var periods []scene.Item
		texts := map[string]int{}
		for _, it := range sc.Items {
			switch it.Role {
			case "timeline-period":
				periods = append(periods, it)
				boxes = append(boxes, it)
			case "timeline-event":
				boxes = append(boxes, it)
			}
			if it.Kind == scene.Text {
				texts[it.Text]++
			}
		}
		if len(periods) != 3 || len(boxes) != 3+9 {
			t.Fatalf("%d periods and %d boxes", len(periods), len(boxes))
		}
		for i, p := range periods {
			if p.X != periods[0].X || p.W != periods[0].W || (i > 0 && p.Y < periods[i-1].Y+periods[i-1].H) {
				t.Errorf("period %d is not in the column below the previous one: %+v", i, p.Bounds())
			}
		}
		for i := range boxes {
			a := boxes[i].Bounds()
			if boxes[i].Role == "timeline-event" && a.X1 > 650.5 {
				t.Errorf("event reaches %.0f px, beyond the page width", a.X1)
			}
			for j := i + 1; j < len(boxes); j++ {
				b := boxes[j].Bounds()
				if a.X0 < b.X1-0.5 && b.X0 < a.X1-0.5 && a.Y0 < b.Y1-0.5 && b.Y0 < a.Y1-0.5 {
					t.Errorf("boxes overlap: %+v and %+v", a, b)
				}
			}
		}
		for _, want := range []string{"Oś", "Pierwsza", "Druga", "A", "C", "ostatnie", "x"} {
			if texts[want] == 0 {
				t.Errorf("text %q is missing", want)
			}
		}
		if texts["zdarzenie na osi"] != 7 {
			t.Errorf("repeated event drawn %d times, want 7", texts["zdarzenie na osi"])
		}
	}
}
