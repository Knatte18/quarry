MILL_REVIEW_BEGIN
# Review: resolve + expand (T4)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5), high reasoning effort
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] Provenance and Technical context contradict on batch 5
**Section:** Provenance (¶ "Batch 5 (`spans`) landed…") vs Technical context (¶ "Everything T4 builds on is unmerged") **Issue:** Provenance says `unitDirs`/`symbolsOfUnit`/`SpansOf` "are now written and committed on `engine-core`" and read from code (confirmed: `wts/engine-core/internal/engine/resolve.go` holds all six functions), while Technical context still says they "are **design only** — T3's batch 5 (`spans`) had not started"; the same paragraph calls the verification list "six-item" when it has eight, and D2 plus the Q&A call `resolve.go` "merged"/"now-written" although Provenance states nothing is on `main`. **Fix:** delete the superseded Technical-context paragraph, fix the item count, and use one word ("committed on `engine-core`, unmerged") everywhere.

### [BLOCKING:scope] The first of Scope's "two permitted exceptions" is never defined
**Section:** Scope → Out ("The one permitted exceptions are two: the mechanical follow-through T3 itself scheduled (see Technical context), …") **Issue:** Technical context describes no T3-scheduled follow-through, and the only engine change listed anywhere is the `golang.go` comment — which D12 and the Q&A both call "Scope's mechanical-follow-through exception", collapsing the two into one; the carve-out from "no change to `internal/engine`" is therefore unbounded. **Fix:** either name the second exception's concrete change (file and card) or state that the stale comment is the single exception.

### [BLOCKING:scope] D17's fixture inventory does not cover the Testing section
**Section:** D17 vs Testing 3/12/15a/16 **Issue:** verified against the committed tree: `testdata/tree/pkg` holds only `Alpha()` and `Beta()` and no method at all, so D17's `found` fixture ("a plain package-level function and a method, in the existing `testdata/tree/pkg`") does not exist; no committed fixture has a type with methods in **several** files (Testing 12's whole point); nothing covers Testing 16's non-lexicographic `os.ReadDir` order; Testing 15a needs a third run-time tree though D17 admits exactly two; and the ignore-filter tree D17's rationale claims as an exception is enumerated nowhere and is already covered by T3's `TestSpansOf_IgnoreFilter`. **Fix:** enumerate every fixture the Testing section consumes, and say where each lives given D17's own rule against perturbing T3's enumerated `testdata/` trees.

### [NIT:consistency] D9's memo API names an undefined type and an impossible error
**Section:** D9 **Issue:** `dirs map[string]unitDirsResult` and `dirsOf(unit) (unitDirsResult, error)` reference a `unitDirsResult` type defined nowhere in the discussion, and give `dirsOf` an `error` return that `func (r *Repo) unitDirs(unit string) (dirs []string, collision bool)` — no error return, verified in `resolve.go` — cannot produce. **Fix:** define the struct inline and drop the error from `dirsOf`.

### [NIT:decision] The benchmark's Loomyard gate has no stated disposition
**Section:** D16 **Issue:** both artefacts are said to be gated "by T3's D17 rule exactly", but T3's helper is `func loomyardRepo(t *testing.T) string` and cannot be called from `BenchmarkResolveTwentyGlyphs`; widening it to `testing.TB` would touch `loomyard_test.go`, which "Files T4 touches" does not list. **Fix:** say whether T4 duplicates the gate in `loomyard_timing_test.go` or widens T3's helper.

### [NIT:consistency] Misattributed measurement in D16
**Section:** D16 ("§5 measures `internal/reedengine` at 67 files and 65 ms for a single glyph") **Issue:** plan §5's 65 ms row is "one glyph, 35-file package (317 KB)"; the 67-file figure is §4's `toc internal/reedengine` example, a different package and a different measurement. **Fix:** cite the 35-file row, or drop the package name.

## Verdict

REQUEST_CHANGES
Two self-contradictions and an incomplete fixture inventory must be settled before plan writing.
MILL_REVIEW_END
