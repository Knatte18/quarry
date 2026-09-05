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
- A new unit-derivation helper in `internal/engine`, exported, that computes each side's glyph unit
  per changed directory from a supplied `base name -> package clause` map, reusing the engine's
  existing clause-vote rule rather than a second copy of it.
- A new package `internal/gitsrc`: a thin, read-only git plumbing layer (`git diff --name-status
  --no-renames`, `git show`, `git ls-tree`) that turns `--from R1 [--to R2]` into the core's entry
  batch, with the working tree as the after side when `--to` is absent.
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
  - `signature` — the verbatim `Signature` text differs.
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

- Decision: a symbol's **body token stream** is the sequence of `(node kind, node text)` pairs of
  every *named, leaf* tree-sitter node under the declaration's body-bearing child, in source order,
  taken from the same parse the extractor already performs. Whitespace, comments inside the body,
  and line numbers contribute nothing. A symbol with no body (a Go type alias, an interface method,
  a `var` with no initialiser) has the empty stream.

  Two symbols are **identical modulo the renamed identifier** when their body token streams have the
  same length and, at every position, either the pairs are equal, or both nodes are `identifier`
  nodes whose texts are respectively the deleted symbol's `Name` and the created symbol's `Name`.
  This is an exact structural test with no threshold and no tuning knob.
- Rationale: comparing token streams rather than bytes is what makes the comparison insensitive to
  reformatting and to line movement while staying exact. Restricting the substitution rule to
  `identifier` nodes carrying exactly the two names in question is what keeps the exact tier
  *exact*: it admits the recursive self-call a renamed function makes and the receiver-less use of
  its own name, and admits nothing else. The whole tier is defined so a test can prove it and a
  reader can predict it, which is the property "AST-exact, no threshold, quarry asserts it"
  demands.
- Rejected: comparing the raw byte span (reformatting would break it); comparing a hash of the
  normalized source text (same problem, plus it cannot express "modulo the renamed identifier");
  full tree-edit distance (a threshold in disguise, and quadratic).

### Exact-tier rename scope: unit-wide, and unique or nothing

- Decision: an exact-tier rename pairs a deleted symbol `D` with a created symbol `C` when **all**
  of the following hold:
  1. Same glyph unit (`D.Glyph.Unit == C.Glyph.Unit`), on the before side for `D` and the after
     side for `C`. Cross-unit never pairs.
  2. Same owner chain (`sameOwner`) and same `Kind`.
  3. Different `Name` (identical names are the `modified`/`file`-move case, not a rename).
  4. Body token streams identical modulo the renamed identifier, per the previous decision.
  5. Signature text identical modulo the renamed identifier, under the same substitution rule.
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
  - `signature_identical_modulo_name` (bool)
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
    changed paths between two revisions, read a blob at a revision, list a directory's entries at a
    revision. It shells out with `os/exec` and returns bytes and errors; it holds no quarry types.
  - `quarry` facade — two methods. `(*Repo).Delta(entries)` delegates to the engine unchanged, as
    every other facade method does. `(*Repo).DeltaGit(from, to, target string) (DeltaAnswer, error)`
    is the convenience: it uses `internal/gitsrc` to build the entry batch (including the
    per-directory clause maps the unit helper needs on each side), then calls `Delta`.
  The CLI calls `DeltaGit`.
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
  - `git diff --name-status --no-renames <from> <to> -- <pathspec>` for the changed-path list when
    `--to` is a revision; `git diff --name-status --no-renames <from> -- <pathspec>` when the after
    side is the working tree.
  - `git show <rev>:<path>` to read a blob at a revision. The after side reads from disk instead
    when `--to` is absent.
  - `git ls-tree --name-only <rev> <dir>/` to enumerate a changed directory's files at a revision,
    so the package-clause vote for that side can be taken over the whole directory; combined with
    `git show` for each `.go` file's clause. The working-tree side reuses the engine's existing
    on-disk `dirPackage` path instead.
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

