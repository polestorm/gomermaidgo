package layered

import (
	"math"
	"sort"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Kolory rysunku. Wygląd jest stały i własny; nie naśladuje motywów Mermaid.
const (
	colorText       = "#1b1f24"
	colorLine       = "#3b4754"
	colorNodeFill   = "#eef2f7"
	colorNodeStroke = "#55657a"
	colorNoteFill   = "#fff6c2"
	colorNoteStroke = "#b59a1b"
	colorFrame      = "#8a94a3"
	colorSolid      = "#3b4754"
	colorRegion     = "#f1f4f8"
)

const shapeNote graph.Shape = "note"

func (l *layouter) wrapLimit(shape graph.Shape) float64 {
	limit := l.u(l.baseWrap())
	if shape == graph.Rhombus || shape == graph.Circle || shape == graph.DoubleCircle {
		limit = l.u(l.baseWrap() * 0.75)
	}
	return limit * l.narrow()
}

// narrow zwraca mnożnik szerokości pól przy dopasowaniu do strony (1, gdy go nie ma).
func (l *layouter) narrow() float64 {
	if l.fit <= 0 {
		return 1
	}
	return l.fit
}

// nodeSize zwraca rzeczywisty rozmiar kształtu mieszczącego podany tekst.
func (l *layouter) nodeSize(shape graph.Shape, text []string) (w, h float64) {
	tw, th := l.textSize(text)
	// Pole jest wyraźnie większe od tekstu: napis ma wokół siebie powietrze, a pola o różnej
	// długości tekstu nie różnią się tak mocno szerokością.
	// Pola są płaskie: szerokość co najmniej trzykrotnie większa od wysokości pola z jedną linią
	// tekstu, więc krótkie napisy dają pola tej samej szerokości i schemat wygląda równo.
	padX, padY := l.u(28), l.u(12)
	minW := l.u(140)
	if f := l.narrow(); f < 1 {
		// Dopasowanie do strony: mniejsze marginesy i najmniejsza szerokość, żeby pola mogły się zwęzić.
		padX, minW = l.u(28)*math.Max(f, 0.5), l.u(140)*math.Max(f, 0.4)
	}
	w, h = math.Max(tw+2*padX, minW), math.Max(th+2*padY, l.u(44))
	switch shape {
	case graph.Stadium:
		w += h / 2
	case graph.Subroutine:
		w += l.u(16)
	case graph.Cylinder:
		h += l.u(14)
	case graph.Circle, graph.DoubleCircle:
		d := math.Max(math.Hypot(tw, th)+l.u(16), l.u(44))
		if shape == graph.DoubleCircle {
			d += l.u(10)
		}
		w, h = d, d
	case graph.Rhombus:
		// Romb o szerokości s·tw i wysokości t·th mieści tekst, gdy 1/s + 1/t = 1;
		// s = 1 + √(th/tw) daje najmniejszą sumę szerokości i wysokości.
		tw, th = math.Max(tw, l.u(20))+l.u(12), th+l.u(8)
		s := 1 + math.Sqrt(th/tw)
		w, h = tw*s, th*s/(s-1)
	case graph.Hexagon, graph.LoopLimit:
		w += h / 2
	case graph.Parallelogram, graph.ParallelogramAlt, graph.Trapezoid, graph.TrapezoidAlt:
		w += 2 * h / 3
	case graph.Asymmetric:
		w += h / 4
	case graph.StateStart:
		return l.u(16), l.u(16)
	case graph.StateEnd:
		return l.u(20), l.u(20)
	case graph.Choice:
		return l.u(30), l.u(30)
	case graph.Fork, graph.Join:
		// Belka leży w poprzek kierunku diagramu.
		if l.vertical {
			return l.u(72), l.u(8)
		}
		return l.u(8), l.u(72)
	case shapeNote:
		return tw + 2*l.u(10), th + 2*l.u(8)
	}
	return math.Round(w), math.Round(h)
}

// textBlock rysuje linie tekstu wyśrodkowane w punkcie (cx, cy).
func (l *layouter) textBlock(lines []string, cx, cy float64, role string) {
	top := cy - float64(len(lines))*l.lh/2
	for k, s := range lines {
		if s == "" {
			continue
		}
		y := top + float64(k)*l.lh + l.lh/2 + 0.35*l.fs
		if graph.HasMarks(s) {
			// Linia z wyróżnieniami: fragmenty w różnych stylach stoją jeden za drugim.
			x := cx - l.width(s)/2
			for _, r := range graph.Segments(s) {
				w := l.runWidth(r)
				l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: y, W: w, H: l.lh,
					Text: strings.TrimRight(r.Text, " \t"), Anchor: "start", Bold: r.Bold, Italic: r.Italic, Fill: colorText})
				x += w
			}
			continue
		}
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: cx,
			Y: y, W: l.width(s), H: l.lh, Text: s, Anchor: "middle", Fill: colorText})
	}
}

// rect zwraca rzeczywisty prostokąt pola o środku (x, y) i rozmiarze (w, h) w osiach umownych.
func (l *layouter) rect(x, y, w, h float64) (cx, cy, rw, rh float64) {
	p := l.point(x, y)
	rw, rh = l.abstract(w, h)
	return p.X, p.Y, rw, rh
}

func (l *layouter) draw() {
	l.spreadPorts()
	if l.lanes != nil {
		l.drawLanes()
	}
	l.prepareRoutes()
	l.drawLevel(l.levels[""])
}

// portRange zwraca szerokość odcinka krawędzi pola, na którym wolno zaczepiać krawędzie.
// Kształty zbiegające się w wierzchołek mają jeden punkt zaczepienia.
func (l *layouter) portRange(it *item) float64 {
	w := it.nodeW()
	if it.kind == kindCluster {
		return w - 2*l.u(20)
	}
	switch it.node.Shape {
	case graph.Rhombus, graph.Choice:
		// Krawędzie wychodzą z boków przy wierzchołku, każda w swoim miejscu: widać, która
		// odpowiedź dokąd prowadzi.
		return w * 0.45
	case graph.Circle, graph.DoubleCircle:
		return w * 0.5
	case graph.StateStart, graph.StateEnd:
		return 0
	case graph.Stadium:
		return w - it.h
	case graph.Hexagon, graph.LoopLimit:
		return w - it.h/2 - 2*l.u(6)
	case graph.Parallelogram, graph.ParallelogramAlt, graph.Trapezoid, graph.TrapezoidAlt:
		return w - 2*it.h/3 - 2*l.u(6)
	}
	return w - 2*l.u(8)
}

// spreadPorts rozsuwa punkty zaczepienia krawędzi wychodzących z jednego boku pola, w kolejności
// położenia ich drugich końców, żeby krawędzie nie zaczynały się w jednym punkcie i nie krzyżowały
// tuż przy polu.
func (l *layouter) spreadPorts() {
	type end struct {
		e     *ledge
		upper bool    // koniec przy dolnym boku górnego pola
		key   float64 // położenie następnego punktu krawędzi
	}
	bottom, top := map[*item][]end{}, map[*item][]end{}
	var order []*item
	seen := map[*item]bool{}
	note := func(it *item) {
		if !seen[it] {
			seen[it] = true
			order = append(order, it)
		}
	}
	for _, s := range append([]string{""}, l.subgraphIDs()...) {
		for _, e := range l.levels[s].edges {
			if e.e.Style == graph.Invisible || e.flat {
				continue
			}
			next, prev := e.v.nodeX(), e.u.nodeX()
			// Koniec w podgrafie: liczy się punkt na jego ramce, nie środek ramki.
			if e.lowerCont != nil {
				next = e.lowerCont.u.x
			}
			if e.upperCont != nil {
				prev = e.upperCont.v.x
			}
			if len(e.chain) > 0 {
				next, prev = e.chain[0].x, e.chain[len(e.chain)-1].x
			}
			// Koniec prowadzony dalej we wnętrzu podgrafu albo leżący na ramce nie zaczepia się o pole.
			if e.upperCont == nil && e.u.kind != kindPort {
				bottom[e.u] = append(bottom[e.u], end{e, true, next})
				note(e.u)
			}
			if e.lowerCont == nil && e.v.kind != kindPort {
				top[e.v] = append(top[e.v], end{e, false, prev})
				note(e.v)
			}
		}
	}
	// straighten przesuwa punkt zaczepienia w oś następnego punktu krawędzi, jeśli ten leży
	// w zasięgu boku pola: krawędź biegnie wtedy prosto, bez drobnego uskoku. Kolejność punktów
	// na boku i odstęp między nimi zostają zachowane.
	straighten := func(it *item, ends []end) {
		usable := l.portRange(it)
		if usable <= 0 {
			return
		}
		get := func(en end) float64 {
			if en.upper {
				return en.e.upperPort
			}
			return en.e.lowerPort
		}
		for i, en := range ends {
			var target float64
			switch {
			case len(en.e.chain) > 0 && en.upper:
				target = en.e.chain[0].x
			case len(en.e.chain) > 0:
				target = en.e.chain[len(en.e.chain)-1].x
			case en.upper && en.e.lowerCont != nil:
				target = en.e.lowerCont.u.x
			case !en.upper && en.e.upperCont != nil:
				target = en.e.upperCont.v.x
			case en.upper && en.e.v.kind == kindPort:
				target = en.e.v.x
			case !en.upper && en.e.u.kind == kindPort:
				target = en.e.u.x
			case en.upper && en.e.lowerCont == nil && en.e.v.kind == kindNode:
				target = en.e.v.nodeX() + en.e.lowerPort
			case !en.upper && en.e.upperCont == nil && en.e.u.kind == kindNode:
				target = en.e.u.nodeX() + en.e.upperPort
			default:
				continue
			}
			off := target - it.nodeX()
			if math.Abs(off) > usable/2 {
				continue
			}
			// Jedyna krawędź wchodząca do pola od góry trafia w środek jego boku. Przesunięcie zostaje
			// tylko wtedy, gdy jest drobne: krawędź biegnie wtedy prosto, a uskok byłby gorszy niż
			// zaczepienie odrobinę obok środka.
			// Wyjątek: krawędzie z belki rozwidlenia albo złączenia biegną z niej pionowo, jak w UML.
			if !en.upper && len(ends) == 1 && it.kind == kindNode && math.Abs(off) >= l.u(12) && !bar(en.e.u) {
				continue
			}
			set := func(en end, v float64) {
				if en.upper {
					en.e.upperPort = v
				} else {
					en.e.lowerPort = v
				}
			}
			// Sąsiedni punkt, który stoi na drodze, odsuwa się, jeśli ma dokąd; inaczej uskok zostaje.
			if i > 0 && off < get(ends[i-1])+l.u(10) {
				if i > 1 || off-l.u(14) < -usable/2 {
					continue
				}
				set(ends[i-1], off-l.u(14))
			}
			if i+1 < len(ends) && off > get(ends[i+1])-l.u(10) {
				if i+2 < len(ends) || off+l.u(14) > usable/2 {
					continue
				}
				set(ends[i+1], off+l.u(14))
			}
			set(en, off)
		}
	}
	assign := func(it *item, ends []end) {
		k := len(ends)
		if k < 2 {
			return
		}
		usable := l.portRange(it)
		if usable <= 0 {
			return
		}
		sort.SliceStable(ends, func(a, b int) bool { return ends[a].key < ends[b].key })
		step := math.Min(l.u(40), usable/float64(k))
		for i, en := range ends {
			off := (float64(i) - float64(k-1)/2) * step
			if en.upper {
				en.e.upperPort = off
			} else {
				en.e.lowerPort = off
			}
		}
	}
	// Romb decyzji: gdy jedna gałąź biegnie prosto w dół, pozostałe wychodzą z bocznych wierzchołków,
	// jak w klasycznym schemacie blokowym: główny przebieg idzie osią, odgałęzienie w bok.
	if !l.o.CurvedEdges {
		for _, it := range order {
			bottom[it] = sideExitsOf(l, it, bottom[it], func(en end) (*ledge, float64) { return en.e, en.key })
		}
	}
	for _, it := range order {
		assign(it, bottom[it])
		assign(it, top[it])
	}
	// Dwa przebiegi: przy krawędzi bez punktów pośrednich jeden koniec ustawia się względem drugiego.
	for pass := 0; pass < 2; pass++ {
		for _, it := range order {
			straighten(it, bottom[it])
			straighten(it, top[it])
		}
	}
}

