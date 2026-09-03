MILL_REVIEW_BEGIN
# Review: Ladder harness around headless claude -p (T2)

```yaml
duration_s: 203.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), accessed via the Claude Agent SDK
reviewed_file: /home/knatte/Code/quarry/wts/ladder-harness/_mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Gate 2 check (c) can never fire fatally
**Section:** gates, antecedent 2 **Issue:** Antecedent 2 is scanned "once per task and cached as a **boolean**", and the same paragraph states the token occurs in 12 tracked files at pin `975578cd` — so the boolean is `true` for both ladder-referenced tasks, every check-(c) hit is classified `target_origin`, and the "fatal only when it has none" branch is unreachable for the only pin the harness will ever run. **Fix:** state whether antecedent 2 is per-occurrence (match the emitted string against target content) or, if it stays a boolean, say plainly that check (c) is an observation and that (a)/(b) are the only fatal checks.

### [BLOCKING:design] Auto-memory check needs the mangling being deleted
**Section:** no-tmp-paths (auto-memory) vs Technical context "Deletes wholesale" **Issue:** The harness must "read the resolved auto-memory directory at startup", i.e. resolve `…/projects/-home-knatte-Code-loomyard-wts-loomyard/memory/`, but the only stated way to derive that path is `correlate.go`'s "empirically-derived dot-to-hyphen directory mangling", which the same document deletes wholesale and forbids porting. **Fix:** name the resolution mechanism (port the mangling deliberately, read `memory_paths` from a cheap probe invocation, or take the directory as configuration) and say what happens when it cannot be resolved.

### [BLOCKING:design] Treatment-cell tool grant is unprobed
**Section:** claude-invocation **Issue:** Every probe covers the control shape (`--allowedTools` omitted); the treatment shape (`--tools Read,Grep,Glob,Bash` **plus** `--allowedTools mcp__<name>__toc`) is never probed, and §9a's only `--allowedTools` probe ran with `--tools ""`, so nothing shows built-ins still execute when an allowlist is present — if they do not, Bash/Read are denied in treatment and granted in control, the exact arm asymmetry one-preamble-for-every-cell exists to remove. The live smoke test runs `a0-none`, so T2 cannot catch it and T7 pays. **Fix:** probe the treatment invocation (or state the assumption and add a treatment-arm assertion to the smoke test / stub-MCP integration test).

### [NIT:decision] Gate-2-failed rep has no summary disposition
**Demoted-from:** BLOCKING
**Section:** resume-and-failure (failure taxonomy) / gates **Issue:** A gate-2 failure is "complete as a failed rep", but only `max_turns` and scorer-failed reps are excluded from medians and counted (`max_turns_count`, `unscored_count`); whether a leaked control rep's cost metrics enter the control cell's medians — the baseline every T7 contrast rests on — is unstated, as is any count for it in `summary.json`. **Fix:** state explicitly whether a gate-2-failed rep contributes cost metrics, and add its counter to `summary.json` alongside the other two.

### [BLOCKING:design] `report` has no stated source for cell metadata
**Section:** command-surface / Testing ("`report` over a fixture `raw/`") **Issue:** `ladder report --results <root>` takes no `--config`, yet re-deriving `summary.json` requires per-cell `allowed` (gate 1), the `ladder` letter and its control (`RangesDisjoint` comparison) and `incomplete[]` — none of which `raw/` is said to carry; the `run.json` payload's contents are never specified beyond `state`. **Fix:** decide the source (a `--config` flag, `provenance.json`'s `ladder_file`, or cell metadata embedded in `run.json`) and specify what the committed fixture root must contain.

### [NIT:decision] `LADDER_LOOMYARD_REPO` has no stated reader
**Section:** Technical context "Environment" / command-surface **Issue:** The value "comes from a gitignored `.scratch/ladder.env`", but shell wrappers are banned and the documented entry is a bare `go run …`, so nothing states whether the harness parses that file or the operator must export the variable. **Fix:** say which.

### [NIT:scope] `.gitignore` rule location unspecified
**Section:** Scope ("A `.gitignore` rule for `results/**/raw/`") **Issue:** The pattern contains a slash, so in the repo-root `.gitignore` it anchors to the root and would not match `bench/loomyard-eval/ladder/results/<root>/raw/`; the failure is silent and its consequence is tracked transcripts carrying machine paths, the one constraint results-raw-ignored exists to satisfy. **Fix:** name the file the rule goes in (or an anchor-free pattern).

### [NIT:decision] Task files' `## Scope` section has no stated disposition
**Section:** Technical context "Task file structure" / one-preamble-for-every-cell **Issue:** `## Scope` is enumerated as a section of every task file and is silently dropped by inclusion-based extraction, so instructions like task 01's "Do not read outside these two packages" never reach the agent; whether V1 included it is not recorded. **Fix:** state the disposition of `## Scope` and whether dropping it changes the measurement's continuity with `v1-final`.

## Verdict

REQUEST_CHANGES
Gate 2's fatal branch is unreachable, and three run-path mechanisms are unresolved.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 4._
MILL_REVIEW_END
