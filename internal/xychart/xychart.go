// Package xychart rozbiera i rysuje wykres słupkowy i liniowy (xychart).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package xychart

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni wykresu.
const (
	CodeHeader      = "xychart-header"
	CodeUnsupported = "xychart-unsupported"
	CodeStatement   = "xychart-statement"
	CodeData        = "xychart-data"
	CodeEmpty       = "xychart-empty"
	maxPoints       = 200
	maxSeries       = 12
)

// Series to seria danych: słupki albo linia.
type Series struct {
	Bar    bool
	Name   string
	Values []float64
	Line   int
}

// Chart to wykres.
type Chart struct {
	Title      string
	XTitle     string
	YTitle     string
	Horizontal bool     // wykres poziomy: kategorie z góry na dół, wartości w poziomie
	Categories []string // etykiety osi poziomej; puste przy osi liczbowej
	XRange     *[2]float64
	YRange     *[2]float64
	Series     []Series
	AccTitle   string
	AccDescr   string
}

var (
	rangeTail = regexp.MustCompile(`^(.*?)\s*(-?\d+(?:\.\d+)?)\s*-->\s*(-?\d+(?:\.\d+)?)$`)
	listTail  = regexp.MustCompile(`^(.*?)\s*\[(.*)\]$`)
)

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

// Parse rozbiera źródło wykresu.
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
			switch s {
			case "xychart-beta", "xychart":
			case "xychart-beta horizontal", "xychart horizontal":
				c.Horizontal = true
			default:
				return nil, syntax.Errorf(line, CodeHeader, "expected \"xychart-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		word, rest := s, ""
		if i := strings.IndexAny(s, " \t["); i >= 0 {
			word, rest = s[:i], strings.TrimSpace(s[i:])
		}
		switch word {
		case "title":
			c.Title = unquote(rest)
		case "x-axis", "y-axis":
			title, list, rng := rest, "", (*[2]float64)(nil)
			if m := listTail.FindStringSubmatch(rest); m != nil {
				title, list = m[1], m[2]
			} else if m := rangeTail.FindStringSubmatch(rest); m != nil {
				lo, _ := strconv.ParseFloat(m[2], 64)
				hi, _ := strconv.ParseFloat(m[3], 64)
				if hi <= lo {
					return nil, syntax.Errorf(line, CodeStatement, "axis range must grow: %s --> %s", m[2], m[3])
				}
				title, rng = m[1], &[2]float64{lo, hi}
			}
			if word == "y-axis" {
				if list != "" || strings.Contains(rest, "[") {
					return nil, syntax.Errorf(line, CodeStatement, "the y axis takes a title and a range, not a list of categories")
				}
				c.YTitle, c.YRange = unquote(title), rng
				continue
			}
			c.XTitle, c.XRange = unquote(title), rng
			if strings.Contains(rest, "[") {
				c.Categories = nil
				for _, part := range strings.Split(list, ",") {
					if part = unquote(part); part == "" {
						return nil, syntax.Errorf(line, CodeStatement, "empty category on the x axis")
					}
					c.Categories = append(c.Categories, part)
				}
				if len(c.Categories) > maxPoints {
					return nil, syntax.Errorf(line, CodeStatement, "too many categories (limit %d)", maxPoints)
				}
			}
		case "bar", "line":
			m := listTail.FindStringSubmatch(rest)
			if m == nil {
				return nil, syntax.Errorf(line, CodeData, "%s needs a list of numbers in square brackets", word)
			}
			sr := Series{Bar: word == "bar", Name: unquote(m[1]), Line: line}
			for _, part := range strings.Split(m[2], ",") {
				v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
				if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
					return nil, syntax.Errorf(line, CodeData, "%q is not a number", strings.TrimSpace(part))
				}
				sr.Values = append(sr.Values, v)
			}
			if len(sr.Values) > maxPoints {
				return nil, syntax.Errorf(line, CodeData, "too many values in a series (limit %d)", maxPoints)
			}
			if len(c.Series) >= maxSeries {
				return nil, syntax.Errorf(line, CodeData, "too many series (limit %d)", maxSeries)
			}
			c.Series = append(c.Series, sr)
		default:
			return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"xychart-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(c.Series) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the chart needs at least one bar or line series")
	}
	want := len(c.Series[0].Values)
	if len(c.Categories) > 0 {
		want = len(c.Categories)
	}
	for _, sr := range c.Series {
		if len(sr.Values) != want {
			return nil, syntax.Errorf(sr.Line, CodeData, "series has %d values, expected %d", len(sr.Values), want)
		}
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
	Shadows    bool // cień pod słupkami
}

