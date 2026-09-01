# Task 05 — pre-implementation impact analysis (same-package, comment-adjacent conflation)

Type: impact analysis (pre-implementation — "what would I need to touch",
not a review of a diff already made)
Verb under test: `impact`/`assert-no-callers`/`refs` (with `within` scoping)
Status: runnable now

## Why this task exists

Task 04 targeted interface-method conflation and came back a wash: every
config, including the no-tool control, scored perfect 1.00 recall/precision
on `internal/shedadapters`'s `Shuttle.Run` vs `BurlerRunner.Run` — the two
same-named methods have different signatures and a careful agent reading
types (not just names) resolves it correctly without gopls. Every
quarry-tool config was strictly *slower* than the control for zero
accuracy gain.

This task looks for a sharper trap in the same category, found by
searching Loomyard's real codebase (not fabricated) for a same-named call
that a fast, pattern-matching grep pass is more likely to misfire on than
task 04's decoy was. It is not a guaranteed win — the two `Resolve`
methods below also differ in argument shape, so a careful type-reading
agent can still get this right by hand. What's different is the decoy's
*placement*: it lives inside the target method's own defining package,
immediately after doc-comment prose that narrates the target method's own
resolution flow — not in some unrelated file three subsystems away. If
that placement doesn't move the needle either, that is itself the useful
result: it would mean grep-based disambiguation is more robust to
same-package proximity traps than expected, not just that this benchmark
keeps picking easy cases by accident.

## Setup (do this once, before dispatching arms)

Pinned to `PINNED_SHA` from the top-level README
(`975578cda8d6f3a81580bd4e73725e060211b766`) — every line cited below was
verified to match exactly at this pin via `git show <sha>:<path>`.

```
git -C /home/hanf/Code/loomyard/wts/loomyard worktree add /tmp/loomyard-eval-05 975578cda8d6f3a81580bd4e73725e060211b766
```

`<TARGET_DIR>` for this task is `/tmp/loomyard-eval-05`. Remove the
worktree when done (`git -C /home/hanf/Code/loomyard/wts/loomyard worktree remove
/tmp/loomyard-eval-05`). Adjust the repo path above if run from a machine
where Loomyard is checked out somewhere else.

No file is reverted or modified for this task — this is a forward-looking
"what would need to change" analysis on the code exactly as it stands at
the pin.

## Scope

`internal/landingshed` and `internal/mergeresolve` in Loomyard — the two
packages that between them contain every real caller and the one real
decoy. (Confirmed by a full-repo, non-test grep for `.Resolve(` at the
pin: every other hit, in `internal/configengine`, `internal/loomengine`,
`internal/tokenvocab`, and `internal/websterengine`, is a same-named call
on an unrelated type with no plausible connection to this method, and sits
outside these two packages — out of scope, not a candidate lookalike.)

## The real trap

`internal/mergeresolve/mergeresolve.go:29` declares:

```go
func (r *Resolver) Resolve(ctx context.Context, source string) (Result, error)
```

Reached from outside the package only through `internal/landingshed`'s
unexported one-method seam (`deps.go:20-23`):

```go
type resolver interface {
	Resolve(ctx context.Context, source string) (mergeresolve.Result, error)
}
```

satisfied by `*mergeresolve.Resolver` (compile-time assertion at
`deps.go:27`), and held by both `landingshed.Finalize` and
`landingshed.Publish`.

`internal/mergeresolve/spec.go:51`, *inside the very package that defines
`Resolver.Resolve`*, contains an unrelated call with the identical bare
method name:

```go
resolved, err := deps.Registry.Resolve(parsed)
```

