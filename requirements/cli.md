# Wymagania programu z linii poleceń


| Id | Wymaganie |
|---|---|
| CLI-001 | Wejście jest plikiem podanym jako argument albo flagą `-i`/`--input`; bez pliku albo dla `-` czytane jest standardowe wejście. Wynik trafia do pliku z `-o`/`--output`; bez niego na standardowe wyjście, a przy wejściu z `-i` do pliku o nazwie wejścia z dopisanym rozszerzeniem formatu. Format wybiera `-e`/`--outputFormat` albo rozszerzenie wyniku; format nieobsługiwany, dwa wejścia i nieznana flaga są błędem. |
| CLI-002 | Flagi `mermaid-cli`: `-b`/`--backgroundColor` ustawia tło (`transparent` oznacza brak), `-q`/`--quiet` wyłącza uwagi i ostrzeżenia. Flagi bez odpowiednika (`-t`, `-w`, `-H`, `-s`, `-c`, `-C`, `-p`, `-I`, `-f` i ich długie nazwy) są przyjmowane i pomijane; o każdej poza `-f` program pisze uwagę na wyjściu błędów. |
| CLI-003 | Wejście Markdown (`.md`, `.markdown`): każdy blok `mermaid` trafia do pliku `wynik-N.format`, obok wyniku albo w katalogu z `-a`/`--artefacts`. Gdy wynik jest plikiem Markdown, dostaje dokument z blokami zamienionymi na `![diagram](./wynik-N.svg)`; pozostała treść zostaje bez zmian. Blok z błędem zostaje w dokumencie, błąd ma numer linii dokumentu, a program kończy się niepowodzeniem po przejściu całości. |
