MILL_REVIEW_BEGIN
# Review: The glyph package (T1)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5, high reasoning effort)
reviewed_file: /home/knatte/Code/quarry/wts/glyph-package/_mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Reason vocabulary is closed but never enumerated
**Section:** Decisions §"One `*ParseError` with a closed Reason vocabulary" + Testing §`golang_test.go`
**Issue:** The set is declared closed and "one per reject in the spec", yet only eight names appear anywhere (`unsupported_language`, `no_separator`, `unit_empty`, `unit_dot_segment`, `unit_bad_rune`, `member_bad_rune`, `member_too_deep`, `member_keyword`) while the reject table lists ~24 cases, and the generation rule contradicts the examples: whitespace rejects share `unit_bad_rune`/`member_bad_rune`, but the member-alphabet rationale promises `*dualHandler.Handle` and `(*dualHandler).Handle` "their own messages rather than a generic 'bad rune'". Unmapped cases include empty member, leading `/`, trailing `/`, `//`, `Box[T]`, `Renderer.Draw(int)`, and the §7 dotted spelling — and `unit_empty` is not said to cover both "unit half empty" and "empty segment".
**Fix:** List the `Reason` constants explicitly and map every reject row in the Testing section to exactly one of them, since the plan's own completeness test iterates the constants.

### [BLOCKING:consistency] §2/§3 Python and C# examples cannot be split-tested
**Section:** Decisions §"Python and C# examples as split-only tests" vs Testing §`parse_test.go`
**Issue:** The decision says "every Python and C# example in §1 **and §3**" is a structural-split test "asserting the example divides into the unit and member halves", but every §3 Python/C# example is a bare member fragment with no `#` (`Beta.Inner.handle`, `Renderer.Draw(int,string)`, `Draw(ref int)`, `List<>`, `Renderer.this[int]`, `Box.operator +(Box,Box)`, `Renderer.~Renderer()`, `Renderer.ILayout.Draw(int)`), so no split test is possible; §2's non-Go unit examples (`loomyard.engine`, `Loomyard.Engine.Layout`, `global`) have the same shape. The Testing section in fact covers only §1's three full-glyph rows, so the two sections disagree about what the done criterion "every example and corner case in §1–§3 is a test" demands.
**Fix:** State the criterion's scope for non-splittable §2/§3 fragments — either explicitly out of T1's reach with the reason recorded, or covered by some named alternative — and align the decision's wording with the Testing section.

### [NIT:consistency] Struct-equality method vs the no-dependency rule
**Section:** Constraints / Testing §`golang_test.go`
**Issue:** Accept cases assert the whole `Glyph` (two `[]string` fields), but no comparison method is named; the cited `golang-testing` skill prescribes `cmp.Diff` from `github.com/google/go-cmp`, which is not in `go.mod` and would contradict "no dependency outside the standard library" — while the repo's existing tests use `reflect.DeepEqual` (`internal/quarryengine/toc/toc_test.go:700`, `extension_test.go:36`). Note `go list -deps ./glyph` would not catch a test-only dependency.
**Fix:** Say that test dependencies are covered by the stdlib-only constraint and name `reflect.DeepEqual` (or `slices.Equal`) as the repo precedent.

### [NIT:consistency] Two wordings of the `go list -deps` pass condition
**Section:** Technical context §"Allowed imports" vs Testing §"Not tests"
**Issue:** One place says the output "must show nothing else and no cgo" (only `fmt`, `strings`, `unicode`), the other says it is "expected to list only standard-library packages"; the transitive stdlib closure of `fmt` is far larger than three packages, so the stricter wording cannot be the literal pass condition.
**Fix:** Keep one phrasing — "every listed package is standard library, and no non-stdlib module appears" — and drop the three-package reading.

### [NIT:consistency] Cited skills are not at `.claude/`
**Section:** Technical context §"Existing conventions to follow"
**Issue:** The file points at `.claude` skills `golang-comments`, `golang-testing`, `golang-build`; no `.claude` directory exists in this worktree or at the hub root — the skills ship as millhouse plugin skills.
**Fix:** Refer to them as plugin skills by name rather than by a `.claude` path.

## Verdict

REQUEST_CHANGES
Reason vocabulary unenumerated; the §3 split-only test decision is impossible as written.
MILL_REVIEW_END
