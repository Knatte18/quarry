# HANDOFF — next steps after the 2026-09-01 quarry-improvement research

Context for a fresh session on any machine. The research report this plan executes on is
committed at `docs/research/quarry-improvement-research.md` (conclusions, evidence, external
references); the state it starts from: the 2026-08-30 45-run matrix and the 2026-09-01 task-05
ladder are both run, scored, and committed (`bench/loomyard-eval/ladder/results/`);
`ladder-followup.yaml` (the fix-verification matrix) has **never** been run — no results root, no
`followup-*` session dirs, and its yaml still carries stale paths from the pre-WSL2 machine.

Everything below runs on different machines. **Nothing may hardcode a machine path** — not to the
Loomyard checkout, not to this repo's own absolute location. Machine-specific locations come from
local environment variables (or are derived at runtime from the repo the command runs in), never
from tracked files.

## 0. Portability prerequisite — de-hardcode the ladder configs (BLOCKER for 1–3)

Current state, verified 2026-09-01: all three ladder yamls hardcode absolute machine paths.

- `ladder.yaml`: `session_dir_template: /home/knatte/...`, `source_repo: /home/knatte/...`
- `ladder-followup.yaml`: same two fields, also `/home/knatte/...`
- `ladder-task05.yaml`: same two fields, `/home/hanf/...`
- Task files `04-*.md` / `05-*.md`: setup sections embed `git -C /home/hanf/Code/loomyard/...`
  worktree commands.

Required change, in `bench/loomyard-eval/ladder/internal/ladder` config loading:

1. **One required env var: `LADDER_LOOMYARD_REPO`** — absolute path to the local Loomyard
   checkout. The yaml `source_repo` field becomes the literal string `env:LADDER_LOOMYARD_REPO`
   (or is dropped and always resolved from the env var — pick one, document it in the yaml
   comment). Resolution happens at config load; an unset or non-existent path fails loud in
   `RequirePins`-style validation, before any session is prepared. Do **not** name it `QUARRY_*`:
   the harness deliberately clears/manages `QUARRY_STATE_DIR`/`QUARRY_BUILD_TAGS`/`QUARRY_CONFIG`
   at three application points (see ladder README "Run environment"), and a fourth `QUARRY_`-
   prefixed variable invites confusion with that machinery.
2. **`session_dir_template` becomes repo-root-relative.** The loader resolves the running quarry
   repo root (`git rev-parse --show-toplevel` from the config file's own directory, or walk up
   from the yaml path) and joins a relative template, e.g.
   `.scratch/ladder-sessions/{config_id}-{n}`. This preserves the README's hard constraint that
   session dirs sit under an already-trusted ancestor, never bare `/tmp` — the constraint is
   about *which tree*, not which absolute string.
3. **Target worktree paths (`/tmp/loomyard-eval-*`, `cold_worktree_template`) may stay `/tmp`** —
   they are the *target* workspace, not the session dir, are machine-neutral as written, and the
   trust rule does not apply to them. Leave as is.
4. **Task files**: rewrite the setup commands to use `"$LADDER_LOOMYARD_REPO"` instead of a
   literal checkout path (the pinned SHA stays literal — that is the point of a pin).
5. Update the ladder README's "How to run" for the new env var, and add a unit test that a config
   with an absolute `session_dir_template` or a literal home-directory `source_repo` is rejected,
   so the hardcoding cannot come back.

## 1. Run `ladder-followup.yaml` (fix verification) — first run after 0 lands

What it is: a targeted before/after re-test of the three 2026-08-30 configs whose behavior was
driven by since-fixed quarry-mcp defects, on task 04, `reps: 2`, own results root
(`results/<YYYY-MM-DD>-followup`), config ids intentionally identical to the originals for
by-id comparison across results roots. Never point it at the main matrix's results root.

- **b1-symbol**: unscoped `workspace_symbol` saturated gopls's 100-hit cap (~25.6k chars of
  noise). Fix under test: `within` scoping (commit 1592f4e).
- **b4-lsp-trio**: the same noise compounding with the other two LSP tools. Same fix.
- **b2-definition**: 0-based position addressing caused observed friction (a mis-aimed batch, an
  awk column hunt, a retry). Fix under test: tool descriptions steering to symbol-form
  addressing (commit ee84d8d).
- **b0-none**: fresh control.

Also owned by this run (per the ladder README "Metrics"): dispatch the **deny-list probe**, capture
the real denial record's text as `denial_shape_observed` in `probe.json`, validate
`DenialShapePattern` against it, and clear the `denied_tool_attempts_provisional` marker.

Success criteria: cost-shaped metrics (turns, tokens, duration) for b1/b4/b2 fall to overlap with
the control's ranges. Correctness is expected unchanged — it was already perfect in August; this
run settles whether the LSP rungs' *cost* was a bug artifact or inherent. Write a short
`conclusion.md` into the results root either way.

## 2. Two cheap discriminating arms (the highest-value new measurements)

Both are motivated in the research report §1 conditions (2)/(3) and §6; run them before building
any new task.

### 2a. The "annex" arm — quarry as pre-processing, no tools granted

New config shape: the run agent gets **zero** quarry tools, but `prepare-session` mechanically
pre-computes quarry output for the task's target and injects it into the session as an attachment
the prompt references (e.g. the `impact` result for the target symbol on an impact task; `toc_dir`
for the scoped packages on an exploration task). Harness work:

- A config field (e.g. `annex: impact` / `annex: toc`) driving a generation step in
  `prepare-session` that shells the quarry CLI (not MCP) against the pinned worktree and writes
  the result into the session dir.
