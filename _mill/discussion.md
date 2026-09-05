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
- Three new exported entry points in `internal/engine`, all extracted from the existing
  `dirPackage`/`unitFor` implementations rather than copied: `PackageClause` (bytes → package
  clause), `(*Repo).ClauseMapForDir` (on-disk clause map for one directory), and
  `UnitsForClauseMap` (the clause vote plus the unit derivation). `dirPackage` is refactored to call
  the first and third so there is exactly one implementation of each rule.
- A new package `internal/gitsrc`: a thin, read-only git plumbing layer (`git rev-parse --verify`,
  `git diff --name-status --no-renames`, `git ls-files --others --exclude-standard`, `git show`,
  `git ls-tree`) returning paths, bytes and errors only, with the working tree as the after side
  when `--to` is absent and untracked files included on that path.
- Two facade methods on `quarry.Repo`: `Delta(entries)` (pure) and `DeltaGit(from, to, target)`
  (convenience, delegating to `internal/gitsrc` then `Delta`).
- A new CLI verb `delta`, with `--from`, `--to`, the existing `--text` and `--root`, exactly one
  target (`.` meaning the whole repository), the standard JSON envelope and the standard exit-code
  contract.
- New JSON and text renderers in `quarry/` (`RenderDeltaJSON`, `RenderDeltaText`), sharing the
  existing `renderJSON` encoder configuration.
- Goldens (JSON and text) for created, deleted, modified, exact-tier rename, evidence-tier rename,
  a mixed batch, and a per-entry extraction failure inside an otherwise good batch.
- A T3-style real-history test pinned to `d413ceb..49304ca` in this repository.
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
  working name `UnitsForDir(dirRel string, clauses map[string]string) (dirPkg string, unitOf
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
    about the batch.
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

### What `modified` means, and the `changed` field

- Decision: a symbol present on both sides under the same key is `modified` when **any** of the
  following differ, and the entry carries a `changed` array naming which of them did, drawn from
  the closed set `["body", "signature", "doc", "file"]`:
  - `body` — the body token stream differs (see the next decision for what a token stream is).
  - `signature` — the verbatim `Signature` text differs. This is `SignatureCut`'s own span, which is
    byte-for-byte the span the signature token stream is built over (see the next decision), so the
    text comparison here and the token comparison used for renames can never disagree about which
    bytes are "the signature".
  - `doc` — the `Doc` text differs.
  - `file` — the symbol's repository-relative file differs between the sides.

  A symbol whose only difference is its line numbers (`Start`/`SigEnd`/`End` shifted because
  something above it grew or shrank) is **not** modified and appears nowhere in the delta. A
  `modified` entry carries the after-side `Symbol` in full plus a `before` block holding the
  before-side `file`, `start`, `sigend` and `end`.
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

- Decision: the two streams are defined by **byte range**, split at exactly the byte
  `internal/engine/nodes.go`'s `SignatureCut` already cuts at — never by node exclusion.

  Let `bodyStart` be `body.StartByte()` when the declaration has a body-bearing child, and
  `decl.EndByte()` when it has none. `body` is the same node `SignatureCut(decl, body, src)` and
  `SigEnd(decl, body)` already receive, resolved per language exactly as the strategy already
  resolves it — for Go, `decl.ChildByFieldName("body")` for functions and methods, `goTypeBody(spec)`
  for types. Then:
  - the **signature token stream** is every leaf node of `decl` whose start byte lies in
    `[decl.StartByte(), bodyStart)`;
  - the **body token stream** is every leaf node of `decl` whose start byte lies in
    `[bodyStart, decl.EndByte())`.

  Both are sequences of `(node kind, node text)` pairs in source order, taken from the same parse
  the extractor already performs. **Anonymous leaf nodes are included** — operators, keywords and
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
  The two symbols may live in different files. If condition 6 fails — two deleted symbols both pair
  exactly with one created symbol, or vice versa — none of the involved symbols is asserted as
  renamed; they all fall through to the evidence tier instead.
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

- Decision: the symbol table is keyed by `(Symbol.ID, Symbol.Kind)`. When a key holds more than one
  symbol on a side — Go's several `func init()` in one package all carry the id `<unit>#init`, per
  `internal/engine/golang.go`'s own doc comment — the two sides are compared as **multisets of body
  token stream hashes**: equal multisets means unchanged; any difference reports **one** `modified`
  entry for that key, with `changed` naming the dimensions and a `count_before`/`count_after` pair
  when the multiplicities differ. A multi-occurrence key is never a rename candidate on either
  tier.
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
  empty-but-non-nil slice means an existing empty file. The answer carries a `files` array with one
  entry per input entry, in the input's own order, each holding `path` and a closed `disposition`
  word:
  - `added`, `removed`, `changed` — the file was extracted on the sides where it exists.
  - `unsupported` — the file's extension resolves to no registered strategy. It contributes no
    symbols on either side and is not an error.
  - `error` — extraction failed for this entry; the entry additionally carries `error` with the
    message. It contributes no symbols to any delta list.
  A failing entry never fails the batch: the whole `Delta` call still returns a nil error, and the
  answer is still rendered as a success envelope.
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
  - `(*engine.Repo).ClauseMapForDir(dirRel string) (map[string]string, error)` — the on-disk clause
    map for one directory, ignore-filtered exactly as `dirPackage` already is. Used for the
    working-tree side.
  - `engine.UnitsForClauseMap(dirRel string, clauses map[string]string) (dirPkg string, unitOf
    func(base string) string)` — the vote and the derivation, extracted from `dirPackage`'s
    tie-break and `unitFor` so both callers share one implementation rather than a reimplementation.

  Which layer calls which: for a **revision** side, `DeltaGit` asks `internal/gitsrc` for the
  directory's file names (`git ls-tree`) and each `.go` file's bytes (`git show`), calls
  `engine.PackageClause` on those bytes to build the map, then `engine.UnitsForClauseMap`. For the
  **working-tree** side it calls `(*engine.Repo).ClauseMapForDir` and then the same
  `UnitsForClauseMap`. `internal/gitsrc` never sees a clause, a unit, or a tree-sitter node.
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
  - `git rev-parse --verify <rev>^{commit}` for each supplied revision, **first**, so an
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
    `engine.PackageClause`. The working-tree side calls `(*engine.Repo).ClauseMapForDir` instead.
  Nothing else. No `checkout`, no `stash`, no index write, no config write, no `-M`/`-C`.
