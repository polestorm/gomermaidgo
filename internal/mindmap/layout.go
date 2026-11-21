package mindmap

import (
	"math"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // każda gałąź główna w swoim kolorze palety
	Shadows    bool // cień pod polami
}

const (
	colorText     = "#1b1f24"
	colorFill     = "#eef2f7"
	colorRootFill = "#dde3ea"
	colorStroke   = "#55657a"
	colorLine     = "#3b4754"
)

// branch to gałąź drzewa w układzie.
type branch struct {
	node     *graph.Node
	lines    []string
	w, h     float64
	children []*branch
	span     float64 // wysokość zajmowana przez gałąź z potomkami
	x, y     float64 // lewy górny róg pola
	tone     int     // numer gałęzi głównej, od 0; -1 dla korzenia
}

type layouter struct {
	o      Options
	fs, lh float64
	sc     *scene.Scene
}

func (l *layouter) u(v float64) float64 { return v * l.fs / 16 }

// run mierzy fragment tekstu w jednym stylu; pogrubienie jest szersze od kroju zwykłego.
func (l *layouter) run(text string, bold bool) float64 {
	w := l.o.Measure(text, l.fs)
	if bold {
		w *= 1.07
	}
	return w
}

// width mierzy linię; znaki wyróżnień nie zajmują miejsca.
func (l *layouter) width(s string) float64 {
	if !graph.HasMarks(s) {
		return l.o.Measure(s, l.fs)
	}
	w := 0.0
	for _, r := range graph.Segments(s) {
		w += l.run(r.Text, r.Bold)
	}
	return w
}

// build tworzy gałąź z węzła grafu i jej potomków, z rozmiarami pól.
func (l *layouter) build(g *graph.Graph, children map[string][]string, id string, tone int) *branch {
	n := g.Node(id)
	b := &branch{node: n, tone: tone}
	for _, line := range n.Label {
		b.lines = append(b.lines, textwrap.Lines([]string{graph.PerWord(line)}, l.u(180), l.width)...)
	}
	tw := 0.0
	for _, s := range b.lines {
		tw = math.Max(tw, l.width(s))
	}
	th := float64(len(b.lines)) * l.lh
	switch n.Shape {
	case graph.Circle:
		// Koło dla krótkiego tekstu; dłuższy dostaje elipsę, żeby pole nie rosło nad miarę.
		if d := math.Max(tw, th) + 2*l.u(12); tw <= l.u(110) {
			b.w, b.h = d, d
		} else {
			b.w, b.h = tw+2*l.u(22), th+2*l.u(14)
		}
	case graph.Hexagon:
		b.w, b.h = tw+2*l.u(22), th+2*l.u(7)
	case graph.Stadium:
		b.w, b.h = tw+2*l.u(18), th+2*l.u(7)
	default:
		b.w, b.h = tw+2*l.u(14), th+2*l.u(7)
	}
	b.w, b.h = math.Round(b.w), math.Round(b.h)
	for i, c := range children[id] {
		t := tone
		if tone < 0 {
			t = i
		}
		b.children = append(b.children, l.build(g, children, c, t))
	}
	return b
}

// measure wyznacza wysokość zajmowaną przez gałąź: jej pole albo stos potomków, co wyższe.
func (l *layouter) measure(b *branch) float64 {
	total := 0.0
	for i, c := range b.children {
		if i > 0 {
			total += l.u(10)
		}
		total += l.measure(c)
	}
	b.span = math.Max(b.h, total)
	return b.span
}

// stack zwraca wysokość stosu gałęzi z odstępami.
func (l *layouter) stack(bs []*branch) float64 {
	total := 0.0
	for i, c := range bs {
		if i > 0 {
			total += l.u(10)
		}
		total += c.span
	}
	return total
}

// columns zwraca największą szerokość pola na każdej głębokości (od 0) wśród podanych gałęzi.
func columns(bs []*branch, depth int, widths []float64) []float64 {
	for _, b := range bs {
		for len(widths) <= depth {
			widths = append(widths, 0)
		}
		widths[depth] = math.Max(widths[depth], b.w)
		widths = columns(b.children, depth+1, widths)
	}
	return widths
}

