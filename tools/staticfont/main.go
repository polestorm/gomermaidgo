// Polecenie staticfont zapisuje krój zmienny jako statyczny w jego odmianie domyślnej: usuwa
// tabele odmian (fvar, gvar, avar, HVAR, VVAR, MVAR, STAT, cvar) oraz tabele, których rysowanie
// konturów nie potrzebuje (GSUB, GPOS, GDEF, vhea, vmtx). Kontury, odwzorowanie znaków i metryki
// zostają bez zmian. Tak powstał dołączony do modułu plik fonts/NotoEmoji-Regular.ttf.
//
//	go run ./tools/staticfont wejście.ttf wyjście.ttf
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

var drop = map[string]bool{"fvar": true, "gvar": true, "avar": true, "HVAR": true, "VVAR": true, "MVAR": true, "STAT": true, "cvar": true,
	"GSUB": true, "GPOS": true, "GDEF": true, "vhea": true, "vmtx": true}

func checksum(b []byte) uint32 {
	var sum uint32
	for i := 0; i+4 <= len(b); i += 4 {
		sum += binary.BigEndian.Uint32(b[i:])
	}
	return sum
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: staticfont in.ttf out.ttf")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil || len(data) < 12 {
		fmt.Fprintln(os.Stderr, "staticfont: cannot read the font:", err)
		os.Exit(1)
	}
	type table struct {
		tag  string
		body []byte
	}
	var tables []table
	for i, n := 0, int(binary.BigEndian.Uint16(data[4:])); i < n; i++ {
		rec := data[12+16*i:]
		tag := string(rec[:4])
		off, length := int(binary.BigEndian.Uint32(rec[8:])), int(binary.BigEndian.Uint32(rec[12:]))
		if off+length > len(data) {
			fmt.Fprintln(os.Stderr, "staticfont: table", tag, "lies outside the file")
			os.Exit(1)
		}
		if !drop[tag] {
			body := append([]byte(nil), data[off:off+length]...)
			if tag == "head" {
				binary.BigEndian.PutUint32(body[8:], 0) // suma całego pliku: liczona na końcu
			}
			tables = append(tables, table{tag, body})
		}
	}
	sort.Slice(tables, func(a, b int) bool { return tables[a].tag < tables[b].tag })
	n := len(tables)
	search, selector := 1, 0
	for search*2 <= n {
		search, selector = search*2, selector+1
	}
	out := make([]byte, 12+16*n)
	copy(out, data[:4])
	binary.BigEndian.PutUint16(out[4:], uint16(n))
	binary.BigEndian.PutUint16(out[6:], uint16(search*16))
	binary.BigEndian.PutUint16(out[8:], uint16(selector))
	binary.BigEndian.PutUint16(out[10:], uint16(n*16-search*16))
	head := -1
	for i, t := range tables {
		padded := append(append([]byte(nil), t.body...), make([]byte, (4-len(t.body)%4)%4)...)
		rec := out[12+16*i:]
		copy(rec, t.tag)
		binary.BigEndian.PutUint32(rec[4:], checksum(padded))
		binary.BigEndian.PutUint32(rec[8:], uint32(len(out)))
		binary.BigEndian.PutUint32(rec[12:], uint32(len(t.body)))
		if t.tag == "head" {
			head = len(out)
		}
		out = append(out, padded...)
	}
	if head < 0 {
		fmt.Fprintln(os.Stderr, "staticfont: the font has no head table")
		os.Exit(1)
	}
	binary.BigEndian.PutUint32(out[head+8:], 0xB1B0AFBA-checksum(out))
	if err := os.WriteFile(os.Args[2], out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "staticfont:", err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d tables, %d bytes (from %d)\n", os.Args[2], n, len(out), len(data))
}
