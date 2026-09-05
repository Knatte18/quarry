MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), per the system prompt's own model identification
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:scope] Card 17 misses text_test.go's own `Dir:` literals
**Location:** batch 2 / card 17
**Issue:** `quarry/text_test.go:350` and `:355` build `ResolveResult{..., Dir: &DirAnswer{...}}` in `TestRenderResolveText`'s table (rows `PathFound`, `PathFoundOneFileEntry`); card 17's `Requirements:` name the `Dir` references in `repo_test.go` and `render_test.go` only, and its closing sentence "Both files break the `quarry` test binary's compile" counts two files, not three — so `quarry` will not compile at the end of batch 2, breaking the overview's green-compile Shared Decision.
**Fix:** Name those two rows in card 17 and say they are renamed to `Listing:` and otherwise retargeted or kept.

### [BLOCKING:consistency] Card 15's branch numbering contradicts its own insertion order
**Location:** batch 2 / card 15
**Issue:** The card inserts the `r.Listing != nil` arm "between the existing `r.Status == \"\"` branch and the existing `r.ID != \"\"` branch" — position 2 — then instructs "branch 3 describes the listing branch and branch 4 the reduced default"; `RenderResolveText`'s doc comment in `quarry/text.go` numbers exactly the switch arms in code order, so following the second sentence produces a comment that contradicts the code the first sentence mandates (and describes the ordering bug the card exists to prevent).
**Fix:** State the renumbering as listing = branch 2, glyph = branch 3, default = branch 4.

### [BLOCKING:scope] `ResolveResult`'s own doc comments left falsified after card 12
**Location:** batch 2 / cards 10 and 12
**Issue:** Card 10 is the only card editing `internal/engine/answer.go`, and it names only the renamed field's comment plus the file header. Four statements on `ResolveResult` are false once card 12 deletes the path branch: the type doc "the answer to one target passed to Resolve: a glyph or a repository-relative path"; `Unit`'s "It is never set on a path result, because a path belongs to no unit"; `Error`'s "or a path target rejected as outside the repository"; `ID`'s "Present only for a glyph target".
**Fix:** Add these four sentences to card 10's (or card 12's) `Requirements:`, with `internal/engine/answer.go` in that card's `Edits:`.

### [BLOCKING:scope] Card 19 leaves `expand.go`'s header and worker comment stale
**Location:** batch 3 / card 19
**Issue:** `internal/engine/expand.go`'s file header reads "and NotATypeError, the one typed failure the verb returns", which `SelfGlyphError` falsifies, and the unexported `expand`'s doc comment narrates "parses target with glyph.Parse ... sets id ... reads m.dirsOf(g.Unit)" with no self gate between them; card 19 requires only `SelfGlyphError`'s own type doc comment, and "do not change ... any other line of the worker" reads as forbidding the worker-comment edit.
**Fix:** Name both the file header sentence and the `expand` doc comment's gate placement in card 19's `Requirements:`.

### [BLOCKING:scope] `runTOC`'s new exit-2 disposition is documented nowhere
**Location:** batch 4 / cards 26 and 28
**Issue:** Card 26 adds an `exitUsage` branch to `runTOC`, but `Run`'s doc comment in `internal/cli/cli.go` enumerates runTOC's pipeline ("1. Convert the target ... Escaping the root is exit 1") and the `exitUsage` constant enumerates its three causes ("an unparseable flag, a missing or extra target, or a --root that does not resolve to a directory"); neither is in card 28's exhaustively-claimed "eight paragraphs", and card 26 orders no comment edit.
**Fix:** Add runTOC step 1 and the `exitUsage` constant comment to card 26 or to card 28's list (and update the "eight paragraphs" count).

### [NIT:consistency] Card 28's paragraph locator points at the wrong paragraph
**Location:** batch 4 / card 28
**Issue:** The card says to rewrite "the paragraph immediately above [the Known contract gap], which describes this package classifying a target by `strings.Contains` on sight"; in `internal/cli/doc.go` the paragraph immediately above the gap is the failure-envelope "ok" key paragraph, and the `strings.Contains` paragraph is two above it.
**Fix:** Drop the positional clause and identify the paragraph by its opening words alone.

### [NIT:consistency] Card 31 leaves `target-echo-asymmetry`'s premise dead
**Location:** batch 4 / card 31
**Issue:** `internal/cli/cli_test.go`'s `target-echo-asymmetry` subtest exists to contrast a relativised path `Target` ("pkg" from an absolute argument) with a verbatim glyph `Target`; card 23 removes the relativisation, so no asymmetry remains, yet card 31 only reclassifies the absolute-path row's exit code and never says the test's name and second half go.
**Fix:** Say explicitly whether that subtest is deleted or renamed to what it now asserts.

## Verdict

REQUEST_CHANGES
One compile-breaking scope gap, one self-contradictory card, three unowned doc-comment edits.
MILL_REVIEW_END
