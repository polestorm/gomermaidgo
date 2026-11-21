package gomermaidgo

import (
	"github.com/polestorm/gomermaidgo/internal/kanban"
	"github.com/polestorm/gomermaidgo/internal/packet"
	"github.com/polestorm/gomermaidgo/internal/radar"
	"github.com/polestorm/gomermaidgo/internal/scene"
	"github.com/polestorm/gomermaidgo/internal/treemap"
)

// Typy diagramów dodane po pierwszych dwunastu.
const (
	TypeKanban  = "kanban"
	TypePacket  = "packet"
	TypeRadar   = "radar"
	TypeTreemap = "treemap"
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
	{TypeTreemap, []string{"treemap-beta", "treemap"}, func(source string, o layoutOptions) (*scene.Scene, error) {
		d, err := treemap.Parse(source)
		if err != nil {
			return nil, err
		}
		return treemap.Layout(d, treemap.Options(o)), nil
	}},
}
