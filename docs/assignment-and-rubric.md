# Assignment Requirements and Rubric (summary)

> Summary of the Week 6 assignment, *Data Pipelines with Concurrency* (100 points, due 2026-11-01), in our
> own words, so every tool and reviewer can check the spec against it. The Canvas page is the authoritative source.

## Scenario

A technology startup's managers want to **reduce processing times for data engineering tasks**. They believe
**Go concurrency** can shorten data pipeline throughput times, and want a replicated pipeline example to
explore the possibilities.

## Source to replicate

Amrit Singh (CODEHEIM) — image processing pipeline with goroutines:
code `github.com/code-heim/go_21_goroutines_pipeline` (Apache-2.0), video tutorial on YouTube (episode 21).
Four stages connected by channels: **load (decode) → resize (500×500) → grayscale → save (JPEG)**.

## Required steps

1. Clone the GitHub repository.
2. Build and run the program in its **original form**.
3. Add **error checking** for image file input and output.
4. Replace the four input images with **images of our choosing**.
5. Add **unit tests**.
6. Add **benchmark methods** that capture pipeline throughput times; the program must run **with and without
   goroutines**.
7. Make additional modifications as we see fit.
8. Build, test, and run **with and without goroutines**; **compare processing times**.
9. Write a complete **README.md**.
10. *(Optional)* **Preserve the aspect ratio**: detect the input size and keep the output's shape, instead
    of the fixed 500×500 in `image_processing.Resize`.

## Deliverables

- The **cloneable GitHub URL** (ending in `.git`) in the Canvas comments.
- `README.md` as complete documentation, including how to run and test the program.
- A **GenAI Tools** section in the README describing how AI tools were used, plus **conversation logs**
  (plain text files or links).

## Rubric — five criteria, 20 points each

Ratings per criterion: Outstanding 20 (all criteria) · Meets 15 (most) · Needs Improvement 10 (many not met) ·
Partially Acceptable 5 (most not met) · Unacceptable 0.

| Criterion | What earns full marks |
|---|---|
| **[C] Coding rules, organization, aesthetics** | Go modules, idiomatic Go; meaningful names, not too long or short; no useless temporaries; clean blocks; one task per block; simple control flow; distribute the NOT in Boolean expressions; self-documenting code with comments on the logic (McConnell 2004, 777–817). |
| **[T] Testing and software metrics** | Unit tests of critical components with synthetic data; logs and profiles; memory and processing measured for components and the whole program; Monte Carlo benchmark *if the requirements call for it* (they do not this week). |
| **[D] Design and development** | Clean, efficient, understandable design meeting every requirement; reusable general-purpose modules. |
| **[Doc] Documentation** | Effective Git/GitHub use; README noting the roles of programs and data, how to test and use the application. |
| **[A] Application** | Instructions to build an executable (`.exe` for Windows or `.app` for macOS); runs to completion; checks user input and explains incorrect input; clean UX/UI. |
