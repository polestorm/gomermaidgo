package xychart

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: XY-001 XY-002
func TestParse(t *testing.T) {
	c, err := Parse("%% komentarz\nxychart-beta\n  title \"Tytuł wykresu\"\n  accTitle: Dostępny\n  x-axis \"Miesiąc\" [sty, \"lut y\", mar]\n  y-axis Sztuki 0 --> 50\n  bar [1, 2.5, -3]\n  line \"Trend\" [2, 2, 2]\n  bar[4,5,6]")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "Tytuł wykresu" || c.AccTitle != "Dostępny" || c.XTitle != "Miesiąc" || c.YTitle != "Sztuki" {
		t.Errorf("titles: %+v", c)
	}
	if strings.Join(c.Categories, "|") != "sty|lut y|mar" || c.YRange == nil || *c.YRange != [2]float64{0, 50} || c.XRange != nil {
		t.Errorf("axes: %q %v %v", c.Categories, c.YRange, c.XRange)
	}
	if len(c.Series) != 3 || !c.Series[0].Bar || c.Series[0].Values[2] != -3 || c.Series[1].Bar || c.Series[1].Name != "Trend" || c.Series[2].Values[0] != 4 {
		t.Errorf("series: %+v", c.Series)
	}
	c, err = Parse("xychart\nx-axis Czas 0 --> 10\ny-axis 1.5 --> 2.5\nline [1, 2]")
	if err != nil || c.XRange == nil || *c.XRange != [2]float64{0, 10} || c.XTitle != "Czas" || c.YTitle != "" || *c.YRange != [2]float64{1.5, 2.5} {
		t.Errorf("numeric axes: %+v %v", c, err)
	}
}

// REQ: XY-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                                       {CodeHeader, 0},
		"xycharts\nbar [1]":                                      {CodeHeader, 1},
		"xychart-beta horizontal\nbar [1]":                       {CodeUnsupported, 1},
		"xychart-beta\ncoś innego":                               {CodeStatement, 2},
		"xychart-beta\nbar 1, 2":                                 {CodeData, 2},
		"xychart-beta\nbar [1, dwa]":                             {CodeData, 2},
		"xychart-beta\ny-axis 10 --> 5\nbar [1]":                 {CodeStatement, 2},
		"xychart-beta\ny-axis [a, b]\nbar [1]":                   {CodeStatement, 2},
		"xychart-beta\nx-axis [a, , c]\nbar [1]":                 {CodeStatement, 2},
		"xychart-beta\nx-axis [a, b]\nbar [1]":                   {CodeData, 3},
		"xychart-beta\nbar [1, 2]\nline [1]":                     {CodeData, 3},
		"xychart-beta\ntitle Sam tytuł":                          {CodeEmpty, 0},
		"xychart-beta\nbar [" + strings.Repeat("1,", 200) + "1]": {CodeData, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: XY-003
func TestTicks(t *testing.T) {
	for _, c := range []struct {
		lo, hi float64
		want   string
	}{
		{0, 40, "0 10 20 30 40"},
		{-5, 10, "-5 0 5 10"},
		{0, 1, "0.0 0.2 0.4 0.6 0.8 1.0"},
		{4000, 11000, "4000 6000 8000 10000"},
		{0.01, 0.035, "0.010 0.015 0.020 0.025 0.030 0.035"},
	} {
		ticks, step := Ticks(c.lo, c.hi)
		var got []string
		for _, v := range ticks {
			got = append(got, Number(v, step))
		}
		if strings.Join(got, " ") != c.want {
			t.Errorf("ticks %v–%v: got %q, want %q", c.lo, c.hi, strings.Join(got, " "), c.want)
		}
	}
}

// REQ: XY-003 XY-004
// Słupki stoją na osi zera i sięgają wartości, serie słupków obok siebie, linie różnią się
// kreskowaniem, legenda ma nazwy serii, a każda kategoria ma podpis.
func TestLayout(t *testing.T) {
	c, err := Parse("xychart-beta\ntitle T\nx-axis \"Oś X\" [a, b, c]\ny-axis \"Oś Y\"\nbar \"Pierwsza\" [10, 20, -10]\nbar \"Druga\" [5, 5, 5]\nline \"L1\" [1, 2, 3]\nline \"L2\" [3, 2, 1]")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []Options{{Measure: font.ApproxTextWidth, FontSize: 16}, {Measure: font.ApproxTextWidth, FontSize: 16, Colors: true, Shadows: true}} {
		sc := Layout(c, o)
		var bars, lines []scene.Item
		var zero float64
		texts := map[string]int{}
		for _, it := range sc.Items {
			switch it.Role {
			case "chart-bar":
				bars = append(bars, it)
			case "chart-line":
				lines = append(lines, it)
			case "chart-axis":
				if it.Y == it.Y2 {
					zero = it.Y
				}
			}
			if it.Kind == scene.Text {
				texts[it.Text]++
			}
		}
		if len(bars) != 6 || len(lines) != 2 {
			t.Fatalf("%d bars and %d lines", len(bars), len(lines))
		}
		// Słupki pierwszej serii: 10, 20, -10; drugiej: 5, 5, 5.
		near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
		if b := bars[0]; !near(b.Y+b.H, zero) || !near(bars[1].H, 2*b.H) {
			t.Errorf("bars should stand on the zero axis with heights in proportion: %+v %+v", bars[0].Bounds(), bars[1].Bounds())
		}
		if b := bars[2]; !near(b.Y, zero) || !near(b.H, bars[0].H) {
			t.Errorf("a negative bar should hang below the zero axis: %+v", b.Bounds())
		}
		if !near(bars[3].X, bars[0].X+bars[0].W) || bars[3].Fill == bars[0].Fill {
			t.Errorf("the second bar series should stand beside the first in another tone")
		}
		if lines[0].Dash == lines[1].Dash {
			t.Errorf("line series should differ in dash pattern")
		}
		if p := lines[0].Points; len(p) != 3 || p[0].Y <= p[2].Y || p[0].X >= p[1].X {
			t.Errorf("line points: %+v", p)
		}
		for _, want := range []string{"T", "Oś X", "Oś Y", "a", "b", "c", "Pierwsza", "Druga", "L1", "L2", "0", "20", "-10"} {
			if texts[want] != 1 {
				t.Errorf("text %q drawn %d times, want once", want, texts[want])
			}
		}
	}
}

// REQ: XY-003
// Gdy etykiety kategorii się nie mieszczą, zostaje co któraś; pierwsza zawsze.
func TestCrowdedCategories(t *testing.T) {
	src := "xychart-beta\nx-axis ["
	values := "line ["
	for i := 0; i < 60; i++ {
		if i > 0 {
			src, values = src+", ", values+", "
		}
		src += "tydzień " + string(rune('A'+i%26)) + string(rune('a'+i/26))
		values += "1"
	}
	c, err := Parse(src + "]\n" + values + "]")
	if err != nil {
		t.Fatal(err)
	}
	shown := 0
	var prev float64
	for _, it := range Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16}).Items {
		if it.Role == "axis-label" && strings.HasPrefix(it.Text, "tydzień") {
			if shown > 0 && it.X-prev < it.W {
				t.Errorf("category labels overlap at %.0f", it.X)
			}
			shown, prev = shown+1, it.X
		}
	}
	if shown == 0 || shown >= 60 {
		t.Errorf("%d category labels shown out of 60", shown)
	}
}
