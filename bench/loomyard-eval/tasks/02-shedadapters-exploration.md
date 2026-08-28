# Task 02 — shed pipeline exploration

Type: exploration
Verb under test: `toc` (especially `toc dir` on a large directory)
Status: runnable now

## Setup (do this once, before dispatching A/B/C)

Pinned to `PINNED_SHA` from the top-level README (`975578cda8d6f3a81580bd4e73725e060211b766`),
not the live main checkout:

```
git -C /home/knatte/Code/loomyard/wts/loomyard worktree add /tmp/loomyard-eval-02 975578cda8d6f3a81580bd4e73725e060211b766
```

`<TARGET_DIR>` for this task is `/tmp/loomyard-eval-02`. Remove the worktree
when done (`git -C /home/knatte/Code/loomyard/wts/loomyard worktree remove /tmp/loomyard-eval-02`).

## Scope

`internal/shedbuild`, `internal/shedadapters`, `internal/shedcheck` in
Loomyard. `shedadapters` alone is ~8.4k lines — the largest package in the
cluster — so this task also stress-tests `toc dir` on a directory too big to
comfortably read file-by-file.

## `<TASK TEXT>` (identical for A, B, C)

> Map how a build artifact flows through the "shed" pipeline: from
> `internal/shedbuild`, through `internal/shedadapters`, to
> `internal/shedcheck`. Your answer must identify:
>
> 1. The type(s) that represent a build artifact as it crosses each package
>    boundary — same type reused throughout, or does each package have its
>    own representation with a conversion step?
> 2. The specific function(s) at each handoff point (shedbuild →
>    shedadapters, shedadapters → shedcheck).
> 3. What `shedadapters` actually contributes to the pipeline — its role,
>    not just its existence.
>
> Scope your answer to these three packages.

## Notes for whoever prepares C's fasit / scores this

Not investigated in detail when this task was drafted — genuinely open,
which is fine for an exploration task. If C's answer turns out trivial (e.g.
the three packages barely interact), that is itself a useful finding: pick a
different subsystem for a re-run rather than forcing a scorecard out of a
degenerate case.