// sideExitsOf wybiera spośród krawędzi wychodzących z dolnego boku rombu te, które wyjdą z bocznych
// wierzchołków, oznacza je i zwraca pozostałe. Warunki: jedna z krawędzi biegnie prosto w osi pola;
// krawędź boczna skręca poza obrysem pola; w rzędzie pola nic nie stoi na jej drodze; z każdego
// wierzchołka wychodzi najwyżej jedna krawędź.
func sideExitsOf[E any](l *layouter, it *item, ends []E, of func(E) (*ledge, float64)) []E {
	if it.kind != kindNode || (it.node.Shape != graph.Rhombus && it.node.Shape != graph.Choice) || len(ends) < 2 {
		return ends
	}
	axis := it.nodeX()
	straight := false
	for _, en := range ends {
		if _, key := of(en); math.Abs(key-axis) < l.u(3) {
			straight = true
		}
	}
	if !straight {
		return ends
	}
	var row []*item
	first, _ := of(ends[0])
	for _, r := range first.lv.ranks {
		for _, o := range r {
			if o == it {
				row = r
			}
		}
	}
	half := it.nodeW() / 2
	used := map[float64]bool{}
	kept := ends[:0:0]
	for _, en := range ends {
		e, key := of(en)
		side := 1.0
		if key < axis {
			side = -1
		}
		free := !e.rev && (e.e.StartHead == graph.NoHead || e.e.StartHead == "") && !used[side] &&
			math.Abs(key-axis) > half+l.u(16)
		if free {
			lo, hi := math.Min(key, axis+side*half)-l.u(12), math.Max(key, axis+side*half)+l.u(12)
			for _, o := range row {
				if o != it && o.x+o.w/2 > lo && o.x-o.w/2 < hi {
					free = false
				}
			}
		}
		if !free {
			kept = append(kept, en)
			continue
		}
		used[side] = true
		e.sideExit = side
	}
	return kept
}

func (l *layouter) subgraphIDs() []string {
	ids := make([]string, len(l.g.Subgraphs))
	for i, s := range l.g.Subgraphs {
		ids[i] = s.ID
	}
	return ids
}

func (l *layouter) drawLevel(lv *level) {
	defer l.enter(lv)()
	if chainBlock(lv) {
		l.drawLevelContent(lv)
		l.drawChain(lv)
		return
	}
	if lv.sub != nil {
		cx, cy, w, h := l.rect(lv.offsetX+lv.w/2, lv.offsetY+lv.h/2, lv.w, lv.h)
		if l.o.boxes != nil {
			l.o.boxes[lv.sub.ID] = scene.Box{X0: cx - w/2, Y0: cy - h/2, X1: cx + w/2, Y1: cy + h/2}
		}
		from := len(l.sc.Items)
		frame := scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "cluster",
			X: cx - w/2, Y: cy - h/2, W: w, H: h, RX: l.u(4), Stroke: colorFrame, StrokeWidth: 1}
		if lv.sub.Region {
			// Region współbieżny: osobne pole z lekko przyciemnionym tłem i przerywanym obrysem.
			// Obrys przerywany zostaje: na wydruku samo tło bywa niewidoczne.
			frame.Fill, frame.RX, frame.Role, frame.Dash = colorRegion, l.u(6), "region", "5 4"
			defer func(prev string) { l.bg = prev }(l.bg)
			l.bg = colorRegion
		}
		paint := lv.sub.Paint
		// Tryb kolorowy: ramka podgrafu dostaje jasne tło i obrys w kolejnym kolorze serii, od drugiego:
		// pierwszy (niebieski) mają pola, więc pole nie ginie na tle ramki. Kolor ze źródła wygrywa.
		if l.o.Colors && !l.o.painted && !lv.sub.Region && !l.titleStrip(lv) && (paint.Fill == "" || paint.Fill == "none") {
			// Kolor nie może być kolorem żadnego pola stojącego bezpośrednio w ramce: kolejny z serii.
			used := map[string]bool{}
			for _, it := range lv.items {
				if it.kind == kindNode {
					fill, _ := l.tone(it.node.Shape)
					used[fill] = true
				}
			}
			k := l.subgraphIndex(lv.sub.ID) + 1
			for used[scene.Distinct(k).Fill] {
				k++
			}
			tone := scene.Distinct(k)
			frame.Fill, frame.Stroke = tone.Fill, tone.Stroke
			defer func(prev string) { l.bg = prev }(l.bg)
			l.bg = frame.Fill
		}
		if paint.Fill != "" && paint.Fill != "none" {
			// Wnętrze ramki ma własne tło: podkłady etykiet w środku biorą jego kolor.
			frame.Fill = paint.Fill
			defer func(prev string) { l.bg = prev }(l.bg)
			l.bg = paint.Fill
		}
		strip := l.titleStrip(lv)
		if strip {
			frame.RX = l.u(8)
		}
		l.sc.Add(frame)
		defer func() { l.repaint(from, from+1, paint, colorFrame) }()
		if strip {
			// Stan złożony: tytuł w pasku z lekkim tłem, oddzielonym linią od wnętrza.
			_, th := l.textSize(lv.titleLines)
			x0, y0, sh := cx-w/2, cy-h/2, th+2*l.u(6)
			if frame.Fill == "" {
				l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "cluster-title-strip", X: x0, Y: y0, W: w, H: sh, RX: frame.RX, Fill: colorNodeFill})
				l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "cluster-title-strip", X: x0, Y: y0 + sh/2, W: w, H: sh / 2, Fill: colorNodeFill})
			}
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerBackground, Role: "cluster-title-rule", X: x0, Y: y0 + sh, X2: x0 + w, Y2: y0 + sh, Stroke: colorFrame, StrokeWidth: 1})
			// Pasek u dołu, bez tekstu: duża ramka stanu jest rozpoznawalna także od spodu.
			// Jest niższy od promienia narożnika, więc jego dolne rogi idą łukiem obrysu.
			x1, y1, fh, r := x0+w, y0+h, l.u(footStrip), frame.RX
			dx := math.Sqrt(r*r - (r-fh)*(r-fh))
			if frame.Fill == "" {
				// Punkty kontrolne łuku: styczne w obu końcach, długość dla kąta łuku.
				k := 4.0 / 3 * math.Tan(math.Acos((r-fh)/r)/4) * r
				tx, ty := (r-fh)/r, dx/r // styczna w górnym końcu łuku, w stronę dołu
				l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerBackground, Role: "cluster-foot-strip", Fill: colorNodeFill,
					Segments: []scene.Segment{
						{Cmd: 'M', To: scene.Point{X: x0 + r - dx, Y: y1 - fh}},
						{Cmd: 'C', C1: scene.Point{X: x0 + r - dx + k*tx, Y: y1 - fh + k*ty}, C2: scene.Point{X: x0 + r - k, Y: y1}, To: scene.Point{X: x0 + r, Y: y1}},
						{Cmd: 'L', To: scene.Point{X: x1 - r, Y: y1}},
						{Cmd: 'C', C1: scene.Point{X: x1 - r + k, Y: y1}, C2: scene.Point{X: x1 - r + dx - k*tx, Y: y1 - fh + k*ty}, To: scene.Point{X: x1 - r + dx, Y: y1 - fh}},
						{Cmd: 'L', To: scene.Point{X: x0 + r - dx, Y: y1 - fh}},
					}})
			}
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerBackground, Role: "cluster-title-rule", X: x0 + r - dx, Y: y1 - fh, X2: x1 - r + dx, Y2: y1 - fh, Stroke: colorFrame, StrokeWidth: 1})
			// Obrys jeszcze raz, na wierzchu: pasek zakrył jego wewnętrzną połowę.
			outline := frame
			outline.Fill, outline.Role = "", "cluster-outline"
			at := len(l.sc.Items)
			l.sc.Add(outline)
			defer func() {
				l.repaint(at-1, at+1, graph.Paint{Stroke: paint.Stroke, StrokeWidth: paint.StrokeWidth, Dash: paint.Dash}, "")
			}()
			defer func(at int) { l.repaint(at, at+len(lv.titleLines), graph.Paint{Color: paint.Color}, "") }(len(l.sc.Items))
			l.textBlock(lv.titleLines, cx, y0+sh/2, "cluster-title")
		} else {
			l.drawFrameTitle(lv, cx, cy, w, h, paint)
		}
	}
	l.drawLevelContent(lv)
}

// drawFrameTitle rysuje tytuł zwykłej ramki: przy lewej górnej krawędzi, z podkładem w kolorze tła.
func (l *layouter) drawFrameTitle(lv *level, cx, cy, w, h float64, paint graph.Paint) {
	{
		if len(lv.titleLines) == 0 {
			return // ramka bez tytułu (region) nie dostaje pustego podkładu
		}
		defer func(at int) { l.repaint(at, at+1+len(lv.titleLines), graph.Paint{Color: paint.Color}, "") }(len(l.sc.Items))
		// Tytuł ma podkład w kolorze tła: krawędź wchodząca do ramki znika pod nim, zamiast go przekreślać.
		tw, th := l.textSize(lv.titleLines)
		tx, ty := cx-w/2+l.u(12)+tw/2, cy-h/2+l.u(6)+th/2
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "cluster-title-background",
			X: tx - tw/2 - l.u(4), Y: ty - th/2, W: tw + 2*l.u(4), H: th, Fill: l.labelBackground(), Knockout: l.labelBackground() == ""})
		l.textBlock(lv.titleLines, tx, ty, "cluster-title")
	}
}

// drawLevelContent rysuje wnętrze poziomu: pola, podgrafy i krawędzie.
func (l *layouter) drawLevelContent(lv *level) {
	for _, row := range lv.ranks {
		for _, it := range row {
			switch it.kind {
			case kindCluster:
				l.drawLevel(it.child)
			case kindNode:
				cx, cy, w, h := l.rect(it.nodeX(), it.y, it.nodeW(), it.h)
				if l.o.boxes != nil {
					l.o.boxes[it.node.ID] = scene.Box{X0: cx - w/2, Y0: cy - h/2, X1: cx + w/2, Y1: cy + h/2}
				}
				if it.node.Shape == graph.Entity || it.node.Shape == graph.Class || it.node.Shape == graph.Requirement {
					from := len(l.sc.Items)
					if it.node.Shape != graph.Entity {
						l.drawClass(it.node, cx, cy, w, h)
					} else {
						l.drawEntity(it.node, cx, cy, w, h)
					}
					l.repaint(from, len(l.sc.Items), it.node.Paint, "")
					if l.o.Shadows {
						l.sc.AddShadows(from, l.u(3), "node")
					}
				} else {
					l.drawNode(it.node.Shape, it.text, cx, cy, w, h, it.node.Paint)
				}
				if a := it.anchor; a != nil {
					// Łącznik notatki: krótka linia kropkowana między bokami pola i notatki.
					from := l.point(a.nodeX()+it.anchorSide*a.nodeW()/2, it.y)
					to := l.point(it.x-it.anchorSide*it.w/2, it.y)
					l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "note-link",
						X: from.X, Y: from.Y, X2: to.X, Y2: to.Y, Stroke: colorNoteStroke, StrokeWidth: 1.5, Dash: "3 3"})
				}
				for i, e := range lv.loops[it] {
					l.drawLoop(it, e, i)
				}
			}
		}
	}
	for _, r := range lv.routes {
		l.drawEdge(r)
	}
}

