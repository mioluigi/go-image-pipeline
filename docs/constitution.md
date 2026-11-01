# Project Constitution — Data Pipelines with Concurrency (Go image pipeline)

> **Purpose of this file.** The non-negotiable rules for the project. Every spec, plan, task, and line of
> code is checked against them. If a later decision conflicts with this file, either the decision changes
> or this file is amended deliberately, with a note in `docs/decisions.md`.
>
> **Source of authority.** Derived from the Week 6 assignment and its **Programming Assignment Rubric**
> (`docs/assignment-and-rubric.md`), carried forward from Weeks 4–5 where the lessons still apply. Each
> rule is tagged with the rubric criterion it protects: **[C]** Coding, **[T]** Testing & metrics,
> **[D]** Design, **[Doc]** Documentation, **[A]** Application.

---

## Article 1 — Correctness before speed

1.1 A concurrent run is only trusted when it produces **the same output images as the sequential run**,
pixel for pixel, checked by a program. **[T]**
1.2 Speed is only reported for a mode that has passed 1.1. A fast wrong answer is not a result. **[T]**
1.3 Every numeric claim in the README is reproducible by a command a grader can run and is copied from
program output — never hand-typed. **[Doc]**
1.4 Concurrent code passes the **race detector** (`go test -race`). **[T]**

## Article 2 — Idiomatic, readable Go (McConnell 2004, ch. 32) **[C]**

2.1 Go modules; `gofmt` and `go vet` clean.
2.2 Meaningful, specific names; not abstract (`channel1`, `data`), not overly long.
2.3 No useless temporary variables or intermediate results.
2.4 Functions do **one thing**; large blocks are split into small, named functions.
2.5 Plain `if/else` and `for`; in complex Boolean expressions, **distribute the NOT**.
2.6 Self-documenting code: every exported identifier has a doc comment; comments explain *why*.
2.7 Errors are returned and handled, never ignored; **no `panic`** for bad files. One bad image is reported
and does not stop the others.
2.8 Every goroutine has a clear owner and a clear end: each channel is closed by its single sender, and no
goroutine is left blocked when an image fails.

## Article 3 — Test-first, measured **[T]**

3.1 Write the failing test before the implementation of every critical component, and confirm each
important test can fail by deliberately breaking the code (mutation check).
3.2 Standard library `testing`; table-driven tests; edge cases; **synthetic images with known answers**
(for example, a solid colour whose grey value is known in advance).
3.3 Benchmarks use `testing.B` with allocation reporting and report **throughput (images per second)**;
CPU and memory profiles are generated and kept.
3.4 Sequential and concurrent modes time **the same work on the same machine**, printing outside timed code,
with medians of repeated runs.

## Article 4 — Clean, reusable design **[D]**

4.1 Image operations and the pipeline live in **reusable Go packages** with no printing and no `os.Exit`.
Command-line handling and messages belong in `main`.
4.2 Sequential and concurrent modes call **the same stage functions**; only the scheduling differs, so the
timing comparison measures concurrency and nothing else.
4.3 Pin every dependency version (`go.mod`, `go.sum`).
4.4 Respect the original work: keep CODEHEIM's commits and Apache-2.0 `LICENSE`, and state what we changed.

## Article 5 — Documentation and Git hygiene **[Doc]**

5.1 One `README.md` is the complete documentation: purpose, roles of programs and data, how to build, run,
test and benchmark, results, and the **recommendation to management**.
5.2 One task per commit with clear messages; specs, plan, tasks, reviews, decisions versioned in the repo.
5.3 The README has a **GenAI Tools** section; **dialogs with AI tools are in the repository** as plain text
or PDF (required deliverable).

## Article 6 — A finished, friendly application **[A]**

6.1 Documented build commands produce a macOS binary, a Windows `.exe`, and a macOS `.app`.
6.2 The program **runs to completion with no arguments** using the bundled images.
6.3 User input is validated; bad input yields a clear message and a non-zero exit code — never a panic.
6.4 The program **never overwrites an input image**.
6.5 Output is clean and readable for a non-programmer; logs go to standard error.

## Article 7 — How we work with AI tools

7.1 **Claude Code** is the only tool that writes files in this repo during a task.
7.2 **Gemini (web)** independently reviews specs, plans, and the finished repository; it does not edit.
7.3 Luis approves every spec, plan, and task list, and reads every diff before it is committed.
7.4 If implementation reveals the spec is wrong, **update the spec first**, then the code.
7.5 Every review finding gets a recorded decision — accept, adjust, or reject — with a reason.
7.6 **Spike before planning:** the riskiest technical assumption is tested with a throwaway experiment
before the plan is written, and the result is recorded in `docs/spikes/`.
