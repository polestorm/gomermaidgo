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