// prepareRoutes wyznacza przebieg krawędzi wszystkich poziomów, zanim cokolwiek zostanie narysowane:
// najpierw przebieg każdej krawędzi i przydział torów odcinkom poprzecznym na jej poziomie, potem
// mostki wspólnie dla wszystkich poziomów o tych samych osiach, bo krawędź prowadzona w głąb
// podgrafu krzyżuje się też z krawędziami tego podgrafu. Kolejność jest kolejnością rysowania:
// zawartość podgrafu przed krawędziami poziomu, w którym leży.
func (l *layouter) prepareRoutes() {
	type frame struct {
		vertical, flip bool
		ox, oy         float64
	}
	groups := map[frame][]*edgeRoute{}
	var frames []frame
	var visit func(lv *level)
	visit = func(lv *level) {
		for _, row := range lv.ranks {
			for _, it := range row {
				if it.kind == kindCluster {
					visit(it.child)
				}
			}
		}
		defer l.enter(lv)()
		if lv.vertical && !l.o.CurvedEdges {
			l.routesWithLabels(lv)
		} else {
			lv.routes = l.routes(lv)
			if !l.o.CurvedEdges {
				l.assignTracks(lv.routes)
			}
		}
		f := frame{lv.vertical, lv.flip, lv.ox, lv.oy}
		if _, ok := groups[f]; !ok {
			frames = append(frames, f)
		}
		groups[f] = append(groups[f], lv.routes...)
	}
	visit(l.levels[""])
	if !l.o.CurvedEdges {
		for _, f := range frames {
			l.findHops(groups[f])
		}
	}
}

// routes zwraca przebiegi krawędzi jednego poziomu, przygotowane do rysowania.
func (l *layouter) routes(lv *level) []*edgeRoute {
	var routes []*edgeRoute
	// Najpierw przebiegi wszystkich krawędzi poziomu: wyrównanie przejść przez ramki musi znać
	// przejścia pozostałych krawędzi przez te same ramki.
	all := map[*ledge][]routePoint{}
	inside := func(e *ledge) bool { return chainBlock(lv) && e.u.kind == kindNode && e.v.kind == kindNode }
	// LAY-028: punkty pośrednie wszystkich krawędzi poziomu wyrównywane kilka razy: przesunięcie jednej
	// krawędzi zwalnia miejsce dla sąsiedniej.
	for pass := 0; pass < 3; pass++ {
		for _, e := range lv.edges {
			if !e.part && !e.flat && e.e.Style != graph.Invisible && !inside(e) {
				l.alignChain(e)
			}
		}
	}
	for _, e := range lv.edges {
		if !e.part && e.e.Style != graph.Invisible && !inside(e) {
			all[e] = l.route(e)
		}
	}
	for _, e := range lv.edges {
		if e.part || e.e.Style == graph.Invisible || inside(e) {
			continue
		}
		// Dwa przebiegi: po wyrównaniu przejścia przez ramkę odcinek przed nim może dołączyć do niego.
		l.snapCrossings(all[e])
		l.snapCrossings(all[e])
	}
	for _, e := range lv.edges {
		if e.part || e.e.Style == graph.Invisible || inside(e) {
			continue
		}
		topHead, bottomHead := e.e.StartHead, e.e.EndHead
		if e.rev {
			topHead, bottomHead = bottomHead, topHead
		}
		pts := withoutTurnBacks(all[e])
		// Różnice położeń poniżej progu to zaokrąglenia: odcinek ma być dokładnie prosty, nie skośny.
		for i := 1; i < len(pts); i++ {
			if math.Abs(pts[i].x-pts[i-1].x) < straightTolerance {
				pts[i].x = pts[i-1].x
			}
		}
		r := &edgeRoute{e: e, topHead: topHead, bottomHead: bottomHead}
		first := e
		for first.upperCont != nil {
			first = first.upperCont
		}
		r.side = first.sideExit != 0
		last := len(pts) - 1
		r.top, r.bottom = pts[0], pts[last]
		if e.flat {
			r.flat = true
			dir := 1.0
			if pts[last].x < pts[0].x {
				dir = -1
			}
			pts[0].x += dir * l.headLength(topHead)
			pts[last].x -= dir * l.headLength(bottomHead)
			r.pts = pts
			routes = append(routes, r)
			continue
		}
		pts[0].y += l.headLength(topHead)
		pts[last].y -= l.headLength(bottomHead)
		// Punkty pośrednie leżące na jednej prostej z sąsiadami są zbędne, a punkt, który po
		// skróceniu końca o grot znalazł się za nim, cofałby linię.
		kept := pts[:1]
		for i := 1; i < last; i++ {
			prev, next := kept[len(kept)-1], pts[i+1]
			if prev.x == pts[i].x && pts[i].x == next.x && !(r.side && i == 1) {
				continue
			}
			kept = append(kept, pts[i])
		}
		r.pts = append(kept, pts[last])
		routes = append(routes, r)
	}
	return routes
}

// tone zwraca kolory węzła w trybie kolorowym: kolor mówi, czym węzeł jest w procesie.
func (l *layouter) tone(shape graph.Shape) (fill, stroke string) {
	if l.o.painted {
		return "#ffffff", colorNodeStroke
	}
	if !l.o.Colors {
		return colorNodeFill, colorNodeStroke
	}
	t := scene.Palette[0] // krok
	switch shape {
	case graph.Rhombus, graph.Hexagon, graph.Choice:
		t = scene.Palette[2] // decyzja
	case graph.Stadium, graph.Circle, graph.DoubleCircle:
		t = scene.Palette[1] // początek, koniec
	case graph.Cylinder, graph.Subroutine:
		t = scene.Palette[3] // dane, podproces
	case graph.Parallelogram, graph.ParallelogramAlt, graph.Trapezoid, graph.TrapezoidAlt, graph.Asymmetric:
		t = scene.Palette[5] // wejście, wyjście, czynność ręczna
	}
	return t.Fill, t.Stroke
}

// repaint nakłada wygląd narzucony przez źródło na kształty sceny o numerach od from do to:
// wypełnienie zastępuje wypełnienie domyślne, obrys zastępuje kolor stroke, kolor tekstu
// dotyczy tekstów. Kształty w kolorze pełnym (groty, stan początkowy) zmieniają się z obrysem.
func (l *layouter) repaint(from, to int, p graph.Paint, stroke string) {
	if p.IsZero() {
		return
	}
	for i := from; i < to && i < len(l.sc.Items); i++ {
		it := &l.sc.Items[i]
		if it.Kind == scene.Text {
			if p.Color != "" {
				it.Fill = p.Color
			}
			continue
		}
		if strings.HasSuffix(it.Role, "-background") {
			continue
		}
		if p.Stroke != "" && stroke != "" && it.Fill == stroke {
			it.Fill = p.Stroke
		} else if p.Fill != "" && it.Fill != "" && it.Fill != colorSolid {
			it.Fill = p.Fill
			if p.Fill == "none" {
				it.Fill = ""
			}
		}
		if it.Stroke == "" {
			continue
		}
		if p.Stroke != "" {
			it.Stroke = p.Stroke
		}
		if p.StrokeWidth != 0 {
			it.StrokeWidth = p.StrokeWidth
		}
		if p.Dash != "" {
			it.Dash = p.Dash
		}
	}
}

// entityColumns zwraca szerokości kolumn tabeli atrybutów encji i wysokości nagłówka oraz wiersza.
func (l *layouter) entityColumns(n *graph.Node) (cols [4]float64, head, row float64) {
	for _, r := range n.Table {
		for c := range cols {
			if r[c] != "" {
				cols[c] = math.Max(cols[c], l.width(r[c])+2*l.u(10))
			}
		}
	}
	_, th := l.textSize(n.Label)
	return cols, th + 2*l.u(10), l.lh + l.u(10)
}

// entitySize zwraca rozmiar encji: nagłówek z nazwą i tabela atrybutów.
func (l *layouter) entitySize(n *graph.Node) (w, h float64) {
	cols, head, row := l.entityColumns(n)
	tw, _ := l.textSize(n.Label)
	w = math.Max(tw+2*l.u(24), l.u(120))
	w = math.Max(w, cols[0]+cols[1]+cols[2]+cols[3])
	return math.Round(w), head + float64(len(n.Table))*row
}

// drawEntity rysuje encję: ramka, nagłówek z nazwą, wiersze atrybutów rozdzielone liniami.
func (l *layouter) drawEntity(n *graph.Node, cx, cy, w, h float64) {
	fill, stroke := l.tone(graph.Entity)
	cols, head, row := l.entityColumns(n)
	x0, y0 := cx-w/2, cy-h/2
	body := "#ffffff"
	if len(n.Table) == 0 {
		body = fill
	}
	l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "node", X: x0, Y: y0, W: w, H: h, Fill: body, Stroke: stroke, StrokeWidth: 1.5})
	if len(n.Table) > 0 {
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "node-detail", X: x0, Y: y0, W: w, H: head, Fill: fill, Stroke: stroke, StrokeWidth: 1.5})
	}
	l.textBlock(n.Label, cx, y0+head/2, "node-label")
	for i, r := range n.Table {
		top := y0 + head + float64(i)*row
		if i > 0 {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "node-detail", X: x0, Y: top, X2: x0 + w, Y2: top, Stroke: colorFrame, StrokeWidth: 1})
		}
		x := x0
		for c, cell := range r {
			if cell != "" {
				l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "attribute", X: x + l.u(10),
					Y: top + row/2 + 0.35*l.fs, W: l.width(cell), H: l.lh, Text: cell, Anchor: "start", Fill: colorText})
			}
			x += cols[c]
		}
	}
}

// classParts zwraca wymiary części klasy: nagłówka z adnotacją i nazwą, wiersza składowej
// oraz przedziałów atrybutów i metod (0, gdy klasa nie ma żadnych składowych).
func (l *layouter) classParts(n *graph.Node) (head, row, attrs, methods float64) {
	lines := float64(len(n.Label))
	if n.Annotation != "" {
		lines++
	}
	head, row = lines*l.lh+2*l.u(8), l.lh+l.u(2)
	if len(n.Members) == 0 {
		return head, row, 0, 0
	}
	count := [2]float64{}
	for _, m := range n.Members {
		if m.Method {
			count[1]++
		} else {
			count[0]++
		}
	}
	// Pusty przedział zostaje jako wąski pasek: widać, że klasa nie ma atrybutów albo metod.
	part := func(k float64) float64 { return math.Max(k*row+2*l.u(5), l.u(12)) }
	if n.Shape == graph.Requirement {
		return head, row, part(count[0]), 0 // wymaganie ma jeden przedział pól
	}
	return head, row, part(count[0]), part(count[1])
}

// boldWidth mierzy tekst odmianą pogrubioną.
func (l *layouter) boldWidth(s string) float64 {
	return l.width(string(graph.MarkBold) + s + string(graph.MarkBold))
}

// classSize zwraca rozmiar klasy: nagłówek i przedziały składowych.
func (l *layouter) classSize(n *graph.Node) (w, h float64) {
	head, _, attrs, methods := l.classParts(n)
	for _, s := range n.Label {
		w = math.Max(w, l.boldWidth(s))
	}
	if n.Annotation != "" {
		w = math.Max(w, l.width("«"+n.Annotation+"»"))
	}
	for _, m := range n.Members {
		w = math.Max(w, l.width(m.Text))
	}
	return math.Round(math.Max(w+2*l.u(12), l.u(120))), head + attrs + methods
}

