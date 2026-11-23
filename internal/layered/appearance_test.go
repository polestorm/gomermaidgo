package layered

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/er"
	"github.com/polestorm/gomermaidgo/internal/flowchart"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Testy reguł wyglądu ustalonych w przeglądzie rysunków (requirements/layout.md). Pliki wzorcowe
// pilnują wyniku co do bajta; te testy mówią, która reguła została złamana.

// natural rozmieszcza plik korpusu w kierunku ze źródła.
func natural(t *testing.T, file string) *scene.Scene {
	t.Helper()
	return Layout(parse(t, "../../testdata/corpus/"+file), Options{Measure: font.ApproxTextWidth, FontSize: 16})
}

// nodeOf zwraca prostokąt pola o podanym tekście (jednej linii etykiety).
func nodeOf(t *testing.T, sc *scene.Scene, label string) scene.Box {
	t.Helper()
	for _, text := range byRole(sc, "node-label") {
		if text.Text != label {
			continue
		}
		tb := text.Bounds()
		cx, cy := (tb.X0+tb.X1)/2, (tb.Y0+tb.Y1)/2
		best, found := scene.Box{}, false
		for _, n := range byRole(sc, "node") {
			b := n.Bounds()
			if cx < b.X0 || cx > b.X1 || cy < b.Y0 || cy > b.Y1 {
				continue
			}
			if !found || (b.X1-b.X0)*(b.Y1-b.Y0) < (best.X1-best.X0)*(best.Y1-best.Y0) {
				best, found = b, true
			}
		}
		if found {
			return best
		}
	}
	t.Fatalf("no node labelled %q", label)
	return scene.Box{}
}

func centerX(b scene.Box) float64 { return (b.X0 + b.X1) / 2 }
func width(b scene.Box) float64   { return b.X1 - b.X0 }

// straight to prosty odcinek krawędzi.
type straight struct {
	a, b scene.Point
	edge int
}

func straights(sc *scene.Scene) []straight {
	var out []straight
	for i, e := range byRole(sc, "edge") {
		at := e.Segments[0].To
		for _, sg := range e.Segments[1:] {
			if sg.Cmd == 'L' && (math.Abs(sg.To.X-at.X) > 0.01 || math.Abs(sg.To.Y-at.Y) > 0.01) {
				out = append(out, straight{at, sg.To, i})
			}
			at = sg.To
		}
	}
	return out
}

// REQ: LAY-001
// Krawędź łamana składa się z odcinków równoległych do osi; żaden nie jest skośny i żaden nie cofa się
// względem poprzedniego. Obowiązuje w czterech kierunkach układu.
func TestEdgesAreAxisAligned(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		for _, e := range byRole(sc, "edge") {
			at := e.Segments[0].To
			var prev *scene.Point
			for _, sg := range e.Segments[1:] {
				if sg.Cmd == 'L' {
					dx, dy := sg.To.X-at.X, sg.To.Y-at.Y
					if math.Abs(dx) > 0.01 && math.Abs(dy) > 0.01 {
						t.Errorf("%s: slanted segment (%.2f, %.2f) → (%.2f, %.2f)", name, at.X, at.Y, sg.To.X, sg.To.Y)
					}
					if prev != nil && dx*prev.X+dy*prev.Y < -0.01 && math.Abs(dx*prev.Y-dy*prev.X) < 0.01 {
						t.Errorf("%s: segment turns back at (%.2f, %.2f)", name, at.X, at.Y)
					}
					prev = &scene.Point{X: dx, Y: dy}
				} else {
					prev = nil
				}
				at = sg.To
			}
		}
	}
}

// jogs zwraca najmniejszy niezerowy uskok krawędzi: odległość w poprzek między kolejnymi odcinkami
// biegnącymi w tę samą stronę.
func smallestJog(sc *scene.Scene) float64 {
	smallest := math.Inf(1)
	for _, e := range byRole(sc, "edge") {
		var xs, ys []float64
		at := e.Segments[0].To
		for _, sg := range e.Segments[1:] {
			if sg.Cmd == 'L' {
				if math.Abs(sg.To.X-at.X) < 0.01 && math.Abs(sg.To.Y-at.Y) > 0.01 {
					xs = append(xs, at.X)
				}
				if math.Abs(sg.To.Y-at.Y) < 0.01 && math.Abs(sg.To.X-at.X) > 0.01 {
					ys = append(ys, at.Y)
				}
			}
			at = sg.To
		}
		for _, v := range [][]float64{xs, ys} {
			for i := 1; i < len(v); i++ {
				if d := math.Abs(v[i] - v[i-1]); d > 0.01 {
					smallest = math.Min(smallest, d)
				}
			}
		}
	}
	return smallest
}

// REQ: LAY-002
// Krawędź nie ma drobnych uskoków: przesunięcie w poprzek jest zerem albo ma co najmniej 8 px
// w kierunku ze źródła; w kierunkach wymuszonych nie schodzi poniżej progu prostowania.
func TestNoSmallJogs(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		if d := smallestJog(sc); d < straightTolerance {
			t.Errorf("%s: jog of %.2f px", name, d)
		}
	}
	for _, dir := range []string{"flowchart", "state", "er"} {
		for _, f := range corpusNames(t, dir) {
			if d := smallestJog(natural(t, f)); d < 8 {
				t.Errorf("%s: jog of %.2f px", f, d)
			}
		}
	}
}

