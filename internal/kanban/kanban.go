// Package kanban rozbiera i rysuje tablicę kanban.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package kanban

import (
	"math"
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
	"github.com/polestorm/gomermaidgo/internal/textwrap"
)

// Kody błędów składni tablicy kanban.
const (
	CodeHeader    = "kanban-header"
	CodeStatement = "kanban-statement"
	CodeEmpty     = "kanban-empty"
	maxColumns    = 12
	maxCards      = 200
)

// Card to karta zadania: tekst i opcjonalne dane z bloku @{ … }.
type Card struct {
	Text     []string
	Ticket   string
	Assigned string
	Priority string
	Line     int
}

// Column to kolumna tablicy.
type Column struct {
	Title []string
	Cards []Card
	Line  int
}

// Board to tablica kanban.
type Board struct {
	Columns  []*Column
	AccTitle string
	AccDescr string
}

var (
	item  = regexp.MustCompile(`^[^\[\]@]*\[(.*)\]$`)
	meta  = regexp.MustCompile(`^(.*?)\s*@\{(.*)\}$`)
	field = regexp.MustCompile(`(\w+)\s*:\s*(?:'([^']*)'|"([^"]*)"|([^,]*))`)
)

// Parse rozbiera źródło tablicy. Wcięcie wyznacza poziom: najpłytsze linie są kolumnami,
// głębsze kartami ostatniej kolumny.
func Parse(source string) (*Board, error) {
	b := &Board{}
	var acc syntax.Accessibility
	seenHeader := false
	columnIndent, cards := -1, 0
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			if s != "kanban" {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"kanban\", found %q", s)
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		indent := 0
		for _, c := range raw {
			if c == ' ' {
				indent++
			} else if c == '\t' {
				indent += 4
			} else {
				break
			}
		}
		var card Card
		body := s
		if m := meta.FindStringSubmatch(s); m != nil {
			body = m[1]
			for _, f := range field.FindAllStringSubmatch(m[2], -1) {
				value := strings.TrimSpace(f[2] + f[3] + f[4])
				switch f[1] {
				case "ticket":
					card.Ticket = value
				case "assigned":
					card.Assigned = value
				case "priority":
					card.Priority = value
				}
			}
		}
		if m := item.FindStringSubmatch(body); m != nil {
			body = m[1]
		}
		text := textwrap.Split(body)
		if len(text) == 0 {
			return nil, syntax.Errorf(line, CodeStatement, "column or card has no text: %q", s)
		}
		if columnIndent < 0 || indent <= columnIndent {
			if len(b.Columns) >= maxColumns {
				return nil, syntax.Errorf(line, CodeStatement, "too many columns (limit %d)", maxColumns)
			}
			columnIndent = indent
			b.Columns = append(b.Columns, &Column{Title: text, Line: line})
			continue
		}
		if cards++; cards > maxCards {
			return nil, syntax.Errorf(line, CodeStatement, "too many cards (limit %d)", maxCards)
		}
		card.Text, card.Line = text, line
		col := b.Columns[len(b.Columns)-1]
		col.Cards = append(col.Cards, card)
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"kanban\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(b.Columns) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the board needs at least one column")
	}
	b.AccTitle, b.AccDescr = acc.Title, acc.Descr
	return b, nil
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // kolumny w kolejnych tonach palety
	Shadows    bool // cień pod kartami
}

const (
	colorText   = "#1b1f24"
	colorMuted  = "#5b6570"
	colorColumn = "#eef2f7"
	colorLine   = "#9aa5b1"
	colorCard   = "#ffffff"
	colorStrong = "#3b4754"
)

// Layout rozmieszcza tablicę: kolumny obok siebie, w każdej tytuł i karty jedna pod drugą.
// Karta ma tekst, pod nim numer zgłoszenia, osobę i priorytet; pilność widać po grubości
// lewej krawędzi, więc czyta się ją także bez koloru.
func Layout(b *Board, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.8)
	width := func(s string) float64 { return o.Measure(s, fs) }
	sc := &scene.Scene{Title: b.AccTitle, Desc: b.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	colW, gap, pad := u(200), u(12), u(10)
	inner := colW - 2*pad
	from := len(sc.Items)
	bottom := 0.0
	type placed struct {
		x, top, h float64
		tone      int
	}
	var frames []placed
	for i, col := range b.Columns {
		x := float64(i) * (colW + gap)
		title := textwrap.Lines(col.Title, inner, width)
		y := pad
		for k, s := range title {
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "kanban-column-title", X: x + colW/2, Y: y + float64(k)*lh + lh/2 + 0.35*fs,
				W: width(s), H: lh, Text: s, Anchor: "middle", Bold: true, Fill: colorText})
		}
		y += float64(len(title))*lh + pad
		for _, c := range col.Cards {
			lines := textwrap.Lines(c.Text, inner-2*u(8), width)
			var details []string
			for _, d := range []string{c.Ticket, c.Assigned, c.Priority} {
				if d != "" {
					details = append(details, d)
				}
			}
			h := float64(len(lines))*lh + 2*u(8)
			if len(details) > 0 {
				h += small*1.3 + u(2)
			}
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "kanban-card", X: x + pad, Y: y, W: inner, H: h, RX: u(4),
				Fill: colorCard, Stroke: colorLine, StrokeWidth: 1})
			// Priorytet: pasek przy lewej krawędzi karty, tym grubszy, im pilniej.
			if bar := map[string]float64{"very high": 6, "high": 4, "low": 2, "very low": 1}[strings.ToLower(c.Priority)]; bar > 0 {
				sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "kanban-priority", X: x + pad, Y: y, W: u(bar), H: h, Fill: colorStrong})
			}
			for k, s := range lines {
				sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "kanban-card-text", X: x + pad + u(10), Y: y + u(8) + float64(k)*lh + lh/2 + 0.35*fs,
					W: width(s), H: lh, Text: s, Anchor: "start", Fill: colorText})
			}
			if len(details) > 0 {
				s := strings.Join(details, " · ")
				sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "kanban-card-details", X: x + pad + u(10), Y: y + h - u(8) - 0.2*small,
					W: o.Measure(s, small), H: small * 1.25, Text: s, Anchor: "start", FontSize: small, Fill: colorMuted})
			}
			y += h + u(8)
		}
		frames = append(frames, placed{x, 0, y - u(8) + pad, i})
		bottom = math.Max(bottom, y-u(8)+pad)
	}
	if o.Shadows {
		sc.AddShadows(from, u(2), "kanban-card")
	}
	// Tła kolumn sięgają do dołu najdłuższej.
	for _, f := range frames {
		fill, line := colorColumn, colorLine
		if o.Colors {
			t := scene.Palette[f.tone%len(scene.Palette)]
			fill, line = t.Fill, t.Stroke
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "kanban-column", X: f.x, Y: 0, W: colW, H: bottom, RX: u(6),
			Fill: fill, Stroke: line, StrokeWidth: 1})
	}
	return sc
}
