# Decision log

Decisions made **after** a document was approved, newest first: date, decision, reason, documents updated.

| Date | Decision | Reason | Updated |
|---|---|---|---|
| 2026-11-01 | Claude's workspace fetches modules with `GOPROXY=direct GOSUMDB=off`; the committed `go.mod` and `go.sum` are unchanged by this. | `proxy.golang.org` and `sum.golang.org` are blocked from Claude's workspace only (spike finding 7). The `go.sum` hash for `nfnt/resize` is CODEHEIM's original, so the Mac verifies it normally. | — |
