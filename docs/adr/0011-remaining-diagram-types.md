# 0011 Dziewięć brakujących typów diagramów rysujemy wszystkie, po kolei

Status: przyjęta · 2026-11-26

## Kontekst

Mermaid 12.2 rozpoznaje 33 typy diagramów, biblioteka rysuje 24. `mmdg` rysuje wszystkie, bo uruchamia sam
Mermaid, a `go-mermaid` 20. Brakuje: `zenuml`, `venn-beta`, `treeView-beta`, `wardley-beta`, `cynefin-beta`,
`railroad-beta` (cztery zapisy), `eventmodeling`, `usecase-beta` i `agentflow`.

Dokumentacja Mermaid (12.3.0, odczyt 2026-11-26) podaje:

| Typ | Słowo kluczowe | Od wersji | Beta |
|---|---|---|---|
| Venn | `venn-beta` | 11.12.3 (dziennik zmian: 11.13.0) | tak |
| TreeView | `treeView-beta` | 11.14.0 | tak |
| Wardley | `wardley-beta` | 11.14.0 | tak |
| Event Modeling | `eventmodeling` | 11.15.0 | nie oznaczony |
| Cynefin | `cynefin-beta` | 11.16.0 | tak |
| Railroad | `railroad-beta`, `railroad-ebnf-beta`, `railroad-abnf-beta`, `railroad-peg-beta` | 11.16.0 | tak |
| Use Case | `usecase-beta` | 12.0.0 | tak |
| Agentflow | `agentflow-beta`, od 12.3.0 także `agentflow` | 12.0.0 | tak (dziennik zmian) |
| ZenUML | `zenuml` | osobny pakiet `@mermaid-js/mermaid-zenuml` | nie oznaczony |

Wcześniejszy przypis w README mówił, że osiem z nich dodano w Mermaid 12 jako beta. To było błędne: większość
pochodzi z wersji 11.12–11.16, z tego samego okresu co rysowane już `ishikawa-beta` (11.13) i `swimlane-beta`
(11.16). Wśród 168 publicznych diagramów (`testdata/wild/sources.tsv`) nie ma żadnego z tych typów, ale zbiór
zebrano wyszukiwaniem, które ich nazw nie zawierało.

## Decyzja

1. Rysujemy wszystkie dziewięć typów, jeden po drugim, bez pośpiechu. Każdy typ to osobna runda: wymagania
   w `requirements/`, pliki korpusu z `%% REQ:`, wpis w `docs/profile.md`, obrazek w galerii i przegląd
   właściciela na stronie porównania z `mmdg`. Typy powstają jeden po drugim bez czekania na przegląd:
   właściciel ocenia je w wolnej chwili, a poprawki z przeglądu robimy w osobnych rundach (decyzja
   właściciela z 2026-11-26).
2. Kolejność, od najtańszych: `treeView`, `venn`, `cynefin`, `usecase`, `zenuml`, `wardley`, `eventmodeling`,
   `agentflow`, `railroad`. Najpierw te, które wykorzystują istniejące części (drzewo, układ warstwowy, diagram
   sekwencji, plansza o stałym układzie).
3. Wzorcem jest dokumentacja składni Mermaid; wygląd jest własny, jak w pozostałych typach. `mmdg` uruchamiamy
   wyłącznie jako program do porównania (`docs/provenance.md`).
4. Przykłady do przeglądu leżą w `testdata/planned/<typ>/`. Po wdrożeniu typu przenosimy je do
   `testdata/corpus/<typ>/` z liniami `%% REQ:`. Stronę przeglądu typu tworzy
   `go run ./tools/review -dir testdata/planned/<typ> -kind <typ> -ref 'mmdg -q -i {in} -o {out}' -o out/review-<typ>.html`.
5. Typ oznaczony w Mermaid jako beta przyjmujemy pod obiema nazwami, z przyrostkiem `-beta` i bez, tak jak
   `swimlane-beta`. Gdy Mermaid zmieni składnię typu beta, poprawiamy parser i korpus w osobnej rundzie.

## Skutki

- 2026-11-26: wszystkie dziewięć typów wdrożone; biblioteka rysuje 33 z 33 typów Mermaid 12.2. Przykłady przeniesiono
  z `testdata/planned/` do `testdata/corpus/<typ>/`, a strony przeglądu buduje
  `go run ./tools/review -dir testdata/corpus/<typ> -kind <typ> -ref 'mmdg -q -i {in} -o {out}' -o out/review-<typ>.html`.

- Wiersz „Diagram types” w README rośnie z 24 do 33 w miarę wdrażania; liczbę zmieniamy po każdym typie.
- Przypis `[^types]` w README poprawiono: podaje wersje z tabeli wyżej.
- Ryzyko: składnia typów beta może się zmienić; koszt to poprawka parsera, a nie układu.
