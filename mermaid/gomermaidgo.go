// Package mermaid renders diagrams written in Mermaid syntax to SVG in pure Go:
// no JavaScript, no browser and no dependencies outside the standard library.
// The same text and the same options always give the same SVG, byte for byte.
//
// The supported range of syntax is described in docs/profile.md. An element outside that range
// gives an error with a line number, never an incomplete drawing.
package mermaid

import (
	"errors"
	"fmt"
	"github.com/polestorm/gomermaidgo/internal/fontdata"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/polestorm/gomermaidgo/fonts"
	"github.com/polestorm/gomermaidgo/internal/c4"
	"github.com/polestorm/gomermaidgo/internal/class"
	"github.com/polestorm/gomermaidgo/internal/er"
	"github.com/polestorm/gomermaidgo/internal/flowchart"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/gantt"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/journey"
	"github.com/polestorm/gomermaidgo/internal/layered"
	"github.com/polestorm/gomermaidgo/internal/mindmap"
	"github.com/polestorm/gomermaidgo/internal/pie"
	"github.com/polestorm/gomermaidgo/internal/quadrant"
	"github.com/polestorm/gomermaidgo/internal/requirement"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/sequence"
	"github.com/polestorm/gomermaidgo/internal/state"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/timeline"
	"github.com/polestorm/gomermaidgo/internal/usecase"
	"github.com/polestorm/gomermaidgo/internal/xychart"
)

// SyntaxError is an error in the diagram source, with a line number and a stable code.
type SyntaxError = syntax.Error

// ErrUnsupportedType means a diagram type outside the supported range.
var ErrUnsupportedType = errors.New("gomermaidgo: unsupported diagram type")

// Diagram types.
const (
	TypeSequence    = "sequenceDiagram"
	TypeFlowchart   = "flowchart"
	TypeState       = "stateDiagram"
	TypeGantt       = "gantt"
	TypeER          = "erDiagram"
	TypePie         = "pie"
	TypeClass       = "classDiagram"
	TypeTimeline    = "timeline"
	TypeJourney     = "journey"
	TypeMindmap     = "mindmap"
	TypeXYChart     = "xychart"
	TypeQuadrant    = "quadrantChart"
	TypeRequirement = "requirementDiagram"
	TypeC4          = "C4"
	TypeUseCase     = "usecase"
)

// SupportedTypes returns the supported diagram types.
func SupportedTypes() []string {
	out := []string{TypeSequence, TypeFlowchart, TypeState, TypeGantt, TypeER, TypePie, TypeClass, TypeTimeline, TypeJourney, TypeMindmap, TypeXYChart, TypeQuadrant, TypeRequirement, TypeC4, TypeUseCase}
	for _, t := range moreTypes {
		out = append(out, t.name)
	}
	return out
}

type config struct {
	wrap       wrapSettings // łamanie napisów z konfiguracji w źródle (wrapSettingsFrom)
	fontData   []byte
	boldData   []byte
	italicData []byte
	measure    func(text string, size float64) float64
	fontFamily string
	fontSize   float64
	background string
	colors     bool
	shadows    bool
	classes    bool
	emojiText  bool
	fitMode    string  // "", "auto", "on", "off"
	fitRatio   float64 // próg proporcji dla trybu auto; 0 oznacza 2
	foldMode   string  // "", "auto", "off"
	foldType   string  // "", "return", "snake"
	foldChains string  // "", "auto", "off"
	foldRanks  string  // "", "auto", "off"
	curved     bool
	theme      string
	themeSet   bool
}

// Option changes the rendering settings.
type Option func(*config)

// WithFont supplies a font file (TrueType or OpenType) from which the renderer reads character
// widths. The same typeface must later draw the text, so its name has to be given in WithFontFamily.
func WithFont(data []byte) Option { return func(c *config) { c.fontData = data } }

// WithStyledFonts supplies the files of the bold and italic styles of the typeface from WithFont;
// they are used to measure emphasis in Markdown texts. Either may be nil: bold text is then
// measured with the regular style plus a 6% allowance, italic text with the regular style.
func WithStyledFonts(bold, italic []byte) Option {
	return func(c *config) { c.boldData, c.italicData = bold, italic }
}

// WithMeasure supplies a custom text measuring function (width in pixels for a given font size);
// it takes precedence over WithFont.
func WithMeasure(f func(text string, size float64) float64) Option {
	return func(c *config) { c.measure = f }
}

