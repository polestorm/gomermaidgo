package gomermaidgo

import (
	"math"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// preamble zdejmuje ze źródła to, co stoi przed diagramem albo obok niego i nie jest jego składnią:
// blok nagłówkowy YAML między liniami „---” oraz dyrektywy „%%{ … }%%”, także wieloliniowe.
// Zdjęte linie zostają puste, więc numery linii w komunikatach błędów się nie zmieniają.
//
// Z bloku nagłówkowego czytany jest tylko tytuł (klucz title na najwyższym poziomie); konfiguracja
// (config, motywy) jest przyjmowana i pomijana, tak jak zawartość dyrektyw: wygląd rysunku jest
// własny i nie zależy od ustawień Mermaid.
func preamble(source string) (clean, title string, err error) {
	// Znaki o zerowej szerokości (wklejane z edytorów i stron) nie należą do składni.
	source = strings.NewReplacer("\r\n", "\n", "\u200b", "", "\ufeff", "").Replace(source)
	lines := strings.Split(source, "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i < len(lines) && strings.TrimSpace(lines[i]) == "---" {
		start := i
		lines[i] = ""
		for i++; ; i++ {
			if i >= len(lines) {
				return "", "", &SyntaxError{Line: start + 1, Code: "frontmatter", Message: "front matter opened with \"---\" is not closed"}
			}
			text := lines[i]
			lines[i] = ""
			if strings.TrimSpace(text) == "---" {
				break
			}
			// Klucz na najwyższym poziomie: bez wcięcia.
			if key, value, ok := strings.Cut(text, ":"); ok && key == "title" {
				title = strings.TrimSpace(value)
				if len(title) >= 2 && (title[0] == '"' || title[0] == '\'') && title[len(title)-1] == title[0] {
					title = title[1 : len(title)-1]
				}
			}
		}
	}
	// Dyrektywa wieloliniowa: od linii z „%%{” do linii z „}%%”.
	for k := 0; k < len(lines); k++ {
		if !strings.HasPrefix(strings.TrimSpace(lines[k]), "%%{") {
			continue
		}
		start := k
		for ; k < len(lines); k++ {
			closed := strings.Contains(lines[k], "}%%")
			lines[k] = ""
			if closed {
				break
			}
		}
		if k >= len(lines) {
			return "", "", &SyntaxError{Line: start + 1, Code: "directive", Message: "directive opened with \"%%{\" is not closed with \"}%%\""}
		}
	}
	return strings.Join(lines, "\n"), title, nil
}

// addTitle dopisuje tytuł z bloku nagłówkowego nad rysunkiem, wyśrodkowany. Tytuł trafia też do
// elementu title, jeśli diagram nie ma własnego tytułu dostępności.
func addTitle(sc *scene.Scene, title string, measure func(string, float64) float64) {
	if title == "" || sc == nil {
		return
	}
	if measure == nil {
		measure = font.ApproxTextWidth
	}
	if sc.Title == "" {
		sc.Title = title
	}
	fs := sc.FontSize
	if fs <= 0 {
		fs = 16
	}
	lh := math.Round(fs * 1.25)
	b := sc.Bounds()
	w := measure(title, fs)
	sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "diagram-title", X: (b.X0 + b.X1) / 2, Y: b.Y0 - fs*0.9, W: w, H: lh,
		Text: title, Anchor: "middle", Bold: true, Fill: "#1b1f24"})
}
