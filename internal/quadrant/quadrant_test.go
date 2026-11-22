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
	c, err := Parse("%% komentarz\nquadrantChart\n  title Tytuł\n  accDescr: Opis\n  x-axis Mało --> Dużo\n  y-axis \"Tylko dół\"\n  quadrant-1 Pierwsza\n  quadrant-3 \"Trzecia\"\n  Punkt A: [0.3, 0.6]\n  Punkt B:::k: [1, 0] radius: 12, color: #ff3300, nieznana: 1, stroke-width: zła\n  \"C\": [0,1]\n  classDef k color: #109060, stroke-color: navy, stroke-width: 2px, radius: 900")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "Tytuł" || c.AccDescr != "Opis" || c.XLeft != "Mało" || c.XRight != "Dużo" || c.YBottom != "Tylko dół" || c.YTop != "" {
		t.Errorf("titles and axes: %+v", c)
	}
	if c.Quadrants != [4]string{"Pierwsza", "", "Trzecia", ""} {
		t.Errorf("quadrants: %q", c.Quadrants)
	}
	// Własności przy punkcie mają pierwszeństwo przed klasą; klasa może być zdefiniowana po punkcie.
	want := []Point{{"Punkt A", 0.3, 0.6, 9, Style{}}, {"Punkt B", 1, 0, 10, Style{Radius: 12, Color: "#ff3300", StrokeColor: "navy", StrokeWidth: 2}}, {"C", 0, 1, 11, Style{}}}
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
// Punkt stoi w swoim miejscu kwadratu z wyglądem ze źródła, nazwa mniejszym pismem pod kropką
// i w obrębie wykresu, tytuł ćwiartki u jej góry, opisy osi pionowej obrócone.
func TestLayout(t *testing.T) {
	c, err := Parse("quadrantChart\ntitle T\nx-axis Lewo --> Prawo\ny-axis Dół --> Góra\nquadrant-1 Q1\nquadrant-2 Q2\nquadrant-3 Q3\nquadrant-4 Q4\nŚrodek: [0.5, 0.5]\nRóg: [1, 1] radius: 10, color: #b42318, stroke-color: #000000, stroke-width: 3\nGóra ćwiartki: [0.75, 0.95]\nLewy dół: [0, 0]")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16, Shadows: true})
	var frame scene.Box
	var points, grounds, titles, names, axes []scene.Item
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
		case "quadrant-point-label":
			names = append(names, it)
		case "quadrant-title":
			titles = append(titles, it)
		case "axis-label":
			axes = append(axes, it)
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	size := frame.X1 - frame.X0
	if len(points) != 4 || len(titles) != 4 || len(names) != 4 || math.Abs(size-(frame.Y1-frame.Y0)) > 0.01 {
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
	if p := points[1]; p.R != 10 || p.Fill != "#b42318" || p.Stroke != "#000000" || p.StrokeWidth != 3 || points[0].R != 5 || points[0].Fill == p.Fill {
		t.Errorf("point style from the source: %+v, default %+v", p, points[0])
	}
	if len(grounds) != 0 {
		t.Errorf("point names should have no ground: %d", len(grounds))
	}
	overlap := func(a, b scene.Box) bool { return a.X0 < b.X1 && b.X0 < a.X1 && a.Y0 < b.Y1 && b.Y0 < a.Y1 }
	for i, name := range names {
		b := name.Bounds()
		if name.FontSize != 12 {
			t.Errorf("name %q: size %v", name.Text, name.FontSize)
		}
		if b.X0 < frame.X0-0.5 || b.X1 > frame.X1+0.5 || b.Y0 < frame.Y0-0.5 || b.Y1 > frame.Y1+0.5 {
			t.Errorf("the name of point %d leaves the chart: %+v", i, b)
		}
		// Nazwa nie nachodzi na tytuł ćwiartki, żadną kropkę ani inną nazwę.
		for _, title := range titles {
			if overlap(b, title.Bounds()) {
				t.Errorf("name %q lies on the quadrant title %q", name.Text, title.Text)
			}
		}
		for j := range points {
			if overlap(b, points[j].Bounds()) || (j != i && overlap(b, names[j].Bounds())) {
				t.Errorf("name %q lies on point %d or its name", name.Text, j)
			}
		}
	}
	// Nazwa stoi pod kropką, gdy nic jej tam nie przeszkadza.
	if b := names[0].Bounds(); b.Y0 < points[0].Y+points[0].R-1 || math.Abs((b.X0+b.X1)/2-points[0].X) > 0.01 {
		t.Errorf("the name of the middle point should stand under it: %+v", b)
	}
	// Tytuły stoją u góry swoich ćwiartek; tytuł pierwszej ustąpił kropce leżącej u jej góry
	// i zaczyna się tuż pod nią, a żaden nie leży pod kropką.
	if titles[1].Y > frame.Y0+40 || titles[2].Y != titles[3].Y || titles[2].Y < frame.Y0+size/2 || titles[2].Y > frame.Y0+size/2+40 {
		t.Errorf("quadrant titles: %v %v %v %v", titles[0].Y, titles[1].Y, titles[2].Y, titles[3].Y)
	}
	if tb := titles[0].Bounds(); tb.Y0 < points[2].Y+points[2].R-4 || tb.Y0 > points[2].Y+points[2].R+12 {
		t.Errorf("the title of the first quadrant should start just under the point at its top: %+v", tb)
	}
	for _, title := range titles {
		for j := range points {
			if overlap(title.Bounds(), points[j].Bounds()) {
				t.Errorf("quadrant title %q lies under point %d", title.Text, j)
			}
		}
	}
	// Opisy osi pionowej: obrócone, z lewej strony wykresu, każdy na wysokości swojej połowy.
	vertical := 0
	for _, a := range axes {
		if !a.Vertical {
			continue
		}
		vertical++
		b := a.Bounds()
		mid := frame.Y0 + size/4
		if a.Text == "Dół" {
			mid = frame.Y0 + 3*size/4
		}
		if b.X1 > frame.X0 || b.X0 < sc.Bounds().X0 || math.Abs((b.Y0+b.Y1)/2-mid) > 0.01 || b.Y1-b.Y0 < b.X1-b.X0 {
			t.Errorf("axis label %q: %+v", a.Text, b)
		}
	}
	if vertical != 2 {
		t.Errorf("vertical axis labels: %d", vertical)
	}
	for _, want := range []string{"T", "Lewo", "Prawo", "Dół", "Góra", "Q1", "Q4", "Środek", "Róg", "Lewy dół"} {
		if texts[want] != 1 {
			t.Errorf("text %q drawn %d times, want once", want, texts[want])
		}
	}
}
