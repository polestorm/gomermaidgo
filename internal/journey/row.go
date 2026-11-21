package journey

import (
	"math"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

const (
	colorTaskFill    = "#eef2f7"
	colorTaskLine    = "#55657a"
	colorSectionLine = "#55657a"
	colorSectionRow  = "#dde3ea"
)

// Odcienie uczestników w rysunku neutralnym: sąsiednie różnią się jasnością.
var actorGrays = []string{"#3b4754", "#c9d1da", "#7b8794", "#ffffff", "#55657a", "#eef2f7"}

// layoutRow rozmieszcza podróż tak jak Mermaid: kroki w rzędzie pod nagłówkami sekcji, pod każdym
// krokiem buźka na wysokości oceny, z lewej legenda uczestników, a na krokach ich znaczniki.
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

	// Uczestnicy w kolejności pierwszego wystąpienia.
	var actors []string
	index := map[string]int{}
	actorW := 0.0
	for _, t := range d.Tasks {
		for _, a := range t.Actors {
			if _, ok := index[a]; !ok {
				index[a] = len(actors)
				actors = append(actors, a)
				actorW = math.Max(actorW, width(a))
			}
		}
	}
	actorColor := func(i int) string {
		if o.Colors {
			return scene.Palette[i%len(scene.Palette)].Stroke
		}
		return actorGrays[i%len(actorGrays)]
	}
	x0 := 0.0
	if len(actors) > 0 {
		x0 = math.Round(u(7)*2 + u(8) + actorW + u(28))
	}

	taskW, gap := u(150), u(10)
	names := make([][]string, len(d.Tasks))
	taskH := 0.0
	for i, t := range d.Tasks {
		names[i] = textwrap.Lines([]string{t.Name}, taskW-2*u(8), width)
		taskH = math.Max(taskH, float64(len(names[i]))*lh+2*u(8))
	}
	// Sekcje: kolejne kroki o tym samym tytule sekcji tworzą jeden nagłówek nad sobą.
	type span struct {
		from, to int
		lines    []string
	}
	var spans []span
	sectionH := 0.0
	for i, t := range d.Tasks {
		if t.Section == "" {
			continue
		}
		if n := len(spans); n > 0 && spans[n-1].to == i-1 && d.Tasks[i-1].Section == t.Section {
			spans[n-1].to = i
			continue
		}
		spans = append(spans, span{from: i, to: i})
	}
	for k := range spans {
		w := float64(spans[k].to-spans[k].from+1)*(taskW+gap) - gap
		spans[k].lines = textwrap.Lines([]string{d.Tasks[spans[k].from].Section}, w-2*u(8), width)
		sectionH = math.Max(sectionH, float64(len(spans[k].lines))*lh+2*u(6))
	}

	y := 0.0
	if d.Title != "" {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "title", X: x0, Y: lh/2 + 0.35*fs, W: width(d.Title), H: lh,
			Text: d.Title, Anchor: "start", Fill: colorText})
		y = lh + u(14)
	}
	left := func(i int) float64 { return x0 + float64(i)*(taskW+gap) }
	tone := func(k int) (fill, line string) {
		if o.Colors {
			p := scene.Palette[k%len(scene.Palette)]
			return p.Fill, p.Stroke
		}
		return colorTaskFill, colorTaskLine
	}
	sectionOf := make([]int, len(d.Tasks))
	for i := range sectionOf {
		sectionOf[i] = -1
	}
	for k, s := range spans {
		for i := s.from; i <= s.to; i++ {
			sectionOf[i] = k
		}
		fill, line := tone(k)
		if !o.Colors {
			fill, line = colorSectionRow, colorSectionLine
		}
		w := left(s.to) + taskW - left(s.from)
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "journey-section", X: left(s.from), Y: y, W: w, H: sectionH, RX: u(4),
			Fill: fill, Stroke: line, StrokeWidth: 1.5})
		block(s.lines, left(s.from)+w/2, y+(sectionH-float64(len(s.lines))*lh)/2, "journey-section-title", true)
	}
	if len(spans) > 0 {
		y += sectionH + u(8)
	}
	taskTop := y
	zoneTop := taskTop + taskH + u(14)
	faceR := u(15)
	level := func(score int) float64 { return zoneTop + faceR + float64(5-score)*u(26) }
	base := level(1) + faceR + u(14)
	right := left(len(d.Tasks)-1) + taskW

	// Oś u dołu: kierunek podróży.
	sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "journey-axis", X: x0, Y: base, X2: right + u(8), Y2: base, Stroke: colorLine, StrokeWidth: 2})
	sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerLifeline, Role: "journey-axis", Fill: colorLine,
		Points: []scene.Point{{X: right + u(20), Y: base}, {X: right + u(8), Y: base - u(6)}, {X: right + u(8), Y: base + u(6)}}})

	from := len(sc.Items)
	for i, t := range d.Tasks {
		fill, line := colorTaskFill, colorTaskLine
		if sectionOf[i] >= 0 {
			fill, line = tone(sectionOf[i])
		}
		x, cx := left(i), left(i)+taskW/2
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "journey-drop", X: cx, Y: taskTop + taskH, X2: cx, Y2: base,
			Stroke: colorGrid, StrokeWidth: 1.5, Dash: "4 4"})
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "journey-task", X: x, Y: taskTop, W: taskW, H: taskH, RX: u(4),
			Fill: fill, Stroke: line, StrokeWidth: 1.5})
		block(names[i], cx, taskTop+(taskH-float64(len(names[i]))*lh)/2, "journey-task-label", false)
		for k, a := range t.Actors {
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "journey-actor", X: x + u(12) + float64(k)*u(13), Y: taskTop,
				R: u(5.5), Fill: actorColor(index[a]), Stroke: colorLine, StrokeWidth: 1})
		}
		face(sc, cx, level(t.Score), faceR, t.Score, o.Colors)
	}
	if o.Shadows {
		sc.AddShadows(from, u(3), "journey-task", "journey-face")
	}

	// Legenda uczestników z lewej, na wysokości kroków.
	for i, a := range actors {
		mid := taskTop + lh/2 + float64(i)*(lh+u(4))
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "journey-actor", X: u(7), Y: mid, R: u(7), Fill: actorColor(i), Stroke: colorLine, StrokeWidth: 1})
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "journey-actor-name", X: u(7)*2 + u(8), Y: mid + 0.35*fs, W: width(a), H: lh,
			Text: a, Anchor: "start", Fill: colorText})
	}
	return sc
}

