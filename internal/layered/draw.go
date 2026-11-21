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
)

const shapeNote graph.Shape = "note"

func (l *layouter) wrapLimit(shape graph.Shape) float64 {
	if shape == graph.Rhombus || shape == graph.Circle || shape == graph.DoubleCircle {
		return l.u(150)
	}
	return l.u(200)
}

// nodeSize zwraca rzeczywisty rozmiar kształtu mieszczącego podany tekst.
func (l *layouter) nodeSize(shape graph.Shape, text []string) (w, h float64) {
	tw, th := l.textSize(text)
	padX, padY := l.u(20), l.u(12)
	w, h = math.Max(tw+2*padX, l.u(56)), math.Max(th+2*padY, l.u(44))
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
	case graph.Hexagon:
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
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: cx,
			Y: top + float64(k)*l.lh + l.lh/2 + 0.35*l.fs, W: l.width(s), H: l.lh, Text: s, Anchor: "middle", Fill: colorText})
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
	case graph.Rhombus, graph.Choice, graph.Circle, graph.DoubleCircle, graph.StateStart, graph.StateEnd:
		return 0
	case graph.Stadium:
		return w - it.h
	case graph.Hexagon:
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
			if e.e.Style == graph.Invisible {
				continue
			}
			next, prev := e.v.nodeX(), e.u.nodeX()
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
		step := math.Min(l.u(18), usable/float64(k))
		for i, en := range ends {
			off := (float64(i) - float64(k-1)/2) * step
			if en.upper {
				en.e.upperPort = off
			} else {
				en.e.lowerPort = off
			}
		}
	}
	for _, it := range order {
		assign(it, bottom[it])
		assign(it, top[it])
	}
}

func (l *layouter) subgraphIDs() []string {
	ids := make([]string, len(l.g.Subgraphs))
	for i, s := range l.g.Subgraphs {
		ids[i] = s.ID
	}
	return ids
}

func (l *layouter) drawLevel(lv *level) {
	if lv.sub != nil {
		cx, cy, w, h := l.rect(lv.offsetX+lv.w/2, lv.offsetY+lv.h/2, lv.w, lv.h)
		from := len(l.sc.Items)
		frame := scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "cluster",
			X: cx - w/2, Y: cy - h/2, W: w, H: h, RX: l.u(4), Stroke: colorFrame, StrokeWidth: 1}
		if lv.sub.Region {
			frame.Role, frame.Dash = "region", "5 4" // region współbieżny: ramka przerywana, bez tytułu
		}
		paint := lv.sub.Paint
		if paint.Fill != "" && paint.Fill != "none" {
			// Wnętrze ramki ma własne tło: podkłady etykiet w środku biorą jego kolor.
			frame.Fill = paint.Fill
			defer func(prev string) { l.bg = prev }(l.bg)
			l.bg = paint.Fill
		}
		l.sc.Add(frame)
		defer func() { l.repaint(from, from+1, paint, colorFrame) }()
		if len(lv.titleLines) > 0 {
			defer func(at int) { l.repaint(at, at+1+len(lv.titleLines), graph.Paint{Color: paint.Color}, "") }(len(l.sc.Items))
		}
		// Tytuł ma podkład w kolorze tła: krawędź wchodząca do ramki znika pod nim, zamiast go przekreślać.
		tw, th := l.textSize(lv.titleLines)
		tx, ty := cx-w/2+l.u(12)+tw/2, cy-h/2+l.u(6)+th/2
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "cluster-title-background",
			X: tx - tw/2 - l.u(4), Y: ty - th/2, W: tw + 2*l.u(4), H: th, Fill: l.labelBackground()})
		l.textBlock(lv.titleLines, tx, ty, "cluster-title")
	}
	for _, row := range lv.ranks {
		for _, it := range row {
			switch it.kind {
			case kindCluster:
				l.drawLevel(it.child)
			case kindNode:
				cx, cy, w, h := l.rect(it.nodeX(), it.y, it.nodeW(), it.h)
				l.drawNode(it.node.Shape, it.text, cx, cy, w, h, it.node.Paint)
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
	for _, e := range lv.edges {
		if !e.part {
			l.drawEdge(e)
		}
	}
}

// tone zwraca kolory węzła w trybie kolorowym: kolor mówi, czym węzeł jest w procesie.
func (l *layouter) tone(shape graph.Shape) (fill, stroke string) {
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
		a, hw := l.u(9), l.u(4.5)
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerActivation, Role: "arrowhead",
			Points: []scene.Point{at(0, 0), at(a, hw), at(a, -hw)}, Fill: colorLine})
		return a * 0.7
	case graph.CircleHead:
		r := l.u(4.5)
		c := at(r, 0)
		l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerActivation, Role: "arrowhead", X: c.X, Y: c.Y, R: r,
			Fill: "#ffffff", Stroke: colorLine, StrokeWidth: 1.5})
		return 2 * r
	case graph.CrossHead:
		r := l.u(4.5)
		for _, s := range []float64{-1, 1} {
			a, b := at(l.u(6)-r, s*r), at(l.u(6)+r, -s*r)
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
	if l.o.Background != "" {
		return l.o.Background
	}
	return "#ffffff"
}

