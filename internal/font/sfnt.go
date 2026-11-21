// Package font czyta z pliku fontu TrueType/OpenType tylko to, czego potrzebuje układ
// diagramu: szerokości znaków. Źródłem jest specyfikacja OpenType (tabele head, hhea,
// maxp, hmtx, cmap); pakiet nie rasteryzuje i nie kształtuje tekstu.
package font

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Face to odczytane metryki jednego kroju.
type Face struct {
	unitsPerEm uint16
	advances   []uint16 // szerokość dla glifów 0..numberOfHMetrics-1
	numGlyphs  int
	segments   []segment // cmap format 4
	groups     []group   // cmap format 12
	data       []byte
	cmap4      int // przesunięcie podtabeli formatu 4 (dla idRangeOffset), 0 gdy brak
}

type segment struct {
	start, end   uint16
	delta        uint16
	rangeOffset  uint16
	rangeOffsetP int // pozycja pola idRangeOffset w danych
}

type group struct {
	start, end, glyph uint32
}

var errShort = errors.New("font: file too short")

type reader struct{ b []byte }

func (r reader) u16(off int) (uint16, error) {
	if off < 0 || off+2 > len(r.b) {
		return 0, errShort
	}
	return binary.BigEndian.Uint16(r.b[off:]), nil
}

func (r reader) u32(off int) (uint32, error) {
	if off < 0 || off+4 > len(r.b) {
		return 0, errShort
	}
	return binary.BigEndian.Uint32(r.b[off:]), nil
}

// Parse czyta pojedynczy font (nie kolekcję TTC).
func Parse(data []byte) (*Face, error) {
	r := reader{data}
	tag, err := r.u32(0)
	if err != nil {
		return nil, err
	}
	switch tag {
	case 0x00010000, 0x4F54544F, 0x74727565: // TrueType, "OTTO", "true"
	default:
		return nil, fmt.Errorf("font: unsupported container %#08x", tag)
	}
	n, err := r.u16(4)
	if err != nil {
		return nil, err
	}
	tables := map[string][2]int{}
	for i := 0; i < int(n); i++ {
		rec := 12 + 16*i
		if rec+16 > len(data) {
			return nil, errShort
		}
		off, _ := r.u32(rec + 8)
		length, _ := r.u32(rec + 12)
		if int(off) > len(data) || int(off)+int(length) > len(data) {
			return nil, errShort
		}
		tables[string(data[rec:rec+4])] = [2]int{int(off), int(length)}
	}
	need := func(name string) (int, error) {
		t, ok := tables[name]
		if !ok {
			return 0, fmt.Errorf("font: missing table %q", name)
		}
		return t[0], nil
	}
	f := &Face{data: data}
	head, err := need("head")
	if err != nil {
		return nil, err
	}
	if f.unitsPerEm, err = r.u16(head + 18); err != nil {
		return nil, err
	}
	if f.unitsPerEm == 0 {
		return nil, errors.New("font: unitsPerEm is zero")
	}
	hhea, err := need("hhea")
	if err != nil {
		return nil, err
	}
	numH, err := r.u16(hhea + 34)
	if err != nil {
		return nil, err
	}
	maxp, err := need("maxp")
	if err != nil {
		return nil, err
	}
	ng, err := r.u16(maxp + 4)
	if err != nil {
		return nil, err
	}
	f.numGlyphs = int(ng)
	hmtx, err := need("hmtx")
	if err != nil {
		return nil, err
	}
	if numH == 0 {
		return nil, errors.New("font: no horizontal metrics")
	}
	f.advances = make([]uint16, numH)
	for i := range f.advances {
		if f.advances[i], err = r.u16(hmtx + 4*i); err != nil {
			return nil, err
		}
	}
	cmap, err := need("cmap")
	if err != nil {
		return nil, err
	}
	if err := f.readCmap(r, cmap); err != nil {
		return nil, err
	}
	return f, nil
}

