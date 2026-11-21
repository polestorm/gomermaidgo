// Package radar rozbiera i rysuje wykres radarowy (radar).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package radar

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni wykresu radarowego.
const (
	CodeHeader    = "radar-header"
	CodeStatement = "radar-statement"
	CodeData      = "radar-data"
	CodeEmpty     = "radar-empty"
	maxAxes       = 24
	maxCurves     = 12
)

// Axis to oś wykresu: identyfikator i podpis.
type Axis struct{ ID, Label string }

// Curve to seria: podpis i wartość na każdej osi, w kolejności osi.
type Curve struct {
	Label  string
	Values []float64
	Line   int
}

// Chart to wykres radarowy.
type Chart struct {
	Title    string
	Axes     []Axis
	Curves   []Curve
	Min, Max float64
	HasMax   bool
	Ticks    int
	Polygon  bool // siatka z wielokątów zamiast okręgów
	Legend   bool
	AccTitle string
	AccDescr string
}

var (
	named = regexp.MustCompile(`^([\p{L}\p{N}_-]+)\s*(?:\[\s*"([^"]*)"\s*\])?$`)
	curve = regexp.MustCompile(`^curve\s+([\p{L}\p{N}_-]+)\s*(?:\[\s*"([^"]*)"\s*\])?\s*\{(.*)\}$`)
)

// Parse rozbiera źródło wykresu radarowego.
func Parse(source string) (*Chart, error) {
	c := &Chart{Ticks: 5, Legend: true}
	var acc syntax.Accessibility
	seenHeader := false
	type raw struct {
		label, body string
		line        int
	}
	var curves []raw
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, text := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(text)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "radar-beta" && s != "radar" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"radar-beta\", found %q", s)
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
		number := func() (float64, error) {
			v, err := strconv.ParseFloat(rest, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return 0, syntax.Errorf(line, CodeStatement, "%s needs a number, found %q", word, rest)
			}
			return v, nil
		}
		switch word {
		case "title":
			c.Title = strings.Trim(rest, `"`)
		case "axis":
			for _, part := range strings.Split(rest, ",") {
				m := named.FindStringSubmatch(strings.TrimSpace(part))
				if m == nil {
					return nil, syntax.Errorf(line, CodeStatement, "axis cannot be read: %q", strings.TrimSpace(part))
				}
				label := m[2]
				if label == "" {
					label = m[1]
				}
				if len(c.Axes) >= maxAxes {
					return nil, syntax.Errorf(line, CodeStatement, "too many axes (limit %d)", maxAxes)
				}
				c.Axes = append(c.Axes, Axis{m[1], label})
			}
		case "curve":
			m := curve.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeData, "curve needs a name and values in braces, found %q", s)
			}
			label := m[2]
			if label == "" {
				label = m[1]
			}
			if len(curves) >= maxCurves {
				return nil, syntax.Errorf(line, CodeData, "too many curves (limit %d)", maxCurves)
			}
			curves = append(curves, raw{label, m[3], line})
		case "max", "min":
			v, err := number()
			if err != nil {
				return nil, err
			}
			if word == "max" {
				c.Max, c.HasMax = v, true
			} else {
				c.Min = v
			}
		case "ticks":
			v, err := number()
			if err != nil || v < 1 || v > 20 || v != math.Trunc(v) {
				return nil, syntax.Errorf(line, CodeStatement, "ticks needs a whole number from 1 to 20, found %q", rest)
			}
			c.Ticks = int(v)
		case "graticule":
			if rest != "circle" && rest != "polygon" {
				return nil, syntax.Errorf(line, CodeStatement, "graticule is circle or polygon, found %q", rest)
			}
			c.Polygon = rest == "polygon"
		case "showLegend":
			if rest != "true" && rest != "false" {
				return nil, syntax.Errorf(line, CodeStatement, "showLegend is true or false, found %q", rest)
			}
			c.Legend = rest == "true"
		default:
			return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"radar-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(c.Axes) < 3 {
		return nil, syntax.Errorf(0, CodeEmpty, "the chart needs at least three axes")
	}
	if len(curves) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the chart needs at least one curve")
	}
	// Wartości serii: lista w kolejności osi albo pary oś: wartość.
	index := map[string]int{}
	for i, a := range c.Axes {
		index[a.ID] = i
	}
	top := math.Inf(-1)
	for _, r := range curves {
		values := make([]float64, len(c.Axes))
		parts := strings.Split(r.body, ",")
		if len(parts) != len(c.Axes) {
			return nil, syntax.Errorf(r.line, CodeData, "curve has %d values, expected %d", len(parts), len(c.Axes))
		}
		seen := map[int]bool{}
		for i, part := range parts {
			at, text := i, strings.TrimSpace(part)
			if key, value, ok := strings.Cut(text, ":"); ok {
				k, found := index[strings.TrimSpace(key)]
				if !found {
					return nil, syntax.Errorf(r.line, CodeData, "curve refers to unknown axis %q", strings.TrimSpace(key))
				}
				at, text = k, strings.TrimSpace(value)
			}
			v, err := strconv.ParseFloat(text, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, syntax.Errorf(r.line, CodeData, "%q is not a number", text)
			}
			if seen[at] {
				return nil, syntax.Errorf(r.line, CodeData, "curve gives two values for axis %q", c.Axes[at].ID)
			}
			seen[at] = true
			values[at] = v
			top = math.Max(top, v)
		}
		c.Curves = append(c.Curves, Curve{r.label, values, r.line})
	}
	if !c.HasMax {
		c.Max = top
	}
	if c.Max <= c.Min {
		return nil, syntax.Errorf(0, CodeData, "the largest value (%v) must be above the smallest (%v)", c.Max, c.Min)
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
	Colors     bool // serie w kolorach palety
	Shadows    bool // przyjmowane dla zgodności opcji; wykres nie ma cieni
}

