package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// start uruchamia program z argumentami i zwraca wyjścia.
func start(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errs bytes.Buffer
	j, err := parse(args, &errs)
	if err != nil {
		return "", errs.String(), err
	}
	j.stdin, j.stdout, j.stderr = strings.NewReader(stdin), &out, &errs
	err = j.run()
	return out.String(), errs.String(), err
}

// REQ: PDF-005
// Rozszerzenie pliku wyniku wybiera format: .pdf daje dokument PDF, pozostałe SVG.
func TestOutputFormatByExtension(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "d.mmd")
	if err := os.WriteFile(in, []byte("flowchart TD\nA --> B"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"d.svg": "<svg ", "d.txt": "<svg "}
	if _, ok := formats[".pdf"]; ok {
		want["d.pdf"], want["D.PDF"] = "%PDF-", "%PDF-"
	} else {
		want["d.pdf"] = "<svg " // program zbudowany ze znacznikiem nopdf
	}
	for name, prefix := range want {
		out := filepath.Join(dir, name)
		if _, _, err := start(t, "", "-o", out, in); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		data, _ := os.ReadFile(out)
		if !bytes.HasPrefix(data, []byte(prefix)) {
			t.Errorf("%s starts with %q, want %q", name, data[:5], prefix)
		}
	}
}

// REQ: CLI-001
// Wejście i wyjście: argument albo -i, standardowe wejście i wyjście, plik obok wejścia dla -i.
func TestInputAndOutput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "d.mmd")
	os.WriteFile(in, []byte("flowchart TD\nA --> B"), 0o644)
	if out, _, err := start(t, "", in); err != nil || !strings.HasPrefix(out, "<svg ") {
		t.Errorf("argument without -o should write SVG to stdout: %v", err)
	}
	if out, _, err := start(t, "pie\n\"a\": 1"); err != nil || !strings.Contains(out, "pie-slice") {
		t.Errorf("stdin to stdout: %v", err)
	}
	if out, _, err := start(t, "", "--input", in); err != nil || out != "" {
		t.Errorf("-i without -o writes a file: %q %v", out, err)
	}
	if data, _ := os.ReadFile(in + ".svg"); !bytes.HasPrefix(data, []byte("<svg ")) {
		t.Errorf("-i d.mmd should write d.mmd.svg")
	}
	if _, ok := formats[".pdf"]; ok {
		if _, _, err := start(t, "", "-i", in, "-e", "pdf"); err != nil {
			t.Fatal(err)
		}
		if data, _ := os.ReadFile(in + ".pdf"); !bytes.HasPrefix(data, []byte("%PDF-")) {
			t.Errorf("-e pdf should write d.mmd.pdf")
		}
	}
	for _, args := range [][]string{{"-i", in, in}, {in, in}, {"-e", "gif", in}, {"-s", "-2", in}, {"-nieznana"}} {
		if _, _, err := start(t, "", args...); err == nil {
			t.Errorf("%v: want an error", args)
		}
	}
}

// REQ: CLI-002
// Flagi mermaid-cli: tło, tryb cichy, flagi bez odpowiednika przyjmowane z uwagą.
func TestCompatibilityFlags(t *testing.T) {
	src := "flowchart TD\nA --> B"
	out, errs, err := start(t, src, "-b", "#fff8e1", "--width", "800", "-H", "600", "-c", "c.json", "-C", "s.css", "-p", "p.json", "-f", "-I", "id")
	if err != nil || !strings.Contains(out, "#fff8e1") {
		t.Fatalf("background: %v", err)
	}
	for _, name := range []string{"width", "height", "configFile", "cssFile", "puppeteerConfigFile", "svgId"} {
		if !strings.Contains(errs, "--"+name+" is accepted") {
			t.Errorf("note about --%s is missing", name)
		}
	}
	out, errs, err = start(t, src, "--backgroundColor", "transparent", "--width", "5", "--quiet")
	if err != nil || errs != "" || strings.Contains(out, "transparent") {
		t.Errorf("quiet run: stderr %q, error %v", errs, err)
	}
}

