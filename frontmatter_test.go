package gomermaidgo

import (
	"errors"
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
