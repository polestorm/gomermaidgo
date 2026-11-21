// Polecenie refcompare porównuje wynik gomermaidgo z rendererem odniesienia uruchamianym
// jako osobny program (czarna skrzynka). Nie porównuje pikseli ani geometrii, tylko treść:
// czy oba rysunki zawierają te same teksty tyle samo razy. Dla typów, które mają już parser,
// ale jeszcze nie mają układu, po naszej stronie porównywane są teksty z modelu.
//
//	go run ./tools/refcompare -ref 'mmdg -i {in} -o {out}' testdata/corpus/sequence/*.mmd
//
// {in} i {out} są zastępowane ścieżkami pliku źródłowego i pliku SVG. Renderer odniesienia
// nie jest zależnością modułu i nie jest z nim łączony.
package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/polestorm/gomermaidgo"
	"github.com/polestorm/gomermaidgo/internal/flowchart"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/state"
)

// modelTexts zwraca teksty z modelu grafu dla typów bez układu.
func modelTexts(source string) (map[string]int, error) {
	kind, err := gomermaidgo.DetectType(source)
	if err != nil {
		return nil, err
	}
	var g *graph.Graph
	if kind == gomermaidgo.TypeFlowchart {
		g, err = flowchart.Parse(source)
	} else {
		g, err = state.Parse(source)
	}
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, s := range g.Texts() {
		out[strings.Join(strings.Fields(s), " ")]++
	}
	return out, nil
}

// texts zwraca teksty rysunku: zawartość elementów text (każdy tspan osobno, jeśli są)
// oraz akapitów w foreignObject, po znormalizowaniu odstępów.
func texts(svg []byte) (map[string]int, error) {
	out := map[string]int{}
	dec := xml.NewDecoder(bytes.NewReader(svg))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity
	var stack []string
	var buf strings.Builder
	flush := func() {
		s := strings.Join(strings.Fields(buf.String()), " ")
		buf.Reset()
		if s != "" {
			out[s]++
		}
	}
	inside := func() bool {
		for _, n := range stack {
			if n == "text" || n == "foreignObject" {
				return true
			}
		}
		return false
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if inside() {
				flush()
			}
			stack = append(stack, el.Name.Local)
		case xml.EndElement:
			if inside() {
				flush()
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if inside() {
				buf.Write(el)
			}
		}
	}
}

func diff(ours, ref map[string]int) (missing, extra []string) {
	for s, n := range ref {
		if ours[s] < n {
			missing = append(missing, fmt.Sprintf("%q ×%d (ref %d)", s, ours[s], n))
		}
	}
	for s, n := range ours {
		if ref[s] < n {
			extra = append(extra, fmt.Sprintf("%q ×%d (ref %d)", s, n, ref[s]))
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return
}

func main() {
	ref := flag.String("ref", "", "reference renderer command with {in} and {out}")
	flag.Parse()
	if *ref == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: refcompare -ref 'cmd {in} {out}' file.mmd...")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "refcompare")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	same, differ, failed := 0, 0, 0
	fmt.Println("| Plik | Teksty u nas | Teksty w odniesieniu | Brakuje u nas | Nadmiar u nas |")
	fmt.Println("|---|---:|---:|---|---|")
	for _, in := range flag.Args() {
		name := filepath.Base(in)
		src, err := os.ReadFile(in)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var ours map[string]int
		oursSVG, err := gomermaidgo.Render(string(src))
		switch {
		case errors.Is(err, gomermaidgo.ErrLayoutPending):
			ours, err = modelTexts(string(src))
		case err == nil:
			ours, err = texts(oursSVG)
		}
		if err != nil {
			fmt.Printf("| %s | błąd: %v | | | |\n", name, err)
			failed++
			continue
		}
		out := filepath.Join(tmp, name+".svg")
		var args []string
		for _, a := range strings.Fields(*ref) {
			args = append(args, strings.NewReplacer("{in}", in, "{out}", out).Replace(a))
		}
		if msg, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			fmt.Printf("| %s | | błąd odniesienia: %v %s | | |\n", name, err, strings.TrimSpace(string(msg)))
			failed++
			continue
		}
		refSVG, err := os.ReadFile(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		theirs, err := texts(refSVG)
		if err != nil {
			fmt.Printf("| %s | | błąd XML odniesienia: %v | | |\n", name, err)
			failed++
			continue
		}
		missing, extra := diff(ours, theirs)
		total := func(m map[string]int) (n int) {
			for _, v := range m {
				n += v
			}
			return
		}
		if len(missing)+len(extra) == 0 {
			same++
		} else {
			differ++
		}
		fmt.Printf("| %s | %d | %d | %s | %s |\n", name, total(ours), total(theirs), strings.Join(missing, "; "), strings.Join(extra, "; "))
	}
	fmt.Printf("\nZgodne: %d, różne: %d, błędy: %d.\n", same, differ, failed)
}
