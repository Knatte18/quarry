# Plan: Port the capability-ladder bench harness to Go

```yaml
task: Port the capability-ladder bench harness to Go
slug: port-ladder-bench-to-go
approved: true
started: '20260830-112301'
parent: 'main'
root: ""
verify: go vet ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: ladder-config
    file: 01-ladder-config.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 2
    name: transcript-usage
    file: 02-transcript-usage.md
    depends-on: [1]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 3
    name: transcript-gates
    file: 03-transcript-gates.md
    depends-on: [2]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 4
    name: daemon-cold-gates
    file: 04-daemon-cold-gates.md
    depends-on: [3]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 5
    name: run-state
    file: 05-run-state.md
    depends-on: [4]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 6
    name: scoring
    file: 06-scoring.md
    depends-on: [1]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 7
    name: summarize
    file: 07-summarize.md
    depends-on: [5]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 8
    name: worktree-server-warm
    file: 08-worktree-server-warm.md
    depends-on: [5]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 9
    name: task-schema-planning
    file: 09-task-schema-planning.md
    depends-on: [5]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 10
    name: session-provisioning
    file: 10-session-provisioning.md
    depends-on: [6, 8, 9]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 11
    name: cli-session-commands
    file: 11-cli-session-commands.md
    depends-on: [10]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 12
    name: cli-run-commands
    file: 12-cli-run-commands.md
    depends-on: [7, 11]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 13
    name: e2e-test
    file: 13-e2e-test.md
    depends-on: [12]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 14
    name: skill-docs-cleanup
    file: 14-skill-docs-cleanup.md
    depends-on: [13]
    verify: go test ./bench/loomyard-eval/ladder/...
```

## Shared Decisions

### Decision: package layout and binary shape

- **Decision:** one binary at `bench/loomyard-eval/ladder/cmd/ladderbench` with cobra subcommands, and
  all logic in the package at `bench/loomyard-eval/ladder/internal/ladder/`. No file is added under the
  repository's own `cmd/` or `internal/` trees.
- **Rationale:** benchmark tooling stays out of the product's binary namespace, which holds only the two
  product binaries today, and Go's `internal/` visibility rule scoped under the ladder directory makes
  the package structurally unimportable from the product tree. One build, one path in the runbook.
- **Applies to:** all batches

### Decision: port fidelity and doc comments

- **Decision:** every ported unit keeps its Python counterpart's rationale in the corresponding Go doc
  comment. Where a field, gate, or metric changes meaning under the new dispatch model, the doc comment
  states what changed and why, rather than silently carrying the old wording.
- **Rationale:** the Python docstrings carry the reasoning for choices that look arbitrary from the code
  alone — the separated grep counters, the load-bearing section boundary, the liveness-not-existence
  keying of the cold gate. Losing that reasoning during a mechanical port is how a later change quietly
  breaks an invariant nobody can see any more.
- **Applies to:** all batches

### Decision: committed prompt and rule text is reproduced byte for byte

- **Decision:** the preamble constants, the two scoring rules, and every other piece of committed prompt
  text are reproduced exactly, with no rewording, reflowing, or tidying.
- **Rationale:** this text is the measured stimulus. Any drift in it changes what the matrix measures
  while leaving every test that does not compare the literal still passing.
- **Applies to:** ladder-config, scoring

### Decision: the harness never dispatches; the session does

- **Decision:** nothing in the Go tree spawns a model call. The subprocess dispatch, argv assembly, and
  stage drivers have no counterpart — their place is taken by the subcommand surface plus the tracked
  orchestration skill.
- **Rationale:** the dispatch tool exists only inside a live session, which is the whole point of the
  swap: one supervised run per session, killable on sight. A Go binary that dispatched would reintroduce
  the unwatchable subprocess the task exists to remove.
- **Applies to:** all batches

### Decision: no smoke launch is performed by this plan

- **Decision:** the single non-matrix session launch the discussion put in the `prepare-session` batch
  is **not** performed. This plan takes the discussion's own named default instead: the two
  implementation risks — the setting-source flag combination that isolates settings while still loading
  project-local agent definitions and user-scope skills, and whether a server declaration's environment
  block replaces or augments the inherited environment — move to the follow-up matrix task, and the
  fallbacks for each ship documented but unverified, stated in the README rather than left implicit.
- **Rationale:** the smoke launch is a supervised interactive session the operator watches, which an
  autonomous implementation run cannot carry out. Recording the risks as unverified is honest; having an
  implementer claim they were settled would not be.
- **Applies to:** cli-session-commands, skill-docs-cleanup

### Decision: three verification tiers, and what each one is for

