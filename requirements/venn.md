# Wymagania: diagram Venna


| Id | Wymaganie |
|---|---|
| VENN-001 | Źródło zaczynające się od `venn-beta` albo `venn` daje diagram Venna. `title` dodaje tytuł nad kołami; komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| VENN-002 | `set Id`, `set Id["podpis"]` i `union Id,Id[,Id]["podpis"]` deklarują zbiory i części wspólne; nazwa może być w cudzysłowie, a nazwy w `union` muszą być wcześniej zadeklarowane. Przyrostek `:N` podaje wielkość. Wcięte linie `text Id["podpis"]` należą do ostatniego zbioru albo części wspólnej. |
| VENN-003 | `style Id[,Id…] właściwości` nadaje wygląd zbiorowi, części wspólnej (lista nazw jej zbiorów) albo elementowi `text`: `fill`, `fill-opacity`, `stroke`, `stroke-width`, `color`. Wypełnienie części wspólnej pokrywa cały obszar wspólny jej kół. |
| VENN-004 | Rysunek: zbiory to koła o polu proporcjonalnym do wielkości (bez wielkości: równe), z przezroczystym wypełnieniem, więc części wspólne są ciemniejsze. Odległość kół wynika z wielkości części wspólnej, gdy jest podana; zbiory bez części wspólnej stoją osobno; część wspólna trzech zbiorów obejmuje części wspólne par. Podpis zbioru stoi w jego części poza innymi kołami, podpis części wspólnej w jej obszarze, elementy `text` pod podpisem; długie podpisy są łamane. Domyślnie rysunek jest neutralny, `WithColors` nadaje kołom tony palety. |
| VENN-005 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, brak zbioru, nieznana instrukcja, zbiór zadeklarowany dwa razy, `union` z mniej niż dwoma zbiorami albo z nazwą niezadeklarowaną, `text` bez zbioru, zła wielkość albo przezroczystość, styl nieznanego celu, więcej niż trzy zbiory albo 60 elementów `text`. |
