// Package png writes a diagram as a PNG image. The drawing is made from the same scene as SVG and
// PDF, by the package's own anti-aliasing rasterizer; text is drawn with the glyph outlines of the
// font with which the drawing was measured. The package is optional and has no dependencies outside
// the standard library.
package png

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	stdpng "image/png"
	"math"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/bridge"
	"github.com/polestorm/gomermaidgo/internal/csscolor"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/glyf"
	"github.com/polestorm/gomermaidgo/internal/raster"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/mermaid"
)

// DefaultScale is the number of image pixels per drawing pixel in Render: 2, that is 192 dots per
// inch, so that the image is sharp on high-density screens and in print.
const DefaultScale = 2

// MaxPixels limits the area of the image; a larger drawing is the error ErrTooLarge.
const MaxPixels = 64 << 20

// ErrTooLarge is returned by rendering when the image would exceed MaxPixels.
var ErrTooLarge = errors.New("png: the image would be too large; use a smaller scale")

// ErrOutlines is returned when rendering with a font that has no TrueType outlines (e.g. OpenType with CFF).
var ErrOutlines = glyf.ErrOutlines

// Render converts diagram source to a PNG image at the scale DefaultScale, with a transparent background
// unless WithBackground is given. It accepts the same options as mermaid.Render. The font must be known
// from a file: the bundled one or the one given in WithFont; otherwise it returns mermaid.ErrFontRequired.
func Render(source string, opts ...mermaid.Option) ([]byte, error) {
	return RenderScale(source, DefaultScale, opts...)
}

// RenderScale works like Render, at the given scale: 1 gives an image of the SVG drawing's pixel size.
func RenderScale(source string, scale float64, opts ...mermaid.Option) ([]byte, error) {
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
	c        *raster.Canvas
	scale    float64
	dx, dy   float64 // przesunięcie początku rysunku
	faces    map[string]*face
	fallback *face // krój zapasowy; nil, gdy go nie ma
}

func (p *painter) face(data []byte) (*face, error) {
	if f, ok := p.faces[string(data)]; ok {
		return f, nil
	}
	if len(data) == 0 {
		return nil, mermaid.ErrFontRequired
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
	// Znak, którego krój nie ma, bierze krój zapasowy, jeśli ten go ma; selektory wariantu
	// i łącznik zerowej szerokości są pomijane.
	type glyph struct {
		f   *face
		gid int
	}
	var glyphs []glyph
	total := 0.0
	for _, c := range it.Text {
		use := f
		if !f.cmap.Has(c) {
			if font.Ignorable(c) {
				continue
			}
			if p.fallback != nil && p.fallback.cmap.Has(c) {
				use = p.fallback
			}
		}
		gid := int(use.cmap.GlyphID(c))
		glyphs = append(glyphs, glyph{use, gid})
		total += float64(use.outlines.Advance(gid)) / float64(use.outlines.UnitsPerEm) * size
	}
	x := it.X
	switch it.Anchor {
	case "middle":
		x -= total / 2
	case "end":
		x -= total
	}
	var polygons [][]raster.Point
	for _, gl := range glyphs {
		f, gid := gl.f, gl.gid
		k := size / float64(f.outlines.UnitsPerEm)
		contours, seen := f.cache[gid]
		if !seen {
			contours = f.outlines.Contours(gid)
			f.cache[gid] = contours
		}
		for _, contour := range contours {
			poly := make([]raster.Point, len(contour))
			for i, q := range contour {
				gx, gy := x+q.X*k, it.Y-q.Y*k
				if it.Vertical {
					gx, gy = it.X+(gy-it.Y), it.Y-(gx-it.X)
				}
				poly[i] = p.at(gx, gy)
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
	if len(data.Fallback) > 0 {
		if f, err := p.face(data.Fallback); err == nil {
			p.fallback = f
		}
	}
	if sc.Background != "" {
		p.item(scene.Item{Kind: scene.Rect, X: x0, Y: y0, W: width, H: height, Fill: sc.Background}, sc.FontSize, fonts)
	}
	items, holes := sc.Ordered()
	for _, it := range items {
		if !scene.Cut(it, holes) {
			p.item(it, sc.FontSize, fonts)
			continue
		}
		// Linia pod podpisami przy przezroczystym tle: piksele pól podpisów wracają do stanu sprzed niej.
		img := p.c.Image
		type patch struct {
			r   image.Rectangle
			pix []uint8
		}
		var saved []patch
		for _, o := range holes {
			r := image.Rect(int(math.Round((o.X0-x0)*scale)), int(math.Round((o.Y0-y0)*scale)),
				int(math.Round((o.X1-x0)*scale)), int(math.Round((o.Y1-y0)*scale))).Intersect(img.Bounds())
			if r.Empty() {
				continue
			}
			pt := patch{r: r}
			for y := r.Min.Y; y < r.Max.Y; y++ {
				at := img.PixOffset(r.Min.X, y)
				pt.pix = append(pt.pix, img.Pix[at:at+4*r.Dx()]...)
			}
			saved = append(saved, pt)
		}
		p.item(it, sc.FontSize, fonts)
		for _, pt := range saved {
			for y, k := pt.r.Min.Y, 0; y < pt.r.Max.Y; y++ {
				at := img.PixOffset(pt.r.Min.X, y)
				k += copy(img.Pix[at:at+4*pt.r.Dx()], pt.pix[k:k+4*pt.r.Dx()])
			}
		}
	}
	return p.c.Image, nil
}
