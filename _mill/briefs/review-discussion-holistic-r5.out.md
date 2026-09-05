MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), high effort
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] D10a's `Unit` reversal not propagated to D14/Testing/Q&A
**Section:** D14, Testing (`quarry/text_test.go`, engine rows), Q&A entry 9
**Issue:** D10a(b) sets `Unit` on a self-glyph `not_found`, but D14 still asserts "prints `<path># not_found` with no unit suffix (D10 leaves `Unit` empty)", `quarry/text_test.go`'s row pins that same no-suffix output, and Q&A entry 9 still answers "`unit` never" — while the engine Testing row correctly pins `unit: not_found`. `quarry/text.go:243–252` prints `" (unit found)"`/`" (unit not_found)"` whenever `r.Unit != ""`, so the two Testing rows demand contradictory bytes and one of them must fail.
**Fix:** Rewrite D14's parenthetical and the `quarry/text_test.go` row to expect `<path># not_found (unit found|not_found)`, and mark Q&A entry 9 superseded by the D10a entry.

### [BLOCKING:scope] D7's falsified-paragraph inventory is wrong in count and in location
**Section:** D7 (i)–(vi), Q&A "How many `internal/cli` doc paragraphs does this falsify?"
**Issue:** The Q&A answers "Four, all named in D7" while D7 names six — a direct self-contradiction in an inventory D7 declares complete. Worse, D7 (iv) attributes two claims to `cli.go`'s `Run` doc comment *step 1*, but only the `strings.Contains` narration is there (`cli.go:200–206`); "the parser has already guaranteed the target contains a `#`" is a separate paragraph at `cli.go:219–221`, the `runExpand` preamble. A plan writer editing "step 1" leaves that sentence standing, and it is exactly what D19 falsifies.
**Fix:** Reconcile the Q&A count with D7 and split (iv) into two entries with their real line ranges, `cli.go:200–206` and `cli.go:219–221`.

### [BLOCKING:scope] `glyph/` doc-comment inventory omits two falsified statements
**Section:** Scope §In, `glyph/` bullet
**Issue:** The bullet names exactly two comment edits — `glyph/glyph.go:17–19` and `ParseError.Detail` at `glyph/errors.go:99–100` — and nothing else. Two further statements in the same package become false: `glyph/doc.go:1–2` ("Package glyph names one symbol in a repository: the form is unit \"#\" member, one name for one source symbol"), which the self form contradicts since `internal/logger#` names a directory and `…/logger.go#` a file; and `glyph/errors.go:18` ("ReasonNoSeparator fires when the input has no \"#\": a glyph needs a unit and a member"), false once a member is optional. D4 rewrites only the `reasonText` map entry at line 82, not the constant's own comment. The `Detail` sentence is also duplicated on the field itself at `errors.go:108–109`, outside the cited 99–100 range.
**Fix:** Add `glyph/doc.go:1–2`, `glyph/errors.go:18` and `glyph/errors.go:108–109` to the `glyph/` scope bullet with their required new wording.

## Verdict

REQUEST_CHANGES
Two doc-comment inventories are incomplete and D10a's `Unit` reversal contradicts D14 and Testing.
MILL_REVIEW_END
