# HANDOFF

## State as of 2026-09-03 evening — the rewrite

**T0 is done: `main` holds only the Go tree-sitter extractor.** V1 is frozen on branch `v1-final`
(worktree `wts/v1-final` on the machine that ran it; on any other machine, `git worktree add
wts/v1-final v1-final`). The deletion was squash-merged as `a18d490`; the six-commit branch history
is on tag `archive/delete-v1`.

What `main` contains now:

- `internal/quarryengine` — the cgo guard pair, a short `doc.go`, `ErrLanguageUnsupported`.
- `internal/quarryengine/toc` — the Go strategy, the shared helpers, the extension table (`.go` only).
- `internal/quarryengine/treesitter` — the Go grammar only. `go.mod` requires `go-tree-sitter` and
  `tree-sitter-go`, nothing else.
- `docs/rewrite-plan.md` (the plan; §12 is the task table) and `docs/glyph.md` (the identifier
  contract). Both are current as of `d437cfb`.
- `bench/loomyard-eval/ladder/ladder*.yaml` and `bench/loomyard-eval/tasks/` (prompts and
  `*.fasit.json`) — the inputs the new harness (T2) and the first ladder run (T7) need.
- `docs/research/**` and this file — the measurement record. `CLAUDE.md` is one line: Go, no Python.

Decisions taken 2026-09-03, all written into the plan and the spec:

- **Go only.** Other languages are added one at a time, when wanted, with extractors written against
  the glyph contract; the V1 Python and C# extractors are on `v1-final` as reference only.
- **The listing verb is `toc`**, not `map`: it is a table of contents, and `map` is a keyword in Go.
- **T6 MCP is thin**: one `toc` tool, nothing more until a ladder cell measures more.
- Python/C# and Loomyard's adoption of glyphs are **not tasks** in this repository's §12.
- **No Python** in this repository, no exceptions. The V1 results roots and `gen_compact_toc.py` are
  on `v1-final`; their raw transcripts were never committed and sit outside the repository on the
  WSL2 host (`v1-raw-results/` beside the worktrees).

**Next.** The mill wiki backlog holds `glyph-package` (T1) and `ladder-harness` (T2), wave 1, each
with a proposal, unclaimed. Spawn them with `mill-spawn` when ready; T2 needs `.scratch/ladder.env`
with `LADDER_LOOMYARD_REPO` in its worktree. For mechanical tasks later (T5, T6) use `mill-quick`
rather than the full pipeline; T0 showed the review rounds are overhead for a deletion.

Millhouse notes: the wiki is a daemon-backed store that renders `Home.md` and the proposal files —
edit tasks through the `millpy-*` wrappers or `wiki._client`, never the files. `mill-merge` strips
`_mill/` before squashing; a manual squash must `git rm -r _mill` first.

## The V1 record (2026-09-02 and before)

> Everything below is the V1 record. Its measurements and rules still hold and the rewrite plan cites
> them, but the "next step" items in §3–§6 are replaced by `docs/rewrite-plan.md` §9 and §12.
> The results roots, the harness and its README it refers to are on `v1-final`
> (`bench/loomyard-eval/ladder/results/<root>/conclusion.md` there).

The research report this descends from is `docs/research/quarry-improvement-research.md`. Every
results root named below has a `conclusion.md` that is the record of that run; read those before
re-deriving anything from `summary.json`.

## 0. DONE: the working tree is committed

Committed 2026-09-02 as `40206e7`. Keep the rule it existed for: commit before running anything,
because the matrix builds the server under test from the working tree and a dirty tree is the single
biggest source of invalid runs (see §2, rule 1). What that commit contained:

1. Engine + CLI + MCP: compact toc form — `internal/quarryengine/toc/compact.go` (+test),
   `quarry/facade.go` (`CompactTOCFile/Dir`), `internal/cli/toc.go` (`--compact`),
   `internal/mcpserver/{mcpserver,tools_toc}.go` (`compact` input, `--toc-format`),
   `cmd/quarry-mcp/main.go`.
2. Harness: `run.sh` + `run-*.sh` one-command entry points, `tools/runmatrix --all` with
   `provenance.json` and per-rep `server_hashes`, worktree `prune` before `add`, `GateRunPrompt`,
   annex support (`internal/ladder/annex.go`, `annex:` config field, `annexes:` task recipes),
   `toc_format:` config field, `SKILL.md` outcome-marker rule, README updates, `.gitignore`
   `/ladderbench`.
