# Batch: control-sweep-and-card

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
batch: control-sweep-and-card
number: 2
cards: 8
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestRenderPrompt|TestLoadTaskFile|TestLoadCardFile|TestMCPConfigDocument|TestCheck|TestE2E|TestPreMatrix'
depends-on: [1]
```

## Batch Scope

This batch does the two harness changes that make a three-cell, tool-less ladder letter possible:
the complete call-site sweep that switches every server-related branch from `IsControl()` to
`GrantsTools()` while leaving every comparison-baseline branch alone, and the per-config card that
gives the three arms three different prompts from one task file. It is one batch because the sweep
and the card share the same file — `run.go`'s `runCellRepetition` is both the site of the blinding
gate the sweep widens and the site of the prompt render the card extends — and because the argv
regression the sweep fixes is only observable through the same end-to-end test the card must not
disturb.

The external interface batch 4, 5 and 6 consume is: `RenderPrompt`'s fourth parameter, the exported
`LoadCardFile`, and the post-sweep behaviour that a tool-less non-control cell builds no server,
gets no `--allowedTools` argv entry, receives the empty-servers MCP document, and is still subject
to the pre-dispatch blinding check.

Batch-local decision: the sweep is verified by re-running both greps before editing, not by trusting
the enumeration written during the discussion. The eleven sites the discussion lists were confirmed
against this worktree's tip during planning, but the plan says re-run rather than assume, because a
site that appears between planning and implementation would otherwise be silently missed.

## Cards

### Card 5: Re-run the sweep greps and switch run.go's six server-related branches

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  First re-run both enumeration greps over the harness package and confirm the site list is exactly
  the eleven the plan classifies:

  ```
  grep -rn "IsControl()" bench/loomyard-eval/ladder --include=*.go | grep -v _test
  grep -rn "\.Allowed" bench/loomyard-eval/ladder --include=*.go | grep -v _test
  ```

  Any site the greps turn up that is not classified below is classified by the same question — does
  this branch depend on whether the cell has an MCP server attached, or on whether it is the
  letter's comparison baseline? — and the answer is recorded in the commit message.
  Then switch exactly six branches in `run.go`, and no others:
  1. `needsServer`'s loop condition from `!c.IsControl()` to `c.GrantsTools()` — the server is built
     only for a cell that has one;
  2. the `ServerHashes` skip inside the `needsServer` block from `c.IsControl()` to
     `!c.GrantsTools()` — a hash is recorded only for the cells the server serves;
  3. the pre-dispatch `CheckRenderedControlPrompt` guard in `runCellRepetition` from
     `cfg.IsControl()` to `!cfg.GrantsTools()` — blinding must cover all three tool-less cells;
  4. the `CheckServerConnected` guard inside the attempt loop from `!cfg.IsControl()` to
     `cfg.GrantsTools()` — only a granted cell has a server to connect;
  5. the post-run `CheckBlinding` guard from `cfg.IsControl()` to `!cfg.GrantsTools()` — same reason
     as (3);
  6. the `--allowedTools` argv append in `invokeMeasuredProcess` from `!cfg.IsControl()` to
     `cfg.GrantsTools()`.
  Leave the two comparison-baseline sites in `writeCompleteState` — the `controlForLadder`
  assignment and the `IsControl` field on the written `RunState` — reading `cfg.IsControl()`
  unchanged, so the rung-versus-control pairing keeps pairing `e1-pack` and `e2-files` against
  `e0-names`.
  Site (6) is the reason this sweep is written out rather than summarised: left as it is, `e1-pack`
  and `e2-files` would each receive two argv entries `e0-names` does not get, `--allowedTools`
  followed by an empty string, with CLI-dependent semantics for an empty allowlist — an unintended
  difference in the invocation itself, in a matrix whose whole design is that the card is the only
  difference between arms.
  Update the two inline comments that name the old predicate: the one above site (4), which says a
  control cell is never this check's concern, and the one above site (5), which says the checks are
  gated by `IsControl` at their call site. Both now say "a cell that grants no tools".
- **Commit:** `fix(ladder): gate server build, connect check and allowedTools on GrantsTools`

### Card 6: Switch the MCP document's empty-servers branch

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Switch `MCPConfigDocument`'s first branch from `cfg.IsControl()` to `!cfg.GrantsTools()`, so a
  cell that grants no tools gets the empty-servers document regardless of whether it is its letter's
  control.
  This is not a confound but a crash: with `l.Server` nil — which is what a ladder file with no
  `server:` block has — a tool-less non-control cell reaching the granted branch returns
  `mcp config for cell e1-pack: ladder file declares no server block` and the run dies before rep 1.
  Reword `MCPConfigDocument`'s doc comment, which currently defines the first branch as "a control —
  a config whose allowed list is empty": it is now "a cell that grants no tools". The granted
  branch's description and the `{target_dir}` substitution contract are unchanged.
  Confirm while editing that the scorer's own call, which passes a zero-valued `Config`, still takes
  the empty-servers branch: a zero `Config` has a nil `Control` and an empty `Allowed`, so
  `GrantsTools` is false. State that confirmation in the commit message; no code change follows from
  it.
- **Commit:** `fix(ladder): give any tool-less cell the empty-servers mcp document`

### Card 7: Reword the three stale gate doc comments

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Three doc comments in `gates.go` describe a world in which "control" and "grants no tools" are the
  same thing. Reword them; attach no behaviour change to any of the three.
  1. `CheckGrantedToolUsed`'s comment says it "returns nil for a control cell (an empty allowed
     list)". Its own test is `len(cfg.Allowed) == 0`, which already means *grants no tools* and is
     already correct under the split — so the code stays exactly as it is and only the wording
     changes, to "a cell that grants no tools". Leave the `len(cfg.Allowed) == 0` test spelled
     inline rather than routing it through `GrantsTools`: `summarizeCell` reconstructs a `Config`
     from a results root's own run state, which carries `allowed` but has no `control` field to
     carry, so the reconstructed value's `Allowed` is the only field this check may depend on.
     Record that reasoning in the comment, so a later reader does not "tidy" it into
     `!cfg.GrantsTools()` and quietly make the reconstruction load-bearing on a field it cannot
     populate.
  2. `CheckServerConnected`'s comment says a control cell's allowed list is empty and that this
     matches how the blinding checks are gated by `IsControl` at the call site. Both halves are now
     wrong; say instead that the caller gates this check on a cell that grants tools.
  3. `CheckBlinding`'s comment says it "applies per rep, only for a control cell (an empty allowed
     list)". Say instead that it applies to a cell that grants no tools, and that the gating lives at
     the call site.
- **Commit:** `docs(ladder): reword gate comments for the control/grants-tools split`

### Card 8: Render a per-config card into the prompt

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/live_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a fourth parameter `card string` to `RenderPrompt`, after `toolNames`. When `card` is the
  empty string the section list is exactly the six sections it is today, so a config with no card
  renders byte for byte what it renders now. When `card` is non-empty it is inserted as one more
  section immediately after `target.TaskText` and before `PARALLEL_BLOCK`. Update `RenderPrompt`'s
  doc comment to name the new order and to state the empty-card guarantee.
  Add an exported `LoadCardFile(path string) (string, error)` to `prompt.go`, alongside
  `LoadTaskFile`: it reads the file whole, returns its text with trailing newlines trimmed, and
  wraps a read failure as `load card file %s: %w`. Unlike `LoadTaskFile` it extracts nothing — a
  card is prompt text in its entirety, and an extractor would be a second place the card's shape is
  defined.
  In `runCellRepetition`, after the existing `LoadTaskFile` call, load the card when `cfg.Card` is
  non-empty, resolving it with the existing repository-relative resolver exactly as `task.TaskFile`
  is resolved, and pass the result to `RenderPrompt`. An empty `cfg.Card` passes the empty string
  and loads nothing.
  Update the four existing call sites so the package compiles: the one in `run.go` above, and the
  ones in `prompt_test.go`, `gates_test.go` and `prematrix_test.go` and `live_test.go`. Each of the
  test call sites passes `""` unless that test is specifically about a card.
  This card changes an exported function's signature, so the compile break is the point: every
  caller is updated in this same commit rather than left for the next batch.
- **Commit:** `feat(ladder): render an optional per-config card into the prompt`

### Card 9: Test card rendering and the byte-identical no-card path

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/prompt_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `TestRenderPrompt_CardLandsAfterTaskTextBeforeParallelBlock`, asserting the card text appears
  after the task text and before the parallel block, using the same index-ordering technique the
  existing section-order test uses.
  Add `TestRenderPrompt_NoCardIsByteIdenticalToTodaysOutput`, which renders the same
  `TaskContent`, target directory and tool list twice — once with `""` — and compares the result
  against a golden string held in the test. This is the regression that makes the
  backwards-compatibility claim checkable rather than asserted: every config in the real
  `ladder-toc.yaml` omits `card`, and every previous results root was produced by the no-card
  rendering.
  Add `TestLoadCardFile`, covering a card read back with trailing newlines trimmed, and a missing
  path returning an error naming the file.
- **Commit:** `test(ladder): cover card rendering and the unchanged no-card prompt`

### Card 10: Test the empty-servers document for a tool-less non-control cell

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/mcp_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `TestMCPConfigDocument_ToolLessNonControl`, driving a `Ladder` whose `Server` is nil and a
  `Config` with an empty `Allowed` and an explicit `control: false`, and asserting the returned
  document is the empty-servers document and that no error is returned. This is the regression for
  the crash card 6 fixes: before the switch this input returns the "declares no server block" error
  and kills the run before rep 1.
  Leave the existing granted-cell and no-server-block-errors tests asserting exactly what they
  assert today — a cell that grants tools with no server block is still an error, and that must not
  become reachable-only-by-accident.
- **Commit:** `test(ladder): cover the empty-servers document for a tool-less rung`

### Card 11: Test that blinding still fires for a tool-less non-control cell

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  The existing test at the foot of the granted-tool-used test asserts that a `Config` with a nil
  `Allowed` reports `IsControl` true, documenting that the gating lives at the call site. That
  assertion is now the wrong shape: under the split, an explicit `control: false` makes the same
  config not a control while leaving it tool-less.
  Replace it with an assertion over both predicates: a `Config` with a nil `Allowed` and an explicit
  `control: false` reports `IsControl` false and `GrantsTools` false, and therefore still satisfies
  the `!GrantsTools()` condition the blinding call sites now use. Keep the surrounding comment's
  intent — that the checks are gated at the call site, not inside the check — and update its wording
  to name the new predicate.
  This is the case that would silently stop being checked if the blinding switch had been made on
  `IsControl()` instead: two of the three cells in the new matrix are non-control, and an unblinded
  rung would leak the target repository's origin into a measured transcript with nothing to catch it.
- **Commit:** `test(ladder): assert blinding gating covers a tool-less non-control cell`

### Card 12: Test the argv regression end to end

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a subtest to `TestE2E` — name it `ToolLessNonControlGetsNoAllowedTools` — driving a synthetic
  ladder file with two configs under one ladder letter, both with an empty `allowed`, one with
  `control: true` and one with `control: false`, at one repetition each.
  Drive it with the existing all-control environment helper, not the granted-cell one: that helper
  sets the fake binary's control-mode variable, under which the fake asserts `--allowedTools` is
  absent from the measured invocation. The fake's variable is named for the control concept but its
  assertion is about the flag's absence, which is exactly the property under test here; say so in the
  subtest's own comment so the mismatch between the variable's name and its meaning is not read as a
  mistake.
  The subtest must reach a completed repetition for both cells, so the run also proves the
  tool-less non-control cell builds no server, receives the empty-servers document and passes its
  pre-dispatch blinding check — the three other halves of the sweep, none of which have a cheaper
  observation point than a real end-to-end run.
  Add one further subtest, `TwoRungsPairAgainstOneControl`, that summarises the same results root
  and asserts two comparison rows are built, both naming the control cell. This is the
  `summarize` half of the split: the rung-versus-control pairing must keep working when a letter
  carries one control and two rungs.
- **Commit:** `test(ladder): cover tool-less non-control dispatch and two-rung pairing`

## Batch Tests

`verify:` runs `go test` against `./bench/loomyard-eval/ladder/internal/ladder/` with
`-run 'TestRenderPrompt|TestLoadTaskFile|TestLoadCardFile|TestMCPConfigDocument|TestCheck|TestE2E|TestPreMatrix'`.

That pattern is deliberately broader than the tests this batch adds, because the sweep touches
shared machinery: `TestCheck` covers every gate test in `gates_test.go`, including the ones this
batch does not edit but whose call-site contract it changes; `TestE2E` is the only place the argv,
server-build, MCP-document and blinding halves of the sweep are observable together; and
`TestPreMatrix` re-renders every control prompt in the real `ladder-toc.yaml` through the new
four-parameter `RenderPrompt`, which is the backwards-compatibility check for the signature change.
`TestLoadTaskFile` and `TestRenderPrompt` cover the prompt half.

The module-wide `go build ./...` at the batch boundary is load-bearing here: `RenderPrompt` is
exported and has five call sites across three test files and one production file, so a missed one is
a compile error rather than a silent behaviour change.
