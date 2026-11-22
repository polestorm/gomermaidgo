package pdf

import "github.com/polestorm/gomermaidgo/internal/csscolor"

// color czyta kolor w zapisie CSS; patrz csscolor.Parse.
func color(s string) (r, g, b float64, ok bool) { return csscolor.Parse(s) }