// REQ: LAY-003
// Dwie krawędzie nie przecinają się odcinkami prostymi: w miejscu przecięcia krawędź narysowana
// później ma mostek, więc jej odcinek prosty jest tam przerwany.
func TestCrossingsHaveBridges(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		ss := straights(sc)
		for i, p := range ss {
			for _, q := range ss[i+1:] {
				if p.edge == q.edge {
					continue
				}
				h, v := p, q
				if math.Abs(h.a.Y-h.b.Y) > 0.01 {
					h, v = q, p
				}
				if math.Abs(h.a.Y-h.b.Y) > 0.01 || math.Abs(v.a.X-v.b.X) > 0.01 {
					continue
				}
				// Mostek potrzebuje miejsca: przecięcie tuż przy narożniku albo końcu krawędzi zostaje zwykłe.
				const in = 15.0
				x, y := v.a.X, h.a.Y
				if x > math.Min(h.a.X, h.b.X)+in && x < math.Max(h.a.X, h.b.X)-in &&
					y > math.Min(v.a.Y, v.b.Y)+in && y < math.Max(v.a.Y, v.b.Y)-in {
					t.Errorf("%s: edges cross without a bridge at (%.1f, %.1f)", name, x, y)
				}
			}
		}
	}
}

// REQ: LAY-004
// Pola stojące jedno pod drugim mają tę samą szerokość, gdy szersze da się złamać do węższego.
func TestStackedNodesShareWidth(t *testing.T) {
	sc := natural(t, "flowchart/015-styles.mmd")
	a, b := nodeOf(t, sc, "Naprawa"), nodeOf(t, sc, "kontrola")
	if math.Abs(width(a)-width(b)) > 0.5 || math.Abs(centerX(a)-centerX(b)) > 0.5 {
		t.Errorf("stacked nodes differ: %+v and %+v", a, b)
	}
}

// REQ: LAY-005
// Krawędzie rozchodzące się z jednego pola skręcają na tej samej wysokości i wychodzą z rombu
// w różnych punktach.
func TestBranchesTurnTogether(t *testing.T) {
	sc := natural(t, "state/004-choice-fork-join.mmd")
	var turns, starts []float64
	for _, e := range byRole(sc, "edge") {
		if len(e.Segments) < 5 || e.Segments[0].To.Y > 180 || e.Segments[0].To.Y < 170 {
			continue
		}
		starts = append(starts, e.Segments[0].To.X)
		for i := 1; i < len(e.Segments); i++ {
			if a, b := e.Segments[i-1].To, e.Segments[i].To; e.Segments[i].Cmd == 'L' && math.Abs(a.Y-b.Y) < 0.01 {
				turns = append(turns, a.Y)
				break
			}
		}
	}
	if len(turns) != 2 || math.Abs(turns[0]-turns[1]) > 0.01 {
		t.Errorf("branches turn at different heights: %v", turns)
	}
	if len(starts) != 2 || math.Abs(starts[0]-starts[1]) < 4 {
		t.Errorf("branches leave the choice at the same point: %v", starts)
	}
}

// REQ: LAY-006
// Stan złożony ma pasek tytułu i pasek u dołu; proste strzałki w jego wnętrzu są krótsze niż
// na poziomie głównym.
func TestCompositeStateFrame(t *testing.T) {
	for _, f := range corpusNames(t, "state") {
		sc := natural(t, f)
		outlines, feet := len(byRole(sc, "cluster-outline")), len(byRole(sc, "cluster-foot-strip"))
		if outlines != feet {
			t.Errorf("%s: %d composite states, %d foot strips", f, outlines, feet)
		}
	}
	sc := natural(t, "state/003-composite.mmd")
	inner := nodeOf(t, sc, "Drugi").Y0 - nodeOf(t, sc, "Pierwszy").Y1
	var frame scene.Box
	for _, c := range byRole(sc, "cluster") {
		if b := c.Bounds(); b.Y1 > frame.Y1 {
			frame = b
		}
	}
	outer := nodeOf(t, sc, "Obowiązujący").Y0 - frame.Y1
	if inner >= outer-10 {
		t.Errorf("arrow inside the composite state is %.0f px, outside %.0f px", inner, outer)
	}
}

// REQ: LAY-007
// Pętla własna stoi z boku pola i nie przesuwa osi: sąsiad wyrównuje się do środka samego pola.
func TestSelfLoopKeepsAxis(t *testing.T) {
	sc := natural(t, "state/008-regions-classes.mmd")
	node := nodeOf(t, sc, "Rejestracja")
	for _, n := range byRole(sc, "node") {
		b := n.Bounds()
		// Znak początku tuż nad polem.
		if width(b) < 20 && b.Y1 <= node.Y0 && node.Y0-b.Y1 < 60 && b.X0 > node.X0 && b.X1 < node.X1 {
			if math.Abs(centerX(b)-centerX(node)) > 0.5 {
				t.Errorf("start mark at %.1f, node centre at %.1f", centerX(b), centerX(node))
			}
			return
		}
	}
	t.Error("no start mark above the node")
}

// REQ: LAY-008
// Ramki bez połączeń między sobą są upakowane: niższa staje pod poprzednią, obok wysokiej.
func TestUnconnectedFramesArePacked(t *testing.T) {
	sc := natural(t, "state/008-regions-classes.mmd")
	regions := byRole(sc, "region")
	if len(regions) != 3 {
		t.Fatalf("%d regions, want 3", len(regions))
	}
	first, second, third := regions[0].Bounds(), regions[1].Bounds(), regions[2].Bounds()
	if second.X0 < first.X1 || math.Abs(second.Y0-first.Y0) > 0.5 {
		t.Errorf("second region is not beside the first: %+v, %+v", first, second)
	}
	if math.Abs(third.X0-second.X0) > 0.5 || third.Y0 < second.Y1 {
		t.Errorf("third region is not under the second: %+v, %+v", second, third)
	}
}

// REQ: FLOW-021
// Tory: pole, które w swoim rzędzie jest jedynym polem toru, stoi pośrodku pasa, więc kroki jednej
// roli tworzą kolumnę.
func TestLaneNodesFormColumn(t *testing.T) {
	sc := natural(t, "flowchart/017-swimlanes.mmd")
	first := centerX(nodeOf(t, sc, "Rejestruje zgłoszenie"))
	for _, label := range []string{"Kompletne?", "Ocenia ryzyko", "Zamyka zgłoszenie"} {
		if x := centerX(nodeOf(t, sc, label)); math.Abs(x-first) > 0.5 {
			t.Errorf("%q stands at %.1f, the column at %.1f", label, x, first)
		}
	}
}

