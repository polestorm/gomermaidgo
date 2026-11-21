//go:build !nopdf

package main

import "github.com/polestorm/gomermaidgo/pdf"

func init() { formats[".pdf"] = pdf.Render }
