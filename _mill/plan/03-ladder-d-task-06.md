# Batch: ladder-d-task-06

```yaml
task: "Ladder breadth (M1)"
batch: "ladder-d-task-06"
number: 3
cards: 2
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestLoadTaskFile|TestLoadLadder|TestRenderPrompt'
depends-on: []
```

## Batch Scope

This batch authors ladder d's task from nothing: a cold-start orientation shape whose prompt names no
package and no file, asking the agent to locate, in an unfamiliar repository, which package(s) own a
named behaviour and what the entry points into that behaviour are. It is the condition under which a
directory-level toc is most plausibly worth its prompt cost — the agent does not know where to look,
so the first cheap survey is the whole value — and it is the far end of a scope axis whose near end
is ladder b, where the task text names the file outright. It is one batch because the subject pick,
the prompt and the fasit are one authoring act: the fasit's exhaustive read is what confirms the
subject was a good pick, and neither half is usable alone.

The external interface batch 4 consumes is the pair of paths
`bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md` and
`bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.fasit.json`.

Batch-local decisions that differ from the overview's `## Shared Decisions`:

- **The subject is picked by the implementer, not by this plan.** The plan fixes the three
  constraints the pick must satisfy and the record it must leave behind; naming a subject here would
  fix it from a reading of the pinned tree this planning session never made.
- **The pick is provisional until the fasit read confirms it.** Constraints (a) and (c) below rest at
  pick time on a reading good enough to be plausible; only card 9's exhaustive read establishes
  either. If that read disconfirms one, the remedy is a pre-matrix subject swap and nothing else.

## Cards

### Card 8: author the cold-start orientation task file

- **Context:**
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
  - `.scratch/ladder.env`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Pick the subject first. Resolve the pinned Loomyard checkout by reading `LADDER_LOOMYARD_REPO` out
  of `.scratch/ladder.env`, add a worktree at
  `975578cda8d6f3a81580bd4e73725e060211b766`, and survey it far enough to choose a behaviour
  satisfying all three constraints:

  (a) the real answer spans at least two packages;
  (b) none of those package names appears anywhere in the **rendered** prompt — the task-text
      blockquote and the schema block alike, since `RenderPrompt` assembles both;
  (c) it is answerable entirely from the pinned SHA.

  Constraints (a) and (c) are provisional at this point — the pick rests on a reading good enough to
  be plausible, and card 9's exhaustive read is what establishes either for real.

  Then write the task file in tasks 01/02's shape, with these sections in this order:

  - An `# Task 06 — <subject> cold-start orientation` heading, then the `Type: exploration`,
    `Verb under test: `toc`` and `Status: runnable now` lines tasks 01 and 02 both carry. Naming the
    verb here is safe and matches the two existing files: `LoadTaskFile` extracts only the
    `` `<TASK TEXT>` `` blockquote and the schema block, so nothing in this header reaches a rendered
    prompt.
  - A `## Setup` section pinning `975578cda8d6f3a81580bd4e73725e060211b766` and using
    `$LADDER_LOOMYARD_REPO` in its `git worktree add` line, exactly as tasks 01 and 02 do. Never
    write an absolute path from this machine.
  - A `## Scope` section stating plainly that this task's scope is the whole repository and that the
    prompt deliberately names no package — this section is prose for a human reader and is never
    extracted into the prompt.
  - A `` ## `<TASK TEXT>` `` section whose blockquote is the prompt itself: a whole-repository
    question naming no package and no file, asking which package(s) own the named behaviour and what
    the entry points into that behaviour are. `extractTaskText` collects lines beginning with `>`
    until the first line that is neither a blockquote line nor blank, stripping a leading `>` and at
    most one following space, so an indented sub-list inside the quote keeps its relative
    indentation.
  - A `## Output schema (exploration tasks)` section carrying the fenced JSON block copied from
    `01-reed-geometry-exploration.md`, which is already in this card's `Context:`. Copy it from task
    01 specifically, not from task 02: this batch declares `depends-on: []` and can run before batch
    2's card 6 adds that section to `02-shedadapters-exploration.md`, where it does not exist today.

    Then make **exactly one** change to the copy, per the overview's
    `neutral-schema-example-values-in-the-new-task-files` decision: replace the `relevant_files`
    example value `"internal/reedengine/geometry.go"` with the placeholder `"path/to/file.go"`,
    matching the placeholder already used in that block's `key_symbols` entry. This is load-bearing
    for this task specifically, not cosmetic. `RenderPrompt` puts `SchemaBlock` into every rendered
    prompt, so copying the block unchanged would put a real Loomyard package path into a prompt whose
    whole shape is "names no package and no file" — breaking constraint (b) outright if the subject
    picked above lands in `internal/reedengine` or `internal/reedcli`, and anchoring the agent on a
    real package even when it does not. Card 6 makes the identical change to task 02, so the two new
    files' blocks stay byte-identical to each other.

    The section must come **before** the notes section below: `extractSchemaBlock` takes the
    first fenced JSON block after the first `## Output schema` line, and a notes section that
    happened to carry a fenced block ahead of the schema would be extracted instead.
  - A `## Notes for whoever prepares C's fasit / scores this` section recording the chosen subject
    and, one constraint per sentence, why it satisfies (a), (b) and (c). This record is what makes a
    later reader able to judge the pick rather than take it on trust. It is also where a subject swap
    and its reason are recorded, should card 9 force one.

  The prompt text must contain none of the bare tokens `quarry`, `toc`, or the server name `quarry`
  — `CheckRenderedControlPrompt` is fatal on any of them for a control cell, and `d0-none` is a
  control. Read `gates.go` and `match.go` to see exactly what the bare-token matcher treats as a
  match before writing the prompt; the offline assertion in batch 4 is what catches a miss, but
  writing the prompt with the rule in view is cheaper than discovering it there. Do not change
  `gates.go`, `match.go` or `prompt.go`.

  Remove the pinned worktree when done.
