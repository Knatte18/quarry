# Can quarry ever beat grep on correctness — and how should quarry change either way?

Research report, 2026-09-01. Grounded in: the three benchmark rounds (scout-vs-grep n=1 docs, the
2026-08-30 45-run ladder, the 2026-09-01 task-05 ladder), the ladder harness and task files, the
`internal/quarryengine` + `internal/mcpserver` implementations, read-only inspection of the Loomyard
target codebase, and external published work (references at the bottom). Scope per the operator's
follow-up: not only "is there an untested condition where quarry separates on correctness," but also
"how could quarry itself be rewritten to be more effective."

## 1. Top-line conclusion

**The proven value of quarry's tools is efficiency and answer-shape, not correctness, and on the
task shapes tested so far that is very likely the ceiling — the external record now confirms this
rather than contradicting it.** The only controlled grep-vs-LSP ablation published anywhere
(arXiv 2608.13568, Aug 2026, Python/TS) reaches the same conclusion as the ladder, including on
purpose-built symbol-collision tasks like task 05: capable models disambiguate via grep plus careful
reading, and LSP tooling added tokens without adding correctness — on multi-file rename edits grep
actually *beat* the LSP arms on pass@1 (1.00 vs 0.67–0.83).

But the same record identifies four conditions under which the calculus plausibly flips, and **none
of the three rounds run so far has exercised any of them**:

1. **Scale** — every ladder task pre-scopes to one or two packages, handing grep its
   directory-scoping advantage for free and keeping the whole search space skimmable in one pass.
   The external claims of grep breakdown concentrate at large-codebase scale (Cursor's measured
   gains at >1,000 files; Sourcegraph's unverified ~400K-LOC claim).
2. **Answer-set size / dispersion** — the largest ground-truth set ever tested is 31 callers
   (scout-vs-grep task 2, n=1); the ladder tasks have 2–a handful. Per-candidate manual
   type-verification is where grep's cost grows linearly and its recall decays; that regime is
   untested at n=3.
3. **Model capability** — the ladder ran claude-sonnet-5 everywhere. 2608.13568 found LSP helped
   only its *weakest* model; Cursor's accuracy deltas ranged 6.5–23.5% by model. A Haiku-class run
   arm is the single cheapest discriminating experiment available.
4. **Task shape: reference-completeness with structurally invisible references** — promoted methods
   via embedding, method values, interface satisfaction with no assertion convention. Go idioms that
   break the `.Name(`-pattern-plus-receiver-reading strategy every winning grep agent used. Untested.

Honest assessment of each: (1) and (3) have real external evidence behind them; (2) and (4) are
mechanistically plausible but speculative — and 2608.13568's collision tasks are a warning that
"looks grep-hostile" keeps turning out to be solvable by careful reading. If a fourth round on the
task designs in §4 *still* shows no correctness separation, the negative conclusion should be
accepted as durable for codebases of Loomyard's size and discipline, and investment should go to §5
(making quarry cheaper and higher-altitude), not to more trap-hunting.

## 2. What the three rounds actually established (and two corrections)

- **Established positive:** `toc_dir`/`toc_file` is a real efficiency win (9→4 turns, 154k→71k
  cached tokens on exploration, same answer quality). This matches the strongest external pattern:
  structure-as-*overview* (Aider's repo map, Agentless's hierarchical localization) is the form of
  code intelligence with the best published track record — not structure-as-precision.
- **Established negative:** on impact-analysis tasks scoped to 1–2 packages with ≤31-site answers
  and 1 decoy, every tool config matches a no-tool control's correctness and costs more. Task 05
  pushed the decoy to same-package, comment-adjacent placement and still got zero separation; the
  control's own answers show why — the winning strategy is type-reading (interface decl +
  compile-time assertion + receiver type), not pattern-matching, and Sonnet-class agents do it
  reliably.
- **Correction 1 — task 05's precision number is an artifact, and the docs overstate the result.**
  `results/2026-09-01-task05/summary.json` records precision 0.133 (recall 1.0) for *every* cell,
  not the "perfect recall/precision" `docs/when-to-use-quarry.md` describes. Cause (verified in
  `raw/c0-none/1/answer.json`): every agent in every config listed the 13
  `mergeresolve_test.go` call sites in `callers_to_update`, because the task text asks for call
  sites that "would need a `reason` argument threaded in to keep compiling" — which literally
  includes tests — while the fasit's scoring notes exclude them. So precision was uniformly
  deflated by a task-text/fasit mismatch and could not have separated anything. The comparative
  conclusion (no separation) stands, since all cells were identical, but: (a) future impact tasks
  must state explicitly whether test files count (they should — they do need edits to keep the
  build green, and `go vet ./...` type-checks them); (b) when-to-use-quarry.md's "perfect
  recall/precision" phrasing should be corrected.