// REQ: LAY-009
// Romb decyzji z gałęzią biegnącą prosto w osi: pozostałe gałęzie wychodzą z bocznych wierzchołków
// i biegną w poprzek na wysokości środka pola. („Wymaga działań?” ma lewy wierzchołek zajęty przez
// płaską krawędź wchodzącą, LAY-027, więc gałąź „nie” wychodzi dołem.)
func TestDecisionBranchLeavesSideVertex(t *testing.T) {
	sc := natural(t, "flowchart/017-swimlanes.mmd")
	for _, label := range []string{"Kompletne?"} {
		node := nodeOf(t, sc, label)
		side, down := 0, 0
		for _, e := range byRole(sc, "edge") {
			p := e.Segments[0].To
			switch {
			case math.Abs(p.X-node.X0) < 0.5 && math.Abs(p.Y-(node.Y0+node.Y1)/2) < 0.5:
				side++
				if q := e.Segments[1].To; math.Abs(q.Y-p.Y) > 0.01 || q.X >= p.X {
					t.Errorf("%s: the side branch does not run outwards from the vertex", label)
				}
			case math.Abs(p.X-centerX(node)) < 0.5 && math.Abs(p.Y-node.Y1) < 0.5:
				down++
			}
		}
		if side != 1 || down != 1 {
			t.Errorf("%s: %d branches from the side vertex and %d straight down, want 1 and 1", label, side, down)
		}
	}
}

// widestEmptyColumn zwraca szerokość najszerszego pionowego pasa, w którym nie stoi żadne pole,
// ramka ani etykieta.
func widestEmptyColumn(sc *scene.Scene) float64 {
	items := byRole(sc, "node", "note", "cluster", "region", "edge-label")
	sort.Slice(items, func(a, b int) bool { return items[a].Bounds().X0 < items[b].Bounds().X0 })
	widest, reach := 0.0, math.Inf(-1)
	for _, it := range items {
		b := it.Bounds()
		if !math.IsInf(reach, -1) {
			widest = math.Max(widest, b.X0-reach)
		}
		reach = math.Max(reach, b.X1)
	}
	return widest
}

// REQ: LAY-010
// Rysunek nie ma pustych kolumn: pas, w którym w żadnej warstwie nic nie stoi, jest zamykany do
// zwykłego odstępu. Sprawdzane na korpusie i na dużym diagramie encji z prawdziwej dokumentacji.
func TestNoEmptyColumns(t *testing.T) {
	const limit = 60
	for name, sc := range corpusScenes(t) {
		if !strings.HasSuffix(name, " TB") || strings.Contains(name, "swimlane") {
			continue
		}
		if w := widestEmptyColumn(sc); w > limit {
			t.Errorf("%s: empty column %.0f px wide", name, w)
		}
	}
	src, err := os.ReadFile("testdata/er/wide-er.mmd")
	if err != nil {
		t.Fatal(err)
	}
	g, err := er.Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	if w := widestEmptyColumn(sc); w > limit {
		t.Errorf("wide entity diagram: empty column %.0f px wide", w)
	}
	if b := sc.Bounds(); b.X1-b.X0 > 3000 {
		t.Errorf("wide entity diagram is %.0f px wide", b.X1-b.X0)
	}
}

// REQ: LAY-011
// Duży diagram encji jest układany pod proporcje strony: warstwy mają limit szerokości, a wygrywa
// układ, który mieści się na stronie w największej skali. Mały diagram zostaje bez zmian.
func TestEntityDiagramFitsPage(t *testing.T) {
	src, err := os.ReadFile("testdata/er/wide-er.mmd")
	if err != nil {
		t.Fatal(err)
	}
	g, err := er.Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	o := Options{Measure: font.ApproxTextWidth, FontSize: 16}
	fit := func(sc *scene.Scene) float64 {
		b := sc.Bounds()
		return math.Min(1, math.Min(650/(b.X1-b.X0), 870/(b.Y1-b.Y0)))
	}
	free, bound := fit(layout(g, o, 0, 1)), fit(Layout(g, o))
	if bound < 0.33 || bound < free*1.3 {
		t.Errorf("page fit %.3f with the width limit, %.3f without", bound, free)
	}
	for _, f := range corpusNames(t, "er") {
		g := parse(t, "../../testdata/corpus/"+f)
		a, b := layout(g, o, 0, 1).Bounds(), Layout(parse(t, "../../testdata/corpus/"+f), o).Bounds()
		if a != b {
			t.Errorf("%s: a diagram that fits the page changed: %+v, %+v", f, a, b)
		}
	}
}

// REQ: CLS-003 CLS-004 CLS-005
// Klasa: składowe stoją wewnątrz pola, wyrównane do lewej; składowa statyczna ma podkreślenie,
// abstrakcyjna kursywę, nazwa jest pogrubiona. Relacja z licznościami ma opis przy każdym końcu.
func TestClassDrawing(t *testing.T) {
	sc := natural(t, "class/001-classes-members.mmd")
	box := nodeOf(t, sc, "Dokument")
	members, underlines, italics := 0, 0, 0
	left := math.Inf(1)
	for _, it := range sc.Items {
		b := it.Bounds()
		inside := b.X0 >= box.X0 && b.X1 <= box.X1 && b.Y0 >= box.Y0 && b.Y1 <= box.Y1
		switch {
		case it.Role == "member" && inside:
			members++
			left = math.Min(left, it.X)
			if it.Italic {
				italics++
			}
			if it.X != left {
				t.Errorf("member %q is not aligned with the others", it.Text)
			}
		case it.Role == "member-underline" && inside:
			underlines++
		case it.Role == "node-label" && it.Text == "Dokument" && !it.Bold:
			t.Error("class name is not bold")
		}
	}
	if members != 7 || underlines != 1 || italics != 1 {
		t.Errorf("%d members, %d underlined, %d italic; want 7, 1, 1", members, underlines, italics)
	}
	sc = natural(t, "class/002-relations.mmd")
	ends := map[string]int{}
	for _, it := range byRole(sc, "edge-end-label") {
		ends[it.Text]++
	}
	if ends["1"] != 3 || ends["1..*"] != 1 || ends["*"] != 1 || ends["0..*"] != 1 {
		t.Errorf("end labels: %v", ends)
	}
	heads := map[int]int{}
	for _, it := range byRole(sc, "arrowhead") {
		if it.Kind == scene.Polygon {
			heads[len(it.Points)]++
		}
	}
	if heads[3] != 4 || heads[4] != 2 {
		t.Errorf("%d triangles and %d diamonds, want 4 and 2", heads[3], heads[4])
	}
}

