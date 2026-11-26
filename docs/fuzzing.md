# Fuzzing

Cele: `FuzzRender` w trzech pakietach. Ziarna to cały korpus `testdata/corpus`.

| Cel | Pakiet | Sprawdza |
|---|---|---|
| `FuzzRender` | `mermaid` | dowolne źródło: brak paniki; `Check` przyjmuje każde źródło, które `Render` przyjmuje; SVG jest poprawnym XML |
| `FuzzRender` | `pdf` | to samo źródło daje SVG wtedy i tylko wtedy, gdy daje PDF; PDF jest kompletny (`%%EOF`) |
| `FuzzRender` | `png` | źródło do 2000 znaków daje poprawny obraz PNG albo błąd, nigdy panikę |

Długie kampanie biegną na Synology „StormSyn” (DS224+, x86_64, 4 rdzenie Celeron J4125, ok. 10 GB RAM) przez wspólną
kolejkę `~/fuzz/queue.sh`, z `nice -n 19` i trzema workerami (jeden rdzeń zostaje dla DSM). Binarki testów są
kompilowane na Macu (`GOOS=linux GOARCH=amd64 go test -c`), a katalog zadania ma układ repozytorium: `<pakiet>/<pakiet>.test`
i `testdata/corpus`. Znalezione błędy trafiają do `<pakiet>/testdata/fuzz/FuzzRender` w katalogu zadania. Czas podajemy jako
czas zegarowy na NAS-ie, który jest wolniejszy od Maca.

## Kampanie

| Data | Commit | Host | Cele i czas | Wynik |
|---|---|---|---|---|
| 2026-11-26 (w kolejce) | 599ab96 | StormSyn | `mermaid` 8 h, `pdf` 4 h, `png` 4 h, po kolei | czeka na koniec kampanii gohighlightgo (start ok. 2026-11-27 09:00) |

Katalog zadania: `~/fuzz/gomermaidgo/599ab96/`, dziennik `run.log`.
