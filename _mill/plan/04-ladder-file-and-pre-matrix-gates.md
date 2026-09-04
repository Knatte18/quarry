# Batch: ladder-file-and-pre-matrix-gates

```yaml
task: "Ladder breadth (M1)"
batch: "ladder-file-and-pre-matrix-gates"
number: 4
cards: 4
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/
depends-on: [2, 3]
```

## Batch Scope

This batch declares the six cells the matrix runs and gates every one of them offline before a single
real `claude -p` call is spent. It amends `ladder-toc.yaml` with ladders c and d — two new `tasks:`
entries and four new configs — and does a whole-header pass over that file's comment block, then adds
the four offline assertions that stand between an authoring typo and thirty wasted real runs:
`LoadLadder` accepting the amended file with one control per ladder letter, `LoadTaskFile` succeeding
on both new task files, both new fasits parsing with the exploration schema's scored keys and the
right pin, and `CheckRenderedControlPrompt` returning nil for every control cell's fully rendered
prompt.

It is one batch because all four assertions are the same gate — cheap, pure, offline, and worthless
individually. It depends on batches 2 and 3 because every one of them reads a file those batches
write. It is the last batch before real budget is spent.

The external interface batch 5 consumes is the amended `ladder-toc.yaml` itself, specifically the six
cell ids `b0-none`, `b8-toc-dir`, `c0-none`, `c1-toc-dir`, `d0-none`, `d1-toc-dir`, which batch 5
passes to `--cells` as one list.

Batch-local decisions that differ from the overview's `## Shared Decisions`:

- **One ladder file, not a second one.** Ladders c and d go into the existing `ladder-toc.yaml`.
  `config.go`'s `validate` requires exactly one control per ladder letter that appears and imposes no
  cap on the number of letters, so the header's "two task groups per file" line is a stale comment
  rather than a format rule. A second file would mean duplicate ids across files, two invocations,
  two provenance records, and a reader who has to know which file a `b8-toc-dir` number came from.
- **The header pass is a whole-header re-read, not a fix to named lines.** Every statement in that
  comment block is re-read against the file as this batch leaves it and rewritten where it has become
  false. Card 10 names four statements that are known stale; finding only those four is not the pass
  being done.
- **The pre-matrix assertions get their own file.** `prematrix_test.go` rather than an addition to
  `config_test.go` or `gates_test.go`: they are one coherent gate keyed on real committed stimulus,
  and a reader looking for "what runs before we spend money" should find it in one place.

## Cards

### Card 10: amend ladder-toc.yaml with ladders c and d

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.fasit.json`
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.fasit.json`
- **Edits:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add two entries to the `tasks:` map, each in the shape the two existing entries use — `task_file`,
  `pinned_sha`, `schema`, `fasit`, in that order:

  - `02-shedadapters-exploration`: task file
    `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`, pin
    `975578cda8d6f3a81580bd4e73725e060211b766`, schema `exploration`, fasit
    `bench/loomyard-eval/tasks/02-shedadapters-exploration.fasit.json`.
  - `06-loomyard-cold-start-orientation`: task file
    `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`, the same pin, schema
    `exploration`, fasit `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.fasit.json`.

  Add four entries to `configs:`, keeping the existing four unchanged and following the file's own
  one-comment-per-ladder-group layout:

  - `c0-none` — ladder `c`, task `02-shedadapters-exploration`, `allowed: []`.
  - `c1-toc-dir` — ladder `c`, task `02-shedadapters-exploration`, `allowed: [toc]`.
  - `d0-none` — ladder `d`, task `06-loomyard-cold-start-orientation`, `allowed: []`.
  - `d1-toc-dir` — ladder `d`, task `06-loomyard-cold-start-orientation`, `allowed: [toc]`.

  Ladder b's two ids stay exactly as they are. `b8-toc-dir` is not renamed to `b1-toc-dir` for
  tidiness: the id is deliberately chosen to stay unique against the main matrix's `b1`..`b7`, and
  cross-root comparison by id depends on that.

  Change nothing else in the machine-read body: `run_model`, `reps`, `run_effort`, `max_turns`, the
  `scorer` block, `quarry_tools`, the `server` block and `source_repo` all stay at their current
  values. `LoadLadder` rejects unrecognised keys, so no new key is introduced.

  Then do a **whole-header pass** over the file's leading comment block: re-read every statement in
  it against the file as this card leaves it and rewrite each one that has become false. Four are
  known stale and fixing only these four is not the pass being done:

  - `# Design: four surviving cells, run by the T2 headless harness ...` — it is eight cells now.
  - The line stating the ladder letters allow only two task groups per file, in the
    "Deliberately NOT here" section's task-02 paragraph — `validate` imposes no cap on letters, so
    this was a stale comment rather than a rule.
  - The "Deliberately NOT here" entry naming a whole-repo, no-scope-hint task as absent — ladder d is
    exactly that task.
  - `Task 02 (three packages) has no fasit either.` — this task authors it.

  Add ladder c and ladder d paragraphs to the per-ladder design list, in the shape ladders a and b
  already use: the task, the scope it exercises, and what a result there would mean. Keep the
  reading-the-result guidance, the harness-rule reminders and the run command; update the run
  command's `--results` example to a `-breadth` root and its `--cells` list to the six cells this
  matrix runs. The T2/T7 sentence naming which cells those tasks ran is history and stays accurate
  as it is — extend it rather than replacing it.

  Read `config.go` to confirm `validate`'s rules before writing: `source_repo` exactly
  `env:LADDER_LOOMYARD_REPO`; non-zero `run_model`, `run_effort`, `max_turns`, `reps`,
  `scorer.model`, `scorer.effort`; per task a non-empty `task_file`, `pinned_sha`, `fasit` and a
  `schema` of exactly `"exploration"` or `"impact"`; unique config ids; every `configs[].task` a key
  in `tasks`; every entry of `configs[].allowed` present in `quarry_tools`; and exactly one control
  per ladder letter that appears. Do not change `config.go`.
