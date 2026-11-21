// Package gomermaidgo renderuje diagramy zapisane składnią Mermaid do SVG w czystym Go:
// bez JavaScriptu, bez przeglądarki i bez zależności poza biblioteką standardową.
// Ten sam tekst i te same opcje dają zawsze ten sam SVG co do bajta.
//
// Obsługiwany zakres składni opisuje docs/profile.md. Element spoza zakresu daje błąd
// z numerem linii, nigdy niepełny rysunek.
package gomermaidgo

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/polestorm/gomermaidgo/fonts"
	"github.com/polestorm/gomermaidgo/internal/flowchart"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/layered"
	"github.com/polestorm/gomermaidgo/internal/sequence"
	"github.com/polestorm/gomermaidgo/internal/state"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// SyntaxError to błąd w źródle diagramu, z numerem linii i stałym kodem.
type SyntaxError = syntax.Error

// ErrUnsupportedType oznacza typ diagramu spoza obsługiwanego zakresu.
var ErrUnsupportedType = errors.New("gomermaidgo: unsupported diagram type")

// Typy diagramów.
const (
	TypeSequence  = "sequenceDiagram"
	TypeFlowchart = "flowchart"
	TypeState     = "stateDiagram"
)

// SupportedTypes zwraca obsługiwane typy diagramów.
func SupportedTypes() []string { return []string{TypeSequence, TypeFlowchart, TypeState} }

type config struct {
	fontData   []byte
	measure    func(text string, size float64) float64
	fontFamily string
	fontSize   float64
	background string
	colors     bool
	shadows    bool
}

// Option zmienia ustawienia renderowania.
type Option func(*config)

// WithFont podaje plik fontu (TrueType albo OpenType), z którego renderer czyta szerokości
// znaków. Ten sam krój musi potem narysować tekst, więc jego nazwę trzeba podać w WithFontFamily.
func WithFont(data []byte) Option { return func(c *config) { c.fontData = data } }

// WithMeasure podaje własną funkcję pomiaru tekstu (szerokość w pikselach dla rozmiaru pisma);
// ma pierwszeństwo przed WithFont.
func WithMeasure(f func(text string, size float64) float64) Option {
	return func(c *config) { c.measure = f }
}

// WithFontFamily ustawia wartość atrybutu font-family w SVG. Domyślnie "Recursive, sans-serif":
// krój dołączony do modułu (pakiet fonts), którym renderer wtedy także mierzy tekst. Kto podaje
// inny krój, powinien podać też jego plik (WithFont) albo funkcję pomiaru (WithMeasure); bez
// nich działa miara zastępcza, która zawyża.
func WithFontFamily(family string) Option { return func(c *config) { c.fontFamily = family } }

// WithFontSize ustawia rozmiar pisma w pikselach. Domyślnie 16.
func WithFontSize(px float64) Option { return func(c *config) { c.fontSize = px } }

// WithBackground ustawia kolor tła. Domyślnie tło jest przezroczyste.
func WithBackground(color string) Option { return func(c *config) { c.background = color } }

// WithColors włącza tryb kolorowy: uczestnicy diagramu sekwencji dostają kolejne kolory palety,
// a węzły grafów kolor według roli (krok, decyzja, początek i koniec). Domyślnie rysunek jest
// neutralny, bo dobrze wygląda także w druku czarno-białym.
func WithColors() Option { return func(c *config) { c.colors = true } }

// WithShadows dodaje cień pod uczestnikami, węzłami i notatkami. Cień jest zwykłym kształtem, nie filtrem.
func WithShadows() Option { return func(c *config) { c.shadows = true } }

