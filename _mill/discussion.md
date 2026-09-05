# Discussion: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)

```yaml
task: 'P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)'
slug: glyphs-verb
status: discussing
parent: main
```

## Problem

Loomyard's planning boundary (issue #226) needs a compact, complete index of a target's
symbols — one line per symbol, the glyph spelled ready to copy — so that an existing symbol's
spelling is always **copied** from a quarry answer and never composed by an LLM. Today the only
way to get every symbol of a subtree out of quarry is `quarry toc <dir> --depth all --symbols`,
whose answer carries a full docstring, a verbatim signature and a recursive directory envelope
per entry. That is the right answer for a human reading unfamiliar code and the wrong one for a
planner that wants a lookup table: it is one to two orders of magnitude larger than the
information the consumer uses.

Why now: this is the last of the three plan-alphabet primitives (`docs/roadmap.md` point 2). Its
one ordering constraint — C1's contract merge — was satisfied 2026-09-05 (`49304ca`); 2b (the
glyph-maker, `486d416`) and the goldens move (`b827a3a`) are on main; 2c (diff-to-symbols) runs in
parallel, and its only contact points with this task are the CLI dispatch table and a
`docs/rewrite-plan.md` §5 paragraph — resolve at merge time, not here.

The design pressure the task carries, from the roadmap's own words: **"views filter; no view is
ever forced."** Extraction underneath must stay complete; a lossy projection must be a view over
that complete answer and never a second, cheaper extraction path. And **named verbs are frozen
flag presets over the one query** — `quarry glyphs <target>` must run the exact same code as its
`toc` flag expansion, enforced by a golden test requiring byte-identical output, not by a
convention a later edit can quietly break.

## Scope

**In:**

- A `--view` flag on the `toc` verb, with a closed two-value vocabulary (`full`, `glyphs`) and
  absent ≡ `full`.
- A new answer shape for the `glyphs` view: a flat, non-recursive projection of the complete toc
  answer, carrying per symbol only `id`, `kind`, `file` and the `start`–`end` span, plus an
  explicit list of files that could not be fully read or parsed.
- A pure, exported projection in the facade that turns a complete `DirAnswer` into that shape,
  plus its JSON and text renderers.
- A `glyphs` verb on the CLI, implemented as an argv rewrite to a frozen `toc` flag expansion —
  one preset table, one code path.
- A byte-identity golden test proving `quarry glyphs <target>` is byte-identical to its documented
  `toc` expansion, for a file target and a directory target, in both JSON and `--text`.
- Facade `Glyphs()` built from the same frozen preset constant the CLI's rewrite uses.
- Goldens over the pinned Loomyard checkout for the glyphs view on a file, a directory, and a
  directory with `--depth`.
- `docs/rewrite-plan.md` §5: one paragraph on the view mechanism, one on the preset rule.
- `usageText` gains the verb, the flag, and the preset's expansion spelled literally.

**Out:**

- Any change to the viewless `toc` answer — the JSON envelope, the text view, exit codes, flag
  semantics. The existing goldens under `internal/cli/testdata/` must not need regeneration; that
  is the regression gate for this rule.
- Any change under `internal/engine/`'s answer construction: the walk keeps extracting docs,
  headers and signatures exactly as today. The view filters afterwards.
- An MCP tool for `glyphs` (see the `mcp-no-tool` decision).
- `--view` on `resolve`, `expand` or `name`.
- Any override of the frozen preset (no `quarry glyphs --depth 1`).
- Anything about plans, cards, handles or Loomyard's pipeline — that is issue #226's, in
  Loomyard's repository.
- Performance work: the glyphs view does the same extraction the full view does and is not
  expected to be faster.
- A second language. Go is the only alphabet; nothing here is language-specific beyond what the
  existing walk already is.

## Decisions

### view-flag-spelling

- Decision: the mechanism is `--view <name>`, a value-taking flag on `toc` only, valid for `toc`
  exactly as `--depth`, `--symbols` and `--no-symbols` already are, and rejected for every other
  verb with the existing `"%s is not valid for %s"` message.
- Rationale: the roadmap names `--view` literally (`docs/roadmap.md` line 38). A value-taking flag
  leaves room for a third view without a third boolean, and matches `--depth`'s existing shape,
  including both the `--view glyphs` and `--view=glyphs` forms `parseArgs`'s `strings.Cut` already
  supports for every value flag.
