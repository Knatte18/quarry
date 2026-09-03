# Discussion: Ladder harness around headless claude -p (T2)

```yaml
task: Ladder harness around headless claude -p (T2)
slug: ladder-harness
status: discussing
parent: main
```

## Problem

The benchmark harness that produced every number the quarry rewrite rests on was 17 000 lines
(9 000 non-test) and was deleted in T0. Its size was architectural, not incidental: it drove an
*interactive* Claude Code session in tmux, which ran a skill, which dispatched a subagent, and then
reconstructed what had happened by hunting for the subagent's transcript under `~/.claude/projects`.
Every gate and rule it carried existed because the harness did not control the run — the prompt
gate, the outcome marker, the one-tmux-session rule, the transcript hunt, the static per-cell agent
definitions, the post-hoc turn ceiling, the unusable `output_tokens`.

Headless `claude -p` gives a harness direct control of the prompt, the MCP server, the tool
allowlist, the turn ceiling, the transcript and the usage numbers. **Why now:** T7 — the first
ladder run against the rewritten engine — is the regression gate for the whole rewrite, and it has
nothing to run with. T2 is wave 1's long pole and needs no quarry code until T7, so it starts now
and overlaps waves 2–5.

This task builds that replacement: one Go program of roughly 1 000–1 500 non-test lines, with tests
besides (see Constraints), under
`bench/loomyard-eval/ladder/`, replacing `cmd/ladderbench`, `internal/ladder`, `tools/runmatrix`,
`run*.sh`, `launch-session.sh` and `.claude/skills/ladder-run`.

## Scope

**In:**

- One Go program under `bench/loomyard-eval/ladder/`: `cmd/ladder/` (thin `main`) plus
  `internal/ladder/` (one package, files per concern).
- A yaml loader for the kept ladder shape, with every V1 coupling removed.
- Per cell and repetition: restore the task worktree at the pinned commit, write the per-cell MCP
  config, run `claude -p` with the fixed flag set, tee `stream-json` to
  `results/<root>/raw/<cell>/<rep>/transcript.jsonl`.
- Metrics computed from the stream's assistant records, never from the final `modelUsage`.
- The scorer: a second `claude -p` call, Opus against the fasit, JSON out, with the answer redacted
  so the scorer cannot identify the rung.
- `summary.json`, `provenance.json`, and the per-cell table — printed to stdout **and** written to
  `results/<root>/table.txt` by both `run` and `report`.
- A single-run advisory lock, so two `ladder run` invocations cannot share a worktree root.
- Resume, keyed on a per-rep `run.json` written last.
- The two surviving gates: the granted tool was used (per cell, warning), and the control cells'
  blinding check (per rep, fatal).
- Migration of `ladder-toc.yaml` to the new shape; deletion of the five other `ladder*.yaml`.
- Recovery of the exploration output schema into `01-reed-geometry-exploration.md`.
- A `.gitignore` rule for `results/**/raw/`.
- One new root-module dependency, `gopkg.in/yaml.v3` (see module-and-yaml-dependency).
- Tests: a stub MCP server the test provides, a fake `claude` binary, and an env-guarded live smoke
  test.

**Out:**

- Any quarry code. `cmd/quarry-mcp` does not exist until T6; the harness must run a control-only
  matrix with no server at all, which is exactly what its own done-criterion (`a0-none`) requires.
- Annexes (`ladder-annex.yaml`, V1's `annex.go`, 562 lines) — needs engine features that do not
  exist.
- Cold cells, daemon warm-up, `workspace_symbol` warming — there is no daemon in phase 1
  (plan §10).
- `toc_format` / compact-view cells — needs a server flag that does not exist.
- Per-cell prompt text (the "grep-toc control" `ladder-toc.yaml`'s own comment describes). Not in
  §9a; not in this task.
- Statistical machinery. Cell comparison stays V1's `RangesDisjoint` (non-overlapping min–max), no
  significance testing.
- Running the actual T7 matrix. T2 proves one `reps: 1` run of `a0-none` end to end.
- Fixing `README.md`'s stale mention of a `map` verb (repo-level, not this task).

## Decisions

### package-layout

- Decision: `bench/loomyard-eval/ladder/cmd/ladder/main.go` (thin) plus
  `bench/loomyard-eval/ladder/internal/ladder/` — one package, files per concern:
  `config.go` (yaml shape + validation), `worktree.go` (pinned worktree lifecycle), `mcp.go`
  (server build + per-cell MCP config document), `prompt.go` (preamble + task-text extraction +
  schema), `run.go` (the per-rep loop, resume, failure handling), `stream.go` (stream-json record
  types + parsing), `metrics.go`, `score.go`, `gates.go`, `provenance.go`, `summarize.go`,
  `report.go` (the table).
- Rationale: the same shape V1 proved; keeps `ladder*.yaml` at their current tracked paths;
  matches the plan's stated rule for T3 ("one package, files per concern, never a package per
  verb"); internals stay unit-testable without exporting them from `main`.
- Rejected: a single `package main` (yaml and Go sources share a directory, nothing importable by
  tests); several packages by concern (the "package per verb" shape the plan rejects).

### results-raw-ignored

- Decision: `results/**/raw/` is **gitignored**. Committed per root:
  `results/<root>/summary.json`, `results/<root>/provenance.json`, `results/<root>/table.txt` and
  `results/<root>/conclusion.md`. Ignored: `raw/<cell>/<rep>/{transcript.jsonl, answer.json,
  answer.redacted.json, usage.json, score.json, run.json}`.
  **Producers:** `summary.json`, `provenance.json` and `table.txt` are written by the harness
  (`run`, and `table.txt`/`summary.json` again by `report`). `conclusion.md` is **hand-written** by
  whoever analyses the root — it is T7's deliverable per plan §12, not a harness output, and the
  harness neither creates nor templates it.
- Rationale: this settles plan §11's open decision. Transcripts contain absolute host paths, and
  `HANDOFF.md` §1 states that no tracked file may carry a machine path — committing them
  would require V1's whole `redact.go` machinery back as a correctness-critical gate. The derived
  artifacts plus `provenance.json`'s hashes carry the claim; the raw evidence stays on the host
  that produced it, which is where the operator who wrote the conclusion can still read it.
- Rejected: committing everything redacted (resurrects the redaction gate, adds megabytes per
  root); committing raw gzipped (same machine-path problem, compressed).

### no-tmp-paths

- Decision: the harness derives every filesystem location itself, and **task worktrees live outside
  the quarry repository, at a path containing no `quarry` token**:
  `${LADDER_WORKTREE_ROOT:-${XDG_CACHE_HOME:-$HOME/.cache}/ladder-eval}/worktrees/<task-id>`. The
  harness's own scratch — generated MCP config documents, rendered prompts, the built server binary
  — stays under `<quarry-repo>/.scratch/ladder/`. Nothing is written to `/tmp`. At startup the
  harness **asserts two invariants** about the resolved worktree root and refuses to run if either
  fails, naming `LADDER_WORKTREE_ROOT` as the override: (i) it is not under the quarry repo root;
  (ii) it does not contain the case-insensitive substring `quarry`. The yaml keys `worktree:`,
  `session_dir_template:`, `cold:`, `warm_counterpart:`, `cold_worktree_template:`, `annex:`,
  `annexes:` and `toc_format:` are removed from the shape, and the loader **errors** with a named
  message when it encounters any of them.
- Rationale: `/tmp` is banned by this task's constraints, but the worktree path also has to survive
  the blinding gate, which scans the whole marshalled transcript — and the cell's own `cwd` is in
  that transcript, in the `system.init` record and in absolute paths inside tool blocks. Assertion
  (i) exists because check (b) matches the quarry repo root: a worktree under
  `<quarry-repo>/.scratch/…` would make every control rep fail fatally, including this task's own
  `a0-none` done-criterion. Assertion (ii) exists because check (c) matches a case-insensitive bare
  `quarry` outside a `tool_result`, so a worktree root named `quarry-ladder` would fail exactly the
  same way — the token has to be absent from the path, not just the repo root. V1's gate was sound
  only because its worktrees were at `/tmp/loomyard-eval-01`, which trips neither check. `~/.cache`
  is not `/tmp` (stable, per-user, not world-writable), holds no tracked file, and is resolved at
  runtime so no machine path is committed.
- Rejected: `<quarry-repo>/.scratch/ladder-worktrees/` and `…/quarry-ladder/…` (each trips one of
  the two checks above); whitelisting the worktree prefix inside checks (b)/(c) (the gate's whole
  job is to notice quarry strings in a blinded transcript, and carving out the one path that is
  guaranteed to appear in every transcript makes the check nearly unfalsifiable while adding a
  correctness-critical special case); keeping `worktree:` and validating it (a field whose only
  remaining job is to be rejected); silently ignoring unknown keys (hides drift).
- **What actually reaches the transcript, probed.** The two assertions cover the worktree root, but
  the harness also passes `--mcp-config <quarry-repo>/.scratch/ladder/…json` into every cell,
  including controls — a path carrying both the repo root and a `quarry` token. Whether that
  argument reaches the stream was measured, not assumed: a `claude -p` run from
  `~/.cache/ladder-eval/worktrees/probe` with `--mcp-config` pointing at a file under the quarry
  repo produced a transcript containing **zero** case-insensitive `quarry` occurrences. The
  invocation's own argv is not echoed anywhere in `stream-json`. The only two fields that carried
  the token when cwd was inside the quarry repo were `system.init`'s `cwd` and `memory_paths`, and
  both are derived from cwd — so relocating the worktree clears both:
  `memory_paths.auto` became `…/projects/-home-knatte--cache-ladder-eval-worktrees-probe/memory/`.
  The harness's own scratch may therefore stay under `<quarry-repo>/.scratch/ladder/`.
- **A second reason the relocation matters, beyond the gate.** `memory_paths.auto` is the
  operator's per-project auto-memory directory, keyed on cwd. With cwd inside the quarry repo it
  resolved to the *quarry project's* memory — notes written while working on this very benchmark.
  A fresh worktree at the relocated path gets its own project key and therefore an empty memory
  directory, so no operator notes about quarry can reach a blinded cell. This is blinding in
  substance, not just gate arithmetic.
- **Correction to an earlier draft of this decision:** a previous revision argued that a worktree
  inside the quarry repo also let a blinded agent holding bare `Bash` read quarry's own source via
  `cat ../../../internal/…`. That is **false**, and it was probed rather than assumed after the
  claim was challenged: `cat ../../CLAUDE.md` from a cwd inside this repo was blocked with
  `"For security, Claude Code may only concatenate files from the allowed working directories for
  this session"` and recorded in the `result` record's `permission_denials`. Bash is confined to
  the session's working directory. The false-positive reasoning above stands on its own and is the
  sole reason for relocating.
- Note on the brief's wording: the task brief says "scratch under `.scratch/`". That is honoured
  for the harness's own scratch. Task worktrees are the deliberate exception, for the blinding
  reason above; they are a checkout of *Loomyard*, not quarry scratch.
- Note on directory trust, **not probed**: V1's `/tmp` lesson was specifically that Claude Code
  treats a fresh, untrusted directory as one where `permissions.allow` **from `settings.json`** is
  silently ignored (deny rules still apply). This design never uses a settings file — it passes
  `--setting-sources ""` and puts the whole tool grant on the command line via `--tools` and
  `--allowedTools` — so the mechanism that failed for V1 is not in play. That is reasoning, not a
  measurement, and a freshly created worktree in a new location is exactly the case V1 warned
  about. It is therefore verified rather than assumed: the live smoke test asserts the
  `system.init` record's `tools` list is exactly `["Bash","Glob","Grep","Read"]` (plus any granted
  `mcp__*` names) when run from inside a **freshly created** ladder worktree. If that assertion
  fails, the tool grant is being silently degraded and every metric from such a run is void.

### one-ladder-file

- Decision: `ladder-toc.yaml` is migrated to the new shape. `ladder.yaml`, `ladder-compact.yaml`,
  `ladder-annex.yaml`, `ladder-task05.yaml` and `ladder-followup.yaml` are **deleted**.
- Rationale: all five declare the seven V1 tool names, and variously `annexes:`, cold cells and a
  daemon — none of which exist after T0. Each is therefore unloadable under the yaml-shape decision
  above: the loader errors on `cold:`, `annexes:`, `toc_format:` and `session_dir_template:`, and
  rejects an `allowed` entry naming a tool the file's own `quarry_tools` cannot honestly declare.
  Keeping a config file that cannot be loaded is worse than not keeping it. All five are recoverable
  at `origin/v1-final`.
  (HANDOFF §2's related principle is narrower than an earlier draft of this rationale claimed — as
  written it is about *languages*: "Nothing in the tree describes a language it does not support: no
  dormant branches, no tests asserting absence." It is cited here as the spirit these deletions
  follow, not as a rule that covers config files.)
- Rejected: migrating all six (four would describe tools and features that will not exist for
  several waves); moving them to a `v1-configs/` directory (dead config under a nicer name).

### sequential-execution

- Decision: cells and repetitions run strictly one at a time, in cell-minor order (rep 1 of every
  cell, then rep 2 of every cell, …).
- Rationale: `duration_ms` and `total_cost_usd` are measured metrics; parallel runs share rate
  limits and the prompt cache and would confound both. Cell-minor ordering spreads prompt-cache
  warmth evenly across cells rather than systematically favouring whichever cell runs second.
  Resume is what makes a long serial run restartable, which is what parallelism would have been
  for.
- Rejected: a worker pool across cells (faster wall clock, confounded timings); cell-major
  ordering (systematically warms the cache for later cells).

### exploration-schema-recovery

- Decision: the exploration output schema is recovered from
  `origin/v1-final:bench/loomyard-eval/README.md` §"Output schemas" and inlined into
  `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md` under a
  `## Output schema (exploration tasks)` heading, mirroring how `04-shedadapters-shuttle-impact.md`
  already carries its impact schema at line 104. The verbatim JSON to inline is:

  ```json
  {
    "relevant_files": ["internal/reedengine/geometry.go", "..."],
    "key_symbols": [
      {"name": "FuncOrTypeName", "file": "path/to/file.go", "role": "one sentence"}
    ],
    "summary": "3-6 sentences explaining how the mechanism works end to end",
    "confidence": "high|medium|low",
    "open_questions": ["anything left uncertain, if any"]
  }
  ```

- Rationale: **this is a blocker discovered during exploration.** V1 read the exploration schema
  from `bench/loomyard-eval/README.md`, which T0 deleted; only the impact schema survives, inline
  in its own task file. Task 01 is an exploration task, and `ladder-toc.yaml`'s `a0-none` — this
  task's own done-criterion — uses it. One schema per task file means one code path in the loader
  and no cross-file lookup.
- Rejected: restoring the whole V1 README (a protocol document describing tmux dispatch and seven
  tools); a separate `tasks/schemas/*.json` keyed by the yaml's `schema:` field (splits the task
  prompt from the shape it must answer in).

