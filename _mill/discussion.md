# Discussion: Facade + CLI, toc (T5a)

```yaml
task: Facade + CLI, toc (T5a)
slug: facade-cli-toc
status: discussing
parent: main
```

## Problem

The quarry rewrite has an engine but no way for anything outside the module to call it.
`internal/engine` is, by its own package doc, a package that "returns typed results and typed
errors only. It never emits JSON, never decides an exit code, and never resolves a caller's cwd" —
and it lives under `internal/`, so Loomyard cannot import it at all. Three of the four surfaces the
rewrite plan lists in §7 do not exist: the Go facade, the CLI, and the MCP.

**Why now:** T5a is on the critical path. The measurement that justifies the whole rewrite (T7)
runs `toc` over MCP; T6 can only be the thin MCP the plan demands if the facade and CLI already
answer `toc`. The chain is T0 → T1 → T3 → **T5a** → T6 → T7, and T3 (the engine core) merged at
`6ab5b27`. Nothing downstream can start until the public `toc` surface exists.

This task builds the first two public surfaces — the `quarry/` facade package and the `quarry` CLI
binary — for the `toc` verb alone, on top of the merged engine. `resolve` and `expand` join the same
two surfaces in T5b; nothing here may block on them, and nothing here may invent anything T5b would
have to redo.

## Scope

**In:**

- A new public package `quarry/` (the facade): an opened-repository handle, the `toc` query
  returning the engine's typed answer, and the two renderers (JSON envelope, lossless text view).
- A new binary `cmd/quarry` with one verb, `toc`, plus the flag set of plan §5
  (`--depth N|all`, `--symbols`) and this task's `--text`, `--root`, `--no-symbols`.
- The JSON envelope of plan §4 as emitted bytes: pretty-printed, key order fixed by the engine's
  struct field order, `ok` present only on failure so it agrees with the exit code.
- The exit-code contract (0/1/2/3) and the failure envelope.
- The lossless text view of the same data — no keys, no defaults, prose intact — specified below to
  the character, since plan §4 gives only two abridged examples.
- Repository-root discovery and cwd-relative target interpretation (the engine does neither).
- `docs/research/output-formats/after/`: the new outputs for the "before" side's `toc` commands,
  which double as this task's golden fixtures.
- Loomyard-free unit tests for flag parsing, exit-code mapping, root discovery, and text rendering.

**Out:**

- **Any change to `internal/engine`.** T5a is purely additive. If a genuine engine defect surfaces,
  it is reported, not fixed here — the engine's answer shape is pinned by T3's own byte-for-byte
  goldens and by T4, which is running in parallel.
- **`resolve` and `expand`** on either surface. T5b.
- **MCP.** T6. This task exports the renderers T6 will need and stops there.
- **Multiple targets in one `toc` invocation.** Decided out; see `single-target`.
- **A `--compact` view of any kind.** The lossy compact view is deleted by design (plan §1 lesson 4);
  the text view replaces it and is lossless.
- **Caching, a parser pool, a daemon, an index.** Plan §10.
- **Reading T4's in-flight artifacts.** T4 and T5a are deliberately independent (plan §12); the
  envelope is fixed by plan §4, not by anything T4 decides.
- **Machine paths in any tracked file**, including the new `after/` outputs.
- **Any change to `docs/rewrite-plan.md`, `docs/glyph.md`, `HANDOFF.md`, or `README.md`.** The README
  says "There is no command to build yet"; updating it is a one-line true statement T5b/T6 will
  rewrite anyway, so leave it.

## Decisions

### facade-types-are-aliases

- **Decision:** `quarry/` declares no answer structs of its own. It exposes the engine's types
  through Go type aliases:
  `type DirAnswer = engine.DirAnswer`, `FileEntry`, `Symbol`, `Kind`, `TOCOptions`, plus the `Kind`
  constants, `DepthAll`, and the error sentinels `ErrTargetNotFound` / `ErrTargetOutsideRepo` /
  `ErrLanguageUnsupported` re-declared as `var ErrTargetNotFound = engine.ErrTargetNotFound` so
  `errors.Is` keeps working across the boundary.
- **Rationale:** an alias to a type in an `internal/` package is nameable and usable by an external
  importer — the `internal` rule is enforced on import paths, not on types reached through an alias.
  So Loomyard can write `var a quarry.DirAnswer` without importing `internal/engine`. This gives the
  public surface with zero duplication. The engine's `answer.go` already carries the exact JSON tag
  set plan §4 fixes, and its own doc says no field is added or renamed without a Shared Decision
  change; a parallel struct in `quarry/` would be a second place that could drift from it — exactly
  the "three design generations in one CLI" failure the rewrite exists to end.
- **Rejected:** *(a)* facade declares its own structs plus a conversion layer — duplication, drift,
  and a conversion function that must be re-audited every time the envelope changes. *(b)* Move the
  answer types into a third public package that both `internal/engine` and `quarry/` import — a real
  option, but it moves T3's just-merged code for no gain the aliases do not already give, and T5a is
  additive by decision. *(c)* Return `engine.DirAnswer` directly with no alias — compiles, but
  Loomyard could then only use `:=` and never declare the type, and `go doc quarry` would point at
  an unimportable package.

### facade-entry-point

- **Decision:** the facade mirrors the engine's shape exactly:

  ```go
  func Open(root string) (*Repo, error)
  func (r *Repo) TOC(target string, opts TOCOptions) (DirAnswer, error)
  ```

  `Open` takes an absolute path to an existing directory and delegates to `engine.Open`, wrapping
  its error. `TOC` delegates to `engine.(*Repo).TOC` unchanged — no filtering, no re-shaping, no
  defaulting beyond what `TOCOptions` already encodes.
- **Rationale:** the facade's job in phase 1 is to be a *public name* for the engine, not a second
  design. A one-to-one mapping means there is no facade-specific behaviour to document, test twice,
  or diverge. The plan calls the facade "the primary surface" precisely because it is the thinnest.
- **Rejected:** *(a)* package-level `quarry.TOC(root, target, opts)` with no handle — re-validates
  the root on every call and gives a long-lived Go consumer nowhere to hold state if phase 2 ever
  wants one. *(b)* A single options struct carrying the root — a different shape from the engine for
  no reason, and it makes `root` a per-query rather than per-repository fact.

### renderers-live-in-the-facade

