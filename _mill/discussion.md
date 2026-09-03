# Discussion: Delete V1, keep the tree-sitter extractors (T0)

```yaml
task: Delete V1, keep the tree-sitter extractors (T0)
slug: delete-v1
status: discussing
parent: main
```

## Problem

Quarry is being rewritten around one identifier — the glyph — and three queries (`map`, `resolve`,
`members`), with tree-sitter as the only extraction backend in phase 1. The specification is
`docs/rewrite-plan.md`; the identifier contract is `docs/glyph.md`. The plan (§2, and the T0 row of
§12) states that the first commit on `main` after the plan is written is the deletion, so no
half-V1 code survives into the new work.

This task is that deletion. Everything V1 — the language-server layer, the seven-verb CLI, the MCP
server, the `quarry/` facade, their fixtures, the V1 benchmark harness and its skill, the V1
documents — is removed. What stays is the tree-sitter extraction layer, which the measurements in
`HANDOFF.md` identified as the part that works, plus the benchmark record (results, ladder yaml,
task prompts) and the two rewrite documents. V1 is frozen and reachable on branch `v1-final`;
nothing is merged from it, and nothing deleted here is lost.

Why now: every task after T0 — T1 (`glyph/`), T2 (harness rewrite), T3 (engine core) — builds on
this deletion. Until it lands, each of those would be written against a tree that still contains the
architecture they replace.

## Scope

**In:**

- Delete the seven-verb surface: `cmd/quarry`, `cmd/quarry-mcp`, `internal/cli`,
  `internal/mcpserver`, `internal/output`, the `quarry/` facade, `.mcp.json`.
- Delete the LSP layer: `internal/lock`, `internal/proc`,
  `internal/quarryengine/{daemon,lsp,query,impact}`, and all of
  `internal/quarryengine/registry` except the extension→language table.
- Delete `internal/quarryengine/layering_test.go` and
  `internal/quarryengine/seam_enforcement_test.go`.
- Move the extension→language table (`registry/extension.go` + `extension_test.go`) into
  `internal/quarryengine/toc`; `registry/` then disappears entirely.
- Trim the root package `internal/quarryengine` to `cgoguard.go`, `cgoguard_nocgo.go`, a rewritten
  `doc.go`, and an `errors.go` holding only `ErrNoLanguage` and `ErrLanguageUnsupported`.
- Drop Rust and TypeScript everywhere in kept code: the two grammars in
  `treesitter.go`'s `grammars` map, the `.rs`/`.ts`/`.tsx` rows in the extension table, the
  `typescript` branch in `toc/classify.go`'s `TestFile`, every rust/typescript test case, and the
  doc comments naming five languages.
- Delete `testdata/` in full (`impactfixture`, `clockfixture`, `buildtagfixture`, including their
  three nested `go.mod` files).
- Delete the V1 harness: `bench/loomyard-eval/ladder/{cmd,internal,tools}`,
  `bench/loomyard-eval/ladder/run*.sh`, `launch-session.sh`,
  `bench/loomyard-eval/ladder/README.md`, `bench/loomyard-eval/README.md`,
  `.claude/skills/ladder-run/SKILL.md`.
- Delete the V1 documents: `docs/mcp-setup.md`, `docs/when-to-use-quarry.md`,
  `docs/servers.yaml.example`.
- Replace `README.md` with a short stub (text fixed below, under Decisions →
  `readme-stub`).
- Trim `.gitignore` to kept paths.
- `go mod tidy` so no Rust/TypeScript grammar and no LSP/JSON-RPC module remains in `go.mod`.

**Out:**

- No new features, no new packages, no re-implementation. If a kept package fails to build because
  it imported a deleted one, the import and the dependent code path are removed — never rewritten.
- `glyph/`, the new CLI, the new MCP server, and the new harness are T1, T5, T6 and T2. Nothing of
  them is created here.
- `internal/quarryengine/toc` and `internal/quarryengine/treesitter` are not restructured beyond
  the Rust/TypeScript removal and receiving the extension table. Their remaining tests stay green
  unmodified except where they name a deleted language or package.