// face rysuje buźkę oceny: uśmiech dla 4 i 5, usta proste dla 3, smutek dla 1 i 2.
func face(sc *scene.Scene, cx, cy, r float64, score int, colors bool) {
	fill := "#ffffff"
	if colors {
		tone := scene.Palette[5] // zielony
		if score < 3 {
			tone = scene.Palette[4] // malinowy
		} else if score == 3 {
			tone = scene.Palette[2] // bursztynowy
		}
		fill = tone.Fill
	}
	sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "journey-face", X: cx, Y: cy, R: r, Fill: fill, Stroke: colorLine, StrokeWidth: 1.5})
	for _, side := range []float64{-1, 1} {
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerLabel, Role: "journey-face-detail", X: cx + side*r*0.35, Y: cy - r*0.25, R: r * 0.1, Fill: colorLine})
	}
	at := func(dx, dy float64) scene.Point { return scene.Point{X: cx + dx*r, Y: cy + dy*r} }
	var mouth []scene.Segment
	switch {
	case score > 3:
		mouth = []scene.Segment{{Cmd: 'M', To: at(-0.45, 0.2)}, {Cmd: 'C', C1: at(-0.2, 0.62), C2: at(0.2, 0.62), To: at(0.45, 0.2)}}
	case score == 3:
		mouth = []scene.Segment{{Cmd: 'M', To: at(-0.4, 0.4)}, {Cmd: 'L', To: at(0.4, 0.4)}}
	default:
		mouth = []scene.Segment{{Cmd: 'M', To: at(-0.45, 0.58)}, {Cmd: 'C', C1: at(-0.2, 0.16), C2: at(0.2, 0.16), To: at(0.45, 0.58)}}
	}
	sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLabel, Role: "journey-face-detail", Segments: mouth, Stroke: colorLine, StrokeWidth: 1.5})
}
