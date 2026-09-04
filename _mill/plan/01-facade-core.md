# Batch: facade-core

```yaml
task: "Facade + CLI, toc (T5a)"
batch: "facade-core"
number: 1
cards: 4
verify: go test ./quarry/...
depends-on: []
```

## Batch Scope

This batch creates the `quarry/` package and its non-rendering half: the package doc, the type
aliases and error sentinels that make the engine's answer shape nameable from outside the module,
and the `Open`/`TOC` entry point that mirrors `engine.Open`/`engine.(*Repo).TOC` one-to-one. It is
one batch because the aliases and the entry point are a single compilable unit — `TOC`'s signature
is written in terms of the aliases — and because nothing else in the plan can start until the
package exists.

The external interface batches 2, 3 and 4 consume: `quarry.Repo`, `quarry.Open`, `(*Repo).TOC`,
the aliases `DirAnswer`/`FileEntry`/`Symbol`/`Kind`/`TOCOptions`, the `Kind` constants, `DepthAll`,
and the sentinels `ErrTargetNotFound`/`ErrTargetOutsideRepo`/`ErrLanguageUnsupported`.

Batch-local decision: `quarry/` is split across three source files by role (`doc.go`, `quarry.go`,
`repo.go`) rather than one, matching how `internal/engine` splits `doc.go`/`answer.go`/`repo.go`.

## Cards

### Card 1: package doc for `quarry/`

- **Context:**
  - `internal/engine/doc.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `quarry/doc.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/doc.go` declaring `package quarry` with a package doc comment
  only — no other declaration. The comment states that `quarry` is the public Go facade over the
  extraction engine, that it is the primary surface named by `docs/rewrite-plan.md` §7 item 2, and
  that it exists because `internal/engine` cannot be imported from outside this module. It states
  that the package adds no behaviour of its own: the answer types are aliases for the engine's, the
  query methods delegate unchanged, and the only code the package owns is the two renderers
  (`RenderJSON`, `RenderErrorJSON`, `RenderText` — added in batch 2). It states that the package
  holds no cache, no parser pool, and no state beyond the repository root, per
  `docs/rewrite-plan.md` §10's phase-1 non-goals. Follow `internal/engine/doc.go`'s tone: what the
  package is for and what it deliberately does not do.
- **Commit:** `docs(quarry): add package doc for the public facade`

### Card 2: answer-type aliases, Kind constants, DepthAll, error sentinels

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/repo.go`
  - `internal/engine/errors.go`
  - `quarry/doc.go`
- **Edits:** none
- **Creates:**
  - `quarry/quarry.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/quarry.go` in `package quarry`, importing
  `github.com/Knatte18/quarry/internal/engine`. Declare, as Go **type aliases** (`=`, never a
  defined type):
  `type DirAnswer = engine.DirAnswer`, `type FileEntry = engine.FileEntry`,
  `type Symbol = engine.Symbol`, `type Kind = engine.Kind`, `type TOCOptions = engine.TOCOptions`.
  Declare the five `Kind` constants as aliases to the engine's values —
  `const KindFunction = engine.KindFunction` and likewise `KindMethod`, `KindType`, `KindConst`,
  `KindVar` — and `const DepthAll = engine.DepthAll`.
  Declare the three error sentinels as variables holding the engine's own values, so `errors.Is`
  keeps working across the boundary: `var ErrTargetNotFound = engine.ErrTargetNotFound`,
  `var ErrTargetOutsideRepo = engine.ErrTargetOutsideRepo`,
  `var ErrLanguageUnsupported = engine.ErrLanguageUnsupported`.
  Do not re-declare any struct. Do not add a field, a method, or a conversion function.
  Each declaration's doc comment says why the alias exists — that an alias to a type in an
  `internal/` package is nameable and usable by an external importer, because Go enforces the
  `internal` rule on import paths rather than on types reached through an alias, so a consumer can
  write `var a quarry.DirAnswer` without importing `internal/engine`. The sentinels' comment says
  they are the same values, not copies, which is what keeps `errors.Is` transitive across the
  facade.
- **Commit:** `feat(quarry): alias the engine's answer types into the public facade`

### Card 3: Repo, Open, and the TOC query

- **Context:**
  - `internal/engine/repo.go`
  - `internal/engine/toc.go`
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
  - `quarry/doc.go`
