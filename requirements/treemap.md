# Wymagania: mapa drzewa


| Id | Wymaganie |
|---|---|
| TRM-001 | Źródło zaczynające się od `treemap-beta` (albo `treemap`) daje mapę; `title` nadaje tytuł; hierarchię wyznacza wcięcie; `"Nazwa"` jest sekcją, `"Nazwa": wartość` liściem; skrót `:::` i `classDef` są przyjmowane i pomijane; komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| TRM-002 | Pole każdego liścia jest proporcjonalne do jego wartości, a sekcja obejmuje swoje dzieci; prostokąty jednego rodzica się nie nakładają i mieszczą w jego polu. Podział dąży do prostokątów bliskich kwadratom. |
| TRM-003 | Każda nazwa i wartość liścia jest na rysunku: w polu, w rozmiarze, w którym się mieści, a gdy pole jest za małe, w liście pod mapą, z numerem wpisanym w pole. Nazwa sekcji stoi w pasku u góry jej pola, a przy prawej krawędzi paska suma wartości jej liści. |
| TRM-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, linia niebędąca węzłem, wartość ujemna albo niebędąca liczbą, dziecko liścia, więcej niż 300 węzłów, brak wartości powyżej zera. |
