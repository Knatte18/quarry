# Batch: goldens-history-docs

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'goldens-history-docs'
number: 9
cards: 4
verify: go test ./quarry/ -run 'TestDeltaGolden|TestDeltaRealHistory'
depends-on: [8]
```

## Batch Scope

This batch closes the task's acceptance requirements: the seven committed golden cases in both
output views, the real-history check pinned to a commit pair in this repository, and the two
documentation edits.
It depends on batch 8 only for ordering — the goldens and the history test go through the facade,
not the command line — so that the whole verb exists before its output is pinned.

The goldens live in package `quarry` for the reason the overview's Shared Decision on their location
records: both required views are rendered by that package's own renderers, the engine cannot import
them, and expressing seven in-memory batches as seven fixture repositories would put the git layer
inside a fixture whose subject is the core.

Batch-local decision: the golden files are produced under this package's own update flag and are
never hand-written.
A hand-written golden pins bytes nothing produced and passes forever, which is the one failure a
golden exists to prevent.

## Cards

### Card 48: the seven golden cases in both views

- **Context:**
  - `quarry/delta.go`
  - `quarry/render.go`
  - `quarry/text.go`
  - `quarry/quarry.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/golden_test.go`
  - `internal/cli/after_test.go`
- **Edits:** none
- **Creates:**
  - `quarry/delta_golden_test.go`
  - `quarry/testdata/delta/created.json`
  - `quarry/testdata/delta/created.txt`
  - `quarry/testdata/delta/deleted.json`
  - `quarry/testdata/delta/deleted.txt`
  - `quarry/testdata/delta/modified.json`
  - `quarry/testdata/delta/modified.txt`
  - `quarry/testdata/delta/rename-exact.json`
  - `quarry/testdata/delta/rename-exact.txt`
  - `quarry/testdata/delta/rename-evidence.json`
  - `quarry/testdata/delta/rename-evidence.txt`
  - `quarry/testdata/delta/mixed.json`
  - `quarry/testdata/delta/mixed.txt`
  - `quarry/testdata/delta/entry-error.json`
  - `quarry/testdata/delta/entry-error.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestDeltaGolden` in package `quarry`, a table over exactly seven cases:
  created, deleted, modified, an exact-tier rename, an evidence-tier rename, a mixed batch
  exercising several dispositions and both tiers at once, and a per-entry extraction failure inside
  an otherwise good batch.
  Each case builds its entries from string literals in the test file, calls the pure delta method,
  wraps the answer with two revision strings, and compares both rendered views byte for byte against
  their committed files.
  Declare this package's own update flag, following the two existing precedents in this repository —
  each package's tests build their own binary, so a flag of the same name in another package is not
  a conflict — and under it rewrite the golden from the current run instead of comparing.
  Add a comment stating that these files are produced only that way and never hand-written, and that
  the test function's name and any regeneration command are load-bearing on each other, since a
  differently named function would make a regeneration run a silent no-op producing no files at all.
  Choose the mixed case to include a symbol whose only change is its line position, asserted absent
  from the golden, and at least one file echo carrying a lossy flag, so the goldens pin those two
  behaviours as bytes rather than only as assertions.
  Every case must be deterministic across runs; if any golden differs between two consecutive
  regenerations, the ordering rule is incomplete and the fix belongs in the core rather than here.
- **Commit:** `test(quarry): pin the seven delta golden cases in both output views`

### Card 49: the real-history check

- **Context:**
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/loomyard_test.go`
  - `internal/cli/loomyard_test.go`
