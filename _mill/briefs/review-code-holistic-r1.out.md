MILL_REVIEW_BEGIN
# Review: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a) — holistic

```yaml
verdict: APPROVE
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-09-05
```

Verified end-to-end across all three batches against the overview, all `## Shared Decisions`, and
every batch file's cards.

- `quarry/view.go` + `quarry/view_test.go` (batch 1): `GlyphsAnswer` has the exact three
  untagged fields; `GlyphView` walks depth-first matching `dirBlocks`'s own order, clears
  Doc/Signature/SigEnd, sets `File` via `joinRel`, mutates nothing and shares no backing array
  (confirmed by the no-mutation test). `glyphSymbol`/`glyphsEnvelope` match the fixed five/three-key
  shapes verbatim, encode through the shared `renderJSON`. `RenderGlyphsText` implements the
  documented line grammar and empty-string/blank-line-separator rules exactly, and does not touch
  `writeSymbolLine`. `quarry/repo.go`'s `GlyphsOptions`/`Glyphs` normalise `""`→`"."` once and
  pass errors through unwrapped. `quarry/doc.go`'s query/renderer counts and the "adds no behaviour"
  and "only code it owns" clauses are all correctly updated to five queries / eleven renderers.
- `internal/cli/flags.go` (batch 2): `--view`'s closed vocabulary, the after-loop
  order-independent symbols-default/`--no-symbols` rejection, the five-verb gate, `glyphsPreset` as
  a `var` with the no-append discipline, and `parseGlyphsArgs`'s pre-scan (naming `glyphs` in every
  rejection, `--root` value consumption, unreachable re-parse) all match the plan's exact message
  strings and behaviour. `runTOC`'s glyphs branch in `cli.go` correctly ignores `targetIsFile` with
  the required explanatory comment. Doc-comment updates named by `doc-comments-carry-the-reasoning`
  (Run's pipeline, `usageText`, `internal/cli/doc.go`, `quarry/doc.go`, `quarry/repo.go`,
  `TestParseArgs_FiveVerbGate`) are all present and accurate.
- Card 12/13 tests (`glyphs_test.go`) correctly spell the expansion literally vs. reading
  `glyphsPreset`, per their differing purposes, exactly as required.
- Batch 3: the five new goldens (`glyphs-dir.txt`, `glyphs-dir-text.txt`, `glyphs-file.txt`,
  `glyphs-file-text.txt`, `toc-view-glyphs-depth.txt`) match the documented byte shape (no
  `signature`/`doc`/`sigend` keys, no directory/file lines under text); the depth golden has exactly
  172 `internal/boardengine` symbol lines as the plan's probe fixed. `after_test.go`'s header and
  `INDEX.md`'s counts/verb-span prose are both updated to twenty files / four verbs, with the
  depth row's note explaining the `toc` (not `glyphs`) spelling and the `internal/boardengine`
  target. `docs/rewrite-plan.md` §5 gains `[--view full|glyphs]` plus the two required paragraphs;
  `docs/roadmap.md` has point 2a removed and its intro correctly renumbered to two surfaces.

No file outside the plan's reference lists was found; `internal/engine/` is untouched; no helper is
duplicated across batches; no Shared Decision is violated.

## Verdict

APPROVE
Implementation matches the plan and its Shared Decisions faithfully across all three batches; no findings.
MILL_REVIEW_END
