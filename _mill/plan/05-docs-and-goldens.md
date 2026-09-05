# Batch: docs-and-goldens

```yaml
task: "Glyph self-form and the resolve contract (C1)"
batch: "docs-and-goldens"
number: 5
cards: 6
verify: LADDER_LOOMYARD_REPO=/home/knatte/Code/loomyard/wts/loomyard go test ./glyph/... ./internal/cli/...
depends-on: [4]
```

## Rename mechanic

For each `Moves:` pair the implementer MUST:

1. Run `git mv <old> <new>` FIRST, before making any other change to the moved file.
2. Make ONLY surgical edits — touch only the lines that must change after the move.
3. Use a full-file `Creates:` entry only for genuinely new files that have no predecessor.
4. Never write the relocated file from scratch and delete the original — that breaks git rename
   history and inflates review diffs.

For the one pair in this batch the surgical edit is performed by the golden regeneration rather than
by hand: `git mv` first, then the `-update` run rewrites the moved file's bytes in place.

## Batch Scope

This batch writes the contract down. `docs/glyph.md` is the contract — its own first section says
anything not stated there is not part of it — so the self form and the `Self` constructor are not
real until they appear there (D15). `docs/rewrite-plan.md` says how quarry implements the contract
and carries two sentences the task reverses (D16). The `after/` goldens are the committed byte-level
evidence and must be regenerated against the pinned checkout rather than edited (D17). The
docs-as-tests table makes the done-when clause "every example is a test" satisfiable by one
reviewable list.

It comes last because every example it writes down must already work: a doc example that the parser
rejects is a failing test in the same batch.

Batch-local decision beyond the overview's Shared Decisions: the per-language self-form rows for
Python and C# are documentation only. There is no extractor to test either alphabet against, so
card 36's table asserts the Go rows and carries the two non-Go rows as commented-out placeholders
naming the task that will enable them, rather than silently omitting them.

## Cards

### Card 33: `docs/glyph.md` §1 and §2 — the form and the unit

- **Context:**
  - `glyph/parse.go`
  - `glyph/glyph.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `docs/glyph.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In §1, the form line gains the self form, and the sentence stating that any glyph
  splits at its first `"#"` becomes one stating that a glyph contains exactly one `"#"` and splits
  there. The example table gains two rows: `internal/reedengine/render#`, the package itself, and
  `internal/reedengine/render/focus.go#`, the file itself. In §2, the unit table defines the Go unit
  as the package directory, so a file's self glyph names a left half that is not a unit under §2 as
  written — that is the sentence this section must fix. Add a paragraph stating that in the self form
  the left half is the thing's own repository-relative path or unit name, whichever the language
  spells, with one row per language for what a trailing `"#"` means: for Go, a package directory or a
  file, both spelled as repository-relative paths, since Go's unit already is one; for Python, the
  module or package itself, which is the file, so Python has no separate file self glyph; for C#, the
  namespace itself, and since C# has no file-level unit it has no file self glyph at all. State in
  one sentence that the external test unit is addressable as a member glyph but has no self form of
  its own, because the self form names a path and the `_test` pseudo-path is not one. Leave §2's
  existing unit corner cases alone.
- **Commit:** `docs(glyph): specify the self form in the form and unit sections`

### Card 34: `docs/glyph.md` §3, §5, §6 and §7 — the member, resolution, the API and the neighbours

- **Context:**
  - `glyph/self.go`
  - `glyph/glyph.go`
  - `internal/engine/walk.go`
  - `internal/cli/doc.go`
