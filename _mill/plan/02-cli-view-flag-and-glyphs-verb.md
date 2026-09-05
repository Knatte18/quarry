# Batch: cli-view-flag-and-glyphs-verb

```yaml
task: 'P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)'
batch: 'cli-view-flag-and-glyphs-verb'
number: 2
cards: 8
verify: LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ ./quarry/
depends-on: [1]
```

## Batch Scope

This batch delivers the whole CLI surface of the glyphs view: the `--view` flag on `toc`, the
symbols default the view implies, the `glyphs` verb as an argv rewrite to a frozen `toc`
expansion with its own pre-rewrite validation so every rejection names the verb the caller typed,
the one render branch in `runTOC`, the help text, and the two tests that are this task's central
enforcement mechanism — the byte-identity pairs and the preset-single-source drift check. It is
one batch because five of its eight cards land in `internal/cli/flags.go` and its test file, and
because the byte-identity assertion is meaningless until every one of them has landed.

The batch opens with a card that has no diff at all: it creates the pinned Loomyard checkout at
`.scratch/loomyard-pin` that this batch's own `verify:` command and batch 3's goldens both depend
on. Without it, every case that needs real repository bytes skips and the batch reports green
having asserted nothing.

What batch 3 consumes from here: a working `quarry glyphs <target>` and `quarry toc --view glyphs`
whose output can be captured as golden files, and the `afterGoldenCase` table shape unchanged.

Batch-local decisions beyond `## Shared Decisions` in the overview:

- **The verb enumeration's word order is `toc, glyphs, resolve, expand, or name`.** `glyphs` sits
  next to `toc` because it is a preset over it, not in alphabetical or append order. The two
  message literals and the two test rows pinning them must all agree on this one spelling.
- **The two flag rules that cannot be checked at the point the flag is read are checked after the
  flag loop instead.** Both the glyphs-view symbols default and the `--no-symbols` rejection depend
  on the view, and the view may be given after the symbols flag on the command line, so an
  at-the-flag check would be order-dependent. See card 8.

## Cards

### Card 6: pin a Loomyard checkout the golden and identity tests can read

- **Context:**
  - `internal/cli/loomyard_test.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** This card produces no diff. Its job is to put a Loomyard checkout pinned at
  commit `72c23d9` — the commit `loomyardPin` names and every committed golden was taken from — at
  `.scratch/loomyard-pin` relative to the repository root, so that this batch's `verify:` command
  and batch 3's both resolve `LADDER_LOOMYARD_REPO` to a real checkout instead of skipping.

  This machine has a Loomyard clone at the absolute path
  `/home/knatte/Code/loomyard/wts/loomyard`, whose HEAD is *not* the pin but which does contain
  commit `72c23d9` as a reachable object. Clone it locally and detach onto the pin:
  `git clone /home/knatte/Code/loomyard/wts/loomyard .scratch/loomyard-pin` followed by
  `git -C .scratch/loomyard-pin checkout 72c23d9`. A clone reads the source repository and mutates
  nothing in it; do not create a git worktree in the source repository, and run no git command that
  writes to it.

  Then confirm the gate is satisfied, because `loomyardRepo` skips rather than fails when it is
  not: `git -C .scratch/loomyard-pin rev-parse HEAD` must print a hash beginning `72c23d9`, and
  `LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ -run TestAfterGoldens
  -v` must report the existing golden cases as run, not skipped. If the clone step cannot be
  completed on this machine, stop and report it rather than continuing — every remaining card in
  this batch and in batch 3 is verified through this checkout, and continuing without it produces a
  plan that reports green while asserting nothing.

  If `.scratch/loomyard-pin` already exists and already resolves to the pin, this card is a no-op
  beyond the two confirmation commands. `.scratch/` is gitignored at the repository root, so
  nothing here is ever committed — which is why this card's `Commit:` is `none`.
- **Commit:** none

### Card 7: the --view flag on toc

- **Context:**
  - `quarry/quarry.go`
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/cli/flags.go`
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a `view string` field to `request` in `internal/cli/flags.go`, documented in
  that struct's own doc comment alongside the existing `root`, `symbols` and `unit` sentences: it
  holds the `--view` value exactly as given, and is empty when the flag was absent, which means the
  same thing as `"full"`.

  Add a `case "--view":` to `parseArgs`'s flag switch, placed with the other toc-only flags, doing
  four things in this order: reject with the existing `"%s is not valid for %s"` message when
  `verb` is not `"toc"`, checked before the value is read so rejection takes precedence over value
  validation, exactly as the `--depth` case already does; read the value through the existing
  `nextValue` closure, so both the `--view glyphs` and `--view=glyphs` forms work with no new
  parsing; reject a missing value with the existing `"%s requires a value"` message; and reject any
  value other than `"full"` or `"glyphs"` with the literal message
  `--view must be "full" or "glyphs", got %q`, formatted with the value. On success assign the
  value to `req.view`.

  Extend `parseArgs`'s own doc comment where it enumerates which flags are valid for which verb, so
  `--view` is named there with `--depth`, `--symbols` and `--no-symbols` as toc-only, and state
  that the vocabulary is closed at two values and that an absent flag means `full` — an unknown
  value being a usage error rather than a silent fallback to the complete answer is the whole point
  of the closed set.

  Add rows to `internal/cli/flags_test.go`. A new `TestParseArgs_View` table asserting: `--view
  full` and `--view glyphs` set `req.view` to that value; `--view=glyphs` does the same; a viewless
  `toc` leaves `req.view` empty; `--view` with no value is a usage error; `--view bogus` is a usage
  error whose message is exactly `--view must be "full" or "glyphs", got "bogus"`. Add rows to the
  existing `TestParseArgs_UsageErrors` table asserting `--view` is rejected for each of `resolve`,
  `expand` and `name` with `--view is not valid for <verb>`, matching the shape of the `--depth`
  rows already there.