// drawLabel rysuje etykietę krawędzi z podkładem w kolorze tła.
func (l *layouter) drawLabel(lines []string, x, y float64) {
	p := l.point(x, y)
	tw, th := l.textSize(lines)
	l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "edge-label-background",
		X: p.X - tw/2 - l.u(4), Y: p.Y - th/2, W: tw + 2*l.u(4), H: th, Fill: l.labelBackground()})
	l.textBlock(lines, p.X, p.Y, "edge-label")
}

type routePoint struct{ x, y float64 }

// route zwraca punkty krawędzi od górnego końca do dolnego. Koniec leżący głębiej, w podgrafie,
// jest prowadzony odcinkiem z układu tego podgrafu: od pola do punktu na ramce albo odwrotnie.
func (l *layouter) route(e *ledge) []routePoint {
	var pts []routePoint
	switch {
	case e.upperCont != nil:
		pts = l.route(e.upperCont)
	case e.u.kind == kindPort:
		border := e.lv.offsetY
		pts = append(pts, routePoint{e.u.x, border})
		if e.u.y-border > 0.5 {
			pts = append(pts, routePoint{e.u.x, e.u.y})
		}
	default:
		pts = append(pts, routePoint{e.u.nodeX() + e.upperPort, e.u.y + e.u.h/2})
	}
	for _, d := range e.chain {
		if d.band > 0 {
			pts = append(pts, routePoint{d.x, d.y - d.band/2}, routePoint{d.x, d.y + d.band/2})
		} else {
			pts = append(pts, routePoint{d.x, d.y})
		}
	}
	switch {
	case e.lowerCont != nil:
		pts = append(pts, l.route(e.lowerCont)...)
	case e.v.kind == kindPort:
		border := e.lv.offsetY + e.lv.h
		if border-e.v.y > 0.5 {
			pts = append(pts, routePoint{e.v.x, e.v.y})
		}
		pts = append(pts, routePoint{e.v.x, border})
	default:
		pts = append(pts, routePoint{e.v.nodeX() + e.lowerPort, e.v.y - e.v.h/2})
	}
	return pts
}

// drawEdge rysuje krawędź: w paśmie warstwy biegnie prosto, między pasmami łagodnym łukiem.
func (l *layouter) drawEdge(e *ledge) {
	if e.e.Style == graph.Invisible {
		return
	}
	topHead, bottomHead := e.e.StartHead, e.e.EndHead
	if e.rev {
		topHead, bottomHead = bottomHead, topHead
	}
	pts := l.route(e)
	last := len(pts) - 1
	from := len(l.sc.Items)
	defer func() { l.repaint(from, len(l.sc.Items), e.e.Paint, colorLine) }()
	pts[0].y += l.headAt(topHead, pts[0].x, pts[0].y, 0, -1)
	pts[last].y -= l.headAt(bottomHead, pts[last].x, pts[last].y, 0, 1)

	segments := []scene.Segment{{Cmd: 'M', To: l.point(pts[0].x, pts[0].y)}}
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		if math.Abs(a.x-b.x) < 0.5 {
			segments = append(segments, scene.Segment{Cmd: 'L', To: l.point(b.x, b.y)})
			continue
		}
		mid := (a.y + b.y) / 2
		segments = append(segments, scene.Segment{Cmd: 'C', C1: l.point(a.x, mid), C2: l.point(b.x, mid), To: l.point(b.x, b.y)})
	}
	width, dash := l.strokeOf(e.e)
	l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "edge", Segments: segments,
		Stroke: colorLine, StrokeWidth: width, Dash: dash})
	if e.label != nil {
		l.drawLabel(e.label.text, e.label.x, e.label.y)
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
		lines := l.wrap(e.Label, l.u(200))
		w, _ := l.abstract(l.textSize(lines))
		l.drawLabel(lines, edge+l.u(40)+w/2, it.y)
	}
}
