package gitgraph

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

const sample = "%% komentarz\ngitGraph LR:\n  accTitle: Historia\n  commit id: \"a\"\n  commit id: \"b\" tag: \"v1\" type: HIGHLIGHT\n  branch \"gałąź boczna\"\n  commit id: \"c\"\n  checkout main\n  commit type: REVERSE\n  merge \"gałąź boczna\" id: \"m\" tag: \"v2\"\n  branch druga\n  switch main\n  commit id: \"d\"\n  checkout druga\n  cherry-pick id: \"d\"\n"

// REQ: GIT-001 GIT-002
func TestParse(t *testing.T) {
	g, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if g.AccTitle != "Historia" || strings.Join(g.Branches, "|") != "main|gałąź boczna|druga" || len(g.Commits) != 7 {
		t.Fatalf("graph: %+v", g)
	}
	c := g.Commits
	if c[0].Parent != -1 || c[1].Parent != 0 || c[1].Tag != "v1" || c[1].Type != Highlight {
		t.Errorf("first commits: %+v %+v", c[0], c[1])
	}
	if c[2].Branch != 1 || c[2].Parent != 1 || c[2].ID != "c" {
		t.Errorf("first commit of a branch should point at its fork: %+v", c[2])
	}
	if c[3].Branch != 0 || c[3].Parent != 1 || c[3].Type != Reverse || c[3].ID != "" {
		t.Errorf("commit after checkout: %+v", c[3])
	}
	if c[4].Merged != 2 || c[4].Parent != 3 || c[4].ID != "m" || c[4].Tag != "v2" {
		t.Errorf("merge: %+v", c[4])
	}
	if c[6].Branch != 2 || c[6].Picked != 5 || c[6].Parent != 4 {
		t.Errorf("cherry-pick on a branch forked from the merge: %+v", c[6])
	}
	if g, err := Parse("gitGraph\ncommit"); err != nil || len(g.Commits) != 1 {
		t.Errorf("plain header: %v", err)
	}
}

