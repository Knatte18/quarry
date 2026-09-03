# Batch: answer-and-walk

```yaml
task: "Engine core (T3)"
batch: "answer-and-walk"
number: 3
cards: 10
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
depends-on: [1, 2]
```

## Batch Scope

This batch replaces the answer shape and the entry point: the two unrelated results `FileTOC` and
`DirTOC` become the one recursive `DirAnswer` of plan §4, the package-level `TOCFile`/`TOCDir`
become `Repo.TOC`, and the walk gains everything §4's directory answer needs — the
directory's package with its tie-break, the package doc, the language rule, every non-gitignored file rather than only `.go` files, lexicographic ordering, symlinks as name-only entries, target validation, and a per-call ignore set.

`Symbol` is deliberately untouched here: it keeps `Name`, `Owner` and `Docstring`, and the Go walk
keeps its three kinds. Batch 4 replaces it with the glyph-keyed symbol and widens the walk. Splitting
at that seam is what keeps both batches Sonnet-sized: this one is structure, I/O and answer shape;
that one is symbol identity and extraction.

The external interface batches 4 and 5 consume: `Repo`, `Open`, `TOC`, `TOCOptions`, `DepthAll`,
`DirAnswer`, `FileEntry`, and the two unexported walk seams `dirPackage` and `fileUnitPackage`.

Batch-local decision: committed fixture trees live under `internal/engine/testdata/` and contain no
`.gitignore` of their own — ignore behaviour is exercised from `.scratch/` trees per batch 2's
decision. Committed fixture `.go` files are walked by batch 6's round trip over quarry itself, which
is consistent by construction (both sides read the same files the same way) and therefore needs no
exclusion.

## Cards

### Card 13: The recursive answer types

- **Context:**
  - `docs/rewrite-plan.md`
  - `internal/engine/doc.go`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Delete `FileTOC`, `DirEntry` and `DirTOC`. Declare in their place:

  ```go
  type DirAnswer struct {
      Dir      string      `json:"dir"`
      Package  string      `json:"package,omitempty"`
      Language string      `json:"language,omitempty"`
      Doc      string      `json:"doc,omitempty"`
      Files    []FileEntry `json:"files,omitempty"`
      Dirs     []DirAnswer `json:"dirs,omitempty"`
  }

  type FileEntry struct {
      Name      string    `json:"name"`
      Header    string    `json:"header,omitempty"`
      Test      bool      `json:"test,omitempty"`
      Generated bool      `json:"generated,omitempty"`
      Package   string    `json:"package,omitempty"`
      Language  string    `json:"language,omitempty"`
      Lossy     bool      `json:"lossy,omitempty"`
      Error     string    `json:"error,omitempty"`
      Symbols   *[]Symbol `json:"symbols,omitempty"`
  }

  const DepthAll = -1

  type TOCOptions struct {
      Depth   int
      Symbols *bool
  }
  ```

  `Symbols` is the one pointer field and its doc comment must say why: absent means symbols were not
  requested, a present `[]` means they were requested and the file has no declaration, and
  `encoding/json`'s `omitempty` drops a nil and an empty slice alike — so only `*[]Symbol`
  distinguishes the two. `Test` and `Generated` are plain `bool` with `omitempty`, replacing V1's
  `*bool`: the pointer encoded "this language has no rule", a state that cannot arise while Go is the
  only language, and §4 forbids emitting `test: false` either way. Say that in their doc comments.
  `Lossy` carries what the old `Partial` field carried and its comment must say the rename frees the
  word `partial` for C#'s meaning; `Lossy` and `Error` stay mutually exclusive by construction.
  `TOCOptions.Depth` documents `0` as direct children at identity-plus-doc only, `N` as N levels
  filled, and `DepthAll` as the whole tree; `TOCOptions.Symbols` documents nil as the per-target
  default. Rewrite the file's header comment for its new name and contents. `Symbol` and `Kind` are
  not touched by this card.
- **Commit:** `feat(engine): replace FileTOC and DirTOC with the recursive DirAnswer`

