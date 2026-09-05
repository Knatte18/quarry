# Review: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] glyphs-answer-shape rests on a false omitempty premise
**Section:** Decisions / glyphs-answer-shape (and the Testing bullet "the glyphs JSON renderer")
**Issue:** The decision's rationale claims "the cleared fields simply drop out of the JSON,
because `doc`, `signature` and `sigend` all carry `omitempty` while `id`, `kind`, `file`,
`start` and `end` do not." That is false on both halves as the code stands:
`internal/engine/answer.go`'s `Symbol.Signature` tag is `json:"signature"` with **no**
`omitempty` (and `File` **does** carry `omitempty` — harmless here since the view always
fills it, but the sentence misstates it). Reusing the engine's `Symbol` with `Signature`
cleared therefore emits `"signature": ""` on every symbol of the glyphs view, which
contradicts this discussion's own JSON-renderer test requirement that `doc`/`signature`/
`sigend` be "absent from every symbol". The plan writer cannot produce the promised key set
from the decided mechanism.
**Suggested fix:** Decide the actual mechanism and record it: either (a) the view marshals
through its own thin shadow struct (or a custom `MarshalJSON` on a local type) that omits
the three fields — noting how that squares with the one-symbol-shape rationale — or
(b) a decided contract change adding `omitempty` to `Symbol.Signature` in
`internal/engine/answer.go` (that file's doc comment requires an explicit decision for any
key-set change; also state why no existing golden changes: no committed golden emits an
empty `signature`, and toc/resolve/expand always fill it). Correct the `file` half of the
sentence either way.

## Verdict

REQUEST_CHANGES
One false code premise breaks the view's promised JSON key set; everything else verified against the codebase and is coherent.
