# Batch: conclusion

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "conclusion"
number: 9
cards: 1
verify: null
depends-on: [8]
```

## Batch Scope

This batch writes the tracked synthesis of the whole matrix. It is separate from batch 8 because it is judgement over finished data rather than execution, and because the reporting discipline it must obey is strict enough to be worth reviewing on its own: every claim in it has to be mechanically checkable against the tracked summary sitting beside it.

## Cards

### Card 28: write the conclusion

- **Context:**
  - `bench/loomyard-eval/ladder/README.md`
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/results/2026-08-29/summary.json`
  - `bench/loomyard-eval/ladder/results/2026-08-29/probe.json`
  - `bench/loomyard-eval/ladder/results/2026-08-29/cold_cell.json`
  - `bench/loomyard-eval/results/2026-08-28/01-reed-geometry-exploration/c.json`
  - `bench/loomyard-eval/results/2026-08-28/04-shedadapters-shuttle-impact/c.json`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-08-29/conclusion.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Synthesise the matrix into prose, sourcing every number from the tracked summary and never from a run's raw artifacts or from memory of watching the runs. Required content:
  - **What was run** — 14 configs across two ladders plus the cold cell, `n = 3`, the pinned run model, the pinned scorer, sequential dispatch, and the dates.
  - **Per-ladder findings** — for each ladder, which rungs separated from which on which metrics, phrased strictly by the disjoint-range bar. A rung is described as outperforming another only where their ranges are disjoint; everything else is reported as "not separated at n = 3", in those words. Rung-vs-rung is the primary comparison and is where the cleanest attribution claims live.
  - **The steering confound** — every rung-vs-control delta is reported as "capability + steering", never attributed to the capability alone, and the grep-count metrics are not compared between a quarry rung and a control anywhere in the document. The control is read as a floor, not as a matched pair.
  - **The decoy** — the per-config `decoy_admitted_count` reported as its own column, never folded into a precision figure. Admitting `burler.go:373` means the run would have shipped a broken edit to an unrelated interface, which task 04's own scoring notes call materially worse than missing a real caller; if a rung's runs admit it and another's do not, say so plainly even where nothing else separates them.
  - **Ladder B's correctness axis** — if all eight rungs including the control score at or near 100%, say plainly that this task cannot separate them on correctness and that every Ladder-B claim therefore rests on the efficiency ranges alone. Do not present a saturated correctness axis as evidence that each rung "works".
  - **Warm versus cold** — the `a5-bundle` against `a5-bundle-cold` contrast under the same disjoint-range bar, when the summary carries one. When it does not, read the recorded disposition and say which reason applies, offering no numbers in its place — the cold cell was partial, with fewer than three confirmed-cold runs to draw a range from; the supervised strategy was unavailable on the machine, or the cold runs invoked no daemon-backed tool at all — `toc_file` and `toc_dir` never start a daemon, so a bundle agent that answered task 01 with toc calls alone measured nothing about warmth. Either way the warm-versus-cold question that two committed scorecards delegated here remains open, and the document says so plainly rather than reading a duration difference as a warmth effect.
  - **Process observations** — the per-config dirtied-run counts read as findings about how far a rung had to go to answer, the grep-count comparisons between quarry rungs as the measure of routing-around behaviour, and any repeated schema errors on one entry-shape family as a usability finding about the two-shape design.
  - **What was not enforced** — the blinding is detection-based rather than structural, and the scorer's blinding is "does not know which config", not "can infer nothing about method". State both as limits rather than claiming them away.
  - **Statistical honesty** — `n = 3` stated explicitly, and no significance claim of any kind.
  - **Cross-suite comparison** — correctness may be set beside the sibling suite's committed results for the same tasks and fasit; duration, tokens, cost, turns, tool counts, and grep counts are not compared to it, even informally, and the document says why.
  - **Open items** — anything the matrix did not settle, plus any quarry defect the runs surfaced, written up here and filed rather than fixed, since quarry is out of this task's scope.
- **Commit:** `bench(ladder): write the per-capability benchmark conclusion`

## Batch Tests

`verify:` is null: this batch produces prose, and the discussion places the wording of the conclusion explicitly outside what is tested. Its verification is that every quantitative claim it makes is traceable to a record in the tracked summary beside it, and that no claim exceeds the disjoint-range bar — both of which are read by review rather than by a test command.
