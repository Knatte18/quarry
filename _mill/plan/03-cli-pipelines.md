# Batch: cli-pipelines

```yaml
task: "Facade + CLI, resolve + expand (T5b)"
batch: "cli-pipelines"
number: 3
cards: 4
verify: go test ./internal/cli/
depends-on: [2]
```

## Batch Scope

This batch gives the two new verbs their request pipelines: the two table-testable exit-code mapping functions, the two pipelines themselves, the dispatch that routes a parsed request to one of three pipelines, and the package documentation for the widened surface. It is one batch because all four cards edit one file and its one test file, and because the mapping functions, the pipelines and the dispatch cannot be reviewed apart from each other — the step that decides an exit code is the step that decides the message, and this batch is where that pairing is written for the two new verbs.

After this batch both verbs work end to end. The remaining batch adds the committed evidence and the machine-independent status gate.

Batch-local decisions beyond the overview's Shared Decisions:

- The existing table-of-contents pipeline is extracted from `Run` into its own named function without one behavioural change, so `Run` becomes a short dispatch over three named pipelines rather than one long body with two more grafted onto it. Every existing assertion about that pipeline must keep passing unchanged, which is this batch's proof the extraction changed nothing.
- Each verb gets its own named mapping function rather than one function with a verb argument, mirroring the existing mapping function's own stated rationale: a table from inputs to a code, testable directly.
- The two new pipelines perform no stat. For the resolve verb "the target does not exist" is the engine's own answer with a payload, and pre-empting it with the failure path would destroy exactly that answer; the expand verb takes a glyph only and does no path work at all.

## Cards

### Card 10: two table-testable exit-code mappings

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/expand.go`
  - `glyph/errors.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add three mapping functions to `internal/cli/cli.go`, beside the existing `codeForTOCError` and following its stated rationale — a named function rather than an inline chain at the call site, precisely so a table test can be written directly against it. Leave `codeForTOCError` itself unchanged.

  ```go
  func codeForResolveResult(r quarry.ResolveResult) int
  func codeForExpandAnswer(a quarry.ExpandAnswer) int
  func codeForExpandError(err error) int
  ```

  `codeForResolveResult` switches on the result's status: `quarry.StatusFound` and `quarry.StatusMultipart` return `exitOK`; `quarry.StatusNotFound` and `quarry.StatusAmbiguous` return `exitNegative`; the empty status returns `exitNegative`, because an empty status means a pre-resolution rejection carried by the error field; the default returns `exitInternal`. The default is unreachable — the status vocabulary is closed — and exists so a value the engine never produces cannot silently route to a zero exit code. Say that in the doc comment.

  `codeForExpandAnswer` switches the same way over the narrower vocabulary the expand answer admits: `quarry.StatusFound` returns `exitOK`; `quarry.StatusNotFound` and `quarry.StatusAmbiguous` return `exitNegative`; the default returns `exitInternal`, unreachable for the same reason.

  `codeForExpandError` maps the error the facade's expand method returns. A nil error returns `exitOK`. When `errors.As(err, &notType)` against `*quarry.NotATypeError` succeeds, return `exitNegative`. When `errors.As(err, &parseErr)` against `*glyph.ParseError` succeeds, return `exitNegative`. Anything else returns `exitInternal` — which is what routes the missing-head-span invariant failure, returned by the engine as a plain formatted error naming an invariant violation in the walk, to exit 3 with no message parsing anywhere.

  This card adds one import to `internal/cli/cli.go`: the module's public `glyph` package, needed to name the concrete parse-error type. That package is public, pure Go, has no dependencies and no cgo, so the import costs nothing. Do not add a facade alias for the parse error or its reason type: aliasing a type that is already reachable would be a second spelling of the grammar's own error type.

  In `internal/cli/cli_test.go`, add one table test per new function, beside the existing `TestCodeForTOCError`, covering every row of the overview's exit-code table that these functions decide: for the resolve mapping, all four statuses plus the empty-status rejection; for the expand answer mapping, all three statuses it admits; for the expand error mapping, nil, a not-a-type value, a wrapped not-a-type value, a parse-error value, a wrapped parse-error value, and a plain formatted error standing in for the invariant failure. The wrapped cases exist to pin that each mapping reaches its type through `errors.As` and not through a direct type assertion.
- **Commit:** `feat(cli): add the resolve and expand exit-code mappings`

