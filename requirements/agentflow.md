# Wymagania: przepływ agentów (agentflow)


| Id | Wymaganie |
|---|---|
| AGF-001 | Źródło zaczynające się od `agentflow` albo `agentflow-beta`, z opcjonalnym kierunkiem `TB`, `TD`, `BT`, `LR`, `RL` (domyślnie TB), daje przepływ agentów rysowany jak schemat blokowy; numery linii w błędach są numerami linii źródła. Komentarze `%%` są pomijane. |
| AGF-002 | Węzeł `id["podpis"]` z metadanymi `@{ … }` w jednej albo w kilku liniach: `shape` z aliasami `task` (zaokrąglony prostokąt), `tool` (podproces), `input` (równoległobok), `decision` (romb), `refdoc` (prostokąt z przerywanym obrysem), `action` (sześciokąt) albo nazwą kształtu schematu blokowego; `label` zastępuje podpis; pozostałe klucze (`description`, `params`, `returns`, `model` i inne) są przyjmowane i nie zmieniają rysunku. |
| AGF-003 | Krawędzie `-->` (kolejność), `-.-` (odwołanie, kropkowana bez grotu), `--x` (porażka), z podpisem `a -- tak --> b` i w łańcuchu `a --> b --> c`. |
| AGF-004 | Kontener `flow id["podpis"] … end` (zagnieżdżany) to podgraf; `global … end` zostawia swoje węzły na najwyższym poziomie; `id@{ view: "collapsed" }` (w linii kontenera albo później) zwija kontener do jednego węzła, do którego prowadzą krawędzie jego treści; metadane kontenera (`algorithm`) są przyjmowane. `connector id["podpis"]` to węzeł o kształcie pastylki, a narzędzie z `connectorRef: id` łączy się z nim linią kropkowaną. |
| AGF-005 | Błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, `end` bez kontenera, niezamknięty kontener, cudzysłów w podpisie z metadanych; błędy węzłów i krawędzi zgłasza parser schematu blokowego (np. kształt spoza profilu). |
