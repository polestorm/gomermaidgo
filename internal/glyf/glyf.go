// Package glyf czyta kontury glifów z kroju TrueType (tabela glyf) jako wielokąty, dla wyjść
// rastrowych. Źródłem jest specyfikacja OpenType: tabele head, maxp, hhea, hmtx, loca i glyf.
package glyf

import (
	"encoding/binary"
	"errors"
)

// ErrOutlines zwraca Parse dla kroju bez konturów TrueType, np. z konturami CFF.
var ErrOutlines = errors.New("the font has no TrueType outlines (glyf table); fonts with CFF outlines are not supported")

// Point to punkt konturu w jednostkach kroju, z osią y w górę.
type Point struct{ X, Y float64 }

// Font to krój odczytany w zakresie potrzebnym do rysowania konturów.
type Font struct {
	data        []byte
	loca, glyf  [2]int
	hmtx        int
	UnitsPerEm  int
	numGlyphs   int
	numHMetrics int
	longLoca    bool
}

func u16(b []byte, off int) int { return int(binary.BigEndian.Uint16(b[off:])) }
func s16(b []byte, off int) int { return int(int16(binary.BigEndian.Uint16(b[off:]))) }
func u32(b []byte, off int) int { return int(binary.BigEndian.Uint32(b[off:])) }

// Parse czyta katalog tabel i wielkości potrzebne do odnajdywania glifów.
func Parse(data []byte) (*Font, error) {
	if len(data) < 12 {
		return nil, errors.New("font file too short")
	}
	tables := map[string][2]int{}
	for i, n := 0, u16(data, 4); i < n; i++ {
		rec := 12 + 16*i
		if rec+16 > len(data) {
			return nil, errors.New("font file too short")
		}
		off, length := u32(data, rec+8), u32(data, rec+12)
		if off < 0 || length < 0 || off+length > len(data) {
			return nil, errors.New("font table lies outside the file")
		}
		tables[string(data[rec:rec+4])] = [2]int{off, length}
	}
	if _, ok := tables["glyf"]; !ok {
		return nil, ErrOutlines
	}
	for name, least := range map[string]int{"head": 54, "hhea": 36, "maxp": 6, "hmtx": 4, "loca": 2} {
		if t, ok := tables[name]; !ok || t[1] < least {
			return nil, errors.New("font table " + name + " is missing or too short")
		}
	}
	f := &Font{data: data, loca: tables["loca"], glyf: tables["glyf"], hmtx: tables["hmtx"][0]}
	head := tables["head"][0]
	f.UnitsPerEm = u16(data, head+18)
	f.longLoca = u16(data, head+50) != 0
	f.numHMetrics = u16(data, tables["hhea"][0]+34)
	f.numGlyphs = u16(data, tables["maxp"][0]+4)
	width := 2
	if f.longLoca {
		width = 4
	}
	if f.UnitsPerEm == 0 || f.numHMetrics == 0 || tables["hmtx"][1] < 4*f.numHMetrics || f.loca[1] < width*(f.numGlyphs+1) {
		return nil, errors.New("font metrics or glyph index are too short")
	}
	return f, nil
}

// Advance zwraca szerokość glifu w jednostkach kroju.
func (f *Font) Advance(gid int) int {
	if gid < 0 {
		return 0
	}
	if gid >= f.numHMetrics {
		gid = f.numHMetrics - 1 // glify poza tabelą mają szerokość ostatniego wpisu
	}
	return u16(f.data, f.hmtx+4*gid)
}

// raw zwraca dane glifu z tabeli glyf; pusty wycinek dla glifu bez konturu.
func (f *Font) raw(gid int) []byte {
	if gid < 0 || gid >= f.numGlyphs {
		return nil
	}
	var a, b int
	if f.longLoca {
		a, b = u32(f.data, f.loca[0]+4*gid), u32(f.data, f.loca[0]+4*gid+4)
	} else {
		a, b = 2*u16(f.data, f.loca[0]+2*gid), 2*u16(f.data, f.loca[0]+2*gid+2)
	}
	if a < 0 || b <= a || b > f.glyf[1] {
		return nil
	}
	return f.data[f.glyf[0]+a : f.glyf[0]+b]
}

// Flagi punktów glifu prostego i składników glifu złożonego (nazwy ze specyfikacji).
const (
	onCurve      = 1
	xShort       = 2
	yShort       = 4
	repeat       = 8
	xSame        = 16
	ySame        = 32
	argsAreWords = 1
	argsAreXY    = 2
	haveScale    = 8
	more         = 32
	haveXYScale  = 64
	haveTwoByTwo = 128
	maxDepth     = 6
	curveSteps   = 6 // odcinków na łuk drugiego stopnia; litery mają na rysunku kilkanaście pikseli
)

