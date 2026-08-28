All 5 cards (17, 18, 19, 20, 21) committed across 3 commits (18+19+20 combined into one, as required by the Strategy interface's compile-time constraints). All cards committed, verify passed.

{"status":"success","commit_sha":"97aa6711114b5c31bf83e5d0c9557cc12b5238bc","session_id":"bd2e15ea-3887-4baa-983e-da32e4b9d7c9","cards_done":[17,18,19,20,21]}

Summary: 5 of 5 cards committed this turn.

- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/nodes.go` (card 17) — shared tree-sitter node helpers: `NodeText`, `Line`, `SignatureCut`, `SigEnd`, `CommentBlockAbove`, `LeadingBlocks`, `CommentBlock`.
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/golang.go` (cards 18, 19, 20 — combined into one commit since `Register(goStrategy{})` requires the full `Strategy` interface to compile, so no subset of these three cards' methods could be committed alone without breaking the build) — Go function/method/type extraction plus `Package`, `Header`, `Generated`, `TestFile`.
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/golang_test.go` (card 21) — table-driven tests covering all required cases (docstring association, blank-line adjacency, receiver owner stripping, function-body exclusion, all `type_declaration` shapes including grouped/single-spec-grouped, `SigEnd` derivation, `Package`, `Header`, `Generated`, `TestFile`, and lossy partial-parse recovery).

Verified the tree-sitter-go v0.25.0 grammar shapes (type_alias vs type_spec, grouped-declaration `(` marker, interface body's bodyless `{`) by parsing real fixtures with a throwaway dump program under `.scratch/` (removed before finishing, never committed). `go test ./internal/quarryengine/toc` passes (batch verify command), `go vet` is clean, and `gofmt -l` reports no issues on any new file.