- **Decision:** the JSON and text renderers are exported from `quarry/`, not from the CLI and not
  from a new `internal/` package:

  ```go
  func RenderJSON(a DirAnswer) ([]byte, error)          // pretty, 2-space, trailing "\n"
  func RenderErrorJSON(msg string) []byte               // {"ok": false, "error": "..."} + "\n"
  func RenderText(a DirAnswer, targetIsFile bool) string
  ```

- **Rationale:** plan §12 requires T6's MCP to be *thin* and to sit *over the facade*. If the
  renderers live in `cmd/quarry`, T6 either re-implements them (a second generation of the
  envelope) or imports a `main` package (impossible). Exporting them here makes T6 a transport
  shim and nothing else, and makes the CLI a flag parser and nothing else.
- **Rejected:** *(a)* `internal/render` imported by both — works inside the module, but Loomyard
  could never render an answer it already holds, and there is no reason to withhold it. *(b)* CLI
  owns rendering — forces T6 to duplicate. *(c)* Methods on `DirAnswer` (`a.JSON()`, `a.Text()`) —
  impossible, `DirAnswer` is an alias for a type in another package, so `quarry/` cannot define
  methods on it. This is a hard constraint of the alias decision and mill-plan must not plan around
  it.

### ok-is-absent-on-success

- **Decision:** a successful `toc` emits the bare directory answer with **no `ok` key**. A failed
  `toc` emits `{"ok": false, "error": "<message>"}` and nothing else. The exit code carries success.
- **Rationale:** plan §4 says two things that only this reading satisfies together. First, "`ok`
  agrees with the exit code" — the V1 defect was `ok: true` alongside exit 2. Second, in the same
  section's "shared facts once, defaults never" list, "**`ok: true` inside data**" is named
  explicitly as V1 clutter to remove, alongside `test: false` and empty `dirs: []`. And the
  task's done-when requires "the envelope matches plan §4 byte-for-byte where §4 gives examples" —
  every §4 example is a bare object with no `ok` key at all. Present-only-when-false satisfies all
  three: `ok` never lies about the exit code, never appears as a default, and the examples match.
- **Rejected:** *(a)* `ok: true` always present as a sibling key — contradicts §4's own clutter list
  and breaks byte-for-byte against every §4 example. *(b)* A wrapper `{"ok": …, "answer": {…}}` —
  invents a second envelope layer §4 does not describe and breaks the examples. *(c)* No `ok` key
  ever, even on failure — then a caller reading the JSON alone cannot tell a failure from a
  truncated read.

### failure-envelope-and-exit-codes

- **Decision:**

  | exit | meaning | JSON on stdout |
  |---|---|---|
  | 0 | answered | the directory answer |
  | 1 | the query has a negative answer: target not found, or target outside the repository | `{"ok": false, "error": …}` |
  | 2 | usage error: unknown flag, missing or extra argument, unparseable `--depth`, `--root` that is not a directory, no repository root discoverable | `{"ok": false, "error": …}` |
  | 3 | internal error: an I/O failure the engine reported that is neither of the above | `{"ok": false, "error": …}` |

  The JSON envelope always goes to **stdout**, including on failure, **and including under
  `--text`** — there is no text rendering of a failure. A human-readable message (and, for exit 2,
  the usage text) goes to **stderr**. The failure payload carries `ok` and `error` only — no `kind`,
  no `status`.

  `--text` selects a view of an *answer*; a failure has no answer to view, and its payload is two
  fields with no prose in it, so a text spelling of the same two fields would be a second envelope
  for zero gain. A `--text` caller that must distinguish success from failure reads the exit code,
  which is what the exit code is for.
- **Rationale:** a scripting contract that puts machine output on stdout unconditionally lets
  `quarry toc x | jq` work on both paths; a caller that wants the human message reads stderr or the
  exit code. Splitting `not found` (1) from `usage` (2) matters to a gate: exit 1 is quarry
  answering "no", exit 2 is the caller having asked wrong, and conflating them makes a wrapper
  script unable to tell a real negative from its own bug. Exit 3 keeps genuine I/O failure from
  masquerading as a negative answer. A `kind` field is deliberately omitted: the exit code already
  discriminates, and duplicating it inside the payload is precisely the redundancy §4's
  "shared facts once" rule bans.
  Mapping: `errors.Is(err, engine.ErrTargetNotFound)` → 1, `errors.Is(err, engine.ErrTargetOutsideRepo)`
  → 1, anything else from `TOC` → 3, everything caught before `TOC` runs → 2.
- **Rejected:** *(a)* 0/1 only — a gate cannot tell "no such directory" from "you typo'd a flag".
  *(b)* failure JSON on stderr — then stdout is empty on failure and a pipeline sees nothing to
  parse, which is what makes the exit code the *only* signal again. *(c)* a `kind` discriminator —
  redundant with the exit code.

### single-target

- **Decision:** `quarry toc` takes **exactly one** target. Two or more targets is a usage error
  (exit 2). The facade likewise offers a single-target `TOC`; a Go caller that wants several loops.
- **Rationale:** plan §5 sketches the verb as `toc <dir|file>...`, but plan §4 — which is the
  section the task's constraints name as fixing the envelope — defines exactly one answer shape, the
  recursive directory answer, and gives no shape whatsoever for "several answers". Every way of
  emitting more than one answer invents an envelope §4 does not fix, and the task's constraint is
  explicit: "The envelope is fixed by plan §4". Shipping an invented multi-answer shape now is how a
  second generation starts. The single-target CLI plus the facade's loop covers every consumer that
  exists today: T6's MCP tool takes one target, and Loomyard's diff-to-symbols use (plan §8.2) is Go
  code calling the facade, where a loop is the natural spelling anyway.
- **Rejected, and recorded so a later task can pick one deliberately rather than rediscover them:**
  *(a)* **Top-level JSON array always.** One shape for all arities, but breaks the byte-for-byte
  done-when against every §4 example.
  *(b)* **Arity-dependent: bare object for one target, array for N > 1.** Keeps §4's examples exact
  and the caller always knows its own arity — but a shape that changes with the argument count is
  the kind of deviation §4 opens by naming as what made V1 three generations.
  *(c)* **Common-ancestor spine:** answer several targets as the directory answer for their nearest
  common ancestor, with `dirs` filled along the path to each target and the spine carrying identity
  only, exactly as a depth-cut subdirectory does. This is the most §4-consistent of the three — one
  recursive type, shared facts once — and is the one to revisit first if multi-target is ever
  wanted. It is out of T5a because it needs its own decisions on ordering, on de-duplication, and
  on a target nested inside another target, none of which §4 settles.