// drawClass rysuje klasę: nagłówek z adnotacją i pogrubioną nazwą, pod nim atrybuty i metody
// w osobnych przedziałach. Składowa statyczna jest podkreślona, abstrakcyjna pochylona.
func (l *layouter) drawClass(n *graph.Node, cx, cy, w, h float64) {
	fill, stroke := l.tone(graph.Class)
	head, row, attrs, methods := l.classParts(n)
	x0, y0 := cx-w/2, cy-h/2
	body := "#ffffff"
	if len(n.Members) == 0 {
		body = fill
	}
	l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "node", X: x0, Y: y0, W: w, H: h, Fill: body, Stroke: stroke, StrokeWidth: 1.5})
	if len(n.Members) > 0 {
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "node-detail", X: x0, Y: y0, W: w, H: head, Fill: fill, Stroke: stroke, StrokeWidth: 1.5})
		if methods > 0 {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "node-detail", X: x0, Y: y0 + head + attrs, X2: x0 + w, Y2: y0 + head + attrs, Stroke: stroke, StrokeWidth: 1})
		}
	}
	top := y0 + l.u(8)
	if n.Annotation != "" {
		s := "«" + n.Annotation + "»"
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "node-label", X: cx, Y: top + l.lh/2 + 0.35*l.fs,
			W: l.width(s), H: l.lh, Text: s, Anchor: "middle", Fill: colorText})
		top += l.lh
	}
	for _, s := range n.Label {
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "node-label", X: cx, Y: top + l.lh/2 + 0.35*l.fs,
			W: l.boldWidth(s), H: l.lh, Text: s, Anchor: "middle", Bold: true, Fill: colorText})
		top += l.lh
	}
	next := [2]float64{y0 + head + l.u(5), y0 + head + attrs + l.u(5)}
	for _, m := range n.Members {
		k := 0
		if m.Method {
			k = 1
		}
		mid, x := next[k]+row/2, x0+l.u(12)
		next[k] += row
		plain, italic := graph.Plain(m.Text), graph.HasMarks(m.Text)
		tw := l.width(m.Text)
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "member", X: x, Y: mid + 0.35*l.fs,
			W: tw, H: l.lh, Text: plain, Anchor: "start", Italic: italic, Fill: colorText})
		if m.Static {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabel, Role: "member-underline", X: x, Y: mid + 0.35*l.fs + l.u(3),
				X2: x + tw, Y2: mid + 0.35*l.fs + l.u(3), Stroke: colorText, StrokeWidth: 1})
		}
	}
}

func (l *layouter) drawNode(shape graph.Shape, text []string, cx, cy, w, h float64, paint graph.Paint) {
	colorNodeFill, colorNodeStroke := l.tone(shape)
	from := len(l.sc.Items)
	defer func() {
		l.repaint(from, len(l.sc.Items), paint, "")
		if l.o.Shadows && shape != graph.StateStart && shape != graph.StateEnd && shape != graph.Fork && shape != graph.Join {
			l.sc.AddShadows(from, l.u(3), "node", "note")
		}
	}()
	x0, y0, x1, y1 := cx-w/2, cy-h/2, cx+w/2, cy+h/2
	box := func(rx float64, fill, stroke string) {
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "node", X: x0, Y: y0, W: w, H: h, RX: rx,
			Fill: fill, Stroke: stroke, StrokeWidth: 1.5})
	}
	poly := func(pts ...scene.Point) {
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "node", Points: pts,
			Fill: colorNodeFill, Stroke: colorNodeStroke, StrokeWidth: 1.5})
	}
	line := func(ax, ay, bx, by float64) {
		l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "node-detail", X: ax, Y: ay, X2: bx, Y2: by,
			Stroke: colorNodeStroke, StrokeWidth: 1.5})
	}
	circles := 0
	circle := func(r float64, fill, stroke string) {
		role := "node"
		if circles++; circles > 1 {
			role = "node-detail" // wewnętrzny okrąg podwójnego koła i stanu końcowego
		}
		l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: role, X: cx, Y: cy, R: r,
			Fill: fill, Stroke: stroke, StrokeWidth: 1.5})
	}
	pt := func(x, y float64) scene.Point { return scene.Point{X: x, Y: y} }
	textX := cx

	switch shape {
	case graph.Round:
		box(l.u(8), colorNodeFill, colorNodeStroke)
	case graph.Stadium:
		box(h/2, colorNodeFill, colorNodeStroke)
	case graph.Subroutine:
		box(0, colorNodeFill, colorNodeStroke)
		line(x0+l.u(8), y0, x0+l.u(8), y1)
		line(x1-l.u(8), y0, x1-l.u(8), y1)
	case graph.Cylinder:
		cap := l.u(14)
		ellipse := func(y float64) {
			l.sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "node-detail", X: cx, Y: y, W: w, H: cap,
				Fill: colorNodeFill, Stroke: colorNodeStroke, StrokeWidth: 1.5})
		}
		ellipse(y1 - cap/2)
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "node", X: x0, Y: y0 + cap/2, W: w, H: h - cap, Fill: colorNodeFill})
		line(x0, y0+cap/2, x0, y1-cap/2)
		line(x1, y0+cap/2, x1, y1-cap/2)
		ellipse(y0 + cap/2)
		cy += cap / 4
	case graph.Circle:
		circle(w/2, colorNodeFill, colorNodeStroke)
	case graph.DoubleCircle:
		circle(w/2, colorNodeFill, colorNodeStroke)
		circle(w/2-l.u(5), "", colorNodeStroke)
	case graph.Asymmetric:
		poly(pt(x0, y0), pt(x1, y0), pt(x1, y1), pt(x0, y1), pt(x0+h/4, cy))
		textX += h / 8
	case graph.Rhombus, graph.Choice:
		poly(pt(cx, y0), pt(x1, cy), pt(cx, y1), pt(x0, cy))
	case graph.Hexagon:
		m := h / 4
		poly(pt(x0+m, y0), pt(x1-m, y0), pt(x1, cy), pt(x1-m, y1), pt(x0+m, y1), pt(x0, cy))
	case graph.LoopLimit:
		m := h / 4
		poly(pt(x0+m, y0), pt(x1-m, y0), pt(x1, y0+m), pt(x1, y1), pt(x0, y1), pt(x0, y0+m))
	case graph.Parallelogram:
		m := h / 3
		poly(pt(x0+m, y0), pt(x1, y0), pt(x1-m, y1), pt(x0, y1))
	case graph.ParallelogramAlt:
		m := h / 3
		poly(pt(x0, y0), pt(x1-m, y0), pt(x1, y1), pt(x0+m, y1))
	case graph.Trapezoid:
		m := h / 3
		poly(pt(x0+m, y0), pt(x1-m, y0), pt(x1, y1), pt(x0, y1))
	case graph.TrapezoidAlt:
		m := h / 3
		poly(pt(x0, y0), pt(x1, y0), pt(x1-m, y1), pt(x0+m, y1))
	case graph.StateStart:
		circle(w/2, colorSolid, "")
	case graph.StateEnd:
		circle(w/2, "", colorSolid)
		circle(w/2-l.u(4), colorSolid, "")
	case graph.Fork, graph.Join:
		box(l.u(2), colorSolid, "")
	case shapeNote:
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "note", X: x0, Y: y0, W: w, H: h,
			Fill: colorNoteFill, Stroke: colorNoteStroke, StrokeWidth: 1})
	default:
		box(0, colorNodeFill, colorNodeStroke)
	}
	role := "node-label"
	if shape == shapeNote {
		role = "note-label"
	}
	l.textBlock(text, textX, cy, role)
}

func (l *layouter) strokeOf(e *graph.Edge) (width float64, dash string) {
	switch e.Style {
	case graph.Thick:
		return 3, ""
	case graph.Dotted:
		return 1.5, "3 3"
	}
	return 1.5, ""
}

// headAt rysuje zakończenie krawędzi w punkcie (x, y) osi umownych, skierowane wzdłuż (dx, dy),
// i zwraca, o ile trzeba skrócić linię, żeby nie wystawała spod zakończenia.
func (l *layouter) headAt(h graph.Head, x, y, dx, dy float64) float64 {
	// (px, py) to kierunek prostopadły.
	px, py := -dy, dx
	at := func(along, across float64) scene.Point { return l.point(x-dx*along+px*across, y-dy*along+py*across) }
	switch h {
	case graph.ArrowHead:
		a, hw := l.u(12), l.u(6)
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerActivation, Role: "arrowhead",
			Points: []scene.Point{at(0, 0), at(a, hw), at(a, -hw)}, Fill: colorLine})
		return a * 0.7
	case graph.CircleHead:
		r := l.u(4.5)
		c := at(r, 0)
		l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerActivation, Role: "arrowhead", X: c.X, Y: c.Y, R: r,
			Fill: "#ffffff", Stroke: colorLine, StrokeWidth: 1.5})
		return 2 * r
	case graph.OnlyOne, graph.ZeroOrOne, graph.OneOrMore, graph.ZeroOrMore:
		// Liczność związku: przy polu „jeden” (kreska) albo „wiele” (kurza stopka), dalej
		// „co najmniej jeden” (kreska) albo „zero” (kółko). Linia dochodzi do samego pola.
		stroke := func(a, b scene.Point) {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerActivation, Role: "arrowhead",
				X: a.X, Y: a.Y, X2: b.X, Y2: b.Y, Stroke: colorLine, StrokeWidth: 1.5})
		}
		hw := l.u(6)
		outer := l.u(7)
		if h == graph.OneOrMore || h == graph.ZeroOrMore {
			foot := l.u(11)
			stroke(at(foot, 0), at(0, hw))
			stroke(at(foot, 0), at(0, -hw))
			outer = foot + l.u(5)
		} else {
			stroke(at(l.u(7), hw), at(l.u(7), -hw))
			outer = l.u(13)
		}
		if h == graph.ZeroOrOne || h == graph.ZeroOrMore {
			r := l.u(4.5)
			c := at(outer+r-l.u(1), 0)
			l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerActivation, Role: "arrowhead", X: c.X, Y: c.Y, R: r,
				Fill: l.labelBackground(), Stroke: colorLine, StrokeWidth: 1.5})
		} else {
			stroke(at(outer, hw), at(outer, -hw))
		}
		return 0
	case graph.Triangle:
		// Dziedziczenie i realizacja: pusty trójkąt, linia kończy się na jego podstawie.
		a, hw := l.u(14), l.u(7.5)
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerActivation, Role: "arrowhead",
			Points: []scene.Point{at(0, 0), at(a, hw), at(a, -hw)}, Fill: l.labelBackground(), Stroke: colorLine, StrokeWidth: 1.5})
		return a
	case graph.DiamondFilled, graph.DiamondHollow:
		// Kompozycja (pełny romb) i agregacja (pusty): linia kończy się na dalszym wierzchołku.
		a, hw := l.u(18), l.u(5.5)
		fill := colorLine
		if h == graph.DiamondHollow {
			fill = l.labelBackground()
		}
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerActivation, Role: "arrowhead",
			Points: []scene.Point{at(0, 0), at(a/2, hw), at(a, 0), at(a/2, -hw)}, Fill: fill, Stroke: colorLine, StrokeWidth: 1.5})
		return a
	case graph.OpenArrow:
		// Asocjacja skierowana i zależność: otwarty grot z dwóch kresek, linia dochodzi do pola.
		a, hw := l.u(11), l.u(6)
		for _, side := range []float64{-1, 1} {
			p, q := at(0, 0), at(a, side*hw)
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerActivation, Role: "arrowhead",
				X: p.X, Y: p.Y, X2: q.X, Y2: q.Y, Stroke: colorLine, StrokeWidth: 1.5})
		}
		return 0
	case graph.CrossHead:
		// Krzyżyk stoi tuż przy polu, a linia kończy się w jego środku.
		r := l.u(5)
		for _, s := range []float64{-1, 1} {
			a, b := at(l.u(8)-r, s*r), at(l.u(8)+r, -s*r)
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerActivation, Role: "arrowhead",
				X: a.X, Y: a.Y, X2: b.X, Y2: b.Y, Stroke: colorLine, StrokeWidth: 1.5})
		}
	}
	return 0
}

func (l *layouter) labelBackground() string {
	if l.bg != "" {
		return l.bg
	}
	return "" // tło rysunku: podkład scene.Item.Knockout
}

