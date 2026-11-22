# Wymagania wspólne dla wszystkich typów


| Id | Wymaganie |
|---|---|
| COM-001 | Blok nagłówkowy YAML między liniami `---` przed diagramem jest przyjmowany dla każdego typu. Klucz `title` daje tytuł rysowany nad diagramem, wyśrodkowany, i trafia do elementu `title`, jeśli diagram nie ma własnego `accTitle`; pozostałe klucze (`config`, motywy) są pomijane. Blok niezamknięty jest błędem z numerem linii. |
| COM-002 | Dyrektywa `%%{ … }%%`, także rozciągnięta na kilka linii, jest przyjmowana i pomijana w każdym miejscu źródła; dyrektywa niezamknięta jest błędem z numerem linii. |
| COM-003 | Blok nagłówkowy i dyrektywy nie zmieniają numerów linii w komunikatach błędów diagramu. |
