package architecture

import (
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

const sample = "%% komentarz\narchitecture-beta\n  title Tytuł\n  accTitle: Dostępny\n  group g(cloud)[Grupa]\n  group w[Wewnętrzna] in g\n  service a(server)[Usługa A] in g\n  service b(database) in w\n  service c[Bez ikony]\n  junction j in g\n  service osobna(disk)[Osobna]\n  a:R --> L:c\n  a:B -- T:j\n  j:L <-[opis]-> T:b\n  b{group}:R -- L:j\n"

// REQ: ARC-001 ARC-002
func TestParse(t *testing.T) {
	// {group} przy usłudze bez grupy jest błędem, więc próbka do rozbioru ma go przy usłudze w grupie.
	d, err := Parse("architecture-beta\n  title Tytuł\n  accTitle: Dostępny\n  group g(cloud)[Grupa]\n  group w[Wewnętrzna] in g\n  service a(server)[Usługa A] in g\n  service b(database) in w\n  service c[Bez ikony]\n  junction j in g\n  a:R --> L:c\n  j:L <-[opis]-> T:b\n  b{group}:B -- T:a{group}\n")
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Tytuł" || d.AccTitle != "Dostępny" || len(d.Groups) != 2 || len(d.Nodes) != 4 || len(d.Edges) != 3 {
		t.Fatalf("diagram: %+v", d)
	}
	if g := d.Groups[1]; g.ID != "w" || g.Label != "Wewnętrzna" || g.Parent != "g" || g.Icon != "" {
		t.Errorf("nested group: %+v", g)
	}
	if a := d.Nodes[0]; a.Icon != "server" || a.Label != "Usługa A" || a.Group != "g" {
		t.Errorf("service: %+v", a)
	}
	if b, c, j := d.Nodes[1], d.Nodes[2], d.Nodes[3]; b.Label != "b" || b.Icon != "database" || c.Label != "Bez ikony" || c.Group != "" || !j.Junction || j.Group != "g" {
		t.Errorf("services and junction: %+v %+v %+v", b, c, j)
	}
	if e := d.Edges[0]; e.From != "a" || e.FromSide != "R" || e.To != "c" || e.ToSide != "L" || e.FromArrow || !e.ToArrow {
		t.Errorf("arrow edge: %+v", e)
	}
	if e := d.Edges[1]; !e.FromArrow || !e.ToArrow || e.Label != "opis" || e.FromSide != "L" || e.ToSide != "T" {
		t.Errorf("labelled edge with two arrows: %+v", e)
	}
	if e := d.Edges[2]; !e.FromGroup || !e.ToGroup || e.FromArrow || e.ToArrow {
		t.Errorf("edge between groups: %+v", e)
	}
}

// REQ: ARC-004
func TestParseErrors(t *testing.T) {
	head := "architecture-beta\nservice a(server)[A]\n"
	many := "architecture-beta\n"
	for i := 0; i < 101; i++ {
		many += "service s" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + "\n"
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                    {CodeHeader, 0},
		"architectures\nservice a":            {CodeHeader, 1},
		head + "to nie jest instrukcja":       {CodeStatement, 3},
		head + "service a(disk)[Drugi]":       {CodeNode, 3},
		head + "group a[Też a]":               {CodeNode, 3},
		head + "service b in nieznana":        {CodeNode, 3},
		head + "a:R -- L:nieznana":            {CodeEdge, 3},
		head + "a:R -- L:a":                   {CodeEdge, 3},
		head + "service b\na{group}:R -- L:b": {CodeEdge, 4},
		head + "service b\na:X -- L:b":        {CodeStatement, 4},
		"architecture-beta\ngroup g[G]":       {CodeEmpty, 0},
		many:                                  {CodeNode, 102},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: ARC-002 ARC-003
// Położenie w siatce wynika z boków krawędzi; usługa spoza grupy nie stoi w jej obrębie; części
// niepołączone stoją jedna pod drugą.
func TestGrid(t *testing.T) {
	d, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	g := Grid(d)
	rel := func(a, b string) [2]int { return [2]int{g[b][0] - g[a][0], g[b][1] - g[a][1]} }
	if r := rel("a", "j"); r != [2]int{0, 1} {
		t.Errorf("j should stand below a: %v", r)
	}
	if r := rel("j", "b"); r != [2]int{-1, 1} {
		t.Errorf("an edge from the left side to a top side puts b down-left of j: %v", r)
	}
	// c jest po prawej stronie a, poza kolumnami zajętymi przez grupę g (a, j, b).
	if r := rel("a", "c"); r[1] != 0 || r[0] < 1 {
		t.Errorf("c should stand to the right of a: %v", r)
	}
	seen := map[[2]int]string{}
	for id, p := range g {
		if other, dup := seen[p]; dup {
			t.Errorf("%s and %s share a cell", id, other)
		}
		seen[p] = id
		if p[0] < 0 || p[1] < 0 {
			t.Errorf("%s has a negative position %v", id, p)
		}
	}
	lowest := 0
	for _, id := range []string{"a", "b", "c", "j"} {
		lowest = max(lowest, g[id][1])
	}
	if g["osobna"][1] <= lowest {
		t.Errorf("an unconnected service should stand below the rest: row %d, the rest ends at %d", g["osobna"][1], lowest)
	}
}

// REQ: ARC-003
func TestLayout(t *testing.T) {
	d, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true, Shadows: true})
	var services, groups []scene.Item
	roles, texts := map[string]int{}, map[string]int{}
	for _, it := range sc.Items {
		roles[it.Role]++
		switch it.Role {
		case "architecture-service":
			services = append(services, it)
		case "architecture-group":
			groups = append(groups, it)
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	// Usługi w kolejności źródła: a, b, c, osobna. Grupy: g, w.
	if len(services) != 4 || len(groups) != 2 || roles["architecture-junction"] != 1 || roles["edge"] != 4 || roles["arrowhead"] != 3 {
		t.Fatalf("%d services, %d groups, roles %v", len(services), len(groups), roles)
	}
	inside := func(s, g scene.Item) bool {
		a, b := s.Bounds(), g.Bounds()
		return a.X0 >= b.X0 && a.X1 <= b.X1 && a.Y0 >= b.Y0 && a.Y1 <= b.Y1
	}
	a, b, c := services[0], services[1], services[2]
	if !inside(a, groups[0]) || !inside(b, groups[0]) || !inside(b, groups[1]) || inside(a, groups[1]) {
		t.Errorf("services should lie in the frames of their groups")
	}
	if inside(c, groups[0]) {
		t.Errorf("a service outside the group lies in its frame")
	}
	if p, q := groups[1].Bounds(), groups[0].Bounds(); p.X0 < q.X0 || p.X1 > q.X1 || p.Y0 < q.Y0 || p.Y1 > q.Y1 {
		t.Errorf("the nested group should lie in its parent: %+v %+v", p, q)
	}
	for _, want := range []string{"Tytuł", "Grupa", "Wewnętrzna", "Usługa A", "b", "Bez ikony", "Osobna", "opis"} {
		if texts[want] != 1 {
			t.Errorf("text %q drawn %d times, want once", want, texts[want])
		}
	}
}
