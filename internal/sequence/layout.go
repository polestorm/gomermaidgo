package sequence

import (
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"math"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Measure zwraca szerokość tekstu w pikselach dla podanego rozmiaru pisma.
type Measure func(text string, size float64) float64

// Options to ustawienia układu.
type Options struct {
	Measure    Measure
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // uczestnicy w kolejnych kolorach palety
	Shadows    bool // cień pod uczestnikami i notatkami
	// Wrap łamie teksty komunikatów i notatek szersze niż WrapWidth (konfiguracja Mermaid
	// „wrap: true”); WrapWidth 0 = 200 px przy piśmie 16 px.
	Wrap      bool
	WrapWidth float64
}

// Kolory rysunku. Wygląd jest stały i własny; nie naśladuje motywów Mermaid.
const (
	colorText       = "#1b1f24"
	colorLine       = "#3b4754"
	colorLifeline   = "#9aa5b1"
	colorBoxFill    = "#eef2f7"
	colorBoxStroke  = "#55657a"
	colorNoteFill   = "#fff6c2"
	colorNoteStroke = "#b59a1b"
	colorFrame      = "#6b7785"
	colorTabFill    = "#e3e8ef"
	colorActivation = "#dbe3ec"
	colorBadge      = "#3b4754"
	colorBadgeText  = "#ffffff"
)

type frame struct {
	kind     string
	title    string
	color    string
	y0       float64
	first    int // indeks pierwszego elementu sceny dodanego wewnątrz bloku
	minX     float64
	maxX     float64
	sections []section
}

type section struct {
	y    float64
	text string
}

type layout struct {
	d  *Diagram
	o  Options
	sc *scene.Scene

	fs, lh float64
	x      []float64 // środki uczestników
	boxW   []float64
	header float64   // wysokość nagłówka uczestników
	start  []float64 // początek linii życia każdego uczestnika
	end    []float64 // koniec linii życia uczestnika, który zakończył życie; NaN = do stopki
	boxPad []float64 // poszerzenie ramki grupy „box” z każdej strony ponad uczestników

	cursor float64
	lastY  float64
	active [][]float64 // początki otwartych aktywacji
	frames []*frame
	fills  []string // kolory tła pod bieżącym miejscem: tło rysunku i otwarte wyróżnienia „rect”
	plain  bool     // tło rysunku nie jest zapisane kolorem (przezroczyste albo nazwane): podkłady na nim to Knockout
	number int
}

// Stałe wymiary w pikselach dla rozmiaru pisma 16; skalują się z rozmiarem pisma.
func (l *layout) u(v float64) float64 { return math.Round(v * l.fs / 16) }

func (l *layout) width(text string) float64 { return l.o.Measure(text, l.fs) }

func (l *layout) maxWidth(lines []string) float64 {
	w := 0.0
	for _, s := range lines {
		w = math.Max(w, l.width(s))
	}
	return w
}

// baseline zwraca położenie linii bazowej dla wiersza o górnej krawędzi top.
func (l *layout) baseline(top float64) float64 { return top + l.lh/2 + 0.35*l.fs }

func (l *layout) text(lines []string, x, top float64, anchor, role, fill string, layer int) {
	if fill == "" {
		fill = colorText
	}
	for k, s := range lines {
		if s == "" {
			continue
		}
		l.sc.Add(scene.Item{Kind: scene.Text, Layer: layer, Role: role, X: x, Y: l.baseline(top + float64(k)*l.lh),
			W: l.width(s), H: l.lh, Text: s, Anchor: anchor, Fill: fill})
	}
}

// labelBackground kładzie pod etykietą prostokąt w kolorze tła, który przerywa linie życia.
func (l *layout) labelBackground(x, top, w, h float64) {
	if w <= 0 {
		return
	}
	fill := l.fillAt(x-l.u(3), x+w+l.u(3))
	// Na samym tle rysunku, które nie jest zapisane kolorem: podkład w kolorze tła (scene.Item.Knockout).
	knockout := l.plain && len(l.fills) == 1 && fill == l.fills[0]
	if knockout {
		fill = ""
	}
	l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: "label-background",
		X: x - l.u(3), Y: top + l.u(1), W: w + 2*l.u(3), H: h - l.u(2), Fill: fill, Knockout: knockout})
}

// fillAt zwraca kolor tła pod odcinkiem od x0 do x1: wyróżnienie „rect” albo tło rysunku,
// a pod nim kolor grupy „box”, jeśli odcinek mieści się w niej cały.
func (l *layout) fillAt(x0, x1 float64) string {
	fill := l.fills[len(l.fills)-1]
	if len(l.fills) > 1 {
		return fill
	}
	for k, b := range l.d.Boxes {
		if b.Color == "" {
			continue
		}
		pad := l.boxPad[k]
		if x0 >= l.x[b.First]-l.boxW[b.First]/2-pad && x1 <= l.x[b.Last]+l.boxW[b.Last]/2+pad {
			return blend(b.Color, fill)
		}
	}
	return fill
}