// place ustawia stos gałęzi po jednej stronie rodzica. Pola jednej głębokości zaczynają się
// w tej samej kolumnie: po prawej stronie są dosunięte do lewej krawędzi kolumny, po lewej
// do prawej. Każda gałąź stoi pośrodku wysokości swoich potomków.
func (l *layouter) place(bs []*branch, edge, top float64, dir float64, widths []float64, depth int) {
	for _, b := range bs {
		if dir > 0 {
			b.x = edge
		} else {
			b.x = edge - b.w
		}
		b.y = top + (b.span-b.h)/2
		next := edge + dir*(widths[depth]+l.u(44))
		l.place(b.children, next, top+(b.span-l.stack(b.children))/2, dir, widths, depth+1)
		top += b.span + l.u(10)
	}
}

// Layout rozmieszcza mapę myśli jako drzewo: korzeń, a gałęzie główne po jego prawej stronie albo,
// gdy rysunek lepiej mieści się wtedy na stronie, po obu stronach.
func Layout(g *graph.Graph, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	l := &layouter{o: o, fs: fs, lh: math.Round(fs * 1.25)}
	l.sc = &scene.Scene{Title: g.AccTitle, Desc: g.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: l.u(12), Background: o.Background}
	children := map[string][]string{}
	for _, e := range g.Edges {
		children[e.From] = append(children[e.From], e.To)
	}
	root := l.build(g, children, g.Nodes[0].ID, -1)
	l.measure(root)

	// Podział gałęzi głównych na strony: wszystkie po prawej albo pierwsze po prawej, a reszta
	// po lewej, tak żeby obie strony były podobnej wysokości. Wygrywa układ, który mieści się
	// na stronie w większej skali; przy równej skali jednostronny, bo czyta się go jak listę.
	fit := func(w, h float64) float64 { return math.Min(1, math.Min(l.u(650)/w, l.u(870)/h)) }
	extent := func(bs []*branch) (w, h float64) {
		for _, cw := range columns(bs, 0, nil) {
			w += cw + l.u(44)
		}
		return w, l.stack(bs)
	}
	right, left := root.children, []*branch(nil)
	rw, rh := extent(right)
	best := fit(root.w+rw, math.Max(root.h, rh))
	if len(root.children) > 1 {
		half, cut, sum := l.stack(root.children)/2, 1, 0.0
		for i, c := range root.children {
			sum += c.span + l.u(10)
			if sum >= half {
				cut = i + 1
				break
			}
		}
		cut = min(cut, len(root.children)-1)
		aw, ah := extent(root.children[:cut])
		bw, bh := extent(root.children[cut:])
		if two := fit(root.w+aw+bw, math.Max(root.h, math.Max(ah, bh))); two > best+0.02 {
			right, left = root.children[:cut], root.children[cut:]
		}
	}
	lw, _ := extent(left)
	height := math.Max(root.h, math.Max(l.stack(right), l.stack(left)))
	root.x, root.y = lw, (height-root.h)/2
	l.place(right, root.x+root.w+l.u(44), (height-l.stack(right))/2, 1, columns(right, 0, nil), 0)
	l.place(left, root.x-l.u(44), (height-l.stack(left))/2, -1, columns(left, 0, nil), 0)

	l.links(root, right, 1)
	l.links(root, left, -1)
	from := len(l.sc.Items)
	l.draw(root, true)
	if o.Shadows {
		l.sc.AddShadows(from, l.u(3), "node")
	}
	return l.sc
}

// tones zwraca wypełnienie i obrys gałęzi.
func (l *layouter) tones(b *branch, root bool) (fill, stroke string) {
	if l.o.Colors && b.tone >= 0 {
		t := scene.Palette[b.tone%len(scene.Palette)]
		return t.Fill, t.Stroke
	}
	if root {
		return colorRootFill, colorStroke
	}
	return colorFill, colorStroke
}

