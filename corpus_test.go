package gomermaidgo

import (
	"bytes"
	"encoding/xml"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/sequence"
)

var update = flag.Bool("update", false, "rewrite golden SVG files")

func corpusFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("testdata/corpus/*/*.mmd")
	if err != nil || len(files) == 0 {
		t.Fatalf("no corpus files: %v", err)
	}
	return files
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
	"rect":     {"class", "x", "y", "width", "height", "rx", "fill", "fill-opacity", "stroke", "stroke-width", "stroke-dasharray"},
	"line":     {"class", "x1", "y1", "x2", "y2", "fill", "stroke", "stroke-width", "stroke-dasharray"},
	"polyline": {"class", "points", "fill", "stroke", "stroke-width", "stroke-dasharray"},
	"polygon":  {"class", "points", "fill", "stroke", "stroke-width", "stroke-dasharray"},
	"circle":   {"class", "cx", "cy", "r", "fill", "fill-opacity", "stroke", "stroke-width", "stroke-dasharray"},
	"text":     {"class", "x", "y", "text-anchor", "font-size", "fill"},
}

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
				switch a.Name.Local {
				case "x", "y", "width", "height", "x1", "y1", "x2", "y2", "cx", "cy", "r", "rx", "points", "viewBox", "stroke-width", "fill-opacity", "font-size", "stroke-dasharray":
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
	}
}

// REQ: SEQ-002 SEQ-003 SEQ-004 SEQ-006 SEQ-010
// Każdy tekst ze źródła (etykiety uczestników, komunikatów, notatek, tytuły bloków) jest w SVG.
func TestCorpusLabelsPresent(t *testing.T) {
	for _, f := range corpusFiles(t) {
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
				if have[line] < 2 {
					t.Errorf("%s: participant label %q should appear at the top and at the bottom", f, line)
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

var reqRef = regexp.MustCompile(`SEQ-\d{3}`)

// Każde wymaganie ma pokrycie w korpusie albo w testach, a każde odwołanie wskazuje istniejące wymaganie.
func TestRequirementsCovered(t *testing.T) {
	defined := map[string]bool{}
	for _, line := range strings.Split(readFile(t, "requirements/sequence.md"), "\n") {
		if strings.HasPrefix(line, "| SEQ-") {
			defined[reqRef.FindString(line)] = true
		}
	}
	if len(defined) == 0 {
		t.Fatal("no requirements found")
	}
	used := map[string]bool{}
	var sources []string
	sources = append(sources, corpusFiles(t)...)
	for _, pattern := range []string{"*_test.go", "internal/*/*_test.go"} {
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
	for _, f := range corpusFiles(t) {
		if !strings.Contains(readFile(t, f), "REQ:") {
			t.Errorf("%s has no REQ line", f)
		}
	}
}

// REQ: SEQ-011
func TestUnsupportedType(t *testing.T) {
	for _, src := range []string{"flowchart TD\n A-->B", "graph LR\n A-->B", "stateDiagram-v2\n [*] --> A", "pie\n \"a\": 1"} {
		if _, err := Render(src); err == nil || !strings.Contains(err.Error(), "unsupported diagram type") {
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
		svg, err := Render(src)
		if err != nil {
			return
		}
		if err := xml.Unmarshal(svg, new(struct{})); err != nil {
			t.Fatalf("not well-formed XML: %v\nsource: %q", err, src)
		}
	})
}
