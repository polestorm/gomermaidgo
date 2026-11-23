package ishikawa

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: ISH-001 ISH-002
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\nishikawa\n  accTitle: Przyczyny\n  Problem<br>w dwóch liniach\n  Materiał\n    Partia\n      Wilgotność\n        Brak suszenia\n    Dostawca\n  Metoda\n\tInstrukcja")
	if err != nil {
		t.Fatal(err)
	}
	if d.AccTitle != "Przyczyny" || strings.Join(d.Effect, "|") != "Problem|w dwóch liniach" || len(d.Categories) != 2 {
		t.Fatalf("diagram: %+v", d)
	}
	m := d.Categories[0]
	if m.Text[0] != "Materiał" || len(m.Children) != 2 || m.Children[0].Children[0].Children[0].Text[0] != "Brak suszenia" {
		t.Errorf("first category: %+v", m)
	}
	if c := d.Categories[1]; c.Text[0] != "Metoda" || len(c.Children) != 1 || c.Children[0].Text[0] != "Instrukcja" {
		t.Errorf("second category: %+v", c)
	}
}

// REQ: ISH-004
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                         {CodeHeader, 0},
		"fishbone\nA":              {CodeHeader, 1},
		"ishikawa-beta\n%% nic":    {CodeEmpty, 0},
		"ishikawa-beta\n  Problem": {CodeEmpty, 0},
		"ishikawa-beta\nP\nK\n a\n  b\n   c\n    d": {CodeStatement, 7},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.30q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: ISH-003
func TestLayout(t *testing.T) {
	d, err := Parse("ishikawa-beta\n  Problem\n  A\n    przyczyna pierwsza z długim tekstem, który się złamie\n      szczegół\n    druga\n  B\n    trzecia\n  C\n    czwarta")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var cats, bones, texts []scene.Item
	var effect, spine scene.Item
	for _, it := range sc.Items {
		switch {
		case it.Role == "ishikawa-bone":
			bones = append(bones, it)
		case it.Role == "ishikawa-category":
			cats = append(cats, it)
		case it.Role == "ishikawa-effect" && it.Kind == scene.Rect:
			effect = it
		case it.Role == "ishikawa-spine" && it.Kind == scene.Line:
			spine = it
		}
		if it.Kind == scene.Text {
			texts = append(texts, it)
		}
	}
	if len(cats) != 3 {
		t.Fatalf("%d categories", len(cats))
	}
	// Kategorie na przemian nad osią i pod nią, w kolejności od lewej; problem po prawej, w osi.
	if !(cats[0].Y+cats[0].H < 0 && cats[1].Y > 0 && cats[2].Y+cats[2].H < 0) {
		t.Errorf("categories should alternate above and below the spine: %v %v %v", cats[0].Y, cats[1].Y, cats[2].Y)
	}
	if !(bones[0].X2 < bones[1].X2 && bones[1].X2 < bones[2].X2) || effect.X < bones[2].X2 {
		t.Errorf("categories should go left to right with the problem on the right: %v %v %v, problem %v", bones[0].X2, bones[1].X2, bones[2].X2, effect.X)
	}
	if effect.Y >= 0 || effect.Y+effect.H <= 0 || spine.X2 > effect.X {
		t.Errorf("the spine should end at the problem box centred on it")
	}
	for i := range texts {
		for j := i + 1; j < len(texts); j++ {
			a, b := texts[i].Bounds(), texts[j].Bounds()
			if a.X0 < b.X1 && b.X0 < a.X1 && a.Y0 < b.Y1 && b.Y0 < a.Y1 {
				t.Errorf("texts %q and %q overlap", texts[i].Text, texts[j].Text)
			}
		}
	}
}
