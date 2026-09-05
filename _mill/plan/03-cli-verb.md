# Batch: cli-verb

```yaml
task: 'The glyph-maker: declaration to glyph (P1, roadmap 2b)'
batch: 'cli-verb'
number: 3
cards: 6
verify: go test ./internal/cli/
depends-on: [2]
```

## Batch Scope

This batch adds the fourth CLI verb: flag parsing, the exit-code mapper, the verb's own pipeline,
the help text, and the tests and goldens that pin all of it. It is one batch because the six cards
edit four small files that are read together — a change to the flag gate without the matching
dispatch, help text, and message assertions leaves the package in a state no reviewer can evaluate.
The batch touches the shared `Run` pipeline in one specific way — `name` dispatches before the
working directory is read — and rewrites the doc comments that state the old step count.

Batch-local decisions. First, `--unit` presence is tested by comparing the parsed value against the
empty string rather than by carrying a separate "flag was seen" bool: an explicit `--unit ""` is
reported as missing, which is the right answer for the same reason an absent flag is, and the simpler
shape has one fewer state to reason about. Second, the goldens are produced by running the new table
under the package's existing update flag against no checkout at all — every case here is
machine-independent, so this is the one golden table in the package that needs no environment.

## Cards

### Card 8: the `name` verb in parseArgs

- **Context:**
  - `quarry/name.go`
  - `quarry/quarry.go`
  - `internal/cli/cli.go`
  - `internal/cli/usage.go`
- **Edits:**
  - `internal/cli/flags.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/cli/flags.go`:

  Add a `unit string` field to `request`, documented as holding the `--unit` value exactly as given,
  empty when the flag was absent.

  Widen the verb gate: the `verb != "toc" && verb != "resolve" && verb != "expand"` chain gains a
  fourth clause for `name`.

  Both `"no verb given; expected: toc, resolve, or expand"` messages — they are byte-identical and
  both must change — become `"no verb given; expected: toc, resolve, expand, or name"`.

  Add a `--unit` case to the flag switch. It is valid for `name` only: for any other verb return the
  per-verb rejection `fmt.Sprintf("%s is not valid for %s", name, verb)`, checked at the point the
  flag is recognised so the rejection takes precedence over value validation, exactly as the
  `--depth` case already does. Otherwise read the value through `nextValue` and reject a missing one
  with the existing `"%s requires a value"` shape.

  Add a `name` rejection to the `--root` case, with the same per-verb message shape: the verb reads
  nothing from the filesystem, and a flag that is accepted and does nothing is worse than one refused
  with a reason.

  `--depth`, `--symbols` and `--no-symbols` need no new code — their existing `verb != "toc"` guards
  already reject them for `name` — but the card's tests must pin that they do.

  After the target-count check, add: when the verb is `name` and `req.unit` is empty, return the
  usage error `"--unit is required for name"`. This follows the per-verb rejection shape rather than
  the value-shape one, since it names a verb requirement rather than a malformed flag value.

  Update the `parseArgs` doc comment in the same edit: the verb gate now accepts four verbs; the
  sentence `--text and --root are valid for all three verbs` becomes false in two ways at once and
  must be rewritten to say that `--text` is valid for every verb while `--root` is valid for the
  three repository verbs only; and the doc comment gains a sentence for `--unit` — required for
  `name`, rejected for every other verb.
- **Commit:** `feat(cli): parse the name verb, its required --unit, and its flag rejections`

### Card 9: runName, codeForNameResult, and the dispatch order

