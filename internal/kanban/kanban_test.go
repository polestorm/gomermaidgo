package kanban

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: KBN-001 KBN-002
func TestParse(t *testing.T) {
	b, err := Parse("%% komentarz\nkanban\n  accTitle: Tablica\n  Pierwsza\n    [Karta bez id]\n    k2[Druga<br/>w dwóch liniach]@{ ticket: T-1, assigned: 'Jan K.', priority: \"High\", inne: x }\n  kol2[Druga kolumna]\n\t\tgłęboko wcięta karta\n  Trzecia")
	if err != nil {
		t.Fatal(err)
	}
	if b.AccTitle != "Tablica" || len(b.Columns) != 3 {
		t.Fatalf("board: %+v", b)
	}
	first := b.Columns[0]
	if first.Title[0] != "Pierwsza" || len(first.Cards) != 2 || first.Cards[0].Text[0] != "Karta bez id" {
		t.Errorf("first column: %+v", first)
	}
	if c := first.Cards[1]; strings.Join(c.Text, "|") != "Druga|w dwóch liniach" || c.Ticket != "T-1" || c.Assigned != "Jan K." || c.Priority != "High" {
		t.Errorf("card with details: %+v", c)
	}
	if c := b.Columns[1]; c.Title[0] != "Druga kolumna" || len(c.Cards) != 1 || len(b.Columns[2].Cards) != 0 {
		t.Errorf("second and third column: %+v %+v", c, b.Columns[2])
	}
}

// REQ: KBN-004
func TestParseErrors(t *testing.T) {
	many := "kanban\n"
	for i := 0; i < 13; i++ {
		many += "kolumna\n"
	}
	cards := "kanban\nkolumna\n" + strings.Repeat("  karta\n", 201)
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":               {CodeHeader, 0},
		"kanbans\nA":     {CodeHeader, 1},
		"kanban\n  k[ ]": {CodeStatement, 2},
		"kanban\n%% nic": {CodeEmpty, 0},
		many:             {CodeStatement, 14},
		cards:            {CodeStatement, 203},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.30q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: KBN-003
func TestLayout(t *testing.T) {
	b, err := Parse("kanban\n  A\n    [pierwsza karta z długim tekstem który się złamie]@{ priority: 'Very High', ticket: X-1 }\n    [druga]@{ priority: 'Low' }\n    [trzecia]\n  B\n    [czwarta]")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(b, Options{Measure: font.ApproxTextWidth, FontSize: 16, Shadows: true, Colors: true})
	var columns, cards, bars []scene.Item
	texts := map[string]int{}
	for _, it := range sc.Items {
		switch it.Role {
		case "kanban-column":
			columns = append(columns, it)
		case "kanban-card":
			cards = append(cards, it)
		case "kanban-priority":
			bars = append(bars, it)
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	if len(columns) != 2 || len(cards) != 4 || len(bars) != 2 {
		t.Fatalf("%d columns, %d cards, %d priority bars", len(columns), len(cards), len(bars))
	}
	if columns[0].W != columns[1].W || columns[0].H != columns[1].H || columns[1].X < columns[0].X+columns[0].W {
		t.Errorf("columns should stand side by side with a common size: %+v %+v", columns[0].Bounds(), columns[1].Bounds())
	}
	for i, c := range cards[:3] {
		if c.X < columns[0].X || c.X+c.W > columns[0].X+columns[0].W || c.Y+c.H > columns[0].Y+columns[0].H {
			t.Errorf("card %d lies outside its column", i)
		}
		if i > 0 && c.Y < cards[i-1].Y+cards[i-1].H {
			t.Errorf("card %d overlaps the previous one", i)
		}
	}
	if bars[0].W <= bars[1].W {
		t.Errorf("a more urgent card should have a thicker bar: %v and %v", bars[0].W, bars[1].W)
	}
	for _, want := range []string{"A", "B", "druga", "czwarta", "X-1 · Very High", "Low"} {
		if texts[want] != 1 {
			t.Errorf("text %q drawn %d times, want once", want, texts[want])
		}
	}
}
