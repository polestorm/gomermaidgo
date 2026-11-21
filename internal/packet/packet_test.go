package packet

import (
	"math"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: PKT-001 PKT-002
func TestParse(t *testing.T) {
	d, err := Parse("%% komentarz\npacket-beta\ntitle \"Nagłówek\"\naccDescr: Opis\n0-3: \"Wersja\"\n4: \"Flaga\"\n+11: \"Długość\"\n16 - 47: \"Dane\"")
	if err != nil {
		t.Fatal(err)
	}
	want := []Field{{0, 3, "Wersja", 5}, {4, 4, "Flaga", 6}, {5, 15, "Długość", 7}, {16, 47, "Dane", 8}}
	if d.Title != "Nagłówek" || d.AccDescr != "Opis" || len(d.Fields) != len(want) {
		t.Fatalf("diagram: %+v", d)
	}
	for i, f := range d.Fields {
		if f != want[i] {
			t.Errorf("field %d: got %+v, want %+v", i, f, want[i])
		}
	}
}

// REQ: PKT-004
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                     {CodeHeader, 0},
		"packets\n0-7: \"a\"":                  {CodeHeader, 1},
		"packet-beta\n0-7 a":                   {CodeStatement, 2},
		"packet-beta\n1-7: \"a\"":              {CodeRange, 2},
		"packet-beta\n0-7: \"a\"\n9-15: \"b\"": {CodeRange, 3},
		"packet-beta\n0-7: \"a\"\n4-15: \"b\"": {CodeRange, 3},
		"packet-beta\n7-0: \"a\"":              {CodeRange, 2},
		"packet-beta\n+0: \"a\"":               {CodeRange, 2},
		"packet-beta\n0-4000: \"a\"":           {CodeRange, 2},
		"packet-beta\ntitle Sam tytuł":         {CodeEmpty, 0},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: PKT-003
func TestLayout(t *testing.T) {
	d, err := Parse("packet-beta\n0-7: \"A\"\n8: \"B\"\n9-39: \"Długie pole\"\n40-63: \"C\"")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	var fields []scene.Item
	bits, labels := map[string]int{}, map[string]int{}
	for _, it := range sc.Items {
		switch it.Role {
		case "packet-field":
			fields = append(fields, it)
		case "packet-bit":
			bits[it.Text]++
		case "packet-label":
			labels[it.Text]++
		}
	}
	// Pole 9–39 jest w dwóch kawałkach: 9–31 w pierwszym wierszu i 32–39 w drugim.
	if len(fields) != 5 || labels["Długie pole"] != 2 {
		t.Fatalf("%d field pieces, long label drawn %d times", len(fields), labels["Długie pole"])
	}
	bit := fields[1].W
	for i, bitsWide := range []float64{8, 1, 23, 8, 24} {
		if math.Abs(fields[i].W-bitsWide*bit) > 0.01 {
			t.Errorf("piece %d is %.1f px wide, want %v bits of %.1f px", i, fields[i].W, bitsWide, bit)
		}
	}
	if fields[2].X+fields[2].W != 32*bit || fields[3].X != 0 || fields[3].Y <= fields[2].Y || fields[4].X != fields[3].X+fields[3].W {
		t.Errorf("the long field should fill the first row and continue at the start of the second")
	}
	for _, want := range []string{"0", "7", "8", "9", "31", "32", "39", "40", "63"} {
		if bits[want] != 1 {
			t.Errorf("bit number %s drawn %d times, want once", want, bits[want])
		}
	}
}