- **Known divergence this task deliberately leaves standing.** `docs/rewrite-plan.md` §5 spells the
  verb `toc <dir|file>...`, so once T5a merges the plan text claims an arity the CLI answers with
  exit 2. That file is on this task's Out list and must not be edited here. Amending or annotating
  §5's spelling is a one-line follow-up owned outside T5a; the plan writer carries the
  single-target contract as decided above and does not attempt to reconcile the plan text.

### cli-shape

- **Decision:** one binary, `cmd/quarry`, built as `quarry` at the repository root. Invocation:

  ```
  quarry toc <target> [--depth N|all] [--symbols|--no-symbols] [--text] [--root <path>]
  ```

  The verb is a positional word, not a subcommand framework: `main` dispatches on `os.Args[1]`, and
  an unknown or missing verb is a usage error (exit 2). The parseable core is
  `internal/cli.Run(args []string, stdout, stderr io.Writer) int` — `main` is `os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))`
  and holds nothing else.
- **Rationale:** the repository's own `.gitignore` already reserves `/quarry` as "a binary built at
  the repo root" while re-including the `quarry/` package directory, so the intended binary name is
  settled. A `Run(args, stdout, stderr) int` core is the standard Go pattern for a testable CLI: the
  golden tests capture exactly the bytes the binary emits without building or exec'ing anything, and
  the exit code is a return value rather than a process status. No CLI framework is added — the
  module has three dependencies today (tree-sitter, its grammar, yaml for the harness) and one verb
  with five flags does not justify a fourth.
- **Rejected:** *(a)* everything in `package main` — untestable without `os/exec`. *(b)* cobra or
  similar — a dependency for one verb. *(c)* a binary per verb (`quarry-toc`) — T5b would add two
  more binaries and the plan describes one CLI.

### root-discovery-and-target-interpretation

- **Decision:** the CLI resolves the repository root by walking up from the process's working
  directory looking for a `.git` entry (file or directory, so a worktree's `.git` file counts), and
  stops at the filesystem root. `--root <path>` overrides discovery entirely and skips the walk; the
  path is made absolute and must be an existing directory. If discovery finds nothing and `--root`
  was not given, that is a usage error (exit 2) with a message naming both fixes.

  The **base directory for a relative target is `--root` when `--root` was given, and the process's
  working directory otherwise.** One rule, one sentence: `--root` says "answer as if run at this
  root", so it rebases target interpretation along with discovery — the alternative would make
  `quarry toc --root <repo> internal/logger` fail from anywhere outside that repo, which is the only
  way `--root` is ever used from a script or a test. An absolute target is accepted in both modes
  and needs no base.

  The target is resolved against that base to an absolute path, then converted to a clean,
  forward-slash, repository-relative path before it reaches the engine. A target that resolves
  outside the root is reported as `ErrTargetOutsideRepo` (exit 1) by the CLI itself, without calling
  the engine. Native path separators are accepted on input; every path in the output is
  forward-slash and repository-root relative, which is what the engine already produces.
- **Rationale:** the engine deliberately does "no git discovery … and no cwd resolution", and its
  doc says so; that work has to happen somewhere, and the CLI is the only layer that has a cwd. Doing
  it this way means `quarry toc .` inside `internal/logger` answers for `internal/logger`, which is
  what every comparable tool (`git`, `rg`, `go`) does, while the *output* stays repo-root relative as
  plan §4 requires — so the two path frames never mix: input is where the user is, output is where
  the repository is.
- **Rejected:** *(a)* targets interpreted as repository-relative regardless of cwd — matches the
  "before" outputs' invocation style (which was always run from the repository root) but makes
  `quarry toc .` mean the repository root from inside a subdirectory, which is surprising and
  silently expensive. *(b)* cwd *is* the root unless `--root` is passed — then running from a
  subdirectory silently changes every emitted path and every glyph unit, producing answers that
  look right and are wrong. *(c)* `--root` mandatory — hostile for interactive use, and the ladder
  and MCP would both have to compute it.

### flag-semantics

- **Decision:**
  - `--depth <N|all>`: `N` a non-negative integer, `all` maps to `engine.DepthAll`. A negative
    integer or an unparseable value is a usage error (exit 2). Default 0, matching plan §4's table.
    For a file target `--depth` is accepted and ignored, as the engine already documents; it is not
    an error.
  - `--symbols` sets `TOCOptions.Symbols` to `&true`; `--no-symbols` sets it to `&false`; neither
    leaves it `nil`, which is the engine's per-target default (true for a file target, false for a
    directory target, per plan §4's table). Both flags together is a usage error.
  - `--text` selects the lossless text view; absent means JSON. There is no `--json` flag — JSON is
    the default and naming it would imply a third format exists.
  - `--root <path>` per `root-discovery-and-target-interpretation`.
  - `--help` and `-h`, at any position and with or without a verb, print the usage text to **stdout**
    and exit **0**. They are stated explicitly because the flag set is otherwise closed and would
    silently route the one flag every user tries first into "unknown flag → exit 2". An explicit
    request for help is a successful query about the CLI, not a usage error; usage text on *stderr*
    with exit 2 is reserved for the case where the caller got the invocation wrong.
- **Rationale:** `TOCOptions.Symbols` is a `*bool` precisely so "not asked" is distinguishable from
  "asked for false", and the CLI is the layer that must express three states with flags. A bare
  `--symbols` bool cannot say "false explicitly", and `-symbols=false` (what Go's `flag` package
  gives) is an obscure spelling for a documented behaviour, so the explicit `--no-symbols` pair is
  worth the one extra flag. `--depth all` cannot be a plain int flag, so depth is parsed as a string
  and validated by hand.
- **Rejected:** *(a)* `--format json|text` — extensibility for a third format that plan §10's
  "compact-by-default" non-goal says will not exist. *(b)* `--symbols=false` only — works, is
  undiscoverable. *(c)* accepting `--depth -1` as `all` — an internal sentinel leaking into the
  user-facing contract.

### json-rendering-details

- **Decision:** `RenderJSON` uses `encoding/json` with two-space indentation, no HTML escaping
  (`json.Encoder` with `SetEscapeHTML(false)` and `SetIndent("", "  ")`), and a single trailing
  newline. Key order is the struct field declaration order in `internal/engine/answer.go`, which is
  already plan §4's order (`dir`, `package`, `language`, `doc`, `files`, `dirs`; and within a file
  entry `name`, `header`, `test`, `generated`, `package`, `language`, `lossy`, `error`, `symbols`).
