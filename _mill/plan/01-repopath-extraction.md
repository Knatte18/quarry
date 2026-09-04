# Batch: repopath-extraction

```yaml
task: "MCP, thin (T6)"
batch: "repopath-extraction"
number: 1
cards: 4
verify: go test ./internal/repopath/... ./internal/cli/...
depends-on: []
```

## Rename mechanic

For each `Moves:` pair the implementer MUST:

1. Run `git mv <old> <new>` FIRST, before making any other change to the moved file.
2. Make ONLY surgical edits — touch only the lines that must change after the move (package
   declaration, imports, identifier retargeting, seam splits).
3. Use a full-file `Creates:` entry only for genuinely new files that have no predecessor.
4. Never write the relocated file from scratch and delete the original — that breaks git rename
   history and inflates review diffs.

## Batch Scope

This batch lifts the repository-root discovery and target-relativisation logic out of
`internal/cli` into a new leaf package `internal/repopath`, exports it, and replaces its unexported
`usageError` return with two exported sentinels. `internal/cli` then calls the new package and
re-formats its own two user-facing sentences from values it already holds, so the CLI's observable
behaviour — messages, exit codes, golden bytes — is unchanged. The moved tests re-land in
`internal/repopath` with a per-package fixture helper, and a new exact-string test in `internal/cli`
pins the two sentences that nothing pins today.

It is one batch because the move, the caller update, and the test relocation cannot land separately
without leaving the tree uncompilable, and it is the first batch because batch 2's MCP server is the
second consumer of exactly these two behaviours.

**External interface batch 2 consumes:** `repopath.ResolveRoot(flagRoot, cwd string) (string, error)`,
`repopath.DiscoverRoot(startDir string) (string, error)`,
`repopath.RepoRelTarget(root, base, target string) (string, error)`, and the sentinels
`repopath.ErrNoRepositoryRoot` and `repopath.ErrRootNotDirectory`.

Batch-local decision, differing from nothing in the overview: `internal/cli/scratchtree_test.go`
keeps its helper. `internal/cli/cli_test.go` still uses `writeScratchTree`, so the CLI's own copy
stays exactly where it is and only its header sentence naming its callers is corrected;
`internal/repopath` takes a copy, it does not take the original.

## Cards

### Card 1: extract `internal/repopath` from `internal/cli`

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/doc.go`
  - `quarry/quarry.go`
- **Edits:** none
- **Creates:**
  - `internal/repopath/doc.go`
- **Deletes:** none
- **Moves:**
  - `internal/cli/root.go` -> `internal/repopath/root.go`
  - `internal/cli/target.go` -> `internal/repopath/target.go`
- **Requirements:**
  Change the package clause of both moved files from `package cli` to `package repopath`.
  Export the three functions, keeping their signatures and bodies otherwise unchanged:
  `discoverRoot` becomes `DiscoverRoot(startDir string) (string, error)`, `resolveRoot` becomes
  `ResolveRoot(flagRoot, cwd string) (string, error)`, `repoRelTarget` becomes
  `RepoRelTarget(root, base, target string) (string, error)`. `ResolveRoot`'s internal call to
  `discoverRoot` retargets to `DiscoverRoot`.
  Declare two exported sentinels in the moved root file, since `usageError` does not exist in this
  package: `ErrNoRepositoryRoot` and `ErrRootNotDirectory`, both `errors.New` values whose own text
  is namespaced to this package and is never user-visible through the CLI. Return them wrapped with
  `fmt.Errorf` and `%w` so the wrap carries the offending path: `DiscoverRoot`'s failure branch
  wraps `ErrNoRepositoryRoot` and names `startDir`; `ResolveRoot`'s not-a-directory branch wraps
  `ErrRootNotDirectory` and names `flagRoot` as given, not the absolutised form.
  `RepoRelTarget` keeps returning `quarry.ErrTargetOutsideRepo` for both escape branches, unchanged
  and unwrapped — the CLI already matches on it with `errors.Is` and must keep succeeding.
  Adjust the moved doc comments: they currently name `usageError` and "the CLI" as the caller, and
  cross-reference batch numbers from a previous task's plan. Restate them for a package with two
  callers — the CLI and the MCP server — and state that both callers format their own user-facing
  sentences from the sentinels rather than propagating this package's error strings.
  Write a package doc comment in the new `doc.go`, following the shape of the existing
  `internal/cli/doc.go`.
  This package must not import `internal/engine` or any package below it; `quarry` is its only
  intra-module import.
- **Commit:** `refactor(repopath): extract root discovery and target relativisation from internal/cli`

### Card 2: point `internal/cli` at `repopath` and keep its message bytes

- **Context:**
  - `internal/cli/flags.go`
  - `internal/repopath/root.go`
  - `internal/repopath/target.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Correct the one sentence in the package doc comment that lists repository-root discovery among what
  this package does. After this card that work lives in `repopath`; the CLI resolves a root by calling
  it. Change that clause and nothing else in the file.
  Replace `Run`'s calls to the now-moved unexported helpers with `repopath.ResolveRoot(req.root, cwd)`
  and `repopath.RepoRelTarget(root, base, req.target)`, importing
  `github.com/Knatte18/quarry/internal/repopath`.
  Add a named unexported function `rootUsageMessage(err error, flagRoot, cwd string) (string, bool)`
  to this file. It returns `("no repository root found above " + cwd + "; pass --root", true)` when
  `errors.Is(err, repopath.ErrNoRepositoryRoot)`, `("--root is not a directory: " + flagRoot, true)`
  when `errors.Is(err, repopath.ErrRootNotDirectory)`, and `("", false)` otherwise. It is a named
  function rather than an inline switch precisely so card 4's table test can assert both sentences
  directly, without a fixture that cannot exist (the no-root case is unreachable from inside this
  repository without changing the process working directory, which these tests never do).
  `Run` calls it on `ResolveRoot`'s error: on `true`, fail with `exitUsage` and `withUsage` true,
  exactly as the current `usageError` branch does; on `false`, fail with `exitInternal` and the
  `internal error: ` prefix, exactly as the current fallthrough does.
  Do not propagate `err.Error()` from either sentinel into a user-facing message. The sentinel text
  is namespaced to `repopath` and would leak an internal package name into the CLI's contract, and
  the first sentence cannot be reproduced by wrapping at all — the sentinel text would land
  mid-sentence.
  Everything else in the pipeline is unchanged: the same `base` selection, the same `Lstat`, the
  same `codeForTOCError` mapping, the same exit codes, the same rendering.
  Update `Run`'s doc-comment step list only where it names the moved helpers; the step ordering and
  every stated exit-code disposition stay as they are.
  Two stale references inside the function body need correcting too, and this card owns both. The
  inline comment on the root-resolution failure branch asserts that the two moved helpers are
  contracted to return only a `usageError` — false after this card, since they now return wrapped
  sentinels — and it cites a card number belonging to a previous task's plan, which means nothing
  here. Restate what that branch is actually guarding after the change and drop the stale card
  citation. Do not delete the branch itself: its point is that every step of this pipeline spells both
  dispositions, and that point survives the refactor.
