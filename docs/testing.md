# Testy i porównania

## Warstwy

| Warstwa | Gdzie | Co sprawdza |
|---|---|---|
| korpus | `testdata/corpus/<typ>/*.mmd` | po pliku na grupę elementów profilu; polskie znaki w każdym; linia `%% REQ:` wskazuje wymagania |
| pliki wzorcowe | `testdata/golden/<typ>/*.svg` | wynik co do bajta; zmiana przez `go test . -update` i przegląd różnicy |
| poprawność SVG | `TestCorpusValidSVG` | poprawny XML, przestrzeń nazw SVG, tylko dozwolone elementy i atrybuty, liczby w atrybutach geometrii |
| treść | `TestCorpusLabelsPresent` | każdy tekst ze źródła jest w SVG; etykiety uczestników dwa razy (góra i dół) |
| niezmienniki układu | `internal/sequence/layout_test.go` | teksty nie nachodzą na siebie; nagłówki uczestników nie kolidują ze sobą ani z etykietami |
| błędy | `TestParseErrors` | każdy błąd ma numer linii i kod |
| powtarzalność | `TestCorpusDeterministic` | kilka renderowań daje ten sam wynik |
| odporność | `FuzzRender` | dowolne wejście nie powoduje paniki, a wynik jest poprawnym XML |
| pokrycie wymagań | `TestRequirementsCovered` | każde wymaganie ma test albo plik korpusu; każde odwołanie wskazuje istniejące wymaganie |

## Porównanie z rendererem odniesienia

`tools/refcompare` uruchamia cudzy renderer jako osobny program i porównuje **treść** obu rysunków: zbiór tekstów
z krotnością. Nie porównuje pikseli ani położeń, bo wygląd jest własny.

```sh
go run ./tools/refcompare -ref 'mmdg -i {in} -o {out} --no-font-embed' testdata/corpus/sequence/*.mmd
```

Wynik z 2026-11-21, odniesienie: `mmdg` v0.13.0 (port mermaid 12.2.0), uruchomiony jako program:

| Plik | Teksty u nas | Teksty w odniesieniu | Różnice |
|---|---:|---:|---|
| 001-basic-messages | 6 | 6 | brak |
| 002-participants-actors | 13 | 13 | brak |
| 003-arrow-types | 15 | 15 | brak |
| 004-notes | 14 | 14 | brak |
| 005-activations | 15 | 15 | brak |
| 006-blocks | 24 | 24 | brak |
| 007-nested-blocks | 24 | 24 | brak |
| 008-autonumber | 17 | 17 | brak |
| 009-self-and-long-labels | 12 | 12 | brak |
| 010-polish-pangram | 8 | 8 | brak |

Zgodne 10 z 10. Wyników renderera odniesienia nie przechowujemy w repozytorium.

## Przejście przez gopdfgo

2026-11-21: cztery diagramy korpusu (aktywacje, bloki, bloki zagnieżdżone, numeracja), renderowane z fontem Noto Sans,
osadzone w HTML i zamienione na PDF przez `gopdfgo` (webrender 0.1.0-dev): 4 strony, 25 kB, układ i polskie znaki
zgodne z podglądem w przeglądarce. Próba ręczna; automatyczny test wymaga decyzji, po której stronie ma żyć.
