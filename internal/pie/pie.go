// Package pie rozbiera i rysuje wykres kołowy (pie).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package pie

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni wykresu kołowego.
const (
	CodeHeader    = "pie-header"
	CodeStatement = "pie-statement"
	CodeValue     = "pie-value"
	CodeEmpty     = "pie-empty"
	maxSlices     = 24
)

// Slice to jeden wycinek: etykieta i wartość.
type Slice struct {
	Label string
	Value float64
	Text  string // wartość tak, jak zapisano ją w źródle
	Line  int
}

// Chart to wykres kołowy.
type Chart struct {
	Title    string
	ShowData bool
	Slices   []Slice
	AccTitle string
	AccDescr string
}

var (
	header = regexp.MustCompile(`^pie(?:\s+(showData))?(?:\s+title\s+(.*))?$`)
	slice  = regexp.MustCompile(`^"([^"]*)"\s*:\s*(\S+)$`)
)

// Parse rozbiera źródło wykresu kołowego.
func Parse(source string) (*Chart, error) {
	c := &Chart{}
	var acc syntax.Accessibility
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
			m := header.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"pie\", optionally with \"showData\" and \"title\", found %q", s)
			}
			c.ShowData, c.Title = m[1] != "", strings.TrimSpace(m[2])
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		switch {
		case s == "showData":
			c.ShowData = true
		case s == "title" || strings.HasPrefix(s, "title "):
			c.Title = strings.TrimSpace(strings.TrimPrefix(s, "title"))
		default:
			m := slice.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "expected a slice written as \"label\" : value, found %q", s)
			}
			label := strings.TrimSpace(m[1])
			v, err := strconv.ParseFloat(m[2], 64)
			switch {
			case label == "":
				return nil, syntax.Errorf(line, CodeStatement, "slice label is empty")
			case err != nil || math.IsNaN(v) || math.IsInf(v, 0):
				return nil, syntax.Errorf(line, CodeValue, "slice value %q is not a number", m[2])
			case v < 0:
				return nil, syntax.Errorf(line, CodeValue, "slice value %q is negative", m[2])
			case seen[label]:
				return nil, syntax.Errorf(line, CodeStatement, "slice %q is defined twice", label)
			case len(c.Slices) >= maxSlices:
				return nil, syntax.Errorf(line, CodeStatement, "too many slices (limit %d)", maxSlices)
			}
			seen[label] = true
			c.Slices = append(c.Slices, Slice{Label: label, Value: v, Text: m[2], Line: line})
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"pie\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	total := 0.0
	for _, s := range c.Slices {
		total += s.Value
	}
	if total <= 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the chart needs at least one slice with a value above zero")
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
	Colors     bool // wycinki w kolorach palety zamiast odcieni szarości
	Shadows    bool // cień pod kołem
}

const (
	colorText  = "#1b1f24"
	colorLight = "#ffffff"
	colorEdge  = "#3b4754"
)

// Odcienie domyślne: sąsiednie wycinki różnią się jasnością, więc wykres czyta się także
// w druku czarno-białym. W trybie kolorowym wycinki biorą kolory palety na zmianę z jej tłami.
var grays = []string{"#3b4754", "#c9d1da", "#7b8794", "#eef2f7", "#55657a", "#dde3ea", "#66727f", "#f6f8fa"}

// dark mówi, czy na tle o podanym kolorze czytelny jest tekst jasny.
func dark(color string) bool {
	if len(color) != 7 {
		return false
	}
	v, err := strconv.ParseUint(color[1:], 16, 32)
	if err != nil {
		return false
	}
	r, g, b := float64(v>>16&255), float64(v>>8&255), float64(v&255)
	return 0.299*r+0.587*g+0.114*b < 140
}

// Percent zapisuje udział wycinka: liczba całkowita, a poniżej 10% z jednym miejscem po przecinku.
func Percent(share float64) string {
	p := share * 100
	if p >= 10 || p == math.Trunc(p) {
		return strconv.Itoa(int(math.Round(p))) + "%"
	}
	return strconv.FormatFloat(math.Round(p*10)/10, 'f', 1, 64) + "%"
}

