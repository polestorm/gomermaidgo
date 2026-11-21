# Wymagania: tablica kanban


| Id | Wymaganie |
|---|---|
| KBN-001 | Źródło zaczynające się od `kanban` daje tablicę; wcięcie wyznacza poziom: najpłytsze linie są kolumnami, głębsze kartami ostatniej kolumny, w kolejności ze źródła. Komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| KBN-002 | Tekst kolumny i karty to `id[Tekst]`, `[Tekst]` albo sam tekst; `<br/>` łamie linię, długi tekst jest łamany w szerokości karty. Blok `@{ ticket: …, assigned: …, priority: … }` za kartą daje jej numer zgłoszenia, osobę i priorytet; inne klucze są pomijane. |
| KBN-003 | Rysunek: kolumny obok siebie o wspólnej szerokości i wysokości, tytuł u góry, karty jedna pod drugą; dane karty stoją pod jej tekstem, a priorytet widać po grubości paska przy lewej krawędzi karty, bez koloru. Karty się nie nakładają i mieszczą się w swojej kolumnie. |
| KBN-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, kolumna albo karta bez tekstu, przekroczenie limitów (12 kolumn, 200 kart), brak kolumn. |
