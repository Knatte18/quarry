# Batch: facade-delta

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'facade-delta'
number: 6
cards: 7
verify: go test ./quarry/ -run 'TestDelta' && go test ./internal/mcpserver/ -run 'TestFacadeOnly|TestStdout'
depends-on: [2, 4, 5]
```

## Batch Scope

This batch puts both delta methods on the public facade: the pure `Delta`, which delegates to the
engine unchanged as every other facade method does, and the convenience `DeltaGit`, which drives the
git layer for paths and bytes, turns those bytes into package clauses through the engine's exported
helpers, derives each side's units, and then calls `Delta`.
It also declares the one new type that is a facade type rather than an engine alias, aliases the
delta answer types and the git error identity, and corrects the two doc comments that will otherwise
be false the moment `DeltaGit` lands.

It depends on all three of the independent roots and on the core: batch 2 for the clause and unit
helpers, batch 4 for the core method, batch 5 for the git plumbing.

`DeltaGit` is the one place in this task where a layering claim is genuinely traded away, and cards
31 and 36 exist to record that honestly rather than let two doc comments keep asserting something
that stopped being true.

The external interface batches 7 and 8 consume: the wrapped answer type both renderers take, and
`(*Repo).DeltaGit(from, to, target string) (GitDeltaAnswer, error)`, whose errors carry the facade's
own aliased git error identity.

Batch-local decision: `GitDeltaAnswer` is declared in package `quarry` rather than in the engine,
because the engine is defined as knowing nothing about git and a revision-bearing type there would
contradict that; it is the single documented exception to the facade's aliases-only habit.

## Cards

### Card 31: alias the delta answer types and the git error identity

- **Context:**
  - `internal/engine/delta_answer.go`
  - `internal/gitsrc/errors.go`
  - `quarry/doc.go`
- **Edits:**
  - `quarry/quarry.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add aliases for every delta answer type the engine declares — the entry type,
  the answer, the file echo, the two closed vocabularies, the location block, the modified entry,
  the renamed pair, the candidate entry, the candidate and the signals — following the existing
  alias convention in this file exactly: an alias, never a defined type, so an external importer can
  name the shape without importing the internal engine package.
  Add the closed vocabularies' constant values as aliases too, mirroring how the existing kind and
  status constants are re-exported here.
  Alias the git layer's three sentinels and two typed errors the same way, as values and type
  aliases rather than copies, so matching by sentinel identity and extracting by type both stay
  transitive across the facade for a caller that never imports the internal packages — the same
  argument the existing sentinel and typed-error aliases in this file already make.
  Each new alias carries a doc comment giving that reason rather than repeating the aliased type's
  own description.
  Correct this file's own header comment in the same commit: it describes the file as declaring the
  aliases that make the engine's answer shape and error identity nameable from outside the module,
  and this card adds aliases for a second internal package's error identity, so the header must name
  both sources rather than the engine alone.
- **Commit:** `feat(quarry): alias the delta answer types and the git error identity`

### Card 32: declare the wrapped git answer and the pure Delta method

- **Context:**
  - `internal/engine/delta_answer.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `internal/engine/delta.go`
  - `internal/engine/repo.go`
- **Edits:**
  - `quarry/repo.go`
- **Creates:**
  - `quarry/delta.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Give the facade a seam to its own repository root, which it does not have today:
  the facade's repository type holds only the engine handle, and the engine's own root field is
  unexported with no accessor, so nothing in package `quarry` can name the root the git layer must be
  opened against.
  Add a root field to the facade's repository type and capture the constructor's already-validated
  root argument into it, in the same constructor, so both delta methods and any later git caller read
  one value.
  Correct that type's own doc comment in the same commit: it states that the type holds only the
  engine handle, which this card falsifies, and it draws its concurrency-safety claim from that
  premise — restate the claim over the two fields it will actually hold, both of which are read-only
  after construction.
  Capture it on the facade rather than exporting an accessor from the engine: the root is a value the
  facade constructor is already handed, and exporting an engine accessor would widen the engine's
  surface for a caller that never needed to ask it anything.
  Then create the facade's delta file with a file comment stating that it holds the two delta methods
  and the one new facade-declared type, and why that type is not an engine alias.
  Declare `GitDeltaAnswer`, embedding the aliased delta answer and adding a `from` field carrying the
  revision string exactly as given and a `to` field carrying the revision string as given or a JSON
  null for the working tree.
  Declare the two revision fields *before* the embedded answer so the emitted key order puts them
  first, and make the to field a pointer so an absent revision marshals as an explicit null rather
  than being omitted — the key must be present, since its null is the statement that the after side
  is the working tree.
  Its doc comment must state why the core's own answer carries no revision information at all: the
  core knows nothing about git, and a field it can never populate would be a lie in its own type.
  Declare `(*Repo).Delta(entries []DeltaEntry) (DeltaAnswer, error)`, delegating to the engine's own
  method unchanged with no filtering, no re-shaping and no defaulting, exactly as the existing three
  query methods on this facade do.
