# 2026-09-02 toc ladder — conclusion

`ladder-toc.yaml` via `run-toc.sh`, reps 5, host OSL-1033 (WSL2), server built from `6ddf737` + the
day's uncommitted harness changes (`provenance.json`). 30 runs, all ingested and scored.

**Three repetitions are invalid and are excluded from every number below.** Both causes are harness
weaknesses, now fixed or gated (see "What went wrong in the harness").

- `a1-toc-file` rep 4: the orchestrator dispatched the run agent with the dispatch *description*
  ("ladderbench run a1-toc-file rep 4 attempt 1") as its whole prompt, not the task prompt. The agent
  spent 15 tool calls reading the harness source and `SKILL.md` to work out what it was supposed to do,
  then did the task. 23 turns, 939k cache_read, recall 0.35.
- `a1-toc-file` rep 5 and `a2-toc-dir` rep 3: the quarry-mcp binary is rebuilt from the working tree
  before every repetition, and the compact toc form (`compact: true`, see `ladder-compact.yaml`) landed
  in that tree at 13:48–13:52 while the matrix was running. Every session prepared after 13:55 (a2,
  b8, b9, and a1 rep 5) ran a server whose toc tools advertised `compact`; in these two reps the agent
  opted in and received the compact text (14 KB instead of 49 KB). Their cost is a compact-form cost,
  not a JSON-form cost. The other post-13:55 reps returned JSON and are kept, with the caveat that
  their tool description differed from the pre-13:55 ones by one sentence.

## Numbers (median [min–max], per results root, this host)

| cell | n | turns | greps | cache_read | cache_creation | duration | recall | precision | Read chars | tool chars |
|---|---|---|---|---|---|---|---|---|---|---|
| a0-none | 5 | 8 [7–11] | 4 [4–6] | 127k [116–180] | 26k [23–33] | 67 s | 0.43 [0.38–0.54] | 0.94 | 35k | 0 |
| a1-toc-file | 3 | 7 [6–8] | 1 [1–3] | 131k [105–160] | 38k [36–40] | 67 s | 0.35 [0.30–0.45] | 0.93 | 29k | 27k |
| a2-toc-dir | 4 | 4 [4–4] | 0 | 83k [82–106] | 41k [14–42] | 53 s | 0.42 [0.38–0.47] | 0.94 | 41k | 49k |
| b0-none | 5 | 6 [4–7] | 3 [2–4] | 90k [49–105] | 16k [0–21] | 22 s | 1.0 | 1.0 | 0 | 0 |
| b8-toc-dir | 5 | 4 [4–7] | 2 [1–5] | 77k [48–122] | 23k [12–42] | 29 s | 1.0 | 1.0 | 22k | 11k |
| b9-toc-file | 5 | 5 [4–5] | 1 [0–1] | 100k [49–103] | 51k [44–55] | 31 s | 1.0 | 1.0 | 70k | 25k |

"Read chars" is the total size of `Read` tool results in the transcript; "tool chars" the total size
of quarry tool results. `output_tokens` is unusable on this host (see the 2026-09-02-followup
conclusion) and is not shown.

## Ladder a — toc dir's August result reproduces; toc file does not help

**a2-toc-dir** is the August finding again, now at n=4 on a different host: half the turns (4 vs 8),
zero greps instead of 4, cache_read down a third (83k vs 127k), recall and precision unchanged
(0.42 vs 0.43, 0.94 vs 0.94). Every rep made exactly one `toc_dir` call on the two packages (49 KB of
JSON), then read 5–6 files. The mechanism is visible in the transcripts: the control spends its first
three or four turns on `ls` + grep + reading `doc.go` to build the same file map the toc hands over
in one call. Note what the toc did *not* change: Read volume (41k vs 35k chars) — the agent reads the
same amount of real code either way. The saving is the discovery rounds, not the reading.

**a1-toc-file** does not separate from the control: 7 turns vs 8, cache_read 131k vs 127k, recall
0.35 vs 0.43 (worse, n=3, inside the spread). The agent has to guess which files to `toc_file` before
it has a map, so it either calls it on 7–8 files at once (25 KB, most of it for files it never opens)
or on the wrong ones and then greps anyway. toc file is a second-step tool; granted alone it is a
worse first step than `ls`.

Recall is 0.35–0.54 in every cell, with the spread inside each cell as wide as the gap between cells.
Five reps did not tighten that; the exploration fasit's 20 files and ~30 symbols leave a lot of room
for scoring variance. Toc does not change correctness on this task in either direction.

## Ladder b — the negative control behaves as one

With the file named in the task, `toc_dir` (b8) overlaps the control on every metric: cache_read 77k
vs 90k with ranges 48–122k vs 49–105k, turns 4 vs 6, greps 2 vs 3. Every rep still called the tool
once (11 KB), and then did what the control does: grep for `Run(`. Nothing gained, one call's worth
of prompt cost paid (cache_creation 23k vs 16k).

`toc_file` (b9) is worse than the control: the agent `ls`es the package and calls `toc_file` on all
9–10 non-test files (25 KB), then reads 70 KB of source (three whole files) in every rep — more than
any other cell — and cache_creation is 51k vs 16k. The map of the named file adds nothing the task text
did not already give; the map of the whole package costs a round and 25 KB. Precision and recall are
1.0 everywhere; correctness on task 04 has never moved.

## What this settles

1. **toc dir is a first-step map, and only that.** It pays when the agent does not know where to
   look (ladder a) and not at all when it does (ladder b). This is the second independent measurement.
2. **toc file alone is not a rung worth keeping.** In both ladders it made the agent read more, not
   less.
3. **Cost, not correctness.** Recall did not move in any cell of either ladder. The claim to make for
   toc is "same answer for two-thirds of the tokens and half the turns on unfamiliar code".
4. The compact form (`ladder-compact.yaml`) and injection (`ladder-annex.yaml`) are the two ways to
   make the toc-dir win cheaper still; the two accidental compact reps here (a1/5, a2/3) are not
   evidence either way and were excluded.

## What went wrong in the harness

- **No prompt check on the run agent.** Ingest gates check tools, model, turns, blinding, and the
  worktree, but not that the subagent was given the task prompt. a1 rep 4 passed every gate. Fixed:
  `GateRunPrompt` is fatal when the transcript's first user message is not the generated prompt.
- **The server under test is rebuilt from the live working tree per repetition.** Editing quarry
  source while a matrix runs changes the thing being measured mid-matrix, silently. Fixed in
  `runmatrix`: the server binary is hashed after every `prepare-session` and recorded per rep in
  `provenance.json`; the final table refuses to print a clean summary and flags the root when more
  than one distinct binary was used. The operator rule is in `run.sh`'s banner: do not edit quarry
  source while a matrix is running.
- **The orchestrator wrote the outcome marker with the Write tool** (a1 rep 3), which is not on the
  allow list, and the matrix waited on a permission prompt until the operator noticed. Fixed in
  `SKILL.md` (Bash only for the marker).

## What to do

- Invalidate and re-run `a1-toc-file` reps 4 and 5 and `a2-toc-dir` rep 3 with
  `ladderbench invalidate` followed by `run-toc.sh` against this root, if a clean n=5 is wanted; the
  conclusions above do not depend on it.
- Run `run-compact.sh` and `run-annex.sh`, in that order, each with a clean working tree.
