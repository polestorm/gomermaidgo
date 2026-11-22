package scene

import (
	"math"
	"strconv"
	"testing"
)

// hue zwraca barwę koloru #rrggbb w stopniach.
func hue(color string) float64 {
	v, _ := strconv.ParseUint(color[1:], 16, 32)
	r, g, b := float64(v>>16&255)/255, float64(v>>8&255)/255, float64(v&255)/255
	hi, lo := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := hi - lo
	var h float64
	switch {
	case d == 0:
		return 0
	case hi == r:
		h = math.Mod((g-b)/d, 6)
	case hi == g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return math.Mod(h*60+360, 360)
}

func apart(a, b string) float64 {
	d := math.Abs(hue(a) - hue(b))
	return math.Min(d, 360-d)
}

// REQ: COM-004
// Kolory serii: dwa pierwsze leżą po przeciwnych stronach koła barw, każdy następny z palety jest
// daleko od wszystkich wcześniejszych, a po wyczerpaniu palety sąsiedzi nadal się różnią.
func TestDistinct(t *testing.T) {
	if d := apart(Distinct(0).Stroke, Distinct(1).Stroke); d < 150 {
		t.Errorf("the first two colours are %.0f degrees apart", d)
	}
	seen := map[string]bool{}
	for i := 0; i < len(Palette); i++ {
		c := Distinct(i)
		if seen[c.Stroke] {
			t.Errorf("colour %d repeats", i)
		}
		seen[c.Stroke] = true
		for k := 0; k < i; k++ {
			if d := apart(c.Stroke, Distinct(k).Stroke); d < 40 {
				t.Errorf("colours %d and %d are only %.0f degrees apart", k, i, d)
			}
		}
		if i < 4 {
			for k := 0; k < i; k++ {
				if d := apart(c.Stroke, Distinct(k).Stroke); d < 55 {
					t.Errorf("among the first four, colours %d and %d are only %.0f degrees apart", k, i, d)
				}
			}
		}
	}
	if len(seen) != len(Palette) {
		t.Errorf("the first %d colours should be the whole palette", len(Palette))
	}
	for i := len(Palette); i < 40; i++ {
		a, b := Distinct(i), Distinct(i+1)
		if len(a.Stroke) != 7 || len(a.Fill) != 7 || apart(a.Stroke, b.Stroke) < 90 {
			t.Errorf("generated colours %d and %d: %s %s", i, i+1, a.Stroke, b.Stroke)
		}
	}
	if Distinct(-2) != Distinct(2) {
		t.Errorf("a negative index should not panic or differ")
	}
}
