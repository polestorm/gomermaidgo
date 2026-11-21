package pdf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// ErrOutlines zwraca osadzanie kroju, który nie ma konturów TrueType (tabeli glyf), np. kroju
// OpenType z konturami CFF.
var ErrOutlines = errors.New("pdf: the font has no TrueType outlines (glyf table); fonts with CFF outlines are not supported")

// ttf to krój TrueType odczytany w zakresie potrzebnym do osadzenia w PDF. Źródłem jest
// specyfikacja OpenType: tabele head, hhea, maxp, hmtx, loca, glyf, post, OS/2 i name.
type ttf struct {
	data        []byte
	tables      map[string][2]int // przesunięcie i długość
	unitsPerEm  int
	numGlyphs   int
	numHMetrics int
	longLoca    bool
	bbox        [4]int
	ascent      int
	descent     int
	capHeight   int
	italicAngle float64
	name        string
}

func be16(b []byte, off int) int  { return int(binary.BigEndian.Uint16(b[off:])) }
func bes16(b []byte, off int) int { return int(int16(binary.BigEndian.Uint16(b[off:]))) }
func be32(b []byte, off int) int  { return int(binary.BigEndian.Uint32(b[off:])) }

func parseTTF(data []byte) (*ttf, error) {
	if len(data) < 12 {
		return nil, errors.New("pdf: font file too short")
	}
	n := be16(data, 4)
	f := &ttf{data: data, tables: map[string][2]int{}}
	for i := 0; i < n; i++ {
		rec := 12 + 16*i
		if rec+16 > len(data) {
			return nil, errors.New("pdf: font file too short")
		}
		off, length := be32(data, rec+8), be32(data, rec+12)
		if off < 0 || length < 0 || off+length > len(data) {
			return nil, errors.New("pdf: font table lies outside the file")
		}
		f.tables[string(data[rec:rec+4])] = [2]int{off, length}
	}
	if _, ok := f.tables["glyf"]; !ok {
		return nil, ErrOutlines
	}
	for name, least := range map[string]int{"head": 54, "hhea": 36, "maxp": 6, "hmtx": 4, "loca": 2} {
		if t, ok := f.tables[name]; !ok || t[1] < least {
			return nil, fmt.Errorf("pdf: font table %q is missing or too short", name)
		}
	}
	head := f.tables["head"][0]
	f.unitsPerEm = be16(data, head+18)
	if f.unitsPerEm == 0 {
		return nil, errors.New("pdf: unitsPerEm is zero")
	}
	for i := range f.bbox {
		f.bbox[i] = bes16(data, head+36+2*i)
	}
	f.longLoca = be16(data, head+50) != 0
	hhea := f.tables["hhea"][0]
	f.ascent, f.descent, f.numHMetrics = bes16(data, hhea+4), bes16(data, hhea+6), be16(data, hhea+34)
	f.numGlyphs = be16(data, f.tables["maxp"][0]+4)
	f.capHeight = f.ascent
	if t, ok := f.tables["OS/2"]; ok && t[1] >= 90 && be16(data, t[0]) >= 2 {
		f.capHeight = bes16(data, t[0]+88)
	}
	if t, ok := f.tables["post"]; ok && t[1] >= 8 {
		f.italicAngle = float64(int32(binary.BigEndian.Uint32(data[t[0]+4:]))) / 65536
	}
	width := 2
	if f.longLoca {
		width = 4
	}
	if f.numHMetrics == 0 || f.tables["hmtx"][1] < 4*f.numHMetrics || f.tables["loca"][1] < width*(f.numGlyphs+1) {
		return nil, errors.New("pdf: font metrics or glyph index are too short")
	}
	f.name = f.postScriptName()
	return f, nil
}

// postScriptName czyta nazwę PostScript kroju z tabeli name (identyfikator 6); bez niej „Font”.
func (f *ttf) postScriptName() string {
	t, ok := f.tables["name"]
	if !ok || t[1] < 6 {
		return "Font"
	}
	d := f.data[t[0] : t[0]+t[1]]
	count, strings := be16(d, 2), be16(d, 4)
	for i := 0; i < count && 6+12*i+12 <= len(d); i++ {
		rec := 6 + 12*i
		if be16(d, rec+6) != 6 {
			continue
		}
		off, length := strings+be16(d, rec+10), be16(d, rec+8)
		if off+length > len(d) {
			continue
		}
		raw := d[off : off+length]
		var name []byte
		if platform := be16(d, rec); platform == 3 || platform == 0 { // UTF-16BE
			for k := 0; k+1 < len(raw); k += 2 {
				name = append(name, raw[k+1])
			}
		} else {
			name = raw
		}
		// W nazwie obiektu PDF zostają tylko litery, cyfry i łącznik.
		var clean []byte
		for _, c := range name {
			if c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '-' {
				clean = append(clean, c)
			}
		}
		if len(clean) > 0 {
			return string(clean)
		}
	}
	return "Font"
}

// advance zwraca szerokość glifu w jednostkach kroju.
func (f *ttf) advance(gid int) int {
	if gid >= f.numHMetrics {
		gid = f.numHMetrics - 1 // glify poza tabelą mają szerokość ostatniego wpisu
	}
	return be16(f.data, f.tables["hmtx"][0]+4*gid)
}

