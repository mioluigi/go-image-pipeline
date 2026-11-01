# Spike 001 — The original pipeline: does it run, where does it break, how much can concurrency gain?

| Field | Value |
|---|---|
| Date | 2026-11-01 |
| Machine | Claude's Linux workspace, **2 CPUs**, Go 1.24.7 (indicative only; official numbers come from Luis's Mac) |
| Code | CODEHEIM `main` at `40f5842`, unchanged; timing harness in `001-evidence/timing_main.go.txt` |
| Raw output | `001-evidence/timing-output.txt` |

## Questions

1. Does the original program build and run as published? (Assignment step 2)
2. How does it fail on bad input and output? (step 3)
3. How much can the **one-goroutine-per-stage** design speed things up, and what limits it? (steps 6, 8)

## Results

### 1. Original form: builds and runs ✅

`go build` then `./pipeline`: four lines of `Success!`, exit 0, **0.35 s** wall time. Inputs are four
1536 × 1536 colour JPEGs; outputs are 500 × 500 grayscale JPEGs in `images/output/`. Because the sample
images are square, the fixed 500 × 500 resize does not distort *them*.

### 2. Failure modes (all confirmed by running them)

| Case | What happened | Consequence |
|---|---|---|
| Input file missing | `panic: open images/missing.jpeg: no such file or directory` | Whole program dies; the other three images are lost. |
| Corrupt file (not a JPEG) | `panic: image: unknown format` | Same. |
| **PNG input** | `panic: image: unknown format` | Only the JPEG decoder is registered (`image/jpeg` import); PNG is never recognised. |
| `images/output/` missing | `panic: open images/output/image1.jpeg: no such file or directory` | The program never creates its output folder. |
| **Input outside `images/`** | **The input file was overwritten** with the 500 × 500 grayscale result (checksum changed). | `strings.Replace(p, "images/", "images/output/", 1)` finds nothing to replace, so the output path equals the input path. **Data loss.** |
| Non-square input (1536 × 768) | Saved as 500 × 500 | Aspect ratio distorted (assignment's optional item). |
| JPEG with EXIF orientation tag 6 (phone photo held upright) | Tag ignored; output not rotated; tag dropped | Go's `image/jpeg` does not apply EXIF orientation. Phone photos may come out sideways. |

Also: every panic happens **inside a stage goroutine**, so there is no chance to report which image failed
in a friendly way, and nothing cleans up.

### 3. Where the time goes, and the ceiling on speed-up

Stage totals for the four sample images, run one after another:

| Stage | Time | Share |
|---|---|---|
| Decode JPEG | 170 ms | 34 % |
| **Resize (Lanczos3)** | **241 ms** | **49 %** |
| Grayscale | 64 ms | 13 % |
| Encode JPEG | 19 ms | 4 % |
| **Total** | **494 ms** | |

Median of 7 runs (file reading and writing excluded, so only processing is timed):

| Images | Sequential | Original pipeline (1 goroutine per stage) | Worker pool (2 workers, whole image per worker) |
|---|---|---|---|
| 4 | 422 ms | 287 ms (**×1.47**) | 244 ms (×1.73) |
| 16 | 1.638 s | 1.124 s (**×1.46**) | 1.008 s (×1.63) |
| 64 | 6.765 s | 4.668 s (**×1.45**) | 3.933 s (×1.72) |

## Findings that shape the spec ⚠️

1. **The original pipeline's speed-up is capped by its slowest stage, not by the number of CPUs.** With one
   goroutine per stage, at best the stages overlap fully, so throughput is limited by resize:
   494 ms ÷ 241 ms ≈ **×2.05 ceiling**, however many cores the Mac has. Adding more images does not help
   (×1.47 → ×1.45). This is the most important point for management: *concurrency helps, but how it is
   arranged decides how much.*
2. **A worker pool (fan-out) is not capped this way:** each worker runs all four stages on its own image, so
   it should scale with cores. With only 2 CPUs here it already beats the pipeline; on the Mac the gap
   should widen. This is a design question for the spec (Q3).
3. **Error handling must change design, not just add `if err != nil`:** a failed image has to travel down
   the pipeline as a reported failure, so the other images still finish and no goroutine is left waiting.
4. **The output path rule is dangerous** and must be replaced with an explicit output folder that is never
   the input folder.
5. **Input formats:** registering `image/png` is one import; JPEG and PNG can both be accepted.
6. **EXIF orientation** is not handled by Go's standard library; our chosen images should be checked, and the
   README should say so (Q2).
7. **Workspace note:** `proxy.golang.org` and `sum.golang.org` are blocked in Claude's workspace;
   `GOPROXY=direct` with the upstream `go.sum` works. The committed `go.mod`/`go.sum` stay normal.
8. **The resize library `github.com/nfnt/resize` is no longer maintained** (its README says so; pinned
   version from 2018) but works and is what the original uses. It already supports keeping the aspect ratio:
   `Resize` with 0 for one dimension, or `Thumbnail(maxW, maxH, …)` to fit inside a box. Keeping it keeps the
   replication faithful.
