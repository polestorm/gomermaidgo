package railroad

import (
	"math"

	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // terminale i nieterminale w różnych tonach
	Shadows    bool // cień pod polami
}

const (
	colorText  = "#1b1f24"
	colorTrack = "#3b4754"
	colorBox   = "#eef2f7"
	colorTerm  = "#ffffff"
	colorMuted = "#55657a"
)

// box to wymiary węzła: szerokość i wysokość nad torem oraz pod nim (tor biegnie przez y = 0,
// wejście z lewej w x = 0, wyjście z prawej w x = w).
type box struct{ w, up, down float64 }

type layouter struct {
	o                     Options
	fs, h, pad, gap, r, v float64
	small                 float64
	sc                    *scene.Scene
	sizes                 map[*Node]box
}

func (l *layouter) u(v float64) float64 { return math.Round(v * l.fs / 16) }

// size liczy wymiary węzła (z pamięcią, bo rysowanie pyta o nie wiele razy).
func (l *layouter) size(n *Node) box {
	if b, ok := l.sizes[n]; ok {
		return b
	}
	var b box
	switch n.Kind {
	case Terminal, NonTerminal, Special:
		b = box{l.o.Measure(n.Text, l.fs) + 2*l.pad, l.h / 2, l.h / 2}
		if n.Kind == Special {
			b.w += l.u(12)
		}
	case Skip:
	case Sequence:
		for i, c := range n.Children {
			cb := l.size(c)
			b.w += cb.w
			if i > 0 {
				b.w += l.gap
			}
			b.up, b.down = math.Max(b.up, cb.up), math.Max(b.down, cb.down)
		}
	case Choice, Optional:
		kids := l.branches(n)
		inner := 0.0
		for _, c := range kids {
			inner = math.Max(inner, l.size(c).w)
		}
		b.w = inner + 4*l.r
		b.up, b.down = l.size(kids[0]).up, l.size(kids[0]).down
		for _, c := range kids[1:] {
			cb := l.size(c)
			b.down += math.Max(l.v+cb.up, 2*l.r) + cb.down
		}
		// Gałąź pod torem ma co najmniej dwa promienie łuku odstępu.
	case OneOrMore, ZeroOrMore:
		inner := l.loopInner(n)
		cb := l.size(inner)
		b = box{cb.w + 4*l.r, cb.up, l.loopY(n)}
		if n.Label != "" {
			b.down += l.small * 1.25
		}
		if n.Kind == ZeroOrMore {
			// Zero albo więcej: pętla i obejście nad elementem.
			b.up = math.Max(cb.up+l.v, 2*l.r)
		}
	}
	l.sizes[n] = b
	return b
}

// branches zwraca gałęzie wyboru; opcja to wybór między pustą drogą na torze a elementem.
func (l *layouter) branches(n *Node) []*Node {
	if n.Kind == Optional {
		return []*Node{{Kind: Skip}, n.Children[0]}
	}
	return n.Children
}

func (l *layouter) loopInner(n *Node) *Node { return n.Children[0] }

// loopY zwraca wysokość powrotnej linii pętli pod torem.
func (l *layouter) loopY(n *Node) float64 {
	return math.Max(l.size(l.loopInner(n)).down+l.v, 2*l.r)
}

// path dodaje linię toru złożoną z odcinków.
func (l *layouter) path(segs ...scene.Segment) {
	l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerShape, Role: "railroad-track", Segments: segs, Fill: "none", Stroke: colorTrack, StrokeWidth: 1.5})
}

func move(x, y float64) scene.Segment   { return scene.Segment{Cmd: 'M', To: scene.Point{X: x, Y: y}} }
func lineTo(x, y float64) scene.Segment { return scene.Segment{Cmd: 'L', To: scene.Point{X: x, Y: y}} }

