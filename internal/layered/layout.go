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
}

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
	band      float64 // wysokość warstwy, w której leży pole

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

	lv *level
	// Identyfikatory pól ze źródła na górnym i dolnym końcu (po odwróceniu); pusty dla punktu na ramce.
	topID, bottomID string
	// Gdy koniec leży głębiej, w podgrafie, dalszy odcinek krawędzi jest prowadzony w układzie
	// tego podgrafu, między polem a punktem na jego ramce.
	upperCont, lowerCont *ledge
	part                 bool // odcinek krawędzi z poziomu wyżej; nie jest rysowany osobno
}

// level to układ jednego zakresu: całego diagramu albo wnętrza podgrafu.
type level struct {
	sub    *graph.Subgraph
	items  []*item
	edges  []*ledge
	loops  map[*item][]*graph.Edge
	ranks  [][]*item
	factor int

	w, h             float64 // rozmiar z marginesami
	padX0, padY0     float64
	offsetX, offsetY float64 // położenie początku poziomu w osiach umownych diagramu
	titleLines       []string

	// Układ osi poziomu. Podgraf z własnym kierunkiem (turned) ma własne osie umowne, zaczepione
	// w punkcie (ox, oy) rysunku; pozostałe poziomy dziedziczą osie rodzica.
	pinPorts               bool // porządkowanie nie rusza warstw złożonych z samych punktów na ramce
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
	g  *graph.Graph
	o  Options
	sc *scene.Scene

	fs, lh   float64
	vertical bool    // TB albo BT: oś warstw jest pionowa
	flip     bool    // BT albo RL: oś warstw odwrócona
	ox, oy   float64 // zaczepienie bieżących osi umownych na rysunku

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

// wrap łamie linie szersze niż limit na granicach wyrazów.
func (l *layouter) wrap(lines []string, limit float64) []string {
	var out []string
	for _, line := range lines {
		line = graph.PerWord(line)
		if l.width(line) <= limit {
			out = append(out, line)
			continue
		}
		current := ""
		for _, word := range strings.Fields(line) {
			candidate := word
			if current != "" {
				candidate = current + " " + word
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
	l := &layouter{g: g, o: o, fs: o.FontSize, levels: map[string]*level{}, itemOf: map[string]*item{}}
	l.lh = math.Round(l.fs * 1.25)
	l.vertical = g.Direction == graph.TopBottom || g.Direction == graph.BottomTop
	l.flip = g.Direction == graph.BottomTop || g.Direction == graph.RightLeft
	l.sc = &scene.Scene{Title: g.AccTitle, Desc: g.AccDescr, FontFamily: o.FontFamily, FontSize: l.fs, Margin: l.u(12), Background: o.Background}

	l.build()
	l.orient()
	l.measure()
	l.assignEdges()
	l.rankAll(l.levels[""])
	l.sizeAll(l.levels[""])
	l.place(l.levels[""], 0, 0)
	l.draw()
	return l.sc
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
		entries = append(entries, entry{s.Line, i, it, s.Parent})
	}
	for i, n := range l.g.Nodes {
		it := &item{kind: kindNode, node: n}
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
			} else if it.kind == kindNode {
				it.w, it.h = l.abstract(l.nodeSize(it.node.Shape, it.text))
			}
		}
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
	// Regiony współbieżne stoją jeden za drugim wzdłuż osi warstw: wymusza to niewidoczna
	// krawędź między kolejnymi regionami tego samego stanu.
	last := map[string]string{}
	for _, s := range l.g.Subgraphs {
		if !s.Region {
			continue
		}
		if prev := last[s.Parent]; prev != "" {
			add(&graph.Edge{From: prev, To: s.ID, Style: graph.Invisible, MinLen: 1})
		}
		last[s.Parent] = s.ID
	}
	for _, n := range l.g.Notes {
		id := "[note]" + n.Target + "#" + n.Side
		for l.itemOf[id] != nil {
			id += "'"
		}
		scope := l.scopeOf(n.Target)
		it := &item{kind: kindNode, node: &graph.Node{ID: id, Shape: "note", Parent: scope}}
		it.text = l.wrap(n.Text, l.u(200))
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
				w, h := l.abstract(l.textSize(l.wrap(e.Label, l.u(200))))
				_ = h
				extra += w + l.u(8)
			}
			if extra > it.loopExtra {
				it.w += extra - it.loopExtra
				it.loopExtra = extra
			}
		}
	}
	l.order(lv)
	l.finish(lv)

	// Drugie podejście: skoro wiadomo już, gdzie na tym poziomie leżą drugie końce krawędzi
	// wchodzących do podgrafów, punkty na ich ramkach ustawiamy w tej kolejności i wnętrza
	// rozmieszczamy od nowa.
	changed := false
	for _, it := range lv.items {
		if it.kind == kindCluster && l.alignPorts(lv, it) {
			l.finish(it.child)
			it.w, it.h = it.child.w, it.child.h
			changed = true
		}
	}
	if changed {
		l.finish(lv)
	}
}

