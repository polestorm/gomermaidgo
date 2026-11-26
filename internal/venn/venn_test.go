package venn

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: VENN-001 VENN-002 VENN-003
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\nvenn-beta\n  title \"Zespoły\"\n  accTitle: Opis\n  set A[\"Alfa\"]:20\n    text A1[\"React\"]\n  set \"B b\"\n  union A,\"B b\"[\"AB\"]:3\n    text AB1\n  style A fill:#ff6b6b,fill-opacity:0.4\n  style \"B b\",A color:#333\n  style A1 color:red")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Zespoły" || d.AccTitle != "Opis" || len(d.Sets) != 2 || len(d.Unions) != 1 {
		t.Fatalf("diagram: %+v", d)
	}
	a, b, u := d.Sets[0], d.Sets[1], d.Unions[0]
	if a.Label != "Alfa" || a.Size != 20 || !a.HasSize || a.Paint.Fill != "#ff6b6b" || a.Opacity != 0.4 || len(a.Texts) != 1 || a.Texts[0].Paint.Color != "red" {
		t.Errorf("set A: %+v", a)
	}
	if b.IDs[0] != "B b" || b.HasSize {
		t.Errorf("set B: %+v", b)
	}
	if u.Label != "AB" || u.Size != 3 || u.Paint.Color != "#333" || len(u.Texts) != 1 || u.Texts[0].Label != "AB1" {
		t.Errorf("union: %+v", u)
	}
}

// REQ: VENN-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                         {CodeHeader, 0},
		"venn-diagram\nset A":                      {CodeHeader, 1},
		"venn-beta\n%% nic":                        {CodeEmpty, 0},
		"venn-beta\ncircle A":                      {CodeStatement, 2},
		"venn-beta\nset A\nset A":                  {CodeStatement, 3},
		"venn-beta\nset A\nunion A":                {CodeStatement, 3},
		"venn-beta\nset A\nunion A,B":              {CodeStatement, 3},
		"venn-beta\ntext X":                        {CodeStatement, 2},
		"venn-beta\nset A:zero":                    {CodeStatement, 2},
		"venn-beta\nset A\nstyle Z fill:red":       {CodeStatement, 3},
		"venn-beta\nset A\nstyle A fill-opacity:2": {CodeStatement, 3},
		"venn-beta\nset A\nset B\nset C\nset D":    {CodeStatement, 5},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.30q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

func circles(sc *scene.Scene) []scene.Item {
	var out []scene.Item
	for _, it := range sc.Items {
		if it.Kind == scene.Circle && it.Role == "venn-set" {
			out = append(out, it)
		}
	}
	return out
}

// REQ: VENN-004
func TestLayoutSizesAndOverlap(t *testing.T) {
	d, err := Parse("venn-beta\nset A:20\nset B:5\nunion A,B:2")
	if err != nil {
		t.Fatal(err)
	}
	cs := circles(Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16}))
	if len(cs) != 2 {
		t.Fatalf("%d circles", len(cs))
	}
	// Pole koła proporcjonalne do wielkości: stosunek promieni to pierwiastek ze stosunku wielkości.
	if r := cs[0].R / cs[1].R; math.Abs(r-2) > 0.05 {
		t.Errorf("radius ratio %.3f, want 2", r)
	}
	// Pole soczewki odpowiada wielkości części wspólnej (2 z 20 → jedna dziesiąta koła A).
	dist := math.Hypot(cs[1].X-cs[0].X, cs[1].Y-cs[0].Y)
	want := math.Pi * cs[0].R * cs[0].R * 2 / 20
	if got := lensArea(cs[0].R, cs[1].R, dist); math.Abs(got-want)/want > 0.01 {
		t.Errorf("lens area %.0f, want %.0f", got, want)
	}
}