- **Context:**
  - `quarry/name.go`
  - `quarry/render.go`
  - `quarry/text.go`
  - `quarry/quarry.go`
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `internal/repopath/root.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/cli/cli.go`:

  Add `codeForNameResult(r quarry.NameResult) int`, a named function beside the three existing
  mappers so a table test can be written directly against it. It returns `exitOK` when `r.ID` is
  non-empty; `exitInternal` when `r.Reason` equals `quarry.NameReasonInternal`; `exitNegative` for
  any other non-empty `r.Error`; and `exitInternal` for the shape where neither `ID` nor `Error` is
  set, unreachable and spelled only so a value the facade never produces cannot silently route to a
  zero exit code. The internal check comes before the generic error check, so the ordering is not an
  accident a later edit can reorder.

  Add `runName(req request, stdout, stderr io.Writer) int`, taking no root and no base directory,
  executing these steps in this fixed order:

  1. Call `quarry.Name` with a one-element slice holding `quarry.Declaration{Unit: req.unit, Decl: req.target}`.
     A returned slice whose length is not exactly one is `exitInternal`, named with the count — the
     facade contracts a positional one-to-one mapping, so this is unreachable and is stated so a
     contract change cannot silently produce a zero exit code. Take the single result.
  2. Check `result.Reason == quarry.NameReasonInternal` **before rendering anything**. When it is,
     take `fail`'s path — the compact error envelope on stdout, the same sentence on stderr, exit 3 —
     passing `result.Error` as the message, which already carries the `internal error: ` prefix exit
     3's own rule requires. No payload is written on this route, and the function returns here.
  3. Otherwise render the payload — `quarry.RenderNameText` under `--text`, `quarry.RenderNameJSON`
     otherwise — write it to stdout, and only then compute and return `codeForNameResult(result)`. A
     render error or a failed write is `exitInternal`. The payload-before-code order matters for the
     same reason it does in `runResolve`: a negative answer must be rendered, never replaced by the
     failure envelope.

  State in `runName`'s doc comment why steps 2 and 3 are ordered this way: read the other way round,
  the two rules would emit a success payload followed by an error envelope on the same stdout.

  In `Run`, dispatch `name` immediately after the help check and before `os.Getwd()`, as an early
  return rather than a case in the verb switch below. The verb reads nothing from the filesystem, and
  resolving a root first would make it fail with a no-repository-root message in a directory where
  the answer is perfectly computable — a failure with no relationship to the question asked.

  Rewrite `Run`'s doc comment. It currently states the shared pipeline as `the four steps every verb
  shares`, which is no longer true. The replacement framing is two steps every verb shares — parse,
  and the help branch — then the `name` early return, then two more steps for the three repository
  verbs: read the working directory, and resolve the root. Add `runName`'s own numbered pipeline
  alongside the three existing ones, and state there that the maker's `internal` reason is the one
  per-entry failure the CLI does not render as a payload.

  Update the two `unreachable for every word other than the three verbs` comments on the verb
  switch's `default` — one in `Run`'s doc comment, one in the switch body. The switch still holds
  three cases, because `name` returned earlier, so the correct rewrite names the three repository
  verbs and says `name` never reaches this switch, rather than changing three to four.

  Update the doc comments of `runTOC`, `runResolve` and `runExpand`: each opens by referring back to
  `Run`'s shared four steps, a step count that no longer exists. Each becomes a reference to Run's
  two shared steps and the two repository steps that follow them.

  Update `rootUsageMessage`'s doc comment: its parenthetical says the no-root case is unreachable
  without changing the process working directory, `which these tests never do`. Card 12 adds a test
  that does exactly that, so the sentence must be rewritten to name the working-directory change as
  the way that case is reached.
- **Commit:** `feat(cli): runName, codeForNameResult, and dispatch before root resolution`

### Card 10: the help text

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/cli/usage.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/cli/usage.go`. The whole file is ASCII only and byte-compared in tests: no em dash,
  no typographic quote may enter in this edit.

  The usage block gains a fourth line, in verb order after the three existing ones:
  `quarry name <declaration> --unit <unit> [--text]`. The declaration is the verb's single
  positional target and the unit is a flag, which is what keeps every verb at exactly one positional.

  The flags list gains one row for `--unit <unit>`, marked `name only` in its own description in the
  same style the toc-only rows use, describing it as the glyph unit the declaration will belong to.
  The `--root <path>` row's description gains a `not valid for name` marker, since that flag is no
  longer shared by every verb.

  The exit-code-1 prose enumerating negative-answer causes gains the maker's own: a declaration that
  names no single symbol.

  Update this file's own doc comment. It explains the one-combined-flags-list layout in terms of
  `three per-verb sections` and `the three shared flags`; both counts change, and `--root` is no
  longer shared by every verb, so the second phrase needs rewording rather than renumbering.
- **Commit:** `docs(cli): usage text and flags list for the name verb`

### Card 11: parseArgs table rows

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
- **Edits:**
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/cli/flags_test.go`:

  Update the two existing expected-message rows carrying
  `"no verb given; expected: toc, resolve, or expand"` to the new four-verb sentence.

  Rename `TestParseArgs_ThreeVerbGate` to `TestParseArgs_FourVerbGate` and add a `name` row to its
  table: its name and its table both encode the count, so both change in one edit. The `name` row
  must supply `--unit`, since the verb requires it.

  Add usage-error rows: `--unit` rejected for each of `toc`, `resolve` and `expand`, with the
  per-verb message shape; `--root` rejected for `name`; `--depth`, `--symbols` and `--no-symbols`
  each rejected for `name`; `--unit` given with no value producing the requires-a-value message;
  `name` invoked with no `--unit` producing `"--unit is required for name"`; and `name` with zero and
  with two positional targets producing the existing exactly-one-target message.

  Add accepted-shape rows: `name` with `--unit` and one target lands the unit and the target in the
  request unchanged, in both the space-separated and the `--unit=value` equals forms; and `--text` is
  accepted for `name`.
- **Commit:** `test(cli): parseArgs rows for the name verb and its flag gates`

### Card 12: the verb's behaviour tests

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `internal/cli/cli_test.go`
  - `internal/cli/glyph5_test.go`
  - `internal/cli/scratchtree_test.go`
  - `quarry/quarry.go`
  - `quarry/name.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/name_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/cli/name_test.go`, in the in-process `Run`-with-buffers style the existing tests
  in this package use. Cover:

  `codeForNameResult` as a direct table test, like the three existing mappers: an id set gives 0; the
  internal reason gives 3; any other error gives 1; the neither-id-nor-error shape gives 3.

  The internal-reason bytes: a result carrying that reason takes the compact error envelope on
  stdout, the same sentence on stderr, exit 3, and writes no payload — asserted as bytes, not only as
  an exit code, since the divergence between the facade and the CLI is exactly a bytes difference.
  Construct the result directly and drive the renderer and mapper, since the facade cannot be made to
  produce the internal reason while Go is always wired.

  The no-repository-root proof, asserting both halves in one place with `t.Chdir("/")`: from the
  filesystem root, `quarry name --unit u/v "func F() error"` exits 0 with the expected id on stdout,
  while `quarry toc .` exits 2 with the no-repository-root usage sentence. One without the other
  proves nothing — the second half is what shows the first is not passing for some unrelated reason.
  Use `t.Chdir`, which restores the working directory itself and fails a parallel test rather than
  corrupting one; do not mark this test parallel. The test reads nothing at the filesystem root and
  writes nothing anywhere, so the never-write-to-a-system-directory rule is untouched.

  The multi-line head in both views, as one test: a declaration head spanning lines — an ungrouped
  var with a composite literal — run under `--text` produces stdout containing exactly one newline,
  at the end, while the same invocation without `--text` produces JSON whose `target` value carries
  its newlines intact.

  Help-text assertions: `usageText` contains the `name` usage line, the `--unit` flag row, and the
  `--root` not-valid-for-name marker, and remains ASCII only — assert every byte is below 0x80, which
  is what keeps an em dash or a typographic quote from entering later.
- **Commit:** `test(cli): name verb behaviour, the no-root proof, and the multi-line views`

### Card 13: the name goldens

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/after_test.go`
  - `internal/engine/name.go`
  - `quarry/name.go`