// Contours zwraca kontury glifu jako zamknięte wielokąty: łuki drugiego stopnia są już zamienione
// na odcinki. Glif uszkodzony albo pusty daje brak konturów.
func (f *Font) Contours(gid int) [][]Point {
	return f.contours(gid, 0)
}

func (f *Font) contours(gid, depth int) [][]Point {
	g := f.raw(gid)
	if len(g) < 10 || depth > maxDepth {
		return nil
	}
	if s16(g, 0) < 0 {
		var out [][]Point
		f.components(g, func(child int, a, b, c, d, dx, dy float64) {
			for _, contour := range f.contours(child, depth+1) {
				moved := make([]Point, len(contour))
				for i, q := range contour {
					moved[i] = Point{a*q.X + c*q.Y + dx, b*q.X + d*q.Y + dy}
				}
				out = append(out, moved)
			}
		})
		return out
	}
	xs, ys, flags, ends := f.points(g)
	var out [][]Point
	start := 0
	for _, end := range ends {
		if end >= start && end < len(xs) {
			out = append(out, outline(xs[start:end+1], ys[start:end+1], flags[start:end+1]))
		}
		start = end + 1
	}
	return out
}

// points czyta punkty glifu prostego: współrzędne, flagi i końce konturów. Dane uszkodzone dają
// brak punktów.
func (f *Font) points(g []byte) (xs, ys []float64, flags []byte, ends []int) {
	n := s16(g, 0)
	p := 10
	if p+2*n+2 > len(g) || n == 0 {
		return nil, nil, nil, nil
	}
	ends = make([]int, n)
	for i := range ends {
		ends[i] = u16(g, p+2*i)
		if i > 0 && ends[i] < ends[i-1] {
			return nil, nil, nil, nil
		}
	}
	count := ends[n-1] + 1
	p += 2 * n
	p += 2 + u16(g, p) // instrukcje
	flags = make([]byte, 0, count)
	for len(flags) < count {
		if p >= len(g) {
			return nil, nil, nil, nil
		}
		fl := g[p]
		p++
		flags = append(flags, fl)
		if fl&repeat != 0 {
			if p >= len(g) {
				return nil, nil, nil, nil
			}
			for k := int(g[p]); k > 0 && len(flags) < count; k-- {
				flags = append(flags, fl)
			}
			p++
		}
	}
	coords := func(short, same byte) []float64 {
		out := make([]float64, count)
		v := 0
		for i, fl := range flags {
			switch {
			case fl&short != 0:
				if p >= len(g) {
					return nil
				}
				d := int(g[p])
				p++
				if fl&same == 0 {
					d = -d
				}
				v += d
			case fl&same == 0:
				if p+2 > len(g) {
					return nil
				}
				v += s16(g, p)
				p += 2
			}
			out[i] = float64(v)
		}
		return out
	}
	if xs = coords(xShort, xSame); xs == nil {
		return nil, nil, nil, nil
	}
	if ys = coords(yShort, ySame); ys == nil {
		return nil, nil, nil, nil
	}
	return xs, ys, flags, ends
}

// Segment to odcinek konturu: prosty (Ctrl równe To) albo łuk drugiego stopnia z punktem
// sterującym Ctrl. Kontur zaczyna się w punkcie Start i kończy w nim ostatnim odcinkiem.
type Segment struct {
	Ctrl, To Point
	Curve    bool
}

// Path to jeden zamknięty kontur glifu.
type Path struct {
	Start    Point
	Segments []Segment
}

// outlinePath zamienia punkty jednego konturu na odcinki. Dwa kolejne punkty poza krzywą mają
// między sobą domyślny punkt na krzywej, w połowie drogi.
func outlinePath(xs, ys []float64, flags []byte) Path {
	n := len(xs)
	at := func(i int) Point { return Point{xs[i%n], ys[i%n]} }
	on := func(i int) bool { return flags[i%n]&onCurve != 0 }
	mid := func(a, b Point) Point { return Point{(a.X + b.X) / 2, (a.Y + b.Y) / 2} }
	// Początek: pierwszy punkt na krzywej, a gdy takiego nie ma, środek między ostatnim a pierwszym.
	first := -1
	for i := 0; i < n; i++ {
		if on(i) {
			first = i
			break
		}
	}
	var path Path
	if first < 0 {
		path.Start = mid(at(n-1), at(0))
		first = -1 // pętla zaczyna od punktu 0, który jest punktem sterującym
	} else {
		path.Start = at(first)
	}
	for k := 1; k <= n; k++ {
		i := first + k
		p := at(i)
		if on(i) {
			path.Segments = append(path.Segments, Segment{Ctrl: p, To: p})
			continue
		}
		// Punkt sterujący: koniec łuku to następny punkt na krzywej albo środek do następnego sterującego.
		next := at(i + 1)
		end := next
		if !on(i + 1) {
			end = mid(p, next)
		} else {
			k++ // koniec łuku zużywa następny punkt
		}
		path.Segments = append(path.Segments, Segment{Ctrl: p, To: end, Curve: true})
	}
	return path
}

