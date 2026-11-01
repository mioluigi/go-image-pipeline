# Review 001 — Spec 001 reviewed by Gemini

| Field | Value |
|---|---|
| Artifact reviewed | `specs/001-concurrent-image-pipeline/spec.md` (draft v1) |
| Reviewer | Gemini (web), prompted as a strict grader and senior Go engineer |
| Checked against | `docs/constitution.md`, `docs/assignment-and-rubric.md`, `docs/spikes/001-original-pipeline.md` |
| Date | 2026-11-01 |
| Verdict given | *Approve after changes* (coverage command in AC-2; error-path safety for Constitution 2.8) |
| Outcome | 3 accepted, 3 accepted with changes, 1 rejected → spec draft v2 |

| # | Finding (summarized) | Severity | Decision | Reason / change made |
|---|---|---|---|---|
| 1 | Worker pool and aspect ratio are not tied to assignment step 7 ("additional modifications"). | Low | ✅ Accept, widened | Added section 9a, **assignment-step traceability**: every one of the ten steps maps to FRs and ACs, not only step 7. |
| 2 | FR-14 and Q7 should state the same defaults and flag error behaviour. | Low | 🔶 Accept with change | Gemini found no actual conflict. The real gap was **exit codes**, spread across FR-6 and FR-14. Added **FR-14a**: 0 all images saved, 1 at least one image failed, 2 invalid usage (nothing processed). FR-14 now names the Q7 flags and defaults. |
| 3 | AC-2 does not say how a grader checks the 90 % coverage. | Medium | ✅ Accept | **AC-2** now names `go test -cover` on the reusable packages, with its output saved in `results/`. |
| 4 | An error on one image could leave goroutines blocked or leaked; mandate `context` cancellation or clean channel teardown in the plan. | High | 🔶 Accept with change | **Requirement accepted, mechanism not mandated.** Added **FR-6a**, which is testable: after any run, including runs with failed images, the number of running goroutines returns to its starting value, checked by a test. *How* (context or channel teardown) is a plan decision. Note: under Q5 (report and continue), an image error never stops a stage from reading, so cancellation is not needed for the error path; the plan explains this. Also added **FR-6b**, a risk Gemini did not raise: concurrent modes finish images **out of order**, so results are reported in input order. |
| 5 | Section 5 should say the pipeline ceiling is architectural while the worker pool scales with cores. | Low | ✅ Accept, corrected | Added to section 5. One correction: Gemini wrote that the ≈ ×2.05 ceiling holds "on 2 CPUs". It does not depend on CPUs at all; it comes from stage shares. Those shares **will change** with our images and the aspect-preserving resize (Q4), so section 5 now says the ceiling is recomputed from the Mac's measured stage times (FR-12). |
| 6 | Agrees with all ten recommendations Q1–Q10. | — | ✅ Noted | No change. Luis still decides (Constitution 7.3). |
| 7 | Scope: make the macOS `.app` a stretch goal. | Low | ❌ Reject (cut list added instead) | The rubric's **Application** criterion asks for `.exe` or `.app` instructions, and the tested Week 5 build script is reused, so the `.app` costs minutes, not hours. (Gemini also cited "FR-15", which is whole-program metrics, not the `.app`.) The concern about time is fair, so section 10a now gives an **explicit cut order** if the evening runs short: worker pool first, then the outside-the-program memory tool. |

## Lesson recorded

Finding 4 was the most valuable: right risk, wrong level. A spec should state the property a grader can
test (no goroutine left behind), not the mechanism. Turning it into a measurable requirement also exposed a
second concurrency risk, out-of-order results, that neither the draft nor the review had named.
