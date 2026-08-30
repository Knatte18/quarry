# Batch: docs-and-bench-alignment

```yaml
task: "Rethink quarry-mcp's per-call targetDir ergonomics"
batch: "docs-and-bench-alignment"
number: 2
cards: 3
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests -q
depends-on: [1]
```

## Batch Scope

This batch aligns everything outside the Go package with the behaviour batch 1 established: the
setup documentation gains the scoping contract and the honest statement of what the removal costs,
and the bench ladder stops spending prompt tokens telling the model not to set a property the schema
no longer permits. It closes with the completeness gate the discussion mandates — three greps plus a
mandatory re-read of all six input structs' doc comments and `exceptSet`'s, since three of the
structs state the removed override's existence purely by count and `exceptSet` paraphrases the
deleted helper without naming it, so no token grep can find any of the four.

It is a separate batch from the Go work because it shares almost no `Context:` with it and cannot be
written correctly until batch 1's behaviour is real: the documentation describes what the tools now
do, and the bench prompt edit is only honest once the property is gone. It depends on batch 1 for
exactly that reason.

Batch-local decision: this batch touches `bench/loomyard-eval/ladder/`, one of the two sanctioned
Python exceptions `CLAUDE.md` names. Per the overview's `python-exception-is-edited-not-extended`
decision, it edits two existing files and creates no new Python anywhere.

## Cards

### Card 6: Document the launch-scoped contract and the escape hatches in mcp-setup.md

- **Context:**
  - `.mcp.json`
  - `cmd/quarry-mcp/main.go`
  - `internal/mcpserver/mcpserver.go`
  - `internal/mcpserver/tools_toc.go`
  - `internal/cli/paths.go`
- **Edits:**
  - `docs/mcp-setup.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a new section to `docs/mcp-setup.md` covering the server's scoping contract,
  placed after the existing `## What the committed .mcp.json does` section and before
  `## Cold-start behaviour`. It must state four things.
  First: the server's target directory is fixed once, at launch, by `--target-dir` or by the process
  working directory, and there is no per-call way to change it — no tool accepts a target directory
  as an input property. Name `ResolveLaunchTargetDir` as the function that resolves it and
  `NewServer`'s absolute-path guard as what enforces it.
  Second: cwd inheritance is what makes per-project scoping automatic, and this is the reason no
  per-call override is needed. Where a client launches a project-scoped server with the project root
  as the server process's working directory — which is what the committed argument-free `.mcp.json`
  relies on — each worktree's session gets its own server process rooted at its own worktree, with
  its own daemon state directory and its own gopls, needing no configuration and no repointing.
  Neither of the two claims in that paragraph may be written as an unconditional absolute, and each
  is qualified differently.
  The cwd claim is client behaviour this repository does not control and cannot verify: `.mcp.json`
  carries only `command` and `args`, `cmd/quarry-mcp/main.go` only calls `os.Getwd` through
  `ResolveLaunchTargetDir`, and the existing `docs/mcp-setup.md` already hedges the same point where
  it notes the effective default is visible on stderr even though `.mcp.json` never states it
  explicitly. Scope it to clients that launch a project-scoped server from the project root, and name
  the `quarry-mcp: resolved target directory <path>` stderr line as how an operator confirms it for a
  given session rather than assuming it. Do not present it as a universal client guarantee — it is
  the entire justification for removing the per-call override, so overstating it is the one claim in
  this document that most needs to be exact.
  The state-keying claim is qualified on different grounds: `cli.ResolveStateDir`
  (`internal/cli/paths.go`) reaches `workspaceKey` only in its default user-cache tier, and an
  explicit `--state-dir` or a set `$QUARRY_STATE_DIR` becomes the leaf verbatim with the target path
  never entering the key. Write the automatic-isolation guarantee as holding for the default tier,
  and name `--state-dir` and `$QUARRY_STATE_DIR` as the exception where two servers pinned to one
  explicit state directory do share it — the `## Launch-only flags` section this card leaves in place
  still documents `--state-dir`, so an unqualified absolute here would contradict the same file two
  sections down. Read `internal/cli/paths.go` to confirm the tier precedence and the digest's inputs
  before writing this paragraph; both `workspaceKey` and `ResolveStateDir` are defined there.
  Third: the escape hatch for a genuinely cross-repository or cross-worktree query is a second named
  server entry in the client's own MCP configuration, with an explicit `--target-dir` pointing at the
  other root. Show it as a short JSON snippet in the same shape as the committed `.mcp.json`, with a
  second key alongside `quarry`. State plainly that this is the one capability the design costs: a
  session rooted at one repository cannot ask about another through the same server entry.
  Fourth, and it must not be omitted: `toc_file` and `toc_dir` retain a partial escape the five
  language-server-backed tools do not. Those two accept an absolute target path, which is resolved
  as given rather than against the launch root, so they can read a file or directory outside it. The
  five language-server-backed tools cannot: even with an absolute file or URI, the query is served by
  the gopls rooted at the server's target directory, so a path outside that root will not resolve
  correctly. Do not imply absolute paths are a general escape hatch — the asymmetry is the point of
  this paragraph.
  Also update the existing `## Launch-only flags` section's `--target-dir` bullet to state that it is
  the only way to set the target directory, since no per-call property overrides it any more. Leave
  the `--config`, `--state-dir`, and `--timeout` bullets, the cold-start section, the
  missing-toolchain section, and the warm-start section unchanged. Follow the file's existing
  one-clause-per-line prose style.
  Also extend the document's opening summary — the sentence beginning `This document covers` and the
  clause list under it — with a clause naming the new scoping-contract section, so the enumeration
  still describes what the file covers. Extending it is the deliberate choice here rather than
  leaving it alone: the new section is a top-level one about the server's core contract, unlike
  `## Launch-only flags`, which the summary already omits as reference material. Do not renumber or
  reword the summary's existing clauses.
