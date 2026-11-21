# Wymagania: wykres Gantta

Każde wymaganie ma identyfikator `GANTT-nnn`; pliki korpusu (`testdata/corpus/gantt/*.mmd`) i testy odwołują się do
nich przez `REQ: GANTT-nnn`.

| Id | Wymaganie |
|---|---|
| GANTT-001 | Źródło zaczynające się od `gantt` daje model i rysunek; komentarze `%%` i puste linie są pomijane; `title`, `section`, `accTitle`, `accDescr`. |
| GANTT-002 | Zadanie `nazwa : [znaczniki,] [id,] [początek,] koniec`: początek datą w zapisie `dateFormat` albo `after id…` (najpóźniejszy koniec), bez początku zaczyna się po poprzednim zadaniu; koniec datą, czasem trwania (`ms`, `s`, `m`, `h`, `d`, `w`, `M`, `y`, także ułamkowym) albo `until id`. Zadanie może odwoływać się do zadania zapisanego niżej. |
| GANTT-003 | Znaczniki `done`, `active`, `crit`, `milestone` zmieniają wygląd: stan jest rozróżnialny bez koloru; kamień milowy jest rombem. Nazwa mieści się w pasku albo stoi obok niego. |
| GANTT-004 | `excludes` (`weekends`, nazwy dni tygodnia, daty `YYYY-MM-DD`) i `weekend friday`: czas trwania liczy tylko dni robocze; dni wyłączone są zacieniowane, gdy podziałka na to pozwala. `inclusiveEndDates` wlicza dzień końca. |
| GANTT-005 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem: nieznana data, nieznany identyfikator, zależność cykliczna, koniec przed początkiem, zapis dat spoza obsługiwanych znaczników. |
| GANTT-006 | Podziałka czasu: `axisFormat` w notacji strftime, `tickInterval` (dzień, tydzień, miesiąc, rok); bez niego odstęp jest dobierany tak, żeby opisy się nie stykały. Wynik nie zależy od dnia uruchomienia: znacznik „dziś” nie jest rysowany. |
