package glyf

import (
	"testing"

	"github.com/polestorm/gomermaidgo/fonts"
	"github.com/polestorm/gomermaidgo/internal/font"
)

// REQ: PNG-003
// Kontury z dołączonego kroju: litera z otworem, litera złożona ze znakiem diakrytycznym, odstęp.
func TestContours(t *testing.T) {
	data := fonts.Recursive()
	f, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	cmap, err := font.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	count := func(c rune) int { return len(f.Contours(int(cmap.GlyphID(c)))) }
	if count('o') != 2 || count('l') != 1 || count(' ') != 0 {
		t.Errorf("contours of o, l, space: %d %d %d", count('o'), count('l'), count(' '))
	}
	if count('ó') != count('o')+1 || count('ż') < 2 || count('Ł') < 1 {
		t.Errorf("letters with marks: ó %d, ż %d, Ł %d", count('ó'), count('ż'), count('Ł'))
	}
	// Kontur leży w polu glifu: między linią podstawową a wysokością wersalików, na szerokość znaku.
	gid := int(cmap.GlyphID('H'))
	for _, contour := range f.Contours(gid) {
		for _, p := range contour {
			if p.X < 0 || p.X > float64(f.Advance(gid)) || p.Y < -1 || p.Y > float64(f.UnitsPerEm) {
				t.Fatalf("point of H outside its box: %+v", p)
			}
		}
	}
	if f.Advance(gid) <= 0 || f.Contours(-1) != nil || f.Contours(1<<20) != nil {
		t.Errorf("advance or glyphs out of range")
	}
	if _, err := Parse([]byte("to nie jest krój")); err == nil {
		t.Errorf("Parse should reject a file that is not a font")
	}
	// Uszkodzone dane glifów nie mogą wywołać paniki.
	broken := append([]byte{}, data...)
	for i := f.glyf[0]; i < f.glyf[0]+f.glyf[1]; i += 7 {
		broken[i] ^= 0xa5
	}
	if g, err := Parse(broken); err == nil {
		for gid := 0; gid < g.numGlyphs; gid++ {
			g.Contours(gid)
		}
	}
}
