# Wymagania: diagram związków encji

Każde wymaganie ma identyfikator `ER-nnn`; pliki korpusu (`testdata/corpus/er/*.mmd`) i testy odwołują się do nich
przez `REQ: ER-nnn`.

| Id | Wymaganie |
|---|---|
| ER-001 | Źródło zaczynające się od `erDiagram` daje model grafu i rysunek układem warstwowym (niezmienniki FLOW-013); komentarze `%%` i puste linie są pomijane; `direction`, `accTitle`, `accDescr`. |
| ER-002 | Encja: sama nazwa, nazwa w cudzysłowie, alias `NAZWA["Opis"]`, blok atrybutów `{ typ nazwa [PK, FK, UK] ["komentarz"] }`; encja użyta tylko w związku powstaje sama. Każdy tekst jest na rysunku; atrybuty stoją w tabeli pod nazwą. |
| ER-003 | Związek `A <liczność><linia><liczność> B [: etykieta]`: liczności `\|o`, `\|\|`, `}o`, `}\|` (lustrzanie po prawej), linia `--` (ciągła) albo `..` (kropkowana); liczność jest rysowana przy encji, której dotyczy. |
| ER-004 | Wygląd ze źródła jak w schemacie blokowym (FLOW-017): `classDef`, `class`, `:::`, `style`. Związek encji z samą sobą jest rysowany pętlą. |
| ER-005 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem. |
