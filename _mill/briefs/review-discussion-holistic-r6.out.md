MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/toc-verbs/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:consistency] Superseded grammar-set/build-tag material survives
**Section:** Scope ("In", line 138-139) and Technical context (lines 866-869)
**Issue:** Scope still lists the deliverable `README "Building and running" update for the grammar-subset build tags (see the grammar-set Decision)`, and Technical context still cites `GOTREESITTER_GRAMMAR_SET` plus "see the 'Grammar set' Decision for the table and the choice" — but no `### Grammar set` Decision exists (only 26 `###` sections, none named that), the chosen backend Decision states "no external grammar build step and no build tags", Testing (1003-1006) says "there is no build-tag surface", and Q&A 1072 marks the grammar-set question superseded; `GOTREESITTER_GRAMMAR_SET` is the rejected pure-Go library's env var.
**Fix:** Delete the grammar-subset scope line and the dangling "Grammar set" Decision references, or replace them with a one-line "superseded by the cgo reversal" note carrying no deliverable.

### [BLOCKING:design] `--lang typescript|rust` on `toc dir` has two answers
**Section:** "Language detection: by file extension" (429-431) vs "Unparseable input" (703-715)
**Issue:** The `--lang` rule says a recognised-but-unimplemented value "gets the same 'not yet supported by toc' error the extension path gives" (for `toc file` that is `output.Err`, exit 1), while for `toc dir` the extension path does not error at all — `.ts`/`.rs` files are *listed* with a per-file `error` key, status `found`, exit 0. `quarry toc dir --lang rust <dir>` therefore has two defensible, observably different outcomes (Err/exit 1 vs `ok:true` + error entries/exit 0), each requiring a different test.
**Fix:** State explicitly which of the two `toc dir --lang <unimplemented>` produces, and add it to the batch outcome→status table.

### [NIT:scope] Root command Short escapes the stale-prose invariant
**Section:** Scope, exact-claim invariant clauses (a)-(e)
**Issue:** `internal/cli/cli.go:51` sets the root cobra `Short` to "code intelligence lookups (references, definitions, symbol search) across supported languages" — a capability enumeration with no count word, so neither grep (a)/(b) nor invariant clauses (a)-(e) reach it, leaving `quarry --help` describing a tool without toc.
**Fix:** Add the root command's `Short` to the inventory, or widen clause (b) from "a count of quarry's verbs" to "any count or enumeration of quarry's verbs or capabilities".

### [NIT:design] No fallback if the mingw cross-compile cannot be run
**Section:** "Parsing backend" — Windows builds ("Unverified", 220-225)
**Issue:** The recipe is explicitly unrun and the implementation batch "must actually run it and adjust if needed", but no disposition is given for the case where no mingw-w64 toolchain is available to the batch — ship the recipe unverified, or block, is left to the implementer, against a constraint that says windows "must not regress".
**Fix:** State the fallback (e.g. document the recipe marked unverified and file a follow-up) so the batch is not blocked by a missing toolchain.

## Verdict

REQUEST_CHANGES
Stale grammar-set deliverable and an undecided `toc dir --lang` unsupported-language outcome must be resolved.
MILL_REVIEW_END
