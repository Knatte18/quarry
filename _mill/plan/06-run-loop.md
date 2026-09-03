# Batch: run-loop

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "run-loop"
number: 6
cards: 3
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [3, 4, 5]
```

## Batch Scope

This batch is the loop everything else exists to serve: the on-disk contract for one repetition,
and the sequential driver that prepares a worktree, renders a prompt, runs the measured process,
tees its stream, computes metrics, applies the gates, scores, and writes the rep's six raw files
with the state file last. It is one batch because the failure taxonomy is only expressible as one
piece of control flow — five outcomes with five different dispositions, sharing one write order.
The external interface batch 7 consumes is `RepDir`, `ReadRunState`, `RepIsComplete` and `Run`.

Batch-local decision: no test in this batch executes the measured process. Its end-to-end coverage
lands in batch 8 against a fake binary; here only the on-disk state contract is unit-tested, which
is the part `report` depends on and the part a killed run must survive.

## Cards

### Card 25: the repetition state contract

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `runstate.go`. `RepDir(resultsRoot, cellID string,
  rep int) string` returns the raw repetition directory, `<results-root>/raw/<cell>/<rep>`.
  Declare the six per-repetition filenames as package-level string constants here, the single place
  they are spelled — `TranscriptFile = "transcript.jsonl"`, `AnswerFile = "answer.json"`,
  `RedactedAnswerFile = "answer.redacted.json"`, `UsageFile = "usage.json"`,
  `ScoreFile = "score.json"` and `RunStateFile = "run.json"` — matching the overview's
  the-six-per-repetition-filenames decision; every writer and reader in batches 6, 7 and 8
  references these constants rather than a literal.
  Declare `RunState` with `json` tags for exactly the payload the discussion fixes: `state`,
  `config_id`, `ladder`, `task`, `allowed`, `is_control`, `control_for_ladder`, `server_name`,
  `mcp_prefix`, `rep`, `model`, `effort`, `max_turns`, `scored`, `score_skip_reason`,
  `observations` as a slice of gate findings, `blinding_failed` and `max_turns_hit`. The server name
  and MCP prefix are carried per rep on purpose: the prefixed-tool-use count, and therefore gate 1,
  must be recomputable by a consumer that never reads the ladder file. Implement `WriteRunState(dir
  string, s RunState) error` writing the state file, `ReadRunState(dir string) (RunState, error)`,
  and `RepIsComplete(dir string) bool` returning true if and only if the state file parses, its
  state field is the literal `complete`, **and** its blinding-failed flag is false. A rep discarded
  for blinding is therefore written as complete so its transcript and reason survive on disk, yet
  does not satisfy resume — the next invocation re-attempts it once the operator fixes the cause,
  which is what keeps a memory-taint discard recoverable. Implement `InvalidateRep(dir string)
  (attempts int, err error)` renaming the directory to the same path suffixed with a dot, the word
  `invalid`, a dash and the next unused attempt number starting at one, and returning how many such
  directories now exist; declare `MaxAttempts = 3`.
- **Commit:** `feat(ladder): define the per-rep state contract and the invalid-rep rename`

### Card 26: the sequential run loop

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `run.go`. It uses the package-level `BuiltinTools`
  slice declared in `config.go` by batch 1 as its tools value and never re-spells those names.
  Declare `RunOptions` carrying the ladder file
  path, the results root, the selected cell ids, an optional repetition override, the claude binary
  path, the quarry repository root and a `Runner`, and `Run(ctx context.Context, opts RunOptions)
  (exitNonZero bool, err error)`.
  Startup, in order: load the ladder file; resolve the selected cells, defaulting to every config in
  the file, and error on an id the file does not contain; resolve the effective repetition count as
  the override when given, else the file's own; resolve the quarry repository root, the target
  repository and the worktree root
  with their assertions; acquire the advisory lock and defer its release; call the provenance reader
  and refuse immediately when an existing record's effective repetition count differs from this
  invocation's; call the invocation collector once, merge its result into the record and write the
  record, so a run that dies mid-matrix still leaves its own invocation on disk; when the existing
  record already carries memory paths, scan them **before the first
  new repetition** and abort on a fatal finding, because a resumed run skips the very repetition
  that would otherwise reveal them; build the server once when and only when a selected cell has a
  non-empty allowed list, recording its hash into the record for every cell-and-repetition pair it
  serves.
  The loop is strictly sequential and **cell-minor**: repetition 1 of every selected cell, then
  repetition 2 of every cell, and so on. Nothing runs concurrently — duration and cost are measured
  metrics, and parallel runs share both the rate limit and the prompt cache. Skip any repetition
  whose directory already satisfies the completeness predicate.
  Per repetition: prepare or restore the pinned worktree; render the prompt from the task file's
  extracted content, the worktree path and the cell's tool names, which are the BuiltinTools slice
  from `config.go` plus each granted tool prefixed with the MCP prefix; for a control cell run gate check (d) on that
  rendered prompt **before dispatch** and, on a finding, write the repetition as complete with the
  blinding-failed flag set and move on without spending an API call; write the per-cell MCP
  configuration; invoke the measured process through the runner seam with the working directory set
  to the pinned worktree, standard input from the null device, and exactly the flag set the
  discussion's claude-invocation decision fixes — model, effort, max turns, the BuiltinTools slice
  from `config.go` as the tools value, the granted tool names as an allowlist **omitted entirely for a control cell**, the per-cell MCP
  configuration with strict configuration mode, stream-json output with verbose on, session
  persistence off, and empty setting sources. No permission-mode flag is passed. Tee standard output
  to the repetition's transcript file as it arrives.
  Then, in this write order with the state file last: parse the transcript; compute the metrics with
  the prefix from the loaded file and stamp the effort the harness passed, since the CLI does not
  echo it; on the first completed repetition of a fresh root, take the memory paths from the
  session-init record, persist them into the provenance record at that moment rather than at the end
  of the run, and scan them, discarding the repetition exactly like a blinding failure when tainted
  and aborting the run; run gate 2 for a control cell over the whole marshalled transcript; extract
  the answer as the **last** fenced JSON block of the final assistant record's concatenated text and
  decode it, with no schema-key validation; redact it and dispatch the scorer; write the six files
  the overview's the-six-per-repetition-filenames decision names, in that order — `transcript.jsonl`
  already tee'd, then `answer.json`, `answer.redacted.json`, `usage.json`, `score.json` and
  `run.json` last; record the dirtied observation from the worktree's
  porcelain status and restore the worktree.
  The scorer is a **second measured-binary invocation**, through the same runner seam and the same
  claude binary path, and it is the only other process this loop runs. Add
  `RunScorer(ctx context.Context, r Runner, claudeBin string, l *Ladder, task Task, prompt string)
  (ScoreRecord, error)` to `run.go`, invoking the binary with the scorer model and effort taken from
  the ladder file's own scorer block — never the cell's run model or effort — an empty tools value,
  a turn ceiling of one, an empty MCP configuration document with strict configuration mode,
  stream-json output with verbose on, session persistence off, empty setting sources, and standard
  input from the null device. It runs with the working directory set to the harness's own scratch
  directory, **not** the pinned worktree: the scorer needs no codebase and must not be given one.
  Its prompt is the assembled four-part scorer prompt built from the rule the task's schema selects,
  the extracted task text, the meta-stripped fasit and the redacted answer; its reply is parsed with
  the scorer-reply parser.
  Dispositions, exactly five and no others: a non-zero exit, an unparseable stream or an error flag
  in the result record is an infrastructure failure, and a missing or undecodable fenced answer is a
  formatting miss — both rename the repetition directory and retry up to the attempt ceiling, after
  which the cell is recorded incomplete, the run continues with the remaining cells, and the process
  exits non-zero. A max-turns terminal reason is **complete, not a failure**: full cost metrics, no
  answer file, the scorer is not invoked, and the score record is the unscored stand-in carrying the
  max-turns reason. A fatal gate-2 finding — check (a) or (b) after the run, exactly like check (d)
  before it — writes the repetition's state as complete **with the blinding-failed flag set to
  true**, and is **never** retried: it is deterministic, so three retries buy three identical
  failures at full price. Setting that flag is what the whole void-repetition mechanism rests on —
  the completeness predicate returns false for it, so the next invocation re-attempts it once the
  operator fixes the cause; the summariser excludes it from both medians, does not count it as
  present, and puts its cell in the invalid list; and the process exits non-zero. A finding written
  without the flag would satisfy resume and be silently skipped forever. A
  scorer failure retries **only the scorer**, up to the `MaxAttempts` ceiling card 25 declares —
  the same constant the invalid-repetition path uses, not a second hand-written three — never the
  measured run, and then writes the unscored stand-in carrying the scorer-failed reason.
  Return a non-zero exit signal when any cell is incomplete or any cell has a repetition with the
  blinding-failed flag set. Do not retry a max-turns repetition and do not re-execute a measured
  repetition because its scorer failed.
- **Commit:** `feat(ladder): drive the sequential per-cell run loop with resume and gates`

### Card 27: repetition state tests

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write `runstate_test.go` over a temporary directory asserting: the repetition
  directory path shape; a written state round-trips every field including the observations slice;
  the completeness predicate is true only for a parsing state whose state field is complete and
  whose blinding-failed flag is false — with explicit cases for a missing file, a truncated file, a
  state that is not complete, and a complete state whose blinding-failed flag is true, the last
  being the case that must return false so a void repetition is re-attempted; and the invalidation
  rename produces the first, second and third suffixed directories in order and reports the
  attempt count, with the ceiling constant asserted to be three.
- **Commit:** `test(ladder): cover the rep state contract and the invalid-rep rename`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers this batch through `runstate_test.go`,
plus every earlier batch's tests, which still guard the pieces the loop composes. The loop itself is
deliberately untested here: it can only be exercised against a process, and the fake binary that
provides one is batch 8's first card. The single assertion in this batch that must not be weakened
is that a complete repetition carrying the blinding-failed flag does **not** satisfy resume — that
is what makes a discarded control repetition recoverable instead of permanently skipped.
