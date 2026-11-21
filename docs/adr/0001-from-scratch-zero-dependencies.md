# 0001 Renderer pisany od zera, bez zależności, z własnym wyglądem

Status: przyjęta (robocza) · 2026-11-21

## Kontekst

Istniejące renderery Mermaid bez przeglądarki są albo bardzo duże i obciążone licencyjnie (port oryginału z fragmentami
na LGPL i EPL, interpreter JS w zależnościach), albo małe, lecz o słabym układzie. Potrzebny jest składnik, który da się
w całości przeczytać, zwalidować i połączyć statycznie z produktem zamkniętym.

## Decyzja

1. Kod powstaje od zera ze specyfikacji i publikacji (`../provenance.md`); cudze renderery służą tylko jako czarna
   skrzynka do porównań.
2. Moduł nie ma zależności poza biblioteką standardową Go. Dotyczy to także testów.
3. Każdy typ diagramu przechodzi tę samą ścieżkę: tekst → model → scena (`internal/scene`) → SVG. Testy niezmienników
   pracują na scenie.
4. SVG składa się z elementów podstawowych z atrybutami prezentacji: bez `foreignObject`, arkuszy stylów, skryptów,
   odnośników i identyfikatorów (groty strzałek są wielokątami, nie znacznikami `marker`).
5. Szerokość tekstu pochodzi z fontu podanego przez wywołującego (własny czytnik `cmap` i `hmtx`) albo z jego funkcji
   pomiaru; bez nich działa miara zastępcza, która zawyża.
6. Wygląd jest własny i stały. Zgodność z Mermaid oznacza zgodność znaczenia zapisu i treści rysunku, nie pikseli.
7. Element składni spoza profilu daje błąd z numerem linii i stałym kodem.

## Konsekwencje

+ mały zakres walidacji, jedna licencja, wynik powtarzalny;
+ kilka diagramów na jednej stronie HTML nie koliduje (brak identyfikatorów i stylów);
− sami utrzymujemy zgodność ze składnią w zakresie profilu;
− układ grafów (etap 3) trzeba napisać z publikacji; jakość oryginału nie jest celem.
