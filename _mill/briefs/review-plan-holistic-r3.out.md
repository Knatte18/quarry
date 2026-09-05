MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] `_test` self form: doc says none, code pins one
**Location:** batch 5 card 33 vs batch 1 cards 6/9, batch 2 card 16, batch 4 card 31
**Issue:** Card 33 makes `docs/glyph.md` §2 state the external test unit "has no self form of its own", while card 6 tables `internal/engine/testdata/tree/pkg_test#` as a *self-form accept* with `IsSelf` true, card 9 requires `Self(Go, ".../pkg_test")` to succeed and round-trip, card 16 pins it to `StatusNotFound` + `Unit: found` (verified reachable: `unitDirs` in `internal/engine/resolve.go` strips `_test` and `dirExists` finds `.../pkg`), and card 31 pins a CLI row for it — so the grammar accepts a form the contract document is told to deny.
**Fix:** Pick one disposition and state it on card 33 — e.g. the `_test` self glyph is well-formed and always resolves `not_found`, never "has no self form" — since card 36 turns §2's claims into `glyph/docs_test.go` rows and would otherwise table a reject that `Parse` accepts.

### [NIT:consistency] Falsified-comment count disagrees with the overview
**Location:** `_mill/plan/00-overview.md` "doc comments are part of every edit" vs batch 4 card 28
**Issue:** The Shared Decision says the discussion enumerates "seven paragraphs in `internal/cli` (D7)"; card 28 opens "Ten paragraphs across these three files ... and all ten change" and then enumerates ten.
**Fix:** Reconcile the two counts in one direction so the acceptance condition is a single number.

### [NIT:consistency] Batch 2 claims parallelism the DAG forbids
**Location:** `_mill/plan/02-engine-resolve-contract.md` Batch Scope vs `00-overview.md` `batches:` and batch 3 Batch Scope
**Issue:** Batch 2 states batches 2 and 3 "touch disjoint files ... and may run in parallel", but the Batch Index declares batch 3 `depends-on: [2]`, and batch 3's own scope justifies that edge as deliberate sequencing so each batch's `verify:` is a statement about that batch alone.
**Fix:** Drop the "may run in parallel" clause from batch 2's scope, or restate it as "no data dependency, sequenced anyway".

### [NIT:scope] `parses == 0` does not prove "no directory was read"
**Location:** batch 3 card 22
**Issue:** The card asserts the self gate by "reading its `parses` counter as zero" as proof "no directory was read", but `parses` is incremented only in `unitMemo.symbolsOf` (`internal/engine/resolve.go`) before `symbolsOfUnit`; the directory reads live in `dirsOf` → `unitDirs` → `dirExists`'s `os.Lstat`, which is uncounted. The assertion still passes and still proves the gate precedes `symbolsOf`, but the stated claim is wider than the seam measures.
**Fix:** Reword the card's rationale to "no unit was parsed" (or add an explicit dirs-memo emptiness assertion) so the test's doc comment does not overclaim.

## Verdict

REQUEST_CHANGES
One contract/grammar contradiction over the `_test` self form; three consistency nits.
MILL_REVIEW_END
