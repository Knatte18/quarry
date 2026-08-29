# Scorecard — Task 04: pre-implementation impact analysis (interface-method conflation)

Dispatched directly by the top-level orchestrating session. This task was
built specifically to test the one scenario tasks 01-03 didn't: a real,
naturally-occurring interface-method-name collision in Loomyard
(`shedadapters.Shuttle.Run` vs `shedadapters.BurlerRunner.Run`, sitting as
sibling fields on the same struct) — the exact shape where a text search
cannot distinguish "real caller of the interface method in question" from
"coincidentally same-named method on something else" without resolving
types, and where a compiler alone (unlike task 03) can't help either, since
this is a *pre-implementation* question, not a diff already made.

## Correctness (A/B vs C, the sharp check)

All three found the identical 3 real callers and correctly excluded the
`burler.go:373` decoy — the one mistake this task exists to catch. Neither
A nor B included the decoy in `callers_to_update`. Recall/precision on the
scored core question: **100% for both A and B.**

C went further (unbounded effort, its own compiler experiment as ground
truth): identified that `bouncer.go:466`'s enclosing function has no `ctx`
in scope and would need threading from its caller, found a second minor
lookalike (`webster.go:75`, a lowercase func-typed field, not a method at
all), and flagged the cascading `var _ Shuttle = ...` assertion. None of
this changes the A/B score — the task's own scope note says not to
penalize for missing things outside the literal ask, and neither A nor B
made the one mistake being tested for.

## Efficiency

| | tokens | tool_uses | duration | grep calls despite quarry |
|---|---|---|---|---|
| A (quarry) | 51,164 | 14 | 80.0s | 1 (justified — see below) |
| B (baseline) | 51,587 | 12 | 77.6s | -- |
| C (fasit) | 80,662 | 32 | 273.8s | -- |

A and B are statistically indistinguishable — a wash in both directions
(A: fewer tokens, more tool calls, slightly slower; B: more tokens, fewer
tool calls, slightly faster). A's one grep call was inspected directly
(not just self-reported) and was a reasonable choice: quarry's `impact`
resolves *real* callers of an interface method, but has no verb for
"enumerate every textually-similar-but-different call site" — the
exclusion list is inherently a text-search question, and quarry doesn't
claim to answer it.

## Verdict

**This was the task engineered to give quarry its single best shot** — a
real interface-method collision, the one thing grep is structurally unable
to disambiguate without type resolution — and both arms got it perfectly
right, at effectively identical cost. quarry did not measurably help even
here.

The likely reason, inspecting B's actual reasoning: this particular
collision, while real, only required *one hop* of type-tracing (struct
field -> its declared interface type -> that interface's method set) to
resolve, and Go's declarations are all textually local and visible — a
capable agent with Read/Grep can follow "which struct, which field, which
interface" by hand about as reliably as an LSP does, at this depth. quarry's
structural advantage over grep should widen with deeper indirection (a
value passed through several function boundaries before the call, or many
more than two same-named candidates to sift), not the two-interface,
single-package case tested here. That remains untested, and after three
tasks each designed to favor quarry along a different axis (raw tool
speed, missed-caller correctness, interface conflation) with no measurable
win on any of them, further narrowing the search for a scenario where it
helps has a low prior at this point.
