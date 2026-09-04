# Quarry roadmap

What happens next, in order. Updated 2026-09-04, after T7. The build record — what was made,
by which task, in which wave — is git history (`archive/<slug>` tags) and `HANDOFF.md`; this
file only ever says what is ahead.

**The standing rule:** nothing is built without a measured win behind it. T7 enforced it: the
August `toc` cost win did not reproduce cleanly (`results/2026-09-04-toc`, flat-to-reversed at
n=5, tool demonstrably used, correctness unchanged), so the build queue stops until measurement
says where — or whether — the surface pays.

## Next wave: measure

| task | scope | done when |
|---|---|---|
| **M1 ladder breadth** | run ladder b (`ladder-toc.yaml` task 04, the negative control, cells `b0-none`/`b8-toc-dir`); add one or two new task shapes where a toc plausibly pays (multi-package navigation, cold-start orientation on a tree the agent has never seen); if the OSL-1033 host is still available, one same-config rerun there to isolate the host variable | every measured cell is a real MCP cell completing end to end (the third harness rule); `conclusion.md` names where toc separates, or states that it nowhere does |
| **M2 harness observability** | persist the invalidation cause (runner error or exit code) into a discarded attempt's directory before `InvalidateRep` renames it — T7's a0 retries left nothing on disk saying why | an artificially failed attempt's `N.invalid-1/` carries a readable reason file |

M2 is small and independent; it can run as `mill-quick` or fold into M1's pre-matrix work.

## Parked

**T8, the type checker** (`impact`, `assert-no-callers`, `verified`, the DAG tightening) is
parked, not cancelled. It unparks on either of:

- a measured win from M1 that re-establishes the surface's value to an agent, or
- an explicit re-justification by *function* — the §8.1 validator's Delete gate needs
  `assert-no-callers` regardless of agent token costs — recorded here as the reason, in its own
  words, before the task is written.

Its open decision (gopls vs `go/packages` in-process) is decided when it unparks, not before.

## Small and independent, any time

- Move the `docs/research/output-formats/after/` goldens to `internal/cli/testdata/` — they are
  living test fixtures; the research directory stays a frozen record. `mill-quick` candidate.
- Wiki grooming: the completed rewrite tasks' `[done]` entries.

## External (not tasks in this repository)

- Loomyard adopts glyphs: `planparser` imports `glyph`, the validator calls `resolve`, the plan
  format's spelling changes (`internal/shedrecipe#Lookup`). Unblocked since T5b merged; work in
  Loomyard's repository.

## Not tasks yet

A second language (Python, then C#, per `docs/glyph.md`) becomes a task when it is wanted, one
language per task: its alphabet in `glyph/`, an extractor written fresh against the contract,
its `expand` head, its package-doc source. Done when the T3-style round trip over a real
repository in that language reaches 100 %.
