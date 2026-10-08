MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewed_file: _mill/discussion.md
date: 2026-10-08
```

## Findings

### [BLOCKING:design] repopath test option creates an import cycle
**Section:** Testing, "repopath" bullet; Path normalisation.
**Issue:** `internal/repopath/target_test.go` is an internal test (`package repopath`) importing `quarry`, so once `quarry/enclose.go` imports `repopath` the "kept on `quarry` aliases" alternative fails `go test ./internal/repopath` with "import cycle not allowed in test"; "same values" is irrelevant to the cycle.
**Fix:** Decide that `target_test.go` (and `target.go`'s doc comment naming `quarry.ErrTargetOutsideRepo`) moves to the `engine` sentinels, and drop the alternative.

### [NIT:design] Lstat errors other than not-exist have no reason
**Section:** Rejection vocabulary, checks 6–7; Engine architecture, working-tree bullet.
**Issue:** The working-tree `os.Lstat` step assigns only not-exist, directory and symlink; an EACCES or ENOTDIR from `Lstat` (a parent directory without search permission, a file used as a directory segment) is unassigned, and the Testing bullet's "unreadable directory" fixture can hit exactly this branch when built with `chmod 000`.
**Fix:** State that any other `Lstat` error maps to `unreadable` (or ENOTDIR to `missing_file`), matching the stated "no per-file whole-call error".

### [NIT:design] Identical-span parent and child both survive
**Section:** Innermost rule by span containment.
**Issue:** A one-line interface `type I interface{ M() }` gives the type and its method identical spans, so by the "not identical" clause neither is the other's child and a location on that line returns both, which the Consequences list does not state.
**Fix:** Add this case to Consequences (and to the working-tree test list) so the plan writer does not treat it as a bug to special-case.

## Verdict

REQUEST_CHANGES
One infeasible repopath test alternative must be resolved; design is otherwise complete and source-consistent.
MILL_REVIEW_END
