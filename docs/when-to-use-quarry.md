# When to grant an agent quarry's tools

Practical guidance for whoever decides which quarry-mcp tools an agent gets — a `.mcp.json`/tool-allowlist
choice, or a prompt telling an agent what's available. Grounded in three independent measurements that reach
the same conclusion from different angles: [scout-vs-grep.md](scout-vs-grep.md) /
[scout-agent-usage-findings.md](scout-agent-usage-findings.md) (hand-picked hard cases, n=1 per cell, pre-rename),
the loomyard-eval capability ladder (`bench/loomyard-eval/ladder`, n=3 per cell, 2026-08-30/31), and its
task-05 follow-up (`bench/loomyard-eval/ladder/results/2026-09-01-task05`, n=3 per cell, 2026-09-01) built
specifically to probe a genuine same-name symbol collision the main ladder never tested.

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
  Task 2 in scout-vs-grep.md, the one clean, uncomplicated win in that document. Caveat: this isn't
  automatic just because two symbols share a name — see the task-05 result below, where a comparably sharp
  `.Resolve(` collision was still fully solvable by grep alone.
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
- **Even a same-package, comment-adjacent name collision, on its own, isn't enough.** The task-05 follow-up
  was purpose-built to find a sharper trap than task 04's: `mergeresolve.Resolver.Resolve` vs.
  `modelspec.Registry.Resolve`, both spelled `.Resolve(`, sitting in the same package that defines the real
  target method, three lines below a doc comment narrating that method's own resolution flow — a case a
  careless grep-and-skim pass looks primed to get wrong. It didn't: `impact`, `assert_no_callers`, and
  `textDocument_references` all matched the no-tool control's perfect recall/precision, and two of the three
  (`assert_no_callers`, `textDocument_references`) were measurably *slower* than the control for it. A name
  collision alone doesn't guarantee grep fails — the collision has to actually be reachable by a plausible
  grep-and-skim path the agent would take, not just theoretically confusable.
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

The open question from the first two task ladders — does quarry ever separate from grep on *correctness*,
not just cost, on a genuine disambiguation trap — was the specific thing task 05 was built to close. It
closes it negatively, at least for the one case tested: even a real, verified (compiler-confirmed) same-name
collision sitting inside the very package that defines the target method was fully navigable by a
grep-reading agent with no tools at all. That is one data point, not a proof that no disambiguation trap
exists that quarry's tools would resolve and grep wouldn't — a case where the colliding symbol is *not*
package-local and *not* narratively adjacent to the target, or where the codebase is large enough that a
plausible grep pass wouldn't happen to land on the right file at all, remains untested. But the bar for
"grep can't solve this" is evidently higher than "the two symbols share a name," even under fairly hostile
conditions. Treat the guidance above as "how to spend tool calls efficiently," not "proof tools make answers
more correct" — that claim now has one real, purpose-built test against it and did not survive.
