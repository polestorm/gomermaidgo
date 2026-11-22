// Package pdf zapisuje diagram jako jednostronicowy dokument PDF, bez pośrednictwa SVG i HTML.
//
// Rysunek pochodzi z tej samej sceny co SVG z pakietu gomermaidgo, więc oba formaty mają te same
// położenia i rozmiary. Tekst jest prawdziwym tekstem (da się go zaznaczać i wyszukiwać), a krój
// jest osadzony w pliku w wersji obciętej do użytych znaków. Źródłem jest specyfikacja PDF
// (ISO 32000-1) i specyfikacja OpenType.
//
// Pakiet jest opcjonalny: program, który go nie importuje, nie zawiera jego kodu.
package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/polestorm/gomermaidgo"
	"github.com/polestorm/gomermaidgo/internal/bridge"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Render zamienia źródło diagramu na dokument PDF o stronie dopasowanej do rysunku. Przyjmuje te
// same opcje co gomermaidgo.Render. Krój musi być znany z pliku: dołączony albo podany w WithFont;
// inaczej zwraca gomermaidgo.ErrFontRequired.
func Render(source string, opts ...gomermaidgo.Option) ([]byte, error) {
	options := make([]any, len(opts))
	for i, o := range opts {
		options[i] = o
	}
	sc, fonts, err := bridge.Build(source, options)
	if err != nil {
		return nil, err
	}
	return write(sc, fonts)
}

// px to punkty PDF na piksel rysunku: piksel CSS ma 1/96 cala, punkt 1/72.
const px = 0.75

// face to jeden krój w dokumencie: plik, odwzorowanie znaków i glify użyte na rysunku.
type face struct {
	ttf   *ttf
	cmap  *font.Face
	used  map[int]rune // glif → znak, dla tabeli ToUnicode
	index int          // numer zasobu /F
}

func (f *face) gid(r rune) int {
	g := int(f.cmap.GlyphID(r))
	if _, seen := f.used[g]; !seen {
		f.used[g] = r
	}
	return g
}

