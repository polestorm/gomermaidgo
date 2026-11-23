// Polecenie gomermaidgo renderuje diagram albo wszystkie diagramy dokumentu Markdown do SVG albo PDF.
//
//	gomermaidgo [flagi] [-o wynik] [wejście.mmd | wejście.md]
//
// Wejście: plik podany jako argument albo flagą -i; bez pliku (albo „-”) standardowe wejście.
// Wynik: plik z -o; bez -o SVG trafia na standardowe wyjście, a przy wejściu podanym flagą -i,
// tak jak w mermaid-cli, do pliku o nazwie wejścia z dopisanym rozszerzeniem formatu.
// Format wynika z flagi -e albo z rozszerzenia pliku wyniku: .svg (domyślnie), .pdf albo .png; program
// zbudowany ze znacznikami nopdf i nopng (go build -tags nopdf,nopng) nie zawiera tych formatów.
// Skalę obrazu PNG ustawia -s (domyślnie 2 piksele obrazu na piksel rysunku).
//
// Wejście Markdown (.md, .markdown): każdy blok „mermaid” trafia do osobnego pliku wynik-N.format.
// Gdy wynik też jest plikiem Markdown, dostaje treść dokumentu z blokami zamienionymi na obrazki.
//
// Flagi mermaid-cli (-i, -o, -e, -b, -t, -s, -q, -a i ich długie nazwy) działają tak samo; pozostałe
// (-w, -H, -c, -C, -p, -f, -I) są przyjmowane i pomijane z uwagą na wyjściu błędów.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/markdown"
	"github.com/polestorm/gomermaidgo/mermaid"
)

// formats to dodatkowe formaty wyniku według rozszerzenia pliku. Rejestrują je pliki kompilowane
// warunkowo (np. pdf.go), więc program zbudowany bez nich zna tylko SVG.
var formats = map[string]func(string, ...mermaid.Option) ([]byte, error){}

// scale to skala obrazu rastrowego z flagi -s; zero oznacza skalę domyślną formatu.
var scale float64

// job to jedno uruchomienie programu po rozbiorze flag.
type job struct {
	in, out   string
	fromFlag  bool   // wejście podane flagą -i: brak -o oznacza plik obok wejścia
	format    string // "svg", "pdf" albo pusty: według rozszerzenia wyniku
	artefacts string // katalog na obrazki dokumentu Markdown
	quiet     bool
	opts      []mermaid.Option
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

func main() {
	j, err := parse(os.Args[1:], os.Stderr)
	if err == nil {
		j.stdin, j.stdout, j.stderr = os.Stdin, os.Stdout, os.Stderr
		err = j.run()
	}
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gomermaidgo:", strings.TrimPrefix(err.Error(), "gomermaidgo: "))
		os.Exit(1)
	}
}

