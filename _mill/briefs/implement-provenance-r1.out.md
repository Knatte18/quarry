All 3 declared cards in the batch file (Card 13, 14, 15) have matching commits in the log (f4def0e, 64db707, 6ea9b20), plus two additional commits (plan edit and go.mod chore) that were required side-effects of Card 15's test code. 3 of 3 cards committed — all complete.

{"status":"success","commit_sha":"0009f84978ffc6b8948f95590e3a933dd9c9db42","session_id":"64c56c81-a564-46a5-8927-9d793199276d","cards_done":[13,14,15]}

Files touched:
- /home/knatte/Code/quarry/wts/ladder-kickstart/bench/loomyard-eval/ladder/internal/ladder/provenance.go
- /home/knatte/Code/quarry/wts/ladder-kickstart/bench/loomyard-eval/ladder/internal/ladder/provenance_test.go
- /home/knatte/Code/quarry/wts/ladder-kickstart/_mill/plan/03-provenance.md (plan edit, adding `go.mod` to Card 15's Edits list)
- /home/knatte/Code/quarry/wts/ladder-kickstart/go.mod (mechanical `go mod tidy` promotion of `github.com/google/go-cmp` to a direct require, needed by Card 15's test)

{"status":"success","commit_sha":"0009f84978ffc6b8948f95590e3a933dd9c9db42","session_id":"64c56c81-a564-46a5-8927-9d793199276d","cards_done":[13,14,15]}