# Batch: gitsrc

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'gitsrc'
number: 5
cards: 6
verify: go test ./internal/gitsrc/
depends-on: []
```

## Batch Scope

This batch creates `internal/gitsrc`, a new package holding read-only git plumbing and nothing else:
verify a root is a repository top-level, verify a revision, list changed paths with their status
letters, list untracked paths, read a blob at a revision, and list a directory's immediate children
on either side.
It returns paths, bytes and errors only — no quarry type, no tree-sitter node and no package clause,
because a clause can only come from a strategy inside the engine's parse seam, which is the engine's
job and never this package's.

It is a root batch: the package does not exist yet, so it shares no file with batch 1 or batch 2.
It is a separate package rather than functions in the engine because the engine's own doc contract
is that it reads the source as it is at that moment, with no process spawning anywhere in it.
This is the first production package in the repository to shell out; every other use of process
execution today is in test files and the benchmark harness.

The external interface batch 6 consumes: an opened handle that has already proved its root is the
repository top-level, the six read operations above, and the error identity described in the
overview's Shared Decision on reaching git error identity through the facade.

Batch-local decision: the package takes an explicit repository root and passes it to every git
invocation rather than relying on the process working directory, matching how the existing tests in
this repository already invoke git.

## Cards

### Card 25: package doc, error identity, and the git invocation helper

- **Context:**
  - `internal/engine/errors.go`
  - `internal/engine/repo.go`
  - `internal/engine/doc.go`
- **Edits:** none
- **Creates:**
  - `internal/gitsrc/doc.go`
  - `internal/gitsrc/errors.go`
  - `internal/gitsrc/gitsrc.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the package doc stating: this package is read-only git plumbing; it runs
  no command that writes, so no checkout, no stash, no index write, no config write; it returns
  paths, bytes and errors only and knows nothing about symbols, clauses or units; and it never uses
  git's own rename or copy detection, which is a similarity threshold and therefore the one thing
  the delta contract forbids.
  Declare three sentinels — `ErrNotARepository`, `ErrRootNotTopLevel`, `ErrUnknownRevision` — and
  two typed errors that wrap them so both matching styles work: `RootNotTopLevelError` with fields
  for the root as given and the top-level git reported, and `UnknownRevisionError` with a field for
  the revision spelled exactly as the caller gave it.
  Each typed error's `Error` method spells a full sentence, and each implements `Unwrap` returning
  its matching sentinel, so a caller matching the sentinel by identity and a caller extracting the
  fields by type both succeed against the same value.
  Their doc comments must say that the fields exist so the command-line layer can spell its own
  user-facing sentence from the value rather than by parsing a message.
  Declare an unexported helper that runs one git invocation against an explicit repository root,
  captures standard output, and returns a wrapped error naming the subcommand on failure.
  Every operation in this package goes through that one helper, so the root-passing rule and the
  failure-wrapping rule each have one implementation.
  The helper must not prefix its errors with anything reading like an internal package path, since
  the command-line layer carries a failed git command's message whole behind its own internal-error
  prefix.
- **Commit:** `feat(gitsrc): add the read-only git plumbing package skeleton and error identity`

### Card 26: open a repository and verify a revision

- **Context:**
  - `internal/gitsrc/doc.go`
  - `internal/gitsrc/errors.go`
  - `internal/repopath/root.go`
