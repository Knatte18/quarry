All 3 of 3 cards are committed (9, 10, 11), and `go test ./internal/cli/` passes. The working tree is clean.

I also changed `quarry/enclose.go`, which isn't in this batch's file lists. `golangci-lint run` flagged an S1007 finding there (a regexp that should be a raw string), and card 11 requires fixing lint findings in plan-touched files. I fixed it in the card 11 commit. The only other lint output was parse errors in the deliberately broken fixtures under `internal/engine/testdata/broken/`, which I left alone.

{"status":"success","commit_sha":"379251425389e244ee090c0e120d0a05a3ba0f88","session_id":"0c609161-d8cd-4597-b513-d779171be8cc","cards_done":[9,10,11]}
