//go:build !nopng

package main

import (
	"github.com/polestorm/gomermaidgo/mermaid"
	"github.com/polestorm/gomermaidgo/png"
)

func init() {
	formats[".png"] = func(source string, opts ...mermaid.Option) ([]byte, error) {
		if scale > 0 {
			return png.RenderScale(source, scale, opts...)
		}
		return png.Render(source, opts...)
	}
}