- **Edits:**
  - `internal/gitsrc/gitsrc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare an opened-repository type holding the absolute root, and a constructor
  taking that root.
  The constructor asks git for the top-level of the given root, first of all, before any other
  operation.
  A root that is not inside a repository at all returns the not-a-repository sentinel.
  A root inside a repository whose top-level is elsewhere returns the top-level typed error carrying
  both paths.
  The comparison puts **both** sides through symlink evaluation and then cleaning before comparing
  them: the repository-root resolver this project already uses performs a join and a clean and never
  resolves symlinks, while git prints the physical path, so a raw string comparison would reject a
  perfectly valid repository reached through a symlink.
  A temporary directory is exactly such a path on any platform whose temp directory is symlinked,
  so without this the check would reject this project's own test fixtures; record that reason in the
  doc comment, because it is the kind of rule a later reader deletes as redundant.
  Declare a revision verification method running git's own verify form against the revision resolved
  to a commit, returning the unknown-revision typed error carrying the revision exactly as given.
  Its doc comment must state that the caller runs it for every supplied revision before any diff, so
  an unresolvable revision is reported as such rather than surfacing later as a failed diff.
  The requirement that the root be the repository top-level exists because git emits paths relative
  to the top-level while quarry consumes them as root-relative; without it every path in the answer
  would be silently wrong and a pathspec would select the wrong subtree.
- **Commit:** `feat(gitsrc): verify the repository top-level and a supplied revision`

### Card 27: list changed paths and untracked paths

- **Context:**
  - `internal/gitsrc/doc.go`
  - `internal/gitsrc/errors.go`
- **Edits:**
  - `internal/gitsrc/gitsrc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare a change type carrying a repository-relative path and its raw git status
  letter, and a method returning the changed paths between two revisions under a pathspec, plus a
  method returning untracked paths under a pathspec.
  The changed-paths method takes the after revision as an empty string to mean the working tree,
  which selects the one-revision form of the diff.
  Both forms pass the flag that disables rename and copy detection.
  That flag is a correctness requirement rather than an optimisation: git's detection is a
  similarity threshold, and letting it run would mean quarry's answer silently inherited the
  heuristic the whole two-tier design exists to replace.
  With it, a rename arrives as a delete plus an add and is classified by the table comparison, which
  is the only classifier this query is allowed to have.
  The untracked method passes the standard-exclusion flag, so an untracked file matched by a
  gitignore pattern is never picked up while build output and ignored artefacts stay out of the
  batch.
  Its doc comment must state that the caller invokes it only when the after side is the working
  tree, and why: a card that creates a file and has not yet staged it is the normal state at the
  moment the primary consumer asks the question, and without this that file's symbols would be
  silently absent with the files echo unable to record the omission.
  Every path-emitting invocation in this package uses git's null-delimited output form and splits on
  the null byte.
  Without it git applies its path-quoting configuration and quotes any non-ASCII or control-character
  path while delimiting with newlines, so such a path would be read at the wrong location — silently,
  since a mangled path simply fails to open — and a newline inside a filename would split one path
  into two.
  This method returns the raw status letter and maps nothing: the mapping from letter to disposition
  belongs to the layer that builds entries, since this package returns paths, bytes and errors only.
- **Commit:** `feat(gitsrc): list changed paths with status letters and untracked paths`

### Card 28: read a blob and enumerate a directory's immediate children on both sides

- **Context:**
  - `internal/gitsrc/doc.go`
  - `internal/gitsrc/errors.go`
- **Edits:**
  - `internal/gitsrc/gitsrc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Declare a method reading one path's bytes at a revision, and two methods
  enumerating a directory's files — one for a revision side and one for the working-tree side.
  Both enumeration methods return the same *set* by contract: the directory's immediate children
  only, never a subdirectory's files.
  That is the set the existing clause vote runs over, since the walk reads one directory at a time
  and recurses separately, and the unit is a per-directory fact, so a subdirectory's clause must
  never enter this directory's vote.
  The two underlying git commands are unequal and each must be trimmed to that set: the revision-side
  listing is non-recursive already, so it yields the immediate entries directly; the working-tree
  listing is inherently recursive and has no non-recursive mode, so its output must be filtered to
  paths carrying no further separator after the directory prefix.
  Both then keep only Go source files.
  Without that trim the working-tree side sweeps in subdirectory files the revision side never sees,
  the two sides' dominant clauses can disagree, and a disagreement there changes the unit of every
  glyph in the directory, turning the whole directory into a create-plus-delete storm — the exact
  failure this rule exists to prevent.
  The working-tree enumeration lists index entries as well as untracked ones, so it can name a file
  that is not present on disk; that is expected, and reading such a file is the caller's problem
  rather than this method's.
  Both enumerations are tracked-inclusive and neither applies any ignore filtering of its own, so a
  file that is both tracked and matched by a gitignore pattern is listed on both sides or on
  neither; state that in the doc comments, because the symmetry is what the comparison depends on.
- **Commit:** `feat(gitsrc): read blobs and enumerate a directory's immediate children per side`