- **Commit:** `feat(quarry): capture the repository root and declare GitDeltaAnswer and Delta`

### Card 33: DeltaGit's revision handling and batch assembly

- **Context:**
  - `internal/gitsrc/gitsrc.go`
  - `internal/gitsrc/errors.go`
  - `internal/engine/delta_answer.go`
  - `internal/engine/delta.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
- **Edits:**
  - `quarry/delta.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare `(*Repo).DeltaGit(from, to, target string) (GitDeltaAnswer, error)`.
  The target is already a repository-relative path when it arrives; this method performs no path
  arithmetic on it and hands it to the git layer as the pathspec.
  Open the git layer against the root field card 32 adds to the facade's repository type, which is
  what performs the top-level verification, then verify each supplied revision before any diff runs,
  propagating the git layer's errors unchanged so the facade's aliased identity survives.
  Assemble the batch: the changed paths between the two sides, plus — when and only when the after
  side is the working tree — the untracked paths, each of which enters the batch with nil before
  bytes and its on-disk bytes after and therefore reads as added, exactly like a staged new file.
  Map each status letter to an entry by the total table the discussion fixes: added gives nil before
  bytes and the after bytes; modified gives both sides' bytes; deleted gives the before bytes and a
  nil after; a typechange gives both sides' bytes, where a side that is now a symlink yields its
  link text, which is not parseable source and therefore contributes no symbols on that side exactly
  as any other unparseable content does; an unmerged path is refused before extraction with the
  refusal message the discussion fixes; and any other letter is refused with a message naming the
  letter verbatim.
  The rename and copy letters cannot appear, because rename and copy detection are disabled, but the
  catch-all row covers them anyway rather than relying on that argument.
  A read failure on either side — a blob read failing, or a working-tree file being unreadable
  during assembly — is also a pre-set refusal naming the side and the underlying error, and never a
  failure of the whole call: a disk-read failure during assembly is neither a failed git command nor
  grounds to fail a batch, so it is one entry's problem reported as that entry's refusal.
  Read the after side from disk rather than through git when the after side is the working tree.
  Return the wrapped answer with the two revision strings echoed exactly as the caller gave them.
- **Commit:** `feat(quarry): add DeltaGit revision verification and batch assembly`

### Card 34: DeltaGit's per-directory clause vote and unit derivation

- **Context:**
  - `internal/engine/units.go`
  - `internal/engine/delta_answer.go`
  - `internal/gitsrc/gitsrc.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
- **Edits:**
  - `quarry/delta.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Fill each entry's two units.
  For every directory that has at least one changed Go file, and on both sides independently, derive
  the directory's dominant package clause and then each changed file's unit, using the engine's
  three exported helpers and never a reimplementation of the vote.
  For a revision side, enumerate the directory's immediate Go children at that revision through the
  git layer, read each one's bytes through the git layer, and turn those bytes into clauses with the
  engine's bytes-to-clause helper.
  For the working-tree side, enumerate by the git layer's working-tree listing and hand that file
  list to the engine's on-disk clause-map method, which reads from disk and skips any name it cannot
  read — an unstaged deletion is exactly such a name and must never fail the call.
  Feed the resulting clause map to the engine's unit helper and take each changed file's unit from
  the function it returns.
  Both sides must vote over the same set — the directory's immediate Go children, never a
  subdirectory's files — which is the set the git layer's two enumeration methods already promise.
  Skip the vote entirely for a directory in which no Go file changed; that is one of the two stated
  mitigations for this decision's cost and it is not a cache.
  Price the cost honestly in the doc comment rather than understating it: this is not one invocation
  per changed directory, but one enumeration plus one blob read and one parse per Go file of every
  changed directory, on both sides, so a one-file change in a twenty-file package costs on the order
  of forty blob reads and forty parses before any delta work begins.
  The primary consumer calls this once per plan card, so the price belongs where a reader meets it.
- **Commit:** `feat(quarry): derive per-directory units for both sides of a git delta`

### Card 35: amend the two doc comments that claim the facade adds no behaviour

- **Context:**
  - `quarry/delta.go`
  - `quarry/quarry.go`
