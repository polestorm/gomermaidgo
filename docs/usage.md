# Usage

- [Rendering a diagram](#rendering-a-diagram)
- [Fonts](#fonts)
- [Colours, shadows and themes](#colours-shadows-and-themes)
- [Emoji and symbols](#emoji-and-symbols)
- [Fitting to an A4 page](#fitting-to-an-a4-page)
- [PDF](#pdf)
- [PNG](#png)
- [Command line](#command-line)
- [Markdown documents](#markdown-documents)
- [What the SVG contains](#what-the-svg-contains)

## Rendering a diagram

```go
import "github.com/polestorm/gomermaidgo/mermaid"

svg, err := mermaid.Render(source)
```

`Render` takes the Mermaid source and options (`mermaid.With…`) and returns the SVG. The same source and options
always give the same SVG, byte for byte. A construct outside the supported profile ([`profile.md`](profile.md), in
Polish) is an error with a line number and a fixed code, never a partial drawing.

## Fonts

By default text is measured with the bundled Arimo typeface (`fonts.Arimo()`, SIL Open Font License 1.1, see
`fonts/OFL-Arimo.txt`), and the SVG asks for `Arimo, Arial, 'Liberation Sans', sans-serif`. Arimo has the same
character widths as Arial and Liberation Sans, so the drawing keeps its layout on Windows, macOS and most Linux
desktops without any font file; elsewhere, give the file to whatever draws the SVG (a page through `@font-face`, a PDF
converter as an embedded font). To use another typeface, pass both its file and its name:

```go
svg, err := mermaid.Render(source,
	mermaid.WithFont(fontBytes),         // measure text with the font that will draw it
	mermaid.WithFontFamily("Noto Sans"),
)
```

```sh
gomermaidgo -font NotoSans-Regular.ttf -family "Noto Sans" -o diagram.svg diagram.mmd
```

The font must have TrueType outlines (`glyf`); fonts with CFF outlines and font collections (`.ttc`) are not read.
A drawing uses one text font: for Japanese, Chinese or Korean pass a font that has those characters (see
[scripts and symbols](gallery/scripts.md)).

## Colours, shadows and themes

Colours (`WithColors`, `-colors`) and shadows (`WithShadows`) are opt-in; the default drawing is neutral and prints
well in black and white.

Themes: `neutral` (the default, made for print), `default` (colours), `forest`, `dark` and `base`. They are this
library's own looks under Mermaid's theme names, so a source that selects a theme in its front matter or `init`
directive gets the counterpart. `WithTheme` (or `-t`) overrides the source.

## Emoji and symbols

Emoji and other characters the main typeface lacks are drawn with a bundled fallback, a static cut of Noto Emoji
(monochrome, OFL 1.1). In SVG they are written as outlines, so the drawing looks the same everywhere and needs no
font file for them; `WithEmojiText` (`-emoji-text`) keeps them as text instead. PDF keeps them as text with the
font embedded.

## Fitting to an A4 page

Mermaid lays a diagram out in one long row or column, whatever its length. This library can lay it out again so
that it fits an A4 page at a larger scale; the [page fitting gallery](gallery/page-fitting.md) shows each option
with and without it.

| What | Default | In the source | Go | Command line |
|---|---|---|---|---|
| Fit: narrower fields, more text wrapping | `auto` | `%% layout: fit` / `nofit` | `WithFit`, `WithFitRatio` | `-fit`, `-fit-ratio` |
| Fold a long diagram into rows or columns | `auto` | `%% layout: fold` / `nofold` | `WithFold` | `-fold` |
| How folded rows continue: `return` or `snake` | `return` | `%% layout: fold snake` | `WithFoldType` | `-fold-type` |
| Fold a long chain inside a diagram | off | `%% layout: fold-chains` | `WithFoldChains` | `-fold-chains` |
| Stagger a long rank of leaves | off | `%% layout: fold-ranks` | `WithFoldRanks` | `-fold-ranks` |

- Fitting: a graph drawing at least twice as wide as it is tall (`WithFitRatio`) is laid out again with narrower
  fields, and the version that fits the page at the largest scale is kept.
- Folding: a long, simple diagram is folded into rows (LR) or columns (TB) joined by connectors; every row runs in
  the same direction (`return`) or every other row is reversed (`snake`).
- Chain folding: a long chain of nodes inside a diagram is folded into columns (TB) or rows (LR) of its own, when
  that makes the drawing at least 20% larger on the page.
- Rank staggering: a node with six or more leaves stands them in two staggered ranks instead of one very long rank,
  under the same 20% condition.

An option passed in Go or on the command line takes precedence over the line in the source.

## PDF

PDF output is an optional package: it writes the same drawing straight to a one-page PDF, with real text and the
font embedded as a subset. No SVG or HTML in between, and still no dependencies.

```go
doc, err := pdf.Render(source) // import "github.com/polestorm/gomermaidgo/pdf"
```

```sh
gomermaidgo -o diagram.pdf diagram.mmd   # the extension selects the format
go build -tags nopdf ./cmd/gomermaidgo   # a build without the PDF writer
```

## PNG

PNG output is another optional package, with its own rasteriser and glyph outlines read from the font file:

```go
img, err := png.Render(source) // import "github.com/polestorm/gomermaidgo/png"; 2 image pixels per drawing pixel
```

```sh
gomermaidgo -o diagram.png -s 3 diagram.mmd
go build -tags nopdf,nopng ./cmd/gomermaidgo   # SVG only
```

`png.RenderScale` takes another scale.

## Command line

```sh
go install github.com/polestorm/gomermaidgo/cmd/gomermaidgo@latest
gomermaidgo -o diagram.svg diagram.mmd
```

The command also takes a Markdown document and the flags of `mermaid-cli` (`-i`, `-o`, `-e`, `-b`, `-t`, `-s`,
`-q`, `-a`), so it can stand in for `mmdc` in a script. Flags with no counterpart here (`-w`, `-H`, `-c`, `-C`,
`-p`, `-I`) are accepted and ignored with a note. `gomermaidgo -h` lists all flags.

```sh
gomermaidgo -i README.template.md -o README.md   # README-1.svg, README-2.svg … and the document with images
gomermaidgo -i diagram.mmd -e pdf                # writes diagram.mmd.pdf
```

## Markdown documents

`markdown.Embed` replaces every fenced `mermaid` block with an inline SVG figure before the document reaches a
Markdown parser (the parser must pass raw HTML blocks through), and `markdown.CSS` holds the page rules, including
the height limit that keeps a tall diagram from being cut at a page break.

```go
md, problems := markdown.Embed(source, markdown.Options{}) // import "github.com/polestorm/gomermaidgo/markdown"
```

## What the SVG contains

The output uses only basic SVG 1.1 elements (`rect`, `line`, `polyline`, `polygon`, `circle`, `ellipse`, `text` and
`path` restricted to M, L and C commands) with presentation attributes: no `foreignObject`, no style sheets, no
scripts, no links. The one exception is a `clipPath` (in `defs`, applied through `g`) on a transparent background:
lines under edge labels and frame titles are cut out, so a label sits on whatever the drawing is placed on instead
of a white patch. With a background colour the label backing is painted in that colour and the clip is not used.

Elements carry no `class` attributes by default. `WithClasses` (`-classes`) adds one with the element's role
(`node`, `edge`, `edge-label`, `arrowhead`, `lane` and others), so a page that embeds the SVG inline can restyle it
with its own CSS; the drawing is otherwise the same, and the file about a fifth larger.
