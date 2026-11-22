# Wymagania: diagram architektury


| Id | Wymaganie |
|---|---|
| ARC-001 | Źródło zaczynające się od `architecture-beta` (albo `architecture`) daje diagram. `service id(ikona)[Podpis]` definiuje usługę, `group id(ikona)[Podpis]` grupę, `junction id` węzeł rozgałęzienia; każde może leżeć w grupie (`in grupa`), a grupy mogą się zagnieżdżać. Ikona i podpis są opcjonalne. Komentarze i puste linie są pomijane; `title` nadaje tytuł; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| ARC-002 | Krawędź `a:R -- L:b` łączy wskazane boki (L, R, T, B); `-->`, `<--` i `<-->` dają groty, `-[tekst]-` etykietę, a `{group}` po identyfikatorze przenosi koniec na ramkę grupy usługi. Bok, z którego krawędź wychodzi, wyznacza położenie drugiego końca: po prawej, po lewej, wyżej albo niżej, a przy bokach na różnych osiach po skosie. |
| ARC-003 | Rysunek: usługi stoją w siatce, każda jako ikona z podpisem pod spodem (znane ikony: `database`, `server`, `disk`, `cloud`, `internet`; inne są pustym polem); grupa jest ramką wokół swoich usług i grup podrzędnych, z podpisem u góry oddzielonym od ikon odstępem; usługa spoza grupy nie leży w jej ramce. Żadne dwie usługi nie zajmują tego samego pola siatki; części niepołączone stoją jedna pod drugą. |
| ARC-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nierozpoznana linia, identyfikator zdefiniowany dwa razy, nieznana grupa, krawędź z nieznaną usługą albo do samej siebie, `{group}` przy usłudze bez grupy, więcej niż 100 usług, brak usług. |