- **Edits:**
  - `docs/glyph.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In §3, add a short paragraph above the per-language sections stating that an
  empty member is the self form in every alphabet — that much is shared — and that for Go, and only
  for Go, removing the trailing `"#"` yields the plain repository-relative path in both directions
  with no other conversion. Scope that path-conversion sentence to Go explicitly: it holds only
  because Go's unit is spelled as a repository-relative path, and stating it unscoped would assert
  something false of a Python dotted module and a C# namespace. In §5, replace the closing paragraph
  that says `resolve` also accepts a repository-relative path told apart by having no `"#"` with the
  new contract: `toc` takes paths, `resolve` takes glyphs, a bare path is rejected with a message
  naming the fix, and a self glyph answers with the listing block and the same statuses. Add the rule
  that a `"#"` in a path segment is an explicit error at both verbs, and the asymmetry it lives
  beside — the contract governs what a caller may name, while the walk's own spellability rule
  governs what it may mint, so a directory whose name carries a `"#"` is still listed without symbols
  when it is encountered below a listed target. State in §5 that the repository root cannot be
  addressed by `resolve` at all, since a lone `"#"` fails as an empty unit and a dot segment is
  rejected, and that `toc` on the root remains the way to ask. In §6, add that a trailing `"#"` is
  safe in every format already listed and is never optional, since the canonical form keeps it, and
  extend the Go API list — which today names the language type, the glyph struct, `Parse` and
  `Glyph.String()` and nothing else — with `Self(lang Language, path string) (Glyph, error)` and
  `Glyph.IsSelf() bool`, with their signatures. That list is load-bearing: §1 says anything not
  stated in this document is not part of the contract, so leaving them out would put the compose
  constructor outside the contract it exists to be. In §7, rewrite the bullet about file paths, which
  states the retired rule that a repository-relative path has no `"#"` and that this is how `resolve`
  tells the two apart.
- **Commit:** `docs(glyph): specify the self form's member, resolution, API and neighbours`

### Card 35: `docs/rewrite-plan.md` §4 and §5

- **Context:**
  - `docs/glyph.md`
  - `glyph/self.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `docs/rewrite-plan.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Three sentences change, not one. In §4, replace the sentence saying a target
  without a `"#"` is a path and one with a `"#"` is a glyph with the new rule: `toc` takes paths,
  `resolve` takes glyphs, and a trailing `"#"` addresses a unit or file as itself. Leave the
  neighbouring sentence about `toc` listing every non-gitignored file as it is. Rewrite the next
  bullet, which claims that what `toc` lists is what `resolve` takes: `toc`'s file and directory
  entries carry no identifier — only a symbol entry does — so once `resolve` takes glyphs only, a
  consumer holding a listing cannot feed a file entry back to `resolve` without concatenating the
  directory, a slash, the name and a `"#"` by hand, which is the printing the one-implementation rule
  forbids outside package `glyph`. The replacement states how the round trip is actually made: a
  symbol entry's identifier is already a glyph, and a file or directory entry's self glyph is built
  from its path with the `Self` constructor. This matters more than it looks — the adoption this task
  is timed for is precisely the consumer that walks a listing and resolves what it finds. In §5,
  change the `resolve` heading's argument from a glyph-or-path alternation to a glyph alone, and add
  one sentence to its body: a bare path is rejected pre-resolution with a message naming the fix, and
  a trailing `"#"` is how a whole file is a plan target. Note the separator reject on §5's `toc` line
  and leave the rest of that line as it is. Leave §5's performance numbers and its phase-2 parking
  untouched.
- **Commit:** `docs(plan): replace the retired target-classification rule`

### Card 36: the docs-as-tests table

- **Context:**
  - `docs/glyph.md`
  - `glyph/parse.go`
  - `glyph/errors.go`
  - `glyph/glyph.go`
  - `glyph/self.go`
- **Edits:** none
- **Creates:**
  - `glyph/docs_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create one table in package `glyph` enumerating every example and every reject
  that appears in the rewritten `docs/glyph.md` §1, §2, §3 and §5, so the done-when clause that every
  example is a test is satisfied by one reviewable list rather than by assertions scattered across
  files. Each accept row names the glyph string and the parsed shape the document claims for it; each
  reject row names the string and the reason. Cite the section each row came from in the row itself,
  the way the existing reject tables in this package already carry a section field, so a reader can
  check the list against the document without guessing. The §2 per-language self-form rows for Python
  and C# are documentation only — there is no extractor to test either alphabet against — so carry
  them as commented-out placeholders naming the task that will enable them, and assert the Go rows.
  The file header states that this table is the executable form of the document and that a row added
  to the document without a row here is the drift this file exists to catch.
- **Commit:** `test(glyph): make every documented example a row in one table`

### Card 37: regenerate the `after/` goldens

- **Context:**
  - `docs/research/output-formats/after/resolve-glyph.txt`
  - `docs/research/output-formats/after/toc-dir.txt`
  - `internal/cli/loomyard_test.go`
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/cli/after_test.go`
- **Creates:**
  - `docs/research/output-formats/after/resolve-self-dir.txt`
  - `docs/research/output-formats/after/resolve-self-file-text.txt`
  - `docs/research/output-formats/after/resolve-self-dir-text.txt`
