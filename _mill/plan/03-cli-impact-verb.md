# Batch: cli-impact-verb

```yaml
task: "Add `impact` verb for caller-context lookup"
batch: "cli-impact-verb"
number: 3
cards: 4
verify: go build ./... && go test ./internal/cli/...
depends-on: [2]
```

## Batch Scope

This batch delivers the user-facing verb: a new `internal/cli/impact.go` holding
`impactCommand()` and the three small `impact`-typed helpers the result type forces, the
registration of that command on the cobra tree, the two CLI-package doc sites that enumerate the
verb set, the CLI-level tests, and the correction of the hand-enumerated four-verb build-tags table
that would otherwise leave the new verb's flag registration untested.
It is one batch because the command, its registration, and the table that mechanically enforces its
flag set are a single reviewable unit — the table's second loop is what turns the Scope "Out" claim
that `impact` gains no `--no-verify` flag from prose into a check.

Batch-local decision on file placement, recorded because it deviates from `discussion.md`:
that document's `cli-shape-mirrors-refs` decision places `impactCommand()` in `internal/cli/cli.go`,
alongside the four existing commands.
This plan puts it in a new `internal/cli/impact.go` instead, together with the three `impact`-typed
helpers, following `internal/cli/toc.go`'s established precedent of giving a verb group its own file
plus the helpers only it uses.
The reason is size and blast radius: `internal/cli/cli.go` is already 972 lines, the new command and
its three helpers add a comparable block again, and card 11 must make three surgical doc edits to
that same file — keeping the new code out of it makes both diffs reviewable.
Nothing else about `cli-shape-mirrors-refs` changes: the command's shape, flags, query construction,
batch behaviour, and exit-code contract all follow `refsCommand` exactly as that decision requires.
This is a recorded deviation, not drift.

Batch-local decision beyond `## Shared Decisions`: `buildQuery` is *not* a shared helper to call.
It is a per-command local closure, duplicated today at both `refsCommand` and `definitionCommand`,
so `impactCommand` re-creates it the same way. That closure is what makes `--in-file` compose with
batch mode and keeps a positional from ever being position-parsed under `--in-file`.

Batch-local decision on test reach: `internal/cli` has no fake language server, and every existing
test in the package either drives a path that fails before any server spawn or calls a pure helper
directly. This batch's tests follow that same split rather than inventing an LSP fake — the
end-to-end claim is proved by the live-tier test in batch 4.

## Cards

