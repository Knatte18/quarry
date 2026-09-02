---
name: ladder-run
description: Drive one benchmark session's dispatch, ingest, and score loop.
---

# ladder-run

This skill drives one live session end to end for the capability-ladder benchmark harness. It never
names the harness's own target server here or in its description — every subagent transcript enumerates
a session's available skills by name and description verbatim, with no tool call involved, so this
file's own frontmatter is itself a leak channel a blinded run agent's transcript would otherwise carry.

The operator, not this skill, owns launching each session. A session is the whole supervision unit: a
rogue dispatch is killed by closing the one session it is running in, nothing more surgical is provided
or needed.

There are two session shapes this skill drives, never both in the same session:

- A **run session**, one per repetition of one config, running exactly one dispatched run agent.
- The single shared **scoring session**, running one dispatched scorer per ingested-but-unscored run.

## Single-flight rule

Exactly one dispatch is in flight at a time, in either session shape. Never start a second dispatch
before the first one's ingest/record step has finished. This is mechanically checked on the Go side (the
run session's `ingest` step refuses to run out of order), but the operator must still never fire a second
dispatch concurrently with the first — the check catches an out-of-order write, not a live race.

## Run-session attempt loop

For a run session prepared with `prepare-session --config-id <id> --rep <n>`:

```
for attempt k in 1..MAX_ATTEMPTS:
  warm                              # skipped entirely for the cold config -- see below
  dispatch one run agent            # description: ladderbench's own DispatchDescription
                                     # (config id, repetition, attempt), so ingest can find it later
  ingest --config-id <id> --rep <n> # correlates the transcript, extracts usage/answer, runs the gates
  restore-worktree --config-id <id> --rep <n>   # unconditional, every attempt, whatever the outcome
                                                 # -- never called for the cold config, see below
  if ingest reported truncated: outcome = truncated; halt the whole matrix, do not retry; break
  if ingest reported failed:    invalidate --config-id <id> --rep <n>; go to the next attempt
  outcome = ingested; break      # ingest reported "ingested" -- this repetition's run session is done
if the loop ran out of attempts without breaking: outcome = exhausted
prepare-session --release
write outcome (exactly the string "ingested", "truncated", or "exhausted", nothing else) to
  <scratch-dir>/.ladder-run-outcome        # with Bash: printf '%s' ingested > .ladder-run-outcome
```

`restore-worktree` runs unconditionally, immediately after `ingest` and before the loop even looks at
the outcome, because it is what erases a dirtied worktree — and `ingest` must take that worktree
observation before the restore, not after, since the restore is precisely what would erase the evidence.

A `truncated` outcome from `ingest` is never retried. `max_turns` is a matrix-wide constant, so a second
attempt would hit the same ceiling identically; halt the matrix rather than burn another attempt on it.

Releasing the session lock (`prepare-session --release`) is always the session's last step before the
outcome-file write below, whether the repetition finished, was truncated, or exhausted its attempts. A
held lock blocks every later session from starting.

**Always write `<scratch-dir>/.ladder-run-outcome` as the session's actual final step, after the lock is
released, exactly once, for every terminal state, and write it with Bash (`printf '%s' <outcome> >
.ladder-run-outcome`), never with the Write or Edit tool.** This session's settings allow Read, Grep, Glob,
and Bash only; a Write call is not on the allow list, so it stops on a permission prompt that nobody is
watching, and the whole matrix waits behind it (observed 2026-09-02, a1-toc-file rep 3). A `claude` session never exits on its own once it
finishes responding — it waits for the next human message indefinitely, the same as any other interactive
session. An operator driving this by hand does not need the file (they can just read this session's own
final message and close it, or type the next prompt themselves), but an automated driver polling for
"is this session done, and did it succeed" from outside has no other reliable signal to watch for short of
scraping this session's own transcript, which this file exists to make unnecessary. Do not write it for
any other reason, and do not write it more than once per session.

### The cold config

`warm` is skipped entirely for the cold config: a cold repetition's whole premise is that its worktree
starts with no resident daemon, and warming it up is precisely what would start one.

The cold config's failure path is a **full session relaunch**, never an in-session retry. A cold
session's target MCP server process lives for the whole session, so its daemon survives
across attempts within one session — re-clearing the state directory mid-session to retry would delete
the state file the cold-before precondition reads the daemon's pid from, making a still-live daemon
invisible to it. A second attempt would then be *reported* cold while actually running against the
daemon the first attempt started, which is exactly the confound the cold cell exists to rule out. So on
a cold-before precondition failure, `prepare-session` itself records the abort and invalidates the
repetition; the operator's response is to close the session and run `prepare-session --config-id
<cold-id> --rep <n>` again for the next attempt, not to retry anything inside the session that just
failed. `restore-worktree` is never called for the cold config either: a cold repetition's worktree is
disposable per attempt and torn down outright (`cold-cell --teardown --rep <n>`), never reset in place.

## Scoring-session loop

For the single shared scoring session, prepared once with `prepare-session --scoring`:

```
for each run directory with an ingest marker and no run marker, in (config, rep) order:
  redact --config-id <id> --rep <n>       # writes the redacted answer, prints the scorer prompt
  dispatch the scorer agent with that printed prompt
  record-score --config-id <id> --rep <n> # reads the scorer's reply from stdin, writes score.json,
                                           # and -- once the run's artifact set is complete -- run.json
prepare-session --release
write "scored" to <scratch-dir>/.ladder-run-outcome   # with Bash, as above -- never the Write tool
```

Same reason and same rule as the run-session loop's own outcome-file step: write it once, as the actual
final step, after the lock is released.

`next-run --scoring` reports the next pending run in this order; iterate it until nothing is pending.
Every run in this loop was ingested by a run session that has already ended, so this session never
overlaps a run session's own dispatch — no session ever contains both a run agent and a scoring prompt,
because the scorer prompt embeds the task's unstripped answer key, and a run agent that saw that would
be scoring itself.

The scoring session is releasable the same way a run session is: `prepare-session --release` as the
final step, once every pending run has been scored.

## What this skill does not do

It does not invoke a dispatch tool itself — the operator's live session does that, once per loop
iteration above, using the correlation description or scorer prompt this skill's steps produce. It does
not decide which session to launch next; that is the operator's own scheduling call, subject only to the
single-flight rule and the cross-session lock the `prepare-session`/`--release` pair enforces.