- **Commit:** `feat(cli): --view flag on toc with a closed full|glyphs vocabulary`

### Card 8: the glyphs view implies symbols

- **Context:**
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/flags.go`
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `parseArgs`, immediately after the flag loop ends and before the existing
  target-count check, add the two view-dependent symbols rules the discussion's
  `view-glyphs-implies-symbols` decision fixes:

  - when `req.view` is `"glyphs"` and `req.symbols` is non-nil and points to `false`, return the
    usage error whose message is exactly `--no-symbols is not valid with --view glyphs`;
  - when `req.view` is `"glyphs"` and `req.symbols` is still nil, set it to a pointer to `true`.

  Both checks must live after the loop rather than inside the `--symbols`/`--no-symbols`/`--view`
  cases, and the code must say why in a comment: the two flags can be given in either order, so a
  check at the point either one is read would accept `--no-symbols --view glyphs` and reject
  `--view glyphs --no-symbols`, or vice versa. After the loop the whole invocation is known and the
  rule is order-independent.

  The reasoning belongs in `parseArgs`'s doc comment too: the engine's own per-target default for
  `TOCOptions.Symbols` is nil meaning false for a directory target, so `toc --view glyphs <dir>`
  with no symbols flag would answer with an empty symbol list — a view whose entire content is
  symbols returning none, indistinguishable at the consumer from a directory that declares nothing.
  `--no-symbols` is rejected rather than honoured for the same reason: it asks for a view of
  nothing. State that this is a default, not a filter — `--view full --no-symbols` and a viewless
  `--no-symbols` are untouched, so the complete answer stays one flag away.

  Add a `TestParseArgs_ViewGlyphsSymbols` table to `internal/cli/flags_test.go` asserting:
  `toc --view glyphs t` yields a non-nil `symbols` pointing to true; `toc --view glyphs --symbols t`
  is unchanged and also true; `toc --symbols --view glyphs t` likewise, proving order-independence;
  `toc --view glyphs --no-symbols t` and `toc --no-symbols --view glyphs t` are both usage errors
  with the message `--no-symbols is not valid with --view glyphs`; and — the cases that prove the
  default is scoped to this view — `toc --view full --no-symbols t` and `toc --no-symbols t` both
  still yield a non-nil `symbols` pointing to false, and `toc --view full t` and viewless `toc t`
  both still yield a nil `symbols`.
- **Commit:** `feat(cli): --view glyphs defaults symbols on and rejects --no-symbols`

### Card 9: the glyphs verb, its pre-scan, and the argv rewrite

- **Context:**
  - `internal/cli/cli.go`
  - `quarry/quarry.go`
  - `docs/rewrite-plan.md`
- **Edits:**
  - `internal/cli/flags.go`
  - `internal/cli/flags_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add to `internal/cli/flags.go` a package-level `var glyphsPreset =
  []string{"--view", "glyphs", "--depth", "all", "--symbols"}`, with a doc comment stating that it
  is the frozen expansion `quarry glyphs <target>` rewrites to, that it is a `var` only because Go
  has no constant slice, that no code may append into it — the rewrite below builds a fresh slice,
  and appending into this one would let a second invocation in the same process see the first
  invocation's target — and that its exact tokens are spelled in three other places that must
  change with it: `usageText`, `docs/rewrite-plan.md` section 5, and the byte-identity test.

  Add `glyphs` to the verb gate: the four-way string comparison that today rejects any verb other
  than `toc`, `resolve`, `expand` and `name` accepts `glyphs` as a fifth. Change both
  `"no verb given; expected: toc, resolve, expand, or name"` literals — the one for an empty
  argument slice and the one for a first argument beginning with a hyphen — to
  `no verb given; expected: toc, glyphs, resolve, expand, or name`. Leave the `"unknown verb: %s"`
  message exactly as it is: it enumerates nothing, so it has nothing to gain a verb.

  Immediately after the verb gate, before `req` is built and before any flag is read, add the
  `glyphs` branch. It runs a pre-scan over the post-verb tokens whose only job is to make every
  rejection name `glyphs`, and only then rewrites and re-parses. The pre-scan walks the tokens with
  the same index-based loop shape the main loop uses, splitting each flag token with
  `strings.Cut(tok, "=")` on the first `=` exactly as the main loop does, and:
  - a token not beginning with `-` is counted as a target;
  - `--view`, `--depth`, `--symbols`, `--no-symbols` and `--unit` are each rejected with the
    existing `"%s is not valid for %s"` format, with the flag name as given and the literal verb
    `glyphs` — so `quarry glyphs --depth 1 x` says `--depth is not valid for glyphs`, never `toc`;
  - `--text` is accepted and takes no value;
  - `--root` is accepted and consumes its value the same way the main loop's `nextValue` closure
    does: the part after `=` when the token carried one, otherwise the following token, which the
    loop then skips. This is what keeps `glyphs --root <path> <target>` counted as one target
    rather than two;
  - any other flag token is rejected with the existing verb-free `"unknown flag: %s"` message,
    formatted with the whole token as given, matching the main loop's own use of `tok` rather than
    `name`;
  - after the walk, a target count other than one is rejected with
    `glyphs takes exactly one target, got %d`.

  Only when the pre-scan passes, build the rewritten argument slice — the literal `"toc"`, then
  `glyphsPreset`'s tokens, then the original post-verb tokens — into a freshly allocated slice that
  shares no backing array with `glyphsPreset`, and return `parseArgs` of it directly. Recursion
  depth is one and bounded by construction, because the rewritten slice's verb is `toc` and `toc`
  has no preset; say so in a comment. Add a sentence, in the style of the unreachable-default
  branches in `internal/cli/cli.go`, stating that the re-parse cannot produce a usage error because
  the pre-scan has already rejected everything it could reject, and that if it ever did, its
  message would name `toc`.

  Extend `parseArgs`'s doc comment: the verb gate now accepts five verbs; `glyphs` is a frozen
  preset validated against its own flag rules and then rewritten to its `toc` expansion, so nothing
  downstream — not the dispatch switch, not `runTOC`, not the renderers — can tell a `glyphs`
  invocation from its expansion; and `req.verb` is `"toc"` after the rewrite, which is why the
  dispatch switch needs no new case.

  In `internal/cli/flags_test.go`, change the two rows pinning the enumerating messages — the
  `missing-verb` row and the `first-arg-is-flag` row — to the new five-verb spelling, and leave the
  `unknown-verb` row exactly as it is. Add a `TestParseArgs_Glyphs` table asserting: `glyphs x`
  parses; `glyphs --text x` sets `text`; `glyphs --root /repo x` and `glyphs --root=/repo x` both
  set `root` and leave one target; each of `--view`, `--depth`, `--symbols`, `--no-symbols` and
  `--unit` is rejected with a message naming `glyphs` and never `toc`; an unknown flag yields
  `unknown flag: --nope`; zero targets yields `glyphs takes exactly one target, got 0` and two
  targets `... got 2`; and `glyphs --help x` still returns the help request, since the help scan
  runs before the verb gate. Add the load-bearing case as its own named test: `parseArgs([]string{"glyphs", "x"})`
  returns a `request` deep-equal to `parseArgs([]string{"toc", "--view", "glyphs", "--depth", "all", "--symbols", "x"})`,
  compared with `reflect.DeepEqual` after dereferencing or separately comparing the `symbols`
  pointer, since two distinct pointers to `true` are not `DeepEqual`-equal as pointers.
