# Rodowód kodu

Stan na 2026-11-23.

## Źródła, z których wolno korzystać

| Źródło | Do czego |
|---|---|
| publiczna dokumentacja składni Mermaid (mermaid.js.org) | co znaczy dany zapis; lista elementów profilu |
| specyfikacja SVG 1.1 (W3C) | elementy i atrybuty wyniku |
| specyfikacja OpenType (tabele `head`, `hhea`, `maxp`, `hmtx`, `cmap`, `glyf`, `loca`, `OS/2`, `post`, `name`) | czytnik szerokości znaków w `internal/font`, kontury glifów w `internal/glyf` (PNG, emoji jako kontury w SVG), podzbiór kroju w PDF, wersja statyczna kroju w `tools/staticfont` |
| specyfikacja PDF (ISO 32000-1) | zapis PDF w `pdf/`; kompresja strumieni przez `compress/zlib` z biblioteki standardowej |
| opisy rasteryzacji z dokładnym pokryciem piksela (sumowanie pól ze znakiem wzdłuż linii obrazu) | rasteryzator w `internal/raster`; plik PNG zapisuje `image/png` z biblioteki standardowej |
| CSS Color Module Level 4 (W3C) | nazwy kolorów i zapisy `#rgb`, `#rrggbb`, `rgb()` w `internal/csscolor` |
| publikacje o układzie warstwowym grafów: Sugiyama, Tagawa, Toda, „Methods for Visual Understanding of Hierarchical System Structures” (1981); Gansner, Koutsofios, North, Vo, „A Technique for Drawing Directed Graphs” (1993); Brandes, Köpf, „Fast and Simple Horizontal Coordinate Assignment” (2001) | układ schematu blokowego i stanów (`internal/layered`): z dwóch pierwszych pochodzą kroki, metoda barycentryczna, zamiana sąsiadów, wagi krawędzi i metoda priorytetowa; trzecia jest planowana |
| wynik cudzych rendererów uruchamianych jako osobny program | porównanie treści rysunku (`tools/refcompare`) |

## Czego nie wolno

Czytać, kopiować ani przenosić na Go kodu innych rendererów Mermaid i bibliotek układu grafów. Przeniesienie cudzego
kodu jest utworem zależnym i wnosi jego licencję (w tym LGPL i EPL w istniejących rozwiązaniach).

## Stan faktyczny

- Cały kod repozytorium napisano od zera. Jedyne pliki stron trzecich to dołączone kroje, wymienione w `NOTICE`:
  Arimo (The Arimo Project Authors) i statyczna wersja Noto Emoji (Google LLC), oba na licencji SIL Open Font License
  1.1, z tekstami licencji w `fonts/`. Noto Emoji zmodyfikowano narzędziem `tools/staticfont` (usunięte tablice
  wariantów i tablice zbędne do rysowania konturów). Font używany w teście czytnika jest składany w kodzie testu.
- Układ warstwowy napisano z pamięci opisów algorytmów w wymienionych publikacjach, bez otwierania cudzych
  implementacji; rekurencyjne rozmieszczanie podgrafów, rozsuwanie punktów zaczepienia i wachlarz wokół wspólnego
  sąsiada są rozwiązaniami własnymi.
- Wygląd (kolory, paleta trybu kolorowego, wymiary, kształt ludzika, zakładka ramki) jest własny. Po porównaniu
  rysunków z rendererem odniesienia przyjęto trzy cechy widoczne na jego wyniku (nie w kodzie): większe odstępy
  względem pisma, kropkowane ramki bloków i krzywą dla komunikatu do siebie. Z konwencji Mermaid pochodzi tylko to, co jest
  znaczeniem zapisu: rodzaje strzałek, nazwa rodzaju bloku w zakładce ramki, tytuł sekcji w nawiasach kwadratowych,
  powtórzenie uczestników na dole.
- Zastrzeżenie: przed powstaniem repozytorium, w sesji rozpoznania, przeczytano fragmenty projektu `go-mermaid`
  (README, opis architektury, funkcja szerokości tekstu) oraz dokumentację i audyt licencji projektu `mmdg`. Nie czytano
  ich parserów, układu diagramu sekwencji ani kodu rysującego. Szerokość tekstu liczymy z tablic własnego kroju
  (`hmtx`), nie z tabeli szerokości, jakiej używa `go-mermaid`.
- Do porównania w README (2026-11-23) `go-mermaid` 0.1.6 uruchamiano wyłącznie jako program; przeczytano jego README,
  `go.mod` i listę importowanych modułów (`go list -deps`), bez kodu źródłowego.
- Licencja: Apache 2.0 (`LICENSE`, `NOTICE`).
