# Profil składni

Obsługiwany zakres składni Mermaid. Wszystko poza tabelą „Obsługiwane” daje błąd z numerem linii.

## Wspólne dla wszystkich typów

- Blok nagłówkowy YAML między liniami `---` jest przyjmowany; `title` daje tytuł nad rysunkiem (COM-001).
- Dyrektywa `%%{init: … }%%`, także wieloliniowa, jest przyjmowana (COM-002).
- Z bloku nagłówkowego i dyrektywy czytany jest motyw (`theme`) i cztery zmienne motywu (`primaryColor`,
  `lineColor`, `primaryTextColor`, `background`); reszta konfiguracji jest pomijana. Motywy są własne, pod nazwami
  Mermaid: `neutral` (domyślny), `default` (kolorowy), `forest`, `dark`, `base`. Wywołujący może narzucić motyw
  (`WithTheme`, flaga `-t`), co wyłącza wybór ze źródła (THM-001…004, ADR 0005).
- Numery linii w komunikatach błędów liczą się od początku źródła, razem z nagłówkiem (COM-003).

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
  po raz drugi. Gdy komunikat kończący ma grot-krzyżyk (`-x`), grot i znak zakończenia są jednym krzyżykiem.
  Użycie takiego uczestnika po zakończeniu jest błędem;
- etykieta komunikatu do siebie stoi nad pętlą; u ostatniego uczestnika kończy się nad pętlą i rośnie w lewo, żeby
  nie poszerzać rysunku;
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
| identyfikatory, Markdown, dostępność | `krok-1.a`; ``"`tekst **pogrubiony** i *pochylony*`"``; `accTitle`, `accDescr` | FLOW-018 |
| kierunek podgrafu | `direction LR` w podgrafie niepołączonym krawędziami z resztą | FLOW-019 |
| tory | komentarz `%% layout: swimlanes` przed albo po nagłówku: podgrafy najwyższego poziomu stają się pasami | FLOW-021 |

### Poza profilem (błąd `flowchart-unsupported`)

Kształty zapisu `@{ shape: … }`, których nie rysujemy pod starą składnią (np. `doc`, `cloud`, `delay`; pozostałe
działają, FLOW-022), `interpolate`, `flowchart-elk`.

### Świadome różnice wobec Mermaid

- ikony Font Awesome w tekście (`fa:fa-user`) nie są rysowane: znacznik znika z etykiety, a etykieta złożona
  z samej ikony dostaje jej nazwę; rysunek nie zawiera cudzych krojów ikon;
- tekst na krawędzi może stać także bez odstępów (`A --tekst--> B`);
- identyfikator węzła składa się z liter, cyfr i podkreślenia, z łącznikiem albo kropką między nimi;
- właściwości stylu inne niż pięć honorowanych (czcionka, zaokrąglenia, przejścia) są pomijane; błędna wartość
  honorowanej właściwości jest błędem. Przy ciemnym wypełnieniu kolor tekstu trzeba podać samemu (`color`);
- `click` jest przyjmowane i pomijane: rysunek statyczny nie ma odnośników;
- znaczniki `<b>`, `<strong>`, `<i>` i `<em>` w etykiecie schematu blokowego dają pogrubienie i kursywę, także gdy
  obejmują złamanie linii; pozostałe znaczniki HTML są rysowane jako tekst (renderer odniesienia je usuwa);
- tekst Markdown może zajmować kilka linii źródła, koniec linii jest złamaniem tekstu; wyróżnienie musi się
  zamknąć w tej samej linii; pogrubienie z kursywą naraz jest rysowane jako pogrubienie
  z atrybutem kursywy, mierzone odmianą pogrubioną;
- węzeł należy do podgrafu, w którym po raz pierwszy wystąpił wewnątrz jakiegokolwiek podgrafu; późniejsze wystąpienia
  go nie przenoszą;
- krawędź między podgrafem a jego własną zawartością jest błędem;
- tekst w nawiasach ostrych (`<znaczniki>`) zostaje na rysunku dosłownie; Mermaid traktuje go jak znacznik HTML i usuwa.
  Dotyczy też diagramu stanów. Jedynym rozpoznawanym znacznikiem jest `<br/>`;
- kody encji (`#35;`) nie są rozwijane;
- limity: 500 węzłów, 1000 krawędzi, 10 poziomów zagnieżdżenia.

### Tory

Tory nie mają w Mermaid własnej składni. Dyrektywa jest komentarzem, więc to samo źródło w innych narzędziach rysuje
się jak zwykły schemat z podgrafami. Pas biegnie przez cały rysunek wzdłuż kierunku diagramu (kolumny przy `TB`,
wiersze przy `LR`), nagłówek z tytułem stoi przy początku. Warstwy są wspólne dla wszystkich pasów: położenie kroku
wzdłuż pasa mówi o kolejności w całym procesie. Krawędź zmienia pas raz, przy swoim początku. Pole, które
w swoim rzędzie jest jedynym polem toru, stoi pośrodku pasa, więc kroki jednej roli tworzą kolumnę. Tory są
tylko w schemacie blokowym: stan złożony nie jest rolą, więc diagram stanów ich nie ma. W torach nie ma
podgrafów zagnieżdżonych ani krawędzi do samego toru (błąd), a `direction` w podgrafie nie ma znaczenia.

