# Discussion fix — round 3

Source review: `_mill/reviews/20260905-111512-discussion-review-r3.md` (verdict REQUEST_CHANGES, reviewer `opushigh`, 1 BLOCKING / 5 NIT, 2 demoted from BLOCKING).

All six findings verified against the worktree before fixing. Under `--orch` the decision tree's PUSH BACK path is unavailable; every finding was treated as FIX. This round also folds in **task item 5**, added to the wiki body by the orchestrator while round 3 was running.

## Fixed

- "[NIT:consistency] `expand <bare-path>` cannot carry D4's message": verified `internal/cli/cli.go:418` builds `"expand " + req.target + ": " + string(parseErr.Reason)` — the bare reason word. D19 now changes that branch to `"expand: " + parseErr.Error()` in the same edit and records the consequence for `expand`'s existing grammar rejections; the rationale's actionable-hint claim is conditioned on the message-source change; the `cli_test.go` row now requires asserting the full sentence and pins `expand a#b#c`'s changed shape.

- "[BLOCKING:design] D8's `toc` exit-2 branch leaves `usageText` undecided": verified `cli.go:251` and `cli.go:269` pass `withUsage: true` while every `fail` in `runTOC` passes `false`. D8 now fixes `fail(..., exitUsage, msg, true)` explicitly, with the reasoning (the flag tracks the exit code, not the enclosing function) and the note that either choice had local precedent. The `cli_test.go` row requires asserting sentence *and* `usageText` block, since a substring check would pass on the wrong bytes.

- "[NIT:scope] `usage.go`'s `resolve <glyph|path>` line left false": verified `internal/cli/usage.go:21`. Added **D21** making `usage.go` a named part of the contract surface: line 21 becomes `quarry resolve <glyph>`, the exit-codes block is confirmed against D19 rather than assumed, and the file's ASCII-only byte-comparable constraint is preserved. D19's weaker "checked for wording" sentence now defers to D21; Scope-In and the module map name the file.

- "[NIT:scope] Two further falsified doc paragraphs unnamed": verified `internal/cli/doc.go:11–13` ("`resolve` takes either a path, by that same rule, or a glyph") and `cli.go`'s `Run` doc comment ("the parser has already guaranteed the target contains a `#`"). D7's inventory goes from two paragraphs to four, each named with what makes it false.

- "[NIT:consistency] D12 names a function `SpansOf` does not call": verified `SpansOf` keeps its own inline owner/name filter and never calls `matchesFor` — a duplication `matchesFor`'s doc comment records deliberately. D12 restated against the inline filter; the empty-slice conclusion is unchanged.

- "[NIT:design] `IsSelf()` ignores `Params`, which D1's representation includes": verified against `String()`'s doc comment (nil-versus-non-nil `Params`, not length, decides the parentheses). `IsSelf` now tests all three fields, with the `a#()` counter-example recorded as the reason.

## Folded in (not a review finding)

- **Task item 5 — composition is API, never manual concatenation.** Re-read from the wiki. D20 promoted from a D16(b) consequence to a first-class decision carrying the body's rationale, its two named rejects (decision 3's separator rule, no empty path), and the delegation-to-`Parse` mechanism with the sole concatenation deliberately inside the package. The `Self(lang, path)` versus `Self(path)` question is decided for `lang` with the package-convention argument. Testing gains group (a) compose∘parse round trip in both directions (the body's done-when wording — path identity, not merely successful parse), group (b) rejects mirroring decision 3 driven from a shared table with `Parse`, and group (c) the end-to-end no-concatenation round trip; `glyph/self_test.go` is added as TDD candidate 4. Problem, Scope-In and the scenario list updated from four adjustments to five.

## Pushed Back

None.
