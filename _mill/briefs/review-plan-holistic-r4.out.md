MILL_REVIEW_BEGIN
# Review: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a) — holistic

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [NIT:consistency] Two --root doc claims go stale, unnamed by any card
**Location:** batch 2 / card 9 (and card 7)
**Issue:** `internal/cli/flags.go:50-51`'s `parseArgs` doc says "--root is valid for the three repository verbs only", and `internal/cli/flags_test.go:271`'s `TestParseArgs_TextOnEveryVerbRootOnRepositoryVerbs` doc repeats it; card 9's pre-scan accepts `--root` on a fifth verb, so both become incomplete — the same staleness card 9 already catches for `TestParseArgs_FourVerbGate`, but these two are not named anywhere.
**Fix:** name both in card 9's already-mandated `parseArgs` doc-comment extension, so the overview's `doc-comments-carry-the-reasoning` rule covers every comment this card falsifies.

### [NIT:consistency] Batch 2's Batch Tests contradicts card 10 and omits card 11
**Location:** batch 2 / `## Batch Tests`, final paragraph
**Issue:** it lists `internal/cli/cli_test.go` under "files it covers as a regression gate **without changing**", but card 10's `Edits:` is exactly `internal/cli/cli.go` and `internal/cli/cli_test.go`; it also omits `internal/cli/usage_test.go`, which card 11 `Creates:`.
**Fix:** move `cli_test.go` to the changed list and add `usage_test.go` (cards 10 and 11), so the batch's own coverage prose matches its cards.

### [NIT:consistency] Card 3 states the zero-symbol incomplete block two ways
**Location:** batch 1 / card 3
**Issue:** the rule bullet says "when `a.Incomplete` is non-empty, a single blank line follows the symbol lines, then one line per path", which for a zero-symbol answer yields a leading `"\n"`; the test bullet then requires "the leading blank line does not produce a leading empty line before any symbol line exists". The rule and its test disagree about the emitted bytes for the one case the test names.
**Fix:** state the separator rule conditionally in the rule bullet — the blank line is emitted only when `a.Symbols` is non-empty — so the renderer spec and its own test agree.

### [NIT:scope] Card 14 never gives `verbArgs` for the five new golden rows
**Location:** batch 3 / card 14
**Issue:** `afterGoldenCase` has five fields and `TestAfterGoldens` builds argv from `verbArgs` (`internal/cli/after_test.go:169`), but the card specifies only `golden`, `verb`, `invocation` and `exitCode` per row — while explicitly contrasting `invocation` against `verbArgs`, so the omission reads as deliberate rather than as "derive it".
**Fix:** spell each row's `verbArgs` slice, or state once that it is the `invocation` string split on spaces.

### [NIT:consistency] Shared Decision 1's absolutism is not scoped to the two overrides
**Location:** overview / `the-discussion-decisions-are-binding` vs `glyphs-options-is-exported`, `the-preset-is-a-var-not-a-const`
**Issue:** decision 1 makes every discussion decision "a fixed constraint on this plan, not a suggestion", yet decision 2 deliberately overrides `facade-shape`'s `glyphsOptions()` spelling and decision 4 overrides `preset-expansion`'s "package-level constant". Both overrides are justified; decision 1 does not carve them out, so it formally forbids two of its own siblings.
**Fix:** add a clause to decision 1 naming the two decisions this plan deliberately supersedes and pointing at their entries.

## Verdict

APPROVE
Sound, source-accurate plan; five prose-level nits, none affecting behaviour.
MILL_REVIEW_END
