All 8 cards (3 through 10) declared in this batch are committed, matching their exact `Commit:` messages in the log. Two additional `plan:` commits were made mid-batch to extend the plan's Edits: list before touching `lsp/lspclient.go` outside its originally-declared scope (exporting `Lang` and adding `Closed()` on `lsp.Client`, both required for cross-package compilation once `daemon` and `lsp` split apart) — these are plan-extension commits, not card commits, and don't count against the 8-card total. The working tree is clean of tracked modifications, and both verify commands (`go test ./...` and `go test -tags lsp -run "^$" ./...`) pass.

{"status":"success","commit_sha":"84682a68dfd5637d3c65d9c0f5a9295e4d4449dd","session_id":"bcdf2ffd-245d-4fb9-a69b-d01a9333ed00","cards_done":[3,4,5,6,7,8,9,10]}

Summary of work: all 8 cards of batch `02-engine-repackage.md` are complete — 8 of 8 cards committed, verify green. Key files touched:
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/{errors.go,position.go,log.go,doc.go}` — root leaf package
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/lsp/{lspclient.go,wire.go,lspclient_test.go,lspclient_guard_test.go,position_test.go}`
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/registry/{registry.go,load.go,detect.go,*_test.go}`
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/daemon/{ensureserver.go,toolchain.go,daemonstate.go,probe.go,*_test.go}` and `daemon/daemontest/daemontest.go`
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/query/{refs.go,definition.go,symbol.go,*_test.go}`
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/quarry/facade.go` (new thin facade), `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/seam_enforcement_test.go` (widened guard)
- Plan extension: `/home/knatte/Code/quarry/wts/quarry-thin-facade/_mill/plan/02-engine-repackage.md` (documented exporting `lsp.Client.Lang` and `lsp.Client.Closed()`, both required for compilation once `daemon` split from `lsp`).
