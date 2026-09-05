# Discussion: Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
slug: ladder-kickstart
status: discussing
parent: main
```

## Problem

The measurement programme has closed negative on *pull* mode — the agent calling `toc` mid-session.
T7, M1 and M3 between them found no shape and no sample size where a directory-level table of
contents separates from its control on any cost metric (`docs/roadmap.md`, and the two conclusions
it points at). Every one of those cells handed the agent a *tool* and measured whether it paid for
itself.

This task measures the other half of the hypothesis, the §8.2 mechanism in `docs/rewrite-plan.md`:
*push* mode. A plan card's glyphs are resolved mechanically **before** the agent starts, and the
resulting file+span pack is injected into the prompt. The agent spends zero turns looking. If push
mode does not separate either, the surface has been measured from both directions and the roadmap's
parked-T8 condition is closed twice over; if it does separate, the win belongs to pre-resolution
rather than to an MCP tool, which is a different product than the one T8 assumed.

Why now: M3 was the last cell of the pull-mode programme, and the roadmap's build queue is stopped
until measurement says where the surface pays. This is the one remaining designed measurement.

No MCP server is involved in any cell here. All three arms are pure prompt variants, so the
harness's fragile MCP path is not exercised and control blinding is trivial.

## Scope

**In:**

- A new ladder file, `bench/loomyard-eval/ladder/ladder-kickstart.yaml`, with one task and three
  cells (`e0-names`, `e2-files`, `e1-pack`), no `server:` block and no `quarry_tools`.
- A new benchmark task file + fasit under `bench/loomyard-eval/tasks/` (07), plus three per-cell
  *card* files.
- Harness change 1: a per-config `card:` field, rendered into the prompt.
- Harness change 2: decouple "is this letter's control" (comparison baseline + blinding) from
  "grants no tools" (server build), so three tool-less cells can share one ladder letter.
- Harness change 3: a `ladder pack` subcommand that generates the kick-start pack through
  `quarry.Repo.Resolve` and records it in `provenance.json`, plus a pre-dispatch verification that
  the pack in e1's card is the pack provenance recorded.
- Harness change 4 (the standing roadmap small item): `WriteProvenance` creates the results root.
- Running the 3 × 10 matrix and writing `conclusion.md` under
  `bench/loomyard-eval/ladder/results/<date>-kickstart/`.
- `docs/roadmap.md`: strike the `mkdir -p` small item, add one line under the measured record
  pointing at the new results root.

**Out:**

- M4b, the edit-task variant (agent revising code in a throwaway worktree, scored by diff/tests).
  It becomes a task only if e1 separates here.
- Any change to the MCP server, the engine, or the `quarry` facade's behaviour. The facade is
  *called* by the new pack command; it is not modified.
- Any new scoring rule or schema kind. The task uses the existing `exploration` schema.
- A Go implementation of Mann–Whitney U. The arithmetic is shown by hand in `conclusion.md`, as
  `results/2026-09-05-ladder-d/conclusion.md` does.
- Any change to `ladder-toc.yaml` or to the three existing results roots.
- Reviving `ControlFor` as a live code path (it is currently unused outside `config.go`).

## Decisions

### D1 — the three arms differ by a per-config *card*, not by three task files

- **Decision:** `Config` gains an optional `card` field naming a repository-relative markdown file.
  `RenderPrompt` takes the card's text as one more section and places it **immediately after
  `target.TaskText`**, before `PARALLEL_BLOCK`. A config with no `card` renders exactly the prompt it
  renders today, byte for byte.
- **Rationale:** `run.go`'s `runCellRepetition` derives the pinned worktree as
  `TaskWorktreePath(worktreeRoot, cfg.Task)` — one worktree per *task key*. Three task entries would
  give the three arms three different worktree paths, and `renderBody` names that path in the
  prompt, so the arms would differ by an unintended string. One task key means one worktree, one
  `targetDir`, and the card as the only difference between arms.
- **Rejected:** three `tasks:` entries with three `task_file`s (the `targetDir` confound above, plus
  three worktree checkouts of the same SHA); placeholder substitution inside one task file (a second
  templating language in `prompt.go` for no gain).

### D2 — "control" is decoupled from "grants no tools"

- **Decision:** `Config` gains `control *bool` (yaml `control:`). `Config.IsControl()` returns
  `*c.Control` when set and `len(c.Allowed) == 0` otherwise. A new `Config.GrantsTools()` returns
  `len(c.Allowed) > 0`.
- **The complete call-site sweep.** Every `IsControl()` call site in the package was enumerated by
  `grep -rn "IsControl()" bench/loomyard-eval/ladder --include=*.go | grep -v _test` during this
  discussion, and each is classified below. The plan must re-run that grep and confirm the list is
  still exactly these ten before editing, so the enumeration is checked rather than assumed; any new
  site the grep turns up is classified by the same question — *does this branch depend on whether the
  cell has an MCP server attached, or on whether it is the letter's comparison baseline?*

  | site | today | becomes | why |
  | --- | --- | --- | --- |
  | `run.go:164` `needsServer` | `!c.IsControl()` | `c.GrantsTools()` | builds the server only for a cell that has one |
  | `run.go:182` `ServerHashes` skip | `c.IsControl()` | `!c.GrantsTools()` | records a hash only for cells the server serves |
  | `run.go:368` `CheckRenderedControlPrompt` | `cfg.IsControl()` | `!cfg.GrantsTools()` | blinding must cover all three e-cells |
  | `run.go:415` `CheckServerConnected` | `!cfg.IsControl()` | `cfg.GrantsTools()` | only a granted cell has a server to connect |
  | `run.go:558` `CheckBlinding` | `cfg.IsControl()` | `!cfg.GrantsTools()` | blinding must cover all three e-cells |
  | `run.go:664` `--allowedTools` argv append | `!cfg.IsControl()` | `cfg.GrantsTools()` | **would otherwise append `--allowedTools ""` to e1/e2 only** |
  | `mcp.go:47` `MCPConfigDocument` empty-servers branch | `cfg.IsControl()` | `!cfg.GrantsTools()` | **would otherwise send e1/e2 down the granted-cell branch, which errors on `l.Server == nil`** |
  | `run.go:902` `ControlForLadder` | `cfg.IsControl()` | unchanged | comparison baseline |
  | `run.go:912` `RunState.IsControl` | `cfg.IsControl()` | unchanged | comparison baseline |
  | `config.go:131` `ControlFor` / `config.go:232` `validate` | `c.IsControl()` | unchanged | comparison baseline; `validate` keeps "exactly one control per ladder letter that appears" |

- **The two sites in bold are the reason the sweep is written out rather than summarised.**
  `run.go:664` would give `e1-pack` and `e2-files` two extra argv entries (`--allowedTools` and an
  empty string) that `e0-names` does not get — an unintended arm difference in the CLI invocation
  itself, with CLI-dependent semantics for an empty allowlist, and exactly the class of confound D1
  exists to remove. `mcp.go:47` is worse than a confound: with `l.Server == nil` in this ladder file,
  a non-control cell reaching the granted branch returns `mcp config for cell e1-pack: ladder file
  declares no server block` and the run dies before rep 1.
- `runstate`'s `IsControl` field and `summarize`'s `Control` flag keep reading `IsControl()`, so the
  rung-vs-control table pairs `e1-pack` and `e2-files` against `e0-names`.
- **Rationale:** all three e-cells grant no tools, so today's `IsControl()` would call all three
  controls: `validate` rejects the file, and `summarize.go:318` would find no rung to pair. Yet
  blinding *should* apply to all three (none of them may leak the word `quarry`) and the server
  *should not* be built for any of them. The two meanings have been the same thing only by accident
  in `ladder-toc.yaml`; ladder e is the first file where they differ.
- **Backwards compatibility:** every config in `ladder-toc.yaml` omits `control:`, so `IsControl()`
  returns exactly what it returns today and no existing behaviour, hash or result changes.
- **Rejected:** relaxing validate to "at least one control" (leaves `summarize` picking an arbitrary
  baseline); granting e1/e2 a nominal tool so they stop being controls (would build and attach the
  MCP server, destroying the "no MCP cell" property that makes this matrix cheap and blinding
  trivial).

### D3 — the pack is generated once by a `ladder pack` subcommand and verified at dispatch

- **Decision:** a third subcommand, `ladder pack`, with flags `--config`, `--results` and
  `--card` (the e1 card path to write). It:
  1. prepares/restores the pinned worktree for the ladder file's task, exactly as `run` does;
  2. opens that worktree through the `quarry` facade and makes **one** batched
     `(*quarry.Repo).Resolve(targets)` call over the full glyph list, read from the ladder file's new
     `pack_targets:` list;
  3. halts, naming the offending target, if any result's status is not `found` — `not_found`,
     `ambiguous`, `multipart` and a pre-resolution `error` are all fatal;
  4. renders the pack (D5) and writes it into the e1 card file between two fixed sentinel lines, and
     writes the raw resolve JSON to `<results>/pack-resolve.json`;
  5. writes/merges `provenance.json` with a `kickstart_pack` block (D4).
- `run` then verifies, once per invocation before rep 1: the e1 card's sentinel-delimited pack block
  hashes to `kickstart_pack.pack_sha256`, and `provenance.quarry_commit` /`quarry_dirty` still match
  the block's own recorded pair. Any mismatch is a hard error naming both hashes and does not spend
  an API call.
- **Rationale:** the proposal's requirement is "the pack in the prompt is the one provenance
  records". A generated card plus a hash check enforces exactly that, mechanically, rather than
  trusting an author. Generating once satisfies "before rep 0", and harness rule 1 (the code under
  test is pinned per rep by binary/commit hash) is what makes one generation valid for the whole
  matrix.
- **Note:** the ladder harness is in the same Go module as `quarry` (`go.mod` declares
  `github.com/Knatte18/quarry`; there is no second `go.mod`), so `ladder pack` imports
  `github.com/Knatte18/quarry/quarry` directly. This is the batched facade's first real exercise, as
  the proposal intends. No cell calls it at run time.
- **Rejected:** hand-writing the pack into the card and recording only its hash (nothing ties the
  hash to a real resolve); regenerating per rep (contradicts "generated ONCE, before rep 0", and
  would let a mid-matrix edit change the treatment).

### D4 — the pack block lives inside `provenance.json`

- **Decision:** `Provenance` gains `KickstartPack *KickstartPack \`json:"kickstart_pack,omitempty"\``
  with fields: `generated_at`, `quarry_commit`, `quarry_dirty`, `loomyard_commit`, `targets []string`,
  `pack_sha256`, `resolve_sha256`, `card_file`. `MergeProvenance` carries an existing block forward
  unchanged; a `next` invocation never sets it (only `ladder pack` does). A resumed root whose block
  differs from the card on disk fails the D3 verification.