- Rejected: a boolean `--flat` (a second view later means a second mutually-exclusive boolean, the
  exact `--symbols`/`--no-symbols` pairing that already needs hand-written conflict checks);
  `--compact` (the word names the *retired* lossy view — `docs/research/output-formats/toc-dir-compact.txt`,
  whose 0.96→0.82 precision loss is `docs/rewrite-plan.md` §1's fourth measured lesson, and whose
  removal `internal/cli/testdata/INDEX.md` records as deliberate. Reusing the name would suggest
  that view came back).

### view-vocabulary

- Decision: the vocabulary is closed at exactly two values, `full` and `glyphs`. An absent
  `--view` means `full`; `--view full` is a no-op that produces byte-identical output to a viewless
  invocation. Any other value is a usage error (exit 2) naming the flag and listing the valid
  values, in the shape `--depth`'s own rejection already uses:
  `--view must be "full" or "glyphs", got %q`.
- Rationale: naming the default makes "no view is ever forced" checkable — a caller can spell the
  complete answer explicitly. A closed set means an unknown view is a usage error rather than a
  silent fallback to the full answer, which is the failure mode that would make a typo in an
  automated caller look like a working invocation returning far more than asked for.
- Rejected: accepting only `glyphs` (no way to spell the default, so a wrapper composing flags has
  to conditionally omit the flag); an open string reaching the engine (a typo silently degrades to
  the full answer).

### glyphs-answer-shape

- Decision: the `glyphs` view's answer is a new, flat, non-recursive value:

  ```go
  // GlyphsAnswer is the glyphs view's whole answer.
  type GlyphsAnswer struct {
      Target     string   `json:"target"`
      Symbols    []Symbol `json:"symbols"`
      Incomplete []string `json:"incomplete,omitempty"`
  }
  ```

  `Symbols` holds the engine's own `Symbol` values — the §4 entries, not a new per-line struct —
  with `Doc`, `Signature` and `SigEnd` cleared and `File` filled with the symbol's
  repository-relative path. `Target` echoes the query's repository-relative target. `Incomplete`
  is every file entry in the answer whose `Error` or `Lossy` field was set, as a repository-relative
  path, sorted, omitted when empty.
- Rationale: the roadmap's own words are "a flat projection … no recursive envelope", and
  `Symbol.File` exists for exactly this case — its doc comment already says it is empty inside a
  toc answer because the symbol sits in its file's entry, and filled where entries span files
  (`resolve`, `expand`). A flat list spanning files is that case. Reusing `Symbol` rather than
  declaring a narrower line type keeps one symbol shape across all four queries, which is the
  property `internal/cli/testdata/INDEX.md` already records as what distinguishes the rewritten CLI
  from V1 — and the cleared fields simply drop out of the JSON, because `doc`, `signature` and
  `sigend` all carry `omitempty` while `id`, `kind`, `file`, `start` and `end` do not.
- Rejected: the same nested `DirAnswer` with fields blanked (keeps the recursive envelope the
  roadmap explicitly excludes, and leaves a caller walking `dirs`/`files` to find symbols — the
  planner would have to reimplement the flattening); a bespoke `GlyphLine` struct (a second symbol
  shape, and the drift `Symbol`'s own stored-`ID` design exists to prevent); a bare top-level JSON
  array (no room for `incomplete`, and every other quarry answer is an object).

### incomplete-is-explicit

- Decision: a file the walk could not read or could only parse lossily contributes its path to
  `Incomplete` and contributes whatever symbols it did yield to `Symbols`. In the text view the
  incomplete paths are a trailing block, one path per line, preceded by a single blank line, each
  line spelled `[incomplete] <path>`. When `Incomplete` is empty the block, and its blank line, are
  absent — so a complete answer's text view is symbol lines and nothing else.
- Rationale: this view exists so a consumer can conclude "this symbol is not in the target" from an
  absent line. A file that failed to parse silently omits every symbol it declares, which turns a
  read failure into a wrong negative answer at the consumer. The `lossy`/`error` facts are already
  in the complete answer (`FileEntry.Lossy`, `FileEntry.Error`); dropping them here would be the
  view *losing information that changes the answer's meaning*, which is different from dropping a
  docstring.
- Rejected: silently omitting them (a wrong negative at the consumer, undetectable); carrying each
  file's full error string (the view's line grammar is one record per line and an OS error message
  is arbitrary prose — the path is enough to send the caller to `toc` for the detail, which is the
  "the complete answer stays one flag away" rule); failing the whole query on any lossy file (a
  single unparseable vendored file would make the verb useless on a real repository).

