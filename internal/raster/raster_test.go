package raster

import (
	"math"
	"testing"
)

func alpha(c *Canvas, x, y int) int { return int(c.Image.Pix[y*c.Image.Stride+4*x+3]) }

// REQ: PNG-002
// Pokrycie piksela jest polem kształtu w jego obrębie; kierunek przeciwny wycina otwór.
func TestFillCoverage(t *testing.T) {
	c := New(20, 20)
	square := func(x0, y0, x1, y1 float64) []Point { return []Point{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}} }
	c.Fill([][]Point{square(2, 2, 10.5, 10.25)}, 1, 0, 0, 1)
	for _, tc := range []struct{ x, y, want int }{{1, 5, 0}, {2, 2, 255}, {9, 9, 255}, {10, 5, 128}, {5, 10, 64}, {10, 10, 32}, {11, 5, 0}, {5, 11, 0}} {
		if got := alpha(c, tc.x, tc.y); math.Abs(float64(got-tc.want)) > 1 {
			t.Errorf("pixel %d,%d: alpha %d, want %d", tc.x, tc.y, got, tc.want)
		}
	}
	if c.Image.Pix[5*c.Image.Stride+4*5] != 255 || c.Image.Pix[5*c.Image.Stride+4*5+1] != 0 {
		t.Errorf("colour inside the square should be red")
	}
	// Otwór: wielokąt wewnętrzny o przeciwnym kierunku.
	c = New(20, 20)
	hole := square(6, 6, 12, 12)
	hole[1], hole[3] = hole[3], hole[1]
	c.Fill([][]Point{square(2, 2, 16, 16), hole}, 0, 0, 0, 1)
	if alpha(c, 4, 4) != 255 || alpha(c, 8, 8) != 0 || alpha(c, 14, 14) != 255 {
		t.Errorf("hole: %d %d %d", alpha(c, 4, 4), alpha(c, 8, 8), alpha(c, 14, 14))
	}
	// Dwa nakładające się wielokąty o tym samym kierunku nie dają pokrycia ponad pełne.
	c = New(20, 20)
	c.Fill([][]Point{square(2, 2, 12, 12), square(8, 8, 16, 16)}, 0, 0, 0, 0.5)
	if a, b := alpha(c, 4, 4), alpha(c, 10, 10); a != b || math.Abs(float64(a-128)) > 1 {
		t.Errorf("overlap should not double the cover: %d %d", a, b)
	}
	// Kształt wystający poza obraz jest przycinany bez śladów w pozostałych wierszach.
	c = New(10, 10)
	c.Fill([][]Point{square(-5, -5, 4, 4), square(7, 7, 30, 30)}, 0, 0, 0, 1)
	if alpha(c, 0, 0) != 255 || alpha(c, 5, 5) != 0 || alpha(c, 9, 9) != 255 || alpha(c, 9, 2) != 0 || alpha(c, 2, 9) != 0 {
		t.Errorf("clipping: %d %d %d %d %d", alpha(c, 0, 0), alpha(c, 5, 5), alpha(c, 9, 9), alpha(c, 9, 2), alpha(c, 2, 9))
	}
}

// REQ: PNG-002
// Obrys ma podaną grubość, załamania są wypełnione, a wzór kresek dzieli linię na kawałki.
func TestStrokeAndDash(t *testing.T) {
	c := New(30, 30)
	c.Fill(Stroke([]Point{{5, 10}, {20, 10}, {20, 25}}, 2, false), 0, 0, 0, 1)
	for _, tc := range []struct{ x, y, want int }{{10, 9, 255}, {10, 10, 255}, {10, 8, 0}, {10, 11, 0}, {4, 9, 0}, {19, 20, 255}, {20, 20, 255}, {21, 20, 0}, {19, 9, 255}, {20, 25, 0}} {
		if got := alpha(c, tc.x, tc.y); math.Abs(float64(got-tc.want)) > 2 {
			t.Errorf("pixel %d,%d: alpha %d, want %d", tc.x, tc.y, got, tc.want)
		}
	}
	pieces := Dash([]Point{{0, 0}, {10, 0}, {10, 10}}, []float64{4, 2}, false)
	if len(pieces) != 4 {
		t.Fatalf("pieces: %v", pieces)
	}
	total := 0.0
	for _, p := range pieces {
		for i := 0; i+1 < len(p); i++ {
			total += math.Hypot(p[i+1].X-p[i].X, p[i+1].Y-p[i].Y)
		}
	}
	// 20 jednostek linii: kreski 4+4+4 i końcowa 2.
	if math.Abs(total-14) > 1e-9 || len(pieces[1]) != 3 {
		t.Errorf("dash length %v, piece across the corner %v", total, pieces[1])
	}
	if got := Dash([]Point{{0, 0}, {5, 0}}, nil, false); len(got) != 1 {
		t.Errorf("no pattern: %v", got)
	}
	if pts := Cubic([]Point{{0, 0}}, Point{0, 10}, Point{10, 10}, Point{10, 0}); pts[len(pts)-1] != (Point{10, 0}) || len(pts) < 5 {
		t.Errorf("cubic: %v", pts)
	}
}
