MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5)
reviewed_file: plan/
date: 2026-08-27
```

## Findings

### [BLOCKING:scope] Card 10 misses 12 ensureSupervised/ensureNative call sites
**Location:** batch 3 / card 10
**Issue:** The card changes `ensureSupervised` and `ensureNative` signatures but its `Edits:` omits `daemon/supervised_test.go` (3 `ensureSupervised(` sites, untagged — breaks `go test ./internal/quarryengine/daemon/`), `daemon/supervised_lsp_test.go` (2 sites) and `daemon/supervised_integration_test.go` (3 sites, both `//go:build lsp` — break the batch's own leading `go vet -tags lsp ./...`); it also names only the two `EnsureServer` sites in `ensureserver_integration_test.go` while that file additionally calls `ensureNative(` at lines 72 and 111.
**Fix:** Add the three supervised test files to `Edits:` and require updating every `ensureSupervised`/`ensureNative` call site, not just `EnsureServer`/`finalizeConnection`/`nativeArgv`.

### [BLOCKING:design] documentSymbol classification phase has no deadline
**Location:** batch 4 / card 15
**Issue:** The card brackets the definition/implementation/references phases and the per-reference loop with `context.WithTimeout(ctx, timeout)`, but the directional-mode "one `textDocument/documentSymbol` per distinct file URI" calls are given no deadline, while the card simultaneously lists "a `documentSymbol` call errored" as a skip trigger and elsewhere requires the timed-out flag be set on server timeouts — so it both assumes and omits a bound.
**Fix:** State the deadline the documentSymbol classification phase runs under (its own `context.WithTimeout(ctx, timeout)`, per the invariant `lspclient.go`'s package doc and `lookup`'s doc both already assert).

### [BLOCKING:consistency] Card 17 leaves three stale count comments in facade_test.go
**Location:** batch 4 / card 17
**Issue:** The card carefully retargets `facade.go`'s "exactly 29 identifiers" sentence, but `quarry/facade_test.go` carries three more checkable counts it never mentions — "each of the fourteen aliased types" (line 56), "The eight blank-identifier assignments below" (line 103), "each of the seven re-exported sentinel error values" (line 117) — all of which the card's own additions falsify (fifteen, ten, eight).
**Fix:** Require the card to update those three counts in the same edit, on the same "a checkable count left stale is worse than none" rationale it already states.

### [BLOCKING:scope] Context omits registry.go (card 12) and paths.go (card 24)
**Location:** batch 3 / card 12; batch 6 / card 24
**Issue:** Card 12's `Requirements:` names `registry.Registry`, the synthetic entry's `Command`, and an entry "whose `InitializationOptions` is nil` — all declared in `internal/quarryengine/registry/registry.go`, which is in neither its `Context:` nor `Edits:`. Card 24 names `resolveStateDir` in `internal/cli/paths.go`, likewise absent from its `Context:`.
**Fix:** Add `internal/quarryengine/registry/registry.go` to card 12's `Context:` and `internal/cli/paths.go` to card 24's.

### [NIT:consistency] Card 18 justifies paths.go siting by an import it does not add
**Location:** batch 5 / card 18
**Issue:** The batch-local decision sites `resolveBuildTags` in `internal/cli/paths.go` because "that file already imports `os`", but the function's body calls `quarry.NormalizeBuildTags`, so the card silently requires a new `github.com/Knatte18/quarry/quarry` import into a file that currently imports only stdlib.
**Fix:** State the new import explicitly in the card's `Requirements:`.

## Verdict

REQUEST_CHANGES
Batch 3 will not compile; three smaller scope and consistency gaps also need closing.
MILL_REVIEW_END
