# 0002 Dołączony krój domyślny: Recursive

Status: zastąpiona przez ADR 0007 · 2026-11-21

## Kontekst

Renderer musi znać szerokości znaków, a rysunek wygląda dobrze tylko wtedy, gdy tekst rysuje ten sam krój, którym go
zmierzono. Dotąd domyślne było `sans-serif` z miarą zastępczą, która zawyża: pola wychodziły za szerokie, a wygląd
zależał od tego, co podstawi przeglądarka albo konwerter PDF. Po przeglądzie pięciu krojów (`tools/fontsheet`) wybrany
został Recursive.

## Decyzja

1. Moduł zawiera pliki `fonts/Recursive-Regular.ttf`, `Recursive-Bold.ttf` i `Recursive-Italic.ttf` (niezmienione,
   SIL Open Font License 1.1, tekst w `fonts/OFL.txt`) i udostępnia je przez pakiet `fonts`. Odmiany pogrubiona
   i pochylona mierzą i rysują wyróżnienia w tekstach Markdown.
2. Bez opcji `WithFontFamily` rysunek ma `font-family="Recursive, sans-serif"`, a tekst jest mierzony dołączonym
   plikiem.
3. Kto podaje własny krój, podaje też jego plik (`WithFont`) albo funkcję pomiaru (`WithMeasure`); inaczej działa miara
   zastępcza.
4. Program rysujący SVG musi dostać ten sam plik (`fonts.Recursive()`): strona przez `@font-face`, konwerter PDF jako
   font do osadzenia.

## Konsekwencje

+ wynik domyślny jest od razu poprawnie zmierzony i wszędzie wygląda tak samo;
+ zasada „bez zależności” dotyczy kodu i pozostaje spełniona; font jest zasobem, nie pakietem;
− plik wykonywalny rośnie o około 1,3 MB (trzy odmiany);
− w repozytorium jest druga licencja (OFL 1.1 dla fontu); rozpowszechniając font, trzeba dołączyć jej tekst;
− bez dostarczenia fontu odbiorcy tekst narysuje krój zastępczy o innych szerokościach.
