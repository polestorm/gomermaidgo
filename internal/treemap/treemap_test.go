package treemap

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: TRM-001
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\ntreemap-beta\ntitle Tytuł\naccTitle: Dostępny\nclassDef k fill:#fff\n\"Sekcja\"\n    \"Liść\": 10\n    \"Podsekcja\":::k\n        \"Głęboki\": 2.5\n\"Sam liść\": 4:::k")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Tytuł" || d.AccTitle != "Dostępny" || len(d.Roots) != 2 {
		t.Fatalf("diagram: %+v", d)
	}
	s := d.Roots[0]
	if s.Leaf || s.Value != 12.5 || len(s.Children) != 2 || !s.Children[0].Leaf || s.Children[0].Text != "10" {
		t.Errorf("section: %+v", s)
	}
	if sub := s.Children[1]; sub.Leaf || sub.Value != 2.5 || sub.Children[0].Label != "Głęboki" {
		t.Errorf("nested section: %+v", sub)
	}
	if l := d.Roots[1]; !l.Leaf || l.Value != 4 || l.Label != "Sam liść" {
		t.Errorf("top-level leaf: %+v", l)
	}
}

// REQ: TRM-004
func TestParseErrors(t *testing.T) {
	many := "treemap-beta\n" + strings.Repeat("\"x\": 1\n", 301)
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                   {CodeHeader, 0},
		"treemaps\n\"a\": 1":                 {CodeHeader, 1},
		"treemap-beta\nbez cudzysłowów: 1":   {CodeStatement, 2},
		"treemap-beta\n\" \": 1":             {CodeStatement, 2},
		"treemap-beta\n\"a\": dużo":          {CodeValue, 2},
		"treemap-beta\n\"a\": -1":            {CodeValue, 2},
		"treemap-beta\n\"a\": 1\n  \"b\": 2": {CodeStatement, 3},
		"treemap-beta\n\"a\"\n  \"b\": 0":    {CodeEmpty, 0},
		"treemap-beta\ntitle Sam tytuł":      {CodeEmpty, 0},
		many:                                 {CodeStatement, 302},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: TRM-002 TRM-003
func TestLayout(t *testing.T) {
	d, err := Parse("treemap-beta\n\"Duża sekcja\"\n    \"A\": 60\n    \"B\": 30\n\"C\": 9\n\"Bardzo długa nazwa małego liścia\": 1")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	var boxes []scene.Item
	texts := map[string]int{}
	note := ""
	for _, it := range sc.Items {
		if it.Role == "treemap-node" {
			boxes = append(boxes, it)
		}
		if it.Role == "treemap-note" {
			note += it.Text + "\n"
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	// Kolejność rysowania: sekcja, jej liście A i B, potem C i mały liść.
	if len(boxes) != 5 {
		t.Fatalf("%d boxes", len(boxes))
	}
	area := func(i int) float64 { return boxes[i].W * boxes[i].H }
	section, a, b, c, tiny := 0, 1, 2, 3, 4
	total := area(section) + area(c) + area(tiny)
	for i, share := range map[int]float64{section: 0.9, c: 0.09, tiny: 0.01} {
		if math.Abs(area(i)/total-share) > 0.001 {
			t.Errorf("box %d takes %.3f of the map, want %.2f", i, area(i)/total, share)
		}
	}
	if math.Abs(area(a)/area(b)-2) > 0.01 {
		t.Errorf("A should be twice as large as B: %.0f and %.0f", area(a), area(b))
	}
	for _, i := range []int{a, b} {
		p, q := boxes[i].Bounds(), boxes[section].Bounds()
		if p.X0 < q.X0 || p.Y0 < q.Y0 || p.X1 > q.X1+0.01 || p.Y1 > q.Y1+0.01 {
			t.Errorf("leaf %d lies outside its section", i)
		}
	}
	if p, q := boxes[a].Bounds(), boxes[b].Bounds(); p.X0 < q.X1-0.01 && q.X0 < p.X1-0.01 && p.Y0 < q.Y1-0.01 && q.Y0 < p.Y1-0.01 {
		t.Error("leaves of one section overlap")
	}
	for _, want := range []string{"Duża sekcja", "A", "B", "C", "60", "30"} {
		if texts[want] != 1 {
			t.Errorf("text %q drawn %d times, want once", want, texts[want])
		}
	}
	if !strings.Contains(note, "Bardzo długa nazwa małego liścia: 1") || texts["1"] != 1 {
		t.Errorf("a leaf too small for its name should be numbered and listed below the map: %q", note)
	}
}
