// Package gitgraph rozbiera i rysuje graf historii Gita (gitGraph).
// Składnia według publicznej dokumentacji Mermaid, w zakresie profilu z docs/profile.md.
package gitgraph

import (
	"math"
	"regexp"
	"strings"

	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// Kody błędów składni grafu Gita.
const (
	CodeHeader      = "gitgraph-header"
	CodeUnsupported = "gitgraph-unsupported"
	CodeStatement   = "gitgraph-statement"
	CodeBranch      = "gitgraph-branch"
	CodeCommit      = "gitgraph-commit"
	CodeEmpty       = "gitgraph-empty"
	maxCommits      = 300
	maxBranches     = 24
)

// Rodzaje commitów.
const (
	Normal    = "NORMAL"
	Reverse   = "REVERSE"
	Highlight = "HIGHLIGHT"
)

// Commit to commit na gałęzi. Parent to numer poprzednika na tej samej gałęzi albo commitu,
// od którego gałąź odchodzi (-1 dla pierwszego commitu). Merged i Picked wskazują commit
// włączony scaleniem albo skopiowany (-1, gdy nie dotyczy).
type Commit struct {
	Branch int
	ID     string // identyfikator ze źródła; pusty, gdy nie podano
	Tag    string
	Type   string
	Parent int
	Merged int
	Picked int
	Line   int
}

// Graph to graf historii: gałęzie w kolejności powstania i commity w kolejności źródła.
type Graph struct {
	Branches []string
	Commits  []Commit
	AccTitle string
	AccDescr string
}

var (
	header = regexp.MustCompile(`^gitGraph\s*(LR|TB|BT)?\s*:?$`)
	option = regexp.MustCompile(`(id|tag|type|msg|parent|order)\s*:\s*(?:"([^"]*)"|(\S+))`)
	ident  = regexp.MustCompile(`^(?:"([^"]+)"|([^\s"]+))`)
)

// Parse rozbiera źródło grafu Gita.
func Parse(source string) (*Graph, error) {
	g := &Graph{Branches: []string{"main"}}
	var acc syntax.Accessibility
	seenHeader := false
	current := 0
	head := map[int]int{} // gałąź → numer jej ostatniego commitu
	byID := map[string]int{}
	index := map[string]int{"main": 0}
	fork := map[int]int{} // gałąź bez commitów → commit, od którego odchodzi
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for n, raw := range strings.Split(source, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if !seenHeader {
			m := header.FindStringSubmatch(s)
			if m == nil {
				return nil, syntax.Errorf(line, CodeHeader, "expected \"gitGraph\", found %q", s)
			}
			if m[1] == "TB" || m[1] == "BT" {
				return nil, syntax.Errorf(line, CodeUnsupported, "only the left-to-right orientation is supported")
			}
			seenHeader = true
			continue
		}
		if acc.Line(s) {
			continue
		}
		word, rest := s, ""
		if i := strings.IndexAny(s, " \t"); i >= 0 {
			word, rest = s[:i], strings.TrimSpace(s[i+1:])
		}
		opts := map[string]string{}
		for _, m := range option.FindAllStringSubmatch(rest, -1) {
			opts[m[1]] = m[2] + m[3]
		}
		name := ""
		if m := ident.FindStringSubmatch(rest); m != nil && !option.MatchString(m[0]+" ") && !strings.HasSuffix(m[2], ":") {
			name = m[1] + m[2]
		}
		add := func(c Commit) error {
			if len(g.Commits) >= maxCommits {
				return syntax.Errorf(line, CodeCommit, "too many commits (limit %d)", maxCommits)
			}
			if c.ID != "" {
				if _, dup := byID[c.ID]; dup {
					return syntax.Errorf(line, CodeCommit, "commit id %q is used twice", c.ID)
				}
				byID[c.ID] = len(g.Commits)
			}
			switch c.Type {
			case "":
				c.Type = Normal
			case Normal, Reverse, Highlight:
			default:
				return syntax.Errorf(line, CodeCommit, "commit type is NORMAL, REVERSE or HIGHLIGHT, found %q", c.Type)
			}
			c.Branch, c.Line = current, line
			if h, ok := head[current]; ok {
				c.Parent = h
			} else if f, ok := fork[current]; ok {
				c.Parent = f
			} else {
				c.Parent = -1
			}
			head[current] = len(g.Commits)
			g.Commits = append(g.Commits, c)
			return nil
		}
		switch word {
		case "commit":
			if err := add(Commit{ID: opts["id"], Tag: opts["tag"], Type: opts["type"], Merged: -1, Picked: -1}); err != nil {
				return nil, err
			}
		case "branch":
			if name == "" {
				return nil, syntax.Errorf(line, CodeBranch, "branch needs a name")
			}
			if _, exists := index[name]; exists {
				return nil, syntax.Errorf(line, CodeBranch, "branch %q already exists", name)
			}
			if len(g.Branches) >= maxBranches {
				return nil, syntax.Errorf(line, CodeBranch, "too many branches (limit %d)", maxBranches)
			}
			h, ok := head[current]
			if !ok {
				return nil, syntax.Errorf(line, CodeBranch, "branch %q starts from a branch without commits", name)
			}
			index[name] = len(g.Branches)
			fork[len(g.Branches)] = h
			g.Branches = append(g.Branches, name)
			current = index[name]
		case "checkout", "switch":
			b, ok := index[name]
			if !ok {
				return nil, syntax.Errorf(line, CodeBranch, "%s refers to unknown branch %q", word, name)
			}
			current = b
		case "merge":
			b, ok := index[name]
			if !ok {
				return nil, syntax.Errorf(line, CodeBranch, "merge refers to unknown branch %q", name)
			}
			h, has := head[b]
			if b == current || !has {
				return nil, syntax.Errorf(line, CodeBranch, "branch %q cannot be merged here: it is the current branch or has no commits", name)
			}
			if _, here := head[current]; !here {
				if _, forked := fork[current]; !forked {
					return nil, syntax.Errorf(line, CodeBranch, "the current branch has no commits to merge into")
				}
			}
			if err := add(Commit{ID: opts["id"], Tag: opts["tag"], Type: opts["type"], Merged: h, Picked: -1}); err != nil {
				return nil, err
			}
		case "cherry-pick":
			src, ok := byID[opts["id"]]
			if !ok {
				return nil, syntax.Errorf(line, CodeCommit, "cherry-pick refers to unknown commit id %q", opts["id"])
			}
			if g.Commits[src].Branch == current {
				return nil, syntax.Errorf(line, CodeCommit, "cherry-pick takes a commit from another branch")
			}
			// Kopia bez własnego znacznika dostaje znacznik ze źródłem, żeby było widać, co skopiowano.
			tag := opts["tag"]
			if tag == "" {
				tag = "cherry-pick:" + opts["id"]
			}
			if err := add(Commit{Tag: tag, Merged: -1, Picked: src}); err != nil {
				return nil, err
			}
		default:
			return nil, syntax.Errorf(line, CodeStatement, "unrecognised statement %q", s)
		}
	}
	if !seenHeader {
		return nil, syntax.Errorf(0, CodeHeader, "empty source: expected \"gitGraph\"")
	}
	if acc.Open() {
		return nil, syntax.Errorf(0, CodeStatement, "accDescr is not closed with \"}\"")
	}
	if len(g.Commits) == 0 {
		return nil, syntax.Errorf(0, CodeEmpty, "the graph needs at least one commit")
	}
	g.AccTitle, g.AccDescr = acc.Title, acc.Descr
	return g, nil
}

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // każda gałąź w swoim kolorze palety
	Shadows    bool // przyjmowane dla zgodności opcji; graf nie ma cieni
}

