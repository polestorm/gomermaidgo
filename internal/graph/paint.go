package graph

import (
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Paint to wygląd narzucony przez źródło (classDef, class, style, linkStyle, „:::”).
// Puste pole oznacza „bez zmiany”.
type Paint struct {
	Fill        string  // wypełnienie kształtu; "none" = bez wypełnienia
	Stroke      string  // kolor obrysu albo linii
	Color       string  // kolor tekstu
	StrokeWidth float64 // grubość obrysu albo linii; 0 = bez zmiany
	Dash        string  // wzór kresek, liczby rozdzielone spacją
}

// IsZero mówi, że źródło niczego nie narzuciło.
func (p Paint) IsZero() bool { return p == Paint{} }

// Merge nakłada over na p: pola ustawione w over wygrywają.
func (p Paint) Merge(over Paint) Paint {
	if over.Fill != "" {
		p.Fill = over.Fill
	}
	if over.Stroke != "" {
		p.Stroke = over.Stroke
	}
	if over.Color != "" {
		p.Color = over.Color
	}
	if over.StrokeWidth != 0 {
		p.StrokeWidth = over.StrokeWidth
	}
	if over.Dash != "" {
		p.Dash = over.Dash
	}
	return p
}

var (
	colorValue = syntax.Lazy(`^(?:#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6}|rgb\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*\)|[a-zA-Z]{3,20})$`)
	lengthPx   = syntax.Lazy(`^(\d+(?:\.\d+)?)(?:px)?$`)
	dashValue  = syntax.Lazy(`^\d+(?:\.\d+)?(?:[ ,]+\d+(?:\.\d+)?)*$`)
)

// ParsePaint czyta listę właściwości w zapisie CSS („fill:#f9f,stroke:#333,stroke-width:2px”).
// Rozumie fill, stroke, color, stroke-width i stroke-dasharray; pozostałe właściwości pomija,
// bo dotyczą rzeczy, których rysunek nie ma (czcionki, przejścia, zaokrąglenia).
// Błędna wartość rozumianej właściwości jest błędem składni.
func ParsePaint(props string, line int) (Paint, error) {
	var p Paint
	for _, item := range splitProps(props) {
		name, value, ok := strings.Cut(item, ":")
		if !ok {
			return p, syntax.Errorf(line, CodeInvalid, "style property %q needs \"name:value\"", strings.TrimSpace(item))
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "!important"))
		bad := func() (Paint, error) {
			return p, syntax.Errorf(line, CodeInvalid, "style property %s has unsupported value %q", name, value)
		}
		switch name {
		case "fill", "stroke", "color":
			if !colorValue.MatchString(value) {
				return bad()
			}
			value = strings.Join(strings.Fields(value), "")
			switch name {
			case "fill":
				p.Fill = value
			case "stroke":
				p.Stroke = value
			default:
				p.Color = value
			}
		case "stroke-width":
			m := lengthPx.FindStringSubmatch(value)
			if m == nil {
				return bad()
			}
			w, _ := strconv.ParseFloat(m[1], 64)
			if w > 12 {
				w = 12
			}
			p.StrokeWidth = w
		case "stroke-dasharray":
			if !dashValue.MatchString(value) {
				return bad()
			}
			p.Dash = strings.Join(strings.FieldsFunc(value, func(r rune) bool { return r == ' ' || r == ',' }), " ")
		}
	}
	return p, nil
}

