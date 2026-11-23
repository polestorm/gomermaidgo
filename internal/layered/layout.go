// Package layered rozmieszcza graf metodą warstwową i tworzy scenę.
//
// Kroki pochodzą z publikacji o rysowaniu grafów skierowanych (Sugiyama, Tagawa, Toda 1981;
// Gansner, Koutsofios, North, Vo 1993): usunięcie cykli, podział na warstwy, węzły pomocnicze
// na długich krawędziach, redukcja przecięć metodą barycentryczną z zamianą sąsiadów,
// rozmieszczenie w warstwie przez iteracyjne dociąganie do sąsiadów. Podgrafy są rozmieszczane
// rekurencyjnie: wnętrze podgrafu jest osobnym układem, a w układzie rodzica jednym polem.
//
// Układ liczy w osiach umownych: x to oś kolejności w warstwie, y to oś warstw. Kierunek
// diagramu (TB, BT, LR, RL) jest nakładany dopiero przy rysowaniu.
package layered

import (
	"math"
	"sort"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Options to ustawienia układu.
type Options struct {
	Measure func(text string, size float64) float64
	// MeasureStyled mierzy tekst pogrubiony albo pochylony; nil oznacza miarę zwykłą,
	// dla pogrubienia powiększoną o 6%.
	MeasureStyled func(text string, size float64, bold, italic bool) float64
	FontSize      float64
	FontFamily    string
	Background    string
	Colors        bool // kolor według roli węzła: krok, decyzja, początek i koniec
	Shadows       bool // cień pod węzłami i notatkami
	// CurvedEdges przywraca krawędzie prowadzone łagodnymi łukami; domyślnie są łamane pod kątem
	// prostym, z zaokrąglonymi narożnikami.
	CurvedEdges bool
	// Fit: dopasowanie do strony. Układ jest powtarzany z węższymi polami (tekst łamany częściej)
	// i wygrywa wersja mieszcząca się na stronie A4 w największej skali. FitAuto (domyślnie) robi to
	// tylko dla rysunku co najmniej FitRatio razy szerszego niż wysoki; FitOn dla każdego, który nie
	// mieści się na stronie; FitOff nigdy.
	Fit      FitMode
	FitRatio float64 // próg proporcji szerokości do wysokości dla FitAuto i FoldAuto; 0 oznacza 2
	// Fold: zwijanie długiego, prostego diagramu w kilka rzędów (LR) albo kolumn (TB), gdy rysunek
	// jest co najmniej FitRatio razy dłuższy niż szeroki i nie mieści się na stronie A4 w skali 1.
	Fold     FoldMode
	FoldType FoldType

	// FoldChains zwija długie łańcuchy pól wewnątrz diagramu w kolumny (TB) albo rzędy (LR), gdy
	// rysunek zyskuje na tym co najmniej 20% skali na stronie A4 (LAY-022). Sposób: FoldType.
	FoldChains bool
	// FoldRanks rozkłada długą warstwę liści jednego węzła (albo korzeni wchodzących do jednego
	// węzła) na dwie podwarstwy ustawione zygzakiem, gdy rysunek zyskuje na tym co
	// najmniej 20% skali na stronie A4 (LAY-025).
	FoldRanks bool
	chainBad  *bool // ustawiane, gdy blok łańcucha musiałby być przecięty (próba w foldChains)

	// NoWrap wyłącza automatyczne łamanie napisów (konfiguracja Mermaid „markdownAutoWrap: false”
	// albo „wrap: false”): zostają przełamania autora; łamie się tylko linia szersza niż authorLimit.
	NoWrap bool
	// WrapWidth to granica łamania napisów w polach i na krawędziach przy piśmie 16 px
	// („flowchart.wrappingWidth”); 0 = 200 px.
	WrapWidth float64

	boxes map[string]scene.Box // wypełniane przy rysowaniu, gdy nie nil (layoutBoxes)
	// painted: źródło samo nadaje wypełnienie choć jednemu polu albo ramce. Pola bez koloru
	// zostają wtedy neutralne (białe), a ramki bez koloru bez tła trybu kolorowego: brak koloru
	// znaczy w takim rysunku „zwykłe pole”, a nie „pokoloruj według palety”.
	painted bool
}

// FoldMode to tryb zwijania.
type FoldMode int

const (
	FoldAuto FoldMode = iota
	FoldOff
)

// FoldType to sposób przejścia do następnego rzędu: powrót do początku albo wężyk.
type FoldType int

const (
	FoldReturn FoldType = iota // każdy rząd w tym samym kierunku; łącznik wraca pod rzędem na początek
	FoldSnake                  // co drugi rząd w przeciwnym kierunku; łącznik przy końcu rzędu
)

// FitMode to tryb dopasowania do strony.
type FitMode int

const (
	FitAuto FitMode = iota
	FitOn
	FitOff
)

type itemKind int

const (
	kindNode itemKind = iota
	kindCluster
	kindDummy
	kindLabel
	kindPort // punkt na ramce podgrafu, przez który przechodzi krawędź z zewnątrz
)

// item to pole w układzie jednego poziomu: węzeł, podgraf, punkt pomocniczy albo etykieta krawędzi.
type item struct {
	kind  itemKind
	node  *graph.Node
	child *level   // dla podgrafu
	text  []string // linie etykiety (węzła albo krawędzi)

	w, h      float64 // rozmiar w osiach umownych
	loopExtra float64 // miejsce po stronie +x na pętle własne
	// textShift: środek etykiety krawędzi względem linii krawędzi (LAY-028: linia biegnie prosto,
	// a etykieta stoi tam, gdzie się mieści, z linią przechodzącą przez jej pole).
	textShift float64
	band      float64 // wysokość warstwy, w której leży pole

	lane        int // numer toru, gdy diagram ma tory
	column      int // kolumna w rzędzie: części w jednej kolumnie stoją jedna za drugą
	shelf       int // rząd, w którym stoi część diagramu zawierająca to pole (patrz shelve)
	rank, order int
	x, y        float64
	up, down    []*item
	portTop     bool // punkt wejścia przy początku osi warstw (w przeciwnym razie przy końcu)

	// Notatka stoi w warstwie pola, którego dotyczy, tuż obok niego: po stronie +x (1) albo −x (−1).
	anchor     *item
	anchorSide float64
}

// nodeX to środek samego kształtu (bez miejsca na pętle).
func (it *item) nodeX() float64 { return it.x - it.loopExtra/2 }

// nodeW to szerokość samego kształtu.
func (it *item) nodeW() float64 { return it.w - it.loopExtra }

func (it *item) real() bool { return it.kind == kindNode || it.kind == kindCluster }

// ledge to krawędź na poziomie, na którym spotykają się jej końce.
type ledge struct {
	e     *graph.Edge
	u, v  *item // górny i dolny koniec w układzie (po odwróceniu krawędzi wstecznych)
	rev   bool
	chain []*item
	label *item

	// Przesunięcia punktów zaczepienia wzdłuż krawędzi pola: górnego końca i dolnego.
	upperPort, lowerPort float64
	// sideExit: krawędź wychodzi z bocznego wierzchołka rombu (−1: po stronie −x, +1: po stronie +x)
	// i biegnie w poprzek na wysokości środka pola, zanim skręci wzdłuż osi warstw.
	sideExit float64

	lv *level
	// Identyfikatory pól ze źródła na górnym i dolnym końcu (po odwróceniu); pusty dla punktu na ramce.
	topID, bottomID string
	// Gdy koniec leży głębiej, w podgrafie, dalszy odcinek krawędzi jest prowadzony w układzie
	// tego podgrafu, między polem a punktem na jego ramce.
	upperCont, lowerCont *ledge
	part                 bool // odcinek krawędzi z poziomu wyżej; nie jest rysowany osobno
	flat                 bool // LAY-027: oba końce w jednej warstwie, krawędź biegnie w poprzek
}

// level to układ jednego zakresu: całego diagramu albo wnętrza podgrafu.
type level struct {
	sub    *graph.Subgraph
	items  []*item
	edges  []*ledge
	loops  map[*item][]*graph.Edge
	routes []*edgeRoute // przebiegi krawędzi poziomu, gotowe do rysowania
	ranks  [][]*item

	w, h             float64 // rozmiar z marginesami
	padX0, padY0     float64
	offsetX, offsetY float64 // położenie początku poziomu w osiach umownych diagramu
	titleLines       []string

	// Układ osi poziomu. Podgraf z własnym kierunkiem (turned) ma własne osie umowne, zaczepione
	// w punkcie (ox, oy) rysunku; pozostałe poziomy dziedziczą osie rodzica.
	free                   map[*item][2]float64 // położenia ramek rozmieszczonych swobodnie (packBoxes)
	shelved                bool                 // części poziomu dostały rzędy i kolumny w shelve
	pinPorts               bool                 // porządkowanie nie rusza warstw złożonych z samych punktów na ramce
	vertical, flip, turned bool
	ox, oy                 float64
}

// enter przełącza układ osi na osie poziomu i zwraca funkcję, która przywraca poprzednie.
func (l *layouter) enter(lv *level) func() {
	v, f, ox, oy := l.vertical, l.flip, l.ox, l.oy
	l.vertical, l.flip, l.ox, l.oy = lv.vertical, lv.flip, lv.ox, lv.oy
	return func() { l.vertical, l.flip, l.ox, l.oy = v, f, ox, oy }
}

// orient ustala osie każdego poziomu. Kierunek podany w podgrafie obowiązuje tylko wtedy, gdy
// żadna krawędź nie łączy jego wnętrza z resztą diagramu (tak samo rozstrzyga to Mermaid);
// inaczej wnętrze ma kierunek otoczenia.
func (l *layouter) orient() {
	inside := func(id, sub string) bool {
		for s := l.scopeOf(id); s != ""; s = l.g.Subgraph(s).Parent {
			if s == sub {
				return true
			}
		}
		return false
	}
	isolated := func(sub string) bool {
		for _, e := range l.g.Edges {
			if inside(e.From, sub) != inside(e.To, sub) {
				return false
			}
		}
		return true
	}
	var set func(lv *level)
	set = func(lv *level) {
		for _, it := range lv.items {
			if it.kind != kindCluster {
				continue
			}
			c := it.child
			c.vertical, c.flip = lv.vertical, lv.flip
			if d := c.sub.Direction; d != "" && isolated(c.sub.ID) {
				v := d == graph.TopBottom || d == graph.BottomTop
				f := d == graph.BottomTop || d == graph.RightLeft
				c.turned = v != lv.vertical || f != lv.flip
				c.vertical, c.flip = v, f
			}
			set(c)
		}
	}
	root := l.levels[""]
	root.vertical, root.flip = l.vertical, l.flip
	set(root)
}

type layouter struct {
	bg string // wypełnienie ramki, w której trwa rysowanie; pusty = tło rysunku
	// free: punkty przecięcia krawędzi z ramkami i zakres, w którym mogą się przesuwać wzdłuż ramki.
	free map[routePoint][2]float64
	// frameCross: punkty, w których krawędź przechodzi przez ramkę (podzbiór free).
	frameCross map[routePoint]bool
	g          *graph.Graph
	o          Options
	sc         *scene.Scene

	fs, lh   float64
	vertical bool    // TB albo BT: oś warstw jest pionowa
	flip     bool    // BT albo RL: oś warstw odwrócona
	ox, oy   float64 // zaczepienie bieżących osi umownych na rysunku

	lanes  []*lane        // tory; nil = zwykły układ
	laneOf map[string]int // węzeł → numer toru
	// rankLimit: mnożnik limitu szerokości warstwy w diagramie encji; 0 = bez limitu.
	rankLimit float64
	// fit: mnożnik granicy łamania tekstu i najmniejszej szerokości pola (dopasowanie do strony).
	fit float64

	levels map[string]*level
	itemOf map[string]*item // węzeł albo podgraf → jego pole na poziomie rodzica
}

func (l *layouter) u(v float64) float64 { return math.Round(v * l.fs / 16) }

func (l *layouter) width(s string) float64 {
	if !graph.HasMarks(s) {
		return l.o.Measure(s, l.fs)
	}
	w := 0.0
	for _, r := range graph.Runs(s) {
		w += l.runWidth(r)
	}
	return w
}

func (l *layouter) runWidth(r graph.Run) float64 {
	switch {
	case !r.Bold && !r.Italic:
		return l.o.Measure(r.Text, l.fs)
	case l.o.MeasureStyled != nil:
		return l.o.MeasureStyled(r.Text, l.fs, r.Bold, r.Italic)
	case r.Bold:
		return l.o.Measure(r.Text, l.fs) * 1.06
	}
	return l.o.Measure(r.Text, l.fs)
}

// wrap łamie linie szersze niż limit na granicach wyrazów. Napis złamany przez autora (kilka linii,
// np. <br>) zachowuje jego przełamania: program nie dokłada własnych, także przy dopasowaniu do strony.
// Bezpiecznik: linia szersza niż authorLimit i tak się łamie.
func (l *layouter) wrap(lines []string, limit float64) []string {
	// Linie pochodzą od autora tylko w schemacie i diagramie stanów; w C4, klasach czy wymaganiach
	// kolejne linie (nazwa, typ, opis) składa sam program.
	if l.o.NoWrap || len(lines) > 1 && (l.g.Kind == "flowchart" || l.g.Kind == "state") {
		limit = math.Max(limit, l.u(l.authorLimit()))
	}
	var out []string
	for _, line := range lines {
		line = graph.PerWord(line)
		if l.width(line) <= limit {
			out = append(out, line)
			continue
		}
		current := ""
		for _, word := range l.hyphenParts(strings.Fields(line), limit) {
			// Dalsza część słowa złamanego po łączniku dokleja się bez spacji.
			sep := " "
			if rest, ok := strings.CutPrefix(word, glue); ok {
				word, sep = rest, ""
			}
			candidate := word
			if current != "" {
				candidate = current + sep + word
			}
			if current != "" && l.width(candidate) > limit {
				out = append(out, current)
				current = word
			} else {
				current = candidate
			}
		}
		if current != "" {
			out = append(out, current)
		}
	}
	return out
}

// authorLimit to szerokość (przy piśmie 16 px), powyżej której łamie się nawet linia z napisu
// złamanego przez autora albo z rysunku bez automatycznego łamania: dwa razy więcej niż zwykła
// granica pola.
func (l *layouter) authorLimit() float64 { return 2 * l.baseWrap() }

// baseWrap to zwykła granica łamania napisu w polu i na krawędzi przy piśmie 16 px.
func (l *layouter) baseWrap() float64 {
	if l.o.WrapWidth > 0 {
		return l.o.WrapWidth
	}
	return 200
}

// glue oznacza w wyniku hyphenParts dalszą część słowa, która dokleja się do poprzedniej bez spacji.
const glue = "\x00"

// hyphenParts przy dopasowaniu do strony dzieli słowo dłuższe niż granica po łącznikach
// („binding-markup” → „binding-”, „markup”), żeby i ono mogło przejść do następnej linii. Słowa
// z wyróżnieniem (znaczniki * i _) zostają całe, bo podział rozdzieliłby znaczniki.
func (l *layouter) hyphenParts(words []string, limit float64) []string {
	if l.narrow() >= 1 {
		return words
	}
	var out []string
	for _, w := range words {
		if l.width(w) <= limit || strings.ContainsAny(w, "*_") || !strings.Contains(strings.Trim(w, "-"), "-") {
			out = append(out, w)
			continue
		}
		rest, prefix := w, ""
		for {
			i := strings.Index(rest[1:], "-")
			if i < 0 || i+2 >= len(rest) {
				out = append(out, prefix+rest)
				break
			}
			out = append(out, prefix+rest[:i+2])
			rest, prefix = rest[i+2:], glue
		}
	}
	return out
}

func (l *layouter) textSize(lines []string) (w, h float64) {
	for _, s := range lines {
		w = math.Max(w, l.width(s))
	}
	return w, float64(len(lines)) * l.lh
}

// abstract zamienia rozmiar rzeczywisty (szerokość, wysokość) na osie umowne.
func (l *layouter) abstract(w, h float64) (float64, float64) {
	if l.vertical {
		return w, h
	}
	return h, w
}

// point zamienia punkt z osi umownych na współrzędne rysunku.
func (l *layouter) point(x, y float64) scene.Point {
	if l.flip {
		y = -y
	}
	if l.vertical {
		return scene.Point{X: x + l.ox, Y: y + l.oy}
	}
	return scene.Point{X: y + l.ox, Y: x + l.oy}
}

// Layout rozmieszcza graf i zwraca scenę.
func Layout(g *graph.Graph, o Options) *scene.Scene {
	if o.FontSize <= 0 {
		o.FontSize = 16
	}
	o.painted = g.SourceFills
	if o.FoldChains {
		g = foldChains(g, o)
	}
	if o.FoldRanks {
		g = spreadFans(g, o)
	}
	if g.Kind != "er" {
		best := layout(g, o, 0, 1)
		skipFit := false
		switch o.Fit {
		case FitOff:
			skipFit = true
		case FitAuto:
			ratio := o.FitRatio
			if ratio <= 0 {
				ratio = 2
			}
			if b := best.Bounds(); b.X1-b.X0 < ratio*(b.Y1-b.Y0) {
				skipFit = true
			}
		}
		if skipFit {
			if folded := fold(g, o, best, 1); folded != nil {
				return folded
			}
			return best
		}
		// Dopasowanie do strony: kolejne próby z węższymi polami; wygrywa wyraźnie większa skala.
		bestScale, bestFit := pageScale(best, o.FontSize), 1.0
		for _, fit := range []float64{0.8, 0.65, 0.5, 0.38, 0.28} {
			if bestScale >= 1 {
				break
			}
			sc := layout(g, o, 0, fit)
			if s := pageScale(sc, o.FontSize); s > bestScale+0.02 {
				best, bestScale, bestFit = sc, s, fit
			}
		}
		if folded := fold(g, o, best, bestFit); folded != nil {
			return folded
		}
		return best
	}
	// Diagram encji: kierunek nie niesie znaczenia, więc wygrywa układ, który mieści się na stronie
	// w największej skali. Kandydaci różnią się limitem szerokości warstwy; pierwszy jest bez limitu
	// i zostaje, gdy rysunek i tak mieści się na stronie.
	u := o.FontSize / 16
	pageW, pageH := 650*u, 870*u
	var best *scene.Scene
	bestScale := 0.0
	for _, factor := range []float64{0, 1.4, 1, 0.75, 0.55} {
		sc := layout(g, o, factor, 1)
		b := sc.Bounds()
		w, h := b.X1-b.X0, b.Y1-b.Y0
		scale := math.Min(1, math.Min(pageW/w, pageH/h))
		// Limit wygrywa tylko wyraźną poprawą skali: drobna różnica nie jest warta dłuższych krawędzi.
		if best == nil || scale > bestScale+0.02 {
			best, bestScale = sc, scale
		}
		if factor == 0 && scale >= 1 {
			break
		}
	}
	return best
}

// pageScale zwraca skalę, w której rysunek mieści się na stronie A4 z marginesami, w lepszym
// z dwóch położeń strony (najwyżej 1).
func pageScale(sc *scene.Scene, fs float64) float64 {
	u := fs / 16
	b := sc.Bounds()
	w, h := b.X1-b.X0+2*sc.Margin, b.Y1-b.Y0+2*sc.Margin
	portrait := math.Min(650*u/w, 870*u/h)
	landscape := math.Min(870*u/w, 650*u/h)
	return math.Min(1, math.Max(portrait, landscape))
}

// layout rozmieszcza graf przy podanym limicie szerokości warstwy (0: bez limitu) i szerokości pól
// (fit: mnożnik granicy łamania tekstu i najmniejszej szerokości pola; 1 = zwykła).
func layout(g *graph.Graph, o Options, rankLimit, fit float64) *scene.Scene {
	var lanes []*lane
	var laneOf map[string]int
	if g.Layout == "swimlanes" && len(g.Subgraphs) > 0 {
		g, lanes, laneOf = flattenLanes(g)
	}
	l := &layouter{g: g, o: o, fs: o.FontSize, levels: map[string]*level{}, itemOf: map[string]*item{}, lanes: lanes, laneOf: laneOf, rankLimit: rankLimit, fit: fit}
	l.lh = math.Round(l.fs * 1.25)
	l.vertical = g.Direction == graph.TopBottom || g.Direction == graph.BottomTop
	l.flip = g.Direction == graph.BottomTop || g.Direction == graph.RightLeft
	l.sc = &scene.Scene{Title: g.AccTitle, Desc: g.AccDescr, FontFamily: o.FontFamily, FontSize: l.fs, Margin: l.u(12), Background: o.Background}

	l.build()
	l.orient()
	l.measure()
	l.evenChains()
	l.assignEdges()
	l.rankAll(l.levels[""])
	l.sizeAll(l.levels[""])
	l.place(l.levels[""], 0, 0)
	l.draw()
	return l.sc
}

// layoutBoxes rozmieszcza graf jak layout i zwraca też prostokąty narysowanych pól i ramek według
// identyfikatorów (do łączenia części zwiniętego diagramu).
func layoutBoxes(g *graph.Graph, o Options, fit float64) (*scene.Scene, map[string]scene.Box) {
	boxes := map[string]scene.Box{}
	o.boxes = boxes
	return layout(g, o, 0, fit), boxes
}

// build tworzy poziomy i pola węzłów; noty diagramu stanów stają się węzłami z krawędzią do celu.
func (l *layouter) build() {
	l.levels[""] = &level{loops: map[*item][]*graph.Edge{}}
	for _, s := range l.g.Subgraphs {
		l.levels[s.ID] = &level{sub: s, loops: map[*item][]*graph.Edge{}, titleLines: l.wrap(s.Title, l.u(260))}
	}
	type entry struct {
		line  int
		order int
		it    *item
		scope string
	}
	var entries []entry
	for i, s := range l.g.Subgraphs {
		it := &item{kind: kindCluster, child: l.levels[s.ID]}
		l.itemOf[s.ID] = it
		order := i
		if s.Chain > 0 {
			// Blok łańcucha stoi w kolejności tam, gdzie jego pierwsze pole: inaczej usuwanie cykli
			// mogłoby zacząć od niego i odwrócić krawędź wchodzącą zamiast krawędzi wstecznej.
			for k, n := range l.g.Nodes {
				if n.Parent == s.ID {
					order = len(l.g.Subgraphs) + k
					break
				}
			}
		}
		entries = append(entries, entry{s.Line, order, it, s.Parent})
	}
	for i, n := range l.g.Nodes {
		it := &item{kind: kindNode, node: n, lane: l.laneOf[n.ID]}
		it.text = l.wrap(n.Label, l.wrapLimit(n.Shape))
		l.itemOf[n.ID] = it
		entries = append(entries, entry{n.Line, len(l.g.Subgraphs) + i, it, n.Parent})
	}
	sort.SliceStable(entries, func(a, b int) bool {
		if entries[a].line != entries[b].line {
			return entries[a].line < entries[b].line
		}
		return entries[a].order < entries[b].order
	})
	for _, e := range entries {
		lv := l.levels[e.scope]
		lv.items = append(lv.items, e.it)
	}
}

// measure nadaje polom węzłów rozmiar w osiach ich poziomu.
func (l *layouter) measure() {
	for _, lv := range l.levels {
		restore := l.enter(lv)
		for _, it := range lv.items {
			if it.kind == kindNode && it.node.Shape == graph.Entity {
				it.w, it.h = l.abstract(l.entitySize(it.node))
			} else if it.kind == kindNode && (it.node.Shape == graph.Class || it.node.Shape == graph.Requirement) {
				it.w, it.h = l.abstract(l.classSize(it.node))
			} else if it.kind == kindNode {
				it.w, it.h = l.abstract(l.nodeSize(it.node.Shape, it.text))
			}
		}
		restore()
	}
}

// footStrip to wysokość paska zamykającego stan złożony od dołu, w pikselach przy czcionce 16.
const footStrip = 4

// titleStrip mówi, czy tytuł ramki stoi w osobnym pasku u góry: tak wyglądają stany złożone.
func (l *layouter) titleStrip(lv *level) bool {
	return l.g.Kind == "state" && lv.sub != nil && !lv.sub.Region && len(lv.titleLines) > 0
}

// boxLike mówi, czy pole jest zwykłym polem z tekstem, którego rozmiar można wyrównać z sąsiadami.
func boxLike(it *item) bool {
	if it.kind != kindNode || it.anchor != nil {
		return false
	}
	switch it.node.Shape {
	case graph.Rect, graph.Round, graph.Stadium, graph.Subroutine:
		return true
	}
	return false
}

// even wyrównuje rozmiar pól w jednym wymiarze: w poprzek osi warstw (cross) albo wzdłuż niej.
// Pola o rozmiarze co najmniej 60% największego dostają rozmiar największego; mniejsze zostają,
// żeby krótki napis nie dostał pola kilka razy za dużego. Gdy wyrównywana jest rzeczywista
// szerokość, a najszersze pole ma jedną linię tekstu, którą da się złamać na dwie tak, żeby
// zmieściło się w szerokości drugiego co do wielkości, pole jest łamane zamiast poszerzać resztę.
func (l *layouter) even(items []*item, cross bool) {
	if len(items) < 2 {
		return
	}
	get := func(it *item) float64 {
		if cross {
			return it.w
		}
		return it.h
	}
	widest, second := items[0], 0.0
	for _, it := range items[1:] {
		if get(it) > get(widest) {
			widest = it
		}
	}
	for _, it := range items {
		if it != widest {
			second = math.Max(second, get(it))
		}
	}
	if realWidth := cross == l.vertical; realWidth && len(widest.text) == 1 && get(widest) > second+1 {
		lines := l.wrap(widest.node.Label, second-2*l.u(28))
		if len(lines) == 2 {
			if w, h := l.nodeSize(widest.node.Shape, lines); w <= second+0.5 {
				widest.text = lines
				widest.w, widest.h = l.abstract(w, h)
			}
		}
	}
	target := 0.0
	for _, it := range items {
		target = math.Max(target, get(it))
	}
	for _, it := range items {
		if get(it) >= 0.6*target {
			if cross {
				it.w = target
			} else {
				it.h = target
			}
		}
	}
}

// evenChains wyrównuje w poprzek osi warstw pola połączone w prosty ciąg (krawędź jest jedyną
// wychodzącą z pierwszego i jedyną wchodzącą do drugiego): takie pola stoją jedno za drugim
// i przy różnych rozmiarach wyglądają nierówno.
func (l *layouter) evenChains() {
	out, in := map[string]int{}, map[string]int{}
	for _, e := range l.g.Edges {
		if e.From != e.To {
			out[e.From]++
			in[e.To]++
		}
	}
	parent := map[*item]*item{}
	var find func(it *item) *item
	find = func(it *item) *item {
		if p := parent[it]; p != nil && p != it {
			parent[it] = find(p)
			return parent[it]
		}
		return it
	}
	for _, e := range l.g.Edges {
		a, b := l.itemOf[e.From], l.itemOf[e.To]
		if a == nil || b == nil || a == b || !boxLike(a) || !boxLike(b) || a.node.Parent != b.node.Parent || out[e.From] != 1 || in[e.To] != 1 {
			continue
		}
		parent[find(a)] = find(b)
	}
	groups := map[*item][]*item{}
	var roots []*item
	for _, n := range l.g.Nodes {
		it := l.itemOf[n.ID]
		if it == nil || !boxLike(it) {
			continue
		}
		r := find(it)
		if groups[r] == nil {
			roots = append(roots, r)
		}
		groups[r] = append(groups[r], it)
	}
	for _, r := range roots {
		restore := l.enter(l.levels[r.node.Parent])
		l.even(groups[r], true)
		restore()
	}
}

// scopeOf zwraca zakres, w którym leży pole węzła albo podgrafu.
func (l *layouter) scopeOf(id string) string {
	if n := l.g.Node(id); n != nil {
		return n.Parent
	}
	return l.g.Subgraph(id).Parent
}

// assignEdges przypisuje każdą krawędź do najgłębszego poziomu wspólnego dla obu końców
// i podnosi końce do pól tego poziomu.
func (l *layouter) assignEdges() {
	chain := func(id string) []string {
		var scopes []string
		for s := l.scopeOf(id); ; s = l.g.Subgraph(s).Parent {
			scopes = append(scopes, s)
			if s == "" {
				return scopes
			}
		}
	}
	lift := l.lift
	add := func(e *graph.Edge) {
		if e.From == e.To {
			lv := l.levels[l.scopeOf(e.From)]
			it := l.itemOf[e.From]
			lv.loops[it] = append(lv.loops[it], e)
			return
		}
		a, b := chain(e.From), chain(e.To)
		in := map[string]bool{}
		for _, s := range b {
			in[s] = true
		}
		scope := ""
		for _, s := range a {
			if in[s] {
				scope = s
				break
			}
		}
		lv := l.levels[scope]
		le := &ledge{e: e, u: lift(e.From, scope), v: lift(e.To, scope), lv: lv}
		if le.u == le.v {
			return // nie występuje po kontroli spójności grafu; krawędź bez rysunku
		}
		lv.edges = append(lv.edges, le)
	}
	for _, e := range l.g.Edges {
		add(e)
	}
	for _, n := range l.g.Notes {
		id := "[note]" + n.Target + "#" + n.Side
		for l.itemOf[id] != nil {
			id += "'"
		}
		scope := l.scopeOf(n.Target)
		it := &item{kind: kindNode, node: &graph.Node{ID: id, Shape: "note", Parent: scope}}
		it.text = l.wrap(n.Text, l.u(l.baseWrap()))
		restore := l.enter(l.levels[scope])
		it.w, it.h = l.abstract(l.nodeSize("note", it.text))
		restore()
		l.itemOf[id] = it
		it.anchor, it.anchorSide = l.itemOf[n.Target], 1
		if n.Side == "left" {
			it.anchorSide = -1
		}
		lv := l.levels[scope]
		lv.items = append(lv.items, it)
	}
}

// lift zwraca pole, które na poziomie scope reprezentuje węzeł albo podgraf id.
func (l *layouter) lift(id, scope string) *item {
	for l.scopeOf(id) != scope {
		id = l.scopeOf(id)
	}
	return l.itemOf[id]
}

// rankAll dzieli na warstwy poziom, a potem jego podgrafy. Kolejność od góry jest potrzebna, bo
// dopiero po podziale poziomu wiadomo, czy krawędź z zewnątrz wchodzi do podgrafu od strony
// początku, czy końca osi warstw; podgraf dostaje wtedy punkt na ramce i odcinek do pola wewnątrz.
func (l *layouter) rankAll(lv *level) {
	l.rank(lv)
	// Pola jednej warstwy stoją obok siebie: wyrównujemy ich rozmiar wzdłuż osi warstw.
	restore := l.enter(lv)
	byRank := map[int][]*item{}
	var ranks []int
	for _, it := range lv.items {
		if boxLike(it) {
			if byRank[it.rank] == nil {
				ranks = append(ranks, it.rank)
			}
			byRank[it.rank] = append(byRank[it.rank], it)
		}
	}
	for _, r := range ranks {
		l.even(byRank[r], false)
	}
	restore()
	for _, e := range lv.edges {
		if e.u.kind == kindCluster && e.topID != "" && l.itemOf[e.topID] != e.u {
			child := e.u.child
			port := &item{kind: kindPort, w: l.u(8)}
			child.items = append(child.items, port)
			part := &ledge{e: &graph.Edge{MinLen: 1}, u: l.lift(e.topID, child.sub.ID), v: port, lv: child, topID: e.topID, part: true}
			child.edges = append(child.edges, part)
			e.upperCont = part
		}
		if e.v.kind == kindCluster && e.bottomID != "" && l.itemOf[e.bottomID] != e.v {
			child := e.v.child
			port := &item{kind: kindPort, w: l.u(8), portTop: true}
			child.items = append(child.items, port)
			part := &ledge{e: &graph.Edge{MinLen: 1}, u: port, v: l.lift(e.bottomID, child.sub.ID), lv: child, bottomID: e.bottomID, part: true}
			child.edges = append(child.edges, part)
			e.lowerCont = part
		}
	}
	for _, it := range lv.items {
		if it.kind == kindCluster {
			l.rankAll(it.child)
		}
	}
}

// sizeAll porządkuje i rozmieszcza poziom po rozmieszczeniu jego podgrafów.
func (l *layouter) sizeAll(lv *level) {
	defer l.enter(lv)()
	for _, it := range lv.items {
		if it.kind == kindCluster {
			l.sizeAll(it.child)
			it.w, it.h = it.child.w, it.child.h
			if c := it.child; c.turned {
				// Wnętrze ma własne osie: jego rozmiar przechodzi przez rozmiar rzeczywisty.
				rw, rh := c.w, c.h
				if !c.vertical {
					rw, rh = rh, rw
				}
				it.w, it.h = l.abstract(rw, rh)
			}
		}
		for _, e := range lv.loops[it] {
			extra := l.u(40)
			if len(e.Label) > 0 {
				w, h := l.abstract(l.textSize(l.wrap(e.Label, l.u(l.baseWrap())*l.narrow())))
				_ = h
				extra += w + l.u(8)
			}
			if extra > it.loopExtra {
				it.w += extra - it.loopExtra
				it.loopExtra = extra
			}
		}
	}
	if l.lanes == nil && !chainBlock(lv) {
		l.shelve(lv)
	}
	l.order(lv)
	if l.lanes != nil {
		l.groupByLane(lv)
	}
	l.finish(lv)

	// Drugie podejście: skoro wiadomo już, gdzie na tym poziomie leżą drugie końce krawędzi
	// wchodzących do podgrafów, punkty na ich ramkach ustawiamy w tej kolejności i wnętrza
	// rozmieszczamy od nowa.
	// Ramki mogą wskazywać na siebie nawzajem, więc wyrównanie powtarzamy, najwyżej dwa razy.
	for round := 0; round < 2; round++ {
		changed := false
		for _, it := range lv.items {
			if it.kind == kindCluster && l.alignPorts(lv, it) {
				l.finish(it.child)
				it.w, it.h = it.child.w, it.child.h
				changed = true
			}
		}
		if !changed {
			break
		}
		l.finish(lv)
	}
}

// portOffset zwraca położenie punktu na ramce podgrafu względem środka ramki, wzdłuż osi kolejności.
func portOffset(cluster, port *item) float64 {
	c := cluster.child
	return c.padX0 + port.x - c.w/2
}

// alignPorts ustawia punkty na ramce podgrafu w kolejności położenia drugich końców ich krawędzi
// na poziomie lv i porządkuje wnętrze na nowo. Zmiana zostaje, jeśli nie dodaje przecięć we wnętrzu.
func (l *layouter) alignPorts(lv *level, cluster *item) bool {
	child := cluster.child
	// hint: położenie drugiego końca; second rozstrzyga remisy (kilka krawędzi do jednej ramki).
	hint, second := map[*item]float64{}, map[*item]float64{}
	for _, e := range lv.edges {
		if e.upperCont != nil && e.u == cluster {
			x := e.v.x
			switch {
			case len(e.chain) > 0:
				x = e.chain[0].x
			case e.lowerCont != nil:
				// Drugi koniec też jest podgrafem: wyjścia układają się według ramek docelowych, a do
				// jednej ramki w obecnej kolejności; ramka dolna dopasuje potem swoje wejścia do nich.
				second[e.upperCont.v] = portOffset(cluster, e.upperCont.v)
			}
			hint[e.upperCont.v] = x
		}
		if e.lowerCont != nil && e.v == cluster {
			x := e.u.x
			switch {
			case len(e.chain) > 0:
				x = e.chain[len(e.chain)-1].x
			case e.upperCont != nil:
				// Drugi koniec też jest podgrafem: liczy się punkt na jego ramce, nie środek ramki.
				x = e.u.x + portOffset(e.u, e.upperCont.v)
			}
			hint[e.lowerCont.u] = x
		}
	}
	if len(hint) < 2 {
		return false
	}
	before := make([][]*item, len(child.ranks))
	for i, r := range child.ranks {
		before[i] = append([]*item(nil), r...)
	}
	count := crossings(child.ranks)
	moved := false
	for _, row := range child.ranks {
		if !portsOnly(row) {
			continue
		}
		sorted := append([]*item(nil), row...)
		sort.SliceStable(sorted, func(a, b int) bool {
			if hint[sorted[a]] != hint[sorted[b]] {
				return hint[sorted[a]] < hint[sorted[b]]
			}
			return second[sorted[a]] < second[sorted[b]]
		})
		for i, it := range sorted {
			moved = moved || row[i] != it
			row[i], it.order = it, i
		}
	}
	if !moved {
		return false
	}
	child.pinPorts = true
	l.order(child)
	child.pinPorts = false
	if crossings(child.ranks) > count {
		for i, r := range before {
			copy(child.ranks[i], r)
			for k, it := range r {
				it.order = k
			}
		}
		return false
	}
	return true
}

// finish rozmieszcza uporządkowany poziom i wyznacza jego rozmiar z marginesami.
func (l *layouter) finish(lv *level) {
	defer l.enter(lv)()
	contentW, contentH := l.position(lv)

	// Marginesy podgrafu: tytuł leży przy rzeczywistej górnej krawędzi ramki.
	pad := l.u(20)
	padX0, padX1, padY0, padY1 := pad, pad, pad, pad
	// Warstwa punktów na ramce sama odsuwa zawartość od ramki; zwykły margines byłby drugim odstępem.
	if lv.sub != nil {
		for _, it := range lv.items {
			if it.kind == kindPort && it.portTop {
				padY0 -= pad
				break
			}
		}
		for _, it := range lv.items {
			if it.kind == kindPort && !it.portTop {
				padY1 -= pad
				break
			}
		}
	}
	if lv.sub != nil {
		titleW, titleH := l.textSize(lv.titleLines)
		top := titleH + l.u(10)
		if l.titleStrip(lv) {
			top = titleH + 2*l.u(6) + l.u(8) // pasek tytułu i odstęp pod nim
		}
		// Stan złożony ma też wąski pasek u dołu, w kolorze paska tytułu.
		bottom := 0.0
		if l.titleStrip(lv) {
			bottom = l.u(footStrip)
		}
		switch {
		case l.vertical && !l.flip:
			padY0, padY1 = padY0+top, padY1+bottom
		case l.vertical:
			padY1, padY0 = padY1+top, padY0+bottom
		default:
			padX0, padX1 = padX0+top, padX1+bottom
		}
		// Ramka musi pomieścić tytuł na szerokość rzeczywistą.
		need := titleW + 2*l.u(12)
		if l.vertical {
			if extra := need - (contentW + padX0 + padX1); extra > 0 {
				padX0, padX1 = padX0+extra/2, padX1+extra/2
			}
		} else if extra := need - (contentH + padY0 + padY1); extra > 0 {
			padY0, padY1 = padY0+extra/2, padY1+extra/2
		}
	} else {
		padX0, padX1, padY0, padY1 = 0, 0, 0, 0
	}
	if chainBlock(lv) {
		padX0, padX1, padY0, padY1 = 0, 0, 0, 0 // blok niewidoczny: bez marginesów i tytułu
	}
	lv.padX0, lv.padY0 = padX0, padY0
	lv.w, lv.h = contentW+padX0+padX1, contentH+padY0+padY1
}

// rank usuwa cykle, dzieli pola na warstwy i wstawia punkty pomocnicze na długich krawędziach.
func (l *layouter) rank(lv *level) {
	defer l.enter(lv)()
	// Krawędź z etykietą jest o jedną warstwę dłuższa: etykieta zajmuje w tej warstwie miejsce jak
	// pole, obok pól innych gałęzi. Krawędzie bez etykiet zostają krótkie. Krawędzie wielokrotne
	// między tą samą parą pól też się wydłużają: każda dostaje w warstwie pośredniej własny punkt.
	// Liczą się rzeczywiste końce ze źródła: dwie krawędzie z różnych pól podgrafu do tego samego
	// celu mają na ramce osobne punkty i nie są krawędzią wielokrotną.
	count := map[[2]string]int{}
	pair := func(e *ledge) [2]string {
		if e.e.From > e.e.To {
			return [2]string{e.e.To, e.e.From}
		}
		return [2]string{e.e.From, e.e.To}
	}
	for _, e := range lv.edges {
		if e.e.From != "" {
			count[pair(e)]++
		}
	}
	extra := map[*ledge]int{}
	for _, e := range lv.edges {
		// Opisy przy obu końcach potrzebują miejsca tak jak etykieta.
		if len(e.e.Label) > 0 || count[pair(e)] > 1 || (e.e.StartLabel != "" && e.e.EndLabel != "") {
			extra[e] = 1
		}
	}
	index := map[*item]int{}
	for i, it := range lv.items {
		index[it] = i
	}
	out := make([][]*ledge, len(lv.items))
	for _, e := range lv.edges {
		out[index[e.u]] = append(out[index[e.u]], e)
	}

	// Usunięcie cykli: krawędź prowadząca do pola będącego na ścieżce przeszukiwania jest odwracana.
	const white, gray, black = 0, 1, 2
	color := make([]int, len(lv.items))
	var visit func(i int)
	visit = func(i int) {
		color[i] = gray
		for _, e := range out[i] {
			switch j := index[e.v]; color[j] {
			case white:
				visit(j)
			case gray:
				e.rev = true
			}
		}
		color[i] = black
	}
	// Przeszukiwanie zaczyna się od pól, do których nic nie prowadzi (stan początkowy, pierwszy krok
	// procesu), żeby odwracane były krawędzie powrotne, a nie te, które wyznaczają główny kierunek.
	entered := make([]bool, len(lv.items))
	for _, e := range lv.edges {
		entered[index[e.v]] = true
	}
	for pass := 0; pass < 2; pass++ {
		for i := range lv.items {
			if color[i] == white && (pass == 1 || !entered[i]) {
				visit(i)
			}
		}
	}
	for _, e := range lv.edges {
		if e.rev {
			e.u, e.v = e.v, e.u
		}
		if !e.part {
			e.topID, e.bottomID = e.e.From, e.e.To
			if e.rev {
				e.topID, e.bottomID = e.bottomID, e.topID
			}
		}
	}

	// LAY-027: w diagramie torów krawędź między torami może zostać w jednej warstwie i biec
	// w poprzek, prosto od boku pola do boku pola, jak przekazanie sprawy między rolami. Dotyczy
	// krawędzi bez etykiety i bez długości nadanej przez autora; płaska krawędź nie może przechodzić
	// przez pole w torze pośrednim ani biec po płaskiej krawędzi tej samej warstwy (wtedy wraca do
	// zwykłej długości i podział liczony jest od nowa).
	zero := map[*ledge]bool{}
	if l.lanes != nil {
		for _, e := range lv.edges {
			if e.e.From != "" && e.u != e.v && e.u.kind == kindNode && e.v.kind == kindNode && e.u.lane != e.v.lane &&
				extra[e] == 0 && e.e.MinLen <= 1 && e.e.Style != graph.Invisible {
				zero[e] = true
			}
		}
	}
	// Warstwy: najdłuższa ścieżka od źródeł, z minimalną długością krawędzi.
	minLen := func(e *ledge) int {
		if zero[e] {
			return 0
		}
		return e.e.MinLen + extra[e]
	}
	indeg := make([]int, len(lv.items))
	succ := make([][]*ledge, len(lv.items))
	for _, e := range lv.edges {
		indeg[index[e.v]]++
		succ[index[e.u]] = append(succ[index[e.u]], e)
	}
	var rank []int
	for {
		rank = make([]int, len(lv.items))
		indeg := append([]int(nil), indeg...)
		// Punkty wejścia leżą w warstwie zerowej, nad całą zawartością podgrafu.
		hasTopPort := false
		for _, it := range lv.items {
			hasTopPort = hasTopPort || (it.kind == kindPort && it.portTop)
		}
		if hasTopPort {
			for i, it := range lv.items {
				if it.kind != kindPort || !it.portTop {
					rank[i] = 1
				}
			}
		}
		var queue []int
		for i := range lv.items {
			if indeg[i] == 0 {
				queue = append(queue, i)
			}
		}
		for len(queue) > 0 {
			i := queue[0]
			queue = queue[1:]
			for _, e := range succ[i] {
				j := index[e.v]
				rank[j] = max(rank[j], rank[i]+minLen(e))
				if indeg[j]--; indeg[j] == 0 {
					queue = append(queue, j)
				}
			}
		}
		// Źródła dosuwamy do następników, żeby nie wisiały wysoko nad resztą.
		for i := range lv.items {
			if len(succ[i]) == 0 || lv.items[i].kind == kindPort {
				continue
			}
			hasIn := false
			for _, e := range lv.edges {
				if e.v == lv.items[i] {
					hasIn = true
					break
				}
			}
			if hasIn {
				continue
			}
			best := math.MaxInt
			for _, e := range succ[i] {
				best = min(best, rank[index[e.v]]-minLen(e))
			}
			rank[i] = max(rank[i], best)
		}
		// Podział z najdłuższej ścieżki jest dopuszczalny; sympleks sieciowy skraca w nim krawędzie.
		// Punkty na ramce nie biorą udziału: ich warstwy są ustalone.
		var inner []int
		number := map[int]int{}
		for i, it := range lv.items {
			if it.kind != kindPort {
				number[i] = len(inner)
				inner = append(inner, i)
			}
		}
		var triples [][3]int
		for _, e := range lv.edges {
			a, okA := number[index[e.u]]
			b, okB := number[index[e.v]]
			if okA && okB {
				triples = append(triples, [3]int{a, b, minLen(e)})
			}
		}
		if len(triples) > 0 {
			local := make([]int, len(inner))
			for k, i := range inner {
				local[k] = rank[i]
			}
			networkSimplex(len(inner), triples, local)
			base := 0
			if hasTopPort {
				base = 1
			}
			for k, i := range inner {
				rank[i] = local[k] + base
			}
		}
		if l.lanes != nil {
			l.placeLoose(lv, rank, index)
		}
		bad := l.flatConflicts(lv, rank, index, zero)
		if len(bad) == 0 {
			break
		}
		for _, e := range bad {
			delete(zero, e)
		}
	}
	for e := range zero {
		e.flat = rank[index[e.u]] == rank[index[e.v]]
	}
	if l.rankLimit > 0 {
		l.boundRankWidth(lv, rank, succ, index, minLen)
	}
	// Punkty wyjścia leżą w osobnej, ostatniej warstwie, pod całą zawartością podgrafu.
	last := 0
	for i, it := range lv.items {
		if it.kind != kindPort || it.portTop {
			last = max(last, rank[i])
		}
	}
	maxRank := 0
	for i, it := range lv.items {
		if it.kind == kindPort && !it.portTop {
			rank[i] = last + 1
		}
		it.rank = rank[i]
		maxRank = max(maxRank, rank[i])
	}
	for _, it := range lv.items {
		if it.anchor != nil {
			it.rank = it.anchor.rank
		}
	}

	// Punkty pomocnicze i pole etykiety na każdej krawędzi dłuższej niż jedna warstwa.
	for _, e := range lv.edges {
		if e.flat {
			continue // płaska krawędź nie ma punktów pośrednich ani sąsiadów w warstwach obok
		}
		span := e.v.rank - e.u.rank
		labelRank := -1
		if len(e.e.Label) > 0 {
			labelRank = e.u.rank + span/2
		}
		prev := e.u
		for r := e.u.rank + 1; r < e.v.rank; r++ {
			// Punkty pośrednie biegną w torze dolnego końca: krawędź zmienia tor raz, przy początku.
			d := &item{kind: kindDummy, rank: r, w: l.u(8), lane: e.v.lane}
			if r == labelRank {
				d.kind = kindLabel
				d.text = l.wrap(e.e.Label, l.u(l.baseWrap())*l.narrow())
				tw, th := l.textSize(d.text)
				d.w, d.h = l.abstract(tw+l.u(10), th+l.u(4))
				e.label = d
			}
			e.chain = append(e.chain, d)
			prev.down = append(prev.down, d)
			d.up = append(d.up, prev)
			prev = d
		}
		prev.down = append(prev.down, e.v)
		e.v.up = append(e.v.up, prev)
	}

	// Kolejność początkowa: przeszukiwanie w głąb od pól w kolejności źródła.
	lv.ranks = make([][]*item, maxRank+1)
	seen := map[*item]bool{}
	var walk func(it *item)
	walk = func(it *item) {
		if seen[it] {
			return
		}
		seen[it] = true
		lv.ranks[it.rank] = append(lv.ranks[it.rank], it)
		for _, d := range it.down {
			walk(d)
		}
	}
	for _, it := range lv.items {
		walk(it)
	}
	for _, r := range lv.ranks {
		for i, it := range r {
			it.order = i
		}
	}
}

// crossings liczy przecięcia krawędzi między sąsiednimi warstwami.
func crossings(ranks [][]*item) int {
	total := 0
	for r := 0; r+1 < len(ranks); r++ {
		var pairs [][2]int
		for _, it := range ranks[r] {
			for _, d := range it.down {
				pairs = append(pairs, [2]int{it.order, d.order})
			}
		}
		sort.Slice(pairs, func(a, b int) bool {
			if pairs[a][0] != pairs[b][0] {
				return pairs[a][0] < pairs[b][0]
			}
			return pairs[a][1] < pairs[b][1]
		})
		// Liczba inwersji drugiej współrzędnej, drzewem Fenwicka.
		tree := make([]int, len(ranks[r+1])+1)
		for k, p := range pairs {
			seenLE := 0
			for i := p[1] + 1; i > 0; i -= i & -i {
				seenLE += tree[i]
			}
			total += k - seenLE
			for i := p[1] + 1; i < len(tree); i += i & -i {
				tree[i]++
			}
		}
	}
	return total
}

// boundRankWidth ogranicza szerokość warstw diagramu encji. Kierunek takiego diagramu nie niesie
// znaczenia, więc o układzie mogą decydować proporcje strony: gdy warstwa jest za szeroka, pola,
// których przesunięcie wydłuża najmniej krawędzi (najpierw liście), schodzą do warstwy niżej,
// a rysunek rośnie wzdłuż osi warstw zamiast w poprzek. Minimalne długości krawędzi zostają.
func (l *layouter) boundRankWidth(lv *level, rank []int, succ [][]*ledge, index map[*item]int, minLen func(*ledge) int) {
	n := len(lv.items)
	if n < 2 {
		return
	}
	gapX, gapY := l.u(48), l.u(110)
	// Szerokość docelowa: pole zajęte przez wszystkie encje ułożone w proporcjach strony.
	area, widest := 0.0, 0.0
	for _, it := range lv.items {
		area += (it.w + gapX) * (it.h + gapY)
		widest = math.Max(widest, it.w+gapX)
	}
	pageW, pageH := l.u(650), l.u(870)
	if !l.vertical {
		pageW, pageH = pageH, pageW
	}
	limit := math.Max(2*widest, l.rankLimit*math.Max(pageW, math.Sqrt(area*pageW/pageH)))
	indeg := make([]int, n)
	for _, e := range lv.edges {
		indeg[index[e.v]]++
	}
	var push func(i, to int)
	push = func(i, to int) {
		if rank[i] >= to {
			return
		}
		rank[i] = to
		for _, e := range succ[i] {
			push(index[e.v], to+minLen(e))
		}
	}
	// Szerokość warstwy: jej pola oraz punkty i etykiety krawędzi, które przez nią przechodzą.
	width := func(r int) (total float64, members []int) {
		for i, it := range lv.items {
			if rank[i] == r {
				total += it.w + gapX
				members = append(members, i)
			}
		}
		for _, e := range lv.edges {
			a, b := rank[index[e.u]], rank[index[e.v]]
			if a < r && r < b {
				total += l.u(18)
				if len(e.e.Label) > 0 && r == a+(b-a)/2 {
					tw, _ := l.abstract(l.textSize(l.wrap(e.e.Label, l.u(l.baseWrap())*l.narrow())))
					total += tw + l.u(10)
				}
			}
		}
		return total, members
	}
	for r := 0; r <= 4*n; r++ {
		for {
			total, members := width(r)
			if total <= limit || len(members) < 2 {
				break
			}
			// Schodzi pole, którego przesunięcie kosztuje najmniej: mało krawędzi wchodzących
			// (te się wydłużają), dużo wychodzących; przy remisie późniejsze w źródle.
			best := members[0]
			for _, i := range members[1:] {
				if indeg[i]-len(succ[i]) <= indeg[best]-len(succ[best]) {
					best = i
				}
			}
			push(best, r+1)
		}
	}
}

// shelve układa części poziomu niepołączone krawędziami w rzędy, tak żeby całość miała proporcje
// zbliżone do strony: części stojące w jednym rzędzie obok siebie, kolejne rzędy jeden za drugim
// wzdłuż osi warstw. Bez tego wiele części dawałoby rysunek bardzo szeroki, a ustawione jedna za
// drugą bardzo długi. Kolejność części ze źródła zostaje.
func (l *layouter) shelve(lv *level) {
	for _, it := range lv.items {
		if it.kind == kindPort {
			return // warstwy punktów na ramce muszą zostać pierwszą i ostatnią
		}
	}
	comps := components(lv)
	if len(comps) < 2 {
		return
	}
	type part struct {
		rows   [][]*item
		w, h   float64
		ranks  int
		lo, hi int
	}
	parts := make([]part, len(comps))
	for k, rows := range comps {
		p := part{rows: rows, lo: -1}
		for r, row := range rows {
			if len(row) == 0 {
				continue
			}
			if p.lo < 0 {
				p.lo = r
			}
			p.hi = r
			width, band := -l.u(40), 0.0
			for _, it := range row {
				width += it.w + l.u(40)
				band = math.Max(band, it.h)
			}
			p.w = math.Max(p.w, width)
			p.h += band + l.u(44)
		}
		p.ranks = p.hi - p.lo + 1
		parts[k] = p
	}
	// Poziom złożony z samych ramek bez połączeń (regiony współbieżne, osobne podgrafy) pakujemy
	// swobodnie, bez wspólnych warstw: niższe ramki mogą stanąć jedna pod drugą obok wyższej.
	boxes := true
	for _, p := range parts {
		n := 0
		for _, row := range p.rows {
			for _, it := range row {
				n++
				boxes = boxes && it.kind == kindCluster
			}
		}
		boxes = boxes && n == 1
	}
	if boxes {
		l.packBoxes(lv)
		return
	}
	// Miarą jest skala, w jakiej rysunek zmieści się na stronie (około 650 × 870 px przy piśmie 16 px,
	// czyli A4 albo Letter z marginesami): im większa, tym czytelniej; przy równej wygrywa mniejsze pole.
	pageW, pageH := l.u(650), l.u(870)
	if !l.vertical {
		pageW, pageH = pageH, pageW
	}
	type slot struct{ shelf, column, offset int }
	// pack ustawia części kolejno: pod poprzednią, jeśli kolumna nie stanie się przez to dłuższa od
	// najdłuższej w rzędzie (stack); obok, jeśli rząd nie przekroczy szerokości limit; inaczej w nowym rzędzie.
	pack := func(limit float64, stack bool) (slots []slot, depth []int, scale, area float64) {
		slots = make([]slot, len(parts))
		var colW, colH []float64 // szerokości i wysokości kolumn bieżącego rzędu
		var colRanks []int
		width, height := 0.0, 0.0
		rowWidth := func() float64 {
			w := -l.u(48)
			for _, c := range colW {
				w += c + l.u(48)
			}
			return w
		}
		closeRow := func() {
			if len(colW) == 0 {
				return
			}
			h, d := 0.0, 0
			for i := range colW {
				h, d = math.Max(h, colH[i]), max(d, colRanks[i])
			}
			width, height = math.Max(width, rowWidth()), height+h
			depth = append(depth, d)
			colW, colH, colRanks = nil, nil, nil
		}
		for k, p := range parts {
			last := len(colW) - 1
			tallest := 0
			for i := 0; i < last; i++ {
				tallest = max(tallest, colRanks[i])
			}
			switch {
			case stack && last >= 1 && colRanks[last]+p.ranks <= tallest:
				slots[k] = slot{len(depth), last, colRanks[last]}
				colW[last], colH[last], colRanks[last] = math.Max(colW[last], p.w), colH[last]+p.h, colRanks[last]+p.ranks
			case last >= 0 && rowWidth()+l.u(48)+p.w <= limit+0.5:
				slots[k] = slot{len(depth), last + 1, 0}
				colW, colH, colRanks = append(colW, p.w), append(colH, p.h), append(colRanks, p.ranks)
			default:
				closeRow()
				slots[k] = slot{len(depth), 0, 0}
				colW, colH, colRanks = []float64{p.w}, []float64{p.h}, []int{p.ranks}
			}
		}
		closeRow()
		return slots, depth, math.Min(1, math.Min(pageW/width, pageH/height)), width * height
	}
	var best []slot
	var bestDepth []int
	bestScale, bestArea, limit := -1.0, 0.0, -l.u(48)
	for _, p := range parts {
		limit += l.u(48) + p.w // kolejne dopuszczalne szerokości rzędu: tyle, ile zajmują pierwsze części
		for _, candidate := range []float64{limit, p.w} {
			for _, stack := range []bool{false, true} {
				slots, depth, scale, area := pack(candidate, stack)
				if scale > bestScale+1e-9 || (scale > bestScale-1e-9 && area < bestArea-0.5) {
					best, bestDepth, bestScale, bestArea = slots, depth, scale, area
				}
			}
		}
	}
	single := true
	for _, sl := range best {
		single = single && sl.shelf == 0 && sl.offset == 0
	}
	if single {
		return
	}
	base := make([]int, len(bestDepth)+1)
	for i, d := range bestDepth {
		base[i+1] = base[i] + d
	}
	ranks := make([][]*item, base[len(bestDepth)])
	for k, p := range parts {
		sl := best[k]
		for r := p.lo; r <= p.hi; r++ {
			for _, it := range p.rows[r] {
				it.rank, it.shelf, it.column = base[sl.shelf]+sl.offset+r-p.lo, sl.shelf, sl.column
				ranks[it.rank] = append(ranks[it.rank], it)
			}
		}
	}
	lv.ranks, lv.shelved = ranks, true
	for _, row := range lv.ranks {
		sort.SliceStable(row, func(a, b int) bool { return row[a].column < row[b].column })
		for i, it := range row {
			it.order = i
		}
	}
}

// packBoxes rozmieszcza niepołączone ramki poziomu w rzędach i kolumnach: ramka staje pod
// poprzednią, jeśli kolumna nie będzie przez to wyższa od najwyższej w rzędzie; obok, jeśli rząd
// mieści się w limicie szerokości; inaczej w nowym rzędzie. Spośród limitów wygrywa ten, przy
// którym całość mieści się na stronie w największej skali. Kolejność ramek ze źródła zostaje.
func (l *layouter) packBoxes(lv *level) {
	var items []*item
	for _, row := range lv.ranks {
		items = append(items, row...)
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].child.sub.Line < items[b].child.sub.Line })
	gapX, gapY := l.u(40), l.u(36)
	pageW, pageH := l.u(650), l.u(870)
	if !l.vertical {
		pageW, pageH = pageH, pageW
	}
	type column struct {
		w, h  float64
		items []*item
	}
	pack := func(limit, slack float64) (rows [][]column, width, height float64) {
		var cur []column
		rowWidth := func() float64 {
			w := -gapX
			for _, c := range cur {
				w += c.w + gapX
			}
			return w
		}
		flush := func() {
			if len(cur) == 0 {
				return
			}
			h := 0.0
			for _, c := range cur {
				h = math.Max(h, c.h)
			}
			if len(rows) > 0 {
				height += gapY
			}
			width, height = math.Max(width, rowWidth()), height+h
			rows, cur = append(rows, cur), nil
		}
		for _, it := range items {
			last := len(cur) - 1
			tallest := 0.0
			for i := 0; i < last; i++ {
				tallest = math.Max(tallest, cur[i].h)
			}
			switch {
			case last >= 1 && cur[last].h+gapY+it.h <= tallest*(1+slack)+0.5:
				c := &cur[last]
				c.w, c.h, c.items = math.Max(c.w, it.w), c.h+gapY+it.h, append(c.items, it)
			case last >= 0 && rowWidth()+gapX+it.w <= limit+0.5:
				cur = append(cur, column{it.w, it.h, []*item{it}})
			default:
				flush()
				cur = []column{{it.w, it.h, []*item{it}}}
			}
		}
		flush()
		return rows, width, height
	}
	var best [][]column
	bestScale, bestArea, limit := -1.0, 0.0, -gapX
	for _, it := range items {
		limit += gapX + it.w
		for _, candidate := range []float64{limit, it.w} {
			// Kolumna może też nieco przerosnąć najwyższą ramkę rzędu, jeśli całość na tym zyskuje.
			for _, slack := range []float64{0, 0.25} {
				rows, w, h := pack(candidate, slack)
				scale, area := math.Min(1, math.Min(pageW/w, pageH/h)), w*h
				if scale > bestScale+1e-9 || (scale > bestScale-1e-9 && area < bestArea-0.5) {
					best, bestScale, bestArea = rows, scale, area
				}
			}
		}
	}
	lv.free = map[*item][2]float64{}
	y := 0.0
	for _, row := range best {
		x, rowH := 0.0, 0.0
		for _, c := range row {
			top := y
			for _, it := range c.items {
				lv.free[it] = [2]float64{x + it.w/2, top + it.h/2}
				top += it.h + gapY
			}
			x += c.w + gapX
			rowH = math.Max(rowH, c.h)
		}
		y += rowH + gapY
	}
}

