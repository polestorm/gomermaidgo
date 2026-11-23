// Package mermaid renderuje diagramy zapisane składnią Mermaid do SVG w czystym Go:
// bez JavaScriptu, bez przeglądarki i bez zależności poza biblioteką standardową.
// Ten sam tekst i te same opcje dają zawsze ten sam SVG co do bajta.
//
// Obsługiwany zakres składni opisuje docs/profile.md. Element spoza zakresu daje błąd
// z numerem linii, nigdy niepełny rysunek.
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
	"github.com/polestorm/gomermaidgo/internal/xychart"
)

// SyntaxError to błąd w źródle diagramu, z numerem linii i stałym kodem.
type SyntaxError = syntax.Error

// ErrUnsupportedType oznacza typ diagramu spoza obsługiwanego zakresu.
var ErrUnsupportedType = errors.New("gomermaidgo: unsupported diagram type")

// Typy diagramów.
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
)

// SupportedTypes zwraca obsługiwane typy diagramów.
func SupportedTypes() []string {
	out := []string{TypeSequence, TypeFlowchart, TypeState, TypeGantt, TypeER, TypePie, TypeClass, TypeTimeline, TypeJourney, TypeMindmap, TypeXYChart, TypeQuadrant, TypeRequirement, TypeC4}
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

// Option zmienia ustawienia renderowania.
type Option func(*config)

// WithFont podaje plik fontu (TrueType albo OpenType), z którego renderer czyta szerokości
// znaków. Ten sam krój musi potem narysować tekst, więc jego nazwę trzeba podać w WithFontFamily.
func WithFont(data []byte) Option { return func(c *config) { c.fontData = data } }

// WithStyledFonts podaje pliki odmiany pogrubionej i pochylonej kroju z WithFont; służą do
// pomiaru wyróżnień w tekstach Markdown. Każdy może być nil: pogrubienie jest wtedy mierzone
// krojem zwykłym z zapasem 6%, kursywa krojem zwykłym.
func WithStyledFonts(bold, italic []byte) Option {
	return func(c *config) { c.boldData, c.italicData = bold, italic }
}

// WithMeasure podaje własną funkcję pomiaru tekstu (szerokość w pikselach dla rozmiaru pisma);
// ma pierwszeństwo przed WithFont.
func WithMeasure(f func(text string, size float64) float64) Option {
	return func(c *config) { c.measure = f }
}

// WithFontFamily ustawia wartość atrybutu font-family w SVG. Domyślnie fonts.DefaultFamily:
// krój dołączony do modułu (pakiet fonts), którym renderer wtedy także mierzy tekst. Kto podaje
// inny krój, powinien podać też jego plik (WithFont) albo funkcję pomiaru (WithMeasure); bez
// nich działa miara zastępcza, która zawyża.
func WithFontFamily(family string) Option { return func(c *config) { c.fontFamily = family } }

// WithFontSize ustawia rozmiar pisma w pikselach. Domyślnie 16.
func WithFontSize(px float64) Option { return func(c *config) { c.fontSize = px } }

// WithBackground ustawia kolor tła. Domyślnie tło jest przezroczyste.
func WithBackground(color string) Option { return func(c *config) { c.background = color } }

// WithCurvedEdges prowadzi krawędzie grafów łagodnymi łukami. Domyślnie krawędzie są łamane pod
// kątem prostym, z zaokrąglonymi narożnikami.
func WithCurvedEdges() Option { return func(c *config) { c.curved = true } }

// WithColors włącza tryb kolorowy: uczestnicy diagramu sekwencji dostają kolejne kolory palety,
// a węzły grafów kolor według roli (krok, decyzja, początek i koniec). Domyślnie rysunek jest
// neutralny, bo dobrze wygląda także w druku czarno-białym.
func WithColors() Option { return func(c *config) { c.colors = true } }

// WithShadows dodaje cień pod uczestnikami, węzłami i notatkami. Cień jest zwykłym kształtem, nie filtrem.
func WithShadows() Option { return func(c *config) { c.shadows = true } }

// WithClasses dopisuje do elementów SVG atrybut class z ich rolą (node, edge, edge-label, arrowhead,
// lane…), żeby strona osadzająca rysunek mogła go stylować własnym CSS. Domyślnie wyłączone: bez
// klas plik jest o około piątą część mniejszy, a wygląd ten sam (atrybuty prezentacji).
func WithClasses() Option { return func(c *config) { c.classes = true } }

// WithEmojiText zostawia w SVG emoji (znaki kroju zapasowego) jako tekst z nazwą kroju „Noto Emoji”.
// Domyślnie są zapisane jako kontury: rysunek wygląda wtedy tak samo w każdym programie i nie
// potrzebuje pliku kroju zapasowego, ale samego emoji nie da się skopiować ani wyszukać (słowa
// obok zostają tekstem). PDF i PNG ta opcja nie dotyczy.
func WithEmojiText() Option { return func(c *config) { c.emojiText = true } }

// Tryby dopasowania do strony (WithFit).
const (
	FitAuto = "auto" // domyślny: dopasowanie tylko rysunku co najmniej WithFitRatio razy szerszego niż wysoki
	FitOn   = "on"   // dopasowanie każdego rysunku, który nie mieści się na stronie A4 w skali 1
	FitOff  = "off"  // bez dopasowania
)

// WithFit wybiera tryb dopasowania do strony dla diagramów rysowanych układem warstwowym (schemat
// blokowy, stany, klasy, wymagania, C4): rysunek jest układany jeszcze raz z węższymi polami, w których
// tekst łamie się częściej, i wygrywa wersja mieszcząca się na stronie A4 w największej skali.
// Domyślny jest FitAuto. W źródle to samo ustawia linia "%% layout: fit" (FitOn) albo
// "%% layout: nofit" (FitOff); opcja ma pierwszeństwo przed źródłem. Nieznany tryb jest pomijany.
func WithFit(mode string) Option { return func(c *config) { c.fitMode = mode } }

// WithFitRatio ustawia próg trybu FitAuto: dopasowanie rusza, gdy szerokość rysunku jest co najmniej
// tyle razy większa od wysokości. Domyślnie 2.
func WithFitRatio(ratio float64) Option { return func(c *config) { c.fitRatio = ratio } }

// Tryby i sposoby zwijania (WithFold, WithFoldType).
const (
	FoldAuto   = "auto"   // domyślny: zwija prosty diagram co najmniej WithFitRatio razy dłuższy niż szeroki
	FoldOff    = "off"    // bez zwijania
	FoldReturn = "return" // domyślny sposób: każdy rząd w tym samym kierunku, łącznik wraca do początku
	FoldSnake  = "snake"  // wężyk: co drugi rząd w przeciwnym kierunku
)

// WithFold wybiera tryb zwijania długiego, prostego diagramu układu warstwowego w kilka rzędów (LR)
// albo kolumn (TB). Domyślny jest FoldAuto. W źródle to samo ustawiają linie "%% layout: fold"
// (auto) i "%% layout: nofold" (off); opcja ma pierwszeństwo przed źródłem.
func WithFold(mode string) Option { return func(c *config) { c.foldMode = mode } }

// WithFoldChains wybiera tryb zwijania długich łańcuchów pól wewnątrz diagramu (co najmniej sześć
// pól połączonych jeden do jednego) w kilka kolumn (TB) albo rzędów (LR). Domyślny jest FoldOff;
// FoldAuto zwija łańcuch, jeśli rysunek zyskuje co najmniej 20% skali na stronie A4. Sposób
// przejścia do następnej kolumny wybiera WithFoldType. W źródle: "%% layout: fold-chains"
// (auto) i "%% layout: nofold-chains" (off); opcja ma pierwszeństwo przed źródłem.
func WithFoldChains(mode string) Option { return func(c *config) { c.foldChains = mode } }

// WithFoldRanks wybiera tryb rozkładania długiej warstwy liści: gdy jeden węzeł ma co najmniej
// sześć następników bez dalszych połączeń (albo tyle samo poprzedników bez innych połączeń), stoją
// one w dwóch podwarstwach ustawionych zygzakiem, więc rysunek LR nie jest wąskim
// słupem, a TB szerokim pasem. Domyślny jest FoldOff; FoldAuto rozkłada warstwę, jeśli rysunek
// zyskuje co najmniej 20% skali na stronie A4. W źródle: "%% layout: fold-ranks" (auto)
// i "%% layout: nofold-ranks" (off); opcja ma pierwszeństwo przed źródłem.
func WithFoldRanks(mode string) Option { return func(c *config) { c.foldRanks = mode } }

// WithFoldType wybiera sposób przejścia do następnego rzędu: FoldReturn (domyślny) albo FoldSnake.
// W źródle: "%% layout: fold return" albo "%% layout: fold snake".
func WithFoldType(kind string) Option { return func(c *config) { c.foldType = kind } }

// DetectType zwraca typ diagramu z pierwszej znaczącej linii źródła.
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

// Check sprawdza składnię źródła bez rysowania. Zwraca nil, *SyntaxError albo ErrUnsupportedType.
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

// Render zamienia źródło diagramu na dokument SVG.
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

// MissingGlyphs zwraca znaki tekstów rysunku, których nie ma w kroju użytym do pomiaru
// (dołączonym albo podanym w WithFont), w kolejności kodów. Taki znak zostanie narysowany
// krojem zastępczym albo pustym polem, a jego szerokość jest tylko przybliżona. Gdy mierzy
// funkcja z WithMeasure albo miara zastępcza, wynik jest pusty, bo krój nie jest znany.
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
