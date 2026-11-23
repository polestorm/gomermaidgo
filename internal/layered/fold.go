package layered

import (
	"math"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Zwijanie długiego diagramu. Diagram dzieli się na kolejne grupy warstw najwyższego poziomu; każda
// grupa jest rozmieszczana osobno, a grupy stają jedna pod drugą (LR) albo obok siebie (TB). Krawędzie
// między grupami rysują łączniki: przy wężyku co druga grupa biegnie w przeciwnym kierunku, a łącznik
// skręca przy końcu rzędu; przy powrocie wszystkie grupy biegną w jednym kierunku, a łącznik wraca
// w odstępie między rzędami na początek następnego.
//
// Zwijany jest tylko diagram prosty: warstwy najwyższego poziomu tworzą graf bez cykli, a między
// sąsiednimi grupami biegną najwyżej dwie krawędzie, każda z ostatniej warstwy grupy do pierwszej
// warstwy następnej, zakończona grotem albo bez zakończenia. Ramki podgrafów nie są dzielone.

const maxFoldLinks = 2

// foldUnit zwraca jednostkę najwyższego poziomu, do której należy węzeł albo podgraf id.
func foldUnit(g *graph.Graph, id string) string {
	parent := ""
	if n := g.Node(id); n != nil {
		parent = n.Parent
	} else if s := g.Subgraph(id); s != nil {
		parent = s.Parent
	} else {
		return id
	}
	for parent != "" {
		id = parent
		s := g.Subgraph(parent)
		if s == nil {
			break
		}
		parent = s.Parent
	}
	return id
}

// foldRanks numeruje jednostki najwyższego poziomu najdłuższą drogą; nil, gdy jest cykl.
func foldRanks(g *graph.Graph) (map[string]int, [][2]string) {
	rank := map[string]int{}
	var units []string
	add := func(u string) {
		if _, ok := rank[u]; !ok {
			rank[u] = 0
			units = append(units, u)
		}
	}
	for _, n := range g.Nodes {
		add(foldUnit(g, n.ID))
	}
	var links [][2]string
	indeg := map[string]int{}
	succ := map[string][]string{}
	for _, e := range g.Edges {
		a, b := foldUnit(g, e.From), foldUnit(g, e.To)
		if a == b {
			continue
		}
		add(a)
		add(b)
		links = append(links, [2]string{a, b})
		succ[a] = append(succ[a], b)
		indeg[b]++
	}
	var queue []string
	for _, u := range units {
		if indeg[u] == 0 {
			queue = append(queue, u)
		}
	}
	seen := 0
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		seen++
		for _, v := range succ[u] {
			rank[v] = max(rank[v], rank[u]+1)
			if indeg[v]--; indeg[v] == 0 {
				queue = append(queue, v)
			}
		}
	}
	if seen != len(units) {
		return nil, nil
	}
	// Tylko diagram spójny: niezależne części cięte razem przeplatałyby łączniki.
	parent := map[string]string{}
	var find func(x string) string
	find = func(x string) string {
		if p, ok := parent[x]; ok && p != x {
			r := find(p)
			parent[x] = r
			return r
		}
		parent[x] = x
		return x
	}
	for _, l := range links {
		parent[find(l[0])] = find(l[1])
	}
	for _, u := range units {
		if find(u) != find(units[0]) {
			return nil, nil
		}
	}
	return rank, links
}

