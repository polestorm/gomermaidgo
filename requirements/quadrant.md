# Wymagania: wykres ćwiartkowy


| Id | Wymaganie |
|---|---|
| QDR-001 | Źródło zaczynające się od `quadrantChart` daje wykres; `title` nadaje tytuł; `x-axis lewy --> prawy` i `y-axis dolny --> górny` opisują końce osi (drugi opis jest opcjonalny); `quadrant-1` do `quadrant-4` nadają tytuły ćwiartkom: prawej górnej, lewej górnej, lewej dolnej, prawej dolnej. Komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| QDR-002 | Punkt to `Nazwa: [x, y]` ze współrzędnymi od 0 do 1; x rośnie w prawo, y w górę. Styl punktu po nawiasie, skrót `:::` i `classDef` są przyjmowane i pomijane. |
| QDR-003 | Rysunek: kwadrat podzielony na cztery ćwiartki, każdy punkt w swoim miejscu z nazwą obok (przy prawej krawędzi po lewej stronie punktu), nazwa na własnym podkładzie. Tytuł ćwiartki stoi tam, gdzie nie zasłaniają go punkty: u góry, u dołu albo pośrodku. Opisy osi stoją poziomo: pod wykresem i z jego lewej strony. |
| QDR-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nierozpoznana linia, współrzędna spoza zakresu 0–1 albo niebędąca liczbą, oś bez opisu, więcej niż 100 punktów. |