- Rationale: `--no-renames` is not an optimisation, it is a correctness requirement. Git's rename
  detection is a similarity threshold; letting it run would mean quarry's answer silently inherited
  a heuristic the whole two-tier design exists to replace. With `--no-renames` a rename arrives as a
  delete plus an add and is classified by the table comparison, which is the only classifier this
  query is allowed to have. The clause-vote calls are the price of the unit decision above; they
  are per changed *directory*, not per changed file, and they read package clauses only.
- Rejected: `git diff -M` and trusting git's rename pairs (a threshold heuristic in the one place
  the contract forbids one); `git diff --name-only` alone (loses the add/delete status, which the
  entry disposition needs).

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
    the value spelled exactly as given — and the usage text on stderr. `internal/gitsrc` runs
    `git rev-parse --verify <rev>^{commit}` for each supplied revision before anything else and
    returns a distinguishable sentinel (`gitsrc.ErrUnknownRevision`) that the CLI checks with
    `errors.Is`, never by parsing git's message.
  - Exit codes: **0** whenever the delta was computed — including an empty delta, and including a
    batch in which some entries have `disposition: error`. **2** for a usage error, which now
    includes an unresolvable revision. **3** for an internal failure (a git command that failed for
    any *other* reason, a render failure, a stdout write failure). **1** is reachable only through
    `RepoRelTarget`'s target-escapes-the-root rejection, exactly as it is for `toc`; the query
    itself has no negative answer, because "nothing changed" is a true answer to "what changed", not
    a negative one, and `Run`'s doc comment says so.
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
  given, or `null` for the working tree). The CLI renders the wrapped form.
- Rationale: the core knows nothing about git, and a field it can never populate would be a lie in
  its own type. Wrapping puts the revision echo exactly where the revisions are known.
- Rejected: an optional `from`/`to` on the core answer left empty by the pure path (an
  always-omitted key on the one call path most consumers use).

### `--text` renders a lossless text view

- Decision: `quarry delta --text` emits a lossless text view of the same data, via a new
  `quarry.RenderDeltaText`. It follows the conventions the three existing text renderers already
  establish in `quarry/text.go`.
- Rationale: `--text` is valid for every verb; a verb that ignored it would be the only exception.
  `docs/rewrite-plan.md` §4 fixes the text view as lossless, and the task's own "Done when" asks for
  goldens in both JSON and text views.
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
the exported `(*engine.Repo).ClauseMapForDir`. Export is not decoration here — package `quarry`
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
adds `self.go`. It contains, verifiably:
- **created**: `glyph#Self`, `glyph#Glyph.IsSelf`, `internal/engine#SelfGlyphError.Error`,
  `internal/engine#Repo.resolveSelfTarget`.
- **deleted**: `internal/engine#Repo.resolvePathTarget`, `internal/repopath#RepoRelPath`,
  `internal/cli#isGlyphTarget` (the latter two outside the two pinned directories).
- **modified**: several, across both units.
- **an evidence-tier rename**: `Repo.resolvePathTarget` → `Repo.resolveSelfTarget` — same unit
  (`internal/engine`), same owner (`Repo`), same kind (method), different name, and a changed body
  and signature (`target string` → `unit string`), which is exactly the demotion the evidence tier
  exists for.
No exact-tier rename occurs naturally in this history; the exact tier is asserted by a synthetic
unit test instead, which is what the task's "Done when" asks for.

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
- an interface's signature stream asserted byte-identical to the symbol's own `Signature` string,
  proving the two definitions cut at the same place.
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
  diff is attempted.

