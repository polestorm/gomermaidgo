package cynefin

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: CYN-001 CYN-002 CYN-003
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\ncynefin-beta\n  title \"Mapa\"\n  accTitle: Opis\n  complex\n    \"A\"\n  clear\n    \"B\"\n  complex\n    \"C\"\n  complex --> complicated : \"Wzorzec\"\n  clear --> chaotic\n  chaotic --> chaotic : \"x\"")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Mapa" || d.AccTitle != "Opis" {
		t.Errorf("title %q, accTitle %q", d.Title, d.AccTitle)
	}
	if strings.Join(d.Items[Complex], "|") != "A|C" || strings.Join(d.Items[Clear], "|") != "B" {
		t.Errorf("items: %v", d.Items)
	}
	if len(d.Transitions) != 2 || d.Transitions[0] != (Transition{Complex, Complicated, "Wzorzec", 11}) || d.Transitions[1].Label != "" {
		t.Errorf("transitions: %+v", d.Transitions)
	}
}

// REQ: CYN-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                             {CodeHeader, 0},
		"cynefin-alpha":                {CodeHeader, 1},
		"cynefin-beta\n\"A\"":          {CodeStatement, 2},
		"cynefin-beta\nclear\n\"A":     {CodeStatement, 3},
		"cynefin-beta\nclear\nA":       {CodeStatement, 3},
		"cynefin-beta\nclear --> calm": {CodeStatement, 2},
		"cynefin-beta\nobvious\n\"A\"": {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.30q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

func boxOf(it scene.Item) scene.Box { return it.Bounds() }

func overlaps(a, b scene.Box) bool {
	return a.X0 < b.X1 && b.X0 < a.X1 && a.Y0 < b.Y1 && b.Y0 < a.Y1
}

// REQ: CYN-004
func TestLayout(t *testing.T) {
	d, err := Parse("cynefin-beta\ncomplex\n\"Badanie\"\ncomplicated\n\"Analiza\"\nchaotic\n\"Gaszenie pożaru\"\nclear\n\"Procedura\"\nconfusion\n\"a\"\n\"b\"\n\"c\"\n\"d\"\n\"e\"\ncomplex --> complicated : \"Wzorzec\"\nclear --> chaotic : \"Klif\"\ncomplex --> clear")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var frame, ellipse scene.Item
	labels := map[string]scene.Item{}
	var items []scene.Item
	cliff, arrows := 0, 0
	for _, it := range sc.Items {
		switch it.Role {
		case "cynefin-frame":
			frame = it
		case "cynefin-confusion":
			ellipse = it
		case "cynefin-domain-label", "cynefin-item-label":
			labels[it.Text] = it
		case "cynefin-item":
			items = append(items, it)
		case "cynefin-cliff":
			cliff++
		case "cynefin-transition":
			if it.Kind == scene.Polygon {
				arrows++
			}
		}
	}
	if cliff != 1 || arrows != 3 {
		t.Errorf("cliff %d, arrowheads %d", cliff, arrows)
	}
	// Stałe położenie domen: ćwiartka według znaku współrzędnych środka planszy.
	for name, sx, sy := "Complex", -1.0, -1.0; ; {
		l := labels[name]
		if math.Copysign(1, l.X) != sx || math.Copysign(1, l.Y) != sy {
			t.Errorf("%s at %v,%v", name, l.X, l.Y)
		}
		switch name {
		case "Complex":
			name, sx, sy = "Complicated", 1, -1
			continue
		case "Complicated":
			name, sx, sy = "Chaotic", -1, 1
			continue
		case "Chaotic":
			name, sx, sy = "Clear", 1, 1
			continue
		}
		break
	}
	// Zagubienie: trzy elementy i „+2 more”.
	if _, ok := labels["+2 more"]; !ok || labels["d"].Text != "" {
		t.Errorf("confusion items: %v", labels)
	}
	// Elementy domen leżą w ramce i nie wchodzą w elipsę ani w siebie nawzajem.
	fb, eb := boxOf(frame), boxOf(ellipse)
	for i, it := range items {
		b := boxOf(it)
		if b.X0 < fb.X0 || b.X1 > fb.X1 || b.Y0 < fb.Y0 || b.Y1 > fb.Y1 {
			t.Errorf("item %d outside the frame", i)
		}
		inEllipse := b.X0 >= eb.X0 && b.X1 <= eb.X1 && b.Y0 >= eb.Y0 && b.Y1 <= eb.Y1
		if !inEllipse && overlaps(b, eb) {
			t.Errorf("item %d runs into the confusion ellipse", i)
		}
		for j := i + 1; j < len(items); j++ {
			if overlaps(b, boxOf(items[j])) {
				t.Errorf("items %d and %d overlap", i, j)
			}
		}
	}
}