### Card 14: The Strategy contract gains PackageDoc and loses the known returns

- **Context:**
  - `internal/engine/classify.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/strategy.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add one method to the `Strategy` interface:
  `PackageDoc(root *ts.Node, src []byte) string` — the file's package documentation, or the empty
  string when this file carries none. Its doc comment must state that this is a different rule from
  `Header`: `Header` returns the first non-directive leading block, `PackageDoc` returns only a block
  that is both immediately above the package clause and recognisable as package documentation by the
  language's own convention, and both are needed because a file can carry one, the other, or both.
  Change `Generated(root *ts.Node, src []byte) (generated, known bool)` to
  `Generated(root *ts.Node, src []byte) bool` and
  `TestFile(base string) (isTest, known bool)` to `TestFile(base string) bool`, and rewrite both doc
  comments: the `known` half existed only to feed the `*bool` fields card 13 removes, and with Go the
  only language and both Go rules always known it has no consumer left; a second language
  reintroduces the field and the return together. `TestFileByName` and `GeneratedByBanner` in
  `classify.go` keep their two-value signatures — they are the shared per-language table and a
  language with no rule still needs a way to say so there. Leave `Register`, `StrategyFor`,
  `Implemented` and `swapRegistry` unchanged.
- **Commit:** `feat(engine): add PackageDoc to Strategy and drop the known returns`

### Card 15: The Go package-doc rule

- **Context:**
  - `internal/engine/nodes.go`
  - `internal/engine/text.go`
  - `internal/engine/strategy.go`
  - `internal/engine/classify.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Implement `PackageDoc` on `goStrategy`. It finds the `package_clause` child of
  `root`, takes the comment block immediately above it — the same prev-sibling walk with the
  blank-line boundary that `CommentBlockAbove` performs — strips it with `StripComment(raw, "//")`,
  and returns `FirstParagraph` of it **only when** the stripped block's first line begins with
  `Package ` followed by the package name this file declares. Otherwise it returns the empty string.
  The prefix test is what the rule exists for: a file can carry a file header and a package doc as
  two separate leading blocks, and only the prefix tells them apart. Say that in the method's doc
  comment and name the failure the prefix prevents — an adjacent file header being read as the
  package doc.

  Change `Generated` and `TestFile` on `goStrategy` to the one-value signatures card 14 declares,
  discarding the second value `GeneratedByBanner` and `TestFileByName` return. Change nothing else
  in this file: the declaration walk, the span rules, `Header` and `Package` are batch 4's business.
- **Commit:** `feat(engine): implement the Go package-doc rule`

### Card 16: Repo, Open, and target validation

