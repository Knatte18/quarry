# Batch: benchmark-content

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
batch: benchmark-content
number: 6
cards: 5
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix|TestLoadLadder_RealKickstartFile'
depends-on: [1, 2, 4]
```

## Batch Scope

This batch authors every non-code artefact the matrix consumes: the ladder file, the task file, the
fasit, the three per-cell cards, and the offline pre-matrix gate that checks all of them before any
API call is spent. It is one batch because the six files are one design — the three cards must carry
identical glyph-name lists, the ladder file's glyph list must equal that list, the task file's
question must be answerable from exactly those glyphs, and the fasit must be the ground truth for
that question at that pin. Splitting them would let three of the six drift.

It depends on the config batch because the ladder file uses `control:`, `card:`, `pack:` and
`pack_targets:` and would not load without them, and on the sweep batch because the pre-matrix
blindness check renders all three prompts through the four-parameter render function and asserts the
blinding gate that the sweep widened to all three cells. It depends on the pack batch because the
treatment card's sentinel lines must be spelled exactly as the pack constants spell them, and the
pre-matrix gate extracts that card's block through the pack batch's own extractor.

The target repository is read through the path the harness itself resolves — the environment
variable, falling back to the untracked scratch env file. No file this batch creates may carry that
path.

Batch-local decision: the three `Uses:` lists being identical across the three cards is a property no
code enforces. It is checked by eye against the three files before rep 1, and the pre-matrix card
carries a test that checks it mechanically as well, so the eye check has a backstop.

## Cards

### Card 24: Author the ladder file

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create the ladder file with a header comment in the same register as the existing one: what this
  matrix tests, why it exists now, what each cell is, what is deliberately not here, and the exact
  commands to run it — the pack command first, then the run command.
  Run parameters mirror the existing file so the numbers sit on the same scale as the previous
  results roots: run model `claude-sonnet-5`, run effort `medium`, max turns `60`, scorer model
  `claude-opus-5` at effort `high`, and the source-repo literal the loader requires. Repetitions are
  `10` — the predeclared n, fixed before rep 1 and never grown after looking at any result.
  There is no server block and no tool list. Neither is required, no cell grants tools so the server
  is never built, and an absent tool list additionally removes any chance of the rendered-prompt
  blinding check firing on an ordinary English word that happens to be a tool name.
  One task entry, keyed `07-fabric-merge-state-tracing`, naming the task file and fasit this batch's
  other cards create, schema `exploration`, pinned to the full forty-character form of the commit the
  discussion names — the short form in the proposal is a prefix, and the file must carry the full
  SHA, matching how the existing file spells its own pin.
  A top-level `pack_targets` list of exactly these nine glyphs, in this order, which is also the
  order the cards' `Uses:` lists use:

  ```
  internal/fabriccli#addWeftVerbs
  internal/fabricengine#Fabric.MergeInProgress
  internal/fabricengine#Fabric.MergeContinue
  internal/fabricengine#mergeInProgressReason
  internal/fabricengine#Fabric.mergeRecordExists
  internal/fabricengine#Fabric.foreignMergeStatePresent
  internal/fabricengine#ErrMergeInProgress
  internal/gitrepo#Repo.MergeHeadPresent
  internal/mergeresolve#Resolver.Resolve
  ```

  Three configs, all under ladder letter `e`, all on that one task, all with an empty allowed list
  and their own card path into the cards directory: `e0-names` with `control: true`, `e1-pack` with
  `control: false` and `pack: true`, and `e2-files` with `control: false`.
  The two explicit `control: false` entries are mandatory, not stylistic. The default for a
  tool-less cell is control, so omitting the key would make all three controls, the loader would
  reject the file with the reworded exactly-one-control message, and the summary would build zero
  comparison rows. Write that reason as a comment beside them.
  One task key, not three, is what keeps the three arms on one pinned worktree: the run loop derives
  the worktree path from the task key and the rendered prompt names that path, so three task entries
  would make the arms differ by an unintended string — which is the confound the card mechanism
  exists to remove.
  This header is also where every note about the design lives, because a card file cannot hold one:
  the card loader reads a card whole and a markdown comment survives into the prompt as literal text,
  so a note written into a card would become an arm-only block of prompt text. Two notes in
  particular belong here. First, that the treatment minus the descriptive arm is not a clean spans
  contrast, since the treatment also carries the signature and the parallel-read instruction — which
  is exactly why the descriptive arm is declared descriptive and no test is run on it. Second, that
  the correctness gate is the summary-match flag alone, and that recall and precision are recorded
  but never compared across arms, because both non-control cards name the same seven files — the
  treatment inside its pack block, the descriptive one as a plain list — so both arms' file recall is
  inflated by construction and only the control's is earned.
- **Commit:** `feat(bench): add the kick-start ladder file`

### Card 25: Author the task file

- **Context:**
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create the task file following the existing task file's section order exactly: a title line, the
  type and status lines, a setup section naming the pin, a scope section, the task-text heading with
  its blockquote, the output-schema heading with its fenced JSON block, and a closing notes section
  for whoever prepares the fasit.
  The loader extracts exactly two things — the blockquote under the heading containing the task-text
  marker, dedented, and the first fenced JSON block after the output-schema heading — and its
  extraction is inclusion-based on purpose, so the notes section is never leaked into a prompt. Do
  not restructure the file to try to hide it.
  The question text goes in the blockquote and is transcribed verbatim from the discussion's own
  wording: the repository distinguishes a merge the fabric layer itself recorded from one merely
  present in the underlying git checkout because someone ran a merge there by hand, and the answer
  must cover four numbered points — which predicate computes each kind and what on-disk evidence each
  reads and why the read-only probe cannot substitute for the guard the mutating verbs consult; what
  a sibling mutating verb does while a fabric-recorded merge is in progress, which typed error
  carries that refusal, and how that differs from the foreign-state outcome; how the command-line
  layer surfaces the fabric-recorded state and which predicate it calls; and where the automated
  conflict resolver sits — what it is handed, what it does when the merge cannot be finished, and
  whether it participates in the bookkeeping at all.
  The question text is identical for all three arms; the card is what differs, and the card is
  rendered after this text.
  The output schema is the exploration schema, copied from the existing task file's own fenced block
  so the two stay byte-identical. It must carry the placeholder example values and must not name any
  real package path — the standing guard that no real target-repository path re-enters a prompt
  through the schema block.
  Neither the blockquote nor the schema block may contain the word this matrix blinds against. The
  pack's own content is target-repository paths and Go signatures, so this is a constraint on the
  prose only.
  The notes section records why this subject was chosen: the pin is literally the commit that
  surfaced merge-in-progress in fabric status, so the mechanism is present and coherent there; the
  answer requires holding several predicates and one typed error together rather than looking one
  thing up; it spans four packages and seven files; and it is untouched by tasks 01 through 06, so no
  existing fasit overlaps it. It also records the substitution rule verbatim, as card 29 will need
  it.
- **Commit:** `feat(bench): add the fabric merge-state tracing task`

### Card 26: Author the fasit

- **Context:**
  - `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.fasit.json`
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Author the fasit the way the existing one was authored: one dedicated reference pass at the pinned
  SHA with an unbounded budget, then a cross-check by a second independent method.
  Resolve the target repository path the way the harness does — the environment variable first,
  falling back to the untracked scratch env file directly beneath the repository root — and read the
  pinned content with git's own show verb against that path. Never edit the target checkout and never
  create a worktree of it by hand; reading a pinned blob needs neither.
  The reference pass must actually trace the mechanism, not paraphrase the question: read every one
  of the nine glyphs' declarations at the pin, plus enough of their callers to answer all four
  numbered points, and write the answer against the exploration schema's own keys — the relevant
  files, the key symbols with a one-sentence role each, a three-to-six-sentence summary explaining
  the mechanism end to end, a confidence, and any open questions.
  The cross-check is two things, both recorded in the meta block's role field: a clean build of the
  target repository at the pin, and a per-symbol confirmation that every symbol the fasit names is
  present at the pin. Both were performed during planning against all nine glyphs and all seven
  files; redo them rather than trusting that record, since the fasit is what every rep is scored
  against and a stale confirmation is worth nothing.
  The meta block carries the same keys the existing fasit's does — task, type, pinned SHA, scope,
  date, arm, role, see-also — and is dropped before scoring, so it may name this repository freely.
  Everything outside the meta block is scored, so it must not.
  The relevant-files list must be exactly the seven files the nine glyphs live in. That list is also
  what one card's own file list is derived from, and the two must agree.
  A fasit whose scored keys are thin deflates recall uniformly across all three arms and hides the
  separation this matrix exists to find, so err toward completeness in the key-symbols list rather
  than brevity.
- **Commit:** `feat(bench): add the fabric merge-state tracing fasit`

### Card 27: Author the three cards

- **Context:**
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json`
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/cards/07-e0-names.md`
  - `bench/loomyard-eval/cards/07-e1-pack.md`
  - `bench/loomyard-eval/cards/07-e2-files.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  All three cards open with the same `Uses:` heading and the same nine glyph names in the same order
  as the ladder file's glyph list, byte for byte identical across the three files. Everything below
  that differs, and nothing else may.
  The control card is the `Uses:` list and nothing else. No locations, no file names, no counts.
  The descriptive card is the `Uses:` list plus a `Files:` list of the seven distinct file paths the
  nine glyphs live in, deduplicated, with no per-glyph mapping and no line numbers. This is the plan-
  card format the operator's own tooling writes today, which is what this arm exists to represent —
  giving it a per-glyph file mapping would make it a third treatment rather than that format.
  The treatment card is the `Uses:` list, then the two pack sentinel lines with an empty region
  between them, then one instruction: read all listed spans in parallel, in one turn, before doing
  anything else. The sentinels must be spelled exactly as the pack constants spell them, on their own
  lines, once each, in order — the pack command writes between them and the run gate hashes what is
  between them, and a card whose sentinels are misspelled fails both.
  The parallel-read instruction lives in this one card only. Front-loading reads is possible only
  when locations are known, so it is the mechanism under test rather than a prompt trick withheld
  from the other two — a card with no locations cannot follow it. The longer prompt's input-token
  cost is part of the treatment and is captured by the cost metric.
  No card may contain the word this matrix blinds against, nor the server name, both of which are the
  same token here. All three cards are rendered into prompts that the pre-dispatch blinding check
  now covers, and a finding there voids the repetition before any call is spent.
  A card file has no comment mechanism: the card loader reads the file whole with no extraction, and a
  markdown comment line survives into the prompt as literal text. So a card contains prompt text and
  nothing else — no note to a human reader, no explanation of the experiment, no marker of which arm
  it is. A block of that kind in one card only would be an arm-only difference describing the
  measurement inside the measurement, which is the exact class of confound the one-task-one-prompt
  design exists to remove. Every such note belongs in the ladder file's header comment, which card 24
  writes and which no prompt ever reads.