// attachments zwraca, dla pary (pole, podgraf), przesunięcie względem środka podgrafu punktu na
// jego ramce, przez który przechodzi krawędź łącząca tę parę. Dzięki temu sąsiad podgrafu
// ustawia się pod punktem wyjścia krawędzi, a nie pod środkiem ramki.
func (l *layouter) attachments(lv *level) map[[2]*item]float64 {
	sum, count := map[[2]*item]float64{}, map[[2]*item]float64{}
	add := func(other, cluster, port *item) {
		k := [2]*item{other, cluster}
		sum[k] += portOffset(cluster, port)
		count[k]++
	}
	for _, e := range lv.edges {
		if e.upperCont != nil {
			below := e.v
			if len(e.chain) > 0 {
				below = e.chain[0]
			}
			add(below, e.u, e.upperCont.v)
		}
		if e.lowerCont != nil {
			above := e.u
			if len(e.chain) > 0 {
				above = e.chain[len(e.chain)-1]
			}
			add(above, e.v, e.lowerCont.u)
		}
	}
	for k := range sum {
		sum[k] /= count[k]
	}
	return sum
}

// order zmniejsza liczbę przecięć: przebiegi barycentryczne w dół i w górę, potem zamiany sąsiadów.
func (l *layouter) order(lv *level) {
	snapshot := func() [][]*item {
		out := make([][]*item, len(lv.ranks))
		for i, r := range lv.ranks {
			out[i] = append([]*item(nil), r...)
		}
		return out
	}
	restore := func(s [][]*item) {
		for i, r := range s {
			lv.ranks[i] = append(lv.ranks[i][:0], r...)
			for k, it := range lv.ranks[i] {
				it.order = k
			}
		}
	}
	// flipTies: pola o równym położeniu sąsiadów zamieniają kolejność. Zwykły przebieg zostawia je
	// tak, jak stały, i potrafi utknąć w układzie, z którego wyprowadza dopiero zamiana w dwóch
	// warstwach naraz (np. dwie krawędzie z jednego pola, z których jedna mija pole celu drugiej).
	flipTies := false
	sweep := func(r int, neighbours func(*item) []*item) {
		row := lv.ranks[r]
		if lv.pinPorts && portsOnly(row) {
			return
		}
		type keyed struct {
			it  *item
			key float64
		}
		var movable []keyed
		fixed := make([]bool, len(row))
		for i, it := range row {
			ns := neighbours(it)
			if len(ns) == 0 {
				fixed[i] = true // pole bez sąsiadów zostaje na swoim miejscu
				continue
			}
			sum := 0.0
			for _, n := range ns {
				sum += float64(n.order)
			}
			movable = append(movable, keyed{it, sum / float64(len(ns))})
		}
		if flipTies {
			for i, j := 0, len(movable)-1; i < j; i, j = i+1, j-1 {
				movable[i], movable[j] = movable[j], movable[i]
			}
		}
		sort.SliceStable(movable, func(a, b int) bool { return movable[a].key < movable[b].key })
		next := make([]*item, len(row))
		k := 0
		for i := range row {
			if fixed[i] {
				next[i] = row[i]
			} else {
				next[i] = movable[k].it
				k++
			}
		}
		copy(row, next)
		for i, it := range row {
			it.order = i
		}
	}
	best, bestCount := snapshot(), crossings(lv.ranks)
	if lv.pinPorts {
		bestCount = math.MaxInt // kolejność zastana powstała przy innych punktach na ramce
	}
	for iter := 0; iter < 32 && bestCount > 0; iter++ {
		// Po zwykłych przebiegach, jeśli przecięcia zostały: od najlepszego układu jeden przebieg
		// z odwróconymi remisami, potem znów zwykłe.
		if iter >= 24 && iter%4 == 0 {
			restore(best)
		}
		flipTies = iter >= 24 && iter%4 < 2
		if iter%2 == 0 {
			for r := 1; r < len(lv.ranks); r++ {
				sweep(r, func(it *item) []*item { return it.up })
			}
		} else {
			for r := len(lv.ranks) - 2; r >= 0; r-- {
				sweep(r, func(it *item) []*item { return it.down })
			}
		}
		if c := crossings(lv.ranks); c < bestCount {
			best, bestCount = snapshot(), c
		}
	}
	restore(best)

	// Zamiana sąsiadów: para zamienia się miejscami, jeśli zmniejsza to liczbę przecięć jej krawędzi.
	cross := func(a, b []*item) int {
		n := 0
		for _, x := range a {
			for _, y := range b {
				if x.order > y.order {
					n++
				}
			}
		}
		return n
	}
	for round := 0; round < 8 && bestCount > 0; round++ {
		improved := false
		for _, row := range lv.ranks {
			if lv.pinPorts && portsOnly(row) {
				continue
			}
			for i := 0; i+1 < len(row); i++ {
				a, b := row[i], row[i+1]
				now := cross(a.up, b.up) + cross(a.down, b.down)
				swapped := cross(b.up, a.up) + cross(b.down, a.down)
				if swapped < now {
					row[i], row[i+1] = b, a
					a.order, b.order = i+1, i
					improved = true
				}
			}
		}
		if !improved {
			break
		}
	}

	// Krawędzie wychodzące z jednego podgrafu przez różne punkty ramki: ich sąsiednie pola stoją
	// w kolejności tych punktów, o ile nie dodaje to przecięć gdzie indziej.
	attach := l.attachments(lv)
	shared := func(a, b *item) (float64, float64, bool) {
		for _, side := range [][]*item{a.up, a.down} {
			for _, c := range side {
				off, ok := attach[[2]*item{a, c}]
				if !ok {
					continue
				}
				if other, ok := attach[[2]*item{b, c}]; ok {
					return off, other, true
				}
			}
		}
		return 0, 0, false
	}
	for _, row := range lv.ranks {
		for round := 0; round < len(row); round++ {
			swappedAny := false
			for i := 0; i+1 < len(row); i++ {
				a, b := row[i], row[i+1]
				offA, offB, ok := shared(a, b)
				if !ok || offA <= offB {
					continue
				}
				if cross(b.up, a.up)+cross(b.down, a.down) > cross(a.up, b.up)+cross(a.down, b.down) {
					continue
				}
				row[i], row[i+1] = b, a
				a.order, b.order = i+1, i
				swappedAny = true
			}
			if !swappedAny {
				break
			}
		}
	}

	// Notatki nie mają krawędzi, więc porządkowanie ich nie rusza: stawiamy je obok pola, którego dotyczą.
	for _, row := range lv.ranks {
		var notes []*item
		for _, it := range row {
			if it.anchor != nil {
				notes = append(notes, it)
			}
		}
		for _, n := range notes {
			rest := make([]*item, 0, len(row))
			for _, it := range row {
				if it != n {
					rest = append(rest, it)
				}
			}
			at := 0
			for i, it := range rest {
				if it == n.anchor {
					at = i
					if n.anchorSide > 0 {
						at = i + 1
					}
				}
			}
			copy(row, append(rest[:at:at], append([]*item{n}, rest[at:]...)...))
		}
		for i, it := range row {
			it.order = i
		}
	}
}

