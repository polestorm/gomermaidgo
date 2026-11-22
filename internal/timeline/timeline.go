// Package timeline rozbiera i rysuje oś czasu (timeline).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package timeline

import (
	"math"
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni osi czasu.
const (
	CodeHeader    = "timeline-header"
	CodeStatement = "timeline-statement"
	CodeEmpty     = "timeline-empty"
	maxPeriods    = 100
	maxEvents     = 20
)

// Period to okres na osi: jego nazwa i zdarzenia. Section to tytuł sekcji, do której należy.
type Period struct {
	Label   []string
	Events  [][]string
	Section string
	Line    int
}

// Diagram to oś czasu.
type Diagram struct {
	Title    string
	Periods  []*Period
	AccTitle string
	AccDescr string
	// List wybiera układ z osią z góry na dół (dyrektywa „%% layout: list”) zamiast osi poziomej
	// jak w Mermaid.
	List bool
}

var listDirective = regexp.MustCompile(`^%%\s*layout:\s*list\s*$`)

// Parse rozbiera źródło osi czasu.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	seenHeader := false
	section := ""
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if listDirective.MatchString(s) {
			// Dyrektywa w komentarzu: inne narzędzia ją pomijają i rysują układ domyślny.
			d.List = true
			continue
		}
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			// Kierunek po nagłówku jest przyjmowany i pomijany: o układzie decyduje dyrektywa.
			if s != "timeline" && s != "timeline TD" && s != "timeline LR" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"timeline\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		switch {
		case s == "title" || strings.HasPrefix(s, "title "):
			d.Title = strings.TrimSpace(strings.TrimPrefix(s, "title"))
		case s == "section" || strings.HasPrefix(s, "section "):
			if section = strings.TrimSpace(strings.TrimPrefix(s, "section")); section == "" {
				return nil, syntax.Errorf(line, CodeStatement, "section needs a title")
			}
		default:
			parts := strings.Split(s, ":")
			var p *Period
			if strings.TrimSpace(parts[0]) == "" {
				// Linia zaczynająca się od dwukropka dopisuje zdarzenia poprzedniemu okresowi.
				if len(d.Periods) == 0 || d.Periods[len(d.Periods)-1].Section != section {
					return nil, syntax.Errorf(line, CodeStatement, "events without a time period before them")
				}
				p = d.Periods[len(d.Periods)-1]
			} else {
				if len(d.Periods) >= maxPeriods {
					return nil, syntax.Errorf(line, CodeStatement, "too many time periods (limit %d)", maxPeriods)
				}
				p = &Period{Label: textwrap.Split(parts[0]), Section: section, Line: line}
				if len(p.Label) == 0 {
					return nil, syntax.Errorf(line, CodeStatement, "time period has no text")
				}
				d.Periods = append(d.Periods, p)
			}
			for _, part := range parts[1:] {
				if ev := textwrap.Split(part); len(ev) > 0 {
					if len(p.Events) >= maxEvents {
						return nil, syntax.Errorf(line, CodeStatement, "too many events in one time period (limit %d)", maxEvents)
					}
					p.Events = append(p.Events, ev)
				}
			}
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"timeline\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(d.Periods) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the timeline needs at least one time period")
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
	Colors     bool // sekcje (albo okresy, gdy sekcji nie ma) w kolejnych kolorach palety
	Shadows    bool // cień pod okresami i zdarzeniami
}

const (
	colorText        = "#1b1f24"
	colorAxis        = "#3b4754"
	colorPeriodFill  = "#e3e8ef"
	colorPeriodLine  = "#55657a"
	colorEventFill   = "#ffffff"
	colorEventLine   = "#9aa5b1"
	colorSectionFill = "#f1f4f8"
)

// Layout rozmieszcza oś czasu: domyślnie poziomo, jak w Mermaid; z dyrektywą „%% layout: list”
// z góry na dół.
func Layout(d *Diagram, o Options) *scene.Scene {
	if d.List {
		return layoutList(d, o)
	}
	return layoutRow(d, o)
}

