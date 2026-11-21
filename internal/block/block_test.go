package block

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: BLK-001 BLK-002
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\nblock-beta\n  accTitle: Bloki\n  columns 3\n  a b[\"Z etykietą i spacją\"]:2\n  space:2 c((\"Koło\"))\n  block:grupa:3\n    columns auto\n    d{\"Romb\"} e{{\"Sześć\"}} f[(\"Walec\")] g([\"Pigułka\"]) h(\"Okrągły\")\n    block\n      i<[\"Strzałka\"]>(left):2\n    end\n  end\n  a --> b\n  b -- \"z etykietą\" --> c\n  c -.-> d\n  d --- e\n  classDef k fill:#fde\n  class a,b k\n  style c stroke:#00f")
	if err != nil {
		t.Fatal(err)
	}
	root := d.Root
	if d.AccTitle != "Bloki" || root.Columns != 3 || len(root.Children) != 5 {
		t.Fatalf("root: %+v", root)
	}
	a, b, space, c, group := root.Children[0], root.Children[1], root.Children[2], root.Children[3], root.Children[4]
	if a.ID != "a" || a.Label[0] != "a" || a.Shape != Rect || a.Span != 1 || a.Paint.Fill != "#fde" {
		t.Errorf("plain block: %+v", a)
	}
	if b.Label[0] != "Z etykietą i spacją" || b.Span != 2 || b.Paint.Fill != "#fde" {
		t.Errorf("block with a label and a width: %+v", b)
	}
	if space.Shape != Space || space.Span != 2 || c.Shape != Circle || c.Paint.Stroke != "#00f" {
		t.Errorf("space and circle: %+v %+v", space, c)
	}
	if group.Shape != Group || group.ID != "grupa" || group.Span != 3 || group.Columns != 0 || len(group.Children) != 6 {
		t.Fatalf("group: %+v", group)
	}
	var shapes []string
	for _, ch := range group.Children {
		shapes = append(shapes, ch.Shape)
	}
	if strings.Join(shapes, " ") != "diamond hexagon cylinder stadium round group" {
		t.Errorf("shapes: %v", shapes)
	}
	if arrow := group.Children[5].Children[0]; arrow.Shape != Arrow || arrow.Direction != "left" || arrow.Span != 2 || arrow.Label[0] != "Strzałka" {
		t.Errorf("block arrow: %+v", arrow)
	}
	want := []Edge{{"a", "b", nil, true, false, 14}, {"b", "c", []string{"z etykietą"}, true, false, 15}, {"c", "d", nil, true, true, 16}, {"d", "e", nil, false, false, 17}}
	for i, e := range d.Edges {
		if e.From != want[i].From || e.To != want[i].To || strings.Join(e.Label, "|") != strings.Join(want[i].Label, "|") || e.Arrow != want[i].Arrow || e.Dotted != want[i].Dotted || e.Line != want[i].Line {
			t.Errorf("edge %d: %+v", i, e)
		}
	}
}

// REQ: BLK-004
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                 {CodeHeader, 0},
		"blocks\na":                        {CodeHeader, 1},
		"block-beta\na[\"niezamknięty":     {CodeStatement, 2},
		"block-beta\ncolumns 0":            {CodeStatement, 2},
		"block-beta\ncolumns wiele":        {CodeStatement, 2},
		"block-beta\na:99":                 {CodeStatement, 2},
		"block-beta\na\na":                 {CodeBlock, 3},
		"block-beta\na\nend":               {CodeBlock, 3},
		"block-beta\nblock:g\na":           {CodeBlock, 2},
		"block-beta\na\na --> nieznany":    {CodeEdge, 3},
		"block-beta\na\nstyle b fill:#fff": {CodeStatement, 3},
		"block-beta\na\nclass a nieznana":  {CodeStatement, 3},
		"block-beta\n%% nic":               {CodeEmpty, 0},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: BLK-003
func TestLayout(t *testing.T) {
	d, err := Parse("block-beta\ncolumns 3\na b:2\nspace c d\nblock:g:3\ncolumns 2\ne f\nend\nh:2 i:2\na --> c\nc -- \"etykieta\" --> d")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []Options{{Measure: font.ApproxTextWidth, FontSize: 16}, {Measure: font.ApproxTextWidth, FontSize: 16, Colors: true, Shadows: true}} {
		sc := Layout(d, o)
		boxes := map[string]scene.Box{}
		var blocks []scene.Item
		var group scene.Box
		edges, texts := 0, map[string]int{}
		for _, it := range sc.Items {
			switch it.Role {
			case "block":
				blocks = append(blocks, it)
			case "block-group":
				group = it.Bounds()
			case "edge":
				edges++
			}
			if it.Kind == scene.Text {
				texts[it.Text]++
			}
		}
		for i, name := range []string{"a", "b", "c", "d", "e", "f", "h", "i"} {
			boxes[name] = blocks[i].Bounds()
		}
		near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
		a, b, c, dd := boxes["a"], boxes["b"], boxes["c"], boxes["d"]
		if !near(a.Y0, b.Y0) || !near(b.X1-b.X0, 2*(a.X1-a.X0)+(b.X0-a.X1)) {
			t.Errorf("a block two columns wide should span two columns and the gap: %+v %+v", a, b)
		}
		// Puste miejsce zajmuje pierwszą kolumnę drugiego rzędu: c stoi w drugiej, pod początkiem b.
		if !near(c.X0, b.X0) || c.Y0 < a.Y1 || !near(dd.Y0, c.Y0) || !near(dd.X1, b.X1) {
			t.Errorf("second row: c %+v d %+v", c, dd)
		}
		e, f := boxes["e"], boxes["f"]
		if e.X0 < group.X0 || f.X1 > group.X1 || e.Y0 < group.Y0 || e.Y1 > group.Y1 || !near(group.X0, a.X0) || !near(group.X1, b.X1) {
			t.Errorf("the group should span the grid and hold its blocks: %+v e %+v f %+v", group, e, f)
		}
		// h i i mają po dwie kolumny: nie mieszczą się w jednym rzędzie trzech kolumn.
		if h, i := boxes["h"], boxes["i"]; i.Y0 < h.Y1 || !near(i.X0, h.X0) {
			t.Errorf("a block that does not fit the row should start the next one: %+v %+v", h, i)
		}
		for x := range blocks {
			for y := x + 1; y < len(blocks); y++ {
				p, q := blocks[x].Bounds(), blocks[y].Bounds()
				if p.X0 < q.X1-0.01 && q.X0 < p.X1-0.01 && p.Y0 < q.Y1-0.01 && q.Y0 < p.Y1-0.01 {
					t.Errorf("blocks %d and %d overlap", x, y)
				}
			}
		}
		if edges != 2 || texts["etykieta"] != 1 || texts["a"] != 1 || texts["i"] != 1 {
			t.Errorf("%d edges, texts %v", edges, texts)
		}
	}
	// Brzeg kształtu: połączenie do koła i do rombu kończy się na krzywej, nie na prostokącie.
	x, y := border(Circle, [4]float64{0, 0, 100, 100}, 150, 150)
	if math.Abs(math.Hypot(x-50, y-50)-50) > 0.01 {
		t.Errorf("circle border at (%.1f, %.1f)", x, y)
	}
	if x, y = border(Diamond, [4]float64{0, 0, 100, 100}, 150, 150); math.Abs(x-75) > 0.01 || math.Abs(y-75) > 0.01 {
		t.Errorf("diamond border at (%.1f, %.1f)", x, y)
	}
}