- **Context:**
  - `internal/engine/errors.go`
  - `internal/engine/answer.go`
  - `internal/engine/ignore.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/repo.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** New file `repo.go`, `package engine`. Declare:

  - `type Repo struct` holding the absolute repository root and nothing else. Its doc comment must
    state that nothing is cached on it — the gitignore pattern set is collected fresh on every call
    and discarded when the call returns — and give the reason: a long-lived process would otherwise
    serve a stale file list after a `.gitignore` edit, indistinguishable from a bug in the walk.
  - `func Open(root string) (*Repo, error)` — `root` must be an absolute path naming an existing
    directory; anything else is an error. `Open` performs no git discovery and no cwd resolution.
  - `var ErrTargetOutsideRepo` and `var ErrTargetNotFound`, both package-level sentinels created
    with `errors.New` and always returned wrapped via `fmt.Errorf("...: %w", ...)` so `errors.Is`
    survives wrapping.
  - `func (r *Repo) resolveTarget(target string) (rel string, info os.FileInfo, err error)` — the
    single validation path, in this order: a target that is absolute, or that leaves the root once
    cleaned (any leading `..`), returns `ErrTargetOutsideRepo`; a target that does not exist under
    the root returns `ErrTargetNotFound`; otherwise it returns the cleaned repository-relative,
    forward-slash path and the stat result. The stat is `os.Lstat`, never `os.Stat`, and the comment
    must say why: `os.Stat` follows a symlink and would silently descend into its target, which
    contradicts the never-follow rule for the one path that rule does not otherwise cover. `""` and
    `"."` both mean the repository root and are valid, and the root's own `rel` is `"."`.

  Validation deliberately does not consult the ignore set: the filter exists so a listing is not
  noise, not to make a file unaddressable, so an explicitly named gitignored target is answered.
  Say so in `resolveTarget`'s doc comment.
- **Commit:** `feat(engine): add Repo, Open and target validation`

### Card 17: The directory walk

- **Context:**
  - `internal/engine/repo.go`
  - `internal/engine/ignore.go`
  - `internal/engine/headers.go`
  - `internal/engine/extension.go`
  - `internal/engine/strategy.go`
  - `internal/engine/classify.go`
  - `internal/engine/answer.go`
  - `internal/engine/text.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/walk.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** New file `walk.go`, `package engine`. It holds the per-directory work `TOC`
  drives, as unexported methods on `*Repo`:

  - `dirPackage(dirRel string, entries []os.DirEntry) (pkg string, clauses map[string]string)` —
    read every `.go` file's package clause in the directory (via `treesitter.WithTree` and
    `goStrategy.Package`), return the directory's package and the per-file clause map. The
    directory's package is the most common clause among files whose clause does not end in `_test`;
    when every file's clause ends in `_test`, it is the most common clause overall. On a tie the
    lexicographically smallest clause wins, and the comment must say why: without a tie-break the
    answer would depend on `os.ReadDir`'s order, which is exactly what the ordering rule exists to
    eliminate.
  - `dirDoc(dirRel string, clauses map[string]string, pkg string) string` — the directory's package
    documentation. Candidates are the directory's `.go` files whose clause equals `pkg`, in sorted
    order with `doc.go` tried first; the first non-empty `goStrategy.PackageDoc` result wins. No
    match returns the empty string, which `omitempty` turns into an absent key rather than an empty
    one.
  - `fileEntry(dirRel, base string, dirPkg, dirLang string, clause string, wantSymbols bool) FileEntry`
    — build one file entry. `Name` is the base name. A file with a language (per
    `LanguageForExtension`) is read and parsed: `Header` is `FirstParagraph` of the strategy's
    `Header`, `Test` and `Generated` come from the strategy, `Package` is emitted only when the
    file's clause differs from `dirPkg`, `Language` is emitted only when the file's language differs
    from `dirLang`, and `Symbols` is set to a non-nil pointer — to a possibly-empty slice — only when
    `wantSymbols`. A file with no language gets `Header` from `HeaderForFile` and never gets
    `Symbols`, whatever `wantSymbols` says. A read failure or invalid UTF-8 sets `Error` and leaves
    `Header`, `Lossy` and `Symbols` unset; the file is still listed, never skipped. A parse that
    reports an error sets `Lossy`. `Error` and `Lossy` are never both set.
  - `walkDir(dirRel string, ig *ignoreSet, depth int, wantSymbols bool, identityOnly bool) (DirAnswer, error)`
    — the recursion. On entry it calls `ig.extend(dirRel)` and on exit `ig.trim` of the same count,
    so a directory's own patterns are in force for its subtree and are dropped again on the way out.
    `walkDir` owns the extend/trim for **its own** directory, at every level including the first:
    the caller hands it a set already carrying the chain from the root down to the target's
    **parent** and never the target's own, so no directory's patterns are appended twice and the
    trim accounting stays exact. State that split in the comment, since it is not inferable from
    either side alone.
    It reads the directory once with `os.ReadDir`, drops every entry `ig.match` excludes, and never
    descends into an excluded directory. An entry whose `DirEntry.Type()` has `fs.ModeSymlink` set
    becomes a `FileEntry` carrying `Name` alone — never descended into, never opened, no `Header`, no
    `Symbols` — whatever its target is; detection is on `Type()` and never on `IsDir()`, and the
    comment must say why: `IsDir()` is false for a symlink to a directory, so keying on it would
    emit a directory as a file entry and read a "header" through the link. Not following also means
    the walk is finite by construction and needs no visited set. `identityOnly` fills only `Dir`,
    `Package` and `Doc` — the shape a subdirectory takes at the depth cut. `Files` is sorted
    lexicographically by `Name` and `Dirs` by `Dir`, both with `sort.Slice` over the raw string, no
    case folding and no locale.

  The answer's `Dir` is the repository-relative path with forward slashes, `"."` for the root.
  `Language` on the directory answer is the language of its package, present only when there is one.
