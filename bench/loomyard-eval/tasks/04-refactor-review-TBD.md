# Task 04 — second missed-caller review (TBD)

Type: code review
Verb under test: `impact`
Status: **blocked** — both on `impact` landing and on picking/verifying a
target commit. Do not run this task until both are resolved.

## What to do before this is runnable

Same recipe as `03-reed-attach-geometry-review.md`: find a real Loomyard
commit that changes a function's name or signature and touches 2+ call
sites, build a worktree, revert exactly one of the updated call sites to its
pre-commit content, and write the resulting compile/logic break up as the
review task, with the true finding recorded as ground truth (not shown to
A/B/C).

## Candidate

`1d306ff3` ("Producer-agnostic final-summary artifact + wire Finalize")
touches `internal/landingshed/{deps,finalize,publish}.go` and
`internal/loomcli/landingdeps.go` together — shape suggests an engine/CLI
call-site pair similar to task 03, but the diff was not read in detail when
this file was drafted. Read `git show 1d306ff3` in the Loomyard checkout
first and confirm it actually has an omittable caller before committing to
it as this task's target; if it doesn't fit cleanly, pick a different commit
using the same criteria (rename or signature change, 2+ call sites, one
revertable in isolation to reproduce a real break).
