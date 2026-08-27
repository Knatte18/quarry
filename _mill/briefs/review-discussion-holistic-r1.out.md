MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude (Anthropic), Opus-class model
reviewed_file: _mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Signature rule undefined for type declarations
**Section:** Decisions §"Signature: exact source text, body excluded" + Technical context §Go
**Issue:** The rule is "first byte up to the start of its body node", but a Go `type_declaration` has no `body` field — so for the 37 multi-line `type X struct {` declarations in this repo the signature would be the entire struct body, which is exactly the token blowup the verb exists to avoid; the spike (`.scratch/tsspike/main.go:115`) hides this with an undiscussed `SplitN(sig,"\n",2)[0]` first-line hack.
**Fix:** State the signature rule for type/struct/class/interface kinds explicitly (and for a grouped `type ( … )` block: one symbol or one per `type_spec`).

### [BLOCKING:design] Batch mode cannot reuse runBatch as claimed
**Section:** Decisions §"Batch mode, consistent with the existing four verbs"
**Issue:** `runBatch` (`internal/cli/cli.go:909`) hard-codes the per-entry key as `"symbol"` (line 918) and ranks a closed vocabulary `found/not_found/ambiguous/error` (0<1<2<3); the discussion says "the machinery already exists" without deciding whether toc emits `"symbol": "<path>"`, generalises the shared helper (changing all four existing verbs' shape), or adds its own driver — nor which status a missing file, an unsupported `.ts`, or a `partial: true` parse maps to.
**Fix:** Decide the per-entry key and give an explicit toc-outcome → batchStatus/exit-rank mapping, including `partial`.

### [BLOCKING:consistency] `--lang` semantics contradict the no-resolveContext rule
**Section:** Decisions §"Language detection: by file extension" vs Technical context §Existing structure
**Issue:** `--lang` is said to work "matching the existing verbs' flag", but for those verbs the flag is a *registry key* validated inside `DetectLanguage` against the registry loaded by `resolveContext` (`internal/cli/cli.go:153`, flag help at :205) — which the same discussion says toc must not be forced through; it is therefore unstated whether toc validates `--lang`, against what vocabulary, and what `--lang go` on a `.py` file does.
**Fix:** Define toc's `--lang` value set and validation path independently of the registry, and state the behaviour on an extension/`--lang` mismatch.

### [BLOCKING:decision] Grammar-set build tag left explicitly unmeasured
**Section:** Technical context §Spike results (final paragraph)
**Issue:** "Binary-size impact of the default embedded set was not measured … should be checked during implementation" leaves `grammar_set_core` / `GOTREESITTER_GRAMMAR_SET` / `grammar_blobs_external` with no disposition, while 206 embedded grammar blobs land in every `go build -o quarry ./cmd/quarry` — a size regression that could invalidate the pure-Go-default premise after the work is done.
**Fix:** Decide now which grammar-set mode ships by default, or state an explicit size budget and the fallback if it is exceeded.

### [NIT:scope] Stale exact-count doc invariants not in the work inventory
**Section:** Scope §In
**Issue:** Scope lists only the README verb list, but `quarry/facade.go:8` asserts "exactly the 29 identifiers … no more, no less", `quarry/facade_test.go:103` asserts "the eight blank-identifier assignments below reference every delegating function", `internal/cli/cli.go:7` says "exposing four verbs", and README:3 calls quarry "an LSP-backed code intelligence tool".
**Fix:** Add these four artefacts to the inventory, including the facade_test.go signature-assertion convention for `TOCFile`/`TOCDir`.

### [NIT:design] "First paragraph" truncation undefined and untested
**Section:** Decisions §"`toc dir`: one level, all known code extensions, truncated headers"
**Issue:** No definition of a paragraph boundary in a `//`-per-line Go block or a C# `///` block (bare `//` line? blank source line?), and the Testing section names no truncation test.
**Fix:** Define the boundary per comment form and add it to the header-extraction test list.

### [NIT:design] Per-file I/O failure inside `toc dir` has no disposition
**Section:** Decisions §"Unparseable input" / §"`toc dir`: one level…"
**Issue:** "Never fail the whole call" is stated for parse errors only; an unreadable, non-UTF8, or huge file with a code extension, and an empty directory's result shape, are unaddressed.
**Fix:** Extend the partial-result rule to per-file read failures with a stated key, and state the empty-directory envelope.

## Verdict

REQUEST_CHANGES
Four decisions missing or contradicted; signature and batch rules are not implementable as written.
MILL_REVIEW_END