3. Ladders and results: `ladder-toc.yaml`, `ladder-compact.yaml`, `ladder-annex.yaml`,
   `results/2026-09-01-followup/conclusion.md`, `results/2026-09-02-followup/`,
   `results/2026-09-02-toc/`.

`results/**/raw/` is gitignored; the transcripts stay on the machine that ran them. The 2026-09-01
followup conclusion argues for un-ignoring it (1.8 MB per 12 runs); not done yet.

## 1. What has been established (read the conclusions for detail)

| results root | what | verdict |
|---|---|---|
| `2026-08-30` | 45-run main matrix, tasks 01 and 04 | only toc_dir separated from control; every LSP rung flat or worse |
| `2026-09-01-task05` | task 05, c0–c3 | no separation; `output_tokens` broken on WSL2 host |
| `2026-09-01-followup` | fix verification, invalid | b1 never used its tool; one b4 row was an August transcript |
| `2026-09-02-followup` | fix verification, valid | `within` scoping works; description steering does not; b4 lost recall once from tool-surface friction |
| `2026-09-02-toc` | toc-only ladder, reps 5 | **toc_dir reproduces: turns 8→4, cache_read 127k→83k, recall unchanged.** toc_file alone never helps. Named-file task: no toc gain. Three reps excluded (harness incidents, §2) |
| `2026-09-02-compact` | compact ladder, two aborted starts | no data; `ABANDONED.md` says why the root and its stale lock are left in place |
| `2026-09-02-compact2` | compact ladder, reps 2 | compact delivers the bytes (49k→13k, cache_creation 57k→29k) but a2's turn win does not reproduce under it and precision drops 0.96→0.82. **The form as built is rejected; the default flip is dropped, not deferred.** n=2 settles it because the claim was that the form is free (§3) |

The one-line summary of the whole programme so far: on Sonnet-class agents, quarry's LSP tools do
not beat grep on correctness or cost; a directory table of contents halves exploration cost on
unfamiliar code without changing the answer. Everything below is about making that one win cheaper
and deciding where it ships.

## 2. Rules learned the hard way (each one cost a run)

1. **Do not edit quarry source while a matrix is running.** `prepare-session` rebuilds `quarry-mcp`
   from the working tree before *every repetition*. On 2026-09-02 the compact-toc code landed mid-run;
   every later session got a server advertising `compact: true`, and two agents used it. Now recorded:
   `provenance.json` carries `server_hashes` per rep, the final table prints `!! the quarry-mcp binary
   changed during this root` when more than one build was used, and `run.sh` warns when the tree is
   dirty. The rule still has to be obeyed by the operator; the harness only catches the violation.
2. **The run agent's prompt is now gated.** An orchestrator once dispatched the subagent with the
   dispatch description as its whole prompt (a1-toc-file rep 4). `GateRunPrompt` fails ingest unless
   the first user message opens with `PARALLEL_OPENING` and contains the task text. A rep that fails
   it is re-attempted by the loop; it is never scored.
3. **The outcome marker is written with Bash, never the Write tool.** `Write` is not on the session's
   allow list; a Write call stops on a permission prompt nobody is watching and the matrix waits
   behind it. `SKILL.md` says so now.
4. **`/tmp` worktrees vanish on reboot but stay registered.** `BuildWorktree` prunes before adding.
   Nothing to do; noted so nobody "fixes" it back.
5. **`output_tokens` is unusable on the WSL2 host (OSL-1033).** Claude Code 2.1.258 transcripts carry
   only streaming snapshots. Use `cache_read_input_tokens`, `cache_creation_input_tokens`, and turns.
6. **Cost numbers are per results root.** Never compare `duration_ms` or tokens across roots or
   hosts; correctness (recall/precision) may be compared by config id across roots.
7. **One tmux session at a time.** `run.sh` refuses to start while a `ladder-run` tmux session
   exists. If a previous run was interrupted: `tmux kill-session -t ladder-run`, then re-run; the
   driver resumes from what the results root already records.

## 3. CLOSED: the compact ladder ran, and the compact form is rejected

**Ran 2026-09-02 at reps 2: `results/2026-09-02-compact2/` — read its `conclusion.md`.** The compact form
cuts the bytes exactly as designed (49 KB → 13 KB, `cache_creation` 57k → 29k), but a2's turn halving does
not reproduce under it (5.5 [4–7], no longer separated from the control) and precision falls from 0.96 to
0.82, below the control's 0.935. Read volume went *down*, not up: the agent answered from the thinner map
rather than compensating for it.

