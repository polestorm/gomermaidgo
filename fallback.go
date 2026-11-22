package gomermaidgo

import (
	"strings"

	"github.com/polestorm/gomermaidgo/fonts"
	"github.com/polestorm/gomermaidgo/internal/font"
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