- **Rationale:** the "before" outputs have alphabetically sorted keys — a `map` was marshalled
  somewhere in V1 — which is why `toc-dir.txt` reads `dirs`, `files`, `ok` and puts `generated`
  before `header`. Struct-order output is both the natural result of marshalling the typed answer
  and the order §4's examples are written in, so it is what makes the byte-for-byte done-when
  achievable at all. `SetEscapeHTML(false)` matters because headers and docs are real prose and
  contain `<`, `>` and `&`; the default encoder would emit `<` and make the output unreadable
  and unequal to §4's examples.
- **Rejected:** *(a)* compact single-line JSON from the CLI — the before side is pretty-printed and
  a human reads these files; T6 can call `json.Marshal` on the same value if the MCP wants compact.
  *(b)* a hand-written marshaller to control key order — the struct tags already give it.

### text-view-grammar

- **Decision:** plan §4 gives two abridged text examples and no rules, so the grammar is fixed here
  in full. `RenderText` emits, with no trailing whitespace on any line and exactly one trailing
  newline:

  **Prose normalisation, everywhere.** Every `doc`, `header`, `signature` and `error` value has its
  internal newlines and runs of whitespace collapsed to single spaces before it is printed. `error`
  is in the list because `FileEntry.Error` is an arbitrary error string — an `os` or UTF-8 message
  quarry does not author — and it is emitted inside a bracketed tag, so a multi-line value would
  break the one-record-per-line property the whole format rests on. This is not an
  invention: §4's own text example prints
  `placement is one resolved pane: its tmux pane id and the row height it has been assigned.` for a
  `doc` whose JSON value contains `it\nhas been assigned.` "Prose intact" means nothing is
  truncated or dropped, not that source line breaks survive. Nothing else is altered — no
  re-wrapping, no ellipsis, no width limit.

  **Directory form** (the target was a directory). One block per directory answer, in depth-first
  order, blocks separated by one blank line. Each block:

  ```
  <dir> (package <pkg>, <lang>), <N> files
  <doc>
  <name>[ <tags>]: <header>
  ...
  ```

  - Line 1: `<dir>`, then ` (package <pkg>, <lang>)` only when `package` is present — and `, <lang>`
    within it only when `language` is present — then `, <N> files` only when the answer has files,
    with `1 file` singular. A depth-cut subdirectory that carries only identity and doc emits line 1
    and its doc line, nothing more.
  - Line 2: the directory's `doc`, normalised, on one line. Omitted entirely when absent — no blank
    line, no placeholder. A missing package doc is a repository invariant plan §8.2 wants
    *reportable*, and its absence in the text view is visible as the file list starting immediately.
  - One line per file entry, in the answer's own order (the engine sorts lexicographically by name):
    the bare `name` — not the path, the directory heads the block — then the tags, then `: <header>`
    when a header is present. A file with no header emits the name and tags and no colon.
  - `<tags>` is a space-separated run of bracketed markers, emitted only when the underlying field is
    present, in this fixed order: `[test]`, `[generated]`, `[package <pkg>]`, `[language <lang>]`,
    `[lossy]`, `[error <message>]`. This is the "before" side's own compact-view convention
    (`logsdir_test.go [test] (package logger_test):`) with parentheses regularised to brackets so
    one delimiter carries every deviation.
  - When a file entry carries `symbols`, its symbol lines follow immediately (see below), and the
    next file's line follows those. A file whose `symbols` key is absent contributes no symbol lines
    and no marker.

  **File form** (the target was a file). §4's example exactly — a single line carrying the file's
  full repository-relative path and the enclosing directory's package facts, then the symbol lines:

  ```
  <path> (package <pkg>, <lang>)[ <tags>]: <header>
  <symbol lines>
  ```

  `<path>` is the engine's own repository-relative join of the answer's `dir` and the entry's
  `name` — `walk.go`'s `joinRel`, which returns the bare name when `dir` is `"."`. A file target at
  the repository root therefore emits `README.md`, never `./README.md`; the naive `<dir>/<name>`
  template would emit the latter and contradict this grammar's own "full repository-relative path".

  The directory's `doc` is not printed in this form (§4's example omits it), and the block is not
  headed by a directory line. This form is selected by `RenderText`'s `targetIsFile` argument, never
  inferred: a directory holding exactly one file and no subdirectories is indistinguishable from a
  file target by shape alone, so the caller — which knows what it asked for — must say.

  **Symbol lines**, in both forms, one per symbol in the answer's order, each followed by its doc:

  ```
  <start>-<end>[ (sig <start>-<sigend>)] <id>: <signature>
      <doc>
  ```

  - The `(sig …)` group is omitted when `sigend` is absent (zero — the engine's documented marker for
    a symbol with no body, such as a Go type alias).
  - The doc line is indented four spaces and omitted entirely when the symbol has no doc — no blank
    line, no placeholder.
  - `<id>` is the glyph verbatim. Plan §4: "Glyphs as keys in every output".
- **Rationale:** every rule above is either read off one of §4's two examples or is the smallest
  choice consistent with "no keys, no defaults, prose intact": a field that is absent in JSON is
  absent in the text, which is what makes the two views carry the same information. Blocks-with-full-paths
  rather than indentation keeps every directory line copy-pasteable as a next argument and keeps the
  format flat enough that `--depth all` output is greppable.
- **Rejected:** *(a)* indentation-nested subdirectories — deep trees indent off the right edge and
  the dir line stops being a usable path. *(b)* preserving source newlines in prose — contradicts
  §4's own worked example and would break the one-record-per-line property the whole format rests
  on. *(c)* `key: value` suffixes instead of bracketed tags — that is the JSON view spelled worse,
  and §4 says the text view has no keys. *(d)* rendering a file target with the directory form —
  one fewer code path, but does not match §4's example.

### text-view-is-not-a-wire-format

- **Decision:** the text view is a rendering. It is lossless in the sense that no datum present in
  the answer is dropped, but it is not required to be machine-parseable back into a `DirAnswer`, and
  no parser or round-trip test is written. JSON remains the contract.
