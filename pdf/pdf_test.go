package pdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/polestorm/gomermaidgo"
	"github.com/polestorm/gomermaidgo/fonts"
)

func corpus(t *testing.T) []string {
	t.Helper()
	files, _ := filepath.Glob("../testdata/corpus/*/*.mmd")
	if len(files) == 0 {
		t.Fatal("no corpus files")
	}
	return files
}

func source(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var (
	objHead  = regexp.MustCompile(`(?m)^(\d+) 0 obj\n`)
	mediaBox = regexp.MustCompile(`/MediaBox \[0 0 ([\d.]+) ([\d.]+)\]`)
	svgSize  = regexp.MustCompile(`<svg[^>]* width="([\d.]+)" height="([\d.]+)"`)
	svgText  = regexp.MustCompile(`<text[^>]*>([^<]*)</text>`)
	fontRef  = regexp.MustCompile(`/F(\d+) (\d+) 0 R`)
	toUni    = regexp.MustCompile(`/ToUnicode (\d+) 0 R`)
	bfchar   = regexp.MustCompile(`<([0-9A-F]{4})> <([0-9A-F]+)>`)
	showText = regexp.MustCompile(`BT ((?:/F\d+ [\d.]+ Tf [^<]*<[0-9A-F]*> Tj )+)ET`)
	showRun  = regexp.MustCompile(`/F(\d+) [\d.]+ Tf [^<]*<([0-9A-F]*)> Tj`)
)

// objects dzieli dokument na obiekty według tabeli odsyłaczy i sprawdza, że każdy wpis tabeli
// wskazuje początek swojego obiektu.
func objects(t *testing.T, name string, doc []byte) map[int][]byte {
	t.Helper()
	if !bytes.HasPrefix(doc, []byte("%PDF-1.7\n")) || !bytes.HasSuffix(doc, []byte("%%EOF\n")) {
		t.Fatalf("%s: not a complete PDF file", name)
	}
	at := bytes.LastIndex(doc, []byte("startxref\n"))
	xref, err := strconv.Atoi(strings.TrimSpace(string(doc[at+10 : len(doc)-6])))
	if err != nil || !bytes.HasPrefix(doc[xref:], []byte("xref\n")) {
		t.Fatalf("%s: startxref does not point at the cross-reference table", name)
	}
	lines := strings.Split(string(doc[xref:at]), "\n")
	var count int
	fmt.Sscanf(lines[1], "0 %d", &count)
	out := map[int][]byte{}
	for n := 1; n < count; n++ {
		off, _ := strconv.Atoi(lines[2+n][:10])
		head := fmt.Sprintf("%d 0 obj\n", n)
		if !bytes.HasPrefix(doc[off:], []byte(head)) {
			t.Fatalf("%s: cross-reference entry %d does not point at its object", name, n)
		}
		end := bytes.Index(doc[off:], []byte("\nendobj\n"))
		out[n] = doc[off+len(head) : off+end]
	}
	if got := len(objHead.FindAll(doc, -1)); got < count-1 {
		t.Fatalf("%s: %d objects in the file, %d in the table", name, got, count-1)
	}
	return out
}

func inflate(t *testing.T, obj []byte) []byte {
	t.Helper()
	a, b := bytes.Index(obj, []byte("stream\n")), bytes.LastIndex(obj, []byte("\nendstream"))
	if a < 0 || b < 0 {
		t.Fatal("object has no stream")
	}
	z, err := zlib.NewReader(bytes.NewReader(obj[a+7 : b]))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// texts odczytuje teksty strony: numery glifów z treści zamienia na znaki tabelą ToUnicode
// kroju, którym tekst jest zapisany.
func texts(t *testing.T, objs map[int][]byte) []string {
	t.Helper()
	tables := map[string]map[string]string{}
	for _, m := range fontRef.FindAllSubmatch(objs[3], -1) {
		n, _ := strconv.Atoi(string(m[2]))
		u := toUni.FindSubmatch(objs[n])
		if u == nil {
			t.Fatalf("font %s has no ToUnicode table", m[1])
		}
		un, _ := strconv.Atoi(string(u[1]))
		table := map[string]string{}
		for _, c := range bfchar.FindAllSubmatch(inflate(t, objs[un]), -1) {
			var units []uint16
			for i := 0; i+4 <= len(c[2]); i += 4 {
				v, _ := strconv.ParseUint(string(c[2][i:i+4]), 16, 16)
				units = append(units, uint16(v))
			}
			table[string(c[1])] = string(utf16.Decode(units))
		}
		tables[string(m[1])] = table
	}
	var out []string
	// Napis może składać się z kilku odcinków, każdy swoim krojem (krój główny i zapasowy).
	for _, text := range showText.FindAllSubmatch(inflate(t, objs[4]), -1) {
		var s strings.Builder
		for _, m := range showRun.FindAllSubmatch(text[1], -1) {
			for i := 0; i+4 <= len(m[2]); i += 4 {
				c, ok := tables[string(m[1])][string(m[2][i:i+4])]
				if !ok {
					t.Fatalf("glyph %s of font F%s is not in its ToUnicode table", m[2][i:i+4], m[1])
				}
				s.WriteString(c)
			}
		}
		out = append(out, s.String())
	}
	return out
}

func unescape(s string) string {
	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&amp;", "&").Replace(s)
}

// REQ: PDF-001 PDF-002 PDF-003
// Każdy plik korpusu daje poprawnie zbudowany dokument PDF: strona ma rozmiar rysunku SVG
// w punktach, a teksty strony są tymi samymi tekstami co w SVG, w tej samej kolejności.
func TestCorpus(t *testing.T) {
	for _, f := range corpus(t) {
		src := source(t, f)
		doc, err := Render(src)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		objs := objects(t, f, doc)
		svg, err := gomermaidgo.Render(src, gomermaidgo.WithEmojiText()) // PDF trzyma emoji jako tekst
		if err != nil {
			t.Fatal(err)
		}
		box, size := mediaBox.FindSubmatch(objs[3]), svgSize.FindSubmatch(svg)
		for i := 1; i <= 2; i++ {
			pt, _ := strconv.ParseFloat(string(box[i]), 64)
			pxs, _ := strconv.ParseFloat(string(size[i]), 64)
			if math.Abs(pt-pxs*0.75) > 0.01 {
				t.Errorf("%s: page is %s pt, the SVG %s px", f, box[i], size[i])
			}
		}
		var want []string
		for _, m := range svgText.FindAllSubmatch(svg, -1) {
			// Selektor wariantu nie ma glifu i w PDF jest pomijany.
			if s := strings.ReplaceAll(unescape(string(m[1])), "\ufe0f", ""); s != "" {
				want = append(want, s)
			}
		}
		got := texts(t, objs)
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s: texts differ from the SVG\n got %q\nwant %q", f, got, want)
		}
	}
}

// REQ: PDF-001
// Wynik jest powtarzalny co do bajta, także z opcjami wyglądu; tło, kolory i cienie dają
// poprawny dokument, a cień jest kształtem półprzezroczystym.
func TestDeterministicAndOptions(t *testing.T) {
	src := source(t, "../testdata/corpus/flowchart/015-styles.mmd")
	opts := []gomermaidgo.Option{gomermaidgo.WithBackground("#ffffff"), gomermaidgo.WithColors(), gomermaidgo.WithShadows(), gomermaidgo.WithFontSize(20)}
	a, err := Render(src, opts...)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Render(src, opts...)
	if !bytes.Equal(a, b) {
		t.Error("two renderings of the same source differ")
	}
	objs := objects(t, "styles", a)
	if !bytes.Contains(objs[3], []byte("/ExtGState << /G0 << /Type /ExtGState /ca 0.1 >>")) {
		t.Errorf("shadows should use a transparent graphics state: %s", objs[3])
	}
	content := string(inflate(t, objs[4]))
	if !strings.Contains(content, "1 1 1 rg") || !strings.Contains(content, "/G0 gs") || !strings.Contains(content, "] 0 d") {
		t.Error("background, shadow or dashed line is missing from the page content")
	}
}

// REQ: PDF-002
// Krój jest osadzony jako podzbiór: kontury tylko użytych glifów, numery glifów bez zmian,
// poprawna suma kontrolna pliku; plik jest wielokrotnie mniejszy od całego kroju.
func TestSubset(t *testing.T) {
	data := fonts.Arimo()
	f, err := parseTTF(data)
	if err != nil {
		t.Fatal(err)
	}
	used := map[int]bool{}
	w := &writer{byData: map[string]*face{}, opacity: map[string]string{}}
	fc, _ := w.face(data)
	for _, r := range "Zażółć gęślą jaźń 123" {
		used[fc.gid(r)] = true
	}
	sub := f.subset(used)
	s, err := parseTTF(sub)
	if err != nil {
		t.Fatalf("the subset is not a readable font: %v", err)
	}
	if s.numGlyphs != f.numGlyphs || !s.longLoca {
		t.Errorf("glyph count %d (was %d), long loca %v", s.numGlyphs, f.numGlyphs, s.longLoca)
	}
	closed := map[int]bool{0: true}
	for g := range used {
		closed[g] = true
	}
	f.closure(closed)
	if len(closed) <= len(used) {
		t.Error("accented letters are composite glyphs: the closure should add their parts")
	}
	for g := 0; g < f.numGlyphs; g++ {
		got, want := s.glyph(g), f.glyph(g)
		if closed[g] && !bytes.Equal(bytes.TrimRight(got, "\x00"), bytes.TrimRight(want, "\x00")) {
			t.Fatalf("glyph %d changed in the subset", g)
		}
		if !closed[g] && len(got) != 0 {
			t.Fatalf("unused glyph %d kept its outline", g)
		}
		if s.advance(g) != f.advance(g) {
			t.Fatalf("advance of glyph %d changed", g)
		}
	}
	var sum uint32
	for i := 0; i < len(sub); i += 4 {
		sum += uint32(sub[i])<<24 | uint32(sub[i+1])<<16 | uint32(sub[i+2])<<8 | uint32(sub[i+3])
	}
	if sum != 0xB1B0AFBA {
		t.Errorf("whole-file checksum is %#x", sum)
	}
	if len(sub)*5 > len(data) {
		t.Errorf("subset is %d bytes of %d: too large", len(sub), len(data))
	}
	if _, ok := s.tables["cmap"]; ok {
		t.Error("the subset should not carry tables that drawing does not need")
	}
}

// REQ: PDF-004
// Bez pliku kroju nie ma czego osadzić: miara własna i krój znany tylko z nazwy dają błąd;
// krój bez konturów TrueType także. Własny krój z pliku jest osadzany, a odmiana niepodana
// korzysta z kroju zwykłego.
func TestFonts(t *testing.T) {
	src := "flowchart TD\nA[\"`**gruby** i *pochylony*`\"] --> B"
	if _, err := Render(src, gomermaidgo.WithMeasure(func(string, float64) float64 { return 10 })); !errors.Is(err, gomermaidgo.ErrFontRequired) {
		t.Errorf("custom measure: got %v", err)
	}
	if _, err := Render(src, gomermaidgo.WithFontFamily("Arial")); !errors.Is(err, gomermaidgo.ErrFontRequired) {
		t.Errorf("font known by name only: got %v", err)
	}
	doc, err := Render(src, gomermaidgo.WithFont(fonts.ArimoItalic()), gomermaidgo.WithFontFamily("Własny"))
	if err != nil {
		t.Fatal(err)
	}
	objs := objects(t, "own font", doc)
	if refs := fontRef.FindAllSubmatch(objs[3], -1); len(refs) != 1 {
		t.Errorf("one font file given for all styles should be embedded once, got %d", len(refs))
	}
	got := texts(t, objs)
	sort.Strings(got)
	if strings.Join(got, "|") != "B| i |gruby|pochylony" && strings.Join(got, "|") != "B|gruby|i|pochylony" {
		t.Errorf("texts: %q", got)
	}
	otto := append([]byte("OTTO"), make([]byte, 60)...)
	if _, err := parseTTF(otto); !errors.Is(err, ErrOutlines) {
		t.Errorf("font without glyf: got %v", err)
	}
	if _, err := Render("flowchart TD\nA --> "); err == nil {
		t.Error("a syntax error should be returned, not drawn")
	}
}

// REQ: PDF-003
func TestColors(t *testing.T) {
	for in, want := range map[string][3]float64{
		"#ff0000": {1, 0, 0}, "#0F0": {0, 1, 0}, "rgb(0, 0, 255)": {0, 0, 1}, "White": {1, 1, 1},
		"rebeccapurple": {0.4, 0.2, 0.6}, "nieznany": {0, 0, 0}, "rgb(300,0,0)": {1, 0, 0},
	} {
		r, g, b, ok := color(in)
		if !ok || math.Abs(r-want[0]) > 0.001 || math.Abs(g-want[1]) > 0.001 || math.Abs(b-want[2]) > 0.001 {
			t.Errorf("%q: got %v %v %v %v", in, r, g, b, ok)
		}
	}
	for _, in := range []string{"", "none", "transparent"} {
		if _, _, _, ok := color(in); ok {
			t.Errorf("%q should paint nothing", in)
		}
	}
}

// REQ: PDF-001
// Dowolne źródło, które daje SVG, daje też PDF, bez paniki.
func FuzzRender(f *testing.F) {
	files, _ := filepath.Glob("../testdata/corpus/*/*.mmd")
	for _, p := range files {
		b, _ := os.ReadFile(p)
		f.Add(string(b))
	}
	f.Fuzz(func(t *testing.T, src string) {
		_, svgErr := gomermaidgo.Render(src)
		doc, err := Render(src)
		if (svgErr == nil) != (err == nil) {
			t.Fatalf("SVG error %v, PDF error %v", svgErr, err)
		}
		if err == nil && !bytes.HasSuffix(doc, []byte("%%EOF\n")) {
			t.Fatal("incomplete PDF")
		}
	})
}

// REQ: PDF-002
// Znak, którego nie ma w kroju, jest rysowany glifem „brak znaku”, ale tekst do kopiowania
// i wyszukiwania zostaje prawdziwy: napis z takim znakiem niesie go wprost (ActualText), a tabela
// ToUnicode nie przypisuje glifowi braku żadnego konkretnego znaku.
func TestMissingGlyphsKeepTheirText(t *testing.T) {
	doc, err := Render("flowchart LR\nA[\"漢 Zgoda\"] --> B[\"語 Start\"]\nB --> C[Zwykły]")
	if err != nil {
		t.Fatal(err)
	}
	objs := objects(t, "missing", doc)
	var content string
	for _, o := range objs {
		if bytes.Contains(o, []byte("/FlateDecode")) {
			if s := string(inflate(t, o)); strings.Contains(s, " Tj ET") {
				content = s
			}
		}
	}
	encode := func(s string) string {
		var b strings.Builder
		for _, u := range utf16.Encode([]rune(s)) {
			fmt.Fprintf(&b, "%04X", u)
		}
		return b.String()
	}
	for _, want := range []string{"漢 Zgoda", "語 Start"} {
		if !strings.Contains(content, "/ActualText <FEFF"+encode(want)+"> >> BDC") {
			t.Errorf("text %q should be carried as ActualText", want)
		}
	}
	if strings.Count(content, "BDC") != 2 || strings.Count(content, "EMC") != 2 || strings.Contains(content, encode("Zwykły")) {
		t.Errorf("only the two texts with missing glyphs should be wrapped")
	}
	for _, got := range texts(t, objs) {
		if strings.Contains(got, "漢") && strings.Contains(got, "Start") {
			t.Errorf("the missing-glyph slot should not stand for the first missing character: %q", got)
		}
	}
}

// REQ: COM-005
// Emoji są w PDF tekstem zapisanym krojem zapasowym, osadzonym jako drugi podzbiór; napis mieszany
// składa się z odcinków obu krojów.
func TestEmojiUseTheFallbackFont(t *testing.T) {
	src := "flowchart LR\nA[\"🚀 Start ✅\"] --> B[Zwykły]"
	doc, err := Render(src)
	if err != nil {
		t.Fatal(err)
	}
	objs := objects(t, "emoji", doc)
	if n := len(fontRef.FindAllSubmatch(objs[3], -1)); n != 2 {
		t.Fatalf("the page should use two fonts, has %d", n)
	}
	if !bytes.Contains(doc, []byte("+NotoEmoji")) {
		t.Errorf("the fallback font should be embedded as a subset")
	}
	got := texts(t, objs)
	if strings.Join(got, "|") != "🚀|Start|✅|Zwykły" {
		t.Errorf("texts: %q", got)
	}
	var content string
	for _, o := range objs {
		if bytes.Contains(o, []byte("/FlateDecode")) {
			if s := string(inflate(t, o)); strings.Contains(s, " Tj ET") {
				content = s
			}
		}
	}
	if strings.Contains(content, "ActualText") || len(showRun.FindAllString(content, -1)) != 4 {
		t.Errorf("a mixed text should be three texts without ActualText: %d texts", len(showRun.FindAllString(content, -1)))
	}
	plain, _ := Render("flowchart LR\nA[Zwykły]")
	if bytes.Contains(plain, []byte("NotoEmoji")) {
		t.Errorf("a drawing without emoji should not embed the fallback font")
	}
}
