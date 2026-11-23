package gomermaidgo

import (
	"math"
	"regexp"
	"strconv"

	"errors"
	"github.com/polestorm/gomermaidgo/fonts"
	"strings"
	"testing"
)

// REQ: COM-001 COM-002 COM-003
func TestPreamble(t *testing.T) {
	src := "\n---\ntitle: \"Tytuł z nagłówka\"\nconfig:\n  theme: dark\n  title: nie ten\n---\n%%{init: {\n  \"theme\": \"forest\"\n}}%%\nflowchart TD\n  A --> B\n  %%{ init: { \"flowchart\": { \"curve\": \"linear\" } } }%%\n  B --> C\n"
	svg, err := Render(src)
	if err != nil {
		t.Fatal(err)
	}
	out := string(svg)
	if !strings.Contains(out, "<title>Tytuł z nagłówka</title>") || !strings.Contains(out, `class="diagram-title"`) || strings.Count(out, "Tytuł z nagłówka") != 2 {
		t.Errorf("the front matter title should be the accessible title and a text above the drawing")
	}
	if strings.Contains(out, "theme") || strings.Contains(out, "nie ten") || strings.Count(out, `class="edge"`) != 2 {
		t.Errorf("configuration and directives should leave no trace in the drawing")
	}
	if kind, err := DetectType(src); err != nil || kind != TypeFlowchart {
		t.Errorf("type behind front matter: %q %v", kind, err)
	}
	if err := Check(src); err != nil {
		t.Errorf("Check: %v", err)
	}
	// Tytuł dostępności ze źródła diagramu wygrywa z tytułem z nagłówka, który nadal stoi nad rysunkiem.
	svg, err = Render("---\ntitle: Z nagłówka\n---\nflowchart TD\naccTitle: Własny\nA --> B")
	if err != nil || !strings.Contains(string(svg), "<title>Własny</title>") || !strings.Contains(string(svg), ">Z nagłówka</text>") {
		t.Errorf("accTitle should stay the accessible title: %v", err)
	}
	// Tytuł działa dla każdego typu, także rysowanego bez układu grafu.
	for _, body := range []string{"pie\n\"a\": 1", "sequenceDiagram\nA->>B: x", "kanban\n  K\n    [karta]"} {
		svg, err := Render("---\ntitle: Wspólny\n---\n" + body)
		if err != nil || !strings.Contains(string(svg), ">Wspólny</text>") {
			t.Errorf("%q: title missing or error %v", body, err)
		}
	}
	// Numery linii w błędach liczą się od początku źródła.
	var se *SyntaxError
	_, err = Render("---\ntitle: T\n---\n%%{init: {}}%%\nflowchart TD\nA -->")
	if !errors.As(err, &se) || se.Line != 6 {
		t.Errorf("error after front matter should point at line 6: %v", err)
	}
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"---\ntitle: T\nflowchart TD\nA --> B": {"frontmatter", 1},
		"flowchart TD\n%%{init: {\nA --> B":    {"directive", 2},
	} {
		_, err := Render(src)
		if !errors.As(err, &se) || se.Code != want.code || se.Line != want.line {
			t.Errorf("%q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: COM-005
// Emoji są mierzone krojem zapasowym i nie są zgłaszane jako brakujące; znak, którego nie ma
// żaden krój, nadal jest zgłaszany. Selektor wariantu nie zajmuje miejsca.
func TestEmojiFallback(t *testing.T) {
	width := func(src string) float64 {
		svg, err := Render(src)
		if err != nil {
			t.Fatal(err)
		}
		m := regexp.MustCompile(`<svg[^>]* width="([\d.]+)"`).FindSubmatch(svg)
		w, _ := strconv.ParseFloat(string(m[1]), 64)
		return w
	}
	plain, one, two := width("flowchart TD\nA[Nazwa pola abcd]"), width("flowchart TD\nA[🚀Nazwa pola abcd]"), width("flowchart TD\nA[🚀🚀Nazwa pola abcd]")
	// Krój zapasowy ma znaki szersze niż litery: każde emoji poszerza pole o tę samą, znaną wartość.
	if step := one - plain; step < 16 || step > 26 || math.Abs((two-one)-step) > 1.5 {
		t.Errorf("widths: %.1f, %.1f, %.1f", plain, one, two)
	}
	if with := width("flowchart TD\nA[⚠\ufe0fNazwa pola abcd]"); math.Abs(with-width("flowchart TD\nA[⚠Nazwa pola abcd]")) > 0.01 {
		t.Errorf("the variation selector should take no room")
	}
	// Domyślnie emoji jest konturem w kolorze tekstu, bez nazwy kroju zapasowego; słowo obok zostaje tekstem.
	outlined, _ := Render("flowchart TD\nA[🚀 Start]")
	o := string(outlined)
	if strings.Contains(o, "🚀") || strings.Contains(o, "Noto Emoji") || !regexp.MustCompile(`<path class="node-label emoji" d="M[^"]+" fill="#1b1f24"/>`).MatchString(o) || !strings.Contains(o, ">Start</text>") {
		t.Errorf("the emoji should be an outline: %s", o)
	}
	svg, _ := Render("flowchart TD\nA[🚀 Start]", WithEmojiText())
	// Z WithEmojiText nazwę kroju zapasowego niesie tylko odcinek z emoji; reszta napisu i rysunek zostają przy
	// kroju głównym, a odcinki stoją jeden za drugim.
	out := string(svg)
	m := regexp.MustCompile(`<text[^>]* x="([\d.]+)"[^>]* font-family="Noto Emoji"[^>]*>🚀</text>\s*<text[^>]* x="([\d.]+)"[^>]*>Start</text>`).FindStringSubmatch(out)
	if m == nil || !strings.Contains(out, `<svg xmlns="http://www.w3.org/2000/svg" version="1.1"`) || strings.Count(out, "Noto Emoji") != 1 || !strings.Contains(out, `font-family="Arimo, Arial, 'Liberation Sans', sans-serif"`) {
		t.Fatalf("the emoji should be its own text with the fallback family: %s", out)
	}
	x0, _ := strconv.ParseFloat(m[1], 64)
	x1, _ := strconv.ParseFloat(m[2], 64)
	if d := x1 - x0; d < 22 || d > 36 {
		t.Errorf("the second run should start an emoji and a space after the first: %.1f", d)
	}
	missing, err := MissingGlyphs("flowchart TD\nA[🚀 ✅ ⚠\ufe0f 漢]")
	if err != nil || string(missing) != "漢" {
		t.Errorf("missing glyphs: %q %v", string(missing), err)
	}
	// Krój podany przez wywołującego też dostaje krój zapasowy.
	if m, err := MissingGlyphs("flowchart TD\nA[🚀]", WithFont(fonts.ArimoBold()), WithFontFamily("X")); err != nil || len(m) != 0 {
		t.Errorf("caller's font: %q %v", string(m), err)
	}
}

// REQ: LAY-018
// Tryby dopasowania z opcji i z linii w źródle; opcja ma pierwszeństwo, auto jest domyślny.
func TestFitOptions(t *testing.T) {
	src := "flowchart LR\nA[Pierwszy długi krok procesu] --> B[Drugi długi krok procesu] --> C[Trzeci długi krok procesu] --> D[Czwarty długi krok procesu] --> E[Piąty długi krok procesu] --> F[Szósty długi krok procesu]"
	width := func(svg []byte) float64 {
		m := regexp.MustCompile(`<svg[^>]* width="([\d.]+)"`).FindSubmatch(svg)
		w, _ := strconv.ParseFloat(string(m[1]), 64)
		return w
	}
	def, _ := Render(src, WithFold(FoldOff))
	auto, _ := Render(src, WithFit(FitAuto), WithFold(FoldOff))
	off, _ := Render(src, WithFit(FitOff), WithFold(FoldOff))
	nofit, _ := Render("%% layout: nofit\n"+src, WithFold(FoldOff))
	forced, _ := Render("%% layout: nofit\n"+src, WithFit(FitOn), WithFold(FoldOff))
	high, _ := Render(src, WithFitRatio(100), WithFold(FoldOff))
	if string(def) != string(auto) || width(def) >= width(off)*0.85 {
		t.Errorf("default should be auto and narrower: %.0f vs off %.0f", width(def), width(off))
	}
	if string(nofit) != string(off) || width(forced) != width(def) || string(high) != string(off) {
		t.Errorf("directive, option precedence or ratio: nofit %.0f, forced %.0f, high ratio %.0f", width(nofit), width(forced), width(high))
	}
}

// REQ: COM-006
func TestLabelBackingFollowsBackground(t *testing.T) {
	src := readFile(t, "testdata/corpus/flowchart/005-edge-labels.mmd")
	clear, err := Render(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(clear), "<clipPath") || !strings.Contains(string(clear), `<g clip-path="url(#label-holes-`) {
		t.Error("transparent background: lines under labels should be clipped")
	}
	if strings.Contains(string(clear), `class="edge-label-background"`) || strings.Contains(string(clear), `fill="#ffffff"/>`) {
		t.Error("transparent background: label backings should not be painted")
	}
	filled, err := Render(src, WithBackground("#123456"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(filled), "<clipPath") || !regexp.MustCompile(`class="edge-label-background"[^>]*fill="#123456"`).Match(filled) {
		t.Error("background colour: label backings should be painted in it, without a clip")
	}
}

// REQ: COM-007
func TestUnstyledNodesStayNeutral(t *testing.T) {
	src := "graph LR\n  A[Plain] --> B[Styled]\n  style B fill:#cce5ff\n"
	for _, opts := range [][]Option{nil, {WithColors()}} {
		svg, err := Render(src, opts...)
		if err != nil {
			t.Fatal(err)
		}
		fills := regexp.MustCompile(`class="node"[^>]*fill="([^"]+)"`).FindAllStringSubmatch(string(svg), -1)
		if len(fills) != 2 || fills[0][1] != "#ffffff" || fills[1][1] != "#cce5ff" {
			t.Errorf("colors=%v: node fills %v, want #ffffff and #cce5ff", opts != nil, fills)
		}
	}
}

// REQ: LAY-020
func TestAuthorLineBreaksKept(t *testing.T) {
	src := "graph LR\n  A[Month 1<br/>$250] --> B[Month 2<br/>$280] --> C[Month 3<br/>$310] --> D[Month 4<br/>$295] --> E[Month 5<br/>$420] --> F[Month 6<br/>$450] --> G[Month 7<br/>$480]\n"
	svg, err := Render(src, WithFit(FitOn), WithFold(FoldOff))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{">Month 1<", ">Month 7<", ">$480<"} {
		if !strings.Contains(string(svg), want) {
			t.Errorf("fit on: line %q should stay as the author wrote it", want)
		}
	}
}

// REQ: COM-008
func TestWrapSettingsFromSource(t *testing.T) {
	long := "Izolacja calej partii i oznaczenie wyrobow"
	lines := func(svg []byte) int { return strings.Count(string(svg), `class="node-label"`) }
	render := func(src string) []byte {
		t.Helper()
		svg, err := Render(src)
		if err != nil {
			t.Fatal(err)
		}
		return svg
	}
	base := lines(render("flowchart LR\n  A[" + long + "]\n"))
	if base < 2 {
		t.Fatalf("default: the long label should wrap, got %d line(s)", base)
	}
	for _, src := range []string{
		"%%{init: {\"markdownAutoWrap\": false}}%%\nflowchart LR\n  A[" + long + "]\n",
		"---\nconfig:\n  markdownAutoWrap: false\n---\nflowchart LR\n  A[" + long + "]\n",
		"%%{init: {\"wrap\": false}}%%\nflowchart LR\n  A[" + long + "]\n",
	} {
		if n := lines(render(src)); n != 1 {
			t.Errorf("auto wrap off: label should stay on one line, got %d\n%s", n, src)
		}
	}
	if n := lines(render("%%{init: {\"flowchart\": {\"wrappingWidth\": 100}}}%%\nflowchart LR\n  A[" + long + "]\n")); n <= base {
		t.Errorf("wrappingWidth 100: want more than %d lines, got %d", base, n)
	}
	msg := "sequenceDiagram\n  A->>B: " + long + " " + long + "\n"
	count := func(svg []byte) int { return strings.Count(string(svg), `class="message-label"`) }
	if a, b := count(render(msg)), count(render("%%{init: {\"sequence\": {\"wrap\": true}}}%%\n"+msg)); a != 1 || b < 2 {
		t.Errorf("sequence wrap: want 1 line without and several with wrap, got %d and %d", a, b)
	}
}

// REQ: LAY-021
func TestChainKeepsAxis(t *testing.T) {
	svg, err := Render(readFile(t, "testdata/corpus/flowchart/026-chain-axis.mmd"), WithFold(FoldOff))
	if err != nil {
		t.Fatal(err)
	}
	rect := regexp.MustCompile(`class="node" x="([-\d.]+)" y="[-\d.]+" width="([-\d.]+)"`).FindAllStringSubmatch(string(svg), -1)
	rhomb := regexp.MustCompile(`class="node" points="([-\d.]+),`).FindStringSubmatch(string(svg))
	if len(rect) < 3 || rhomb == nil {
		t.Fatal("nodes not found")
	}
	x, _ := strconv.ParseFloat(rect[2][1], 64)
	w, _ := strconv.ParseFloat(rect[2][2], 64)
	top, _ := strconv.ParseFloat(rhomb[1], 64)
	if math.Abs(x+w/2-top) > 0.5 {
		t.Errorf("decision centre %.1f is off the chain axis %.1f", top, x+w/2)
	}
}

// REQ: LAY-022
func TestFoldChains(t *testing.T) {
	src := readFile(t, "testdata/corpus/flowchart/027-fold-chains.mmd")
	columns := func(opts ...Option) int {
		svg, err := Render(src, opts...)
		if err != nil {
			t.Fatal(err)
		}
		// Środki pól łańcucha (od „Ocena ryzyka” do „Raport CAPA”) w osi x.
		xs := map[string]bool{}
		for _, m := range regexp.MustCompile(`<text class="node-label" x="([-\d.]+)"[^>]*>(Ocena ryzyka|Szkolenie personelu|Raport CAPA)<`).FindAllStringSubmatch(string(svg), -1) {
			xs[m[1]] = true
		}
		return len(xs)
	}
	if n := columns(); n < 2 {
		t.Errorf("fold-chains from the source: chain should stand in at least 2 columns, got %d", n)
	}
	if n := columns(WithFoldChains(FoldOff)); n != 1 {
		t.Errorf("fold-chains off: chain should stand in one column, got %d", n)
	}
	if n := columns(WithFoldChains(FoldAuto), WithFoldType(FoldSnake)); n < 2 {
		t.Errorf("fold-chains snake: chain should stand in at least 2 columns, got %d", n)
	}
	// Domyślnie wyłączone: bez linii w źródle łańcuch zostaje w jednej kolumnie.
	plain := strings.Replace(src, "%% layout: fold-chains\n", "", 1)
	svg, err := Render(plain)
	if err != nil {
		t.Fatal(err)
	}
	xs := map[string]bool{}
	for _, m := range regexp.MustCompile(`<text class="node-label" x="([-\d.]+)"[^>]*>(Ocena ryzyka|Raport CAPA)<`).FindAllStringSubmatch(string(svg), -1) {
		xs[m[1]] = true
	}
	if len(xs) != 1 {
		t.Errorf("default: chain folding should be off, got %d columns", len(xs))
	}
}

// REQ: LAY-023 LAY-024
func TestMiddleBranchAndArrowClearance(t *testing.T) {
	svg, err := Render(readFile(t, "testdata/corpus/flowchart/027-fold-chains.mmd"), WithFoldChains(FoldOff))
	if err != nil {
		t.Fatal(err)
	}
	s := string(svg)
	// Romb „Skuteczne?” i pole „Działania uzupełniające”: środek pod środkiem.
	rhomb := regexp.MustCompile(`<polygon class="node" points="([-\d.]+),`).FindStringSubmatch(s)
	label := regexp.MustCompile(`<text class="node-label" x="([-\d.]+)"[^>]*>Działania uzupełniające<`).FindStringSubmatch(s)
	if rhomb == nil || label == nil {
		t.Fatal("decision or middle branch not found")
	}
	if rhomb[1] != label[1] {
		t.Errorf("middle branch centre %s is not under the decision centre %s", label[1], rhomb[1])
	}
	// Każda krawędź schematu kończąca się zakrętem ma za nim co najmniej 4 px prostej przed grotem.
	tail := regexp.MustCompile(`C[-\d. ]+ ([-\d.]+) ([-\d.]+) L([-\d.]+) ([-\d.]+)" fill="none"`)
	for _, f := range corpusFiles(t) {
		if !strings.Contains(f, "/flowchart/") {
			continue
		}
		out, err := Render(readFile(t, f))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range tail.FindAllStringSubmatch(string(out), -1) {
			x0, _ := strconv.ParseFloat(m[1], 64)
			y0, _ := strconv.ParseFloat(m[2], 64)
			x1, _ := strconv.ParseFloat(m[3], 64)
			y1, _ := strconv.ParseFloat(m[4], 64)
			if d := math.Hypot(x1-x0, y1-y0); d < 4 {
				t.Errorf("%s: edge ends %.1f px after its last turn (%s,%s)", f, d, m[3], m[4])
			}
		}
	}
}

// REQ: LAY-025
func TestFoldRanks(t *testing.T) {
	src := readFile(t, "testdata/corpus/flowchart/028-fold-ranks.mmd")
	leaves := regexp.MustCompile(`<text class="node-label" x="([-\d.]+)" y="([-\d.]+)"[^>]*>(Dokumenty|Zapisy|Audyty|Niezgodności|CAPA|Szkolenia|Zakupy|Dostawcy|Walidacja|Reklamacje|Ryzyko|Przegląd|Infrastruktura|Pomiary)<`)
	columns := func(src string, opts ...Option) (int, bool) {
		svg, err := Render(src, opts...)
		if err != nil {
			t.Fatal(err)
		}
		xs, ys := map[string]bool{}, map[string]float64{}
		for _, m := range leaves.FindAllStringSubmatch(string(svg), -1) {
			xs[m[1]] = true
			ys[m[3]], _ = strconv.ParseFloat(m[2], 64)
		}
		ordered, last := true, math.Inf(-1)
		for _, name := range []string{"Dokumenty", "Zapisy", "Audyty", "Niezgodności", "CAPA", "Szkolenia", "Zakupy", "Dostawcy", "Walidacja", "Reklamacje", "Ryzyko", "Przegląd", "Infrastruktura", "Pomiary"} {
			ordered = ordered && ys[name] > last
			last = ys[name]
		}
		return len(xs), ordered
	}
	if n, ordered := columns(src); n != 2 || !ordered {
		t.Errorf("fold-ranks from the source: want 2 staggered columns in source order, got %d columns, ordered %v", n, ordered)
	}
	if n, _ := columns(src, WithFoldRanks(FoldOff)); n != 1 {
		t.Errorf("fold-ranks off: leaves should stand in one column, got %d", n)
	}
	if n, _ := columns(strings.Replace(src, "%% layout: fold-ranks\n", "", 1)); n != 1 {
		t.Errorf("default: spreading should be off, got %d columns", n)
	}
}
