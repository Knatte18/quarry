{"status":"success","commit_sha":"1a4deb7f7902962957cf7314738c57520e531979","session_id":"46bba1d5-5a59-46ce-ab70-c68ed99e74ec","cards_done":[23,24,25]}

All 3 of 3 cards in this batch are committed: card 23 (4a14c70, build-tag fixture module, already present from before this session), card 24 (1c76155, build-tag visibility live test, already present), and card 25 (1a4deb7, this session's work -- the `declarationMatchSet` package-scoped fix plus the CLI-level live interface-conflation test).

Key files touched this turn:
- `internal/quarryengine/query/verify.go` -- `declarationMatchSet` now scopes `implLocs` inclusion by directory identity with `defLocs`'s URI (via new `uriDir` helper) instead of interface/concrete `SymbolKind` classification; removed the now-dead `isInterfaceDeclaration`, `rangeContains`, `symbolKindInterface`.
- `internal/quarryengine/query/callers.go` -- removed `classifyImplementations` and the `SupportsDocumentSymbol()` gate that fed it; `callersFromClient` calls `declarationMatchSet(defLocs, implLocs)` directly.
- `internal/quarryengine/query/verify_test.go`, `internal/quarryengine/query/callers_test.go` -- rewritten to assert the package-scoped directory semantics instead of the removed documentSymbol classification.
- `docs/implementation-widening-spike.md` -- Counts section recomputed by hand against a live pinned-gopls re-run: `references-verified` 7->3, `callers-verified` 6->2.
- `internal/cli/assertnocallers_lsp_test.go` -- new live-gopls test at the CLI level pinning the fix and the `--no-verify` escape hatch; updated its expected count from 6 to 2 to match the corrected figure.
- `_mill/plan/06-live-tier-tests.md` -- added `callers.go`/`callers_test.go` to card 25's `Edits:` list before touching them (protocol step 2), committed separately as `6685600`.

Batch verify (`PATH="$HOME/.cache/quarry/tools/go/v0.23.0:$PATH" go test -tags lsp ./internal/quarryengine/query/ ./internal/cli/`) passes, along with module-wide `go vet ./...`, `go vet -tags lsp ./...`, and `go test ./internal/quarryengine/`. No uncommitted tracked changes remain, and no stray daemon processes from the pinned toolchain were left running.

{"status":"success","commit_sha":"1a4deb7f7902962957cf7314738c57520e531979","session_id":"46bba1d5-5a59-46ce-ab70-c68ed99e74ec","cards_done":[23,24,25]}
