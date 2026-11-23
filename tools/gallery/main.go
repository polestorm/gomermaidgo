// Polecenie gallery rysuje obrazki galerii (docs/gallery/img) ze źródeł w docs/gallery/src:
// po jednym rysunku każdego typu, przykłady dopasowania do strony A4 w kilku wariantach opcji
// i przykłady pism. Obrazki są w SVG na białym tle (czytelne także w ciemnym motywie GitHuba);
// przykład japoński jest w PNG, bo potrzebuje kroju CJK, którego moduł nie dołącza.
//
//	go run ./tools/gallery -cjk NotoSansJP.ttf
//
// Bez -cjk przykład japoński zostaje bez zmian. Na wyjściu wypisuje rozmiary rysunków, z których
// korzystają opisy na stronach galerii.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/mermaid"
	"github.com/polestorm/gomermaidgo/png"
)

// shot to jeden obrazek: źródło, plik wynikowy i opcje.
type shot struct {
	src, out string
	opts     []mermaid.Option
}

func main() {
	root := flag.String("dir", "docs/gallery", "gallery directory with src/ and img/")
	cjk := flag.String("cjk", "", "TrueType font with Japanese glyphs for the Japanese example (optional)")
	flag.Parse()

	off := []mermaid.Option{mermaid.WithFit(mermaid.FitOff), mermaid.WithFold(mermaid.FoldOff),
		mermaid.WithFoldChains(mermaid.FoldOff), mermaid.WithFoldRanks(mermaid.FoldOff)}
	shots := []shot{
		{"fold.mmd", "layout/fold-off.svg", off},
		{"fold.mmd", "layout/fold.svg", nil},
		{"fold.mmd", "layout/fold-snake.svg", []mermaid.Option{mermaid.WithFoldType(mermaid.FoldSnake)}},
		{"fit.mmd", "layout/fit-off.svg", off},
		{"fit.mmd", "layout/fit-on.svg", []mermaid.Option{mermaid.WithFit(mermaid.FitOn), mermaid.WithFold(mermaid.FoldOff)}},
		{"fit.mmd", "layout/fit-fold.svg", nil},
		{"fold-chains.mmd", "layout/fold-chains-off.svg", off},
		{"fold-chains.mmd", "layout/fold-chains.svg", nil},
		{"fold-ranks.mmd", "layout/fold-ranks-off.svg", off},
		{"fold-ranks.mmd", "layout/fold-ranks.svg", nil},
		{"scripts/multilingual.mmd", "scripts/multilingual.svg", nil},
	}
	types, err := filepath.Glob(filepath.Join(*root, "src", "types", "*.mmd"))
	check(err)
	for _, f := range types {
		name := strings.TrimSuffix(filepath.Base(f), ".mmd")
		shots = append(shots, shot{"types/" + name + ".mmd", "types/" + name + ".svg", nil})
	}
	for _, s := range shots {
		source := read(filepath.Join(*root, "src", s.src))
		svg, err := mermaid.Render(source, append([]mermaid.Option{mermaid.WithBackground("white")}, s.opts...)...)
		check(err)
		write(filepath.Join(*root, "img", s.out), svg)
		fmt.Printf("%-32s %s\n", s.out, size(svg))
	}
	if *cjk != "" {
		font, err := os.ReadFile(*cjk)
		check(err)
		source := read(filepath.Join(*root, "src", "scripts", "japanese.mmd"))
		img, err := png.Render(source, mermaid.WithBackground("white"), mermaid.WithFont(font))
		check(err)
		write(filepath.Join(*root, "img", "scripts", "japanese.png"), img)
		fmt.Printf("%-32s %d bytes\n", "scripts/japanese.png", len(img))
	}
}

var viewBox = regexp.MustCompile(`viewBox="[-\d.]+ [-\d.]+ ([\d.]+) ([\d.]+)"`)

// size zwraca szerokość i wysokość rysunku z atrybutu viewBox.
func size(svg []byte) string {
	if m := viewBox.FindSubmatch(svg); m != nil {
		return string(m[1]) + " × " + string(m[2])
	}
	return "?"
}

func read(path string) string {
	b, err := os.ReadFile(path)
	check(err)
	return string(b)
}

func write(path string, data []byte) {
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	check(os.WriteFile(path, data, 0o644))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "gallery:", err)
		os.Exit(1)
	}
}
