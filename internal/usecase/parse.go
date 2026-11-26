// Package usecase rozbiera diagram przypadków użycia UML (usecase w Mermaid) do wspólnego modelu
// grafu, który układa i rysuje pakiet layered. Składnia według publicznej dokumentacji Mermaid,
// w zakresie profilu z docs/profile.md.
package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni diagramu przypadków użycia.
const (
	CodeHeader    = "usecase-header"
	CodeStatement = "usecase-statement"
	maxElements   = 400
)

var (
	identifier = syntax.Lazy(`^[\p{L}\p{N}_]+$`)
	// deklaracja: Id, Id("…"), Id(…), Id[…] albo "Nazwa", potem @{…}, <<Stereotyp>>, :::klasa
	declaration = syntax.Lazy(`(?s)^(?:([\p{L}\p{N}_]+)(?:\((.*?)\)|\[(.*?)\])?|"([^"]*)")\s*(@\{[^}]*\})?\s*(<<[^>]*>>)?\s*((?::::[A-Za-z0-9_-]+)*)$`)
	// relacja: A [id@]operator B, z podpisem „A -- tekst --> B” albo „A ..> : include B”
	relation = syntax.Lazy(`^([\p{L}\p{N}_]+)\s+(?:([\p{L}\p{N}_]+)@)?(-{2,}>|<-{2,}|--o|o--|--x|x--|--\|>|\.\.>|-{2,})\s+(.*)$`)
	labelled = syntax.Lazy(`^([\p{L}\p{N}_]+)\s+(?:([\p{L}\p{N}_]+)@)?--\s+(.+?)\s+(-{2,}>)\s*([\p{L}\p{N}_]+)$`)
	strongMD = syntax.Lazy(`\*\*([^*]+)\*\*`)
	emphMD   = syntax.Lazy(`\*([^*]+)\*`)
	entities = strings.NewReplacer("#quot;", `"`, "#39;", "'", "#40;", "(", "#41;", ")", "#91;", "[", "#93;", "]", "#96;", "`", "#35;", "#", "#59;", ";")
)

// item to zadeklarowany aktor albo przypadek użycia, w kolejności źródła.
type item struct {
	id, parent string
	shape      graph.Shape
	label      []string
	line       int
}

