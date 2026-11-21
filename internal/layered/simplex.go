package layered

// networkSimplex poprawia dopuszczalny podział na warstwy tak, żeby suma długości krawędzi była
// najmniejsza (Gansner, Koutsofios, North, Vo: „A Technique for Drawing Directed Graphs”, 1993,
// rozdział 2). Węzły mają numery od 0 do n−1, krawędź to trójka: początek, koniec, najmniejsza
// długość. rank na wejściu spełnia rank[koniec] − rank[początek] ≥ długość dla każdej krawędzi
// i jest poprawiany w miejscu; każda spójna część grafu kończy z najmniejszą warstwą równą zero.
//
// Wartości przecięć są liczone od nowa po każdej wymianie krawędzi drzewa. Jest to wolniejsze niż
// przyrostowe liczenie z publikacji, ale proste, a grafy mają najwyżej kilkaset węzłów; liczba
// wymian jest ograniczona, więc w najgorszym razie wynik jest dopuszczalny, choć nie najlepszy.
func networkSimplex(n int, edges [][3]int, rank []int) {
	adj := make([][]int, n)
	for k, e := range edges {
		adj[e[0]] = append(adj[e[0]], k)
		adj[e[1]] = append(adj[e[1]], k)
	}
	slack := func(k int) int { return rank[edges[k][1]] - rank[edges[k][0]] - edges[k][2] }
	other := func(k, v int) int {
		if edges[k][0] == v {
			return edges[k][1]
		}
		return edges[k][0]
	}

	comp := make([]int, n)
	for i := range comp {
		comp[i] = -1
	}
	for start := 0; start < n; start++ {
		if comp[start] >= 0 {
			continue
		}
		// Spójna część grafu zawierająca węzeł start.
		nodes := []int{start}
		comp[start] = start
		for i := 0; i < len(nodes); i++ {
			for _, k := range adj[nodes[i]] {
				if w := other(k, nodes[i]); comp[w] < 0 {
					comp[w] = start
					nodes = append(nodes, w)
				}
			}
		}
		if len(nodes) > 1 {
			simplexComponent(nodes, edges, adj, rank, slack, other)
		}
		low := rank[nodes[0]]
		for _, v := range nodes {
			low = min(low, rank[v])
		}
		for _, v := range nodes {
			rank[v] -= low
		}
	}
}

func simplexComponent(nodes []int, edges [][3]int, adj [][]int, rank []int, slack func(int) int, other func(int, int) int) {
	inTree := map[int]bool{nodes[0]: true}
	treeEdge := map[int]bool{}
	tree := []int{nodes[0]}

	// Drzewo rozpinające z krawędzi napiętych (o zerowym luzie); gdy takie krawędzie się kończą,
	// całe drzewo przesuwa się o najmniejszy luz krawędzi łączącej je z resztą.
	for len(tree) < len(nodes) {
		for i := 0; i < len(tree); i++ {
			for _, k := range adj[tree[i]] {
				if w := other(k, tree[i]); !inTree[w] && slack(k) == 0 {
					inTree[w], treeEdge[k] = true, true
					tree = append(tree, w)
				}
			}
		}
		if len(tree) == len(nodes) {
			break
		}
		best := -1
		for _, v := range tree {
			for _, k := range adj[v] {
				if !inTree[other(k, v)] && (best < 0 || slack(k) < slack(best) || (slack(k) == slack(best) && k < best)) {
					best = k
				}
			}
		}
		delta := slack(best)
		if inTree[edges[best][1]] {
			delta = -delta // drzewo zawiera koniec krawędzi: przesuwa się w stronę jej początku
		}
		for _, v := range tree {
			rank[v] += delta
		}
	}

	// side zaznacza węzły po stronie początku krawędzi drzewa k, po jej usunięciu z drzewa.
	side := func(k int) map[int]bool {
		tail := map[int]bool{edges[k][0]: true}
		queue := []int{edges[k][0]}
		for i := 0; i < len(queue); i++ {
			for _, t := range adj[queue[i]] {
				if t == k || !treeEdge[t] {
					continue
				}
				if w := other(t, queue[i]); !tail[w] {
					tail[w] = true
					queue = append(queue, w)
				}
			}
		}
		return tail
	}
	var keys []int
	seen := map[int]bool{}
	for _, v := range nodes {
		for _, k := range adj[v] {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sortInts(keys)

	for iter := 0; iter < 4*len(nodes)+20; iter++ {
		leave, tail := -1, map[int]bool(nil)
		for _, k := range keys {
			if !treeEdge[k] {
				continue
			}
			t := side(k)
			cut := 0
			for _, f := range keys {
				a, b := t[edges[f][0]], t[edges[f][1]]
				if a && !b {
					cut++
				} else if !a && b {
					cut--
				}
			}
			if cut < 0 {
				leave, tail = k, t
				break
			}
		}
		if leave < 0 {
			return
		}
		// Wchodzi krawędź o najmniejszym luzie biegnąca od strony końca do strony początku.
		enter := -1
		for _, f := range keys {
			if treeEdge[f] || tail[edges[f][0]] || !tail[edges[f][1]] {
				continue
			}
			if enter < 0 || slack(f) < slack(enter) {
				enter = f
			}
		}
		if enter < 0 {
			return
		}
		// Strona początku przesuwa się o luz krawędzi wchodzącej: ta staje się napięta, a krawędzie
		// drzewa po obu stronach zostają napięte.
		delta := slack(enter)
		for v := range tail {
			rank[v] -= delta
		}
		delete(treeEdge, leave)
		treeEdge[enter] = true
	}
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
