# Discussion: Engine core (T3)

```yaml
task: Engine core (T3)
slug: engine-core
status: discussing
parent: main
```

## Problem

The extractor kept under `internal/quarryengine/` after the T0 deletion still speaks V1: its
`Symbol` has a bare `Name` and a bare `Owner` string and no glyph at all, its `toc` answers are two
unrelated shapes (`FileTOC` and a one-level `DirTOC` whose `Dirs` is a list of bare names), it lists
only `.go` files, it lists only functions/methods/types, it knows nothing about package
documentation, nothing about Go's external test package, and it carries a lossy "compact" view that
a measured ladder run (`2026-09-02-compact2`) showed costs precision.

All three queries of the rewrite — `toc`, `resolve`, `expand` — stand on a symbol model that knows
its glyph, its owner chain and its head span, and on one recursive directory answer. The `glyph`
package (T1) is merged on `main` and is the only implementation of the grammar. T3 is the step that
moves the engine onto that model and produces §4's answer shape exactly, because T4 (`resolve` +
`expand`) and T5a (facade + CLI) both land on top of it and T5a's envelope is fixed by §4, not by
T4 — T3's done-when is what pins it byte for byte.

**Why now:** wave 1 is merged (`dace812` "The glyph package (T1)"). T3 is alone in wave 2 and is on
the critical path to the measurement: T0 → T1 → **T3** → T5a → T6 → T7.

## Scope

**In:**

- Settle the engine package layout: collapse `internal/quarryengine/` and `internal/quarryengine/toc/`
  into one package `internal/engine`, files per concern, and move the cgo guard so it still fires.
- `Symbol` gains its glyph (`glyph.Glyph`, the single source of truth for unit/owner/name), its head
  span, and a `File` field; `Docstring` is renamed `Doc`; the `Kind` vocabulary grows.
- The Go unit walk: directory → unit; the external `_test` package as a second unit; several `init`
  in one package all carrying the one glyph `<unit>#init`.
