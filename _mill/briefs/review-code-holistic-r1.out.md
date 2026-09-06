MILL_REVIEW_BEGIN
# Review: M4 matrix run: execute the descoped kick-start batch (cards 29-32) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-09-06
```

## Findings

### [BLOCKING:consistency] conclusion.md's dirty-invocation causal story contradicts CollectInvocation's ordering
**Location:** `bench/loomyard-eval/ladder/results/2026-09-06-kickstart/conclusion.md:196-201` vs `bench/loomyard-eval/ladder/internal/ladder/pack.go:227-345`
**Issue:** `Pack` calls `CollectInvocation` (which polls `git status --porcelain`) *before* it writes the card, `pack-resolve.json` or `provenance.json` (pack.go lines 270-320). So invocation 1's recorded `quarry_dirty_files` (`07-e1-pack.md`, the results-root dir) cannot be that invocation's "own just-written pack outputs" as the conclusion claims — they must be uncommitted leftovers from invocation 0's completed write, meaning the tree was *not* clean when invocation 1 launched either, a second instance of the `commit-clean-before-each-harness-invocation` Shared Decision being missed, not one.
**Fix:** Correct the coverage section to say invocation 1 ran against invocation 0's uncommitted output (not self-referential same-invocation output), and note the decision was missed twice in a row before card 29's step 4 was actually followed for the `run` call.

### [NIT:consistency] "self-referential" framing undersells two decision misses as harmless by construction
**Location:** `bench/loomyard-eval/ladder/results/2026-09-06-kickstart/conclusion.md:196-201`
**Issue:** Given the above, both pack invocations (0 and 1) ran on a dirty tree, not just invocation 0 as narrated ("the first ran against a dirty tree... the second... is the one whose pack-resolve.json and treatment-card block are what this root carries" — implying the second was clean going in).
**Fix:** State plainly that neither `pack` invocation satisfied the preflight commit-clean step; only the `run` invocation (2) did, which is the one the Shared Decision actually gates measurement validity on.

## Verdict

REQUEST_CHANGES
Statistics, artefact cross-references and roadmap edits check out; fix the conclusion's inaccurate dirty-invocation causal narrative.
MILL_REVIEW_END
