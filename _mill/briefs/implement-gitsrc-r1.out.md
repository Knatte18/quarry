All 6 cards (25-30) are committed - matching each card's `Commit:` message, plus one extra fix commit (6d8c8c6) discovered/corrected mid-batch for card 27's ChangedPaths to satisfy card 30's unmerged-letter test. 6 of 6 cards committed, all complete.

{"status":"success","commit_sha":"2fc79dc80c2575229547d6de36ff0948c2ae9cd9","session_id":"7f30f82e-4037-479c-bc41-1b60f6b36177","cards_done":[25,26,27,28,29,30]}

Summary of work: created `internal/gitsrc` (doc.go, errors.go, gitsrc.go, fixture_test.go, gitsrc_test.go) at `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/gitsrc/`, implementing `Open`, `VerifyRevision`, `ChangedPaths`, `UntrackedPaths`, `ReadBlob`, `DirFilesAtRevision`, `DirFilesInWorkingTree`, and the error-identity sentinels/typed errors per the batch plan. All 6 cards (25-30) committed; `go test ./internal/gitsrc/` passes. One additional fix commit (6d8c8c6) was needed mid-batch: empirical testing showed a plain rev-vs-worktree `git diff` structurally cannot report the "U" (unmerged) status letter since it bypasses the index, so `ChangedPaths` now runs a supplementary index-aware query to surface conflicted paths correctly - this was necessary to satisfy card 30's own required test coverage.

{"status":"success","commit_sha":"2fc79dc80c2575229547d6de36ff0948c2ae9cd9","session_id":"7f30f82e-4037-479c-bc41-1b60f6b36177","cards_done":[25,26,27,28,29,30]}
