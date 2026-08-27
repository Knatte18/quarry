# Batch: architecture-guards

```yaml
task: Thin quarry/ facade over internal/quarryengine
batch: architecture-guards
number: 3
cards: 2
verify: go test ./... && go test -tags lsp -run "^$" ./...
depends-on: [2]
```

## Batch Scope

This batch adds the two guards that make the new structure durable rather than merely present: a layering test that pins the package DAG batch 2 created, and a facade test that pins the facade's alias identity. Both are new files; neither touches production code. It is a separate batch from the repackaging because both guards need the finished layout to walk, and because a guard that is written in the same commit as the thing it guards proves less than one written against a tree that already compiles.

It has no interface for a later batch to consume, and it does not overlap with batch 4 — batch 4 edits doc comments in the four subpackages and the engine overview, this batch creates two test files.

## Cards

### Card 11: Pin the package DAG with a layering guard

- **Context:**
  - `internal/quarryengine/seam_enforcement_test.go`
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/daemon/ensureserver.go`
  - `internal/quarryengine/daemon/daemontest/daemontest.go`
  - `internal/quarryengine/query/refs.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/layering_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/quarryengine/layering_test.go` declaring `package quarryengine`, with a table-driven `TestLayeringInvariant_ImportDirections`. The table encodes the allowed set of `internal/quarryengine/...` imports per package: `quarryengine` (the root) may import none of them; `lsp` and `registry` may import the root only; `daemon` may import the root, `registry`, and `lsp`; `query` may import the root, `registry`, `lsp`, and `daemon`; `daemontest` may import `daemon`. Imports outside the `github.com/Knatte18/quarry/internal/quarryengine` prefix are not the subject of this test and are ignored — the seam guard already covers banned imports, and this one covers direction only. Unlike `seam_enforcement_test.go`, this walk visits BOTH production and `_test.go` files, because a test file importing across the DAG is the realistic way the layering rots. There is exactly one carve-out: any `_test.go` file, in any package, may import `internal/quarryengine/daemon/daemontest` — that is what `daemontest` exists for. Nothing else is exempt: a `registry` test importing `daemon`, or a `query` test importing `daemon` directly instead of through `daemontest`, must fail. Failures name the offending file and the offending import path. Keep the same `runtime.Caller(0)`-anchored root resolution and the same "scanned zero files → `t.Fatal`" guard `seam_enforcement_test.go` uses. Do not edit `seam_enforcement_test.go` — the two tests share the shape of their directory walk but not a helper; duplicating ~15 lines of `filepath.WalkDir` is cheaper than coupling two guards that answer different questions. Before accepting this test as green, plant a violating import (for example, `registry` importing `daemon`), confirm the test fails and names it, then revert.
- **Commit:** `test(quarry): pin the internal/quarryengine package DAG with a layering guard`

### Card 12: Pin the facade's alias identity

- **Context:**
  - `quarry/facade.go`
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/position.go`
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/query/symbol.go`
- **Edits:** none
- **Creates:**
  - `quarry/facade_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/facade_test.go` declaring `package quarry`. It asserts that the facade is a pure re-export rather than a wrapper, which is the one way it can break silently. For each of the fourteen aliased types, assign a value of the underlying engine type to the facade type and back with no conversion — an assignment that compiles only if the alias is a true `=` alias and fails to compile if someone later turns it into a defined type. For each of the seven sentinels, assert identity against the engine value (for example, `if ErrNoLanguage != quarryengine.ErrNoLanguage { t.Error(...) }`), so a re-created `errors.New` is caught rather than a merely equal message. Reference each of the eight delegating functions — assigning each to a variable of its expected `func` type is enough — so a signature drift fails the build. This test needs no runtime behaviour: it is a compile-time surface check with a handful of identity assertions, and that is deliberate. Do not edit `quarry/facade.go`; if a mismatch is found, that is a real defect in batch 2 card 9 and must be fixed there in `facade.go`, not papered over here.
- **Commit:** `test(quarry): assert the facade is a pure alias re-export`

## Batch Tests

`verify:` runs `go test ./... && go test -tags lsp -run "^$" ./...`, unchanged from batch 2 — both new files are ordinary untagged tests picked up by the first invocation, and the second keeps the tagged files compiling as the batch boundary moves.

The two new tests cover a gap the existing suite structurally cannot: `layering_test.go` covers the import directions between the five engine packages plus `daemontest`, and `facade_test.go` covers alias-versus-defined-type identity across the facade's 29 identifiers. Neither has a meaningful runtime scenario — the compiler and a handful of pointer comparisons do the work — so the acceptance bar for both is the plant-fail-revert exercise named in each card, not an assertion count. A guard that has never been observed to fail has not been tested.
