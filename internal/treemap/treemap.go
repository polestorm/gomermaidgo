// Package treemap rozbiera i rysuje mapę drzewa (treemap): prostokąty o polu proporcjonalnym
// do wartości, zagnieżdżone według hierarchii.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package treemap

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni mapy drzewa.
const (
	CodeHeader    = "treemap-header"
	CodeStatement = "treemap-statement"
	CodeValue     = "treemap-value"
	CodeEmpty     = "treemap-empty"
	maxNodes      = 300
)

// Node to węzeł: sekcja z dziećmi albo liść z wartością.
type Node struct {
	Label    string
	Value    float64 // dla liścia wartość ze źródła, dla sekcji suma dzieci
	Text     string  // wartość liścia tak, jak zapisano ją w źródle
	Children []*Node
	Leaf     bool
	Line     int
}

// Diagram to mapa drzewa.
type Diagram struct {
	Title    string
	Roots    []*Node
	AccTitle string
	AccDescr string
}

var entry = regexp.MustCompile(`^"([^"]*)"\s*(?::\s*([^:\s]+))?\s*(?::::[\p{L}\p{N}_-]+)?$`)

// Parse rozbiera źródło mapy drzewa. Hierarchię wyznacza wcięcie; linia z wartością jest liściem.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	seenHeader := false
	type open struct {
		indent int
		node   *Node
	}
	var stack []open
	count := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "treemap-beta" && s != "treemap" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"treemap-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		if s == "title" || strings.HasPrefix(s, "title ") {
			d.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(s, "title")), `"`)
			continue
		}
		if strings.HasPrefix(s, "classDef ") {
			continue // wygląd ze źródła nie jest rysowany
		}
		m := entry.FindStringSubmatch(s)
		if m == nil || strings.TrimSpace(m[1]) == "" {
			return nil, syntax.Errorf(line, CodeStatement, "expected \"name\" or \"name\": value, found %q", s)
		}
		node := &Node{Label: strings.TrimSpace(m[1]), Line: line}
		if m[2] != "" {
			v, err := strconv.ParseFloat(m[2], 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
				return nil, syntax.Errorf(line, CodeValue, "value %q is not a number of zero or more", m[2])
			}
			node.Leaf, node.Value, node.Text = true, v, m[2]
		}
		if count++; count > maxNodes {
			return nil, syntax.Errorf(line, CodeStatement, "too many nodes (limit %d)", maxNodes)
		}
		indent := 0
		for _, c := range raw {
			if c == ' ' {
				indent++
			} else if c == '\t' {
				indent += 4
			} else {
				break
			}
		}
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			d.Roots = append(d.Roots, node)
		} else {
			parent := stack[len(stack)-1].node
			if parent.Leaf {
				return nil, syntax.Errorf(line, CodeStatement, "%q has a value, so it cannot have children", parent.Label)
			}
			parent.Children = append(parent.Children, node)
		}
		stack = append(stack, open{indent, node})
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"treemap-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	var sum func(n *Node) float64
	sum = func(n *Node) float64 {
		if !n.Leaf {
			n.Value = 0
			for _, c := range n.Children {
				n.Value += sum(c)
			}
		}
		return n.Value
	}
	total := 0.0
	for _, r := range d.Roots {
		total += sum(r)
	}
	if total <= 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the treemap needs at least one value above zero")
	}
	d.AccTitle, d.AccDescr = acc.Title, acc.Descr
	return d, nil
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // gałęzie najwyższego poziomu w kolejnych tonach palety
	Shadows    bool // przyjmowane dla zgodności opcji; mapa nie ma cieni
}

const (
	colorText    = "#1b1f24"
	colorMuted   = "#5b6570"
	colorLine    = "#55657a"
	colorSection = "#dde3ea"
	colorLeaf    = "#f6f8fa"
)

type box struct{ x, y, w, h float64 }

// squarify dzieli prostokąt między węzły proporcjonalnie do wartości, układając je w rzędach
// tak, żeby prostokąty były możliwie bliskie kwadratom (Bruls, Huizing, van Wijk 2000).
func squarify(nodes []*Node, area box) []box {
	out := make([]box, len(nodes))
	order := make([]int, 0, len(nodes))
	total := 0.0
	for i, n := range nodes {
		if n.Value > 0 {
			order = append(order, i)
			total += n.Value
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return nodes[order[a]].Value > nodes[order[b]].Value })
	if total <= 0 || area.w <= 0 || area.h <= 0 {
		return out
	}
	scale := area.w * area.h / total
	// worst zwraca najgorszy stosunek boków w rzędzie o podanych polach, ułożonym wzdłuż boku side.
	worst := func(row []float64, side float64) float64 {
		sum, lo, hi := 0.0, math.Inf(1), 0.0
		for _, a := range row {
			sum, lo, hi = sum+a, math.Min(lo, a), math.Max(hi, a)
		}
		return math.Max(side*side*hi/(sum*sum), sum*sum/(side*side*lo))
	}
	for start := 0; start < len(order); {
		side := math.Min(area.w, area.h)
		row := []float64{nodes[order[start]].Value * scale}
		end := start + 1
		for end < len(order) {
			next := append(append([]float64(nil), row...), nodes[order[end]].Value*scale)
			if worst(next, side) > worst(row, side) {
				break
			}
			row, end = next, end+1
		}
		sum := 0.0
		for _, a := range row {
			sum += a
		}
		thick := sum / side
		pos := 0.0
		for k, a := range row {
			length := a / thick
			if area.w >= area.h { // rząd stoi pionowo przy lewej krawędzi
				out[order[start+k]] = box{area.x, area.y + pos, thick, length}
			} else {
				out[order[start+k]] = box{area.x + pos, area.y, length, thick}
			}
			pos += length
		}
		if area.w >= area.h {
			area.x, area.w = area.x+thick, area.w-thick
		} else {
			area.y, area.h = area.y+thick, area.h-thick
		}
		start = end
	}
	return out
}

