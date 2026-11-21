package gantt

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // sekcje w kolejnych kolorach palety
	Shadows    bool // cień pod zadaniami
}

// Kolory rysunku; stan zadania jest czytelny także w druku czarno-białym (jasność i obrys).
const (
	colorText       = "#1b1f24"
	colorMuted      = "#5b6570"
	colorGrid       = "#d5dae1"
	colorAxis       = "#6b7785"
	colorBand       = "#f6f8fa"
	colorExcluded   = "#eceff3"
	colorTaskFill   = "#eef2f7"
	colorTaskStroke = "#55657a"
	colorActiveFill = "#c9d6e8"
	colorDoneFill   = "#e6e8eb"
	colorDoneStroke = "#9aa5b1"
	colorCritFill   = "#fde8e8"
	colorCritStroke = "#b42318"
	colorMilestone  = "#3b4754"
)

var (
	monthNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	dayNames   = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
)

// strftime zapisuje chwilę według wzorca z axisFormat; nieznane znaczniki zostają dosłownie.
// Nazwy miesięcy i dni są angielskie, jak w Mermaid.
func strftime(format string, t time.Time) string {
	var b strings.Builder
	pad := func(v, width int) {
		s := strconv.Itoa(v)
		for len(s) < width {
			s = "0" + s
		}
		b.WriteString(s)
	}
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			b.WriteByte(format[i])
			continue
		}
		i++
		switch format[i] {
		case 'Y':
			pad(t.Year(), 4)
		case 'y':
			pad(t.Year()%100, 2)
		case 'm':
			pad(int(t.Month()), 2)
		case 'd':
			pad(t.Day(), 2)
		case 'e':
			b.WriteString(strconv.Itoa(t.Day()))
		case 'H':
			pad(t.Hour(), 2)
		case 'M':
			pad(t.Minute(), 2)
		case 'S':
			pad(t.Second(), 2)
		case 'j':
			pad(t.YearDay(), 3)
		case 'b':
			b.WriteString(monthNames[t.Month()-1][:3])
		case 'B':
			b.WriteString(monthNames[t.Month()-1])
		case 'a':
			b.WriteString(dayNames[t.Weekday()][:3])
		case 'A':
			b.WriteString(dayNames[t.Weekday()])
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
	}
	return b.String()
}

func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ticks zwraca chwile podziałki w odstępie n jednostek, wyrównane do kalendarza.
func ticks(lo, hi time.Time, n int, unit string) []time.Time {
	var t time.Time
	step := func(t time.Time) time.Time { return t }
	switch unit {
	case "hour":
		t = time.Date(lo.Year(), lo.Month(), lo.Day(), lo.Hour()/n*n, 0, 0, 0, time.UTC)
		step = func(t time.Time) time.Time { return t.Add(time.Duration(n) * time.Hour) }
	case "day":
		t = day(lo)
		step = func(t time.Time) time.Time { return t.AddDate(0, 0, n) }
	case "week":
		t = day(lo)
		for t.Weekday() != time.Monday {
			t = t.AddDate(0, 0, -1)
		}
		step = func(t time.Time) time.Time { return t.AddDate(0, 0, 7*n) }
	case "month":
		t = time.Date(lo.Year(), time.Month((int(lo.Month())-1)/n*n+1), 1, 0, 0, 0, 0, time.UTC)
		step = func(t time.Time) time.Time { return t.AddDate(0, n, 0) }
	default:
		t = time.Date(lo.Year()/n*n, 1, 1, 0, 0, 0, 0, time.UTC)
		step = func(t time.Time) time.Time { return t.AddDate(n, 0, 0) }
	}
	var out []time.Time
	for ; !t.After(hi) && len(out) < 2000; t = step(t) {
		if !t.Before(lo) {
			out = append(out, t)
		}
	}
	return out
}

