# Discussion: Glyph self-form and the resolve contract (C1)

```yaml
task: Glyph self-form and the resolve contract (C1)
slug: glyph-self-form
status: discussing
parent: main
```

## Problem

Quarry addresses source symbols by glyph — `unit#member` — and addresses files and directories by
plain repository-relative path. Today `resolve` accepts both and tells them apart by a single rule:
a target containing `#` is a glyph, everything else is a path. That rule is cheap and it is wrong in
three ways at once. A file and a package have no glyph spelling at all, so a plan card whose
deliverable is a whole file is addressed in a different alphabet from a card whose deliverable is a
function. The `#`-containment test is implemented three times — once in `internal/cli/flags.go:153`
as `expand`'s own usage gate, once in `internal/cli/cli.go`'s `runResolve` on the argument as
given, and once in `internal/engine` on the repository-relative form. The last two disagree whenever
a directory name in the chain contains a `#`, a gap `internal/cli/doc.go` records in prose and
deliberately does not test; the first makes the two verbs disagree with each other, since
`expand <path>` is refused as a usage error with exit 2 while `resolve <path>` is answered, so the
same malformed argument gets two different exit codes depending on the verb. And `resolve`'s answer
names the path block `dir`, a key that repeats
the inner `dir` path field one level down and reads as "directory" when it also carries a single
file's entry.

**Why now:** the Loomyard adoption (`docs/roadmap.md`, External) is about to consume the envelope
and the glyph alphabet. The envelope has zero external consumers today, so these four adjustments
are free right now and breaking the day Loomyard is wired up. They were settled with the operator on
2026-09-05 and they land together because they touch the same files: `glyph/`, `internal/engine`,
`internal/cli`, `quarry/`, `docs/glyph.md`, `docs/rewrite-plan.md`.

## Scope

**In:**

- `glyph/`: accept an empty member as the **self form** (`unit#` names the unit or file itself);
  require exactly one `#` in a glyph string, with a dedicated rejection reason; retire the
  now-unreachable `member_empty` reason; make the no-separator rejection message actionable by
  naming the fix; add the `Self` constructor (D20) and update `glyph/doc.go`'s
  "exports no constructor" sentence.
- `internal/engine`: `Resolve` takes glyphs only — every target goes to `glyph.Parse`, and
  `isGlyphTarget` is deleted; a self glyph is answered with the listing block the old bare path
  produced; `Expand` rejects a self glyph with a typed error.
- `internal/engine/answer.go`: rename `ResolveResult.Dir` → `ResolveResult.Listing`, JSON key
  `dir` → `listing`. The inner `DirAnswer.Dir` path field is untouched.
- `internal/cli`: `runResolve` stops classifying and stops doing path arithmetic — the argument goes
  to the facade verbatim; `parseArgs`' `expand` usage gate is deleted (D19); `runTOC` (via
  `internal/repopath`) rejects a path target whose repository-relative form contains `#`.
- `internal/repopath`: the separator reject for path targets; delete the now-unused exported
  `RepoRelPath` **wrapper only** — the unexported `repoRelPath` stays, since `repoRelTarget` is built
  on it.
- `internal/engine/repo.go`: the new `ErrTargetHasSeparator` sentinel, beside the two target
  sentinels already declared there.
- `internal/mcpserver/toc.go`: `tocResult` gains the sentinel branch, with the same wording the CLI
  emits (D8).
- `quarry/`: `RenderResolveText` gains a listing branch keyed on the renamed field; the new sentinel
  is aliased alongside the other three.
- `docs/glyph.md`: the self form specified, with examples and rejects; §1/§2/§3/§5/§6/§7 updated.
- `docs/rewrite-plan.md` §4/§5: the "a target without `#` is a path" rule replaced by the new
  contract.
- `internal/cli/doc.go`: the contract-gap note deleted and replaced by the new rule's statement.
- Goldens under `docs/research/output-formats/after/` and the tests that pin them.

**Out:**

- Loomyard-side changes of any kind. This repository only.
- `toc`'s traversal, answer shape, depth rules, symbol rules, or ignore handling. The only
  behavioural change `toc` receives is the target-level separator reject (see the Decision, which
  records the tension with the task's "no behavior change to `toc`" constraint and why the reject is
  the minimum item 3 requires).
- The walk's `unitSpellable` rule. A `#`-bearing directory encountered *below* a listed target is
  still listed without symbols, exactly as today — the new grammar rule keeps it unspellable through
  a different reason, not a different outcome.
- Symbol glyphs. The T3 round trip must keep passing unchanged; nothing about `unit#member`
  resolution moves.
- Any new query, verb, flag, or MCP tool. The MCP server exposes `toc` only.
- Python and C# **extractors**. They are specified prose, not code. Their *documented* self form is
  in scope, because `docs/glyph.md` §2 and §3 must say what `unit#` means under each unit definition
  — see D15; what is out of scope is implementing either alphabet.
- The repository root's own unit. Addressing the root as a glyph stays impossible (see Risks).
- Caching, indexing, performance work.

## Decisions

### D1 — The self form is an empty member, not a new field

- **Decision:** a glyph whose member half is empty is the self form. In `glyph.Glyph` that is
  `Owner == nil && Name == "" && Params == nil` with `Unit` set. Add one predicate,
  `func (g Glyph) IsSelf() bool { return len(g.Owner) == 0 && g.Name == "" }`. `String()` is not
  changed: it already prints `Unit + "#"` for that value, which is the canonical form the contract
  requires.
- **Rationale:** the contract sentence — **for Go**, whose unit is spelled as a repository-relative
  path — is "removing the trailing `#` yields the plain path, that is the whole conversion, both
  directions". Representing self as the absence of a member makes that sentence true of the code by
  construction, in both `Parse` and `String`, with no third state to keep consistent. `String()`
  needing no edit is the evidence the representation is the right one. The representation itself is
  alphabet-independent — an empty member is an empty member in any alphabet — but the
  *path* reading of the left half is Go-only, per D15's §2 edit; do not restate it unscoped.
- **Rejected:** a `Self bool` field — introduces a value that can disagree with `Name` (self and
  named at once) and forces `String()` to branch. A sentinel `Name` value such as `"."` or `"self"` —
  a name a real Go symbol could legally carry, and it would print as `unit#self`, breaking the
  removing-the-`#`-yields-the-path rule.

### D2 — `Parse` accepts the empty member before the member alphabet runs

- **Decision:** in `parseGo`, after `checkGoUnit` succeeds, `member == ""` returns
  `Glyph{Lang: Go, Unit: unit}` immediately. `checkGoMember` is therefore never called with an empty
  string, and its `member == ""` branch is deleted along with `ReasonMemberEmpty`.
- **Rationale:** the unit half must still be validated for a self glyph — `#`, `.#`, `a//b#` are all
  rejects — so the gate belongs after `checkGoUnit` and before the member alphabet. Keeping a
  constant in a vocabulary documented as closed and exhaustively enumerated in `Reasons`, when
  nothing can produce it, would make that documentation false.
- **Rejected:** keeping `ReasonMemberEmpty` as reserved for a future alphabet — the *empty member* is
  alphabet-independent (it is the absence of a member, not a member syntax question), so every
  alphabet admits it and no alphabet can ever produce the reason. This is a narrower claim than "the
  self form is language-free", which is **false** and is corrected in D15: what `unit#` *means*, and
  whether a file has a self glyph at all, differ per language because §2 defines the unit
  differently per language. Only the parse-side rule — an empty member is never an error — is
  shared. Handling the empty member inside
  `checkGoMember` — it would have to return a "this was self" signal alongside `owner`/`name`,
  adding a third return channel to a function whose contract is validate-and-split.