- **Rationale:** plan §4 is explicit — "JSON is the contract for the CLI and the facade. The MCP
  `content[].text` block carries a lossless text view of the same data". The text view's audience is
  an LLM reading prose, and whether it beats JSON for an agent is stated in the same paragraph to be
  "a ladder cell, not an assumption". Building a parser would fix a syntax nobody has measured a
  need for and would immediately constrain the prose (a doc containing `: ` or a leading `[` would
  become a quoting problem).
- **Rejected:** define a grammar and round-trip it — cost with no consumer, and it would force
  escaping into a view whose whole point is unescaped prose.

### t5a-does-not-change-the-engine

- **Decision:** no file under `internal/` is modified. If the facade or the golden run exposes an
  engine defect, the plan records it as a finding for a follow-up task and works around it in the
  `after/` fixtures by capturing what the engine actually emits, with a note.
- **Rationale:** T3's own done-when pins the engine's output byte for byte against Loomyard, and T4
  is in flight against the same package in a parallel worktree. An engine edit here means a merge
  conflict with T4 at best and a silent contradiction of T3's goldens at worst. The task's own
  constraint — "the envelope is fixed by plan §4, not by anything T4 decides" — cuts both ways.
- **Rejected:** allow small engine fixes — there is no such thing as a small change to a contract
  two other tasks have pinned.

### grammars-and-concurrency

- **Decision:** the facade adds no caching, no parser pool, and no state beyond the repository root.
  §7's "grammars loaded once per process" is recorded as **already satisfied**: the Go grammar is a
  static symbol linked into the binary, and `treesitter.WithTree`'s per-call `ts.NewLanguage` only
  wraps that static pointer in a Go value — it does not read or build a grammar. The facade's `*Repo`
  is documented as safe for concurrent use by multiple goroutines, because it holds only the root
  string and every query reads the filesystem fresh, exactly as `engine.Repo` documents.
- **Rationale:** plan §10 lists "a daemon, an index, or a cache in phase 1" as a non-goal, and §5's
  measurements conclude "Nothing on quarry's side is worth optimising yet". Adding a parser pool to
  satisfy a phrase that is already true would be both a non-goal and unmeasured.
- **Rejected:** a `sync.Pool` of parsers or a package-level cached `*ts.Language` in the facade —
  unmeasured, and it would put mutable state in the one layer the plan wants thin.

### after-outputs-are-the-goldens

- **Decision:** `docs/research/output-formats/after/` holds four output files and an index, and those
  four files *are* the golden fixtures the task's tests compare against. A test gated on
  `LADDER_LOOMYARD_REPO` at pin `72c23d9` — the same gate and pin `internal/engine/loomyard_test.go`
  already implements — runs the CLI in-process and compares its bytes to the committed file; under
  `-update` it rewrites the file instead.

  | after file | command captured |
  |---|---|
  | `after/toc-dir.txt` | `quarry toc internal/logger` |
  | `after/toc-file.txt` | `quarry toc internal/logger/logger.go` |
  | `after/toc-dir-text.txt` | `quarry toc --text internal/logger` |
  | `after/toc-file-text.txt` | `quarry toc --text internal/logger/logger.go` |
  | `after/INDEX.md` | the before→after mapping and what changed |

  Each `.txt` is exactly: the invocation line `$ quarry toc …`, a blank line, and the output
  verbatim. **No exit-code trailer.** This matches the four before-side files these are paired with
  — `toc-dir.txt` ends at its closing `}`, `toc-file.txt` likewise, and both `toc-*-compact.txt` end
  at their last text line. The before side's `INDEX.md` claims "the exit code is at the bottom of
  each file", but that is true only of `impact*`, `definition*` and `assert-no-callers`, never of
  the `toc` files; do not propagate the claim. Since these files are byte-compared goldens, the
  trailer decides their bytes, so it is decided on its own merits here: all four commands exit 0,
  a constant line saying so adds nothing a reader needs, and the exit code is asserted by the test
  rather than by a line in the fixture. `after/INDEX.md` states the
  Loomyard checkout and pin the outputs were taken at (no absolute path), maps each before file to
  its successor, and records that `toc-dir-compact.txt` and `toc-file-compact.txt` have **no**
  successor by design: the compact view was the lossy one-sentence-per-file view whose
  precision loss (0.96 → 0.82) is plan §1 lesson 4, and the lossless text view replaces it rather
  than continuing it.
- **Rationale:** the task's done-when asks for `after/` outputs, and its scope asks for golden tests
  on the same commands. Making them one artifact means the committed evidence and the regression
  gate cannot disagree, and it reuses a pattern already in the repository rather than inventing a
  second golden mechanism. The two targets are the before side's own two `toc` commands, so the
  pairing is exact.
- **Rejected:** *(a)* goldens under `testdata/` plus a separate manual step that writes `after/` —
  two artifacts that drift, and the `after/` files stop being evidence of anything the tests check.
  *(b)* Using the plan §4 targets (`internal/reedengine/render`) for `after/` — those already have
  goldens in `internal/engine/testdata/loomyard/` from T3; the `after/` side exists to pair with the
  *before* side, which used `internal/logger`.

### golden-tests-run-the-cli-in-process

- **Decision:** the golden test calls `cli.Run([]string{"toc", "--root", <loomyard>, <target>, …},
  &stdout, &stderr)` with the target given **repository-relative**, captures stdout, asserts the
  returned exit code is 0, and compares the assembled file. No `go build`, no `os/exec`, no
  `t.Chdir`.
- **Rationale:** `Run` is the whole CLI below `os.Exit`, so this exercises flag parsing, the engine
  call, rendering, and the exit code in one assertion, at unit-test speed and with no build step in
  the test. Passing `--root` explicitly rather than changing directory keeps the test free of
  process-global state, which matters because Go tests in a package share a process and the root
  discovery walk reads the working directory. A repository-relative target is correct here *because*
  `--root` rebases target interpretation (see `root-discovery-and-target-interpretation`); without
  that rule the target would resolve against the quarry test process's own cwd, land outside the
  Loomyard root, and return exit 1 instead of a golden. The two decisions are load-bearing on each
  other and must not be changed independently.
- **Rejected:** *(a)* build the binary and exec it — slow, needs a writable temp dir and a working
  toolchain inside the test, and adds nothing since `main` is one line. *(b)* `t.Chdir` into the
  Loomyard checkout to exercise discovery — process-global, and discovery gets its own dedicated
  test on a synthesised tree instead.

## Technical context

**The engine's public surface, all of it.** `internal/engine` (`repo.go`, `toc.go`, `answer.go`):

