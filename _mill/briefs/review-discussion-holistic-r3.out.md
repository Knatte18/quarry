MILL_REVIEW_BEGIN
# Review: Per-capability quarry-mcp benchmark suite

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-4-x class model (Claude, Anthropic); exact build id not self-observable
reviewed_file: _mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:consistency] grep metric defined two ways
**Section:** § Metrics vs § Testing (`extract_usage.py`)
**Issue:** Metrics decision states #006's `grep_fallback_count` covers Bash `"command"` fields only (`bash_grep_count`) and keeps `grep_tool_count` strictly separate — verified: `bench/loomyard-eval/README.md` "Dispatch protocol" step 4 greps only Bash commands — but Testing specifies the test scenario as "the `grep_fallback_count` definition (Grep tool calls *plus* Bash commands containing `grep`/`rg`) matching #006's", which is the combined field, and the Q&A log still names `grep_fallback_count` as a reported field.
**Fix:** Fix the Testing scenario and Q&A entry to use the three-field vocabulary (`bash_grep_count` = #006-comparable, `grep_tool_count`, derived `grep_fallback_total`) so the TDD target is unambiguous.

### [BLOCKING:design] Cold cell's supervised check has no signal to read
**Section:** § Daemon warmth ("confirms from the server's own stderr/state that the connection is supervised")
**Issue:** `cmd/quarry-mcp/main.go` writes exactly one stderr line (`quarry-mcp: resolved target directory %s`) and nothing under `internal/` writes to stderr at all; `ConnKind` is never surfaced to any client-observable output, so the stderr half of the mechanism cannot exist and only an inferred state-artifact check (supervised writes `daemon.json`/daemon log under the state dir, native writes none) is available.
**Fix:** State the one concrete, source-grounded detection mechanism the harness uses for supervised-vs-native, given quarry may not be modified.

### [BLOCKING:scope] Scoring step has no artifact, script, or gate
**Section:** § Correctness scoring vs § Layout vs § Runs dispatch sequentially
**Issue:** Scope promises "a scoring step", but the layout names only `run_ladder.py` / `extract_usage.py` / `summarize.py`, no per-run score file is listed beside `usage.json`, `run.json`'s completion criteria are answer-parsed + usage-extracted + gates-passed with scoring absent, and `summarize.py`'s tests cover only medians/ranges — so recall/precision/`decoy_admitted`/`lookalikes_matched` have no producer, no storage location, and no resumability semantics.
**Fix:** Name where the 45 blinded scoring calls are driven from, what file their output lands in, and whether a run counts complete before it is scored.

### [BLOCKING:design] Shared warm worktree is mutable across 18 runs
**Section:** § Technical context ("builds each task's worktree once and shares it read-only across that task's runs")
**Issue:** Read-only is asserted, not structural: every run has Bash in its allow-set and the worktree is an ordinary writable checkout, so a run that writes scratch files or runs `go build` leaves state visible to the next 17 runs of that task — the same cross-run contamination the blind-process decision exists to eliminate — and no validation gate checks worktree cleanliness between runs.
**Fix:** Decide the enforcement (per-run clean check / restore, or accept and justify) and add it to the per-run gate list.

### [NIT:decision] Pinned model id never named
**Section:** § Enforcement ("Run environment") and § Constraints
**Issue:** The pin mechanism and its gate are fully specified, but no model id is chosen, and 45-run cost and comparability to #006's orientation numbers both depend on it.
**Fix:** Name the id, or state explicitly that the operator sets it once in `ladder.yaml` before the matrix starts.

## Verdict

REQUEST_CHANGES
Four blocking issues: one metric contradiction, one unavailable signal, one missing scoring artifact, one unguarded shared worktree.
MILL_REVIEW_END