// extent zgłasza zajęty zakres poziomy do bieżącej ramki.
func (l *layout) extent(x0, x1 float64) {
	if len(l.frames) == 0 {
		return
	}
	f := l.frames[len(l.frames)-1]
	f.minX, f.maxX = math.Min(f.minX, x0), math.Max(f.maxX, x1)
}

// wrapped zwraca kopię diagramu z tekstami komunikatów i notatek złamanymi na granicach wyrazów.
func wrapped(d *Diagram, o Options) *Diagram {
	limit := o.WrapWidth
	if limit <= 0 {
		limit = 200
	}
	limit *= o.FontSize / 16
	out := *d
	out.Events = make([]Event, len(d.Events))
	copy(out.Events, d.Events)
	for i := range out.Events {
		if e := &out.Events[i]; e.Kind == EventMessage || e.Kind == EventNote {
			e.Text = textwrap.Lines(e.Text, limit, func(s string) float64 { return o.Measure(s, o.FontSize) })
		}
	}
	return &out
}

// Layout rozmieszcza diagram i zwraca scenę.
func Layout(d *Diagram, o Options) *scene.Scene {
	if o.FontSize <= 0 {
		o.FontSize = 16
	}
	if o.Wrap {
		d = wrapped(d, o)
	}
	l := &layout{d: d, o: o, fs: o.FontSize}
	l.lh = math.Round(l.fs * 1.25)
	title := d.AccTitle
	if title == "" {
		title = d.Title
	}
	l.number = d.NumberStart - d.NumberStep
	l.sc = &scene.Scene{Title: title, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: l.fs, Margin: l.u(12), Background: o.Background}
	l.active = make([][]float64, len(d.Participants))
	base := o.Background
	if _, _, _, _, ok := parseColor(base); !ok {
		base = "#ffffff" // przezroczyste albo nazwane tło: odcienie wyróżnień liczone od bieli
		l.plain = true
	}
	l.fills = []string{base}

	l.columns()

	top := 0.0
	if d.Title != "" {
		mid := (l.x[0] + l.x[len(l.x)-1]) / 2
		l.text([]string{d.Title}, mid, top, "middle", "title", "", scene.LayerLabel)
		top += l.lh + l.u(12)
	}
	boxTop := top
	if len(d.Boxes) > 0 {
		// Nad uczestnikami zostaje miejsce na tytuły grup.
		top += l.u(10)
		for _, b := range d.Boxes {
			if b.Title != "" {
				top = boxTop + l.lh + l.u(12)
			}
		}
	}
	l.participants(top, func(p *Participant) bool { return !p.Created })
	headerBottom := top + l.header
	l.start, l.end = make([]float64, len(d.Participants)), make([]float64, len(d.Participants))
	for i := range l.start {
		l.start[i], l.end[i] = headerBottom, math.NaN()
	}
	l.cursor = headerBottom + l.u(24)
	l.lastY = l.cursor

	for _, e := range d.Events {
		switch e.Kind {
		case EventMessage:
			if e.From == e.To {
				l.selfMessage(e)
			} else {
				l.message(e)
			}
		case EventNote:
			l.note(e)
		case EventActivate:
			l.active[e.Target] = append(l.active[e.Target], l.lastY)
		case EventDeactivate:
			l.deactivate(e.Target, l.lastY)
		case EventBlockStart:
			l.blockStart(e)
		case EventBlockSection:
			l.blockSection(e)
		case EventBlockEnd:
			l.blockEnd()
		}
	}
	for i := range l.active {
		for len(l.active[i]) > 0 {
			l.deactivate(i, l.cursor)
		}
	}

	footerTop := l.cursor + l.u(6)
	for _, p := range d.Participants {
		if p.figure() {
			// Miejsce na odstęp między końcem linii życia a głową figurki w stopce.
			footerTop += l.u(10)
			break
		}
	}
	if d.NoFooter {
		footerTop = l.cursor + l.u(6) - l.header - l.u(10)
	} else {
		l.participants(footerTop, func(p *Participant) bool { return !p.Destroyed })
	}
	for i, p := range d.Participants {
		end := footerTop
		if d.NoFooter {
			end = l.cursor + l.u(6)
		} else if p.figure() {
			head := footerTop + (l.header-(l.u(48)+float64(len(p.Label))*l.lh))/2
			end = math.Min(end, head-l.u(8))
		}
		if !math.IsNaN(l.end[i]) {
			end = l.end[i]
		}
		l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "lifeline",
			X: l.x[i], Y: l.start[i], X2: l.x[i], Y2: end, Stroke: colorLifeline, StrokeWidth: 1.5})
	}
	for k, b := range d.Boxes {
		pad := l.boxPad[k]
		x0, x1 := l.x[b.First]-l.boxW[b.First]/2-pad, l.x[b.Last]+l.boxW[b.Last]/2+pad
		it := scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "participant-group",
			X: x0, Y: boxTop, W: x1 - x0, H: footerTop + l.header + l.u(10) - boxTop, RX: l.u(4),
			Stroke: colorFrame, StrokeWidth: 1}
		if b.Color != "" {
			it.Fill, it.FillOpacity = splitColor(b.Color)
		}
		l.sc.Add(it)
		if b.Title != "" {
			l.text([]string{b.Title}, (x0+x1)/2, boxTop+l.u(6), "middle", "participant-group-title", "", scene.LayerLabel)
		}
	}
	return l.sc
}

