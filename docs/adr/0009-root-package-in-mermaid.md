# 0009 Pakiet główny w katalogu `mermaid/`, w katalogu głównym tylko trzy pliki

Status: przyjęta · 2026-11-23

## Kontekst

Repozytorium ma zostać udostępnione publicznie. Katalog główny miał kilkanaście plików: kod pakietu głównego
z testami, `CLAUDE.md`, `NOTICE`, `LICENSE`, `README.md` i `go.mod`. Właściciel chce, żeby w katalogu głównym były
tylko podkatalogi i najwyżej trzy pliki.

## Decyzja

1. Kod pakietu głównego przechodzi do `mermaid/` (pakiet `mermaid`). Ścieżka importu zmienia się
   z `github.com/polestorm/gomermaidgo` na `github.com/polestorm/gomermaidgo/mermaid`, a wywołanie
   z `gomermaidgo.Render` na `mermaid.Render`; nazwa współgra z pakietami `pdf`, `png` i `markdown`.
2. W katalogu głównym zostają `README.md`, `LICENSE` i `go.mod` (moduł Go musi mieć `go.mod` w katalogu głównym).
3. `NOTICE` dotyczy głównie dołączonych krojów, więc przechodzi do `fonts/NOTICE`; README wskazuje to miejsce.
4. `CLAUDE.md` przechodzi do `.claude/CLAUDE.md` (miejsce czytane przez Claude Code) i zostaje w repozytorium.
5. `testdata/` i `requirements/` zostają w katalogu głównym, bo czytają je też testy pakietów `internal/`;
   testy w `mermaid/` sięgają do nich przez `../`.

## Skutki

- Zmiana niezgodna wstecz dla wywołujących. Przed upublicznieniem jedynym znanym użytkownikiem jest `gopressgo`
  (pusty import w `deps.go`, wersja przypięta w `go.mod`); przy podniesieniu wersji import trzeba zmienić na
  `…/gomermaidgo/mermaid`.
- Wzorce zmieniamy przez `go test ./mermaid -update`.

## Rozważone i odrzucone (2026-11-23)

Krótszy import `github.com/polestorm/gomermaidgo` (wywołanie `gomermaidgo.Render`) przez plik pośredniczący
w katalogu głównym, przekazujący publiczne nazwy z `mermaid/` (aliasy typów, zmienne z funkcjami, stałe).
Odrzucone: czwarty plik w katalogu głównym i druga lista około 60 publicznych nazw, którą trzeba uzupełniać przy
każdej nowej opcji (z testem strażnikiem pilnującym zgodności). Import pisze się raz, a `mermaid.Render` pasuje do
pakietów `pdf`, `png` i `markdown`.
