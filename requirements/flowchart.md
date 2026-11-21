# Wymagania: schemat blokowy


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
| FLOW-012 | Każdy tekst ze źródła jest na rysunku; etykiety szersze niż limit (200 px przy piśmie 16 px, 150 px dla rombu i koła) są łamane na granicach wyrazów. |
| FLOW-013 | Układ warstwowy w czterech kierunkach: kształty węzłów nie nachodzą na siebie; ramki podgrafów są rozłączne albo zagnieżdżone i zawierają swoje węzły w całości; etykiety krawędzi nie nachodzą na węzły ani na siebie; każda krawędź poza niewidoczną ma ścieżkę; cykle, pętle własne i krawędzie wielokrotne są rysowane. |
| FLOW-014 | Krawędź do węzła wewnątrz podgrafu przechodzi przez punkt na ramce i jest prowadzona we wnętrzu podgrafu tak jak pozostałe krawędzie: nie przecina węzłów, także przy podgrafach zagnieżdżonych i między dwoma podgrafami. |
| FLOW-015 | Kilka krawędzi między tą samą parą węzłów jest rysowanych osobno, obok siebie, także gdy węzeł ma jeden punkt zaczepienia (romb, koło). |
| FLOW-016 | Węzły, do których nie prowadzi żadna krawędź, leżą w pierwszej warstwie, a w cyklu odwracana jest krawędź powrotna; części diagramu niepołączone krawędziami leżą obok siebie bez pustych przerw. |