// REQ: CLI-003
// Dokument Markdown: każdy blok do osobnego pliku, a w wyniku Markdown obrazki w miejscu bloków.
func TestMarkdownDocument(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "opis.md")
	doc := "# Tytuł\n\n```mermaid\nflowchart TD\nA --> B\n```\n\ntekst\n\n```go\nx := 1\n```\n\n~~~mermaid\npie\n\"a\": 1\n~~~\n"
	os.WriteFile(in, []byte(doc), 0o644)
	out := filepath.Join(dir, "wynik.md")
	_, errs, err := start(t, "", "-i", in, "-o", out)
	if err != nil || !strings.Contains(errs, "2 diagrams written") {
		t.Fatalf("%v %q", err, errs)
	}
	got, _ := os.ReadFile(out)
	want := "# Tytuł\n\n![diagram](./wynik-1.svg)\n\ntekst\n\n```go\nx := 1\n```\n\n![diagram](./wynik-2.svg)\n"
	if string(got) != want {
		t.Errorf("document:\n%s\nwant:\n%s", got, want)
	}
	for _, name := range []string{"wynik-1.svg", "wynik-2.svg"} {
		if data, _ := os.ReadFile(filepath.Join(dir, name)); !bytes.HasPrefix(data, []byte("<svg ")) {
			t.Errorf("%s is missing", name)
		}
	}
	// Wynik z rozszerzeniem obrazka: same pliki obrazków, bez dokumentu.
	if _, _, err := start(t, "", "-q", "-o", filepath.Join(dir, "rys.svg"), in); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "rys-2.svg")); err != nil {
		t.Errorf("rys-2.svg: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "rys.svg")); err == nil {
		t.Errorf("rys.svg should not be written for a Markdown input")
	}
	// Katalog na obrazki i odsyłacze względem dokumentu.
	if _, _, err := start(t, "", "-q", "-i", in, "-o", out, "-a", filepath.Join(dir, "img")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); !strings.Contains(string(got), "![diagram](./img/wynik-1.svg)") {
		t.Errorf("artefacts link: %s", got)
	}
	// Blok z błędem zostaje w dokumencie; błąd ma numer linii dokumentu.
	os.WriteFile(in, []byte("a\n\n```mermaid\nflowchart TD\nA -->\n```\n\n```mermaid\npie\n\"a\": 1\n```\n"), 0o644)
	_, errs, err = start(t, "", "-i", in, "-o", out)
	if err == nil || !strings.Contains(errs, "line 5") || !strings.Contains(err.Error(), "1 of 2") {
		t.Errorf("broken block: %v %q", err, errs)
	}
	if got, _ := os.ReadFile(out); !strings.Contains(string(got), "```mermaid\nflowchart TD\nA -->\n```") || !strings.Contains(string(got), "wynik-2.svg") {
		t.Errorf("broken block should stay, the other should be drawn: %s", got)
	}
	if _, _, err := start(t, "", in); err == nil {
		t.Errorf("Markdown without -o: want an error")
	}
}

// REQ: PNG-004
// Rozszerzenie .png albo -e png daje obraz; -s ustawia skalę. Program zbudowany ze znacznikiem
// nopng nie zna formatu.
func TestPNGOutput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "d.mmd")
	os.WriteFile(in, []byte("flowchart TD\nA --> B"), 0o644)
	if _, ok := formats[".png"]; !ok {
		if _, _, err := start(t, "", "-e", "png", in); err == nil {
			t.Errorf("a build without PNG should reject -e png")
		}
		return
	}
	size := func(args ...string) int {
		out := filepath.Join(dir, "d.png")
		if _, _, err := start(t, "", append(args, "-o", out, in)...); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(out)
		if !bytes.HasPrefix(data, []byte("\x89PNG\r\n")) {
			t.Fatalf("not a PNG file")
		}
		// Szerokość obrazu z nagłówka IHDR.
		return int(data[16])<<24 | int(data[17])<<16 | int(data[18])<<8 | int(data[19])
	}
	two, one, three := size(), size("-s", "1"), size("--scale", "3")
	if two != 2*one || three != 3*one {
		t.Errorf("widths at scales 1, 2, 3: %d %d %d", one, two, three)
	}
	if out, _, err := start(t, "", "-e", "png", in); err != nil || !strings.HasPrefix(out, "\x89PNG") {
		t.Errorf("-e png to stdout: %v", err)
	}
}

// REQ: THM-004
// Flaga -t wybiera motyw i ma pierwszeństwo przed źródłem; nazwa nieznana jest błędem.
func TestThemeFlag(t *testing.T) {
	src := "%%{init: {\"theme\": \"dark\"}}%%\nflowchart TD\nA --> B"
	fromSource, _, err := start(t, src)
	if err != nil || !strings.Contains(fromSource, "#14181d") {
		t.Fatalf("theme from the source: %v", err)
	}
	locked, _, err := start(t, src, "-t", "neutral")
	plain, _, _ := start(t, "\nflowchart TD\nA --> B")
	if err != nil || locked != plain {
		t.Errorf("-t neutral should override the source theme: %v", err)
	}
	if forest, _, err := start(t, src, "--theme", "forest"); err != nil || forest == plain || strings.Contains(forest, "#14181d") {
		t.Errorf("--theme forest: %v", err)
	}
	if _, _, err := start(t, src, "-t", "tęcza"); err == nil {
		t.Errorf("unknown theme: want an error")
	}
}
