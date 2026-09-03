# Batch: environment-and-provenance

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "environment-and-provenance"
number: 5
cards: 5
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [1, 2, 4]
```

## Batch Scope

This batch owns everything outside the process the harness measures: where the target checkout
comes from, where the pinned worktrees live and why that location is load-bearing for the blinding
gate, the single-run advisory lock, the per-cell MCP configuration document and the lazy server
build, and the committed provenance record with its merge-on-resume policy. It is one batch because
all of it is startup-and-teardown around the run loop, and the worktree path decision, the lock
path decision and the provenance record are three consequences of the same constraint. The external
interface batch 6 consumes is `ResolveQuarryRepoRoot`, `ResolveLoomyardRepo`, `ResolveWorktreeRoot`,
`AcquireRunLock`, `PrepareWorktree`, `RestoreWorktree`, `WorktreeStatus`, `MCPConfigDocument`,
`WriteMCPConfig`, `BuildServer`, and the provenance reader, collector, merger and writer.
It depends on batch 4 because every observation this batch produces — the memory-path scan, the
server-hash drift warning and the fingerprint comparison — is returned as the `Finding` type that
batch declares.

Batch-local decisions: the advisory lock lives in `worktree.go` rather than a file of its own,
because it is created at the worktree-root path that file already resolves. Every external process
this batch runs goes through the injectable runner seam the overview defines — no direct
`exec.Command` at a call site — which is what makes the whole offline test layer possible.

## Cards

### Card 20: environment resolution, worktree paths and the two startup assertions

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
  - `.scratch/ladder.env`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `worktree.go` declaring the `Cmd` struct, the
  `Runner` interface and the `ExecRunner` production implementation exactly as the overview's
  injectable-external-commands decision fixes them — including the `Env` map, which `ExecRunner`
  merges over the inherited environment, and the separate `Stderr` writer — then the environment
  layer on top of them.
  `ResolveLoomyardRepo(quarryRepoRoot string) (string, error)` reads the process environment
  variable `LADDER_LOOMYARD_REPO` first and, when it is unset or empty, parses simple
  `KEY=VALUE` lines out of `.scratch/ladder.env` beneath the quarry repository root — that exact
  path, a file directly inside the scratch directory and not inside the harness's own
  `.scratch/ladder/` subdirectory — ignoring blank lines and lines beginning with a hash. When neither yields a value, return an
  error naming both the variable and the file. Nothing else may read that variable: shell wrappers
  are banned and the documented entry point is a bare `go run`, which cannot source a file.
  `ResolveQuarryRepoRoot(start string) (string, error)` walks upward from the given directory to the
  enclosing git repository's top level and returns it; it is the single producer of that path, which
  cards 22, 23 and 26 and the run subcommand all take as an input.
  `ResolveWorktreeRoot(quarryRepoRoot string) (string, error)` resolves the base directory as the environment variable
  `LADDER_WORKTREE_ROOT` when set, else `XDG_CACHE_HOME` joined with `ladder-eval`, else the user's
  home directory joined with `.cache` and `ladder-eval`. Nothing is ever placed under a system
  temporary directory. It then asserts two invariants on the resolved absolute path and returns an
  error naming `LADDER_WORKTREE_ROOT` as the override when either fails: the path is not the
  supplied quarry repository root and is not under it, and the path does not contain the
  case-insensitive substring
  `quarry` — a lowercased `strings.Contains` written inline here, deliberately **not** routed
  through the shared matcher: that matcher's two classes govern content the harness searches, and
  this is a filesystem path the harness is about to use, tested case-insensitively where the
  matcher's composed-string form is case-sensitive. Card 2's doc comment records the same boundary
  from the other side. Both assertions exist because the cell's own working directory reaches the transcript
  and gate 2's checks (b) and (c) scan it — a worktree under the repository, or one merely named
  after it, would fail every control rep by construction. A task worktree is
  `<worktree-root>/worktrees/<task-id>`.
  `PrepareWorktree(ctx, r Runner, repo, taskID, pinnedSHA, dest string) error` creates a detached
  worktree of the target repository at the pinned commit when `dest` does not exist and otherwise
  leaves it in place; `RestoreWorktree(ctx, r Runner, dest string) error` returns it to the pinned
  state; `WorktreeStatus(ctx, r Runner, dest string) (string, error)` returns porcelain status
  output for the dirtied observation. All three go through the runner seam.
- **Commit:** `feat(ladder): resolve the target repo and pinned worktree paths outside the repo`

### Card 21: the single-run advisory lock

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `AcquireRunLock(worktreeRoot, resultsRoot string) (release func() error,
  err error)` to `worktree.go`. It creates the file `.ladder.lock` directly under the resolved
  worktree root — one level above the worktrees directory, never inside a task worktree, where it
  would trip the dirtied observation every rep and be removed by the pinned restore — opening it
  with the create-and-exclusive flags so a second holder fails rather than truncating. It writes the
  current process id and the results root into the file, and the returned release function removes
  it. When the file already exists, return an error whose message is exactly
  `another ladder run holds <path> (pid N, results <root>)` with the values read back from the
  existing file, falling back to a plain message when the file is unreadable or malformed. A stale
  lock is cleared by the operator; do not reap it automatically and do not test process liveness —
  guessing whether a pid belongs to a live run across a reboot is how a lock becomes a no-op.
- **Commit:** `feat(ladder): guard concurrent runs with an advisory lock above the worktrees`

### Card 22: the per-cell MCP config document and the lazy server build

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `mcp.go`. `MCPConfigDocument(l *Ladder, cfg
  Config, serverBinary, targetDir string) ([]byte, error)` returns the JSON document for one cell:
  for a control — a config whose allowed list is empty — exactly an object whose servers map is
  empty, and nothing else; for a granted cell, an object declaring exactly one server under the
  ladder file's server name, with the built binary as its command, the server block's argument list
  with every occurrence of the literal placeholder token `{target_dir}` — that exact spelling,
  braces included — replaced by the pinned worktree path, and the server block's environment map.
  This placeholder is a **new contract this plan defines**, not an existing one: no ladder file in
  the tree uses it today, and the migrated file ships with no argument list at all, so the
  substitution has no consumer until the MCP-server task writes one. Document the token in two
  places so that task spells it identically: a doc comment on this function naming it as the only
  placeholder the argument list supports, and one line in the migrated ladder file's header comment
  (card 5) stating that the server block's future argument list may use `{target_dir}` and that the
  harness substitutes the pinned worktree path for it. A granted cell whose ladder file declares no server block is an error naming the
  cell id and stating that the file declares no server block — raised here, at run time, never at
  load time. Also give `MCPConfigDocument` a doc comment stating that `{target_dir}` is the only
  placeholder the argument list supports.
  `WriteMCPConfig(dir string, doc []byte) (string, error)` writes the document under the
  harness's own scratch directory at `.scratch/ladder/` beneath the quarry repository root and
  returns its path; that location is deliberate and was measured — the invocation's own argument
  list is not echoed anywhere in the stream, so a configuration path under the repository never
  reaches a transcript. `BuildServer(ctx, r Runner, quarryRepoRoot, buildTarget, outPath string) (sha256 string,
  err error)` runs the Go build through the runner seam with the command's working directory set to
  the quarry repository root — the build target is written as a repository-relative path in the
  ladder file, so resolving it against the harness process's own working directory would make the
  build depend on where the operator happened to invoke it — and setting `CGO_ENABLED` to `1` in the
  command's environment map rather than mutating the harness's own process environment, because the target server links C grammars and the failure otherwise reads as
  unrelated, then returns the hex sha256 of the produced binary. The build happens at most once per
  invocation and only when a selected cell has a non-empty allowed list; when every selected cell
  is a control, nothing is built and the build target is never referenced.
- **Commit:** `feat(ladder): write per-cell mcp config and build the server lazily`

### Card 23: the provenance record and its merge-on-resume policy

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `provenance.go` declaring `Provenance` with `json`
  tags for `written_at`, `ladder_file`, `quarry_commit`, `quarry_dirty`, `quarry_dirty_files`,
  `loomyard_commit`, `loomyard_repo_sha256`, `hostname`, `go_version`, `claude_version`,
  `server_hashes` as a map keyed by the cell-and-rep pair, `selected_cells`, `reps_effective`,
  `memory_path_hashes`, `server_name`, `session_fingerprints` keyed the same way, and `invocations`
  as a slice. Declare `ProvenanceFile = "provenance.json"` as a package-level constant here, the
  single spelling of that name; card 28 does the same for the summary file and card 29 for the
  table file, each in the file that writes it.
  **Three fields exist in hashed or relative form precisely because this file is committed and no
  tracked file may carry a machine path.** The target-repository field is the hex sha256 of the
  resolved path, never the path. `ladder_file` is stored relative to the quarry repository root when
  the operator's value resolves under it, and as its base name otherwise — an operator who passes an
  absolute path must not thereby write their home directory into a tracked file. And the resolved
  auto-memory directories are **not** stored here as paths: they are real absolute host paths, so
  this record carries only `memory_path_hashes`, the hex sha256 of each, while the paths themselves
  are written to `raw/memory-paths.json` inside the results root's raw tree, which the
  results-tree-gitignore-scope decision keeps untracked. The resumed-run scan reads them from there.
  That placement is a deliberate refinement of the discussion's own wording, which said the paths go
  in the provenance record: the discussion also fixes that record as committed and forbids machine
  paths in tracked files, and those two statements cannot both hold. The raw tree is where the
  discussion already puts host-specific evidence, resume only ever happens on the host that produced
  the root, and the hashes keep "were these the same directories" answerable from the committed file
  alone.
  `SessionFingerprint` lifts, from one rep's session-init record, the CLI version, the model, the
  permission mode, the tool list, the server list, whether the memory-path map was non-empty, and
  the counts of skills and slash commands. `Invocation` carries `written_at`, `ladder_file`,
  `selected_cells`,
  `reps_effective`, `quarry_commit`, `quarry_dirty`, `quarry_dirty_files`, `loomyard_commit`,
  `loomyard_repo_sha256`, `claude_version`, `go_version`, `hostname`, `server_name`,
  `memory_path_hashes` and that invocation's `server_hashes` — every top-level field the merge
  derives has a carrier here, so no top-level field is left without a producer.
  Implement `ReadProvenance(resultsRoot string) (*Provenance, error)`, returning a nil record and no
  error when the file is absent — a fresh root, not a fault — and an error naming the file when it
  is present but unparseable. Implement `WriteProvenance(resultsRoot string, p *Provenance) error`.
  Implement `CollectInvocation(ctx context.Context, r Runner, in CollectInput) (Invocation, error)`,
  the single producer of an invocation's own facts, taking the quarry repository root, the operator's
  ladder-file path, the resolved
  target repository path, the ladder file's server name, the selected cell ids, the effective
  repetition count and the claude
  binary path, and filling every `Invocation` field except the two the run loop learns later:
  the quarry commit, dirty flag and dirty-file list from the repository's
  own status and revision output through the runner seam; the target repository's commit the same
  way; the hex sha256 of the resolved target-repository path string; the ladder-file path reduced to
  its repository-relative or base-name form per the rule above; the server name copied from the
  loaded file; the host name; the Go version
  from the runtime rather than a subprocess; and the CLI version from a version invocation of the
  claude binary through the same seam, because the probe host and the plan's reference host reported
  different versions and the value must be recorded rather than assumed. Exactly two fields are
  **not** filled here because they are unknown at startup: the server-hash map, which the run loop
  writes as it builds and assigns the binary, and the memory-path hashes, which the run loop writes
  once the first completed repetition reveals the paths. The run loop also writes each repetition's
  `SessionFingerprint` straight into the record's fingerprint map as that repetition completes, by
  the same route and for the same reason — a fingerprint is a per-repetition fact, so it cannot come
  from a once-per-invocation collector.
  Implement `MergeProvenance(existing *Provenance, next Invocation) (*Provenance, error)`: append
  the invocation, then derive the top-level fields from the invocation list — `selected_cells` is
  the **union** across invocations, because the set of reps a root was ever asked to produce is what
  the incomplete list needs; `memory_path_hashes` is the union; `server_hashes` merges by key;
  `written_at`, `quarry_commit`, `quarry_dirty`, `quarry_dirty_files`, `loomyard_commit`,
  `claude_version`, `go_version` and `hostname` take the **latest** invocation's value, since each
  answers "what was true the last time this root was written to" while every invocation's own value
  stays readable in the array — which is what makes "was the source edited mid-matrix" answerable
  across a resume; `ladder_file`, `loomyard_repo_sha256` and `server_name` must be **identical**
  across invocations, and a differing value is an error naming both, on the same footing as the rule
  below: a root assembled from two different ladder files, two different checkouts or two different
  server names is not one root.
  `reps_effective` must be identical across invocations, and a differing value is an error naming
  both, refusing the run at startup — a per-cell sample size that varies within one root breaks the
  only property that makes a root's numbers comparable. Never overwrite an existing record: a
  narrower second run would otherwise erase cells from the incomplete list and make an unfinished
  root read as finished.
  Implement `ScanMemoryPaths(paths []string) (*Finding, error)`, walking each named directory and
  returning a fatal finding naming the first file whose content matches the bare token `quarry`
  under the shared matcher. The two return channels divide as follows, and the division is
  load-bearing. A **named path that does not exist on disk** is a fatal `*Finding`, not an `error`
  and not silence: the harness cannot tell "this directory is clean" from "this directory was never
  scanned", and reporting the second as the first is precisely the silent failure V1's derived path
  produced. An **empty path list** — the CLI named no memory directory at all — is neither a finding
  nor an error: the run continues and the fact is recorded in the fingerprint, since an absent memory
  directory is the outcome the check wants. The `error` channel is reserved for an I/O failure while
  reading a directory or file that does exist. The paths are read from the transcript's session-init
  record, never
  derived: resolving the project directory from a repository path requires V1's reverse-engineered
  name mangling, which this task deletes and must not resurrect, and which would fail silently by
  scanning a directory that does not exist. Add `WarnOnServerHashDrift(p *Provenance) *Finding`
  returning a non-fatal finding when more than one distinct server hash appears in a root, and
  `CompareFingerprints(p *Provenance) []Finding` reporting each rep's drift from the root's first
  fingerprint as non-fatal observations. Do not carry V1's build-stamp field: it read a
  structurally constant value and carried no information.
- **Commit:** `feat(ladder): record provenance and merge it across resumed invocations`

### Card 24: environment, mcp and provenance tests

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/worktree_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/mcp_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write `worktree_test.go` with a recording fake runner asserting the exact
  argument vectors for worktree creation, restore and status, plus a table over the worktree-root
  assertions: a path under a supplied repository root is refused and the error names the override
  variable; a path containing `Quarry` in any casing is refused; the cache-derived default is
  accepted; an explicit override outside the repository and free of the token is accepted. Add
  environment-resolution cases: the process variable wins; with it unset the env file under a
  temporary root is parsed and comment and blank lines are ignored; with neither, the error names
  both. Add lock cases: acquiring twice fails with a message carrying the first holder's pid and
  results root; the release function removes the file; and the lock path is the worktree root's own
  child, asserted by string equality so a future refactor cannot move it inside a task worktree.
  Write `mcp_test.go` asserting a control cell's document has an empty servers map and no other
  key, a granted cell's document names exactly one server with the placeholder substituted, a
  granted cell against a ladder file with no server block errors naming the cell id, and the build
  helper sets the cgo variable in the command's environment map rather than in the harness process's
  own environment — asserted by inspecting the recorded command, and by asserting the harness's own
  environment is unchanged after the call — and that the recorded command's working directory is the
  supplied quarry repository root, so a repository-relative build target resolves. Write `provenance_test.go` asserting: a read of a root
  with no record returns a nil record and no error while a malformed record errors naming the file;
  a written record round-trips; the collector fills the commit, dirty, host, Go-version and
  CLI-version fields from the recording fake runner's canned output, puts the target repository's
  **hash** where the path would be, and reduces an absolute ladder-file path under the supplied
  repository root to its repository-relative form and one outside it to its base name; and the
  merge policy: two invocations union their selected cells and memory-path hashes, server hashes
  merge by key, the latest invocation's commit and version fields win at the top level while both
  invocations stay readable in the array, a differing effective rep count is refused naming both
  values, a differing ladder file, target-repository hash or server name is likewise refused, and a
  record built from an invocation whose inputs were all absolute paths round-trips through JSON with
  **no absolute path anywhere in the output** — assert that by scanning the marshalled bytes for the
  temporary directory's own prefix, so the assertion fails if any future field reintroduces one.
  Add memory-scan cases over a temporary
  directory: a clean directory yields no finding, a file containing the token yields a fatal
  finding naming the file, a named directory that does not exist yields a **fatal finding** naming
  it rather than silence or an error, and an empty path list yields neither a finding nor an error. Add a server-hash-drift case and a fingerprint-drift case.
- **Commit:** `test(ladder): cover environment resolution, the lock, mcp config and provenance`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers this batch through three new test files.
Two assertions carry more weight than the rest: the worktree-root refusal table, because those two
startup assertions are what make gate 2's checks (b) and (c) satisfiable by construction rather
than by luck, and the memory-scan case for a non-existent directory, because reporting that case is
the whole reason the paths are read from the transcript instead of derived.
