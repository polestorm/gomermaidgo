// Package png zapisuje diagram jako obraz PNG. Rysunek powstaje z tej samej sceny co SVG i PDF,
// własnym rasteryzatorem z wygładzaniem krawędzi; tekst jest rysowany konturami glifów kroju,
// którym rysunek został zmierzony. Pakiet jest opcjonalny i nie ma zależności poza biblioteką
// standardową.
package png

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	stdpng "image/png"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo"
	"github.com/polestorm/gomermaidgo/internal/bridge"
	"github.com/polestorm/gomermaidgo/internal/csscolor"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/glyf"
	"github.com/polestorm/gomermaidgo/internal/raster"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// DefaultScale to liczba pikseli obrazu na piksel rysunku w Render: 2, czyli 192 punkty na cal,
// żeby obraz był ostry na ekranach o dużej gęstości i w druku.
const DefaultScale = 2

// MaxPixels ogranicza powierzchnię obrazu; większy rysunek jest błędem ErrTooLarge.
const MaxPixels = 64 << 20

// ErrTooLarge zwraca rysowanie, gdy obraz przekroczyłby MaxPixels.
var ErrTooLarge = errors.New("png: the image would be too large; use a smaller scale")

// ErrOutlines zwraca rysowanie krojem bez konturów TrueType (np. OpenType z konturami CFF).
var ErrOutlines = glyf.ErrOutlines

// Render zamienia źródło diagramu na obraz PNG w skali DefaultScale, z przezroczystym tłem, jeśli
// nie podano WithBackground. Przyjmuje te same opcje co gomermaidgo.Render. Krój musi być znany
// z pliku: dołączony albo podany w WithFont; inaczej zwraca gomermaidgo.ErrFontRequired.
func Render(source string, opts ...gomermaidgo.Option) ([]byte, error) {
	return RenderScale(source, DefaultScale, opts...)
}

// RenderScale działa jak Render, w podanej skali: 1 daje obraz o wymiarach rysunku SVG w pikselach.
func RenderScale(source string, scale float64, opts ...gomermaidgo.Option) ([]byte, error) {
	if !(scale > 0) || math.IsInf(scale, 0) {
		return nil, fmt.Errorf("png: scale %v is not a positive number", scale)
	}
	options := make([]any, len(opts))
	for i, o := range opts {
		options[i] = o
	}
	sc, fonts, err := bridge.Build(source, options)
	if err != nil {
		return nil, err
	}
	img, err := draw(sc, fonts, scale)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := stdpng.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// face to jeden krój: kontury i odwzorowanie znaków na glify.
type face struct {
	outlines *glyf.Font
	cmap     *font.Face
	cache    map[int][][]glyf.Point
}

// painter rysuje scenę na obrazie.
type painter struct {
	c      *raster.Canvas
	scale  float64
	dx, dy float64 // przesunięcie początku rysunku
	faces  map[string]*face
}

func (p *painter) face(data []byte) (*face, error) {
	if f, ok := p.faces[string(data)]; ok {
		return f, nil
	}
	if len(data) == 0 {
		return nil, gomermaidgo.ErrFontRequired
	}
	outlines, err := glyf.Parse(data)
	if err != nil {
		if errors.Is(err, glyf.ErrOutlines) {
			return nil, err
		}
		return nil, fmt.Errorf("png: %w", err)
	}
	cmap, err := font.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("png: %w", err)
	}
	f := &face{outlines: outlines, cmap: cmap, cache: map[int][][]glyf.Point{}}
	p.faces[string(data)] = f
	return f, nil
}

// at przelicza punkt rysunku na piksele obrazu.
func (p *painter) at(x, y float64) raster.Point {
	return raster.Point{X: (x - p.dx) * p.scale, Y: (y - p.dy) * p.scale}
}

const kappa = 0.5523

