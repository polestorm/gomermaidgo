package gomermaidgo

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var paintAttr = regexp.MustCompile(`(fill|stroke)="(#[0-9a-f]{6})"`)

// REQ: THM-001
// Motyw neutralny nie zmienia rysunku, default jest trybem kolorowym, a każdy motyw rysuje każdy
// diagram korpusu z tymi samymi kształtami i tekstami.
func TestThemesOverCorpus(t *testing.T) {
	files, _ := filepath.Glob("testdata/corpus/*/*.mmd")
	shape := regexp.MustCompile(` (fill|stroke)="[^"]*"`)
	for _, path := range files {
		data, _ := os.ReadFile(path)
		src := string(data)
		plain, err := Render(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range Themes() {
			got, err := Render(src, WithTheme(name))
			if err != nil {
				t.Errorf("%s, %s: %v", path, name, err)
				continue
			}
			switch name {
			case ThemeNeutral, ThemeBase:
				if string(got) != string(plain) {
					t.Errorf("%s: theme %s should not change the drawing", path, name)
				}
			case ThemeDefault:
				if colored, _ := Render(src, WithColors()); string(got) != string(colored) {
					t.Errorf("%s: theme default should equal WithColors", path)
				}
			default:
				// Te same elementy, inne kolory.
				a := shape.ReplaceAllString(string(plain), "")
				b := strings.Replace(shape.ReplaceAllString(string(got), ""), "<rect x=\"", "<rect x=\"", 1)
				if name == ThemeDark {
					// Motyw ciemny dokłada tylko tło: te same prostokąty co rysunek z podanym tłem (podkłady
					// podpisów są wtedy malowane w kolorze tła zamiast wycinać linie).
					filled, _ := Render(src, WithBackground("#ffffff"))
					if strings.Count(string(got), "<rect") != strings.Count(string(filled), "<rect") {
						t.Errorf("%s: dark theme should add exactly the background", path)
					}
				} else if a != b {
					t.Errorf("%s: theme %s changed more than colours", path, name)
				}
			}
		}
	}
}

// REQ: THM-001 THM-004
func TestThemeColors(t *testing.T) {
	src := "flowchart TD\nA[Zwykły] --> B[Własny]\nstyle B fill:#fde8e8,stroke:#b42318"
	forest, err := Render(src, WithTheme(ThemeForest))
	if err != nil {
		t.Fatal(err)
	}
	green := 0
	for _, m := range paintAttr.FindAllStringSubmatch(string(forest), -1) {
		r, g, b, _ := parseHex(m[2])
		if g > r && g > b {
			green++
		}
		if b > g+4 {
			t.Errorf("forest theme left a blue tone: %s", m[2])
		}
	}
	if green < 4 || !strings.Contains(string(forest), "#fde8e8") || !strings.Contains(string(forest), "#b42318") {
		t.Errorf("forest: %d green paints; colours from the source should stay", green)
	}
	dark, _ := Render(src, WithTheme(ThemeDark))
	out := string(dark)
	if !strings.Contains(out, `fill="#14181d"`) || strings.Contains(out, "#fde8e8") || !strings.Contains(out, "#b42318") {
		t.Errorf("dark: background, darkened pale fill and kept stroke expected")
	}
	// Każdy napis ma kontrast z ciemnym tłem albo z polem, na którym stoi: żaden nie został ciemny.
	for _, m := range regexp.MustCompile(`<text[^>]* fill="(#[0-9a-f]{6})"`).FindAllStringSubmatch(out, -1) {
		if luminance(m[1]) < 0.3 {
			t.Errorf("dark theme left dark text %s", m[1])
		}
	}
	if own, _ := Render(src, WithTheme(ThemeDark), WithBackground("#101010")); !strings.Contains(string(own), `fill="#101010"`) || strings.Contains(string(own), "#14181d") {
		t.Errorf("the caller's background should win over the theme's")
	}
	// Jasne pole ze źródła w motywie ciemnym: tekst na nim musi zostać czytelny.
	bright, _ := Render("flowchart TD\nA[Tekst]\nstyle A fill:#ffff00", WithTheme(ThemeDark))
	m := regexp.MustCompile(`<text[^>]* fill="(#[0-9a-f]{6})"[^>]*>Tekst`).FindStringSubmatch(string(bright))
	if m == nil || contrast(m[1], "#ffff00") < 3 {
		t.Errorf("text on a bright fill from the source: %v", m)
	}
}

func parseHex(s string) (r, g, b int, ok bool) {
	var v int
	for _, c := range s[1:] {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= int(c - '0')
		default:
			v |= int(c-'a') + 10
		}
	}
	return v >> 16 & 255, v >> 8 & 255, v & 255, true
}

// REQ: THM-002 THM-003
func TestThemeFromSource(t *testing.T) {
	body := "flowchart TD\nA --> B"
	dark, _ := Render(body, WithTheme(ThemeDark))
	forest, _ := Render(body, WithTheme(ThemeForest))
	plain, _ := Render(body)
	for name, src := range map[string]string{
		"front matter": "---\nconfig:\n  theme: dark\n---\n" + body,
		"directive":    "%%{init: {'theme': 'dark'}}%%\n" + body,
		"json":         "%%{ init: { \"theme\": \"dark\", \"flowchart\": {} } }%%\n" + body,
	} {
		got, err := Render(src)
		if err != nil || !strings.Contains(string(got), `fill="#14181d"`) || strings.Count(string(got), "<path") != strings.Count(string(dark), "<path") {
			t.Errorf("%s: the source should select the dark theme (%v)", name, err)
		}
		if locked, _ := Render(src, WithTheme(ThemeForest)); strings.Contains(string(locked), "#14181d") || len(locked) != len(forest)+len(src)-len(src) {
			t.Errorf("%s: WithTheme should win over the source", name)
		}
	}
	if got, err := Render("%%{init: {\"theme\": \"tęcza\"}}%%\n" + body); err != nil {
		t.Errorf("unknown theme in the source should fall back: %v", err)
	} else if got2, _ := Render("%%{init: {\"theme\": \"nieznany\"}}%%\n" + body); string(got2) != string(plain) || len(got) == 0 {
		t.Errorf("unknown theme in the source should give the neutral drawing")
	}
	if _, err := Render(body, WithTheme("tęcza")); !errors.Is(err, ErrUnknownTheme) {
		t.Errorf("unknown theme in the option: %v", err)
	}
	// Zmienne: barwa z primaryColor, kolor linii, tekstu i tła.
	vars := "%%{init: {\"theme\": \"base\", \"themeVariables\": {\"primaryColor\": \"#ff8800\", \"lineColor\": \"#aa0000\", \"primaryTextColor\": \"#003366\", \"background\": \"#fffdf5\", \"fontSize\": \"30px\"}}}%%\n"
	got, err := Render(vars + body)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	for _, want := range []string{`stroke="#aa0000"`, `fill="#003366"`, `fill="#fffdf5"`} {
		if !strings.Contains(out, want) {
			t.Errorf("theme variables: %s is missing", want)
		}
	}
	warm := 0
	for _, m := range paintAttr.FindAllStringSubmatch(out, -1) {
		if r, g, b, _ := parseHex(m[2]); r > g && g > b {
			warm++
		}
	}
	if warm < 2 {
		t.Errorf("primaryColor should tint the neutral tones: %d warm paints", warm)
	}
	if ignored, _ := Render(vars+body, WithTheme(ThemeNeutral)); string(ignored) != string(plain) {
		t.Errorf("WithTheme should switch the variables of the source off")
	}
}
