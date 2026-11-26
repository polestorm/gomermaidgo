// Package eventmodeling rozbiera i rysuje model zdarzeń (Event Modeling): ramki czasu ułożone
// od lewej do prawej, każda z jednym elementem (ekran, proces, polecenie, model odczytu,
// zdarzenie) w torze swojego rodzaju, i powiązania między ramkami.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package eventmodeling

import (
	"math"
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
	for _, l := range lanes {
		l.y = y
		y += l.h
	}
	// Kolumny ramek: element ramki zaczyna się za poprzednim, z odstępem na strzałkę.
	gap := u(36)
	x := headW + u(20)
	for _, b := range boxes {
		b.x = x
		b.y = b.lane.y + (b.lane.h-b.h)/2
		x += b.w + gap
	}
	width := x - gap + u(20)

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
	clip := func(bx *box, tx, ty float64) (float64, float64) {
		cx, cy := bx.x+bx.w/2, bx.y+bx.h/2
		dx, dy := tx-cx, ty-cy
		k := math.Min(math.Abs(bx.w/2/dx), math.Abs(bx.h/2/dy))
		if math.IsInf(k, 0) || math.IsNaN(k) {
			k = 0
		}
		return cx + dx*k, cy + dy*k
	}
	for _, e := range edges {
		acx, acy := e.a.x+e.a.w/2, e.a.y+e.a.h/2
		bcx, bcy := e.b.x+e.b.w/2, e.b.y+e.b.h/2
		x1, y1 := clip(e.a, bcx, bcy)
		x2, y2 := clip(e.b, acx, acy)
		dx, dy := x2-x1, y2-y1
		l := math.Hypot(dx, dy)
		if l < u(4) {
			continue
		}
		ux, uy := dx/l, dy/l
		a, hw := u(10), u(5)
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "eventmodeling-edge", X: x1, Y: y1, X2: x2 - ux*a*0.8, Y2: y2 - uy*a*0.8, Stroke: colorEdge, StrokeWidth: 1.5})
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "eventmodeling-edge", Fill: colorEdge, Stroke: colorEdge, StrokeWidth: 1,
			Points: []scene.Point{{X: x2, Y: y2}, {X: x2 - ux*a - uy*hw, Y: y2 - uy*a + ux*hw}, {X: x2 - ux*a + uy*hw, Y: y2 - uy*a - ux*hw}}})
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
