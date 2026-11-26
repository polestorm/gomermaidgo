// Polecenie review składa jedną stronę HTML do przeglądu korpusu przez człowieka: dla każdego
// pliku tytuł, opis tego, co sprawdza, wymagania, źródło, nasz rysunek i (opcjonalnie) rysunek
// renderera odniesienia. Strona pozwala zaznaczyć ocenę i skopiować uwagi.
//
//	go run ./tools/review -font NotoSans-Regular.ttf -family "Noto Sans" \
//	    -ref 'mmdg -i {in} -o {out}' -o out/review.html
//
// Renderer odniesienia jest uruchamiany jako osobny program; jego wynik trafia tylko do strony.
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/polestorm/gomermaidgo/fonts"
	"github.com/polestorm/gomermaidgo/mermaid"
)

type entry struct {
	Path, Kind, Title, Checks string
	Reqs                      []string
	Source                    string
	SVG                       string
	Styled                    string // nasz rysunek w trybie kolorowym z cieniami
	Err                       string
	Ref                       string // data URI albo pusty
	RefWidth                  string // szerokość naturalna rysunku odniesienia w px
	Shown                     string // źródło bez linii opisowych korpusu
	RefErr                    string
	Theme                     string // motyw rysunku wyrywkowego
	Themed                    string
}

var viewBoxWidth = regexp.MustCompile(`viewBox="[-\d.]+ [-\d.]+ ([\d.]+) [\d.]+"`)

var reqID = regexp.MustCompile(`\b[A-Z]{3,5}-\d{3}\b`)

func requirements() map[string]string {
	out := map[string]string{}
	docs, _ := filepath.Glob("requirements/*.md")
	for _, doc := range docs {
		data, _ := os.ReadFile(doc)
		for _, line := range strings.Split(string(data), "\n") {
			cells := strings.Split(line, "|")
			if len(cells) >= 3 && reqID.MatchString(strings.TrimSpace(cells[1])) {
				out[strings.TrimSpace(cells[1])] = strings.TrimSpace(cells[2])
			}
		}
	}
	return out
}

// typeKinds przypisuje typom rozpoznanym w źródle nazwy katalogów korpusu, od których zależą nagłówki.
var typeKinds = map[string]string{mermaid.TypeSequence: "sequence", mermaid.TypeFlowchart: "flowchart", mermaid.TypeState: "state",
	mermaid.TypeGantt: "gantt", mermaid.TypeER: "er", mermaid.TypePie: "pie", mermaid.TypeClass: "class", mermaid.TypeTimeline: "timeline",
	mermaid.TypeJourney: "journey", mermaid.TypeMindmap: "mindmap", mermaid.TypeXYChart: "xychart", mermaid.TypeQuadrant: "quadrant",
	mermaid.TypeRequirement: "requirement", mermaid.TypeKanban: "kanban", mermaid.TypeIshikawa: "ishikawa", mermaid.TypePacket: "packet", mermaid.TypeRadar: "radar",
	mermaid.TypeTreemap: "treemap", mermaid.TypeSankey: "sankey", mermaid.TypeGit: "gitgraph", mermaid.TypeBlock: "block",
	mermaid.TypeArchitecture: "architecture", mermaid.TypeC4: "c4", mermaid.TypeTreeView: "treeview", mermaid.TypeVenn: "venn", mermaid.TypeCynefin: "cynefin", mermaid.TypeUseCase: "usecase", mermaid.TypeZenUML: "zenuml", mermaid.TypeWardley: "wardley", mermaid.TypeEventModel: "eventmodeling", mermaid.TypeAgentflow: "agentflow"}

// sampleThemes to motywy pokazywane wyrywkowo; kolorowy (default) ma własną kolumnę.
var sampleThemes = []string{mermaid.ThemeForest, mermaid.ThemeDark}