- **Commit:** `feat(engine): add the recursive directory walk`

### Card 18: Repo.TOC and the depth and symbols knobs

- **Context:**
  - `internal/engine/walk.go`
  - `internal/engine/repo.go`
  - `internal/engine/ignore.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Delete `TOCFile`, `TOCDir`, `buildDirEntry` and `resolveLanguage`. In their place
  declare `func (r *Repo) TOC(target string, opts TOCOptions) (DirAnswer, error)`.

  It validates `target` through `resolveTarget`, builds a fresh `ignoreSet` for the repository root
  and extends it along the chain from the root down to the target's **parent** — never the target's
  own directory, which `walkDir` extends itself on entry — and then answers:

  - A directory target: `walkDir` on it. `opts.Depth` of `0` fills the target's own `files` and
    lists its direct subdirectories as identity-plus-doc answers; `N` fills the files of
    subdirectories N levels down, each level's leaf `dirs` again identity-plus-doc; `DepthAll`
    recurses to the bottom.
  - A file target: the enclosing directory's `dir`, `package`, `language` and `doc`, with `files`
    holding exactly that one entry and no `dirs`. `opts.Depth` is ignored for a file target — there
    is nothing below a file to fill — and a non-zero depth with a file target is not an error. Say
    that in the doc comment.
  - A target that is itself a symlink: the name-only file entry inside its parent's directory
    answer, per the walk's rule, never followed and never opened.

  `opts.Symbols` nil means `true` for a file target and `false` for a directory target; a non-nil
  value wins for every file entry at every depth. Rewrite the file's header comment: it no longer
  describes two entry points and two post-processing rules but one entry point, the knobs, and the
  validation order. The `langOverride` parameter is gone for good — the alphabet is chosen per file,
  never per repository — and `ErrLanguageUnsupported` has no caller in this file any more.
- **Commit:** `feat(engine): replace TOCFile and TOCDir with Repo.TOC`

### Card 19: The committed fixture trees

- **Context:**
  - `internal/engine/walk.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/testdata/tree/README.md`
  - `internal/engine/testdata/tree/Makefile`
  - `internal/engine/testdata/tree/config.yaml`
  - `internal/engine/testdata/tree/notes.rst`
  - `internal/engine/testdata/tree/pkg/doc.go`
  - `internal/engine/testdata/tree/pkg/alpha.go`
  - `internal/engine/testdata/tree/pkg/beta.go`
  - `internal/engine/testdata/tree/pkg/alpha_test.go`
  - `internal/engine/testdata/tree/pkg/export_test.go`
  - `internal/engine/testdata/tree/sub/doc.go`
  - `internal/engine/testdata/tree/sub/deep/leaf.go`
  - `internal/engine/testdata/broken/ok.go`
  - `internal/engine/testdata/broken/syntax.go`
  - `internal/engine/testdata/broken/invalid.go`
  - `internal/engine/testdata/tiebreak/lib.go`
  - `internal/engine/testdata/tiebreak/tool.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Small, deliberate fixture trees. Under `tree/`: `pkg/` declares `package pkg` in
  `doc.go` (carrying a `// Package pkg ...` block with a second paragraph after a blank line),
  `alpha.go`, `beta.go` and `alpha_test.go`, while `export_test.go` declares `package pkg_test` — one
  directory, the external-test deviation, and a package doc whose second paragraph must not be
  emitted. `sub/` declares `package sub` in `doc.go` with a leading file header that is NOT prefixed
  `Package ` immediately above the clause, so the directory's `doc` is absent; `sub/deep/leaf.go`
  declares `package deep` and exists so depth cuts have something to cut. The four non-Go files
  exercise the header tables: `README.md` has an ATX heading and a paragraph, `Makefile` a leading
  `#` block, `config.yaml` a leading `#` block, and `notes.rst` no rule at all, so its entry is a
  name alone.

  Under `broken/`: `ok.go` is well-formed `package broken`, `syntax.go` is `package broken` with a
  deliberate syntax error so the parse reports one, and `invalid.go` holds a byte sequence that is
  not valid UTF-8. Under `tiebreak/`: `lib.go` declares `package alpha` and `tool.go` declares
  `package main` behind a `//go:build ignore` constraint, so the directory splits evenly between two
  clauses and the tie-break decides.

  None of these directories carries a `.gitignore` — ignore behaviour is exercised from `.scratch/`
  trees. Keep every file short; each exists to pin one rule, not to be realistic.
