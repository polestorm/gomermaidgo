# Profil składni

Obsługiwany zakres składni Mermaid. Wszystko poza tabelą „Obsługiwane” daje błąd z numerem linii.

## Diagram sekwencji (`sequenceDiagram`)

### Obsługiwane

| Element | Zapis | Wymaganie |
|---|---|---|
| nagłówek, komentarze | `sequenceDiagram`, `%% komentarz` | SEQ-001 |
| uczestnik, aktor, alias | `participant A`, `actor A`, `participant A as Etykieta` | SEQ-002 |
| uczestnik domyślny | pierwsze użycie w komunikacie albo notatce | SEQ-002 |
| komunikat | `A->>B: tekst`; strzałki `->`, `-->`, `->>`, `-->>`, `-x`, `--x`, `-)`, `--)`, `<<->>`, `<<-->>` | SEQ-003 |
| komunikat do siebie | `A->>A: tekst` | SEQ-003 |
| notatka | `Note right of A: t`, `Note left of A: t`, `Note over A: t`, `Note over A,B: t` | SEQ-004 |
| aktywacja | `activate A`, `deactivate A`, `A->>+B: t`, `B-->>-A: t` | SEQ-005 |
| bloki | `loop`, `alt`/`else`, `opt`, `par`/`and`, `critical`/`option`, `break`, zamykane `end`; dowolnie zagnieżdżone | SEQ-006, SEQ-007 |
| wyróżnienie tła | `rect rgb(r, g, b)`, `rect rgba(r, g, b, a)`, `rect #rrggbb` … `end` | SEQ-007 |
| numeracja | `autonumber`, `autonumber 10`, `autonumber 10 5` (początek, krok) | SEQ-008 |
| grupy uczestników | `box [kolor] [tytuł]` … `end`; kolor: `rgb()`, `rgba()`, `#hex`, nazwa podstawowa, `transparent` | SEQ-014 |
| tworzenie i kończenie uczestnika | `create participant A`, `create actor A`, `destroy A` (każde przed komunikatem, którego dotyczy) | SEQ-015 |
| dostępność | `accTitle: tekst`, `accDescr: tekst`, `accDescr { … }` → `title` i `desc` w SVG | SEQ-016 |
| tytuł | `title Tekst` | SEQ-010 |
| podział linii w tekście | `<br/>`, `<br>` | SEQ-010 |

### Poza profilem (błąd `sequence-unsupported`)

`link`, `links`, `properties`, `details`.

### Świadome różnice wobec Mermaid

- średnik nie rozdziela instrukcji i kody encji (`#35;`) nie są rozwijane: tekst trafia do rysunku dosłownie;
- nie ma zawijania tekstu (`wrap`), dyrektyw `%%{init}%%` ani bloku konfiguracji `---`;
- limity: 50 uczestników, 2000 instrukcji, 20 poziomów zagnieżdżenia bloków;
- dezaktywacja nieaktywnego uczestnika jest błędem, także w zapisie skróconym;
- uczestnik zakończony przez `destroy` kończy się krzyżykiem i nie ma stopki; Mermaid rysuje w tym miejscu jego pole
  po raz drugi. Użycie takiego uczestnika po zakończeniu jest błędem;
- etykieta komunikatu do siebie stoi nad pętlą; u ostatniego uczestnika kończy się nad pętlą i rośnie w lewo, żeby
  nie poszerzać rysunku;
- w grupie `box` z kolorem podkłady etykiet przechodzących przez linie życia pozostają w kolorze tła rysunku;
- wygląd jest własny; motywy i zmienne motywu nie są obsługiwane. Kryterium wyglądu jest czytelność, nie zgodność
  z notacją UML: ramki bloków (`alt`, `loop`, `opt` i pozostałe) są kropkowane, żeby nie konkurowały z uczestnikami
  i komunikatami, a komunikat do siebie jest jedną krzywą;
