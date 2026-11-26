# Wymagania: mapa Cynefin


| Id | Wymaganie |
|---|---|
| CYN-001 | Źródło zaczynające się od `cynefin-beta` albo `cynefin` daje mapę Cynefin. `title` dodaje tytuł nad planszą; komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| CYN-002 | Nazwa domeny w osobnej linii (`complex`, `complicated`, `clear`, `chaotic`, `confusion`) otwiera jej listę; następne linie w cudzysłowie to jej elementy. Domenę można zadeklarować kilka razy; elementy się dopisują. |
| CYN-003 | `domena --> domena` z opcjonalnym `: "podpis"` to przejście; przejście domeny do niej samej jest pomijane. |
| CYN-004 | Rysunek: plansza z czterema domenami o stałym położeniu (złożona u góry z lewej, skomplikowana u góry z prawej, chaotyczna u dołu z lewej, oczywista u dołu z prawej) i zagubieniem w elipsie pośrodku. Granice są łagodnie wygięte; dolna połowa pionowej granicy, między domeną oczywistą a chaotyczną, jest grubsza (klif). Domena ma nazwę, model działania i rodzaj praktyki przy zewnętrznej krawędzi i elementy w ramkach; w zagubieniu najwyżej trzy elementy i ramka „+N more”. Przejścia to łuki ze strzałką między domenami, wygięte od środka planszy, z podpisem po zewnętrznej stronie; przejście po przekątnej omija elipsę. Domyślnie rysunek jest neutralny, `WithColors` barwi tła domen. |
| CYN-005 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, element przed nazwą domeny, element bez zamykającego cudzysłowu, przejście z nazwą spoza pięciu domen, nieznana instrukcja (także element bez cudzysłowu), ponad 120 elementów. |
