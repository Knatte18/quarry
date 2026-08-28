{"status":"success","commit_sha":"3b883cf09c43eda3e54e8435dae3d1a0fcec32bf","session_id":"57e0f6ae-f9cd-4878-84d0-2f671474febd","cards_done":[9,10,11,12]}

All 4 of 4 cards committed this turn (cards 9, 10, 11, 12), verified honestly via `git log 2cb6c64..HEAD --oneline` against the batch's `## Cards` section — no partial completion.

Summary of work:
- `internal/quarryengine/lsp/lspclient.go`, `lspclient_test.go` — Card 9: `Client.Initialize` now takes a trailing `initOptions map[string]any`, sending `initializationOptions` iff non-nil; added fake-server assertions for both cases.
- `internal/quarryengine/daemon/ensureserver.go`, `doc.go`, and five daemon test files — Card 10: threaded `initOptions` through `finalizeConnection`, `ensureNative`, `ensureSupervised`, `EnsureServer`; `nativeArgv` gained a `private bool` param so a tagged native query spawns an unshared gopls instead of the shared `-remote=auto` daemon; doc.go updated in the three places the plan named.
- `internal/quarryengine/query/refs.go`, `symbol.go`, and three test files — Card 11: added `Options.BuildTags`, the `detectAndRender` helper (detect -> normalize -> render, raising `ErrBuildTagsUnsupported` before any connection), and threaded the rendered map through `acquireConnection` to both the daemon and legacy paths.
- `internal/quarryengine/query/buildtags_test.go` (new) — Card 12: covers the hard error across `References`/`Definition`/`Symbol`, the before-connection ordering, the empty-tag-set no-op complements, and order-independence of the defensive re-normalization.

Batch verify (`go vet -tags lsp ./...` then `go test ./internal/quarryengine/lsp/ ./internal/quarryengine/daemon/ ./internal/quarryengine/query/`) passed clean. No uncommitted tracked changes remain.

{"status":"success","commit_sha":"3b883cf09c43eda3e54e8435dae3d1a0fcec32bf","session_id":"57e0f6ae-f9cd-4878-84d0-2f671474febd","cards_done":[9,10,11,12]}
