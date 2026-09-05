# Batch: goldens-and-docs

```yaml
task: 'P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)'
batch: 'goldens-and-docs'
number: 3
cards: 5
verify: LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ ./internal/engine/
depends-on: [2]
```

## Prior failure

- Round 1: "Card 15 requires go test ./internal/engine/ to pass with Loomyard cases executed, but
  TestRoundTrip_LoomyardNaming fails on a missing testdata/loomyard/naming-counts.json golden that
  is confirmed pre-existing (identical failure reproduces on main with the same pinned checkout)
  and unrelated to this task; the only fix is generating that golden under
  internal/engine/testdata/, forbidden by the overview's no-file-under-internal-engine-is-modified
  Shared Decision (applies to all batches)."

## Batch Scope

This batch turns the working verb into committed evidence and updates the two documents that
describe it. Five new golden files over the pinned Loomyard checkout, their rows in the golden
table and in the before-to-after index, the index's own counts and the paragraph distinguishing
this view from the retired compact view, the rewrite plan's section 5, and the removal of the
roadmap point this task closes. It is one batch because the golden table row and the golden file it
produces cannot be split across a boundary without leaving a red window that outlives the batch,
and because the two documents describe exactly the artefacts the first three cards produce.

This batch adds no product code. Every card either writes a test-table row, generates a fixture
from it, or edits prose.

Batch-local decisions beyond `## Shared Decisions` in the overview:

- **The depth golden's target is named by the plan, not chosen by a probe at implementation time.**
  The discussion left it to an ordered candidate probe because it was written without a checkout to
  settle it from. That probe was run while this plan was under review, against a clone of
  `/home/knatte/Code/loomyard/wts/loomyard` detached at `72c23d9`, and its result retires the rule
  rather than confirming it: of the whole repository at that pin, exactly three directories have a
  subdirectory that itself holds Go files, and the discussion's own fallback is the worst of the
  four candidates, not the safe one. Card 14 carries the measurements and the single target they
  select. Do not re-run the candidate ladder.
- **The five new goldens open a red window inside this batch, exactly as the existing fifteen did.**
  `internal/cli/after_test.go`'s own file comment already documents and justifies that window for
  its own batch. Card 14 adds five rows that fail until card 15 generates their files; the batch's
  `verify:` runs only once every card has landed. Do not paper over the window by skipping on a
  missing golden — after card 15, a missing golden is a real regression.

## Cards

### Card 14: the depth target probe and the five new golden table rows

- **Context:**
  - `internal/cli/loomyard_test.go`
  - `internal/cli/testdata/INDEX.md`
  - `internal/cli/flags.go`
  - `internal/cli/cli_test.go`