**Two repetitions settle it, and there is no reps-5 re-run to do.** The claim under test was that the form
is *free* — same behaviour, fewer bytes. A null-effect claim needs only enough resolution to see the effect
it claims to preserve, and this root has it: a2 reproduced the established turn halving at n=2. a9 did not,
and precision went 2/2 the wrong way, below every a2 and every control observation.

So **the default flip below is dropped, not deferred**, and §6 item 7 (compact impact/refs) loses its
premise with it. What is rejected is *this* form — one sentence per file, `leadMaxRunes` 120. A less
aggressive compact form would be a new form and a new cell (`a10-toc-dir-compact-wide` against a9), not a
re-run of this ladder; nothing in the programme currently depends on it. **Next real work is §6, headed by
6.1.**

The rest of this section is kept as the operating manual for `run.sh` and the per-root checks, which every
future ladder still needs.

**Question (as it stood).** The toc_dir win is paid as a 49 KB JSON tool result per run. The compact form
(`quarry toc dir --compact`, MCP `compact: true`, forced per session by `--toc-format`) renders the
same map — same paths, same line numbers — as plain text at a third of the bytes (10.9 KB → 3.1 KB
for a 25-file package; ~12 KB for the task-01 scope). Does the agent do the same thing with it?

**Ladder.** `bench/loomyard-eval/ladder/ladder-compact.yaml`: task 01, reps 5, three cells —
`a0-none`, `a2-toc-dir` (JSON forced), `a9-toc-dir-compact` (compact forced). Tool delivery only;
annex delivery is deferred (§5). The ids match the toc ladder's so the two roots line up.

**Prerequisites on the machine** (all checked by `run.sh` before anything starts):
- `go`, a C toolchain (`cc`, for tree-sitter under CGO), `claude` (Claude Code CLI, logged in),
  `tmux`, `git`; `gopls` is not needed for this ladder (toc is tree-sitter only).
- A Loomyard checkout containing commit `975578cda8d6f3a81580bd4e73725e060211b766`. Point the
  harness at it with **one** gitignored file, `<repo-root>/.scratch/ladder.env`, containing
  `LADDER_LOOMYARD_REPO=/abs/path/to/loomyard` (or export the variable). No tracked file may carry a
  machine path.
- A clean `git status` (rule 1).

**Run** (from anywhere; it resolves the repo root itself):

```
bench/loomyard-eval/ladder/run-compact.sh
```

It does everything: preflight, builds `ladderbench` and `quarry-mcp`, creates or restores the task
worktree at `/tmp/loomyard-eval-01`, then for each cell and rep: `prepare-session` → `warm` → a live
`claude` session in tmux session `ladder-run` running `/ladder-run` → `ingest`. Then the scoring
session, `summarize`, `provenance.json`, and a per-cell table. Watch with
`tmux attach -t ladder-run`. Expect ~1 minute per run on the WSL2 host plus a scoring session: about
25 minutes for 15 runs. It is resumable: re-run the same command and it skips what the root already
has. Results land in `bench/loomyard-eval/ladder/results/<today>-compact/`.

**Before reading any number**, check the root:
- `provenance.json`: `quarry_dirty` false (or only the results files), one distinct value in
  `server_hashes`. **Do not check `server_vcs_modified`** — it is structurally `"true"` on every run,
  because `runmatrix` creates the untracked results root before the binary is built; see the harness
  note in `results/2026-09-02-compact2/conclusion.md`. `server_hashes` is the real evidence that the
  binary did not change mid-run.
- The printed table has no `!!` lines (an unused tool, or a changed server binary).
- `raw/*/*/ingest.json` has no fatal findings; a rep that failed `run_prompt` was re-attempted.
- Spot-check one `a9` transcript: the `toc_dir` result starts with `{"results":[{"compact":"# ` and is
  ~12 KB; one `a2` transcript: the result carries `"files":[` and is ~49 KB.

**Then write `results/<today>-compact/conclusion.md`** in the shape of
`results/2026-09-02-toc/conclusion.md`: provenance line, exclusions if any, the per-cell table
(median [min–max] of turns, greps, cache_read, cache_creation, duration, recall, precision, plus Read
chars and tool chars from the transcripts), and the reading. The reading is decided by a9 vs a2:
- cache_creation and cache_read down, turns still 4, recall still ~0.42 → the form is free; adopt it
  (next bullet).
- turns or Read volume up → the one-sentence-per-file cap (120 chars) cost the agent information it
  used; try `leadMaxRunes` higher or `--doc-sentences`-style knobs before concluding.
