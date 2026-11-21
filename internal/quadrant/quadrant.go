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
	point    = regexp.MustCompile(`^(.+?)(?::::[\p{L}\p{N}_-]+)?\s*:\s*\[\s*([^,\]]+?)\s*,\s*([^,\]]+?)\s*\](.*)$`)
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
			// wygląd punktów ze źródła nie jest rysowany
		case quadrant.MatchString(s):
			m := quadrant.FindStringSubmatch(s)
			c.Quadrants[m[1][0]-'1'] = unquote(m[2])
		default:
			m := point.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
			}
			x, errX := strconv.ParseFloat(m[2], 64)
			y, errY := strconv.ParseFloat(m[3], 64)
			if errX != nil || errY != nil || !(x >= 0 && x <= 1) || !(y >= 0 && y <= 1) {
				return nil, syntax.Errorf(line, CodePoint, "point coordinates must be numbers from 0 to 1, found [%s, %s]", m[2], m[3])
			}
			if len(c.Points) >= maxPoints {
				return nil, syntax.Errorf(line, CodePoint, "too many points (limit %d)", maxPoints)
			}
			c.Points = append(c.Points, Point{Name: unquote(m[1]), X: x, Y: y, Line: line})
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"quadrantChart\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
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
	// Opisy osi pionowej stoją poziomo z lewej strony, każdy przy swojej połowie wykresu.
	bottom, top := wrap(c.YBottom, u(120)), wrap(c.YTop, u(120))
	sideW := 0.0
	for _, s := range append(append([]string{}, bottom...), top...) {
		sideW = math.Max(sideW, width(s))
	}
	x0 := 0.0
	if sideW > 0 {
		x0 = math.Round(sideW + u(12))
	}
	y0 := 0.0
	if c.Title != "" {
		block([]string{c.Title}, x0+half, lh/2, "middle", "title", colorText, false)
		y0 = lh + u(12)
	}
	// Miejsca zajęte przez punkty i ich nazwy: tytuł ćwiartki staje tam, gdzie ich nie zasłania.
	type box struct{ x0, y0, x1, y1 float64 }
	type label struct {
		x      float64
		anchor string
		w      float64
	}
	labels := make([]label, len(c.Points))
	var taken []box
	for i, p := range c.Points {
		px, py := x0+p.X*size, y0+(1-p.Y)*size
		w := width(p.Name)
		// Nazwa stoi po prawej stronie punktu, a przy prawej krawędzi wykresu po lewej.
		l := label{px + u(9), "start", w}
		if l.x+w > x0+size-u(4) {
			l = label{px - u(9) - w, "end", w}
		}
		labels[i] = l
		taken = append(taken, box{math.Min(l.x, px-u(5)) - u(2), py - lh/2, math.Max(l.x+w, px+u(5)) + u(2), py + lh/2})
	}
	// Ćwiartki: 0 prawa górna, 1 lewa górna, 2 lewa dolna, 3 prawa dolna.
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
		// Tytuł u góry ćwiartki; gdy stoją tam punkty, u dołu, a w ostateczności pośrodku.
		top := corner[q][1] + u(10)
		for _, candidate := range []float64{top, corner[q][1] + half - u(10) - th, corner[q][1] + (half-th)/2} {
			free := true
			for _, t := range taken {
				if t.x0 < cx+tw/2 && cx-tw/2 < t.x1 && t.y0 < candidate+th && candidate < t.y1 {
					free = false
					break
				}
			}
			if free {
				top = candidate
				break
			}
		}
		for k, line := range lines {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerShape, Role: "quadrant-title", X: cx, Y: top + float64(k)*lh + lh/2 + 0.35*fs,
				W: width(line), H: lh, Text: line, Anchor: "middle", Bold: true, Fill: colorMuted})
		}
	}
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0, Y: y0, W: size, H: size, Stroke: colorLine, StrokeWidth: 1.5})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0 + half, Y: y0, X2: x0 + half, Y2: y0 + size, Stroke: colorLine, StrokeWidth: 1})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "quadrant-frame", X: x0, Y: y0 + half, X2: x0 + size, Y2: y0 + half, Stroke: colorLine, StrokeWidth: 1})

	from := len(sc.Items)
	for i, p := range c.Points {
		px, py := x0+p.X*size, y0+(1-p.Y)*size
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "quadrant-point", X: px, Y: py, R: u(5), Fill: colorPoint, Stroke: "#ffffff", StrokeWidth: 1})
		l := labels[i]
		// Podkład pod nazwą: linie podziału i tytuł ćwiartki nie przekreślają tekstu.
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabel, Role: "quadrant-point-background", X: l.x - u(2), Y: py - lh/2, W: l.w + u(4), H: lh,
			RX: u(3), Fill: "#ffffff", FillOpacity: 0.85})
		lx := l.x
		if l.anchor == "end" {
			lx += l.w
		}
		block([]string{p.Name}, lx, py, l.anchor, "quadrant-point-label", colorText, false)
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "quadrant-point")
	}

	if len(bottom) > 0 {
		block(bottom, x0-u(12), y0+half+half/2, "end", "axis-label", colorMuted, false)
	}
	if len(top) > 0 {
		block(top, x0-u(12), y0+half/2, "end", "axis-label", colorMuted, false)
	}
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
