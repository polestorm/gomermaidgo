# 0003 Wynik PDF: własny pisarz ze sceny, w opcjonalnym pakiecie

Status: przyjęta · 2026-11-21

## Kontekst

Rysunek w PDF dało się dotąd uzyskać tylko drogą okrężną: SVG na stronie HTML i zewnętrzny konwerter. Ta droga zostaje
dla dokumentów (tekst, nagłówki, kilka rysunków na stronach), ale kto chce sam rysunek, musiałby stawiać całą ścieżkę.
Rozważone były trzy wyjścia: program sklejający renderer z konwerterem (wnosi wszystkie zależności konwertera),
wydzielenie z konwertera jego biblioteki PDF (kilkanaście tysięcy linii przejętego kodu, sprzeczne z zasadą czystego
rodowodu z ADR 0001) i własny pisarz.

## Decyzja

1. Pakiet `pdf` zapisuje scenę rysunku bezpośrednio jako jednostronicowy PDF. Scena jest ta sama, z której powstaje
   SVG, więc oba formaty nie mogą się rozjechać, a nowy typ diagramu dostaje PDF bez zmian w pisarzu.
2. Pisarz powstaje ze specyfikacji PDF (ISO 32000-1) i OpenType, bez zależności: kompresję daje biblioteka standardowa.
3. Tekst jest tekstem: krój TrueType osadzony jako font złożony z kodowaniem Identity-H i tabelą ToUnicode. Osadzany
   jest podzbiór: kontury tylko użytych glifów i ich składników, z zachowaniem numerów glifów (bez przenumerowania),
   bez tabel niepotrzebnych do rysowania.
4. Pakiet jest opcjonalny. Biblioteka: kto nie importuje `pdf`, nie ma jego kodu. Program z linii poleceń: format
   rejestruje plik kompilowany warunkowo, a znacznik `nopdf` buduje program bez niego.
5. Pakiet główny nie wystawia sceny publicznie; `pdf` dostaje ją przez `internal/bridge`.
6. Format wymaga pliku kroju. Rysunek mierzony funkcją albo krojem znanym tylko z nazwy daje błąd, a nie PDF
   z krojem zastępczym.

## Poza zakresem

PDF/A, znaczniki struktury (tagged PDF), kroje z konturami CFF, kerning i ligatury (tekst jest składany z szerokości
znaków, tak jak jest mierzony), strona w formacie papieru. Dokumenty wielostronicowe z tekstem składa konwerter HTML.

## Konsekwencje

+ `gomermaidgo -o rysunek.pdf` działa samodzielnie; plik ma kilkanaście kilobajtów;
+ wynik jest powtarzalny co do bajta: bez dat i losowych identyfikatorów;
+ program z PDF jest większy o około 0,2 MB;
− drugi pisarz do utrzymania: nowy rodzaj kształtu w scenie trzeba dopisać w obu;
− krój OpenType z konturami CFF nie da PDF, choć daje SVG.
