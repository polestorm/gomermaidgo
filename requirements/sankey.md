# Wymagania: wykres Sankeya


| Id | Wymaganie |
|---|---|
| SNK-001 | Źródło zaczynające się od `sankey-beta` (albo `sankey`) daje wykres; każda następna linia to przepływ `źródło,cel,wartość`; pole w cudzysłowach może zawierać przecinek, a podwojony cudzysłów oznacza cudzysłów; komentarze i puste linie są pomijane. |
| SNK-002 | Węzły stoją w kolumnach od lewej do prawej według najdłuższej drogi od węzłów bez dopływów; węzeł bez odpływów stoi w ostatniej kolumnie. Wysokość węzła jest proporcjonalna do większej z sum jego dopływów i odpływów, a szerokość wstęgi do wartości przepływu, w jednej skali dla całego wykresu. Węzły jednej kolumny się nie nakładają. |
| SNK-003 | Każdy węzeł ma obok nazwę i wartość; każdy przepływ ma wstęgę od prawej krawędzi źródła do lewej krawędzi celu; wstęgi jednego węzła leżą jedna pod drugą w kolejności położenia drugiego końca. |
| SNK-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, linia niebędąca przepływem, wartość niebędąca liczbą powyżej zera, przepływ węzła do samego siebie, cykl, więcej niż 500 przepływów, brak przepływów. |