const (
	colorText  = "#1b1f24"
	colorMuted = "#5b6570"
	colorLine  = "#3b4754"
	colorLabel = "#eef2f7"
	colorTag   = "#ffffff"
)

// Layout rozmieszcza graf: gałęzie są poziomymi torami w kolejności powstania, commity stoją
// od lewej do prawej w kolejności źródła. Odejście gałęzi, scalenie i skopiowanie commitu są
// łącznikami między torami.
func Layout(g *Graph, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return v * fs / 16 }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.8)
	sc := &scene.Scene{Title: g.AccTitle, Desc: g.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}

	// Odstęp commitów mieści najdłuższy identyfikator i znacznik.
	step := u(52)
	for _, c := range g.Commits {
		step = math.Max(step, math.Max(o.Measure(c.ID, small), o.Measure(c.Tag, small)+u(12))+u(12))
	}
	labelW := 0.0
	for _, b := range g.Branches {
		labelW = math.Max(labelW, o.Measure(b, fs))
	}
	x0 := labelW + 2*u(10) + u(24)
	lane := u(64)
	at := func(i int) (x, y float64) { return x0 + float64(i)*step, float64(g.Commits[i].Branch)*lane + u(28) }
	tone := func(branch int) string {
		if o.Colors {
			return scene.Palette[branch%len(scene.Palette)].Stroke
		}
		return colorLine
	}

	// Tory: etykieta gałęzi z lewej i linia od pierwszego do ostatniego commitu gałęzi.
	first, last := map[int]int{}, map[int]int{}
	for i, c := range g.Commits {
		if _, ok := first[c.Branch]; !ok {
			first[c.Branch] = i
		}
		last[c.Branch] = i
	}
	for b, name := range g.Branches {
		f, ok := first[b]
		if !ok {
			continue // gałąź bez commitów nie ma toru
		}
		_, y := at(f)
		w := o.Measure(name, fs) + 2*u(10)
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "git-branch", X: labelW + 2*u(10) - w, Y: y - lh/2 - u(4), W: w, H: lh + u(8), RX: u(4),
			Fill: colorLabel, Stroke: tone(b), StrokeWidth: 1.5})
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "git-branch-label", X: labelW + 2*u(10) - w/2, Y: y + 0.35*fs, W: o.Measure(name, fs), H: lh,
			Text: name, Anchor: "middle", Fill: colorText})
		xa, _ := at(f)
		xb, _ := at(last[b])
		if b == 0 {
			xa = labelW + 2*u(10) // tor gałęzi głównej zaczyna się przy jej etykiecie
		}
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "git-lane", X: xa, Y: y, X2: xb, Y2: y, Stroke: tone(b), StrokeWidth: 2})
	}
	// elbow rysuje łącznik między torami: wzdłuż toru źródła, potem w poprzek do celu, z jednym
	// zaokrąglonym narożnikiem. Gdy turnAtStart, najpierw w poprzek, potem wzdłuż toru celu.
	elbow := func(xa, ya, xb, yb float64, turnAtStart bool, stroke, dash, role string) {
		r := math.Min(u(10), math.Min(math.Abs(xb-xa), math.Abs(yb-ya)))
		dy := 1.0
		if yb < ya {
			dy = -1
		}
		const k = 0.45
		var segments []scene.Segment
		if turnAtStart {
			segments = []scene.Segment{{Cmd: 'M', To: scene.Point{X: xa, Y: ya}}, {Cmd: 'L', To: scene.Point{X: xa, Y: yb - dy*r}},
				{Cmd: 'C', C1: scene.Point{X: xa, Y: yb - dy*r*k}, C2: scene.Point{X: xa + r*k, Y: yb}, To: scene.Point{X: xa + r, Y: yb}},
				{Cmd: 'L', To: scene.Point{X: xb, Y: yb}}}
		} else {
			segments = []scene.Segment{{Cmd: 'M', To: scene.Point{X: xa, Y: ya}}, {Cmd: 'L', To: scene.Point{X: xb - r, Y: ya}},
				{Cmd: 'C', C1: scene.Point{X: xb - r*k, Y: ya}, C2: scene.Point{X: xb, Y: ya + dy*r*k}, To: scene.Point{X: xb, Y: ya + dy*r}},
				{Cmd: 'L', To: scene.Point{X: xb, Y: yb}}}
		}
		sc.Add(scene.Item{Kind: scene.Path, Layer: scene.LayerLifeline, Role: role, Segments: segments, Stroke: stroke, StrokeWidth: 2, Dash: dash})
	}
	for i, c := range g.Commits {
		x, y := at(i)
		if c.Parent >= 0 && g.Commits[c.Parent].Branch != c.Branch {
			px, py := at(c.Parent)
			elbow(px, py, x, y, true, tone(c.Branch), "", "git-fork")
		}
		if c.Merged >= 0 {
			mx, my := at(c.Merged)
			elbow(mx, my, x, y, false, tone(g.Commits[c.Merged].Branch), "", "git-merge")
		}
		if c.Picked >= 0 {
			px, py := at(c.Picked)
			elbow(px, py, x, y, false, tone(g.Commits[c.Picked].Branch), "4 3", "git-pick")
		}
	}
	r := u(8)
	for i, c := range g.Commits {
		x, y := at(i)
		stroke := tone(c.Branch)
		switch {
		case c.Type == Highlight:
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "git-commit", X: x - r, Y: y - r, W: 2 * r, H: 2 * r, Fill: stroke, Stroke: stroke, StrokeWidth: 2})
		case c.Merged >= 0 || c.Picked >= 0:
			// Scalenie i kopia: pusty okrąg, żeby odróżnić je od zwykłego commitu.
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "git-commit", X: x, Y: y, R: r, Fill: "#ffffff", Stroke: stroke, StrokeWidth: 2.5})
			if c.Merged >= 0 {
				sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "git-commit-detail", X: x, Y: y, R: r * 0.45, Fill: stroke})
			}
		default:
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "git-commit", X: x, Y: y, R: r, Fill: stroke, Stroke: stroke, StrokeWidth: 2})
		}
		if c.Type == Reverse {
			for _, d := range []float64{-1, 1} {
				sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLabel, Role: "git-commit-detail", X: x - r*0.5, Y: y - d*r*0.5, X2: x + r*0.5, Y2: y + d*r*0.5,
					Stroke: "#ffffff", StrokeWidth: 2})
			}
		}
		if c.ID != "" {
			// Podkład pod identyfikatorem: łącznik gałęzi odchodzącej w dół nie przekreśla tekstu.
			w := o.Measure(c.ID, small)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabel, Role: "git-commit-id-background", X: x - w/2 - u(2), Y: y + r + u(4), W: w + u(4), H: small * 1.3, Fill: "#ffffff"})
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "git-commit-id", X: x, Y: y + r + u(4) + small, W: o.Measure(c.ID, small), H: small * 1.25,
				Text: c.ID, Anchor: "middle", FontSize: small, Fill: colorMuted})
		}
		if c.Tag != "" {
			w := o.Measure(c.Tag, small) + u(12)
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "git-tag", X: x - w/2, Y: y - r - u(6) - small*1.4, W: w, H: small * 1.4, RX: u(3),
				Fill: colorTag, Stroke: colorLine, StrokeWidth: 1})
			sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "git-tag-label", X: x, Y: y - r - u(6) - small*0.4, W: o.Measure(c.Tag, small), H: small * 1.25,
				Text: c.Tag, Anchor: "middle", FontSize: small, Fill: colorText})
		}
	}
	return sc
}
