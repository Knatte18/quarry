# Deep quarry-mcp review from ladder benchmark findings

```yaml
task: quarry-mcp-ladder-findings-review
data: bench/loomyard-eval/ladder/results/2026-08-30/raw (42 runs, 14 configs x 3 reps, read in full)
source_review: internal/mcpserver (all non-test files), internal/quarryengine/toc, internal/quarryengine/query, internal/cli
date: 2026-08-31
```

## Scope caveat: this analysis is efficiency-only

**There is no correctness or accuracy data in this analysis.**
As of this review, `bench/loomyard-eval/ladder/results/2026-08-30/raw/` contains no `a5-bundle-cold/` directory and no `score.json` anywhere (verified by directory listing and a recursive find), so the cold cell and the scoring session have not run.
Every comparison below is turns, duration, cost, and payload size — never answer quality.
One run (b7-bundle/1) produced an answer I judge likely wrong against the task's own stated requirements (section 5.8), which is itself a warning: the cheapest-looking config is not automatically the best one, and nothing here should be treated as a tool-removal argument until scored.
All 42 answers to task 04 except b7-bundle/1 named the same three call sites and the same excluded lookalike, so on that task the efficiency comparison is at least comparing apples to apples.

## 1. Data-quality findings (read these before the numbers)

### 1.1 usage.json multiply-counted every multi-block API call — now fixed

`ExtractUsage` (bench/loomyard-eval/ladder/internal/ladder/usage.go:124-131 at commit 319d623) summed `message.usage` over every assistant transcript record and reported `num_turns` as the record count.
Claude Code writes one subagent-transcript record per content block;
every record of one API call repeats that call's usage snapshot under one `message.id` (verified: matching `requestId` across same-id records in a2-toc-dir/1).
Consequences, measured against per-message re-aggregation of the raw transcripts:

- a2-toc-dir/1: reported `num_turns: 10` vs 4 actual API calls; reported `cache_read_input_tokens: 152179` vs 70,726 actual (2.15x).
- a0-none/1: reported 18 turns vs 9 actual; 309,862 cache-read vs 162,998 actual (1.90x).
- The inflation factor varies per run with the blocks-per-message ratio, so cross-config comparisons of the raw usage.json numbers are systematically biased, not just scaled.

The task brief's note that the summed fields "ARE valid for cost estimation" is therefore wrong as stated — billing is per API call, and the summing was per record.
Fixed in this branch: `assistantCallGroups`/`perCallUsage` now aggregate once per message id (bench/loomyard-eval/ladder/internal/ladder/usage.go), with a real-format fixture (`testdata/multi-record-call.jsonl`).
Existing 2026-08-30 usage.json files on disk still carry the inflated numbers; re-ingesting the transcripts would correct them.

### 1.2 output_tokens is unreliable in both directions

Per-record `output_tokens` is a streaming snapshot: within one message the records of a2-toc-dir/1 carry 4 then 88, and the final answer-bearing record of that run carries `output_tokens: 1` for a multi-hundred-token answer.
Neither per-record summing nor last-record reads recover the true output;
the fix takes the per-call maximum, documented as a lower bound.
Output is the smallest cost component (~$0.01-0.08/run at $15/MTok), so this does not move the comparisons, but per-run output numbers should not be quoted as precise.

### 1.3 b6-assert-no-callers/2 is contaminated and must be excluded

Its transcript's user prompt is the literal dispatch string `ladderbench run b6-assert-no-callers rep 2 attempt 1` — not the composed task prompt (raw/b6-assert-no-callers/2/transcript.jsonl, first user record).
The agent then explored the harness's own files to reconstruct its task: it found `.scratch/ladder-sessions/`, read the agent definition and `.mcp.json`, and parsed rep 1's `transcript.jsonl` with python3 to extract rep 1's prompt, before finally doing the task (correctly).
That is a dispatch bug in whatever session drove that rep, and the rep is neither independent nor comparable (14 real API calls, ~8 of them harness spelunking).
b6's clean reps (1 and 3) average 5.0 real turns and $0.101 — which changes b6 from mid-pack to the second-cheapest config in ladder B.

### 1.4 b7-bundle/1's 119s duration is a harness stall, not tool cost