- **Rationale:** `run` writes `provenance.json` through `MergeProvenance` at startup, so a block
  written by `ladder pack` into a field the struct does not know would be silently dropped on the
  first `run`. It has to be a real field.
- **Rejected:** a sibling `pack.json` referenced from nowhere (the proposal asks for provenance to
  record it, and an unreferenced file is not a record).

### D5 — pack line format: locations and signature, never the docstring

- **Decision:** one line per glyph, rendered by the pack command itself from the resolve JSON:

  ```
  <glyph> → <file> <start>-<end>
      <signature>
  ```

  `<file>` is `Symbol.File` (repository-relative), `<start>`/`<end>` are `Symbol.Start`/`Symbol.End`,
  and `<signature>` is `Symbol.Signature` with internal newlines collapsed to single spaces. Glyphs
  are emitted in the ladder file's `pack_targets:` order. `Symbol.Doc` is **not** emitted.
- **Rationale:** the treatment under test is *knowing where things are*. `quarry.RenderResolveText`
  is available and would be zero new code, but it emits `Doc` — the complete docstring per symbol —
  which would turn the treatment into "locations plus a large slab of prose content" and make a win
  uninterpretable. `SigEnd` is likewise dropped: the instruction is to read the whole span.
- **Rejected:** reusing `RenderResolveText` verbatim (the doc confound); emitting spans only, no
  signature (the proposal specifies the signature, and it is what makes the line legible without a
  read).

