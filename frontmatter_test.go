package gomermaidgo

import (
	"math"
	"regexp"
	"strconv"

	"errors"
	"github.com/polestorm/gomermaidgo/fonts"
	"strings"
	"testing"
)

// REQ: COM-001 COM-002 COM-003
func TestPreamble(t *testing.T) {
	src := "\n---\ntitle: \"Tytuł z nagłówka\"\nconfig:\n  theme: dark\n  title: nie ten\n---\n%%{init: {\n  \"theme\": \"forest\"\n}}%%\nflowchart TD\n  A --> B\n  %%{ init: { \"flowchart\": { \"curve\": \"linear\" } } }%%\n  B --> C\n"
	svg, err := Render(src)
	if err != nil {
		t.Fatal(err)
	}
	out := string(svg)
	if !strings.Contains(out, "<title>Tytuł z nagłówka</title>") || !strings.Contains(out, `class="diagram-title"`) || strings.Count(out, "Tytuł z nagłówka") != 2 {
		t.Errorf("the front matter title should be the accessible title and a text above the drawing")
	}
	if strings.Contains(out, "theme") || strings.Contains(out, "nie ten") || strings.Count(out, `class="edge"`) != 2 {
		t.Errorf("configuration and directives should leave no trace in the drawing")
	}
	if kind, err := DetectType(src); err != nil || kind != TypeFlowchart {
		t.Errorf("type behind front matter: %q %v", kind, err)
	}
	if err := Check(src); err != nil {
		t.Errorf("Check: %v", err)
	}
	// Tytuł dostępności ze źródła diagramu wygrywa z tytułem z nagłówka, który nadal stoi nad rysunkiem.
	svg, err = Render("---\ntitle: Z nagłówka\n---\nflowchart TD\naccTitle: Własny\nA --> B")
	if err != nil || !strings.Contains(string(svg), "<title>Własny</title>") || !strings.Contains(string(svg), ">Z nagłówka</text>") {
		t.Errorf("accTitle should stay the accessible title: %v", err)
	}
	// Tytuł działa dla każdego typu, także rysowanego bez układu grafu.
	for _, body := range []string{"pie\n\"a\": 1", "sequenceDiagram\nA->>B: x", "kanban\n  K\n    [karta]"} {
		svg, err := Render("---\ntitle: Wspólny\n---\n" + body)
		if err != nil || !strings.Contains(string(svg), ">Wspólny</text>") {
			t.Errorf("%q: title missing or error %v", body, err)
		}
	}
	// Numery linii w błędach liczą się od początku źródła.
	var se *SyntaxError
	_, err = Render("---\ntitle: T\n---\n%%{init: {}}%%\nflowchart TD\nA -->")
	if !errors.As(err, &se) || se.Line != 6 {
		t.Errorf("error after front matter should point at line 6: %v", err)
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"---\ntitle: T\nflowchart TD\nA --> B": {"frontmatter", 1},
		"flowchart TD\n%%{init: {\nA --> B":    {"directive", 2},
	} {
		_, err := Render(src)
		if !errors.As(err, &se) || se.Code != want.code || se.Line != want.line {
			t.Errorf("%q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: COM-005
// Emoji są mierzone krojem zapasowym i nie są zgłaszane jako brakujące; znak, którego nie ma
// żaden krój, nadal jest zgłaszany. Selektor wariantu nie zajmuje miejsca.
func TestEmojiFallback(t *testing.T) {
	width := func(src string) float64 {
		svg, err := Render(src)
		if err != nil {
			t.Fatal(err)
		}
		m := regexp.MustCompile(`<svg[^>]* width="([\d.]+)"`).FindSubmatch(svg)
		w, _ := strconv.ParseFloat(string(m[1]), 64)
		return w
	}
	plain, one, two := width("flowchart TD\nA[Nazwa pola abcd]"), width("flowchart TD\nA[🚀Nazwa pola abcd]"), width("flowchart TD\nA[🚀🚀Nazwa pola abcd]")
	// Krój zapasowy ma znaki szersze niż litery: każde emoji poszerza pole o tę samą, znaną wartość.
	if step := one - plain; step < 16 || step > 26 || math.Abs((two-one)-step) > 1.5 {
		t.Errorf("widths: %.1f, %.1f, %.1f", plain, one, two)
	}
	if with := width("flowchart TD\nA[⚠\ufe0fNazwa pola abcd]"); math.Abs(with-width("flowchart TD\nA[⚠Nazwa pola abcd]")) > 0.01 {
		t.Errorf("the variation selector should take no room")
	}
	// Domyślnie emoji jest konturem w kolorze tekstu, bez nazwy kroju zapasowego; słowo obok zostaje tekstem.
	outlined, _ := Render("flowchart TD\nA[🚀 Start]")
	o := string(outlined)
	if strings.Contains(o, "🚀") || strings.Contains(o, "Noto Emoji") || !regexp.MustCompile(`<path class="node-label emoji" d="M[^"]+" fill="#1b1f24"/>`).MatchString(o) || !strings.Contains(o, ">Start</text>") {
		t.Errorf("the emoji should be an outline: %s", o)
	}
	svg, _ := Render("flowchart TD\nA[🚀 Start]", WithEmojiText())
	// Z WithEmojiText nazwę kroju zapasowego niesie tylko odcinek z emoji; reszta napisu i rysunek zostają przy
	// kroju głównym, a odcinki stoją jeden za drugim.
	out := string(svg)
	m := regexp.MustCompile(`<text[^>]* x="([\d.]+)"[^>]* font-family="Noto Emoji"[^>]*>🚀</text>\s*<text[^>]* x="([\d.]+)"[^>]*>Start</text>`).FindStringSubmatch(out)
	if m == nil || !strings.Contains(out, `<svg xmlns="http://www.w3.org/2000/svg" version="1.1"`) || strings.Count(out, "Noto Emoji") != 1 || !strings.Contains(out, `font-family="Arimo, Arial, 'Liberation Sans', sans-serif"`) {
		t.Fatalf("the emoji should be its own text with the fallback family: %s", out)
	}
	x0, _ := strconv.ParseFloat(m[1], 64)
	x1, _ := strconv.ParseFloat(m[2], 64)
	if d := x1 - x0; d < 22 || d > 36 {
		t.Errorf("the second run should start an emoji and a space after the first: %.1f", d)
	}
	missing, err := MissingGlyphs("flowchart TD\nA[🚀 ✅ ⚠\ufe0f 漢]")
	if err != nil || string(missing) != "漢" {
		t.Errorf("missing glyphs: %q %v", string(missing), err)
	}
	// Krój podany przez wywołującego też dostaje krój zapasowy.
	if m, err := MissingGlyphs("flowchart TD\nA[🚀]", WithFont(fonts.ArimoBold()), WithFontFamily("X")); err != nil || len(m) != 0 {
		t.Errorf("caller's font: %q %v", string(m), err)
	}
}

// REQ: LAY-018
// Tryby dopasowania z opcji i z linii w źródle; opcja ma pierwszeństwo, auto jest domyślny.
func TestFitOptions(t *testing.T) {
	src := "flowchart LR\nA[Pierwszy długi krok procesu] --> B[Drugi długi krok procesu] --> C[Trzeci długi krok procesu] --> D[Czwarty długi krok procesu] --> E[Piąty długi krok procesu] --> F[Szósty długi krok procesu]"
	width := func(svg []byte) float64 {
		m := regexp.MustCompile(`<svg[^>]* width="([\d.]+)"`).FindSubmatch(svg)
		w, _ := strconv.ParseFloat(string(m[1]), 64)
		return w
	}
	def, _ := Render(src, WithFold(FoldOff))
	auto, _ := Render(src, WithFit(FitAuto), WithFold(FoldOff))
	off, _ := Render(src, WithFit(FitOff), WithFold(FoldOff))
	nofit, _ := Render("%% layout: nofit\n"+src, WithFold(FoldOff))
	forced, _ := Render("%% layout: nofit\n"+src, WithFit(FitOn), WithFold(FoldOff))
	high, _ := Render(src, WithFitRatio(100), WithFold(FoldOff))
	if string(def) != string(auto) || width(def) >= width(off)*0.85 {
		t.Errorf("default should be auto and narrower: %.0f vs off %.0f", width(def), width(off))
	}
	if string(nofit) != string(off) || width(forced) != width(def) || string(high) != string(off) {
		t.Errorf("directive, option precedence or ratio: nofit %.0f, forced %.0f, high ratio %.0f", width(nofit), width(forced), width(high))
	}
}
