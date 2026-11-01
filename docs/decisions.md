# Decision log

Decisions made **after** a document was approved, newest first: date, decision, reason, documents updated.

| Date | Decision | Reason | Updated |
|---|---|---|---|
| 2026-11-01 | The program's end-to-end tests call `run(args, stdout, stderr) int` in-process instead of running the compiled program with `os/exec` (plan §6). `main` only calls `os.Exit(run(…))`. | Same coverage of flags, messages and exit codes; the program's code counts toward coverage; tests run in seconds. A panic would still fail the test. | plan.md §6 (note) |
| 2026-11-01 | Claude's workspace fetches modules with `GOPROXY=direct GOSUMDB=off`; the committed `go.mod` and `go.sum` are unchanged by this. | `proxy.golang.org` and `sum.golang.org` are blocked from Claude's workspace only (spike finding 7). The `go.sum` hash for `nfnt/resize` is CODEHEIM's original, so the Mac verifies it normally. | — |