const (
	colorText  = "#1b1f24"
	colorMuted = "#5b6570"
	colorGrid  = "#c9d1da"
	colorLine  = "#3b4754"
)

var dashes = []string{"", "7 4", "2 3", "9 3 2 3"}

// Layout rozmieszcza wykres: osie rozchodzą się ze środka, pierwsza w górę, kolejne zgodnie
// z ruchem wskazówek zegara; siatka z okręgów albo wielokątów; każda seria to wielokąt przez
// swoje wartości. Serie różnią się kreskowaniem, więc wykres czyta się bez koloru.
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
	text := func(s string, x, mid float64, anchor, role, fill string) {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: mid + 0.35*fs, W: width(s), H: lh, Text: s, Anchor: anchor, Fill: fill})
	}

	radius := u(150)
	n := len(c.Axes)
	// Miejsce na podpisy osi po bokach i u góry.
	side := 0.0
	for _, a := range c.Axes {
		side = math.Max(side, width(a.Label))
	}
	cx, cy := side+u(14)+radius, lh+u(10)+radius
	if c.Title != "" {
		text(c.Title, cx, lh/2, "middle", "title", colorText)
		cy += lh + u(8)
	}
	at := func(i int, r float64) scene.Point {
		a := 2 * math.Pi * float64(i) / float64(n)
		return scene.Point{X: cx + r*math.Sin(a), Y: cy - r*math.Cos(a)}
	}
	for t := 1; t <= c.Ticks; t++ {
		r := radius * float64(t) / float64(c.Ticks)
		if c.Polygon {
			pts := make([]scene.Point, n)
			for i := range pts {
				pts[i] = at(i, r)
			}
			sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerBackground, Role: "radar-grid", Points: pts, Stroke: colorGrid, StrokeWidth: 1})
		} else {
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerBackground, Role: "radar-grid", X: cx, Y: cy, R: r, Stroke: colorGrid, StrokeWidth: 1})
		}
	}
	for i, a := range c.Axes {
		end := at(i, radius)
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerBackground, Role: "radar-axis", X: cx, Y: cy, X2: end.X, Y2: end.Y, Stroke: colorGrid, StrokeWidth: 1})
		p := at(i, radius+u(12))
		anchor := "middle"
		if p.X > cx+u(4) {
			anchor = "start"
		} else if p.X < cx-u(4) {
			anchor = "end"
		}
		text(a.Label, p.X, p.Y, anchor, "radar-axis-label", colorMuted)
	}
	for k, cv := range c.Curves {
		stroke := colorLine
		if o.Colors {
			stroke = scene.Palette[k%len(scene.Palette)].Stroke
		}
		pts := make([]scene.Point, n)
		for i, v := range cv.Values {
			share := (math.Max(c.Min, math.Min(c.Max, v)) - c.Min) / (c.Max - c.Min)
			pts[i] = at(i, radius*share)
		}
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "radar-curve", Points: pts, Fill: stroke, FillOpacity: 0.12,
			Stroke: stroke, StrokeWidth: 2, Dash: dashes[k%len(dashes)]})
		for _, p := range pts {
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "radar-point", X: p.X, Y: p.Y, R: u(3), Fill: "#ffffff", Stroke: stroke, StrokeWidth: 1.5})
		}
	}
	if c.Legend {
		x, y := cx+radius+side+u(30), cy-radius
		for k, cv := range c.Curves {
			stroke := colorLine
			if o.Colors {
				stroke = scene.Palette[k%len(scene.Palette)].Stroke
			}
			mid := y + float64(k)*(lh+u(4)) + lh/2
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "radar-swatch", X: x, Y: mid, X2: x + u(22), Y2: mid, Stroke: stroke, StrokeWidth: 2, Dash: dashes[k%len(dashes)]})
			text(cv.Label, x+u(28), mid, "start", "radar-legend", colorText)
		}
	}
	return sc
}