- **Commit:** `test(engine): add the committed walk fixture trees`

### Card 20: Port the existing tests onto Repo.TOC

- **Context:**
  - `internal/engine/toc.go`
  - `internal/engine/repo.go`
  - `internal/engine/walk.go`
  - `internal/engine/answer.go`
  - `internal/engine/treesitter/treesitter.go`
  - `internal/engine/scratchtree_test.go`
- **Edits:**
  - `internal/engine/toc_test.go`
  - `internal/engine/toc_integration_test.go`
  - `internal/engine/golang_test.go`
  - `internal/engine/classify_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `toc_test.go`, delete every case whose subject is gone —
  `TestTOCFile_UnsupportedExtension`, `TestTOCFile_LangOverrideWinsOverExtensionMismatch` and
  `TestTOCDir_LangOverrideRestrictsListing` test behaviour this batch removes on purpose. Rewrite
  every surviving case against `Open` plus `TOC`: the header-truncation, package, sig-end ordering,
  invalid-UTF-8, partial-parse, file-ordering, subdirectory, non-code-file, empty-directory,
  generated-banner and unreadable-file cases all have a direct equivalent on the new answer, with
  `Partial` becoming `Lossy` and the `*bool` assertions on `Test`/`Generated` becoming plain-bool
  ones. `TestTOCDir_NonCodeFileNotListed` inverts: a non-code file IS now listed, with a header and
  no symbols. `TestImplemented_MatchesRegisteredStrategies` and
  `TestExtensionLanguages_AllHaveGrammars` carry over unchanged.

  In `toc_integration_test.go`, replace the `TOCFile` call with `Open` on the module root and `TOC`
  on the repository-relative path of the treesitter source file, with `TOCOptions{}`. Read the one
  file entry out of `Files` and keep every assertion loose exactly as its own comment argues —
  language, package, not lossy, non-empty header, the three expected function names, and ascending
  starts.

  Every fixture in `toc_test.go` currently lands in a `t.TempDir()` — nineteen call sites including
  the `writeTempFile` helper — which the system-temp ban and the fixture Shared Decision both
  forbid. Rebuild every one of them through `writeScratchTree`, and delete `writeTempFile` in favour
  of it.

  In `golang_test.go` and `classify_test.go`, adjust what the changed `Strategy` signatures force:
  calls to `Generated` and `TestFile` now take one return value, and `classify_test.go`'s
  `fakeStrategy` — which implements the whole interface — gains a `PackageDoc` method returning the
  empty string and drops the second return from its own `Generated` and `TestFile`. Without that the
  package does not build. Do not otherwise rewrite them; batch 4 owns their widening.
- **Commit:** `test(engine): port the existing tests onto Repo.TOC`

### Card 21: Walk and target-validation tests

- **Context:**
  - `internal/engine/walk.go`
  - `internal/engine/repo.go`
  - `internal/engine/toc.go`
  - `internal/engine/answer.go`
  - `internal/engine/testdata/tree/pkg/doc.go`
  - `internal/engine/testdata/tiebreak/lib.go`
  - `internal/engine/scratchtree_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/repo_test.go`
  - `internal/engine/walk_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** `repo_test.go` covers target validation: an absolute target, a
  `..`-escaping target and a nonexistent target each asserted with `errors.Is` against their own
  sentinel; a gitignored target answered rather than refused; a target that is itself a symlink
  answered as a name-only entry inside its parent, not followed; and `""` and `"."` both answering
  the root with `dir` equal to `"."`. It also covers `Open` rejecting a relative root and a root that
  is not a directory.

  `walk_test.go` covers the walk itself against the committed fixtures and `.scratch/` trees:
  the directory's `package` and its `doc` for `tree/pkg` (first paragraph only) and the absent `doc`
  for `tree/sub` (a file header without the `Package ` prefix is not package documentation); the
  `package` deviation key appearing only on the external-test file, and a directory whose package is
  legitimately named `httptest` NOT being split; the tie-break under `tiebreak/` picking the
  lexicographically smaller clause and doing so identically across repeated runs; ordering
  over a `.scratch/` tree whose creation order is not lexicographic, asserting `files` and `dirs`
  come back sorted and symbols in source order; symlinks over a `.scratch/` tree with a
  symlink to a directory, a symlink to a file and a symlink cycle, asserting each is a name-only
  entry, that `DepthAll` terminates, and that nothing behind a link is listed or parsed; and a
  descendant `.gitignore` two levels down being honoured under `DepthAll`.
