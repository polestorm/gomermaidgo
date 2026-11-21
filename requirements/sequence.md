# Wymagania: diagram sekwencji

Każde wymaganie ma identyfikator `SEQ-nnn`. Pliki korpusu (`testdata/corpus/sequence/*.mmd`) i testy odwołują się do nich
przez `REQ: SEQ-nnn`; test `TestRequirementsCovered` sprawdza, że każde wymaganie ma pokrycie i że każde odwołanie
wskazuje istniejące wymaganie.

| Id | Wymaganie |
|---|---|
| SEQ-001 | Źródło zaczynające się od `sequenceDiagram` daje dokument SVG; komentarze `%%` i puste linie są pomijane. |
| SEQ-002 | Uczestnicy: `participant` (prostokąt) i `actor` (ludzik), z aliasem `as`; uczestnik użyty tylko w komunikacie powstaje sam, w kolejności pierwszego użycia. Uczestnicy są powtórzeni na dole. |
| SEQ-003 | Komunikaty: `->`, `-->`, `->>`, `-->>`, `-x`, `--x`, `-)`, `--)`, `<<->>`, `<<-->>`, w obu kierunkach i do siebie; etykieta nad linią. |
| SEQ-004 | Notatki: `Note right of`, `Note left of`, `Note over` jednego albo dwóch uczestników. |
| SEQ-005 | Aktywacje: `activate`/`deactivate` oraz skróty `+` i `-` przy odbiorcy; aktywacje zagnieżdżone są przesunięte. |
| SEQ-006 | Bloki: `loop`, `alt`/`else`, `opt`, `par`/`and`, `critical`/`option`, `break`, zamykane `end`; ramka z rodzajem i tytułem. |
| SEQ-007 | Bloki można zagnieżdżać; `rect` z kolorem `rgb()`, `rgba()` albo `#hex` daje wyróżnienie tła. |
| SEQ-008 | `autonumber` numeruje komunikaty od 1 w kolejności źródła. |
| SEQ-009 | Etykieta komunikatu między sąsiednimi uczestnikami mieści się między ich liniami życia; teksty nie nachodzą na siebie. |
| SEQ-010 | Tekst może zawierać dowolne znaki Unicode (w tym polskie), znaki specjalne XML i podział linii `<br/>`; `title` daje tytuł. |
| SEQ-011 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem; nigdy niepełny rysunek. |
| SEQ-012 | Wynik jest poprawnym XML, zawiera tylko dozwolone elementy i atrybuty SVG, bez skryptów, odnośników i `foreignObject`; ten sam tekst i opcje dają ten sam wynik co do bajta. |
| SEQ-013 | Szerokości tekstu pochodzą z podanego fontu (tabele `cmap` formatu 4 i 12, `hmtx`); bez fontu działa miara zastępcza. |
| SEQ-014 | Grupy uczestników `box [kolor] [tytuł]` … `end`: ramka obejmuje uczestników od nagłówka do stopki, z tytułem u góry; kolor w zapisie `rgb()`, `rgba()`, `#hex`, nazwą podstawową albo `transparent`; ramka mieści swój tytuł. |
| SEQ-015 | `create participant` i `create actor` stawiają uczestnika na wysokości następnego komunikatu, który musi go łączyć z innym uczestnikiem; `destroy` kończy linię życia krzyżykiem po następnym komunikacie z jego udziałem; uczestnik zakończony nie ma stopki i nie może być dalej używany. |
| SEQ-016 | `accTitle` i `accDescr` (także wieloliniowe w klamrach) trafiają do elementów `title` i `desc` dokumentu SVG; dotyczy wszystkich typów diagramów. |