// fold zwija diagram, jeśli jest dość długi i prosty, a zwinięty mieści się na stronie w wyraźnie
// większej skali; inaczej zwraca nil.
func fold(g *graph.Graph, o Options, base *scene.Scene, fit float64) *scene.Scene {
	if o.Fold == FoldOff || g.Layout == "swimlanes" || (g.Direction != graph.LeftRight && g.Direction != graph.TopBottom) {
		return nil
	}
	horizontal := g.Direction == graph.LeftRight
	ratio := o.FitRatio
	if ratio <= 0 {
		ratio = 2
	}
	b := base.Bounds()
	length, breadth := b.X1-b.X0, b.Y1-b.Y0
	if !horizontal {
		length, breadth = breadth, length
	}
	baseScale := pageScale(base, o.FontSize)
	// Krótki rysunek się nie zwija: poziomy mieści się na szerokość pionowej kartki A4.
	if length < ratio*breadth || (horizontal && length <= 650*o.FontSize/16) {
		return nil
	}
	// Ocena: najpierw skala na stronie A4, potem proporcje bliskie kartki (√2 : 1).
	shape := func(sc *scene.Scene) float64 {
		b := sc.Bounds()
		return math.Abs(math.Log((b.X1 - b.X0) / (b.Y1 - b.Y0) / math.Sqrt2))
	}
	rank, links := foldRanks(g)
	if rank == nil {
		return nil
	}
	last := 0
	for _, r := range rank {
		last = max(last, r)
	}
	// Dozwolone cięcia: po warstwie k, gdy każda krawędź przez cięcie biegnie z k do k+1, jest ich
	// najwyżej maxFoldLinks i mają zwykłe zakończenia.
	edgeUnits := func(e *graph.Edge) (int, int) { return rank[foldUnit(g, e.From)], rank[foldUnit(g, e.To)] }
	valid := map[int]bool{}
	for k := 0; k < last; k++ {
		ok, n := true, 0
		for _, e := range g.Edges {
			a, b := edgeUnits(e)
			if a > k || b <= k {
				continue
			}
			if a != k || b != k+1 || !plainHead(e.StartHead) || !plainHead(e.EndHead) {
				ok = false
				break
			}
			if e.Style != graph.Invisible {
				n++
			}
		}
		valid[k] = ok && n <= maxFoldLinks
	}
	_ = links

	// Długość warstw z rysunku bazowego: jednostka to pole albo ramka.
	o.Fold = FoldOff
	_, boxes := layoutBoxes(g, o, fit)
	extent := make([]float64, last+1)
	for u, r := range rank {
		if bx, ok := boxes[u]; ok {
			l := bx.X1 - bx.X0
			if !horizontal {
				l = bx.Y1 - bx.Y0
			}
			extent[r] = math.Max(extent[r], l)
		}
	}
	total := 0.0
	for _, e := range extent {
		total += e
	}

	// Zwinięcie musi dać większą skalę albo, przy tej samej, wyraźnie lepsze proporcje.
	var best *scene.Scene
	bestScale, bestShape := baseScale, shape(base)-0.4
	for parts := 2; parts <= 4 && parts <= last+1; parts++ {
		// Cięcia najbliżej równych części długości.
		var cuts []int
		sum, k := 0.0, 0
		for p := 1; p < parts; p++ {
			target := total * float64(p) / float64(parts)
			bestK, bestD := -1, math.Inf(1)
			for ; k < last; k++ {
				sum += extent[k]
				if valid[k] && math.Abs(sum-target) < bestD {
					bestK, bestD = k, math.Abs(sum-target)
				}
				if sum >= target {
					k++
					break
				}
			}
			if bestK < 0 || (len(cuts) > 0 && bestK <= cuts[len(cuts)-1]) {
				cuts = nil
				break
			}
			cuts = append(cuts, bestK)
			sum = 0
			for _, e := range extent[:bestK+1] {
				sum += e
			}
			k = bestK + 1
		}
		if len(cuts) != parts-1 {
			continue
		}
		sc := assembleFold(g, o, fit, rank, cuts, horizontal)
		if sc == nil {
			continue
		}
		// Diagram pionowy zwija się tylko dla większej skali: pionowa kartka dobrze mieści wysoki rysunek.
		s, sh := pageScale(sc, o.FontSize), shape(sc)
		if s > bestScale*1.05 || (horizontal && s >= bestScale-0.001 && sh < bestShape) {
			best, bestScale, bestShape = sc, s, sh
		}
	}
	return best
}

func plainHead(h graph.Head) bool { return h == "" || h == graph.NoHead || h == graph.ArrowHead }

