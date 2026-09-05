# Batch: cli-repopath-mcp

```yaml
task: "Glyph self-form and the resolve contract (C1)"
batch: "cli-repopath-mcp"
number: 4
cards: 10
verify: go test ./internal/cli/... ./internal/repopath/... ./internal/mcpserver/...
depends-on: [3]
```

## Batch Scope

This batch removes the last two `#`-containment classifiers — `runResolve`'s and `parseArgs`' — and
adds the one reject the contract's "everywhere" clause requires on the verb that still takes paths.
It is one batch because those changes are the same sentence stated at four surfaces: the CLI's
resolve pipeline (D6), the CLI's expand usage gate (D19), `internal/repopath`'s target conversion
(D6, D8), and the MCP server's error mapping (D8). Their doc comments and the usage text are edited
in the same batch (D7, D21), because the sentence D7 mandates in `internal/cli/doc.go` — that
classification happens exactly once and it is `glyph.Parse` doing it — is false the day it is written
unless every card here has landed.

It closes the bounded red window batches 2 and 3 opened: after this batch,
`go test ./...` passes everywhere except the `after/` goldens, which batch 5 regenerates.

Batch-local decision beyond the overview's Shared Decisions: `toc`'s separator reject prints the
usage block after its sentence. Both existing exit-2 sites in `internal/cli/cli.go` pass `true` for
`fail`'s `withUsage` parameter while every `fail` inside `runTOC` passes `false`, so either flag had
local precedent. `true` wins because the flag tracks the exit code rather than the enclosing
function: exit 2 means the caller asked wrong, and printing the usage block is what every other exit
2 does.

## Cards

### Card 23: `resolve` performs no path arithmetic

- **Context:**
  - `internal/repopath/target.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `runResolve`, delete the `strings.Contains(target, "#")` branch and the
  `repopath.RepoRelPath` call inside it, so `req.target` reaches `repo.Resolve` verbatim. The local
  `target` variable becomes redundant; pass `req.target` directly. Drop the `strings` import from
  `internal/cli/cli.go` — card 28 removes this file's other mention of `strings.Contains`, which is
  prose in a doc comment, so nothing in this file uses the package afterwards. Keep the `repopath`
  import: `runTOC` still calls `repopath.RepoRelTarget`. The deliberate consequence is that a
  cwd-relative resolve target stops working: a self glyph's unit is repository-relative like every
  other unit, so the spelling is the full repository-relative path with a trailing `"#"` from
  anywhere. `toc` keeps cwd-relative targets, because paths are its domain. Change `runResolve`'s
  rendering, its exit-code mapping and `codeForResolveResult` in no way.
- **Commit:** `refactor(cli)!: pass every resolve target to the facade verbatim`

### Card 24: the exported `RepoRelPath` wrapper goes

- **Context:**
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/repopath/target.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Delete the exported `RepoRelPath` wrapper and its doc comment from
  `internal/repopath/target.go`. Card 23 removed its only caller. Keep the unexported `repoRelPath`
  exactly as it is: `repoRelTarget` is built on it, and its own doc comment's closing sentences
  describe the deleted wrapper as the caller that wants the arithmetic-only conversion — rewrite
  those sentences so they describe `repoRelTarget` as the only caller instead. Update the file
  header, which names both exported functions, to name `RepoRelTarget` alone.
- **Commit:** `refactor(repopath)!: delete the unused RepoRelPath wrapper`

### Card 25: `toc`'s path targets reject the separator

- **Context:**
  - `internal/engine/repo.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/repopath/target.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `repoRelTarget`, after the existing escape check that returns
  `quarry.ErrTargetOutsideRepo`, return `quarry.ErrTargetHasSeparator` when any `"/"`-separated
  segment of the cleaned repository-relative result contains a `"#"`. The order matters: an escaping
  target still reports the escape, which is why the new check goes after the existing one and not
  before it. Extend `repoRelTarget`'s doc comment and `RepoRelTarget`'s with the new sentinel.
  This is a behaviour change to `toc` — a target that succeeds today fails after — and the task's
  constraint against behaviour changes to `toc` is read as protecting its traversal, answer shape and
  defaults, which this leaves untouched; the contract's rule that a `"#"` in a path segment is an
  explicit error cannot hold "everywhere" while the one path-taking verb accepts such a target
  silently. Do not prune or skip such a target: silence is exactly what the rule bans.
- **Commit:** `feat(repopath)!: reject a path target carrying the glyph separator`

### Card 26: `runTOC` maps the sentinel to a usage error