- **Commit:** `bench(tasks): author task 06, the cold-start orientation shape`

### Card 9: author task 06's fasit by reference-agent protocol

- **Context:**
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.fasit.json`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.fasit.json`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `.scratch/ladder.env`
- **Edits:**
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`
- **Creates:**
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.fasit.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Author the fasit under the same arm-C reference-agent protocol card 7 states: an exhaustive read of
  the pinned Loomyard checkout with no turn or token discipline applied, exempt from this card's
  `Context:` allowlist, which governs files inside this repository only. Nothing is written into the
  Loomyard checkout. Resolve the checkout through `LADDER_LOOMYARD_REPO` in `.scratch/ladder.env`,
  work at `975578cda8d6f3a81580bd4e73725e060211b766`, and remove the worktree when done.

  Answer the prompt card 8 wrote: which package(s) own the named behaviour, and what the entry points
  into that behaviour are. Cross-check by a second independent method — a `go build` or `go vet`
  experiment, `git log` / `git show` over the relevant history, or grep sweeps for the identified
  symbols' call sites — and name the method used in `_meta.role`.

  Write the file in tasks 01/04's shape: a `_meta` object carrying `task`
  (`"06-loomyard-cold-start-orientation"`), `type` (`"exploration"`), `pinned_sha` (the shared pin),
  `scope`, `date`, `arm` (`"C"`) and `role`; then `relevant_files`, `key_symbols` (objects each with
  `name`, `file` and one-sentence `role`), `summary` (3–6 sentences), `confidence` and
  `open_questions`. Read `score.go` to confirm which keys `ExplorationRule` scores against; do not
  change it.

  This exhaustive read is what establishes constraints (a) and (c) from card 8 for real. If it
  disconfirms either — the true answer turns out to sit in one package, or it cannot be reached
  confidently at the pin — swap ladder d's subject, re-author both the task file and this fasit, and
  record the swap and its reason in the task file's notes section. Do that now, before the matrix;
  after the invocation starts, the prompt and the fasit are measured stimulus under the
  no-mid-matrix-edit rule and the swap window is closed.

  A subject swap made here also re-opens card 8's blinding constraint: the replacement prompt must
  again contain none of the bare tokens `quarry` or `toc`, and batch 4's offline assertion is what
  confirms it.
- **Commit:** `bench(tasks): author task 06's exploration fasit`

## Batch Tests

`verify` runs `TestLoadTaskFile*`, `TestLoadLadder*` and `TestRenderPrompt*` in the `ladder` package
— the same narrow scope batch 2 uses and for the same reason: this batch adds no Go code, and those
are the only existing tests that read real task files and the real ladder file off disk.

Neither new file has an assertion against it inside this batch. Both are covered by the pre-matrix
offline gates in batch 4, which is where they can be: `LoadTaskFile` succeeding on task 06 and
returning a non-empty task text and schema block, the fasit parsing with the exploration schema's
three scored keys plus `confidence` and `open_questions`, `_meta.pinned_sha` matching the ladder
entry's own pin, and — the one that matters most here —
`CheckRenderedControlPrompt` returning nil for `d0-none`'s fully rendered prompt. That last check is
a pure function of the task file and the ladder file, costs nothing, and is the only pre-matrix gate
that catches a new prompt carrying the bare token `toc` or `quarry` before it voids a control cell
for the whole matrix. Those assertions cannot live here because each of them also needs the amended
`ladder-toc.yaml`, which batch 4 writes.
