package block

import (
	"math"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // bloki w kolejnych tonach palety według kolumny
	Shadows    bool // cień pod blokami
}

const (
	colorText   = "#1b1f24"
	colorFill   = "#eef2f7"
	colorStroke = "#55657a"
	colorLine   = "#3b4754"
	colorFrame  = "#9aa5b1"
)

type layouter struct {
	o      Options
	fs, lh float64
	sc     *scene.Scene
	lines  map[*Item][]string
	box    map[*Item][4]float64 // x, y, szerokość, wysokość
	edges  bool                 // diagram ma połączenia
	label  float64              // szerokość najszerszej etykiety połączenia
}

func (l *layouter) u(v float64) float64    { return v * l.fs / 16 }
func (l *layouter) width(s string) float64 { return l.o.Measure(s, l.fs) }

// gap to odstęp między blokami: większy, gdy diagram ma połączenia, żeby zmieściły się groty
// i etykiety.
func (l *layouter) gap() float64 {
	if l.edges {
		return math.Max(l.u(46), l.label+l.u(34)) // etykieta połączenia mieści się między blokami
	}
	return l.u(12)
}
func (l *layouter) columns(g *Item) int {
	if g.Columns > 0 {
		return g.Columns
	}
	n := 0
	for _, c := range g.Children {
		n += c.Span
	}
	return max(n, 1)
}

// rows dzieli elementy bloku złożonego na rzędy siatki: element, który nie mieści się w reszcie
// rzędu, zaczyna następny.
func (l *layouter) rows(g *Item) [][]*Item {
	cols := l.columns(g)
	var rows [][]*Item
	used := cols
	for _, c := range g.Children {
		span := min(c.Span, cols)
		if used+span > cols {
			rows = append(rows, nil)
			used = 0
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], c)
		used += span
	}
	return rows
}

// natural zwraca rozmiar, którego element potrzebuje sam z siebie.
func (l *layouter) natural(it *Item) (w, h float64) {
	switch it.Shape {
	case Space:
		return l.u(40), l.u(40)
	case Group:
		cols := l.columns(it)
		colW, total := 0.0, 0.0
		rows := l.rows(it)
		for _, row := range rows {
			rowH := 0.0
			for _, c := range row {
				cw, ch := l.natural(c)
				span := float64(min(c.Span, cols))
				colW = math.Max(colW, (cw-(span-1)*l.gap())/span)
				rowH = math.Max(rowH, ch)
			}
			total += rowH
		}
		pad := l.gap()
		if it == nil || it.Line == 0 {
			pad = 0 // korzeń nie ma ramki
		}
		return float64(cols)*colW + float64(cols-1)*l.gap() + 2*pad, total + float64(len(rows)-1)*l.gap() + 2*pad
	}
	lines := textwrap.Lines(it.Label, l.u(180), l.width)
	l.lines[it] = lines
	tw := 0.0
	for _, s := range lines {
		tw = math.Max(tw, l.width(s))
	}
	w, h = tw+2*l.u(16), float64(len(lines))*l.lh+2*l.u(10)
	switch it.Shape {
	case Circle:
		d := math.Max(w, h)
		return d, d
	case Diamond:
		return w + l.u(30), h + l.u(20)
	case Hexagon, Stadium, Arrow:
		w += l.u(20)
	case Cylinder:
		h += l.u(12)
	}
	return math.Max(w, l.u(70)), math.Max(h, l.u(44))
}

// place ustawia element w podanym polu; blok złożony rozdziela swoje pole między dzieci, tak że
// kolumny mają równą szerokość, a rząd wysokość najwyższego elementu.
func (l *layouter) place(it *Item, x, y, w, h float64) {
	l.box[it] = [4]float64{x, y, w, h}
	if it.Shape != Group {
		return
	}
	pad := l.gap()
	if it.Line == 0 {
		pad = 0
	}
	cols := l.columns(it)
	colW := (w - 2*pad - float64(cols-1)*l.gap()) / float64(cols)
	top := y + pad
	for _, row := range l.rows(it) {
		rowH := 0.0
		for _, c := range row {
			_, ch := l.natural(c)
			rowH = math.Max(rowH, ch)
		}
		col := 0
		for _, c := range row {
			span := min(c.Span, cols)
			cw := float64(span)*colW + float64(span-1)*l.gap()
			l.place(c, x+pad+float64(col)*(colW+l.gap()), top, cw, rowH)
			col += span
		}
		top += rowH + l.gap()
	}
}

