package layered

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// lane to tor: pas biegnący wzdłuż osi warstw przez cały rysunek. Kroki procesu stoją w pasie
// swojej roli, a warstwy są wspólne dla wszystkich pasów, więc położenie wzdłuż osi warstw mówi
// o kolejności w całym procesie.
type lane struct {
	title  []string
	lo, hi float64 // zakres pasa w poprzek osi warstw
}

// flattenLanes zamienia graf z podgrafami-torami na graf płaski i listę torów. Węzły spoza
// podgrafów trafiają do toru bez tytułu, stojącego jako pierwszy.
func flattenLanes(g *graph.Graph) (*graph.Graph, []*lane, map[string]int) {
	flat := graph.New(g.Kind)
	flat.Direction, flat.AccTitle, flat.AccDescr = g.Direction, g.AccTitle, g.AccDescr
	index := map[string]int{}
	var lanes []*lane
	free := false
	for _, n := range g.Nodes {
		free = free || n.Parent == ""
	}
	if free {
		lanes = append(lanes, &lane{})
	}
	for _, s := range g.Subgraphs {
		index[s.ID] = len(lanes)
		lanes = append(lanes, &lane{title: s.Title})
	}
	of := map[string]int{}
	for _, n := range g.Nodes {
		c, _ := flat.EnsureNode(n.ID, "", n.Shape, n.Line)
		*c = *n
		c.Parent = ""
		of[n.ID] = index[n.Parent] // węzeł bez podgrafu: tor zerowy
	}
	for _, e := range g.Edges {
		c := *e
		flat.AddEdge(&c)
	}
	return flat, lanes, of
}

// laneExtents wyznacza szerokości torów: każdy mieści najszerszy rząd swoich pól i swój tytuł.
func (l *layouter) laneExtents(lv *level, sep func(a, b *item) float64) {
	pad := l.u(28)
	width := make([]float64, len(l.lanes))
	for k, ln := range l.lanes {
		tw, _ := l.textSize(ln.title) // w torze poziomym tytuł jest obrócony: jego szerokość biegnie w poprzek
		least := l.u(190)
		if !l.vertical {
			least = l.u(96) // pas poziomy może być niski: mieści jeden rząd pól
		}
		width[k] = math.Max(least, tw+2*l.u(16))
	}
	for _, row := range lv.ranks {
		need := make([]float64, len(l.lanes))
		var prev *item
		for _, it := range row {
			if prev != nil && prev.lane == it.lane {
				need[it.lane] += sep(prev, it) - prev.w/2 - it.w/2
			}
			need[it.lane] += it.w
			prev = it
		}
		for k := range need {
			width[k] = math.Max(width[k], need[k]+2*pad)
		}
	}
	x := 0.0
	for k, ln := range l.lanes {
		ln.lo, ln.hi = x, x+width[k]
		x = ln.hi
	}
}

// confine dosuwa pola rzędu do zakresu ich torów, zachowując kolejność i odstępy.
func (l *layouter) confine(row []*item, sep func(a, b *item) float64) {
	pad := l.u(28)
	for i, it := range row {
		lo := l.lanes[it.lane].lo + pad + it.w/2
		if i > 0 && row[i-1].lane == it.lane {
			lo = math.Max(lo, row[i-1].x+sep(row[i-1], it))
		}
		it.x = math.Max(it.x, lo)
	}
	for i := len(row) - 1; i >= 0; i-- {
		it := row[i]
		hi := l.lanes[it.lane].hi - pad - it.w/2
		if i+1 < len(row) && row[i+1].lane == it.lane {
			hi = math.Min(hi, row[i+1].x-sep(it, row[i+1]))
		}
		it.x = math.Min(it.x, hi)
	}
}

// groupByLane ustawia pola każdego rzędu torami, zachowując kolejność wewnątrz toru.
func (l *layouter) groupByLane(lv *level) {
	// Koniec płaskiej krawędzi (LAY-027) stoi w swoim torze od strony drugiego końca, żeby krawędź
	// nie biegła przez sąsiednie pole tego samego toru.
	side := map[*item]int{}
	for _, e := range lv.edges {
		if !e.flat {
			continue
		}
		if e.v.lane > e.u.lane {
			side[e.u]++
			side[e.v]--
		} else {
			side[e.u]--
			side[e.v]++
		}
	}
	sign := func(it *item) int { return min(1, max(-1, side[it])) }
	for _, row := range lv.ranks {
		sort.SliceStable(row, func(a, b int) bool {
			if row[a].lane != row[b].lane {
				return row[a].lane < row[b].lane
			}
			return sign(row[a]) < sign(row[b])
		})
		for i, it := range row {
			it.order = i
		}
	}
	// Grupowanie przestawia pola względem kolejności z porządkowania i może dodać przecięcia.
	// Sąsiedzi z jednego toru zamieniają się miejscami, jeśli zmniejsza to liczbę przecięć i nie
	// odsuwa końca płaskiej krawędzi od jej strony.
	cross := func(a, b []*item) int {
		n := 0
		for _, x := range a {
			for _, y := range b {
				if x.order > y.order {
					n++
				}
			}
		}
		return n
	}
	for round := 0; round < 8; round++ {
		improved := false
		for _, row := range lv.ranks {
			for i := 0; i+1 < len(row); i++ {
				a, b := row[i], row[i+1]
				if a.lane != b.lane || sign(a) != sign(b) {
					continue
				}
				if cross(b.up, a.up)+cross(b.down, a.down) < cross(a.up, b.up)+cross(a.down, b.down) {
					row[i], row[i+1] = b, a
					a.order, b.order = i+1, i
					improved = true
				}
			}
		}
		if !improved {
			break
		}
	}
}

