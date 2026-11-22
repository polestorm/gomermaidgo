package sankey

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: SNK-001
func TestParse(t *testing.T) {
	c, err := Parse("%% komentarz\nsankey-beta\n\nA,B,10\n\"Z, przecinkiem\",B,2.5\nB , \"Cytat \"\"w\"\" nazwie\" , 12.5")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.Nodes, "|") != "A|B|Z, przecinkiem|Cytat \"w\" nazwie" || len(c.Links) != 3 {
		t.Fatalf("chart: %+v", c)
	}
	if l := c.Links[2]; l.From != "B" || l.To != "Cytat \"w\" nazwie" || l.Value != 12.5 || l.Line != 6 {
		t.Errorf("link: %+v", l)
	}
}

// REQ: SNK-004
func TestParseErrors(t *testing.T) {
	many := "sankey-beta\n"
	for i := 0; i < 501; i++ {
		many += "A,B,1\n"
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                 {CodeHeader, 0},
		"sankeys\nA,B,1":                   {CodeHeader, 1},
		"sankey-beta\nA,B":                 {CodeStatement, 2},
		"sankey-beta\nA,B,1,2":             {CodeStatement, 2},
		"sankey-beta\n,B,1":                {CodeStatement, 2},
		"sankey-beta\n\"A,B,1":             {CodeStatement, 2},
		"sankey-beta\nA,B,dużo":            {CodeValue, 2},
		"sankey-beta\nA,B,0":               {CodeValue, 2},
		"sankey-beta\nA,A,1":               {CodeCycle, 2},
		"sankey-beta\nA,B,1\nB,C,1\nC,A,1": {CodeCycle, 2},
		"sankey-beta\n%% nic":              {CodeEmpty, 0},
		many:                               {CodeStatement, 502},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || (want.code != CodeCycle && e.Line != want.line) {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: SNK-002 SNK-003
func TestLayout(t *testing.T) {
	c, err := Parse("sankey-beta\nA,M,30\nB,M,10\nM,X,25\nM,Y,15\nB,Koniec,5")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	nodes := map[string]scene.Item{}
	var order []scene.Item
	var links []scene.Item
	texts := map[string]int{}
	for _, it := range sc.Items {
		switch it.Role {
		case "sankey-node":
			order = append(order, it)
		case "sankey-link":
			links = append(links, it)
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	for i, n := range c.Nodes {
		nodes[n] = order[i]
	}
	if len(order) != 6 || len(links) != 5 {
		t.Fatalf("%d nodes and %d links", len(order), len(links))
	}
	// Kolumny: A i B, potem M, potem X, Y i Koniec (węzeł bez odpływów stoi w ostatniej).
	if nodes["A"].X != nodes["B"].X || nodes["M"].X <= nodes["A"].X || nodes["X"].X <= nodes["M"].X || nodes["Y"].X != nodes["X"].X || nodes["Koniec"].X != nodes["X"].X {
		t.Errorf("columns: A %.0f B %.0f M %.0f X %.0f Y %.0f Koniec %.0f", nodes["A"].X, nodes["B"].X, nodes["M"].X, nodes["X"].X, nodes["Y"].X, nodes["Koniec"].X)
	}
	scale := nodes["A"].H / 30
	for name, value := range map[string]float64{"B": 15, "M": 40, "X": 25, "Y": 15, "Koniec": 5} {
		if math.Abs(nodes[name].H-value*scale) > 0.01 {
			t.Errorf("node %s is %.1f px high, want %.1f for value %v", name, nodes[name].H, value*scale, value)
		}
	}
	for _, column := range [][]string{{"A", "B"}, {"X", "Y", "Koniec"}} {
		for i := 1; i < len(column); i++ {
			p, q := nodes[column[i-1]], nodes[column[i]]
			if math.Min(p.Y+p.H, q.Y+q.H) > math.Max(p.Y, q.Y)+0.01 {
				t.Errorf("nodes %s and %s overlap", column[i-1], column[i])
			}
		}
	}
	// Pierwsza wstęga: z A do M, o szerokości 30 w skali, od prawej krawędzi A do lewej krawędzi M.
	s := links[0].Segments
	if s[0].To.X != nodes["A"].X+nodes["A"].W || s[1].To.X != nodes["M"].X || math.Abs(s[2].To.Y-s[1].To.Y-30*scale) > 0.01 {
		t.Errorf("first band: %+v", s)
	}
	for _, want := range []string{"A", "M", "Koniec", "30", "40", "5"} {
		if texts[want] == 0 {
			t.Errorf("text %q is missing", want)
		}
	}
}

// bands zbiera wstęgi rysunku: punkty górnej krawędzi (x, y) i grubość.
type band struct {
	top       []scene.Point
	thickness float64
}

func bandsAndNodes(sc *scene.Scene) (bands []band, nodes []scene.Item) {
	for _, it := range sc.Items {
		switch it.Role {
		case "sankey-node":
			nodes = append(nodes, it)
		case "sankey-link":
			// Górna krawędź to pierwsza połowa ścieżki: do pierwszego odcinka prowadzącego w dół.
			b := band{}
			for i, sg := range it.Segments {
				if i > 0 && sg.Cmd == 'L' && sg.To.X == it.Segments[i-1].To.X && sg.To.Y > it.Segments[i-1].To.Y {
					b.thickness = sg.To.Y - it.Segments[i-1].To.Y
					break
				}
				b.top = append(b.top, sg.To)
			}
			bands = append(bands, b)
		}
	}
	return bands, nodes
}

// REQ: SNK-005
// Wstęgi się nie krzyżują, gdy da się tego uniknąć, a wstęga biegnąca przez kilka kolumn nie
// przechodzi po węzłach kolumn pośrednich.
func TestBandsDoNotCrossOrCoverNodes(t *testing.T) {
	for _, src := range []string{
		"sankey-beta\nZamowienia,Kontrola,120\nKontrola,Magazyn,95\nKontrola,Zwrot,25\nMagazyn,Produkcja,80\nMagazyn,Rezerwa,15\nProdukcja,Wyrob gotowy,74\nProdukcja,Odpad,6",
		"sankey-beta\nA,M,30\nB,M,10\nM,X,25\nM,Y,15\nB,Koniec,5",
		"sankey-beta\nA,B,10\nA,C,20\nB,D,4\nB,E,6\nC,E,20\nA,E,7",
	} {
		c, err := Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		sc := Layout(c, Options{Measure: font.ApproxTextWidth, FontSize: 16})
		bands, nodes := bandsAndNodes(sc)
		if len(bands) != len(c.Links) {
			t.Fatalf("%d bands for %d links", len(bands), len(c.Links))
		}
		// Każda para wstęg: na wspólnych pionach (krawędziach kolumn) ich kolejność jest ta sama
		// i przedziały się nie nakładają.
		for i := range bands {
			for j := i + 1; j < len(bands); j++ {
				sign := 0.0
				for _, p := range bands[i].top {
					for _, q := range bands[j].top {
						if p.X != q.X {
							continue
						}
						if math.Min(p.Y+bands[i].thickness, q.Y+bands[j].thickness) > math.Max(p.Y, q.Y)+0.01 {
							t.Errorf("bands %d and %d overlap at x=%.0f", i, j, p.X)
						}
						d := p.Y - q.Y
						if sign != 0 && d*sign < 0 {
							t.Errorf("bands %d and %d cross", i, j)
						}
						if d != 0 {
							sign = d
						}
					}
				}
			}
		}
		// Wstęga na pionie kolumny nie wchodzi na węzeł, który nie jest jej końcem.
		for i, b := range bands {
			for k, p := range b.top {
				if k == 0 || k == len(b.top)-1 {
					continue
				}
				for _, n := range nodes {
					if (p.X == n.X || p.X == n.X+n.W) && math.Min(p.Y+b.thickness, n.Y+n.H) > math.Max(p.Y, n.Y)+0.01 {
						t.Errorf("band %d runs over a node at x=%.0f", i, n.X)
					}
				}
			}
		}
	}
}

// REQ: SNK-006
// „%% layout: wrap” łamie nazwy ostatniej kolumny na słowach i zostawia wykresowi więcej miejsca.
func TestWrapLabels(t *testing.T) {
	src := "sankey-beta\nWejście,Bardzo długa nazwa końcowego węzła,10\nWejście,Krótki,5"
	plain, _ := Parse(src)
	wrapped, err := Parse("%% layout: wrap\n" + src)
	if err != nil || !wrapped.Wrap || plain.Wrap {
		t.Fatalf("directive: %v", err)
	}
	o := Options{Measure: font.ApproxTextWidth, FontSize: 16}
	last := func(sc *scene.Scene) (x float64, lines int) {
		for _, it := range sc.Items {
			if it.Role == "sankey-node" {
				x = math.Max(x, it.X)
			}
			if it.Role == "sankey-label" && it.Text != "Wejście" && it.Text != "Krótki" {
				lines++
			}
		}
		return x, lines
	}
	x0, n0 := last(Layout(plain, o))
	x1, n1 := last(Layout(wrapped, o))
	if n0 != 1 || n1 < 3 || x1 <= x0+40 {
		t.Errorf("wrapping: %d line at x=%.0f, wrapped %d lines at x=%.0f", n0, x0, n1, x1)
	}
}
