MILL_REVIEW_BEGIN
# Review: Facade + CLI, toc (T5a) — holistic

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Anthropic Claude, Opus-class; harness reports claude-opus-5
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [NIT:consistency] File comment may become a second package doc
**Location:** Overview `comment-discipline-matches-answer-go`; cards 2, 3, 5, 6, 9-12, 16, 17 **Issue:** The decision requires every new file to "open with a file-level comment" but never says it must be separated from the `package` clause by a blank line, which is how `answer.go`/`repo.go`/`toc.go` avoid a second package doc comment beside `doc.go`'s. **Fix:** State the blank-line separation in the decision, as the engine files already practise it.

### [NIT:scope] CGO_ENABLED=0 glyph build check has no home
**Location:** Overview `cgo-stays-required-and-unguarded`; all batches **Issue:** `discussion.md`'s Testing "Build checks" names a documented `CGO_ENABLED=0 go build ./glyph/...` staying green, but no card and no `verify:` records or runs it; the decision covers only the "add no build tag" half. **Fix:** Name the check's disposition (run once at the last batch, or record it as already covered by T1's existing assertion).

### [NIT:design] Step 4's non-usageError branch unstated
**Location:** Batch 4, card 16 **Issue:** Steps 1, 5, 6 and 9 each state both the usage and the internal disposition, but step 4 states only "On a `usageError` return `fail(..., exitUsage, ..., true)`", leaving a non-`usageError` from `resolveRoot` undecided (closed only implicitly by card 11 returning that type exclusively). **Fix:** Say explicitly that any other error from `resolveRoot` is `exitInternal`, or that card 11's contract makes the branch unreachable.

## Verdict

APPROVE
Decisions, sequencing, context and source-grounded claims all check out; three minor gaps.
MILL_REVIEW_END
