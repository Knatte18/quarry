All packages build, vet clean, and pass tests. Both files staged as required. No content was discarded — everything from both sides was preserved and combined (this branch's resolve/expand verb pipelines and the `repoRelPath`/`repoRelTarget` arithmetic-vs-strict split, plus the parent's `internal/repopath` package extraction/export and `rootUsageMessage` root-error formatting). I did extend the resolution slightly beyond the literal conflict-marker regions in `cli.go` — two call sites (`runResolve`'s `repoRelPath` call, and stale doc-comment references to the old local function names) were outside the markers but would have silently broken compilation or become factually stale once the two real conflicts were resolved, so I updated them to keep the merge internally consistent; this is exactly the kind of contradiction the brief's step 8 self-check calls out.

Key files:
- `/home/knatte/Code/quarry/wts/facade-cli-resolve-expand/internal/cli/cli.go`
- `/home/knatte/Code/quarry/wts/facade-cli-resolve-expand/internal/repopath/target.go`

{"status":"success"}