- **Context:**
  - `quarry/quarry.go`
  - `internal/repopath/target.go`
  - `internal/cli/usage.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `runTOC`, the `repopath.RepoRelTarget` error block today tests
  `errors.Is(err, quarry.ErrTargetOutsideRepo)` and falls through to an internal-error arm. Add an
  explicit `errors.Is(err, quarry.ErrTargetHasSeparator)` branch before that fall-through, returning
  `fail` with `exitUsage` and the message `target contains the glyph separator "#": ` followed by the
  target as given, and with `fail`'s `withUsage` parameter set to `true` so `usageText` follows the
  sentence on stderr. Exit 2 rather than 1 because the target is malformed, not missing or out of
  scope, which is what separates it from the two existing exit-1 path failures in this function. Pass
  `req.target`, the argument as given, not the relativised form: the relativisation is what failed.
  Change `codeForTOCError` in no way — this error never reaches it, since the failure happens before
  `repo.TOC` is called. Card 28 owns the two comments this branch falsifies — `Run`'s doc comment
  step 1 for `runTOC`, and the `exitUsage` constant's enumeration of its causes — so make the code
  change here and leave both comment edits to that card, rather than editing the same file's prose
  from two cards.
- **Commit:** `feat(cli)!: map the separator sentinel to a usage error in toc`

### Card 27: `expand` loses its usage gate and gains the actionable message

- **Context:**
  - `glyph/errors.go`
  - `internal/engine/expand.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/flags.go`
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Delete the `verb == "expand" && !strings.Contains(req.target, "#")` gate in
  `parseArgs` and the usage message it builds. Keep the `strings` import in
  `internal/cli/flags.go`: `parseArgs` still calls `strings.HasPrefix` and `strings.Cut`. Rewrite
  the closing sentences of `parseArgs`' own doc comment, which state that for `expand` specifically a
  target containing no `"#"` is rejected there and that its fixture-free table test rests on that
  property — the rejection and the property both go. In `runExpand`, the `*glyph.ParseError` branch
  builds its message from the reason word alone; change it to `"expand: "` followed by
  `parseErr.Error()`, so the message carries the grammar's full sentence including the
  repository-relative clause and names the target once rather than twice — the `*ParseError` already
  quotes its own input. Without this change the gate's deletion would emit the bare reason word,
  which is strictly less actionable than the usage message it replaced. This also changes the message
  for `expand`'s existing grammar rejections, which is the same improvement applied consistently.
  Add the `*SelfGlyphError` branch to `runExpand` beside the `*NotATypeError` branch, using
  `errors.As` against `*quarry.SelfGlyphError` and producing `expand ` followed by the value's `ID`
  field and `: not a type, self` — spelled from the value's fields rather than the error's own text,
  so the engine's package-name prefix never leaks, exactly as the `*NotATypeError` branch does. Add
  a matching `*quarry.SelfGlyphError` arm to `codeForExpandError` returning `exitNegative`, placed
  beside its existing `*quarry.NotATypeError` and `*glyph.ParseError` arms. That function's fallback
  is `exitInternal`, so without the new arm a self glyph given to `expand` would exit 3 — an internal
  error — rather than the negative exit the answer is: the verb ran to a definite conclusion and the
  glyph names no type. The `glyph:` prefix the rewritten `*glyph.ParseError` message carries is
  intended and is not the leak the surrounding code guards against: `github.com/Knatte18/quarry/glyph`
  is a public package named in the contract, not one of the internal packages whose names the exit-1
  message rule keeps out, and the full sentence is exactly what makes the message actionable. Card 28
  records that distinction in the doc comment. A bare path given to `expand` now exits 1 with the
  payload-plus-message shape instead of exit 2 with a usage message, which is the deliberate breaking
  change this card carries.
- **Commit:** `refactor(cli)!: delete expand's usage gate and carry the grammar's own message`

### Card 28: the package doc states the new contract, and `--help` stops promising the old one

- **Context:**
  - `glyph/parse.go`
  - `internal/repopath/target.go`
- **Edits:**
  - `internal/cli/doc.go`
  - `internal/cli/cli.go`
  - `internal/cli/usage.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Ten paragraphs across these three files are now false or incomplete, and all ten
  change.
  In `internal/cli/doc.go`: delete the "Known contract gap" paragraph outright, and replace it with a
  statement of the new rule — `resolve` takes glyphs, `toc` takes paths, classification happens
  exactly once and it is `glyph.Parse` doing it, and a `"#"` in a path segment is an explicit error at
  both verbs rather than a reclassification; rewrite the paragraph opening "A target containing a
  `"#"` is classified as a glyph by this package, on sight", since that code is gone — identify it by
  those opening words rather than by position, as it is two paragraphs above the contract-gap one and
  not adjacent to it; and rewrite the three-verbs paragraph, whose sentence saying `resolve` takes
  either a path or a glyph is exactly the contract this task reverses. In `internal/cli/cli.go`:
  rewrite the `exitUsage` constant's doc comment, which enumerates three causes — an unparseable
  flag, a missing or extra target, and a `--root` that does not resolve to a directory — and now
  needs card 26's fourth, a target carrying the glyph separator; its closing sentence saying the TOC
  query is never called on this path stays true, since card 26's branch fires before the repository
  is opened, so confirm it rather than rewriting it. Rewrite `runTOC`'s step 1 in `Run`'s doc
  comment, which documents the target conversion as producing exit 1 for an escaping target and
  nothing else, to carry the separator reject and its exit 2. Rewrite step 1 of the same doc
  comment's `runResolve` list — a different numbered list from `runTOC`'s, further down — which
  narrates that verb's classification and its `RepoRelPath` conversion, both deleted by card 23;
  rewrite the separate `runExpand` preamble paragraph asserting that the parser
  has already guaranteed the target contains a `"#"`, which card 27's deleted gate falsifies — this is
  a distinct paragraph from step 1, and editing step 1 alone would leave it standing; and rewrite
  step 3, which documents the expand `*glyph.ParseError` message as the value's reason word and the
  same word the resolve verb puts in its payload's reason key, which card 27 replaced with the
  error's full text, adding the `*SelfGlyphError` branch to the same numbered list; and amend the
  closing paragraph on message composition, which states that an exit-1 message never carries a
  wrapped chain because quarry spells those conditions itself and passing a chain through would leak
  an internal package name into a public contract. That rule stands and is not being relaxed: it
  names *internal* packages, and `github.com/Knatte18/quarry/glyph` is a public package the contract
  already names, so the grammar's own sentence is quarry spelling the condition rather than leaking
  one. Say that explicitly, so the next reader does not read card 27's message as a violation of the
  paragraph directly above it. In `internal/cli/usage.go`: change the `resolve` line's argument from `<glyph|path>` to `<glyph>`. The
  `toc` and `expand` lines stay as they are. Re-read the exit-codes block against card 27's change
  and confirm it needs no edit — its exit-1 line already ends "or not a well-formed glyph", which now
  also covers a bare path given to `expand`. Preserve this file's ASCII-only, byte-comparable
  constraint: no em dash and no typographic quotes.
