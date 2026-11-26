package treeview

import (
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// shape zapisuje drzewo w jednej linii: nazwa[/], dzieci w nawiasach.
func shape(nodes []*Node) string {
	var parts []string
	for _, n := range nodes {
		s := label(n)
		if len(n.Children) > 0 {
			s += "(" + shape(n.Children) + ")"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

// REQ: TREE-001 TREE-002 TREE-003
func TestParseIndentation(t *testing.T) {
	tr, err := Parse("%% komentarz\ntreeView-beta\n  accTitle: Projekt\n    proj/\n        \"src dir\"/\n            a.go\n\t\t\tb.go\n        README.md\n    LICENSE")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shape(tr.Roots), "proj/(src dir/(a.go b.go) README.md) LICENSE"; got != want {
		t.Errorf("tree %q, want %q", got, want)
	}
	if tr.AccTitle != "Projekt" {
		t.Errorf("accTitle %q", tr.AccTitle)
	}
}

// REQ: TREE-002
func TestParseBoxDrawing(t *testing.T) {
	for _, src := range []string{
		"treeView\n├── src/\n│   ├── a.ts\n│   └── b.ts\n│\n└── README.md",
		"treeView\n┣━━ src/\n┃   ┣━━ a.ts\n┃   ┗━━ b.ts\n┗━━ README.md",
	} {
		tr, err := Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := shape(tr.Roots), "src/(a.ts b.ts) README.md"; got != want {
			t.Errorf("tree %q, want %q", got, want)
		}
		if !tr.Roots[0].Branch || tr.Roots[0].Children[0].Branch != true {
			t.Errorf("branch flags not set")
		}
	}
	// Wynik polecenia tree: gałęzie od kolumny zero należą do korzenia „.”.
	tr, err := Parse("treeView-beta\n.\n├── a\n│   └── b\n└── c")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shape(tr.Roots), ".(a(b) c)"; got != want {
		t.Errorf("tree output %q, want %q", got, want)
	}
}

// REQ: TREE-003
func TestParseAnnotations(t *testing.T) {
	tr, err := Parse("treeView-beta\n  App.tsx :::highlight icon(logos:react) ## main component\n  docs/ icon(folder):::extra ## a ## b\n  \"x y\" ## z")
	if err != nil {
		t.Fatal(err)
	}
	a, d, q := tr.Roots[0], tr.Roots[1], tr.Roots[2]
	if a.Label != "App.tsx" || !a.Highlighted() || a.Icon != "logos:react" || a.Desc != "main component" {
		t.Errorf("first: %+v", a)
	}
	if d.Label != "docs" || !d.Dir() || d.Icon != "folder" || d.Desc != "a ## b" || strings.Join(d.Classes, ",") != "extra" {
		t.Errorf("second: %+v", d)
	}
	if q.Label != "x y" || q.Desc != "z" || q.Dir() {
		t.Errorf("third: %+v", q)
	}
}

// REQ: TREE-005
func TestParseErrors(t *testing.T) {
	deep := "treeView-beta\n"
	for i := 0; i <= maxDepth; i++ {
		deep += strings.Repeat(" ", i) + "d\n"
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                              {CodeHeader, 0},
		"tree\na":                       {CodeHeader, 1},
		"treeView-beta\n%% nic":         {CodeEmpty, 0},
		"treeView-beta\n  \"a b":        {CodeStatement, 2},
		"treeView-beta\n  a icon(x":     {CodeStatement, 2},
		"treeView-beta\n  a :::":        {CodeStatement, 2},
		"treeView-beta\n  :::highlight": {CodeStatement, 2},
		"treeView-beta\n  \"a\" b":      {CodeStatement, 2},
		deep:                            {CodeStatement, maxDepth + 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.30q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: TREE-004
func TestLayout(t *testing.T) {
	tr, err := Parse("treeView-beta\n├── src/\n│   ├── main.go :::highlight ## punkt wejścia\n│   └── util.go icon(file)\n└── README.md ## opis")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(tr, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var lines, marks, icons int
	var labels, descs []scene.Item
	for _, it := range sc.Items {
		switch it.Role {
		case "treeview-line":
			lines++
		case "treeview-highlight":
			marks++
		case "treeview-icon":
			icons++
		case "treeview-label":
			labels = append(labels, it)
		case "treeview-description":
			descs = append(descs, it)
		}
	}
	// Pień bez nazwy i pień src/: dwie pionowe; poziome do czterech wpisów.
	if lines != 6 || marks != 1 || icons != 2 || len(labels) != 4 || len(descs) != 2 {
		t.Fatalf("lines %d, marks %d, icons %d, labels %d, descriptions %d", lines, marks, icons, len(labels), len(descs))
	}
	if !labels[0].Bold || labels[0].Text != "src/" || labels[1].Bold {
		t.Errorf("directory labels: %+v %+v", labels[0], labels[1])
	}
	if descs[0].X != descs[1].X || !descs[0].Italic {
		t.Errorf("descriptions not in one column: %v %v", descs[0].X, descs[1].X)
	}
	for i := 1; i < len(labels); i++ {
		if labels[i].Y <= labels[i-1].Y {
			t.Errorf("rows overlap: %v after %v", labels[i].Y, labels[i-1].Y)
		}
		if labels[i].X+labels[i].W > descs[0].X && (labels[i].Text == "main.go" || labels[i].Text == "README.md") {
			t.Errorf("label %q runs into the description column", labels[i].Text)
		}
	}
	// Ten sam wynik przy każdym uruchomieniu.
	if string(sc.SVG()) != string(Layout(tr, Options{Measure: font.ApproxTextWidth, FontSize: 16}).SVG()) {
		t.Error("layout is not deterministic")
	}
}
