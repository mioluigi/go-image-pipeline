#!/usr/bin/env bash
# Runs every measurement for the README (task T12, plan section 7) and saves
# the raw output in results/. Nothing in the README is typed by hand:
# scripts/tables.sh turns these files into tables.
#
#   results/machine.txt          computer, operating system, CPU count, Go version, date
#   results/tests.txt            go test -race -cover ./... (AC-2)
#   results/original-run.txt     CODEHEIM's program unchanged, and timed against ours (AC-1)
#   results/program-<mode>.txt   the program on the bundled photos, each mode, with logs
#   results/stages-32.txt        the program, sequential, 32 images: per-image stage times
#   results/bench.txt            BenchmarkRun: 3 modes × 4, 16, 64 synthetic photos, COUNT runs
#   results/bench-bundled.txt    BenchmarkRunBundled: 3 modes × the bundled photos, COUNT runs
#   results/cpu-profile.txt      CPU profile, sequential mode, 16 synthetic photos
#   results/memory-profile.txt   memory-allocation profile of the same
#   results/whole-program.txt    complete program per mode: median wall time and peak memory
#
# Run on macOS (or Linux) from any directory, with the computer plugged in and
# other programs closed:
#   scripts/measure.sh
# COUNT (benchmark repeats, default 5) and RUNS (whole-program repeats,
# default 5) can be set, e.g. COUNT=3 RUNS=3 scripts/measure.sh

set -euo pipefail
COUNT="${COUNT:-5}"
RUNS="${RUNS:-5}"
cd "$(dirname "$0")/.."

for setting in COUNT RUNS; do
  if ! [[ "${!setting}" =~ ^[1-9][0-9]*$ ]]; then
    echo "measure.sh: $setting must be a positive whole number, got '${!setting}'" >&2
    exit 2
  fi
done

# The program embeds images/, so Go cannot build it without at least one photo there.
if ! ls images/*.jpg images/*.jpeg images/*.png images/*.JPG images/*.JPEG images/*.PNG >/dev/null 2>&1; then
  echo "measure.sh: images/ has no .jpg, .jpeg or .png photos; add them first (task T02)" >&2
  exit 2
fi

step() { echo "measure.sh: $*" >&2; }
trap 'echo "measure.sh: STOPPED because the step above failed; results/ is incomplete, so do not run tables.sh" >&2' ERR

mkdir -p results dist
if [[ "$(uname)" == "Darwin" ]]; then
  computer="$(sysctl -n machdep.cpu.brand_string) · $(sysctl -n hw.memsize | awk '{printf "%.0f GB", $1 / 1073741824}') · macOS $(sw_vers -productVersion)"
  cpus="$(sysctl -n hw.ncpu)"
else
  computer="$(uname -sm)"
  cpus="$(nproc)"
fi
{
  echo "machine: $computer"
  echo "cpus: $cpus"
  echo "go: $(go version | cut -d' ' -f3)"
  echo "measured: $(date '+%Y-%m-%d %H:%M %Z')"
} > results/machine.txt

step "building the program and the measurement tool"
go build -o dist/go-image-pipeline .
go build -o dist/measure ./cmd/measure

step "tests with the race detector and coverage"
go test -race -cover ./... > results/tests.txt

step "CODEHEIM's original program, and ours on the same images"
RUNS="$RUNS" scripts/run-original.sh > results/original-run.txt

for mode in sequential pipeline workers; do
  step "the program on the bundled photos, $mode mode"
  dist/go-image-pipeline -mode "$mode" -out dist/output -verbose > "results/program-$mode.txt" 2>&1
done

# 32 images: the bundled photos copied 8 times, for steadier stage shares and
# to see how memory grows with concurrency. (Not 64: with full-size phone
# photos, 3 modes × 64 images × RUNS runs would take over ten minutes.)
step "preparing 32 images"
rm -rf dist/many && mkdir -p dist/many
for copy in $(seq 1 8); do
  for photo in images/*; do
    cp "$photo" "dist/many/copy$copy-$(basename "$photo")"
  done
done
step "stage times, sequential mode, 32 images"
dist/go-image-pipeline -mode sequential -in dist/many -out dist/many-output -verbose > results/stages-32.txt 2>&1

step "benchmarks on synthetic photos, $COUNT runs (several minutes)"
go test -run='^$' -bench='^BenchmarkRun$' -benchtime=3x -benchmem -count="$COUNT" ./pipeline/ > results/bench.txt
step "benchmarks on the bundled photos, $COUNT runs"
go test -run='^$' -bench='^BenchmarkRunBundled$' -benchtime=3x -benchmem -count="$COUNT" ./pipeline/ > results/bench-bundled.txt

# Sequential mode is profiled so each stage's cost shows up on its own,
# without goroutine scheduling mixed in.
step "CPU and memory profiles, sequential mode"
go test -run='^$' -bench='^BenchmarkRun$/^mode=sequential$/^images=16$' -benchtime=5x \
  -o dist/pipeline.test -cpuprofile dist/cpu.prof -memprofile dist/memory.prof ./pipeline/ > /dev/null
go tool pprof -top -nodecount=25 dist/pipeline.test dist/cpu.prof > results/cpu-profile.txt 2>/dev/null
go tool pprof -top -nodecount=25 -sample_index=alloc_space dist/pipeline.test dist/memory.prof > results/memory-profile.txt 2>/dev/null

step "whole program, $RUNS runs each"
labels=()
for mode in sequential pipeline workers; do
  labels+=("$mode-4=dist/go-image-pipeline -mode $mode -out dist/output")
  labels+=("$mode-32=dist/go-image-pipeline -mode $mode -in dist/many -out dist/many-output")
done
dist/measure -runs "$RUNS" "${labels[@]}" > results/whole-program.txt

step "done; now run: scripts/tables.sh > results/tables.md && scripts/readme.sh"
