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
  `doc.go` (package doc), `answer.go` (`DirAnswer`, `FileEntry`, `Symbol`, `Kind`, `Span`),
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
  file entry; it is filled by `SpansOf` and by T4's `resolve`/`expand`, whose entries do span files.
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
- Note: a grouped `const`/`var` spec's docstring association reuses the grouped-type rule already in
  `nodes.go` (`CommentBlockAbove` walking a spec's prev-siblings inside the declaration), so this is
  a new node kind on an existing rule, not a new rule.
- Rejected: keeping the current three kinds (contradicts glyph.md); one symbol per *spec* rather than
  per name (`var x, y int` would then have one glyph for two symbols, and neither `x` nor `y` would
  resolve).

### D6 — `init` is one glyph, many declarations; the round trip is set-equality per glyph

- Decision: every `func init()` in a package gets `id: "<unit>#init"`. `toc` lists each declaration
  separately, with its own span, all carrying that one id. The round-trip check asserts, per glyph,
  that the *set* of spans `SpansOf` returns equals the *set* of spans `toc` listed under that glyph.
- Rationale: glyph.md §3 — "`internal/logger#init` is one glyph, and with several `init` functions it
  resolves `multipart`, every one returned in run order (file order, then line)". A one-to-one
  round trip would be unsatisfiable by construction here, and equally so for build-tag duplicates
  (which resolve `ambiguous`, several declarations, one glyph). Set-equality is the honest form of
  "zero misses, zero extras".
- `SpansOf` returns spans ordered by file then start line, so the ordering guarantee T4 needs is
  already in place.
- Rejected: strict one-to-one (fails on `init` and on build tags, the two cases §6 explicitly calls
  out); suffixing duplicate glyphs (`init#2`) — glyph.md forbids any spelling not in the alphabet.

### D7 — The unit walk: directory → unit, plus the external `_test` unit

- Decision: per directory,
  1. read every `.go` file's package clause;
  2. the **directory's package** is the most common clause among files whose clause does not end in
     `_test`; if every file's clause ends in `_test`, it is the most common clause overall;
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
  within a segment, and the "pattern without a slash matches at any depth" rule. Patterns are
  collected from `.gitignore` files from the repository root down to the directory being listed,
  later files and later patterns winning. `.git/` is always excluded.
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
  `/* … */`; `.yaml`/`.yml`/`.toml`/`.sh`/`.bash`/`.zsh`/`Makefile` → a leading `#` block, skipping a
  shebang line. Every other extension yields no header, so the entry is `name` alone. None of these
  files ever gets `symbols`, and the `symbols` knob does not change them.
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
  a first-class value the engine holds, and holding it is also where the gitignore pattern set (D9)
  is cached for the process. The engine does no git discovery and no cwd resolution — that is the
  CLI's job in T5a, which keeps the engine testable against a fixture directory. `langOverride`
  contradicts §4's "the alphabet is chosen per file, never per repository".
- Rejected: package-level functions taking a root on every call (the ignore set would be re-read per
  call); the engine discovering the root itself (couples the engine to git and to a cwd).

### D16 — `SpansOf`: the internal span lookup

- Decision: `resolve.go` gets

  ```go
  type Span struct {
      File   string
      Start  int
      SigEnd int
      End    int
  }

  func (r *Repo) SpansOf(g glyph.Glyph) ([]Span, error)
  ```

  It maps the glyph's unit to a directory (stripping a trailing `_test` and filtering to the external
  test package when present, per D7), parses that directory's `.go` files, and returns every
  declaration whose owner chain and name match, ordered by file then start line. It has **no** status
  vocabulary: zero matches is an empty slice, not `not_found`.
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
- Rationale: no tracked file may carry a machine path (CLAUDE.md / task constraints). The
  skip-versus-fail split is the difference between "this machine has no Loomyard", which is normal,
  and "this machine has the wrong Loomyard", which would let the task's own done-criterion pass
  without ever being checked. On this host the checkout is present and already at the pin.
- The gitignored `.scratch/ladder.env` holding `LADDER_LOOMYARD_REPO=<path>` is recreated per
  machine, exactly as T2 does.
- Rejected: skipping on a pin mismatch (silently unverifiable done-criterion); committing a fixture
  copy of Loomyard (large, and the round trip is meant to be over a real repository).

## Technical context

**What exists on `main` and what it becomes.**

| today | after T3 |
|---|---|
| `internal/quarryengine/{doc,errors,cgoguard,cgoguard_nocgo}.go` | `internal/engine/{doc,errors}.go`; the guard pair → `internal/cgoguard/` (D2) |
| `internal/quarryengine/toc/types.go` | `internal/engine/answer.go`, rewritten per D3 / D12 |
| `internal/quarryengine/toc/toc.go` | `internal/engine/toc.go` + `walk.go` + `repo.go`, rewritten per D13 / D15 |
| `internal/quarryengine/toc/strategy.go` | `internal/engine/strategy.go`; `Strategy` gains `PackageDoc(root, src) string` and its `Symbols` gains the unit so it can build glyphs |
| `internal/quarryengine/toc/golang.go` | `internal/engine/golang.go`, widened per D5, gaining `PackageDoc` per D8 |
| `internal/quarryengine/toc/nodes.go` | `internal/engine/nodes.go`, mostly unchanged — reuse it |
| `internal/quarryengine/toc/comments.go` | folded into `internal/engine/text.go` with `FirstParagraph` |
| `internal/quarryengine/toc/classify.go` | `internal/engine/classify.go`, unchanged |
| `internal/quarryengine/toc/extension.go` | `internal/engine/extension.go` + the header-rule table (D10) |
| `internal/quarryengine/toc/{compact,sentences}.go` | deleted, except `FirstParagraph` (D14) |
| `internal/quarryengine/treesitter/` | `internal/engine/treesitter/`, gaining the cgoguard blank import (D2) |

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
  at that moment.
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
11. `SpansOf` (D16): a hit, a miss (empty slice, no error), an external-test unit, and a glyph whose
    unit directory does not exist.

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
- **Q:** Does T3 build the JSON envelope? **A:** [auto-pick] No — T3 emits §4's payload objects with their JSON tags and tests them marshalled; the `ok`/`status` envelope, exit codes and the text view are T5a. **Why:** §12 assigns the envelope to T5a but pins its shape with T3's byte-for-byte criterion, so the tags belong here and the wrapper does not.
