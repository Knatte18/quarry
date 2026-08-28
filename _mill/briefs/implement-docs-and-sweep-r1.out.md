All 8 cards in the docs-and-sweep batch are complete: cards 47-53 each produced their own commit (7 commits total, verified against the log since the batch-start commit `c1900be0`), and card 54 (Commit: none) was completed by independently re-running both sweep greps and reading the three non-grep-reachable clauses — the sweep found nothing stale, so it made zero commits as its own contract specifies. `go test ./internal/quarryengine ./quarry ./internal/cli` passes with `-count=1`. `git status --porcelain --untracked-files=no` shows no dirty tracked files.

Key files touched:
- `/home/knatte/Code/quarry/wts/toc-verbs/docs/toc-docstring-association.md` (new)
- `/home/knatte/Code/quarry/wts/toc-verbs/README.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/doc.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/quarry/facade.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cli.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/seam_enforcement_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/mill-config.yaml`

`internal/quarryengine/layering_test.go` and `quarry/facade_test.go` were verified (per card 52 and the card-54 sweep) to already be correct and needed no edits.

{"status":"success","commit_sha":"ed03c589b2e4d690d0f3076beabc6270d02de5c8","session_id":"2d1fbb06-52e3-4391-98e8-fe0bae1bbdfa","cards_done":[47,48,49,50,51,52,53,54]}
