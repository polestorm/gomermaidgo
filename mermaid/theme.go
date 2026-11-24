package mermaid

import (
	"fmt"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"math"
	"sort"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/csscolor"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Theme names. The look of each is this library's own; the names match the Mermaid themes, so that
// source written for Mermaid selects the counterpart.
const (
	ThemeNeutral = "neutral" // default: shades of gray, legible in black-and-white print
	ThemeDefault = "default" // colored: the same as WithColors
	ThemeForest  = "forest"  // green
	ThemeDark    = "dark"    // a light drawing on a dark background
	ThemeBase    = "base"    // neutral, meant to be tinted with variables (primaryColor)
)

// Themes returns the theme names in alphabetical order.
func Themes() []string {
	names := []string{ThemeNeutral, ThemeDefault, ThemeForest, ThemeDark, ThemeBase}
	sort.Strings(names)
	return names
}

// ErrUnknownTheme is returned by rendering when WithTheme was given a name not in Themes.
var ErrUnknownTheme = fmt.Errorf("gomermaidgo: unknown theme (known: %s)", strings.Join(Themes(), ", "))

// WithTheme selects the theme of the drawing. A theme given here takes precedence over the theme set
// in the diagram source (front matter, init directive) and turns its variables off: a document that is
// to look uniform sets the theme itself, even if only ThemeNeutral. Without this option the source may
// select the theme; an unknown name in the source means the default theme.
func WithTheme(name string) Option {
	return func(c *config) { c.theme, c.themeSet = name, true }
}

// theme to rozstrzygnięty motyw jednego rysunku.
type theme struct {
	name       string
	colors     bool    // tryb kolorowy układu
	tint       bool    // barwienie odcieni neutralnych
	hue, sat   float64 // barwa i najmniejsze nasycenie po zabarwieniu
	dark       bool
	background string // tło, gdy wywołujący nie podał własnego
	line, text string // kolor linii i tekstu ze zmiennych motywu
}

var (
	themeName = syntax.Lazy(`["']?\btheme["']?\s*:\s*["']?([A-Za-z]+)`)
	themeVar  = syntax.Lazy(`["']?\b(primaryColor|lineColor|primaryTextColor|textColor|background)["']?\s*:\s*["']?(#[0-9a-fA-F]{6}\b|#[0-9a-fA-F]{3}\b|[A-Za-z]+)`)
)

// resolveTheme wybiera motyw: z opcji, a bez niej ze źródła.
func resolveTheme(c config, settings string) (theme, error) {
	name := ThemeNeutral
	vars := map[string]string{}
	if c.themeSet {
		name = c.theme
	} else {
		if m := themeName.FindStringSubmatch(settings); m != nil {
			name = m[1]
		}
		for _, m := range themeVar.FindAllStringSubmatch(settings, -1) {
			vars[m[1]] = m[2]
		}
	}
	th := theme{name: name}
	switch name {
	case ThemeNeutral, ThemeBase:
	case ThemeDefault:
		th.colors = true
	case ThemeForest:
		th.tint, th.hue, th.sat = true, 128, 0.22
	case ThemeDark:
		th.dark, th.background = true, "#14181d"
	default:
		if c.themeSet {
			return theme{}, fmt.Errorf("%w: %q", ErrUnknownTheme, name)
		}
		th.name = ThemeNeutral // nazwa spoza listy w cudzym źródle nie jest błędem
	}
	if v := vars["primaryColor"]; v != "" {
		if r, g, b, ok := csscolor.Parse(v); ok {
			h, s, _ := hsl(r, g, b)
			if s > 0.02 {
				th.tint, th.hue, th.sat = true, h, math.Min(math.Max(s*0.5, 0.15), 0.45)
			}
		}
	}
	th.line = vars["lineColor"]
	th.text = vars["primaryTextColor"]
	if th.text == "" {
		th.text = vars["textColor"]
	}
	if v := vars["background"]; v != "" && v != "transparent" {
		th.background = v
	}
	return th, nil
}

// hsl zamienia składowe od 0 do 1 na barwę w stopniach, nasycenie i jasność.
func hsl(r, g, b float64) (h, s, l float64) {
	hi, lo := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (hi + lo) / 2
	d := hi - lo
	if d == 0 {
		return 0, 0, l
	}
	s = d / (1 - math.Abs(2*l-1))
	switch hi {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, l
}

// hex zamienia barwę, nasycenie i jasność na zapis #rrggbb.
func hex(h, s, l float64) string {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g = c, x
	case h < 120:
		r, g = x, c
	case h < 180:
		g, b = c, x
	case h < 240:
		g, b = x, c
	case h < 300:
		r, b = x, c
	default:
		r, b = c, x
	}
	q := func(v float64) int { return int(math.Round(math.Min(math.Max(v+m, 0), 1) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", q(r), q(g), q(b))
}

// luminance to jasność względna koloru według WCAG, do liczenia kontrastu.
func luminance(color string) float64 {
	r, g, b, _ := csscolor.Parse(color)
	lin := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// neutral mówi, czy kolor należy do odcieni neutralnych rysunku: szarości i stonowane błękity,
// z których składa się wygląd domyślny. Kolory palety i kolory nadane w źródle są nasycone.
// Nasycenie bardzo jasnych kolorów jest liczbowo wysokie przy znikomej różnicy składowych, więc
// dla nich liczy się sama różnica (chroma).
func neutral(h, s, chroma float64) bool {
	return s < 0.02 || ((s < 0.35 || chroma < 0.09) && h >= 195 && h <= 225)
}

// color przelicza jeden kolor rysunku na kolor motywu.
func (th theme) color(value string, line, text bool) string {
	r, g, b, ok := csscolor.Parse(value)
	if !ok {
		return value
	}
	h, s, l := hsl(r, g, b)
	if !neutral(h, s, math.Max(r, math.Max(g, b))-math.Min(r, math.Min(g, b))) {
		// Kolor nasycony zostaje; w motywie ciemnym jasne tła palety ciemnieją, żeby nie świeciły.
		if th.dark && l > 0.8 {
			return hex(h, s*0.6, 0.24)
		}
		return value
	}
	if l < 0.5 {
		if text && th.text != "" {
			return th.text
		}
		if line && th.line != "" {
			return th.line
		}
	}
	if th.tint && l > 0.02 && l < 0.995 {
		h, s = th.hue, math.Max(s, th.sat)
	}
	if th.dark {
		// Odwrócenie jasności zachowuje kontrasty: biały podkład staje się tłem, ciemny tekst jasnym.
		l = 0.09 + (1-l)*0.82
	}
	if !th.tint && !th.dark {
		return value
	}
	return hex(h, s, l)
}

// apply przemalowuje scenę na motyw, a potem poprawia tekst, który po zmianie kolorów (albo na
// kolorze nadanym w źródle) straciłby kontrast z tłem, na którym stoi.
func (th theme) apply(sc *scene.Scene) {
	if sc == nil {
		return
	}
	if sc.Background == "" {
		sc.Background = th.background
	}
	if !th.tint && !th.dark && th.line == "" && th.text == "" {
		return
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		if it.Role == "shadow" {
			continue
		}
		if it.Kind == scene.Text {
			it.Fill = th.color(it.Fill, false, true)
			continue
		}
		it.Fill = th.color(it.Fill, it.Kind == scene.Polygon || it.Kind == scene.Path && it.Stroke == "", false)
		it.Stroke = th.color(it.Stroke, true, false)
	}
	page := sc.Background
	if page == "" {
		page = "#ffffff"
	}
	// Kolejność malowania: warstwy, a w warstwie kolejność dodania.
	order := make([]int, len(sc.Items))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return sc.Items[order[a]].Layer < sc.Items[order[b]].Layer })
	light, dark := th.color("#1b1f24", false, false), th.color("#ffffff", false, false)
	for at, i := range order {
		it := &sc.Items[i]
		if it.Kind != scene.Text || it.Fill == "" {
			continue
		}
		b := it.Bounds()
		cx, cy := (b.X0+b.X1)/2, (b.Y0+b.Y1)/2
		under := page
		for _, k := range order[:at] {
			o := sc.Items[k]
			if o.Kind == scene.Text || o.Kind == scene.Line || o.Kind == scene.Polyline || o.Role == "shadow" || o.Fill == "" || o.Fill == "none" {
				continue
			}
			if o.FillOpacity > 0 && o.FillOpacity < 0.5 {
				continue
			}
			if ob := o.Bounds(); cx >= ob.X0 && cx <= ob.X1 && cy >= ob.Y0 && cy <= ob.Y1 {
				under = o.Fill
			}
		}
		if contrast(it.Fill, under) >= 3 {
			continue
		}
		if contrast(light, under) >= contrast(dark, under) {
			it.Fill = light
		} else {
			it.Fill = dark
		}
	}
}