// REQ: VENN-004
func TestLayoutThreeSetsAndSeparate(t *testing.T) {
	d, err := Parse("venn-beta\nset A\nset B\nset C\nunion A,B,C[\"Wszystkie\"]")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	cs := circles(sc)
	for i := range cs {
		for j := i + 1; j < len(cs); j++ {
			if math.Hypot(cs[i].X-cs[j].X, cs[i].Y-cs[j].Y) >= cs[i].R+cs[j].R {
				t.Errorf("circles %d and %d do not overlap", i, j)
			}
		}
	}
	// Podpis części wspólnej trzech zbiorów stoi we wszystkich trzech kołach.
	for _, it := range sc.Items {
		if it.Text == "Wszystkie" {
			for _, c := range cs {
				if math.Hypot(it.X-c.X, it.Y-c.Y) > c.R {
					t.Errorf("label outside circle at %v,%v", c.X, c.Y)
				}
			}
		}
	}
	// Zbiór bez części wspólnej stoi osobno.
	d, _ = Parse("venn-beta\nset A\nset B\nset C\nunion A,B")
	cs = circles(Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16}))
	if math.Hypot(cs[2].X-cs[0].X, cs[2].Y-cs[0].Y) < cs[0].R+cs[2].R || math.Hypot(cs[2].X-cs[1].X, cs[2].Y-cs[1].Y) < cs[1].R+cs[2].R {
		t.Error("separate set overlaps the others")
	}
	if strings.Contains(string(Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16}).SVG()), "NaN") {
		t.Error("NaN in SVG")
	}
}

// REQ: VENN-004
func TestLabelsFitTheirRegions(t *testing.T) {
	d, err := Parse(`venn-beta
set W["Znajomość normy"]
set P["Doświadczenie w branży"]
set K["Komunikacja"]
union W,P["Ocena zgodności"]
union P,K["Rozmowa z kierownikiem"]
union W,K["Szkolenie"]
union W,P,K["Audytor wiodący"]`)
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	cs := circles(sc)
	// Obszar każdego podpisu: w których kołach (W, P, K) leży.
	want := map[string][3]bool{
		"Znajomość": {true, false, false}, "normy": {true, false, false},
		"Komunikacja": {false, false, true},
		"Ocena":       {true, true, false}, "zgodności": {true, true, false},
		"Rozmowa": {false, true, true}, "kierownikiem": {false, true, true},
		"Szkolenie": {true, false, true},
		"Audytor":   {true, true, true}, "wiodący": {true, true, true},
	}
	seen := 0
	for _, it := range sc.Items {
		if it.Kind != scene.Text || it.Role == "venn-title" {
			continue
		}
		for word, in := range want {
			if !strings.Contains(it.Text, word) {
				continue
			}
			seen++
			fs := it.FontSize
			w := font.ApproxTextWidth(it.Text, fs) * 1.06
			for _, p := range [][2]float64{{it.X - w/2, it.Y - 0.8*fs}, {it.X + w/2, it.Y - 0.8*fs}, {it.X - w/2, it.Y + 0.2*fs}, {it.X + w/2, it.Y + 0.2*fs}} {
				for k, c := range cs {
					if inside := math.Hypot(p[0]-c.X, p[1]-c.Y) < c.R; inside != in[k] {
						t.Errorf("%q: corner %v in circle %d = %v", it.Text, p, k, inside)
					}
				}
			}
		}
	}
	if seen < len(want) {
		t.Errorf("found %d of %d labels", seen, len(want))
	}
}

// REQ: VENN-004
func TestLeaderForTightOverlap(t *testing.T) {
	d, err := Parse("venn-beta\nset A:30\nset B:30\nunion A,B[\"Wspólne przeglądy wymagań i testów akceptacyjnych\"]:1")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	cs := circles(sc)
	leaders := 0
	for _, it := range sc.Items {
		if it.Role == "venn-leader" {
			leaders++
		}
		if it.Kind == scene.Text && strings.Contains(it.Text, "akceptacyjnych") {
			for _, c := range cs {
				if math.Hypot(it.X-c.X, it.Y-c.Y) < c.R {
					t.Errorf("leader label inside a circle")
				}
			}
		}
	}
	if leaders != 2 {
		t.Errorf("leader items: %d, want line and dot", leaders)
	}
}