// columns wyznacza środki uczestników tak, żeby etykiety komunikatów i notatki mieściły
// się między liniami życia.
func (l *layout) columns() {
	n := len(l.d.Participants)
	l.boxW = make([]float64, n)
	l.header = 0
	for i, p := range l.d.Participants {
		w := l.maxWidth(p.Label)
		lines := float64(len(p.Label))
		if p.figure() {
			l.boxW[i] = math.Max(l.u(28), w)
			if !p.Actor {
				l.boxW[i] = math.Max(l.u(56), w) // symbol granicy, sterowania albo encji
			}
			l.header = math.Max(l.header, l.u(48)+lines*l.lh)
		} else {
			l.boxW[i] = math.Max(l.u(100), w+2*l.u(20))
			extra := 0.0
			switch p.Kind {
			case "database":
				extra = l.u(14) // pokrywy walca
			case "collections":
				extra = l.u(6) // przesunięcie tylnego pola
			case "queue":
				l.boxW[i] += l.u(14)
			}
			l.header = math.Max(l.header, lines*l.lh+2*l.u(14)+extra)
		}
	}
	// SEQ-019: okienka uczestników mają wspólną szerokość najszerszego z nich (do u(160)), żeby
	// rząd nagłówków był równy; dłuższa nazwa poszerza tylko własne okienko.
	common := 0.0
	for i, p := range l.d.Participants {
		if !p.figure() {
			common = math.Max(common, l.boxW[i])
		}
	}
	common = math.Min(common, l.u(160))
	for i, p := range l.d.Participants {
		if !p.figure() {
			l.boxW[i] = math.Max(l.boxW[i], common)
		}
	}
	group := make([]int, n) // numer grupy „box” uczestnika + 1; 0 = poza grupami
	for k, b := range l.d.Boxes {
		for i := b.First; i <= b.Last; i++ {
			group[i] = k + 1
		}
	}
	gap := make([]float64, max(n-1, 0))
	for i := range gap {
		gap[i] = l.boxW[i]/2 + l.boxW[i+1]/2 + l.u(56)
		if group[i] != group[i+1] {
			gap[i] += l.u(12) // ramka grupy nie może dotykać sąsiada spoza grupy
		}
	}
	atLeast := func(i int, v float64) {
		if i >= 0 && i < len(gap) {
			gap[i] = math.Max(gap[i], v)
		}
	}
	// Wymaganie rozłożone na kilka odstępów: brakującą szerokość dzielimy po równo.
	span := func(lo, hi int, need float64) {
		sum := 0.0
		for i := lo; i < hi; i++ {
			sum += gap[i]
		}
		if need > sum && hi > lo {
			add := (need - sum) / float64(hi-lo)
			for i := lo; i < hi; i++ {
				gap[i] += add
			}
		}
	}
	type wide struct {
		lo, hi int
		need   float64
	}
	var deferred []wide
	badge := 0.0
	if l.d.Autonumber {
		badge = l.u(24)
	}
	for _, e := range l.d.Events {
		switch e.Kind {
		case EventMessage:
			w := l.maxWidth(e.Text)
			if e.From == e.To {
				if e.From == n-1 {
					// Ostatni uczestnik: etykieta kończy się nad pętlą i rośnie w lewo, więc nie
					// poszerza rysunku; potrzebuje miejsca przed poprzednią linią życia.
					atLeast(e.From-1, w-l.u(8)-l.u(36)+l.u(16))
					continue
				}
				atLeast(e.From, math.Max(l.selfReach(), l.u(10)+badge+w)+l.u(16)+l.half(e.From+1))
				continue
			}
			lo, hi := min(e.From, e.To), max(e.From, e.To)
			need := w + 2*l.u(22) + l.u(12) + badge
			if e.Created != 0 {
				need += l.half(e.Created - 1) // strzałka kończy się na krawędzi tworzonego uczestnika
			}
			if hi-lo == 1 {
				atLeast(lo, need)
			} else {
				deferred = append(deferred, wide{lo, hi, need})
			}
		case EventNote:
			w := l.maxWidth(e.Text) + 2*l.u(10)
			switch e.Placement {
			case NoteRightOf:
				atLeast(e.Over[0], l.u(16)+w+l.u(14)+l.half(e.Over[0]+1)/4)
			case NoteLeftOf:
				atLeast(e.Over[0]-1, l.u(16)+w+l.u(14)+l.half(e.Over[0]-1)/4)
			case NoteOver:
				if e.Over[0] == e.Over[1] {
					atLeast(e.Over[0]-1, w/2+l.u(12))
					atLeast(e.Over[0], w/2+l.u(12))
				} else {
					deferred = append(deferred, wide{e.Over[0], e.Over[1], w - 2*l.u(20)})
				}
			}
		}
	}
	for _, d := range deferred {
		span(d.lo, d.hi, d.need)
	}
	// Ramka grupy musi pomieścić swój tytuł: brak dzielimy na obie strony.
	l.boxPad = make([]float64, len(l.d.Boxes))
	for k, b := range l.d.Boxes {
		l.boxPad[k] = l.u(10)
		span := l.boxW[b.First]/2 + l.boxW[b.Last]/2 + 2*l.boxPad[k]
		for i := b.First; i < b.Last; i++ {
			span += gap[i]
		}
		if need := l.width(b.Title) + 2*l.u(10); need > span {
			extra := (need - span) / 2
			l.boxPad[k] += extra
			if b.First > 0 {
				gap[b.First-1] += extra
			}
			if b.Last < n-1 {
				gap[b.Last] += extra
			}
		}
	}
	// SEQ-019: równe odstępy między liniami życia, gdy wyrównanie do największego odstępu
	// poszerza rysunek najwyżej o ćwierć; inaczej każdy odstęp zostaje taki, jak wymaga treść.
	if len(gap) > 1 {
		sum, most := 0.0, 0.0
		for _, g := range gap {
			sum += g
			most = math.Max(most, g)
		}
		if most*float64(len(gap)) <= 1.25*sum {
			for i := range gap {
				gap[i] = most
			}
		}
	}
	l.x = make([]float64, n)
	for i := 1; i < n; i++ {
		l.x[i] = math.Round(l.x[i-1] + gap[i-1])
	}
}