- **Commit:** `feat(cli): the glyphs verb as a frozen argv rewrite to toc`

### Card 10: runTOC renders the glyphs view

- **Context:**
  - `internal/cli/flags.go`
  - `quarry/view.go`
  - `quarry/render.go`
  - `quarry/text.go`
  - `internal/repopath/target.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `runTOC` in `internal/cli/cli.go`, after the `codeForTOCError` block has
  returned the answer without error and before the existing `if req.text` branch, add one branch
  for the glyphs view: when `req.view` is `"glyphs"`, project the answer with
  `quarry.GlyphView(rel, answer)` — `rel`, the repository-relative target the existing
  `repopath.RepoRelTarget` call already produced, not `req.target` — and render it with
  `quarry.RenderGlyphsText` under `--text` and `quarry.RenderGlyphsJSON` otherwise, writing to
  stdout and returning `exitOK`. A render error, or a failed write of its bytes, is `exitInternal`
  through the same `fail` helper every other step uses, so the failure envelope is written the same
  way here as everywhere else.

  Add a sentence in this function stating that the existing `targetIsFile` bool, which the two
  viewless renderers need, is deliberately not consulted by the glyphs branch: that view renders
  identically for a file and a directory target, because it has no directory or file line to choose
  a form for. Without the sentence the asymmetry reads as an oversight, since every other text
  rendering in this package needs the flag.

  Update `Run`'s numbered pipeline doc comment, which is this package's real specification: the
  dispatch switch's introduction says the command has three repository verbs reached from it, and
  that is still true — `glyphs` never reaches the switch, because `parseArgs` has already rewritten
  it to `toc` — so say that explicitly rather than leaving a reader to infer it. Extend runTOC's
  own numbered step 5 from "quarry.RenderText under --text, quarry.RenderJSON otherwise" to name
  the glyphs branch and the two renderers it selects, and state the condition it branches on.
- **Commit:** `feat(cli): runTOC renders the glyphs view under --view glyphs`

### Card 11: the help text and the package doc comment

- **Context:**
  - `internal/cli/flags.go`
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/cli/usage.go`
  - `internal/cli/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `internal/cli/usage.go`, extend `usageText` under its own stated rules —
  ASCII only, no em dash, no typographic quotes; one combined flag list with per-verb validity in
  each flag's description; per-verb shapes in the usage block above:

  - the usage block's `toc` line gains `[--view <name>]`, placed before `[--depth N|all]`;
  - the usage block gains a `glyphs` line immediately after the `toc` line, spelling
    `quarry glyphs <target> [--text] [--root <path>]` — the three query flags are absent from it
    because the verb rejects them;
  - the flag list gains a `--view <name>` entry, placed immediately before `--depth`, whose
    description names it toc-only and spells the closed vocabulary and the default: `full`, the
    default and the complete answer, or `glyphs`, the flat one-line-per-symbol projection. Keep the
    description column's existing start position — the other entries align their descriptions at a
    fixed column and `--view <name>` is short enough to keep it;
  - after the flag list and before the exit-codes block, add a short block spelling the preset
    expansion literally, so `--help` alone answers "what does glyphs do":
    `quarry glyphs <target>` is `quarry toc --view glyphs --depth all --symbols <target>`, with a
    following sentence saying the expansion is frozen and the three query flags are therefore not
    accepted on `glyphs`.

  Extend `usageText`'s own doc comment where it explains why the flag list is combined, so the new
  preset block is accounted for rather than looking like an exception to the two-block rule: it is
  neither a verb shape nor a flag, it is the one thing a preset verb needs stated that neither
  block can carry.

  In `internal/cli/doc.go`, the package comment says the command has four verbs and describes each.
  Update it to five, describing `glyphs` as taking the same repository-relative or cwd-relative path
  `toc` takes, and stating that it is not a fifth pipeline: `parseArgs` rewrites it to a frozen
  `toc` expansion, so it reaches exactly the same parsing, the same dispatch and the same
  renderers, and nothing below the parser knows the verb exists. That last point matters for the
  paragraph immediately following, which says `toc` is the only verb this package converts with
  `internal/repopath/` before the engine sees the target — still true, and now true for a second
  spelling of the same verb.
- **Commit:** `docs(cli): usageText and the package comment gain the glyphs verb and --view`

### Card 12: the byte-identity test

- **Context:**
  - `internal/cli/loomyard_test.go`
  - `internal/cli/after_test.go`
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `internal/cli/glyphs_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/glyphs_test.go` with a file-level comment saying what the
  file is for and why it is not a golden file: this is the task's central enforcement mechanism,
  and a failure here means the `glyphs` preset has grown a second code path rather than remaining
  an argv rewrite.

  Write `TestGlyphsIsByteIdenticalToItsExpansion`, gated on `loomyardRepo(t)` exactly as
  `TestAfterGoldens` is. It covers four pairs — the cross product of a file target and a directory
  target with JSON and `--text`. Use `internal/logger` as the directory target and
  `internal/logger/logger.go` as the file target, the same two targets the existing golden table
  uses, so the fixture set does not grow. For each pair, run `Run` twice into separate
  `bytes.Buffer` pairs: once with the `glyphs` argv — the verb, `--root`, the checkout, the
  `--text` flag when the pair calls for it, and the target — and once with the expansion argv — the
  verb `toc`, `--root`, the checkout, `--view glyphs`, `--depth all`, `--symbols`, the same
  `--text` flag when applicable, and the same target. Require all three of stdout, stderr and the
  returned exit code to be identical between the two runs, and additionally require the exit code
  to be `exitOK` and stderr to be empty, so a pair that fails identically in both runs cannot pass
  as a match.

  Spell the expansion's flag tokens literally in the test rather than reading `glyphsPreset`. The
  test's whole purpose is to catch a change to that variable that is not matched by a change to the
  documented expansion; reading the variable would make it pass by construction. Say so in the
  test's doc comment, and name the three other places the same tokens are spelled — `usageText`,
  `docs/rewrite-plan.md` section 5, and `glyphsPreset`'s own doc comment — so a reader knows what
  else must move when this test is deliberately changed.
- **Commit:** `test(cli): glyphs is byte-identical to its documented toc expansion`

### Card 13: the preset-single-source drift test

- **Context:**
  - `internal/cli/flags.go`
  - `quarry/repo.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/glyphs_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestGlyphsPresetMatchesFacadeOptions` to `internal/cli/glyphs_test.go`. It
  needs no repository and no checkout gate: it parses the CLI's own preset and compares the parse
  against the facade's frozen options.

  Build the argv the rewrite builds — the literal `"toc"`, then `glyphsPreset`'s tokens, then a
  placeholder target — pass it through `parseArgs`, and assert the resulting `request`'s `depth`
  equals `quarry.GlyphsOptions().Depth` and that its `symbols` pointer is non-nil and dereferences
  to the same value `quarry.GlyphsOptions().Symbols` dereferences to. Compare the pointed-to
  values, never the pointers: the two are independently allocated and can never be equal as
  pointers. Also assert `request.view` is `"glyphs"`, which is the third field the preset fixes and
  the one with no counterpart in `TOCOptions` — state in the test's doc comment that the view is
  deliberately not part of the facade's options, because the facade returns the projected answer
  type directly and has no view to select.

  Unlike card 12's identity test, this one reads `glyphsPreset` rather than spelling the tokens: the
  property it asserts is that the two sides of the CLI/facade boundary agree, so both sides must be
  read from their real sources. Say so in the test's doc comment, alongside a sentence explaining
  why the assertion cannot live in package `quarry` — `internal/cli/` imports `quarry`, so the
  reverse import would be a cycle, which is why `GlyphsOptions` is exported at all.
- **Commit:** `test(cli): the glyphs preset and the facade's options do not drift`