- **Commit:** `refactor(cli): call internal/repopath and format the root-resolution messages locally`

### Card 3: re-land the moved tests in `internal/repopath`

- **Context:**
  - `internal/repopath/root.go`
  - `internal/repopath/target.go`
  - `internal/cli/cli_test.go`
- **Edits:**
  - `internal/cli/scratchtree_test.go`
- **Creates:**
  - `internal/repopath/scratchtree_test.go`
- **Deletes:** none
- **Moves:**
  - `internal/cli/root_test.go` -> `internal/repopath/root_test.go`
  - `internal/cli/target_test.go` -> `internal/repopath/target_test.go`
- **Requirements:**
  Change the package clause of both moved test files from `package cli` to `package repopath`, and
  retarget every call to the exported names from card 1.
  Replace the two `err.(usageError)` type assertions with `errors.Is` against the new sentinels —
  `ErrNoRepositoryRoot` in the discovery no-root case, `ErrRootNotDirectory` in the two
  `ResolveRoot` failure cases — and drop the now-unused `usageError`-shaped local variables. The
  existing substring assertions on `"pass --root"` and on the echoed `--root` path move to the
  wrapped error's own `Error()` string.
  Keep the `mustGetwd` helper with the moved discovery test; it is used by no other file in
  `internal/cli`.
  Write the new `scratchtree_test.go` as a per-package copy of `internal/cli/scratchtree_test.go`,
  with two changes: the scratch subdirectory becomes `.scratch/repopath-tests/` instead of
  `.scratch/cli-tests/`, so this package does not share a parent directory with a package that tests
  in parallel; and the header comment states it is a deliberate per-package copy for this package,
  naming the same reason the CLI's own copy names. `internal/repopath/` sits two directories below
  the module root, the same depth as `internal/cli/`, so the `runtime.Caller(0)` walk keeps its three
  `filepath.Dir` steps unchanged.
  In `internal/cli/scratchtree_test.go`, correct the one header sentence naming the two test files
  this card removes from that package as the helper's chief users; after this card its user is the
  pipeline test. Change that sentence and nothing else — the helper's body, its scratch subdirectory
  and its doc comment stay exactly as they are, because `internal/cli/cli_test.go` still uses it.
  Every other test file in `internal/cli` must pass unchanged — that is this refactor's regression
  gate.
- **Commit:** `test(repopath): move the root and target tests with their functions`

### Card 4: pin the CLI's two root-resolution sentences as exact strings

- **Context:**
  - `internal/cli/cli.go`
  - `internal/repopath/root.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/message_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a table test over `rootUsageMessage` asserting both sentences as exact strings, not
  substrings: for an error wrapping `repopath.ErrNoRepositoryRoot` it must return
  `no repository root found above /some/cwd; pass --root` for `cwd = "/some/cwd"`, and for an error
  wrapping `repopath.ErrRootNotDirectory` it must return `--root is not a directory: ../given` for
  `flagRoot = "../given"` — the path as given, not an absolutised form. Add a third case asserting
  that an unrelated error returns `("", false)`.
  Build the input errors by wrapping the sentinels with `fmt.Errorf` and `%w`, mirroring how
  `repopath` returns them, so the test also proves the `errors.Is` match survives the wrap.
  Explain in the file header why this test exists: the two sentences were asserted only by substring
  before this task and carry no golden, so the refactor could have drifted them silently. This test
  is what makes the "same messages" claim checkable.
- **Commit:** `test(cli): assert the two root-resolution messages as exact strings`

## Batch Tests

`verify: go test ./internal/repopath/... ./internal/cli/...` runs both the relocated package's own
suite and the whole of `internal/cli`'s existing suite. Scoping to these two packages is correct
here: nothing outside them imports the moved functions, and `internal/cli`'s untouched test files —
`cli_test.go`, `flags_test.go`, `after_test.go`, `plan4_test.go`, `loomyard_test.go` — are the
regression gate proving the CLI's messages, exit codes and golden bytes did not move. The overview's
module-wide `go vet ./...` catches any cross-package compile fallout from the move at this batch's
boundary.