- **Commit:** `docs(cli): state the glyph-only resolve contract at every surface`

### Card 29: the MCP server reports the separator as a user error

- **Context:**
  - `quarry/quarry.go`
  - `internal/cli/cli.go`
  - `internal/repopath/target.go`
- **Edits:**
  - `internal/mcpserver/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `tocResult`, the `repopath.RepoRelTarget` error block is one
  `errors.Is(err, quarry.ErrTargetOutsideRepo)` test plus an else-arm prefixing `internal error: `.
  Add an explicit `errors.Is(err, quarry.ErrTargetHasSeparator)` branch before that else-arm,
  returning `errorResult` with the same sentence `runTOC` emits: `target contains the glyph separator
  "#": ` followed by the target as given. The two surfaces must emit byte-identical text so they
  cannot drift. Without the branch the sentinel falls into the else-arm and a malformed user target
  is reported to an MCP client as a server fault carrying the engine's package-namespaced sentinel
  text. Change no other line of this file: the server exposes the `toc` tool only, its traversal and
  answer shape are out of scope, and there is no `resolve` tool here for the field rename to reach.
- **Commit:** `feat(mcpserver): report a separator-bearing target as a user error`

### Card 30: `repopath` tests for the reject and the retained helper

- **Context:**
  - `internal/repopath/target.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/repopath/target_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add cases asserting `RepoRelTarget` returns an error satisfying
  `errors.Is(err, quarry.ErrTargetHasSeparator)` for a target whose cleaned relative form carries a
  `"#"` in its first segment, in a middle segment, and in its basename; that an escaping target still
  reports `quarry.ErrTargetOutsideRepo` rather than the separator error; and that a clean target is
  unaffected. Keep `TestRepoRelPath_LeadingDotDotNotRejected` and
  `TestRepoRelPath_AgreesWithRepoRelTarget`: both call the unexported `repoRelPath`, which card 24
  retains, so deleting them would drop coverage of behaviour this task preserves. Rename both off the
  deleted exported symbol's name so no test is named after a function that no longer exists. Amend
  the agreement test: its claim today is that the two functions agree on every input that does not
  escape the root, which narrows once `repoRelTarget` gains the separator reject — the amended claim
  is that they agree on every input that neither escapes the root nor contains a `"#"`, with the
  separator divergence asserted as its own row rather than left implicit. Delete only the tests that
  exercise the deleted exported wrapper, if any exist beyond those two.