// ellipse zwraca obwód elipsy jako łamaną zamkniętą.
func (p *painter) ellipse(cx, cy, rx, ry float64) []raster.Point {
	pts := []raster.Point{p.at(cx+rx, cy)}
	pts = raster.Cubic(pts, p.at(cx+rx, cy+ry*kappa), p.at(cx+rx*kappa, cy+ry), p.at(cx, cy+ry))
	pts = raster.Cubic(pts, p.at(cx-rx*kappa, cy+ry), p.at(cx-rx, cy+ry*kappa), p.at(cx-rx, cy))
	pts = raster.Cubic(pts, p.at(cx-rx, cy-ry*kappa), p.at(cx-rx*kappa, cy-ry), p.at(cx, cy-ry))
	pts = raster.Cubic(pts, p.at(cx+rx*kappa, cy-ry), p.at(cx+rx, cy-ry*kappa), p.at(cx+rx, cy))
	return pts[:len(pts)-1]
}

// rect zwraca obwód prostokąta, także z zaokrąglonymi narożnikami.
func (p *painter) rect(x, y, width, height, r float64) []raster.Point {
	r = math.Min(r, math.Min(width, height)/2)
	x1, y1 := x+width, y+height
	if r <= 0 {
		return []raster.Point{p.at(x, y), p.at(x1, y), p.at(x1, y1), p.at(x, y1)}
	}
	k := r * (1 - kappa)
	pts := []raster.Point{p.at(x+r, y), p.at(x1-r, y)}
	pts = raster.Cubic(pts, p.at(x1-k, y), p.at(x1, y+k), p.at(x1, y+r))
	pts = append(pts, p.at(x1, y1-r))
	pts = raster.Cubic(pts, p.at(x1, y1-k), p.at(x1-k, y1), p.at(x1-r, y1))
	pts = append(pts, p.at(x+r, y1))
	pts = raster.Cubic(pts, p.at(x+k, y1), p.at(x, y1-k), p.at(x, y1-r))
	pts = append(pts, p.at(x, y+r))
	pts = raster.Cubic(pts, p.at(x, y+k), p.at(x+k, y), p.at(x+r, y))
	return pts[:len(pts)-1]
}

// shape maluje kształt z łamanych: wypełnienie, a na nim obrys.
func (p *painter) shape(it scene.Item, lines [][]raster.Point, closed bool) {
	if r, g, b, ok := csscolor.Parse(it.Fill); ok && it.Kind != scene.Line {
		alpha := 1.0
		if it.FillOpacity > 0 && it.FillOpacity < 1 {
			alpha = it.FillOpacity
		}
		p.c.Fill(lines, r, g, b, alpha)
	}
	r, g, b, ok := csscolor.Parse(it.Stroke)
	if !ok {
		return
	}
	width := it.StrokeWidth
	if width == 0 {
		width = 1
	}
	var pattern []float64
	for _, d := range strings.FieldsFunc(it.Dash, func(c rune) bool { return c == ' ' || c == ',' }) {
		v, err := strconv.ParseFloat(d, 64)
		if err != nil || v < 0 {
			pattern = nil
			break
		}
		pattern = append(pattern, v*p.scale)
	}
	var outline [][]raster.Point
	for _, line := range lines {
		if len(pattern) == 0 {
			outline = append(outline, raster.Stroke(line, width*p.scale, closed)...)
			continue
		}
		for _, piece := range raster.Dash(line, pattern, closed) {
			outline = append(outline, raster.Stroke(piece, width*p.scale, false)...)
		}
	}
	p.c.Fill(outline, r, g, b, 1)
}

// text rysuje linię tekstu konturami glifów, od położenia wynikającego z zakotwiczenia.
func (p *painter) text(it scene.Item, size float64, f *face) {
	r, g, b, ok := csscolor.Parse(it.Fill)
	if !ok || it.Text == "" {
		return
	}
	em := float64(f.outlines.UnitsPerEm)
	var gids []int
	units := 0
	for _, c := range it.Text {
		gid := int(f.cmap.GlyphID(c))
		gids = append(gids, gid)
		units += f.outlines.Advance(gid)
	}
	x := it.X
	switch it.Anchor {
	case "middle":
		x -= float64(units) / em * size / 2
	case "end":
		x -= float64(units) / em * size
	}
	k := size / em
	var polygons [][]raster.Point
	for _, gid := range gids {
		contours, seen := f.cache[gid]
		if !seen {
			contours = f.outlines.Contours(gid)
			f.cache[gid] = contours
		}
		for _, contour := range contours {
			poly := make([]raster.Point, len(contour))
			for i, q := range contour {
				poly[i] = p.at(x+q.X*k, it.Y-q.Y*k)
			}
			polygons = append(polygons, poly)
		}
		x += float64(f.outlines.Advance(gid)) * k
	}
	p.c.Fill(polygons, r, g, b, 1)
}

