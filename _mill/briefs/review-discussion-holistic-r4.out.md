MILL_REVIEW_BEGIN
# Review: The glyph-maker: declaration to glyph (P1, roadmap 2b)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), Anthropic
reviewed_file: /home/knatte/Code/quarry/wts/glyph-maker/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] Pinned counts cannot be "regenerated under -update"
**Section:** Testing § round trip, step 5 (and Q&A "vacuous-pass guard")
**Issue:** The three counts are specified as "constants in the test ... regenerated only under the existing `-update` flag, exactly as this package's other Loomyard goldens are", but `-update` (`internal/engine/loomyard_test.go:19`) only reaches `golden_test.go:113-119`, which writes `testdata/loomyard/<name>`; a Go `const` in a `_test.go` file can never be rewritten by a flag, so the two halves of the sentence cannot both hold.
**Fix:** Pick one mechanism — a counts golden file under `internal/engine/testdata/loomyard/` compared and rewritten by the existing helper, or hand-maintained constants with `-update` explicitly not involved — and say which.

### [BLOCKING:design] `internal/cli/testdata/name/` goldens have no harness
**Section:** Testing § Goldens
**Issue:** `internal/cli` has no `testdata/` directory and exactly one golden helper, `compareAfterGolden` (`after_test.go:189-211`), which hard-codes the `docs/research/output-formats/after/` path and runs under `TestAfterGoldens`, gated by `loomyardRepo(t)` and by a `-update` flag whose own description (`internal/cli/loomyard_test.go:29`) names after/ and `LADDER_LOOMYARD_REPO`; the discussion says the twelve new files "run everywhere with no environment gate" but never says how they are compared or regenerated, whether each row pins an exit code, or whether they carry after/'s `$ quarry ...` invocation header.
**Fix:** State the new table's compare/update mechanism (new helper vs. reused `-update`, and if reused, add that flag's description and doc comment to the docs inventory), plus the per-row exit code and file-body shape.

### [NIT:consistency] Payload-then-code rule contradicts the `internal` path
**Section:** Decisions § Exit codes
**Issue:** The same decision states "that entry takes `fail`'s path — ... exit 3, no payload" and, two bullets later, "The payload is written to stdout before the exit code is computed, matching `runResolve`"; taken literally an `internal` result emits a success payload followed by the error envelope on stdout. The closing clause ("the one place a `NameResult` does not round-trip to its own exit code through `codeForNameResult` alone") also contradicts this decision's own `codeForNameResult` rule, which already returns `exitInternal` for `Reason == internal`.
**Fix:** Order the steps explicitly — check `Reason == internal` before rendering anything — and drop or reword the round-trip clause.

### [NIT:design] `Error` sentence text is unspecified for the maker's four reasons
**Section:** Decisions § The reason vocabulary / § Text view / Testing § Goldens
**Issue:** Only the propagated-`glyph.Reason` case names its `Error` text (`*glyph.ParseError`'s `Error()`); `parse`, `no_declaration`, `several_declarations` and `internal` have no stated sentence, nor does the missing-`--unit` usage error, yet both the text renderer (`<target> error <reason>: <message>`) and two of the six golden cases pin those bytes.
**Fix:** Spell the four sentences (or a one-line shape rule for them) and the missing-`--unit` usage sentence, as the `"--unit is not valid for toc"` shape already is.

### [NIT:scope] `internal/cli/doc.go` disposition covers only lines 11 and 13
**Section:** Testing § Docs and the verb-set inventory
**Issue:** Lines 15-20 ("A target is handed to the facade verbatim ... whenever the verb takes a glyph"; "'toc' is the only verb that still takes a path") and 22-29 (negative answers always "carrying a status word") become incomplete for a verb whose target is neither a path nor a glyph and whose negative payload carries `error`/`reason` with no status; neither range has a stated disposition, and both are inside a file the inventory already touches.
**Fix:** Give those two paragraphs an explicit keep/extend disposition alongside the existing 11,13 entry.

## Verdict

REQUEST_CHANGES
Two golden/update mechanisms are unresolved; the rest of the discussion verifies cleanly against source.
MILL_REVIEW_END
