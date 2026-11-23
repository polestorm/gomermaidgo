# Testy i porównania

## Warstwy

| Warstwa | Gdzie | Co sprawdza |
|---|---|---|
| korpus | `testdata/corpus/<typ>/*.mmd` | po pliku na grupę elementów profilu; polskie znaki w każdym; linia `%% REQ:` wskazuje wymagania |
| pliki wzorcowe | `testdata/golden/<typ>/*.svg` | wynik co do bajta; zmiana przez `go test ./mermaid -update` i przegląd różnicy |
| pliki wzorcowe modelu | `testdata/golden/<typ>/*.txt` | dla typów grafowych: model grafu z parsera (`Graph.Dump`), obok pliku SVG |
| niezmienniki układu grafu | `internal/layered/layout_test.go` | cały korpus w czterech kierunkach: węzły nie nachodzą na siebie, ramki rozłączne albo zagnieżdżone, etykiety krawędzi nie kolidują, każda krawędź ma ścieżkę i nie przechodzi przez obcy węzeł; cykle i pętle; graf na granicy limitów |
| reguły wyglądu | `internal/layered/appearance_test.go` | reguły z przeglądu rysunków (`requirements/layout.md`): odcinki krawędzi równoległe do osi, bez cofania i drobnych uskoków; mostek w każdym przecięciu, także między poziomami podgrafów; równe pola w ciągu; wspólna wysokość skrętu gałęzi; paski stanu złożonego i krótsze strzałki w jego wnętrzu; oś pola z pętlą własną; pakowanie ramek bez połączeń; kolumny w torach |
| wynik PDF | `pdf/pdf_test.go` | cały korpus: budowa dokumentu i tabela odsyłaczy, rozmiar strony równy rysunkowi SVG, teksty strony odczytane przez tabelę ToUnicode równe tekstom SVG w tej samej kolejności; podzbiór kroju (kontury tylko użytych glifów, suma kontrolna); powtarzalność; błędy przy braku pliku kroju; odporność (`FuzzRender`) |
| poprawność SVG | `TestCorpusValidSVG` | poprawny XML, przestrzeń nazw SVG, tylko dozwolone elementy i atrybuty, liczby w atrybutach geometrii |
| treść | `TestCorpusLabelsPresent` | każdy tekst ze źródła jest w SVG; etykiety uczestników dwa razy (góra i dół) |
| niezmienniki układu | `internal/sequence/layout_test.go` | teksty nie nachodzą na siebie; nagłówki uczestników nie kolidują ze sobą ani z etykietami |
| błędy | `TestParseErrors` | każdy błąd ma numer linii i kod |
| powtarzalność | `TestCorpusDeterministic` | kilka renderowań daje ten sam wynik |
| odporność | `FuzzRender` | dowolne wejście nie powoduje paniki, a wynik jest poprawnym XML |
| pokrycie wymagań | `TestRequirementsCovered` | każde wymaganie ma test albo plik korpusu; każde odwołanie wskazuje istniejące wymaganie |

## Przegląd przez człowieka

`tools/review` składa jedną stronę HTML (`out/review.html`, poza repozytorium): dla każdego pliku korpusu tytuł, opis,
wymagania, źródło, nasz rysunek i opcjonalnie rysunek renderera odniesienia; ocenę i uwagi można skopiować jako tekst.

```sh
go run ./tools/review -font NotoSans-Regular.ttf -family "Noto Sans" -ref 'mmdg -i {in} -o {out}' -o out/review.html
```

`tools/fontsheet` składa osobną stronę (`out/fonts.html`) porównującą kroje pisma: próbki tekstu i trzy diagramy
korpusu w każdym kroju, w dwóch rozmiarach pisma. Pliki fontów leżą poza repozytorium.

## Droga do PDF

`tools/pdfcheck` składa stronę HTML ze wszystkich diagramów korpusu, oddaje ją zewnętrznemu konwerterowi i sprawdza,
że powstał PDF, a każdy wyraz rysunków jest w nim tekstem (wyciąganym osobnym programem), tyle samo razy.