### CLI shape, and the exit-code contract

- Decision:
  - `quarry delta <target> --from <rev> [--to <rev>] [--text] [--root <path>]`.
  - Exactly one target, as every other verb requires, and `.` means the whole repository. The
    target is used as the git pathspec and as the scope filter.
  - `--from` is required for `delta` and is a usage error (exit 2) when absent. `--to` is optional;
    absent means the working tree.
  - `--from` and `--to` are valid for `delta` only, rejected for the other three verbs with the same
    "`<flag>` is not valid for `<verb>`" message shape `--depth` already uses. `--depth`,
    `--symbols` and `--no-symbols` are rejected for `delta` the same way.
  - Exit codes: **0** whenever the delta was computed — including an empty delta, and including a
    batch in which some entries have `disposition: error`. **2** for a usage error. **3** for an
    internal failure (a git command that failed, a render failure, a stdout write failure). **1 is
    unreachable for this verb**, and the fact is stated in `Run`'s doc comment: this query has no
    negative answer, because "nothing changed" is a true answer to "what changed", not a negative
    one.
- Rationale: keeping the one-target rule intact avoids touching `parseArgs`' target-count check for
  one verb's sake, and `.` already means the repository root everywhere else in the CLI. Declaring
  exit 1 unreachable rather than inventing a meaning for it keeps the four-code contract honest —
  the alternatives (exit 1 for an empty delta, exit 1 when an entry errored) would each make a
  successful, complete answer look like a failure to a shell gate.
- Rejected: making the target optional for `delta` only (a per-verb exception in the argument
  parser, for no gain over `.`); exit 1 on an empty delta; exit 1 when any entry errored.

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
contributes no symbols at all — the delta core must honour that same rule, or a file at the
repository root would produce symbols in the delta that `toc` would never emit. The new helper
should reuse `dirPackage`'s vote logic by extracting it into a form that takes a
`map[base]clause` rather than reading the directory, so both the on-disk caller and the git caller
share one implementation.

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
- Additive only: no existing envelope, verb, flag, exit code, or golden changes. The committed
  goldens under `docs/research/output-formats/after/` and `internal/mcpserver/testdata/golden/`
  must stay byte-identical.
- Quarry reads. Never writes, never touches the working tree or the index. Every git call is read
  plumbing.
- No MCP tool for this query.
- No cache, no index, no daemon, no parser pool (`docs/rewrite-plan.md` §10).
- No fuzzy matching, no threshold, no score cut-off anywhere in the query.
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
error, not a panic.

**`quarry` facade and renderers.** Key-order and byte-contract tests for `RenderDeltaJSON`
mirroring the existing `TestRenderExpandJSON_KeyOrder` shape; a text-view test mirroring
`TestRenderExpandText`. A test asserting `DeltaGit` on a fixture repository agrees with
`Delta` called on the same entries assembled by hand — the two paths must not be able to disagree.

**Goldens.** Committed golden files (JSON and text) under `internal/engine/testdata/delta/` or
`internal/cli/testdata/`, following the existing `compareGolden`/`compareAfterGolden` pattern, for
the seven cases the task's "Done when" enumerates: created, deleted, modified, exact-tier rename,
evidence-tier rename, a mixed batch, and a per-entry extraction failure inside an otherwise good
batch. Goldens are produced under the existing `-update` flag convention and are never
hand-written.

**CLI.** Table tests over `Run` for: `--from` absent → exit 2 with the usage text; `--from` or
`--to` given to `toc`/`resolve`/`expand` → exit 2 naming the flag and the verb; `--depth` given to
`delta` → exit 2; zero or two targets → exit 2; a well-formed call producing an empty delta → exit
0; a batch containing an errored entry → exit 0 with the error visible in the payload; a failing
git invocation → exit 3 with the `internal error: ` prefix; `--text` producing the text view. A
`codeForDelta` table test mirroring `TestCodeForTOCError`.

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
