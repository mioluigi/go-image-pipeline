# Tasks 001 — Ordered checklist to implement Plan 001

| Field | Value |
|---|---|
| Status | **APPROVED — 2026-11-01** |
| Implements | `plan.md` → `spec.md` (APPROVED v2) |

**How to work this list:** one task = one commit (message starts with the task ID). 🔴 tasks write tests that
must fail first; 🟢 tasks make them pass. 🤖 Claude writes and pushes · 👤 Luis (anything on the Mac:
photos, official timings, executables, AI dialogs, Gemini, submission). After every 🤖 task: `git pull`, read
the diff, run "Verify".

**Deadline tonight:** after Phase 3 the repository is submittable with placeholder results; Phases 4–6 make
it complete. If time runs short, cut in the order of spec §10a (✂︎ marks).

---

## Phase 0 — Approval and setup
- [x] **T00 👤 Approve** spec v2 (answer Q1–Q10), this plan, and this list; **create the empty GitHub repo**
  `mioluigi/go-image-pipeline` (no README, licence, or .gitignore). Claude then pushes the history.
- [x] **T01 🤖 Scaffolding.** Module path and `go 1.22`; rename `image_processing/` → `imageprocessing/`;
  `CLAUDE.md`; `.gitignore` (`output/`, `dist/`, `*.out`, `*.prof`, `*.test`); `docs/decisions.md`;
  remove CODEHEIM's committed `images/output/`.
  Verify: `go build ./... && go vet ./...`.
- [ ] **T02 👤 Photos (Q2).** Four of your own: Photos → File → Export → JPEG, ≤ 5 MB each, at least one
  landscape and one portrait. Drop them in `images/` replacing CODEHEIM's four; commit. *(Can run in
  parallel with T03–T08; Claude checks orientation tags and sizes after.)*

## Phase 1 — Reusable packages
- [x] **T03 🔴 `imageprocessing` tests** (known grey values, `Fit` sizes, decode/encode errors).
- [x] **T04 🟢 `imageprocessing` implementation.** Verify: `go test -cover ./imageprocessing/` ≥ 90 %.
- [ ] **T05 🔴 `pipeline` tests** (identical output across modes, failures, order, goroutines, `-race`).
- [ ] **T06 🟢 `pipeline`: sequential and pipeline modes.** Verify: `go test -race -cover ./pipeline/` ≥ 90 %.
- [ ] **T07 🔴🟢 `pipeline`: workers mode.** ✂︎ first cut.
- [ ] **T08 🤖 Benchmarks** (`BenchmarkRun`, `BenchmarkRunBundled`; images/s; allocations).

## Phase 2 — The program
- [ ] **T09 🔴 End-to-end tests** for the program (no args; exit 2 cases; exit 1 with a corrupt file; inputs unchanged).
- [ ] **T10 🟢 `main.go`** with embedded images, flags, messages, exit codes.

## Phase 3 — Measurement
- [ ] **T11 🤖 `scripts/run-original.sh`, `cmd/measure` (from Week 5) ✂︎ second cut, `scripts/measure.sh`, `scripts/tables.sh`.**
- [ ] **T12 👤 Run `scripts/measure.sh` on the Mac** (plugged in, apps closed); commit `results/`.

## Phase 4 — Executables
- [ ] **T13 🤖 `scripts/build.sh`, `scripts/build-macos-app.sh`.**
- [ ] **T14 👤 Build all three; double-click the `.app`.**

## Phase 5 — Documentation and submission
- [ ] **T15 🤖 README** (what changed from CODEHEIM; tables by script; the ceiling explained; recommendation;
  **GenAI Tools** section for Luis to confirm).
- [ ] **T16 👤 AI dialogs.** Export this Claude conversation and the Gemini reviews to `docs/ai-dialogs/` (PDF).
- [ ] **T17 👤 Gemini review of the repository** against the rubric; Claude triages into `docs/reviews/002-…`.
- [ ] **T18 👤🤖 Final acceptance** (fresh clone, every AC) **and submit** `https://github.com/mioluigi/go-image-pipeline.git` in Canvas.

## Traceability

| Acceptance criterion | Tasks |
|---|---|
| AC-1 original form recorded | T11, T12 |
| AC-2 tests, race, coverage | T03–T07, T09 |
| AC-3 identical output across modes | T05–T07 |
| AC-4 benchmarks | T08, T12 |
| AC-5 stage timings, profiles | T06, T10, T12 |
| AC-6 whole program | T11, T12 |
| AC-7 no-argument run; input errors; inputs unchanged | T09, T10 |
| AC-8 executables | T13, T14 |
| AC-9 README | T15 |
| AC-10 AI dialogs | T16 |
| AC-11 submitted | T18 |