// drawLabel rysuje etykietę krawędzi z podkładem w kolorze tła.
func (l *layouter) drawLabel(lines []string, x, y float64) {
	p := l.point(x, y)
	tw, th := l.textSize(lines)
	l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "edge-label-background",
		X: p.X - tw/2 - l.u(4), Y: p.Y - th/2, W: tw + 2*l.u(4), H: th, Fill: l.labelBackground(), Knockout: l.labelBackground() == ""})
	l.textBlock(lines, p.X, p.Y, "edge-label")
}

type routePoint struct{ x, y float64 }

// straightTolerance: odcinek, którego końce różnią się w poprzek o mniej, jest rysowany jako prosty.
// Różnice rzędu piksela biorą się z zaokrągleń położeń i dawałyby widoczny, bezsensowny uskok.
const straightTolerance = 2.5

// inset mówi, o ile brzeg kształtu jest bliżej środka pola niż krawędź jego prostokąta
// w miejscu przesuniętym o off od osi: romb i koło zwężają się ku bokom.
func inset(it *item, off float64) float64 {
	if it.kind != kindNode || off == 0 {
		return 0
	}
	half := it.nodeW() / 2
	switch it.node.Shape {
	case graph.Rhombus, graph.Choice:
		return it.h / 2 * math.Min(math.Abs(off)/half, 1)
	case graph.Circle, graph.DoubleCircle:
		r := it.h / 2
		return r - math.Sqrt(math.Max(r*r-off*off, 0))
	}
	return 0
}

// route zwraca punkty krawędzi od górnego końca do dolnego. Koniec leżący głębiej, w podgrafie,
// jest prowadzony odcinkiem z układu tego podgrafu: od pola do punktu na ramce albo odwrotnie.
func (l *layouter) route(e *ledge) []routePoint {
	if e.flat {
		// LAY-027: od boku pola do boku pola, w poprzek, na wysokości środków obu pól.
		dir := 1.0
		if e.v.x < e.u.x {
			dir = -1
		}
		return []routePoint{{e.u.nodeX() + dir*e.u.nodeW()/2, e.u.y}, {e.v.nodeX() - dir*e.v.nodeW()/2, e.v.y}}
	}
	l.alignChain(e)
	var pts []routePoint
	switch {
	case e.upperCont != nil:
		pts = l.route(e.upperCont)
	case e.u.kind == kindPort:
		border := e.lv.offsetY
		pts = append(pts, l.crossing(e.lv, routePoint{e.u.x, border}))
		if e.u.y-border > 0.5 {
			pts = append(pts, l.crossing(e.lv, routePoint{e.u.x, e.u.y}))
		}
	case e.sideExit != 0:
		// Początek w bocznym wierzchołku; drugi punkt, pod miejscem skrętu, dopisuje koniec funkcji.
		pts = append(pts, routePoint{e.u.nodeX() + e.sideExit*e.u.nodeW()/2, e.u.y})
	default:
		x := e.u.nodeX() + e.upperPort
		pts = append(pts, routePoint{x, e.u.y + e.u.h/2 - inset(e.u, e.upperPort)})
		// Pole niższe niż jego warstwa: krawędź biegnie prosto do granicy warstwy, żeby nie skręcać
		// obok wyższych sąsiadów.
		if (e.u.band-e.u.h)/2 > l.u(12) {
			pts = append(pts, routePoint{x, e.u.y + e.u.band/2})
		}
	}
	for _, d := range e.chain {
		lo, hi := l.dummyRoom(e.lv, d)
		if d.band > 0 {
			pts = append(pts, l.movable(routePoint{d.x, d.y - d.band/2}, lo, hi), l.movable(routePoint{d.x, d.y + d.band/2}, lo, hi))
		} else {
			pts = append(pts, l.movable(routePoint{d.x, d.y}, lo, hi))
		}
	}
	switch {
	case e.lowerCont != nil:
		pts = append(pts, l.route(e.lowerCont)...)
	case e.v.kind == kindPort:
		border := e.lv.offsetY + e.lv.h
		if border-e.v.y > 0.5 {
			pts = append(pts, l.crossing(e.lv, routePoint{e.v.x, e.v.y}))
		}
		pts = append(pts, l.crossing(e.lv, routePoint{e.v.x, border}))
	default:
		x := e.v.nodeX() + e.lowerPort
		// Tylko gdy między granicą warstwy a polem mieści się grot; inaczej skrócenie o grot cofnęłoby
		// koniec krawędzi za ten punkt.
		if (e.v.band-e.v.h)/2 > l.u(12) {
			pts = append(pts, routePoint{x, e.v.y - e.v.band/2})
		}
		pts = append(pts, routePoint{x, e.v.y - e.v.h/2 + inset(e.v, e.lowerPort)})
	}
	if e.sideExit != 0 && e.upperCont == nil && len(pts) > 1 {
		turn := routePoint{pts[1].x, e.u.y + math.Max(e.u.band, e.u.h)/2}
		pts = append(pts[:1], append([]routePoint{turn}, pts[1:]...)...)
	}
	return pts
}

// drawEdge rysuje krawędź: w paśmie warstwy biegnie prosto, między pasmami łagodnym łukiem.
// edgeRoute to przebieg jednej krawędzi przygotowany do rysowania.
type edgeRoute struct {
	e                   *ledge
	pts                 []routePoint // punkty przebiegu, skrócone o zakończenia
	top, bottom         routePoint   // końce przy polach, przed skróceniem
	topHead, bottomHead graph.Head
	track               map[int]float64   // położenie odcinka poprzecznego dla odcinka o danym numerze
	hops                map[int][]float64 // miejsca na odcinku poprzecznym, w których krawędź przeskakuje nad inną
	// runHops: to samo dla odcinków wzdłuż osi warstw; klucz to numer odcinka i jego część
	// (0: przed odcinkiem poprzecznym albo cały odcinek prosty, 1: za odcinkiem poprzecznym).
	runHops map[[2]int][]float64
	side    bool // pierwszy odcinek wychodzi w poprzek z bocznego wierzchołka rombu
	flat    bool // LAY-027: cała krawędź to jeden odcinek w poprzek, w jednej warstwie
	// labelAt: środek etykiety postawionej na odcinku poziomym (LAY-031); nil, gdy etykieta stoi
	// w swojej warstwie.
	labelAt *routePoint
	order   int // kolejność rysowania: krawędź późniejsza przeskakuje nad wcześniejszą
}

// findHops wyznacza przecięcia krawędzi łamanych: odcinek poprzeczny jednej krawędzi z odcinkiem
// biegnącym wzdłuż osi warstw innej. W takim miejscu odcinek poprzeczny dostaje półokrągły mostek,
// jak na rysunku technicznym, żeby przecięcia nie pomylić z połączeniem.
func (l *layouter) findHops(routes []*edgeRoute) {
	type along struct {
		r         *edgeRoute
		key       [2]int
		x, y0, y1 float64
	}
	var runs []along
	for k, r := range routes {
		r.order, r.hops, r.runHops = k, map[int][]float64{}, map[[2]int][]float64{}
		for i := 0; i+1 < len(r.pts); i++ {
			a, b := r.pts[i], r.pts[i+1]
			if mid, ok := r.track[i]; ok {
				runs = append(runs, along{r, [2]int{i, 0}, a.x, a.y, mid}, along{r, [2]int{i, 1}, b.x, mid, b.y})
			} else {
				runs = append(runs, along{r, [2]int{i, 0}, a.x, a.y, b.y})
			}
		}
	}
	h, corner := l.u(6), l.u(8)
	for _, r := range routes {
		for i := 0; i+1 < len(r.pts); i++ {
			mid, ok := r.track[i]
			if !ok {
				continue
			}
			lo, hi := math.Min(r.pts[i].x, r.pts[i+1].x), math.Max(r.pts[i].x, r.pts[i+1].x)
			for _, v := range runs {
				// Mostek potrzebuje miejsca: z dala od narożników obu krawędzi.
				if v.r == r || v.x < lo+corner+h || v.x > hi-corner-h || mid < v.y0+corner+h || mid > v.y1-corner-h {
					continue
				}
				if r.order > v.r.order {
					r.hops[i] = append(r.hops[i], v.x)
				} else {
					v.r.runHops[v.key] = append(v.r.runHops[v.key], mid)
				}
			}
		}
	}
	// Mostki w kolejności wzdłuż odcinka; zbyt bliskie siebie zlewają się w jeden.
	thin := func(at []float64) []float64 {
		sort.Float64s(at)
		kept := at[:0]
		for _, x := range at {
			if len(kept) == 0 || x-kept[len(kept)-1] > 2*h+l.u(2) {
				kept = append(kept, x)
			}
		}
		return kept
	}
	for _, r := range routes {
		for i, at := range r.hops {
			r.hops[i] = thin(at)
		}
		for k, at := range r.runHops {
			r.runHops[k] = thin(at)
		}
	}
}

// headLength mówi, o ile zakończenie skraca linię krawędzi.
func (l *layouter) headLength(h graph.Head) float64 {
	switch h {
	case graph.ArrowHead:
		return l.u(12) * 0.7
	case graph.CircleHead:
		return l.u(9)
	case graph.CrossHead:
		return l.u(8) // linia kończy się w środku krzyżyka
	case graph.Triangle:
		return l.u(14)
	case graph.DiamondFilled, graph.DiamondHollow:
		return l.u(18)
	}
	return 0
}

