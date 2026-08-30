MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude (Anthropic) — Opus-class model, per the harness ID "claude-opus-5"
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:design] run.json observation chain is ported broken
**Location:** batch 5 cards 23/24, batch 7 card 31, batch 13 card 67
**Issue:** `write_run_json`'s payload stores observations as `observations: [{gate, message}]`, but `load_runs` lifts them with `if observation in run_record` over top-level keys `worktree_dirtied` / `target_origin_quarry_mention` / `cold_no_daemon_backed_call` — so in the Python they are never lifted, and `worktree_dirtied_count`, `daemon_backed_runs` and `compare_warm_cold`'s all-no-daemon suppression are dead. Cards 23/24 port the payload shape verbatim and card 31 ports `_RUN_JSON_OBSERVATIONS` verbatim, yet card 67 requires an assertion that "a non-fatal observation taken at gate time reaches the summarised cell".
**Fix:** state in card 24 (`RunJSONPayload`) or card 31 (`LoadRuns`) which of the two shapes changes, so the gate-report → ingest-record → run-marker → summary chain the e2e test asserts actually connects.

### [BLOCKING:design] Repo-root-relative paths break under Go's per-package test cwd
**Location:** batch 9 cards 39/40 (also card 60)
**Issue:** `ladder.yaml`'s `task_file` / `fasit` values and `_BENCHMARK_README_PATH = "bench/loomyard-eval/README.md"` are repo-root-relative; pytest resolved them only because it ran from the repo root. `go test` runs each binary with cwd = the package directory, and `TaskTextFor(l *Ladder, taskKey string)` / `SchemaFor(l *Ladder, taskKey string)` carry no repo-root parameter, so the tests those cards mandate ("test both real task files", read the benchmark README) cannot open their inputs.
**Fix:** name the repo-root resolution in cards 39/40 — a `repoRoot` parameter on both signatures, or an equivalent single derivation site the tests and card 60 both use.

### [BLOCKING:design] ParseScorerReply rests on a validation that does not exist
**Location:** batch 6 card 30 (and card 61)
**Issue:** the card says to keep "the Python's validation of the score record's required fields", but `score_run.py`'s `_extract_fenced_json` only parses — it validates nothing. The required set is also schema-dependent (`summary_matches` for exploration; `decoy_admitted`/`lookalikes_matched` for impact), and `ParseScorerReply(reply string)` takes no schema, so nothing can decide what is missing. Separately, `score_run` stamps `prompt_template = task.schema` into `score.json`; no card states whether `ScoreRecord` keeps or drops it.
**Fix:** name the required field set (and how the schema reaches the parser) in card 30, and give `prompt_template` an explicit keep-or-drop disposition.

### [BLOCKING:scope] BuildServer loses the CGO_ENABLED=1 build environment
**Location:** batch 8 card 36
**Issue:** `build_server` sets `CGO_ENABLED=1` and `cwd=repo_root`, and names the C-toolchain requirement in its failure message — the toc verbs link tree-sitter C grammars. Card 36's requirements never mention `CGO_ENABLED`, and the prescribed seam `build func(args ...string) (string, error)` carries neither an env nor a cwd, so the plan gives the value no place to live.
**Fix:** name `CGO_ENABLED=1` and the `repoRoot` working directory in card 36's requirements, or widen the seam to carry them.

### [BLOCKING:scope] Card 24's Context omits the file defining GateReport
**Location:** batch 5 card 24
**Issue:** the requirements name `NewIngestRecord(configID string, rep, attempt int, report GateReport) IngestRecord`, but `GateReport` is defined in `bench/loomyard-eval/ladder/internal/ladder/gates.go`, which appears in neither `Context:` nor `Edits:` for this card (card 26 does list it).
**Fix:** add `bench/loomyard-eval/ladder/internal/ladder/gates.go` to card 24's `Context:`.

### [NIT:scope] prepare-session's environment precondition could be checked vacuously
**Location:** batch 11 card 52, batch 10 card 45
**Issue:** `CheckEnvironmentPrecondition(env []string)` is meant to inspect the operator's own shell, but card 52 only says the run path "runs the environment precondition" without naming the argument — passing `ScrubbedEnv()` (which forces both keys empty) makes the check pass unconditionally, the exact dead-check failure this plan repeatedly guards against elsewhere.
**Fix:** state in card 52 that the precondition receives `os.Environ()`, never `ScrubbedEnv()`.

### [NIT:decision] Scoring session's pin set contradicts the discussion
**Location:** batch 11 card 52
**Issue:** `_mill/discussion.md:212` says the scoring `prepare-session` "calls the full `require_pins`", while `:249` says `prepare-session` enforces the narrower set. Card 52 applies the narrower set to every path, including scoring, without recording that it is resolving a conflict.
**Fix:** note in card 52 which discussion statement it supersedes and why (the narrower set already covers `Scorer.Model`/`Scorer.Effort`).

## Verdict

REQUEST_CHANGES
Four cross-batch design gaps and one Context omission need resolving before implementation.
MILL_REVIEW_END