// position wyznacza współrzędne pól poziomu i zwraca rozmiar zawartości.
func (l *layouter) position(lv *level) (width, height float64) {
	if chainBlock(lv) {
		return l.chainPositions(lv)
	}
	if lv.free != nil {
		for _, row := range lv.ranks {
			for _, it := range row {
				at := lv.free[it]
				it.x, it.y, it.band = math.Round(at[0]), math.Round(at[1]), it.h
				width, height = math.Max(width, it.x+it.w/2), math.Max(height, it.y+it.h/2)
			}
		}
		return width, height
	}
	// LAY-025: przebieg krawędzi do dalszego liścia stoi między bliższymi liśćmi; odstęp od nich
	// jest taki, żeby para „liść + przebieg” zajmowała tyle, co dalszy liść z odstępem, i przebieg
	// trafiał w środek dalszego liścia.
	spread := map[*item]bool{}
	for _, e := range lv.edges {
		if e.e != nil && e.e.Spread {
			for _, d := range e.chain {
				spread[d] = true
			}
		}
	}
	gap := func(a, b *item) float64 {
		switch {
		case a.real() && b.real():
			return l.u(48)
		case a.kind == kindDummy && b.kind == kindDummy:
			return l.u(10)
		case a.real() && spread[b]:
			return math.Max(l.u(18), (l.u(48)-b.w)/2)
		case b.real() && spread[a]:
			return math.Max(l.u(18), (l.u(48)-a.w)/2)
		}
		return l.u(18)
	}
	sep := func(a, b *item) float64 { return a.w/2 + b.w/2 + gap(a, b) }
	attach := l.attachments(lv)
	// arrange rozmieszcza wzdłuż osi kolejności jedną spójną część poziomu.
	if l.lanes != nil {
		l.laneExtents(lv, sep)
	}
	arrange := func(rows [][]*item) {
		for _, row := range rows {
			for i, it := range row {
				if i == 0 {
					it.x = it.w / 2
				} else {
					it.x = row[i-1].x + sep(row[i-1], it)
				}
				if l.lanes != nil {
					it.x = (l.lanes[it.lane].lo + l.lanes[it.lane].hi) / 2 // początek: środek toru
				}
			}
			if l.lanes != nil {
				l.confine(row, sep)
			}
		}
		weight := func(a, b *item) float64 {
			switch {
			case !a.real() && !b.real():
				return 8 // długie krawędzie mają biec prosto
			case !a.real() || !b.real():
				return 2
			}
			return 1
		}
		// Metoda priorytetowa (Sugiyama, Tagawa, Toda 1981): pola warstwy są ustawiane w kolejności
		// malejącego priorytetu; każde przesuwa się do pozycji pożądanej tak daleko, jak pozwalają
		// pola już ustawione, i odsuwa pola o niższym priorytecie. Punkty pomocnicze mają najwyższy
		// priorytet, dzięki czemu długie krawędzie biegną prosto.
		place := func(row []*item, want []float64, priority []float64) {
			n := len(row)
			fixed := make([]bool, n)
			idx := make([]int, n)
			for i := range idx {
				idx[i] = i
			}
			sort.SliceStable(idx, func(a, b int) bool { return priority[idx[a]] > priority[idx[b]] })
			for _, i := range idx {
				target := want[i]
				if target > row[i].x {
					// Granica: najbliższe ustawione pole po prawej minus odstępy pól pomiędzy.
					limit, need := math.Inf(1), 0.0
					for j := i + 1; j < n; j++ {
						need += sep(row[j-1], row[j])
						if fixed[j] {
							limit = row[j].x - need
							break
						}
					}
					row[i].x = math.Max(row[i].x, math.Min(target, limit))
					for j := i + 1; j < n && !fixed[j]; j++ {
						row[j].x = math.Max(row[j].x, row[j-1].x+sep(row[j-1], row[j]))
					}
				} else if target < row[i].x {
					limit, need := math.Inf(-1), 0.0
					for j := i - 1; j >= 0; j-- {
						need += sep(row[j], row[j+1])
						if fixed[j] {
							limit = row[j].x + need
							break
						}
					}
					row[i].x = math.Min(row[i].x, math.Max(target, limit))
					for j := i - 1; j >= 0 && !fixed[j]; j-- {
						row[j].x = math.Min(row[j].x, row[j+1].x-sep(row[j], row[j+1]))
					}
				}
				fixed[i] = true
			}
			if l.lanes != nil {
				l.confine(row, sep)
			}
		}
		pull := func(row []*item, useUp, useDown bool) {
			want := make([]float64, len(row))
			priority := make([]float64, len(row))
			for i, it := range row {
				sum, total, count := 0.0, 0.0, 0
				add := func(ns []*item, siblings func(*item) []*item) {
					for _, n := range ns {
						w := weight(it, n)
						if off, ok := attach[[2]*item{it, n}]; ok {
							sum += w * (n.nodeX() + off) // pod punktem na ramce podgrafu
						} else if off, ok := attach[[2]*item{n, it}]; ok {
							sum += w * (n.nodeX() - off)
						} else {
							sum += w * (n.nodeX() + fanOffset(it, siblings(n), sep))
						}
						total += w
						count++
					}
				}
				if useUp {
					add(it.up, func(n *item) []*item { return n.down })
				}
				if useDown {
					add(it.down, func(n *item) []*item { return n.up })
				}
				want[i] = it.x
				if total > 0 {
					// Osie liczą się między samymi polami: miejsce na pętlę własną stoi z boku.
					want[i] = sum/total + it.loopExtra/2
					priority[i] = float64(count)
					if count == 1 {
						// Pole z jednym sąsiadem może stanąć dokładnie w jego osi; ma pierwszeństwo
						// przed polem, które i tak uśrednia położenie dwóch sąsiadów.
						priority[i] = 2.5
					}
					if !it.real() {
						priority[i] += 1000
					}
				} else {
					priority[i] = -1 // pole bez sąsiadów po tej stronie ustępuje wszystkim
				}
			}
			place(row, want, priority)
		}
		for iter := 0; iter < 4; iter++ {
			for r := 1; r < len(rows); r++ {
				pull(rows[r], true, false)
			}
			for r := len(rows) - 2; r >= 0; r-- {
				pull(rows[r], false, true)
			}
		}
		for r := 1; r < len(rows); r++ {
			pull(rows[r], true, false)
		}
		l.settleFrames(lv, rows, place)

		// Tory: pole, które w swoim rzędzie jest jedynym polem toru, staje pośrodku pasa. Kroki
		// jednej roli tworzą wtedy kolumnę, a nie schodki ciągnięte ku sąsiadom z innych torów.
		if l.lanes != nil {
			for _, row := range rows {
				count := map[int]int{}
				for _, it := range row {
					count[it.lane]++
				}
				want := make([]float64, len(row))
				priority := make([]float64, len(row))
				for i, it := range row {
					want[i], priority[i] = it.x, 100
					if it.real() && count[it.lane] == 1 {
						ln := l.lanes[it.lane]
						want[i], priority[i] = (ln.lo+ln.hi)/2+it.loopExtra/2, 2000
					}
				}
				place(row, want, priority)
			}
			// Pola, które w rzędzie są jedynymi polami swojego toru (punkty pośrednie krawędzi się nie
			// liczą), stają w jednej osi toru: w medianie swoich położeń. Punkty pośrednie ustępują.
			solo := func(row []*item) map[*item]bool {
				count := map[int]int{}
				for _, it := range row {
					if it.real() {
						count[it.lane]++
					}
				}
				out := map[*item]bool{}
				for _, it := range row {
					if it.real() && count[it.lane] == 1 {
						out[it] = true
					}
				}
				return out
			}
			at := map[int][]float64{}
			// Zakres osi, w którym każde z tych pól mieści się w pasie (z marginesem jak w confine).
			lo, hi := map[int]float64{}, map[int]float64{}
			for _, row := range rows {
				for it := range solo(row) {
					at[it.lane] = append(at[it.lane], it.x-it.loopExtra/2)
					ln := l.lanes[it.lane]
					a, b := ln.lo+l.u(28)+it.w/2-it.loopExtra/2, ln.hi-l.u(28)-it.w/2-it.loopExtra/2
					if v, ok := lo[it.lane]; !ok || a > v {
						lo[it.lane] = a
					}
					if v, ok := hi[it.lane]; !ok || b < v {
						hi[it.lane] = b
					}
				}
			}
			axis := map[int]float64{}
			for k, xs := range at {
				sort.Float64s(xs)
				axis[k] = xs[len(xs)/2]
				if lo[k] <= hi[k] {
					axis[k] = math.Max(lo[k], math.Min(hi[k], axis[k]))
				}
			}
			for _, row := range rows {
				alone := solo(row)
				want := make([]float64, len(row))
				priority := make([]float64, len(row))
				for i, it := range row {
					want[i], priority[i] = it.x, 100
					if alone[it] {
						want[i], priority[i] = axis[it.lane]+it.loopExtra/2, 2000
					}
				}
				place(row, want, priority)
			}
		}

		// Krawędź łamana ma skręcać raz, przy końcu, z którego rozchodzi się kilka krawędzi; jej
		// punkty pośrednie i etykieta stają w osi drugiego końca, jeśli ten ma po tej stronie
		// tylko tę jedną krawędź. Pola zostają na miejscu.
		if !l.o.CurvedEdges {
			here := map[*item]bool{}
			for _, row := range rows {
				for _, it := range row {
					here[it] = true
				}
			}
			axis := map[*item]float64{}
			// nearby: skręt z początku krawędzi e do osi x byłby drobnym uskokiem. Początek w podgrafie to
			// punkt na jego ramce; początek w polu może przesunąć się w zasięgu boku pola.
			nearby := func(x float64, e *ledge) bool {
				from, reach := e.u.nodeX(), 0.0
				if e.upperCont != nil {
					from = e.u.x + portOffset(e.u, e.upperCont.v)
				} else if e.u.kind == kindNode {
					reach = l.portRange(e.u) / 2
				}
				d := math.Abs(x - from)
				return d > reach+0.5 && d < reach+l.u(16)
			}
			for _, e := range lv.edges {
				if len(e.chain) == 0 || !here[e.chain[0]] || !e.u.real() || !e.v.real() {
					continue
				}
				out, in := len(e.u.down), len(e.v.up)
				switch {
				case e.lowerCont != nil && (e.upperCont != nil || out == 1) && !nearby(e.v.x+portOffset(e.v, e.lowerCont.u), e):
					// Koniec w podgrafie ma na ramce własny punkt, do którego wchodzi tylko ta krawędź:
					// punkty pośrednie stają w jego osi, więc krawędź skręca raz, przy początku, i dochodzi
					// do pola prosto. Tylko gdy początek też jest pojedynczy (jedno wyjście z pola albo własny
					// punkt na ramce) i gdy skręt przy początku nie byłby drobnym uskokiem.
					for _, d := range e.chain {
						axis[d] = e.v.x + portOffset(e.v, e.lowerCont.u)
					}
				case in == 1 && out >= 1:
					// Koniec w podgrafie: osią jest punkt na jego ramce, nie środek ramki.
					off := attach[[2]*item{e.chain[len(e.chain)-1], e.v}]
					for _, d := range e.chain {
						axis[d] = e.v.x - e.v.loopExtra/2 + off
					}
				case out == 1:
					off := attach[[2]*item{e.chain[0], e.u}]
					for _, d := range e.chain {
						axis[d] = e.u.x - e.u.loopExtra/2 + off
					}
				case len(e.chain) > 1:
					// Oba końce mają po kilka krawędzi: punkty pośrednie stają w jednej osi, tej
					// przy dolnym końcu, żeby krawędź skręcała raz, a nie po drodze.
					for _, d := range e.chain {
						axis[d] = e.chain[len(e.chain)-1].x
					}
				}
			}
			if len(axis) > 0 {
				for _, row := range rows {
					want := make([]float64, len(row))
					priority := make([]float64, len(row))
					for i, it := range row {
						want[i], priority[i] = it.x, 100
						if x, ok := axis[it]; ok {
							want[i], priority[i] = x, 500
						} else if it.real() {
							priority[i] = 1000
						}
					}
					place(row, want, priority)
				}
				// Pole, z którego krawędzie się rozchodzą albo w którym się schodzą, staje pośrodku
				// swoich sąsiadów; pole połączone z sąsiadem jedną prostą krawędzią zostaje w jego osi.
				mean := func(ns []*item) (float64, bool) {
					if len(ns) == 0 {
						return 0, false
					}
					sum := 0.0
					for _, n := range ns {
						sum += n.nodeX()
					}
					return sum / float64(len(ns)), true
				}
				for _, row := range rows {
					want := make([]float64, len(row))
					priority := make([]float64, len(row))
					moved := false
					for i, it := range row {
						want[i], priority[i] = it.x, 1000
						straight := (len(it.up) == 1 && len(it.up[0].down) == 1) || (len(it.down) == 1 && len(it.down[0].up) == 1)
						if !it.real() || straight || it.loopExtra > 0 || len(it.up)+len(it.down) < 2 || l.lanes != nil {
							continue
						}
						up, okUp := mean(it.up)
						down, okDown := mean(it.down)
						switch {
						case okUp && okDown:
							want[i] = (up + down) / 2
						case okUp:
							want[i] = up
						default:
							want[i] = down
						}
						priority[i], moved = 500, true
					}
					if moved {
						place(row, want, priority)
					}
				}
			}
		}

		// Pole z jednym dopływem z pola, które ma tylko to jedno wyjście, staje w osi poprzednika, gdy
		// różnią się o mniej niż u(16): prosta krawędź poza środkiem jednego z pól wygląda jak rozjazd
		// łańcucha. Pole przesuwa się tylko w wolnym miejscu warstwy, bez odsuwania sąsiadów (LAY-021).
		if l.lanes == nil {
			for _, row := range rows {
				for i, it := range row {
					if it.kind != kindNode || len(it.up) != 1 {
						continue
					}
					u := it.up[0]
					if u.kind != kindNode || len(u.down) != 1 {
						continue
					}
					d := u.nodeX() - it.nodeX()
					if math.Abs(d) < 0.5 || math.Abs(d) >= l.u(16) {
						continue
					}
					if i > 0 && it.x+d < row[i-1].x+sep(row[i-1], it)-0.01 || i+1 < len(row) && it.x+d > row[i+1].x-sep(it, row[i+1])+0.01 {
						continue
					}
					it.x += d
				}
			}
		}

		// Pole albo podpis środkowy z nieparzystej liczby (co najmniej trzech) wyjść pola, do którego
		// prowadzi tylko ta krawędź, staje pod nim: krawędź z dolnego wierzchołka biegnie prosto,
		// a skrajne odchodzą symetrycznie; pole pod takim podpisem idzie za nim. Miejsca ustępują
		// tylko punkty pomocnicze i podpisy innych krawędzi, przesuwane razem z całą krawędzią,
		// żeby nie powstał na niej nowy uskok; pola się nie przesuwają (LAY-023).
		if l.lanes == nil {
			rowOf, indexOf := map[*item]int{}, map[*item]int{}
			for r, row := range rows {
				for k, it := range row {
					rowOf[it], indexOf[it] = r, k
				}
			}
			pinned := map[*item]bool{} // już wyrównane: nie ustępują miejsca
			moved := map[*item]bool{}  // przesunięte przez te kroki
			movable := func(it *item) bool {
				return !pinned[it] && (it.kind == kindDummy || it.kind == kindLabel) && len(it.up) == 1 && len(it.down) == 1
			}
			// try przesuwa pola do podanych położeń, rozpychając sąsiadów w ich warstwach; wszystkie
			// albo żadne. Zwraca, czy się udało.
			try := func(targets map[*item]float64) bool {
				moves := map[*item]float64{}
				for it, x := range targets {
					moves[it] = x
				}
				var row []*item
				push := func(k int, to float64) bool {
					p := row[k]
					if !movable(p) {
						return false
					}
					d := to - p.x
					moves[p] = to
					for q := p; len(q.up) == 1 && movable(q.up[0]); q = q.up[0] {
						moves[q.up[0]] = q.up[0].x + d
					}
					for q := p; len(q.down) == 1 && movable(q.down[0]); q = q.down[0] {
						moves[q.down[0]] = q.down[0].x + d
					}
					return true
				}
				at := func(it *item) float64 {
					if x, ok := moves[it]; ok {
						return x
					}
					return it.x
				}
				for v := range targets {
					i := indexOf[v]
					row = rows[rowOf[v]]
					for k := i - 1; k >= 0; k-- {
						if limit := at(row[k+1]) - sep(row[k], row[k+1]); at(row[k]) > limit+0.01 && !push(k, limit) {
							return false
						}
					}
					for k := i + 1; k < len(row); k++ {
						if limit := at(row[k-1]) + sep(row[k-1], row[k]); at(row[k]) < limit-0.01 && !push(k, limit) {
							return false
						}
					}
				}
				// Przesunięte krawędzie nie mogą wejść na sąsiadów w swoich warstwach.
				for it := range moves {
					r := rows[rowOf[it]]
					for k := 0; k+1 < len(r); k++ {
						if at(r[k+1])-at(r[k]) < sep(r[k], r[k+1])-0.01 {
							return false
						}
					}
				}
				for it, x := range moves {
					it.x = x
					moved[it] = true
				}
				for it := range targets {
					pinned[it] = true
				}
				return true
			}
			// Liczba krawędzi wychodzących z pola w źródle (bez pętli własnych): krawędź wsteczna
			// odwrócona w układzie nie jest wyjściem.
			outs := map[string]int{}
			for _, e := range l.g.Edges {
				if e.From != e.To {
					outs[e.From]++
				}
			}
			for _, row := range rows {
				for _, v := range row {
					if (v.kind != kindNode && v.kind != kindLabel) || len(v.up) != 1 {
						continue
					}
					u := v.up[0]
					if u.kind != kindNode || u.node == nil || len(u.down) < 3 || len(u.down)%2 == 0 || outs[u.node.ID] != len(u.down) {
						continue
					}
					kids := append([]*item(nil), u.down...)
					sort.SliceStable(kids, func(a, b int) bool { return kids[a].x < kids[b].x })
					if kids[len(kids)/2] != v {
						continue
					}
					target := u.nodeX()
					d := target - v.nodeX()
					if math.Abs(d) < 0.5 || math.Abs(d) > l.u(120) {
						continue
					}
					targets := map[*item]float64{v: v.x + d}
					if v.kind == kindLabel {
						// Podpis idzie pod pole tylko razem z polem, do którego prowadzi: inaczej uskok
						// przeniósłby się między podpis a pole.
						if len(v.down) != 1 || v.down[0].kind != kindNode || len(v.down[0].up) != 1 {
							continue
						}
						w := v.down[0]
						targets[w] = w.x + target - w.nodeX()
					}
					try(targets)
				}
			}
			// Długa krawędź z uskokiem między punktami pomocniczymi: odstający punkt wraca do osi, którą
			// ma większość punktów tej krawędzi, jeśli obok da się przesunąć podpisy i punkty innych krawędzi.
			seen := map[*item]bool{}
			for _, row := range rows {
				for _, it := range row {
					if it.kind != kindDummy || seen[it] || (len(it.up) == 1 && it.up[0].kind == kindDummy) {
						continue
					}
					var run []*item
					for q := it; q != nil && q.kind == kindDummy && !seen[q]; {
						run, seen[q] = append(run, q), true
						if len(q.down) != 1 {
							break
						}
						q = q.down[0]
					}
					count := map[float64]int{}
					axis, best := 0.0, 0
					for _, q := range run {
						count[q.x]++
						if count[q.x] > best {
							axis, best = q.x, count[q.x]
						}
					}
					if best < 2 || best == len(run) {
						continue
					}
					for _, q := range run {
						if q.x != axis && math.Abs(q.x-axis) <= l.u(60) {
							try(map[*item]float64{q: axis})
						}
					}
				}
			}
			// Pole, którego sąsiedzi w krawędziach to same punkty pomocnicze i podpisy, a któryś z nich
			// został przesunięty, dosuwa się w zwolnione miejsce do ich średniej, bez odsuwania innych.
			for _, row := range rows {
				for i, v := range row {
					if v.kind != kindNode || pinned[v] {
						continue
					}
					ns := append(append([]*item(nil), v.up...), v.down...)
					touched, straight, sum := false, false, 0.0
					for _, n := range ns {
						if n.real() {
							straight = true // pole sąsiaduje z polem: zostaje, gdzie je ustawił układ
						}
						touched = touched || moved[n]
						straight = straight || math.Abs(n.nodeX()-v.nodeX()) < 0.5
						sum += n.nodeX()
					}
					if !touched || straight || len(ns) == 0 {
						continue
					}
					want := v.x + sum/float64(len(ns)) - v.nodeX()
					if i > 0 {
						want = math.Max(want, row[i-1].x+sep(row[i-1], v))
					}
					if i+1 < len(row) {
						want = math.Min(want, row[i+1].x-sep(v, row[i+1]))
					}
					if math.Abs(want-v.x) >= 0.5 && (i == 0 || want >= row[i-1].x+sep(row[i-1], v)-0.01) &&
						(i+1 == len(row) || want <= row[i+1].x-sep(v, row[i+1])+0.01) {
						v.x = want
					}
				}
			}
			// Skręt długiej krawędzi tuż przed jej końcem przechodzi o warstwę wyżej, jeśli tam odstęp jest
			// pusty, a w odstępie przy końcu skręcają też inne krawędzie: linie się nie stłaczają.
			// jogsBelow liczy skręty w odstępie pod warstwą r, które w poprzek zachodzą na zakres lo–hi.
			jogsBelow := func(r int, skip *item, lo, hi float64) int {
				n := 0
				if r+1 >= len(rows) {
					return 0
				}
				for _, it := range rows[r+1] {
					for _, up := range it.up {
						a, b := math.Min(up.nodeX(), it.nodeX()), math.Max(up.nodeX(), it.nodeX())
						if up != skip && b-a > 0.5 && a < hi+l.u(10) && lo-l.u(10) < b {
							n++
						}
					}
				}
				return n
			}
			for _, row := range rows {
				for _, d := range row {
					if d.kind != kindDummy || len(d.down) != 1 || len(d.up) != 1 || d.down[0].kind == kindDummy {
						continue
					}
					end, r := d.down[0], rowOf[d]
					target := end.nodeX()
					lo, hi := math.Min(d.x, target), math.Max(d.x, target)
					// Koniec w zasięgu boku pola: krawędź i tak dochodzi prosto (straighten), nie ma skrętu.
					if end.kind == kindNode && math.Abs(d.x-target) <= l.portRange(end)/2 {
						continue
					}
					if math.Abs(d.x-target) < 0.5 || r == 0 || jogsBelow(r, d, lo, hi) == 0 || jogsBelow(r-1, nil, lo, hi) > 0 {
						continue
					}
					try(map[*item]float64{d: target})
				}
			}
		}

		// Notatki dosuwamy do pól, których dotyczą; mają pierwszeństwo przed resztą warstwy.
		for _, row := range rows {
			want := make([]float64, len(row))
			priority := make([]float64, len(row))
			any := false
			for i, it := range row {
				want[i], priority[i] = it.x, 0
				if it.anchor != nil {
					want[i] = it.anchor.x + it.anchorSide*sep(it.anchor, it)
					priority[i] = 1
					any = true
				}
			}
			if any {
				place(row, want, priority)
			}
		}

	}

	// Części niepołączone krawędziami są rozmieszczane osobno i ustawiane obok siebie; inaczej
	// odpychałyby się nawzajem i zostawiały puste miejsce.
	if l.lanes != nil {
		// Tory: cały poziom jest jedną całością, a położenia w poprzek wyznaczają pasy.
		arrange(lv.ranks)
	} else {
		for _, rows := range components(lv) {
			arrange(rows)
		}
	}
	// Części stoją w rzędach i kolumnach wyznaczonych przez shelve: kolumna ma szerokość
	// najszerszej swojej części, a każdy rząd zaczyna od początku osi kolejności.
	if l.lanes == nil {
		type cell struct{ shelf, column int }
		type span struct {
			rows   [][]*item
			lo, hi float64
			at     cell
		}
		var spans []span
		colWidth := map[cell]float64{}
		for _, rows := range components(lv) {
			sp := span{rows: rows, lo: math.Inf(1), hi: math.Inf(-1)}
			for _, row := range rows {
				for _, it := range row {
					sp.lo, sp.hi = math.Min(sp.lo, it.x-it.w/2), math.Max(sp.hi, it.x+it.w/2)
					sp.at = cell{it.shelf, it.column}
				}
			}
			if !lv.shelved {
				sp.at = cell{0, len(spans)} // bez rzędów każda część jest osobną kolumną
			}
			spans = append(spans, sp)
			colWidth[sp.at] = math.Max(colWidth[sp.at], sp.hi-sp.lo)
		}
		for _, sp := range spans {
			offset := 0.0
			for c := 0; c < sp.at.column; c++ {
				if w, ok := colWidth[cell{sp.at.shelf, c}]; ok {
					offset += w + l.u(48)
				}
			}
			for _, row := range sp.rows {
				for _, it := range row {
					it.x += offset - sp.lo
				}
			}
		}
		// LAY-026: część dosuwa się do poprzednich tak blisko, jak pozwalają pola tych warstw,
		// w których sama stoi; obrys poprzednich części w innych warstwach jej nie odpycha.
		// Krawędzie poprzednich części mają w tych warstwach przebiegi (pola), więc też trzymają
		// odstęp, a ich skręty między warstwami leżą w odstępach, w których część nie ma pól.
		if !lv.shelved {
			var placed []*item
			for k, sp := range spans {
				lo, hi := math.MaxInt, math.MinInt
				left := math.Inf(1)
				for _, row := range sp.rows {
					for _, it := range row {
						lo, hi = min(lo, it.rank), max(hi, it.rank)
						left = math.Min(left, it.x-it.w/2)
					}
				}
				if k > 0 {
					limit := 0.0
					for _, p := range placed {
						if p.rank >= lo && p.rank <= hi {
							limit = math.Max(limit, p.x+p.w/2+l.u(48))
						}
					}
					if shift := left - limit; shift > 0 {
						for _, row := range sp.rows {
							for _, it := range row {
								it.x -= shift
							}
						}
					}
				}
				for _, row := range sp.rows {
					placed = append(placed, row...)
				}
			}
		}
	}
	for _, row := range lv.ranks {
		sort.SliceStable(row, func(a, b int) bool { return row[a].x < row[b].x })
		for i, it := range row {
			it.order = i
		}
	}

	// Puste kolumny: pas w poprzek osi warstw, w którym w żadnej warstwie nic nie stoi, tylko
	// wydłuża krawędzie, które go przecinają. Wszystko za takim pasem dosuwa się do zwykłego odstępu.
	if l.lanes == nil {
		var all []*item
		for _, row := range lv.ranks {
			all = append(all, row...)
		}
		sort.SliceStable(all, func(a, b int) bool { return all[a].x-all[a].w/2 < all[b].x-all[b].w/2 })
		shift, reach, keep := 0.0, math.Inf(-1), l.u(48)
		for _, it := range all {
			lo, hi := it.x-it.w/2, it.x+it.w/2
			if !math.IsInf(reach, -1) && lo-reach > keep {
				shift += lo - reach - keep
			}
			reach = math.Max(reach, hi)
			it.x -= shift
		}
	}

	left := math.Inf(1)
	for _, row := range lv.ranks {
		for _, it := range row {
			left = math.Min(left, it.x-it.w/2)
		}
	}
	if math.IsInf(left, 1) || l.lanes != nil {
		left = 0
	}
	if l.lanes != nil {
		width = l.lanes[len(l.lanes)-1].hi
	}
	// Odstęp między warstwami pól mieści strzałkę; przy warstwie złożonej z samych etykiet i punktów
	// pośrednich jest krótszy, bo razem z nią i tak daje miejsce na krawędź.
	anyReal := func(row []*item) bool {
		for _, it := range row {
			if it.real() {
				return true
			}
		}
		return false
	}
	y := 0.0
	for r, row := range lv.ranks {
		band := 0.0
		turns := 0 // krawędzie, które w odstępie nad tą warstwą zmieniają położenie w poprzek
		for _, it := range row {
			band = math.Max(band, it.h)
			for _, up := range it.up {
				// Gałąź rombu daleko z boku wyjdzie z jego bocznego wierzchołka (sideExitsOf) i skręci
				// na wysokości rombu, nie w odstępie pod nim.
				side := up.kind == kindNode && len(up.down) >= 2 && (up.node.Shape == graph.Rhombus || up.node.Shape == graph.Choice) &&
					math.Abs(it.nodeX()-up.nodeX()) > up.nodeW()/2+l.u(16)
				if math.Abs(up.nodeX()-it.nodeX()) > 0.5 && !side {
					turns++
				}
			}
		}
		if r > 0 {
			// Warstwa samych punktów na ramce potrzebuje tylko miejsca na łuk krawędzi.
			gap := l.u(44)
			if portsOnly(row) || portsOnly(lv.ranks[r-1]) {
				gap = l.u(24)
			} else if !anyReal(row) || !anyReal(lv.ranks[r-1]) {
				gap = l.u(30)
			} else if turns == 0 && lv.sub != nil && l.g.Kind == "state" {
				// Wnętrze stanu złożonego: proste strzałki są krótsze niż na poziomie głównym, bo
				// ramka i tak zajmuje miejsce, a jej zawartość czyta się jako jedną całość.
				gap = l.u(26)
			}
			// Krawędzie skręcające tuż nad polem: tory odcinków poprzecznych i pod najniższym kawałek prostej
			// przed grotem (zapas toru przy grocie w assignTracks).
			// Najniższy tor zostawia przed grotem kawałek prostej (LAY-024); kolejne tory mają odstęp jak
			// dotąd, a rozsuwają się tylko w miejscu, które już jest (assignTracks). Encje nie mają grotów.
			if !l.o.CurvedEdges && turns > 0 && anyReal(row) && l.g.Kind != "er" {
				gap = math.Max(gap, l.u(34)+float64(turns-1)*l.u(14))
			}
			// Krawędzie łamane: każdy odcinek poprzeczny potrzebuje własnego toru.
			if !l.o.CurvedEdges && turns > 2 {
				gap = math.Max(gap, math.Min(l.u(30)+float64(turns)*l.u(10), l.u(160)))
			}
			y += gap
		}
		for _, it := range row {
			it.x = math.Round(it.x - left)
			it.y = y + band/2
			if it.kind == kindCluster && it.child.sub.Region {
				it.y = y + it.h/2 // regiony w jednym rzędzie zaczynają na tej samej wysokości
			}
			it.band = band
			width = math.Max(width, it.x+it.w/2)
		}
		y += band
	}
	return width, y
}