The transcript shows an 82-second gap between the first assistant message (16:54:19, the run's first-ever quarry MCP calls) and the second (16:55:41);
every later inter-turn gap is normal.
b7/1 was the first run of the whole matrix (2026-08-30 16:54, before every a-ladder run), and the timeline matches the MCP trust-dialog stall the harness subsequently fixed (commits 0a6c56f 2026-08-30 21:35, 19d92d3 2026-08-31 07:53).
b7's other reps ran 44-48s.

### 1.5 Minor: the matrix spans two days

a0-a5 and b0-none/1-2, b7 ran 2026-08-30; b0-none/3 and b1-b6 ran 2026-08-31 05:19-06:42 (transcript timestamps).
Same harness and prompts, so I treat them as one matrix, but it is worth knowing when comparing durations.

## 2. Corrected results matrix

Recomputed from the transcripts: `turns` = unique API message ids; cost = per-call input/output/cacheR/cacheW at $3/$15/$0.30/$3.75 per MTok; `last ctx` = the final assistant record's own `input + cache_read + cache_creation` (true end-of-run context).

Task 01 — exploration (reedengine/reedcli geometry reconciliation):

| config | real turns | duration s | cost $ | last ctx (tok) |
|---|---|---|---|---|
| a0-none | 9.0 | 60 | 0.236 | 31.7k |
| a1-toc-file | 6.7 | 56 | 0.222 | 31.1k |
| a2-toc-dir | 4.3 | 49 | 0.248 | 47.1k |
| a3-toc-pair | 5.3 | 59 | 0.275 | 48.7k |
| a4-toc-pair-symbol | 6.7 | 52 | 0.292 | 45.8k |
| a5-bundle | 6.3 | 69 | 0.271 | 51.0k |

Task 04 — impact analysis (Shuttle.Run call sites, interface-method conflation):

| config | real turns | duration s | cost $ | last ctx (tok) |
|---|---|---|---|---|
| b0-none | 5.0 | 20 | 0.138 | 23.6k |
| b1-symbol | 4.3 | 33 | 0.278 | 58.1k |
| b2-definition | 8.3 | 35 | 0.164 | 23.5k |
| b3-references | 6.7 | 35 | 0.137 | 26.5k |
| b4-lsp-trio | 8.0 | 48 | 0.230 | 36.8k |
| b5-impact | 8.0 | 31 | 0.158 | 26.6k |
| b6-assert-no-callers | 8.0 (5.0 excl. rep 2) | 33 | 0.132 (0.101 excl. rep 2) | 22.1k |
| b7-bundle | 6.7 | 70 (46 excl. rep 1 stall) | 0.201 | 32.8k |

Per-quarry-tool result payloads across all 42 runs (chars, from every tool_result):

| tool | calls | payload range | mean |
|---|---|---|---|
| toc_dir | 13 | 11,470 - 48,709 | ~39.7k |
| workspace_symbol | 8 | 343 - 25,598 | ~15.8k |
| toc_file (successful) | 17 | 2,323 - 24,756 | ~10.9k |
| textDocument_references | 19 | 152 - 1,903 | ~1.0k |
| impact | 3 | 1,530 - 1,563 | ~1.5k |
| textDocument_definition | 6 | 269 - 1,137 | ~0.7k |
| assert_no_callers | 3 | 440 - 495 | ~0.5k |

## 3. Seed findings, re-verified

### Seed 1 — workspace_symbol lacks scoping: confirmed, and now fixed

Source: `symbolEntry` accepted only `query` (internal/mcpserver/tools_symbol.go:23-28 pre-fix), `symbolInput` explicitly documented the absent `within` (tools_symbol.go:48-50 pre-fix), while `resolveLSPEntry` applies `cli.FilterWithin` post-hoc for definition/references (internal/mcpserver/tools_lsp.go:125-128), and `quarry.SymbolMatch` carries the `File` field the filter needs (internal/quarryengine/query/symbol.go:25-31).
Empirics: confirmed and sharpened.
b1's `{"query":"Run"}` and `{"query":"Shuttle"}` each returned exactly 100 symbols — gopls's workspace/symbol cap saturated by fuzzy matching — for ~25.4-25.6k chars per two-query call, in all three reps.
Worse, b4-lsp-trio/1's specific `{"query":"BouncerConfig"}` returned 13,774 chars led by module-cache noise: `go-git`'s `doGetHostWithPortFromSSHConfig` and `go-github`'s `GenerateOrgJITConfig` fuzzy-subsequence-match "BouncerConfig" and outrank the repo's own type in the result order.
The role nuance in the seed also held: b7-bundle never called workspace_symbol in any of its three reps, and the one clean a-ladder use (a4-toc-pair-symbol/3, `{"query":"liveBoxLocked"}`, 343 chars) was bootstrapping a name it had not yet located.
Implemented: per-entry `within` on workspace_symbol plus `--within` on the CLI symbol verb, filtering through a new `cli.FilterSymbolsWithin` that shares `FilterWithin`'s normalization (commit 1592f4e).
A `within` scoped to the project root alone would have cut the module-cache noise entirely.

### Seed 2 — toc_dir payloads: size right, content wrong

The content claim was wrong: `TOCDir` emits per file only language, package, first-paragraph header, and test/generated flags — never signatures or docstrings (`DirEntry`, internal/quarryengine/toc/types.go:76-102; "it emits headers, never docstrings", toc.go:146-147).
The size claim was right: every 2-directory a-ladder call returned 46,839-48,709 chars (10 such calls), and b7's single-directory calls 11,470.
The payload driver is loomyard's multi-sentence file headers times file count (test files included — roughly half the entries), not symbol detail.
The seed's "names only / names + one line" idea therefore maps to a header-truncation or exclude-tests option, not a detail-level switch — see recommendation R3.
Separately, the tool's own description was false — it claimed "1-based, inclusive line numbers on every listed file's own toc_file result shape" (tools_toc.go:230-231 pre-fix) while the result carries no line numbers and no symbols at all; fixed in commit ee84d8d.
The efficiency story flips the seed's framing, though: see seed 6.

### Seed 3 — TOC-then-Read is by design: confirmed, not re-flagged

Confirmed in every transcript: toc results select files, Reads fetch logic.
The one real redundancy runs the other direction: in a3/a4, agents called `toc_file` on the exact files they then immediately Read in full (e.g. a3-toc-pair/1 toc_file on windowsize/attach/attach/geometry, then Reads of the same files), adding ~9-15k chars of payload with no visible turn saving over a2's toc_dir-then-Read flow.

### Seed 4 — definition's position friction: confirmed, plus a wire bug it exposed

b2-definition/3 is exactly as described: a 4-target position batch aimed at wrong columns, an ad hoc `awk` printing per-character 0-based column indices for line 143, then a corrected batch that hit (raw/b2-definition/3, tool uses 6-9).
Corrected magnitude: 8.3 real turns vs baseline 5.0 (the seed's 16.3 vs 11.0 were inflated record counts).
The mis-aimed batch also exposed a genuine wire bug: gopls answers a non-identifier position with an empty location list and no error, and the entry came back `"status":"found","resolution":"complete"` with **no `definitions` key at all** — `omitempty` drops the empty-but-non-nil slice `referenceFieldsWire` deliberately builds (internal/mcpserver/result.go:54-60), on `definitions`, `references`, and assert's `callers` alike (defeating tools_assert.go's own "always a non-nil, possibly-empty slice" comment).
Fixed with `omitzero` tags plus marshalling tests (commit ee84d8d).
By contrast the symbol-addressed form worked cleanly wherever used (b2-definition/2, b7-bundle/2 and /3), and the 1-based native tools (impact, assert_no_callers) had zero position friction across all their runs — agents naturally produce 1-based editor-convention positions.
The tool descriptions now steer to symbol-form addressing (commit ee84d8d).

### Seed 5 — the trio's whole > parts: direction holds, superlative does not

Corrected: b4-lsp-trio averages 8.0 real turns — worse than b0 (5.0), b1 (4.3), and b3 (6.7), but marginally better than b2-definition alone (8.3).
The mechanism is visible and is not a harness artifact: all three b4 reps paired their (correct, sufficient) references call with a speculative parallel `workspace_symbol {"query":"Run"}` costing ~12k chars of fuzzy noise that contributed nothing to any answer (raw/b4-lsp-trio/1-3), encouraged by the benchmark preamble's use-parallel-tool-calls instruction.
Cost: $0.230 avg, second highest in ladder B after b1 — the noise is cached and re-billed every following turn.

### Seed 6 — ladder A build-up: substantially corrected

With real API-call counts, the picture inverts after a1:

- a1-toc-file vs a0: -2.3 turns, -6% cost — a clean win (seed confirmed).
- a2-toc-dir vs a1: **another -2.4 turns** (4.3 vs 6.7, less than half the baseline's 9.0) and the fastest wall-clock in the ladder, for +12% cost over a1 (+5% over baseline).
  The seed's "everything after toc_file traded cost for flat-to-negative gains" is wrong for a2: toc_dir is the strongest single addition in the whole matrix on turns and duration.
  a2-toc-dir/1 is the flagship: one toc_dir call, four targeted Reads, done in 4 API calls / 44s — and the 46.8k payload carried real signal, surfacing `attachgeometry_integration_test.go` via its header, which the a0 grep-based runs never found and whose content deepened all three a2/a3 answers.
- a3 (+toc_file): +1.0 turns and +11% cost over a2 — the toc_file-then-Read redundancy of seed 3.
- a4 (+workspace_symbol): +1.4 turns, +6% over a3 — confirmed regression, now attributable: rep 3 wandered (8 calls), and the marginal tool got one 343-char use in three reps.
- a5 (full bundle): 6.3 turns, $0.271 — no better than a3 on anything.

The corrected conclusion: for exploration, toc_dir alone (or toc_dir + Read) is the optimum; each further tool adds payload and decision overhead without turn gains.

### Seed 7 — cache-write premium: confirmed and quantified

Cache-write is 25% over base input ($3.75 vs $3.00/MTok) and every early payload is re-read at $0.30/MTok on each later call.
b1's end-of-run context (58.1k avg vs baseline's 23.6k) is the clearest case: its corrected cost of $0.278 is 2.0x baseline with the fewest turns of any B config — the cost lives in the 25.4k symbol payload plus 70k chars of whole-file Reads (bouncer.go 33k, burler.go 26k, singlellm.go 11k) that the references-less tool shape forced.
The general rule the matrix supports: a tool's typical payload size matters more than its call count, because context is re-billed per subsequent call.

## 4. What was implemented on this branch

```yaml
commits:
  - ee84d8d: omitzero wire fix (definitions/references/callers/symbols) + toc_dir/docSentences/LSP-tool description corrections, with marshalling tests
  - 1592f4e: within scoping for workspace_symbol (per-entry) and the CLI symbol verb (--within), via cli.FilterSymbolsWithin; parameter-matrix transport test row flipped deliberately
  - 7fdb4f2: ladder harness per-API-call usage accounting (assistantCallGroups/perCallUsage) + real-format fixture
verify: go build ./... && go vet ./... && go test ./... — all 16 packages ok
```

## 5. Additional findings (not in the seeds)

### 5.1 The docSentences quoting footgun (fixed)

5 of the 6 sessions that supplied `docSentences` sent `"\"all\""` — quotes inside the JSON string — and burned a retry turn on `toc: invalid doc-sentences value` (a1-toc-file/1, a1-toc-file/3, a3-toc-pair/2, a4-toc-pair-symbol/2, a5-bundle/3).
The schema description itself embedded escaped quotes around the sentinel (tools_toc.go:34 pre-fix), and the models copied them faithfully.
Reworded without nested quotes in commit ee84d8d.
Lesson worth generalizing: any quoted literal inside a tool description is likely to be reproduced verbatim inside JSON string values.

### 5.2 Ambiguous candidates leak 1-based positions into 0-based tools (documented; structural fix deferred)

`classifyLSPError` passes `ErrAmbiguousSymbol.Candidates` through unconverted (internal/mcpserver/result.go:98-101);
those strings are built by `lsp.FormatLocation` as 1-based display strings (internal/quarryengine/lsp/wire.go:85-92).
So `textDocument_definition`/`references` — which declare "0-based on both input and output" — hand back 1-based `file:line:col` candidates.
Observed: a5-bundle/2 and /3 got `windowsize.go:42:18` for a symbol whose true 0-based position is 41:17;
a5-bundle/3's agent hand-converted correctly, a5-bundle/2's agent abandoned the lookup.
The descriptions now state the convention and steer the retry to `textDocument+symbol` (commit ee84d8d).
Converting the strings would break CLI/MCP parity for identical queries; emitting structured 0-based candidate objects is the clean fix but is a schema change — see R6.

### 5.3 workspace_symbol emits names its sibling tools cannot consume (documented; resolution feature deferred)

workspace_symbol results name methods `Type.Method` (e.g. `Shuttle.Run`, `Engine.liveBoxLocked` — gopls convention).
The symbol inputs of every other tool require an exact bare name: `collectInFileMatches` strips gopls's `(Receiver).Method` and compares whole names (internal/quarryengine/query/refs.go:388-397 and 380-387 doc), so a qualified name never matches.
Observed: b4-lsp-trio/1's `{"symbol":"Shuttle.Run","textDocument":...}` → `not_found`.
Descriptions now warn on both sides (commit ee84d8d);
real qualified-name resolution (owner-checked, not owner-stripped — stripping would reintroduce exactly the conflation quarry exists to prevent) is R5.

### 5.4 Module-cache noise is a general LSP-results problem, not just workspace_symbol's

Besides seed 1's fuzzy hits, ambiguous-candidate lists also leak dependency paths: a5-bundle/3's project-wide `{"symbol":"planLayout"}` references call returned candidates including `/home/knatte/go/pkg/mod/github.com/%21proton%21mail/go-crypto@v1.1.6/...` (URL-encoded, out of project).
`within` cannot reach candidates (it filters successful results only).
See R6.

### 5.5 references + within is the task-04 efficiency optimum; assert and impact are close behind

The model workflow appeared twice, identically (b3-references/1 and /3): Read the interface file, one references call at the declaration with `"within":"internal/shedadapters"`, 604 chars back containing exactly the declaration plus the three true call sites, lookalike excluded by type reasoning — $0.107/$0.137 per run, at or below the no-tool baseline's cost with the conflation question answered by the server instead of by hand.
Without `within` (b3/2), the same call returned cross-package references (burlerengine, mergeresolve, shuttlecli — gopls includes structurally-related interface-method references), which confused the agent into re-running the identical call.
b5-impact answered the whole question in one 1,530-char call carrying each caller's enclosing function and signature (raw/b5-impact/1-3; b5/2 also used `within` and, at $0.099, was the second-cheapest run in the matrix).
b6's clean reps used assert_no_callers off-label as a compact caller lister (`violation:true` + 3 callers, 440-495 chars);
b6/3 was the cheapest correct run of all 42 at $0.057.
The off-label use worked because the callers list is exactly a scoped, declaration-excluded reference set — worth keeping in mind when weighing R7 (result caps) against tool-splitting proposals.

### 5.6 Whole-file Reads, not tool payloads, dominated b1's context

b1's agents read bouncer.go (32,939 chars) and burler.go (26,108 chars) in full in every rep;
b0's baseline agents answered the same question with `cat singlellm.go` plus greps, never opening those files whole.
The benchmark preamble's "do NOT reach for grep as a reflex" instruction plus a tool set with no reference capability left whole-file reading as the only verification move.
Tool-exposure decisions and prompt guidance interact; neither alone explains the cost.

### 5.7 MCP schema overhead is visible but modest

First-turn context: ~8.6k tokens for no-tool configs, ~9.3-9.7k for one-tool configs, 12.9k for the 7-tool b7 bundle — roughly 4.3k tokens of schema for the full bundle, cached and re-read every turn.
At b7's scale that is ~$0.02-0.03/run: real, not decisive.

### 5.8 The one likely-wrong answer came from trusting a reference set as a complete world

b7-bundle/1 answered `"excluded_lookalikes": []`, asserting no same-named lookalike exists in the package — but burler.go:373's `BurlerRunner.Run` call is exactly the lookalike the task demands excluded, and every other run found it.
The path: a references call on `Shuttle.Run` correctly did not contain the lookalike; the agent, following the preamble's "do not double-check with grep", never enumerated `.Run(` textually, so it never learned there was something to exclude.
references answers "who references this symbol", not "what same-named strangers exist" — the latter needs a text or symbol search.
The references description now says so explicitly (commit ee84d8d), but this is equally a prompt-design lesson for the harness preamble: forbidding grep converts a completeness question into a blind spot.
And it is the strongest argument for waiting on the scoring run before drawing tool-value conclusions.

## 6. Recommendations

Ordered by (impact x confidence) / effort.
R1-R2 are the highest-value follow-ups; the fixes already landed on this branch are not repeated here.

**R1 — Re-ingest the 2026-08-30 matrix and re-derive the summary.**
The on-disk usage.json files still carry the per-record inflated numbers (1.1), b6/2 is contaminated (1.3), and b7/1's duration includes an 82s harness stall (1.4).
Any decision made from the existing summaries inherits all three.
Effort: small (the fixed ExtractUsage is on this branch; ladderbench presumably re-ingests from `transcript.jsonl`).

**R2 — Guard the dispatch path against prompt loss.**
b6/2 ran with the dispatcher's own invocation string as its task prompt and reverse-engineered the task from a sibling rep's artifacts.
At ingest time the composed prompt is reproducible (preamble builder + task file);
ingest should compare the transcript's first user message against the expected prompt and refuse or loudly flag a mismatch.
Effort: small-medium; touches harness files the operator currently has uncommitted changes in, so I did not implement it here.

**R3 — Give toc_dir a lighter header mode rather than removing anything.**
toc_dir bought the matrix's biggest turn win (seed 6) and its biggest payloads (~47-49k chars for two directories).
Two orthogonal trims, both preserving the current default: a `docSentences`-style header-sentence cap (a 1-sentence mode would have cut the observed calls by very roughly half, given loomyard's multi-sentence headers), and/or an option to drop `test:true` files, which are ~half of each listing and were never what the task-01 agents were surveying.
Match any new option in the CLI `toc dir` verb for parity.
Do not conflate with the already-tracked `toc-dir-list-subfolders` task (subdirectory names), which is being worked separately.

**R4 — Consider exact-match-first ordering for workspace_symbol results.**
gopls's fuzzy scoring ranked module-cache subsequence matches above the repo's own `BouncerConfig` (5.4, seed 1).
A stable re-sort in `symbolFromClient` — exact name matches first, then in-TargetDir before out — would fix the worst presentation problem for CLI and MCP at once, complementing `within`.
Engine-level change with an ordering contract to define; medium effort, medium confidence (the scoring run may show the noise rarely matters).

**R5 — Qualified-name symbol resolution.**
Accept `Type.Method` in symbol inputs by matching owner and method exactly (never by stripping the owner), so workspace_symbol output round-trips into definition/references/impact.
This is a real engine feature (touches `collectInFileMatches` and the workspace-symbol disambiguation path in refs.go); the description warning landed in ee84d8d is the interim mitigation.

**R6 — Structured, convention-consistent ambiguous candidates.**
Emit candidates as `{file, line, character}` objects in each tool's own line convention (and filterable by `within`), instead of pass-through 1-based display strings (5.2, 5.4).
Schema change; coordinate with CLI parity policy before doing it.

**R7 — A results cap on the LSP-mirrored tools is not yet justified by this data.**
The big payloads were toc_dir (useful) and workspace_symbol (now scopable);
references/definition/impact/assert payloads never exceeded 1.9k chars in 31 calls.
Revisit only if scored data shows truncation-safe cases.

**R8 — Benchmark-prompt revision for the next matrix.**
Three preamble effects showed up in the data: the anti-grep rule created b7/1's blind spot (5.8) and b1's whole-file reads (5.6), and the parallel-calls rule paid for speculative 12k workspace_symbol calls in every b4 rep (seed 5).
Softening "do NOT reach for grep" into "prefer the tools for what they cover; use text search for textual-enumeration questions" would measure the tools' real value rather than instruction-following side effects.

**R9 — Line-convention convergence question for the operator.**
The 0-based LSP-mirror convention caused every observed position miss (seed 4), while the 1-based native tools caused none.
The descriptions now steer toward symbol addressing, which sidesteps the issue; actually flipping the LSP tools to 1-based would be a breaking semantics change and deliberately mirrors LSP, so I only note the evidence.

## 7. Method notes

Turn/cost/context numbers come from per-message re-aggregation of all 42 `transcript.jsonl` files (grouping assistant records by `message.id`, taking each call's final usage snapshot, max for output), cross-checked against usage.json for the fields that agreed.
Payload sizes are the character lengths of every tool_result block, matched to tool_use ids.
The extraction tooling lived in the session scratchpad and is not committed; the corrected per-run table is reproducible from the fixed ExtractUsage plus the raw transcripts.
Every transcript was read in full in condensed form (all prompts, assistant text, tool inputs, result sizes, and result heads), with raw-transcript spot checks where condensation was ambiguous (b6/2's prompt, a2/1's usage records, b7/1's timestamps).
