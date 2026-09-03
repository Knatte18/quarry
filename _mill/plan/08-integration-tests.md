# Batch: integration-tests

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "integration-tests"
number: 8
cards: 5
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [7]
```

## Batch Scope

This batch is the offline proof that the whole program works, plus the one live test that proves
the assumptions about the real CLI that no offline test can. It builds two small helper binaries
from the package's own test data — a stub MCP server and a fake measured process — drives the run
loop end to end against them including resume and the failure path, checks the report subcommand
against a committed fixture root and golden output, and adds the environment-guarded smoke test.
It depends on batch 7 because every one of those paths ends in the summariser and the table renderer
that batch adds. The binary's own flag wiring is deliberately **not** exercised by any test: the
tests drive the library entry points directly, since `main.go` is thirty lines of flag parsing over
them and an in-process test of a `package main` would prove nothing the entry-point tests do not.
The binary is instead verified by the hand-run done-criterion the discussion names.

Batch-local decision: both helper binaries are built by the test with the Go toolchain into a
temporary directory, never committed as binaries and never added to the module's own build. They
live under the package's test data directory so they are excluded from a normal build, and each is
its own `package main`.

## Cards

### Card 32: the stub MCP server

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/mcp_test.go`
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/stubmcp/main.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** the mcp test file already exists from batch 5; extend it rather than replacing
  its cases. Write a small `package main` under the stub directory speaking JSON-RPC over standard
  input and output: it answers an initialize request, answers a tools-list request by advertising
  exactly two tools — one the test's cell grants and one it does not — and answers a tools-call
  request with a deterministic JSON text payload. It exits on end of input. Keep it under roughly a
  hundred lines and depend on nothing outside the standard library. Add a test that builds it with
  the Go toolchain into a temporary directory, generates the per-cell MCP configuration document
  through the harness's own writer for a cell granting one of the two tools, launches the server
  using the command and arguments **that document declares** rather than a hand-written command
  line, completes the initialize and tools-list handshake, and asserts that the tool names the
  harness would pass as its allowlist — the prefix applied to the cell's granted names — correspond
  to tools the server advertises. That correspondence is the whole point: it proves the generated
  document is well-formed, that its declared command launches, and that the two sides agree on
  names.
- **Commit:** `test(ladder): drive a stub mcp server from the harness-generated config`

