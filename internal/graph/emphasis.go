package graph

import "strings"

// Znaczniki wyróżnienia w tekście etykiety: każdy przełącza swój styl. Są to znaki z obszaru
// prywatnego Unicode; źródło diagramu nie może ich zawierać, więc pochodzą tylko z parsera.
const (
	MarkBold   = ''
	MarkItalic = ''
)

// Run to fragment tekstu w jednym stylu.
type Run struct {
	Text         string
	Bold, Italic bool
}

// HasMarks mówi, czy tekst zawiera znaczniki wyróżnienia.
func HasMarks(s string) bool { return strings.ContainsAny(s, string([]rune{MarkBold, MarkItalic})) }

// Plain zwraca tekst bez znaczników wyróżnienia.
func Plain(s string) string {
	if !HasMarks(s) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if r == MarkBold || r == MarkItalic {
			return -1
		}
		return r
	}, s)
}

// Runs dzieli tekst na fragmenty o stałym stylu.
func Runs(s string) []Run {
	var out []Run
	var cur Run
	flush := func() {
		if cur.Text != "" {
			out = append(out, cur)
		}
		cur.Text = ""
	}
	for _, r := range s {
		switch r {
		case MarkBold:
			flush()
			cur.Bold = !cur.Bold
		case MarkItalic:
			flush()
			cur.Italic = !cur.Italic
		default:
			cur.Text += string(r)
		}
	}
	flush()
	return out
}

// mark otacza tekst znacznikami stylu.
func mark(text string, bold, italic bool) string {
	if bold {
		text = string(MarkBold) + text + string(MarkBold)
	}
	if italic {
		text = string(MarkItalic) + text + string(MarkItalic)
	}
	return text
}

// PerWord przepisuje tekst tak, żeby każdy wyraz niósł własne znaczniki; po takim zabiegu
// linię można łamać między wyrazami bez gubienia stylu.
func PerWord(s string) string {
	if !HasMarks(s) {
		return s
	}
	var b strings.Builder
	space := false
	for _, run := range Runs(s) {
		word := ""
		emit := func() {
			if word != "" {
				b.WriteString(mark(word, run.Bold, run.Italic))
				word = ""
			}
		}
		for _, r := range run.Text {
			if r == ' ' || r == '\t' {
				emit()
				space = true
				continue
			}
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			word += string(r)
		}
		emit()
	}
	return b.String()
}

// Segments zwraca fragmenty do narysowania jeden za drugim: sąsiednie fragmenty w tym samym
// stylu są złączone, a odstęp należy do fragmentu, który go poprzedza (odstęp na początku
// elementu tekstowego SVG znika, na końcu nie szkodzi).
func Segments(s string) []Run {
	var out []Run
	for _, run := range Runs(s) {
		if len(out) > 0 && strings.TrimSpace(run.Text) == "" {
			out[len(out)-1].Text += run.Text
			continue
		}
		if lead := len(run.Text) - len(strings.TrimLeft(run.Text, " \t")); lead > 0 && len(out) > 0 {
			out[len(out)-1].Text += run.Text[:lead]
			run.Text = run.Text[lead:]
		}
		if n := len(out); n > 0 && out[n-1].Bold == run.Bold && out[n-1].Italic == run.Italic {
			out[n-1].Text += run.Text
			continue
		}
		out = append(out, run)
	}
	return out
}

// showMarks zapisuje znaczniki tak, jak wyglądają w źródle Markdown; służy plikom wzorcowym.
func showMarks(s string) string {
	return strings.NewReplacer(string(MarkBold), "**", string(MarkItalic), "*").Replace(s)
}
