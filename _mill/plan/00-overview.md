# Plan: Facade + CLI, toc (T5a)

```yaml
task: "Facade + CLI, toc (T5a)"
slug: "facade-cli-toc"
approved: false
started: "20260904-061252"
parent: "main"
root: ""
verify: go build ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: facade-core
    file: 01-facade-core.md
    depends-on: []
    verify: go test ./quarry/...
  - number: 2
    name: facade-renderers
    file: 02-facade-renderers.md
    depends-on: [1]
    verify: go test ./quarry/...
  - number: 3
    name: cli-parsing
    file: 03-cli-parsing.md
    depends-on: [1]
    verify: go test ./internal/cli/...
  - number: 4
    name: cli-pipeline
    file: 04-cli-pipeline.md
    depends-on: [2, 3]
    verify: go test ./internal/cli/... ./cmd/quarry/...
  - number: 5
    name: goldens-and-after
    file: 05-goldens-and-after.md
    depends-on: [4]
    verify: go test ./internal/cli/...
```

## Shared Decisions

### Decision: no-file-under-internal-engine-is-modified

- **Decision:** no batch may edit, create, or delete any file under `internal/engine/`, `glyph/`, or
  `internal/cgoguard/`. If the facade, the CLI, or the golden run exposes an engine defect, the
  implementer records it in its report and works around it on the T5a side — for the `after/`
  fixtures, by capturing what the engine actually emits and noting it in `after/INDEX.md`.
- **Rationale:** `discussion.md`'s `t5a-does-not-change-the-engine`. T3's own goldens
  (`internal/engine/testdata/loomyard/`) pin the engine byte for byte, and T4 is in flight against
  the same package in a parallel worktree; an engine edit here is a merge conflict at best and a
  silent contradiction of T3's goldens at worst.
- **Applies to:** all batches

### Decision: engine-unexported-helpers-are-not-reachable

- **Decision:** three things the discussion names by their engine spelling are unexported and cannot
  be called from `quarry/` or `internal/cli`, and each is re-implemented locally rather than
  exported from the engine:
  1. `walk.go`'s `joinRel` — batch 2 declares its own unexported `joinRel` in `quarry/text.go`.
  2. `loomyard_test.go`'s `loomyardRepo`, `loomyardPin` and the `-update` flag — batch 5 declares its
     own copy in `internal/cli/loomyard_test.go`. Go test helpers are not importable across packages.
  3. `repo.go`'s `resolveTarget` — the CLI does its own path arithmetic (`internal/cli/target.go`)
     and its own `os.Lstat`, per `target-kind-and-the-cli-stat`.
- **Rationale:** the alternative in each case is exporting an engine identifier, which
  `no-file-under-internal-engine-is-modified` forbids. Naming this once here stops five batches
  each rediscovering it and reaching for an engine edit.
- **Applies to:** all batches

### Decision: alias-types-carry-no-methods

- **Decision:** `quarry.DirAnswer` and friends are Go type aliases for types declared in
  `internal/engine`. `quarry/` therefore **cannot** declare methods on them — no `a.JSON()`, no
  `a.Text()`. Every renderer is a package-level function taking the value as its first argument.
- **Rationale:** Go forbids declaring a method on a type defined in another package, and an alias
  does not change where the type is defined. `discussion.md`'s `renderers-live-in-the-facade` calls
  this out as a hard constraint the plan must not design around.
- **Applies to:** all batches

### Decision: json-encoder-spacing-is-the-byte-contract

- **Decision:** wherever `discussion.md` writes a JSON payload inline as prose — notably
  `{"ok": false, "error": …}` — the spaces after the colons are prose, not a byte specification. The
  emitted bytes are whatever `encoding/json` produces for the chosen encoder settings:
  - `RenderJSON` uses `SetIndent("", "  ")`, so it is pretty-printed with two-space indentation.
  - `RenderErrorJSON` uses no indent, so it is compact: `{"ok":false,"error":"<msg>"}` followed by a
    single `\n`, with **no** space after either colon.
  Every test and every golden asserts the encoder's actual bytes, never the prose spelling.