// outline spłaszcza kontur do wielokąta.
func outline(xs, ys []float64, flags []byte) []Point {
	path := outlinePath(xs, ys, flags)
	poly := []Point{path.Start}
	cur := path.Start
	for _, sg := range path.Segments {
		if !sg.Curve {
			poly = append(poly, sg.To)
			cur = sg.To
			continue
		}
		for s := 1; s <= curveSteps; s++ {
			t := float64(s) / curveSteps
			u := 1 - t
			poly = append(poly, Point{u*u*cur.X + 2*u*t*sg.Ctrl.X + t*t*sg.To.X, u*u*cur.Y + 2*u*t*sg.Ctrl.Y + t*t*sg.To.Y})
		}
		cur = sg.To
	}
	return poly
}

// components przechodzi składniki glifu złożonego: numer glifu, macierz (a b c d) i przesunięcie.
func (f *Font) components(g []byte, visit func(child int, a, b, c, d, dx, dy float64)) {
	p := 10
	for {
		if p+4 > len(g) {
			return
		}
		flags, gid := u16(g, p), u16(g, p+2)
		p += 4
		var dx, dy float64
		if flags&argsAreWords != 0 {
			if p+4 > len(g) {
				return
			}
			dx, dy = float64(s16(g, p)), float64(s16(g, p+2))
			p += 4
		} else {
			if p+2 > len(g) {
				return
			}
			dx, dy = float64(int8(g[p])), float64(int8(g[p+1]))
			p += 2
		}
		if flags&argsAreXY == 0 {
			dx, dy = 0, 0 // dopasowanie punktami: rzadkie, traktowane jak brak przesunięcia
		}
		a, b, c, d := 1.0, 0.0, 0.0, 1.0
		f2 := func(off int) float64 { return float64(s16(g, off)) / 16384 }
		switch {
		case flags&haveScale != 0:
			if p+2 > len(g) {
				return
			}
			a, d = f2(p), f2(p)
			p += 2
		case flags&haveXYScale != 0:
			if p+4 > len(g) {
				return
			}
			a, d = f2(p), f2(p+2)
			p += 4
		case flags&haveTwoByTwo != 0:
			if p+8 > len(g) {
				return
			}
			a, b, c, d = f2(p), f2(p+2), f2(p+4), f2(p+6)
			p += 8
		}
		visit(gid, a, b, c, d, dx, dy)
		if flags&more == 0 {
			return
		}
	}
}

// Paths zwraca kontury glifu jako odcinki z łukami, bez spłaszczania: do zapisu w formacie
// wektorowym. Glif złożony jest składany ze składników jak w Contours.
func (f *Font) Paths(gid int) []Path {
	return f.paths(gid, 0)
}

func (f *Font) paths(gid, depth int) []Path {
	g := f.raw(gid)
	if len(g) < 10 || depth > maxDepth {
		return nil
	}
	if s16(g, 0) >= 0 {
		xs, ys, flags, ends := f.points(g)
		var out []Path
		start := 0
		for _, end := range ends {
			if end >= start && end < len(xs) {
				out = append(out, outlinePath(xs[start:end+1], ys[start:end+1], flags[start:end+1]))
			}
			start = end + 1
		}
		return out
	}
	var out []Path
	f.components(g, func(child int, a, b, c, d, dx, dy float64) {
		move := func(q Point) Point { return Point{a*q.X + c*q.Y + dx, b*q.X + d*q.Y + dy} }
		for _, p := range f.paths(child, depth+1) {
			moved := Path{Start: move(p.Start)}
			for _, sg := range p.Segments {
				moved.Segments = append(moved.Segments, Segment{Ctrl: move(sg.Ctrl), To: move(sg.To), Curve: sg.Curve})
			}
			out = append(out, moved)
		}
	})
	return out
}
