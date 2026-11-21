# 0002 Dołączony krój domyślny: Recursive

Status: przyjęta (robocza) · 2026-11-21

## Kontekst

Renderer musi znać szerokości znaków, a rysunek wygląda dobrze tylko wtedy, gdy tekst rysuje ten sam krój, którym go
zmierzono. Dotąd domyślne było `sans-serif` z miarą zastępczą, która zawyża: pola wychodziły za szerokie, a wygląd
zależał od tego, co podstawi przeglądarka albo konwerter PDF. Po przeglądzie pięciu krojów (`tools/fontsheet`) wybrany
został Recursive.

## Decyzja

1. Moduł zawiera plik `fonts/Recursive-Regular.ttf` (niezmieniony, SIL Open Font License 1.1, tekst w `fonts/OFL.txt`)
   i udostępnia go przez pakiet `fonts`.
2. Bez opcji `WithFontFamily` rysunek ma `font-family="Recursive, sans-serif"`, a tekst jest mierzony dołączonym
   plikiem.
3. Kto podaje własny krój, podaje też jego plik (`WithFont`) albo funkcję pomiaru (`WithMeasure`); inaczej działa miara
   zastępcza.
4. Program rysujący SVG musi dostać ten sam plik (`fonts.Recursive()`): strona przez `@font-face`, konwerter PDF jako
   font do osadzenia.

## Konsekwencje

+ wynik domyślny jest od razu poprawnie zmierzony i wszędzie wygląda tak samo;
+ zasada „bez zależności” dotyczy kodu i pozostaje spełniona; font jest zasobem, nie pakietem;
− plik wykonywalny rośnie o około 0,4 MB;
− w repozytorium jest druga licencja (OFL 1.1 dla fontu); rozpowszechniając font, trzeba dołączyć jej tekst;
− bez dostarczenia fontu odbiorcy tekst narysuje krój zastępczy o innych szerokościach.
