# Wymagania wspólne dla wszystkich typów


| Id | Wymaganie |
|---|---|
| COM-001 | Blok nagłówkowy YAML między liniami `---` przed diagramem jest przyjmowany dla każdego typu. Klucz `title` daje tytuł rysowany nad diagramem, wyśrodkowany, i trafia do elementu `title`, jeśli diagram nie ma własnego `accTitle`; pozostałe klucze (`config`, motywy) są pomijane. Blok niezamknięty jest błędem z numerem linii. |
| COM-002 | Dyrektywa `%%{ … }%%`, także rozciągnięta na kilka linii, jest przyjmowana i pomijana w każdym miejscu źródła; dyrektywa niezamknięta jest błędem z numerem linii. |
| COM-003 | Blok nagłówkowy i dyrektywy nie zmieniają numerów linii w komunikatach błędów diagramu. |
| COM-004 | W trybie kolorowym elementy serii (uczestnicy, gałęzie, sekcje, wycinki, serie danych) dostają kolory w kolejności, w której każdy następny leży możliwie daleko w barwie od wcześniejszych: dwa pierwsze po przeciwnych stronach koła barw. Po wyczerpaniu palety kolory powstają przez obrót barwy, na zmianę ciemniejsze i jaśniejsze, i sąsiedzi nadal się różnią. |
| COM-005 | Znak, którego krój główny nie ma, jest mierzony i rysowany krojem zapasowym, jeśli ten go ma. Krojem zapasowym jest dołączony Noto Emoji (jednobarwne emoji), także gdy krój główny podaje wywołujący. W PDF krój zapasowy jest osadzany jako osobny podzbiór tylko wtedy, gdy rysunek go używa, a tekst pozostaje tekstem; w PNG emoji rysują kontury kroju zapasowego; w SVG napis ze znakami kroju zapasowego jest podzielony na odcinki jednego kroju, a nazwę kroju zapasowego niesie tylko odcinek z tymi znakami (nie lista krojów całego rysunku, bo krój z emoji ma własne cyfry i program rysujący mógłby po nie sięgnąć). Selektory wariantu i łącznik zerowej szerokości, których krój nie ma, są pomijane. `MissingGlyphs` zgłasza tylko znaki, których nie ma żaden z krojów. |
