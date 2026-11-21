# Wymagania: wykres kołowy


| Id | Wymaganie |
|---|---|
| PIE-001 | Źródło zaczynające się od `pie` daje wykres; w nagłówku mogą stać `showData` i `title Tekst`; `title` i `showData` mogą też stać w osobnych liniach; komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| PIE-002 | Wycinek to `"etykieta" : wartość`; wartość jest liczbą nieujemną, także ułamkową. Wycinki są rysowane w kolejności ze źródła, od godziny dwunastej zgodnie z ruchem wskazówek zegara, a ich kąty są proporcjonalne do wartości. |
| PIE-003 | Każda etykieta jest w legendzie obok koła, w kolejności wycinków, z próbką wypełnienia; wycinek o udziale co najmniej 4% ma na sobie udział procentowy; tytuł stoi nad wykresem. Wypełnienia sąsiednich wycinków różnią się jasnością, żeby wykres dało się czytać w druku czarno-białym. |
| PIE-004 | `showData` dopisuje w legendzie wartość ze źródła w nawiasach kwadratowych; wycinek o wartości zero jest tylko w legendzie. |
| PIE-005 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, linia niebędąca wycinkiem, wartość niebędąca liczbą albo ujemna, etykieta pusta albo powtórzona, więcej niż 24 wycinki, brak wycinka o wartości powyżej zera. |
