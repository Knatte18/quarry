# 2026-09-06 kick-start pack matrix (M4) — conclusion

`ladder-kickstart.yaml` via `go run ./bench/loomyard-eval/ladder/cmd/ladder run --config
bench/loomyard-eval/ladder/ladder-kickstart.yaml --results bench/loomyard-eval/ladder/results/2026-09-06-kickstart`,
no `--cells`, no `--reps`. Host `Neve` (per `provenance.json`'s `hostname` field — the shell's own
`hostname` reports the worktree name in this sandbox and is not used here), branch
`kickstart-matrix-run`, quarry commit `33e09356a131d79c7b7f9a070761f5a1a5480b59` (the run
invocation's own `quarry_commit`), server name `quarry` with no server built (`server_hashes: {}` —
no cell in this matrix grants tools). `provenance.json.invocations` has three entries: invocation 0
and invocation 1 are card 29's two `pack` calls (the first ran against a dirty tree because the
implementer brief markdown had not yet been committed; the second, after that commit, is the one
whose `pack-resolve.json` and treatment-card block are what this root carries); invocation 2, written
2026-09-06T09:56:18Z with `reps_effective: 10` and `quarry_dirty: false`, is this card's single `run`
call. The matrix was **not resumed** — `LADDER_EXIT=0` on the one and only launch, no abnormal
process death, no stale lock. 30 measured repetitions across three arms (`e0-names`, `e1-pack`,
`e2-files`) at task 07, all ingested and scored: `unscored_count: 0` in every cell, `recall_n` and
`precision_n` are 10 in every cell per `table.txt`.

This root stands alone. It is not pooled with `results/2026-09-05-ladder-d`,
`results/2026-09-04-toc` or `results/2026-09-04-breadth` — different task, different manipulation
(pre-resolved glyph spans in the prompt, not a granted `toc` tool), different results root — and no
number below is compared against them.

## Numbers (median [min–max], this results root only)

Every figure is quoted verbatim from `summary.json`'s per-cell `metrics` block or `table.txt`'s
`recall_n`/`precision_n` columns; none is re-derived. Column names are `table.txt`'s; `summary.json`
spells four of them differently: `cache_read` is `cache_read_input_tokens`, `cache_creation` is
`cache_creation_input_tokens`, `prefixed_tool_uses` is `quarry_tool_uses`, and `grep_fallback` is
`grep_fallback_total`.

| metric | e0-names (control) | e1-pack | e2-files |
|---|---|---|---|
| turns | 18 [14–19] | 10 [9–13] | 10 [10–12] |
| duration_ms | 59234 [49381–118036] | 49132 [41870–58486] | 46123 [38028–53245] |
| cost_usd | 0.4325 [0.264–0.5267] | 0.3004 [0.2558–0.4161] | 0.4171 [0.4116–0.4609] |
| cache_read | 189309 [96953–369443] | 115443 [101439–168417] | 167430 [164774–297549] |
| cache_creation | 46897 [19780–57382] | 30702 [24230–49661] | 49962 [48135–51766] |
| output_tokens | 27.5 [8–71] | 10 [6–14] | 9.5 [6–27] |
| input_tokens_total | 231635 [125423–423587] | 146735 [138062–218086] | 217403.5 [213470–348082] |
| tool_uses | 17 [13–18] | 9 [8–12] | 9 [9–11] |
| prefixed_tool_uses | 0 [0–0] | 0 [0–0] | 0 [0–0] |
| grep_fallback | 6 [5–8] | 1 [1–2] | 1 [1–3] |
| tool_result_bytes | 105895 [39286–125995] | 67595 [53441–118272] | 118609 [114706–119724] |
| read_bytes | 92463 [30651–106445] | 59750 [43232–113224] | 114540 [112470–114910] |
| recall (n) | 0.93 [0.85–1] (n=10) | 0.86 [0.8–0.94] (n=10) | 0.89 [0.85–0.94] (n=10) |
| precision (n) | 0.85 [0.79–0.92] (n=10) | 0.825 [0.72–0.94] (n=10) | 0.895 [0.78–0.94] (n=10) |

All three cells realised the full n=10; no arm's realised n falls below its predeclared count, so the
primary test below is **not void**. Per arm, unconditionally:

| arm | realised n | turn-ceiling count (`max_turns_count`) | unscored count |
|---|---|---|---|
| e0-names | 10 | 0 | 0 |
| e1-pack | 10 | 0 | 0 |
| e2-files | 10 | 0 | 0 |

