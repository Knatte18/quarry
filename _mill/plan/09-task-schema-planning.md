# Batch: task-schema-planning

```yaml
task: Port the capability-ladder bench harness to Go
batch: task-schema-planning
number: 9
cards: 4
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [5]
```

## Batch Scope

Ports the prompt-input extraction and matrix planning halves of `run_ladder.py`: task-text extraction
with its load-bearing section boundary, output-schema extraction for both schema families, the run
enumeration functions, and scratch-directory derivation from the new `session_dir_template` field.

It depends on the run-state batch because the run pair type it enumerates against is defined there,
and reusing it beats introducing a second pair type that every caller would have to convert between.

The external interface later batches consume is `TaskTextFor`, `SchemaFor`, `PlanRuns`, `MainRuns`,
`ColdRuns`, and `SessionDir`.

Batch-local decision: `SessionDir` is the single derivation site for every scratch directory path — the
45 run sessions, the scoring session, and the two probe sessions all go through it, so no caller ever
formats the template itself.

## Cards

### Card 39: Task-text extraction and its section boundary

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/task.go`
  - `bench/loomyard-eval/ladder/internal/ladder/task_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `TASK_TEXT_HEADING` as an exported constant, `_section` as an unexported
  `section`, and `task_text_for` as `TaskTextFor(l *Ladder, repoRoot, taskKey string) (string, error)`.
  The repository root is an explicit parameter because the ladder file's task and answer-key paths are
  repo-root-relative and nothing else in this package can resolve them: the Python worked only because
  pytest happened to run from the repository root, while Go runs each test binary with its own package
  directory as the working directory, so a test reading a committed task file could not open it. Every
  call site that resolves a ladder-declared path — the task text here, the schema below, and the
  answer-key read in the scoring command — routes through this same parameter rather than deriving a
  root of its own. The
  section boundary is load-bearing rather than tidiness: extraction stops at the next `## ` heading,
  because the following section carries task 01's fasit leads and names task 04's real callers and its
  decoy outright, so an extractor that over-reads pastes the answer key into every prompt. Say that in
  the doc comment. Keep the blockquote-prefix stripping and the surrounding blank-line trimming, the
  error on a missing heading, and the error on an empty extracted body. Test both real task files,
  including the negative assertions that matter most: the extracted task 04 text must contain neither
  its decoy nor its scoring notes, and the extracted task 01 text must contain none of its fasit leads.
- **Commit:** `feat(ladder): port task-text extraction with its section boundary`

### Card 40: Output-schema extraction

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
  - `bench/loomyard-eval/README.md`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/task.go`
  - `bench/loomyard-eval/ladder/internal/ladder/task_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `_first_fenced_json_block` on top of `ExtractFencedJSON` rather than a second
  fence pattern, taking its `block` half — fences included — because the extracted schema is embedded
  verbatim into the preamble as measured stimulus and the fences are part of that text, port the impact-schema heading, the exploration-schemas heading, the exploration
  marker, and the benchmark README path as exported or unexported constants matching the Python's
  values, and port `schema_for` as `SchemaFor(l *Ladder, repoRoot, taskKey string) (string, error)`, taking
  the repository root for the same reason and resolving the benchmark README's repo-root-relative path
  against it. Selection is
  driven by the task's declared schema field, never by the task key. Test that the impact schema comes
  from the impact task's own output-schema section, that the exploration schema comes from the
  benchmark README's schemas section under its exploration marker, and that an unknown schema and a
  missing section each error.
- **Commit:** `feat(ladder): port output-schema extraction`

### Card 41: Run enumeration

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/plan.go`
  - `bench/loomyard-eval/ladder/internal/ladder/plan_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `plan_runs` as `PlanRuns(l *Ladder) []RunPair`, `main_runs` as `MainRuns`, and
  `cold_runs` as `ColdRuns`, reusing the `RunPair` type the run-state batch defines rather than
  introducing a second pair type. Test that the committed ladder plans 45 pairs, that the main set is
  the 42 non-cold pairs and the cold set the 3 cold ones, and that ordering is deterministic.
- **Commit:** `feat(ladder): port matrix run enumeration`

### Card 42: Scratch-directory derivation

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/plan.go`
  - `bench/loomyard-eval/ladder/internal/ladder/plan_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `SessionDir(l *Ladder, configID string, n int) (string, error)` substituting
  the `{config_id}` and `{n}` placeholders in `SessionDirTemplate`, erroring when the template is
  unset. Its doc comment must state that `{n}` is the repetition index uniformly, and that the scoring
  session uses a config id of `scoring` with `{n}` of 1 while the two probe sessions use
  `probe-allowlist` and `probe-denylist` with `{n}` of 1. Test that all 45 run pairs derive distinct
  directories, and that the scoring and two probe session directories are each distinct from every run
  session's and from each other.
- **Commit:** `feat(ladder): derive session scratch directories from the template`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `task_test.go` and `plan_test.go` plus every
other test file in the ladder subtree. The task-text and schema tests read the committed task files and
benchmark README directly, which is deliberate — the section boundary is a property of those files, and
a synthetic fixture would let real drift in them go undetected.
