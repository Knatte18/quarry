# Batch: cli-parsing

```yaml
task: "Facade + CLI, toc (T5a)"
batch: "cli-parsing"
number: 3
cards: 7
verify: go test ./internal/cli/...
depends-on: [1]
```

## Batch Scope

This batch creates `internal/cli` and its four pure layers: the usage text, the hand-rolled flag
parser, repository-root resolution, and target conversion. None of them calls the engine or the
facade's `TOC`; every one is a function over strings and, at most, a `.scratch/cli-tests/` fixture
tree, which is what makes this the batch that carries most of the task's test weight. Batch 4 adds `Run`, which
composes these four in the fixed order `target-kind-and-the-cli-stat` sets out.

It depends on batch 1 only — `internal/cli/target.go` imports `quarry` for the
`ErrTargetOutsideRepo` sentinel — and not on batch 2, so it can be built while the renderers are
still in flight.

The external interface batch 4 consumes: the `request` struct, `parseArgs`, `usageError`,
`usageText`, `resolveRoot`, `discoverRoot`, and `repoRelTarget`.

Batch-local decision: the parser is hand-rolled over a `[]string` rather than built on the standard
library's `flag` package, per the overview's `no-new-module-dependencies` — `flag` cannot express
`--depth all` alongside `--depth 3`, nor `--no-symbols` as a third state distinct from an absent
`--symbols`.

## Cards

### Card 9: package doc and usage text

- **Context:**
  - `internal/engine/doc.go`
  - `docs/rewrite-plan.md`
  - `_mill/discussion.md`
- **Edits:** none
- **Creates:**
  - `internal/cli/doc.go`
  - `internal/cli/usage.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/doc.go` declaring `package cli` with a package doc comment
  only. It states that `cli` is the whole of the `quarry` command below `os.Exit`: flag parsing,
  repository-root discovery, cwd-relative target interpretation, the exit-code mapping, and the
  choice of renderer. It states that `cmd/quarry` holds one line and nothing else, and that this
  split exists so the golden tests can capture exactly the bytes the binary emits without building
  or exec'ing anything. It states that this package is the only layer with a working directory —
  `internal/engine` deliberately performs no git discovery and no cwd resolution — and that the two
  path frames never mix: input is interpreted where the user is, output is always repository-root
  relative with forward slashes.

  Create `internal/cli/usage.go` declaring `const usageText = ...` — a raw string literal ending in
  a single newline, with no leading blank line and no trailing whitespace on any line — holding
  exactly:

```
quarry - a table of contents for a source repository

usage:
  quarry toc <target> [flags]

flags:
  --depth <N|all>   how far to recurse into subdirectories (default 0)
  --symbols         populate every file entry's symbols
  --no-symbols      leave every file entry's symbols unpopulated
  --text            emit the lossless text view instead of JSON
  --root <path>     use <path> as the repository root instead of discovering one
  -h, --help        print this text and exit 0

exit codes:
  0  answered
  1  negative answer: target not found, or target outside the repository
  2  usage error
  3  internal error
```

  Use ASCII only — no em dash, no typographic quotes — so the text is stable across terminals and
  byte-comparable in tests. Do not add a `--json` flag or mention one: JSON is the default, and
  naming it would imply a third format exists.
- **Commit:** `feat(cli): add the cli package doc and usage text`

### Card 10: the flag parser

