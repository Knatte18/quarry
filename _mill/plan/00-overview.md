# Plan: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
slug: 'diff-to-symbols'
approved: false
started: '20260905-154250'
parent: 'main'
root: ""
verify: go vet ./...
discussion_sha: a380c2974510fe220a4c23a627c8c8ec835f7d3d
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: engine-symbol-seam
    file: 01-engine-symbol-seam.md
    depends-on: []
    verify: go test ./internal/engine/ -run 'TestSymbolOffsets|TestSignatureSpan'
  - number: 2
    name: engine-unit-exports
    file: 02-engine-unit-exports.md
    depends-on: []
    verify: go test ./internal/engine/ -run 'TestPackageClause|TestUnitsForClauseMap|TestClauseMapForFiles|TestWalk|TestRepoTOC|TestResolve|TestExpand|TestSpansOf|TestRoundTrip'
  - number: 3
    name: delta-types-and-tokens
    file: 03-delta-types-and-tokens.md
    depends-on: [1]
    verify: go test ./internal/engine/ -run 'TestTokenStreams|TestIdenticalModuloName|TestBodyTokenSimilarity'
  - number: 4
    name: delta-core
    file: 04-delta-core.md
    depends-on: [3]
    verify: go test ./internal/engine/ -run 'TestDelta'
  - number: 5
    name: gitsrc
    file: 05-gitsrc.md
    depends-on: []
    verify: go test ./internal/gitsrc/
  - number: 6
    name: facade-delta
    file: 06-facade-delta.md
    depends-on: [2, 4, 5]
    verify: go test ./quarry/ -run 'TestDelta'
  - number: 7
    name: delta-renderers
    file: 07-delta-renderers.md
    depends-on: [6]
    verify: go test ./quarry/ -run 'TestRenderDelta'
  - number: 8
    name: cli-delta-verb
    file: 08-cli-delta-verb.md
    depends-on: [7]
    verify: go test ./internal/cli/ -run 'TestParseArgs|TestRun|TestCodeFor'
  - number: 9
    name: goldens-history-docs
    file: 09-goldens-history-docs.md
    depends-on: [8]
    verify: go test ./quarry/ -run 'TestDeltaGolden|TestDeltaRealHistory'
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: the discussion is the specification, and its exact tier is the highest-risk part

- **Decision:** `_mill/discussion.md` is the authoritative specification for every behavioural
  question in this plan.
  Where a card's Requirements restate a rule, the restatement is a convenience, not a
  substitute — an implementer who finds the two disagreeing follows the discussion and the
  disagreement is a plan defect worth reporting.
  **The exception is a departure this plan states explicitly as its own decision**, of which there is
  exactly one: the goldens-location decision below, which names the discussion's Testing section as
  what it departs from and gives the reason.
  A stated departure wins over the discussion; an unstated disagreement never does.
  Any future departure must be recorded the same way — as its own Shared Decision naming the passage
  it overrides — or it is a defect rather than a decision.
  Its own closing Q&A entry records that no discussion-review round ever returned APPROVE and
  names conditions 1–7 of the exact-tier rename classifier as the part most likely to still
  carry a defect.
- **Rationale:** the discussion is 1227 lines and settles six rounds of review; a plan that
  paraphrased it would become a second, drifting specification, which is the parallel-implementation
  failure this codebase's one-implementation rule exists to prevent.
- **Applies to:** all batches

### Decision: no existing behaviour, key set, golden or exit code changes

- **Decision:** this task is purely additive.
  `toc`, `resolve` and `expand` keep their behaviour, their emitted key sets and their committed
  goldens byte-for-byte.
  The three new `Symbol` fields are all `json:"-"`, so no existing JSON output moves by one byte.
  Two refactors touch existing code paths — batch 2 rewrites `dirPackage` and `unitFor` to call
  newly exported helpers — and both are required to be behaviour-preserving, proved by the existing
  `internal/engine` suite staying green.
- **Rationale:** the discussion's Constraints section requires every existing committed golden to
  stay byte-identical wherever it lives at merge time.
- **Applies to:** all batches

### Decision: `internal/cli` reaches git error identity through the facade, never by importing `internal/gitsrc`

- **Decision:** `internal/gitsrc` declares three sentinels — `ErrNotARepository`,
  `ErrRootNotTopLevel`, `ErrUnknownRevision` — and, for the two that must be spelled with a value
  in a user-facing sentence, two typed errors whose `Unwrap` returns the matching sentinel:
  `RootNotTopLevelError` (fields `Root`, `TopLevel`) and `UnknownRevisionError` (field `Rev`).
  All five are aliased in package `quarry`, exactly as `ErrTargetNotFound` and `NotATypeError`
  already are, and the CLI matches only the facade's names with `errors.Is` and `errors.As`.
- **Rationale:** `internal/mcpserver/layering_test.go` records that the facade-only rule — no
  package above the facade imports `internal/engine` or anything below it directly — holds by
  convention everywhere it holds today, "including in internal/cli".
  A direct `internal/gitsrc` import from the CLI would be the first break in that convention.
  The paired sentinel-plus-typed-error shape satisfies the discussion's own two requirements at
  once: it keeps the documented `errors.Is(err, gitsrc.ErrUnknownRevision)` check working, and it
  gives the CLI the revision string and the top-level path it must spell in its own sentence
  without parsing any message — the pattern `NotATypeError` already sets in `runExpand`.
