# Plan: Batch-answer contract: per-target coverage + fail-closed Status helpers

```yaml
task: 'Batch-answer contract: per-target coverage + fail-closed Status helpers'
slug: batch-answer-contract
approved: true
started: '20260909-061811'
parent: main
root: ""
verify: go vet ./...
discussion_sha: 991aae1c65528b4aed72fcfb3aa5a277757b5b72
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: engine-vocabulary
    file: 01-engine-vocabulary.md
    depends-on: []
    verify: go test ./internal/engine/ ./quarry/
  - number: 2
    name: coverage-verifiers
    file: 02-coverage-verifiers.md
    depends-on: []
    verify: go test ./internal/engine/
  - number: 3
    name: consumers-and-docs
    file: 03-consumers-and-docs.md
    depends-on: [1, 2]
    verify: go test ./internal/cli/ ./quarry/ ./glyph/
```

## Shared Decisions

### Decision: additive-only, semver-minor

- **Decision:** Every change compiles against existing callers unchanged. No exported signature
  changes, no removed or renamed exported symbols, and no change to the emitted JSON key set — no
  new struct field, no new struct tag, no renamed tag. New exported API is limited to one package
  var per package — `engine.Statuses`, re-exported through the facade as `quarry.Statuses` — and
  two methods on existing types.
- **Rationale:** The task exists so loomyard can adopt through a plain `go.mod` bump, and
  `internal/engine/answer.go`'s own file header fixes its JSON tag set as closed. Methods and a
  package var add no tag, so that rule is satisfied without touching the Shared Decision the header
  names.
- **Applies to:** all batches

### Decision: doc-comment register

- **Decision:** Every new exported symbol gets a long doc comment in this codebase's own register —
  it states what the symbol is, why the shape is what it is, and names the alternative that was
  rejected. Every new unexported verifier gets one saying which invariant it guards and why it
  panics rather than returning an error.
- **Rationale:** The surrounding files write exactly this way; a terse one-liner beside them would
  read as unfinished. The rejected-alternative clause is what stops a later reader "simplifying"
  `Known()` into a range over `Statuses`.
- **Applies to:** all batches

### Decision: producer-side enforcement is a panic, once, at the engine

- **Decision:** Each batch verb verifies its own answer slice immediately before returning and
  panics on a violation. The `quarry` facade never re-verifies, and no consumer keeps a guard for
  what the producer now guarantees.
- **Rationale:** A coverage violation is unreachable by construction; if it fires the engine is
  broken and every answer in the batch is untrustworthy. `Name` returns no error at all by design,
  so an error return is unavailable there without a breaking change, and an error from `Resolve`
  would hand the caller one more condition to guard — which is the defect this task removes. The
  repository already panics on this class of invariant violation at
  `internal/engine/strategy.go:74`.
- **Applies to:** all batches

### Decision: the two panic message shapes are fixed

- **Decision:** An arity violation names the verb and the got/want lengths and carries no index. An
  echo mismatch names the verb, the index, and the got/want pair. For `Name`, whose echo is two
  fields, the message names the diverging field, `Unit` is compared first, and the verifier panics
  on the first diverging field rather than accumulating both.
- **Rationale:** An arity violation genuinely has no offending index, so one combined message shape
  would force an invented one. The `Unit`-first tie-break plus the field label are what make a
  two-field divergence produce one deterministic message, which is what lets the tests assert
  message content at all.
- **Applies to:** coverage-verifiers, consumers-and-docs

### Decision: `Known()` is a switch, never a range over `Statuses`

- **Decision:** `Status.Known()`'s body is a `switch` over the four constants. It is never
  implemented by ranging the exported `Statuses` slice.
- **Rationale:** `Statuses` is an exported slice and therefore mutable by any caller, so the range
  form would let `quarry.Statuses[0] = "nonsense"` silently rewrite `Known()`'s answer process-wide
  — the opposite of fail-closed. The switch also keeps `Known()` and `Statuses` two independently
  written enumerations of one vocabulary, which is what makes the truth-table test a real
  cross-check instead of a tautology.
- **Applies to:** engine-vocabulary

### Decision: Go verify commands carry no `PYTHONPATH=` prefix

- **Decision:** Every `verify:` in this plan is a bare Go command. The `PYTHONPATH= ` isolation
  prefix is not used.
- **Rationale:** That prefix exists for Python/mill projects so a test subprocess does not inherit
  the mill cache scripts directory. This repository is Go — `CLAUDE.md` forbids introducing Python —
  and the prefix would be meaningless noise in front of `go test`.
- **Applies to:** all batches

### Decision: verify scopes are per-package, and the module-wide gate is `go vet ./...`

- **Decision:** Each batch's `verify:` names only the packages that batch touches. The overview's
  module-wide `verify:` is `go vet ./...`.
- **Rationale:** `go test ./...` after every implementer and fixer round would re-run the whole
  suite many times per batch for no added signal, since batches 1 and 2 touch only
  `internal/engine` and `quarry`. `go vet ./...` is the cheap cross-package compile-and-vet gate
  that catches a regression outside a batch's own scope at the batch boundary. `pipeline.done_gate`
  is already configured as `go test ./... && golangci-lint run`, so the full suite and the linter
  still run once before the task is marked done.
- **Applies to:** all batches

## All Files Touched

- `docs/glyph.md`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/engine/answer.go`
- `internal/engine/answer_test.go`
- `internal/engine/name.go`
- `internal/engine/name_test.go`
- `internal/engine/resolve.go`
- `internal/engine/resolve_test.go`
- `quarry/quarry.go`
- `quarry/quarry_test.go`
- `quarry/text.go`
