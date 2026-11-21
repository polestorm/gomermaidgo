// Polecenie gomermaidgo renderuje plik z diagramem do SVG.
//
//	gomermaidgo [-font plik.ttf] [-family nazwa] [-size 16] [-background kolor] [-colors] [-shadows] [-o wynik.svg] [wejście.mmd]
//
// Bez pliku wejściowego czyta standardowe wejście; bez -o pisze na standardowe wyjście.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/polestorm/gomermaidgo"
)

func main() {
	fontPath := flag.String("font", "", "font file (TTF/OTF) used to measure text")
	family := flag.String("family", "", "font-family written to the SVG")
	size := flag.Float64("size", 16, "font size in px")
	background := flag.String("background", "", "background colour (default: transparent)")
	out := flag.String("o", "", "output file (default: stdout)")
	colors := flag.Bool("colors", false, "colour participants and nodes")
	shadows := flag.Bool("shadows", false, "draw shadows under participants, nodes and notes")
	flag.Parse()

	var extra []gomermaidgo.Option
	if *colors {
		extra = append(extra, gomermaidgo.WithColors())
	}
	if *shadows {
		extra = append(extra, gomermaidgo.WithShadows())
	}
	if err := run(*fontPath, *family, *size, *background, *out, flag.Arg(0), extra); err != nil {
		fmt.Fprintln(os.Stderr, "gomermaidgo:", err)
		os.Exit(1)
	}
}

func run(fontPath, family string, size float64, background, out, in string, extra []gomermaidgo.Option) error {
	var source []byte
	var err error
	if in == "" || in == "-" {
		source, err = io.ReadAll(os.Stdin)
	} else {
		source, err = os.ReadFile(in)
	}
	if err != nil {
		return err
	}
	opts := append([]gomermaidgo.Option{gomermaidgo.WithFontSize(size)}, extra...)
	if fontPath != "" {
		data, err := os.ReadFile(fontPath)
		if err != nil {
			return err
		}
		opts = append(opts, gomermaidgo.WithFont(data))
	}
	if family != "" {
		opts = append(opts, gomermaidgo.WithFontFamily(family))
	}
	if background != "" {
		opts = append(opts, gomermaidgo.WithBackground(background))
	}
	svg, err := gomermaidgo.Render(string(source), opts...)
	if err != nil {
		return err
	}
	if missing, _ := gomermaidgo.MissingGlyphs(string(source), opts...); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "gomermaidgo: warning: the font has no glyphs for %q\n", string(missing))
	}
	if out == "" {
		_, err = os.Stdout.Write(svg)
		return err
	}
	return os.WriteFile(out, svg, 0o644)
}
