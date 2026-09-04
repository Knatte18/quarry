# Batch: ladder-c-task-02

```yaml
task: "Ladder breadth (M1)"
batch: "ladder-c-task-02"
number: 2
cards: 2
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestLoadTaskFile|TestLoadLadder|TestRenderPrompt'
depends-on: []
```

## Batch Scope

This batch makes ladder c's task runnable. `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
is drafted against the shared pin and its shape — a three-package shed-pipeline exploration where
`shedadapters` alone is ~8.4k lines — is exactly the multi-package navigation shape this task wants.
It has two blockers and only two: it carries no `## Output schema` heading, without which
`LoadTaskFile` hard-fails, and it has no fasit, without which `ExplorationRule` has nothing to score
against. This batch removes both. It is one batch because the schema heading and the fasit are the
same task file's two missing halves and neither is useful without the other.

The external interface batch 4 consumes is the pair of paths
`bench/loomyard-eval/tasks/02-shedadapters-exploration.md` and
`bench/loomyard-eval/tasks/02-shedadapters-exploration.fasit.json`, which batch 4 names in a new
`tasks:` entry in `ladder-toc.yaml` and asserts against in the pre-matrix offline gates.

Batch-local decisions that differ from the overview's `## Shared Decisions`:

- **The task file is finished, not replaced.** The existing prompt, scope section and drafting notes
  stay byte-for-byte as they are; card 6 adds a section and changes nothing else. Rewriting a
  drafted, reviewed prompt would discard work and change the measured stimulus for no gain.
- **The schema heading goes before the answer-key notes section.** `LoadTaskFile` takes the *first*
  fenced JSON block after the *first* line starting with `## Output schema`, and its extraction is
  inclusion-based — it never looks for or excludes an answer-key heading. A schema section placed
  after the notes would still load, but placing it before matches task 01 and 04 and removes any
  chance of a future notes edit introducing a fenced block that gets picked up instead.

## Cards

### Card 6: add the output-schema section to task 02

- **Context:**
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
- **Edits:**
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Insert a `## Output schema (exploration tasks)` section into
  `02-shedadapters-exploration.md`, positioned after the `` ## `<TASK TEXT>` `` section's blockquote
  and before the existing `## Notes for whoever prepares C's fasit / scores this` section.

  Its content mirrors task 01's own section exactly: the same one-line sentence recording that the
  schema was recovered from the V1 benchmark protocol document after that document was deleted,
  followed by the same fenced JSON block carrying `relevant_files`, `key_symbols` (an array of one
  object with `name`, `file`, `role`), `summary`, `confidence` and `open_questions`. Copy that block
  from `01-reed-geometry-exploration.md` rather than retyping it, so the two files' schema blocks
  stay identical — `ExplorationRule` scores against the same three keys for both.

  Change nothing else in the file. The header, the setup section, the scope section, the
  `` `<TASK TEXT>` `` blockquote and the drafting-notes section all stay byte-for-byte as they are.
  In particular `Status: runnable now` is already correct once this section exists.

  Read `prompt.go` to confirm `outputSchemaHeadingPrefix` is matched at the start of a line and that
  `extractSchemaBlock` takes the first fenced block after it. Do not change `prompt.go`.
- **Commit:** `bench(tasks): add the output-schema section to task 02`

### Card 7: author task 02's fasit by reference-agent protocol