### D6 — the three cards

All three cards carry the same `Uses:` glyph-name list, in the same order. They differ only below it.

- **`e0-names`** (control, `control: true`): the `Uses:` list and nothing else. No locations.
- **`e2-files`** (secondary, descriptive): the `Uses:` list, plus a `Files:` list of the seven
  distinct file paths the glyphs live in, deduplicated, no per-glyph mapping, no line numbers. This
  is Millhouse's current plan-card format and C1's backup mode.
- **`e1-pack`** (treatment): the `Uses:` list, then the sentinel-delimited generated pack block (D5),
  then one instruction: read **all** listed spans in parallel, in one turn, before doing anything
  else.
- **Rationale for the instruction living in e1 only:** front-loading reads is only possible when
  locations are known. That is the mechanism under test, not a prompt trick withheld from the
  controls — a control cannot follow it. The longer prompt's input-token cost is part of the
  treatment and is captured by `cost_usd`.
- **Rationale for e2's shape:** the primary test is e1 vs e0, so e2 need only answer "do spans beat a
  plain file list". Giving e2 a per-glyph file mapping would make it a third treatment rather than
  the format Millhouse actually writes today.
- **Consequence, stated openly:** e1 minus e2 is not a clean "spans" contrast — e1 also adds the
  signature and the parallel-read instruction. e2 is declared descriptive precisely because of this;
  no test is run on it.

