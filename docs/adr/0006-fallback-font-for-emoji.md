# 0006 Krój zapasowy: dołączony Noto Emoji dla znaków spoza kroju głównego

Status: przyjęta · 2026-11-23

## Kontekst

Krój domyślny Recursive (ADR 0002) nie ma emoji. W próbie na publicznych diagramach miało je 28 z 62 narysowanych
bloków. Taki znak wychodził w PDF i PNG jako puste pole, a w SVG rysował go krój zastępczy przeglądarki o innej
szerokości niż zmierzona.

## Decyzja

1. Moduł dołącza drugi krój, Noto Emoji (jednobarwne emoji, licencja OFL 1.1), jako krój zapasowy. Znak, którego krój
   główny nie ma, jest mierzony i rysowany krojem zapasowym, jeśli ten go ma. Dotyczy to także kroju podanego przez
   wywołującego.
2. Dołączony plik to odmiana domyślna kroju zmiennego, bez tabel odmian i tabel niepotrzebnych do rysowania konturów
   (`tools/staticfont`): 0,84 MB zamiast 1,98 MB. Kontury i odwzorowanie znaków są bez zmian.
3. Emoji są jednobarwne. Kolorowe wymagają formatów kroju, których nie czytamy, i osobnej obsługi w PDF.
4. W scenie napis ze znakami kroju zapasowego jest dzielony na odcinki jednego kroju. W SVG nazwę kroju zapasowego
   niesie tylko odcinek z tymi znakami, nie lista krojów całego rysunku: Noto Emoji ma własne cyfry i program
   rysujący SVG mógłby po nie sięgnąć.
5. W PDF krój zapasowy jest osadzany jako osobny podzbiór tylko wtedy, gdy rysunek go używa.

## Skutki

- Program z linii poleceń rośnie z 6,9 do 7,8 MB, paczka modułu z 1,05 do 1,62 MB.
- Kto rysuje SVG poza przeglądarką (konwerter do PDF), musi dać mu oba kroje i nazwać je wprost (`@font-face`).
  Konwerter, który sam dobiera kroje z katalogu plików, może narysować cyfry krojem zapasowym.
- Nie czytamy tabel podstawień, więc sekwencje złożone (flagi, odcienie skóry, rodziny) wychodzą jako osobne znaki.
- Znaki, których nie ma żaden z krojów (np. pismo chińskie), nadal są pustym polem i są zgłaszane przez
  `MissingGlyphs`.
