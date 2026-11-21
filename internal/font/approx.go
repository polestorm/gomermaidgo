package font

import "unicode"

// ApproxAdvance to miara zastępcza, gdy wywołujący nie podał fontu: klasy szerokości
// dobrane tak, żeby raczej zawyżać niż zaniżać (etykieta ma się zmieścić w kształcie).
// Nie pochodzi z żadnego konkretnego kroju.
func ApproxAdvance(c rune) float64 {
	switch c {
	case 'i', 'j', 'l', 'I', '.', ',', ':', ';', '!', '|', '\'', ' ', 'ı', 'ł':
		return 0.30
	case 'f', 't', 'r', '(', ')', '[', ']', '{', '}', '-', '/', '\\', '"', '„', '”', '“', '*':
		return 0.38
	case 'm', 'w', 'M', 'W', '@', '%', '—':
		return 0.90
	}
	switch {
	case c >= '0' && c <= '9':
		return 0.60
	case unicode.IsUpper(c):
		return 0.72
	case unicode.IsLower(c):
		return 0.58
	case c > 0x2E80: // pisma o znakach pełnej szerokości
		return 1.0
	}
	return 0.62
}

// ApproxTextWidth zwraca szerokość tekstu w pikselach według miary zastępczej.
func ApproxTextWidth(text string, size float64) float64 {
	w := 0.0
	for _, c := range text {
		w += ApproxAdvance(c) * size
	}
	return w
}
