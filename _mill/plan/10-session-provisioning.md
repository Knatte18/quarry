# Batch: session-provisioning

```yaml
task: Port the capability-ladder bench harness to Go
batch: session-provisioning
number: 10
cards: 8
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [6, 8, 9]
```

## Batch Scope

Builds everything a session needs before it is launched: the generated agent definitions that carry the
tool allowlist, the two launch preconditions, the cross-session lockfile, and the three session
materialisers. This is the batch that replaces `build_argv` and `launch_run` — the `claude -p` flag
assembly has no counterpart, because per-rung restriction is now a static subagent-type definition
rather than a per-call flag.

The external interface the CLI batches consume is `RunAgentDefinition`, `ScorerAgentDefinition`,
`ProbeAgentDefinition`, `CheckEnvironmentPrecondition`, `ScanSkillsForLeak`, `AcquireSessionLock`,
`ReleaseSessionLock`, `InstallSkill`, `PrepareRunSession`, `PrepareScoringSession`,
`PrepareProbeSession`, and `LaunchCommand`.

Batch-local decisions. First, the tool allowlist in a generated definition is the primary enforcement
layer and the settings deny-list is the backup: an allowlist is structurally stronger, since a blinded
arm never sees the prefixed namespace at all rather than seeing it and being denied. Second, a session
type never carries the other type's definition — a run session gets the run agent definition only, a
scoring session gets the scorer definition only. Third, `InstallSkill` takes its source path as a
parameter, so this batch can land and be tested before the tracked skill file itself exists.

## Cards

### Card 43: Model alias mapping and the run agent definition

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/internal/ladder/preamble.go`
  - `bench/loomyard-eval/ladder/internal/ladder/settings.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `ModelAlias(modelID string) (string, error)` mapping a pinned model id to the
  agent-definition alias — the opus id to `opus` and the sonnet id to `sonnet` — and erroring on an
  unmapped id. Its doc comment must record that the mapping is verified empirically by the model
  pinning gate on the first real run rather than sitting unchecked. Add
  `RunAgentDefinition(l *Ladder, c LadderConfig, runModel string) (name, body string, err error)`
  producing a Claude Code agent definition: YAML frontmatter carrying a name, a description, the
  resolved model alias, the configured run effort, and a `tools:` allowlist of `Read`, `Grep`, `Glob`,
  `Bash` plus that config's allowed quarry tools under their prefixed client-side names. The allowlist
  never contains `Task`, which makes the old uniform `Task` denial structural — a run agent cannot
  spawn its own subagents. The doc comment must record that. For a config whose allowed set is empty
  the definition's name, description, and body must contain no case-insensitive "quarry" and no
  prefixed name at all, because the definition sits in the blinded agent's own cwd. Test that a blinded
  config's definition contains no case-insensitive "quarry" anywhere, that a rung's allowlist is
  exactly its allowed tools prefixed plus the four base tools, that no definition grants `Task`, and
  that an unmapped model id errors.
- **Commit:** `feat(ladder): generate the run agent definition with its tool allowlist`

### Card 44: Scorer and probe agent definitions

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/internal/ladder/settings.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `ScorerAgentDefinition(l *Ladder) (name, body string, err error)` producing the
  single scorer definition, identical for every config since only the scorer model and effort
  parameterise it, pinned to the opus alias at the configured effort and granting **no tools at all**.
  Its doc comment must state that a zero-tool scorer is what makes one shared scoring session safe
  across the whole matrix: it cannot read the session transcript accumulating the other runs' answer
  keys, or anything else on disk. Add
  `ProbeAgentDefinition(l *Ladder, kind string) (name, body string, err error)` for the two probe
  kinds: the allowlist probe grants the four base tools but not the prefixed impact tool, while the
  deny-list probe does grant it. Both ask the agent to call that tool and report verbatim what
  happened. Test that the scorer definition grants nothing, that the allowlist probe's definition omits
  the prefixed impact tool while the deny-list probe's includes it, and that an unknown probe kind
  errors.
- **Commit:** `feat(ladder): generate the scorer and probe agent definitions`

### Card 45: The environment precondition

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/precondition.go`
  - `bench/loomyard-eval/ladder/internal/ladder/precondition_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `CheckEnvironmentPrecondition(env []string) error`, hard-failing when either
  `QUARRY_STATE_DIR` or `QUARRY_BUILD_TAGS` is set non-empty in the operator's own shell, naming the
  offending variable. This is the second of the three points the environment scrub applies at, and it
  exists partly as cover for the unestablished question of whether an MCP server declaration's
  environment block replaces or augments the inherited environment — the doc comment must say so, and
  must state that it checks the operator's shell at launch rather than the environment the server
  actually receives. An empty or absent value passes. Test rejection of each variable set non-empty,
  and acceptance when each is empty and when each is absent.
- **Commit:** `feat(ladder): add the environment launch precondition`

