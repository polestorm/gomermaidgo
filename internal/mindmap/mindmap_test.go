package mindmap

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: MND-001 MND-002 MND-004
func TestParse(t *testing.T) {
	g, err := Parse("%% komentarz\nmindmap\n  root((Korzeń))\n    A\n      ::icon(fa fa-book)\n      a1[Kwadrat<br/>w dwóch liniach]\n         głębiej\n      a2(Zaokrąglony)\n    :::klasa\n    B{{Sześciokąt}}\n\tc))Wybuch((\n      d)Chmura(\n    \"`**gruby** i *pochylony*`\"")
	if err != nil {
		t.Fatal(err)
	}
	var edges []string
	for _, e := range g.Edges {
		edges = append(edges, graph.Plain(g.Node(e.From).Label[0])+">"+graph.Plain(g.Node(e.To).Label[0]))
	}
	want := "Korzeń>A A>Kwadrat Kwadrat>głębiej A>Zaokrąglony Korzeń>Sześciokąt Korzeń>Wybuch Wybuch>Chmura Korzeń>gruby i pochylony" // tabulacja liczy się jak cztery spacje
	if got := strings.Join(edges, " "); got != want {
		t.Errorf("tree:\n got %s\nwant %s", got, want)
	}
	shapes := map[string]graph.Shape{}
	for _, n := range g.Nodes {
		shapes[graph.Plain(n.Label[0])] = n.Shape
	}
	for label, shape := range map[string]graph.Shape{"Korzeń": graph.Circle, "A": graph.Round, "Kwadrat": graph.Rect, "Zaokrąglony": graph.Round,
		"Sześciokąt": graph.Hexagon, "Wybuch": graph.Hexagon, "Chmura": graph.Stadium} {
		if shapes[label] != shape {
			t.Errorf("%s: shape %s, want %s", label, shapes[label], shape)
		}
	}
	if n := g.Nodes[2]; strings.Join(n.Label, "|") != "Kwadrat|w dwóch liniach" {
		t.Errorf("line break: %q", n.Label)
	}
	if last := g.Nodes[len(g.Nodes)-1]; !graph.HasMarks(last.Label[0]) {
		t.Errorf("markdown string lost its emphasis: %q", last.Label)
	}
	if g, err := Parse("mindmap\nSam korzeń"); err != nil || g.Nodes[0].Shape != graph.Stadium {
		t.Errorf("root without a shape: %v %v", g, err)
	}
}

// REQ: MND-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                {CodeHeader, 0},
		"mindmaps\n root":                 {CodeHeader, 1},
		"mindmap\n  root\n  drugi korzeń": {CodeRoot, 3},
		"mindmap\n    root\n  płycej":     {CodeRoot, 3},
		"mindmap\n  root\n    a[ ]":       {CodeStatement, 3},
		"mindmap\n%% nic":                 {CodeEmpty, 0},
		"mindmap\n  ::icon(x)":            {CodeEmpty, 0},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

func boxes(sc *scene.Scene) map[string]scene.Box {
	out := map[string]scene.Box{}
	var nodes []scene.Item
	for _, it := range sc.Items {
		if it.Role == "node" {
			nodes = append(nodes, it)
		}
	}
	for _, it := range sc.Items {
		if it.Role != "node-label" {
			continue
		}
		for _, n := range nodes {
			b := n.Bounds()
			if it.X >= b.X0 && it.X <= b.X1 && it.Y >= b.Y0 && it.Y <= b.Y1 {
				out[it.Text] = b
			}
		}
	}
	return out
}

// REQ: MND-003 MND-006
// Układ listowy. Drzewo: potomkowie w jednej kolumnie i w kolejności ze źródła, rodzic pośrodku ich wysokości,
// pola rozłączne.
func TestLayoutIsATree(t *testing.T) {
	g, err := Parse("mindmap\n%% layout: list\n  K\n    A\n      a1\n      a2 dłuższe\n      a3\n    B\n    C\n      c1")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	b := boxes(sc)
	if len(b) != 8 {
		t.Fatalf("%d boxes found, want 8", len(b))
	}
	mid := func(k string) float64 { return (b[k].Y0 + b[k].Y1) / 2 }
	for _, col := range [][]string{{"A", "B", "C"}, {"a1", "a2 dłuższe", "a3", "c1"}} {
		for i, k := range col {
			if b[k].X0 != b[col[0]].X0 {
				t.Errorf("%s does not start in the column of %s", k, col[0])
			}
			if i > 0 && b[k].Y0 < b[col[i-1]].Y1 {
				t.Errorf("%s is not below %s", k, col[i-1])
			}
		}
	}
	if math.Abs(mid("A")-(mid("a1")+mid("a3"))/2) > 0.5 || math.Abs(mid("C")-mid("c1")) > 0.5 || math.Abs(mid("K")-(b["a1"].Y0+b["c1"].Y1)/2) > 0.5 {
		t.Errorf("parents are not centred on their children: %v", b)
	}
	if b["A"].X0 <= b["K"].X1 || b["a1"].X0 <= b["A"].X1 {
		t.Errorf("children are not to the right of their parents")
	}
	var all []scene.Box
	for _, box := range b {
		all = append(all, box)
	}
	for i := range all {
		for j := i + 1; j < len(all); j++ {
			if p, q := all[i], all[j]; p.X0 < q.X1 && q.X0 < p.X1 && p.Y0 < q.Y1 && q.Y0 < p.Y1 {
				t.Errorf("boxes overlap: %+v and %+v", p, q)
			}
		}
	}
	if edges := 0; true {
		for _, it := range sc.Items {
			if it.Role == "edge" {
				edges++
			}
		}
		if edges != 7 {
			t.Errorf("%d links drawn, want 7", edges)
		}
	}
}

// REQ: MND-003 MND-006
// Układ domyślny stawia gałęzie główne po obu stronach korzenia, a liście lewej gałęzi dalej
// w lewo; układ listowy stawia wszystkie po prawej.
func TestLayoutUsesBothSides(t *testing.T) {
	tree := "  Korzeń\n"
	for i := 0; i < 4; i++ {
		tree += fmt.Sprintf("    Gałąź %d\n", i)
		for k := 0; k < 3; k++ {
			tree += fmt.Sprintf("      liść %d.%d\n", i, k)
		}
	}
	g, err := Parse("mindmap\n" + tree)
	if err != nil {
		t.Fatal(err)
	}
	b := boxes(Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16}))
	left, right := 0, 0
	for i := 0; i < 4; i++ {
		if k := fmt.Sprintf("Gałąź %d", i); b[k].X1 < b["Korzeń"].X0 {
			left++
		} else if b[k].X0 > b["Korzeń"].X1 {
			right++
		}
	}
	if left != 2 || right != 2 {
		t.Errorf("%d branches on the left and %d on the right, want 2 and 2", left, right)
	}
	if b["liść 3.0"].X1 >= b["Gałąź 3"].X0 || b["liść 0.0"].X0 <= b["Gałąź 0"].X1 {
		t.Errorf("leaves should stand on the outer side of their branch")
	}
	list, err := Parse("mindmap\n%% layout: list\n" + tree)
	if err != nil || list.Layout != "list" {
		t.Fatalf("list directive: %v", err)
	}
	b = boxes(Layout(list, Options{Measure: font.ApproxTextWidth, FontSize: 16}))
	for i := 0; i < 4; i++ {
		if k := fmt.Sprintf("Gałąź %d", i); b[k].X0 <= b["Korzeń"].X1 {
			t.Errorf("list layout: %s should stand to the right of the root", k)
		}
	}
}
