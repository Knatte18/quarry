# When to grant an agent quarry's tools

Practical guidance for whoever decides which quarry-mcp tools an agent gets — a `.mcp.json`/tool-allowlist
choice, or a prompt telling an agent what's available. Grounded in two independent measurements that reach
the same conclusion from different angles: [scout-vs-grep.md](scout-vs-grep.md) /
[scout-agent-usage-findings.md](scout-agent-usage-findings.md) (hand-picked hard cases, n=1 per cell, pre-rename)
and the loomyard-eval capability ladder (`bench/loomyard-eval/ladder`, n=3 per cell, 2026-08-30/31).

**The short version: quarry's tools are an efficiency lever, not a correctness guarantee, and they cost
more than they save on a task a skilled grep-based agent can already solve.** Grant them when the task has
a real disambiguation problem grep can't solve on its own — not by default, not "just in case."

## Grant them when...

- **The codebase is unfamiliar and you need orientation fast.** `toc_dir`/`toc_file` were the single
  clearest win in the ladder data: `toc_dir` alone cut a fresh-exploration task from 9 turns and 154k
  cached tokens (control) to 4 turns and 71k — for the same answer quality. This is the safest tool to
  grant by default for "get your bearings" work.
- **The symbol name is genuinely ambiguous and un-anchored.** Same bare name in multiple packages, an
  unexported interface method, a name so generic grep returns dozens of unrelated hits with no
  distinguishing token (`Run`, `Resolve`, `clock.Now`). This is where LSP resolution earns its cost — see
  Task 2 in scout-vs-grep.md, the one clean, uncomplicated win in that document.
- **You need "what breaks if I change/delete this," not just "where is this mentioned."** `impact` and
  `assert_no_callers` answer a question grep structurally cannot: which of the textual hits are *real*,
  type-checked call sites, and what does each caller's full enclosing declaration look like. Both were
  cheap in the ladder data (1 quarry call, no separation from the control on turns) and correct.

## Think twice when...

- **A grep with a distinctive anchor would already resolve it.** Package-qualified names, rare type names,
  file/directory scoping — a careful grep-based agent routes around ambiguity just as well and cheaper. On
  the ladder's impact task, every configuration including zero tools scored perfect recall/precision; the
  answer key had no disambiguation problem to solve, so every LSP-tool config was strictly *slower* than
  the no-tool control for no accuracy gain. Don't grant tools reflexively for a task type you haven't
  checked actually needs them.
- **You're about to grant the full seven-tool bundle "to be safe."** The bundle configs (`a5-bundle`,
  `b7-bundle`) had the highest cost in every ladder cell they appeared in — the most cached context, the
  longest wall-clock — without beating the narrower, task-matched rungs on correctness. Pick the one or two
  tools the task actually needs instead of granting everything.

## Always do this when you do grant them

- **Scope `workspace_symbol`/`textDocument_references`/`impact`/`assert_no_callers` with `within` whenever
  you already know the package or directory.** Unscoped queries on a generic name return every
  structurally-matching hit workspace-wide — gopls' own documented behavior for interface methods, not a
  bug — and the caller pays for filtering that noise out by hand. Task 3 in scout-vs-grep.md is the
  cautionary tale: an unscoped `refs` call returned ~30 hits for a 3-site answer, *with* the
  `"resolution":"complete"` trust marker still set. `--within`/`within` (added specifically in response to
  that finding) turns the same query into the correct 3.
- **Prefer symbol-form or `file:line:character` addressing over a bare name once you have a location.** A
  bare-name `workspace_symbol` search is a project-wide fuzzy match and a second round trip to disambiguate;
  a `toc_file`/`toc_dir` call already gives you every symbol's exact position up front — use it.
- **Don't trust a completeness marker on a generic symbol name without a sanity check.** "Complete" means
  "every structurally-matching result was returned," not "every result is the one you meant." Scope the
  query instead of trusting the marker to have scoped it for you.

## What this doesn't tell you yet

Both task ladders here have a real design gap: the loomyard-eval impact task (Ladder B) turned out fully
solvable by grep alone, so it measured cost, not accuracy, for every rung. Nothing in this data confirms or
rules out a correctness win on a task grep genuinely can't solve — that needs a harder fasit, not more reps
of the same one. Treat the guidance above as "how to spend tool calls efficiently," not "proof tools make
answers more correct" — the latter is still an open question.
