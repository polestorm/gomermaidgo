// Package eventmodeling rozbiera i rysuje model zdarzeń (Event Modeling): ramki czasu ułożone
// od lewej do prawej, każda z jednym elementem (ekran, proces, polecenie, model odczytu,
// zdarzenie) w torze swojego rodzaju, i powiązania między ramkami.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package eventmodeling

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni modelu zdarzeń.
const (
	CodeHeader    = "eventmodeling-header"
	CodeStatement = "eventmodeling-statement"
	CodeEmpty     = "eventmodeling-empty"
	maxFrames     = 200
)

// Rodzaje elementów.
const (
	UI        = "ui"
	Processor = "processor"
	Command   = "command"
	ReadModel = "readmodel"
	Event     = "event"
)

var kinds = map[string]string{"ui": UI, "pcr": Processor, "processor": Processor, "cmd": Command, "command": Command,
	"rmo": ReadModel, "readmodel": ReadModel, "evt": Event, "event": Event}

// group zwraca grupę torów rodzaju: 0 ekrany i procesy, 1 polecenia i modele odczytu, 2 zdarzenia.
func group(kind string) int {
	switch kind {
	case UI, Processor:
		return 0
	case Command, ReadModel:
		return 1
	}
	return 2
}

var groupNames = []string{"UI / Automation", "Command / Read model", "Events"}

// Frame to ramka czasu z jednym elementem.
type Frame struct {
	Number    int
	Kind      string
	Namespace string
	Name      string
	Data      []string // wiersze danych „klucz: wartość”
	DataRef   string   // [[Blok]]
	Reset     bool     // ramka resetu: bez wywnioskowanego powiązania z poprzednią
	From      []int    // jawne powiązania ->> NN
	Line      int
}

// Model to ramki w kolejności źródła i bloki danych.
type Model struct {
	Frames   []*Frame
	Data     map[string][]string
	AccTitle string
	AccDescr string
}

var (
	frameRule = syntax.Lazy(`^(tf|timeframe|rf|resetframe)\s+(\d+)\s+(\w+)\s+([\p{L}_][\p{L}\p{N}_.]*)\s*(.*)$`)
	dataRule  = syntax.Lazy(`^data\s+([\p{L}_][\p{L}\p{N}_]*)\s*(?:` + "`" + `\w+` + "`" + `)?\s*\{\s*(.*)$`)
)

