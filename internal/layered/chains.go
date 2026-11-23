package layered

import (
	"math"
	"sort"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Zwijanie łańcuchów (LAY-022): długi łańcuch pól wewnątrz diagramu, połączonych jeden do jednego,
// staje się niewidocznym blokiem, w którym pola stoją w kilku kolumnach (TB) albo rzędach (LR).
// Wejście do bloku trafia do pierwszego pola, wyjście biegnie z ostatniego, przez zwykłe punkty
// na ramce podgrafu; reszta układu traktuje blok jak każdy podgraf.

// minChain to najkrótszy łańcuch, który warto zwijać.
const minChain = 6

// findChains zwraca łańcuchy co najmniej minChain pól, od najdłuższego. Ogniwo łańcucha to krawędź
// bez podpisu, z prostym grotem, między dwoma węzłami o wspólnym rodzicu, jedyna wychodząca
// z początku i jedyna wchodząca do końca, biegnąca naprzód w kolejności pól ze źródła (krawędź
// wsteczna zamyka cykl i w układzie biegnie pod prąd). Pole na początku z kilkoma wejściami albo
// wejściem wstecznym i pole na końcu z kilkoma wyjściami albo wyjściem wstecznym odpadają: blok ma
// jedno wejście od góry i jedno wyjście u dołu.
// Pola z notatkami i pętlami własnymi nie wchodzą do łańcucha; łańcuch, którego pola łączy jeszcze
// inna krawędź, jest pomijany.
func findChains(g *graph.Graph) [][]string {
	index := map[string]int{}
	for i, n := range g.Nodes {
		index[n.ID] = i
	}
	back := func(e *graph.Edge) bool {
		a, okA := index[e.From]
		b, okB := index[e.To]
		return okA && okB && b <= a
	}
	in, out := map[string][]*graph.Edge{}, map[string][]*graph.Edge{}
	noted := map[string]bool{}
	for _, e := range g.Edges {
		if e.From == e.To {
			noted[e.From] = true // pętla własna poszerza pole; blok jej nie rysuje
			continue
		}
		out[e.From] = append(out[e.From], e)
		in[e.To] = append(in[e.To], e)
	}
	for _, n := range g.Notes {
		noted[n.Target] = true
	}
	next, prev := map[string]string{}, map[string]string{}
	for _, e := range g.Edges {
		a, b := g.Node(e.From), g.Node(e.To)
		if a == nil || b == nil || e.From == e.To || a.Parent != b.Parent || noted[e.From] || noted[e.To] || back(e) ||
			len(e.Label) > 0 || e.Style == graph.Invisible || !plainHead(e.StartHead) || e.StartHead == graph.ArrowHead ||
			!plainHead(e.EndHead) || len(out[e.From]) != 1 || len(in[e.To]) != 1 {
			continue
		}
		next[e.From], prev[e.To] = e.To, e.From
	}
	anyBack := func(es []*graph.Edge) bool {
		for _, e := range es {
			if back(e) {
				return true
			}
		}
		return false
	}
	var chains [][]string
	for _, n := range g.Nodes {
		if _, ok := next[n.ID]; !ok {
			continue
		}
		if _, ok := prev[n.ID]; ok {
			continue
		}
		ids := []string{n.ID}
		for id := next[n.ID]; id != ""; id = next[id] {
			ids = append(ids, id)
		}
		// Blok ma jedno wejście i jedno wyjście: rozgałęzienie na końcu (np. decyzja) zostaje poza nim.
		for len(ids) > 0 && (anyBack(in[ids[0]]) || len(in[ids[0]]) > 1) {
			ids = ids[1:]
		}
		for len(ids) > 0 && (anyBack(out[ids[len(ids)-1]]) || len(out[ids[len(ids)-1]]) > 1) {
			ids = ids[:len(ids)-1]
		}
		if len(ids) < minChain {
			continue
		}
		member := map[string]bool{}
		for _, id := range ids {
			member[id] = true
		}
		links := 0
		for _, e := range g.Edges {
			if member[e.From] && member[e.To] {
				links++
			}
		}
		if links == len(ids)-1 {
			chains = append(chains, ids)
		}
	}
	sort.SliceStable(chains, func(a, b int) bool { return len(chains[a]) > len(chains[b]) })
	return chains
}

// foldChains zwija łańcuchy, które zwiększają skalę rysunku na stronie A4 co najmniej o 20%,
// i zwraca graf z blokami łańcuchów (albo graf bez zmian).
func foldChains(g *graph.Graph, o Options) *graph.Graph {
	if g.Layout != "" || (g.Kind != "flowchart" && g.Kind != "state") {
		return g
	}
	chains := findChains(g)
	if len(chains) == 0 {
		return g
	}
	o.Fit, o.Fold = FitOff, FoldOff
	current := g
	scale := pageScale(layout(g, o, 0, 1), o.FontSize)
	for _, ids := range chains {
		counts := []int{2, 3, 4}
		snake := o.FoldType == FoldSnake
		if snake {
			counts = []int{3, 5} // nieparzyście: ostatnia kolumna biegnie w dół i łańcuch wychodzi dołem
		}
		var best *graph.Graph
		bestScale := scale * 1.2
		for _, k := range counts {
			if len(ids)/k < 2 {
				continue
			}
			need := bestScale
			if best != nil {
				need = bestScale * 1.1 // więcej kolumn to więcej łączników: musi dać wyraźnie więcej
			}
			candidate := current.WithChain(ids, k, snake)
			bad := false
			try := o
			try.chainBad = &bad
			s := pageScale(layout(candidate, try, 0, 1), o.FontSize)
			if !bad && s >= need-1e-9 {
				best, bestScale = candidate, s
			}
		}
		if best != nil {
			current, scale = best, bestScale
		}
	}
	return current
}

// chainBlock mówi, czy poziom jest blokiem zwiniętego łańcucha.
func chainBlock(lv *level) bool { return lv.sub != nil && lv.sub.Chain > 0 }

// chainNodes zwraca pola bloku w kolejności łańcucha.
func chainNodes(lv *level) []*item {
	next := map[*item]*item{}
	target := map[*item]bool{}
	for _, e := range lv.edges {
		if e.u.kind == kindNode && e.v.kind == kindNode {
			a, b := e.u, e.v
			if e.rev {
				a, b = b, a
			}
			next[a], target[b] = b, true
		}
	}
	var first *item
	for _, it := range lv.items {
		if it.kind == kindNode && !target[it] {
			first = it
			break
		}
	}
	var out []*item
	for it := first; it != nil && len(out) <= len(lv.items); it = next[it] {
		out = append(out, it)
	}
	return out
}

// chainGeometry to układ bloku: kolumny pól, ich położenia i wysokości przejść łączników.
type chainGeometry struct {
	columns [][]*item
	colX    []float64 // lewa krawędź kolumny
	colW    []float64
	gapX    float64
	up      []bool // kolumna biegnie pod prąd (wężyk)
}

func (l *layouter) chainGeometry(lv *level) chainGeometry {
	nodes := chainNodes(lv)
	k := lv.sub.Chain
	var g chainGeometry
	// Po równo: kolumny różnią się najwyżej o jedno pole, dłuższe są na końcu, żeby wyjście z bloku
	// wychodziło spod ostatniego wiersza, a nie biegło przez pusty.
	for i, at := 0, 0; i < k && at < len(nodes); i++ {
		size := len(nodes) / k
		if i >= k-len(nodes)%k {
			size++
		}
		g.columns = append(g.columns, nodes[at:at+size])
		at += size
	}
	g.gapX = l.u(48)
	x := 0.0
	for i, col := range g.columns {
		w := 0.0
		for _, it := range col {
			w = math.Max(w, it.w)
		}
		// Pola kolumny mają jedną szerokość, jak pola zwykłego łańcucha.
		for _, it := range col {
			it.w = w
		}
		g.colX, g.colW = append(g.colX, x), append(g.colW, w)
		g.up = append(g.up, lv.sub.Snake && i%2 == 1)
		x += w + g.gapX
	}
	return g
}

// chainPositions rozmieszcza wnętrze bloku łańcucha w kolumnach i zwraca jego rozmiar.
func (l *layouter) chainPositions(lv *level) (width, height float64) {
	g := l.chainGeometry(lv)
	per := 0
	for _, col := range g.columns {
		per = max(per, len(col))
	}
	rowH := make([]float64, per)
	for _, col := range g.columns {
		for r, it := range col {
			rowH[r] = math.Max(rowH[r], it.h)
		}
	}
	margin, gapY := l.u(36), l.u(44)
	y := margin
	rowY := make([]float64, per)
	for r := range rowH {
		rowY[r] = y + rowH[r]/2
		y += rowH[r] + gapY
	}
	y -= gapY
	height = y + margin
	for i, col := range g.columns {
		for r, it := range col {
			row := r
			if g.up[i] {
				row = per - 1 - r
			}
			it.x, it.y, it.band = g.colX[i]+g.colW[i]/2, rowY[row], it.h
		}
	}
	width = g.colX[len(g.colX)-1] + g.colW[len(g.colW)-1]
	// Punkty na ramce: przy polu, z którym łączą się w środku bloku; wejście tylko do pierwszego
	// pola, wyjście tylko z ostatniego, inaczej blok byłby przecinany (chainBad).
	nodes := chainNodes(lv)
	first, last := nodes[0], nodes[len(nodes)-1]
	for _, e := range lv.edges {
		var port, inner *item
		switch {
		case e.u.kind == kindPort:
			port, inner = e.u, e.v
		case e.v.kind == kindPort:
			port, inner = e.v, e.u
		default:
			continue
		}
		want := last
		if port.portTop {
			want = first
		}
		if inner != want && l.o.chainBad != nil {
			*l.o.chainBad = true
		}
		port.x, port.band = inner.x, 0
		port.y = 0
		if !port.portTop {
			port.y = height
		}
	}
	return width, height
}

// drawChain rysuje krawędzie wewnątrz bloku łańcucha: proste w kolumnie i łączniki między kolumnami.
func (l *layouter) drawChain(lv *level) {
	g := l.chainGeometry(lv)
	// Wysokości łączników liczone tak samo jak w chainPositions.
	top, bottom := math.Inf(1), math.Inf(-1)
	for _, col := range g.columns {
		for _, it := range col {
			top, bottom = math.Min(top, it.y-it.h/2), math.Max(bottom, it.y+it.h/2)
		}
	}
	high, low := top-l.u(36)/2, bottom+l.u(36)/2 // w połowie marginesu bloku (chainPositions)
	edge := map[[2]*item]*graph.Edge{}
	for _, e := range lv.edges {
		if e.u.kind == kindNode && e.v.kind == kindNode {
			a, b := e.u, e.v
			if e.rev {
				a, b = b, a
			}
			edge[[2]*item{a, b}] = e.e
		}
	}
	var nodes []*item
	column := map[*item]int{}
	for i, col := range g.columns {
		for _, it := range col {
			nodes = append(nodes, it)
			column[it] = i
		}
	}
	for i := 0; i+1 < len(nodes); i++ {
		a, b := nodes[i], nodes[i+1]
		e := edge[[2]*item{a, b}]
		if e == nil {
			continue
		}
		ca, cb := column[a], column[b]
		var pts [][2]float64
		switch {
		case ca == cb && !g.up[ca]:
			pts = [][2]float64{{a.x, a.y + a.h/2}, {b.x, b.y - b.h/2}}
		case ca == cb:
			pts = [][2]float64{{a.x, a.y - a.h/2}, {b.x, b.y + b.h/2}}
		case !g.up[ca] && !g.up[cb]:
			// Powrót: spod ostatniego pola kolumny do odstępu między kolumnami, w górę i do pierwszego
			// pola następnej kolumny od góry.
			// Położenia pól są już bezwzględne (po place), g.colX względne: odstęp liczony z pól.
			right, left := math.Inf(-1), math.Inf(1)
			for _, it := range g.columns[ca] {
				right = math.Max(right, it.x+it.w/2)
			}
			for _, it := range g.columns[cb] {
				left = math.Min(left, it.x-it.w/2)
			}
			gx := (right + left) / 2
			pts = [][2]float64{{a.x, a.y + a.h/2}, {a.x, low}, {gx, low}, {gx, high}, {b.x, high}, {b.x, b.y - b.h/2}}
		case !g.up[ca]:
			pts = [][2]float64{{a.x, a.y + a.h/2}, {a.x, low}, {b.x, low}, {b.x, b.y + b.h/2}}
		default:
			pts = [][2]float64{{a.x, a.y - a.h/2}, {a.x, high}, {b.x, high}, {b.x, b.y - b.h/2}}
		}
		real := make([]scene.Point, len(pts))
		for k, p := range pts {
			real[k] = l.point(p[0], p[1])
		}
		drawLink(l.sc, e, real, l.u)
	}
}