// parse rozbiera argumenty. Każda flaga mermaid-cli ma nazwę krótką i długą.
func parse(args []string, stderr io.Writer) (*job, error) {
	fs := flag.NewFlagSet("gomermaidgo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	j := &job{}
	var fontPath, family, background, input, themeName string
	var size float64
	var colors, shadows, curved, emojiText, classes bool
	var fitMode, foldMode, foldType, foldChains, foldRanks string
	var fitRatio float64
	both := func(p *string, short, long, usage string) {
		fs.StringVar(p, short, "", usage)
		fs.StringVar(p, long, "", "same as -"+short)
	}
	both(&input, "i", "input", "input file: a diagram or a Markdown document (\"-\" for stdin)")
	both(&j.out, "o", "output", "output file; the extension selects the format: .svg (default), .pdf, .png or .md for Markdown input")
	both(&j.format, "e", "outputFormat", "output format: svg, pdf or png (default: from the output extension)")
	both(&background, "b", "backgroundColor", "background colour (default: transparent)")
	both(&themeName, "t", "theme", "theme: "+strings.Join(mermaid.Themes(), ", ")+" (default: the theme named in the source, else neutral)")
	both(&j.artefacts, "a", "artefacts", "directory for the images of a Markdown document (default: next to the output)")
	fs.StringVar(&background, "background", "", "same as -b")
	fs.BoolVar(&j.quiet, "q", false, "no notes or warnings on stderr")
	fs.BoolVar(&j.quiet, "quiet", false, "same as -q")
	fs.StringVar(&fontPath, "font", "", "font file (TTF/OTF) used to measure text")
	fs.StringVar(&family, "family", "", "font-family written to the SVG")
	fs.Float64Var(&size, "size", 16, "font size in px")
	fs.BoolVar(&colors, "colors", false, "colour participants and nodes")
	fs.BoolVar(&shadows, "shadows", false, "draw shadows under participants, nodes and notes")
	fs.BoolVar(&classes, "classes", false, "add class attributes with element roles to the SVG, for styling with CSS")
	fs.BoolVar(&curved, "curved", false, "draw graph edges as smooth curves instead of right angles with rounded corners")
	fs.BoolVar(&emojiText, "emoji-text", false, "keep emoji in SVG as text set in the Noto Emoji font instead of outlines")
	// Bez flagi decyduje źródło diagramu (linie %% layout: …), a bez niego wartość domyślna.
	fs.StringVar(&fitMode, "fit", "", "fit graph diagrams to an A4 page by narrowing their fields: auto (default; only drawings at least -fit-ratio times wider than tall), on or off")
	fs.Float64Var(&fitRatio, "fit-ratio", 0, "width-to-height ratio from which -fit auto and -fold auto act (default 2)")
	fs.StringVar(&foldMode, "fold", "", "fold a long, simple graph diagram into rows (LR) or columns (TB): auto (default) or off")
	fs.StringVar(&foldRanks, "fold-ranks", "", "spread a long rank of leaves of one node into staggered sub-ranks in a graph diagram: off (default) or auto")
	fs.StringVar(&foldChains, "fold-chains", "", "fold long chains of nodes inside a graph diagram into columns (TB) or rows (LR): off (default) or auto")
	fs.StringVar(&foldType, "fold-type", "", "how a folded diagram continues: return (default; every row in the same direction) or snake (every other row reversed)")
	// Flagi mermaid-cli bez odpowiednika: wygląd jest własny, a wynik wektorowy.
	ignored := map[string]*string{}
	for _, names := range [][2]string{{"w", "width"}, {"H", "height"}, {"c", "configFile"},
		{"C", "cssFile"}, {"p", "puppeteerConfigFile"}, {"I", "svgId"}} {
		v := new(string)
		ignored[names[1]] = v
		fs.StringVar(v, names[0], "", "accepted for mermaid-cli compatibility and ignored")
		fs.StringVar(v, names[1], "", "same as -"+names[0])
	}
	scale = 0
	fs.Float64Var(&scale, "s", 0, "scale of a PNG image: pixels per drawing pixel (default 2)")
	fs.Float64Var(&scale, "scale", 0, "same as -s")
	var fit bool
	fs.BoolVar(&fit, "f", false, "accepted for mermaid-cli compatibility; the PDF page always fits the drawing")
	fs.BoolVar(&fit, "pdfFit", false, "same as -f")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	switch {
	case input != "" && fs.NArg() > 0:
		return nil, fmt.Errorf("input given twice: -i %s and %s", input, fs.Arg(0))
	case fs.NArg() > 1:
		return nil, fmt.Errorf("more than one input file: %s", strings.Join(fs.Args(), " "))
	case input != "":
		j.in, j.fromFlag = input, true
	default:
		j.in = fs.Arg(0)
	}
	if scale < 0 || scale > 16 {
		return nil, fmt.Errorf("scale %v is outside the range from 0 to 16", scale)
	}
	j.format = strings.ToLower(j.format)
	if j.format != "" && j.format != "svg" {
		if _, ok := formats["."+j.format]; !ok {
			return nil, fmt.Errorf("output format %q is not supported (svg%s)", j.format, known())
		}
	}
	if !j.quiet {
		for _, name := range []string{"width", "height", "configFile", "cssFile", "puppeteerConfigFile", "svgId"} {
			if *ignored[name] != "" {
				fmt.Fprintf(stderr, "gomermaidgo: note: --%s is accepted for compatibility and ignored\n", name)
			}
		}
	}
	j.opts = []mermaid.Option{mermaid.WithFontSize(size)}
	if colors {
		j.opts = append(j.opts, mermaid.WithColors())
	}
	switch foldMode {
	case "":
	case mermaid.FoldAuto, mermaid.FoldOff:
		j.opts = append(j.opts, mermaid.WithFold(foldMode))
	default:
		return nil, fmt.Errorf("-fold must be auto or off, not %q", foldMode)
	}
	switch foldChains {
	case "":
	case mermaid.FoldAuto, mermaid.FoldOff:
		j.opts = append(j.opts, mermaid.WithFoldChains(foldChains))
	default:
		return nil, fmt.Errorf("-fold-chains must be auto or off, not %q", foldChains)
	}
	switch foldRanks {
	case "":
	case mermaid.FoldAuto, mermaid.FoldOff:
		j.opts = append(j.opts, mermaid.WithFoldRanks(foldRanks))
	default:
		return nil, fmt.Errorf("-fold-ranks must be auto or off, not %q", foldRanks)
	}
	switch foldType {
	case "":
	case mermaid.FoldReturn, mermaid.FoldSnake:
		j.opts = append(j.opts, mermaid.WithFoldType(foldType))
	default:
		return nil, fmt.Errorf("-fold-type must be return or snake, not %q", foldType)
	}
	if fitRatio < 0 {
		return nil, fmt.Errorf("-fit-ratio must not be negative")
	}
	if fitRatio > 0 {
		j.opts = append(j.opts, mermaid.WithFitRatio(fitRatio))
	}
	switch fitMode {
	case "":
	case mermaid.FitAuto, mermaid.FitOn, mermaid.FitOff:
		j.opts = append(j.opts, mermaid.WithFit(fitMode))
	default:
		return nil, fmt.Errorf("-fit must be auto, on or off, not %q", fitMode)
	}
	if emojiText {
		j.opts = append(j.opts, mermaid.WithEmojiText())
	}
	if curved {
		j.opts = append(j.opts, mermaid.WithCurvedEdges())
	}
	if shadows {
		j.opts = append(j.opts, mermaid.WithShadows())
	}
	if classes {
		j.opts = append(j.opts, mermaid.WithClasses())
	}
	if themeName != "" {
		j.opts = append(j.opts, mermaid.WithTheme(themeName))
	}
	if fontPath != "" {
		data, err := os.ReadFile(fontPath)
		if err != nil {
			return nil, err
		}
		j.opts = append(j.opts, mermaid.WithFont(data))
	}
	if family != "" {
		j.opts = append(j.opts, mermaid.WithFontFamily(family))
	}
	if background != "" && background != "transparent" {
		j.opts = append(j.opts, mermaid.WithBackground(background))
	}
	return j, nil
}

// known wylicza formaty dodatkowe wbudowane w program, do komunikatu błędu.
func known() string {
	s := ""
	for _, ext := range []string{".pdf", ".png"} {
		if _, ok := formats[ext]; ok {
			s += ", " + ext[1:]
		}
	}
	return s
}

// isMarkdown mówi, czy nazwa pliku wskazuje na dokument Markdown.
func isMarkdown(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".md" || ext == ".markdown"
}

// renderer zwraca funkcję rysującą i rozszerzenie dla wyniku o podanej nazwie.
func (j *job) renderer(out string) (func(string, ...mermaid.Option) ([]byte, error), string) {
	ext := strings.ToLower(filepath.Ext(out))
	if j.format != "" {
		ext = "." + j.format
	}
	if alt, ok := formats[ext]; ok {
		return alt, ext
	}
	return mermaid.Render, ".svg"
}

func (j *job) run() error {
	var source []byte
	var err error
	if j.in == "" || j.in == "-" {
		source, err = io.ReadAll(j.stdin)
	} else {
		source, err = os.ReadFile(j.in)
	}
	if err != nil {
		return err
	}
	out := j.out
	if out == "" && j.fromFlag && j.in != "-" {
		// Jak w mermaid-cli: wynik obok wejścia, z dopisanym rozszerzeniem formatu.
		_, ext := j.renderer("")
		out = j.in + ext
	}
	if isMarkdown(j.in) {
		return j.document(string(source), out)
	}
	render, _ := j.renderer(out)
	data, err := render(string(source), j.opts...)
	if err != nil {
		return err
	}
	j.warn(string(source), "")
	if out == "" || out == "-" {
		_, err = j.stdout.Write(data)
		return err
	}
	return os.WriteFile(out, data, 0o644)
}

// warn ostrzega o znakach, których nie ma w kroju.
func (j *job) warn(source, where string) {
	if j.quiet {
		return
	}
	if missing, _ := mermaid.MissingGlyphs(source, j.opts...); len(missing) > 0 {
		fmt.Fprintf(j.stderr, "gomermaidgo: warning: %sthe font has no glyphs for %q\n", where, string(missing))
	}
}

// document rysuje wszystkie diagramy dokumentu Markdown do plików wynik-N.format. Gdy wynik jest
// plikiem Markdown, zapisuje też dokument z blokami zamienionymi na obrazki. Blok z błędem
// zostaje w dokumencie bez zmian, a program kończy się błędem po przejściu całego dokumentu.
func (j *job) document(source, out string) error {
	if out == "" || out == "-" {
		return errors.New("a Markdown input needs an output file (-o)")
	}
	toMarkdown := isMarkdown(out)
	stem := strings.TrimSuffix(out, filepath.Ext(out))
	render, ext := j.renderer(out)
	if toMarkdown {
		render, ext = j.renderer("")
	}
	dir := filepath.Dir(out)
	if j.artefacts != "" {
		if err := os.MkdirAll(j.artefacts, 0o755); err != nil {
			return err
		}
		stem = filepath.Join(j.artefacts, filepath.Base(stem))
	}
	count := 0
	result, problems := markdown.Replace(source, func(b markdown.Block) ([]string, error) {
		data, err := render(b.Source, j.opts...)
		if err != nil {
			return nil, err
		}
		j.warn(b.Source, "diagram "+strconv.Itoa(b.Index)+": ")
		name := stem + "-" + strconv.Itoa(b.Index) + ext
		if err := os.WriteFile(name, data, 0o644); err != nil {
			return nil, err
		}
		count++
		link, err := filepath.Rel(dir, name)
		if err != nil {
			link = name
		}
		return []string{"![diagram](./" + filepath.ToSlash(link) + ")"}, nil
	})
	if toMarkdown {
		if err := os.WriteFile(out, []byte(result), 0o644); err != nil {
			return err
		}
	}
	if !j.quiet {
		fmt.Fprintf(j.stderr, "gomermaidgo: %d diagrams written\n", count)
	}
	for _, p := range problems {
		fmt.Fprintf(j.stderr, "gomermaidgo: %s: %v\n", j.in, p)
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d of %d diagrams could not be drawn", len(problems), count+len(problems))
	}
	return nil
}