// Layout rozmieszcza diagram: elementy stoją w siatce kolumn w kolejności ze źródła, blok złożony
// ma własną siatkę w ramce, a połączenia są odcinkami między brzegami bloków.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	l := &layouter{o: o, fs: fs, lh: math.Round(fs * 1.25), lines: map[*Item][]string{}, box: map[*Item][4]float64{}}
	l.edges = len(d.Edges) > 0
	for _, e := range d.Edges {
		for _, s := range e.Label {
			l.label = math.Max(l.label, l.width(s))
		}
	}
	l.sc = &scene.Scene{Title: d.AccTitle, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: l.u(12), Background: o.Background}
	w, h := l.natural(d.Root)
	l.place(d.Root, 0, 0, w, h)
	from := len(l.sc.Items)
	byID := map[string]*Item{}
	var draw func(it *Item, column int)
	draw = func(it *Item, column int) {
		if it.ID != "" {
			byID[it.ID] = it
		}
		b := l.box[it]
		switch it.Shape {
		case Space:
		case Group:
			if it.Line != 0 {
				stroke, dash := colorFrame, "5 4"
				if it.Paint.Stroke != "" {
					stroke, dash = it.Paint.Stroke, it.Paint.Dash
				}
				l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "block-group", X: b[0], Y: b[1], W: b[2], H: b[3], RX: l.u(6),
					Fill: it.Paint.Fill, Stroke: stroke, StrokeWidth: 1, Dash: dash})
			}
			col := 0
			for _, c := range it.Children {
				draw(c, col)
				col += c.Span
			}
		default:
			l.shape(it, b, column)
		}
	}
	draw(d.Root, 0)
	if o.Shadows {
		l.sc.AddShadows(from, l.u(3), "block")
	}
	for _, e := range d.Edges {
		l.edge(e, byID)
	}
	return l.sc
}

// shape rysuje blok o podanym kształcie z etykietą pośrodku.
func (l *layouter) shape(it *Item, b [4]float64, column int) {
	x, y, w, h := b[0], b[1], b[2], b[3]
	cx, cy := x+w/2, y+h/2
	fill, stroke, sw := colorFill, colorStroke, 1.5
	if l.o.Colors {
		t := scene.Distinct(column)
		fill, stroke = t.Fill, t.Stroke
	}
	if it.Paint.Fill != "" {
		fill = it.Paint.Fill
	}
	if it.Paint.Stroke != "" {
		stroke = it.Paint.Stroke
	}
	if it.Paint.StrokeWidth > 0 {
		sw = it.Paint.StrokeWidth
	}
	item := scene.Item{Layer: scene.LayerShape, Role: "block", Fill: fill, Stroke: stroke, StrokeWidth: sw, Dash: it.Paint.Dash}
	poly := func(pts ...scene.Point) {
		item.Kind, item.Points = scene.Polygon, pts
	}
	switch it.Shape {
	case Circle:
		item.Kind, item.X, item.Y, item.W, item.H = scene.Ellipse, cx, cy, w, h
	case Diamond:
		poly(scene.Point{X: cx, Y: y}, scene.Point{X: x + w, Y: cy}, scene.Point{X: cx, Y: y + h}, scene.Point{X: x, Y: cy})
	case Hexagon:
		k := math.Min(h/2, l.u(14))
		poly(scene.Point{X: x + k, Y: y}, scene.Point{X: x + w - k, Y: y}, scene.Point{X: x + w, Y: cy}, scene.Point{X: x + w - k, Y: y + h}, scene.Point{X: x + k, Y: y + h}, scene.Point{X: x, Y: cy})
	case Arrow:
		// Strzałka blokowa: trzon z grotem w podanym kierunku; „x” i „y” dają groty z obu stron.
		k, t := math.Min(l.u(18), w/3), h*0.22
		switch it.Direction {
		case "left":
			poly(scene.Point{X: x, Y: cy}, scene.Point{X: x + k, Y: y}, scene.Point{X: x + k, Y: y + t}, scene.Point{X: x + w, Y: y + t},
				scene.Point{X: x + w, Y: y + h - t}, scene.Point{X: x + k, Y: y + h - t}, scene.Point{X: x + k, Y: y + h})
		case "up":
			k, t = math.Min(l.u(18), h/3), w*0.22
			poly(scene.Point{X: cx, Y: y}, scene.Point{X: x + w, Y: y + k}, scene.Point{X: x + w - t, Y: y + k}, scene.Point{X: x + w - t, Y: y + h},
				scene.Point{X: x + t, Y: y + h}, scene.Point{X: x + t, Y: y + k}, scene.Point{X: x, Y: y + k})
		case "down":
			k, t = math.Min(l.u(18), h/3), w*0.22
			poly(scene.Point{X: cx, Y: y + h}, scene.Point{X: x + w, Y: y + h - k}, scene.Point{X: x + w - t, Y: y + h - k}, scene.Point{X: x + w - t, Y: y},
				scene.Point{X: x + t, Y: y}, scene.Point{X: x + t, Y: y + h - k}, scene.Point{X: x, Y: y + h - k})
		case "x":
			poly(scene.Point{X: x, Y: cy}, scene.Point{X: x + k, Y: y}, scene.Point{X: x + k, Y: y + t}, scene.Point{X: x + w - k, Y: y + t}, scene.Point{X: x + w - k, Y: y},
				scene.Point{X: x + w, Y: cy}, scene.Point{X: x + w - k, Y: y + h}, scene.Point{X: x + w - k, Y: y + h - t}, scene.Point{X: x + k, Y: y + h - t}, scene.Point{X: x + k, Y: y + h})
		default: // right; „y” też rysujemy w prawo, bo kierunek pionowy z dwoma grotami jest rzadki
			poly(scene.Point{X: x + w, Y: cy}, scene.Point{X: x + w - k, Y: y}, scene.Point{X: x + w - k, Y: y + t}, scene.Point{X: x, Y: y + t},
				scene.Point{X: x, Y: y + h - t}, scene.Point{X: x + w - k, Y: y + h - t}, scene.Point{X: x + w - k, Y: y + h})
		}
	default:
		item.Kind, item.X, item.Y, item.W, item.H = scene.Rect, x, y, w, h
		switch it.Shape {
		case Round:
			item.RX = l.u(8)
		case Stadium:
			item.RX = h / 2
		}
	}
	if it.Shape == Cylinder {
		// Walec: korpus między dwiema elipsami; górna jest widoczna w całości.
		ry := l.u(7)
		l.sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "block-detail", X: cx, Y: y + h - ry, W: w, H: 2 * ry, Fill: fill, Stroke: stroke, StrokeWidth: sw})
		item.Y, item.H, item.Stroke = y+ry, h-2*ry, ""
		l.sc.Add(item)
		for _, ex := range []float64{x, x + w} {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "block-detail", X: ex, Y: y + ry, X2: ex, Y2: y + h - ry, Stroke: stroke, StrokeWidth: sw})
		}
		l.sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "block-detail", X: cx, Y: y + ry, W: w, H: 2 * ry, Fill: fill, Stroke: stroke, StrokeWidth: sw})
	} else {
		l.sc.Add(item)
	}
	ink := colorText
	if it.Paint.Color != "" {
		ink = it.Paint.Color
	}
	lines := l.lines[it]
	top := cy - float64(len(lines))*l.lh/2
	for k, s := range lines {
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "block-label", X: cx, Y: top + float64(k)*l.lh + l.lh/2 + 0.35*l.fs,
			W: l.width(s), H: l.lh, Text: s, Anchor: "middle", Fill: ink})
	}
}

