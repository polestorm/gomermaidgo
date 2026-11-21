# Rodowód kodu

Stan na 2026-11-21.

## Źródła, z których wolno korzystać

| Źródło | Do czego |
|---|---|
| publiczna dokumentacja składni Mermaid (mermaid.js.org) | co znaczy dany zapis; lista elementów profilu |
| specyfikacja SVG 1.1 (W3C) | elementy i atrybuty wyniku |
| specyfikacja OpenType (tabele `head`, `hhea`, `maxp`, `hmtx`, `cmap`) | czytnik szerokości znaków w `internal/font` |
| publikacje o układzie warstwowym grafów: Gansner, Koutsofios, North, Vo, „A Technique for Drawing Directed Graphs” (1993); Brandes, Köpf, „Fast and Simple Horizontal Coordinate Assignment” (2001) | układ schematu blokowego i stanów (jeszcze nie napisany) |
| wynik cudzych rendererów uruchamianych jako osobny program | porównanie treści rysunku (`tools/refcompare`) |

## Czego nie wolno

Czytać, kopiować ani przenosić na Go kodu innych rendererów Mermaid i bibliotek układu grafów. Przeniesienie cudzego
kodu jest utworem zależnym i wnosi jego licencję (w tym LGPL i EPL w istniejących rozwiązaniach).

## Stan faktyczny

- Cały kod repozytorium napisano od zera; nie ma w nim plików stron trzecich ani fontów. Font używany w teście czytnika
  jest składany w kodzie testu.
- Wygląd (kolory, wymiary, kształt ludzika, zakładka ramki) jest własny. Z konwencji Mermaid pochodzi tylko to, co jest
  znaczeniem zapisu: rodzaje strzałek, nazwa rodzaju bloku w zakładce ramki, tytuł sekcji w nawiasach kwadratowych,
  powtórzenie uczestników na dole.
- Zastrzeżenie: przed powstaniem repozytorium, w sesji rozpoznania, przeczytano fragmenty projektu `go-mermaid`
  (README, opis architektury, funkcja szerokości tekstu) oraz dokumentację i audyt licencji projektu `mmdg`. Nie czytano
  ich parserów, układu diagramu sekwencji ani kodu rysującego. Czy to wymaga formalnego rozdzielenia sesji, jest
  pytaniem otwartym w ADR produktu, który zamawia ten klocek.
- Licencja repozytorium nie jest jeszcze ustalona; repozytorium jest lokalne.
