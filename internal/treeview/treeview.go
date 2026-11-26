// Package treeview rozbiera i rysuje drzewo katalogów (treeView w Mermaid): hierarchię zapisaną
// wcięciami albo znakami ramek, tak jak wypisuje ją polecenie tree.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package treeview

import (
	"math"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni drzewa.
const (
	CodeHeader    = "treeview-header"
	CodeStatement = "treeview-statement"
	CodeEmpty     = "treeview-empty"
	maxNodes      = 2000
	maxDepth      = 32
)

// Node to plik albo katalog.
type Node struct {
	Label    string   // nazwa bez końcowego ukośnika
	Slash    bool     // nazwa zapisana z ukośnikiem na końcu (katalog)
	Classes  []string // klasy z :::nazwa
	Icon     string   // treść icon(...); "none" ukrywa ikonę
	Desc     string   // opis po ##
	Branch   bool     // linia zapisana znakami ramek (├── └── ┣━━ ┗━━)
	Children []*Node
	Line     int
}

// Dir mówi, czy węzeł jest katalogiem: zapisany z ukośnikiem albo mający zawartość.
func (n *Node) Dir() bool { return n.Slash || len(n.Children) > 0 }

// Highlighted mówi, czy węzeł ma wbudowaną klasę highlight.
func (n *Node) Highlighted() bool {
	for _, c := range n.Classes {
		if c == "highlight" {
			return true
		}
	}
	return false
}

// Tree to węzły najwyższego poziomu.
type Tree struct {
	Roots    []*Node
	AccTitle string
	AccDescr string
}

// isBranch mówi, czy znak zaczyna gałąź w zapisie znakami ramek.
func isBranch(r rune) bool { return r == '├' || r == '└' || r == '┣' || r == '┗' }

// isTrunk mówi, czy znak jest pionową kreską prowadzącą do dalszych gałęzi.
func isTrunk(r rune) bool { return r == '│' || r == '┃' }

// Parse rozbiera źródło. Poziom wyznacza kolumna: dla linii zwykłej pierwszego znaku nazwy, dla
// linii ze znakami ramek znaku gałęzi. Węzeł jest dzieckiem najbliższej wcześniejszej linii
// w mniejszej kolumnie, więc wynik polecenia tree (korzeń „.” i gałęzie od kolumny zero) też się
// rozbiera.
func Parse(source string) (*Tree, error) {
	t := &Tree{}
	var acc syntax.Accessibility
	seenHeader := false
	type open struct {
		col  int
		node *Node
		dep  int
	}
	var stack []open
	count := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		raw = strings.ReplaceAll(raw, "\t", "    ")
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "treeView-beta" && s != "treeView" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"treeView-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		// Kolumna i początek nazwy: spacje i pionowe kreski poprzedzają gałąź albo nazwę.
		runes := []rune(raw)
		col, start, branch := -1, -1, false
		for i := 0; i < len(runes); i++ {
			r := runes[i]
			if r == ' ' || isTrunk(r) {
				continue
			}
			if isBranch(r) {
				col, branch = i+1, true // o jeden w głąb: gałąź z kolumny zero należy do korzenia „.”
				j := i + 1
				for j < len(runes) && (runes[j] == '─' || runes[j] == '━') {
					j++
				}
				start = j
			} else {
				col, start = i, i
			}
			break
		}
		if col < 0 {
			continue // sama pionowa kreska: odstęp między gałęziami
		}
		rest := strings.TrimSpace(string(runes[start:]))
		if strings.HasPrefix(rest, "%%") {
			continue
		}
		node, err := parseNode(rest, line)
		if err != nil {
			return nil, err
		}
		node.Branch = branch
		if count++; count > maxNodes {
			return nil, syntax.Errorf(line, CodeStatement, "too many entries (limit %d)", maxNodes)
		}
		for len(stack) > 0 && stack[len(stack)-1].col >= col {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			t.Roots = append(t.Roots, node)
			stack = append(stack, open{col, node, 0})
			continue
		}
		parent := stack[len(stack)-1]
		if parent.dep+1 >= maxDepth {
			return nil, syntax.Errorf(line, CodeStatement, "entries nested too deep (limit %d levels)", maxDepth)
		}
		parent.node.Children = append(parent.node.Children, node)
		stack = append(stack, open{col, node, parent.dep + 1})
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"treeView-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(t.Roots) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the tree has no entries")
	}
	t.AccTitle, t.AccDescr = acc.Title, acc.Descr
	return t, nil
}

