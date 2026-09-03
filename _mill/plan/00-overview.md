# Plan: Ladder harness around headless claude -p (T2)

```yaml
task: "Ladder harness around headless claude -p (T2)"
slug: "ladder-harness"
approved: false
started: "20260903-171341"
parent: "main"
root: ""
verify: go build ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: foundations-and-config
    file: 01-foundations-and-config.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 2
    name: stream-and-metrics
    file: 02-stream-and-metrics.md
    depends-on: [1]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 3
    name: prompt-and-schema
    file: 03-prompt-and-schema.md
    depends-on: [1]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 4
    name: gates-and-scorer
    file: 04-gates-and-scorer.md
    depends-on: [1, 2, 3]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 5
    name: environment-and-provenance
    file: 05-environment-and-provenance.md
    depends-on: [1, 2, 4]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 6
    name: run-loop
    file: 06-run-loop.md
    depends-on: [3, 4, 5]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 7
    name: summarize-report-cli
    file: 07-summarize-report-cli.md
    depends-on: [6]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 8
    name: integration-tests
    file: 08-integration-tests.md
    depends-on: [7]
    verify: go test ./bench/loomyard-eval/ladder/...
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: v1-reference-reads-go-through-git-show

- **Decision:** the V1 harness sources this task ports from are not on disk and must never be
  checked out or merged. Every reference read is a single read-only Bash command of the form
  `git show origin/v1-final:<path> | sed -n '<a>,<b>p'` — except that this repo bans `sed`, so use
  `git show origin/v1-final:<path>` piped through `awk 'NR>=a && NR<=b'`, or unpiped for a whole
  file. Because these paths do not exist in the worktree they can never appear in a card's
  `Context:` list; each card that needs one names the exact `origin/v1-final:` path and the exact
  Go identifier to port in its `Requirements:` prose instead. Reading a V1 file this way is
  explicitly authorised for every batch.
- **Rationale:** `Context:` is an allowlist of worktree files; a V1 path listed there would be a
  plan defect (the file does not exist) and would fail the plan validator's non-existent-path
  check. The port targets are named identifiers, not line ranges, so a small drift in V1's line
  numbering cannot mislead the implementer.
- **Applies to:** all batches

### Decision: one-package-files-per-concern

- **Decision:** all harness logic lives in the single Go package `ladder` at
  `bench/loomyard-eval/ladder/internal/ladder/`, one file per concern, with `package ladder` at the
  top of every file. The only other Go directory is `bench/loomyard-eval/ladder/cmd/ladder/`, a
  thin `package main` that parses flags and calls exported entry points. No sub-packages, no
  package-per-verb. Test files sit beside their subject as `<name>_test.go` in the same package
  (internal tests), so unexported identifiers are directly testable.
- **Rationale:** the shape V1 proved and the rule the rewrite plan states for T3; it keeps the
  `ladder*.yaml` files at their current tracked paths and keeps internals unit-testable without
  exporting them through `main`.
- **Applies to:** all batches

### Decision: file-layout-additions-beyond-the-discussion-list

- **Decision:** in addition to the twelve concern files the discussion names (`config.go`,
  `worktree.go`, `mcp.go`, `prompt.go`, `run.go`, `stream.go`, `metrics.go`, `score.go`,
  `gates.go`, `provenance.go`, `summarize.go`, `report.go`) this plan adds exactly three more:
  `match.go` (the single shared token matcher), `fenced.go` (the ported `ExtractFencedJSON`) and
  `runstate.go` (the `run.json` payload, the completeness predicate and the `.invalid-<k>`
  rename). The single-run advisory lock lives inside `worktree.go` rather than a file of its own,
  because it is created and removed at the worktree-root path that file already resolves.
- **Rationale:** the token matcher and the fenced-JSON extractor each have three consumers and the
  discussion's own token-matching decision exists precisely to stop them being re-derived per
  consumer; giving each one file is what makes "one place" enforceable. `runstate.go` separates
  the on-disk rep-state contract from the run loop that drives it, which is what lets `report`
  read a results root without touching `run.go`.
- **Applies to:** all batches

### Decision: error-posture

- **Decision:** every loader, gate and parser returns a wrapped `error` naming the offending
  file, key or cell id; the harness never logs-and-continues on a configuration fault. Fatal
  conditions (startup assertions, load errors, a fatal gate finding) abort before any API call is
  made. Non-fatal findings (gate 1, `worktree_dirtied`, session-fingerprint drift, the
  `target_origin_quarry_mention` observation) are recorded in the run's own JSON output and
  printed, never returned as errors. Use `fmt.Errorf("...: %w", err)` for wrapping and package-level
  `errors.New` sentinels only where a caller must branch on identity.
- **Rationale:** the expensive resource here is API spend; a misconfiguration must cost nothing,
  and an observation must never abort a root that is already half-paid-for.
- **Applies to:** all batches

### Decision: injectable-external-commands

- **Decision:** every external process the harness runs — `claude`, `git`, `go build` — is invoked
  through an injectable seam, not `exec.Command` at the call site. Define one `Runner` interface in
  `worktree.go` with the shape `Run(ctx context.Context, c Cmd) error`, where `Cmd` is a struct
  carrying `Dir string`, `Name string`, `Args []string`, `Env map[string]string` (entries merged
  over the inherited environment), `Stdin io.Reader`, `Stdout io.Writer` and `Stderr io.Writer`,
  plus an `ExecRunner` production implementation. The environment map and the error writer are
  both load-bearing: the server build forces a cgo variable through the first, and the fake binary
  the offline tests drive reports its flag-assertion failures on the second. The `claude` binary
  path additionally comes from a `--claude-bin` flag defaulting to `claude`.
- **Rationale:** the whole offline test layer (fake `claude`, stub MCP server, resume and failure
  tests) depends on this seam; V1 already took a builder seam for the same reason.
- **Applies to:** batches 5, 6, 7, 8

### Decision: no-machine-paths-in-tracked-output

- **Decision:** no file this task creates or writes at runtime into a tracked location may contain
  an absolute host path. `provenance.json` carries `loomyard_repo_sha256`, never the resolved
  repository path. `summary.json` and `table.txt` carry cell ids, metrics, and — as the only
  identification of the root they describe — the results root's **base name**, never the path the
  operator passed: a root invoked by absolute path would otherwise write a machine path into two
  tracked files. Neither file carries a wall-clock write time; that belongs in `provenance.json`,
  which is not golden-compared. Those two exclusions together are also what make batch 8's
  byte-for-byte golden comparison possible at all. Test fixtures
  committed under `testdata/` use relative or clearly synthetic paths.
- **Rationale:** `HANDOFF.md` section 1 states the rule; it is the reason the raw transcript tree is
  gitignored rather than committed.
- **Applies to:** all batches

### Decision: where-the-done-criterion's-test-half-is-enforced

- **Decision:** the task's done-criterion is a repository-wide `go build ./... && go test ./...`.
  The build half is the overview's module-wide `verify:`, run at every batch boundary. The test half
  is **not** run at every batch boundary: an unscoped repository-wide test command as a per-boundary
  gate is what the plan validator's `verify-full-suite` check exists to refuse, and each batch's own
  `verify:` already runs every test this task writes. The repository-wide test half is instead
  enforced once, at the end, by the hub's configured done-gate — mill-go runs
  `go test ./...` from the repository root before marking the task done, which is the gate that
  catches a regression in a package outside this task's batch scopes.
- **Rationale:** the two halves guard different things. The build half is cheap and catches the one
  cross-package change this task makes, the new module dependency; the test half is a whole-repo
  regression suite whose right frequency is once, not once per batch. Naming where it runs is what
  keeps the done-criterion accounted for rather than assumed.
- **Applies to:** all batches

### Decision: the-four-built-in-tools

- **Decision:** every cell, control and treatment alike, is granted exactly the built-in tool set
  `Read`, `Grep`, `Glob`, `Bash` — passed as the CLI's tools value in that spelling, and reported
  back by the session-init record as the sorted list `["Bash","Glob","Grep","Read"]`. Declare it
  once as a package-level slice in `config.go` (batch 1, card 3) — the earliest file in the DAG, so
  every consuming site is strictly downstream of it — and reference that identifier from every other
  site rather than re-spelling the names: the rendered prompt's tool list, gate check (d)'s passing
  case in the tests, the run loop's tools value, and the live test's session-init assertion. The
  fake binary of batch 8 is the one site that cannot reference it, since it is a separate `package
  main` built into a temporary directory; the test passes it the expected value through an
  environment variable built from the slice, so the fake still never spells the names itself.
  `Bash` is
  granted bare, exactly as V1 did — narrowing it to read-only command patterns would make denied
  Bash calls a behavioural difference between arms, which is the confound one identical preamble
  exists to remove.
- **Rationale:** five cards depend on this one set, and a set spelled five times is a set that will
  eventually be spelled four ways. The sorted-versus-passed distinction matters too: the value
  passed and the value reported differ in order, and a test asserting the wrong one fails for the
  wrong reason.
- **Applies to:** batches 1, 3, 4, 6, 8

### Decision: the-six-per-repetition-filenames

- **Decision:** one repetition directory holds exactly six files, with these exact names, written by
  the run subcommand in this order and with the state file last: `transcript.jsonl` (the tee'd
  stream), `answer.json` (the decoded fenced answer), `answer.redacted.json` (that answer after the
  scorer's redaction — kept because it is what the scorer actually saw, which is otherwise
  unreconstructable when a score looks wrong), `usage.json` (the computed metrics), `score.json`
  (the scorer's parsed reply or its unscored stand-in) and `run.json` (the repetition state). The
  metrics file is **diagnostic only**: the report path recomputes every metric from the transcript,
  so an accounting bug is fixed by re-reporting, never by trusting the stored figures. A repetition
  that hit the turn ceiling writes no answer file and no redacted-answer file.
- **Rationale:** four of the six names appear in the fixture tree and two were previously referred to
  only by description; a name that exists only as prose is a name two cards will spell differently.
- **Applies to:** batches 6, 7, 8

### Decision: go-test-stays-offline-and-free

- **Decision:** `go test ./bench/loomyard-eval/ladder/...` must pass with no network, no
  credentials and no spend. The only live test is guarded by `LADDER_LIVE_TEST=1` and calls
  `t.Skip` when it is unset. No test may write outside `t.TempDir()` or read outside the package's
  own `testdata/`.
- **Rationale:** this is a stated task constraint and it is what makes the batch `verify:` command
  usable as a gate on every implementer round.
- **Applies to:** all batches

### Decision: line-budget

- **Decision:** the non-test Go the harness ships targets roughly 1 000–1 500 lines. Test code,
  the two test-built helper binaries and `testdata/` fixtures are outside that budget. When a card
  can be satisfied by a smaller surface, take the smaller surface: no options structs with unused
  fields, no interfaces with one implementation outside the injectable-external-commands seam, no
  exported identifier without a caller in this plan.
- **Rationale:** the size of the V1 harness was architectural, and the rewrite's whole claim is
  that direct control of the process removes the machinery. A plan that lands 3 000 lines has not
  made that claim true.
- **Applies to:** all batches

### Decision: results-tree-gitignore-scope

- **Decision:** the new `bench/loomyard-eval/ladder/.gitignore` contains exactly the single
  pattern `results/*/raw/`. Because the pattern contains a slash it anchors to that `.gitignore`'s
  own directory, so it matches `bench/loomyard-eval/ladder/results/<root>/raw/` and nothing else —
  in particular it does not match the committed fixture tree under
  `bench/loomyard-eval/ladder/internal/ladder/testdata/`, whose `raw/` directories must stay
  tracked.
- **Rationale:** the same anchoring property that makes a root-level rule fail silently is what
  makes this file-local rule correct and scoped; batch 8's fixture root depends on it not
  over-matching.
- **Applies to:** batches 1, 8

## All Files Touched

_Full union of every `Creates:` / `Edits:` / `Moves:` **target** path across every batch, sorted alphabetically (Move **source** paths are excluded — they disappear, like `Deletes:` tokens).
Cards are the source of truth;
this section is the input `_plan_validate.py`'s `all-files-touched-mismatch` check cross-references against the derived union of every card's `Edits:`/`Creates:`/Move-target paths, to catch drift between the hand/agent-maintained list here and that derived union._

- `bench/loomyard-eval/ladder/.gitignore`
- `bench/loomyard-eval/ladder/cmd/ladder/main.go`
- `bench/loomyard-eval/ladder/internal/ladder/config.go`
- `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
- `bench/loomyard-eval/ladder/internal/ladder/fenced_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/live_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/match.go`
- `bench/loomyard-eval/ladder/internal/ladder/match_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
- `bench/loomyard-eval/ladder/internal/ladder/mcp_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
- `bench/loomyard-eval/ladder/internal/ladder/metrics_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
- `bench/loomyard-eval/ladder/internal/ladder/prompt_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- `bench/loomyard-eval/ladder/internal/ladder/provenance_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/report.go`
- `bench/loomyard-eval/ladder/internal/ladder/report_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/run.go`
- `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/score.go`
- `bench/loomyard-eval/ladder/internal/ladder/score_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/stream.go`
- `bench/loomyard-eval/ladder/internal/ladder/stream_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
- `bench/loomyard-eval/ladder/internal/ladder/summarize_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/golden-summary.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/golden-table.txt`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/provenance.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/answer.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/answer.redacted.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/run.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/score.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/1/transcript.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/2/run.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/2/score.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/results/root/raw/a0-none/2/transcript.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/stubmcp/main.go`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/tasks/no-schema-heading.md`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/grouped-usage.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/leaked-prefix.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/max-turns.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/target-origin-quarry.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/tool-bytes.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
- `bench/loomyard-eval/ladder/internal/ladder/worktree_test.go`
- `bench/loomyard-eval/ladder/ladder-toc.yaml`
- `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
- `go.mod`
- `go.sum`