### Ograniczenia rysunku (wspólne ze stanami)

- `direction` w podgrafie i w stanie złożonym zmienia kierunek wnętrza tylko wtedy, gdy żadna krawędź nie łączy
  wnętrza z resztą diagramu (tak samo rozstrzyga to Mermaid); inaczej wnętrze ma kierunek otoczenia;
- krawędź wychodząca z podgrafu biegnie od swojego punktu na ramce; punkty na ramce stoją w kolejności drugich
  końców krawędzi w otoczeniu, o ile nie dodaje to przecięć we wnętrzu podgrafu;
- regiony współbieżne są osobnymi polami z lekko przyciemnionym tłem i przerywanym obrysem, bez tytułu;
- poziom złożony z samych ramek bez połączeń (regiony, osobne podgrafy) jest pakowany swobodnie: niższe ramki stają
  jedna pod drugą obok wyższej, w kolejności źródła; miarą jest skala, w jakiej całość mieści się na stronie;
- części diagramu (i wnętrza podgrafu albo stanu złożonego) niepołączone krawędziami układają się w rzędy tak, żeby
  całość miała proporcje zbliżone do strony (3:4); w rzędzie stoją obok siebie, w kolejności źródła;
- krawędź wskazująca podgraf kończy się na jego ramce; krawędź do węzła wewnątrz podgrafu przechodzi przez punkt na
  ramce i jest prowadzona we wnętrzu między węzłami (FLOW-014); może przejść pod tytułem ramki, który ma wtedy podkład;
- krawędź niewidoczna (`~~~`) wpływa tylko na rozmieszczenie;
- krawędź z etykietą jest o jedną warstwę dłuższa: etykieta zajmuje warstwę jak pole, obok pól innych gałęzi,
  a krawędzie bez etykiet zostają krótkie. Krawędzie wielokrotne między tą samą parą pól też są o warstwę dłuższe;
- dodatkowe znaki linii (`--->`) wydłużają krawędź o warstwę każdy, także gdy cel nie ma dalszych połączeń;
  renderer odniesienia w takim przypadku ustawia cele w jednej linii;
- krawędzie są łamane pod kątem prostym, z zaokrąglonymi narożnikami; odcinki poprzeczne w jednym odstępie między
  warstwami dostają osobne tory, a odstęp rośnie z ich liczbą. Opcja `WithCurvedEdges` (w programie `-curved`)
  przywraca łagodne łuki. Krawędź łamana skręca raz, przy końcu, z którego rozchodzi się kilka krawędzi; jej
  etykieta stoi w osi drugiego końca, a pole z kilkoma gałęziami stoi pośrodku nich;
- romb decyzji, z którego jedna gałąź biegnie prosto w dół: pozostałe gałęzie wychodzą z bocznych wierzchołków
  i biegną w bok na wysokości środka rombu, jak w klasycznym schemacie blokowym (główny przebieg osią,
  odgałęzienie w bok). Gdy żadna gałąź nie biegnie prosto albo coś stoi na drodze, gałęzie wychodzą z boków przy
  dolnym wierzchołku. Krawędzi wstecznej (w górę) i łuków (`WithCurvedEdges`) reguła nie dotyczy;
- w torach kolejność pól jednego toru jest po pogrupowaniu poprawiana tak, żeby nie dodawać przecięć; punkty
  pośrednie krawędzi, której oba końce mają po kilka krawędzi, stoją w jednej osi przy dolnym końcu;
- diagram encji jest układany pod proporcje strony: zbyt szeroka warstwa oddaje pola (najpierw encje z jedną
  relacją) warstwie niżej, więc duży model rośnie w dół zamiast w bok; mały diagram zostaje bez zmian;
- puste kolumny są zamykane: pas, w którym w żadnej warstwie nic nie stoi, kurczy się do zwykłego odstępu,
  a krawędzie, które go przecinały, są krótsze;
- gdzie krawędzie łamane się przecinają, krawędź zapisana w źródle później przeskakuje nad wcześniejszą półokrągłym
  mostkiem, o ile obok jest na niego miejsce; dotyczy to także krawędzi z różnych poziomów zagnieżdżenia
  podgrafów;
- punkt zaczepienia krawędzi przesuwa się w oś jej następnego punktu, jeśli ten leży w zasięgu boku pola: krawędź
  biegnie wtedy prosto, bez drobnego uskoku;
- zakończenie krawędzi kółkiem albo krzyżykiem stoi przy polu, a linia kończy się na nim, nie na polu;
- pola połączone w prosty ciąg (jedna krawędź wychodząca i jedna wchodząca) mają ten sam rozmiar w poprzek
  kierunku diagramu, a pola jednej warstwy ten sam rozmiar wzdłuż niego, o ile różnią się najwyżej o 40%;
  najszersze pole o jednej linii tekstu jest łamane na dwie, jeśli zmieści się wtedy w szerokości sąsiada;
