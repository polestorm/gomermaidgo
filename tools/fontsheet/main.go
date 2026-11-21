// Polecenie fontsheet składa stronę HTML porównującą kroje pisma na tych samych diagramach.
// Każdy krój jest osadzony w stronie i tym samym plikiem mierzony przez renderer, więc rysunek
// pokazuje dokładnie to, co dałby dokument z tym krojem.
//
//	go run ./tools/fontsheet -o out/fonts.html "Inter=out/fonts/Inter-Variable.ttf" "Noto Sans=…"
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"github.com/polestorm/gomermaidgo"
)

var diagrams = []string{
	"testdata/corpus/sequence/006-blocks.mmd",
	"testdata/corpus/flowchart/011-capa-process.mmd",
	"testdata/corpus/state/004-choice-fork-join.mmd",
}

const sample = "Zażółć gęślą jaźń. Pchnąć w tę łódź jeża lub ośm skrzyń fig. 0123456789 „cudzysłów” – półpauza § 7.5.1"

func main() {
	out := flag.String("o", "out/fonts.html", "output file")
	sizes := flag.String("sizes", "16,14", "font sizes in px for the diagrams")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: fontsheet [-o file] \"Name=font.ttf\"...")
		os.Exit(2)
	}
	var px []float64
	for _, s := range strings.Split(*sizes, ",") {
		var v float64
		fmt.Sscanf(strings.TrimSpace(s), "%g", &v)
		px = append(px, v)
	}

	var faces, body, nav strings.Builder
	for i, arg := range flag.Args() {
		name, path, ok := strings.Cut(arg, "=")
		if !ok {
			fmt.Fprintf(os.Stderr, "expected Name=path, got %q\n", arg)
			os.Exit(2)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		family := fmt.Sprintf("sheet-%d", i)
		fmt.Fprintf(&faces, "@font-face{font-family:%q;src:url(data:font/ttf;base64,%s)}\n", family, base64.StdEncoding.EncodeToString(data))
		fmt.Fprintf(&nav, "<a href=\"#f%d\">%s</a>", i, html.EscapeString(name))
		fmt.Fprintf(&body, "<article class=\"card\" id=\"f%d\" data-id=%q>\n<header><h2>%s</h2><code>%s · %d kB</code>"+
			"<label><input type=\"radio\" name=\"pick\" value=%q> wybieram ten krój</label></header>\n", i, name, html.EscapeString(name), html.EscapeString(path), len(data)/1024, name)
		for _, size := range []int{20, 16, 14, 12} {
			fmt.Fprintf(&body, "<p class=\"sample\" style=\"font-family:'%s';font-size:%dpx\"><span>%d px</span>%s</p>\n", family, size, size, html.EscapeString(sample))
		}
		for _, size := range px {
			fmt.Fprintf(&body, "<h3>Diagramy, pismo %g px</h3>\n<div class=\"row\">\n", size)
			for _, d := range diagrams {
				src, err := os.ReadFile(d)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				svg, err := gomermaidgo.Render(string(src), gomermaidgo.WithFont(data), gomermaidgo.WithFontFamily(family), gomermaidgo.WithFontSize(size))
				if err != nil {
					fmt.Fprintf(&body, "<div class=\"draw\"><p class=\"error\">%s</p></div>\n", html.EscapeString(err.Error()))
					continue
				}
				fmt.Fprintf(&body, "<div class=\"draw\">%s</div>\n", svg)
			}
			body.WriteString("</div>\n")
		}
		body.WriteString("</article>\n")
	}
	page := fmt.Sprintf(pageHTML, faces.String(), time.Now().Format("2006-01-02 15:04"), nav.String(), body.String())
	if err := os.WriteFile(*out, []byte(page), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d fonts, %.1f MB\n", *out, flag.NArg(), float64(len(page))/1e6)
}

const pageHTML = `<!doctype html>
<html lang="pl">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gomermaidgo: kroje pisma</title>
<style>
%s
:root { --ink:#1b1f24; --muted:#5d6875; --line:#d5dae1; --paper:#ffffff; --desk:#f3f5f8; --accent:#2f5d8a; }
* { box-sizing:border-box; }
body { margin:0; background:var(--desk); color:var(--ink); font:15px/1.5 system-ui, -apple-system, "Segoe UI", sans-serif; font-optical-sizing:none; }
.top { position:sticky; top:0; z-index:5; background:var(--paper); border-bottom:1px solid var(--line); padding:10px 20px; display:flex; flex-wrap:wrap; gap:8px 18px; align-items:baseline; }
.top h1 { font-size:17px; margin:0; }
.top .meta { color:var(--muted); font-size:13px; }
.top nav a { color:var(--accent); margin-right:12px; }
#picked { font-weight:600; }
main { max-width:1700px; margin:0 auto; padding:8px 20px 80px; }
.intro { max-width:95ch; color:var(--muted); }
.card { background:var(--paper); border:1px solid var(--line); border-left:5px solid var(--line); border-radius:8px; margin:0 0 22px; padding:16px 20px; scroll-margin-top:60px; }
.card.picked { border-left-color:var(--accent); }
.card header { display:flex; flex-wrap:wrap; gap:8px 16px; align-items:baseline; margin-bottom:8px; }
.card h2 { font-size:20px; margin:0; }
.card code { color:var(--muted); font-size:12.5px; }
.card header label { margin-left:auto; cursor:pointer; }
.sample { margin:4px 0; line-height:1.35; overflow-wrap:anywhere; }
.sample span { display:inline-block; width:52px; color:var(--muted); font:12px system-ui, sans-serif; }
h3 { font-size:12px; text-transform:uppercase; letter-spacing:.05em; color:var(--muted); margin:16px 0 6px; font-weight:600; }
.row { display:grid; grid-template-columns:repeat(3, minmax(0, 1fr)); gap:12px; align-items:start; }
@media (max-width: 1200px) { .row { grid-template-columns:1fr; } }
.draw { border:1px solid var(--line); border-radius:6px; padding:8px; background:#fff; overflow:auto; }
.draw svg { max-width:100%%; height:auto; display:block; margin:0 auto; }
.error { color:#b3421c; }
</style>
</head>
<body>
<div class="top">
  <h1>gomermaidgo: kroje pisma</h1>
  <span class="meta">wygenerowano %s</span>
  <nav>%s</nav>
  <span id="picked"></span>
</div>
<main>
<p class="intro">Każdy krój jest osadzony w tej stronie i tym samym plikiem zmierzony przez renderer, więc szerokości pól
zgadzają się z tekstem. Rysunki są pomniejszane do szerokości kolumny; rozmiar pisma podany przy wierszu to rozmiar
przed pomniejszeniem. Wybór zapisuje się w przeglądarce.</p>
%s
</main>
<script>
(function () {
  var KEY = 'gomermaidgo-font';
  var picked = null;
  try { picked = localStorage.getItem(KEY); } catch (e) {}
  function show() {
    document.querySelectorAll('.card').forEach(function (c) {
      var on = c.dataset.id === picked;
      c.classList.toggle('picked', on);
      c.querySelector('input[type=radio]').checked = on;
    });
    document.getElementById('picked').textContent = picked ? 'Wybrany: ' + picked : '';
  }
  document.querySelectorAll('input[name=pick]').forEach(function (r) {
    r.addEventListener('change', function () { picked = r.value; try { localStorage.setItem(KEY, picked); } catch (e) {} show(); });
  });
  show();
})();
</script>
</body>
</html>
`
