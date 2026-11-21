# Profil składni

Obsługiwany zakres składni Mermaid. Wszystko poza tabelą „Obsługiwane” daje błąd z numerem linii.

## Diagram sekwencji (`sequenceDiagram`)

### Obsługiwane

| Element | Zapis | Wymaganie |
|---|---|---|
| nagłówek, komentarze | `sequenceDiagram`, `%% komentarz` | SEQ-001 |
| uczestnik, aktor, alias | `participant A`, `actor A`, `participant A as Etykieta` | SEQ-002 |
| uczestnik domyślny | pierwsze użycie w komunikacie albo notatce | SEQ-002 |
| komunikat | `A->>B: tekst`; strzałki `->`, `-->`, `->>`, `-->>`, `-x`, `--x`, `-)`, `--)`, `<<->>`, `<<-->>` | SEQ-003 |
| komunikat do siebie | `A->>A: tekst` | SEQ-003 |
| notatka | `Note right of A: t`, `Note left of A: t`, `Note over A: t`, `Note over A,B: t` | SEQ-004 |
| aktywacja | `activate A`, `deactivate A`, `A->>+B: t`, `B-->>-A: t` | SEQ-005 |
| bloki | `loop`, `alt`/`else`, `opt`, `par`/`and`, `critical`/`option`, `break`, zamykane `end`; dowolnie zagnieżdżone | SEQ-006, SEQ-007 |
| wyróżnienie tła | `rect rgb(r, g, b)`, `rect rgba(r, g, b, a)`, `rect #rrggbb` … `end` | SEQ-007 |
| numeracja | `autonumber` | SEQ-008 |
| tytuł | `title Tekst` | SEQ-010 |
| podział linii w tekście | `<br/>`, `<br>` | SEQ-010 |

### Poza profilem (błąd `sequence-unsupported`)

`box` (grupy uczestników), `create` i `destroy`, `link`, `links`, `properties`, `details`, `accTitle`, `accDescr`,
`autonumber` z numerem początkowym i krokiem.

### Świadome różnice wobec Mermaid

- średnik nie rozdziela instrukcji i kody encji (`#35;`) nie są rozwijane: tekst trafia do rysunku dosłownie;
- nie ma zawijania tekstu (`wrap`), dyrektyw `%%{init}%%` ani bloku konfiguracji `---`;
- limity: 50 uczestników, 2000 instrukcji, 20 poziomów zagnieżdżenia bloków;
- dezaktywacja nieaktywnego uczestnika jest błędem, także w zapisie skróconym;
- wygląd jest własny; motywy i zmienne motywu nie są obsługiwane.

## Schemat blokowy, diagram stanów

Jeszcze nieobsługiwane (`docs/roadmap.md`).