- **Context:**
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.fasit.json`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.fasit.json`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `.scratch/ladder.env`
- **Edits:**
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
- **Creates:**
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.fasit.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Author the fasit under the arm-C reference-agent protocol. This card's agent reads the pinned
  Loomyard checkout **exhaustively**, with no turn or token discipline applied — that unbounded read
  is the protocol, and it is exempt from this card's `Context:` allowlist, which governs files inside
  this repository only. Nothing is written into the Loomyard checkout.

  Resolve the checkout by reading `LADDER_LOOMYARD_REPO` out of `.scratch/ladder.env`, then add a
  worktree at `975578cda8d6f3a81580bd4e73725e060211b766` — the same SHA the task file's own setup
  section names — and read `internal/shedbuild`, `internal/shedadapters` and `internal/shedcheck` at
  that pin. Remove the worktree when done. Never hardcode the checkout path into any file this card
  writes.

  Answer the task file's own three numbered questions: the type(s) representing a build artifact at
  each package boundary and whether a conversion step exists, the specific function(s) at each
  handoff point (shedbuild → shedadapters, shedadapters → shedcheck), and what `shedadapters`
  contributes to the pipeline as a role rather than as an existence claim.

  Cross-check the answer by a **second independent method** before writing: a `go build` or `go vet`
  experiment against the pinned worktree, `git log` / `git show` over the relevant history, or grep
  sweeps for the identified symbols' call sites. Name the method used in the `_meta.role` line.

  Write the file in tasks 01/04's shape exactly:

  - A `_meta` object carrying `task` (`"02-shedadapters-exploration"`), `type` (`"exploration"`),
    `pinned_sha` (the shared pin above), `scope` (the three package paths as an array), `date` (the
    date this card runs, `YYYY-MM-DD`), `arm` (`"C"`), and `role` — a line naming the reference-agent
    protocol and the second method used. `StripFasitMeta` drops `_meta` before the scorer ever sees
    it, so the block is free documentation and costs no score fidelity.
  - `relevant_files`: an array of repository-relative paths, in the pinned Loomyard tree's own terms.
  - `key_symbols`: an array of objects each carrying `name`, `file` and `role`, where `role` is one
    sentence.
  - `summary`: 3–6 sentences explaining the pipeline end to end.
  - `confidence` and `open_questions`.

  `relevant_files`, `key_symbols` and `summary` must be genuinely good — `ExplorationRule` computes
  recall and precision against exactly those, so an under-specified fasit deflates recall uniformly
  across the control and the rung and hides the separation this matrix exists to find. Read
  `score.go` to confirm which keys the rule reads. Do not change `score.go`.

  If the answer turns out degenerate — the three packages barely interact, or the read cannot reach a
  confident answer at the pin — do not force a scorecard out of it. Swap ladder c's subject and
  re-author now, before the matrix, and record the swap and its reason in the task file's own notes
  section. After the matrix starts, the prompts and the fasit are measured stimulus under the
  no-mid-matrix-edit rule and that window is closed.

  A swap **keeps the identifier `02-shedadapters-exploration` everywhere it already appears**: the
  task file name, the fasit file name, the fasit's `_meta.task` value, and card 10's `tasks:` key and
  its two `configs:` entries' `task:` field. Only the prompt body, the `## Scope` section and the
  notes section change. The id is an opaque key `validate` matches `configs[].task` against — it is
  not required to describe the subject, and renaming it would ripple through the ladder file, both
  new cards in batch 4, and this plan's own `All Files Touched`. Card 12's `droppedSubstrings` for
  task 02 are then picked from the file as actually written rather than from this plan's examples,
  since a swap makes `8.4k lines` stale.
- **Commit:** `bench(tasks): author task 02's exploration fasit`

## Batch Tests

`verify` runs `TestLoadTaskFile*`, `TestLoadLadder*` and `TestRenderPrompt*` in the `ladder` package.
That scope is deliberately narrow: this batch adds no Go code, and those three test families are the
ones that read real task files and the real ladder file off disk, so they are the only existing tests
a change to `02-shedadapters-exploration.md` can break. The new assertions that cover task 02
directly — `LoadTaskFile` succeeding on it, its fasit parsing and carrying the exploration schema's
scored keys, and `CheckRenderedControlPrompt` returning nil for `c0-none`'s rendered prompt — land in
batch 4, because each of them also needs task 06 and the amended ladder file, which do not exist
until then. Splitting them across batches 2 and 3 would put two cards on the same test file in two
parallel batches.

The fasit itself has no runnable assertion in this batch. Its correctness is a matter of the
reference-agent protocol card 7 states, and its machine-checkable shape (valid JSON, the three scored
keys plus `confidence` and `open_questions`, `_meta.pinned_sha` matching the ladder entry) is
asserted by the pre-matrix gate in batch 4.
