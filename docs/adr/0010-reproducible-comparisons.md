# 0010 Porównania z innymi rendererami da się powtórzyć z repozytorium

Status: przyjęta · 2026-11-26

## Kontekst

README porównuje bibliotekę z `mmdg` i `go-mermaid`: czas, pamięć, rozmiar wyniku, pokrycie publicznych diagramów.
Liczby pochodziły ze skryptu w katalogu tymczasowym. Także lista źródeł 175 publicznych diagramów leżała tylko tam
i część listy przepadła przy sprzątaniu. Nikt z zewnątrz nie mógł tych liczb sprawdzić, a my sami nie
umielibyśmy ich odtworzyć. Właściciel zdecydował, że przy każdym porównaniu, zwłaszcza wygranym, wskazujemy wsad
i metodę, a test ma być powtarzalny dla konkurencji.

## Decyzja

1. Pomiar wykonuje `tools/bench`, narzędzie w repozytorium.
   - Uruchamia każdy program jako osobny proces, z poleceniami zapisanymi w kodzie.
   - Zapisuje sumy SHA-256 i wersje programów oraz opis maszyny.
   - Czas mierzy jako najlepszy z trzech przebiegów na plik, a w wyniku podaje medianę po plikach.
   - Pamięć mierzy jako szczytową pamięć rezydentną procesu (`getrusage`).
   - Sprawdza, czy wynik jest identyczny co do bajta między przebiegami.
2. Publiczne diagramy opisuje manifest `testdata/wild/sources.tsv`: repozytorium, przypięty commit, ścieżka, linia
   bloku i SHA-256. Pobiera je i sprawdza `tools/wildcorpus`. Samych diagramów nie przechowujemy, bo są cudze.
3. Surowy wynik, z którego wzięto liczby do README, leży w `docs/benchmarks/` z datą. Metodę i polecenia do
   powtórzenia opisuje `docs/benchmarks.md`, po angielsku, bo odsyła do niego README.
4. Liczba porównania bez takiego źródła nie trafia do README.

## Skutki

- Odtworzona lista źródeł ma 168 bloków. Siedmiu bloków z czterech plików nie ma już na GitHubie; wszystkie trzy
  programy je odrzucały, więc liczby pokrycia się nie zmieniły (139 / 138 / 137 / 123).
- Powtórny pomiar z 2026-11-26 zmienił kilka liczb w README: na przykład `go-mermaid` zajmował 4,9 ms na SVG zamiast
  3,9 ms. Liczbę zależności liczymy teraz z `go version -m`, więc dla `go-mermaid` wynosi 5 zamiast 10.
- Pomiar trzeba powtarzać tym samym narzędziem przy każdej zmianie liczb.
