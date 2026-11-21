# Plan

| Etap | Zakres | Stan |
|---|---|---|
| 1 | diagram sekwencji: parser, układ, SVG, korpus, wymagania | zrobione 2026-11-21 (profil w `profile.md`) |
| 2 | parser schematu blokowego (`flowchart`/`graph`) i stanów (`stateDiagram-v2`); wspólny model grafu (`internal/graph`) | zrobione 2026-11-21 |
| 3 | układ warstwowy (`internal/layered`): usuwanie cykli, warstwy, redukcja przecięć, rozmieszczenie, krawędzie, podgrafy rekurencyjnie, etykiety krawędzi, kształty węzłów | pierwsza wersja 2026-11-21; ograniczenia w `profile.md` |
| 4 | v2: kolejne typy (ER, klasy, Gantt, mapa myśli, oś czasu) | później |

Do zrobienia w etapie 1, zanim uznamy go za zamknięty:

- zgłaszanie znaków, których nie ma w podanym foncie (dziś liczone, ale niewidoczne dla wywołującego);
- `box` (grupy uczestników) i `autonumber` z parametrami, jeśli okażą się potrzebne;
- automatyczny test przejścia przez `gopdfgo`;
- pomiar pisma pogrubionego (dziś rysunek nie używa pogrubienia).

Decyzje etapu 3 (2026-11-21): `direction` w podgrafie nie zmienia układu; krawędź do podgrafu kończy się na ramce;
krawędź niewidoczna jest tylko ograniczeniem rozmieszczenia.

Do poprawy w układzie, w kolejności wpływu na czytelność:

1. krawędzie przekraczające ramkę podgrafu: prowadzenie przez punkt na ramce i omijanie węzłów wnętrza (dziś wnętrze
   podgrafu jest rozmieszczane bez wiedzy o krawędziach z zewnątrz);
2. podział na warstwy metodą sympleksu sieciowego zamiast najdłuższej ścieżki (krótsze krawędzie, węższy rysunek);
3. rozmieszczenie metodą Brandesa i Köpfa (bardziej zwarte i symetryczne niż metoda priorytetowa);
4. krawędzie wielokrotne między tą samą parą węzłów;
5. strona notatki w diagramie stanów; `direction` we wnętrzu podgrafu;
6. regiony współbieżne stanów i identyfikatory z łącznikiem, jeśli okażą się potrzebne.
