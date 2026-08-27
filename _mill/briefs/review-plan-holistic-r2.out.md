MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Anthropic Claude, Opus-class; environment reports claude-opus-5 (unverifiable from inside)
reviewed_file: plan/
date: 2026-08-27
```

## Findings

### [BLOCKING:scope] Card 16 ConnKind test violates the query layering row
**Location:** batch 4 / card 16, last bullet
**Issue:** Asserting `teardownConnection(client, ConnKindNative/ConnKindLegacy/ConnKindSupervised, true)` from a `query` `_test.go` needs an import of `internal/quarryengine/daemon`, which `layering_test.go`'s `{pkgDir:"query", isTestRow:true}` row forbids (allowed set is root, registry, lsp, daemontest), and `daemontest.go` re-exports no `ConnKind` constant — so the card as written breaks the `layering-is-non-negotiable` Shared Decision it inherits.
**Fix:** State the disposition — either add a `daemontest` `ConnKind` re-export (with its own `Creates`/`Edits` and a `Context` entry) or drop the teardown assertion from `query` and site it where `daemon` is importable.

### [BLOCKING:scope] Layering/seam guards never re-run after batch 2
**Location:** 00-overview.md `verify:` + batches 3–6 `verify:`
**Issue:** `go test ./internal/quarryengine/` (the only place `layering_test.go` and `seam_enforcement_test.go` run) appears solely in batch 2's verify; the module-wide `verify:` is `go vet` only, which type-checks but cannot fail a custom guard test. Batches 3–6 add new files to `lsp`, `daemon`, `query`, `quarry` and `cli` with no guard re-run, so the finding above (and any similar import drift) would land green.
**Fix:** Add `./internal/quarryengine/` to the verify command of every batch that adds or edits a file under `internal/quarryengine/`, or put it in the overview `verify:`.

### [BLOCKING:design] Live-tier cards leak detached supervised gopls daemons
**Location:** batch 6 / cards 24 and 25
**Issue:** Go's entry has `HasNativeDaemon: true`, so both tests route through `ensureSupervised`, which spawns a detached daemon with `daemonIdleTimeout = 10 * time.Minute` that `teardownConnection`'s `ConnKindSupervised` branch deliberately never kills. Card 24 spawns two (two distinct `t.TempDir()` state dirs) and card 25 one, and neither card prescribes the repo's established reap — `refs_integration_test.go` pairs its `t.TempDir()` with `daemontest.StateFile` + `t.Cleanup(daemontest.KillRecordedDaemon)`. Card 24's `Context` omits `daemontest` entirely; card 25 lives in `internal/cli`, which the `layering-is-non-negotiable` decision confines to `quarry/facade.go`, so it has no sanctioned reap path at all.
**Fix:** Give card 24 the `daemontest` state-file + cleanup pairing (adding `daemontest.go` to its `Context`), and state card 25's disposition explicitly — reap mechanism, or a recorded decision that the CLI live test may not spawn a supervised daemon.

### [NIT:consistency] Shared Decision `Applies to:` omits the batch that implements it
**Location:** 00-overview.md `rendered-options-non-nil-means-tagged`
**Issue:** `Applies to:` lists initialization-options-plumbing, callers-verification-entry-point and cli-surface, but the `(nil, nil)`-for-empty contract is *implemented* by batch 2 card 8, which restates it as a batch-local decision instead.
**Fix:** Add `registry-build-tag-template` to that decision's `Applies to:` line.

### [NIT:consistency] Card 29 contradicts card-scoped-commits
**Location:** batch 7 / card 29, `Commit: none`
**Issue:** The `card-scoped-commits` Shared Decision says every card produces its own commit with the message its `Commit:` field names; card 29 files two GitHub issues and names none.
**Fix:** Note the file-less exception in the Shared Decision itself so the card and the decision stop disagreeing.

### [NIT:scope] Context omits files whose identifiers the Requirements name
**Location:** batch 4 / card 13; batch 5 / card 20
**Issue:** Card 13's `Requirements` name `quarryengine.ErrServerTimeoutSentinel` (`errors.go`) and `lsp.Client`/`lsp.Position` with neither file in `Context:`; card 20 names `opts.SkipVerification`, a field card 15 adds to `Options` in `internal/quarryengine/query/refs.go`, which is absent from its `Context:` (only `callers.go` is listed).
**Fix:** Add `internal/quarryengine/errors.go` and `internal/quarryengine/lsp/lspclient.go` to card 13's `Context:`, and `internal/quarryengine/query/refs.go` to card 20's.

## Verdict

REQUEST_CHANGES
Two layering/verify gaps and an unreaped live-tier daemon need disposition before implementation.
MILL_REVIEW_END