- **Applies to:** all batches

### Decision: the seven required goldens live in `quarry/testdata/delta/`

- **Decision:** the seven golden cases are committed as fourteen files (a `.json` and a `.txt` per
  case) under `quarry/testdata/delta/`, produced by a test in package `quarry` under that package's
  own `-update` flag.
  This is a deliberate, stated departure from the discussion's Testing section, which named
  `internal/engine/testdata/delta/` or `internal/cli/testdata/` as the two candidate locations.
- **Rationale:** the goldens are required in both the JSON view and the text view, and both
  renderers are declared in package `quarry`.
  `internal/engine` cannot import them — that is an import cycle — so it can produce neither view.
  `internal/cli` can produce both, but only by driving `Run`, which would need a throwaway git
  repository built per case; the seven cases are batches of in-memory byte pairs fed to the pure
  `Delta` path, and building seven fixture repositories to express them would put the git layer
  inside a fixture whose subject is the core.
  Package `quarry` is the one place that holds the pure core's answer and both renderers at once.
- **Applies to:** goldens-history-docs

### Decision: Go verify commands carry no `PYTHONPATH=` prefix

- **Decision:** every `verify:` in this plan is a native `go test` invocation scoped with `-run` (or
  by package, where the whole package is new in that batch).
  No command carries the `PYTHONPATH= ` prefix.
- **Rationale:** that prefix exists to keep a Python test subprocess from inheriting the mill cache
  scripts directory.
  This repository is Go; `CLAUDE.md` forbids introducing Python here, so the prefix would name an
  interpreter this repository never runs.
- **Applies to:** all batches

### Decision: new test function names are load-bearing on the `verify:` `-run` patterns

- **Decision:** each batch's `verify:` selects its tests by `-run` over the exact test-function
  names its cards require.
  A card that renames or re-prefixes a test function must change that batch's `verify:` in the same
  commit.
- **Rationale:** a `-run` pattern that silently matches nothing is a verify command that passes
  without running anything, which is worse than a failing one.
  This mirrors `internal/cli/after_test.go`'s own recorded rule that `TestAfterGoldens`' name and
  its regeneration command are load-bearing on each other.
- **Applies to:** all batches

### Decision: the pure core never derives a unit, and never touches git or the filesystem

- **Decision:** `(*engine.Repo).Delta` takes `[]DeltaEntry`, each carrying its own `BeforeUnit` and
  `AfterUnit`, and reads nothing outside its arguments.
  Every unit derivation happens above it, in `(*quarry.Repo).DeltaGit`, through the three exported
  engine helpers batch 2 adds.
- **Rationale:** the discussion's first Decision, restated here because it is the boundary two
  batches implement from opposite sides.
- **Applies to:** delta-core, facade-delta

### Decision: batches 1, 2 and 5 are independent roots

- **Decision:** the symbol byte-offset seam, the exported clause and unit helpers, and
  `internal/gitsrc` share no file and no dependency, and may be implemented in any order or
  concurrently.
- **Rationale:** batch 1 edits `internal/engine/answer.go` and `internal/engine/golang.go`; batch 2
  edits `internal/engine/walk.go` and creates `internal/engine/units.go`; batch 5 creates a package
  that does not yet exist.
  The three file sets are disjoint.
- **Applies to:** engine-symbol-seam, engine-unit-exports, gitsrc

## All Files Touched

- `docs/rewrite-plan.md`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/doc.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/usage.go`
- `internal/engine/answer.go`
- `internal/engine/delta.go`
- `internal/engine/delta_answer.go`
- `internal/engine/delta_order_test.go`
- `internal/engine/delta_rename_test.go`
- `internal/engine/delta_test.go`
- `internal/engine/delta_tokens.go`
- `internal/engine/delta_tokens_test.go`
- `internal/engine/golang.go`
- `internal/engine/offsets_test.go`
- `internal/engine/units.go`
- `internal/engine/units_test.go`
- `internal/engine/walk.go`
- `internal/gitsrc/doc.go`
- `internal/gitsrc/errors.go`
- `internal/gitsrc/fixture_test.go`
- `internal/gitsrc/gitsrc.go`
- `internal/gitsrc/gitsrc_test.go`
- `quarry/delta.go`
- `quarry/delta_golden_test.go`
- `quarry/delta_history_test.go`
- `quarry/delta_test.go`
- `quarry/doc.go`
- `quarry/quarry.go`
- `quarry/render.go`
- `quarry/render_test.go`
- `quarry/repo.go`
- `quarry/testdata/delta/created.json`
- `quarry/testdata/delta/created.txt`
- `quarry/testdata/delta/deleted.json`
- `quarry/testdata/delta/deleted.txt`
- `quarry/testdata/delta/entry-error.json`
- `quarry/testdata/delta/entry-error.txt`
- `quarry/testdata/delta/mixed.json`
- `quarry/testdata/delta/mixed.txt`
- `quarry/testdata/delta/modified.json`
- `quarry/testdata/delta/modified.txt`
- `quarry/testdata/delta/rename-evidence.json`
- `quarry/testdata/delta/rename-evidence.txt`
- `quarry/testdata/delta/rename-exact.json`
- `quarry/testdata/delta/rename-exact.txt`
- `quarry/text.go`
- `quarry/text_test.go`