// num zapisuje liczbę z dokładnością do trzech miejsc, bez zbędnych zer; zawsze tak samo.
func num(v float64) string {
	v = math.Round(v*1000) / 1000
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

type writer struct {
	buf     bytes.Buffer // strumień treści strony
	faces   []*face
	byData  map[string]*face
	opacity map[string]string // wartość /ca → nazwa stanu graficznego
}

func (w *writer) op(format string, args ...any) { fmt.Fprintf(&w.buf, format+"\n", args...) }

// face zwraca krój dla pliku; ten sam plik podany jako dwie odmiany jest osadzany raz.
func (w *writer) face(data []byte) (*face, error) {
	if f, ok := w.byData[string(data)]; ok {
		return f, nil
	}
	t, err := parseTTF(data)
	if err != nil {
		return nil, err
	}
	c, err := font.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	f := &face{ttf: t, cmap: c, used: map[int]rune{}, index: len(w.faces)}
	w.faces = append(w.faces, f)
	w.byData[string(data)] = f
	return f, nil
}

// Okrąg i zaokrąglony narożnik jako krzywe Béziera: punkty kontrolne w tej części promienia.
const kappa = 0.5523

func (w *writer) ellipse(cx, cy, rx, ry float64) {
	w.op("%s %s m", num(cx+rx), num(cy))
	w.op("%s %s %s %s %s %s c", num(cx+rx), num(cy+ry*kappa), num(cx+rx*kappa), num(cy+ry), num(cx), num(cy+ry))
	w.op("%s %s %s %s %s %s c", num(cx-rx*kappa), num(cy+ry), num(cx-rx), num(cy+ry*kappa), num(cx-rx), num(cy))
	w.op("%s %s %s %s %s %s c", num(cx-rx), num(cy-ry*kappa), num(cx-rx*kappa), num(cy-ry), num(cx), num(cy-ry))
	w.op("%s %s %s %s %s %s c", num(cx+rx*kappa), num(cy-ry), num(cx+rx), num(cy-ry*kappa), num(cx+rx), num(cy))
	w.op("h")
}

func (w *writer) rect(x, y, width, height, r float64) {
	r = math.Min(r, math.Min(width, height)/2)
	if r <= 0 {
		w.op("%s %s %s %s re", num(x), num(y), num(width), num(height))
		return
	}
	k := r * (1 - kappa)
	x1, y1 := x+width, y+height
	w.op("%s %s m", num(x+r), num(y))
	w.op("%s %s l", num(x1-r), num(y))
	w.op("%s %s %s %s %s %s c", num(x1-k), num(y), num(x1), num(y+k), num(x1), num(y+r))
	w.op("%s %s l", num(x1), num(y1-r))
	w.op("%s %s %s %s %s %s c", num(x1), num(y1-k), num(x1-k), num(y1), num(x1-r), num(y1))
	w.op("%s %s l", num(x+r), num(y1))
	w.op("%s %s %s %s %s %s c", num(x+k), num(y1), num(x), num(y1-k), num(x), num(y1-r))
	w.op("%s %s l", num(x), num(y+r))
	w.op("%s %s %s %s %s %s c", num(x), num(y+k), num(x+k), num(y), num(x+r), num(y))
	w.op("h")
}

// paint ustawia kolory, grubość i wzór linii oraz przezroczystość kształtu i zwraca operator
// malowania ścieżki: wypełnienie, obrys, oba albo n (nic).
func (w *writer) paint(it scene.Item) string {
	r, g, b, fill := color(it.Fill)
	if fill {
		w.op("%s %s %s rg", num(r), num(g), num(b))
		if it.FillOpacity > 0 && it.FillOpacity < 1 {
			key := num(it.FillOpacity)
			name, ok := w.opacity[key]
			if !ok {
				name = fmt.Sprintf("G%d", len(w.opacity))
				w.opacity[key] = name
			}
			w.op("/%s gs", name)
		}
	}
	r, g, b, stroke := color(it.Stroke)
	if stroke {
		width := it.StrokeWidth
		if width == 0 {
			width = 1
		}
		w.op("%s %s %s RG", num(r), num(g), num(b))
		w.op("%s w", num(width))
		if dash := strings.FieldsFunc(it.Dash, func(c rune) bool { return c == ' ' || c == ',' }); len(dash) > 0 {
			ok := true
			for _, d := range dash {
				if v, err := strconv.ParseFloat(d, 64); err != nil || v < 0 {
					ok = false
				}
			}
			if ok {
				w.op("[%s] 0 d", strings.Join(dash, " "))
			}
		}
	}
	switch {
	case fill && stroke:
		return "B"
	case fill:
		return "f"
	case stroke:
		return "S"
	}
	return "n"
}

// text zapisuje linię tekstu: numery glifów w kroju osadzonym, od położenia wynikającego
// z zakotwiczenia i szerokości policzonej z tego samego kroju.
func (w *writer) text(it scene.Item, size float64, f *face) {
	r, g, b, ok := color(it.Fill)
	if !ok || it.Text == "" {
		return
	}
	var hex strings.Builder
	units := 0
	for _, c := range it.Text {
		gid := f.gid(c)
		units += f.ttf.advance(gid)
		fmt.Fprintf(&hex, "%04X", gid)
	}
	width := float64(units) / float64(f.ttf.unitsPerEm) * size
	x := it.X
	switch it.Anchor {
	case "middle":
		x -= width / 2
	case "end":
		x -= width
	}
	w.op("%s %s %s rg", num(r), num(g), num(b))
	// Układ strony ma oś y w dół; tekst odwraca ją z powrotem, żeby litery nie stały do góry nogami.
	if it.Vertical {
		// Ćwierć obrotu wokół punktu zaczepienia: kierunek pisma biegnie w górę strony.
		w.op("BT /F%d %s Tf 0 -1 -1 0 %s %s Tm <%s> Tj ET", f.index, num(size), num(it.X), num(it.Y-(x-it.X)), hex.String())
		return
	}
	w.op("BT /F%d %s Tf 1 0 0 -1 %s %s Tm <%s> Tj ET", f.index, num(size), num(x), num(it.Y), hex.String())
}

func (w *writer) item(it scene.Item, fontSize float64, fonts [3]*face) {
	w.op("q")
	defer w.op("Q")
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
		w.text(it, size, f)
		return
	case scene.Rect:
		op := w.paint(it)
		w.rect(it.X, it.Y, it.W, it.H, it.RX)
		w.op(op)
	case scene.Circle:
		op := w.paint(it)
		w.ellipse(it.X, it.Y, it.R, it.R)
		w.op(op)
	case scene.Ellipse:
		op := w.paint(it)
		w.ellipse(it.X, it.Y, it.W/2, it.H/2)
		w.op(op)
	case scene.Line:
		it.Fill = ""
		op := w.paint(it)
		w.op("%s %s m %s %s l", num(it.X), num(it.Y), num(it.X2), num(it.Y2))
		w.op(op)
	case scene.Polyline, scene.Polygon:
		if len(it.Points) == 0 {
			return
		}
		op := w.paint(it)
		for i, p := range it.Points {
			cmd := "l"
			if i == 0 {
				cmd = "m"
			}
			w.op("%s %s %s", num(p.X), num(p.Y), cmd)
		}
		if it.Kind == scene.Polygon {
			w.op("h")
		}
		w.op(op)
	case scene.Path:
		if len(it.Segments) == 0 {
			return
		}
		op := w.paint(it)
		for _, sg := range it.Segments {
			switch sg.Cmd {
			case 'M':
				w.op("%s %s m", num(sg.To.X), num(sg.To.Y))
			case 'L':
				w.op("%s %s l", num(sg.To.X), num(sg.To.Y))
			case 'C':
				w.op("%s %s %s %s %s %s c", num(sg.C1.X), num(sg.C1.Y), num(sg.C2.X), num(sg.C2.Y), num(sg.To.X), num(sg.To.Y))
			}
		}
		w.op(op)
	}
}

