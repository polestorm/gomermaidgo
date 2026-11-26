// Package venn rozbiera i rysuje diagram Venna: zbiory jako koła i ich części wspólne.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package venn

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni diagramu Venna.
const (
	CodeHeader    = "venn-header"
	CodeStatement = "venn-statement"
	CodeEmpty     = "venn-empty"
	maxSets       = 3
	maxTexts      = 60
)

// Text to element wypisany w obszarze zbioru albo części wspólnej.
type Text struct {
	ID, Label string
	Paint     graph.Paint
}

// Area to zbiór (jedno Id) albo część wspólna (kilka Id).
type Area struct {
	IDs     []string
	Label   string // pusty: zbiór podpisany swoim Id, część wspólna bez podpisu
	Size    float64
	HasSize bool
	Texts   []*Text
	Paint   graph.Paint
	Opacity float64 // fill-opacity ze stylu; 0 = domyślne
	Line    int
}

// Diagram to zbiory i części wspólne w kolejności źródła.
type Diagram struct {
	Title    string
	Sets     []*Area
	Unions   []*Area
	AccTitle string
	AccDescr string
}

// Parse rozbiera źródło: set, union, wcięte text pod ostatnim zbiorem albo częścią wspólną,
// style i title.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	seenHeader := false
	sets := map[string]*Area{}
	texts := map[string]*Text{}
	var last *Area
	count := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "venn-beta" && s != "venn" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"venn-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		word, rest, _ := strings.Cut(s, " ")
		rest = strings.TrimSpace(rest)
		switch word {
		case "title":
			d.Title = unquote(rest)
		case "set":
			ids, label, size, has, err := parseArea(rest, line)
			if err != nil {
				return nil, err
			}
			if len(ids) != 1 {
				return nil, syntax.Errorf(line, CodeStatement, "set takes one name; use union for an overlap")
			}
			if sets[ids[0]] != nil {
				return nil, syntax.Errorf(line, CodeStatement, "set %q is already defined", ids[0])
			}
			if len(d.Sets) == maxSets {
				return nil, syntax.Errorf(line, CodeStatement, "more than %d sets are outside the profile", maxSets)
			}
			a := &Area{IDs: ids, Label: label, Size: size, HasSize: has, Line: line}
			sets[ids[0]] = a
			d.Sets = append(d.Sets, a)
			last = a
		case "union":
			ids, label, size, has, err := parseArea(rest, line)
			if err != nil {
				return nil, err
			}
			if len(ids) < 2 {
				return nil, syntax.Errorf(line, CodeStatement, "union needs at least two sets")
			}
			for _, id := range ids {
				if sets[id] == nil {
					return nil, syntax.Errorf(line, CodeStatement, "union uses %q before its set line", id)
				}
			}
			if findUnion(d, ids) != nil {
				return nil, syntax.Errorf(line, CodeStatement, "union %s is already defined", strings.Join(ids, ","))
			}
			a := &Area{IDs: ids, Label: label, Size: size, HasSize: has, Line: line}
			d.Unions = append(d.Unions, a)
			last = a
		case "text":
			if last == nil {
				return nil, syntax.Errorf(line, CodeStatement, "text must follow a set or union")
			}
			ids, label, _, has, err := parseArea(rest, line)
			if err != nil {
				return nil, err
			}
			if len(ids) != 1 || has {
				return nil, syntax.Errorf(line, CodeStatement, "text takes one name and a label")
			}
			if texts[ids[0]] != nil || sets[ids[0]] != nil {
				return nil, syntax.Errorf(line, CodeStatement, "name %q is already used", ids[0])
			}
			if count++; count > maxTexts {
				return nil, syntax.Errorf(line, CodeStatement, "too many text lines (limit %d)", maxTexts)
			}
			if label == "" {
				label = ids[0]
			}
			t := &Text{ID: ids[0], Label: label}
			texts[ids[0]] = t
			last.Texts = append(last.Texts, t)
		case "style":
			target, props := cutOutsideQuotes(rest)
			p, err := graph.ParsePaint(props, line)
			if err != nil {
				return nil, err
			}
			opacity := 0.0
			for _, item := range strings.Split(props, ",") {
				name, value, _ := strings.Cut(item, ":")
				if strings.TrimSpace(strings.ToLower(name)) == "fill-opacity" {
					v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
					if err != nil || v < 0 || v > 1 {
						return nil, syntax.Errorf(line, CodeStatement, "fill-opacity must be a number from 0 to 1")
					}
					opacity = math.Max(v, 0.001)
				}
			}
			ids := splitIDs(target)
			switch {
			case len(ids) > 1 && findUnion(d, ids) != nil:
				u := findUnion(d, ids)
				u.Paint, u.Opacity = u.Paint.Merge(p), pick(opacity, u.Opacity)
			case len(ids) == 1 && sets[ids[0]] != nil:
				a := sets[ids[0]]
				a.Paint, a.Opacity = a.Paint.Merge(p), pick(opacity, a.Opacity)
			case len(ids) == 1 && texts[ids[0]] != nil:
				texts[ids[0]].Paint = texts[ids[0]].Paint.Merge(p)
			default:
				return nil, syntax.Errorf(line, CodeStatement, "style target %q is not a set, union or text", target)
			}
		default:
			return nil, syntax.Errorf(line, CodeStatement, "unknown statement %q", s)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"venn-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(d.Sets) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the diagram needs at least one set")
	}
	d.AccTitle, d.AccDescr = acc.Title, acc.Descr
	return d, nil
}

