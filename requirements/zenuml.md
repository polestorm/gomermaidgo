# Wymagania: diagram sekwencji ZenUML


| Id | Wymaganie |
|---|---|
| ZEN-001 | Źródło zaczynające się od `zenuml` daje diagram sekwencji rysowany jak `sequenceDiagram`. `title` w pierwszej linii po nagłówku dodaje tytuł; komentarze `%%` są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| ZEN-002 | Uczestnicy powstają przy pierwszym użyciu w kolejności źródła; deklaracja `Nazwa` ustala kolejność, `A as Alicja` nadaje podpis, `@Actor Nazwa` rysuje postać, inne adnotatory (`@Database`, `@Boundary` i podobne) stoją jako stereotyp «Database» nad nazwą. Deklaracja po pierwszym użyciu jest błędem. |
| ZEN-003 | Komunikat asynchroniczny `A->B: tekst` ma otwarty grot; poprzedzony `@return` albo `@reply` jest odpowiedzią (linia przerywana). Komentarz `// tekst` przed komunikatem staje się notatką nad nim; komentarz przy uczestniku nie jest rysowany. |
| ZEN-004 | Wywołanie synchroniczne `B.metoda(argumenty)` (albo `A->B.metoda()`) wychodzi od wywołanego w bieżącym bloku, na najwyższym poziomie od postaci bez nazwy, i aktywuje B; blok `{ … }` zawiera wywołania wychodzące od B. `return wartość` w bloku albo przypisanie `[Typ] zmienna = B.metoda()` daje odpowiedź przerywaną kończącą aktywację. `new A(argumenty)`, także z przypisaniem, tworzy uczestnika komunikatem «create». |
| ZEN-005 | Bloki sterujące: `while`, `for`, `forEach`, `foreach` i `loop` dają pętlę z warunkiem; `if(…)`, `} else if(…) {` i `} else {` alternatywę z gałęziami; `opt`, `par`; `try`, `} catch {`, `} finally {` ramkę try z gałęziami. Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nieznana instrukcja albo blok, niezamknięta albo nadmiarowa klamra, gałąź po niewłaściwym bloku, `return` poza wywołaniem, ponowne utworzenie istniejącego uczestnika, przekroczenie limitów (1000 komunikatów, 30 poziomów bloków). |
