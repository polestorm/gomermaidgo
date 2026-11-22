package timeline

import (
	"math"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// layoutRow rozmieszcza oś czasu tak jak Mermaid: okresy w rzędzie od lewej do prawej, pod nimi
// oś ze strzałką, a pod osią zdarzenia każdego okresu jedno pod drugim. Sekcja ma nagłówek nad
// swoimi okresami.
func layoutRow(d *Diagram, o Options) *scene.Scene {
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
	block := func(lines []string, cx, top float64, role string, bold bool) {
		for k, s := range lines {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: cx, Y: top + float64(k)*lh + lh/2 + 0.35*fs,
				W: width(s), H: lh, Text: s, Anchor: "middle", Bold: bold, Fill: colorText})
		}
	}

	colW, gap := u(150), u(10)
	left := func(i int) float64 { return float64(i) * (colW + gap) }
	labels := make([][]string, len(d.Periods))
	periodH := 0.0
	for i, p := range d.Periods {
		labels[i] = textwrap.Lines(p.Label, colW-2*u(8), width)
		periodH = math.Max(periodH, float64(len(labels[i]))*lh+2*u(8))
	}
	type span struct {
		from, to int
		lines    []string
	}
	var spans []span
	sectionH := 0.0
	for i, p := range d.Periods {
		if p.Section == "" {
			continue
		}
		if n := len(spans); n > 0 && spans[n-1].to == i-1 && d.Periods[i-1].Section == p.Section {
			spans[n-1].to = i
			continue
		}
		spans = append(spans, span{from: i, to: i})
	}
	for k := range spans {
		w := left(spans[k].to) + colW - left(spans[k].from)
		spans[k].lines = textwrap.Lines([]string{d.Periods[spans[k].from].Section}, w-2*u(8), width)
		sectionH = math.Max(sectionH, float64(len(spans[k].lines))*lh+2*u(6))
	}
	// Ton: sekcja ma jeden dla wszystkich swoich okresów; bez sekcji każdy okres ma własny.
	tones := make([]int, len(d.Periods))
	next := 0
	for i, p := range d.Periods {
		if i > 0 && p.Section != "" && p.Section == d.Periods[i-1].Section {
			tones[i] = tones[i-1]
			continue
		}
		tones[i] = next
		next++
	}
	palette := func(t int) (fill, line string) {
		if !o.Colors {
			return colorPeriodFill, colorPeriodLine
		}
		p := scene.Distinct(t)
		return p.Fill, p.Stroke
	}

	y := 0.0
	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "title", X: 0, Y: lh/2 + 0.35*fs, W: width(d.Title), H: lh,
			Text: d.Title, Anchor: "start", Fill: colorText})
		y = lh + u(14)
	}
	for _, s := range spans {
		fill, line := palette(tones[s.from])
		if !o.Colors {
			fill = colorSectionFill
		}
		w := left(s.to) + colW - left(s.from)
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "timeline-section", X: left(s.from), Y: y, W: w, H: sectionH, RX: u(4),
			Fill: fill, Stroke: line, StrokeWidth: 1.5})
		block(s.lines, left(s.from)+w/2, y+(sectionH-float64(len(s.lines))*lh)/2, "timeline-section-title", true)
	}
	if len(spans) > 0 {
		y += sectionH + u(8)
	}
	periodTop := y
	axisY := periodTop + periodH + u(20)
	right := left(len(d.Periods)-1) + colW
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "timeline-axis", X: 0, Y: axisY, X2: right + u(8), Y2: axisY, Stroke: colorAxis, StrokeWidth: 2})
	sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerLifeline, Role: "timeline-axis", Fill: colorAxis,
		Points: []scene.Point{{X: right + u(20), Y: axisY}, {X: right + u(8), Y: axisY - u(6)}, {X: right + u(8), Y: axisY + u(6)}}})

	from := len(sc.Items)
	for i, p := range d.Periods {
		fill, line := palette(tones[i])
		x, cx := left(i), left(i)+colW/2
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "timeline-period", X: x, Y: periodTop, W: colW, H: periodH, RX: u(4),
			Fill: fill, Stroke: line, StrokeWidth: 1.5})
		block(labels[i], cx, periodTop+(periodH-float64(len(labels[i]))*lh)/2, "timeline-period-label", false)
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "timeline-mark", X: cx, Y: axisY, R: u(5), Fill: colorEventFill, Stroke: colorAxis, StrokeWidth: 2})
		eventLine := colorEventLine
		if o.Colors {
			eventLine = line
		}
		top := axisY + u(20)
		for _, ev := range p.Events {
			lines := textwrap.Lines(ev, colW-2*u(8), width)
			h := float64(len(lines))*lh + 2*u(7)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "timeline-event", X: x, Y: top, W: colW, H: h, RX: u(4),
				Fill: colorEventFill, Stroke: eventLine, StrokeWidth: 1})
			block(lines, cx, top+u(7), "timeline-event-label", false)
			top += h + u(8)
		}
		// Łącznik okresu z jego zdarzeniami: linia przerywana od okresu do ostatniego zdarzenia.
		end := axisY
		if len(p.Events) > 0 {
			end = top - u(8)
		}
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "timeline-link", X: cx, Y: periodTop + periodH, X2: cx, Y2: end,
			Stroke: colorAxis, StrokeWidth: 1.5, Dash: "4 4"})
	}
	if o.Shadows {
		sc.AddShadows(from, u(3), "timeline-period", "timeline-event")
	}
	return sc
}
