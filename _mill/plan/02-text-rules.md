# Batch: text-rules

```yaml
task: "Engine core (T3)"
batch: "text-rules"
number: 2
cards: 5
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./internal/...
depends-on: [1]
```

## Batch Scope

This batch adds the two pure-text rules the new walk depends on and nothing else: the gitignore
matcher and the non-code header extractors. Both are pure functions over strings and
paths with no tree-sitter node in any signature, so they are written test-first and land before the
walk that consumes them. Nothing in the package calls them yet, so the batch cannot regress any
existing behaviour — that is why it is separated from batch 3.

The external interface batch 3 consumes: `newIgnoreSet`, `(*ignoreSet).extend`, `(*ignoreSet).match`
in `ignore.go`, and `HeaderForFile` in `headers.go`.

Batch-local decision: every fixture that exercises `.gitignore` behaviour is built at run time under
`.scratch/engine-tests/<test-name>/`, never committed under `testdata/`. A committed fixture tree
cannot contain a file its own `.gitignore` excludes — git would refuse to track it without
`git add -f`, and a force-added-but-ignored file is exactly the confusing state these tests exist to
reason about. `.scratch/` is gitignored at the repository root, so such trees are also invisible to
the batch 6 round trip over quarry itself.

## Cards

### Card 8: The gitignore matcher

- **Context:**
  - `.gitignore`
  - `internal/engine/doc.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/ignore.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** New file `ignore.go`, `package engine`. Declare an unexported
  `ignorePattern` struct holding the parsed form of one `.gitignore` line (the glob, whether it is
  negated, whether it is directory-only, and the repository-relative directory of the `.gitignore`
  file it came from), and an unexported `ignoreSet` type holding an ordered slice of them.
  Declare:

  - `newIgnoreSet(root string) *ignoreSet` — an empty set for a repository rooted at `root`.
  - `func (s *ignoreSet) extend(dirRel string) error` — read the `.gitignore` file in the
    repository-relative directory `dirRel` when one exists, append its patterns to the set, and
    record how many were appended so `trim` can undo it; a missing file is not an error.
  - `func (s *ignoreSet) trim(n int)` — drop the last `n` appended patterns, so the walk can leave a
    directory and lose that directory's own patterns again.
  - `func (s *ignoreSet) match(pathRel string, isDir bool) bool` — report whether `pathRel` (a
    repository-relative, forward-slash path) is excluded.

  Parsing rules: a blank line and a line whose first character is `#` are skipped; a leading `!`
  marks a negation; a trailing `/` marks directory-only; a leading `/` anchors the pattern to its own
  `.gitignore`'s directory; a pattern containing a `/` anywhere other than its trailing position is
  likewise anchored to that directory; a pattern with no `/` matches at any depth below that
  directory. Within one path segment, `*` matches any run of non-`/` characters and `?` matches one
  non-`/` character; `**` matches across segments. Later patterns win over earlier ones, so `match`
  evaluates the set in order and returns the last match's polarity. `.git` is always excluded as a
  directory, unconditionally and before any pattern is consulted.

  Directory pruning is part of this rule and not the caller's: state in the file's own header comment
  that a directory excluded by `match` is never descended into, so nothing beneath it can be
  re-included by a later pattern unless the directory itself is re-included first — git's own rule,
  and the one quarry's own `/quarry` plus `!/quarry/` pair turns on. Also state in that comment what
  is deliberately NOT supported: `core.excludesFile`, `.git/info/exclude`, and any `.gitattributes`
  interaction.
- **Commit:** `feat(engine): add the gitignore matcher`

### Card 9: The gitignore matcher's table tests

- **Context:**
  - `internal/engine/ignore.go`
  - `.gitignore`
- **Edits:** none
- **Creates:**
  - `internal/engine/ignore_test.go`
  - `internal/engine/scratchtree_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** `scratchtree_test.go` declares one shared test helper,
  `writeScratchTree(t *testing.T, name string, files map[string]string) string`, which resolves the
  module root from `runtime.Caller(0)`, creates `.scratch/engine-tests/<name>/`, writes each
  `files` entry (key = a forward-slash relative path, value = its contents, parent directories
  created as needed), registers a `t.Cleanup` that removes the tree, and returns its absolute path.

  The helper writes regular files only. A test needing a symlink, a directory-only entry, an
  unreadable file, or a specific creation order creates it itself on the path the helper returns —
  that is deliberate, so one helper does not grow a mode/type/order parameter for the handful of
  cases that need one. Say so in its doc comment.

  It must never call `t.TempDir()` — the system temp directory is banned by this task's constraints,
  and `.scratch/` is the sanctioned location.

  `ignore_test.go` drives `newIgnoreSet`/`extend`/`trim`/`match` through a table covering every
  pattern form card 8 lists: a comment line, a blank line, a bare name matching at any depth, an
  anchored `/name`, a directory-only `name/`, an interior-slash pattern (anchored by its slash,
  directory-only by its trailing one), `*` and `?` within a segment, `**` across segments, and a
  negation that re-includes a previously excluded path. Two cases use the repository's own real
  patterns as fixtures: the `/quarry` plus `!/quarry/` pair — the binary is excluded and the
  directory is not — and a `**/`-prefixed pattern. A separate case asserts that `.git` is excluded
  with no pattern present at all, and another that `trim` restores the previous set exactly.
- **Commit:** `test(engine): table-test the gitignore matcher`

### Card 10: The non-code header rules

- **Context:**
  - `internal/engine/text.go`
  - `internal/engine/doc.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/headers.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** New file `headers.go`, `package engine`. Declare
  `type headerRule func(src []byte) string` and the five concrete rules the next card's tables map
  to: `markdownHeader`, `htmlCommentHeader`, `cssCommentHeader`, `scriptCommentHeader` and
  `hashBlockHeader`. Each is pure: bytes in, prose out, no I/O, no tree-sitter. This card declares
  the type and the rules only; the tables and the entry point that dispatches through them are the
  next card's, so nothing here refers to a table that does not exist yet.

  - `markdownHeader` returns the file's first ATX (`#`-prefixed) or setext (underlined with `=` or
    `-`) heading, then the first paragraph that follows it, joined by a newline. A file with no
    heading returns the empty string.
  - `htmlCommentHeader` returns the prose of a leading `<!-- ... -->` comment, delimiters stripped.
  - `cssCommentHeader` returns the prose of a leading `/* ... */` comment.
  - `scriptCommentHeader` returns the prose of a leading run of `//` lines, or of a leading
    `/* ... */` comment, whichever the file starts with.
  - `hashBlockHeader` returns the prose of a leading run of `#`-prefixed lines, skipping a shebang
    line (`#!`) when it is the first line.

  Every rule returns the first paragraph only, via `FirstParagraph`, so a long leading comment does
  not become the header. Leading blank lines are skipped before the rule looks for its delimiter; a
  comment that does not start the file is not a header.