- recall down → same; do not adopt until understood.

**~~If compact holds, flip the defaults~~ — DROPPED, it did not hold. Kept for the record only; do not
do any of this:**
- MCP server: default `compact` for both toc tools, JSON on `compact: false`. The only reader of MCP
  output is an LLM. Update the two tool descriptions to describe the compact lines.
- CLI: keep `toc dir|file` JSON as the scripting contract; add a `quarry map <path…>` verb that
  takes files and directories alike and prints the compact form. Then point `docs/when-to-use-quarry.md`
  at `map` as the recommended first call on unfamiliar code.
- Optionally re-run `run-toc.sh` afterwards against a fresh root if a clean n=5 JSON-vs-compact baseline
  under the new defaults is wanted; not required.

## 4. Open follow-ups from the toc run (cheap, do when convenient)

- `results/2026-09-02-toc`: reps a1-toc-file 4 and 5 and a2-toc-dir 3 are excluded in the conclusion
  but still in `summary.json`. To get a clean n=5: `ladderbench invalidate --config-id <id> --rep <n>
  --ladder bench/loomyard-eval/ladder/ladder-toc.yaml --results-root <root>` for each, then
  `run-toc.sh` with that root as its optional second argument re-runs only those three and
  re-summarizes. The conclusions do not depend on it.
- Un-ignore `results/**/raw/` so transcripts travel with the repo (see the 2026-09-01-followup
  conclusion for why).
- The orchestrator session (the `/ladder-run` skill runner) spends 10–20 tool calls reading harness
  source before dispatching, every rep. It does not touch the measured subagent, only wall-clock;
  tightening `SKILL.md`'s opening ("do not read harness source; the commands below are complete")
  would cut a minute per rep.

## 5. Deferred, with the reason

- **Annex ladder (`ladder-annex.yaml`, `run-annex.sh`) — built, tested, dry-run, not to be run now.**
  Injecting pre-computed quarry output into the prompt presumes (1) the caller already knows exactly
  what the agent is going to do, and (2) a non-interactive dispatch. That is the Loomyard shape, not
  the current one; run it when the Loomyard integration (§7) is the next thing being built.

  **2026-09-02: the annex recipes are confirmed correctly shaped for their target.** Loomyard's plans
  name *symbols*, which is what `plan-pack` and `impact` take (`symbol` + `in_file`). `b10-annex-impact`
  and `b11-annex-plan` are therefore feedable from a real Loomyard plan as designed — that was an
  assumption when the ladder was written, and it holds.

  **Testing this in Millhouse was considered and dropped.** Mill plans name files, not symbols, so
  neither `plan-pack` nor `impact` can be fed from one. The only pack a file list supports is
  `quarry toc file <files>`, which is `a1-toc-file` shaped — the retired rung, which never separated
  from control and actively hurt in `2026-09-02-toc` (recall 0.35 vs 0.43, precision 0.86 vs 0.94).
  Quarry was built for Loomyard's shape; measuring it against Millhouse's would test the wrong pack.
  Do not re-propose it.

  The harness support stays: `annex:` on a config, `annexes:` recipes on a task (`toc-dir`, `toc-full`,
  `toc-file`, `impact`, `plan-pack`, `compact: true`, `drop_callers: N`), generated by
  `prepare-session` from the quarry CLI, injected by `next-run` as one neutral paragraph, copied at
  ingest as `annex.txt` + `annex.meta.json`. The degraded-annex cell (one caller dropped) is the
  measurement behind the "never inject unverified" rule in §6 item 2 (per-entry `verified`).
- **Weak-model arm (Haiku)** — dropped. Operator does not use Haiku (too weak for reasoning and for
  fixing failing tests), so the deployment question it answers does not arise.
- **Tasks 06–08** (interface-method impact at scale, whole-repo dispersion, structurally invisible
  references) — designs in the research report §4; build only if an LSP-shaped question comes back.
  The toc results do not motivate them.
- **toc_file as a rung** — retired; it never separated from control in either ladder.
- **Review-time injection** — negative prior stands (`bench/loomyard-eval/scripts/gen_compact_toc.py`
  experiment; task-03 scorecard); not re-tested.

## 6. What quarry is actually for — the three-way split (2026-09-02)

Established by measurement on 2026-09-02, working through what a Loomyard plan would need. This
supersedes the flat ranking that used to open this section: it says *which* engine work is worth doing
and why the rest was never going to pay.