A clean ten-of-ten matrix in every arm on both counts. `blinding_failed_count` is 0 in every cell and
neither the `incomplete` nor the `invalid` list in `table.txt`/`summary.json` names any repetition.

## The predeclared rule, restated

Transcribed, not reinterpreted, from the overview's Shared Decisions and the frozen batch spec:
primary comparison **e1-pack against e0-names** (control); primary metrics **turns** and **cost_usd**;
one-sided Mann–Whitney U with the alternative that the treatment (`e1-pack`) is lower; **ten** per arm;
**alpha 0.05**; **critical U at or below 27**; reject the null for a metric only when its U is at or
below that value. `e2-files` is the descriptive arm: medians and ranges only, no test.

### Per-repetition values (from `raw/<cell>/<rep>/usage.json`)

| rep | e0-names turns | e1-pack turns | e0-names cost_usd | e1-pack cost_usd |
|---|---|---|---|---|
| 1 | 18 | 10 | 0.5210 | 0.2793 |
| 2 | 16 | 10 | 0.4394 | 0.4107 |
| 3 | 18 | 11 | 0.2640 | 0.3214 |
| 4 | 14 | 9 | 0.2756 | 0.3909 |
| 5 | 19 | 10 | 0.4449 | 0.4161 |
| 6 | 15 | 13 | 0.4710 | 0.2694 |
| 7 | 19 | 11 | 0.2976 | 0.2558 |
| 8 | 19 | 10 | 0.5267 | 0.2574 |
| 9 | 15 | 9 | 0.4140 | 0.3835 |
| 10 | 18 | 11 | 0.4256 | 0.2610 |

### Metric 1 — turns

Combined N = 20, ranks 1…20, ties averaged. `e0-names` values: 14,15,15,16,18,18,18,19,19,19.
`e1-pack` values: 9,9,10,10,10,10,11,11,11,13.

Ascending order with midranks: 9→1.5 (×2), 10→4.5 (×4), 11→8 (×3), 13→10, 14→11, 15→12.5 (×2),
16→14, 18→16 (×3), 19→19 (×3).

`e1-pack` holds two 9s, **four** 10s, three 11s, one 13:

    R1 = 2(1.5) + 4(4.5) + 3(8) + 1(10) = 3 + 18 + 24 + 10 = 55

Total rank sum N(N+1)/2 = 20·21/2 = 210, so R0 = 210 − 55 = 155 (checked directly against
`e0-names`'s own values: 10 has one 14, two 15s, one 16, three 18s, three 19s →
11 + 2(12.5) + 14 + 3(16) + 3(19) = 11 + 25 + 14 + 48 + 57 = 155 ✓).

    U1 = R1 − n1(n1+1)/2 = 55 − 55 = 0
    U0 = R0 − n2(n2+1)/2 = 155 − 55 = 100       (U1 + U0 = 100 = n1·n2 ✓)

**U1 = 0 ≤ 27 → reject the null at α = 0.05.** `e1-pack` runs fewer turns than `e0-names`.

### Metric 2 — cost_usd

No ties (twenty distinct values). Combined ascending order (all twenty values, `a`=`e1-pack`,
`b`=`e0-names`, rank in parentheses): 0.2558a(1), 0.2574a(2), 0.2610a(3), 0.2640b(4), 0.2694a(5),
0.2756b(6), 0.2793a(7), 0.2976b(8), 0.3214a(9), 0.3835a(10), 0.3909a(11), 0.4107a(12), 0.4140b(13),
0.4161a(14), 0.4256b(15), 0.4394b(16), 0.4449b(17), 0.4710b(18), 0.5210b(19), 0.5267b(20).

`e1-pack` takes ranks 1, 2, 3, 5, 7, 9, 10, 11, 12, 14:

    R1 = 1+2+3+5+7+9+10+11+12+14 = 74

`e0-names` takes the remaining ranks 4, 6, 8, 13, 15, 16, 17, 18, 19, 20:

    R0 = 4+6+8+13+15+16+17+18+19+20 = 136   (R1 + R0 = 74 + 136 = 210 = N(N+1)/2 ✓)

    U1 = R1 − n1(n1+1)/2 = 74 − 55 = 19
    U0 = R0 − n2(n2+1)/2 = 136 − 55 = 81         (U1 + U0 = 100 ✓)

**U1 = 19 ≤ 27 → reject the null at α = 0.05.** `e1-pack` costs less than `e0-names`.

