# Testy i porównania

## Warstwy

| Warstwa | Gdzie | Co sprawdza |
|---|---|---|
| korpus | `testdata/corpus/<typ>/*.mmd` | po pliku na grupę elementów profilu; polskie znaki w każdym; linia `%% REQ:` wskazuje wymagania |
| pliki wzorcowe | `testdata/golden/<typ>/*.svg` | wynik co do bajta; zmiana przez `go test . -update` i przegląd różnicy |
| pliki wzorcowe modelu | `testdata/golden/<typ>/*.txt` | dla typów grafowych: model grafu z parsera (`Graph.Dump`), obok pliku SVG |
| niezmienniki układu grafu | `internal/layered/layout_test.go` | cały korpus w czterech kierunkach: węzły nie nachodzą na siebie, ramki rozłączne albo zagnieżdżone, etykiety krawędzi nie kolidują, każda krawędź ma ścieżkę; cykle i pętle; graf na granicy limitów |
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

Schemat blokowy i diagram stanów, 2026-11-21, to samo odniesienie. Od etapu 3 porównywane są **wyrazy** z krotnością
(nie linie), bo renderery łamią długie etykiety w różnych miejscach:

| Korpus | Pliki | Zgodne | Różnice |
|---|---:|---:|---|
| `sequence` | 10 | 10 | brak |
| `flowchart` | 11 | 10 | `009-text`: tekst `<znaczniki>` odniesienie usuwa jako znacznik HTML, my zostawiamy (różnica świadoma, `profile.md`) |
| `state` | 6 | 5 | `006-document-lifecycle`: ta sama różnica |

Pierwsze porównanie parserów wykazało jeszcze dwie różnice, które okazały się naszymi błędnymi założeniami i zostały
poprawione: linia `%%` wewnątrz notatki wieloliniowej jest komentarzem (nie tekstem notatki), a `state Id` bez opisu nie
jest rysowane przez odniesienie i nie ma go w dokumentacji (teraz błąd `state-unsupported`).

## Przejście przez gopdfgo

2026-11-21: cztery diagramy korpusu (aktywacje, bloki, bloki zagnieżdżone, numeracja), renderowane z fontem Noto Sans,
osadzone w HTML i zamienione na PDF przez `gopdfgo` (webrender 0.1.0-dev): 4 strony, 25 kB, układ i polskie znaki
zgodne z podglądem w przeglądarce. Próba ręczna; automatyczny test wymaga decyzji, po której stronie ma żyć.

2026-11-21, grafy: sześć diagramów (proces z podgrafem, podgrafy zagnieżdżone, wszystkie kształty węzłów, wybór
z rozwidleniem i złączeniem, stany złożone, cykl życia dokumentu w układzie poziomym), te same ustawienia: 4 strony,
23 kB; krzywe (`path` z poleceniami M, L, C), elipsy walca, groty i etykiety zgodne z podglądem w przeglądarce.
