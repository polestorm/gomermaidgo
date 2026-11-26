// Polecenie bench mierzy czas, pamięć i rozmiar wyników kilku rendererów uruchamianych jako osobne programy
// na tym samym zbiorze plików. Z niego pochodzą liczby z tabeli porównania w README; metodę opisuje
// docs/benchmarks.md.
//
//	go run ./tools/bench -gomermaidgo ./gmg -mmdg mmdg -go-mermaid mermaid -json out/bench.json testdata/corpus
//
// Każdy program jest czarną skrzynką: liczy się kod wyjścia i niepusty plik wynikowy. Czas obejmuje start
// procesu. Inne programy dodaje się flagą -cmd 'nazwa:format:polecenie {in} {out}'.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// command to jedno polecenie programu dla jednego formatu.
type command struct {
	Program  string `json:"program"`
	Format   string `json:"format"`
	Template string `json:"command"`
}

// args rozwija szablon polecenia dla pliku wejściowego i wyjściowego.
func (c command) args(in, out string) []string {
	f := strings.Fields(c.Template)
	for i, a := range f {
		a = strings.ReplaceAll(a, "{in}", in)
		f[i] = strings.ReplaceAll(a, "{out}", out)
	}
	return f
}

// cmdList zbiera powtarzaną flagę -cmd.
type cmdList []command

func (l *cmdList) String() string { return fmt.Sprint(len(*l)) }

func (l *cmdList) Set(s string) error {
	p := strings.SplitN(s, ":", 3)
	if len(p) != 3 || p[0] == "" || p[1] == "" || !strings.Contains(p[2], "{in}") || !strings.Contains(p[2], "{out}") {
		return fmt.Errorf("want name:format:command with {in} and {out}, got %q", s)
	}
	*l = append(*l, command{p[0], p[1], p[2]})
	return nil
}

// Polecenia trzech porównywanych programów. PNG w skali 1 (jeden piksel obrazu na piksel rysunku),
// bo domyślne skale programów są różne.
func known(gmg, mmdg, gm string) []command {
	var c []command
	if gmg != "" {
		c = append(c,
			command{"gomermaidgo", "svg", gmg + " -q -o {out} {in}"},
			command{"gomermaidgo", "pdf", gmg + " -q -o {out} {in}"},
			command{"gomermaidgo", "png", gmg + " -q -s 1 -o {out} {in}"})
	}
	if mmdg != "" {
		c = append(c,
			command{"mmdg", "svg", mmdg + " -q -i {in} -o {out}"},
			command{"mmdg", "pdf", mmdg + " -q -i {in} -o {out}"},
			command{"mmdg", "png", mmdg + " -q -s 1 -i {in} -o {out}"})
	}
	if gm != "" {
		c = append(c,
			command{"go-mermaid", "svg", gm + " -o {out} {in}"},
			command{"go-mermaid", "png", gm + " -png -scale 1 -o {out} {in}"})
	}
	return c
}

// stat to wynik jednego programu w jednym formacie.
type stat struct {
	Program    string  `json:"program"`
	Rendered   int     `json:"rendered"`
	MedianMs   float64 `json:"median_ms,omitempty"`
	MedianMB   float64 `json:"median_peak_rss_mb,omitempty"`
	OutputMB   float64 `json:"output_mb,omitempty"`
	Unstable   int     `json:"files_differing_between_runs"`
	Binary     string  `json:"binary_sha256,omitempty"`
	Version    string  `json:"version,omitempty"`
	ms, mb     []float64
	outputSize int64
}

// formatResult to wyniki wszystkich programów w jednym formacie.
type formatResult struct {
	Format string  `json:"format"`
	Common int     `json:"common"`
	Any    int     `json:"rendered_by_any"`
	Stats  []*stat `json:"programs"`
}

// report to cały wynik przebiegu, zapisywany jako JSON obok liczb w README.
type report struct {
	Date     string         `json:"date"`
	Host     string         `json:"host"`
	Files    int            `json:"files"`
	Runs     int            `json:"runs_per_file"`
	Commands []command      `json:"commands"`
	Inputs   []string       `json:"inputs"`
	Results  []formatResult `json:"results"`
}

