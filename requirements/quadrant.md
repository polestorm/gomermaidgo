# Wymagania: wykres ćwiartkowy


| Id | Wymaganie |
|---|---|
| QDR-001 | Źródło zaczynające się od `quadrantChart` daje wykres; `title` nadaje tytuł; `x-axis lewy --> prawy` i `y-axis dolny --> górny` opisują końce osi (drugi opis jest opcjonalny); `quadrant-1` do `quadrant-4` nadają tytuły ćwiartkom: prawej górnej, lewej górnej, lewej dolnej, prawej dolnej. Komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| QDR-002 | Punkt to `Nazwa: [x, y]` ze współrzędnymi od 0 do 1; x rośnie w prawo, y w górę. Wygląd punktu podany po nawiasie (`radius`, `color`, `stroke-color`, `stroke-width`) albo w klasie (`classDef nazwa …`, `Nazwa:::nazwa`) jest rysowany; własność przy punkcie ma pierwszeństwo przed klasą, a własność nieznana albo o złej wartości jest pomijana. |
| QDR-003 | Rysunek: kwadrat podzielony na cztery ćwiartki, każdy punkt w swoim miejscu. Nazwa punktu jest pisana mniejszym pismem (0,75 wielkości podstawowej), bez podkładu, i stoi pod kropką; gdy nachodziłaby tam na tytuł ćwiartki, inną kropkę albo nazwę albo wyszła poza wykres, staje nad kropką, z prawej albo z lewej. Tytuł ćwiartki stoi u jej góry; gdy leży tam kropka, zaczyna się tuż pod nią. Opisy osi poziomej stoją pod wykresem, a osi pionowej z lewej strony, obrócone i czytane z dołu do góry, każdy przy swojej połowie. |
| QDR-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nierozpoznana linia, współrzędna spoza zakresu 0–1 albo niebędąca liczbą, oś bez opisu, więcej niż 100 punktów. |
