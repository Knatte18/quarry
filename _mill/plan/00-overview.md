# Plan: Engine core (T3)

```yaml
task: "Engine core (T3)"
slug: "engine-core"
approved: false
started: "20260903-183356"
parent: "main"
root: ""
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go vet ./... && CGO_ENABLED=1 go test ./internal/...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: layout
    file: 01-layout.md
    depends-on: []
    verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
  - number: 2
    name: text-rules
    file: 02-text-rules.md
    depends-on: [1]
    verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
  - number: 3
    name: answer-and-walk
    file: 03-answer-and-walk.md
    depends-on: [1, 2]
    verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
  - number: 4
    name: glyph-symbols
    file: 04-glyph-symbols.md
    depends-on: [3]
    verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
  - number: 5
    name: spans
    file: 05-spans.md
    depends-on: [4]
    verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
  - number: 6
    name: goldens-and-round-trip
    file: 06-goldens-and-round-trip.md
    depends-on: [5]
    verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go vet ./... && CGO_ENABLED=1 go test ./internal/...
```

## Shared Decisions

### Decision: the glyph package is read, never modified

- **Decision:** `glyph/` is imported and its `Parse`/`String` are called; no file under it is edited
  by any batch. Every question about what a glyph may spell — whether a unit is legal, whether an id
  round-trips, whether type parameters belong in a member — is answered by calling that package, not
  by restating a rule in the engine.
- **Rationale:** `docs/glyph.md` makes the `glyph` package the one implementation of the grammar, and
  the discussion's Scope puts it out of bounds. A second implementation inside the engine is exactly
  the drift the rule exists to prevent, and the round-trip criterion cannot catch it, because it
  compares two readings of one walk.
- **Applies to:** all batches

### Decision: the emitted key set is plan §4's and is closed

- **Decision:** `DirAnswer` emits `dir`, `package`, `language`, `doc`, `files`, `dirs`; `FileEntry`
  emits `name`, `header`, `test`, `generated`, `package`, `language`, `symbols`, plus the two failure
  keys `lossy` and `error`; `Symbol` emits `id`, `kind`, `file`, `start`, `sigend`, `end`,
  `signature`, `doc`. Every optional key is `omitempty`; `test: false`, `generated: false` and an
  empty `dirs` are never emitted. No key is added or renamed without amending this decision.
- **Rationale:** this task's done-criterion pins the envelope byte for byte, and the facade and CLI
  that wrap it land in a later task on top of exactly these payload objects. The two failure keys are
  absent on the happy path, so the plan's own examples still reproduce.
- **Applies to:** answer-and-walk, glyph-symbols, spans, goldens-and-round-trip

### Decision: nothing is cached, including the ignore set

- **Decision:** `Repo` holds the repository root and nothing else. The gitignore pattern set is
  collected fresh on every `TOC` and `SpansOf` call, extended per directory as the walk descends,
  and discarded when the call returns. No index, no daemon, no concurrency.
- **Rationale:** the task's constraints say every answer reads source as it is at that moment, and a
  `.gitignore` is source for this purpose. A process-lifetime pattern set would go stale under the
  long-lived MCP server a later task adds, and the resulting wrong file list would be
  indistinguishable from a bug in the walk.
- **Applies to:** answer-and-walk, spans

### Decision: fixture trees split by whether they involve .gitignore

- **Decision:** fixtures that do not exercise gitignore behaviour are committed under
  `internal/engine/testdata/`. Fixtures that do — and the symlink, unreadable-file and
  creation-order cases — are built at run time under `.scratch/engine-tests/<test-name>/` through
  the shared `writeScratchTree` helper and removed in `t.Cleanup`. `t.TempDir()` is never used.
- **Rationale:** a committed tree cannot contain a file its own `.gitignore` excludes — git refuses
  to track it without a force-add, and a force-added-but-ignored file is exactly the confusing state
  those tests exist to reason about. The system temp directory is banned by this task's constraints;
  `.scratch/` is the sanctioned location and is gitignored at the repository root, which also keeps
  run-time trees invisible to the round trip over quarry itself.
- **Applies to:** text-rules, answer-and-walk, spans

### Decision: committed Go fixtures are part of the round trip, and that is fine

- **Decision:** the `.go` files under `internal/engine/testdata/` are walked by the round trip over
  quarry itself and are not excluded from it.
- **Rationale:** the round trip compares what `toc` listed against what the span lookup returns, and
  both read the same files through the same rules — a deliberately broken fixture is lossy on both
  sides, an invalid-UTF-8 fixture yields an error entry and no symbols on both sides, and a fixture
  in a directory the alphabet cannot spell yields no symbols on both sides. Excluding them would
  weaken the criterion for no gain; including them is free coverage of the awkward cases.
- **Applies to:** answer-and-walk, glyph-symbols, spans, goldens-and-round-trip

### Decision: every batch leaves the tree building and every test passing