- Blinding care: the annex text must not name quarry or any tool — present it as a neutral
  "pre-computed analysis attachment"; run it through the same redaction vocabulary before it
  enters the session, and note in the results that annex arms are steering-confounded like every
  rung (README "Design rationale", preamble confound).

This tests the integration Loomyard would actually ship (mechanical pre-processing — see §6 of the
research report) with the existing scoring machinery: does pre-computed verified truth at zero
agent-side tool cost beat both the control and the tool-granted rungs?

**Known negative prior, operator-reported (no committed writeup found):** an earlier informal
bench injected a compact TOC into a *reviewer's* prompt and found little to no gain — consistent
with the committed task-03 scorecard (a baseline reviewer won decisively; the diff already tells
a reviewer where to look, and the compiler is ground truth for resolution breaks). The trace of
that experiment is `bench/loomyard-eval/scripts/gen_compact_toc.py`. Consequence: scope the annex
arm to **plan-time/implementer-shaped injection** (an `impact` annex when the task names symbols
to change; a toc pack for *unfamiliar-code exploration*, where the toc win was actually
measured), not reviewer prompts — and this run is the documented re-test of the injection
question, so record its results either way.

### 2b. The weak-model arm

Same task, at minimum `c0-none` + `c1-impact` (task 05 configs), with
`run_model: claude-haiku-4-5-20251001`, own results root. External evidence (report §3, ref [1])
says LSP tools helped only the weakest model tested; the ladder has only ever run sonnet-class.
Correctness comparison across model arms is allowed; cost-shaped numbers stay per-results-root as
always. If tools rescue a weak model's correctness, the deployment guidance inverts (grant tools
to cheap agents), even though the sonnet-class ceiling stands.

## 3. Fourth-round task designs (design committed, build gated on 2's outcome)

Specs live in the research report §4; the compiler-fasit bar is `go vet ./...` after a signature
patch (not `go build` — vet type-checks `_test.go` files, and test call sites are part of the
honest answer; see §4 below).

- **Task 06** — interface-method impact at implementer scale: `shedengine.ShedProducer.Call`,
  ~13 implementers across ≥4 packages, whole-repo scope, wrapper chains. Recall-at-scale, the
  untested direction. Expected outcome is honestly still "no separation" (Loomyard's `var _`
  assertion discipline is a grep anchor) — the run's value is establishing that codebase
  *discipline*, not agent skill, is what closes the gap.
- **Task 07** — whole-repo, no-scope-hint, high-dispersion method impact (survey procedure in the
  report; needs a ≥25-caller method with same-named decoys, found at the pin).
- **Task 08** — structurally invisible references (embedded-type promotion / method values;
  candidate symbols and the required build-time survey are in the report).

Build 06 first; 07/08 only if 06 or the arms in §2 show a thread worth pulling.

## 4. Documentation corrections (independent — do immediately, no run needed)

- **`docs/when-to-use-quarry.md`**: the task-05 characterization ("matched the no-tool control's
  perfect recall/precision") does not match the committed data. `results/2026-09-01-task05/
  summary.json` records precision **0.133** (recall 1.0) in every cell: every agent in every
  config listed the 13 `mergeresolve_test.go` call sites in `callers_to_update` because the task
  text asks for sites that "would need a `reason` argument threaded in to keep compiling" (which
  includes tests) while the fasit's scoring notes exclude tests. Correct the doc to "identical
  recall/precision across all cells; precision uniformly deflated by a task-text/fasit mismatch
  on test files". The comparative no-separation conclusion stands.
- **Task-authoring rule going forward** (add to the ladder README or the tasks' shared
  conventions): every impact task must state explicitly whether `_test.go` call sites belong in
  `callers_to_update`, and fasits are generated with `go vet ./...` so test sites are enumerated.

## 5. Quarry engine/MCP changes (independent of any benchmark round)

Ranked list with rationale in the research report §5. Short form:

1. `rename_impact` tool wrapping LSP `textDocument/rename` — the fasit procedure as a product.
2. Expose `implementations` (`textDocument/implementation` is already wired internally).
3. `within` scoping default-on (workspace-wide becomes the explicit opt-out).
4. Per-entry `verified: true|false|"partial"` on refs/impact — **prerequisite for §6**; today
   verification can silently skip while the entry still says `resolution: complete`.
5. Unify all tools to 1-based positions at the MCP boundary.
6. A compact (grep-shaped) output form to cut per-call token cost.
7. Warm-daemon support beyond Go, or document the other languages as cold-spawn-only.

## 6. Loomyard mechanical integration (after 5.4 lands)

The direction the evidence actually supports (report §6): quarry called mechanically from
Loomyard's own Go code via the `quarry/` facade — plan-time impact annexes injected into
implementer prompts, toc context packs for unfamiliar-code exploration, and deterministic
`verify:` gates on Deletes/Moves. (Review-time diff-impact annexes are deliberately dropped from
this list: an earlier informal bench found injecting a TOC into a reviewer prompt gained little —
see the negative prior under §2a — and for compiling Go the compiler already polices resolution
breaks; only re-add review-shaped annexes if the §2a re-test surprises.) The gates are explicitly blocked on §5 item 4: a gate consuming
unverified results marked `resolution: complete` re-creates the 31-false-positive incident
(`docs/research/scout-agent-usage-findings.md`). The annex arm (§2a) is the measurement of this
integration's value.

## Dependency order

```
0 (de-hardcode)  ──► 1 (followup run) ──► 2a/2b (annex + weak-model arms) ──► 3 (task 06+)
4 (doc fixes)    — independent, do now
5 (quarry changes) — independent of runs; 5.4 gates 6
6 (Loomyard mechanical) — after 5.4; measured by 2a
```