### one-preamble-for-every-cell

- Decision: one prompt template, identical for every cell in a matrix. The only per-cell difference
  is the list of tool names available, generated from what the cell is granted. No preference
  language, no anti-grep language, no naming of quarry, the arm, or the rung. Prompt =
  `PARALLEL_OPENING` + body (target dir + the available tool names, neutrally listed) + the
  `<TASK TEXT>` block extracted from the task file + `PARALLEL_BLOCK` + the closing
  schema-only-output sentence + the schema JSON.

  **The three fixed constants are ported verbatim** from
  `origin/v1-final:…/internal/ladder/preamble.go`: `PARALLEL_OPENING` (preamble.go:14-16),
  `PARALLEL_BLOCK` (preamble.go:20-33) and `closingSentence` (preamble.go:44-45). All three are
  themselves byte-for-byte copies of the committed preambles in the V1
  `bench/loomyard-eval/README.md`, so they carry the measurement's continuity. The body is the one
  part written fresh: V1's `B_PREAMBLE_BODY` (preamble.go:39-41) is the control's, and
  `mcpPreambleBody` (preamble.go:95-103) generated the treatment's — the pair this decision
  collapses into one neutral body naming whatever tools the cell has.

  "PARALLEL" in both constant names refers to **parallel tool calls within one agent's turn**, not
  to parallel A/B/C arms; nothing in their text mentions arms, agents or comparison, so they carry
  no framing that a one-cell-at-a-time design contradicts. The A/B/C vocabulary lived in the V1
  README's dispatch protocol and in the task files' own `## Setup` sections, neither of which
  reaches the prompt.
- Rationale: V1 gave the control and the treatment *different steering*. Its treatment preamble
  said "Use these tools as your PRIMARY tool … Do NOT reach for grep/ripgrep as a reflex", which
  the control never saw. That is why `summarize.go:46` had to exclude grep metrics from
  rung-vs-control comparison entirely — a known confound the summariser worked around instead of
  removing. With one preamble, a2-vs-a0 measures the tool, not the instruction to prefer it.
- Rejected: porting `preamble.go`'s constants verbatim including the control variant (preserves
  byte-comparability with v1 numbers at the price of keeping the confound); per-cell prompt text
  from the yaml (a feature `ladder-toc.yaml`'s own comment says would be needed for a grep-toc
  control; not in §9a, not in this task).
- **Risk, stated deliberately:** removing the "use this tool first, don't grep" steering may mean
  a treatment cell never calls its granted tool, in which case gate 1 fires and the cell measures
  the tool's prompt cost rather than the tool. It may also move T7's numbers relative to
  `v1-final:results/2026-09-02-toc/`. That is acceptable because HANDOFF §3 already says cost
  numbers are comparable only within one root, and T7 measures its own control in its own root.
  T7's done-criterion should be read as "a2 separates from *its own* control", not "a2 reproduces
  the August absolute numbers".

### metrics

- Decision: per rep, the harness computes:

  From the final `result` record: `num_turns`, `duration_ms`, `duration_api_ms`, `total_cost_usd`,
  `terminal_reason`, `stop_reason`, `is_error`, `permission_denials`.

  Summed over **assistant records grouped by `message.id`** (never from `modelUsage`):
  `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`. For
  each group, take any record's `input_tokens`/cache figures and the **maximum**
  `output_tokens` across the group's records.

  From `tool_use` content blocks: `tool_uses` (total), `tool_uses_breakdown` (`map[name]count`),
  `quarry_tool_uses` (uses whose name starts with the MCP tool prefix — see mcp-tool-prefix, never
  a hardcoded literal), `grep_tool_count` (native
  `Grep` calls), `bash_grep_count` (`Bash` calls whose `command` matches V1's regex verbatim:
  `` `(^|[|&;\s])(grep|rg)\b` ``), `grep_fallback_total` (the sum of the last two, reporting only).

  New per §9a, with no V1 counterpart: `tool_result_bytes` (total UTF-8 byte length of every
  `tool_result` block's text content), `tool_result_bytes_breakdown` (the same, keyed by the tool
  name of the matching `tool_use`), and `read_bytes` (the `Read`-tool subset of that).

  Also recorded, not computed: `model` (first assistant record's `message.model`) and `effort` (the
  flag the harness passed — the CLI does not echo it).

