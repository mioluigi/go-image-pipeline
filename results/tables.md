Measured on **Apple M1 Max · 64 GB · macOS 27.0.1**, 10 CPUs, Go 1.27.2, 2026-11-01 17:14 EST.

#### 1. The original program, unchanged (CODEHEIM commit `40f5842`)

Built and ran with exit code 0, printing `Success!` 4 times. Same four CODEHEIM images, whole program, median of 5 runs:

| Program | Time (ms) | Peak memory (MB) |
|---|---:|---:|
| original | 131 | 42 |
| ours-sequential | 195 | 42 |
| ours-pipeline | 135 | 43 |
| ours-workers | 83 | 67 |

#### 2. Throughput benchmarks — synthetic 1536 × 1024 photos (median of the runs)

| Mode | Images | Time per run (ms) | Images per second | Speed-up vs sequential | Memory allocated per run (MB) | Allocations per run |
|---|---:|---:|---:|---:|---:|---:|
| sequential | 4 | 174 | 22.9 | ×1.00 | 40 | 666316 |
| sequential | 16 | 666 | 24.0 | ×1.00 | 159 | 2665216 |
| sequential | 64 | 2655 | 24.1 | ×1.00 | 635 | 10660846 |
| pipeline | 4 | 126 | 31.7 | ×1.38 | 40 | 666317 |
| pipeline | 16 | 449 | 35.6 | ×1.48 | 159 | 2665223 |
| pipeline | 64 | 1761 | 36.3 | ×1.51 | 635 | 10660849 |
| workers | 4 | 58 | 68.8 | ×3.00 | 40 | 666321 |
| workers | 16 | 157 | 102.1 | ×4.25 | 159 | 2665229 |
| workers | 64 | 553 | 115.7 | ×4.80 | 635 | 10660853 |

#### 3. Throughput benchmarks — the bundled photos

| Mode | Images | Time per run (ms) | Images per second | Speed-up vs sequential | Memory allocated per run (MB) | Allocations per run |
|---|---:|---:|---:|---:|---:|---:|
| sequential | 4 | 706 | 5.7 | ×1.00 | 234 | 751193 |
| pipeline | 4 | 554 | 7.2 | ×1.27 | 234 | 751197 |
| workers | 4 | 249 | 16.1 | ×2.83 | 234 | 751198 |

#### 4. Where the time goes — sequential mode, 32 images

| Stage | Time (ms) | Share |
|---|---:|---:|
| load | 3946 | 69 % |
| resize | 1579 | 28 % |
| grayscale | 121 | 2 % |
| save | 89 | 2 % |
| **total** | **5734** | |

**Ceiling for the one-goroutine-per-stage pipeline:** total ÷ slowest stage (load) = 5734 ÷ 3946 = **×1.45**, however many CPUs.

#### 5. Whole program — median of 5 runs (start-up, reading and writing files included)

| Mode | Images | Time (ms) | Speed-up vs sequential | Peak memory (MB) |
|---|---:|---:|---:|---:|
| sequential | 4 | 724 | ×1.00 | 172 |
| sequential | 32 | 5622 | ×1.00 | 159 |
| pipeline | 4 | 570 | ×1.27 | 173 |
| pipeline | 32 | 4083 | ×1.38 | 205 |
| workers | 4 | 265 | ×2.73 | 255 |
| workers | 32 | 1474 | ×3.81 | 974 |

#### 6. Test coverage (`go test -race -cover ./...`)

| Package | Coverage |
|---|---:|
| program (main) | 96.1% |
| cmd/measure | 93.0% |
| imageprocessing | 100.0% |
| pipeline | 100.0% |