- pole z jednym sąsiadem staje w jego osi przed polem, które uśrednia położenie kilku sąsiadów;
- pola są płaskie, o szerokości co najmniej 140 px przy piśmie 16 px; z rombu i koła krawędzie wychodzą osobno,
  z boków przy wierzchołku;
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
- stan złożony ma tytuł w osobnym pasku u góry ramki, z lekkim tłem i linią pod spodem, a ramka ma mocniej
  zaokrąglone narożniki; u dołu ramkę zamyka wąski pasek (4 px) w kolorze paska tytułu, bez tekstu, żeby
  dużą ramkę stanu dało się rozpoznać także od spodu; podgraf schematu blokowego zachowuje tytuł przy lewej
  krawędzi;
- wewnątrz stanu złożonego (i regionu) proste strzałki bez etykiety są krótsze niż na poziomie głównym
  (odstęp warstw 26 zamiast 44 px); odstęp przy etykiecie i przy krawędzi ze skrętem zostaje bez zmian;
- przejście między stanem złożonym a jego własną zawartością jest błędem.

## Wykres Gantta (`gantt`)

### Obsługiwane

| Element | Zapis | Wymaganie |
|---|---|---|
| nagłówek, tytuł, sekcje | `gantt`, `title Tekst`, `section Nazwa` | GANTT-001 |
| zadanie | `Nazwa : [znaczniki,] [id,] [początek,] koniec` | GANTT-002 |
| początek | data w zapisie `dateFormat`, `after id1 id2`, albo brak (po poprzednim zadaniu) | GANTT-002 |
| koniec | data, czas trwania (`3d`, `2w`, `12h`, `1.5d`, `6M`, `1y`), `until id` | GANTT-002 |
| znaczniki | `done`, `active`, `crit`, `milestone` | GANTT-003 |
| zapis dat | `dateFormat` ze znaczników `YYYY`, `YY`, `MM`, `M`, `DD`, `D`, `HH`, `mm`, `ss` | GANTT-002 |
| dni wyłączone | `excludes weekends`, nazwy dni, daty `YYYY-MM-DD`; `weekend friday`; `inclusiveEndDates` | GANTT-004 |
| podziałka | `axisFormat` (strftime: `%Y %y %m %d %e %H %M %S %j %b %B %a %A`), `tickInterval 1day`/`week`/`month`/`year` | GANTT-006 |
| dostępność | `accTitle`, `accDescr` | GANTT-001 |

### Poza profilem (błąd `gantt-unsupported`)

`includes`, `displayMode compact`, znaczniki pionowe `vert`.

### Świadome różnice wobec Mermaid

- znacznik „dziś” nie jest rysowany nigdy (`todayMarker` jest przyjmowane i pomijane): ten sam tekst ma zawsze dawać
  ten sam rysunek;
- czas trwania przy `excludes` liczy czas pracy: dni wyłączone się nie liczą, także gdy zadanie na takim dniu się
  zaczyna; koniec podany datą nie jest przesuwany;
- `tickInterval` jest honorowany, o ile opisy podziałki się nie stykają; inaczej odstęp rośnie;
- nazwy miesięcy i dni w `axisFormat` są angielskie; `topAxis` i `click` są przyjmowane i pomijane;
- nazwa zadania kończy się na pierwszym dwukropku;
- limity: 500 zadań, lata 1900–2200, rozpiętość do 100 lat.

## Diagram związków encji (`erDiagram`)

Stan: rysowany układem warstwowym; encja jest polem z nagłówkiem i tabelą atrybutów.

### Obsługiwane

| Element | Zapis | Wymaganie |
|---|---|---|
| nagłówek, kierunek | `erDiagram`, `direction LR` | ER-001 |
| encja | `NAZWA`, `"Nazwa ze spacją"`, `NAZWA["Opis"]`, `NAZWA { … }` | ER-002 |
| atrybut | `typ nazwa`, `typ nazwa PK`, `typ nazwa PK, FK "komentarz"` | ER-002 |
| związek | `A \|\|--o{ B : etykieta`; liczności `\|o`, `\|\|`, `}o`, `}\|` i lustrzane; linia `--` albo `..` | ER-003 |
| wygląd ze źródła | `classDef`, `class`, `NAZWA:::klasa`, `style` | ER-004 |
| dostępność | `accTitle`, `accDescr` | ER-001 |

### Poza profilem

Związki zapisane słowami (`only one to zero or more`; błąd `er-unsupported`), atrybuty o typach ze spacją.

### Świadome różnice wobec Mermaid

- nazwa encji bez cudzysłowu składa się z liter, cyfr, podkreślenia i łącznika;
- etykieta związku jest opcjonalna;
- limit 100 atrybutów w encji; limity grafu jak w schemacie blokowym.


## Wykres kołowy (`pie`)