// WithFontFamily sets the value of the font-family attribute in the SVG. The default is
// fonts.DefaultFamily: the typeface bundled with the module (package fonts), which the renderer then also
// uses to measure text. A caller who names another typeface should also supply its file (WithFont) or
// a measuring function (WithMeasure); without them a fallback measure is used, which overestimates.
func WithFontFamily(family string) Option { return func(c *config) { c.fontFamily = family } }

// WithFontSize sets the font size in pixels. The default is 16.
func WithFontSize(px float64) Option { return func(c *config) { c.fontSize = px } }

// WithBackground sets the background color. By default the background is transparent.
func WithBackground(color string) Option { return func(c *config) { c.background = color } }

// WithCurvedEdges routes graph edges as smooth curves. By default edges bend at right angles,
// with rounded corners.
func WithCurvedEdges() Option { return func(c *config) { c.curved = true } }

// WithColors turns the color mode on: sequence diagram participants get successive palette colors,
// and graph nodes get a color according to their role (step, decision, start and end). By default
// the drawing is neutral, because it then also looks good in black-and-white print.
func WithColors() Option { return func(c *config) { c.colors = true } }

// WithShadows adds a shadow under participants, nodes and notes. The shadow is a plain shape, not a filter.
func WithShadows() Option { return func(c *config) { c.shadows = true } }

// WithClasses adds to SVG elements a class attribute with their role (node, edge, edge-label, arrowhead,
// lane…), so that the page embedding the drawing can style it with its own CSS. Off by default: without
// classes the file is about one fifth smaller and looks the same (presentation attributes).
func WithClasses() Option { return func(c *config) { c.classes = true } }

// WithEmojiText keeps emoji (characters of the fallback font) in the SVG as text with the font name
// "Noto Emoji". By default they are written as outlines: the drawing then looks the same in every
// program and does not need the fallback font file, but the emoji itself cannot be copied or searched
// for (the words next to it remain text). This option does not affect PDF and PNG.
func WithEmojiText() Option { return func(c *config) { c.emojiText = true } }

// Fit-to-page modes (WithFit).
const (
	FitAuto = "auto" // default: fits only a drawing at least WithFitRatio times wider than tall
	FitOn   = "on"   // fits every drawing that does not fit on an A4 page at scale 1
	FitOff  = "off"  // no fitting
)

// WithFit selects the fit-to-page mode for diagrams drawn with the layered layout (flowchart, state,
// class, requirement, C4): the drawing is laid out again with narrower boxes, in which the text
// wraps more often, and the version that fits on an A4 page at the largest scale wins.
// The default is FitAuto. In the source the same is set by the line "%% layout: fit" (FitOn) or
// "%% layout: nofit" (FitOff); the option takes precedence over the source. An unknown mode is ignored.
func WithFit(mode string) Option { return func(c *config) { c.fitMode = mode } }

// WithFitRatio sets the threshold of the FitAuto mode: fitting starts when the width of the drawing
// is at least this many times greater than its height. The default is 2.
func WithFitRatio(ratio float64) Option { return func(c *config) { c.fitRatio = ratio } }

// Folding modes and folding types (WithFold, WithFoldType).
const (
	FoldAuto   = "auto"   // default mode: folds a simple diagram at least WithFitRatio times longer than wide
	FoldOff    = "off"    // no folding
	FoldReturn = "return" // default type: every row in the same direction, the connector returns to the start
	FoldSnake  = "snake"  // serpentine: every other row in the opposite direction
)

// WithFold selects the mode of folding a long, simple layered-layout diagram into several rows (LR)
// or columns (TB). The default is FoldAuto. In the source the same is set by the lines
// "%% layout: fold" (auto) and "%% layout: nofold" (off); the option takes precedence over the source.
func WithFold(mode string) Option { return func(c *config) { c.foldMode = mode } }

// WithFoldChains selects the mode of folding long chains of boxes inside a diagram (at least six
// boxes connected one to one) into several columns (TB) or rows (LR). The default is FoldOff;
// FoldAuto folds a chain if the drawing gains at least 20% of scale on an A4 page. The way of
// moving on to the next column is selected by WithFoldType. In the source: "%% layout: fold-chains"
// (auto) and "%% layout: nofold-chains" (off); the option takes precedence over the source.
func WithFoldChains(mode string) Option { return func(c *config) { c.foldChains = mode } }