// Parse rozbiera źródło do grafu.
func Parse(source string) (*graph.Graph, error) {
	g := graph.New("usecase")
	g.Direction = graph.TopBottom
	var acc syntax.Accessibility
	seenHeader := false
	var items []*item
	declared := map[string]*item{}
	type rel struct {
		edge *graph.Edge
		id   string
	}
	var rels []rel
	edgeIDs := map[string]*graph.Edge{}
	classDefs := map[string]graph.Paint{}
	type use struct {
		targets, class string
		line           int
	}
	var classUses []use
	type styleUse struct {
		target string
		paint  graph.Paint
		line   int
	}
	var styles []styleUse
	boundaries := map[string]bool{}
	boundary := ""
	notes := 0
	var tables []*graph.Node
	lines := joinOpen(strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n"))
	for k := 0; k < len(lines); k++ {
		line := lines[k].line
		s := strings.TrimSpace(lines[k].text)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "usecase-beta" && s != "usecase" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"usecase-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		if len(items)+len(rels) > maxElements {
			return nil, syntax.Errorf(line, CodeStatement, "too many elements (limit %d)", maxElements)
		}
		word, rest, _ := strings.Cut(s, " ")
		rest = strings.TrimSpace(rest)
		declare := func(it *item) error {
			if declared[it.id] != nil || boundaries[it.id] {
				return syntax.Errorf(line, CodeStatement, "%q is declared twice", it.id)
			}
			it.parent, it.line = boundary, line
			declared[it.id] = it
			items = append(items, it)
			return nil
		}
		switch {
		case word == "direction":
			d, ok := graph.ParseDirection(rest)
			if !ok {
				return nil, syntax.Errorf(line, CodeStatement, "unknown direction %q", rest)
			}
			g.Direction = d
		case word == "end" && rest == "":
			if boundary == "" {
				return nil, syntax.Errorf(line, CodeStatement, "end without systemBoundary")
			}
			boundary = ""
		case word == "systemBoundary":
			if boundary != "" {
				return nil, syntax.Errorf(line, CodeStatement, "system boundaries cannot be nested")
			}
			m := declaration.FindStringSubmatch(rest)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "bad system boundary %q", rest)
			}
			id, title := m[1], m[3]
			if m[4] != "" {
				id, title = strings.ReplaceAll(m[4], " ", "_"), m[4]
			}
			if title == "" {
				title = id
			}
			if boundaries[id] || declared[id] != nil {
				return nil, syntax.Errorf(line, CodeStatement, "%q is declared twice", id)
			}
			t, err := label(title, line)
			if err != nil {
				return nil, err
			}
			if err := g.AddSubgraph(&graph.Subgraph{ID: id, Title: t, Line: line}); err != nil {
				return nil, err
			}
			boundaries[id], boundary = true, id
			for _, c := range classNames(m[7]) {
				classUses = append(classUses, use{id, c, line})
			}
		case word == "actor":
			m := declaration.FindStringSubmatch(rest)
			if m == nil || m[1] == "" || m[3] != "" {
				return nil, syntax.Errorf(line, CodeStatement, "bad actor %q (write actor Id or actor Id(\"label\"))", rest)
			}
			if err := checkMeta(m[5], line, "type", "icon", "business"); err != nil {
				return nil, err
			}
			text := m[2]
			if text == "" {
				text = m[1]
			}
			t, err := label(text, line)
			if err != nil {
				return nil, err
			}
			if err := declare(&item{id: m[1], shape: graph.Actor, label: stereotype(m[6], t)}); err != nil {
				return nil, err
			}
			for _, c := range classNames(m[7]) {
				classUses = append(classUses, use{m[1], c, line})
			}
		case word == "note" && strings.HasPrefix(rest, "for "):
			target, text, ok := strings.Cut(strings.TrimSpace(rest[4:]), " ")
			if !ok || !identifier.MatchString(target) {
				return nil, syntax.Errorf(line, CodeStatement, "write note for Id \"text\"")
			}
			t, err := label(strings.TrimSpace(text), line)
			if err != nil {
				return nil, err
			}
			notes++
			id := fmt.Sprintf("note-%d", notes)
			n, err := g.EnsureNode(id, "", graph.NoteBox, line)
			if err != nil {
				return nil, err
			}
			n.Label, n.Explicit = t, true
			rels = append(rels, rel{edge: &graph.Edge{From: id, To: target, Style: graph.Dotted, StartHead: graph.NoHead, EndHead: graph.NoHead, MinLen: 1, Line: line}})
		case word == "json":
			id, body, _ := strings.Cut(rest, "@")
			id = strings.TrimSpace(id)
			if !identifier.MatchString(id) {
				return nil, syntax.Errorf(line, CodeStatement, "bad JSON node %q", rest)
			}
			// Treść JSON do domknięcia nawiasów, także przez kilka linii.
			for strings.Count(body, "{")+strings.Count(body, "[") > strings.Count(body, "}")+strings.Count(body, "]") && k+1 < len(lines) {
				k++
				body += "\n" + lines[k].text
			}
			body = strings.TrimSpace(body)
			class := ""
			if i := strings.LastIndex(body, ":::"); i > strings.LastIndex(body, "}") {
				body, class = strings.TrimSpace(body[:i]), body[i+3:]
			}
			rows, err := jsonRows(body, line)
			if err != nil {
				return nil, err
			}
			if declared[id] != nil || boundaries[id] {
				return nil, syntax.Errorf(line, CodeStatement, "%q is declared twice", id)
			}
			n, err := g.EnsureNode(id, "", graph.Entity, line)
			if err != nil {
				return nil, err
			}
			n.Label, n.Table, n.Explicit = []string{id}, rows, true
			tables = append(tables, n)
			declared[id] = &item{id: id, shape: graph.Entity, line: line}
			if class != "" {
				classUses = append(classUses, use{id, class, line})
			}
		case word == "classDef":
			names, props, _ := strings.Cut(rest, " ")
			p, err := graph.ParsePaint(props, line)
			if err != nil {
				return nil, err
			}
			g.DefineClass(names, p)
			for _, n := range strings.Split(names, ",") {
				classDefs[strings.TrimSpace(n)] = p
			}
		case word == "class":
			targets, classes, _ := strings.Cut(rest, " ")
			for _, c := range strings.Split(classes, ",") {
				classUses = append(classUses, use{targets, strings.TrimSpace(c), line})
			}
		case word == "style":
			target, props, _ := strings.Cut(rest, " ")
			p, err := graph.ParsePaint(props, line)
			if err != nil {
				return nil, err
			}
			styles = append(styles, styleUse{target, p, line})
		case strings.Contains(s, "@{") && identifier.MatchString(strings.TrimSpace(s[:strings.Index(s, "@{")])) && strings.HasSuffix(s, "}"):
			// Metadane podane później (granica, aktor, krawędź): typ granicy i animacja nie zmieniają rysunku.
		case relation.MatchString(s) || labelled.MatchString(s):
			e, id, err := parseRelation(s, line)
			if err != nil {
				return nil, err
			}
			rels = append(rels, rel{e, id})
			if id != "" {
				if edgeIDs[id] != nil {
					return nil, syntax.Errorf(line, CodeStatement, "edge id %q is used twice", id)
				}
				edgeIDs[id] = e
			}
		default:
			m := declaration.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeStatement, "unknown statement %q", s)
			}
			if err := checkMeta(m[5], line, "business"); err != nil {
				return nil, err
			}
			id, text, shape := m[1], m[2], graph.UseCase
			switch {
			case m[4] != "":
				id, text = strings.ReplaceAll(m[4], " ", "_"), m[4]
			case m[3] != "":
				text, shape = m[3], graph.Rect
			case text == "":
				text = id
			}
			t, err := label(text, line)
			if err != nil {
				return nil, err
			}
			if err := declare(&item{id: id, shape: shape, label: stereotype(m[6], t)}); err != nil {
				return nil, err
			}
			for _, c := range classNames(m[7]) {
				classUses = append(classUses, use{id, c, line})
			}
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"usecase-beta\"")
	}
	if boundary != "" {
		return nil, syntax.Errorf(0, CodeStatement, "systemBoundary %q is not closed with end", boundary)
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	// Węzły: najpierw zadeklarowane, potem końce relacji bez deklaracji (przypadki użycia).
	for _, it := range items {
		n, err := g.EnsureNode(it.id, it.parent, it.shape, it.line)
		if err != nil {
			return nil, err
		}
		n.Label, n.Explicit = it.label, true
	}
	for _, r := range rels {
		for _, end := range []string{r.edge.From, r.edge.To} {
			if g.Node(end) == nil && !boundaries[end] {
				if _, err := g.EnsureNode(end, "", graph.UseCase, r.edge.Line); err != nil {
					return nil, err
				}
				g.Node(end).Label = []string{end}
			}
		}
		if boundaries[r.edge.From] || boundaries[r.edge.To] {
			return nil, syntax.Errorf(r.edge.Line, CodeStatement, "a relationship joins actors and use cases, not a system boundary")
		}
		from, to := g.Node(r.edge.From), g.Node(r.edge.To)
		if r.edge.EndHead == graph.Triangle && (from.Shape == graph.Actor) != (to.Shape == graph.Actor) {
			return nil, syntax.Errorf(r.edge.Line, CodeStatement, "generalisation joins two actors or two use cases")
		}
		if from.Shape == graph.NoteBox && (to.Shape == graph.Entity || to.Shape == graph.NoteBox) {
			return nil, syntax.Errorf(r.edge.Line, CodeStatement, "a note is attached to an actor or a use case")
		}
		if err := g.AddEdge(r.edge); err != nil {
			return nil, err
		}
	}
	// Style krawędzi z identyfikatorem stosujemy wprost; resztę obsługuje model grafu.
	for _, u := range classUses {
		var rest []string
		for _, t := range strings.Split(u.targets, ",") {
			t = strings.TrimSpace(t)
			if e := edgeIDs[t]; e != nil {
				p, ok := classDefs[u.class]
				if !ok {
					return nil, syntax.Errorf(u.line, CodeStatement, "class %q is not defined", u.class)
				}
				e.Paint = e.Paint.Merge(p)
				continue
			}
			rest = append(rest, t)
		}
		if len(rest) > 0 {
			g.AssignClass(strings.Join(rest, ","), u.class, u.line)
		}
	}
	for _, st := range styles {
		if e := edgeIDs[st.target]; e != nil {
			e.Paint = e.Paint.Merge(st.paint)
			continue
		}
		g.SetStyle(st.target, st.paint, st.line)
	}
	if err := g.Finish(); err != nil {
		return nil, err
	}
	g.AccTitle, g.AccDescr = acc.Title, acc.Descr
	return g, nil
}

