All verify commands pass, no uncommitted tracked changes, and HEAD (392c54f2707e36a3157e094010b1c1beab89742e) differs from the recorded baseline (f62cac3cae1c80dce026123f338f43554e94df70).

{"status":"success","commit_sha":"392c54f2707e36a3157e094010b1c1beab89742e","session_id":"86b7ade0-00bd-4ec2-9160-445b848363d9"}

Summary of work:
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve_test.go`: removed the duplicated `openQuarryRoot` helper (identical to `walk_test.go`'s `openModuleRepo`), swept all 16 call sites to use `openModuleRepo`, dropped the now-unused `runtime` import.
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/expand_test.go`: swept 7 more `openQuarryRoot` call sites (same duplication pattern reaching into this file) to `openModuleRepo`, and corrected the stale header-comment reference to the removed helper.
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve.go`: fixed the NIT - `symbolsOfUnit`'s doc comment said grouping "is realised in unitMemo, below" when `unitMemo` is declared earlier in the file; changed to "above".

Both findings from `_mill/reviews/20260904-065319-code-review-r1.md` were FIX'd (no push-back needed - both were factually accurate with no harm from fixing). Commit `392c54f2707e36a3157e094010b1c1beab89742e` pushed to `origin/resolve-expand`. All five batch `verify:` commands (`go build ./...`, `go vet ./internal/engine/`, `go test ./internal/engine/` x4) passed.