- kolory i cienie są opcjami wywołującego (`WithColors`, `WithShadows`), domyślnie wyłączonymi: w diagramie sekwencji
  uczestnicy dostają kolejne kolory palety, w grafach kolor mówi, czym jest węzeł (krok, decyzja, początek i koniec,
  dane, wejście i wyjście); cień jest zwykłym kształtem, nie filtrem.

## Schemat blokowy (`flowchart`, `graph`)

Stan: rysowany układem warstwowym (`internal/layered`).

### Obsługiwane

| Element | Zapis | Wymaganie |
|---|---|---|
| nagłówek i kierunek | `flowchart TD`, `graph LR`; kierunki `TB`, `TD`, `BT`, `LR`, `RL` | FLOW-001 |
| węzeł bez kształtu | `A` (prostokąt, etykietą jest identyfikator) | FLOW-002 |
| kształty | `A[t]`, `A(t)`, `A([t])`, `A[[t]]`, `A[(t)]`, `A((t))`, `A(((t)))`, `A>t]`, `A{t}`, `A{{t}}`, `A[/t/]`, `A[\t\]`, `A[/t\]`, `A[\t/]` | FLOW-002 |
| krawędzie | `-->`, `---`, `-.->`, `-.-`, `==>`, `===`, `--o`, `--x`, `<-->`, `o--o`, `x--x`, `~~~` | FLOW-003 |
| etykieta krawędzi | `A -->\|tekst\| B`, `A -- tekst --> B`, `A == tekst ==> B`, `A -. tekst .-> B` | FLOW-004 |
| długość krawędzi | dodatkowe znaki: `--->`, `----`, `-..->`, `===>`, `~~~~` | FLOW-005 |
| łańcuchy i grupy | `A --> B --> C`, `A & B --> C & D` | FLOW-006 |
| podgrafy | `subgraph id [tytuł]`, `subgraph id`, `subgraph tytuł ze spacjami` … `end`; `direction` w podgrafie; krawędź do podgrafu | FLOW-007 |
| tekst | cudzysłów dla nawiasów i znaków specjalnych, `<br/>` | FLOW-008 |
| średnik | rozdziela instrukcje, może kończyć linię | FLOW-009 |
| rysunek | kształty, układ w czterech kierunkach, teksty z łamaniem | FLOW-012, FLOW-013 |
| wygląd ze źródła | `classDef nazwa właściwości`, `class A,B nazwa`, `A:::nazwa`, `style A właściwości`, `linkStyle 0,1 właściwości`, `linkStyle default …`; właściwości: `fill`, `stroke`, `color`, `stroke-width`, `stroke-dasharray` | FLOW-017 |
| identyfikatory, Markdown, dostępność | `krok-1.a`; ``"`tekst **pogrubiony**`"`` (bez wyróżnień na rysunku); `accTitle`, `accDescr` | FLOW-018 |
| kierunek podgrafu | `direction LR` w podgrafie niepołączonym krawędziami z resztą | FLOW-019 |

### Poza profilem (błąd `flowchart-unsupported`)

Zapis kształtu `@{ shape: … }`, `interpolate`, `flowchart-elk`, teksty Markdown rozciągnięte na kilka linii źródła.

### Świadome różnice wobec Mermaid

- identyfikator węzła składa się z liter, cyfr i podkreślenia, z łącznikiem albo kropką między nimi;
- właściwości stylu inne niż pięć honorowanych (czcionka, zaokrąglenia, przejścia) są pomijane; błędna wartość
  honorowanej właściwości jest błędem. Przy ciemnym wypełnieniu kolor tekstu trzeba podać samemu (`color`);
- `click` jest przyjmowane i pomijane: rysunek statyczny nie ma odnośników;
- tekst Markdown traci znaczniki pogrubienia i kursywy, bo rysunek ma jeden krój;
- węzeł należy do podgrafu, w którym po raz pierwszy wystąpił wewnątrz jakiegokolwiek podgrafu; późniejsze wystąpienia
  go nie przenoszą;