- **Edits:**
  - `internal/cli/after_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** The depth golden's target and format are fixed by this card and are not
  re-derived: the invocation is `toc --text --view glyphs --depth 1 internal/boardengine`.

  Do not run the discussion's candidate ladder. It was run during this plan's own review, against a
  clone of the source repository detached at `72c23d9`, and every one of its four candidates fails:
  `internal/shedadapters` has no subdirectory at all at that pin, so it can never satisfy the
  "a `dirs` entry that itself carries `files`" test; `internal/reedengine` has one but 67 Go files
  in scope and 616 symbols at `--depth 1`; `internal/fabricengine` is larger still; and the ladder's
  own guaranteed fallback, the repository root at `--depth 1`, yields **zero** symbols, because no
  Go file sits at the root or one level below it. A zero-symbol depth golden would assert the exact
  opposite of what this row exists to assert. Exactly three directories in the repository at that
  pin have a subdirectory holding Go files — `internal/boardengine` (18 own files plus 6 in
  `boardtest`), `internal/shuttleengine` (22 plus 18), and `internal/reedengine` (67 plus 12) — and
  `internal/boardengine` is the smallest of the three.

  The format is `--text` rather than JSON, and that is the size bound doing its job rather than a
  stylistic preference: `internal/boardengine` at `--depth 1` carries 172 symbols, which is 172
  lines of the glyphs text view and roughly 1,200 lines of its JSON, against a bound of about 200
  and a largest existing golden of 254 lines (`toc-file.txt`). No JSON coverage is lost by the
  choice — two of the four other new goldens are JSON, and card 10's machine-independent
  `internal/cli/cli_test.go` cases already assert the JSON key set for a `toc --view glyphs`
  invocation with no symbols flag.

  What the row still proves is unchanged and is worth stating in a comment beside it: 172 symbols
  rather than zero is the assertion that the view's implicit symbols default works, and those
  symbols span `internal/boardengine` and `internal/boardengine/boardtest`, which is the assertion
  that the flat projection crosses directory boundaries and fills each line's file prefix. Note
  that `internal/logger`, the target the other four goldens use, cannot serve here: its committed
  golden has no `dirs` key at this pin, so it has no subdirectory and a depth golden there would
  prove neither property.

  If the generated golden turns out not to match these numbers, stop and report it rather than
  adjusting the target: the pin is fixed, so a mismatch means the measurement or the pin resolution
  is wrong, and picking a different directory would bury that.

  Then add five rows to `afterGoldenCases` in `internal/cli/after_test.go`, each with
  `exitCode: exitOK`, each spelling its `invocation` suffix literally rather than deriving it from
  `verbArgs`, exactly as every existing row does — that literal spelling is what keeps a
  machine-specific `--root` out of a committed golden:

  - golden `glyphs-dir.txt`, verb `glyphs`, invocation `internal/logger`;
  - golden `glyphs-dir-text.txt`, verb `glyphs`, invocation `--text internal/logger`;
  - golden `glyphs-file.txt`, verb `glyphs`, invocation `internal/logger/logger.go`;
  - golden `glyphs-file-text.txt`, verb `glyphs`, invocation `--text internal/logger/logger.go`;
  - golden `toc-view-glyphs-depth.txt`, verb `toc`, invocation
    `--text --view glyphs --depth 1 internal/boardengine`.

  Each row's fifth field, `verbArgs`, is the row's own `invocation` string split on single spaces —
  `TestAfterGoldens` builds Run's argv from `verbArgs` while the `invocation` string is only ever
  written into the golden's first line, so the two must describe the same invocation or the golden
  records a command it did not run. Spell `verbArgs` out per row as a literal `[]string`, the way
  every existing row does; do not compute it from `invocation` at test time, since deriving one from
  the other is what would let a divergence go unnoticed.

  The fifth row carries no `--symbols` token, and that omission is deliberate and load-bearing: the
  view supplies that default, and a non-empty symbol list in this golden *is* the assertion that
  the default works. A version of this row that spelled `--symbols` would pass whether the default
  works or not. Put that sentence in a comment above the row, because it is the one row a later
  editor would "fix" by adding the flag. The row is a `toc` invocation rather than a `glyphs` one
  because the preset is frozen at `--depth all` and rejects `--depth`.

  Update this file's own header comment: it says "the fifteen testdata/ golden files" twice, and
  the count is now twenty. Update both, and extend the sentence that says the table spans three
  verbs — it spans four now, and the fourth is a preset over the first.
- **Commit:** `test(cli): five glyphs golden table rows and the depth target probe`

### Card 15: generate the five golden files

- **Context:**
  - `internal/cli/after_test.go`
  - `internal/cli/loomyard_test.go`
  - `internal/cli/testdata/INDEX.md`
- **Edits:** none
- **Creates:**
  - `internal/cli/testdata/glyphs-dir.txt`
  - `internal/cli/testdata/glyphs-dir-text.txt`
  - `internal/cli/testdata/glyphs-file.txt`
  - `internal/cli/testdata/glyphs-file-text.txt`
  - `internal/cli/testdata/toc-view-glyphs-depth.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Generate the five files card 14's rows name, by running the regeneration command
  the index documents, with the pinned checkout this task created:
  `LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ -run TestAfter -update`.
  The `TestAfter` prefix match is load-bearing on the test function's name; do not rename it.

  Then enforce the regression gate, which is this task's own done-criterion and the only mechanical
  proof that the view is additive. The discussion's `viewless-output-unchanged` decision names both
  golden sets, not one: `git status --porcelain internal/cli/testdata/ internal/engine/testdata/`
  must show exactly the five new untracked files under the first and **no modification to any
  pre-existing golden under either**. The engine half is not decoration even though no card in this
  plan edits `internal/engine/`: those goldens are gated on the same environment variable and have
  therefore never run on this machine either, so "unchanged" is a claim this task is the first to be
  in a position to check. Run them: `LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test
  ./internal/engine/` must pass with its Loomyard cases executed rather than skipped.

  If any pre-existing file under either directory was rewritten by the `-update` run, that is a
  defect introduced by this task's product code, not a golden that needs updating — revert the
  whole `-update` run, fix the code, and regenerate. Under no circumstances commit a modified
  pre-existing golden.

  Inspect the five generated files before committing them, since `-update` writes whatever the code
  produced: each is exactly the invocation line beginning `$ quarry `, a blank line, and stdout
  verbatim, with no exit-code trailer; no absolute path appears anywhere in any of them; the two
  JSON files carry a `symbols` array and no `signature`, `doc` or `sigend` key on any symbol
  object; the two text files carry only symbol lines of the documented shape, with no directory
  line, file line, docstring or signature; and `toc-view-glyphs-depth.txt` carries a **non-empty**
  symbol list, which is the assertion its missing `--symbols` token exists to make — an empty one
  means the view's symbols default is not working and the code, not the golden, is wrong. Check the
  depth golden against the roughly-200-line size bound; if it exceeds it, card 14's candidate
  selection was wrong and must be redone, not accepted.

  Finally re-run the full package test without `-update` and confirm all twenty golden cases pass.
