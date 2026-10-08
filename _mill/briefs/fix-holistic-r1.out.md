All four findings are fixed in one commit, and the verify commands from all three batch plans pass. I ran them as one `go test` call over the engine, repopath, quarry, mcpserver and cli packages. The commit is pushed, and there are no uncommitted tracked changes.

- **`quarry/doc.go`:** I rewrote the passages the review flagged. The query and renderer counts and the `DeltaGit`-only wording are gone, and `Enclose` now appears. I added the file to the `Edits:` list of card 7 in `_mill/plan/02-facade-enclose.md` and to the file list in `_mill/plan/00-overview.md`.
- **`internal/cli/doc.go`:** "not a sixth pipeline" is now "has no pipeline of its own".
- **`internal/cli/flags_test.go`:** I removed the counts from both comments and said `--text` is rejected for enclose. I also added an `enclose` row to the verb-gate table, which the review noted was missing.
- **`internal/cli/flags.go` and `quarry/repo.go`:** I reflowed the `parseArgs` and `Open` doc comments.

{"status":"success","commit_sha":"558b0ee9858a4b915676da4af2803e8e68d71d42","session_id":"d9c1de7b-d798-4ac5-978a-16ba4381b50d"}