- **Commit:** `docs: document quarry-mcp's launch-scoped target directory and its escape hatches`

### Card 7: Drop the targetDir half of the bench ladder's prompt instruction

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `_mill/discussion.md`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `_mcp_preamble_body` in
  `bench/loomyard-eval/ladder/scripts/ladder_config.py`, replace the two-line prompt sentence inside
  the returned f-string. Its first line is currently
  `Never set targetDir or buildTags on any of these calls -- the server is` and its second line is
  `already rooted at the correct target codebase.` Replace them with the first line
  `Never set buildTags on any of these calls -- the default build-tag set is` and the second line
  `the one this run is scoped to.` The rationale clause must change together with the key list, not
  just the key list: "the server is already rooted at the correct target codebase" justifies only the
  removed half, and leaving it attached to a `buildTags`-only instruction would be a non-sequitur.
  Keep the two-line wrap and the surrounding blank lines exactly as they are so the prompt block's
  shape is unchanged, and keep both lines at the same column the originals sit at inside the
  f-string. Change nothing else in the file. Use these two replacement lines verbatim: they are the
  exact text `_mill/discussion.md`'s Scope section prescribes, and this prompt is measured benchmark
  input, so its wording is not a free choice at implementation time.
  The replacement rationale is deliberately worded for the model being benchmarked rather than
  copied from `gate_no_target_override`'s own docstring, which frames the same constraint as the
  pinned-worktree constraint and the cold cell's daemon key. Both statements describe the same fact —
  the run is scoped to one pinned build-tag set, and departing from it changes the daemon key — but
  the prompt's audience is a model deciding whether to set a parameter, while the docstring's is a
  bench maintainer reading a fatal gate. Do not import the gate's phrasing into the prompt; the
  divergence is intended, not an oversight.
  In `bench/loomyard-eval/ladder/tests/test_ladder_config.py`, narrow the assertion in
  `test_mcp_preamble_forbids_binary_paths_and_cli_syntax` from
  `assert "Never set targetDir or buildTags" in prompt` to
  `assert "Never set buildTags on any of these calls" in prompt`. Keep it a literal substring
  assertion — the test exists so the instruction survives prompt refactors, and that value holds only
  as long as the literal tracks the prompt. The new literal is short enough to sit on one source line
  despite the prompt's two-line wrap. Leave the module-level `_FORBIDDEN_LITERALS` tuple unchanged;
  the replacement prompt text contains none of its entries. Leave
  `bench/loomyard-eval/ladder/scripts/gates.py` unmodified: `gate_no_target_override` keeps checking
  both keys with `fatal=True`, because it still guards the constraint it was written for — a run must
  not retarget away from its pinned worktree, which stays fully reachable through `buildTags` — and
  its `targetDir` arm becomes close to unreachable rather than redundant. Note for the reviewer that
  this gate is not, and never was, a check that the property left the published schema: it inspects
  transcript `tool_input` maps only, so it would pass unchanged with the property still in place. The
  schema-level guarantee is batch 1's card 1, and neither substitutes for the other.