- **Commit:** `bench(ladder): add ladders c and d to the toc ladder file`

### Card 11: update the real-ladder-file config test to eight cells

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Update `TestLoadLadder_RealTocFile` in `config_test.go`, which today asserts the four surviving
  cell ids and two controls:

  - Extend its `wantIDs` set to all eight ids: `a0-none`, `a2-toc-dir`, `b0-none`, `b8-toc-dir`,
    `c0-none`, `c1-toc-dir`, `d0-none`, `d1-toc-dir`. Its existing length check against `len(wantIDs)`
    then covers the count without a second literal.
  - Extend its `ControlFor` assertions from letters `a` and `b` to all four: `ControlFor("c")` returns
    `c0-none` and `ControlFor("d")` returns `d0-none`, in the same two-line shape the existing
    assertions use.
  - Add an assertion that both new task ids, `02-shedadapters-exploration` and
    `06-loomyard-cold-start-orientation`, are keys in the loaded `Tasks` map and that each carries
    `Schema` `"exploration"` and the shared pin `975578cda8d6f3a81580bd4e73725e060211b766`.

  Its `QuarryTools` assertion (`[toc]`, one entry) and both `MCPPrefix` assertions are unchanged —
  this batch adds no tool and no server change. Update the function's doc comment, which today says
  "the four surviving cell ids ... the two controls", to match what it now asserts.

  This test is the cheap guard against a typo in card 10 costing thirty real runs, which is why it is
  updated in the same batch that writes the file.
- **Commit:** `test(ladder): assert the eight-cell toc ladder file loads`

### Card 12: assert both new task files load

- **Context:**
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/prompt_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Extend `TestLoadTaskFile_RealTaskFiles`'s table in `prompt_test.go` with two new cases, one per new
  task file, in the same struct shape the two existing cases use (`name`, `path`, `wantTaskText`,
  `wantSchemaBlock`, `droppedSubstrings`), with paths relative to the test's own directory exactly as
  the existing entries are.

  For each new case, add package-level string constants alongside the existing `task01TaskText` /
  `task01SchemaBlock` / `task04TaskText` / `task04SchemaBlock` constants, holding the expected task
  text and schema block verbatim. Both new files carry the identical exploration schema block, so
  their two `wantSchemaBlock` constants must be byte-identical to each other and to `task01SchemaBlock`
  — if they are not, card 6 or card 8 did not copy the block as required and that is the finding this
  assertion exists to surface.

  Each case's `droppedSubstrings` names at least one substring drawn from the setup section, one from
  the scope section, and one from the notes section of that file, so the test proves the
  inclusion-based extractor did not leak any of the three. For task 02, the pinned SHA and the
  `worktree add` line are setup-section candidates and `8.4k lines` is a scope-section candidate. For
  task 06, pick equivalents from the file card 8 actually wrote.

  This is the direct regression against task 02's missing `## Output schema` heading — before card 6
  the file could not be loaded at all — and the equivalent first load of task 06. Do not change
  `prompt.go`.
- **Commit:** `test(ladder): assert task 02 and task 06 load and leak nothing`

