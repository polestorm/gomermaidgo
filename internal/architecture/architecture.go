// Package architecture rozbiera i rysuje diagram architektury (architecture): usługi z ikonami
// w grupach, połączone krawędziami wychodzącymi ze wskazanych boków.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package architecture

import (
	"math"
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni diagramu architektury.
const (
	CodeHeader    = "architecture-header"
	CodeStatement = "architecture-statement"
	CodeNode      = "architecture-node"
	CodeEdge      = "architecture-edge"
	CodeEmpty     = "architecture-empty"
	maxNodes      = 100
)

// Node to usługa albo węzeł rozgałęzienia (Junction).
type Node struct {
	ID       string
	Icon     string
	Label    string
	Group    string
	Junction bool
	Line     int
}

// Group to grupa usług; Parent to grupa nadrzędna albo pusty napis.
type Group struct {
	ID, Icon, Label, Parent string
	Line                    int
}

// Edge to połączenie: z boku FromSide węzła From do boku ToSide węzła To. Boki: L, R, T, B.
// FromGroup i ToGroup mówią, że koniec dotyczy grupy węzła, a nie samego węzła.
type Edge struct {
	From, To           string
	FromSide, ToSide   string
	FromGroup, ToGroup bool
	FromArrow, ToArrow bool
	Label              string
	Line               int
}

// Diagram to diagram architektury.
type Diagram struct {
	Title    string
	Groups   []*Group
	Nodes    []*Node
	Edges    []Edge
	AccTitle string
	AccDescr string
}

const id = `[\p{L}\p{N}_][\p{L}\p{N}_-]*`

var (
	declare = regexp.MustCompile(`^(group|service)\s+(` + id + `)\s*(?:\(([^)]*)\))?\s*(?:\[([^\]]*)\])?\s*(?:in\s+(` + id + `))?$`)
	junct   = regexp.MustCompile(`^junction\s+(` + id + `)\s*(?:in\s+(` + id + `))?$`)
	link    = regexp.MustCompile(`^(` + id + `)(\{group\})?\s*:\s*([LRTB])\s*(<)?-(?:\[([^\]]*)\])?-(>)?\s*([LRTB])\s*:\s*(` + id + `)(\{group\})?$`)
)

// Parse rozbiera źródło diagramu architektury.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	seenHeader := false
	groups, nodes := map[string]*Group{}, map[string]*Node{}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "architecture-beta" && s != "architecture" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"architecture-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		if s == "title" || strings.HasPrefix(s, "title ") {
			d.Title = strings.TrimSpace(strings.TrimPrefix(s, "title"))
			continue
		}
		taken := func(name string) error {
			if groups[name] != nil || nodes[name] != nil {
				return syntax.Errorf(line, CodeNode, "%q is defined twice", name)
			}
			return nil
		}
		parent := func(name string) error {
			if name != "" && groups[name] == nil {
				return syntax.Errorf(line, CodeNode, "unknown group %q", name)
			}
			return nil
		}
		if m := declare.FindStringSubmatch(s); m != nil {
			if err := taken(m[2]); err != nil {
				return nil, err
			}
			if err := parent(m[5]); err != nil {
				return nil, err
			}
			label := strings.Trim(strings.TrimSpace(m[4]), `"`)
			if label == "" {
				label = m[2]
			}
			if m[1] == "group" {
				g := &Group{ID: m[2], Icon: strings.TrimSpace(m[3]), Label: label, Parent: m[5], Line: line}
				groups[g.ID] = g
				d.Groups = append(d.Groups, g)
				continue
			}
			if len(d.Nodes) >= maxNodes {
				return nil, syntax.Errorf(line, CodeNode, "too many services (limit %d)", maxNodes)
			}
			nd := &Node{ID: m[2], Icon: strings.TrimSpace(m[3]), Label: label, Group: m[5], Line: line}
			nodes[nd.ID] = nd
			d.Nodes = append(d.Nodes, nd)
			continue
		}
		if m := junct.FindStringSubmatch(s); m != nil {
			if err := taken(m[1]); err != nil {
				return nil, err
			}
			if err := parent(m[2]); err != nil {
				return nil, err
			}
			if len(d.Nodes) >= maxNodes {
				return nil, syntax.Errorf(line, CodeNode, "too many services (limit %d)", maxNodes)
			}
			nd := &Node{ID: m[1], Group: m[2], Junction: true, Line: line}
			nodes[nd.ID] = nd
			d.Nodes = append(d.Nodes, nd)
			continue
		}
		if m := link.FindStringSubmatch(s); m != nil {
			e := Edge{From: m[1], FromGroup: m[2] != "", FromSide: m[3], FromArrow: m[4] != "", Label: strings.TrimSpace(m[5]),
				ToArrow: m[6] != "", ToSide: m[7], To: m[8], ToGroup: m[9] != "", Line: line}
			for _, end := range []string{e.From, e.To} {
				if nodes[end] == nil {
					return nil, syntax.Errorf(line, CodeEdge, "edge refers to unknown service %q", end)
				}
			}
			if e.From == e.To {
				return nil, syntax.Errorf(line, CodeEdge, "edge from %q to itself", e.From)
			}
			if (e.FromGroup && nodes[e.From].Group == "") || (e.ToGroup && nodes[e.To].Group == "") {
				return nil, syntax.Errorf(line, CodeEdge, "{group} needs a service that lies in a group")
			}
			d.Edges = append(d.Edges, e)
			continue
		}
		return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"architecture-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(d.Nodes) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the diagram needs at least one service")
	}
	d.AccTitle, d.AccDescr = acc.Title, acc.Descr
	return d, nil
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // grupy najwyższego poziomu w kolejnych tonach palety
	Shadows    bool // cień pod ikonami usług
}

