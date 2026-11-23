// Package bridge udostępnia pakietom wyjściowym (np. pdf) wewnętrzną drogę od źródła diagramu do
// sceny, bez wystawiania sceny w publicznym API. Funkcję ustawia pakiet główny przy starcie.
package bridge

import "github.com/polestorm/gomermaidgo/internal/scene"

// Fonts to pliki krojów, którymi rysunek został zmierzony: zwykły, pogrubiony i pochylony.
// Odmiana niepodana przez wywołującego jest równa krojowi zwykłemu.
type Fonts struct {
	Regular, Bold, Italic []byte
	Fallback              []byte // krój zapasowy dla znaków spoza powyższych; może być nil
}

// Build buduje scenę ze źródła. Opcje są wartościami typu mermaid.Option.
var Build func(source string, options []any) (*scene.Scene, Fonts, error)
