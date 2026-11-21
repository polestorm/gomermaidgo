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
| FLOW-017 | Wygląd ze źródła: `classDef` (także kilka nazw i klasa `default`), `class`, skrót `:::`, `style` dla węzła i podgrafu, `linkStyle` z numerami krawędzi albo `default`. Honorowane właściwości: `fill`, `stroke`, `color`, `stroke-width`, `stroke-dasharray`; pozostałe są pomijane, błędna wartość honorowanej jest błędem. Styl bezpośredni wygrywa z klasą. `click` jest przyjmowane i pomijane. |
| FLOW-018 | Identyfikator może zawierać łącznik i kropkę między literami i cyframi; tekst Markdown w odwrotnych apostrofach daje pogrubienie (`**`) i kursywę (`*`, `_`), mierzone odmianami kroju i zachowane po złamaniu linii; znaczniki `<b>`, `<strong>`, `<i>`, `<em>` w etykiecie dają to samo wyróżnienie, także przez złamanie linii, a inne znaczniki zostają tekstem; tekst Markdown może zajmować kilka linii źródła (koniec linii łamie tekst, numeracja linii w błędach zostaje); `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| FLOW-019 | `direction` w podgrafie (i w stanie złożonym) zmienia kierunek jego wnętrza, gdy żadna krawędź nie łączy wnętrza z resztą diagramu; podgrafy zagnieżdżone dziedziczą ten kierunek; w przeciwnym razie wnętrze ma kierunek otoczenia. Niezmienniki układu (FLOW-013) obowiązują także wtedy. |
| FLOW-020 | Podział na warstwy minimalizuje sumę długości krawędzi (sympleks sieciowy) przy zachowaniu minimalnych długości z FLOW-005; części niepołączone zaczynają od tej samej warstwy. |
| FLOW-021 | Tory: komentarz `%% layout: swimlanes` zamienia podgrafy najwyższego poziomu w pasy biegnące przez cały rysunek wzdłuż kierunku diagramu, z tytułem w nagłówku przy początku; każde pole leży w pasie swojego podgrafu, pola spoza podgrafów w pasie bez tytułu; warstwy są wspólne dla wszystkich pasów; pole będące w swoim rzędzie jedynym polem toru stoi pośrodku pasa. Podgraf zagnieżdżony i krawędź do toru są błędem. Bez dyrektywy to samo źródło rysuje zwykłe podgrafy. |
