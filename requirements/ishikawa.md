# Wymagania: diagram Ishikawy (przyczyn i skutków)


| Id | Wymaganie |
|---|---|
| ISH-001 | Źródło zaczynające się od `ishikawa-beta` albo `ishikawa` daje diagram przyczyn i skutków. Pierwsza linia po nagłówku to problem (skutek); następne to przyczyny. Komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| ISH-002 | Wcięcie wyznacza poziom: linie na wcięciu pierwszej przyczyny (albo płytszym) są kategoriami, głębsze są przyczynami najbliższej wcześniejszej linii o mniejszym wcięciu, najwyżej trzy poziomy pod kategorią. `<br>` łamie linię, długi tekst jest łamany. |
| ISH-003 | Rysunek: pozioma oś z grotem prowadzi do pola z problemem po prawej; kategorie stoją w kolejności źródła od lewej, na przemian nad osią i pod nią, każda w polu na końcu ukośnej ości. Przyczyny to poziome kreski przy ości z tekstem nad kreską, przyczyny szczegółowe wiszą na pionowym odgałęzieniu pod swoją przyczyną. Teksty się nie nakładają; domyślnie rysunek jest neutralny, `WithColors` nadaje kategoriom tony palety. |
| ISH-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, brak problemu albo kategorii, linia bez tekstu, zbyt głębokie zagnieżdżenie, przekroczenie limitów (12 kategorii, 200 przyczyn). |