// cutOutsideQuotes dzieli tekst na pierwszej spacji poza cudzysłowem.
func cutOutsideQuotes(s string) (string, string) {
	quoted := false
	for i, r := range s {
		switch {
		case r == '"':
			quoted = !quoted
		case r == ' ' && !quoted:
			return s[:i], strings.TrimSpace(s[i+1:])
		}
	}
	return s, ""
}

func pick(v, old float64) float64 {
	if v > 0 {
		return v
	}
	return old
}

// parseArea czyta „Id[,Id…][\"podpis\"][:wielkość]”.
func parseArea(s string, line int) (ids []string, label string, size float64, has bool, err error) {
	if i := strings.LastIndex(s, ":"); i >= 0 && !strings.Contains(s[i:], `"`) && !strings.Contains(s[i:], "]") {
		v, e := strconv.ParseFloat(strings.TrimSpace(s[i+1:]), 64)
		if e != nil || v <= 0 {
			return nil, "", 0, false, syntax.Errorf(line, CodeStatement, "size after \":\" must be a positive number")
		}
		size, has, s = v, true, strings.TrimSpace(s[:i])
	}
	if i := strings.Index(s, "["); i >= 0 {
		if !strings.HasSuffix(s, "]") {
			return nil, "", 0, false, syntax.Errorf(line, CodeStatement, "label must be closed with \"]\"")
		}
		label = unquote(strings.TrimSpace(s[i+1 : len(s)-1]))
		s = strings.TrimSpace(s[:i])
	}
	ids = splitIDs(s)
	for _, id := range ids {
		if id == "" {
			return nil, "", 0, false, syntax.Errorf(line, CodeStatement, "empty name in %q", s)
		}
	}
	if len(ids) == 0 {
		return nil, "", 0, false, syntax.Errorf(line, CodeStatement, "a name is missing")
	}
	return ids, label, size, has, nil
}

// splitIDs dzieli listę nazw po przecinkach; nazwa może być w cudzysłowie.
func splitIDs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		out = append(out, unquote(strings.TrimSpace(p)))
	}
	return out
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// findUnion zwraca część wspólną tych samych zbiorów, bez względu na kolejność nazw.
func findUnion(d *Diagram, ids []string) *Area {
	key := func(ids []string) string {
		c := append([]string(nil), ids...)
		sort.Strings(c)
		return strings.Join(c, "\x00")
	}
	k := key(ids)
	for _, u := range d.Unions {
		if key(u.IDs) == k {
			return u
		}
	}
	return nil
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // koła w tonach palety
	Shadows    bool // bez znaczenia: koła są przezroczyste
}

