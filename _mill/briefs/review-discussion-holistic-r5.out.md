MILL_REVIEW_BEGIN
# Review: resolve + expand (T4)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude (Anthropic); this harness reports the model id as claude-opus-5
reviewed_file: /home/knatte/Code/quarry/wts/resolve-expand/_mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] Scope puts the collision fixture in testdata/
**Section:** Scope "In", last-but-one bullet, vs D17 / D6
**Issue:** Scope says "Fixtures under `internal/engine/testdata/` ... including a build-tag `ambiguous` pair **and the `_test`-directory collision**", while D17 and D6 place the collision fixture under `.scratch/` and reject committing a `testdata/foo/` sibling; a plan writer following Scope would commit it, and `internal/engine/resolve_test.go:148-151` (`TestSpansOf_LiteralFirst`) asserts `collision == false` for `internal/engine/testdata/foo_test`, so that card would break a T3 test the Testing gate requires to keep passing untouched.
**Fix:** Reword the Scope bullet so the collision fixture is named as the `.scratch/` exception, matching D17's table.

### [BLOCKING:consistency] Two contract gaps or three
**Section:** Scope "Out" (`docs/glyph.md` bullet), D18, final Q&A entry
**Issue:** Scope says "T4 records **two** contract gaps ... see D18" and the closing Q&A says "**two** gaps are recorded in code comments and **neither** is closed", but D18 is titled "Three contract gaps" and enumerates three (the symlink gap added later); the deliverable is one comment card per gap, so the count is not cosmetic.
**Fix:** Update both statements to three, or state explicitly which gap is not a comment deliverable.

### [NIT:consistency] "Signature only, no body change" is not achievable as written
**Section:** Scope exception 2 / D16
**Issue:** The stated widening is `loomyardRepo(t *testing.T)` → `loomyardRepo(tb testing.TB)`, which renames the parameter and therefore rewrites all seven `t.Helper/Skip/Skipf/Fatalf` references in the body of `internal/engine/loomyard_test.go:50-72`; the discussion twice calls it "no body change".
**Fix:** Either keep the parameter named `t` (then it genuinely is signature-only) or drop the "no body change" claim and say "behaviour-preserving".

### [NIT:consistency] "D17's skip/fail rule" names this document's D17
**Section:** Testing, Loomyard heading; Q&A "How is the 150 ms criterion asserted"
**Issue:** Both cite "D17's skip/fail rule", but T4's own D17 is the fixture inventory and carries no gate rule; D16 correctly says "T3's D17".
**Fix:** Write "T3's D17" in both places.

### [NIT:design] D17's stated reason for the `.scratch/` collision tree is not the binding one
**Section:** D6 (last paragraph) and D17 ("why there" column)
**Issue:** The reason given is that a committed `testdata/foo/` would "perturb the existing walk and round-trip assertions that enumerate that tree", but `roundtrip_test.go`'s `tupleSetDiff` is an explicitly duplicate-tolerant multiset comparison and no T3 test enumerates `testdata/`'s children; the assertion that actually breaks is `TestSpansOf_LiteralFirst`'s `collision == false`. The conclusion is right, the premise is not.
**Fix:** Cite `TestSpansOf_LiteralFirst`'s collision assertion as the reason, so the plan writer knows which test constrains the placement.

## Verdict

REQUEST_CHANGES
Two self-contradictions on fixture placement and gap count; every other claim verified against source.
MILL_REVIEW_END
