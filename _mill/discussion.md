# Discussion: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
task: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)
slug: diff-to-symbols
status: discussing
parent: main
```

## Problem

Quarry answers three questions today — `toc` (what is here), `resolve` (where is this, right now),
`expand` (what does this type consist of). Loomyard's orchestration design (issue #226) needs a
fourth, and it is the one quarry does not have: **what changed, in symbols?** Loomyard binds
placeholder handles at card-done, runs a scope guard comparing a card's diff against its declared
targets, detects drift (deleted symbols still referenced by the remaining plan), and repairs
renames — all mechanical, no LLM in the loop. Every one of those needs a symbol-level delta
between two versions of the code, keyed by glyph.

**Why now:** this is the third of the three plan-alphabet primitives in `docs/roadmap.md` point 2.
Its one ordering constraint — C1's contract merge, which changed the envelope and the glyph
self-form these primitives emit and consume — was satisfied on 2026-09-05 (`49304ca`), so it is
unblocked. It runs in parallel with the glyph-maker task (2b) and with Loomyard's own adoption
work; the contact points with 2b are only the CLI dispatch table, one facade method, and a
`docs/rewrite-plan.md` §5 paragraph, which are trivial merge conflicts resolved at merge time.

The load-bearing design constraint is that this query **never parses a textual diff**. A textual
diff is a view of bytes; the question is about symbols. The mechanism is instead **double
extraction plus symbol-table comparison**: extract both versions of each changed file with the
same extractor `toc` uses, then compare the resulting symbol tables. Git's only job is to avoid
extracting files that did not change; correctness lives entirely in the table comparison.

## Scope

**In:**

- A new pure core in `internal/engine`: given a batch of `(path, before-bytes, after-bytes, units)`
  entries, produce a symbol-table delta — `created`, `deleted`, `modified`, `renamed` (exact tier),
  and `rename_candidates` (evidence tier), plus a `files` block echoing every entry's disposition.
- Three new **JSON-hidden byte-offset fields on `engine.Symbol`** — `DeclStart`, `BodyStart`,
  `DeclEnd` — filled by every builder in `internal/engine/golang.go` from the same node it already
  hands `SignatureCut`. This is the seam the token streams need; without it nothing in the
  extractor's output can reach a symbol's declaration node. The emitted JSON key set is unchanged
  (all three are `json:"-"`, like `HeadStart`/`HeadEnd`).
- Three new exported entry points in `internal/engine`, all extracted from the existing
  `dirPackage`/`unitFor` implementations rather than copied: `PackageClause` (bytes → package
  clause), `(*Repo).ClauseMapForFiles` (on-disk clause map for a supplied file list), and
  `UnitsForClauseMap` (the clause vote plus the unit derivation). `dirPackage` is refactored to call
  the first and third so there is exactly one implementation of each rule.
- A new package `internal/gitsrc`: a thin, read-only git plumbing layer (`git rev-parse --verify`,
  `git diff --name-status --no-renames`, `git ls-files --others --exclude-standard`, `git show`,
  `git ls-tree`) returning paths, bytes and errors only, with the working tree as the after side
  when `--to` is absent and untracked files included on that path.
- Two facade methods on `quarry.Repo`: `Delta(entries)` (pure) and `DeltaGit(from, to, target)`
  (convenience, delegating to `internal/gitsrc` then `Delta`).
- A new CLI verb `delta`, with `--from`, `--to`, the existing `--text` and `--root`, exactly one
  target, resolved through `repopath.RepoRelTarget` exactly as `toc`'s is (so `.` means the current
  directory, not the repository root), the standard JSON envelope and the standard exit-code
  contract.
- New JSON and text renderers in `quarry/` (`RenderDeltaJSON`, `RenderDeltaText`), sharing the
  existing `renderJSON` encoder configuration.
- Goldens (JSON and text) for created, deleted, modified, exact-tier rename, evidence-tier rename,
  a mixed batch, and a per-entry extraction failure inside an otherwise good batch.
- A T3-style real-history test pinned to `d413ceb..49304ca` in this repository.
- An honest amendment to the "facade adds no behaviour of its own" claim in **both**
  `quarry/doc.go` and `quarry/repo.go`'s `Open` doc comment, naming `DeltaGit` as the exception.
- One new paragraph in `docs/rewrite-plan.md` §5, and one line in §7's mechanical-use list.

**Out:**

- **No MCP tool.** The caller is Loomyard's pipeline through the facade, never an LLM. Nothing in
  `internal/mcpserver` or `cmd/quarry-mcp` changes.
- **No change to any existing verb, envelope, or exit code.** This is purely additive. `toc`,
  `resolve` and `expand` keep their current behaviour and their current goldens byte-for-byte.
- **No plan, card, or handle vocabulary.** Quarry knows nothing about Loomyard's concepts; those
  live in issue #226.
- **Quarry never picks a rename candidate.** The exact tier is asserted; the evidence tier is
  reported with signals and no verdict. There is no threshold, no score cut-off, and no ranking
  that means anything beyond a deterministic ordering.
- **No git rename detection.** Git's own `-M` heuristic is a similarity threshold, which is exactly
  what this contract forbids; `--no-renames` is passed so a rename reaches quarry as a delete plus
  an add and is classified by the table comparison alone.
- **No new language.** Go only, as everywhere else in quarry.
- **No writes of any kind.** Every git call is read plumbing; the working tree and the index are
  never touched.
- **No cache, no index, no daemon** (`docs/rewrite-plan.md` §10 phase-1 non-goals stand).
- **`docs/roadmap.md` is not edited by this task** — the roadmap says what is ahead; removing a
  completed item is the merge's job, not this branch's.

## Decisions

### The unit is supplied to the core, never derived inside it

- Decision: each core entry carries an explicit `BeforeUnit` and `AfterUnit` string. The core never
  derives a unit and never touches the filesystem or git. A separate exported engine helper —
  `engine.UnitsForClauseMap(dirRel string, clauses map[string]string) (dirPkg string, unitOf
  func(base string) string)` or equivalent — derives them, and is called once per changed directory
  per side by the layer above (the git layer, or a caller of the pure facade method).
- Rationale: the glyph unit is a *directory-level* fact. `internal/engine/walk.go`'s `unitFor`
  derives it from three inputs — the directory's repository-relative path, the directory's dominant
  package clause, and the file's own clause — and the dominant clause is a vote over *every* Go file
  in the directory, not only the changed ones. A byte pair for one file simply does not carry that
  information. Making the unit an explicit input keeps the core a pure function of its arguments
  (trivially table-testable, no fixtures), puts the cost of establishing it where a reader can see
  it, and keeps the derivation rule in exactly one place — the existing vote rule in `dirPackage`
  and `unitFor` — rather than reimplementing it.
- Rejected: (a) *voting over the changed entries alone* — wrong precisely when the directory's real
  package clause is not represented among the changed files. Concretely: a directory whose package
  is `foo`, where the only changed file is an external test file with clause `foo_test`, votes
  `dirPkg = "foo_test"`, and `unitFor` then returns the directory itself instead of the
  `_test`-suffixed unit. Silent, and wrong in exactly the case a plan cares about.
  (b) *approximating from the path plus the file's own clause* (unit is `dir+"_test"` when the
  clause ends in `_test`, else `dir`) — this is the shortcut `walk.go`'s own doc comment explicitly
  warns against: it splits a package legitimately named `httptest` or `mytest` into a unit it does
  not have. `walk.go` keys on `clause == dirPkg+"_test"` for that reason, and this task does not get
  to weaken that rule.

### The verb is `delta`

- Decision: the CLI verb, the facade method, and the engine method are all named `delta`
  (`quarry delta`, `(*quarry.Repo).Delta`, `(*engine.Repo).Delta`).
- Rationale: quarry's verbs name the question asked, not the mechanism — `toc` is "what is here",
  `resolve` is "where is this", `expand` is "what does this consist of". The question here is "what
  is the difference, in symbols", and `delta` is the noun for it. It also does the one job the name
  must do: it does not suggest a textual diff. `docs/rewrite-plan.md` §9 lists conflating symbols
  with positions and views among the non-goals, and this whole design turns on *not* parsing a
  diff; naming the verb `diff` would invite exactly the misreading the mechanism exists to avoid,
  in the one place — the command line — where a reader has no doc comment in front of them.
- Rejected: `diff` (invites the textual-diff reading; the flags `--from`/`--to` would reinforce it);
  `changes` (vaguer, and reads as a listing of commits).

### Two rename tiers, two separate keys

- Decision: the answer carries `renamed` and `rename_candidates` as two distinct arrays.
  - `renamed` holds **exact-tier** pairs only. Quarry asserts these. Each exact pair's deleted
    symbol is **removed** from `deleted` and its created symbol is **removed** from `created` — the
    rename is the assertion, so the constituent create and delete are no longer true statements
    about the batch. Because of that removal, the pair is the **only** place either symbol survives,
    so it carries both in full: `from` is the before-side `Symbol` and `to` is the after-side
    `Symbol`, each with its own `id`, `kind`, `file` and spans. An id-only pair would discard the
    created symbol's location, which is exactly what Loomyard's handle rebinding needs.
    (`created` and `deleted` are themselves `[]Symbol`.)
  - `rename_candidates` holds **evidence-tier** reports. Quarry asserts nothing. The deleted symbol
    **stays** in `deleted` and every candidate created symbol **stays** in `created`; the candidate
    block only cross-references them by id and attaches signals. Each entry is one deleted id plus
    an array of candidates, each candidate carrying a created id and its signals.
- Rationale: this is the `ambiguous` philosophy of `docs/rewrite-plan.md` §3 spelled for renames.
  An asserted outcome changes the answer; a reported outcome adds information and removes none. A
  consumer that ignores `rename_candidates` entirely still gets a correct, conservative delta —
  the create and the delete are both there — which is the behaviour a mechanical gate wants when it
  does not want to reason about renames. A consumer that reads the candidates (Loomyard's
  drift-review path) sees every possibility with the evidence behind it and picks nothing on
  quarry's behalf. Suppressing the create and delete for an evidence-tier candidate would be a
  silent pick in disguise.
- Rejected: one flat list with a `change` discriminator field (loses the tier separation the
  contract is built on, and forces every consumer to filter); per-file grouping (renames are
  unit-wide, so a per-file grouping would have to duplicate or split every cross-file pair).

### Every array in the answer has a stated, deterministic order

- Decision: the answer's ordering is total, and every rule is documented as **ordering, never
  ranking**:
  - `created`, `deleted` and the `after` array of a `modified` entry: **file ascending, then `Start`
    ascending, then `id` ascending, then `kind` ascending** — `symbolsOfUnit`'s existing
    file-then-`Start` rule with a total tie-break appended. The tie-break is not decoration:
    `goUngroupedConstOrVarSymbols` and `goGroupedConstOrVarSymbols` give **every name in one spec the
    same `Start` and `End`**, so `const a, b = 1, 2` yields symbols that file-then-`Start` alone
    cannot separate, and `sort.SliceStable` would then preserve whatever order the `(ID, Kind)` map
    range happened to produce — randomised per run. `(file, Start, id, kind)` is unique because the
    table key `(id, kind)` is.
  - the `changed` array inside a `modified` entry: **the closed set's own declaration order** —
    `body`, `signature`, `doc`, `file` — never the order the dimensions were discovered in.
  - `modified`: **`id` ascending, then `kind` ascending**, the table key itself, so the order does
    not depend on which occurrence happened to be seen first.
  - `renamed`: **`from.id` ascending, then `to.id` ascending.**
  - `rename_candidates` entries: **deleted `id` ascending**; candidates *within* an entry keep the
    similarity-then-`id` order already stated in the evidence-tier decision.
  - `files`: **the input batch's own order**, unchanged, so a caller can index it against what it
    submitted.
  - a `modified` entry's `before` array: **positionally aligned with nothing** — it is independently
    sorted file-then-`start`, because a multi-occurrence key has no occurrence identity to align by.
- Rationale: the symbol table is keyed by `(ID, Kind)` and Go's map iteration order is deliberately
  randomised, so an answer assembled by ranging over it is non-deterministic. Committed JSON and
  text goldens — which this discussion requires for seven cases (see Testing) — cannot be byte-stable
  without this, and a pipeline diffing two delta outputs would see phantom changes. Reusing
  `symbolsOfUnit`'s existing file-then-line rule rather than inventing one keeps a symbol list
  ordered the same way wherever a caller meets it.
- Rejected: leaving order unspecified (goldens impossible, and the plan would have to invent a rule
  anyway); ordering `created`/`deleted` by `id` (breaks the file-then-line convention every other
  symbol list in quarry follows).

### The repository root must be git's top-level

- Decision: `DeltaGit` verifies `git -C <root> rev-parse --show-toplevel` and requires it to equal
  `<root>`. **Both sides of that comparison are put through `filepath.EvalSymlinks` first**, then
  `filepath.Clean`. `repopath.ResolveRoot` only does `filepath.Join` plus `filepath.Clean` and never
  resolves symlinks, while `git rev-parse --show-toplevel` prints the *physical* path — so a `--root`
  reached through a symlink would fail a raw string comparison on a perfectly valid repository.
  `t.TempDir()` is exactly such a path on any platform whose temp directory is symlinked, so
  without this the check would reject the project's own test fixtures. Two failures, both with quarry's own sentence and neither carrying git's raw message:
  - not a git repository at all → **exit 2**, `not a git repository: <root>`;
  - a git repository whose top-level is elsewhere → **exit 2**,
    `--root is not the repository top-level: <root> (top-level is <toplevel>)`.
  `internal/gitsrc` returns two sentinels (`ErrNotARepository`, `ErrRootNotTopLevel`) that the CLI
  matches with `errors.Is`, alongside `ErrUnknownRevision`.
- Rationale: `repopath.ResolveRoot` accepts **any existing directory** for `--root` and skips `.git`
  discovery entirely — it stats the path and returns it. So `git -C <root>` can legitimately run
  inside a repository whose top-level is *above* `<root>`, and `git diff --name-status` and
  `git ls-tree` then emit paths relative to the **git top-level** while quarry consumes them as
  `<root>`-relative. Every path in the answer would be silently wrong, and the pathspec would select
  the wrong subtree. A `--root` outside any repository is worse still: every git call fails and the
  user gets exit 3 with git's raw message for what is plainly a usage mistake. Requiring equality
  and refusing loudly is the honest option; only this verb needs it, because only this verb consults
  git, so `toc`, `resolve` and `expand` keep accepting any directory exactly as they do today.
- Rejected: translating between the two prefixes (doable, but it makes every path in the answer
  depend on a relationship the caller cannot see, and silently changes what a pathspec means);
  letting git's own failure surface as exit 3 (reports a usage mistake as an internal error, which
  `Run`'s doc comment forbids).

### What `modified` means, and the `changed` field

- Decision: a symbol present on both sides under the same key is `modified` when **any** of the
  following differ, and the entry carries a `changed` array naming which of them did, drawn from
  the closed set `["body", "signature", "doc", "file"]`:
  - `body` — the body token stream differs (see the next decision for what a token stream is).
  - `signature` — the `Signature` text differs. It is built over the same byte span the signature
    token stream is (see the next decision, including the synthesized keyword prefix for grouped
    declarations), so the text comparison here and the token comparison used for renames cannot
    disagree about which bytes are "the signature".
  - `doc` — the `Doc` text differs.
  - `file` — the symbol's repository-relative file differs between the sides.

  A symbol whose only difference is its line numbers (`Start`/`SigEnd`/`End` shifted because
  something above it grew or shrank) is **not** modified and appears nowhere in the delta. A
  `modified` entry carries `after` (the after-side `Symbol`s in full) and `before` (the before-side
  `file`/`start`/`sigend`/`end` blocks) as **arrays** — length one in the ordinary case; see "The
  symbol table key, and `func init`" for why the shape is uniformly an array.
- Rationale: line-shift-insensitivity is the entire point of glyph identity — "a plan needs a
  stable name; an implementer needs where that is right now, which moves with every edit"
  (`docs/rewrite-plan.md` §2 item 4). Beyond that, all four kinds of change matter to a real
  consumer and none of them subsumes the others: Loomyard's scope guard reads signature changes,
  its documentation-drift check and quarry's own repository-invariants use read doc changes, its
  handle binding reads the file move, and the card done-check reads the body. Collapsing them into
  one bit would force every consumer to re-derive the distinction from the spans, which is the
  parallel-implementation failure this codebase's one-implementation rule exists to prevent. Naming
  which dimension changed costs one array and answers the "does a doc-comment-only change count as
  modified?" question honestly: yes, and the answer says so.
- Rejected: body tokens only (a doc-only or signature-only change would vanish from the delta,
  which is wrong for three of the four named consumers); any byte difference in the declaration
  span (would report every symbol below an edit as modified, since spans move).

### The body token stream, and the exact-tier identity test

- Decision, and the seam it needs: **`Symbol` gains three JSON-hidden byte offsets**, and the two
  streams are defined by **byte range** over them, split at exactly the byte
  `internal/engine/nodes.go`'s `SignatureCut` already cuts at — never by node exclusion.

  `Strategy.Symbols(unit, root, src) []Symbol` returns symbols only, and `Symbol` carries lines and
  text but no byte offsets, so nothing in today's extractor output can reach a symbol's declaration
  node. Rather than add a `Strategy` method or walk the tree a second time, `Symbol` gains
  `DeclStart`, `BodyStart` and `DeclEnd` — byte offsets, all `json:"-"`, following the precedent
  `HeadStart`/`HeadEnd` already set for JSON-hidden fields that exist for one consumer. Every builder
  in `internal/engine/golang.go` fills them from **the same node it already hands `SignatureCut`**:
  - `goDeclSymbol` (function, method): `decl` is the declaration node, `body` its `"body"` field.
  - `goUngroupedTypeSymbol`: `decl` is the `type_declaration`, `body` is `goTypeBody(spec)`.
  - `goGroupedTypeSymbol`: `decl` is **the spec**, `body` is `goTypeBody(spec)`.
  - `goUngroupedConstOrVarSymbols` / `goGroupedConstOrVarSymbols`: `decl` is the declaration and the
    spec respectively, and `body` is **nil** for both.
  - `goInterfaceMethodSymbols`: `decl` is the `method_elem`, `body` is nil.
  `BodyStart` is `body.StartByte()` when `body` is non-nil and `DeclEnd` when it is nil, so the
  nil-body case needs no separate branch anywhere downstream. Then:
  - the **signature token stream** is every leaf node whose start byte lies in
    `[DeclStart, BodyStart)`;
  - the **body token stream** is every leaf node whose start byte lies in
    `[BodyStart, DeclEnd)`.

  The delta core already parses each side's bytes inside one `treesitter.WithTree` callback; in that
  same callback it walks the root's leaves **once**, in source order, and assigns each leaf to
  **every** symbol range that contains it — not to one. Symbol spans nest and overlap, and both are
  intended: an interface's `method_elem` leaves belong to the interface type's *body* stream and to
  that method symbol's own *signature* stream simultaneously, and the several symbols of
  `const a, b = 1, 2` share one span verbatim, so each of them gets the same stream. One walk, many
  assignments — no per-symbol node lookup, no second parse, no second walk.

  **What the invariant against `Symbol.Signature` actually is.** The signature stream spans exactly
  the bytes `SignatureCut` cuts, but `Symbol.Signature` is not always those bytes verbatim: it is
  `strings.TrimSpace` of them, and for the two grouped shapes a keyword is **synthesized** in front —
  `goGroupedTypeSymbol` builds `"type " + SignatureCut(spec, body, src)` and
  `goGroupedConstOrVarSymbols` builds `"const " + …` or `"var " + …`. The invariant is therefore:
  *the signature token stream covers the same byte span `SignatureCut` was given, and
  `Symbol.Signature` equals that span trimmed, with a synthesized `"type "`, `"const "` or `"var "`
  prefix for a grouped declaration and no prefix otherwise.* A test asserting byte-for-byte equality
  in general would be false; the test asserts this invariant per shape instead.

  Both streams are sequences of `(node kind, node text)` pairs in source order, taken from the same
  parse the extractor already performs. **Anonymous leaf nodes are included** — operators, keywords and
  punctuation (`+`, `-`, `++`, `--`, `:=`, `return`, `if`, `{`) are anonymous in the tree-sitter Go
  grammar, being grammar string literals rather than named rules, and a stream restricted to *named*
  leaves would omit every one of them. Whitespace and line numbers contribute nothing. Neither
  stream includes the doc block: `decl` does not span it, and `Doc` is compared separately.

  **Why a byte range and not "the declaration minus its body child".** `goTypeBody` returns the bare
  `"{"` **leaf** as an `interface_type`'s body-bearing child — its own doc comment in
  `internal/engine/golang.go` records that the grammar exposes "no named body node and no 'body'
  field at all" for an interface. Under a node-based rule an interface's body stream would be that
  one leaf and nothing else: method-set and embedded-interface changes would be invisible to `body`,
  and two unrelated interfaces (`type Reader interface{…}` and `type Closer interface{…}`) would
  satisfy the identity test below and be **asserted** as a rename — again in the one tier quarry
  asserts. The mirror-image rule, "the declaration node with its body child excluded", would leave
  every interface method element in the *signature* stream, where `Symbol.Signature` — cut at
  `body.StartByte()` — does not have them, so `changed:["signature"]` (verbatim text) and
  `signature_identical_modulo_name` (token stream) would be computed over different spans of the
  same declaration and could disagree. The byte split has neither problem: an interface's method
  elements begin after the `"{"` and land in the body stream, a struct's fields land there too
  (`field_declaration_list` starts at its own `"{"`), and the signature stream is byte-for-byte the
  span `SignatureCut` returns, so the text comparison and the token comparison are over the same
  bytes by construction.

  A declaration with no body-bearing child — `type ID string`, `type Alias = T` — has
  `bodyStart == decl.EndByte()`, so its signature stream is the whole declaration and its body
  stream is empty, matching `SignatureCut`'s own nil-body branch and `SigEnd`'s zero.

  Two token streams are **identical modulo the renamed identifier** when they have the same length
  and, at every position, either the pairs are equal, or both nodes are `identifier` nodes whose
  texts are respectively the deleted symbol's `Name` and the created symbol's `Name`. This is an
  exact structural test with no threshold and no tuning knob.
- Rationale: comparing token streams rather than bytes is what makes the comparison insensitive to
  reformatting and to line movement while staying exact. Including anonymous leaves is a
  correctness requirement, not a detail: with named leaves only, `x + y` and `x - y` produce
  byte-identical streams, so a real semantic change would be reported as *unchanged* (breaking the
  card done-check the Problem section names as a consumer), and two symbols differing only in such
  tokens would satisfy the identity test and be **asserted** as an exact-tier rename that is false —
  in the one tier the contract says quarry asserts. Restricting the *substitution* rule to
  `identifier` nodes carrying exactly the two names in question is what keeps the exact tier
  *exact*: it admits the recursive self-call a renamed function makes and the receiver-less use of
  its own name, and admits nothing else. Anonymous nodes can never be substituted, since they are
  not `identifier` nodes, so widening the stream does not widen the substitution. The whole tier is
  defined so a test can prove it and a reader can predict it, which is the property "AST-exact, no
  threshold, quarry asserts it" demands.
- Rejected: named leaves only (silently drops every operator and keyword — see the rationale);
  defining the two streams by node exclusion rather than byte range (breaks on interfaces, both
  ways — see the rationale); comparing the raw byte span (reformatting would break it); comparing a
  hash of the normalized source text (same problem, plus it cannot express "modulo the renamed
  identifier"); full tree-edit distance (a threshold in disguise, and quadratic).

### Exact-tier rename scope: unit-wide, and unique or nothing

- Decision: an exact-tier rename pairs a deleted symbol `D` with a created symbol `C` when **all**
  of the following hold:
  1. Same glyph unit (`D.Glyph.Unit == C.Glyph.Unit`), on the before side for `D` and the after
     side for `C`. Cross-unit never pairs.
  2. Same owner chain (`sameOwner`) and same `Kind`.
  3. Different `Name` (identical names are the `modified`/`file`-move case, not a rename).
  4. Body token streams identical modulo the renamed identifier, per the previous decision.
  5. **Signature token streams** identical modulo the renamed identifier, under the same node-based
     substitution rule — never a textual substitution over the verbatim `Signature` string. A
     textual `Run` → `Execute` replacement would also hit the `Runner` in
     `func (r *Runner) Run() error`, producing a false mismatch (or, with a naive replace, a false
     match); the head's own token stream has real `identifier` nodes to key on, so the substitution
     applies to whole identifiers and to nothing else.
  6. **Exactly one** such `C` exists for that `D`, and exactly one such `D` for that `C`.
  7. **Both symbols have a non-empty body stream** (`BodyStart < DeclEnd` on each side), and neither
     side's file had a partial parse (see the `lossy` flag under entry dispositions).
  The two symbols may live in different files. If condition 6 fails — two deleted symbols both pair
  exactly with one created symbol, or vice versa — none of the involved symbols is asserted as
  renamed; they all fall through to the evidence tier instead. If condition 7 fails, the pair falls
  through to the evidence tier as well.

  **Why condition 7 exists — bodyless kinds can never be asserted.** `goUngroupedConstOrVarSymbols`,
  `goGroupedConstOrVarSymbols` and `goInterfaceMethodSymbols` all pass `body == nil`, and a type
  alias has no body either, so for every const, var, alias and interface method `BodyStart` equals
  `DeclEnd` and the body stream is empty on both sides. Condition 4 is then vacuously satisfied by
  *any* two of them, and `const A = 1` deleted alongside `const B = 1` created would be **asserted**
  as a rename — with both vanishing from `created` and `deleted`, since an exact pair removes its
  constituents. That is precisely the false-assertion failure this design calls a defect for two
  unrelated interfaces; a signature-only match is not evidence enough to assert on. Such pairs are
  confined to the evidence tier, where they are reported with `signature_identical_modulo_name: true`
  and a `body_token_similarity` of `1.0` over two empty streams, and quarry decides nothing.

  **Classification is relative to the scoped batch, not to the unit as it exists on disk.** The
  batch is whatever the pathspec selected, so a rename whose other half lies outside the target is
  reported as a plain `deleted` (or `created`) with no candidate at all — quarry cannot pair against
  a symbol it was never given. A caller that cares about renames must scope at least to the unit;
  Loomyard's scope guard, which passes a card's declared targets, gets narrower batches than that by
  design and must read the delta as "within this scope", never as "in this package".
- Rationale: the unit is the identity scope of a glyph, so it is the correct boundary: a
  declaration moving between files inside one Go package is routine and must still be a rename,
  while a declaration moving *between* packages changes its glyph's unit half and is a move, which
  is a different fact and must not be smuggled in under the rename name. Condition 6 is the
  `ambiguous` rule applied one more time: several matches means nothing was chosen, so nothing is
  asserted, and the situation is reported as evidence instead of resolved by an arbitrary
  tie-break.
- Rejected: same-file only (would miss the common "rename and move to its own file" refactor,
  reporting it as an unrelated create plus delete); whole-batch, any unit (would assert a rename
  across a package boundary, silently discarding the unit change that is the more important fact).

### Evidence-tier gating and signals

- Decision: a `(D, C)` pair is an **evidence-tier candidate** when it satisfies exact-tier
  conditions 1–3 (same unit, same owner chain, same `Kind`, different `Name`) and is not part of an
  asserted exact-tier pair. There is **no similarity threshold** — structural facts alone gate the
  candidate set, so the query has no tuning knob anywhere.

  Each candidate carries a `signals` object of purely mechanical, individually explainable fields:
  - `signature_identical_modulo_name` (bool): computed over the signature *token streams* under the
    node-based substitution rule, exactly as exact-tier condition 5 is — never over the verbatim
    `Signature` text.
  - `body_token_similarity` (float in `[0,1]`): the Jaccard coefficient of the two body token
    streams treated as multisets of `(kind, text)` pairs, with the identifier bearing the symbol's
    own name normalised to a single placeholder on both sides. `1.0` when both streams are empty.
  - `body_tokens_before`, `body_tokens_after` (ints): the two stream lengths.
  - `doc_identical` (bool)

  There is **no composite score**. Candidates within one entry are sorted by
  `body_token_similarity` descending, then by created `id` ascending — documented in the type's doc
  comment as a deterministic ordering for reproducible output and **explicitly not a ranking, not a
  recommendation, and not a verdict**.
- Rationale: every signal is a fact a caller can check independently; a single blended score would
  invite "take the top one", which is the silent pick the contract forbids in the strongest terms.
  `docs/rewrite-plan.md` §9 rules out "Fuzzy matching of any kind" as a way of *answering* —
  "Unknown is `not_found`; several is `ambiguous`" — and this tier is the `ambiguous` branch of that
  rule: `body_token_similarity` never reaches a classification, it only describes a candidate quarry
  has explicitly declined to resolve. No asserted outcome anywhere in this query reads it.
  Jaccard over multisets is O(n) and adequate for a signal nobody decides on — a more expensive
  order-sensitive metric would buy precision that quarry, by contract, is not allowed to spend.
  Omitting the threshold means the answer never depends on a magic number, which is what makes the
  evidence tier honest.
- Rejected: a single similarity score (invites the silent pick); raw token streams in the output
  (unbounded payload; the consumer is a pipeline, not a diff viewer); a similarity floor (a
  threshold, and therefore forbidden).
- Accepted cost, stated: a directory with `d` deleted and `c` created symbols of the same kind and
  owner yields up to `d × c` candidate pairs. A large refactor can therefore produce a large
  candidate block. This is accepted rather than capped: a cap is a threshold under another name,
  and silently truncating candidates is the one thing worse than listing many. The pathological
  case is documented in the answer type's doc comment.

### The symbol table key, and `func init`

- Decision: the symbol table is keyed by `(Symbol.ID, Symbol.Kind)`.

  Every occurrence — single or multiple — reduces to one **comparison tuple**:
  `(body token stream hash, signature text, doc text, file)`. That is the same four dimensions the
  `changed` array is drawn from, so the comparison and the reporting cannot cover different ground.

  When a key holds more than one symbol on a side — Go's several `func init()` in one package all
  carry the id `<unit>#init`, per `internal/engine/golang.go`'s own doc comment — the two sides are
  compared as **multisets of those tuples**, never of body hashes alone: equal multisets means
  unchanged; any difference reports **one** `modified` entry for that key. Its `changed` array is
  the union of the dimensions that differ across the multiset difference, so a doc-only change to
  one of two `init` functions is reported as `changed:["doc"]` rather than vanishing.

  **The `modified` entry shape is uniformly arrays**, so a multi-occurrence key needs no second
  shape: the entry carries `after` (every after-side occurrence's `Symbol`, in file-then-line order)
  and `before` (every before-side occurrence's `file`/`start`/`sigend`/`end`, same order). For the
  ordinary single-occurrence case both arrays have length one. When the multiplicities differ, that
  is visible from the array lengths themselves and needs no separate `count_before`/`count_after`
  pair.

  A multi-occurrence key is never a rename candidate on either tier.
- Rationale: including `Kind` in the key means a `const` replaced by a `var` of the same name is a
  delete plus a create — two different declarations — rather than a modification, which is the
  truthful reading. The multiset rule handles `init` without a special case for the word "init" and
  without inventing per-occurrence identities the glyph contract does not define: quarry has no way
  to name the second `init` in a package, so it must not pretend it can pair them up.
- Rejected: keying on `ID` alone (a kind change would be reported as a modification of "the same"
  symbol, which it is not); minting synthetic per-occurrence ids (would emit ids no consumer could
  feed back to `resolve` — the printing rule forbids composing glyph strings outside package
  `glyph`).

### Entry dispositions, whole-file adds and removes, and per-entry failures

- Decision: the core entry's `Before` and `After` byte slices are each `nil` when the file did not
  exist on that side (`nil` before = the file was added; `nil` after = the file was removed). An
  empty-but-non-nil slice means an existing empty file.

  **`DeltaEntry` carries a pre-extraction refusal.** The entry is
  `(Path, Before, After, BeforeUnit, AfterUnit, Refusal string)`. A non-empty `Refusal` means the
  layer that assembled the batch already decided this entry cannot be extracted: the core skips it
  entirely — no parse on either side — and emits `disposition: error` with `Refusal` as the message,
  in the entry's own position in `files`. Without this field the git layer has no way to express the
  refusals the status-letter table requires of it. The git layer may pre-set a `Refusal` in exactly
  three cases, and no others:
  - an **unmerged** path (`U`): `unmerged path`;
  - an **unrecognised status letter**: a message naming the letter verbatim;
  - a **read failure on either side** — `git show` failing for a blob, or the working-tree file
    being unreadable during batch assembly: a message naming the side and the underlying error.
  The third case is why the field exists rather than the git layer simply dropping the entry: a
  disk-read failure while assembling the batch is neither "a git command that failed" (so exit 3
  does not cover it) nor grounds to fail the batch (which "a failing entry never fails the batch"
  forbids). It is one entry's problem, and this is how one entry reports it. The answer carries a `files` array with one
  entry per input entry, in the input's own order, each holding `path` and a closed `disposition`
  word:
  - `added`, `removed`, `changed` — the file was extracted on the sides where it exists.
  - `unsupported` — the file's extension resolves to no registered strategy. It contributes no
    symbols on either side and is not an error.
  - `error` — extraction failed for this entry, or the entry was refused before extraction (an
    unmerged path, an unrecognised git status letter — see the status-letter table under "Git
    plumbing"); the entry additionally carries `error` with the message. It contributes no symbols
    to any delta list.
  A failing entry never fails the batch: the whole `Delta` call still returns a nil error, and the
  answer is still rendered as a success envelope.

  **Partial parses are reported, and they block assertion.** `treesitter.WithTree` hands back
  `root.HasError()`, and `FileEntry.Lossy` already exists for exactly this state. The `files` entry
  therefore carries `lossy_before` and `lossy_after` flags, set independently per side. A lossy side
  still contributes its surviving symbols, exactly as `walkDir` and `symbolsOfUnit` already do — but
  **no symbol from a lossy side may be asserted at the exact tier** (condition 7 above), because a
  truncated symbol table manufactures spurious `deleted` entries, and a spurious delete is exactly
  the input that turns an exact-tier assertion into a confident lie. The delta is still reported in
  full; the flags are what let a consumer discount it.
- Rationale for the lossy flags: the working-tree side is Loomyard's card-done path, where a
  mid-edit file with a syntax error is normal rather than exceptional. Silently reporting a
  truncated table as ordinary `deleted` entries would tell the pipeline that symbols were removed
  when the file merely does not parse yet. Reporting is enough — refusing the entry outright would
  discard real information, and the `asserted`/`reported` split the whole design turns on says the
  right response to reduced confidence is to stop asserting, not to stop answering.
- Rationale: echoing every entry is what makes the batch auditable — a caller can tell "this file
  had no symbol changes" apart from "this file was never read", which is a distinction a
  mechanical gate must make before it trusts an empty delta. It mirrors `docs/rewrite-plan.md` §4's
  rule that `toc` lists every non-gitignored file, not only source. The per-entry failure isolation
  is the resolve pattern this facade is explicitly asked to follow: "per-entry extraction errors
  fail their entry, never the batch".
- Rejected: dropping non-Go entries silently (a consumer cannot then distinguish an unexamined file
  from an unchanged one); failing the batch on any entry error (one unreadable file would destroy an
  otherwise complete answer).

### Layering: pure core, thin git layer, both on the facade

- Decision: three layers, in three places.
  - `internal/engine/delta.go` — the pure core. `(*engine.Repo).Delta(entries []DeltaEntry)
    (DeltaAnswer, error)`. Knows nothing about git, revisions, or directories. It parses each
    supplied byte slice with `treesitter.WithTree` and the registered `Strategy` for the file's
    extension — the *same* extractor `toc`, `resolve` and `expand` use — and compares the resulting
    tables. The returned error is non-nil only for a failure of the call as a whole, never for one
    entry.
  - `internal/gitsrc` — a new package holding the read-only git plumbing and nothing else: list
    changed paths between two revisions, list untracked working-tree paths, read a blob at a
    revision, list a directory's entries at a revision, and resolve a revision. It shells out with
    `os/exec` and returns **paths, bytes and errors only** — no quarry types, no tree-sitter, and no
    package clauses (a clause can only come from `Strategy.Package` inside `treesitter.WithTree`,
    which is the engine's job, never this package's).
  - `quarry` facade — two methods. `(*Repo).Delta(entries)` delegates to the engine unchanged, as
    every other facade method does. `(*Repo).DeltaGit(from, to, target string) (GitDeltaAnswer,
    error)` is the convenience: it drives `internal/gitsrc` for paths and bytes, turns those bytes
    into clause maps by calling the engine's own exported clause helpers (below), derives the units,
    and then calls `Delta`.
  The CLI calls `DeltaGit`.

  **The clause-map seam, named explicitly.** Three exported engine entry points carry it, because
  package `quarry` cannot reach `dirPackage` or `unitFor` — both are unexported methods and
  functions on `internal/engine`, and export is the only way across the package boundary:
  - `engine.PackageClause(base string, src []byte) (clause string, ok bool)` — resolves the language
    from `base`'s extension via `LanguageForExtension`, looks the `Strategy` up with `StrategyFor`,
    parses `src` inside `treesitter.WithTree`, and returns `Strategy.Package(root, src)`. `ok` is
    false for an unknown extension, unreadable UTF-8, a parse failure, or an empty clause — the same
    four conditions under which `dirPackage` records no clause today. This is the one function that
    turns bytes into a clause, and `dirPackage` is refactored to call it so there is not a second
    copy of that sequence.
  - `(*engine.Repo).ClauseMapForFiles(dirRel string, bases []string) (map[string]string, error)` —
    the on-disk clause map for exactly the base names in `bases`, read from `dirRel` and passed
    through `PackageClause`. It takes the file list rather than enumerating the directory itself,
    which is what lets both sides of a comparison be voted over **one** enumeration rule chosen by
    the caller (see the next decision). Note that `dirPackage` does not filter either: `walkDir`
    hands it entries that are already ignore-filtered, so "reuse `dirPackage`" was never a statement
    about enumeration.
  - `engine.UnitsForClauseMap(dirRel string, clauses map[string]string) (dirPkg string, unitOf
    func(base string) string)` — the vote and the derivation, extracted from `dirPackage`'s
    tie-break and `unitFor` so both callers share one implementation rather than a reimplementation.

  Which layer calls which: for a **revision** side, `DeltaGit` asks `internal/gitsrc` for the
  directory's file names (`git ls-tree`) and each `.go` file's bytes (`git show`), calls
  `engine.PackageClause` on those bytes to build the map, then `engine.UnitsForClauseMap`. For the
  **working-tree** side it enumerates by the same rule (see the next decision), calls
  `(*engine.Repo).ClauseMapForFiles` with that list, and then the same `UnitsForClauseMap`.
  `internal/gitsrc` never sees a clause, a unit, or a tree-sitter node.

  **One enumeration rule, both sides.** The rule is stated as a *set*, not as a command: **the
  immediate `.go` children of the directory — never a subdirectory's files.** That is what
  `dirPackage` votes over (`walkDir` reads one directory with `os.ReadDir` and recurses separately),
  and the unit is a per-directory fact, so a subdirectory's clause must never enter this
  directory's vote.

  The two commands are unequal and must each be trimmed to that set:
  - revision side: `git ls-tree --name-only <rev> <dir>/` is **non-recursive** already, so it yields
    the immediate entries; drop any name that is not a `.go` file.
  - working-tree side: `git ls-files --cached --others --exclude-standard -- <dir>/` is **inherently
    recursive** and has no non-recursive mode, so its output is filtered to paths with **no further
    `/` after the `<dir>/` prefix**, then to `.go` files.

  Without that trim the working-tree side sweeps in subdirectory files the revision side never sees
  — `internal/engine/treesitter/treesitter.go`, clause `treesitter`, would vote in
  `internal/engine`'s ballot on one side only — and the two sides' `dirPkg` can disagree. That
  disagreement changes the unit, which changes every glyph in the directory, which turns the whole
  directory into a create-plus-delete storm: the exact failure this decision exists to prevent.

  Both commands are tracked-inclusive and neither applies the engine's `ignoreSet`, so a
  tracked-but-gitignored `.go` file votes on **both** sides or neither. Reading the working-tree
  side's bytes from disk is a detail of *how the bytes are fetched*, never of *which files count*.

  **A listed file that cannot be read contributes no clause.** `git ls-files --cached` lists index
  entries whether or not the file is present in the working tree, so a `.go` file deleted but not
  staged — a routine case for this verb — is handed to `ClauseMapForFiles` and cannot be read. That
  method **skips** such a base name and records no clause for it, exactly as `dirPackage` already
  does today for an unreadable file, invalid UTF-8, a parse failure, or an empty clause. It does
  **not** return an error for it: a plain unstaged deletion must never fail the whole `DeltaGit`
  call. `ClauseMapForFiles`' `error` return is reserved for a failure of the call itself, never for
  one file's absence.
- Rationale: the task fixes the layering — the core takes byte pairs and knows nothing about git —
  and the reason is testability: a pure core needs no repository, no fixture tree and no commits to
  be exercised exhaustively. Putting `DeltaGit` on the facade rather than only in the CLI is what
  serves the actual primary consumer: Loomyard's pipeline is Go code that binds handles at card-done
  by comparing the working tree against a commit, and forcing it to reimplement the git resolution
  would be a second implementation of the one thing this layer exists to hold. `internal/gitsrc` is a
  separate package rather than functions in `internal/engine` because the engine's own doc contract
  is "reads the source as it is at that moment" with no process spawning anywhere in it.
- Rejected: git resolution in the CLI only (would force Loomyard to reimplement it); git plumbing
  inside `internal/engine` (contradicts the engine's stated character); a `go-git` dependency (a new
  third-party dependency for three plumbing commands, in a repo whose only non-stdlib dependencies
  today are tree-sitter and the MCP SDK).

### Git plumbing: exactly which calls, and `--no-renames`

- Decision: `internal/gitsrc` runs exactly these, all read-only, all with `git -C <root>`:
  - `git rev-parse --show-toplevel`, **first of all**, to verify `<root>` is itself the repository
    top-level (see "The repository root must be git's top-level").
  - `git rev-parse --verify <rev>^{commit}` for each supplied revision, next, so an
    unresolvable revision is reported as such rather than surfacing as a failed diff (see the
    exit-code decision).
  - `git diff --name-status --no-renames <from> <to> -- <pathspec>` for the changed-path list when
    `--to` is a revision; `git diff --name-status --no-renames <from> -- <pathspec>` when the after
    side is the working tree.
  - `git ls-files --others --exclude-standard -- <pathspec>` **when and only when the after side is
    the working tree**, to enumerate untracked files (see the next decision).
  - `git show <rev>:<path>` to read a blob at a revision. The after side reads from disk instead
    when `--to` is absent.
  - `git ls-tree --name-only <rev> <dir>/` to enumerate a changed directory's files at a revision,
    so the package-clause vote for that side can be taken over the whole directory; combined with
    `git show` for each `.go` file's bytes, which the facade turns into clauses via
    `engine.PackageClause`. The working-tree side is enumerated with
    `git ls-files --cached --others --exclude-standard -- <dir>/` and read from disk via
    `(*engine.Repo).ClauseMapForFiles`. Both listings are trimmed to the directory's **immediate**
    `.go` children before voting — `ls-tree` is non-recursive already, `ls-files` is not — so both
    sides vote over the same set.
  Every path-emitting call uses **`-z`** (`git diff --name-status -z`, `git ls-files -z`,
  `git ls-tree --name-only -z`) and the NUL-delimited output is split on NUL. Without it git applies
  `core.quotePath` and C-quotes any non-ASCII or control-character path, and delimits with `\n`, so
  such a path would be read at the wrong location — silently, since a mangled path simply fails to
  open. `-z` also removes the newline-in-filename case entirely.

  Nothing else. No `checkout`, no `stash`, no index write, no config write, no `-M`/`-C`.

  **The status letter is mapped explicitly, and the mapping is total:**

  | letter | meaning | entry |
  |---|---|---|
  | `A` | added | `nil` before, bytes after → `disposition: added` |
  | `M` | modified | bytes both sides → `disposition: changed` |
  | `D` | deleted | bytes before, `nil` after → `disposition: removed` |
  | `T` | typechange (file ↔ symlink) | bytes both sides → `disposition: changed`; a side that is now a symlink yields its link text, which is not parseable Go and therefore produces no symbols on that side, exactly as any other unparseable content does |
  | `U` | unmerged | **no extraction**: `disposition: error`, message `unmerged path` |
  | anything else | — | `disposition: error`, message naming the letter verbatim |

  `R` and `C` cannot appear, because `--no-renames` disables both rename and copy detection; the
  `anything else` row covers them anyway rather than relying on that argument. `U` is reachable on
  the working-tree path during a merge conflict — the primary consumer's own path — and a
  conflicted file's working-tree content is conflict markers, which must never be extracted as if
  it were code.
- Rationale: `--no-renames` is not an optimisation, it is a correctness requirement. Git's rename
  detection is a similarity threshold; letting it run would mean quarry's answer silently inherited
  a heuristic the whole two-tier design exists to replace. With `--no-renames` a rename arrives as a
  delete plus an add and is classified by the table comparison, which is the only classifier this
  query is allowed to have. The clause-vote calls are the price of the unit decision above; they
  are per changed *directory*, not per changed file, and they read package clauses only.
- Rejected: `git diff -M` and trusting git's rename pairs (a threshold heuristic in the one place
  the contract forbids one); `git diff --name-only` alone (loses the add/delete status, which the
  entry disposition needs); statting the target (see the CLI decision).
- **Cost, stated honestly.** The clause vote is *not* "one call per changed directory". It needs, for
  every changed directory and **on both sides**, one `git ls-tree` plus one `git show` process spawn
  and one tree-sitter parse **per `.go` file in that directory** — not per changed file. A one-file
  change in `internal/engine/` (about 20 non-test `.go` files, more with tests) therefore costs on
  the order of 40 `git show` spawns and 40 parses before any delta work begins. That is the price of
  the correct-unit decision, and the plan should price it as such, because the primary consumer
  calls this per card. Two mitigations are available and neither is a cache: the vote is skipped
  entirely for a directory in which no `.go` file changed, and the working-tree side reads from disk
  rather than spawning `git show` per file.

### Untracked working-tree files are included; tracked-but-gitignored files are not excluded

- Decision, two halves.
  1. **Untracked files are enumerated.** When `--to` is absent — the working-tree path — the batch
     is `git diff --name-status --no-renames <from>` **plus**
     `git ls-files --others --exclude-standard`. An untracked file appears with `nil` before-bytes
     and its on-disk bytes after, and therefore with `disposition: added`, exactly like a staged
     new file.
  2. **Tracked-but-gitignored files are not additionally filtered out.** `git diff` and `git ls-tree`
     enumerate tracked content; a file that is both tracked and matched by a `.gitignore` pattern is
     listed by them and is kept in the batch, even though `toc` would not list it (the engine's
     `ignoreSet` filters it out of every walk, and `dirPackage`'s doc comment records that such a
     file "never votes in the tie-break and never contributes a clause"). `--exclude-standard` on
     the untracked enumeration means an ignored *untracked* file is never picked up, so the only
     divergence is the tracked-and-ignored case.
- Rationale for half 1: the working-tree comparison is precisely the case the Problem section names
  as Loomyard's card-done handle binding, and a card that creates a file and has not yet `git add`ed
  it is the normal state at that moment. Without this, that file's symbols would be silently absent
  from `created` — and worse, the `files` echo could not even record the omission, because git never
  listed the file, which defeats the one property that block exists to provide (telling "no symbol
  changes" apart from "never read").
- Rationale for half 2: this is a deliberate, documented divergence from `toc`'s listing rule, not
  an oversight. The question "what changed between these two versions" is a question about tracked
  content; suppressing a real change to a tracked file because some `.gitignore` mentions it would
  hide the change, and the `.gitignore` that governs is itself version-dependent — the chain at
  `<from>` and the chain at `<to>` can differ, so an ignore filter here would need a revision to be
  taken against and would make the two sides' clause votes disagree with each other. Using one
  enumeration rule on both sides keeps the sides mutually consistent, which is the property the
  comparison depends on; agreeing with `toc`'s listing is not, since no part of this query is
  defined as "what `toc` would list". The consequence — `delta` can report a symbol `toc` never
  lists — is stated in the answer type's doc comment.
- Rejected: excluding untracked files (breaks the primary consumer's primary case, silently);
  applying the engine's `ignoreSet` to the git-sourced batch (needs a revision to resolve the
  `.gitignore` chain against, and picking either side makes the two sides' enumerations differ);
  `git ls-files --others` without `--exclude-standard` (would sweep in build output and every
  ignored artefact in the tree).

### CLI shape, and the exit-code contract

- Decision:
  - `quarry delta <target> --from <rev> [--to <rev>] [--text] [--root <path>]`.
  - Exactly one target, as every other verb requires. The target goes through
    `repopath.RepoRelTarget(root, base, target)` first, exactly as `runTOC`'s does, and the
    resulting **repository-relative** path is what is handed to git as the pathspec — `git -C <root>`
    interprets a pathspec against the root, so this is the one conversion that makes the argument
    mean the same thing to quarry and to git. Consequently `.` means **the current directory**, not
    the repository root, when run from a subdirectory — identically to `quarry toc .`, since `base`
    is the working directory unless `--root` is given. To scope the whole repository from a
    subdirectory, pass `--root <root> .` or run from the root, again identically to `toc`.
    `RepoRelTarget`'s existing rejections carry over unchanged: a target escaping the root is exit 1,
    and a target carrying the glyph separator `#` is exit 2.
  - `--from` is required for `delta` and is a usage error (exit 2) when absent. `--to` is optional;
    absent means the working tree.
  - `--from` and `--to` are valid for `delta` only, rejected for the other three verbs with the same
    "`<flag>` is not valid for `<verb>`" message shape `--depth` already uses. `--depth`,
    `--symbols` and `--no-symbols` are rejected for `delta` the same way.
  - **An unresolvable revision is exit 2**, with quarry's own sentence — `unknown revision: <rev>`,
    the value spelled exactly as given — and the usage text on stderr. A `<root>` that is not a git
    repository, or is not that repository's top-level, is exit 2 the same way (see "The repository
    root must be git's top-level"). `internal/gitsrc` runs
    `git rev-parse --verify <rev>^{commit}` for each supplied revision before anything else and
    returns a distinguishable sentinel (`gitsrc.ErrUnknownRevision`) that the CLI checks with
    `errors.Is`, never by parsing git's message.
  - Exit codes: **0** whenever the delta was computed — including an empty delta, and including a
    batch in which some entries have `disposition: error`. **2** for a usage error, which now
    includes an unresolvable revision. **3** for an internal failure (a git command that failed for
    any *other* reason, a render failure, a stdout write failure). **1** is reachable only through
    `RepoRelTarget`'s target-escapes-the-root rejection; the query itself has no negative answer,
    because "nothing changed" is a true answer to "what changed", not a negative one, and `Run`'s
    doc comment says so.
  - **`delta` never stats its target**, unlike `runTOC`, which does its own `os.Lstat` and returns
    exit 1 with `target not found: <rel>`. A pathspec matching nothing produces **exit 0 with an
    empty delta**, not exit 1.
- Rationale for not statting: a target that does not exist *now* may well have existed at `--from`,
  and a deleted directory is exactly the change this query exists to report — `runTOC`'s stat is
  right for a verb that asks "what is here", and wrong for one that asks "what changed". Statting
  would make `delta internal/oldpkg --from HEAD~5` refuse to answer precisely when the answer is
  most interesting. An unmatched pathspec is likewise a true, empty answer rather than a negative
  one: the caller asked what changed under a path, and nothing did.
- Rationale: keeping the one-target rule intact avoids touching `parseArgs`' target-count check for
  one verb's sake. Routing the target through `RepoRelTarget` is what keeps one argument from
  meaning two different things — quarry resolves it against the working directory, git would resolve
  it against the root — and it inherits `toc`'s already-tested escape and separator rejections for
  free. Exit 2 for an unresolvable revision follows `exitUsage`'s own documented meaning ("the
  caller asked wrong … a `--root` that does not resolve to a directory"): a revision that does not
  resolve is the same class of mistake, it is the most likely user error for this verb, and routing
  it to exit 3 would report it as `internal error: ` with git's raw message — the opposite of the
  "quarry spells the conditions it defines" rule `Run`'s doc comment states. Declaring the query
  itself to have no negative answer keeps the four-code contract honest: exit 1 for an empty delta,
  or for a batch containing an errored entry, would each make a successful, complete answer look
  like a failure to a shell gate.
- Rejected: making the target optional for `delta` only (a per-verb exception in the argument
  parser, for no gain over `.`); using the raw target as the git pathspec without `RepoRelTarget`
  (one argument, two meanings); exit 3 for an unresolvable revision; exit 1 on an empty delta;
  exit 1 when any entry errored.

### The answer carries no revision information; the git layer wraps it

- Decision: `DeltaAnswer` — the core's answer — has no `from` or `to` field, and no field naming a
  revision, a commit, or the working tree. `DeltaGit` returns a `GitDeltaAnswer` that embeds the
  `DeltaAnswer` and adds `from` (the revision string as given) and `to` (the revision string as
  given, or `null` for the working tree).

  **`GitDeltaAnswer` is declared in package `quarry`, not in the engine** — it is the only new type
  that is a facade type rather than an engine alias, because the engine is defined as knowing
  nothing about git and a revision-bearing type there would contradict that. The facade's
  "aliases only" habit yields here for one type with one reason. Both renderers take the **wrapped**
  form — `RenderDeltaJSON(a GitDeltaAnswer) ([]byte, error)` and
  `RenderDeltaText(a GitDeltaAnswer) string` — since the CLI is their only caller and it always has
  revisions in hand; a Go caller holding a bare `DeltaAnswer` from the pure `Delta` path wraps it
  with empty `from`/`to` to render, or reads the struct directly, which is what a facade consumer
  does anyway.
- Rationale: the core knows nothing about git, and a field it can never populate would be a lie in
  its own type. Wrapping puts the revision echo exactly where the revisions are known.
- Rejected: an optional `from`/`to` on the core answer left empty by the pure path (an
  always-omitted key on the one call path most consumers use).

### `--text` renders a lossless text view

- Decision: `quarry delta --text` emits a lossless text view of the same data, via a new
  `quarry.RenderDeltaText`. It follows the conventions the three existing text renderers already
  establish in `quarry/text.go`.
- Rationale: `--text` is valid for every verb; a verb that ignored it would be the only exception.
  `docs/rewrite-plan.md` §4 fixes the text view as lossless, and this discussion requires goldens in
  both JSON and text views (see Testing).
- Rejected: JSON only.

## Technical context

**The extractor is reached through `Strategy`, never by naming `goStrategy`.** Every existing call
site resolves the language with `engine.LanguageForExtension(filepath.Ext(base))` and then
`engine.StrategyFor(lang)`; the core must do the same, so "the SAME extractor toc uses" is true by
construction rather than by comment. See `internal/engine/walk.go`'s `fileEntry` and
`internal/engine/resolve.go`'s `symbolsOfDir` for the two existing instances of this pattern.

**Parsing goes through `treesitter.WithTree` only.** `internal/engine/treesitter/treesitter.go` is
the one place in the tree that constructs a `*ts.Parser` or a `*ts.Tree`, and its doc comment is
explicit that callers must never retain the `*ts.Node` past their own callback's return — the node
is invalidated when `WithTree` closes the tree. The core must therefore build its token streams
(and its `[]Symbol`) *inside* the callback, into values that own their own memory (strings, not
node pointers).

**The unit rule lives in two functions.** `unitFor(dirRel, dirPkg, fileClause)` in
`internal/engine/walk.go` is the derivation; `dirPackage(dirRel, entries)` in the same file is the
clause vote (most common non-`_test`-suffixed clause; if every clause ends in `_test`, most common
overall; lexicographically smallest breaks a tie). `dirRel == "."` is a documented special case
returning `""`, and a file whose unit is unspellable by `glyph.Parse` (checked by `unitSpellable`)
contributes no symbols at all — the delta core must honour that same rule. Both are refactored, not
copied: `dirPackage`'s "bytes to clause" step becomes the exported `engine.PackageClause`, its
tie-break plus `unitFor` become the exported `engine.UnitsForClauseMap`, and the on-disk map becomes
the exported `(*engine.Repo).ClauseMapForFiles`, which takes the file list rather than enumerating. Export is not decoration here — package `quarry`
cannot call `dirPackage` or `unitFor`, both unexported, and it is the layer that assembles the batch.

**`SignatureCut`, `SigEnd` and the body-bearing child.** `internal/engine/nodes.go`'s
`SignatureCut(decl, body, src)` returns the trimmed bytes `[decl.StartByte(), body.StartByte())`,
or the whole declaration when `body` is nil; `SigEnd` returns `body`'s start line, or 0 when nil.
The `body` node is resolved per declaration kind by the strategy —
`decl.ChildByFieldName("body")` for Go functions and methods, `goTypeBody(spec)` for types. Read
`goTypeBody`'s doc comment before implementing the token streams: it returns
`field_declaration_list` for a struct but the bare **`"{"` leaf** for an interface, "which the
grammar exposes with no named body node and no 'body' field at all", and nil for `type ID string`
and `type Alias = T`. That asymmetry is exactly why the token streams in the Decisions are defined
by byte range rather than by node containment.

**Grouped declarations pass the *spec*, not the declaration.** `goGroupedTypeSymbol` and
`goGroupedConstOrVarSymbols` call `SignatureCut(spec, …)` and prepend a synthesized keyword —
`"type "`, `"const "`, `"var "` — so `Symbol.Signature` for a grouped shape is *not* the cut bytes
verbatim. The new `DeclStart`/`BodyStart`/`DeclEnd` must be filled from whichever node that builder
passed, so the byte spans and the `Signature` string always describe the same declaration.

**Interface methods are already symbols.** `goInterfaceMethodSymbols` extracts each `method_elem` of
an interface as its own `KindMethod` `Symbol`, owned by the interface, with `SignatureCut(elem, nil,
src)` and no body. Two consequences for the delta: adding a method to an interface shows up both as
a **created** method symbol and as a **modified** interface type (its body stream now covers the new
element) — both are true and neither is redundant; and an interface method's own body stream is
always empty, so two interface methods differing only in their signatures are correctly compared
through the signature stream, not the body one.

**`Symbol.File` is filled by the caller, not the strategy.** `Strategy.Symbols(unit, root, src)`
leaves `File` empty — `symbolsOfDir` assigns `sym.File = fileRel` at its own call site, and
`walkDir` leaves it empty because a toc symbol already sits inside its file's entry. The delta core
must therefore set `File` itself, from the entry's own `path`, on **each side** independently: the
`changed:["file"]` dimension, a `modified` entry's `before` block, and a `renamed` pair's `from`/`to`
all read it, and all three would silently compare empty strings otherwise.

**`Symbol` carries no byte offsets today.** `Strategy.Symbols(unit, root, src)` returns `[]Symbol`
and nothing else, and byte offsets appear only inside `nodes.go`'s `SignatureCut`. The three new
JSON-hidden fields are the whole seam; no `Strategy` method is added, and the delta core does not
walk the tree a second time — it buckets leaves by byte offset in the same `WithTree` callback that
produced the symbols.

**Where the symbol shape comes from.** `internal/engine/answer.go` declares `Symbol`, `Kind`,
`Status`, `DirAnswer`, `FileEntry`, `ResolveResult`, `ExpandAnswer`. Its own file comment states
that every JSON tag there is the closed emitted key set, and that a field is not added or renamed
without a corresponding Shared Decision. The new `DeltaEntry`, `DeltaAnswer`, `DeltaFile`,
`ModifiedSymbol`, `RenamedPair`, `RenameCandidate` and `RenameSignals` types belong in a new file
alongside it (working name `internal/engine/delta_answer.go`) or in `delta.go` itself; they extend
the key set additively and change nothing existing.

**`Symbol.HeadStart`/`HeadEnd` are JSON-hidden** and consumed only by `expand`. The delta core
carries them through unchanged and never compares them — they are a projection of the same
declaration node the other spans come from.

**Two doc comments carry the "adds no behaviour" claim, not one.** `quarry/doc.go` says the facade
"adds no behaviour of its own", and `quarry/repo.go`'s `Open` doc comment repeats it verbatim
("the facade adds no behaviour of its own"). `DeltaGit` is the first method that does more than
delegate, so **both** need the same honest amendment; amending only `doc.go` leaves the other false.

**Facade aliasing.** `quarry/quarry.go` exposes engine types as **aliases** (`type DirAnswer =
engine.DirAnswer`), not defined types, so an external importer can name them without importing
`internal/engine`, and `errors.Is`/`errors.As` stay transitive. The new delta types must be aliased
the same way. `quarry/doc.go` states the facade "adds no behaviour of its own" — `DeltaGit` is the
first method that does more than delegate, so that doc comment needs an honest amendment naming it
and saying why (the git layer is a caller-facing convenience, not query behaviour).

**Renderers.** `quarry/render.go` holds one unexported `renderJSON` that fixes the byte contract:
two-space indent, one trailing newline, `SetEscapeHTML(false)`. Every success renderer delegates to
it; `RenderErrorJSON` deliberately does not (compact failure envelope). `RenderDeltaJSON` must
delegate to `renderJSON`. `quarry/text.go` holds the three text renderers to follow for
`RenderDeltaText`.

**CLI.** `internal/cli/flags.go` holds the hand-rolled `parseArgs` with the verb gate (`toc`,
`resolve`, `expand` today — `delta` is a fourth), the per-flag/per-verb validity checks, and the
"exactly one target" rule. `internal/cli/cli.go` holds `Run`'s dispatch switch, the four exit-code
constants, `fail` (the single failure path: envelope to stdout, same sentence to stderr), and one
`codeFor…` mapping function per verb declared as a named function specifically so a table test can
be written against it — `codeForDelta` should follow that pattern even though it is nearly
constant. `internal/cli/usage.go` holds `usageText`, which is ASCII-only and byte-compared in
tests; it needs the new usage line, the two new flags, and no change to the exit-code block.

**The error-message rule.** `Run`'s doc comment states that an exit-1 or exit-2 message never
carries the engine's wrapped chain, because leaking an internal package name into a public contract
is forbidden; exit 3 carries `err.Error()` whole behind an `"internal error: "` prefix. A failed git
command is an exit-3 condition and therefore carries its message whole — but `internal/gitsrc` must
not prefix its errors with anything that reads like a quarry-internal package path in a
user-visible sentence beyond that one prefix.

**No cgo in `glyph`.** `glyph/` is pure Go and must stay importable without tree-sitter (Loomyard
imports it). All new code needing tree-sitter goes in `internal/engine`, `internal/gitsrc` (no
tree-sitter, but engine-adjacent), `quarry/`, and `internal/cli`. `internal/cgoguard` is
blank-imported by `treesitter` so a `CGO_ENABLED=0` build fails with a clear message.

**Existing exec usage.** `os/exec` appears in this repo today only in `_test.go` files and in the
`bench/` harness. `internal/gitsrc` is the first production package to shell out. It should take an
explicit repository root and never rely on the process working directory (`git -C <root>`), matching
how the tests already invoke git.

**Real-history pin.** `49304ca` ("Glyph self-form and the resolve contract", C1) has parent
`d413ceb`. Over `internal/engine/` it modifies `answer.go`, `expand.go`, `repo.go`, `resolve.go`,
`walk.go`; over `glyph/` it modifies `doc.go`, `errors.go`, `glyph.go`, `golang.go`, `parse.go` and
adds `self.go`. The assertions below are **presence-only over the in-scope subset** — "these
symbols appear in this list" — never exact-set equality, because the full sets were not enumerated
here and two of the known deletes fall outside the pinned directories:
- **created** (in scope): `glyph#Self`, `glyph#Glyph.IsSelf`, `internal/engine#SelfGlyphError`
  (the type, declared in `expand.go`) and `internal/engine#SelfGlyphError.Error` (its method),
  `internal/engine#Repo.resolveSelfTarget`.
- **deleted** (in scope): `internal/engine#Repo.resolvePathTarget`. Also deleted in this commit but
  **outside** the `glyph/` + `internal/engine/` scoping, and therefore not asserted by this test:
  `internal/repopath#RepoRelPath` and `internal/cli#isGlyphTarget`.
- **modified**: several, across both units; asserted as "at least one per unit", not enumerated.
- **an evidence-tier rename**: `Repo.resolvePathTarget` → `Repo.resolveSelfTarget` — same unit
  (`internal/engine`), same owner (`Repo`), same kind (method), different name, and a changed body
  and signature (`target string` → `unit string`), which is exactly the demotion the evidence tier
  exists for.
No exact-tier rename occurs naturally in this history; the exact tier is asserted by a synthetic
unit test instead — see Testing, where this discussion requires it.

## Constraints

- No `CONSTRAINTS.md` exists at the hub root.
- `CLAUDE.md`: this is a Go repo; **do not introduce Python**.
- Go only, as a language target. No Python or C# extractor work here.
- Additive only: no existing envelope, verb, flag, exit code, or golden changes. **Every existing
  committed golden stays byte-identical, wherever it lives at merge time.** Today those are
  `docs/research/output-formats/after/` and `internal/mcpserver/testdata/golden/`, but a parallel
  `mill-quick`-sized task listed under `docs/roadmap.md`'s "Small and independent" section is moving
  the `docs/research/output-formats/after/` set to `internal/cli/testdata/`. If that task merges
  first, this constraint and the Testing section's `compareAfterGolden` reference name the new
  location, not the old one — the requirement is the bytes, never the path.
- Quarry reads. Never writes, never touches the working tree or the index. Every git call is read
  plumbing.
- No MCP tool for this query.
- No cache, no index, no daemon, no parser pool (`docs/rewrite-plan.md` §10).
- **Nothing quarry decides is fuzzy.** No threshold, no score cut-off, and no similarity value
  anywhere in a classification path: every asserted outcome (`created`, `deleted`, `modified`,
  `renamed`) is reached by exact structural tests only. `body_token_similarity` is a **reported
  signal on a candidate quarry does not resolve**, and is the one similarity value in the query;
  `docs/rewrite-plan.md` §9's "Fuzzy matching of any kind" non-goal governs what quarry *decides*
  — "Unknown is `not_found`; several is `ambiguous`" — and the evidence tier is the `ambiguous`
  side of that rule, not an exception to it.
- No new third-party dependency.
- `glyph/` stays cgo-free.
- `go test ./... && golangci-lint run` green.

## Testing

_The acceptance requirements below — the seven golden cases, both output views, the synthetic
exact-tier assertion, and the real-history check — originate in the wiki task body's "Done when"
section for slug `diff-to-symbols`. They are **not** in `_mill/status.md` (whose `task_description`
is one line) and **not** in `docs/roadmap.md` point 2c (which carries no such clause), so they are
restated in full here: a plan writer needs no wiki access to know what "done" means._

**TDD candidates, in order:** (1) the body-token-stream extraction and the
identical-modulo-identifier predicate; (2) the table comparison producing
created/deleted/modified; (3) the two-tier rename classifier. All three are pure functions over
in-memory data and have no reason to be written after their tests.

**`internal/engine` — the pure core (table tests, no fixtures).** The core takes byte slices, so
every case is two string literals in the test file. Scenarios that must be covered:

- created only; deleted only; modified only; a file added (`nil` before); a file removed (`nil`
  after); an empty batch.
- a symbol whose only change is its line position, asserted **absent** from every delta list.
- each `changed` dimension in isolation — body-only, signature-only, doc-only, file-only (the same
  symbol moved between two files in one unit) — and a combination.
- a reformatted body (whitespace and comment changes only) asserted **not** modified.
- **anonymous-token changes, the regression the stream definition exists to prevent:** a body
  changing `x++` to `x--`, and one changing `a + b` to `a - b`, each asserted **modified** with
  `changed:["body"]`; and a rename whose bodies differ only in such a token asserted **demoted to
  the evidence tier**, never present in `renamed`. A named-leaf-only stream would pass every other
  test in this list and fail these four.
- a signature rename hazard: `func (r *Runner) Run() error` renamed to
  `func (r *Runner) Execute() error`, asserted `signature_identical_modulo_name: true` — a textual
  `Run`→`Execute` substitution would corrupt the `Runner` receiver and get this wrong.
- **interface declarations, the second regression the byte-range split exists to prevent:** adding a
  method to an interface asserted `modified` with `changed:["body"]`; and two unrelated interfaces
  of the same owner and kind (`type Reader interface{ Read() }` deleted, `type Closer interface{
  Close() }` created) asserted **not** an exact-tier rename. Under a node-containment definition
  both would fail, because `goTypeBody` returns the bare `"{"` leaf for an interface.
- a struct with a changed field, and a `type Alias = T` changed to `type Alias = U`, each asserted
  `modified` — the nil-body branch must not collapse to an empty comparison.
- the **signature invariant, per shape**: for each of the five builders (ungrouped func/method,
  ungrouped type, grouped type, ungrouped const/var, grouped const/var, plus interface `method_elem`)
  the signature token stream's byte span equals the span that builder handed `SignatureCut`, and
  `Symbol.Signature` equals that span trimmed with the synthesized `"type "`/`"const "`/`"var "`
  prefix where the builder adds one. A flat byte-identity assertion would be false for the grouped
  shapes.
- **`func init` multi-occurrence across all four dimensions:** two `init` before and two after
  differing only in the *doc* of one of them, asserted `modified` with `changed:["doc"]` — a
  multiset of body hashes alone would report nothing here. Likewise a signature-only difference.
- a `modified` entry's `after`/`before` arrays asserted length one for an ordinary symbol and
  length two for a two-occurrence `init`, with the multiplicity change visible from the lengths.
- **interface method extraction:** adding a method to an interface asserted to produce *both* a
  `created` `KindMethod` symbol owned by the interface *and* a `modified` entry for the interface
  type, since `goInterfaceMethodSymbols` already emits interface methods as their own symbols.
- exact-tier rename: identical body modulo the identifier, including a recursive self-call, with the
  pair asserted present in `renamed` and **absent** from `created` and `deleted`.
- exact-tier demotion: the same rename with one extra statement in the body, asserted to land in
  `rename_candidates` with the create and delete both **still present**.
- exact-tier ambiguity: two deleted symbols each pairing exactly with one created symbol, asserted
  to produce no `renamed` entry at all and evidence entries instead.
- cross-file rename inside one unit, asserted exact; cross-unit rename, asserted **not** paired on
  either tier (it is a plain create plus delete).
- `func init` multiplicity: two `init` before, two after with one body changed → exactly one
  `modified` entry; two before, three after → one entry with the count fields set; asserted never a
  rename candidate.
- a `const` replaced by a `var` of the same name → one create and one delete, never a modification.
- an entry whose extension has no strategy → `disposition: unsupported`, no symbols, no error.
- **bodyless kinds are never asserted:** `const A = 1` deleted plus `const B = 1` created asserted
  **not** in `renamed`, but present in `rename_candidates` with
  `signature_identical_modulo_name: true` and `body_token_similarity: 1.0` over two empty streams;
  likewise a renamed `var`, a renamed `type Alias = T`, and a renamed interface method. Without
  condition 7 every one of these is a false assertion.
- **a partial parse:** a syntactically broken after side asserted to set `lossy_after` on the file
  echo, to still contribute its surviving symbols, and to block any exact-tier assertion involving
  that file — a rename that would otherwise be exact is demoted to evidence.
- **an exact-tier pair carries both symbols in full:** the `renamed` entry's `from` and `to` each
  asserted to hold `id`, `kind`, `file` and spans, and both constituents asserted absent from
  `created`/`deleted` — the pair is the only surviving record of either location.
- **leaf assignment is many-to-many:** an interface `method_elem`'s leaves asserted present in both
  the interface type's body stream and that method symbol's own signature stream; the several
  symbols of `const a, b = 1, 2` asserted to share one identical stream.
- **scoped-batch classification:** a rename whose other half lies outside the pathspec asserted to
  appear as a plain `deleted` with no candidate, proving classification is relative to the batch.
- a per-entry extraction failure (invalid UTF-8) inside an otherwise good batch → that entry
  `disposition: error` with a message, every other entry's symbols present, `Delta` returns a nil
  error.
- unit handling: an entry whose supplied unit is unspellable contributes no symbols, matching
  `unitSpellable`'s rule.

**`internal/engine` — the unit-derivation helper.** Table tests over `map[base]clause` inputs
covering: a plain package; a package with an external test file (`foo` + `foo_test` → two units); a
package legitimately named `httptest` (asserted **not** split); a tie broken lexicographically; a
directory where every clause ends in `_test`; `dirRel == "."`. These are the cases the "voting over
changed entries alone" rejection above names, and they are the reason this helper exists.

**`internal/gitsrc`.** Tests build a small throwaway git repository in `t.TempDir()` with a handful
of commits (using `os/exec`, as the existing `loomyard_test.go` helpers already do), then assert:
the changed-path list and its statuses; blob reads at a revision; the directory listing at a
revision; the working-tree-as-after-side path; and that a rename in that fixture arrives as a
delete plus an add (proving `--no-renames` is in force). A failing git command surfaces as an
error, not a panic. Additionally:
- **an untracked, never-`git add`ed `.go` file in the working tree is enumerated** and reaches the
  delta as `disposition: added` with its symbols in `created` — the case Loomyard's card-done
  binding depends on;
- an untracked file matched by `.gitignore` is **not** enumerated (`--exclude-standard` in force);
- a tracked-but-gitignored `.go` file **is** kept, with the divergence from `toc`'s listing
  asserted deliberately so the decision cannot be silently reverted;
- an unresolvable revision returns `gitsrc.ErrUnknownRevision`, matched with `errors.Is`, before any
  diff is attempted;
- **the status-letter mapping is total**: a table test over `A`, `M`, `D`, `T`, `U` and an
  unrecognised letter, asserting the disposition each produces and that `U` yields
  `disposition: error` with no extraction attempted — a conflicted file's content is conflict
  markers, and extracting it as Go would be a silent lie;
- **both sides of a directory's clause vote enumerate the same file set**: a fixture with a
  tracked-but-gitignored `.go` file asserts that file votes on the revision side and on the
  working-tree side alike, so `dirPkg` — and therefore every glyph unit in the directory — agrees.
  A divergence here turns a whole directory into a create-plus-delete storm, so this is asserted
  directly rather than left to follow from the enumeration code;
- **ordering is total**: a fixture containing `const a, b = 1, 2` (every name sharing one `Start`
  and `End`) asserted to produce a byte-identical answer across repeated runs — file-then-`Start`
  alone cannot separate those symbols, so this is the case that proves the `(file, Start, id, kind)`
  tie-break is doing work; and a `modified` entry's `changed` array asserted in the closed set's
  declaration order regardless of discovery order;
- **a pre-set `Refusal`**: an entry carrying one asserted to be skipped without any parse on either
  side and to surface as `disposition: error` with that exact message, in its own input position;
- **`-z` parsing**: a fixture path containing a non-ASCII character asserted to round-trip, which it
  cannot under `core.quotePath`'s C-quoting without `-z`;
- **top-level normalisation**: a repository reached through a symlinked path (`t.TempDir()` on a
  platform with a symlinked temp directory is exactly this) asserted **accepted**, not rejected as
  `ErrRootNotTopLevel` — the check must compare `EvalSymlinks`'d paths on both sides;
- **the immediate-children trim**: a fixture directory with a `.go` file in a *subdirectory*
  declaring a different package asserts that file votes on **neither** side — `git ls-files` is
  recursive and `git ls-tree` is not, so without the trim the two sides disagree;
- **an unstaged deletion**: a `.go` file removed from the working tree but still in the index is
  listed by `git ls-files --cached`, and `ClauseMapForFiles` asserted to skip it and record no
  clause rather than return an error — a plain deletion must not fail the whole call;
- **root-vs-top-level**: a `--root` pointing at a subdirectory of a repository asserted to produce
  `ErrRootNotTopLevel`, and a `--root` outside any repository `ErrNotARepository`, both before any
  diff runs.

**`quarry` facade and renderers.** Key-order and byte-contract tests for `RenderDeltaJSON`
mirroring the existing `TestRenderExpandJSON_KeyOrder` shape; a text-view test mirroring
`TestRenderExpandText`. A test asserting `DeltaGit` on a fixture repository agrees with
`Delta` called on the same entries assembled by hand — the two paths must not be able to disagree.

**Ordering.** One test asserts every top-level array's stated order (see the ordering decision) on a
batch large enough that Go's randomised map iteration would surface — the goldens below are
byte-unstable without it, so this is asserted directly rather than relied upon implicitly.

**Goldens.** Committed golden files (JSON and text) under `internal/engine/testdata/delta/` or
`internal/cli/testdata/`, following the existing `compareGolden`/`compareAfterGolden` pattern
(wherever those helpers' own fixtures live at merge time — see Constraints on the in-flight
goldens move), for
the seven cases enumerated here: created, deleted, modified, exact-tier rename,
evidence-tier rename, a mixed batch, and a per-entry extraction failure inside an otherwise good
batch. Goldens are produced under the existing `-update` flag convention and are never
hand-written.

**CLI.** Table tests over `Run` for: `--from` absent → exit 2 with the usage text; `--from` or
`--to` given to `toc`/`resolve`/`expand` → exit 2 naming the flag and the verb; `--depth` given to
`delta` → exit 2; zero or two targets → exit 2; a well-formed call producing an empty delta → exit
0; a batch containing an errored entry → exit 0 with the error visible in the payload; **an
unresolvable revision (`--from bogus`) → exit 2 with `unknown revision: bogus` and the usage text,
never git's raw message**; a git invocation failing for another reason → exit 3 with the
`internal error: ` prefix; a target escaping the root → exit 1; a target carrying `#` → exit 2;
`--text` producing the text view; **a pathspec matching nothing → exit 0 with an empty delta**, and
a target naming a path that no longer exists but did at `--from` → exit 0 with that path's symbols
in `deleted`, proving `delta` performs no stat where `runTOC` does. A `codeForDelta` table test
mirroring `TestCodeForTOCError`; a `--root` that is a subdirectory of a repository, and one outside
any repository, each → exit 2 with quarry's own sentence rather than exit 3 with git's.
One test must pin the **target-resolution** rule: `delta .` run from a subdirectory scopes to that
subdirectory, and produces the same scope `toc .` does from the same directory — the two verbs
resolve one argument the same way or the CLI has two meanings for it.

**Real-history check (T3-style).** One test running `DeltaGit("d413ceb", "49304ca", ...)` against
this repository, scoped separately to `glyph/` and to `internal/engine/`, asserting a hand-verified
expectation: **presence** of each symbol listed under "Real-history pin" above in its stated list
(never exact-set equality — the pin's sets are explicitly partial), at least one `modified` entry
per unit, and the `resolvePathTarget` → `resolveSelfTarget` pair present in
`rename_candidates` (not in `renamed`) with `signature_identical_modulo_name: false`. The test must
skip cleanly — never fail — when either revision is unreachable (a shallow clone), following the
asymmetry `loomyardRepo`'s doc comment already establishes for a missing versus a wrong-commit
checkout.

**Docs.** `docs/rewrite-plan.md` §5 gains a one-paragraph `delta` contract after the `toc`
paragraph, and §7's mechanical-use list has its "diff-to-symbols" item updated to name the built
verb. No other document changes on this branch.

## Q&A log

- **Q:** How does the pure core obtain each symbol's glyph unit, given that `unitFor` needs the directory's dominant package clause and a byte pair carries no directory context? **A:** [auto-pick] Each core entry carries an explicit `Unit` per side; a named exported helper above the core derives it per changed directory per side, reusing the engine's existing vote rule. **Why:** the unit is a directory-level fact; voting over changed files alone is silently wrong when the directory's real clause is absent from the changed set, and the path+own-clause shortcut is the one `walk.go` explicitly warns against (`httptest`).
- **Q:** What is the CLI verb name? **A:** [auto-pick] `delta`. **Why:** quarry's verbs name the question, not the mechanism, and `delta` cannot be misread as a textual diff — which is the exact misreading this design exists to prevent, in the one place a reader has no doc comment.
- **Q:** How are the two rename tiers shaped in the output? **A:** [auto-pick] Two separate keys — `renamed` (exact, asserted, pair removed from created/deleted) and `rename_candidates` (evidence, reported, create and delete both retained). **Why:** an asserted outcome changes the answer, a reported one only adds information; suppressing the create and delete for an evidence candidate would be a silent pick in disguise.
- **Q:** What counts as `modified` — does a doc-comment-only change? **A:** [auto-pick] Yes; body, signature, doc and file each count, and a `changed` array names which. A pure line shift never counts. **Why:** all four matter to a real consumer and none subsumes the others; collapsing them would force every consumer to re-derive the distinction.
- **Q:** Is exact-tier rename detection per-file or unit-wide? **A:** [auto-pick] Unit-wide across the batch — same unit, owner and kind, different name, and exactly one candidate or demote to evidence. **Why:** the unit is a glyph's identity scope; a cross-file move inside a package is routine, a cross-package move is a different fact that must not be smuggled in under the rename name.
- **Q:** What similarity signals does the evidence tier report? **A:** [auto-pick] Mechanical, individually explainable fields plus one token-multiset Jaccard, no composite score, ordering documented as deterministic and explicitly not a verdict. **Why:** a blended score invites "take the top one", which is the silent pick the contract forbids.
- **Q:** What gates the evidence-tier candidate set? **A:** [auto-pick] Structural facts only (same unit, kind, owner; different name) — no similarity threshold anywhere. **Why:** a floor is a tuning knob, and the answer must never depend on a magic number. The `d × c` pathological case is accepted and documented rather than capped, since a cap is a threshold under another name.
- **Q:** How are whole-file adds/removes and unsupported files represented? **A:** [auto-pick] `nil` bytes mean absent on that side; every entry is echoed in a `files` block with a closed `disposition` word. **Why:** a caller must be able to distinguish "no symbol changes" from "never read" before trusting an empty delta.
- **Q:** How do per-entry extraction failures surface? **A:** [auto-pick] `disposition: error` plus a message in the `files` block; no symbols from that entry; the batch still succeeds and exits 0. **Why:** the resolve pattern the task mandates — per-entry errors fail their entry, never the batch.
- **Q:** Where do the core, the git layer and the facade methods live? **A:** [auto-pick] `internal/engine/delta.go` (pure), a new `internal/gitsrc` (read-only plumbing), and two facade methods `Delta` and `DeltaGit`; the CLI calls `DeltaGit`. **Why:** Loomyard's pipeline is a Go caller that needs the git convenience too; putting it only in the CLI would force a second implementation of the one thing that layer exists to hold.
- **Q:** What are `--from`/`--to` semantics, and does `delta` keep the one-target rule? **A:** [auto-pick] `--from` required, `--to` optional (absent = working tree); exactly one target as every verb. **Why:** keeps `parseArgs`' target-count rule intact. _(Superseded in part by review r2: the claim that `.` means the repository root was wrong — see the r2 entry below. The target goes through `RepoRelTarget`, so `.` means the current directory.)_
- **Q:** Which git commands, and is git's own rename detection used? **A:** [auto-pick] `git diff --name-status --no-renames`, `git show`, `git ls-tree`; git's `-M` is never used. **Why:** `--no-renames` is a correctness requirement, not an optimisation — git's rename detection is a similarity threshold, exactly what the two-tier design replaces.
- **Q:** Does `delta` support `--text`? **A:** [auto-pick] Yes, a lossless text view. **Why:** `--text` is valid for every verb, and this discussion requires goldens in both views.
- **Q:** What are `delta`'s exit codes? **A:** [auto-pick] 0 on any computed delta (empty included, errored entries included), 2 usage, 3 internal; exit 1 documented as unreachable. **Why:** "nothing changed" is a true answer, not a negative one; the alternatives would make a complete answer look like a failure to a shell gate.
- **Q:** How is a symbol that moved between files reported? **A:** [auto-pick] `modified` with `changed:["file"]` plus a `before` block carrying the prior file and span. **Why:** Loomyard's handle binding needs the new location, and the glyph is unchanged so it is not a create/delete pair.
- **Q:** What keys the symbol table, and how is `func init` handled? **A:** [auto-pick] `(ID, Kind)`; a key with multiplicity > 1 is compared as a multiset of body-token hashes and is never a rename candidate. **Why:** including `Kind` makes a const→var swap a delete plus a create, which is truthful; minting per-occurrence ids would emit glyphs no consumer could feed back to `resolve`.
- **Q:** Which commits does the real-history check pin? **A:** [auto-pick] `d413ceb..49304ca` (C1), scoped to `glyph/` and `internal/engine/`, hand-verified, skipping cleanly when the revisions are unreachable. **Why:** it contains real creates, deletes, modifications, and a genuine evidence-tier rename (`resolvePathTarget` → `resolveSelfTarget`); no exact-tier rename occurs naturally, so the exact tier is asserted synthetically instead.
- **Q:** What is the test approach? **A:** [auto-pick] Pure table tests over in-memory byte pairs (TDD on comparison and tiering), committed JSON/text goldens produced under `-update`, CLI table tests, a `t.TempDir()` git fixture for `internal/gitsrc`, and the real-history test. **Why:** the core is pure by design specifically so it needs no fixture tree to be exercised exhaustively.
- **Q:** Which documents change? **A:** [auto-pick] `docs/rewrite-plan.md` §5 gains the `delta` contract paragraph and §7's mechanical-use item is updated; `docs/roadmap.md` is untouched. **Why:** the roadmap states what is ahead; removing a completed item belongs to the merge, not this branch.
- **Q:** Does the answer echo the revisions it was computed from? **A:** [auto-pick] No — the core answer carries no revision information; `DeltaGit` returns a wrapper adding `from` and `to` (`null` for the working tree). **Why:** the core knows nothing about git, and a field it can never populate would be a lie in its own type.
- **Q:** [review r1, BLOCKING] The body token stream was defined over *named* leaf nodes, but operators and keywords are anonymous in the tree-sitter Go grammar — should they be included? **A:** [auto-pick] Yes: every leaf, anonymous included. **Why:** with named leaves only, `a + b` → `a - b` reports *unchanged* and two symbols differing only in operators can be **asserted** as an exact-tier rename — wrong in the one tier quarry asserts. The substitution rule keys on `identifier` nodes, so widening the stream does not widen the substitution.
- **Q:** [review r1, NIT] "Signature identical modulo the renamed identifier" was specified against verbatim text, which has no nodes to key on. **A:** [auto-pick] Compare the signature's own token stream under the same node-based rule. **Why:** a textual `Run`→`Execute` also hits the `Runner` receiver in `func (r *Runner) Run() error`.
- **Q:** [review r1, NIT] The goldens constraint pinned a path a parallel task is relocating. **A:** [auto-pick] Require the bytes, not the path, and name the in-flight move. **Why:** `goldens-move` may merge first, inviting a false "constraint violated" reading.
- **Q:** [review r2, BLOCKING] `goTypeBody` returns the bare `"{"` leaf for an interface, so a node-containment body stream is empty for every interface. **A:** [auto-pick] Define both streams by **byte range** split at `SignatureCut`'s own cut point. **Why:** node containment makes interface method-set changes invisible and lets two unrelated interfaces be asserted as a rename; the mirror rule ("declaration minus body child") instead leaves interface methods in the *signature* stream, where `Symbol.Signature` does not have them.
- **Q:** [review r2, BLOCKING] "Signature token stream" had two incompatible readings, so `changed:["signature"]` and `signature_identical_modulo_name` could be computed over different spans. **A:** [auto-pick] Same byte-range fix — the signature stream is byte-for-byte the span `SignatureCut` returns. **Why:** the text comparison and the token comparison must be over the same bytes by construction, not by agreement.
- **Q:** [review r2, BLOCKING] With `--to` absent, `git diff` lists tracked files only — what happens to a file a card created but never `git add`ed? **A:** [auto-pick] Enumerate untracked files with `git ls-files --others --exclude-standard`; they arrive as `disposition: added`. **Why:** the working-tree path *is* Loomyard's card-done binding, and without this the file's symbols are silently absent from `created` with the `files` echo unable to record the omission.
- **Q:** [review r2, BLOCKING] `internal/gitsrc` was specified as holding no quarry types and no tree-sitter, yet also as building the per-directory clause maps — which only `Strategy.Package` inside `treesitter.WithTree` can produce. **A:** [auto-pick] Name three exported engine entry points — `PackageClause`, `(*Repo).ClauseMapForFiles`, `UnitsForClauseMap` — and have the facade call them; `gitsrc` returns paths, bytes and errors only. **Why:** `dirPackage` and `unitFor` are unexported, so package `quarry` cannot reach them; export is the only seam across the boundary, and the refactor keeps one implementation of each rule.
- **Q:** [review r2, NIT] Is the git-sourced batch gitignore-filtered, given `toc` never lists an ignored file? **A:** [auto-pick] Untracked files are filtered (`--exclude-standard`); tracked-but-gitignored files are kept, as a documented divergence from `toc`'s listing rule. **Why:** the `.gitignore` chain is itself version-dependent, so filtering would need a revision to be taken against and would make the two sides' enumerations — and their clause votes — disagree with each other; mutual consistency between the sides is the property the comparison depends on, agreement with `toc` is not.
- **Q:** [review r2, NIT] The rationale claimed "`.` already means the repository root everywhere else in the CLI" — is that true? **A:** [auto-pick] No, it was wrong. `runTOC` resolves through `RepoRelTarget(root, base, target)` with `base = cwd` unless `--root`, so `toc .` from a subdirectory names that subdirectory. `delta`'s target goes through the same call, and `.` means the current directory. **Why:** git would resolve a raw pathspec against the root while quarry resolves against the cwd — one argument with two meanings.
- **Q:** [review r2, NIT] "No fuzzy matching anywhere in the query" contradicted emitting a Jaccard float. **A:** [auto-pick] Narrow the constraint to "nothing quarry decides is fuzzy" and cite `docs/rewrite-plan.md` §9. **Why:** §9's non-goal governs how quarry *answers*; the evidence tier is the `ambiguous` branch of that same rule, and no asserted outcome reads the similarity value.
- **Q:** [review r2, NIT] `--from bogus` would fail inside git and land on exit 3 with git's raw message. **A:** [auto-pick] `git rev-parse --verify` each revision first; an unresolvable one is exit 2 with `unknown revision: <rev>`, matched via an `errors.Is` sentinel. **Why:** `exitUsage` already means "the caller asked wrong … a `--root` that does not resolve"; a revision that does not resolve is the same class, and it is this verb's most likely user error.
- **Q:** [review r3, BLOCKING] The token streams were defined over `decl`/`body` nodes, but `Strategy.Symbols` returns only `[]Symbol` and `Symbol` carries no byte offsets — what is the seam? **A:** [auto-pick] Add three JSON-hidden byte offsets to `Symbol` (`DeclStart`, `BodyStart`, `DeclEnd`), filled by each builder from the same node it already hands `SignatureCut`; the core buckets leaves by offset in the same `WithTree` callback. **Why:** it needs no new `Strategy` method and no second walk, and it follows the `HeadStart`/`HeadEnd` precedent for JSON-hidden fields serving one consumer.
- **Q:** [review r3, BLOCKING] "`Symbol.Signature` is byte-for-byte the signature stream's span" — is that true? **A:** [auto-pick] No. `goGroupedTypeSymbol` builds `"type " + SignatureCut(spec, …)` and `goGroupedConstOrVarSymbols` builds `"const "/"var " + …`, and grouped shapes pass the *spec* rather than the declaration. The invariant is restated per shape: same byte span, trimmed, plus a synthesized keyword prefix for grouped declarations. **Why:** the flat claim was false, and the test it justified could not have passed.
- **Q:** [review r3, BLOCKING] A multi-occurrence key was compared as a multiset of *body* hashes while its `changed` array names four dimensions — so a doc-only change to one of two `init` functions reports nothing. **A:** [auto-pick] Compare a multiset of full comparison tuples `(body hash, signature, doc, file)`, and make the `modified` entry's `after`/`before` uniformly arrays. **Why:** the comparison and the reporting must cover the same dimensions, and arrays remove the need for a second entry shape or a `count_before`/`count_after` pair.
- **Q:** [review r3, BLOCKING] `git diff --name-status` also emits `T` and `U` — how do they map? **A:** [auto-pick] A total table: `A`→added, `M`/`T`→changed, `D`→removed, `U`→`disposition: error` with no extraction, anything else→error naming the letter. **Why:** `U` is reachable on the working-tree path during a conflict, and a conflicted file's content is conflict markers; extracting it as Go would be a silent lie.
- **Q:** [review r3, NIT] The clause vote used `git ls-tree` on a revision side and an ignore-filtered on-disk read on the working-tree side, so a tracked-and-ignored file voted on one side only. **A:** [auto-pick] One enumeration rule for both sides, chosen by the facade and passed to `ClauseMapForFiles` as an explicit file list. **Why:** a `dirPkg` differing between the sides changes the unit and turns the whole directory into a create-plus-delete storm. (Also corrected: `dirPackage` does not filter — `walkDir` hands it pre-filtered entries.)
- **Q:** [review r3, NIT] Scope still said `.` means the whole repository, contradicting the r2 correction. **A:** [auto-pick] Corrected in Scope and the superseded Q&A entry annotated. **Why:** a plan writer reading Scope alone would implement the withdrawn rule.
- **Q:** [review r3, NIT] The unit helper was named `UnitsForDir` in one decision and `UnitsForClauseMap` elsewhere. **A:** [auto-pick] `UnitsForClauseMap` throughout. **Why:** one name; the map-taking form is what both callers share.
- **Q:** [review r4, BLOCKING] A partial parse (`root.HasError()`) had no representation, so a syntactically broken side reports a truncated table as ordinary `deleted` entries. **A:** [auto-pick] Add `lossy_before`/`lossy_after` to the file echo; the side still contributes its surviving symbols, but no symbol from a lossy side may be asserted at the exact tier. **Why:** the working-tree side is the card-done path, where mid-edit files are normal; a spurious delete is exactly the input that turns an assertion into a confident lie, and the right response to reduced confidence is to stop asserting, not to stop answering.
- **Q:** [review r4, BLOCKING] The `renamed` entry's payload was never specified, yet an exact pair removes both constituents from `created`/`deleted`. **A:** [auto-pick] The pair carries `from` and `to` as full `Symbol`s. **Why:** removal makes the pair the only surviving record of the created symbol's file and span — which is what Loomyard's handle rebinding needs.
- **Q:** [review r4, BLOCKING] "Exit 1 is reachable only through `RepoRelTarget`, exactly as for `toc`" — true? And does `delta` stat its target? **A:** [auto-pick] The `toc` claim was wrong (`runTOC` also returns exit 1 from its own `os.Lstat`); `delta` performs **no** stat, and an unmatched pathspec is exit 0 with an empty delta. **Why:** a path that does not exist now may have existed at `--from`, and a deleted directory is exactly the change this query exists to report.
- **Q:** [review r4, BLOCKING] `const`, `var`, type aliases and interface methods all have `body == nil`, so `BodyStart == DeclEnd` and the exact tier's body condition is vacuous — `const A = 1` / `const B = 1` would be asserted as a rename. **A:** [auto-pick] Add condition 7: a non-empty body stream on both sides (and no lossy side) is required for the exact tier; bodyless kinds are confined to the evidence tier. **Why:** a signature-only match is not evidence enough to assert on, and this is the same false-assertion failure already called a defect for two unrelated interfaces.
- **Q:** [review r4, NIT] "Buckets each leaf into the stream its start byte falls in" is singular, but symbol spans nest and overlap. **A:** [auto-pick] A leaf is assigned to **every** containing symbol range; the overlap is intended. **Why:** an interface `method_elem`'s leaves belong to the interface's body stream and to that method's own signature stream at once, and `const a, b = 1, 2`'s symbols share one span verbatim.
- **Q:** [review r4, NIT] "Per changed directory, not per changed file" understated the clause-vote cost. **A:** [auto-pick] Restated: one `git show` spawn plus one parse per `.go` file of every changed directory, on both sides — roughly 40 spawns for a one-file change in `internal/engine/`. **Why:** the primary consumer calls this per card, so the plan must price it; the two available mitigations (skip directories with no changed `.go` file, read the working-tree side from disk) are noted and neither is a cache.
- **Q:** [review r4, NIT] Rename pairing is "unit-wide across the batch", but the batch is pathspec-scoped. **A:** [auto-pick] State that classification is relative to the scoped batch: a rename whose other half falls outside the target is a plain `deleted` with no candidate. **Why:** quarry cannot pair against a symbol it was never given, and Loomyard's scope guard passes narrow batches by design.
- **Q:** [review r5, BLOCKING] "One enumeration rule, both sides" named two commands that do not agree — `git ls-tree` is non-recursive, `git ls-files` is inherently recursive. **A:** [auto-pick] State the rule as a *set* — the directory's immediate `.go` children — and trim each command to it (`ls-files` output filtered to paths with no further `/` after the prefix). **Why:** otherwise `internal/engine/treesitter/treesitter.go` votes in `internal/engine`'s ballot on one side only, the two `dirPkg` values diverge, and the directory becomes a create-plus-delete storm — the exact failure the decision exists to prevent.
- **Q:** [review r5, BLOCKING] No ordering was stated for `created`, `deleted`, `modified`, `renamed` or `rename_candidates`, yet they are built from a `(ID, Kind)`-keyed map. **A:** [auto-pick] A total ordering rule per array — file-then-`Start` for symbol lists (reusing `symbolsOfUnit`'s rule), key order for `modified`, `id` order for the rename arrays, input order for `files` — documented as ordering, never ranking. **Why:** Go randomises map iteration, so the seven required goldens cannot be byte-stable without it, and a pipeline diffing two deltas would see phantom changes.
- **Q:** [review r5, BLOCKING] `repopath.ResolveRoot` accepts any directory for `--root` and skips `.git` discovery, so `git -C <root>` can run in a repository whose top-level is above `<root>`. **A:** [auto-pick] Verify `git rev-parse --show-toplevel` equals `<root>`; a mismatch and a non-repository are each exit 2 with quarry's own sentence, via `errors.Is` sentinels. **Why:** git emits paths relative to the top-level while quarry consumes them as root-relative, so every path in the answer would be silently wrong; and a `--root` outside a repository would report a usage mistake as exit 3 with git's raw message.
- **Q:** [review r5, BLOCKING] `git ls-files --cached` lists index entries whether or not the file is on disk, so an unstaged deletion is handed to `ClauseMapForFiles`, which reads from disk. **A:** [auto-pick] `ClauseMapForFiles` skips a base name it cannot read or decode and records no clause, matching `dirPackage`; its `error` return is reserved for a failure of the call itself. **Why:** an unstaged deletion is the routine case this verb exists to report and must never fail the whole `DeltaGit` call.
- **Q:** [review r5, NIT] `Strategy.Symbols` leaves `Symbol.File` empty — who fills it? **A:** [auto-pick] The delta core, from each entry's own `path`, per side. **Why:** `changed:["file"]`, the `before` block and the `renamed` pair all read it, and all three would silently compare empty strings otherwise.
- **Q:** [review r5, NIT] The real-history pin's created/deleted sets read as exact-set assertions but are partial (`SelfGlyphError` the type is missing; two deletes are out of scope). **A:** [auto-pick] Complete the in-scope lists, mark the two out-of-scope deletes as such, and state the assertions are presence-only. **Why:** an exact-set reading would make the test fail on symbols the pin never claimed to enumerate.
- **Q:** [review r5, NIT] The "facade adds no behaviour of its own" claim sits in `quarry/repo.go` as well as `quarry/doc.go`. **A:** [auto-pick] Amend both. **Why:** amending only `doc.go` leaves the other doc comment false once `DeltaGit` lands.
- **Q:** [review r6, BLOCKING] The stated ordering was not total: `created`/`deleted`/`after` sort file-then-`Start` with `sort.SliceStable`, but the pre-sort order is a map range, and `const a, b = 1, 2` gives every name the same `Start`. **A:** [auto-pick] Append `id` then `kind` as a tie-break (unique, since the table key is), and fix the `changed` array to the closed set's declaration order. **Why:** without it the goldens are still byte-unstable in exactly the case the discussion itself uses as an example elsewhere.
- **Q:** [review r6, BLOCKING] The status-letter table requires the git layer to emit `unmerged path` and unknown-letter errors, but `DeltaEntry` had no field to carry a refusal. **A:** [auto-pick] Add a `Refusal string` field; a non-empty value makes the core skip the entry entirely and emit `disposition: error` with that message. The git layer may pre-set it for an unmerged path, an unrecognised letter, and a read failure on either side. **Why:** a disk-read failure during batch assembly is neither a failed git command (so exit 3 does not cover it) nor grounds to fail the batch — it is one entry's problem, and this is how one entry reports it.
- **Q:** [review r6, BLOCKING] How are `git rev-parse --show-toplevel` and `<root>` compared, given `ResolveRoot` never calls `EvalSymlinks` while git prints the physical path? **A:** [auto-pick] `EvalSymlinks` both sides, then `Clean`. **Why:** a raw comparison would reject a valid repository reached through a symlink — including `t.TempDir()` on any platform with a symlinked temp directory, i.e. the project's own fixtures.
- **Q:** [review r6, NIT] The git calls assume unquoted, newline-delimited paths. **A:** [auto-pick] `-z` on every path-emitting call, split on NUL. **Why:** `core.quotePath` C-quotes non-ASCII paths by default, and a mangled path fails to open silently.
- **Q:** [review r6, NIT] `GitDeltaAnswer` had no stated package, and the renderers' parameter type was unstated. **A:** [auto-pick] Declared in package `quarry` (the one new facade type that is not an engine alias, since a revision-bearing type in the engine would contradict its no-git rule); both renderers take the wrapped form. **Why:** neither package was an unambiguous home under the rules as written.
- **Q:** [review r6, NIT] Four passages justified requirements by "the task's 'Done when'", which is in neither `status.md` nor `docs/roadmap.md`. **A:** [auto-pick] Cite the wiki task body explicitly once, state where it is *not*, and restate the requirements as this discussion's own. **Why:** the discussion must stand alone — a plan writer should need no wiki access to know what "done" means.
