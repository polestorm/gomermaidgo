// Polecenie pdfcheck sprawdza drogę do PDF: składa stronę HTML ze wszystkich diagramów korpusu,
// oddaje ją zewnętrznemu konwerterowi (czarna skrzynka, np. gopdfgo) i sprawdza, że powstał
// plik PDF, a po podaniu programu wyciągającego tekst także to, że każdy wyraz rysunków jest
// w PDF tekstem, tyle samo razy.
//
//	go run ./tools/pdfcheck -pdf 'gopdfgo -in {in} -out {out} -fonts {fonts}' -text 'pdftotext {in} -' testdata/corpus/*/*.mmd
//
// {in}, {out} i {fonts} są zastępowane ścieżkami strony HTML, pliku PDF i katalogu z krojem
// dołączonym. Konwerter nie jest zależnością modułu i nie jest z nim łączony.
package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/polestorm/gomermaidgo"
	"github.com/polestorm/gomermaidgo/fonts"
)

// words dopisuje do out wyrazy elementów text dokumentu SVG.
func words(svg []byte, out map[string]int) error {
	dec := xml.NewDecoder(bytes.NewReader(svg))
	inText := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			inText = el.Name.Local == "text"
		case xml.EndElement:
			inText = false
		case xml.CharData:
			if inText {
				for _, w := range strings.Fields(string(el)) {
					out[w]++
				}
			}
		}
	}
}

func command(template string, values map[string]string) *exec.Cmd {
	fields := strings.Fields(template)
	for i, f := range fields {
		for k, v := range values {
			f = strings.ReplaceAll(f, "{"+k+"}", v)
		}
		fields[i] = f
	}
	return exec.Command(fields[0], fields[1:]...)
}

func run() error {
	pdf := flag.String("pdf", "", "converter command with {in}, {out} and {fonts}")
	text := flag.String("text", "", "text extraction command with {in}, writing to standard output (optional)")
	keep := flag.String("keep", "", "directory to keep the page and the PDF in (optional)")
	flag.Parse()
	if *pdf == "" || flag.NArg() == 0 {
		return fmt.Errorf("usage: pdfcheck -pdf 'cmd {in} {out} {fonts}' [-text 'cmd {in}'] file.mmd...")
	}
	dir := *keep
	if dir == "" {
		tmp, err := os.MkdirTemp("", "pdfcheck")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		dir = tmp
	}
	fontDir := filepath.Join(dir, "fonts")
	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(fontDir, "Recursive-Regular.ttf"), fonts.Recursive(), 0o644); err != nil {
		return err
	}

	want := map[string]int{}
	var page strings.Builder
	// Diagram wyższy niż strona zostałby ucięty, bo rysunku nie da się podzielić między strony:
	// strona ogranicza też jego wysokość, a rysunek skaluje się z zachowaniem proporcji.
	page.WriteString(`<!doctype html><html lang="pl"><head><meta charset="utf-8"><title>gomermaidgo: korpus</title>` +
		`<style>body{font-family:"Recursive";font-size:10pt} h2{font-size:11pt;margin:12pt 0 4pt} h2{break-after:avoid} svg{display:block;max-width:100%;max-height:230mm;width:auto;height:auto}</style></head><body>` + "\n")
	for _, f := range flag.Args() {
		source, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		svg, err := gomermaidgo.Render(string(source))
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		if err := words(svg, want); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		title := filepath.Base(filepath.Dir(f)) + "/" + filepath.Base(f)
		for _, w := range strings.Fields(title) {
			want[w]++
		}
		fmt.Fprintf(&page, "<h2>%s</h2>\n%s", html.EscapeString(title), svg)
	}
	page.WriteString("</body></html>\n")
	in, out := filepath.Join(dir, "corpus.html"), filepath.Join(dir, "corpus.pdf")
	if err := os.WriteFile(in, []byte(page.String()), 0o644); err != nil {
		return err
	}
	if msg, err := command(*pdf, map[string]string{"in": in, "out": out, "fonts": fontDir}).CombinedOutput(); err != nil {
		return fmt.Errorf("converter failed: %w\n%s", err, msg)
	} else if len(bytes.TrimSpace(msg)) > 0 {
		fmt.Printf("converter output:\n%s\n", msg)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return err
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return fmt.Errorf("%s is not a PDF file", out)
	}
	fmt.Printf("%d diagrams, PDF %d kB\n", flag.NArg(), len(data)/1024)
	if *text == "" {
		return nil
	}
	extracted, err := command(*text, map[string]string{"in": out}).Output()
	if err != nil {
		return fmt.Errorf("text extraction failed: %w", err)
	}
	have := map[string]int{}
	for _, w := range strings.Fields(string(extracted)) {
		have[w]++
	}
	var lost []string
	for w, n := range want {
		if have[w] < n {
			lost = append(lost, fmt.Sprintf("%q ×%d (PDF %d)", w, n, have[w]))
		}
	}
	sort.Strings(lost)
	if len(lost) > 0 {
		return fmt.Errorf("%d words of the drawings are missing from the PDF text:\n%s", len(lost), strings.Join(lost, "\n"))
	}
	fmt.Printf("all %d distinct words of the drawings are text in the PDF\n", len(want))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
