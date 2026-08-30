MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:scope] No card writes answer.json or usage.json
**Location:** batch 12 card 58 (`ingest`); cross-checked against batch 5 card 26 and batch 6 card 28
**Issue:** Card 58 says "extract usage ... parse the answer" but never states that either result is serialised into the run directory, and no card anywhere defines a write site; yet `GateRunCompleteArtifacts` (card 26) requires `answer.json` and `usage.json` by name and `WriteRedacted(runDir)` (card 28) reads `answer.json`. `Usage` is only described as "serialising to the `usage.json` shape" (card 10), which fixes the shape, not the writer.
**Fix:** State in card 58 that `ingest` writes `<run_dir>/usage.json` and `<run_dir>/answer.json`, naming the function or the exact filenames, the same way it names the ingest marker.

### [BLOCKING:decision] probe.json's denied_tools_advertised key has no disposition
**Location:** batch 7 card 34 (`BuildSummary`) vs batch 12 card 62 (`probe-record`)
**Issue:** `build_summary` sets `_meta.denied_tool_attempts_reported` from `probe.json["denied_tools_advertised"]` (summarize.py:368). Card 62 fixes probe.json's keys as `allowlist_blocks`, `denylist_blocks`, and `denial_shape_observed` — that key is gone. Card 34 ports `build_summary` without saying whether the meta field is dropped, retargeted at a probe key, or left reading a key nothing writes.
**Fix:** Give card 34 an explicit disposition for `denied_tool_attempts_reported` against the new probe.json shape.

### [BLOCKING:design] Cold preparation abort never advances the attempt index
**Location:** batch 11 card 53, read against batch 5 cards 24/25 and batch 12 card 63
**Issue:** A cold-before failure aborts preparation and writes `<n>.cold-abort-<k>.json` where `k` is the current attempt index, but no run directory exists yet, so `invalidate` cannot run and no `<n>.invalid-<k>` sibling is created. `NextAttempt` therefore returns the same `k` on every relaunch: repeated aborts overwrite one record, `MaxAttempts` never bounds them, and `ColdCellDisposition` (card 63) sees at most one live-daemon cause however many times it occurred.
**Fix:** State in card 53 what advances the attempt index (and what halts) on a preparation abort, or make the abort filename independent of `k`.

### [NIT:consistency] Card 7 asserts parity ExtractFencedJSON does not have
**Location:** batch 1 card 7
**Issue:** The card says the port keeps "the same semantics, including the error on no block found and on an unknown selector". Python's `extract_fenced_json` returns `None` on no block — deliberately, so each call site raises its own contextually-worded error (ladder_config.py:433-437) — and silently treats any non-`"first"` selector as `"last"`. Neither error exists in the source.
**Fix:** Reword as a deliberate change and say whether call sites (cards 30, 40, 58) still wrap it with their own contextual message.

### [NIT:consistency] Skill-leak scan is narrower than the channel it covers
**Location:** batch 10 card 46 (`DefaultSkillRoots`), batch 14 card 70
**Issue:** A real `skill_listing` record on this machine enumerates ~19 built-in/managed skills (`dataviz`, `claude-api`, `update-config`, `simplify`, `run`, `init`, `review`, `security-review`, …) that live under neither of the two fixed roots, so the `ScanReport`'s "every root scanned with its count" cannot make that portion visible. The two roots themselves are correct — `~/.claude/skills/` is genuinely empty and skills live under `~/.claude/plugins/cache/*/*/*/skills/*/SKILL.md`, both verified.
**Fix:** Have card 46's doc comment (and card 70's residual-leak-channel section) record that built-in skills appear in `skill_listing` from outside the scanned roots.

## Verdict

REQUEST_CHANGES
Three gaps: unwritten run artifacts, an orphaned probe.json key, and an unbounded cold abort.
MILL_REVIEW_END