- **Edits:**
  - `internal/cli/loomyard_test.go`
- **Creates:**
  - `internal/cli/name_golden_test.go`
  - `internal/cli/testdata/name/method.json`
  - `internal/cli/testdata/name/method.txt`
  - `internal/cli/testdata/name/function.json`
  - `internal/cli/testdata/name/function.txt`
  - `internal/cli/testdata/name/type.json`
  - `internal/cli/testdata/name/type.txt`
  - `internal/cli/testdata/name/unknown-receiver.json`
  - `internal/cli/testdata/name/unknown-receiver.txt`
  - `internal/cli/testdata/name/malformed.json`
  - `internal/cli/testdata/name/malformed.txt`
  - `internal/cli/testdata/name/multi-symbol.json`
  - `internal/cli/testdata/name/multi-symbol.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/cli/name_golden_test.go` holding two things.

  `compareNameGolden(t *testing.T, name, got string)`, reading `testdata/name/<name>` relative to the
  package directory: a byte-for-byte comparison, or a write under the package's existing update flag,
  creating the directory when it does not yet exist. It is shaped exactly like `compareAfterGolden`
  in `internal/cli/after_test.go` and deliberately does not call it: that helper hard-codes the frozen
  research path, and its own caller is gated on a Loomyard checkout, both wrong for a table that must
  run on a machine with no checkout.

  A table test driving one in-process `Run` per case and comparing both views, with each case
  carrying its own expected exit code in the table rather than inside a golden file. Six cases, each
  producing two files:

  - a method, `method.json` / `method.txt`, exit 0
  - a free function, `function.json` / `function.txt`, exit 0
  - a type, `type.json` / `type.txt`, exit 0
  - a method on a receiver type that does not exist, `unknown-receiver.json` /
    `unknown-receiver.txt`, exit 0
  - a malformed declaration, `malformed.json` / `malformed.txt`, exit 1
  - a fragment declaring several symbols, `multi-symbol.json` / `multi-symbol.txt`, exit 1

  Each file holds the payload bytes and nothing else — no invocation header — per the overview's
  goldens-are-payload-bytes-only decision. The table needs no environment gate of any kind: the maker
  reads no repository, so these run everywhere.

  Produce the twelve files by running the new table once under the update flag and committing what it
  writes, rather than hand-writing them. A hand-written golden pins invented bytes and passes forever.

  Edit `internal/cli/loomyard_test.go` in the same card: the package's update flag is reused rather
  than added, because a second `flag.Bool` of the same name panics within one binary. Its description
  currently reads
  `regenerate the after/ goldens under docs/research/output-formats/after from the current LADDER_LOOMYARD_REPO checkout`,
  wrong in both halves for these files — they sit elsewhere and need no checkout. Widen the
  description and the accompanying doc comment to name both golden sets and to say that only one of
  them needs a checkout.
- **Commit:** `test(cli): name goldens for both views across the six required cases`

## Batch Tests

`verify: go test ./internal/cli/` runs this package's whole suite. No `-run` narrowing: five of the
six cards edit files the existing suite already covers — the flag parser, the exit-code mappers, the
help text asserted byte for byte in several places — so running those existing tests is the point,
not an overreach. The Loomyard-gated cases in this package skip on a machine with no checkout, which
is the state this task is implemented on, so the suite stays fast.

Card 13 opens a bounded red window inside this batch, the same shape `internal/cli/after_test.go`
already documents for its own goldens: the table compares against twelve files that the card's own
update run creates. The window closes within the card and never leaves the batch, because the batch's
`verify:` only runs once every card has landed.