var kindNames = map[string]string{"sequence": "Diagram sekwencji", "flowchart": "Schemat blokowy", "state": "Diagram stanów", "gantt": "Wykres Gantta", "er": "Diagram związków encji", "pie": "Wykres kołowy", "class": "Diagram klas", "timeline": "Oś czasu", "journey": "Podróż użytkownika", "mindmap": "Mapa myśli", "xychart": "Wykres słupkowy i liniowy", "quadrant": "Wykres ćwiartkowy", "kanban": "Tablica kanban", "packet": "Diagram pakietu", "radar": "Wykres radarowy", "treemap": "Mapa drzewa", "requirement": "Diagram wymagań", "sankey": "Wykres Sankeya", "c4": "Diagram C4", "gitgraph": "Graf historii Gita", "block": "Diagram blokowy", "architecture": "Diagram architektury", "ishikawa": "Diagram Ishikawy",
	"treeview": "Drzewo katalogów", "venn": "Diagram Venna", "cynefin": "Mapa Cynefin", "usecase": "Diagram przypadków użycia",
	"zenuml": "Diagram sekwencji ZenUML", "wardley": "Mapa Wardleya", "eventmodeling": "Model zdarzeń", "agentflow": "Przepływ agentów",
	"railroad": "Diagram składni (railroad)"}