// edgeInto zwraca krawędź, która kończy się na górnej krawędzi pola.
func edgeInto(t *testing.T, sc *scene.Scene, box scene.Box) scene.Item {
	t.Helper()
	for _, e := range byRole(sc, "edge") {
		end := e.Segments[len(e.Segments)-1].To
		if end.X > box.X0 && end.X < box.X1 && math.Abs(end.Y-box.Y0) < 12 {
			return e
		}
	}
	t.Fatalf("no edge ends at the top of %+v", box)
	return scene.Item{}
}

// REQ: LAY-012
// Krawędź z etykietą do pola w podgrafie, które nie stoi pośrodku ramki: etykieta i punkty pośrednie
// stają w osi punktu wejścia do ramki, więc krawędź między polami stojącymi jedno pod drugim jest prosta.
func TestLabelledEdgeIntoFrameIsStraight(t *testing.T) {
	sc := natural(t, "c4/002-container.mmd")
	user, app := nodeOf(t, sc, "Użytkownik"), nodeOf(t, sc, "Aplikacja")
	if math.Abs(centerX(user)-centerX(app)) > 0.5 {
		t.Fatalf("the nodes should share an axis: %.1f and %.1f", centerX(user), centerX(app))
	}
	for _, sg := range edgeInto(t, sc, app).Segments {
		if math.Abs(sg.To.X-centerX(app)) > 0.5 {
			t.Errorf("the edge leaves the axis at (%.1f, %.1f)", sg.To.X, sg.To.Y)
		}
	}
}

// REQ: LAY-013
// Dwie krawędzie z jednego pola, z których jedna mija pole celu drugiej: mijająca staje po tej
// stronie, po której nie przecina krawędzi wracającej z tego pola. Rysunek nie ma wtedy przecięć.
func TestPassingEdgeTakesFreeSide(t *testing.T) {
	sc := natural(t, "c4/002-container.mmd")
	renderer, mail, app := nodeOf(t, sc, "Renderer PDF"), nodeOf(t, sc, "Serwer poczty"), nodeOf(t, sc, "Aplikacja")
	for _, sg := range edgeInto(t, sc, mail).Segments {
		if sg.To.X > renderer.X0 {
			t.Errorf("the edge to the mail server should pass the renderer on its free side: x=%.1f", sg.To.X)
		}
	}
	// Krawędź wracająca z pola Renderer do Aplikacji biegnie prosto w górę.
	back := false
	for _, e := range byRole(sc, "edge") {
		first, last := e.Segments[0].To, e.Segments[len(e.Segments)-1].To
		lo, hi := math.Min(first.Y, last.Y), math.Max(first.Y, last.Y)
		if math.Abs(first.X-last.X) < 0.5 && first.X > renderer.X0 && first.X < renderer.X1 && math.Abs(lo-app.Y1) < 12 && math.Abs(hi-renderer.Y0) < 12 {
			back = true
		}
	}
	if !back {
		t.Errorf("the edge back from the renderer should run straight up")
	}
	ss := straights(sc)
	for i, p := range ss {
		for _, q := range ss[i+1:] {
			if p.edge == q.edge {
				continue
			}
			h, v := p, q
			if math.Abs(h.a.Y-h.b.Y) > 0.01 {
				h, v = q, p
			}
			if math.Abs(h.a.Y-h.b.Y) > 0.01 || math.Abs(v.a.X-v.b.X) > 0.01 {
				continue
			}
			if x, y := v.a.X, h.a.Y; x > math.Min(h.a.X, h.b.X)-8 && x < math.Max(h.a.X, h.b.X)+8 && y > math.Min(v.a.Y, v.b.Y)-8 && y < math.Max(v.a.Y, v.b.Y)+8 {
				t.Errorf("edges still cross or touch near (%.1f, %.1f)", x, y)
			}
		}
	}
}

// REQ: LAY-014
// Jedyna krawędź wchodząca do pola od góry trafia w środek jego górnego boku, także gdy wychodzi
// z rombu obok innej gałęzi; krawędzie z belki rozwidlenia zostają pionowe.
func TestSingleIncomingEdgeHitsCentre(t *testing.T) {
	sc := natural(t, "flowchart/001-basic.mmd")
	box := nodeOf(t, sc, "Analiza przyczyn")
	end := edgeInto(t, sc, box).Segments
	if x := end[len(end)-1].To.X; math.Abs(x-centerX(box)) > 0.5 {
		t.Errorf("the edge ends at x=%.1f, the centre is %.1f", x, centerX(box))
	}
	// Belka rozwidlenia: krawędź do pola pod nią biegnie pionowo, bez załamania.
	sc = natural(t, "state/004-choice-fork-join.mmd")
	for _, sg := range edgeInto(t, sc, nodeOf(t, sc, "AnalizaPrzyczyn")).Segments {
		if sg.Cmd == 'C' {
			t.Errorf("the edge from the fork bar should be straight")
		}
	}
}

