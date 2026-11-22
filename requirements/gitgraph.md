# Wymagania: graf historii Gita


| Id | Wymaganie |
|---|---|
| GIT-001 | Źródło zaczynające się od `gitGraph` (także `gitGraph:` i `gitGraph LR:`) daje graf; historia zaczyna się na gałęzi `main`. `commit` dodaje commit na bieżącej gałęzi, z opcjami `id: "…"`, `tag: "…"` i `type: NORMAL`, `REVERSE` albo `HIGHLIGHT`. Komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| GIT-002 | `branch nazwa` tworzy gałąź od ostatniego commitu bieżącej i na nią przechodzi; `checkout` i `switch` zmieniają bieżącą gałąź; `merge nazwa` dodaje na bieżącej gałęzi commit scalający ostatni commit wskazanej; `cherry-pick id: "…"` dodaje kopię commitu z innej gałęzi, ze znacznikiem `cherry-pick:identyfikator`, jeśli nie podano własnego. Nazwa gałęzi może być w cudzysłowach. |
| GIT-003 | Rysunek: gałęzie są poziomymi torami w kolejności powstania, z nazwą z lewej; commity stoją od lewej do prawej w kolejności źródła, każdy na torze swojej gałęzi; odejście gałęzi, scalenie i kopia (linią przerywaną) są łącznikami między torami. Identyfikator stoi pod commitem, znacznik nad nim; commit wyróżniony jest kwadratem, odwrócony ma krzyżyk, scalający i kopia są puste w środku. |
| GIT-004 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nierozpoznana linia, gałąź już istniejąca albo nieznana, scalenie gałęzi bieżącej albo pustej, powtórzony identyfikator, nieznany rodzaj commitu, kopia nieznanego commitu albo z tej samej gałęzi, przekroczenie limitów (300 commitów, 24 gałęzie), brak commitów. |
| GIT-005 | `gitGraph TB:` i `gitGraph BT:` dają graf w pionie: gałęzie są kolumnami w kolejności powstania, z nazwą u góry (TB) albo u dołu (BT); commity biegną w dół albo w górę w kolejności ze źródła. Identyfikator i znacznik stoją z prawej strony commitu, a odstęp kolumn je mieści. Rozgałęzienia, scalenia i kopie są rysowane tak jak w poziomie. |
