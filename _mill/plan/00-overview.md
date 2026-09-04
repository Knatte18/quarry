# Plan: Ladder breadth (M1)

```yaml
task: "Ladder breadth (M1)"
slug: "ladder-breadth"
approved: false
started: "20260904-165748"
parent: "main"
root: ""
verify: null
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: m2-invalidation-reason-file
    file: 01-m2-invalidation-reason-file.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/
  - number: 2
    name: ladder-c-task-02
    file: 02-ladder-c-task-02.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestLoadTaskFile|TestLoadLadder|TestRenderPrompt'
  - number: 3
    name: ladder-d-task-06
    file: 03-ladder-d-task-06.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestLoadTaskFile|TestLoadLadder|TestRenderPrompt'
  - number: 4
    name: ladder-file-and-pre-matrix-gates
    file: 04-ladder-file-and-pre-matrix-gates.md
    depends-on: [1, 2, 3]
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/
  - number: 5
    name: matrix-and-conclusion
    file: 05-matrix-and-conclusion.md
    depends-on: [1, 4]
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/
  - number: 6
    name: doc-propagation
    file: 06-doc-propagation.md
    depends-on: [5]
    verify: null
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: go-only-no-python

- **Decision:** Every artefact this task produces is Go, YAML, JSON or Markdown. No Python is introduced anywhere, in test fixtures or scripts included.
- **Rationale:** `CLAUDE.md` at the repository root states the rule for this repo without exception.
- **Applies to:** all batches

### Decision: never-use-sed

- **Decision:** No shell command any card runs may invoke `sed`. Use the `Edit`/`Read`/`Write` tools, or `awk`/`grep`/plain `cat` when a shell one-liner is genuinely needed.
- **Rationale:** The global operator rule: `sed` triggers a permission prompt on every invocation, which stops autonomous work mid-run.
- **Applies to:** all batches

### Decision: no-machine-path-in-a-tracked-file

- **Decision:** No file this task commits may carry an absolute filesystem path from this machine. That covers `conclusion.md`, both new task files, both new fasit files, the amended `ladder-toc.yaml`, `docs/roadmap.md` and `HANDOFF.md`. The Loomyard checkout is always referred to through `LADDER_LOOMYARD_REPO`, never spelled out.
- **Rationale:** The repository-wide rule the harness already enforces on itself — it is why `provenance.json` stores `memory_path_hashes` rather than paths. The `detail` field of the M2 reason file is constructed rather than taken from `err.Error()` for the same reason (see `m2-one-reason-file-for-every-cause` in `_mill/discussion.md`).
- **Applies to:** all batches

### Decision: code-under-test-is-untouched

- **Decision:** No card edits `cmd/quarry-mcp/`, `quarry/`, `internal/engine/` or the CLI. The only source this task changes is the ladder harness under `bench/loomyard-eval/ladder/` and the ladder's own task, fasit and config files under `bench/loomyard-eval/tasks/`, plus the two documentation files in batch 6.
- **Rationale:** `docs/rewrite-plan.md` §2's first harness rule: the code under test is `main`'s, unmodified, and a per-repetition binary hash in `provenance.json` records it.
- **Applies to:** all batches

### Decision: m2-lands-before-the-matrix

- **Decision:** Batch 5 declares `depends-on: [1, 4]`, so the matrix invocation cannot start until batch 1's M2 change is committed and its tests are green. The matrix then runs against the post-M2 harness, and the breadth matrix's own non-control cells (`b8-toc-dir`, `c1-toc-dir`, `d1-toc-dir`) completing end to end are what satisfies the third harness rule for this harness change.
- **Rationale:** A harness change cannot land mid-matrix, and sequencing it first satisfies both the no-mid-matrix-edit rule and the harness-change-proof rule at no extra run cost. A clean tree at invocation time also keeps `provenance.json`'s `quarry_dirty` honest.
- **Applies to:** all batches

### Decision: shared-pin-for-every-task

- **Decision:** Both new tasks pin `975578cda8d6f3a81580bd4e73725e060211b766`, the same SHA tasks 01–05 already use. It appears in each new task file's setup section, in each new fasit's `_meta.pinned_sha`, and in each new `tasks:` entry in `ladder-toc.yaml`.
- **Rationale:** One tree state under all three shapes means a difference between shapes is the shape, not the tree.
- **Applies to:** ladder-c-task-02, ladder-d-task-06, ladder-file-and-pre-matrix-gates

### Decision: cards-that-read-the-pinned-checkout-freely

- **Decision:** A card's `Context:` allowlist governs files **inside this repository**. Three cards additionally read the pinned Loomyard worktree with no turn or token discipline, and that reading is deliberately unbounded rather than a plan defect: the two fasit-authoring cards (7 and 9), whose exhaustive read is the arm-C protocol itself, and card 8, whose subject survey must range over the whole tree precisely because the shape it is picking for is one where no package is named in advance. None of the three writes anything into that checkout, and each removes the worktree it added when done.
- **Rationale:** `fasit-authored-by-a-reference-agent-card` in `_mill/discussion.md` requires an exhaustive read cross-checked by a second independent method; an allowlist over a foreign repository cannot express that and would defeat the protocol. Card 8 is on the same footing for the same reason — `ladder-d-cold-start-orientation` requires a subject whose real answer spans at least two packages and whose package names appear nowhere in the prompt, and no bounded read can establish either.
- **Applies to:** ladder-c-task-02, ladder-d-task-06

### Decision: results-raw-tree-stays-untracked

- **Decision:** `bench/loomyard-eval/ladder/.gitignore`'s `results/*/raw/` entry is not modified and no card commits anything under a results root's `raw/` subtree. The results root's own `provenance.json`, `summary.json`, `table.txt` and `conclusion.md` are committed.
- **Rationale:** Settled by T7; re-litigating it here would change what the record contains for reasons unrelated to what this task measures.
- **Applies to:** matrix-and-conclusion

### Decision: separation-decision-rule

- **Decision:** A shape is reported as **separating** only when all three hold within this results root: (1) at least one cost metric's comparison entry is `separated: true`, (2) that metric's median moves in the direction the toc hypothesis predicts (rung cheaper than control), and (3) neither recall nor precision degrades — the correctness comparison is not `separated: true` in the control's favour. Anything short of all three is **no separation**, with medians and ranges quoted so a reader can see how close it came.
- **Rationale:** `separated` is a strict no-overlap test on min–max ranges that can miss a real effect at n=5, and a cost win bought with degraded correctness is not a win for a tool whose stated purpose is complete extraction. Naming the rule before the numbers exist is what keeps it from being fitted to them.
- **Applies to:** matrix-and-conclusion, doc-propagation

### Decision: neutral-schema-example-values-in-the-new-task-files

- **Decision:** The exploration schema block tasks 02 and 06 carry keeps task 01's keys, structure and prose byte-for-byte, with exactly one change: the `relevant_files` example value `"internal/reedengine/geometry.go"` becomes the placeholder `"path/to/file.go"`, matching the placeholder `key_symbols` entry already in that block. The two new files' blocks are byte-identical to each other and are deliberately **not** byte-identical to task 01's.
- **Rationale:** `RenderPrompt` puts `SchemaBlock` into every rendered prompt, so task 01's example path would appear verbatim in task 06's prompt — a prompt whose entire shape is "names no package and no file". That silently breaks card 8's constraint (b) outright if the chosen subject lands in `internal/reedengine` or `internal/reedcli`, and even when it does not, a real package path shown as an example is an anchor for an agent whose whole task is deciding where to look. Task 02's prompt names its own three packages, so the stray path is only noise there, but both new files use the same block so the two shapes' schema is not a variable between them. Only the example *values* change; `ExplorationRule` scores `relevant_files`, `key_symbols` and `summary`, and those keys are untouched, so scoring is unaffected. Per-task schema variation is already the norm — task 04 carries the impact schema — so this is not a new kind of difference.
- **Applies to:** ladder-c-task-02, ladder-d-task-06, ladder-file-and-pre-matrix-gates

### Decision: results-root-date-substitution

- **Decision:** This plan pins the results root as `bench/loomyard-eval/ladder/results/2026-09-04-breadth/`. If the matrix invocation begins on a different calendar date, that date replaces `2026-09-04` **everywhere the path appears in this plan and in every file the plan produces** — batch 5's cards 15 and 16, batch 6's cards 17 and 18, and the overview's `## All Files Touched`.
- **Rationale:** The root name is a label; `provenance.json` carries the real timestamps. But cards 17 and 18 write the path and the root name into two tracked documents, so a substitution scoped to batch 5 alone would leave `docs/roadmap.md` and `HANDOFF.md` citing a root that does not exist. A name contradicting the day the run started is worse than the drift, and a tracked doc citing a nonexistent path is worse than both.
- **Applies to:** matrix-and-conclusion, doc-propagation

### Decision: conventional-commit-prefixes

- **Decision:** Every card that produces a diff uses a conventional-commit prefix with a scope drawn from the area it touches: `feat(ladder)`, `test(ladder)`, `bench(tasks)`, `bench(ladder)`, `docs(roadmap)`, `docs(handoff)`. A verification-only card whose `Edits:`/`Creates:`/`Deletes:`/`Moves:` are all `none` carries `Commit: none` instead, per the plan-card convention; batch 5's card 14 is the one such card in this plan.
- **Rationale:** Matches the prefixes already in this repository's history and keeps one commit per card legible in `git log --oneline`.
- **Applies to:** all batches

## All Files Touched

- `HANDOFF.md`
- `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/prompt_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/run.go`
- `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- `bench/loomyard-eval/ladder/internal/ladder/runstate_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/testdata/fakeclaude/main.go`
- `bench/loomyard-eval/ladder/ladder-toc.yaml`
- `bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md`
- `bench/loomyard-eval/ladder/results/2026-09-04-breadth/provenance.json`
- `bench/loomyard-eval/ladder/results/2026-09-04-breadth/summary.json`
- `bench/loomyard-eval/ladder/results/2026-09-04-breadth/table.txt`
- `bench/loomyard-eval/tasks/02-shedadapters-exploration.fasit.json`
- `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
- `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.fasit.json`
- `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`
- `docs/roadmap.md`