- `docs/research/**`, `docs/toc-docstring-association.md`, `docs/glyph.md`,
  `docs/rewrite-plan.md`, `HANDOFF.md`, `bench/loomyard-eval/tasks/**`,
  `bench/loomyard-eval/results/**`, `bench/loomyard-eval/ladder/results/**`,
  `bench/loomyard-eval/ladder/ladder*.yaml` and
  `bench/loomyard-eval/scripts/gen_compact_toc.py` are kept byte for byte. They are the record and
  the fasit; they are not edited even where they describe deleted verbs or carry machine paths.
- The `# === mill-managed ===` block at the bottom of `.gitignore` is not touched.
- Nothing is merged or cherry-picked from `v1-final`.

## Decisions

### docs-deleted-not-superseded

- Decision: the V1 documents are deleted outright — `docs/mcp-setup.md`,
  `docs/when-to-use-quarry.md`, `docs/servers.yaml.example`, `.mcp.json` — and `README.md` becomes
  a stub. Nothing is marked "superseded".
- Rationale: the wiki task body (older) said these could be trimmed or marked superseded; plan §2
  and §12's T0 row (newer, commit `51510f1`) say they are deleted, because they describe the seven
  verbs, `cmd/quarry-mcp` and the daemon, all of which live on `v1-final`. What is archived is
  deleted. T5 and T6 write the new documents and the new `.mcp.json` when the surface they describe
  exists.
- Rejected: keeping them with superseded markers (leaves prose describing code that no longer
  exists in the tree); a split where only `.mcp.json` and `servers.yaml.example` go.

### readme-stub

- Decision: `README.md` is replaced in full by exactly this text:

  ```markdown
  # quarry

  Quarry is being rewritten around one identifier — the glyph — and three
  queries: `map`, `resolve`, `members`. Extraction is tree-sitter only, for
  Go, Python and C#.

  The rewrite is specified in two documents:

  - [`docs/rewrite-plan.md`](docs/rewrite-plan.md) — what is deleted, what is
    built, in what order.
  - [`docs/glyph.md`](docs/glyph.md) — the identifier contract.

  Version 1 — the seven-verb CLI, the MCP server and the language-server
  layer — is frozen on branch `v1-final`. Nothing is merged from it.

  ## Building

  The tree-sitter backend is a cgo binding, so `go build`, `go run` and
  `go test` require `CGO_ENABLED=1` and a C toolchain. There is no command
  to build yet; `go test ./...` runs the extractors' tests.
  ```

- Rationale: plan §2 asks for a stub naming the rewrite, the two documents, and `v1-final`. The
  cgo requirement is a real, current build-time fact a reader hits immediately, so it earns its
  place. The stub names Go, Python and C# only, and describes nothing that does not exist yet.
- Rejected: dropping the Building section (the cgo requirement is the one thing a fresh clone needs
  to know); a single paragraph with just the two links.

### root-package-kept-trimmed

- Decision: `internal/quarryengine` survives as a package. It keeps `cgoguard.go`,
  `cgoguard_nocgo.go`, a rewritten short `doc.go`, and an `errors.go` reduced to `ErrNoLanguage`
  and `ErrLanguageUnsupported`. `log.go`, `position.go`, and the seven LSP-specific error types
  (`ErrServerNotFound`, `ErrSymbolNotFound`, `ErrAmbiguousSymbol`, `ErrResolverUnsupported`,
  `ErrServerTimeout`, `ErrServerSpawnTimeout`, `ErrBuildTagsUnsupported`, with their sentinels) are
  deleted.
- Rationale: `toc` imports the root package for exactly those two sentinels, and `cgoguard.go`'s
  own comment explains why the cgo guard cannot live in `treesitter` — under `CGO_ENABLED=0` the
  `treesitter` package cannot compile at all, so a guard placed there is unreachable exactly when
  it is needed. Keeping the package trimmed is pure deletion; no code moves.
- Rejected: collapsing the package (moving the two sentinels into `toc` and the cgo guard into a
  new package) — that is re-architecting, which T0 is not; keeping the package verbatim as dead
  code for T3 to reuse.

### extension-table-into-toc

- Decision: `internal/quarryengine/registry/extension.go` and `extension_test.go` move to
  `internal/quarryengine/toc` (package `toc`). `registry/` is deleted whole. `toc.go` drops the
  `registry` import and calls `LanguageForExtension`, `ExtensionsForLanguage` and
  `ExtensionLanguages` unqualified.
