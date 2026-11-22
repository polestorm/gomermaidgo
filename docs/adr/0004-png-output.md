# 0004 Wynik PNG: własny rasteryzator ze sceny, w opcjonalnym pakiecie

Status: przyjęta · 2026-11-22

## Kontekst

Część odbiorców nie przyjmuje rysunków wektorowych: komunikatory, niektóre edytory i systemy zgłoszeń chcą obrazka.
Renderery oparte na przeglądarce robią PNG zrzutem ekranu; my nie mamy przeglądarki ani zależności (ADR 0001).
Biblioteka standardowa Go koduje PNG, ale nie rysuje kształtów ani tekstu.

## Decyzja

1. Pakiet `png` rysuje scenę rysunku na obrazie rastrowym. Scena jest ta sama, z której powstają SVG i PDF
   (ADR 0003), więc nowy typ diagramu dostaje PNG bez zmian w pakiecie.
2. Rasteryzator jest własny (`internal/raster`): kształt to zbiór zamkniętych wielokątów, pokrycie piksela liczy się
   z pola kształtu w jego obrębie, a obrys jest sumą prostokątów odcinków i kół w załamaniach. Krzywe są zamieniane
   na odcinki. Bez nadpróbkowania i bez zależności; kodowanie pliku daje biblioteka standardowa.
3. Tekst jest rysowany konturami glifów TrueType czytanymi wprost z pliku kroju (`internal/glyf`), tym samym krojem,
   którym rysunek został zmierzony. Bez hintingu: przy skali domyślnej nie jest potrzebny.
4. Skala domyślna to 2 piksele obrazu na piksel rysunku, bo obraz w skali 1 jest nieostry w druku i na ekranach
   o dużej gęstości. `RenderScale` i flaga `-s` ustawiają inną.
5. Pakiet jest opcjonalny, tak jak `pdf`: znacznik `nopng` buduje program bez niego.
6. Czytanie kolorów CSS jest wspólne dla `pdf` i `png` (`internal/csscolor`). Czytanie tabel kroju pozostaje osobne
   w obu pakietach: `pdf` buduje podzbiór pliku, `png` potrzebuje tylko konturów.

## Poza zakresem

Hinting, kerning i ligatury, kroje z konturami CFF, końce i złącza linii inne niż zaokrąglone złącza i proste końce,
formaty JPEG i WebP, metadane rozdzielczości w pliku.
