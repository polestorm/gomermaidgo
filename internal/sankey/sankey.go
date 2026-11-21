// Package sankey rozbiera i rysuje wykres Sankeya: przepływy między węzłami jako wstęgi
// o szerokości proporcjonalnej do wartości.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package sankey

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni wykresu Sankeya.
const (
	CodeHeader    = "sankey-header"
	CodeStatement = "sankey-statement"
	CodeValue     = "sankey-value"
	CodeCycle     = "sankey-cycle"
	CodeEmpty     = "sankey-empty"
	maxLinks      = 500
)

// Link to przepływ ze źródła do celu.
type Link struct {
	From, To string
	Value    float64
	Line     int
}

// Chart to wykres Sankeya. Nodes: nazwy węzłów w kolejności pierwszego wystąpienia.
type Chart struct {
	Nodes []string
	Links []Link
}

// fields dzieli linię CSV na pola; pole w cudzysłowach może zawierać przecinek, a podwojony
// cudzysłów oznacza cudzysłów.
func fields(s string) ([]string, bool) {
	var out []string
	var b strings.Builder
	quoted := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quoted && c == '"' && i+1 < len(s) && s[i+1] == '"':
			b.WriteByte('"')
			i++
		case c == '"':
			quoted = !quoted
		case c == ',' && !quoted:
			out = append(out, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	return append(out, strings.TrimSpace(b.String())), !quoted
}

// Parse rozbiera źródło wykresu: po nagłówku linie „źródło,cel,wartość”.
func Parse(source string) (*Chart, error) {
	c := &Chart{}
	seenHeader := false
	seen := map[string]bool{}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "sankey-beta" && s != "sankey" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"sankey-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		f, closed := fields(s)
		if !closed || len(f) != 3 || f[0] == "" || f[1] == "" {
			return nil, syntax.Errorf(line, CodeStatement, "expected a flow written as source,target,value, found %q", s)
		}
		v, err := strconv.ParseFloat(f[2], 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return nil, syntax.Errorf(line, CodeValue, "flow value %q is not a number above zero", f[2])
		}
		if f[0] == f[1] {
			return nil, syntax.Errorf(line, CodeCycle, "flow from %q to itself", f[0])
		}
		if len(c.Links) >= maxLinks {
			return nil, syntax.Errorf(line, CodeStatement, "too many flows (limit %d)", maxLinks)
		}
		for _, name := range f[:2] {
			if !seen[name] {
				seen[name] = true
				c.Nodes = append(c.Nodes, name)
			}
		}
		c.Links = append(c.Links, Link{f[0], f[1], v, line})
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"sankey-beta\"")
	}
	if len(c.Links) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the chart needs at least one flow")
	}
	if _, line := columns(c); line > 0 {
		return nil, syntax.Errorf(line, CodeCycle, "flows form a cycle")
	}
	return c, nil
}

// columns wyznacza kolumnę każdego węzła: długość najdłuższej drogi od węzła bez dopływów.
// Gdy przepływy tworzą cykl, zwraca numer linii jednego z przepływów cyklu.
func columns(c *Chart) (map[string]int, int) {
	col := map[string]int{}
	indeg := map[string]int{}
	out := map[string][]Link{}
	for _, l := range c.Links {
		indeg[l.To]++
		out[l.From] = append(out[l.From], l)
	}
	var queue []string
	for _, n := range c.Nodes {
		if indeg[n] == 0 {
			queue = append(queue, n)
		}
	}
	done := 0
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		done++
		for _, l := range out[n] {
			col[l.To] = max(col[l.To], col[n]+1)
			if indeg[l.To]--; indeg[l.To] == 0 {
				queue = append(queue, l.To)
			}
		}
	}
	if done < len(c.Nodes) {
		for _, l := range c.Links {
			if indeg[l.To] > 0 {
				return nil, l.Line
			}
		}
	}
	return col, 0
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // węzły i ich wstęgi w kolorach palety
	Shadows    bool // przyjmowane dla zgodności opcji; wykres nie ma cieni
}

const (
	colorText  = "#1b1f24"
	colorMuted = "#5b6570"
	colorNode  = "#3b4754"
	colorBand  = "#7b8794"
)

