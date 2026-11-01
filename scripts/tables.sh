#!/usr/bin/env bash
# Builds the README's results tables from the files scripts/measure.sh saved,
# so no number is ever typed by hand (Constitution 1.3). Times are shown in
# milliseconds (ms), memory in megabytes (MB = 2^20 bytes).
#
# Usage, from any directory:
#   scripts/tables.sh > results/tables.md
# RESULTS names another folder of result files (used to try this script out).
#
# Uses only POSIX awk features, so it runs with the awk shipped with macOS.

set -euo pipefail
cd "$(dirname "$0")/.."
RESULTS="${RESULTS:-results}"

files=(machine.txt tests.txt original-run.txt stages-32.txt bench.txt bench-bundled.txt whole-program.txt)
for file in "${files[@]}"; do
  if [[ ! -s "$RESULTS/$file" ]]; then
    echo "tables.sh: missing or empty $RESULTS/$file (run scripts/measure.sh first)" >&2
    exit 1
  fi
done
cd "$RESULTS"

awk '
# median of values[1..count], sorted in place (insertion sort; counts are small)
function median(values, count,    i, j, held) {
  for (i = 2; i <= count; i++) {
    held = values[i]
    for (j = i - 1; j >= 1 && values[j] > held; j--) values[j + 1] = values[j]
    values[j + 1] = held
  }
  if (count % 2) return values[(count + 1) / 2]
  return (values[count / 2] + values[count / 2 + 1]) / 2
}

# field returns the value of key=value in the current line
function field(key,    i, pair) {
  for (i = 1; i <= NF; i++) {
    split($i, pair, "=")
    if (pair[1] == key) return pair[2]
  }
  return ""
}

# milliseconds converts a Go duration such as 152.3ms, 873.2µs, 1.2s or 500ns
function milliseconds(text) {
  if (text ~ /ms$/) return substr(text, 1, length(text) - 2) + 0
  if (text ~ /ns$/) return substr(text, 1, length(text) - 2) / 1000000
  if (text ~ /s$/ && text ~ /[0-9]s$/) return substr(text, 1, length(text) - 1) * 1000
  sub(/[^0-9.]+$/, "", text)   # microseconds: the µ sign is two bytes in some awks
  return text / 1000
}

