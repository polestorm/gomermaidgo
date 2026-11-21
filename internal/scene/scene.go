// Package scene to wspólny model rysunku dla wszystkich typów diagramów: lista prostych
// kształtów z rolami. Układ diagramu tworzy scenę, a ten pakiet zapisuje ją jako SVG.
// Testy niezmienników układu pracują na scenie, nie na tekście SVG.
package scene

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// Kind to rodzaj kształtu.
type Kind int

const (
	Rect Kind = iota
	Line
	Polyline
	Polygon
	Circle
	Text
	Ellipse
	Path
)

// Segment to odcinek ścieżki: przesunięcie (M), linia (L) albo krzywa Béziera trzeciego stopnia (C).
// Dla M i L liczy się tylko To; dla C także punkty kontrolne C1 i C2.
type Segment struct {
	Cmd    byte
	C1, C2 Point
	To     Point
}

// Warstwy rysowania, od spodu.
const (
	LayerBackground = iota
	LayerLifeline
	LayerLabelBackground
	LayerActivation
	LayerFrame
	LayerShape
	LayerLabel
)

// Point to punkt w pikselach.
type Point struct{ X, Y float64 }

// Item to jeden kształt. Znaczenie pól zależy od Kind:
// Rect: X, Y, W, H, RX; Line: X, Y, X2, Y2; Polyline i Polygon: Points;
// Circle: X, Y (środek), R; Ellipse: X, Y (środek), W, H (średnice); Path: Segments; Text: X, Y (punkt zaczepienia na linii bazowej), W, H (zmierzone pole).
type Item struct {
	Kind        Kind
	Layer       int
	Role        string // np. "participant", "message-label"; trafia do atrybutu class
	X, Y, W, H  float64
	X2, Y2      float64
	R, RX       float64
	Points      []Point
	Segments    []Segment
	Text        string
	Anchor      string // "start", "middle", "end"
	FontSize    float64
	Fill        string
	FillOpacity float64 // 0 oznacza „bez atrybutu”
	Stroke      string
	StrokeWidth float64
	Dash        string
}

// Scene to cały rysunek.
type Scene struct {
	Items      []Item
	Title      string
	FontFamily string
	FontSize   float64
	Margin     float64
	Background string // pusty = przezroczyste tło
}

// Tone to para kolorów jednego elementu: obrys i wypełnienie.
type Tone struct{ Stroke, Fill string }

// Palette to kolory dla trybu kolorowego: własny zestaw o podobnej jasności, czytelny z ciemnym tekstem.
var Palette = []Tone{
	{"#3b6fb6", "#e8f0fb"}, // niebieski
	{"#1f8a7a", "#e3f4f1"}, // morski
	{"#b7791f", "#fcf1dc"}, // bursztynowy
	{"#7a56b3", "#f0eafa"}, // fioletowy
	{"#b8456b", "#fbe9ef"}, // malinowy
	{"#4d8a2f", "#ebf5e4"}, // zielony
}

// AddShadows dopisuje cień pod każdym kształtem dodanym od pozycji from, którego rola jest na
// liście: przesuniętą kopię kształtu w półprzezroczystej czerni. To zwykły kształt, nie filtr,
// więc w PDF zostaje wektorem.
func (s *Scene) AddShadows(from int, offset float64, roles ...string) {
	for _, it := range s.Items[from:] {
		match := false
		for _, r := range roles {
			match = match || it.Role == r
		}
		if !match || it.Fill == "" || it.Kind == Text || it.Kind == Line || it.Kind == Path || it.Kind == Polyline {
			continue
		}
		sh := it
		sh.Role, sh.Layer, sh.Fill, sh.FillOpacity, sh.Stroke = "shadow", LayerFrame, "#000000", 0.1, ""
		sh.X, sh.Y = it.X+offset, it.Y+offset
		if len(it.Points) > 0 {
			sh.Points = make([]Point, len(it.Points))
			for i, p := range it.Points {
				sh.Points[i] = Point{X: p.X + offset, Y: p.Y + offset}
			}
		}
		s.Items = append(s.Items, sh)
	}
}

// Add dopisuje kształt.
func (s *Scene) Add(it Item) { s.Items = append(s.Items, it) }

// Box to prostokąt otaczający.
type Box struct{ X0, Y0, X1, Y1 float64 }