- **Rationale:** the failure envelope is asserted byte for byte by
  `discussion.md`'s Testing block ("stdout is exactly … plus a newline"), so its spacing has to be
  decided somewhere rather than left for the implementer to guess from prose. Choosing the
  encoder's own output means no hand-built string can drift from it.
- **Applies to:** batch 2, batch 4

### Decision: comment-discipline-matches-answer-go

- **Decision:** every new exported identifier in `quarry/` and every new declaration in
  `internal/cli` carries a doc comment saying **why** it is shaped as it is, not what it is —
  the standard `internal/engine/answer.go` sets. Each new file opens with a file-level comment
  naming what the file holds, as every existing file in this repository does.
- **Rationale:** `discussion.md`'s Technical context names `answer.go`'s comment discipline as the
  bar new public types are held to. Stating it once here keeps five batches from drifting apart.
- **Applies to:** all batches

### Decision: no-new-module-dependencies

- **Decision:** `go.mod` and `go.sum` are not edited by any batch. The CLI's flag parsing is
  hand-rolled over `os.Args`; no CLI framework is added, and `flag` from the standard library is not
  used for the verb's own flags either (it cannot express `--depth all` alongside `--depth 3`, nor
  `--no-symbols` as a distinct third state).
- **Rationale:** `discussion.md`'s Constraints ("No new module dependencies") and `cli-shape`'s
  rejection of cobra.
- **Applies to:** all batches

### Decision: done-gate-lint-half-already-fails-on-pre-existing-debt

- **Decision:** recorded as a finding, **not** acted on by any batch. `pipeline.done_gate` in the hub
  `mill-config.yaml` is `go test ./... && golangci-lint run`. On the current worktree tip, before any
  T5a change, `golangci-lint run` exits 1 with three `errcheck` findings, all in
  `bench/loomyard-eval/ladder/` and all unrelated to T5a:
  `internal/ladder/e2e_test.go:560`, `internal/ladder/run.go:92`, `internal/ladder/worktree_test.go:42`.
  No batch touches `bench/`, and no batch edits `mill-config.yaml`.
- **Rationale:** the debt predates this task and lies wholly outside its scope; fixing it here would
  widen T5a into an unrelated package, and changing `done_gate` would make an operator configuration
  decision on the operator's behalf. The consequence is stated plainly so it is not discovered as a
  surprise: mill-go's done gate will fail at the end of this task on those three findings, and
  clearing them (or narrowing `done_gate`) is an operator call outside this plan.
  The task's own constraint — `go build ./... && go test ./...` green — is unaffected and is what
  every batch verifies against.
- **Applies to:** all batches

### Decision: a-symbols-key-is-never-guaranteed-by---symbols

- **Decision:** no card, test, fixture, or renderer may assume that passing `--symbols` (or
  `TOCOptions.Symbols` set to true) makes a `symbols` key appear on a file entry. Both views emit
  nothing at all for a file whose `Symbols` is nil, and that is a correct answer, never a defect to
  chase. Concretely, three engine behaviours produce a nil `Symbols` despite the flag:
  1. **A file directly under the repository root.** `internal/engine/walk.go`'s `unitFor` returns
     the empty string when `dirRel` is `"."`, and `unitSpellable("")` is false — its doc names "the
     repository root's empty unit" as a rejection — so a root-level Go file never carries symbols.
  2. **A file whose unit the Go alphabet cannot spell**, e.g. a path segment holding a space, a `.`
     or `..` segment, a backslash, or an ASCII control rune.
  3. **A file with no language**, which never gets symbols whatever the flag says.
  Any test asserting that `--symbols` populated something must therefore query a **named
  subdirectory** of its fixture root whose unit is spellable, never the fixture root itself.
- **Rationale:** `_mill/discussion.md`'s Technical context lists this as gotcha 1 of the seven "all
  of which the plan must account for". A fixture that puts its Go file at the fixture root and then
  asserts a `symbols` key would fail for a reason that looks like a facade or CLI bug and is
  neither — and "fixing" it would mean editing the engine, which is forbidden.
- **Applies to:** batch 1, batch 2, batch 4

