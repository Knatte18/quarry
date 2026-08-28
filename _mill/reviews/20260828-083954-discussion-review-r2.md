MILL_REVIEW_BEGIN
# Review: Add `impact` verb for caller-context lookup

```yaml
duration_s: 231.0
verdict: REQUEST_CHANGES
reviewer_model: opus
reviewer_self_id: Anthropic Claude, Opus-class; runtime reports claude-opus-5 and I have no way to verify that independently
reviewed_file: /home/knatte/Code/quarry/wts/impact-verb/_mill/discussion.md
date: 2026-08-28
```

## Findings

### [NIT:consistency] Batch `symbol` key collides with the identity object
**Demoted-from:** BLOCKING
**Section:** cli-shape-mirrors-refs + json-key-set
**Issue:** `runBatch` (`internal/cli/cli.go:944-947`) writes `entry := {"symbol": arg, "status": ...}` and then merges the per-entry fields over it, so an `impact` entry carrying the top-level `symbol` identity object would overwrite the batch envelope's query-string `symbol` key — the very key the batch contract identifies entries by. No sibling verb's field set has this collision.
**Fix:** Decide the batch-mode shape explicitly: rename the identity object in batch entries, nest the whole result under one key, or drop `symbol` from batch entries.

### [BLOCKING:design] "Always verified" is not what `Callers` guarantees
**Section:** Scope→Out ("`impact` always runs verified") + callers-come-from-quarry-callers-verified
**Issue:** `callersFromClient` (`internal/quarryengine/query/callers.go:63-123`) silently sets `verify = false` — returning the raw, unfiltered reference set with a nil error — when the declaration-side `textDocument/definition` errors or returns zero locations, or when `SupportsImplementation()` is false or `textDocument/implementation` errors. `SkipVerification=false` therefore does not mean verification ran; for a server without implementation support the caller set is unverified, yet the output still claims `"resolution":"complete"`.
**Fix:** State the disposition for the silent-skip path — accept it and document what `resolution` covers, or surface a per-result verified/degraded signal (which needs an engine-side decision, since `Callers` does not report it today).

### [BLOCKING:design] Empty declaration set has no defined output
**Section:** resolved-symbol-definition-range + json-key-set
**Issue:** The `definition` field is defined as "the first entry of `Callers`' declaration set", but that set is returned empty with a nil error in exactly the swallowed-definition-error cases above; `definition` is not on the `omitempty` list, so the contract has no value for it. The same gap hits the top-level `symbol` object, whose provenance is never stated in any decision — it appears only in the JSON example.
**Fix:** State where the `symbol` fields come from, and what both `symbol` and `definition` become when the declaration set is empty or the declaring file has no toc strategy (omitted, null, or an error field).

### [NIT:scope] `facade_test.go` enforcement claim overstated; work item missing
**Demoted-from:** BLOCKING
**Section:** Technical context → "Guards that will fail if the new package is added carelessly"
**Issue:** The discussion says `quarry/facade_test.go` "enforces that every declaration in `quarry/facade.go` is a type alias… a struct definition or any computation fails this test". It does not: the file is hand-written compile-time checks over an enumerated list (alias pairs at lines 26-137, blank-identifier func assignments at lines 142-157). A new `Impact` and new result types are covered by nothing until those blocks are extended, so the guard silently under-covers — and extending them is not in the Scope "In" inventory.
**Fix:** Correct the claim and add the `facade_test.go` alias-pair/blank-identifier additions (and the stale "fourteen delegating functions" comment) to the in-scope work list.

### [NIT:decision] `seam_enforcement_test.go` constant left as "check whether"
**Section:** Technical context → Guards
**Issue:** The discussion punts the second `minPackageDirs`; the source answers it — that floor is deliberately one below the real count (`seam_enforcement_test.go:100-106`), so no bump is needed either way.
**Fix:** State "no bump" rather than leaving a check for the plan writer.

### [NIT:scope] Stale package enumerations in the two guard files
**Section:** Doc updates (Scope "In") + the "seven-package DAG" gotcha
**Issue:** The doc inventory names README, `doc.go`, `facade.go`, and `cli.go`, but the same enumerations are stale in `seam_enforcement_test.go:2-3,10,100-105` (including a third "seven-package DAG") and `layering_test.go:20,53-57,166-170` — both files are edited by this work anyway.
**Fix:** Add those comment updates to the doc-update list, or state that guard-file comments are deliberately out of scope.

### [NIT:scope] Fixture location unstated; one option trips the layering guard
**Section:** Testing → "file-level tests against fixtures"
**Issue:** `layering_test.go:104-139` walks every `.go` file under `internal/quarryengine/` with no `testdata` exemption, so a Go fixture at `internal/quarryengine/impact/testdata/` fails with "no layering row declared"; existing engine fixtures live at repo-root `testdata/`.
**Fix:** Say the fixtures go under repo-root `testdata/`, as the `testdata/clockfixture` reference implies.

## Verdict

REQUEST_CHANGES
Batch key collision, unverified-caller path, and empty declaration set need decisions.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
