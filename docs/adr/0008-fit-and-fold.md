# 0008 Dopasowanie do strony i zwijanie: domyślnie włączone dla rysunków długich

Status: przyjęta · 2026-11-23

## Kontekst

Diagramy rysowane w dokumentach trafiają na strony A4. Schemat LR z kilkunastoma polami w jednym ciągu jest kilka
razy szerszy niż wysoki i w druku musi być mocno pomniejszony. Mermaid nie ma na to rozwiązania.

## Decyzja

1. **Dopasowanie** (ADR-owe wymaganie LAY-018): rysunek układany jest ponownie z węższymi polami (tekst łamie się
   częściej, także po łącznikach w długich słowach) i wygrywa wersja o największej skali na A4. Tryb `auto`
   (domyślny) działa od progu proporcji 2 : 1.
2. **Zwijanie** (LAY-019): długi, prosty diagram dzielony jest na grupy warstw ustawione w rzędach (LR) albo
   kolumnach (TB), połączone łącznikami w odstępach między rzędami. Domyślny sposób to powrót do początku: każdy
   rząd czyta się w tym samym kierunku, jak tekst. Wężyk jest opcją.
3. Zwijany jest tylko diagram, który da się przeciąć czysto: bez cykli i z najwyżej dwiema krawędziami między
   sąsiednimi grupami, bez dzielenia ramek. Pozostałe zostają bez zmian, bo łączniki przeskakujące przez rzędy
   byłyby trudniejsze do czytania niż szeroki rysunek.
4. Diagram LR zwija się, gdy jest szerszy niż pionowa kartka A4 i zwinięcie daje większą skalę albo proporcje
   bliższe kartki; diagram TB tylko dla większej skali, bo pionowa kartka dobrze mieści wysoki rysunek.
5. Oba mechanizmy można wyłączyć opcją, flagą albo linią w źródle (`%% layout: nofit`, `%% layout: nofold`).
6. **Zwijanie łańcuchów** (LAY-022, dopisane 2026-11-23): długi łańcuch pól wewnątrz diagramu, którego nie da się
   zwinąć w całości (cykle, rozgałęzienia), staje w kilku kolumnach (TB) albo rzędach (LR) niewidocznego bloku.
   To osobna opcja, **domyślnie wyłączona**: pionowa kartka dobrze mieści wysoki rysunek, a kolumny łańcucha
   zmieniają kolejność czytania, więc nie każdy autor tego chce. Włączona działa jak zwijanie: tylko przy zysku
   skali na A4 co najmniej 20%; sposób `return` (domyślny) albo `snake` z `WithFoldType`. Blok jest podgrafem
   bez ramki: wejście i wyjście przechodzą przez zwykłe punkty na jego obrysie, więc reszta układu się nie zmienia.
7. **Rozkładanie warstwy liści** (LAY-025, dopisane 2026-11-23): pole z co najmniej sześcioma liśćmi stawia co
   drugi warstwę dalej, zygzakiem. Też osobna opcja, domyślnie wyłączona, przy zysku skali co najmniej 20%.
   Realizacja to wydłużenie co drugiej krawędzi o warstwę (jak ręczne `--->` autora), więc kolejność, porządkowanie
   i trasowanie zostają wspólne; przebiegi dłuższych krawędzi mają między bliższymi liśćmi taki odstęp, żeby
   trafiać prosto w środki dalszych. Trzy podwarstwy odrzuciliśmy: dawały uskoki przy tym samym zysku.

## Skutki

- Wygląd części rysunków zależy od proporcji, a nie tylko od źródła; zmiana źródła może przestawić rysunek z jednego
  rzędu na kilka.
- Zwinięty rysunek nie jest porównywalny z rysunkiem Mermaid pod względem układu (treść ta sama).
- Diagramy encji mają własne dopasowanie (LAY-011) i nie są zwijane; tory (`swimlanes`) też nie.
