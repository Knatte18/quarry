# Batch: m2-invalidation-reason-file

```yaml
task: "Ladder breadth (M1)"
batch: "m2-invalidation-reason-file"
number: 1
cards: 5
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/
depends-on: []
```

## Batch Scope

This batch delivers M2 in full: a single `invalid_reason.txt` written into every discarded attempt
directory immediately before `InvalidateRep` renames it away, on all four paths that invalidate an
attempt, replacing today's `server_connect_failure.txt` which covers only one of them. It is one
batch because the constant, the renderer, the call-site classification, the fixture variant that
makes the fourth cause reachable offline, and the tests that prove all of it are a single
change — splitting them would leave a commit where the constant exists with no writer, or a writer
with no proof it is called on the paths that actually invalidate.

The external interface batch 5 consumes is the on-disk file itself: after this batch, a reader of a
`<rep>.invalid-<k>/` directory finds `invalid_reason.txt` carrying `cell`, `repetition`, `attempt`,
`cause`, an optional `exit_code`, and a constructed single-line `detail`. Batch 5's conclusion reads
`cause` values and counts out of those files for its coverage section.

Batch-local decisions that differ from the overview's `## Shared Decisions`:

- **The writer lives in `runstate.go`, not `run.go`.** `runstate.go`'s own header states it declares
  the on-disk contract for one repetition, and the reason file is exactly that. `run.go` keeps the
  classification, which is a property of the attempt loop rather than of the on-disk format. This
  also makes the renderer unit-testable in `runstate_test.go` without driving the whole loop.
- **Cards are ordered assertions-first where the fixture allows it.** Card 4 writes e2e assertions
  that fail until card 5 wires the loop. That is deliberate — the discussion names M2 as the TDD
  candidate. Cards 1–3 are ordered ahead of it only because card 4's assertions cannot compile
  without the constant, the renderer and the new fixture variant existing first.
- **`ServerConnectFailureFile` is deleted in card 5, not card 1.** Deleting it in card 1 would leave
  `run.go` referring to a constant that no longer exists at that card's own commit. Card 5 removes
  the constant, its last Go reference and `writeServerConnectFailure` together.

## Cards

### Card 1: the reason-file constant, record, renderer and writer

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add to `runstate.go`, alongside the existing `ServerConnectFailureFile` constant which this card
  leaves in place:

  - `InvalidReasonFile`, a string constant with the value `"invalid_reason.txt"`. Document it the
    way `ServerConnectFailureFile` is documented today: it is not one of the six per-repetition file
    names, it exists only inside a discarded attempt directory, and it is written before
    `InvalidateRep` renames that directory away.
  - Four cause constants, each a string: `CauseRunnerError` = `"runner_error"`, `CauseResultError`
    = `"result_error"`, `CauseUnparseableAnswer` = `"unparseable_answer"`,
    `CauseServerNotConnected` = `"server_not_connected"`. These four are the whole enumeration; no
    fifth value is ever written.
  - `InvalidReason`, a struct with exported fields `Cell string`, `Repetition int`, `Attempt int`,
    `Cause string`, `Detail string`, and `ExitCode *int`. `ExitCode` is a pointer so an
    unrecoverable exit status is expressed as absence rather than as a sentinel value.
    Document on the `Attempt` field that it is the attempt's index within the invocation that
    produced it, that it is deliberately not the directory's `.invalid-<n>` suffix, and that the two
    coincide only on a fresh repetition directory.
  - `invalidReasonDetailMaxLen`, an unexported int constant with the value `200`.
  - `sanitizeDetail(detail string) string`, unexported: replaces every `\r` and `\n` in `detail`
    with a single space, collapses runs of whitespace to one space, and trims leading and trailing
    space. The length bound is measured in **bytes**, as `len(s)`, not in runes: when the trimmed
    result's `len` exceeds `invalidReasonDetailMaxLen`, truncate it to at most
    `invalidReasonDetailMaxLen - 3` bytes **without splitting a UTF-8 rune** and append the
    three-byte ASCII ellipsis `...`, so `len` of the returned string is never greater than
    `invalidReasonDetailMaxLen`. Use the ASCII `...` rather than the single-character `…`, which is
    three bytes and would make the bound read differently depending on whether a reader thinks in
    bytes or runes — the ambiguity this wording exists to remove.

    `runstate.go` does not import `strings` today; this function needs it, so add it to the import
    block alongside the existing `encoding/json`, `fmt`, `os` and `path/filepath`.
  - `RenderInvalidReason(r InvalidReason) string`: returns the file's text, one `key: value` pair
    per line, each line newline-terminated, in exactly this order — `cell`, `repetition`, `attempt`,
    `cause`, `exit_code`, `detail`. The `exit_code` line is emitted only when `r.ExitCode` is
    non-nil and is omitted entirely otherwise. `detail` is passed through `sanitizeDetail` before it
    is written.
  - `WriteInvalidReason(dir string, r InvalidReason) error`: writes `RenderInvalidReason(r)` to
    `filepath.Join(dir, InvalidReasonFile)` with mode `0o644`, wrapping a write failure as
    `fmt.Errorf("write invalid reason %s: %w", path, err)` — the same shape
    `writeServerConnectFailure` uses today.

  Read `gates.go` only to confirm the `Finding` type's `Message` field, which card 5 supplies as the
  `Detail` for the `server_not_connected` cause. Do not change `gates.go`.
