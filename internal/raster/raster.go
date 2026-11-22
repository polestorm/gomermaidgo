// Package raster zamienia kształty wektorowe na piksele z wygładzaniem krawędzi. Każdy kształt to
// zbiór zamkniętych wielokątów; pokrycie piksela liczy się dokładnie, z pola wielokąta w jego
// obrębie (sumowanie pól ze znakiem wzdłuż linii obrazu), bez nadpróbkowania.
package raster

import (
	"image"
	"math"
)

// Point to punkt w pikselach obrazu.
type Point struct{ X, Y float64 }

// Canvas to obraz w budowie.
type Canvas struct {
	Image *image.RGBA
	w, h  int
	acc   []float64 // pola ze znakiem dla bieżącego kształtu, wiersz po wierszu
	// Zakres wierszy i kolumn dotkniętych przez bieżący kształt.
	x0, y0, x1, y1 int
}

// New tworzy przezroczysty obraz o podanych wymiarach.
func New(width, height int) *Canvas {
	return &Canvas{Image: image.NewRGBA(image.Rect(0, 0, width, height)), w: width, h: height,
		acc: make([]float64, (width+2)*height), x0: width, y0: height}
}

// line dopisuje do bufora pól odcinek krawędzi wielokąta.
func (c *Canvas) line(p0, p1 Point) {
	// Poza obrazem w poziomie krawędź przesuwa się na jego brzeg: pole wewnątrz obrazu się nie zmienia.
	limit := float64(c.w)
	p0.X, p1.X = math.Min(math.Max(p0.X, 0), limit), math.Min(math.Max(p1.X, 0), limit)
	if p0.Y == p1.Y {
		return
	}
	dir := 1.0
	if p0.Y > p1.Y {
		dir, p0, p1 = -1, p1, p0
	}
	slope := (p1.X - p0.X) / (p1.Y - p0.Y)
	x := p0.X
	if p0.Y < 0 {
		x -= p0.Y * slope
	}
	first := int(math.Max(math.Floor(p0.Y), 0))
	last := int(math.Min(math.Ceil(p1.Y), float64(c.h)))
	stride := c.w + 2
	for y := first; y < last; y++ {
		row := c.acc[y*stride : (y+1)*stride]
		dy := math.Min(float64(y+1), p1.Y) - math.Max(float64(y), p0.Y)
		next := x + slope*dy
		d := dy * dir
		xa, xb := math.Min(x, next), math.Max(x, next)
		fa := math.Floor(xa)
		ia, ib := int(fa), int(math.Ceil(xb))
		if ib <= ia+1 {
			// Krawędź w obrębie jednego piksela: pole dzieli się między niego i resztę wiersza.
			m := (x+next)/2 - fa
			row[ia] += d - d*m
			row[ia+1] += d * m
		} else {
			s := 1 / (xb - xa)
			fx := xa - fa
			a0 := 0.5 * s * (1 - fx) * (1 - fx)
			tail := xb - float64(ib) + 1
			am := 0.5 * s * tail * tail
			row[ia] += d * a0
			if ib == ia+2 {
				row[ia+1] += d * (1 - a0 - am)
			} else {
				a1 := s * (1.5 - fx)
				row[ia+1] += d * (a1 - a0)
				for i := ia + 2; i < ib-1; i++ {
					row[i] += d * s
				}
				a2 := a1 + float64(ib-ia-3)*s
				row[ib-1] += d * (1 - a2 - am)
			}
			row[ib] += d * am
		}
		c.x0, c.x1 = min(c.x0, ia), max(c.x1, ib+1)
		x = next
	}
	if first < last {
		c.y0, c.y1 = min(c.y0, first), max(c.y1, last)
	}
}

// Fill maluje kształt z wielokątów kolorem o składowych od 0 do 1. Wnętrze wyznacza reguła
// niezerowej liczby obiegów: wielokąty o tym samym kierunku się sumują, o przeciwnym wycinają otwór.
func (c *Canvas) Fill(polygons [][]Point, r, g, b, alpha float64) {
	for _, poly := range polygons {
		for i := range poly {
			c.line(poly[i], poly[(i+1)%len(poly)])
		}
	}
	stride := c.w + 2
	pr, pg, pb := r*alpha, g*alpha, b*alpha
	for y := c.y0; y < c.y1; y++ {
		row := c.acc[y*stride : (y+1)*stride]
		sum := 0.0
		for x := c.x0; x < min(c.x1+1, stride); x++ {
			sum += row[x]
			row[x] = 0
			cover := math.Min(math.Abs(sum), 1)
			if cover < 1.0/512 || x >= c.w {
				continue
			}
			// Składanie „nad” na składowych pomnożonych przez krycie.
			px := c.Image.Pix[y*c.Image.Stride+4*x:]
			keep := 1 - cover*alpha
			px[0] = uint8(math.Round(pr*cover*255 + float64(px[0])*keep))
			px[1] = uint8(math.Round(pg*cover*255 + float64(px[1])*keep))
			px[2] = uint8(math.Round(pb*cover*255 + float64(px[2])*keep))
			px[3] = uint8(math.Round(alpha*cover*255 + float64(px[3])*keep))
		}
	}
	c.x0, c.y0, c.x1, c.y1 = c.w, c.h, 0, 0
}