### Decision: fixture-trees-live-under-scratch-never-t-tempdir

- **Decision:** every on-disk fixture tree in this task is built under `.scratch/<pkg>-tests/<name>/`
  at the repository root, never with `t.TempDir()`. Two packages need such trees, and each declares
  its own helper because Go test helpers are not importable across packages:
  `quarry/scratchtree_test.go` (batch 1 card 4) and `internal/cli/scratchtree_test.go` (batch 3
  card 14). Both mirror `internal/engine/scratchtree_test.go`'s `writeScratchTree` — resolve the
  module root from `runtime.Caller(0)`, `os.RemoveAll` any stale tree, write each entry, and
  register a `t.Cleanup` that removes it — and both write regular files only, leaving symlinks and
  other special entries for the calling test to create on the returned path.
  The `runtime.Caller(0)` walk is **not** copied verbatim: the number of `filepath.Dir` steps is a
  function of the copy's own depth below the module root. `internal/engine/` and `internal/cli/` are
  two directories down and need three `filepath.Dir` calls (file to package dir to `internal` to
  module root); `quarry/` is one directory down and needs two. Copying the engine's three steps into
  `quarry/` would resolve the module root to the repository's **parent** and silently write
  `.scratch/quarry-tests/` outside the repository, where the tests still pass and this decision is
  violated invisibly.
- **Rationale:** `internal/engine/scratchtree_test.go`'s helper doc states the convention verbatim
  ("It never calls `t.TempDir()`: the system temp directory is banned … `.scratch/` — gitignored at
  the repository root — is the sanctioned location instead"), and every engine test follows it. The
  ban is not merely T3-scoped: writing tests, fixtures, or any ephemeral scratch into a system
  temporary directory is prohibited outright for this repository. `.scratch/` is already gitignored
  by the repo-root `.gitignore`, so nothing the helper writes can be committed by accident.
  `bench/loomyard-eval/ladder` does use `t.TempDir()`, but it is the vendored benchmark harness and
  is not the convention new first-party packages follow.
- **Applies to:** all batches

### Decision: verify-scope-is-per-batch

- **Decision:** each batch's `verify:` runs only the packages that batch touches
  (`go test ./quarry/...` or `go test ./internal/cli/...`), never `go test ./...`. The module-wide
  `verify: go build ./...` in this file's frontmatter runs at each batch boundary and is what catches
  a cross-package break. Go test commands take no `PYTHONPATH=` prefix — that rule is Python-only.
- **Rationale:** `go test ./...` compiles and runs the whole module including `bench/`, which is slow
  and unrelated; `go build ./...` is the cheap whole-module compile the template asks for.
- **Applies to:** all batches

### Decision: cgo-stays-required-and-unguarded

- **Decision:** no batch adds a `//go:build cgo` tag, or any build tag, to `quarry/`,
  `cmd/quarry/`, or `internal/cli`. `CGO_ENABLED=0 go build ./...` is expected to fail through
  `internal/cgoguard`'s `!cgo` file, and that failure is the guard working.
- **Rationale:** `discussion.md`'s Constraints say so explicitly; a tag added to make a
  `CGO_ENABLED=0` build pass would hide the guard T1 exists to provide.
- **Applies to:** all batches

## All Files Touched

- `cmd/quarry/main.go`
- `docs/research/output-formats/after/INDEX.md`
- `docs/research/output-formats/after/toc-dir-text.txt`
- `docs/research/output-formats/after/toc-dir.txt`
- `docs/research/output-formats/after/toc-file-text.txt`
- `docs/research/output-formats/after/toc-file.txt`
- `internal/cli/after_test.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/doc.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/loomyard_test.go`
- `internal/cli/plan4_test.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `internal/cli/scratchtree_test.go`
- `internal/cli/target.go`
- `internal/cli/target_test.go`
- `internal/cli/usage.go`
- `quarry/doc.go`
- `quarry/quarry.go`
- `quarry/render.go`
- `quarry/render_test.go`
- `quarry/repo.go`
- `quarry/repo_test.go`
- `quarry/scratchtree_test.go`
- `quarry/text.go`
- `quarry/text_test.go`