// splitProps dzieli listę właściwości na przecinkach poza nawiasami; przecinek wewnątrz
// stroke-dasharray („5,5”) zostaje przy poprzedniej właściwości, bo nie zaczyna nowej nazwy.
func splitProps(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 && strings.Contains(s[i+1:], ":") && startsProperty(s[i+1:]) {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	if rest := strings.TrimSpace(strings.TrimRight(s[start:], "; \t")); rest != "" {
		out = append(out, rest)
	}
	return out
}

var propertyStart = syntax.Lazy(`^\s*[a-zA-Z-]+\s*:`)

func startsProperty(s string) bool { return propertyStart.MatchString(s) }

// classUse to przypisanie klasy albo stylu do elementu, rozstrzygane w Finish.
type classUse struct {
	target string
	class  string // nazwa klasy albo pusty napis, gdy to styl bezpośredni
	paint  Paint
	line   int
}

// DefineClass zapisuje definicję klasy (classDef).
func (g *Graph) DefineClass(names string, paint Paint) {
	if g.classes == nil {
		g.classes = map[string]Paint{}
	}
	for _, n := range strings.Split(names, ",") {
		if n = strings.TrimSpace(n); n != "" {
			g.classes[n] = g.classes[n].Merge(paint)
		}
	}
}

// AssignClass przypisuje klasę elementom wymienionym po przecinku (class, „:::”).
func (g *Graph) AssignClass(targets, class string, line int) {
	for _, t := range strings.Split(targets, ",") {
		if t = strings.TrimSpace(t); t != "" {
			g.uses = append(g.uses, classUse{target: t, class: strings.TrimSpace(class), line: line})
		}
	}
}

// SetStyle nadaje elementom styl bezpośredni (style); ma pierwszeństwo przed klasami.
func (g *Graph) SetStyle(targets string, paint Paint, line int) {
	for _, t := range strings.Split(targets, ",") {
		if t = strings.TrimSpace(t); t != "" {
			g.uses = append(g.uses, classUse{target: t, paint: paint, line: line})
		}
	}
}

// SetLinkStyle nadaje styl krawędziom o podanych numerach (od zera, w kolejności źródła);
// „default” dotyczy wszystkich.
func (g *Graph) SetLinkStyle(indexes string, paint Paint, line int) {
	g.linkUses = append(g.linkUses, classUse{target: indexes, paint: paint, line: line})
}

// resolvePaint rozstrzyga klasy i style po wczytaniu całego źródła: klasa „default” jest
// podstawą każdego węzła, potem klasy w kolejności przypisania, na końcu style bezpośrednie.
func (g *Graph) resolvePaint() error {
	base, hasBase := g.classes["default"]
	if hasBase {
		g.SourceFills = base.Fill != "" && base.Fill != "none"
		for _, n := range g.Nodes {
			n.Paint = base
		}
	}
	apply := func(u classUse, direct bool) error {
		paint := u.paint
		if !direct {
			var ok bool
			if paint, ok = g.classes[u.class]; !ok {
				return syntax.Errorf(u.line, CodeInvalid, "class %q is not defined with classDef", u.class)
			}
		}
		// Nieznany element pomijamy: klasa przypisana na zapas nie psuje rysunku.
		filled := paint.Fill != "" && paint.Fill != "none"
		if g.nodeIndex[u.target] != nil || g.subIndex[u.target] != nil {
			g.SourceFills = g.SourceFills || filled
		}
		if n := g.nodeIndex[u.target]; n != nil {
			n.Paint = n.Paint.Merge(paint)
		} else if s := g.subIndex[u.target]; s != nil {
			s.Paint = s.Paint.Merge(paint)
		}
		return nil
	}
	for _, direct := range []bool{false, true} {
		for _, u := range g.uses {
			if (u.class == "") != direct {
				continue
			}
			if err := apply(u, direct); err != nil {
				return err
			}
		}
	}
	for _, u := range g.linkUses {
		if strings.TrimSpace(u.target) == "default" {
			for _, e := range g.Edges {
				e.Paint = u.paint.Merge(e.Paint)
			}
			continue
		}
		for _, f := range strings.Split(u.target, ",") {
			i, err := strconv.Atoi(strings.TrimSpace(f))
			if err != nil || i < 0 || i >= len(g.Edges) {
				return syntax.Errorf(u.line, CodeInvalid, "linkStyle refers to edge %q; the diagram has %d edges numbered from 0", strings.TrimSpace(f), len(g.Edges))
			}
			g.Edges[i].Paint = g.Edges[i].Paint.Merge(u.paint)
		}
	}
	return nil
}

func (p Paint) dump() string {
	if p.IsZero() {
		return ""
	}
	var parts []string
	add := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	add("fill", p.Fill)
	add("stroke", p.Stroke)
	add("color", p.Color)
	if p.StrokeWidth != 0 {
		add("width", strconv.FormatFloat(p.StrokeWidth, 'f', -1, 64))
	}
	add("dash", strings.ReplaceAll(p.Dash, " ", ","))
	return " paint[" + strings.Join(parts, " ") + "]"
}
