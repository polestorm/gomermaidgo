# Plan

| Etap | Zakres | Stan |
|---|---|---|
| 1 | diagram sekwencji: parser, układ, SVG, korpus, wymagania | zrobione 2026-11-21 (profil w `profile.md`) |
| 2 | parser schematu blokowego (`flowchart`/`graph`) i stanów (`stateDiagram-v2`); wspólny model grafu (`internal/graph`) | zrobione 2026-11-21 |
| 3 | układ warstwowy (`internal/layered`): usuwanie cykli, warstwy, redukcja przecięć, rozmieszczenie, krawędzie, podgrafy rekurencyjnie, etykiety krawędzi, kształty węzłów | pierwsza wersja 2026-11-21; ograniczenia w `profile.md` |
| 4 | v2: kolejne typy | wykres Gantta i diagram związków encji zrobione 2026-11-21; dalej: klasy, wykresy, oś czasu, mapa myśli |
| 6 | tory (`%% layout: swimlanes`), krawędzie łamane z mostkami, wyrównywanie pól, rzędy pod proporcje strony | zrobione 2026-11-21 po przeglądzie rysunków |
| 7 | próba na cudzych diagramach; diagram encji pod proporcje strony; nowe typy: wykres kołowy (`pie`), diagram klas (`classDiagram`), oś czasu (`timeline`), podróż użytkownika (`journey`), mapa myśli (`mindmap`), wykres słupkowy i liniowy (`xychart-beta`), wykres ćwiartkowy (`quadrantChart`) | zrobione 2026-11-21 |
| 8 | wynik PDF bez pośrednictwa SVG i HTML: opcjonalny pakiet `pdf`, krój osadzony jako podzbiór (ADR 0003) | zrobione 2026-11-21 |
| 9 | pozostałe typy Mermaid: `kanban`, `packet`, `radar`, `treemap`, `requirementDiagram`, `sankey`, C4, `gitGraph`, `block`, `architecture`; rejestr typów w `mermaid/types.go` | zrobione 2026-11-22 |
| 10 | składnia spoza diagramów i zgodność z cudzymi źródłami: blok nagłówkowy, dyrektywy, nazwane kształty, przestrzenie nazw klas; program przyjmuje Markdown i flagi `mermaid-cli` | zrobione 2026-11-22 |
| 11 | wynik PNG: opcjonalny pakiet `png`, własny rasteryzator i kontury glifów (ADR 0004) | zrobione 2026-11-22 |
| 12 | motywy: własne, pod nazwami Mermaid, wybierane opcją, flagą `-t` albo w źródle; tekst obrócony w scenie (ADR 0005) | zrobione 2026-11-22 |
| 13 | krój zapasowy z emoji (Noto Emoji, jednobarwne) w pomiarze, SVG, PDF i PNG (ADR 0006) | zrobione 2026-11-23 |
| 14 | krój domyślny Arimo (szerokości jak Arial i Liberation Sans) zamiast Recursive (ADR 0007) | zrobione 2026-11-23 |
| 15 | dopasowanie do strony i zwijanie długich diagramów w rzędy (ADR 0008) | zrobione 2026-11-23 |
| 16 | dziewięć brakujących typów Mermaid, po kolei, każdy po przeglądzie właściciela (ADR 0011): `treeView`, `venn`, `cynefin`, `usecase`, `zenuml`, `wardley`, `eventmodeling`, `agentflow`, `railroad`; przykłady w `testdata/planned/` | zaplanowane 2026-11-26; `treeView` (TREE-001..005), `venn` (VENN-001..005) i `cynefin` (CYN-001..005) zrobione 2026-11-26, czekają na przegląd |
| 5 | diagramy w dokumencie Markdown (`markdown.Embed`, reguły CSS dla wydruku) | zrobione 2026-11-21 |

Zrobione 2026-11-21 po pierwszym przeglądzie: krój domyślny Recursive (ADR 0002), `box`, `create`/`destroy`,
`autonumber` z parametrami, `accTitle`/`accDescr`, style ze źródła (`classDef`, `class`, `style`, `linkStyle`, `:::`),
regiony współbieżne, `direction` w podgrafie, zgłaszanie znaków spoza fontu (`MissingGlyphs`), test drogi do PDF
(`tools/pdfcheck`).

Zrobione też: pogrubienie i kursywa w tekstach Markdown (odmiany Bold i Italic kroju w module), kolor podkładu
etykiet w grupie `box`, regiony współbieżne w stosie z linią podziału, uzgodnienie punktów na ramce z otoczeniem.

Decyzje etapu 3 (2026-11-21): `direction` w podgrafie zmienia układ tylko w podgrafie niepołączonym z resztą; krawędź do podgrafu kończy się na ramce;
krawędź niewidoczna jest tylko ograniczeniem rozmieszczenia.

Do poprawy w układzie, w kolejności wpływu na czytelność:

1. ~~krawędzie przekraczające ramkę podgrafu~~: zrobione 2026-11-21. Podgraf dostaje punkty wejścia i wyjścia na
   ramce, a odcinek do pola wewnątrz jest prowadzony w układzie podgrafu; kolejność punktów na ramce nie jest jeszcze
   uzgadniana z układem rodzica, więc dwie krawędzie mogą się skrzyżować tuż przed ramką;
   Zrobione też 2026-11-21: usuwanie cykli zaczyna od pól bez krawędzi wchodzących (początek procesu jest na
   górze, odwracane są krawędzie powrotne); części diagramu niepołączone krawędziami są rozmieszczane osobno
   i ustawiane obok siebie;
2. ~~podział na warstwy metodą sympleksu sieciowego~~: zrobione 2026-11-21 (FLOW-020); wartości przecięć liczone od
   nowa po każdej wymianie, bez przyrostowego liczenia z publikacji;
3. rozmieszczenie metodą Brandesa i Köpfa (bardziej zwarte i symetryczne niż metoda priorytetowa);
4. ~~krawędzie wielokrotne między tą samą parą węzłów~~: zrobione 2026-11-21 (FLOW-015);
5. ~~strona notatki w diagramie stanów~~: zrobione 2026-11-21 (STATE-011);
6. ~~uzgodnienie kolejności punktów na ramce podgrafu z układem otoczenia~~: zrobione 2026-11-21 (jeden poziom
   w głąb; podgrafy zagnieżdżone głębiej zachowują kolejność z własnego wnętrza).
