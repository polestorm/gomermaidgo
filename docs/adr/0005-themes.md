# 0005 Motywy: własne, pod nazwami Mermaid, jako przemalowanie sceny

Status: przyjęta · 2026-11-22

## Kontekst

Rysunek miał jeden wygląd (odcienie szarości pod druk) i tryb kolorowy. Źródła pisane dla Mermaid często wybierają
motyw w bloku nagłówkowym albo dyrektywie `init`, a program `mermaid-cli` ma flagę `-t`. Kolory są zapisane w ponad
dwudziestu pakietach typów diagramów; motyw przekazywany do każdego z nich oznaczałby zmianę we wszystkich i w każdym
nowym typie.

## Decyzja

1. Motywy są własne, ale noszą nazwy motywów Mermaid (`default`, `neutral`, `forest`, `dark`, `base`), żeby cudze
   źródło wybierało odpowiednik. Kolorów Mermaid nie kopiujemy. Domyślny pozostaje `neutral`: dotychczasowy wygląd.
2. Motyw przemalowuje gotową scenę, po układzie: odcienie neutralne rysunku (szarości i stonowane błękity) dostają
   barwę motywu, a w motywie ciemnym odwróconą jasność. Typy diagramów i formaty wyniku o motywach nie wiedzą, więc
   nowy typ dostaje je bez pracy. `default` jest wyjątkiem: włącza istniejący tryb kolorowy układu.
3. Kolory nasycone zostają: paleta trybu kolorowego i kolory nadane w źródle stylami.
4. Po przemalowaniu każdy napis jest sprawdzany względem tła, na którym stoi; poniżej kontrastu 3:1 dostaje jasny albo
   ciemny kolor motywu. To chroni także tekst na polu pokolorowanym w źródle.
5. Wybiera wywołujący (`WithTheme`, flaga `-t`), a bez jego wyboru źródło diagramu. Dokument, który ma wyglądać
   jednolicie, podaje motyw sam, choćby `neutral`; wyłącza to także zmienne motywu ze źródła.
6. Ze zmiennych motywu (`themeVariables`) działają cztery: `primaryColor` (barwa odcieni neutralnych), `lineColor`,
   `primaryTextColor`/`textColor`, `background`.

## Skutki i ograniczenia

- Rozpoznanie „odcienia neutralnego” opiera się na barwie i nasyceniu, więc blady błękit nadany w źródle też zostanie
  zabarwiony.
- Motyw ciemny ustawia tło rysunku; jest przeznaczony na ekran, nie do druku.
- Poza zakresem: własny arkusz CSS (`-C`), pozostałe zmienne motywu, motywy definiowane przez wywołującego.
