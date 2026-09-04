MILL_REVIEW_BEGIN
# Review: resolve + expand (T4)

```yaml
duration_s: 311.6
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), high reasoning effort
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Testing 16 asserts Expand Members under a collision
**Demoted-from:** BLOCKING
**Section:** Testing 16, and D17's "ordering, collision union" row **Issue:** D14 row 2 and D11 make `Expand` answer `ambiguous` with `Candidates` and no `Head`/`Members` for every collision with at least one match, so the `.scratch/` collision tree can never produce the `Expand` `Members` that Testing 16 (and Testing 15c, which asserts the opposite on the same tree) requires. **Fix:** assert `Expand`'s `Candidates` beside `Resolve`'s over the collision tree, and leave member-sort coverage to the separate `testdata/methods/` case Testing 16 already names.

### [NIT:scope] D17's fixture inventory omits Testing 13, 14 and 15's kind cases
**Demoted-from:** BLOCKING
**Section:** D17 **Issue:** D17 states it enumerates "every fixture the Testing section consumes", but has no row for Testing 13's interface, Testing 14's member-less type, or Testing 15's `func`/`const` non-type glyphs; the obvious sources (`testdata/glyphs/iface.go`'s `Iface`, `decls.go`'s `Weekday`/`UngroupedConst`, verified present on `engine-core`) sit in the one directory D17's rejected alternatives warn against reusing, so a plan writer cannot tell whether they are sanctioned. **Fix:** add rows for those three cases naming the fixture and its directory, and state why reusing `testdata/glyphs/` for read-only assertions adds no package to a tree T3 asserts on.

### [NIT:design] D5/D6 precedence unstated for `init` under a collision
**Section:** D5, D6, Testing 1 **Issue:** D6's "whatever `n` is" implies the collision check precedes D5's `multipart` row, but unlike D14 the order is never stated for `Resolve`, and Testing 1's table has no "several `init` under a collision" row to pin it. **Fix:** state the check order in D5/D6 as D14 does, and add the row to Testing 1's table.

### [NIT:consistency] Two verifiable counts are wrong
**Section:** Scope exception 2 / D16; D11 **Issue:** `loomyard_test.go`'s `loomyardRepo` body makes five `t.*` calls (`Helper`, `Skip`, two `Skipf`, `Fatalf`), not "seven"; `glyph/errors.go` declares 16 reasons, so excluding `no_separator` leaves fifteen, not "the other fourteen". **Fix:** drop or correct the numbers — both substantive claims (every call is a `testing.TB` method, so the widening leaves the body untouched; one grammar owns every rejection) verify as written.

## Verdict

APPROVE
One testing assertion contradicts D14, and D17's self-declared complete fixture inventory is not.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