func main() {
	var extra cmdList
	gmg := flag.String("gomermaidgo", "", "gomermaidgo binary (build it with go build ./cmd/gomermaidgo)")
	mmdg := flag.String("mmdg", "", "mmdg binary")
	gm := flag.String("go-mermaid", "", "go-mermaid binary (its command is called mermaid)")
	flag.Var(&extra, "cmd", "extra program: name:format:command with {in} and {out}; may repeat")
	formats := flag.String("formats", "svg,pdf,png", "formats to measure, comma separated")
	runs := flag.Int("runs", 3, "timed runs per file and program; 0 only counts rendered files")
	work := flag.String("work", "", "directory for output files (default: a new temporary directory)")
	jsonOut := flag.String("json", "", "write the full result as JSON to this file")
	flag.Parse()
	inputs := flag.Args()
	if len(inputs) == 0 {
		inputs = []string{"testdata/corpus"}
	}
	files, err := collect(inputs)
	if err != nil || len(files) == 0 {
		fatal(fmt.Errorf("no .mmd files in %v: %v", inputs, err))
	}
	cmds := append(known(*gmg, *mmdg, *gm), extra...)
	if len(cmds) == 0 {
		fatal(fmt.Errorf("no programs given; see -h"))
	}
	if *work == "" {
		if *work, err = os.MkdirTemp("", "bench"); err != nil {
			fatal(err)
		}
	}
	rep := report{Date: time.Now().UTC().Format(time.RFC3339), Host: host(), Files: len(files), Runs: *runs, Commands: cmds, Inputs: inputs}
	for _, f := range strings.Split(*formats, ",") {
		var sel []command
		for _, c := range cmds {
			if c.Format == f {
				sel = append(sel, c)
			}
		}
		if len(sel) > 0 {
			rep.Results = append(rep.Results, measure(f, sel, files, *runs, *work))
		}
	}
	printMarkdown(os.Stdout, rep)
	if *jsonOut != "" {
		b, _ := json.MarshalIndent(rep, "", " ")
		if err := os.WriteFile(*jsonOut, append(b, '\n'), 0o644); err != nil {
			fatal(err)
		}
	}
}