// components dzieli pola poziomu na części spójne (połączone krawędziami albo notatką ze stanem)
// i zwraca dla każdej jej warstwy, z zachowaniem kolejności pól; części są w kolejności źródła.
func components(lv *level) [][][]*item {
	parent := map[*item]*item{}
	var find func(it *item) *item
	find = func(it *item) *item {
		if p, ok := parent[it]; ok && p != it {
			root := find(p)
			parent[it] = root
			return root
		}
		parent[it] = it
		return it
	}
	for _, row := range lv.ranks {
		for _, it := range row {
			for _, d := range it.down {
				parent[find(d)] = find(it)
			}
			if it.anchor != nil {
				parent[find(it)] = find(it.anchor)
			}
		}
	}
	index := map[*item]int{}
	var out [][][]*item
	place := func(it *item) {
		root := find(it)
		k, ok := index[root]
		if !ok {
			k = len(out)
			index[root] = k
			out = append(out, make([][]*item, len(lv.ranks)))
		}
		_ = k
	}
	// Kolejność części według pierwszego pola w kolejności źródła, potem pozostałych (punkty pomocnicze).
	for _, it := range lv.items {
		place(it)
	}
	for r, row := range lv.ranks {
		for _, it := range row {
			place(it)
			k := index[find(it)]
			out[k][r] = append(out[k][r], it)
		}
	}
	// Części z punktem na ramce stoją w kolejności tych punktów (ustalonej z drugimi końcami krawędzi
	// na poziomie wyżej); zajmują swoje dotychczasowe miejsca, a pozostałe części zostają, gdzie były.
	key := func(part [][]*item) (int, bool) {
		for _, row := range part {
			for _, it := range row {
				if it.kind == kindPort {
					return it.order, true
				}
			}
		}
		return 0, false
	}
	var slots []int
	var withPorts [][][]*item
	for i, part := range out {
		if _, ok := key(part); ok {
			slots = append(slots, i)
			withPorts = append(withPorts, part)
		}
	}
	sort.SliceStable(withPorts, func(a, b int) bool {
		ka, _ := key(withPorts[a])
		kb, _ := key(withPorts[b])
		return ka < kb
	})
	for i, slot := range slots {
		out[slot] = withPorts[i]
	}
	return out
}

