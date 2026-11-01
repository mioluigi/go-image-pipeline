# Review 002 — the finished repository reviewed by Gemini

| Field | Value |
|---|---|
| Artifact reviewed | `github.com/mioluigi/go-image-pipeline` at commit `ecf6784` (all tasks through T15, T12 redone on the final photos) |
| Reviewer | Gemini (web), prompted as a strict grader and senior Go engineer (prompt in the dialog PDF) |
| Dialog | `docs/ai-dialogs/gemini-repository-review.pdf` |
| Date | 2026-11-01 |
| Score given | **98 / 100** (Coding 18, Testing 20, Design 20, Documentation 20, Application 20) |
| Verdict given | *Submit as is* (optionally add `context` cancellation) |
| Outcome | 1 finding rejected with evidence; 2 statements in the review found inaccurate; no code changes |

## Findings

| # | Finding (summarized) | Severity | Decision | Reason / evidence |
|---|---|---|---|---|
| 1 | If an image fails mid-pipeline, stage goroutines could block on unread channel sends and leak; wrap the run in `context.WithCancel`. (Also the reason for the 18/20 in Coding.) | Medium | ❌ Reject (already handled, verified) | The same concern as Review 001 #4, where the **requirement** was accepted as FR-6a and the mechanism left to the plan. A failed image does not stop anything: the error travels inside the job, later stages pass it through, and no stage or collector stops reading early, so every channel is drained and closed by its single sender (plan §4). `TestNoGoroutineIsLeftBehind` counts goroutines before and after every mode with failing and missing images in the input; it passed **50 repetitions under `-race`** today, and the T07 mutation check showed it fails when a collector does stop early. Cancellation would only be needed if the program stopped at the first error, which decision Q5 rejected; adding it now would be code with no case to test. |

## Statements in the review that do not match the repository

| Statement | What the repository says | Effect |
|---|---|---|
| "The README accurately reflects Spike 001: the pipeline hits a ≈ ×2 ceiling due to the Lanczos3 resize bottleneck." | On the bundled 12 MP photos the slowest stage is **loading** (65 %), and the measured ceiling is ×1.54 (README table 4 and headline; spike addendum). ×2.05 / resize was the spike's prediction on CODEHEIM's smaller images, and the README says so. | None on the code. It shows the review's evidence check relied on the spike, not on `results/tables.md`. |
| "Committed AI conversation dialogs" / step 10 "included in `docs/ai-dialogs/`". | At review time only the Gemini spec review was committed; the Claude build conversation is added in T16, before submission. | Tracked by AC-10 and the final acceptance check. |

## Lesson recorded

A high score from a reviewer is only as good as what it read. Both inaccurate statements describe an earlier
state of the project, so the evidence check that matters is the one that runs: `scripts/tables.sh`
regenerates every number, and the final acceptance check (T18) compares the README with it on a fresh clone.
