You are investigating an open, unresolved question about a Go tool called "quarry" (repo root: /home/hanf/Code/quarry/wts/quarry). You have no prior context beyond what's in this prompt — ground everything you claim in what you actually read/find, not assumption.

## Background

Quarry is an MCP server (cmd/quarry-mcp, engine in internal/quarryengine) that gives an LLM coding agent LSP/tree-sitter-backed code-intelligence tools instead of relying on grep/Read: `toc_dir`, `toc_file`, `textDocument_definition`, `textDocument_references`, `workspace_symbol`, `impact`, and `assert_no_callers` (the canonical seven — see internal/quarryengine and bench/loomyard-eval/ladder/internal/ladder/ladder.go's `QuarryTools` constant for the authoritative list).

There is a homegrown benchmark, the "loomyard-eval capability ladder" at bench/loomyard-eval/ladder/, that dispatches real Claude Code agent sessions against a pinned commit of a separate codebase (Loomyard, checked out as a sibling repo at ~/Code/loomyard/wts/loomyard) with different tool allowlists, and scores recall/precision/cost per config. Read bench/loomyard-eval/ladder/README.md for the harness's full design.

Three rounds of this benchmark have run so far, and the pattern is consistently negative on *correctness*:

1. **Main 2026-08-30 matrix** (bench/loomyard-eval/ladder/results/2026-08-30/summary.json, 45 runs, ladder.yaml). Exploration tasks (Ladder A) showed `toc_dir`/`toc_file` as a clear efficiency win (fewer turns, less cached context, same answer quality). Impact-analysis tasks (Ladder B) showed ZERO correctness separation — a no-tool control already got perfect recall/precision by grep alone, on every LSP-tool config, and every tool config was strictly slower for it.
2. **A followup** (bench/loomyard-eval/ladder/ladder-followup.yaml) re-testing three configs after two quarry bug fixes (commits 1592f4e, ee84d8d) — check whether this has actually been run yet and what its results directory shows, if it exists.
3. **A brand-new task 05** (bench/loomyard-eval/tasks/05-mergeresolve-resolve-impact.md, results at bench/loomyard-eval/ladder/results/2026-09-01-task05/summary.json), purpose-built to find the sharpest possible genuine symbol-name collision: `mergeresolve.Resolver.Resolve` vs. `modelspec.Registry.Resolve`, both spelled `.Resolve(`, sitting in the *same package* that defines the real target method, three lines below a doc comment narrating that method's own resolution flow. This STILL found zero correctness separation: `impact`, `assert_no_callers`, and `textDocument_references` all matched the no-tool control's perfect recall/precision, and two of the three were measurably *slower* than the control.

The distilled guidance from all this is written up at docs/when-to-use-quarry.md — read it fully; it documents this negative pattern explicitly and names what's still untested. Also read docs/scout-vs-grep.md and docs/scout-agent-usage-findings.md for the earlier, smaller-scale (n=1) evidence this guidance also draws on.

## The open question

Is there a *class* of task, codebase condition, or quarry capability gap where these LSP/tree-sitter-backed tools would actually separate from a careful grep-based agent **on correctness**, not just efficiency? The operator (who has run all three rounds above) does not want you to just re-confirm the existing negative finding — they specifically believe there are unexplored angles here and want you to find them, using both the codebase itself and outside research. If the honest answer after real investigation is "no, and here is a well-reasoned case why not," that is also an acceptable and useful conclusion — but it must be earned by actual investigation, not asserted from the existing three data points alone.

## Your task

### Part 1 — Ground yourself in what's actually been tried and ruled out
- Read the ladder harness code (bench/loomyard-eval/ladder/internal/ladder/*.go — especially ladder.go, summarize.go, score.go), the task files (bench/loomyard-eval/tasks/*.md), the results summaries named above, and the three docs named above.
- Read internal/quarryengine's doc.go and enough of the tool implementations (look for impact.go, refs.go, verify.go, registry.go, position.go and similar under internal/quarryengine/) to understand quarry's actual current capabilities and structural limits — what can it do that grep cannot, even in principle (type-checked resolution, cross-file rename impact, interface-implementer discovery, generics/embedding-aware navigation, etc.)?
- Identify precisely which task *shapes* have been tried (fresh-codebase exploration; single-symbol rename-impact analysis with one same-named decoy) and which plausible shapes have NOT: e.g. a public API with dozens of real call sites and several same-named decoys spread across many files (not just one); interface-satisfaction discovery across packages (who implements this interface?); a codebase or query scope much larger than fits comfortably in one grep-and-skim pass; Go idioms that break naive text search (embedded-struct method promotion, generics instantiation sites, closures capturing a method value, reflection-based dispatch); multi-hop call chains (A calls B calls the target, and only B is textually adjacent to a decoy); cross-package interface method sets where the same method name is required by the interface contract itself (not incidental).

### Part 2 — External research (use web search)
Search for prior art on this exact question: does semantic/LSP-based code search or "code intelligence" tooling measurably outperform grep/text-search for LLM coding agents, and under what conditions? Look for:
- Published benchmarks or ablations from agentic coding tools/papers (SWE-bench-adjacent tooling studies, design writeups from Cursor, GitHub Copilot, Sourcegraph Cody, Aider, Cline, or similar, academic papers on tool-augmented code agents) that measured grep/text-search vs. LSP/AST-based navigation head-to-head.
- Blog posts or technical writeups from teams building coding agents about when they found symbol-aware tools worth the latency/complexity cost, and when they found plain text search sufficient.
- Any research or writeups on what codebase properties (size, language, idiom density, monorepo vs. polyrepo, symbol name distinctiveness) predict when textual search breaks down for an LLM agent specifically (as opposed to a human).
Cite your sources with URLs.

## Deliverable

Write a single markdown report to `.scratch/quarry-improvement-research.md` at the repo root (/home/hanf/Code/quarry/wts/quarry/.scratch/quarry-improvement-research.md). Do not modify any other file in the quarry or loomyard repos, and do not run the ladder benchmark. The report must:

1. State plainly whether you found real evidence — from the codebase, the existing results, or external research — that a specific untested condition would separate quarry from grep on correctness, or whether the honest conclusion is "the tools' proven value is efficiency, not correctness, and here is the reasoning why that's likely the ceiling." Do not inflate weak evidence to sound more conclusive than it is; flag speculation as speculation.
2. If you found promising untested conditions: propose 1-3 concrete new ladder-task designs (in the style of bench/loomyard-eval/tasks/*.md) specific enough that someone could build each one without further discussion — including what would make its fasit (ground truth) unambiguous and independently verifiable (task 05's compiler-verification method — patching the signature and running `go build ./...` to get the real caller list — is the bar this suite already holds itself to; match or beat it).
3. Separately, propose any quarry tool/feature gaps you noticed while reading internal/quarryengine (a capability that doesn't exist yet but would answer a real question grep structurally cannot) that might change the calculus even on already-tested task shapes.
4. List your external sources with URLs in a references section.

Keep the report focused and evidence-driven — a few pages, not a sprawling document. When you're done, your final message should just confirm the file was written and give a 3-5 sentence summary of your top-line conclusion.
