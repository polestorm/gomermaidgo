package quadrant

import (
	"math"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: QDR-001 QDR-002
func TestParse(t *testing.T) {
	c, err := Parse("%% komentarz\nquadrantChart\n  title Tytuł\n  accDescr: Opis\n  x-axis Mało --> Dużo\n  y-axis \"Tylko dół\"\n  quadrant-1 Pierwsza\n  quadrant-3 \"Trzecia\"\n  classDef k color: #109060\n  Punkt A: [0.3, 0.6]\n  Punkt B:::k: [1, 0] radius: 12, color: #ff3300\n  \"C\": [0,1]")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "Tytuł" || c.AccDescr != "Opis" || c.XLeft != "Mało" || c.XRight != "Dużo" || c.YBottom != "Tylko dół" || c.YTop != "" {
		t.Errorf("titles and axes: %+v", c)
	}
	if c.Quadrants != [4]string{"Pierwsza", "", "Trzecia", ""} {
		t.Errorf("quadrants: %q", c.Quadrants)
	}
	want := []Point{{"Punkt A", 0.3, 0.6, 10}, {"Punkt B", 1, 0, 11}, {"C", 0, 1, 12}}
	if len(c.Points) != len(want) {
		t.Fatalf("points: %+v", c.Points)
	}
	for i, p := range c.Points {
		if p != want[i] {
			t.Errorf("point %d: got %+v, want %+v", i, p, want[i])
		}
	}
}

// REQ: QDR-004
func TestParseErrors(t *testing.T) {
	many := "quadrantChart\n"
	for i := 0; i < 101; i++ {
		many += "P: [0.5, 0.5]\n"
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                              {CodeHeader, 0},
		"quadrant\nA: [0, 0]":           {CodeHeader, 1},
		"quadrantChart\nsam tekst":      {CodeStatement, 2},
		"quadrantChart\nx-axis":         {CodeStatement, 2},
		"quadrantChart\nquadrant-5 Zła": {CodeStatement, 2},
		"quadrantChart\nA: [1.2, 0]":    {CodePoint, 2},
		"quadrantChart\nA: [0.5, -0.1]": {CodePoint, 2},
		"quadrantChart\nA: [pół, 0.5]":  {CodePoint, 2},
		"quadrantChart\nA: [NaN, 0.5]":  {CodePoint, 2},
		many:                            {CodePoint, 102},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: QDR-003
// Punkt stoi w swoim miejscu kwadratu, jego nazwa mieści się w wykresie, a tytuł ćwiartki nie
// leży pod żadnym punktem ani nazwą.
func TestLayout(t *testing.T) {
	c, err := Parse("quadrantChart\ntitle T\nx-axis Lewo --> Prawo\ny-axis Dół --> Góra\nquadrant-1 Q1\nquadrant-2 Q2\nquadrant-3 Q3\nquadrant-4 Q4\nŚrodek: [0.5, 0.5]\nRóg: [1, 1]\nGóra ćwiartki: [0.75, 0.93]\nLewy dół: [0, 0]")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16, Shadows: true})
	var frame scene.Box
	var points, grounds, titles []scene.Item
	texts := map[string]int{}
	for _, it := range sc.Items {
		switch it.Role {
		case "quadrant-frame":
			if it.Kind == scene.Rect {
				frame = it.Bounds()
			}
		case "quadrant-point":
			points = append(points, it)
		case "quadrant-point-background":
			grounds = append(grounds, it)
		case "quadrant-title":
			titles = append(titles, it)
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	size := frame.X1 - frame.X0
	if len(points) != 4 || len(titles) != 4 || math.Abs(size-(frame.Y1-frame.Y0)) > 0.01 {
		t.Fatalf("%d points, %d titles, frame %+v", len(points), len(titles), frame)
	}
	at := func(i int, x, y float64) {
		if math.Abs(points[i].X-(frame.X0+x*size)) > 0.01 || math.Abs(points[i].Y-(frame.Y1-y*size)) > 0.01 {
			t.Errorf("point %d stands at (%.1f, %.1f)", i, points[i].X, points[i].Y)
		}
	}
	at(0, 0.5, 0.5)
	at(1, 1, 1)
	at(3, 0, 0)
	for i, g := range grounds {
		b := g.Bounds()
		if b.X1 > frame.X1+0.5 {
			t.Errorf("the name of point %d runs past the right edge", i)
		}
		for _, title := range titles {
			tb := title.Bounds()
			if b.X0 < tb.X1 && tb.X0 < b.X1 && b.Y0 < tb.Y1 && tb.Y0 < b.Y1 {
				t.Errorf("quadrant title %q lies under the name of point %d", title.Text, i)
			}
		}
	}
	// Tytuł pierwszej ćwiartki ustąpił punktowi u jej góry.
	if titles[0].Text != "Q1" || titles[0].Y < frame.Y0+size/4 {
		t.Errorf("the title of the first quadrant should move away from the point at its top: y=%.0f", titles[0].Y)
	}
	for _, want := range []string{"T", "Lewo", "Prawo", "Dół", "Góra", "Q1", "Q4", "Środek", "Róg", "Lewy dół"} {
		if texts[want] != 1 {
			t.Errorf("text %q drawn %d times, want once", want, texts[want])
		}
	}
}
