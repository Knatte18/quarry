MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Card 17 states no rule that makes invalid UTF-8 an error entry
**Location:** batch 4 / card 17 (with cards 22 and 48)
**Issue:** Card 17's disposition rules cover only a pre-set `Refusal`, an unregistered strategy, nil/nil slices, and "a failure to extract a side at all", but `WithTree` in `internal/engine/treesitter/treesitter.go` performs no UTF-8 validation and returns an error only for an unwired language, a `SetLanguage` failure, or a nil tree — so undecodable bytes parse to a partial tree and would set a lossy flag, never `disposition: error`; card 22 nonetheless requires "an entry whose bytes are not valid source giving an error disposition" and card 48's `entry-error` golden pins that same case (the discussion names invalid UTF-8 as the failure).
**Fix:** State the UTF-8 rejection explicitly in card 17 as an extraction failure per side, as card 6 already does for `PackageClause`, so the rule the tests and the golden assert has a stated implementation.

### [BLOCKING:decision] Card 37 edits internal/mcpserver with no recorded departure
**Location:** batch 6 / card 37 (with card 51 and the overview's Shared Decisions)
**Issue:** The discussion's Scope "Out" list states "Nothing in `internal/mcpserver` or `cmd/quarry-mcp` changes", yet card 37 edits `internal/mcpserver/layering_test.go`; the plan's first Shared Decision requires any departure to be "its own Shared Decision naming the passage it overrides — or it is a defect rather than a decision", and names the goldens-location decision as the only one, while card 51 separately calls this "the single deliberate exception".
**Fix:** Record the mcpserver edit as its own Shared Decision naming the Scope "Out" bullet it overrides, or drop card 37 and keep the git-import rule as convention.

### [BLOCKING:scope] internal/repopath/target.go's stale toc-only claims are unowned
**Location:** batch 8 / card 44 (and `## All Files Touched`)
**Issue:** Card 44 routes the delta target through `repopath.RepoRelTarget`, which falsifies two doc comments in `internal/repopath/target.go` — `repoRelTarget`'s "toc is the only verb that reaches this function's exported form" and `RepoRelTarget`'s "the CLI's toc verb, and the MCP server's own target conversion" — but that file appears only in card 44's `Context:`, in no card's `Edits:`, and not in `## All Files Touched`, so the implementer cannot correct them; the plan corrects the equivalent stale claim in `internal/cli/doc.go`, `flags.go`, `usage.go`, `cli.go`, `quarry/doc.go` and `quarry/repo.go`.
**Fix:** Add `internal/repopath/target.go` to card 44's (or card 45's) `Edits:` and to `## All Files Touched`, naming the two comments.

### [NIT:consistency] Card 6 rests its read-ordering gate on an environment-gated test
**Location:** batch 2 / card 6
**Issue:** Card 6 argues a read-before-extension-check regression is "one the timing assertion in `internal/engine/loomyard_timing_test.go` selected by this batch's `verify:` would catch", but `TestResolve_TwentyGlyphsUnder150ms` calls `loomyardRepo`, which per `internal/engine/loomyard_test.go` skips whenever `LADDER_LOOMYARD_REPO` is unset — the normal state on most machines — and also skips under `-short`.
**Fix:** Restate the claim as "catches it only where a pinned Loomyard checkout exists", so the ordering rule is not presented as mechanically guarded everywhere.

### [NIT:consistency] Card 48 miscounts the existing `-update` flag precedents
**Location:** batch 9 / card 48
**Issue:** The card says to follow "the two existing precedents in this repository", but three packages declare their own `-update` flag today: `internal/engine/loomyard_test.go`, `internal/mcpserver/toc_golden_test.go` and `internal/cli/loomyard_test.go`.
**Fix:** Say three, so the per-package-binary rationale reads against the real set.

## Verdict

REQUEST_CHANGES
Three blocking gaps: unstated UTF-8 rule, unrecorded mcpserver departure, unowned repopath doc.
MILL_REVIEW_END