## Batch Tests

`verify: LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ ./quarry/` runs
two package test binaries and nothing else — not `./...`. The scope is deliberate on both halves:
`./internal/cli/` is where seven of this batch's eight cards land, and `./quarry/` is here as the
regression gate for batch 1's own surface, since cards 10 and 13 are the first real callers of
`GlyphView`, `RenderGlyphsJSON`, `RenderGlyphsText` and `GlyphsOptions` and a wrong signature or a
wrong assumption about them shows up as a compile failure in that package pair. Both packages'
tests run in well under a second.

The environment variable is not optional decoration. `loomyardRepo` *skips* when it is unset, so
without it card 12's four byte-identity pairs — the task's central enforcement mechanism — and the
existing fifteen golden cases all skip silently, and the batch reports green having asserted
nothing about real repository bytes. `$PWD` expands to the repository root because `verify:` runs
from there, which makes the value absolute (which `loomyardRepo`'s own stat requires) without
putting a machine-specific path in a committed file. Card 6 is what creates the checkout that value
points at, which is why it is the batch's first card.

Files this batch's verify covers that it also changes: `internal/cli/flags_test.go` (cards 7, 8, 9)
and `internal/cli/glyphs_test.go` (cards 12, 13). Files it covers as a regression gate without
changing: `internal/cli/cli_test.go`, `internal/cli/after_test.go`, `internal/cli/message_test.go`,
`internal/cli/name_test.go`, and every test file under `quarry/`.
