MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5
reviewed_file: /home/knatte/Code/quarry/wts/port-ladder-bench-to-go/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:consistency] Deferred `none` scoring breaks single-flight
**Section:** §Single-flight dispatch vs §Tool exposure ("Therefore a `none` config gets one session per repetition")
**Issue:** `ingest` must fail when rep `n-1` has "neither a `run.json` nor an exhausted attempt record", but `run.json` is written by `record-score` (confirmed: `execute_run` writes it only after scoring, `gates.is_complete` requires `state == "complete"`), which for a `none` config is deferred to a later session — so `none` reps 2 and 3 hard-fail at `ingest`, and `next-run`'s resume filter re-offers rep 1 as pending in every subsequent run session.
**Fix:** State the predicate's `none`-config exemption (or an intermediate on-disk marker written by `ingest`) and say which of `run.json` / `is_complete` / resume filtering it changes.

### [BLOCKING:consistency] `{n}` rule does not cover per-rep `none` sessions
**Section:** §`ladder.yaml` additions ("`{n}` is the session index within the config")
**Issue:** `{n}` is defined as "1 for a warm session (a config has exactly one), and the repetition index for a cold session", with `--rep` taken "for a cold config and defaults `{n}` to `1` otherwise" — but a `none` config now has three run sessions, which would all collide on `/tmp/ladder-session-a0-none-1`, breaking the disposability claim and `ingest`'s one-directory search space for exactly the reason `{n}` was introduced. §One session per config's "`--rep <n>` where a session covers one repetition" already contradicts it, and the Testing section repeats the stale rule.
**Fix:** Restate `{n}` as the session index for any config with more than one session, and make `--rep` required for `none` as well as cold.

### [BLOCKING:decision] `none` scoring session has no stated provisioning
**Section:** §Tool exposure, §Subcommand surface
**Issue:** The dedicated `none` scoring session is named but never given a disposition: no `prepare-session` flag materialises it, no scratch dir is derived for it, and its interaction with the `.session-active` lockfile and with `require_pins` is unstated — `prepare-session`'s enumerated flags are `--config-id` / `--rep` / `--probe` / `--release` / `--run-model` only.
**Fix:** Name the invocation that prepares a scoring session, its scratch dir, and which agent definitions it writes.

### [BLOCKING:design] Fasit channel closed for `none` only, not for rungs
**Section:** §Tool exposure ("The parent session transcript is a third channel, and it carries the fasit")
**Issue:** The 23-session split rests on rep 1's fasit sitting within `Read` range of reps 2–3 in the same session; the 12 rung sessions run 3 reps with scoring inline and have the identical exposure, and rung run agents hold `Read`/`Bash`. The stated reason ("those sessions are not blinded, so there is nothing to leak") conflates tool-provenance blinding with answer-key exposure, and `gate_blinding` is applied only when `config.allowed` is empty (verified in `gates.py` `run_gates`) — so nothing detects it for a rung, and the inflated score is the primary measured outcome.
**Fix:** Either apply the same per-repetition split to rung sessions or state explicitly why fasit exposure is acceptable for a rung and what detects it.

### [BLOCKING:consistency] Q&A log carries unmarked superseded entries
**Section:** §Q&A log
**Issue:** "How many sessions? A: 17 — 14 warm plus 3 cold" contradicts the decided 23 and is not marked superseded (unlike the skill-delivery entry, which is); "What is a session's step order?" answers "…record-score, restore-worktree, next rep", contradicting §The per-repetition session loop, which puts `restore-worktree` immediately after `ingest` and before `redact` — the placement `run_matrix` actually uses and which the loop's own rationale depends on.
**Fix:** Mark both entries superseded with the current values, as the skill-delivery entry already is.

### [NIT:decision] `/tmp/quarry-bench` disposition given only for the gate
**Section:** §`gate_blinding` under the new topology
**Issue:** The dead literal is explicitly dropped from `gate_blinding`, but the same literal is an alternation branch in `score_run.py`'s `redact_text` (line 61) and is listed under what ports; the Testing section then asks for "every alternation branch" to be tested.
**Fix:** State whether the redaction branch is kept or dropped alongside the gate check.

### [NIT:scope] Scorer definition written into a blinded arm's cwd
**Section:** §Tool exposure ("exhaustively"), §Session launch
**Issue:** A `none` run session does no scoring, yet its scratch dir is specified to contain the scorer agent definition, which weakens the minimal-surface argument the exhaustive list exists to make.
**Fix:** Say whether the scorer definition is written only for sessions that dispatch a scorer.

## Verdict

REQUEST_CHANGES
The `none` per-repetition split is under-specified and contradicts single-flight, `{n}`, and the fasit rationale.
MILL_REVIEW_END
