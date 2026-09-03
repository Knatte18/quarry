MILL_REVIEW_BEGIN
# Review: The glyph package (T1)

```yaml
duration_s: 234.0
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude, Opus-class (reported ID claude-opus-5)
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [NIT:consistency] Detail-per-reason rule contradicts itself
**Demoted-from:** BLOCKING
**Section:** Decisions → "One `*ParseError` …" → "What `Detail` holds, per reason"
**Issue:** The rule splits the fifteen reasons into "eleven that name a specific piece" and "four where `Detail` is the empty string", then states the reader can tell them apart by a blank `Detail` — but three of the eleven necessarily produce a blank `Detail`: `unit_empty` (the whole unit half, which is empty by definition), `unit_empty_segment` (`a//b` → offending segment is `""`) and `member_empty_component` (`a..b` → offending component is `""`). The clause "the whole unit half for `unit_empty`'s sibling cases where no smaller piece is at fault" also names no cases, so a plan writer cannot tell which reasons it covers.
**Fix:** State `Detail` per reason unambiguously (or drop the "blank means one of the four" invariant), and replace "`unit_empty`'s sibling cases" with the explicit reason names.

### [NIT:design] Invalid UTF-8 input has no stated disposition
**Section:** Decisions → "The Go unit alphabet" / "The Go member alphabet"; Testing
**Issue:** Both alphabets are defined over runes (`unicode.IsSpace`, `unicode.IsLetter`, ASCII control), but the discussion never says what an input carrying invalid UTF-8 does. Under the stated rules a bad byte in the unit half is *accepted* (not `\`, not control, not space) and round-trips, while in the member half it becomes `member_not_identifier` with a U+FFFD `Detail` rather than the offending byte.
**Fix:** Record the disposition explicitly — accepted in the unit, rejected in the member — or add a reject, and add one table row either way.

### [NIT:consistency] Zero-Glyph-on-error is pinned for one reason only
**Section:** Testing → `parse_test.go`; the reject table
**Issue:** `unsupported_language` is specified to return "that reason and the zero `Glyph`"; for the other fourteen reasons the returned `Glyph` value is unspecified, so a plan writer may legitimately return a partially populated struct (e.g. `Lang` and `Unit` set when the member half fails). The reject table asserts `Reason` only and would not catch the difference.
**Fix:** State once that `Parse` returns the zero `Glyph` on every error, and have the reject table assert it.

## Verdict

APPROVE
The `Detail` specification contradicts its own blank-means-no-detail invariant for three reasons.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
