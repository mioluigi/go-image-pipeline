#!/usr/bin/env bash
# Builds dist/ImagePipeline.app, a macOS application bundle for the rubric's
# ".app" requirement (decision Q8). Adapted from the Week 5 project.
#
# The program is a command-line tool, and a double-clicked app has no terminal
# to show its output. So the bundle is a thin wrapper: its launcher opens
# Terminal and runs the program there three times — without goroutines, then
# with the pipeline, then with the worker pool — on the built-in photos, so the
# window shows the comparison. It writes to ~/Desktop/ImagePipeline-output and
# waits for Return so the results stay on screen. The program itself is unchanged.
#
# The program uses no C code, so Go cross-compiles it: this script runs on any
# computer and builds for Apple Silicon (arm64) Macs.
#
# Usage, from any directory:
#   scripts/build-macos-app.sh            then, on a Mac, double-click dist/ImagePipeline.app

set -euo pipefail
cd "$(dirname "$0")/.."

app=dist/ImagePipeline.app
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"

# 1. The program, photos embedded.
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o "$app/Contents/Resources/go-image-pipeline" .

# 2. A script Terminal runs: the three modes in turn, then wait so the window
#    stays readable. Logs (standard error) go to the usual place for macOS app
#    logs, so the window shows only the results.
cat >"$app/Contents/Resources/run-in-terminal.command" <<'SCRIPT'
#!/bin/bash
clear
program="$(dirname "$0")/go-image-pipeline"
output="$HOME/Desktop/ImagePipeline-output"
log_file="$HOME/Library/Logs/ImagePipeline.log"
mkdir -p "$(dirname "$log_file")"
echo "Converting the built-in photos to grayscale three ways, to compare the times."
for mode in sequential pipeline workers; do
  echo
  echo "=== $mode ==="
  if ! "$program" -mode "$mode" -out "$output" 2>>"$log_file"; then
    echo "The program could not finish. Details: $log_file"
    tail -n 3 "$log_file"
  fi
done
echo
read -r -p "Press Return to close this window. " _
SCRIPT

# 3. The app's launcher: hand the script to Terminal.
cat >"$app/Contents/MacOS/ImagePipeline" <<'SCRIPT'
#!/bin/bash
open -a Terminal "$(dirname "$0")/../Resources/run-in-terminal.command"
SCRIPT
chmod +x "$app/Contents/MacOS/ImagePipeline" "$app/Contents/Resources/run-in-terminal.command"

# 4. The metadata macOS needs to treat the folder as an application.
cat >"$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>               <string>ImagePipeline</string>
  <key>CFBundleDisplayName</key>        <string>Image Pipeline</string>
  <key>CFBundleIdentifier</key>         <string>com.github.mioluigi.imagepipeline</string>
  <key>CFBundleExecutable</key>         <string>ImagePipeline</string>
  <key>CFBundlePackageType</key>        <string>APPL</string>
  <key>CFBundleShortVersionString</key> <string>1.0</string>
  <key>CFBundleVersion</key>            <string>1</string>
  <key>LSMinimumSystemVersion</key>     <string>13.0</string>
  <key>LSArchitecturePriority</key>     <array><string>arm64</string></array>
</dict>
</plist>
PLIST

echo "Built $app — double-click it in Finder, or run: open $app"