const (
	colorText   = "#1b1f24"
	colorStroke = "#3b4754"
	colorFill   = "#55657a"
	colorItem   = "#3b4754"
)

type circle struct{ x, y, r float64 }

func (c circle) contains(x, y float64) bool { return math.Hypot(x-c.x, y-c.y) <= c.r+1e-6 }

// Layout rysuje koła: promień według wielkości zbioru (pole koła proporcjonalne do wielkości),
// odległość środków według wielkości części wspólnej, gdy jest podana; zbiory bez części
// wspólnej stoją osobno. Podpisy stoją w obszarach, do których należą.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return math.Round(v * fs / 16) }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.85)
	sc := &scene.Scene{Title: d.AccTitle, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	// Promienie: największy zbiór ma promień R, pozostałe według pierwiastka z wielkości.
	R := u(120)
	maxSize := 0.0
	for _, s := range d.Sets {
		if s.HasSize {
			maxSize = math.Max(maxSize, s.Size)
		}
	}
	size := func(a *Area) float64 {
		if a.HasSize {
			return a.Size
		}
		if maxSize > 0 {
			return maxSize
		}
		return 1
	}
	if maxSize == 0 {
		maxSize = 1
	}
	cs := make([]circle, len(d.Sets))
	for i, s := range d.Sets {
		cs[i].r = R * math.Sqrt(math.Min(size(s), maxSize)/maxSize)
		cs[i].r = math.Max(cs[i].r, u(40))
	}
	// Odległości par: część wspólna z wielkością daje pole soczewki, bez wielkości odległość
	// równą średniej promieni; para bez wspólnej części stoi osobno.
	index := map[string]int{}
	for i, s := range d.Sets {
		index[s.IDs[0]] = i
	}
	overlap := func(i, j int) (*Area, bool) {
		var pair *Area
		any := false
		for _, un := range d.Unions {
			in := 0
			for _, id := range un.IDs {
				if index[id] == i || index[id] == j {
					in++
				}
			}
			if in == 2 {
				any = true
				if len(un.IDs) == 2 {
					pair = un
				}
			}
		}
		return pair, any
	}
	dist := func(i, j int) float64 {
		a, b := cs[i].r, cs[j].r
		pair, any := overlap(i, j)
		if !any {
			return a + b + u(30)
		}
		if pair != nil && pair.HasSize {
			unit := math.Pi * R * R / maxSize
			return lensDistance(a, b, math.Min(pair.Size*unit, 0.95*math.Pi*math.Min(a, b)*math.Min(a, b)))
		}
		return (a + b) / 2
	}
	// Zbiory połączone częściami wspólnymi stoją razem (dwa w rzędzie, trzy w trójkącie);
	// zbiór bez żadnej części wspólnej stoi osobno po prawej.
	linked := func(i int) bool {
		for j := range cs {
			if _, any := overlap(i, j); j != i && any {
				return true
			}
		}
		return false
	}
	var group, alone []int
	for i := range cs {
		if linked(i) {
			group = append(group, i)
		} else {
			alone = append(alone, i)
		}
	}
	switch len(group) {
	case 2:
		cs[group[1]].x = dist(group[0], group[1])
	case 3:
		d01, d02, d12 := dist(0, 1), dist(0, 2), dist(1, 2)
		cs[1].x = d01
		// Trzecie koło nad odcinkiem pierwszych dwóch; przy niespełnionej nierówności trójkąta
		// odległość do drugiego koła jest przycinana.
		d12 = math.Max(math.Abs(d01-d02)+1, math.Min(d12, d01+d02-1))
		x := (d01*d01 + d02*d02 - d12*d12) / (2 * d01)
		cs[2].x, cs[2].y = x, -math.Sqrt(math.Max(0, d02*d02-x*x))
	}
	right, mid := 0.0, 0.0
	for k, i := range group {
		right = math.Max(right, cs[i].x+cs[i].r)
		mid += cs[i].y / float64(len(group))
		if k == 0 {
			right = cs[i].x + cs[i].r
		}
	}
	for k, i := range alone {
		if k == 0 && len(group) == 0 {
			right = -cs[i].r - u(30)
		}
		cs[i].x, cs[i].y = right+u(30)+cs[i].r, mid
		right = cs[i].x + cs[i].r
	}

	// Tytuł nad kołami.
	minY, minX, maxX := math.Inf(1), math.Inf(1), math.Inf(-1)
	for _, c := range cs {
		minY, minX, maxX = math.Min(minY, c.y-c.r), math.Min(minX, c.x-c.r), math.Max(maxX, c.x+c.r)
	}
	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "venn-title", X: (minX + maxX) / 2, Y: minY - u(20), W: o.Measure(d.Title, fs*1.15), H: lh,
			Text: d.Title, Anchor: "middle", Bold: true, FontSize: math.Round(fs * 1.15), Fill: colorText})
	}

	// Koła: przezroczyste wypełnienie, więc części wspólne są ciemniejsze.
	for i, s := range d.Sets {
		fill, stroke, opacity := colorFill, colorStroke, 0.08
		if o.Colors {
			t := scene.Distinct(i)
			fill, stroke, opacity = t.Stroke, t.Stroke, 0.18
		}
		width := 1.5
		if s.Paint.Fill != "" {
			fill, opacity = s.Paint.Fill, 0.35
		}
		if s.Paint.Stroke != "" {
			stroke = s.Paint.Stroke
		}
		if s.Paint.StrokeWidth > 0 {
			width = s.Paint.StrokeWidth
		}
		opacity = pick(s.Opacity, opacity)
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "venn-set", X: cs[i].x, Y: cs[i].y, R: cs[i].r,
			Fill: fill, FillOpacity: opacity, Stroke: stroke, StrokeWidth: width})
	}
	// Część wspólna z wypełnieniem ze stylu: wielokąt obszaru wspólnego wszystkich jej kół.
	for _, un := range d.Unions {
		if un.Paint.Fill == "" {
			continue
		}
		var group []circle
		for _, id := range un.IDs {
			group = append(group, cs[index[id]])
		}
		if pts := intersection(group); len(pts) > 2 {
			sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "venn-union", Points: pts,
				Fill: un.Paint.Fill, FillOpacity: pick(un.Opacity, 0.5), Stroke: "none"})
		}
	}

	// Podpisy: zbiór w swojej części poza innymi kołami, część wspólna w środku obszaru wspólnego.
	place := func(a *Area, x, y, wrapW float64, bold bool) {
		var lines []string
		if a.Label != "" || len(a.IDs) == 1 {
			l := a.Label
			if l == "" {
				l = a.IDs[0]
			}
			lines = textwrap.Lines(textwrap.Split(l), wrapW, func(s string) float64 { return o.Measure(s, fs) })
		}
		var items [][]string
		var owners []*Text
		n := 0
		for _, t := range a.Texts {
			ls := textwrap.Lines(textwrap.Split(t.Label), wrapW, func(s string) float64 { return o.Measure(s, small) })
			items, owners = append(items, ls), append(owners, t)
			n += len(ls)
		}
		ih := math.Round(small * 1.25)
		h := float64(len(lines))*lh + float64(n)*ih
		top := y - h/2
		for k, s := range lines {
			fill := colorText
			if a.Paint.Color != "" {
				fill = a.Paint.Color
			}
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "venn-label", X: x, Y: top + float64(k)*lh + lh/2 + 0.35*fs, W: o.Measure(s, fs), H: lh,
				Text: s, Anchor: "middle", Bold: bold, Fill: fill})
		}
		top += float64(len(lines)) * lh
		for k, ls := range items {
			t := owners[k]
			fill := colorItem
			if t.Paint.Color != "" {
				fill = t.Paint.Color
			} else if a.Paint.Color != "" {
				fill = a.Paint.Color
			}
			for _, s := range ls {
				sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "venn-text", X: x, Y: top + ih/2 + 0.35*small, W: o.Measure(s, small), H: ih,
					Text: s, Anchor: "middle", FontSize: small, Fill: fill})
				top += ih
			}
		}
	}
	for i, s := range d.Sets {
		c := cs[i]
		x, y := c.x, c.y
		if linked(i) {
			// Punkt w części koła poza innymi: od środka połączonych kół przez środek tego koła.
			cx, cy := 0.0, 0.0
			for _, j := range group {
				cx, cy = cx+cs[j].x/float64(len(group)), cy+cs[j].y/float64(len(group))
			}
			dx, dy := c.x-cx, c.y-cy
			if l := math.Hypot(dx, dy); l > 0 {
				x, y = c.x+dx/l*c.r*0.45, c.y+dy/l*c.r*0.45
			}
		}
		place(s, x, y, c.r*0.9, true)
	}
	for _, un := range d.Unions {
		var group []circle
		for _, id := range un.IDs {
			group = append(group, cs[index[id]])
		}
		pts := intersection(group)
		if len(pts) < 3 {
			continue
		}
		x, y := centroid(pts)
		var outside []scene.Point
		if len(un.IDs) == 2 && len(cs) == 3 {
			// Soczewka pary przy trzecim kole: podpis odsunięty od obszaru wspólnego wszystkich trzech.
			var third circle
			for i, c := range cs {
				if index[un.IDs[0]] != i && index[un.IDs[1]] != i {
					third = c
				}
			}
			for _, p := range pts {
				if !third.contains(p.X, p.Y) {
					outside = append(outside, p)
				}
			}
			if len(outside) > 2 {
				x, y = centroid(outside)
			}
		}
		// Szerokość łamania: rozpiętość obszaru w poziomie na wysokości podpisu.
		region := pts
		if outside != nil {
			region = outside
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, p := range region {
			if math.Abs(p.Y-y) < lh*1.5 {
				lo, hi = math.Min(lo, p.X), math.Max(hi, p.X)
			}
		}
		w := u(60)
		if hi > lo {
			w = math.Max(w, 0.85*(hi-lo))
		}
		place(un, x, y, w, false)
	}
	return sc
}

