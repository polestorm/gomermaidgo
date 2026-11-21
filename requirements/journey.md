# Wymagania: podróż użytkownika


| Id | Wymaganie |
|---|---|
| JRN-001 | Źródło zaczynające się od `journey` daje diagram; `title Tekst` nadaje tytuł; komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| JRN-002 | Krok to `nazwa: ocena: uczestnik, uczestnik`; ocena jest liczbą całkowitą od 1 do 5, uczestnicy są opcjonalni. Układ domyślny, jak w Mermaid: kroki stoją w rzędzie w kolejności ze źródła, pod każdym buźka na wysokości oceny (uśmiech dla 4 i 5, usta proste dla 3, smutek dla 1 i 2), u dołu oś ze strzałką; uczestnicy mają legendę z lewej i znaczniki na swoich krokach. |
| JRN-003 | `section Tytuł` grupuje następne kroki; tytuł sekcji stoi w nagłówku nad jej krokami. |
| JRN-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, linia niebędąca krokiem, ocena spoza zakresu, sekcja bez tytułu, więcej niż 100 kroków, brak kroków. |
| JRN-005 | Komentarz `%% layout: list` wybiera układ listowy: kroki stoją z góry na dół, każdy z nazwą, znacznikiem oceny na skali od 1 do 5 i uczestnikami; linia łączy oceny kolejnych kroków; ocena jest czytelna bez koloru. Bez dyrektywy to samo źródło rysuje układ domyślny. |
