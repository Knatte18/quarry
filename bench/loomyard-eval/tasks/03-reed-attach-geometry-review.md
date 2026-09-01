# Task 03 — reintroduced missed-caller bug (code review)

Type: code review
Verb under test: `impact` (also usable: `refs`, `assert-no-callers`)
Status: **blocked** until `impact` appears in `quarry --help`

## Setup (do this once, before dispatching A/B/C)

This reconstructs a real, verified missed-caller bug from Loomyard history —
real commit `8b14f32ba34c766d06787c6be5c084368287c90d` ("reed: attach
doesn't reconcile session geometry with the terminal") renamed the free
function `attachArgv(socket, session string) []string` into a method
`Engine.AttachArgv(cols, rows int) []string` and updated its callers. The
commit touched two callers: `internal/reedcli/attach.go` and
`internal/loomcli/run.go`. This task's reviewable state keeps the first
caller updated and silently reverts the second — reproducing exactly the
class of bug `impact`/`refs` exists to catch.

```
git -C "$LADDER_LOOMYARD_REPO" worktree add /tmp/loomyard-review-03 8b14f32ba34c766d06787c6be5c084368287c90d
git -C /tmp/loomyard-review-03 checkout 8b14f32ba34c766d06787c6be5c084368287c90d~1 -- internal/loomcli/run.go
```

Point A/B/C's exploration at `/tmp/loomyard-review-03` instead of the main
Loomyard checkout for this task only. Remove the worktree when done
(`git -C "$LADDER_LOOMYARD_REPO" worktree remove /tmp/loomyard-review-03`).

Ground truth (do not reveal to A/B/C): `internal/loomcli/run.go` still calls
the deleted free function `attachArgv`, which no longer exists after the
rename — this is a real compile break, not a style nit. This is the one
thing a correct review must catch.

## `<TASK TEXT>` (identical for A, B, C)

> A colleague renamed `attachArgv(socket, session string) []string` (a free
> function that built a tmux attach argv) into `Engine.AttachArgv(cols, rows
> int) []string` (a method that also chains a layout command sized to the
> caller's terminal), and updated `internal/reedcli/attach.go` to use it.
> Review this change: does it correctly account for every existing caller of
> the old `attachArgv`, or does something still assume the old
> function/signature? Point to the exact file and line for anything you
> find, and say how you verified it — don't just assert it.
>
> The codebase to review is checked out at `/tmp/loomyard-review-03`.

## Notes for whoever scores this

Precision here matters as much as recall: does the agent's `evidence` field
show it actually traced callers (via `impact`/`refs`/grep), or did it just
guess plausibly? A correct verdict reached by luck should be flagged, not
scored the same as one backed by real verification.