const (
	colorText   = "#1b1f24"
	colorFill   = "#eef2f7"
	colorStroke = "#55657a"
	colorLine   = "#3b4754"
	colorFrame  = "#9aa5b1"
)

var delta = map[string][2]int{"L": {-1, 0}, "R": {1, 0}, "T": {0, -1}, "B": {0, 1}}

// Grid rozmieszcza węzły w siatce: bok, z którego wychodzi krawędź, mówi, po której stronie
// sąsiada stoi drugi koniec. Zwraca kolumnę i wiersz każdego węzła, od zera.
func Grid(d *Diagram) map[string][2]int {
	type hop struct {
		to   string
		step [2]int
	}
	next := map[string][]hop{}
	for _, e := range d.Edges {
		a, b := delta[e.FromSide], delta[e.ToSide]
		// Boki na różnych osiach (np. z lewej do góry) znaczą, że drugi koniec stoi po skosie:
		// w lewo i niżej. Boki na jednej osi dają zwykłe sąsiedztwo.
		if (a[0] == 0) != (b[0] == 0) {
			a, b = [2]int{a[0] - b[0], a[1] - b[1]}, [2]int{b[0] - a[0], b[1] - a[1]}
		}
		next[e.From] = append(next[e.From], hop{e.To, a})
		next[e.To] = append(next[e.To], hop{e.From, b})
	}
	pos := map[string][2]int{}
	came := map[string][2]int{} // kierunek, w którym węzeł został dostawiony do sąsiada
	used := map[[2]int]bool{}
	maxRow := -1
	for _, start := range d.Nodes {
		if _, done := pos[start.ID]; done {
			continue
		}
		// Kolejna część niepołączona z poprzednimi zaczyna się pod nimi.
		origin := [2]int{0, maxRow + 1}
		pos[start.ID], used[origin] = origin, true
		queue := []string{start.ID}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, h := range next[cur] {
				if _, done := pos[h.to]; done {
					continue
				}
				p := [2]int{pos[cur][0] + h.step[0], pos[cur][1] + h.step[1]}
				for used[p] { // zajęte pole: dalej w tym samym kierunku
					p = [2]int{p[0] + h.step[0], p[1] + h.step[1]}
				}
				pos[h.to], used[p], came[h.to] = p, true, h.step
				queue = append(queue, h.to)
			}
		}
		// Część mogła urosnąć w górę: przesuwamy ją pod poprzednie.
		maxRow = shift(d, pos, used, start.ID, maxRow)
	}
	// Ramka grupy jest prostokątem wokół jej usług: usługa spoza grupy, która wypadła w jego
	// obrębie, odsuwa się dalej w kierunku, w którym została dostawiona.
	member := func(n *Node, group string) bool {
		for g := n.Group; g != ""; {
			if g == group {
				return true
			}
			parent := ""
			for _, c := range d.Groups {
				if c.ID == g {
					parent = c.Parent
				}
			}
			g = parent
		}
		return false
	}
	for round := 0; round < 4; round++ {
		moved := false
		for _, g := range d.Groups {
			x0, y0, x1, y1 := math.MaxInt, math.MaxInt, math.MinInt, math.MinInt
			for _, n := range d.Nodes {
				if member(n, g.ID) {
					p := pos[n.ID]
					x0, y0, x1, y1 = min(x0, p[0]), min(y0, p[1]), max(x1, p[0]), max(y1, p[1])
				}
			}
			for _, n := range d.Nodes {
				step, ok := came[n.ID]
				if !ok || member(n, g.ID) {
					continue
				}
				p := pos[n.ID]
				inside := func(q [2]int) bool { return q[0] >= x0 && q[0] <= x1 && q[1] >= y0 && q[1] <= y1 }
				if !inside(p) {
					continue
				}
				delete(used, p)
				for inside(p) || used[p] {
					p = [2]int{p[0] + step[0], p[1] + step[1]}
				}
				pos[n.ID], used[p], moved = p, true, true
			}
		}
		if !moved {
			break
		}
	}
	minX := math.MaxInt
	for _, p := range pos {
		minX = min(minX, p[0])
	}
	minY := math.MaxInt
	for _, p := range pos {
		minY = min(minY, p[1])
	}
	for id, p := range pos {
		pos[id] = [2]int{p[0] - minX, p[1] - minY}
	}
	return pos
}

