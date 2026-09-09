# Batch: coverage-verifiers

```yaml
task: 'Batch-answer contract: per-target coverage + fail-closed Status helpers'
batch: coverage-verifiers
number: 2
cards: 3
verify: go test ./internal/engine/
depends-on: []
```

## Batch Scope

This batch delivers the per-target coverage half of the task: one unexported verifier per batch
verb, each living in its own verb's file, each called immediately before that verb returns its
answer slice, and their white-box tests. It is one batch because the two verifiers are the same
shape written twice, their message formats are fixed together by one Shared Decision, and their
tests share a single house style for asserting a panic.

The external interface batch 3 consumes is not a symbol but a guarantee: after this batch, `resolve`
and `Name` cannot return an answer slice that is short, long, or out of order, which is what makes
`internal/cli/cli.go`'s two consumer-side arity guards unreachable and therefore deletable.

Batch-local decision, differing from nothing in the overview but worth stating: the verifier for
`resolve` is called from the unexported worker `resolve`, not from the exported `Resolve`. The
exported wrapper only builds the `unitMemo` and delegates, so verifying there would leave the
white-box test path that calls the worker directly unguarded, and would put the check away from the
loop whose invariant it guards.

## Cards

### Card 4: `resolve`'s coverage verifier

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/strategy.go`
- **Edits:**
  - `internal/engine/resolve.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add an unexported function `verifyResolveCoverage(targets []string, results []ResolveResult)` to
  `internal/engine/resolve.go`, placed immediately after the unexported worker `resolve`, and call
  it from `resolve` on the success path only.

  The body performs two checks, in this order:

  1. Arity. When `len(results) != len(targets)`, panic with a message built by `fmt.Sprintf` in
     exactly this shape, naming the verb and the got/want lengths and no index:

     ```go
     panic(fmt.Sprintf("engine: resolve returned %d results for %d targets", len(results), len(targets)))
     ```

  2. Per-index echo. Ranging `targets` by index, when `results[i].Target` differs from `targets[i]`,
     panic with exactly this shape, naming the verb, the index, and both echo values:

     ```go
     panic(fmt.Sprintf("engine: resolve result %d answers %q; want %q", i, results[i].Target, targets[i]))
     ```

     The `%q` verbs are what make the exemplar `engine: resolve result 2 answers "a/b#X"; want
     "c/d#Y"` come out with its quotes. Panic on the first divergence; do not accumulate.

  A zero-length `targets` is a no-op, not a panic — the arity check passes with both lengths zero
  and the loop body never runs. `fmt` is already imported in this file; add no import.

  Call the verifier from `resolve` on the success return path only:
  `verifyResolveCoverage(targets, results)` immediately before `return results, nil`. Do not call it
  before, or instead of, the `return nil, err` early return inside the loop — a nil slice with a
  non-nil error is this verb's documented whole-call failure shape, not a coverage violation, and
  verifying there would panic on every legitimate engine failure.

  The verifier's doc comment must state which invariant it guards — that `resolve` answers every
  target positionally, one answer per target in argument order with `Target` echoed verbatim on
  every answer, rejections included — and why it panics rather than returning an error: the
  violation is unreachable by construction, so if it fires the engine is broken and every answer in
  the batch is untrustworthy, while an error return would hand the caller exactly the condition this
  task exists to stop callers from having to check. Name `internal/engine/strategy.go`'s
  duplicate-`Strategy`-registration panic as the in-repository precedent for panicking on an
  invariant violation of this class. State in the same comment that the failure path is deliberately
  not verified, and why.
- **Commit:** `feat(engine): verify resolve's positional coverage before returning`

### Card 5: `Name`'s coverage verifier