- **Commit:** `test(cli): the five glyphs goldens over the pinned Loomyard checkout`

### Card 16: the before-to-after index

- **Context:**
  - `internal/cli/after_test.go`
  - `internal/cli/testdata/glyphs-dir.txt`
  - `internal/cli/testdata/glyphs-dir-text.txt`
  - `internal/cli/testdata/glyphs-file.txt`
  - `internal/cli/testdata/glyphs-file-text.txt`
  - `internal/cli/testdata/toc-view-glyphs-depth.txt`
- **Edits:**
  - `internal/cli/testdata/INDEX.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** The index's before-to-after table is total — every after-side file has its own
  row and its own exit-code cell — so it gains one row per new golden, each in the existing
  `| *(none)* | <file> | 0 | new: <what it shows> |` shape, since none of the five has a before-side
  counterpart. The depth row's note must name `internal/boardengine` and say why that subtree
  rather than `internal/logger` — which has no subdirectory at this pin — so a reader can see which
  subtree the depth case actually covers; it must also state why the row is a `toc` invocation
  rather than a `glyphs` one, and why it is the one new golden in text form only.

  Update the counts: the table's own prose says the table spans three verbs — it spans four now —
  and the closing section says "These fifteen files", which becomes twenty. Both the index and
  `internal/cli/after_test.go`'s header comment carry that count and must agree.

  Add one sentence distinguishing this view from the retired compact view, placed with the existing
  **The compact view is gone, not replaced.** paragraph so the two claims are read together. The
  distinction is not stylistic: the compact view was lossy about the *content* of what it listed —
  one prose sentence per file, from which an agent then answered questions, which is where the
  measured 0.96 to 0.82 precision drop came from. The glyphs view drops fields but never drops a
  symbol, and its consumer looks up spellings rather than answering from it; the `incomplete` list
  exists precisely so that "this symbol is not in the target" stays a sound conclusion. State that,
  so the new rows cannot be read as the compact view returning under a new name.
- **Commit:** `docs(cli): index the five glyphs goldens and distinguish the view from compact`

### Card 17: the rewrite plan's section 5

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `quarry/view.go`
- **Edits:**
  - `docs/rewrite-plan.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In section 5, "The queries", the `toc` paragraph's bold-led flag list currently
  reads `toc <dir|file> [--depth N|all] [--symbols]`. Add `[--view full|glyphs]` to it, in the
  position the help text uses.

  Then add the two paragraphs the task requires, in the same bold-led register as the existing
  per-verb paragraphs, placed after the `toc` paragraph and before the `name` paragraph:

  - **the view mechanism.** `--view` selects a projection of the one complete answer; extraction
    underneath is unchanged, and the projection is a pure function applied after the query returns,
    so no view can influence what is extracted. The `glyphs` view is flat and non-recursive: one
    entry per symbol carrying id, kind, file and span, plus an explicit list of files that could not
    be read or fully parsed. That `incomplete` list is what makes "an absent entry means the symbol
    is not in the target" a sound conclusion rather than a guess. State the scope of that promise in
    the same paragraph, because the promise and its limit must travel together: it holds for the
    frozen `glyphs` preset, which is `--depth all`, and there only — a direct
    `toc --view glyphs --depth N` answer is truncated by construction and contributes nothing to
    `incomplete`, because a depth-cut answer is indistinguishable from a genuinely empty leaf.
  - **the preset rule.** A named verb is a frozen flag preset over the one query, never a second
    implementation: `quarry glyphs <target>` is spelled literally as
    `quarry toc --view glyphs --depth all --symbols <target>`, and the CLI reaches it by rewriting
    its own argument slice and re-parsing, so nothing below the parser can tell the two apart. That
    property is enforced by a test requiring byte-identical stdout, stderr and exit code for both
    spellings, over a file target and a directory target in both formats, rather than by a
    convention a later edit can quietly break. `--symbols` is spelled explicitly in the expansion
    rather than left to the default, because the per-target default is false for a directory
    target; the preset does not silently change if a default ever does.

  Do not change any other section of this file, and do not restate the exit-code or envelope rules
  sections 4 and 5 already carry.
