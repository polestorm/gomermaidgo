package wardley

import (
	"math"
	"sort"
	"strconv"

	"github.com/polestorm/gomermaidgo/internal/scene"
)

// Options to ustawienia układu.
type Options struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool // ewolucja i siły w kolorach
	Shadows    bool // bez znaczenia: mapa nie ma pól
}

const (
	colorText   = "#1b1f24"
	colorMuted  = "#55657a"
	colorAxis   = "#3b4754"
	colorGrid   = "#8a96a3"
	colorLine   = "#55657a"
	colorDot    = "#ffffff"
	colorDeco   = "#dfe4ea"
	colorEvolve = "#3b4754"
	colorRed    = "#b3261e"
	colorForce  = "#eef2f7"
)

// defaultStages to klasyczne etapy ewolucji z granicami.
var defaultStages = []Stage{
	{[]string{"Genesis"}, 0.17},
	{[]string{"Custom built"}, 0.40},
	{[]string{"Product", "(+ rental)"}, 0.70},
	{[]string{"Commodity", "(+ utility)"}, 1},
}

// Layout rysuje mapę: oś widoczności z lewej, oś ewolucji na dole z etapami rozdzielonymi
// przerywanymi liniami, składniki jako kółka z podpisem, kotwice jako pogrubiony podpis,
// zależności jako linie, ewolucję jako przerywaną strzałkę do przyszłego położenia.
func Layout(m *Map, o Options) *scene.Scene {
	fs := o.FontSize
	if fs <= 0 {
		fs = 16
	}
	u := func(v float64) float64 { return math.Round(v * fs / 16) }
	lh := math.Round(fs * 1.25)
	small := math.Round(fs * 0.8)
	W, H := u(820), u(520)
	if m.Width > 0 && m.Height > 0 {
		W, H = math.Max(u(300), m.Width-u(140)), math.Max(u(200), m.Height-u(100))
	}
	sc := &scene.Scene{Title: m.AccTitle, Desc: m.AccDescr, FontFamily: o.FontFamily, FontSize: fs, Margin: u(12), Background: o.Background}
	px := func(evo float64) float64 { return evo * W }
	py := func(vis float64) float64 { return (1 - vis) * H }
	text := func(s string, x, y, size float64, anchor string, bold, italic bool, fill, role string) {
		sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: role, X: x, Y: y, W: o.Measure(s, size), H: size * 1.25,
			Text: s, Anchor: anchor, Bold: bold, Italic: italic, FontSize: size, Fill: fill})
	}
	var segs [][4]float64 // linie na planszy, których podpisy składników unikają
	line := func(x1, y1, x2, y2 float64, stroke string, width float64, dash, role string) {
		if role != "wardley-axis" {
			segs = append(segs, [4]float64{x1, y1, x2, y2})
		}
		// Linie leżą pod podkładami podpisów: podpis je zasłania albo wycina.
		sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: role, X: x1, Y: y1, X2: x2, Y2: y2, Stroke: stroke, StrokeWidth: width, Dash: dash})
	}
	var bands []float64 // prawe granice etapów ewolucji (0–1)
	// box zwraca prostokąt podkładu podpisu: x0, y0, x1, y1.
	box := func(s string, x, y, size float64, anchor string, bold bool) [4]float64 {
		w := o.Measure(s, size)
		if bold {
			w *= 1.06
		}
		x0 := x - w/2
		switch anchor {
		case "start":
			x0 = x
		case "end":
			x0 = x - w
		}
		return [4]float64{x0 - u(2), y - size*0.95, x0 + w + u(2), y + size*0.35}
	}
	var placed [][4]float64 // podkłady podpisów już narysowanych
	// label rysuje podpis na podkładzie: w trybie kolorowym w odcieniach etapów, na których stoi
	// (podkład dzieli się na granicach etapów, więc wygląda jak przezroczysty), a w neutralnym
	// w kolorze tła rysunku (przy tle przezroczystym podkład wycina linie pod podpisem).
	label := func(s string, x, y, size float64, anchor string, bold, italic bool, fill, role string) {
		b := box(s, x, y, size, anchor, bold)
		placed = append(placed, b)
		if !o.Colors {
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: role + "-background",
				X: b[0], Y: b[1], W: b[2] - b[0], H: b[3] - b[1], Knockout: true})
		} else {
			from := 0.0
			for i, until := range bands {
				to := until * W // poza planszą (za ostatnim etapem) podkład nie jest potrzebny: nie ma tam linii
				if x0, x1 := math.Max(b[0], from), math.Min(b[2], to); x1 > x0 {
					sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerLabelBackground, Role: role + "-background",
						X: x0, Y: b[1], W: x1 - x0, H: b[3] - b[1], Fill: scene.Distinct(i).Fill})
				}
				from = to
			}
		}
		text(s, x, y, size, anchor, bold, italic, fill, role)
	}
	// clash liczy kolizje prostokąta z liniami, innymi podpisami i kółkami składników.
	var dots [][3]float64
	clash := func(b [4]float64) int {
		n := 0
		for _, sg := range segs {
			for k := 0; k <= 20; k++ {
				t := float64(k) / 20
				x, y := sg[0]+(sg[2]-sg[0])*t, sg[1]+(sg[3]-sg[1])*t
				if x > b[0] && x < b[2] && y > b[1] && y < b[3] {
					n++
					break
				}
			}
		}
		for _, p := range placed {
			if b[0] < p[2] && p[0] < b[2] && b[1] < p[3] && p[1] < b[3] {
				n += 2
			}
		}
		for _, d := range dots {
			if d[0]+d[2] > b[0] && d[0]-d[2] < b[2] && d[1]+d[2] > b[1] && d[1]-d[2] < b[3] {
				n += 2
			}
		}
		// Podpis wychodzący poza planszę (poza osie albo za ostatni etap) liczy się jak kilka kolizji:
		// stoi wtedy poza tłem etapów, a przy prawej krawędzi wychodzi poza rysunek.
		if b[0] < 0 || b[2] > W || b[1] < 0 || b[3] > H {
			n += 3
		}
		return n
	}
	arrow := func(x, y, dx, dy float64, fill string) {
		l := math.Hypot(dx, dy)
		if l == 0 {
			return
		}
		dx, dy = dx/l, dy/l
		a, w := u(10), u(5)
		sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "wardley-arrowhead", Fill: fill, Stroke: fill, StrokeWidth: 1,
			Points: []scene.Point{{X: x, Y: y}, {X: x - dx*a - dy*w, Y: y - dy*a + dx*w}, {X: x - dx*a + dy*w, Y: y - dy*a - dx*w}}})
	}

	if m.Title != "" {
		text(m.Title, W/2, -u(20), fs*1.15, "middle", true, false, colorText, "wardley-title")
	}
	// Osie: widoczność (pionowa, z grotem u góry) i ewolucja (pozioma, z grotem z prawej).
	line(0, H, 0, 0, colorAxis, 1.5, "", "wardley-axis")
	arrow(0, -u(2), 0, -1, colorAxis)
	line(0, H, W, H, colorAxis, 1.5, "", "wardley-axis")
	arrow(W+u(2), H, 1, 0, colorAxis)
	sc.Add(scene.Item{Kind: scene.Text, Layer: scene.LayerLabel, Role: "wardley-axis-label", X: -u(10), Y: H / 2, W: o.Measure("Value chain", small), H: small * 1.25,
		Text: "Value chain", Anchor: "middle", Vertical: true, FontSize: small, Fill: colorMuted})
	text("Visible", -u(6), u(4)+small*0.35, small, "end", false, true, colorMuted, "wardley-axis-label")
	text("Invisible", -u(6), H-u(4), small, "end", false, true, colorMuted, "wardley-axis-label")
	text("Evolution", W, H+u(10)+2*small*1.25+small*0.9, small, "end", false, true, colorMuted, "wardley-axis-label")
	// Etapy ewolucji: granice podane w źródle albo równe odcinki, nazwy pod osią.
	stages := defaultStages
	if len(m.Stages) > 0 {
		stages = append([]Stage(nil), m.Stages...)
		for i := range stages {
			if stages[i].Until < 0 {
				stages[i].Until = float64(i+1) / float64(len(stages))
			}
		}
		stages[len(stages)-1].Until = 1
	}
	for _, st := range stages {
		bands = append(bands, st.Until)
	}
	from := 0.0
	for i, st := range stages {
		if o.Colors {
			// Tryb kolorowy: każdy etap ewolucji ma swój jasny odcień.
			sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "wardley-stage-band", X: px(from), Y: 0, W: px(st.Until) - px(from), H: H,
				Fill: scene.Distinct(i).Fill})
		}
		if i < len(stages)-1 {
			sc.Add(scene.Item{Kind: scene.Line, Layer: scene.LayerLifeline, Role: "wardley-stage", X: px(st.Until), Y: 0, X2: px(st.Until), Y2: H,
				Stroke: colorGrid, StrokeWidth: 1.25, Dash: "5 4"})
		}
		for k, s := range st.Names {
			text(s, px((from+st.Until)/2), H+u(8)+float64(k)*small*1.25+small*0.9, small, "middle", k == 0, k > 0, colorMuted, "wardley-stage-label")
		}
		from = st.Until
	}

	at := func(c *Component) (float64, float64) { return px(c.At.Evo), py(c.At.Vis) }
	r := u(6)
	// Potoki: ramka wokół składników potoku na wysokości rodzica.
	pipes := map[string][2]float64{}
	var pipeNames []string
	for _, c := range m.Components {
		if c.Pipeline == "" {
			continue
		}
		span, ok := pipes[c.Pipeline]
		if !ok {
			span = [2]float64{math.Inf(1), math.Inf(-1)}
			pipeNames = append(pipeNames, c.Pipeline)
		}
		pipes[c.Pipeline] = [2]float64{math.Min(span[0], px(c.At.Evo)), math.Max(span[1], px(c.At.Evo))}
	}
	for _, name := range pipeNames {
		span, parent := pipes[name], m.Component(name)
		y := py(parent.At.Vis) + u(26)
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "wardley-pipeline", X: span[0] - u(16), Y: y - u(12), W: span[1] - span[0] + u(32), H: u(24),
			Fill: "none", Stroke: colorLine, StrokeWidth: 1})
		_, py0 := at(parent)
		line(px(parent.At.Evo), py0+r, px(parent.At.Evo), y-u(12), colorLine, 1, "", "wardley-pipeline")
	}
	pipeY := func(c *Component) float64 {
		_, y := at(c)
		if c.Pipeline != "" {
			return y + u(26)
		}
		return y
	}
	point := func(c *Component) (float64, float64) { x, _ := at(c); return x, pipeY(c) }

	// Zależności: linia między środkami, skrócona o promień kółka.
	for _, l := range m.Links {
		a, b := m.Component(l.From), m.Component(l.To)
		x1, y1 := point(a)
		x2, y2 := point(b)
		dx, dy := x2-x1, y2-y1
		d := math.Hypot(dx, dy)
		if d < 2*r {
			continue
		}
		ux, uy := dx/d, dy/d
		sa, sb := r, r
		if a.Anchor {
			sa = lh / 2
		}
		dash := ""
		if l.Dashed {
			dash = "5 4"
		}
		line(x1+ux*sa, y1+uy*sa, x2-ux*sb, y2-uy*sb, colorLine, 1.2, dash, "wardley-link")
		if l.EndArrow {
			arrow(x2-ux*sb, y2-uy*sb, ux, uy, colorLine)
		}
		if l.StartArrow {
			arrow(x1+ux*sa, y1+uy*sa, -ux, -uy, colorLine)
		}
		if l.Label != "" {
			mx, my := (x1+x2)/2, (y1+y2)/2
			label(l.Label, mx, my+small*0.35, small, "middle", false, true, colorMuted, "wardley-link-label")
		}
	}
	// Przeszkody dla podpisów składników: kółka (z dekoratorami pod nimi), belki bezwładności,
	// podpisy kotwic, notatki, siły i kółka adnotacji.
	for _, c := range m.Components {
		x, y := point(c)
		if c.Anchor {
			placed = append(placed, box(c.Name, x+c.LabelDX, y+0.35*fs+c.LabelDY, fs, "middle", true))
			continue
		}
		rr := r
		for _, d := range c.Decorators {
			if d == "inertia" {
				placed = append(placed, [4]float64{x + u(16), y - u(11), x + u(21), y + u(11)})
			} else {
				rr = u(14)
			}
		}
		dots = append(dots, [3]float64{x, y, rr})
	}
	// Kółko adnotacji stojące na kółku składnika (te same współrzędne) odsuwa się ukośnie
	// w miejsce, które najmniej koliduje; jego położenie jest ustalone przed podpisami.
	annotAt := map[int][2]float64{}
	for k, t := range m.Texts {
		x, y := px(t.At.Evo), py(t.At.Vis)
		switch t.Kind {
		case "note":
			placed = append(placed, box(t.Text, x, y, small, "start", true))
		case "accelerator", "deaccelerator":
			w, h := math.Max(u(70), o.Measure(t.Text, small)+u(24)), u(26)
			placed = append(placed, [4]float64{x - w/2, y - h/2, x + w/2, y + h/2})
		case "annotation":
			onDot := false
			for _, d := range dots {
				if math.Hypot(d[0]-x, d[1]-y) < d[2]+u(9) {
					onDot = true
				}
			}
			if onDot {
				best, bestN := [2]float64{x, y}, math.MaxInt
				for _, d := range [][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
					cx, cy := x+d[0]*u(24), y+d[1]*u(24)
					if n := clash([4]float64{cx - u(9), cy - u(9), cx + u(9), cy + u(9)}); n < bestN {
						best, bestN = [2]float64{cx, cy}, n
					}
				}
				x, y = best[0], best[1]
			}
			annotAt[k] = [2]float64{x, y}
			dots = append(dots, [3]float64{x, y, u(9)})
		}
	}
	// around stawia podpis kółka w pierwszym z sześciu miejsc wokół niego, które nie wchodzi na
	// linie, inne podpisy, kółka ani poza planszę; gdy każde koliduje, w tym z najmniejszą liczbą kolizji.
	// rr to zasięg kółka razem z dekoratorem pod nim, right — dodatkowy odstęp z prawej (belka
	// bezwładności).
	around := func(name string, x, y, rr, right float64, italic bool, fill, role string) {
		type spot struct {
			x, y   float64
			anchor string
		}
		spots := []spot{{x + rr + right + u(4), y - r, "start"}, {x + rr + right + u(4), y + r + small*0.8, "start"},
			{x - rr - u(4), y - r, "end"}, {x - rr - u(4), y + r + small*0.8, "end"},
			{x, y - rr - u(6), "middle"}, {x, y + rr + small + u(2), "middle"}}
		best, bestN := spots[0], math.MaxInt
		for _, sp := range spots {
			if n := clash(box(name, sp.x, sp.y, small, sp.anchor, false)); n < bestN {
				best, bestN = sp, n
			}
		}
		label(name, best.x, best.y, small, best.anchor, false, italic, fill, role)
	}
	// Ewolucja: przerywana strzałka do przyszłego położenia i kółko z przerywanym obrysem.
	red := colorEvolve
	if o.Colors {
		red = colorRed
	}
	for _, e := range m.Evolves {
		c := m.Component(e.Name)
		x1, y := point(c)
		x2 := px(e.To)
		dir := 1.0
		if x2 < x1 {
			dir = -1
		}
		line(x1+dir*r, y, x2-dir*(r+u(2)), y, red, 1.5, "6 4", "wardley-evolve")
		arrow(x2-dir*(r+u(2)), y, dir, 0, red)
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "wardley-evolved", X: x2, Y: y, R: r, Fill: colorDot, Stroke: red, StrokeWidth: 1.5, Dash: "3 2"})
		dots = append(dots, [3]float64{x2, y, r})
		around(c.Name, x2, y, r, 0, true, red, "wardley-evolved-label")
	}
	for _, t := range m.Trends {
		c := m.Component(t.Name)
		x1, y1 := point(c)
		x2, y2 := px(t.At.Evo), py(t.At.Vis)
		line(x1, y1, x2, y2, colorLine, 1, "2 3", "wardley-trend")
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "wardley-trend", X: x2, Y: y2, R: r * 0.7, Fill: "none", Stroke: colorLine, StrokeWidth: 1, Dash: "2 2"})
	}

	// Składniki: dekoratory pod kółkiem, kółko, podpis obok (domyślnie z prawej u góry).
	for _, c := range m.Components {
		x, y := point(c)
		if c.Anchor {
			label(c.Name, x+c.LabelDX, y+0.35*fs+c.LabelDY, fs, "middle", true, false, colorText, "wardley-anchor")
			continue
		}
		for _, d := range c.Decorators {
			s := u(13)
			switch d {
			case "build":
				sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerBackground, Role: "wardley-decorator", Fill: colorDeco, Stroke: colorLine, StrokeWidth: 1,
					Points: []scene.Point{{X: x, Y: y - s}, {X: x + s, Y: y + s*0.75}, {X: x - s, Y: y + s*0.75}}})
			case "buy":
				sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerBackground, Role: "wardley-decorator", Fill: colorDeco, Stroke: colorLine, StrokeWidth: 1,
					Points: []scene.Point{{X: x, Y: y - s}, {X: x + s, Y: y}, {X: x, Y: y + s}, {X: x - s, Y: y}}})
			case "outsource":
				sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerBackground, Role: "wardley-decorator", X: x - s*0.8, Y: y - s*0.8, W: 1.6 * s, H: 1.6 * s, Fill: colorDeco, Stroke: colorLine, StrokeWidth: 1})
			case "market":
				sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerBackground, Role: "wardley-decorator", X: x, Y: y, R: s, Fill: colorDeco, Stroke: colorLine, StrokeWidth: 1})
			case "inertia":
				sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "wardley-inertia", X: x + u(16), Y: y - u(11), W: u(5), H: u(22), Fill: colorAxis})
			}
		}
		sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "wardley-component", X: x, Y: y, R: r, Fill: colorDot, Stroke: colorAxis, StrokeWidth: 1.5})
		lx, ly := x+r+u(4), y-r
		if c.Pipeline != "" {
			lx, ly = x, y+u(12)+small+u(4)
			label(c.Name, lx+c.LabelDX, ly+c.LabelDY, small, "middle", false, false, colorText, "wardley-component-label")
			continue
		}
		if c.LabelDX != 0 || c.LabelDY != 0 {
			label(c.Name, lx+c.LabelDX, ly+c.LabelDY, small, "start", false, false, colorText, "wardley-component-label")
			continue
		}
		rr, right := r, 0.0
		for _, d := range c.Decorators {
			if d == "inertia" {
				right = u(18)
			} else {
				rr = u(14)
			}
		}
		around(c.Name, x, y, rr, right, false, colorText, "wardley-component-label")
	}

	// Notatki, siły i adnotacje.
	var notes []Text
	for k, t := range m.Texts {
		x, y := px(t.At.Evo), py(t.At.Vis)
		if p, ok := annotAt[k]; ok {
			x, y = p[0], p[1]
		}
		switch t.Kind {
		case "note":
			text(t.Text, x, y, small, "start", true, false, colorText, "wardley-note")
		case "accelerator", "deaccelerator":
			w, h := math.Max(u(70), o.Measure(t.Text, small)+u(24)), u(26)
			fill := colorForce
			if o.Colors {
				fill = "#e6f4ea"
				if t.Kind == "deaccelerator" {
					fill = "#fbe9e7"
				}
			}
			pts := []scene.Point{{X: x - w/2, Y: y - h/2}, {X: x + w/2 - u(10), Y: y - h/2}, {X: x + w/2, Y: y}, {X: x + w/2 - u(10), Y: y + h/2}, {X: x - w/2, Y: y + h/2}}
			if t.Kind == "deaccelerator" {
				pts = []scene.Point{{X: x + w/2, Y: y - h/2}, {X: x - w/2 + u(10), Y: y - h/2}, {X: x - w/2, Y: y}, {X: x - w/2 + u(10), Y: y + h/2}, {X: x + w/2, Y: y + h/2}}
			}
			sc.Add(scene.Item{Kind: scene.Polygon, Layer: scene.LayerShape, Role: "wardley-" + t.Kind, Points: pts, Fill: fill, Stroke: colorLine, StrokeWidth: 1})
			text(t.Text, x, y+small*0.35, small, "middle", false, false, colorText, "wardley-"+t.Kind+"-label")
		case "annotation":
			notes = append(notes, t)
			sc.Add(scene.Item{Kind: scene.Circle, Layer: scene.LayerShape, Role: "wardley-annotation", X: x, Y: y, R: u(9), Fill: colorDot, Stroke: colorAxis, StrokeWidth: 1})
			text(strconv.Itoa(t.Number), x, y+small*0.35, small, "middle", true, false, colorText, "wardley-annotation")
		}
	}
	if len(notes) > 0 {
		sort.SliceStable(notes, func(a, b int) bool { return notes[a].Number < notes[b].Number })
		x, y := u(16), u(16)
		if m.Annotations != nil {
			x, y = px(m.Annotations.Evo), py(m.Annotations.Vis)
		}
		w := 0.0
		for _, t := range notes {
			w = math.Max(w, o.Measure(strconv.Itoa(t.Number)+". "+t.Text, small))
		}
		sc.Add(scene.Item{Kind: scene.Rect, Layer: scene.LayerShape, Role: "wardley-annotations", X: x, Y: y, W: w + u(16), H: float64(len(notes))*small*1.25 + u(12),
			Fill: colorDot, Stroke: colorGrid, StrokeWidth: 1})
		for k, t := range notes {
			text(strconv.Itoa(t.Number)+". "+t.Text, x+u(8), y+u(6)+(float64(k)+0.5)*small*1.25+small*0.35, small, "start", false, false, colorText, "wardley-annotations")
		}
	}
	return sc
}