- **Decision:** each batch's `verify:` is `CGO_ENABLED=1 go build ./...` followed by
  `CGO_ENABLED=1 go test ./internal/...`, and the module-wide `verify:` run at every batch boundary
  is the same pair with `CGO_ENABLED=1 go vet ./...` between them. The module-wide gate runs the
  tests too, not vet alone: this task's own done-criterion — the quarry round trip and the Loomyard
  goldens — is ordinary Go tests, and a gate that skipped them would never exercise the thing the
  task is for. No batch may leave a package uncompilable or a test deleted without its subject.
- **Rationale:** the build is part of the gate because the cgo guard has no test files at all — a
  build is the only thing that exercises it — and because Go compiles a package as a whole, so a
  half-applied type change is a build failure rather than a test failure. `./internal/...` rather
  than `./...` scopes each batch to what it touches: `glyph/` is never modified by this task.
- **Applies to:** all batches

### Decision: renames go through git mv, never delete-and-recreate

- **Decision:** every file relocation in batch 1 is expressed as a `Moves:` pair and performed with
  `git mv` before any edit to the moved file; the edits that follow are surgical — package clause,
  imports, identifier retargets, doc-comment path references.
- **Rationale:** batch 1 relocates twenty files at once. Written from scratch and deleted, the diff
  is unreviewable and git loses the rename history that makes the later batches' changes legible
  against the original.
- **Applies to:** layout

### Decision: the discussion's own contract wording is the source of truth for comments

- **Decision:** where a card asks for a rule to be stated in a comment — why the guard is its own
  package, why the receiver's type parameters are stripped, why the symlink test keys on `Type()`
  rather than `IsDir()`, why the unit lookup is literal-first — the comment states the rule *and*
  the failure it prevents. Where the discussion records a gap in the identifier contract rather than
  closing it, the comment records it as a gap.
- **Rationale:** several of these rules cannot be caught by the round trip, because it compares two
  readings of one walk. A comment naming the failure is what a later reader has instead.
- **Applies to:** all batches

## All Files Touched

- `README.md`
- `go.mod`
- `go.sum`
- `internal/cgoguard/cgoguard.go`
- `internal/cgoguard/cgoguard_nocgo.go`
- `internal/engine/answer.go`
- `internal/engine/answer_test.go`
- `internal/engine/classify.go`
- `internal/engine/classify_test.go`
- `internal/engine/doc.go`
- `internal/engine/errors.go`
- `internal/engine/extension.go`
- `internal/engine/extension_test.go`
- `internal/engine/glyph_test.go`
- `internal/engine/golang.go`
- `internal/engine/golang_test.go`
- `internal/engine/golden_test.go`
- `internal/engine/headers.go`
- `internal/engine/headers_test.go`
- `internal/engine/ignore.go`
- `internal/engine/ignore_test.go`
- `internal/engine/loomyard_test.go`
- `internal/engine/nodes.go`
- `internal/engine/repo.go`
- `internal/engine/repo_test.go`
- `internal/engine/resolve.go`
- `internal/engine/resolve_test.go`
- `internal/engine/roundtrip_test.go`
- `internal/engine/scratchtree_test.go`
- `internal/engine/strategy.go`
- `internal/engine/testdata/broken/invalid.go`
- `internal/engine/testdata/broken/ok.go`
- `internal/engine/testdata/broken/syntax.go`
- `internal/engine/testdata/foo_test/literal.go`
- `internal/engine/testdata/glyphs/blank.go`
- `internal/engine/testdata/glyphs/decls.go`
- `internal/engine/testdata/glyphs/generic.go`
- `internal/engine/testdata/glyphs/iface.go`
- `internal/engine/testdata/glyphs/inits.go`
- `internal/engine/testdata/loomyard/render-dir.json`
- `internal/engine/testdata/loomyard/render-layout-file.json`
- `internal/engine/testdata/tiebreak/lib.go`
- `internal/engine/testdata/tiebreak/tool.go`
- `internal/engine/testdata/tree/Makefile`
- `internal/engine/testdata/tree/README.md`
- `internal/engine/testdata/tree/config.yaml`
- `internal/engine/testdata/tree/notes.rst`
- `internal/engine/testdata/tree/pkg/alpha.go`
- `internal/engine/testdata/tree/pkg/alpha_test.go`
- `internal/engine/testdata/tree/pkg/beta.go`
- `internal/engine/testdata/tree/pkg/doc.go`
- `internal/engine/testdata/tree/pkg/export_test.go`
- `internal/engine/testdata/tree/sub/deep/leaf.go`
- `internal/engine/testdata/tree/sub/doc.go`
- `internal/engine/testdata/units/root.go`
- `internal/engine/testdata/units/test data/pkg/spaced.go`
- `internal/engine/text.go`
- `internal/engine/text_test.go`
- `internal/engine/toc.go`
- `internal/engine/toc_integration_test.go`
- `internal/engine/toc_test.go`
- `internal/engine/treesitter/treesitter.go`
- `internal/engine/treesitter/treesitter_test.go`
- `internal/engine/walk.go`
- `internal/engine/walk_test.go`
