// Package sequence obsługuje diagram sekwencji: tekst → model → scena.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package sequence

// Diagram to rozebrany diagram sekwencji.
type Diagram struct {
	Title        string
	Autonumber   bool
	Participants []*Participant
	Events       []Event
}

// Participant to uczestnik: prostokąt albo aktor (ludzik).
type Participant struct {
	ID    string
	Label []string // linie etykiety
	Actor bool
	Line  int
}

// EventKind to rodzaj zdarzenia na osi czasu.
type EventKind int

const (
	EventMessage EventKind = iota
	EventNote
	EventBlockStart
	EventBlockSection
	EventBlockEnd
	EventActivate
	EventDeactivate
)

// LineStyle i HeadStyle opisują strzałkę komunikatu.
type LineStyle int

const (
	Solid LineStyle = iota
	Dotted
)

type HeadStyle int

const (
	HeadNone  HeadStyle = iota // ->
	HeadArrow                  // ->>
	HeadCross                  // -x
	HeadOpen                   // -)
)

// NotePlacement to położenie notatki.
type NotePlacement int

const (
	NoteRightOf NotePlacement = iota
	NoteLeftOf
	NoteOver
)

// Event to jedno zdarzenie; pola zależą od Kind.
type Event struct {
	Kind EventKind
	Line int // numer linii w źródle

	// Message
	From, To       int // indeksy uczestników
	LineStyle      LineStyle
	Head           HeadStyle
	Bidirectional  bool
	ActivateTo     bool // skrót „+”
	DeactivateFrom bool // skrót „-”

	// Note
	Placement NotePlacement
	Over      [2]int // zakres uczestników (dla NoteOver może być dwóch)

	// Activate, Deactivate
	Target int

	// BlockStart, BlockSection
	Block string // loop, alt, opt, par, critical, break, rect
	Color string // dla rect

	Text []string // linie tekstu komunikatu, notatki, tytułu bloku
}