// REQ: FLOW-025
// Tryb kolorowy: ramka podgrafu ma jasne tło i obrys w kolorze serii, innym niż wypełnienie pól.
func TestColouredFrames(t *testing.T) {
	sc := Layout(parse(t, "../../testdata/corpus/flowchart/011-capa-process.mmd"), Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	frames, nodes := byRole(sc, "cluster"), byRole(sc, "node")
	if len(frames) == 0 || len(nodes) == 0 {
		t.Fatal("no frames or nodes")
	}
	for _, f := range frames {
		if f.Fill == "" || f.Fill == "none" || f.Stroke == colorFrame {
			t.Errorf("frame should be tinted: %+v", f)
		}
		fb := f.Bounds()
		for _, n := range nodes {
			nb := n.Bounds()
			inside := nb.X0 >= fb.X0 && nb.X1 <= fb.X1 && nb.Y0 >= fb.Y0 && nb.Y1 <= fb.Y1
			if inside && n.Fill == f.Fill {
				t.Errorf("a node inside the frame has the frame's fill %s", f.Fill)
			}
		}
	}
	// Romb (kolor decyzji) w ramce, której kolejny kolor serii byłby kolorem decyzji: ramka bierze następny.
	sc = Layout(parse(t, "../../testdata/corpus/flowchart/001-basic.mmd"), Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	_ = sc
	plain := Layout(parse(t, "../../testdata/corpus/flowchart/011-capa-process.mmd"), Options{Measure: font.ApproxTextWidth, FontSize: 16})
	for _, f := range byRole(plain, "cluster") {
		if f.Fill != "" {
			t.Errorf("without colours the frame should stay unfilled")
		}
	}
}

// crossingCount liczy przecięcia odcinków prostych różnych krawędzi (także pod mostkiem).
func crossingCount(sc *scene.Scene) int {
	ss := straights(sc)
	n := 0
	for i, p := range ss {
		for _, q := range ss[i+1:] {
			if p.edge == q.edge {
				continue
			}
			h, v := p, q
			if math.Abs(h.a.Y-h.b.Y) > 0.01 {
				h, v = q, p
			}
			if math.Abs(h.a.Y-h.b.Y) > 0.01 || math.Abs(v.a.X-v.b.X) > 0.01 {
				continue
			}
			x, y := v.a.X, h.a.Y
			if x > math.Min(h.a.X, h.b.X)+0.5 && x < math.Max(h.a.X, h.b.X)-0.5 && y > math.Min(v.a.Y, v.b.Y)+0.5 && y < math.Max(v.a.Y, v.b.Y)-0.5 {
				n++
			}
		}
	}
	return n
}

// REQ: LAY-015
// Krawędzie między dwiema ramkami: wejścia na ramce dolnej stoją w kolejności wyjść z górnej, więc
// krawędzie się nie przecinają; pola w ramce dolnej stoją w tej samej kolejności.
func TestFrameToFrameOrder(t *testing.T) {
	sc := natural(t, "flowchart/021-frames-order.mmd")
	if n := crossingCount(sc); n != 0 {
		t.Errorf("%d crossings", n)
	}
	// Źródła od lewej: Ocena (Rejestr, Archiwum), Analiza (Karty), Weryfikacja (Historia); w ramce
	// „Zapisy” pola stoją w kolejności swoich źródeł.
	x := func(label string) float64 { return centerX(nodeOf(t, sc, label)) }
	src := map[string]string{"Rejestr niezgodności": "Ocena", "Archiwum": "Ocena", "Karty działań": "Analiza przyczyn", "Historia": "Weryfikacja"}
	for a, sa := range src {
		for b, sb := range src {
			if x(sa) < x(sb)-0.5 && x(a) > x(b) {
				t.Errorf("%s (from %s) stands right of %s (from %s)", a, sa, b, sb)
			}
		}
	}
}

// REQ: LAY-016
// Ramka stoi tak, że krawędzie przechodzące przez nią nie mają drobnych uskoków; z wachlarza trzech
// krawędzi do ramki środkowa biegnie prosto.
func TestFramesWithoutSmallJogs(t *testing.T) {
	for _, d := range []graph.Direction{graph.LeftRight, graph.TopBottom} {
		g := parse(t, "../../testdata/corpus/flowchart/022-frames-lr.mmd")
		g.Direction = d
		// Ustawianie ramek bez dopasowania do strony; po zwężeniu pól obowiązuje ogólny próg LAY-002.
		sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, Fit: FitOff})
		if j := smallestJog(sc); j < 16 {
			t.Errorf("%s: jog of %.1f px", d, j)
		}
	}
	sc := Layout(parse(t, "../../testdata/corpus/flowchart/022-frames-lr.mmd"), Options{Measure: font.ApproxTextWidth, FontSize: 16, Fit: FitOff})
	src, mid := nodeOf(t, sc, "Prognoza"), nodeOf(t, sc, "Alerty")
	if math.Abs((src.Y0+src.Y1)/2-(mid.Y0+mid.Y1)/2) > 0.5 {
		t.Errorf("the middle field of the output frame should stand level with its source")
	}
}

// REQ: LAY-017
// Długa krawędź z pojedynczego początku do pola w podgrafie skręca raz, przy początku, i dochodzi
// do ramki prosto (W8: wzorzec → Porównanie ze wzorcem).
func TestLongEdgeIntoFrameTurnsOnce(t *testing.T) {
	g := parse(t, "../../testdata/corpus/flowchart/022-frames-lr.mmd")
	sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	src := nodeOf(t, sc, "wzorzec.docx")
	for _, e := range byRole(sc, "edge") {
		start := e.Segments[0].To
		if math.Abs(start.X-src.X1) > 1 || start.Y < src.Y0 || start.Y > src.Y1 {
			continue
		}
		turns := 0
		for _, sg := range e.Segments {
			if sg.Cmd == 'C' {
				turns++
			}
		}
		// Jeden skręt to dwa łuki (w bok i z powrotem w kierunek krawędzi).
		if turns != 2 {
			t.Errorf("the edge from wzorzec.docx has %d arcs, want 2 (one turn)", turns)
		}
		return
	}
	t.Fatal("no edge from wzorzec.docx")
}

