package venn

import (
	"math"
	"sort"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Rozmieszczenie podpisów. Każdy obszar (część zbioru poza innymi kołami, część wspólna
// poza pozostałymi kołami) jest próbkowany na siatce; sumy prefiksowe komórek spoza obszaru
// pozwalają w stałym czasie sprawdzić, czy prostokąt bloku tekstu leży cały w obszarze.
// Dla każdego łamania i skali szukamy miejsca z największym zapasem od brzegów, potem
// środkujemy blok w dostępnym miejscu w poziomie i w pionie. Wszystkie obszary dostają
// jedno wspólne powiększenie kół i jedną skalę tekstu (chooseSize); obszar, w którym tekst
// nie mieści się nawet wtedy, dostaje podpis poza kołami z kreską odnośnika.

// grid to obszar próbkowany w komórkach o boku step.
type grid struct {
	x0, y0, step float64
	nx, ny       int
	out          []int // sumy prefiksowe komórek poza obszarem, (nx+1)×(ny+1)
	cells        int   // liczba komórek w obszarze
}

func newGrid(x0, y0, x1, y1, step float64, in func(x, y float64) bool) *grid {
	g := &grid{x0: x0, y0: y0, step: step, nx: int(math.Ceil((x1 - x0) / step)), ny: int(math.Ceil((y1 - y0) / step))}
	if g.nx < 1 || g.ny < 1 {
		g.nx, g.ny = 0, 0
		return g
	}
	w := g.nx + 1
	g.out = make([]int, w*(g.ny+1))
	for iy := 0; iy < g.ny; iy++ {
		row := 0
		for ix := 0; ix < g.nx; ix++ {
			if in(x0+(float64(ix)+0.5)*step, y0+(float64(iy)+0.5)*step) {
				g.cells++
			} else {
				row++
			}
			g.out[(iy+1)*w+ix+1] = g.out[iy*w+ix+1] + row
		}
	}
	return g
}

// fits mówi, czy prostokąt o środku (cx, cy) i wymiarach w×h leży cały w obszarze.
func (g *grid) fits(cx, cy, w, h float64) bool {
	ix0 := int(math.Floor((cx - w/2 - g.x0) / g.step))
	ix1 := int(math.Floor((cx + w/2 - g.x0) / g.step))
	iy0 := int(math.Floor((cy - h/2 - g.y0) / g.step))
	iy1 := int(math.Floor((cy + h/2 - g.y0) / g.step))
	if ix0 < 0 || iy0 < 0 || ix1 >= g.nx || iy1 >= g.ny {
		return false
	}
	w1 := g.nx + 1
	return g.out[(iy1+1)*w1+ix1+1]-g.out[iy0*w1+ix1+1]-g.out[(iy1+1)*w1+ix0]+g.out[iy0*w1+ix0] == 0
}

// best szuka środka prostokąta w×h z największym zapasem od brzegów obszaru, a potem
// środkuje go w dostępnym miejscu w poziomie i w pionie.
func (g *grid) best(w, h float64) (x, y, pad float64, ok bool) {
	pad = -1
	for iy := 0; iy < g.ny; iy += 2 {
		for ix := 0; ix < g.nx; ix += 2 {
			cx, cy := g.x0+(float64(ix)+0.5)*g.step, g.y0+(float64(iy)+0.5)*g.step
			// Tylko środek, który bije dotychczasowy zapas, jest mierzony dokładniej.
			if !g.fits(cx, cy, w+2*(pad+g.step/2), h+2*(pad+g.step/2)) {
				continue
			}
			lo, hi := math.Max(0, pad+g.step/2), math.Max(0, pad+g.step/2)+g.step
			for g.fits(cx, cy, w+2*hi, h+2*hi) && hi < 1e4 {
				lo, hi = hi, hi*2
			}
			for k := 0; k < 6; k++ {
				if m := (lo + hi) / 2; g.fits(cx, cy, w+2*m, h+2*m) {
					lo = m
				} else {
					hi = m
				}
			}
			if lo > pad {
				x, y, pad, ok = cx, cy, lo, true
			}
		}
	}
	if !ok {
		return 0, 0, 0, false
	}
	for k := 0; k < 2; k++ {
		x = g.center(x, y, w, h, 1, 0)
		y = g.center(x, y, w, h, 0, 1)
	}
	return x, y, pad, true
}

// center przesuwa prostokąt w kierunku (dx, dy) na środek ciągłego przedziału położeń,
// w których jeszcze się mieści.
func (g *grid) center(x, y, w, h, dx, dy float64) float64 {
	s := g.step / 2
	a, b := 0.0, 0.0
	for g.fits(x-(a+s)*dx, y-(a+s)*dy, w, h) && a < 1e4 {
		a += s
	}
	for g.fits(x+(b+s)*dx, y+(b+s)*dy, w, h) && b < 1e4 {
		b += s
	}
	return x*dx + y*dy + (b-a)/2
}

// textBlock to podpis obszaru i jego elementy złamane na linie w jednej skali.
type textBlock struct {
	label, items []string
	itemOwner    []*Text
	fs, small    float64
	lh, ih       float64
	w, h         float64 // wymiary tekstu z marginesem
}

type blockStyle struct {
	fs, small, lh, ih, margin float64
	bold                      bool
	measure                   func(string, float64) float64
}

func (st blockStyle) build(a *Area, limit, scale float64) textBlock {
	b := textBlock{fs: math.Round(st.fs*scale*10) / 10, small: math.Round(st.small*scale*10) / 10, lh: st.lh * scale, ih: st.ih * scale}
	lw := func(s string) float64 {
		if st.bold {
			return st.measure(s, b.fs) * 1.06
		}
		return st.measure(s, b.fs)
	}
	iw := func(s string) float64 { return st.measure(s, b.small) }
	width := 0.0
	if text := areaText(a); text != "" {
		b.label = textwrap.Lines(textwrap.Split(text), limit, lw)
		for _, s := range b.label {
			width = math.Max(width, lw(s))
		}
	}
	for _, t := range a.Texts {
		for _, s := range textwrap.Lines(textwrap.Split(t.Label), limit, iw) {
			b.items, b.itemOwner = append(b.items, s), append(b.itemOwner, t)
			width = math.Max(width, iw(s))
		}
	}
	b.w = width + 2*st.margin
	b.h = float64(len(b.label))*b.lh + float64(len(b.items))*b.ih + st.margin
	return b
}

// limits to szerokości łamania do wypróbowania: od najdłuższego wyrazu do tekstu bez łamania.
func (st blockStyle) limits(a *Area, scale float64) []float64 {
	lo, hi := 0.0, 0.0
	add := func(text string, size float64, bold bool) {
		for _, line := range textwrap.Split(text) {
			f := 1.0
			if bold {
				f = 1.06
			}
			hi = math.Max(hi, st.measure(line, size)*f)
			for _, word := range strings.Fields(line) {
				lo = math.Max(lo, st.measure(word, size)*f)
			}
		}
	}
	add(areaText(a), math.Round(st.fs*scale*10)/10, st.bold)
	for _, t := range a.Texts {
		add(t.Label, math.Round(st.small*scale*10)/10, false)
	}
	out := []float64{hi}
	for k := 1; k <= 8 && hi > lo; k++ {
		out = append(out, hi-(hi-lo)*float64(k)/8)
	}
	return out
}

func areaText(a *Area) string {
	if a.Label != "" {
		return a.Label
	}
	if len(a.IDs) == 1 {
		return a.IDs[0]
	}
	return ""
}

// fit to najlepsze ułożenie bloku w obszarze w danej skali.
type fit struct {
	b    textBlock
	x, y float64
	pad  float64
	ok   bool
}

func fitIn(g *grid, a *Area, st blockStyle, scale float64) fit {
	// Dla każdej liczby linii najwęższe łamanie (linie najbardziej wyrównane).
	byLines := map[int]textBlock{}
	var counts []int
	for _, limit := range st.limits(a, scale) {
		b := st.build(a, limit, scale)
		n := len(b.label) + len(b.items)
		if _, ok := byLines[n]; !ok {
			counts = append(counts, n)
		}
		if old, ok := byLines[n]; !ok || b.w <= old.w {
			byLines[n] = b
		}
	}
	// Z ułożeń, które się mieszczą, wygrywa to z najmniejszą liczbą linii, o ile jego zapas
	// od brzegów jest wystarczający (nie mniejszy niż dwa marginesy albo 60% największego).
	sort.Ints(counts)
	fits := make([]fit, len(counts))
	maxPad := -1.0
	for k, n := range counts {
		b := byLines[n]
		if x, y, pad, ok := g.best(b.w, b.h); ok {
			fits[k] = fit{b, x, y, pad, true}
			maxPad = math.Max(maxPad, pad)
		}
	}
	for _, f := range fits {
		if f.ok && f.pad >= math.Min(2*st.margin, 0.6*maxPad) {
			return f
		}
	}
	return fit{}
}

// emit rysuje blok o środku (x, y).
func (b textBlock) emit(sc *scene.Scene, a *Area, x, y float64, bold bool) {
	top := y - b.h/2 + (b.h-float64(len(b.label))*b.lh-float64(len(b.items))*b.ih)/2
	for _, s := range b.label {
		fill := colorText
		if a.Paint.Color != "" {
			fill = a.Paint.Color
		}
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "venn-label", X: x, Y: top + b.lh/2 + 0.35*b.fs, W: b.w, H: b.lh,
			Text: s, Anchor: "middle", Bold: bold, FontSize: b.fs, Fill: fill})
		top += b.lh
	}
	for k, s := range b.items {
		t := b.itemOwner[k]
		fill := colorItem
		if t.Paint.Color != "" {
			fill = t.Paint.Color
		} else if a.Paint.Color != "" {
			fill = a.Paint.Color
		}
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "venn-text", X: x, Y: top + b.ih/2 + 0.35*b.small, W: b.w, H: b.ih,
			Text: s, Anchor: "middle", FontSize: b.small, Fill: fill})
		top += b.ih
	}
}

