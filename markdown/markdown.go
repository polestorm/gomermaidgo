// Package markdown wstawia diagramy do dokumentu Markdown: każdy blok kodu ogrodzony, którego
// opis zaczyna się od słowa „mermaid”, zamienia na rysunek SVG osadzony jako blok HTML.
// Działa na tekście, przed parserem Markdown, więc nie zależy od żadnego parsera; parser musi
// tylko przepuszczać bloki HTML (w goldmark: opcja WithUnsafe).
package markdown

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/polestorm/gomermaidgo"
)

// CSS to reguły dla strony i wydruku: rysunek mieści się na szerokość i na wysokość strony
// (rysunku nie da się podzielić między strony, więc wyższy zostałby ucięty) i nie odrywa się
// od podpisu. Wysokość 230 mm odpowiada stronie A4 z marginesami; inny format wymaga innej.
const CSS = `figure.diagram{margin:1em 0;break-inside:avoid;page-break-inside:avoid}
figure.diagram svg{display:block;margin:0 auto;max-width:100%;max-height:230mm;width:auto;height:auto}
pre.diagram-error{border:1px solid #b42318;padding:.6em .8em;white-space:pre-wrap}
pre.diagram-error strong{display:block;color:#b42318;margin-bottom:.6em}
`

// Problem opisuje blok, którego nie udało się narysować.
type Problem struct {
	Line int   // linia dokumentu Markdown, w której leży błąd (albo początek bloku)
	Err  error // błąd z gomermaidgo.Render
}

func (p Problem) Error() string { return fmt.Sprintf("line %d: %v", p.Line, p.Err) }

// Options to ustawienia zamiany.
type Options struct {
	Render      []gomermaidgo.Option // opcje rysowania wspólne dla wszystkich bloków
	ErrorPrefix string               // początek komunikatu w miejscu błędnego bloku; domyślnie "Diagram error"
}

// fence rozpoznaje linię otwierającą blok ogrodzony: znak ogrodzenia, jego długość i opis.
func fence(line string) (char byte, length int, info string, ok bool) {
	s := strings.TrimLeft(line, " ")
	if len(line)-len(s) > 3 || len(s) < 3 || (s[0] != '`' && s[0] != '~') {
		return 0, 0, "", false
	}
	char = s[0]
	for length < len(s) && s[length] == char {
		length++
	}
	info = strings.TrimSpace(s[length:])
	if length < 3 || (char == '`' && strings.Contains(info, "`")) {
		return 0, 0, "", false
	}
	return char, length, info, true
}

// closes mówi, czy linia zamyka blok otwarty ogrodzeniem ze znaku char o długości length.
func closes(line string, char byte, length int) bool {
	c, n, info, ok := fence(line)
	return ok && c == char && n >= length && info == ""
}

// Embed zamienia bloki „mermaid” na rysunki. Blok z błędem zostaje w dokumencie jako źródło
// z komunikatem, a błąd trafia do wyniku z numerem linii dokumentu. Bloki zagnieżdżone
// w cytatach i listach (wcięte albo poprzedzone „>”) oraz wewnątrz innych bloków kodu
// pozostają nietknięte.
func Embed(source string, o Options) (string, []Problem) {
	prefix := o.ErrorPrefix
	if prefix == "" {
		prefix = "Diagram error"
	}
	lines := strings.Split(source, "\n")
	var out []string
	var problems []Problem
	for i := 0; i < len(lines); i++ {
		char, length, info, ok := fence(strings.TrimRight(lines[i], "\r"))
		if !ok {
			out = append(out, lines[i])
			continue
		}
		end := i + 1
		for end < len(lines) && !closes(strings.TrimRight(lines[end], "\r"), char, length) {
			end++
		}
		word, _, _ := strings.Cut(info, " ")
		if word != "mermaid" || end >= len(lines) {
			// Inny blok kodu albo blok niezamknięty: przepisujemy bez zmian, razem z zawartością.
			last := min(end, len(lines)-1)
			out = append(out, lines[i:last+1]...)
			i = last
			continue
		}
		body := make([]string, 0, end-i-1)
		for _, l := range lines[i+1 : end] {
			body = append(body, strings.TrimRight(l, "\r"))
		}
		diagram := strings.Join(body, "\n")
		svg, err := gomermaidgo.Render(diagram, o.Render...)
		if err != nil {
			at, message := i+1, err.Error()
			var se *gomermaidgo.SyntaxError
			if errors.As(err, &se) {
				message = se.Message // numer linii podajemy sami, w numeracji dokumentu
				if se.Line > 0 {
					at += se.Line
				}
			}
			problems = append(problems, Problem{Line: at, Err: err})
			// Blok „pre” kończy się dopiero na swoim znaczniku zamykającym, więc puste linie
			// źródła go nie przerywają.
			out = append(out, "", fmt.Sprintf(`<pre class="diagram-error"><strong>%s (line %d): %s</strong>%s</pre>`,
				html.EscapeString(prefix), at, html.EscapeString(message), html.EscapeString(diagram)), "")
		} else {
			out = append(out, "", `<figure class="diagram">`, strings.TrimRight(string(svg), "\n"), `</figure>`, "")
		}
		i = end
	}
	return strings.Join(out, "\n"), problems
}
