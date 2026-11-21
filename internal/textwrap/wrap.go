// Package textwrap łamie tekst na linie o ograniczonej szerokości. Służy typom diagramów
// rysowanym bez układu grafowego (oś czasu, podróż użytkownika).
package textwrap

import (
	"regexp"
	"strings"
)

var lineBreak = regexp.MustCompile(`(?i)<br\s*/?>`)

// Split dzieli tekst na linie w miejscach znacznika <br>.
func Split(text string) []string {
	var out []string
	for _, part := range lineBreak.Split(text, -1) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Lines łamie każdą linię na granicach wyrazów tak, żeby nie przekraczała limitu szerokości.
// Wyraz szerszy niż limit zostaje w całości.
func Lines(lines []string, limit float64, width func(string) float64) []string {
	var out []string
	for _, line := range lines {
		if width(line) <= limit {
			out = append(out, line)
			continue
		}
		current := ""
		for _, word := range strings.Fields(line) {
			candidate := word
			if current != "" {
				candidate = current + " " + word
			}
			if current != "" && width(candidate) > limit {
				out = append(out, current)
				current = word
			} else {
				current = candidate
			}
		}
		if current != "" {
			out = append(out, current)
		}
	}
	return out
}
