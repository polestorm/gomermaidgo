# Wymagania: wynik PDF


| Id | Wymaganie |
|---|---|
| PDF-001 | `pdf.Render` zamienia źródło diagramu każdego obsługiwanego typu na jednostronicowy dokument PDF, z tymi samymi opcjami co `mermaid.Render` i tymi samymi błędami składni. Dokument jest poprawnie zbudowany (nagłówek, obiekty, tabela odsyłaczy wskazująca początki obiektów), a wynik jest powtarzalny co do bajta: bez dat i losowych identyfikatorów. |
| PDF-002 | Tekst rysunku jest w PDF tekstem: te same napisy co w SVG, w tej samej kolejności, zapisane krojem osadzonym w pliku, z tabelą ToUnicode pozwalającą je zaznaczać i wyszukiwać. Krój jest osadzony jako podzbiór: kontury tylko użytych glifów i ich składników, numery glifów bez zmian, bez tabel niepotrzebnych do rysowania. Ten sam plik kroju podany dla kilku odmian jest osadzany raz. Napis ze znakiem, którego krój nie ma (rysowanym glifem braku znaku), niesie swój tekst wprost (`ActualText`), więc kopiowanie i wyszukiwanie zwracają znak ze źródła. |
| PDF-003 | Rysunek pochodzi z tej samej sceny co SVG: strona ma rozmiar rysunku (piksel to 0,75 punktu), kształty mają te same położenia, kolory (`#rgb`, `#rrggbb`, `rgb()`, nazwy CSS), grubości i wzory linii, a wypełnienie półprzezroczyste zostaje półprzezroczyste. Tytuł i opis dostępności trafiają do informacji o dokumencie. |
| PDF-004 | Format wymaga pliku kroju: dołączonego albo podanego w `WithFont`. Rysunek mierzony funkcją z `WithMeasure` albo krojem znanym tylko z nazwy daje błąd `ErrFontRequired`, a krój bez konturów TrueType błąd `ErrOutlines`. Odmiana pogrubiona albo pochylona niepodana przez wywołującego korzysta z kroju zwykłego. |
| PDF-005 | Format jest opcjonalny: żyje w osobnym pakiecie `pdf`, którego program nieimportujący nie zawiera; program z linii poleceń wybiera go rozszerzeniem `.pdf` pliku wyniku i daje się zbudować bez niego znacznikiem `nopdf`. |
