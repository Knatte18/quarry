# Batch: treesitter-backend

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "treesitter-backend"
number: 1
cards: 7
verify: go test ./internal/quarryengine ./internal/quarryengine/treesitter ./internal/quarryengine/registry
depends-on: []
```

## Batch Scope

This batch adds the parsing backend and the file-extension language map — everything the toc
orchestration layer needs before it can exist, and nothing that depends on it. It brings the
tree-sitter modules into `go.mod`, creates `internal/quarryengine/treesitter` as the single place in
the tree that constructs a `*ts.Parser` or a `*ts.Tree`, adds the extension-to-language map to
`internal/quarryengine/registry`, and adds a loud build-time guard so a `CGO_ENABLED=0` build fails
with a sentence a human can act on rather than a linker dump.

The external interface batch 2 consumes is exactly two things: `treesitter.WithTree`, the
parse-and-release seam, and `registry.LanguageForExtension`.

Batch-local decision that differs from the overview: the CGO guard's two build-tagged files live in
the engine **root** package, not in `treesitter`. Under `CGO_ENABLED=0` the `treesitter` package
cannot compile at all — it transitively imports cgo — so a guard placed there is unreachable exactly
when it is needed. The root package imports no cgo, so its guard test still builds and still runs,
and it is the first `internal/quarryengine/...` package `go test ./...` reaches.

## Cards

### Card 1: tree-sitter module dependencies

- **Context:**
  - `.scratch/cgobench/go.mod`
- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add the six tree-sitter modules at the exact versions the Shared Decision
  "tree-sitter dependency set and pinned versions" pins — `github.com/tree-sitter/go-tree-sitter`
  v0.25.0, `github.com/tree-sitter/tree-sitter-go` v0.25.0,
  `github.com/tree-sitter/tree-sitter-python` v0.25.0,
  `github.com/tree-sitter/tree-sitter-c-sharp` v0.23.5,
  `github.com/tree-sitter/tree-sitter-typescript` v0.23.2, and
  `github.com/tree-sitter/tree-sitter-rust` v0.24.2 — plus the indirect
  `github.com/mattn/go-pointer` v0.0.1. Add them by editing `go.mod`'s require blocks to those exact
  versions and then running `go mod tidy` to populate `go.sum`; never run `go get <module>@latest`,
  which would resolve different versions than the node shapes in this plan were confirmed against.
  Every one of these modules is already in the local module cache, so this must succeed with no
  network access. Confirm afterwards that `go build ./...` still succeeds.
  Leave the `go 1.26` directive and the three pre-existing requires unchanged.
- **Commit:** `build(deps): pin tree-sitter runtime and five grammar modules`

### Card 2: loud CGO_ENABLED=0 build guard

- **Context:**
  - `internal/quarryengine/log.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/cgoguard.go`
  - `internal/quarryengine/cgoguard_nocgo.go`
  - `internal/quarryengine/cgoguard_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add a build-tagged pair declaring one unexported package-level constant,
  `cgoEnabled`. `cgoguard.go` carries `//go:build cgo` and sets it to `true`; `cgoguard_nocgo.go`
  carries `//go:build !cgo` and sets it to `false`. Both files declare `package quarryengine` and
  neither imports anything.
  Add `TestCGOEnabled_BuildLinksC` in `cgoguard_test.go`, which calls `t.Fatal` with a message naming
  `CGO_ENABLED=1` and the need for a C toolchain when `cgoEnabled` is false. The test must fail, never
  skip — a skipped guard is exactly the green-run-against-an-unbuildable-binary outcome this exists to
  prevent.
  Both new production files must sit in the engine root package so they still compile when
  `CGO_ENABLED=0` makes every tree-sitter-importing package unbuildable. State that reasoning in
  `cgoguard.go`'s own file header comment, so the placement is not later "tidied" into the
  `treesitter` package where it would be unreachable.
- **Commit:** `test(quarryengine): fail loudly when built without cgo`

### Card 3: the treesitter parsing backend

