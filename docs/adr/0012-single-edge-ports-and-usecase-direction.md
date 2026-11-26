# 0012 Pojedyncza krawędź wychodzi ze środka boku; przypadki użycia domyślnie w kierunku LR

Status: przyjęta · 2026-11-26

## Kontekst

Przegląd właściciela (usecase/001–003, porównanie z `mmdg`) wskazał trzy rzeczy:

- krawędź, która jako jedyna wychodzi z boku pola (`«include»` z „Zgłoś reklamację”, `«extend»` z „Raport 8D”),
  zaczynała się obok środka boku: punkt zaczepienia szedł w oś etykiety, choć krawędź i tak skręcała dalej;
- przy aktorze, z którego wychodzą cztery krawędzie gęsto obok siebie, krawędź do „Zatwierdź dokument” miała uskok
  rzędu 3 px, bo sąsiednie punkty nie pozwalały jej się wyprostować;
- diagram bez `direction` układał się z góry na dół, a `mmdg` i wszystkie przykłady dokumentacji Mermaid układają
  go od lewej do prawej; dokumentacja domyślnego kierunku nie podaje. Przy wielu aktorach układ LR ma więcej miejsca:
  aktorzy stoją w kolumnie po lewej, przypadki użycia w ramce po prawej.

## Decyzja

1. Jedyna krawędź wychodząca z boku pola zaczyna się w jego środku (do 4 px obok), jeśli przesunięcie punktu
   zaczepienia i tak nie wyprostowałoby jej do drugiego końca. Krawędź, która po przesunięciu biegnie prosto od pola
   do pola, zostaje prosta (to zasada LAY-014 dla krawędzi wchodzącej, rozszerzona na wychodzącą). Punkty pośrednie
   krawędzi mogą stanąć o kilka pikseli od osi końca, jeśli ten koniec za nimi pójdzie.
2. Gdy gęsto ustawione punkty zaczepienia nie pozwalają wyprostować krawędzi o kilka pikseli, cała grupa układa się na
   nowo wokół niej, z odstępem nie mniejszym niż 4 px. Zasięg punktów zaczepienia aktora rośnie z 16 do 24 px.
3. Domyślny kierunek diagramu przypadków użycia to `LR`.
4. Odmiany aktora z dokumentacji Mermaid są rysowane po swojemu: `type: hollow` to postać obrysowana konturem,
   `business: true` dodaje ukośną kreskę przez dół głowy aktora albo przy prawym dolnym brzegu elipsy przypadku użycia.
   `type: awesome` i ikony zostają zwykłą postacią.

## Skutki

Zmiana punktów zaczepienia dotyczy wszystkich typów z układem warstwowym. W korpusie zmieniły się poza przypadkami
użycia dwa rysunki (`flowchart/007`, `flowchart/018`), oba z mniejszą liczbą załamań; rysunki z prostymi krawędziami
obok środka boku (np. `flowchart/001`, `flowchart/026`) zostały bez zmian. Test `TestSingleOutgoingEdgeLeavesCentre`
pilnuje reguły, `TestNoSmallJogs` — braku drobnych uskoków.
