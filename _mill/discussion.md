# Discussion: The glyph package (T1)

```yaml
task: The glyph package (T1)
slug: glyph-package
status: discussing
parent: main
```

## Problem

Quarry is being rewritten around one identifier, the glyph — `unit#member`, as in
`internal/reedengine/render#Renderer.Draw`. It is the name every quarry query takes and returns, and
the name Loomyard's plan cards will carry as their targets. `docs/glyph.md` is the contract; §6 of it
says in as many words that **there is one implementation of the glyph grammar, and it is the `glyph`
package** — Loomyard's plan parser imports it rather than re-implementing parsing, printing or
canonicalisation.

Right now that package does not exist. `main` after T0 holds only the Go tree-sitter extractor
(`internal/quarryengine`, `.../toc`, `.../treesitter`) and the two rewrite documents. T1 is wave 1 of
`docs/rewrite-plan.md` §12 and the first new package: it adds `glyph/` with `type Language`,
`type Glyph`, `Parse(lang, s)` and `Glyph.String()`, implementing exactly the Go alphabet of
`docs/glyph.md` §1–§3.

**Why now:** T1 blocks T3 (engine core re-keys `toc` by glyph), which blocks T4 and everything after.
It is also the package Loomyard imports **without the engine**, so it must be pure Go with no cgo and
no dependencies — which the engine, a cgo tree-sitter binding, can never be. Nothing else in the tree
can host this code.

## Scope

**In:**

- A new package `github.com/Knatte18/quarry/glyph` at `glyph/` in the repository root.
- `type Language string`, with the single constant `Go Language = "go"`.
- `type Glyph struct { Lang Language; Unit string; Owner []string; Name string; Params []string }`
  — the struct shape is fixed verbatim by `docs/glyph.md` §6 and is not open for redesign.
- `Parse(lang Language, s string) (Glyph, error)` — the syntactic check. Reads no source. Performs
  the language-free structural split at the first `#`, then validates both halves against the Go
  alphabet of §2–§3.
- `Glyph.String() string` — the canonical spelling, a total pure printer.
- An exported `*ParseError` with a closed `Reason` vocabulary, so every reject in the spec is an
  error whose message names what was wrong and whose cause is machine-checkable.
- Table tests built from every example and every corner case in `docs/glyph.md` §1–§3, including all
  rejects, each case citing the spec section it came from.

**Out:**

