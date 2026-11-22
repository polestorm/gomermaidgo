# Wymagania: wykres Sankeya


| Id | Wymaganie |
|---|---|
| SNK-001 | Źródło zaczynające się od `sankey-beta` (albo `sankey`) daje wykres; każda następna linia to przepływ `źródło,cel,wartość`; pole w cudzysłowach może zawierać przecinek, a podwojony cudzysłów oznacza cudzysłów; komentarze i puste linie są pomijane. |
| SNK-002 | Węzły stoją w kolumnach od lewej do prawej według najdłuższej drogi od węzłów bez dopływów; węzeł bez odpływów stoi w ostatniej kolumnie. Wysokość węzła jest proporcjonalna do większej z sum jego dopływów i odpływów, a szerokość wstęgi do wartości przepływu, w jednej skali dla całego wykresu. Węzły jednej kolumny się nie nakładają. |
| SNK-003 | Każdy węzeł ma obok nazwę i wartość; każdy przepływ ma wstęgę od prawej krawędzi źródła do lewej krawędzi celu; wstęgi jednego węzła leżą jedna pod drugą w kolejności położenia środka drugiego końca. |
| SNK-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, linia niebędąca przepływem, wartość niebędąca liczbą powyżej zera, przepływ węzła do samego siebie, cykl, więcej niż 500 przepływów, brak przepływów. |
| SNK-005 | Węzły ustawiają się w pionie względem sąsiadów (ku średniej ważonej środków źródeł i celów), więc wstęgi biegną możliwie prosto i nie krzyżują się, gdy da się tego uniknąć. Przepływ przez kilka kolumn zajmuje miejsce w kolumnach pośrednich: węzły tych kolumn nie stoją na jego drodze, a wstęga jest jednym kształtem. |
| SNK-006 | Linia `%% layout: wrap` włącza łamanie nazw węzłów na słowach: w ostatniej kolumnie do szerokości najdłuższego słowa wśród jej nazw (długa nazwa nie zwęża wtedy wykresu), we wcześniejszych do odstępu między kolumnami (nazwa nie wchodzi na następną kolumnę). Bez tej linii nazwy stoją w jednej linii. |