### Card 33: the fake measured process

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/grouped-usage.jsonl`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write a `package main` that stands in for the measured binary. It stands in for
  **two** different invocations — the measured cell and the scorer — and it distinguishes them from
  the arguments alone, with no out-of-band signal, because both reach it through the same binary
  path within one process run of the harness and per-invocation environment is therefore not
  available. The discriminator is the pair card 26 fixes for exactly this purpose: an invocation
  carrying `--tools ""` together with `--max-turns 1` is the scorer; anything else is a measured
  cell. Assert a different flag set on each branch, failing with a non-zero exit and a message on
  standard error when an expected flag is absent or carries an unexpected value.
  For a **measured cell**: the model, the
  effort, the turn ceiling, the built-in tool set named in the overview's the-four-built-in-tools
  decision, the MCP configuration path, strict
  configuration mode, stream-json output, verbose, no session persistence and empty setting sources;
  plus the allowlist flag **absent** when an environment variable the test sets marks the invocation
  as a control, and present with the expected prefixed names otherwise.
  For the **scorer**: the scorer model and effort — asserted to differ from the cell's, so a mix-up
  fails the test rather than passing quietly — the empty tools value, the turn ceiling of one, an
  MCP configuration path, strict configuration mode, stream-json output, verbose, no session
  persistence and empty setting sources; and the allowlist flag absent unconditionally.
  Those failure messages are readable by the test because the runner seam carries a separate error
  writer, per the overview's injectable-external-commands decision.
  The fake never
  spells the built-in tool names itself — it is a separate `package main` and cannot reference the
  harness package's slice — so the test passes the expected tools value in through an environment
  variable built from that slice, and the fake compares against what it was given. It then writes a
  canned stream to standard output, chosen by an environment variable the test sets, covering at
  least: a normal run whose final assistant record ends with a fenced JSON answer; a run whose
  result record carries a max-turns terminal reason and no fenced answer; a run that exits non-zero
  after a partial stream; a run whose final message carries no fenced block at all; and a scorer
  reply. Emitting a fixed canned stream rather than replaying a committed transcript keeps the
  fake's behaviour selectable per case; where a realistic record shape is needed, copy it from the
  existing grouped-usage fixture. Keep it under roughly a hundred and fifty lines, standard library
  only.
- **Commit:** `test(ladder): add a flag-asserting fake claude binary`

### Card 34: the end-to-end run, resume and failure tests

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/report.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write `e2e_test.go` building the fake binary once and running the harness's run
  entry point against a synthetic ladder file, a synthetic task file and a synthetic fasit, all
  written to a temporary directory, with the worktree root pointed at a temporary directory through
  the override variable and the target repository a small local git repository the test initialises
  and commits so a pinned worktree can actually be created. Assert the happy path end to end: the
  worktree is prepared, the MCP configuration is written, the process is invoked with the expected
  flags, the transcript is tee'd to the repetition directory, the metrics are computed, the gates
  run, the scorer is invoked as its own second process with a redacted answer and the ladder file's
  scorer model and effort, and the repetition directory holds exactly the six files the overview's
  the-six-per-repetition-filenames decision names and ends with
  the state file — assert the state file's modification time is not earlier than every other file in
  that directory, which is what makes resume safe against a kill. Assert the summary, the provenance
  record and the table are all written at the results root and that the printed table equals the
  written one. Add a resume test that keeps the effective repetition count **constant** across both
  invocations, since a differing one is refused at startup by the provenance rule and a test that
  changed it could never pass: run a two-cell, two-repetition matrix restricted by the cell selector
  to the first cell, then run it again with no cell restriction, and assert that the first cell's
  two repetitions were skipped — their transcripts unchanged by modification time and content —
  while the second cell's two were produced. That is also the case the merge rule's union of
  selected cells exists for, so the test covers both at once. Add a failure
  test: with the fake configured to fail, assert three suffixed invalid directories are produced,
  the cell appears in the summary's incomplete list, the run continues past that cell, and the entry
  point reports a non-zero exit. Add a blinding test: with the fake configured to emit a transcript
  carrying the MCP prefix on a control cell, assert the repetition is written complete with the
  blinding-failed flag set, that it is **not** retried, that the cell lands in the invalid list, and
  that a second run re-attempts that same repetition rather than skipping it. Add a lock test:
  a second run against the same worktree root fails with the message naming the first holder.
- **Commit:** `test(ladder): exercise run, resume, failure and blinding end to end`

### Card 35: the report subcommand over a committed fixture root

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/report.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/provenance.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/transcript.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/answer.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/answer.redacted.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/score.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/run.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/2/transcript.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/2/score.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/2/run.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/golden-summary.json`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/results/golden-table.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:** commit a complete raw tree for one control cell and two repetitions and **no
  summary file**: repetition 1 is an ordinary scored run, carrying its answer and the redacted answer
  the scorer saw; repetition 2 hit the turn ceiling, so it
  carries neither answer file and a score record whose scored flag is false with the max-turns
  reason — the absence of both answer files there is what a ceiling repetition looks like, not an
  omission.
  Neither transcript may contain an absolute host path, and the provenance record must carry the
  selected cells and the effective repetition count, since those are what the incomplete calculation
  needs. The metrics file is the one per-repetition name deliberately absent from **both**
  repetitions: the report path recomputes every metric from the transcript, and a fixture that cannot
  supply stored metrics proves it. Every other file the six-filenames decision calls for is present
  wherever that decision says it should be. Extend
  `e2e_test.go` with a report case that copies the fixture root into a temporary directory **under
  the same base-name directory `root`**, runs the
  summarise-and-report path over the copy, and compares the produced summary and table byte-for-byte
  against the two golden files with no field normalisation and no exclusions. That comparison is
  exact only because the summary and the table each carry the root's base name and no wall-clock
  time, per the overview's no-machine-paths-in-tracked-output decision, and because the CLI version
  in the header comes from the fixture's own provenance record; preserving the base name across the
  copy is therefore load-bearing, not incidental. Meanwhile the fake binary — placed on the path for the duration — asserts
  it was **never invoked**, which is the subcommand's entire justification. Assert in the same case
  that the recall and precision medians exclude the ceiling repetition while its cost metrics are
  included, and that the max-turns and unscored counters are populated. Write the two golden files
  from the implementation's own first correct output, then read them back and confirm by eye that
  the medians match the two transcripts by hand before committing them.
- **Commit:** `test(ladder): re-derive a committed fixture root without running anything`

### Card 36: the guarded live smoke test

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/live_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write `live_test.go` guarded by the environment variable `LADDER_LIVE_TEST` set
  to `1`, calling the skip helper immediately when it is unset so the offline suite stays free and
  deterministic. When enabled, it runs one repetition of the control cell `a0-none` from the
  migrated ladder file against the real CLI, in a **freshly created** worktree — the test removes
  any existing worktree for that task first, so the directory under test is genuinely new. It then
  asserts three things about the resulting transcript. First, the session-init record's tool list is
  exactly the four built-in tools named in the overview's the-four-built-in-tools decision plus any granted prefixed names, which for this control cell means the
  four alone: a silently degraded tool grant in a new directory would void every metric from such a
  run, and this is the assertion that catches it. Second, the server list is empty, proving strict
  configuration mode held and the operator's own personal servers did not load. Third — and this is
  the one the advertised list cannot give — at least one Bash tool use in the transcript has a
  matching tool result carrying output, and the result record's permission-denial list contains no
  entry for a read-only command: the advertised list says what was offered, only an executed call
  proves the grant works from a fresh directory.
- **Commit:** `test(ladder): add the guarded live smoke test for a fresh worktree`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` runs the whole package. Everything in this batch
except card 36 runs offline, free and deterministically; the live test skips unless its guard
variable is set, so the batch verify command stays usable on every implementer round. The golden
comparison in card 35 is the batch's sharpest assertion — it fails on any accounting change, which
is correct, and the fix is to re-derive the goldens deliberately rather than to loosen the
comparison.
