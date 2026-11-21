# Wymagania: diagram stanów


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
| STATE-010 | Każdy tekst ze źródła jest na rysunku; stan początkowy, końcowy, wybór, rozwidlenie i złączenie mają własne kształty bez tekstu. |
| STATE-011 | Układ warstwowy jak dla schematu blokowego (FLOW-013); notatka stoi w warstwie swojego stanu, po wskazanej stronie (w układzie pionowym: po lewej albo po prawej), połączona z nim krótką linią kropkowaną. |
| STATE-012 | Regiony współbieżne: separator `--` wewnątrz stanu złożonego dzieli go na regiony rysowane ramkami przerywanymi bez tytułu; `[*]` ma zasięg regionu. |
| STATE-013 | Wygląd ze źródła: `classDef`, `class`, skrót `:::` przy identyfikatorze stanu i `style`, z tymi samymi właściwościami co w schemacie blokowym (FLOW-017). |
| STATE-014 | Identyfikator stanu może zawierać łącznik i kropkę; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
