package markdown

import (
	"strings"
	"testing"
)

// REQ: SEQ-018
func TestEmbed(t *testing.T) {
	src := "# Procedura\n\nTekst.\n\n```mermaid\nflowchart TD\n    A[Start] --> B[Koniec]\n```\n\n" +
		"~~~~ mermaid tytuł\nsequenceDiagram\n    A->>B: Zażółć\n~~~~\n\n" +
		"```go\nfmt.Println(\"```mermaid\")\n```\n\n" +
		"````markdown\n```mermaid\nflowchart TD\n    X --> Y\n```\n````\n\n" +
		"```mermaid\nflowchart TD\n\n    A -->\n```\n\nKoniec.\n\n```mermaid\nflowchart TD\n  niezamknięty\n"
	out, problems := Embed(src, Options{})
	if n := strings.Count(out, `<figure class="diagram">`); n != 2 {
		t.Errorf("want 2 figures, got %d", n)
	}
	if !strings.Contains(out, ">Zażółć</text>") || !strings.Contains(out, ">Koniec</text>") {
		t.Errorf("diagram texts are missing")
	}
	// Bloki kodu innych języków i blok mermaid zagnieżdżony w bloku kodu zostają bez zmian.
	for _, keep := range []string{"```go\nfmt.Println(\"```mermaid\")\n```", "````markdown\n```mermaid\nflowchart TD\n    X --> Y\n```\n````", "```mermaid\nflowchart TD\n  niezamknięty\n"} {
		if !strings.Contains(out, keep) {
			t.Errorf("block should be left untouched: %q", keep)
		}
	}
	if len(problems) != 1 || problems[0].Line != 29 {
		t.Fatalf("problems: %+v", problems)
	}
	if !strings.Contains(out, `<pre class="diagram-error"><strong>Diagram error (line 29): `) || !strings.Contains(out, "A --&gt;</pre>") {
		t.Errorf("the failed block should stay as escaped source with a message")
	}
	// Blok HTML kończy się na pustej linii, więc rysunek nie może jej zawierać.
	start := strings.Index(out, `<figure class="diagram">`)
	end := strings.Index(out[start:], "</figure>")
	if strings.Contains(out[start:start+end], "\n\n") {
		t.Errorf("figure contains a blank line; a Markdown parser would end the HTML block there")
	}
	if !strings.HasPrefix(out, "# Procedura\n\nTekst.\n\n\n<figure") || !strings.Contains(out, "\nKoniec.\n") {
		t.Errorf("surrounding text changed")
	}
	if same, p := Embed("Bez diagramów.\n", Options{}); same != "Bez diagramów.\n" || p != nil {
		t.Errorf("a document without diagrams must come back unchanged")
	}
}