const (
	colorText  = "#1b1f24"
	colorMuted = "#5b6570"
	colorAxis  = "#3b4754"
	colorGrid  = "#d5dae1"
)

var (
	barGrays = []string{"#66727f", "#c9d1da", "#3b4754", "#e3e8ef"}
	dashes   = []string{"", "7 4", "2 3", "9 3 2 3"}
)

// Ticks dobiera okrągły krok podziałki dla zakresu i zwraca jej wartości.
func Ticks(lo, hi float64) (ticks []float64, step float64) {
	raw := (hi - lo) / 5
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	step = 10 * mag
	for _, m := range []float64{1, 2, 5} {
		if raw <= m*mag {
			step = m * mag
			break
		}
	}
	for v := math.Ceil(lo/step-1e-9) * step; v <= hi+step*1e-9; v += step {
		ticks = append(ticks, math.Round(v/step)*step+0) // +0 zamienia ujemne zero na zwykłe
	}
	return ticks, step
}

// Number zapisuje wartość podziałki z tyloma miejscami po przecinku, ile wymaga krok.
func Number(v, step float64) string {
	decimals := 0
	if step < 1 {
		decimals = int(math.Ceil(-math.Log10(step) - 1e-9))
	}
	return strconv.FormatFloat(v, 'f', decimals, 64)
}

// Layout rozmieszcza wykres: oś pionowa z okrągłą podziałką i liniami siatki, kategorie na osi
// poziomej, słupki obok siebie w obrębie kategorii, linie z punktami.
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

	if c.Horizontal {
		return layoutHorizontal(c, o)
	}
	n := len(c.Series[0].Values)
	bars, lo, hi := valueRange(c)
	ticks, step := Ticks(lo, hi)
	labelW := 0.0
	for _, t := range ticks {
		labelW = math.Max(labelW, width(Number(t, step)))
	}

	y := 0.0
	plotX, plotW, plotH := math.Round(labelW+u(10)), u(520), u(280)
	if c.Title != "" {
		text(c.Title, plotX+plotW/2, lh/2, "middle", "title", colorText)
		y = lh + u(12)
	}
	if c.YTitle != "" {
		text(c.YTitle, 0, y+lh/2, "start", "axis-title", colorMuted)
		y += lh + u(6)
	}
	plotY := y + u(6)
	at := func(v float64) float64 { return plotY + plotH - (math.Max(lo, math.Min(hi, v))-lo)/(hi-lo)*plotH }
	for _, t := range ticks {
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerBackground, Role: "chart-grid", X: plotX, Y: at(t), X2: plotX + plotW, Y2: at(t), Stroke: colorGrid, StrokeWidth: 1})
		text(Number(t, step), plotX-u(8), at(t), "end", "axis-label", colorMuted)
	}

	band := plotW / float64(n)
	centre := func(i int) float64 { return plotX + band*(float64(i)+0.5) }
	tone := func(i int) string {
		if o.Colors {
			return scene.Distinct(i).Stroke
		}
		return barGrays[i%len(barGrays)]
	}
	from := len(sc.Items)
	group, k := band*0.7, 0
	zero := at(0)
	for i, sr := range c.Series {
		if !sr.Bar {
			continue
		}
		w := group / float64(bars)
		for j, v := range sr.Values {
			top, bottom := math.Min(at(v), zero), math.Max(at(v), zero)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "chart-bar", X: centre(j) - group/2 + float64(k)*w, Y: top, W: w, H: bottom - top,
				Fill: tone(i), Stroke: colorAxis, StrokeWidth: 1})
		}
		k++
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "chart-bar")
	}
	lines := 0
	for i, sr := range c.Series {
		if sr.Bar {
			continue
		}
		pts := make([]scene.Point, len(sr.Values))
		for j, v := range sr.Values {
			pts[j] = scene.Point{X: centre(j), Y: at(v)}
		}
		stroke := colorAxis
		if o.Colors {
			stroke = tone(i)
		}
		// Linie odróżnia wzór kreskowania, więc wykres czyta się także bez koloru.
		sc.Add(scene.Item{Kind: scene.Polyline, Layer: scene.LayerShape, Role: "chart-line", Points: pts, Stroke: stroke, StrokeWidth: 2, Dash: dashes[lines%len(dashes)]})
		for _, p := range pts {
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "chart-point", X: p.X, Y: p.Y, R: u(3.5), Fill: "#ffffff", Stroke: stroke, StrokeWidth: 1.5})
		}
		lines++
	}
	// Osie na wierzchu siatki i słupków.
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabel, Role: "chart-axis", X: plotX, Y: plotY, X2: plotX, Y2: plotY + plotH, Stroke: colorAxis, StrokeWidth: 1.5})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabel, Role: "chart-axis", X: plotX, Y: zero, X2: plotX + plotW, Y2: zero, Stroke: colorAxis, StrokeWidth: 1.5})

	// Etykiety osi poziomej: kategorie ze źródła albo wartości z zakresu liczbowego; gdy są
	// szersze niż ich miejsce, zostaje co któraś.
	labels := categoryLabels(c)
	widest := 0.0
	for _, s := range labels {
		widest = math.Max(widest, width(s))
	}
	every := int(math.Max(1, math.Ceil((widest+u(8))/band)))
	y = plotY + plotH + u(8)
	for i, s := range labels {
		if i%every == 0 {
			text(s, centre(i), y+lh/2, "middle", "axis-label", colorMuted)
		}
	}
	y += lh
	if c.XTitle != "" {
		y += u(4)
		text(c.XTitle, plotX+plotW/2, y+lh/2, "middle", "axis-title", colorMuted)
		y += lh
	}
	legend(sc, c, o, plotX, plotW, y, text, tone)
	return sc
}