func portsOnly(row []*item) bool {
	for _, it := range row {
		if it.kind != kindPort {
			return false
		}
	}
	return len(row) > 0
}

// fanOffset zwraca przesunięcie pola względem wspólnego sąsiada: pola połączone z tym samym
// sąsiadem mają się rozłożyć symetrycznie wokół niego, a nie ustawić w jednym miejscu.
func fanOffset(it *item, siblings []*item, sep func(a, b *item) float64) float64 {
	if len(siblings) < 2 {
		return 0
	}
	group := make([]*item, 0, len(siblings))
	seen := map[*item]bool{}
	for _, s := range siblings {
		if !seen[s] {
			seen[s] = true
			group = append(group, s)
		}
	}
	sort.SliceStable(group, func(a, b int) bool { return group[a].order < group[b].order })
	pos, at := 0.0, 0.0
	for i, s := range group {
		if i > 0 {
			pos += sep(group[i-1], s)
		}
		if s == it {
			at = pos
		}
	}
	return at - pos/2
}

// place przelicza położenia pól na osie umowne całego diagramu.
func (l *layouter) place(lv *level, ox, oy float64) {
	defer l.enter(lv)()
	lv.offsetX, lv.offsetY = ox, oy
	for _, row := range lv.ranks {
		for _, it := range row {
			it.x += ox + lv.padX0
			it.y += oy + lv.padY0
			if it.kind != kindCluster {
				continue
			}
			c := it.child
			if !c.turned {
				c.ox, c.oy = lv.ox, lv.oy
				l.place(c, it.x-it.w/2, it.y-it.h/2)
				continue
			}
			// Podgraf z własnym kierunkiem: jego osie zaczepiamy tak, żeby rzeczywisty
			// prostokąt wnętrza pokrył się z polem podgrafu na tym poziomie.
			cx, cy, rw, rh := l.rect(it.x, it.y, it.w, it.h)
			c.ox, c.oy = 0, 0
			restore := l.enter(c)
			p0, p1 := l.point(0, 0), l.point(c.w, c.h)
			restore()
			c.ox, c.oy = cx-rw/2-math.Min(p0.X, p1.X), cy-rh/2-math.Min(p0.Y, p1.Y)
			l.place(c, 0, 0)
		}
	}
}

