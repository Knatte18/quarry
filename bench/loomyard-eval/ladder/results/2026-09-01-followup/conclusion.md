# 2026-09-01 follow-up matrix — conclusion (written 2026-09-02, from the committed summary only)

**Superseded: re-run on 2026-09-02, see `../2026-09-02-followup/conclusion.md`.**

**Verdict: this run does not verify the fixes it was dispatched to verify. Its committed numbers must
not be read as "b1 fixed, b2/b4 worse".** Two of the three cells are unusable as evidence for the
question HANDOFF.md §1 asked, and the third is a genuine but unexplained signal. Re-run it.

The raw per-run data (`raw/`, gitignored) lives only on the machine that ran the matrix; everything below
is derived from `summary.json` alone. The `raw/` tree of this results root is what settles the open
points, and it should be pulled into the repo before anything else is concluded from this run.

## Was the fixed server actually under test?

Probably yes, but it is not recorded anywhere. `1592f4e` (`within` scoping) and `ee84d8d` (symbol-form
steering) are not ancestors of `main` by hash; they were squash-merged as `4ccabc1`, whose tree is
byte-identical to the branch tip (`git diff fa46984 4ccabc1` is empty), and `main`'s
`internal/mcpserver/tools_symbol.go` carries the per-entry `within` field and the new descriptions.
`prepare-session` rebuilds `quarry-mcp` from the running checkout on every call, so a checkout at
`d3dde00` or later built a server with both fixes. What no artifact records is the built binary's
`vcs.revision`/`vcs.modified`, so a dirty or stale checkout on the run machine cannot be ruled out from
the committed data. `runmatrix --all` now writes `provenance.json` into the results root to close this.

## b1-symbol — the "fix" was never exercised

`quarry_tool_uses` is **0 in both repetitions** (`stats.quarry_tool_uses` median/min/max all 0; the
August cell had exactly 1 in every rep). The agent never called `workspace_symbol`, so `within` scoping
was never on the wire. The cell's lower cost (cache_read 107k → 78k, duration 33.6s → 26.1s) is the
cost of *not using the tool*, i.e. this cell behaved as a `b0-none` with an extra tool schema in the
prompt — and its `cache_creation_input_tokens` (38k vs the control's 16k) is that schema's cost.

Most plausible cause: the rewritten `workspace_symbol` description now says it is "a discovery tool for
when a symbol's file is not yet known — once it is, prefer textDocument_definition/textDocument_references",
and task 04 names the target file. The description steers the agent away from the only tool it was
granted. That is a finding about the description, not a verification of `within`.

## b4-lsp-trio — one of the two runs is an August transcript

One followup repetition matches an August repetition on **every** usage metric simultaneously:

| metric | August rep (max of n=3) | followup rep |
|---|---|---|
| num_turns | 10 | 10 |
| tool_uses | 11 | 11 |
| quarry_tool_uses | 5 | 5 |
| input_tokens | 20 | 20 |
| output_tokens | 3162 | 3162 |
| cache_creation_input_tokens | 40299 | 40299 |
| cache_read_input_tokens | 262562 | 262562 |
| duration_ms | 50478 | 50478 |

Wall-clock identical to the millisecond is not two independent runs; it is the same transcript ingested
into two results roots. The other followup repetition (13 turns, cache_read 260743, 49146 ms) matches
nothing in August and is presumably genuine, so this cell is n=1 new data plus one stale row. The
"substantially worse" reading in commit `6ddf737` rests on that.

How it happened is not determinable from here: session-dir templates were distinct in every committed
version (`{config_id}-{n}` vs `followup-{config_id}-{n}`), so `LocateTranscript`'s per-project glob
should not have crossed roots. Check on the run machine: `raw/b4-lsp-trio/*/transcript.meta.json`,
`ingest.json` (attempt index), and the mtimes of `raw/b4-lsp-trio/*/transcript.jsonl` against the
August root's.

## b2-definition — genuine runs, worse, cause unknown

No metric value coincides with August, so both repetitions look new. The agent made more server calls
(2–3 vs 1–2) and used more turns (9–11 vs 7–9). Consistent with — but not proof of — the new
description's steering toward bare-name `symbol` addressing producing an `ambiguous` status on a name
like `Run` (many `Run` methods at the pin) and a retry cycle that position addressing did not have.
Only the transcripts can confirm this. Note also that `duration_ms` grew more than turns did
(46–60 s for 9–11 turns vs 33–37 s for 7–9), which may be API latency on the day; the control's
duration was flat (20.3 s → 22.4 s), so the effect is not purely environmental.

## Control

`b0-none` reproduces August within range on every metric. Recall and precision are 1.0 in every cell of
both runs, as expected; this run was about cost only.

## What to do

1. Pull `raw/` from the run machine into this results root (and stop gitignoring `raw/` — the task-05
   tree is 1.8 MB for 12 runs; the transcripts are the only thing that can answer any of the above).
2. Re-run the three cells with `bench/loomyard-eval/ladder/run-followup.sh` against a fresh results
   root; the printed table flags any tool-granted cell whose agent never used its tool.
3. Decide whether the `workspace_symbol` description should stop steering away from itself when it is
   the only tool granted — otherwise a b1 cell can never measure `within`.
4. Deny-list probe: `denial_shape_observed` came back `TOOL-NOT-IN-SCHEMA`, which does not match
   `DenialShapePattern`; `denied_tool_attempts` stays provisional (unchanged from commit `6ddf737`).