// border zwraca punkt, w którym odcinek ze środka bloku w stronę (tx, ty) przecina brzeg jego
// kształtu: prostokąta, elipsy albo rombu.
func border(shape string, b [4]float64, tx, ty float64) (float64, float64) {
	cx, cy := b[0]+b[2]/2, b[1]+b[3]/2
	dx, dy := tx-cx, ty-cy
	if dx == 0 && dy == 0 {
		return cx, cy
	}
	a, h := b[2]/2, b[3]/2
	var scale float64
	switch shape {
	case Circle:
		scale = 1 / math.Sqrt(dx*dx/(a*a)+dy*dy/(h*h))
	case Diamond:
		scale = 1 / (math.Abs(dx)/a + math.Abs(dy)/h)
	default:
		scale = math.Inf(1)
		if dx != 0 {
			scale = math.Min(scale, a/math.Abs(dx))
		}
		if dy != 0 {
			scale = math.Min(scale, h/math.Abs(dy))
		}
	}
	return cx + dx*scale, cy + dy*scale
}

// edge rysuje połączenie: odcinek między brzegami pól, z grotem przy celu i etykietą pośrodku.
func (l *layouter) edge(e Edge, byID map[string]*Item) {
	a, b := l.box[byID[e.From]], l.box[byID[e.To]]
	ax, ay := border(byID[e.From].Shape, a, b[0]+b[2]/2, b[1]+b[3]/2)
	bx, by := border(byID[e.To].Shape, b, a[0]+a[2]/2, a[1]+a[3]/2)
	length := math.Hypot(bx-ax, by-ay)
	if length < 1 {
		return
	}
	ux, uy := (bx-ax)/length, (by-ay)/length
	ex, ey := bx, by
	if e.Arrow {
		head, half := l.u(11), l.u(5.5)
		ex, ey = bx-ux*head*0.7, by-uy*head*0.7
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerActivation, Role: "arrowhead", Fill: colorLine, Points: []scene.Point{
			{X: bx, Y: by}, {X: bx - ux*head - uy*half, Y: by - uy*head + ux*half}, {X: bx - ux*head + uy*half, Y: by - uy*head - ux*half}}})
	}
	dash := ""
	if e.Dotted {
		dash = "3 3"
	}
	l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "edge", X: ax, Y: ay, X2: ex, Y2: ey, Stroke: colorLine, StrokeWidth: 1.5, Dash: dash})
	if len(e.Label) > 0 {
		mx, my := (ax+bx)/2, (ay+by)/2
		tw := 0.0
		for _, s := range e.Label {
			tw = math.Max(tw, l.width(s))
		}
		th := float64(len(e.Label)) * l.lh
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "edge-label-background", X: mx - tw/2 - l.u(3), Y: my - th/2, W: tw + l.u(6), H: th, Knockout: true})
		for k, s := range e.Label {
			l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "edge-label", X: mx, Y: my - th/2 + float64(k)*l.lh + l.lh/2 + 0.35*l.fs,
				W: l.width(s), H: l.lh, Text: s, Anchor: "middle", Fill: colorText})
		}
	}
}