// valueRange zwraca liczbę serii słupkowych i zakres osi wartości: ze źródła albo z danych,
// zaokrąglony do podziałki.
func valueRange(c *Chart) (bars int, lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, sr := range c.Series {
		if sr.Bar {
			bars++
		}
		for _, v := range sr.Values {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if c.YRange != nil {
		lo, hi = c.YRange[0], c.YRange[1]
	} else {
		// Słupek mierzy się od zera, więc zero należy do zakresu; sama linia może zaczynać wyżej.
		if bars > 0 {
			lo, hi = math.Min(lo, 0), math.Max(hi, 0)
		}
		if hi == lo {
			lo, hi = lo-1, hi+1
		}
		_, step := Ticks(lo, hi)
		lo, hi = math.Floor(lo/step+1e-9)*step, math.Ceil(hi/step-1e-9)*step
	}
	return bars, lo, hi
}

// categoryLabels zwraca etykiety osi kategorii: ze źródła albo wartości z zakresu liczbowego.
func categoryLabels(c *Chart) []string {
	n := len(c.Series[0].Values)
	labels := c.Categories
	if len(labels) == 0 {
		labels = make([]string, n)
		for i := range labels {
			labels[i] = strconv.Itoa(i + 1)
			if c.XRange != nil {
				v := c.XRange[0]
				if n > 1 {
					v += (c.XRange[1] - c.XRange[0]) * float64(i) / float64(n-1)
				}
				labels[i] = strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
			}
		}
	}
	return labels
}

// legend rysuje legendę pod wykresem, gdy serie mają nazwy.
func legend(sc *scene.Scene, c *Chart, o Options, plotX, plotW, y float64, text func(s string, x, mid float64, anchor, role, fill string), tone func(int) string) {
	fs := sc.FontSize
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	width := func(s string) float64 { return o.Measure(s, fs) }
	// Legenda, gdy serie mają nazwy.
	x := plotX
	y += u(10)
	lines := 0
	for i, sr := range c.Series {
		if !sr.Bar {
			lines++
		}
		if sr.Name == "" {
			continue
		}
		if sr.Bar {
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "chart-swatch", X: x, Y: y + lh/2 - u(6), W: u(18), H: u(12), Fill: tone(i), Stroke: colorAxis, StrokeWidth: 1})
		} else {
			stroke := colorAxis
			if o.Colors {
				stroke = tone(i)
			}
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "chart-swatch", X: x, Y: y + lh/2, X2: x + u(18), Y2: y + lh/2, Stroke: stroke, StrokeWidth: 2, Dash: dashes[(lines-1)%len(dashes)]})
		}
		text(sr.Name, x+u(24), y+lh/2, "start", "chart-legend", colorText)
		x += u(24) + width(sr.Name) + u(20)
		if x > plotX+plotW-u(80) {
			x, y = plotX, y+lh+u(4)
		}
	}
}