### projection-is-pure-and-late

- Decision: the projection lives in the facade as an exported pure function over a complete
  answer — `func GlyphView(target string, a DirAnswer) GlyphsAnswer` in a new `quarry/view.go` —
  and is applied *after* `Repo.TOC` returns. `internal/engine/` is not modified. Both the CLI's
  `--view glyphs` path and the facade's `Glyphs()` call this one function.
- Rationale: "extraction stays complete underneath" is the roadmap's own constraint, and the
  cheapest way to guarantee it is for the view to be unable to influence extraction at all — a
  pure function over the finished answer cannot. It is also what makes the view directly
  table-testable with hand-built `DirAnswer` values and no repository, mirroring how
  `codeForTOCError` and `rootUsageMessage` are named functions precisely so tests can address them.
  The engine keeps its single answer type and its single walk.
- Rejected: filtering during the walk (`opts.View` reaching `walkDir`, skipping doc/signature
  extraction) — faster, but it makes the view a second extraction path, which is the one thing the
  task forbids, and it would put a view concept inside the package the task says not to touch;
  projecting inside the renderers (two implementations, one per format, and the facade's `Glyphs()`
  would have no shared code to call).

### file-field-is-filled-by-the-view

- Decision: `GlyphView` fills each `Symbol.File` by joining the enclosing `DirAnswer.Dir` with the
  enclosing `FileEntry.Name`, using the same `joinRel` rule `quarry/text.go` already declares. The
  engine still emits an empty `File` inside a toc answer; nothing about the complete answer changes.
- Rationale: flattening destroys the nesting that made `File` redundant, so the flattener is
  exactly the layer that must restore it. Doing it here rather than in the engine keeps the
  viewless answer byte-identical, which is a done-criterion.
- Rejected: making the engine always fill `File` (changes every existing toc golden — forbidden);
  leaving `File` empty and having the consumer track nesting (there is no nesting left to track).

### preset-expansion

- Decision: the frozen expansion is exactly

  ```
  quarry glyphs <target>  ==  quarry toc --view glyphs --depth all --symbols <target>
  ```

  spelled once as a package-level constant in `internal/cli/flags.go` and named in `usageText`,
  in `docs/rewrite-plan.md` §5, and in the byte-identity golden.
- Rationale: the consumer wants "a complete index of a target" — for a directory target that is the
  whole subtree, so `--depth all`. `--symbols` must be explicit rather than relying on the
  per-target default, because that default is *false* for a directory target, which would make
  `quarry glyphs <dir>` answer with no symbols at all. Spelling both flags means the preset does
  not silently change if a default ever does.
- Rejected: `--depth 0` (answers only the target directory's own files, so a planner indexing a
  package tree gets a partial index and cannot tell); passing the caller's own `--depth` through
  (not a frozen preset — see `preset-is-frozen`).

### preset-is-frozen

- Decision: `glyphs` accepts `--text` and `--root` and nothing else. `--view`, `--depth`,
  `--symbols` and `--no-symbols` are rejected for `glyphs` with the existing
  `"%s is not valid for %s"` usage error, exactly as they are for `resolve` and `expand` today.
- Rationale: `--text` and `--root` are not query flags — one selects the output format and the
  other tells the CLI where the repository is; both are already valid across verbs. The three query
  flags *are* the preset, and a caller that wants to vary them is asking for `toc`, which is one
  flag away and fully documented. Allowing an override would also make "byte-identical to its
  documented expansion" untestable as a single fixed pair of argv.
- Rejected: allowing a `--depth` override (breaks the frozen-preset rule the roadmap states and
  turns `glyphs` into an alias rather than a preset).

### rewrite-mechanism