- Rationale: the `message.id` grouping is V1's one genuinely load-bearing accounting rule — Claude
  Code writes one transcript record per content block, each repeating the call's usage snapshot,
  and naive per-record summing inflated tokens 2.15× on a real matrix run. The grep regex matches
  grep/rg as a *leading command word*, not as a substring anywhere, which is why it is copied
  verbatim rather than re-derived. `modelUsage` is excluded because it folds in Claude Code's own
  Haiku overhead (confirmed in my probe: 916 Haiku input tokens on a two-turn Sonnet run).
- Rejected: taking token counts from the final `result.usage` (one line of code, forbidden by the
  plan for the reason above); omitting the new byte metrics (§9a names "tool calls and bytes, Read
  bytes" explicitly).
- **Retired from V1, deliberately:** `num_turns` and `duration_ms` as post-hoc reconstructions (the
  `result` record reports both); `output_tokens` as a lower-bound-only figure (usable now);
  `denied_tool_attempts` and `denied_tool_attempts_provisional` (`DenialShapePattern` was never
  validated against a real denial and the provisional flag was hardcoded `true` —
  `permission_denials` in the `result` record replaces both); `agent_id` and `transcript_source`
  (subagent-dispatch bookkeeping with no meaning under `claude -p`).

### claude-invocation

- Decision: every cell runs, with cwd set to the pinned task worktree and stdin from `/dev/null`:

  ```
  claude -p "<prompt>"
    --model <run_model> --effort <run_effort>
    --max-turns <max_turns>
    --tools Read,Grep,Glob,Bash
    --allowedTools "<granted tool names, each <mcp-prefix><tool>; see mcp-tool-prefix>"
    --mcp-config <per-cell config file> --strict-mcp-config
    --output-format stream-json --verbose
    --no-session-persistence
    --setting-sources ""
  ```

  stdout is tee'd to `raw/<cell>/<rep>/transcript.jsonl` as it arrives. `--allowedTools` is omitted
  entirely on a control cell (see the blinding gate). A control cell still gets
  `--mcp-config <file> --strict-mcp-config` where the file is `{"mcpServers":{}}`.

- Rationale: verified by probe on this host (Claude Code **2.1.236**, 2026-09-03; note the plan's
  §9a probe table records 2.1.259, so `claude --version` goes into provenance rather than being
  assumed).
  - **No `--permission-mode` flag is passed.** The session runs in the CLI's `default` mode, which
    `system.init` reports as `permissionMode: default`. Adding `bypassPermissions` or `dontAsk`
    would change what the measured agent can do and is not needed — see the probe below.
  - `--tools Read,Grep,Glob,Bash` reproduces V1's blinded-control base set (`settings.go:79`)
    for **every** cell, so the only difference between arms is the MCP grant. `Bash` is granted
    bare, exactly as V1 did: narrowing it to read-only command patterns would make *denied Bash
    calls* a behavioural difference between arms.
  - **The exact control configuration is probe-verified**, not inferred from the earlier
    `--allowedTools "Bash(grep:*)"` probe. With `--tools Read,Grep,Glob,Bash`,
    `--setting-sources ""`, `--strict-mcp-config` and **`--allowedTools` omitted entirely**:
    a read-only `Bash` call (`grep -n '^func' x.go`) executed, returned stdout, and the `result`
    record's `permission_denials` was `[]`. Read-only built-ins likewise need no allowlist entry (a
    `Grep` call ran with none). So a blinded control needs no `--allowedTools` at all, and V1's
    `settings.json` `permissions.allow` has no successor because none is required.
  - **Writes are denied by the CLI, and the denial is reported.** Probed: `touch dirty-marker.txt`
    was blocked (`"For security, Claude Code may only create or modify files in the allowed working
    directories"`) and appeared in `permission_denials`. So the `worktree_dirtied` observation is a
    backstop against something the CLI already prevents, not the primary defence. Bash is also
    confined to the working directory for reads: `cat ../../CLAUDE.md` was blocked the same way.
  - Because denials are reported, `permission_denials` (count and entries) is recorded per rep, so
    a cell whose agent burns turns on blocked commands is visible rather than silently slower.
  - `--strict-mcp-config` is **mandatory, not cosmetic**. Verified: without it, and even with
    `--setting-sources ""`, the operator's personal claude.ai MCP servers (Gmail, Google Drive)
    loaded into the session, adding 40+ tool definitions and taking the first assistant record's
    `cache_creation_input_tokens` from ~17 000 to ~40 000. With `--strict-mcp-config` and an empty
    document, the `system.init` record's `tools` list was exactly `[Bash, Glob, Grep, Read]` and
    `mcp_servers` was `[]`.
- Rejected: `--tools ""` as literally shown in §9a's probe table — that probe was isolating the MCP
  allowlist; with built-ins disabled the `none` control cannot read the codebase and the ladder
  collapses. `--bare` for full hermeticity — unusable: it forces `ANTHROPIC_API_KEY` auth and this
  host is OAuth (`apiKeySource: none` in the probe's `system.init` record).
- **Known residual contamination, recorded not fixed:** even with `--setting-sources ""`, the
  session still loads the operator's auto-memory directory, 16 skills and 48 slash commands
  (observed in `system.init`). This is constant across cells within a root, so the rung-vs-control
  contrast holds, but it inflates the baseline and is why the session fingerprint below exists.

### provenance

- Decision: `results/<root>/provenance.json` carries `written_at`, `ladder_file`, `quarry_commit`,
  `quarry_dirty`, `quarry_dirty_files`, `loomyard_commit`, `loomyard_repo_sha256` (sha256 of the
  resolved `LADDER_LOOMYARD_REPO` path — the hash, never the path), `hostname`, `go_version`,
  `claude_version` (from `claude --version`), `server_hashes` (`"<cell>/<rep>" → sha256`, empty
  when no cell grants an MCP tool), and `session_fingerprints`: for each rep, the fields lifted
  from that run's `system.init` record — `claude_code_version`, `model`, `permissionMode`, `tools`,
  `mcp_servers`, whether `memory_paths` was non-empty, and the counts of `skills` and
  `slash_commands`.
- Rationale: V1 wrote `LADDER_LOOMYARD_REPO` verbatim into committed JSON, which violates the
  no-machine-paths rule that now applies (`provenance.json` is committed under the
  results-raw-ignored decision). Hashing keeps "was this the same checkout" answerable without
  carrying the path. `server_hashes` per repetition *is* HANDOFF rule 1's enforcement ("do not edit
  source mid-matrix"); the harness warns loudly when more than one distinct hash appears in a root,
  reproducing V1's `serverChangedWarning`. The session fingerprint exists because the residual
  contamination above means "comparable within one root" would otherwise rest on nothing checkable:
  the harness compares each rep's fingerprint against the root's first and reports any drift as a
  non-fatal observation.
- Rejected: V1's struct as-is plus `claude_version` (carries the machine path); dropping the
  session fingerprint (smaller, leaves the root's central assumption unverifiable).
- Note: V1's `server_vcs_modified` field is dropped. It reads a build stamp that is structurally
  constant and carried no information; `server_hashes` is the check that works.

### gates

- Decision: exactly two.
  - **Gate 1 — granted tool was used.** Per *cell*, non-fatal. For every config with a non-empty
    `allowed`, if `max(quarry_tool_uses)` across its reps is `0`, record it in `summary.json` and
    print `!! <id>: tool-granted config whose agent never called a granted tool in any repetition
    -- this cell measures the tool's prompt cost, not the tool`.
  - **Gate 2 — control blinding.** Per *rep*, fatal, applied only when the cell's `allowed` is
    empty. All three checks scan **the whole rep transcript, marshalled back to JSON** — every
    record, every field, `system.init`'s `cwd` included — not a selected subset of fields. Three
    checks, in order, short-circuiting: (a) it contains the MCP tool prefix (see mcp-tool-prefix)
    → fatal; (b) it contains the quarry repo root path → fatal; (c) case-insensitive bare `quarry`,
    classified by **provenance rather than by location**: the occurrence is `target_origin` — the
    non-fatal observation `target_origin_quarry_mention` — when the token also appears in some
    `tool_result` payload **earlier in the same transcript**; it is fatal only when it appears with
    no such antecedent. V1's mechanism is still how the source set is identified (re-marshal with
    every `tool_result` block's nested content replaced by `"REDACTED"`, as `redactToolResultContent`
    did), but that split now selects provenance instead of being the verdict itself.
    **A gate-2 failure is not retried:** the rep fails once, is recorded, and the cell moves on —
    it never enters the `.invalid-<k>` path in resume-and-failure.
  Scanning everything rather than selected fields is what makes the two startup assertions in
  no-tmp-paths load-bearing: the worktree path appears in the transcript, so it must trip neither
  (b) nor (c) by construction.
- Rationale: gate 1 is a judgment about a whole cell — a `none` cell calling nothing is the
  expected case, and a granted cell calling nothing is itself the finding, not a reason to abort a
  root that costs real money. Gate 2 catches leakage that invalidates a single control run, so it
  must kill that run. The `tool_result` carve-out exists because the target codebase (Loomyard)
  mentions the word "quarry" in its own tracked prose, and a bare string match would halt the
  matrix over the target's own files.
  Check (c)'s provenance rule replaces a location rule that had a live false-positive path: the
  rationale for the carve-out is that Loomyard's own tracked prose says "quarry", and a control
  agent that quotes or paraphrases such a file **in its own assistant text** would have failed
  fatally under a location test — deterministically, so the retry path would have burned three reps
  reproducing it before recording the cell incomplete. Asking "did this token come from the target
  codebase?" is the question the carve-out was always trying to ask. Not retrying gate-2 failures
  follows from the same observation: a blinding failure is a property of the run's configuration or
  of the target's contents, not a flake, so repeating it cannot help.
- Rejected: making both fatal; making both advisory (gate 2 exists precisely because a leaked
  control is not a measurement); keeping check (c) as a location test (the false-positive path
  above); retrying gate-2 failures (three deterministic repeats of the same failure).
- **Retired from V1:** `GateRunPrompt` (the prompt is a positional argument the harness controls),
  `GateMaxTurns` (`--max-turns` is enforced by the CLI and reported as `terminal_reason:
  max_turns`), `GateModelPinned` (`--model` is passed), `GateNoTargetOverride` and
  `GateDeniedToolsNotUsed` (`--strict-mcp-config` plus `--allowedTools` enforce both), every
  cold/daemon gate.
- **Kept as a non-fatal observation, not a gate:** `worktree_dirtied` — after each rep, if
  `git status --porcelain` in the task worktree is non-empty, record it and restore the worktree
  before the next rep. `Bash` is granted bare, so this is how mutation is noticed.

### single-run-lock

- Decision: concurrent `ladder run` invocations are **out of scope**, and enforced rather than
  assumed. `run` takes an advisory lock by creating `<worktree-root>/.ladder.lock` with `O_EXCL`,
  writing its pid and results root into it, and removing it on exit; a second invocation fails
  immediately with `another ladder run holds <path> (pid N, results <root>)`. A stale lock is
  cleared by the operator, not automatically — the harness does not guess whether a pid belongs to
  a live run.
- Rationale: V1 needed `session_dir_template` precisely so one matrix's scratch could not collide
  with another's (`ladder-toc.yaml`'s own comment: "distinct from every other ladder file's
  template so a toc dispatch never collides with another matrix's scratch directories"), and V1's
  `lock.go` is being deleted with no successor. Because worktrees are keyed on `<task-id>` alone
  and harness scratch on a single `.scratch/ladder/` path, two simultaneous runs sharing a task
  would restore the same worktree underneath each other mid-run — silently corrupting both. One
  lock is the whole mitigation, and it is a fraction of the machinery `session_dir_template` +
  `lock.go` needed.
- Rejected: per-run scratch namespacing (V1's approach — reinstates a templating field the yaml
  shape just dropped, to support a mode nobody wants); no guard at all (the failure is silent
  cross-contamination of an expensive root, which is the worst possible failure mode here);
  automatic stale-lock reaping (guessing pid liveness across reboots is how a lock becomes a
  no-op).

### resume-and-failure

- Decision: a rep is complete iff `results/<root>/raw/<cell>/<rep>/run.json` parses and has
  `state: "complete"`. It is written **last**, after the transcript, the answer, the metrics, the
  score and the gates. A rep that fails (non-zero exit, unparseable stream, missing answer block,
  fatal gate) has its directory renamed to `<dir>.invalid-<k>` and is retried; after **3** attempts
  the cell is recorded in `summary.json`'s `incomplete[]`, the run continues with the remaining
  cells, and the process exits non-zero at the end.
- Rationale: write-last is what makes resume safe against a kill mid-rep. Continuing salvages the
  rest of an expensive root while the non-zero exit and the `incomplete[]` list make the shortfall
  impossible to miss.
- Rejected: aborting the matrix on the third failure (loses every unrun cell over one bad config);
  resuming on the transcript file's existence (a killed run leaves a truncated transcript that
  would read as done).