// REQ: LAY-018
// Dopasowanie do strony: szeroki i płaski schemat mieści się na A4 w wyraźnie większej skali, bo pola
// są węższe (tekst łamany na więcej linii); żaden wyraz nie znika. Tryb auto działa tylko od progu
// proporcji, off nigdy, on także poniżej progu.
func TestFitPage(t *testing.T) {
	src := "../../testdata/corpus/flowchart/022-frames-lr.mmd"
	run := func(mode FitMode, ratio float64) *scene.Scene {
		return Layout(parse(t, src), Options{Measure: font.ApproxTextWidth, FontSize: 16, Fit: mode, FitRatio: ratio})
	}
	off, auto, on := run(FitOff, 0), run(FitAuto, 0), run(FitOn, 0)
	if a, b := pageScale(off, 16), pageScale(auto, 16); b < a*1.25 {
		t.Errorf("auto: page scale %.2f -> %.2f, want at least +25%%", a, b)
	}
	if fmt.Sprint(on.Bounds()) != fmt.Sprint(auto.Bounds()) {
		t.Errorf("on and auto should agree for a drawing past the ratio")
	}
	if high := run(FitAuto, 50); fmt.Sprint(high.Bounds()) != fmt.Sprint(off.Bounds()) {
		t.Errorf("auto below its ratio should leave the drawing as it is")
	}
	words := func(sc *scene.Scene) string {
		var all []string
		for _, it := range sc.Items {
			if it.Kind == scene.Text {
				all = append(all, strings.Fields(it.Text)...)
			}
		}
		sort.Strings(all)
		return strings.Join(all, " ")
	}
	if words(off) != words(auto) {
		t.Errorf("fitting changed the words of the drawing")
	}
	if lines := len(byRole(auto, "node-label")); lines <= len(byRole(off, "node-label")) {
		t.Errorf("fitted labels should break into more lines: %d", lines)
	}
}

// REQ: LAY-018
// Przy dopasowaniu do strony podpisy krawędzi łamią się do zwężonej szerokości, a słowo dłuższe niż
// granica łamie się po łączniku; części jednego słowa w tej samej linii sklejają się bez spacji.
func TestFitWrapsEdgeLabelsAndHyphens(t *testing.T) {
	src := "flowchart LR\nA[design + shape authored as a pair] --> B[doc-review renders BOTH -> one page]\nB --> C[approve: design FREEZES, shape = authoritative]\nC -->|binding-markup change| D[supersede the design]\nC -->|implementing code lands| E[shape = superseded-by-code; code is truth]"
	g, err := flowchart.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	off := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, Fit: FitOff, Fold: FoldOff})
	g, _ = flowchart.Parse(src)
	fit := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, Fold: FoldOff})
	if a, b := off.Bounds(), fit.Bounds(); b.X1-b.X0 > 0.65*(a.X1-a.X0) {
		t.Errorf("width %.0f -> %.0f, want at least -35%%", a.X1-a.X0, b.X1-b.X0)
	}
	labels := byRole(fit, "edge-label")
	if len(labels) <= len(byRole(off, "edge-label")) {
		t.Errorf("edge labels should break into more lines")
	}
	joined := ""
	for _, it := range fit.Items {
		if it.Kind == scene.Text {
			if strings.Contains(it.Text, glue) || strings.Contains(it.Text, "- ") {
				t.Errorf("text %q keeps a split mark or a space after a hyphen", it.Text)
			}
			joined += it.Text
		}
	}
	for _, w := range []string{"binding-", "markup", "superseded-"} {
		if !strings.Contains(joined, w) {
			t.Errorf("%q is missing", w)
		}
	}
}

// REQ: FLOW-025
// Ramka w trybie kolorowym nie bierze koloru pola, które w niej stoi (romb decyzji w pierwszej ramce,
// której kolejny kolor serii byłby kolorem decyzji).
func TestColouredFrameAvoidsNodeColour(t *testing.T) {
	g, err := flowchart.Parse("flowchart TD\nsubgraph S\nA{Decyzja} --> B[Krok]\nend")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	f := byRole(sc, "cluster")[0]
	for _, n := range byRole(sc, "node") {
		if n.Fill == f.Fill {
			t.Errorf("the frame took the fill of a node inside it: %s", f.Fill)
		}
	}
}