// assembleFold rozmieszcza grupy warstw oddzielone cięciami i łączy je łącznikami.
func assembleFold(g *graph.Graph, o Options, fit float64, rank map[string]int, cuts []int, horizontal bool) *scene.Scene {
	u := func(v float64) float64 { return v * o.FontSize / 16 }
	group := func(r int) int {
		i := 0
		for i < len(cuts) && r > cuts[i] {
			i++
		}
		return i
	}
	parts := len(cuts) + 1
	type segment struct {
		sc    *scene.Scene
		boxes map[string]scene.Box
		dx    float64
		dy    float64
		rev   bool
	}
	segs := make([]segment, parts)
	for i := range segs {
		nodes, subs := map[string]bool{}, map[string]bool{}
		for _, n := range g.Nodes {
			if group(rank[foldUnit(g, n.ID)]) == i {
				nodes[n.ID] = true
			}
		}
		for _, s := range g.Subgraphs {
			if group(rank[foldUnit(g, s.ID)]) == i {
				subs[s.ID] = true
			}
		}
		part := g.Part(nodes, subs)
		rev := o.FoldType == FoldSnake && i%2 == 1
		if rev {
			if horizontal {
				part.Direction = graph.RightLeft
			} else {
				part.Direction = graph.BottomTop
			}
		}
		sc, boxes := layoutBoxes(part, o, fit)
		segs[i] = segment{sc: sc, boxes: boxes, rev: rev}
	}
	// Osie umowne: wzdłuż przebiegu (flow) i w poprzek, w którą stronę idą kolejne rzędy (stack).
	flow := func(b scene.Box) (float64, float64) {
		if horizontal {
			return b.X0, b.X1
		}
		return b.Y0, b.Y1
	}
	stack := func(b scene.Box) (float64, float64) {
		if horizontal {
			return b.Y0, b.Y1
		}
		return b.X0, b.X1
	}
	real := func(f, s float64) scene.Point {
		if horizontal {
			return scene.Point{X: f, Y: s}
		}
		return scene.Point{X: s, Y: f}
	}
	// Krawędzie między grupami i ich podpisy.
	crossing := make([][]*graph.Edge, parts)
	for _, e := range g.Edges {
		a, b := group(rank[foldUnit(g, e.From)]), group(rank[foldUnit(g, e.To)])
		if a != b && e.Style != graph.Invisible {
			crossing[a] = append(crossing[a], e)
		}
	}
	labelSize := func(e *graph.Edge) (float64, float64) {
		w := 0.0
		for _, line := range e.Label {
			w = math.Max(w, o.Measure(graph.PerWord(line), o.FontSize))
		}
		return w, float64(len(e.Label)) * math.Round(o.FontSize*1.25)
	}
	// Odstęp między rzędami: każdy łącznik ma w nim własny tor, wysoki na tyle, żeby zmieścić podpis.
	gaps := make([]float64, parts)
	lanes := make([][]float64, parts) // położenie toru łącznika liczone od początku odstępu
	for i, es := range crossing {
		at := u(20)
		for _, e := range es {
			// Rozmiar podpisu w poprzek odstępu: wysokość w rzędach (LR), szerokość w kolumnach (TB).
			h := 0.0
			if len(e.Label) > 0 {
				w, lh := labelSize(e)
				h = lh
				if !horizontal {
					h = w + u(8)
				}
			}
			lanes[i] = append(lanes[i], at+h/2)
			at += math.Max(h, u(4)) + u(10)
		}
		gaps[i] = math.Max(u(48), at+u(10))
	}
	maxFlow := 0.0
	for _, sg := range segs {
		f0, f1 := flow(sg.sc.Bounds())
		maxFlow = math.Max(maxFlow, f1-f0)
	}
	cursor := 0.0
	for i := range segs {
		bb := segs[i].sc.Bounds()
		f0, f1 := flow(bb)
		s0, s1 := stack(bb)
		df := -f0
		if segs[i].rev {
			df = maxFlow - f1
		}
		ds := cursor - s0
		if horizontal {
			segs[i].dx, segs[i].dy = df, ds
		} else {
			segs[i].dx, segs[i].dy = ds, df
		}
		cursor += s1 - s0 + gaps[i]
	}

	out := &scene.Scene{Title: segs[0].sc.Title, Desc: segs[0].sc.Desc, FontFamily: segs[0].sc.FontFamily, FontSize: segs[0].sc.FontSize,
		Margin: segs[0].sc.Margin, Background: segs[0].sc.Background}
	for _, sg := range segs {
		for _, it := range sg.sc.Items {
			out.Add(shiftItem(it, sg.dx, sg.dy))
		}
	}
	box := func(i int, id string) (scene.Box, bool) {
		b, ok := segs[i].boxes[id]
		if !ok {
			return b, false
		}
		return scene.Box{X0: b.X0 + segs[i].dx, Y0: b.Y0 + segs[i].dy, X1: b.X1 + segs[i].dx, Y1: b.Y1 + segs[i].dy}, true
	}
	rowFlow := func(i int) (float64, float64) {
		bb := segs[i].sc.Bounds()
		f0, f1 := flow(bb)
		if horizontal {
			return f0 + segs[i].dx, f1 + segs[i].dx
		}
		return f0 + segs[i].dy, f1 + segs[i].dy
	}
	rowStack := func(i int) (float64, float64) {
		bb := segs[i].sc.Bounds()
		s0, s1 := stack(bb)
		if horizontal {
			return s0 + segs[i].dy, s1 + segs[i].dy
		}
		return s0 + segs[i].dx, s1 + segs[i].dx
	}
	type link struct {
		e       *graph.Edge
		pts     []scene.Point
		labelAt scene.Point
		lw, lh  float64
	}
	// route prowadzi łącznik e z rzędu i na torze lane; nil, gdy brakuje pola.
	route := func(i, j int, e *graph.Edge, lane float64) *link {
		src, ok1 := box(i, e.From)
		dst, ok2 := box(i+1, e.To)
		if !ok1 || !ok2 {
			return nil
		}
		sf0, sf1 := flow(src)
		ss0, ss1 := stack(src)
		tf0, tf1 := flow(dst)
		ts0, ts1 := stack(dst)
		off := float64(j) * u(10)
		lw, lh := labelSize(e)
		// Odstęp między rzędami: łącznik schodzi do niego ze źródła, biegnie nim i wchodzi do celu
		// od strony poprzedniego rzędu. Gdy pod źródłem albo nad celem stoi inne pole, łącznik
		// omija je bokiem rzędu.
		_, bottom := rowStack(i)
		channel := bottom + lane
		sMid, tMid := (sf0+sf1)/2, (tf0+tf1)/2
		var head []scene.Point
		if blocked(segs[i].boxes, segs[i].dx, segs[i].dy, e.From, horizontal, true) {
			fs, fe := rowFlow(i)
			x, from := fe+u(20)+off, sf1
			if segs[i].rev {
				x, from = fs-u(20)-off, sf0
			}
			head = []scene.Point{real(from, (ss0+ss1)/2), real(x, (ss0+ss1)/2), real(x, channel)}
		} else {
			head = []scene.Point{real(sMid, ss1), real(sMid, channel)}
		}
		var tail []scene.Point
		if blocked(segs[i+1].boxes, segs[i+1].dx, segs[i+1].dy, e.To, horizontal, false) {
			fs, fe := rowFlow(i + 1)
			x, to := fs-u(20)-off, tf0
			if segs[i+1].rev {
				x, to = fe+u(20)+off, tf1
			}
			tail = []scene.Point{real(x, channel), real(x, (ts0+ts1)/2), real(to, (ts0+ts1)/2)}
		} else {
			tail = []scene.Point{real(tMid, channel), real(tMid, ts0)}
		}
		// Pola zachodzą na siebie wzdłuż rzędu i nic nie stoi na drodze: jedna prosta przez część wspólną,
		// w miejscu najbliższym osi obu pól.
		if len(head) == 2 && len(tail) == 2 {
			lo, hi := math.Max(sf0, tf0)+u(10), math.Min(sf1, tf1)-u(10)
			if lo <= hi {
				x := math.Min(math.Max((sMid+tMid)/2, lo), hi)
				head = []scene.Point{real(x, ss1), real(x, channel)}
				tail = []scene.Point{real(x, channel), real(x, ts0)}
			}
		}
		k := &link{e: e, pts: append(head, tail...), lw: lw, lh: lh}
		// Podpis na torze łącznika: pośrodku odcinka biegnącego wzdłuż odstępu, a gdy łącznik schodzi
		// prosto, obok niego.
		a, b := head[len(head)-1], tail[0]
		k.labelAt = scene.Point{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
		along, size, f := math.Abs(a.X-b.X), lw, a.X
		if !horizontal {
			along, size, f = math.Abs(a.Y-b.Y), lh, a.Y
		}
		if along < size+u(16) {
			k.labelAt = real(f+size/2+u(10), channel)
		}
		return k
	}
	for i, es := range crossing {
		if i+1 >= parts {
			continue
		}
		// Kolejność torów: ta, w której łączniki się nie krzyżują (przy dwóch łącznikach obie próby).
		orders := [][]int{{0, 1}}
		if len(es) == 2 {
			orders = append(orders, []int{1, 0})
		}
		var chosen []*link
		for _, ord := range orders {
			var links []*link
			for j, e := range es {
				k := route(i, j, e, lanes[i][ord[j]])
				if k == nil {
					return nil
				}
				links = append(links, k)
			}
			if len(links) < 2 || !polylinesCross(links[0].pts, links[1].pts) {
				chosen = links
				break
			}
		}
		if chosen == nil {
			return nil // łączniki krzyżują się w każdej kolejności: diagram nie jest prosty
		}
		for _, k := range chosen {
			drawLink(out, k.e, k.pts, u)
			if len(k.e.Label) > 0 {
				drawLinkLabel(out, k.e, k.labelAt, k.lw, k.lh, o)
			}
		}
	}
	return out
}

// shiftItem przesuwa kształt o (dx, dy).
func shiftItem(it scene.Item, dx, dy float64) scene.Item {
	it.X += dx
	it.Y += dy
	if it.Kind == scene.Line {
		it.X2 += dx
		it.Y2 += dy
	}
	if len(it.Points) > 0 {
		pts := make([]scene.Point, len(it.Points))
		for i, p := range it.Points {
			pts[i] = scene.Point{X: p.X + dx, Y: p.Y + dy}
		}
		it.Points = pts
	}
	if len(it.Segments) > 0 {
		segs := make([]scene.Segment, len(it.Segments))
		for i, s := range it.Segments {
			s.To = scene.Point{X: s.To.X + dx, Y: s.To.Y + dy}
			s.C1 = scene.Point{X: s.C1.X + dx, Y: s.C1.Y + dy}
			s.C2 = scene.Point{X: s.C2.X + dx, Y: s.C2.Y + dy}
			segs[i] = s
		}
		it.Segments = segs
	}
	return it
}

// drawLink rysuje łącznik: łamaną z zaokrąglonymi narożnikami i grotami jak krawędź schematu.
func drawLink(sc *scene.Scene, e *graph.Edge, pts []scene.Point, u func(float64) float64) {
	width, dash := 1.5, ""
	switch e.Style {
	case graph.Thick:
		width = 3
	case graph.Dotted:
		dash = "3 3"
	}
	stroke := colorLine
	if e.Paint.Stroke != "" {
		stroke = e.Paint.Stroke
	}
	if e.Paint.StrokeWidth > 0 {
		width = e.Paint.StrokeWidth
	}
	head := func(at, from scene.Point) scene.Point {
		dx, dy := at.X-from.X, at.Y-from.Y
		d := math.Hypot(dx, dy)
		if d == 0 {
			return at
		}
		dx, dy = dx/d, dy/d
		a, hw := u(12), u(6)
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerActivation, Role: "arrowhead", Fill: stroke,
			Points: []scene.Point{at, {X: at.X - dx*a - dy*hw, Y: at.Y - dy*a + dx*hw}, {X: at.X - dx*a + dy*hw, Y: at.Y - dy*a - dx*hw}}})
		return scene.Point{X: at.X - dx*a*0.7, Y: at.Y - dy*a*0.7}
	}
	n := len(pts)
	if e.EndHead == graph.ArrowHead {
		pts[n-1] = head(pts[n-1], pts[n-2])
	}
	if e.StartHead == graph.ArrowHead {
		pts[0] = head(pts[0], pts[1])
	}
	r := u(8)
	segs := []scene.Segment{{Cmd: 'M', To: pts[0]}}
	for i := 1; i < n-1; i++ {
		a, c, b := pts[i-1], pts[i], pts[i+1]
		in := math.Min(r, math.Hypot(c.X-a.X, c.Y-a.Y)/2)
		out := math.Min(r, math.Hypot(b.X-c.X, b.Y-c.Y)/2)
		rr := math.Min(in, out)
		toward := func(p, q scene.Point, d float64) scene.Point {
			l := math.Hypot(q.X-p.X, q.Y-p.Y)
			if l == 0 {
				return p
			}
			return scene.Point{X: p.X + (q.X-p.X)/l*d, Y: p.Y + (q.Y-p.Y)/l*d}
		}
		p1, p2 := toward(c, a, rr), toward(c, b, rr)
		segs = append(segs, scene.Segment{Cmd: 'L', To: p1},
			scene.Segment{Cmd: 'C', C1: toward(p1, c, rr*0.55), C2: toward(p2, c, rr*0.55), To: p2})
	}
	segs = append(segs, scene.Segment{Cmd: 'L', To: pts[n-1]})
	sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "edge", Segments: segs, Stroke: stroke, StrokeWidth: width, Dash: dash})
}