- **Commit:** `feat(ladder): add invalid_reason.txt record, renderer and writer`

### Card 2: unit test for the renderer

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `TestRenderInvalidReason` to `runstate_test.go`, covering the renderer as a pure function:

  - A case with a recoverable exit code asserts the rendered text contains an `exit_code: 1` line
    and that all six keys appear in the documented order.
  - A case with `ExitCode` nil asserts no line beginning `exit_code:` appears anywhere in the
    output, and that the other five keys are still present in order.
  - A case whose `Detail` carries embedded newlines asserts the rendered text has exactly one line
    per emitted key — i.e. the multi-line detail did not break the one-pair-per-line shape.
  - A case whose `Detail` is longer than `invalidReasonDetailMaxLen` asserts the rendered `detail`
    value's `len` — its **byte** length, matching card 1's stated bound — is at most
    `invalidReasonDetailMaxLen`, and that it ends with the ASCII `...`. Assert against `len`, not
    against a rune count, so the assertion cannot pass or fail depending on which reading the
    implementer picked.
  - A case whose `Detail` is over-length **and built from multi-byte runes** asserts
    `utf8.ValidString` on the rendered `detail` value. The two assertions above are both satisfied by
    a naive byte slice that splits a rune mid-sequence, so without this case card 1's one non-obvious
    requirement — truncate without splitting a UTF-8 rune — ships unverified. This case needs
    `unicode/utf8` in `runstate_test.go`'s import block.

  Do not add a test for `WriteInvalidReason`'s file I/O here — the e2e cases in card 4 prove the
  file reaches disk on the paths that matter. `TestInvalidateRep` is not touched: the rename
  semantics are unchanged by this batch and its existing three-attempt suffix assertions must stay
  green exactly as they are.
- **Commit:** `test(ladder): unit-test the invalid-reason renderer`

### Card 3: a fixture variant that produces a failing result record

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a new `FAKE_CLAUDE_STREAM` variant, `result_error`, as a new `case` in `writeCellStream`'s
  `switch`. The switch carries five variants today — `normal`, `max_turns`, `no_fence`,
  `leak_prefix`, `partial_fail` — so this one is the sixth. It writes one assistant record via
  `writeAssistant` and then
  `writeResult(w, "error_during_execution", "end_turn", true)`, and returns normally so the process
  exits zero.

  The new case must be added to the `switch` rather than reached through its `default` branch: the
  `default` branch calls `fail` on an unrecognised value by design, and that loud-failure behaviour
  for a genuinely unknown variant is preserved unchanged.

  This variant exists because `result_error` is otherwise unreachable offline. Every existing
  variant calls `writeResult` with `isError=false`, and the one variant that omits a result record
  entirely (`partial_fail`) exits 1 — which the ordered classification in card 5 attributes to
  `runner_error`, not `result_error`. Without this variant one of the four enumerated causes would
  ship unproven.

  Read `stream.go` to confirm the field names the transcript parser reads off a result record
  (`terminal_reason`, `stop_reason`, `is_error`) so the new case's `writeResult` arguments land in
  the right positions. Do not change `stream.go`.
