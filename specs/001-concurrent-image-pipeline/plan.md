# Plan 001 — How we will build Spec 001

| Field | Value |
|---|---|
| Status | **APPROVED — 2026-11-01** |
| Implements | `spec.md` (APPROVED v2) |
| Governing rules | `docs/constitution.md` |
| Evidence | `docs/spikes/001-original-pipeline.md`, `docs/reviews/001-spec-review.md` |

> The spec says *what* and *why*; this plan says *how*, with the reason for each choice.

---

## 1. Technical context

| Item | Choice | Why |
|---|---|---|
| Go | `go 1.22` in `go.mod`; built with the Mac's Go (≥ 1.27) | Q10: per-iteration loop variables. |
| Module path | `github.com/mioluigi/go-image-pipeline` (was `goroutines_pipeline`) | Q1: importable from its real home. |
| Resize | `github.com/nfnt/resize`, same pinned version, Lanczos3 | Keeps the measured workload identical to CODEHEIM's (spec §10). |
| Image formats | Decode JPEG and PNG (`image/jpeg`, `image/png`); encode JPEG at the library's default quality | Q6; default quality = CODEHEIM's `jpeg.Encode(…, nil)`. |
| Logging | `log/slog` text handler to standard error; `-verbose` lowers the level to Debug | FR-13. |
| Whole-program metrics | `cmd/measure`, copied from Week 5 (same author, tested) | FR-15. |

## 2. Repository layout

```
go-image-pipeline/
├── go.mod, go.sum, LICENSE (CODEHEIM, Apache-2.0, kept)
├── main.go                  the program: flags, validation, messages, exit codes (stays at the root, as in the original)
├── main_test.go             end-to-end tests of the program
├── images/                  Luis's four photos (Q2), embedded into the program (images.go)
├── imageprocessing/         REUSABLE: Decode, Fit, Grayscale, Encode — renamed from image_processing/ (Go package names have no underscores)
├── pipeline/                REUSABLE: Job, Result, the three modes, stage timing
├── cmd/measure/             whole-program wall time and peak memory (from Week 5)
├── scripts/                 run-original.sh, measure.sh, tables.sh, readme.sh, build.sh, build-macos-app.sh
├── results/                 measured outputs (text), committed
├── docs/, specs/            the process record
└── output/                  default output folder (git-ignored)
```

## 3. Package `imageprocessing` — one image, one step at a time

| Function | Behaviour | Errors |
|---|---|---|
| `Decode(r io.Reader) (image.Image, error)` | JPEG or PNG. | Wrapped `image.Decode` error ("unsupported or corrupt image"). |
| `Fit(img image.Image, maxWidth, maxHeight uint) image.Image` | Largest size inside the box that **keeps the ratio** (Q4); square 1536 → 500 × 500 as before. Smaller images are scaled up to the box too, so every output has the same longer side. | — |
| `Grayscale(img image.Image) *image.Gray` | **CODEHEIM's algorithm unchanged** (`color.GrayModel` per pixel). | — |
| `Encode(w io.Writer, img image.Image) error` | JPEG, default quality. | Encoder error. |

Readers and writers instead of paths: the package never touches the file system, so tests run on bytes in
memory, and file errors are handled in one place (`pipeline`). The `Fit` arithmetic is ours (not
`resize.Thumbnail`, which never enlarges) so it can be tested exactly, then calls `resize.Resize`.

## 4. Package `pipeline` — scheduling only

```go
type Job struct {               // one image travelling through the stages
    Index      int              // position in the input list (FR-6b)
    InputName  string
    OutputPath string
    Image      image.Image
    Size       image.Point      // original size, for the report
    Err        error            // set by the first stage that fails; later stages pass the job through
    StageTimes [4]time.Duration // load, resize, grayscale, save (FR-12)
}

func Run(mode Mode, inputs fs.FS, names []string, cfg Config) []Result // results in input order
```

**Four stage functions** (`load`, `resize`, `grayscale`, `save`) take and return a `Job`. Each one does
nothing if `job.Err` is already set, and records its own duration. All three modes call these same four
functions (Constitution 4.2):

| Mode | Scheduling | Goroutines |
|---|---|---|
| `sequential` | `for` loop: all four stages on one image, then the next. | none |
| `pipeline` | CODEHEIM's design: one goroutine per stage, unbuffered channels, each stage closes its output when its input closes. | 4 (+ collector in the caller) |
| `workers` | `-workers` goroutines (default `runtime.NumCPU()`), each running all four stages on one image at a time, fed by one job channel; a `sync.WaitGroup` closes the results channel. | N |

