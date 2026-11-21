package syntax

import "strings"

// Accessibility zbiera tytuł i opis dostępności (accTitle, accDescr), wspólne dla wszystkich
// typów diagramów. Trafiają do elementów <title> i <desc> dokumentu SVG.
type Accessibility struct {
	Title, Descr string
	open         bool // trwa wieloliniowy opis „accDescr { … }”
}

// Line przyjmuje jedną linię źródła (bez odstępów na końcach) i mówi, czy ją zużyła.
func (a *Accessibility) Line(s string) bool {
	if a.open {
		if end := strings.Index(s, "}"); end >= 0 {
			a.append(s[:end])
			a.open = false
		} else {
			a.append(s)
		}
		return true
	}
	switch {
	case strings.HasPrefix(s, "accTitle"):
		rest := strings.TrimSpace(s[len("accTitle"):])
		if !strings.HasPrefix(rest, ":") {
			return false
		}
		a.Title = strings.TrimSpace(rest[1:])
		return true
	case strings.HasPrefix(s, "accDescr"):
		rest := strings.TrimSpace(s[len("accDescr"):])
		switch {
		case strings.HasPrefix(rest, ":"):
			a.Descr = strings.TrimSpace(rest[1:])
		case strings.HasPrefix(rest, "{"):
			a.Descr = ""
			a.open = true
			return a.Line(rest[1:])
		default:
			return false
		}
		return true
	}
	return false
}

// Open mówi, czy opis wieloliniowy nie został zamknięty klamrą.
func (a *Accessibility) Open() bool { return a.open }

func (a *Accessibility) append(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	if a.Descr != "" {
		a.Descr += " "
	}
	a.Descr += s
}