// str zapisuje napis PDF w UTF-16BE ze znacznikiem kolejności bajtów, jako ciąg szesnastkowy.
func str(s string) string {
	var b strings.Builder
	b.WriteString("<FEFF")
	for _, u := range utf16.Encode([]rune(s)) {
		fmt.Fprintf(&b, "%04X", u)
	}
	b.WriteString(">")
	return b.String()
}

func deflate(data []byte) []byte {
	var b bytes.Buffer
	z, _ := zlib.NewWriterLevel(&b, zlib.BestCompression)
	z.Write(data)
	z.Close()
	return b.Bytes()
}

// write składa dokument: stronę z rysunkiem i osadzone kroje.
func write(sc *scene.Scene, data bridge.Fonts) ([]byte, error) {
	w := &writer{byData: map[string]*face{}, opacity: map[string]string{}}
	var fonts [3]*face
	for i, d := range [][]byte{data.Regular, data.Bold, data.Italic} {
		f, err := w.face(d)
		if err != nil {
			return nil, err
		}
		fonts[i] = f
	}
	bounds := sc.Bounds()
	x0, y0 := bounds.X0-sc.Margin, bounds.Y0-sc.Margin
	width, height := bounds.Width()+2*sc.Margin, bounds.Height()+2*sc.Margin

	// Układ strony: piksele rysunku, początek w lewym górnym rogu, oś y w dół, jak w SVG.
	w.op("%s 0 0 %s %s %s cm", num(px), num(-px), num(-x0*px), num((y0+height)*px))
	if sc.Background != "" {
		w.item(scene.Item{Kind: scene.Rect, X: x0, Y: y0, W: width, H: height, Fill: sc.Background}, sc.FontSize, fonts)
	}
	items := make([]scene.Item, len(sc.Items))
	copy(items, sc.Items)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Layer < items[j].Layer })
	for _, it := range items {
		w.item(it, sc.FontSize, fonts)
	}

	// Obiekty dokumentu. Numery: 1 katalog, 2 drzewo stron, 3 strona, 4 treść, 5 informacje,
	// potem po pięć na każdy użyty krój.
	var objects [][]byte
	add := func(body string) int {
		objects = append(objects, []byte(body))
		return len(objects)
	}
	stream := func(dict string, content []byte) string {
		z := deflate(content)
		return fmt.Sprintf("<< %s /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream", dict, len(z), z)
	}
	add("<< /Type /Catalog /Pages 2 0 R >>")
	add("<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	add("") // strona: uzupełniana po krojach
	add(stream("", w.buf.Bytes()))
	info := "<< /Producer (gomermaidgo)"
	if sc.Title != "" {
		info += " /Title " + str(sc.Title)
	}
	if sc.Desc != "" {
		info += " /Subject " + str(sc.Desc)
	}
	add(info + " >>")

	var resources strings.Builder
	for _, f := range w.faces {
		if len(f.used) == 0 {
			continue
		}
		gids := make([]int, 0, len(f.used))
		usedSet := map[int]bool{}
		for g := range f.used {
			gids = append(gids, g)
			usedSet[g] = true
		}
		sort.Ints(gids)
		// Znacznik podzbioru: sześć wielkich liter wyliczonych z użytych glifów, zawsze te same.
		h := fnv.New32a()
		for _, g := range gids {
			fmt.Fprintf(h, "%d,", g)
		}
		tag, sum := make([]byte, 6), h.Sum32()
		for i := range tag {
			tag[i] = byte('A' + sum%26)
			sum /= 26
		}
		name := string(tag) + "+" + f.ttf.name
		scale := 1000 / float64(f.ttf.unitsPerEm)
		var widths, unicode strings.Builder
		for _, g := range gids {
			fmt.Fprintf(&widths, " %d [%s]", g, num(float64(f.ttf.advance(g))*scale))
		}
		for start := 0; start < len(gids); start += 100 {
			chunk := gids[start:min(start+100, len(gids))]
			fmt.Fprintf(&unicode, "%d beginbfchar\n", len(chunk))
			for _, g := range chunk {
				fmt.Fprintf(&unicode, "<%04X> <", g)
				for _, u := range utf16.Encode([]rune{f.used[g]}) {
					fmt.Fprintf(&unicode, "%04X", u)
				}
				unicode.WriteString(">\n")
			}
			unicode.WriteString("endbfchar\n")
		}
		cmap := "/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n" +
			"/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n" +
			"/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n" +
			unicode.String() + "endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n"
		subset := f.ttf.subset(usedSet)

		first := len(objects) + 1
		flags := 4 // symboliczny: znaki są wskazywane numerami glifów
		if f.ttf.italicAngle != 0 {
			flags |= 64
		}
		add(fmt.Sprintf("<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H /DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>", name, first+1, first+4))
		add(fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> "+
			"/FontDescriptor %d 0 R /DW 1000 /W [%s ] /CIDToGIDMap /Identity >>", name, first+2, widths.String()))
		add(fmt.Sprintf("<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%s %s %s %s] /ItalicAngle %s /Ascent %s /Descent %s /CapHeight %s /StemV 80 /FontFile2 %d 0 R >>",
			name, flags, num(float64(f.ttf.bbox[0])*scale), num(float64(f.ttf.bbox[1])*scale), num(float64(f.ttf.bbox[2])*scale), num(float64(f.ttf.bbox[3])*scale),
			num(f.ttf.italicAngle), num(float64(f.ttf.ascent)*scale), num(float64(f.ttf.descent)*scale), num(float64(f.ttf.capHeight)*scale), first+3))
		add(stream(fmt.Sprintf("/Length1 %d", len(subset)), subset))
		add(stream("", []byte(cmap)))
		fmt.Fprintf(&resources, " /F%d %d 0 R", f.index, first)
	}
	res := "<< /Font <<" + resources.String() + " >>"
	if len(w.opacity) > 0 {
		keys := make([]string, 0, len(w.opacity))
		for k := range w.opacity {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return w.opacity[keys[i]] < w.opacity[keys[j]] })
		res += " /ExtGState <<"
		for _, k := range keys {
			res += fmt.Sprintf(" /%s << /Type /ExtGState /ca %s >>", w.opacity[k], k)
		}
		res += " >>"
	}
	objects[2] = []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] /Resources %s >> /Contents 4 0 R >>", num(width*px), num(height*px), res))

	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", i+1)
		out.Write(body)
		out.WriteString("\nendobj\n")
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R /Info 5 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes(), nil
}
