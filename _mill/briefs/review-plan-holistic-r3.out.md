MILL_REVIEW_BEGIN
# Review: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Anthropic Claude, Opus-class; harness reports claude-opus-5, which I cannot verify from inside the session
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:scope] usageText gains five entries with no test at all
**Location:** batch 2 / card 11 (Edits: `internal/cli/usage.go`, `internal/cli/doc.go` — no test file)
**Issue:** `internal/cli/name_test.go`'s `TestUsageText_NameVerb` (lines 113-128) is this package's established pattern for pinning a new verb's usage line, its flag row and the ASCII-only rule; card 11 adds a `glyphs` usage line, a `--view <name>` flag row and a literal preset-expansion block and pins none of them. The discussion's `preset-expansion` decision names `usageText` as one of three places the frozen tokens are spelled, and card 12 deliberately refuses to read `glyphsPreset`, so nothing mechanically ties the help text's copy of `--view glyphs --depth all --symbols` to the real preset — it can drift with every batch verify green.
**Fix:** give card 11 a test-file edit adding the `TestUsageText_*` counterpart for the glyphs line, the `--view` row and the preset block.

### [BLOCKING:consistency] Card 5 prescribes a doc.go claim card 1 falsifies
**Location:** batch 1 / card 5, against card 1
**Issue:** card 5 tells the implementer to rewrite `quarry/doc.go`'s second paragraph so the surviving claim is "the answer types are the engine's, and the four delegating queries add no filtering". Verified against `quarry/doc.go:6` and `quarry/quarry.go:11-27`: every current answer type is a type *alias* for an engine type, but card 1 declares `GlyphsAnswer` as a package-owned defined struct in `quarry/view.go` — so the prescribed replacement is stale on the day it is written. The same paragraph's renderer sentence also calls the renderers "the only code it owns beyond the queries themselves", which `GlyphView` and the shadow structs falsify; card 5 names only the count in that sentence.
**Fix:** have card 5 state the replacement as "the engine's types plus this package's one projected answer type", and dispose of the "only code it owns" clause explicitly.

### [BLOCKING:scope] Repo.Glyphs's error pass-through is asserted nowhere
**Location:** batch 1 / card 4
**Issue:** card 4 requires `Glyphs` to return the zero `GlyphsAnswer` and the engine's error unchanged so `errors.Is(err, ErrTargetNotFound)` keeps working through it, but its two prescribed test cases are the deep-equal drift assertion and the `""`/`"."` target echo — neither reaches the error branch. Verified against `internal/cli/cli.go:371-428`: `runTOC` calls `repo.TOC` plus `quarry.GlyphView` directly and never calls `Repo.Glyphs`, so no CLI test covers it either; the method's error contract is untested everywhere in the plan.
**Fix:** add a case to card 4 asserting `Glyphs("no/such/dir")` on a `writeScratchTree` fixture returns the zero answer and an error satisfying `errors.Is(err, ErrTargetNotFound)`.

### [NIT:consistency] quarry/render.go's file header left stale by card 2
**Location:** batch 1 / card 2
**Issue:** `quarry/render.go:1-6` states the file "declares the JSON renderers the facade exports: RenderJSON, RenderResolveJSON, RenderExpandJSON and RenderNameJSON, the four successful envelopes, sharing one unexported encoder configuration in renderJSON". After card 2 the facade exports a fifth success renderer that also delegates to `renderJSON` but is declared in `view.go`, and no card disposes of that header — while the overview's `doc-comments-carry-the-reasoning` decision enumerates six other headers it does dispose of.
**Fix:** name `quarry/render.go`'s header in card 2 (or in the Shared Decision's list) with the one-clause correction it needs.

### [NIT:consistency] internal/cli/flags.go's file header left stale by card 9
**Location:** batch 2 / card 9, against batch 1 / card 4
**Issue:** `internal/cli/flags.go:1-4` says the file "declares request ... and parseArgs". Card 9 adds a third package-level declaration, `glyphsPreset`, and updates only `parseArgs`'s own doc comment. Card 4 treats `quarry/repo.go`'s header as stale for exactly this reason (it named Repo, Open and TOC and the card adds two more declarations), so the two cards apply different rules to the same situation.
**Fix:** add the flags.go file header to card 9's doc-comment updates.

### [NIT:consistency] the-pinned-checkout-lives-in-scratch rests on a wrong mechanism
**Location:** overview / `### Decision: the-pinned-checkout-lives-in-scratch`
**Issue:** the rationale says `$PWD` is used because the expansion must be absolute, "which `loomyardRepo`'s own `os.Stat` requires". Verified against `internal/cli/loomyard_test.go:52-63`: `os.Stat` accepts a relative path fine. The real reason absolute is required is that `go test ./internal/cli/` runs each test binary with its cwd set to the package source directory, so a relative value would resolve against `internal/cli/`. The conclusion is right and the commands are correct; the stated reason is not, and it is the sentence a fixer would read before "simplifying" the value.
**Fix:** restate the rationale as the per-package cwd of `go test`.

## Verdict

REQUEST_CHANGES
Two untested surfaces and one prescribed-stale doc claim; everything else verified sound.
MILL_REVIEW_END