func (l *layout) half(i int) float64 {
	if i < 0 || i >= len(l.boxW) {
		return 0
	}
	return l.boxW[i] / 2
}

// selfReach to odległość od linii życia do miejsca, gdzie zaczyna się etykieta komunikatu do siebie.
func (l *layout) selfReach() float64 { return l.u(8) + l.u(36) + l.u(8) }

func (l *layout) participants(top float64, keep func(*Participant) bool) {
	from := len(l.sc.Items)
	defer func() {
		if l.o.Shadows {
			l.sc.AddShadows(from, l.u(3), "participant")
		}
	}()
	for i, p := range l.d.Participants {
		if !keep(p) {
			continue
		}
		cx := l.x[i]
		lines := float64(len(p.Label))
		colorBoxFill, colorBoxStroke := colorBoxFill, colorBoxStroke
		if l.o.Colors {
			tone := scene.Distinct(i)
			colorBoxFill, colorBoxStroke = tone.Fill, tone.Stroke
		}
		x0, x1, y1 := cx-l.boxW[i]/2, cx+l.boxW[i]/2, top+l.header
		shape := func(kind scene.Kind, x, y, w, h float64, fill, stroke string) {
			l.sc.Add(scene.Item{Kind: kind, Layer: scene.LayerShape, Role: "participant", X: x, Y: y, W: w, H: h,
				Fill: fill, Stroke: stroke, StrokeWidth: 1.5})
		}
		edge := func(ax, ay, bx, by float64) {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "participant-detail", X: ax, Y: ay, X2: bx, Y2: by,
				Stroke: colorBoxStroke, StrokeWidth: 1.5})
		}
		textY := top + (l.header-lines*l.lh)/2
		switch {
		case p.Kind == "database":
			// Walec: dolna pokrywa, wypełniony korpus, boki i górna pokrywa; podpis pod górną pokrywą.
			c := l.u(14)
			l.sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "participant", X: cx, Y: y1 - c/2, W: l.boxW[i], H: c,
				Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
			l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "participant-detail", X: x0, Y: top + c/2, W: l.boxW[i], H: l.header - c, Fill: colorBoxFill})
			edge(x0, top+c/2, x0, y1-c/2)
			edge(x1, top+c/2, x1, y1-c/2)
			l.sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "participant-detail", X: cx, Y: top + c/2, W: l.boxW[i], H: c,
				Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
			textY += c / 4
		case p.Kind == "queue":
			// Walec położony: lewa pokrywa, korpus, górna i dolna krawędź, prawa pokrywa.
			c := l.u(14)
			l.sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "participant", X: x0 + c/2, Y: top + l.header/2, W: c, H: l.header,
				Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
			l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "participant-detail", X: x0 + c/2, Y: top, W: l.boxW[i] - c, H: l.header, Fill: colorBoxFill})
			edge(x0+c/2, top, x1-c/2, top)
			edge(x0+c/2, y1, x1-c/2, y1)
			l.sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "participant-detail", X: x1 - c/2, Y: top + l.header/2, W: c, H: l.header,
				Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
			cx -= c / 4
		case p.Kind == "collections":
			// Zbiór: tylne pole przesunięte w prawo i w górę, przednie na nim.
			o := l.u(6)
			l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "participant-detail", X: x0 + o, Y: top, W: l.boxW[i] - o, H: l.header - o, RX: l.u(3),
				Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
			l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "participant", X: x0, Y: top + o, W: l.boxW[i] - o, H: l.header - o, RX: l.u(3),
				Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
			cx -= o / 2
			textY += o / 2
		case !p.figure():
			shape(scene.Rect, x0, top, l.boxW[i], l.header, colorBoxFill, colorBoxStroke)
			l.sc.Items[len(l.sc.Items)-1].RX = l.u(3)
		}
		if !p.figure() {
			l.text(p.Label, cx, textY, "middle", "participant-label", "", scene.LayerLabel)
			continue
		}
		t := top + (l.header-(l.u(48)+lines*l.lh))/2
		stroke := func(x1, y1, x2, y2 float64) {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "actor",
				X: cx + l.u(x1), Y: t + l.u(y1), X2: cx + l.u(x2), Y2: t + l.u(y2), Stroke: colorBoxStroke, StrokeWidth: 1.5})
		}
		ring := func(x, y float64) {
			l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "actor",
				X: cx + l.u(x), Y: t + l.u(y), R: l.u(14), Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
		}
		switch p.Kind {
		case "boundary":
			// Granica (UML): koło z kreską poziomą do pionowej belki po lewej.
			stroke(-26, 8, -26, 36)
			stroke(-26, 22, -14, 22)
			ring(0, 22)
		case "control":
			// Sterowanie: koło z grotem na górze.
			ring(0, 22)
			stroke(5, 2, -2, 8)
			stroke(-2, 8, 5, 14)
		case "entity":
			// Encja: koło z kreską pod spodem.
			ring(0, 20)
			stroke(-14, 38, 14, 38)
		default:
			l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "actor",
				X: cx, Y: t + l.u(8), R: l.u(8), Fill: colorBoxFill, Stroke: colorBoxStroke, StrokeWidth: 1.5})
			stroke(0, 16, 0, 32)
			stroke(-12, 22, 12, 22)
			stroke(0, 32, -10, 44)
			stroke(0, 32, 10, 44)
		}
		l.text(p.Label, cx, t+l.u(48), "middle", "participant-label", "", scene.LayerLabel)
	}
}

