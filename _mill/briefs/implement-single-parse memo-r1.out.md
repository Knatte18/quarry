All 5 cards are done: 4 committed, plus card 1, which has `Commit: none`. The batch's `verify:` command passes on all four packages (`internal/engine`, `internal/cli`, `internal/mcpserver`, `quarry`). The final grep for `dirPackage`, `pass one`, `pass two` and `two parse passes` printed nothing.

- **Card 1:** the pinned Loomyard clone already existed at 72c23d9. The baseline run reported `ok` for all four packages and no Loomyard or TwentyGlyphs test skipped.
- **Cards 2-5:** committed and pushed.
  - Card 2 added `internal/engine/memo.go`.
  - Card 3 moved the walk consumer (`walk.go`, `toc.go`) onto the memo.
  - Card 4 moved the member consumer and Resolve onto it, renamed `parses` to `extractions`, updated the tests, and deleted `dirPackage`.
  - Card 5 updated doc comments in `walk.go`, `units.go`, `text.go` and `delta.go`.
- **Wording change:** in the `walk.go` header I changed "roughly 1.2 s" to "under a second". The 1.2 s figure no longer followed from the stated 616 ms single parse.

The working tree has no tracked changes.

{"status":"success","commit_sha":"556566b7b8f0bd8ab95f841b667929f1799d7207","session_id":"bfdf3e6a-89d3-47ef-a418-a5273f93fc7d","cards_done":[1,2,3,4,5]}
