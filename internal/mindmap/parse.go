// Package mindmap rozbiera mapę myśli (mindmap) do modelu grafu: drzewa od korzenia.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package mindmap

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni mapy myśli.
const (
	CodeHeader    = "mindmap-header"
	CodeStatement = "mindmap-statement"
	CodeRoot      = "mindmap-root"
	CodeEmpty     = "mindmap-empty"
)

var (
	lineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)
	strong    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	emphasis  = regexp.MustCompile(`\*([^*]+)\*`)

	// Kształty w kolejności dopasowywania: dłuższe ograniczniki przed krótszymi.
	shapes = []struct {
		pattern *regexp.Regexp
		shape   graph.Shape
	}{
		{regexp.MustCompile(`^[^\[\](){}]*\(\((.*)\)\)$`), graph.Circle},
		{regexp.MustCompile(`^[^\[\](){}]*\)\)(.*)\(\($`), graph.Hexagon}, // „wybuch” rysowany jak sześciokąt
		{regexp.MustCompile(`^[^\[\](){}]*\)(.*)\($`), graph.Stadium},     // „chmura” rysowana jak pole z półkolami
		{regexp.MustCompile(`^[^\[\](){}]*\{\{(.*)\}\}$`), graph.Hexagon},
		{regexp.MustCompile(`^[^\[\](){}]*\[(.*)\]$`), graph.Rect},
		{regexp.MustCompile(`^[^\[\](){}]*\((.*)\)$`), graph.Round},
	}
)

// text czyści tekst gałęzi: zdejmuje cudzysłowy, zamienia wyróżnienia Markdown na znaki stylu
// i dzieli na linie.
func text(raw string) []string {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s = s[1 : len(s)-1]
	}
	if len(s) >= 2 && strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
		s = strong.ReplaceAllString(s[1:len(s)-1], string(graph.MarkBold)+"$1"+string(graph.MarkBold))
		s = emphasis.ReplaceAllString(s, string(graph.MarkItalic)+"$1"+string(graph.MarkItalic))
	}
	var out []string
	for _, part := range lineBreak.Split(s, -1) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	// Wyróżnienie obejmujące złamanie linii: każda linia dostaje własną parę znaków.
	for _, mark := range []string{string(graph.MarkBold), string(graph.MarkItalic)} {
		open := false
		for i := range out {
			if open {
				out[i] = mark + out[i]
			}
			if open = strings.Count(out[i], mark)%2 == 1; open {
				out[i] += mark
			}
		}
	}
	return out
}

// Parse rozbiera źródło mapy myśli. Hierarchię wyznacza wcięcie: rodzicem gałęzi jest najbliższa
// wcześniejsza gałąź o mniejszym wcięciu.
func Parse(source string) (*graph.Graph, error) {
	g := graph.New("mindmap")
	g.Direction = graph.LeftRight
	type open struct {
		indent int
		id     string
	}
	var stack []open
	seenHeader := false
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "mindmap" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"mindmap\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if strings.HasPrefix(s, "::") {
			continue // ikona albo klasa stylu gałęzi: rysunek ich nie ma
		}
		indent := 0
		for _, c := range raw {
			if c == ' ' {
				indent++
			} else if c == '\t' {
				indent += 4
			} else {
				break
			}
		}
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 && len(g.Nodes) > 0 {
			return nil, syntax.Errorf(line, CodeRoot, "the mind map has one root; %q is not indented under it", s)
		}
		shape, body := graph.Round, s
		explicit := false
		for _, sh := range shapes {
			if m := sh.pattern.FindStringSubmatch(s); m != nil {
				shape, body, explicit = sh.shape, m[1], true
				break
			}
		}
		if len(g.Nodes) == 0 && !explicit {
			shape = graph.Stadium // korzeń bez podanego kształtu wyróżnia się od gałęzi
		}
		label := text(body)
		if len(label) == 0 {
			return nil, syntax.Errorf(line, CodeStatement, "branch has no text: %q", s)
		}
		id := fmt.Sprintf("n%d", len(g.Nodes)+1)
		node, err := g.EnsureNode(id, "", shape, line)
		if err != nil {
			return nil, err
		}
		node.Label, node.Explicit = label, true
		if len(stack) > 0 {
			if err := g.AddEdge(&graph.Edge{From: stack[len(stack)-1].id, To: id, Style: graph.Normal, StartHead: graph.NoHead, EndHead: graph.NoHead, Line: line}); err != nil {
				return nil, err
			}
		}
		stack = append(stack, open{indent, id})
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"mindmap\"")
	}
	if len(g.Nodes) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the mind map needs a root")
	}
	if err := g.Finish(); err != nil {
		return nil, err
	}
	return g, nil
}
