# Batch: cli-delta-verb

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'cli-delta-verb'
number: 8
cards: 6
verify: go test ./internal/cli/ -run 'TestParseArgs|TestRun|TestCodeFor'
depends-on: [7]
```

## Batch Scope

This batch adds the fourth verb to the command line: argument parsing for the two new revision
flags, the usage text, the verb's own pipeline, its exit-code mapping, and the two doc comments that
currently state the command has three verbs.
It depends on batch 7 because the pipeline's last step calls both new renderers.

The exit-code contract is the load-bearing part and is deliberately unlike the other three verbs':
this query has no negative answer, because nothing changed is a true answer to what changed rather
than a negative one.
An empty delta, and a batch in which some entries failed, are both a plain success.
An unresolvable revision, a root that is not a repository, and a root that is not that repository's
top-level are all usage errors carrying quarry's own sentence, never git's raw message behind an
internal-error prefix.

Batch-local decision: this verb performs no stat on its target, unlike the table-of-contents verb,
which does its own and returns a negative answer when the target is missing.
A path that does not exist now may well have existed at the from revision, and a deleted directory
is exactly the change this query exists to report; a pathspec matching nothing is a true, empty
answer.

## Cards

### Card 42: parse the delta verb and its two revision flags

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/usage.go`
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/cli/flags_test.go`
- **Edits:**
  - `internal/cli/flags.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add two fields to the parsed request shape: the from revision and the to
  revision, each empty when the flag was absent, exactly as the existing root field already works.
  Reject an explicitly empty value for either flag as a usage error naming the flag, which is what
  makes one empty string mean one thing: an absent to revision is the working tree, and there is no
  second way to spell that.
  Do not add a third field recording whether the flag was given — the git delta method takes the two
  revisions as plain strings with the empty string meaning the working tree, so a
  present-but-empty state would have no way to reach it and no defined behaviour if it did.
  Add the fourth verb to the verb gate and to the two messages that enumerate the accepted verbs.
  Accept the two new flags, valid for this verb only, rejecting each for the other three with the
  same message shape the depth flag already uses, and checked at the point the flag is recognised so
  that rejection takes precedence over the flag's own value validation.
  The depth, symbols and no-symbols flags are already rejected for any verb other than the
  table-of-contents one, so they need no new branch — confirm that rather than adding one.
  Each new flag requires a value, using the same next-value helper the existing value-taking flags
  use, so both the equals form and the space-separated form work and a value containing an equals
  sign survives verbatim.
  A missing from revision is a usage error for this verb, raised in the parser, which stays pure
  over its argument slice: it resolves no path, stats nothing and reads no working directory.
  Keep the exactly-one-target rule intact for every verb, including this one; do not add a per-verb
  exception to the target-count check.
  Do not classify the target further here.
  Correct this file's own stale doc claims in the same commit, since both become false with this
  edit: the parser's doc comment states that the verb gate accepts exactly the three existing verbs,
  and that the text and root flags are valid for all three — name four in both places, and describe
  the two new flags as valid for the new verb only, in the same sentence that already scopes the
  depth, symbols and no-symbols flags to the table-of-contents verb.
- **Commit:** `feat(cli): parse the delta verb and its --from and --to flags`

### Card 43: the usage text

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/cli.go`
  - `internal/cli/cli_test.go`
- **Edits:**
  - `internal/cli/usage.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the fourth verb's own usage line to the usage block, after the three
  existing ones, spelling its required and optional flags in the same bracketed style, and add the
  two new flags to the combined flags list, each marked as valid for this verb only in its own
  description exactly as the existing verb-scoped flags are.
  Keep the text ASCII only — no typographic dash and no typographic quotes — since it is
  byte-compared in tests and must be stable across terminals.
  Leave the exit-codes block unchanged: this verb introduces no new code and reaches the negative
  code only through the target-escapes-the-root rejection every path-taking verb already inherits,
  so the block's existing wording still covers it.
  Do not introduce a flag naming the JSON format; JSON is the default and naming it would imply a
  third format exists.
  Several existing tests compare this constant byte for byte or assert its presence and absence on
  the two output streams; those comparisons are against the constant itself rather than a
  hand-copied string, so they follow this edit without change — confirm that rather than editing
  them.