// shift przesuwa w dół część zaczynającą się w start, jeśli wyszła ponad wiersz, od którego
// miała się zaczynać, i zwraca najniższy zajęty wiersz.
func shift(d *Diagram, pos map[string][2]int, used map[[2]int]bool, start string, floor int) int {
	reach := map[string]bool{}
	adj := map[string][]string{}
	for _, e := range d.Edges {
		adj[e.From] = append(adj[e.From], e.To)
		adj[e.To] = append(adj[e.To], e.From)
	}
	stack := []string{start}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if reach[cur] {
			continue
		}
		reach[cur] = true
		stack = append(stack, adj[cur]...)
	}
	top := math.MaxInt
	for id := range reach {
		top = min(top, pos[id][1])
	}
	if move := floor + 1 - top; move > 0 {
		for id := range reach {
			p := pos[id]
			delete(used, p)
		}
		for id := range reach {
			p := [2]int{pos[id][0], pos[id][1] + move}
			pos[id], used[p] = p, true
		}
	}
	bottom := floor
	for _, p := range pos {
		bottom = max(bottom, p[1])
	}
	return bottom
}

// Layout rozmieszcza diagram: usługi w siatce wyznaczonej przez boki krawędzi, każda jako ikona
// z podpisem; grupy jako ramki wokół swoich usług; krawędzie między wskazanymi bokami.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	width := func(s string) float64 { return o.Measure(s, fs) }
	title := d.AccTitle
	if title == "" {
		title = d.Title
	}
	sc := &scene.Scene{Title: title, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	grid := Grid(d)
	cellW, cellH, icon := u(170), u(164), u(56)
	y0 := 0.0
	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "title", X: 0, Y: lh/2 + 0.35*fs, W: width(d.Title), H: lh, Text: d.Title, Anchor: "start", Fill: colorText})
		y0 = lh + u(14)
	}
	centre := func(id string) (float64, float64) {
		p := grid[id]
		return float64(p[0])*cellW + cellW/2, y0 + float64(p[1])*cellH + u(48) + icon/2
	}

	// Ramki grup: pole obejmujące usługi grupy i ramki grup podrzędnych, z marginesem.
	byID := map[string]*Node{}
	for _, n := range d.Nodes {
		byID[n.ID] = n
	}
	type frame struct{ x0, y0, x1, y1 float64 }
	frames := map[string]*frame{}
	depth := map[string]int{}
	var extent func(g *Group) *frame
	extent = func(g *Group) *frame {
		if f, ok := frames[g.ID]; ok {
			return f
		}
		f := &frame{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		grow := func(x0, y0, x1, y1 float64) {
			f.x0, f.y0, f.x1, f.y1 = math.Min(f.x0, x0), math.Min(f.y0, y0), math.Max(f.x1, x1), math.Max(f.y1, y1)
		}
		for _, n := range d.Nodes {
			if n.Group == g.ID {
				cx, cy := centre(n.ID)
				// Nad ikoną miejsce na podpis grupy i odstęp pod nim.
				grow(cx-cellW/2+u(8), cy-icon/2-u(42), cx+cellW/2-u(8), cy+icon/2+2*lh+u(10))
			}
		}
		for _, child := range d.Groups {
			if child.Parent == g.ID {
				c := extent(child)
				depth[g.ID] = max(depth[g.ID], depth[child.ID]+1)
				if !math.IsInf(c.x0, 1) {
					grow(c.x0-u(8), c.y0-u(8), c.x1+u(8), c.y1+u(8))
				}
			}
		}
		frames[g.ID] = f
		return f
	}
	for i, g := range d.Groups {
		f := extent(g)
		if math.IsInf(f.x0, 1) {
			continue // grupa bez usług nie ma ramki
		}
		fill, stroke := "", colorFrame
		if o.Colors && g.Parent == "" {
			t := scene.Distinct(i)
			fill, stroke = t.Fill, t.Stroke
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "architecture-group", X: f.x0, Y: f.y0, W: f.x1 - f.x0, H: f.y1 - f.y0, RX: u(6),
			Fill: fill, Stroke: stroke, StrokeWidth: 1.5, Dash: "6 4"})
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "architecture-group-label", X: f.x0 + u(10), Y: f.y0 + u(4) + lh/2 + 0.35*fs, W: width(g.Label), H: lh,
			Text: g.Label, Anchor: "start", Bold: true, Fill: colorText})
	}

	// Krawędzie: od środka boku ikony (albo ramki grupy) do środka boku drugiego końca.
	attach := func(id, side string, group bool) (float64, float64) {
		cx, cy := centre(id)
		n := byID[id]
		half := icon / 2
		if n.Junction {
			half = 0
		}
		x, y := cx+float64(delta[side][0])*half, cy+float64(delta[side][1])*half
		if group {
			f := frames[n.Group]
			switch side {
			case "L":
				x = f.x0
			case "R":
				x = f.x1
			case "T":
				y = f.y0
			case "B":
				y = f.y1
			}
		}
		return x, y
	}
	head := func(x, y float64, side string) {
		// Grot wskazuje w stronę węzła: przeciwnie do boku, z którego krawędź wychodzi.
		dx, dy := -float64(delta[side][0]), -float64(delta[side][1])
		a, h := u(10), u(5)
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerActivation, Role: "arrowhead", Fill: colorLine, Points: []scene.Point{
			{X: x, Y: y}, {X: x - dx*a - dy*h, Y: y - dy*a + dx*h}, {X: x - dx*a + dy*h, Y: y - dy*a - dx*h}}})
	}
	for _, e := range d.Edges {
		ax, ay := attach(e.From, e.FromSide, e.FromGroup)
		bx, by := attach(e.To, e.ToSide, e.ToGroup)
		horizontalA, horizontalB := e.FromSide == "L" || e.FromSide == "R", e.ToSide == "L" || e.ToSide == "R"
		pts := []scene.Point{{X: ax, Y: ay}}
		switch {
		case horizontalA && horizontalB && math.Abs(ay-by) > 0.5:
			mid := (ax + bx) / 2
			pts = append(pts, scene.Point{X: mid, Y: ay}, scene.Point{X: mid, Y: by})
		case !horizontalA && !horizontalB && math.Abs(ax-bx) > 0.5:
			mid := (ay + by) / 2
			pts = append(pts, scene.Point{X: ax, Y: mid}, scene.Point{X: bx, Y: mid})
		case horizontalA && !horizontalB:
			pts = append(pts, scene.Point{X: bx, Y: ay})
		case !horizontalA && horizontalB:
			pts = append(pts, scene.Point{X: ax, Y: by})
		}
		pts = append(pts, scene.Point{X: bx, Y: by})
		sc.Add(scene.Item{Kind: scene.Polyline, Layer: scene.LayerLifeline, Role: "edge", Points: pts, Stroke: colorLine, StrokeWidth: 1.5})
		if e.FromArrow {
			head(ax, ay, e.FromSide)
		}
		if e.ToArrow {
			head(bx, by, e.ToSide)
		}
		if e.Label != "" {
			// Etykieta pośrodku najdłuższego odcinka, na własnym podkładzie.
			best, at := 0.0, pts[0]
			for i := 1; i < len(pts); i++ {
				if l := math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y); l > best {
					best, at = l, scene.Point{X: (pts[i].X + pts[i-1].X) / 2, Y: (pts[i].Y + pts[i-1].Y) / 2}
				}
			}
			w := width(e.Label)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "edge-label-background", X: at.X - w/2 - u(3), Y: at.Y - lh/2, W: w + u(6), H: lh, Knockout: true})
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "edge-label", X: at.X, Y: at.Y + 0.35*fs, W: w, H: lh, Text: e.Label, Anchor: "middle", Fill: colorText})
		}
	}

	from := len(sc.Items)
	for _, n := range d.Nodes {
		cx, cy := centre(n.ID)
		if n.Junction {
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "architecture-junction", X: cx, Y: cy, R: u(4), Fill: colorLine})
			continue
		}
		x, y := cx-icon/2, cy-icon/2
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "architecture-service", X: x, Y: y, W: icon, H: icon, RX: u(8), Fill: colorFill, Stroke: colorStroke, StrokeWidth: 1.5})
		pictogram(sc, n.Icon, cx, cy, icon*0.3)
		for k, s := range textwrap.Lines([]string{n.Label}, cellW-u(24), width) {
			// Podkład pod podpisem: krawędź wychodząca z dolnego boku ikony nie przekreśla tekstu.
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "architecture-label-background", X: cx - width(s)/2 - u(2), Y: y + icon + u(4) + float64(k)*lh,
				W: width(s) + u(4), H: lh, Knockout: true})
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "architecture-label", X: cx, Y: y + icon + u(4) + float64(k)*lh + lh/2 + 0.35*fs,
				W: width(s), H: lh, Text: s, Anchor: "middle", Fill: colorText})
		}
	}
	if o.Shadows {
		sc.AddShadows(from, u(3), "architecture-service")
	}
	return sc
}

