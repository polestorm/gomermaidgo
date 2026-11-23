# gomermaidgo

Renders diagrams written in Mermaid syntax to SVG (and optionally PDF and PNG) in pure Go: no JavaScript, no browser, no dependencies outside the
Go standard library. The same source and options always give the same SVG, byte for byte.

```go
import "github.com/polestorm/gomermaidgo/mermaid"

svg, err := mermaid.Render(source)
```

## Why gomermaidgo

Compared with [`mmdg`](https://github.com/bvolpato/mermaid-go-renderer), the closest alternative (details in
[At a glance](#at-a-glance)):

- **6.5× smaller program, no JavaScript engine, no dependencies**: 8.2 MB against 53 MB, standard library only.
- **2–4× faster, about half the memory**: an SVG in 9 ms instead of 38 ms, a PDF 3× and a PNG 2× faster.
- **Fits diagrams to an A4 page**: narrows fields, folds long processes into rows, folds long chains and staggers
  long ranks, so a printed diagram stays readable ([examples](docs/gallery/page-fitting.md)).
- **32× smaller SVG, 8× smaller PDF**: plain SVG 1.1 without embedded fonts, CSS or scripts; PDF with real,
  searchable text.
- **Made for documents**: a neutral look that prints well in black and white, Markdown documents with embedded
  diagrams, and the flags of `mermaid-cli`.

## Contents

- [Why gomermaidgo](#why-gomermaidgo) · [At a glance](#at-a-glance) · [Examples](#examples) ·
  [Diagram types](#diagram-types) · [Documentation](#documentation) ·
  [How the comparison was measured](#how-the-comparison-was-measured) · [License](#license)
- [Usage](docs/usage.md): fonts, colours and themes, fitting to A4, PDF, PNG, command line, Markdown
- [Gallery](docs/gallery/README.md): [page fitting](docs/gallery/page-fitting.md),
  [all diagram types](docs/gallery/types.md), [scripts and symbols](docs/gallery/scripts.md)
- [Supported syntax](docs/profile.md) (in Polish)

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
| Diagram types[^types] | 24 of Mermaid's 33 | 33 |
| Syntax coverage | a documented profile; anything outside it is an error with a line number | everything Mermaid accepts |
| This repository's corpus (89 diagrams) | 89 | 86 |
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
[^bench]: Median over the 79 corpus diagrams both programs rendered at the time of measuring, best of three runs of the command-line program
    per file, process start included; memory is the peak resident size. Measured on 2026-11-23 on one Apple-silicon
    laptop.
[^types]: Mermaid 12.2 recognises 33 diagram types. Not drawn here: `zenuml` (a separate Mermaid add-on) and eight
    types added in Mermaid 12 as beta: `venn`, `treeView`, `wardley`, `cynefin`, `railroad`, `eventmodeling`,
    `usecase` and `agentflow`.
[^wild]: 175 blocks marked `mermaid` collected from public GitHub repositories; 36 are rejected by both programs
    (fragments without a diagram header, or not Mermaid at all), which leaves 139.

## Examples

A 13-step process written left to right. Mermaid draws it as one strip, 16 times wider than tall; this library
folds it into rows by default, so it fits an A4 page at about four times the scale:

![A supplier approval process folded into three rows](docs/gallery/img/layout/fold.svg)

<table>
<tr>
<td width="50%"><img src="docs/gallery/img/layout/fit-fold.svg" alt="A nonconforming product process, fitted and folded"></td>
<td width="50%"><img src="docs/gallery/img/types/02-sequence.svg" alt="A document approval sequence diagram"></td>
</tr>
<tr>
<td>Long labels and a wide branching: fields are narrowed (text wraps more often) and the diagram is folded,
from 1770 × 404 to 631 × 756.</td>
<td>A sequence diagram with numbering, a loop, alternatives, a highlighted block and a note.</td>
</tr>
</table>

More in the [gallery](docs/gallery/README.md): [page fitting](docs/gallery/page-fitting.md) with and without each
option, [all 24 diagram types](docs/gallery/types.md), and [scripts and symbols](docs/gallery/scripts.md).

## Diagram types

Status: early. Drawn: `flowchart`/`graph`, `sequenceDiagram`, `classDiagram`, `stateDiagram-v2`, `erDiagram`,
`gantt`, `pie`, `journey`, `gitGraph`, `mindmap`, `timeline`, `quadrantChart`, `xychart-beta`, `sankey-beta`,
`requirementDiagram`, C4 diagrams, `block-beta`, `packet-beta`, `kanban`, `architecture-beta`, `radar-beta`,
`treemap-beta`, `ishikawa-beta` and `swimlane-beta`; one example of each is in the
[gallery](docs/gallery/types.md). Graphs use a layered layout written from the published algorithms. The supported
syntax and its current limits are listed in [`docs/profile.md`](docs/profile.md); a construct outside it is an error
with a line number, never a partial drawing.

## Documentation

- [`docs/usage.md`](docs/usage.md): how to use the library and the command.
- [`docs/gallery/`](docs/gallery/README.md): pictures with their sources.
- In Polish: [`docs/profile.md`](docs/profile.md) (supported syntax), [`docs/provenance.md`](docs/provenance.md)
  (sources and clean-room rules), [`docs/testing.md`](docs/testing.md) (corpus, invariants, comparison with
  reference renderers), [`requirements/`](requirements/) (numbered requirements), [`docs/adr/`](docs/adr/)
  (decisions).

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