### Obsługiwane

| Element | Składnia | Wymaganie |
|---|---|---|
| nagłówek | `pie`, `pie showData`, `pie title Tekst`, `pie showData title Tekst` | PIE-001 |
| tytuł i wartości w legendzie | `title Tekst`, `showData` w osobnej linii | PIE-001, PIE-004 |
| wycinek | `"etykieta" : 42.5` | PIE-002 |
| dostępność | `accTitle`, `accDescr` | PIE-001 |

### Poza profilem

Konfiguracja w bloku `%%{init}%%` i `---` (położenie tekstu, szerokość obrysu), motywy.

### Świadome różnice wobec Mermaid

- wycinki są w kolejności ze źródła, od godziny dwunastej zgodnie z ruchem wskazówek zegara;
- wypełnienia domyślne to odcienie szarości o naprzemiennej jasności, żeby wykres dało się czytać w druku
  czarno-białym; `WithColors` daje kolory palety;
- udział procentowy stoi na wycinku tylko wtedy, gdy wycinek ma co najmniej 4% koła; mniejszego nie da się
  podpisać bez nachodzenia na sąsiadów (renderer odniesienia podpisuje każdy);
- udział jest zaokrąglany do liczby całkowitej, a poniżej 10% do jednego miejsca po przecinku;
- etykieta wycinka nie może być pusta ani się powtarzać, a wartość ujemna jest błędem; wykres bez wycinka
  o wartości powyżej zera jest błędem; limit to 24 wycinki.

## Diagram klas (`classDiagram`)

Rysowany tym samym układem warstwowym co schemat blokowy; obowiązują te same reguły krawędzi i te same
ograniczenia rysunku.

### Obsługiwane

| Element | Składnia | Wymaganie |
|---|---|---|
| nagłówek | `classDiagram`, `classDiagram-v2` | CLS-001 |
| klasa | `class Nazwa`, `class Nazwa["Etykieta"]`, `class Lista~T~` | CLS-001 |
| składowe | `class Nazwa { … }` po jednej w linii; `Nazwa : +pole`; `Nazwa : +metoda(arg) Typ` | CLS-002 |
| klasyfikatory | `metoda()*` (abstrakcyjna, kursywa), `metoda()$` i `pole$` (statyczna, podkreślenie) | CLS-003 |
| adnotacja | `<<interface>>` w bloku klasy; `<<interface>> Nazwa` | CLS-003 |
| relacje | `<\|--`, `*--`, `o--`, `-->`, `--`, `..>`, `..\|>`, `..` i zapisy odwrócone | CLS-004 |
| etykieta i liczności | `A "1" --> "0..*" B : etykieta` | CLS-005 |
| kierunek, notatka | `direction LR`; `note for Nazwa "tekst"` | CLS-006 |
| wygląd | `classDef`, `class Nazwa:::styl`, `cssClass "A,B" styl`, `style Nazwa …` | CLS-007 |
| dostępność, odsyłacze | `accTitle`, `accDescr`; `click`, `link`, `callback` (pomijane) | CLS-006 |
| przestrzeń nazw | `namespace Nazwa { class A … }` | CLS-009 |
| blok w jednej linii, notatka | `class A { <<interface>> }`; `note "tekst"` | CLS-009 |
| interfejs „lizak” | `Interfejs ()-- Klasa`, `Klasa --() Interfejs` | CLS-009 |

### Poza profilem (błąd `class-unsupported`)

Zagnieżdżone przestrzenie nazw, nazwy klas w odwrotnych apostrofach, konfiguracja `hideEmptyMembersBox`.

### Świadome różnice wobec Mermaid

- klasa bez składowych jest prostym polem z nazwą; renderer odniesienia rysuje zawsze trzy przedziały;
- klasa ze składowymi ma dwa przedziały; pusty zostaje jako wąski pasek;
- asocjacja skierowana i zależność mają otwarty grot (dwie kreski), jak w UML, a nie grot wypełniony;
- liczności stoją obok linii tuż przy polu; krawędź z licznościami przy obu końcach jest o warstwę dłuższa,
  żeby opisy się nie stykały;
- identyfikatorem klasy jest nazwa bez parametru typu: `Lista~T~` i `Lista` to ta sama klasa;
- kolor nadany klasie stylem obejmuje całe pole, także linie przedziałów;
- interfejs „lizak” jest zwykłą relacją z kółkiem przy końcu, a sam interfejs polem klasy (w Mermaid samo kółko
  z nazwą).

## Oś czasu (`timeline`)

### Obsługiwane

| Element | Składnia | Wymaganie |
|---|---|---|
| nagłówek, tytuł | `timeline`; `title Tekst` | TML-001 |
| okres ze zdarzeniami | `2024 : zdarzenie : zdarzenie`; kolejna linia `: zdarzenie` dopisuje zdarzenia | TML-002 |
| sekcja | `section Tytuł` | TML-003 |
| układ listowy | komentarz `%% layout: list` | TML-006 |
| dostępność | `accTitle`, `accDescr` | TML-001 |

