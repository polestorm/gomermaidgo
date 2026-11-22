# Wymagania: diagram wymagań


| Id | Wymaganie |
|---|---|
| RQD-001 | Źródło zaczynające się od `requirementDiagram` daje model grafu. Blok `rodzaj nazwa { … }` definiuje wymaganie (`requirement`, `functionalRequirement`, `interfaceRequirement`, `performanceRequirement`, `physicalRequirement`, `designConstraint`) albo element (`element`); nazwa może być w cudzysłowach. Komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| RQD-002 | Pola wymagania: `id`, `text`, `risk` (Low, Medium, High) i `verifymethod` (Analysis, Inspection, Test, Demonstration); pola elementu: `type` i `docref`. Na rysunku pole ma rodzaj w cudzysłowach ostrokątnych nad pogrubioną nazwą, a pod nimi pola jako „Etykieta: wartość”; długa wartość jest łamana na kilka linii. |
| RQD-003 | Relacje `a - rodzaj -> b` i `b <- rodzaj - a` (contains, copies, derives, satisfies, verifies, refines, traces) są krawędziami od `a` do `b` z rodzajem jako etykietą; `contains` jest linią ciągłą z kółkiem przy elemencie zawierającym, pozostałe (zależności) linią przerywaną z otwartym grotem przy celu, jak w SysML. `direction`, `classDef`, `class`, skrót `:::` i `style` działają jak w schemacie blokowym (FLOW-017). |
| RQD-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nieznany rodzaj bloku, pola albo relacji, zła wartość ryzyka albo metody weryfikacji, niezamknięty blok, nazwa zdefiniowana dwa razy, relacja z nazwą bez definicji, nierozpoznana linia. |
