# gomermaidgo

Renders diagrams written in Mermaid syntax to SVG (and optionally PDF and PNG) in pure Go: no JavaScript, no browser, no dependencies outside the
Go standard library. The same source and options always give the same SVG, byte for byte.

```go
svg, err := gomermaidgo.Render(source)
```

By default text is measured with, and the SVG asks for, the bundled Recursive typeface (`fonts.Recursive()`, SIL Open
Font License 1.1, see `fonts/OFL.txt`). Give the same file to whatever draws the SVG: a page through `@font-face`, a
PDF converter as an embedded font. To use another typeface, pass both its file and its name:

```go
svg, err := gomermaidgo.Render(source,
	gomermaidgo.WithFont(fontBytes),         // measure text with the font that will draw it
	gomermaidgo.WithFontFamily("Noto Sans"),
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
attributes: no `foreignObject`, no style sheets, no scripts, no links, no ids.

Documentation (in Polish): `docs/profile.md` (supported syntax), `docs/provenance.md` (sources and clean-room rules),
`docs/testing.md` (corpus, invariants, comparison with reference renderers), `requirements/` (numbered requirements).

## Compared with a Mermaid-based renderer

Measured on 2026-11-22 against `mmdg` 0.13.0 (Mermaid 12.2.0 with an embedded JavaScript engine), on the 70 diagrams
of this repository's corpus that both programs render, on one Apple-silicon laptop. Time is the best of three runs
of the command-line program per file, process start included; memory is the peak resident size. Time and memory
are medians over the files; the output figure is the total size of the 70 files written. Both programs read the
same 70 sources (43 kB in all).

| | | gomermaidgo | mmdg[^ref] | Difference |
|---|---|---:|---:|---|
| Binary | size | 6.9 MB | 53.4 MB | 87% smaller |
| | dependencies | 0 | 10 | - |
| SVG | conversion time | 7 ms | 32 ms | 77% faster |
| | memory usage | 14 MB | 43 MB | 68% less |
| | output size[^out] | 0.34 MB | 11.8 MB | 97% smaller |
| PDF | conversion time | 9 ms | 33 ms | 73% faster |
| | memory usage | 19 MB | 55 MB | 65% less |
| | output size[^out] | 1.0 MB | 10.9 MB | 91% smaller |
| PNG | conversion time | 16 ms | 38 ms | 57% faster |
| | memory usage | 21 MB | 46 MB | 54% less |
| | output size[^out] | 2.7 MB | 2.3 MB | 16% larger |

[^ref]: `mmdg` is the reference renderer this library is compared with throughout its tests and reviews.
[^out]: Total size of the 70 files written, one per corpus diagram.

PNG is measured at scale 1 in both programs (one image pixel per drawing pixel); this library's default is scale 2.
Percentages are computed from the unrounded measurements.

The corpus is written for this library's profile, so it says nothing about coverage: Mermaid accepts syntax this
library rejects (see `docs/profile.md`), and the drawings differ on purpose. Three corpus diagrams are left out
because the other program does not render them.

## License

Apache License 2.0, see `LICENSE` and `NOTICE`. The bundled Recursive typeface is distributed under the SIL Open Font
License 1.1 (`fonts/OFL.txt`).
