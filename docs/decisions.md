# Decision log

Decisions made **after** a document was approved, newest first: date, decision, reason, documents updated.

| Date | Decision | Reason | Updated |
|---|---|---|---|
| 2026-11-01 | The `.app` is cross-compiled for Apple Silicon by `scripts/build-macos-app.sh` on any computer (plan §8 said "on an Apple Silicon Mac"). Its Terminal window runs the three modes in turn on the built-in photos and writes to `~/Desktop/ImagePipeline-output`. | No C code, so no reason to require a Mac to build it. Showing all three modes makes the double-clicked app demonstrate the comparison, not just one run. | plan.md §8 (note) |
| 2026-11-01 | `scripts/run-original.sh` also times our program against the original on CODEHEIM's four images (whole program, same machine). Program runs use 32 images (bundled photos × 8), not 64. | A direct original-versus-ours comparison on identical input. 32 keeps T12 to minutes with full-size phone photos; benchmarks still cover 64. | — |
| 2026-11-01 | The program's end-to-end tests call `run(args, stdout, stderr) int` in-process instead of running the compiled program with `os/exec` (plan §6). `main` only calls `os.Exit(run(…))`. | Same coverage of flags, messages and exit codes; the program's code counts toward coverage; tests run in seconds. A panic would still fail the test. | plan.md §6 (note) |
| 2026-11-01 | Claude's workspace fetches modules with `GOPROXY=direct GOSUMDB=off`; the committed `go.mod` and `go.sum` are unchanged by this. | `proxy.golang.org` and `sum.golang.org` are blocked from Claude's workspace only (spike finding 7). The `go.sum` hash for `nfnt/resize` is CODEHEIM's original, so the Mac verifies it normally. | — |
