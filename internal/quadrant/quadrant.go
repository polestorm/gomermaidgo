// Package quadrant rozbiera i rysuje wykres ćwiartkowy (quadrantChart).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package quadrant

import (
	"math"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni wykresu ćwiartkowego.
const (
	CodeHeader    = "quadrant-header"
	CodeStatement = "quadrant-statement"
	CodePoint     = "quadrant-point"
	maxPoints     = 100
)

// Point to punkt wykresu: nazwa i położenie od 0 do 1 na każdej osi.
type Point struct {
	Name string
	X, Y float64
	Line int
	Style
}

// Style to wygląd punktu ze źródła: z klasy (classDef) albo z zapisu po współrzędnych.
// Wartość zerowa pola oznacza wygląd domyślny.
type Style struct {
	Radius      float64
	Color       string
	StrokeColor string
	StrokeWidth float64
}

// apply dopisuje do stylu własności z zapisu „radius: 8, color: #b42318, stroke-width: 2px”.
// Własność nieznana albo o złej wartości jest pomijana.
func (st *Style) apply(list string) {
	for _, part := range strings.Split(list, ",") {
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		number := func() float64 {
			v, err := strconv.ParseFloat(strings.TrimSuffix(value, "px"), 64)
			if err != nil || !(v > 0 && v <= 40) {
				return 0
			}
			return v
		}
		switch key {
		case "radius":
			if v := number(); v > 0 {
				st.Radius = v
			}
		case "stroke-width":
			if v := number(); v > 0 {
				st.StrokeWidth = v
			}
		case "color":
			if paint.MatchString(value) {
				st.Color = value
			}
		case "stroke-color":
			if paint.MatchString(value) {
				st.StrokeColor = value
			}
		}
	}
}

// Chart to wykres ćwiartkowy. Quadrants: 0 prawa górna, 1 lewa górna, 2 lewa dolna, 3 prawa dolna.
type Chart struct {
	Title         string
	XLeft, XRight string
	YBottom, YTop string
	Quadrants     [4]string
	Points        []Point
	AccTitle      string
	AccDescr      string
}

var (
	point    = syntax.Lazy(`^(.+?)(?::::([\p{L}\p{N}_-]+))?\s*:\s*\[\s*([^,\]]+?)\s*,\s*([^,\]]+?)\s*\](.*)$`)
	classDef = syntax.Lazy(`^classDef\s+([\p{L}\p{N}_-]+)\s+(.+)$`)
	// Kolor: zapis szesnastkowy albo nazwa; nic, co mogłoby wyjść poza atrybut.
	paint    = syntax.Lazy(`^(#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6}|[a-zA-Z]+)$`)
	quadrant = syntax.Lazy(`^quadrant-([1-4])\s+(.+)$`)
)

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

// Parse rozbiera źródło wykresu ćwiartkowego.
func Parse(source string) (*Chart, error) {
	c := &Chart{}
	var acc syntax.Accessibility
	seenHeader := false
	classes := map[string]Style{}
	var classOf []string // klasa każdego punktu; definicja może stać po punkcie
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "quadrantChart" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"quadrantChart\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		word, rest := s, ""
		if i := strings.IndexAny(s, " \t"); i >= 0 {
			word, rest = s[:i], strings.TrimSpace(s[i+1:])
		}
		switch {
		case word == "title":
			c.Title = unquote(rest)
		case word == "x-axis" || word == "y-axis":
			lo, hi, _ := strings.Cut(rest, "-->")
			if lo = unquote(lo); lo == "" {
				return nil, syntax.Errorf(line, CodeStatement, "%s needs a label", word)
			}
			if word == "x-axis" {
				c.XLeft, c.XRight = lo, unquote(hi)
			} else {
				c.YBottom, c.YTop = lo, unquote(hi)
			}
		case word == "classDef":
			m := classDef.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "classDef needs a name and properties, found %q", s)
			}
			st := classes[m[1]]
			st.apply(m[2])
			classes[m[1]] = st
		case quadrant.MatchString(s):
			m := quadrant.FindStringSubmatch(s)
			c.Quadrants[m[1][0]-'1'] = unquote(m[2])
		default:
			m := point.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
			}
			x, errX := strconv.ParseFloat(m[3], 64)
			y, errY := strconv.ParseFloat(m[4], 64)
			if errX != nil || errY != nil || !(x >= 0 && x <= 1) || !(y >= 0 && y <= 1) {
				return nil, syntax.Errorf(line, CodePoint, "point coordinates must be numbers from 0 to 1, found [%s, %s]", m[3], m[4])
			}
			if len(c.Points) >= maxPoints {
				return nil, syntax.Errorf(line, CodePoint, "too many points (limit %d)", maxPoints)
			}
			pt := Point{Name: unquote(m[1]), X: x, Y: y, Line: line}
			pt.Style.apply(m[5])
			classOf = append(classOf, m[2])
			c.Points = append(c.Points, pt)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"quadrantChart\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	// Własności zapisane przy punkcie mają pierwszeństwo przed klasą.
	for i := range c.Points {
		own, st := c.Points[i].Style, classes[classOf[i]]
		if own.Radius > 0 {
			st.Radius = own.Radius
		}
		if own.Color != "" {
			st.Color = own.Color
		}
		if own.StrokeColor != "" {
			st.StrokeColor = own.StrokeColor
		}
		if own.StrokeWidth > 0 {
			st.StrokeWidth = own.StrokeWidth
		}
		c.Points[i].Style = st
	}
	c.AccTitle, c.AccDescr = acc.Title, acc.Descr
	return c, nil
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // ćwiartki w tonach palety
	Shadows    bool // przyjmowane dla zgodności opcji; wykres nie ma cieni poza punktami
}