// assignTracks rozdziela odcinki poprzeczne krawędzi łamanych na tory. Odcinki leżące w tym samym
// odstępie między warstwami, których zakresy się pokrywają, dostają różne tory, w kolejności,
// która nie dodaje przecięć: przy biegu w stronę +x krawędź zaczynająca się wcześniej skręca
// później (bliżej następnej warstwy), przy biegu w stronę −x odwrotnie.
func (l *layouter) assignTracks(routes []*edgeRoute) {
	type jog struct {
		r      *edgeRoute
		seg    int
		a, b   routePoint
		lo, hi float64
		track  int
		// Zapas od końców odcinka: przy grocie większy, żeby między narożnikiem a grotem został
		// kawałek prostej linii, a grot nie siedział tuż przy zakręcie.
		padA, padB float64
	}
	var jogs []*jog
	for _, r := range routes {
		r.track = map[int]float64{}
		if r.flat {
			r.track[0] = r.pts[0].y // odcinek poprzeczny na wysokości pól, poza odstępem warstw
			continue
		}
		for i := 0; i+1 < len(r.pts); i++ {
			a, b := r.pts[i], r.pts[i+1]
			if i == 0 && r.side {
				r.track[0] = a.y // odcinek poprzeczny na wysokości wierzchołka, poza odstępem warstw
				continue
			}
			if math.Abs(a.x-b.x) >= straightTolerance {
				j := &jog{r: r, seg: i, a: a, b: b, lo: math.Min(a.x, b.x) - l.u(5), hi: math.Max(a.x, b.x) + l.u(5), padA: l.u(7), padB: l.u(7)}
				if i == 0 && r.topHead != "" && r.topHead != graph.NoHead {
					j.padA = l.u(18)
				}
				if i+2 == len(r.pts) && r.bottomHead != "" && r.bottomHead != graph.NoHead {
					j.padB = l.u(18)
				}
				jogs = append(jogs, j)
			}
		}
	}
	sort.SliceStable(jogs, func(i, k int) bool { return jogs[i].a.y < jogs[k].a.y })
	for start := 0; start < len(jogs); {
		// Grupa odcinków z jednego odstępu między warstwami: ich zakresy wzdłuż osi warstw się pokrywają.
		end, maxA, minB := start+1, jogs[start].a.y, jogs[start].b.y
		for end < len(jogs) && jogs[end].a.y < minB-1 {
			maxA, minB = math.Max(maxA, jogs[end].a.y), math.Min(minB, jogs[end].b.y)
			end++
		}
		group := append([]*jog(nil), jogs[start:end]...)
		start = end
		sort.SliceStable(group, func(i, k int) bool {
			p, q := group[i], group[k]
			pr, qr := p.b.x > p.a.x, q.b.x > q.a.x
			switch {
			case pr != qr:
				return pr
			case pr && p.a.x != q.a.x:
				return p.a.x < q.a.x
			case pr:
				return p.b.x < q.b.x
			case p.a.x != q.a.x:
				return p.a.x > q.a.x
			}
			return p.b.x > q.b.x
		})
		tracks := 1
		for i, j := range group {
			for _, prev := range group[:i] {
				m := l.u(5)
				if (prev.b.x > prev.a.x) != (j.b.x > j.a.x) {
					m = 0 // odcinki rozchodzące się w przeciwne strony mogą leżeć na jednej wysokości
				}
				if prev.lo+l.u(5)-m < j.hi-l.u(5)+m && j.lo+l.u(5)-m < prev.hi-l.u(5)+m {
					j.track = max(j.track, prev.track+1)
				}
			}
			tracks = max(tracks, j.track+1)
		}
		// Odcinek, który zaczyna się w osi, w której inny się kończy, biegnie wyżej od niego: inaczej
		// zejście jednej krawędzi i dojście drugiej leżą na sobie i wyglądają jak połączenie pól.
		for round := 0; round < 2*len(group); round++ {
			changed := false
			for _, p := range group {
				for _, q := range group {
					if p != q && math.Abs(p.a.x-q.b.x) < straightTolerance && p.track <= q.track {
						p.track = q.track + 1
						changed = true
					}
				}
			}
			for i, j := range group {
				for _, prev := range group[:i] {
					m := l.u(5)
					if (prev.b.x > prev.a.x) != (j.b.x > j.a.x) {
						m = 0
					}
					if prev.lo+l.u(5)-m < j.hi-l.u(5)+m && j.lo+l.u(5)-m < prev.hi-l.u(5)+m && j.track == prev.track {
						j.track++
						changed = true
					}
				}
			}
			if !changed {
				break
			}
		}
		for _, j := range group {
			tracks = max(tracks, j.track+1)
		}
		lo, hi := maxA+l.u(7), minB-l.u(7)
		for _, j := range group {
			lo, hi = math.Max(lo, j.a.y+j.padA), math.Min(hi, j.b.y-j.padB)
		}
		if hi < lo {
			// Za mało miejsca na zapasy przy narożnikach i grotach: tory zajmują cały odstęp, żeby
			// odcinki różnych krawędzi nie wypadły na jednej wysokości.
			lo, hi = maxA+l.u(3), minB-l.u(3)
			if hi < lo || tracks == 1 {
				lo, hi = (maxA+minB)/2, (maxA+minB)/2
			}
		}
		step := 0.0
		if tracks > 1 {
			// Tory są na tyle daleko od siebie, żeby mostek zmieścił się obok narożnika sąsiedniej krawędzi.
			step = math.Min(l.u(20), (hi-lo)/float64(tracks-1))
		}
		for _, j := range group {
			// Tor zerowy leży najbliżej następnej warstwy.
			j.r.track[j.seg] = (lo+hi)/2 + (float64(tracks-1)/2-float64(j.track))*step
		}
	}
}

// drawEdge rysuje krawędź: w paśmie warstwy biegnie prosto; między pasmami skręca dwoma kątami
// prostymi z zaokrąglonymi narożnikami albo, na życzenie, łagodnym łukiem.
func (l *layouter) drawEdge(route *edgeRoute) {
	r := route
	e, pts := r.e, r.pts
	from := len(l.sc.Items)
	defer func() { l.repaint(from, len(l.sc.Items), e.e.Paint, colorLine) }()
	if r.flat {
		l.drawFlatEdge(r)
		return
	}
	l.headAt(r.topHead, r.top.x, r.top.y, 0, -1)
	l.headAt(r.bottomHead, r.bottom.x, r.bottom.y, 0, 1)
	// Opisy przy końcach (liczność): obok linii, tuż przy polu.
	topLabel, bottomLabel := e.e.StartLabel, e.e.EndLabel
	if e.rev {
		topLabel, bottomLabel = bottomLabel, topLabel
	}
	for _, end := range []struct {
		text string
		at   routePoint
		dir  float64
	}{{topLabel, r.top, 1}, {bottomLabel, r.bottom, -1}} {
		if end.text == "" {
			continue
		}
		w := l.width(end.text)
		p := l.point(end.at.x+l.u(8)+w/2, end.at.y+end.dir*l.u(13))
		if !l.vertical {
			p = l.point(end.at.x+l.u(13), end.at.y+end.dir*(l.u(8)+w/2))
		}
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "edge-end-label", X: p.X, Y: p.Y + 0.35*l.fs,
			W: w, H: l.lh, Text: end.text, Anchor: "middle", Fill: colorText})
	}

	segments := []scene.Segment{{Cmd: 'M', To: l.point(pts[0].x, pts[0].y)}}
	// run prowadzi krawędź wzdłuż osi warstw do punktu (x, y), z mostkami nad krawędziami
	// narysowanymi wcześniej, które w tym miejscu biegną w poprzek.
	run := func(x, y float64, key [2]int) {
		const q = 0.55
		hh := l.u(6)
		for _, at := range route.runHops[key] {
			segments = append(segments,
				scene.Segment{Cmd: 'L', To: l.point(x, at-hh)},
				scene.Segment{Cmd: 'C', C1: l.point(x-hh*q, at-hh), C2: l.point(x-hh, at-hh*q), To: l.point(x-hh, at)},
				scene.Segment{Cmd: 'C', C1: l.point(x-hh, at+hh*q), C2: l.point(x-hh*q, at+hh), To: l.point(x, at+hh)})
		}
		segments = append(segments, scene.Segment{Cmd: 'L', To: l.point(x, y)})
	}
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		if i == 0 && route.side {
			// Wyjście z bocznego wierzchołka: w poprzek do miejsca skrętu, jeden narożnik, dalej prosto.
			dir := 1.0
			if b.x < a.x {
				dir = -1
			}
			rad := math.Max(0, math.Min(l.u(8), math.Min(math.Abs(b.x-a.x)/2, b.y-a.y)))
			hops := append([]float64(nil), route.hops[0]...)
			if dir < 0 {
				sort.Sort(sort.Reverse(sort.Float64Slice(hops)))
			}
			for _, x := range hops {
				const q = 0.55
				hh := l.u(6)
				segments = append(segments,
					scene.Segment{Cmd: 'L', To: l.point(x-dir*hh, a.y)},
					scene.Segment{Cmd: 'C', C1: l.point(x-dir*hh, a.y-hh*q), C2: l.point(x-dir*hh*q, a.y-hh), To: l.point(x, a.y-hh)},
					scene.Segment{Cmd: 'C', C1: l.point(x+dir*hh*q, a.y-hh), C2: l.point(x+dir*hh, a.y-hh*q), To: l.point(x+dir*hh, a.y)})
			}
			const k = 0.45
			segments = append(segments,
				scene.Segment{Cmd: 'L', To: l.point(b.x-dir*rad, a.y)},
				scene.Segment{Cmd: 'C', C1: l.point(b.x-dir*rad*k, a.y), C2: l.point(b.x, a.y+rad*k), To: l.point(b.x, a.y+rad)})
			run(b.x, b.y, [2]int{0, 1})
			continue
		}
		if math.Abs(a.x-b.x) < straightTolerance {
			run(b.x, b.y, [2]int{i, 0})
			continue
		}
		mid := (a.y + b.y) / 2
		if l.o.CurvedEdges {
			segments = append(segments, scene.Segment{Cmd: 'C', C1: l.point(a.x, mid), C2: l.point(b.x, mid), To: l.point(b.x, b.y)})
			continue
		}
		if y, ok := r.track[i]; ok {
			mid = y
		}
		r := math.Max(0, math.Min(l.u(8), math.Min(math.Abs(b.x-a.x)/2, math.Min(mid-a.y, b.y-mid))))
		dir := 1.0
		if b.x < a.x {
			dir = -1
		}
		// Narożnik to ćwiartka okręgu przybliżona krzywą: punkty kontrolne w 0,55 promienia od końców.
		const k = 0.45
		run(a.x, mid-r, [2]int{i, 0})
		segments = append(segments,
			scene.Segment{Cmd: 'C', C1: l.point(a.x, mid-r*k), C2: l.point(a.x+dir*r*k, mid), To: l.point(a.x+dir*r, mid)})
		// Mostki nad krawędziami, które ten odcinek przecina: półokrąg z dwóch ćwiartek.
		hops := append([]float64(nil), route.hops[i]...)
		if dir < 0 {
			sort.Sort(sort.Reverse(sort.Float64Slice(hops)))
		}
		for _, x := range hops {
			const h, q = 6.0, 0.55
			hh := l.u(h)
			segments = append(segments,
				scene.Segment{Cmd: 'L', To: l.point(x-dir*hh, mid)},
				scene.Segment{Cmd: 'C', C1: l.point(x-dir*hh, mid-hh*q), C2: l.point(x-dir*hh*q, mid-hh), To: l.point(x, mid-hh)},
				scene.Segment{Cmd: 'C', C1: l.point(x+dir*hh*q, mid-hh), C2: l.point(x+dir*hh, mid-hh*q), To: l.point(x+dir*hh, mid)})
		}
		segments = append(segments,
			scene.Segment{Cmd: 'L', To: l.point(b.x-dir*r, mid)},
			scene.Segment{Cmd: 'C', C1: l.point(b.x-dir*r*k, mid), C2: l.point(b.x, mid+r*k), To: l.point(b.x, mid+r)})
		run(b.x, b.y, [2]int{i, 1})
	}
	width, dash := l.strokeOf(e.e)
	l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "edge", Segments: segments,
		Stroke: colorLine, StrokeWidth: width, Dash: dash})
	if e.label != nil {
		if route.labelAt != nil {
			l.drawLabel(e.label.text, route.labelAt.x, route.labelAt.y)
		} else {
			l.drawLabel(e.label.text, e.label.x+e.label.textShift, e.label.y)
		}
	}
}

// drawLoop rysuje krawędź węzła do samego siebie po stronie +x pola.
func (l *layouter) drawLoop(it *item, e *graph.Edge, index int) {
	if e.Style == graph.Invisible {
		return
	}
	from := len(l.sc.Items)
	defer func() { l.repaint(from, len(l.sc.Items), e.Paint, colorLine) }()
	edge := it.nodeX() + it.nodeW()/2
	dy := math.Min(l.u(8), it.h/4) + float64(index)*l.u(3)
	reach := l.u(30)
	y0, y1 := it.y-dy, it.y+dy
	startX := edge + l.headAt(e.StartHead, edge, y0, -1, 0)
	endX := edge + l.headAt(e.EndHead, edge, y1, -1, 0)
	out := edge + l.u(12)
	segments := []scene.Segment{
		{Cmd: 'M', To: l.point(startX, y0)},
		{Cmd: 'L', To: l.point(out, y0)},
		{Cmd: 'C', C1: l.point(out+reach, y0), C2: l.point(out+reach, y1), To: l.point(out, y1)},
		{Cmd: 'L', To: l.point(endX, y1)},
	}
	width, dash := l.strokeOf(e)
	l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "edge", Segments: segments,
		Stroke: colorLine, StrokeWidth: width, Dash: dash})
	if len(e.Label) > 0 {
		lines := l.wrap(e.Label, l.u(l.baseWrap())*l.narrow())
		w, _ := l.abstract(l.textSize(lines))
		l.drawLabel(lines, edge+l.u(40)+w/2, it.y)
	}
}

// bar mówi, czy pole jest belką rozwidlenia albo złączenia.
func bar(it *item) bool {
	return it.kind == kindNode && (it.node.Shape == graph.Fork || it.node.Shape == graph.Join)
}