### answer-extraction

- Decision: the cell's answer is the **last** ` ```json … ``` ` fenced block in the final assistant
  record's concatenated text content. V1's `ExtractFencedJSON(text, "last")` is ported verbatim
  from `origin/v1-final:…/internal/ladder/fenced.go:36-54`, including its regex
  `` (?s)```json\s*(.*?)\s*``` `` (fenced.go:17) and its two-value return (the block **with** fences,
  used when embedding the schema into the prompt; the **inner** text, decoded as the answer). V1's
  deliberate divergence from the Python original is kept: an unrecognised selector is an error, not
  a silent fallthrough.
  - "Last", not "first", because the preamble embeds the schema as its own fenced block and the
    agent may restate it mid-answer; the closing instruction is "end your final message with ONLY a
    fenced json code block", so the final one is the answer by construction.
  - The extracted inner text is decoded as JSON. **No schema-key validation** beyond decoding: the
    scorer judges the content, and a structurally valid answer that omits a schema key is a low
    recall score, not a harness failure.
  - Failure semantics: no fenced block (`ErrNoFencedJSONBlock`) or JSON that will not decode ⇒ the
    rep fails with `answer extraction: <reason>` and enters the `.invalid-<k>` retry path in
    resume-and-failure. This is the "missing answer block" failure named there, and it *is* worth
    retrying — it is a nondeterministic formatting miss by the model, unlike a gate-2 failure.
- Rationale: this was undefined in an earlier draft while `answer.json` and "missing answer block"
  were already named — the extraction rule is what makes both meaningful. Reusing V1's function
  keeps one regex across all three call sites that need it (schema block out of the task file, the
  cell's answer, the scorer's reply), which is the property its doc comment argues for.
- Rejected: first-match (picks the schema echo); last top-level JSON object by brace matching (the
  prompt asks for a fence, so parsing without one accepts malformed output the measured agent was
  told not to produce); schema-key validation as a hard gate (turns a scoring signal into a run
  failure).

### scorer

- Decision: the scorer is a second `claude -p` call per rep:
  `--model <scorer.model> --effort <scorer.effort> --tools "" --mcp-config <empty>
  --strict-mcp-config --max-turns 1 --output-format stream-json --verbose --no-session-persistence
  --setting-sources ""`, stdin `/dev/null`. The prompt is V1's four-part assembly, unchanged:

  ```
  <rule>
  ## Task

  <task text>

  ## Reference fasit

  ```json
  <fasit with _meta stripped>
  ```

  ## Answer to score

  ```json
  <redacted answer>
  ```
  ```

  The two rules are ported verbatim from `origin/v1-final:…/internal/ladder/score.go:222-252`
  (`ExplorationRule` and `ImpactRule`), selected by the task's `schema:` field. The set of required
  score fields is derived at runtime by regexing `"(\w+)":` out of the rule's own fenced example,
  as V1 did, so the rule text and its validator cannot drift. The answer is redacted before
  embedding: the alternation is built from the ladder file's own `quarry_tools` (bare and
  prefixed with the MCP tool prefix of mcp-tool-prefix), the quarry repo root and the task worktree
  path.
- Rationale: recall and precision here are judgment calls (does the answer's summary describe the
  same mechanism the fasit found, not merely overlapping filenames), which is why they are produced
  by the model and only validated for presence in Go. Redaction preserves V1's stated invariant —
  the scorer must never learn which rung it is grading. The alternation is rebuilt from the yaml
  because V1 hardcoded the seven tool names there, and that was `score.go`'s only V1 coupling.
