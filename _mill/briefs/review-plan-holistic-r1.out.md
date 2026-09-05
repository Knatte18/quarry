MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5, high reasoning effort)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:scope] `quarry/render_test.go` never follows the Listing rename
**Location:** batch 2 / cards 10, 17 (and overview `## All Files Touched`)
**Issue:** Verified in `quarry/render_test.go`: line 190 builds `ResolveResult{Target: "pkg", Status: StatusFound, Dir: &DirAnswer{Dir: "pkg"}}`, line 219 builds the same field in `TestRenderResolveJSON_KeyOrder`, and line 226 pins the JSON key list ending `` `"dir"` ``; the file appears in no card's `Edits:` and in no `## All Files Touched` entry, so card 10's field rename breaks the `quarry` test binary's compile and batch 2's own `verify: go test ./quarry/...` fails at build time.
**Fix:** Add `quarry/render_test.go` to card 17's `Edits:` (rename both `Dir:` composite-literal keys to `Listing:` and the pinned `"dir"` key to `"listing"`) and to the overview's file list.

### [BLOCKING:scope] `TestRepoResolve_PathTarget` asserts the retired path contract
**Location:** batch 2 / cards 12, 17
**Issue:** `quarry/repo_test.go`'s `TestRepoResolve_PathTarget` (lines 200-221) calls `r.Resolve([]string{"sub"})` and asserts `StatusFound` plus a non-nil `Dir`; card 12 makes a bare path a `no_separator` rejection with an empty `Status`, yet card 17 instructs "change no assertion there beyond the field name", so batch 2's `verify: go test ./quarry/...` goes red — and the plan's bounded-red-window decision is scoped to `internal/cli` only, not `quarry`.
**Fix:** Extend card 17 to retarget that test's argument to the self glyph `"sub#"` and rename the test off "PathTarget", in the same batch that removes path targets.

### [BLOCKING:design] `codeForExpandError` maps `SelfGlyphError` to exit 3, not exit 1
**Location:** batch 4 / card 27 (premise also stated in batch 3 / card 19)
**Issue:** Card 27 asserts `codeForExpandError` "already returns the negative exit for any error other than the internal ones, so confirm rather than assume it needs no edit"; verified against `codeForExpandError` in `internal/cli/cli.go`, it returns `exitNegative` only for `*quarry.NotATypeError` and `*glyph.ParseError` and `exitInternal` for everything else, so a `*quarry.SelfGlyphError` exits 3 — contradicting card 19 ("must map it to the negative exit") and card 31's required "expand of a self glyph: exit 1".
**Fix:** Make card 27 require an explicit `*quarry.SelfGlyphError` arm in `codeForExpandError` returning `exitNegative`, and drop the false "needs no edit" premise.

### [BLOCKING:design] No module-wide compile gate enforces the green-build decision
**Location:** overview `verify: null` + Shared Decision "the module-wide compile stays green at every batch boundary" / batch 2 card 18
**Issue:** Card 18 exists solely to keep `internal/cli`'s test binary compiling after card 10's rename, but batch 2's `verify:` is scoped to `./internal/engine/... ./quarry/...` and the overview's module-wide `verify:` is `null`, so nothing in the plan ever compiles `internal/cli` at that boundary; the decision's own acceptance condition is unchecked until batch 4.
**Fix:** Set the overview's `verify:` to a cheap compile-only command such as `go build ./...` (compile/smoke scope, not a suite run), so the stated per-boundary invariant is actually gated.

### [NIT:scope] Card 27 names `*quarry.SelfGlyphError` without `quarry/quarry.go` in `Context:`
**Location:** batch 4 / card 27
**Issue:** The card requires `errors.As` against `*quarry.SelfGlyphError`, whose alias card 21 declares in `quarry/quarry.go`, but that file is in neither `Context:` nor `Edits:` — unlike cards 26 and 29, which both list it.
**Fix:** Add `quarry/quarry.go` to card 27's `Context:`.

### [NIT:consistency] The new expand parse message leaks the `glyph:` prefix
**Location:** batch 4 / cards 27, 28
**Issue:** Card 27 sets the message to `"expand: " + parseErr.Error()`, which renders as `expand: glyph: parse "x" as go: ...`, while the same card spells the `SelfGlyphError` branch from struct fields expressly so a package-name prefix "never leaks"; `internal/cli/cli.go`'s closing doc paragraph on exit-1 message composition ("quarry spells them", no wrapped chain) is not among card 28's seven named rewrites.
**Fix:** State on card 27 whether the `glyph:` prefix is intended, and add that closing paragraph to card 28's list if it is.

## Verdict

REQUEST_CHANGES
Two batch-2 files break its own verify; expand's self-glyph exit code is wrong.
MILL_REVIEW_END