// parseNode rozbiera nazwę z dopiskami: nazwa (zwykła albo w cudzysłowie, ukośnik na końcu
// oznacza katalog), potem w dowolnej kolejności :::klasa i icon(...), a na końcu ## opis.
func parseNode(s string, line int) (*Node, error) {
	n := &Node{Line: line}
	var after string
	if strings.HasPrefix(s, `"`) {
		end := strings.IndexByte(s[1:], '"')
		if end < 0 {
			return nil, syntax.Errorf(line, CodeStatement, "quoted name is not closed: %q", s)
		}
		n.Label, after = s[1:1+end], s[2+end:]
		if strings.HasPrefix(after, "/") {
			n.Slash, after = true, after[1:]
		}
	} else {
		cut := len(s)
		if i := strings.Index(s, ":::"); i >= 0 {
			cut = i
		}
		for _, m := range []string{" icon(", " ##", "\t##"} {
			if i := strings.Index(s, m); i >= 0 && i < cut {
				cut = i
			}
		}
		n.Label, after = strings.TrimSpace(s[:cut]), s[cut:]
	}
	if strings.HasSuffix(n.Label, "/") && len(n.Label) > 1 {
		n.Label, n.Slash = strings.TrimSuffix(n.Label, "/"), true
	}
	if strings.TrimSpace(n.Label) == "" {
		return nil, syntax.Errorf(line, CodeStatement, "entry has no name: %q", s)
	}
	for {
		after = strings.TrimSpace(after)
		switch {
		case after == "":
			return n, nil
		case strings.HasPrefix(after, "##"):
			n.Desc = strings.TrimSpace(after[2:])
			return n, nil
		case strings.HasPrefix(after, ":::"):
			name := after[3:]
			if i := strings.IndexAny(name, " \t"); i >= 0 {
				name, after = name[:i], name[i:]
			} else {
				after = ""
			}
			if name == "" {
				return nil, syntax.Errorf(line, CodeStatement, "class name missing after \":::\"")
			}
			n.Classes = append(n.Classes, name)
		case strings.HasPrefix(after, "icon("):
			end := strings.IndexByte(after, ')')
			if end < 0 {
				return nil, syntax.Errorf(line, CodeStatement, "icon( is not closed with \")\"")
			}
			n.Icon, after = strings.TrimSpace(after[5:end]), after[end+1:]
			if n.Icon == "" {
				return nil, syntax.Errorf(line, CodeStatement, "icon() needs a name")
			}
		default:
			return nil, syntax.Errorf(line, CodeStatement, "unexpected text after the name: %q", after)
		}
	}
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // ikony i wyróżnienie w kolorach
	Shadows    bool // bez znaczenia: drzewo nie ma pól
}

const (
	colorText   = "#1b1f24"
	colorDesc   = "#55657a"
	colorLine   = "#8a96a3"
	colorMark   = "#eef2f7" // podkład wyróżnienia w trybie neutralnym
	colorMarkLn = "#55657a"
	colorIcon   = "#55657a"
)

// row to węzeł ułożony w rzędzie.
type row struct {
	n      *Node
	level  int
	parent int // indeks rzędu rodzica; -1 dla korzenia (-2: korzeń wiszący na pniu bez nazwy)
	x      float64
	mid    float64
	labelW float64
	icon   bool
}