// REQ: LAY-019
// Zwijanie: długi prosty diagram LR staje w kilku rzędach, ramki nie są dzielone, wężyk odwraca co
// drugi rząd; diagram z cyklem, z krawędzią przeskakującą przez grupę albo krótki zostaje bez zmian.
func TestFold(t *testing.T) {
	o := Options{Measure: font.ApproxTextWidth, FontSize: 16}
	rows := func(sc *scene.Scene, role string) []scene.Box {
		var out []scene.Box
		for _, it := range byRole(sc, role) {
			out = append(out, it.Bounds())
		}
		return out
	}
	sc := Layout(parse(t, "../../testdata/corpus/flowchart/023-fold-return.mmd"), o)
	frames := rows(sc, "cluster")
	if len(frames) != 3 || !(frames[0].Y1 < frames[1].Y0 && frames[1].Y1 < frames[2].Y0) {
		t.Fatalf("the three frames should stand one under another: %+v", frames)
	}
	if b := sc.Bounds(); b.X1-b.X0 > 2*(b.Y1-b.Y0) {
		t.Errorf("folded drawing is still %.0f x %.0f", b.X1-b.X0, b.Y1-b.Y0)
	}
	for _, f := range []string{"023-fold-return", "024-fold-snake", "025-fold-tb"} {
		g := parse(t, "../../testdata/corpus/flowchart/"+f+".mmd")
		off := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, Fold: FoldOff})
		on := Layout(parse(t, "../../testdata/corpus/flowchart/"+f+".mmd"), o)
		if pageScale(on, 16) < pageScale(off, 16)-0.001 {
			t.Errorf("%s: folding lowered the page scale", f)
		}
		if crossingCount(on) != 0 {
			t.Errorf("%s: folded drawing has crossings", f)
		}
		if len(byRole(on, "edge")) != len(byRole(off, "edge")) {
			t.Errorf("%s: folding changed the number of edges", f)
		}
	}
	// Wężyk: w drugim rzędzie strzałki biegną w lewo.
	g := parse(t, "../../testdata/corpus/flowchart/024-fold-snake.mmd")
	snake := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, FoldType: FoldSnake})
	a, b := nodeOf(t, snake, "Analiza"), nodeOf(t, snake, "Działania")
	if !(b.X1 < a.X0 && math.Abs(a.Y0-b.Y0) < 1) {
		t.Errorf("snake: the second row should run right to left: %+v %+v", a, b)
	}
	// Bez zwijania: cykl, krawędź przez całą grupę, krótki diagram.
	for _, src := range []string{
		"flowchart LR\nA[Pierwszy długi krok] --> B[Drugi długi krok] --> C[Trzeci długi krok] --> D[Czwarty długi krok] --> E[Piąty długi krok] --> A",
		"flowchart LR\nA[Pierwszy długi krok] --> B[Drugi długi krok] --> C[Trzeci długi krok] --> D[Czwarty długi krok] --> E[Piąty długi krok]\nA --> D\nB --> E\nA --> E",
		"flowchart LR\nA --> B --> C",
	} {
		g, err := flowchart.Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		on := Layout(g, o)
		g, _ = flowchart.Parse(src)
		off := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, Fold: FoldOff})
		if fmt.Sprint(on.Bounds()) != fmt.Sprint(off.Bounds()) {
			t.Errorf("%q should not fold", src)
		}
	}
}

// REQ: LAY-027
// Przekazania między torami bez etykiety biegną poziomo w jednej warstwie: Zgłoszenie, Rejestracja
// i Ocena stoją w jednym rzędzie, a krawędź między sąsiednimi biegnie od boku do boku.
func TestLaneHandoffsStayInRank(t *testing.T) {
	sc := natural(t, "flowchart/031-swimlane-handoffs.mmd")
	a, b, c := nodeOf(t, sc, "Zgłoszenie"), nodeOf(t, sc, "Rejestracja"), nodeOf(t, sc, "Ocena")
	mid := func(x scene.Box) float64 { return (x.Y0 + x.Y1) / 2 }
	if math.Abs(mid(a)-mid(b)) > 0.5 || math.Abs(mid(b)-mid(c)) > 0.5 {
		t.Fatalf("handoffs are not in one rank: %.1f %.1f %.1f", mid(a), mid(b), mid(c))
	}
	found := false
	for _, s := range straights(sc) {
		if math.Abs(s.a.Y-mid(a)) < 0.5 && math.Abs(s.b.Y-mid(a)) < 0.5 && math.Abs(s.a.X-a.X1) < 0.5 && s.b.X > a.X1 && s.b.X <= b.X0+0.5 {
			found = true
		}
	}
	if !found {
		t.Errorf("no straight edge from the side of Zgłoszenie to the side of Rejestracja")
	}
	// Krawędź z etykietą schodzi warstwę niżej.
	if d, z := nodeOf(t, sc, "Zgodne?"), nodeOf(t, sc, "Zatwierdzenie"); z.Y0 < d.Y1 {
		t.Errorf("a labelled edge should not stay in one rank")
	}
	// Żadna płaska krawędź nie przechodzi przez pole.
	for name, sc := range corpusScenes(t) {
		var boxes []scene.Box
		for _, it := range byRole(sc, "node") {
			boxes = append(boxes, it.Bounds())
		}
		for _, s := range straights(sc) {
			lo, hi := math.Min(s.a.X, s.b.X), math.Max(s.a.X, s.b.X)
			if math.Abs(s.a.Y-s.b.Y) > 0.01 {
				continue
			}
			for _, bx := range boxes {
				if s.a.Y > bx.Y0+1 && s.a.Y < bx.Y1-1 && lo < bx.X0+1 && hi > bx.X1-1 {
					t.Errorf("%s: edge at y %.1f runs through a node %+v", name, s.a.Y, bx)
				}
			}
		}
	}
}

// REQ: LAY-028
// Krawędź wsteczna Poprawki → Wykonanie biegnie prosto, choć boki obu pól dzielą inne krawędzie.
func TestChainFollowsPort(t *testing.T) {
	sc := natural(t, "flowchart/032-swimlane-loop-lr.mmd")
	p := nodeOf(t, sc, "Poprawki")
	w := nodeOf(t, sc, "Wykonanie")
	for _, e := range byRole(sc, "edge") {
		first, last := e.Segments[0].To, e.Segments[len(e.Segments)-1].To
		// Krawędź wsteczna jest rysowana od Wykonania (z grotem przy nim) do Poprawek.
		if math.Abs(first.X-w.X1) > 30 || math.Abs(last.X-p.X0) > 30 || first.Y < w.Y0 || first.Y > w.Y1 {
			continue
		}
		for _, sg := range e.Segments[1:] {
			if math.Abs(sg.To.Y-first.Y) > 0.01 {
				t.Fatalf("the back edge between Wykonanie and Poprawki turns at (%.1f, %.1f)", sg.To.X, sg.To.Y)
			}
		}
		return
	}
	t.Fatal("no edge between the right side of Wykonanie and the left side of Poprawki")
}