// WithFoldRanks selects the mode of spreading out a long rank of leaves: when one node has at least
// six successors with no further connections (or as many predecessors with no other connections),
// they stand in two sub-ranks arranged in a zigzag, so an LR drawing is not a narrow
// column and a TB drawing is not a wide strip. The default is FoldOff; FoldAuto spreads the rank out
// if the drawing gains at least 20% of scale on an A4 page. In the source: "%% layout: fold-ranks"
// (auto) and "%% layout: nofold-ranks" (off); the option takes precedence over the source.
func WithFoldRanks(mode string) Option { return func(c *config) { c.foldRanks = mode } }

// WithFoldType selects the way of moving on to the next row: FoldReturn (the default) or FoldSnake.
// In the source: "%% layout: fold return" or "%% layout: fold snake".
func WithFoldType(kind string) Option { return func(c *config) { c.foldType = kind } }

// DetectType returns the diagram type from the first significant line of the source.
func DetectType(source string) (string, error) {
	source, _, _, err := preamble(source)
	if err != nil {
		return "", err
	}
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
		case "flowchart", "graph", "swimlane-beta", "swimlane":
			return TypeFlowchart, nil
		case "stateDiagram-v2", "stateDiagram":
			return TypeState, nil
		case "gantt":
			return TypeGantt, nil
		case "erDiagram":
			return TypeER, nil
		case "pie":
			return TypePie, nil
		case "classDiagram", "classDiagram-v2":
			return TypeClass, nil
		case "timeline":
			return TypeTimeline, nil
		case "journey":
			return TypeJourney, nil
		case "mindmap":
			return TypeMindmap, nil
		case "xychart-beta", "xychart":
			return TypeXYChart, nil
		case "quadrantChart":
			return TypeQuadrant, nil
		case "requirementDiagram":
			return TypeRequirement, nil
		case "C4Context", "C4Container", "C4Component", "C4Dynamic", "C4Deployment":
			return TypeC4, nil
		case "usecase-beta", "usecase":
			return TypeUseCase, nil
		}
		for _, t := range moreTypes {
			if slices.Contains(t.headers, strings.TrimRight(word, ";")) {
				return t.name, nil
			}
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
		case c < 0x20 && c != '\t' && c != '\r', c == 0xFFFE, c == 0xFFFF, c == graph.MarkBold, c == graph.MarkItalic:
			return &SyntaxError{Line: line, Code: "control-character", Message: fmt.Sprintf("control character U+%04X is not allowed", c)}
		}
		i += size
	}
	return nil
}

// Check checks the syntax of the source without drawing. It returns nil, *SyntaxError or ErrUnsupportedType.
func Check(source string) error {
	if err := checkCharacters(source); err != nil {
		return err
	}
	source, _, _, err := preamble(source)
	if err != nil {
		return err
	}
	kind, err := DetectType(source)
	if err != nil {
		return err
	}
	for _, t := range moreTypes {
		if t.name == kind {
			_, err = t.scene(source, layoutOptions{Measure: font.ApproxTextWidth, FontSize: 16})
			return err
		}
	}
	switch kind {
	case TypeFlowchart:
		_, err = flowchart.Parse(source)
	case TypeState:
		_, err = state.Parse(source)
	case TypeGantt:
		_, err = gantt.Parse(source)
	case TypeER:
		_, err = er.Parse(source)
	case TypePie:
		_, err = pie.Parse(source)
	case TypeClass:
		_, err = class.Parse(source)
	case TypeRequirement:
		_, err = requirement.Parse(source)
	case TypeC4:
		_, err = c4.Parse(source)
	case TypeUseCase:
		_, err = usecase.Parse(source)
	case TypeTimeline:
		_, err = timeline.Parse(source)
	case TypeJourney:
		_, err = journey.Parse(source)
	case TypeMindmap:
		_, err = mindmap.Parse(source)
	case TypeXYChart:
		_, err = xychart.Parse(source)
	case TypeQuadrant:
		_, err = quadrant.Parse(source)
	default:
		_, err = sequence.Parse(source)
	}
	return err
}

// fitDirective ustawia w źródle tryb dopasowania do strony (WithFit).
// wrapSettings to ustawienia łamania napisów zapisane w źródle, w bloku nagłówkowym albo dyrektywie
// %%{init}%%: markdownAutoWrap, wrap (globalnie albo w sequence) i flowchart.wrappingWidth.
type wrapSettings struct {
	off   bool    // markdownAutoWrap: false albo wrap: false
	on    bool    // wrap: true (sekwencja: łamanie tekstów komunikatów i notatek)
	width float64 // wrappingWidth; 0 = domyślne 200 px
}

