# Data Pipelines with Concurrency — a Go image pipeline, with and without goroutines

**Northwestern MSDS 431 · Week 6 · Luis Varela**

This repository replicates and extends Amrit Singh's (CODEHEIM) Go image-processing pipeline
([code](https://github.com/code-heim/go_21_goroutines_pipeline), Apache-2.0). Photos go through four stages:
**load → resize → grayscale → save**. The program can run them three ways:

| Mode | How the work is scheduled | Goroutines |
|---|---|---|
| `sequential` | one photo at a time, all four stages, then the next photo | none |
| `pipeline` | CODEHEIM's design: one goroutine per stage, joined by channels | 4 |
| `workers` | a pool of goroutines, each running all four stages on one photo | one per CPU (adjustable) |

All three modes call **the same stage functions**, and a test proves they produce **byte-identical output files**,
so the time differences below measure concurrency and nothing else.

> **The question from management:** can Go concurrency reduce data pipeline processing times?
> **Short answer:** yes, but *how* the work is split between goroutines decides how much. See the
> [recommendation](#recommendation-to-management).

---

## Quick start

Requires Go 1.22 or newer (developed with Go 1.27 on macOS).

```bash
git clone https://github.com/mioluigi/go-image-pipeline.git
cd go-image-pipeline
go run .                     # the four built-in photos, pipeline mode → ./output/
go run . -mode sequential    # the same without goroutines
go test ./...                # every test
```

---

## What changed from CODEHEIM's original

CODEHEIM's three commits are kept at the start of this repository's history (`git log`), followed by ours,
one task per commit. The original builds and runs unchanged (results table 1 below), but
[Spike 001](docs/spikes/001-original-pipeline.md) found these problems by running it:

| Original behaviour | Now |
|---|---|
| Any bad file (missing, corrupt, PNG) **panics** inside a goroutine and the other images are lost. | Each failure is reported by file name; the other images are still saved; exit code 1. |
| The output folder must already exist, or the program panics. | Created if missing. |
| An input outside `images/` was **overwritten** by its own grayscale result (the output path came from `strings.Replace`). | Outputs go to a separate folder; the program refuses an output folder that is the input folder (even through a symbolic link). |
| Every photo stretched to 500 × 500. | **Aspect ratio kept** (optional assignment item): the longer side becomes 500. Square photos still give 500 × 500. |
| JPEG only. | JPEG and PNG in; JPEG out. |
| Paths fixed in the code; one way to run. | Flags for mode, folders, size and worker count; input checked before any work starts. |
| Only the pipeline design. | Also `sequential` (no goroutines, the baseline) and `workers` (a pool). |
| No tests or measurements. | Unit and end-to-end tests, race detector, benchmarks, stage timing, profiles, whole-program measurements. |

The resize library (`nfnt/resize`, Lanczos3) and the grayscale algorithm are deliberately **unchanged**: they are
the workload being measured.

---

## Using the program

```
go-image-pipeline [options]

  -mode sequential|pipeline|workers   how to schedule the work (default pipeline)
  -in FOLDER        JPEG and PNG photos to convert (default: the photos built into the program)
  -out FOLDER       where to save the grayscale JPEGs; created if missing (default output)
  -size N           outputs fit inside N × N pixels, keeping their shape (default 500; 1–10000)
  -workers N        goroutines in workers mode (default: one per CPU; 1–1024)
  -verbose          log every photo's stage times to standard error
```

Example, from a development run with a corrupt file in the folder (2-CPU workspace):

```
$ go-image-pipeline -mode workers -in in -out gray
FAILED  corrupt.jpeg  unsupported or corrupt image (JPEG and PNG are accepted): image: unknown format
saved   drawing.png   1536x1536 -> 500x500  gray/drawing.jpg
saved   image1.jpeg   1536x1536 -> 500x500  gray/image1.jpg
saved   wide.jpeg     1536x768 -> 500x250   gray/wide.jpg

3 saved, 1 failed in 0.22 s (mode workers, 2 workers)
Time in each stage, summed over images: load 0.19 s, resize 0.19 s, grayscale 0.02 s, save 0.02 s
(Stages overlap when goroutines are used, so these sums can exceed the total time.)
Output folder: /tmp/spike/demo/gray
```

| Exit code | Meaning |
|---|---|
| 0 | every photo saved |
| 1 | at least one photo failed; the others were saved |
| 2 | invalid usage (bad flag, missing or empty input folder, output folder = input folder, two inputs with the same output name); nothing processed |

Results are printed in input order in every mode. Logs (timestamped, `log/slog`) go to standard error, so
`2>/dev/null` hides them.

---

## Roles of programs and data

| Path | Role |
|---|---|
| `main.go`, `report.go`, `images.go` | The program: flags and validation, the printed report, the built-in photos (embedded with `go:embed`, so executables run on their own). |
| `imageprocessing/` | **Reusable package**, one image at a time: `Decode` (JPEG, PNG), `FitSize`/`Fit` (resize keeping the ratio), `Grayscale`, `Encode`. Reads and writes `io.Reader`/`io.Writer`; never touches the file system. |
| `pipeline/` | **Reusable package**, many images: `Run` in one of the three modes, results in input order, per-stage times; reads from any `fs.FS`. |
| `images/` | Four of the author's own photos (replacing CODEHEIM's), built into the program. |
| `cmd/measure/` | Runs whole programs repeatedly and reports median wall time and peak memory (reused from Weeks 4–5). |
| `scripts/` | `run-original.sh` (CODEHEIM's program, unchanged), `measure.sh` (every measurement), `tables.sh` + `readme.sh` (the tables below), `build.sh` + `build-macos-app.sh` (executables). |
| `results/` | Raw measured output from the author's Mac; the tables below are generated from it. |
| `docs/`, `specs/` | The development record: constitution, assignment summary, spike, spec, plan, tasks, reviews, decisions, AI dialogs. |

### How the concurrent modes stay correct

- **A failure travels with its photo.** The stage that fails records the error in the photo's job; later stages
  pass that job through untouched. No stage ever stops reading early, so every goroutine ends when its input
  channel closes — none is left blocked (a test counts goroutines before and after every mode, with failing photos).
- **Each channel has one sender, which closes it.** In the pipeline, closing spreads from the first stage to the
  last. In the pool, a `sync.WaitGroup` closes the results channel after the last worker finishes.
- **Order:** photos finish in any order in the concurrent modes; each carries its input position, and `Run`
  puts results back in input order.
- **No shared mutable state:** jobs are passed by value through channels; results are collected in one goroutine.

---

## Testing

```bash
go test ./...                                    # all tests
go test -race ./...                              # with the race detector
go test -cover ./imageprocessing/ ./pipeline/    # coverage of the reusable packages
go vet ./... && gofmt -l .                       # lint (gofmt -l prints nothing)
```

| Level | What is tested | Test data |
|---|---|---|
| `imageprocessing` | Grey values against independently computed BT.601 luma; `FitSize` for square, landscape, portrait, small, extreme and empty sizes; JPEG and PNG decoding; empty, garbage, GIF, truncated files; write errors | Synthetic images built in the tests |
| `pipeline` | **Byte-identical output in every mode**; failures don't stop others; input order; no goroutine left behind; save errors; worker counts; bad setup | In-memory folders (`fstest.MapFS`) of random-pixel images with fixed seeds |
| program | No arguments; every mode; 15 invalid-usage cases (exit 2); a corrupt file (exit 1); **input files unchanged after a run**; `-verbose`; `-h` | Temporary folders |

**Mutation checks:** important tests were confirmed to fail by deliberately breaking the code (results out of
order, error pass-through removed, channels left open, early exit from the collector, output-equals-input check
removed, and others) — 20 of 20 behavioural mutations caught; details in the commit messages for T04, T07 and T10.

---

## Results

`scripts/measure.sh` runs every measurement and saves raw output in `results/`; `scripts/tables.sh` builds
these tables; `scripts/readme.sh` copies them here. **No number below was typed by hand.**

<!-- tables:start -->
*(Tables appear here after `scripts/measure.sh` has been run on the Mac — task T12.)*
<!-- tables:end -->

To reproduce (about ten minutes; plugged in, other programs closed):

```bash
scripts/measure.sh
scripts/tables.sh > results/tables.md && scripts/readme.sh
```

Benchmarks alone: `go test -run '^$' -bench . -benchmem ./pipeline/`
(`BenchmarkRun`: synthetic 1536 × 1024 photos at 4, 16 and 64 images; `BenchmarkRunBundled`: the built-in photos;
each reports time per run, **images per second**, and allocations).
Profiles: `results/cpu-profile.txt` and `results/memory-profile.txt` (`go tool pprof -top`, sequential mode).

---

## Why the pipeline's speed-up has a ceiling

With one goroutine per stage, at best all four stages work at once on different photos. Then a photo leaves the
pipeline every time the **slowest** stage finishes one, so throughput can never exceed

> speed-up ≤ (total time of all stages) ÷ (time of the slowest stage)

Resize is the slowest stage by far (table 4), so the pipeline's ceiling is computed there from measured stage
times. **More CPUs do not raise it**: the pipeline has only four goroutines, and three of them spend most of
their time waiting for resize. The worker pool has no such ceiling — each worker does all the stages for its
own photo — so it keeps scaling until the CPUs (or memory and disk) run out. The price is memory: a pool holds
one full-size decoded photo per worker at once (table 5, peak memory).

---

## Recommendation to management

<!-- Final wording after T12, from the measured tables. -->

1. **Use Go concurrency for this kind of work — it pays.** Image conversion (and most per-record data
   transformation) is independent work per item, which goroutines parallelize with little code.
2. **Prefer a worker pool over a stage-per-goroutine pipeline** when items are independent. The pipeline is
   capped by its slowest stage, however many CPUs the server has; the pool scales with cores.
3. **Use stages when they are genuinely different resources**, e.g. one stage waits on the network or disk and
   another uses the CPU, or a stage must run in order. Then size each stage separately (more goroutines for the
   slow stage) instead of one each.
4. **Budget memory with the worker count.** Each worker holds a full photo in memory; cap the pool to fit the
   machine.
5. **Make the concurrent version prove itself:** identical output to a sequential run, the race detector, and a
   test that no goroutine is left behind — the three checks this project uses.

---

## Building executables

```bash
scripts/build.sh
```

| File | For | How to run |
|---|---|---|
| `dist/go-image-pipeline` | this computer | `dist/go-image-pipeline -h` |
| `dist/go-image-pipeline.exe` | Windows, 64-bit | from Command Prompt: `go-image-pipeline.exe` (a double-click closes the window when it finishes) |
| `dist/ImagePipeline.app` | macOS, Apple Silicon | double-click; Terminal opens and runs all three modes on the built-in photos, saving to `~/Desktop/ImagePipeline-output` |

The program uses no C code, so Go cross-compiles all three from any computer. The photos are embedded, so each
executable works on its own.

---

## Limitations

- **Phone-photo rotation (EXIF orientation) is not applied**, as in the original: Go's standard library ignores
  it. The bundled photos were exported upright from Photos; a sideways phone photo may come out sideways.
- Timings come from one Mac; other machines will differ, especially in the worker pool's speed-up (it depends on
  CPU count).
- Only JPEG and PNG input; output is always JPEG.

---

## How this was built (Spec-First, Evidence-Driven Development)

| Document | Purpose |
|---|---|
| [`docs/constitution.md`](docs/constitution.md) | Non-negotiable rules derived from the rubric |
| [`docs/assignment-and-rubric.md`](docs/assignment-and-rubric.md) | The assignment in our own words |
| [`docs/spikes/001-original-pipeline.md`](docs/spikes/001-original-pipeline.md) | Running the original before planning: failure modes and the speed-up ceiling |
| [`specs/001-concurrent-image-pipeline/spec.md`](specs/001-concurrent-image-pipeline/spec.md) | What and why, with acceptance criteria and traceability to the rubric and the assignment steps |
| [`plan.md`](specs/001-concurrent-image-pipeline/plan.md), [`tasks.md`](specs/001-concurrent-image-pipeline/tasks.md) | How, and the ordered task list (one commit per task) |
| [`docs/reviews/`](docs/reviews/) | Independent reviews and the decision on every finding |
| [`docs/decisions.md`](docs/decisions.md) | Changes made after approval, with reasons |

---

## GenAI Tools

<!-- Luis confirms this statement before submission (spec Q9). -->

Two AI tools were used, with distinct roles, under the rules in [Constitution Article 7](docs/constitution.md):

| Tool | Service tier | Role |
|---|---|---|
| **Claude** (Anthropic, model Claude Opus 5.5) | Claude Pro | **Builder.** Ran the spike on the original program; drafted the constitution, spec, plan and task list; wrote the tests first, then the code; ran the mutation checks; wrote the scripts and this README; triaged review findings. |
| **Gemini** (Google, web) | Gemini Ultra | **Independent reviewer.** Reviewed the draft spec against the rubric (findings and decisions in [`docs/reviews/001-spec-review.md`](docs/reviews/001-spec-review.md)) and the finished repository. Did not edit files. |

The author approved the spec, plan and task list, made every decision in spec section 11, chose and exported the
photos, ran the official measurements and builds on his Mac, and reviewed every change before it was committed.
These are paid subscriptions the author already holds.

**Conversation logs** are in [`docs/ai-dialogs/`](docs/ai-dialogs/): the Claude build conversation and the
Gemini reviews (PDF).

---

## License and credit

Original pipeline © Amrit Singh / CODEHEIM, Apache License 2.0 ([`LICENSE`](LICENSE)); see the
[video tutorial](https://www.youtube.com/watch?v=8Rn8yOQH62k) and [codeheim.io](https://www.codeheim.io/).
Changes in this repository are described above and in the commit history.