- **Commit:** `feat(cli): add the delta verb and its flags to the usage text`

### Card 44: the delta verb's pipeline and exit-code mapping

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `internal/repopath/target.go`
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `quarry/render.go`
  - `quarry/text.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the fourth branch to the verb dispatch and the verb's own pipeline function,
  continuing from the shared steps the existing dispatch already performs, and a named mapping
  function from the facade's returned error to an exit code — named rather than inlined, following
  the convention the four existing mapping functions in this file set precisely so a table test can
  be written against it, even though this one is nearly constant.
  Correct this file's own stale doc claims in the same commit, since all three become false with this
  edit: the entry point's doc comment states that it switches on the verb and calls one of the three
  existing pipeline functions, and that its default case is unreachable for every word other than
  the three verbs; and the usage-code constant's own comment scopes the glyph-separator rejection to
  a table-of-contents target, which is now one of two path-taking verbs that raise it.
  The dispatch switch's default case also carries an inline comment repeating the three-verb claim,
  beside the code rather than in a doc comment; correct it too, since it is the one a reader meets
  while editing the switch.
  Add this verb's pipeline to the entry point's numbered per-verb description as well, at the same
  level of detail the other three get, since that comment is where each pipeline's fixed step order
  is stated.
  The pipeline converts the target through the shared repository-relative target helper first,
  exactly as the table-of-contents verb's pipeline does, so one argument cannot mean two things:
  quarry resolves a relative target against the caller's working directory while git would resolve a
  raw pathspec against the root.
  The consequence is that a lone dot means the current directory rather than the repository root
  when run from a subdirectory, identically to the table-of-contents verb, and that helper's two
  existing rejections carry over unchanged — a target escaping the root is the negative code, and a
  target carrying the glyph separator is a usage error with the usage text.
  It then opens the facade and calls the git delta method, and performs **no** stat on the target at
  any point.
  Map the facade's errors: an unresolvable revision, a root that is not a repository, and a root
  that is not that repository's top-level are each a usage error carrying quarry's own sentence,
  spelled from the aliased typed error's own fields through type extraction rather than by parsing
  any message, exactly as the expand verb's pipeline already spells its own two sentences.
  The revision sentence names the revision exactly as given; the top-level sentence names both the
  root and the top-level git reported.
  Any other failure is the internal code carrying the wrapped message whole behind the existing
  internal-error prefix, which is where a git command failing for any other reason lands.
  A computed delta is always the success code — including an empty delta and including a batch in
  which some entries carry an error disposition, since either of those returning a failure code
  would make a complete answer look like a failure to a shell gate.
  Render with the text renderer under the text flag and the JSON renderer otherwise, with a render
  failure or a failed write to standard output being the internal code, as every other pipeline in
  this file already does.
- **Commit:** `feat(cli): add the delta verb's pipeline and exit-code mapping`

