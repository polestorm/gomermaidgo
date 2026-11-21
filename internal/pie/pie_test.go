package pie

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: PIE-001 PIE-002 PIE-004
func TestParse(t *testing.T) {
	c, err := Parse("%% komentarz\npie showData title W nagłówku\n  title Niżej wygrywa\n  accTitle: Dostępny\n  \"A b\" : 3\n  \"C\":0.5\n\n  \"D\" : 0\n")
	if err != nil {
		t.Fatal(err)
	}
	if !c.ShowData || c.Title != "Niżej wygrywa" || c.AccTitle != "Dostępny" || len(c.Slices) != 3 {
		t.Fatalf("chart: %+v", c)
	}
	if s := c.Slices[1]; s.Label != "C" || s.Value != 0.5 || s.Text != "0.5" || s.Line != 6 {
		t.Errorf("slice: %+v", s)
	}
	if c, err := Parse("pie\n\"X\" : 1"); err != nil || c.ShowData || c.Title != "" {
		t.Errorf("plain header: %+v %v", c, err)
	}
}

// REQ: PIE-005
func TestParseErrors(t *testing.T) {
	many := "pie\n"
	for i := 0; i < 25; i++ {
		many += "\"s" + string(rune('a'+i)) + "\" : 1\n"
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                             {CodeHeader, 0},
		"pies\n\"A\" : 1":              {CodeHeader, 1},
		"pie\nA : 1":                   {CodeStatement, 2},
		"pie\n\"A\" : dużo":            {CodeValue, 2},
		"pie\n\"A\" : -1":              {CodeValue, 2},
		"pie\n\"A\" : NaN":             {CodeValue, 2},
		"pie\n\" \" : 1":               {CodeStatement, 2},
		"pie\n\"A\" : 1\n\"A\" : 2":    {CodeStatement, 3},
		"pie\n\"A\" : 0":               {CodeEmpty, 0},
		"pie title Sam tytuł":          {CodeEmpty, 0},
		"pie\naccDescr {\nbez końca\n": {CodeStatement, 0},
		many:                           {CodeStatement, 26},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: PIE-002 PIE-003 PIE-004
// Kąty wycinków są proporcjonalne do wartości, legenda ma wszystkie etykiety w kolejności,
// a udział stoi tylko na wycinkach dość dużych.
func TestLayout(t *testing.T) {
	c, err := Parse("pie showData\n\"Ćwierć\" : 25\n\"Połowa\" : 50\n\"Mało\" : 1\n\"Reszta\" : 24\n\"Zero\" : 0")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var slices, legend, percents []scene.Item
	for _, it := range sc.Items {
		switch it.Role {
		case "pie-slice":
			slices = append(slices, it)
		case "pie-legend":
			legend = append(legend, it)
		case "pie-percent":
			percents = append(percents, it)
		}
	}
	if len(slices) != 4 {
		t.Fatalf("%d slices drawn, want 4 (the zero slice has none)", len(slices))
	}
	// Pierwszy wycinek: od godziny dwunastej do trzeciej.
	first := slices[0].Segments
	centre, start, end := first[0].To, first[1].To, first[len(first)-2].To
	if math.Abs(start.X-centre.X) > 0.01 || start.Y >= centre.Y || math.Abs(end.Y-centre.Y) > 0.01 || end.X <= centre.X {
		t.Errorf("first slice should run from twelve to three o'clock: %+v %+v %+v", centre, start, end)
	}
	// Połowa koła składa się z dwóch krzywych ćwierćobrotowych.
	if curves := len(slices[1].Segments) - 3; curves != 2 {
		t.Errorf("half of the circle uses %d curves, want 2", curves)
	}
	want := []string{"Ćwierć [25]", "Połowa [50]", "Mało [1]", "Reszta [24]", "Zero [0]"}
	for i, it := range legend {
		if i >= len(want) || it.Text != want[i] {
			t.Errorf("legend row %d: %q", i, it.Text)
		}
	}
	var texts []string
	for _, it := range percents {
		texts = append(texts, it.Text)
	}
	if got := strings.Join(texts, " "); got != "25% 50% 24%" {
		t.Errorf("percent labels: %q", got)
	}
	for share, want := range map[float64]string{0.625: "63%", 0.05: "5%", 0.0425: "4.3%", 1: "100%"} {
		if got := Percent(share); got != want {
			t.Errorf("Percent(%v) = %q, want %q", share, got, want)
		}
	}
}

// REQ: PIE-003
// Jeden wycinek zajmuje całe koło; sąsiednie wypełnienia różnią się jasnością.
func TestWholeCircleAndTones(t *testing.T) {
	c, _ := Parse("pie\n\"Wszystko\" : 7")
	sc := Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	found := false
	for _, it := range sc.Items {
		found = found || (it.Role == "pie-slice" && it.Kind == scene.Circle)
	}
	if !found {
		t.Error("a single slice should be drawn as a full circle")
	}
	for i := range grays {
		if dark(grays[i]) == dark(grays[(i+1)%len(grays)]) {
			t.Errorf("neighbouring tones %s and %s have the same lightness class", grays[i], grays[(i+1)%len(grays)])
		}
	}
}
