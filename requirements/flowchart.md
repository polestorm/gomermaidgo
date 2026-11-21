# Wymagania: schemat blokowy

Stan: parser (etap 2). Wymagania dotyczące rysunku dojdą z układem warstwowym (etap 3).

| Id | Wymaganie |
|---|---|
| FLOW-001 | Źródło zaczynające się od `flowchart` albo `graph`, z opcjonalnym kierunkiem `TB`, `TD`, `BT`, `LR`, `RL`, daje model grafu; komentarze `%%` i puste linie są pomijane. |
| FLOW-002 | Węzły: sam identyfikator (prostokąt z identyfikatorem jako etykietą) oraz 14 kształtów: `[ ]`, `( )`, `([ ])`, `[[ ]]`, `[( )]`, `(( ))`, `((( )))`, `> ]`, `{ }`, `{{ }}`, `[/ /]`, `[\ \]`, `[/ \]`, `[\ /]`. Późniejsza definicja kształtu zastępuje wcześniejszą. |
| FLOW-003 | Krawędzie: linia zwykła, gruba, kropkowana i niewidoczna; zakończenia: brak, grot, kółko, krzyżyk; warianty dwustronne `<-->`, `o--o`, `x--x`. |
| FLOW-004 | Etykieta krawędzi w zapisie `-->|tekst|` oraz w środku krawędzi (`-- tekst -->`, `== tekst ==>`, `-. tekst .->`), także w cudzysłowie. |
| FLOW-005 | Dodatkowe znaki linii wydłużają krawędź: minimalna liczba warstw między końcami rośnie o jeden na każdy dodatkowy znak. |
| FLOW-006 | Łańcuch `A --> B --> C` i grupy `A & B --> C & D` dają krawędź dla każdej pary sąsiednich grup. |
| FLOW-007 | Podgrafy `subgraph id [tytuł]` … `end`, zagnieżdżone, z `direction`; węzeł należy do podgrafu, w którym po raz pierwszy wystąpił wewnątrz podgrafu; krawędź może prowadzić do podgrafu. |
| FLOW-008 | Tekst w cudzysłowie może zawierać nawiasy, znaki krawędzi i średnik; `<br/>` dzieli linie; identyfikatory mogą zawierać litery Unicode, cyfry i podkreślenie. |
| FLOW-009 | Średnik rozdziela instrukcje w linii i może kończyć linię. |
| FLOW-010 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem. |
| FLOW-011 | Spójność grafu: krawędź łącząca podgraf z jego własną zawartością, identyfikator użyty jako węzeł z kształtem i jako podgraf oraz przekroczenie limitów (500 węzłów, 1000 krawędzi, 10 poziomów) są błędami. |