### Card 45: correct the package doc's verb count

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
- **Edits:**
  - `internal/cli/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** The package doc states that the command has three verbs and then describes each
  one; add the fourth to that enumeration, stating that it takes a path target and two revisions.
  The same comment states that the table-of-contents verb is the only verb that still takes a path
  and the only one this package converts with the shared target helper before the engine sees it;
  both halves of that claim are false once this verb lands, so correct them to name the two
  path-taking verbs rather than one.
  Leave the classification paragraph's substance intact — the grammar is still the only classifier,
  and no surface in this package tests a target for the separator to decide whether it is a path or
  a glyph — but extend its last sentence so the separator rule is stated as holding for both
  path-taking verbs.
  Leave the failure-envelope paragraph untouched: this verb changes nothing about what that key
  means.
- **Commit:** `docs(cli): describe the fourth verb in the package doc`

### Card 46: argument-parsing tests for the new verb and flags

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestParseArgs_Delta` and `TestParseArgs_DeltaFlagValidity`, following the
  table shape the existing parser tests in this file use and asserting on the parsed request rather
  than on any side effect, since the parser is pure over its argument slice.
  The first covers: both revision flags in the space-separated form and in the equals form; an
  absent to revision leaving its field empty, which is how the working tree is spelled; a value
  containing an equals sign surviving verbatim; a missing from revision rejected as a usage error;
  each revision flag given without a value rejected as a usage error; each revision flag given an
  explicitly empty value in the equals form rejected as a usage error naming the flag, so the empty
  string has exactly one meaning; and zero targets and two targets each rejected by the existing
  exactly-one-target rule with the count named.
  The second covers the validity matrix in both directions: each revision flag rejected for each of
  the other three verbs with a message naming the flag and the verb, and the depth, symbols and
  no-symbols flags each rejected for the new verb with the same message shape.
  Extend the existing three-verb gate test, or add a case to it, so the accepted verb set is
  asserted as four rather than three and an unknown verb is still rejected by name.
  Assert that the help flag still wins over every other complaint when given alongside this verb
  with no target and no revisions.
- **Commit:** `test(cli): assert delta argument parsing and flag validity`

### Card 47: end-to-end exit-code and behaviour tests for the new verb

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
  - `quarry/delta.go`
  - `quarry/quarry.go`
  - `internal/cli/loomyard_test.go`
- **Edits:**
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestRun_Delta`, `TestRun_DeltaTargetResolution` and `TestCodeForDeltaError`,
  each building its own throwaway git repository under a temporary directory and skipping cleanly
  when no git binary is available.
  `TestRun_Delta` covers, as a table over the whole command entry point: a well-formed call
  producing an empty delta, asserted the success code; a batch containing an entry with an error
  disposition, asserted the success code with that entry's error visible in the payload; an
  unresolvable from revision, asserted a usage error whose message names the revision exactly as
  given and is accompanied by the usage text on the error stream, and asserted **not** to carry
  git's own message; a root that is a subdirectory of a repository and a root outside any
  repository, each asserted a usage error carrying quarry's own sentence rather than the internal
  code with git's; a target escaping the root, asserted the negative code; a target carrying the
  glyph separator, asserted a usage error; the text flag producing the text view; and a pathspec
  matching nothing, asserted the success code with an empty delta rather than the negative code.
  Include the case that proves this verb performs no stat: a target naming a path that no longer
  exists but did at the from revision, asserted the success code with that path's symbols in the
  deleted array.
  `TestRun_DeltaTargetResolution` pins the shared rule: a lone dot given to this verb from a
  subdirectory scopes to that subdirectory, and produces the same scope the table-of-contents verb's
  lone dot produces from the same directory — the two verbs resolve one argument the same way or the
  command line has two meanings for it.
  `TestCodeForDeltaError` is a table from a returned error to the code, mirroring the existing
  mapping-function table test for the table-of-contents verb, and must include a wrapped error to
  assert the identity check survives wrapping.
- **Commit:** `test(cli): assert the delta verb's exit codes, target resolution and no-stat rule`

## Batch Tests

`verify:` runs `go test ./internal/cli/ -run 'TestParseArgs|TestRun|TestCodeFor'`.
The scope is deliberately wider than this batch's own additions and covers every existing parser
test, every existing pipeline test and all four existing mapping-function tables in
`internal/cli/flags_test.go` and `internal/cli/cli_test.go`.
That width is the point: this batch edits the shared argument parser, the shared usage constant and
the shared dispatch, so the tests most likely to catch a regression here are the ones already
pinning the other three verbs' behaviour through those same three surfaces — several of them
compare the usage constant byte for byte, and the verb gate is one switch all four verbs pass
through.
The one existing test function deliberately left out of the pattern is the committed-goldens table
in `internal/cli/after_test.go`, which needs an external checkout and skips on most machines; it
still runs under the repository-wide done gate.
