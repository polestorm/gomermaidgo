# 0007 Krój domyślny: Arimo zamiast Recursive

Status: przyjęta · 2026-11-23 · zastępuje ADR 0002

## Kontekst

Rysunek SVG zapisuje tekst jako tekst i podaje nazwę kroju; szerokości pól są liczone z dołączonego kroju. Recursive
(ADR 0002) prawie nigdzie nie jest zainstalowany, więc SVG otwarty poza naszą drogą (przeglądarka, edytor, cudzy
konwerter) rysował tekst krojem zastępczym o innych szerokościach: tekst wychodził z pól albo zostawiał w nich luz.
Zamiana tekstu na krzywe odpada, bo usuwa wyszukiwanie i znacznie powiększa plik.

## Decyzja

1. Krojem domyślnym jest Arimo (OFL 1.1) w trzech odmianach, pliki niezmienione. Arimo ma te same szerokości znaków
   co Arial i Liberation Sans.
2. SVG podaje `font-family="Arimo, Arial, 'Liberation Sans', sans-serif"`. Na Windows, macOS i iOS rysuje Arial, na
   większości dystrybucji Linuksa Liberation Sans; układ zostaje taki, jak zmierzony. PDF i PNG rysują Arimo z pliku.
3. Kroje są wbudowane bez kompresji: rozpakowanie kosztowałoby kilka milisekund przy każdym uruchomieniu programu,
   a archiwum wydania i paczka modułu i tak są skompresowane.

## Skutki

- Program rośnie o 0,17 MB (Arimo ma też cyrylicę, grekę i hebrajski).
- Rysunki są o kilka procent węższe niż z Recursive.
- Bez zmian zostaje Android i systemy bez Ariala i Liberation Sans: tam pomaga tylko podanie pliku kroju.
- Wygląd jest neutralny zamiast charakterystycznego; dla rysunków pod druk to zaleta.
- API pakietu `fonts` zmienia nazwy: `Arimo()`, `ArimoBold()`, `ArimoItalic()`, `ArimoLicense()`, `DefaultFamily`.
