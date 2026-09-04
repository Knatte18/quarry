Finding processed through the decision tree:

**Finding:** [NIT:decision] go.sum retains checksums for unused tree-sitter grammars (go.sum:11-34)

- **VERIFY:** Accurate — go.sum does list h1: entries for tree-sitter-c, -cpp, -embedded-template, -html, -java, -javascript, -json, -php, -python, -ruby, -rust, none of which go.mod requires.
- **HARM CHECK / resolution:** The finding itself specifies the correct fix path is conditional: confirm whether `go mod tidy` retains these entries on its own before touching anything. I ran `CGO_ENABLED=1 go mod tidy` in the worktree and diffed `go.sum`/`go.mod` afterward — zero changes (`git diff go.sum go.mod` empty, `git status --porcelain` empty). This confirms the entries are retained by `go mod tidy` itself (module-graph pruning that needs indirect dependents' go.mod files to fully prune), not leftover cruft from before tidying.
- **Action:** No code change required — this is the legitimate no-op branch the finding itself anticipated ("Confirm these are retained by `go mod tidy` itself ... rather than leftover cruft").

All `verify:` commands from the batch plan files ran clean (both the `build && test` variant used by batches 1-5 and the `build && vet && test` variant used by the overview and batch 6):
- `CGO_ENABLED=1 go build ./...` — clean
- `CGO_ENABLED=1 go vet ./...` — clean
- `CGO_ENABLED=1 go test ./internal/...` — all packages `ok`

`git status --porcelain --untracked-files=no` shows no tracked modifications, and HEAD equals the recorded baseline (`6dc1e792e0ff2f350d156c48fd5de0f1a1b4039d`) — consistent with the sole finding being a legitimate no-op requiring no code change, per the brief's exception for that case.

Relevant files inspected: `/home/knatte/Code/quarry/wts/engine-core/go.mod`, `/home/knatte/Code/quarry/wts/engine-core/go.sum`, `/home/knatte/Code/quarry/wts/engine-core/_mill/reviews/20260904-044603-code-review-r1.md`, `/home/knatte/Code/quarry/wts/engine-core/_mill/plan/00-overview.md`.

{"status":"success","commit_sha":"6dc1e792e0ff2f350d156c48fd5de0f1a1b4039d","session_id":"4f6926c9-f707-4f8f-8cec-ec3ad995730f"}

{"status":"success","commit_sha":"6dc1e792e0ff2f350d156c48fd5de0f1a1b4039d","session_id":"4f6926c9-f707-4f8f-8cec-ec3ad995730f"}
