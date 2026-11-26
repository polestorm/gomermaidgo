# Wymagania: drzewo katalogów (treeView)


| Id | Wymaganie |
|---|---|
| TREE-001 | Źródło zaczynające się od `treeView-beta` albo `treeView` daje drzewo katalogów. Komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| TREE-002 | Poziom wyznacza wcięcie (tabulator liczy się jak cztery spacje) albo znaki ramek: `├──`, `└──`, `│` i ich ciężkie odmiany `┣━━`, `┗━━`, `┃`. Kolumną linii zwykłej jest jej pierwszy znak, kolumną linii z gałęzią znak gałęzi przesunięty o jeden; węzeł należy do najbliższej wcześniejszej linii w mniejszej kolumnie. Dzięki temu wynik polecenia `tree` (korzeń `.` i gałęzie od kolumny zero) rozbiera się wprost, a gałęzie bez korzenia wiszą na wspólnym pniu. |
| TREE-003 | Nazwa jest zwykła albo w cudzysłowie; ukośnik na końcu (także po cudzysłowie) oznacza katalog, a węzeł z zawartością jest katalogiem także bez ukośnika. Po nazwie, w dowolnej kolejności: `:::klasa` (wbudowana `highlight` wyróżnia wpis, inne klasy są przyjmowane i nie zmieniają rysunku), `icon(nazwa)`; na końcu `## opis`. |
| TREE-004 | Rysunek: jeden rząd na wpis, wcięcie na poziom, kreska od pnia rodzica do każdego dziecka. Katalogi są pogrubione i zachowują zapisany ukośnik; opisy stoją pochylone w jednej kolumnie za najdłuższą nazwą z opisem. `icon(folder)` i `icon(file)` rysują własne ikony; ikona z zestawu (`icon(pakiet:nazwa)`) rysuje ikonę według rodzaju wpisu, `icon(none)` żadnej. Domyślnie rysunek jest neutralny; `WithColors` barwi ikony i wyróżnienie. |
| TREE-005 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, drzewo bez wpisów, wpis bez nazwy, niezamknięty cudzysłów albo `icon(`, pusta klasa, nieznany tekst po nazwie, przekroczenie limitów (2000 wpisów, 32 poziomy). |
