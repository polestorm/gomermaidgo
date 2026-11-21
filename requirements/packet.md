# Wymagania: diagram pakietu


| Id | Wymaganie |
|---|---|
| PKT-001 | Źródło zaczynające się od `packet-beta` (albo `packet`) daje diagram; `title` nadaje tytuł; komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| PKT-002 | Pole to `0-15: "nazwa"`, `16: "nazwa"` (jeden bit) albo `+8: "nazwa"` (długość od pierwszego wolnego bitu). Pola następują po sobie bez przerw, od bitu 0. |
| PKT-003 | Rysunek: wiersze po 32 bity; pole dłuższe niż reszta wiersza przechodzi do następnego; nad każdym kawałkiem pola stoją numery jego pierwszego i ostatniego bitu; nazwa stoi w polu w rozmiarze, w którym się mieści. Szerokość pola jest proporcjonalna do liczby bitów. |
| PKT-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, linia niebędąca polem, przerwa albo nakładanie się pól, koniec przed początkiem, długość zerowa, pakiet dłuższy niż 2048 bitów, brak pól. |
