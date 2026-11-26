// Package agentflow rozbiera przepływ agentów (agentflow w Mermaid): przekłada źródło na schemat
// blokowy (kształty-aliasy, kontenery flow jako podgrafy, blok global, zwinięte kontenery)
// i rozbiera go parserem schematów blokowych. Numery linii zostają takie jak w źródle.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package agentflow

import (
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/flowchart"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni przepływu agentów (pozostałe błędy zgłasza parser schematu blokowego).
const (
	CodeHeader    = "agentflow-header"
	CodeStatement = "agentflow-statement"
)

// aliases to kształty przepływu agentów zapisane nazwami kształtów schematu blokowego.
var aliases = map[string]string{"task": "rounded", "tool": "subroutine", "input": "lean-right", "decision": "diamond",
	"refdoc": "lin-doc", "action": "hexagon"}

var (
	ident    = `[\p{L}_][\p{L}\p{N}_-]*`
	header   = syntax.Lazy(`^agentflow(?:-beta)?(?:\s+(TB|TD|BT|LR|RL))?$`)
	flowRule = syntax.Lazy(`^flow\s+(` + ident + `)\s*(?:\[\s*"?([^"\]]*)"?\s*\])?\s*(?:@\{.*\})?$`)
	connRule = syntax.Lazy(`^connector\s+(` + ident + `)\s*(?:\[\s*"?([^"\]]*)"?\s*\])?\s*(?:@\{.*\})?$`)
	metaOnly = syntax.Lazy(`^(` + ident + `)@\{(.*)\}$`)
	nodeMeta = syntax.Lazy(`(` + ident + `)(?:\[\s*"([^"]*)"\s*\]|\[([^\]"]*)\])?@\{([^{}]*)\}`)
	attr     = syntax.Lazy(`([\w-]+)\s*:\s*(?:"([^"]*)"|'([^']*)'|([^,}]*))`)
	// collapsedNode rozpoznaje w przekładzie węzeł zwiniętego kontenera (zob. Translate).
	collapsedNode = syntax.Lazy(`^(` + ident + `)@\{ shape: subroutine, label: "[^"]*", collapsed \}$`)
	collapsedV    = syntax.Lazy(`\bview\s*:\s*["']?collapsed["']?`)
	declared      = syntax.Lazy(`(?:^|[\s&>-])(` + ident + `)\s*(?:\[|@\{|\(|$|\s*(?:-->|-\.-|--x|--|&))`)
)

// Parse przekłada źródło na schemat blokowy i rozbiera je do grafu.
func Parse(source string) (*graph.Graph, error) {
	fc, err := Translate(source)
	if err != nil {
		return nil, err
	}
	g, err := flowchart.Parse(fc)
	if err != nil {
		return nil, err
	}
	// Zwinięte kontenery: zapisane jako podproces, rysowane własnym kształtem (tytuł, kreska,
	// trzy kropki), żeby nie myliły się z narzędziem.
	for _, line := range strings.Split(fc, "\n") {
		if m := collapsedNode.FindStringSubmatch(line); m != nil {
			if n := g.Node(m[1]); n != nil {
				n.Shape = graph.Collapsed
			}
		}
	}
	return g, nil
}

// block to otwarty kontener w czasie przekładu.
type block struct {
	kind  string // flow albo global
	id    string
	label string
}

// Translate zwraca źródło schematu blokowego z tą samą liczbą linii.
func Translate(source string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	// Metadane w kilku liniach: łączone w pierwszą, pozostałe linie zostają puste.
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "@{") {
			for strings.Count(lines[i], "{") > strings.Count(lines[i], "}") && i+1 < len(lines) {
				j := i + 1
				for j < len(lines) && lines[j] == "\x00" {
					j++
				}
				if j >= len(lines) {
					break
				}
				lines[i] += ", " + strings.TrimSpace(lines[j])
				lines[j] = "\x00"
			}
			// YAML w kilku liniach daje „{, klucz: wartość, …}”.
			lines[i] = strings.ReplaceAll(lines[i], "{, ", "{ ")
		}
	}
	// Pierwsze przejście: kontenery, zwinięcia i węzły w zwiniętych kontenerach.
	collapsed := map[string]bool{}
	containers := map[string]bool{}
	seenHeader := false
	for _, raw := range lines {
		s := strings.TrimSpace(raw)
		if m := flowRule.FindStringSubmatch(s); m != nil {
			containers[m[1]] = true
			if collapsedV.MatchString(s) {
				collapsed[m[1]] = true
			}
		}
		if m := metaOnly.FindStringSubmatch(s); m != nil && collapsedV.MatchString(m[2]) {
			collapsed[m[1]] = true
		}
	}
	members := map[string]string{} // węzeł w zwiniętym kontenerze → kontener
	var stack []block
	hidden := "" // najbardziej zewnętrzny zwinięty kontener, w którym jesteśmy
	for _, raw := range lines {
		s := strings.TrimSpace(raw)
		switch {
		case flowRule.MatchString(s):
			m := flowRule.FindStringSubmatch(s)
			stack = append(stack, block{"flow", m[1], m[2]})
			if hidden == "" && collapsed[m[1]] {
				hidden = m[1]
			} else if hidden != "" {
				members[m[1]] = hidden
			}
			continue
		case s == "global":
			stack = append(stack, block{kind: "global"})
			continue
		case s == "end":
			if len(stack) > 0 {
				if top := stack[len(stack)-1]; top.kind == "flow" && top.id == hidden {
					hidden = ""
				}
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if hidden == "" || inGlobal(stack) {
			continue
		}
		for _, m := range declared.FindAllStringSubmatch(s, -1) {
			if id := m[1]; !containers[id] && id != "end" && id != "global" {
				members[id] = hidden
			}
		}
	}

	// Drugie przejście: przekład linia po linii.
	out := make([]string, len(lines))
	var refs [][2]string // narzędzie i łącznik z connectorRef
	stack, hidden = nil, ""
	for i, raw := range lines {
		line := i + 1
		s := strings.TrimSpace(raw)
		if s == "\x00" || s == "" || strings.HasPrefix(s, "%%") {
			if s != "\x00" {
				out[i] = raw
			}
			continue
		}
		if !seenHeader {
			m := header.FindStringSubmatch(s)
			if m == nil {
				return "", syntax.Errorf(line, CodeHeader, "expected \"agentflow\", found %q", s)
			}
			dir := m[1]
			if dir == "" {
				dir = "TB"
			}
			out[i], seenHeader = "flowchart "+dir, true
			continue
		}
		switch {
		case flowRule.MatchString(s):
			m := flowRule.FindStringSubmatch(s)
			stack = append(stack, block{"flow", m[1], m[2]})
			label := m[2]
			if label == "" {
				label = m[1]
			}
			switch {
			case hidden != "":
			case collapsed[m[1]]:
				hidden = m[1]
				out[i] = m[1] + `@{ shape: subroutine, label: "` + label + `", collapsed }`
			default:
				out[i] = "subgraph " + m[1] + `["` + label + `"]`
			}
			continue
		case s == "global":
			stack = append(stack, block{kind: "global"})
			continue
		case s == "end":
			if len(stack) == 0 {
				return "", syntax.Errorf(line, CodeStatement, "end without flow or global")
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			switch {
			case top.kind == "global":
			case top.id == hidden:
				hidden = ""
			case hidden == "":
				out[i] = "end"
			}
			continue
		}
		if hidden != "" && !inGlobal(stack) {
			continue // treść zwiniętego kontenera nie jest rysowana
		}
		if m := metaOnly.FindStringSubmatch(s); m != nil && containers[m[1]] {
			continue // metadane kontenera (view, algorithm) zostały już uwzględnione
		}
		if m := connRule.FindStringSubmatch(s); m != nil {
			label := m[2]
			if label == "" {
				label = m[1]
			}
			out[i] = m[1] + `@{ shape: stadium, label: "` + label + `" }`
			continue
		}
		// Węzły z metadanymi: kształt-alias i podpis do zapisu schematu blokowego, reszta kluczy pomijana.
		var err error
		s = nodeMeta.ReplaceAllStringFunc(s, func(found string) string {
			m := nodeMeta.FindStringSubmatch(found)
			id, label, shape := m[1], m[2]+m[3], ""
			for _, a := range attr.FindAllStringSubmatch(m[4], -1) {
				v := strings.TrimSpace(a[2] + a[3] + a[4])
				switch a[1] {
				case "shape":
					shape = v
				case "label":
					label = v
				case "connectorRef":
					if syntax.Lazy(`^` + ident + `$`).MatchString(v) {
						refs = append(refs, [2]string{id, v})
					}
				}
			}
			if alias, ok := aliases[shape]; ok {
				shape = alias
			}
			parts := []string{}
			if shape != "" {
				parts = append(parts, "shape: "+shape)
			}
			if label != "" {
				if strings.Contains(label, `"`) {
					err = syntax.Errorf(line, CodeStatement, "a label cannot contain a quote")
				}
				parts = append(parts, `label: "`+label+`"`)
			}
			if len(parts) == 0 {
				return id
			}
			return id + "@{ " + strings.Join(parts, ", ") + " }"
		})
		if err != nil {
			return "", err
		}
		// Końce krawędzi w zwiniętych kontenerach wskazują kontener.
		for id, c := range members {
			s = regexp.MustCompile(`(^|[^\p{L}\p{N}_-])`+regexp.QuoteMeta(id)+`($|[^\p{L}\p{N}_-])`).ReplaceAllString(s, "${1}"+c+"${2}")
		}
		// Krawędź między węzłami tego samego zwiniętego kontenera znika: „c --> c” staje się „c”.
		for c := range collapsed {
			self := regexp.MustCompile(`(^|[^\p{L}\p{N}_-])` + regexp.QuoteMeta(c) + `\s*(?:-->|-\.-|--x)\s*` + regexp.QuoteMeta(c) + `($|[^\p{L}\p{N}_-])`)
			for self.MatchString(s) {
				s = self.ReplaceAllString(s, "${1}"+c+"${2}")
			}
		}
		out[i] = s
	}
	if !seenHeader {
		return "", syntax.Errorf(0, CodeHeader, "empty source: expected \"agentflow\"")
	}
	if len(stack) > 0 {
		return "", syntax.Errorf(0, CodeStatement, "flow or global is not closed with end")
	}
	// Narzędzie korzystające z łącznika: linia kropkowana bez grotu.
	for _, r := range refs {
		out = append(out, r[0]+" -.- "+r[1])
	}
	return strings.Join(out, "\n"), nil
}

func inGlobal(stack []block) bool {
	for _, b := range stack {
		if b.kind == "global" {
			return true
		}
	}
	return false
}