- **Commit:** `docs: rewrite-plan section 5 gains the view mechanism and the preset rule`

### Card 18: close roadmap point 2a

- **Context:**
  - `docs/rewrite-plan.md`
- **Edits:**
  - `docs/roadmap.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Remove point 2a — the bullet beginning **2a. The `glyphs` verb — the planner's
  index.** — from `docs/roadmap.md` in its entirety. The file's own header says it "only ever says
  what is ahead", and this task is what makes 2a past; the build record lives in git history, not
  here.

  Only point 2a is removed. Leave the text of points 2b and 2c byte-for-byte as it is, and do not
  add, adjust or infer a status marker on either: neither point carries one today, and this task
  has no standing to record anything about their state. Then check the introductory paragraph of
  point 2 for what the removal makes wrong: it calls the group "three plan-unaware surfaces" and
  says "Any order among the three", and both readings must be adjusted to the two points that
  remain. Make no other change to this file —
  in particular, do not touch the standing-rule paragraph or point 1, and do not add a "done"
  marker for 2a, since recording completion is exactly what this file says it does not do.
- **Commit:** `docs: remove roadmap point 2a, closed by the glyphs verb`

## Batch Tests

`verify: LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ ./internal/engine/`
runs two package test binaries. `./internal/cli/` is where the batch's three test-and-fixture cards
land. `./internal/engine/` is here for one reason only: its own Loomyard-pinned goldens are the
other half of the regression gate the discussion's `viewless-output-unchanged` decision names, and
they are gated on the same environment variable, so this is the only place in the plan they ever
execute. `./quarry/` is deliberately not in scope: this batch adds no product code and touches no
file that package compiles or reads, so including it would only lengthen the loop that runs after
every implementer and fixer round. The repo-wide `pipeline.done_gate` covers the rest of the tree
at the end of the task — but note it sets no `LADDER_LOOMYARD_REPO`, so it re-runs both packages
with every checkout-gated case skipping; this batch's own verify is where those cases actually run.

The environment variable is what makes this batch's verify mean anything at all. Without it,
`loomyardRepo` skips and all twenty golden cases — the five this batch creates included — plus every
engine-side Loomyard case report as skipped rather than run, so a wrong golden would pass. Card 6 in
batch 2 created the checkout it points at; if `.scratch/loomyard-pin` is missing when this batch
runs, stop and recreate it per that card rather than accepting a skipped run.

The two prose cards, 17 and 18, have no runnable surface: `docs/rewrite-plan.md` and
`docs/roadmap.md` are not compiled, linted or tested by anything in this repository. They are
verified by review against the artefacts they describe — the preset tokens in
`internal/cli/flags.go`, the help text in `internal/cli/usage.go`, and the `incomplete` semantics in
`quarry/view.go` — which is why each of those files is in the cards' `Context:`.

The regression gate this batch exists to satisfy is not expressible as a test command: it is that
`git status` shows no pre-existing golden modified. Card 15 states it as an explicit step with an
explicit failure disposition, because a green `go test` run after a wrongly-accepted `-update` would
prove nothing.