- **Edits:** none
- **Creates:**
  - `quarry/delta_history_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestDeltaRealHistory` in package `quarry`, running the git delta method
  against this repository itself over the commit pair the discussion's real-history pin names, scoped
  separately to the glyph package's directory and to the engine package's directory.
  Every assertion is presence-only over the in-scope subset — these symbols appear in this array —
  and never exact-set equality, because the pin's sets are explicitly partial and two of the
  commit's deletions fall outside both scopes.
  Assert the presence of each created symbol the pin lists, the presence of the one in-scope deleted
  symbol, at least one modified entry per unit, and the pin's one genuine rename present in the
  candidate array rather than in the renamed array, with its signature signal false.
  That last assertion is the load-bearing one: it is the only place in this task where the
  evidence-tier demotion is exercised against real history rather than a synthetic case, and the
  discussion records that no exact-tier rename occurs naturally in this history, which is why the
  exact tier is asserted synthetically in the core's own tests instead.
  The test must skip cleanly, never fail, when either revision is unreachable — a shallow clone is
  exactly that case — following the skip-versus-fail asymmetry the existing environment-gated
  helpers in this repository already establish: a machine that cannot see the history is a normal
  state, while a machine that can and disagrees is a real failure.
  Resolve the repository root from the test's own location rather than from any environment
  variable, since the repository under test is this one.
- **Commit:** `test(quarry): pin the delta against real history over the C1 commit pair`

### Card 50: the two documentation edits

- **Context:**
  - `quarry/delta.go`
  - `internal/cli/usage.go`
  - `docs/roadmap.md`
- **Edits:**
  - `docs/rewrite-plan.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add one paragraph to the queries section, placed after the table-of-contents
  paragraph, stating the new verb's contract in that section's own voice and at its own level of
  detail: what question it answers, its target and two revisions with the after side defaulting to
  the working tree, that it never parses a textual diff but compares two symbol tables extracted by
  the same extractor the other verbs use, the two rename tiers with one asserted and one reported,
  and that it has no threshold anywhere.
  Keep it to one paragraph — the other three verbs get one each, and a longer entry would make this
  verb read as the section's subject.
  Then update the mechanical-use list in the Loomyard section, whose item currently names this task
  by its slug as work still to come, so it names the built verb instead.
  Change no other document on this branch.
  In particular the roadmap is not edited here: it states what is ahead, and removing a completed
  item is the merge's job rather than this branch's.
- **Commit:** `docs: add the delta contract to the rewrite plan and update the mechanical-use list`

### Card 51: final acceptance sweep

- **Context:**
  - `docs/rewrite-plan.md`
  - `quarry/delta_golden_test.go`
  - `quarry/delta_history_test.go`
  - `internal/cli/usage.go`
  - `internal/cli/after_test.go`
  - `internal/mcpserver/toc_golden_test.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Verification-only card, producing no diff.
  Confirm the additive-only constraint held end to end: every committed golden that existed before
  this task is byte-identical, wherever it lives at merge time — the fifteen files the command-line
  package's own golden table pins and the set the mcp server package pins — and no existing verb,
  envelope, flag or exit code changed.
  Confirm nothing under the mcp server package or its command changed at all, since this task adds
  no tool there.
  Confirm the glyph package still compiles without tree-sitter and gained no dependency, since an
  external consumer imports it and it must stay free of cgo.
  Confirm no new third-party dependency entered the module files.
  Confirm the repository-wide gate passes: the whole test suite and the linter, which is the command
  already configured as this task's done gate.
  If any check fails, the fix belongs in the batch that broke it, never in the golden or the
  constraint.
- **Commit:** none

## Batch Tests

`verify:` runs `go test ./quarry/ -run 'TestDeltaGolden|TestDeltaRealHistory'`, selecting the two
test functions cards 48 and 49 add.
The golden table runs everywhere: it builds its batches from string literals and needs no
repository, no git binary and no external checkout.
The real-history test runs against this repository itself and skips cleanly wherever the pinned
commit pair is unreachable, so a shallow clone reports a skip rather than a failure.
Card 51's sweep rests on the repository-wide done gate rather than on this batch's own verify, since
what it checks — every pre-existing golden still byte-identical, the whole suite green, the linter
clean — is exactly what that gate runs.
