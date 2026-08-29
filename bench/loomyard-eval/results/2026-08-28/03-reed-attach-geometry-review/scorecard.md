# Scorecard — Task 03: reintroduced missed-caller bug (code review)

Dispatched directly by the top-level orchestrating session (three parallel
top-level `Agent` calls). Ran against the reconstructed real Loomyard bug:
worktree at `/tmp/loomyard-review-03`, pinned to commit `8b14f32ba` with
`internal/loomcli/run.go` reverted to its pre-fix content, reproducing a
genuine `go build` failure (`internal/loomcli/run.go:295:46: undefined:
attachArgv`) — confirmed before dispatch and again independently by all
three agents.

## Correctness (A/B vs C)

All three agents found the **same single real issue**, at the same location,
with `severity: blocking`, `confidence: high`:

| | verdict | location | matches C's finding |
|---|---|---|---|
| A | unsafe | internal/loomcli/run.go:295 | yes |
| B | unsafe | internal/loomcli/run.go:295 | yes |
| C (fasit) | needs-changes | internal/loomcli/run.go:295 | -- |

`verdict` differs only in label choice (`unsafe` vs `needs-changes` — the
schema doesn't define a sharp boundary between them for a single blocking
compile-break); the substance is identical across all three. Recall and
precision are both 1/1 = 100% for A and B against C's one real issue: no
false positives, no misses, by either arm.

**C found something neither A nor B did**, though it does not change the
scored finding: there were originally *two* identical free functions
(`reedcli/attach.go` and `loomcli/bootstrap.go`), both deleted by the real
commit, and the reintroduced bug is the *functional* half of the geometry
fix the commit was about — `lyx loom run` is the other registered
terminal-handover path (per `CONSTRAINTS.md`/`docs/overview.md`), not just
an unrelated stale reference. C also ran a counterfactual `go build
-overlay` proving line 295 is the *only* remaining break. This is
consistent with C's role (unbounded effort, cross-verification) rather than
a gap in A or B's correctness on the actual scored question.

## Efficiency

| | tokens | tool_uses | duration | grep calls despite quarry |
|---|---|---|---|---|
| A (quarry) | 64,510 | 21 | 128.9s | 2 |
| B (baseline) | 43,886 | 8 | 52.1s | -- |
| C (fasit) | 66,529 | 36 | 270.9s | -- |

**B won decisively on every efficiency axis, for the identical correct
answer.** B used less than half A's tokens, roughly a third of A's tool
calls, and about 40% of A's wall-clock time.

## Why quarry didn't help here — a task-design finding, not just a tool gap

This task's bug is a renamed identifier that no longer exists anywhere in
the codebase — `go build` catches it immediately and unambiguously, with
zero symbol-resolution work required. B's fastest path (`git show` the
commit, `grep -rn attachArgv`, `go build ./...`) reaches the same certainty
quarry's LSP-backed resolution does, because **the compiler is already the
ground truth for "does this call site still resolve" in a statically-typed
language** — quarry's added precision (interface dispatch, build-tag-gated
call sites) has nothing to contribute when the failure mode is a plain
undefined-symbol compile error.

This suggests task 03 undersells quarry's actual differentiator. A fairer
test of "can grep/build miss a caller that quarry catches" needs a caller
that still *compiles* but is *semantically* wrong after the change — e.g. an
interface satisfied structurally so no compile error fires, or a caller
whose types still line up but whose assumptions (units, ordering, a
renamed-but-type-compatible parameter) no longer hold. Task 04 (not yet
built) should specifically target that shape rather than repeating a
compile-breaking rename, or the benchmark will keep measuring "is `go
build` sufficient," which it usually is, rather than "is quarry's LSP
precision sufficient where `go build` and grep are not."

## Verdict

No measurable quarry advantage on this task, on this single run — B was
faster, cheaper, and equally correct. Not because quarry's answer was wrong
or its tools misbehaved (no CLI bugs surfaced this round, unlike task 01),
but because this specific bug shape (an undefined-symbol compile break) is
exactly the case `go build` alone already resolves with certainty, leaving
quarry's LSP-backed precision no room to differentiate. The real test of
quarry's value for code review is still open — pending a task where the
missed caller compiles but is behaviorally wrong.