// alignPorts ustawia punkty na ramce podgrafu w kolejności położenia drugich końców ich krawędzi
// na poziomie lv i porządkuje wnętrze na nowo. Zmiana zostaje, jeśli nie dodaje przecięć we wnętrzu.
func (l *layouter) alignPorts(lv *level, cluster *item) bool {
	child := cluster.child
	hint := map[*item]float64{}
	for _, e := range lv.edges {
		if e.upperCont != nil && e.u == cluster {
			n := e.v
			if len(e.chain) > 0 {
				n = e.chain[0]
			}
			hint[e.upperCont.v] = n.x
		}
		if e.lowerCont != nil && e.v == cluster {
			n := e.u
			if len(e.chain) > 0 {
				n = e.chain[len(e.chain)-1]
			}
			hint[e.lowerCont.u] = n.x
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
		sort.SliceStable(sorted, func(a, b int) bool { return hint[sorted[a]] < hint[sorted[b]] })
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
		switch {
		case l.vertical && !l.flip:
			padY0 += top
		case l.vertical:
			padY1 += top
		default:
			padX0 += top
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
	lv.padX0, lv.padY0 = padX0, padY0
	lv.w, lv.h = contentW+padX0+padX1, contentH+padY0+padY1
}

// rank usuwa cykle, dzieli pola na warstwy i wstawia punkty pomocnicze na długich krawędziach.
func (l *layouter) rank(lv *level) {
	defer l.enter(lv)()
	lv.factor = 1
	pairs := map[[2]*item]bool{}
	for _, e := range lv.edges {
		if len(e.e.Label) > 0 {
			lv.factor = 2 // warstwy pośrednie dają miejsce na etykiety krawędzi
		}
		// Krawędzie wielokrotne też potrzebują warstwy pośredniej: każda dostaje w niej własny punkt.
		key := [2]*item{e.u, e.v}
		if e.u == e.v {
			continue
		}
		if pairs[key] || pairs[[2]*item{e.v, e.u}] {
			lv.factor = 2
		}
		pairs[key] = true
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

	// Warstwy: najdłuższa ścieżka od źródeł, z minimalną długością krawędzi.
	minLen := func(e *ledge) int { return e.e.MinLen * lv.factor }
	indeg := make([]int, len(lv.items))
	succ := make([][]*ledge, len(lv.items))
	for _, e := range lv.edges {
		indeg[index[e.v]]++
		succ[index[e.u]] = append(succ[index[e.u]], e)
	}
	rank := make([]int, len(lv.items))
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
		span := e.v.rank - e.u.rank
		labelRank := -1
		if len(e.e.Label) > 0 {
			labelRank = e.u.rank + span/2
		}
		prev := e.u
		for r := e.u.rank + 1; r < e.v.rank; r++ {
			d := &item{kind: kindDummy, rank: r, w: l.u(8)}
			if r == labelRank {
				d.kind = kindLabel
				d.text = l.wrap(e.e.Label, l.u(200))
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

// attachments zwraca, dla pary (pole, podgraf), przesunięcie względem środka podgrafu punktu na
// jego ramce, przez który przechodzi krawędź łącząca tę parę. Dzięki temu sąsiad podgrafu
// ustawia się pod punktem wyjścia krawędzi, a nie pod środkiem ramki.
func (l *layouter) attachments(lv *level) map[[2]*item]float64 {
	sum, count := map[[2]*item]float64{}, map[[2]*item]float64{}
	add := func(other, cluster, port *item) {
		c := cluster.child
		k := [2]*item{other, cluster}
		sum[k] += c.padX0 + port.x - c.w/2
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
	for iter := 0; iter < 24 && bestCount > 0; iter++ {
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
	gap := func(a, b *item) float64 {
		switch {
		case a.real() && b.real():
			return l.u(48)
		case a.kind == kindDummy && b.kind == kindDummy:
			return l.u(10)
		}
		return l.u(18)
	}
	sep := func(a, b *item) float64 { return a.w/2 + b.w/2 + gap(a, b) }
	attach := l.attachments(lv)
	// arrange rozmieszcza wzdłuż osi kolejności jedną spójną część poziomu.
	arrange := func(rows [][]*item) {
		for _, row := range rows {
			for i, it := range row {
				if i == 0 {
					it.x = it.w / 2
				} else {
					it.x = row[i-1].x + sep(row[i-1], it)
				}
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
							sum += w * (n.x + off) // pod punktem na ramce podgrafu
						} else if off, ok := attach[[2]*item{n, it}]; ok {
							sum += w * (n.x - off)
						} else {
							sum += w * (n.x + fanOffset(it, siblings(n), sep))
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
					want[i] = sum / total
					priority[i] = float64(count)
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
	for _, rows := range components(lv) {
		arrange(rows)
	}
	shift := 0.0
	for k, rows := range components(lv) {
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, row := range rows {
			for _, it := range row {
				lo, hi = math.Min(lo, it.x-it.w/2), math.Max(hi, it.x+it.w/2)
			}
		}
		if k > 0 {
			shift += l.u(48)
		}
		for _, row := range rows {
			for _, it := range row {
				it.x += shift - lo
			}
		}
		shift += hi - lo
	}
	for _, row := range lv.ranks {
		sort.SliceStable(row, func(a, b int) bool { return row[a].x < row[b].x })
		for i, it := range row {
			it.order = i
		}
	}

	left := math.Inf(1)
	for _, row := range lv.ranks {
		for _, it := range row {
			left = math.Min(left, it.x-it.w/2)
		}
	}
	if math.IsInf(left, 1) {
		left = 0
	}
	// Bez warstw pośrednich odstęp mieści samą strzałkę; z nimi (etykiety, krawędzie wielokrotne)
	// każda połowa odstępu jest krótsza, a razem z warstwą pośrednią dają miejsce na etykietę.
	rankGap := l.u(44)
	if lv.factor > 1 {
		rankGap = l.u(32)
	}
	y := 0.0
	for r, row := range lv.ranks {
		band := 0.0
		turns := 0 // krawędzie, które w odstępie nad tą warstwą zmieniają położenie w poprzek
		for _, it := range row {
			band = math.Max(band, it.h)
			for _, up := range it.up {
				if math.Abs(up.x-it.x) > 0.5 {
					turns++
				}
			}
		}
		if r > 0 {
			// Warstwa samych punktów na ramce potrzebuje tylko miejsca na łuk krawędzi.
			gap := rankGap
			if portsOnly(row) || portsOnly(lv.ranks[r-1]) {
				gap = l.u(24)
			}
			// Krawędzie łamane: każdy odcinek poprzeczny potrzebuje własnego toru.
			if !l.o.CurvedEdges && turns > 2 {
				gap = math.Max(gap, math.Min(l.u(30)+float64(turns)*l.u(7), l.u(150)))
			}
			y += gap
		}
		for _, it := range row {
			it.x = math.Round(it.x - left)
			it.y = y + band/2
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
