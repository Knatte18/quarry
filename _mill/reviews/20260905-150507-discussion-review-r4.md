MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
duration_s: 293.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5
reviewed_file: /home/knatte/Code/quarry/wts/diff-to-symbols/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Partial parse has no disposition
**Section:** "Entry dispositions, whole-file adds and removes, and per-entry failures"
**Issue:** The closed disposition set is `added|removed|changed|unsupported|error`; the words "lossy" and "partial" appear nowhere in the discussion, yet `treesitter.WithTree` hands back `root.HasError()` and `FileEntry.Lossy` exists precisely for this state — a syntactically broken side yields a truncated symbol table that is reported as ordinary `deleted` entries.
**Fix:** State how a partial parse is represented (a `lossy` flag on the file echo, or a stated decision that it is deliberately invisible), given the working-tree side is Loomyard's card-done path where mid-edit files are normal.

### [BLOCKING:design] `renamed` entry payload unspecified
**Section:** "Two rename tiers, two separate keys"
**Issue:** `rename_candidates` entries are specified (deleted id + candidates carrying created id and signals) and `modified` is specified in full, but `RenamedPair` is only ever called "an exact-tier pair" — and its constituent create and delete are *removed* from `created`/`deleted`, so an id-only pair loses the created symbol's file and span entirely.
**Fix:** Name the fields of an exact-tier pair (and confirm `created`/`deleted` are `[]Symbol`), since removal from the other lists makes this the only place the after-side location survives.

### [NIT:consistency] One-walk leaf bucketing vs nested symbols
**Demoted-from:** BLOCKING
**Section:** "The body token stream, and the exact-tier identity test"
**Issue:** "buckets each leaf into the stream its start byte falls in" is singular, but symbol spans nest: an interface's `method_elem` leaves must be in the interface type's body stream *and* in that method symbol's own signature stream (the same section requires both), and the several symbols of `const a, b = 1, 2` share one span verbatim.
**Fix:** State that a leaf is assigned to every symbol range containing it, not to one, and that overlap between a type's stream and its member symbols' streams is intended.

### [BLOCKING:design] Exit 1 claim about `toc` is false; missing target undecided
**Section:** "CLI shape, and the exit-code contract"
**Issue:** "1 is reachable only through `RepoRelTarget`'s target-escapes-the-root rejection, exactly as it is for `toc`" — `runTOC` also returns `exitNegative` from its own `os.Lstat` ("target not found: "+rel) in `internal/cli/cli.go`, so the premise is wrong and it leaves undecided what `quarry delta <path-that-does-not-exist-now>` does (a stat would also wrongly reject a target deleted between `--from` and the working tree).
**Fix:** Decide and state whether `delta` stats its target at all, and what an unmatched pathspec produces (exit 0 with an empty delta, or exit 1).

### [BLOCKING:design] Bodyless kinds make the exact tier signature-only
**Section:** "Exact-tier rename scope" / "The body token stream"
**Issue:** `goUngroupedConstOrVarSymbols`, `goGroupedConstOrVarSymbols` and `goInterfaceMethodSymbols` pass `body == nil`, so `BodyStart == DeclEnd` and condition 4 is vacuous for every const, var, alias and interface method — `const A = 1` deleted plus `const B = 1` created is *asserted* as a rename and both vanish from `created`/`deleted`, the same false-assertion outcome the discussion calls a defect for two unrelated interfaces.
**Fix:** State whether a bodyless kind may be asserted at the exact tier on signature identity alone, or is confined to the evidence tier.

### [NIT:consistency] Clause-vote cost understated
**Section:** "Git plumbing: exactly which calls, and `--no-renames`"
**Issue:** "they are per changed *directory*, not per changed file" is wrong about its own mechanism: the vote needs one `git show` process spawn plus one tree-sitter parse per `.go` file in every changed directory, on both sides — ~40 spawns for a one-file change in `internal/engine/`.
**Fix:** Restate the cost honestly (per file of every changed directory, both sides) so the plan prices the primary consumer's per-card call.

### [NIT:design] Target scoping silently reclassifies renames
**Section:** "CLI shape" / "Exact-tier rename scope"
**Issue:** Rename pairing is unit-wide "across the batch", but the batch is the pathspec-scoped diff, so a rename or file-move whose other half falls outside the target is reported as a plain `deleted` with no candidate — and Loomyard's scope guard is described as passing a card's declared targets.
**Fix:** State that classification is relative to the scoped batch, not to the unit as a whole.

## Verdict

REQUEST_CHANGES
Five decisions missing or resting on false premises; two smaller inaccuracies recorded.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 4._
MILL_REVIEW_END
