# gomermaidgo

Renders diagrams written in Mermaid syntax to SVG in pure Go: no JavaScript, no browser, no dependencies outside the
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

Status: early. Supported diagram types: `sequenceDiagram`, `flowchart`/`graph` and `stateDiagram-v2`. Graphs use a
layered layout written from the published algorithms; its current limits are listed in `docs/profile.md`. A construct outside the supported profile (`docs/profile.md`) is an error with a line number,
never a partial drawing.

The output uses only basic SVG 1.1 elements (`rect`, `line`, `polyline`, `polygon`, `circle`, `ellipse`, `text` and
`path` restricted to M, L and C commands) with presentation
attributes: no `foreignObject`, no style sheets, no scripts, no links, no ids.

Documentation (in Polish): `docs/profile.md` (supported syntax), `docs/provenance.md` (sources and clean-room rules),
`docs/testing.md` (corpus, invariants, comparison with reference renderers), `requirements/` (numbered requirements).
