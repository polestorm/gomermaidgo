# Wymagania: reguły wyglądu układu grafu

Reguły ustalone w przeglądzie rysunków. Dotyczą wszystkich typów rysowanych układem warstwowym (schemat blokowy,
diagram stanów, diagram encji). Testy są w `internal/layered/appearance_test.go`.

| Id | Wymaganie |
|---|---|
| LAY-001 | Krawędź łamana (domyślna) składa się z odcinków równoległych do osi, połączonych zaokrąglonymi narożnikami: żaden odcinek nie jest skośny i żaden nie cofa się względem poprzedniego. Obowiązuje w czterech kierunkach układu. |
| LAY-002 | Krawędź nie ma drobnych uskoków: przesunięcie w poprzek między kolejnymi odcinkami jest zerem albo ma co najmniej 8 px w kierunku ze źródła; w kierunkach wymuszonych nie schodzi poniżej progu prostowania (2,5 px). |
| LAY-003 | Dwie krawędzie nie przecinają się odcinkami prostymi: w miejscu przecięcia krawędź narysowana później ma półokrągły mostek, o ile jest na niego miejsce (przecięcie leży dalej niż 14 px od narożnika i końca obu krawędzi). Dotyczy także krawędzi z różnych poziomów zagnieżdżenia podgrafów i torów. |
| LAY-004 | Pola stojące jedno pod drugim mają tę samą szerokość i oś, gdy szersze da się złamać do szerokości węższego. |
| LAY-005 | Krawędzie rozchodzące się z jednego pola skręcają na tej samej wysokości; z rombu wychodzą w różnych punktach, z boków przy wierzchołku. |
| LAY-006 | Stan złożony ma tytuł w pasku u góry i wąski pasek u dołu; proste strzałki bez etykiety w jego wnętrzu są krótsze niż na poziomie głównym. |
| LAY-007 | Pętla własna stoi z boku pola i nie przesuwa osi: sąsiad wyrównuje się do środka samego pola. |
| LAY-008 | Ramki bez połączeń między sobą (regiony współbieżne, osobne podgrafy) są upakowane w kolejności ze źródła: niższa staje pod poprzednią, obok wysokiej. |
| LAY-009 | Romb decyzji, z którego jedna gałąź biegnie prosto w osi pola: pozostałe gałęzie wychodzą z bocznych wierzchołków (najwyżej jedna z każdego), biegną w poprzek na wysokości środka pola i skręcają raz. Warunki: gałąź skręca poza obrysem rombu i w rzędzie rombu nic nie stoi na jej drodze; inaczej wychodzi z boku przy dolnym wierzchołku (LAY-005). |
| LAY-010 | Rysunek nie ma pustych kolumn: pas w poprzek osi warstw, w którym w żadnej warstwie nic nie stoi, jest zamykany do zwykłego odstępu między polami. Nie dotyczy torów, których szerokość wyznacza pas. |
| LAY-011 | Diagram encji jest układany pod proporcje strony, bo jego kierunek nie niesie znaczenia: gdy warstwa jest szersza niż limit, pola, których przesunięcie wydłuża najmniej krawędzi (najpierw liście), schodzą do warstwy niżej. Spośród kilku limitów wygrywa układ, który mieści się na stronie w największej skali; układ bez limitu zostaje, gdy rysunek i tak się mieści albo limit nie daje wyraźnej poprawy. Schematu blokowego i stanów reguła nie dotyczy: tam warstwa oznacza kolejność. |