// readCmap wybiera najlepszą podtabelę Unicode: pełny zakres (format 12) przed BMP (format 4).
func (f *Face) readCmap(r reader, cmap int) error {
	n, err := r.u16(cmap + 2)
	if err != nil {
		return err
	}
	best4, best12 := 0, 0
	for i := 0; i < int(n); i++ {
		rec := cmap + 4 + 8*i
		platform, err := r.u16(rec)
		if err != nil {
			return err
		}
		encoding, _ := r.u16(rec + 2)
		off, err := r.u32(rec + 4)
		if err != nil {
			return err
		}
		unicode := platform == 0 || (platform == 3 && (encoding == 1 || encoding == 10))
		if !unicode {
			continue
		}
		sub := cmap + int(off)
		format, err := r.u16(sub)
		if err != nil {
			return err
		}
		switch format {
		case 4:
			if best4 == 0 {
				best4 = sub
			}
		case 12:
			if best12 == 0 {
				best12 = sub
			}
		}
	}
	if best12 != 0 {
		count, err := r.u32(best12 + 12)
		if err != nil {
			return err
		}
		if int(count) > (len(r.b)-best12)/12 {
			return errShort
		}
		f.groups = make([]group, count)
		for i := range f.groups {
			g := best12 + 16 + 12*i
			s, err := r.u32(g)
			if err != nil {
				return err
			}
			e, _ := r.u32(g + 4)
			gl, err := r.u32(g + 8)
			if err != nil {
				return err
			}
			f.groups[i] = group{s, e, gl}
		}
		return nil
	}
	if best4 == 0 {
		return errors.New("font: no Unicode cmap subtable (format 4 or 12)")
	}
	segX2, err := r.u16(best4 + 6)
	if err != nil {
		return err
	}
	segCount := int(segX2) / 2
	endP := best4 + 14
	startP := endP + 2*segCount + 2
	deltaP := startP + 2*segCount
	rangeP := deltaP + 2*segCount
	f.segments = make([]segment, segCount)
	for i := range f.segments {
		e, err := r.u16(endP + 2*i)
		if err != nil {
			return err
		}
		s, _ := r.u16(startP + 2*i)
		d, _ := r.u16(deltaP + 2*i)
		ro, err := r.u16(rangeP + 2*i)
		if err != nil {
			return err
		}
		f.segments[i] = segment{start: s, end: e, delta: d, rangeOffset: ro, rangeOffsetP: rangeP + 2*i}
	}
	return nil
}

// glyph zwraca numer glifu znaku albo 0, gdy font go nie ma.
func (f *Face) glyph(c rune) uint32 {
	if f.groups != nil {
		lo, hi := 0, len(f.groups)
		for lo < hi {
			m := (lo + hi) / 2
			g := f.groups[m]
			switch {
			case uint32(c) < g.start:
				hi = m
			case uint32(c) > g.end:
				lo = m + 1
			default:
				return g.glyph + uint32(c) - g.start
			}
		}
		return 0
	}
	if c > 0xFFFF {
		return 0
	}
	u := uint16(c)
	for _, s := range f.segments {
		if u > s.end {
			continue
		}
		if u < s.start {
			return 0
		}
		if s.rangeOffset == 0 {
			return uint32(u + s.delta)
		}
		p := s.rangeOffsetP + int(s.rangeOffset) + 2*int(u-s.start)
		if p+2 > len(f.data) {
			return 0
		}
		g := binary.BigEndian.Uint16(f.data[p:])
		if g == 0 {
			return 0
		}
		return uint32(g + s.delta)
	}
	return 0
}

// Advance zwraca szerokość znaku w jednostkach em (1 = rozmiar fontu) oraz informację,
// czy font ma ten znak. Dla brakującego znaku zwraca szerokość glifu zastępczego.
func (f *Face) Advance(c rune) (em float64, ok bool) {
	g := f.glyph(c)
	idx := int(g)
	if idx >= len(f.advances) {
		// Glify poza tabelą dziedziczą ostatnią zapisaną szerokość (specyfikacja hmtx).
		idx = len(f.advances) - 1
	}
	return float64(f.advances[idx]) / float64(f.unitsPerEm), g != 0
}

// TextWidth zwraca szerokość tekstu w pikselach dla rozmiaru size oraz liczbę znaków,
// których font nie zawiera.
func (f *Face) TextWidth(text string, size float64) (width float64, missing int) {
	for _, c := range text {
		em, ok := f.Advance(c)
		if !ok {
			missing++
		}
		width += em * size
	}
	return width, missing
}
