MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
duration_s: 260.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5)
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] No seam from a Symbol to its decl/body nodes
**Section:** "The body token stream, and the exact-tier identity test" + Layering
**Issue:** The streams are defined over `decl`/`body` *nodes*, but `Strategy.Symbols(unit, root, src) []Symbol` (`internal/engine/strategy.go:29`) returns only `[]Symbol`, `Symbol` carries lines and text but no byte offsets (`answer.go:59-97`), and byte offsets appear nowhere outside `nodes.go:18,35` — so "the same extractor toc uses" yields no way to reach each symbol's declaration node.
**Fix:** Decide the seam explicitly — a new `Strategy` method returning per-symbol streams (or byte spans), versus a second walk in `delta.go` — and add it to Scope's list of new exported engine entry points.

### [BLOCKING:design] `Signature` is not byte-identical to the cut span
**Section:** "What `modified` means" and "The body token stream…"
**Issue:** The claim that `Symbol.Signature` is "byte-for-byte the span the signature token stream is built over" is false for grouped declarations: `golang.go:334` builds `"type " + SignatureCut(spec, body, src)` and `golang.go:504` builds `"const "/"var " + SignatureCut(spec, nil, src)`, and the `decl` node passed to `SignatureCut` is the declaration for ungrouped shapes but the *spec* for grouped ones — so the named test "an interface's signature stream asserted byte-identical to the symbol's own `Signature`" cannot hold in general.
**Fix:** State the `(decl, body)` pair per shape for all five kinds (including const/var's nil body) and restate the invariant to account for the synthesized keyword prefix.

### [NIT:consistency] Clause vote uses two enumeration rules
**Demoted-from:** BLOCKING
**Section:** "Layering" + "Untracked … tracked-but-gitignored"
**Issue:** For a revision side the vote is taken over `git ls-tree` (unfiltered, tracked only); for the working-tree side over `ClauseMapForDir`, described as "ignore-filtered exactly as `dirPackage` already is" and reading untracked files from disk — so a tracked-and-ignored `.go` file votes on one side and not the other, contradicting the rationale that "using one enumeration rule on both sides keeps the sides mutually consistent". A disagreeing `dirPkg` changes the unit and turns every symbol in that directory into a create plus a delete.
**Fix:** Fix one enumeration rule for the clause vote on both sides and state it; note also that `dirPackage` does not filter — `walkDir` hands it pre-filtered entries.

### [BLOCKING:design] Multi-occurrence keys compare bodies only
**Section:** "The symbol table key, and `func init`"
**Issue:** A key with multiplicity > 1 is compared as "multisets of body token stream hashes", yet the entry is said to carry `changed` "naming the dimensions" from the four-word set — a doc-only or signature-only change to one of two `init` functions produces equal body-hash multisets and therefore reports nothing, contradicting the `modified` decision; and which after-side `Symbol` the single `modified` entry carries (and what its `before` block holds) is undefined.
**Fix:** Define the multiset over all compared dimensions, and state which symbol and `before` block a multi-occurrence `modified` entry carries.

### [NIT:consistency] Scope says `.` means the whole repository
**Demoted-from:** BLOCKING
**Section:** Scope ("In", CLI bullet)
**Issue:** Scope states `delta` takes "exactly one target (`.` meaning the whole repository)", while the CLI decision — and the r2 correction in the Q&A log — establish that the target goes through `RepoRelTarget(root, base, target)` with `base = cwd`, so `.` means the current directory. A plan writer reading Scope would implement the superseded rule.
**Fix:** Correct the Scope bullet (and the stale Q&A answer) to match the CLI decision.

### [BLOCKING:design] `--name-status` letters beyond A/M/D unmapped
**Section:** "Git plumbing" + "Entry dispositions…"
**Issue:** Dispositions are derived from a nil/non-nil byte model, but `git diff --name-status` also emits `T` (typechange, e.g. file↔symlink) and `U` (unmerged, reachable on the working-tree path during a conflict — the primary consumer's own path); the discussion never says how those rows map to an entry or a disposition.
**Fix:** State the status-letter → disposition mapping, including the disposition or error for unrecognised letters.

### [NIT:consistency] Helper named twice
**Section:** "The unit is supplied to the core" vs Scope/Layering
**Issue:** The unit-derivation helper is `UnitsForDir(dirRel, clauses)` in the first decision and `UnitsForClauseMap(dirRel, clauses)` in Scope, Layering and Technical context.
**Fix:** Pick one name and use it throughout.

## Verdict

REQUEST_CHANGES
Token-stream seam, signature invariant, clause-vote asymmetry and multi-key comparison all need resolving.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 4._
MILL_REVIEW_END
