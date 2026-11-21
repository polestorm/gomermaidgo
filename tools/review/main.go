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

	"github.com/polestorm/gomermaidgo"
	"github.com/polestorm/gomermaidgo/fonts"
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

var kindNames = map[string]string{"sequence": "Diagram sekwencji", "flowchart": "Schemat blokowy", "state": "Diagram stanów", "gantt": "Wykres Gantta", "er": "Diagram związków encji", "pie": "Wykres kołowy", "class": "Diagram klas"}

func main() {
	fontPath := flag.String("font", "", "font file used to measure text and embedded in the page")
	family := flag.String("family", "sans-serif", "font-family for our drawings")
	ref := flag.String("ref", "", "reference renderer command with {in} and {out} (optional)")
	out := flag.String("o", "out/review.html", "output file")
	flag.Parse()

	var opts []gomermaidgo.Option
	// Bez wskazania fontu rysunki używają kroju dołączonego; strona musi go mieć, żeby tekst
	// pasował do pól.
	fontFace := fmt.Sprintf("@font-face{font-family:%q;src:url(data:font/ttf;base64,%s)}", fonts.RecursiveFamily, base64.StdEncoding.EncodeToString(fonts.Recursive())) +
		fmt.Sprintf("@font-face{font-family:%q;font-weight:bold;src:url(data:font/ttf;base64,%s)}", fonts.RecursiveFamily, base64.StdEncoding.EncodeToString(fonts.RecursiveBold())) +
		fmt.Sprintf("@font-face{font-family:%q;font-style:italic;src:url(data:font/ttf;base64,%s)}", fonts.RecursiveFamily, base64.StdEncoding.EncodeToString(fonts.RecursiveItalic()))
	if *fontPath != "" {
		data, err := os.ReadFile(*fontPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		opts = append(opts, gomermaidgo.WithFont(data), gomermaidgo.WithFontFamily(*family))
		fontFace = fmt.Sprintf("@font-face{font-family:%q;src:url(data:font/ttf;base64,%s)}", *family, base64.StdEncoding.EncodeToString(data))
	}
	tmp, err := os.MkdirTemp("", "review")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	var entries []entry
	for _, kind := range []string{"sequence", "flowchart", "state", "gantt", "er", "pie", "class"} {
		files, _ := filepath.Glob("testdata/corpus/" + kind + "/*.mmd")
		sort.Strings(files)
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
			if svg, err := gomermaidgo.Render(e.Source, opts...); err != nil {
				e.Err = err.Error()
			} else {
				e.SVG = string(svg)
				styled, _ := gomermaidgo.Render(e.Source, append(opts[:len(opts):len(opts)], gomermaidgo.WithColors(), gomermaidgo.WithShadows())...)
				e.Styled = string(styled)
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
	current := ""
	for i, e := range entries {
		if e.Kind != current {
			current = e.Kind
			fmt.Fprintf(&b, "<h2 id=%q>%s</h2>\n", e.Kind, kindNames[e.Kind])
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