// glyph zwraca dane konturu glifu z tabeli glyf; pusty wycinek dla glifu bez konturu.
func (f *ttf) glyph(gid int) []byte {
	if gid < 0 || gid >= f.numGlyphs {
		return nil
	}
	loca, glyf := f.tables["loca"], f.tables["glyf"]
	var a, b int
	if f.longLoca {
		a, b = be32(f.data, loca[0]+4*gid), be32(f.data, loca[0]+4*gid+4)
	} else {
		a, b = 2*be16(f.data, loca[0]+2*gid), 2*be16(f.data, loca[0]+2*gid+2)
	}
	if a < 0 || b <= a || b > glyf[1] {
		return nil
	}
	return f.data[glyf[0]+a : glyf[0]+b]
}

// closure dopisuje do zbioru glify, z których składają się glify złożone.
func (f *ttf) closure(used map[int]bool) {
	var visit func(gid, depth int)
	visit = func(gid, depth int) {
		g := f.glyph(gid)
		if len(g) < 10 || bes16(g, 0) >= 0 || depth > 8 {
			return
		}
		// Glif złożony: lista składników, każdy z flagami, numerem glifu i przekształceniem.
		for p := 10; p+4 <= len(g); {
			flags, part := be16(g, p), be16(g, p+2)
			if !used[part] {
				used[part] = true
				visit(part, depth+1)
			}
			p += 4
			if flags&0x0001 != 0 { // argumenty 16-bitowe
				p += 4
			} else {
				p += 2
			}
			switch {
			case flags&0x0008 != 0: // jedna skala
				p += 2
			case flags&0x0040 != 0: // skala w obu osiach
				p += 4
			case flags&0x0080 != 0: // macierz 2×2
				p += 8
			}
			if flags&0x0020 == 0 { // nie ma dalszych składników
				break
			}
		}
	}
	for gid := range used {
		visit(gid, 0)
	}
}

// subset zwraca krój TrueType zawierający kontury tylko użytych glifów. Numery glifów zostają,
// więc tekst w PDF odwołuje się do nich bez przenumerowania; glify nieużyte są puste. Zostają
// tylko tabele potrzebne do rysowania: bez cmap, GSUB, GPOS, name i post.
func (f *ttf) subset(used map[int]bool) []byte {
	keep := map[int]bool{0: true}
	for gid := range used {
		keep[gid] = true
	}
	f.closure(keep)
	var glyf []byte
	loca := make([]byte, 4*(f.numGlyphs+1))
	for gid := 0; gid < f.numGlyphs; gid++ {
		binary.BigEndian.PutUint32(loca[4*gid:], uint32(len(glyf)))
		if keep[gid] {
			glyf = append(glyf, f.glyph(gid)...)
			for len(glyf)%4 != 0 {
				glyf = append(glyf, 0)
			}
		}
	}
	binary.BigEndian.PutUint32(loca[4*f.numGlyphs:], uint32(len(glyf)))
	head := append([]byte(nil), f.data[f.tables["head"][0]:f.tables["head"][0]+f.tables["head"][1]]...)
	binary.BigEndian.PutUint32(head[8:], 0)  // suma kontrolna całości, wyliczana na końcu
	binary.BigEndian.PutUint16(head[50:], 1) // loca w formacie długim

	tables := map[string][]byte{"head": head, "loca": loca, "glyf": glyf}
	for _, name := range []string{"hhea", "maxp", "hmtx", "cvt ", "fpgm", "prep"} {
		if t, ok := f.tables[name]; ok {
			tables[name] = f.data[t[0] : t[0]+t[1]]
		}
	}
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)

	n := len(names)
	entry, power := 0, 1
	for power*2 <= n {
		power, entry = power*2, entry+1
	}
	out := make([]byte, 12+16*n)
	binary.BigEndian.PutUint32(out[0:], 0x00010000)
	binary.BigEndian.PutUint16(out[4:], uint16(n))
	binary.BigEndian.PutUint16(out[6:], uint16(power*16))
	binary.BigEndian.PutUint16(out[8:], uint16(entry))
	binary.BigEndian.PutUint16(out[10:], uint16(n*16-power*16))
	checksum := func(b []byte) uint32 {
		var sum uint32
		for i := 0; i < len(b); i += 4 {
			var word [4]byte
			copy(word[:], b[i:min(i+4, len(b))])
			sum += binary.BigEndian.Uint32(word[:])
		}
		return sum
	}
	headAt := 0
	for i, name := range names {
		t := tables[name]
		rec := 12 + 16*i
		copy(out[rec:], name)
		binary.BigEndian.PutUint32(out[rec+4:], checksum(t))
		binary.BigEndian.PutUint32(out[rec+8:], uint32(len(out)))
		binary.BigEndian.PutUint32(out[rec+12:], uint32(len(t)))
		if name == "head" {
			headAt = len(out)
		}
		out = append(out, t...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	binary.BigEndian.PutUint32(out[headAt+8:], 0xB1B0AFBA-checksum(out))
	return out
}
