# How the comparison is measured

Every number in the [comparison table](../README.md#comparison) can be measured again with the tools and inputs
in this repository. This page lists the inputs, the programs and their versions, the exact commands, what each
number means and how to repeat the run. The raw results behind the README are in [`benchmarks/`](benchmarks/).

## Inputs

| Set | Where | Used for |
|---|---|---|
| Corpus: 90 diagrams of 23 types | [`testdata/corpus/`](../testdata/corpus/), in the repository | time, memory, output size, determinism |
| Public diagrams: 168 blocks from 53 Markdown files in 38 public GitHub repositories | manifest [`testdata/wild/sources.tsv`](../testdata/wild/sources.tsv); fetched by `go run ./tools/wildcorpus` | coverage ("Public diagrams") |

The corpus is written for this library's syntax profile, so it is not used to compare how many diagrams each
program accepts. Coverage is compared only on the public diagrams.

The public diagrams belong to their authors, so the repository does not copy them. The manifest lists the following
for every block:

- the repository;
- the commit, pinned to the version that was read;
- the file path;
- the line of the opening ```` ```mermaid ```` fence;
- the SHA-256 of the block.

`tools/wildcorpus` downloads each file at its commit from `raw.githubusercontent.com` and cuts out the block. It then
checks the checksum and fails on any mismatch.

How the set was collected:

- On 2026-11-21 and 2026-11-22, GitHub code search was run for `` ```mermaid <keyword> `` in `.md` files.
  - The keywords for the first batch were: `sequenceDiagram`, `stateDiagram-v2`, `erDiagram`, `flowchart TD`,
    `flowchart LR`, `graph TD` and `gantt dateFormat`.
  - The keywords for the second batch were: `classDiagram`, `gitGraph`, `pie title`, `mindmap`, `timeline`, `journey`,
    `quadrantChart`, `xychart-beta`, `requirementDiagram`, `C4Context`, `C4Container`, `kanban`, `architecture-beta`,
    `block-beta`, `sankey-beta`, `packet-beta`, `radar-beta` and `treemap-beta`.
- Every distinct `mermaid` block in the files found was kept, as written, including fragments that are not diagrams.

The first collection had 175 blocks. Seven of them, from four files, are no longer on GitHub, so they are not in the
manifest. All three programs had rejected all seven, so the coverage counts are the same with or without them.

## Programs

| Program | How to get it | Version measured |
|---|---|---|
| gomermaidgo | `go build -o gomermaidgo ./cmd/gomermaidgo` in this repository | commit `3b3170a`, go1.26.6 |
| mmdg | `go install github.com/bvolpato/mermaid-go-renderer/cmd/mmdg@v0.14.1` | 0.14.1 (Mermaid 12.2.0); `go install` fetched the go1.27.4 toolchain its `go.mod` asks for |
| go-mermaid | `go install github.com/zkrebbekx/go-mermaid/cmd/mermaid@v0.1.6` | 0.1.6, go1.26.6 |

The other two programs are only run as programs; their code is not read (see [`provenance.md`](provenance.md)).
Each result file records the SHA-256 and module version of every binary that was measured.

## Commands

`tools/bench` runs exactly these commands. `{in}` is the source file and `{out}` the output file; the format follows
from the file extension.

| Program | SVG and PDF | PNG |
|---|---|---|
| gomermaidgo | `gomermaidgo -q -o {out} {in}` | `gomermaidgo -q -s 1 -o {out} {in}` |
| mmdg | `mmdg -q -i {in} -o {out}` | `mmdg -q -s 1 -i {in} -o {out}` |
| go-mermaid | `mermaid -o {out} {in}` (SVG only, no PDF) | `mermaid -png -scale 1 -o {out} {in}` |

PNG is compared at scale 1, one image pixel per drawing pixel, because the programs have different default scales.
To add another renderer, use `-cmd 'name:format:command {in} {out}'`.

## What the numbers mean

- **Rendered**: the program exits with status 0 and writes a non-empty file. How well the diagram is drawn is not
  judged here.
- **Common set**: time, memory and output size are taken only on the files that every program in that format
  rendered. On the corpus that is 63 files for SVG and PNG, and 87 for PDF, which go-mermaid does not write.
- **Time**: wall-clock time of the whole process, so process start is included. For mmdg that includes starting its
  embedded JavaScript engine. Each file is run 3 times and the best run counts; the table shows the median over files.
- **Memory**: the peak resident set size of the process, as reported by `getrusage` for the finished child. The
  lowest of the 3 runs counts; the table shows the median over files.
- **Output size**: the total size of the files written for the common set.
- **Deterministic output**: the 3 runs of every file give byte-identical files. This held for all three programs,
  in all formats.
- **Dependencies**: Go modules linked into the program, that is, the `dep` lines of `go version -m <binary>`.
  **Program size** is the binary built with the commands above, without extra flags.
- **Diagram types** come from each project's documentation and its `-list-types` output. **GitHub popularity** comes
  from `gh api repos/<owner>/<repo>` on the date given.

## Repeating the run

```sh
git clone https://github.com/polestorm/gomermaidgo && cd gomermaidgo
mkdir -p bin && go build -o bin/gomermaidgo ./cmd/gomermaidgo
GOBIN=$PWD/bin go install github.com/bvolpato/mermaid-go-renderer/cmd/mmdg@v0.14.1
GOBIN=$PWD/bin go install github.com/zkrebbekx/go-mermaid/cmd/mermaid@v0.1.6
export PATH=$PWD/bin:$PATH

# time, memory, size and determinism on the corpus (about 8 minutes)
go run ./tools/bench -gomermaidgo gomermaidgo -mmdg mmdg -go-mermaid mermaid -json out/corpus.json testdata/corpus

# coverage on the public diagrams
go run ./tools/wildcorpus -o out/wild
go run ./tools/bench -gomermaidgo gomermaidgo -mmdg mmdg -go-mermaid mermaid -formats svg -runs 0 -json out/public.json out/wild
```

The tool prints Markdown tables and, with `-json`, writes the full result: commands, binary checksums, versions,
host, and per-format counts and medians.

## Published run

The numbers in the README come from these two files, measured on 2026-11-26:

- [`benchmarks/2026-11-26-corpus.json`](benchmarks/2026-11-26-corpus.json);
- [`benchmarks/2026-11-26-public.json`](benchmarks/2026-11-26-public.json).

The run used macOS on an Apple M4 laptop with 10 cores and 32 GB of RAM, on mains power, with other programs running
(load average 2–3).

Absolute times move with the machine and its load. For example, go-mermaid took 3.9 ms per SVG in a run on
2026-11-23 and 4.9 ms in this one. Compare the ratios between programs in the same run, not times from different
runs.
