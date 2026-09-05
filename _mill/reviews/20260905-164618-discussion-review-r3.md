MILL_REVIEW_BEGIN
# Review: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)

```yaml
duration_s: 232.6
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (asserted by runtime environment; no independent means to verify)
reviewed_file: /home/knatte/Code/quarry/wts/glyphs-verb/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [NIT:consistency] JSON wire type for the answer is undeclared
**Demoted-from:** BLOCKING
**Section:** `glyphs-answer-shape` + `glyphs-json-shadow-struct`
**Issue:** `glyphs-answer-shape` attributes the emitted bytes to `GlyphsAnswer`'s own tags ("`Symbols` carries `json:"symbols"` with no `omitempty`, so a zero-symbol answer emits `"symbols": []`"), but `glyphs-json-shadow-struct` says the JSON is not produced by marshalling those `Symbol` values — `RenderGlyphsJSON` "encodes the result" of mapping into `[]glyphSymbol`, which cannot be `GlyphsAnswer` (its field is `[]Symbol`) and must not be a bare array (`glyphs-answer-shape` rejects that). The envelope type actually handed to `renderJSON`, and its `target`/`symbols`/`incomplete` tags, is never declared. Consequence left undecided: `GlyphsAnswer` is a new exported type whose tags, if kept, make `json.Marshal(ans)` by a facade caller emit `"signature": ""` on every symbol — a different document from `RenderGlyphsJSON`'s, i.e. tags that promise a key set the type does not produce.
**Fix:** State which value `RenderGlyphsJSON` passes to `renderJSON` (declaring the wire envelope struct and its tags if that is the answer), and state the disposition of `GlyphsAnswer`'s own JSON tags — kept and divergent, or dropped.

### [NIT:scope] Depth golden's directory target is unnamed and unbounded
**Section:** Testing → "Golden tests over the pinned checkout"
**Issue:** `toc-view-glyphs-depth.txt` is spelled `<dir-with-a-subtree>`, deferred to an implementer probe against the pin, with a fallback ("say so in `testdata/INDEX.md`") that makes the golden itself conditional; the INDEX row count and `after_test.go`'s "fifteen files" therefore have two possible outcomes. Every existing golden is scoped to `internal/logger` (confirmed to have no `dirs` key at the pin), so the new target is a fresh subtree with no stated size bound on a committed `--depth 1 --view glyphs` listing.
**Fix:** Name the candidate directory (or a small ordered candidate list) and state the acceptable golden size, so the plan's file inventory and counts are unconditional.

### [NIT:design] "Absent line means absent symbol" is unqualified under `--depth N`
**Section:** `incomplete-is-explicit`
**Issue:** The stated consumer invariant is unscoped, but `toc --view glyphs --depth 1` is an explicitly supported and goldened invocation in which deeper directories contribute no symbols and no `incomplete` entry — the same wrong-negative the decision exists to prevent, through a second door. `GlyphView` cannot even detect it: a depth-cut `DirAnswer` with no `Files`/`Dirs` is indistinguishable from an empty leaf.
**Fix:** Scope the invariant to the frozen `--depth all` preset in one sentence, and state explicitly that depth truncation contributes nothing to `Incomplete`.

## Verdict

APPROVE
One blocking gap: the glyphs view's JSON envelope type and `GlyphsAnswer`'s tags are undecided.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
