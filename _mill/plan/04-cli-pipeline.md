# Batch: cli-pipeline

```yaml
task: "Facade + CLI, toc (T5a)"
batch: "cli-pipeline"
number: 4
cards: 3
verify: go test ./internal/cli/... ./cmd/quarry/...
depends-on: [2, 3]
```

## Batch Scope

This batch composes batch 3's four pure layers and batch 2's renderers into `cli.Run` — the whole
CLI below `os.Exit` — and adds the one-line `cmd/quarry` binary over it. It owns the request
pipeline's fixed six-step order, the exit-code mapping, the CLI-authored error messages, and the
failure envelope. It is one batch because those four things are a single contract: the step that
decides an exit code is the step that decides the message, and splitting them would let the two
drift, which is the V1 `ok`-versus-exit-code defect in a different costume.

After this batch `quarry toc <target>` is a working command. The external interface batch 5
consumes: `cli.Run`.

Batch-local decision: the failure path is emitted by one helper rather than at each `return` site,
so the "stdout gets the envelope, stderr gets the same sentence" rule is written once and cannot be
half-applied.

## Cards

### Card 16: Run, the request pipeline, and the exit-code mapping

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/root.go`
  - `internal/cli/target.go`
  - `internal/cli/usage.go`
  - `internal/cli/doc.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `quarry/render.go`
  - `quarry/text.go`
  - `internal/engine/repo.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/cli.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/cli.go` in `package cli`, importing
  `github.com/Knatte18/quarry/quarry`.

  Declare the four exit codes as named constants with doc comments giving each one's meaning:
  `exitOK = 0` (answered), `exitNegative = 1` (the query has a negative answer),
  `exitUsage = 2` (the caller asked wrong), `exitInternal = 3` (an I/O or render failure).

  Declare `func fail(stdout, stderr io.Writer, code int, msg string, withUsage bool) int`: it writes
  `quarry.RenderErrorJSON(msg)` to `stdout`, writes `msg` followed by `"\n"` to `stderr`, then
  writes `usageText` to `stderr` as well when `withUsage` is true, and returns `code`. Its doc
  comment records that the JSON envelope always goes to stdout — including on failure and including
  under `--text` — so a pipeline parsing stdout finds a parseable object on every path, and that the
  stderr sentence is byte-identical to the envelope's `error` value so the two channels can never
  disagree about what went wrong. It also records that there is no text rendering of a failure: the
  payload is two fields with no prose in it, and a `--text` caller that must distinguish success
  from failure reads the exit code.

  Declare `func Run(args []string, stdout, stderr io.Writer) int`, where `args` is `os.Args[1:]`.
  Execute exactly these steps in this order:
  1. `parseArgs(args)`. On a `usageError`, return `fail(..., exitUsage, err.Error(), true)`. On any
     other non-nil error, return `fail(..., exitInternal, "internal error: "+err.Error(), false)`.
  2. When the parsed request has `help` set, write `usageText` to **stdout** and return `exitOK`.
     Help is a successful query about the CLI, not a usage error; usage text on stderr with exit 2
     is reserved for the case where the caller got the invocation wrong.
  3. `os.Getwd()`. On error return `fail(..., exitInternal, "internal error: "+err.Error(), false)`.
     This is the one place in the package that reads the working directory.
  4. `resolveRoot(req.root, cwd)`. On a `usageError` return `fail(..., exitUsage, ..., true)`.
  5. Compute the base for a relative target: the resolved root when `req.root` was given, the
     working directory otherwise. Call `repoRelTarget(root, base, req.target)`. When the error
     satisfies `errors.Is(err, quarry.ErrTargetOutsideRepo)`, return
     `fail(..., exitNegative, "target outside repository: "+req.target, false)` — naming the target
     **as given**, since a target that escaped the root has no meaningful repository-relative form.
     Any other error is `exitInternal`.
  6. `os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))`. `os.IsNotExist(err)` returns
     `fail(..., exitNegative, "target not found: "+rel, false)`, naming the **repository-relative**
     target. Any other stat error is `exitInternal` with the `internal error: ` prefix. Otherwise
     record `targetIsFile := !info.IsDir()`. Use `os.Lstat`, never `os.Stat`: a symlink named as the
     target is treated as a file and not followed, which is the rule `engine.resolveTarget` already
     follows for the same reason.
  7. `quarry.Open(root)`. On error, `exitInternal`.
  8. `repo.TOC(rel, quarry.TOCOptions{Depth: req.depth, Symbols: req.symbols})`. When the error
     satisfies `errors.Is(err, quarry.ErrTargetNotFound)` return
     `fail(..., exitNegative, "target not found: "+rel, false)`; when it satisfies
     `errors.Is(err, quarry.ErrTargetOutsideRepo)` return the outside-repository sentence at
     `exitNegative`; any other error is `exitInternal`. Steps 5 and 6 have already excluded both in
     the common case — these branches exist because the target can be removed between the stat and
     the walk, and the CLI must not report that race as success. Say so in a comment.
  9. Render. Under `--text`, write `quarry.RenderText(answer, targetIsFile)` to stdout. Otherwise
     call `quarry.RenderJSON(answer)`; on its error return `exitInternal` with the
     `internal error: ` prefix, and otherwise write the bytes. A write failure on stdout is
     `exitInternal` — that is an I/O failure, which is what 3 already means. Return `exitOK`.

  The `error` value never carries the engine's wrapped chain for exit 1 or exit 2: those name
  conditions quarry itself defines, so quarry spells them, and passing
  `engine: resolve target "x": engine: target not found` through would leak an internal package
  name into a public contract. Exit 3 is the opposite case and carries `err.Error()` whole behind
  the one `internal error: ` prefix. Record this in `Run`'s doc comment.
  Every message is single-line: do not embed a newline in any `fail` message.
