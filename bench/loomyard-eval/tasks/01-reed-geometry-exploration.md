# Task 01 — reed terminal-geometry exploration

Type: exploration
Verb under test: `toc`
Status: runnable now

## Setup (do this once, before dispatching A/B/C)

Pinned to `PINNED_SHA` from the top-level README (`975578cda8d6f3a81580bd4e73725e060211b766`),
not the live main checkout:

```
git -C "$LADDER_LOOMYARD_REPO" worktree add /tmp/loomyard-eval-01 975578cda8d6f3a81580bd4e73725e060211b766
```

`<TARGET_DIR>` for this task is `/tmp/loomyard-eval-01`. Remove the worktree
when done (`git -C "$LADDER_LOOMYARD_REPO" worktree remove /tmp/loomyard-eval-01`).

## Scope

`internal/reedengine` and `internal/reedcli` in Loomyard — a tmux-backed
terminal session manager (spawn/attach/geometry/render). Do not read outside
these two packages unless a symbol under them clearly requires it (e.g. a
type it embeds from elsewhere) — the point is to see how well each arm
surveys a ~60-file, two-package subsystem, not the whole repo.

## `<TASK TEXT>` (identical for A, B, C)

> Explain how a reed session's terminal geometry is reconciled when the
> operator's terminal window changes size or when re-attaching to an
> existing session. Your explanation must cover:
>
> 1. Where/how the current terminal size is read.
> 2. How that size reaches the session's stored or live geometry state.
> 3. Any point where a previously-known geometry is compared against a
>    freshly-read one (reconciliation), and what happens when they differ.
> 4. Which files and functions form this path, end to end.
>
> Scope your answer to `internal/reedengine` and `internal/reedcli`.

## Notes for whoever prepares C's fasit / scores this

Do not hand these to A/B/C — they are leads for verifying the fasit, not the
answer key itself:

- Files that plausibly matter based on filenames alone: `geometry.go`,
  `windowsize.go` (+ its test), `reconcile.go` (+ its test), `attach.go`,
  `headerpane.go`, and `render/` (`layout.go`, `rules.go`, `policy.go`).
- Real commit `8b14f32b` ("reed: attach doesn't reconcile session geometry
  with the terminal") touched exactly this area and is a strong signal for
  what the real mechanism looks like — read it (`git show 8b14f32b` in the
  Loomyard checkout) before finalizing what "correct" looks like, but do not
  just paste its commit message as the fasit; C should still independently
  verify the current (post-fix) code matches.
