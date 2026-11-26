// Package sequence obsługuje diagram sekwencji: tekst → model → scena.
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package sequence

// Diagram to rozebrany diagram sekwencji.
type Diagram struct {
	Title        string
	AccTitle     string // tytuł dostępności (accTitle)
	AccDescr     string // opis dostępności (accDescr)
	Autonumber   bool
	NumberStart  int // pierwszy numer komunikatu
	NumberStep   int // przyrost numeru
	Boxes        []Box
	Participants []*Participant
	Events       []Event
	NoFooter     bool // uczestnicy nie są powtórzeni na dole (ZenUML)
}

// Box to grupa sąsiednich uczestników (instrukcja „box”).
type Box struct {
	Title       string
	Color       string // pusty = sama ramka
	First, Last int    // indeksy uczestników
}

// Participant to uczestnik: prostokąt albo aktor (ludzik).
type Participant struct {
	ID    string
	Label []string // linie etykiety
	Actor bool
	Kind  string // odmiana: "" (prostokąt), boundary, control, entity, database, collections, queue
	Line  int

	Created   bool // powstaje w trakcie diagramu (instrukcja „create”)
	Destroyed bool // kończy życie w trakcie diagramu (instrukcja „destroy”)
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
	Created        int  // indeks uczestnika tworzonego tym komunikatem + 1; 0 = brak
	Destroyed      int  // indeks uczestnika kończącego życie tym komunikatem + 1; 0 = brak

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

// Kinds to odmiany uczestnika poza prostokątem i aktorem.
var Kinds = []string{"boundary", "control", "entity", "database", "collections", "queue"}

// figure mówi, czy uczestnik jest symbolem z podpisem pod nim (aktor, granica, sterowanie,
// encja), a nie polem z podpisem w środku.
func (p *Participant) figure() bool {
	return p.Actor || p.Kind == "boundary" || p.Kind == "control" || p.Kind == "entity"
}
