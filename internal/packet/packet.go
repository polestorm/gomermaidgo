// Package packet rozbiera i rysuje diagram pakietu (packet): pola bitowe w wierszach po 32 bity.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package packet

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramu pakietu.
const (
	CodeHeader    = "packet-header"
	CodeStatement = "packet-statement"
	CodeRange     = "packet-range"
	CodeEmpty     = "packet-empty"
	maxBits       = 2048
	rowBits       = 32
)

// Field to pole pakietu: zakres bitów (włącznie) i nazwa.
type Field struct {
	From, To int
	Label    string
	Line     int
}

// Diagram to diagram pakietu.
type Diagram struct {
	Title    string
	Fields   []Field
	AccTitle string
	AccDescr string
}

var field = regexp.MustCompile(`^(\+)?(\d+)(?:\s*-\s*(\d+))?\s*:\s*"([^"]*)"$`)

// Parse rozbiera źródło diagramu pakietu. Pola muszą następować po sobie bez przerw, od bitu 0.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	seenHeader := false
	next := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "packet-beta" && s != "packet" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"packet-beta\", found %q", s)
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
		m := field.FindStringSubmatch(s)
		if m == nil {
			return nil, syntax.Errorf(line, CodeStatement, "expected a field written as 0-15: \"name\", found %q", s)
		}
		a, _ := strconv.Atoi(m[2])
		from, to := a, a
		switch {
		case m[1] == "+": // długość w bitach, od pierwszego wolnego bitu
			if a < 1 || m[3] != "" {
				return nil, syntax.Errorf(line, CodeRange, "a relative field needs a length of at least one bit")
			}
			from, to = next, next+a-1
		case m[3] != "":
			to, _ = strconv.Atoi(m[3])
		}
		if to < from {
			return nil, syntax.Errorf(line, CodeRange, "field ends at bit %d before it starts at bit %d", to, from)
		}
		if from != next {
			return nil, syntax.Errorf(line, CodeRange, "field starts at bit %d, expected bit %d: fields follow one another without gaps", from, next)
		}
		if to >= maxBits {
			return nil, syntax.Errorf(line, CodeRange, "packet is too long (limit %d bits)", maxBits)
		}
		d.Fields = append(d.Fields, Field{From: from, To: to, Label: strings.TrimSpace(m[4]), Line: line})
		next = to + 1
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"packet-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(d.Fields) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the packet needs at least one field")
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
	Colors     bool // pola w kolejnych tonach palety
	Shadows    bool // przyjmowane dla zgodności opcji; diagram nie ma cieni
}

const (
	colorText  = "#1b1f24"
	colorMuted = "#5b6570"
	colorFill  = "#eef2f7"
	colorLine  = "#55657a"
)

// Layout rozmieszcza pola w wierszach po 32 bity. Pole dłuższe niż reszta wiersza przechodzi do
// następnego; nad każdym kawałkiem stoją numery jego pierwszego i ostatniego bitu.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.7)
	title := d.AccTitle
	if title == "" {
		title = d.Title
	}
	sc := &scene.Scene{Title: title, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}
	bit, rowH, numH := u(19), u(36), small*1.25+u(2)
	y0 := 0.0
	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "title", X: bit * rowBits / 2, Y: lh/2 + 0.35*fs, W: o.Measure(d.Title, fs), H: lh,
			Text: d.Title, Anchor: "middle", Fill: colorText})
		y0 = lh + u(10)
	}
	number := func(v int, x, top float64, anchor string) {
		s := strconv.Itoa(v)
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "packet-bit", X: x, Y: top + small, W: o.Measure(s, small), H: small * 1.25,
			Text: s, Anchor: anchor, FontSize: small, Fill: colorMuted})
	}
	for i, f := range d.Fields {
		fill := colorFill
		if o.Colors {
			fill = scene.Distinct(i).Fill
		}
		for from := f.From; from <= f.To; {
			row := from / rowBits
			to := min(f.To, (row+1)*rowBits-1)
			x, w := float64(from%rowBits)*bit, float64(to-from+1)*bit
			top := y0 + float64(row)*(numH+rowH+u(10))
			if to > from {
				number(from, x+u(4), top, "start")
				number(to, x+w-u(4), top, "end")
			} else {
				number(from, x+w/2, top, "middle")
			}
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "packet-field", X: x, Y: top + numH, W: w, H: rowH, Fill: fill, Stroke: colorLine, StrokeWidth: 1.5})
			// Nazwa w rozmiarze, w którym mieści się w kawałku pola; gdy nie mieści się w żadnym,
			// zostaje w najmniejszym.
			size := fs
			for _, candidate := range []float64{fs, math.Round(fs * 0.8), math.Round(fs * 0.65)} {
				size = candidate
				if o.Measure(f.Label, size) <= w-u(4) {
					break
				}
			}
			switch tw := o.Measure(f.Label, size); {
			case f.Label == "":
			case tw > w-u(4) && tw <= rowH-u(6):
				// Pole węższe niż nazwa nawet najmniejszym pismem (zwykle jednobitowe): nazwa stoi
				// pionowo, czytana z dołu do góry, i nie wchodzi na sąsiednie pola.
				sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "packet-label", X: x + w/2 + 0.35*size, Y: top + numH + rowH/2,
					W: tw, H: size * 1.25, Text: f.Label, Anchor: "middle", Vertical: true, FontSize: size, Fill: colorText})
			default:
				sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "packet-label", X: x + w/2, Y: top + numH + rowH/2 + 0.35*size,
					W: tw, H: size * 1.25, Text: f.Label, Anchor: "middle", FontSize: size, Fill: colorText})
			}
			from = to + 1
		}
	}
	return sc
}