- **Commit:** `test(repopath): pin the separator reject and the retained arithmetic`

### Card 31: CLI tests for the new contract

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `internal/engine/answer.go`
  - `glyph/errors.go`
- **Edits:**
  - `internal/cli/cli_test.go`
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Retarget every resolve row in `internal/cli/cli_test.go` whose target is a bare
  path. A directory target and a file target become their self-glyph spellings, with a trailing
  `"#"`, and keep their exit 0 and their listing assertions. The rows that give `resolve` a
  non-existent name, a `..` target and an absolute path become bare-path rejections: exit 1, the
  payload on stdout, nothing on stderr, and a `reason` of `no_separator`. Add a row for
  `resolve` with a two-separator target: exit 1 with the payload on stdout. Add a row for a self
  glyph naming a missing file: exit 1. Add a row for a self glyph naming an external test unit: exit
  1. Cover both the JSON and the `--text` view for each resolve row, since the contract requires
  both. Add a row for `expand` of a self glyph: exit 1, the failure envelope on stdout and the
  message on stderr. Add the `toc` row for a target carrying a `"#"`: exit 2, whose stderr is the
  separator sentence followed by the full usage block — assert both parts, since the sentence alone
  would pass a substring check while pinning the wrong bytes. Add the two-verb agreement test: the
  same bare path given to `expand` and to `resolve` both exit 1, asserted in one test so their
  agreement is what the test is about rather than a coincidence of two separate rows, and assert
  `expand`'s full sentence rather than the bare reason word, since emitting the reason word alone is
  precisely the regression card 27 guards against. Pin the changed message for a two-separator target
  given to `expand`. In `internal/cli/flags_test.go`, delete the case pinning the deleted
  `expand` usage gate, which asserts the message beginning `expand takes a glyph`; grep for that
  string first rather than assuming where it is.
  The `target-echo-asymmetry` subtest in `internal/cli/cli_test.go` loses its premise entirely: it
  exists to contrast a relativised path target, echoed as its repository-relative form, against a
  glyph target echoed verbatim, and card 23 removes the relativisation so both halves now echo the
  argument verbatim. Rename it to say that the target field is always the argument verbatim, and
  retarget its first half to assert exactly that on the absolute-path rejection — the rejection
  payload's target is the absolute argument as given, not a relativised form. Keep its second half,
  the glyph row, unchanged; the test now asserts a rule rather than a contrast.
  Do not edit `internal/cli/glyph5_test.go`: it uses member glyphs only; confirm that rather than
  assuming it.
- **Commit:** `test(cli): pin the glyph-only resolve contract end to end`

### Card 32: the MCP error surface's exact sentence

- **Context:**
  - `internal/mcpserver/toc.go`
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/mcpserver/toc_errors_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a case asserting a `"#"`-bearing target reaches the server's error surface
  with the separator message rather than a listing. Assert the exact sentence, not merely that an
  error came back: the failure this guards against is the sentinel falling into the
  `internal error: ` else-arm, which still produces an error result and would pass a shape-only
  assertion. Assert too that the sentence is byte-identical to the one `runTOC` emits, so the two
  surfaces cannot drift apart later — spell the expected sentence once in this test and state in its
  doc comment which CLI site it mirrors. Follow the existing cases' shape in this file for building
  the fixture and reading the result.
- **Commit:** `test(mcpserver): pin the separator sentence byte for byte`

## Batch Tests

`verify: go test ./internal/cli/... ./internal/repopath/... ./internal/mcpserver/...` covers exactly
the three packages this batch changes. `internal/cli`'s run includes `cli_test.go`, `flags_test.go`,
`glyph5_test.go`, `message_test.go`, `plan4_test.go` and `scratchtree_test.go`; the `after/` golden
test in the same package skips, because `LADDER_LOOMYARD_REPO` is unset for this batch and its
goldens still hold the pre-rename bytes until batch 5 regenerates them.

The regression gates inside that run are `TestRun_ExitCodeMapping` and `TestCodeForExpandError`,
which fail if a code moves without its table row, and `TestRun_UsageTextPlacement`, which fails if
the separator reject's `withUsage` flag disagrees with the exit code it carries.