type rect struct{ x0, y0, x1, y1 float64 }

func (r rect) overlaps(o rect) bool { return r.x0 < o.x1 && o.x0 < r.x1 && r.y0 < o.y1 && o.y0 < r.y1 }

// segmentHits mówi, czy odcinek przecina prostokąt (próbkowanie co kilka punktów).
func (r rect) segmentHits(x0, y0, x1, y1 float64) bool {
	n := int(math.Hypot(x1-x0, y1-y0)/2) + 1
	for k := 0; k <= n; k++ {
		t := float64(k) / float64(n)
		x, y := x0+(x1-x0)*t, y0+(y1-y0)*t
		if x > r.x0 && x < r.x1 && y > r.y0 && y < r.y1 {
			return true
		}
	}
	return false
}

// job to obszar do podpisania.
type job struct {
	a    *Area
	g    *grid
	st   blockStyle
	bold bool
}

// makeJobs buduje obszary podpisów dla kół cs.
func makeJobs(d *Diagram, cs []circle, index map[string]int, o Options, u func(float64) float64) []job {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	lh, small := math.Round(fs*1.25), math.Round(fs*0.85)
	step := math.Max(1, u(2))
	var jobs []job
	region := func(members map[int]bool, exclusive bool) *grid {
		x0, y0, x1, y1 := math.Inf(-1), math.Inf(-1), math.Inf(1), math.Inf(1)
		for i := range members {
			c := cs[i]
			x0, y0 = math.Max(x0, c.x-c.r), math.Max(y0, c.y-c.r)
			x1, y1 = math.Min(x1, c.x+c.r), math.Min(y1, c.y+c.r)
		}
		return newGrid(x0, y0, x1, y1, step, func(x, y float64) bool {
			for i, c := range cs {
				if members[i] != c.contains(x, y) && (members[i] || exclusive) {
					return false
				}
			}
			return true
		})
	}
	for i, s := range d.Sets {
		st := blockStyle{fs: fs, small: small, lh: lh, ih: math.Round(small * 1.25), margin: u(4), bold: true, measure: o.Measure}
		jobs = append(jobs, job{s, region(map[int]bool{i: true}, true), st, true})
	}
	for _, un := range d.Unions {
		if areaText(un) == "" && len(un.Texts) == 0 {
			continue
		}
		members := map[int]bool{}
		for _, id := range un.IDs {
			members[index[id]] = true
		}
		g := region(members, true)
		if g.cells == 0 {
			g = region(members, false)
		}
		if g.cells == 0 {
			continue
		}
		st := blockStyle{fs: fs, small: small, lh: lh, ih: math.Round(small * 1.25), margin: u(4), measure: o.Measure}
		jobs = append(jobs, job{un, g, st, false})
	}
	return jobs
}

