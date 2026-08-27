MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping)

```yaml
duration_s: 278.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: _mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Verification drops interface-dispatch callers
**Section:** `verification-over-scope-heuristics` (the "Consequence worth stating explicitly")
**Issue:** Only the interface-method→concrete direction is dispositioned; the reverse is unaddressed and is a false negative. `docs/scout-multilang.md:140` records that gopls unifies an interface method's references with every implementer's, so `assert-no-callers <ConcreteType.Method>` returns interface-dispatch sites; `textDocument/definition` at such a site resolves to the *interface's* declaration, not the concrete method's, so verification drops it — the gate goes green on a delete that breaks interface satisfaction and the build.
**Fix:** State the disposition explicitly (accept + document + pin with a test, or widen the match set to declarations the queried symbol implements/is implemented by) and add the mirrored scenario to the verification-filter test list, which today only pins the interface-method direction.

### [NIT:decision] Two cited evidence docs have no disposition
**Demoted-from:** BLOCKING
**Section:** Scope → In → Documentation (four items); Technical context → Prior evidence
**Issue:** `docs/scout-agent-usage-findings.md:46-47,:59` states the default `assert-no-callers` behaviour "still returns the same class of false positive" and leaves "should `--within` become the default" as an unresolved open follow-up — both falsified/answered by this task — and `docs/scout-vs-grep.md:109-110` carries the same opt-in framing; the discussion cites both under Prior evidence and never says update, annotate, or deliberately leave. The closed four-item list reads as exhaustive, and the rule applied is inconsistent: `docs/scout-multilang.md` is an equally dated record and *is* corrected because "leaving a known-wrong figure in the document this task cites as its evidence would undercut the evidence".
**Fix:** Give both files an explicit disposition in the documentation inventory — either a fifth/sixth item, or a one-line statement that dated findings records are frozen and only `scout-multilang.md:17` is exempted, with the reason.

### [NIT:consistency] filter-ordering names four stages; code has three
**Section:** `filter-ordering`; Testing → "Filter ordering"
**Issue:** The order is given as verification → `--within` → `--except` → declaration exclusion, but `filterUnexpectedCallers` (`internal/cli/cli.go:671-688`) performs `--except` and the declaration exclusion in one pass, and Technical context pins it "stays as it is". A plan writer could read the four-stage phrasing as mandating a split refactor.
**Fix:** Note that the last two are one existing function and order-independent (both are removals), so no restructuring is implied.

### [NIT:design] Tag normalization sited in `internal/cli` only
**Section:** `tag-set-normalization`; Testing → hermetic tier
**Issue:** Normalization is assigned to `internal/cli`, but `Options.BuildTags` is public via `quarry.Options`; a non-CLI caller passing `[""]` or `["b","a"]` bypasses both the empty-set no-op ("byte-identical to today's") and the set-not-spelling state-dir keying, and the engine's hard-error predicate is specified over "normalized" tags it never computes.
**Fix:** State whether the engine re-normalizes defensively or whether normalization is explicitly the caller's obligation (as `StateDir` already is, `refs.go:76-79`).

### [NIT:consistency] Two line citations do not support their claims
**Section:** `template-substitution-semantics`; Scope → Documentation
**Issue:** `registry.go:53-60` is the `Entry` struct and says nothing about overlay whole-replacement (that is `registry.go:14-16` and `load.go:55-58`); the `assert-no-callers` interface-method paragraph starts at `cli.go:546`, not `:544`. Both underlying claims are true as verified.
**Fix:** Correct the two ranges so the citations remain checkable.

## Verdict

REQUEST_CHANGES
Verification's concrete-method false negative and two undispositioned evidence docs must be resolved first.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
MILL_REVIEW_END