// arc to ćwiartka okręgu od (x0, y0) do (x1, y1): najpierw wzdłuż kierunku (dx, dy), na końcu
// prostopadle do niego.
func arc(x0, y0, x1, y1 float64, horizontalFirst bool) scene.Segment {
	const k = 0.5523
	c1, c2 := scene.Point{X: x0 + (x1-x0)*k, Y: y0}, scene.Point{X: x1, Y: y1 - (y1-y0)*k}
	if !horizontalFirst {
		c1, c2 = scene.Point{X: x0, Y: y0 + (y1-y0)*k}, scene.Point{X: x1 - (x1-x0)*k, Y: y1}
	}
	return scene.Segment{Cmd: 'C', C1: c1, C2: c2, To: scene.Point{X: x1, Y: y1}}
}

// draw rysuje węzeł z wejściem w (x, y).
func (l *layouter) draw(n *Node, x, y float64) {
	b := l.size(n)
	switch n.Kind {
	case Terminal, NonTerminal, Special:
		fill, rx := colorBox, 0.0
		switch n.Kind {
		case Terminal:
			fill, rx = colorTerm, l.h/2
		case NonTerminal:
			rx = l.u(2)
		}
		if l.o.Colors {
			switch n.Kind {
			case Terminal:
				fill = scene.Palette[1].Fill
			case NonTerminal:
				fill = scene.Palette[0].Fill
			default:
				fill = scene.Palette[2].Fill
			}
		}
		role := map[Kind]string{Terminal: "railroad-terminal", NonTerminal: "railroad-nonterminal", Special: "railroad-special"}[n.Kind]
		if n.Kind == Special {
			e := l.u(6)
			l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: role, Fill: fill, Stroke: colorTrack, StrokeWidth: 1.5,
				Points: []scene.Point{{X: x, Y: y}, {X: x + e, Y: y - l.h/2}, {X: x + b.w - e, Y: y - l.h/2}, {X: x + b.w, Y: y}, {X: x + b.w - e, Y: y + l.h/2}, {X: x + e, Y: y + l.h/2}}})
		} else {
			l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: role, X: x, Y: y - l.h/2, W: b.w, H: l.h, RX: rx,
				Fill: fill, Stroke: colorTrack, StrokeWidth: 1.5})
		}
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role + "-label", X: x + b.w/2, Y: y + 0.35*l.fs, W: l.o.Measure(n.Text, l.fs), H: l.fs * 1.25,
			Text: n.Text, Anchor: "middle", Italic: n.Kind == Special, Bold: n.Kind == NonTerminal && false, Fill: colorText})
	case Skip:
	case Sequence:
		cx := x
		for i, c := range n.Children {
			if i > 0 {
				l.path(move(cx, y), lineTo(cx+l.gap, y))
				cx += l.gap
			}
			l.draw(c, cx, y)
			cx += l.size(c).w
		}
	case Choice, Optional:
		kids := l.branches(n)
		inner := b.w - 4*l.r
		r := l.r
		by := y
		for i, c := range kids {
			cb := l.size(c)
			if i > 0 {
				prev := l.size(kids[i-1])
				by += prev.down + math.Max(l.v+cb.up, 2*r)
			}
			if i == 0 {
				l.path(move(x, y), lineTo(x+2*r, y))
				l.draw(c, x+2*r, y)
				l.path(move(x+2*r+cb.w, y), lineTo(x+b.w, y))
				continue
			}
			// Zejście z toru, gałąź i powrót na tor.
			l.path(move(x, y), arc(x, y, x+r, y+r, true), lineTo(x+r, by-r), arc(x+r, by-r, x+2*r, by, false))
			l.draw(c, x+2*r, by)
			l.path(move(x+2*r+cb.w, by), lineTo(x+2*r+inner, by), arc(x+2*r+inner, by, x+3*r+inner, by-r, true),
				lineTo(x+3*r+inner, y+r), arc(x+3*r+inner, y+r, x+4*r+inner, y, false))
		}
	case OneOrMore, ZeroOrMore:
		inner := l.loopInner(n)
		cb := l.size(inner)
		r := l.r
		xl, xr := x+2*r, x+2*r+cb.w
		l.path(move(x, y), lineTo(xl, y))
		l.draw(inner, xl, y)
		l.path(move(xr, y), lineTo(x+b.w, y))
		// Pętla: z wyjścia elementu w dół, w lewo pod elementem i z powrotem na wejście.
		ly := y + l.loopY(n)
		l.path(move(xr, y), arc(xr, y, xr+r, y+r, true), lineTo(xr+r, ly-r), arc(xr+r, ly-r, xr, ly, false),
			lineTo(xl, ly), arc(xl, ly, xl-r, ly-r, true), lineTo(xl-r, y+r), arc(xl-r, y+r, xl, y, false))
		// Grot na powrotnej linii pokazuje kierunek czytania.
		mx, a := (xl+xr)/2, l.u(5)
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "railroad-track", Fill: colorTrack, Stroke: colorTrack, StrokeWidth: 1,
			Points: []scene.Point{{X: mx - a, Y: ly}, {X: mx + a, Y: ly - a}, {X: mx + a, Y: ly + a}}})
		if n.Label != "" {
			l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "railroad-repeat", X: mx, Y: ly + l.small*1.25*0.5 + 0.35*l.small + l.u(4),
				W: l.o.Measure(n.Label, l.small), H: l.small * 1.25, Text: n.Label, Anchor: "middle", FontSize: l.small, Fill: colorMuted})
		}
		if n.Kind == ZeroOrMore {
			// Obejście nad elementem: zero powtórzeń.
			ty := y - b.up
			l.path(move(x, y), arc(x, y, x+r, y-r, true), lineTo(x+r, ty+r), arc(x+r, ty+r, x+2*r, ty, false),
				lineTo(xr, ty), arc(xr, ty, xr+r, ty+r, true), lineTo(xr+r, y-r), arc(xr+r, y-r, x+b.w, y, false))
		}
	}
}