// sizes to kolejne próby: najpierw większe koła (do 1,3 raza), potem mniejszy tekst (do 0,8).
var sizes = [][2]float64{{1, 1}, {1.15, 1}, {1.3, 1}, {1.3, 0.9}, {1.3, 0.8}}

// chooseSize wybiera powiększenie kół i skalę tekstu: pierwszą próbę, w której wszystkie
// podpisy mieszczą się w swoich obszarach, a gdy takiej nie ma — tę, w której mieści się
// ich najwięcej (pozostałe dostaną kreskę odnośnika).
func chooseSize(d *Diagram, cs []circle, index map[string]int, o Options, u func(float64) float64) (grow, scale float64) {
	bestN := -1
	for _, sz := range sizes {
		scaled := make([]circle, len(cs))
		for i, c := range cs {
			scaled[i] = circle{c.x * sz[0], c.y * sz[0], c.r * sz[0]}
		}
		jobs := makeJobs(d, scaled, index, o, u)
		n := 0
		for _, j := range jobs {
			if fitIn(j.g, j.a, j.st, sz[1]).ok {
				n++
			}
		}
		if n == len(jobs) {
			return sz[0], sz[1]
		}
		if n > bestN {
			bestN, grow, scale = n, sz[0], sz[1]
		}
	}
	return grow, scale
}