// subgraphIndex zwraca numer podgrafu w kolejności ze źródła.
func (l *layouter) subgraphIndex(id string) int {
	for i, s := range l.g.Subgraphs {
		if s.ID == id {
			return i
		}
	}
	return 0
}

// crossing zapamiętuje punkt, w którym krawędź przechodzi przez ramkę podgrafu lv: może on przesuwać
// się wzdłuż boku ramki (w jej obrębie), bo na rysunku to tylko miejsce przecięcia linii z ramką.
func (l *layouter) crossing(lv *level, p routePoint) routePoint {
	if l.frameCross == nil {
		l.frameCross = map[routePoint]bool{}
	}
	l.frameCross[p] = true
	return l.movable(p, lv.offsetX+l.u(6), lv.offsetX+lv.w-l.u(6))
}

// movable zapamiętuje punkt krawędzi, który może przesunąć się w poprzek osi warstw w zakresie lo..hi.
func (l *layouter) movable(p routePoint, lo, hi float64) routePoint {
	if l.free == nil {
		l.free = map[routePoint][2]float64{}
	}
	l.free[p] = [2]float64{lo, hi}
	return p
}

// dummyRoom zwraca zakres, w którym punkt pośredni krawędzi może stanąć bez zbliżenia do sąsiadów
// w swojej warstwie bardziej niż zwykły odstęp.
func (l *layouter) dummyRoom(lv *level, d *item) (lo, hi float64) {
	lo, hi = math.Inf(-1), math.Inf(1)
	if d.rank < 0 || d.rank >= len(lv.ranks) {
		return d.x, d.x
	}
	row := lv.ranks[d.rank]
	gap := func(o *item) float64 {
		if o.kind == kindDummy {
			return l.u(10)
		}
		return l.u(8)
	}
	for _, o := range row {
		if o == d {
			continue
		}
		w := o.w
		if o.thin {
			w = l.u(8)
		}
		if o.x < d.x {
			lo = math.Max(lo, o.x+o.textShift+w/2+gap(o))
		} else {
			hi = math.Min(hi, o.x+o.textShift-w/2-gap(o))
		}
	}
	return lo, hi
}

// snapCrossings usuwa drobne uskoki przy przejściu przez ramkę: odcinek w poprzek ramki przesuwa się
// w oś odcinka przed nim albo za nim, jeśli zostaje wtedy mniej drobnych uskoków (zero albo wyraźny
// skręt są dobre), a nowe miejsce leży w obrębie boku ramki.
func (l *layouter) snapCrossings(pts []routePoint) {
	clear := l.u(16)
	small := func(d float64) int {
		if d = math.Abs(d); d > 0.5 && d < clear {
			return 1
		}
		return 0
	}
	for i := 1; i < len(pts)-1; {
		bounds, ok := l.free[pts[i]]
		if !ok {
			i++
			continue
		}
		// Odcinek do przesunięcia: kolejne punkty w jednej osi, wszystkie ruchome; zakres to część
		// wspólna ich zakresów. Przesuwa się tylko odcinek, który przechodzi przez ramkę.
		j, frame := i, l.frameCross[pts[i]]
		for j+1 < len(pts)-1 && pts[j+1].x == pts[i].x {
			b, free := l.free[pts[j+1]]
			if !free {
				break
			}
			bounds = [2]float64{math.Max(bounds[0], b[0]), math.Min(bounds[1], b[1])}
			frame = frame || l.frameCross[pts[j+1]]
			j++
		}
		if !frame {
			i = j + 1
			continue
		}
		x, prev, next := pts[i].x, pts[i-1].x, pts[j+1].x
		// passes mówi, czy przesunięcie do c mija inne przejście przez tę samą ramkę albo się do niego
		// zbliża: kolejność krawędzi na boku ramki ma zostać ta sama.
		passes := func(c float64) bool {
			lo, hi := math.Min(x, c)-l.u(6), math.Max(x, c)+l.u(6)
			for k := i; k <= j; k++ {
				if !l.frameCross[pts[k]] {
					continue
				}
				for q := range l.frameCross {
					if math.Abs(q.y-pts[k].y) < 0.5 && q.x != x && q.x > lo && q.x < hi {
						return true
					}
				}
			}
			return false
		}
		best, worst := x, small(x-prev)+small(next-x)
		for _, c := range []float64{prev, next} {
			if c < bounds[0] || c > bounds[1] || passes(c) {
				continue
			}
			if n := small(c-prev) + small(next-c); n < worst {
				best, worst = c, n
			}
		}
		// Odcinek przed przejściem złożony z samych punktów pośrednich (bez stałego pola) może przesunąć
		// się w oś przejścia: krawędź skręca wtedy raz, przy początku, i dochodzi do ramki prosto.
		if best == x && small(x-prev) == 1 {
			k := i - 1
			for k > 0 && pts[k-1].x == pts[k].x {
				k--
			}
			lo, hi := math.Inf(-1), math.Inf(1)
			movable := k > 0
			for m := k; m < i && movable; m++ {
				b, free := l.free[pts[m]]
				if !free {
					movable = false
					break
				}
				lo, hi = math.Max(lo, b[0]), math.Min(hi, b[1])
			}
			if movable && x >= lo && x <= hi && small(x-pts[k-1].x) == 0 {
				for m := k; m < i; m++ {
					old := pts[m]
					pts[m].x = x
					l.free[pts[m]] = l.free[old]
					delete(l.free, old)
				}
			}
		}
		if best != x {
			for k := i; k <= j; k++ {
				old := pts[k]
				pts[k].x = best
				// Punkt zostaje ruchomy pod nowym położeniem: drugi przebieg może go jeszcze dołączyć.
				l.free[pts[k]] = l.free[old]
				delete(l.free, old)
				if l.frameCross[old] {
					delete(l.frameCross, old)
					l.frameCross[pts[k]] = true
				}
			}
		}
		i = j + 1
	}
}

