# Wymagania: diagram stanów

Stan: parser (etap 2). Wymagania dotyczące rysunku dojdą z układem warstwowym (etap 3).

| Id | Wymaganie |
|---|---|
| STATE-001 | Źródło zaczynające się od `stateDiagram-v2` albo `stateDiagram` daje model grafu; komentarze `%%` i puste linie są pomijane. |
| STATE-002 | Przejścia `A --> B` z opcjonalną etykietą po dwukropku; `[*]` jako źródło to stan początkowy, jako cel to stan końcowy. |
| STATE-003 | Opis stanu: `state "Opis" as Id` oraz `Id : opis`; kolejne `Id : opis` dopisują linie. |
| STATE-004 | Stan użyty tylko w przejściu albo podany samym identyfikatorem w osobnej linii ma identyfikator jako etykietę; `<br/>` dzieli linie. |
| STATE-005 | Stany złożone `state Id { … }`, zagnieżdżone; `[*]` wewnątrz ma zasięg stanu złożonego; przejście może prowadzić do stanu złożonego. |
| STATE-006 | `state Id <<choice>>`, `<<fork>>`, `<<join>>` dają węzły wyboru, rozwidlenia i złączenia. |
| STATE-007 | Notatki `note right of Id : tekst`, `note left of Id : tekst` oraz wieloliniowe, zamykane `end note`; komentarze wewnątrz notatki są pomijane. |
| STATE-008 | `direction` ustawia kierunek diagramu albo stanu złożonego. |
| STATE-009 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem. |