- **Commit:** `test(engine): cover the walk, ordering, symlinks and target validation`

### Card 22: Answer-shape and knob tests

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/toc.go`
  - `internal/engine/walk.go`
  - `internal/engine/testdata/tree/README.md`
  - `internal/engine/testdata/broken/ok.go`
  - `internal/engine/scratchtree_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/answer_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Assert the **marshalled JSON**, not the struct, so `omitempty` is pinned:
  `dirs` and `test` and `generated` absent when they would be false or empty, `error` and `lossy`
  absent on the happy path, and the `symbols` three-state — the key absent on a directory query, the
  key present as `[]` for a symbol-bearing query of a file with no declaration, and the key present
  and populated otherwise.

  Cover the knobs over the committed `tree/` fixture: `Depth` of `0` versus `1` versus
  `DepthAll`, with `0` listing direct subdirectories as `dir`, `package` and `doc` and no other key;
  `Symbols` defaulting per target kind for a file target and a directory target, and both explicit
  overrides winning; a file target answering as a one-entry directory answer that carries the
  enclosing directory's facts and no `dirs`; and a non-zero `Depth` on a file target changing
  nothing.

  Cover the failure entries over the committed `broken/` fixture: an unreadable file (created
  in a `.scratch/` tree, since a committed file cannot be unreadable), an invalid-UTF-8 file and a
  file the grammar reports an error on — `error` and `lossy` set, never both, and the file still
  listed rather than skipped. Cover the extensionless-file rule over `Makefile` and a file in
  neither header table.

  Cover ignore-set freshness with two `TOC` calls on **one** `Repo`, the `.scratch/` fixture's
  `.gitignore` rewritten between them: the second answer must reflect the new patterns. Nothing else
  in this task would notice a set accidentally cached on `Repo`.
- **Commit:** `test(engine): pin the answer shape, the knobs and the failure entries`

## Batch Tests

`verify:` is the same build-then-test pair the earlier batches use, so it covers
`internal/engine`, `internal/engine/treesitter` and `internal/cgoguard`.

New test files: `repo_test.go`, `walk_test.go` and `answer_test.go`. Ported: `toc_test.go` and
`toc_integration_test.go`, both rewritten onto `Open`/`TOC`; `golang_test.go` and `classify_test.go`
adjusted only for the changed `Strategy` signatures. `ignore_test.go`, `headers_test.go`,
`text_test.go`, `extension_test.go` and `treesitter/treesitter_test.go` must keep passing untouched.

Fixtures split two ways for the reason batch 2 states: committed trees under
`internal/engine/testdata/` for everything that does not involve `.gitignore`, and run-time trees
under `.scratch/engine-tests/` (via `writeScratchTree`) for the gitignore, symlink, unreadable-file
and ordering cases.
