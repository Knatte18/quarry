MILL_REVIEW_BEGIN
# Review: Thin quarry/ facade over internal/quarryengine

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnetmax
reviewer_self_id: Claude Sonnet 5 (claude-sonnet-5)
reviewed_file: /home/knatte/Code/quarry/wts/quarry-thin-facade/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:consistency] File-count claims wrong: 14/20 actual, not 20/21
**Section:** Problem; Scope; "Package layout" Decision table; "Tests move with their files" Decision.
**Issue:** Problem says "20 production files... plus 21 test files." Actual `quarry/` (verified by directory listing) has 14 production `.go` files and 20 test `.go` files — matching the Package-layout table's own 14-file list and the 20-file test-placement enumeration later in the same document. The headline counts contradict both the source tree and the discussion's own detailed appendices.
**Fix:** Correct Problem/Scope to 14 production / 20 test files (or reconcile against whichever list is authoritative); re-check the ~2,900/~4,100 LOC estimates derived from the wrong counts.

### [BLOCKING:design] refs_test.go crosses the query/daemon package boundary
**Section:** Technical context, "Test seams stay inside one package" bullet; "Tests move with their files" Decision.
**Issue:** `refs_test.go`'s `TestReferences_HasNativeDaemonRoutesThroughEnsureServer` calls `withFakeInstaller(t, ...)` and `withTempUserCacheDir(t)`, both unexported functions declared only in `toolchain_test.go` (which reassign `installGoToolchain`/`userCacheDir`, also unexported, both in `toolchain.go`). The plan places `refs_test.go` in `query/` and `toolchain_test.go`/`toolchain.go` in `daemon/` — an unexported cross-file test dependency that cannot compile once split, directly contradicting the claim "no test seam has to cross a package boundary."
**Fix:** Name this seam and decide its disposition (export a test-only fake-installer hook, duplicate the fake in `query/`, or relocate this subtest) before the split is executed.

### [BLOCKING:consistency] Stray "scout:" prefix untouched by rename and its grep check
**Section:** Problem statement; "scoutengine: message prefixes become quarry:" Decision; Testing → "Message-prefix rename."
**Issue:** Problem claims "every error and log message in the engine still carries a `scoutengine: ` prefix," but `errors.go:160` (`(*ErrServerSpawnTimeout).Error()`) reads `"scout: gave up waiting for the supervised daemon..."` — a different, pre-existing prefix, correctly excluded from the verified 59-occurrence `scoutengine: ` count. The Testing acceptance check (`grep -rn 'scoutengine'`) also won't match `"scout: "`, so this sibling error type is silently left un-renamed and inconsistent with the other five error types' new `quarry: ` prefix.
**Fix:** Name this outlier explicitly and decide its disposition (fold into the rename to `quarry: `, or state why it stays `scout: `).

### [NIT:consistency] errors.go identifier-usage counts don't match source
**Section:** "Package layout" Decision → Rationale ("daemon: 5 identifiers; query: 7").
**Issue:** Grepping the four daemon-half files (`ensureserver.go`, `toolchain.go`, `daemonstate.go`, `probe.go`) for constructed or `errors.Is`-referenced `Err*` identifiers finds 2 (`ErrServerNotFound`, `ErrServerSpawnTimeout`), not 5; the query-half files (`refs.go`, `symbol.go`, `definition.go`) yield about 6, not 7. The shared-leaf conclusion itself still holds either way.
**Fix:** Recompute or drop the specific counts; the qualitative claim (both halves depend on `errors.go`) doesn't need them.

## Verdict

REQUEST_CHANGES
Two factual self-contradictions and one unaddressed cross-package test dependency must be resolved first.
MILL_REVIEW_END
