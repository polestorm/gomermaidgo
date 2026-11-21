package font

import (
	"encoding/binary"
	"math"
	"testing"
)

// buildFont składa minimalny font z trzema glifami: .notdef (500), A (600), ł (300), em = 1000.
// Font jest tworzony w teście, żeby repozytorium nie zawierało cudzych plików fontów.
func buildFont(cmap []byte) []byte {
	be16 := func(v ...uint16) []byte {
		b := make([]byte, 2*len(v))
		for i, x := range v {
			binary.BigEndian.PutUint16(b[2*i:], x)
		}
		return b
	}
	head := make([]byte, 54)
	binary.BigEndian.PutUint16(head[18:], 1000) // unitsPerEm
	hhea := make([]byte, 36)
	binary.BigEndian.PutUint16(hhea[34:], 3) // numberOfHMetrics
	maxp := make([]byte, 6)
	binary.BigEndian.PutUint16(maxp[4:], 3) // numGlyphs
	hmtx := be16(500, 0, 600, 0, 300, 0)

	tables := []struct {
		tag  string
		data []byte
	}{{"cmap", cmap}, {"head", head}, {"hhea", hhea}, {"hmtx", hmtx}, {"maxp", maxp}}
	out := make([]byte, 12+16*len(tables))
	binary.BigEndian.PutUint32(out, 0x00010000)
	binary.BigEndian.PutUint16(out[4:], uint16(len(tables)))
	for i, t := range tables {
		rec := out[12+16*i:]
		copy(rec, t.tag)
		binary.BigEndian.PutUint32(rec[8:], uint32(len(out)))
		binary.BigEndian.PutUint32(rec[12:], uint32(len(t.data)))
		out = append(out, t.data...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	return out
}

func cmapFormat4() []byte {
	be16 := func(v ...uint16) []byte {
		b := make([]byte, 2*len(v))
		for i, x := range v {
			binary.BigEndian.PutUint16(b[2*i:], x)
		}
		return b
	}
	// Segmenty: A → glif 1, ł (U+0142) → glif 2, znacznik końca 0xFFFF.
	sub := be16(4, 0, 0, 6, 0, 0, 0) // format, length, language, segCountX2, searchRange, entrySelector, rangeShift
	sub = append(sub, be16(0x41, 0x142, 0xFFFF)...)
	sub = append(sub, be16(0)...) // reservedPad
	sub = append(sub, be16(0x41, 0x142, 0xFFFF)...)
	sub = append(sub, be16(0xFFC0, 0, 1)...) // idDelta: A przez deltę, ł przez tablicę glifów
	sub = append(sub, be16(0, 4, 0)...)      // idRangeOffset: dla ł wskazuje na glyphIdArray
	sub = append(sub, be16(2)...)            // glyphIdArray
	binary.BigEndian.PutUint16(sub[2:], uint16(len(sub)))
	header := be16(0, 1, 3, 1) // version, numTables, platform 3, encoding 1
	header = append(header, 0, 0, 0, 12)
	return append(header, sub...)
}

func cmapFormat12() []byte {
	b := make([]byte, 12+16+2*12)
	binary.BigEndian.PutUint16(b[2:], 1)  // numTables
	binary.BigEndian.PutUint16(b[4:], 3)  // platform
	binary.BigEndian.PutUint16(b[6:], 10) // encoding
	binary.BigEndian.PutUint32(b[8:], 12) // offset
	sub := b[12:]
	binary.BigEndian.PutUint16(sub, 12)
	binary.BigEndian.PutUint32(sub[4:], uint32(len(sub)))
	binary.BigEndian.PutUint32(sub[12:], 2)
	for i, g := range [][3]uint32{{0x41, 0x41, 1}, {0x142, 0x142, 2}} {
		binary.BigEndian.PutUint32(sub[16+12*i:], g[0])
		binary.BigEndian.PutUint32(sub[20+12*i:], g[1])
		binary.BigEndian.PutUint32(sub[24+12*i:], g[2])
	}
	return b
}

// REQ: SEQ-013
func TestParseAndMeasure(t *testing.T) {
	for name, cmap := range map[string][]byte{"format 4": cmapFormat4(), "format 12": cmapFormat12()} {
		f, err := Parse(buildFont(cmap))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		w, missing := f.TextWidth("AłA?", 20)
		// A 0,6 em + ł 0,3 em + A 0,6 em + brakujący znak 0,5 em = 2,0 em = 40 px.
		if math.Abs(w-40) > 1e-9 || missing != 1 {
			t.Errorf("%s: width %v missing %d, want 40 and 1", name, w, missing)
		}
	}
}

// REQ: SEQ-013
func TestParseRejectsGarbage(t *testing.T) {
	good := buildFont(cmapFormat4())
	// Ostatnie dwa bajty to dopełnienie tabeli maxp; ich brak nie psuje fontu.
	for n := 0; n < len(good)-2; n += 7 {
		if _, err := Parse(good[:n]); err == nil {
			t.Errorf("truncated font of %d bytes was accepted", n)
		}
	}
	if _, err := Parse([]byte("<svg>not a font</svg>")); err == nil {
		t.Error("non-font data was accepted")
	}
}

// REQ: SEQ-013
func TestApproxIsNotNarrowerThanTypical(t *testing.T) {
	// Miara zastępcza ma zawyżać: tekst zmierzony nią nie może być węższy niż 0,45 em na znak.
	s := "Zażółć gęślą jaźń 0123456789"
	if w := ApproxTextWidth(s, 10); w < 0.45*10*float64(len([]rune(s))) {
		t.Errorf("approximate width %v is suspiciously small", w)
	}
}