// frameLink to krawędź przechodząca przez ramkę podgrafu na danym poziomie: punkt na ramce
// (przesunięcie off względem środka ramki) i drugi koniec po tej stronie ramki.
type frameLink struct {
	cluster, other *item
	off            float64
	otherPort      *item // punkt na ramce drugiego końca, gdy ten też jest podgrafem
}

func frameLinks(lv *level) []frameLink {
	var out []frameLink
	for _, e := range lv.edges {
		if e.upperCont != nil {
			link := frameLink{cluster: e.u, other: e.v, off: portOffset(e.u, e.upperCont.v)}
			if len(e.chain) > 0 {
				link.other = e.chain[0]
			} else if e.lowerCont != nil {
				link.otherPort = e.lowerCont.u
			}
			out = append(out, link)
		}
		if e.lowerCont != nil {
			link := frameLink{cluster: e.v, other: e.u, off: portOffset(e.v, e.lowerCont.u)}
			if len(e.chain) > 0 {
				link.other = e.chain[len(e.chain)-1]
			} else if e.upperCont != nil {
				link.otherPort = e.upperCont.v
			}
			out = append(out, link)
		}
	}
	return out
}

// settleFrames przesuwa ramki podgrafów tak, żeby krawędzie przechodzące przez ramkę nie miały
// drobnych uskoków; przy remisie wygrywa położenie z większą liczbą krawędzi biegnących prosto. Uskok krawędzi jest dobry, gdy jest zerem, gdy pole po drugiej stronie wyprostuje
// go, przesuwając swój punkt zaczepienia w zasięgu boku, albo gdy jest wyraźnym skrętem; drobny jest
// każdy inny. Ramka staje w położeniu, w którym drobnych uskoków jest najmniej: obecnym albo takim,
// w którym jedna z krawędzi biegnie dokładnie prosto; przy remisie bliżej obecnego.
func (l *layouter) settleFrames(lv *level, rows [][]*item, place func(row []*item, want, priority []float64)) {
	links := frameLinks(lv)
	if len(links) == 0 {
		return
	}
	clear := l.u(16)
	target := func(k frameLink) (x, tolerance float64) {
		switch {
		case k.otherPort != nil:
			return k.other.x + portOffset(k.other, k.otherPort), 0
		case k.other.kind == kindNode:
			return k.other.nodeX(), math.Max(l.portRange(k.other)/2, 0)
		}
		return k.other.x, 0
	}
	for _, row := range rows {
		for i, c := range row {
			if c.kind != kindCluster {
				continue
			}
			var mine []frameLink
			for _, k := range links {
				if k.cluster == c {
					mine = append(mine, k)
				}
			}
			if len(mine) == 0 {
				continue
			}
			// score: najpierw liczba drobnych uskoków, potem liczba krawędzi, które nie biegną prosto.
			score := func(shift float64) (small, bent int) {
				for _, k := range mine {
					x, tol := target(k)
					d := math.Abs(x - (c.x + shift + k.off))
					if d > 0.5 && d > tol+0.5 {
						bent++
						if d < clear {
							small++
						}
					}
				}
				return small, bent
			}
			best := 0.0
			bestSmall, bestBent := score(0)
			if bestSmall == 0 {
				continue
			}
			for _, k := range mine {
				x, _ := target(k)
				shift := x - (c.x + k.off)
				sm, be := score(shift)
				if sm < bestSmall || (sm == bestSmall && (be < bestBent || (be == bestBent && math.Abs(shift) < math.Abs(best)))) {
					best, bestSmall, bestBent = shift, sm, be
				}
			}
			if best == 0 {
				continue
			}
			want := make([]float64, len(row))
			priority := make([]float64, len(row))
			for j, it := range row {
				want[j], priority[j] = it.x, 100
			}
			want[i], priority[i] = c.x+best, 3000
			place(row, want, priority)
		}
	}
}
