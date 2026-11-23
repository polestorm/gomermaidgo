package mermaid

import (
	"github.com/polestorm/gomermaidgo/internal/architecture"
	"github.com/polestorm/gomermaidgo/internal/block"
	"github.com/polestorm/gomermaidgo/internal/gitgraph"
	"github.com/polestorm/gomermaidgo/internal/ishikawa"
	"github.com/polestorm/gomermaidgo/internal/kanban"
	"github.com/polestorm/gomermaidgo/internal/packet"
	"github.com/polestorm/gomermaidgo/internal/radar"
	"github.com/polestorm/gomermaidgo/internal/sankey"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/treemap"
)

// Typy diagramów dodane po pierwszych dwunastu.
const (
	TypeKanban       = "kanban"
	TypePacket       = "packet"
	TypeRadar        = "radar"
	TypeTreemap      = "treemap"
	TypeSankey       = "sankey"
	TypeGit          = "gitGraph"
	TypeBlock        = "block"
	TypeArchitecture = "architecture"
	TypeIshikawa     = "ishikawa"
)

// layoutOptions to ustawienia przekazywane układowi typu diagramu.
type layoutOptions struct {
	Measure    func(text string, size float64) float64
	FontSize   float64
	FontFamily string
	Background string
	Colors     bool
	Shadows    bool
}

// diagramType opisuje typ diagramu rysowany wprost do sceny: nazwę, słowa nagłówka, po których
// jest rozpoznawany, i drogę od źródła do sceny.
type diagramType struct {
	name    string
	headers []string
	scene   func(source string, o layoutOptions) (*scene.Scene, error)
}

var moreTypes = []diagramType{
	{TypeKanban, []string{"kanban"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		b, err := kanban.Parse(source)
		if err != nil {
			return nil, err
		}
		return kanban.Layout(b, kanban.Options(o)), nil
	}},
	{TypePacket, []string{"packet-beta", "packet"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		d, err := packet.Parse(source)
		if err != nil {
			return nil, err
		}
		return packet.Layout(d, packet.Options(o)), nil
	}},
	{TypeRadar, []string{"radar-beta", "radar"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		c, err := radar.Parse(source)
		if err != nil {
			return nil, err
		}
		return radar.Layout(c, radar.Options(o)), nil
	}},
	{TypeArchitecture, []string{"architecture-beta", "architecture"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		d, err := architecture.Parse(source)
		if err != nil {
			return nil, err
		}
		return architecture.Layout(d, architecture.Options(o)), nil
	}},
	{TypeBlock, []string{"block-beta", "block"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		d, err := block.Parse(source)
		if err != nil {
			return nil, err
		}
		return block.Layout(d, block.Options(o)), nil
	}},
	{TypeGit, []string{"gitGraph", "gitGraph:"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		g, err := gitgraph.Parse(source)
		if err != nil {
			return nil, err
		}
		return gitgraph.Layout(g, gitgraph.Options(o)), nil
	}},
	{TypeSankey, []string{"sankey-beta", "sankey"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		c, err := sankey.Parse(source)
		if err != nil {
			return nil, err
		}
		return sankey.Layout(c, sankey.Options(o)), nil
	}},
	{TypeIshikawa, []string{"ishikawa-beta", "ishikawa"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		d, err := ishikawa.Parse(source)
		if err != nil {
			return nil, err
		}
		return ishikawa.Layout(d, ishikawa.Options(o)), nil
	}},
	{TypeTreemap, []string{"treemap-beta", "treemap"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		d, err := treemap.Parse(source)
		if err != nil {
			return nil, err
		}
		return treemap.Layout(d, treemap.Options(o)), nil
	}},
}
