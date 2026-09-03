MILL_REVIEW_BEGIN
# Review: Ladder harness around headless claude -p (T2)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (per runtime environment; no independent means to verify)
reviewed_file: /home/knatte/Code/quarry/wts/ladder-harness/_mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:consistency] provenance.json lacks the field report reads
**Section:** provenance vs command-surface
**Issue:** command-surface says `incomplete[]` is derived from "`provenance.json`'s record of the run's selected cells", but the provenance decision's enumerated payload has no such field (and no expected-`reps` field), so `report` cannot tell a cell that ran 3 of 5 reps from one that ran 3 of 3.
**Fix:** add the selected-cell list and the effective `reps` to provenance's field list, or state how `report` reconstructs the expected rep set from `raw/` alone.

### [BLOCKING:design] Memory-path blinding check is undefined on resume
**Section:** no-tmp-paths ("Auto-memory is a real blinding surface")
**Issue:** the check reads `memory_paths` from "the first rep of a root" and aborts before further reps; on a resumed run rep 1 is already `complete` and is skipped, and the fingerprint persists only *whether* `memory_paths` was non-empty — not the paths — so the resumed invocation has nothing to scan and the sole defence against operator-memory unblinding silently lapses.
**Fix:** state where the resolved paths are persisted and when the scan runs on resume, and give the triggering first rep a disposition (valid observation, or discarded like a gate-2 failure).

### [BLOCKING:consistency] ladder-toc.yaml ships server args T6 has not chosen
**Section:** ladder-toc-migration vs server-block
**Issue:** the migration says the file ships at T2 merge with `server:` "fully written ... and the args T6 will accept", while server-block's rationale states "T6 has not decided its flags, so they become yaml data" — T2 cannot write an argument list that does not yet exist.
**Fix:** decide whether T2 ships V1's `--target-dir {target_dir}` as a placeholder T7 corrects, or omits `args:` until T6 lands.

### [BLOCKING:design] Token matching semantics unspecified for check (d) and redaction
**Section:** gates check (d) / scorer
**Issue:** after ladder-toc-migration `quarry_tools` is the 3-character token `toc`; check (d) is fatal pre-dispatch on a control prompt containing "any name from `quarry_tools` ... bare", and the scorer redacts the same names from free model prose — under a substring reading, ordinary words containing `toc` kill every control rep or corrupt the text recall/precision is judged on, while check (c) uses the different word "bare `quarry`" for the same idea.
**Fix:** state one matching rule (word-boundary vs substring, case sensitivity) applied by check (c), check (d) and the redactor alike.

### [NIT:consistency] T7 cell set contradicts plan §12
**Section:** ladder-toc-migration
**Issue:** "T7 drops `--cells` and runs the whole matrix" (4 cells across two tasks after migration), but plan §12's T7 row is `run ladder-toc.yaml (a0-none, a2-toc-dir) ... reps 5`.
**Fix:** say T7 selects the two named cells, or note the widening as a deliberate change to §12.

### [NIT:consistency] input_tokens_total absent from the metrics list
**Section:** metrics vs cache-contamination
**Issue:** `input_tokens_total` is introduced only in cache-contamination and is missing from the metrics decision's enumerated per-rep fields; likewise the raw-ignore pattern is written `results/**/raw/` in the decision and `results/*/raw/` in Scope and the Q&A.
**Fix:** list `input_tokens_total` under metrics and use one spelling of the ignore pattern.

## Verdict

REQUEST_CHANGES
Four gaps: provenance field, resume-time memory scan, undecided server args, token matching rule.
MILL_REVIEW_END