### Card 13: the pre-matrix offline gate

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/tasks/02-shedadapters-exploration.fasit.json`
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.fasit.json`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `prematrix_test.go` in `package ladder`, carrying a file header comment stating what it is:
  the offline gate that runs before any real `claude -p` call, keyed on the real committed
  `ladder-toc.yaml`, the real committed task files and the real committed fasits, so an authoring
  mistake is caught for free rather than after thirty real runs.

  Two test functions:

  - `TestPreMatrix_ControlPromptsAreBlind`. For each control cell id in `b0-none`, `c0-none`,
    `d0-none`: load `../../ladder-toc.yaml` with `LoadLadder`, find that config, load its task's
    `task_file` with `LoadTaskFile`, render the prompt the same way `run.go`'s `runCellRepetition`
    does — `RenderPrompt(content, dest, grantedToolNames(l, cfg))` with an arbitrary non-empty `dest`
    string standing in for the pinned worktree path — and assert `CheckRenderedControlPrompt(prompt,
    BlindingInput{MCPPrefix: l.MCPPrefix(), ServerName: l.ServerName(), QuarryRepoRoot: <the test's
    own placeholder root>}, l.QuarryTools)` returns nil. On a non-nil finding, fail with the cell id
    and the finding's `Message`, so the failure names which prompt tripped which token.

    This is the only pre-matrix check that catches a new prompt carrying the bare token `toc` or
    `quarry` before it voids a control cell for the whole matrix. It matters because a void control
    repetition is written by `writeVoidRepetition`, is flagged `blinding_failed`, produces no
    `invalid_reason.txt` at all — that path never reaches the attempt loop — and does not abort the
    run, so every invocation re-attempts and re-fails it deterministically while the paired rung cell
    spends five real calls against a control that can never complete. Read `run.go` for the exact
    render call and `gates.go` for the check's own rules; change neither.

    `dest` must be a fixed placeholder string, not a real path on this machine — this is a test file
    and it is tracked.

  - `TestPreMatrix_NewFasitsAreWellFormed`. For each of the two new fasit paths, relative to the
    test's own directory: read the file, assert it decodes as JSON into a `map[string]any`, assert it
    carries the exploration schema's three scored keys `relevant_files`, `key_symbols` and `summary`
    plus `confidence` and `open_questions`, assert `relevant_files` and `key_symbols` are both
    non-empty arrays, assert each `key_symbols` entry is an object carrying non-empty `name`, `file`
    and `role`, and assert `_meta.pinned_sha` equals the `PinnedSHA` of the task entry that names
    that fasit in the loaded `ladder-toc.yaml` — read the pin from the ladder file rather than
    re-spelling it, so the two can never drift apart silently.

    A non-empty `relevant_files` and `key_symbols` is the machine-checkable half of the
    degenerate-fasit guard: `ExplorationRule` computes recall and precision against exactly those, so
    an empty one deflates recall uniformly across the control and the rung and hides the separation
    this matrix exists to find. The judgement half — whether the answer is substantive rather than
    merely present — is cards 7 and 9's own, and this test does not attempt it. Read `score.go` to
    confirm which keys the rule reads; do not change it.

  Both functions use only the standard library plus this package's own exported entry points.
- **Commit:** `test(ladder): add the pre-matrix offline gate for control prompts and fasits`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/internal/ladder/` runs the whole `ladder` package
rather than a `-run` filter. That is the right scope here: card 10 changes the one ladder file that
`TestLoadLadder_RealTocFile` and the new `prematrix_test.go` both read off disk, and cards 6 and 8 in
the upstream batches changed two task files that `TestLoadTaskFile_RealTaskFiles` reads, so a
regression can surface in more than one test family. The package's tests are offline and cheap —
`testdata/fakeclaude` stands in for the claude binary, and every assertion this batch adds is a pure
function of committed files with no process invocation at all.

The files this batch's tests cover:

- `config_test.go` — `TestLoadLadder_RealTocFile`, extended to eight cells, four controls, and the
  two new task entries' schema and pin.
- `prompt_test.go` — `TestLoadTaskFile_RealTaskFiles`, extended with task 02 and task 06 including
  the leak assertions that prove the setup, scope and notes sections stay out of the extracted
  content.
- `prematrix_test.go` — `TestPreMatrix_ControlPromptsAreBlind` and
  `TestPreMatrix_NewFasitsAreWellFormed`, the two gates that stand directly between an authoring
  mistake and real budget.

Together these four assertions are the complete offline layer named in `_mill/discussion.md`'s
Testing section. One residual risk they cannot catch is a subtly bad new prompt — one that is blind
and loads cleanly but asks the wrong thing. Two real runs (`c1-toc-dir` and `d1-toc-dir` at
`--reps 1`) would surface it, and batch 5's preflight card states the terms under which that smoke
run may be taken; it is not taken on this task's own authority, because it costs real budget and the
cost discipline reserves any widening for the operator.
