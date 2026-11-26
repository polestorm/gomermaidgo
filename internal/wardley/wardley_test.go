package wardley

import (
	"math"
	"strings"
	"testing"

	"github.com/polestorm/gomermaidgo/internal/font"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/syntax"
)

// REQ: WARD-001 WARD-002 WARD-003 WARD-004
func TestParse(t *testing.T) {
	m, err := Parse(`wardley-beta
title Mapa
size [1000, 600]
accTitle: Opis
evolution Genesis / Idea@0.2 -> Custom -> Product@0.8 -> Commodity
anchor User [0.95, 0.5]
component API [0.7, 0.6] label [5, -5] (build) (inertia)  %% komentarz
component "Custom Service" [0.4, 0.3] (buy)
component real-time processing [0.2, 0.8]
pipeline API {
  component "v1" [0.5]
  component "v2" [0.7]
}
User -> API
API -.-> "Custom Service"
"Custom Service" +'dane'> real-time processing
API -> real-time processing; wywołanie
API +<> "Custom Service"
evolve API 0.85
API -.- (0.9, 0.75)
note "Uwaga" [0.5, 0.5]
annotations [0.1, 0.1]
annotation 1,[0.7, 0.6] "Pierwsza"
accelerator "Chmura" [0.3, 0.5]`)
	if err != nil {
		t.Fatal(err)
	}
	if m.Title != "Mapa" || m.Width != 1000 || m.AccTitle != "Opis" || len(m.Stages) != 4 || m.Stages[0].Until != 0.2 || strings.Join(m.Stages[0].Names, "|") != "Genesis|Idea" || m.Stages[1].Until != -1 {
		t.Errorf("map: %+v", m)
	}
	api := m.Component("API")
	if api == nil || api.At != (Pos{0.7, 0.6}) || api.LabelDX != 5 || strings.Join(api.Decorators, ",") != "build,inertia" {
		t.Errorf("API: %+v", api)
	}
	if c := m.Component("real-time processing"); c == nil || c.At.Evo != 0.8 {
		t.Errorf("unquoted name with spaces: %+v", c)
	}
	if v2 := m.Component("v2"); v2 == nil || v2.Pipeline != "API" || v2.At != (Pos{0.7, 0.7}) {
		t.Errorf("pipeline: %+v", v2)
	}
	want := []Link{
		{From: "User", To: "API"},
		{From: "API", To: "Custom Service", Dashed: true, EndArrow: true},
		{From: "Custom Service", To: "real-time processing", EndArrow: true, Label: "dane"},
		{From: "API", To: "real-time processing", Label: "wywołanie"},
		{From: "API", To: "Custom Service", StartArrow: true, EndArrow: true},
	}
	for i, w := range want {
		l := m.Links[i]
		l.Line = 0
		if l != w {
			t.Errorf("link %d: %+v, want %+v", i, l, w)
		}
	}
	if len(m.Evolves) != 1 || m.Evolves[0].To != 0.85 || len(m.Trends) != 1 || m.Trends[0].At != (Pos{0.75, 0.9}) {
		t.Errorf("evolve %+v, trends %+v", m.Evolves, m.Trends)
	}
	if len(m.Texts) != 3 || m.Texts[1].Number != 1 || m.Annotations == nil {
		t.Errorf("texts %+v", m.Texts)
	}
}

// REQ: WARD-005
func TestParseErrors(t *testing.T) {
	for src, want := range map[string]struct {
		code string
		line int
	}{
		"":                                     {CodeHeader, 0},
		"wardley-map":                          {CodeHeader, 1},
		"wardley-beta\ncomponent A [1.2, 0.5]": {CodeStatement, 2},
		"wardley-beta\ncomponent A [0.5]":      {CodeStatement, 2},
		"wardley-beta\ncomponent A [0.2, 0.5]\ncomponent A [0.3, 0.5]":            {CodeStatement, 3},
		"wardley-beta\ncomponent A [0.2, 0.5]\nA -> B":                            {CodeStatement, 3},
		"wardley-beta\nevolve A 0.5":                                              {CodeStatement, 2},
		"wardley-beta\ncomponent A [0.2, 0.5] (rent)":                             {CodeStatement, 2},
		"wardley-beta\ncomponent A [0.2, 0.5]\npipeline A {\n  component B [0.3]": {CodeStatement, 0},
		"wardley-beta\nnote Uwaga [0.5, 0.5]":                                     {CodeStatement, 2},
		"wardley-beta\nfoo bar":                                                   {CodeStatement, 2},
	} {
		_, err := Parse(src)
		e, ok := err.(*syntax.Error)
		if !ok || e.Code != want.code || e.Line != want.line {
			t.Errorf("%.40q: got %v, want code %s at line %d", src, err, want.code, want.line)
		}
	}
}