Both identities hold at the realised full size: R1 + R0 = 210 = N(N+1)/2, and U1 + U0 = 100 = n1·n2,
for both metrics. Both predeclared primary metrics reject the null in the predicted direction.

### Correctness, for completeness

Recall's median fell (0.86 vs 0.93) and precision's median fell slightly (0.825 vs 0.85) for `e1-pack`
against the control, both descriptive comparisons only (see the recall/precision caveat below), and
neither range excludes the other (`e1-pack` recall [0.8–0.94] vs control [0.85–1]; precision
[0.72–0.94] vs [0.79–0.92]) — no degradation is asserted from these overlapping ranges, and none
would rescue or reverse a result the two primary metrics already settle.

## Verdict: e1-pack separates on both primary metrics

Applying the predeclared rule verbatim: `e1-pack` vs `e0-names` rejects the null at α = 0.05 for
**both** turns (U = 0) and cost_usd (U = 19), both well inside the U ≤ 27 threshold and both moving in
the predicted direction (treatment lower). The surface has now been measured from the push-mode
direction — glyph spans pre-resolved into the prompt before the agent starts, rather than left for a
mid-session tool to resolve — and it separates. The win here belongs to pre-resolution itself, not to
a tool invoked mid-session: `e1-pack`'s prompt already carries the nine spans and their signatures, so
the agent never needs to look them up, which is a different product from the mid-session `toc`/`glyph`
tool measured (and not separated) in the ladder-a/b/c/d roots.

One sentence on the M4b condition: because `e1` separates here, the M4b edit-task variant (an agent
revising code in a throwaway worktree, starting from a pre-resolved pack rather than from a granted
tool) now becomes a candidate follow-up, with this results root as its justification; card 32 adds it
to the roadmap.

## Recall and precision are descriptive only, never compared across arms

Both non-control arms have file recall inflated by construction: the treatment card (`e1-pack`) names
the seven fasit-relevant files verbatim inside its pack block, and the descriptive card (`e2-files`)
names the same seven as a plain list. No contingency substitution triggered in card 29 — all nine
`pack_targets` resolved `found` on the first enumeration — so both non-control cards name all seven of
the fasit's `relevant_files`, not six of seven; there is no vacated file to account for. The control
(`e0-names`) is the only arm whose file recall is earned by search rather than handed to it, which is
why its own recall (0.93 median) running highest of the three is not read as a mechanism finding: it
is the one arm without the inflation the other two share.

The other known asymmetry: `e1-pack` minus `e2-files` is not a clean spans-vs-names contrast, because
the treatment card also carries the resolved signature text and a "read all listed spans in parallel"
instruction that the plain-files card does not. The turns/cost separation measured above is `e1-pack`
against the control only; `e2-files` is descriptive and any `e1` vs `e2` gap conflates the signature
and the parallel-read instruction with the spans-vs-names difference and is not decomposed here.

## Secondary observations (reported, never tested)

- **`e2-files` (descriptive) also separates from control on turns and tool_uses** by the range-overlap
  heuristic in `table.txt` (`separated: true`), consistent with `e1-pack`'s tested result, though
  `e2-files` carries no significance test under the predeclared rule.
- **Read bytes and wall time (`duration_ms`)** run lower for both non-control arms than for the
  control (e1-pack 59750 vs 92463; e2-files 114540 vs 92463 — `e2-files` in fact reads *more* bytes
  than the control despite fewer turns, consistent with its cards' plain file list steering the agent
  straight to full-file reads rather than spans), but neither is a primary metric and neither is
  tested.
- **Recall of the fasit's nine listed key symbols in the agent's own answer**, tallied by matching each
  answer's `key_symbols[].name` against the fasit's nine `key_symbols` names (a hand tally over all 30
  `answer.json` files, not a scored field): `e0-names` named a median of 8 of 9 (range 6–9), `e1-pack` a
  median of 7 of 9 (range 6–8), `e2-files` a median of 7 of 9 (range 7–8). This is descriptive only —
  it is not the fasit's `relevant_files` recall/precision score above, and it is not tested.

## Coverage

- **No permitted repetition deletion.** Card 30 deleted no repetition directory on any branch — the
  run completed on its first and only launch, so the memory-path-taint deletion-avoidance branch was
  never reached. This root makes the same "no permitted repetition deletion" assertion the reference
  root (`2026-09-05-ladder-d`) makes.
