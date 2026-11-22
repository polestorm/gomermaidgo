package png

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	stdpng "image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/polestorm/gomermaidgo"
	"github.com/polestorm/gomermaidgo/fonts"
)

var svgSize = regexp.MustCompile(`<svg[^>]* width="([\d.]+)" height="([\d.]+)"`)

func nrgba(c color.Color) color.NRGBA { return color.NRGBAModel.Convert(c).(color.NRGBA) }

func decode(t *testing.T, name string, data []byte) image.Image {
	t.Helper()
	img, err := stdpng.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return img
}

// REQ: PNG-001 PNG-002
// Każdy diagram korpusu daje poprawny obraz o wymiarach rysunku SVG razy skala, z treścią w środku
// i przezroczystym brzegiem.
func TestCorpus(t *testing.T) {
	files, _ := filepath.Glob("../testdata/corpus/*/*.mmd")
	if len(files) == 0 {
		t.Fatal("no corpus files")
	}
	for _, path := range files {
		src, _ := os.ReadFile(path)
		data, err := Render(string(src))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		img := decode(t, path, data)
		svg, _ := gomermaidgo.Render(string(src))
		m := svgSize.FindSubmatch(svg)
		w, _ := strconv.ParseFloat(string(m[1]), 64)
		h, _ := strconv.ParseFloat(string(m[2]), 64)
		b := img.Bounds()
		if b.Dx() != int(math.Ceil(w*2)) || b.Dy() != int(math.Ceil(h*2)) {
			t.Errorf("%s: image %dx%d, drawing %vx%v", path, b.Dx(), b.Dy(), w, h)
		}
		painted := 0
		for y := 0; y < b.Dy(); y += 3 {
			for x := 0; x < b.Dx(); x += 3 {
				if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
					painted++
				}
			}
		}
		if _, _, _, a := img.At(0, 0).RGBA(); a != 0 || painted < 50 {
			t.Errorf("%s: corner alpha %d, painted samples %d", path, a, painted)
		}
	}
}

// REQ: PNG-001 PNG-002
func TestScaleBackgroundDeterminism(t *testing.T) {
	src := "flowchart TD\nA[Początek] --> B[Koniec]\nstyle A fill:#ff0000,stroke:#0000ff"
	a, err := Render(src)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := Render(src); !bytes.Equal(a, b) {
		t.Errorf("the same source should give the same file")
	}
	two := decode(t, "x2", a).Bounds()
	one, err := RenderScale(src, 1)
	if err != nil {
		t.Fatal(err)
	}
	img := decode(t, "x1", one)
	if b := img.Bounds(); b.Dx()*2 != two.Dx() || b.Dy()*2 != two.Dy() {
		t.Errorf("scale 1 is %v, scale 2 is %v", b, two)
	}
	// Kolory ze stylu: czerwone wnętrze pola i niebieski obrys są na obrazie.
	red, blue := 0, 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			switch c := nrgba(img.At(x, y)); {
			case c == color.NRGBA{255, 0, 0, 255}:
				red++
			case c.B > 150 && c.R < 120 && c.G < 60 && c.A > 200:
				blue++
			}
		}
	}
	if red < 500 || blue < 100 {
		t.Errorf("styled colours: red %d, blue %d", red, blue)
	}
	withBg, err := RenderScale(src, 1, gomermaidgo.WithBackground("#fff8e1"))
	if err != nil {
		t.Fatal(err)
	}
	if got := nrgba(decode(t, "bg", withBg).At(0, 0)); got != (color.NRGBA{0xff, 0xf8, 0xe1, 255}) {
		t.Errorf("background corner: %v", got)
	}
	// Cień jest półprzezroczysty: na przezroczystym tle zostają piksele czerni o częściowym kryciu.
	shadow, err := RenderScale("flowchart TD\nA --> B", 1, gomermaidgo.WithShadows())
	if err != nil {
		t.Fatal(err)
	}
	faint := 0
	simg := decode(t, "shadow", shadow)
	for y := 0; y < simg.Bounds().Dy(); y++ {
		for x := 0; x < simg.Bounds().Dx(); x++ {
			if c := nrgba(simg.At(x, y)); c.A > 15 && c.A < 40 && c.R < 8 {
				faint++
			}
		}
	}
	if faint < 20 {
		t.Errorf("shadow pixels: %d", faint)
	}
	for _, scale := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := RenderScale(src, scale); err == nil {
			t.Errorf("scale %v: want an error", scale)
		}
	}
	if _, err := RenderScale(src, 400); !errors.Is(err, ErrTooLarge) {
		t.Errorf("huge scale: %v", err)
	}
	var se *gomermaidgo.SyntaxError
	if _, err := Render("flowchart TD\nA -->"); !errors.As(err, &se) || se.Line != 2 {
		t.Errorf("syntax error should pass through: %v", err)
	}
}

// ink liczy ciemne piksele obrazu.
func ink(t *testing.T, src string, opts ...gomermaidgo.Option) int {
	t.Helper()
	data, err := RenderScale(src, 2, opts...)
	if err != nil {
		t.Fatal(err)
	}
	img := decode(t, src, data)
	n := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if c := nrgba(img.At(x, y)); c.A > 200 && int(c.R)+int(c.G)+int(c.B) < 200 {
				n++
			}
		}
	}
	return n
}

// REQ: PNG-003
// Tekst jest rysowany konturami: więcej liter to więcej farby, pogrubienie ją pogrubia, znak
// diakrytyczny dokłada swoją część; bez pliku kroju format zwraca błąd.
func TestText(t *testing.T) {
	base := ink(t, "flowchart TD\nA[\"mmmm\"]")
	if more := ink(t, "flowchart TD\nA[\"mmmmmmmm\"]"); more < base+base/3 {
		t.Errorf("twice the letters: %d against %d", more, base)
	}
	if bold := ink(t, "flowchart TD\nA[\"**mmmm**\"]"); bold < base+base/10 {
		t.Errorf("bold: %d against %d", bold, base)
	}
	if plain, marked := ink(t, "flowchart TD\nA[\"oooooooo\"]"), ink(t, "flowchart TD\nA[\"óóóóóóóó\"]"); marked < plain+40 {
		t.Errorf("letters with marks: %d against %d", marked, plain)
	}
	if custom := ink(t, "flowchart TD\nA[\"mmmm\"]", gomermaidgo.WithFont(fonts.RecursiveBold())); custom <= base {
		t.Errorf("WithFont: %d against %d", custom, base)
	}
	measure := gomermaidgo.WithMeasure(func(s string, size float64) float64 { return float64(len(s)) * size / 2 })
	if _, err := Render("flowchart TD\nA --> B", measure); !errors.Is(err, gomermaidgo.ErrFontRequired) {
		t.Errorf("WithMeasure: %v", err)
	}
}

// REQ: PNG-001
// Dowolne źródło daje obraz albo błąd, nigdy panikę.
func FuzzRender(f *testing.F) {
	files, _ := filepath.Glob("../testdata/corpus/*/*.mmd")
	for _, path := range files {
		if src, err := os.ReadFile(path); err == nil {
			f.Add(string(src))
		}
	}
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 2000 {
			return
		}
		data, err := RenderScale(src, 1)
		if err != nil {
			return
		}
		if _, err := stdpng.Decode(bytes.NewReader(data)); err != nil {
			t.Fatalf("invalid image: %v", err)
		}
	})
}