### Card 11: the resolve pipeline, and Run becomes a dispatch

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/target.go`
  - `internal/cli/root.go`
  - `internal/engine/resolve.go`
  - `quarry/repo.go`
  - `quarry/render.go`
  - `quarry/text.go`
  - `internal/cli/scratchtree_test.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Restructure `Run` in `internal/cli/cli.go` into a dispatch, and add the resolve pipeline.

  Keep in `Run` itself, unchanged in order and behaviour, the four steps every verb shares: parse the arguments; on a help request write the usage text to stdout and return `exitOK`; read the working directory; resolve the repository root. Keep the existing base-directory rule after root resolution — the base a relative target is interpreted against is the root when the root flag was given and the working directory otherwise.

  Extract everything after that point into:

  ```go
  func runTOC(req request, root, base string, stdout, stderr io.Writer) int
  ```

  moving the existing body verbatim, with no behavioural change: the same path conversion, the same stat, the same open, the same query, the same mapping through `codeForTOCError`, the same two render branches, the same messages. Then have `Run` switch on `req.verb` and call `runTOC` or `runResolve`. This card's switch carries those two arms only — the expand arm and its own differently-shaped signature are added by the next card, so this card's commit compiles on its own. The switch's default returns `exitInternal` with an internal-error message rather than falling through to a zero exit code; that default is what the expand verb reaches until the next card lands, and it is unreachable for every other word because the parser already rejects them.

  Add:

  ```go
  func runResolve(req request, root, base string, stdout, stderr io.Writer) int
  ```

  executing these steps in this fixed order:

  1. Classify the target with `strings.Contains(req.target, "#")`. A glyph target is passed to the facade verbatim — no path conversion, no rebasing, no stat — because a glyph's unit is repository-relative by the grammar's own definition and path arithmetic on it would corrupt it. A path target is converted with `repoRelPath`, the arithmetic-only helper, and the converted form is passed through unconditionally, including one that begins with `..`. Only `repoRelPath` erroring is a failure here, and it returns `exitNegative` through the failure path with the message `target outside repository: ` followed by the target as given — the same message and code the table-of-contents verb already emits for the same condition.
  2. Open the repository. A failure is `exitInternal` with the internal-error prefix.
  3. Call the facade's resolve method with a one-element slice holding the target from step 1. A non-nil error is `exitInternal` with the internal-error prefix: an engine read failure is not an answer about a glyph.
  4. A returned slice whose length is not exactly one is `exitInternal` with an internal-error message naming the count. The facade contracts a positional one-to-one mapping, so this is unreachable and is stated so a contract change cannot silently produce a zero exit code.
  5. Render the single result: the resolve text renderer under the text flag, the resolve JSON renderer otherwise. A render error or a failed write of its bytes to stdout is `exitInternal` with the internal-error prefix.
  6. Return `codeForResolveResult` of that result.

  Steps 5 and 6 are in that order deliberately: the payload is written before the code is computed, so a negative answer is rendered rather than replaced by the failure envelope.

  Extend `Run`'s doc comment: keep the existing numbered pipeline description for the table-of-contents verb, now describing `runTOC`, and add the resolve verb's own numbered steps as above, stating that it performs no stat and why — the target not existing is the engine's own answer with a payload, and pre-empting it would destroy it.

  In `internal/cli/cli_test.go`, extend the fixture the pipeline tests share so it also carries a package declaring one free function and one type with a method, so the resolve verb has something spellable to answer about. Add no build-tag-duplicated declaration to this fixture: two declarations of one name resolve to the ambiguous status, which would contradict this card's own first case, and the ambiguous status is covered by the self-built tree in batch 4 instead. Add end-to-end cases running the resolve verb with explicit root flags and buffer sinks, asserting exit code, stdout bytes and stderr bytes together:

  - a glyph naming the free function: exit 0, a payload whose status is found, empty stderr;
  - a glyph whose unit exists but whose member does not: exit 1, a payload whose status is not-found and whose unit is found, empty stderr;
  - a glyph whose unit does not exist: exit 1, a payload whose status is not-found and whose unit is not-found;
  - a glyph the grammar rejects: exit 1, a payload carrying an error string and a reason word, empty stderr — not the failure envelope;
  - a repository-relative path naming a directory: exit 0, a payload whose status is found and which carries a directory answer;
  - a repository-relative path naming a file: exit 0, a payload whose status is found and whose directory answer is that file's enclosing directory carrying exactly that one file entry. This case pins the claim that a file path target is answered and rendered in the directory form with no file-versus-directory flag plumbed through, on every machine rather than only where the pinned checkout exists;
  - a path that does not exist: exit 1, a payload whose status is not-found and which carries no unit key;
  - a path escaping the root: exit 1, a payload carrying the engine's own error string, emitted verbatim, and empty stderr;
  - the text flag over the found glyph, over the found directory path, and over the found file path, asserting the exact rendered bytes;
  - a path target given relative to a subdirectory, asserting the payload's target field echoes the repository-relative form, together with a glyph target asserting its target field echoes the argument verbatim — cover both in one test so the asymmetry is pinned rather than discovered.

  Every existing assertion in this file must keep passing unchanged; that is the extraction's proof.
- **Commit:** `feat(cli): add the resolve pipeline and dispatch Run over three verbs`

