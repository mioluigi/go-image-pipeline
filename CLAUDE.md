# CLAUDE.md — project context

Data Pipelines with Concurrency (Week 6): replicate CODEHEIM's Go image pipeline (decode → resize → grayscale →
save), make it robust, run it with and without goroutines (sequential, pipeline, worker pool), prove every
mode produces identical images, measure throughput, and recommend to management.

## Read these first (in order)

1. `docs/constitution.md` — non-negotiable rules (rubric-derived)
2. `specs/001-concurrent-image-pipeline/spec.md` — what and why (APPROVED v2)
3. `specs/001-concurrent-image-pipeline/plan.md` — how (APPROVED)
4. `specs/001-concurrent-image-pipeline/tasks.md` — ordered checklist; work the next unchecked task
5. `docs/spikes/001-original-pipeline.md` — what was measured before planning

## Working rules

- One task per commit; message starts with the task ID (`T03: …`).
- Test first: write the failing test, run it, then implement; check important tests by breaking the code.
- If implementation shows the spec or plan is wrong, update the document first, then the code.
- Record changes after approval in `docs/decisions.md`; review findings in `docs/reviews/`.
- Never hand-type results into the README; generate them from `results/`.
- Performance numbers come only from Luis's Mac, and only for modes that pass the identical-output test.
- CODEHEIM's three commits and `LICENSE` stay in the history (Apache-2.0).

## Key commands

```bash
go test ./...                                         # all tests
go test -race ./...                                   # race detector
go test -cover ./imageprocessing/ ./pipeline/         # coverage (≥ 90 % each)
go vet ./... && gofmt -l .                            # lint (gofmt -l prints nothing)
go test -bench . -benchmem ./pipeline/                # throughput benchmarks
go run .                                              # bundled images, pipeline mode
go run . -mode sequential                             # without goroutines
go run . -mode workers -in ~/Pictures/x -out /tmp/out # your own folder
```

## Layout

`imageprocessing/` decode, fit, grayscale, encode (one image) · `pipeline/` the three modes and stage timing ·
`main.go` the program · `images/` bundled photos (embedded) · `cmd/measure/` whole-program time and memory ·
`scripts/` measurement, tables, builds · `results/` measured outputs · `docs/`, `specs/` the process record.
