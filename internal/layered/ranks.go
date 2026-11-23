package layered

import "github.com/polestorm/gomermaidgo/internal/graph"

// LAY-025: rozkładanie długiej warstwy liści (opcja fold-ranks).
//
// Węzeł z wieloma liśćmi (następnikami bez dalszych połączeń) stawia je w jednej warstwie, która
// w kierunku LR robi się bardzo wysoka, a w TB bardzo szeroka. Rozkładamy taką warstwę na dwie
// podwarstwy ustawione zygzakiem: krawędź do co drugiego liścia jest o warstwę dłuższa. Trzy
// podwarstwy dawały uskoki (przebiegi dwóch krawędzi między parą bliższych liści nie trafiały
// w środki dalszych), więc ich nie próbujemy. Przebiegi
// dłuższych krawędzi stoją w bliższej podwarstwie między liśćmi, więc linie do dalszych liści
// przechodzą przez szczeliny i niczego nie przecinają, a kolejność liści zostaje ze źródła.
// To samo dotyczy wielu korzeni (poprzedników bez innych połączeń) wchodzących do jednego węzła.

// minFan to najmniejsza liczba liści, od której warstwę warto rozkładać.
const minFan = 6

// fans zwraca grupy krawędzi (indeksy w g.Edges) łączących węzeł z jego liśćmi albo korzeniami,
// w kolejności źródła.
func fans(g *graph.Graph) [][]int {
	parent := map[string]string{}
	for _, n := range g.Nodes {
		parent[n.ID] = n.Parent
	}
	in, out := map[string]int{}, map[string]int{}
	for _, e := range g.Edges {
		out[e.From]++
		in[e.To]++
	}
	isNode := func(id string) bool { _, ok := parent[id]; return ok }
	leaves := map[string][]int{}
	roots := map[string][]int{}
	var hubs []string
	seen := map[string]bool{}
	for i, e := range g.Edges {
		if e.From == e.To || !isNode(e.From) || !isNode(e.To) || parent[e.From] != parent[e.To] {
			continue
		}
		if in[e.To] == 1 && out[e.To] == 0 {
			leaves[e.From] = append(leaves[e.From], i)
			if !seen[e.From] {
				seen[e.From] = true
				hubs = append(hubs, e.From)
			}
		}
		if out[e.From] == 1 && in[e.From] == 0 && !(in[e.To] == 1 && out[e.To] == 0) {
			roots["\x00"+e.To] = append(roots["\x00"+e.To], i)
			if !seen["\x00"+e.To] {
				seen["\x00"+e.To] = true
				hubs = append(hubs, "\x00"+e.To)
			}
		}
	}
	var groups [][]int
	for _, h := range hubs {
		gr := leaves[h]
		if h[0] == 0 {
			gr = roots[h]
		}
		authored := false // autor sam nadał długości krawędziom: jego układ zostaje
		for _, i := range gr {
			authored = authored || g.Edges[i].MinLen > 1
		}
		if len(gr) >= minFan && !authored {
			groups = append(groups, gr)
		}
	}
	return groups
}

// withLengths zwraca kopię grafu, w której krawędzie grupy dostają na przemian 0, 1, … k−1
// dodatkowych warstw (oznaczone jako Spread, patrz odstępy w position).
func withLengths(g *graph.Graph, group []int, k int) *graph.Graph {
	c := *g
	c.Edges = make([]*graph.Edge, len(g.Edges))
	copy(c.Edges, g.Edges)
	for j, i := range group {
		if extra := j % k; extra > 0 {
			e := *c.Edges[i]
			e.MinLen = max(e.MinLen, 1) + extra
			e.Spread = true
			c.Edges[i] = &e
		}
	}
	return &c
}

// spreadFans rozkłada długie warstwy liści, gdy rysunek mieści się przez to na A4 w wyraźnie
// większej skali (jak foldChains: co najmniej 20%).
func spreadFans(g *graph.Graph, o Options) *graph.Graph {
	if g.Layout != "" || (g.Kind != "flowchart" && g.Kind != "state") {
		return g
	}
	groups := fans(g)
	if len(groups) == 0 {
		return g
	}
	o.Fit, o.Fold, o.FoldRanks = FitOff, FoldOff, false
	current := g
	scale := pageScale(layout(g, o, 0, 1), o.FontSize)
	for _, group := range groups {
		candidate := withLengths(current, group, 2)
		if s := pageScale(layout(candidate, o, 0, 1), o.FontSize); s >= scale*1.2-1e-9 {
			current, scale = candidate, s
		}
	}
	return current
}
