MILL_REVIEW_BEGIN
# Review: The glyph-maker: declaration to glyph (P1, roadmap 2b)

```yaml
duration_s: 279.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] RenderNameJSON's parameter type is never stated
**Section:** Scope; Technical context (`quarry/render.go`); Exit codes step 3
**Issue:** Every other new signature is spelled exactly (`Name(decls []Declaration) []NameResult`, `RenderNameText(r NameResult) string`, `codeForNameResult(r NameResult) int`), but `RenderNameJSON` appears only by name; the Text view decision scopes *only* the text renderer to one result ("no text rendering of a whole batch"), leaving JSON open on the single-versus-batch axis the facade decision treats as load-bearing.
**Fix:** State `RenderNameJSON`'s signature explicitly — one `NameResult` or `[]NameResult` — and say whether a batch has a JSON view at all.

### [NIT:scope] Three sibling doc comments carry the falsified four-step framing
**Section:** Verb-set statements → Help text and messages
**Issue:** The inventory names `cli.go:180,302` and "Run's doc comment", but `cli.go:308` (`runTOC`), `cli.go:370` (`runResolve`) and `cli.go:404` (`runExpand`) each open with "continuing from Run's shared four steps" — the exact sentence the `name`-needs-no-repository-root decision falsifies. Verified in source.
**Fix:** Add the three `runX` doc comments to the inventory alongside `Run`'s own, since one named edit has four sites, not one.

### [NIT:consistency] Q&A log carries two superseded answers as current
**Section:** Q&A log
**Issue:** Line 880 gives the MCP rationale as "fixed by the task and §9", which the No-MCP decision explicitly identifies as an earlier draft's wrong citation (§7 "How Loomyard uses it" carries the LLM-surfaces rule; §9 is Non-goals — confirmed in `docs/rewrite-plan.md`). Line 889 still reads "three counts pinned as constants … regenerated under `-update`", which the body calls two mechanisms that cannot both hold.
**Fix:** Correct line 880 to §7 and mark line 889 superseded by the r4 counts-golden answer, as other revised entries are marked.

## Verdict

REQUEST_CHANGES
One exported renderer signature is unstated on the batch-versus-single axis.
MILL_REVIEW_END