// links rysuje połączenia rodzica z gałęziami po jednej stronie: wspólny pień w połowie odstępu
// między kolumnami i odejścia do pól, z zaokrąglonymi narożnikami.
func (l *layouter) links(parent *branch, bs []*branch, dir float64) {
	if len(bs) == 0 {
		return
	}
	py := parent.y + parent.h/2
	px := parent.x + parent.w
	if dir < 0 {
		px = parent.x
	}
	// Pień stoi w stałej odległości przed kolumną potomków, niezależnie od szerokości rodzica.
	edge := bs[0].x
	if dir < 0 {
		edge = bs[0].x + bs[0].w
	}
	for _, b := range bs {
		if dir < 0 {
			edge = math.Max(edge, b.x+b.w)
		} else {
			edge = math.Min(edge, b.x)
		}
	}
	trunk := edge - dir*l.u(22)
	for _, b := range bs {
		cy := b.y + b.h/2
		cx := b.x
		if dir < 0 {
			cx = b.x + b.w
		}
		_, stroke := l.tones(b, false)
		if !l.o.Colors {
			stroke = colorLine
		}
		segments := []scene.Segment{{Cmd: 'M', To: scene.Point{X: px, Y: py}}}
		if dy := cy - py; math.Abs(dy) < 0.5 {
			segments = append(segments, scene.Segment{Cmd: 'L', To: scene.Point{X: cx, Y: py}})
		} else {
			r := math.Min(l.u(8), math.Abs(dy)/2)
			s := 1.0
			if dy < 0 {
				s = -1
			}
			const k = 0.45
			segments = append(segments,
				scene.Segment{Cmd: 'L', To: scene.Point{X: trunk - dir*r, Y: py}},
				scene.Segment{Cmd: 'C', C1: scene.Point{X: trunk - dir*r*k, Y: py}, C2: scene.Point{X: trunk, Y: py + s*r*k}, To: scene.Point{X: trunk, Y: py + s*r}},
				scene.Segment{Cmd: 'L', To: scene.Point{X: trunk, Y: cy - s*r}},
				scene.Segment{Cmd: 'C', C1: scene.Point{X: trunk, Y: cy - s*r*k}, C2: scene.Point{X: trunk + dir*r*k, Y: cy}, To: scene.Point{X: trunk + dir*r, Y: cy}},
				scene.Segment{Cmd: 'L', To: scene.Point{X: cx, Y: cy}})
		}
		l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "edge", Segments: segments, Stroke: stroke, StrokeWidth: 1.5})
		l.links(b, b.children, dir)
	}
}

// draw rysuje pole gałęzi z tekstem, potem jej potomków.
func (l *layouter) draw(b *branch, root bool) {
	fill, stroke := l.tones(b, root)
	sw := 1.5
	if root {
		sw = 2
	}
	item := scene.Item{Layer: scene.LayerShape, Role: "node", Fill: fill, Stroke: stroke, StrokeWidth: sw}
	cx, cy := b.x+b.w/2, b.y+b.h/2
	switch b.node.Shape {
	case graph.Circle:
		item.Kind, item.X, item.Y, item.W, item.H = scene.Ellipse, cx, cy, b.w, b.h
	case graph.Hexagon:
		d := math.Min(b.h/2, l.u(12))
		item.Kind = scene.Polygon
		item.Points = []scene.Point{{X: b.x + d, Y: b.y}, {X: b.x + b.w - d, Y: b.y}, {X: b.x + b.w, Y: cy},
			{X: b.x + b.w - d, Y: b.y + b.h}, {X: b.x + d, Y: b.y + b.h}, {X: b.x, Y: cy}}
	default:
		item.Kind, item.X, item.Y, item.W, item.H = scene.Rect, b.x, b.y, b.w, b.h
		switch b.node.Shape {
		case graph.Stadium:
			item.RX = b.h / 2
		case graph.Round:
			item.RX = l.u(6)
		}
	}
	l.sc.Add(item)
	top := cy - float64(len(b.lines))*l.lh/2
	for k, s := range b.lines {
		y := top + float64(k)*l.lh + l.lh/2 + 0.35*l.fs
		if !graph.HasMarks(s) {
			l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "node-label", X: cx, Y: y, W: l.width(s), H: l.lh,
				Text: s, Anchor: "middle", Fill: colorText})
			continue
		}
		// Linia z wyróżnieniami: fragmenty w różnych stylach stoją jeden za drugim.
		x := cx - l.width(s)/2
		for _, r := range graph.Segments(s) {
			w := l.run(r.Text, r.Bold)
			l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "node-label", X: x, Y: y, W: w, H: l.lh,
				Text: strings.TrimRight(r.Text, " \t"), Anchor: "start", Bold: r.Bold, Italic: r.Italic, Fill: colorText})
			x += w
		}
	}
	for _, c := range b.children {
		l.draw(c, false)
	}
}