// area to pole wielokąta ze znakiem.
func area(p []Point) float64 {
	sum := 0.0
	for i := range p {
		q := p[(i+1)%len(p)]
		sum += p[i].X*q.Y - q.X*p[i].Y
	}
	return sum / 2
}

// oriented zwraca wielokąt o dodatnim polu, żeby części obrysu sumowały się, a nie znosiły.
func oriented(p []Point) []Point {
	if area(p) < 0 {
		for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
			p[i], p[j] = p[j], p[i]
		}
	}
	return p
}

// Stroke zamienia łamaną na wielokąty obrysu o podanej grubości: prostokąt na każdy odcinek
// i koło w każdym załamaniu. Końce łamanej otwartej są ścięte prosto.
func Stroke(points []Point, width float64, closed bool) [][]Point {
	var out [][]Point
	half := width / 2
	n := len(points)
	if n < 2 || half <= 0 {
		return nil
	}
	segments := n - 1
	if closed {
		segments = n
	}
	sides := max(8, int(math.Ceil(half*2.5)))
	joint := func(p Point) {
		circle := make([]Point, sides)
		for i := range circle {
			a := 2 * math.Pi * float64(i) / float64(sides)
			circle[i] = Point{p.X + half*math.Cos(a), p.Y + half*math.Sin(a)}
		}
		out = append(out, oriented(circle))
	}
	for i := 0; i < segments; i++ {
		a, b := points[i], points[(i+1)%n]
		dx, dy := b.X-a.X, b.Y-a.Y
		length := math.Hypot(dx, dy)
		if length == 0 {
			continue
		}
		nx, ny := -dy/length*half, dx/length*half
		out = append(out, oriented([]Point{{a.X + nx, a.Y + ny}, {b.X + nx, b.Y + ny}, {b.X - nx, b.Y - ny}, {a.X - nx, a.Y - ny}}))
		if i > 0 || closed {
			joint(a)
		}
	}
	return out
}

// Dash dzieli łamaną na kawałki według wzoru kresek i przerw.
func Dash(points []Point, pattern []float64, closed bool) [][]Point {
	total := 0.0
	for _, v := range pattern {
		total += v
	}
	if len(pattern) == 0 || total <= 0 {
		return [][]Point{points}
	}
	if closed && len(points) > 0 {
		points = append(append([]Point{}, points...), points[0])
	}
	var out [][]Point
	var piece []Point
	index, left, on := 0, pattern[0], true
	for i := 0; i+1 < len(points); i++ {
		a, b := points[i], points[i+1]
		length := math.Hypot(b.X-a.X, b.Y-a.Y)
		pos := 0.0
		if on && len(piece) == 0 {
			piece = append(piece, a)
		}
		for length-pos > left {
			pos += left
			t := pos / length
			p := Point{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
			if on {
				out = append(out, append(piece, p))
				piece = nil
			} else {
				piece = []Point{p}
			}
			on = !on
			index = (index + 1) % len(pattern)
			left = pattern[index]
			if left <= 0 {
				left = 1e-6
			}
		}
		left -= length - pos
		if on {
			piece = append(piece, b)
		}
	}
	if on && len(piece) > 1 {
		out = append(out, piece)
	}
	return out
}

// Cubic dopisuje do łamanej krzywą Béziera trzeciego stopnia od ostatniego punktu.
func Cubic(points []Point, c1, c2, to Point) []Point {
	from := points[len(points)-1]
	// Liczba odcinków z długości łamanej kontrolnej: około jednego na trzy piksele.
	length := math.Hypot(c1.X-from.X, c1.Y-from.Y) + math.Hypot(c2.X-c1.X, c2.Y-c1.Y) + math.Hypot(to.X-c2.X, to.Y-c2.Y)
	steps := min(max(int(math.Ceil(length/3)), 4), 200)
	for s := 1; s <= steps; s++ {
		t := float64(s) / float64(steps)
		u := 1 - t
		k0, k1, k2, k3 := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
		points = append(points, Point{k0*from.X + k1*c1.X + k2*c2.X + k3*to.X, k0*from.Y + k1*c1.Y + k2*c2.Y + k3*to.Y})
	}
	return points
}