- **Decision:** every batch's own `verify:` is `go test ./bench/loomyard-eval/ladder/...`, which runs
  the tests. The module-wide check at each batch boundary is `go vet ./...`, which type-checks every
  package in the module but runs no tests. The repo-wide test run is neither of these: it is the
  pipeline's configured done gate, executed by the orchestrator from the repository root before the
  task is marked done, and no batch invokes it.
- **Rationale:** the three tiers answer three different questions. The batch verify asks whether this
  batch's own code works, and scoping it to the ladder subtree keeps it fast, since nothing outside
  that subtree imports the package. The module-wide vet asks whether anything else in the module
  stopped compiling — worth checking at every boundary precisely because it is cheap, and worth being
  repo-wide because a subtree-scoped vet could not answer that question at all. The done gate asks
  whether the repository's full suite still passes, which is worth exactly once, at the end.
- **Applies to:** all batches

### Decision: test seams substitute processes, never mock the logic

- **Decision:** tests substitute the git runner, the build invocation, and the server-spawning seam, and
  assert on the argument vectors those seams receive. No test mocks a ported unit's own logic, and no
  test spawns a real daemon, builds a real binary, or touches a real pinned worktree.
- **Rationale:** this is the division the Python suite already drew — pure units unit-tested, the
  dispatch layer exercised by running the matrix. Under the new design the dispatch layer is the skill
  plus the session, which no Go test can drive, so the Go tests cover strictly more of the binary than
  the pytest suite covered of the Python.
- **Applies to:** all batches

### Decision: repo-wide done gate is unchanged

- **Decision:** the configured repo-wide done gate stays as it is. No lint command is added to it.
- **Rationale:** a Go linter is not installed in this environment, so defaulting the gate to one would
  make every future task in this hub depend on a tool that is not present. The gate's existing
  repo-wide test command is what the tier decision above relies on running once at the end.
- **Applies to:** all batches

## All Files Touched

- `.claude/skills/ladder-run/SKILL.md`
- `.gitignore`
- `CLAUDE.md`
- `bench/loomyard-eval/ladder/README.md`
- `bench/loomyard-eval/ladder/cmd/ladderbench/coldcell.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/coldcell_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/ingest.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/ingest_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/invalidate.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/invalidate_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/main.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/nextrun.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/nextrun_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/proberecord.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/proberecord_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/recordscore.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/recordscore_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/redact.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/redact_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/restoreworktree.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/restoreworktree_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/root_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/summarize.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/summarize_test.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/warm.go`
- `bench/loomyard-eval/ladder/cmd/ladderbench/warm_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
- `bench/loomyard-eval/ladder/internal/ladder/agentdef_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/coldcell.go`
- `bench/loomyard-eval/ladder/internal/ladder/coldcell_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/correlate.go`
- `bench/loomyard-eval/ladder/internal/ladder/correlate_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
- `bench/loomyard-eval/ladder/internal/ladder/daemon_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
- `bench/loomyard-eval/ladder/internal/ladder/fenced_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- `bench/loomyard-eval/ladder/internal/ladder/ladder_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/lock.go`
- `bench/loomyard-eval/ladder/internal/ladder/lock_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/plan.go`
- `bench/loomyard-eval/ladder/internal/ladder/plan_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/preamble.go`
- `bench/loomyard-eval/ladder/internal/ladder/preamble_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/precondition.go`
- `bench/loomyard-eval/ladder/internal/ladder/precondition_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/score.go`
- `bench/loomyard-eval/ladder/internal/ladder/score_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/server.go`
- `bench/loomyard-eval/ladder/internal/ladder/server_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/session.go`
- `bench/loomyard-eval/ladder/internal/ladder/session_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/settings.go`
- `bench/loomyard-eval/ladder/internal/ladder/settings_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
- `bench/loomyard-eval/ladder/internal/ladder/summarize_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/task.go`
- `bench/loomyard-eval/ladder/internal/ladder/task_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/bundle-mixed-tools.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/cold-native-fallback.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/denied-attempt.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-answer-fasit.json`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-ladder.yaml`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-transcript.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/errored-tool-result.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/none-target-origin-mention.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/targetdir-override.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/zero-tool-calls.jsonl`
- `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
- `bench/loomyard-eval/ladder/internal/ladder/transcript_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/usage.go`
- `bench/loomyard-eval/ladder/internal/ladder/usage_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/warm.go`
- `bench/loomyard-eval/ladder/internal/ladder/warm_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
- `bench/loomyard-eval/ladder/internal/ladder/worktree_test.go`
- `bench/loomyard-eval/ladder/ladder.yaml`