### Card 10: The impact command and its result-typed helpers

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/toc.go`
  - `internal/cli/paths.go`
  - `internal/cli/cwdcontext.go`
  - `internal/cli/exec.go`
  - `internal/output/output.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/impact.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `impactCommand() *cobra.Command`, mirroring `refsCommand`'s shape exactly: the same
  `CwdFrom` / target-directory defaulting / `resolveContext` / `resolveBuildTags` / `buildOptions`
  sequence, the same per-command local `buildQuery` closure dispatching to `inFileQuery` when
  `--in-file` is set and `parseQuery` otherwise, `Args: cobra.MinimumNArgs(1)`, a single-argument
  path, and a batch path through the existing `runBatch`, symbol-keyed.
  Register exactly six flags, matching `refsCommand`'s set: `--target-dir`, `--lang`, `--timeout`
  (default `30*time.Second`), `--in-file`, `--within`, and `--build-tags`.
  Copy `refsCommand`'s help strings verbatim for four of them — `--target-dir`, `--lang`,
  `--in-file`, and `--build-tags`. Two are exempt and need their own wording.

  `--timeout`: copy `assertNoCallersCommand`'s wording instead — "deadline for each LSP request
  phase (initialize, resolve, references/definition)".
  `refsCommand`'s narrower "(initialize, resolve, references)" would be false here, because
  `impact` goes through `query.Callers`, which also runs a definition phase, an implementation
  phase, and a per-reference verification phase — the same phase set `assert-no-callers` already
  describes with the wider wording.
  The help string must not name the tree-sitter parse phase either, since `--timeout` deliberately
  does not cover it.

  `--within`: `refsCommand`'s text reads "restrict results to references whose file lies within this
  directory", which is false on this verb — here the flag filters the **caller list only** and
  leaves `target` and `definition` untouched, exactly as `filterImpactWithin` below requires.
  Word it as restricting the reported callers to those whose file lies within this directory
  (relative to `--target-dir`, or absolute), and keep a pointer to the interface-method conflation
  note in the Long help, which is the reason the flag is useful here.
  Register no `--no-verify` flag and never set `SkipVerification` — that flag is
  `assert-no-callers`-only.

  Write the command's Long help in `refsCommand`'s style, and include, verbatim, the
  `verification-is-best-effort-and-resolution-means-what-it-means-on-refs` Shared Decision's wording:
  `"resolution":"complete"` means *the language server returned every reference for the query as
  given*, and asserts nothing about per-caller verification having run nor about enclosing ranges
  having resolved. Place that next to a `--within` note explaining the interface-method conflation,
  since an unverified set is precisely the noisy case.
  Also document that both the resolved symbol's `definition` range and each caller's
  `enclosing_range` include the docstring immediately preceding the declaration; that a caller entry
  with no `enclosing_range` and no `error` is a file-scope reference rather than a failed lookup;
  that a caller entry carrying an `error` is a file that could not be parsed; that a symbol with no
  callers exits 0 with an empty `callers` array; and that `--timeout` covers the LSP request phases
  only, never the tree-sitter parse phase.

  Add `emitImpactResult(ctx context.Context, out io.Writer, result quarry.ImpactResult, err error)`,
  reproducing `emitLookupResult`'s error routing and its order, adding no branch of its own.
  Note the branch count differs between the two helpers and the card is precise about which is
  which: `emitLookupResult` has **two** error paths — an `*quarry.ErrAmbiguousSymbol` matched via
  `errors.As` emits `candidates` through `output.Ok` and forces exit 2, and every other error
  (including `quarry.ErrSymbolNotFoundSentinel`, which it deliberately does not special-case) falls
  through to `output.Err`'s hardcoded exit 1.
  Reproduce exactly those two, in that order.
  On success, marshal the result through the existing `structToFields` helper, add
  `"resolution": "complete"` to the returned map, and emit it through `output.Ok`.
  A `structToFields` failure is itself an `output.Err` exit 1 — but not with that helper's own
  message verbatim.
  `structToFields` wraps both of its failure modes with a literal `toc: ` prefix, because it was
  written for the `toc` verbs, and the reuse decision keeps it unchanged.
  So `emitImpactResult` must re-word the failure before emitting it — prefix it as an `impact`
  marshalling failure — otherwise this verb's error envelope names a verb the caller never invoked.
  `classifyImpactError` re-words it identically, so the single-argument and batch shapes carry the
  same message.

  Add `classifyImpactError(err error, result quarry.ImpactResult) (batchStatus, map[string]any)`,
  the batch-mode counterpart. This one genuinely has **three** error branches, unlike
  `emitImpactResult`'s two, because batch mode must distinguish `not_found` from `error` in the
  status vocabulary where the single-argument shape collapses both onto exit 1.
  Preserve `classifyLookupError`'s routing and order exactly: `*quarry.ErrAmbiguousSymbol` via
  `errors.As` → `statusAmbiguous` with `candidates`; `quarry.ErrSymbolNotFoundSentinel` via
  `errors.Is` → `statusNotFound` with no extra fields; anything else → `statusError` with an `error`
  field.
  Add no fourth branch and do not reorder the three.
  The nil-error branch yields `statusFound` carrying the marshalled result and the same
  `"resolution": "complete"` marker each found entry gets today.

  Add `filterImpactWithin(result quarry.ImpactResult, within, baseDir string) quarry.ImpactResult`,
  filtering the caller list only and leaving `target` and `definition` untouched.
  It must reproduce `filterWithin`'s three normalization steps on the `within` value before any
  comparison — join onto `baseDir` when the value is relative, then `filepath.Abs`, then
  `filepath.Clean` — and only then call the existing type-agnostic `isWithinDir` helper per entry.
  That normalization is load-bearing, not ceremony: every compared path is absolute, so an
  un-normalized relative value makes `filepath.Rel` error and silently filters every caller out,
  producing an empty-but-successful answer.
  The filtered caller slice must stay non-nil even when nothing survives, so it still marshals as
  `[]`.
  Record in its doc comment that `--within` is a CLI flag with no engine option behind it, so
  `quarry.Impact` itself is unfiltered.

  Apply `filterImpactWithin` on both the single-argument and the batch path, only when the error is
  nil and the flag is non-empty, exactly as `refsCommand` applies `filterWithin`.
  Import nothing under `internal/quarryengine`: reach the engine only through the facade.
