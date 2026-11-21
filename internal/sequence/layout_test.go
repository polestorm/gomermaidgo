package sequence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
)

func corpusScenes(t *testing.T) map[string]*scene.Scene {
	t.Helper()
	files, _ := filepath.Glob("../../testdata/corpus/sequence/*.mmd")
	if len(files) == 0 {
		t.Fatal("no corpus files")
	}
	out := map[string]*scene.Scene{}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		d, err := Parse(string(src))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out[filepath.Base(f)] = Layout(d, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	}
	return out
}

func overlap(a, b scene.Box) bool {
	const eps = 0.5
	return a.X0 < b.X1-eps && b.X0 < a.X1-eps && a.Y0 < b.Y1-eps && b.Y0 < a.Y1-eps
}

// REQ: SEQ-009
// Żadne dwa teksty nie nachodzą na siebie.
func TestTextsDoNotOverlap(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		var texts []scene.Item
		for _, it := range sc.Items {
			if it.Kind == scene.Text {
				texts = append(texts, it)
			}
		}
		for i := range texts {
			for j := i + 1; j < len(texts); j++ {
				if overlap(texts[i].Bounds(), texts[j].Bounds()) {
					t.Errorf("%s: %q (%s) overlaps %q (%s)", name, texts[i].Text, texts[i].Role, texts[j].Text, texts[j].Role)
				}
			}
		}
	}
}

// REQ: SEQ-009
// Etykiety komunikatów i notatek nie nachodzą na nagłówki uczestników, a nagłówki na siebie.
func TestShapesDoNotCollide(t *testing.T) {
	for name, sc := range corpusScenes(t) {
		var boxes, labels []scene.Item
		for _, it := range sc.Items {
			switch it.Role {
			case "participant":
				boxes = append(boxes, it)
			case "message-label", "note", "frame-title", "frame-tab":
				labels = append(labels, it)
			}
		}
		for i := range boxes {
			for j := i + 1; j < len(boxes); j++ {
				if overlap(boxes[i].Bounds(), boxes[j].Bounds()) {
					t.Errorf("%s: participant boxes overlap", name)
				}
			}
			for _, lb := range labels {
				if overlap(boxes[i].Bounds(), lb.Bounds()) {
					t.Errorf("%s: %s %q overlaps a participant box", name, lb.Role, lb.Text)
				}
			}
		}
	}
}

// REQ: SEQ-005 SEQ-007 SEQ-008
// Scena zawiera kształty, których wymaga źródło.
func TestSceneContents(t *testing.T) {
	count := func(sc *scene.Scene, role string, kind scene.Kind) int {
		n := 0
		for _, it := range sc.Items {
			if it.Role == role && it.Kind == kind {
				n++
			}
		}
		return n
	}
	scenes := corpusScenes(t)
	if n := count(scenes["005-activations.mmd"], "activation", scene.Rect); n != 4 {
		t.Errorf("activations: got %d bars, want 4", n)
	}
	if n := count(scenes["007-nested-blocks.mmd"], "highlight", scene.Rect); n != 2 {
		t.Errorf("nested blocks: got %d highlights, want 2", n)
	}
	if n := count(scenes["007-nested-blocks.mmd"], "frame", scene.Rect); n != 4 {
		t.Errorf("nested blocks: got %d frames, want 4", n)
	}
	if n := count(scenes["008-autonumber.mmd"], "sequence-number", scene.Text); n != 5 {
		t.Errorf("autonumber: got %d numbers, want 5", n)
	}
	if n := count(scenes["003-arrow-types.mmd"], "message", scene.Line); n != 11 {
		t.Errorf("arrow types: got %d message lines, want 11", n)
	}
}

// REQ: SEQ-007
func TestBlend(t *testing.T) {
	cases := []struct{ top, under, want string }{
		{"rgba(200,220,255,0.4)", "#ffffff", "rgb(233,241,255)"},
		{"rgb(232,244,234)", "#ffffff", "rgb(232,244,234)"},
		{"#f00", "#ffffff", "rgb(255,0,0)"},
		{"nonsense", "#ffffff", "#ffffff"},
	}
	for _, c := range cases {
		if got := blend(c.top, c.under); got != c.want {
			t.Errorf("blend(%q, %q) = %q, want %q", c.top, c.under, got, c.want)
		}
	}
}