- Decision: `parseArgs` recognises `glyphs` as a verb, validates the verb-specific flag rules
  above, then rewrites its own argument slice to the expansion and re-parses it as `toc`, returning
  the resulting `request` unchanged. The rewrite happens on the argv slice, before any `request`
  field is set from a query flag, so there is exactly one place a preset's values are spelled and
  the `toc` verb's own parsing is the only parsing that ever produces the query fields.
  Concretely: on seeing verb `glyphs`, build
  `append([]string{"toc"}, append(glyphsPreset, rest...)...)` where `rest` is the original
  post-verb tokens, and return `parseArgs` of that. Recursion depth is one and bounded by
  construction — the rewritten slice's verb is `toc`, which has no preset.
- Rationale: this makes "runs the exact same code path" true of *parsing* as well as of execution:
  after `parseArgs` returns, nothing downstream — not `Run`'s dispatch switch, not `runTOC`, not
  the renderers — can tell a `glyphs` invocation from its expansion, which is precisely what the
  byte-identity golden asserts and what makes that golden hard to break accidentally.
  `req.verb` is `"toc"` after the rewrite, so `Run`'s switch needs no new case at all.
- Rejected: a `req.preset` field consulted later (something downstream can then branch on the
  preset, which is the drift the golden exists to prevent); a separate `runGlyphs` pipeline (a
  parallel implementation, explicitly forbidden); expanding in `Run` rather than `parseArgs` (the
  flag-validity rules for `glyphs` live in the parser, so the verb would be half-known in two
  places).

### facade-shape

- Decision: `func (r *Repo) Glyphs(target string) (GlyphsAnswer, error)` in `quarry/repo.go`,
  implemented as `r.TOC(target, glyphsOptions())` followed by `GlyphView(target, answer)`, where
  `glyphsOptions()` returns the same `TOCOptions` the CLI's preset expansion parses to
  (`Depth: DepthAll`, `Symbols: &true`). It returns the engine's error unchanged, exactly as `TOC`
  does, so `errors.Is(err, ErrTargetNotFound)` keeps working through it.
- Rationale: a `Repo` method, not a package-level function, because unlike `Name` this query does
  read the repository. Returning `GlyphsAnswer` rather than `DirAnswer` is the point of the verb.
  Building the options from a shared source with the CLI preset is what the drift test asserts.
- Rejected: a package-level function (claims no repository dependency, but there is one);
  returning `[]Symbol` (drops `Incomplete`); taking `TOCOptions` (that is `TOC`).

### preset-single-source

- Decision: the CLI's preset token slice and the facade's `glyphsOptions()` are asserted equivalent
  by a test that parses the CLI preset through `parseArgs` and compares the resulting
  `request.depth`/`request.symbols` against the facade's options, rather than by sharing a Go value
  across the package boundary. The token slice stays in `internal/cli`; `glyphsOptions` stays in
  `quarry`.
- Rationale: the CLI's preset is inherently a token slice (it must be parseable argv for the
  rewrite to be a rewrite) and the facade's is inherently a struct; no single value is both.
  `internal/cli` already imports `quarry`, so the test can reach both sides from the `cli` package.
  This is the "no drift from the CLI's" done-criterion made mechanical.
- Rejected: exporting the token slice from `quarry` and having the CLI import it (puts a CLI
  concept in the facade's public API); trusting the two to be edited together (the exact convention
  the task says to replace with a test).

### text-line-shape

- Decision: one line per symbol, in the answer's own order:

  ```
  <file>:<start>-<end> <kind> <id>
  ```

  e.g. `internal/logger/logger.go:155-163 function internal/logger#stderrHandlerSnapshot`. No
  directory line, no file line, no header, no docstring, no signature, no `(sig …)` clause. The
  incomplete block, when present, follows per `incomplete-is-explicit`. The rendered string has no
  trailing whitespace on any line and ends with exactly one `\n`, matching `RenderText`'s existing
  contract; an answer with no symbols and no incomplete files renders as the empty string.
- Rationale: `kind` must be spelled explicitly here, because in the existing symbol line the kind is
  only inferable from the signature — which this view drops. Leading with the file keeps the lines
  sortable and greppable by location, matching the existing `writeSymbolLine`'s own file-prefix
  form for `resolve`/`expand`. Emitting nothing but symbol lines makes the view trivially
  machine-readable, which is what the consumer does with it.
