# Scorecard — Task 01: reed terminal-geometry exploration

Date: 2026-08-28
Target: `internal/reedengine` + `internal/reedcli`, Loomyard pinned to
`975578cda8d6f3a81580bd4e73725e060211b766` (worktree `/tmp/loomyard-eval-01`,
removed after this run).

Scored by the orchestrator (this agent), reading `a.json`/`b.json` against
`c.json`, cross-checked against `git show 8b14f32b` (the real historical fix
for this exact mechanism) before scoring.

## Fasit sanity check

C's answer matches the real fix commit `8b14f32ba34c766` closely: the
told-box (`AttachArgv`, client's own `term.GetSize` reading) vs. live-box
(`liveBoxLocked`, `display-message` query) split, the two geometry option
pins with effective-value readback ("reconciliation" in the strict sense —
reed compares what it *pinned* against what tmux *actually honored*, not an
old-vs-new geometry diff), the literal-`;`-chained `select-layout`, and the
absence of any persisted last-known geometry are all present and correct in
C's summary and symbol list. C is a trustworthy fasit for this task.

## relevant_files

| | C (total 13) | A matched | A recall | A precision | B matched | B recall | B precision |
|---|---|---|---|---|---|---|---|
| files | 13 | 8/13 | 0.62 | 8/9 = 0.89 | 10/13 | 0.77 | 10/10 = 1.00 |

- A's only unmatched entry: `internal/reedengine/reconcile.go` — not
  hallucinated, just tangential (dead-pane reconciliation, not geometry
  reconciliation); a defensible over-inclusion, not a wrong answer.
- A's misses vs. C: `lifecycle.go`, `overlay.go`,
  `attachgeometry_integration_test.go`, `internal/reedcli/resume.go`,
  `internal/reedcli/up.go`.
- B's misses vs. C: `overlay.go`, `internal/reedcli/resume.go`,
  `internal/reedcli/up.go`. Every file B listed was corroborated by C.

## key_symbols

| | C (total 11 concepts) | A matched | A recall | A precision | B matched | B recall | B precision |
|---|---|---|---|---|---|---|---|
| symbols | 11 | 9/11 | 0.82 | 9/10 = 0.90 | 10/11 | 0.91 | 11/11 = 1.00 |

- A missed `parseWindowSize` and `errAttachChainSuppressed` as named symbols;
  its one unmatched entry (`Geometry` struct) is a reasonable inclusion, not
  wrong, just not something C called out as a "key symbol" (C does list
  `geometry.go` as a relevant file for the same reason A does).
- B named `parseWindowSize` explicitly (matching C) and discussed
  `errAttachChainSuppressed` by name in its prose explanation, even though
  the final JSON block's `key_symbols` array omits it as a discrete entry.

## Summary quality (qualitative)

Both A and B correctly identify the actual mechanism C found: two
independent, unpersisted geometry sources (told box at attach, live box
everywhere else), the option-pin/readback reconciliation, and the
literal-`;` chained `select-layout` trick that lets a resize land verbatim
instead of being silently rescaled by tmux. Both correctly conclude reed
persists no last-known geometry.

One real difference: on "does an in-place terminal resize (no new attach)
ever get handled by reed code," **B states directly and confidently** that
it is delegated entirely to tmux's own `window-size=latest` pin and that no
`resize`/`SIGWINCH`/hook path exists anywhere in the two packages — matching
C's claim almost verbatim. **A hedges the same finding as an "open
question"** ("[not] found... no SIGWINCH or push-based resize handler
exists") despite otherwise rating its own confidence "high" — a minor
internal inconsistency in A's answer, and the one point where B's answer
reads as a closer match to C's.

## Efficiency (only meaningful because A and B scored comparably on
correctness — B if anything scored a bit better, so this is not "a faster
wrong answer")

| | tokens | tool_uses | duration_ms |
|---|---|---|---|
| A (with quarry) | 78,829 | 30 | 150,137 |
| B (without quarry) | 80,894 | 20 | 156,462 |
| C (reference) | 74,465 | 23 | 158,209 |

A used 33% *more* tool calls than B (30 vs. 20) while using marginally fewer
tokens (-2.6%) and finishing marginally faster (-4.0%, ~6.3s). Tool-call
count did not translate into a token or wall-clock win here, and did not
translate into a correctness win either — B's recall and precision were
equal to or higher than A's on both relevant_files and key_symbols.

## Verdict

**On this single pilot task, quarry (`toc`) did not show a measurable
advantage over plain Read/Grep/Bash/Glob** — B matched or slightly exceeded
A on both recall and precision (files and symbols) and on qualitative
fidelity to the fasit's most subtle claim (no in-place-resize handling in
reed's own code), despite A spending 50% more tool calls to get there;
A's edge was a small (4%) wall-clock and (2.6%) token saving, not a
correctness win. A plausible explanation specific to this codebase: Loomyard's
files carry unusually rich, load-bearing header/doc comments (as seen
throughout `reedengine`), which makes plain `Read`+`Grep` nearly as
informative as `quarry toc`'s structured survey for a "read every file's
purpose" exploration task — the gap `toc` is meant to close (fast
docstring/signature survey without opening whole files) may be narrower here
than it would be on a less-documented codebase. Per the design rationale
("one run per arm per task, first pass... rerun if ambiguous"), this result
is not sharply ambiguous — B did not just tie, it edged out A — so no rerun
is triggered by the README's own rule, but it is a single run and should not
be over-read as a general verdict on quarry.