func (p *painter) item(it scene.Item, fontSize float64, fonts [3]*face) {
	switch it.Kind {
	case scene.Text:
		size := fontSize
		if it.FontSize > 0 {
			size = it.FontSize
		}
		f := fonts[0]
		if it.Bold {
			f = fonts[1] // pogrubienie z kursywą jest rysowane odmianą pogrubioną, tak jak mierzone
		} else if it.Italic {
			f = fonts[2]
		}
		p.text(it, size, f)
	case scene.Rect:
		p.shape(it, [][]raster.Point{p.rect(it.X, it.Y, it.W, it.H, it.RX)}, true)
	case scene.Circle:
		p.shape(it, [][]raster.Point{p.ellipse(it.X, it.Y, it.R, it.R)}, true)
	case scene.Ellipse:
		p.shape(it, [][]raster.Point{p.ellipse(it.X, it.Y, it.W/2, it.H/2)}, true)
	case scene.Line:
		p.shape(it, [][]raster.Point{{p.at(it.X, it.Y), p.at(it.X2, it.Y2)}}, false)
	case scene.Polyline, scene.Polygon:
		if len(it.Points) == 0 {
			return
		}
		pts := make([]raster.Point, len(it.Points))
		for i, q := range it.Points {
			pts[i] = p.at(q.X, q.Y)
		}
		p.shape(it, [][]raster.Point{pts}, it.Kind == scene.Polygon)
	case scene.Path:
		// Ścieżka może mieć kilka części; każda zaczyna się od „M”.
		var lines [][]raster.Point
		var cur []raster.Point
		for _, sg := range it.Segments {
			switch {
			case sg.Cmd == 'M' || len(cur) == 0:
				if len(cur) > 1 {
					lines = append(lines, cur)
				}
				cur = []raster.Point{p.at(sg.To.X, sg.To.Y)}
			case sg.Cmd == 'L':
				cur = append(cur, p.at(sg.To.X, sg.To.Y))
			case sg.Cmd == 'C':
				cur = raster.Cubic(cur, p.at(sg.C1.X, sg.C1.Y), p.at(sg.C2.X, sg.C2.Y), p.at(sg.To.X, sg.To.Y))
			}
		}
		if len(cur) > 1 {
			lines = append(lines, cur)
		}
		p.shape(it, lines, false)
	}
}

// draw rysuje scenę na obrazie o wymiarach rysunku pomnożonych przez skalę.
func draw(sc *scene.Scene, data bridge.Fonts, scale float64) (*image.RGBA, error) {
	bounds := sc.Bounds()
	x0, y0 := bounds.X0-sc.Margin, bounds.Y0-sc.Margin
	width, height := bounds.Width()+2*sc.Margin, bounds.Height()+2*sc.Margin
	w, h := int(math.Ceil(width*scale)), int(math.Ceil(height*scale))
	if w < 1 || h < 1 || float64(w)*float64(h) > MaxPixels {
		return nil, ErrTooLarge
	}
	p := &painter{c: raster.New(w, h), scale: scale, dx: x0, dy: y0, faces: map[string]*face{}}
	var fonts [3]*face
	for i, d := range [][]byte{data.Regular, data.Bold, data.Italic} {
		f, err := p.face(d)
		if err != nil {
			return nil, err
		}
		fonts[i] = f
	}
	if sc.Background != "" {
		p.item(scene.Item{Kind: scene.Rect, X: x0, Y: y0, W: width, H: height, Fill: sc.Background}, sc.FontSize, fonts)
	}
	items := make([]scene.Item, len(sc.Items))
	copy(items, sc.Items)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Layer < items[j].Layer })
	for _, it := range items {
		p.item(it, sc.FontSize, fonts)
	}
	return p.c.Image, nil
}