### Card 12: the expand pipeline

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/root.go`
  - `internal/engine/expand.go`
  - `glyph/errors.go`
  - `quarry/repo.go`
  - `quarry/render.go`
  - `quarry/text.go`
  - `internal/cli/scratchtree_test.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add to `internal/cli/cli.go`:

  ```go
  func runExpand(req request, root string, stdout, stderr io.Writer) int
  ```

  It takes no base directory, because this verb accepts a glyph only and performs no path work at all. Add the third arm to `Run`'s switch in this same card, calling it with that shorter argument list.

  Its steps, in this fixed order:

  1. Open the repository. A failure is `exitInternal` with the internal-error prefix.
  2. Call the facade's expand method with the target verbatim. The parser has already guaranteed the target contains a separator.
  3. On a non-nil error, branch by type through `errors.As`, never by parsing a message:
     - a not-a-type value returns `exitNegative` through the failure path, with usage suppressed, and the message `expand `, the value's identifier field, `: not a type, kind `, and the value's kind field. The sentence is quarry's own, spelled from the value's fields, rather than the error's own text, because the existing rule forbids leaking the engine's package-name prefix through an exit-1 or exit-2 message.
     - a parse-error value returns `exitNegative` through the failure path, with usage suppressed, and the message `expand `, the target as given, `: `, and the value's reason word. The reason word is the only actionable part of the rejection and is the same word the resolve verb puts in its payload's reason key, so dropping it would make the two verbs disagree about the same rejection.
     - anything else returns `exitInternal` with the internal-error prefix, which is where the missing-head-span invariant failure lands.
     Route these three through `codeForExpandError` for the code rather than restating the classification, so the mapping stays the single table-tested source of the code.
  4. On a nil error, render the answer: the expand text renderer under the text flag, the expand JSON renderer otherwise. A render error or a failed write to stdout is `exitInternal` with the internal-error prefix.
  5. Return `codeForExpandAnswer` of the answer.

  Extend `Run`'s doc comment with the expand verb's numbered steps, stating that it performs neither path conversion nor a stat.

  In `internal/cli/cli_test.go`, add end-to-end cases for the expand verb against the same shared fixture, asserting exit code, stdout bytes and stderr bytes together:

  - a glyph naming a type with members: exit 0, a payload whose status is found, carrying a head and its members, empty stderr;
  - a glyph naming a type with no members: exit 0, found, a head and no members key — not a not-found and not an error;
  - a glyph whose unit exists but whose member does not: exit 1, not-found with unit found;
  - a glyph naming a free function: exit 1, the failure envelope on stdout, the exact sentence on stderr, and no usage text on either stream;
  - a target carrying a separator that the grammar still rejects: exit 1, the failure envelope, the message ending in the reason word, and no usage text;
  - a target with no separator: exit 2, the failure envelope on stdout, the exact parser message on stderr followed by the usage text — and, since the parser decides it, no repository need exist for the case;
  - the text flag over the found type, asserting the exact rendered bytes, including the blank line between the head and the members;
  - the text flag over a failure, asserting the failure envelope still goes to stdout as JSON, since that rule applies on every failure path including under the text flag.
- **Commit:** `feat(cli): add the expand pipeline`

### Card 13: describe the widened command-line surface

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/target.go`
- **Edits:**
  - `internal/cli/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Update the package doc comment in `internal/cli/doc.go` so it describes three verbs rather than one. Keep the existing statements — that this package is the whole of the command below the process exit, that it is the only layer with a working directory, and that input is interpreted where the user is while output is always repository-root relative with forward slashes — and add:

  - that the command has three verbs, and which argument class each accepts;
  - that a target containing a `#` is classified as a glyph by this package and handed to the facade verbatim, with no path arithmetic and no stat applied to it, because a glyph's unit is repository-relative by the grammar's own definition;
  - that the failure envelope's `ok` key marks that quarry could not answer, and never that the answer is negative — a negative resolution outcome is a payload carrying a status word;
  - a known contract gap, recorded in the same style the engine records its own: this package classifies the argument as given, but hands the engine the repository-relative form, and the engine classifies that string again. The two disagree in exactly one case — a path target whose repository-relative form acquires a `#` from a directory name — where the caller gets a grammar rejection naming the reason rather than the path answer asked for. Closing it needs the target's class to travel with the target, which means a new engine signature, and that is out of this task's scope. State that a `#` in a directory name is legal but pathological, that no repository quarry is measured against contains one, and that the gap is deliberately not pinned by a test, because a test would pin behaviour this task considers wrong but unfixable at this layer.
- **Commit:** `docs(cli): describe three verbs and record the reclassification gap`

## Batch Tests

`verify: go test ./internal/cli/` runs the whole `internal/cli` test binary, which is the right scope here rather than a narrower one: this batch restructures `Run`, which every pipeline test in the package exercises, so a per-file scope would miss the regressions the extraction could cause. The binary runs in well under a second, and the evidence-golden test in it skips on a machine with no Loomyard checkout, so nothing in this scope is slow or machine-dependent.

The new assertions all live in `internal/cli/cli_test.go`: three mapping tables in card 10, and the two verbs' end-to-end cases in cards 11 and 12 — each asserting exit code, stdout bytes and stderr bytes together, which is what pins the rule that the failure envelope goes to stdout while stderr carries the same sentence.