- **Commit:** `chore(bench): drop the stale targetDir half of the ladder's prompt instruction`

### Card 8: Completeness gate — greps plus the mandatory input-struct doc-comment re-read

- **Context:**
  - `internal/mcpserver/callcontext.go`
  - `internal/mcpserver/translate.go`
  - `internal/mcpserver/mcpserver.go`
  - `internal/mcpserver/nativeentry.go`
  - `internal/mcpserver/lspentry.go`
  - `internal/mcpserver/tools_lsp.go`
  - `internal/mcpserver/tools_symbol.go`
  - `internal/mcpserver/tools_impact.go`
  - `internal/mcpserver/tools_assert.go`
  - `internal/mcpserver/tools_toc.go`
  - `internal/mcpserver/callcontext_test.go`
  - `internal/mcpserver/layering_test.go`
  - `internal/mcpserver/lspentry_test.go`
  - `internal/mcpserver/nativeentry_test.go`
  - `internal/mcpserver/result_test.go`
  - `internal/mcpserver/schema_test.go`
  - `internal/mcpserver/stdio_lsp_test.go`
  - `internal/mcpserver/tocentry_test.go`
  - `internal/mcpserver/tools_assert_test.go`
  - `internal/mcpserver/tools_impact_test.go`
  - `internal/mcpserver/tools_lsp_test.go`
  - `internal/mcpserver/tools_toc_test.go`
  - `internal/mcpserver/transport_errors_test.go`
  - `internal/mcpserver/transport_test.go`
  - `internal/mcpserver/translate_test.go`
  - `docs/mcp-setup.md`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** This card is purely diagnostic and produces no diff. Run the four checks this
  batch's `## Batch Tests` section specifies, in order, and confirm each passes. The card exists
  because two of the four cannot be expressed as an automated assertion and would otherwise be
  skipped.
  Check one is a case-insensitive grep for the `targetdir` token across the `internal/mcpserver`
  package's **production** files only — every `.go` file except `_test.go` files — and across
  `docs/mcp-setup.md`. It must return only intentional survivors: Go identifiers —
  `Config.TargetDir`, `callContext.TargetDir`, `quarry.Options.TargetDir`, `ResolveLaunchTargetDir`,
  the `targetDir` parameters of `nativeEntry.query`, `lspEntry.query`, `resolveEntryFile`,
  `exceptSet`, `resolveTOCFileEntry`, and `resolveTOCDirEntry`, and the local `targetDir` variables
  in `tocFileHandler` and `tocDirHandler` — plus prose naming the server's target directory. It must never return a schema property name, a
  `jsonschema` tag mentioning `targetDir` as a settable parameter, or per-call override phrasing.
  Check one-b is the same case-insensitive grep restricted to this package's `_test.go` files, run
  as a separate pass with its own whitelist because the production whitelist above does not sanction
  what legitimately appears in tests. Its intentional survivors are: `Config{TargetDir: ...}` struct
  construction and `cfg.TargetDir` field reads used to root a fixture; `opts.TargetDir` reads inside
  a facade stub; `ResolveLaunchTargetDir` calls; test-function names carrying the token, which today
  are `TestExceptSet_ResolvesAgainstTargetDirNotProcessCwd`,
  `TestCallTool_TargetDirIsAbsoluteEvenFromRelativeProcessCwd`,
  `TestAssertHandler_RelativeExceptResolvesAgainstTargetDir`, and
  `TestCallTool_AssertNoCallers_RelativeExceptResolvesAgainstTargetDir`, plus the two batch 1 adds,
  `TestResolveCall_TargetDirIsAlwaysConfigTargetDir` and
  `TestCallTool_TargetDirIsRejectedAsWholeCallError`; test-local identifiers, which today are
  `launchTargetDir` and `gotTargetDir` in `transport_errors_test.go`, the `targetDir` local in
  `TestExceptSet_ResolvesAgainstTargetDirNotProcessCwd` in `nativeentry_test.go`, and the `targetDir`
  table field in `translate_test.go`; and prose in test doc comments. What it must never return is a `targetDir`
  key set inside a JSON arguments literal or an input-struct literal — with exactly one sanctioned
  exception, the deliberately-rejected `{"targets":[{"symbol":"S"}],"targetDir":"/somewhere/else"}`
  literal batch 1's card 1 adds, whose whole purpose is to be refused.
  Check two is a grep for `effectiveTargetDir` across the whole package including tests, which must
  return zero hits. That identifier has no intentional survivors and is deliberately excluded from
  both whitelists above, which otherwise read as if any identifier spelling passes.
  Check three is the second enumeration criterion, and it is mandatory rather than optional: re-read
  the doc comment on all six input structs — `lspInput`, `symbolInput`, `impactInput`, `assertInput`,
  `tocFileInput`, and `tocDirInput` — and on `exceptSet`, and confirm each still describes its
  subject accurately. `exceptSet` belongs on this list for the same reason the six structs do: its
  comment paraphrased the deleted helper as `the effective absolute target directory` without naming
  it, so no grep for either spelling of the token would have surfaced it. Three of the six struct
  comments stated the removed override's existence purely by count and contain neither spelling
  either. No grep can substitute for this check.
  Do not narrow, scope down, or skip a check to make it pass. If any check fails, do not patch the
  offending file from inside this card — this card declares no `Edits:` and makes no commit, and a
  silent fix here would land uncommitted, unreviewed, and outside any card's declared file set.
  Report the failure instead, naming the file, the line, and the offending text, so the batch fails
  visibly and the card that owns that file is corrected. All four checks are expected to pass on
  arrival, because cards 3, 4, 5, and 6 already did the work they verify.