// Layout rozmieszcza wykres: koło z wycinkami w kolejności ze źródła, od godziny dwunastej zgodnie
// z ruchem wskazówek zegara, udział procentowy na wycinku, obok legenda.
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
	colors := grays
	if o.Colors {
		colors = nil
		for i := range scene.Palette {
			colors = append(colors, scene.Distinct(i).Stroke)
		}
		for i := range scene.Palette {
			colors = append(colors, scene.Distinct(i).Fill)
		}
	}

	// tone zwraca wypełnienie wycinka; ostatni wycinek nie może dostać tego samego co pierwszy,
	// bo na kole z nim sąsiaduje.
	tone := func(i int) string {
		if i > 0 && i == len(c.Slices)-1 && i%len(colors) == 0 {
			return colors[2]
		}
		return colors[i%len(colors)]
	}

	total := 0.0
	for _, s := range c.Slices {
		total += s.Value
	}
	legend := make([]string, len(c.Slices))
	legendW := 0.0
	for i, s := range c.Slices {
		legend[i] = s.Label
		if c.ShowData {
			legend[i] += " [" + s.Text + "]"
		}
		legendW = math.Max(legendW, width(legend[i]))
	}
	radius, swatch, rowH := u(110), u(14), lh+u(8)
	legendH := float64(len(legend))*rowH - u(8)
	bodyH := math.Max(2*radius, legendH)
	top := 0.0
	bodyW := 2*radius + u(36) + swatch + u(8) + legendW
	if c.Title != "" {
		text(c.Title, math.Max(bodyW, width(c.Title))/2, lh/2, "middle", "title", colorText)
		top = lh + u(14)
	}
	cx, cy := radius, top+bodyH/2

	// point zwraca punkt na okręgu o promieniu r dla kąta liczonego od godziny dwunastej.
	point := func(angle, r float64) scene.Point {
		return scene.Point{X: cx + r*math.Sin(angle), Y: cy - r*math.Cos(angle)}
	}
	if o.Shadows {
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerFrame, Role: "shadow", X: cx + u(3), Y: cy + u(3), R: radius, Fill: "#000000", FillOpacity: 0.1})
	}
	angle := 0.0
	for i, s := range c.Slices {
		share := s.Value / total
		sweep := share * 2 * math.Pi
		fill := tone(i)
		if share > 0 {
			if share >= 1-1e-9 {
				sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "pie-slice", X: cx, Y: cy, R: radius, Fill: fill, Stroke: colorEdge, StrokeWidth: 1})
			} else {
				// Łuk z kawałków najwyżej ćwierćobrotowych; każdy to jedna krzywa Béziera.
				segments := []scene.Segment{{Cmd: 'M', To: scene.Point{X: cx, Y: cy}}, {Cmd: 'L', To: point(angle, radius)}}
				pieces := int(math.Ceil(sweep / (math.Pi / 2)))
				step := sweep / float64(pieces)
				k := 4.0 / 3 * math.Tan(step/4) * radius
				for p := 0; p < pieces; p++ {
					a0, a1 := angle+float64(p)*step, angle+float64(p+1)*step
					from, to := point(a0, radius), point(a1, radius)
					segments = append(segments, scene.Segment{Cmd: 'C',
						C1: scene.Point{X: from.X + k*math.Cos(a0), Y: from.Y + k*math.Sin(a0)},
						C2: scene.Point{X: to.X - k*math.Cos(a1), Y: to.Y - k*math.Sin(a1)}, To: to})
				}
				segments = append(segments, scene.Segment{Cmd: 'L', To: scene.Point{X: cx, Y: cy}})
				sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerShape, Role: "pie-slice", Segments: segments, Fill: fill, Stroke: colorLight, StrokeWidth: 1.5})
			}
			// Udział na wycinku, jeśli wycinek jest dość duży, żeby go pomieścić.
			if label := Percent(share); share >= 0.04 {
				at := point(angle+sweep/2, radius*0.66)
				if share >= 1-1e-9 {
					at = scene.Point{X: cx, Y: cy}
				}
				ink := colorText
				if dark(fill) {
					ink = colorLight
				}
				text(label, at.X, at.Y, "middle", "pie-percent", ink)
			}
		}
		angle += sweep
	}
	if len(c.Slices) > 1 || total > 0 {
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "pie-outline", X: cx, Y: cy, R: radius, Stroke: colorEdge, StrokeWidth: 1.5})
	}

	x, y := 2*radius+u(36), top+(bodyH-legendH)/2
	for i, s := range legend {
		mid := y + float64(i)*rowH + lh/2
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "pie-swatch", X: x, Y: mid - swatch/2, W: swatch, H: swatch,
			Fill: tone(i), Stroke: colorEdge, StrokeWidth: 1})
		text(s, x+swatch+u(8), mid, "start", "pie-legend", colorText)
	}
	return sc
}
