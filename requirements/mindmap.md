# Wymagania: mapa myśli


| Id | Wymaganie |
|---|---|
| MND-001 | Źródło zaczynające się od `mindmap` daje drzewo; komentarze `%%` i puste linie są pomijane. Hierarchię wyznacza wcięcie: rodzicem gałęzi jest najbliższa wcześniejsza gałąź o mniejszym wcięciu. Mapa ma jeden korzeń. |
| MND-002 | Każdy tekst gałęzi jest na rysunku; `<br/>` łamie linię, długi tekst jest łamany na granicach wyrazów; linie `::icon(…)` i `:::klasa` są pomijane. |
| MND-003 | Układ jest drzewem: gałęzie jednej głębokości zaczynają się w tej samej kolumnie, w kolejności ze źródła; gałąź stoi pośrodku wysokości swoich potomków; pola się nie nakładają. Gałęzie główne stoją po prawej stronie korzenia albo po obu stronach, gdy rysunek wyraźnie lepiej mieści się wtedy na stronie. |
| MND-004 | Kształty gałęzi: `[kwadrat]`, `(zaokrąglony)`, `((koło))`, `{{sześciokąt}}`, `))wybuch((` i `)chmura(`; przed kształtem może stać identyfikator. Tekst może być w cudzysłowach; tekst Markdown w odwrotnych apostrofach daje pogrubienie i kursywę. Gałąź bez kształtu jest polem zaokrąglonym, a korzeń bez kształtu polem z półkolami. |
| MND-005 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, drugi korzeń, gałąź bez tekstu, brak korzenia. Limity grafu (FLOW-011) obowiązują także tutaj. |