### Poza profilem

Konfiguracja w bloku `%%{init}%%` (`disableMulticolor`, motywy).

### Układy

Domyślny układ jest taki jak w Mermaid: okresy w rzędzie od lewej do prawej, pod nimi oś ze strzałką, pod osią
zdarzenia każdego okresu jedno pod drugim, nagłówek sekcji nad jej okresami. Dłuższa oś nie mieści się wtedy
w szerokości strony i jest pomniejszana.

Układ listowy (`%% layout: list`) prowadzi oś z góry na dół: okresy stoją przy osi, a zdarzenia okresu po prawej,
zawijane w szerokości strony. Rysunek rośnie w dół i mieści się na stronie przy dowolnej liczbie okresów. Dyrektywa
jest komentarzem, więc to samo źródło w innych narzędziach rysuje układ domyślny.

### Świadome różnice wobec Mermaid

- `<br/>` w tekście okresu i zdarzenia łamie linię (renderer odniesienia rysuje znacznik dosłownie);
- kierunek podany po nagłówku (`timeline LR`, `timeline TD`) jest przyjmowany i pomijany;
- domyślnie rysunek jest neutralny; `WithColors` nadaje ton palety sekcji, a bez sekcji każdemu okresowi;
- dwukropek zawsze rozdziela zdarzenia, więc tekst z dwukropkiem (np. godzina `10:30`) trzeba zapisać inaczej;
  tak samo jest w Mermaid.

## Podróż użytkownika (`journey`)

### Obsługiwane

| Element | Składnia | Wymaganie |
|---|---|---|
| nagłówek, tytuł | `journey`; `title Tekst` | JRN-001 |
| krok | `Nazwa kroku: 4: Uczestnik, Uczestnik`; uczestnicy opcjonalni | JRN-002 |
| sekcja | `section Tytuł` | JRN-003 |
| układ listowy | komentarz `%% layout: list` | JRN-005 |
| dostępność | `accTitle`, `accDescr` | JRN-001 |

### Układy

Domyślny układ jest taki jak w Mermaid: kroki w rzędzie pod nagłówkami sekcji, pod każdym krokiem buźka na
wysokości oceny, u dołu oś ze strzałką, z lewej legenda uczestników, a na krokach ich znaczniki.

Układ listowy (`%% layout: list`) stawia kroki z góry na dół: nazwa, znacznik oceny na skali od 1 do 5
i uczestnicy wypisani przy kroku; linia łączy oceny kolejnych kroków. Mieści się na stronie przy wielu krokach,
a ocena jest w nim czytelna bez koloru (położenie na skali; znacznik oceny 1–2 pełny, 3 szary, 4–5 pusty).

### Świadome różnice wobec Mermaid

- buźka ma trzy miny: uśmiech dla ocen 4 i 5, usta proste dla 3, smutek dla 1 i 2;
- domyślnie rysunek jest neutralny, a uczestnicy różnią się odcieniem szarości; `WithColors` daje kolory palety
  sekcjom, uczestnikom i buźkom;
- ocena spoza zakresu 1–5 albo niecałkowita jest błędem.

## Mapa myśli (`mindmap`)

### Obsługiwane

| Element | Składnia | Wymaganie |
|---|---|---|
| nagłówek, hierarchia | `mindmap`; kolejne linie, wcięcie wyznacza rodzica | MND-001 |
| gałąź | sam tekst; `"tekst w cudzysłowach"`; `<br/>` łamie linię | MND-002 |
| kształty | `id[kwadrat]`, `id(zaokrąglony)`, `id((koło))`, `id{{sześciokąt}}`, `id))wybuch((`, `id)chmura(` | MND-004 |
| tekst Markdown | ``"`**pogrubiony** i *pochylony*`"`` | MND-004 |
| ikony, klasy | `::icon(…)`, `:::klasa` (pomijane) | MND-002 |
| układ listowy | komentarz `%% layout: list` | MND-006 |

### Układy

Domyślnie korzeń stoi pośrodku, a gałęzie główne po obu jego stronach, jak w klasycznej mapie myśli. Układ listowy
(`%% layout: list`) stawia wszystkie gałęzie po prawej stronie korzenia i czyta się jak konspekt. Dyrektywa jest
komentarzem, więc to samo źródło w innych narzędziach rysuje układ domyślny.

### Świadome różnice wobec Mermaid

- oba układy są drzewem z gałęziami jednej głębokości w jednej kolumnie i połączeniami pod kątem prostym; Mermaid
  rozkłada gałęzie swobodnie wokół korzenia i łączy je łukami. Drzewo jest powtarzalne i mieści się na stronie;
- „wybuch” jest rysowany jak sześciokąt, a „chmura” jak pole z półkolami: rysunek ma tylko proste kształty;
- koło z dłuższym tekstem staje się elipsą, żeby pole nie rosło nad miarę;
- gałąź bez kształtu jest polem zaokrąglonym (w Mermaid tekstem z podkreśleniem), a korzeń bez kształtu polem
  z półkolami;
