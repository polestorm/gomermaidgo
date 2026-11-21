// Package c4 rozbiera diagramy C4 (C4Context, C4Container, C4Component, C4Dynamic, C4Deployment)
// do modelu grafu. Składnia według publicznej dokumentacji Mermaid, w zakresie profilu
// z docs/profile.md.
package c4

import (
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni diagramów C4.
const (
	CodeHeader    = "c4-header"
	CodeStatement = "c4-statement"
	CodeArguments = "c4-arguments"
	CodeBoundary  = "c4-boundary"
)

var (
	call    = regexp.MustCompile(`^([A-Za-z_]+)\s*\((.*)\)\s*(\{)?$`)
	headers = map[string]bool{"C4Context": true, "C4Container": true, "C4Component": true, "C4Dynamic": true, "C4Deployment": true}

	// Rodzaje elementów: nazwa na rysunku i to, czy czwarty argument jest technologią.
	bases = map[string]struct {
		label string
		tech  bool
	}{
		"Person": {"Person", false}, "System": {"System", false},
		"Container": {"Container", true}, "Component": {"Component", true},
	}
)

// arguments dzieli listę argumentów na przecinkach poza cudzysłowami, zdejmuje cudzysłowy
// i pomija argumenty nazwane ($klucz="wartość"), które dotyczą wyglądu.
func arguments(s string) ([]string, bool) {
	var out []string
	var b strings.Builder
	quoted := false
	flush := func() {
		if a := strings.TrimSpace(b.String()); !strings.HasPrefix(a, "$") {
			out = append(out, strings.TrimSpace(strings.Trim(a, `"`)))
		}
		b.Reset()
	}
	for _, c := range s {
		switch {
		case c == '"':
			quoted = !quoted
			b.WriteRune(c)
		case c == ',' && !quoted:
			flush()
		default:
			b.WriteRune(c)
		}
	}
	flush()
	return out, !quoted
}

// Parse rozbiera źródło diagramu C4.
func Parse(source string) (*graph.Graph, error) {
	g := graph.New("c4")
	var acc syntax.Accessibility
	seenHeader := false
	title, titleLine := "", 0
	var stack []string
	var opened []int
	scope := func() string {
		if len(stack) == 0 {
			return ""
		}
		return stack[len(stack)-1]
	}
	arg := func(args []string, i int) string {
		if i < len(args) {
			return args[i]
		}
		return ""
	}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if !headers[s] {
				return nil, syntax.Errorf(line, CodeHeader, "expected C4Context, C4Container, C4Component, C4Dynamic or C4Deployment, found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		if s == "}" {
			if len(stack) == 0 {
				return nil, syntax.Errorf(line, CodeBoundary, "\"}\" without an open boundary")
			}
			stack, opened = stack[:len(stack)-1], opened[:len(opened)-1]
			continue
		}
		if s == "title" || strings.HasPrefix(s, "title ") {
			title, titleLine = strings.TrimSpace(strings.TrimPrefix(s, "title")), line
			continue
		}
		m := call.FindStringSubmatch(s)
		if m == nil {
			return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
		}
		name := m[1]
		args, ok := arguments(m[2])
		if !ok {
			return nil, syntax.Errorf(line, CodeArguments, "a quoted argument of %s is not closed", name)
		}
		need := func(count int) error {
			for i := 0; i < count; i++ {
				if arg(args, i) == "" {
					return syntax.Errorf(line, CodeArguments, "%s needs at least %d arguments", name, count)
				}
			}
			return nil
		}
		switch {
		case strings.HasPrefix(name, "Update"):
			// wygląd i ustawienia układu ze źródła nie są rysowane
		case strings.HasSuffix(name, "Boundary") || name == "Node" || name == "Node_L" || name == "Node_R" || name == "Deployment_Node":
			if m[3] == "" {
				return nil, syntax.Errorf(line, CodeBoundary, "%s opens a block: the line must end with \"{\"", name)
			}
			if err := need(2); err != nil {
				return nil, err
			}
			kind := strings.ReplaceAll(strings.TrimSuffix(name, "_Boundary"), "_", " ")
			switch {
			case name == "Boundary":
				kind = arg(args, 2)
			case !strings.HasSuffix(name, "Boundary"): // węzeł wdrożenia: typ jest argumentem
				kind = arg(args, 2)
			}
			lines := []string{args[1]}
			if kind != "" {
				lines = append(lines, "["+kind+"]")
			}
			sub := &graph.Subgraph{ID: args[0], Title: lines, Parent: scope(), Line: line, Paint: graph.Paint{Dash: "6 4"}}
			if err := g.AddSubgraph(sub); err != nil {
				return nil, err
			}
			stack, opened = append(stack, args[0]), append(opened, line)
		case name == "Rel" || name == "BiRel" || strings.HasPrefix(name, "Rel_") || name == "RelIndex":
			if name == "RelIndex" { // pierwszy argument to numer kroku w diagramie dynamicznym
				if len(args) < 1 {
					return nil, syntax.Errorf(line, CodeArguments, "RelIndex needs an index")
				}
				args = args[1:]
			}
			if err := need(3); err != nil {
				return nil, err
			}
			for _, end := range args[:2] {
				if g.Node(end) == nil && g.Subgraph(end) == nil {
					return nil, syntax.Errorf(line, CodeArguments, "relationship refers to unknown element %q", end)
				}
			}
			e := &graph.Edge{From: args[0], To: args[1], Style: graph.Normal, StartHead: graph.NoHead, EndHead: graph.ArrowHead, Label: []string{args[2]}, Line: line}
			if tech := arg(args, 3); tech != "" {
				e.Label = append(e.Label, "["+tech+"]")
			}
			switch name {
			case "BiRel":
				e.StartHead = graph.ArrowHead
			case "Rel_Back":
				e.StartHead, e.EndHead = graph.ArrowHead, graph.NoHead
			}
			if err := g.AddEdge(e); err != nil {
				return nil, err
			}
		default:
			if m[3] != "" {
				return nil, syntax.Errorf(line, CodeBoundary, "%s does not open a block", name)
			}
			// Element: rdzeń nazwy (Person, System, Container, Component), przyrostki Db, Queue i _Ext.
			core := strings.TrimSuffix(name, "_Ext")
			external := core != name
			shape := graph.Rect
			switch {
			case strings.HasSuffix(core, "Db"):
				core, shape = strings.TrimSuffix(core, "Db"), graph.Cylinder
			case strings.HasSuffix(core, "Queue"):
				core, shape = strings.TrimSuffix(core, "Queue"), graph.Round
			}
			base, known := bases[core]
			if !known {
				return nil, syntax.Errorf(line, CodeStatement, "unknown element %q", name)
			}
			if core == "Person" {
				shape = graph.Stadium
			}
			if err := need(2); err != nil {
				return nil, err
			}
			kind, description := base.label, arg(args, 2)
			if external {
				kind = "External " + kind
			}
			if base.tech {
				description = arg(args, 3)
				if tech := arg(args, 2); tech != "" {
					kind += ": " + tech
				}
			}
			node, err := g.EnsureNode(args[0], scope(), shape, line)
			if err != nil {
				return nil, err
			}
			if node.Explicit {
				return nil, syntax.Errorf(line, CodeArguments, "element %q is defined twice", args[0])
			}
			node.Explicit = true
			node.Label = []string{string(graph.MarkBold) + args[1] + string(graph.MarkBold), "«" + kind + "»"}
			if description != "" {
				node.Label = append(node.Label, description)
			}
			if external {
				node.Paint = graph.Paint{Fill: "#f6f8fa", Stroke: "#7b8794", Dash: "5 4"}
			}
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected a C4 diagram header")
	}
	if len(stack) > 0 {
		return nil, syntax.Errorf(opened[len(opened)-1], CodeBoundary, "boundary %q is not closed with \"}\"", stack[len(stack)-1])
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	// Tytuł diagramu jest ramką wokół całości: układ grafu nie ma osobnego miejsca na tytuł.
	if title != "" {
		const frame = "c4 title"
		for _, s := range g.Subgraphs {
			if s.Parent == "" {
				s.Parent = frame
			}
		}
		for _, node := range g.Nodes {
			if node.Parent == "" {
				node.Parent = frame
			}
		}
		if err := g.AddSubgraph(&graph.Subgraph{ID: frame, Title: []string{title}, Line: titleLine}); err != nil {
			return nil, err
		}
	}
	g.AccTitle, g.AccDescr = acc.Title, acc.Descr
	if g.AccTitle == "" {
		g.AccTitle = title
	}
	if err := g.Finish(); err != nil {
		return nil, err
	}
	return g, nil
}
