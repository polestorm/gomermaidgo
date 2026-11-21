package radar

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: RDR-001 RDR-002
func TestParse(t *testing.T) {
	c, err := Parse("%% komentarz\nradar-beta\n  title \"Oceny\"\n  accTitle: Dostępny\n  axis a[\"Pierwsza\"], b\n  axis c[\"Trzecia\"]\n  curve x[\"Seria X\"]{1, 2, 3}\n  curve y{c: 30, a: 10, b: 20}\n  min 1\n  ticks 4\n  graticule polygon\n  showLegend false")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "Oceny" || c.AccTitle != "Dostępny" || len(c.Axes) != 3 || c.Axes[0] != (Axis{"a", "Pierwsza"}) || c.Axes[1] != (Axis{"b", "b"}) {
		t.Errorf("title and axes: %+v", c)
	}
	if len(c.Curves) != 2 || c.Curves[0].Label != "Seria X" || c.Curves[1].Label != "y" || c.Curves[1].Values[0] != 10 || c.Curves[1].Values[2] != 30 {
		t.Errorf("curves: %+v", c.Curves)
	}
	if c.Min != 1 || c.Max != 30 || c.HasMax || c.Ticks != 4 || !c.Polygon || c.Legend {
		t.Errorf("options: %+v", c)
	}
}

// REQ: RDR-004
func TestParseErrors(t *testing.T) {
	head := "radar-beta\naxis a, b, c\n"
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                    {CodeHeader, 0},
		"radars\naxis a":                      {CodeHeader, 1},
		head + "coś innego":                   {CodeStatement, 3},
		"radar-beta\naxis a, b\ncurve x{1,2}": {CodeEmpty, 0},
		head + "max 5":                        {CodeEmpty, 0},
		head + "curve x{1, 2}":                {CodeData, 3},
		head + "curve x{1, 2, trzy}":          {CodeData, 3},
		head + "curve x{a: 1, b: 2, d: 3}":    {CodeData, 3},
		head + "curve x{a: 1, a: 2, c: 3}":    {CodeData, 3},
		head + "curve x 1 2 3":                {CodeData, 3},
		head + "curve x{1, 2, 3}\nmax 0":      {CodeData, 0},
		head + "ticks 0":                      {CodeStatement, 3},
		head + "graticule kwadrat":            {CodeStatement, 3},
		head + "axis \"zła\"":                 {CodeStatement, 3},
		"radar-beta\naxis " + strings.TrimSuffix(strings.Repeat("a,", 25), ","): {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: RDR-003
func TestLayout(t *testing.T) {
	c, err := Parse("radar-beta\ntitle T\naxis g[\"Góra\"], p[\"Prawo\"], d[\"Dół\"], l[\"Lewo\"]\ncurve a[\"Pełna\"]{10, 10, 10, 10}\ncurve b[\"Połowa\"]{5, 0, 5, 20}\nmax 10")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var curves, grid []scene.Item
	texts := map[string]int{}
	for _, it := range sc.Items {
		switch it.Role {
		case "radar-curve":
			curves = append(curves, it)
		case "radar-grid":
			grid = append(grid, it)
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	if len(curves) != 2 || len(grid) != 5 || grid[0].Kind != scene.Circle {
		t.Fatalf("%d curves, %d grid rings", len(curves), len(grid))
	}
	cx, cy, r := grid[4].X, grid[4].Y, grid[4].R
	full, half := curves[0].Points, curves[1].Points
	near := func(p scene.Point, x, y float64) bool { return math.Abs(p.X-x) < 0.01 && math.Abs(p.Y-y) < 0.01 }
	if !near(full[0], cx, cy-r) || !near(full[1], cx+r, cy) || !near(full[2], cx, cy+r) || !near(full[3], cx-r, cy) {
		t.Errorf("the full curve should touch the outer ring up, right, down and left: %+v", full)
	}
	// Połowa zakresu to połowa promienia; zero to środek; wartość ponad zakres zostaje na brzegu.
	if !near(half[0], cx, cy-r/2) || !near(half[1], cx, cy) || !near(half[3], cx-r, cy) {
		t.Errorf("distances should follow the values: %+v", half)
	}
	if curves[0].Dash == curves[1].Dash {
		t.Error("curves should differ in dash pattern")
	}
	for _, want := range []string{"T", "Góra", "Prawo", "Dół", "Lewo", "Pełna", "Połowa"} {
		if texts[want] != 1 {
			t.Errorf("text %q drawn %d times, want once", want, texts[want])
		}
	}
}
