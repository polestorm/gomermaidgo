// Package ishikawa rozbiera i rysuje diagram przyczyn i skutków (diagram Ishikawy, „rybia ość”).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package ishikawa

import (
	"math"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni diagramu Ishikawy.
const (
	CodeHeader    = "ishikawa-header"
	CodeStatement = "ishikawa-statement"
	CodeEmpty     = "ishikawa-empty"
	maxCategories = 12
	maxCauses     = 200
	maxDepth      = 3 // poziomy przyczyn pod kategorią
)

// Cause to kategoria albo przyczyna z przyczynami szczegółowymi.
type Cause struct {
	Text     []string
	Children []*Cause
	Line     int
}

// Diagram to problem (skutek) i kategorie przyczyn.
type Diagram struct {
	Effect     []string
	Categories []*Cause
	AccTitle   string
	AccDescr   string
}

// Parse rozbiera źródło. Pierwsza linia po nagłówku to problem; następne to przyczyny, a wcięcie
// wyznacza poziom: linie na wcięciu pierwszej z nich (albo płytszym) są kategoriami, głębsze są
// przyczynami najbliższej wcześniejszej linii o mniejszym wcięciu.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	seenHeader := false
	base := -1
	type open struct {
		indent int
		cause  *Cause
		depth  int
	}
	var stack []open
	causes := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "ishikawa-beta" && s != "ishikawa" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"ishikawa-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		text := textwrap.Split(s)
		if len(text) == 0 {
			return nil, syntax.Errorf(line, CodeStatement, "line has no text: %q", s)
		}
		if d.Effect == nil {
			d.Effect = text
			continue
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
		if causes++; causes > maxCauses {
			return nil, syntax.Errorf(line, CodeStatement, "too many causes (limit %d)", maxCauses)
		}
		c := &Cause{Text: text, Line: line}
		if base < 0 || indent <= base {
			if len(d.Categories) >= maxCategories {
				return nil, syntax.Errorf(line, CodeStatement, "too many categories (limit %d)", maxCategories)
			}
			if base < 0 {
				base = indent
			}
			d.Categories = append(d.Categories, c)
			stack = []open{{indent, c, 0}}
			continue
		}
		for len(stack) > 1 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		if parent.depth+1 > maxDepth {
			return nil, syntax.Errorf(line, CodeStatement, "causes nested too deep (limit %d levels under a category)", maxDepth)
		}
		parent.cause.Children = append(parent.cause.Children, c)
		stack = append(stack, open{indent, c, parent.depth + 1})
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"ishikawa-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if d.Effect == nil {
		return nil, syntax.Errorf(0, CodeEmpty, "the diagram needs a problem (the first line after the header)")
	}
	if len(d.Categories) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the diagram needs at least one category of causes")
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
	Colors     bool // kategorie w kolejnych tonach palety
	Shadows    bool // cień pod polami kategorii i problemu
}

const (
	colorText  = "#1b1f24"
	colorFill  = "#eef2f7"
	colorLine  = "#55657a"
	colorTwig  = "#9aa5b1"
	colorHead  = "#3b4754"
	slope      = 0.5 // przesunięcie ości w poziomie na jednostkę wysokości
	causeWidth = 140 // szerokość łamania tekstu przyczyny (przy kroju 16 px)
	labelWidth = 160 // szerokość łamania nazwy kategorii i problemu
)

// row to jedna przyczyna na ości: poziom, linie tekstu i położenie kreski względem osi.
type row struct {
	depth int
	lines []string
	textW float64
	y     float64 // wysokość kreski, na której stoi tekst (ekranowa, oś w y = 0)
	x0    float64 // lewy koniec kreski
	x1    float64 // prawy koniec kreski (na ości albo na pionowym odgałęzieniu rodzica)
	stem  float64 // x pionowego odgałęzienia do przyczyn szczegółowych
	last  int     // indeks ostatniego potomka (dla odgałęzienia)
}

// bone to kategoria z przyczynami, ułożona względem swojego punktu na osi (x = 0).
type bone struct {
	label        []string
	labelW, labH float64
	rows         []*row
	up           bool    // nad osią
	height       float64 // odległość końca ości od osi
	minX, maxX   float64
	at           float64 // położenie punktu na osi
	tone         int
}