### D7 — the question and the glyph list

- **Subject:** the weft merge lifecycle in Loomyard at the pinned commit — how the repository
  distinguishes a *fabric-recorded* merge in progress from *foreign* git merge state left by an
  operator's own `git merge`, and what each layer does with that distinction.
- **Why this subject:** the pin `72c23d9` is literally "Surface merge-in-progress in fabric status",
  so the mechanism is present and coherent at the pin; the answer genuinely requires holding several
  predicates and one typed error together (control-flow / invariant tracing, not lookup); it spans
  four packages and seven files; and it is untouched by tasks 01–06, so no existing fasit overlaps
  it.
- **`pack_targets` (9 glyphs, 4 packages, 7 files), in card order:**

  | glyph | file at `72c23d9` |
  | --- | --- |
  | `internal/fabriccli#addWeftVerbs` | `internal/fabriccli/weft_verbs.go` |
  | `internal/fabricengine#Fabric.MergeInProgress` | `internal/fabricengine/mergelifecycle.go` |
  | `internal/fabricengine#Fabric.MergeContinue` | `internal/fabricengine/mergelifecycle.go` |
  | `internal/fabricengine#mergeInProgressReason` | `internal/fabricengine/mergeguards.go` |
  | `internal/fabricengine#Fabric.mergeRecordExists` | `internal/fabricengine/mergestate.go` |
  | `internal/fabricengine#Fabric.foreignMergeStatePresent` | `internal/fabricengine/mergestate.go` |
  | `internal/fabricengine#ErrMergeInProgress` | `internal/fabricengine/mergeerrors.go` |
  | `internal/gitrepo#Repo.MergeHeadPresent` | `internal/gitrepo/merge.go` |
  | `internal/mergeresolve#Resolver.Resolve` | `internal/mergeresolve/mergeresolve.go` |

  Every one of these was confirmed present at `72c23d9` by `git show 72c23d9:<file>` during this
  discussion. What was **not** confirmed is that each resolves `found` and unambiguous through the
  facade — that is exactly what D3's step 3 gates, before rep 0.
- **Substitution rule if a glyph does not resolve `found`:** do not weaken the gate and do not edit
  the engine. Replace the offending glyph with another symbol from the same package and the same
  mechanism (candidates in reserve: `internal/fabricengine#Fabric.MergeAbort`,
  `internal/fabricengine#Fabric.mergeStateOrForeignErr`, `internal/gitrepo#Repo.ConflictedFiles`),
  re-run `ladder pack`, and record the substitution and its reason in the task file's notes section.
  Re-run the fasit's cross-check afterwards. This happens before rep 0 or not at all: once rep 1 has
  run, the glyph list is frozen for the whole root.
- **Question text** (goes in the task file's `` ## `<TASK TEXT>` `` blockquote, identical for all
  three arms; the card follows it):

  > This repository distinguishes two different kinds of "a merge is in progress" on a weft: one the
  > fabric layer itself recorded, and one that is merely present in the underlying git checkout
  > because someone ran `git merge` there by hand. Using the symbols listed below, explain how that
  > distinction is drawn and enforced, end to end. Your explanation must cover:
  >
  > 1. Which predicate computes each of the two kinds, what on-disk evidence each one reads, and why
  >    the read-only probe cannot be substituted for the guard the mutating verbs consult.
  > 2. What a sibling mutating verb does while a fabric-recorded merge is in progress, which typed
  >    error carries that refusal, and how that outcome differs from the one produced when only the
  >    foreign git merge state is present.
  > 3. How the command-line layer surfaces the fabric-recorded state, and which of the two predicates
  >    it calls to do so.
  > 4. Where the automated conflict resolver sits in this picture: what it is handed, what it does
  >    when the merge cannot be finished, and whether it participates in the in-progress bookkeeping
  >    at all.

- **Rejected subjects:** config reconciliation against templates (task 06's subject — a fasit already
  exists and the mechanism is contaminated); the shed recipe/plan pipeline (broad but shallow —
  mostly registry lookup, so the answer is closer to enumeration than to tracing).

### D8 — schema, fasit, and the correctness gate

- **Schema:** `exploration`. The answer shape (`relevant_files`, `key_symbols`, `summary`,
  `confidence`, `open_questions`) fits a tracing answer, and `ExplorationRule` already scores it. No
  third schema kind, no new scoring rule, no change to `score.go`.