// layoutList rozmieszcza oś czasu z góry na dół: oś z lewej, przy niej okresy, a zdarzenia okresu
// w rzędach po prawej. Rysunek rośnie w dół i mieści się w szerokości strony przy dowolnej
// liczbie okresów.
func layoutList(d *Diagram, o Options) *scene.Scene {
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
	block := func(lines []string, cx, top float64, role string) {
		for k, s := range lines {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: cx, Y: top + float64(k)*lh + lh/2 + 0.35*fs,
				W: width(s), H: lh, Text: s, Anchor: "middle", Fill: colorText})
		}
	}
	size := func(lines []string, padX, padY float64) (w, h float64) {
		for _, s := range lines {
			w = math.Max(w, width(s))
		}
		return math.Round(w + 2*padX), float64(len(lines))*lh + 2*padY
	}

	// Kolumna okresów ma wspólną szerokość; zdarzenia zajmują resztę szerokości strony.
	// Okresy stoją po lewej stronie osi, zdarzenia po prawej, tak jak w układzie domyślnym
	// okresy nad osią, a zdarzenia pod nią.
	periodX := 0.0
	periodW := u(90)
	labels := make([][]string, len(d.Periods))
	for i, p := range d.Periods {
		labels[i] = textwrap.Lines(p.Label, u(150), width)
		w, _ := size(labels[i], u(12), 0)
		periodW = math.Max(periodW, w)
	}
	axisX := periodX + periodW + u(20)
	eventsX := axisX + u(20)
	room := math.Min(u(650)-eventsX, u(300))
	type tile struct {
		lines      []string
		x, y, w, h float64
	}
	// Wspólna szerokość pól zdarzeń: najszersze zdarzenie po złamaniu.
	eventW := u(120)
	for _, p := range d.Periods {
		for _, ev := range p.Events {
			w, _ := size(textwrap.Lines(ev, room-2*u(10), width), u(10), 0)
			eventW = math.Max(eventW, w)
		}
	}
	y := 0.0
	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "title", X: 0, Y: lh/2 + 0.35*fs, W: width(d.Title), H: lh,
			Text: d.Title, Anchor: "start", Fill: colorText})
		y = lh + u(16)
	}
	type row struct {
		top, h float64
		tiles  []tile
		tone   int
	}
	rows := make([]row, len(d.Periods))
	type strip struct {
		top  float64
		text string
		tone int
	}
	var strips []strip
	right := eventsX
	tone, section := -1, ""
	for i, p := range d.Periods {
		// Sekcja ma jeden ton dla wszystkich swoich okresów; bez sekcji każdy okres ma własny.
		if p.Section == "" {
			tone++
		} else if p.Section != section {
			tone++
			strips = append(strips, strip{y, p.Section, tone})
			y += lh + 2*u(6) + u(12)
		}
		section = p.Section
		_, ph := size(labels[i], u(12), u(8))
		r := row{top: y, h: ph, tone: tone}
		// Zdarzenia okresu stoją jedno pod drugim, w kolejności ze źródła, w kolumnie o wspólnej
		// szerokości: tak jak w układzie domyślnym, tylko obok okresu zamiast pod nim.
		top := 0.0
		for k, ev := range p.Events {
			lines := textwrap.Lines(ev, room-2*u(10), width)
			_, h := size(lines, u(10), u(7))
			if k > 0 {
				top += u(6)
			}
			r.tiles = append(r.tiles, tile{lines, eventsX, y + top, eventW, h})
			top += h
		}
		right = math.Max(right, eventsX+eventW)
		r.h = math.Max(r.h, top)
		rows[i] = r
		y += r.h + u(16)
	}
	bottom := y - u(16)

	palette := func(t int) (fill, line string) {
		if !o.Colors {
			return colorPeriodFill, colorPeriodLine
		}
		p := scene.Distinct(t)
		return p.Fill, p.Stroke
	}
	for _, s := range strips {
		fill, _ := palette(s.tone)
		if !o.Colors {
			fill = colorSectionFill
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "timeline-section", X: periodX, Y: s.top,
			W: right - periodX, H: lh + 2*u(6), RX: u(4), Fill: fill})
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "timeline-section-title", X: periodX + u(12), Y: s.top + u(6) + lh/2 + 0.35*fs,
			W: width(s.text), H: lh, Text: s.text, Anchor: "start", Bold: true, Fill: colorText})
	}
	// Oś: od pierwszego okresu za ostatni, z grotem na końcu.
	first := rows[0].top + (float64(len(labels[0]))*lh+2*u(8))/2
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "timeline-axis", X: axisX, Y: first, X2: axisX, Y2: bottom + u(8),
		Stroke: colorAxis, StrokeWidth: 2})
	sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerLifeline, Role: "timeline-axis", Fill: colorAxis,
		Points: []scene.Point{{X: axisX, Y: bottom + u(20)}, {X: axisX - u(6), Y: bottom + u(8)}, {X: axisX + u(6), Y: bottom + u(8)}}})
	from := len(sc.Items)
	for i, r := range rows {
		fill, line := palette(r.tone)
		_, ph := size(labels[i], u(12), u(8))
		mid := r.top + ph/2
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "timeline-link", X: periodX + periodW, Y: mid, X2: axisX, Y2: mid, Stroke: colorAxis, StrokeWidth: 1.5})
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "timeline-mark", X: axisX, Y: mid, R: u(5), Fill: colorEventFill, Stroke: colorAxis, StrokeWidth: 2})
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "timeline-period", X: periodX, Y: r.top, W: periodW, H: ph, RX: u(4),
			Fill: fill, Stroke: line, StrokeWidth: 1.5})
		block(labels[i], periodX+periodW/2, r.top+u(8), "timeline-period-label")
		if len(r.tiles) > 0 {
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "timeline-link", X: axisX, Y: mid, X2: eventsX, Y2: mid, Stroke: line, StrokeWidth: 1})
		}
		eventLine := colorEventLine
		if o.Colors {
			eventLine = line
		}
		// Linia przerywana przez kolumnę zdarzeń, widoczna w odstępach między polami, pokazuje ich
		// kolejność, tak jak w układzie domyślnym.
		if n := len(r.tiles); n > 1 {
			first, last := r.tiles[0], r.tiles[n-1]
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "timeline-stem", X: first.x + first.w/2, Y: first.y + first.h, X2: first.x + first.w/2, Y2: last.y,
				Stroke: line, StrokeWidth: 1.5, Dash: "4 4"})
		}
		for _, t := range r.tiles {
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "timeline-event", X: t.x, Y: t.y, W: t.w, H: t.h, RX: u(4),
				Fill: colorEventFill, Stroke: eventLine, StrokeWidth: 1})
			block(t.lines, t.x+t.w/2, t.y+u(7), "timeline-event-label")
		}
	}
	if o.Shadows {
		sc.AddShadows(from, u(3), "timeline-period", "timeline-event")
	}
	return sc
}
