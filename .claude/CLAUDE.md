# gomermaidgo — instrukcje dla Claude Code

Projekt: renderer diagramów zapisanych składnią Mermaid do SVG w czystym Go. Cele: zero zależności poza biblioteką
standardową, bez JS, wynik powtarzalny co do bajta, SVG z samych elementów podstawowych (rysuje go `gopdfgo`),
mały kod, który da się przeczytać i zwalidować. Projekt jest agnostyczny wobec odbiorców.

Zacznij od `README.md` (pakiet główny to `mermaid/`), potem `docs/roadmap.md`, `docs/profile.md` i `docs/provenance.md`.

Zasady:
- **czysty rodowód** (`docs/provenance.md`): piszemy ze specyfikacji i publikacji. Nie czytaj, nie kopiuj i nie przenoś
  kodu innych rendererów Mermaid ani bibliotek układu grafów (mermaid, dagre, ELK, mmdg, go-mermaid, mermaid-rs-renderer,
  merman, D2, Graphviz). Wolno czytać publiczną dokumentację składni i uruchamiać cudze renderery jako czarną skrzynkę,
  żeby porównać wynik;
- **zero zależności**: `go.mod` nie ma sekcji `require`; narzędzia porównawcze wywołują cudze programy przez proces;
- nazwy plików, pakietów i identyfikatorów po angielsku; komentarze i dokumentacja po polsku; `README.md` po angielsku;
  galeria `docs/gallery/`, `docs/usage.md` (przedłużenie README) i `docs/profile.md` (obsługiwana składnia) też po angielsku, obrazki odtwarza `go run ./tools/gallery`
  (po zmianie wyglądu warto je odtworzyć);
  komunikaty błędów biblioteki po angielsku, ze stałym kodem do tłumaczenia przez wywołującego;
- wzorcem są normy (SVG W3C, OpenType) i dokumentacja składni Mermaid, nie zachowanie innych rendererów;
- każdy obsługiwany element składni ma wymaganie w `requirements/`, plik w `testdata/corpus/` z linią `%% REQ:` i wpis
  w `docs/profile.md`; element nieobsługiwany daje błąd z numerem linii;
- pliki wzorcowe `testdata/golden/` zmieniamy przez `go test ./mermaid -update` i oglądamy różnicę przed commitem;
- wygląd jest własny i stały: nie odtwarzamy motywów Mermaid ani wyniku co do piksela; celem jest rysunek czytelny
  i przewidywalny;
- każda decyzja = plik w `docs/adr/`;
- opisy commitów po angielsku; przed commitem muszą przechodzić `go vet ./...` i `go test ./...`;
- commit i push do `origin` (prywatne `polestorm/gomermaidgo`) po każdej zakończonej rundzie pracy, bez pytania
  (decyzja użytkownika z 2026-11-21).