- **Commit:** `test(ladder): add a result_error stream variant to the fakeclaude fixture`

### Card 4: e2e assertions for all four causes, re-entry and the happy path

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  These assertions are written before card 5 wires the loop and are expected to fail until it does.
  Every new subtest is a `t.Run` inside the existing `TestE2E`, built with the existing helpers
  `newE2EEnv`, `baseLadder`, `writeSyntheticLadderFile`, `setFakeClaudeEnv`,
  `setFakeClaudeEnvGranted`, `runOpts` and `summarizeAndWriteReport`.

  - `InvalidReasonRunnerError`: a single control cell at `reps: 1` under
    `FAKE_CLAUDE_STREAM=partial_fail`. Assert the repetition directory itself is gone, and that for
    each `k` in `1..MaxAttempts` the directory `<repdir>.invalid-<k>` carries `InvalidReasonFile`
    whose text names the cell id, the repetition number, `cause: runner_error`, `exit_code: 1`, and
    `attempt: <k>`.
  - `InvalidReasonUnparseableAnswer`: the same shape under `FAKE_CLAUDE_STREAM=no_fence`. Assert
    `cause: unparseable_answer`, `attempt: <k>` per directory, and that no line beginning
    `exit_code:` appears — the fixture exits zero on this path.
  - `InvalidReasonResultError`: the same shape under the new `FAKE_CLAUDE_STREAM=result_error`
    variant from card 3. Assert `cause: result_error`, `attempt: <k>` per directory, and no
    `exit_code:` line.
  - `InvalidReasonReEntry`: run a `partial_fail` matrix to exhaustion over a fresh root, then run a
    second `Run` over the **same** results root with the same options and the same stream. Assert
    the second invocation produced `<repdir>.invalid-4` and that its `invalid_reason.txt` carries
    `attempt: 1`, not `attempt: 4`. This is the `m2-attempt-numbering` divergence pinned rather than
    assumed; every other case above asserts the equality only because its root starts empty.
  - `GrantedCellServerNeverConnects` (existing subtest, retargeted): change `ServerConnectFailureFile`
    to `InvalidReasonFile` in its `reasonPath` construction, keep its existing assertions that the
    file names the cell id and the server name, and add an assertion that the file carries
    `cause: server_not_connected`. Its `summary.Incomplete` assertion is unchanged — that assertion
    is what keeps the `connectFailures == attempts` whole-run abort covered, and this batch must not
    change that behaviour.
  - `HappyPath` (existing subtest): add an assertion that `InvalidReasonFile` does **not** exist in
    the completed repetition's own directory. The reason file belongs only inside a discarded
    attempt directory, never inside a repetition that completed.

  Add one unexported helper to this file, `readInvalidReason(t *testing.T, dir string) string`, that
  reads `filepath.Join(dir, InvalidReasonFile)` and fails the test with the path on a read error, so
  the four new subtests do not each re-spell that. Keep the assertions substring-based against the
  rendered text, matching how the existing `GrantedCellServerNeverConnects` case already checks its
  reason file.
- **Commit:** `test(ladder): e2e assertions for every invalidation cause and re-entry`

