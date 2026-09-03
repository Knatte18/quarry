MILL_REVIEW_BEGIN
# Review: The glyph package (T1)

```yaml
duration_s: 212.0
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic), high reasoning effort
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [NIT:consistency] Q&A log re-bans `#` in the unit segment
**Section:** Q&A log, "What characters may a unit segment contain?" **Issue:** The answer lists `#` among the banned runes, while the Decision "The Go unit alphabet" states `#` "needs no ban and gets none" and says explicitly not to implement an unreachable `#` check. **Fix:** Drop `#` from that Q&A answer, or annotate it as superseded by the Decision.

### [NIT:design] String's owner-join rule is never stated
**Section:** Decisions → "No constructor, and String is a pure printer"; Testing → `string_test.go` **Issue:** The `Params` printing rule is pinned now for C#'s sake, but the rule for joining a multi-element `Owner` with `Name` is never written down — the over-deep-`Owner` case asserts only that `String()` returns. **Fix:** State the print form (`Unit "#" join(Owner…, Name, ".") [params]`) so the later alphabets inherit it rather than re-deciding it.

## Verdict

APPROVE
Decisions, reject vocabulary, precedence and test tables are complete and match the spec.
MILL_REVIEW_END