| the question | who wins | why |
|---|---|---|
| where is symbol X | **grep** | one scoped grep, 1 hit, 5 ms — Go declarations have a fixed shape (`func (d dualHandler) stderr(`), and a plan already names the package, so the haystack is one directory |
| where does X **begin and end** | **quarry `toc file`** | the boundaries are syntactic, not textual; grep cannot produce them and guessing fails silently |
| what **calls** X | **quarry `impact`** | interface dispatch leaves no textual trace; no amount of grepping can prove it |

**Row 1 explains every negative result in the programme.** `quarry definition --in-file logger.go stderr`
returns `{"file":…,"line":206,"character":22}` — a bare position. That is the same line grep finds in
5 ms, and it costs a gopls daemon to produce. `b2-definition` never separated from the control in any
run because it returns what grep already returns. The same reasoning covers `workspace_symbol` and
`references` on a symbol the caller can already name.

**Row 2 is the unexploited primitive, and it is the one thing here that is cheap, measured, and
LSP-free.** `toc file` emits `start` / `sigend` / `end` per symbol. The reason it cannot be replaced by
`grep -A N`: **the docstring precedes the signature**, so a grep hit lands in the middle of the symbol,
not at its start. Measured over 1741 symbols in 400 non-test Loomyard files:

```
docstring lines above the func line : p50=3   p90=12   p99=29   max=52
total span (start..end)             : p50=14  p90=52   p99=145  max=971
```

A fixed window misses in both directions, and — the part that matters — **truncation is silent**. The
agent gets half a docstring or half a body with no signal that anything is missing. That is precisely
the failure mode the compact ladder measured (§3): confident answers off an incomplete map, precision
0.96 → 0.82. The alternatives without quarry are read-the-whole-file (413 lines for a 6-line symbol) or
guess-a-window (silently wrong). Exact spans remove the choice.

Cost of producing them, cold binary, tree-sitter only, no daemon, no gopls:

```
one package (11 files)        19 ms
whole repo (439 non-test .go) 318 ms
```

### Ranked work, after the split

1. **Python (and C#) nested types are dropped by `toc`.** A fixture with `class Beta:` containing
   `class Inner:` containing `handle` emits `Beta` and `Beta.handle` and nothing else — `Inner` and its
   method are absent, and `Owner` carries only one level, so `Beta.Inner.handle` cannot be expressed at
   all. This is the only *extraction* loss found, and extraction loss is unrecoverable without
   re-parsing. Go is unaffected (no nested types).
2. Per-entry `verified: true|false|"partial"` on refs/impact — prerequisite for any mechanical gate,
   and for any injected pack an agent is expected to trust rather than re-check (§5). Unchanged in
   rank; the trust argument in §5 makes it the precondition for the whole annex direction, not item 4
   on a list.
3. `within` scoping default-on (workspace-wide becomes the explicit opt-out). Verified working
   2026-09-02; both agents had to know to pass it.
4. ~~`rename_impact` tool wrapping LSP `textDocument/rename`~~ — **shipped upstream.** `gopls mcp`
   exposes `go_rename_symbol`, taking `file` + `symbol` + `new_name` (§8). Building it here would be
   duplication. It also inherited row 3's open question — does anyone act on it without verification.
5. Expose `implementations` (`textDocument/implementation` is already wired internally). Row 3.
6. 1-based positions at the MCP boundary and lenient input. **But not "`Type.Method` qualifiers
   stripped"** — that was backwards. The receiver is half the key, not noise: `(package, receiver type,
   name)` is what makes a Go method unique. It must be interpreted, never dropped.
7. ~~Compact output for impact/refs~~ — **dropped**, see §3.
8. Warm-daemon support beyond Go — **deprioritised by the split.** Rows 1 and 2 need no daemon at all,
   and row 3 is the part that has never paid. Document the other languages as cold-spawn-only.

### Already true, do not "fix" it

- **`Symbol.Owner` works.** It carries the enclosing type's bare name for a method, in Go *and* Python:
  `Handle owner=dualHandler` and `Handle owner=durableHandler` are fully distinguished, as are Python's
  `handle owner=Alpha` / `owner=Beta` / module-level `owner=""`. A qualified name of the form
  `package#Type.Method` therefore resolves against today's shipped binary with no engine change. (Two
  earlier readings in this session said otherwise; both were probes querying a nonexistent `receiver`
  key.)

### The design rule the compact result taught

**Extraction is complete; the view filters; no view is ever the only one on offer.**