var (
	autoWrapKey  = syntax.Lazy(`["']?\bmarkdownAutoWrap["']?\s*:\s*["']?(true|false)\b`)
	wrapKey      = syntax.Lazy(`["']?\bwrap["']?\s*:\s*["']?(true|false)\b`)
	wrapWidthKey = syntax.Lazy(`["']?\bwrappingWidth["']?\s*:\s*["']?(\d+(?:\.\d+)?)\b`)
)

// wrapSettingsFrom czyta ustawienia łamania z konfiguracji źródła. Szerokość spoza 50–2000 px
// jest pomijana: źródło jest danymi z zewnątrz.
func wrapSettingsFrom(settings string) wrapSettings {
	var w wrapSettings
	if m := autoWrapKey.FindStringSubmatch(settings); m != nil && m[1] == "false" {
		w.off = true
	}
	if m := wrapKey.FindStringSubmatch(settings); m != nil {
		w.on, w.off = m[1] == "true", w.off || m[1] == "false"
	}
	if m := wrapWidthKey.FindStringSubmatch(settings); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && v >= 50 && v <= 2000 {
			w.width = v
		}
	}
	return w
}

var fitDirective = syntax.Lazy(`(?m)^\s*%%\s*layout:\s*(fit|nofit)\s*$`)

// foldDirective ustawia w źródle zwijanie (WithFold, WithFoldType).
var foldDirective = syntax.Lazy(`(?m)^\s*%%\s*layout:\s*(fold|nofold)(?:\s+(return|snake))?\s*$`)

// foldMode rozstrzyga tryb zwijania: opcja, potem linia w źródle, domyślnie auto.
func foldMode(c config, source string) layered.FoldMode {
	mode := c.foldMode
	if mode == "" {
		if m := foldDirective.FindStringSubmatch(source); m != nil && m[1] == "nofold" {
			mode = FoldOff
		}
	}
	if mode == FoldOff {
		return layered.FoldOff
	}
	return layered.FoldAuto
}

// chainsDirective ustawia w źródle zwijanie łańcuchów (WithFoldChains).
var chainsDirective = syntax.Lazy(`(?m)^\s*%%\s*layout:\s*(fold-chains|nofold-chains)(?:\s+(return|snake))?\s*$`)

// foldChains rozstrzyga tryb zwijania łańcuchów: opcja, potem linia w źródle, domyślnie wyłączone.
func foldChains(c config, source string) bool {
	mode := c.foldChains
	if mode == "" {
		if m := chainsDirective.FindStringSubmatch(source); m != nil && m[1] == "fold-chains" {
			mode = FoldAuto
		}
	}
	return mode == FoldAuto
}

// ranksDirective ustawia w źródle rozkładanie warstw liści (WithFoldRanks).
var ranksDirective = syntax.Lazy(`(?m)^\s*%%\s*layout:\s*(fold-ranks|nofold-ranks)\s*$`)

// foldRanks rozstrzyga tryb rozkładania warstw: opcja, potem linia w źródle, domyślnie wyłączone.
func foldRanks(c config, source string) bool {
	mode := c.foldRanks
	if mode == "" {
		if m := ranksDirective.FindStringSubmatch(source); m != nil && m[1] == "fold-ranks" {
			mode = FoldAuto
		}
	}
	return mode == FoldAuto
}

// foldType rozstrzyga sposób zwijania: opcja, potem linia w źródle, domyślnie powrót.
func foldType(c config, source string) layered.FoldType {
	kind := c.foldType
	if kind == "" {
		if m := chainsDirective.FindStringSubmatch(source); m != nil && m[2] != "" {
			kind = m[2]
		}
		if m := foldDirective.FindStringSubmatch(source); m != nil && m[2] != "" {
			kind = m[2]
		}
	}
	if kind == FoldSnake {
		return layered.FoldSnake
	}
	return layered.FoldReturn
}

// fitMode rozstrzyga tryb dopasowania: opcja, potem linia w źródle, domyślnie auto.
func fitMode(c config, source string) layered.FitMode {
	mode := c.fitMode
	if mode == "" {
		if m := fitDirective.FindStringSubmatch(source); m != nil {
			mode = map[string]string{"fit": FitOn, "nofit": FitOff}[m[1]]
		}
	}
	switch mode {
	case FitOn:
		return layered.FitOn
	case FitOff:
		return layered.FitOff
	}
	return layered.FitAuto
}

