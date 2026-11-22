package gomermaidgo

import (
	"errors"

	"github.com/polestorm/gomermaidgo/fonts"
	"github.com/polestorm/gomermaidgo/internal/bridge"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// ErrFontRequired zwracają formaty, które osadzają krój w pliku (PDF), gdy rysunek był mierzony
// funkcją z WithMeasure albo krojem znanym tylko z nazwy: nie ma wtedy pliku do osadzenia.
var ErrFontRequired = errors.New("gomermaidgo: this output format embeds the font; use the bundled font or pass the font file with WithFont")

func init() {
	bridge.Build = func(source string, options []any) (*scene.Scene, bridge.Fonts, error) {
		opts := make([]Option, len(options))
		var c config
		for i, o := range options {
			opts[i] = o.(Option)
			opts[i](&c)
		}
		var f bridge.Fonts
		switch {
		case c.fontData != nil:
			f = bridge.Fonts{Regular: c.fontData, Bold: c.boldData, Italic: c.italicData}
		case c.measure == nil && c.fontFamily == "":
			f = bridge.Fonts{Regular: fonts.Arimo(), Bold: fonts.ArimoBold(), Italic: fonts.ArimoItalic()}
		default:
			return nil, f, ErrFontRequired
		}
		f.Fallback = fonts.NotoEmoji()
		if f.Bold == nil {
			f.Bold = f.Regular
		}
		if f.Italic == nil {
			f.Italic = f.Regular
		}
		sc, _, err := build(source, opts)
		return sc, f, err
	}
}
