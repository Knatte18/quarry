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
no longer permits. It closes with the completeness gate the discussion mandates — two greps plus a
mandatory re-read of all six input structs' doc comments, since three of them state the removed
override's existence purely by count and no token grep can find them.

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
  per-call override is needed. A project-scoped `.mcp.json` is launched by the client with the
  project root as its working directory, so each worktree's session gets its own server process
  rooted at its own worktree, with its own daemon state directory and its own gopls. State is keyed
  by a hash of the cleaned absolute target path, so two worktrees of the same repository never share
  a daemon, lock, socket, or language-server process. No configuration and no repointing is
  involved.
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
- **Commit:** `docs: document quarry-mcp's launch-scoped target directory and its escape hatches`

### Card 7: Drop the targetDir half of the bench ladder's prompt instruction

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/gates.py`
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
  f-string. Change nothing else in the file.
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
  - `docs/mcp-setup.md`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Run the three checks this batch's `## Batch Tests` section specifies, in order,
  and confirm each passes. This card exists because two of the three cannot be expressed as an
  automated assertion and would otherwise be skipped.
  Check one is a case-insensitive grep for the `targetdir` token across the `internal/mcpserver`
  package and `docs/mcp-setup.md`. It must return only intentional survivors: Go identifiers —
  `Config.TargetDir`, `callContext.TargetDir`, `quarry.Options.TargetDir`, `ResolveLaunchTargetDir`,
  the `targetDir` parameters of `nativeEntry.query`, `lspEntry.query`, `resolveEntryFile`, and
  `exceptSet`, and the local `targetDir` variables in `tocFileHandler` and `tocDirHandler` — plus
  prose naming the server's target directory. It must never return a schema property name, a
  `jsonschema` tag mentioning `targetDir` as a settable parameter, or per-call override phrasing.
  Check two is a grep for `effectiveTargetDir` across the same package, which must return zero hits.
  That identifier has no intentional survivors and is deliberately excluded from check one's
  identifier whitelist, which otherwise reads as if any identifier spelling passes.
  Check three is the second enumeration criterion, and it is mandatory rather than optional: re-read
  the doc comment on all six input structs — `lspInput`, `symbolInput`, `impactInput`, `assertInput`,
  `tocFileInput`, and `tocDirInput` — and confirm each still describes its struct accurately. Neither
  grep can substitute for it. Three of those comments stated the removed override's existence purely
  by count and contain neither spelling of the token, and `exceptSet`'s comment paraphrased the
  deleted helper without naming it, so a grep-only completeness check passes over all four.
  If any check fails, fix the offending comment or string in the file it lives in and re-run all
  three. Do not narrow a check to make it pass.
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

The three completeness checks card 8 runs are:

1. `grep -rni 'targetdir' internal/mcpserver/ docs/mcp-setup.md` — case-insensitive, so it matches
   `TargetDir` as well as `targetDir`. Expected: intentional survivors only, per card 8's
   enumeration.
2. `grep -rn 'effectiveTargetDir' internal/mcpserver/` — expected: zero hits, no exceptions.
3. A re-read of all six input structs' doc comments, confirming each still describes its struct
   accurately. This is not automatable and is why card 8 is a card rather than a line in this
   section.

Neither grep is sufficient alone; check 3 runs alongside them, never instead of them.