// Width i Height zwracają wymiary pola.
func (b Box) Width() float64  { return b.X1 - b.X0 }
func (b Box) Height() float64 { return b.Y1 - b.Y0 }

// Bounds zwraca pole zajmowane przez kształt (bez grubości linii).
func (it Item) Bounds() Box {
	switch it.Kind {
	case Rect:
		return Box{it.X, it.Y, it.X + it.W, it.Y + it.H}
	case Line:
		return Box{math.Min(it.X, it.X2), math.Min(it.Y, it.Y2), math.Max(it.X, it.X2), math.Max(it.Y, it.Y2)}
	case Circle:
		return Box{it.X - it.R, it.Y - it.R, it.X + it.R, it.Y + it.R}
	case Ellipse:
		return Box{it.X - it.W/2, it.Y - it.H/2, it.X + it.W/2, it.Y + it.H/2}
	case Path:
		// Punkty kontrolne otaczają krzywą, więc pole jest co najwyżej nieco za duże.
		b := Box{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		grow := func(p Point) {
			b.X0, b.Y0 = math.Min(b.X0, p.X), math.Min(b.Y0, p.Y)
			b.X1, b.Y1 = math.Max(b.X1, p.X), math.Max(b.Y1, p.Y)
		}
		for _, sg := range it.Segments {
			if sg.Cmd == 'C' {
				grow(sg.C1)
				grow(sg.C2)
			}
			grow(sg.To)
		}
		return b
	case Text:
		x0 := it.X
		switch it.Anchor {
		case "middle":
			x0 = it.X - it.W/2
		case "end":
			x0 = it.X - it.W
		}
		// Linia bazowa leży ok. 0,8 wysokości pisma poniżej górnej krawędzi pola.
		return Box{x0, it.Y - 0.8*it.H, x0 + it.W, it.Y + 0.2*it.H}
	default:
		b := Box{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		for _, p := range it.Points {
			b.X0, b.Y0 = math.Min(b.X0, p.X), math.Min(b.Y0, p.Y)
			b.X1, b.Y1 = math.Max(b.X1, p.X), math.Max(b.Y1, p.Y)
		}
		return b
	}
}

// Bounds zwraca pole całej sceny.
func (s *Scene) Bounds() Box {
	b := Box{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, it := range s.Items {
		ib := it.Bounds()
		b.X0, b.Y0 = math.Min(b.X0, ib.X0), math.Min(b.Y0, ib.Y0)
		b.X1, b.Y1 = math.Max(b.X1, ib.X1), math.Max(b.Y1, ib.Y1)
	}
	if len(s.Items) == 0 {
		return Box{}
	}
	return b
}

// num zapisuje liczbę z dokładnością do dwóch miejsc, bez zbędnych zer; zawsze tak samo.
func num(v float64) string {
	v = math.Round(v*100) / 100
	if v == 0 { // także -0
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

// Escape zabezpiecza tekst do wstawienia w treść i atrybuty XML.
func Escape(s string) string { return escaper.Replace(s) }

func attr(b *strings.Builder, name, value string) {
	if value == "" {
		return
	}
	b.WriteString(" ")
	b.WriteString(name)
	b.WriteString(`="`)
	b.WriteString(Escape(value))
	b.WriteString(`"`)
}

func (it Item) paint(b *strings.Builder) {
	fill := it.Fill
	if fill == "" {
		fill = "none"
	}
	attr(b, "fill", fill)
	if it.FillOpacity > 0 && it.FillOpacity < 1 {
		attr(b, "fill-opacity", num(it.FillOpacity))
	}
	if it.Stroke != "" {
		attr(b, "stroke", it.Stroke)
		w := it.StrokeWidth
		if w == 0 {
			w = 1
		}
		attr(b, "stroke-width", num(w))
		attr(b, "stroke-dasharray", it.Dash)
	}
}

func pathData(segments []Segment) string {
	var b strings.Builder
	pt := func(p Point) {
		b.WriteString(num(p.X))
		b.WriteByte(' ')
		b.WriteString(num(p.Y))
	}
	for i, sg := range segments {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(sg.Cmd)
		if sg.Cmd == 'C' {
			pt(sg.C1)
			b.WriteByte(' ')
			pt(sg.C2)
			b.WriteByte(' ')
		}
		pt(sg.To)
	}
	return b.String()
}

func points(ps []Point) string {
	var b strings.Builder
	for i, p := range ps {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(num(p.X))
		b.WriteByte(',')
		b.WriteString(num(p.Y))
	}
	return b.String()
}

// SVG zapisuje scenę jako samodzielny dokument SVG 1.1 złożony wyłącznie z elementów
// podstawowych; wygląd jest w atrybutach prezentacji, bez arkusza stylów.
func (s *Scene) SVG() []byte {
	bounds := s.Bounds()
	x0, y0 := bounds.X0-s.Margin, bounds.Y0-s.Margin
	w, h := bounds.Width()+2*s.Margin, bounds.Height()+2*s.Margin

	items := make([]Item, len(s.Items))
	copy(items, s.Items)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Layer < items[j].Layer })

	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" version="1.1"`)
	attr(&b, "width", num(w))
	attr(&b, "height", num(h))
	attr(&b, "viewBox", num(x0)+" "+num(y0)+" "+num(w)+" "+num(h))
	attr(&b, "role", "img")
	attr(&b, "font-family", s.FontFamily)
	attr(&b, "font-size", num(s.FontSize))
	b.WriteString(">\n")
	if s.Title != "" {
		b.WriteString("<title>")
		b.WriteString(Escape(s.Title))
		b.WriteString("</title>\n")
	}
	if s.Background != "" {
		b.WriteString("<rect")
		attr(&b, "x", num(x0))
		attr(&b, "y", num(y0))
		attr(&b, "width", num(w))
		attr(&b, "height", num(h))
		attr(&b, "fill", s.Background)
		b.WriteString("/>\n")
	}
	for _, it := range items {
		switch it.Kind {
		case Rect:
			b.WriteString("<rect")
			attr(&b, "class", it.Role)
			attr(&b, "x", num(it.X))
			attr(&b, "y", num(it.Y))
			attr(&b, "width", num(it.W))
			attr(&b, "height", num(it.H))
			if it.RX > 0 {
				attr(&b, "rx", num(it.RX))
			}
			it.paint(&b)
			b.WriteString("/>\n")
		case Line:
			b.WriteString("<line")
			attr(&b, "class", it.Role)
			attr(&b, "x1", num(it.X))
			attr(&b, "y1", num(it.Y))
			attr(&b, "x2", num(it.X2))
			attr(&b, "y2", num(it.Y2))
			it.paint(&b)
			b.WriteString("/>\n")
		case Polyline, Polygon:
			name := "polyline"
			if it.Kind == Polygon {
				name = "polygon"
			}
			b.WriteString("<" + name)
			attr(&b, "class", it.Role)
			attr(&b, "points", points(it.Points))
			it.paint(&b)
			b.WriteString("/>\n")
		case Circle:
			b.WriteString("<circle")
			attr(&b, "class", it.Role)
			attr(&b, "cx", num(it.X))
			attr(&b, "cy", num(it.Y))
			attr(&b, "r", num(it.R))
			it.paint(&b)
			b.WriteString("/>\n")
		case Ellipse:
			b.WriteString("<ellipse")
			attr(&b, "class", it.Role)
			attr(&b, "cx", num(it.X))
			attr(&b, "cy", num(it.Y))
			attr(&b, "rx", num(it.W/2))
			attr(&b, "ry", num(it.H/2))
			it.paint(&b)
			b.WriteString("/>\n")
		case Path:
			b.WriteString("<path")
			attr(&b, "class", it.Role)
			attr(&b, "d", pathData(it.Segments))
			it.paint(&b)
			b.WriteString("/>\n")
		case Text:
			b.WriteString("<text")
			attr(&b, "class", it.Role)
			attr(&b, "x", num(it.X))
			attr(&b, "y", num(it.Y))
			if it.Anchor != "" && it.Anchor != "start" {
				attr(&b, "text-anchor", it.Anchor)
			}
			if it.FontSize > 0 && it.FontSize != s.FontSize {
				attr(&b, "font-size", num(it.FontSize))
			}
			attr(&b, "fill", it.Fill)
			b.WriteString(">")
			b.WriteString(Escape(it.Text))
			b.WriteString("</text>\n")
		}
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}