// Number zapisuje wartość węzła: bez zbędnych zer, najwyżej dwa miejsca po przecinku.
func Number(v float64) string {
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// Layout rozmieszcza wykres: węzły w kolumnach od lewej do prawej według najdłuższej drogi od
// źródeł, wysokość węzła i szerokość wstęgi proporcjonalne do wartości w jednej skali dla całego
// wykresu, nazwa i wartość obok węzła.
func Layout(c *Chart, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	width := func(s string) float64 { return o.Measure(s, fs) }
	sc := &scene.Scene{FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	col, _ := columns(c)
	last := 0
	in, out := map[string]float64{}, map[string]float64{}
	for _, l := range c.Links {
		out[l.From] += l.Value
		in[l.To] += l.Value
		last = max(last, col[l.To])
	}
	value := func(n string) float64 { return math.Max(in[n], out[n]) }
	// Węzeł bez odpływów jest końcem przepływu i stoi w ostatniej kolumnie, obok pozostałych końców.
	for _, n := range c.Nodes {
		if out[n] == 0 {
			col[n] = last
		}
	}
	cols := make([][]string, last+1)
	for _, n := range c.Nodes {
		cols[col[n]] = append(cols[col[n]], n)
	}
	// Kolejność w kolumnie: węzły ustawiają się według średniego położenia sąsiadów z kolumny
	// po lewej, żeby wstęgi krzyżowały się jak najmniej.
	pos := map[string]float64{}
	for _, column := range cols {
		for i, n := range column {
			pos[n] = float64(i)
		}
	}
	for k := 1; k <= last; k++ {
		sum, weight := map[string]float64{}, map[string]float64{}
		for _, l := range c.Links {
			if col[l.To] == k {
				sum[l.To] += pos[l.From] * l.Value
				weight[l.To] += l.Value
			}
		}
		column := cols[k]
		sort.SliceStable(column, func(a, b int) bool { return sum[column[a]]/weight[column[a]] < sum[column[b]]/weight[column[b]] })
		for i, n := range column {
			pos[n] = float64(i)
		}
	}

	// Jedna skala dla całego wykresu: najwyższa kolumna razem z odstępami mieści się w wysokości.
	gap, nodeW := u(14), u(14)
	height := u(320)
	for _, column := range cols {
		height = math.Max(height, float64(len(column))*u(34))
	}
	scale := math.Inf(1)
	for _, column := range cols {
		total := 0.0
		for _, n := range column {
			total += value(n)
		}
		scale = math.Min(scale, (height-gap*float64(len(column)-1))/total)
	}
	labelW := 0.0
	for _, n := range cols[last] {
		labelW = math.Max(labelW, math.Max(width(n), width(Number(value(n)))))
	}
	plotW := u(620) - labelW - u(8)
	step := 0.0
	if last > 0 {
		step = (plotW - nodeW) / float64(last)
	}
	type rect struct{ x, y, h float64 }
	at := map[string]rect{}
	for k, column := range cols {
		total := gap * float64(len(column)-1)
		for _, n := range column {
			total += value(n) * scale
		}
		y := (height - total) / 2
		for _, n := range column {
			h := value(n) * scale
			at[n] = rect{float64(k) * step, y, h}
			y += h + gap
		}
	}
	tone := func(n string) string {
		if !o.Colors {
			return ""
		}
		for i, name := range c.Nodes {
			if name == n {
				return scene.Palette[i%len(scene.Palette)].Stroke
			}
		}
		return ""
	}

	// Wstęgi: przy każdym węźle ułożone w kolejności położenia drugiego końca, żeby się nie plotły.
	links := append([]Link(nil), c.Links...)
	fromOff, toOff := map[int]float64{}, map[int]float64{}
	order := func(key func(Link) string, other func(Link) string, off map[int]float64) {
		idx := make([]int, len(links))
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(a, b int) bool { return at[other(links[idx[a]])].y < at[other(links[idx[b]])].y })
		used := map[string]float64{}
		for _, i := range idx {
			off[i] = used[key(links[i])]
			used[key(links[i])] += links[i].Value * scale
		}
	}
	order(func(l Link) string { return l.From }, func(l Link) string { return l.To }, fromOff)
	order(func(l Link) string { return l.To }, func(l Link) string { return l.From }, toOff)
	for i, l := range links {
		a, b := at[l.From], at[l.To]
		t := l.Value * scale
		x0, x1 := a.x+nodeW, b.x
		y0, y1 := a.y+fromOff[i], b.y+toOff[i]
		mid := (x0 + x1) / 2
		fill := colorBand
		if c := tone(l.From); c != "" {
			fill = c
		}
		sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "sankey-link", Fill: fill, FillOpacity: 0.35, Segments: []scene.Segment{
			{Cmd: 'M', To: scene.Point{X: x0, Y: y0}},
			{Cmd: 'C', C1: scene.Point{X: mid, Y: y0}, C2: scene.Point{X: mid, Y: y1}, To: scene.Point{X: x1, Y: y1}},
			{Cmd: 'L', To: scene.Point{X: x1, Y: y1 + t}},
			{Cmd: 'C', C1: scene.Point{X: mid, Y: y1 + t}, C2: scene.Point{X: mid, Y: y0 + t}, To: scene.Point{X: x0, Y: y0 + t}},
			{Cmd: 'L', To: scene.Point{X: x0, Y: y0}},
		}})
	}
	for _, n := range c.Nodes {
		r := at[n]
		fill := colorNode
		if c := tone(n); c != "" {
			fill = c
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "sankey-node", X: r.x, Y: r.y, W: nodeW, H: r.h, Fill: fill})
		// Nazwa i wartość po prawej stronie węzła; w ostatniej kolumnie także, na miejscu
		// zostawionym poza wykresem.
		x, mid := r.x+nodeW+u(6), r.y+r.h/2
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "sankey-label", X: x, Y: mid - u(3), W: width(n), H: lh,
			Text: n, Anchor: "start", Fill: colorText})
		v := Number(value(n))
		small := math.Round(fs * 0.8)
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "sankey-value", X: x, Y: mid + small + u(2), W: o.Measure(v, small), H: small * 1.25,
			Text: v, Anchor: "start", FontSize: small, Fill: colorMuted})
	}
	return sc
}
