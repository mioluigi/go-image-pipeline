Measured on **Apple M1 Max · 64 GB · macOS 27.0.1**, 10 CPUs, Go 1.27.2, 2026-11-01 18:02 EST.

**Headline** (whole program, 32 photos, median of 5 runs):

- **Worker pool: ×3.54 faster** than sequential; **pipeline: ×1.50**.
- **Pipeline ceiling ×1.54**, set by its slowest stage (load, 65 % of the work) — not by the 10 CPUs.
- **Peak memory:** sequential 174 MB, pipeline 223 MB, worker pool 719 MB (×4.1 sequential).
- **Robustness cost:** on CODEHEIM's four images the original took 126 ms and our pipeline mode 131 ms.

#### 1. The original program, unchanged (CODEHEIM commit `40f5842`)

Built and ran with exit code 0, printing `Success!` 4 times. Same four CODEHEIM images, whole program, median of 5 runs:

| Program | Time (ms) | Peak memory (MB) |
|---|---:|---:|
| original | 126 | 42 |
| ours-sequential | 186 | 42 |
| ours-pipeline | 131 | 42 |
| ours-workers | 87 | 67 |

#### 2. Throughput benchmarks — synthetic 1536 × 1024 photos (median of the runs)

| Mode | Images | Time per run (ms) | Images per second | Speed-up vs sequential | Memory allocated per run (MB) | Allocations per run |
|---|---:|---:|---:|---:|---:|---:|
| sequential | 4 | 174 | 23.0 | ×1.00 | 40 | 666320 |
| sequential | 16 | 664 | 24.1 | ×1.00 | 159 | 2665216 |
| sequential | 64 | 2656 | 24.1 | ×1.00 | 635 | 10660846 |
| pipeline | 4 | 130 | 30.8 | ×1.34 | 40 | 666317 |
| pipeline | 16 | 457 | 35.0 | ×1.45 | 159 | 2665223 |
| pipeline | 64 | 1758 | 36.4 | ×1.51 | 635 | 10660849 |
| workers | 4 | 60 | 66.9 | ×2.91 | 40 | 666322 |
| workers | 16 | 152 | 105.3 | ×4.37 | 159 | 2665230 |
| workers | 64 | 565 | 113.2 | ×4.70 | 635 | 10660853 |

#### 3. Throughput benchmarks — the bundled photos

| Mode | Images | Time per run (ms) | Images per second | Speed-up vs sequential | Memory allocated per run (MB) | Allocations per run |
|---|---:|---:|---:|---:|---:|---:|
| sequential | 4 | 574 | 7.0 | ×1.00 | 213 | 703922 |
| pipeline | 4 | 413 | 9.7 | ×1.39 | 213 | 703932 |
| workers | 4 | 211 | 19.0 | ×2.72 | 213 | 703935 |

#### 4. Where the time goes — sequential mode, 32 images

| Stage | Time (ms) | Share |
|---|---:|---:|
| load | 3038 | 65 % |
| resize | 1442 | 31 % |
| grayscale | 113 | 2 % |
| save | 78 | 2 % |
| **total** | **4670** | |

**Ceiling for the one-goroutine-per-stage pipeline:** total ÷ slowest stage (load) = 4670 ÷ 3038 = **×1.54**, however many CPUs.

#### 5. Whole program — median of 5 runs (start-up, reading and writing files included)

| Mode | Images | Time (ms) | Speed-up vs sequential | Peak memory (MB) |
|---|---:|---:|---:|---:|
| sequential | 4 | 580 | ×1.00 | 166 |
| sequential | 32 | 4559 | ×1.00 | 174 |
| pipeline | 4 | 424 | ×1.37 | 165 |
| pipeline | 32 | 3040 | ×1.50 | 223 |
| workers | 4 | 234 | ×2.48 | 228 |
| workers | 32 | 1288 | ×3.54 | 719 |

#### 6. Test coverage (`go test -race -cover ./...`)

| Package | Coverage |
|---|---:|
| program (main) | 96.1% |
| cmd/measure | 93.0% |
| imageprocessing | 100.0% |
| pipeline | 100.0% |
