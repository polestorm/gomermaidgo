// Package cynefin rozbiera i rysuje mapę Cynefin: pięć domen o stałym położeniu (złożona,
// skomplikowana, chaotyczna, oczywista i zagubienie w środku), elementy w domenach i przejścia.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package cynefin

import (
	"math"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni mapy Cynefin.
const (
	CodeHeader    = "cynefin-header"
	CodeStatement = "cynefin-statement"
	maxItems      = 120
	maxShown      = 3 // elementów zagubienia na rysunku; reszta jako „+N more”
)

// Domeny w kolejności rysowania.
const (
	Complex     = "complex"
	Complicated = "complicated"
	Chaotic     = "chaotic"
	Clear       = "clear"
	Confusion   = "confusion"
)

var domains = []string{Complex, Complicated, Chaotic, Clear, Confusion}

// Transition to przejście między domenami.
type Transition struct {
	From, To, Label string
	Line            int
}

// Diagram to elementy domen i przejścia.
type Diagram struct {
	Title       string
	Items       map[string][]string
	Transitions []Transition
	AccTitle    string
	AccDescr    string
}

func isDomain(s string) bool {
	for _, d := range domains {
		if s == d {
			return true
		}
	}
	return false
}

// Parse rozbiera źródło: nazwa domeny w osobnej linii otwiera jej listę, linie w cudzysłowie
// to elementy, „a --> b : "podpis"” to przejście.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{Items: map[string][]string{}}
	var acc syntax.Accessibility
	seenHeader := false
	current := ""
	count := 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "cynefin-beta" && s != "cynefin" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"cynefin-beta\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		switch {
		case s == "title" || strings.HasPrefix(s, "title "):
			d.Title = unquote(strings.TrimSpace(strings.TrimPrefix(s, "title")))
		case isDomain(s):
			current = s
			if d.Items[s] == nil {
				d.Items[s] = []string{}
			}
		case strings.Contains(s, "-->"):
			from, rest, _ := strings.Cut(s, "-->")
			to, label, _ := strings.Cut(rest, ":")
			from, to = strings.TrimSpace(from), strings.TrimSpace(to)
			if !isDomain(from) || !isDomain(to) {
				return nil, syntax.Errorf(line, CodeStatement, "a transition joins two domains (complex, complicated, clear, chaotic, confusion): %q", s)
			}
			if from != to { // przejście domeny do niej samej nie ma rysunku
				d.Transitions = append(d.Transitions, Transition{from, to, unquote(strings.TrimSpace(label)), line})
			}
		case strings.HasPrefix(s, `"`):
			if current == "" {
				return nil, syntax.Errorf(line, CodeStatement, "an item must follow a domain name")
			}
			if len(s) < 2 || !strings.HasSuffix(s, `"`) {
				return nil, syntax.Errorf(line, CodeStatement, "item is not closed with a quote: %q", s)
			}
			if count++; count > maxItems {
				return nil, syntax.Errorf(line, CodeStatement, "too many items (limit %d)", maxItems)
			}
			d.Items[current] = append(d.Items[current], s[1:len(s)-1])
		default:
			return nil, syntax.Errorf(line, CodeStatement, "unknown statement %q (items are written in quotes)", s)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"cynefin-beta\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	d.AccTitle, d.AccDescr = acc.Title, acc.Descr
	return d, nil
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // domeny w tonach palety
	Shadows    bool // cień pod elementami
}

const (
	colorText    = "#1b1f24"
	colorSub     = "#55657a"
	colorFrame   = "#3b4754"
	colorBadge   = "#9aa5b1"
	colorMiddle  = "#f3f5f8"
	colorArrow   = "#3b4754"
	colorBadgeBg = "#ffffff"
	colorLabelBg = "#ffffff"
)

// names i descriptions to podpisy domen: nazwa, model działania i rodzaj praktyki.
var names = map[string]string{Complex: "Complex", Complicated: "Complicated", Chaotic: "Chaotic", Clear: "Clear", Confusion: "Confusion"}

