# Batch: cli-argument-layer

```yaml
task: "Facade + CLI, resolve + expand (T5b)"
batch: "cli-argument-layer"
number: 2
cards: 3
verify: go test ./internal/cli/
depends-on: [1]
```

## Batch Scope

This batch widens the command line's argument layer to three verbs without touching the request pipeline: the path helper is split so a target that escapes the root can reach the engine, the parser gates three verbs with per-verb flag validity and per-verb arity, and the help text is rewritten to describe the widened surface. It is one batch because all three files are small, are read together by anyone changing the argument layer, and share one property the parser's fixture-free table test rests on: nothing here resolves a path, stats anything, or reads a working directory.

The external interface the next batch consumes is: `repoRelPath`, the arithmetic-only path helper; a `request` whose `verb` field is one of three words; and the rewritten `usageText`.

Batch-local decisions beyond the overview's Shared Decisions:

- The parser stays hand-rolled and its `request` struct gains no field. The depth and symbols fields simply stay at their zero values for the two new verbs, which is what a per-verb flag-validity check makes safe.
- The help text keeps one combined flags list rather than three per-verb sections, with the three table-of-contents-only flags marked as such in their own descriptions. That is what makes the per-verb rejection messages legible from the help text alone.
- After this batch the parser accepts the two new verbs while the pipeline still answers every verb as a table-of-contents query. That intermediate state is not user-visible: no test exercises it, and batch 3 closes it.

## Cards

### Card 7: split the path helper into arithmetic and rejection halves

- **Context:**
  - `internal/engine/repo.go`
  - `internal/cli/scratchtree_test.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/target.go`
  - `internal/cli/target_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Split `repoRelTarget` in `internal/cli/target.go` into two functions, behaviour-preserving for every existing caller:

  ```go
  func repoRelPath(root, base, target string) (string, error)
  func repoRelTarget(root, base, target string) (string, error)
  ```

  `repoRelPath` holds the arithmetic that `repoRelTarget` holds today — the absolute-versus-relative join, the `filepath.Rel` call, the conversion to forward slashes, and the final `path.Clean` — and returns the cleaned relative form even when it begins with `..`. It returns an error only when `filepath.Rel` itself fails, and that error stays `quarry.ErrTargetOutsideRepo`, which is the value the existing implementation already returns there.

  `repoRelTarget` becomes a wrapper: it calls `repoRelPath`, returns that call's error unchanged when there is one, then applies the existing leading-`..` rejection, returning `quarry.ErrTargetOutsideRepo` when the result is exactly `..` or begins with `../`, and otherwise returns the value unchanged. Its exported behaviour is byte-identical to today's for every input.

  Do not add a mode flag to either function. Two named functions read better than one function whose contract a boolean argument changes, for two call sites.

  Move the existing doc comment's arithmetic prose — the no-filesystem-access rule, the never-resolve-symlinks rule, the native-separators-in-forward-slashes-out rule — onto `repoRelPath`, and leave `repoRelTarget`'s own comment naming what it adds: the outside-repository rejection, and which caller wants it. State on `repoRelPath` that a caller which wants a target that escapes the root to reach the engine calls this one, so the engine's own outside-repository rule produces the answer rather than the command line synthesising a second copy of it.

  Update the file's header comment, which today names only `repoRelTarget`.

  In `internal/cli/target_test.go`, every existing case must keep passing unchanged against `repoRelTarget` — that is this card's own proof the split changed nothing. Add cases asserting that `repoRelPath` returns the leading-`..` form rather than rejecting it, for a sibling directory of the root and for a parent of the root, and that `repoRelPath` and `repoRelTarget` agree on every input that does not escape the root, including the root itself and a nested path.
- **Commit:** `refactor(cli): split repoRelTarget into arithmetic and rejection halves`

### Card 8: three verbs, per-verb flag validity, per-verb arity

- **Context:**
  - `internal/cli/cli.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/flags.go`
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Change `parseArgs` in `internal/cli/flags.go` as follows, keeping its hand-rolled structure, its `request` struct unchanged, and its purity over the argument slice — it must still resolve no path, stat nothing, and read no working directory:

  - The verb gate accepts `toc`, `resolve` and `expand`. The no-verb message — emitted both for an empty argument slice and for a first argument beginning with `-` — becomes `no verb given; expected: toc, resolve, or expand`. The unknown-verb message keeps its current wording.
  - Flag validity is checked per verb. `--depth`, `--symbols` and `--no-symbols` are valid for `toc` only; on either new verb each is a usage error whose message is the flag's own name, then ` is not valid for `, then the verb — for example `--depth is not valid for resolve`. `--text` and `--root` stay valid for all three verbs, and the existing `--help` and `-h` scan, which wins from any position before anything else is examined, is unchanged.
  - The rejection fires at the point the flag is recognised, so a per-verb rejection takes precedence over that flag's own value validation. Giving `--depth` a bad value on `resolve` reports the not-valid-for-this-verb message, not the bad-value one.
  - The arity message is spelled per verb: the verb, then ` takes exactly one target, got `, then the count. Every verb still requires exactly one target.
  - After the arity check, and only for `expand`, reject a target containing no `#` with the usage message `expand takes a glyph (a target containing "#"), got: ` followed by the target as given. Use `strings.Contains(target, "#")`. This belongs in the parser because it keeps the rejection pure over the argument slice — no root discovery, no engine call — which is the property the parser's fixture-free table test rests on, and because the exit-2 constant is documented as a path the query is never run on.

  Every message above is a `usageError`, the existing type, so each maps to exit 2 through the existing pipeline without any change at the call site.

  Extend `parseArgs`'s doc comment to state the three-verb gate and the per-verb flag-validity rule, and to state that the missing-separator rejection for the new glyph-only verb lives here rather than after the engine has run.

  In `internal/cli/flags_test.go`, three existing rows of `TestParseArgs_UsageErrors` are invalidated by this card and must be updated in it, not merely added to:

  - the `missing-verb` row and the `first-arg-is-flag` row both assert the old no-verb message and must assert the new one;
  - the `unknown-verb` row uses `resolve` as its unknown verb, which this card makes valid — `parseArgs` then returns a nil error and the row fails on its own guard. Replace that argument with a word that is none of the three verbs, and update the expected message to match.

  Then extend the existing pure table with: the three-verb gate, including the new no-verb message and the unchanged unknown-verb message; every per-verb flag-validity rejection, one case per flag per new verb; both new arity messages, for zero targets and for two; the missing-separator rejection and its exact message; a `#`-bearing target accepted by the parser for the glyph-only verb, since the grammar's own rejection of it belongs to a later stage; and `--help` and `-h` still winning from any position for each of the three verbs. Assert exact message strings. Every case stays fixture-free.