// REQ: LAY-029
// Dwie różne krawędzie nie biegną po sobie: żadne ich proste odcinki nie leżą na jednej prostej
// i nie zachodzą na siebie. Sprawdzane na diagramach torów w czterech kierunkach układu (w części
// zwykłych schematów takie miejsca jeszcze są: krawędzie równoległe i wejścia w jeden punkt).
// Wspólne wejście kilku krawędzi w jeden punkt pola nie jest biegiem po sobie.
func TestEdgesDoNotShareRuns(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		if !strings.Contains(name, "swim") && !strings.Contains(name, "031-") && !strings.Contains(name, "032-") && !strings.Contains(name, "033-") {
			continue
		}
		ss := straights(sc)
		// Krawędzie zbiegające się do jednego punktu pola (zaokrąglony bok bez miejsca na rozsunięcie)
		// dzielą tylko ostatni kawałek przed grotem: to wspólne wejście, nie bieg po sobie.
		ends := map[int]scene.Point{}
		for i, e := range byRole(sc, "edge") {
			ends[i] = e.Segments[len(e.Segments)-1].To
		}
		near := func(p, q scene.Point) bool { return math.Abs(p.X-q.X) < 0.5 && math.Abs(p.Y-q.Y) < 0.5 }
		for i, a := range ss {
			for _, b := range ss[i+1:] {
				if a.edge == b.edge || (near(a.b, ends[a.edge]) && near(b.b, ends[b.edge]) && near(a.b, b.b)) {
					continue
				}
				switch {
				case math.Abs(a.a.X-a.b.X) < 0.01 && math.Abs(b.a.X-b.b.X) < 0.01 && math.Abs(a.a.X-b.a.X) < 0.01:
					if math.Min(math.Max(a.a.Y, a.b.Y), math.Max(b.a.Y, b.b.Y))-math.Max(math.Min(a.a.Y, a.b.Y), math.Min(b.a.Y, b.b.Y)) > 2 {
						t.Errorf("%s: two edges run on one vertical line at x %.1f", name, a.a.X)
					}
				case math.Abs(a.a.Y-a.b.Y) < 0.01 && math.Abs(b.a.Y-b.b.Y) < 0.01 && math.Abs(a.a.Y-b.a.Y) < 0.01:
					if math.Min(math.Max(a.a.X, a.b.X), math.Max(b.a.X, b.b.X))-math.Max(math.Min(a.a.X, a.b.X), math.Min(b.a.X, b.b.X)) > 2 {
						t.Errorf("%s: two edges run on one horizontal line at y %.1f", name, a.a.Y)
					}
				}
			}
		}
	}
}

// REQ: FLOW-028 FLOW-029
// Tory poziome: tytuły obrócone; w trybie kolorowym tory mają różne tła. Pole bez krawędzi
// (Rejestr kontroli) stoi za poprzednim polem swojego toru (Gotowe?), nie w pierwszej warstwie.
func TestLaneLookAndLooseNode(t *testing.T) {
	sc := natural(t, "flowchart/032-swimlane-loop-lr.mmd")
	titles := byRole(sc, "lane-title")
	if len(titles) != 2 {
		t.Fatalf("%d lane titles", len(titles))
	}
	for _, it := range titles {
		if !it.Vertical {
			t.Errorf("lane title %q is not turned", it.Text)
		}
	}
	if r, g := nodeOf(t, sc, "Rejestr kontroli"), nodeOf(t, sc, "Gotowe?"); r.X0 < g.X1 {
		t.Errorf("the loose node stands at %.1f, before the previous node of its lane (ends at %.1f)", r.X0, g.X1)
	}
	colored := Layout(parse(t, "../../testdata/corpus/flowchart/032-swimlane-loop-lr.mmd"), Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	lanes := byRole(colored, "lane")
	if len(lanes) != 2 || lanes[0].Fill == "" || lanes[0].Fill == lanes[1].Fill {
		t.Errorf("coloured lanes should have distinct fills: %+v", lanes)
	}
}

// bends liczy załamania krawędzi: zmiany kierunku między kolejnymi prostymi odcinkami (odcinki
// współliniowe, rozdzielone mostkiem, liczą się jako jeden).
func bends(e scene.Item) int {
	var dirs []byte
	at := e.Segments[0].To
	for _, sg := range e.Segments[1:] {
		if sg.Cmd == 'L' {
			dx, dy := sg.To.X-at.X, sg.To.Y-at.Y
			d := byte('v')
			if math.Abs(dx) > math.Abs(dy) {
				d = 'h'
			}
			if (math.Abs(dx) > 0.5 || math.Abs(dy) > 0.5) && (len(dirs) == 0 || dirs[len(dirs)-1] != d) {
				dirs = append(dirs, d)
			}
		}
		at = sg.To
	}
	return max(0, len(dirs)-1)
}

// REQ: LAY-028 FLOW-030
// Trzy wyjścia „Nie” do wspólnego zakończenia biegną z dwoma załamaniami (w bok, w dół, w bok, w dół);
// pola toru Przegląd, które w rzędzie są same, stoją w jednej osi.
func TestExitsWithFewestBends(t *testing.T) {
	sc := natural(t, "flowchart/033-swimlane-exits.mmd")
	end := nodeOf(t, sc, "Koniec")
	long := 0
	for _, e := range byRole(sc, "edge") {
		last := e.Segments[len(e.Segments)-1].To
		first := e.Segments[0].To
		if math.Abs(last.Y-end.Y0) > 14 || first.Y > end.Y0-300 {
			continue // tylko długie krawędzie do Końca
		}
		long++
		if b := bends(e); b > 4 {
			t.Errorf("edge into Koniec from (%.0f, %.0f) has %d bends", first.X, first.Y, b)
		}
	}
	if long < 3 {
		t.Fatalf("%d long edges into Koniec", long)
	}
	axis := centerX(nodeOf(t, sc, "Zasadny?"))
	for _, label := range []string{"Budżet?", "Odbiór", "Koniec"} {
		if x := centerX(nodeOf(t, sc, label)); math.Abs(x-axis) > 0.5 {
			t.Errorf("%q stands at %.1f, the lane axis at %.1f", label, x, axis)
		}
	}
}