var descriptions = map[string][]string{
	Complex:     {"Probe – Sense – Respond", "Emergent practice"},
	Complicated: {"Sense – Analyse – Respond", "Good practice"},
	Chaotic:     {"Act – Sense – Respond", "Novel practice"},
	Clear:       {"Sense – Categorise – Respond", "Best practice"},
}

// tones to odcienie domen w trybie kolorowym.
var tones = map[string]string{Complex: "#e8f0fb", Complicated: "#e6f4ea", Chaotic: "#fbe9e7", Clear: "#fff6dc"}

// block to treść domeny: nazwa, opis i elementy, każdy element w liniach.
type block struct {
	items  [][]string
	w, h   float64
	badgeW []float64
}

// Layout rysuje planszę: cztery domeny w ćwiartkach (złożona i skomplikowana u góry, chaotyczna
// i oczywista u dołu), łagodnie wygięte granice, z których dolna połowa pionowej to klif między
// domeną oczywistą a chaotyczną, i zagubienie w elipsie pośrodku. Treść domen stoi przy
// zewnętrznych krawędziach, więc środek zostaje na przejścia.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return math.Round(v * fs / 16) }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.8)
	slh := math.Round(small * 1.25)
	item := math.Round(fs * 0.875)
	ilh := math.Round(item * 1.25)
	pad, gap := u(16), u(6)
	sc := &scene.Scene{Title: d.AccTitle, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	blocks := map[string]*block{}
	for _, dom := range domains {
		b := &block{}
		b.w = o.Measure(names[dom], fs) * 1.06
		for _, s := range descriptions[dom] {
			b.w = math.Max(b.w, o.Measure(s, small))
		}
		b.h = lh + float64(len(descriptions[dom]))*slh
		list := d.Items[dom]
		if dom == Confusion && len(list) > maxShown {
			list = append(append([]string(nil), list[:maxShown]...), "+"+itoa(len(d.Items[dom])-maxShown)+" more")
		}
		for _, s := range list {
			lines := textwrap.Lines(textwrap.Split(s), u(200), func(t string) float64 { return o.Measure(t, item) })
			w := 0.0
			for _, l := range lines {
				w = math.Max(w, o.Measure(l, item))
			}
			b.items = append(b.items, lines)
			b.badgeW = append(b.badgeW, w+2*u(8))
			b.w = math.Max(b.w, w+2*u(8))
			b.h += gap + float64(len(lines))*ilh + 2*u(4)
		}
		blocks[dom] = b
	}
	// Elipsa zagubienia opisuje prostokąt jej treści.
	cb := blocks[Confusion]
	ellW := math.Max(u(150), math.Sqrt2*(cb.w+u(16)))
	ellH := math.Max(u(90), math.Sqrt2*(cb.h+u(8)))
	// Punkty przejść leżą wokół elipsy; treść domen musi się zmieścić nad nimi albo pod nimi.
	ax, ay := ellW/2+u(56), ellH/2+u(40)
	qW, qH := math.Max(u(300), ax+u(90)), u(200)
	// Podpisy przejść stoją po zewnętrznej stronie łuków i muszą się zmieścić w ramce.
	for _, t := range d.Transitions {
		if t.Label != "" {
			qW = math.Max(qW, ax+u(18)+o.Measure(t.Label, small)+u(24))
		}
	}
	for _, dom := range domains[:4] {
		b := blocks[dom]
		qW = math.Max(qW, b.w+2*pad)
		qH = math.Max(qH, b.h+pad+ay+u(28))
	}

	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "cynefin-title", X: 0, Y: -qH - u(16), W: o.Measure(d.Title, fs*1.15), H: lh,
			Text: d.Title, Anchor: "middle", Bold: true, FontSize: math.Round(fs * 1.15), Fill: colorText})
	}
	// Tła domen w trybie kolorowym: ćwiartki ramki.
	if o.Colors {
		for _, dom := range domains[:4] {
			sx, sy := side(dom)
			x, y := 0.0, 0.0
			if sx < 0 {
				x = -qW
			}
			if sy < 0 {
				y = -qH
			}
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "cynefin-domain", X: x, Y: y, W: qW, H: qH, Fill: tones[dom]})
		}
	}
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "cynefin-frame", X: -qW, Y: -qH, W: 2 * qW, H: 2 * qH, RX: u(6), Fill: "none", Stroke: colorFrame, StrokeWidth: 1.5})
	// Granice: pionowa wygięta w prawo u góry i w lewo u dołu (klif), pozioma odwrotnie.
	bend := u(16)
	pt := func(x, y float64) scene.Point { return scene.Point{X: x, Y: y} }
	curve := func(a, c1, c2, b scene.Point, width float64, role string) {
		sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerShape, Role: role, Fill: "none", Stroke: colorFrame, StrokeWidth: width,
			Segments: []scene.Segment{{Cmd: 'M', To: a}, {Cmd: 'C', C1: c1, C2: c2, To: b}}})
	}
	curve(pt(0, -qH), pt(bend, -qH*0.66), pt(bend, -qH*0.33), pt(0, 0), 1.5, "cynefin-boundary")
	curve(pt(-qW, 0), pt(-qW*0.66, bend), pt(-qW*0.33, bend), pt(0, 0), 1.5, "cynefin-boundary")
	curve(pt(0, 0), pt(qW*0.33, -bend), pt(qW*0.66, -bend), pt(qW, 0), 1.5, "cynefin-boundary")
	curve(pt(0, 0), pt(-bend, qH*0.33), pt(-bend, qH*0.66), pt(0, qH), 3.5, "cynefin-cliff")
	sc.Add(scene.Item{Kind: scene.Ellipse, Layer: scene.LayerShape, Role: "cynefin-confusion", X: 0, Y: 0, W: ellW, H: ellH, Fill: colorMiddle, Stroke: colorFrame, StrokeWidth: 1.5})

	badges := len(sc.Items)
	// Treść domeny: u góry nazwa, opis, elementy w dół; u dołu kolejność odwrócona, nazwa przy krawędzi.
	text := func(s string, x, mid, size float64, anchor string, bold, italic bool, fill, role string) {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: mid + 0.35*size, W: o.Measure(s, size), H: size * 1.25,
			Text: s, Anchor: anchor, Bold: bold, Italic: italic, FontSize: size, Fill: fill})
	}
	for _, dom := range domains {
		b := blocks[dom]
		sx, sy := side(dom)
		var x float64
		anchor := "start"
		switch {
		case dom == Confusion:
			x, anchor = 0, "middle"
		case sx < 0:
			x = -qW + pad
		default:
			x, anchor = qW-pad, "end"
		}
		// Kolejne wiersze treści od góry: (wysokość, rysuj w punkcie środka).
		type piece struct {
			h    float64
			draw func(mid float64)
		}
		head := []piece{{lh, func(m float64) { text(names[dom], x, m, fs, anchor, true, false, colorText, "cynefin-domain-label") }}}
		for _, s := range descriptions[dom] {
			s := s
			head = append(head, piece{slh, func(m float64) { text(s, x, m, small, anchor, false, true, colorSub, "cynefin-domain-description") }})
		}
		var items []piece
		for k, lines := range b.items {
			lines, w := lines, b.badgeW[k]
			h := float64(len(lines))*ilh + 2*u(4)
			items = append(items, piece{h + gap, func(m float64) {
				bx := x
				switch anchor {
				case "end":
					bx = x - w
				case "middle":
					bx = x - w/2
				}
				sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "cynefin-item", X: bx, Y: m - h/2 + gap/2, W: w, H: h, RX: u(4),
					Fill: colorBadgeBg, Stroke: colorBadge, StrokeWidth: 1})
				for i, l := range lines {
					text(l, bx+w/2, m-h/2+gap/2+u(4)+(float64(i)+0.5)*ilh, item, "middle", false, false, colorText, "cynefin-item-label")
				}
			}})
		}
		seq := append(head, items...)
		if sy > 0 && dom != Confusion {
			seq = append(items, reverse(head)...)
		}
		total := 0.0
		for _, p := range seq {
			total += p.h
		}
		y := -qH + pad
		switch {
		case dom == Confusion:
			y = -total / 2
		case sy > 0:
			y = qH - pad - total
		}
		for _, p := range seq {
			p.draw(y + p.h/2)
			y += p.h
		}
	}
	// Przejścia: łuk między punktami domen, wygięty od środka planszy, z podpisem.
	anchorOf := func(dom string) scene.Point {
		if dom == Confusion {
			return pt(0, 0)
		}
		sx, sy := side(dom)
		return pt(sx*ax, sy*ay)
	}
	for _, t := range d.Transitions {
		a, b := anchorOf(t.From), anchorOf(t.To)
		// Koniec przy zagubieniu leży na brzegu elipsy.
		edge := func(p, toward scene.Point) scene.Point {
			dx, dy := toward.X-p.X, toward.Y-p.Y
			k := 1 / math.Sqrt(dx*dx/(ellW*ellW/4)+dy*dy/(ellH*ellH/4))
			return pt(p.X+dx*k, p.Y+dy*k)
		}
		if t.From == Confusion {
			a = edge(a, b)
		}
		if t.To == Confusion {
			b = edge(b, a)
		}
		mx, my := (a.X+b.X)/2, (a.Y+b.Y)/2
		dx, dy := b.X-a.X, b.Y-a.Y
		l := math.Hypot(dx, dy)
		nx, ny := -dy/l, dx/l
		if nx*mx+ny*my < 0 { // normalna od środka planszy
			nx, ny = -nx, -ny
		}
		off := u(18)
		if mx == 0 && my == 0 { // po przekątnej: łuk mija elipsę
			off = math.Max(ellW, ellH)/2 + u(20)
		}
		c := pt(mx+nx*off*2, my+ny*off*2) // wierzchołek łuku leży w połowie drogi do punktu kontrolnego
		// Groty skracają łuk, żeby nie wchodził w grot.
		head := u(10)
		end := b
		tx, ty := b.X-c.X, b.Y-c.Y
		tl := math.Hypot(tx, ty)
		tx, ty = tx/tl, ty/tl
		end = pt(b.X-tx*head*0.8, b.Y-ty*head*0.8)
		sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerShape, Role: "cynefin-transition", Fill: "none", Stroke: colorArrow, StrokeWidth: 1.5,
			Segments: []scene.Segment{{Cmd: 'M', To: a}, {Cmd: 'C', C1: pt(a.X+(c.X-a.X)*2/3, a.Y+(c.Y-a.Y)*2/3), C2: pt(end.X+(c.X-end.X)*2/3, end.Y+(c.Y-end.Y)*2/3), To: end}}})
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "cynefin-transition",
			Points: []scene.Point{b, pt(b.X-tx*head-ty*head*0.45, b.Y-ty*head+tx*head*0.45), pt(b.X-tx*head+ty*head*0.45, b.Y-ty*head-tx*head*0.45)},
			Fill:   colorArrow, Stroke: colorArrow, StrokeWidth: 1})
		if t.Label != "" {
			// Podpis po zewnętrznej stronie łuku, na nieprzezroczystym podkładzie: przykrywa granice domen.
			w := o.Measure(t.Label, small)
			reach := math.Abs(nx)*(w/2+u(6)) + math.Abs(ny)*(slh/2+u(4))
			lx, ly := mx+nx*(off+reach), my+ny*(off+reach)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabel, Role: "cynefin-transition-label", X: lx - w/2 - u(4), Y: ly - slh/2, W: w + u(8), H: slh, RX: u(3),
				Fill: colorLabelBg})
			text(t.Label, lx, ly, small, "middle", false, false, colorText, "cynefin-transition-label")
		}
	}
	if o.Shadows {
		sc.AddShadows(badges, u(2), "cynefin-item")
	}
	return sc
}

// side zwraca ćwiartkę domeny: -1 lewo albo góra, 1 prawo albo dół.
func side(dom string) (float64, float64) {
	switch dom {
	case Complex:
		return -1, -1
	case Complicated:
		return 1, -1
	case Chaotic:
		return -1, 1
	case Clear:
		return 1, 1
	}
	return 0, 0
}

func reverse[T any](s []T) []T {
	out := make([]T, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