```sh
go run ./tools/pdfcheck -pdf 'gopdfgo -in {in} -out {out} -fonts {fonts}' -text 'pdftotext {in} -' testdata/corpus/*/*.mmd
```

Konwerter jest czarną skrzynką jak renderer odniesienia; nie jest zależnością modułu, więc sprawdzenie nie wchodzi
do `go test`. Wynik z 2026-11-21 (`gopdfgo` 0.1.0-dev, `pdftotext`): 43 diagramy, 30 stron, wszystkie 588 różnych
wyrazów jest w PDF tekstem, krój Recursive osadzony (także odmiana pogrubiona i pochylona).

Ustalenie z tego testu: rysunku nie da się podzielić między strony, więc diagram wyższy niż strona jest ucinany.
Strona musi ograniczać także wysokość rysunku (`max-height` razem z `width:auto; height:auto`); rysunek skaluje się
wtedy z zachowaniem proporcji.

Próba ręczna całej drogi z 2026-11-21: dokument Markdown z trzema blokami `mermaid` (jeden błędny) przez
`markdown.Embed`, parser `goldmark` z przepuszczaniem HTML (moduł próbny poza repozytorium) i `gopdfgo`: dwa rysunki
i komunikat błędu na stronie, akapity i lista wokół nienaruszone. Ta próba nie jest zautomatyzowana, bo parser Markdown
byłby zależnością.

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

Ponowny przebieg na całym korpusie, 2026-11-21 (po torach, mostkach i zmianach układu), to samo odniesienie:

| Korpus | Pliki | Zgodne | Różnice (wszystkie świadome) |
|---|---:|---:|---|
| `sequence` | 12 | 11 | `012-create-destroy`: uczestnik kończony ma u nas jedno pole, w odniesieniu dwa |
| `flowchart` | 18 | 17 | `009-text`: `<znaczniki>` |
| `state` | 8 | 7 | `006-document-lifecycle`: `<znaczniki>` |
| `gantt` | 3 | 1 | daty na osi: własny dobór podziałki |
| `er` | 2 | 1 | zapis kluczy „PK, FK” |

Razem 37 z 43. Pliki z torami zgadzają się, bo dyrektywa jest komentarzem i odniesienie rysuje zwykłe podgrafy.

Pierwsze porównanie parserów wykazało jeszcze dwie różnice, które okazały się naszymi błędnymi założeniami i zostały
poprawione: linia `%%` wewnątrz notatki wieloliniowej jest komentarzem (nie tekstem notatki), a `state Id` bez opisu nie
jest rysowane przez odniesienie i nie ma go w dokumentacji (teraz błąd `state-unsupported`).

## Stan po dodaniu wszystkich typów (2026-11-22)

Korpus ma 89 diagramów w 23 typach, a wymagań z testami jest 218.

- Porównanie treści z `mmdg`: 49 plików zgodnych, 18 różnych, 2 bez wyniku odniesienia. Różnice są opisane
  w `profile.md` przy każdym typie (podziałki osi, skala ocen w podróży, znaczniki priorytetu, numery pod mapą
  drzewa i podobne). Bez wyniku: `sankey/001` (Mermaid odrzuca w tym typie polskie litery) i `block/001`
  (odniesienie kończy się własnym błędem przy połączeniu do rombu).
- Droga HTML → `gopdfgo`: 69 diagramów, wszystkie 1058 różnych wyrazów jest w PDF tekstem.
- Bezpośredni PDF (`pdf.Render`): 69 plików, Ghostscript czyta wszystkie bez ostrzeżeń.

Kontrola drogi przez `gopdfgo` znalazła przy okazji jedną rzecz w rysunku: numery sąsiednich bitów w diagramie
pakietu stały tak blisko siebie, że wyciąganie tekstu sklejało je w jedną liczbę („7” i „8” w „78”). Odstęp został
zwiększony.