- **Context:**
  - `internal/cli/usage.go`
  - `internal/cli/doc.go`
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/flags.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/flags.go` in `package cli`.

  Declare `type usageError string` with `func (e usageError) Error() string { return string(e) }`.
  Its doc comment records that a `usageError` is the one error class that maps to exit 2, that its
  string is the CLI's own single-line complaint with no prefix and no wrapped chain, and that it is
  a distinct type rather than a sentinel precisely because each occurrence carries its own message.

  Declare `type request struct` with the fields `verb string`, `target string`, `depth int`,
  `symbols *bool`, `text bool`, `root string`, `help bool`. `root` holds the `--root` value exactly
  as given, empty when the flag was absent. `symbols` is nil when neither `--symbols` nor
  `--no-symbols` was given, which is the engine's per-target default.

  Declare `func parseArgs(args []string) (request, error)`, where `args` is `os.Args[1:]`. Behaviour:
  - Scan for `--help` or `-h` at **any** position, before anything else, and before rejecting any
    unknown flag or missing verb. When found, return a `request` with `help: true` and a nil error.
  - With no arguments at all, return `usageError("no verb given; expected: toc")`.
  - The first argument is the verb. Anything other than `toc` returns
    `usageError(fmt.Sprintf("unknown verb: %s", verb))`. A first argument beginning with `-` is
    reported as a missing verb, not an unknown flag, with the same message as the empty case.
  - Parse the remaining arguments. A token beginning with `-` is a flag — **single dash as well as
    double**, so `-x` is rejected as an unknown flag rather than silently becoming the target. The
    only single-dash token the parser recognises is `-h`, and the help pre-scan above has already
    consumed it. Accept both the space-separated form (`--depth 3`) and the equals form
    (`--depth=3`), splitting on the first `=` with `strings.Cut`. A value-taking flag with no value
    following it returns `usageError(fmt.Sprintf("%s requires a value", flag))`. Any unrecognised
    flag, one dash or two, returns `usageError(fmt.Sprintf("unknown flag: %s", flag))` — the token
    as given, including its dashes.
    The bare token `-` is also an unknown flag: quarry reads no target from stdin, so the
    conventional stdin spelling has no meaning here and silently treating it as a filename would be
    worse than rejecting it. The bare token `--` is likewise an unknown flag rather than an
    end-of-flags separator; the flag set is closed and a target never begins with a dash.
  - `--depth` takes `all`, mapping to `quarry.DepthAll`, or a non-negative decimal integer parsed
    with `strconv.Atoi`. A negative integer, a non-integer, or an empty value returns
    `usageError(fmt.Sprintf("--depth must be a non-negative integer or \"all\", got %q", value))`.
    The default is 0.
  - `--symbols` sets `symbols` to a pointer to `true`; `--no-symbols` to a pointer to `false`. Both
    given in one invocation returns
    `usageError("--symbols and --no-symbols cannot both be given")`, whatever their order and even
    when one is repeated.
  - `--text` sets `text`. `--root` takes the following token verbatim into `root`.
  - Every remaining token after the verb — that is, every token not beginning with `-` — is a
    target. Zero targets returns
    `usageError("toc takes exactly one target, got 0")`; two or more returns the same sentence with
    the actual count. Exactly one sets `target`.
  Do not resolve any path, stat anything, or read the working directory here — this function is pure
  over its argument slice, which is what lets its table test run with no fixtures.
- **Commit:** `feat(cli): add the hand-rolled flag parser`

### Card 11: repository-root resolution

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/doc.go`
  - `internal/engine/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/root.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/root.go` in `package cli`.

  Declare `func discoverRoot(startDir string) (string, error)`: starting at `startDir`, test whether
  `filepath.Join(dir, ".git")` exists using `os.Lstat`, and return `dir` on the first hit. A `.git`
  **file** counts exactly as a `.git` directory does — that is the git-worktree case, and rejecting
  it would make quarry unusable inside every mill worktree. Step to `filepath.Dir(dir)` and stop
  when the parent equals the current directory, i.e. the filesystem root has been reached, returning
  `usageError(fmt.Sprintf("no repository root found above %s; pass --root", startDir))`.

  Declare `func resolveRoot(flagRoot, cwd string) (string, error)`: when `flagRoot` is empty,
  delegate to `discoverRoot(cwd)`. Otherwise make `flagRoot` absolute by joining it onto `cwd` when
  it is relative and cleaning it, then `os.Stat` it: a stat error or a non-directory returns
  `usageError(fmt.Sprintf("--root is not a directory: %s", flagRoot))`, naming the path **as given**
  rather than the resolved one, so the message echoes what the caller typed. On success return the
  cleaned absolute path. `--root` skips the walk entirely — it never falls back to discovery.

  Both functions take their starting directory as a parameter rather than calling `os.Getwd`
  themselves, so their tests need no process-global state; `Run` (batch 4) is the one place that
  reads the working directory. Say so in the doc comments.
- **Commit:** `feat(cli): add repository-root discovery and --root resolution`

