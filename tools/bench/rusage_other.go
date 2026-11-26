//go:build !unix

package main

import "os"

// peakRSS: poza systemami uniksowymi pamięci nie mierzymy.
func peakRSS(*os.ProcessState) int64 { return 0 }
