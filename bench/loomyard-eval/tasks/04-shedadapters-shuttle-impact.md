# Task 04 — pre-implementation impact analysis (interface-method conflation)

Type: impact analysis (pre-implementation — "what would I need to touch",
not a review of a diff already made)
Verb under test: `impact`/`refs` (also usable: `assert-no-callers`)
Status: runnable now

## Why this task exists

Tasks 01/02 (exploration) and 03 (post-hoc review of a compile-breaking
rename) both showed no measurable quarry advantage — see their scorecards.
Task 03's own postmortem flagged why: its bug was a plain undefined-symbol
compile break, which `go build` alone catches with certainty regardless of
tool, leaving quarry's LSP precision nothing to differentiate on.

This task targets the one thing quarry's design specifically claims an edge
on and neither prior task tested: **interface-method conflation** — a
call site that reads identically to an unrelated call site (same method
name, different interface, different concrete type), where a text search
cannot distinguish "real caller of the interface method in question" from
"coincidentally same-named method on something else" without actually
resolving types. This is also the realistic shape of the actual use case
motivating this whole benchmark: an implementer finding every real impact
of a signature change *before* making it, not reviewing it after the fact.

Found by searching Loomyard's real codebase for a genuine, unconstructed
occurrence of this pattern (not fabricated) — see below.

## Setup (do this once, before dispatching A/B/C)

Pinned to `PINNED_SHA` from the top-level README
(`975578cda8d6f3a81580bd4e73725e060211b766`), not the live main checkout —
all three candidate lines below were verified to match exactly at this
pin.

```
git -C /home/knatte/Code/loomyard/wts/loomyard worktree add /tmp/loomyard-eval-04 975578cda8d6f3a81580bd4e73725e060211b766
```

`<TARGET_DIR>` for this task is `/tmp/loomyard-eval-04`. Remove the worktree
when done (`git -C /home/knatte/Code/loomyard/wts/loomyard worktree remove /tmp/loomyard-eval-04`).

No file is reverted or modified for this task — unlike task 03, this is a
forward-looking "what would need to change" analysis on the code exactly as
it stands at the pin, not a reconstructed historical bug.

## Scope

`internal/shedadapters` in Loomyard.

## The real conflation

`internal/shedadapters/singlellm.go:38-41` declares:

```go
type Shuttle interface {
	Run(shuttleengine.Spec) (shuttleengine.Result, error)
	Attach(shuttleengine.Spec) (shuttleengine.Result, bool, error)
}
```

satisfied by `*shuttleengine.Runner` (asserted at `singlellm.go:43`).

`internal/shedadapters/burler.go:25-27` *independently* declares an
unrelated interface with a same-named method:

```go
type BurlerRunner interface {
	Run(p burlerengine.Profile, opts burlerengine.RunOpts) (burlerengine.Result, error)
}
```

satisfied by `*burlerengine.Engine`. Different package, different
signature, different purpose (`BurlerRunner.Run` drives one review round;
`Shuttle.Run` spawns an LLM shuttle turn). The two interfaces sit as
sibling fields on the very same struct, `BurlerProducer` (`burler.go:62-67`:
`runner BurlerRunner` and `attach Shuttle`) — about as adversarial a
same-package collision as occurs naturally in this codebase, not a
contrived one.

## `<TASK TEXT>` (identical for A, B, C)

> You are about to change the `Shuttle` interface's `Run` method in
> `internal/shedadapters/singlellm.go`, adding a `context.Context` first
> parameter:
>
> ```go
> // before
> Run(shuttleengine.Spec) (shuttleengine.Result, error)
> // after
> Run(ctx context.Context, spec shuttleengine.Spec) (shuttleengine.Result, error)
> ```
>
> Before making this change, identify every real call site **within
> `internal/shedadapters`** that invokes this specific interface method and
> would need a `context.Context` argument threaded in to keep compiling.
>
> Some other call sites in this same package call a *different* method that
> also happens to be named `Run` — those must not be listed, since changing
> `Shuttle.Run`'s signature does not affect them. For every call site you
> list, say how you confirmed it actually resolves to `Shuttle.Run` and not
> a different method of the same name.

## Output schema (impact-analysis tasks)

```json
{
  "callers_to_update": [
    {"file": "internal/shedadapters/....go", "line": N, "evidence": "how you confirmed this resolves to Shuttle.Run specifically"}
  ],
  "excluded_lookalikes": [
    {"file": "internal/shedadapters/....go", "line": N, "reason": "why this same-named call site is NOT Shuttle.Run"}
  ],
  "confidence": "high|medium|low",
  "open_questions": ["anything left uncertain, if any"]
}
```

## Notes for whoever scores this (ground truth — do not reveal to A/B/C)

**Real callers of `Shuttle.Run` within `internal/shedadapters` (3):**
- `internal/shedadapters/singlellm.go:143` — `result, err := p.shuttle.Run(spec)`
- `internal/shedadapters/bouncer.go:466` — `result, err = b.cfg.Shuttle.Run(spec)`
- `internal/shedadapters/bouncer.go:580` — `result, runErr = b.cfg.Shuttle.Run(spec)`

**The decoy that must be excluded:**
- `internal/shedadapters/burler.go:373` — `result, runErr := p.runner.Run(profile, attemptOpts)`, which is
  `BurlerRunner.Run`, not `Shuttle.Run` (different receiver field `runner`
  vs `shuttle`/`Shuttle`, different arity, different types).

**Scoring is sharper than tasks 01-03's recall/precision alone.** Besides
recall/precision on `callers_to_update` against the 3 real callers above,
check specifically whether `burler.go:373` was wrongly included in
`callers_to_update` (a genuine conflation failure — the exact failure mode
this task exists to surface) versus correctly placed in
`excluded_lookalikes` or simply never mentioned. Wrongly including it is a
materially worse mistake than a missed real caller: it means the agent
would have shipped a broken edit to an unrelated interface. Note this
distinction explicitly in the scorecard rather than folding it into a
single precision number.

Outside `internal/shedadapters`, the identical `Shuttle` interface shape is
independently redeclared in three more packages (`treadleengine`,
`mergeresolve`, `burlerengine`), each satisfied by the same
`*shuttleengine.Runner` — real, but out of this task's scope (`internal/
shedadapters` only). Do not penalize A/B/C for not finding these; the task
text scopes to one package specifically so the comparison stays as
apples-to-apples as tasks 01/02's own directory scoping.