### Card 12: target conversion

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/root.go`
  - `internal/engine/repo.go`
  - `quarry/quarry.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/target.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/target.go` in `package cli`, importing
  `github.com/Knatte18/quarry/quarry`.

  Declare `func repoRelTarget(root, base, target string) (string, error)`. `root` is the absolute
  repository root from `resolveRoot`; `base` is the directory a relative target is interpreted
  against; `target` is the argument as given. Behaviour:
  - Build the absolute target: `filepath.Clean(target)` when `filepath.IsAbs(target)`, otherwise
    `filepath.Join(base, target)`, which cleans as it joins.
  - `rel, err := filepath.Rel(root, abs)`. On error, or when the forward-slash form of `rel` is
    exactly `..` or begins with `../`, return `""` and `quarry.ErrTargetOutsideRepo` unwrapped, so
    the caller's `errors.Is` succeeds and the caller composes the message from the target as given.
  - Otherwise return `path.Clean(filepath.ToSlash(rel))`. `filepath.Rel` returns `"."` when the
    target is the root itself, which the engine accepts as the root.

  Use `filepath.Abs`-style joining and `filepath.Rel` only — never `filepath.EvalSymlinks`. Its doc
  comment records why: the engine's `resolveTarget` uses `os.Lstat` and never `os.Stat`, so a
  symlink named directly as the target is answered as a file rather than followed, and resolving
  symlinks here would defeat that rule before the engine ever saw the path.
  It also records that this function touches no filesystem at all — it is pure path arithmetic, so
  a target that escapes the root is rejected before any stat happens, which is step 3 of
  `target-kind-and-the-cli-stat`'s fixed order.
  Native separators are accepted on input; the returned path is always forward-slash and
  repository-root relative, which is the form the engine takes and emits.
- **Commit:** `feat(cli): add cwd-and-root-relative target conversion`

### Card 13: flag parser tests

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `quarry/quarry.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/flags_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/flags_test.go` in `package cli`, table-driven, with no
  filesystem access at all.
  Cover `--depth` accepting `0`, `3`, and `all` (the last yielding `quarry.DepthAll`), and rejecting
  `-1`, `x`, and the empty value; the equals form `--depth=all` parsing identically to the
  space form.
  Cover `--symbols` yielding a non-nil pointer to `true`, `--no-symbols` a non-nil pointer to
  `false`, neither yielding nil, and both together being an error in either order.
  Cover `--text` setting the flag and being absent by default.
  Cover `--root` capturing its value verbatim, including a relative one, and `--root` with no
  following token being an error.
  Cover the usage-error cases: unknown flag (asserting the message is exactly
  `unknown flag: --depht` for that input), missing target, two targets (asserting the count in the
  message), missing verb, unknown verb, and a first argument that is a flag.
  Cover the single-dash hole explicitly, as its own table rows: a post-verb `-x` is
  `usageError("unknown flag: -x")` and **not** a target, so it can never reach the exit-1 path as
  `target not found: -x`; the bare `-` and the bare `--` are likewise unknown flags; and `-h`
  remains the one single-dash token that succeeds, via the help pre-scan.
  Cover `--help` and `-h` in every position — alone, before the verb, after the verb, after a
  target, and alongside an otherwise-invalid flag — each yielding `help: true` and a nil error, so
  help wins over every other complaint.
  For every error case assert the concrete type is `usageError`, which is what batch 4's exit-code
  mapping keys off.
- **Commit:** `test(cli): pin flag parsing and its usage errors`

### Card 14: root discovery tests

- **Context:**
  - `internal/cli/root.go`
  - `internal/cli/flags.go`
  - `internal/engine/scratchtree_test.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/root_test.go`
  - `internal/cli/scratchtree_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/scratchtree_test.go` in `package cli` declaring
  `func writeScratchTree(t *testing.T, name string, files map[string]string) string`, mirroring
  `internal/engine/scratchtree_test.go`'s helper of the same name: resolve the module root from
  `runtime.Caller(0)`, build the tree under `.scratch/cli-tests/<name>/`, `os.RemoveAll` any stale
  tree first, create parent directories as needed, register a `t.Cleanup` that removes the tree,
  and return its absolute path. It writes regular files only — a test needing a symlink, an empty
  directory, or a `.git` entry creates it itself on the returned path. Its doc comment states that
  it never calls `t.TempDir()` because the system temp directory is banned for this repository's
  tests and `.scratch/` is the sanctioned location, and that it is a deliberate per-package copy
  because Go test helpers are not importable across packages. Cards 15 and 18 use this same helper.

  Create `internal/cli/root_test.go` in `package cli`, building every fixture with
  `writeScratchTree` and never calling `t.TempDir()`, `t.Chdir`, or `os.Chdir`.
  Cover `discoverRoot` finding a `.git` **directory** from a nested subdirectory several levels
  down; finding a `.git` **file** (the worktree case) the same way; and returning the nearest root
  when two are nested — in each case the fixture creates its own `.git` entry inside the tree
  `writeScratchTree` returns, so it is found before the walk can reach any real ancestor.

  The no-root-found case needs care and gets its own named test rather than a table row: a fixture
  under `.scratch/` is inside this repository, so a walk from it always finds this repository's own
  `.git` and the branch is unreachable from there. Test it instead by calling `discoverRoot` on the
  filesystem root path itself, which has no `.git` on a normal machine, and assert the error's
  concrete type is `usageError` and its message contains `pass --root`. Guard the case with a
  `t.Skip` naming the reason when a `.git` entry does exist at the filesystem root, so the test
  reports "not applicable here" rather than failing on an unusual machine. Do not assert the walk
  terminates at a specific absolute path — that is machine-dependent.
  State in the test's comment why `t.TempDir()` is not used to get an out-of-repository directory:
  the system temp directory is banned for this repository's tests, and the filesystem-root call
  exercises the same terminating branch without one.
  Cover `resolveRoot` short-circuiting the walk when `flagRoot` is given, even inside a tree that
  does contain a `.git`; accepting a relative `flagRoot` by joining it onto `cwd`; returning a
  `usageError` when `flagRoot` names a file rather than a directory, and when it names nothing at
  all, with the message echoing the path as given rather than the resolved one.
- **Commit:** `test(cli): pin root discovery and --root resolution`

### Card 15: target conversion tests

- **Context:**
  - `internal/cli/target.go`
  - `internal/cli/scratchtree_test.go`
  - `quarry/quarry.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/target_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/target_test.go` in `package cli`, using card 14's
  `writeScratchTree` for the one symlink case — creating the symlink itself on the returned path,
  since the helper writes regular files only — and pure string inputs elsewhere. Do not call
  `t.TempDir()`.
  Cover that a cwd-relative target and the equivalent absolute target convert to the same
  repository-relative path; that a target naming the root itself converts to `.`; that a nested
  target converts to a forward-slash path with no leading `./`.
  Cover the rebasing rule explicitly, as its own case: with `base` set to the root (the `--root`
  path), a relative target resolves under the root; with `base` set to a subdirectory (the no-`--root`
  cwd path), the same relative target resolves under that subdirectory. This is the case the
  discussion's round-2 review caught, and `golden-tests-run-the-cli-in-process` depends on it, so
  it gets its own named test rather than a table row.
  Cover that `..` escaping the root returns an error satisfying
  `errors.Is(err, quarry.ErrTargetOutsideRepo)`, as does an absolute target outside the root.
  Cover that native separators are accepted on input and normalised to forward slashes on output.
  Cover that a symlink named as the target is not resolved through: create a directory, a symlink
  pointing at it, and assert the conversion returns the symlink's own path rather than the
  directory's.
- **Commit:** `test(cli): pin target conversion, rebasing, and escape rejection`

## Batch Tests

`verify: go test ./internal/cli/...` runs the three new test files over the `writeScratchTree`
helper card 14 adds. Scoped to `internal/cli`, the only package this batch touches; the module-wide
`go build ./...` at the batch boundary catches a cross-package compile break. Every test here is
Loomyard-free and runs on any machine: fixtures are `.scratch/cli-tests/` trees or plain strings,
and no test changes the process working directory, so the package's tests remain safe to run in
parallel with each other. The one environment-dependent case — no repository root above the
filesystem root — skips with a reason rather than failing when the machine is unusual.