- Package-doc extraction (Go's `// Package x ...` block above the package clause), first paragraph.
- The recursive directory answer of plan §4 — `dir`, `package`, `language`, `doc`, `files`, `dirs` —
  with the `depth` and `symbols` knobs.
- Every non-gitignored file listed, not only `.go`: non-code files get `name` and a `header` read by
  a per-format text rule, never `symbols`.
- `toc` re-keyed by glyph: the engine's symbol entry emits `id`, not `name`/`owner`.
- Widen the Go declaration walk to every declaration glyph.md gives a glyph: package-level `func`,
  `type`, `const`, `var`, methods, and interface methods.
- An internal span lookup (`SpansOf`) sufficient for the round-trip done-criterion, living in the
  `resolve.go` file T4 will grow.
- Delete the lossy compact view and the doc-sentence trimming option.
- Fix `README.md`'s stale verb list — it still names the three queries "`map`, `resolve`, `members`",
  and all of `map` and `members` are dead words (`toc`, `resolve`, `expand`).

**Out:**

- `resolve` as a public verb with its status vocabulary (`found`/`not_found`/`ambiguous`/`multipart`,
  `unit: found|not_found`, paths as targets) — **T4**.
- `expand` — **T4**. T3 only supplies the head span it will read.
- The `quarry/` facade, the CLI, the one envelope's `ok`/`status` wrapper, exit codes, and the
  lossless text view — **T5a**. T3 emits the §4 *payload* objects; the envelope wraps them later.
- MCP — **T6**.
- Any language but Go. The extension table stays Go-only for *languages*; other extensions get
  header rules only.
- Any cache, index, daemon or parallelism. §5's measured numbers say nothing on quarry's side is
  worth optimising yet.
- `impact`, `assert-no-callers`, `verified`, a type checker — phase 2.
- Changes to `bench/`, `docs/`, or the `glyph` package. The engine never re-implements the grammar.

## Decisions

### D1 — Package layout: one package, `internal/engine`

- Decision: collapse `internal/quarryengine` and `internal/quarryengine/toc` into **`internal/engine`**,
  one package, files per concern. `treesitter` stays a subpackage at `internal/engine/treesitter`
  (it is a seam, not a verb). Planned files:
  `doc.go` (package doc), `answer.go` (`DirAnswer`, `FileEntry`, `Symbol`, `Kind`),
  `repo.go` (`Repo`, `Open`, path resolution), `ignore.go` (gitignore matching),
  `walk.go` (directory walk, unit identity, per-directory package/language), `toc.go` (the `TOC`
  entry point and the depth/symbols knobs), `resolve.go` (the internal span lookup),
  `strategy.go` (the per-language contract and registry), `golang.go` (the Go strategy),
  `nodes.go` (tree-sitter node helpers), `text.go` (`StripComment`, `FirstParagraph`),
  `headers.go` (non-code header rules), `extension.go` (extension tables),
  `classify.go` (directive / generated-banner / test-name rules), `errors.go`.
- Rationale: plan §12 T3 says "one package, files per concern, never a package per verb", and names
  settling the layout as T3's job precisely because T4 and T5a land in it next. `quarryengine`
  stutters against the module path (`github.com/Knatte18/quarry/internal/quarryengine`) and was V1's
  name; nothing outside `internal/` imports it today (there is no `cmd/`, no facade), so the rename
  costs one pass of import rewriting and never again.
- Rejected: keeping the `toc/` subpackage (a package per verb, exactly what the plan forbids);
  keeping the `quarryengine` name (stutter, and the one free moment to change it is now).

### D2 — The cgo guard moves to `internal/cgoguard`, imported by `treesitter`

- Decision: move `cgoguard.go` / `cgoguard_nocgo.go` verbatim (adjusting the package clause and the
  doc comments) into a new package `internal/cgoguard` that imports nothing, and have
  `internal/engine/treesitter` blank-import it (`_ "github.com/Knatte18/quarry/internal/cgoguard"`).
- Rationale: the guard's whole purpose is to fail a `CGO_ENABLED=0` build with a readable message
  *before* the raw cgo linker error. It works today only because `internal/quarryengine` imports no
  cgo. Under D1 the engine package itself transitively imports cgo, so a guard left in it would be
  unreachable exactly when it is needed. Making the guard a *dependency of* `treesitter` puts it
  strictly earlier in the build graph than anything that links tree-sitter, so its error always
  comes first.
- Rejected: leaving the guard in `internal/engine` (unreachable, per above); leaving a vestigial
  `internal/quarryengine` package holding only the guard (a package that exists for one build tag,
  named after the thing it is no longer); deleting the guard (the linker dump it prevents is far
  less actionable — see the file's own comment).

### D3 — `Symbol` carries the glyph, and the glyph is the only name

- Decision:

  ```go
  type Symbol struct {
      Glyph     glyph.Glyph `json:"-"`
      ID        string      `json:"id"`        // Glyph.String(), set at build time
      Kind      Kind        `json:"kind"`
      File      string      `json:"file,omitempty"`
      Start     int         `json:"start"`
      SigEnd    int         `json:"sigend,omitempty"`
      End       int         `json:"end"`
      Signature string      `json:"signature"`
      Doc       string      `json:"doc,omitempty"`
      HeadStart int         `json:"-"`
      HeadEnd   int         `json:"-"`
  }
  ```

  The separate `Name` and `Owner` fields are **removed**; a caller reads `sym.Glyph.Name` and
  `sym.Glyph.Owner`. `Docstring` is renamed `Doc` to match §4's key.
- Rationale: §4 fixes the symbol entry's key set as `id`, `kind`, `start`, `sigend`, `end`,
  `signature`, `doc`, and `file` "wherever entries can span files". Two parallel spellings of the
  same identity (a `Name` string and a `Glyph.Name`) is exactly the drift the "one implementation of
  the glyph grammar" rule exists to prevent. `ID` is stored rather than computed per marshal so the
  emitted key set is a plain struct with plain tags and needs no custom `MarshalJSON`.
- `File` is empty (and therefore omitted) in a `toc` answer, where the symbol already sits inside its
  file entry; it is filled by `SpansOf` (D16, which returns `Symbol` values precisely so this field
  has a writer in T3) and by T4's `resolve`/`expand`, whose entries do span files.
- Rejected: a custom `MarshalJSON` computing `id` (hides the contract); keeping `Name`/`Owner`
  alongside the glyph (two sources of truth); making `Glyph` unexported (T4 and the facade need it
  typed, not re-parsed from a string).

### D4 — The head span is explicit, and for Go equals the type declaration's own span

- Decision: `Symbol.HeadStart` / `HeadEnd`, JSON-hidden, populated only for `Kind == KindType`, where
  in Go they equal the type declaration's own `Start`/`End`. Zero for every other kind.
- Rationale: plan §9 step 2 and §12 T3 both name the head span as something `Symbol` gains in this
  task, and §5 defines it: "the type's *head* (its own lines … in Go the `type` block)". Go's head
  falls out trivially because a Go type never contains its methods; Python's and C#'s will not
  (class span minus member spans), so the field is where the language difference lands. Making it
  explicit now means `expand` (T4) reads a field instead of re-deriving a rule.
- Rejected: letting T4 derive it from `Start`/`End` (works for Go, silently wrong for the first
  language that needs it, and contradicts the task text); emitting it in JSON (§4's key set does not
  carry it, and the byte-for-byte examples would break).

### D5 — Kind vocabulary widens; every glyphed Go declaration is listed

- Decision: `Kind` becomes the closed set `function`, `method`, `type`, `const`, `var`. The Go walk
  gains: `const_declaration` and `var_declaration` at file scope (one symbol per declared *name*,
  covering both the grouped `const ( … )` and the multi-name `var x, y int` shapes), and interface
  methods (`method_elem` inside an `interface_type`, `Kind == method`, owner = the interface's type
  name).
- Rationale: glyph.md §3's table says Go glyphs cover "package-level `func`, `type`, `const`, `var`;
  methods; interface methods". A `toc` that lists fewer than the glyph contract names leaves those
  declarations unaddressable, and the round-trip criterion would pass vacuously over them.
- **Span and signature per shape**, stated as explicitly as the type case already is. In every row
  below, "the node's span" means `End` is the node's own last line and **`Start` is the first line of
  the comment block `CommentBlockAbove` attaches, when it returns one, and the node's own first line
  otherwise** — the rule `goDeclSymbol` already implements and the one §4's `placement` demonstrates
  (`"start": 16` with the `type` clause on line 20). Stating it per row matters because the round
  trip cannot catch a wrong reading: it compares `toc`'s output against a lookup built from the same
  code.

  | shape | Start / End | Signature | SigEnd |
  |---|---|---|---|
  | ungrouped `const X = 1`, `var x int` | the **declaration's** span, doc via `CommentBlockAbove(decl)` — the same reason the ungrouped type uses `decl` and not `spec`: `decl` is the node with `source_file` siblings | `SignatureCut(decl, nil, src)`, i.e. the whole declaration text trimmed, carrying the `const`/`var` keyword | `0` |
  | grouped `const ( … )` / `var ( … )`, one spec | the **spec's** span, doc via `CommentBlockAbove(spec)` | the keyword prepended to the spec's own text — `"const " + SignatureCut(spec, nil, src)`, mirroring the grouped type's `"type " + …` so grouped and ungrouped render identically | `0` |
  | several names in one spec, `var x, y int` | one symbol per name, **all sharing that spec's span, doc and signature text** | as its row above | `0` |
  | an `iota` spec that is a bare name, `B` with no type and no value | the spec's span | the keyword plus the spec text, so `"const B"` — verbatim, never synthesised from the preceding spec | `0` |
  | interface method (`method_elem`) | the `method_elem`'s span, doc via `CommentBlockAbove(method_elem)` | the `method_elem`'s own text | `0` |

  `SigEnd` is `0` for every one of them: none has a body-bearing child, which is exactly the type
  alias case `nodes.go:SigEnd` already documents, and `omitempty` therefore omits the key.
- Several names in one spec produce **distinct glyphs over identical spans**, which the round trip
  handles without a special case: D6 asserts set-equality *per glyph*, and each of `x` and `y` maps
  to that one span.
- Note: a grouped `const`/`var` spec's docstring association reuses the grouped-type rule already in
  `nodes.go` (`CommentBlockAbove` walking a spec's prev-siblings inside the declaration), so this is
  a new node kind on an existing rule, not a new rule. The grouped-versus-ungrouped test is
  `goDeclIsGrouped`'s literal-`(`-child check, for the same reason it is used on types: a
  single-spec group is legal and a spec-count test would misroute it.
- Rejected: keeping the current three kinds (contradicts glyph.md); one symbol per *spec* rather than
  per name (`var x, y int` would then have one glyph for two symbols, and neither `x` nor `y` would
  resolve).

### D6 — `init` is one glyph, many declarations; the round trip is set-equality per glyph

- Decision: every `func init()` in a package gets `id: "<unit>#init"`. `toc` lists each declaration
  separately, with its own span, all carrying that one id. The round-trip check asserts, per glyph,
  that the *set* of `(File, Start, SigEnd, End)` tuples `SpansOf` returns equals the *set* of tuples
  `toc` listed under that glyph.
- Rationale: glyph.md §3 — "`internal/logger#init` is one glyph, and with several `init` functions it
  resolves `multipart`, every one returned in run order (file order, then line)". A one-to-one
  round trip would be unsatisfiable by construction here, and equally so for build-tag duplicates
  (which resolve `ambiguous`, several declarations, one glyph). Set-equality is the honest form of
  "zero misses, zero extras".
- `SpansOf` returns its symbols ordered by file then start line, so the ordering guarantee T4 needs is
  already in place.
- Rejected: strict one-to-one (fails on `init` and on build tags, the two cases §6 explicitly calls
  out); suffixing duplicate glyphs (`init#2`) — glyph.md forbids any spelling not in the alphabet.

### D7 — The unit walk: directory → unit, plus the external `_test` unit

- Decision: per directory,
  1. read every `.go` file's package clause;
  2. the **directory's package** is the most common clause among files whose clause does not end in
     `_test`; if every file's clause ends in `_test`, it is the most common clause overall. **On a
     tie, the lexicographically smallest clause wins** — a directory holding a `//go:build ignore`
     `package main` file beside a one-file package splits evenly, and without a tie-break the
     directory's `package`, every file's `package` deviation key and every glyph unit below it would
     depend on `os.ReadDir`'s order, which is exactly what D18 exists to eliminate;
  3. a file whose clause is exactly `<dirPackage>_test` belongs to the unit `<dirRelPath>_test`;
     every other `.go` file belongs to `<dirRelPath>`;
  4. `_test.go` files declaring the package itself belong to the package's own unit (glyph.md §2).
- The file entry emits `package` only when the file's clause differs from the directory's — which is
  exactly the `render_test` case §4 names.
- The directory answer's `dir` is the repository-relative path with forward slashes; `package` and
  `language` are the directory's, stated once.
- Rationale: this is glyph.md §2's Go unit corner case stated as a walk. Deriving the external-test
  unit from `<dirPackage>_test` rather than from any `_test`-suffixed clause is what keeps a package
  legitimately named `mytest` or `httptest` from being mistaken for one.
- Rejected: treating any `_test`-suffixed package as external (misfires on real package names);
  treating `_test.go` *files* as the discriminator (wrong — §6 and glyph.md both key on the package
  clause, not the filename).

**The repository root has no unit, so its own `.go` files carry no symbols.** A `.go` file directly
in the root would take the unit `""` under the rule above, and `glyph.Parse` rejects it —
`glyph/golang.go:checkGoUnit` returns `ReasonUnitEmpty` for `""` and `ReasonUnitDotSegment` for `"."`.
So: files in the root directory are **listed** like any other file (`name`, `header`, `test`,
`generated`) but their entry carries **no `symbols`**, and `SpansOf` never produces a root span. The
root directory answer's `dir` is `"."`.

- Rationale: T3 must not invent an alphabet element. glyph.md §6 makes the `glyph` package the one
  implementation of the grammar and this discussion's Scope puts that package out of bounds, so the
  only alternative to excluding root symbols would be minting a unit spelling `Parse` rejects —
  which fails Testing 15's parse assertion and leaves `SpansOf` nothing to invert. Emitting nothing
  is the honest answer to a name the contract cannot spell.
- The cost is presently zero and measured: **neither quarry nor Loomyard has a single `.go` file in
  its repository root**, so nothing the done-criterion checks is lost, and the round trip stays exact
  because a file with no listed symbols has nothing to invert.
- **Recorded as a glyph.md gap, not closed here.** glyph.md §2 spells a Go unit as "the path relative
  to the repository root" and never says what the root itself spells. The two candidate answers — a
  reserved literal (as C# reserves `global` for the global namespace) or the module's own path from
  `go.mod` — both change the alphabet, so they belong in a glyph.md amendment against a repository
  that actually needs one, not in T3.
- Rejected: minting `"."` or `""` as the unit (rejected by `Parse` today, so every such glyph would
  be unreadable); adding a `glyph` package change to this task (out of scope, and a single-file task
  changing the shared identifier contract is exactly the coupling §7's ordering avoids).

### D8 — Package doc: the block above the clause that begins `Package <name>`

- Decision: a directory's `doc` is `FirstParagraph` of the comment block immediately preceding the
  `package` clause whose first line begins with `Package <packageName>`. Candidate files are the
  directory's own-unit `.go` files in sorted order, with `doc.go` tried first. No match → `doc` is
  absent, never empty.
- Rationale: verified against the byte-for-byte target. Loomyard `internal/reedengine/render/types.go`
  carries **two** leading blocks — a file header ("types.go defines the closed display vocabulary …")
  and, after a blank line and immediately above `package render`, the package doc ("Package render
  owns the closed display vocabulary …"). §4's expected `doc` for that directory is the first
  paragraph of the second block; the package comment continues after a blank line ("The package is
  deliberately split into two layers …") and §4 does not carry it — matching §4's "carried whole
  (one paragraph)". Requiring the `Package <name>` prefix is what stops `checksum.go`'s adjacent file
  header from being read as the package doc. The existing `Header` rule (first *non-directive*
  block) is a different rule and both are needed; they stay separate methods on `Strategy`.
- The 29 Loomyard packages with no package doc are the expected `absent` case, and are the §8.2
  invariant `toc` can report.
- Rejected: "any comment adjacent to the package clause" (picks up file headers — provably wrong on
  the target directory); concatenating every file's package comment the way `go/doc` does (§4 asks
  for one paragraph, and concatenation has no defined order across files).

### D9 — gitignore: a matcher in the engine, no new dependency

- Decision: implement gitignore matching in `internal/engine/ignore.go`, supporting comments, blank
  lines, `!` negation, leading-`/` anchoring, trailing-`/` directory-only patterns, `**`, `*` and `?`
  within a segment, the "a pattern containing a slash anywhere but its trailing position is anchored
  to its own `.gitignore`'s directory" rule, and the "pattern with no slash matches at any depth"
  rule. (Loomyard's `plugins/prowler/bin/` is the interior-slash case: anchored by its slash,
  directory-only by its trailing one.) Patterns are
  collected from `.gitignore` files from the repository root down to **and including every directory
  the walk enters**, later files and later patterns winning: entering a subdirectory appends that
  directory's own `.gitignore` to the set in force below it, and leaving it drops those patterns
  again. `SpansOf` builds the same set along the root → unit-directory chain, so the two entry points
  filter identically. `.git/` is always excluded.
- This supersedes D22's "once per call, walking root-to-target" phrasing, which would have read only
  the chain down to the *argument* and then descended blind: under `--depth all` a `.gitignore` in
  any descendant would never be read, while `SpansOf` — whose target *is* the unit directory — would
  read it, and a descendant-ignored `.go` file would become a round-trip **miss** under Testing
  14/15. "Never cached" (D22) and "extended as the walk descends" are independent properties, and
  both hold: nothing survives the call.
- Explicitly **not** supported (documented in the file's own comment): `core.excludesFile`,
  `.git/info/exclude`, and per-file `.gitattributes` interaction.
- Rationale: §4 says `toc` lists "every file in the directory that is not gitignored", so the rule is
  part of the answer, not an optimisation. Both `.gitignore` files in play — quarry's own (with the
  `/quarry` + `!/quarry/` negation pair) and Loomyard's (anchored `/lyx`, dir-only `.dev-bin/`,
  `**/`-prefixed mill block) — exercise exactly the subset above and nothing more, so the subset is
  measured against the two repositories the tests run on rather than guessed.
- Rejected: shelling out to `git check-ignore` (makes every answer depend on a git binary and on the
  tree being a repository, which quarry does not otherwise require, and puts a subprocess in the
  read path §5 measures in milliseconds); adding `github.com/sabhiram/go-gitignore` (the engine's
  only dependency today is tree-sitter, and this is ~150 lines with table tests).

### D10 — Non-code files: a per-format header rule, never `symbols`

- Decision: a second extension table, `extensionHeaderRules`, maps an extension to a pure-text header
  extractor. `.md` → first ATX or setext heading plus the first paragraph after it; `.html`/`.htm` →
  a leading `<!-- … -->`; `.css` → a leading `/* … */`; `.js`/`.mjs`/`.ts` → a leading `//` block or
  `/* … */`; `.yaml`/`.yml`/`.toml`/`.sh`/`.bash`/`.zsh` → a leading `#` block, skipping a
  shebang line. Every other extension yields no header, so the entry is `name` alone. None of these
  files ever gets `symbols`, and the `symbols` knob does not change them.
- **Two tables, not one.** `extensionHeaderRules` is keyed by extension; a second, small
  `baseNameHeaderRules` is consulted **only when `filepath.Ext` returns `""`**, and holds
  `Makefile` and `Dockerfile` → the same `#`-block rule. An extensionless file is a real case
  (`Makefile` is the obvious one) and an extension-keyed map cannot express it, so it gets its own
  lookup rather than a sentinel key that reads like an extension and is not one.
- Rationale: §4 states the rule and the reason ("the question `toc` answers is whether a file is
  worth opening"). These are text rules, not grammars: adding a tree-sitter grammar per markup
  format would pull in cgo dependencies §2 just deleted.
- Rejected: listing only files with a language (that is V1's behaviour, and the exact thing §4
  changes); giving non-code files a `language` key (§4: a file without a language gets `name` and a
  `header`).

### D11 — `language` on a file entry, and the per-file alphabet

- Decision: the extension → *language* table stays `.go` only. A file entry emits `language` only
  when the file has a language and it differs from the directory's; a directory with no `package`
  says nothing and its files each carry their own. Implement the rule now; note in the code that it
  is unreachable while Go is the only language.
- Rationale: §4's "The alphabet is chosen per file, never per repository" is a shape decision, and
  the shape is what T5a's envelope is built on. Implementing the rule with one language costs a
  comparison; retrofitting it later means changing the answer type after the envelope is fixed.
- Rejected: deferring the field to the second language (changes §4's key set after T5a pins it).

### D12 — The answer types, and `partial` renamed `lossy`

- Decision:

  ```go
  type DirAnswer struct {
      Dir      string      `json:"dir"`
      Package  string      `json:"package,omitempty"`
      Language string      `json:"language,omitempty"`
      Doc      string      `json:"doc,omitempty"`
      Files    []FileEntry `json:"files,omitempty"`
      Dirs     []DirAnswer `json:"dirs,omitempty"`
  }

  type FileEntry struct {
      Name      string   `json:"name"`
      Header    string   `json:"header,omitempty"`
      Test      bool     `json:"test,omitempty"`
      Generated bool     `json:"generated,omitempty"`
      Package   string   `json:"package,omitempty"`
      Language  string   `json:"language,omitempty"`
      Lossy     bool     `json:"lossy,omitempty"`
      Error     string   `json:"error,omitempty"`
      Symbols   []Symbol `json:"symbols,omitempty"`
  }
  ```

  Every optional key is `omitempty`; `dirs: []` and `test: false` are never emitted. V1's `*bool`
  discipline on `Test`/`Generated` is dropped.
- Rationale: §4's "Shared facts once, defaults never" is explicit that `test: false`,
  `generated: false` and an empty `dirs: []` are the V1 clutter this rewrite removes. The `*bool`
  existed to distinguish "false" from "this language has no rule"; with Go the only language and
  Go's rules always known, the pointer encodes a distinction that cannot arise. A future language
  that genuinely cannot tell reintroduces it then, against a real case.
- `Lossy` is §6's required rename of the `partial` field ("the word `partial` in toc today means
  'lossy parse' — rename that field"), freeing `partial` for C#'s meaning. `Error` and `Lossy` stay
  mutually exclusive by construction, as today. Both are absent on the happy path, so §4's examples
  still reproduce byte for byte.
- Rejected: keeping `Partial` (§6 says rename); keeping the `*bool`s (YAGNI, and it emits
  `"test": false` for a language that has a rule saying false, which §4 forbids either way).

### D13 — The `depth` and `symbols` knobs

- Decision:

  ```go
  const DepthAll = -1

  type TOCOptions struct {
      Depth   int   // 0 = direct children as identity + doc only; N; DepthAll
      Symbols *bool // nil = the per-target default
  }

  func (r *Repo) TOC(target string, opts TOCOptions) (DirAnswer, error)
  ```

  `Depth: 0` fills the target's own `files` and lists its **direct** subdirectories as `DirAnswer`s
  carrying `dir`, `package` and `doc` and nothing else. `Depth: N` fills the files of subdirectories
  down N levels, each level's leaf `dirs` again being identity+doc. `DepthAll` recurses to the
  bottom. `Symbols` nil means `true` when `target` is a file and `false` when it is a directory;
  a non-nil value wins for every file entry at every depth.
- A file target answers as "a directory answer with one file entry" (§4): the enclosing directory's
  `dir`/`package`/`language`/`doc`, `files` holding exactly that one entry, no `dirs`.
- Rationale: this is §4's knob table stated as a type. `Depth: 0` is direct children only because
  that is what the table says ("`0` lists subdirectories by `dir`, `package` and `doc` only");
  §4's prose "`toc internal` — every package under it with its doc" is loose, and the whole-tree
  orientation map is what `--depth all` is for. `Symbols` is a `*bool` here — unlike D12's fields —
  because the default genuinely depends on the target kind and "the caller did not say" is a third
  state the engine must be able to see.
- Rejected: `Depth: 0` meaning all descendants at identity+doc (contradicts the knob table, and
  makes `--depth 1` mean something incoherent); a `Symbols bool` (a zero value would silently mean
  "off" for a file target, exactly the `Options{}` trap the current `DocSentences` field documents).

### D14 — Delete the compact view and doc-sentence trimming

- Decision: delete `compact.go`, `compact_test.go`, `sentences.go`'s `FirstSentences` and its tests,
  and the `Options.DocSentences` field and its `applyDocSentences` helper. Keep `FirstParagraph`
  (used by the header and package-doc rules); it moves to `text.go` alongside `StripComment`.
- Rationale: §10 makes compact-by-default a non-goal and §4 requires `doc` "complete, never truncated
  by extraction". The ladder measured the cost directly: `2026-09-02-compact2` cut bytes 4× and
  precision 0.96 → 0.82. The *lossless* text view §4 describes for the MCP block is a different
  artefact and is T5a's, not a survivor of this one.
- Rejected: keeping them unused (dead code the layout decision exists to prevent); keeping
  `DocSentences` as an option (an option whose only measured setting is harmful).

### D15 — `Repo`, and no language override

- Decision: the engine's entry point is a `Repo` value:

  ```go
  func Open(root string) (*Repo, error)   // root is an absolute, existing directory
  ```

  Every `target` argument and every emitted path is repository-relative with forward slashes. The
  `langOverride` parameter on both current entry points is **removed**.
- Rationale: §4's paths are "relative to the repository root, never absolute", so the root has to be
  a first-class value the engine holds. `Repo` holds the root and nothing else — the gitignore
  pattern set is read fresh per call, never cached on it (D22). The engine does no git discovery and
  no cwd resolution — that is the
  CLI's job in T5a, which keeps the engine testable against a fixture directory. `langOverride`
  contradicts §4's "the alphabet is chosen per file, never per repository".
- Rejected: package-level functions taking a root on every call (the ignore set would be re-read per
  call); the engine discovering the root itself (couples the engine to git and to a cwd).

### D16 — `SpansOf`: the internal span lookup

- Decision: `resolve.go` gets

  ```go
  func (r *Repo) SpansOf(g glyph.Glyph) ([]Symbol, error)
  ```

  It returns full `Symbol` values — the same type `toc` emits — each with `File` set to the
  declaration's repository-relative path. There is no separate `Span` type: a bare span would leave
  `Symbol.File` (D3) with no writer anywhere in T3, a dead carrier of exactly the kind D14 and D21
  reject elsewhere, and T4's `resolve` needs the whole entry anyway, since §4 says `resolve` returns
  "this entry and nothing else for a symbol". The round trip compares the
  `(File, Start, SigEnd, End)` tuples of what `toc` listed against those of what `SpansOf` returned.
- **`SpansOf` validates its argument through `glyph.Parse`.** A `Glyph` is a plain struct a caller
  can build by hand, so before anything is read `SpansOf` round-trips it —
  `glyph.Parse(g.Lang, g.String())` — and returns the resulting `*glyph.ParseError` wrapped on
  failure. That one check covers the empty unit, a `.`/`..` segment, a member that is too deep, and
  every other alphabet violation, and it covers them by *calling the one implementation of the
  grammar* rather than restating its rules in the engine. It is also what makes D7's claim
  "`SpansOf` never produces a root span" true rather than aspirational: `Glyph{Unit: ""}` is rejected
  here, instead of silently resolving to the repository root and returning spans `toc` never listed.
  A non-Go `Lang` is rejected by the same call, with `ErrLanguageUnsupported` per D21 taking
  precedence so the error names the real cause.

  It maps the glyph's unit to a directory **literal-first**, parses that directory's `.go` files, and
  returns every declaration whose owner chain and name match, ordered by file then start line. It has
  **no** status vocabulary: zero matches is an empty slice, not `not_found`.
- **Unit → directory, literal-first (the inverse of D7).** Given unit `U`:
  1. if the directory `U` exists, search it, restricted to files belonging to unit `U` by D7's rule —
     i.e. excluding any file whose clause is `<dirPackage>_test`, which belongs to `U + "_test"`;
  2. otherwise, if `U` ends in `_test` and the directory `strings.TrimSuffix(U, "_test")` exists,
     search that directory restricted to files whose clause is exactly `<dirPackage>_test`;
  3. otherwise, no directory — an empty slice (T4 turns this into `unit: not_found`).

  In every branch the directory's `.go` files are filtered through **the same ignore set `TOC` uses**
  (D9, collected fresh per call per D22) before being parsed. Without that filter a gitignored `.go`
  file beside listed ones would contribute spans `toc` never listed — a round-trip *extra* under
  Testing 14/15, and a `resolve` in T4 that points at a file `toc` says does not exist.
  When **both** directories exist, both interpretations are collected and the collision is recorded
  on the result for T4 to report as `ambiguous`; T3 itself returns the union of spans, so the round
  trip stays exact either way.
- Rationale for literal-first: a directory literally named `foo_test/` is legal Go, and D7's walk
  gives its declarations the unit `…/foo_test`. An unconditional strip would send the lookup into a
  `foo/` directory that need not exist, so one glyph string would name two different units. Neither
  quarry nor Loomyard has such a directory today, which means the round-trip criterion would never
  exercise the case — exactly why the rule has to be right by construction rather than by test, and
  T4's public `resolve` inherits it unchanged.
- **Gap recorded:** glyph.md §2 gives the external test unit the pseudo-path `<dir>_test` without
  saying what happens when a real directory spells the same string. This discussion fixes quarry's
  behaviour (literal-first, both-exist → `ambiguous`); amending glyph.md is not T3's to do, and the
  case belongs in T4's status-vocabulary work.
- Rationale: the task states "an internal span lookup suffices; the public `resolve` verb with its
  status vocabulary is T4". Putting it in `resolve.go` creates the file T4 grows rather than a file
  T4 has to move.
- Rejected: an unexported function (the round-trip test is a `_test.go` in the same package either
  way, but T4 needs it exported from the engine for the facade, and exporting it later is churn);
  building the status vocabulary now (that is T4's scope, and doing it here means two authors own
  one contract).

### D17 — Loomyard access and the pin

- Decision: the Loomyard checkout path comes from `LADDER_LOOMYARD_REPO` in the environment, never
  from a tracked file, matching T2's convention. Tests that need it: **skip** when the variable is
  unset or names a nonexistent directory; **fail** when it is set but the checkout's `HEAD` is not
  `72c23d9` (the commit §4's examples were taken at).
- **Mechanism for the pin check:** the test runs `exec.Command("git", "-C", <repo>, "rev-parse",
  "HEAD")` and compares the prefix. It does **not** parse `.git/HEAD` — the Loomyard checkout is a
  git *worktree*, whose `.git` is a file pointing into `worktrees/`, so the naive read gets the wrong
  ref or none. A `git` binary that is missing or errors → `t.Skip` with the reason, treated as "no
  usable checkout" rather than as a mismatch. D9's rejection of a git subprocess does not apply here:
  that was about the engine's per-answer read path, measured in milliseconds; this is one process per
  test run, in a test, and it is the only exact way to read a worktree's HEAD.
- Rationale: no tracked file may carry a machine path (CLAUDE.md / task constraints). The
  skip-versus-fail split is the difference between "this machine has no Loomyard", which is normal,
  and "this machine has the wrong Loomyard", which would let the task's own done-criterion pass
  without ever being checked. On this host the checkout is present and already at the pin.
- The gitignored `.scratch/ladder.env` holding `LADDER_LOOMYARD_REPO=<path>` is recreated per
  machine, exactly as T2 does.
- Rejected: skipping on a pin mismatch (silently unverifiable done-criterion); committing a fixture
  copy of Loomyard (large, and the round trip is meant to be over a real repository).

### D18 — Ordering is the engine's, and it is lexicographic

- Decision: `DirAnswer.Files` is sorted lexicographically by `Name`; `DirAnswer.Dirs` is sorted
  lexicographically by `Dir`; `FileEntry.Symbols` is in source order, ascending by `Start`. The
  engine sorts; the caller never does.
- Rationale: the current `types.go` says outright that "Ordering is the caller's (internal/cli's)
  responsibility, not this package's" — and that caller is T5a, out of scope here. §4's examples are
  alphabetical and the golden tests (Testing 12–13) compare marshalled JSON, so an unpinned order
  makes the byte-for-byte criterion untestable and a `-update` run would silently freeze whatever
  order `os.ReadDir` happened to produce on the machine that ran it. Sorting in the engine is also
  what makes the round trip (Testing 14–15) reproducible.
- Sorting is by the raw string, `sort.Strings` semantics, with no case folding and no locale — the
  same rule `TOCDir` already applies to its own two lists.
- Rejected: leaving it to T5a (the answer would not be reproducible, and T3 is the task whose
  done-criterion pins the bytes); insertion order from `os.ReadDir` (documented as unspecified).

### D19 — Symlinks are never followed

- Decision: the walk never follows a symlink. A directory entry whose `DirEntry.Type()` has
  `fs.ModeSymlink` set is emitted as a **file entry carrying `name` alone** — never descended into,
  never opened, no `header`, no `symbols` — whatever its target is. Detection is on `Type()`, never
  on `IsDir()`.
- Rationale: T3 introduces directory recursion for the first time (V1's `TOCDir` never descends), so
  `--depth all` over a real tree could otherwise loop forever or list one package under two paths,
  which would break glyph uniqueness — a package reached through a link and through its real path
  would spell two units for one set of declarations. `os.ReadDir`'s `IsDir()` is false for a
  symlink-to-directory, so keying on it would silently emit a directory as a *file* entry and read a
  "header" through the link; keying on `Type()` is what avoids that. Not following also means no
  visited-set is needed: the walk is finite by construction rather than by bookkeeping.
- This is live in quarry's own worktree today — `.active`, `.portals` and `.wiki` are directory
  symlinks — though all three are gitignored (D9) and so never reach this rule. Testing 14's
  self-round-trip is what would have found it the hard way.
- Rejected: following with a visited set (admits the same package under two units, and the answer
  then depends on which path was walked first); omitting symlinks entirely (§4's question is whether
  a file is worth opening, and hiding a tree member answers it wrongly).

### D20 — `target` validation and the error vocabulary

- Decision: `TOC`'s `target` is a repository-relative, slash-separated path. Validation, in order:
  1. an absolute path, or one that leaves the root once cleaned (any leading `..`) →
     `ErrTargetOutsideRepo`;
  2. a path that does not exist under the root → `ErrTargetNotFound`;
  3. a target that is **itself a symlink** is answered the way D19 lists one — a `name`-only file
     entry inside its parent's directory answer, never followed, never opened — so the stat is
     `os.Lstat`, never `os.Stat`. Naming the call matters: `os.Stat` follows the link and would
     silently descend into the target, contradicting D19 for the one path D19 does not cover;
  4. otherwise it is answered, **even when it is gitignored** — the ignore rule filters *listings*,
     never an explicit ask.

  Both sentinels are package-level wrapped with `fmt.Errorf("...: %w", …)`, so `errors.Is` survives
  wrapping. `""` and `"."` both mean the repository root and are valid.
- Rationale: T5a's `ok`/`status` and exit codes map exactly this vocabulary, and the discussion's own
  position is that T3 pins the envelope's shape — pinning only its success half would hand the
  failure half to a later task with nothing to build on. Answering a gitignored explicit target is
  the honest reading of §4: the filter exists so a listing is not noise, not to make a file
  unaddressable.
- Rejected: a single opaque error (T5a cannot map it to a status without string matching); refusing a
  gitignored target (`resolve`/`toc` on a real file would then fail for a reason the caller cannot
  see from the path).

### D21 — `ErrLanguageUnsupported` survives, with exactly one caller

- Decision: keep the sentinel, and make `SpansOf` its only caller — `SpansOf(g)` with
  `g.Lang != glyph.Go` returns it. Both of its current triggers are gone: D10 lists every file
  regardless of language, so an unmapped extension is no longer an error, and D15 deletes
  `langOverride`, so a language can no longer be requested. Its doc comment is rewritten to say so.
- Rationale: the sentinel would otherwise be dead code or, worse, silently repurposed. `SpansOf` is a
  real remaining trigger: a `glyph.Glyph` is a struct a caller can build by hand with any `Lang`, so
  the engine needs a defined answer for a language it has no extractor for, and T4's `resolve`
  inherits it.
- Rejected: deleting it (T4 would reintroduce it under a different name); keeping it for the
  extension case (that case is no longer an error).

### D22 — Nothing is cached, including the ignore set

- Decision: `Repo` holds the root and nothing else. The `.gitignore` pattern set is collected fresh
  on each `TOC` and `SpansOf` call — never read from a previous call, and extended per directory as
  the walk descends per D9's superseding paragraph — and discarded entirely when the call returns.
- Rationale: this closes a contradiction with the Constraints ("No cache, index, daemon or
  concurrency in the engine … Every answer reads source as it is at that moment"). It is not
  pedantry: T6's MCP server is a long-lived process, so a process-lifetime pattern set would go stale
  the moment a `.gitignore` is edited under it, and the resulting wrong file list would be
  indistinguishable from a bug in the walk. §5's measurements say nothing on quarry's side is worth
  optimising yet, and reading a handful of small files per call is far below the tree-sitter cost
  that dominates.
- Rejected: caching for the process lifetime (stale under T6); caching with an mtime check (that is
  the phase-1 cache §10 forbids, arriving early under another name).

## Technical context

**What exists on `main` and what it becomes.**

| today | after T3 |
|---|---|
| `internal/quarryengine/{doc,errors,cgoguard,cgoguard_nocgo}.go` | `internal/engine/{doc,errors}.go`; the guard pair → `internal/cgoguard/` (D2) |
| `internal/quarryengine/toc/types.go` | `internal/engine/answer.go`, rewritten per D3 / D12 |
| `internal/quarryengine/toc/toc.go` | `internal/engine/toc.go` + `walk.go` + `repo.go`, rewritten per D13 / D15 |
| `internal/quarryengine/toc/strategy.go` | `internal/engine/strategy.go`; `Strategy` gains `PackageDoc(root, src) string`, its `Symbols` gains the unit so it can build glyphs, and `Generated`/`TestFile` lose their `known` return (see below) |
| `internal/quarryengine/toc/doc.go` | folded into `internal/engine/doc.go` — the package comment merges with the root one, minus the sentence-boundary rule D14 deletes |
| `internal/quarryengine/toc/golang.go` | `internal/engine/golang.go`, widened per D5, gaining `PackageDoc` per D8 |
| `internal/quarryengine/toc/nodes.go` | `internal/engine/nodes.go`, mostly unchanged — reuse it |
| `internal/quarryengine/toc/comments.go` | folded into `internal/engine/text.go` with `FirstParagraph` |
| `internal/quarryengine/toc/classify.go` | `internal/engine/classify.go`, unchanged |
| `internal/quarryengine/toc/extension.go` | `internal/engine/extension.go` + the header-rule table (D10) |
| `internal/quarryengine/toc/{compact,sentences}.go` | deleted, except `FirstParagraph` (D14) |
| `internal/quarryengine/treesitter/` | `internal/engine/treesitter/`, gaining the cgoguard blank import (D2) |

**The `known` returns are dropped from `Strategy`.** `Generated(root, src) (generated, known bool)`
and `TestFile(base) (isTest, known bool)` become `Generated(root, src) bool` and
`TestFile(base) bool`. The `known` half existed only to feed D12's `*bool` fields, which D12 removes:
its whole job was to distinguish "this language has no rule" from "no", and with Go the only language
and both Go rules always known, it has no consumer left. `classify.go`'s `TestFileByName` and
`GeneratedByBanner` keep their two-value signatures — they are the shared per-language table and a
future language with no rule still needs to say so there — and the Go strategy simply discards the
second value. A second language reintroduces the field and the return together, against a real case.

**Helpers to reuse rather than rewrite.** `nodes.go` is the load-bearing file and its rules are
already correct against the byte-for-byte target:

- `Line` — the single 0-based → 1-based conversion.
- `SignatureCut(decl, body, src)` — a byte range, never a line truncation.
- `SigEnd(decl, body)` — body's start line, 0 when there is no body.
- `CommentBlockAbove(n, src)` — the prev-sibling comment walk with the blank-line boundary; it
  already doubles as the grouped-spec walk one level down, which D5's `const`/`var` specs need.
- `LeadingBlocks(root, src)` — the downward walk the header rule and the package-doc rule both use.
- `StripComment`, `FirstParagraph`, `IsDirectiveBlock`, `TestFileByName`, `GeneratedByBanner`.

**Verified against the target (Loomyard `72c23d9`, `internal/reedengine/render/layout.go`).** The
existing span rules already reproduce §4's numbers: `placement`'s doc comment starts at line 16, the
`type placement struct` clause is line 20, the closing brace is line 29 — §4 says
`"start": 16, "sigend": 20, "end": 29`. The directory has exactly the 12 files §4 names, all `.go`.
The current `Header` rule already yields §4's `layout.go` header verbatim (lines 1–7, one paragraph).

**Gotchas found during exploration.**

- `types.go` in the target directory carries a file header *and* a package doc as two separate
  leading blocks — the case D8's `Package <name>` prefix rule exists for.
- `goTypeBody` deliberately does not use `ChildByFieldName("body")`: a Go `type_spec` has no `body`
  field, and the naive call silently makes the whole struct body the signature. Keep the comment.
- `goDeclIsGrouped` keys on the literal `(` child, not on the spec count, because `type ( X int )`
  is legal with one spec. The same distinction applies to `const`/`var` under D5.
- `treesitter.WithTree` invalidates the root node when it returns; nothing may retain a `*ts.Node`
  past its callback. Every symbol must be fully materialised inside the callback.
- The pinned grammar is tree-sitter-go v0.25.0. Node kind names for `const`/`var`/interface methods
  must be confirmed against that pin before the walk is written, not assumed.
- `go.mod` still carries `tree-sitter-python` and `tree-sitter-rust` as indirect requirements. They
  are unused; `go mod tidy` at the end of the task should drop them.
- `README.md` names the three queries as "`map`, `resolve`, `members`" — both words are stale
  (`toc`, `resolve`, `expand`). Fix the stub while the engine is being renamed.

**The build needs cgo.** `CGO_ENABLED=1` and a C toolchain; `go build ./... && go test ./...` is the
gate.

## Constraints

- Go repository. **No Python** (`CLAUDE.md`).
- No tracked file may carry a machine path. The Loomyard checkout comes from
  `LADDER_LOOMYARD_REPO`; the gitignored `.scratch/ladder.env` holds it per machine.
- Scratch goes under `.scratch/`, never `/tmp` or any system temp directory.
- No new module dependency beyond what `go.mod` already has (tree-sitter + the Go grammar). The
  `glyph` package must stay pure Go with no dependencies — T3 imports it and never modifies it.
- The engine never re-implements the glyph grammar: parsing, printing and canonicalisation are
  `glyph`'s alone.
- No cache, index, daemon or concurrency in the engine (§5, §10). Every answer reads source as it is
  at that moment — and the gitignore pattern set is source for this purpose, so it too is read fresh
  per call (D22). `Repo` holds the root and nothing else.
- The walk never follows a symlink (D19), so it is finite by construction.
- The emitted key set is §4's and is closed. The two additions (`lossy`, `error`) are failure keys,
  absent on the happy path, so §4's examples still reproduce byte for byte.
- `go build ./... && go test ./...` green, and one merge to `main`.

## Testing

**TDD candidates** (pure functions, no tree-sitter, write the table first):

1. `ignore.go` — the gitignore matcher. Table over the pattern forms of D9, plus the two real
   `.gitignore` files (quarry's `/quarry` + `!/quarry/` pair, Loomyard's anchored, dir-only and
   `**/`-prefixed patterns) as fixtures.
2. `headers.go` — the non-code header extractors. One table per format, including: a Markdown file
   with a setext heading, one whose first paragraph follows a blank line, a shell script with a
   shebang before its comment block, a YAML file with no leading comment, an unknown extension.
3. `text.go` — `FirstParagraph` (kept), against a package comment that continues after a blank line
   (the `render` case) and one that does not.

**Fixture-driven unit tests** (`internal/engine/testdata/`, small Go trees committed in-repo):

4. The Go declaration walk: a package-level `func`, a method with a pointer and a value receiver, an
   ungrouped and a grouped `type`, a grouped `const`, a `var x, y int`, an interface with two
   methods, a type alias (no body → `sigend` absent). Assert each symbol's `id`, `kind`, span and
   `signature`.
5. Glyph assignment: `cmd/lyx#run` from a `package main`; `internal/logger#dualHandler.Handle` from a
   method; an interface method's owner.
6. The unit walk (D7): a directory with `pkg`, `pkg`'s own `_test.go`, and a `pkg_test` external
   test file — three files, two units; assert the file entry's `package` deviation key appears only
   on the external-test file. A directory whose package is legitimately named `httptest` must **not**
   be split.
7. Several `init` in one package → three symbols, one id, spans in file-then-line order (D6).
8. Package doc (D8): a `doc.go` carrying `// Package x …`; a directory where only a non-`doc.go`
   file carries it; a directory where a file header sits adjacent to the clause without the
   `Package` prefix (→ `doc` absent); a package whose comment has a second paragraph (→ first only).
9. The answer shape (D12/D13): `depth: 0` vs `1` vs `all`; `symbols` defaults for a file target and
   a directory target and both explicit overrides; a file target answering as a one-entry directory
   answer. Assert the *marshalled JSON*, so `omitempty` on `dirs`, `test` and `generated` is pinned.
10. Failure entries: an unreadable file, an invalid-UTF-8 file, a file the grammar reports an error
    on → `error` / `lossy` set and mutually exclusive, and the file still listed, never skipped.
11. `SpansOf` (D16): a hit, a miss (empty slice, no error), an external-test unit, a glyph whose unit
    directory does not exist, a fixture with a directory literally named `foo_test/` (literal-first
    wins), the both-exist collision (union returned, collision recorded), and a non-Go `Lang`
    (`ErrLanguageUnsupported`, D21).
11a. Ordering (D18): a fixture directory whose `os.ReadDir` order is not lexicographic — assert
    `files` and `dirs` come back sorted and symbols come back in source order.
11b. Symlinks (D19): a fixture with a symlink to a directory, a symlink to a file, and a symlink
    cycle (`a/ → b/ → a/`); assert each is a `name`-only file entry, that `--depth all` terminates,
    and that nothing behind a link is ever listed or parsed.
11c. Target validation (D20): an absolute target, a `..`-escaping target, a nonexistent target
    (each its own sentinel via `errors.Is`), a gitignored target (answered, not refused), a target
    that is itself a symlink (`name`-only entry, not followed), and `""` and `"."` both meaning the
    root, whose answer carries `dir: "."`.
11d. The root package (D7): a fixture with a `.go` file in the fixture root — assert it is listed
    with its header and that its entry carries **no** `symbols`, and that `SpansOf` returns nothing
    for any name in it.
11e. `const`/`var` derivation (D5): the five shapes of D5's table — ungrouped, grouped, several
    names in one spec (distinct ids, identical spans), a bare `iota` spec, and an interface method —
    each asserted on `id`, span, `signature` and the absence of `sigend`.
11f. Extensionless files (D10): a `Makefile` with a leading `#` block resolves through the
    base-name table; a `Dockerfile`; an extensionless file in neither table gets `name` alone.
11g. Ignore-set freshness (D22): two `TOC` calls on **one** `Repo` with the fixture's `.gitignore`
    rewritten between them — the second answer must reflect the new patterns. Nothing else in T3
    would notice a set accidentally cached on `Repo`, because the staleness it prevents only shows
    inside T6's long-lived process.
11h. Descendant `.gitignore` (D9): a fixture with an ignore file two levels down — assert
    `--depth all` honours it and that `SpansOf` on that descendant unit filters identically, so the
    file is neither a round-trip miss nor an extra.
11i. `SpansOf` argument validation (D16): a hand-built `Glyph{Unit: ""}`, one with a `..` segment,
    and one with a three-component member — each rejected via the `glyph.Parse` round-trip, with the
    `*glyph.ParseError` surfacing rather than a silent root-directory answer.
11j. Package-clause tie-break (D7): a fixture directory splitting evenly between two clauses —
    assert the lexicographically smaller one becomes the directory's `package` and that the answer is
    identical across repeated runs.

**Golden tests against Loomyard** (env-gated per D17):

12. `toc internal/reedengine/render` and `toc internal/reedengine/render/layout.go`, marshalled to
    JSON and compared to committed goldens under `internal/engine/testdata/loomyard/`. The goldens
    are the §4 examples with the plan's prose elisions (`...`) filled in from the real source —
    "byte for byte, apart from prose" means the *structure and key set* are exact and the prose is
    whatever the file actually says. Generate them once with a `-update` flag and review the diff
    against §4 by hand before committing.
13. `toc internal/reedengine --depth 0` — assert the subdirectory entry carries `dir`, `package` and
    `doc` and no other key, per §4's second example.

**Round trip** (the task's headline criterion):

14. Over **quarry itself** — always runs, no environment needed. Walk the repository with
    `--depth all --symbols`, collect every symbol, group by `id`, and assert for each glyph that
    `SpansOf(Parse(Go, id))` returns exactly the same *set* of spans (D6). Zero misses, zero extras.
15. Over **all of Loomyard** — the same assertion, env-gated, skipped under `-short`. Also assert
    that every listed symbol's `id` round-trips through `glyph.Parse` → `String()` unchanged, which
    is what "every declaration `toc` lists has a glyph" means operationally.

**Ported tests.** The existing `toc_test.go`, `golang_test.go`, `classify_test.go`,
`comments_test.go` and `extension_test.go` cases still describe live rules; port them onto the new
types rather than rewriting them. `compact_test.go` and `sentences_test.go`'s `FirstSentences` cases
are deleted with their subjects (D14); `sentences_test.go`'s `FirstParagraph` cases move to
`text_test.go`. `treesitter_test.go` is unchanged apart from its import path.

`toc_integration_test.go` is **ported, not deleted**: its value is that it runs the pipeline against
real repository source nobody wrote to satisfy this package's tests, and that value survives intact.
Every one of its couplings breaks, so all three are updated together — `TOCFile(path, "",
Options{DocSentences: 1})` becomes `Repo.TOC` on a file target (D13/D15), the hard-coded
`internal/quarryengine/treesitter/treesitter.go` becomes `internal/engine/treesitter/treesitter.go`
(D1), and the four-`filepath.Dir` climb from `runtime.Caller(0)` becomes three, since the test file
moves up one directory level. Its loose assertions on symbol names, kinds and range ordering stay
loose, for the reason its own comment gives.

**Gate.** `CGO_ENABLED=1 go build ./... && go test ./...`, plus `go vet ./...` and `go mod tidy`
leaving no diff.

## Q&A log

- **Q:** Where does the engine package live, and does the `toc/` subpackage survive? **A:** [auto-pick] One package `internal/engine`, files per concern; `treesitter` stays a subpackage. **Why:** plan §12 T3 says "one package, files per concern, never a package per verb", `quarryengine` stutters against the module path, and nothing outside `internal/` imports it yet, so the rename is free exactly now.
- **Q:** Where does the cgo guard go once the engine itself imports cgo transitively? **A:** [auto-pick] A new `internal/cgoguard` package, blank-imported by `treesitter`. **Why:** a guard inside the engine would be unreachable exactly when needed; making it a dependency of `treesitter` puts its readable error strictly before the linker error.
- **Q:** Does `Symbol` keep `Name`/`Owner` alongside the glyph? **A:** [auto-pick] No — `Glyph` is the single source of truth and `ID` is its string form. **Why:** two spellings of one identity is the drift the "one implementation of the grammar" rule exists to prevent.
- **Q:** Is the head span an explicit field or derived by T4? **A:** [auto-pick] Explicit `HeadStart`/`HeadEnd`, JSON-hidden, type-only. **Why:** the task text names it as something `Symbol` gains here; Go's head is trivially the type span but Python's and C#'s will not be, and the field is where that difference lands.
- **Q:** Does the walk gain `const`, `var` and interface methods? **A:** [auto-pick] Yes, with one symbol per declared name. **Why:** glyph.md §3 gives all of them glyphs; listing fewer leaves them unaddressable and makes the round trip pass vacuously.
- **Q:** How does the round trip handle several `init` under one glyph? **A:** [auto-pick] Set-equality of spans per glyph, not one-to-one. **Why:** glyph.md makes `<unit>#init` one glyph resolving `multipart`; a one-to-one check is unsatisfiable there and for build-tag duplicates.
- **Q:** How is the external test unit detected? **A:** [auto-pick] A package clause exactly equal to `<dirPackage>_test`, not any `_test` suffix. **Why:** a package legitimately named `httptest` or `mytest` must not be split into a second unit.
- **Q:** What counts as the package doc? **A:** [auto-pick] The block immediately above the `package` clause whose first line begins `Package <name>`, first paragraph, `doc.go` preferred. **Why:** verified against the byte-for-byte target — `render/types.go` carries a file header *and* a package doc, and only the prefix rule tells them apart.
- **Q:** How is gitignore handled? **A:** [auto-pick] A matcher implemented in the engine, no new dependency. **Why:** it is part of the answer, not an optimisation; shelling out to git puts a subprocess in a path measured in milliseconds and couples the engine to a git binary, and the needed subset is small and measured against the two repositories the tests run on.
- **Q:** What does `depth: 0` mean? **A:** [auto-pick] Direct children only, as `dir`/`package`/`doc`. **Why:** §4's knob table says so; the whole-tree orientation map is what `--depth all` is for, and any other reading makes `--depth 1` incoherent.
- **Q:** Do the compact view and `DocSentences` survive? **A:** [auto-pick] Deleted. **Why:** §10 makes compact-by-default a non-goal, §4 requires `doc` complete, and the ladder measured the cost directly (precision 0.96 → 0.82). The lossless text view is a different artefact and is T5a's.
- **Q:** `Test`/`Generated` as `*bool` or plain `bool`? **A:** [auto-pick] Plain `bool` with `omitempty`. **Why:** §4 forbids emitting `test: false`; the pointer encoded "this language has no rule", a state that cannot arise while Go is the only language.
- **Q:** What happens when `LADDER_LOOMYARD_REPO` points at the wrong commit? **A:** [auto-pick] Skip when unset, **fail** when set but `HEAD` is not `72c23d9`. **Why:** a skip on drift makes the task's own done-criterion silently unverifiable; a skip on absence is normal on a machine with no checkout.
- **Q:** How does `SpansOf` map a unit ending in `_test` back to a directory, given a directory may legally be named `foo_test/`? **A:** [auto-pick, discussion-review r1 gap] Literal-first: search the directory named exactly by the unit; only if it does not exist and the unit ends `_test`, strip and filter to the external test package; both existing → collect both and record the collision for T4 to report `ambiguous`. **Why:** an unconditional strip makes one glyph string name two different units, and neither quarry nor Loomyard has such a directory, so the round trip would never catch it — the rule has to be right by construction, and T4 inherits it.
- **Q:** Does T3 build the JSON envelope? **A:** [auto-pick] No — T3 emits §4's payload objects with their JSON tags and tests them marshalled; the `ok`/`status` envelope, exit codes and the text view are T5a. **Why:** §12 assigns the envelope to T5a but pins its shape with T3's byte-for-byte criterion, so the tags belong here and the wrapper does not.