// REQ: GIT-004
func TestParseErrors(t *testing.T) {
	head := "gitGraph\ncommit id: \"a\"\n"
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                         {CodeHeader, 0},
		"gitgraph\ncommit":         {CodeHeader, 1},
		head + "push":              {CodeStatement, 3},
		head + "branch":            {CodeBranch, 3},
		head + "branch main":       {CodeBranch, 3},
		"gitGraph\nbranch x":       {CodeBranch, 2},
		head + "checkout nieznana": {CodeBranch, 3},
		head + "merge main":        {CodeBranch, 3},
		head + "branch x\nmerge x": {CodeBranch, 4},
		head + "branch x\ncheckout main\nmerge x":      {CodeBranch, 5},
		head + "commit id: \"a\"":                      {CodeCommit, 3},
		head + "commit type: DZIWNY":                   {CodeCommit, 3},
		head + "cherry-pick id: \"zz\"":                {CodeCommit, 3},
		head + "cherry-pick id: \"a\"":                 {CodeCommit, 3},
		"gitGraph\n%% nic":                             {CodeEmpty, 0},
		"gitGraph\n" + strings.Repeat("commit\n", 301): {CodeCommit, 302},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.50q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: GIT-003
func TestLayout(t *testing.T) {
	g, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	var commits, labels []scene.Item
	roles := map[string]int{}
	texts := map[string]int{}
	for _, it := range sc.Items {
		roles[it.Role]++
		switch it.Role {
		case "git-commit":
			commits = append(commits, it)
		case "git-branch":
			labels = append(labels, it)
		}
		if it.Kind == scene.Text {
			texts[it.Text]++
		}
	}
	if len(commits) != 7 || len(labels) != 3 {
		t.Fatalf("%d commits and %d branch labels", len(commits), len(labels))
	}
	centre := func(it scene.Item) (float64, float64) {
		b := it.Bounds()
		return (b.X0 + b.X1) / 2, (b.Y0 + b.Y1) / 2
	}
	var xs, ys []float64
	for _, c := range commits {
		x, y := centre(c)
		xs, ys = append(xs, x), append(ys, y)
	}
	for i := 1; i < len(xs); i++ {
		if xs[i] <= xs[i-1] {
			t.Errorf("commit %d does not stand to the right of the previous one", i)
		}
	}
	if !(ys[0] == ys[1] && ys[1] == ys[3] && ys[3] == ys[4] && ys[2] > ys[0] && ys[6] > ys[2]) {
		t.Errorf("commits should lie on the lane of their branch: %v", ys)
	}
	for i, l := range labels {
		if _, y := centre(l); y != []float64{ys[0], ys[2], ys[6]}[i] || l.X+l.W > xs[0] {
			t.Errorf("branch label %d is not at the left of its lane", i)
		}
	}
	if commits[1].Kind != scene.Rect || commits[0].Kind != scene.Circle || commits[4].Fill != "#ffffff" {
		t.Errorf("commit marks: highlight %v, normal %v, merge fill %q", commits[1].Kind, commits[0].Kind, commits[4].Fill)
	}
	// Dwa odejścia gałęzi, jedno scalenie, jedna kopia; trzy tory.
	if roles["git-fork"] != 2 || roles["git-merge"] != 1 || roles["git-pick"] != 1 || roles["git-lane"] != 3 {
		t.Errorf("links: %v", roles)
	}
	for _, want := range []string{"main", "gałąź boczna", "druga", "a", "b", "c", "m", "d", "v1", "v2"} {
		if texts[want] != 1 {
			t.Errorf("text %q drawn %d times, want once", want, texts[want])
		}
	}
}

// REQ: GIT-005
// Graf w pionie: gałęzie są kolumnami, commity biegną w dół (TB) albo w górę (BT), nazwa gałęzi
// stoi przed pierwszym commitem, a opis commitu z jego prawej strony, przed następną kolumną.
func TestVertical(t *testing.T) {
	for _, dir := range []string{"TB", "BT"} {
		g, err := Parse("gitGraph " + dir + ":\ncommit id: \"a\"\nbranch boczna\ncommit id: \"b\" tag: \"v1\"\ncheckout main\ncommit id: \"c\"\nmerge boczna")
		if err != nil || g.Direction != dir {
			t.Fatalf("%s: %v", dir, err)
		}
		sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
		var commits, labels, ids []scene.Item
		lanes, merges := 0, 0
		for _, it := range sc.Items {
			switch it.Role {
			case "git-commit":
				commits = append(commits, it)
			case "git-branch":
				labels = append(labels, it)
			case "git-commit-id":
				ids = append(ids, it)
			case "git-lane":
				lanes++
				if it.X != it.X2 {
					t.Errorf("%s: a lane should be vertical", dir)
				}
			case "git-merge":
				merges++
			}
		}
		if len(commits) != 4 || len(labels) != 2 || lanes != 2 || merges != 1 {
			t.Fatalf("%s: %d commits, %d labels, %d lanes, %d merges", dir, len(commits), len(labels), lanes, merges)
		}
		centre := func(it scene.Item) (float64, float64) { b := it.Bounds(); return (b.X0 + b.X1) / 2, (b.Y0 + b.Y1) / 2 }
		x0, y0 := centre(commits[0])
		x1, y1 := centre(commits[1])
		x2, y2 := centre(commits[2])
		if x0 != x2 || x1 <= x0 {
			t.Errorf("%s: columns of main and the branch: %v %v %v", dir, x0, x1, x2)
		}
		forward := y0 < y1 && y1 < y2
		_, ly := centre(labels[0])
		if dir == "BT" {
			forward = y0 > y1 && y1 > y2
			if ly < y0 {
				t.Errorf("BT: the branch name should stand under the first commit")
			}
		} else if ly > y0 {
			t.Errorf("TB: the branch name should stand over the first commit")
		}
		if !forward {
			t.Errorf("%s: commits should run in source order: %v %v %v", dir, y0, y1, y2)
		}
		// Opis commitu głównej gałęzi kończy się przed kolumną gałęzi bocznej.
		if b := ids[0].Bounds(); b.X0 < x0 || b.X1 >= x1 {
			t.Errorf("%s: commit id leaves its column: %+v", dir, b)
		}
	}
}

// REQ: GIT-006
// Linie prowadzące torów, odstępy z własnych podpisów i znak kopii.
func TestGuidesSpacingAndPickMark(t *testing.T) {
	src := "commit id: \"a\"\ncommit id: \"b\"\nbranch boczna\ncommit id: \"bardzo-dlugi-identyfikator-commitu\"\ncheckout main\ncommit id: \"c\"\ncherry-pick id: \"bardzo-dlugi-identyfikator-commitu\"\ncommit id: \"d\""
	for _, header := range []string{"gitGraph\n", "gitGraph TB:\n"} {
		g, err := Parse(header + src)
		if err != nil {
			t.Fatal(err)
		}
		sc := Layout(g, Options{Measure: font.ApproxTextWidth, FontSize: 16})
		var guides, lanes, commits []scene.Item
		details := 0
		for _, it := range sc.Items {
			switch it.Role {
			case "git-guide":
				guides = append(guides, it)
			case "git-lane":
				lanes = append(lanes, it)
			case "git-commit":
				commits = append(commits, it)
			case "git-commit-detail":
				details++
			}
		}
		if len(guides) != 2 || len(lanes) != 2 || len(commits) != 6 {
			t.Fatalf("%q: %d guides, %d lanes, %d commits", header, len(guides), len(lanes), len(commits))
		}
		length := func(it scene.Item) float64 { return math.Abs(it.X2-it.X) + math.Abs(it.Y2-it.Y) }
		for i, gd := range guides {
			if gd.Dash == "" || gd.StrokeWidth >= lanes[i].StrokeWidth || length(gd) < length(lanes[i]) || length(guides[0])-length(gd) > 0.01 {
				t.Errorf("%q: guide %d should be thin, dashed and as long as the graph: %+v", header, i, gd)
			}
		}
		// Kopia: koło z dwiema wiśniami (dwa kółka i dwa ogonki).
		if details != 4 {
			t.Errorf("%q: cherry-pick mark has %d parts, want 4", header, details)
		}
		if header == "gitGraph\n" {
			// Krótkie podpisy stoją gęsto, długi rozsuwa tylko swoich sąsiadów.
			x := func(i int) float64 { b := commits[i].Bounds(); return (b.X0 + b.X1) / 2 }
			if near, far := x(1)-x(0), x(2)-x(1); near > 60 || far < 2*near {
				t.Errorf("spacing: %v between short ids, %v next to the long one", near, far)
			}
		}
	}
}