// Layout rozmieszcza mapę: pole każdego liścia jest proporcjonalne do jego wartości, a sekcja
// obejmuje swoje dzieci i ma nazwę w pasku u góry. Liść ma nazwę i wartość, o ile się mieszczą.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.8)
	width := func(s string) float64 { return o.Measure(s, fs) }
	title := d.AccTitle
	if title == "" {
		title = d.Title
	}
	sc := &scene.Scene{Title: title, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}
	area := box{0, 0, u(620), u(400)}
	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "title", X: area.w / 2, Y: lh/2 + 0.35*fs, W: width(d.Title), H: lh,
			Text: d.Title, Anchor: "middle", Fill: colorText})
		area.y = lh + u(10)
	}
	var notes []string
	numbered := 0
	var draw func(nodes []*Node, area box, depth, tone int)
	draw = func(nodes []*Node, area box, depth, tone int) {
		for i, b := range squarify(nodes, area) {
			n := nodes[i]
			if b.w <= 0 || b.h <= 0 {
				continue
			}
			t := tone
			if depth == 0 {
				t = i
			}
			fill, line := colorLeaf, colorLine
			if !n.Leaf {
				fill = colorSection
			}
			if o.Colors {
				p := scene.Palette[t%len(scene.Palette)]
				fill, line = p.Fill, p.Stroke
			}
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "treemap-node", X: b.x, Y: b.y, W: b.w, H: b.h, Fill: fill, Stroke: line, StrokeWidth: 1})
			if !n.Leaf {
				// Sekcja: nazwa w pasku u góry, dzieci w pozostałym polu z marginesem.
				head := 0.0
				if b.h > lh+u(14) && width(n.Label) <= b.w-u(8) {
					head = lh + u(2)
					sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treemap-section", X: b.x + u(6), Y: b.y + u(2) + lh/2 + 0.35*fs,
						W: width(n.Label), H: lh, Text: n.Label, Anchor: "start", Bold: true, Fill: colorText})
					// Suma sekcji przy prawej krawędzi paska, jeśli jest na nią miejsce.
					if total := strconv.FormatFloat(n.Value, 'f', -1, 64); width(n.Label)+o.Measure(total, small)+u(24) <= b.w {
						sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treemap-total", X: b.x + b.w - u(6), Y: b.y + u(2) + lh/2 + 0.35*small,
							W: o.Measure(total, small), H: small * 1.25, Text: total, Anchor: "end", FontSize: small, Fill: colorMuted})
					}
				}
				pad := u(4)
				draw(n.Children, box{b.x + pad, b.y + head + pad, b.w - 2*pad, b.h - head - 2*pad}, depth+1, t)
				continue
			}
			// Nazwa w największym rozmiarze, w którym mieści się w polu; liść za mały na każdy
			// rozmiar dostaje numer, a jego nazwa i wartość trafiają do listy pod mapą.
			placed := false
			for _, size := range []float64{fs, small, math.Round(fs * 0.65)} {
				measure := func(s string) float64 { return o.Measure(s, size) }
				lines := textwrap.Lines([]string{n.Label}, b.w-u(6), measure)
				fits := true
				for _, s := range lines {
					fits = fits && measure(s) <= b.w-u(4)
				}
				line := math.Round(size * 1.25)
				h := float64(len(lines)) * line
				if !fits || h > b.h-u(4) {
					continue
				}
				withValue := h+small*1.25 <= b.h-u(4) && o.Measure(n.Text, small) <= b.w-u(4)
				top := b.y + (b.h-h)/2
				if withValue {
					top -= small * 1.25 / 2
				}
				for k, s := range lines {
					sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treemap-label", X: b.x + b.w/2, Y: top + float64(k)*line + line/2 + 0.35*size,
						W: measure(s), H: line, Text: s, Anchor: "middle", FontSize: size, Fill: colorText})
				}
				if withValue {
					sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treemap-value", X: b.x + b.w/2, Y: top + h + small,
						W: o.Measure(n.Text, small), H: small * 1.25, Text: n.Text, Anchor: "middle", FontSize: small, Fill: colorMuted})
				} else {
					notes = append(notes, n.Label+": "+n.Text)
				}
				placed = true
				break
			}
			if !placed {
				numbered++
				mark := strconv.Itoa(numbered)
				notes = append(notes, mark+" – "+n.Label+": "+n.Text)
				if size := math.Round(fs * 0.65); o.Measure(mark, size) <= b.w-u(2) && size <= b.h {
					sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treemap-mark", X: b.x + b.w/2, Y: b.y + b.h/2 + 0.35*size,
						W: o.Measure(mark, size), H: size * 1.25, Text: mark, Anchor: "middle", FontSize: size, Fill: colorText})
				}
			}
		}
	}
	draw(d.Roots, area, 0, 0)
	// Lista pod mapą: liście, których nazwa albo wartość nie zmieściła się w polu.
	for i, s := range notes {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treemap-note", X: 0, Y: area.y + area.h + u(10) + float64(i)*small*1.4 + small,
			W: o.Measure(s, small), H: small * 1.25, Text: s, Anchor: "start", FontSize: small, Fill: colorMuted})
	}
	return sc
}