- krawędź między podgrafem a jego własną zawartością jest błędem;
- tekst w nawiasach ostrych (`<znaczniki>`) zostaje na rysunku dosłownie; Mermaid traktuje go jak znacznik HTML i usuwa.
  Dotyczy też diagramu stanów. Jedynym rozpoznawanym znacznikiem jest `<br/>`;
- kody encji (`#35;`) nie są rozwijane;
- limity: 500 węzłów, 1000 krawędzi, 10 poziomów zagnieżdżenia.

### Ograniczenia rysunku (wspólne ze stanami)

- `direction` w podgrafie i w stanie złożonym zmienia kierunek wnętrza tylko wtedy, gdy żadna krawędź nie łączy
  wnętrza z resztą diagramu (tak samo rozstrzyga to Mermaid); inaczej wnętrze ma kierunek otoczenia;
- krawędź wychodząca z podgrafu biegnie od swojego punktu na ramce; kolejność punktów na ramce wynika z układu
  wnętrza i nie jest uzgadniana z otoczeniem, więc przy kilku krawędziach do różnych celów możliwe jest przecięcie;
- regiony współbieżne stoją obok siebie w ramkach przerywanych; Mermaid układa je jeden pod drugim z linią podziału;
- krawędź wskazująca podgraf kończy się na jego ramce; krawędź do węzła wewnątrz podgrafu przechodzi przez punkt na
  ramce i jest prowadzona we wnętrzu między węzłami (FLOW-014); może przejść pod tytułem ramki, który ma wtedy podkład;
- krawędź niewidoczna (`~~~`) wpływa tylko na rozmieszczenie;
- notatka diagramu stanów stoi obok swojego stanu po wskazanej stronie osi kolejności: w układzie pionowym po lewej
  albo po prawej, w układzie poziomym odpowiednio nad albo pod stanem;
- etykiety szersze niż 200 px (150 px w rombie i kole) są łamane na granicach wyrazów.

## Diagram stanów (`stateDiagram-v2`, `stateDiagram`)

Stan: rysowany tym samym układem warstwowym.

### Obsługiwane

| Element | Zapis | Wymaganie |
|---|---|---|
| nagłówek | `stateDiagram-v2`, `stateDiagram` | STATE-001 |
| przejście | `A --> B`, `A --> B : etykieta`; `[*]` jako stan początkowy i końcowy | STATE-002 |
| opis stanu | `state "Opis" as Id`, `Id : opis` (kolejne linie dopisują) | STATE-003 |
| stan bez opisu | użyty w przejściu albo sam identyfikator w linii | STATE-004 |
| stan złożony | `state Id { … }`, `state "Opis" as Id { … }`, zagnieżdżone; przejście do stanu złożonego | STATE-005 |
| wybór, rozwidlenie, złączenie | `state Id <<choice>>`, `<<fork>>`, `<<join>>` | STATE-006 |
| notatka | `note right of Id : tekst`, `note left of Id` … `end note` | STATE-007 |
| rysunek | kształty stanów, układ, teksty | STATE-010, STATE-011 |
| kierunek | `direction LR` na poziomie diagramu albo stanu złożonego | STATE-008 |
| regiony współbieżne | `--` wewnątrz stanu złożonego | STATE-012 |
| wygląd ze źródła | `classDef`, `class A nazwa`, `A:::nazwa`, `style` | STATE-013 |
| dostępność | `accTitle`, `accDescr` | STATE-014 |

### Poza profilem (błąd `state-unsupported`)

`scale`, `hide`,
`state Id` bez opisu, stereotypu i klamry (forma nieopisana w dokumentacji; renderer odniesienia jej nie rysuje).

### Świadome różnice wobec Mermaid

- identyfikator stanu składa się z liter, cyfr i podkreślenia; łącznik jest błędem, tak jak w Mermaid;
- przejście między stanem złożonym a jego własną zawartością jest błędem.
