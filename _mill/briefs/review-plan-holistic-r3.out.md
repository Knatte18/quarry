MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic, Opus-class; ID per environment, self-assessment consistent)
reviewed_file: plan/
date: 2026-08-28
```

## Findings

### [BLOCKING:design] ErrLanguageUnsupported: two conflicting CLI dispositions
**Location:** Overview "Decision: exactly one new engine sentinel" vs. batch 6 cards 36–38
**Issue:** The decision states the sentinel is "classified in `internal/cli` with `errors.Is`", but card 36 only emits `output.Err` with the error's message, and card 38 maps the unsupported-language outcome to `statusError` — the same status every other failure gets — so no card ever calls `errors.Is` on it and the plan never says which of the two approaches internal/cli implements.
**Fix:** Either name the card and the behaviour the `errors.Is` classification produces (distinct message, distinct status, or distinct exit code), or amend the decision to say the CLI reports the wrapped error verbatim and the facade re-export exists for external consumers and card 35's identity test.

### [NIT:consistency] Card 17 cites a non-existent "card 4" rule
**Location:** Batch 3 card 17, `LeadingBlocks` rationale
**Issue:** It justifies returning block text by "card 4's 'no new shared helper per strategy' rule"; card 4 is the treesitter backend test card, and that rule is stated in batch 4's Batch Scope.
**Fix:** Cite batch 4's Batch Scope instead of card 4.

### [NIT:scope] Card 51 undercounts cli.go's stale-prose sites
**Location:** Batch 8 card 51 ("three stale-prose sites in this file")
**Issue:** `cli.go:856` reads `// batchStatus is the per-symbol outcome in batch mode.`; card 38 reuses `batchStatus` unchanged for path-keyed entries, so that comment goes stale too, and it falls under the batch's own "no prose may claim a batch entry is keyed on a symbol" invariant. Card 51 names only the package doc verb list, the package doc batch paragraph, and the root `Short`.
**Fix:** Add the `batchStatus` doc comment to card 51's site list and correct the count.

### [NIT:consistency] Card 52 leaves the seam guard's "five-package DAG" unowned
**Location:** Batch 8 card 52 vs. `internal/quarryengine/seam_enforcement_test.go:11`
**Issue:** Card 52 charters only the header's package *enumeration*, and its "do not restate a count this card has no authority over" sentence can be read as excluding the "five-package DAG" count in the same header, which no other card owns; card 54's `five-package` grep will hit it with no assigned fixer.
**Fix:** Name that phrase explicitly in card 52 as a site it owns.

### [NIT:consistency] Card 34 asks for an import that already exists
**Location:** Batch 6 card 34, final import instruction
**Issue:** It says to import `.../toc` and `.../registry` "alongside the existing engine imports", but `quarry/facade.go:22` already imports `registry`; only the `toc` import is new.
**Fix:** Say only `toc` is added.

### [NIT:consistency] Card 54's remedy is unreachable across batches
**Location:** Batch 8 card 54
**Issue:** It says a stale site found by the sweep is fixed by amending "whichever earlier card owns that file"; `quarry/facade_test.go` is in its Context but is owned by batch 6 card 35, whose commit is already landed by the time batch 8 runs.
**Fix:** State that a stale site owned by an earlier *batch* is fixed by a new commit in batch 8, and name which card takes it.

## Verdict

REQUEST_CHANGES
One conflicting sentinel disposition; the rest are small prose-ownership and cross-reference defects.
MILL_REVIEW_END