// DetectType zwraca typ diagramu z pierwszej znaczącej linii źródła.
func DetectType(source string) (string, error) {
	for n, raw := range strings.Split(source, "\n") {
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		word := s
		if i := strings.IndexAny(s, " \t"); i >= 0 {
			word = s[:i]
		}
		switch strings.TrimRight(word, ";") {
		case "sequenceDiagram":
			return TypeSequence, nil
		case "flowchart", "graph":
			return TypeFlowchart, nil
		case "stateDiagram-v2", "stateDiagram":
			return TypeState, nil
		}
		return "", fmt.Errorf("%w: %q (line %d)", ErrUnsupportedType, word, n+1)
	}
	return "", &SyntaxError{Code: "empty-source", Message: "empty source"}
}

// checkCharacters odrzuca źródło, którego nie da się zapisać w XML 1.0: błędne UTF-8
// i znaki sterujące inne niż tabulacja i koniec linii.
func checkCharacters(source string) error {
	line := 1
	for i := 0; i < len(source); {
		c, size := utf8.DecodeRuneInString(source[i:])
		switch {
		case c == utf8.RuneError && size == 1:
			return &SyntaxError{Line: line, Code: "invalid-utf8", Message: "source is not valid UTF-8"}
		case c == '\n':
			line++
		case c < 0x20 && c != '\t' && c != '\r', c == 0xFFFE, c == 0xFFFF:
			return &SyntaxError{Line: line, Code: "control-character", Message: fmt.Sprintf("control character U+%04X is not allowed", c)}
		}
		i += size
	}
	return nil
}

// Check sprawdza składnię źródła bez rysowania. Zwraca nil, *SyntaxError albo ErrUnsupportedType.
func Check(source string) error {
	if err := checkCharacters(source); err != nil {
		return err
	}
	kind, err := DetectType(source)
	if err != nil {
		return err
	}
	switch kind {
	case TypeFlowchart:
		_, err = flowchart.Parse(source)
	case TypeState:
		_, err = state.Parse(source)
	default:
		_, err = sequence.Parse(source)
	}
	return err
}

// defaultMeasure mierzy krojem dołączonym; plik jest czytany raz na proces.
var defaultMeasure = sync.OnceValue(func() func(string, float64) float64 {
	face, err := font.Parse(fonts.Recursive())
	if err != nil {
		return font.ApproxTextWidth
	}
	return func(text string, size float64) float64 {
		w, _ := face.TextWidth(text, size)
		return w
	}
})

// Render zamienia źródło diagramu na dokument SVG.
func Render(source string, opts ...Option) ([]byte, error) {
	c := config{fontSize: 16}
	for _, o := range opts {
		o(&c)
	}
	if c.fontFamily == "" {
		// Bez wskazania kroju obowiązuje krój dołączony; jego plik służy też do pomiaru.
		c.fontFamily = fonts.RecursiveFamily + ", sans-serif"
		if c.fontData == nil && c.measure == nil {
			c.measure = defaultMeasure()
		}
	}
	if c.fontSize < 6 || c.fontSize > 96 {
		return nil, fmt.Errorf("gomermaidgo: font size %v is outside 6–96 px", c.fontSize)
	}
	measure := c.measure
	if measure == nil && c.fontData != nil {
		face, err := font.Parse(c.fontData)
		if err != nil {
			return nil, fmt.Errorf("gomermaidgo: %w", err)
		}
		measure = func(text string, size float64) float64 {
			w, _ := face.TextWidth(text, size)
			return w
		}
	}
	if measure == nil {
		measure = font.ApproxTextWidth
	}
	if err := checkCharacters(source); err != nil {
		return nil, err
	}
	kind, err := DetectType(source)
	if err != nil {
		return nil, err
	}
	if kind != TypeSequence {
		var g *graph.Graph
		if kind == TypeFlowchart {
			g, err = flowchart.Parse(source)
		} else {
			g, err = state.Parse(source)
		}
		if err != nil {
			return nil, err
		}
		sc := layered.Layout(g, layered.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows})
		return sc.SVG(), nil
	}
	d, err := sequence.Parse(source)
	if err != nil {
		return nil, err
	}
	sc := sequence.Layout(d, sequence.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows})
	return sc.SVG(), nil
}