- **Commit:** `feat(cli): add Run, the request pipeline, and the exit-code mapping`

### Card 17: the quarry binary

- **Context:**
  - `internal/cli/cli.go`
  - `.gitignore`
- **Edits:** none
- **Creates:**
  - `cmd/quarry/main.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `cmd/quarry/main.go` in `package main`, importing
  `github.com/Knatte18/quarry/internal/cli` and `os`. Its `main` is exactly
  `os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))` and the file holds nothing else — no flag
  handling, no error formatting, no output. The file-level comment records why: everything below
  `os.Exit` is testable in-process, so the goldens capture the binary's exact bytes without a build
  step or `os/exec`, and any logic added here would be the one part of the CLI no test covers.
  It also records that the binary is built as `quarry` at the repository root, which `.gitignore`
  already reserves — `/quarry` ignores the built binary while `!/quarry/` keeps the facade package
  directory tracked — so no `.gitignore` change is needed and none is made.
- **Commit:** `feat(cmd): add the quarry binary over cli.Run`

### Card 18: Run pipeline, exit-code, and failure-envelope tests

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/root.go`
  - `internal/cli/target.go`
  - `internal/cli/usage.go`
  - `quarry/quarry.go`
  - `quarry/render.go`
  - `quarry/text.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/cli_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/cli_test.go` in `package cli`. Every case calls
  `Run(args, &stdout, &stderr)` with `bytes.Buffer` sinks and an explicit `--root` pointing at a
  `t.TempDir()` fixture tree, so no test changes the process working directory. Build the fixture
  tree in the temp dir: a directory holding a Go file with a package clause and a doc comment, a
  nested subdirectory, and a symlink to a directory.
  Cover, per `discussion.md`'s Testing block:
  *Exit-code mapping* — a directory target and a file target returning 0; a missing target returning
  1; a `..` target escaping the root returning 1; an unknown flag, a missing target, two targets, an
  unparseable `--depth`, and a `--root` naming a file each returning 2.
  *Request-pipeline ordering* — a target escaping the root returns 1 with the
  `target outside repository:` sentence, proving the check ran before the stat; a missing target
  returns 1 with the `target not found:` sentence naming the repository-relative path; a symlink
  pointing at a directory, given as the target, renders in the **file** form under `--text`, which
  is only possible if `targetIsFile` came from an `os.Lstat` rather than an `os.Stat`.
  *Error-message derivation* — for each non-zero code, the first line written to stderr is
  byte-identical to the `error` value in the stdout envelope; the exit-1 and exit-2 messages never
  contain the substring `internal/engine`; the exit-3 message begins `internal error: `.
  *Failure envelope* — for each non-zero code, stdout is exactly
  `{"ok":false,"error":"<msg>"}` plus a newline and nothing else, and stderr is non-empty; the two
  are never swapped. Repeat the whole set **with `--text` added** to pin that the failure envelope
  stays JSON regardless of the view flag.
  *Usage text placement* — an exit-2 invocation writes the complaint and then `usageText` to
  stderr, and writes no usage text to stdout.
  *`--help`* — `--help` and `-h`, with and without a verb and at several argument positions, write
  `usageText` to stdout, write nothing to stderr, and return 0.
  *Success output* — a directory target with no `--text` writes pretty-printed JSON with no `"ok"`
  key and a single trailing newline; the same target with `--text` writes the text view; both write
  nothing to stderr.
  *Flag pass-through* — `--depth all` and `--symbols` reach the engine, asserted by observing that
  the answer for a nested tree carries the deeper `dirs` entries and that file entries carry
  `symbols`, rather than by mocking `TOC`.
- **Commit:** `test(cli): pin the request pipeline, exit codes, and the failure envelope`

## Batch Tests

`verify: go test ./internal/cli/... ./cmd/quarry/...` runs batch 3's three test files plus this
batch's `cli_test.go`, and compiles `cmd/quarry` (which has no tests of its own — its one line is
covered by every `Run` case in `cli_test.go`). Scoped to the two packages this batch touches; the
module-wide `go build ./...` at the batch boundary catches a cross-package break. Every case here
is Loomyard-free and builds its own `t.TempDir()` fixture, so the batch verifies on any machine.
