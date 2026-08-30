# Batch: cli-session-commands

```yaml
task: Port the capability-ladder bench harness to Go
batch: cli-session-commands
number: 11
cards: 6
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [10]
```

## Batch Scope

Creates the `ladderbench` binary and lands the four subcommands a session runs before and around a
dispatch: `prepare-session` in all its modes, `next-run`, `warm`, and `restore-worktree`. The command
tree is cobra-based and modelled on how this repo already defines one.

The external interface batch 12 consumes is the cobra root command and the shared ladder-loading and
results-root resolution helpers this batch establishes.

Batch-local decision: the smoke launch described in the discussion is **not** performed by this batch.
It is a supervised interactive session launch, which an autonomous implementation run cannot carry out,
so this plan takes the discussion's own named default: both implementation risks — the setting-source
flag combination that isolates settings while still loading project-local agent definitions and
user-scope skills, and whether a server declaration's environment block replaces or augments the
inherited environment — move to the follow-up matrix task, and the fallbacks ship documented but
unverified. The documentation batch is where that gets stated in the README rather than left implicit.

## Cards

### Card 51: The ladderbench command tree

- **Context:**
  - `internal/cli/cli.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/main.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create package `main` for the `ladderbench` binary with a cobra root command whose
  package doc documents the eleven-subcommand surface and why it is split where it is: only a live
  session can invoke the dispatch tool, so every boundary in this surface is a session boundary. Model
  the command-tree definition on the single place this repo already defines one. Add persistent flags
  for the ladder file path and the results root, and unexported helpers that load and validate the
  ladder, resolve the results root, and resolve the repository root, so no subcommand re-derives them.
  Every subcommand added in this batch and the next registers on this root. Test that the root command
  registers without error and that the ladder-loading helper surfaces a load failure rather than
  swallowing it.
- **Commit:** `feat(ladderbench): add the cobra command tree`

### Card 52: prepare-session for run, scoring, and probe sessions

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/internal/ladder/precondition.go`
  - `bench/loomyard-eval/ladder/internal/ladder/lock.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/server.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `prepare-session` subcommand with `--config-id`, `--rep`, `--scoring`,
  `--probe`, `--release`, and `--run-model` flags, where `--rep` is required for every run session of
  every config. The run-session path ensures the task worktrees are at their pins, builds the server
  binary, runs the environment precondition, runs the skill-leak scan — hard-failing only for a config
  whose allowed set is empty and naming the offending skill so the operator relocates it — takes the
  session lock, materialises the scratch directory, installs the orchestration skill, and prints the
  launch command exactly as the session-provisioning batch's fixed flag set defines it — this command
  assembles no flags of its own. The scoring path materialises the scoring session and takes the lock like any other
  session; the probe path materialises the named probe session. `--release` clears the lock and does
  nothing else. `--run-model` overrides the pinned run model for this invocation only and is never
  written back to the ladder file. `prepare-session` enforces the narrower session pin set rather than
  the full one. Test the flag validation matrix: a run session without a repetition errors, mutually
  exclusive modes error together, and the model override satisfies the pin check that would otherwise
  fail.
- **Commit:** `feat(ladderbench): add prepare-session for run, scoring, and probe sessions`

### Card 53: The cold-session preparation path

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Extend `prepare-session` so that preparing the cold config additionally drains the
  daemons the warm sessions left resident on both task worktrees using the bounded exit wait at its
  committed timeout, builds the fresh per-repetition worktree from the cold worktree template with the
  repetition substituted, clears the resolved state directory, asserts the cold-before gate, and points
  that session's server declaration at the new worktree. The doc comment must record that a failed cold
  attempt relaunches the whole session rather than retrying in place, because the session's server
  process outlives an attempt and re-clearing the state directory mid-session would delete the state
  file whose pid the cold-before gate reads, leaving a surviving warm daemon invisible. Test that the
  cold path is selected by the config's cold flag rather than by parsing its id, and that a failing
  cold-before gate aborts preparation before the lock is taken.
- **Commit:** `feat(ladderbench): add the cold-session preparation path`

### Card 54: next-run

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/plan.go`
  - `bench/loomyard-eval/ladder/internal/ladder/task.go`
  - `bench/loomyard-eval/ladder/internal/ladder/preamble.go`
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/nextrun.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/nextrun_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `next-run` subcommand. For a run session it prints this session's
  repetition if it is still pending, its current attempt index, its full assembled prompt, and its
  agent-definition name, and reports nothing pending otherwise. Under `--scoring` it prints the next
  run directory that has been ingested but not scored. The attempt index comes from the run-state
  helper that derives it from the invalidated siblings on disk, never from a session-held counter, so
  the index the correlation description embeds has one derivation site. `next-run` enforces the full
  pin set. Test the pending and nothing-pending paths, the scoring mode's filter, and the attempt index
  advancing with invalidated siblings present.
- **Commit:** `feat(ladderbench): add next-run`

### Card 55: warm

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/warm.go`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/warm.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/warm_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `warm` subcommand, calling the warm-up against the session's server binary
  and target worktree with the scrubbed environment, and reporting the post-condition failure as a
  non-zero exit. Its help text must state that it is skipped entirely for the cold config and that it
  runs once per attempt rather than once per repetition, because the daemon self-expires after its idle
  timeout and a retry that skipped it could dispatch against a cold daemon and silently contaminate the
  warm arm's timings. `warm` enforces the full pin set. Test that it refuses to run for the cold config
  and that a post-condition failure produces a non-zero exit.
- **Commit:** `feat(ladderbench): add warm`

### Card 56: restore-worktree

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/cmd/ladderbench/restoreworktree.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/restoreworktree_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the `restore-worktree` subcommand performing the unconditional post-attempt
  reset, clean, and re-neutralisation of the config's task worktree. Its help text must state that it
  runs after every attempt whatever the outcome, and that it is deliberately not used by cold sessions —
  it resets and re-neutralises a persistent worktree, which is the opposite of a cold repetition's
  disposable one. Test that it refuses to run for the cold config and that it invokes the restore for a
  warm one.
- **Commit:** `feat(ladderbench): add restore-worktree`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers the new command-package test files plus every
library test file, since the pattern matches both the binary and the library package. Subcommand tests
exercise flag validation and the branch each command selects; they substitute the process-spawning
seams rather than building a server, running git, or launching a session.