// Layout rysuje reguły jedna pod drugą: nazwa reguły, a pod nią tor od kółka początku do kółka
// końca. Terminale stoją w zaokrąglonych polach, nieterminale w prostokątach, sekwencje
// specjalne w sześciokątach; wybór rozgałęzia tor, powtórzenie wraca pętlą pod elementem.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	l := &layouter{o: o, fs: fs, sizes: map[*Node]box{}}
	l.h, l.pad, l.gap, l.r, l.v = l.u(30), l.u(12), l.u(16), l.u(10), l.u(12)
	l.small = math.Round(fs * 0.8)
	lh := math.Round(fs * 1.25)
	l.sc = &scene.Scene{Title: d.AccTitle, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: l.u(12), Background: o.Background}
	y := 0.0
	if d.Title != "" {
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "railroad-title", X: 0, Y: lh*0.5 + 0.35*fs*1.15, W: o.Measure(d.Title, fs*1.15), H: lh,
			Text: d.Title, Anchor: "start", Bold: true, FontSize: math.Round(fs * 1.15), Fill: colorText})
		y += lh + l.u(16)
	}
	from := len(l.sc.Items)
	for i, r := range d.Rules {
		if i > 0 {
			y += l.u(24)
		}
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "railroad-rule", X: 0, Y: y + lh/2 + 0.35*fs, W: o.Measure(r.Name, fs), H: lh,
			Text: r.Name, Anchor: "start", Bold: true, Fill: colorText})
		y += lh + l.u(6)
		b := l.size(r.Expr)
		base := y + b.up
		dot := l.u(5)
		lead := l.u(20)
		x0 := l.u(8)
		l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "railroad-start", X: x0, Y: base, R: dot, Fill: colorTrack, Stroke: colorTrack, StrokeWidth: 1})
		l.path(move(x0+dot, base), lineTo(x0+lead, base))
		l.draw(r.Expr, x0+lead, base)
		end := x0 + lead + b.w
		l.path(move(end, base), lineTo(end+lead-dot, base))
		l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "railroad-end", X: end + lead, Y: base, R: dot, Fill: "#ffffff", Stroke: colorTrack, StrokeWidth: 1.5})
		y = base + b.down
	}
	if o.Shadows {
		l.sc.AddShadows(from, l.u(2), "railroad-terminal", "railroad-nonterminal")
	}
	return l.sc
}
