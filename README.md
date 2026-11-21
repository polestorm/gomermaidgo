# gomermaidgo

Renders diagrams written in Mermaid syntax to SVG in pure Go: no JavaScript, no browser, no dependencies outside the
Go standard library. The same source and options always give the same SVG, byte for byte.

```go
svg, err := gomermaidgo.Render(source,
	gomermaidgo.WithFont(fontBytes),         // measure text with the font that will draw it
	gomermaidgo.WithFontFamily("Noto Sans"),
)
```

```sh
go run ./cmd/gomermaidgo -font NotoSans-Regular.ttf -family "Noto Sans" -o diagram.svg diagram.mmd
```

Status: early. `sequenceDiagram` is rendered. `flowchart`/`graph` and `stateDiagram-v2` are parsed and validated
(`Check`), but not drawn yet: `Render` returns `ErrLayoutPending` until the layered layout exists (see `docs/roadmap.md`). A construct outside the supported profile (`docs/profile.md`) is an error with a line number,
never a partial drawing.

The output uses only basic SVG 1.1 elements (`rect`, `line`, `polyline`, `polygon`, `circle`, `text`) with presentation
attributes: no `foreignObject`, no style sheets, no scripts, no links, no ids.

Documentation (in Polish): `docs/profile.md` (supported syntax), `docs/provenance.md` (sources and clean-room rules),
`docs/testing.md` (corpus, invariants, comparison with reference renderers), `requirements/` (numbered requirements).