- **Deletes:** none
- **Moves:**
  - `docs/research/output-formats/after/resolve-path.txt` -> `docs/research/output-formats/after/resolve-self-file.txt`
- **Requirements:** Run `git mv` on the pair above first; the old name no longer describes the file,
  which stops showing a path target. In `internal/cli/after_test.go`, retarget that row: its golden
  becomes the new name, its invocation and its verb arguments become the same file path with a
  trailing `"#"`, and its expected exit code stays 0. Add three rows — a directory self glyph, and
  the text view of each self glyph — naming the three new golden files, with the same repository-
  relative targets and the `--text` flag where the row's name says so. Every row spells its own
  invocation suffix literally rather than deriving it from the argument slice, which is the property
  that keeps a machine-specific root out of a committed golden; follow that convention. Update the
  file header, which counts twelve golden files and calls them the task's committed evidence: it is
  fifteen after this card. Then regenerate with
  `LADDER_LOOMYARD_REPO=/home/knatte/Code/loomyard/wts/loomyard go test ./internal/cli/ -run TestAfter -update`,
  which rewrites every golden from the pinned checkout. Write no golden byte by hand. After the run,
  read `git status` and `git diff` and confirm that only the four self-form goldens changed: the
  method, type and not-found resolve goldens and the four `toc` goldens are re-verified by the same
  run and must come back byte-identical, so a diff in any of them is a real regression from an
  earlier batch rather than an expected update. The checkout at that path is at the commit
  `after_test.go` pins, so the run compares rather than skipping.
- **Commit:** `test(cli): regenerate the after/ goldens for the self form`

### Card 38: `after/INDEX.md` follows the table

- **Context:**
  - `internal/cli/after_test.go`
  - `docs/research/output-formats/after/resolve-self-file.txt`
  - `docs/research/output-formats/after/resolve-self-dir.txt`
- **Edits:**
  - `docs/research/output-formats/after/INDEX.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** The before-to-after mapping table in this file is described as total — every
  before-side file has a row naming its successor or stating why it has none, and every after-side
  file has its own exit-code cell — and that totality is what makes the coverage claim checkable.
  Retarget the row for the renamed golden to the new file name and rewrite its note: the row's
  subject is no longer a repository-relative path as a target but a file addressed as its own glyph.
  Add rows for the three new goldens with their exit codes. Update the sentence naming the directory
  as one real invocation each of a fixed number of files, which changes with the three additions.
  This file is the table's own description and `internal/cli/after_test.go`'s header forbids putting
  that description anywhere else, so it changes in the same batch as the table.
- **Commit:** `docs(research): index the self-form goldens`

## Batch Tests

`verify: LADDER_LOOMYARD_REPO=/home/knatte/Code/loomyard/wts/loomyard go test ./glyph/... ./internal/cli/...`
covers the two packages this batch touches code in: `glyph`, for the new `docs_test.go` table, and
`internal/cli`, for `after_test.go`'s retargeted and added rows.

The environment variable is set for this batch alone, and deliberately. Without it
`TestAfterGoldens` skips, so a golden regenerated wrongly would go unnoticed here and surface only on
another machine; with it the comparison genuinely runs against the checkout at
`72c23d9eecc1fa55add567622093a8bbbfba8c1d`, which is the commit `loomyardPin` names. Every other
batch leaves the variable unset, so no other batch depends on the checkout existing.

The two docs files this batch edits have no runnable surface of their own. `glyph/docs_test.go` is
what makes them checkable: it is the executable form of `docs/glyph.md`'s examples, so a claim added
to the document that the parser rejects fails here rather than drifting silently.