// Layout rysuje diagram: oś pozioma prowadzi do pola z problemem po prawej; kategorie stoją na
// przemian nad osią i pod nią, w kolejności źródła od lewej, a ich ukośne ości schodzą do osi.
// Przyczyny to poziome kreski przy ości z tekstem nad kreską; przyczyny szczegółowe wiszą na
// pionowym odgałęzieniu pod przyczyną, do której należą.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return math.Round(v * fs / 16) }
	lh := math.Round(fs * 1.25)
	width := func(s string) float64 { return o.Measure(s, fs) }
	widest := func(lines []string) float64 {
		w := 0.0
		for _, s := range lines {
			w = math.Max(w, width(s))
		}
		return w
	}
	sc := &scene.Scene{Title: d.AccTitle, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}
	text := func(lines []string, x, top float64, anchor, role string, bold bool) {
		for k, s := range lines {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: top + float64(k)*lh + lh/2 + 0.35*fs,
				W: width(s), H: lh, Text: s, Anchor: anchor, Bold: bold, Fill: colorText})
		}
	}

	// Ości: rzędy przyczyn w kolejności źródła (najpierw przyczyna, potem jej szczegóły).
	bones := make([]*bone, len(d.Categories))
	for i, cat := range d.Categories {
		b := &bone{up: i%2 == 0, tone: i}
		b.label = textwrap.Lines(cat.Text, u(labelWidth), width)
		b.labelW, b.labH = widest(b.label)+2*u(12), float64(len(b.label))*lh+2*u(6)
		var walk func(c *Cause, depth int)
		walk = func(c *Cause, depth int) {
			r := &row{depth: depth, lines: textwrap.Lines(c.Text, u(causeWidth), width)}
			r.textW = widest(r.lines)
			b.rows = append(b.rows, r)
			for _, ch := range c.Children {
				walk(ch, depth+1)
			}
			r.last = len(b.rows) - 1
		}
		for _, c := range cat.Children {
			walk(c, 1)
		}
		bones[i] = b
	}
	// Wysokość rzędu: tekst nad kreską i odstęp do następnego rzędu.
	rowH := func(r *row) float64 { return float64(len(r.lines))*lh + u(12) }
	span := func(b *bone) float64 {
		h := 0.0
		for _, r := range b.rows {
			h += rowH(r)
		}
		return h
	}
	// Wspólna wysokość ości po każdej stronie: ości są równoległe, a nazwy kategorii w jednym rzędzie.
	sideH := [2]float64{}
	for _, b := range bones {
		s := 0
		if !b.up {
			s = 1
		}
		sideH[s] = math.Max(sideH[s], span(b)+u(28))
	}
	for _, b := range bones {
		if b.up {
			b.height = sideH[0]
		} else {
			b.height = sideH[1]
		}
		// Rzędy: nad osią od nazwy kategorii w dół do osi, pod osią od osi w dół do nazwy.
		y := -b.height + u(10)
		if !b.up {
			y = u(18)
		}
		for _, r := range b.rows {
			r.y = y + float64(len(r.lines))*lh + u(2)
			y += rowH(r)
		}
		boneX := func(y float64) float64 { return -math.Abs(y) * slope }
		for k, r := range b.rows {
			if r.depth == 1 {
				r.x1 = boneX(r.y)
			} else {
				// Rodzic: najbliższy wcześniejszy rząd o mniejszym poziomie.
				for p := k - 1; p >= 0; p-- {
					if b.rows[p].depth == r.depth-1 {
						r.x1 = b.rows[p].stem
						break
					}
				}
			}
			r.x0 = r.x1 - r.textW - u(20)
			r.stem = r.x0 + u(14)
		}
		tip := boneX(b.height)
		b.minX, b.maxX = tip-b.labelW/2, math.Max(0, tip+b.labelW/2)
		for _, r := range b.rows {
			b.minX = math.Min(b.minX, r.x0)
		}
	}
	// Rozmieszczenie wzdłuż osi: każda ość staje tak blisko poprzednich po tej samej stronie, jak
	// pozwalają ich kształty pas po pasie (pole nazwy, kreski z tekstem, odgałęzienia i sama ukośna
	// ość), więc przyczyny wchodzą pod skos poprzedniej ości. Punkty na osi idą w kolejności źródła,
	// na przemian nad osią i pod nią.
	gap, step := u(20), u(72)
	var prev *bone
	for i, b := range bones {
		b.at = -b.minX
		mine := b.shapes(u, lh)
		for _, p := range bones[:i] {
			if p.up != b.up {
				continue
			}
			for _, x := range p.shapes(u, lh) {
				for _, y := range mine {
					if x.y0 < y.y1+u(4) && y.y0 < x.y1+u(4) {
						b.at = math.Max(b.at, p.at+x.r-y.l+gap)
					}
				}
			}
		}
		if prev != nil {
			b.at = math.Max(b.at, prev.at+step)
		}
		prev = b
	}
	// Oś zaczyna się tuż przed pierwszą ością; teksty przyczyn mogą sięgać dalej w lewo.
	spineStart, spineEnd := math.Inf(1), 0.0
	for _, b := range bones {
		spineStart = math.Min(spineStart, b.at-u(24))
		spineEnd = math.Max(spineEnd, b.at+b.maxX)
	}

	// Pole problemu po prawej, w osi.
	effect := textwrap.Lines(d.Effect, u(labelWidth), width)
	headW, headH := widest(effect)+2*u(16), float64(len(effect))*lh+2*u(12)
	headX := spineEnd + u(40)
	from := len(sc.Items)
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "ishikawa-effect", X: headX, Y: -headH / 2, W: headW, H: headH, RX: u(4),
		Fill: colorFill, Stroke: colorHead, StrokeWidth: 2})
	text(effect, headX+headW/2, -headH/2+u(12), "middle", "ishikawa-effect-label", true)
	// Oś z grotem przy polu problemu.
	head := u(12)
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "ishikawa-spine", X: spineStart, Y: 0, X2: headX - head, Y2: 0, Stroke: colorHead, StrokeWidth: 2.5})
	sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "ishikawa-spine", Points: []scene.Point{{X: headX, Y: 0}, {X: headX - head, Y: -head / 2}, {X: headX - head, Y: head / 2}},
		Fill: colorHead, Stroke: colorHead, StrokeWidth: 1})

	for _, b := range bones {
		fill, line := colorFill, colorLine
		if o.Colors {
			t := scene.Distinct(b.tone)
			fill, line = t.Fill, t.Stroke
		}
		sign := -1.0
		if !b.up {
			sign = 1
		}
		tipX, tipY := b.at-b.height*slope, sign*b.height
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "ishikawa-bone", X: tipX, Y: tipY, X2: b.at, Y2: 0, Stroke: colorLine, StrokeWidth: 1.5})
		// Nazwa kategorii w polu na końcu ości.
		top := tipY - b.labH
		if !b.up {
			top = tipY
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "ishikawa-category", X: tipX - b.labelW/2, Y: top, W: b.labelW, H: b.labH, RX: u(4),
			Fill: fill, Stroke: line, StrokeWidth: 1.5})
		text(b.label, tipX, top+u(6), "middle", "ishikawa-category-label", true)
		for k, r := range b.rows {
			x0, x1 := b.at+r.x0, b.at+r.x1
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "ishikawa-cause", X: x0, Y: r.y, X2: x1, Y2: r.y, Stroke: colorTwig, StrokeWidth: 1})
			text(r.lines, x1-u(10)-float64(len(r.lines))*lh*slope*boolTo(r.depth == 1), r.y-float64(len(r.lines))*lh-u(2), "end", "ishikawa-cause-label", false)
			if r.last > k {
				sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "ishikawa-cause", X: b.at + r.stem, Y: r.y, X2: b.at + r.stem, Y2: b.rows[r.last].y,
					Stroke: colorTwig, StrokeWidth: 1})
				// Odgałęzienie kończy się na ostatnim bezpośrednim potomku, nie na najgłębszym.
				end := r.y
				for c := k + 1; c <= r.last; c++ {
					if b.rows[c].depth == r.depth+1 {
						end = b.rows[c].y
					}
				}
				sc.Items[len(sc.Items)-1].Y2 = end
			}
		}
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "ishikawa-category", "ishikawa-effect")
	}
	return sc
}