**Why no goroutine is left blocked (FR-6a, Review 001 #4).** Errors travel *inside* the `Job` (`Err`), never
on a separate channel, and no stage stops reading early. So every stage reads until its input closes and
closes its output; the caller reads results until that closes. With report-and-continue (Q5) there is no
early exit, hence no need for `context` cancellation on the error path; adding it would be code with no
case to test. If a future version stops at the first error, `context.Context` is the mechanism to add. The
test checks the property directly: `runtime.NumGoroutine()` before and after `Run`, in every mode, with
failing images in the input.

**Order (FR-6b).** The collector places each result at `results[job.Index]`, so the returned slice is in
input order in every mode.

**Input from `fs.FS`.** The program passes either the embedded `images/` (no arguments) or
`os.DirFS(-in)`. Tests pass `fstest.MapFS` with synthetic images, including corrupt files. Output files are
written with `os.Create` into `cfg.OutputDir`, created with `os.MkdirAll` at the start of `Run`.

## 5. The program (`main.go`)

1. Parse flags (Q7): `-mode` (`pipeline`), `-in` (embedded images), `-out` (`output`), `-size` (500,
   allowed 1–10,000), `-workers` (CPUs, allowed 1–1,024), `-verbose`.
2. Validate **before processing anything** → message + exit 2: unknown mode; size or workers out of range;
   `-in` missing or with no `.jpg/.jpeg/.png` files; `-out` resolving (absolute path, symlinks followed) to the
   same folder as `-in`; two inputs that map to the same output name (`a.jpg`, `a.png` → `a.jpg`).
3. Output name = input base name + `.jpg` (output is always JPEG, so `photo.png` → `photo.jpg`).
4. Run; print one line per image in input order, then a summary with the total time and per-stage totals:
   ```
   ✓ city.jpg     4032 × 3024 → 500 × 375   output/city.jpg
   ✗ broken.jpg   unsupported or corrupt image
   3 saved, 1 failed · mode pipeline · 1.21 s (load 0.40 · resize 0.58 · grayscale 0.17 · save 0.06)
   Output folder: /Users/luis/…/output
   ```
5. Exit 0 / 1 / 2 (FR-14a). The absolute output folder is printed because a double-clicked `.app` has no
   obvious working directory.

## 6. Testing strategy

| Level | What | Data |
|---|---|---|
| `imageprocessing` | Grayscale of solid colours gives the known `GrayModel` value; `Fit` sizes table (square, landscape, portrait, smaller than the box, 1-pixel-wide); decode JPEG, PNG, corrupt, empty, unsupported (GIF); encode error from a failing writer | Synthetic images built in the test |
| `pipeline` | Every mode returns pixel-identical images to `sequential` (FR-9); one failing image among several → others saved, failure named, order kept; missing input name; unwritable output folder; goroutine count restored (FR-6a); `-race` | `fstest.MapFS` with synthetic images (fixed seed) |
| program | No arguments → 4 saved, exit 0; each exit-2 case; folder with one corrupt file → exit 1, others saved; input files' checksums unchanged after a run (FR-7) | Temporary folders; the compiled program run with `os/exec` (Week 5 pattern) |

Mutation checks (Constitution 3.1) on: the `Err` pass-through, the index placement, and the out == in check.

> **Note (decision log, 2026-11-01):** the program-level tests call `run(args, stdout, stderr)` in-process rather
> through `os/exec`; and the image listing became `pipeline.ImageNames`, shared by the program and the benchmarks.

## 7. Performance measurement design

- **Benchmarks** (`pipeline/benchmark_test.go`): `BenchmarkRun/mode=…/images=4,16,64` on synthetic 1536 × 1024
  images (fixed seed, written once to a temporary folder), plus `BenchmarkRunBundled/mode=…` on Luis's photos.
  Each reports `images/s` with `b.ReportMetric` and allocations with `b.ReportAllocs`. `-benchtime=5x -count=5`
  gives five samples per row; `tables.sh` takes medians.
- **Stage timing** (FR-12): summed from `Job.StageTimes`, printed by the program, and saved by `measure.sh`.
- **Profiles** (NFR-4): `go test -bench … -cpuprofile -memprofile`, `go tool pprof -top` text in `results/`.
- **Whole program** (FR-15): `cmd/measure -runs 5` over the three modes of the built binary.
- **Original form** (FR-1): `scripts/run-original.sh` checks out CODEHEIM's commit `40f5842` in a temporary
  `git worktree`, builds, runs and times it, and saves `results/original-run.txt`.
- **Machine** recorded in `results/machine.txt` (model, cores, Go version). Official numbers: Luis's Mac only.

## 8. Executables (AC-8)

`scripts/build.sh` → `dist/image-pipeline` (this machine), `dist/image-pipeline.exe` (Windows amd64), and on an
Apple Silicon Mac `dist/ImagePipeline.app` (Week 5's Terminal-wrapper script, adapted). No CGo is involved, so
every target cross-compiles; images are embedded, so each executable runs alone.

## 9. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Phone photos sideways (EXIF) or HEIC | Luis exports from Photos as JPEG (export applies rotation); Claude checks orientation tags after commit. |
| Large photos make the repo heavy | Export at ≤ 5 MB each (NFR-8); 12 MP JPEGs are typically 2–5 MB. |
| Benchmarks noisy on a laptop | Medians of 5; plugged in; other apps closed; machine recorded. |
| Deadline tonight | Cut order in spec §10a; tasks ordered so a submittable repo exists after Phase 3. |

## 10. Constitution check

| Article | How the plan meets it |
|---|---|
| 1 Correctness first | Pixel-identical test across modes; `-race`; tables generated from `results/`. |
| 2 Idiomatic Go | Small stage functions; errors returned; no panics; one owner per channel. |
| 3 Test-first | 🔴/🟢 task pairs; synthetic known answers; mutation checks; `testing.B` with images/s. |
| 4 Reusable design | `imageprocessing` and `pipeline` print nothing; same stage functions in every mode; CODEHEIM history kept. |
| 5 Documentation | One README; one task per commit; reviews and decisions versioned; AI dialogs included. |
| 6 Application | No-argument run on embedded images; exit codes 0/1/2; inputs never overwritten; three executables. |
| 7 AI workflow | Claude writes, Gemini reviews, Luis approves and reads every diff. |
