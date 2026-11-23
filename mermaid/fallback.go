package mermaid

import (
	"math"
	"strings"
	"sync"

	"github.com/polestorm/gomermaidgo/fonts"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/glyf"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// splitFallback dzieli napisy ze znakami kroju zapasowego na odcinki jednego kroju. Każdy odcinek
// jest osobnym napisem zaczepionym z lewej, w położeniu wynikającym z szerokości poprzednich,
// a odcinek kroju zapasowego niesie jego nazwę. Nazwa kroju zapasowego nie stoi więc na liście
// krojów całego rysunku: program rysujący SVG nie sięgnie po niego dla znaków, które krój główny
// ma (krój z emoji zawiera własne cyfry), a emoji dostaje krój, którym zostało zmierzone.
func splitFallback(sc *scene.Scene, face, fallback *font.Face, styled func(string, float64, bool, bool) float64) {
	uses := func(c rune) bool { return !face.Has(c) && !font.Ignorable(c) && fallback.Has(c) }
	var out []scene.Item
	changed := false
	for _, it := range sc.Items {
		split := false
		if it.Kind == scene.Text {
			for _, c := range it.Text {
				if uses(c) {
					split = true
					break
				}
			}
		}
		if !split {
			out = append(out, it)
			continue
		}
		changed = true
		size := sc.FontSize
		if it.FontSize > 0 {
			size = it.FontSize
		}
		width := func(text string) float64 {
			if (it.Bold || it.Italic) && styled != nil {
				return styled(text, size, it.Bold, it.Italic)
			}
			w, _ := face.TextWidth(text, size)
			return w
		}
		// Odcinki: kolejne znaki tego samego kroju; znak bez szerokości zostaje przy poprzednim.
		type run struct {
			text  string
			emoji bool
		}
		var runs []run
		for _, c := range it.Text {
			e := uses(c)
			if len(runs) > 0 && (runs[len(runs)-1].emoji == e || font.Ignorable(c)) {
				runs[len(runs)-1].text += string(c)
				continue
			}
			runs = append(runs, run{string(c), e})
		}
		total := 0.0
		widths := make([]float64, len(runs))
		for i, r := range runs {
			widths[i] = width(r.text)
			total += widths[i]
		}
		start := 0.0
		switch it.Anchor {
		case "middle":
			start = -total / 2
		case "end":
			start = -total
		}
		for i, r := range runs {
			// Odstęp na początku albo końcu osobnego napisu zniknąłby przy rysowaniu SVG (białe znaki
			// na brzegach są zwijane), więc zamiast niego przesuwamy początek odcinka.
			text := strings.TrimLeft(r.text, " ")
			lead := width(r.text[:len(r.text)-len(text)])
			text = strings.TrimRight(text, " ")
			if text != "" {
				part := it
				part.Text, part.Anchor, part.W = text, "start", width(text)
				if it.Vertical {
					part.Y = it.Y - start - lead // napis obrócony biegnie w górę od punktu zaczepienia
				} else {
					part.X = it.X + start + lead
				}
				if r.emoji {
					part.Family = fonts.EmojiFamily
				}
				out = append(out, part)
			}
			start += widths[i]
		}
	}
	if changed {
		sc.Items = out
	}
}

// emojiOutlines to kontury kroju zapasowego; nil, gdyby pliku nie dało się odczytać.
var emojiOutlines = sync.OnceValue(func() *glyf.Font {
	f, err := glyf.Parse(fonts.NotoEmoji())
	if err != nil {
		return nil
	}
	return f
})

// outlineFallback zamienia odcinki kroju zapasowego (emoji) na kontury: każdy staje się ścieżką
// wypełnioną kolorem tekstu, w tym samym miejscu i rozmiarze.
func outlineFallback(sc *scene.Scene) {
	outlines, cmap := emojiOutlines(), emojiFace()
	if outlines == nil || cmap == nil {
		return
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		if it.Kind != scene.Text || it.Family != fonts.EmojiFamily {
			continue
		}
		size := sc.FontSize
		if it.FontSize > 0 {
			size = it.FontSize
		}
		k := size / float64(outlines.UnitsPerEm)
		// Napis jest zaczepiony z lewej (splitFallback); obrócony biegnie w górę.
		pen := 0.0
		var segments []scene.Segment
		for _, c := range it.Text {
			if font.Ignorable(c) {
				continue
			}
			gid := int(cmap.GlyphID(c))
			at := func(q glyf.Point) scene.Point {
				x, y := it.X+pen+q.X*k, it.Y-q.Y*k
				if it.Vertical {
					x, y = it.X-q.Y*k, it.Y-pen-q.X*k
				}
				return scene.Point{X: round2(x), Y: round2(y)}
			}
			for _, path := range outlines.Paths(gid) {
				cur := path.Start
				segments = append(segments, scene.Segment{Cmd: 'M', To: at(cur)})
				for _, sg := range path.Segments {
					if !sg.Curve {
						segments = append(segments, scene.Segment{Cmd: 'L', To: at(sg.To)})
					} else {
						// Łuk drugiego stopnia zapisany dokładnie jako krzywa trzeciego stopnia.
						c1 := glyf.Point{X: cur.X + 2*(sg.Ctrl.X-cur.X)/3, Y: cur.Y + 2*(sg.Ctrl.Y-cur.Y)/3}
						c2 := glyf.Point{X: sg.To.X + 2*(sg.Ctrl.X-sg.To.X)/3, Y: sg.To.Y + 2*(sg.Ctrl.Y-sg.To.Y)/3}
						segments = append(segments, scene.Segment{Cmd: 'C', C1: at(c1), C2: at(c2), To: at(sg.To)})
					}
					cur = sg.To
				}
			}
			pen += float64(outlines.Advance(gid)) * k
		}
		fill := it.Fill
		*it = scene.Item{Kind: scene.Path, Layer: it.Layer, Role: it.Role + " emoji", Segments: segments, Fill: fill}
		if len(segments) == 0 {
			it.Kind, it.Text = scene.Text, "" // znak bez konturu: nic do narysowania
		}
	}
}

// round2 zaokrągla współrzędną konturu do dziesiątych części piksela: dokładność wystarcza, a plik jest krótszy.
func round2(v float64) float64 { return math.Round(v*10) / 10 }