**`quarry` facade and renderers.** Key-order and byte-contract tests for `RenderDeltaJSON`
mirroring the existing `TestRenderExpandJSON_KeyOrder` shape; a text-view test mirroring
`TestRenderExpandText`. A test asserting `DeltaGit` on a fixture repository agrees with
`Delta` called on the same entries assembled by hand — the two paths must not be able to disagree.

**Goldens.** Committed golden files (JSON and text) under `internal/engine/testdata/delta/` or
`internal/cli/testdata/`, following the existing `compareGolden`/`compareAfterGolden` pattern
(wherever those helpers' own fixtures live at merge time — see Constraints on the in-flight
goldens move), for
the seven cases the task's "Done when" enumerates: created, deleted, modified, exact-tier rename,
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
`--text` producing the text view. A `codeForDelta` table test mirroring `TestCodeForTOCError`.
One test must pin the **target-resolution** rule: `delta .` run from a subdirectory scopes to that
subdirectory, and produces the same scope `toc .` does from the same directory — the two verbs
resolve one argument the same way or the CLI has two meanings for it.

**Real-history check (T3-style).** One test running `DeltaGit("d413ceb", "49304ca", ...)` against
this repository, scoped separately to `glyph/` and to `internal/engine/`, asserting a hand-verified
expectation: the created and deleted sets listed under "Real-history pin" above, at least one
`modified` entry per unit, and the `resolvePathTarget` → `resolveSelfTarget` pair present in
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
- **Q:** What are `--from`/`--to` semantics, and does `delta` keep the one-target rule? **A:** [auto-pick] `--from` required, `--to` optional (absent = working tree); exactly one target as every verb, with `.` meaning the whole repository. **Why:** keeps `parseArgs`' target-count rule intact, and `.` already means the root everywhere else in the CLI.
- **Q:** Which git commands, and is git's own rename detection used? **A:** [auto-pick] `git diff --name-status --no-renames`, `git show`, `git ls-tree`; git's `-M` is never used. **Why:** `--no-renames` is a correctness requirement, not an optimisation — git's rename detection is a similarity threshold, exactly what the two-tier design replaces.
- **Q:** Does `delta` support `--text`? **A:** [auto-pick] Yes, a lossless text view. **Why:** `--text` is valid for every verb, and the task's "Done when" asks for goldens in both views.
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
- **Q:** [review r2, BLOCKING] `internal/gitsrc` was specified as holding no quarry types and no tree-sitter, yet also as building the per-directory clause maps — which only `Strategy.Package` inside `treesitter.WithTree` can produce. **A:** [auto-pick] Name three exported engine entry points — `PackageClause`, `(*Repo).ClauseMapForDir`, `UnitsForClauseMap` — and have the facade call them; `gitsrc` returns paths, bytes and errors only. **Why:** `dirPackage` and `unitFor` are unexported, so package `quarry` cannot reach them; export is the only seam across the boundary, and the refactor keeps one implementation of each rule.
- **Q:** [review r2, NIT] Is the git-sourced batch gitignore-filtered, given `toc` never lists an ignored file? **A:** [auto-pick] Untracked files are filtered (`--exclude-standard`); tracked-but-gitignored files are kept, as a documented divergence from `toc`'s listing rule. **Why:** the `.gitignore` chain is itself version-dependent, so filtering would need a revision to be taken against and would make the two sides' enumerations — and their clause votes — disagree with each other; mutual consistency between the sides is the property the comparison depends on, agreement with `toc` is not.
- **Q:** [review r2, NIT] The rationale claimed "`.` already means the repository root everywhere else in the CLI" — is that true? **A:** [auto-pick] No, it was wrong. `runTOC` resolves through `RepoRelTarget(root, base, target)` with `base = cwd` unless `--root`, so `toc .` from a subdirectory names that subdirectory. `delta`'s target goes through the same call, and `.` means the current directory. **Why:** git would resolve a raw pathspec against the root while quarry resolves against the cwd — one argument with two meanings.
- **Q:** [review r2, NIT] "No fuzzy matching anywhere in the query" contradicted emitting a Jaccard float. **A:** [auto-pick] Narrow the constraint to "nothing quarry decides is fuzzy" and cite `docs/rewrite-plan.md` §9. **Why:** §9's non-goal governs how quarry *answers*; the evidence tier is the `ambiguous` branch of that same rule, and no asserted outcome reads the similarity value.
- **Q:** [review r2, NIT] `--from bogus` would fail inside git and land on exit 3 with git's raw message. **A:** [auto-pick] `git rev-parse --verify` each revision first; an unresolvable one is exit 2 with `unknown revision: <rev>`, matched via an `errors.Is` sentinel. **Why:** `exitUsage` already means "the caller asked wrong … a `--root` that does not resolve"; a revision that does not resolve is the same class, and it is this verb's most likely user error.