- Rationale: `toc` is the table's only remaining caller once the LSP layer is gone. Plan §2
  explicitly permits moving it into `toc` or `treesitter` when `registry` otherwise disappears.
- Rejected: `treesitter` (that package knows nothing about file extensions today, only about
  grammar names); keeping a one-file `registry` package holding only the table.

### rust-typescript-removed-everywhere

- Decision: Rust and TypeScript are not supported. Remove `rust` and `typescript` from
  `treesitter.go`'s `grammars` map and the two grammar imports; remove the `.rs`, `.ts` and `.tsx`
  rows from the extension table; remove the `typescript` branch from `toc/classify.go`'s
  `TestFile`; remove every rust/typescript test case; reword the doc comments in kept code that
  name five languages to name Go, Python and C#.
- Rationale: plan §2 removes the two grammars from `go.mod`; three focus languages. `toc_test.go`
  cross-checks the extension table against `treesitter.Supported`, so the table and the grammar map
  must move together. Leaving dead names in comments and tests describes a capability that does not
  exist.
- Rejected: keeping the table rows and relaxing the cross-check to a subset assertion; keeping the
  `typescript` `TestFile` branch dormant (it is filename-only and would still pass, but it claims
  support that is gone); keeping the grammars.

### unreachable-no-strategy-branch-kept