- cudzysłowy wokół tekstu są zdejmowane (renderer odniesienia rysuje je dosłownie);
- tabulacja we wcięciu liczy się jak cztery spacje;
- domyślnie rysunek jest neutralny; `WithColors` nadaje każdej gałęzi głównej ton palety;
- `accTitle` i `accDescr` nie są częścią składni mapy myśli: taka linia byłaby gałęzią.

## Wykres słupkowy i liniowy (`xychart-beta`, `xychart`)

### Obsługiwane

| Element | Składnia | Wymaganie |
|---|---|---|
| nagłówek, tytuł | `xychart-beta`; `title "Tekst"` | XY-001 |
| oś pozioma | `x-axis [sty, lut, mar]`; `x-axis "Tytuł" [a, b]`; `x-axis Tytuł 0 --> 10` | XY-002 |
| oś pionowa | `y-axis "Tytuł" 0 --> 40`; `y-axis Tytuł`; `y-axis 0 --> 40` | XY-002 |
| serie | `bar [1, 2, 3]`; `line [1, 2, 3]`; z nazwą: `bar "Plan" [1, 2, 3]` | XY-002, XY-004 |
| dostępność | `accTitle`, `accDescr` | XY-001 |

### Poza profilem (błąd `xychart-unsupported`)

Wykres poziomy (`xychart-beta horizontal`). Konfiguracja w bloku `%%{init}%%` i `---` (rozmiary, motywy,
`showDataLabel`).

### Świadome różnice wobec Mermaid

- tytuł osi pionowej stoi poziomo nad osią, a nie obrócony wzdłuż niej;
- bez podanego zakresu oś pionowa przy słupkach obejmuje zero, bo słupek mierzy się od zera; podziałka ma
  okrągłe wartości;
- kilka serii słupków stoi obok siebie w obrębie kategorii (w Mermaid nakładają się na siebie);
- serie różnią się bez koloru: słupki jasnością, linie kreskowaniem; `WithColors` daje kolory palety;
- nazwa serii przed listą wartości daje legendę pod wykresem;
- gdy etykiety kategorii się nie mieszczą, zostaje co któraś, zamiast nachodzić na siebie;
- seria musi mieć tyle wartości, ile jest kategorii; inna liczba jest błędem.
- tytuł i kategorie przyjmujemy także bez cudzysłowów, gdy mają polskie litery albo kilka wyrazów; Mermaid
  wymaga wtedy cudzysłowów, więc w źródle przenośnym warto je pisać;
- podziałka osi pionowej ma inny krok niż w Mermaid, więc liczby przy osi się różnią.

## Wykres ćwiartkowy (`quadrantChart`)

### Obsługiwane

| Element | Składnia | Wymaganie |
|---|---|---|
| nagłówek, tytuł | `quadrantChart`; `title Tekst` | QDR-001 |
| osie | `x-axis Lewy --> Prawy`; `y-axis Dolny --> Górny`; drugi opis opcjonalny | QDR-001 |
| ćwiartki | `quadrant-1 Tekst` … `quadrant-4 Tekst` | QDR-001 |
| punkt | `Nazwa: [0.3, 0.6]` | QDR-002 |
| style punktów | `Nazwa:::klasa: [x, y]`, `radius: 12, color: #f00, stroke-color: #000, stroke-width: 2px` po nawiasie, `classDef` | QDR-002 |
| dostępność | `accTitle`, `accDescr` | QDR-001 |

### Świadome różnice wobec Mermaid

- nazwa punktu ma własny podkład i nie wychodzi poza wykres: przy bocznej krawędzi jest dosunięta, a przy dolnej
  stoi nad kropką;
- ćwiartki stykające się bokiem różnią się jasnością; `WithColors` nadaje im tony palety;
- współrzędna spoza zakresu 0–1 jest błędem.

## Tablica kanban (`kanban`)

| Element | Składnia | Wymaganie |
|---|---|---|
| kolumna | linia na najpłytszym wcięciu: `Tekst` albo `id[Tekst]` | KBN-001 |
| karta | linia wcięta głębiej: `[Tekst]`, `id[Tekst]` albo sam tekst | KBN-002 |
| dane karty | `@{ ticket: NC-17, assigned: 'A. Nowak', priority: 'High' }` | KBN-002 |
| dostępność | `accTitle`, `accDescr` | KBN-001 |

Świadome różnice wobec Mermaid: priorytet (`Very High`, `High`, `Low`, `Very Low`) widać po grubości paska przy lewej
krawędzi karty, a nie po kolorze; dane karty stoją w jednej linii pod tekstem; odsyłacz zgłoszenia (`ticketBaseUrl`)
nie jest rysowany; domyślnie rysunek jest neutralny, `WithColors` nadaje kolumnom tony palety.

## Diagram pakietu (`packet-beta`, `packet`)