- Rejected: reusing `writeSymbolLine` with the signature suppressed (its grammar puts the span
  first without a file prefix inside a toc answer, and threading a suppression flag through it
  would make one function serve two grammars — the thing its own doc comment says it is not);
  tab-separated columns (the rest of the text view is space-separated prose-ish records);
  `<id> <kind> <file>:<span>` (loses the location-first sort).

### span-is-start-end

- Decision: the glyphs view carries `Start` and `End` only. `SigEnd` is cleared and its key
  therefore absent from the JSON (it already carries `omitempty`), and there is no `(sig …)` clause
  in the text line.
- Rationale: the roadmap says "span", and the consumer looks up a spelling and a location. `sigend`
  answers "where does the body begin", a question about reading the code — which is `toc`'s job,
  one flag away.
- Rejected: keeping `sigend` (the view is meant to be small; a field no consumer reads is a cost
  with no buyer).

### mcp-no-tool

- Decision: no MCP tool for `glyphs`. `internal/mcpserver` is not modified by this task.
- Rationale: the standing rule in `docs/roadmap.md` is that nothing is built without a measured win
  behind it, and the MCP surface specifically exists for the ladder harness's measured cells — the
  server's own doc comment records that its name and tool set are contracts of the ladder config.
  The #226 consumer reaches quarry via the facade and the CLI, mechanically, with no LLM in the
  loop. Adding a tool would add a surface with no caller on either side. The roadmap's "only
  presets would ever be exposed as tools" is a constraint on *what could* be exposed, not an
  instruction to expose this one.
- Rejected: registering a `glyphs` tool now (no consumer, and it would need its own golden,
  schema-prose pinning and tools/list assertions in `internal/mcpserver` — cost with no measured
  buyer); exposing `--view` as a property on the existing `toc` tool (same, and it widens a
  surface whose exact prose is pinned by an exact-string assertion).

### viewless-output-unchanged

- Decision: no existing golden under `internal/cli/testdata/` or `internal/engine/` is regenerated
  by this task. If any of them changes, the change is a defect in this task, not a golden that
  needs updating.
- Rationale: this is the task's own done-criterion and the only mechanical proof that the view is
  additive. Stating it as "do not run `-update` on the existing set" makes it actionable for the
  implementer, since the regeneration command exists and is one flag.
- Rejected: nothing — this is a constraint, not a choice.

## Technical context

**The CLI.** `internal/cli/flags.go` holds `request` and the hand-rolled `parseArgs`; the parser is
hand-rolled because `flag` cannot express `--depth all` alongside `--depth 3`. The verb gate is a
literal four-way string comparison and the `"no verb given"` and `"unknown verb"` messages both
enumerate the verbs — three sites to extend for `glyphs`, plus the `--root` validity check (which
excludes `name` by name) and each query flag's `verb != "toc"` guard. Value flags are read through
the local `nextValue()` closure, which handles both `--flag=value` and `--flag value`; `--view`
follows it.

`internal/cli/cli.go` holds `Run` and the four per-verb pipelines. Under the `rewrite-mechanism`
decision, `Run` needs no change at all: `req.verb` is already `"toc"` by the time it dispatches.
`runTOC` needs one branch — after the answer is in hand, if the request's view is `glyphs`, render
through the new renderers instead of `RenderText`/`RenderJSON`. `runTOC`'s existing
`targetIsFile` bool (from the `os.Lstat` it already performs) is not needed by the glyphs view,
which renders the same way for both target shapes; that asymmetry is worth a sentence in the code,
because every other text rendering in the package needs it.

`Run`'s doc comment is a numbered, prose specification of every pipeline — it is the package's real
documentation and it is expected to be extended, not left stale, when `runTOC` grows a branch.
Same for `usageText` in `internal/cli/usage.go`, whose own doc comment fixes its rules: ASCII only,
one combined flag list with per-verb validity stated in each flag's description, per-verb shapes in
the usage block above. The `glyphs` line belongs in the usage block, `--view` in the flag list, and
the preset's expansion is worth spelling literally so `--help` alone answers "what does glyphs
do".