- **Correction 2 — the fix-verification follow-up has not run.** `ladder-followup.yaml` exists
  (re-testing b1-symbol/b2-definition/b4-lsp-trio after the `within`-scoping fix 1592f4e and the
  symbol-form-steering fix ee84d8d), but there is no results directory for it. The two fixes are
  unverified against the defects they were written for. This should run before any fourth round,
  because both fixes bear directly on the friction findings that motivate §5.

## 3. What the external record says (Part 2 of the brief)

Full references in §8. The short version:

- **The one controlled ablation agrees with the ladder.** "Does a Language Server Save Tokens for
  Coding Agents?" [1]: five arms (grep-only / LSP-only / free-choice / forced-semantic / repo-map),
  Claude models, Python+TS. LSP added 6–118% tokens on symbol-named localization; with free choice,
  models used semantic tools 0–6% of the time for localization but ~50% for
  reference-completeness tasks; on rename-type edits grep scored pass@1 1.00 vs 0.67–0.83 for LSP
  arms (LSP's structural exclusion of comments/strings *hurt* rename edits). Its recommendation is
  an adaptive router keyed on task class, model capability, and lexical noise — which is what
  docs/when-to-use-quarry.md is by hand.
- **Interface shape dominates tool semantics.** SWE-agent's ablation [2]: bash-only grep/find
  15.7% vs purpose-built summarized (still lexical) search 18.0% — and a badly shaped iterative
  search variant scored *below* plain grep (12.0%). Agentless [3] hit competitive SWE-bench scores
  with no agent and no LSP, using hierarchical structure overviews. Lesson for quarry: how results
  are shaped and compressed matters more than semantic precision — consistent with toc being the
  ladder's only clear win.
- **The measured pro-semantic results are a different axis.** Cursor's semantic search [4]
  (+12.5% answer accuracy vs grep-only, gains concentrated in >1,000-file codebases) is
  *embeddings for fuzzy queries where the symbol name is unknown* — not LSP precision. LocAgent
  [5] and RepoGraph [6] show graph structure helps localization, but never against a tuned
  grep-agent baseline. Monitor-Guided Decoding [7] proves LSP facts fix identifier hallucination,
  but via constrained decoding, unavailable to a tool-calling agent.
- **Practitioner consensus: grep is the backbone; LSP is for verification-shaped moments.**
  Anthropic dropped embeddings for agentic grep in Claude Code (self-described as "mostly vibes"
  plus internal benchmarks) [8]; Cognition's answer to retrieval cost was RL-trained *parallel
  grep*, not semantic indexing [11]; the best synthesis piece [9] surveys six products all
  defaulting to ripgrep and catalogs LSP's operational failure modes (init latency, indexing
  stalls, broken-config fragility — cf. Lanser-CLI's failure-mode taxonomy [10]). The single best
  pro-LSP *correctness* datapoint anywhere is anecdote-grade: ManoMano's n=1 Java refactor where
  Claude+Serena passed all tests and plain Claude Code failed [13]; Serena itself publishes no
  quantitative eval [14].
- **Nobody has published a controlled grep-vs-LSP correctness ablation on Go, at monorepo scale,
  or with weak models as the explicit variable.** Those are the open cells, and the ladder is
  well-positioned to fill the Go one.

One Loomyard-specific observation that sharpens why grep keeps winning *here*: the codebase's own
discipline is grep's ally. Calls are package-qualified (`hubgeometry.Resolve(`), interfaces carry
`var _ Iface = (*Impl)(nil)` assertions (29 of them — every ShedProducer implementer has one), and
packages are small. Each convention manufactures exactly the textual anchor grep needs. The
codebase *condition* "no assertion discipline, dot-imports, or generated code" is a real,
nameable predictor of grep failure that no round has tested — but building it honestly requires a
target codebase that actually lacks the discipline, not a fabricated one.

## 4. Proposed fourth-round task designs (Part 1 of the deliverable)

All three follow task 05's fasit bar or better: ground truth from the compiler, not from any
agent or tool. Note `go build ./...` does **not** type-check `_test.go` files — use
`go vet ./...` (or `go test -run '^$' ./...`) for the fasit so test call sites are enumerated
too, and state in the task text that test files count. All symbols below verified present at
Loomyard HEAD; re-verify at the pinned SHA (`git show <sha>:<path>`) when building, per task-05
practice.

### Task 06 — interface-method impact at implementer scale (`shedengine.ShedProducer.Call`)

**Question:** "You are adding a `stage string` parameter to `shedengine.ShedProducer.Call`
(`internal/shedengine/producer.go:31`). Identify (a) every method declaration that must change —
every type whose `Call` implements this interface — and (b) every call site that must thread the
new argument. Whole repo, test files included."

**Why this shape is untested:** it inverts the tested direction. Tasks 04/05 asked "which of these
textual hits is real" (precision); this asks "find every implementer of a structural contract"
(recall), across ≥5 packages (`shedadapters`, `preflightshed`, `loomshed`, `shedengine`, plus
wrappers), with ~13 implementers, wrapper chains that are themselves implementers *and* callers
(`loomshed.planWrite.Call` calls `p.inner.Call` — a two-hop structure), and a method name (`Call`)
as generic as Go allows. It is also the exact task shape where 2608.13568's free-choice models
*chose* LSP ~50% of the time — reference-completeness — and where quarry's
`textDocument/implementation` machinery (currently internal-only, see §5) is the structural answer.

**Honest risk:** Loomyard's `var _ shedengine.ShedProducer = ...` assertions give grep a
one-command anchor (`grep -rn "ShedProducer"`). Predicted outcome is therefore *still* no
correctness separation, and the task is worth running precisely to establish that assertion
discipline — a codebase property, not an agent skill — is what closes the gap. Record in the task
file which implementers have assertions; if any lacks one at the pin, that implementer is the
task's sharpest probe.

**Fasit:** at the pin, patch the interface method to
`Call(ctx context.Context, stage string) (Outcome, OutputPointer, error)` and run
`go vet ./...`; the error list *is* the ground truth (assertion failures name every implementer;
call-site errors name every caller). Revert. Unambiguous, independently reproducible, matches the
task-05 bar.

### Task 07 — whole-repo, high-dispersion method impact with no scope hint

**Question:** signature change to a *method* (not a package-qualified function) with the largest
real caller set findable at the pin, task text scoped to the whole repo with no package list.
Candidate: re-use the `hubgeometry.Resolve` family's shape but pick a method invoked through a
receiver (`x.Resolve(...)` / `x.Run(...)`) so no package-qualified grep form exists; survey at the
pin with `grep -rn "\.Run(" --include="*.go"` for a method with ≥25 dispersed callers and ≥3
same-named unrelated methods (the scout-vs-grep doc records 26 raw `.Run(` hits across unrelated
types — that family is the right hunting ground).

**Why this shape is untested:** it removes both of grep's free levers at once — no directory
scoping (whole repo) and no package-qualification anchor (method call) — and pushes the answer set
into the regime where per-candidate type-verification cost is linear in hits and skim-recall decays.
Every prior task had ≤2 packages in scope or a qualified-name anchor or both. This is condition (2)
from §1; the prediction is genuinely uncertain, which is what makes it worth running.

**Fasit:** same compiler method — patch the method signature, `go vet ./...`, record, revert.
The fasit must also record the raw grep hit count and false-positive count at the pin (as
scout-vs-grep did: 46 hits / 11 false), so the task file can state exactly how hostile the textual
surface is.

### Task 08 — structurally invisible references (promotion and method values)

**Question:** impact analysis on a method reached through embedded-type promotion and/or method
values. Verified candidates at HEAD: `fabricengine.MutationRecord` is embedded by three types
(`cleanup.go:85`, `stencilcommit.go:25`, `checkout.go:28`) and carries `Mutated()`
(`mutation.go:226`) — any call `x.Mutated()` on an embedding type never textually mentions
`MutationRecord`; and `boardengine.TaskWithLayer` embeds `Task` (`layer.go:137`), so `Task`'s
methods are callable on a type whose name shares nothing with the target. **Build-time survey
required:** confirm at the pin that promoted call sites (and, ideally, at least one method-value
binding `f := x.Mutated`) actually exist; if `Mutated` has too few callers, hunt with
`go vet`-patching over the embedded types' other promoted methods until one has ≥3 promoted call
sites. Do not fabricate code — if the idiom is too rare at the pin, that fact kills the task and
is itself worth recording.

**Why this shape is untested:** every task so far has had textually regular call sites —
`receiver.Method(args)` with the receiver declared nearby. Promotion breaks receiver-adjacency
reading (the declared type is not the method's type), and a method value breaks the `.Method(`
pattern entirely (no parens at the binding site). This is the one Go-idiom condition where the
*pattern itself*, not just the anchor, fails. 2608.13568 did not test it (Python/TS have no
promotion).

**Fasit:** identical compiler method; `go vet ./...` after the signature patch reports promoted
call sites and method-value bindings alike, because the type-checker sees through both.

**Cross-cutting variation worth one extra cell on whichever task runs:** a `run_model:
claude-haiku-4-5-20251001` arm alongside the sonnet arm, same configs. Condition (3) is the
best-evidenced untested variable, it costs two cells rather than a new task, and a
"tools rescue weak models" result would change the deployment guidance (grant tools to cheap
agents, not to strong ones) even if the sonnet ceiling stands.

## 5. How quarry itself should change (the operator's added question)

Reading the engine and MCP layer with the ladder's findings in hand, the improvements group into
"answer the whole question in one call" (altitude), "adopt grep's structural advantages"
(defaults), and "stop paying avoidable costs" (mechanics). Ordered by expected leverage:

1. **Add a `rename_impact` tool wrapping LSP `textDocument/rename` (gopls prepareRename +
   rename).** This is the highest-leverage single change. The suite's own fasit procedure —
   patch the signature, let the compiler enumerate every affected site — proves the question every
   Ladder-B task asks is mechanically answerable at compiler grade. gopls's rename computes
   exactly that WorkspaceEdit (tests included, promotion included, method values included) in one
   call, without dirtying the tree. Today the agent must assemble the same answer from
   `refs`+manual reasoning; the tool should just return it. This converts the benchmark's
   ground-truth generator into the product. It also subsumes tasks 06–08's difficulty: if
   `rename_impact` ships, the honest benchmark comparison becomes "one tool call vs a full grep
   session," which is the shape toc already wins.
2. **Expose `implementations` as a first-class tool.** `textDocument/implementation` is already
   wired (doc.go: used internally to widen `Callers`' declaration match set) but not exposed.
   "Who implements this interface / what does this dispatch to" is the one question grep answers
   only by convention (`var _` assertions) — in codebases without that discipline it has no
   textual form at all. Cheap to ship: the plumbing exists.
3. **Make `within` scoping the default, not opt-in.** The single most consistent finding across
   all three rounds (scout task 3's 30-hits-for-a-3-site-answer; the b1-symbol gopls 100-hit
   saturation; the still-open follow-up in scout-agent-usage-findings.md) is that workspace-wide
   default scope is a liability grep never pays. Default `within` to the query symbol's own
   package (or the target dir), require an explicit `within: all` to widen. This adopts grep's
   one structural advantage instead of documenting workarounds for its absence.
4. **Unify position conventions to 1-based everywhere.** `textDocument_definition`/`_references`
   speak 0-based LSP coordinates; `impact`/`assert_no_callers` speak 1-based; the tool
   descriptions spend sentences warning agents not to paste one into the other, and
   ladder-followup.yaml documents the observed cost (mis-aimed batch, ad-hoc awk column hunt, a
   retry). Every other surface an agent sees — `grep -n`, compiler errors, editors — is 1-based.
   Convert at the MCP boundary; keep steering to symbol-form addressing as primary (ee84d8d's
   direction, unverified until the follow-up matrix runs).
5. **Wire call hierarchy for multi-hop impact.** doc.go's "No call hierarchy" carve-out has two
   costs. Correctness-side: `impact` is one hop, so transitive blast radius ("A calls B calls the
   target") — the regime where grep's cost is combinatorial (a new frontier of names every hop) —
   is unanswerable. Mechanics-side: `callersFromClient` verifies each candidate reference with its
   own sequential `textDocument/definition` round trip under one shared deadline — O(N) round
   trips, with a deadline-expiry path that silently returns partially-verified results.
   `callHierarchy/incomingCalls` returns verified callers in one request and would replace that
   loop for servers that support it (gopls does).
6. **Make the verification contract honest per-entry.** Verification silently skips (returning
   raw unfiltered refs) when the server lacks implementation support, when the declaration lookup
   errors, or when the loop deadline expires — and the entry still reports
   `"resolution": "complete"`. This is the same trust-marker gap the 2026-07-29 findings doc
   called the most important result of that round, resurfaced one layer up. Add
   `"verified": true|false|"partial"` to every refs/impact entry so an agent (or a deterministic
   `verify:` gate) knows whether the answer needs cross-checking. A tool whose pitch is
   "type-checked truth" must not silently degrade to "grep-grade list with a confident marker."
7. **Offer a compact output form.** The ladder's cost data points one direction: the win (toc) is
   the tool with the densest output; the losses carry JSON envelopes that echo the input target,
   nest results, and spend tokens on `status`/`resolution` scaffolding per entry.
   A `format: "compact"` returning grep-shaped lines (`file:line:col: enclosing-decl-signature`)
   would cut the per-call token cost that the cache_creation separations measured, at zero
   information loss for the common case. (SWE-agent's ablation [2] is the external warrant:
   result presentation moved outcomes more than tool capability did.)
8. **Extend daemon warmth beyond Go — or say the other four languages are cold-spawn-only in the
   tool descriptions.** Python/C#/TS/Rust spawn a server per call (doc.go); per-call init latency
   makes them lose the efficiency comparison by construction, and all benchmark evidence is
   Go-only. Dynamic-language codebases are also where the external record most expects semantic
   resolution to matter (and where Go's grep-friendly conventions don't exist) — a TS target with
   barrel-file re-exports is a plausible fifth-round codebase condition if quarry's TS path is
   ever made warm.

Items 3, 4, 6 are corrections quarry should make regardless of any benchmark outcome; items 1, 2,
5 are the ones that could genuinely change the correctness calculus, because each answers a
question grep cannot express, rather than answering grep's own question more expensively.

## 6. The mechanical path: Loomyard calling quarry without an agent in the loop

The operator's original motivation was Loomyard using quarry to make its Loom usage more
effective. All three benchmark rounds tested only one integration shape — *an agent choosing to
call quarry tools mid-task* — and that is precisely the shape where the evidence (internal and
external) is weakest: the cost is paid in agent turns and tokens, and the value depends on the
agent choosing the right tool, scoping it, and trusting its output correctly. The 2026-07-29
findings doc already drew this conclusion once: scout's durable value is "Go-native,
deterministic use... where no agent is ever choosing whether to trust the result." The external
record agrees from the other side: the one place LSP facts demonstrably bought correctness is
Monitor-Guided Decoding [7], where the static analysis is applied *mechanically*, below the
model's choice layer.

quarry is built for this: the engine/CLI seam (doc.go) means `quarry/` (the facade) returns typed
Go results and typed errors with no CLI, no JSON envelope, no MCP — Loomyard's own Go code (its
gates and orchestration, which are pure Go) can import it directly. Concrete integration shapes,
ordered by how directly they exploit the proven wins:

1. **Plan-time impact annexes (pre-processing).** When a Loom plan or batch names symbols to
   change, move, or delete, mechanically run `impact`/`refs` on each named symbol *before*
   dispatching the implementer, and inject the verified caller list into the prompt. The agent
   never spends a turn or a tool call; the answer arrives pre-computed. This converts the
   ladder's negative finding (tools cost turns for no accuracy gain *when the agent drives them*)
   into a win: the same information at zero agent-side cost is strictly better than either arm
   the ladder measured.
2. **toc context packs.** Same move for exploration: pre-generate `toc_dir`/`toc_file` for the
   packages a task touches and inject them. toc is the proven efficiency win (9→4 turns), it is
   daemon-free tree-sitter parsing (cheap, no warm-up problem), and injection captures the win
   without granting any tool.
3. **Deterministic `verify:` gates on Deletes/Moves.** `assert_no_callers` as a batch gate —
   the original scout-plan-symbol-fields direction — where a plan that deletes or moves a symbol
   cannot pass while verified callers remain. **Prerequisites from §5 are hard requirements
   here:** the 31-false-positive incident (scout-agent-usage-findings.md) happened on exactly
   this shape, and a gate consuming unverified results marked `resolution: complete` re-creates
   it. Item 6 (per-entry `verified` flag, fail-closed when verification skipped) and item 3
   (default scoping) must land before any gate trusts the exit code.
4. **Review-time diff impact.** After a batch completes, mechanically compute the impact of every
   changed exported declaration in the diff and hand the reviewer the verified caller list —
   review becomes checking a list instead of re-deriving it. Same machinery as (1), applied
   post-hoc.
5. **A `rename_impact` gate (once §5 item 1 ships).** For signature-change batches, the
   WorkspaceEdit-derived site list is compiler-grade and can gate "did the implementer touch
   every affected site" deterministically — the fasit procedure as a production check.

Two honest caveats. First, (1), (2) and (4) change *what goes into a prompt*, so their effect is
measurable with the existing ladder harness (an "annex" arm: no tools granted, pre-computed
context injected — worth adding as a rung, since it is the integration Loomyard would actually
ship). Second, the mechanical path inherits quarry's operational fragility without an agent to
route around it — a stalled gopls or a broken build fails the gate, not the agent's plan B — so
gates need the timeout/degradation behavior of §5 item 6 to fail *loud and classified*, never
silently pass or silently block.

## 7. Methodology notes for the fourth round

- Fix the test-file ambiguity (state it in the task text; use `go vet ./...` for the fasit).
- Run `ladder-followup.yaml` first — the two shipped fixes are unverified and gate the
  interpretation of any new friction observations.
- Correct when-to-use-quarry.md's task-05 characterization ("perfect recall/precision" →
  "identical recall/precision across all cells; precision uniformly 0.133 due to a
  task-text/fasit mismatch on test files").
- Add the Haiku arm (see §4) before adding new task shapes beyond 06–08; it is the cheapest test
  of the best-evidenced untested condition.

## 8. References

1. Does a Language Server Save Tokens for Coding Agents? — https://arxiv.org/abs/2608.13568 —
   the only controlled grep-vs-LSP ablation; LSP adds tokens on localization, loses on rename
   correctness (grep 1.00 vs 0.67–0.83 pass@1); models free-choose LSP mainly for
   reference-completeness; recommends adaptive routing.
2. SWE-agent: Agent-Computer Interfaces Enable Automated Software Engineering —
   https://arxiv.org/abs/2405.15793 — search-tool ablation: bash grep 15.7% vs summarized lexical
   search 18.0%, iterative search 12.0%; interface shape dominates.
3. Agentless: Demystifying LLM-based Software Engineering Agents —
   https://arxiv.org/abs/2407.01489 — structure-overview localization, no LSP/agent, competitive
   SWE-bench Lite results.
4. Cursor: Improving agent with semantic search — https://cursor.com/blog/semsearch — embeddings
   (not LSP) vs grep-only: +12.5% avg answer accuracy, gains concentrated in large codebases;
   internal eval.
5. LocAgent: Graph-Guided LLM Agents for Code Localization — https://arxiv.org/abs/2503.09089 —
   graph tools improve localization; no tuned grep-agent baseline.
6. RepoGraph — https://arxiv.org/abs/2410.14684 — repo-level graph plug-in, +32.8% relative on
   SWE-bench across frameworks; structure-as-context.
7. Monitor-Guided Decoding of Code LMs (NeurIPS 2023) —
   https://openreview.net/forum?id=qPUbKxKvXq — LSP facts fix identifier hallucination via
   constrained decoding, a mechanism unavailable to tool-calling agents.
8. Latent Space: Claude Code interview — https://www.latent.space/p/claude-code — Anthropic
   dropped embeddings for agentic grep; "mostly vibes" plus internal benchmarks.
9. Why Coding Agents Still Use grep as Their Search Backbone —
   https://yage.ai/share/why-coding-agents-still-use-grep-en-20260327.html — six-product survey;
   retrieval-funnel model; LSP operational failure modes.
10. Lanser-CLI / RLCSF — https://arxiv.org/abs/2510.22907 — taxonomy of 15+ LSP-specific failure
    modes for agent use.
11. Cognition: SWE-grep — https://cognition.com/blog/swe-grep — RL-trained parallel grep as the
    answer to retrieval cost; >60% of first-turn time was context retrieval.
12. Aider repo map — https://aider.chat/2023/10/22/repomap.html — tree-sitter + PageRank overview;
    the canonical structure-as-overview design (quarry's toc analog).
13. ManoMano: Project Aegis / Serena —
    https://medium.com/manomano-tech/project-aegis-benchmarking-ai-agents-and-why-serena-is-our-new-must-have-311673db35dd
    — n=1 Java refactor; best pro-LSP correctness datapoint found, anecdote-grade.
14. Serena (oraios) — https://github.com/oraios/serena — leading LSP-MCP toolkit; no quantitative
    eval published.
15. Sourcegraph: Why coding agents fail in large codebases —
    https://sourcegraph.com/blog/why-coding-agents-fail-large-codebases — vendor claim of grep
    degradation past ~400K LOC; not independently verified (page bot-blocked during research).
16. agent-lsp — https://github.com/blackwell-systems/agent-lsp — vendor-reported Consul (319K LOC)
    numbers: 17.7MB/5,534 calls grep vs 841KB/119 calls LSP; efficiency, not correctness.
