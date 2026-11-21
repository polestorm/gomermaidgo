package layered

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/polestorm/gomermaidgo/internal/flowchart"
	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/graph"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/state"
)

func parse(t *testing.T, path string) *graph.Graph {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var g *graph.Graph
	if strings.Contains(path, "/flowchart/") {
		g, err = flowchart.Parse(string(src))
	} else {
		g, err = state.Parse(string(src))
	}
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return g
}

// corpusScenes rozmieszcza każdy plik korpusu w czterech kierunkach.
func corpusScenes(t *testing.T) map[string]*scene.Scene {
	t.Helper()
	out := map[string]*scene.Scene{}
	for _, dir := range []string{"flowchart", "state"} {
		files, _ := filepath.Glob("../../testdata/corpus/" + dir + "/*.mmd")
		if len(files) == 0 {
			t.Fatalf("no corpus files for %s", dir)
		}
		for _, f := range files {
			for _, d := range []graph.Direction{graph.TopBottom, graph.BottomTop, graph.LeftRight, graph.RightLeft} {
				g := parse(t, f)
				g.Direction = d
				out[dir+"/"+filepath.Base(f)+" "+string(d)] = Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
			}
		}
	}
	return out
}

func overlap(a, b scene.Box) bool {
	const eps = 0.5
	return a.X0 < b.X1-eps && b.X0 < a.X1-eps && a.Y0 < b.Y1-eps && b.Y0 < a.Y1-eps
}

func inside(a, b scene.Box) bool {
	const eps = 0.5
	return a.X0 >= b.X0-eps && a.Y0 >= b.Y0-eps && a.X1 <= b.X1+eps && a.Y1 <= b.Y1+eps
}

func byRole(sc *scene.Scene, roles ...string) []scene.Item {
	var out []scene.Item
	for _, it := range sc.Items {
		for _, r := range roles {
			if it.Role == r {
				out = append(out, it)
			}
		}
	}
	return out
}

// REQ: FLOW-013 STATE-011
// Kształty węzłów i notatek nie nachodzą na siebie w żadnym kierunku układu.
func TestNodesDoNotOverlap(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		nodes := byRole(sc, "node", "note")
		for i := range nodes {
			for j := i + 1; j < len(nodes); j++ {
				if overlap(nodes[i].Bounds(), nodes[j].Bounds()) {
					t.Errorf("%s: two node shapes overlap: %+v and %+v", name, nodes[i].Bounds(), nodes[j].Bounds())
				}
			}
		}
	}
}

// REQ: FLOW-013 STATE-011
// Ramki podgrafów są rozłączne albo zagnieżdżone, a każdy węzeł leży w ramce w całości albo poza nią.
func TestClustersAreNestedOrDisjoint(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		clusters := byRole(sc, "cluster")
		for i := range clusters {
			a := clusters[i].Bounds()
			for j := i + 1; j < len(clusters); j++ {
				b := clusters[j].Bounds()
				if overlap(a, b) && !inside(a, b) && !inside(b, a) {
					t.Errorf("%s: cluster frames intersect", name)
				}
			}
			for _, n := range byRole(sc, "node", "note") {
				if b := n.Bounds(); overlap(a, b) && !inside(b, a) {
					t.Errorf("%s: a node crosses a cluster frame", name)
				}
			}
			for _, title := range byRole(sc, "cluster-title") {
				if b := title.Bounds(); overlap(a, b) && !inside(b, a) {
					t.Errorf("%s: cluster title %q crosses a cluster frame", name, title.Text)
				}
			}
		}
	}
}

// REQ: FLOW-013 STATE-011
// Etykiety krawędzi nie nachodzą na węzły ani na siebie, a etykieta węzła mieści się w polu węzła.
func TestLabelsDoNotCollide(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		labels := byRole(sc, "edge-label-background")
		nodes := byRole(sc, "node", "note")
		for i, lb := range labels {
			for _, n := range nodes {
				if overlap(lb.Bounds(), n.Bounds()) {
					t.Errorf("%s: an edge label overlaps a node", name)
				}
			}
			for j := i + 1; j < len(labels); j++ {
				if overlap(lb.Bounds(), labels[j].Bounds()) {
					t.Errorf("%s: two edge labels overlap", name)
				}
			}
		}
		for _, text := range byRole(sc, "node-label", "note-label") {
			held := false
			for _, n := range nodes {
				if inside(text.Bounds(), n.Bounds()) {
					held = true
					break
				}
			}
			if !held {
				t.Errorf("%s: node label %q is not inside any node shape", name, text.Text)
			}
		}
	}
}

// REQ: FLOW-013
// Każda krawędź ze źródła, poza niewidocznymi, ma dokładnie jedną ścieżkę na rysunku.
func TestEveryEdgeIsDrawn(t *testing.T) {
	files, _ := filepath.Glob("../../testdata/corpus/flowchart/*.mmd")
	for _, f := range files {
		g := parse(t, f)
		want := 0
		for _, e := range g.Edges {
			if e.Style != graph.Invisible {
				want++
			}
		}
		sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
		if got := len(byRole(sc, "edge")); got != want {
			t.Errorf("%s: %d edge paths, want %d", f, got, want)
		}
	}
}

// REQ: FLOW-013
// Pętla własna, krawędzie wielokrotne i cykl nie psują układu.
func TestLoopsAndCycles(t *testing.T) {
	g, err := flowchart.Parse("flowchart TD\nA --> A\nA -->|raz| B\nA -->|dwa| B\nB --> C\nC --> A\nC -->|sam do siebie| C")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	if got := len(byRole(sc, "edge")); got != 6 {
		t.Errorf("%d edge paths, want 6", got)
	}
	nodes := byRole(sc, "node")
	for i := range nodes {
		for j := i + 1; j < len(nodes); j++ {
			if overlap(nodes[i].Bounds(), nodes[j].Bounds()) {
				t.Error("nodes overlap")
			}
		}
	}
}

// REQ: FLOW-011
// Graf na granicy limitów rozmieszcza się w rozsądnym czasie.
func TestLargeGraphFinishes(t *testing.T) {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for i := 0; i < graph.MaxNodes-1; i++ {
		fmt.Fprintf(&b, "n%d --> n%d\n", i, i+1)
		if i%3 == 0 && i+7 < graph.MaxNodes {
			fmt.Fprintf(&b, "n%d -->|skok| n%d\n", i, i+7)
		}
	}
	g, err := flowchart.Parse(b.String())
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	if d := time.Since(start); d > 20*time.Second {
		t.Errorf("layout of %d nodes took %s", len(g.Nodes), d)
	}
}