### D3 — Exactly one `#`; a second one is its own rejection reason

- **Decision:** `Parse` counts separators before splitting. Zero `#` is `ReasonNoSeparator`; more
  than one is a new `ReasonMultipleSeparators Reason = "multiple_separators"`, text `a glyph has
  exactly one "#"; a unit or member component may not contain one`, `Detail` carrying the input.
  `Reasons` and `reasonText` gain the entry; the vocabulary stays at sixteen values because
  `ReasonMemberEmpty` leaves in the same edit. `splitGlyph` keeps `strings.Cut` and is now called
  only when the count is exactly one.
- **Rationale:** this is task item 3 — "a unit or path segment containing `#` is an explicit error,
  never a silent reclassification" — stated once, in the grammar, where every surface inherits it.
  Today `a#b#c` surfaces as `member_bad_rune` on a `#` rune, which is technically true and tells the
  caller nothing about the separator's role. `checkGoUnit`'s doc comment ("There is no `#` check
  here, because the split that produced unit already consumed the first `#`") must be rewritten to
  cite the count rule instead.
- **Rejected:** leaving it as `member_bad_rune` — that is the silent-ish reclassification the item
  bans, and it cannot fire for the unit half at all. A reason per half (`unit_separator`,
  `member_separator`) — the string is rejected before either half exists; one reason for one rule.

### D4 — The bare-path rejection is `no_separator`, produced by `glyph.Parse`, with an actionable message

- **Decision:** `internal/engine`'s `isGlyphTarget` is deleted and `resolve` hands every target to
  `resolveGlyphTarget`. A bare path therefore fails `glyph.Parse` with the existing
  `ReasonNoSeparator`, and that becomes a `ResolveResult` carrying `target`, `error`, `reason:
  "no_separator"` and no `status` — the pre-resolution rejection shape that already exists.
  `reasonText[ReasonNoSeparator]` is rewritten to name the fix, and `Parse` sets `Detail` to
  `s + "#"` so the message shows the exact spelling the caller wanted. The rendered line reads:
  `glyph: parse "internal/logger" as go: a glyph needs a "#"; a path is addressed as its own glyph by
  appending one to its repository-relative form (internal/logger#)`.
  The clause "to its repository-relative form" is **required, not decorative** — see the Risks entry
  on cwd-relative targets. `Detail` echoes the caller's argument verbatim, so for a cwd-relative
  argument it shows `logger.go#`, which is not a working glyph; without the clause the hint misleads
  exactly the caller it is aimed at. This sentence is the authoritative wording, and it is what
  Testing item 2's one message-shape test pins.
- **Rationale:** one implementation of the grammar. `expand`'s *engine* half already routes a bare
  path through this same reason, so once D19 deletes the CLI's own `expand` gate the two verbs agree
  by construction rather than by two matching checks. Carrying the fix in `Detail` means the
  actionable half is data, not prose a caller has to parse. **Note the dependency:** this claim is
  false today — `internal/cli/flags.go:153` refuses `expand <path>` with a usage error and exit 2
  before `glyph.Parse` is ever reached — and it becomes true only because D19 removes that gate. D4
  and D19 must land together or the rationale here does not hold.
- **Rejected:** a new `ReasonBarePath` — the condition is identical to no-separator, so a second
  constant would be one condition with two names. Rejecting in `internal/cli` before the engine —
  the facade is a public API and `Resolve` is multi-target; a check only the CLI performs would let
  a batching caller through with a bare path.

### D5 — A pre-resolution rejection stays a payload with exit 1 (existing D2 rule, restated not reinvented)

- **Decision:** the bare-path rejection and the multiple-separator rejection are rendered as the
  ordinary resolve payload on **stdout** — JSON `{"target":…,"error":…,"reason":…}` with no `ok` key,
  or the text line `<target> error <reason>: <message>` — and the process exits **1**. The failure
  envelope (`{"ok":false,…}`) is not used. No code change is needed for this: `codeForResolveResult`
  already maps the empty status to `exitNegative`, and `RenderResolveText`'s branch 1 already renders
  it.
- **Rationale:** this is the decision `internal/cli/doc.go` and `docs/rewrite-plan.md` §5 already
  record — "a negative resolution outcome … or, for the pre-resolution case, an error field of its
  own — is a payload … rendered and written to stdout exactly as a positive answer is". The task's
  own wording says to follow the existing D2 decision and not reinvent it. Recording it here so
  mill-plan does not re-derive it.
- **Rejected:** the error envelope with exit 2 — reserved for usage errors with no payload, and a
  rejected target does have a payload naming itself.

### D6 — `resolve` performs no path arithmetic at all

- **Decision:** `runResolve` loses its `strings.Contains(target, "#")` branch and its
  `repopath.RepoRelPath` call; `req.target` goes to `repo.Resolve` verbatim. The exported
  `repopath.RepoRelPath` wrapper, whose only caller this was, is deleted (`repoRelPath` stays,
  unexported, as `repoRelTarget`'s implementation).
- **Rationale:** `internal/cli/doc.go` already states the rule — "a glyph's unit is
  repository-relative by the grammar's own definition; cwd arithmetic on it would corrupt it, the
  same way rebasing a remote URL against a local directory would". Once every `resolve` target is a
  glyph, that rule covers every target, and the CLI's classification disappears rather than being
  fixed. **Consequence, deliberate:** `quarry resolve ./logger.go#` run from inside `internal/` does
  not work; the self glyph's unit is repository-relative like every other unit, so the spelling is
  `internal/logger/logger.go#` from anywhere. `toc` keeps cwd-relative targets, because paths are its
  domain.
- **Rejected:** cwd-resolving the unit half of a self glyph only — that would make one glyph string
  mean two different units depending on where the shell sat, which is the exact property the glyph
  contract exists to deny.

### D7 — This closes the `internal/cli/doc.go` contract gap by construction

- **Decision:** delete the "Known contract gap" paragraph in `internal/cli/doc.go` and replace it
  with a statement of the new rule: `resolve` takes glyphs, `toc` takes paths, classification happens
  exactly once and it is `glyph.Parse` doing it; a `#` in a path segment is an explicit error at both
  verbs, never a reclassification. Also rewrite the paragraph above it — the one describing
  `strings.Contains` classification — since that code is gone.
  **The sentence is only writable because all three classifiers go:** `runResolve`'s (D6),
  the engine's `isGlyphTarget` (D4), and `parseArgs`' `expand` gate (D19). If any one of them
  survives, this doc sentence is false the day it is written — that is the acceptance condition for
  D7, not a nicety.
- **Rationale:** the gap was two of three classifications disagreeing about a path, while the third
  made the two verbs disagree about a bare path's exit code. With no surface classifying, there is
  one classifier and nothing to disagree with. The gap is not narrowed or tested-around; it is
  structurally absent.
- **Rejected:** threading a target class from the CLI into the engine (the fix the old note proposed)
  — unnecessary now, and it would add the engine signature change the note called out-of-scope.

### D8 — `toc` rejects a `#`-bearing path target, at `internal/repopath`

- **Decision:** `repoRelTarget` returns a new sentinel `ErrTargetHasSeparator` when any segment of
  the cleaned repository-relative path contains `#`. The sentinel is declared in
  **`internal/engine/repo.go`**, beside `ErrTargetOutsideRepo` (line 46) and `ErrTargetNotFound`
  (line 50) which is where those two actually live — *not* in `errors.go`, whose header reads "the
  one error sentinel the engine's subpackages share" and which holds only `ErrLanguageUnsupported`.
  Declaring it there would falsify that header; `repo.go` needs no header change, since the two
  target sentinels are already its neighbours. It is aliased through `quarry/quarry.go` as those two
  are, so `errors.Is` stays transitive for an external caller. Sentinel text:
  `engine: target contains the glyph separator "#"`, matching the `engine: target …` style of its
  two neighbours.
  Both surfaces need an **explicit branch**, because both map `RepoRelTarget` errors with a single
  `errors.Is(ErrTargetOutsideRepo)` test and an `else` that says "internal error":
  - `internal/cli/cli.go`'s `runTOC` gains a branch mapping it to **exit 2** (usage) with the message
    `target contains the glyph separator "#": <target as given>`.
  - `internal/mcpserver/toc.go`'s `tocResult` (the `RepoRelTarget` error block at lines 107–112)
    gains the matching branch, returning `errorResult` with the **same sentence** the CLI emits, so
    the two surfaces cannot drift. Without it the sentinel falls into that block's
    `"internal error: " + err.Error()` else-arm and a malformed user target is reported to an MCP
    client as a server fault, carrying the package-namespaced sentinel text — and the
    `toc_errors_test.go` item in Testing would pass against that wrong shape.
- **Rationale:** item 3 says "everywhere". Leaving `toc` to list `a#b/` happily while the contract
  declares such a segment an explicit error would reintroduce exactly the silent acceptance the item
  bans, on the one verb that still takes paths. Exit 2 rather than 1: the target is malformed, not
  missing or out of scope, which is what separates it from the two existing exit-1 path failures.
  **Tension, recorded deliberately:** the task's constraint says "no behavior change to `toc` beyond
  documentation wording", and this *is* a behaviour change — a target that succeeds today fails
  after. The constraint is read as protecting `toc`'s traversal, answer shape and defaults; item 3's
  "everywhere" cannot be satisfied without this one reject. If a reviewer prefers the literal reading
  of the constraint, the reject is the single removable piece and nothing else in this task depends
  on it.
- **Rejected:** rejecting inside `internal/engine`'s `TOC` — `repopath` is where both `toc` callers
  already normalise, and the engine takes an already-repository-relative path. Silently skipping such
  a target — that is the silence item 3 bans.

### D9 — What the walk does is unchanged

- **Decision:** no edit to `walk.go`, `unitFor`, or `unitSpellable`. A directory named `a#b`
  encountered below a listed target is still listed, its files still get `name`/`header`/`test`/
  `generated`, and no file entry in it carries symbols.
- **Rationale:** `unitSpellable` probes `unit + "#x"` through `glyph.Parse`. Under D3 that string has
  two `#` and is rejected as `multiple_separators`; today it is rejected as `member_bad_rune`. The
  reason changes, the boolean does not, so the emitted answer is byte-identical. That is what keeps
  the T3 round trip true by construction: `toc` never emits an `id` for such a directory, so there is
  never an `id` that fails to resolve. The asymmetry with D8 — naming `a#b` as a target is an error,
  encountering it below one is a silent no-symbols listing — is intentional and must be stated in
  `docs/glyph.md`: the contract governs what a caller may *name*, and `unitSpellable`'s own doc
  comment already governs what the walk may *mint*.
- **Rejected:** pruning `#` directories from the walk — a behaviour change to `toc`'s traversal, which
  is out of scope, and it would hide files that are legitimately listable.

### D10 — A self glyph resolves through the existing listing producer and carries `id`

- **Decision:** `resolveGlyphTarget`, on `g.IsSelf()`, calls the existing path-answer helper with
  `g.Unit` and returns `ResolveResult{Target: <argument verbatim>, ID: g.String(), Status: …,
  Listing: …}`. `Unit` is never set on a self result, on any status. The helper is renamed
  `resolveSelfTarget` and its doc comment updated; its three-way disposition (nil error → found +
  listing; `ErrTargetNotFound` → not_found, no listing; `ErrTargetOutsideRepo` → the error field) and
  its `Symbols: &false` option are kept exactly. `<file>#` returns the enclosing directory's answer
  holding the one file entry — the shape `TOC` already produces for a file target, which is what a
  bare path returned before. `<dir>#` returns the directory's answer.
- **Rationale:** same envelope, same statuses, per the task. Setting `ID` keeps parity with every
  other glyph answer and with `Symbol.ID`; a self glyph that alone had no `id` would be the one glyph
  the round-trip discipline could not talk about. `Unit` is suppressed because for a self glyph the
  unit *is* the thing being asked about — `unit: not_found` beside `status: not_found` would restate
  the answer.
- **Rejected:** returning the old path-answer shape with no `id` — makes the self form a
  second-class glyph. Reimplementing the listing lookup for self glyphs — the existing helper already
  inherits the explicitly-named-gitignored-target rule, the never-follow-a-symlink rule and the
  empty-string-and-dot-mean-the-root rule for free; duplicating it would drift.
- **Note:** `resolveSelfTarget`'s `ErrTargetOutsideRepo` branch becomes unreachable, because
  `checkGoUnit` rejects `.` and `..` segments before a unit can escape. Keep the branch as a
  defensive arm and say so in its doc comment rather than deleting a sentinel translation.

### D11 — `expand` of a self glyph is a typed failure, not a `not_found`

- **Decision:** add `SelfGlyphError{ID string}` to `internal/engine/expand.go` beside
  `NotATypeError`, with `Error()` returning `engine: expand <id>: not a type, a self glyph names the
  unit or file itself`. `expand` returns it immediately after a successful parse when `g.IsSelf()`,
  before any directory or symbol work. Alias it through `quarry/quarry.go` as `NotATypeError` is
  aliased, so `errors.As` works for an external caller. `codeForExpandError` returns `exitNegative`
  for it, and `runExpand` gains a branch producing `expand <target>: not a type, self` on stderr with
  the failure envelope on stdout — the same shape the `NotATypeError` branch produces.
- **Rationale:** the task requires `ok: false` naming the kind, "same as any non-type glyph today".
  Without the gate the answer would fall out as `not_found` (a self glyph matches no symbol), which
  says the name is free when the truth is that the verb does not apply. A gate before resolution
  means no directory is read for a question that has no answer.
- **Rejected:** adding a sixth `Kind` value such as `"self"` and reusing `NotATypeError` — `Kind` is
  documented as the closed vocabulary a `Symbol`'s `Kind` field is drawn from and the five values
  `toc` emits; a self glyph is not a symbol, and widening a symbol vocabulary to describe a non-symbol
  would make the "five values, no other value is valid" doc comment false. Answering `not_found` —
  contradicts the task.

### D12 — `SpansOf` of a self glyph returns an empty slice

- **Decision:** no code change to `SpansOf`. A hand-built self `Glyph` round-trips through
  `glyph.Parse` successfully (D2), reaches `matchesFor`, and matches nothing, so the existing
  empty-slice-with-nil-error contract applies. Add a sentence to `SpansOf`'s doc comment saying so.
- **Rationale:** `SpansOf` has no status vocabulary; "nothing matches" is exactly what an empty slice
  with a nil error means there, and a self glyph names no declaration. `SpansOf` is what the T3 round
  trip is written against and this task must not edit it for behaviour.
- **Rejected:** returning an error — would make `SpansOf` the one place with a self-glyph opinion the
  status vocabulary above it does not share.

### D13 — `dir` → `listing`: outer field only, one rename, no MCP surface

- **Decision:** `ResolveResult.Dir *DirAnswer` `json:"dir,omitempty"` becomes
  `ResolveResult.Listing *DirAnswer` `json:"listing,omitempty"`, in the same field position so key
  order in the emitted object is unchanged. `DirAnswer.Dir` — the inner repository-relative path
  string — keeps its name and its `dir` key. The facade needs no edit (`ResolveResult` is an alias).
  `quarry/text.go` updates its two `r.Dir` references and its doc comment. **The MCP server has no
  `resolve` tool** — it exposes `toc` only — so the task's "MCP text view" clause has no referent
  here; the only text surface touched is `quarry.RenderResolveText`, and because the outer field name
  is never printed, the rename changes zero bytes of text output. Record this in the plan so nobody
  hunts for an MCP change that does not exist.
- **Rationale:** `listing` says what the block is — a listing of one file or of a directory's
  contents — where `dir` claimed "directory" while also carrying single-file answers, and repeated
  the inner key's word one level up.
- **Rejected:** renaming the inner `DirAnswer.Dir` too — it is shared with `toc`'s own answer, which
  is out of scope, and `dir` is the right word for a path that is always a directory.

### D14 — `RenderResolveText` gains a listing branch ahead of the glyph branch

- **Decision:** insert a branch before the existing `r.ID != ""` branch: when `r.Listing != nil`,
  emit line 1 as `r.ID + " " + status` followed immediately by `strings.Join(dirBlocks(*r.Listing),
  "\n")`. The existing `default` arm is kept but reduced to line 1 alone (`r.Target + " " + status`),
  documented as the guard for an externally-constructed value, since the engine can no longer produce
  a path result without an `ID`. A self glyph that is `not_found` has a nil `Listing` and falls to the
  glyph branch, which prints `<path># not_found` with no unit suffix (D10 leaves `Unit` empty, and
  that branch already guards on `r.Unit != ""`).
- **Rationale:** branch order is the whole of it — the glyph branch fires on `r.ID != ""`, which a
  self glyph now also satisfies, and would print a bare status line with no listing. Line 1 uses
  `r.ID` rather than `r.Target` so the canonical trailing `#` is what the text view shows, matching
  what the JSON `id` carries.
- **Rejected:** keying the new branch on a status or on a heuristic about the target string — the
  pointer being non-nil is the fact the renderer actually needs.

### D15 — `docs/glyph.md` edits, and every example is a test

- **Decision:** six sites.
  §1: the form line gains the self form and the sentence "any glyph splits at its first `#`" becomes
  "a glyph contains exactly one `#`, and splits there"; the example table gains
  `internal/reedengine/render#` (the package itself) and `internal/reedengine/render/focus.go#` (the
  file itself).
  §2 (**added to this inventory — it is the section the self form contradicts**): the unit table
  defines the Go unit as the *package directory*, so `internal/reedengine/render/focus.go#` names a
  left half that is not a unit under §2 as written. §2 gains a paragraph stating that in the self
  form the left half is the thing's own **repository-relative path or unit name**, whichever the
  language spells, and a per-language row for what `unit#` means:
  Go — a package directory (`internal/reedengine/render#`) *or a file* (`…/focus.go#`), both spelled
  as repository-relative paths, since Go's unit already is one;
  Python — the module or package itself (`loomyard.engine.layout#`), which *is* the file, so Python
  has no separate file self glyph;
  C# — the namespace itself (`Loomyard.Engine.Layout#`); C# has no file-level unit, so it has no file
  self glyph at all.
  §3: a short paragraph, above the per-language sections, stating that an empty member is the self
  form in every alphabet — that much *is* shared — and that **for Go** removing the trailing `#`
  yields the plain repository-relative path, both directions, with no other conversion. The
  path-conversion sentence is explicitly scoped to Go: it is true only because Go's unit is spelled
  as a repository-relative path, and stating it unscoped would assert something false of a Python
  dotted module and a C# namespace.
  §5: replace the closing paragraph ("`resolve` also accepts what is not a glyph … told apart by
  having no `#`") with the new contract — `toc` takes paths, `resolve` takes glyphs, a bare path is
  rejected with a message naming the fix, and a self glyph answers with the `listing` block and the
  same statuses. Add the rule that a `#` in a path segment is an explicit error at both verbs, and
  the D9 asymmetry (what a caller may name vs. what the walk may mint).
  §6: add that a trailing `#` is safe in the formats already listed and is never optional — the
  canonical form keeps it.
  §7: rewrite the "File paths are not glyphs" bullet, which currently states the retired no-`#` rule.
  Every example and every reject named in these sections gets a row in a test table (see Testing).
- **Rationale:** `docs/glyph.md` is the contract — "anything not stated here is not part of the
  contract" — so the self form is not real until it is written there. The task's done-when makes each
  example a test, which is the discipline that keeps the document from drifting from the parser.
- **Rejected:** a new numbered section for the self form — it is a property of the one form, not a
  sixth topic; splitting it out would let §1's form line stay silent about it.

### D16 — `docs/rewrite-plan.md` §4/§5

- **Decision:** three sentences in §4, not one.
  (a) The bullet "A target without `#` is a path; with `#`, a glyph" (line 80) is replaced by
  "`toc` takes paths; `resolve` takes glyphs; a trailing `#` addresses a unit or file as itself".
  The same bullet's neighbouring sentence about `toc` listing every non-gitignored file is untouched.
  (b) **The next bullet (line 81), "Glyphs as keys in every output, under `id`. What `toc` lists is
  what `resolve` takes", becomes false and is rewritten.** `toc`'s file and directory entries carry
  no `id` — only `Symbol` does — so once `resolve` takes glyphs only, a consumer holding a `toc`
  listing cannot feed a file entry back to `resolve` without building `dir + "/" + name + "#"` by
  hand, which is exactly the printing the one-implementation constraint forbids outside package
  `glyph`. The replacement states how the round trip is actually made: a symbol entry's `id` is
  already a glyph; a file or directory entry's self glyph is built from its path with the new
  `glyph.Self` constructor (D20), never by concatenation.
  (c) §5's `**resolve <glyph|path>**` heading becomes `**resolve <glyph>**`, and its body gains one
  sentence: a bare path is rejected pre-resolution with a message naming the fix, and `<path>#` is how
  a whole file is a plan target. §5's `toc` line is untouched apart from noting the separator reject.
- **Rationale:** the plan says how quarry implements the contract; leaving the retired rule in it
  would give a reader two contradictory sources. (b) matters more than it looks: the Loomyard
  adoption this task is timed for is precisely the consumer that walks a `toc` listing and resolves
  what it finds, so a round-trip sentence that no longer holds would send the first external
  consumer straight into hand-concatenation.
- **Rejected:** adding an `id` key to `FileEntry`/`DirAnswer` so the round trip needs no constructor
  — that is a change to `toc`'s answer shape, which is out of scope and which the Constraints
  protect. Rewriting §5's performance numbers or the phase-2 parking — untouched by this task.

### D19 — `expand`'s CLI usage gate is deleted; both verbs reject a bare path identically

- **Decision:** delete the `verb == "expand" && !strings.Contains(req.target, "#")` gate in
  `internal/cli/flags.go:153` and its usage message. A bare path given to `expand` then reaches
  `glyph.Parse` in the engine, fails with `no_separator`, and exits **1** through the existing
  `codeForExpandError` path (which already returns `exitNegative` for a `*glyph.ParseError`) with
  D4's actionable message. `internal/cli/usage.go` is checked for wording that promises the old
  usage error and updated if it does.
- **Rationale:** this is the third `#`-containment classifier, and it is the one that makes the two
  verbs disagree with each other: today `expand <path>` is exit 2 with a usage message while
  `resolve <path>` is exit 1 with a payload, for the same malformed argument. D7's mandated doc.go
  sentence ("classification happens exactly once and it is `glyph.Parse` doing it") cannot be written
  truthfully while this gate stands, and D4's rationale ("`expand` already routes a bare path through
  this same reason") is false at the CLI without this deletion. Deleting it also gives `expand` the
  actionable repository-relative hint, which the hand-written usage string does not carry.
  **Consequence, deliberate and breaking:** `expand <bare-path>` moves from exit 2 to exit 1, and
  from stderr-usage to the payload-plus-message shape. Free now, and named in the squash message
  (D18).
- **Rejected:** keeping the gate for a friendlier early error — it buys a message the grammar now
  produces better, at the cost of the one-classifier property this whole task is built on. Changing
  the gate to match `resolve`'s exit code instead of deleting it — that keeps a second classifier
  alive and leaves D7's sentence unwritable.

### D20 — `glyph.Self`, the constructor that keeps the round trip inside the one implementation

- **Decision:** add `func Self(lang Language, path string) (Glyph, error)` to package `glyph`. It
  validates `path` against `lang`'s unit rules exactly as `Parse` does — by delegating, so there is
  no second copy of the unit alphabet — and returns the self `Glyph` on success, a `*ParseError` on
  failure. `glyph/doc.go`'s sentence "this package exports no constructor and no Validate method" is
  updated in the same edit, since it stops being true.
- **Rationale:** D16(b) needs a consumer holding a `toc` file entry to obtain that file's self glyph
  without printing `dir + "/" + name + "#"` itself. The Constraints say there is one implementation
  of the glyph grammar and Loomyard's parser imports it rather than re-implementing printing; a
  documented round trip that can only be completed by hand-concatenation outside the package would
  contradict that on the first external consumer. One small constructor closes it.
- **Rejected:** telling consumers to call `Parse(lang, path + "#")` — that *is* concatenation, just
  relocated into every consumer, and it produces a confusing error when `path` already contains a
  `#`. Adding `id` to `FileEntry` — a `toc` answer-shape change, out of scope (see D16).

### D17 — Goldens: split by whether the case needs a Loomyard checkout

- **Decision:** the four positive cases go into the `docs/research/output-formats/after/` table in
  `internal/cli/after_test.go`, which is already Loomyard-gated and `-update`-regenerable. Method
  glyph and type glyph already exist there (`resolve-method.txt`, `resolve-glyph.txt`,
  `resolve-glyph-text.txt`) and need only re-verification. `resolve-path.txt` is **renamed**
  `resolve-self-file.txt`, its command line retargeted to `quarry resolve
  internal/logger/logger.go#`, and its `"dir"` key regenerated as `"listing"`; a new
  `resolve-self-dir.txt` covers `quarry resolve internal/logger#`, and text views accompany both.
  `after/INDEX.md` is updated in the same edit — it is the table's own description and the
  `after_test.go` header already forbids putting the description anywhere else.
  The two rejection cases — bare path and separator-bearing segment — go into `internal/cli`'s own
  table tests (`cli_test.go` / `glyph5_test.go`), **not** into `after/`, because they read no source
  and must run on a machine with no Loomyard checkout.
- **Rationale:** `internal/cli/glyph5_test.go`'s own header already records that the `after/`
  evidence goldens cannot carry every case; the same split applies here. Regenerating an `after/`
  golden requires the pinned checkout, so a case that does not need it should not acquire the
  dependency.
- **Rejected:** putting all six in `after/` — makes two pure-grammar tests skip on most machines.
  Keeping `resolve-path.txt`'s filename — the file no longer shows a path target and the name would
  lie.

### D18 — The squash message records the free-breakage window

- **Decision:** the squash-merge message states that the resolve envelope had no external consumers
  at the time of the change, and names the five breaking pieces: `resolve` no longer accepts a bare
  path, `dir` is now `listing`, `expand <bare-path>` moved from exit 2 to exit 1 (D19),
  `member_empty` is gone from the `Reason` vocabulary, `repopath.RepoRelPath` is gone.
- **Rationale:** the task asks for it, and it is the record that makes the breakage defensible when
  Loomyard is wired up later.
- **Rejected:** a `CHANGELOG` entry — the repository has none, and inventing one is out of scope.

## Technical context

**Module map for this task.**

- `glyph/` — pure Go, standard library only, no cgo, no dependencies. `parse.go` holds `Parse` and
  `splitGlyph`; `golang.go` holds `parseGo`, `checkGoUnit`, `checkGoMember` and the identifier/keyword
  helpers; `errors.go` holds the `Reason` vocabulary, the `Reasons` slice, the `reasonText` map and
  `ParseError`; `glyph.go` holds `Glyph` and the total `String()` printer. `Parse` currently checks
  language, then UTF-8, then splits, then dispatches to `parseGo`. The separator count check (D3)
  goes between the UTF-8 check and the split.
- `internal/engine/resolve.go` — `isGlyphTarget` (deleted by D4), `resolveGlyphTarget`,
  `resolvePathTarget` (renamed by D10), `Resolve`/`resolve`, the `unitMemo`, `unitDirs`,
  `symbolsOfUnit`, `matchesFor`, `statusForMatches`, `SpansOf`. `statusForMatches` is shared with
  `expand` and must not be touched — it is the single place a match set becomes a status.
- `internal/engine/expand.go` — `Expand`/`expand`, `NotATypeError` and its kind gate. The self gate
  (D11) goes immediately after the `glyph.Parse` call, before `dirsOf`/`symbolsOf`.
- `internal/engine/answer.go` — `ResolveResult` (the `Dir` → `Listing` rename), `DirAnswer`,
  `FileEntry`, `Symbol`, `Kind`, `Status`. The file's header comment states that the emitted key set
  is closed and no field is renamed without a corresponding Shared Decision change — this task *is*
  that change and the header should say so.
- `internal/engine/walk.go` — `unitFor` and `unitSpellable`. **Read but do not edit** (D9);
  `unitSpellable`'s doc comment lists the rejections it covers and gains `multiple_separators`.
- `internal/engine/repo.go` — **this is where the two target sentinels live**:
  `ErrTargetOutsideRepo` (line 46) and `ErrTargetNotFound` (line 50), beside `resolveTarget`.
  `ErrTargetHasSeparator` (D8) is declared here, with them.
- `internal/engine/errors.go` — holds `ErrLanguageUnsupported` **and nothing else**; its header reads
  "the one error sentinel the engine's subpackages share". Do not add the new sentinel here: it would
  falsify that header and force a header rewrite for no gain. **Read but do not edit.**
- `internal/repopath/target.go` — `repoRelPath` (unexported, **retained**), `RepoRelPath` (the
  exported wrapper, deleted by D6), `repoRelTarget`, `RepoRelTarget`. The separator check goes in
  `repoRelTarget`, after the escape check, so an escaping target still reports the escape.
- `internal/cli/flags.go` — `parseArgs`, whose `expand` usage gate at line 153 is deleted by D19.
  Check `internal/cli/usage.go` in the same edit for wording that promises that error.
- `internal/cli/cli.go` — `runTOC` (line ~301), `runResolve` (line ~359), `runExpand` (line ~403),
  and the four exit-code mappers `codeForTOCError`, `codeForResolveResult`, `codeForExpandAnswer`,
  `codeForExpandError`. The `fail` helper is the one site that writes a failing message.
- `internal/cli/doc.go` — the package doc carrying the classification paragraph and the contract-gap
  note (D7).
- `quarry/` — `quarry.go` (aliases and sentinels), `render.go` (`RenderResolveJSON` and the shared
  `renderJSON` byte contract: two-space indent, one trailing newline, no HTML escaping),
  `text.go` (`RenderResolveText` at ~line 230, `dirBlocks`, `writeSymbolLine`, `normalizeProse`).
  The facade adds no behaviour and holds no state.
- `internal/mcpserver/` — exposes the `toc` tool only, and reaches `repopath.RepoRelTarget` for its
  own target conversion. It inherits D8's `repopath` reject, but **does need its own edit**:
  `tocResult`'s `RepoRelTarget` error block (lines 107–112) is one `errors.Is(ErrTargetOutsideRepo)`
  branch plus an `"internal error: " + err.Error()` else-arm, so the new sentinel needs an explicit
  branch there or it surfaces as a server fault. See D8. `errorResult` is the helper that block uses.

**Gotchas found during exploration.**

- `Glyph.String()` needs no change for the self form: with `Owner` nil, `Name` `""` and `Params` nil
  it already emits `Unit + "#"`. Verify this with a test rather than by reading, because `String`
  builds `parts` from `Owner` plus `Name` and a future edit could start trimming empties.
- `unitSpellable` probes `unit + "#x"`. This is why D3 must be a `Parse`-level count check and not a
  `checkGoUnit`-level rune check: a rune check inside `checkGoUnit` would also fire, but the count
  check is what makes the *reason* right for both the probe and a real two-`#` target.
- `checkGoMember` runs its pointer/parens/brackets checks *before* its `member == ""` check. Deleting
  the empty check (D2) is safe because those three scans over an empty string find nothing, but the
  ordering comment in that function should not be disturbed — it exists for the 7a-before-7b/7c rule
  further down.
- `resolvePathTarget` sets `Symbols: &false` explicitly rather than relying on the per-target default,
  which would switch symbols on for a file target. Preserve that when renaming — a self glyph answers
  *where* a thing is, and *what is inside it* is `toc`'s question.
- `internal/cli/after_test.go` pins twelve files and declares `-update`; `internal/cli/loomyard_test.go`
  declares the `-update` flag and `loomyardPin`, and fails loudly on a checkout at the wrong commit
  rather than skipping. Regenerating any `after/` golden needs `LADDER_LOOMYARD_REPO` at that pin.
- `ResolveResult`'s existing doc comments carry three recorded contract gaps (the external-test-unit
  collision on `Unit`, the language marker on `Candidates`, the symlink-reached unit on
  `dirExists`). None of them are this task's to close; do not delete them while editing neighbouring
  prose.
- Comment density in this repository is high and deliberate: doc comments carry the rationale and the
  rejected alternatives. Every edit here is expected to update the surrounding comment, not just the
  code. `docs/rewrite-plan.md` and the package docs are treated as part of the contract surface.

## Constraints

- No `CONSTRAINTS.md` exists at the hub root.
- `CLAUDE.md`: this is a Go repository — do not introduce Python.
- Go only; no Loomyard-side changes in this repository.
- The breaking change is deliberate and free now; the squash message records that the envelope had
  no external consumers at the time (D18).
- No behaviour change to `toc` beyond documentation wording — read as protecting `toc`'s traversal,
  answer shape and defaults; the one exception, and the reasoning for it, is D8.
- The T3-style round trip must still hold: every `id` that `toc` emits resolves to exactly its span,
  zero misses, zero extras. Symbol glyphs are unchanged by this task.
- `glyph/` stays pure Go, standard library only, no cgo, no dependencies — anything importable
  without the engine.
- The JSON byte contract is fixed: two-space indent, one trailing newline, no HTML escaping, key
  order from struct field declaration order, no hand-written marshallers.
- There is one implementation of the glyph grammar and it is package `glyph`. No surface may
  re-implement parsing, printing, or classification.

## Testing

**TDD candidates — write these first, they are pure and fast.**

1. `glyph/parse_test.go` — the self-form accept table. Every self example named in the rewritten
   `docs/glyph.md` §1/§3 is a row: `internal/reedengine/render#`,
   `internal/reedengine/render/focus.go#`, `cmd/lyx#`, and the `_test` unit form. Each row asserts the
   returned `Glyph` has the expected `Unit`, a nil `Owner`, an empty `Name`, a nil `Params`, and that
   `IsSelf()` is true.
2. `glyph/parse_test.go` — the reject table. `internal/logger` → `no_separator` with `Detail ==
   "internal/logger#"`; `a#b#c` and `a#b#` → `multiple_separators`; `#` → `unit_empty`; `.#` and
   `a/../b#` → `unit_dot_segment`; `a//b#` → `unit_empty_segment`. Assert the `Reason` value, not the
   message text, except for one message-shape test on `no_separator` that pins D4's authoritative
   sentence verbatim — including the "to its repository-relative form" clause, which is the whole
   point of the test. A cwd-relative argument row (`logger.go` → `Detail == "logger.go#"`) is part of
   this table, so the test shows the clause doing its work on the exact input it exists for.
3. `glyph/string_test.go` — `Glyph{Lang: Go, Unit: "a/b"}.String() == "a/b#"`, and the round trip
   `Parse(Go, s).String() == s` for every self-form row in test 1.
4. `glyph/errors_test.go` (or wherever `Reasons` is currently exercised) — `Reasons` still enumerates
   exactly the constants in the block, `member_empty` is absent, `multiple_separators` is present, and
   every `Reason` has a distinct `reasonText` entry.

**Per module.**

- `internal/engine/resolve_test.go` — a self glyph naming a file returns `found` with a `Listing`
  holding exactly one `FileEntry` and no symbols on it; a self glyph naming a directory returns
  `found` with the directory's `Listing`; a self glyph naming neither returns `not_found` with a nil
  `Listing` and an empty `Unit`; `ID` is set to the trailing-`#` form on all three; `Target` is the
  argument verbatim. A bare path returns an empty `Status` with `Reason == "no_separator"`. A
  two-`#` target returns an empty `Status` with `Reason == "multiple_separators"`. A multi-target
  `Resolve` call mixing a symbol glyph, a self glyph and a bare path returns three positionally
  correct results and a nil error — the rejection taints only itself. Use the existing
  `internal/engine/testdata/` fixture trees; add a fixture only if no existing tree has both a
  directory and a file worth self-addressing.
- `internal/engine/expand_test.go` — `expand` of a self glyph returns the zero `ExpandAnswer` and an
  error `errors.As`-reachable as `*SelfGlyphError` with `ID` set; assert that no directory read
  happened by passing a memo and reading its `parses` counter as zero (the existing seam).
- `internal/engine/roundtrip_test.go` and `internal/engine/glyph_test.go` — must pass unchanged. Run
  them as the regression gate on D9.
- `internal/repopath/target_test.go` — `RepoRelTarget` rejects a target whose cleaned relative form
  has a `#` in any segment, at the front, in the middle and in the basename; an escaping target still
  reports `ErrTargetOutsideRepo` rather than the separator error; a clean target is unaffected.
  **Delete only the tests of the exported wrapper.** `TestRepoRelPath_LeadingDotDotNotRejected`
  (line 120) and `TestRepoRelPath_AgreesWithRepoRelTarget` (line 146) call the *unexported*
  `repoRelPath`, which D6 retains — deleting them would drop coverage of behaviour this task
  preserves. Keep both, renaming them off the deleted exported symbol's name, and **amend the
  agreement test**: its current claim is that the two agree on every input that does not escape the
  root, which narrows once `repoRelTarget` gains the separator reject — the amended claim is that
  they agree on every input that neither escapes the root nor contains a `#`, with the separator
  divergence asserted as its own row rather than left implicit.
- `internal/cli/cli_test.go` — exit codes and stream routing, the table the existing
  "Exit-code mapping" test already establishes: `resolve <bare-path>` → exit 1, payload on stdout,
  nothing on stderr; `resolve a#b#c` → exit 1, payload on stdout; `resolve <file>#` on an existing
  file → exit 0; `resolve <file>#` on a missing one → exit 1; `expand <unit>#` → exit 1 with the
  failure envelope on stdout and the message on stderr; `toc 'a#b'` → exit 2 with the usage message.
  Both JSON and `--text` for each of the resolve rows — the done-when requires both views.
  **Plus D19's row, which is a changed exit code and therefore needs pinning both ways:**
  `expand <bare-path>` → exit **1** (not the old usage exit 2), carrying D4's actionable
  repository-relative message, and the same argument to `resolve` → exit 1 as well, asserted in one
  test so the two verbs' agreement is what the test is *about* rather than a coincidence of two
  separate rows.
- `internal/cli/flags_test.go` — any existing case pinning the deleted `expand` usage gate is removed
  or retargeted; grep for the "expand takes a glyph" string before assuming there is none.
- `glyph/` — `Self(Go, "internal/logger")` returns a `Glyph` equal to `Parse(Go, "internal/logger#")`
  and `String()`s back to `internal/logger#`; `Self` rejects a path containing a `#` and a path with
  a `..` segment with the same `*ParseError` reasons `Parse` gives; `Self` with a non-Go language
  returns `unsupported_language`. The round trip D16(b) promises is asserted end to end: take a
  `FileEntry` from a `toc` answer, build its self glyph with `Self`, and resolve it back to that same
  file's listing — no string concatenation anywhere in the test.
- `internal/cli/after_test.go` + `docs/research/output-formats/after/` — the four positive goldens per
  D17, regenerated with `-run TestAfter -update` against the pinned Loomyard checkout, plus
  `after/INDEX.md`.
- `quarry/text_test.go` — `RenderResolveText` for a self-glyph `found` emits `<path># found` then the
  directory block; for a self-glyph `not_found` emits `<path># not_found` with no unit suffix and no
  trailing whitespace; for a rejection emits the branch-1 error line; the `default` arm still emits
  line 1 alone for a hand-built value with no `ID` and no `Listing`. The renderer's invariants — no
  trailing whitespace on any line, exactly one closing `"\n"` — are asserted for every new case.
- `internal/mcpserver/toc_errors_test.go` — a `#`-bearing target reaches the server's error surface
  with the separator message rather than a listing. **Assert the exact sentence, not merely that an
  error came back**: the failure this guards against is the sentinel falling into the
  `"internal error: "` else-arm, which still produces an error result and would pass a
  shape-only assertion. Assert too that the sentence is byte-identical to the one `runTOC` emits, so
  the two surfaces cannot drift apart later.
- Docs-as-tests: a single table in `glyph/` enumerating every example and every reject that appears in
  the rewritten `docs/glyph.md` §1/§2/§3/§5, so the done-when clause "every example is a test" is
  satisfied by one reviewable list rather than scattered assertions. The §2 per-language self-form
  rows are documentation-only for Python and C# — there is no extractor to test against — so the
  table asserts the Go rows and carries the two non-Go rows as commented-out placeholders naming the
  task that will enable them, rather than silently omitting them.

**Scenarios that must be covered, stated once so none is lost:** method glyph (unchanged), type glyph
(unchanged), file self-glyph, directory self-glyph, self-glyph not_found, bare-path rejection,
separator-bearing-segment rejection at `resolve`, separator-bearing path target at `toc`, `expand` of
a self glyph, the `listing` key rename in JSON, the positional multi-target guarantee with a mixed
target list, `expand <bare-path>`'s move to exit 1 alongside `resolve`'s, the MCP separator sentence
matching the CLI's byte for byte, `glyph.Self`'s listing-to-glyph round trip, and the T3 round trip.

## Risks

- **The `toc` separator reject (D8) is a behaviour change** the task's constraints appear to forbid.
  Recorded in D8 with the reasoning and with the note that it is the one removable piece.
- **The repository root cannot be addressed by `resolve` after this change.** `.#` fails
  `unit_dot_segment` and `#` fails `unit_empty`; `quarry resolve .` used to answer. This is the open
  gap `docs/glyph.md` §2 and `unitSpellable`'s doc comment already record — what the root's unit
  should spell is a question for the contract, not for this task. `toc .` remains the way to ask.
  State it in `docs/glyph.md` §5 rather than leaving a reader to discover it.
- **Cwd-relative resolve targets stop working** (D6). `quarry resolve logger.go` from inside
  `internal/logger` used to answer; the new spelling is `internal/logger/logger.go#` from anywhere.
  Deliberate, and the `no_separator` message names the fix — but `Detail` names `logger.go#`, the
  cwd-local spelling, which is *not* the working glyph. **Resolved in D4:** the message says the
  appended form is repository-relative, and D4's quoted sentence carries that clause and is the
  authoritative wording the message-shape test pins (Testing item 2). Recorded here as the reason
  that clause is not optional, not as an open question.
- **`ReasonMemberEmpty` is an exported constant leaving the API.** Free today (zero external
  consumers) and recorded in the squash message.

## Q&A log

- **Q:** How is the self form represented in `glyph.Glyph` — an empty member, a `Self bool`, or a sentinel `Name`? **A:** [auto-pick] Empty member (`Owner` nil, `Name` ""), plus an `IsSelf()` predicate. **Why:** `String()` already prints `Unit + "#"` for that value with no edit, which makes "removing the trailing `#` yields the plain path" true by construction in both directions; a bool or a sentinel adds a state that can disagree with `Name`.
- **Q:** Where does `Parse` admit the empty member, and what happens to `ReasonMemberEmpty`? **A:** [auto-pick] In `parseGo`, after `checkGoUnit` and before `checkGoMember`; delete `ReasonMemberEmpty`. **Why:** the unit half must still be validated for a self glyph, and a constant nothing can produce would falsify the vocabulary's own "closed and exhaustively enumerated" documentation — an empty member is the absence of a member, not a member-syntax question, so no alphabet can ever produce the reason. (Narrower than "the self form is language-free", which is false — see D15.)
- **Q:** How is a separator-bearing segment rejected — a new reason, or the existing `member_bad_rune`? **A:** [auto-pick] A new `multiple_separators` reason from a `Parse`-level count check requiring exactly one `#`. **Why:** item 3 demands an explicit error rather than an incidental one, and a count check states the rule once for both halves; `member_bad_rune` cannot fire for the unit half at all.
- **Q:** Which reason and which layer produces the bare-path rejection? **A:** [auto-pick] The existing `no_separator`, produced by `glyph.Parse`, with `Detail` carrying the appended form. **Why:** one implementation of the grammar; `expand`'s engine half already routes a bare path through this reason, so once D19 deletes the CLI's `expand` gate the two verbs agree by construction rather than by two matching checks.
- **Q:** Does the bare-path rejection use the payload-with-exit-1 shape or the error envelope? **A:** [auto-pick] The payload with exit 1, per the existing D2 decision. **Why:** the task says to follow that decision, not reinvent it, and `codeForResolveResult` plus `RenderResolveText` branch 1 already implement it — zero new code.
- **Q:** Does `resolve` keep any cwd-relative path arithmetic for the self form? **A:** [auto-pick] No — the argument goes to the facade verbatim and `repopath.RepoRelPath` is deleted. **Why:** a glyph's unit is repository-relative by the grammar's own definition, and cwd arithmetic on it would let one glyph string name two units depending on where the shell sat.
- **Q:** Does `toc` also reject a `#`-bearing path target, given the "no behavior change to toc" constraint? **A:** [auto-pick] Yes, at `internal/repopath` with a new `ErrTargetHasSeparator` sentinel, exit 2 on the CLI. **Why:** item 3 says "everywhere", and leaving the one path-taking verb to silently accept such a target reintroduces exactly the silent acceptance it bans; the constraint is read as protecting traversal, answer shape and defaults, and the tension is recorded in D8 as the single removable piece.
- **Q:** Does the walk change — should a `#`-bearing directory be pruned? **A:** [auto-pick] No change at all. **Why:** `unitSpellable` probes `unit + "#x"`, which already fails today and still fails under the new count rule; the reason changes, the emitted answer does not, which keeps the T3 round trip true by construction.
- **Q:** Does a self-glyph result carry `id` and `unit`? **A:** [auto-pick] `id` yes (the trailing-`#` form), `unit` never. **Why:** `id` is parity with every other glyph answer and with `Symbol.ID`; for a self glyph the unit *is* the thing asked about, so `unit: not_found` beside `status: not_found` would restate the answer.
- **Q:** How does `expand` reject a self glyph — a sixth `Kind` value, or a new typed error? **A:** [auto-pick] A new `SelfGlyphError`, gated immediately after parse, aliased through the facade. **Why:** `Kind` is documented as the five values `toc` emits for a `Symbol`, and a self glyph is not a symbol; widening a symbol vocabulary to describe a non-symbol would falsify that doc comment.
- **Q:** Should `SpansOf` reject a self glyph? **A:** [auto-pick] No — it returns the existing empty slice with a nil error, with one doc sentence added. **Why:** `SpansOf` has no status vocabulary, "nothing matches" is exactly what an empty slice means there, and it is the function the T3 round trip is written against — it must not be edited for behaviour.
- **Q:** Does the `dir` → `listing` rename touch the inner path field or the MCP server? **A:** [auto-pick] Outer field only; the MCP server has no `resolve` tool and needs no edit. **Why:** the inner `dir` is shared with `toc`'s answer and is out of scope; the task's "MCP text view" clause has no referent, and the rename changes zero bytes of text output because the outer field name is never printed.
- **Q:** How does the text renderer handle a self-glyph result, given the glyph branch fires on `ID != ""`? **A:** [auto-pick] Insert a `Listing != nil` branch ahead of it, printing `r.ID` then the directory block. **Why:** the pointer being non-nil is the fact the renderer needs; keying on status or on the target string would be a heuristic, and using `r.ID` for line 1 shows the canonical trailing `#` the JSON `id` carries.
- **Q:** Where do the six required goldens live? **A:** [auto-pick] The four positive cases in `docs/research/output-formats/after/` (Loomyard-gated, `-update`-regenerable); the two rejection cases as plain `internal/cli` table tests. **Why:** `glyph5_test.go` already records that the `after/` evidence goldens cannot carry every case, and a test that reads no source should not acquire a pinned-checkout dependency.
- **Q:** Does `docs/glyph.md` get a new numbered section for the self form? **A:** [auto-pick] No — it is stated in §1 and §3 and its consequences in §5/§6/§7. **Why:** it is a property of the one form, not a sixth topic; a separate section would let §1's form line stay silent about it.
- **Q:** `internal/cli/flags.go:153` holds a third `#`-containment classifier — `expand`'s usage gate. Keep it or delete it? **A:** [auto-pick] Delete it (D19); `expand <bare-path>` then exits 1 through `glyph.Parse` like `resolve` does. **Why:** it is the classifier that makes the two verbs disagree with each other — exit 2 with a usage message versus exit 1 with a payload for the same malformed argument — and D7's mandated doc.go sentence "classification happens exactly once" is unwritable while it stands.
- **Q:** Does the MCP server really need no edit for the new separator sentinel? **A:** [auto-pick] No — `tocResult`'s `RepoRelTarget` error block gets an explicit branch with the CLI's exact wording. **Why:** that block is one `errors.Is(ErrTargetOutsideRepo)` test plus an `"internal error: "` else-arm, so without a branch a malformed user target is reported to an MCP client as a server fault carrying package-namespaced sentinel text — and a shape-only test would pass on it.
- **Q:** Is the self form language-free, as D1 and the Scope-Out bullet claimed? **A:** [auto-pick] No — only the empty-member *representation* is shared; the meaning of `unit#` is per-language, and `docs/glyph.md` §2 joins the edit inventory. **Why:** §2 defines the unit as a Python dotted module and a C# namespace, so "removing the trailing `#` yields the plain repository-relative path" is a Go-only sentence, and a *file* self glyph exists only for Go — Python's file is its module, C# has no file-level unit. §2 as written also does not admit a file as a left half at all, which the new paragraph must fix.
- **Q:** Which sentinels live in `internal/engine/errors.go`? **A:** [auto-pick] Only `ErrLanguageUnsupported`; the two target sentinels are in `repo.go:46,50`, and `ErrTargetHasSeparator` joins them there. **Why:** `errors.go`'s header reads "the one error sentinel the engine's subpackages share" — adding a second would falsify it and force an unrelated header rewrite.
- **Q:** `docs/rewrite-plan.md` §4 says "What `toc` lists is what `resolve` takes". Does that survive? **A:** [auto-pick] No — it is rewritten, and `glyph.Self` (D20) is added so a consumer can complete the round trip. **Why:** `toc`'s file and directory entries carry no `id`, so once `resolve` is glyph-only the sentence is only true if consumers hand-concatenate `dir + "/" + name + "#"` — the printing the one-implementation constraint forbids outside package `glyph`, and Loomyard is exactly the consumer that would hit it first.
- **Q:** Delete the `RepoRelPath` tests along with the exported function? **A:** [auto-pick] Only the ones testing the exported wrapper; the two calling the retained unexported `repoRelPath` stay, with the agreement test amended. **Why:** deleting them would drop coverage of behaviour D6 preserves, and the agreement claim narrows once `repoRelTarget` gains the separator reject — the divergence deserves its own asserted row.