// emojiFace to krój zapasowy z emoji; nil, gdyby pliku nie dało się odczytać.
var emojiFace = sync.OnceValue(func() *font.Face {
	face, err := font.Parse(fontdata.NotoEmoji)
	if err != nil {
		return nil
	}
	return face
})

// defaultFace to krój dołączony; plik jest czytany raz na proces.
var defaultFace = sync.OnceValue(func() *font.Face {
	face, err := font.Parse(fontdata.Arimo)
	if err != nil {
		return nil
	}
	face.SetFallback(emojiFace())
	return face
})

// build przechodzi drogę od źródła do sceny. Zwraca też krój użyty do pomiaru; nil, gdy
// mierzyła funkcja wywołującego albo miara zastępcza.
func build(source string, opts []Option) (*scene.Scene, *font.Face, error) {
	if err := checkCharacters(source); err != nil {
		return nil, nil, err
	}
	clean, title, settings, err := preamble(source)
	if err != nil {
		return nil, nil, err
	}
	var c config
	for _, o := range opts {
		o(&c)
	}
	th, err := resolveTheme(c, settings)
	if err != nil {
		return nil, nil, err
	}
	if th.colors {
		opts = append(append([]Option{}, opts...), WithColors())
	}
	if w := wrapSettingsFrom(settings); w != (wrapSettings{}) {
		opts = append(append([]Option{}, opts...), func(c *config) { c.wrap = w })
	}
	sc, face, err := buildScene(clean, opts)
	if err != nil {
		return nil, nil, err
	}
	// Tytuł z bloku nagłówkowego mierzy ten sam krój co resztę rysunku.
	measure := c.measure
	if measure == nil && face != nil {
		measure = func(text string, size float64) float64 {
			w, _ := face.TextWidth(text, size)
			return w
		}
	}
	addTitle(sc, title, measure)
	th.apply(sc)
	if face != nil && emojiFace() != nil {
		styled := defaultStyled()
		if c.fontData != nil {
			styled, _ = styledMeasure(c.boldData, c.italicData)
		}
		splitFallback(sc, face, emojiFace(), styled)
	}
	sc.Classes = c.classes
	return sc, face, nil
}

// buildScene buduje scenę ze źródła bez bloku nagłówkowego i dyrektyw.
func buildScene(source string, opts []Option) (*scene.Scene, *font.Face, error) {
	c := config{fontSize: 16}
	for _, o := range opts {
		o(&c)
	}
	if c.fontSize < 6 || c.fontSize > 96 {
		return nil, nil, fmt.Errorf("gomermaidgo: font size %v is outside 6–96 px", c.fontSize)
	}
	var face *font.Face
	switch {
	case c.measure != nil:
	case c.fontData != nil:
		var err error
		if face, err = font.Parse(c.fontData); err != nil {
			return nil, nil, fmt.Errorf("gomermaidgo: %w", err)
		}
		face.SetFallback(emojiFace())
	case c.fontFamily == "":
		face = defaultFace() // bez wskazania kroju mierzy krój dołączony
	}
	if c.fontFamily == "" {
		c.fontFamily = fonts.DefaultFamily
	}
	measure := c.measure
	if measure == nil && face != nil {
		measure = func(text string, size float64) float64 {
			w, _ := face.TextWidth(text, size)
			return w
		}
	}
	if measure == nil {
		measure = font.ApproxTextWidth
	}
	if err := checkCharacters(source); err != nil {
		return nil, nil, err
	}
	kind, err := DetectType(source)
	if err != nil {
		return nil, nil, err
	}
	for _, t := range moreTypes {
		if t.name == kind {
			sc, err := t.scene(source, layoutOptions{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows})
			return sc, face, err
		}
	}
	if kind == TypeGantt {
		d, err := gantt.Parse(source)
		if err != nil {
			return nil, nil, err
		}
		return gantt.Layout(d, gantt.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows}), face, nil
	}
	if kind == TypePie {
		chart, err := pie.Parse(source)
		if err != nil {
			return nil, nil, err
		}
		return pie.Layout(chart, pie.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows}), face, nil
	}
	if kind == TypeTimeline {
		d, err := timeline.Parse(source)
		if err != nil {
			return nil, nil, err
		}
		return timeline.Layout(d, timeline.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows}), face, nil
	}
	if kind == TypeJourney {
		d, err := journey.Parse(source)
		if err != nil {
			return nil, nil, err
		}
		return journey.Layout(d, journey.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows}), face, nil
	}
	if kind == TypeXYChart {
		chart, err := xychart.Parse(source)
		if err != nil {
			return nil, nil, err
		}
		return xychart.Layout(chart, xychart.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows}), face, nil
	}
	if kind == TypeQuadrant {
		chart, err := quadrant.Parse(source)
		if err != nil {
			return nil, nil, err
		}
		return quadrant.Layout(chart, quadrant.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows}), face, nil
	}
	if kind == TypeMindmap {
		g, err := mindmap.Parse(source)
		if err != nil {
			return nil, nil, err
		}
		return mindmap.Layout(g, mindmap.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows}), face, nil
	}
	if kind != TypeSequence {
		var g *graph.Graph
		switch kind {
		case TypeFlowchart:
			g, err = flowchart.Parse(source)
		case TypeER:
			g, err = er.Parse(source)
		case TypeClass:
			g, err = class.Parse(source)
		case TypeRequirement:
			g, err = requirement.Parse(source)
		case TypeC4:
			g, err = c4.Parse(source)
		case TypeUseCase:
			g, err = usecase.Parse(source)
		default:
			g, err = state.Parse(source)
		}
		if err != nil {
			return nil, nil, err
		}
		var styled func(string, float64, bool, bool) float64
		if face != nil && face == defaultFace() {
			styled = defaultStyled()
		} else if styled, err = styledMeasure(c.boldData, c.italicData); err != nil {
			return nil, nil, err
		}
		return layered.Layout(g, layered.Options{Measure: measure, MeasureStyled: styled, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows, CurvedEdges: c.curved, NoWrap: c.wrap.off, WrapWidth: c.wrap.width, Fit: fitMode(c, source), FitRatio: c.fitRatio, Fold: foldMode(c, source), FoldType: foldType(c, source), FoldChains: foldChains(c, source), FoldRanks: foldRanks(c, source)}), face, nil
	}
	d, err := sequence.Parse(source)
	if err != nil {
		return nil, nil, err
	}
	return sequence.Layout(d, sequence.Options{Measure: measure, FontSize: c.fontSize, FontFamily: c.fontFamily, Background: c.background, Colors: c.colors, Shadows: c.shadows,
		Wrap: c.wrap.on, WrapWidth: c.wrap.width}), face, nil
}

