//go:build unix

package main

import (
	"os"
	"runtime"
	"syscall"
)

// peakRSS zwraca szczytową pamięć rezydentną zakończonego procesu w bajtach. macOS podaje ją w bajtach,
// Linux i pozostałe systemy uniksowe w kilobajtach.
func peakRSS(ps *os.ProcessState) int64 {
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return int64(ru.Maxrss)
	}
	return int64(ru.Maxrss) * 1024
}
