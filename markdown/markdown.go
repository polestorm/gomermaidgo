// Package markdown inserts diagrams into a Markdown document: every fenced code block whose
// info string starts with the word "mermaid" is replaced with an SVG drawing embedded as an HTML block.
// It works on the text, before the Markdown parser, so it does not depend on any parser; the parser
// only has to pass HTML blocks through (in goldmark: the WithUnsafe option).
package markdown

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/polestorm/gomermaidgo/mermaid"
)

// CSS holds the rules for the page and print: a drawing fits the width and the height of the page
// (a drawing cannot be split between pages, so a taller one would be cut off) and is not separated
// from its caption. The height of 230 mm suits an A4 page with margins; another format needs another.
const CSS = `figure.diagram{margin:1em 0;break-inside:avoid;page-break-inside:avoid}
figure.diagram svg{display:block;margin:0 auto;max-width:100%;max-height:230mm;width:auto;height:auto}
pre.diagram-error{border:1px solid #b42318;padding:.6em .8em;white-space:pre-wrap}
pre.diagram-error strong{display:block;color:#b42318;margin-bottom:.6em}
`

// Problem describes a block that could not be rendered.
type Problem struct {
	Line int   // the line of the Markdown document where the error lies (or where the block starts)
	Err  error // the error from mermaid.Render
}

func (p Problem) Error() string { return fmt.Sprintf("line %d: %v", p.Line, p.Err) }

// Options holds the settings of the replacement.
type Options struct {
	Render      []mermaid.Option // rendering options shared by all blocks
	ErrorPrefix string           // start of the message shown in place of a faulty block; default "Diagram error"
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

// Embed replaces "mermaid" blocks with drawings. A block with an error stays in the document as
// source with a message, and the error goes to the result with the document line number. Blocks
// nested in block quotes and lists (indented or preceded by ">") and inside other code blocks
// are left untouched.
func Embed(source string, o Options) (string, []Problem) {
	prefix := o.ErrorPrefix
	if prefix == "" {
		prefix = "Diagram error"
	}
	return Replace(source, func(b Block) ([]string, error) {
		svg, err := mermaid.Render(b.Source, o.Render...)
		if err != nil {
			at, message := b.Line, err.Error()
			var se *mermaid.SyntaxError
			if errors.As(err, &se) {
				message = se.Message // numer linii podajemy sami, w numeracji dokumentu
				if se.Line > 0 {
					at += se.Line
				}
			}
			// Blok „pre” kończy się dopiero na swoim znaczniku zamykającym, więc puste linie
			// źródła go nie przerywają.
			return []string{"", fmt.Sprintf(`<pre class="diagram-error"><strong>%s (line %d): %s</strong>%s</pre>`,
				html.EscapeString(prefix), at, html.EscapeString(message), html.EscapeString(b.Source)), ""}, err
		}
		return []string{"", `<figure class="diagram">`, strings.TrimRight(string(svg), "\n"), `</figure>`, ""}, nil
	})
}

// Block is one "mermaid" block found in the document.
type Block struct {
	Index  int    // the number of the block in the document, from 1
	Line   int    // the document line with the opening fence
	Source string // the diagram source
}

// Replace replaces every "mermaid" block with the lines returned by the function. An error from the
// function goes to the result with the document line number (for a syntax error: the line where it lies);
// lines returned together with an error replace the block anyway, and when the function returns none,
// the block stays unchanged. The rules for recognizing blocks are the same as in Embed.
func Replace(source string, replace func(Block) ([]string, error)) (string, []Problem) {
	lines := strings.Split(source, "\n")
	var out []string
	var problems []Problem
	index := 0
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
		index++
		with, err := replace(Block{Index: index, Line: i + 1, Source: strings.Join(body, "\n")})
		if err != nil {
			at := i + 1
			var se *mermaid.SyntaxError
			if errors.As(err, &se) && se.Line > 0 {
				at += se.Line
			}
			problems = append(problems, Problem{Line: at, Err: err})
		}
		if with == nil {
			with = lines[i : end+1]
		}
		out = append(out, with...)
		i = end
	}
	return strings.Join(out, "\n"), problems
}