# benchmark lines: Name-N  iterations  value unit  value unit ...
function readBenchmark(prefix,    name, i, key) {
  name = $1
  sub(/-[0-9]+$/, "", name)
  sub(/^Benchmark[A-Za-z]+\//, "", name)
  key = prefix SUBSEP name
  if (!(key in seen)) { seen[key] = 1; order[prefix, ++orderCount[prefix]] = name }
  samples[key]++
  for (i = 3; i < NF; i += 2) {
    if ($(i + 1) == "ns/op")    nsPerRun[key, samples[key]] = $i
    if ($(i + 1) == "images/s") imagesPerSecond[key, samples[key]] = $i
    if ($(i + 1) == "B/op")     bytesPerRun[key, samples[key]] = $i
    if ($(i + 1) == "allocs/op") allocsPerRun[key, samples[key]] = $i
  }
}

function medianOf(table, key,    i, values) {
  for (i = 1; i <= samples[key]; i++) values[i] = table[key, i]
  return median(values, samples[key])
}

function printBenchmarkTable(prefix,    i, name, key, parts, count, baseKey, speed) {
  print "| Mode | Images | Time per run (ms) | Images per second | Speed-up vs sequential | Memory allocated per run (MB) | Allocations per run |"
  print "|---|---:|---:|---:|---:|---:|---:|"
  for (i = 1; i <= orderCount[prefix]; i++) {
    name = order[prefix, i]
    key = prefix SUBSEP name
    split(name, parts, "/"); sub(/^mode=/, "", parts[1]); count = parts[2]; sub(/^images=/, "", count)
    baseKey = prefix SUBSEP "mode=sequential/images=" count
    speed = medianOf(nsPerRun, baseKey) / medianOf(nsPerRun, key)
    printf "| %s | %d | %.0f | %.1f | ×%.2f | %.0f | %.0f |\n", parts[1], count,
      medianOf(nsPerRun, key) / 1000000, medianOf(imagesPerSecond, key), speed,
      medianOf(bytesPerRun, key) / 1048576, medianOf(allocsPerRun, key)
  }
}

FILENAME == "machine.txt" { split($0, kv, ": "); machine[kv[1]] = substr($0, length(kv[1]) + 3) }

FILENAME == "tests.txt" && /coverage:/ {
  package = $2; sub(/^github.com\/mioluigi\/go-image-pipeline\/?/, "", package)
  if (package == "") package = "program (main)"
  coverageCount++; coveragePackage[coverageCount] = package; coveragePercent[coverageCount] = $(NF - 2)
}

FILENAME == "original-run.txt" && /^Success!$/ { originalSuccesses++ }
FILENAME == "original-run.txt" && /^exit code:/ { originalExit = $3 }
FILENAME == "original-run.txt" && /^wholeprogram/ {
  originalRows[++originalCount] = field("program"); originalWall[field("program")] = field("median_wall_ms")
  originalMemory[field("program")] = field("median_maxrss_mb")
}

FILENAME == "stages-32.txt" && /msg="image done"/ {
  stageImages++
  stage["load"] += milliseconds(field("load"))
  stage["resize"] += milliseconds(field("resize"))
  stage["grayscale"] += milliseconds(field("grayscale"))
  stage["save"] += milliseconds(field("save"))
}

FILENAME == "bench.txt" && /^BenchmarkRun\// { readBenchmark("synthetic") }
FILENAME == "bench-bundled.txt" && /^BenchmarkRunBundled\// { readBenchmark("bundled") }

FILENAME == "whole-program.txt" && /^wholeprogram/ {
  wholeRows[++wholeCount] = field("program")
  wholeRuns = field("runs"); wholeWall[field("program")] = field("median_wall_ms"); wholeMemory[field("program")] = field("median_maxrss_mb")
}

END {
  printf "Measured on **%s**, %s CPUs, Go %s, %s.\n\n", machine["machine"], machine["cpus"], substr(machine["go"], 3), machine["measured"]

  # Headline: the numbers the recommendation rests on, computed here so the
  # README never types them by hand.
  total = stage["load"] + stage["resize"] + stage["grayscale"] + stage["save"]
  slowest = "load"
  for (name in stage) if (stage[name] > stage[slowest]) slowest = name
  largest = 0
  for (i = 1; i <= wholeCount; i++) { split(wholeRows[i], parts, "-"); if (parts[2] + 0 > largest) largest = parts[2] + 0 }
  print "**Headline** (whole program, " largest " photos, median of " wholeRuns " runs):\n"
  printf "- **Worker pool: ×%.2f faster** than sequential; **pipeline: ×%.2f**.\n", \
    wholeWall["sequential-" largest] / wholeWall["workers-" largest], wholeWall["sequential-" largest] / wholeWall["pipeline-" largest]
  printf "- **Pipeline ceiling ×%.2f**, set by its slowest stage (%s, %.0f %% of the work) — not by the %s CPUs.\n", \
    total / stage[slowest], slowest, 100 * stage[slowest] / total, machine["cpus"]
  printf "- **Peak memory:** sequential %.0f MB, pipeline %.0f MB, worker pool %.0f MB (×%.1f sequential).\n", \
    wholeMemory["sequential-" largest], wholeMemory["pipeline-" largest], wholeMemory["workers-" largest], \
    wholeMemory["workers-" largest] / wholeMemory["sequential-" largest]
  printf "- **Robustness cost:** on CODEHEIM'"'"'s four images the original took %.0f ms and our pipeline mode %.0f ms.\n\n", \
    originalWall["original"], originalWall["ours-pipeline"]

  print "#### 1. The original program, unchanged (CODEHEIM commit `40f5842`)\n"
  printf "Built and ran with exit code %s, printing `Success!` %d times. Same four CODEHEIM images, whole program, median of %d runs:\n\n", originalExit, originalSuccesses, wholeRuns
  print "| Program | Time (ms) | Peak memory (MB) |"
  print "|---|---:|---:|"
  for (i = 1; i <= originalCount; i++) printf "| %s | %.0f | %.0f |\n", originalRows[i], originalWall[originalRows[i]], originalMemory[originalRows[i]]

  print "\n#### 2. Throughput benchmarks — synthetic 1536 × 1024 photos (median of the runs)\n"
  printBenchmarkTable("synthetic")
  print "\n#### 3. Throughput benchmarks — the bundled photos\n"
  printBenchmarkTable("bundled")

  print "\n#### 4. Where the time goes — sequential mode, " stageImages " images\n"
  total = stage["load"] + stage["resize"] + stage["grayscale"] + stage["save"]
  slowest = "load"
  for (name in stage) if (stage[name] > stage[slowest]) slowest = name
  print "| Stage | Time (ms) | Share |"
  print "|---|---:|---:|"
  split("load resize grayscale save", stageOrder, " ")
  for (i = 1; i <= 4; i++) printf "| %s | %.0f | %.0f %% |\n", stageOrder[i], stage[stageOrder[i]], 100 * stage[stageOrder[i]] / total
  printf "| **total** | **%.0f** | |\n\n", total
  printf "**Ceiling for the one-goroutine-per-stage pipeline:** total ÷ slowest stage (%s) = %.0f ÷ %.0f = **×%.2f**, however many CPUs.\n", slowest, total, stage[slowest], total / stage[slowest]

  printf "\n#### 5. Whole program — median of %d runs (start-up, reading and writing files included)\n\n", wholeRuns
  print "| Mode | Images | Time (ms) | Speed-up vs sequential | Peak memory (MB) |"
  print "|---|---:|---:|---:|---:|"
  for (i = 1; i <= wholeCount; i++) {
    label = wholeRows[i]; split(label, parts, "-")
    printf "| %s | %s | %.0f | ×%.2f | %.0f |\n", parts[1], parts[2], wholeWall[label],
      wholeWall["sequential-" parts[2]] / wholeWall[label], wholeMemory[label]
  }

  print "\n#### 6. Test coverage (`go test -race -cover ./...`)\n"
  print "| Package | Coverage |"
  print "|---|---:|"
  for (i = 1; i <= coverageCount; i++) printf "| %s | %s |\n", coveragePackage[i], coveragePercent[i]
}
' machine.txt tests.txt original-run.txt stages-32.txt bench.txt bench-bundled.txt whole-program.txt