// edge zwraca punkt styku komunikatu z linią życia uczestnika po stronie side (-1 lewa, +1 prawa)
// przy level otwartych aktywacjach.
func (l *layout) edge(i int, side float64, level int) float64 {
	if level <= 0 {
		return l.x[i]
	}
	return l.x[i] + side*l.u(5) + float64(level-1)*l.u(4)
}

func (l *layout) deactivate(i int, y float64) {
	stack := l.active[i]
	level := len(stack)
	start := stack[level-1]
	l.active[i] = stack[:level-1]
	h := math.Max(y-start, l.u(8))
	l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerActivation + 0, Role: "activation",
		X: l.x[i] - l.u(5) + float64(level-1)*l.u(4), Y: start, W: l.u(10), H: h,
		Fill: colorActivation, Stroke: colorBoxStroke, StrokeWidth: 1})
}

func dash(s LineStyle) string {
	if s == Dotted {
		return "4 3"
	}
	return ""
}

// head rysuje grot w punkcie (x, y) skierowany w stronę dir (+1 w prawo, -1 w lewo)
// i zwraca, o ile trzeba skrócić linię, żeby nie wystawała spod grotu.
func (l *layout) head(style HeadStyle, x, y, dir float64) float64 {
	a, hw := l.u(9), l.u(4)
	switch style {
	case HeadArrow:
		l.sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "arrowhead",
			Points: []scene.Point{{X: x, Y: y}, {X: x - dir*a, Y: y - hw}, {X: x - dir*a, Y: y + hw}}, Fill: colorLine})
		return a
	case HeadOpen:
		l.sc.Add(scene.Item{Kind: scene.Polyline, Layer: scene.LayerShape, Role: "arrowhead",
			Points: []scene.Point{{X: x - dir*a, Y: y - hw}, {X: x, Y: y}, {X: x - dir*a, Y: y + hw}}, Stroke: colorLine, StrokeWidth: 1.5})
	case HeadCross:
		c, r := x-dir*l.u(6), l.u(4)
		for _, s := range []float64{-1, 1} {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "arrowhead",
				X: c - r, Y: y - s*r, X2: c + r, Y2: y + s*r, Stroke: colorLine, StrokeWidth: 1.5})
		}
	}
	return 0
}