// band to poziomy pas zajęty przez część ości: od y0 do y1, od l do r (względem punktu na osi).
type band struct{ y0, y1, l, r float64 }

// shapes zwraca pasy zajęte przez ość: pole nazwy, kreski przyczyn z tekstem, odgałęzienia
// i samą ukośną ość pociętą na krótkie odcinki.
func (b *bone) shapes(u func(float64) float64, lh float64) []band {
	sign := -1.0
	if !b.up {
		sign = 1
	}
	var out []band
	tip := -b.height * slope
	top := sign*b.height - b.labH
	if !b.up {
		top = b.height
	}
	out = append(out, band{top, top + b.labH, tip - b.labelW/2, tip + b.labelW/2})
	for k, r := range b.rows {
		out = append(out, band{r.y - float64(len(r.lines))*lh - u(2), r.y, r.x0, r.x1})
		if r.last > k {
			out = append(out, band{r.y, b.rows[r.last].y, r.stem, r.stem})
		}
	}
	piece := u(8)
	for d := 0.0; d < b.height; d += piece {
		e := math.Min(d+piece, b.height)
		y0, y1 := sign*d, sign*e
		if y0 > y1 {
			y0, y1 = y1, y0
		}
		out = append(out, band{y0, y1, -e * slope, -d * slope})
	}
	return out
}

func boolTo(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
