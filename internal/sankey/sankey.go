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
	"github.com/polestorm/gomermaidgo/internal/textwrap"
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
	Wrap  bool // %% layout: wrap: nazwy węzłów łamane na słowach
}

var wrapDirective = syntax.Lazy(`^%%\s*layout:\s*wrap\s*$`)

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
		if wrapDirective.MatchString(s) {
			c.Wrap = true
		}
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
	// Węzeł bez odpływów jest końcem przepływu i stoi w ostatniej kolumnie, obok pozostałych końców.
	for _, n := range c.Nodes {
		if out[n] == 0 {
			col[n] = last
		}
	}
	// Przepływ przez kilka kolumn dostaje w każdej kolumnie pośredniej węzeł pomocniczy o swojej
	// grubości: zajmuje tam miejsce, więc węzły tej kolumny nie stają na jego drodze. Układ liczy
	// się na odcinkach między sąsiednimi kolumnami.
	type piece struct {
		from, to string
		value    float64
	}
	var pieces []piece
	chains := make([][]string, len(c.Links)) // węzły, przez które biegnie wstęga, razem z końcami
	nodes := append([]string(nil), c.Nodes...)
	virtual := map[string]bool{}
	size := map[string]float64{}
	for _, n := range c.Nodes {
		size[n] = math.Max(in[n], out[n])
	}
	for i, l := range c.Links {
		chain := []string{l.From}
		for k := col[l.From] + 1; k < col[l.To]; k++ {
			v := "\x00" + strconv.Itoa(i) + "@" + strconv.Itoa(k)
			virtual[v], col[v], size[v] = true, k, l.Value
			nodes = append(nodes, v)
			chain = append(chain, v)
		}
		chain = append(chain, l.To)
		chains[i] = chain
		for k := 0; k+1 < len(chain); k++ {
			pieces = append(pieces, piece{chain[k], chain[k+1], l.Value})
		}
	}
	cols := make([][]string, last+1)
	for _, n := range nodes {
		cols[col[n]] = append(cols[col[n]], n)
	}
	// Kolejność początkowa w kolumnie: według średniego położenia sąsiadów z kolumny po lewej.
	pos := map[string]float64{}
	for _, column := range cols {
		for i, n := range column {
			pos[n] = float64(i)
		}
	}
	for k := 1; k <= last; k++ {
		sum, weight := map[string]float64{}, map[string]float64{}
		for _, p := range pieces {
			if col[p.to] == k {
				sum[p.to] += pos[p.from] * p.value
				weight[p.to] += p.value
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
			total += size[n]
		}
		scale = math.Min(scale, (height-gap*float64(len(column)-1))/total)
	}
	// Nazwa węzła: jedna linia, a przy „%% layout: wrap” łamana na słowach. W ostatniej kolumnie do
	// szerokości najdłuższego słowa wśród jej nazw, żeby jedna długa nazwa nie zwężała wykresu.
	name := map[string][]string{}
	for _, n := range c.Nodes {
		name[n] = []string{n}
	}
	if c.Wrap {
		limit := u(48)
		for _, n := range cols[last] {
			for _, word := range strings.Fields(n) {
				limit = math.Max(limit, width(word))
			}
		}
		for _, n := range cols[last] {
			name[n] = textwrap.Lines([]string{n}, limit, width)
		}
	}
	labelW := 0.0
	for _, n := range cols[last] {
		labelW = math.Max(labelW, width(Number(size[n])))
		for _, line := range name[n] {
			labelW = math.Max(labelW, width(line))
		}
	}
	plotW := u(620) - labelW - u(8)
	step := 0.0
	if last > 0 {
		step = (plotW - nodeW) / float64(last)
	}
	if c.Wrap && last > 0 {
		// Nazwy wcześniejszych kolumn mieszczą się w odstępie do następnej kolumny.
		for _, n := range c.Nodes {
			if col[n] < last {
				name[n] = textwrap.Lines([]string{n}, step-nodeW-u(12), width)
			}
		}
	}

	// Położenie w pionie: węzły zaczynają w stosie pośrodku kolumny, a potem każdy przesuwa się ku
	// średniej ważonej środków swoich sąsiadów, na zmianę względem celów i źródeł. Wstęgi wychodzą
	// przez to możliwie prosto. Po każdym kroku kolumna jest porządkowana: węzły zachowują odstęp
	// i mieszczą się w wysokości wykresu.
	type rect struct{ x, y, h float64 }
	at := map[string]rect{}
	for k, column := range cols {
		total := gap * float64(len(column)-1)
		for _, n := range column {
			total += size[n] * scale
		}
		y := (height - total) / 2
		for _, n := range column {
			h := size[n] * scale
			at[n] = rect{float64(k) * step, y, h}
			y += h + gap
		}
	}
	centre := func(n string) float64 { return at[n].y + at[n].h/2 }
	settle := func(column []string) {
		sort.SliceStable(column, func(a, b int) bool { return centre(column[a]) < centre(column[b]) })
		y := 0.0
		for _, n := range column {
			r := at[n]
			r.y = math.Max(r.y, y)
			at[n] = r
			y = r.y + r.h + gap
		}
		y = height
		for i := len(column) - 1; i >= 0; i-- {
			r := at[column[i]]
			r.y = math.Min(r.y, y-r.h)
			at[column[i]] = r
			y = r.y - gap
		}
	}
	pull := func(column []string, alpha float64, towardTargets bool) {
		moves := make([]float64, len(column))
		for i, n := range column {
			sum, weight := 0.0, 0.0
			for _, p := range pieces {
				if towardTargets && p.from == n {
					sum, weight = sum+centre(p.to)*p.value, weight+p.value
				}
				if !towardTargets && p.to == n {
					sum, weight = sum+centre(p.from)*p.value, weight+p.value
				}
			}
			if weight > 0 {
				moves[i] = (sum/weight - centre(n)) * alpha
			}
		}
		for i, n := range column {
			r := at[n]
			r.y += moves[i]
			at[n] = r
		}
		settle(column)
	}
	const rounds = 24
	for i := 0; i < rounds; i++ {
		alpha := 1 - float64(i)/rounds
		for k := last - 1; k >= 0; k-- {
			pull(cols[k], alpha, true)
		}
		for k := 1; k <= last; k++ {
			pull(cols[k], alpha, false)
		}
	}
	// Wykres zaczyna się od zera: usuwamy wolne miejsce nad najwyższym węzłem.
	top := math.Inf(1)
	for _, r := range at {
		top = math.Min(top, r.y)
	}
	for n, r := range at {
		r.y = math.Round((r.y-top)*100) / 100
		at[n] = r
	}
	tone := func(n string) string {
		if !o.Colors {
			return ""
		}
		for i, name := range c.Nodes {
			if name == n {
				return scene.Distinct(i).Stroke
			}
		}
		return ""
	}

	// Wstęgi: przy każdym węźle ułożone w kolejności położenia drugiego końca, żeby się nie plotły.
	fromOff, toOff := map[[2]string]float64{}, map[[2]string]float64{}
	order := func(key func(piece) string, other func(piece) string, off map[[2]string]float64) {
		idx := make([]int, len(pieces))
		for i := range idx {
			idx[i] = i
		}
		// Kolejność według środka drugiego końca: wstęga do węzła leżącego wyżej wychodzi wyżej.
		sort.SliceStable(idx, func(a, b int) bool { return centre(other(pieces[idx[a]])) < centre(other(pieces[idx[b]])) })
		used := map[string]float64{}
		for _, i := range idx {
			off[[2]string{pieces[i].from, pieces[i].to}] += 0 // klucz istnieje także dla przesunięcia zerowego
			off[[2]string{pieces[i].from, pieces[i].to}] = used[key(pieces[i])]
			used[key(pieces[i])] += pieces[i].value * scale
		}
	}
	order(func(p piece) string { return p.from }, func(p piece) string { return p.to }, fromOff)
	order(func(p piece) string { return p.to }, func(p piece) string { return p.from }, toOff)
	for i, l := range c.Links {
		chain := chains[i]
		t := l.Value * scale
		// Górna krawędź wstęgi przez kolejne węzły drogi, potem dolna z powrotem: jeden kształt,
		// więc na węzłach pomocniczych nie ma szwów.
		type span struct{ x0, y0, x1, y1 float64 }
		var spans []span
		for k := 0; k+1 < len(chain); k++ {
			a, b := at[chain[k]], at[chain[k+1]]
			key := [2]string{chain[k], chain[k+1]}
			spans = append(spans, span{a.x + nodeW, a.y + fromOff[key], b.x, b.y + toOff[key]})
		}
		segments := []scene.Segment{{Cmd: 'M', To: scene.Point{X: spans[0].x0, Y: spans[0].y0}}}
		for k, sp := range spans {
			if k > 0 {
				segments = append(segments, scene.Segment{Cmd: 'L', To: scene.Point{X: sp.x0, Y: sp.y0}})
			}
			mid := (sp.x0 + sp.x1) / 2
			segments = append(segments, scene.Segment{Cmd: 'C', C1: scene.Point{X: mid, Y: sp.y0}, C2: scene.Point{X: mid, Y: sp.y1}, To: scene.Point{X: sp.x1, Y: sp.y1}})
		}
		for k := len(spans) - 1; k >= 0; k-- {
			sp := spans[k]
			mid := (sp.x0 + sp.x1) / 2
			segments = append(segments, scene.Segment{Cmd: 'L', To: scene.Point{X: sp.x1, Y: sp.y1 + t}},
				scene.Segment{Cmd: 'C', C1: scene.Point{X: mid, Y: sp.y1 + t}, C2: scene.Point{X: mid, Y: sp.y0 + t}, To: scene.Point{X: sp.x0, Y: sp.y0 + t}})
		}
		segments = append(segments, scene.Segment{Cmd: 'L', To: scene.Point{X: spans[0].x0, Y: spans[0].y0}})
		fill := colorBand
		if c := tone(l.From); c != "" {
			fill = c
		}
		sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: "sankey-link", Fill: fill, FillOpacity: 0.35, Segments: segments})
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
		lines := name[n]
		mid += float64(len(lines)-1) * lh / 2
		for k, line := range lines {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "sankey-label", X: x, Y: mid - u(3) - float64(len(lines)-1-k)*lh, W: width(line), H: lh,
				Text: line, Anchor: "start", Fill: colorText})
		}
		v := Number(size[n])
		small := math.Round(fs * 0.8)
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "sankey-value", X: x, Y: mid + small + u(2), W: o.Measure(v, small), H: small * 1.25,
			Text: v, Anchor: "start", FontSize: small, Fill: colorMuted})
	}
	return sc
}