- **Three invocations, but the run itself was not resumed.** The provenance record's `invocations` has
  three entries: invocations 0 and 1 are card 29's two `pack` calls (recorded in `provenance.json`
  before this card ever launched `ladder run`); invocation 2 is this card's single `run` call, which
  completed with `LADDER_EXIT=0` on its first attempt. No abnormal process death occurred and no
  resume precondition was ever walked for the run itself.
- **`quarry_dirty: true` on two invocations, both pre-dating the run and both harmless.** `Pack` calls
  `CollectInvocation` — which polls `git status --porcelain` — before it writes the card,
  `pack-resolve.json` or `provenance.json`, so each invocation's recorded dirty files describe the tree
  as it stood *before* that invocation wrote anything of its own. Invocation 0 recorded
  `quarry_dirty_files: ["_mill/briefs/implement-matrix-and-writeup-r1.md"]` — the injected implementer
  brief, mill orchestrator bookkeeping unrelated to the harness or the target repository — and then
  went on to write the card, `pack-resolve.json` and the provenance record itself, uncommitted.
  Invocation 1 recorded
  `quarry_dirty_files: ["bench/loomyard-eval/cards/07-e1-pack.md", "bench/loomyard-eval/ladder/results/2026-09-06-kickstart/"]`
  — not that invocation's own just-written pack outputs (its `CollectInvocation` poll ran before it had
  written anything), but invocation 0's completed, still-uncommitted outputs left over on the tree when
  invocation 1 launched. So the `commit-clean-before-each-harness-invocation` Shared Decision was
  missed twice in a row — before invocation 0 (dirty on the mill brief) and again before invocation 1
  (dirty on invocation 0's uncommitted output) — not once. Neither invocation's dirty path is code the
  harness or the target repository reads, so neither could affect the measurement, but neither pack
  invocation actually satisfied the preflight commit-clean step. Only invocation 2 — the `run` call
  this root measures — recorded `quarry_dirty: false`, matching the Shared Decision: card 29's pack
  outputs were committed before `ladder run` launched, which is the invocation the decision actually
  gates measurement validity on.
- **Gate failures.** `e0-names` and `e1-pack` each pass `summary_matches` on all 10 repetitions.
  `e2-files` fails on one of ten (`raw/e2-files/2/score.json`, `summary_matches: false`) — not more
  than two of ten, so `e2-files`'s cost numbers are not called suspect under the predeclared threshold,
  but the one failure is recorded here since `e2-files` is otherwise reported clean.
- **Server binary.** `server_hashes: {}` for every repetition in every cell — no cell in this matrix
  grants tools, so no server was ever built; the measurement is prompt-content-only, as designed.
- **Branch equivalence.** `git diff --stat main...33e09356a131d79c7b7f9a070761f5a1a5480b59 -- .
  ':(exclude)_mill'` differs from `main` only by this batch's own artifacts: the `07-e1-pack.md`
  sentinel-block rewrite (card 29's intended output, not drift) and the four files under this results
  root (`pack-resolve.json`, `provenance.json`, `summary.json`, `table.txt` — all outputs of this run,
  not inputs the harness or the target repository read). No engine file under
  `bench/loomyard-eval/ladder/internal/ladder/`, `bench/loomyard-eval/ladder/cmd/`, `quarry/`,
  `glyph/`, `internal/` or `cmd/` differs from `main`, and no other card, task or fasit file differs.
  "From clean `main`" holds in substance even though the recorded commit is not on `main`.

Neither cell reports `unscored_count` non-zero and neither falls short of 10/10, so nothing here is
reported as incomplete or unmeasured — all three arms are a full n=10 result.

## What this settles

The push-mode direction — glyph spans pre-resolved into the prompt before the agent starts — separates
from the control on both predeclared primary metrics at the frozen n and rule, unlike the mid-session
`toc`/`glyph` tool measured (and never separated) across ladders a through d. The parked M4 condition
is discharged with a positive result: pre-resolution pays, at least on this one exploration task, in
turns and in cost. `e2-files` (a plain file list, no spans, no signatures) also separates on turns by
the descriptive range-overlap heuristic, which narrows what is doing the work here — a named,
already-pinned file list may carry much of the effect that this task's design attributes to the
resolved spans — but `e2-files` is descriptive only under the frozen rule, so that narrowing is an
observation for future measurement design, not a finding this root tests.

Per the M4b conditional, because e1 separates, M4b (an agent revising code in a throwaway worktree,
started from a pre-resolved pack) is now genuinely ahead; card 32 adds it to `docs/roadmap.md` as a
new numbered point, pointing at this results root as its justification.