// Layout rozmieszcza wykres i zwraca scenę.
func Layout(d *Diagram, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return math.Round(v * fs / 16) }
	width := func(s string) float64 { return o.Measure(s, fs) }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.8)
	title := d.AccTitle
	if title == "" {
		title = d.Title
	}
	sc := &scene.Scene{Title: title, Desc: d.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}
	baseline := func(top float64) float64 { return top + lh/2 + 0.35*fs }
	text := func(s string, x, top float64, anchor, role, fill string) {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: baseline(top), W: width(s), H: lh, Text: s, Anchor: anchor, Fill: fill})
	}

	// Oś czasu: od początku pierwszego dnia do końca ostatniego (dla wykresów godzinowych: do pełnej godziny).
	lo, hi := d.Tasks[0].Start, d.Tasks[0].End
	for _, t := range d.Tasks {
		if t.Start.Before(lo) {
			lo = t.Start
		}
		if t.End.After(hi) {
			hi = t.End
		}
	}
	hourly := hi.Sub(lo) <= 72*time.Hour && (lo != day(lo) || hi != day(hi))
	if hourly {
		lo, hi = lo.Truncate(time.Hour), hi.Add(time.Hour-1).Truncate(time.Hour)
	} else {
		lo = day(lo)
		if hi != day(hi) {
			hi = day(hi).AddDate(0, 0, 1)
		}
	}
	if !hi.After(lo) {
		hi = lo.AddDate(0, 0, 1)
	}

	// Kolumna nazw sekcji.
	sectionW := 0.0
	for _, s := range d.Sections {
		sectionW = math.Max(sectionW, width(s)+2*u(12))
	}
	chartX, chartW := sectionW, u(640)
	scale := chartW / hi.Sub(lo).Seconds()
	xOf := func(t time.Time) float64 { return chartX + t.Sub(lo).Seconds()*scale }

	// Podziałka: zadana albo pierwsza, której opisy się nie stykają.
	type cand struct {
		n    int
		unit string
	}
	candidates := []cand{{1, "day"}, {2, "day"}, {1, "week"}, {2, "week"}, {1, "month"}, {3, "month"}, {6, "month"}, {1, "year"}, {2, "year"}, {5, "year"}, {10, "year"}, {25, "year"}}
	if hourly {
		candidates = append([]cand{{1, "hour"}, {2, "hour"}, {6, "hour"}, {12, "hour"}}, candidates...)
	}
	if d.TickEvery > 0 {
		candidates = append([]cand{{d.TickEvery, d.TickUnit}}, candidates...)
	}
	var marks []time.Time
	for _, c := range candidates {
		marks = ticks(lo, hi, c.n, c.unit)
		fits := len(marks) <= 400
		for i := 0; fits && i+1 < len(marks); i++ {
			need := (o.Measure(strftime(d.AxisFormat, marks[i]), small)+o.Measure(strftime(d.AxisFormat, marks[i+1]), small))/2 + u(12)
			fits = xOf(marks[i+1])-xOf(marks[i]) >= need
		}
		if fits {
			break
		}
	}

	top := 0.0
	if d.Title != "" {
		text(d.Title, chartX+chartW/2, top, "middle", "title", colorText)
		top += lh + u(10)
	}
	axisY := top + small*1.25 + u(6)
	rowH, barH := u(34), u(22)
	bottom := axisY + float64(len(d.Tasks))*rowH

	// Pasy sekcji i ich nazwy.
	for i := 0; i < len(d.Tasks); {
		j := i
		for j < len(d.Tasks) && d.Tasks[j].Section == d.Tasks[i].Section {
			j++
		}
		y0, y1 := axisY+float64(i)*rowH, axisY+float64(j)*rowH
		if s := d.Tasks[i].Section; s >= 0 {
			if s%2 == 0 {
				sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "gantt-section",
					X: 0, Y: y0, W: chartX + chartW, H: y1 - y0, Fill: colorBand})
			}
			text(d.Sections[s], u(12), (y0+y1)/2-lh/2, "start", "gantt-section-label", colorText)
		}
		i = j
	}

	// Dni wyłączone: szare pasy, jeśli dzień jest dość szeroki, żeby je było widać.
	if d.HasExcludes && scale*86400 >= 3 {
		for t := day(lo); t.Before(hi); t = t.AddDate(0, 0, 1) {
			if !d.Excluded(t) {
				continue
			}
			end := t.AddDate(0, 0, 1)
			for end.Before(hi) && d.Excluded(end) {
				end = end.AddDate(0, 0, 1)
			}
			x0, x1 := math.Max(xOf(t), chartX), math.Min(xOf(end), chartX+chartW)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLifeline, Role: "gantt-excluded", X: x0, Y: axisY, W: x1 - x0, H: bottom - axisY, Fill: colorExcluded})
			t = end.AddDate(0, 0, -1)
		}
	}

	// Siatka i opisy podziałki.
	for _, m := range marks {
		x := xOf(m)
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabelBackground, Role: "gantt-grid", X: x, Y: axisY, X2: x, Y2: bottom, Stroke: colorGrid, StrokeWidth: 1})
		label := strftime(d.AxisFormat, m)
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "gantt-tick-label", X: x, Y: axisY - u(6), W: o.Measure(label, small), H: small * 1.25,
			Text: label, Anchor: "middle", FontSize: small, Fill: colorMuted})
	}
	sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerFrame, Role: "gantt-frame", X: chartX, Y: axisY, W: chartW, H: bottom - axisY, Stroke: colorAxis, StrokeWidth: 1})

	// Zadania.
	for i, t := range d.Tasks {
		cy := axisY + float64(i)*rowH + rowH/2
		x0, x1 := xOf(t.Start), xOf(t.End)
		fill, stroke, strokeW, ink := colorTaskFill, colorTaskStroke, 1.5, colorText
		if o.Colors && t.Section >= 0 {
			tone := scene.Palette[t.Section%len(scene.Palette)]
			fill, stroke = tone.Fill, tone.Stroke
		}
		switch {
		case t.Done:
			fill, stroke, ink = colorDoneFill, colorDoneStroke, colorMuted
		case t.Active:
			fill = colorActiveFill
		}
		if t.Critical {
			stroke, strokeW = colorCritStroke, 2
			if !t.Done && !t.Active {
				fill = colorCritFill
			}
		}
		labelW := width(t.Name)
		right := x1
		if t.Milestone {
			mid, r := (x0+x1)/2, u(10)
			if !t.Critical {
				fill, stroke = colorMilestone, colorMilestone
			}
			sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "gantt-milestone",
				Points: []scene.Point{{X: mid, Y: cy - r}, {X: mid + r, Y: cy}, {X: mid, Y: cy + r}, {X: mid - r, Y: cy}}, Fill: fill, Stroke: stroke, StrokeWidth: strokeW})
			x0, right = mid-r, mid+r
		} else {
			w := math.Max(x1-x0, u(3)) // zadanie krótsze niż piksel nadal jest widoczne
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "gantt-task", X: x0, Y: cy - barH/2, W: w, H: barH, RX: u(3),
				Fill: fill, Stroke: stroke, StrokeWidth: strokeW})
			right = x0 + w
			if labelW+2*u(8) <= w {
				text(t.Name, x0+w/2, cy-lh/2, "middle", "gantt-task-label", ink)
				continue
			}
		}
		// Nazwa nie mieści się w pasku: stoi za nim, a gdy tam wyszłaby poza wykres, przed nim.
		if right+u(6)+labelW > chartX+chartW && x0-u(6)-labelW >= chartX {
			text(t.Name, x0-u(6), cy-lh/2, "end", "gantt-task-label", ink)
		} else {
			text(t.Name, right+u(6), cy-lh/2, "start", "gantt-task-label", ink)
		}
	}
	if o.Shadows {
		sc.AddShadows(0, u(3), "gantt-task", "gantt-milestone")
	}
	return sc
}