// srcLine to linia źródła po złączeniu linii przerwanych w napisie Markdown.
type srcLine struct {
	text string
	line int
}

// joinOpen łączy linie, w których napis Markdown ("`…`") przechodzi do następnej linii.
func joinOpen(raw []string) []srcLine {
	var out []srcLine
	for i := 0; i < len(raw); i++ {
		s, start := raw[i], i+1
		for strings.Count(s, "\"`")+strings.Count(s, "`\"") == 1 && i+1 < len(raw) {
			i++
			s += "\n" + strings.TrimSpace(raw[i])
		}
		out = append(out, srcLine{s, start})
	}
	return out
}

// parseRelation czyta relację: powiązanie z grotem, kółkiem, krzyżykiem albo bez, z podpisem;
// include i extend; uogólnienie.
func parseRelation(s string, line int) (*graph.Edge, string, error) {
	e := &graph.Edge{Style: graph.Normal, StartHead: graph.NoHead, EndHead: graph.NoHead, MinLen: 1, Line: line}
	if m := labelled.FindStringSubmatch(s); m != nil {
		t, err := label(m[3], line)
		if err != nil {
			return nil, "", err
		}
		e.From, e.To, e.Label, e.EndHead, e.MinLen = m[1], m[5], t, graph.ArrowHead, len(m[4])-2
		return e, m[2], nil
	}
	m := relation.FindStringSubmatch(s)
	e.From = m[1]
	op, rest := m[3], strings.TrimSpace(m[4])
	switch {
	case op == "..>":
		kind, target, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(rest, ":")), " ")
		target = strings.TrimSpace(target)
		if !strings.HasPrefix(rest, ":") || !ok || (kind != "include" && kind != "extend") || !identifier.MatchString(target) {
			return nil, "", syntax.Errorf(line, CodeStatement, "write A ..> : include B or A ..> : extend B")
		}
		e.To, e.Style, e.EndHead, e.Label = target, graph.Dotted, graph.OpenArrow, []string{"«" + kind + "»"}
		return e, m[2], nil
	case op == "--|>":
		e.EndHead = graph.Triangle
	case strings.HasSuffix(op, ">"):
		e.EndHead, e.MinLen = graph.ArrowHead, len(op)-2
	case strings.HasPrefix(op, "<"):
		e.StartHead, e.MinLen = graph.ArrowHead, len(op)-2
	case op == "--o":
		e.EndHead = graph.CircleHead
	case op == "o--":
		e.StartHead = graph.CircleHead
	case op == "--x":
		e.EndHead = graph.CrossHead
	case op == "x--":
		e.StartHead = graph.CrossHead
	default:
		e.MinLen = len(op) - 1
	}
	if !identifier.MatchString(rest) {
		return nil, "", syntax.Errorf(line, CodeStatement, "relationship target %q is not an identifier", rest)
	}
	e.To = rest
	return e, m[2], nil
}

