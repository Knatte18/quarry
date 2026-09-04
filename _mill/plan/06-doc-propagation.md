# Batch: doc-propagation

```yaml
task: "Ladder breadth (M1)"
batch: "doc-propagation"
number: 6
cards: 2
verify: null
depends-on: [5]
```

## Batch Scope

This batch propagates the conclusion's own finding into the two files that carry the standing record:
`docs/roadmap.md`, which names this conclusion as T8's unpark input, and `HANDOFF.md` §3, whose table
is the measured record. It is one batch because both edits are the same act — stopping the record
from citing a superseded result — and both are written from the conclusion rather than from this
plan's expectations of it. It depends on batch 5 because neither card can be written before the
conclusion states its verdict.

T7's own finding was that the record must stop citing a superseded result. The same propagation is
owed here, which is why it is a batch of this task rather than a follow-up.

Batch-local decisions that differ from the overview's `## Shared Decisions`:

- **Neither card decides whether T8 unparks.** That is the operator's call and the roadmap says so.
  Card 17 rewrites the parked section to state which of its two unpark conditions this root's finding
  does or does not satisfy, and stops there.
- **Both cards are written from the conclusion, not from this plan.** If the conclusion's verdict is
  "no separation anywhere", these cards record that; if it separates on one shape, they record that.
  This plan does not predict which, and neither card may be drafted before the conclusion exists.

## Cards

### Card 17: update the roadmap

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md`
  - `HANDOFF.md`
- **Edits:**
  - `docs/roadmap.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Apply the disposition below to `docs/roadmap.md`. Its own header states the file "only ever says
  what is ahead" — the build record is git history and `HANDOFF.md` — so a completed row has no place
  in it.

  - Remove the entire `## Next wave: measure` section: both its rows (M1 and M2, both done once
    batch 5 lands), the table, the section heading, and the trailing sentence beneath the table
    ("M2 is small and independent; it can run as `mill-quick` or fold into M1's pre-matrix work").
    That sentence describes a scheduling choice this task has already made and would be orphaned
    prose once the table is gone.
  - Do not silently drop the **OSL-1033 host rerun** with that row. It is a live open item the M1 row
    currently carries as a clause, this task declares it Out, and it is deferred rather than
    resolved. Move it to the `## Small and independent, any time` list as its own bullet, named as
    operator-coordinated and as the one remaining way to isolate the host variable behind T7's
    task-01 discrepancy. Striking a row must not delete an open item hidden inside it.
  - Rewrite the `## Parked` section's T8 paragraph to state which of its two named unpark conditions
    this root's finding does or does not satisfy — the first condition is "a measured win from M1
    that re-establishes the surface's value to an agent" — citing
    `bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md` by path. Rewrite it rather
    than deleting it: the parked section is what makes the conclusion legible as T8's input. Do not
    make the unpark decision here and do not re-park it; that is the operator's, and this file
    already says so.
  - Update the header's standing-rule paragraph and its `Updated <date>, after T7` line so they
    reflect this root rather than T7's. The standing rule itself — nothing is built without a measured
    win behind it — is unchanged; what changes is which measurement it currently rests on.

  Take the finding's wording from the conclusion rather than restating it from memory. This file
  carries no absolute filesystem path from this machine.
- **Commit:** `docs(roadmap): propagate the breadth conclusion and retire the measure wave`

### Card 18: add the HANDOFF §3 measured-record row

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md`
- **Edits:**
  - `HANDOFF.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add one row to the `## 3. What was measured, and still holds` table, immediately after the existing
  T7 row so the table reads chronologically. Its `finding` cell states, in the conclusion's own terms,
  what this root measured across the three shapes — ladder b (task 04, the negative control), ladder c
  (task 02, multi-package navigation) and ladder d (task 06, cold-start orientation) — and whether any
  cost or correctness metric separated on any of them under the separation rule. Its `run` cell is
  `` `2026-09-04-breadth`, reps 5 ``, matching the two-part shape the existing rows use.

  Update the prose above the table so it points at this root as the current record rather than
  stopping at T7's. The existing sentence already names T7's root and says the finding did not
  reproduce cleanly; extend it to name this root and what it settles, rather than replacing the T7
  account — both are the record, and the T7 row stays exactly as it is.

  Do not touch the two harness-rule sentences below the table, and do not merge numbers across roots:
  cost numbers compare only within one results root, and the existing rows each stand on their own.

  Take the finding's wording from the conclusion rather than restating it from memory. This file
  carries no absolute filesystem path from this machine.
- **Commit:** `docs(handoff): record the breadth matrix in the measured table`

## Batch Tests

`verify: null`. This batch edits two Markdown documents and has no runnable surface: there is no test
in this repository that reads `docs/roadmap.md` or `HANDOFF.md`, and adding one to give this batch a
verify command would be inventing a gate for the sake of the field. Both files are prose read by
humans and by the next task's planner.

The repository-wide done gate — `go test ./... && golangci-lint run`, configured as
`pipeline.done_gate` — runs from the repository root before this task is marked done, which is what
confirms these two edits broke nothing elsewhere.

What actually checks this batch is review of the two diffs against the conclusion batch 5 wrote:
whether the roadmap's disposition matches `roadmap-row-disposition` (both rows and the section
heading gone, the trailing sentence gone, the OSL-1033 item moved rather than dropped, the parked T8
section rewritten rather than deleted, and no unpark decision made), and whether the HANDOFF row's
finding is the conclusion's own wording rather than a restatement of what this plan expected the
conclusion to say.