// collect zbiera pliki .mmd z podanych plików i katalogów, w stałej kolejności.
func collect(inputs []string) ([]string, error) {
	var files []string
	for _, in := range inputs {
		err := filepath.WalkDir(in, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".mmd") {
				files = append(files, p)
			}
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

// measure: najpierw jedno przejście ustala, które pliki program narysował; czas, pamięć i rozmiar
// liczymy tylko na plikach narysowanych przez wszystkie programy danego formatu.
func measure(format string, cmds []command, files []string, runs int, work string) formatResult {
	res := formatResult{Format: format}
	ok := make([]map[string]bool, len(cmds))
	any := map[string]bool{}
	for i, c := range cmds {
		st := &stat{Program: c.Program}
		st.Binary, st.Version = identify(c)
		res.Stats = append(res.Stats, st)
		ok[i] = map[string]bool{}
		for _, f := range files {
			out := outPath(work, c, f)
			if _, _, err := run(c, f, out); err == nil {
				ok[i][f] = true
				any[f] = true
				st.Rendered++
			}
		}
	}
	res.Any = len(any)
	var common []string
	for _, f := range files {
		all := true
		for i := range cmds {
			all = all && ok[i][f]
		}
		if all {
			common = append(common, f)
		}
	}
	res.Common = len(common)
	if runs == 0 {
		return res
	}
	for i, c := range cmds {
		st := res.Stats[i]
		for _, f := range common {
			out := outPath(work, c, f)
			best, low := time.Duration(1<<62), int64(1<<62)
			var first []byte
			unstable := false
			for r := 0; r < runs; r++ {
				d, rss, err := run(c, f, out)
				if err != nil {
					fatal(fmt.Errorf("%s %s: failed on a later run: %v", c.Program, f, err))
				}
				best, low = min(best, d), min(low, rss)
				b, _ := os.ReadFile(out)
				if r == 0 {
					first = b
				} else if !bytes.Equal(first, b) {
					unstable = true
				}
			}
			if unstable {
				st.Unstable++
			}
			st.ms = append(st.ms, float64(best)/float64(time.Millisecond))
			st.mb = append(st.mb, float64(low)/1e6)
			st.outputSize += int64(len(first))
		}
		st.MedianMs = round(median(st.ms))
		st.MedianMB = round(median(st.mb))
		st.OutputMB = float64(st.outputSize/10000) / 100
	}
	return res
}

// run uruchamia polecenie raz; błąd także wtedy, gdy plik wynikowy jest pusty albo go brak.
// Zwraca czas zegarowy i szczytową pamięć rezydentną procesu w bajtach.
func run(c command, in, out string) (time.Duration, int64, error) {
	os.Remove(out)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	a := c.args(in, out)
	cmd := exec.CommandContext(ctx, a[0], a[1:]...)
	t0 := time.Now()
	err := cmd.Run()
	d := time.Since(t0)
	if err != nil {
		return d, 0, err
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		return d, 0, fmt.Errorf("no output")
	}
	return d, peakRSS(cmd.ProcessState), nil
}

// outPath daje osobny katalog na program i format, żeby wyniki można było potem obejrzeć.
func outPath(work string, c command, in string) string {
	dir := filepath.Join(work, c.Program+"-"+c.Format)
	os.MkdirAll(dir, 0o755)
	name := strings.ReplaceAll(strings.TrimSuffix(in, ".mmd"), string(filepath.Separator), "_")
	return filepath.Join(dir, name+"."+c.Format)
}

// identify zwraca sumę SHA-256 pliku programu i jego wersję, jeśli program ją podaje.
func identify(c command) (string, string) {
	bin := strings.Fields(c.Template)[0]
	p, err := exec.LookPath(bin)
	if err != nil {
		return "", ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", ""
	}
	sum := sha256.Sum256(b)
	ver := ""
	for _, flag := range []string{"--version", "-version"} {
		out, err := exec.Command(p, flag).Output()
		if err == nil && len(out) > 0 && len(out) < 200 {
			ver = strings.TrimSpace(string(out))
			break
		}
	}
	if ver == "" {
		if out, err := exec.Command("go", "version", "-m", p).Output(); err == nil {
			for _, l := range strings.Split(string(out), "\n") {
				f := strings.Fields(l)
				switch {
				case len(f) >= 3 && f[0] == "mod":
					ver = f[1] + " " + f[2]
				case len(f) >= 2 && f[0] == "build" && strings.HasPrefix(f[1], "vcs.revision="):
					ver += " " + strings.TrimPrefix(f[1], "vcs.revision=")[:12]
				case len(f) >= 2 && f[0] == "build" && f[1] == "vcs.modified=true":
					ver += " (modified)"
				}
			}
		}
	}
	return hex.EncodeToString(sum[:]), ver
}

// host opisuje maszynę: system, architekturę, procesor i liczbę rdzeni.
func host() string {
	cpu := ""
	switch runtime.GOOS {
	case "darwin":
		if b, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			cpu = strings.TrimSpace(string(b))
		}
	case "linux":
		if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			for _, l := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(l, "model name") {
					cpu = strings.TrimSpace(l[strings.Index(l, ":")+1:])
					break
				}
			}
		}
	}
	return fmt.Sprintf("%s/%s, %s, %d cores", runtime.GOOS, runtime.GOARCH, cpu, runtime.NumCPU())
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	if n := len(s); n%2 == 1 {
		return s[n/2]
	} else {
		return (s[n/2-1] + s[n/2]) / 2
	}
}

func round(v float64) float64 { return float64(int64(v*10+0.5)) / 10 }

// printMarkdown wypisuje wynik jako tabele gotowe do wklejenia.
func printMarkdown(w *os.File, r report) {
	fmt.Fprintf(w, "%s, %s, %d files, best of %d runs per file\n\n", r.Date, r.Host, r.Files, r.Runs)
	for _, fr := range r.Results {
		fmt.Fprintf(w, "%s: %d files rendered by all, %d by at least one\n\n", strings.ToUpper(fr.Format), fr.Common, fr.Any)
		fmt.Fprintln(w, "| program | rendered | median time ms | median peak memory MB | output MB | files differing between runs |")
		fmt.Fprintln(w, "|---|---:|---:|---:|---:|---:|")
		for _, s := range fr.Stats {
			fmt.Fprintf(w, "| %s | %d | %.1f | %.1f | %.2f | %d |\n", s.Program, s.Rendered, s.MedianMs, s.MedianMB, s.OutputMB, s.Unstable)
		}
		fmt.Fprintln(w)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "bench:", err)
	os.Exit(1)
}
