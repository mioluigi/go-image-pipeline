#!/usr/bin/env bash
# Builds every executable into dist/ (spec AC-8, plan section 8):
#   dist/go-image-pipeline       command-line program for this computer
#   dist/go-image-pipeline.exe   Windows program (64-bit Intel/AMD)
#   dist/ImagePipeline.app       macOS application for Apple Silicon Macs
#
# Go cross-compiles by setting GOOS and GOARCH, and the program uses no C code,
# so all three build on any computer. The photos are embedded, so each
# executable runs on its own with no other files.
#
# Usage, from any directory:
#   scripts/build.sh

set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist

# The program embeds images/, so Go cannot build it without at least one photo there.
photo_count=0
shopt -s nullglob nocaseglob # count .jpg, .JPG, …; a pattern matching nothing adds nothing
for photo in images/*.jpg images/*.jpeg images/*.png; do photo_count=$((photo_count + 1)); done
shopt -u nullglob nocaseglob
if [[ "$photo_count" -eq 0 ]]; then
  echo "build.sh: images/ has no .jpg, .jpeg or .png photos; add them first (task T02)" >&2
  exit 2
fi

go build -o dist/go-image-pipeline .
echo "Built dist/go-image-pipeline"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/go-image-pipeline.exe .
echo "Built dist/go-image-pipeline.exe"

scripts/build-macos-app.sh