### Card 46: The skill-listing leak precondition

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/precondition.go`
  - `bench/loomyard-eval/ladder/internal/ladder/precondition_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `ScanSkillsForLeak(roots []string) (report ScanReport, offenders []string, err error)`
  and `DefaultSkillRoots() []string`. Every subagent transcript carries a record enumerating the
  session's available skills by name and description verbatim, with no tool call involved, so a
  skill's frontmatter is a leak channel into a blinded run agent's transcript that no working-directory
  hygiene can close — say that in the doc comment. The scan covers both the user-scope skills root and
  the plugin-cache root, because installed skills actually live under the plugin cache on this machine
  and scanning the user-scope root alone would pass vacuously. A root that does not exist is skipped
  rather than erroring, and `ScanReport` records every root scanned with the file count found at each
  so a vacuous pass is visible rather than silent. The scan is advisory for a rung and hard-failing for
  a config whose allowed set is empty; the caller applies that distinction. Test that a
  quarry-mentioning frontmatter under either root is reported as an offender, that an absent root is
  recorded as skipped rather than erroring, that a clean tree yields no offenders, and that the report
  names every root with its count.
- **Commit:** `feat(ladder): add the skill-listing leak precondition scan`

### Card 47: The cross-session lockfile

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Edits:**
  - `.gitignore`
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/lock.go`
  - `bench/loomyard-eval/ladder/internal/ladder/lock_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `AcquireSessionLock(resultsRoot, label string) error` and
  `ReleaseSessionLock(resultsRoot string) error`, using a `.session-active` file at the results root
  that names the session it was taken for. Acquiring refuses while a lock naming another session
  exists; releasing an absent lock is not an error. The doc comment must state that this is a guard
  over operator discipline rather than a proof — an operator who deletes the file or launches from a
  second checkout defeats it — and that a pid-based lock is not usable because a session outlives every
  process the harness starts. Add a `bench/loomyard-eval/ladder/results/**/.session-active` ignore
  entry beside the existing per-run artifact rule, with a comment recording that the lock is live
  session state and never part of a committed result. Test that a second acquire for a different label
  fails, that acquire succeeds after a release, that releasing an absent lock is not an error, and that
  re-acquiring the same label is idempotent.
- **Commit:** `feat(ladder): add the cross-session lockfile`

### Card 48: Run session materialisation

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
  - `bench/loomyard-eval/ladder/internal/ladder/settings.go`
  - `bench/loomyard-eval/ladder/internal/ladder/server.go`
  - `bench/loomyard-eval/ladder/internal/ladder/plan.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/internal/ladder/session_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add
  `PrepareRunSession(l *Ladder, c LadderConfig, n int, serverPath, targetDir string) (SessionInputs, error)`
  writing a run session's scratch directory: the settings document always, the run agent definition
  always, and the MCP server declaration **only when the config's allowed set is non-empty**. A config
  whose allowed set is empty gets no server declaration file and is launched with no server flag
  whatsoever, because a declared server named `quarry` exposing a prefixed namespace is itself the
  structural leak the blinding forbids — port that rule verbatim from `write_run_inputs` and record it
  in the doc comment. The scratch directory never receives the scorer definition, and never receives
  the skill. Test that a blinded config's scratch directory contains exactly the definition and the
  settings document and nothing else, that its settings deny exactly the task-spawning tool and name no
  quarry tool, that a rung's directory additionally holds the server declaration, and that neither
  directory holds a scorer definition.
- **Commit:** `feat(ladder): materialise a run session's scratch directory`

### Card 49: Scoring and probe session materialisation

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
  - `bench/loomyard-eval/ladder/internal/ladder/settings.go`
  - `bench/loomyard-eval/ladder/internal/ladder/plan.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/internal/ladder/session_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `PrepareScoringSession(l *Ladder) (SessionInputs, error)`, writing a scratch
  directory holding the scorer definition and a settings document denying only the task-spawning tool —
  no server declaration, no run agent definition, no worktree setup, and no server build. Add
  `PrepareProbeSession(l *Ladder, kind string) (SessionInputs, error)` writing each probe's bespoke
  definition and settings document into its own scratch directory: the allowlist probe's session
  declares the server while its definition withholds the tool, and the deny-list probe's definition
  grants the tool while its settings deny it. The doc comment must record that the two layers are only
  independently established if each is probed with the other neutralised, and that generating the probe
  inputs is in scope here while dispatching them is not. Test both session types' exact write lists,
  and that the scoring session's directory holds no run agent definition and no server declaration.
- **Commit:** `feat(ladder): materialise the scoring and probe sessions`

### Card 50: Skill installation and the launch command

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/plan.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/internal/ladder/session_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `InstallSkill(sourcePath, destRoot string) (string, error)` copying the tracked
  orchestration skill into the user-scope skills root, taking both paths as parameters so it is
  testable before the tracked skill file exists and so the destination is not hardcoded. It is called
  for every session type uniformly and must never write into a scratch directory — the skill body names
  quarry throughout, and a blinded session's scratch directory is that agent's own working directory.
  The doc comment must say so. Add `LaunchCommand(inputs SessionInputs) string` returning the exact
  command line the operator runs, including the server flag only when the session has a server
  declaration. Test that installation creates the destination tree and overwrites an existing copy,
  that it never writes into the scratch directory, and that the launch command omits the server flag
  for a blinded session and includes it for a rung.
- **Commit:** `feat(ladder): install the orchestration skill and print the launch command`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `agentdef_test.go`,
`precondition_test.go`, `lock_test.go`, and `session_test.go` plus every other test file in the ladder
subtree. Every session-materialisation test writes into the test's own temp directory and asserts the
exact file set produced, since "which file exists in which session type" is the whole containment
argument.