- **Fasit:** authored the way task 06's was — one dedicated reference pass at the pinned SHA with an
  unbounded budget, cross-checked by a second independent method (`go build ./...` at the pin, plus
  `git show 72c23d9:<file>` for every symbol named), written to
  `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json` with the same `_meta` shape as
  the existing fasits (`task`, `type`, `pinned_sha`, `scope`, `date`, `arm`, `role`, `see_also`).
  `StripFasitMeta` drops `_meta` before scoring, so `_meta` may name quarry freely.
- **Correctness gate:** a rep passes the gate when the scorer returns `summary_matches: true`.
  `recall` and `precision` are recorded and reported per rep, but are **descriptive only and are
  never compared across arms**: e1's pack names the seven files verbatim in the prompt, so its
  `relevant_files` recall is inflated by construction. This is stated in `conclusion.md` as a known,
  unavoidable property of the design, not discovered afterwards.
- **Gate-failure accounting:** a rep that fails the gate stays in the metric sample and is flagged in
  the per-rep table with a gate column. Dropping reps after looking at their scores is optional
  stopping through a side door. If more than 2 of 10 reps in any arm fail the gate, the conclusion
  says so prominently and treats that arm's cost numbers as suspect — but it still reports them.
- **Rejected:** a numeric recall threshold as the gate (confounded by the pack, per above); a new
  `tracing` schema with its own rule (new scorer surface for no measurement gain).

### D9 — the predeclared decision rule, verbatim and locked

Copied from the task proposal and not reinterpreted:

