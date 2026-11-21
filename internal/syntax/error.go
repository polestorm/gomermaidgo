// Package syntax definiuje błąd składni wspólny dla wszystkich typów diagramów.
package syntax

import "fmt"

// Error to błąd w źródle diagramu. Code jest stałym identyfikatorem do tłumaczenia
// komunikatu przez wywołującego; Message to opis po angielsku.
type Error struct {
	Line    int // numer linii od 1; 0, gdy błąd dotyczy całego źródła
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

// Errorf tworzy błąd z numerem linii i kodem.
func Errorf(line int, code, format string, args ...any) *Error {
	return &Error{Line: line, Code: code, Message: fmt.Sprintf(format, args...)}
}
