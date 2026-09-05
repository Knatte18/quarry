MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] `expand <bare-path>` cannot carry D4's message
**Section:** D19 rationale + Testing (`internal/cli/cli_test.go`)
**Issue:** `runExpand`'s parse-error branch (`internal/cli/cli.go:418`) builds `"expand " + req.target + ": " + string(parseErr.Reason)` — the bare word `no_separator`, never the `*ParseError`'s text — so deleting the flags.go gate yields `expand internal/logger: no_separator`; D19's "gives `expand` the actionable repository-relative hint" and Testing's row "carrying D4's actionable repository-relative message" are both false, and no decision or Scope bullet touches that branch.
**Fix:** either add `runExpand`'s `*glyph.ParseError` branch to the edit inventory with its new message source, or drop the actionable-message claim from D19 and from the cli_test row.

### [BLOCKING:design] D8's `toc` exit-2 branch leaves `usageText` undecided
**Section:** D8 / Testing (`internal/cli/cli_test.go`)
**Issue:** both existing exit-2 sites call `fail(..., withUsage: true)` and print `usageText`, while every `fail` inside `runTOC` passes `false`; D8 fixes the code and the sentence but not the flag, and Testing's "`toc 'a#b'` → exit 2 with the usage message" reads either way. Two plan writers would pin different stderr bytes.
**Fix:** state explicitly whether the separator reject prints `usageText` after its sentence, and pin that choice in the cli_test row.

### [BLOCKING:scope] `usage.go`'s `resolve <glyph|path>` line left false
**Section:** Scope (`internal/cli`) / D19 / D16(c)
**Issue:** D16(c) rewrites the identical heading `resolve <glyph|path>` in `docs/rewrite-plan.md` §5, but `internal/cli/usage.go:21` still reads `quarry resolve <glyph|path>` and is only inspected "for wording that promises the old [expand] usage error" — so `--help` keeps promising a form `resolve` no longer accepts.
**Fix:** add usage.go's resolve usage line to the edit inventory with its replacement text.

### [NIT:scope] Two further falsified doc paragraphs unnamed
**Section:** D7 / Technical context (`internal/cli`)
**Issue:** D7 names the "Known contract gap" paragraph and the one above it, but `internal/cli/doc.go:11–13` ("`resolve` takes either a path, by that same rule, or a glyph") and `cli.go`'s `Run` doc comment (runResolve step 1's `strings.Contains` classification; "the parser has already guaranteed the target contains a `#`") also become false and appear in no inventory.
**Fix:** name both in D7's edit list.

### [NIT:consistency] D12 names a function `SpansOf` does not call
**Section:** D12
**Issue:** D12 says a self glyph "reaches `matchesFor`"; `SpansOf` (`internal/engine/resolve.go:610–615`) keeps its own inline owner/name filter and never calls `matchesFor` — a deliberate duplication `matchesFor`'s own doc comment records. The empty-slice conclusion holds; the mechanism cited does not.
**Fix:** restate D12 against `SpansOf`'s inline filter.

### [NIT:design] `IsSelf()` ignores `Params`, which D1's representation includes
**Section:** D1
**Issue:** D1 defines self as `Owner == nil && Name == "" && Params == nil`, but the predicate tests only `Owner` and `Name`; a hand-built `Glyph{Unit: "a", Params: []string{}}` reports `IsSelf() == true` while `String()` prints `a#()`, breaking the removing-the-`#`-yields-the-path property for exactly the hand-built values D20 and `SpansOf` exist to guard.
**Fix:** say whether `IsSelf` also requires `Params == nil`, or why the two-field form is deliberate.

## Verdict

REQUEST_CHANGES
Three verified contradictions with source; the self-form design itself is sound.
MILL_REVIEW_END
