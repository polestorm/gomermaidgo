// Polecenie wildcorpus pobiera zbiór publicznych diagramów, na którym README porównuje pokrycie rendererów.
// Diagramy są cudze, więc w repozytorium leży tylko manifest (testdata/wild/sources.tsv): repozytorium,
// przypięty commit, ścieżka pliku Markdown, numer linii otwierającej blok ```mermaid i suma SHA-256 treści.
//
//	go run ./tools/wildcorpus -o out/wild
//
// Każdy blok jest sprawdzany sumą; niezgodność albo brak pliku to błąd, a nie cichy pominięty plik.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// source to jeden wiersz manifestu.
type source struct {
	name, repo, commit, path string
	line                     int
	sum                      string
}

func main() {
	manifest := flag.String("manifest", "testdata/wild/sources.tsv", "manifest of public diagrams")
	out := flag.String("o", "out/wild", "directory for the .mmd files")
	flag.Parse()
	srcs, err := read(*manifest)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}
	client := &http.Client{Timeout: time.Minute}
	cache := map[string][]string{}
	bad := 0
	for _, s := range srcs {
		key := s.repo + "@" + s.commit + ":" + s.path
		lines, ok := cache[key]
		if !ok {
			if lines, err = fetch(client, s); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", s.name, err)
				bad++
				continue
			}
			cache[key] = lines
		}
		body, err := block(lines, s.line)
		if err == nil {
			h := sha256.Sum256([]byte(body))
			if got := hex.EncodeToString(h[:]); got != s.sum {
				err = fmt.Errorf("checksum %s, want %s", got, s.sum)
			}
		}
		if err == nil {
			err = os.WriteFile(filepath.Join(*out, s.name+".mmd"), []byte(body), 0o644)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", s.name, err)
			bad++
		}
	}
	fmt.Printf("%d of %d diagrams written to %s\n", len(srcs)-bad, len(srcs), *out)
	if bad > 0 {
		os.Exit(1)
	}
}

// read czyta manifest: kolumny oddzielone tabulatorem, linie z # na początku to komentarze.
func read(path string) ([]source, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var srcs []source
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		l := sc.Text()
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		c := strings.Split(l, "\t")
		if len(c) != 6 {
			return nil, fmt.Errorf("%s:%d: want 6 columns, got %d", path, n, len(c))
		}
		line, err := strconv.Atoi(c[4])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %v", path, n, err)
		}
		srcs = append(srcs, source{c[0], c[1], c[2], c[3], line, c[5]})
	}
	return srcs, sc.Err()
}

// fetch pobiera plik w przypiętej wersji z raw.githubusercontent.com.
func fetch(c *http.Client, s source) ([]string, error) {
	parts := strings.Split(s.path, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	u := "https://raw.githubusercontent.com/" + s.repo + "/" + s.commit + "/" + strings.Join(parts, "/")
	resp, err := c.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", u, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// tak jak przy zbieraniu: końce linii CRLF i CR liczą się jak LF
	return strings.Split(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(string(b)), "\n"), nil
}

// block wycina treść bloku, którego linia otwierająca (```mermaid albo ~~~mermaid) ma numer line (od 1),
// do linii zamykającej tym samym ogrodzeniem; puste linie na brzegach są usuwane, na końcu jest jeden \n.
func block(lines []string, line int) (string, error) {
	if line < 1 || line > len(lines) {
		return "", fmt.Errorf("line %d outside the file", line)
	}
	open := strings.TrimLeft(lines[line-1], " ")
	fence := ""
	for _, f := range []string{"```", "~~~"} {
		if strings.HasPrefix(open, f) {
			n := len(open) - len(strings.TrimLeft(open, f[:1]))
			fence = open[:n]
		}
	}
	if fence == "" || !strings.HasPrefix(strings.TrimSpace(open[len(fence):]), "mermaid") {
		return "", fmt.Errorf("line %d does not open a mermaid block", line)
	}
	j := line
	for j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), fence) {
		j++
	}
	return strings.Trim(strings.Join(lines[line:j], "\n"), "\n") + "\n", nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "wildcorpus:", err)
	os.Exit(1)
}
