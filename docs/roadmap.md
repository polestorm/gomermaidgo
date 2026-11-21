# Plan

| Etap | Zakres | Stan |
|---|---|---|
| 1 | diagram sekwencji: parser, układ, SVG, korpus, wymagania | zrobione 2026-11-21 (profil w `profile.md`) |
| 2 | parser schematu blokowego (`flowchart`/`graph`) i stanów (`stateDiagram-v2`); wspólny model grafu (`internal/graph`) | zrobione 2026-11-21 |
| 3 | układ warstwowy (`internal/layered`): usuwanie cykli, warstwy, redukcja przecięć, rozmieszczenie, krawędzie, podgrafy rekurencyjnie, etykiety krawędzi, kształty węzłów | pierwsza wersja 2026-11-21; ograniczenia w `profile.md` |
| 4 | v2: kolejne typy (ER, klasy, Gantt, mapa myśli, oś czasu) | później |

Zrobione 2026-11-21 po pierwszym przeglądzie: krój domyślny Recursive (ADR 0002), `box`, `create`/`destroy`,
`autonumber` z parametrami, `accTitle`/`accDescr`, style ze źródła (`classDef`, `class`, `style`, `linkStyle`, `:::`),
regiony współbieżne, `direction` w podgrafie, zgłaszanie znaków spoza fontu (`MissingGlyphs`), test drogi do PDF
(`tools/pdfcheck`).

Zostaje: pogrubienie i kursywa w tekstach Markdown (wymaga drugiego kroju i jego pomiaru), kolor podkładu etykiet
w grupie `box` z kolorem.

Decyzje etapu 3 (2026-11-21): `direction` w podgrafie zmienia układ tylko w podgrafie niepołączonym z resztą; krawędź do podgrafu kończy się na ramce;
krawędź niewidoczna jest tylko ograniczeniem rozmieszczenia.

Do poprawy w układzie, w kolejności wpływu na czytelność:

1. ~~krawędzie przekraczające ramkę podgrafu~~: zrobione 2026-11-21. Podgraf dostaje punkty wejścia i wyjścia na
   ramce, a odcinek do pola wewnątrz jest prowadzony w układzie podgrafu; kolejność punktów na ramce nie jest jeszcze
   uzgadniana z układem rodzica, więc dwie krawędzie mogą się skrzyżować tuż przed ramką;
   Zrobione też 2026-11-21: usuwanie cykli zaczyna od pól bez krawędzi wchodzących (początek procesu jest na
   górze, odwracane są krawędzie powrotne); części diagramu niepołączone krawędziami są rozmieszczane osobno
   i ustawiane obok siebie;
2. podział na warstwy metodą sympleksu sieciowego zamiast najdłuższej ścieżki (krótsze krawędzie, węższy rysunek);
3. rozmieszczenie metodą Brandesa i Köpfa (bardziej zwarte i symetryczne niż metoda priorytetowa);
4. ~~krawędzie wielokrotne między tą samą parą węzłów~~: zrobione 2026-11-21 (FLOW-015);
5. ~~strona notatki w diagramie stanów~~: zrobione 2026-11-21 (STATE-011);
6. uzgodnienie kolejności punktów na ramce podgrafu z układem otoczenia (dziś otoczenie dopasowuje się do punktów,
   nie odwrotnie);
7. regiony współbieżne jeden pod drugim, z linią podziału zamiast osobnych ramek.