- **Edits:**
  - `quarry/doc.go`
  - `quarry/repo.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** The claim that this facade adds no behaviour of its own appears twice — once in
  the package doc and once in the doc comment of the constructor — and both are false once the git
  convenience method lands, so amend both rather than one.
  Each amendment names the git method as the single exception and states why it is one: the git
  layer is a caller-facing convenience rather than query behaviour, and putting it only in the
  command line would force the primary Go consumer to reimplement the one thing that layer exists to
  hold.
  Update the package doc's two counts in the same edit, since both become wrong here: it states that
  the package exposes three query methods, which is now four, and that it owns seven renderers,
  which becomes nine once batch 7 lands — write the count that will be true at the end of this task
  and note in the comment that the two new renderers arrive with the delta verb, so the number is
  not briefly wrong for one batch.
  Leave every other claim in both comments untouched, including the phase-one non-goals paragraph,
  which this task does not change: the delta path holds no cache, no index and no state beyond the
  repository root.
- **Commit:** `docs(quarry): name DeltaGit as the exception to the adds-no-behaviour claim`

### Card 36: facade delta tests

- **Context:**
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `internal/engine/delta_answer.go`
  - `internal/gitsrc/gitsrc.go`
  - `internal/gitsrc/errors.go`
  - `quarry/repo_test.go`
- **Edits:** none
- **Creates:**
  - `quarry/delta_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestDeltaGit_AgreesWithDelta`, `TestDeltaGit_StatusLetterMapping`,
  `TestDeltaGit_WorkingTreeSide` and `TestDeltaGit_ErrorIdentity` in package `quarry`, each building
  its own throwaway git repository under a temporary directory and skipping cleanly when no git
  binary is available.
  The agreement test is the load-bearing one: it runs the git method on a fixture and, separately,
  assembles the same entries by hand and runs the pure method on them, asserting the two answers are
  equal apart from the revision echo — the two paths must not be able to disagree, since one is
  documented as a convenience over the other.
  The status-letter test asserts the disposition each letter produces, including that an unmerged
  path yields an error disposition with no extraction attempted, since a conflicted file's content
  is conflict markers and extracting it as source would be a silent lie, and that an unrecognised
  letter yields an error naming the letter.
  The working-tree test asserts that an untracked, never-staged source file is enumerated, reaches
  the delta as added and has its symbols in the created array; that an untracked file matched by a
  gitignore pattern does not; that a tracked-but-gitignored file *is* kept, asserted deliberately so
  the documented divergence from the table-of-contents listing rule cannot be silently reverted; and
  that a source file deleted from the working tree but still in the index does not fail the call.
  The error-identity test asserts that an unresolvable revision, a root that is a subdirectory of a
  repository, and a root outside any repository each surface through the facade's own aliased
  sentinels, matched by identity, and that the two typed errors yield their fields when extracted by
  type.
  Assert as well that both sides of a directory's clause vote enumerate the same file set, by using
  a fixture containing a tracked-but-gitignored source file and asserting the directory produces no
  spurious create-plus-delete storm; a divergence there changes every glyph in the directory, so it
  is asserted directly rather than left to follow from the enumeration code.
- **Commit:** `test(quarry): assert DeltaGit against Delta, the status letters and the git error identity`

### Card 37: make the git-layer layering rule mechanical

- **Context:**
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/cli/cli.go`
  - `internal/gitsrc/gitsrc.go`
- **Edits:**
  - `internal/mcpserver/layering_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** The overview's Shared Decision on reaching git error identity through the facade
  is currently guarded by nothing but this plan's own prose, and the repository has exactly one
  mechanical layering gate: the import check in `internal/mcpserver/layering_test.go`, which today
  forbids one import path across the mcp server package's files and its command's files.
  Extend that check rather than adding a second mechanism.
  Make its forbidden set two paths rather than one, adding the git plumbing package, so the mcp
  server can no more reach the git layer directly than it can the engine.
  Then extend the scanned set with the command-line package's own files, checked against the git
  plumbing path alone — that package legitimately imports neither, but it is the package the Shared
  Decision is actually about, and a convention no test states is one refactor from being lost.
  Do not add the engine path to the command-line package's forbidden set: that would be a new rule
  this task did not decide, and the existing header comment already records that the facade-only rule
  holds there by convention.
  The check reads each file's own import block, so it catches a direct import only; a package
  reaching the git layer transitively through the facade is exactly what the design intends and must
  not fail.
  Update that file's own header comment, which states which packages it covers and which single rule
  it enforces, and extend its existing note about the residual gap to say what the widened check does
  and does not catch.
  Confirm as well, by inspection, that no file added in this batch writes to standard output, which
  the same test's second check already enforces for its own packages.
- **Commit:** `test(mcpserver): forbid direct git-layer imports and cover the CLI package`

## Batch Tests

`verify:` runs two commands.
The first, `go test ./quarry/ -run 'TestDelta'`, selects the four test functions card 36 adds in
`quarry/delta_test.go`; no existing test in this package carries that prefix.
That pattern also matches the golden and real-history tests batch 9 adds later, which is intentional
and harmless — by then those tests exist and should pass.
The second, `go test ./internal/mcpserver/ -run 'TestFacadeOnly|TestStdout'`, runs the two checks in
the layering test card 37 edits, in the package that owns it: an import-rule test is only a gate if
the batch that widens it also runs it.
Every case in card 36 builds a fresh throwaway repository and skips when git is unavailable, so this
batch needs no checkout of any other repository.
Card 35's doc-comment amendments have no runnable surface; the module-wide `go vet ./...` at this
batch's boundary is what proves the package still compiles with them.
