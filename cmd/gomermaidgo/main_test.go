package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// REQ: PDF-005
// Rozszerzenie pliku wyniku wybiera format: .pdf daje dokument PDF, pozostałe SVG.
func TestOutputFormatByExtension(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "d.mmd")
	if err := os.WriteFile(in, []byte("flowchart TD\nA --> B"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"d.svg": "<svg ", "d.txt": "<svg "}
	if _, ok := formats[".pdf"]; ok {
		want["d.pdf"], want["D.PDF"] = "%PDF-", "%PDF-"
	} else {
		want["d.pdf"] = "<svg " // program zbudowany ze znacznikiem nopdf
	}
	for name, prefix := range want {
		out := filepath.Join(dir, name)
		if err := run("", "", 16, "", out, in, nil); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		data, _ := os.ReadFile(out)
		if !bytes.HasPrefix(data, []byte(prefix)) {
			t.Errorf("%s starts with %q, want %q", name, data[:5], prefix)
		}
	}
}