**The facade.** `quarry/` is a thin delegation layer: `quarry.go` is aliases and sentinels,
`repo.go` is the four query methods, `render.go` the JSON renderers (all sharing `renderJSON`'s
encoder configuration — two-space indent, no HTML escaping, one trailing newline), `text.go` the
text renderers. Every new renderer must go through `renderJSON` rather than building its own
encoder, or the byte contract drifts. `text.go` already declares its own `joinRel` and
`normalizeProse`; the projection reuses `joinRel`. Note `render.go`'s header comment: the alias
types carry no methods, because Go forbids declaring a method on a type alias defined in another
package — so every renderer is a package-level function. `GlyphsAnswer` is a *new* type declared in
`quarry`, not an alias, so it could carry methods; it should not, for symmetry with the rest of the
package.

**The engine's answer shape** (`internal/engine/answer.go`) is the file whose doc comment states
that the emitted key set is closed and that no field is added or renamed without a corresponding
decision. This task adds no field to `Symbol`, `FileEntry` or `DirAnswer` — `GlyphsAnswer` is a new
type in the facade, over the same `Symbol`. `FileEntry.Symbols` is a `*[]Symbol` specifically so
"not requested" is distinguishable from "requested, none found"; the projection must treat a nil
`Symbols` as "no symbols to contribute" and not as an error, even though the frozen preset always
requests them (a hand-built answer in a table test will have nil entries).

**The walk** fills `FileEntry.Error` for a file it could not read or decode and `FileEntry.Lossy`
for one whose parse tree reported a syntax error; the two are mutually exclusive. Both are the
input to `Incomplete`.

**The goldens.** `internal/cli/after_test.go` drives a table of `afterGoldenCase` rows through
`Run` in-process against a Loomyard checkout pinned at `72c23d9`, resolved from
`LADDER_LOOMYARD_REPO` by `loomyard_test.go`'s `loomyardRepo`, which *skips* when the machine has
no checkout and *fails* when it has the wrong one. Each golden file is the invocation line
`$ quarry <verb> <invocation>`, a blank line, then stdout verbatim; the expected exit code lives in
the table and in `testdata/INDEX.md`, never in the file. Each row spells its `invocation` suffix
literally so a machine-specific `--root` cannot leak into a committed golden. Regeneration is
`LADDER_LOOMYARD_REPO=<checkout> go test ./internal/cli/ -run TestAfter -update` — and the
`TestAfter` prefix match is load-bearing on the test function's name.

`testdata/INDEX.md` is a total before-to-after table with a row per file on both sides; it must
gain a row per new golden, and its "fifteen files" counts (in the INDEX and in `after_test.go`'s
own header comment) become the new number. The INDEX also already carries the paragraph explaining
that the *old* compact view is gone and not replaced — the new view's rows should not read as that
view returning; a sentence distinguishing them belongs there.

**Docs.** `docs/rewrite-plan.md` §5 "The queries" has one bold-led paragraph per verb; the `toc`
paragraph is the shortest and currently spells the flag list as
`toc <dir|file> [--depth N|all] [--symbols]`. It gains `--view` there, plus the two paragraphs the
task requires. `docs/roadmap.md` point 2a describes this task in the future tense; the roadmap's own
header says it "only ever says what is ahead", so 2a is removed from it as part of this task's
completion — leaving 2b and 2c's already-merged status alone.

**Parallel task contact points.** 2c (diff-to-symbols) will touch the same verb gate in
`parseArgs`, the same three verb-enumerating messages, the same `usageText`, the same
`docs/rewrite-plan.md` §5, and the same `Run` doc comment. Expect textual conflicts in exactly
those places at merge; there is no semantic overlap.

## Constraints

No `CONSTRAINTS.md` at the hub root. From `CLAUDE.md` and the task body:

- Go only. No Python. No new module dependency — everything here is standard library plus what is
  already imported.
- Additive: no existing envelope, flag, verb behaviour, exit code or golden changes.
- One implementation: the preset rewrites flags; it never reimplements or post-processes the query.
  The golden test is the enforcement, not a convention.
- The view filters; extraction underneath stays complete. `internal/engine/` is not modified.
- The verb knows nothing about plans, cards or handles.
- `go test ./... && golangci-lint run` green.
- Comment discipline: this codebase's doc comments carry the *reasoning* — why a function is named
  rather than inlined, why an unreachable branch exists, why two files must not be changed
  independently. New code matches that density, and existing doc comments that a change makes
  incomplete (notably `Run`'s numbered pipeline and `usageText`'s own rules) are updated with it.

