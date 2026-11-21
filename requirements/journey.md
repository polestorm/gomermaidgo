# Wymagania: podróż użytkownika


| Id | Wymaganie |
|---|---|
| JRN-001 | Źródło zaczynające się od `journey` daje diagram; `title Tekst` nadaje tytuł; komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| JRN-002 | Krok to `nazwa: ocena: uczestnik, uczestnik`; ocena jest liczbą całkowitą od 1 do 5, uczestnicy są opcjonalni. Kroki stoją z góry na dół w kolejności ze źródła: nazwa, znacznik oceny na skali od 1 do 5 i uczestnicy; linia łączy oceny kolejnych kroków. Ocena jest czytelna bez koloru. |
| JRN-003 | `section Tytuł` grupuje następne kroki; tytuł sekcji stoi w pasku nad jej krokami. |
| JRN-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, linia niebędąca krokiem, ocena spoza zakresu, sekcja bez tytułu, więcej niż 100 kroków, brak kroków. |
