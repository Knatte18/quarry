MILL_REVIEW_BEGIN
# Review: Add `impact` verb for caller-context lookup

```yaml
duration_s: 317.0
verdict: APPROVE
reviewer_model: opus
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: /home/knatte/Code/quarry/wts/impact-verb/_mill/discussion.md
date: 2026-08-28
```

## Findings

### [NIT:consistency] `definition.error` rule contradicts the caller rule
**Demoted-from:** BLOCKING
**Section:** `### resolved-symbol-definition-range` vs `### no-enclosing-declaration-is-not-an-error` + `### json-key-set`
**Issue:** The definition decision says it uses "the **same** degradation rules used for every caller entry" and then, in the same sentence, "an `error` string carries the reason when one did not [resolve]" — but the caller rules have *two* non-resolving outcomes, and the file-scope one is explicitly *not* an error. So for a target whose declaration line has no enclosing toc symbol though the file parsed fine (a package-level `var`/`const`, a struct field — toc's `Kind` vocabulary is function/method/type only, verified in `internal/quarryengine/toc/types.go:12-22`), it is undefined whether `definition.error` is emitted; the "Precision on when this happens" paragraph covers only the empty-declaration-set case, and the Testing section lists only the no-toc-strategy degraded shape.
**Fix:** State the definition-side disposition for all three outcomes separately — resolved / parsed-but-no-enclosing-symbol (ranges omitted, `error` absent, `target` omitted) / parse-or-language failure (`error` present) — and add the middle shape to the assembly-step test list.

### [NIT:scope] Doc-update inventory misattributes and under-covers
**Section:** Scope "In" (doc updates) + Technical context (gotchas)
**Issue:** Technical context says the phrase "seven-package DAG" appears "exactly once in `doc.go` ... and once more in `quarry/facade.go`", while Scope claims a third in `seam_enforcement_test.go`'s `minPackageDirs` comment; the third occurrence is real but sits at `seam_enforcement_test.go:10` (header), and `TestEngineSeamInvariant_BannedImports`' doc comment (`:26-28`) carries no enumeration to update. Also unlisted: `doc.go:41-46` enumerates the package set a second time outside the "package-DAG section", and `README.md:17` says "All four verbs accept `--build-tags`" while listing six.
**Fix:** Correct the two attributions and add `doc.go:41-46` and `README.md:17` to the inventory.

### [NIT:scope] Batch status classifier omitted from the reuse ledger
**Section:** Technical context ("Existing primitives to reuse" / "Deliberately not on that reuse list")
**Issue:** The ledger lists `runBatch`, `batchStatus`/`statusRank`, `emitAmbiguousOrError` as reusable and calls out `filterWithin`/`filterUnexpectedCallers` as type-incompatible, but never mentions `classifyLookupError`/`emitLookupResult` (`internal/cli/cli.go:766,900`), which are equally typed to `[]quarry.Reference` and equally uncallable on `impact`'s result — yet "add nothing parallel to them" is the stated rule.
**Fix:** Add them to the "deliberately not on that reuse list" bullet and state impact's own error→status mapping (ambiguous / `ErrSymbolNotFoundSentinel` / error).

### [NIT:decision] New facade type names unspecified
**Section:** Scope "In" (facade re-exports) + `### impact-lives-in-its-own-engine-package`
**Issue:** Only the engine entry point is named (`Impact(...) (Result, error)`); the sub-result types and their `quarry/` alias identifiers ("type aliases for the new result types") are never named, yet extending `quarry/facade_test.go`'s enumerated `var` block and blank-identifier func-type assignment is in scope and needs exact identifiers, against a repo convention that prefixes (`TOCSymbol`, `TOCFileResult`).
**Fix:** Name the exported result types and their facade aliases explicitly.

### [NIT:design] Tree-sitter phase has no stated bound or cancellation
**Section:** `### per-call-parse-cache` + `### cli-shape-mirrors-refs` (flag parity)
**Issue:** `--timeout` is documented on `refs` as the per-LSP-request-phase deadline, and `toc.TOCFile` takes no `ctx` (`internal/quarryengine/toc/toc.go:69`), so for a high-fan-in symbol — the verb's stated target case — the per-caller-file parse loop is outside every deadline and its `ctx` honouring is unstated.
**Fix:** State whether the parse loop checks `ctx.Done()` between files and that `--timeout` deliberately does not cover it.

## Verdict

APPROVE
One contradictory degradation rule leaves the `definition` shape undefined for non-function targets.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
