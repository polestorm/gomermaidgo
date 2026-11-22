# 0006 Krój zapasowy: dołączony Noto Emoji dla znaków spoza kroju głównego

Status: przyjęta · 2026-11-23

## Kontekst

Krój domyślny (Recursive z ADR 0002, potem Arimo z ADR 0007) nie ma emoji. W próbie na publicznych diagramach miało je 28 z 62 narysowanych
bloków. Taki znak wychodził w PDF i PNG jako puste pole, a w SVG rysował go krój zastępczy przeglądarki o innej
szerokości niż zmierzona.

## Decyzja

1. Moduł dołącza drugi krój, Noto Emoji (jednobarwne emoji, licencja OFL 1.1), jako krój zapasowy. Znak, którego krój
   główny nie ma, jest mierzony i rysowany krojem zapasowym, jeśli ten go ma. Dotyczy to także kroju podanego przez
   wywołującego.
2. Dołączony plik to odmiana domyślna kroju zmiennego, bez tabel odmian i tabel niepotrzebnych do rysowania konturów
   (`tools/staticfont`): 0,84 MB zamiast 1,98 MB. Kontury i odwzorowanie znaków są bez zmian.
3. Emoji są jednobarwne. Kolorowe wymagają formatów kroju, których nie czytamy, i osobnej obsługi w PDF.
4. W SVG emoji są domyślnie konturami: napis ze znakami kroju zapasowego jest dzielony na odcinki jednego kroju,
   a odcinek z emoji jest zapisywany jako ścieżka w kolorze tekstu (łuki kroju jako krzywe Béziera). Rysunek wygląda
   wtedy tak samo w każdym programie i nie potrzebuje pliku kroju zapasowego. Samego emoji nie da się skopiować ani
   wyszukać, ale słowa obok zostają tekstem; wyszukiwanie po znaku emoji uznaliśmy za nieistotne. `WithEmojiText`
   (flaga `-emoji-text`) zostawia emoji jako tekst; nazwę kroju zapasowego niesie wtedy tylko odcinek z emoji, nie
   lista krojów całego rysunku, bo Noto Emoji ma własne cyfry i program rysujący mógłby po nie sięgnąć.
5. W PDF (pakiet `pdf`) emoji pozostają tekstem: krój zapasowy jest osadzany jako osobny podzbiór tylko wtedy, gdy
   rysunek go używa. PNG rysuje kontury.

## Skutki

- Program z linii poleceń rośnie z 6,9 do 7,8 MB, paczka modułu z 1,05 do 1,62 MB.
- SVG z emoji jest większy: kontur to około 2 kB na każde wystąpienie znaku (diagram z 11 emoji: 28 kB zamiast
  6 kB). Z `WithEmojiText` program rysujący SVG poza przeglądarką musi dostać oba kroje nazwane wprost (`@font-face`).
- Nie czytamy tabel podstawień, więc sekwencje złożone (flagi, odcienie skóry, rodziny) wychodzą jako osobne znaki.
- Znaki, których nie ma żaden z krojów (np. pismo chińskie), nadal są pustym polem i są zgłaszane przez
  `MissingGlyphs`.
