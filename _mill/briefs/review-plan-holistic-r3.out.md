MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:scope] Batch 1 breaks TestGoStrategy_Symbols; no card updates it
**Location:** batch 1 (cards 2–4), batch tests
**Issue:** `assertSymbolsEqual` in `internal/engine/golang_test.go:300` compares the whole `Symbol`
value with `reflect.DeepEqual`, zeroing only `Glyph`; `TestGoStrategy_Symbols`' ~17 rows spell
`Symbol` literals whose new `DeclStart`/`BodyStart`/`DeclEnd` are zero, so filling them in cards 2–4
fails every row — contradicting the batch's own claim that "the broader `internal/engine` suite has
nothing new to say about it" and the Shared Decision that no existing behaviour changes.
**Fix:** add a card in batch 1 editing `internal/engine/golang_test.go` to carry the three offsets in
its expectations (or to zero them in `assertSymbolsEqual`'s copy the way `Glyph` is zeroed), list
that file in `## All Files Touched`, and include `TestGoStrategy` in batch 1's `verify:`.

### [BLOCKING:scope] Card 42's verb-list message change breaks TestParseArgs_UsageErrors
**Location:** batch 8, cards 42 and 46
**Issue:** card 42 rewrites the two `"no verb given; expected: toc, resolve, or expand"` strings at
`internal/cli/flags.go:61` and `:66`, but `internal/cli/flags_test.go:156` and `:158` hand-copy those
exact bytes in `TestParseArgs_UsageErrors`; card 46 is the only card editing that file and never
mentions them, so batch 8's own `verify:` (`-run TestParseArgs`) fails.
**Fix:** name `TestParseArgs_UsageErrors`' two `missing-verb`/`first-arg-is-flag` rows in card 46's
Requirements as expectations to update to the four-verb sentence — the analysis card 43 already
performs for `usageText` (verified: every usage-text comparison references the constant, so card 43's
claim holds; this one does not).

### [BLOCKING:consistency] rewrite-plan.md §1's "three queries" left stale by card 50
**Location:** batch 9, card 50
**Issue:** `docs/rewrite-plan.md:12` reads "three queries over one tree-sitter parse: `toc` …,
`resolve` …, `expand` …"; card 50 edits only §5's query paragraphs and §7's mechanical-use list, so
the document card 50 owns keeps a false count while the plan corrects the identical claim in
`quarry/doc.go` (card 35), `internal/cli/doc.go` (card 45), `flags.go` (card 42) and `cli.go` (card 44).
**Fix:** add §1's enumeration to card 50's Requirements as a third edit in the same commit.

### [NIT:consistency] Two file header comments go stale unfixed
**Location:** batch 6 card 31; batch 8 card 43
**Issue:** `quarry/quarry.go:1` says it declares the aliases making "the engine's answer shape and
error identity nameable" — card 31 adds `internal/gitsrc` aliases there and does not amend it;
`internal/cli/usage.go:11-16` says the flags list is "one combined list rather than three per-verb
sections … not repeated three times", which card 43 makes false and explicitly leaves alone.
**Fix:** add both header corrections to the same commits, as cards 38 and 7 already do for their files.

### [NIT:scope] Text view's rendering of a modified entry's `before` array is unspecified
**Location:** batch 7, cards 39 and 41
**Issue:** card 39 says to reuse `writeSymbolLine` for "every symbol the delta emits", but a modified
entry's `before` is `[]SymbolLocation` (card 11: `file`, `start`, `sigend`, `end`), and
`quarry/text.go:172`'s `writeSymbolLine` takes a `Symbol` and emits `ID`, `Signature` and `Doc`,
which a location has none of — so the one shape that cannot use the shared writer is the one shape
card 39 gives no grammar for, while card 41 asserts a multi-occurrence `before` array losslessly.
**Fix:** state the location line's grammar in card 39's Requirements alongside the kind rule.

### [NIT:scope] Existing all-verbs flag-validity test not extended to the fourth verb
**Location:** batch 8, cards 42 and 46
**Issue:** `internal/cli/flags_test.go:258`'s `TestParseArgs_TextAndRootValidForAllVerbs` tables three
verbs; card 42 makes `flags.go`'s doc claim `--text` and `--root` are valid for four, and card 46's
validity matrix covers only the two new flags, so the widened claim goes unpinned.
**Fix:** add the two delta rows to that test in card 46's Requirements.

### [NIT:design] gitsrc left outside the only mechanical layering gate
**Location:** batch 6, card 37
**Issue:** `internal/mcpserver/layering_test.go:39` forbids `internal/engine` by path prefix only;
card 37 notes this batch "adds a second internal package a future edit could reach for" yet closes
the gap with a one-time manual inspection rather than a row in that test.
**Fix:** have card 37 (or a card in batch 5/6) add `internal/gitsrc` to `forbiddenEngineImport`'s
check as a second forbidden path, so the confirmation survives the task.

## Verdict

REQUEST_CHANGES
Two batches break existing tests their own verify runs; one doc count left stale.
MILL_REVIEW_END
