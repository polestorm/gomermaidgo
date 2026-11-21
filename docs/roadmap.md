# Plan

| Etap | Zakres | Stan |
|---|---|---|
| 1 | diagram sekwencji: parser, układ, SVG, korpus, wymagania | zrobione 2026-11-21 (profil w `profile.md`) |
| 2 | parser schematu blokowego (`flowchart`/`graph`) i stanów (`stateDiagram-v2`); wspólny model grafu | następny |
| 3 | układ warstwowy: usuwanie cykli, ranking, redukcja przecięć, rozmieszczenie, prowadzenie krawędzi, podgrafy, etykiety krawędzi | po etapie 2 |
| 4 | v2: kolejne typy (ER, klasy, Gantt, mapa myśli, oś czasu) | później |

Do zrobienia w etapie 1, zanim uznamy go za zamknięty:

- zgłaszanie znaków, których nie ma w podanym foncie (dziś liczone, ale niewidoczne dla wywołującego);
- `box` (grupy uczestników) i `autonumber` z parametrami, jeśli okażą się potrzebne;
- automatyczny test przejścia przez `gopdfgo`;
- pomiar pisma pogrubionego (dziś rysunek nie używa pogrubienia).