- **Commit:** `feat(cli): add the impact verb and its result-typed emit/classify helpers`

### Card 11: Register impact on the command tree and update the CLI package doc sites

- **Context:**
  - `internal/cli/impact.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `cmd.AddCommand(impactCommand())` to `Command()`, placed after the `assert-no-callers`
  registration and before the `toc` group.
  The reason is source-order readability — grouping the new LSP-backed verb with the other four,
  ahead of the tree-sitter-backed group — and nothing more.
  Do not claim this controls `--help` output: cobra's `EnableCommandSorting` defaults to true and is
  never disabled anywhere in this repo, so `--help` lists subcommands alphabetically regardless of
  registration order.

  Then make three distinct edits to this file's package doc comment, which enumerates the verb set
  three separate times — the verb list, the per-verb batch identity key, and the batch status
  vocabulary.

  First, add `impact` to the verb enumeration in the opening `Package cli builds quarry's own
  command tree` paragraph, describing it as the caller set for a symbol with each caller's enclosing
  declaration range.

  Second, add `impact` to the batch-mode identity-key sentence's symbol-keyed group, alongside
  `refs`/`definition`/`symbol`/`assert-no-callers`, leaving the `toc` verbs' path-keyed group
  unchanged.

  Third, correct the batch status-vocabulary sentence in the same package doc comment.
  It currently scopes the `"ambiguous"` status to "refs/definition only; toc never produces it".
  `classifyImpactError` makes `impact` a third producer of that status, so the parenthetical is
  false the moment this verb lands.
  Restate it to name `impact` alongside `refs` and `definition`, keeping the existing clause that
  `toc` never produces it and the reason given for that.
  This third doc edit falls under the same `doc-site-ownership-by-touching-batch` Shared Decision
  that assigns this file to this batch — it is a hand-enumerated statement about the verb set,
  exactly the category that rule covers, and it carries no "seven-package DAG" phrase to grep for.

  `impact`'s single-argument call has the full 0/1/2 outcome set, so the exit-code contract
  paragraph's existing carve-outs for `symbol` and `toc` need no new exception.

  Change nothing else in `internal/cli/cli.go` — the four existing commands, `resolveContext`,
  `buildOptions`, `runBatch`, `filterWithin`, `isWithinDir`, `emitLookupResult`, and
  `classifyLookupError` are all reused unchanged, and this task adds nothing parallel to them.

  Finally, a fourth doc edit, in a **second** file: `internal/cli/toc.go`'s header comment justifies
  `toc`'s deliberate bypass of `resolveContext` and `buildOptions` by enumerating the verbs those
  helpers exist for — "(refs/definition/symbol/assert-no-callers)".
  `impactCommand` uses both helpers, so add `impact` to that enumeration.
  Change nothing else in that file: its behaviour, its bypass, and the justification's reasoning are
  all untouched by this task — only the verb list inside the justification is stale.
  This is the doc site a verb-set grep is least likely to surface, since the file is named for a
  different verb group entirely and its comment reads as being about `toc` rather than about the
  LSP-backed verbs.
- **Commit:** `feat(cli): register the impact verb and refresh the CLI package doc sites`

### Card 12: CLI-level tests for the impact verb

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/impact.go`
  - `internal/cli/cli_test.go`
  - `internal/cli/toc_test.go`
  - `internal/cli/exec.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/impact_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Cover the pure helpers directly, following `TestEmitLookupResult_AmbiguousSymbolExitsTwo`'s
  table shape — build an exit context with `NewExitContext`, call the helper against a
  `bytes.Buffer`, decode the single-line JSON envelope, and assert on the exit code and the keys.

  For `emitImpactResult`: an `*quarry.ErrAmbiguousSymbol` yields exit 2 with `ok:true` and a
  `candidates` array; a `*quarry.ErrSymbolNotFound` yields exit 1 with `ok:false` and an `error`
  naming the symbol; any other error yields exit 1; and a successful result carries `ok:true`,
  `"resolution":"complete"`, a `target` object, a `definition` object, and a `callers` array.
  Assert as its own case that a successful result with no callers emits `callers` as an empty JSON
  array and never `null`, and that a result whose `Target` and `Definition` are absent omits both
  keys entirely rather than emitting nulls.

  For `classifyImpactError`: the same three error branches map to `statusAmbiguous`, `statusNotFound`
  (with no extra fields), and `statusError` (with an `error` field), and the nil-error branch yields
  `statusFound` carrying `"resolution":"complete"` alongside the marshalled result.

  Add the batch-key collision test, driving the existing `runBatch` with a stub closure that returns
  `classifyImpactError`'s output for a result carrying a populated target: assert each entry's
  `symbol` key still holds the query string the argument supplied, and that the identity object is
  present under `target`. This is what proves the merge-order collision the
  `identity-object-is-keyed-target-not-symbol` Shared Decision avoids.
  Add a mixed-status batch case through the same driver, asserting the worst status wins the exit
  code under `statusRank` and that every entry carries its own `symbol` and `status` keys.

  For `filterImpactWithin`: cover a **relative** `within` value against absolute caller paths as its
  own case, not only an absolute one — a relative value is exactly what an un-normalized filter
  silently turns into an empty result set. Assert in-scope callers survive, out-of-scope callers are
  dropped, `target` and `definition` are untouched, and a filter that drops everything still leaves a
  non-nil, empty caller slice.

  Drive the command itself through `RunCLIIn` for the paths that resolve before any language server
  is contacted: `impact` with no positional argument fails on `cobra.MinimumNArgs(1)`; `impact` in an
  empty temporary directory exits non-zero with a JSON error envelope naming the no-language failure;
  and two positional arguments in that same directory produce a `results` array with one entry per
  argument, each keyed on `symbol` with a `status`, and the worst-status exit code.
  Add the build-tags case as a `RunCLIIn` test too: in a temporary directory containing a
  `pyproject.toml` marker file, `impact --build-tags foo` fails with an error rather than silently
  succeeding, because language detection resolves before any spawn and Python's registry entry
  carries no build-tag template.

  Do not build a fake language server: no such seam exists in this package today, and adding one is
  out of scope for this task.
- **Commit:** `test(cli): cover the impact verb's envelope, batch, and within behaviour`

