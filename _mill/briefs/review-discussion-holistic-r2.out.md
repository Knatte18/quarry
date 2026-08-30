MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/port-ladder-bench-to-go/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:design] Scratch dir contents un-blind the `none` arm
**Section:** §"Tool exposure ... deny-list as backup" / §"The session's working directory is a neutral scratch dir"
**Issue:** The blinded run agent gets `Read`/`Grep`/`Glob`/`Bash` with cwd = the scratch dir, which by decision holds `.claude/settings.json` carrying all seven `mcp__quarry__*` deny names and `.claude/skills/ladder-run/SKILL.md` describing the quarry ladder; `gates.py:215` makes any `mcp__quarry__` substring anywhere in the transcript fatal, so one `cat .claude/settings.json` both leaks and burns an attempt — the claim that the deny list "costs nothing" holds only for enforcement, not for blinding.
**Fix:** Decide explicitly what a `none` session's scratch dir may contain (e.g. omit the deny-list/skill for `none`, or place them outside the agent's cwd) and state it in the enforcement decision.

### [BLOCKING:decision] Cold-session mechanics have no owning subcommand
**Section:** §"The cold cell" / §"Subcommand surface"
**Issue:** `wait_for_daemon_exit` + `DAEMON_EXIT_TIMEOUT_S` (run_ladder.py:938-939, draining resident daemons before the first cold repetition), `build_worktree` from `cold_worktree_template`, `clear_state_dir`, and post-run `remove_worktree` are all named as surviving, but none of the eleven subcommands is said to own them — `restore-worktree` is defined as reset/clean/re-neutralise, not removal.
**Fix:** Assign each cold-cell step to a named subcommand (or say `prepare-session --rep`/`cold-cell` absorb them).

### [BLOCKING:design] `session_dir_template` collides across cold repetitions
**Section:** §"`ladder.yaml` additions" / §"The subagent transcript format"
**Issue:** The template is keyed on `{config_id}` only, yet the three cold sessions share config id `a5-bundle-cold`, so all three reuse one scratch dir — contradicting "each session runs from a disposable scratch directory" and the claim that "each config's search space is one directory" for `ingest`'s `*/subagents/*.meta.json` glob.
**Fix:** State whether the template supports `{n}` (as `cold_worktree_template` does) and what a cold session's dir is.

### [BLOCKING:scope] Probe session inputs have no generation path
**Section:** §"Two preflight probes, not one" / §"Subcommand surface"
**Issue:** Both probes need bespoke sessions (one with the server declared and `impact` absent from the allowlist, one with `impact` allowlisted but denied in `settings.json`); `prepare-session` is defined only as `--config-id <id>` over `ladder.yaml` configs, and only `probe-record` (consumption) is in the surface, so nothing in scope generates the probe agent definitions and settings.
**Fix:** Name the subcommand/flag that materialises each probe session, or state explicitly that probe inputs are the follow-up task's work.

### [NIT:consistency] Superseded counts left in place
**Section:** §"One session per config", §"The cold cell", Q&A log
**Issue:** The decision says 17 sessions but its own rationale says "18 manual launches"; §"The cold cell" says "after all 15 config sessions" where 14 are non-cold; the Q&A carries both "18 total" and "Ten subcommands" against the eleven enumerated.
**Fix:** Normalise to 17 sessions / 14 warm / 11 subcommands throughout, including the older Q&A entries.

### [NIT:consistency] `run_model` shown pinned and shipping null
**Section:** §"One pinned model id per role" vs §"`ladder.yaml` additions"
**Issue:** The first quotes `run_model: claude-sonnet-5`; the second and `ladder.yaml:9` keep `run_model: null` by design, so a plan writer could commit a pinned value.
**Fix:** Mark `claude-sonnet-5` as an operator-supplied example, not the committed value.

### [NIT:consistency] Empty-vs-unset in the reconstructed gate environment
**Section:** §"Environment scrubbing under Agent dispatch" (point 3)
**Issue:** Gates pass `os.Environ()` with both keys "forced empty" while `resolve_state_dir` is said to hard-error on an environment "carrying either variable" — read literally, every gate errors; the Python uses `env.get()` truthiness (gates.py:297-300), and quarry itself treats `""` as unset (internal/cli/paths.go:123).
**Fix:** Say the Go check is non-empty-value, not key-presence.

## Verdict

REQUEST_CHANGES
Blinding leak via scratch dir, unowned cold/probe session mechanics, and a colliding session dir template.
MILL_REVIEW_END
