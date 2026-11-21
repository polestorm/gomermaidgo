package gomermaidgo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/er"
	"github.com/polestorm/gomermaidgo/internal/flowchart"
	"github.com/polestorm/gomermaidgo/internal/gantt"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/sequence"
	"github.com/polestorm/gomermaidgo/internal/state"
)

var update = flag.Bool("update", false, "rewrite golden SVG files")

// sequenceFiles zwraca pliki korpusu diagramów sekwencji.
func sequenceFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("testdata/corpus/sequence/*.mmd")
	if err != nil || len(files) == 0 {
		t.Fatalf("no corpus files: %v", err)
	}
	return files
}

// corpusFiles zwraca wszystkie pliki korpusu.
func corpusFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("testdata/corpus/*/*.mmd")
	if err != nil || len(files) == 0 {
		t.Fatalf("no corpus files: %v", err)
	}
	return files
}

// graphCorpusFiles zwraca pliki korpusu typów grafowych.
func graphCorpusFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, dir := range []string{"flowchart", "state", "er"} {
		m, _ := filepath.Glob("testdata/corpus/" + dir + "/*.mmd")
		if len(m) == 0 {
			t.Fatalf("no corpus files for %s", dir)
		}
		files = append(files, m...)
	}
	return files
}

func parseGraph(t *testing.T, path string) *graph.Graph {
	t.Helper()
	src := readFile(t, path)
	var g *graph.Graph
	var err error
	if strings.Contains(path, "/flowchart/") {
		g, err = flowchart.Parse(src)
	} else if strings.Contains(path, "/er/") {
		g, err = er.Parse(src)
	} else {
		g, err = state.Parse(src)
	}
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return g
}