// label czyści tekst: cudzysłów, napis Markdown w odwrotnych apostrofach (pogrubienie, kursywa,
// złamania linii), kody znaków #quot; i podobne, <br>.
func label(raw string, line int) ([]string, error) {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if strings.HasPrefix(s, "`") {
		if len(s) < 2 || !strings.HasSuffix(s, "`") {
			return nil, syntax.Errorf(line, CodeStatement, "markdown string opened with a backtick is not closed")
		}
		s = strongMD.ReplaceAllString(s[1:len(s)-1], string(graph.MarkBold)+"$1"+string(graph.MarkBold))
		s = emphMD.ReplaceAllString(s, string(graph.MarkItalic)+"$1"+string(graph.MarkItalic))
		s = strings.ReplaceAll(s, "\n", "<br>")
	}
	s = entities.Replace(s)
	out := textwrap.Split(s)
	if len(out) == 0 {
		return nil, syntax.Errorf(line, CodeStatement, "empty label")
	}
	return out, nil
}

// stereotype dopisuje stereotyp «Nazwa» nad podpisem.
func stereotype(s string, label []string) []string {
	if s == "" {
		return label
	}
	return append([]string{"«" + strings.TrimSpace(s[2:len(s)-2]) + "»"}, label...)
}

func classNames(s string) []string {
	var out []string
	for _, c := range strings.Split(s, ":::") {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// checkMeta sprawdza klucze metadanych @{…}: dozwolone są tylko podane.
func checkMeta(meta string, line int, allowed ...string) error {
	if meta == "" {
		return nil
	}
	body := strings.TrimSuffix(strings.TrimPrefix(meta, "@{"), "}")
	for _, kv := range strings.Split(body, ",") {
		key, _, _ := strings.Cut(kv, ":")
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		ok := false
		for _, a := range allowed {
			ok = ok || key == a
		}
		if !ok {
			return syntax.Errorf(line, CodeStatement, "metadata key %q is not allowed here", key)
		}
	}
	return nil
}

// jsonRows spłaszcza obiekt JSON do wierszy klucz i wartość, w kolejności źródła: zagnieżdżone
// klucze jako a.b, elementy tablic jako a[0].
func jsonRows(body string, line int) ([][]string, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	var rows [][]string
	var walk func(path string) error
	walk = func(path string) error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case json.Delim:
			empty := true
			if t == '{' {
				for dec.More() {
					k, err := dec.Token()
					if err != nil {
						return err
					}
					empty = false
					key := k.(string)
					if path != "" {
						key = path + "." + key
					}
					if err := walk(key); err != nil {
						return err
					}
				}
			} else {
				for i := 0; dec.More(); i++ {
					empty = false
					if err := walk(fmt.Sprintf("%s[%d]", path, i)); err != nil {
						return err
					}
				}
			}
			if _, err := dec.Token(); err != nil {
				return err
			}
			if empty && path != "" {
				v := "{}"
				if t == '[' {
					v = "[]"
				}
				rows = append(rows, []string{path, v, "", ""})
			}
		case nil:
			rows = append(rows, []string{path, "null", "", ""})
		default:
			rows = append(rows, []string{path, fmt.Sprint(t), "", ""})
		}
		return nil
	}
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "{") {
		return nil, syntax.Errorf(line, CodeStatement, "a JSON node needs an object: json Id@{ \"key\": value }")
	}
	if err := walk(""); err != nil {
		return nil, syntax.Errorf(line, CodeStatement, "bad JSON: %v", err)
	}
	if dec.More() {
		return nil, syntax.Errorf(line, CodeStatement, "unexpected text after the JSON object")
	}
	// Ten sam klucz dwa razy: wygrywa ostatnia wartość, miejsce zostaje pierwsze.
	seen := map[string]int{}
	var out [][]string
	for _, r := range rows {
		if i, ok := seen[r[0]]; ok {
			out[i] = r
			continue
		}
		seen[r[0]] = len(out)
		out = append(out, r)
	}
	if len(out) == 0 {
		out = [][]string{{"{}", "", "", ""}}
	}
	return out, nil
}