// placeLabels rozmieszcza podpisy wszystkich obszarów w skali tekstu scale.
func placeLabels(sc *scene.Scene, d *Diagram, cs []circle, index map[string]int, o Options, u func(float64) float64, titleBox *rect, scale float64) {
	step := math.Max(1, u(2))
	jobs := makeJobs(d, cs, index, o, u)
	inside := make([]bool, len(jobs))
	for k, j := range jobs {
		inside[k] = fitIn(j.g, j.a, j.st, scale).ok
	}
	var placed []rect
	if titleBox != nil {
		placed = append(placed, *titleBox)
	}
	var callouts []job
	for k, j := range jobs {
		if !inside[k] {
			callouts = append(callouts, j)
			continue
		}
		f := fitIn(j.g, j.a, j.st, scale)
		f.b.emit(sc, j.a, f.x, f.y, j.bold)
		placed = append(placed, rect{f.x - f.b.w/2, f.y - f.b.h/2, f.x + f.b.w/2, f.y + f.b.h/2})
	}
	if len(callouts) == 0 {
		return
	}

	// Podpisy z kreską odnośnika: poza wszystkimi kołami, w kierunku najkrótszej drogi.
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range cs {
		minX, minY, maxX, maxY = math.Min(minX, c.x-c.r), math.Min(minY, c.y-c.r), math.Max(maxX, c.x+c.r), math.Max(maxY, c.y+c.r)
	}
	ext := u(260)
	gap := u(8)
	outside := newGrid(minX-ext, minY-ext, maxX+ext, maxY+ext, step, func(x, y float64) bool {
		for _, c := range cs {
			if math.Hypot(x-c.x, y-c.y) < c.r+gap {
				return false
			}
		}
		return true
	})
	for _, j := range callouts {
		ax, ay, _, ok := j.g.best(0, 0)
		if !ok {
			continue
		}
		b := j.st.build(j.a, math.Max(u(160), minLimit(j.st.limits(j.a, scale))), scale)
		const dirs = 24
		bestCost, bx, by := math.Inf(1), 0.0, 0.0
		for k := 0; k < dirs; k++ {
			th := 2 * math.Pi * float64(k) / dirs
			dx, dy := math.Cos(th), math.Sin(th)
			for t := 0.0; t < 2*ext; t += step {
				x, y := ax+dx*t, ay+dy*t
				if !outside.fits(x, y, b.w, b.h) {
					continue
				}
				r := rect{x - b.w/2, y - b.h/2, x + b.w/2, y + b.h/2}
				clash := false
				for _, p := range placed {
					if r.overlaps(p) {
						clash = true
						break
					}
				}
				if clash {
					continue
				}
				cost := t
				for _, p := range placed {
					if p.segmentHits(ax, ay, x, y) {
						cost += 1e4
					}
				}
				if cost < bestCost {
					bestCost, bx, by = cost, x, y
				}
				break
			}
		}
		if math.IsInf(bestCost, 1) {
			continue
		}
		// Kreska kończy się na brzegu prostokąta podpisu.
		ex, ey := bx-ax, by-ay
		f := math.Inf(-1)
		if ex != 0 {
			f = 1 - (b.w/2+u(2))/math.Abs(ex)
		}
		if ey != 0 {
			f = math.Max(f, 1-(b.h/2+u(2))/math.Abs(ey))
		}
		f = math.Max(f, 0)
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "venn-leader", X: ax, Y: ay, X2: ax + ex*f, Y2: ay + ey*f, Stroke: colorStroke, StrokeWidth: 1})
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "venn-leader", X: ax, Y: ay, R: u(3), Fill: colorStroke, Stroke: "none"})
		b.emit(sc, j.a, bx, by, j.bold)
		placed = append(placed, rect{bx - b.w/2, by - b.h/2, bx + b.w/2, by + b.h/2})
	}
}

func minLimit(ls []float64) float64 { return ls[len(ls)-1] }