- **Commit:** `feat(engine): add the per-format non-code header rules`

### Card 11: The lookup tables and HeaderForFile

- **Context:**
  - `internal/engine/doc.go`
- **Edits:**
  - `internal/engine/extension.go`
  - `internal/engine/headers.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Leave `extensionLanguages` and its three views (`LanguageForExtension`,
  `ExtensionsForLanguage`, `ExtensionLanguages`) exactly as they are — the extension-to-*language*
  table stays Go-only. Add two new unexported lookup tables beside them in `extension.go`:

  - `extensionHeaderRules map[string]headerRule` — keyed by a lowercase, dot-prefixed extension.
  - `baseNameHeaderRules map[string]headerRule` — keyed by an exact file base name, consulted only
    when `filepath.Ext` returns the empty string, and holding `Makefile` and `Dockerfile`.

  Populate `extensionHeaderRules`: `.md` to the Markdown rule; `.html` and `.htm` to the leading
  `<!-- ... -->` rule; `.css` to the leading `/* ... */` rule; `.js`, `.mjs` and `.ts` to the leading
  `//`-block-or-`/* ... */` rule; `.yaml`, `.yml`, `.toml`, `.sh`, `.bash` and `.zsh` to the leading
  `#`-block rule. Add a comment stating that these tables are deliberately separate from
  `extensionLanguages`: an entry here gives a file a header, never a language, and never `symbols`.
  Explain in that same comment why the base-name table exists rather than a sentinel key inside the
  extension table — an extensionless file is a real case and a key that reads like an extension and
  is not one would be a lie.

  Then add the one entry point the walk calls to `headers.go`:
  `HeaderForFile(base string, src []byte) string`. It looks `filepath.Ext(base)` up in
  `extensionHeaderRules`, falls back to `baseNameHeaderRules[base]` when the extension is the empty
  string, and returns the empty string when neither table has an entry. It never consults
  `extensionLanguages` and never parses Go — a Go file's header comes from the Go strategy, not from
  here.
- **Commit:** `feat(engine): add the non-code header tables and HeaderForFile`

### Card 12: Header-rule and FirstParagraph tests

- **Context:**
  - `internal/engine/headers.go`
  - `internal/engine/extension.go`
- **Edits:**
  - `internal/engine/text_test.go`
- **Creates:**
  - `internal/engine/headers_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** `headers_test.go` drives `HeaderForFile` through one table per format, covering
  at minimum: a Markdown file with an ATX heading and a following paragraph; a Markdown file with a
  setext heading; a Markdown file whose first paragraph follows a blank line after the heading; a
  Markdown file with no heading at all; a shell script whose `#` block follows a shebang; a YAML file
  with no leading comment; an HTML file with a leading `<!-- -->`; a CSS file; a JavaScript file with
  a `//` block and one with a `/* */` block; a `Makefile` and a `Dockerfile` resolving through the
  base-name table; an extensionless file in neither table; and an unknown extension. Every case
  asserts the exact returned string.

  In `text_test.go`, extend `TestFirstParagraph` with the two cases the package-doc rule turns on: a package comment
  that continues after a blank line (only the first paragraph is returned) and one that does not
  (the whole text is returned). Do not change the existing `TestStripLineComment` or
  `TestStripComment` cases.
- **Commit:** `test(engine): cover the non-code header rules and FirstParagraph`

## Batch Tests

`verify:` is the same build-then-test pair batch 1 uses. The tests this batch adds —
`ignore_test.go` and `headers_test.go` — are the batch's own verification; the ported tests carried
forward from batch 1 must keep passing, which is what proves the new files changed no existing
behaviour.

`ignore_test.go` builds its fixture trees at run time under `.scratch/engine-tests/` via the
`writeScratchTree` helper card 9 adds, for the reason stated in Batch Scope. `headers_test.go` needs
no tree at all: its subjects take bytes.