// Layout rysuje drzewo: jeden rząd na węzeł, wcięcie na poziom, kreski od pnia rodzica do
// każdego dziecka, katalogi pogrubione, opisy pochylone w jednej kolumnie.
func Layout(t *Tree, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return math.Round(v * fs / 16) }
	lh := math.Round(fs * 1.25)
	rowH := lh + u(6)
	indent := u(22)
	iconS := math.Round(fs * 0.9)
	sc := &scene.Scene{Title: t.AccTitle, Desc: t.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	// Korzenie zapisane gałęziami (├── od kolumny zero) wiszą na wspólnym pniu bez nazwy.
	trunk := false
	for _, r := range t.Roots {
		trunk = trunk || r.Branch
	}
	icons := false
	var rows []*row
	var walk func(n *Node, level, parent int)
	walk = func(n *Node, level, parent int) {
		r := &row{n: n, level: level, parent: parent, icon: n.Icon != "" && n.Icon != "none"}
		icons = icons || r.icon
		rows = append(rows, r)
		me := len(rows) - 1
		for _, c := range n.Children {
			walk(c, level+1, me)
		}
	}
	for _, r := range t.Roots {
		if trunk {
			walk(r, 1, -2)
		} else {
			walk(r, 0, -1)
		}
	}
	iconOff := 0.0
	if icons {
		iconOff = iconS + u(6)
	}
	descX := 0.0
	for i, r := range rows {
		r.x = float64(r.level) * indent
		r.mid = float64(i)*rowH + rowH/2
		r.labelW = o.Measure(label(r.n), fs)
		if r.n.Dir() {
			r.labelW *= 1.06 // pogrubienie jest szersze od miary kroju zwykłego
		}
		if r.n.Desc != "" {
			descX = math.Max(descX, r.x+iconOff+r.labelW+u(24))
		}
	}
	// Kreski: pionowa od rodzica do ostatniego dziecka, pozioma do każdego dziecka.
	stemX := func(parent int) float64 {
		if parent < 0 {
			return u(7)
		}
		return rows[parent].x + u(7)
	}
	// Ostatnie dziecko każdego rodzica; pień bez nazwy ma miejsce pod indeksem len(rows).
	last := make([]int, len(rows)+1)
	for i := range last {
		last[i] = -1
	}
	for i, r := range rows {
		switch {
		case r.parent >= 0:
			last[r.parent] = i
		case r.parent == -2:
			last[len(rows)] = i
		}
	}
	if l := last[len(rows)]; l >= 0 {
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "treeview-line", X: stemX(-2), Y: 0, X2: stemX(-2), Y2: rows[l].mid, Stroke: colorLine, StrokeWidth: 1})
	}
	for p, r := range rows {
		if l := last[p]; l >= 0 {
			x := stemX(p)
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "treeview-line", X: x, Y: r.mid + lh/2, X2: x, Y2: rows[l].mid, Stroke: colorLine, StrokeWidth: 1})
		}
	}
	for _, r := range rows {
		if r.parent != -1 {
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "treeview-line", X: stemX(r.parent), Y: r.mid, X2: r.x - u(4), Y2: r.mid, Stroke: colorLine, StrokeWidth: 1})
		}
	}

	for _, r := range rows {
		lx := r.x + iconOff
		if r.n.Highlighted() {
			fill, stroke := colorMark, colorMarkLn
			if o.Colors {
				fill, stroke = "#fff3cd", "#d9a300"
			}
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "treeview-highlight", X: lx - u(4), Y: r.mid - lh/2, W: r.labelW + u(8), H: lh, RX: u(3),
				Fill: fill, Stroke: stroke, StrokeWidth: 1})
		}
		if r.icon {
			addIcon(sc, r.n, r.x, r.mid, iconS, o.Colors)
		}
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treeview-label", X: lx, Y: r.mid + 0.35*fs, W: r.labelW, H: lh,
			Text: label(r.n), Anchor: "start", Bold: r.n.Dir(), Fill: colorText})
		if r.n.Desc != "" {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "treeview-description", X: descX, Y: r.mid + 0.35*fs, W: o.Measure(r.n.Desc, fs), H: lh,
				Text: r.n.Desc, Anchor: "start", Italic: true, Fill: colorDesc})
		}
	}
	return sc
}

// label to nazwa do narysowania: katalog zapisany z ukośnikiem zachowuje go.
func label(n *Node) string {
	if n.Slash {
		return n.Label + "/"
	}
	return n.Label
}

// addIcon rysuje własną ikonę katalogu albo pliku. Zestawów ikon (icon(pakiet:nazwa)) nie ma
// w bibliotece, więc każda ikona inna niż folder albo file to ikona według rodzaju węzła.
func addIcon(sc *scene.Scene, n *Node, x, mid, s float64, colors bool) {
	folder := n.Icon == "folder" || (n.Icon != "file" && n.Dir())
	top := mid - s/2
	stroke, fill := colorIcon, "none"
	if colors {
		if folder {
			stroke, fill = "#b07d12", "#fbe3a6"
		} else {
			stroke, fill = "#3b6fb6", "#e8f0fb"
		}
	}
	p := func(fx, fy float64) scene.Point {
		return scene.Point{X: math.Round((x+fx*s)*100) / 100, Y: math.Round((top+fy*s)*100) / 100}
	}
	if folder {
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "treeview-icon",
			Points: []scene.Point{p(0, 0.15), p(0.4, 0.15), p(0.5, 0.3), p(1, 0.3), p(1, 0.9), p(0, 0.9)}, Fill: fill, Stroke: stroke, StrokeWidth: 1})
		return
	}
	sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "treeview-icon",
		Points: []scene.Point{p(0.15, 0), p(0.6, 0), p(0.85, 0.25), p(0.85, 1), p(0.15, 1)}, Fill: fill, Stroke: stroke, StrokeWidth: 1})
	sc.Add(scene.Item{Kind: scene.Polyline, Layer: scene.LayerShape, Role: "treeview-icon",
		Points: []scene.Point{p(0.6, 0), p(0.6, 0.25), p(0.85, 0.25)}, Fill: "none", Stroke: stroke, StrokeWidth: 1})
}