func main() {
	fontPath := flag.String("font", "", "font file used to measure text and embedded in the page")
	family := flag.String("family", "sans-serif", "font-family for our drawings")
	bold := flag.String("bold", "", "bold variant of -font (optional)")
	italic := flag.String("italic", "", "italic variant of -font (optional)")
	note := flag.String("note", "", "a line shown under the page title")
	ref := flag.String("ref", "", "reference renderer command with {in} and {out} (optional)")
	out := flag.String("o", "out/review.html", "output file")
	themes := flag.Bool("themes", false, "also show a sample of diagrams in the forest and dark themes")
	dir := flag.String("dir", "", "draw every .mmd file of this directory instead of the corpus, grouped by detected type")
	kindFlag := flag.String("kind", "", "with -dir: put every file in this one group, also types not drawn yet (e.g. testdata/planned/venn)")
	flag.Parse()

	var opts []mermaid.Option
	// Bez wskazania fontu rysunki używają kroju dołączonego; strona musi go mieć, żeby tekst
	// pasował do pól.
	fontFace := fmt.Sprintf("@font-face{font-family:%q;src:url(data:font/ttf;base64,%s)}", fonts.ArimoFamily, base64.StdEncoding.EncodeToString(fonts.Arimo())) +
		fmt.Sprintf("@font-face{font-family:%q;font-weight:bold;src:url(data:font/ttf;base64,%s)}", fonts.ArimoFamily, base64.StdEncoding.EncodeToString(fonts.ArimoBold())) +
		fmt.Sprintf("@font-face{font-family:%q;font-style:italic;src:url(data:font/ttf;base64,%s)}", fonts.ArimoFamily, base64.StdEncoding.EncodeToString(fonts.ArimoItalic())) +
		fmt.Sprintf("@font-face{font-family:%q;src:url(data:font/ttf;base64,%s)}", fonts.EmojiFamily, base64.StdEncoding.EncodeToString(fonts.NotoEmoji()))
	if *fontPath != "" {
		data, err := os.ReadFile(*fontPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		opts = append(opts, mermaid.WithFont(data), mermaid.WithFontFamily(*family))
		// Nazwa kroju strony: pierwsza pozycja listy font-family, bez cudzysłowów.
		name := strings.Trim(strings.TrimSpace(strings.Split(*family, ",")[0]), `'"`)
		fontFace = fmt.Sprintf("@font-face{font-family:%q;src:url(data:font/ttf;base64,%s)}", name, base64.StdEncoding.EncodeToString(data))
		var styled [2][]byte
		for i, path := range []string{*bold, *italic} {
			if path == "" {
				continue
			}
			if styled[i], err = os.ReadFile(path); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			rule := "font-weight:bold"
			if i == 1 {
				rule = "font-style:italic"
			}
			fontFace += fmt.Sprintf("@font-face{font-family:%q;%s;src:url(data:font/ttf;base64,%s)}", name, rule, base64.StdEncoding.EncodeToString(styled[i]))
		}
		if styled[0] != nil || styled[1] != nil {
			opts = append(opts, mermaid.WithStyledFonts(styled[0], styled[1]))
		}
	}
	tmp, err := os.MkdirTemp("", "review")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	var entries []entry
	themed := 0
	// Grupy rysunków: korpus według katalogów typów albo, z flagą -dir, dowolne pliki .mmd według
	// typu rozpoznanego w źródle.
	type group struct {
		kind  string
		files []string
	}
	var groups []group
	skipped := 0
	if *dir == "" {
		for _, kind := range []string{"sequence", "flowchart", "state", "gantt", "er", "pie", "class", "timeline", "journey", "mindmap", "xychart", "quadrant", "kanban", "packet", "radar", "treemap", "requirement", "sankey", "c4", "gitgraph", "block", "architecture", "ishikawa", "treeview", "venn", "cynefin", "usecase", "zenuml", "wardley", "eventmodeling", "agentflow"} {
			files, _ := filepath.Glob("testdata/corpus/" + kind + "/*.mmd")
			sort.Strings(files)
			groups = append(groups, group{kind, files})
		}
	} else {
		files, _ := filepath.Glob(filepath.Join(*dir, "*.mmd"))
		sort.Strings(files)
		byKind := map[string][]string{}
		var kinds []string
		if *kindFlag != "" {
			// typ planowany: biblioteka go jeszcze nie rozpoznaje, więc grupy nie wyznaczamy ze źródła
			byKind[*kindFlag], kinds, files = files, []string{*kindFlag}, nil
		}
		for _, f := range files {
			data, _ := os.ReadFile(f)
			kind, err := mermaid.DetectType(string(data))
			if err != nil {
				skipped++ // bez nagłówka typu: fragment, którego nie narysuje żaden renderer
				continue
			}
			if alias, ok := typeKinds[kind]; ok {
				kind = alias
			}
			if byKind[kind] == nil {
				kinds = append(kinds, kind)
			}
			byKind[kind] = append(byKind[kind], f)
		}
		// Najliczniejsze typy najpierw.
		sort.SliceStable(kinds, func(a, b int) bool { return len(byKind[kinds[a]]) > len(byKind[kinds[b]]) })
		for _, k := range kinds {
			groups = append(groups, group{k, byKind[k]})
		}
	}
	for _, g := range groups {
		kind, files := g.kind, g.files
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			e := entry{Path: f, Kind: kind, Source: string(data)}
			var shown []string
			for _, line := range strings.Split(strings.TrimRight(e.Source, "\n"), "\n") {
				if !strings.HasPrefix(line, "%% REQ:") && !strings.HasPrefix(line, "%% Tytuł:") && !strings.HasPrefix(line, "%% Sprawdza:") {
					shown = append(shown, line)
				}
				switch {
				case strings.HasPrefix(line, "%% REQ:"):
					e.Reqs = reqID.FindAllString(line, -1)
				case strings.HasPrefix(line, "%% Tytuł:"):
					e.Title = strings.TrimSpace(strings.TrimPrefix(line, "%% Tytuł:"))
				case strings.HasPrefix(line, "%% Sprawdza:"):
					e.Checks = strings.TrimSpace(strings.TrimPrefix(line, "%% Sprawdza:"))
				}
			}
			e.Shown = strings.Join(shown, "\n")
			if e.Title == "" {
				e.Title = filepath.Base(f)
			}
			if svg, err := mermaid.Render(e.Source, opts...); err != nil {
				e.Err = err.Error()
			} else {
				e.SVG = string(svg)
				styled, _ := mermaid.Render(e.Source, append(opts[:len(opts):len(opts)], mermaid.WithColors(), mermaid.WithShadows())...)
				e.Styled = string(styled)
				// Z flagą -themes, wyrywkowo: pierwszy diagram każdego typu i co czwarty z kolei dostają rysunek w jednym
				// z motywów, na zmianę, żeby dało się je ocenić bez osobnej strony.
				if first := len(entries) == 0 || entries[len(entries)-1].Kind != kind; *themes && (first || len(entries)%4 == 0) {
					e.Theme = sampleThemes[themed%len(sampleThemes)]
					themed++
					if drawn, err := mermaid.Render(e.Source, append(opts[:len(opts):len(opts)], mermaid.WithTheme(e.Theme))...); err == nil {
						e.Themed = string(drawn)
					}
				}
			}
			if *ref != "" {
				target := filepath.Join(tmp, kind+"-"+filepath.Base(f)+".svg")
				var args []string
				for _, a := range strings.Fields(*ref) {
					args = append(args, strings.NewReplacer("{in}", f, "{out}", target).Replace(a))
				}
				if msg, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
					e.RefErr = strings.TrimSpace(err.Error() + " " + string(msg))
				} else if svg, err := os.ReadFile(target); err != nil {
					e.RefErr = err.Error()
				} else {
					e.Ref = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(svg)
					if m := viewBoxWidth.FindSubmatch(svg); m != nil {
						e.RefWidth = string(m[1])
					}
				}
			}
			entries = append(entries, e)
		}
	}

	reqs := requirements()
	var b strings.Builder
	fmt.Fprintf(&b, pageHead, fontFace, time.Now().Format("2006-01-02 15:04"), len(entries))
	if skipped > 0 {
		*note += fmt.Sprintf(" Pominięto %d bloków bez nagłówka typu (fragmenty, których nie rysuje żaden renderer).", skipped)
	}
	if *note != "" {
		fmt.Fprintf(&b, "<p class=\"checks\" style=\"margin:8px 16px\"><b>%s</b></p>\n", html.EscapeString(*note))
	}
	current := ""
	for i, e := range entries {
		if e.Kind != current {
			current = e.Kind
			name := kindNames[e.Kind]
			if name == "" {
				name = e.Kind
			}
			fmt.Fprintf(&b, "<h2 id=%q>%s</h2>\n", e.Kind, html.EscapeString(name))
		}
		id := e.Kind + "/" + strings.TrimSuffix(filepath.Base(e.Path), ".mmd")
		fmt.Fprintf(&b, "<article class=\"card\" data-id=%q>\n<header><span class=\"num\">%d</span><h3>%s</h3><code class=\"path\">%s</code></header>\n",
			id, i+1, html.EscapeString(e.Title), html.EscapeString(e.Path))
		fmt.Fprintf(&b, "<p class=\"checks\">%s</p>\n<ul class=\"reqs\">", html.EscapeString(e.Checks))
		for _, r := range e.Reqs {
			fmt.Fprintf(&b, "<li><b>%s</b> %s</li>", r, html.EscapeString(strings.ReplaceAll(reqs[r], "`", "")))
		}
		b.WriteString("</ul>\n<div class=\"cols\">\n")
		fmt.Fprintf(&b, "<section><h4>Źródło</h4><pre>%s</pre></section>\n", html.EscapeString(e.Shown))
		b.WriteString("<section><h4>gomermaidgo</h4><div class=\"draw\">")
		if e.Err != "" {
			fmt.Fprintf(&b, "<p class=\"error\">%s</p>", html.EscapeString(e.Err))
		} else {
			b.WriteString(e.SVG)
		}
		b.WriteString("</div></section>\n")
		if e.Styled != "" {
			b.WriteString("<section><h4>gomermaidgo, kolory i cienie (opcje)</h4><div class=\"draw\">" + e.Styled + "</div></section>\n")
		}
		if e.Themed != "" {
			b.WriteString("<section><h4>gomermaidgo, motyw: " + e.Theme + "</h4><div class=\"draw\">" + e.Themed + "</div></section>\n")
		}
		if *ref != "" {
			b.WriteString("<section><h4>Odniesienie</h4><div class=\"draw\">")
			if e.RefErr != "" {
				fmt.Fprintf(&b, "<p class=\"error\">%s</p>", html.EscapeString(e.RefErr))
			} else {
				fmt.Fprintf(&b, "<img alt=\"rysunek renderera odniesienia\" width=%q src=%q>", e.RefWidth, e.Ref)
			}
			b.WriteString("</div></section>\n")
		}
		b.WriteString("</div>\n<footer><label><input type=\"radio\" name=\"v-" + id + "\" value=\"ok\"> OK</label>" +
			"<label><input type=\"radio\" name=\"v-" + id + "\" value=\"fix\"> Do poprawy</label>" +
			"<textarea rows=\"1\" placeholder=\"Uwagi do tego rysunku\"></textarea></footer>\n</article>\n")
	}
	b.WriteString(pageTail)
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d diagrams, %.1f MB\n", *out, len(entries), float64(b.Len())/1e6)
}