- Rejected: skipping redaction (the answer's prose names the tool it used); computing recall and
  precision in Go against the fasit (turns a judgment into string matching).

### server-block

- Decision: the ladder yaml gains an **optional** `server:` block:

  ```yaml
  server:
    name: quarry          # the MCP server name; tool names become mcp__<name>__<tool>
    build: ./cmd/quarry-mcp
    args: ["--target-dir", "{target_dir}"]
    env:
      QUARRY_STATE_DIR: ""
      QUARRY_BUILD_TAGS: ""
  ```

  **The absence of `server:` is a run-time check on the selected cells, never a load-time
  validation.** Loading a file whose cells grant tools while `server:` is absent is legal; the
  harness fails only when it is about to run such a cell, with
  `cell <id> grants <tools> but the ladder file declares no server: block`. The server binary is
  built **lazily**, on the first selected cell with a non-empty `allowed`, and not at all when
  every selected cell is a control. When it is built: once per invocation, `CGO_ENABLED=1` forced
  into the environment, sha256 recorded per `<cell>/<rep>` in `provenance.json`, and a per-cell MCP
  config written declaring exactly that one server. A control cell always gets
  `{"mcpServers":{}}`.
- Rationale: making the block optional is what lets this task's own done-criterion (`a0-none`,
  no tools) run with no quarry code in the tree, exactly as the brief requires — the harness needs
  no quarry code until T7. V1 hardcoded `./cmd/quarry-mcp` and `--target-dir`; T6 has not decided
  its flags, so they become yaml data. `CGO_ENABLED=1` is forced because the tree-sitter backend
  links C grammars and the failure otherwise reads as unrelated (V1's `BuildServer` doc comment).
- Rejected: hardcoding the V1 build target and flags (blocks T2 until T6); taking the server
  binary as a CLI flag (a machine path on the command line, and no build-hash provenance).

### mcp-tool-prefix

- Decision: the client-side MCP tool prefix is computed in **one** place as
  `"mcp__" + server.name + "__"`, defaulting to `mcp__quarry__` when the ladder file declares no
  `server:` block. Every consumer reads that one value: `quarry_tool_uses` in metrics, gate 2's
  check (a), the scorer's redaction alternation, and the `--allowedTools` names built from a cell's
  `allowed` list.
- Rationale: `server:` makes the server name yaml data, so a hardcoded literal `mcp__quarry__` in
  three separate consumers is a silent-failure trap: rename the server and gate 1 reads zero tool
  uses for every cell (the "granted tool never called" warning fires spuriously), gate 2's check (a)
  stops matching real leakage, and the redactor stops removing the tool names from the answer the
  scorer sees. Each failure is silent and each corrupts a different part of the result.
- Rejected: hardcoding the literal and documenting `name:` as decorative (the field then lies);
  per-consumer derivation (three places to keep in step).

### module-and-yaml-dependency

- Decision: the harness lives in the **root module** `github.com/Knatte18/quarry`, not a nested
  one, and adds exactly one external dependency: `gopkg.in/yaml.v3`.
- Rationale: the task's done-criterion is `go build ./... && go test ./...` green *at the repo
  root*, and a nested module is silently excluded from both — the harness would appear to pass by
  not being built. One new dependency is the honest cost of keeping the ladder files in yaml, which
  the plan lists under "Kept". `yaml.v3` is the de-facto standard, is pure Go, and adds no cgo, so
  `go list -deps` stays clean for the engine packages that care (plan §12's T1 criterion).
  HANDOFF §1's description of `go.mod` as tree-sitter-only is a statement of T0's result, not a
  prohibition on later tasks; this is nevertheless a deliberate widening and is recorded as such.
- Rejected: a nested module under `bench/loomyard-eval/ladder/` (keeps the root `go.mod` pure but
  drops the harness out of the done-criterion's own command, and needs a `go.work` to be usable);
  `sigs.k8s.io/yaml` (routes through JSON tags, loses yaml-native comments and error positions, and
  pulls a larger tree); hand-rolling a parser for the kept shape (the shape includes nested maps and
  lists of structs — a parser is more code than the harness's own logic and a fresh source of bugs).

### yaml-shape

- Decision: the loaded shape is exactly:

  ```yaml
  run_model: <string>          # required
  run_effort: <string>         # required
  max_turns: <int>             # required, plain int (no pointer-for-unset)
  reps: <int>                  # required
  scorer:
    model: <string>            # required
    effort: <string>           # required
  quarry_tools: [<string>...]  # the tool names this file's cells may grant; file-local, unvalidated
                               # against any package constant
  source_repo: env:LADDER_LOOMYARD_REPO   # must be exactly this literal
  server: {...}                # optional, see server-block
  tasks:
    <task-id>:
      task_file: <path>        # relative to the quarry repo root
      pinned_sha: <sha>
      schema: exploration|impact
      fasit: <path>            # relative to the quarry repo root
  configs:
    - id: <string>             # unique across the file
      ladder: <string>         # any single letter; the grouping key for control resolution
      task: <task-id>
      allowed: [<string>...]   # subset of this file's quarry_tools
  ```

  Validations kept from V1: `source_repo` must be the literal `env:LADDER_LOOMYARD_REPO`, resolved
  lazily only when a real checkout is needed; duplicate config id; unknown task reference; an
  `allowed` entry not in the file's own `quarry_tools`; **exactly one control per ladder letter
  actually present in the file**, where a control is a config with empty `allowed`; control lookup
  by field, never by parsing the id.

- Rationale: `source_repo`'s literal check is the no-machine-paths rule mechanised. The
  one-control-per-letter-present refinement exists because follow-up files legitimately declare
  only one letter. Field-based control lookup is why `ladder-toc.yaml`'s `b8`/`b9` ids work at
  all.
- **Removed V1 validations, each a coupling:** `quarry_tools` must equal the canonical seven
  (rejects every file once T6 ships one tool called `toc`); `ladder` must be `a` or `b` (an
  arbitrary cap `ladder-task05.yaml` already worked around by labelling `c*` configs `ladder: b`);
  the `session_dir_template` presence and relative-path checks; every cold/warm-counterpart rule;
  `max_turns` as `*int` with `RequirePins` refusing subcommands while unset (that dance existed
  because the ceiling had to be recalibrated post-hoc against a different accounting basis;
  `--max-turns` is a real flag now and the committed `60` is meaningful again).

### ladder-toc-migration

- Decision: `ladder-toc.yaml` becomes: `quarry_tools: [toc]`; `a2-toc-dir` and `b8-toc-dir` get
  `allowed: [toc]` with their **ids unchanged**; `a1-toc-file` and `b9-toc-file` are deleted;
  `worktree:`, `session_dir_template:` and every `cold: false` line are removed. **At T2 merge the
  file ships with its `server:` block fully written** (`build: ./cmd/quarry-mcp`, `name: quarry`,
  and the args T6 will accept), because the block is inert until a tool-granting cell is actually
  selected. T2's own end-to-end run is
  `ladder run --config …/ladder-toc.yaml --cells a0-none --reps 1`, which selects only the control,
  so nothing is built and the absent `cmd/quarry-mcp` is never referenced. T7 drops `--cells` and
  runs the whole matrix. The file's long header comment is updated to describe the new harness,
  keeping the design rationale for the a/b contrast.
- Rationale: T6 ships exactly one tool, `toc`, so `toc_file` has no successor and its two cells
  cannot exist. The ids stay because the file's own comment says they are the cross-root join key
  and T7's done-criterion names `a0-none` and `a2-toc-dir` literally.
- Rejected: leaving the seven V1 names (the file would declare six tools that will never exist and
  a cell that cannot run); reducing to `[toc_dir, toc_file]` (still names tools T6 will not ship).

### command-surface

- Decision: one binary with two subcommands.
  - `ladder run --config <yaml> --results <root> [--cells <id,...>] [--reps <n>] [--claude-bin
    <path>]` — prepares worktrees, builds the server lazily if a selected cell needs one, runs
    every selected cell × rep with resume, scores, gates, writes everything, prints the table and
    also writes it to `results/<root>/table.txt`. `--cells` restricts the run to named config ids
    (this is what lets T2 run `a0-none` alone with no quarry code in the tree); `--reps` overrides
    the file's `reps`.
  - `ladder report --results <root>` — re-derives `summary.json` and rewrites `table.txt` from
    `raw/` without re-running or re-scoring anything.

  No shell wrapper. The documented entry is
  `go run ./bench/loomyard-eval/ladder/cmd/ladder run --config bench/loomyard-eval/ladder/ladder-toc.yaml --results bench/loomyard-eval/ladder/results/<date>-toc`.
- Rationale: V1's twelve subcommands existed because an external orchestrator drove the loop one
  step at a time; `claude -p` lets one process own the whole run. `report` is separate so that a
  summariser or metrics fix does not cost another matrix — the transcripts are already on disk.
- Rejected: a single `run` subcommand (a summariser bug then costs a full re-run); porting V1's
  subcommand set (reinstates the protocol the rewrite removes).

### cache-contamination

- Decision: `cache_read_input_tokens` and `cache_creation_input_tokens` are reported as separate
  metrics and never summed into one number; `input_tokens_total` (`input + cache_read +
  cache_creation`) is reported alongside them. The harness emits the caveat itself, as a header
  line in `table.txt`: the first rep of a root pays cache creation while later reps read it. (The
  root's `conclusion.md` is hand-written by its author, not templated by the harness — see
  results-raw-ignored.) No attempt is made to defeat the cache.
- Rationale: verified in my probe — a second `claude -p` run minutes after the first read 10 308
  cached tokens created by the first. Cell-minor rep ordering spreads this roughly evenly across
  cells, so the contrast holds, but the per-rep numbers are not interchangeable and the
  median-over-reps is the honest statistic. Defeating the cache would make the numbers
  unrepresentative of real use.
- Rejected: cell-major ordering (systematically warms the cache for whichever cell runs later);
  ignoring it (leaves a known artefact undisclosed in the metric the rewrite's regression gate
  rests on).

## Technical context

**Where the reference implementation lives.** Every V1 file cited below is at `origin/v1-final`,
read with `git show origin/v1-final:<path>`; nothing is checked out and nothing is merged from it.
Paths are relative to `bench/loomyard-eval/ladder/`.

- `internal/ladder/score.go:222-252` — `ExplorationRule` and `ImpactRule`, to port verbatim.
- `internal/ladder/score.go:266` — `StripFasitMeta`; `:307-322` — the four-part prompt assembly;
  `:337-352` — `scoreFieldsFromRule`, the regex that derives required fields from the rule's own
  example; `:59-96` — `RedactText` and its alternation (rebuild generically).
- `internal/ladder/usage.go:22` — the grep regex; `:200-213` — `assistantCallGroups`; `:221-228` —
  `perCallUsage` and its max-`output_tokens` rule.
- `internal/ladder/gates.go` — `GateFinding`, `GateReport`, `GateBlinding`; `:229-248` —
  `redactToolResultContent`.
- `internal/ladder/fenced.go:17` — the fenced-JSON regex; `:36-54` — `ExtractFencedJSON`, ported
  verbatim (see answer-extraction). Only the dispatch-side callers of this file die with V1; the
  extractor itself is kept.
- `internal/ladder/preamble.go:14-16, :20-33, :44-45` — `PARALLEL_OPENING`, `PARALLEL_BLOCK`,
  `closingSentence`, ported verbatim; `:39-41` and `:95-103` — `B_PREAMBLE_BODY` and
  `mcpPreambleBody`, the two divergent bodies this task replaces with one.
- `internal/ladder/settings.go:79-96` — the fixed `["Read","Grep","Glob","Bash"]` base set and the
  reasoning for why a blinded control declares nothing else.
- `internal/ladder/runstate.go:22` — the `raw/<cell>/<rep>/` path; `:39-50` — `IsComplete`;
  `:183-211` — the `run.json` payload; `:216-263` — the `.invalid-<k>` rename and
  `MaxAttempts = 3`.
- `internal/ladder/summarize.go:280-299` — `Comparison` and `RangesDisjoint`; `:494-565` —
  `MetricStats`, `CellRecord`, `Summary`, `SummaryMeta`.
- `internal/ladder/ladder.go:80-160` — the yaml structs; `:181-207`, `:232-253`, `:365-373`,
  `:434-446` — the validations, kept and removed as listed above.
- `internal/ladder/server.go` — `BuildServer` (the `CGO_ENABLED=1` reasoning) and
  `MCPConfigDocument`.
- `tools/runmatrix/main.go:593-613` — the `provenance` struct; `:621-628` — `recordServerHash`;
  `:738-741` — the table columns; `:748-766` — `serverChangedWarning`; `:809-819` — gate 1.
- `internal/ladder/e2e_test.go` — worth reading for the end-to-end test shape.
- `bench/loomyard-eval/README.md:252-265` — the exploration output schema (see
  exploration-schema-recovery).

**Deletes wholesale, do not port:** `correlate.go` (168 lines of `~/.claude/projects` transcript
hunting, including an empirically-derived dot-to-hyphen directory mangling), `agentdef.go`,
`session.go`, `daemon.go`, `warm.go`, `coldcell.go`, `annex.go`, `precondition.go`, `lock.go`,
`plan.go`. Of `fenced.go` only the dispatch-side *callers* die; `ExtractFencedJSON` itself is
ported (see answer-extraction).

**Current tree.** `bench/loomyard-eval/` holds only `ladder/` (six yaml files, no Go) and `tasks/`
(five task markdown files, three `*.fasit.json` — tasks 02 and 03 have no fasit and no ladder file
references them). `go.mod` is module `github.com/Knatte18/quarry`, Go 1.26, requiring only
`go-tree-sitter` and `tree-sitter-go`; there is no existing yaml parser. `gopkg.in/yaml.v3` is
added to that module — see the module-and-yaml-dependency decision, which is where the choice and
its alternatives are settled rather than here.

**Task file structure.** A task markdown file carries a `## Setup` section with `/tmp` worktree
commands, a `## Scope` section, a blockquoted `` ## `<TASK TEXT>` `` block, and an answer-key
section that **must never reach the agent**.

Extraction is strictly **inclusion-based**: the harness takes the `<TASK TEXT>` block (dedented
from its `>` blockquote) and the `## Output schema (…)` fenced block, and nothing else. It never
looks for, matches, or excludes the answer-key heading. This matters because that heading is spelt
five different ways across the five task files and no two of the ladder-referenced ones agree:

| file | heading |
|---|---|
| `01-reed-geometry-exploration.md:41` | `## Notes for whoever prepares C's fasit / scores this` |
| `02-shedadapters-exploration.md:42` | `## Notes for whoever prepares C's fasit / scores this` |
| `03-reed-attach-geometry-review.md:46` | `## Notes for whoever scores this` |
| `04-shedadapters-shuttle-impact.md:119` | `## Notes for whoever scores this (ground truth — do not reveal to A/B/C)` |
| `05-mergeresolve-resolve-impact.md:144` | `## Notes for whoever scores this (ground truth — do not reveal to any arm)` |

An exclusion-based extractor keyed on any one spelling would leak the answer key from the other
files, which is a silent invalidation of every affected run rather than an error. The `## Setup`
section's `/tmp` paths are documentation of how the fasit was produced; the harness ignores them
and derives its own worktree paths.

**`stream-json` record shapes, verified on this host (2026-09-03, Claude Code 2.1.236):**

- `{"type":"system","subtype":"init", …}` — first record. Carries `tools`, `mcp_servers`, `model`,
  `permissionMode`, `claude_code_version`, `memory_paths`, `skills`, `slash_commands`, `session_id`.
  This is the session fingerprint's source.
- `{"type":"assistant","message":{…},"parent_tool_use_id":…,"uuid":…,"timestamp":…}` —
  `message.usage` carries `input_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`,
  `output_tokens`; `message.content[]` carries `text` and `tool_use` blocks (`name`, `input`).
- `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":…,"content":…}]},
  "tool_use_result":{…}}` — the tool result. `tool_use_result` is a *structured* mirror (for `Grep`:
  `mode`, `numFiles`, `filenames`, `content`, `numLines`, `totalLines`; for `Bash`: `stdout`,
  `stderr`, `interrupted`, `isImage`). Byte metrics measure the `tool_result` block's text content;
  correlation to a tool name goes through `tool_use_id`.
- `{"type":"result","subtype":"success", …}` — `num_turns`, `duration_ms`, `duration_api_ms`,
  `total_cost_usd`, `terminal_reason`, `stop_reason`, `is_error`, `usage`, `modelUsage`,
  `permission_denials`.
- `{"type":"rate_limit_event", …}` may appear anywhere; the parser must skip unknown `type` values
  rather than erroring.

**Environment.** `LADDER_LOOMYARD_REPO` comes from a gitignored `.scratch/ladder.env`; on this host
it resolves to a Loomyard checkout whose pinned commit `975578cd` carries its own `CLAUDE.md`,
`CONSTRAINTS.md` and `.claude/agents/`. Those are loaded into every cell equally and are part of
the constant baseline the session fingerprint records.

## Constraints

There is no `CONSTRAINTS.md` at the hub root. Constraints from the task brief, `CLAUDE.md`,
`HANDOFF.md` and this discussion:

- Go repository. **No Python.** No shell wrappers.
- **No tracked file may carry a machine path** — stated in `HANDOFF.md` §1, not in `CLAUDE.md`
  (which is three lines and says only "Go repo, do not introduce Python"). This drives
  results-raw-ignored, the
  `loomyard_repo_sha256` field, and the `source_repo: env:…` literal.
- No `/tmp`. The harness's own scratch goes under `.scratch/`, which is gitignored. Task worktrees
  are the one deliberate exception and live outside the quarry repo entirely — see no-tmp-paths for
  why the blinding gate requires it.
- Roughly 1 000–1 500 **non-test** lines, with tests besides. Plan §2's table sets the `~1 000–1 500`
  budget against V1's `9 000 (+8 300 test)` row, so it parallels the non-test figure; §9a's "roughly
  1 000–1 500 lines with tests" means "accompanied by tests", not "including tests". The Testing
  section below — six table-tested files, two test-built helper binaries, resume and failure paths —
  does not fit inside 1 500 lines total and is not meant to.
- Both surviving harness rules are prose in `HANDOFF.md` **§3** ("Two harness rules carry over into
  T2"); HANDOFF has no numbered rules, and plan §9a cites §3 for them too. They are: do not edit the
  code under test while a matrix runs (the binary hash per repetition stays in `provenance.json`),
  and cost numbers are comparable only within one results root.
- Everything else the old harness enforced is retired with the architecture that needed it; `claude
  -p` now guarantees it (plan §9a).
- The task brief's citation of "`HANDOFF.md` §2 rules 1 and 6" is an addressing error in the brief:
  §2 is the decisions list and the rule numbering belongs to the V1 README on `v1-final`, if
  anywhere. The content it points at is correct and is restated above.
- No quarry code may be required to build or test the harness. It must not wait on T1.
- `go build ./... && go test ./...` green; `go test ./...` must not require network, auth, or
  spend money.
- The tree-sitter backend needs `CGO_ENABLED=1` and a C toolchain — the harness's own package must
  not add a cgo dependency, but the server it builds does.

## Testing

**TDD candidates — write the test first:**

- `config.go` — the yaml loader and every validation. Table tests: each accepted shape, each
  rejected shape (duplicate id, unknown task, `allowed` outside `quarry_tools`, wrong
  `source_repo`, zero or two controls in a letter, each retired key producing its named error).
  Pure, no I/O beyond reading a fixture file.
- `metrics.go` — the whole accounting layer against fixture `transcript.jsonl` files under
  `testdata/`. Must cover: the multi-record `message.id` grouping (a fixture where naive summing
  over-counts), max-`output_tokens` within a group, the grep regex's leading-command-word rule
  (`grep foo` counts, `ripgrepping` does not, `cat x | grep y` counts), byte counting for `Read`
  vs other tools, a zero-tool-call transcript, an unknown record `type` being skipped, a
  `terminal_reason: max_turns` transcript.
- `gates.go` — the blinding gate's three checks in order, including the case where `quarry` appears
  only inside a `tool_result` (non-fatal observation) versus in assistant text (fatal), and gate 1
  over a cell with reps where `quarry_tool_uses` is `0` in all versus `0` in some.
- `prompt.go` — `<TASK TEXT>` extraction, run against **both** ladder-referenced task files (`01-…`
  and `04-…`, whose answer-key headings differ) and asserting that no text from the `## Setup` or
  answer-key sections appears in the output. Since extraction is inclusion-based, the assertion is
  that the output equals exactly the task-text and schema blocks — heading spelling is not
  load-bearing and must not be tested as if it were.
- `fenced.go` / answer extraction — the `"last"` selector picks the answer and not the schema echo
  when both fences are present; a reply with no fence returns `ErrNoFencedJSONBlock`; an
  unrecognised selector errors rather than falling through; a fence whose body spans many lines is
  captured whole.
- `score.go` — `scoreFieldsFromRule` derives exactly the fields each rule's example names; fasit
  `_meta` stripping; the redactor removes every `quarry_tools` name bare and prefixed, plus both
  paths.
- `summarize.go` — median/min/max/n; `RangesDisjoint`; `incomplete[]` population.

**Integration, offline and free (the `go test ./...` set):**

- **Stub MCP server.** A small Go program under `internal/ladder/testdata/stubmcp/`, built by the
  test with `go build`, speaking JSON-RPC over stdio: `initialize`, `tools/list` advertising two
  tools (one the cell grants, one it does not), `tools/call` returning a deterministic JSON text
  payload. The test drives the *harness-generated* per-cell MCP config into it — proving that
  document is well-formed, that the declared command and args launch, and that the tool names the
  harness will pass to `--allowedTools` match what the server advertises.
- **Fake `claude`.** A second test-built Go program that ignores its flags (after asserting the
  expected ones are present) and emits a canned `stream-json` stream on stdout, then exits. The
  harness's `run` path is exercised end to end against it: worktree prepare, MCP config write,
  process exec, transcript tee, metrics, gates, scorer (a second canned response), `run.json`,
  `summary.json`, `provenance.json`, table. Includes a resume test — kill after rep 1, re-run,
  assert rep 1 is not re-executed — and a failure test — three bad reps produce
  `.invalid-1..3`, an `incomplete[]` entry and a non-zero exit.
- The `claude` binary path must therefore be injectable (a `--claude-bin` flag defaulting to
  `claude` on `PATH`, or an equivalent seam), as must the `git` and `go build` runners — V1 already
  took a `Builder` seam for exactly this reason.

**Live smoke test, skipped by default:** guarded by an environment variable (e.g.
`LADDER_LIVE_TEST=1`), running one `reps: 1` `a0-none` cell against the real CLI from a **freshly
created** ladder worktree. This is the mechanised form of the task's done-criterion. It must assert
the `system.init` record's `tools` list is exactly `["Bash","Glob","Grep","Read"]` plus any granted
`mcp__*` names, and that `mcp_servers` is empty for a control — i.e. that a brand-new worktree
directory does not silently degrade the tool grant (see the trust note under no-tmp-paths) and that
`--strict-mcp-config` held. It must also assert **runtime** Bash permission, not just the
advertised tool list: at least one `Bash` `tool_use` in the transcript has a `tool_result` carrying
stdout, and `permission_denials` contains no entry for a read-only command. The advertised `tools`
list says what was offered; only an executed call proves the grant works from a fresh directory.

**Done-criterion verification, by hand:** a `reps: 1` run of `ladder-toc.yaml` cell `a0-none` on
this host, after which the operator opens `raw/a0-none/1/transcript.jsonl` and checks by hand that
`summary.json`'s `num_turns`, `tool_uses`, `cache_read_input_tokens`,
`cache_creation_input_tokens`, `output_tokens` and `read_bytes` match what the transcript actually
contains. This is the acceptance step, not an automated test.

## Q&A log

- **Q:** Package layout inside `bench/loomyard-eval/ladder/`? **A:** [auto-pick] `cmd/ladder/main.go` (thin) + `internal/ladder/` (one package, files per concern). **Why:** the shape V1 proved, keeps the yaml at its tracked paths, matches the plan's "files per concern, never a package per verb" rule, and keeps internals unit-testable.
- **Q:** Is `results/**/raw/` committed or ignored (plan §11)? **A:** [auto-pick] Ignored; the derived artifacts (`summary.json`, `provenance.json`, `table.txt`, `conclusion.md`) are committed. **Why:** transcripts carry absolute host paths and no tracked file may carry a machine path; committing them would resurrect V1's whole redaction gate.
- **Q:** Worktree and scratch locations, given the yaml's `/tmp` paths? **A:** [auto-pick, revised in review rounds 1 and 2 — see the round-2 entry below for the final path] Task worktrees go **outside** the quarry repo and carry no `quarry` token; the harness's own scratch stays at `<repo>/.scratch/ladder/`. `worktree:`, `session_dir_template:`, `cold:`, `warm_counterpart:`, `cold_worktree_template:` are removed and the loader errors on them. **Why:** `/tmp` is banned, but the first answer — `<repo>/.scratch/ladder-worktrees/` — collided with the blinding gate: check (b) matches the quarry repo root, which would then be the cell's own `cwd` in every record. *(A round-1 revision of this entry also claimed a blinded agent with bare `Bash` could walk up into quarry's source. That was probed in round 2 and is false — Bash is confined to the session's working directory. The check-(b) reason stands alone.)*
- **Q:** What happens to the other five `ladder*.yaml`? **A:** [auto-pick] Migrate `ladder-toc.yaml`; delete the other five. **Why:** they declare seven tools, annexes, cold cells and a daemon that no longer exist, so the new loader cannot load any of them; all five are recoverable at `origin/v1-final`. *(A round-1/2 draft justified this by generalising HANDOFF §2, which as written is about languages specifically — corrected in round 3; the unloadability argument stands on its own.)*
- **Q:** Cell concurrency? **A:** [auto-pick] Strictly sequential, cell-minor rep ordering. **Why:** duration and cost are measured metrics and parallel runs share rate limits and the prompt cache; resume is what makes a serial run restartable.
- **Q:** Where does the exploration output schema live, now that T0 deleted `bench/loomyard-eval/README.md`? **A:** [auto-pick] Recover it from `origin/v1-final` and inline it into `01-reed-geometry-exploration.md`, mirroring the impact schema in `04-…md`. **Why:** it is the schema `a0-none` — this task's own done-criterion — needs, and one schema per task file collapses V1's two-source parsing into one path.
- **Q:** The prompt each cell agent receives? **A:** [auto-pick] One preamble, identical for every cell; the only per-cell difference is which tool names are listed. **Why:** V1 gave the treatment arm "use these tools as your PRIMARY tool, do NOT reach for grep" steering the control never saw — a confound its own summariser had to work around by excluding grep metrics from comparison.
- **Q:** Which metrics, derived how? **A:** [auto-pick] `result`-record fields for turns/duration/cost/terminal reason; token counts summed over assistant records grouped by `message.id`; tool counts and the new byte metrics from `tool_use`/`tool_result` blocks; V1's grep regex verbatim. **Why:** the `message.id` grouping fixed a 2.15× inflation on a real run; `modelUsage` folds in Claude Code's own Haiku overhead, confirmed by probe.
- **Q:** What goes in `provenance.json`, given it is committed? **A:** [auto-pick] V1's fields minus the machine path, plus `loomyard_repo_sha256`, `claude_version`, and a per-rep session fingerprint from the `system.init` record. **Why:** V1 wrote `LADDER_LOOMYARD_REPO` verbatim into tracked JSON; the fingerprint is what makes "comparable within one root" checkable rather than assumed.
- **Q:** Gate severity and shape? **A:** [auto-pick] Gate 1 (granted tool used) per cell, non-fatal warning; gate 2 (control blinding) per rep, fatal, with V1's three ordered checks and the `tool_result` carve-out. **Why:** a granted cell calling nothing is the finding, not a reason to abort an expensive root; a leaked control is not a measurement.
- **Q:** Resume key and per-rep failure handling? **A:** [auto-pick] `run.json` with `state: "complete"`, written last; failures renamed `.invalid-<k>`, 3 attempts, then `incomplete[]` and continue, exiting non-zero. **Why:** write-last survives a mid-rep kill; continuing salvages the rest of an expensive root while the non-zero exit makes the shortfall unmissable.
- **Q:** Test strategy, given the real dependency is the `claude` binary? **A:** [auto-pick] Three layers — a test-provided stub MCP server driven from the harness-generated config, a fake `claude` binary emitting canned stream-json, and an env-guarded live smoke test. **Why:** satisfies the brief's stub-MCP requirement literally while keeping `go test ./...` offline, free and deterministic.
- **Q:** `ladder-toc.yaml`'s tool names, given T6 ships one tool called `toc`? **A:** [auto-pick] `quarry_tools: [toc]`; `a2-toc-dir` and `b8-toc-dir` get `allowed: [toc]` with ids unchanged; `a1-toc-file` and `b9-toc-file` deleted. **Why:** ids are the cross-root join key and T7's done-criterion names them literally; `toc_file` has no successor in T6's surface.
- **Q:** Command surface? **A:** [auto-pick] One binary, two subcommands: `run` and `report`. No shell wrapper. **Why:** V1's twelve subcommands existed because an external orchestrator drove the loop step by step; `report` is separate so a summariser fix does not cost another matrix.
- **Q:** How is the scorer invoked? **A:** [auto-pick] A second `claude -p` call with `--tools ""`, `--max-turns 1`, V1's four-part prompt verbatim, required fields derived by regex from the rule's own example, and the answer redacted using an alternation built from the file's own `quarry_tools`. **Why:** recall and precision are judgments, not string matches, and the scorer must not learn which rung it is grading.
- **Q:** How is the MCP server under test declared, given `cmd/quarry-mcp` does not exist until T6? **A:** [auto-pick] An optional `server:` block (`name`, `build`, `args` with `{target_dir}`, `env`); absent means control-only with an empty MCP document. **Why:** optionality is what lets this task's own `a0-none` criterion run with no quarry code in the tree; V1's hardcoded build target and flags would block T2 on T6.
- **Q:** How is prompt-cache contamination between reps handled? **A:** [auto-pick] Report cache-read and cache-creation separately, never summed, plus an `input_tokens_total`; state the effect in `conclusion.md`; do not try to defeat the cache. **Why:** a probe showed a second run reading 10 308 tokens cached by the first minutes earlier; cell-minor ordering spreads it evenly, and defeating the cache would make the numbers unrepresentative.
- **Q:** The exact per-cell `claude -p` invocation? **A:** [auto-pick] `--tools Read,Grep,Glob,Bash` for every cell, `--allowedTools` carrying only the granted `mcp__quarry__*` names (omitted entirely on a control), always `--mcp-config … --strict-mcp-config` (empty document on controls), plus `--model`, `--effort`, `--max-turns`, `--output-format stream-json --verbose`, `--no-session-persistence`, `--setting-sources ""`, cwd = the pinned worktree, stdin `/dev/null`. **Why:** probe-verified — without `--strict-mcp-config` the operator's personal Gmail/Drive MCP servers load into the cell and roughly double the system-prompt token count; §9a's `--tools ""` was isolating the allowlist and would leave the control unable to read the codebase; `--bare` is unusable because this host is OAuth.
- **Q:** Bash granted bare, or narrowed to read-only command patterns? **A:** Revised during the interview to bare `Bash`, matching V1's control set exactly, with a `worktree_dirtied` observation and a `git restore` between reps. **Why:** narrowing it would make denied Bash calls a behavioural difference between arms, which is exactly the kind of confound the one-preamble decision exists to remove.
- **Q:** [review round 1] Does the ~1 000–1 500 line budget include tests? **A:** No — non-test lines, tests besides. **Why:** plan §2's table sets that budget against V1's `9 000 (+8 300 test)` row, so it parallels the non-test figure, and the testing plan here cannot fit in 1 500 lines total.
- **Q:** [review round 1] Where do the two surviving harness rules actually live? **A:** `HANDOFF.md` §3, as prose; there are no numbered rules anywhere. **Why:** the task brief cites "§2 rules 1 and 6", but §2 is the decisions list; plan §9a cites §3 correctly. Content right, address fabricated.
- **Q:** [review round 1] Is a fresh worktree "already trusted" by Claude Code? **A:** Not asserted any more — the V1 hazard was `permissions.allow` in `settings.json` being ignored in an untrusted directory, and this design uses no settings file at all (`--setting-sources ""`, grant on the command line). The live smoke test verifies it rather than assuming it, asserting the `system.init` tools list from inside a freshly created worktree. **Why:** every other invocation claim in this document is probe-backed; this one was not.
- **Q:** [review round 2] The chosen worktree root `…/quarry-ladder/…` contains the token `quarry`, which check (c) matches fatally — does every control rep die? **A:** It would have. The root is now `${LADDER_WORKTREE_ROOT:-${XDG_CACHE_HOME:-$HOME/.cache}/ladder-eval}/worktrees/<task-id>`, and the harness asserts at startup that the resolved root is neither under the quarry repo root nor contains a case-insensitive `quarry`. The gate's scan scope is now stated explicitly: the whole marshalled transcript, `system.init`'s `cwd` included. **Why:** the cell's cwd is in the transcript, so the path has to be clean by construction; carving the worktree path out of the gate would make the gate nearly unfalsifiable.
- **Q:** [review round 2] Was the actual control configuration — bare `Bash`, `--allowedTools` omitted — ever probed? **A:** It had not been; it now is. With `--tools Read,Grep,Glob,Bash`, `--setting-sources ""`, `--strict-mcp-config` and no `--allowedTools`: a `grep` via Bash executed with `permission_denials: []`. A write (`touch`) was blocked and recorded as a denial, as was a read outside cwd (`cat ../../CLAUDE.md`). No `--permission-mode` is passed; `default` is what the probe ran under. The live smoke test now asserts an executed Bash call, not just the advertised tools list. **Why:** the earlier probe used `--allowedTools "Bash(grep:*)"`, so it proved a different configuration than the one every cell will use.
- **Q:** [review round 2] Can the migrated `ladder-toc.yaml` load at T2, when it grants `toc` but `cmd/quarry-mcp` does not exist? **A:** Yes. The all-control rule is a **run-time** check on selected cells, not load-time validation, and the server binary is built lazily on the first selected tool-granting cell. The file ships with its `server:` block written; T2 runs `--cells a0-none --reps 1`, which builds nothing. **Why:** the two decisions contradicted each other as written — the file would have been invalid between T2 and T7.
- **Q:** [review round 2] Who writes `table.txt` and `conclusion.md`? **A:** The harness writes `table.txt` (both `run` and `report`) and emits the cache caveat as its header line. `conclusion.md` is hand-written by the root's author — T7's deliverable, not a harness output. **Why:** both were named as committed artifacts with no producer, and "the conclusion.md template" implied a harness feature that does not exist.
- **Q:** [review round 2] What replaces `lock.go` and `session_dir_template` now that worktrees are keyed on `<task-id>` alone? **A:** Concurrent runs are out of scope and enforced: an `O_EXCL` lock at `<worktree-root>/.ladder.lock`, cleared by the operator if stale. **Why:** two simultaneous runs sharing a task would restore the same worktree underneath each other mid-run, silently corrupting both roots.
- **Q:** [review round 2] Where is the no-machine-paths rule actually stated? **A:** `HANDOFF.md` §1. `CLAUDE.md` is three lines and says only "Go repo, do not introduce Python". **Why:** same class of addressing error as round 1's phantom rule numbers.
- **Q:** [review round 2] Is the answer-key heading spelling load-bearing for prompt extraction? **A:** No, and it must not be: extraction is inclusion-based (task-text block + schema block only). The five task files spell that heading five different ways. **Why:** an exclusion-based extractor keyed on one spelling would silently leak the answer key from the others.
- **Q:** [review round 3] How does the harness find the cell's answer in the transcript? **A:** The **last** ` ```json … ``` ` fenced block in the final assistant record's text, via V1's `ExtractFencedJSON(text, "last")` ported verbatim (`fenced.go:36-54`). No fence or undecodable JSON fails the rep and *is* retried; no schema-key validation. **Why:** it was undefined while `answer.json` and "missing answer block" were already named outputs; "last" is required because the prompt embeds the schema as its own fenced block.
- **Q:** [review round 3] Does the `--mcp-config` path — which lives under the quarry repo and contains the token — reach the transcript and kill controls? **A:** No, probed: a run from the relocated worktree with `--mcp-config` pointing under the quarry repo produced **zero** `quarry` occurrences in the whole transcript; argv is not echoed in `stream-json`. Harness scratch may stay at `<repo>/.scratch/ladder/`. **Why:** the earlier probe established `mcp_servers: []`, which is not the same claim.
- **Q:** [review round 3] Anything else in the stream carrying the token? **A:** Yes — `system.init.memory_paths`, which is derived from cwd. Relocating the worktree clears it, and it also means a blinded cell no longer loads the operator's *quarry-project* auto-memory (notes written while working on this benchmark). A fresh worktree gets its own empty project memory. **Why:** found while probing the finding above; it is a substantive blinding fix, not only a gate-arithmetic one.
- **Q:** [review round 3] A control agent quoting Loomyard prose that says "quarry" fails check (c) fatally and deterministically — then gets retried three times. Is that intended? **A:** No. Check (c) now classifies by **provenance**: `target_origin` (non-fatal) when the token also appears in an earlier `tool_result` payload, fatal only without such an antecedent. And gate-2 failures are never retried — the rep fails once. **Why:** a blinding failure is a property of the configuration or the target's contents, not a flake.
- **Q:** [review round 3] Is `mcp__quarry__` hardcoded while `server.name` is yaml data? **A:** It was. The prefix is now computed once as `"mcp__" + server.name + "__"` (default `mcp__quarry__` with no `server:` block) and read by metrics, gate 2 check (a), the redactor and `--allowedTools`. **Why:** a renamed server would otherwise silently zero gate 1, blind gate 2 to real leakage, and stop the redactor hiding tool names from the scorer — three silent, different corruptions.
- **Q:** [review round 3] Where do `PARALLEL_OPENING` and `PARALLEL_BLOCK` come from, and does their "parallel" framing survive one-cell-at-a-time execution? **A:** Ported verbatim from `preamble.go:14-16` and `:20-33` (plus `closingSentence` at `:44-45`), all byte-for-byte copies of the V1 README's committed preambles. "Parallel" means parallel *tool calls in one turn*, not parallel arms; the A/B/C vocabulary never enters the prompt. **Why:** they are the two largest parts of the prompt and had no cited source.
- **Q:** [review round 3] Which module does the harness join, and which yaml library? **A:** The root module `github.com/Knatte18/quarry`, plus `gopkg.in/yaml.v3` — one new dependency, recorded as a deliberate widening of a `go.mod` T0 had reduced to tree-sitter only. **Why:** the done-criterion is `go build ./... && go test ./...` at the repo root, and a nested module is silently excluded from both.
- **Q:** Which V1 gates and metrics are retired? **A:** Retired: `GateRunPrompt`, `GateMaxTurns`, `GateModelPinned`, `GateNoTargetOverride`, `GateDeniedToolsNotUsed`, every cold/daemon gate; `denied_tool_attempts` and `_provisional` (`DenialShapePattern` was never validated against a real denial and the provisional flag was hardcoded `true`), `agent_id`, `transcript_source`, `server_vcs_modified`. **Why:** the CLI now enforces or reports each directly, or the field was structurally constant and carried no information.
