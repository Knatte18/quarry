MILL_REVIEW_BEGIN
# Review: Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Pack-freshness gate keys on quarry VCS state
**Section:** D3 (`run` verification), D4 **Issue:** `run` hard-fails when `provenance.quarry_commit`/`quarry_dirty` no longer match the block's recorded pair, but `MergeProvenance` sets those from the *latest* invocation (`provenance.go:400-418`), so any commit in the quarry repo between `ladder pack` and a `run` — including committing the generated card and the tracked results root — bricks the root, while `quarry_dirty` is structurally true on both sides once the pack writes the card and is therefore vacuous. **Fix:** state which facts the freshness gate actually compares and the operator workflow (commit or not) between pack and run, or drop the commit/dirty half and keep the `pack_sha256` check.

### [BLOCKING:consistency] `ladder pack` flag set omits the claude binary
**Section:** D3 (flags), D4 (`CollectInvocation` inputs) **Issue:** D3 fixes the subcommand's flags at `--config` and `--results` "only", while D4 requires `CollectInvocation` with a `ClaudeBinPath`, which `probeClaudeVersion` (`provenance.go:317-322`) executes and whose failure aborts collection; `run` exposes this as `--claude-bin` (default `claude`, `main.go:58`). **Fix:** decide whether `pack` gains `--claude-bin` or hard-codes the default, and say so in D3.

### [BLOCKING:consistency] Glyph substitution leaves e0/e2 cards stale
**Section:** D7 (substitution rule) vs D6 **Issue:** D6 requires all three cards to carry the identical `Uses:` list (and e2 a `Files:` list derived from it), but `ladder pack` rewrites only e1's sentinel block, and D7's procedure names only `pack_targets`, the task-file note and the fasit cross-check — so a substitution silently leaves e0 and e2 naming a symbol e1 no longer lists, an arm difference in exactly the dimension under test. **Fix:** add the hand-edit of all three cards' `Uses:` lists and e2's `Files:` list to D7's substitution procedure.

### [NIT:consistency] `§8.2 of docs/rewrite-plan.md` does not exist
**Section:** Problem **Issue:** `docs/rewrite-plan.md` has ten top-level sections and no subsections; the push mechanism is §7's "plan-pack generation (resolved spans injected at dispatch, re-resolved, **never cached**)", whose wording is the opposite of D3's generate-once-and-cache. **Fix:** cite §7 and note that the bench deliberately caches because the pin makes one resolve valid for the matrix.

### [NIT:scope] Sweep's grep misses the inline `len(Allowed)` spelling
**Section:** D2 (call-site sweep) **Issue:** the sweep enumerates by `grep IsControl()`, which does not find `gates.go:62`'s `len(cfg.Allowed) == 0` in `CheckGrantedToolUsed`; the branch stays correct under D2 (it means grants-tools) but its doc comment "returns nil for a control cell (an empty allowed list)" goes stale, as does `config.go:239`'s "expected exactly one control (empty allowed list)" message. **Fix:** state that the sweep also covers inline `len(Allowed)` spellings and that both comments/messages are re-worded.

### [NIT:decision] No pre-rep-1 existence check for the e0/e2 cards
**Section:** D11 (validate is struct-level only) **Issue:** only e1's card is verified before rep 1; a typo'd `card:` path on e2 first fails partway through rep 1, after e0 and e1 have already spent API calls. **Fix:** say whether `run` checks every selected config's card file before rep 1, or accept the cost explicitly.

### [NIT:consistency] One pack cell per letter vs one per file
**Section:** D11 (validate rules) vs D3 step 3 **Issue:** validate permits one `pack: true` config *per ladder letter* while `pack_targets` is a single top-level list and `ladder pack` assumes "the ladder file's one pack cell". **Fix:** make the validate rule at most one `pack: true` config per file, or say how `pack` picks among letters.

## Verdict

REQUEST_CHANGES
Three blocking gaps: freshness-gate keys, pack's claude binary, and substitution's stale cards.
MILL_REVIEW_END
