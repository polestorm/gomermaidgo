# gomermaidgo

Renders diagrams written in Mermaid syntax to SVG (and optionally PDF and PNG) in pure Go: no JavaScript, no browser, no dependencies outside the
Go standard library. The same source and options always give the same SVG, byte for byte.

```go
import "github.com/polestorm/gomermaidgo/mermaid"

svg, err := mermaid.Render(source)
```

## At a glance

Compared with [`mmdg`](https://github.com/bvolpato/mermaid-go-renderer)[^ref], the closest alternative: also a Go
program that needs no browser, but one that runs Mermaid itself in an embedded JavaScript engine.

| | gomermaidgo | mmdg 0.14.1 |
|---|---|---|
| Approach | own parsers and layout in Go | Mermaid 12.2.0 (JavaScript) in an embedded engine |
| Dependencies (Go modules) | 0, standard library only | 7 (4 direct, 3 indirect) |
| Program size (`go build`) | 8.2 MB | 53.0 MB |
| SVG: time / memory per diagram[^bench] | 9 ms / 18 MB | 38 ms / 41 MB |
| PDF: time / memory per diagram[^bench] | 12 ms / 24 MB | 38 ms / 54 MB |
| PNG at scale 1: time / memory per diagram[^bench] | 22 ms / 26 MB | 43 ms / 44 MB |
| Output size, 79 diagrams: SVG / PDF / PNG | 0.43 / 1.5 / 3.1 MB | 13.8 / 11.7 / 2.6 MB |
| Diagram types[^types] | 22 of Mermaid's 33 | 33 |
| Syntax coverage | a documented profile; anything outside it is an error with a line number | everything Mermaid accepts |
| This repository's corpus (82 diagrams) | 82 | 79 |
| Public diagrams from GitHub[^wild] | 138 of 139 | 137 of 139 |
| Look | own, made for print; neutral by default | Mermaid's, byte for byte |
| Page fitting | fit to A4, folding of long diagrams, chains and ranks | none |
| SVG content | basic SVG 1.1 elements, no CSS, no scripts, no `foreignObject` | Mermaid's SVG with CSS and embedded fonts |
| PDF and PNG | own writers, real text in PDF, optional (build tags) | yes |
| Markdown documents | yes (`markdown.Embed`, CLI) | yes (CLI) |
| `mermaid-cli` flags | yes | yes |
| Output identical byte for byte on every run | yes | yes |
| License | Apache 2.0 | MIT |
| Popularity (GitHub, 2026-11-23) | new | 5 stars, 1 fork |

[^ref]: `mmdg` is the reference renderer this library is compared with throughout its tests and reviews.
[^bench]: Median over the 79 corpus diagrams both programs render, best of three runs of the command-line program
    per file, process start included; memory is the peak resident size. Measured on 2026-11-23 on one Apple-silicon
    laptop.
[^types]: Mermaid 12.2 recognises 33 diagram types. Not drawn here: `zenuml` (a separate Mermaid add-on) and ten
    types added in Mermaid 12 as beta: `ishikawa`, `venn`, `swimlane`, `treeView`, `wardley`, `cynefin`, `railroad`,
    `eventmodeling`, `usecase` and `agentflow`. Process lanes are available here through `%% layout: swimlanes` on a
    flowchart.
[^wild]: 175 blocks marked `mermaid` collected from public GitHub repositories; 36 are rejected by both programs
    (fragments without a diagram header, or not Mermaid at all), which leaves 139.

## Usage

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

Colours (`WithColors`) and shadows (`WithShadows`) are opt-in; the default drawing is neutral and prints well in black
and white.

```sh
go run ./cmd/gomermaidgo -font NotoSans-Regular.ttf -family "Noto Sans" -o diagram.svg diagram.mmd
```

PDF output is an optional package: it writes the same drawing straight to a one-page PDF, with real text and the
font embedded as a subset. No SVG or HTML in between, and still no dependencies.

```go
doc, err := pdf.Render(source) // import "github.com/polestorm/gomermaidgo/pdf"
```

```sh
go run ./cmd/gomermaidgo -o diagram.pdf diagram.mmd   # the extension selects the format
go build -tags nopdf ./cmd/gomermaidgo                # a build without the PDF writer
```

Emoji and other characters the main typeface lacks are drawn with a bundled fallback, a static cut of Noto Emoji
(monochrome, OFL 1.1). In SVG they are written as outlines, so the drawing looks the same everywhere and needs no
font file for them; `WithEmojiText` (`-emoji-text`) keeps them as text instead. PDF keeps them as text with the
font embedded.

Graph diagrams are fitted to an A4 page by default: a drawing at least twice as wide as it is tall is laid out
again with narrower fields, wrapping text more often, and the version that fits the page at the largest scale is
kept. `WithFit(FitOn|FitOff|FitAuto)` and `WithFitRatio` (or `-fit on|off|auto`, `-fit-ratio`, or a
`%% layout: fit` / `%% layout: nofit` line in the source) change that. Long, simple diagrams are also folded into rows (LR) or
columns (TB) joined by connectors: `return` (default, every row in the same direction) or `snake`; see
`WithFold`, `WithFoldType`, `-fold`, `-fold-type` and `%% layout: fold snake|return` / `%% layout: nofold`.
Optionally (off by default), a long chain of nodes inside a diagram can be folded into columns (TB) or rows (LR)
of its own, when that makes the drawing at least 20% larger on the page: `WithFoldChains(FoldAuto)`,
`-fold-chains auto` or `%% layout: fold-chains`. Likewise (off by default), a node with six or more leaves
can stand them in two staggered ranks instead of one very long rank: `WithFoldRanks(FoldAuto)`,
`-fold-ranks auto` or `%% layout: fold-ranks`.

Themes: `neutral` (the default, made for print), `default` (colours), `forest`, `dark` and `base`. They are this
library's own looks under Mermaid's theme names, so a source that selects a theme in its front matter or `init`
directive gets the counterpart. `WithTheme` (or `-t`) overrides the source.

PNG output is another optional package, with its own rasteriser and glyph outlines read from the font file:

```go
img, err := png.Render(source)            // 2 image pixels per drawing pixel; png.RenderScale for another scale
```

```sh
go run ./cmd/gomermaidgo -o diagram.png -s 3 diagram.mmd
go build -tags nopdf,nopng ./cmd/gomermaidgo   # SVG only
```

The command also takes a Markdown document and the flags of `mermaid-cli` (`-i`, `-o`, `-e`, `-b`, `-t`, `-s`, `-q`, `-a`), so it
can stand in for `mmdc` in a script. Flags with no counterpart here (`-w`, `-H`, `-c`, `-C`, `-p`, `-I`)
are accepted and ignored with a note.

```sh
gomermaidgo -i README.template.md -o README.md   # README-1.svg, README-2.svg … and the document with images
gomermaidgo -i diagram.mmd -e pdf                # writes diagram.mmd.pdf
```

Diagrams inside a Markdown document: `markdown.Embed` replaces every fenced `mermaid` block with an inline SVG figure
before the document reaches a Markdown parser (the parser must pass raw HTML blocks through), and `markdown.CSS` holds
the page rules, including the height limit that keeps a tall diagram from being cut at a page break.

```go
md, problems := markdown.Embed(source, markdown.Options{})
```

Status: early. Supported diagram types: `sequenceDiagram`, `flowchart`/`graph`, `stateDiagram-v2`, `gantt`,
`erDiagram`, `classDiagram`, `pie`, `timeline`, `journey`, `mindmap`, `xychart-beta`, `quadrantChart`,
`kanban`, `packet-beta`, `radar-beta`, `treemap-beta`, `requirementDiagram`, `sankey-beta`, C4 diagrams, `gitGraph`, `block-beta` and `architecture-beta`. Graphs use a
layered layout written from the published algorithms; its current limits are listed in `docs/profile.md`. A construct outside the supported profile (`docs/profile.md`) is an error with a line number,
never a partial drawing.

The output uses only basic SVG 1.1 elements (`rect`, `line`, `polyline`, `polygon`, `circle`, `ellipse`, `text` and
`path` restricted to M, L and C commands) with presentation
attributes: no `foreignObject`, no style sheets, no scripts, no links. The one exception is a `clipPath` (in `defs`,
applied through `g`) on a transparent background: lines under edge labels and frame titles are cut out, so a label
sits on whatever the drawing is placed on instead of a white patch. With a background colour the label backing is
painted in that colour and the clip is not used.

Documentation (in Polish): `docs/profile.md` (supported syntax), `docs/provenance.md` (sources and clean-room rules),
`docs/testing.md` (corpus, invariants, comparison with reference renderers), `requirements/` (numbered requirements).

## How the comparison was measured

Both programs read the same corpus sources (`testdata/corpus/`). Times and memory come from the command-line
programs, so they include process start; for `mmdg` that includes starting its JavaScript engine. PNG is compared at
scale 1 in both programs (one image pixel per drawing pixel); this library's default is scale 2. The corpus is written
for this library's profile, so the corpus row says little about coverage: Mermaid accepts syntax this library rejects
(see `docs/profile.md`), and the drawings differ on purpose. The three corpus diagrams `mmdg` does not render are a
`block-beta` grid and two `sankey-beta` diagrams; the one public diagram this library rejects uses `subgraph` inside a
state diagram.

## License

Apache License 2.0, see `LICENSE` and `fonts/NOTICE`. The bundled Arimo and Noto Emoji typefaces are distributed under the
SIL Open Font License 1.1 (`fonts/OFL-Arimo.txt`, `fonts/OFL-NotoEmoji.txt`).