- **Context:**
  - `internal/quarryengine/log.go`
  - `internal/quarryengine/errors.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/treesitter/treesitter.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package treesitter` with a package doc comment stating that this package
  is the parsing backend — grammar loading, parser construction, and nothing else — and that it is the
  only place in the tree that constructs a `*ts.Parser` or a `*ts.Tree`.
  Import the runtime as `ts "github.com/tree-sitter/go-tree-sitter"` and each grammar's
  `bindings/go` package, using the aliases `tsgo`, `tspython`, `tscsharp`, `tstypescript`, `tsrust`.
  Export:
  - `Supported(lang string) bool` — reports whether a grammar is wired for the canonical language
    name.
  - `Languages() []string` — the sorted canonical names with a wired grammar: `csharp`, `go`,
    `python`, `rust`, `typescript`.
  - `WithTree(lang string, src []byte, fn func(root *ts.Node, partial bool) error) error` — the whole
    parse lifecycle. It resolves the grammar, constructs a parser, `defer`s `parser.Close()`, calls
    `parser.SetLanguage`, calls `parser.Parse(src, nil)`, `defer`s `tree.Close()`, and calls `fn` with
    `tree.RootNode()` and `root.HasError()`. `fn`'s error is returned unchanged. The `defer`s must be
    registered immediately after each successful construction so both allocations are released on
    every route — the success route, the route where `fn` returns an error, and the `partial` route
    alike. `fn` must never retain the `*ts.Node` it is handed beyond its own return, and the doc
    comment must say so, since the node is invalidated by `tree.Close()`.
  - An unknown `lang` returns a **plain, unwrapped** `fmt.Errorf` error naming the language and the
    wired set — deliberately not `quarryengine.ErrNoLanguage`. That sentinel's own doc comment in
    `errors.go`, and the engine's package documentation, both define it narrowly as "no
    registry entry's markers matched under the target directory", which is a detection outcome, not a
    grammar-wiring outcome; reusing it here would make that documented meaning false. Say so in a
    comment at the return, so the sentinel is not "helpfully" reintroduced later.
    `ErrLanguageUnsupported` is not used either: it does not exist yet at this batch (batch 2 card 8
    adds it), and it means "no toc strategy is registered", which is a different layer from "no
    grammar is wired". A nil `*Tree` back from
    `Parse` returns a non-nil error rather than a nil-pointer dereference.
  Use the grammar constructors named in the Shared Decision "the confirmed go-tree-sitter v0.25.0 API
  surface" — in particular `tstypescript.LanguageTypescript()` for TypeScript, not a bare
  `Language()`, which that module does not export.
  Add an unexported package-level `var onRelease func()` test seam, nil in production, invoked from
  `WithTree` after both `Close` calls have run. Document it as existing solely so the release
  behaviour is observable from a test; production code must never assign it.
  This package imports the engine root only. It must not import `registry`, `internal/output`, cobra,
  or `internal/cli`.
- **Commit:** `feat(treesitter): add the tree-sitter parsing backend`

### Card 4: treesitter backend tests

