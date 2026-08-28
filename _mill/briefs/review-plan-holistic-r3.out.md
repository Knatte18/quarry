MILL_REVIEW_BEGIN
# Review: Add `impact` verb for caller-context lookup — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), accessed via the Claude Agent SDK
reviewed_file: plan/
date: 2026-08-28
```

## Findings

### [BLOCKING:design] Live-tier query form is never pinned
**Location:** batch 4 / card 17 **Issue:** the card says "drive `impact` against the fixture's `ApplyDiscount` method through the CLI seam" but never decides between a bare symbol name and a scanned `file:line:col` position; the precedent it says to "follow exactly" (`internal/cli/assertnocallers_lsp_test.go`) deliberately scans a position via `findInterfaceMethodPosition` because name resolution through `workspace/symbol` is fragile for a Go method (gopls reports it as `Invoice.ApplyDiscount`), and this is the one card whose assertions no gate in this pipeline can catch — batch 4's own scope states the `exec.LookPath("gopls")` skip fires in this worktree. **Fix:** state in card 17's `Requirements:` which query form the test uses, and if it is a position, name the scan helper the card reuses.

### [NIT:consistency] Card 5 cites a Shared Decision name that does not exist
**Location:** batch 1 / card 5 **Issue:** the cancellation paragraph cites "the `parse-loop-cancellation-and-timeout-scope` Shared Decision", but `00-overview.md` names it `parse-loop-cancellation-scope`; the cited name is `discussion.md`'s, not the plan's. **Fix:** cite `parse-loop-cancellation-scope`, matching the `## Shared Decisions` heading.

### [NIT:consistency] `--within` help string copied verbatim misstates its scope on `impact`
**Location:** batch 3 / card 10 **Issue:** the card says to copy `refsCommand`'s help strings for five flags including `--within`, whose text reads "restrict results to references whose file lies within this directory … see the interface-method conflation note above"; on `impact` the flag filters the caller list only and leaves `target`/`definition` untouched, which the card itself requires two paragraphs later. **Fix:** exempt `--within` from the verbatim copy the same way `--timeout` already is, and state the caller-only wording.

### [NIT:consistency] `impactCommand`'s file placement diverges from the discussion silently
**Location:** batch 3 / scope + card 10 **Issue:** `discussion.md`'s `cli-shape-mirrors-refs` decision places `impactCommand()` in `internal/cli/cli.go`; the plan creates `internal/cli/impact.go` instead (a defensible call, matching `toc.go`'s precedent) but records no disposition for the discussion's stated location. **Fix:** add the file-placement deviation as a batch-local decision in batch 3's `## Batch Scope`, so a later reader does not read it as drift.

### [NIT:consistency] "three-way error routing … verbatim" contradicts the helper it names
**Location:** batch 3 / card 10 **Issue:** `emitLookupResult` (internal/cli/cli.go:767) has two branches — `errors.As` on `*ErrAmbiguousSymbol`, then `output.Err` for everything else — yet card 10 asks for its "three-way error routing verbatim and in the same order, with no fourth branch" before enumerating a routing that collapses branches 2 and 3 into one. **Fix:** describe it as two branches, or drop "verbatim".

### [NIT:consistency] `repoRoot` copy hazard across a different directory depth
**Location:** batch 1 / card 6 **Issue:** the card says to resolve the repo root "the way `internal/cli/assertnocallers_lsp_test.go`'s `repoRoot` helper does"; that helper hard-codes three `filepath.Dir` calls for a file two directories below the root, while `internal/quarryengine/impact/enclosing_test.go` sits three below and needs four. **Fix:** state that the technique (`runtime.Caller(0)`, not a working directory) is what is borrowed and that the depth differs.

## Verdict

REQUEST_CHANGES
One unpinned decision in the live-tier card; the rest are cross-reference and wording nits.
MILL_REVIEW_END