const pageHead = `<!doctype html>
<html lang="pl">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gomermaidgo: przegląd korpusu</title>
<style>
%s
:root { --ink:#1b1f24; --muted:#5d6875; --line:#d5dae1; --paper:#ffffff; --desk:#f3f5f8; --ok:#1f7a4d; --fix:#b3421c; --accent:#2f5d8a; }
* { box-sizing: border-box; }
body { margin:0; background:var(--desk); color:var(--ink); font:15px/1.5 system-ui, -apple-system, "Segoe UI", sans-serif; }
.top { position:sticky; top:0; z-index:5; background:var(--paper); border-bottom:1px solid var(--line); padding:10px 20px; display:flex; flex-wrap:wrap; gap:10px 18px; align-items:center; }
.top h1 { font-size:17px; margin:0 8px 0 0; }
.top .meta { color:var(--muted); font-size:13px; }
.top nav a { color:var(--accent); margin-right:10px; }
.top .tally { font-variant-numeric:tabular-nums; }
.top button, .top select { font:inherit; padding:5px 12px; border:1px solid var(--line); border-radius:6px; background:var(--paper); color:var(--ink); cursor:pointer; }
.top button.primary { background:var(--accent); border-color:var(--accent); color:#fff; }
main { max-width:1700px; margin:0 auto; padding:8px 20px 80px; }
h2 { font-size:20px; margin:28px 0 10px; }
.card { background:var(--paper); border:1px solid var(--line); border-left:5px solid var(--line); border-radius:8px; margin:0 0 18px; padding:14px 18px; }
.card.ok { border-left-color:var(--ok); }
.card.fix { border-left-color:var(--fix); }
.card header { display:flex; flex-wrap:wrap; align-items:baseline; gap:10px; }
.card h3 { font-size:17px; margin:0; }
.num { color:var(--muted); font-variant-numeric:tabular-nums; }
.path { color:var(--muted); font-size:12.5px; }
.checks { margin:6px 0 4px; max-width:95ch; }
.reqs { margin:0 0 10px; padding-left:18px; color:var(--muted); font-size:13px; max-width:120ch; }
.cols { display:grid; grid-template-columns:repeat(2, minmax(0, 1fr)); gap:14px; align-items:start; }
@media (max-width: 1100px) { .cols { grid-template-columns:1fr; } }
h4 { font-size:12px; text-transform:uppercase; letter-spacing:.05em; color:var(--muted); margin:0 0 4px; font-weight:600; }
pre { margin:0; padding:10px 12px; background:#f7f8fa; border:1px solid var(--line); border-radius:6px; font:12.5px/1.45 ui-monospace, SFMono-Regular, Menlo, monospace; overflow:auto; max-height:520px; white-space:pre; }
.draw { border:1px solid var(--line); border-radius:6px; padding:8px; background:#fff; overflow:auto; }
.draw svg, .draw img { max-width:100%%; height:auto; display:block; margin:0 auto; }
body.actual .draw svg, body.actual .draw img { max-width:none; }
.error { color:var(--fix); margin:0; }
.card footer { display:flex; flex-wrap:wrap; gap:8px 16px; align-items:center; margin-top:12px; padding-top:10px; border-top:1px solid var(--line); }
.card footer label { cursor:pointer; }
.card footer textarea { flex:1 1 320px; font:inherit; padding:5px 8px; border:1px solid var(--line); border-radius:6px; resize:vertical; min-height:32px; }
body.only-open .card.ok { display:none; }
#copied { color:var(--ok); font-size:13px; }
</style>
</head>
<body>
<div class="top">
  <h1>gomermaidgo: przegląd korpusu</h1>
  <span class="meta">wygenerowano %s · %d rysunków</span>
  <nav><a href="#sequence">sekwencje</a><a href="#flowchart">schematy</a><a href="#state">stany</a></nav>
  <span class="tally" id="tally"></span>
  <label><input type="checkbox" id="onlyOpen"> ukryj zaakceptowane</label>
  <label><input type="checkbox" id="actual"> rozmiar rzeczywisty</label>
  <button class="primary" id="copy">Kopiuj oceny i uwagi</button>
  <button id="reset">Wyczyść oceny</button>
  <span id="copied"></span>
</div>
<main>
`

