MILL_REVIEW_BEGIN
# Review: Batch-answer contract: per-target coverage + fail-closed Status helpers

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: _mill/discussion.md
date: 2026-09-09
```

## Findings

### [BLOCKING:consistency] Files-that-change table omits internal/cli/cli.go
**Section:** Technical context → "Files that change" table
**Issue:** The table lists six files and excludes `internal/cli/cli.go`, while Scope, three Decisions and two Q&A entries require three edits there — `codeForResolveResult`'s `case "":` lifted to `Rejected()` (verified at cli.go:85–96), deletion of the `len(results) != 1` guards (verified at cli.go:524 and 659), and removal of the now-unused `strconv` import (verified: cli.go:13, 525, 661 are its only occurrences).
**Fix:** Add the `internal/cli/cli.go` row naming those three edits, so the work-inventory table and the Scope section enumerate the same set.

### [NIT:consistency] `quarry.Resolve` named as an existing facade symbol
**Section:** Decisions → "The facade does not re-verify"
**Issue:** The decision names `quarry.Resolve`, `(*quarry.Repo).Resolve` and `quarry.Name`; the tree has only the method (`quarry/repo.go:87`) and the package-level `quarry.Name` (`quarry/name.go:22`) — there is no package-level `quarry.Resolve`.
**Fix:** Drop `quarry.Resolve` from the list; the decision itself (no facade change) is unaffected.

### [NIT:design] `Name` echo-mismatch message under a two-field divergence
**Section:** Decisions → "Enforcement is a panic from a producer-side verifier"
**Issue:** The message shape for `Name` is "names whichever field diverged", which is undefined when both `Unit` and `Target` diverge at the same index — and the Testing section separately asks the test to assert "both echo values".
**Fix:** State the tie-break (e.g. `Unit` checked first and reported alone, or both reported) so the message assertion in the test is deterministic.

## Verdict

REQUEST_CHANGES
One work-inventory table contradicts Scope by omitting the in-scope CLI file.
MILL_REVIEW_END