| Element | Składnia | Wymaganie |
|---|---|---|
| tytuł | `title Tekst` | PKT-001 |
| pole | `0-15: "nazwa"`, `16: "nazwa"`, `+8: "nazwa"` | PKT-002 |
| dostępność | `accTitle`, `accDescr` | PKT-001 |

Świadome różnice wobec Mermaid: wiersz ma zawsze 32 bity (konfiguracja `bitsPerRow` jest poza profilem); nazwa, która
nie mieści się w polu, jest rysowana mniejszym pismem; tytuł stoi nad diagramem; przerwa między polami jest błędem.

## Wykres radarowy (`radar-beta`, `radar`)

| Element | Składnia | Wymaganie |
|---|---|---|
| tytuł | `title Tekst` | RDR-001 |
| osie | `axis a["Podpis"], b["Podpis"]` | RDR-001 |
| seria | `curve s["Podpis"]{1, 2, 3}`; `curve s{a: 1, b: 2, c: 3}` | RDR-002 |
| zakres i siatka | `max 5`, `min 0`, `ticks 5`, `graticule circle` albo `polygon`, `showLegend false` | RDR-002 |
| dostępność | `accTitle`, `accDescr` | RDR-001 |

Świadome różnice wobec Mermaid: serie różnią się kreskowaniem linii, więc wykres czyta się bez koloru (`WithColors`
daje kolory palety); legenda stoi z prawej strony; seria musi mieć wartość dla każdej osi; wartość spoza zakresu
jest rysowana na jego brzegu.

## Mapa drzewa (`treemap-beta`, `treemap`)

| Element | Składnia | Wymaganie |
|---|---|---|
| tytuł | `title Tekst` | TRM-001 |
| sekcja | `"Nazwa"`, dzieci wcięte głębiej | TRM-001 |
| liść | `"Nazwa": 12.5` | TRM-001 |
| style | `:::klasa`, `classDef` (pomijane) | TRM-001 |
| dostępność | `accTitle`, `accDescr` | TRM-001 |

Świadome różnice wobec Mermaid: liść, którego nazwa nie mieści się w polu nawet mniejszym pismem, dostaje numer,
a nazwa z wartością trafia do listy pod mapą, więc żadna dana nie znika; wartość jest zapisana tak jak w źródle,
bez formatowania; mapa ma stały rozmiar 620 × 400 px; domyślnie rysunek jest neutralny, `WithColors` nadaje ton
palety każdej gałęzi najwyższego poziomu.

## Diagram wymagań (`requirementDiagram`)

Rysowany tym samym układem warstwowym co schemat blokowy i diagram klas.

| Element | Składnia | Wymaganie |
|---|---|---|
| wymaganie | `requirement nazwa { id: 1  text: …  risk: high  verifymethod: test }` i pięć rodzajów szczegółowych | RQD-001, RQD-002 |
| element | `element nazwa { type: …  docref: … }` | RQD-001, RQD-002 |
| relacja | `a - satisfies -> b`; `b <- satisfies - a` | RQD-003 |
| kierunek, wygląd | `direction LR`; `classDef`, `class`, `nazwa:::styl`, `style` | RQD-003 |
| dostępność | `accTitle`, `accDescr` | RQD-001 |

Świadome różnice wobec Mermaid: długi tekst wymagania jest łamany na linie o około 44 znakach; relacja `contains`
ma kółko przy elemencie zawierającym (w Mermaid kółko z krzyżykiem); pozostałe relacje mają otwarty grot i linię
ciągłą; nazwa użyta w relacji musi być zdefiniowana blokiem; wartość pola przyjmujemy także bez cudzysłowów,
gdy ma łącznik albo kropkę (Mermaid wymaga wtedy cudzysłowów).

## Wykres Sankeya (`sankey-beta`, `sankey`)

| Element | Składnia | Wymaganie |
|---|---|---|
| przepływ | `źródło,cel,wartość`; `"nazwa, z przecinkiem",cel,12.5` | SNK-001 |

Świadome różnice wobec Mermaid: przy każdym węźle stoi jego wartość; wstęgi są domyślnie szare i półprzezroczyste,
a `WithColors` nadaje im kolor węzła źródłowego; wykres ma stałą szerokość 620 px i wysokość rosnącą z liczbą węzłów
w kolumnie; nazwy węzłów mogą zawierać polskie litery (Mermaid w tym typie odrzuca znaki spoza ASCII, także
w cudzysłowach); cykl przepływów jest błędem; `accTitle` i `accDescr` nie są częścią składni (linia byłaby przepływem).

## Diagramy C4 (`C4Context`, `C4Container`, `C4Component`, `C4Dynamic`, `C4Deployment`)

Rysowane tym samym układem warstwowym co schemat blokowy.

