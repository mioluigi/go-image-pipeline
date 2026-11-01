# Feature Spec 001 — A robust, measured image pipeline in Go, with and without goroutines

| Field | Value |
|---|---|
| Status | **DRAFT v2 — 2026-11-01** (after Gemini review: `docs/reviews/001-spec-review.md`; awaiting Luis's decisions in section 11) |
| Owner | Luis Varela |
| Course deliverable | Week 6 Assignment: Data Pipelines with Concurrency (100 pts) |
| Governing rules | `docs/constitution.md` · requirements: `docs/assignment-and-rubric.md` |
| Evidence so far | `docs/spikes/001-original-pipeline.md` · review `docs/reviews/001-spec-review.md` |

> **Spec rule:** this document says **what** and **why**. **How** belongs in `plan.md`.

---

## 1. Problem (why)

Management wants shorter data engineering processing times and believes Go concurrency can deliver them.
CODEHEIM's image pipeline is a small, concrete example: four stages (decode → resize → grayscale → save)
joined by channels. Replicating it, making it robust, and **measuring it with and without goroutines** tells
management how much concurrency gains, what it costs in complexity, and — from Spike 001 — that **how the
work is divided matters as much as using goroutines at all**.

## 2. Users and what they need

| User | Needs |
|---|---|
| **Management** | Measured speed-ups, a plain explanation of what limits them, and a recommendation. |
| **Data engineers** | Reusable stage functions and a pipeline that reports bad files instead of crashing. |
| **Grader / any engineer** | Clone, build, run, test, and benchmark from the README alone. |
| **Any user of the program** | One executable that processes the bundled images, or a folder of their own, with clear messages. |

## 3. User stories

- **US-1 Replicate.** As a grader, I want to see the original program built and run unchanged, so the starting point is documented.
- **US-2 Robust.** As a data engineer, I want a missing, unreadable, or unwritable file reported by name while the other images still finish.
- **US-3 Compare.** As a manager, I want processing time and throughput for the same images with and without goroutines, on the same machine.
- **US-4 Understand.** As a manager, I want to know what limits the speed-up and whether a different concurrent design does better.
- **US-5 Faithful images.** As any user, I want outputs that keep the original shape (no stretching), and inputs that are never overwritten.
- **US-6 Usable program.** As any user, I want one executable that runs with no arguments and explains bad input.

## 4. Functional requirements

| ID | Requirement |
|---|---|
| FR-1 | **Original form.** The CODEHEIM program is built and run unchanged, and the result (output, time) is recorded. CODEHEIM's commits and `LICENSE` are kept in the repository history (Constitution 4.4). |
| FR-2 | **Our images.** The four sample images are replaced with images chosen by Luis (Q2), versioned in the repo. |
| FR-3 | **Stages as reusable functions:** decode a file, resize, convert to grayscale, encode and save — each returns an error instead of panicking. |
| FR-4 | **Error checking, input:** missing file, unreadable file, unsupported or corrupt format each produce an error naming the file. Accepted input formats: Q6. |
| FR-5 | **Error checking, output:** the output folder is created if missing; an unwritable folder or failed encode is reported by file name. |
| FR-6 | **A failure travels with its image.** One failed image is reported and does not stop or block the others; the run ends with a summary (succeeded / failed) and a non-zero exit code if any failed. No goroutine is left blocked (Constitution 2.8). |
| FR-6a | **No goroutine left behind.** After every run in every mode — including runs where images fail — the number of running goroutines returns to its value before the run, checked by a test. *(Review 001 #4)* |
| FR-6b | **Results in input order.** Concurrent modes may finish images in any order; results and messages are reported in the order of the input file names, so every mode's output reads the same. *(Review 001 #4)* |
| FR-7 | **Inputs are never overwritten.** Outputs go to a separate output folder; the program refuses to run if the output folder is the input folder. |
| FR-8 | **Modes.** The same stage functions run in each mode chosen in Q3: at minimum **sequential** (no goroutines) and **pipeline** (the original one-goroutine-per-stage design). |
| FR-9 | **Same answer in every mode.** Every mode produces pixel-identical output images to the sequential mode on the same input, checked by a test (Constitution 1.1). |
| FR-10 | **Aspect ratio (optional item, Q4).** Resize keeps the input's width-to-height ratio. |
| FR-11 | **Benchmarks** with `testing.B` for each mode, reporting time per run, **images per second**, and allocations, on synthetic images (fixed seed) and on the bundled images, for several image counts. |
| FR-12 | **Stage timing:** the time spent in each stage is reported, so the bottleneck is shown by measurement, not assumed. |
| FR-13 | **Logging:** timestamped `log/slog` logs to standard error; `-verbose` shows per-image detail. |
| FR-14 | **Program:** runs with **no arguments** on the bundled images in the default mode; the flags and defaults decided in Q7 choose the mode, input folder, output folder, target size, and worker count. Invalid flag values (unknown mode, size or worker count out of range, missing or empty input folder, output folder equal to input folder) produce a clear message before any image is processed. |
| FR-14a | **Exit codes:** **0** every image saved; **1** at least one image failed (the others were saved, FR-6); **2** invalid usage, nothing processed. *(Review 001 #2)* |
| FR-15 | **Whole-program metrics:** wall time and peak memory for each mode, measured from outside the program the same way (the Week 4–5 `cmd/measure` approach). |
| FR-16 | **README** contains: purpose; what changed from CODEHEIM; roles of programs and data; build, run, test and benchmark instructions; results tables; the explanation of the speed-up limit; the **recommendation to management**; and a **GenAI Tools** section. |
| FR-17 | **AI dialogs** (Claude, Gemini) are included in the repository as plain text or PDF. |

## 5. Reference expectations

There is no published timing to match. **The sequential mode is the reference for correctness** (FR-9), and
Spike 001 gives the expected shape of the timings (2-CPU workspace, CODEHEIM's images):

| Measure | Spike result | Expectation on the Mac |
|---|---|---|
| Pipeline speed-up over sequential | ×1.45–×1.47 | Below ≈ ×2.05, the ceiling set by the resize stage (49 % of the work) |
| Worker pool speed-up (2 workers) | ×1.63–×1.73 | Grows with cores, if Q3 includes it |
| Bottleneck stage | Resize (Lanczos3) | Same, unless Q4's sizes change it |

These are hypotheses to confirm or refute on the Mac, not targets. *(Review 001 #5)*

- **The pipeline's ceiling is architectural:** it equals total stage time ÷ slowest stage time and does
  not rise with more CPUs. It **will be recomputed** from the Mac's measured stage times (FR-12), because our
  images and the aspect-preserving resize (Q4) change each stage's share.
- **The worker pool's speed-up depends on the hardware:** it is expected to grow with the number of cores
  until another limit (memory bandwidth, disk) takes over. The README reports the core count with the results.

## 6. Key findings that shape the plan ⚠️ (from Spike 001)

1. **The one-goroutine-per-stage pipeline cannot exceed ≈ ×2 here**, whatever the number of CPUs, because
   throughput is set by the slowest stage (resize).
2. **Every bad file panics today** and kills all images, from inside a goroutine.
3. **The output-path rule overwrote an input file** when the input was outside `images/`.
4. **Only JPEG is decoded**; a PNG fails as "unknown format".
5. **EXIF orientation is ignored** by Go's standard library; phone photos may come out sideways.
6. **The resize library is unmaintained** but sufficient, and can keep the aspect ratio.

## 7. Non-functional requirements

| ID | Requirement | Rubric |
|---|---|---|
| NFR-1 | Code follows Constitution Article 2. | Coding |
| NFR-2 | Test coverage of the reusable packages ≥ 90 %. | Testing |
| NFR-3 | `go test -race ./...` passes. | Testing |
| NFR-4 | CPU and memory profiles of the benchmarks, with `go tool pprof` text summaries kept in `results/`. | Testing |
| NFR-5 | Image operations and pipeline modes are reusable packages, independent of the program's input/output. | Design |
| NFR-6 | From `git clone` to results in ≤ 5 documented commands. | Documentation / Application |
| NFR-7 | README result tables are produced by scripts from files in `results/`, never typed by hand. | Documentation |
| NFR-8 | Every bundled image ≤ 5 MB; the repository stays small enough to clone quickly. | Documentation |

## 8. Acceptance criteria (definition of done)

- [ ] AC-1 The original program's build-and-run output is recorded (FR-1).
- [ ] AC-2 `go test ./...` and `go test -race ./...` pass, including synthetic known-answer tests, every FR-4/FR-5 error case, and the FR-6a goroutine check. `go test -cover` on the reusable packages reports ≥ 90 % for each (NFR-2); its output is saved in `results/`. *(Review 001 #3)*
- [ ] AC-3 Every mode passes the pixel-identical check against sequential (FR-9).
- [ ] AC-4 Benchmarks report time, images per second, and allocations for each mode and several image counts (FR-11).
- [ ] AC-5 Stage timings, CPU and memory profiles are saved and summarized (FR-12, NFR-4).
- [ ] AC-6 Whole-program wall time and peak memory are reported for each mode (FR-15).
- [ ] AC-7 The executable runs with no arguments; each error case has a test and a clear message; inputs are never overwritten.
- [ ] AC-8 Build commands produce a macOS binary, a Windows `.exe`, and a macOS `.app` (Q8). Performance figures come from the Mac.
- [ ] AC-9 README contains every FR-16 item, with tables generated from `results/`.
- [ ] AC-10 AI dialog files are in the repository (FR-17).
- [ ] AC-11 Repository URL (ending in `.git`) submitted in Canvas.

## 9. Rubric traceability

| Rubric criterion (20 pts each) | Satisfied by |
|---|---|
| Coding rules, organization, aesthetics | NFR-1, Constitution Art. 2 |
| Testing and software metrics | FR-9, FR-11–FR-13, FR-15, NFR-2–NFR-4, AC-2–AC-6 |
| Design and development | FR-3, FR-6, FR-6a, FR-6b, FR-8, FR-10, NFR-5 |
| Documentation | FR-1, FR-16, FR-17, NFR-6, NFR-7, AC-9, AC-10 |
| Application | FR-7, FR-14, FR-14a, AC-7, AC-8 |

## 9a. Assignment-step traceability *(Review 001 #1)*

| Assignment step | Satisfied by |
|---|---|
| 1 Clone the repository | FR-1, Q1 |
| 2 Build and run in original form | FR-1, AC-1 |
| 3 Error checking for input and output | FR-4, FR-5, FR-6, FR-6a, FR-7, AC-2, AC-7 |
| 4 Replace the four images | FR-2, Q2 |
| 5 Unit tests | FR-9, NFR-2, NFR-3, AC-2, AC-3 |
| 6 Benchmarks; run with and without goroutines | FR-8, FR-11, FR-12, AC-4 |
| 7 Additional modifications | Worker pool (Q3), PNG input (Q6), never overwrite inputs (FR-7), ordered results (FR-6b), command-line flags (FR-14), stage timing (FR-12), executables (AC-8) |
| 8 Build, test, run with and without goroutines; compare | FR-8, FR-12, FR-15, AC-3–AC-6, AC-9 |
| 9 Complete README | FR-16, AC-9 |
| 10 (Optional) preserve aspect ratio | FR-10, Q4 |

## 10. Out of scope

- Applying EXIF orientation (unless Q2 changes this); other image formats beyond Q6; GPU processing.
- A network service or a graphical interface.
- Replacing the resize algorithm or library (it is the measured workload; changing it would change the comparison).

## 10a. Cut order if time runs short *(Review 001 #7)*

Cut in this order, recording each cut in `docs/decisions.md` and the README's limitations:

1. **Worker-pool mode** (Q3) — the README still explains the ceiling from measured stage times.
2. **The outside-the-program measurement tool** (FR-15) — replaced by `/usr/bin/time -l` on the Mac.

Not cut: error handling, tests, benchmarks, the sequential-versus-pipeline comparison, executables, README.

## 11. Decisions — **for Luis** (recommendation in bold)

| # | Question | Options | Recommendation |
|---|---|---|---|
| Q1 | Repository | (a) new repo, keep CODEHEIM's 3 commits + `LICENSE`, CODEHEIM as `upstream` remote; (b) GitHub "fork"; (c) fresh repo, code copied | **(a) `github.com/mioluigi/go-image-pipeline`** — the assignment says *clone*; history shows exactly what we changed; Apache-2.0 respected. |
| Q2 | Our images | Count, source, sizes | **Four of Luis's own photos** (no third-party rights), at least **two non-square** (landscape and portrait) to show the aspect-ratio fix, each ≤ 5 MB, **EXIF orientation checked** (re-saved upright if needed); EXIF handling itself stays out of scope and is noted in the README. |
| Q3 | Concurrent modes | (a) pipeline only; (b) pipeline + worker pool | **(b)** — Spike 001 shows the pipeline is capped near ×2; a worker pool (each worker runs all stages on one image, workers = CPUs) answers management's real question. Pool is the first task cut if time runs short. |
| Q4 | Aspect ratio (optional) | (a) skip; (b) fit inside 500 × 500 keeping the ratio; (c) fixed width 500, height follows | **(b)** — the longer side becomes 500; square images give the same 500 × 500 as before, so results stay comparable with CODEHEIM's. |
| Q5 | Error policy | (a) stop at the first error; (b) report and continue | **(b)** with a summary and exit code 1 if any image failed. |
| Q6 | Input formats | (a) JPEG only; (b) JPEG and PNG | **(b)** — one import; output stays JPEG. |
| Q7 | Command line | Flags | **`-mode sequential\|pipeline\|workers`** (default `pipeline`, the original design), **`-in`** (default bundled `images/`), **`-out`** (default `output/`), **`-size`** (default 500), **`-workers`** (default CPUs), **`-verbose`**. |
| Q8 | Executables | — | As Weeks 4–5: macOS binary, Windows `.exe`, macOS `.app` (Terminal wrapper). |
| Q9 | AI dialogs and tiers | — | As Week 5: Claude conversation and Gemini reviews in `docs/ai-dialogs/` (PDF); README **GenAI Tools** section states Claude Pro and Gemini Ultra; **Luis confirms the statement**. |
| Q10 | Go version in `go.mod` | keep `go 1.20`; raise | **Raise to `go 1.22`** — from 1.22 each loop iteration has its own variable, which removes a classic goroutine bug; the Mac's newer Go builds it. |