// layoutHorizontal rysuje wykres poziomy: kategorie z góry na dół przy osi pionowej, wartości
// wzdłuż osi poziomej, słupki od zera w prawo (ujemne w lewo).
func layoutHorizontal(c *Chart, o Options) *scene.Scene {
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
	n := len(c.Series[0].Values)
	bars, lo, hi := valueRange(c)
	ticks, step := Ticks(lo, hi)
	labels := categoryLabels(c)
	labelW := 0.0
	for _, s := range labels {
		labelW = math.Max(labelW, width(s))
	}
	// Tytuł osi kategorii stoi z lewej strony, obrócony wzdłuż osi.
	plotX := math.Round(labelW + u(10))
	if c.XTitle != "" {
		plotX += lh + u(6)
	}
	band := math.Max(u(280)/float64(n), lh+u(8))
	plotW, plotH := u(520), band*float64(n)
	plotY := 0.0
	if c.Title != "" {
		text(c.Title, plotX+plotW/2, lh/2, "middle", "title", colorText)
		plotY = lh + u(12)
	}
	at := func(v float64) float64 { return plotX + (math.Max(lo, math.Min(hi, v))-lo)/(hi-lo)*plotW }
	// Opisy podziałki pod wykresem; gdy są szersze niż odstęp podziałki, zostaje co któryś.
	widest := 0.0
	for _, t := range ticks {
		widest = math.Max(widest, width(Number(t, step)))
	}
	every := 1
	if len(ticks) > 1 {
		every = int(math.Max(1, math.Ceil((widest+u(10))/(plotW/float64(len(ticks)-1)))))
	}
	for i, t := range ticks {
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerBackground, Role: "chart-grid", X: at(t), Y: plotY, X2: at(t), Y2: plotY + plotH, Stroke: colorGrid, StrokeWidth: 1})
		if i%every == 0 {
			text(Number(t, step), at(t), plotY+plotH+u(8)+lh/2, "middle", "axis-label", colorMuted)
		}
	}
	centre := func(i int) float64 { return plotY + band*(float64(i)+0.5) }
	tone := func(i int) string {
		if o.Colors {
			return scene.Distinct(i).Stroke
		}
		return barGrays[i%len(barGrays)]
	}
	from := len(sc.Items)
	group, k := band*0.7, 0
	zero := at(0)
	for i, sr := range c.Series {
		if !sr.Bar {
			continue
		}
		h := group / float64(bars)
		for j, v := range sr.Values {
			left, right := math.Min(at(v), zero), math.Max(at(v), zero)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "chart-bar", X: left, Y: centre(j) - group/2 + float64(k)*h, W: right - left, H: h,
				Fill: tone(i), Stroke: colorAxis, StrokeWidth: 1})
		}
		k++
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "chart-bar")
	}
	lines := 0
	for i, sr := range c.Series {
		if sr.Bar {
			continue
		}
		pts := make([]scene.Point, len(sr.Values))
		for j, v := range sr.Values {
			pts[j] = scene.Point{X: at(v), Y: centre(j)}
		}
		stroke := colorAxis
		if o.Colors {
			stroke = tone(i)
		}
		sc.Add(scene.Item{Kind: scene.Polyline, Layer: scene.LayerShape, Role: "chart-line", Points: pts, Stroke: stroke, StrokeWidth: 2, Dash: dashes[lines%len(dashes)]})
		for _, p := range pts {
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "chart-point", X: p.X, Y: p.Y, R: u(3.5), Fill: "#ffffff", Stroke: stroke, StrokeWidth: 1.5})
		}
		lines++
	}
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabel, Role: "chart-axis", X: plotX, Y: plotY + plotH, X2: plotX + plotW, Y2: plotY + plotH, Stroke: colorAxis, StrokeWidth: 1.5})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabel, Role: "chart-axis", X: zero, Y: plotY, X2: zero, Y2: plotY + plotH, Stroke: colorAxis, StrokeWidth: 1.5})
	for i, s := range labels {
		text(s, plotX-u(8), centre(i), "end", "axis-label", colorMuted)
	}
	if c.XTitle != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "axis-title", X: 0.8 * lh, Y: plotY + plotH/2, W: width(c.XTitle), H: lh,
			Text: c.XTitle, Anchor: "middle", Vertical: true, Fill: colorMuted})
	}
	y := plotY + plotH + u(8) + lh
	if c.YTitle != "" {
		y += u(4)
		text(c.YTitle, plotX+plotW/2, y+lh/2, "middle", "axis-title", colorMuted)
		y += lh
	}
	legend(sc, c, o, plotX, plotW, y, text, tone)
	return sc
}