// lensDistance szuka odległości środków kół o promieniach a i b, przy której soczewka ma pole area.
func lensDistance(a, b, area float64) float64 {
	lo, hi := math.Abs(a-b), a+b
	for k := 0; k < 60; k++ {
		mid := (lo + hi) / 2
		if lensArea(a, b, mid) > area {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// lensArea to pole części wspólnej dwóch kół o promieniach a, b i odległości środków d.
func lensArea(a, b, d float64) float64 {
	if d >= a+b {
		return 0
	}
	if d <= math.Abs(a-b) {
		r := math.Min(a, b)
		return math.Pi * r * r
	}
	alpha := math.Acos((d*d + a*a - b*b) / (2 * d * a))
	beta := math.Acos((d*d + b*b - a*a) / (2 * d * b))
	return a*a*(alpha-math.Sin(2*alpha)/2) + b*b*(beta-math.Sin(2*beta)/2)
}

// intersection zwraca obrys części wspólnej kół jako wielokąt: punkty brzegów, które leżą we
// wszystkich kołach, uporządkowane kątem wokół środka (część wspólna kół jest wypukła).
func intersection(group []circle) []scene.Point {
	const steps = 96
	var pts []scene.Point
	for i, c := range group {
		for k := 0; k < steps; k++ {
			t := 2 * math.Pi * float64(k) / steps
			x, y := c.x+c.r*math.Cos(t), c.y+c.r*math.Sin(t)
			in := true
			for j, other := range group {
				if j != i && !other.contains(x, y) {
					in = false
					break
				}
			}
			if in {
				pts = append(pts, scene.Point{X: math.Round(x*100) / 100, Y: math.Round(y*100) / 100})
			}
		}
	}
	if len(pts) < 3 {
		return nil
	}
	mx, my := centroid(pts)
	sort.SliceStable(pts, func(a, b int) bool {
		return math.Atan2(pts[a].Y-my, pts[a].X-mx) < math.Atan2(pts[b].Y-my, pts[b].X-mx)
	})
	return pts
}

func centroid(pts []scene.Point) (float64, float64) {
	x, y := 0.0, 0.0
	for _, p := range pts {
		x, y = x+p.X, y+p.Y
	}
	return x / float64(len(pts)), y / float64(len(pts))
}
