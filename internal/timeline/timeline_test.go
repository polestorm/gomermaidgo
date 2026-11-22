package timeline

import (
	"math"
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

// REQ: TML-002 TML-003 TML-006
// Układ listowy. Okresy stoją w kolumnie o wspólnej szerokości, jeden pod drugim; zdarzenia mieszczą się
// w szerokości strony; żadne dwa pola się nie nakładają; każdy tekst jest na rysunku.
func TestLayout(t *testing.T) {
	src := "timeline\n%% layout: list\ntitle Oś\nsection Pierwsza\nA : " + strings.Repeat("zdarzenie na osi : ", 7) + "ostatnie\nBardzo długa nazwa okresu która się złamie : x\nsection Druga\nC"
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

// REQ: TML-002 TML-003 TML-004
// Układ domyślny: okresy w rzędzie w kolumnach o wspólnej szerokości, zdarzenia okresu pod osią,
// jedno pod drugim w kolumnie okresu; nagłówek sekcji nad jej okresami; pola rozłączne.
func TestRowLayout(t *testing.T) {
	d, err := Parse("timeline\ntitle Oś\nsection Pierwsza\nA : a1 : a2 dłuższe zdarzenie które się złamie : a3\nB\nsection Druga\nC : c1")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []Options{{Measure: font.ApproxTextWidth, FontSize: 16}, {Measure: font.ApproxTextWidth, FontSize: 16, Colors: true, Shadows: true}} {
		sc := Layout(d, o)
		var periods, events, sections, boxes []scene.Item
		var axis scene.Item
		texts := map[string]int{}
		for _, it := range sc.Items {
			switch it.Role {
			case "timeline-period":
				periods = append(periods, it)
				boxes = append(boxes, it)
			case "timeline-event":
				events = append(events, it)
				boxes = append(boxes, it)
			case "timeline-section":
				sections = append(sections, it)
			case "timeline-axis":
				if it.Kind == scene.Line {
					axis = it
				}
			}
			if it.Kind == scene.Text {
				texts[it.Text]++
			}
		}
		if len(periods) != 3 || len(events) != 4 || len(sections) != 2 {
			t.Fatalf("%d periods, %d events, %d sections", len(periods), len(events), len(sections))
		}
		for i, p := range periods {
			if p.Y != periods[0].Y || p.W != periods[0].W || (i > 0 && p.X < periods[i-1].X+periods[i-1].W) {
				t.Errorf("period %d is not in the row after the previous one", i)
			}
			if p.Y+p.H >= axis.Y {
				t.Errorf("period %d is not above the axis", i)
			}
		}
		for i, e := range events[:3] {
			if e.X != periods[0].X || e.Y <= axis.Y || (i > 0 && e.Y < events[i-1].Y+events[i-1].H) {
				t.Errorf("event %d is not stacked under the axis in the column of its period", i)
			}
		}
		if events[3].X != periods[2].X {
			t.Errorf("the event of the last period stands in another column")
		}
		if s := sections[0]; s.X != periods[0].X || s.X+s.W != periods[1].X+periods[1].W || s.Y+s.H > periods[0].Y {
			t.Errorf("section header does not span its periods: %+v", s.Bounds())
		}
		for i := range boxes {
			a := boxes[i].Bounds()
			for j := i + 1; j < len(boxes); j++ {
				b := boxes[j].Bounds()
				if a.X0 < b.X1-0.5 && b.X0 < a.X1-0.5 && a.Y0 < b.Y1-0.5 && b.Y0 < a.Y1-0.5 {
					t.Errorf("boxes overlap: %+v and %+v", a, b)
				}
			}
		}
		for _, want := range []string{"Oś", "Pierwsza", "Druga", "A", "B", "C", "a1", "a3", "c1"} {
			if texts[want] != 1 {
				t.Errorf("text %q drawn %d times, want once", want, texts[want])
			}
		}
	}
}

// REQ: TML-006
// Układ listowy: zdarzenia okresu stoją jedno pod drugim w kolejności ze źródła, w kolumnie
// o wspólnej szerokości, a linia przerywana przez kolumnę łączy pierwsze z ostatnim.
func TestListLayoutStacksEvents(t *testing.T) {
	d, err := Parse("timeline\n%% layout: list\nOkres : Pierwsze : Drugie zdarzenie okresu o dłuższej nazwie : Trzecie\nNastępny : Jedno")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var events, stems, periods []scene.Item
	var texts []string
	for _, it := range sc.Items {
		switch it.Role {
		case "timeline-event":
			events = append(events, it)
		case "timeline-stem":
			stems = append(stems, it)
		case "timeline-period":
			periods = append(periods, it)
		case "timeline-event-label":
			texts = append(texts, it.Text)
		}
	}
	if len(events) != 4 || len(stems) != 1 || len(periods) != 2 {
		t.Fatalf("%d events, %d stems, %d periods", len(events), len(stems), len(periods))
	}
	for i, e := range events {
		if e.X != events[0].X || e.W != events[0].W || e.X < periods[0].X+periods[0].W {
			t.Errorf("event %d should stand in the common column right of the periods: %+v", i, e)
		}
		if i > 0 && e.Y < events[i-1].Y+events[i-1].H {
			t.Errorf("event %d should stand below event %d", i, i-1)
		}
	}
	if texts[0] != "Pierwsze" || texts[len(texts)-1] != "Jedno" {
		t.Errorf("events are out of source order: %q", texts)
	}
	// Linia biegnie pionowo od pierwszego do ostatniego zdarzenia okresu, a następny okres zaczyna się pod nią.
	st := stems[0]
	if st.X != st.X2 || st.Dash == "" || math.Abs(st.Y-(events[0].Y+events[0].H)) > 0.01 || math.Abs(st.Y2-events[2].Y) > 0.01 || periods[1].Y < events[2].Y+events[2].H {
		t.Errorf("stem: %+v", st)
	}
}
