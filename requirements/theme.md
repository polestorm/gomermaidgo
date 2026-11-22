# Wymagania: motywy


| Id | Wymaganie |
|---|---|
| THM-001 | Motywy: `neutral` (domyślny, rysunek bez zmian), `default` (tryb kolorowy, to samo co `WithColors`), `forest` (odcienie neutralne zabarwione na zielono), `dark` (jasny rysunek na ciemnym tle), `base` (neutralny, do barwienia zmiennymi). Motyw działa dla każdego typu diagramu i każdego formatu wyniku, bo przemalowuje gotową scenę. Kolory nasycone, w tym nadane w źródle stylami, zostają; w motywie ciemnym ich jasne tła ciemnieją. |
| THM-002 | Motyw wybiera `WithTheme`; bez tej opcji źródło diagramu: klucz `theme` w bloku nagłówkowym (`config:`) albo w dyrektywie `%%{init}%%`. Opcja ma pierwszeństwo przed źródłem i wyłącza jego zmienne. Nazwa nieznana w opcji jest błędem `ErrUnknownTheme`, a w źródle oznacza motyw domyślny. |
| THM-003 | Zmienne motywu ze źródła (`themeVariables`): `primaryColor` barwi odcienie neutralne barwą podanego koloru, `lineColor` nadaje kolor liniom, `primaryTextColor` albo `textColor` tekstowi, `background` tłu. Pozostałe zmienne są pomijane. |
| THM-004 | Tekst zachowuje czytelność: napis, który po przemalowaniu miałby kontrast z tłem pod sobą poniżej 3:1, dostaje jasny albo ciemny kolor motywu, zależnie od tego, który kontrastuje mocniej. Tło podane przez wywołującego ma pierwszeństwo przed tłem motywu. Program z linii poleceń wybiera motyw flagą `-t`/`--theme`. |
