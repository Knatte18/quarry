# Review: resolve + expand (T4)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] `expand` of a several-`init` glyph has no defined answer
**Demoted-from:** BLOCKING
**Section:** D5, D11, D14
**Issue:** D11 says `ExpandAnswer.Status` is "`found`, `not_found` or `ambiguous`, decided by exactly D5's and D6's rules" — but D5's rules route a bare package-level `init` with several matches to `multipart`, and `expand <unit>#init` is a reachable call (a non-type target is precisely what D14 exists for). D14's `*NotATypeError` fires only on a *unique* `found` non-type match, so several `init` declarations reach neither D14 nor any status D11 admits; D11's "multipart is unreachable for a Go type" is true of types but the target here is not one. The three decisions together leave the case undefined.
**Suggested fix:** State the disposition in D14's precedence list — e.g. a resolution landing on one *or several* declarations of a non-type kind returns `*NotATypeError` naming the kind (`init` → `function`), so `multipart` stays unreachable in `ExpandAnswer` by construction; add the case to Testing 15.

### [BLOCKING:design] I/O-error disposition inside `Resolve` is undecided
**Section:** D2, D7, D8
**Issue:** D2 reserves the call-level `error` for "a failure of the call as a whole (an I/O error the walk could not attribute to one target)", but a `symbolsOfUnit`/`unitDirs` read failure *is* attributable — to exactly that unit's targets — and D8 closes the per-entry `Error` cases to two (parse reject, `ErrTargetOutsideRepo`), neither of which covers it. D7 likewise matches only `ErrTargetNotFound`/`ErrTargetOutsideRepo` from `TOC` and says nothing about any other error. Whether a unit-read failure fails the whole call or becomes per-target error entries changes the public contract (can a `ResolveResult` carry an I/O error?), and the plan writer would have to invent the answer.
**Suggested fix:** Decide it in one sentence — e.g. any engine error other than the matched sentinels fails the call as a whole via D2's `error` return (keeping D8's per-entry `Error` closed to pre-resolution failures) — and add a Testing row for it.

## Verdict

REQUEST_CHANGES
Two undefined dispositions — the several-`init` expand target and engine I/O errors mid-call — must be decided before planning; everything else, including every T3-provenance claim, verified cleanly.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