// withoutTurnBacks usuwa punkt, w którym krawędź zawraca w tej samej osi (np. granica warstwy leży za
// brzegiem pola węższego niż jego warstwa): odcinek biegnie wtedy wprost do następnego punktu.
func withoutTurnBacks(pts []routePoint) []routePoint {
	out := make([]routePoint, 0, len(pts))
	for i, p := range pts {
		if n := len(out); n > 0 && i+1 < len(pts) {
			prev, next := out[n-1], pts[i+1]
			if prev.x == p.x && p.x == next.x && (p.y-prev.y)*(next.y-p.y) < 0 {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// drawFlatEdge rysuje płaską krawędź (LAY-027): prosty odcinek w poprzek osi warstw, z mostkami nad
// krawędziami biegnącymi wzdłuż osi, i zakończenia skierowane w boki pól.
func (l *layouter) drawFlatEdge(r *edgeRoute) {
	e := r.e
	from := len(l.sc.Items)
	defer func() { l.repaint(from, len(l.sc.Items), e.e.Paint, colorLine) }()
	a, b := r.pts[0], r.pts[len(r.pts)-1]
	dir := 1.0
	if b.x < a.x {
		dir = -1
	}
	l.headAt(r.topHead, r.top.x, r.top.y, -dir, 0)
	l.headAt(r.bottomHead, r.bottom.x, r.bottom.y, dir, 0)
	segments := []scene.Segment{{Cmd: 'M', To: l.point(a.x, a.y)}}
	hops := append([]float64(nil), r.hops[0]...)
	if dir < 0 {
		sort.Sort(sort.Reverse(sort.Float64Slice(hops)))
	}
	for _, x := range hops {
		const q = 0.55
		hh := l.u(6)
		segments = append(segments,
			scene.Segment{Cmd: 'L', To: l.point(x-dir*hh, a.y)},
			scene.Segment{Cmd: 'C', C1: l.point(x-dir*hh, a.y-hh*q), C2: l.point(x-dir*hh*q, a.y-hh), To: l.point(x, a.y-hh)},
			scene.Segment{Cmd: 'C', C1: l.point(x+dir*hh*q, a.y-hh), C2: l.point(x+dir*hh, a.y-hh*q), To: l.point(x+dir*hh, a.y)})
	}
	segments = append(segments, scene.Segment{Cmd: 'L', To: l.point(b.x, b.y)})
	width, dash := l.strokeOf(e.e)
	l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "edge", Segments: segments,
		Stroke: colorLine, StrokeWidth: width, Dash: dash})
}

// alignChain (LAY-028) przesuwa punkty pośrednie krawędzi, stojące w jednej osi, w oś punktu
// zaczepienia jednego z końców, jeśli mają tam miejsce: krawędź biegnie wtedy prosto albo skręca
// raz, zamiast zejść z punktu zaczepienia do osi pól i wrócić (np. krawędź wsteczna między polami,
// których boki dzielą inne krawędzie). Punkty stojące już w osi jednego z końców zostają; przesunięte
// trafiają najpierw w oś dolnego końca, potem górnego.
// Etykieta przesuwa się razem z punktami, jeśli mieści się w wolnym miejscu warstwy; punkty w torze
// zostają w jego granicach.
func (l *layouter) alignChain(e *ledge) {
	if len(e.chain) == 0 || e.upperCont != nil || e.lowerCont != nil || e.u.kind != kindNode || e.v.kind != kindNode {
		return
	}
	at, same := e.chain[0].x, true
	for _, d := range e.chain {
		same = same && math.Abs(d.x-at) < straightTolerance
	}
	ux, vx := e.u.nodeX()+e.upperPort, e.v.nodeX()+e.lowerPort
	// Wyjście z bocznego wierzchołka rombu skręca zawsze: oś pierwszego odcinka jest dowolna, byle
	// poziomy odcinek od wierzchołka nie wszedł na sąsiednie pole w rzędzie rombu.
	sideLo, sideHi := math.Inf(-1), math.Inf(1)
	if e.sideExit != 0 {
		ux = math.NaN()
		vertex := e.u.nodeX() + e.sideExit*e.u.nodeW()/2
		if e.sideExit > 0 {
			sideLo = vertex + l.u(16)
		} else {
			sideHi = vertex - l.u(16)
		}
		for _, r := range e.lv.ranks {
			for _, o := range r {
				if o == e.u || o.rank != e.u.rank {
					continue
				}
				if e.sideExit > 0 && o.x > e.u.x {
					sideHi = math.Min(sideHi, o.x+o.textShift-o.w/2-l.u(10))
				}
				if e.sideExit < 0 && o.x < e.u.x {
					sideLo = math.Max(sideLo, o.x+o.textShift+o.w/2+l.u(10))
				}
			}
		}
	}
	// Wspólny zakres osi dla wszystkich punktów: każdy zostaje między swoimi sąsiadami w warstwie
	// (kolejność się nie zmienia, więc nie przybywa przecięć), a punkty w torze zostają w jego
	// granicach. Etykieta nie musi stać na środku linii: linia ma tylko przechodzić przez jej pole,
	// a pole mieścić się w wolnym miejscu warstwy.
	lo, hi := math.Inf(-1), math.Inf(1)
	type box = struct{ lo, hi float64 }
	boxes := map[*item]box{}
	ranges := make([]box, len(e.chain)) // zakres osi linii dla każdego punktu osobno
	for k, d := range e.chain {
		plo, phi := lo, hi
		lo, hi = math.Inf(-1), math.Inf(1)
		rlo, rhi := l.dummyRoom(e.lv, d)
		margin := l.u(8)
		if l.lanes != nil && e.lv == l.levels[""] {
			ln := l.lanes[d.lane]
			rlo, rhi = math.Max(rlo, ln.lo+l.u(4)), math.Min(rhi, ln.hi-l.u(4))
		}
		if d.kind == kindLabel && !d.thin {
			if rhi-rlo < d.w {
				return // etykieta nie mieści się w żadnym położeniu
			}
			boxes[d] = box{rlo, rhi}
			margin = l.u(10)
		}
		lo, hi = math.Max(lo, rlo+margin-l.u(8)), math.Min(hi, rhi-margin+l.u(8))
		if d.kind == kindLabel && !d.thin {
			lo, hi = math.Max(lo, rlo+l.u(10)), math.Min(hi, rhi-l.u(10))
		}
		if k == 0 {
			lo, hi = math.Max(lo, sideLo), math.Min(hi, sideHi)
		}
		ranges[k] = box{lo, hi}
		lo, hi = math.Max(plo, lo), math.Min(phi, hi)
	}
	fits := func(x float64) bool { return x >= lo-0.5 && x <= hi+0.5 }
	move := func(x float64) {
		for _, d := range e.chain {
			d.x, d.textShift = x, 0
			if b, ok := boxes[d]; ok {
				c := math.Max(b.lo+d.w/2, math.Min(b.hi-d.w/2, x))
				d.textShift = c - x
			}
		}
		l.alignEnd(e, x)
	}
	if same {
		for _, x := range []float64{vx, ux} {
			if math.Abs(x-at) < straightTolerance {
				l.alignEnd(e, x) // już w osi jednego z końców; drugi koniec może dojść bez uskoku
				return
			}
		}
	}
	// Najpierw oś dolnego końca: jedyny skręt wypada wtedy przy początku, a grot dochodzi prosto.
	for _, x := range []float64{vx, ux} {
		if fits(x) {
			move(x)
			return
		}
	}
	if same {
		l.alignEnd(e, at) // punkty nie mają dokąd przejść: końce, które mogą, dochodzą do nich prosto
		return
	}
	// Punkty w różnych osiach (każda warstwa po swojemu, także pod etykietą): ułożenie z najmniejszą
	// liczbą załamań, licząc skręt przy początku i przy końcu; przy remisie zostaje obecne.
	if xs := l.fewestBends(e, ranges, ux, vx); xs != nil {
		for k, d := range e.chain {
			d.x, d.textShift = xs[k], 0
			if b, ok := boxes[d]; ok {
				d.textShift = math.Max(b.lo+d.w/2, math.Min(b.hi-d.w/2, xs[k])) - xs[k]
			}
		}
	}
}

// fewestBends szuka położeń punktów pośrednich krawędzi z najmniejszą liczbą załamań. Kolejne punkty
// tworzą odcinek prosty, dopóki ich zakresy mają część wspólną; odcinki są budowane od dołu, a dolny
// może zostać przy osi dolnego końca (grot dochodzi wtedy prosto) albo sięgać jak najwyżej. Każdy
// odcinek staje jak najbliżej następnego, ale nie o drobny uskok (przejście co najmniej u(16)); górny
// w osi górnego końca, jeśli może. Zwraca nil, gdy żadne ułożenie nie ma mniej załamań niż obecne.
func (l *layouter) fewestBends(e *ledge, ranges []struct{ lo, hi float64 }, ux, vx float64) []float64 {
	n := len(e.chain)
	bends := func(xs []float64) int {
		c, prev := 0, ux
		for _, x := range append(append([]float64(nil), xs...), vx) {
			if !math.IsNaN(prev) && math.Abs(x-prev) >= straightTolerance {
				c++
			}
			prev = x
		}
		return c
	}
	now := make([]float64, n)
	for k, d := range e.chain {
		now[k] = d.x
	}
	for _, r := range ranges {
		if r.lo > r.hi {
			return nil // punkt nie ma żadnego miejsca: zostaje, jak był
		}
	}
	clamp := func(lo, hi, x float64) float64 { return math.Max(lo, math.Min(hi, x)) }
	plan := func(keepV bool) []float64 {
		xs := make([]float64, n)
		end := n
		next := vx
		first := true
		for end > 0 {
			lo, hi := ranges[end-1].lo, ranges[end-1].hi
			from := end - 1
			for from > 0 {
				nlo, nhi := math.Max(lo, ranges[from-1].lo), math.Min(hi, ranges[from-1].hi)
				if nlo > nhi || (first && keepV && (vx < nlo || vx > nhi) && vx >= lo && vx <= hi) {
					break
				}
				lo, hi, from = nlo, nhi, from-1
			}
			x := clamp(lo, hi, next)
			if d := math.Abs(x - next); !first && d >= straightTolerance && d < l.u(16) {
				for _, c := range []float64{next - l.u(16), next + l.u(16)} {
					if c >= lo && c <= hi {
						x = c
						break
					}
				}
			}
			if from == 0 && ux >= lo && ux <= hi && math.Abs(x-next) >= straightTolerance {
				x = ux // górny odcinek w osi górnego końca: jego załamanie i tak jest potrzebne
			}
			for k := from; k < end; k++ {
				xs[k] = x
			}
			next, end, first = x, from, false
		}
		return xs
	}
	best, score := []float64(nil), bends(now)
	for _, keepV := range []bool{true, false} {
		if xs := plan(keepV); bends(xs) < score {
			best, score = xs, bends(xs)
		}
	}
	return best
}

// alignEnd usuwa uskok przy końcu krawędzi: punkt zaczepienia przesuwa się w oś x jej punktów
// pośrednich, jeśli krawędź jest jedyną na tym boku pola, a oś leży w zasięgu boku.
func (l *layouter) alignEnd(e *ledge, x float64) {
	alone := func(it *item, upper bool) bool {
		n := 0
		for _, o := range e.lv.edges {
			if o.flat || o.e.Style == graph.Invisible {
				continue
			}
			if (upper && o.u == it && o.upperCont == nil) || (!upper && o.v == it && o.lowerCont == nil) {
				n++
			}
		}
		return n == 1
	}
	for _, upper := range []bool{true, false} {
		if upper && e.sideExit != 0 {
			continue
		}
		it, port := e.v, &e.lowerPort
		if upper {
			it, port = e.u, &e.upperPort
		}
		off := x - it.nodeX()
		jog := math.Abs(off - *port)
		if jog >= straightTolerance && math.Abs(off) <= l.portRange(it)/2 && alone(it, upper) {
			*port = off
		}
	}
}

// routesWithLabels wyznacza przebiegi krawędzi poziomu pionowego razem z miejscami etykiet (LAY-031).
// Etykieta stoi na odcinku poziomym krawędzi, bo tam czyta się ją najlepiej: najpierw krawędzie są
// prostowane tak, jakby etykiety nie zajmowały miejsca w warstwach, potem każda etykieta szuka
// miejsca na najdłuższym odcinku poziomym swojej krawędzi. Krawędzie, których etykiety nigdzie się
// nie zmieściły, są prostowane jeszcze raz, już z etykietą w warstwie.
func (l *layouter) routesWithLabels(lv *level) {
	type spot struct{ x, shift float64 }
	saved := map[*item]spot{}
	var labels []*item
	for _, e := range lv.edges {
		for _, d := range e.chain {
			saved[d] = spot{d.x, d.textShift}
		}
		if e.label != nil && !e.part {
			labels = append(labels, e.label)
			e.label.thin = true
		}
	}
	lv.routes = l.routes(lv)
	l.assignTracks(lv.routes)
	failed := l.placeLabels(lv)
	if len(failed) == 0 {
		return
	}
	for d, sp := range saved {
		d.x, d.textShift = sp.x, sp.shift
	}
	for _, d := range failed {
		d.thin = false
	}
	lv.routes = l.routes(lv)
	l.assignTracks(lv.routes)
	l.placeLabels(lv)
	_ = labels
}

// placeLabels stawia etykiety krawędzi poziomu na odcinkach poziomych: na najdłuższym, na którym
// pole etykiety mieści się z dala od narożników i nie zachodzi na linie innych krawędzi, pola ani
// inne etykiety; najpierw na środku odcinka, potem coraz dalej od niego. Zwraca etykiety, które
// zostają w swoich warstwach.
func (l *layouter) placeLabels(lv *level) []*item {
	type seg struct {
		r          *edgeRoute
		a, b       routePoint
		horizontal bool
	}
	var segs []seg
	for _, r := range lv.routes {
		r.labelAt = nil
		p := r.pts
		for i := 0; i+1 < len(p); i++ {
			a, b := p[i], p[i+1]
			switch {
			case r.flat:
				segs = append(segs, seg{r, a, b, true})
			case i == 0 && r.side:
				segs = append(segs, seg{r, a, routePoint{b.x, a.y}, true}, seg{r, routePoint{b.x, a.y}, b, false})
			case math.Abs(a.x-b.x) < straightTolerance:
				segs = append(segs, seg{r, a, b, false})
			default:
				mid, ok := r.track[i]
				if !ok {
					mid = (a.y + b.y) / 2
				}
				segs = append(segs, seg{r, a, routePoint{a.x, mid}, false}, seg{r, routePoint{a.x, mid}, routePoint{b.x, mid}, true},
					seg{r, routePoint{b.x, mid}, b, false})
			}
		}
	}
	type rect struct{ x0, y0, x1, y1 float64 }
	var obstacles []rect
	for _, row := range lv.ranks {
		for _, it := range row {
			if it.real() {
				obstacles = append(obstacles, rect{it.x - it.w/2 - l.u(4), it.y - it.h/2 - l.u(4), it.x + it.w/2 + l.u(4), it.y + it.h/2 + l.u(4)})
			}
		}
	}
	labelBox := func(d *item, x, y float64) rect { return rect{x - d.w/2, y - d.h/2, x + d.w/2, y + d.h/2} }
	hits := func(b rect, own *edgeRoute) bool {
		for _, o := range obstacles {
			if b.x0 < o.x1 && o.x0 < b.x1 && b.y0 < o.y1 && o.y0 < b.y1 {
				return true
			}
		}
		for _, s := range segs {
			if s.r == own {
				continue
			}
			x0, x1 := math.Min(s.a.x, s.b.x), math.Max(s.a.x, s.b.x)
			y0, y1 := math.Min(s.a.y, s.b.y), math.Max(s.a.y, s.b.y)
			if x0 < b.x1+l.u(2) && b.x0-l.u(2) < x1 && y0 < b.y1+l.u(2) && b.y0-l.u(2) < y1 {
				return true
			}
		}
		return false
	}
	var failed []*item
	placed := map[*edgeRoute]rect{}
	// Etykiety stojące w warstwach (jeszcze nie postawione albo bez miejsca) też są przeszkodami.
	inRank := func(skip *edgeRoute) []rect {
		var out []rect
		for _, r := range lv.routes {
			if r == skip || r.e.label == nil {
				continue
			}
			if b, ok := placed[r]; ok {
				out = append(out, b)
			} else if !r.e.label.thin {
				d := r.e.label
				out = append(out, labelBox(d, d.x+d.textShift, d.y))
			}
		}
		return out
	}
	for _, r := range lv.routes {
		d := r.e.label
		if d == nil {
			continue
		}
		var own []seg
		for _, s := range segs {
			if s.r == r && s.horizontal {
				own = append(own, s)
			}
		}
		sort.SliceStable(own, func(i, k int) bool { return math.Abs(own[i].b.x-own[i].a.x) > math.Abs(own[k].b.x-own[k].a.x) })
		done := false
		for _, s := range own {
			lo, hi := math.Min(s.a.x, s.b.x)+l.u(14)+d.w/2, math.Max(s.a.x, s.b.x)-l.u(14)-d.w/2
			if lo > hi {
				continue
			}
			c := (lo + hi) / 2
			for k := 0; !done && float64(k)*l.u(8) <= (hi-lo)/2+0.5; k++ {
				for _, x := range []float64{c - float64(k)*l.u(8), c + float64(k)*l.u(8)} {
					x = math.Max(lo, math.Min(hi, x))
					b := labelBox(d, x, s.a.y)
					bad := hits(b, r)
					for _, o := range inRank(r) {
						bad = bad || (b.x0 < o.x1 && o.x0 < b.x1 && b.y0 < o.y1 && o.y0 < b.y1)
					}
					if !bad {
						r.labelAt = &routePoint{x, s.a.y}
						placed[r] = b
						done = true
						break
					}
				}
			}
			if done {
				break
			}
		}
		if !done {
			failed = append(failed, d)
		}
	}
	return failed
}