- **Edits:** none
- **Creates:**
  - `quarry/repo.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/repo.go` in `package quarry`. Declare
  `type Repo struct { engine *engine.Repo }` — an opened repository handle wrapping the engine's
  own. Declare `func Open(root string) (*Repo, error)`: it calls `engine.Open(root)` and on error
  returns `nil` and the error wrapped as `fmt.Errorf("quarry: open %q: %w", root, err)`; on success
  it returns `&Repo{engine: er}, nil`. Declare
  `func (r *Repo) TOC(target string, opts TOCOptions) (DirAnswer, error)`: it returns
  `r.engine.TOC(target, opts)` **unchanged** — the error is returned as-is, not wrapped, so
  `errors.Is(err, ErrTargetNotFound)` and `errors.Is(err, ErrTargetOutsideRepo)` succeed on the
  facade's own sentinels for a caller that never imports the engine.
  Document on `Open` that `root` must be an absolute path naming an existing directory and that
  `Open` performs no git discovery and no cwd resolution, because the engine performs neither and
  the facade adds no behaviour — root discovery is the CLI's job (`internal/cli`, batch 3).
  Document on `TOC` that `target` is repository-relative with `""` and `"."` both meaning the root,
  that the returned error is the engine's own so the facade's sentinels match it, and that `TOC`
  applies no filtering, no re-shaping and no defaulting beyond what `TOCOptions` already encodes.
  Document on `Repo` that it is safe for concurrent use by multiple goroutines, because it holds
  only the engine handle, which itself holds only the repository root string and reads the
  filesystem fresh on every query.
  Do not add any other method, any option struct, or any package-level `TOC` function.
- **Commit:** `feat(quarry): add Open and the TOC query over the engine`

### Card 4: facade core tests

- **Context:**
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `internal/engine/repo.go`
  - `internal/engine/answer.go`
  - `internal/engine/toc.go`
  - `internal/engine/repo_test.go`
  - `internal/engine/scratchtree_test.go`
- **Edits:** none
- **Creates:**
  - `quarry/repo_test.go`
  - `quarry/scratchtree_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/scratchtree_test.go` in `package quarry` declaring
  `func writeScratchTree(t *testing.T, name string, files map[string]string) string`, mirroring
  `internal/engine/scratchtree_test.go`'s helper of the same name: resolve the module root from
  `runtime.Caller(0)`, build the tree under `.scratch/quarry-tests/<name>/`, `os.RemoveAll` any
  stale tree first, create parent directories as needed, register a `t.Cleanup` that removes the
  tree, and return its absolute path. It writes regular files only. Its doc comment states that it
  never calls `t.TempDir()` because the system temp directory is banned for this repository's tests
  and `.scratch/` — gitignored at the repository root — is the sanctioned location, and that the
  helper is a deliberate per-package copy because Go test helpers are not importable across
  packages.

  Create `quarry/repo_test.go` in `package quarry`, table-driven where a table fits, building every
  filesystem fixture with `writeScratchTree` — do not read `internal/engine/testdata/`, which
  belongs to the engine's own tests, and do not call `t.TempDir()`.
  Cover: `Open` rejecting a relative root, a non-existent root, and a root that names a file rather
  than a directory, each returning a non-nil error whose message begins `quarry: open`; `Open`
  succeeding on an absolute existing directory and returning a non-nil `*Repo`.
  For the `Open` cases that need a path rather than a tree, use paths under the tree
  `writeScratchTree` returns.
  Cover `TOC` on a small synthesised tree: a directory target answering with the directory's own
  `Dir` and its files; a file target answering with one entry in `Files`.
  Cover the sentinel transitivity that is this batch's whole point: `TOC` on a missing target
  returns an error satisfying `errors.Is(err, ErrTargetNotFound)` — asserted against the **facade's**
  sentinel, not the engine's — and `TOC` on an absolute target returns one satisfying
  `errors.Is(err, ErrTargetOutsideRepo)`.
  Cover that the aliases are aliases and not defined types, with a compile-time assertion that a
  value of the engine's type is assignable to the facade's without conversion, e.g.
  `var _ DirAnswer = engine.DirAnswer{}` and `var _ engine.TOCOptions = TOCOptions{}` at package
  scope. This is what fails loudly if a later edit turns an alias into a defined type.
- **Commit:** `test(quarry): cover Open, TOC delegation, and alias identity`

## Batch Tests

`verify: go test ./quarry/...` runs the new `quarry/repo_test.go` over the `writeScratchTree`
helper `quarry/scratchtree_test.go` adds. Scoped to the
`quarry` package because that is the only package this batch touches; the module-wide
`go build ./...` at the batch boundary catches any cross-package compile break. The engine's own
suite is not re-run — no batch in this plan modifies `internal/engine`.