func (l *layout) badge(x, y float64) {
	if !l.d.Autonumber {
		return
	}
	l.number += l.d.NumberStep
	s := strconv.Itoa(l.number)
	size := math.Round(l.fs * 0.7)
	l.sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "sequence-number", X: x, Y: y, R: l.u(9), Fill: colorBadge})
	l.sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "sequence-number", X: x, Y: y + 0.35*size,
		W: l.o.Measure(s, size), H: size, Text: s, Anchor: "middle", FontSize: size, Fill: colorBadgeText})
}

func (l *layout) message(e Event) {
	dir := 1.0
	if e.To < e.From {
		dir = -1
	}
	fromLevel := len(l.active[e.From])
	toLevel := len(l.active[e.To])
	if e.ActivateTo {
		toLevel++
	}
	x1 := l.edge(e.From, dir, fromLevel)
	x2 := l.edge(e.To, -dir, toLevel)
	// Uczestnik tworzony tym komunikatem stoi na wysokości strzałki; strzałka kończy się na jego krawędzi.
	switch e.Created - 1 {
	case e.To:
		x2 = l.x[e.To] - dir*l.boxW[e.To]/2
	case e.From:
		x1 = l.x[e.From] + dir*l.boxW[e.From]/2
	}

	lines := float64(len(e.Text))
	labelTop := l.cursor
	l.cursor += lines*l.lh + l.u(5)
	y := l.cursor
	if len(e.Text) > 0 {
		mid, w := (x1+x2)/2, l.maxWidth(e.Text)
		if abs(e.To-e.From) > 1 {
			// Etykieta przechodzi przez obce linie życia: podkład zapewnia czytelność.
			l.labelBackground(mid-w/2, labelTop, w, lines*l.lh)
		}
		l.text(e.Text, mid, labelTop, "middle", "message-label", "", scene.LayerLabel)
		l.extent(mid-w/2, mid+w/2)
	}

	startX, endX := x1, x2
	// Komunikat z grotem-krzyżykiem, który kończy życie odbiorcy: jeden krzyżyk na końcu strzałki
	// zamiast dwóch obok siebie.
	merged := e.Destroyed-1 == e.To && e.Head == HeadCross && !e.Bidirectional
	if merged {
		endX = l.x[e.To]
	} else {
		endX -= dir * l.head(e.Head, x2, y, dir)
	}
	if e.Bidirectional {
		startX += dir * l.head(e.Head, x1, y, -dir)
	}
	l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "message",
		X: startX, Y: y, X2: endX, Y2: y, Stroke: colorLine, StrokeWidth: 1.5, Dash: dash(e.LineStyle)})
	l.badge(x1, y)
	l.extent(math.Min(x1, x2), math.Max(x1, x2))

	if e.ActivateTo {
		l.active[e.To] = append(l.active[e.To], y)
	}
	if e.DeactivateFrom {
		l.deactivate(e.From, y)
	}
	l.lastY = y
	l.cursor = y + l.u(28)
	if c := e.Created - 1; c >= 0 {
		l.participants(y-l.header/2, func(p *Participant) bool { return p == l.d.Participants[c] })
		l.start[c] = y + l.header/2
		l.cursor = y + l.header/2 + l.u(24)
	}
	if merged {
		l.destroy(e, y, 0)
	} else {
		l.destroy(e, y, l.u(16))
	}
}

// destroy kończy linię życia uczestnika krzyżykiem tuż pod jego ostatnim komunikatem.
func (l *layout) destroy(e Event, y, below float64) {
	d := e.Destroyed - 1
	if d < 0 {
		return
	}
	for len(l.active[d]) > 0 {
		l.deactivate(d, y)
	}
	cy, r := y+below, l.u(7)
	l.end[d] = cy
	for _, s := range []float64{-1, 1} {
		l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerShape, Role: "destroy",
			X: l.x[d] - r, Y: cy - s*r, X2: l.x[d] + r, Y2: cy + s*r, Stroke: colorLine, StrokeWidth: 2})
	}
	l.cursor = math.Max(l.cursor, cy+r+l.u(20))
}

