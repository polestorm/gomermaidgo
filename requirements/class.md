# Wymagania: diagram klas


| Id | Wymaganie |
|---|---|
| CLS-001 | Źródło zaczynające się od `classDiagram` (albo `classDiagram-v2`) daje model grafu; komentarze `%%` i puste linie są pomijane. Klasa powstaje z deklaracji `class Nazwa`, z relacji albo ze składowej; `class Nazwa["Etykieta"]` nadaje etykietę; parametr typu w tyldach (`Lista~T~`) jest rysowany w nawiasach ostrych. |
| CLS-002 | Składowe: w bloku `class Nazwa { … }` po jednej w linii albo jako `Nazwa : składowa`. Składowa z nawiasem jest metodą, pozostałe atrybutami; atrybuty i metody stoją w osobnych przedziałach, w kolejności ze źródła. Typ zwracany po nawiasie metody jest rysowany po dwukropku. Znaki widoczności (`+`, `-`, `#`, `~`) zostają w tekście. |
| CLS-003 | Klasyfikator `*` (składowa abstrakcyjna) daje kursywę, `$` (statyczna) podkreślenie; stoi po nawiasie metody albo na końcu składowej. Adnotacja `<<tekst>>` w bloku klasy albo jako `<<tekst>> Nazwa` jest rysowana nad nazwą w cudzysłowach ostrokątnych. Klasa bez składowych jest prostym polem z nazwą. |
| CLS-004 | Relacje: `<\|--` dziedziczenie, `*--` kompozycja, `o--` agregacja, `-->` asocjacja, `--` połączenie, `..>` zależność, `..\|>` realizacja, `..` połączenie przerywane; każde zakończenie może stać po dowolnej stronie. Zakończenia: pusty trójkąt, pełny i pusty romb, otwarty grot. Klasa wymieniona po lewej stronie relacji stoi w układzie wyżej. |
| CLS-005 | Etykieta relacji po dwukropku stoi na krawędzi; liczności w cudzysłowach (`A "1" --> "*" B`) stoją przy końcach krawędzi, a krawędź z licznościami przy obu końcach jest o warstwę dłuższa. |
| CLS-006 | `direction` zmienia kierunek układu; `note for Nazwa "tekst"` rysuje notatkę przy klasie (`<br/>` łamie linię); relacja klasy z samą sobą jest pętlą; `accTitle` i `accDescr` trafiają do `title` i `desc`; `click`, `link` i `callback` są przyjmowane i pomijane. |
| CLS-007 | Wygląd ze źródła: `classDef`, skrót `:::` w deklaracji klasy, `cssClass "A,B" nazwa`, `style Nazwa …`, z tymi samymi właściwościami co w schemacie blokowym (FLOW-017). |
| CLS-008 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem: przestrzeń nazw, interfejs „lizak” `()`, notatka bez klasy, niezamknięty blok składowych, nierozpoznana linia. Limity grafu (FLOW-011) obowiązują także tutaj. |