// REQ: WARD-005
func TestLayout(t *testing.T) {
	m, err := Parse("wardley-beta\nanchor U [0.9, 0.5]\ncomponent A [0.6, 0.25] (build)\ncomponent B [0.2, 0.75]\nU -> A\nA +> B\nevolve A 0.6")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var axis, stages, comps int
	var a, b scene.Item
	for _, it := range sc.Items {
		switch it.Role {
		case "wardley-axis":
			axis++
		case "wardley-stage-label":
			stages++
		case "wardley-component":
			comps++
			if comps == 1 {
				a = it
			} else {
				b = it
			}
		}
	}
	if axis != 2 || stages != 6 || comps != 2 {
		t.Fatalf("axes %d, stage labels %d, components %d", axis, stages, comps)
	}
	// Większa widoczność wyżej, większa ewolucja dalej w prawo.
	if !(a.Y < b.Y && a.X < b.X) {
		t.Errorf("A at %v,%v; B at %v,%v", a.X, a.Y, b.X, b.Y)
	}
	if !strings.Contains(string(sc.SVG()), "Commodity") {
		t.Error("default stages missing")
	}
}

// REQ: WARD-005
// Podpis składnika nie wchodzi na linię, gdy obok kółka jest wolne miejsce; tryb kolorowy barwi etapy.
func TestLabelsAvoidLinesAndBands(t *testing.T) {
	m, err := Parse("wardley-beta\ncomponent A [0.5, 0.3]\ncomponent B [0.6, 0.6]\nA -> B")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, Options{Measure: font.ApproxTextWidth, FontSize: 16, Colors: true})
	var link scene.Item
	bands := 0
	for _, it := range sc.Items {
		switch it.Role {
		case "wardley-link":
			link = it
		case "wardley-stage-band":
			bands++
		}
	}
	if bands != 4 {
		t.Errorf("stage bands: %d, want 4", bands)
	}
	for _, it := range sc.Items {
		if it.Role != "wardley-component-label" {
			continue
		}
		b := it.Bounds()
		for k := 0; k <= 50; k++ {
			f := float64(k) / 50
			x, y := link.X+(link.X2-link.X)*f, link.Y+(link.Y2-link.Y)*f
			if x > b.X0 && x < b.X1 && y > b.Y0 && y < b.Y1 {
				t.Errorf("label %q lies on the link", it.Text)
				break
			}
		}
	}
}

// REQ: WARD-005
// Podpis składnika przy prawej krawędzi planszy nie wychodzi poza nią.
func TestLabelStaysOnMap(t *testing.T) {
	m, err := Parse("wardley-beta\ncomponent Energia [0.1, 0.95]")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	right := 0.0
	for _, it := range sc.Items {
		if it.Role == "wardley-axis" && it.Kind == scene.Line {
			right = math.Max(right, math.Max(it.X, it.X2))
		}
	}
	for _, it := range sc.Items {
		if it.Role == "wardley-component-label" && it.Bounds().X1 > right {
			t.Errorf("label ends at %.1f, the map at %.1f", it.Bounds().X1, right)
		}
	}
}

// REQ: WARD-005
// Kółko adnotacji o współrzędnych składnika odsuwa się z jego kółka, a podpis składnika
// z dekoratorem i bezwładnością nie wchodzi na nie.
func TestAnnotationAndDecoratorsClear(t *testing.T) {
	m, err := Parse("wardley-beta\ncomponent A [0.5, 0.5] (build) (inertia)\nannotation 1,[0.5, 0.5] \"x\"")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, Options{Measure: font.ApproxTextWidth, FontSize: 16})
	var comp, ann, inertia scene.Item
	var label scene.Box
	for _, it := range sc.Items {
		switch {
		case it.Role == "wardley-component":
			comp = it
		case it.Role == "wardley-annotation" && it.Kind == scene.Circle:
			ann = it
		case it.Role == "wardley-inertia":
			inertia = it
		case it.Role == "wardley-component-label":
			label = it.Bounds()
		}
	}
	if math.Hypot(comp.X-ann.X, comp.Y-ann.Y) < comp.R+ann.R {
		t.Errorf("annotation circle at %v,%v covers the component at %v,%v", ann.X, ann.Y, comp.X, comp.Y)
	}
	ib := inertia.Bounds()
	if label.X0 < ib.X1 && ib.X0 < label.X1 && label.Y0 < ib.Y1 && ib.Y0 < label.Y1 {
		t.Errorf("label %v overlaps the inertia bar %v", label, ib)
	}
}