// defaultStyled mierzy odmianami kroju dołączonego; pliki są czytane raz na proces.
var defaultStyled = sync.OnceValue(func() func(string, float64, bool, bool) float64 {
	m, _ := styledMeasure(fontdata.ArimoBold, fontdata.ArimoItalic)
	return m
})

// styledMeasure buduje miarę wyróżnień z plików odmian kroju; nil, gdy nie podano żadnego.
func styledMeasure(bold, italic []byte) (func(string, float64, bool, bool) float64, error) {
	if bold == nil && italic == nil {
		return nil, nil
	}
	var faces [2]*font.Face
	for i, data := range [][]byte{bold, italic} {
		if data == nil {
			continue
		}
		f, err := font.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("gomermaidgo: %w", err)
		}
		f.SetFallback(emojiFace())
		faces[i] = f
	}
	return func(text string, size float64, isBold, isItalic bool) float64 {
		f := faces[1]
		if isBold || f == nil {
			f = faces[0]
		}
		if f == nil {
			f = faces[1]
		}
		w, _ := f.TextWidth(text, size)
		return w
	}, nil
}

// Render converts diagram source to an SVG document.
func Render(source string, opts ...Option) ([]byte, error) {
	sc, _, err := build(source, opts)
	if err != nil {
		return nil, err
	}
	var c config
	for _, o := range opts {
		o(&c)
	}
	if !c.emojiText {
		outlineFallback(sc)
	}
	return sc.SVG(), nil
}

// MissingGlyphs returns the characters of the drawing's texts that are missing from the font used
// for measuring (the bundled one or the one given in WithFont), in code point order. Such a character
// will be drawn with a substitute font or as an empty box, and its width is only approximate. When
// the function from WithMeasure or the fallback measure is used, the result is empty: the font is unknown.
func MissingGlyphs(source string, opts ...Option) ([]rune, error) {
	sc, face, err := build(source, opts)
	if err != nil || face == nil {
		return nil, err
	}
	seen := map[rune]bool{}
	var missing []rune
	for _, it := range sc.Items {
		if it.Kind != scene.Text {
			continue
		}
		for _, r := range it.Text {
			if _, ok := face.Advance(r); !ok && !seen[r] {
				seen[r] = true
				missing = append(missing, r)
			}
		}
	}
	slices.Sort(missing)
	return missing, nil
}
