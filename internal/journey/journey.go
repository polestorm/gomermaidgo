// Package journey rozbiera i rysuje podróż użytkownika (journey).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package journey

import (
	"math"
	"strconv"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni podróży użytkownika.
const (
	CodeHeader    = "journey-header"
	CodeStatement = "journey-statement"
	CodeScore     = "journey-score"
	CodeEmpty     = "journey-empty"
	maxTasks      = 100
)

// Task to krok podróży: nazwa, ocena od 1 do 5 i uczestnicy.
type Task struct {
	Name    string
	Score   int
	Actors  []string
	Section string
	Line    int
}

// Diagram to podróż użytkownika.
type Diagram struct {
	Title    string
	Tasks    []*Task
	AccTitle string
	AccDescr string
}

// Parse rozbiera źródło podróży użytkownika.
func Parse(source string) (*Diagram, error) {
	d := &Diagram{}
	var acc syntax.Accessibility
	seenHeader := false
	section := ""
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "journey" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"journey\", found %q", s)
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
			if len(parts) < 2 || len(parts) > 3 || strings.TrimSpace(parts[0]) == "" {
				return nil, syntax.Errorf(line, CodeStatement, "expected a task written as: name: score: actors, found %q", s)
			}
			score, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil || score < 1 || score > 5 {
				return nil, syntax.Errorf(line, CodeScore, "task score must be a whole number from 1 to 5, found %q", strings.TrimSpace(parts[1]))
			}
			if len(d.Tasks) >= maxTasks {
				return nil, syntax.Errorf(line, CodeStatement, "too many tasks (limit %d)", maxTasks)
			}
			t := &Task{Name: strings.TrimSpace(parts[0]), Score: score, Section: section, Line: line}
			if len(parts) == 3 {
				for _, a := range strings.Split(parts[2], ",") {
					if a = strings.TrimSpace(a); a != "" {
						t.Actors = append(t.Actors, a)
					}
				}
			}
			d.Tasks = append(d.Tasks, t)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"journey\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(d.Tasks) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the journey needs at least one task")
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
	Colors     bool // znacznik oceny w kolorze: niska czerwona, średnia bursztynowa, wysoka zielona
	Shadows    bool // cień pod znacznikami ocen
}

const (
	colorText        = "#1b1f24"
	colorMuted       = "#5b6570"
	colorLine        = "#3b4754"
	colorGrid        = "#d5dae1"
	colorSectionFill = "#f1f4f8"
)

// Layout rozmieszcza podróż jako listę kroków z góry na dół: nazwa kroku, ocena na skali od 1 do 5
// i uczestnicy. Linia łącząca oceny kolejnych kroków pokazuje, gdzie podróż idzie gładko,
// a gdzie się załamuje.
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
	text := func(s string, x, mid float64, anchor, role, fill string, bold bool) {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: mid + 0.35*fs, W: width(s), H: lh, Text: s, Anchor: anchor, Bold: bold, Fill: fill})
	}

	names := make([][]string, len(d.Tasks))
	actors := make([]string, len(d.Tasks))
	nameW, actorW := 0.0, 0.0
	for i, t := range d.Tasks {
		names[i] = textwrap.Lines([]string{t.Name}, u(240), width)
		for _, s := range names[i] {
			nameW = math.Max(nameW, width(s))
		}
		actors[i] = strings.Join(t.Actors, ", ")
		actorW = math.Max(actorW, width(actors[i]))
	}
	step := u(34)
	scaleX := math.Round(nameW + u(34))
	at := func(score int) float64 { return scaleX + float64(score-1)*step }
	actorX := at(5) + u(30)
	right := actorX + actorW
	if actorW == 0 {
		right = at(5) + u(14)
	}

	y := 0.0
	if d.Title != "" {
		text(d.Title, 0, lh/2, "start", "title", colorText, false)
		y = lh + u(14)
	}
	for score := 1; score <= 5; score++ {
		text(strconv.Itoa(score), at(score), y+lh/2, "middle", "journey-scale", colorMuted, false)
	}
	y += lh + u(6)
	gridTop := y

	type mark struct {
		x, y  float64
		score int
	}
	var marks []mark
	section := ""
	for i, t := range d.Tasks {
		if t.Section != "" && t.Section != section {
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "journey-section", X: 0, Y: y, W: right, H: lh + 2*u(5), RX: u(4), Fill: colorSectionFill})
			text(t.Section, u(10), y+u(5)+lh/2, "start", "journey-section-title", colorText, true)
			y += lh + 2*u(5) + u(4)
		}
		section = t.Section
		h := float64(len(names[i]))*lh + u(12)
		for k, s := range names[i] {
			text(s, 0, y+u(6)+float64(k)*lh+lh/2, "start", "journey-task", colorText, false)
		}
		if actors[i] != "" {
			text(actors[i], actorX, y+h/2, "start", "journey-actors", colorMuted, false)
		}
		marks = append(marks, mark{at(t.Score), y + h/2, t.Score})
		y += h
	}
	for score := 1; score <= 5; score++ {
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerBackground, Role: "journey-grid", X: at(score), Y: gridTop, X2: at(score), Y2: y, Stroke: colorGrid, StrokeWidth: 1})
	}
	if len(marks) > 1 {
		pts := make([]scene.Point, len(marks))
		for i, m := range marks {
			pts[i] = scene.Point{X: m.x, Y: m.y}
		}
		sc.Add(scene.Item{Kind: scene.Polyline, Layer: scene.LayerLifeline, Role: "journey-line", Points: pts, Stroke: colorLine, StrokeWidth: 1.5})
	}
	from := len(sc.Items)
	for _, m := range marks {
		// Ocena jest czytelna bez koloru: położenie na skali, a znacznik niskiej oceny jest pełny.
		fill := "#ffffff"
		if m.score <= 2 {
			fill = colorLine
		} else if m.score == 3 {
			fill = "#9aa5b1"
		}
		stroke := colorLine
		if o.Colors {
			tone := scene.Palette[5] // zielony
			if m.score <= 2 {
				tone = scene.Palette[4] // malinowy
			} else if m.score == 3 {
				tone = scene.Palette[2] // bursztynowy
			}
			fill, stroke = tone.Stroke, tone.Stroke
		}
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "journey-mark", X: m.x, Y: m.y, R: u(7), Fill: fill, Stroke: stroke, StrokeWidth: 1.5})
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "journey-mark")
	}
	return sc
}
