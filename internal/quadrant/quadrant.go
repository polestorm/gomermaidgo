// Package quadrant rozbiera i rysuje wykres ćwiartkowy (quadrantChart).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package quadrant

import (
	"math"
	"regexp"
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
	point    = regexp.MustCompile(`^(.+?)(?::::([\p{L}\p{N}_-]+))?\s*:\s*\[\s*([^,\]]+?)\s*,\s*([^,\]]+?)\s*\](.*)$`)
	classDef = regexp.MustCompile(`^classDef\s+([\p{L}\p{N}_-]+)\s+(.+)$`)
	// Kolor: zapis szesnastkowy albo nazwa; nic, co mogłoby wyjść poza atrybut.
	paint    = regexp.MustCompile(`^(#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6}|[a-zA-Z]+)$`)
	quadrant = regexp.MustCompile(`^quadrant-([1-4])\s+(.+)$`)
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
	// Ćwiartki: 0 prawa górna, 1 lewa górna, 2 lewa dolna, 3 prawa dolna. Tytuł stoi u góry ćwiartki.
	corner := [4][2]float64{{x0 + half, y0}, {x0, y0}, {x0, y0 + half}, {x0 + half, y0 + half}}
	for q := 0; q < 4; q++ {
		fill := grays[q]
		if o.Colors {
			fill = scene.Palette[q].Fill
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "quadrant", X: corner[q][0], Y: corner[q][1], W: half, H: half, Fill: fill})
		for k, line := range wrap(c.Quadrants[q], half-2*u(12)) {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerShape, Role: "quadrant-title", X: corner[q][0] + half/2,
				Y: corner[q][1] + u(10) + float64(k)*lh + lh/2 + 0.35*fs, W: width(line), H: lh, Text: line, Anchor: "middle", Bold: true, Fill: colorMuted})
		}
	}
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0, Y: y0, W: size, H: size, Stroke: colorLine, StrokeWidth: 1.5})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0 + half, Y: y0, X2: x0 + half, Y2: y0 + size, Stroke: colorLine, StrokeWidth: 1})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0, Y: y0 + half, X2: x0 + size, Y2: y0 + half, Stroke: colorLine, StrokeWidth: 1})

	// Punkty: kropka, a pod nią nazwa mniejszym pismem, na własnym podkładzie.
	small := math.Round(fs * 0.75)
	smallH := math.Round(small * 1.25)
	from := len(sc.Items)
	for _, p := range c.Points {
		px, py := x0+p.X*size, y0+(1-p.Y)*size
		r, fill, stroke, sw := u(5), colorPoint, "#ffffff", 1.0
		if p.Radius > 0 {
			r = u(p.Radius)
		}
		if p.Color != "" {
			fill = p.Color
		}
		if p.StrokeColor != "" {
			stroke = p.StrokeColor
		}
		if p.StrokeWidth > 0 {
			sw = u(p.StrokeWidth)
		}
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "quadrant-point", X: px, Y: py, R: r, Fill: fill, Stroke: stroke, StrokeWidth: sw})
		w := o.Measure(p.Name, small)
		// Nazwa wyśrodkowana pod kropką; przy krawędzi wykresu dosunięta tak, żeby z niego nie wyszła,
		// a przy dolnej krawędzi przeniesiona nad kropkę.
		cx := math.Min(math.Max(px, x0+w/2+u(4)), x0+size-w/2-u(4))
		ty := py + r + u(3)
		if ty+smallH > y0+size-u(2) {
			ty = py - r - u(3) - smallH
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabel, Role: "quadrant-point-background", X: cx - w/2 - u(2), Y: ty, W: w + u(4), H: smallH,
			RX: u(3), Fill: "#ffffff", FillOpacity: 0.85})
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "quadrant-point-label", X: cx, Y: ty + smallH/2 + 0.35*small,
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
