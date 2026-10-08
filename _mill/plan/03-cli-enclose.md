# Batch: cli-enclose

```yaml
task: 'Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)'
batch: cli-enclose
number: 3
cards: 3
verify: go test ./internal/cli/
depends-on: [2]
```

## Batch Scope

This batch delivers the `quarry enclose [--rev <rev>] [--root <path>] (<location>... | --stdin)` verb in `internal/cli`: argument parsing and per-verb flag scoping in `flags.go`, an unexported stdin-taking `run` behind the unchanged `Run`, the `runEnclose` pipeline, the usage text, and a payload-only JSON golden.
It consumes the facade's `EncloseAt` and `RenderEncloseJSON` from batch 2 and nothing else new.
It is one batch because the parser, pipeline and usage text change together and share one test package.

## Cards

### Card 9: parse the enclose verb

- **Context:**
  - `_mill/discussion.md`
- **Edits:**
  - `internal/cli/flags.go`
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add three fields to `request`: `targets []string` (the enclose verb's locations, in order), `rev string` (the `--rev` value, empty when absent) and `stdin bool` (whether `--stdin` was given); document them in `request`'s doc comment.
  - In `parseArgs`: the verb gate accepts `enclose`, and both "no verb given" messages become `no verb given; expected: toc, glyphs, resolve, expand, delta, enclose, or name`.
    `--rev` is valid for `enclose` only, requires a value, and an empty value is `--rev value must not be empty` (as `--from`/`--to` are).
    `--stdin` is valid for `enclose` only and takes no value.
    `--text` given with `enclose` is `--text is not valid for enclose`.
    `--root` stays valid for `enclose`.
    For `enclose`, the "exactly one target" check is replaced by: `--stdin` together with any positional location is `enclose takes locations as arguments or --stdin, not both`; neither is `enclose requires at least one location or --stdin`; otherwise `req.targets` holds the positional locations.
    Every other verb's parsing is unchanged.
  - In `parseGlyphsArgs`, reject `--rev` and `--stdin` with the existing `%s is not valid for %s` message naming `glyphs`, and add both flags to the list of rejected flags in `parseGlyphsArgs`'s doc comment.
  - Update `parseArgs`'s doc comment to state the enclose verb's flags and its many-targets rule, and rewrite its existing sentences the new verb makes false, naming verbs as a group rather than listing or counting them: "The verb gate accepts exactly ... delta and name" (the gate now also accepts enclose), "--text is valid for every verb" (not for enclose), "--root is valid for the five repository verbs (toc, glyphs, resolve, expand and delta)" (the repository verbs, enclose included) and "Every verb requires exactly one target" (every verb but enclose).
  - In `internal/cli/flags_test.go`, update the two expected "no verb given" strings in `TestParseArgs_UsageErrors`, and add a table test `TestParseArgs_Enclose` covering: several positional locations in order; `--stdin` alone; `--rev` in both spellings; `--root`; both positional and `--stdin`; neither; `--text`; an empty `--rev` (`--rev ""` and `--rev=`); `--rev` and `--stdin` on `toc`, `resolve`, `delta` and `glyphs`.
- **Commit:** `feat(cli): parse the enclose verb`

### Card 10: run the enclose verb

- **Context:**
  - `_mill/discussion.md`
  - `internal/cli/flags.go`
  - `internal/cli/cli_test.go`
  - `internal/cli/scratchtree_test.go`
  - `internal/repopath/target.go`
  - `quarry/enclose.go`
  - `quarry/render.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/usage.go`
  - `internal/cli/doc.go`
- **Creates:**
  - `internal/cli/enclose_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - In `internal/cli/cli.go`, move `Run`'s body into a new unexported `func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int`; `Run` keeps its signature and returns `run(args, os.Stdin, stdout, stderr)`.
    `cmd/quarry/main.go` is not changed.
  - Add a `case "enclose":` to the dispatch switch calling a new `runEnclose(req request, root string, stdin io.Reader, stdout, stderr io.Writer) int`:
    1. The locations are `req.targets`, or, under `req.stdin`, the result of reading `stdin` whole with `io.ReadAll` (a read error is `fail(..., exitInternal, "internal error: "+err.Error(), false)`), splitting on `\n`, trimming each line with `strings.TrimSpace`, and skipping blank lines; empty input gives an empty location list.
    2. Locations are passed to the facade verbatim; `runEnclose` performs no `repopath.RepoRelTarget` step and never uses the working directory, because the facade resolves relative paths against the root.
    3. `quarry.Open(root)` (failure is exit 3), then `repo.EncloseAt(req.rev, locations)`.
       A whole-call error maps exactly as `runDelta`'s does, through `codeForDeltaError`, with the sentences `enclose: unknown revision <rev>`, `enclose: root <root> is not the repository top level (top level is <toplevel>)` and `enclose: root is not a git repository: <root>` built from `quarry.UnknownRevisionError`, `quarry.RootNotTopLevelError` and `quarry.ErrNotARepository` with usage on stderr, and anything else `internal error: <err>`.
    4. Write `quarry.RenderEncloseJSON(results)` to stdout (a render or write error is exit 3) and return `exitOK` whatever the per-item statuses.
    Extend `Run`'s doc comment with `runEnclose`'s numbered pipeline in the style of the existing per-verb paragraphs.
    Every "four repository verbs" phrase in `cli.go` (in `Run`'s doc comment and in the dispatch switch's `default` comment) is rewritten to name the repository verbs as a group ("the repository verbs") rather than count them, and `Run`'s doc sentence "calls one of runTOC, runResolve, runExpand, or runDelta" is rewritten to say it calls the verb's own `runX` pipeline, without listing them.
  - In `internal/cli/usage.go`, add the usage line `quarry enclose (<location>... | --stdin) [--rev <rev>] [--root <path>]`, followed on that same usage entry by the statement that location paths are relative to the repository root, not the working directory (an indented continuation line directly under the `enclose` line is acceptable if the line would otherwise be too long), flag lines `--rev <rev>` ("enclose only: answer at this revision instead of the working tree") and `--stdin` ("enclose only: read one location per line from standard input"), and a note on `--text` that it is not valid for enclose.
    The `enclose` entry also names the three location spellings `path:line`, `path:line-line` and `path:line:col`.
    Keep the text ASCII only.
  - In `internal/cli/doc.go`, update every passage the new verb makes false, naming the subsystem rather than counting or listing verbs:
    the sentence saying input is "interpreted where the user is" (enclose location paths are repository-root relative);
    "The command has six verbs, though only five pipelines" (no count);
    the paragraph saying "toc" and "delta" are the two verbs that take a path and are converted with `internal/repopath` in this package (enclose takes locations whose paths the facade resolves against the repository root, with no cwd-relative conversion here);
    and the closing paragraph's "explicit error for both" / "at the two that take a path instead" (enclose answers a `#` in a path segment as an `unaddressable` item, not a usage error).
  - Tests in `internal/cli/enclose_test.go`, calling `run` with a `strings.Reader` for stdin (a helper `runCLIStdin(args []string, stdin string)` beside `runCLI`'s shape), on scratch trees passed with `--root` unless stated:
    positional locations; `--stdin` with blank lines and `\r\n` line endings; empty `--stdin` input answering `[]` with exit 0; both and neither (exit 2); `--text` (exit 2); `--rev` on `toc` (exit 2); an empty `--rev` (exit 2); a bare `foo_test.go:42` answering `missing_file` with exit 0; a `path:line:col` location; a run from a subdirectory of a `newDeltaCLIFixture` repository with no `--root` (`t.Chdir`), where a repository-relative location resolves against the root rather than the cwd; an unknown `--rev` on a `newDeltaCLIFixture` repository (exit 2, failure envelope `enclose: unknown revision <rev>`); a known `--rev` answering at the revision's line numbers.
- **Commit:** `feat(cli): add the enclose verb`

### Card 11: enclose JSON golden

- **Context:**
  - `_mill/discussion.md`
  - `internal/cli/cli.go`
  - `internal/cli/name_golden_test.go`
  - `internal/cli/loomyard_test.go`
  - `internal/cli/scratchtree_test.go`
  - `internal/cli/enclose_test.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/enclose_golden_test.go`
  - `internal/cli/testdata/enclose/batch.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create `internal/cli/enclose_golden_test.go` with `TestEncloseGolden`, following `name_golden_test.go`'s payload-only pattern (its own `compareEncloseGolden` helper honouring the package's existing `updateGoldens` flag, writing under `testdata/enclose/`).
    It builds a fixed scratch tree (`writeScratchTree`) with a small Go package holding a documented function, an interface with a method, a struct type and a `const (` group, plus a root-level `.go` file, and runs one `run` invocation with `--root` and `--stdin` over a fixed list of locations: one found single line, one range over two members, one interface head plus method, one not_found `const (` line, one `past_eof`, one `missing_file`, one `unsupported_language` (`.md`), one root-level `unaddressable`, one `bad_location`, and one `bad_range`.
    The run must exit 0, and its stdout bytes are compared with `testdata/enclose/batch.json`.
  - Generate `testdata/enclose/batch.json` with `go test ./internal/cli/ -run TestEncloseGolden -update`, read it, and confirm by eye that every entry's keys, statuses, reasons and error texts match the discussion and the overview's "rejection message texts" Shared Decision before committing it.
    The golden carries no absolute path: every location in the list is repository-relative.
  - Before committing, run `golangci-lint run` from the repository root and fix every finding in files this plan touched, so the discussion's lint constraint holds at the end of the last batch (the effective `pipeline.done_gate` runs the same command again before the task is marked done).
- **Commit:** `test(cli): pin the enclose JSON golden`

## Batch Tests

`verify: go test ./internal/cli/` runs the CLI package, which holds every test this batch adds or changes: `TestParseArgs_Enclose` and the updated `TestParseArgs_UsageErrors` in `flags_test.go`, `enclose_test.go`, and `TestEncloseGolden`.
The package scope is right because `run` replaces `Run`'s body, so every existing `Run` test in the package is a regression test for card 10.