- `engine.Open(root string) (*Repo, error)` — root must be absolute and an existing directory;
  performs no git discovery and no cwd resolution.
- `(*Repo).TOC(target string, opts TOCOptions) (DirAnswer, error)` — the only query T5a needs.
  `target` is repository-relative with `""` and `"."` both meaning the root. Returns
  `ErrTargetOutsideRepo` (absolute target, or one that cleans outside the root) and
  `ErrTargetNotFound`, both wrapped with `fmt.Errorf("…: %w", …)` so `errors.Is` works.
- `(*Repo).SpansOf(g glyph.Glyph) ([]Symbol, error)` — T5b's, not used here.
- Types: `DirAnswer{Dir, Package, Language, Doc, Files, Dirs}`, `FileEntry{Name, Header, Test,
  Generated, Package, Language, Lossy, Error, Symbols}`, `Symbol{Glyph, ID, Kind, File, Start,
  SigEnd, End, Signature, Doc, HeadStart, HeadEnd}`, `Kind` with five constants, `TOCOptions{Depth
  int, Symbols *bool}`, `DepthAll = -1`.
- `Symbol.Glyph`, `Symbol.HeadStart` and `Symbol.HeadEnd` are `json:"-"` — hidden from the wire,
  consumed by T5b's `expand`. `Symbol.File` is `omitempty` and empty inside a toc answer by design
  (the symbol already sits in its file's entry); leave it that way.
- `FileEntry.Symbols` is `*[]Symbol`, the one pointer field, so nil ("not requested") is
  distinguishable from an empty slice ("requested, none found"). `omitempty` drops both from JSON —
  the distinction exists for Go callers only.

**Gotchas found during exploration, all of which the plan must account for:**

1. **`--symbols` does not guarantee a `symbols` key.** `walk.go`'s `fileEntry` leaves `Symbols` nil
   when the file's unit is unspellable as a glyph, even with `wantSymbols` true, and a file with no
   language never gets symbols whatever the flag says. Neither view may assume the key is present
   after `--symbols`; the text view simply emits no symbol lines.
2. **`Lossy` and `Error` are mutually exclusive** and both are `omitempty` bools/strings; a file that
   could not be read is still listed with `Error` set and no header.
3. **`SigEnd` of zero means "no body"**, not "line 0" — every real line number is 1-based. The text
   view's `(sig …)` group keys off exactly this.
4. **Headers and package docs are already first-paragraph-truncated** by the engine
   (`headers.go`, `golang.go:PackageDoc` both end in `FirstParagraph`). Neither view truncates
   further — "complete, never truncated by extraction" is the engine's promise and the views keep it.
5. **Prose contains `<`, `>`, `&` and `—`.** JSON rendering must disable HTML escaping or the output
   is unreadable and unequal to §4.
6. **The engine's `resolveTarget` uses `os.Lstat`, never `os.Stat`** — a symlink named as the target
   is answered as a file, not followed. The CLI's path conversion must not resolve symlinks either
   (`filepath.Abs` + `filepath.Rel`, not `filepath.EvalSymlinks`), or it would defeat that rule.
7. **`.gitignore` already reserves the layout:** `/quarry` ignores the built binary while `!/quarry/`
   keeps the package directory tracked. No gitignore change is needed and none should be made.

**Existing patterns to follow rather than reinvent:**

- `internal/engine/loomyard_test.go` — the `LADDER_LOOMYARD_REPO` gate: skip when unset or absent,
  **fail** when present but at the wrong commit, `loomyardPin = "72c23d9"`.
- `internal/engine/golden_test.go` — the `-update` flag and the compare-or-rewrite helper.
- `internal/engine/answer.go`'s comment discipline — every field's doc says why it is shaped as it
  is, not what it is. New public types in `quarry/` are held to the same standard.

**Where things live after this task:**

```
quarry/            facade: aliases, Open, TOC, RenderJSON, RenderErrorJSON, RenderText
cmd/quarry/        main only: os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
internal/cli/      Run, flag parsing, root discovery, target conversion, exit-code mapping
docs/research/output-formats/after/   the four outputs + INDEX.md, doubling as goldens
```

## Constraints

There is no `CONSTRAINTS.md` at the hub root. The constraints below come from the task body,
`CLAUDE.md`, and exploration:

- **Go only. No Python.** (`CLAUDE.md`, one line, plus plan §2's "no Python in this repository, no
  exceptions".)
- **No machine paths in any tracked file** — including `after/*.txt` and `after/INDEX.md`. The
  before side violates this in `definition-ambiguous.txt`; the after side must not. The invocation
  lines are written as `$ quarry toc internal/logger`, with the checkout identified by name and pin
  in `INDEX.md` only.
- **The envelope is fixed by plan §4**, not by T4 and not by this task. Do not read T4's in-flight
  artifacts; T4 and T5a are deliberately independent.
- **`go build ./... && go test ./...` green.**
- **`CGO_ENABLED=0` build still green outside the engine.** `glyph/` must stay cgo-free — T1 already
  asserts this and that assertion must keep passing. `quarry/` and `cmd/quarry` both pull in the
  engine and therefore cgo; `internal/cgoguard`'s `!cgo` file is what makes that failure readable,
  and it must not be weakened. Do not add a `//go:build cgo` tag to the facade to "fix" a
  `CGO_ENABLED=0 go build ./...` — that would hide the guard.
- **No new module dependencies.** `go.mod` gains nothing.
- **T5b must not have to redo anything here.** Concretely: the renderers take a `DirAnswer` and will
  need siblings, not rewrites, for `resolve` and `expand` answers; the exit-code table and the
  failure envelope are verb-independent and T5b reuses them unchanged; `internal/cli.Run` dispatches
  on a verb word so a second verb is an added case, not a restructure.
- **Review rounds are capped at 3** for the discussion and plan holistic loops
  (`.millhouse/config.local.yaml`, operator decision 2026-09-04).

## Testing

**`internal/cli` — the bulk of the new tests, all Loomyard-free, all table-driven.** These are the
TDD candidates: every one of them can be written before the code, because the contract above fixes
the answer.

- *Flag parsing.* `--depth` accepting `0`, `3`, `all`; rejecting `-1`, `x`, empty. `--symbols` and
  `--no-symbols` producing the three `*bool` states and rejecting the pair together. `--text`.
  Unknown flag, missing target, two targets, missing verb, unknown verb — each an exit-2 usage error.
- *Exit-code mapping.* A table from a returned `error` to the code: wrapped `ErrTargetNotFound` → 1,
  wrapped `ErrTargetOutsideRepo` → 1, an arbitrary error → 3, a pre-engine validation failure → 2,
  `nil` → 0. This is where the `errors.Is`-through-wrapping behaviour is pinned.
- *Failure envelope.* For each non-zero code: stdout is exactly `{"ok": false, "error": …}` plus a
  newline, stderr is non-empty, and the two never swap. The same assertion is repeated **with
  `--text`** to pin that the failure envelope stays JSON regardless of the view flag.
- *Root discovery.* On a synthesised `t.TempDir()` tree: `.git` as a directory found from a nested
  subdirectory; `.git` as a *file* (the worktree case) also found; nothing found walking to the
  filesystem root → exit 2 naming `--root`; `--root` short-circuiting the walk; `--root` at a
  non-directory → exit 2.
- *Target conversion.* cwd-relative and absolute inputs converted to the same repository-relative
  path; **a relative target rebased against `--root` when `--root` is given, and against cwd when it
  is not** — the case the round-2 review caught, so it gets its own explicit test; `..` escaping the
  root → exit 1; native separators accepted; a symlinked target not resolved through.
- *`--help`.* `--help` and `-h`, with and without a verb and at any argument position, write usage to
  stdout and return 0 — never stderr, never 2.

**`quarry` — the renderers, over hand-built `DirAnswer` values, no filesystem at all.** The second
TDD block:

- *Text view, directory form.* A directory with a package and a doc; one with neither; one with
  `1 file` vs `N files`; a depth-cut subdirectory carrying only identity and doc; nested blocks in
  depth-first order with exactly one blank line between them.
- *Text view, tags.* Each of `test`, `generated`, `package`, `language`, `lossy`, `error` alone, and
  all of them together in the fixed order; plus a multi-line `error` value collapsing to one line,
  so the one-record-per-line property is pinned against the one field quarry does not author.
- *Text view, file form at the repository root.* A file target whose answer's `dir` is `"."` emits
  the bare name, not `./name`.
- *Text view, symbols.* A symbol with and without a doc; one with `sigend` zero (no `(sig …)`
  group); a file entry with a nil `Symbols` and one with an empty non-nil `Symbols` — both must
  render identically, since the distinction is a Go-caller concern.
- *Text view, file form.* §4's file shape, and the assertion that it is chosen by the `targetIsFile`
  argument rather than by the answer's shape — the same one-file answer rendered both ways.
- *Prose normalisation.* A doc containing `\n` collapsing to a single space; runs of whitespace
  collapsing; nothing truncated. Plan §4's own `placement` example is the fixture to use, since it
  is the one case the plan itself shows both sides of.
- *JSON view.* Key order matches struct order; absent fields absent (no `test: false`, no
  `"dirs": []`); no `ok` key on success; `<`, `>`, `&` unescaped; two-space indent; exactly one
  trailing newline.

**Golden tests — the `after/` files.** Gated on `LADDER_LOOMYARD_REPO` at `72c23d9`, skipping when
the checkout is absent and failing loudly when it is present at the wrong commit. Four cases, one
per `after/*.txt`, each running `cli.Run` in-process with `--root` and a repository-relative target,
asserting exit 0, and comparing the assembled file (invocation line, blank line, output) byte for
byte; `-update` rewrites. These are *not* TDD
candidates — a golden can only be produced by running the code against the pinned checkout, never
hand-written, exactly as `golden_test.go` already says.

**The plan §4 byte-for-byte check.** One additional gated case asserting `quarry toc
internal/reedengine/render/layout.go` produces the JSON of §4's file example, modulo the prose the
plan abridges with `...`. T3 already pins this at the engine level; repeating it at the CLI level is
what proves the rendering layer added nothing and dropped nothing.

**Build checks.** `go build ./... && go test ./...` with cgo, plus a documented
`CGO_ENABLED=0 go build ./glyph/...` staying green and the existing T1 cgo-free assertion on
`glyph/` still passing.

## Q&A log

- **Q:** How does the facade expose the envelope types, given the engine is under `internal/`? **A:** [auto-pick] Type aliases in `quarry/` to the engine's types. **Why:** an alias to an internal type is usable by an external importer, so Loomyard gets a nameable type with zero duplication and no second place the envelope can drift.
- **Q:** Where do the JSON and text renderers live? **A:** [auto-pick] Exported from the facade package `quarry/`. **Why:** plan §12 requires T6's MCP to be thin and to sit over the facade; renderers in `cmd/quarry` would force T6 to re-implement the envelope.
- **Q:** What shape is the facade's entry point? **A:** [auto-pick] `Open(root) (*Repo, error)` plus `(*Repo).TOC(target, TOCOptions) (DirAnswer, error)`, mirroring the engine. **Why:** the facade's job is to be a public name for the engine, not a second design.
- **Q:** How is the CLI laid out? **A:** [auto-pick] `cmd/quarry` main plus `internal/cli` with `Run(args, stdout, stderr) int`. **Why:** the standard testable-CLI pattern; goldens capture the binary's exact bytes with no build or exec.
- **Q:** May T5a change `internal/engine`? **A:** [auto-pick] No; T5a is purely additive. **Why:** T3's goldens pin the engine byte for byte and T4 is in flight against the same package in parallel.
- **Q:** Where does `ok` live in the envelope? **A:** [auto-pick] Absent on success, `{"ok": false, "error": …}` on failure. **Why:** plan §4 names "`ok: true` inside data" as V1 clutter *and* requires `ok` to agree with the exit code *and* shows examples with no `ok` key; only present-only-when-false satisfies all three.
- **Q:** What is the exit-code scheme? **A:** [auto-pick] 0 answered, 1 negative answer, 2 usage error, 3 internal error. **Why:** a gate must distinguish "quarry says no" from "the caller asked wrong" from "the disk failed"; conflating them makes wrapper scripts unable to tell a real negative from their own bug.
- **Q:** Does the failure JSON go to stdout or stderr? **A:** [auto-pick] Always stdout, with the human message on stderr. **Why:** a pipeline that parses stdout must find a parseable envelope on every path, or the exit code becomes the only signal again — the V1 defect.
- **Q:** How many targets does one `toc` invocation take? **A:** [auto-pick] Exactly one. **Why:** plan §4 defines one answer shape and no multi-answer envelope; every plural shape invents one. The three rejected alternatives — always-array, arity-dependent, and common-ancestor spine — are recorded under `single-target` so a later task can choose deliberately.
- **Q:** Does the failure payload carry a machine-readable `kind`? **A:** [auto-pick] No, `ok` and `error` only. **Why:** the exit code already discriminates; duplicating it inside the payload is the redundancy §4's "shared facts once" rule bans.
- **Q:** What is the binary and invocation form? **A:** [auto-pick] `cmd/quarry`, invoked `quarry toc <target> [flags]`. **Why:** `.gitignore` already reserves `/quarry` as a repo-root binary while keeping `quarry/` tracked, so the name is settled; one verb with five flags does not justify a CLI framework dependency.
- **Q:** Are targets interpreted relative to cwd or to the repository root? **A:** [auto-pick] cwd-relative on input, repository-root-relative on output. **Why:** `quarry toc .` inside a subdirectory must mean that subdirectory, as in git and rg; keeping the two path frames separate stops them mixing.
- **Q:** How is the repository root found? **A:** [auto-pick] Walk up from cwd for a `.git` entry (file or directory), with `--root` overriding. **Why:** the engine deliberately does no git discovery and no cwd resolution, so the CLI is the only layer that can; a `.git` *file* must count or worktrees break.
- **Q:** How does the CLI express `TOCOptions.Symbols`'s three states? **A:** [auto-pick] `--symbols` true, `--no-symbols` false, neither nil. **Why:** the `*bool` exists precisely so "not asked" differs from "asked for false"; `-symbols=false` is an undiscoverable spelling for a documented behaviour.
- **Q:** How is the text view selected? **A:** [auto-pick] A `--text` boolean, JSON by default. **Why:** exactly two views exist and plan §10 forbids a third; `--format` would advertise extensibility that is a stated non-goal.
- **Q:** How do nested directories render in the text view? **A:** [auto-pick] One block per directory answer, depth-first, blank-line separated, each headed by its full repository-relative path. **Why:** deep trees indent off the right edge, and a full path on the heading line stays copy-pasteable as the next invocation's argument.
- **Q:** How does a file target render in the text view? **A:** [auto-pick] Plan §4's file form exactly — path plus package facts on one line, then symbol lines, no directory doc. **Why:** it is one of only two text examples the plan gives, and the done-when is byte-for-byte where examples exist.
- **Q:** How does the text view mark deviations (test, generated, package, language, lossy, error)? **A:** [auto-pick] Bracketed tags after the name, in a fixed order, only when present. **Why:** it is the before side's own compact-view convention with parentheses regularised to brackets, and §4 says the text view has no keys.
- **Q:** Must the text view be parseable back into a `DirAnswer`? **A:** [auto-pick] No — lossless means no datum dropped; JSON stays the contract. **Why:** §4 states JSON is the contract and that the text view's value is an unmeasured ladder question; a parser would force escaping into a view whose point is unescaped prose.
- **Q:** What happens to internal newlines in `doc`, `header` and `signature` in the text view? **A:** [auto-pick] Collapsed to single spaces, nothing truncated. **Why:** §4's own worked example prints a `doc` containing `\n` as one line, so "prose intact" means complete, not line-break-preserving.
- **Q:** How are absent symbol fields rendered? **A:** [auto-pick] Omit the indented doc line entirely; omit the `(sig a-b)` group when `sigend` is zero. **Why:** "defaults never" applies to both views, and the engine documents zero `sigend` as the no-body marker, not a line number.
- **Q:** Where do the golden fixtures live? **A:** [auto-pick] The `after/` files *are* the goldens, gated on `LADDER_LOOMYARD_REPO` at pin `72c23d9` with `-update` regenerating them. **Why:** one artifact means the committed evidence and the regression gate cannot disagree, and it reuses the pattern T3 already established.
- **Q:** How does the golden test invoke the CLI? **A:** [auto-pick] `cli.Run` in-process with an explicit `--root`. **Why:** it covers parsing, engine call, rendering and exit code in one assertion at unit-test speed, with no build step and no process-global `t.Chdir`.
- **Q:** What files does `after/` contain? **A:** [auto-pick] `toc-dir.txt`, `toc-file.txt`, `toc-dir-text.txt`, `toc-file-text.txt`, and `INDEX.md`. **Why:** the before side's two `toc` commands times the two views; `INDEX.md` records that the lossy `-compact` files have no successor by design, which is plan §1's lesson 4.
- **Q:** What format do the `after/*.txt` files use? **A:** [auto-pick] The before side's: `$ …` invocation line, blank line, output, blank line, `(exit code: N)`. **Why:** the pair is only readable as before/after if both sides are laid out the same way.
- **Q:** Are there tests beyond the goldens? **A:** [auto-pick] Yes — Loomyard-free table tests for flags, exit codes, root discovery, target conversion, and both renderers over hand-built answers. **Why:** the goldens need a checkout most machines lack, so the contract must be pinned by tests that always run.
- **Q:** How is the `CGO_ENABLED=0` constraint verified? **A:** [auto-pick] `glyph/` stays cgo-free via T1's existing assertion; the facade and CLI are expected to need cgo and `internal/cgoguard` keeps that failure readable. **Why:** the task's done-when says exactly this; adding a build tag to the facade to make `CGO_ENABLED=0 go build ./...` pass would hide the guard.
- **Q:** Does the facade need to do anything for "grammars loaded once per process"? **A:** [auto-pick] No — already satisfied; add no cache and no parser pool. **Why:** the Go grammar is a linked-in static and `ts.NewLanguage` only wraps its pointer; plan §10 forbids a phase-1 cache and §5 concludes nothing is worth optimising yet.
- **Q:** Is the facade's `*Repo` safe for concurrent use? **A:** [auto-pick] Yes, and documented so. **Why:** it holds only the root string and every query reads the filesystem fresh, which is exactly what `engine.Repo` already documents about itself.
- **Q:** Does `Symbol.file` get filled inside a toc answer? **A:** [auto-pick] No — it stays omitted, as the engine already does. **Why:** the symbol sits inside its file's own entry; §4 says `file` is carried only "wherever entries can span files", which is T5b's `resolve` and `expand`, not `toc`.
- **Q:** How are path separators handled? **A:** [auto-pick] Forward slashes everywhere on output; native separators accepted on input. **Why:** the engine already emits forward-slash repository-relative paths, and rejecting native separators on input would make the CLI unusable on Windows for no gain.
