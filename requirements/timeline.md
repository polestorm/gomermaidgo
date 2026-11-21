# Wymagania: oś czasu


| Id | Wymaganie |
|---|---|
| TML-001 | Źródło zaczynające się od `timeline` daje oś czasu; `title Tekst` nadaje tytuł; komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. Kierunek po nagłówku (`TD`, `LR`) jest przyjmowany i pomijany. |
| TML-002 | Okres to `nazwa : zdarzenie : zdarzenie`; linia zaczynająca się od dwukropka dopisuje zdarzenia poprzedniemu okresowi; okres może nie mieć zdarzeń; `<br/>` łamie tekst. Każdy tekst ze źródła jest na rysunku. |
| TML-003 | `section Tytuł` grupuje następne okresy; tytuł sekcji stoi w nagłówku nad jej okresami. |
| TML-004 | Układ domyślny, jak w Mermaid: okresy stoją w rzędzie od lewej do prawej, w kolumnach o wspólnej szerokości, pod nimi oś ze strzałką, a pod osią zdarzenia każdego okresu jedno pod drugim. Długie teksty są łamane na granicach wyrazów. Żadne dwa pola się nie nakładają. |
| TML-005 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, zdarzenia bez okresu, sekcja bez tytułu, przekroczenie limitów (100 okresów, 20 zdarzeń w okresie), brak okresów. |
| TML-006 | Komentarz `%% layout: list` wybiera układ z osią z góry na dół: okresy stoją przy osi w kolumnie o wspólnej szerokości, a zdarzenia okresu po prawej, zawijane do kolejnych rzędów w szerokości strony. Bez dyrektywy to samo źródło rysuje układ domyślny. |
