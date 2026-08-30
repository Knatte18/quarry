# Batch: skill-docs-cleanup

```yaml
task: Port the capability-ladder bench harness to Go
batch: skill-docs-cleanup
number: 14
cards: 6
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [13]
```

## Batch Scope

Lands the session-side orchestration skill, rewrites the suite's documentation around the new topology,
and removes the Python tree in the same batch that makes the grandfather note untrue. The skill comes
last because its body names every subcommand, and writing it before the surface exists would have meant
writing it twice.

Batch-local decision: the Python tree is deleted here rather than up front. Keeping a working reference
during the port beats porting blind against the README and git history, and deleting last means the
grandfather clause is removed in the same batch that earns its removal.

## Cards

### Card 68: The ladder-run orchestration skill

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/correlate.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/preparesession.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/ingest.go`
  - `_mill/discussion.md`
- **Edits:** none
- **Creates:**
  - `.claude/skills/ladder-run/SKILL.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the repo-tracked orchestration skill that drives one session end to end. Its
  frontmatter name and description must contain no case-insensitive "quarry", because every subagent
  transcript enumerates the session's available skills by name and description verbatim and a blinded
  run agent's transcript is scanned for exactly that substring. The body specifies the run-session
  attempt loop — warm, skipped entirely for the cold config; dispatch one run agent with the correlation
  description the binary defines; ingest; restore the worktree unconditionally; halt the matrix on a
  truncated outcome; invalidate and retry on failure; release the session lock as the final step — and
  the scoring-session loop of redact, dispatch the scorer, record the score, once per ingested-but-unscored
  run in order. It states the single-flight rule: exactly one dispatch in flight at a time. It states
  that the operator, not the skill, owns launching each session, and that a rogue run is killed by
  closing the one session it is in. The cold config's failure path is a full session relaunch rather
  than an in-session retry, and the skill must say why.
- **Commit:** `feat(ladder): add the ladder-run orchestration skill`

### Card 69: Skill and definition blinding-hygiene tests

- **Context:**
  - `.claude/skills/ladder-run/SKILL.md`
  - `bench/loomyard-eval/ladder/internal/ladder/agentdef.go`
  - `bench/loomyard-eval/ladder/internal/ladder/precondition.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/precondition_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a test asserting that the tracked orchestration skill's frontmatter name and
  description contain no case-insensitive "quarry", reading the tracked file itself rather than a copy,
  so the constraint is enforced against the artefact that actually ships. This is the mechanical half of
  the mitigation whose other half is the launch-time scan; the test comment must say so.
- **Commit:** `test(ladder): assert the orchestration skill's frontmatter is blinding-safe`

### Card 70: README rewrite

- **Context:**
  - `_mill/discussion.md`
  - `.claude/skills/ladder-run/SKILL.md`
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/session.go`
  - `bench/loomyard-eval/ladder/cmd/ladderbench/root.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/README.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rewrite the README in place as the self-contained protocol document for the new
  topology, keeping its existing section structure where it still applies. It must state: the session
  model of one session per repetition plus one shared scoring session, and why no session ever contains
  both a run agent and a scoring prompt; the two-layer enforcement model with the allowlist primary and
  the deny-list backup, and that a blinded session gets no server declaration at all; the metric
  partition, naming the dropped metrics and why a synthesised cost figure is worse than no number; that
  the denial-attempt metric ships provisional and what clears it; that the turn ceiling is now a
  post-hoc gate on assistant records and that the committed threshold was blanked because its
  accounting basis changed; that cross-session serialisation is an operator obligation with the
  lockfile catching only the ordinary mistake; that the matrix targets a pinned Linux checkout and is
  not expected to run on Windows; and the residual leak channels the harness cannot close, stated as
  hygiene rather than as a structural guarantee — naming among them that a session's skill listing
  enumerates built-in and managed skills from outside the two scanned roots, so the launch-time scan
  bounds only the installed skills it can see. It must also record the two implementation risks that
  remain unverified because no smoke launch was performed — the setting-source flag combination, and
  whether a server declaration's environment block replaces or augments the inherited environment —
  naming the documented fallback for each. Replace every reference to the deleted Python entry points
  with the corresponding subcommand.
- **Commit:** `docs(ladder): rewrite the README for the Go harness and session model`

### Card 71: ladder.yaml header refresh

- **Context:**
  - `bench/loomyard-eval/ladder/README.md`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Refresh the file's header comment block so it no longer names the deleted Python
  module as the thing that derives deny-lists and validates the tool list, naming the Go package
  instead, and so it no longer refers to a batch number from the previous task. It must state that two
  operator-supplied pins are now required before the matrix starts rather than one. Leave every
  configuration value untouched — this card edits comments only.
- **Commit:** `docs(ladder): refresh the ladder.yaml header comments`

### Card 72: Drop the grandfather clause

- **Context:**
  - `bench/loomyard-eval/ladder/README.md`
- **Edits:**
  - `CLAUDE.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Remove the ladder directory from the grandfathered-exception note, leaving the
  sibling suite's generator as the only remaining exception and dropping the parenthetical about the
  port task. The Go-only policy statement itself stays.
- **Commit:** `docs: drop the ladder grandfather clause now the port is complete`

### Card 73: Delete the Python tree

- **Context:**
  - `CLAUDE.md`
  - `bench/loomyard-eval/ladder/README.md`
- **Edits:** none
- **Creates:** none
- **Deletes:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_extract_usage.py`
  - `bench/loomyard-eval/ladder/tests/test_gates.py`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
  - `bench/loomyard-eval/ladder/tests/test_score_run.py`
  - `bench/loomyard-eval/ladder/tests/test_summarize.py`
- **Moves:** none
- **Requirements:** Delete every remaining Python file under the ladder suite's script and test
  directories, leaving both directories empty and therefore untracked. The fixtures those tests read
  were already relocated into the Go package's test data, so nothing else references them. After the
  deletion, confirm no file under the ladder suite still references a deleted module by name.
- **Commit:** `refactor(ladder): delete the ported Python harness`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers the blinding-hygiene assertion added to the
precondition tests plus every earlier test file in the ladder subtree. The documentation cards have no
runnable surface of their own; the deletion card is verified by the same run continuing to pass, which
is what demonstrates nothing in the Go tree depended on the Python.
