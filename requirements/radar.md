# Wymagania: wykres radarowy


| Id | Wymaganie |
|---|---|
| RDR-001 | Źródło zaczynające się od `radar-beta` (albo `radar`) daje wykres; `title` nadaje tytuł; `axis id["Podpis"], …` definiuje osie (podpis opcjonalny), także w kilku liniach; komentarze i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| RDR-002 | Seria to `curve id["Podpis"]{…}` z wartościami w kolejności osi albo parami `oś: wartość`, po jednej na każdą oś. `max` i `min` ustalają zakres (bez `max` jest nim największa wartość), `ticks` liczbę pierścieni siatki, `graticule circle` albo `polygon` jej kształt, `showLegend` obecność legendy. |
| RDR-003 | Rysunek: osie rozchodzą się ze środka, pierwsza w górę, kolejne zgodnie z ruchem wskazówek zegara, każda z podpisem na zewnątrz; seria jest wielokątem przez swoje wartości, z odległością od środka proporcjonalną do wartości w zakresie; serie różnią się kreskowaniem; legenda wymienia serie. |
| RDR-004 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nierozpoznana linia, mniej niż trzy osie, brak serii, seria o innej liczbie wartości, nieznana albo powtórzona oś, wartość niebędąca liczbą, zakres pusty, przekroczenie limitów (24 osie, 12 serii). |