- **Commit:** `feat(bench): add the three kick-start cell cards`

### Card 28: Extend the pre-matrix gate to the new file

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
  - `bench/loomyard-eval/cards/07-e0-names.md`
  - `bench/loomyard-eval/cards/07-e1-pack.md`
  - `bench/loomyard-eval/cards/07-e2-files.md`
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json`
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Extend the pre-matrix suite so the new committed file gets the same offline treatment the existing
  one does. It is keyed on real committed artefacts on purpose, so an authoring mistake is caught for
  free rather than after thirty real runs.
  Extend the control-prompt blindness test to load the new ladder file too and, for every one of its
  three cells — not only the control, since the blinding gate now covers every cell that grants no
  tools — render the full prompt the run loop would send, card included, and assert it passes the
  rendered-prompt blinding check. This is the only check that catches a card carrying the blinded
  token before it voids a cell for the whole matrix.
  Extend the fasit well-formedness test to cover the new fasit: it decodes, carries the exploration
  schema's scored keys, has a non-empty relevant-files list and a non-empty key-symbols list with
  well-formed entries, and its meta pinned SHA matches the pin the loaded ladder file records for
  that task.
  Add `TestPreMatrix_KickstartCardsShareOneUsesList`, which reads the three card files and asserts
  their `Uses:` sections are byte-identical. The three lists being identical is the property that
  makes the arms differ only in the dimension under test, and no other code enforces it.
  Add `TestPreMatrix_KickstartUsesListMatchesPackTargets`, which parses each card's `Uses:` entries
  and asserts they equal the loaded ladder file's own glyph list element for element, in order. This
  is a different invariant from the one above and needs its own assertion: three cards can agree with
  each other while all three disagree with the ladder file. That is exactly the state the glyph
  substitution procedure risks producing, since it edits the ladder file's list and the three cards'
  lists as separate steps, and its consequence is a treatment card whose generated block names glyphs
  its own list does not — an arm difference in the dimension under test, introduced by the correction
  meant to prevent one.
  Add `TestPreMatrix_KickstartPackCellCardHasSentinels`, asserting the treatment card's block
  extracts cleanly through the pack block extractor — the authoring-time half of the run gate, which
  otherwise first fails at matrix start.
  Add `TestLoadLadder_RealKickstartFile` to the config test file, mirroring what the existing
  real-file test does for the old file: all three cell ids, the empty tool list, exactly one control,
  the two explicit non-controls, the single pack cell, the nine-entry glyph list in order, the task
  entry's schema and pin, and the three card paths.
- **Commit:** `test(bench): gate the kick-start ladder file offline before the matrix`

## Batch Tests

`verify:` runs `go test` against `./bench/loomyard-eval/ladder/internal/ladder/` with
`-run 'TestPreMatrix|TestLoadLadder_RealKickstartFile'`.

The pre-matrix suite is the whole gate for this batch, because everything this batch produces is
data rather than code: the ladder file is checked by loading it through the real loader and its
validation rules, the three cards are checked by rendering the three real prompts and running the
real blinding check over them, the treatment card's sentinels are checked through the real extractor,
the three cards' glyph lists are checked against each other and against the ladder file's own list,
and the fasit is checked for the shape the scoring rule computes recall and precision against. The
new real-file load test is named separately in the pattern because it lives in the config test file
rather than the pre-matrix one.

What these tests deliberately do not check is judgment: whether the question is a good question and
whether the fasit's answer is correct and substantive. That is the fasit card's own work, verified by
the reference pass and its two cross-checks, and is not attempted here.
