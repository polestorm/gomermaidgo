package mermaid

import (
	"os"
	"path/filepath"
	"testing"
)

// Źródła galerii (docs/gallery/src) są przykładami dla czytelników README: każde musi się rysować.
// Przykład japoński rysuje się też bez kroju CJK (z pustymi znakami), więc jest tu sprawdzany tak samo.
func TestGallerySources(t *testing.T) {
	var files []string
	for _, pattern := range []string{"../docs/gallery/src/*.mmd", "../docs/gallery/src/*/*.mmd"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	if len(files) < 30 {
		t.Fatalf("only %d gallery sources found", len(files))
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Render(string(src)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}