// drawLanes rysuje pasy torów z nagłówkami przy początku osi warstw. W torach poziomych (LR, RL)
// tytuł stoi w nagłówku z lewej strony obrócony o ćwierć obrotu, jak na diagramach torów; w trybie
// kolorowym tory mają na przemian tony palety: nagłówek w tonie, pas w jaśniejszym.
func (l *layouter) drawLanes() {
	root := l.levels[""]
	header, gap := 0.0, l.u(18)
	for _, ln := range l.lanes {
		_, th := l.textSize(ln.title)
		if l.vertical {
			header = math.Max(header, th+l.u(14))
		} else {
			header = math.Max(header, th+l.u(16))
		}
	}
	top, bottom := -(header + gap), root.h+gap
	for k, ln := range l.lanes {
		fill, headFill := "", colorNodeFill
		if l.o.Colors {
			t := scene.Distinct(k)
			fill, headFill = lighter(t.Fill), t.Fill
		}
		cx, cy, w, h := l.rect((ln.lo+ln.hi)/2, (top+bottom)/2, ln.hi-ln.lo, bottom-top)
		band := scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "lane", X: cx - w/2, Y: cy - h/2, W: w, H: h, Fill: fill, Stroke: colorFrame, StrokeWidth: 1}
		l.sc.Add(band)
		cx, cy, w, h = l.rect((ln.lo+ln.hi)/2, top+header/2, ln.hi-ln.lo, header)
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "lane-header", X: cx - w/2, Y: cy - h/2, W: w, H: h,
			Fill: headFill, Stroke: colorFrame, StrokeWidth: 1})
		if l.vertical {
			l.textBlock(ln.title, cx, cy, "lane-title")
			continue
		}
		// Tytuł obrócony, czytany z dołu do góry: linie stoją obok siebie, pierwsza z lewej.
		left := cx - float64(len(ln.title))*l.lh/2
		for i, s := range ln.title {
			if s == "" {
				continue
			}
			l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "lane-title", X: left + float64(i)*l.lh + l.lh/2 + 0.35*l.fs,
				Y: cy, W: l.width(s), H: l.lh, Text: s, Anchor: "middle", Vertical: true, Fill: colorText})
		}
	}
}

// lighter rozjaśnia kolor #rrggbb o połowę drogi do bieli (tło pasa toru w trybie kolorowym).
func lighter(c string) string {
	if len(c) != 7 || c[0] != '#' {
		return c
	}
	var rgb [3]int64
	for i := range rgb {
		v, err := strconv.ParseInt(c[1+2*i:3+2*i], 16, 0)
		if err != nil {
			return c
		}
		rgb[i] = v + (255-v)/2
	}
	return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
}

// flatConflicts zwraca płaskie krawędzie (LAY-027), które trzeba wydłużyć: przechodzące przez
// tor, w którym w tej warstwie stoi pole, albo biegnące po wcześniejszej płaskiej krawędzi tej
// samej warstwy. Krawędzie są brane w kolejności źródła, więc pierwsza z kolidujących zostaje.
func (l *layouter) flatConflicts(lv *level, rank []int, index map[*item]int, zero map[*ledge]bool) []*ledge {
	type flat struct {
		e       *ledge
		r, a, b int // warstwa i zakres torów
	}
	var flats []flat
	for _, e := range lv.edges {
		if !zero[e] || rank[index[e.u]] != rank[index[e.v]] {
			continue
		}
		a, b := e.u.lane, e.v.lane
		if a > b {
			a, b = b, a
		}
		flats = append(flats, flat{e, rank[index[e.u]], a, b})
	}
	var bad []*ledge
	kept := []flat{}
	for _, f := range flats {
		blocked := false
		for i, it := range lv.items {
			if rank[i] == f.r && it.real() && it.lane > f.a && it.lane < f.b {
				blocked = true
			}
		}
		for _, g := range kept {
			if g.r == f.r && max(f.a, g.a) < min(f.b, g.b) {
				blocked = true
			}
		}
		if blocked {
			bad = append(bad, f.e)
		} else {
			kept = append(kept, f)
		}
	}
	return bad
}

// placeLoose stawia pole bez krawędzi w warstwie za poprzednim (w kolejności źródła) polem jego
// toru, a nie w pierwszej warstwie: kolejność kroków w torze zostaje taka jak w źródle.
func (l *layouter) placeLoose(lv *level, rank []int, index map[*item]int) {
	linked := map[*item]bool{}
	for _, e := range lv.edges {
		linked[e.u], linked[e.v] = true, true
	}
	last := map[int]int{} // tor → indeks ostatniego pola tego toru w kolejności źródła
	for i, it := range lv.items {
		if !it.real() {
			continue
		}
		if p, ok := last[it.lane]; ok && !linked[it] {
			rank[i] = rank[p] + 1
		}
		last[it.lane] = i
	}
}