- Decision: keep `buildDirEntry`'s no-strategy branch (`toc.go:199-202`, `entry.Error =
  quarryengine.ErrLanguageUnsupported.Error()`) and its `TOCFile` counterpart, even though neither
  can be reached once the extension table is `{go, python, csharp}` and all three register a
  `Strategy`. Do not add a comment marking them unreachable either.
- Rationale: these are three lines of defensive code guarding an invariant that holds only by
  coincidence at this commit. `toc/strategy.go`'s own `Implemented()` doc says the point of the
  registry is to let callers "distinguish 'designed but not implemented' from 'unknown extension'",
  and T3 and T8 put languages back. Deleting a live guard from a kept package is re-architecting
  it, which T0 is not; the "remove the dependent code path" rule exists for code that no longer
  compiles, not for code that no longer happens to fire.
- Rejected: deleting both branches as dead code; annotating them as currently unreachable (a
  comment that goes stale the moment T8 lands).

### compact-test-row-repointed

- Decision: in `compact_test.go`, repoint the `bad.rs` / `"rust"` fixture row rather than deleting
  it, so `TestCompactDir` keeps a `DirEntry` with `Error` set and the `5 files` count in its
  expected header stays correct.
- Rationale: `DirEntry.Error` survives this task — `buildDirEntry` still sets it from the
  read-failure and invalid-UTF-8 branches (`toc.go:206-214`) — so `CompactDir`'s error-line
  rendering still has a live producer and must keep its coverage. Deleting the row would drop that
  coverage *and* force the header count and the `want` list to change, a materially larger diff
  than repointing.
- Rejected: deleting the row (loses live coverage, larger diff); leaving the choice open for
  mill-plan to make.

### guard-tests-deleted

- Decision: `internal/quarryengine/layering_test.go` and
  `internal/quarryengine/seam_enforcement_test.go` are both deleted, in the *first* commit
  (alongside the seven-verb surface).
- Rationale: `layering_test.go` pins a nine-package DAG of which six packages disappear;
  `seam_enforcement_test.go` bans imports of `internal/output`, `cobra` and `internal/cli`, all
  three deleted, making the ban vacuous. It also `t.Fatalf`s at line 91 when the `quarry/`
  directory is missing, so it must go in the same commit that deletes `quarry/` or that commit is
  not green. A guard that cannot fail is worse than no guard; T5 re-adds a seam test when the CLI
  returns.
- Rejected: trimming both to the surviving packages; keeping a trimmed `layering_test.go` (with
  three packages left it polices almost nothing).

### testdata-deleted-whole

- Decision: delete the entire root `testdata/` directory.
- Rationale: `impactfixture`, `clockfixture` and `buildtagfixture` are all of it, all three are on
  plan §2's delete list, and neither `toc` nor `treesitter` references any path under `testdata/`.
  Their three nested `go.mod` files go with them.
- Rejected: keeping the directory with a placeholder file.

### bench-record-kept-harness-deleted

- Decision: delete the harness code and both benchmark protocol documents
  (`bench/loomyard-eval/README.md` and `bench/loomyard-eval/ladder/README.md`), keep every result
  and input: `bench/loomyard-eval/results/**`, `bench/loomyard-eval/ladder/results/**`,
  `bench/loomyard-eval/ladder/ladder*.yaml`, `bench/loomyard-eval/tasks/**`, and
  `bench/loomyard-eval/scripts/gen_compact_toc.py`.
- Rationale: both READMEs are operating manuals for the harness being deleted, and everything they
  document is on `v1-final`; the `conclusion.md` files inside the results are the record that
  survives. `bench/loomyard-eval/results/**` (the sibling, non-ladder suite) is not named in plan
  §2 either way, but it is the same category of record as `ladder/results/**` and is what
  `HANDOFF.md` draws on, so it is kept. `bench/loomyard-eval/README.md` also carries a
  `/home/knatte/...` machine path, which goes with it.
- Rejected: keeping the sibling suite's README for its methodology; deleting
  `bench/loomyard-eval/results/**` on the grounds that nothing can reproduce it.

### machine-paths-in-kept-records

- Decision: `docs/research/**` keeps its `/home/knatte/...` paths untouched. The "no tracked file
  may carry a machine path" constraint governs new and live files.
- Rationale: the seven affected files under `docs/research/` are captured transcript output and
  frozen research records — `docs/research/output-formats/` is explicitly the "before" side that T5
  adds an `after/` to. Rewriting captured output would falsify the comparison it exists to support.
- Rejected: scrubbing to a `<loomyard>` placeholder; deleting the offending files.

### doc-comment-pass-is-narrow

- Decision: rewrite `internal/quarryengine/doc.go` to a few lines describing what the package holds
  now (the two error sentinels and the cgo guard). Beyond that, fix only those comments in kept
  files that name a deleted package or name Rust/TypeScript. Every other comment stays as written.
- Rationale: the current root `doc.go` is a long description of seven verbs, the daemon, the LSP
  path and five languages — all gone. The rest of `toc`'s and `treesitter`'s comments describe code
  that is not changing, and rewriting them would be churn outside T0's scope.
- Rejected: a full comment pass over `toc` and `treesitter`; deleting `doc.go` entirely.

### five-commits-each-green

- Decision: five commits on the task branch, in this order, each of which must have
  `go build ./... && go test ./...` green:

  1. **Seven-verb surface.** Delete `cmd/quarry`, `cmd/quarry-mcp`, `internal/cli`,
     `internal/mcpserver`, `internal/output`, `quarry/`, `.mcp.json`, *and*
     `internal/quarryengine/layering_test.go` + `internal/quarryengine/seam_enforcement_test.go`.
  2. **LSP layer.** Delete `internal/lock`, `internal/proc`,
     `internal/quarryengine/{daemon,lsp,query,impact}` and `internal/quarryengine/registry`; move
     `extension.go` + `extension_test.go` into `toc`; trim the root package (`errors.go` down to
     the two sentinels, delete `log.go` and `position.go`, rewrite `doc.go`).
  3. **Rust and TypeScript.** Remove them from the grammar map, the extension table,
     `classify.go`'s `TestFile`, the tests, and the doc comments.
  4. **Harness and fixtures.** Delete `bench/loomyard-eval/ladder/{cmd,internal,tools}`,
     `run*.sh`, `launch-session.sh`, both bench READMEs, `.claude/skills/ladder-run/SKILL.md`, and
     `testdata/`.
  5. **Docs, README, gitignore, deps.** Delete the three V1 docs, write the README stub, trim
     `.gitignore`, run `go mod tidy`.
- Rationale: each step is closed under its importers, verified by grep: `internal/{cli,mcpserver,
  output}`, `quarry/` and `cmd/` are imported only by each other; `internal/lock` and
  `internal/proc` only by `daemon` (and one deleted `cli` test); `daemon`, `lsp`, `query` and
  `impact` have no importers outside themselves; `registry` is imported only by `toc` (moved in the
  same commit) and by packages deleted in steps 1–2. So every step ends green, and each leaves a
  reviewable diff of one kind of change.
- Rejected: one single deletion commit; two commits (code, then docs and deps).

### verification-gate

- Decision: `go build ./... && go test ./...` runs at every one of the five commits. At commit 5,
  additionally run `go mod tidy` and then `git diff --exit-code go.mod go.sum` to confirm tidy is
  idempotent. Separately at commit 5, grep `internal/` for `daemon|lsp|gopls|jsonrpc` as a one-off
  done-check; it is not committed as code.
- Rationale: the task's done criteria are exactly "build and test green", "`go mod tidy` leaves no
  Rust/TypeScript grammar and no LSP/JSON-RPC module", and "nothing under `internal/` references a
  daemon". A permanent grep-guard test would police packages that no longer exist — the same
  mistake as keeping `seam_enforcement_test.go`.
- Rejected: build and test only, with no tidy-idempotence check; adding `go vet ./...` at every
  commit.

### gitignore-trimmed-to-kept-paths

- Decision: in `.gitignore`, delete the `/quarry-mcp` entry and its comment (which cites the
  deleted `docs/mcp-setup.md`); keep `/quarry` and the `!/quarry/` re-include but reword their
  comment so it no longer cites `cmd/quarry` or a README build command; keep both
  `bench/loomyard-eval/ladder/results/**` rules (`raw/` and `.session-active`). Leave `*.test`,
  `*.test.exe`, `*.out`, `*.prof`, `.scratch/` and the entire `# === mill-managed ===` block
  untouched.
- Rationale: the results those two ladder rules protect are kept, and T2 rebuilds a harness that
  writes `raw/` again. The `/quarry` re-include is explicitly allowed to stay by the task body, and
  keeping it avoids a future re-add when T5 rebuilds the facade directory.
- Rejected: leaving `.gitignore` untouched; also dropping the two ladder `results/**` rules until
  T2 lands.

## Technical context

**Module.** Single Go module, `github.com/Knatte18/quarry`, Go 1.26. The benchmark harness lives
inside it (no separate `go.mod` under `bench/`), so deleting the harness also removes the last
non-registry user of `gopkg.in/yaml.v3`. The only nested `go.mod` files in the tree are the three
under `testdata/`, all deleted.

**What survives, in full:** `internal/quarryengine` (root, trimmed),
`internal/quarryengine/toc`, `internal/quarryengine/treesitter`. That is the entire Go tree after
this task. `cmd/` is gone; `quarry/` is gone.

**Import facts verified during exploration:**

- `internal/quarryengine/toc/toc.go` imports exactly three internal paths: the root package
  (`quarryengine`), `registry`, and `treesitter`.
- The only root-package symbols `toc` and `treesitter` use are `quarryengine.ErrNoLanguage` and
  `quarryengine.ErrLanguageUnsupported`. `Position` and `Logger` have no users among kept code.
- The only `registry` symbols `toc` uses are `LanguageForExtension` (`toc.go:38`, `toc.go:175`),
  `ExtensionsForLanguage` (`toc.go:161`) and `ExtensionLanguages` (`toc_test.go:802`, `:809`) —
  all four defined in `registry/extension.go`, which is self-contained (imports only `sort` and
  `strings`).
- `internal/quarryengine/{daemon,lsp,query,impact}` have no importers outside the packages being
  deleted.
- `internal/lock` and `internal/proc` are imported only by `daemon` and by one deleted `cli` test.
- Neither `toc` nor `treesitter` references anything under the root `testdata/`.

**Traps found:**

- `internal/quarryengine/seam_enforcement_test.go:91` calls `os.Stat` on the `quarry/` directory
  and `t.Fatalf`s when it is absent. Deleting `quarry/` without deleting this test in the same
  commit leaves that commit red. This is why the guard tests go in commit 1.
- `internal/quarryengine/toc/toc_test.go:802-815` cross-checks `registry.ExtensionLanguages()`
  against `treesitter.Supported()` in one direction and `Implemented()` against
  `ExtensionLanguages()` in the other. Dropping the Rust/TypeScript grammars *without* dropping
  `.rs`/`.ts`/`.tsx` from the table breaks the first check. Dropping both leaves all three sets
  equal to `{csharp, go, python}` and both checks pass unmodified.
- **Three `toc` tests depend on Rust, and only one of them says so in a greppable way.** The
  extension table is what makes a `.rs` file reach `toc` at all, so a test can depend on Rust
  purely through a filename. All three are listed under Testing with their disposition:
  `toc_test.go:765` (`TOCDir(dir, "rust")`), `toc_test.go:547`
  (`TestTOCDir_UnimplementedLanguageOnlyDirectoryIsNonEmpty`, which writes `main.rs` and calls
  `TOCDir(dir, "")`), and `toc_test.go:312`
  (`TestTOCFile_DesignedButUnimplementedLanguage`, which writes `main.rs` and calls
  `TOCFile(path, "")`). The middle one is the trap: it names no language, and it goes **red** in
  commit 3, because once `.rs` leaves the table `toc.go:175` returns `ok == false` and the loop
  `continue`s, so `len(Files)` is 0 where the test asserts 1.
- **How that inventory was derived, so a plan writer can re-derive it rather than trust the list:**
  grepping for `rust`/`typescript` is not sufficient. The complete derivation is (a) grep the
  package for `rust`, `typescript`, `\.rs`, `\.ts`, `\.tsx`; (b) grep for every remaining literal
  filename with one of those extensions, including ones built by `writeTempFile` /
  `writeDirFile` helpers; and (c) after making the change, run `go test ./...` and treat any
  failure in `toc` or `treesitter` as a missed site rather than as a reason to edit an unrelated
  test. Step (c) is the authoritative check; (a) and (b) only make it a short loop.
- `internal/quarryengine/toc/compact_test.go:41,52` build a fixture entry with
  `Name: "bad.rs", Language: "rust", Error: "language not yet supported"` and assert its rendered
  line. `DirEntry.Error` stays reachable after this task through `buildDirEntry`'s read-failure and
  invalid-UTF-8 branches (`toc.go:206-214`), so the rendering this fixture covers still has a live
  producer — **repoint** the row rather than delete it (see the `compact-test-row-repointed`
  decision).
- `internal/quarryengine/toc/classify_test.go:90-97,122` carry TypeScript and Rust cases for
  `TestFile` and `Generated`.
- `internal/quarryengine/treesitter/treesitter_test.go:29-30` parse TypeScript and Rust sources.
- `cgoguard_nocgo.go` deliberately references an undeclared identifier to fail a `CGO_ENABLED=0`
  build with a readable message. It is not broken; do not "fix" it.
- **The comment sites below are illustrative, not exhaustive — the rule in
  `doc-comment-pass-is-narrow` governs.** Any comment in kept code that names a deleted package or
  names Rust/TypeScript is in the pass, whether or not it appears here. Known sites:
  - `toc/strategy.go:76` — the `Strategy` interface doc says it is "designed to accommodate all
    five languages the toc survey covers (Go, Python, C#, TypeScript, Rust)".
  - `toc/classify.go:31,93-96,101-102,126` — the live `typescript` branch and its rust/typescript
    comments.
  - `toc/doc.go:1-11` — names `registry` and `internal/cli`, and cites the root package doc's
    "The engine/CLI split" section that commit 2 rewrites away.
  - `toc/toc.go:128` — names `registry.ExtensionsForLanguage`, which becomes an unqualified
    in-package call in commit 2.
  - `errors.go:1-2` and `errors.go:14-17` — name `internal/cli` as the sole caller and
    `DetectLanguage` as `ErrNoLanguage`'s producer; `DetectLanguage` lives in `registry/detect.go`
    and is deleted, so `ErrNoLanguage`'s doc must be restated in terms of its surviving producer in
    `toc`.
  - `extension.go:1-7` — arrives in `toc` in commit 2 still describing "package registry", the
    marker-based `DetectLanguage` it sat beside, and "the LSP verbs"; its header is rewritten as
    part of the move.

**`go.mod` after `go mod tidy`.** The surviving Go code uses only `github.com/tree-sitter/
go-tree-sitter` plus the `go`, `python` and `c-sharp` grammars. `cobra`, `pflag`, `mousetrap`,
`gofrs/flock`, `modelcontextprotocol/go-sdk`, `jsonschema-go`, `yaml.v3`, and the `tree-sitter-rust`
/ `tree-sitter-typescript` grammars all lose their last user; `go-cmp` is not used by any kept test
either. Let `go mod tidy` decide the final list rather than hand-editing `go.mod`.

**Reference documents.** `docs/rewrite-plan.md` §2 is the authoritative keep/delete table and §12's
T0 row restates it; §1 says why. `docs/glyph.md` is the contract T1 onwards builds on. Both are
kept and are not edited by this task.

## Constraints

- Go repo. Do not introduce Python (`CLAUDE.md`). The single sanctioned exception,
  `bench/loomyard-eval/scripts/gen_compact_toc.py`, is kept and unchanged; do not extend it and do
  not add another Python file.
- Deletion only. If a kept package fails to build because it imported a deleted one, remove the
  import and the dependent code path — never re-implement it.
- No new tracked file may carry a machine path. Existing machine paths inside the kept frozen
  records under `docs/research/**` are exempt (see the `machine-paths-in-kept-records` decision).
- Nothing is merged or cherry-picked from `v1-final`; V1 remains reachable there and that is the
  only place it needs to be.
- The `# === mill-managed (regenerated by mill-setup) ===` block in `.gitignore` is generated; do
  not hand-edit it.
- `_mill/` is task state and is not part of the deletion.
- Every one of the five commits must be green, not just the last one.

## Testing

There are no new tests in this task; T0 removes code and the tests that covered it. The test work
is (a) keeping the surviving `toc` and `treesitter` suites green, and (b) removing exactly the test
cases whose subject no longer exists.

**`internal/quarryengine/toc`** — the suite stays as-is apart from these edits, all in commit 3
except the import fix in commit 2:

- `toc_test.go`, commit 2: drop the `registry.` qualifier on the `ExtensionLanguages()` call sites
  at lines 802 and 809 (and in the comment at 799) when the table moves in-package.
- `toc_test.go`, commit 3: delete **three** tests, all of which depend on Rust being in the
  extension table:
  - line 765, the `TOCDir(dir, "rust")` override test — once `rust` is not a table language the
    override takes the unknown-language path and the expectation no longer holds.
  - line 547, `TestTOCDir_UnimplementedLanguageOnlyDirectoryIsNonEmpty` — writes `main.rs`, calls
    `TOCDir(dir, "")`, asserts `len(Files) == 1`. **This one goes red if it is left in place**, so
    it is not optional cleanup.
  - line 312, `TestTOCFile_DesignedButUnimplementedLanguage` — writes `main.rs`, calls
    `TOCFile(path, "")`, asserts a wrapped `ErrLanguageUnsupported`. It would still pass, but only
    by falling through to the unknown-extension branch that the `readme.md` test at line 304
    already covers, leaving a test whose name and comment describe a path that no longer exists.

  All three covered the same scenario — "extension is in the table, but no `Strategy` is
  registered for its language" — which cannot occur once the table is exactly
  `{go, python, csharp}` and all three register a strategy. Do not replace them with equivalents.
- `classify_test.go`: delete the TypeScript `TestFile` cases and the Rust `TestFile` /
  `Generated` "unknown" cases (commit 3).
- `compact_test.go`: **repoint**, do not delete, the `bad.rs` fixture row at line 41 and its
  expected output line at 52 — change them to a still-possible error entry (a Go file whose
  `Error` comes from the read-failure branch, e.g. `{Name: "broken.go", Language: "go", Error:
  "..."}` with the matching `want` line). The `"# internal/shed (package shed), 5 files"` header at
  line 46 then stays correct, and `CompactDir`'s error-line rendering keeps its coverage (commit
  3). See the `compact-test-row-repointed` decision.
- `extension_test.go`: arrives from `registry` in commit 2 with its package clause changed to
  `toc`; in commit 3 its `.rs` / `.ts` / `.tsx` expectations go.
- Everything else — `comments_test.go`, `sentences_test.go`, `golang_test.go`, `python_test.go`,
  `csharp_test.go`, `toc_integration_test.go` — must pass untouched. If one of them needs an edit,
  that is a signal the deletion went further than intended.

**`internal/quarryengine/treesitter`** — `treesitter_test.go` loses its TypeScript and Rust table
rows at lines 29-30 (commit 3). It contains no call to `Languages()` or `Supported()`, so there is
no set assertion in this package to update and none should be added. The only assertion anywhere
that pins the supported set is `toc_test.go:803`'s cross-check of the extension table against
`treesitter.Supported`, and it needs no edit: with Rust and TypeScript gone from both the grammar
map and the table, all three sets are `{csharp, go, python}` and both directions of the check hold
as written. The `WithTree` lifecycle tests (the `onRelease` seam) are unaffected and must pass
unchanged.

**`internal/quarryengine` (root)** — after `layering_test.go` and `seam_enforcement_test.go` are
deleted in commit 1, the package has no tests. That is correct: it holds two error sentinels and a
build guard.

**Scenarios that must be covered by the run, not by new code:**

- `go build ./... && go test ./...` green at each of the five commits.
- After commit 5, `go mod tidy` followed by `git diff --exit-code go.mod go.sum` produces no diff.
- After commit 5, a grep of `internal/` for `daemon|lsp|gopls|jsonrpc` returns nothing.
- After commit 5, `cmd/` and `quarry/` do not exist as directories, and `testdata/` does not exist.

**No TDD candidates.** Nothing is built; there is no behaviour to drive out with a failing test
first.

## Q&A log

- **Q:** Plan §2 (commit `51510f1`) says the V1 docs and `.mcp.json` are deleted and the README
  becomes a stub, while the older wiki task body says they may be trimmed or marked superseded.
  Which governs? **A:** Plan §2 — delete them; nothing is marked superseded.
- **Q:** Does the root package `internal/quarryengine` survive? **A:** Yes, trimmed to the cgo
  guard pair, a rewritten `doc.go`, and the two error sentinels `toc` uses.
- **Q:** Where does the extension→language table land, `toc` or `treesitter`? **A:** `toc` — its
  only remaining caller.
- **Q:** Are Rust and TypeScript dropped from the extension table as well as the grammar map?
  **A:** Yes. Rust and TypeScript are not supported, and the README must not name them; nothing
  that does not yet exist should be mentioned anywhere.
- **Q:** What happens to `bench/loomyard-eval/README.md` and
  `bench/loomyard-eval/ladder/README.md`, which §2 names neither way? **A:** Delete both —
  everything they describe is on `v1-final`, the results' `conclusion.md` files are the record, and
  the machine path in the former goes with it.
- **Q:** Rust/TypeScript leftovers inside kept code — strip or leave dormant? **A:** Strip
  entirely, comments included.
- **Q:** Keep, trim or delete `layering_test.go` and `seam_enforcement_test.go`? **A:** Delete
  both.
- **Q:** Delete the whole root `testdata/` directory? **A:** Yes.
- **Q:** How far does the `.gitignore` trim go? **A:** Trim to kept paths — drop `/quarry-mcp`,
  keep `/quarry` + `!/quarry/` with a reworded comment, keep both ladder `results/**` rules.
- **Q:** Commit shape? **A:** Five commits, one per layer — **and every one of the five must build
  and test green, not only some of them; order them so that holds.** (This is what forces the two
  guard tests into commit 1, since `seam_enforcement_test.go` fails when `quarry/` is missing.)
- **Q:** Which README stub text? **A:** The four-section version, including the cgo/Building note.
- **Q:** Machine paths inside the kept `docs/research/**`? **A:** Leave them — frozen records, not
  live configuration.
- **Q:** What verification runs? **A:** Build and test at every commit, plus `go mod tidy` with a
  `git diff --exit-code go.mod go.sum` idempotence check at commit 5.
- **Q:** How much of a doc-comment rewrite? **A:** Narrow — rewrite the root `doc.go`, and fix only
  comments that name a deleted package or Rust/TypeScript.
- **Q:** Is the "nothing under `internal/` references a daemon" criterion a committed test?
  **A:** No — a one-off grep at commit 5, recorded as a done-check.
- **Q:** Is `bench/loomyard-eval/results/**` (the sibling, non-ladder suite) kept? **A:** Yes —
  same category of record as `ladder/results/**`.
- **Q:** (review r1) Two more `toc` tests depend on Rust through a filename rather than the word
  "rust", and one of them — `TestTOCDir_UnimplementedLanguageOnlyDirectoryIsNonEmpty` — goes red in
  commit 3. What is their disposition? **A:** Delete both, along with the `TOCDir(dir, "rust")`
  test; the scenario all three cover cannot occur once the table is `{go, python, csharp}`.
- **Q:** (review r1) `buildDirEntry`'s no-strategy branch becomes unreachable. Delete it, keep it,
  or annotate it? **A:** Keep it unannotated — it is a live guard in a kept package, and T3/T8 put
  languages back.