// REQ: GANTT-001
// Model wykresu Gantta z każdego pliku korpusu zgadza się z plikiem wzorcowym.
func TestGanttCorpusGolden(t *testing.T) {
	files, _ := filepath.Glob("testdata/corpus/gantt/*.mmd")
	if len(files) == 0 {
		t.Fatal("no corpus files for gantt")
	}
	for _, f := range files {
		d, err := gantt.Parse(readFile(t, f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		got := d.Dump()
		golden := strings.TrimSuffix(goldenPath(f), ".svg") + ".txt"
		if *update {
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if want, err := os.ReadFile(golden); err != nil || string(want) != got {
			t.Errorf("%s: model differs from %s (run go test . -update and review the diff): %v", f, golden, err)
		}
		// Każda nazwa zadania, sekcji i tytuł są na rysunku.
		svg, err := Render(readFile(t, f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		have := map[string]bool{}
		for _, s := range checkSVG(t, f, svg) {
			have[s] = true
		}
		want := append([]string{d.Title}, d.Sections...)
		for _, task := range d.Tasks {
			want = append(want, task.Name)
		}
		for _, s := range want {
			if s != "" && !have[s] {
				t.Errorf("%s: text %q is missing from the SVG", f, s)
			}
		}
	}
}

// REQ: FLOW-001 STATE-001
// Model grafu z każdego pliku korpusu zgadza się z plikiem wzorcowym.
func TestGraphCorpusGolden(t *testing.T) {
	for _, f := range graphCorpusFiles(t) {
		got := parseGraph(t, f).Dump()
		golden := strings.TrimSuffix(goldenPath(f), ".svg") + ".txt"
		if *update {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Errorf("%s: missing golden file (run go test . -update): %v", f, err)
			continue
		}
		if got != string(want) {
			t.Errorf("%s: model differs from %s\n--- got\n%s", f, golden, got)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func goldenPath(corpus string) string {
	rel := strings.TrimPrefix(corpus, "testdata/corpus/")
	return filepath.Join("testdata/golden", strings.TrimSuffix(rel, ".mmd")+".svg")
}

// REQ: SEQ-012
func TestCorpusGolden(t *testing.T) {
	for _, f := range corpusFiles(t) {
		got, err := Render(readFile(t, f))
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		golden := goldenPath(f)
		if *update {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Errorf("%s: missing golden file (run go test -update): %v", f, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: output differs from %s (run go test -update and review the diff)", f, golden)
		}
	}
}

// REQ: SEQ-012
func TestCorpusDeterministic(t *testing.T) {
	for _, f := range corpusFiles(t) {
		src := readFile(t, f)
		a, err := Render(src)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for i := 0; i < 3; i++ {
			b, _ := Render(src)
			if !bytes.Equal(a, b) {
				t.Fatalf("%s: two renders differ", f)
			}
		}
	}
}

// Dozwolone elementy i atrybuty: podzbiór SVG 1.1, który rysuje każdy renderer.
var allowedSVG = map[string][]string{
	"svg":      {"xmlns", "version", "width", "height", "viewBox", "role", "font-family", "font-size"},
	"title":    {},
	"desc":     {},
	"rect":     {"class", "x", "y", "width", "height", "rx", "fill", "fill-opacity", "stroke", "stroke-width", "stroke-dasharray"},
	"line":     {"class", "x1", "y1", "x2", "y2", "fill", "stroke", "stroke-width", "stroke-dasharray"},
	"polyline": {"class", "points", "fill", "stroke", "stroke-width", "stroke-dasharray"},
	"polygon":  {"class", "points", "fill", "fill-opacity", "stroke", "stroke-width", "stroke-dasharray"},
	"circle":   {"class", "cx", "cy", "r", "fill", "fill-opacity", "stroke", "stroke-width", "stroke-dasharray"},
	"ellipse":  {"class", "cx", "cy", "rx", "ry", "fill", "fill-opacity", "stroke", "stroke-width", "stroke-dasharray"},
	"path":     {"class", "d", "fill", "stroke", "stroke-width", "stroke-dasharray"},
	"text":     {"class", "x", "y", "text-anchor", "font-size", "fill", "font-weight", "font-style"},
}

// pathData dopuszcza tylko polecenia M, L, C z liczbami: bez łuków i poleceń względnych.
var pathSyntax = regexp.MustCompile(`^M-?\d+(\.\d+)? -?\d+(\.\d+)?( [LC]-?\d+(\.\d+)?( -?\d+(\.\d+)?)+)+$`)

var numberList = regexp.MustCompile(`^-?\d+(\.\d+)?([ ,]-?\d+(\.\d+)?)*$`)

// checkSVG sprawdza, że dokument jest poprawnym XML i używa tylko dozwolonego podzbioru SVG.
func checkSVG(t *testing.T, name string, svg []byte) (texts []string) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(svg))
	depth, inText := 0, false
	var current strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%s: not well-formed XML: %v", name, err)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 && (el.Name.Local != "svg" || el.Name.Space != "http://www.w3.org/2000/svg") {
				t.Errorf("%s: root is %q in namespace %q", name, el.Name.Local, el.Name.Space)
			}
			allowed, ok := allowedSVG[el.Name.Local]
			if !ok {
				t.Errorf("%s: element <%s> is not allowed", name, el.Name.Local)
				continue
			}
			for _, a := range el.Attr {
				found := false
				for _, n := range allowed {
					if n == a.Name.Local {
						found = true
					}
				}
				if !found {
					t.Errorf("%s: attribute %s on <%s> is not allowed", name, a.Name.Local, el.Name.Local)
				}
				if a.Name.Local == "d" && !pathSyntax.MatchString(a.Value) {
					t.Errorf("%s: path data %q uses more than M, L, C with numbers", name, a.Value)
				}
				switch a.Name.Local {
				case "rx", "ry", "x", "y", "width", "height", "x1", "y1", "x2", "y2", "cx", "cy", "r", "points", "viewBox", "stroke-width", "fill-opacity", "font-size", "stroke-dasharray":
					if !numberList.MatchString(a.Value) {
						t.Errorf("%s: attribute %s=%q is not a plain number list", name, a.Name.Local, a.Value)
					}
				}
			}
			if el.Name.Local == "text" {
				inText = true
				current.Reset()
			}
		case xml.EndElement:
			depth--
			if el.Name.Local == "text" {
				inText = false
				texts = append(texts, current.String())
			}
		case xml.CharData:
			if inText {
				current.Write(el)
			}
		case xml.ProcInst, xml.Directive:
			t.Errorf("%s: processing instructions and directives are not allowed", name)
		}
	}
	return texts
}

// REQ: SEQ-001 SEQ-012
func TestCorpusValidSVG(t *testing.T) {
	for _, f := range corpusFiles(t) {
		svg, err := Render(readFile(t, f), WithBackground("#ffffff"))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		checkSVG(t, f, svg)
		// Tryb kolorowy z cieniami zmienia tylko barwy i dokłada kształty cienia: te same teksty,
		// ten sam dozwolony podzbiór SVG.
		plain, _ := Render(readFile(t, f))
		styled, err := Render(readFile(t, f), WithColors(), WithShadows())
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if a, b := checkSVG(t, f, plain), checkSVG(t, f+" (colors)", styled); strings.Join(a, "|") != strings.Join(b, "|") {
			t.Errorf("%s: colours and shadows changed the texts", f)
		}
		if bytes.Count(styled, []byte(`class="shadow"`)) == 0 {
			t.Errorf("%s: no shadows were drawn", f)
		}
	}
}

// REQ: SEQ-002 SEQ-003 SEQ-004 SEQ-006 SEQ-010
// Każdy tekst ze źródła (etykiety uczestników, komunikatów, notatek, tytuły bloków) jest w SVG.
func TestCorpusLabelsPresent(t *testing.T) {
	for _, f := range sequenceFiles(t) {
		src := readFile(t, f)
		d, err := sequence.Parse(src)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		svg, err := Render(src)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		have := map[string]int{}
		for _, s := range checkSVG(t, f, svg) {
			have[s]++
		}
		need := func(s string) {
			if s == "" {
				return
			}
			if have[s] == 0 {
				t.Errorf("%s: text %q is missing from the SVG", f, s)
			}
		}
		need(d.Title)
		for _, p := range d.Participants {
			for _, line := range p.Label {
				want := 2
				if p.Destroyed {
					want = 1 // uczestnik, który zakończył życie, nie ma stopki
				}
				if have[line] < want {
					t.Errorf("%s: participant label %q should appear %d times", f, line, want)
				}
			}
		}
		for _, e := range d.Events {
			switch e.Kind {
			case sequence.EventMessage, sequence.EventNote:
				for _, line := range e.Text {
					need(line)
				}
			case sequence.EventBlockStart, sequence.EventBlockSection:
				if len(e.Text) > 0 {
					need("[" + strings.Join(e.Text, " ") + "]")
				}
				if e.Kind == sequence.EventBlockStart && e.Block != "rect" {
					need(e.Block)
				}
			}
		}
	}
}

// REQ: FLOW-012 STATE-010
// Każdy wyraz z modelu grafu jest na rysunku tyle samo razy (etykiety bywają łamane na linie).
func TestGraphCorpusWordsPresent(t *testing.T) {
	for _, f := range graphCorpusFiles(t) {
		svg, err := Render(readFile(t, f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		have := map[string]int{}
		for _, s := range checkSVG(t, f, svg) {
			for _, w := range strings.Fields(s) {
				have[w]++
			}
		}
		want := map[string]int{}
		for _, s := range parseGraph(t, f).Texts() {
			for _, w := range strings.Fields(s) {
				want[w]++
			}
		}
		for w, n := range want {
			if have[w] != n {
				t.Errorf("%s: word %q appears %d times in the SVG, want %d", f, w, have[w], n)
			}
		}
		for w, n := range have {
			if want[w] == 0 {
				t.Errorf("%s: word %q (%d times) is in the SVG but not in the source", f, w, n)
			}
		}
	}
}

var reqRef = regexp.MustCompile(`\b[A-Z]{3,5}-\d{3}\b`)

// Każde wymaganie ma pokrycie w korpusie albo w testach, a każde odwołanie wskazuje istniejące wymaganie.
func TestRequirementsCovered(t *testing.T) {
	defined := map[string]bool{}
	docs, _ := filepath.Glob("requirements/*.md")
	for _, doc := range docs {
		for _, line := range strings.Split(readFile(t, doc), "\n") {
			if id := reqRef.FindString(line); id != "" && strings.HasPrefix(line, "| "+id) {
				defined[id] = true
			}
		}
	}
	if len(defined) == 0 {
		t.Fatal("no requirements found")
	}
	used := map[string]bool{}
	var sources []string
	all, _ := filepath.Glob("testdata/corpus/*/*.mmd")
	sources = append(sources, all...)
	for _, pattern := range []string{"*_test.go", "internal/*/*_test.go", "markdown/*_test.go"} {
		m, _ := filepath.Glob(pattern)
		sources = append(sources, m...)
	}
	for _, f := range sources {
		for _, line := range strings.Split(readFile(t, f), "\n") {
			if !strings.Contains(line, "REQ:") {
				continue
			}
			for _, id := range reqRef.FindAllString(line, -1) {
				used[id] = true
				if !defined[id] {
					t.Errorf("%s refers to unknown requirement %s", f, id)
				}
			}
		}
	}
	for id := range defined {
		if !used[id] {
			t.Errorf("requirement %s has no corpus file or test", id)
		}
	}
	for _, f := range all {
		if !strings.Contains(readFile(t, f), "REQ:") {
			t.Errorf("%s has no REQ line", f)
		}
	}
}

// REQ: SEQ-011
func TestUnsupportedType(t *testing.T) {
	for _, src := range []string{"journey\n title x", "mindmap\n root", "classDiagram\n A <|-- B", "flowchart-elk TD\n A-->B"} {
		if _, err := Render(src); !errors.Is(err, ErrUnsupportedType) {
			t.Errorf("%q: want unsupported type error, got %v", src, err)
		}
	}
	if _, err := Render("  \n%% only a comment\n"); err == nil {
		t.Error("empty source: want an error")
	}
}

// REQ: SEQ-011 SEQ-012
func TestRejectsCharactersInvalidInXML(t *testing.T) {
	for src, code := range map[string]string{
		"sequenceDiagram\nA->>B: \xc4x":  "invalid-utf8",
		"sequenceDiagram\nA->>B: a\x01b": "control-character",
		"sequenceDiagram\nA->>B: \uffff": "control-character",
	} {
		_, err := Render(src)
		se, ok := err.(*SyntaxError)
		if !ok || se.Code != code || se.Line != 2 {
			t.Errorf("%q: got %v, want code %q on line 2", src, err, code)
		}
	}
}

// REQ: SEQ-012
func FuzzRender(f *testing.F) {
	files, _ := filepath.Glob("testdata/corpus/*/*.mmd")
	for _, p := range files {
		b, _ := os.ReadFile(p)
		f.Add(string(b))
	}
	f.Fuzz(func(t *testing.T, src string) {
		checkErr := Check(src)
		svg, err := Render(src)
		if err != nil {
			if checkErr == nil {
				t.Fatalf("Check accepted a source that Render rejects: %v\nsource: %q", err, src)
			}
			return
		}
		if err := xml.Unmarshal(svg, new(struct{})); err != nil {
			t.Fatalf("not well-formed XML: %v\nsource: %q", err, src)
		}
	})
}

// REQ: SEQ-017
func TestMissingGlyphs(t *testing.T) {
	missing, err := MissingGlyphs("sequenceDiagram\nA->>B: Zażółć gęślą jaźń 漢字 ✓ 字")
	if err != nil {
		t.Fatal(err)
	}
	if string(missing) != "字漢" && string(missing) != "✓字漢" {
		t.Errorf("missing glyphs: %q", string(missing))
	}
	for _, f := range corpusFiles(t) {
		if m, err := MissingGlyphs(readFile(t, f)); err != nil || len(m) > 0 {
			t.Errorf("%s: the bundled typeface lacks %q (%v)", f, string(m), err)
		}
	}
	// Bez znanego kroju nie ma czego sprawdzać.
	m, err := MissingGlyphs("flowchart TD\nA[漢字]", WithMeasure(func(s string, size float64) float64 { return size }))
	if err != nil || m != nil {
		t.Errorf("custom measure: %q %v", string(m), err)
	}
}