- Primary comparison: **e1-pack vs e0-names**. Primary metrics: `turns` and `cost_usd` (the harness's
  `turns` and `cost_usd` names in `summarize.go`'s `costMetricNames`).
- One-sided Mann–Whitney U, direction e1 lower, **n = 10 per arm**, α = 0.05, **critical U ≤ 27**.
  Reject the null for a metric only when its U is at or below 27.
- n is fixed before rep 1 and never grows after looking at any result.
- `e2-files` runs at the same n and is **secondary/descriptive**: medians and ranges only, no test.
- Secondary observations, reported and never tested: `read_bytes`, wall time (`duration_ms`), and
  recall of the listed symbols in the answer.
- `conclusion.md` shows its arithmetic — per-rep table, rank sums, U values — exactly as
  `results/2026-09-05-ladder-d/conclusion.md` does. The arithmetic is written out by hand; no Go
  statistics code is added.
- A negative answer is a valid, publishable answer. The conclusion is written the same way in either
  direction.

### D10 — `mkdir -p` the results root inside `WriteProvenance`

- **Decision:** `WriteProvenance` calls `os.MkdirAll(resultsRoot, 0o755)` before writing, and the
  roadmap's "Small and independent" bullet for it is struck in the same squash.
- **Rationale:** it is the single write that fails first on a fresh root (`run.go:133`, before any
  `MkdirAll` runs), and putting the fix there also covers the new `ladder pack` command, which writes
  provenance into a root that may not exist yet. Fixing it at the two call sites instead would leave
  the next caller to rediscover it.

### D11 — ladder file parameters

- `bench/loomyard-eval/ladder/ladder-kickstart.yaml`, mirroring `ladder-toc.yaml`'s run parameters so
  the numbers sit on the same scale as the previous roots: `run_model: claude-sonnet-5`,
  `run_effort: medium`, `max_turns: 60`, `scorer: {model: claude-opus-5, effort: high}`,
  `source_repo: env:LADDER_LOOMYARD_REPO`.
- `reps: 10` — the predeclared n.
- No `server:` block and no `quarry_tools:` entries. `validate` does not require either, and
  `needsServer` is false for a matrix in which no cell grants tools. An empty `quarry_tools` also
  removes any chance of `CheckRenderedControlPrompt` firing on an ordinary English word that happens
  to be a tool name.
- New top-level key `pack_targets: []string` — the glyph list `ladder pack` resolves. `validate`
  requires it to be non-empty when any config declares a `card` containing the pack sentinel, and
  requires every entry to be non-empty and unique.
- The single task entry: `07-fabric-merge-state-tracing`, `pinned_sha: 72c23d9ee...` (the full
  40-character SHA; the short form in the proposal is `72c23d9`), `schema: exploration`.
- Three configs, all `allowed: []`: `e0-names` (`control: true`), `e1-pack`, `e2-files`, each with
  its own `card:`.
- Results root: `bench/loomyard-eval/ladder/results/<YYYY-MM-DD>-kickstart/`, the date being the day
  `ladder pack` runs. `raw/` stays untracked; `conclusion.md`, `provenance.json`, `summary.json`,
  `table.txt` and `pack-resolve.json` are tracked, matching the existing roots.

## Technical context

Everything below was read during this discussion; paths are repository-relative.

**The harness**, `bench/loomyard-eval/ladder/`:

- `internal/ladder/config.go` — `Ladder`/`Task`/`Config` structs, `LoadLadder` (decodes with
  `KnownFields(true)`, so a new yaml key must be added to a struct or the file is rejected), and
  `validate`. `retiredKeys` + `wrapRetiredKeyError` name V1 keys explicitly; `pack_targets` and
  `card` are additions, not revivals, so they do not touch that list. `ControlFor` is unused outside
  this file today — D2 does not change that.
- `internal/ladder/prompt.go` — `LoadTaskFile` extracts exactly two things from a task file: the
  blockquote under a heading containing `` `<TASK TEXT>` ``, dedented, and the first fenced JSON block
  after `## Output schema`. Extraction is inclusion-based on purpose, so an answer-key section in the
  task file is never leaked. `RenderPrompt` joins, in order: `PARALLEL_OPENING`, `renderBody`
  (names `targetDir` and the granted tool names), the task text, `PARALLEL_BLOCK`, `closingSentence`,
  the schema block. D1 inserts the card between the task text and `PARALLEL_BLOCK`.
- `internal/ladder/run.go` — `Run` (provenance merge at `:133`, server build at `:162–192`) and
  `runCellRepetition` (`:340` onward: worktree prepare, `LoadTaskFile`, `RenderPrompt` at `:358`,
  the pre-dispatch control-prompt check at `:368`, the attempt loop). `dest` is
  `TaskWorktreePath(worktreeRoot, cfg.Task)` — per task key, which is why D1 matters.
- `internal/ladder/gates.go` — `CheckRenderedControlPrompt` is fatal and pre-dispatch: a control
  cell's rendered prompt may contain neither the bare token `quarry`, nor the server name, nor any
  `quarry_tools` entry, nor the MCP prefix. With D2 this applies to all three e-cells. **The cards
  and the task file must therefore avoid the word `quarry` entirely** — the pack's own content is
  Loomyard paths and Go signatures, so this is a constraint on the prose, not on the pack.
- `internal/ladder/provenance.go` — `Provenance`, `Invocation`, `MergeProvenance` (existing values
  win; a conflicting new invocation is an error), `WriteProvenance` at `:211` (the `mkdir` fix site).
- `internal/ladder/summarize.go` — `costMetricNames` (`turns`, `cost_usd`, `read_bytes`, …),
  `correctnessMetricNames` (`recall`, `precision`), and the rung-vs-control pairing at `:318`, which
  needs exactly one cell per letter flagged `Control`.
- `internal/ladder/metrics.go` — `Metrics`; the relevant fields are `NumTurns`, `TotalCostUSD`,
  `DurationMS`, `ReadBytes`, `ToolUses`, `GrepToolCount`/`BashGrepCount`.
- `internal/ladder/score.go` — `ExplorationRule` (recall/precision/`summary_matches`),
  `ruleBySchema`, `StripFasitMeta`, `RedactAnswer`.
- `cmd/ladder/main.go` — two subcommands today, `run` and `report`, each a `flag.NewFlagSet`. D3 adds
  a third the same way.

**The code under test**, for the pack command only:

- `quarry/repo.go:46` — `func (r *Repo) Resolve(targets []string) ([]ResolveResult, error)`, the
  batched facade entry point.
- `internal/engine/answer.go:163` — `ResolveResult` (`Target`, `ID`, `Status`, `Unit`, `Symbols`,
  `Candidates`, `Dir`, `Error`, `Reason`) and `:57` `Symbol` (`ID`, `Kind`, `File`, `Start`,
  `SigEnd`, `End`, `Signature`, `Doc`). Statuses: `found`, `not_found`, `ambiguous`, `multipart`.
- `quarry/text.go:228` — `RenderResolveText`, the existing renderer. Read for reference; **not** used
  (D5).

**Glyph spelling** (`docs/glyph.md`): `unit#member`, the unit being the repository-relative directory
for Go. Methods are `unit#Owner.Name` with the owner's bare type name, pointer receivers included
(`internal/logger#dualHandler.stderr` is the documented example), and unexported symbols are ordinary
glyphs.

**The target repository:** `LADDER_LOOMYARD_REPO` from `.scratch/ladder.env`
(`/home/knatte/Code/loomyard/wts/loomyard` on this host — the harness resolves the variable itself;
no tracked file may carry that path). Pin `72c23d9ee`, "Surface merge-in-progress in fabric status".

**Precedent to follow for the write-ups:** `results/2026-09-05-ladder-d/conclusion.md` (predeclared
rule restated, per-rep table, rank sums, U, verdict) and
`bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md` + its fasit (task-file section
order, `_meta` shape, the "notes for whoever prepares the fasit" section).

## Constraints

No `CONSTRAINTS.md` at the hub root. From `CLAUDE.md`, the proposal, and the harness's own rules:

- **Go only.** No Python anywhere, in the harness or in any analysis step.
- **Harness rule 1:** never edit the code under test mid-matrix. The quarry commit and dirty flag are
  recorded per invocation in `provenance.json`; the pack is generated once, before rep 0.
- **Harness rule 2:** compare only within one results root.
- **Harness rule 3** (MCP-cell related) does not bite: no cell grants a tool.
- Never edit the Loomyard checkout. The harness restores the pinned worktree between reps and the
  `worktree_dirtied` observation flags any leak.
- No tracked file may carry a machine path. `results/**/raw/` stays untracked; the target repository
  path comes from `.scratch/ladder.env` only.
- Blinding: no rendered prompt in this matrix may contain the bare token `quarry`, the server name,
  or the MCP prefix (D2 extends this from one cell to all three).
- n = 10 per arm is fixed before rep 1. No optional stopping, no post-hoc rep exclusion.
- A negative result is a publishable result.

## Testing

Go tests only, table-driven, alongside the existing `internal/ladder/*_test.go` files. The package
already has `config_test.go`, `prompt_test.go`, `provenance_test.go`, `gates_test.go`,
`summarize_test.go` and an `e2e_test.go` driving `testdata/fakeclaude` — extend those rather than
adding new files where a home already exists.

**TDD candidates** (write the test first; each has an exact, cheap oracle):

- `config_test.go` — `control:` defaulting: unset + empty `allowed` ⇒ control; unset + non-empty
  `allowed` ⇒ not control; explicit `true`/`false` overrides both. `GrantsTools()` independent of
  `control`. `validate` accepts three tool-less configs under one letter with exactly one
  `control: true`, and rejects zero or two. Every existing `ladder-toc.yaml`-shaped fixture still
  loads unchanged (a regression fixture asserting D2's compatibility claim).
- `config_test.go` — `pack_targets` validation: non-empty when a card carries the pack sentinel,
  entries non-empty and unique; unknown-key rejection still fires for a typo'd key.
- `prompt_test.go` — `RenderPrompt` section order with and without a card; a config with no card
  renders byte-identically to today's output (golden string); the card lands after the task text and
  before `PARALLEL_BLOCK`.
- Pack renderer (new `pack_test.go`) — given a fabricated `[]ResolveResult`, the rendered pack matches
  a golden block: order preserved, `Doc` absent, multi-line signatures collapsed. Given one result
  with each of `not_found`, `ambiguous`, `multipart`, and a pre-resolution `Error`, the renderer's
  caller returns an error naming the offending target. This is pure and needs no repository.
- Pack/card round trip (`pack_test.go`) — writing the pack into a card between sentinels is
  idempotent: writing twice yields the same file; a card missing its sentinels is an error, not a
  silent append.
- `provenance_test.go` — `MergeProvenance` carries an existing `kickstart_pack` forward when the new
  invocation has none, and never invents one. `WriteProvenance` into a non-existent results root
  creates it and succeeds (the D10 regression test, red before the fix).
- `run` verification (extend `e2e_test.go` or `prematrix_test.go`) — a card whose pack block has been
  edited after generation fails before any dispatch, with a message naming both hashes; a matching
  pack proceeds.
- `gates_test.go` — `CheckRenderedControlPrompt` still fires for a tool-less non-control cell under
  D2's `!GrantsTools()` gating (the case that would silently stop being checked if the switch were
  made on `IsControl()` instead).
- Dispatch argv (extend `e2e_test.go`'s recorded-`Cmd` assertions, or a focused test over the argv
  builder) — a **tool-less non-control** cell dispatches with **no** `--allowedTools` flag at all,
  byte-identical argv to the control apart from the prompt; a granted cell still gets
  `--allowedTools mcp__quarry__toc`. This is the D2 sweep's `run.go:664` regression, red before the
  fix.
- `mcp_test.go` — `MCPConfigDocument` returns the empty-servers document for a tool-less
  **non-control** cell, and does so with `l.Server == nil` without erroring. This is the D2 sweep's
  `mcp.go:47` regression, red before the fix.
- `summarize_test.go` — a letter with one control and two rungs produces two comparison rows, both
  against the control.

**Not unit-testable, verified by procedure instead:**

- That every glyph resolves `found` — this is D3's pre-rep-0 gate in the live command, against the
  real pinned worktree. The plan must run `ladder pack` and inspect its output before rep 1.
- The question's quality and the fasit's correctness — the reference pass plus `go build ./...` at
  the pin (D8), and this discussion's review.
- The Mann–Whitney arithmetic in `conclusion.md` — recomputed a second time from the per-rep table
  before the conclusion is committed, rank sums checked against `n1*n2 + n(n+1)/2`.

**Explicitly not tested:** the scorer's judgement, the model's answers, and anything requiring a live
API call in `go test`.

## Q&A log

- **Q:** How do the three arms get different prompt text? **A:** [auto-pick] Per-config `card:` file inserted after the task text. **Why:** one task key means one pinned worktree, so `renderBody`'s `targetDir` string stays identical across arms; three task files would differ by an unintended string.
- **Q:** Three tool-less cells collide with "exactly one control per ladder letter" and with `summarize`'s pairing — how? **A:** [auto-pick] Decouple control (baseline + blinding) from grants-tools (server build) via an optional `control:` field and a new `GrantsTools()`. **Why:** blinding should cover all three cells and the server should be built for none; the two meanings coincided only by accident in `ladder-toc.yaml`.
- **Q:** Where does the pack come from and how is "the pack in the prompt is the one provenance records" enforced? **A:** [auto-pick] A `ladder pack` subcommand generates it into the e1 card and provenance; `run` re-verifies the card's block by hash before rep 1. **Why:** a hash check enforces the requirement mechanically instead of trusting an author.
- **Q:** How does the pack record survive `run`'s provenance merge? **A:** [auto-pick] A real `kickstart_pack` field on `Provenance`, carried forward by `MergeProvenance`. **Why:** `run` rewrites `provenance.json` through `MergeProvenance` at startup, so an unknown field would be silently dropped.
- **Q:** Mann–Whitney arithmetic — hand-computed or a new Go subcommand? **A:** [auto-pick] Hand-computed in `conclusion.md`, exactly as ladder-d does. **Why:** precedent keeps the write-ups comparable, and no other consumer needs the code.
- **Q:** What is the question about? **A:** [auto-pick] The weft merge lifecycle: fabric-recorded vs foreign git merge state, 9 glyphs over 4 packages and 7 files. **Why:** the pin is exactly that commit, the answer requires holding several predicates together rather than looking one up, and no existing task or fasit overlaps it.
- **Q:** Which output schema? **A:** [auto-pick] The existing `exploration` schema. **Why:** it fits a tracing answer and adds no scorer surface.
- **Q:** What defines the correctness gate? **A:** [auto-pick] `summary_matches == true`; recall/precision descriptive only and never compared across arms. **Why:** e1's pack names the seven files verbatim, inflating its `relevant_files` recall by construction.
- **Q:** What happens to a rep that fails the gate? **A:** [auto-pick] It stays in the metric sample, flagged in the per-rep table. **Why:** dropping reps after seeing their scores is optional stopping through a side door.
- **Q:** Where does the `mkdir -p` fix go? **A:** [auto-pick] Inside `WriteProvenance`. **Why:** it is the first write to fail on a fresh root and it also covers the new pack command.
- **Q:** What does e2's card contain? **A:** [auto-pick] The same `Uses:` names plus a deduplicated `Files:` list, no spans. **Why:** that is Millhouse's current plan-card format, which is what e2 exists to represent.
- **Q:** Where does the parallel-read instruction live? **A:** [auto-pick] In e1's card only, after the pack block. **Why:** front-loading reads is only possible when locations are known — it is the mechanism under test, not a prompt trick a control could have used.
- **Q:** What exactly does a pack line carry? **A:** [auto-pick] Glyph, file, start-end, and the signature — never the docstring. **Why:** emitting `Doc` would make the treatment "locations plus content" and render a win uninterpretable.
- **Q:** What if a glyph does not resolve `found`? **A:** [auto-pick] Substitute another symbol from the same package and mechanism, re-run `ladder pack`, record the substitution — before rep 0 or not at all. **Why:** weakening the gate or editing the engine mid-matrix would break harness rule 1.