// pictogram rysuje uproszczony znak ikony w kwadracie o środku (cx, cy) i połowie boku r.
// Znane ikony: database, server, disk, cloud, internet; inne zostają pustym polem.
func pictogram(sc *scene.Scene, name string, cx, cy, r float64) {
	line := func(x0, y0, x1, y1 float64) {
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabel, Role: "architecture-icon", X: cx + x0*r, Y: cy + y0*r, X2: cx + x1*r, Y2: cy + y1*r, Stroke: colorLine, StrokeWidth: 1.5})
	}
	ellipse := func(x, y, w, h float64, fill string) {
		sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerLabel, Role: "architecture-icon", X: cx + x*r, Y: cy + y*r, W: w * r, H: h * r, Fill: fill, Stroke: colorLine, StrokeWidth: 1.5})
	}
	rect := func(x, y, w, h float64) {
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabel, Role: "architecture-icon", X: cx + x*r, Y: cy + y*r, W: w * r, H: h * r, RX: r * 0.12, Fill: "#ffffff", Stroke: colorLine, StrokeWidth: 1.5})
	}
	switch name {
	case "database":
		ellipse(0, 0.65, 1.7, 0.6, "#ffffff")
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabel, Role: "architecture-icon", X: cx - 0.85*r, Y: cy - 0.65*r, W: 1.7 * r, H: 1.3 * r, Fill: "#ffffff"})
		line(-0.85, -0.65, -0.85, 0.65)
		line(0.85, -0.65, 0.85, 0.65)
		ellipse(0, -0.65, 1.7, 0.6, "#ffffff")
	case "server":
		rect(-0.9, -0.85, 1.8, 0.75)
		rect(-0.9, 0.1, 1.8, 0.75)
		for _, y := range []float64{-0.475, 0.475} {
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "architecture-icon", X: cx - 0.55*r, Y: cy + y*r, R: r * 0.1, Fill: colorLine})
		}
	case "disk":
		ellipse(0, 0, 1.8, 1.8, "#ffffff")
		ellipse(0, 0, 0.5, 0.5, colorLine)
	case "internet":
		ellipse(0, 0, 1.8, 1.8, "#ffffff")
		ellipse(0, 0, 0.8, 1.8, "")
		line(-0.9, 0, 0.9, 0)
	case "cloud":
		// Obrys chmury: płaska podstawa i trzy wybrzuszenia z krzywych.
		at := func(x, y float64) scene.Point { return scene.Point{X: cx + x*r, Y: cy + y*r} }
		sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLabel, Role: "architecture-icon", Fill: "#ffffff", Stroke: colorLine, StrokeWidth: 1.5, Segments: []scene.Segment{
			{Cmd: 'M', To: at(-0.6, 0.6)},
			{Cmd: 'C', C1: at(-1.2, 0.6), C2: at(-1.2, -0.2), To: at(-0.6, -0.15)},
			{Cmd: 'C', C1: at(-0.6, -0.9), C2: at(0.3, -0.95), To: at(0.4, -0.35)},
			{Cmd: 'C', C1: at(1.15, -0.45), C2: at(1.2, 0.6), To: at(0.6, 0.6)},
			{Cmd: 'L', To: at(-0.6, 0.6)},
		}})
	}
}