`Symbol`'s own docstrings already state this — `Signature` is "verbatim source text … never
reformatted, never truncated", and `Owner` exists precisely so "a caller composes the qualified form
itself … when it needs one". The rule is not new; it needs enforcing where it was broken.

It was broken in exactly one place, and that place is the one thing measured to do harm: the compact
form's 120-rune lead cap and one-sentence-per-file, made the *only* option by `--toc-format` forcing it
per session. `--doc-sentences` is lossy too but sits at the entry point rather than in extraction,
which is the right layer; its default should still be "everything". Extraction cost is 318 ms for the
whole repository, so no performance argument justifies dropping anything during extraction — and when
the consumer is Go code rather than an LLM, filtering is free.
## 7. Loomyard mechanical integration (later)

The direction the evidence supports: quarry called from Loomyard's own Go code via the `quarry/`
facade, never by the LLM — a toc pack for unfamiliar-code exploration (the measured win), a plan-time
impact/plan pack for "use X and Y to edit Z" implementers (the annex ladder's b11 cell measures it),
and deterministic gates on deletes/moves/renames (`assert-no-callers`, before/after `impact` sets).
Blocked on §6 item 2 (per-entry `verified`): a gate that consumes unverified results marked `resolution: complete` recreates
the 31-false-positive incident (`docs/research/scout-agent-usage-findings.md`). Measured by the annex
ladder when the time comes.

## 8. The MCP surface — gopls now ships its own (2026-09-02)

Full write-up and the captured outputs: `docs/research/mcp-surface.md` and
`docs/research/output-formats/`. Read that before touching the MCP layer. The short version:

- **`gopls mcp` exists**, in the same binary quarry already spawns. Eight tools, four of which have no
  LSP method at all. **Not one takes a position** — every symbol-addressing tool takes `file` +
  a possibly-qualified `symbol`. `go_rename_symbol` is §6's old `rename_impact`, shipped upstream.
- **It returns prose, not JSON, and carries no line numbers.** So §6's row 2 — where a symbol begins
  and ends — is still quarry's alone, at 3–9× smaller output.
- **The thin-wrapper layer is the layer that never measured well**, and `output-formats/symbol.txt`
  shows why: undecoded `SymbolKind` integers, three naming conventions in one array, unlabelled fuzzy
  matches. All faithful to LSP — which is the problem. **LSP assumes the client is an editor**; every
  awkward thing there is an editor affordance with the editor removed.
- **quarry sends every MCP payload twice** (`structuredContent` plus a serialized mirror in
  `content[].text`), but **this costs no context**: verified against a ladder transcript, the agent
  receives one string block. Transport-only, over a local pipe. Do not spend effort on it.
- **JSON is not the MCP payload standard.** The envelope is JSON-RPC 2.0; `content[].text` is an
  opaque string and gopls puts markdown in it. This is a free choice — but it does not rescue the
  compact form, because the model reads `content`, which is exactly the trade-off §3 measured.

**Recommendation, seven MCP tools to four:** keep `toc_file`, `toc_dir`, `impact`,
`assert_no_callers`; drop `textDocument_definition` (a position grep already found),
`textDocument_references` (superseded by `impact`), `workspace_symbol` (fuzzy, undecoded, noisy). The
CLI keeps all seven verbs — different surface, different consumers.

**What gopls MCP does not threaten:** it competes with one of quarry's three surfaces for one of its
three focus languages. The Cobra CLI has no MCP equivalent (`assert-no-callers` is an exit-code
contract for a plan card's `verify:` step), the `quarry/` facade has no MCP equivalent and is the whole
basis of §7, and gopls MCP is Go-only.

**Parked, operator to decide:** whether quarry should be stripped to its tree-sitter half, possibly
folded into Loomyard. Note the layering before deciding — `impact` is built *on* toc
(`enclosingSymbol(fileTOC.Symbols, ref.Line)`, and a target's provenance is "always that one
`toc.Symbol` — never an LSP candidate"), so toc is the bottom layer and LSP sits on top of it.
Stripping means removing the top, not carving out the bottom. quarry was also deliberately extracted
*out* of Loomyard into its own module; reversing that should be a decision, not a side effect.

Scope note: the focus is **three** languages — Go, Python, C# — not the five the engine doc lists.

## Dependency order

```
3 (compact ladder) ──► CLOSED: form rejected; 3's default flip and 6.7 dropped with it
6.1–6.6               — the remaining work; 6.1 is next, 6.2 gates 7
4 (toc-run tidy-ups)  — any time
5 (annex run)         — only when 7 is being built
```