func (l *layout) selfMessage(e Event) {
	i := e.From
	reach, drop := l.u(36), l.u(26)
	level := len(l.active[i])
	xs := l.edge(i, 1, level)
	if e.ActivateTo {
		level++
	}
	xe := l.edge(i, 1, level)
	out := l.x[i] + l.u(8) + reach
	right := out

	// Etykieta stoi nad pętlą, jak nad każdym innym komunikatem; obok pętli poszerzałaby kolumnę.
	if len(e.Text) > 0 {
		tx := xs + l.u(6)
		if l.d.Autonumber {
			tx += l.u(10)
		}
		w, h := l.maxWidth(e.Text), float64(len(e.Text))*l.lh
		if i == len(l.x)-1 && tx+w > out {
			// Ostatni uczestnik: etykieta kończy się nad pętlą i rośnie w lewo, przez własną linię życia.
			l.labelBackground(out-w, l.cursor, w, h)
			l.text(e.Text, out, l.cursor, "end", "message-label", "", scene.LayerLabel)
			l.extent(out-w, out)
		} else {
			l.text(e.Text, tx, l.cursor, "start", "message-label", "", scene.LayerLabel)
			right = math.Max(right, tx+w)
		}
		l.cursor += h + l.u(5)
	}
	y0 := l.cursor
	y1 := y0 + drop

	// Komunikat do siebie jest jedną krzywą: od razu widać, że wraca do tego samego uczestnika.
	short := l.head(e.Head, xe, y1, -1)
	bulge := (out - l.x[i]) * 4 / 3
	l.sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerShape, Role: "message",
		Segments: []scene.Segment{
			{Cmd: 'M', To: scene.Point{X: xs, Y: y0}},
			{Cmd: 'C', C1: scene.Point{X: l.x[i] + bulge, Y: y0}, C2: scene.Point{X: l.x[i] + bulge, Y: y1}, To: scene.Point{X: xe + short + l.u(6), Y: y1}},
			{Cmd: 'L', To: scene.Point{X: xe + short, Y: y1}},
		},
		Stroke: colorLine, StrokeWidth: 1.5, Dash: dash(e.LineStyle)})
	l.badge(xs, y0)
	l.extent(l.x[i], right)

	if e.ActivateTo {
		l.active[i] = append(l.active[i], y1)
	}
	if e.DeactivateFrom {
		l.deactivate(i, y1)
	}
	l.lastY = y1
	l.cursor = y1 + l.u(28)
	l.destroy(e, y1, l.u(16))
}

func (l *layout) note(e Event) {
	padX, padY := l.u(10), l.u(8)
	w := l.maxWidth(e.Text) + 2*padX
	h := float64(len(e.Text))*l.lh + 2*padY
	var x0 float64
	switch e.Placement {
	case NoteRightOf:
		x0 = l.x[e.Over[0]] + l.u(16)
	case NoteLeftOf:
		x0 = l.x[e.Over[0]] - l.u(16) - w
	default:
		lo, hi := l.x[e.Over[0]], l.x[e.Over[1]]
		if e.Over[0] != e.Over[1] {
			w = math.Max(w, hi-lo+2*l.u(20))
		}
		x0 = (lo+hi)/2 - w/2
	}
	y0 := l.cursor
	l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "note", X: x0, Y: y0, W: w, H: h,
		Fill: colorNoteFill, Stroke: colorNoteStroke, StrokeWidth: 1})
	if l.o.Shadows {
		l.sc.AddShadows(len(l.sc.Items)-1, l.u(3), "note")
	}
	l.text(e.Text, x0+w/2, y0+padY, "middle", "note-label", "", scene.LayerLabel)
	l.extent(x0, x0+w)
	l.cursor = y0 + h + l.u(22)
}

func (l *layout) blockStart(e Event) {
	l.cursor += l.u(4)
	f := &frame{kind: e.Block, title: strings.Join(e.Text, " "), color: e.Color, y0: l.cursor, first: len(l.sc.Items), minX: math.Inf(1), maxX: math.Inf(-1)}
	l.frames = append(l.frames, f)
	if e.Block == "rect" {
		l.fills = append(l.fills, blend(e.Color, l.fills[len(l.fills)-1]))
		l.cursor += l.u(8)
	} else {
		l.cursor += l.lh + l.u(4) + l.u(18)
	}
}

func (l *layout) blockSection(e Event) {
	f := l.frames[len(l.frames)-1]
	f.sections = append(f.sections, section{y: l.cursor, text: strings.Join(e.Text, " ")})
	l.cursor += l.lh + l.u(20)
}

