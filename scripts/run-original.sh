#!/usr/bin/env bash
# Builds and runs CODEHEIM's program in its original form (spec FR-1, AC-1),
# then times it against our program on the same four CODEHEIM images.
#
# The original is checked out from this repository's own history (commit
# 40f5842, CODEHEIM's last commit) into a temporary git worktree, so nothing
# is downloaded and our files are untouched. Its relative paths (images/…)
# need it to run from its own folder.
#
# Usage, from any directory (after `go build -o dist/measure ./cmd/measure`
# and `go build -o dist/go-image-pipeline .`, which scripts/measure.sh does):
#   scripts/run-original.sh > results/original-run.txt
# RUNS (repeats for the timing, default 5) can be set.

set -euo pipefail
RUNS="${RUNS:-5}"
cd "$(dirname "$0")/.."
root="$(pwd)"
original_commit=40f5842
worktree="$root/dist/original"

for tool in dist/measure dist/go-image-pipeline; do
  if [[ ! -x "$tool" ]]; then
    echo "run-original.sh: build $tool first (scripts/measure.sh does)" >&2
    exit 2
  fi
done

git worktree remove --force "$worktree" 2>/dev/null || true
git worktree add --quiet --detach "$worktree" "$original_commit"
trap 'git -C "$root" worktree remove --force "$worktree"' EXIT

cd "$worktree"
echo "## CODEHEIM's program, commit $original_commit, unchanged"
echo
echo '$ go build -o pipeline . && ./pipeline'
go build -o pipeline .
mkdir -p images/output
./pipeline
echo "exit code: $?"
echo
echo "## The same four CODEHEIM images: original versus ours (median of $RUNS runs)"
echo
"$root/dist/measure" -runs "$RUNS" \
  "original=./pipeline" \
  "ours-sequential=$root/dist/go-image-pipeline -mode sequential -in images -out $root/dist/ours-on-original" \
  "ours-pipeline=$root/dist/go-image-pipeline -mode pipeline -in images -out $root/dist/ours-on-original" \
  "ours-workers=$root/dist/go-image-pipeline -mode workers -in images -out $root/dist/ours-on-original"