- **Commit:** none

## Batch Tests

`verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests -q`
— the ladder pytest suite, run from the repository root with no `PYTHONPATH` prefix, exactly the
invocation `bench/loomyard-eval/ladder/README.md`'s own "How to test" section documents. `conftest.py`
handles `sys.path` itself, so a prefix is neither needed nor correct here. This suite is the only
runnable test surface this batch touches: card 6 is documentation with no executable assertions, and
card 7's single behavioural change is precisely what one assertion in
`bench/loomyard-eval/ladder/tests/test_ladder_config.py` pins. The suite was green (208 passed)
against this worktree's tip before the plan was written, so a failure after card 7 is card 7's own.

Two assertions in that suite are load-bearing for this batch and must both still pass:
`test_mcp_preamble_forbids_binary_paths_and_cli_syntax`, whose literal card 7 narrows and whose
`_FORBIDDEN_LITERALS` loop confirms the replacement text introduces no forbidden CLI syntax; and
`test_gates.py`'s assertion that `"targetDir"` appears in the gate's finding messages, which is
unaffected because the gate keeps both arms.

The four completeness checks card 8 runs are:

1. `grep -rni --include='*.go' --exclude='*_test.go' 'targetdir' internal/mcpserver/` followed by
   `grep -ni 'targetdir' docs/mcp-setup.md` — case-insensitive, so both match `TargetDir` as well as
   `targetDir`. Expected: production-side intentional survivors only, per card 8's first
   enumeration. Test files are deliberately excluded here and handled by check 1b, because the
   production whitelist does not sanction what legitimately appears in tests.
2. `grep -rni --include='*_test.go' 'targetdir' internal/mcpserver/` — expected: test-side
   intentional survivors only, per card 8's second enumeration, and no `targetDir` key set in a JSON
   arguments literal or an input-struct literal except the one deliberately-rejected literal batch
   1's card 1 adds.
3. `grep -rn 'effectiveTargetDir' internal/mcpserver/` — expected: zero hits, no exceptions, tests
   included.
4. A re-read of all six input structs' doc comments and `exceptSet`'s, confirming each still
   describes its subject accurately. This is not automatable and is why card 8 is a card rather than
   a line in this section.

No grep is sufficient alone; check 4 runs alongside them, never instead of them. Card 8 is
diagnostic only — it declares no `Edits:` and carries `Commit: none`, so a failing check is reported
as a batch failure rather than patched in place.