### Card 5: classify the cause at the attempt loop and write the file

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `run.go`'s `runCellRepetition`, in the measured-invocation attempt loop that today begins
  `connectFailures := 0` and rejects an attempt by calling `writeServerConnectFailure` then
  `InvalidateRep`:

  - Declare a loop-local `attempt := 0` next to `connectFailures := 0`, before the `for {`.
  - Hoist the answer-extraction outcome so it is readable after the accept decision: in the branch
    that today calls `ExtractFencedJSON(text, "last")` and then `json.Unmarshal`, record which of
    the two steps failed into a loop-body-scoped string variable — one fixed phrase for a missing
    fenced block, a different fixed phrase for a block that did not decode. Do not carry the
    extraction error's own text into that variable.
  - After `if accepted { break }` and before the existing `InvalidateRep` call, increment `attempt`
    and build one `InvalidReason` value with `Cell: cfg.ID`, `Repetition: rep`, `Attempt: attempt`,
    then classify `Cause` in this exact order:
    1. `invokeErr != nil` → `CauseRunnerError`.
    2. otherwise `t == nil || t.Result == nil || t.Result.IsError` → `CauseResultError`.
    3. otherwise `serverFinding != nil` → `CauseServerNotConnected`.
    4. otherwise → `CauseUnparseableAnswer`.

    The order is load-bearing: a process that exits non-zero is `runner_error` even though its
    transcript also lacks a usable result record, which is why `result_error` is reachable only from
    a process that exits zero and still reports failure.
  - Construct `Detail` per cause, never from `err.Error()`:
    - `runner_error`: a fixed phrase naming that the measured claude process failed, plus the
      `*exec.ExitError`'s own status string when one is recoverable.
    - `result_error`: `"no result record in transcript"` when `t == nil || t.Result == nil`;
      otherwise a single line naming the result record's `TerminalReason` and `StopReason` values.
    - `unparseable_answer`: the fixed phrase recorded above.
    - `server_not_connected`: `serverFinding.Message`, which is already a short fixed string.

    Passing the wrapped error through is forbidden: `ExecRunner.Run` formats it as
    `run %s %s: %w` over the full argument vector, which carries `-p` with the entire rendered
    prompt, the absolute `--mcp-config` path and the claude binary path — kilobytes, multi-line, and
    full of machine paths, breaking the one-pair-per-line shape in the same stroke. Read
    `worktree.go` to confirm that wrapping shape; do not change it.
  - Set `ExitCode` only for the `runner_error` case and only when a status is recoverable. Add an
    unexported helper `exitCodeOf(err error) (int, bool)` to `run.go` that declares
    `var exitErr *exec.ExitError`, calls `errors.As(err, &exitErr)`, and returns
    `exitErr.ExitCode(), true` on a match and `0, false` otherwise. When it reports false, leave
    `ExitCode` nil so the `exit_code` line is omitted rather than written as a sentinel.
  - Call `WriteInvalidReason(dir, reason)` on **every** rejected attempt, returning
    `repOutcome{}, err` on failure exactly as the existing `writeServerConnectFailure` call site
    does. Keep the `connectFailures++` increment on the `serverFinding != nil` path unchanged and
    keep it before the write, so the `connectFailures == attempts` whole-run abort below continues
    to fire on exactly the same condition it does today.
  - Delete `writeServerConnectFailure` from `run.go` entirely; nothing calls it after this change.
  - Add `errors` and `os/exec` to `run.go`'s import block.

  In `runstate.go`, delete the `ServerConnectFailureFile` constant and its doc comment. After this
  card, no Go source or test in the repository refers to that identifier — verify with a grep across
  `bench/loomyard-eval/ladder/`. Do not touch the two prose references to it in
  `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`: a committed conclusion is a
  frozen record of what the harness did at that time, and rewriting it to match a later harness
  would falsify it.

  `summarize.go` and `report.go` read no reason file and need no change; `summarize.go`'s comment
  referencing `InvalidateRep` stays as it is.
- **Commit:** `feat(ladder): write an invalid_reason.txt on every invalidated attempt`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/internal/ladder/` runs the whole `ladder` package's
test files, which is the correct scope here rather than a narrower `-run` filter: this batch changes
`runstate.go` and `run.go`, the two files the package's e2e, runstate, summarize, provenance and
gates tests all exercise transitively, so a narrower filter would not catch a regression the change
can plausibly cause. The package's tests are offline — `testdata/fakeclaude` stands in for the
claude binary and no network or real CLI is involved — so the full package run is cheap.

The files this batch's tests cover:

- `runstate_test.go` — `TestRenderInvalidReason` (new, card 2) and the unchanged
  `TestInvalidateRep`, whose three-attempt suffix assertions are the regression guard that the
  rename semantics were not touched.
- `e2e_test.go` — the four new subtests plus the retargeted `GrantedCellServerNeverConnects` and the
  extended `HappyPath` (card 4). Between them these prove all four causes reach disk, that the
  attempt counter diverges from the directory suffix on a re-entered root, that a completed
  repetition carries no reason file, and that the `connectFailures == attempts` whole-run abort
  still fires.
- `testdata/fakeclaude/main.go` — not itself under test, but the `result_error` variant it gains in
  card 3 is what makes the third e2e subtest possible.

`MaxAttempts` is still 3 and is asserted so by the existing tests that loop `1..MaxAttempts`; no
card changes it.
