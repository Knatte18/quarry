# Quarry roadmap

What happens next, in order. Updated 2026-09-04, after this task's breadth matrix (M1). The
build record — what was made, by which task, in which wave — is git history (`archive/<slug>`
tags) and `HANDOFF.md`; this file only ever says what is ahead.

**The standing rule:** nothing is built without a measured win behind it. M1 enforced it further:
a breadth matrix across three shapes — the negative control, multi-package exploration, and
whole-repo cold-start orientation — found no shape where directory-level `toc` separates from its
control on any cost metric at n=5
(`bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md`), so the build queue stops
until measurement says where — or whether — the surface pays.

## Parked

**T8, the type checker** (`impact`, `assert-no-callers`, `verified`, the DAG tightening) is
parked, not cancelled. It unparks on either of:

- a measured win from M1 that re-establishes the surface's value to an agent — the breadth
  matrix does not supply one:
  `bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md` finds toc separating from
  its control on no cost metric, in either direction, across all three shapes it measured, at
  n=5, or
- an explicit re-justification by *function* — the §8.1 validator's Delete gate needs
  `assert-no-callers` regardless of agent token costs — recorded here as the reason, in its own
  words, before the task is written.

Whether that leaves T8 parked, or whether the second condition is enough on its own, is the
operator's call — not decided here. Its open decision (gopls vs `go/packages` in-process) is
decided when it unparks, not before.

## Small and independent, any time

- Move the `docs/research/output-formats/after/` goldens to `internal/cli/testdata/` — they are
  living test fixtures; the research directory stays a frozen record. `mill-quick` candidate.
- Wiki grooming: the completed rewrite tasks' `[done]` entries.
- A same-config rerun of ladder a (task 01) on the OSL-1033 host, if and when it becomes
  available again — operator-coordinated; the one remaining way to isolate the host variable
  behind T7's task-01 discrepancy. Deferred by M1, not resolved.

## External (not tasks in this repository)

- Loomyard adopts glyphs: `planparser` imports `glyph`, the validator calls `resolve`, the plan
  format's spelling changes (`internal/shedrecipe#Lookup`). Unblocked since T5b merged; work in
  Loomyard's repository.

## Not tasks yet

A second language (Python, then C#, per `docs/glyph.md`) becomes a task when it is wanted, one
language per task: its alphabet in `glyph/`, an extractor written fresh against the contract,
its `expand` head, its package-doc source. Done when the T3-style round trip over a real
repository in that language reaches 100 %.
