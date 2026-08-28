MILL_REVIEW_BEGIN
# Review: Add `impact` verb for caller-context lookup — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), best-effort self-assessment
reviewed_file: plan/
date: 2026-08-28
```

## Findings

### [BLOCKING:scope] cli.go batch status-vocabulary doc site missed
**Location:** batch 3 / card 11
**Issue:** `internal/cli/cli.go` lines 30-34 state the batch status vocabulary as `"ambiguous" (refs/definition only; toc never produces it...)`; `classifyImpactError` makes `impact` a third producer, but card 11 names only the verb enumeration and the identity-key sentence, then says "Change nothing else in this file", leaving a false statement behind.
**Fix:** name that parenthetical as a fourth edit in card 11, under the same `doc-site-ownership-by-touching-batch` rule that already assigns cli.go to batch 3.

### [BLOCKING:consistency] Card 15's minPackageDirs comment instruction is self-contradictory
**Location:** batch 4 / card 15
**Issue:** the card says keep the "deliberately kept one below the real count" reasoning intact *and* correct the two counts it cites; after batch 1 the two-tree walk visits 10 directories against a floor of 8, so "one below" and "(8, not 9)→corrected" cannot both hold. The card's own premise ("the walk actually visits 9") is already stale relative to its own batch order.
**Fix:** state one disposition — either restate the slack as "deliberately kept below the real count" without a count, or raise the constant — rather than leaving the implementer to reconcile it.

### [BLOCKING:design] The live-tier test is never executed by any gate
**Location:** batch 4 / card 17 + batch 4 `verify:`
**Issue:** card 17 is named across batches 1, 3, and 4 as the sole proof of the brief's central claim (docstring-inclusive `enclosing_range`), yet batch 4's `verify:` only runs `go vet -tags lsp`, and `pipeline.done_gate` is untagged `go test ./...`, so the assertion never runs anywhere in the pipeline.
**Fix:** add a tagged test invocation (e.g. `go test -tags lsp ./internal/cli/`) to batch 4's `verify:` — the `exec.LookPath("gopls")` skip already makes it green on a machine without gopls.

### [BLOCKING:scope] Requirements name identifiers from files absent from `Context:`
**Location:** batch 3 / card 10; batch 1 / cards 1 and 5
**Issue:** card 10 prescribes the `CwdFrom` / `SetExit` sequence, but `CwdFrom` lives in `internal/cli/cwdcontext.go` and `SetExit` in `internal/cli/exec.go`, neither in its `Context:`; card 5 names `internal/cli`'s `filterUnexpectedCallers` with cli.go absent; card 1 names `toc.Kind` and `toc.Symbol.Start` with `toc/types.go` absent.
**Fix:** add `internal/cli/cwdcontext.go` and `internal/cli/exec.go` to card 10's `Context:`, `internal/cli/cli.go` to card 5's, and `internal/quarryengine/toc/types.go` to card 1's.

### [NIT:consistency] Card 14 announces four edits and prescribes five
**Location:** batch 4 / card 14
**Issue:** the card opens "Four distinct edits, not two", enumerates four, then adds a fifth (`impact` into the opening paragraph's list of questions the engine answers) — a hand-counted claim of the kind this plan corrects elsewhere.
**Fix:** restate the count as five, or fold the opening-paragraph edit into the enumeration.

### [NIT:consistency] Reused `structToFields` emits a "toc:"-prefixed error for impact
**Location:** batch 3 / card 10
**Issue:** `internal/cli/toc.go`'s `structToFields` wraps failures as `toc: marshal result: %w` / `toc: unmarshal result: %w`; card 10 routes an impact marshal failure straight into `output.Err`, so the verb's error envelope would name `toc`.
**Fix:** state the disposition — accept the prefix explicitly, or have `emitImpactResult`/`classifyImpactError` re-word the message before emitting.

### [NIT:consistency] `definition` flattens the range that `callers` nests
**Location:** shared decision `json-key-disposition`; batch 1 / card 2
**Issue:** the same enclosing-range concept is emitted as flat `start_line`/`sigend_line`/`end_line` on `definition` but nested under `enclosing_range` on a caller entry, so a consumer parses two shapes for one idea.
**Fix:** record why the shapes differ in the decision's rationale, or reuse `*Range` on `Definition` as well.

### [NIT:consistency] Doc-audit sweep assigned only in Batch Tests, not in the owning card
**Location:** batch 4 / Batch Tests vs card 16
**Issue:** Batch Tests says the sweep is "performed by the implementer as part of card 16 and reported in its commit body", but card 16's `Requirements:` never mentions it; the same section also attributes card 16 (a README edit) to "the CLI package".
**Fix:** move the sweep instruction into card 16's `Requirements:` and correct the package attribution in Batch Tests.

## Verdict

REQUEST_CHANGES
Three doc/verify gaps and one context-completeness gap; the plan's structure is otherwise sound.
MILL_REVIEW_END