// Parse rozbiera źródło.
func Parse(source string) (*Model, error) {
	m := &Model{Data: map[string][]string{}}
	var acc syntax.Accessibility
	seenHeader := false
	numbers := map[int]bool{}
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	for n := 0; n < len(lines); n++ {
		line := n + 1
		s := strings.TrimSpace(lines[n])
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "eventmodeling" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"eventmodeling\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		if v := dataRule.FindStringSubmatch(s); v != nil {
			body := v[2]
			for !strings.Contains(body, "}") {
				if n+1 >= len(lines) {
					return nil, syntax.Errorf(line, CodeStatement, "data block %q is not closed with \"}\"", v[1])
				}
				n++
				body += "\n" + lines[n]
			}
			if _, ok := m.Data[v[1]]; ok {
				return nil, syntax.Errorf(line, CodeStatement, "data block %q is defined twice", v[1])
			}
			m.Data[v[1]] = entries(body[:strings.LastIndex(body, "}")])
			continue
		}
		v := frameRule.FindStringSubmatch(s)
		if v == nil {
			return nil, syntax.Errorf(line, CodeStatement, "unknown statement %q", s)
		}
		kind, ok := kinds[v[3]]
		if !ok {
			return nil, syntax.Errorf(line, CodeStatement, "unknown element type %q (ui, pcr, cmd, rmo, evt)", v[3])
		}
		num, _ := strconv.Atoi(v[2])
		if numbers[num] {
			return nil, syntax.Errorf(line, CodeStatement, "frame number %s is used twice", v[2])
		}
		numbers[num] = true
		if len(m.Frames) >= maxFrames {
			return nil, syntax.Errorf(line, CodeStatement, "too many frames (limit %d)", maxFrames)
		}
		f := &Frame{Number: num, Kind: kind, Name: v[4], Reset: v[1] == "rf" || v[1] == "resetframe", Line: line}
		if i := strings.LastIndex(f.Name, "."); i > 0 {
			f.Namespace, f.Name = f.Name[:i], f.Name[i+1:]
		}
		rest := strings.TrimSpace(v[5])
		// Dane: [[Blok]] albo { … } z opcjonalnym typem w odwrotnych apostrofach, potem ->> NN.
		if strings.HasPrefix(rest, "`") {
			if end := strings.Index(rest[1:], "`"); end >= 0 {
				rest = strings.TrimSpace(rest[end+2:])
			}
		}
		switch {
		case strings.HasPrefix(rest, "[["):
			end := strings.Index(rest, "]]")
			if end < 0 {
				return nil, syntax.Errorf(line, CodeStatement, "[[ is not closed with ]]")
			}
			f.DataRef, rest = strings.TrimSpace(rest[2:end]), strings.TrimSpace(rest[end+2:])
		case strings.HasPrefix(rest, "{"):
			end := strings.Index(rest, "}")
			if end < 0 {
				return nil, syntax.Errorf(line, CodeStatement, "inline data is not closed with \"}\"")
			}
			f.Data, rest = entries(rest[1:end]), strings.TrimSpace(rest[end+1:])
		}
		for rest != "" {
			if !strings.HasPrefix(rest, "->>") {
				return nil, syntax.Errorf(line, CodeStatement, "unexpected %q", rest)
			}
			rest = strings.TrimSpace(rest[3:])
			end := strings.IndexAny(rest, " \t")
			if end < 0 {
				end = len(rest)
			}
			ref, err := strconv.Atoi(rest[:end])
			if err != nil {
				return nil, syntax.Errorf(line, CodeStatement, "->> needs a frame number")
			}
			f.From, rest = append(f.From, ref), strings.TrimSpace(rest[end:])
		}
		m.Frames = append(m.Frames, f)
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"eventmodeling\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(m.Frames) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the model has no time frames")
	}
	for _, f := range m.Frames {
		for _, ref := range f.From {
			if !numbers[ref] {
				return nil, syntax.Errorf(f.Line, CodeStatement, "->> %d: there is no such frame", ref)
			}
		}
		if f.DataRef != "" {
			d, ok := m.Data[f.DataRef]
			if !ok {
				return nil, syntax.Errorf(f.Line, CodeStatement, "data block %q is not defined", f.DataRef)
			}
			f.Data = d
		}
	}
	m.AccTitle, m.AccDescr = acc.Title, acc.Descr
	return m, nil
}

// entries dzieli dane na wiersze „klucz: wartość” po przecinkach albo liniach.
func entries(s string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' }) {
		if part = strings.TrimSpace(part); part != "" {
			k, v, ok := strings.Cut(part, ":")
			if ok {
				part = strings.TrimSpace(k) + ": " + strings.Trim(strings.TrimSpace(v), `'"`)
			}
			out = append(out, part)
		}
	}
	return out
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // elementy w barwach przyjętych w Event Modeling
	Shadows    bool // cień pod elementami
}

const (
	colorText  = "#1b1f24"
	colorMuted = "#55657a"
	colorLane  = "#c3cad3"
	colorBox   = "#ffffff"
	colorEdge  = "#3b4754"
	colorHead  = "#f3f5f8"
)

// fills to barwy rodzajów w trybie kolorowym: polecenie niebieskie, zdarzenie pomarańczowe,
// model odczytu zielony, jak na tablicach Event Modeling.
var fills = map[string]string{UI: "#f3f5f8", Processor: "#ece4f7", Command: "#d6e8fb", ReadModel: "#dcefd2", Event: "#fde0bd"}

// lane to tor: grupa rodzaju i przestrzeń nazw.
type lane struct {
	group     int
	namespace string
	y, h      float64
	index     int // miejsce toru od góry
}