- **Commit:** `feat(cli): gate three verbs with per-verb flag validity and arity`

### Card 9: rewrite the help text for three verbs

- **Context:**
  - `internal/cli/flags.go`
- **Edits:**
  - `internal/cli/usage.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Replace the value of `usageText` in `internal/cli/usage.go` with exactly this, ASCII only, no typographic characters, so it stays byte-comparable in tests:

  ```
  quarry - a table of contents for a source repository

  usage:
    quarry toc <target> [--depth N|all] [--symbols|--no-symbols] [--text] [--root <path>]
    quarry resolve <glyph|path> [--text] [--root <path>]
    quarry expand <glyph> [--text] [--root <path>]

  flags:
    --depth <N|all>   toc only: how far to recurse into subdirectories (default 0)
    --symbols         toc only: populate every file entry's symbols
    --no-symbols      toc only: leave every file entry's symbols unpopulated
    --text            emit the lossless text view instead of JSON
    --root <path>     use <path> as the repository root instead of discovering one
    -h, --help        print this text and exit 0

  exit codes:
    0  answered
    1  negative answer: not found, outside the repository, ambiguous, not a type,
       or not a well-formed glyph
    2  usage error
    3  internal error
  ```

  The flags list stays one combined list rather than three per-verb sections, with the three table-of-contents-only flags marked as such in their own descriptions: that is what makes the per-verb rejection messages legible from the help text alone, without repeating the three shared flags three times. The usage block carries the per-verb bracketed lists so a reader sees each verb's real shape; the two blocks together state validity once each, from two directions.

  Keep the existing comment on the constant stating that it is ASCII only for byte-comparability and that there is no JSON flag because JSON is the default, and extend it to state why the flags list is not split per verb.

  Every existing test that asserts on the help text references the constant by name rather than quoting its content, so no test changes in this card.
- **Commit:** `docs(cli): rewrite the usage text for three verbs`

## Batch Tests

`verify: go test ./internal/cli/` runs the whole `internal/cli` test binary. That scope is deliberate rather than narrower: card 7 changes a helper the pipeline tests exercise indirectly, and card 9 changes a constant several test files compare against, so a per-file scope would miss exactly the regressions this batch could cause. The package's test binary runs in well under a second, and the evidence-golden test in it skips on a machine with no Loomyard checkout, so nothing in this scope is slow or machine-dependent.

The batch's own new assertions live in `internal/cli/flags_test.go` (the three-verb gate, per-verb flag validity, per-verb arity, the missing-separator rejection) and `internal/cli/target_test.go` (the split's behaviour-preservation for the existing helper, plus the new arithmetic-only cases).
