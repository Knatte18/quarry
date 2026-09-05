# Plan: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)

```yaml
task: 'P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)'
slug: glyphs-verb
approved: false
started: '20260905-165109'
parent: 'main'
root: ""
verify: null
discussion_sha: '83172a2e4df0fa30842a8735a4f0abc9dc7253e7'
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: facade-glyphs-view
    file: 01-facade-glyphs-view.md
    depends-on: []
    verify: go test ./quarry/
  - number: 2
    name: cli-view-flag-and-glyphs-verb
    file: 02-cli-view-flag-and-glyphs-verb.md
    depends-on: [1]
    verify: LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ ./quarry/
  - number: 3
    name: goldens-and-docs
    file: 03-goldens-and-docs.md
    depends-on: [2]
    verify: LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ ./internal/engine/
```

## Shared Decisions

### Decision: the-discussion-decisions-are-binding

- **Decision:** every `### <name>` subsection under `_mill/discussion.md`'s `## Decisions` heading is a
  fixed constraint on this plan, not a suggestion. Where a card's `Requirements:` restates one, the
  restatement is the implementer's contract; where this plan is silent, the discussion decision
  still governs. Three decisions are load-bearing enough to repeat here in full, because a card that
  gets them wrong produces code that compiles and passes its own tests: `glyphs-json-shadow-struct`
  (the JSON goes through an unexported envelope, never through the answer type's own tags),
  `viewless-output-unchanged` (no committed golden under `internal/cli/testdata/` **or**
  `internal/engine/testdata/` is regenerated — if one changes, that is this task's defect, and both
  halves are checked, in batch 3 card 15), and `projection-is-pure-and-late` (nothing under
  `internal/engine/` is modified by any card in any batch).
- **Rationale:** the discussion is 720 lines and no card carries all of it; naming the three that
  cannot be re-derived from a card's own text is what keeps a cold Sonnet session from
  "simplifying" the shadow struct away or reaching for `-update` on a red golden.
- **Applies to:** all batches

### Decision: glyphs-options-is-exported

- **Decision:** the facade's frozen option value is exported as
  `func GlyphsOptions() TOCOptions` in `quarry/repo.go`, not unexported as the discussion's
  `facade-shape` decision spells it (`glyphsOptions()`).
- **Rationale:** the discussion's `preset-single-source` decision requires a test that parses the
  CLI preset tokens through `parseArgs` and compares the result against "the facade's options".
  That test must live in `internal/cli` (it is the only package that can name both sides —
  `internal/cli` imports `quarry`, and the reverse import would be a cycle), and an unexported
  `glyphsOptions` is unreachable from there. The only alternatives are to trust the two to be
  edited together — the exact convention `preset-single-source` exists to replace — or to duplicate
  the option values in the test, which asserts nothing. Exporting a two-field option value is a
  smaller public surface than either failure.
- **Applies to:** batch 1, batch 2

### Decision: the-facade-drift-test-uses-a-scratch-tree

- **Decision:** the facade-side equivalence test — `Glyphs(target)` equals
  `GlyphView(target, TOC(target, GlyphsOptions()))` — runs against a `writeScratchTree` fixture in
  package `quarry`, not against the pinned Loomyard checkout the discussion's `## Testing` section
  names.
- **Rationale:** the property asserted is repository-independent — it is that one method is the
  composition of two others — so a checkout adds nothing to it. Package `quarry` has no
  `loomyardRepo` gate today, and adding one would be a third hand-maintained copy of that
  skip-versus-fail asymmetry (`internal/engine` and `internal/cli` each already carry one) whose
  only effect would be to make this assertion skip on every machine without a Loomyard. The
  pinned checkout is still load-bearing for the two places that genuinely need real bytes: the
  golden files and the byte-identity pairs, both in `internal/cli`.
- **Applies to:** batch 1

### Decision: the-preset-is-a-var-not-a-const

- **Decision:** the frozen preset in `internal/cli/flags.go` is a package-level `var
  glyphsPreset = []string{...}`, with a doc comment stating it is a frozen constant that Go cannot
  express as a `const` because a slice is not a constant expression, and that no code may append
  into it.
- **Rationale:** the discussion's `preset-expansion` decision says "a package-level constant"; Go
  has no constant slice. The `var` plus the stated no-append rule is the closest expressible form,
  and the append rule is not decorative — the rewrite in `parseArgs` builds a new slice from the
  preset and the caller's own tokens, and appending into `glyphsPreset`'s backing array would make
  the second invocation in one process see the first's target.
- **Applies to:** batch 2

### Decision: the-pinned-checkout-lives-in-scratch

- **Decision:** the Loomyard checkout pinned at `72c23d9` that the golden and byte-identity tests
  need is a local clone at `.scratch/loomyard-pin` inside this worktree, created by batch 2's first
  card, and every `verify:` command that needs it spells
  `LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin"` — `$PWD` because `verify:` runs from the
  repository root, so the expansion is absolute (which `loomyardRepo`'s own `os.Stat` requires)
  without any machine-specific path appearing in a committed file.
- **Rationale:** this machine has no `LADDER_LOOMYARD_REPO` set, so without this step every golden
  case and every byte-identity pair skips, and the batch would report green having asserted
  nothing — which is precisely the wrong-negative failure mode this whole task exists to prevent
  in its own subject matter. A clone reads the source repository and mutates nothing in it;
  `.scratch/` is gitignored at the repository root, so nothing is committed.
- **Applies to:** batch 2, batch 3

### Decision: no-file-under-internal-engine-is-modified

- **Decision:** no card in any batch edits, creates or deletes a file under `internal/engine/`. The
  engine's `Symbol`, `FileEntry`, `DirAnswer` and `TOCOptions` are read for their tag spellings and
  their documented semantics and are otherwise untouched, including their JSON tags.
- **Rationale:** it is the discussion's `projection-is-pure-and-late` decision, its
  `glyphs-json-shadow-struct` decision's whole reason for existing, and the task's own additivity
  constraint. `internal/engine/answer.go` appears in several cards' `Context:` for exactly this
  reason — it must be read before the renderer is written, and never edited.
- **Applies to:** all batches

### Decision: doc-comments-carry-the-reasoning

- **Decision:** every new exported identifier and every new unexported helper carries a doc comment
  in this codebase's existing register: what it is, and *why* it has the shape it has — why the
  shadow struct exists rather than tags on the answer type, why the text renderer states its own
  byte contract instead of borrowing `RenderText`'s, why the re-parse branch after the argv rewrite
  is unreachable. Existing doc comments a card makes incomplete are updated in the same card:
  specifically `Run`'s numbered pipeline in `internal/cli/cli.go`, `usageText`'s own rules in
  `internal/cli/usage.go`, `internal/cli/doc.go`'s verb count, `quarry/doc.go`'s renderer and query
  counts **and** its "the package adds no behaviour of its own" claim, `quarry/repo.go`'s file
  header naming only Repo, Open and TOC, and `internal/cli/flags_test.go`'s
  `TestParseArgs_FourVerbGate` — whose name and doc comment both state a verb count this task
  changes. Each is named in the card that makes it stale.
- **Rationale:** the discussion's `## Constraints` names comment discipline explicitly, and the
  four doc comments listed above are prose specifications that a reader is entitled to trust; a
  stale one is worse than none.
- **Applies to:** all batches

### Decision: go-verify-commands-carry-no-pythonpath-prefix

- **Decision:** none of the three batch `verify:` commands carries the literal `PYTHONPATH= `
  prefix `mill-config.yaml` requires of a non-null per-batch `verify:`. They are Go commands, and
  the prefix is omitted deliberately, not by oversight.
- **Rationale:** the prefix exists so a Python test subprocess does not inherit the mill cache's
  `PYTHONPATH` and load stale cache modules instead of worktree ones. `go test` reads no
  `PYTHONPATH` at all, so the prefix would be inert decoration on a Go command. The validator check
  that enforces the rule, `verify-not-isolated`, is itself conditional on the project's language and
  does not fire for a Go repository — confirmed by running the validator against this plan before it
  was committed. The sibling plan in this hub, `wts/diff-to-symbols`, carries the same decision for
  the same reason; recording it here keeps the omission an auditable disposition rather than
  something a reviewer has to re-derive.
- **Applies to:** all batches

### Decision: verify-scope-and-the-done-gate

- **Decision:** each batch's `verify:` is scoped to the packages it touches — `./quarry/` for batch
  1, `./internal/cli/ ./quarry/` for batch 2, and `./internal/cli/ ./internal/engine/` for batch 3
  — the engine package appears only there, and only because its own Loomyard-pinned goldens are the
  second half of the regression gate `viewless-output-unchanged` names. The repo-wide gate
  `go test ./... && golangci-lint run` is already configured as `pipeline.done_gate` in
  `mill-config.yaml` and was confirmed green against this worktree's tip on 2026-09-05 before this
  plan was written, so it is left exactly as it is.
- **Rationale:** the batch verify runs after every implementer and fixer round, so it stays narrow;
  the done gate is what catches a regression in a package no batch verify covers.
- **Applies to:** all batches

## All Files Touched

- `docs/rewrite-plan.md`
- `docs/roadmap.md`
- `internal/cli/after_test.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/doc.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/glyphs_test.go`
- `internal/cli/testdata/INDEX.md`
- `internal/cli/testdata/glyphs-dir-text.txt`
- `internal/cli/testdata/glyphs-dir.txt`
- `internal/cli/testdata/glyphs-file-text.txt`
- `internal/cli/testdata/glyphs-file.txt`
- `internal/cli/testdata/toc-view-glyphs-depth.txt`
- `internal/cli/usage.go`
- `quarry/doc.go`
- `quarry/repo.go`
- `quarry/view.go`
- `quarry/view_test.go`