// Layout rysuje model: tory jeden pod drugim (ekrany i procesy, polecenia i modele odczytu,
// zdarzenia; każda przestrzeń nazw to osobny tor w swojej grupie), ramki czasu jako kolumny od
// lewej, element ramki w jej kolumnie i torze, strzałki powiązań.
func Layout(m *Model, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return math.Round(v * fs / 16) }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.8)
	slh := math.Round(small * 1.25)
	sc := &scene.Scene{Title: m.AccTitle, Desc: m.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	// Tory w grupach, w kolejności pierwszego pojawienia; tor bez przestrzeni nazw pierwszy.
	var lanes []*lane
	find := func(f *Frame) *lane {
		for _, l := range lanes {
			if l.group == group(f.Kind) && l.namespace == f.Namespace {
				return l
			}
		}
		return nil
	}
	for g := 0; g < 3; g++ {
		for _, f := range m.Frames {
			if group(f.Kind) == g && find(f) == nil {
				lanes = append(lanes, &lane{group: g, namespace: f.Namespace})
			}
		}
	}
	// Rozmiary elementów.
	type box struct {
		f          *Frame
		w, h, x, y float64
		lane       *lane
	}
	boxes := make([]*box, len(m.Frames))
	for i, f := range m.Frames {
		w := math.Max(u(130), o.Measure(f.Name, fs)*1.06+2*u(12))
		w = math.Max(w, o.Measure(f.Kind, small)+2*u(12))
		for _, d := range f.Data {
			w = math.Max(w, o.Measure(d, small)+2*u(12))
		}
		h := u(8) + slh + lh + float64(len(f.Data))*slh + u(10)
		boxes[i] = &box{f: f, w: w, h: h, lane: find(f)}
	}
	for _, l := range lanes {
		l.h = u(80)
		for _, b := range boxes {
			if b.lane == l {
				l.h = math.Max(l.h, b.h+2*u(16))
			}
		}
	}
	// Kolumna nagłówków torów z lewej.
	headW := 0.0
	laneName := func(l *lane) []string {
		parts := strings.Split(groupNames[l.group], " / ")
		if l.namespace != "" {
			parts = append([]string{l.namespace}, parts...)
		}
		return parts
	}
	for _, l := range lanes {
		for k, s := range laneName(l) {
			size := small
			if k == 0 && l.namespace != "" {
				size = fs
			}
			headW = math.Max(headW, o.Measure(s, size))
		}
	}
	headW += 2 * u(12)
	y := 0.0
	for i, l := range lanes {
		l.y, l.index = y, i
		y += l.h
	}
	// Kolumny ramek: element zaczyna się za początkiem poprzedniego, ale może na nim częściowo
	// stać (w innym torze), co ścieśnia rysunek; kolejność czyta się z lewych krawędzi. W tym
	// samym torze element stoi za poprzednim elementem toru, z odstępem na strzałkę.
	gap := u(36)
	x0 := headW + u(20)
	right := x0
	lastIn := map[*lane]*box{}
	for i, b := range boxes {
		x := x0
		if i > 0 {
			p := boxes[i-1]
			x = p.x + math.Max(u(64), p.w*0.5)
		}
		if l := lastIn[b.lane]; l != nil {
			x = math.Max(x, l.x+l.w+gap)
		}
		b.x = x
		b.y = b.lane.y + (b.lane.h-b.h)/2
		lastIn[b.lane] = b
		right = math.Max(right, b.x+b.w)
	}
	width := right + u(20)

	// Tory: pas z nagłówkiem i linia podziału.
	for i, l := range lanes {
		if o.Colors && i%2 == 1 {
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "eventmodeling-lane", X: 0, Y: l.y, W: width, H: l.h, Fill: "#f7f8fa"})
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "eventmodeling-lane-head", X: 0, Y: l.y, W: headW, H: l.h, Fill: colorHead})
		names := laneName(l)
		top := l.y + l.h/2 - float64(len(names))*slh/2
		for k, s := range names {
			size, bold := small, false
			if k == 0 && l.namespace != "" {
				size, bold = small, true
			}
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "eventmodeling-lane-label", X: u(12), Y: top + (float64(k)+0.5)*slh + 0.35*size,
				W: o.Measure(s, size), H: slh, Text: s, Anchor: "start", Bold: bold, FontSize: size, Fill: colorMuted})
		}
		if i > 0 {
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerBackground, Role: "eventmodeling-lane-line", X: 0, Y: l.y, X2: width, Y2: l.y, Stroke: colorLane, StrokeWidth: 1})
		}
	}
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "eventmodeling-frame", X: 0, Y: 0, W: width, H: y, Fill: "none", Stroke: colorLane, StrokeWidth: 1})
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "eventmodeling-frame", X: headW, Y: 0, X2: headW, Y2: y, Stroke: colorLane, StrokeWidth: 1})

	// Powiązania: wywnioskowane z poprzednią ramką (poza ramką resetu) i jawne ->>.
	byNumber := map[int]*box{}
	for _, b := range boxes {
		byNumber[b.f.Number] = b
	}
	type edge struct{ a, b *box }
	var edges []edge
	for i, b := range boxes {
		if i > 0 && !b.f.Reset {
			edges = append(edges, edge{boxes[i-1], b})
		}
		for _, ref := range b.f.From {
			if a := byNumber[ref]; a != b {
				edges = append(edges, edge{a, b})
			}
		}
	}
	// Trasy strzałek. W tym samym torze, bez elementu pomiędzy: poziomo od boku do boku. Między
	// torami: z górnego albo dolnego boku elementu (tego zwróconego do toru celu) pionowo do pasa
	// przy granicy torów, w którym nie stoją elementy, wzdłuż niego i pionowo w bok celu. Gdy tory
	// nie sąsiadują, trasa przechodzi przez tory pośrednie w kolumnie wolnej od elementów.
	// Wyjścia i wejścia na jednym boku rozkładają się wzdłuż niego; strzałki w jednym pasie biegną
	// na różnych wysokościach.
	type end struct {
		e     int
		other float64 // x drugiego końca, do kolejności na boku
	}
	type route struct {
		pts      []scene.Point
		straight bool
	}
	sideOf := func(from, to *lane) int { // -1 górny bok, +1 dolny
		if to.index < from.index {
			return -1
		}
		return 1
	}
	between := func(a, c *box) bool { // element w tym samym torze między a i c
		for _, o := range boxes {
			if o != a && o != c && o.lane == a.lane && o.x > math.Min(a.x, c.x) && o.x < math.Max(a.x, c.x) {
				return true
			}
		}
		return false
	}
	routes := make([]route, len(edges))
	type sideKey struct {
		b    *box
		side int
	}
	ends := map[sideKey][]end{}
	for i, e := range edges {
		if e.a.lane == e.b.lane && !between(e.a, e.b) {
			routes[i].straight = true
			continue
		}
		sa, sb := sideOf(e.a.lane, e.b.lane), sideOf(e.b.lane, e.a.lane)
		if e.a.lane == e.b.lane { // w tym samym torze z elementem pomiędzy: pasem pod torem
			sa, sb = 1, 1
			if e.a.lane.index == len(lanes)-1 {
				sa, sb = -1, -1
			}
		}
		ends[sideKey{e.a, sa}] = append(ends[sideKey{e.a, sa}], end{i, e.b.x + e.b.w/2})
		ends[sideKey{e.b, sb}] = append(ends[sideKey{e.b, sb}], end{i, e.a.x + e.a.w/2})
	}
	portX := map[[2]int]float64{} // (krawędź, 0 = początek / 1 = koniec) → x na boku
	for k, list := range ends {
		sort.SliceStable(list, func(i, j int) bool { return list[i].other < list[j].other })
		for n, en := range list {
			x := k.b.x + k.b.w*float64(n+1)/float64(len(list)+1)
			which := 0
			if edges[en.e].b == k.b {
				which = 1
			}
			portX[[2]int{en.e, which}] = x
		}
	}
	// Pas przy granicy toru l po stronie side: y środka pasa.
	channel := func(l *lane, side int) float64 {
		if side < 0 {
			return l.y
		}
		return l.y + l.h
	}
	freeColumn := func(lo, hi float64, from, to *lane) float64 {
		minL, maxL := math.Min(float64(from.index), float64(to.index)), math.Max(float64(from.index), float64(to.index))
		clear := func(x float64) bool {
			for _, o := range boxes {
				if float64(o.lane.index) > minL && float64(o.lane.index) < maxL && x > o.x-u(8) && x < o.x+o.w+u(8) {
					return false
				}
			}
			return true
		}
		mid := (lo + hi) / 2
		for d := 0.0; d < width; d += u(4) {
			for _, x := range []float64{mid + d, mid - d} {
				if x > headW+u(8) && x < width-u(8) && clear(x) {
					return x
				}
			}
		}
		return mid
	}
	// Odcinki w pasach: pas (granica torów), zakres x i kierunek pionowych odcinków na końcach
	// (-1 w górę, +1 w dół od pasa).
	type run struct {
		edge, part int
		c          float64
		x1, x2     float64
		d1, d2     int
		y          float64 // wysokość po rozłożeniu
	}
	var runs []*run
	type plan struct {
		xa, xb, xm, ya, yb float64
		two                bool
		r1, r2             *run
	}
	plans := make([]plan, len(edges))
	for i, e := range edges {
		if routes[i].straight {
			continue
		}
		sa, sb := sideOf(e.a.lane, e.b.lane), sideOf(e.b.lane, e.a.lane)
		if e.a.lane == e.b.lane {
			sa, sb = 1, 1
			if e.a.lane.index == len(lanes)-1 {
				sa, sb = -1, -1
			}
		}
		pl := plan{xa: portX[[2]int{i, 0}], xb: portX[[2]int{i, 1}]}
		pl.ya = e.a.y + e.a.h/2 + float64(sa)*e.a.h/2
		pl.yb = e.b.y + e.b.h/2 + float64(sb)*e.b.h/2
		ca, cb := channel(e.a.lane, sa), channel(e.b.lane, sb)
		if ca == cb {
			pl.r1 = &run{edge: i, c: ca, x1: pl.xa, x2: pl.xb, d1: -sa, d2: -sb}
			runs = append(runs, pl.r1)
		} else {
			pl.two = true
			pl.xm = freeColumn(pl.xa, pl.xb, e.a.lane, e.b.lane)
			pl.r1 = &run{edge: i, c: ca, x1: pl.xa, x2: pl.xm, d1: -sa, d2: sa}
			pl.r2 = &run{edge: i, part: 1, c: cb, x1: pl.xm, x2: pl.xb, d1: sb, d2: -sb}
			runs = append(runs, pl.r1, pl.r2)
		}
		plans[i] = pl
	}
	// Rozłożenie odcinków w pasie: pionowy odcinek jednej strzałki nie może przeciąć poziomego
	// innej. Pionowy w górę w zakresie odcinka e wymaga, żeby e leżał niżej; w dół — wyżej.
	// Kolejność z sortowania topologicznego tych warunków (przy cyklu zostaje kolejność tras).
	byChannel := map[float64][]*run{}
	var channels []float64
	for _, r := range runs {
		if _, ok := byChannel[r.c]; !ok {
			channels = append(channels, r.c)
		}
		byChannel[r.c] = append(byChannel[r.c], r)
	}
	for _, c := range channels {
		list := byChannel[c]
		n := len(list)
		above := make([][]bool, n) // above[f][e]: f musi leżeć wyżej niż e
		for f := range above {
			above[f] = make([]bool, n)
		}
		inside := func(x float64, r *run) bool {
			return x > math.Min(r.x1, r.x2)+0.5 && x < math.Max(r.x1, r.x2)-0.5
		}
		for f, rf := range list {
			for e, re := range list {
				if f == e {
					continue
				}
				for _, v := range [][2]float64{{rf.x1, float64(rf.d1)}, {rf.x2, float64(rf.d2)}} {
					if inside(v[0], re) {
						if v[1] < 0 {
							above[f][e] = true
						} else {
							above[e][f] = true
						}
					}
				}
			}
		}
		var order []int
		done := make([]bool, n)
		for len(order) < n {
			picked := -1
			for f := 0; f < n && picked < 0; f++ {
				if done[f] {
					continue
				}
				free := true
				for g := 0; g < n; g++ {
					if !done[g] && g != f && above[g][f] {
						free = false
						break
					}
				}
				if free {
					picked = f
				}
			}
			if picked < 0 { // cykl: pierwszy nieułożony
				for f := 0; f < n; f++ {
					if !done[f] {
						picked = f
						break
					}
				}
			}
			done[picked] = true
			order = append(order, picked)
		}
		step := u(6)
		if n > 1 {
			step = math.Min(step, 2*u(13)/float64(n-1))
		}
		for k, f := range order {
			list[f].y = c + (float64(k)-float64(n-1)/2)*step
		}
	}
	for i := range edges {
		if routes[i].straight {
			continue
		}
		pl := plans[i]
		pts := []scene.Point{{X: pl.xa, Y: pl.ya}, {X: pl.xa, Y: pl.r1.y}}
		if pl.two {
			pts = append(pts, scene.Point{X: pl.xm, Y: pl.r1.y}, scene.Point{X: pl.xm, Y: pl.r2.y}, scene.Point{X: pl.xb, Y: pl.r2.y})
		} else {
			pts = append(pts, scene.Point{X: pl.xb, Y: pl.r1.y})
		}
		pts = append(pts, scene.Point{X: pl.xb, Y: pl.yb})
		// Bez zbędnych punktów: dwa kolejne w tym samym miejscu albo trzy w jednej linii.
		var clean []scene.Point
		for _, p := range pts {
			if n := len(clean); n > 0 && math.Abs(clean[n-1].X-p.X) < 0.5 && math.Abs(clean[n-1].Y-p.Y) < 0.5 {
				continue
			}
			if n := len(clean); n > 1 {
				a, b := clean[n-2], clean[n-1]
				if (math.Abs(a.X-b.X) < 0.5 && math.Abs(b.X-p.X) < 0.5) || (math.Abs(a.Y-b.Y) < 0.5 && math.Abs(b.Y-p.Y) < 0.5) {
					clean[n-1] = p
					continue
				}
			}
			clean = append(clean, p)
		}
		routes[i].pts = clean
	}
	for i, e := range edges {
		pts := routes[i].pts
		if routes[i].straight {
			cy := e.b.y + e.b.h/2
			x1, x2 := e.a.x+e.a.w, e.b.x
			if e.b.x < e.a.x {
				x1, x2 = e.a.x, e.b.x+e.b.w
			}
			pts = []scene.Point{{X: x1, Y: cy}, {X: x2, Y: cy}}
		}
		if len(pts) < 2 {
			continue
		}
		tip, prev := pts[len(pts)-1], pts[len(pts)-2]
		dx, dy := tip.X-prev.X, tip.Y-prev.Y
		l := math.Hypot(dx, dy)
		if l < u(4) {
			continue
		}
		ux, uy := dx/l, dy/l
		a, hw := u(10), u(5)
		line := append([]scene.Point(nil), pts...)
		line[len(line)-1] = scene.Point{X: tip.X - ux*a*0.8, Y: tip.Y - uy*a*0.8}
		sc.Add(scene.Item{Kind: scene.Polyline, Layer: scene.LayerShape, Role: "eventmodeling-edge", Points: line, Fill: "none", Stroke: colorEdge, StrokeWidth: 1.5})
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "eventmodeling-edge", Fill: colorEdge, Stroke: colorEdge, StrokeWidth: 1,
			Points: []scene.Point{tip, {X: tip.X - ux*a - uy*hw, Y: tip.Y - uy*a + ux*hw}, {X: tip.X - ux*a + uy*hw, Y: tip.Y - uy*a - ux*hw}}})
	}

	// Elementy: rodzaj małymi literami u góry, nazwa, dane.
	from := len(sc.Items)
	for _, b := range boxes {
		fill := colorBox
		if o.Colors {
			fill = fills[b.f.Kind]
		}
		rx, dash := u(4), ""
		switch b.f.Kind {
		case UI:
			rx = 0
		case Processor:
			dash = "5 3"
		case Event:
			rx = u(10)
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "eventmodeling-element", X: b.x, Y: b.y, W: b.w, H: b.h, RX: rx,
			Fill: fill, Stroke: colorEdge, StrokeWidth: 1.5, Dash: dash})
		cx := b.x + b.w/2
		top := b.y + u(8)
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "eventmodeling-kind", X: cx, Y: top + slh/2 + 0.35*small, W: o.Measure(b.f.Kind, small), H: slh,
			Text: b.f.Kind, Anchor: "middle", Italic: true, FontSize: small, Fill: colorMuted})
		top += slh
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "eventmodeling-name", X: cx, Y: top + lh/2 + 0.35*fs, W: o.Measure(b.f.Name, fs), H: lh,
			Text: b.f.Name, Anchor: "middle", Bold: true, Fill: colorText})
		top += lh
		for _, d := range b.f.Data {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "eventmodeling-data", X: b.x + u(12), Y: top + slh/2 + 0.35*small, W: o.Measure(d, small), H: slh,
				Text: d, Anchor: "start", FontSize: small, Fill: colorText})
			top += slh
		}
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "eventmodeling-element")
	}
	return sc
}
