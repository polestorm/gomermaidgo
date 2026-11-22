# Wymagania: wynik PNG


| Id | Wymaganie |
|---|---|
| PNG-001 | `png.Render` zamienia źródło diagramu każdego obsługiwanego typu na obraz PNG, z tymi samymi opcjami co `gomermaidgo.Render` i tymi samymi błędami składni. Obraz ma wymiary rysunku SVG pomnożone przez skalę (zaokrąglone w górę): 2 w `Render`, dowolną dodatnią w `RenderScale`. Ten sam diagram daje zawsze ten sam plik. Skala niedodatnia jest błędem, a obraz ponad `MaxPixels` błędem `ErrTooLarge`. |
| PNG-002 | Rysunek pochodzi z tej samej sceny co SVG i PDF: kształty mają te same położenia, kolory, grubości i wzory linii, a wypełnienie półprzezroczyste jest półprzezroczyste. Tło jest przezroczyste, jeśli nie podano `WithBackground`. Krawędzie kształtów są wygładzone pokryciem liczonym z pola kształtu w pikselu. |
| PNG-003 | Tekst jest rysowany konturami glifów kroju, którym rysunek został zmierzony, także glifów złożonych (litery ze znakami diakrytycznymi) i odmian pogrubionej i pochylonej. Format wymaga pliku kroju z konturami TrueType: inaczej zwraca `ErrFontRequired` albo `ErrOutlines`, tak jak PDF (PDF-004). |
| PNG-004 | Format jest opcjonalny: żyje w osobnym pakiecie `png` bez zależności poza biblioteką standardową; program z linii poleceń wybiera go rozszerzeniem `.png` albo `-e png`, skalę ustawia `-s`/`--scale`, i daje się zbudować bez niego znacznikiem `nopng`. |