## Testing

**Pure table tests, no repository (the bulk of the value):**

- `parseArgs` — TDD candidate. `--view full`, `--view glyphs`, `--view=glyphs`, `--view` with no
  value, an unknown `--view` value, `--view` on `resolve`/`expand`/`name`; the `glyphs` verb with
  no flags, with `--text`, with `--root`, with each of `--view`/`--depth`/`--symbols`/`--no-symbols`
  (each rejected), with zero and with two targets; `glyphs` in the `--help` scan (help still wins);
  the `unknown verb` and `no verb given` messages now naming five verbs. The load-bearing case:
  `parseArgs(["glyphs", "x"])` returns a `request` deep-equal to `parseArgs(["toc", "--view",
  "glyphs", "--depth", "all", "--symbols", "x"])` — the rewrite asserted at the parser, before any
  rendering.
- `GlyphView` — TDD candidate, over hand-built `DirAnswer` values: a single-file answer; a nested
  answer two directories deep (order is depth-first, matching `dirBlocks`); a file entry with nil
  `Symbols`; a file entry with an empty-but-present `*[]Symbol`; entries with `Lossy` and with
  `Error` set (both land in `Incomplete`, sorted, deduplicated is moot but stated); `File` filled
  through `joinRel` including the `Dir == "."` root case; `Doc`/`Signature`/`SigEnd` cleared on
  every symbol; the input `DirAnswer` not mutated (the projection copies — a shared-backing-array
  bug here would corrupt a caller's own answer).
- The glyphs text renderer — the line grammar for each kind, an answer with no symbols (empty
  string), an answer with only incomplete files, one with both, and the no-trailing-whitespace /
  single-trailing-newline contract.
- The glyphs JSON renderer — the key set (`target`, `symbols`, `incomplete`), `incomplete` omitted
  when empty, `doc`/`signature`/`sigend` absent from every symbol, and the shared `renderJSON` byte
  contract (two-space indent, one trailing newline).
- `preset-single-source` — the CLI preset tokens parsed through `parseArgs` yield the same depth and
  symbols the facade's `glyphsOptions()` carries.

**Golden tests over the pinned checkout** (new rows in `afterGoldenCases`, all `exitOK`):

- `glyphs-dir.txt` — `glyphs internal/logger`
- `glyphs-dir-text.txt` — `glyphs --text internal/logger`
- `glyphs-file.txt` — `glyphs internal/logger/logger.go`
- `glyphs-file-text.txt` — `glyphs --text internal/logger/logger.go`
- `toc-view-glyphs-depth.txt` — `toc --view glyphs --depth 1 internal/logger` (the directory-with-depth
  case; it is a `toc` invocation rather than a `glyphs` one precisely because the preset is frozen
  at `--depth all`)

Pick a directory target with a real subtree for the depth case if `internal/logger` has no
subdirectory at the pin — the point of the case is that a nested level's symbols appear in the flat
list with their own `file` values.

**The byte-identity golden** — its own test function, not a golden file: for each of a file target
and a directory target, and for each of JSON and `--text`, run `Run(["glyphs", "--root", repo,
<flags>, <target>])` and `Run(["toc", "--root", repo, "--view", "glyphs", "--depth", "all",
"--symbols", <flags>, <target>])` into separate buffers and require both stdout *and* stderr *and*
the exit code to be identical. Four pairs. This is the task's central enforcement mechanism and
should read as such — its doc comment states that a failure here means the preset has grown a
second code path.

**The regression gate** — `go test ./internal/cli/ ./internal/engine/` with the pinned checkout
present must pass with every pre-existing golden file unchanged in `git status`. An implementer who
finds an existing golden failing must fix the code, never run `-update`.

**Facade** — `Glyphs()` against the pinned checkout compared against `GlyphView(target, TOC(target,
glyphsOptions()))` computed in the test, so "no drift from the CLI's" is asserted from the facade
side too.

## Q&A log

- **Q:** How is the view mechanism spelled — `--view <name>`, a `--flat` boolean, or `--compact`? **A:** [auto-pick] `--view <name>`. **Why:** the roadmap names it literally; a value flag scales to a third view without a second mutually-exclusive boolean; "compact" is the retired lossy view's own name and reusing it would read as that view returning.
- **Q:** What is the view vocabulary — closed `full|glyphs` with absent ≡ `full`, `glyphs` only, or an open string? **A:** [auto-pick] closed `full|glyphs`. **Why:** the default becomes spellable, and an unknown value is a usage error rather than a silent degradation to the full answer.
- **Q:** What does the glyphs view return — a flat `[]Symbol`-carrying value, the same nested `DirAnswer` with fields blanked, or a bespoke line struct? **A:** [auto-pick] a flat, non-recursive value carrying the engine's own `Symbol` entries. **Why:** the roadmap says "no recursive envelope", and `Symbol.File` exists for exactly the entries-span-files case; a bespoke struct would be a second symbol shape.
- **Q:** Where does the projection happen — a pure facade function over the complete answer, engine-level filtering during the walk, or render-time only? **A:** [auto-pick] a pure exported facade function applied after `Repo.TOC` returns. **Why:** it makes "extraction stays complete underneath" structurally true, keeps `internal/engine/` untouched, and is table-testable with no repository.
- **Q:** Does `--view` apply to the other verbs? **A:** [auto-pick] `toc` only, exactly like `--depth` and `--symbols`. **Why:** the other three verbs answer about one glyph, not a listing; there is nothing to project.
- **Q:** What are the frozen preset values behind `glyphs`? **A:** [auto-pick] `toc --view glyphs --depth all --symbols`. **Why:** the consumer wants a complete index of the target subtree, and `--symbols` must be explicit because the per-target default is *false* for a directory target — relying on it would make `quarry glyphs <dir>` answer with no symbols.
- **Q:** Which flags does `glyphs` accept? **A:** [auto-pick] `--text` and `--root` only; the three query flags are rejected as not valid for the verb. **Why:** those two are not query flags, and the query flags *are* the preset; an override would make the byte-identity pair untestable as fixed argv.
- **Q:** How is the argv rewrite implemented? **A:** [auto-pick] a preset token slice in `flags.go`; `parseArgs` rewrites and re-parses as `toc`, so `req.verb` is `"toc"` downstream. **Why:** nothing after the parser can distinguish the verb from its expansion, which is what the golden asserts; `Run`'s dispatch switch needs no new case.
- **Q:** What is the facade's shape? **A:** [auto-pick] `func (r *Repo) Glyphs(target string) (GlyphsAnswer, error)`. **Why:** it reads the repository, so it is a method; returning the projected answer is the verb's whole point.
- **Q:** Does `glyphs` get an MCP tool? **A:** [auto-pick] no; `internal/mcpserver` is untouched. **Why:** the standing measured-win rule, and the #226 consumer reaches quarry via the facade and CLI with no LLM in the loop — a tool would have no caller on either side.
- **Q:** What is the text line's exact shape? **A:** [auto-pick] `<file>:<start>-<end> <kind> <id>`, one line per symbol, nothing else. **Why:** kind must be explicit once the signature is dropped; file-first keeps the lines sortable by location, matching the existing symbol line's own file-prefix form.
- **Q:** How do `lossy` and `error` files appear in the view? **A:** [auto-pick] an explicit `incomplete` list (JSON key, trailing `[incomplete] <path>` block in text). **Why:** the consumer concludes "absent line means absent symbol"; a silently-dropped unparseable file turns a read failure into a wrong negative answer. This is why the JSON is a small object rather than a bare array.
- **Q:** Does the view carry `sigend`? **A:** [auto-pick] no; the span is `start`–`end`. **Why:** "where does the body begin" is a question about reading the code, which is `toc`'s job, one flag away.
- **Q:** What does the byte-identity golden cover? **A:** [auto-pick] a file target and a directory target, each in JSON and `--text` — four pairs, comparing stdout, stderr and exit code. **Why:** the two target shapes take different engine paths (`walkDir` vs `fileTargetAnswer`) and the two formats are separate renderers.
- **Q:** How is the "directory with `--depth`" golden expressed, given the preset is frozen at `--depth all`? **A:** [auto-pick] as a `toc --view glyphs --depth 1` invocation. **Why:** it is a view case, not a preset case; forcing it through `glyphs` would require the override the frozen-preset rule forbids.