- **Context:**
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/name.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add an unexported function `verifyNameCoverage(decls []Declaration, results []NameResult)` to
  `internal/engine/name.go`, placed immediately after `Name`, and call it from `Name`.

  The body performs the same two checks as `verifyResolveCoverage` in
  `internal/engine/resolve.go`, adapted to this verb's two-field echo:

  1. Arity. When `len(results) != len(decls)`, panic with exactly this shape:

     ```go
     panic(fmt.Sprintf("engine: name returned %d results for %d declarations", len(results), len(decls)))
     ```

  2. Per-index echo, ranging `decls` by index. `Unit` is compared first: when `results[i].Unit`
     differs from `decls[i].Unit`, panic with

     ```go
     panic(fmt.Sprintf("engine: name result %d echoes unit %q; want %q", i, results[i].Unit, decls[i].Unit))
     ```

     and return from the panic site — do not fall through to the `Target` comparison for that index.
     Only when `Unit` matched is `Target` compared: when `results[i].Target` differs from
     `decls[i].Decl`, panic with

     ```go
     panic(fmt.Sprintf("engine: name result %d echoes target %q; want %q", i, results[i].Target, decls[i].Decl))
     ```

  The `Unit`-first ordering and the field label in each message are load-bearing, not incidental:
  without the ordering a result diverging in both fields would have two equally valid messages, and
  without the label a `Target`-only divergence would print a bare got/want pair with nothing saying
  which field it described. Say so in the doc comment.

  A zero-length `decls` is a no-op. `fmt` is already imported in this file; add no import.

  `Name` has exactly one return path, so the call is unconditional: `verifyNameCoverage(decls,
  results)` immediately before `return results`. Unlike `resolve`, this verb has no whole-call
  failure path to exclude — it returns no error at all — and the doc comment should say that, so the
  asymmetry between the two verifiers' call sites reads as deliberate.

  The doc comment otherwise mirrors card 4's: which invariant it guards (`Name` answers every
  declaration positionally, with `Unit` and `Target` echoing the input's `Unit` and `Decl` verbatim
  on every result, failures included, which is exactly what `nameFailure` exists to guarantee) and
  why a panic rather than an error — here with the extra reason that `Name` returns no error at all
  by design, so an error return is not available without a breaking signature change.
- **Commit:** `feat(engine): verify Name's positional coverage before returning`

### Card 6: white-box tests for both verifiers

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/classify_test.go`
  - `internal/engine/name.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/name_test.go`
  - `internal/engine/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  The verifiers are unexported, so their tests are white-box, in package `engine`, and call them
  directly with hand-built slices — no repository, no filesystem, no fixtures.

  Follow the panic-test house style of `TestRegister_PanicsOnDuplicateLanguage` in
  `internal/engine/classify_test.go`: a deferred closure calling `recover()` that fails when
  `recover()` returns nil. Go one step beyond that precedent by also asserting the recovered value's
  message, because the panic message is the only diagnostic an engine bug of this class will ever
  produce. Recover the value as a `string` via a type assertion on the `any` `recover()` returns,
  failing the test when the assertion does not hold, and compare against the full expected message
  with `!=` rather than a substring match, so a reworded message is caught rather than tolerated.

  Add to `internal/engine/resolve_test.go`, for `verifyResolveCoverage`:

  - a passing case — a correct slice of the same length whose every element echoes its target — that
    asserts no panic occurred;
  - a zero-length case, asserting a no-op rather than a trip;
  - a short-slice case and a long-slice case, each asserting the exact arity message and asserting
    that the message does not name an index, since an arity violation has none;
  - an echo-mismatch case at a non-zero index, asserting the exact echo message including its index
    and its two quoted values.

  Add to `internal/engine/name_test.go`, for `verifyNameCoverage`, the same five shapes over
  `Declaration` and `NameResult` values, plus one further case that is specific to this verb: a
  result at some index whose `Unit` and whose `Target` both diverge from their input, asserting the
  message reports `Unit` alone. That case is the one where an unspecified tie-break would make the
  assertion non-deterministic, so it is the one that pins the Shared Decision.

  Do not modify `TestResolve_ArgumentOrderAndArity` or `TestName_BatchSemantics`. Both already
  assert the positional contract end to end — arity, per-index echo, the repeated-target case, and
  the empty-input empty-non-nil-slice case — and both must keep passing verbatim. If the verifiers
  are right, they will.
- **Commit:** `test(engine): cover both coverage verifiers and their panic messages`

## Batch Tests

`verify: go test ./internal/engine/` runs the one package this batch touches. It covers the new
verifier tests in `internal/engine/resolve_test.go` and `internal/engine/name_test.go` (card 6) and,
critically, re-runs the whole existing engine suite — every test that calls `Resolve`, `resolve`, or
`Name` now runs through the new verifiers, so any real coverage violation anywhere in the existing
fixtures surfaces as a panic in this same run rather than at a consumer later.

The scope is one package because both verifiers and both call sites are inside it. The overview's
module-wide `go vet ./...` runs at the batch boundary; the full `go test ./...` runs once at
`pipeline.done_gate` before the task is marked done.