| Element | Składnia | Wymaganie |
|---|---|---|
| osoba, system | `Person(alias, "Etykieta", "Opis")`, `System(…)`, z przyrostkami `Db`, `Queue`, `_Ext` | C4D-001 |
| kontener, komponent | `Container(alias, "Etykieta", "Technologia", "Opis")`, `Component(…)`, z tymi samymi przyrostkami | C4D-001 |
| relacja | `Rel(a, b, "Etykieta", "Technologia")`, `BiRel`, `Rel_Back`, `Rel_U`/`D`/`L`/`R`, `RelIndex(1, a, b, "…")` | C4D-002 |
| granica | `Enterprise_Boundary(alias, "Etykieta") { … }`, `System_Boundary`, `Container_Boundary`, `Boundary(alias, "Etykieta", "rodzaj")` | C4D-003 |
| węzeł wdrożenia | `Deployment_Node(alias, "Etykieta", "Typ") { … }`, `Node(…)` | C4D-003 |
| tytuł, dostępność | `title Tekst`; `accTitle`, `accDescr` | C4D-003 |
| wygląd, układ | `UpdateElementStyle(…)`, `UpdateRelStyle(…)`, `UpdateLayoutConfig(…)`, argumenty `$klucz="…"` (pomijane) | C4D-003 |

Świadome różnice wobec Mermaid: układ jest warstwowy, wyznaczony przez relacje, a nie siatką o stałej liczbie
kształtów w rzędzie, więc wskazówki kierunku (`Rel_U`, `Rel_L`) i `UpdateLayoutConfig` nie wpływają na położenie;
tytuł jest ramką wokół diagramu; osoba jest polem z półkolami bez ikony; element zewnętrzny odróżnia linia przerywana
i jaśniejsze tło, a nie kolor; rodzaj elementu jest zapisany jak w Mermaid (`[Person]`, `[Software System]`); wszystkie pięć nagłówków rysuje się tak samo.

## Graf historii Gita (`gitGraph`)

| Element | Składnia | Wymaganie |
|---|---|---|
| commit | `commit`, `commit id: "a" tag: "v1" type: HIGHLIGHT` | GIT-001 |
| gałąź | `branch nazwa`, `checkout nazwa`, `switch nazwa` | GIT-002 |
| scalenie, kopia | `merge nazwa tag: "v2"`; `cherry-pick id: "a"` | GIT-002 |
| dostępność | `accTitle`, `accDescr` | GIT-001 |

Poza profilem (błąd `gitgraph-unsupported`): orientacja pionowa (`gitGraph TB:`, `BT:`). Świadome różnice wobec
Mermaid: commit bez podanego identyfikatora nie ma podpisu (Mermaid losuje identyfikator, więc rysunek nie byłby
powtarzalny); identyfikator stoi poziomo pod commitem, a odstęp commitów mieści najdłuższy; gałęzie stoją
w kolejności powstania (`order:` jest pomijane); domyślnie rysunek jest neutralny, `WithColors` nadaje gałęziom
kolory palety.

## Diagram blokowy (`block-beta`, `block`)

| Element | Składnia | Wymaganie |
|---|---|---|
| siatka | `columns 3`, `columns auto` | BLK-001 |
| blok | `id`, `id["Etykieta"]`, z szerokością `id:2`; `space`, `space:2` | BLK-001 |
| kształty | `(…)`, `([…])`, `((…))`, `{…}`, `{{…}}`, `[(…)]`, strzałka `id<["…"]>(right)` | BLK-002 |
| blok złożony | `block:id:3 … end` | BLK-002 |
| połączenia | `a --> b`, `a --- b`, `a -.-> b`, `a -- "tekst" --> b` | BLK-003 |
| wygląd | `style`, `classDef`, `class` | BLK-003 |

Świadome różnice wobec Mermaid: połączenie jest prostym odcinkiem między brzegami kształtów i może przejść nad
innym blokiem; gdy diagram ma połączenia, odstępy między blokami rosną tak, żeby zmieściły się groty i etykiety;
strzałka blokowa z kierunkiem `y` jest rysowana w prawo.

## Diagram architektury (`architecture-beta`, `architecture`)

| Element | Składnia | Wymaganie |
|---|---|---|
| usługa | `service id(ikona)[Podpis]`, z `in grupa` | ARC-001 |
| grupa | `group id(ikona)[Podpis]`, z `in grupa` | ARC-001 |
| rozgałęzienie | `junction id` | ARC-001 |
| krawędź | `a:R -- L:b`, `a:B --> T:b`, `a:R <-[tekst]-> L:b`, `a{group}:R -- L:b` | ARC-002 |
| tytuł, dostępność | `title Tekst`; `accTitle`, `accDescr` | ARC-001 |

Świadome różnice wobec Mermaid: ikony są własnymi, uproszczonymi znakami pięciu wbudowanych nazw (`database`,
`server`, `disk`, `cloud`, `internet`); ikona o innej nazwie, także z zestawów zewnętrznych, jest pustym polem;
ikona grupy nie jest rysowana, grupa ma sam podpis; położenie usług wynika wyłącznie z boków krawędzi, a usługa
spoza grupy, która wypadłaby w obrębie jej ramki, odsuwa się dalej w swoim kierunku; krawędź jest łamaną
z najwyżej dwoma zagięciami i może przejść nad inną usługą.