## Druga próba na publicznych diagramach (2026-11-22)

74 publiczne pliki Markdown wyszukane po nazwach nowszych typów: 81 bloków, z czego 78 się rysuje (przed poprawkami
74). Próba dała trzy poprawki: tekst krawędzi bez odstępów (`A --tekst--> B`), znaki o zerowej szerokości w źródle,
blok klasy w jednej linii. Trzy pozostałe bloki są błędne także dla Mermaid (węzeł bez identyfikatora, `subgraph`
w diagramie stanów, notatka bez cudzysłowów). Po poprawkach pierwsza próba (70 bloków) daje 66 narysowanych
zamiast 62.

Wyszukiwanie zwraca głównie diagramy klas i schematy blokowe; najnowszych typów (architektura, blok, pakiet, radar)
w publicznych plikach prawie nie ma, więc ich składnia pozostaje sprawdzona tylko na własnych przykładach.

## Wynik PDF w cudzych czytnikach

2026-11-21: 57 plików korpusu zapisanych przez `pdf.Render`. `pdftotext` (poppler) wyciąga z każdego te same wyrazy,
co są w SVG (1295 wyrazów); `pdffonts` pokazuje kroje jako osadzone podzbiory z tabelą Unicode; Ghostscript czyta
wszystkie pliki bez ostrzeżeń i błędów. Pliki mają średnio 15 kB (cały korpus 872 kB), z czego większość to
podzbiory krojów. Sprawdzenie ręczne: czytniki są cudzymi programami i nie wchodzą do `go test`.

## Próba na cudzych diagramach

2026-11-21: 150 plików Markdown z publicznych repozytoriów GitHuba (wyszukiwanie bloków `mermaid`), 70 bloków. Pliki
są danymi roboczymi i nie trafiają do repozytorium. Narysowane 62 (schemat blokowy 53, sekwencja 4, stany 3, encje 2);
4 bloki to typy spoza profilu (`journey`, `pie`), 4 mają `...` zamiast treści i nie są poprawnym Mermaid. Treść
wszystkich 62 rysunków zgadza się z odniesieniem po dodaniu znaczników `<b>` i `<i>` (przedtem 61). W 28 diagramach
są znaki, których krój Recursive nie ma (emoji). Rysunki są przeciętnie o 15% szersze niż w odniesieniu i o 4% niższe.
Próba jest mała i przechylona ku prostym schematom blokowym.

Szeroki diagram encji (35 encji, 36 relacji, `internal/layered/testdata/er/wide-er.mmd`; liczby poniżej zmierzono na
wcześniejszej treści o tej samej budowie) pokazał pustą
kolumnę szerokości 1744 px; po jej zamknięciu (LAY-010) rysunek ma 2607 px zamiast 4197. Odniesienie: 2551 px. Z limitem
szerokości warstwy (LAY-011) ma 1769 × 1628 px i mieści się na stronie A4 w skali 0,37 zamiast 0,24.

## Przejście przez gopdfgo

2026-11-21: cztery diagramy korpusu (aktywacje, bloki, bloki zagnieżdżone, numeracja), renderowane z fontem Noto Sans,
osadzone w HTML i zamienione na PDF przez `gopdfgo` (webrender 0.1.0-dev): 4 strony, 25 kB, układ i polskie znaki
zgodne z podglądem w przeglądarce. Próba ręczna; automatyczny test wymaga decyzji, po której stronie ma żyć.

2026-11-21, grafy: sześć diagramów (proces z podgrafem, podgrafy zagnieżdżone, wszystkie kształty węzłów, wybór
z rozwidleniem i złączeniem, stany złożone, cykl życia dokumentu w układzie poziomym), te same ustawienia: 4 strony,
23 kB; krzywe (`path` z poleceniami M, L, C), elipsy walca, groty i etykiety zgodne z podglądem w przeglądarce.
