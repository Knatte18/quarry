# Batch: facade-reexports

```yaml
task: "Add `impact` verb for caller-context lookup"
batch: "facade-reexports"
number: 2
cards: 2
verify: go build ./... && go test ./quarry/...
depends-on: [1]
```

## Batch Scope

This batch re-exports batch 1's engine surface through `quarry/facade.go` so `internal/cli` can reach
it without importing anything under `internal/quarryengine` directly, and extends
`quarry/facade_test.go`'s hand-enumerated compile-time checks to cover the new declarations.
It is one batch because those two files are a matched pair: the facade guard does **not** enforce the
alias/delegation property automatically — it is an enumerated list, so a new facade declaration is
covered by nothing until the list is extended by hand, and the guard silently under-covers rather
than failing.

The external interface batch 3 consumes is `quarry.Impact` plus the aliases `quarry.ImpactResult`,
`quarry.ImpactTarget`, `quarry.ImpactDefinition`, `quarry.ImpactCaller`, and `quarry.ImpactRange`.

Batch-local decision beyond `## Shared Decisions`: no options alias is added.
`Impact` takes the existing `quarry.Options`, which is already `query.Options`, so there is nothing
new to alias on the input side — only the result types are new.

## Cards

### Card 8: Facade aliases and the Impact delegation

- **Context:**
  - `internal/quarryengine/impact/types.go`
  - `internal/quarryengine/impact/impact.go`
- **Edits:**
  - `quarry/facade.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Import `internal/quarryengine/impact` and add five type aliases following the file's existing
  `TOC`-prefix convention, each a true `=` alias with a one-line doc comment in the same
  "X is Y, re-exported unchanged" form the surrounding declarations use: `ImpactResult` for
  `impact.Result`, `ImpactTarget` for `impact.Target`, `ImpactDefinition` for `impact.Definition`,
  `ImpactCaller` for `impact.Caller`, and `ImpactRange` for `impact.Range`.

  Add the one-line delegating function
  `Impact(ctx context.Context, opts Options) (ImpactResult, error)` returning
  `impact.Impact(ctx, opts)`, in the same shape as the existing `Callers` delegation.
  Add no options alias: `Impact` takes the existing `Options`.
  Add nothing else — no struct field, no inline computation, no multi-line body, per the property
  this file's own header comment states and its guard file enforces.

  Correct the header comment's "seven-package DAG" phrase to "eight-package DAG" and add the new
  package to the enumeration that follows it, so the list reads as the root leaf package plus lsp,
  registry, treesitter, daemon, toc, query, and the new one.
  This is one of exactly three occurrences of that phrase in the repo; the other two are batch 4's.
- **Commit:** `feat(quarry): re-export the impact package through the facade`

### Card 9: Facade guard extensions

- **Context:**
  - `internal/quarryengine/impact/types.go`
  - `internal/quarryengine/impact/impact.go`
  - `quarry/facade.go`
- **Edits:**
  - `quarry/facade_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Import `internal/quarryengine/impact` and extend all three enumerated blocks by hand.

  In the alias-pair `var` block, add five engine/facade variable pairs following the existing
  `aliasCheck<Name>Engine` / `aliasCheck<Name>Facade` naming: `impact.Result` against
  `ImpactResult`, `impact.Target` against `ImpactTarget`, `impact.Definition` against
  `ImpactDefinition`, `impact.Caller` against `ImpactCaller`, and `impact.Range` against
  `ImpactRange`.

  In `init()`, add the round-trip assignment for each of those five pairs — engine value into the
  facade-typed variable, then straight back into the engine-typed variable, with no conversion on
  either side.
  Correct that function's doc comment from "twenty-one aliased types" to twenty-six.

  In the blank-identifier func-type `var` block, add
  `_ func(context.Context, Options) (ImpactResult, error) = Impact`, and correct that block's leading
  comment from "The fourteen blank-identifier assignments" to fifteen.

  Do not extend `TestFacadeSentinels_Identity`'s table: this task adds no new sentinel error value,
  so its nine-entry table and its doc comment's count both stay correct.
- **Commit:** `test(quarry): extend the facade guard to cover the impact re-exports`

## Batch Tests

`verify: go build ./... && go test ./quarry/...` is the right scope: `quarry/facade_test.go` is
almost entirely compile-time — the alias pairs, the `init()` round-trips, and the blank-identifier
func-type assignments all fail at build time rather than at assert time — so the compile gate is the
substance of the check and the single runtime test in the package (`TestFacadeSentinels_Identity`)
is unchanged by this batch.
`go build ./...` additionally proves the new facade declarations do not break any existing consumer.

Edited test file: `quarry/facade_test.go` (card 9).
No new test file.
