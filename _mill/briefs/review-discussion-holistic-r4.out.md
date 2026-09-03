MILL_REVIEW_BEGIN
# Review: The glyph package (T1)

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5)
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [NIT:consistency] Whitespace-ban blast radius understated
**Section:** Decision "The Go unit alphabet" + Open question 2 **Issue:** Both say dropping the ban "deletes one predicate in `golang.go` and one reject-table row, and nothing else moves", but the reject table has two whitespace-only unit rows (`internal/my logger#run`, `" internal/logger#run"`), two more annotated "doubly invalid ... **and** a space", and the Decision "Input is not trimmed" asserts leading whitespace is a `unit_bad_rune` reject. **Fix:** State the real radius: two rows deleted, two annotations reworded, and the untrimmed-input decision's unit-half half retracted.

### [NIT:design] `ParseError.Detail` undefined for four reasons; message untested
**Section:** Decision "One `*ParseError` …" **Issue:** `Detail` is specified as "the offending segment, component or rune", but `no_separator`, `unsupported_language`, `member_empty` and `member_too_deep` have none, and no test asserts anything about `Error()` — so Scope's in-scope requirement "an error whose message names what was wrong" is entirely unverified and an exported field of the one package everyone imports is left to the implementer. **Fix:** Say what `Detail` holds for those four (empty is fine) and add one smoke assertion that `Error()` is non-empty and distinct per `Reason`.

### [NIT:scope] §3's receiver-pair example is not in the accept table
**Section:** Testing → `golang_test.go` accepts **Issue:** §3 writes `dualHandler.Handle` and `durableHandler.Handle` as "two glyphs"; the enumerated accept list omits both, while the Decision "Python and C# examples as split-only tests" claims the criterion "is met for every Go example". **Fix:** Add the pair (unit-prefixed, as `Box` and `init` already are) to the accept list.

### [NIT:consistency] Unit segment bans `#`, but no reason covers it
**Section:** Decision "The Go unit alphabet" vs. the `Reason` table and unit precedence **Issue:** The alphabet says a segment "must contain no `/`, no `#`, no `\`…", but `unit_bad_rune` is defined as `\`/control/whitespace only and step 2c omits `#` — unreachable after a first-`#` split, so the clause is dead text a reader will try to implement. **Fix:** Drop `#` from the segment rule, or note it is structurally unreachable.

### [NIT:scope] Test package placement stated for only one file
**Section:** Testing **Issue:** `parse_test.go` is pinned white-box/same-package, but `golang_test.go` and `string_test.go` are not — yet the round trip is "driven from the same table as the accept cases" and the `Reasons` completeness test must range over reject rows in two files, both of which require one shared test package. **Fix:** State that all three test files are `package glyph`.

## Verdict

APPROVE
Complete and internally consistent; five NITs are wording and test-table gaps, none blocking.
MILL_REVIEW_END