- **Context:**
  - `internal/quarryengine/treesitter/treesitter.go`
  - `internal/quarryengine/registry/registry_test.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/treesitter/treesitter_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** table-driven tests over all five languages, each with a trivial valid inline
  source fixture, asserting that `WithTree` resolves the grammar, that the root node is non-nil, that
  `partial` is false, and that `root.Kind()` is the language's expected root kind. This table is the
  canary for a grammar-module version bump that renames or removes an API.
  Add `TestWithTree_ReleasesParserAndTreeOnEveryRoute`: using the `onRelease` seam, assert the counter
  advances once on the success route, once on the route where the callback returns an error, and once
  on a `partial` route driven by a deliberately broken fixture — and assert `WithTree` still returns
  the callback's error unchanged on the middle route.
  Add a test that an unknown language name returns a non-nil error whose message names the language,
  and assert `errors.Is(err, quarryengine.ErrNoLanguage)` is **false** — the negative assertion is the
  point, since it is what pins the sentinel to its documented detection-only meaning.
  Restore `onRelease` to nil in a `t.Cleanup` in every test that assigns it, and do not mark those
  tests `t.Parallel()` — they mutate package state.
- **Commit:** `test(treesitter): cover grammar loading and C-memory release`

### Card 5: extension-to-language map

- **Context:**
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/registry/detect.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/registry/extension.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add a file-extension language map to `package registry`, alongside — never
  replacing — the marker-based `DetectLanguage`. The file header comment must state why the two exist
  side by side: `DetectLanguage` matches marker files against a *directory* with a fixed precedence
  order, which resolves a `.ts` file inside a Go module to `go` — correct for the LSP verbs, wrong for
  a path-scoped one.
  Export:
  - `LanguageForExtension(ext string) (string, bool)` — maps `.go` to `go`, `.py` to `python`, `.cs`
    to `csharp`, `.ts` and `.tsx` to `typescript`, and `.rs` to `rust`. The lookup lowercases `ext`
    and tolerates a missing leading dot, so both `.GO` and `go` resolve. An unknown extension returns
    `("", false)`.
  - `ExtensionsForLanguage(lang string) []string` — the sorted extensions mapping to `lang`, or nil
    for an unknown name.
  - `ExtensionLanguages() []string` — the sorted canonical names the map defines: `csharp`, `go`,
    `python`, `rust`, `typescript`.
  Back all three with one unexported package-level map so the extension set has exactly one
  definition. This file imports the standard library only.
- **Commit:** `feat(registry): add the file-extension language map`

### Card 6: extension map tests

- **Context:**
  - `internal/quarryengine/registry/extension.go`
  - `internal/quarryengine/registry/detect_test.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/registry/extension_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** table test `LanguageForExtension` over all six extensions, an unknown extension, a
  dotless input, and an uppercase input. Assert `ExtensionLanguages` returns exactly the five
  canonical names in sorted order, and that `ExtensionsForLanguage("typescript")` returns both `.ts`
  and `.tsx` sorted while an unknown language returns nil.
  Add one assertion pinning the reason this map exists separately: `LanguageForExtension(".ts")`
  resolves to `typescript` regardless of any directory context, which is the case marker-based
  detection cannot serve.
- **Commit:** `test(registry): cover the file-extension language map`

### Card 7: layering rows for the treesitter package

- **Context:**
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:**
  - `internal/quarryengine/layering_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `treesitterPkg = rootPkg + "/treesitter"` to the import-path constant block,
  and add two `layeringTable` rows for `pkgDir: "treesitter"` — one with `isTestRow: false` and one
  with `isTestRow: true` — each allowing `pathSet(rootPkg)` only, matching the shape `lsp` and
  `registry` already use.
  Without these rows `allowedFor` returns `ok = false` for every file in the new package and
  `TestLayeringInvariant_ImportDirections` fails, so this card lands in the same batch as the package
  itself.
  Update the comment above the constant block, which currently says "The six
  internal/quarryengine/... import paths this guard reasons about", to name the correct count now
  that a seventh path is declared. Leave `minPackageDirs` at 6 in this batch — it is raised to 8 in
  batch 2, once the second new package directory also exists.
- **Commit:** `test(quarryengine): add treesitter rows to the layering guard`

## Batch Tests

`verify: go test ./internal/quarryengine ./internal/quarryengine/treesitter ./internal/quarryengine/registry`
covers exactly the three packages this batch touches: the engine root (the CGO guard test plus the
layering and seam guards that walk the new package directory), the new `treesitter` package, and
`registry` with its new extension map. Nothing outside those three packages changes, so a wider run
would only re-verify untouched code.

New test files: `internal/quarryengine/cgoguard_test.go`,
`internal/quarryengine/treesitter/treesitter_test.go`,
`internal/quarryengine/registry/extension_test.go`. Modified:
`internal/quarryengine/layering_test.go`.

The first run of this batch's verify command compiles the tree-sitter C sources and is expected to
take noticeably longer than a warm run; subsequent runs hit the build cache.
