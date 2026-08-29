MILL_REVIEW_BEGIN
# Review: Per-capability quarry-mcp benchmark suite — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: plan/
date: 2026-08-29
```

## Findings

### [BLOCKING:scope] METRICS drops every impact-only scoring field
**Location:** batch 5 / card 13 (and card 28) **Issue:** `METRICS` enumerates only `recall` and `precision` from `score.json`, yet the same card's median rule says "the impact-only fields on a Ladder-A run" are skipped per run — so `decoy_admitted`, `lookalikes_matched`, and `summary_matches` (all written by card 12) never reach `summary.json`, and card 28, which may source numbers only from `summary.json`, cannot report the `burler.go:373` decoy that task 04's own scoring notes call "materially worse than a missed real caller". **Fix:** state whether the three fields are summarised (and how a boolean/qualitative field yields a median/range) or explicitly excluded, and reconcile card 13's own missing-metric sentence with the list.

### [BLOCKING:design] No stage-selected harness entry point for cards 24-26
**Location:** batch 6 / card 22, batch 8 / cards 24-26 **Issue:** card 22's CLI runs probe → main matrix → cold cell in one invocation with no probe-only or matrix-only mode, but card 24 must "run the probe through the harness and commit its record" as its own card with its own commit, before card 25 starts the 42-run matrix. **Fix:** give card 22's CLI an explicit stage selector, or state in card 24 that the probe boundary is reached by interrupting the resumable run.

### [BLOCKING:consistency] `resolve_state_dir` halts on env the harness already scrubs
**Location:** batch 3 / card 9 vs batch 6 / card 17 **Issue:** card 9 makes `resolve_state_dir` raise `GateError` whenever `QUARRY_STATE_DIR` or `QUARRY_BUILD_TAGS` is set in the environment, with the rationale that an operator's tag set would deepen the state path — but card 17's `run_env()` removes both from every subprocess the harness launches, so the resolved path is correct and the raise is a spurious halt on any machine that has either exported. **Fix:** decide one disposition — either the gate inspects the scrubbed `run_env()` mapping, or card 9's rationale is corrected to say the raise guards the harness's own process only.

### [BLOCKING:decision] `max_turns` and the scorer pin carry no values and no gate
**Location:** batch 1 / card 2, batch 8 / card 24 **Issue:** `ladder.yaml` declares `max_turns:` and `scorer:` (`model`, `effort`) as pinned for the whole matrix but names no value for any of them; `require_run_model` and card 24's preconditions check only `run_model`, so a null `scorer.model` reaches `score_run`'s `--model` and an invented `max_turns` silently bounds all 45 runs. **Fix:** pin the three values in card 2, or extend the operator-prerequisite decision and card 24's precondition list to cover them.

### [BLOCKING:scope] Card 17 Context omits `gates.py` and the LSP handler source
**Location:** batch 6 / card 17 **Issue:** the card's `Requirements:` name `gates.resolve_state_dir` (the `warm_daemon` post-condition) and rest on the claim that `workspace_symbol`'s handler calls `resolveCall` once per call, but `bench/loomyard-eval/ladder/scripts/gates.py` and the file declaring that handler are in neither `Context:` nor `Edits:` — only `tools_toc.go` is, which documents the *negative* case. **Fix:** add both paths to card 17's `Context:`.

### [BLOCKING:design] "Cold cell not run" has no durable record
**Location:** batch 6 / card 21, batch 8 / cards 26-27, batch 9 / card 28 **Issue:** card 21 returns the not-run disposition to its caller only; nothing writes it into the results root, so `summary.json` sees a cell with zero complete runs — indistinguishable from an interrupted one — card 15's CLI exits non-zero on it, and card 27's "one exception" for a legitimately-absent cold cell is unimplementable, while card 28 is required to say which of the two reasons applied. **Fix:** have card 21 persist the disposition (e.g. a cold-cell record beside `probe.json`) and have card 15 read it.

### [BLOCKING:decision] Fasit free-text fields name quarry; only `_meta` is stripped
**Location:** batch 4 / card 12 **Issue:** `strip_fasit_meta` is justified because `_meta`'s `role` contains the word "quarry" beside an answer scrubbed of it — but the committed `04-.../c.json` puts "quarry impact on singlellm.go:39:2 (--within internal/shedadapters)" and similar in three `callers_to_update[].evidence` entries and one `excluded_lookalikes[].reason`, and the plan states no disposition for those fields. **Fix:** state whether the fasit's free-text fields are redacted, left verbatim, or deliberately exempt, and give the reason.

### [NIT:scope] `extract_usage`'s error paths have no named test
**Location:** batch 2 / card 7 **Issue:** the card specifies `TranscriptError` on a malformed line (naming the line number) and on a missing `init`/`result` event, but the enumerated tests cover only well-formed fixtures. **Fix:** name a test per raise.

### [NIT:consistency] Scorer dispatch omits the permission mode every other dispatch carries
**Location:** batch 4 / card 12 **Issue:** the default `runner` passes `--setting-sources ""`, `--strict-mcp-config`, `--model`, `--effort`, `--output-format json`, but not `--permission-mode dontAsk`, unlike the probe and every matrix run — a scoring call that reaches for a tool can block. **Fix:** add the flag or state why the scorer needs no non-interactive mode.

## Verdict

REQUEST_CHANGES
Seven blocking gaps: metric coverage, stage entry points, env-gate contradiction, unpinned config values.
MILL_REVIEW_END