- `Python` and `CSharp` `Language` constants. §6 reserves those names for later ("`Python` and
  `CSharp` are the names reserved for the alphabets below"); the task says
  explicitly: *do not define them, do not stub them.* Their alphabets are exercised only through the
  language-free structural split (see Decision "Python and C# examples as split-only tests").
- Any engine code, file reading, tree-sitter, or filesystem access. `resolve` (T4) is where a glyph
  meets source; `Parse` never touches a disk.
- Any dependency outside the Go standard library, and any cgo.
- Resolution semantics entirely: `found` / `not_found` / `ambiguous` / `multipart`, the `unit:` field
  on a miss, path-without-`#` targets, ordering guarantees. All of §5 is T4.
- Edits to `docs/glyph.md`. Where the spec is unclear this file records the question for the hub; the
  hub fixes the spec, the code does not fix itself around it.
- A validating constructor or `Validate()` method (see Decision "No constructor, String is a pure
  printer").
- Loomyard-side adoption of glyphs; that is Loomyard's repository, after T5.

## Decisions

### Language is a string type, not an iota int

- **Decision:** `type Language string`, with `const Go Language = "go"` as the only value defined.
  `Parse` rejects any other value, including the zero value `""`, with reason `unsupported_language`.
- **Rationale:** the zero value must not be a valid language. With `type Language int` and `iota`,
  `Go` would be the zero value, so `Parse(0, s)` — a caller who forgot the argument, or a
  zero-valued struct field — silently means Go. A string zero value is `""`, which is invalid, so the
  mistake is an error at the first call. It also matches `internal/quarryengine/toc`'s existing
  `Language string` field on `FileTOC`/`DirEntry`, and gives a free `%s` in error messages and any
  future JSON without a marshaller.
- **Rejected:** `type Language int` with `iota` (zero value is a valid language); an interface or a
  registry (over-built for one language).

### Parse is strict: the canonical form is the only accepted Go spelling

- **Decision:** for Go, `Parse` accepts exactly the canonical spelling and nothing else. Each of the
  following is a distinct, named reject, not a normalisation:
  `./internal/logger`, `internal/logger/`, `/internal/logger`, `internal//logger`,
  `internal/../logger`, `Box[T]`, `*dualHandler.Handle`, `(*dualHandler).Handle`,
  `internal/reedengine/render.Renderer.Draw` (Go's own `go doc` spelling, §7).
- **Rationale:** §1 — "Glyphs are case-sensitive and each symbol has exactly one glyph. No short
  form, no alias." §5 — "Resolution never guesses. There is no fuzzy matching, no case folding, no
  'did you mean'." A tolerant parser would create exactly the second spelling the contract exists to
  prevent, and §7 says Go's own dotted names "are not glyphs and are not accepted where a glyph is
  expected: quarry does not try alternatives."
- **Consequence to carry into the tests:** for Go, `Parse` then `String` is the identity **and**
  `String` then `Parse` is the identity, over every accepted input. The task brief's phrase
  "non-canonical spellings the spec says are accepted come out canonical" describes an **empty set**
  for the Go alphabet — the spec names no accepted non-canonical Go spelling. That is a finding, not
  an omission: the round-trip property test is therefore total over the accept table.
- **Rejected:** a fixed tolerance list (`./`-prefix stripping, trailing-slash stripping, type-argument
  stripping, receiver-star stripping) with normalisation on output. Every entry on that list is a
  second spelling of one glyph.

### The split is at the first `#` and is language-free

- **Decision:** `Parse` first splits `s` at the **first** `#`. No `#` at all is the distinct reject
  `no_separator`, whose message says a glyph needs a `#` and that a repository-relative path is not a
  glyph (§7). A `#` appearing in the member half is then a Go **member-alphabet** reject
  (`member_bad_rune`), not a structural one. The split function itself is language-free and is the
  single skeleton every future alphabet reuses.
- **Rationale:** §1 — "Structurally, any glyph splits at its first `#`. That split needs no language."
  Keeping the "more than one `#`" case as a member reject rather than a pre-split reject is what makes
  the skeleton untouched when Python or C# is added: those alphabets decide for themselves what their
  member half may contain.
- **Reject precedence is fixed: the language check runs first.** `Parse` validates `lang` before it
  touches `s` at all, so an input that fails both checks reports `unsupported_language`, never
  `no_separator` — `Parse(Language("python"), "no-hash")` is `unsupported_language`. This is not a
  detail the tests may leave to chance: the `unsupported_language` cases would otherwise have to be
  chosen to be well-formed glyphs to avoid depending on an unstated order, and a later reader would
  not know whether that was deliberate. Stating it lets those cases use any input at all. The
  ordering also follows from what the two checks mean: `lang` says which alphabet `s` is being read
  in, so "which alphabet" must be answerable before "is `s` well-formed".
- **After the language check, the order within one Go parse is:** structural split (`no_separator`),
  then the unit half, then the member half. A string failing both halves reports the unit's reason,
  because the unit is what the message should name first.
- **Rejected:** rejecting any string containing more than one `#` before the split (bakes a Go rule
  into the language-free layer); splitting at the last `#`; validating `s` before `lang` (leaves the
  reason for a doubly-invalid input undefined).

### The Go unit alphabet

- **Decision:** a Go unit is one or more `/`-separated segments. Each segment must be non-empty, must
  not be `.` or `..`, and must contain no `/`, no `#`, no `\`, no ASCII control character, and no
  whitespace (space, tab, newline, or any `unicode.IsSpace` rune). Everything else is allowed —
  Unicode letters, digits, `.`, `-`, `+`, `~` — because a Go package directory may legitimately be
  named any of them. There is no leading `./`, no leading `/`, and no trailing `/`.
- **Rationale:** §2 — the Go unit is "its path relative to the repository root". The `.`/`..`/empty
  bans keep one directory from having two spellings. The backslash ban catches a Windows path pasted
  in. The whitespace ban is the one rule the spec does not state, and it is **this task's proposal,
  not a consequence of §6.** §6 is explicitly willing to live with glyphs that need quoting — it says
  of C#'s `(`, `,` and `<` that a writer should "quote them where a format cares" — so nothing in the
  spec derives a whitespace ban. The argument for it is weaker and separate: a unit containing a space
  gives one directory two easily-confused spellings in a plan file, and no Go repository in evidence
  needs one. T1 adopts it so the alphabet is closed rather than open-ended, and routes it to the hub
  below as a non-blocking spec question the hub may accept or drop. Dropping it deletes one predicate
  in `golang.go` and one reject-table row.
- **`_test` needs no special handling.** §2's external-test-package unit, `internal/logger_test`, is
  an ordinary path as far as the parser is concerned — it satisfies the segment rules with no rule of
  its own. The fixed struct carries no flag for it, and distinguishing "the directory `logger_test`"
  from "the external test package of `logger`" requires reading source, which is T4's job. The spec's
  example is still a test: it parses and round-trips.
- **Rejected:** a portable-filename character class `[A-Za-z0-9._+-]` (would reject legal Go directory
  names, including Unicode ones); "anything but `/` and `#`" (accepts control characters and a
  trailing space, both of which produce two spellings of one directory in practice).

### The Go member alphabet

- **Decision:** the member half is one or two `.`-separated components — `Name` for a package-level
  `func`/`type`/`const`/`var`, `Type.Name` for a method or interface method. Three or more components
  is the reject `member_too_deep` naming Go's lack of nesting. Each component must be a Go identifier:
  the first rune is `_` or `unicode.IsLetter`, every later rune is `_`, `unicode.IsLetter` or
  `unicode.IsDigit`. No parentheses, no `[`, no `]`, no `*` anywhere. On success `Owner` holds the
  first component when there are two and is `nil` when there is one; `Name` holds the last component;
  `Params` is always `nil`.
- **Rationale:** §3 Go — "Go has no nesting and no overloading, so a member never has more than one
  `.` and never has parentheses. Type parameters are not part of a glyph: `Box[T]` is `Box`."
  Rejecting `*` and `(` gives `*dualHandler.Handle` and `(*dualHandler).Handle` their own messages
  rather than a generic "bad rune", which matters because §3 says explicitly that pointer-ness is not
  part of the glyph and a reader will try to write it. Using Go's own Unicode identifier rule rather
  than ASCII is required: Go identifiers are Unicode, and `toc` will emit them in T3.
- **`init` needs no special handling.** §3 — `internal/logger#init` is one glyph and several `init`
  functions make it `multipart` at resolve time. `init` is a plain identifier to the parser; the
  multiplicity is entirely T4's. The spec's example is still a test.
- **Rejected:** ASCII-only identifiers (wrong for Go); accepting `Box[T]` and stripping the type
  arguments (a second spelling — see the strictness decision).

### Go keywords are rejected; the blank identifier is accepted

- **Decision:** a member component that is one of Go's 25 reserved keywords is the reject
  `member_keyword`. A component that is `_` is **accepted**.
- **Rationale:** a keyword can never be a declared name in Go, so rejecting it costs nothing, is a
  fixed table needing no source, and produces a far better message than "bad identifier". `_` is
  different: `func _() {}` and `var _ = …` are legal Go declarations that `toc` can list — this very
  repository has `var _ = quarry_requires_CGO_ENABLED_1_with_a_C_toolchain` in
  `internal/quarryengine/cgoguard_nocgo.go`. Rejecting `_` at parse time would break T3's done
  criterion that **every** declaration `toc` lists has a glyph. A glyph naming `_` will normally
  resolve `ambiguous`, which is exactly the right answer and is §5's business, not `Parse`'s.
- **Rejected:** rejecting both (breaks the T3 round trip); accepting both (loses a free, purely
  syntactic, well-named reject).

### Params: nil means no parentheses, non-nil means parentheses

- **Decision:** `String()` prints parentheses when `Params != nil`, including when it is an empty
  non-nil slice, and prints none when `Params == nil`. Go's `Parse` always leaves `Params` nil.
- **Rationale:** this is decided now, while it is free, because C# needs it later and changing it then
  would change the one implementation everyone imports. §3 C# — "**A method or constructor always
  carries its parameter types**", including `Renderer.Draw()` with none, while "properties, fields,
  events and types have no parentheses". A `len(Params) > 0` rule cannot spell `Draw()` at all; only
  the nil/empty distinction can carry both.
- **Rejected:** parenthesise iff `len(Params) > 0`; a separate boolean field (the struct shape is
  fixed by §6 and gains no field).

### One `*ParseError` with a closed Reason vocabulary

- **Decision:** a single exported error type,
  `type ParseError struct { Lang Language; Input string; Reason Reason; Detail string }`, with
  `Error() string` composing a message that names what was wrong, and an exported
  `type Reason string` whose constants are a closed set (one per reject in the spec). Callers use
  `errors.As`; tests assert on `Reason`, never on message text.
- **Rationale:** the task requires "every reject in the spec is an error with a message that names
  what was wrong". A closed `Reason` vocabulary makes "every reject in the spec" enumerable and
  checkable — the test table can assert one case per constant and a reviewer can see nothing is
  missing. It mirrors `internal/quarryengine/toc`'s existing closed-`Kind` convention, and keeps
  message construction in exactly one place so wording can be improved without breaking tests.
- **`glyph` cannot reuse `internal/quarryengine.ErrLanguageUnsupported`.** That sentinel lives under
  `internal/`, which Loomyard — a different module — may not import. §6 requires `glyph` to be
  importable "by anything … without the engine". The duplication is required, not an oversight; do
  not "fix" it by importing the engine's sentinel.
- **Rejected:** ~12 package-level sentinels wrapped with `%w` (idiomatic and the repo's existing
  pattern, but the count is noisy and it scatters message text across a dozen declarations); plain
  `fmt.Errorf` strings (no stable handle for tests, no enumerable reject set).

### No constructor, and String is a pure printer

- **Decision:** the package exports no `New(...)` and no `Glyph.Validate()`. `String()` is total: it
  never returns an error, never panics, and does not validate. A `Glyph` built by hand is the
  builder's responsibility, and this is documented on the type.
- **Rationale:** T3 builds `Glyph` values directly from the tree-sitter parse rather than from a
  string, and its own done criterion — the whole-repository round trip, where every declaration `toc`
  lists resolves back to its own span — is a stronger check than a constructor would be. Adding a
  constructor now guesses at a signature T3 has not asked for yet. YAGNI; revisit in T3 if it needs
  one, in this package, where it belongs.
- **Rejected:** `New(lang, unit, owner, name, params) (Glyph, error)`; `Glyph.Validate() error`;
  a `String()` that panics on impossible states.

### Files per concern inside one package

- **Decision:** `glyph/doc.go` (package doc), `glyph/glyph.go` (`Language`, `Go`, `Glyph`, `String`),
  `glyph/parse.go` (the language-free split and the language dispatch), `glyph/golang.go` (the Go
  unit and member alphabets), `glyph/errors.go` (`Reason`, `ParseError`). Adding a language later
  adds one file beside `golang.go` and touches `parse.go`'s dispatch only.
- **Rationale:** matches `internal/quarryengine/toc`'s existing layout (`types.go`, `strategy.go`,
  `golang.go`, `classify.go`, …) and `docs/rewrite-plan.md` §12 T3's rule for the engine: "one
  package, files per concern, never a package per verb". It also makes the "the split needs no
  language" claim visible in the file structure rather than only in a comment.
- **Rejected:** a single `glyph.go` (fine today, wrong the moment a second alphabet lands); a
  subpackage per language (splits the one implementation §6 insists on).

### Python and C# examples as split-only tests

- **Decision:** every Python and C# example in §1 and §3 is a test of the **structural split** alone,
  written as a white-box (same-package) test asserting the example divides into the unit and member
  halves the spec documents. In addition, a test asserts that `Parse` with any `Language` other than
  `Go` — including `Language("python")`, `Language("csharp")` and the zero value — returns
  `Reason` `unsupported_language`. No `Python` or `CSharp` constant is defined anywhere.
- **Rationale:** the done criterion is "every example and corner case in `docs/glyph.md` §1–§3 is a
  test", and the Python and C# examples are examples in those sections. They cannot be `Parse` tests,
  because the task forbids defining or stubbing those `Language` values. What they *can* test is the
  claim the task brief makes in its own words — "the structural split at `#` accepts the other
  alphabets' shapes, so adding a language later does not touch the parser's skeleton" — and that is
  the useful half. This is the honest reading of the criterion, and it is recorded here rather than
  silently skipped.
- **Rejected:** skipping those rows (fails the done criterion); defining `Python`/`CSharp` as
  unsupported constants (explicitly banned by the task).

### The root-package unit is rejected, and it is a spec question

- **Decision:** a Go package at the repository root has **no glyph** in this implementation. `""` and
  `.` are both rejects (`unit_empty` and `unit_dot_segment` respectively). This is recorded below as
  a **blocking spec question for the hub**, to be answered before T3.
- **Rationale:** §2 says the Go unit is "its path relative to the repository root" and never says what
  that is for the root itself. Both plausible spellings — `.` (what `go doc .` and `go list ./...`
  accept) and `""` (`#run`) — would be inventions, and inventing one here is precisely the
  "second implementation" §6 exists to prevent. The task is explicit: "Do not edit `docs/glyph.md` to
  fit the code; where the spec is unclear, ask, and the hub fixes the spec."
- **Impact if unanswered:** T3's done criterion is a round trip over **all** of Loomyard with "zero
  misses, zero extras". A library repository with a package in its root — which is common — has
  declarations `toc` will list and `glyph` cannot name. Whether Loomyard specifically has one is not
  known from this worktree and should be checked when the hub answers.
- **Rejected:** accepting `.`; accepting `""`. Either can be adopted in one place once the spec says
  which; the accept table and the round-trip property both extend without restructuring.

### Input is not trimmed

- **Decision:** `Parse` does no trimming. Leading or trailing whitespace is a reject
  (`unit_bad_rune` or `member_bad_rune` as the position dictates), and `""` is `no_separator`.
- **Rationale:** trimming is normalisation, and normalisation is the second spelling the strictness
  decision rules out. A caller that read a glyph off a line is the right place to trim.
- **Rejected:** `strings.TrimSpace` on entry.

## Technical context

- **Module and placement.** `go.mod` declares `module github.com/Knatte18/quarry`, `go 1.26`. The new
  package goes at `glyph/` in the repository root, so its import path is
  `github.com/Knatte18/quarry/glyph` — deliberately **not** under `internal/`, because Loomyard is a
  different module and `internal/` would block it. This is the whole point of §6.
- **What `main` holds today.** `internal/quarryengine` (a cgo build-guard pair, `doc.go`, and
  `ErrLanguageUnsupported`), `internal/quarryengine/toc` (the Go extraction strategy and helpers),
  `internal/quarryengine/treesitter` (the Go grammar seam). `glyph` imports **none** of them and must
  not: `go.mod` already requires `go-tree-sitter`, and any path from `glyph` into the engine drags cgo
  in and fails the done criterion.
- **cgo interaction with the verify step.** `internal/quarryengine/cgoguard_nocgo.go` makes a
  `CGO_ENABLED=0` build of the engine fail at compile time by design. So `CGO_ENABLED=0 go build ./...`
  at the repository root **will** fail, and that is correct, not a regression. The no-cgo claim is
  proved for this package alone with `go list -deps ./glyph`, and `go build ./... && go test ./...`
  is run in the ordinary cgo-enabled configuration. Do not attempt to make the whole tree build
  without cgo.
- **Allowed imports.** Standard library only, and in practice only `fmt`, `strings` and `unicode`.
  `go list -deps ./glyph` must show nothing else and no cgo. Anything more is a design error, not a
  dependency decision.
- **Existing conventions to follow.** Every file in `internal/quarryengine/toc` opens with a
  file-level comment naming what the file holds and why (see `types.go`, `cgoguard.go`); exported
  identifiers carry godoc; closed vocabularies are `string`-based named types with grouped constants
  and a doc comment on each (`toc.Kind`). Match this. See `.claude` skills `golang-comments`,
  `golang-testing`, `golang-build` for the repository's Go rules.
- **The spec sections that matter,** for a planner reading only this file: `docs/glyph.md` §1 (the
  form, the language-free split, the examples table), §2 (the unit per language, the Go `_test` unit,
  the rejected shorter schemes), §3 (the member per language, the Go rules on `.`, parentheses, type
  parameters, receivers and `init`), §6 (the exact Go API and the one-implementation rule), §7 (why
  `go doc` names are not glyphs). §4 and §5 are context, not this task.
- **Downstream consumers, so nothing is designed against them by accident.** T3 constructs `Glyph`
  values from a parse tree and calls `String()`; T4 (`resolve`) is what turns `_test` units, several
  `init` functions and build-tag duplicates into `multipart`/`ambiguous`; Loomyard's `planparser`
  imports this package after T5 and deletes its own name handling.

## Constraints

- No `CONSTRAINTS.md` exists at the hub root; these come from `CLAUDE.md`, the task brief and the
  two rewrite documents.
- **Go repository; no Python is introduced.** (`CLAUDE.md`.)
- **Pure Go, no cgo, no dependencies outside the standard library.** `go list -deps ./glyph` proves
  it. (`docs/glyph.md` §6; `rewrite-plan.md` §12 T1.)
- **No engine code, no file reading, no tree-sitter, no filesystem or network access** in this
  package. `Parse` reads no source. (Task brief.)
- **`docs/glyph.md` is not edited by this task.** Unclear points are asked; the hub fixes the spec.
- **The `Glyph` struct shape and the `Parse`/`String` signatures are fixed verbatim by §6** and are
  not redesigned here.
- **No tracked file may carry a machine path.** (`HANDOFF.md`.)
- **The task ends with `go build ./... && go test ./...` green and one merge to `main`.**
  (`rewrite-plan.md` §12.)

## Testing

TDD is the right shape for the whole package: the spec supplies the cases before any code exists, so
the accept and reject tables can be written first and drive the implementation. Write the tables
first, watch them fail, then implement.

**`parse_test.go` — the language-free layer.**

- The structural split over every example in §1's table, including the Python and C# rows
  (`loomyard.engine.layout#Beta.Inner.handle`, `Loomyard.Engine.Layout#Renderer.Draw(int)`,
  `Loomyard.Engine.Layout#Renderer.Title`): each divides into the unit and member halves the spec
  documents. White-box, same-package, since the split is unexported.
- First-`#` semantics: a string with two `#` splits at the first, and the second `#` reaches the Go
  member validator (which then rejects it).
- `unsupported_language`: `Parse` with `Language("python")`, `Language("csharp")`, `Language("")` and
  some arbitrary value each returns that reason and the zero `Glyph`.
- **Reject precedence**, asserting the order the Decisions section fixes: `Parse(Language("python"),
  "no-hash")` — an input that fails both the language check and the split — returns
  `unsupported_language`, not `no_separator`. One case for a doubly-invalid input pins the order so a
  later refactor cannot silently swap the two checks; because of it, the `unsupported_language` cases
  above are free to use any input rather than being restricted to well-formed glyphs.

**`golang_test.go` — the Go alphabet. The main table, and the bulk of the work.**

*Accepts*, at minimum every Go example the spec writes down, each carrying the section it came from:
`internal/logger#stderrHandlerSnapshot`, `internal/logger#dualHandler.stderr`,
`internal/reedengine/render#Renderer.Draw`, `cmd/lyx#run`, `internal/shedrecipe#Lookup` (§7),
`internal/logger#init` (§3), `internal/logger_test#SomeName` (§2, the external test unit),
`internal/logger#Box` (§3, the type-parameter corner case's canonical form), a single-segment unit,
a deep unit, a Unicode identifier, and `_` as a member name. Each case asserts the **whole** parsed
`Glyph` — `Lang`, `Unit`, `Owner`, `Name` and `Params`, including that `Owner` is nil for a
package-level name and that `Params` is nil always.

*Rejects*, one case per `Reason` constant, asserting the reason and not the message text — and the
table must cover at least: no `#`; empty input; empty unit; empty member; leading `./`; leading `/`;
trailing `/`; `//`; a `.` segment; a `..` segment; a backslash; whitespace inside the unit; leading
and trailing whitespace on the whole input; a control character; a three-component member
(`a.b.c`); a member component that is not an identifier (leading digit); a keyword member (`func`,
and at least one more); `Box[T]`; `*dualHandler.Handle`; `(*dualHandler).Handle`;
`Renderer.Draw(int)` (a C# shape rejected under the Go alphabet); `internal/reedengine/render.Renderer.Draw`
(the `go doc` spelling §7 forbids); a member containing a second `#`. A test that iterates the
`Reason` constants and fails if any has no reject case keeps the table honest as the vocabulary
grows.

*Case sensitivity*: `internal/Logger#Foo` and `internal/logger#foo` are different glyphs and neither
folds into the other.

**`string_test.go` — the printer and the round trip.**

- `String()` on hand-built `Glyph` values: package-level name; `Type.Name` with a one-element `Owner`;
  the nil/empty `Params` distinction, asserting `nil` prints no parentheses and `[]string{}` prints
  `()`, plus a populated `Params` printing `(int,string)` — this last is the C#-shaped case that
  fixes the rule now, and it is a printer test only, with no `Parse` counterpart.
- **Round trip, both directions, over the whole accept table**, driven from the same table as the
  accept cases rather than a second hand-written list: `Parse(Go, s)` then `String()` equals `s` for
  every accepted `s`, and parsing a printed `Glyph` yields an equal `Glyph`. Because Parse is strict
  (see Decisions), this property is total over the accept set with no exceptions to carve out — if a
  case needs an exception, the strictness decision has been violated somewhere.
- `String()` never panics: exercise the zero `Glyph` and a `Glyph` with an over-deep `Owner`, asserting
  only that it returns.

**Not tests:** the no-cgo / stdlib-only guarantee is a verify command, `go list -deps ./glyph`, run in
the plan's verification step and expected to list only standard-library packages. It is not a Go test;
shelling out to the toolchain from a unit test is slow, environment-dependent, and duplicates a check
the plan already runs.

**Whole-task verification:** `go build ./... && go test ./...` green (cgo-enabled, as the engine
requires), plus `go vet ./...` and `go list -deps ./glyph`.

## Open questions for the hub (spec, not code)

These are recorded here rather than resolved in code, per the task's "do not edit `docs/glyph.md` to
fit the code; where the spec is unclear, ask" constraint. Neither blocks T1 from being written and
merged; the first blocks T3 from meeting its done criterion.

1. **Blocking for T3 — the root-package unit.** §2 does not say how a Go package in the repository
   root is spelled. T1 rejects both `.` and `""`. If Loomyard (or any repository the T3 round trip
   runs over) has a root package, §2 needs a sentence before T3 can claim "zero misses". Adopting
   either spelling later is a change in one place in `golang.go` plus one accept-table row.
2. **Non-blocking — whitespace in a unit, offered as a proposal.** §2 does not forbid whitespace in a
   unit segment, and nothing in the spec derives a ban: §6 accepts glyphs that need quoting ("quote
   them where a format cares" about C#'s `(`, `,` and `<`), so "must be safe unquoted" is not a rule
   the contract states. T1 rejects whitespace anyway, on the narrower ground that a unit with a space
   gives one directory two easily-confused spellings in a plan file. **The hub is free to accept the
   rule into §2 or to drop it**; this is not a gap in the spec so much as an offer to close one.
   Dropping it deletes one predicate in `golang.go` and one reject-table row, and nothing else moves.

## Q&A log

- **Q:** How is a Go package at the repository root spelled as a unit (§2 does not say)? **A:** [auto-pick] Reject it — a Go unit is a non-empty relative path with no `.`/`..` segments — and record it as a blocking spec question for the hub. **Why:** inventing `.` or `""` would be a second implementation of a rule the contract does not have, and the task forbids editing the spec to fit the code; T3's round-trip criterion is what forces the hub to answer.
- **Q:** Is Go's `Parse` tolerant of non-canonical spellings (`./unit`, `unit/`, `Box[T]`, `*T.M`, `(*T).M`)? **A:** [auto-pick] Strict — the canonical form is the only accepted form, each of those is a named reject. **Why:** §1 "each symbol has exactly one glyph. No short form, no alias"; §5 "resolution never guesses"; §7 says Go's own dotted names are not accepted where a glyph is expected. Consequence: the set of accepted non-canonical Go spellings is empty, so both round-trip directions are total.
- **Q:** Are Go keywords and the blank identifier `_` valid member names? **A:** [auto-pick] Reject the 25 keywords, accept `_`. **Why:** a keyword can never name a declaration, so rejecting is a free fixed-table check with a good message; `func _()` and `var _` are legal declarations `toc` can list, and rejecting `_` would break T3's "every declaration has a glyph" criterion — a `_` glyph resolving `ambiguous` is §5's business, not `Parse`'s.
- **Q:** How does `Params` distinguish a C# `Draw()` from a field, given the struct shape is fixed? **A:** [auto-pick] `nil` prints no parentheses; non-nil, including an empty slice, prints them. **Why:** §3 requires parentheses on every C# method including zero-parameter ones, and a `len>0` rule cannot spell `Draw()`. Decided now while it is free, because changing it after C# lands changes the one implementation everyone imports.
- **Q:** Is `Language` a string type or an iota int? **A:** [auto-pick] `type Language string` with `Go Language = "go"`. **Why:** the zero value must be invalid — with iota, `Go` would be the zero value and a forgotten argument would silently mean Go. Also matches `toc`'s existing `Language string` field.
- **Q:** How are the rejects modelled as errors? **A:** [auto-pick] One exported `*ParseError` carrying a closed exported `Reason` vocabulary; tests assert on `Reason`, never message text. **Why:** makes "every reject in the spec" enumerable and reviewable, mirrors `toc`'s closed-`Kind` convention, and keeps message wording in one place. Note: `glyph` cannot reuse `internal/quarryengine.ErrLanguageUnsupported` — `internal/` is unreachable from Loomyard's module, so the duplication is required.
- **Q:** Should the package export a validating constructor or `Validate()` for T3's benefit? **A:** [auto-pick] No — nothing beyond `Parse` and `String`. **Why:** YAGNI; T3 builds `Glyph` values directly and its whole-repository round trip is a stronger check than a constructor. Revisit in T3 if it asks for one.
- **Q:** Does `String()` validate or panic on an impossible `Glyph`? **A:** [auto-pick] No — total, pure printer; a hand-built `Glyph` is the builder's responsibility, documented on the type. **Why:** T3 constructs `Glyph` values outside `Parse`, so `String` must be defined for them; a panic in the one package Loomyard imports is the wrong failure mode.
- **Q:** One file or files per concern? **A:** [auto-pick] `doc.go`, `glyph.go`, `parse.go`, `golang.go`, `errors.go`. **Why:** matches `toc`'s layout and plan §12 T3's "one package, files per concern"; makes the language-free skeleton visible in the structure, and a second alphabet becomes one new file.
- **Q:** How are the tests laid out and traced back to the spec? **A:** [auto-pick] Table tests per concern, hand-transcribed, every case naming the `docs/glyph.md` section it came from. **Why:** the done criterion is per-example, so traceability has to be visible in the table; generating cases by reading `docs/glyph.md` at test time is fragile and against the package's no-file-reading rule.
- **Q:** The §1–§3 Python and C# examples must each be a test, but neither language may be defined. How? **A:** [auto-pick] Test the structural split alone on each of them (white-box), plus a test that any non-`Go` `Language` returns `unsupported_language`. **Why:** it is the half of those examples this task can honestly test, and it is exactly the claim the brief makes about the skeleton; skipping them fails the criterion and defining the constants is explicitly banned.
- **Q:** What characters may a unit segment contain? **A:** [auto-pick] Any except `/`, `#`, `\`, ASCII control characters and whitespace; segments non-empty and never `.` or `..`. **Why:** a Go directory may legally be named with Unicode, `.`, `-` or `+`, so a portable-filename class would reject valid units; the whitespace ban is this task's own proposal rather than anything the spec derives (§6 tolerates glyphs that need quoting), and is logged as a non-blocking spec question the hub may accept or drop.
- **Q:** What is a valid member component? **A:** [auto-pick] Go's own Unicode identifier rule — first rune `_` or `unicode.IsLetter`, later runes add `unicode.IsDigit` — with at most two `.`-separated components and no parentheses or brackets. **Why:** §3 fixes the shape; Go identifiers are Unicode and `toc` will emit them in T3, so ASCII-only would be wrong.
- **Q:** Does `Parse` special-case the `_test` external-test unit? **A:** [auto-pick] No — it is an ordinary path to the parser; the meaning is T4's. **Why:** the fixed struct has no flag for it, and telling the directory `logger_test` from the external test package of `logger` needs source, which this package never reads. The spec's example is still a test: it parses and round-trips.
- **Q:** Is "no cgo, stdlib only" a Go test or a verify command? **A:** [auto-pick] A verify command, `go list -deps ./glyph`. **Why:** the done criterion is already phrased as that command; shelling out to the toolchain from a unit test is slow and environment-dependent. Note the engine's `CGO_ENABLED=0` guard means the *repository* cannot build without cgo by design — only this package's dependency list is checked.
- **Q:** Split at the first or last `#`, and how is a second `#` handled? **A:** [auto-pick] First `#`; a `#` in the member half is then a Go member-alphabet reject, and no `#` at all is its own reject naming that a path is not a glyph. **Why:** §1 says the split needs no language; keeping "more than one `#`" out of the pre-split layer is what leaves the skeleton untouched when another alphabet is added.
- **Q:** Is input whitespace trimmed? **A:** [auto-pick] No — leading or trailing whitespace is a reject. **Why:** trimming is normalisation, and normalisation is the second spelling the strictness decision exists to prevent.
