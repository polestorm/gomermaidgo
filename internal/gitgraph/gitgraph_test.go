package gitgraph

import (
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
		"gitGraph TB:\ncommit":     {CodeUnsupported, 1},
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