const (
	colorText  = "#1b1f24"
	colorMuted = "#5b6570"
	colorLine  = "#3b4754"
	colorPoint = "#3b4754"
)

// Wypełnienia ćwiartek w rysunku neutralnym: ćwiartki stykające się bokiem różnią się jasnością.
var grays = [4]string{"#e3e8ef", "#f6f8fa", "#e3e8ef", "#f6f8fa"}

// Layout rozmieszcza wykres: kwadrat podzielony na cztery ćwiartki z tytułami, punkty z nazwami,
// opisy osi pod wykresem i z jego lewej strony.
func Layout(c *Chart, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	width := func(s string) float64 { return o.Measure(s, fs) }
	title := c.AccTitle
	if title == "" {
		title = c.Title
	}
	sc := &scene.Scene{Title: title, Desc: c.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}
	block := func(lines []string, x, mid float64, anchor, role, fill string, bold bool) {
		top := mid - float64(len(lines))*lh/2
		for k, s := range lines {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: top + float64(k)*lh + lh/2 + 0.35*fs,
				W: width(s), H: lh, Text: s, Anchor: anchor, Bold: bold, Fill: fill})
		}
	}
	wrap := func(s string, limit float64) []string {
		if s == "" {
			return nil
		}
		return textwrap.Lines([]string{s}, limit, width)
	}

	size := u(400)
	half := size / 2
	// Opisy osi pionowej są obrócone i czytane z dołu do góry, każdy przy swojej połowie wykresu.
	bottom, top := wrap(c.YBottom, half-u(12)), wrap(c.YTop, half-u(12))
	columns := max(len(bottom), len(top))
	x0 := 0.0
	if columns > 0 {
		x0 = float64(columns)*lh + u(8)
	}
	y0 := 0.0
	if c.Title != "" {
		block([]string{c.Title}, x0+half, lh/2, "middle", "title", colorText, false)
		y0 = lh + u(12)
	}
	type box struct{ x0, y0, x1, y1 float64 }
	hits := func(a, b box) bool { return a.x0 < b.x1 && b.x0 < a.x1 && a.y0 < b.y1 && b.y0 < a.y1 }
	small := math.Round(fs * 0.75)
	smallH := math.Round(small * 1.25)
	// Kropki: położenie i wygląd ze źródła.
	type dot struct {
		x, y, r        float64
		fill, stroke   string
		strokeW, nameW float64
	}
	dots := make([]dot, len(c.Points))
	var taken []box // kropki, a potem tytuły ćwiartek i nazwy już postawione
	for i, p := range c.Points {
		d := dot{x: x0 + p.X*size, y: y0 + (1-p.Y)*size, r: u(5), fill: colorPoint, stroke: "#ffffff", strokeW: 1, nameW: o.Measure(p.Name, small)}
		if p.Radius > 0 {
			d.r = u(p.Radius)
		}
		if p.Color != "" {
			d.fill = p.Color
		}
		if p.StrokeColor != "" {
			d.stroke = p.StrokeColor
		}
		if p.StrokeWidth > 0 {
			d.strokeW = u(p.StrokeWidth)
		}
		dots[i] = d
		taken = append(taken, box{d.x - d.r - u(2), d.y - d.r - u(2), d.x + d.r + u(2), d.y + d.r + u(2)})
	}
	// Ćwiartki: 0 prawa górna, 1 lewa górna, 2 lewa dolna, 3 prawa dolna. Tytuł stoi u góry
	// ćwiartki; gdy leży tam kropka, zsuwa się tuż pod nią, najwyżej do połowy ćwiartki.
	corner := [4][2]float64{{x0 + half, y0}, {x0, y0}, {x0, y0 + half}, {x0 + half, y0 + half}}
	for q := 0; q < 4; q++ {
		fill := grays[q]
		if o.Colors {
			fill = scene.Palette[q].Fill
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "quadrant", X: corner[q][0], Y: corner[q][1], W: half, H: half, Fill: fill})
		lines := wrap(c.Quadrants[q], half-2*u(12))
		if len(lines) == 0 {
			continue
		}
		tw, th := 0.0, float64(len(lines))*lh
		for _, line := range lines {
			tw = math.Max(tw, width(line))
		}
		cx := corner[q][0] + half/2
		top := corner[q][1] + u(10)
		for moved := true; moved; {
			moved = false
			for _, d := range taken[:len(dots)] {
				if at := (box{cx - tw/2, top, cx + tw/2, top + th}); hits(at, d) && d.y1+u(2)+th <= corner[q][1]+half/2+th {
					top, moved = d.y1+u(2), true
				}
			}
		}
		taken = append(taken, box{cx - tw/2, top, cx + tw/2, top + th})
		for k, line := range lines {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerShape, Role: "quadrant-title", X: cx,
				Y: top + float64(k)*lh + lh/2 + 0.35*fs, W: width(line), H: lh, Text: line, Anchor: "middle", Bold: true, Fill: colorMuted})
		}
	}
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0, Y: y0, W: size, H: size, Stroke: colorLine, StrokeWidth: 1.5})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0 + half, Y: y0, X2: x0 + half, Y2: y0 + size, Stroke: colorLine, StrokeWidth: 1})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0, Y: y0 + half, X2: x0 + size, Y2: y0 + half, Stroke: colorLine, StrokeWidth: 1})

	// Nazwa punktu mniejszym pismem: pod kropką, a gdy nachodziłaby tam na tytuł ćwiartki, inną
	// kropkę albo nazwę, nad kropką, z prawej albo z lewej. Wygrywa miejsce bez kolizji, w obrębie
	// wykresu i, jeśli się da, bez przecinania linii podziału.
	from := len(sc.Items)
	for i, p := range c.Points {
		d := dots[i]
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "quadrant-point", X: d.x, Y: d.y, R: d.r, Fill: d.fill, Stroke: d.stroke, StrokeWidth: d.strokeW})
		w, gap := d.nameW, d.r+u(3)
		clampX := func(cx float64) float64 { return math.Min(math.Max(cx, x0+w/2+u(4)), x0+size-w/2-u(4)) }
		candidates := []box{
			{clampX(d.x) - w/2, d.y + gap, clampX(d.x) + w/2, d.y + gap + smallH},
			{clampX(d.x) - w/2, d.y - gap - smallH, clampX(d.x) + w/2, d.y - gap},
			{d.x + gap + u(1), d.y - smallH/2, d.x + gap + u(1) + w, d.y + smallH/2},
			{d.x - gap - u(1) - w, d.y - smallH/2, d.x - gap - u(1), d.y + smallH/2},
		}
		best, bestCost := candidates[0], math.Inf(1)
		for k, cand := range candidates {
			cost := float64(k)
			if cand.x0 < x0+u(2) || cand.x1 > x0+size-u(2) || cand.y0 < y0+u(2) || cand.y1 > y0+size-u(2) {
				cost += 1000
			}
			for j, t := range taken {
				if j != i && hits(cand, t) {
					cost += 100
				}
			}
			if (cand.x0 < x0+half && cand.x1 > x0+half) || (cand.y0 < y0+half && cand.y1 > y0+half) {
				cost += 10
			}
			if cost < bestCost {
				best, bestCost = cand, cost
			}
		}
		taken = append(taken, best)
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "quadrant-point-label", X: (best.x0 + best.x1) / 2, Y: (best.y0+best.y1)/2 + 0.35*small,
			W: w, H: smallH, Text: p.Name, Anchor: "middle", FontSize: small, Fill: colorText})
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "quadrant-point")
	}

	// Opis osi pionowej: linie stoją obok siebie, pierwsza najdalej od wykresu.
	vertical := func(lines []string, mid float64) {
		for k, s := range lines {
			x := x0 - u(8) - float64(len(lines)-1-k)*lh - 0.3*fs
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "axis-label", X: x, Y: mid, W: width(s), H: lh,
				Text: s, Anchor: "middle", Vertical: true, Fill: colorMuted})
		}
	}
	vertical(bottom, y0+half+half/2)
	vertical(top, y0+half/2)
	y := y0 + size + u(8)
	left, right := wrap(c.XLeft, half-u(12)), wrap(c.XRight, half-u(12))
	if len(right) == 0 && len(left) > 0 {
		block(left, x0+half, y+float64(len(left))*lh/2, "middle", "axis-label", colorMuted, false)
	} else {
		if len(left) > 0 {
			block(left, x0+half/2, y+float64(len(left))*lh/2, "middle", "axis-label", colorMuted, false)
		}
		if len(right) > 0 {
			block(right, x0+half+half/2, y+float64(len(right))*lh/2, "middle", "axis-label", colorMuted, false)
		}
	}
	return sc
}