### Card 29: the throwaway-repository fixture builder

- **Context:**
  - `internal/gitsrc/gitsrc.go`
  - `internal/gitsrc/errors.go`
  - `internal/engine/loomyard_test.go`
- **Edits:** none
- **Creates:**
  - `internal/gitsrc/fixture_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a test-only fixture builder that creates a small git repository under a
  temporary directory and drives it with process invocations, as the existing test helpers in this
  repository already do.
  It must support, in one shape a table test can drive: initialising a repository with a fixed
  identity and a fixed default branch name so no machine's global git configuration can change the
  fixture's behaviour; writing a file with given content; staging and committing with a given
  message and returning the resulting commit's identifier; deleting a file from the working tree
  with and without staging the deletion; renaming a file; writing a gitignore file; and leaving a
  file untracked.
  It must also support producing an unmerged path, since a conflicted file is reachable on the
  working-tree side during a merge and is one of the status letters card 27 returns.
  The builder skips the whole test, with the reason, when no git binary is available, following the
  skip-versus-fail asymmetry the existing helpers in this repository already establish: a machine
  without the tool is a normal state, so it skips.
  Every fixture is built fresh per test; no test reuses another's repository.
- **Commit:** `test(gitsrc): add the throwaway git repository fixture builder`

### Card 30: gitsrc behaviour tests

- **Context:**
  - `internal/gitsrc/gitsrc.go`
  - `internal/gitsrc/errors.go`
  - `internal/gitsrc/doc.go`
  - `internal/gitsrc/fixture_test.go`
- **Edits:** none
- **Creates:**
  - `internal/gitsrc/gitsrc_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add table tests over the fixture builder covering every behaviour cards 26 to 28
  promise.
  Opening: a root that is the repository top-level is accepted; a root pointing at a subdirectory of
  a repository returns the top-level typed error, before any diff runs; a root outside any
  repository returns the not-a-repository sentinel, likewise before any diff; and a repository
  reached through a symlinked path is accepted rather than rejected, which is the case that fails
  without evaluating symlinks on both sides.
  Revisions: an unresolvable revision returns the unknown-revision error, matched both by sentinel
  identity and by extracting the revision field, and is reported before any diff is attempted.
  Changed paths: the path list and status letters for added, modified and deleted files; a rename
  in the fixture arriving as a delete plus an add, which is what proves rename detection is
  disabled; an unmerged path appearing with the unmerged letter; and the working-tree-as-after-side
  form listing a change that was never committed.
  Untracked: a never-staged Go file is enumerated; one matched by a gitignore pattern is not.
  Enumeration: the revision side and the working-tree side return the same set for a directory
  containing a tracked-but-gitignored Go file, so that file votes on both sides or neither; a Go file
  in a *subdirectory* declaring a different package appears on neither side, which is the
  immediate-children trim doing its work; and a file removed from the working tree but still in the
  index is listed by the working-tree enumeration, which is the case the clause-map builder must
  tolerate.
  Paths: a fixture path containing a non-ASCII character round-trips through every path-emitting
  operation, which it cannot do without null-delimited output.
  Failures: a git invocation that fails for an ordinary reason surfaces as an error rather than a
  panic.
- **Commit:** `test(gitsrc): assert top-level, revision, status, enumeration and path handling`

## Batch Tests

`verify:` runs `go test ./internal/gitsrc/` — the whole package rather than a `-run` selection,
because every file in the package is new in this batch, so package scope and batch scope are the
same set and there is nothing broader for the pattern to guard against.
Every test builds its own throwaway repository under a temporary directory and skips cleanly on a
machine with no git binary, so the batch is runnable in an environment that has no checkout of any
other repository.
The status-letter mapping is deliberately *not* asserted here: this package returns the raw letter
and maps nothing, so the mapping's own test belongs to batch 6, which owns the mapping.
