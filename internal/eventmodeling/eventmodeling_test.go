package eventmodeling

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: EVM-001 EVM-002 EVM-003
func TestParse(t *testing.T) {
	m, err := Parse("%% c\neventmodeling\naccTitle: Opis\ntf 01 ui Ekran\ntimeframe 02 command Sklep.Dodaj `json`{ opis: string, ilosc: 2 }\ntf 03 evt Dodano [[D1]]\nrf 04 rmo Lista ->> 02 ->> 03\ndata D1 {\n  klient: 'Jan'\n}")
	if err != nil {
		t.Fatal(err)
	}
	if m.AccTitle != "Opis" || len(m.Frames) != 4 {
		t.Fatalf("model: %+v", m)
	}
	f := m.Frames[1]
	if f.Kind != Command || f.Namespace != "Sklep" || f.Name != "Dodaj" || strings.Join(f.Data, "|") != "opis: string|ilosc: 2" {
		t.Errorf("frame 2: %+v", f)
	}
	if d := m.Frames[2].Data; strings.Join(d, "|") != "klient: Jan" {
		t.Errorf("data block: %q", d)
	}
	if r := m.Frames[3]; !r.Reset || r.Kind != ReadModel || len(r.From) != 2 || r.From[1] != 3 {
		t.Errorf("reset frame: %+v", r)
	}
}

// REQ: EVM-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                      {CodeHeader, 0},
		"eventModeling":                         {CodeHeader, 1},
		"eventmodeling\n%% nic":                 {CodeEmpty, 0},
		"eventmodeling\ntf 01 widget A":         {CodeStatement, 2},
		"eventmodeling\ntf 01 ui A\ntf 1 evt B": {CodeStatement, 3},
		"eventmodeling\ntf 01 ui A ->> 07":      {CodeStatement, 2},
		"eventmodeling\ntf 01 ui A [[X]]":       {CodeStatement, 2},
		"eventmodeling\ntf 01 ui A { a: b":      {CodeStatement, 2},
		"eventmodeling\ndata X {\n a: 1":        {CodeStatement, 2},
		"eventmodeling\nslice 1":                {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: EVM-004
func TestLayout(t *testing.T) {
	m, err := Parse("eventmodeling\ntf 01 ui Ekran\ntf 02 cmd Dodaj\ntf 03 evt Dodano\nrf 04 rmo Lista ->> 03\ntf 05 evt Inne")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var xs, ys []float64
	edges := 0
	for _, it := range sc.Items {
		switch it.Role {
		case "eventmodeling-element":
			xs, ys = append(xs, it.X), append(ys, it.Y)
		case "eventmodeling-edge":
			edges++
		}
	}
	if len(xs) != 5 {
		t.Fatalf("%d elements", len(xs))
	}
	for i := 1; i < len(xs); i++ {
		if xs[i] <= xs[i-1] {
			t.Errorf("frame %d is not right of frame %d", i+1, i)
		}
	}
	// Tory: ekran nad poleceniem, polecenie nad zdarzeniem; model odczytu w torze polecenia.
	if !(ys[0] < ys[1] && ys[1] < ys[2] && ys[3] == ys[1] && ys[4] == ys[2]) {
		t.Errorf("lanes: %v", ys)
	}
	// 1→2, 2→3, 3→4 (jawne; reset bez wywnioskowanego), 4→5: linia i grot każda.
	if edges != 8 {
		t.Errorf("%d edge items, want 8", edges)
	}
}

// REQ: EVM-004
// Elementy w różnych torach mogą na siebie zachodzić w poziomie (rysunek jest węższy), w tym samym
// torze nie; strzałki są łamane pod kątem prostym, nie przechodzą przez elementy i nie krzyżują się
// przy wspólnym celu.
func TestCompactColumnsAndRoutes(t *testing.T) {
	m, err := Parse("eventmodeling\ntf 01 cmd Zglos\ntf 02 evt Zgloszona\ntf 03 evt Uznana\ntf 04 evt Zgloszona\nrf 05 rmo Lista ->> 02 ->> 03 ->> 04\ntf 06 ui Panel")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var boxes []scene.Box
	var lines [][]scene.Point
	for _, it := range sc.Items {
		switch {
		case it.Role == "eventmodeling-element":
			boxes = append(boxes, it.Bounds())
		case it.Role == "eventmodeling-edge" && it.Kind == scene.Polyline:
			lines = append(lines, it.Points)
		}
	}
	// 01 (polecenie) i 02 (zdarzenie) w różnych torach: 02 zaczyna się przed końcem 01.
	if boxes[1].X0 >= boxes[0].X1 {
		t.Errorf("frames in different lanes do not overlap: %v %v", boxes[0], boxes[1])
	}
	// 02, 03, 04 w jednym torze: bez zachodzenia.
	for i := 2; i <= 3; i++ {
		if boxes[i].X0 < boxes[i-1].X1 {
			t.Errorf("frames %d and %d in one lane overlap", i, i+1)
		}
	}
	type seg struct{ a, b scene.Point }
	var segs [][]seg
	for _, pts := range lines {
		var ss []seg
		for k := 1; k < len(pts); k++ {
			a, b := pts[k-1], pts[k]
			if math.Abs(a.X-b.X) > 0.5 && math.Abs(a.Y-b.Y) > 0.5 {
				t.Errorf("diagonal segment %v–%v", a, b)
			}
			ss = append(ss, seg{a, b})
			for _, bx := range boxes {
				mx, my := (a.X+b.X)/2, (a.Y+b.Y)/2
				if mx > bx.X0+1 && mx < bx.X1-1 && my > bx.Y0+1 && my < bx.Y1-1 {
					t.Errorf("segment %v–%v runs through an element", a, b)
				}
			}
		}
		segs = append(segs, ss)
	}
	// Strzałki do modelu odczytu (z trzech zdarzeń) nie krzyżują się.
	cross := func(h, v seg) bool {
		if math.Abs(h.a.Y-h.b.Y) > 0.5 || math.Abs(v.a.X-v.b.X) > 0.5 {
			return false
		}
		x, y := v.a.X, h.a.Y
		return x > math.Min(h.a.X, h.b.X)+0.5 && x < math.Max(h.a.X, h.b.X)-0.5 && y > math.Min(v.a.Y, v.b.Y)+0.5 && y < math.Max(v.a.Y, v.b.Y)-0.5
	}
	for i := range segs {
		for j := range segs {
			if i == j {
				continue
			}
			for _, h := range segs[i] {
				for _, v := range segs[j] {
					if cross(h, v) {
						t.Errorf("arrows %d and %d cross", i, j)
					}
				}
			}
		}
	}
}
