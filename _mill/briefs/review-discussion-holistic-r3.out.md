MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Opus-class Anthropic model; runtime reports claude-opus-5, which I cannot independently verify
reviewed_file: /home/knatte/Code/quarry/wts/gopls-query-precision/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Empty tag set against Go's template is undecided
**Section:** `template-substitution-semantics` / `registry-owns-the-spelling`
**Issue:** The decision covers a template *without* `{{tags}}` ("sent verbatim") and *no* template ("hard error"), but never the case that actually dominates — Go's built-in `{buildFlags: ["-tags={{tags}}"]}` with an empty normalized tag set; the Testing section lists "an empty tag set against a template-bearing entry" as a scenario with no stated expected outcome. The three candidate behaviours (send `{"buildFlags":["-tags="]}`, send `{"buildFlags":[]}`, send no `initializationOptions` at all) are not equivalent: `lspclient.go:402` sends no `initializationOptions` today, so two of the three silently change every untagged Go query's session config and contradict the repeatedly-made "untagged behaviour unchanged / byte for byte" promise in `daemon-keyed-by-tag-set` and `native-path-private-spawn`.
**Fix:** State explicitly what is sent when the normalized tag set is empty and a template containing `{{tags}}` exists, and tie it to the back-compat claim.

### [BLOCKING:design] Native private-spawn cost ignores batch mode
**Section:** `native-path-private-spawn` ("Cost, accepted")
**Issue:** The accepted cost is stated per *invocation* ("a tagged query on the native path pays a cold gopls start"), but `refs`, `definition`, and `symbol` accept N positional arguments and make one independent engine call per argument (`internal/cli/cli.go:189-199`, `:331`, `:432`, each calling `buildOptions` + `quarry.References`/`Definition`/`Symbol` inside `runBatch`). With a non-empty tag set on the native path — the only path on windows — an N-symbol batch therefore pays N full cold gopls starts *within a single invocation*, each indexing the whole module, not one. The stated cost analysis rests on a false premise about invocation granularity, and the alternatives were weighed against that premise.
**Fix:** State the batch-mode amplification and either accept it explicitly or say how a batch invocation shares one tagged connection.

### [BLOCKING:consistency] timedOut does not hard-kill a supervised connection
**Section:** `sequential-verification` (Decision: deadline vs teardown)
**Issue:** The decision asserts a verification-phase deadline sets `timedOut` "so `teardownConnection` takes its hard-kill branch", and the Testing section requires asserting "teardown hard-kills". `teardownConnection` (`internal/quarryengine/query/refs.go:137-157`) returns immediately for `daemon.ConnKindSupervised` with no kill and no close, and supervised is Go's primary strategy (`ensureserver.go:67-70` returns `ConnKindSupervised` whenever `ensureSupervised` succeeds). The hard-kill consequence only holds for `ConnKindNative`/`ConnKindLegacy`, so the stated rationale and the named test assertion are wrong on the dominant path.
**Fix:** Restate the deadline/teardown consequence per `ConnKind`, and say what (if anything) should happen to a possibly-stalled *supervised* daemon after a verification deadline.

### [BLOCKING:consistency] Testing plan assigns assertions to tiers that cannot observe them
**Section:** Testing — "Filter ordering (`internal/cli`)" and "Declaration-side degradation (`query`)"
**Issue:** Two hermetic tests are placed in packages that cannot make the stated assertion. "Filter ordering (`internal/cli`): verification → `--within` → `--except` → declaration exclusion" cannot be tested there: `one-connection-verification-entry-point` moves verification into the engine, and `internal/cli` calls `quarry.References`/`Definition` (soon `quarry.Callers`) as direct package-level calls with no injection seam (`cli.go:608`, `:614`), so only the three CLI-side filters are reachable hermetically. Conversely "Declaration-side degradation (`query`) ... must assert the resulting `assert-no-callers` outcome is a violation report, not exit 0" asks a `query`-package test to assert a CLI exit code it has no access to — and this is flagged as "the single most important test in the task".
**Fix:** Reassign each assertion to a tier that can observe it, and name the seam (e.g. a `callersFromClient` helper mirroring `symbolFromClient`) any hermetic engine-side test needs.

### [NIT:decision] scout-multilang.md's "40%" figure has no disposition
**Section:** Problem / Scope ("In" — Documentation)
**Issue:** The Problem section notes `docs/scout-multilang.md:17` and issue #2 state "40%" while `:115`'s own per-symbol counts give 59/106 (~56%), then leaves the fix as "worth a correction to that doc if anyone touches it"; Scope's documentation bullet names only `README.md` and `docs/servers.yaml.example`, so a plan writer cannot tell whether this doc edit is in or out.
**Fix:** Put the `scout-multilang.md` headline correction explicitly in Scope In or Scope Out.

## Verdict

REQUEST_CHANGES
Four blockers: empty-tag rendering undecided, batch cost premise, teardown claim, misplaced test assertions.
MILL_REVIEW_END