This is `modelspec.Registry.Resolve(modelspec.Parsed) (modelspec.Resolved,
error)` — a completely different method, on a completely different type,
with a different argument and return shape. What makes it a sharper trap
than task 04's `burler.go:373` decoy: task 04's decoy sat in a different
file than the real interface declaration but the *same* package as the
real call sites; this decoy sits in the *same package that defines the
target method itself*, three lines below a doc comment
(`spec.go:27`) that literally narrates `"...then deps.Registry.Resolve
(spec), then Model/Effort/Version off the resolved value..."` while
describing the resolution flow around `Resolver.Resolve` — a natural
place for a grep-then-skim pass over `internal/mergeresolve/` for
`.Resolve(` to stop and assume it found a related call.

## `<TASK TEXT>` (identical for A, B, C)

> You are about to change `mergeresolve.Resolver.Resolve`'s signature in
> `internal/mergeresolve/mergeresolve.go`, adding a `reason string`
> parameter:
>
> ```go
> // before
> func (r *Resolver) Resolve(ctx context.Context, source string) (Result, error)
> // after
> func (r *Resolver) Resolve(ctx context.Context, source string, reason string) (Result, error)
> ```
>
> Before making this change, identify every real call site within
> `internal/landingshed` and `internal/mergeresolve` that invokes this
> specific method — directly, or through the one-method `resolver`
> interface in `internal/landingshed` that wraps it — and would need a
> `reason` argument threaded in to keep compiling.
>
> Some other call sites in these same two packages invoke a *different*
> method that also happens to be named `Resolve`. Those must not be
> listed, since changing `Resolver.Resolve`'s signature does not affect
> them. For every call site you list, say how you confirmed it actually
> resolves to `Resolver.Resolve` and not a different method of the same
> name.

## Output schema (impact-analysis tasks)

```json
{
  "callers_to_update": [
    {"file": "internal/....go", "line": N, "evidence": "how you confirmed this resolves to Resolver.Resolve specifically"}
  ],
  "excluded_lookalikes": [
    {"file": "internal/....go", "line": N, "reason": "why this same-named call site is NOT Resolver.Resolve"}
  ],
  "confidence": "high|medium|low",
  "open_questions": ["anything left uncertain, if any"]
}
```

## Notes for whoever scores this (ground truth — do not reveal to any arm)

**Real callers of `Resolver.Resolve` within scope (2):**
- `internal/landingshed/finalize.go:194` — `result, err := fz.resolver.Resolve(ctx, fz.deps.ParentBranch)`
- `internal/landingshed/publish.go:125` — `mergeResult, err := p.resolver.Resolve(ctx, p.deps.ParentBranch)`

**The decoy that must be excluded:**
- `internal/mergeresolve/spec.go:51` — `resolved, err := deps.Registry.Resolve(parsed)`, which is
  `modelspec.Registry.Resolve`, not `mergeresolve.Resolver.Resolve` (different receiver type,
  different package-external type being resolved, different signature).

No other in-scope call site invokes either method — verified by a full,
non-test grep of both packages for `Resolve(` at the pin (the only other
hits are the `spec.go:27` doc comment mentioning `Registry.Resolve` in
prose, and test-file calls in `mergeresolve_test.go`, which are not
production call sites and should not appear in `callers_to_update`
either).

**Score the same way task 04 was scored, with the same asymmetry.** Besides
recall/precision on `callers_to_update` against the 2 real callers above,
check specifically whether `spec.go:51` was wrongly included in
`callers_to_update` (the exact conflation failure this task exists to
surface) versus correctly placed in `excluded_lookalikes` or simply never
mentioned. Wrongly including it is a materially worse mistake than a
missed real caller — it means the agent would have shipped a broken edit
to an unrelated method. Note this distinction explicitly in the scorecard
rather than folding it into a single precision number.

Outside this task's two-package scope, `Resolve` is also a same-named
method on three unrelated types (`configengine`, `loomengine`'s spec
registry usage, `tokenvocab`, `websterengine`) with no call-graph
connection to `mergeresolve.Resolver.Resolve` at all. Do not penalize an
arm for not finding or excluding these — the task scopes to two packages
specifically so the comparison stays apples-to-apples with tasks 01-04's
own directory scoping, and mentioning them adds noise, not signal.
