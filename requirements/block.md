# Wymagania: diagram blokowy


| Id | Wymaganie |
|---|---|
| BLK-001 | Źródło zaczynające się od `block-beta` (albo `block`) daje diagram; `columns n` ustala liczbę kolumn siatki (bez niego albo z `auto` wszystkie elementy stoją w jednym rzędzie). Elementy linii są rozdzielone odstępami: `id`, `id["Etykieta"]` i pozostałe kształty, `space`, każdy z opcjonalną szerokością `:n` w kolumnach. Komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| BLK-002 | Kształty: `[prostokąt]`, `(zaokrąglony)`, `([z półkolami])`, `((koło))`, `{romb}`, `{{sześciokąt}}`, `[(walec)]` i strzałka blokowa `id<["Etykieta"]>(right)` z kierunkiem right, left, up, down albo x. `block:id:n … end` jest blokiem złożonym z własną siatką i ramką; bloki złożone mogą się zagnieżdżać. |
| BLK-003 | Rysunek: elementy stoją w siatce w kolejności ze źródła; kolumny jednej siatki mają równą szerokość, element zajmuje tyle kolumn, ile ma szerokości, a rząd ma wysokość najwyższego elementu; element, który nie mieści się w reszcie rzędu, zaczyna następny. Bloki się nie nakładają. Połączenia `a --> b`, `a --- b`, `a -.-> b`, z etykietą `a -- "tekst" --> b`, są odcinkami między brzegami kształtów. `style`, `classDef` i `class` nadają blokom wygląd. |
| BLK-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, element nie do odczytania, zła liczba kolumn albo szerokość, blok zdefiniowany dwa razy, `end` bez bloku, blok niezamknięty, połączenie albo styl z nieznanym blokiem, zbyt głębokie zagnieżdżenie, więcej niż 300 bloków, brak bloków. |
