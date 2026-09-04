MILL_REVIEW_BEGIN
# Review: Ladder, toc rerun (T7) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic), high reasoning effort
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:scope] Retype breaks an existing string-array fixture
**Location:** batch 2 / cards 3–4 **Issue:** `internal/ladder/testdata/transcripts/tool-bytes.jsonl` line 1 carries `"mcp_servers":["quarry"]`, so after `SessionInit.MCPServers` becomes `[]MCPServerStatus` `ParseTranscript` errors on that line and fails `stream_test.go:86` plus `metrics_test.go:104/144/161`; the fixture appears in no card's `Edits:`/`Creates:` and not in `## All Files Touched`. **Fix:** give a card explicit ownership of updating that fixture (and any other non-empty string-array init line) to the object shape, and add it to `## All Files Touched`.

### [BLOCKING:design] Card 5's finding has nowhere to be recorded
**Location:** batch 2 / card 5 **Issue:** the card places `CheckServerConnected` inside the attempt loop and falls through to `InvalidateRep`, which renames the rep directory to `.invalid-<k>` and skips `writeCompleteState`, yet the same card requires "record the finding in the repetition's observations" — `observations` is only assembled after the loop (run.go:479) and an exhausted rep returns `repOutcome{incomplete: true}` with no state file at all. **Fix:** state where the finding is actually persisted (run log, per-attempt note, or a state file written before invalidation), and make card 9's "the reason is in that repetition's own state file" clause consistent with it.

### [BLOCKING:scope] "The required pin" is an unresolvable identifier in card 6
**Location:** batch 3 / card 6 step (3) **Issue:** the step requires the Loomyard checkout's HEAD to be "at the required pin" without naming it; `72c23d9` is stated only in `_mill/discussion.md` (line 478) and `HANDOFF.md`, neither of which is in card 6's `Context:`, and the one SHA the card's own context does carry — `ladder-toc.yaml`'s `pinned_sha: 975578cd…` — is a different pin. **Fix:** spell `72c23d9` literally in the requirement, or add the file that states it to card 6's `Context:`.

### [BLOCKING:scope] Named symbols from files absent from Context
**Location:** batch 2 / card 5; batch 3 / card 7 **Issue:** card 5's `Requirements:` names `InvalidateRep` and `MaxAttempts`, both declared in `internal/ladder/runstate.go`, which is in neither its `Context:` nor its `Edits:`; card 7 names a `BuildServer` failure while `mcp.go`, where `BuildServer` lives, is absent from its `Context:`. **Fix:** add `runstate.go` to card 5's `Context:` and `mcp.go` to card 7's `Context:`.

### [NIT:consistency] Live test is not "the first pre-matrix gate"
**Location:** `00-overview.md` Decision `the-done-gate-runs-offline` vs batch 1 / batch 3 card 6 **Issue:** the decision states the guarded live test "runs once, explicitly, as the first pre-matrix gate", but batch 1 — the batch actually named `pre-matrix-gates` — contains no live test, and the run is step (4) of card 6 in batch 3, after the offline suite, the clean-tree check and the environment preconditions. **Fix:** restate the decision as "the first step of the matrix batch's gate card" or move the ordering claim to match card 6's numbering.

### [NIT:consistency] Fixture-capture turn omits the mandated `env -u` prefix
**Location:** batch 2 / card 4, route (b) **Issue:** the card prescribes a direct `claude -p` turn to capture the init line, but does not carry the `env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT` prefix that `matrix-runs-backgrounded-under-env-u` makes mandatory for every harness/`claude -p` seam invocation this task makes. **Fix:** add the prefix to route (b)'s capture command, or state explicitly why an unmeasured capture turn is exempt.

## Verdict

REQUEST_CHANGES
Fixture breakage, an unrecordable gate finding, and two unresolvable context gaps.
MILL_REVIEW_END