### Card 13: Correct the hand-enumerated build-tags verb table

- **Context:**
  - `internal/cli/impact.go`
- **Edits:**
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Rename `TestBuildTagsFlag_RegisteredOnAllFourVerbs` to
  `TestBuildTagsFlag_RegisteredOnAllFiveVerbs` and correct its doc comment off the "four verbs"
  wording, keeping the comment's existing statement that the check looks the flags up on the built
  command tree rather than by executing a query.

  Add an `{"impact", impactCommand()}` row to the table.
  Without it the new verb's `--build-tags` registration is untested, and — more importantly — the
  table's second loop, which asserts `--no-verify` is registered on `assert-no-callers` alone, never
  sees the new command. Adding the row is what makes the Scope "Out" claim that `impact` gains no
  `--no-verify` flag mechanically checked rather than merely asserted in prose.

  Change nothing else in this file.
- **Commit:** `test(cli): extend the build-tags verb table to cover impact`

## Batch Tests

`verify: go build ./... && go test ./internal/cli/...` covers both files this batch adds tests to.
`go build ./...` is the gate that catches a facade or engine signature drift reaching the CLI, since
the new command is otherwise only exercised through the package's own tests.

New test file: `internal/cli/impact_test.go` (card 12).
Edited test file: `internal/cli/cli_test.go` (card 13).
Card 11's second edit target, `internal/cli/toc.go`, is a comment-only change with no behavioural
surface; the package's existing `toc` tests cover that file's behaviour and must stay green, which
this same `verify:` already confirms.

What this scope deliberately does **not** prove: that a real language server resolves a symbol and
that each caller's `enclosing_range.start_line` lands on the caller's docstring line rather than its
`func` line. `internal/cli` has no fake language server, and that assertion cannot be proved against
one in any case — it is the live-tier test's job, and batch 4 adds it.