// drawLinkLabel rysuje podpis łącznika na podkładzie w kolorze tła.
func drawLinkLabel(sc *scene.Scene, e *graph.Edge, at scene.Point, w, h float64, o Options) {
	pad := o.FontSize / 4
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "edge-label-background",
		X: at.X - w/2 - pad, Y: at.Y - h/2, W: w + 2*pad, H: h, Knockout: true})
	lh := math.Round(o.FontSize * 1.25)
	for k, line := range e.Label {
		text := strings.TrimSpace(line)
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "edge-label", X: at.X,
			Y: at.Y - h/2 + float64(k)*lh + lh/2 + 0.35*o.FontSize, W: o.Measure(text, o.FontSize), H: lh,
			Text: text, Anchor: "middle", Fill: colorText})
	}
}

// blocked mówi, czy prosty odcinek od pola id do brzegu jego rzędu (pod polem, gdy below, nad nim
// w przeciwnym razie) przeszedłby przez inne pole albo ramkę, która pola nie zawiera.
func blocked(boxes map[string]scene.Box, dx, dy float64, id string, horizontal, below bool) bool {
	b, ok := boxes[id]
	if !ok {
		return true
	}
	inside := func(o, in scene.Box) bool { return o.X0 <= in.X0 && o.Y0 <= in.Y0 && o.X1 >= in.X1 && o.Y1 >= in.Y1 }
	for other, o := range boxes {
		if other == id || inside(o, b) {
			continue
		}
		var overlap, after bool
		if horizontal {
			cx := (b.X0 + b.X1) / 2
			overlap = cx > o.X0 && cx < o.X1
			after = (below && o.Y0 >= b.Y1-0.5) || (!below && o.Y1 <= b.Y0+0.5)
		} else {
			cy := (b.Y0 + b.Y1) / 2
			overlap = cy > o.Y0 && cy < o.Y1
			after = (below && o.X0 >= b.X1-0.5) || (!below && o.X1 <= b.X0+0.5)
		}
		if overlap && after {
			return true
		}
	}
	return false
}

// polylinesCross mówi, czy dwie łamane przecinają się poza końcami.
func polylinesCross(a, b []scene.Point) bool {
	orient := func(p, q, r scene.Point) float64 { return (q.X-p.X)*(r.Y-p.Y) - (q.Y-p.Y)*(r.X-p.X) }
	for i := 0; i+1 < len(a); i++ {
		for j := 0; j+1 < len(b); j++ {
			p1, p2, q1, q2 := a[i], a[i+1], b[j], b[j+1]
			d1, d2 := orient(q1, q2, p1), orient(q1, q2, p2)
			d3, d4 := orient(p1, p2, q1), orient(p1, p2, q2)
			if ((d1 > 1e-6 && d2 < -1e-6) || (d1 < -1e-6 && d2 > 1e-6)) && ((d3 > 1e-6 && d4 < -1e-6) || (d3 < -1e-6 && d4 > 1e-6)) {
				return true
			}
		}
	}
	return false
}
