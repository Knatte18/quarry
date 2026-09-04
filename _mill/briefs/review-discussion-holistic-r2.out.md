MILL_REVIEW_BEGIN
# Review: resolve + expand (T4)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus (Anthropic); the harness names the exact model claude-opus-5, which I cannot independently verify
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:design] Expand has no rule for a unitDirs collision
**Section:** D14 (table) vs D6/D11 **Issue:** D14 calls its five-row table "the full disposition of a match set" and keys every row on match count/kind only, while D6 makes a *single* match under `collision == true` `ambiguous` for `Resolve`; `symbolsOfUnit` returns the union either way, so `Expand` of a `_test`-unit type under a collision would answer `found` where `Resolve` answers `ambiguous` — contradicting D11's stated rationale that "the two verbs can never disagree about what a glyph resolves to". **Fix:** add a collision row to D14's table (or state that D6 applies to both verbs and that `Expand` reads `collision` too), and add the case to Testing 15.

### [BLOCKING:design] The grouping test cannot observe the guarantee it names
**Section:** D9 / Testing 8 **Issue:** D9 fixes the memo as two plain local maps inside `Resolve`, but Testing 8 requires "a counter the test can read directly" on "an explicit type the test constructs and drives"; a plain `map[string][]Symbol` carries no call counter, and asserting one entry per unit is true by construction, so nothing described proves `symbolsOfUnit` ran once per unit inside a real `Resolve` call — the §12 done-criterion and the 150 ms number both rest on it. **Fix:** decide the seam — a named memo type with a parse counter that an unexported `resolve(targets, memo)` takes, or an injected parse func — and state that `Resolve` delegates to it.

### [BLOCKING:design] Error boundary undefined on the glyph branch
**Section:** D2 ("stated exhaustively") / D7 / D10 **Issue:** D2 carves out `ErrTargetNotFound` and `ErrTargetOutsideRepo` by error identity, but only D7 (path targets) says what to do with them; nothing says what a *glyph* target does if `unitDirs`/`symbolsOfUnit` surface one, and nothing states that a unit with no directory must be `(nil, nil)` rather than an error — with `symbolsOfUnit` still unwritten (verified: `wts/engine-core/internal/engine/resolve.go` holds only `unitDirs`/`dirExists`), a missing-directory error would make D2 fail the entire call for exactly the §8.1 Create case that needs `not_found` + `unit: not_found`. **Fix:** state that a unit with no directory is not an engine error and that a sentinel reaching the glyph branch is a call failure (or an answer), and add it as verification item 7.

### [NIT:consistency] The `id` canonicalisation rationale is false for Go
**Section:** D3 (`ID`) **Issue:** `glyph/golang.go` normalises nothing — `Draw (int)` is rejected outright (`ReasonMemberParens`, and spaces by `ReasonUnitBadRune`/`ReasonMemberBadRune`), so for Go `ID` is byte-identical to `Target` on every non-error result; the cited §8.1 `Draw (int)` example is a C# case. **Fix:** keep `ID` but restate the reason (wire-form parity with `Symbol.ID`, not canonicalisation), so Testing 3 does not chase a normalisation case that cannot exist.

### [NIT:consistency] "No directory literally named `<x>_test`" is untrue
**Section:** D6 (note) / D17 (fixtures) **Issue:** T3's tree already carries `internal/engine/testdata/foo_test/literal.go`, and nothing stops a committed `testdata/foo/` sibling, so the collision is not among "the two shapes a committed tree provably cannot express" — the stated ground for building it under `.scratch/` does not hold. **Fix:** either give the real reason (a committed sibling would perturb T3's existing `testdata` walk assertions) or reconsider a committed fixture.

### [NIT:decision] Stale T3 comment describing `expand` gets no disposition
**Section:** D12 / D18 **Issue:** T3's `internal/engine/golang.go:272-274` states the expand verb "renders a type's head by omitting the lines its own member symbols cover", which D12 explicitly contradicts (one contiguous head entry; the subtraction is the consumer's, as `answer.go:69-71` says); Scope forbids touching T3's files and D18 fixes the record at two gaps, so the inconsistency has no owner. **Fix:** say whether correcting that one comment is inside the "mechanical follow-through" exception or is left as-is deliberately.

### [NIT:decision] `Expand` on a target with no `#` — two options, no choice
**Section:** D11 **Issue:** "returns the wrapped `*glyph.ParseError` (or a plain error naming the missing `#`)" leaves the error contract unresolved, and the second option restates a rejection `glyph.Parse` already owns as `ReasonNoSeparator`, against the constraint that every alphabet question is one `glyph.Parse` call. **Fix:** pick the `*glyph.ParseError` from `glyph.Parse` and drop the alternative.

## Verdict

REQUEST_CHANGES
Three gaps: Expand under collision, an unobservable grouping test, and an undefined glyph-branch error boundary.
MILL_REVIEW_END