func (l *layout) blockEnd() {
	f := l.frames[len(l.frames)-1]
	l.frames = l.frames[:len(l.frames)-1]
	n := len(l.x)
	if math.IsInf(f.minX, 1) {
		f.minX, f.maxX = l.x[0]-l.boxW[0]/2, l.x[n-1]+l.boxW[n-1]/2
	}
	pad := l.u(12)
	x0, x1 := f.minX-pad, f.maxX+pad
	y1 := l.cursor

	if f.kind == "rect" {
		l.fills = l.fills[:len(l.fills)-1]
		// SEQ-020: wyróżnienie obejmuje treść z równym marginesem u góry i u dołu, więc jego środek
		// leży między pierwszym podpisem a ostatnią strzałką, a nie na samej strzałce.
		top, bottom := math.Inf(1), math.Inf(-1)
		for _, it := range l.sc.Items[f.first:] {
			switch it.Role {
			case "activation", "lifeline", "label-background", "arrowhead":
				continue
			}
			b := it.Bounds()
			top, bottom = math.Min(top, b.Y0), math.Max(bottom, b.Y1)
		}
		if !math.IsInf(top, 1) && top > f.y0 && bottom+(top-f.y0) < y1 {
			y1 = math.Round(bottom + (top - f.y0))
			l.cursor = y1 + l.u(14)
		}
		fill, opacity := splitColor(f.color)
		if strings.HasPrefix(f.color, "rgba(") && opacity == 0 {
			fill = "" // w pełni przezroczyste wyróżnienie: prostokąt bez wypełnienia
		}
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "highlight",
			X: x0, Y: f.y0, W: x1 - x0, H: y1 - f.y0, Fill: fill, FillOpacity: opacity})
	} else {
		tabW, tabH := l.width(f.kind)+2*l.u(8), l.lh+l.u(4)
		bracket := func(s string) string {
			if s == "" {
				return ""
			}
			return "[" + s + "]"
		}
		// Ramka musi pomieścić zakładkę z tytułem i etykiety sekcji.
		need := tabW + l.u(8) + l.width(bracket(f.title)) + l.u(8)
		for _, s := range f.sections {
			need = math.Max(need, l.width(bracket(s.text))+2*l.u(8))
		}
		x1 = math.Max(x1, x0+need)
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerFrame, Role: "frame",
			X: x0, Y: f.y0, W: x1 - x0, H: y1 - f.y0, RX: l.u(3), Stroke: colorFrame, StrokeWidth: 1.5, Dash: "2 3"})
		l.sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerFrame, Role: "frame-tab",
			X: x0, Y: f.y0, W: tabW, H: tabH, Fill: colorTabFill, Stroke: colorFrame, StrokeWidth: 1})
		l.text([]string{f.kind}, x0+tabW/2, f.y0+l.u(2), "middle", "frame-kind", "", scene.LayerLabel)
		l.labelBackground(x0+tabW+l.u(8), f.y0+l.u(2), l.width(bracket(f.title)), l.lh)
		l.text([]string{bracket(f.title)}, x0+tabW+l.u(8), f.y0+l.u(2), "start", "frame-title", "", scene.LayerLabel)
		for _, s := range f.sections {
			l.sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerFrame, Role: "frame-divider",
				X: x0, Y: s.y, X2: x1, Y2: s.y, Stroke: colorFrame, StrokeWidth: 1, Dash: "6 4"})
			l.labelBackground(x0+l.u(8), s.y+l.u(6), l.width(bracket(s.text)), l.lh)
			l.text([]string{bracket(s.text)}, x0+l.u(8), s.y+l.u(6), "start", "frame-title", "", scene.LayerLabel)
		}
	}
	l.cursor = y1 + l.u(24)
	l.extent(x0, x1)
}

var rgbaPattern = syntax.Lazy(`^rgba\((\d+),(\d+),(\d+),([0-9.]+)\)$`)

// splitColor rozdziela kolor z kanałem alfa na kolor i krycie, bo SVG 1.1 nie zna rgba().
func splitColor(c string) (fill string, opacity float64) {
	if m := rgbaPattern.FindStringSubmatch(c); m != nil {
		a, _ := strconv.ParseFloat(m[4], 64)
		return "rgb(" + m[1] + "," + m[2] + "," + m[3] + ")", a
	}
	return c, 0
}

var rgbPattern = syntax.Lazy(`^rgba?\((\d+),(\d+),(\d+)(?:,([0-9.]+))?\)$`)

// parseColor czyta kolor w zapisie #rgb, #rrggbb, rgb() albo rgba().
func parseColor(c string) (r, g, b, a float64, ok bool) {
	c = strings.Join(strings.Fields(c), "")
	if m := rgbPattern.FindStringSubmatch(c); m != nil {
		r, _ = strconv.ParseFloat(m[1], 64)
		g, _ = strconv.ParseFloat(m[2], 64)
		b, _ = strconv.ParseFloat(m[3], 64)
		a = 1
		if m[4] != "" {
			a, _ = strconv.ParseFloat(m[4], 64)
		}
		return math.Min(r, 255), math.Min(g, 255), math.Min(b, 255), math.Min(a, 1), true
	}
	if strings.HasPrefix(c, "#") && (len(c) == 4 || len(c) == 7) {
		hex := c[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		v, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		return float64(v >> 16), float64(v >> 8 & 0xFF), float64(v & 0xFF), 1, true
	}
	return 0, 0, 0, 0, false
}

// blend zwraca nieprzezroczysty kolor powstały z nałożenia top na under.
func blend(top, under string) string {
	tr, tg, tb, ta, ok := parseColor(top)
	ur, ug, ub, _, ok2 := parseColor(under)
	if !ok || !ok2 {
		return under
	}
	mix := func(t, u float64) string { return strconv.Itoa(int(math.Round(t*ta + u*(1-ta)))) }
	return "rgb(" + mix(tr, ur) + "," + mix(tg, ug) + "," + mix(tb, ub) + ")"
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