const pageTail = `</main>
<script>
(function () {
  var KEY = 'gomermaidgo-review';
  var state = {};
  try { state = JSON.parse(localStorage.getItem(KEY) || '{}'); } catch (e) {}
  var cards = Array.prototype.slice.call(document.querySelectorAll('.card'));
  function save() { try { localStorage.setItem(KEY, JSON.stringify(state)); } catch (e) {} }
  function tally() {
    var ok = 0, fix = 0;
    cards.forEach(function (c) {
      var s = state[c.dataset.id] || {};
      c.classList.toggle('ok', s.verdict === 'ok');
      c.classList.toggle('fix', s.verdict === 'fix');
      if (s.verdict === 'ok') ok++;
      if (s.verdict === 'fix') fix++;
    });
    document.getElementById('tally').textContent = 'OK ' + ok + ' · do poprawy ' + fix + ' · bez oceny ' + (cards.length - ok - fix);
  }
  cards.forEach(function (c) {
    var id = c.dataset.id, s = state[id] || {};
    var note = c.querySelector('textarea');
    note.value = s.note || '';
    c.querySelectorAll('input[type=radio]').forEach(function (r) {
      r.checked = s.verdict === r.value;
      r.addEventListener('change', function () { (state[id] = state[id] || {}).verdict = r.value; save(); tally(); });
    });
    note.addEventListener('input', function () { (state[id] = state[id] || {}).note = note.value; save(); });
  });
  document.getElementById('onlyOpen').addEventListener('change', function (e) { document.body.classList.toggle('only-open', e.target.checked); });
  document.getElementById('actual').addEventListener('change', function (e) { document.body.classList.toggle('actual', e.target.checked); });
  document.getElementById('reset').addEventListener('click', function () {
    state = {}; save();
    cards.forEach(function (c) { c.querySelector('textarea').value = ''; c.querySelectorAll('input[type=radio]').forEach(function (r) { r.checked = false; }); });
    tally();
  });
  document.getElementById('copy').addEventListener('click', function () {
    var lines = [];
    cards.forEach(function (c) {
      var s = state[c.dataset.id] || {};
      var verdict = s.verdict === 'ok' ? 'OK' : s.verdict === 'fix' ? 'DO POPRAWY' : 'bez oceny';
      lines.push(c.dataset.id + ': ' + verdict + (s.note ? ' — ' + s.note.trim() : ''));
    });
    var text = lines.join('\n');
    var done = function () { document.getElementById('copied').textContent = 'skopiowano ' + lines.length + ' pozycji'; };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done, function () { window.prompt('Skopiuj:', text); });
    } else { window.prompt('Skopiuj:', text); }
  });
  tally();
})();
</script>
</body>
</html>
`
