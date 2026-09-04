# Discussion: Facade + CLI, resolve + expand (T5b)

```yaml
task: Facade + CLI, resolve + expand (T5b)
slug: facade-cli-resolve-expand
status: discussing
parent: main
```

## Problem

`resolve` and `expand` exist in the engine — T4 merged them as `(*engine.Repo).Resolve` and
`(*engine.Repo).Expand` — but nothing outside `internal/engine` can call them. `internal/` is
compiler-enforced private, so today the two verbs have no Go caller and no command line. T5a built
the whole public surface for `toc`: the `quarry/` facade, the JSON envelope of `docs/rewrite-plan.md`
§4, the lossless text view, the four exit codes, and the `docs/research/output-formats/after/`
evidence directory. This task gives `resolve` and `expand` that same surface.

**Why now:** plan §12 row T5b, wave 4. Its dependencies (T4 and T5a) are both merged on `main`, so
the code this task extends is real code, not artifacts. It is deliberately off the critical path —
T6's MCP exposes only `toc` — so it runs in parallel with T6 and blocks only Loomyard's own adoption
of glyphs (plan §12's closing note: that work starts in Loomyard's repository after T5b merges).

The task's own hard constraint is that **nothing new is invented**. Every envelope, exit-code and
view question is supposed to be answered already by plan §4 or by T5a's merged code. Exploration
found that this holds for most of the surface and does *not* hold for three questions, which are
recorded below as Decisions D2, D3 and D10 and flagged for the round-1 reviewer.

## Scope

**In:**

- `quarry/` (the facade): type aliases for `ResolveResult`, `ExpandAnswer`, `Status` and
  `NotATypeError`; the four `Status` constants; `(*Repo).Resolve` and `(*Repo).Expand` delegating to
  the engine unchanged; four new renderers (`RenderResolveJSON`, `RenderExpandJSON`,
  `RenderResolveText`, `RenderExpandText`).
- `internal/cli/`: two new verbs, `resolve` and `expand`; per-verb flag validation; per-verb target
  handling (glyph targets bypass all path arithmetic and all stat'ing); the extended exit-code
  mapping; the extended `usageText`.
- `docs/research/output-formats/after/`: eight new golden files and a rewritten `INDEX.md` that
  maps every before-side file to its successor or states why it has none.
- `internal/cli/after_test.go`: the golden table gains the eight new rows and an expected-exit-code
  column.
- A new machine-independent CLI test covering the four `docs/glyph.md` §5 statuses end to end.
- `quarry/doc.go` and `internal/cli/doc.go` updated to describe the widened surface.

**Out:**

- **`internal/engine/` is not modified.** Every file under it stays byte-identical. Both verbs are
  complete in the engine; this task is a surface, and any engine change would mean T4 shipped
  incomplete.
- **`cmd/quarry-mcp` and anything MCP.** T6 owns it, and plan §12 says the MCP exposes neither verb
  until a ladder cell measures it.
- **`glyph/`.** The grammar is T1's and is used through `glyph.Parse` inside the engine only; the CLI
  never parses a glyph itself.
- **`toc`'s behaviour.** Not one byte of `toc`'s output, exit codes or goldens changes. Where a
  helper is refactored to be shared (D8), the refactor is behaviour-preserving and `toc`'s four
  existing goldens must still compare equal without regeneration.
- **`RenderJSON`, `RenderErrorJSON`, `RenderText` signatures.** They keep their exact current
  signatures and byte behaviour; T6 is being written against them in parallel.
- **Multi-target `resolve` on the command line** (D1), **batching, caching or parallelism** (plan §10
  phase-1 non-goals), **`impact`, `refs`, `assert-no-callers`** (phase 2 or dropped, plan §5).
- **`docs/rewrite-plan.md`.** Not edited. Amending §5's variadic `resolve` bullet to match D1 is an
  operator follow-up on `main`, exactly as `toc`'s equivalent amendment was (commit `55161a6`).
- **Anything in Loomyard's repository.**

## Decisions

### D1 — the CLI takes exactly one target per call; the facade keeps the engine's multi-target `Resolve`

- **Decision:** `quarry resolve <target>` and `quarry expand <target>` each take exactly one
  positional target; two or more is a usage error (exit 2). The facade's method keeps the engine's
  signature, `Resolve(targets []string) ([]ResolveResult, error)`, so a Go caller batches many glyphs
  in one call and pays one parse per unit.
- **Rationale:** the task body states it outright ("One target per call, as `toc` settled (plan §5)").
  It is also the only reading that invents nothing: with N targets and mixed statuses — one `found`,
  one `not_found` — a single process exit code has to summarise N answers, and no rule anywhere says
  how. Inventing an aggregate rule is exactly what the task forbids. T5a settled the identical
  question for `toc` on the identical argument ("one invocation, one answer, one exit code", now
  quoted in plan §5). Plan §5's own multi-glyph batching ("Many glyphs in one call are grouped by
  unit and each unit is parsed once") is a *performance* property of the resolution engine, and it
  survives intact on the facade — which plan §7 names the primary surface and §8.2 names as the
  interface Loomyard's mechanical, LLM-free code uses. The CLI is the mirror, not the primary.
- **Rejected:** a variadic CLI matching plan §5's `resolve <glyph|path>...` literally. It would
  force a new aggregate-exit-code rule, and it would make the JSON payload a top-level array where
  every other quarry command emits an object.
- **Tension, and its disposition:** plan §5's `resolve` bullet spells the variadic form and is not
  amended anywhere; the task body's "(plan §5)" citation points at the `toc` sentence, not the
  `resolve` one. Round 1's review confirmed single-target and closed the tension: **amending plan §5's
  `resolve` bullet is an operator follow-up on `main`, not a change this worktree makes** — the same
  move `toc`'s own one-target amendment took (commit `55161a6`, "Plan §5: toc takes one target per
  call, as T5a's discussion settled"), which landed on `main` outside the task that decided it. This
  task edits no line of `docs/rewrite-plan.md`.

### D2 — a negative answer renders the payload, not the error envelope

- **Decision:** when `Resolve` or `Expand` returns a value with a nil error, the CLI renders that
  value with the selected renderer and exits with the code D3 assigns — including for
  `status: not_found`, `status: ambiguous`, and a `ResolveResult` carrying a pre-resolution `error`.
  `fail()` and `RenderErrorJSON` are used only when there is no payload at all: usage errors,
  internal errors, `expand`'s `*NotATypeError` (D4), and `expand`'s exit-1 grammar rejection (D10a).
  Those four are the complete set of payload-free paths; D3's table is the authority and this list
  must be read as tracking it, never as narrowing it.
- **Rationale:** `toc` uses the error envelope on exit 1 because `toc` has no negative payload — the
  engine returns an error, not an answer. `resolve` and `expand` are the opposite case by
  construction: T4 deliberately moved these dispositions *into* the payload
  (`internal/engine/resolve.go`'s `resolveGlyphTarget`, `resolvePathTarget`). Collapsing them into
  `{"ok":false,"error":"..."}` would destroy `unit`, `candidates` and `reason` — the three fields
  plan §5 and §8.1 name as the whole reason the validator asks. Plan §4's rule is "`ok` agrees with
  the exit code"; on this path there is no `ok` key at all, and `status` is the discriminator plan §4
  itself assigns ("`status` per entry: `found`, `not_found`, `ambiguous` (with `candidates`),
  `multipart`"). The absence of `ok` is not a claim of success, so nothing disagrees.
- **Rejected:** (a) error envelope on every nonzero exit — loses the answer, and would make
  `resolve`'s exit-1 output indistinguishable from `toc`'s, which is precisely the V1 flattening the
  rewrite exists to undo; (b) exit 0 for every payload-bearing answer — breaks plan §4's
  ok/exit agreement and contradicts `toc`, where a missing target is exit 1.
- **Consequence to state in `quarry/doc.go` and `internal/cli/doc.go`:** the `ok` key marks
  *"quarry could not answer"*, never *"the answer is negative"*. T5a's `render.go` comment — "a
  caller can tell success from failure by this key's presence alone" — stays true of what it
  describes (failure to answer) and must be reworded to say so, since it is now one of two verbs'
  worth of context.
- **Flagged for review:** this is the largest extension of T5a's machinery in the task.

### D3 — the exit-code table for the two verbs

- **Decision:** the four codes keep their existing meanings (`internal/cli/cli.go`'s constants are
  unchanged) and map as follows.

  | verb | condition | code | stdout |
  |---|---|---|---|
  | `resolve` | `status: found` | 0 | the `ResolveResult` |
  | `resolve` | `status: multipart` | 0 | the `ResolveResult` |
  | `resolve` | `status: not_found` (glyph or path) | 1 | the `ResolveResult` |
  | `resolve` | `status: ambiguous` | 1 | the `ResolveResult` |
  | `resolve` | pre-resolution rejection: `error` set, `status` absent (glyph grammar rejection, or path outside the repository) | 1 | the `ResolveResult` |
  | `expand` | `status: found` | 0 | the `ExpandAnswer` |
  | `expand` | `status: not_found` | 1 | the `ExpandAnswer` |
  | `expand` | `status: ambiguous` | 1 | the `ExpandAnswer` |
  | `expand` | `*NotATypeError` | 1 | the error envelope (D4) |
  | both | unparseable flag, wrong target count, unknown verb, toc-only flag on this verb, `--root` that is not a directory, `discoverRoot` finding no `.git` above the cwd (`no repository root found above <dir>; pass --root`) | 2 | the error envelope + `usageText` on stderr |
  | `resolve` | `repoRelPath` itself erroring (`filepath.Rel` failure — a different volume on Windows, in practice) | 1 | the error envelope, message `target outside repository: <target>` |
  | `expand` | target containing no `#` (D10) | 2 | the error envelope + `usageText` on stderr |
  | `expand` | `glyph.Parse` rejection of a target that *does* contain `#` (`#x`, `member_keyword`, `unit_bad_rune`, …) | 1 | the error envelope, message `expand <target>: <reason>` (D10) |
  | `expand` | the `HeadStart == 0` invariant error (a matched `KindType` symbol carrying no head span) | 3 | the error envelope |
  | both | any remaining error: engine read failure, `len(results) != 1`, render failure, stdout write failure | 3 | the error envelope |

  The rows are checked in table order and the final row is the catch-all: a mapping function written
  from this table must test the named conditions first and fall through to 3, never the reverse. The
  `HeadStart == 0` row is exit 3 because `internal/engine/expand.go` returns it as a plain
  `fmt.Errorf` naming an invariant violation in the walk — an internal failure with no answer behind
  it, which is exactly what exit 3 means — and it is reached with `errors.As` failing for both
  `*NotATypeError` and `*glyph.ParseError`, so no message parsing is needed to tell it apart.
  The `repoRelPath` row is exit 1 with `toc`'s own message for the same condition, byte for byte:
  the target cannot be expressed relative to the root at all, so there is no relative form to hand
  the engine and therefore no payload to render — D2's payload rule cannot apply, and diverging from
  `toc` here would make the same argument exit differently under two verbs. It is unreachable on a
  single-volume POSIX filesystem and is stated so that a mapping written from this table cannot
  silently route it to 3.

- **Rationale:** exit 2 keeps its T5a meaning exactly — "the caller asked wrong about the CLI" — and
  a well-formed invocation naming an unspellable glyph is not that: quarry ran it to a definite
  conclusion and has a payload with a `reason` word to show for it. Exit 1 keeps its T5a meaning
  exactly — "the invocation was well formed and the CLI ran it to a definite, negative conclusion".
  `ambiguous` is a negative conclusion in that sense: nothing was chosen, so no caller can act on it
  without asking again. Putting `ambiguous` on exit 2 would repeat V1's `definition` defect in mirror
  image (`docs/research/output-formats/definition-ambiguous.txt`: `ok: true`, exit 2) — a well-formed
  invocation reported as a usage error.
- **Rejected:** a grammar rejection as exit 2. It would print `usageText` after a message about glyph
  syntax, which is noise, and `fail(..., withUsage: true)` would suppress the `reason` word the
  payload carries.
- **Flagged for review:** the `ambiguous → 1` and `grammar rejection → 1` rows. Neither is written
  down anywhere; both are derived from T5a's own stated meanings for codes 1 and 2.

### D4 — `expand` on a non-type: the error envelope, exit 1, quarry's own sentence

- **Decision:** `Expand` returning `*NotATypeError` produces `fail(stdout, stderr, exitNegative,
  "expand "+e.ID+": not a type, kind "+string(e.Kind), false)` — the error envelope on stdout, the
  same sentence on stderr, no usage text, exit 1. The CLI reaches the fields with
  `errors.As(err, &notType)` against `*quarry.NotATypeError`, never by parsing the message.
- **Rationale:** plan §5 says "The glyph must name a type; on any other kind the answer is
  `ok: false` naming the kind" — `ok: false` is the error envelope, by name. There is no payload
  here: `Expand` returns the zero `ExpandAnswer` alongside the error, so D2's payload rule does not
  apply and the envelope is the only thing there is to render. The sentence is quarry's own, not
  `err.Error()`, because T5a's rule (`internal/cli/cli.go`'s `Run` doc comment) forbids leaking the
  engine's `engine: ` prefix through an exit-1 or exit-2 message.
- **Rejected:** exit 3 (this is not an I/O failure); exit 2 (the invocation is well formed);
  `err.Error()` passed through (leaks `engine:`).

### D5 — the facade's new surface, exactly

- **Decision:** `quarry/quarry.go` gains four aliases and four constants; `quarry/repo.go` gains two
  methods; two new files hold the new renderers.

  ```go
  // quarry/quarry.go
  type ResolveResult = engine.ResolveResult
  type ExpandAnswer  = engine.ExpandAnswer
  type Status        = engine.Status
  type NotATypeError = engine.NotATypeError

  const (
      StatusFound     = engine.StatusFound
      StatusNotFound  = engine.StatusNotFound
      StatusAmbiguous = engine.StatusAmbiguous
      StatusMultipart = engine.StatusMultipart
  )

  // quarry/repo.go
  func (r *Repo) Resolve(targets []string) ([]ResolveResult, error)
  func (r *Repo) Expand(target string) (ExpandAnswer, error)
  ```

  Both methods return the engine's own value and the engine's own error unchanged, exactly as
  `(*Repo).TOC` does, so `errors.As(err, &notType)` against `*quarry.NotATypeError` succeeds for a
  caller that never imports the engine — the same transitivity argument `quarry.go`'s existing
  sentinel comments make for `errors.Is`.
- **Rationale:** the aliases exist for one reason, stated in `quarry/quarry.go`'s own header: Go
  enforces the `internal/` rule on import paths, not on types reached through an alias. The four
  `Status` constants are needed for the same reason the five `Kind` constants are — a caller must be
  able to switch on the status without importing the engine.
- **Rejected:** wrapping the engine types in facade-owned structs (would need a converter per type,
  and a second place the key set could drift from plan §4); adding a facade-level "resolve one
  target" convenience (the CLI's one-target rule is the CLI's, not the facade's — D1).

### D6 — four new renderers, one shared unexported encoder, existing signatures untouched

- **Decision:**

  ```go
  func RenderResolveJSON(r ResolveResult) ([]byte, error)
  func RenderExpandJSON(a ExpandAnswer) ([]byte, error)
  func RenderResolveText(r ResolveResult) string
  func RenderExpandText(a ExpandAnswer) string
  ```

  `RenderJSON`, `RenderErrorJSON` and `RenderText` keep their current signatures and bytes.
  `RenderJSON`, `RenderResolveJSON` and `RenderExpandJSON` all delegate to one new unexported
  `renderJSON(v any) ([]byte, error)` holding the encoder configuration T5a fixed (escape-HTML off,
  two-space indent, exactly one trailing newline).
- **Rationale:** three exported functions that each configure an encoder identically are three places
  the byte contract can drift; one is one. The CLI renders a single `ResolveResult`, never the slice,
  because D1 makes one invocation one answer and every other quarry command emits a JSON object
  rather than an array. No slice renderer is written: nothing needs one, and plan §10's non-goals and
  the repository's YAGNI rule both say not to build it.
- **Rejected:** one generic exported `RenderJSON(v any)` — it would change T5a's signature while T6
  is being written against it, and it would let any type be handed to a renderer contracted to emit
  plan §4's key set.

### D7 — the text-view grammar for `resolve` and `expand`, in full

The grammar below is fixed to the character, the way T5a fixed `toc`'s. Every rule it does not state
is inherited from `quarry/text.go`: `normalizeProse` (collapse internal whitespace runs to single
spaces) is applied to every `Signature`, `Doc` and `Error`; no line carries trailing whitespace; the
returned string ends with exactly one `"\n"`; no keys, no defaults, prose intact.

**The symbol line is `toc`'s symbol line, with a file prefix.** `writeSymbolLine` gains a leading
`sym.File + ":"` emitted only when `sym.File != ""`. Inside a `toc` answer `File` is always empty
(`internal/engine/answer.go` states this: the symbol already sits in its file's entry), so every
existing `toc` golden is byte-identical and this is one grammar with one implementation rather than
two. `resolve` and `expand` symbols always carry `File`, so their lines read:

```
<file>:<start>-<end>[ (sig <start>-<sigend>)] <id>: <signature>
[    <doc>]
```

**`resolve`, glyph target.** Line 1 is `<ID> <status>`. Then:

- `found`, `multipart` — one symbol line per `Symbols` entry, in order.
- `ambiguous` — one symbol line per `Candidates` entry, in order.
- `not_found` — line 1 is instead `<ID> not_found (unit found)` or `<ID> not_found (unit not_found)`,
  and nothing follows. The parenthesised clause is always present on a glyph `not_found`, because the
  engine always sets `Unit` there.

**`resolve`, glyph target rejected before resolution** (`Error` set, `Status` empty). One line, and
`ID` is empty here so the line names the target as given:

```
<Target> error <Reason>: <normalizeProse(Error)>
```

` <Reason>` and its leading space are omitted when `Reason` is empty.

**`resolve`, path target.** Line 1 is `<Target> <status>`. Then:

- `found` — the directory-form block for `*Dir`, produced by the same `dirBlocks` join
  `RenderText(a, false)` uses, starting on the line immediately after line 1.
- `not_found` — nothing follows. No unit clause: a path belongs to no unit, and the engine never sets
  `Unit` on a path result.
- outside the repository (`Error` set, `Status` empty, `Reason` empty) — the same one-line error form
  as above, which degenerates to `<Target> error: <msg>`.

  The directory form is used for a path result even when the target is a file, and no `targetIsFile`
  is plumbed in: `resolvePathTarget` calls `TOC` with `Depth: 0` and symbols off, so a file target's
  answer is its enclosing directory's answer holding exactly one file entry, and the directory form
  renders that losslessly. This is also what lets the CLI drop `toc`'s `os.Lstat` step entirely (D9).

**`expand`.** Line 1 is `<ID> <status>`. Then:

- `found` — the head's symbol line; then, when `Members` is non-empty, one blank line, then one
  symbol line per member in order. The blank line is the same block separator `dirBlocks` already
  uses, so no new marker is invented, and head and members stay distinguishable without a key.
- `ambiguous` — one symbol line per `Candidates` entry, in order. No blank line.
- `not_found` — line 1 is `<ID> not_found (unit found)` or `<ID> not_found (unit not_found)`, and
  nothing follows.

There is no text rendering of `expand`'s `*NotATypeError`: it takes the error path (D4), and T5a's
rule already says there is no text rendering of a failure.

### D8 — target handling: a glyph bypasses all path arithmetic and all stat'ing

- **Decision:** the CLI decides glyph-versus-path with `strings.Contains(target, "#")` and nothing
  else — its own copy of the engine's unexported `isGlyphTarget` rule, since the engine's is not
  reachable. A glyph target is passed to the facade **verbatim**: no `repoRelTarget`, no `Lstat`, no
  `--root` rebasing. A glyph's unit is repository-relative by the grammar's own definition
  (`docs/glyph.md` §2), so cwd arithmetic on it would corrupt it.

  A path target for `resolve` goes through the *arithmetic* half of `repoRelTarget` only.
  `repoRelTarget` is split into two functions, behaviour-preserving for `toc`:

  - `repoRelPath(root, base, target) (string, error)` — the existing arithmetic, returning the
    cleaned forward-slash relative form even when it begins with `..`, and erroring only on
    `filepath.Rel`'s own failure.
  - `repoRelTarget(root, base, target) (string, error)` — calls `repoRelPath` and adds the existing
    `ErrTargetOutsideRepo` rejection. `toc` calls this one and behaves exactly as today.

  `resolve` calls `repoRelPath` and passes the result through unconditionally. A path that escapes
  the root arrives at the engine as a leading-`..` relative path, `resolveTarget` returns
  `ErrTargetOutsideRepo`, and `resolvePathTarget` turns that into the payload's `error` field —
  which is the answer plan §5 wants, produced by the one implementation that owns the rule.
- **Rationale:** the alternative is for the CLI to synthesise a `ResolveResult{Target: …, Error: …}`
  itself when its own arithmetic rejects the path, which would be a second implementation of the
  outside-repo disposition and would have to spell the engine's message to stay consistent. Letting
  the engine own it is the same "reuse rather than restate" argument `resolvePathTarget`'s own doc
  comment makes about reusing `TOC`.
- **Rejected:** teaching `repoRelTarget` a boolean mode flag (a flag argument that changes a
  function's contract, for two call sites, when two named functions read better).
- **What the payload's `target` echoes, and it differs by target class.** `ResolveResult.Target` is
  "the caller's argument, verbatim" from the *engine's* point of view — that is, whatever string the
  CLI handed it. So:
  - **a glyph target** echoes argv verbatim, because the CLI passes it through untouched;
  - **a path target** echoes the **repository-relative form**, not argv, because that is what the CLI
    hands the engine. `quarry resolve logger.go` run inside `internal/logger/` answers
    `"target": "internal/logger/logger.go"`.

  This is the correct behaviour, not a leak: `internal/cli/doc.go` already states the rule as "input
  is interpreted where the user is, output is always repository-root relative", and plan §4 requires
  relative paths and forbids absolute ones. It is written down here because D7's text form puts
  `<Target>` on line 1, so the same rebasing is visible in both views and every golden depends on it.
- **Every `after/` golden and every example in this file is written with cwd == the repository root**
  (in the goldens' case, guaranteed by `--root`), so argv and the repository-relative form coincide
  and the recorded invocation line reads the way a user at the root would type it. D10b's
  `resolve ../x` example is under that same assumption.

### D9 — `resolve` performs no `os.Lstat`; `expand` performs no path work at all

- **Decision:** `resolve`'s pipeline omits `toc`'s step 6 entirely. `expand`'s pipeline omits steps 5
  and 6 (it takes a glyph only).
- **Rationale:** `toc` stats for two reasons, and neither survives here. It stats to tell exit 1 from
  exit 3 — but for `resolve`, "the target does not exist" is the engine's own `status: not_found`
  answer with a payload, and pre-empting it with `fail()` would destroy exactly that answer. And it
  stats to compute `targetIsFile` for `RenderText` — which D7 removes the need for.
- **Consequence:** a `resolve` of a nonexistent path answers `{"target":"x","status":"not_found"}`
  and exits 1, where `toc` of the same path answers `{"ok":false,"error":"target not found: x"}` and
  exits 1. Same code, different stdout, and that difference is D2's whole point.

### D10 — `expand` rejects a `#`-less target on the CLI's own check, before any engine call

- **Decision:** `quarry expand internal/logger` (no `#`) is a usage error, exit 2 with `usageText`,
  message `expand takes a glyph (a target containing "#"), got: <target>`. It is decided by the CLI's
  own `strings.Contains(target, "#")` check — the same one D8 already gives the CLI for `resolve`'s
  glyph-versus-path routing — applied at argument-handling time, **before** root discovery,
  `quarry.Open`, or any call into the facade.
- **Rationale for the routing:** T5a's `exitUsage` constant is documented as "TOC is never called on
  this path" (`internal/cli/cli.go`), so a usage error discovered only *after* the engine has run
  would contradict the code the constant is defined in. D8 already spends the "the CLI writes no
  separator check" purity argument by giving the CLI that exact check, so there is no purity left to
  preserve by routing this through the engine, and `errors.As(err, &parseErr) && parseErr.Reason ==
  glyph.ReasonNoSeparator` would be a second, later, more expensive spelling of a decision the CLI
  has already made. **This routing needs no `glyph/` import; the last bullet below does — see D10a.**
- **Rationale for exit 2 rather than 1:** unlike `resolve`, where the same string is a legitimate
  *path* target and the grammar rejection is a real answer about a real argument class, `expand`
  accepts one argument class only. A target with no `#` is the caller having asked the wrong verb,
  which is what exit 2 means.
- **Rejected:** detecting the case post-engine via `errors.As` on the wrapped `*glyph.ParseError`
  (contradicts exit 2's own documented meaning, and duplicates a decision the CLI already made);
  treating it as an ordinary parse rejection on D4's exit-1 envelope path (an invocation naming no
  glyph at all is a different thing from one naming an unspellable glyph).
- **Every other `glyph.Parse` rejection under `expand`** — a target that *does* contain `#` but that
  the grammar still rejects, `#x` or `member_keyword` or `unit_bad_rune` — is unaffected by this
  routing and takes D4's path: error envelope, exit 1, no usage text. It is a well-formed invocation
  naming an unspellable glyph, and the CLI never parses it itself. Its message and how the CLI gets
  the reason word are D10a.

### D10a — `internal/cli` imports `glyph/` to reach the rejection reason

- **Decision:** for `expand`'s exit-1 grammar rejections, the CLI's message is
  `expand <target>: <reason>` where `<reason>` is `string(parseErr.Reason)`, obtained with
  `errors.As(err, &parseErr)` against `*glyph.ParseError`. **`internal/cli` imports `glyph/`
  directly** to name that type. No alias is added to the facade.
- **Rationale:** `Expand` wraps the parse failure as `engine: expand %s: %w`
  (`internal/engine/expand.go`), so `err.Error()` is barred by D4's no-`engine:`-prefix rule and the
  CLI must spell its own sentence — which needs the reason word, which needs the concrete error
  type. `glyph/` is a **public** package of this module, pure Go with no dependencies and no cgo (T1's
  own done-when: `go list -deps` shows no cgo), so `internal/cli` may import it and pays nothing for
  doing so. The facade's aliases exist for exactly one reason, stated in `quarry/quarry.go`'s header:
  `internal/engine` is unreachable from outside the module. `glyph/` is reachable, so aliasing
  `ParseError` and `Reason` into `quarry/` would be a second spelling of the grammar's own error
  type — which `docs/glyph.md` §6's one-implementation-of-the-grammar rule forbids, and which is the
  drift `Symbol` was given its own `Glyph` field to prevent.
- **Rejected:** aliasing `ParseError`/`Reason` into the facade (a second spelling of a public type,
  against `docs/glyph.md` §6); dropping `<reason>` from the message (the reason word is the only part
  of the rejection a caller can act on, and the engine deliberately surfaces it as a closed
  vocabulary — the same word `resolve` puts in its payload's `reason` key, so dropping it here would
  make the two verbs disagree about the same rejection).
- **Consequence:** Technical context's "`glyph/errors.go` — needed by D10" line is correct as
  written, and `internal/cli`'s import block grows by one public, cgo-free package. A future second
  alphabet changes nothing here: the CLI reads `parseErr.Reason` and never calls `glyph.Parse`.

### D10b — the payload's `error` field is engine text, emitted verbatim

- **Decision:** T5a's no-`engine:`-prefix rule binds **the sentence quarry authors** — `fail()`'s
  message, and therefore stdout's error envelope and stderr alike. It does **not** bind
  `ResolveResult.Error`, which is a data field of the answer, populated by the engine, and carried to
  stdout unchanged by the JSON contract. So `quarry resolve ../x` prints
  `"error": "engine: resolve target \"../x\": engine: target outside repository"` inside its payload
  and exits 1, and D7's text form renders that same string as prose after `normalizeProse`. The
  `after/` goldens and the end-to-end tests pin that exact string byte for byte.
- **Rationale:** the alternative is for the CLI to overwrite a payload field the engine authored,
  which is a second implementation of the outside-repo disposition — precisely what D8 routed through
  the engine to avoid. The rule's purpose is that quarry's *own* prose not name an internal package;
  a data field echoing the producer's error text is a different thing, and rewriting it would make the
  CLI's payload disagree with the facade's, which returns the engine value untouched (D5).
- **Noted defect, not fixed here:** the doubled prefix (`engine: resolve target …: engine: target
  outside repository`) is `internal/engine`'s own wording — `resolveTarget` wraps a sentinel whose
  text already carries the prefix. It is worth tightening, but `internal/engine/` is out of scope for
  this task (Scope/Out), so it is handed to the operator as a follow-up on `main` rather than
  papered over here. The goldens pin today's string; a later engine-wording change regenerates them.

### D11 — flags: `--text` and `--root` only; `--depth` and `--symbols` are rejected

- **Decision:** `resolve` and `expand` accept `--text` and `--root` (and `--help`/`-h`, which is
  scanned for at any position before anything else, unchanged). `--depth` and `--symbols` /
  `--no-symbols` on either verb are usage errors: `--depth is not valid for resolve` and the
  equivalent for each combination.
- **Rationale:** `resolvePathTarget` hardcodes `Depth: 0` and symbols off, and its doc comment gives
  the reason at length — this verb answers *where a thing is*, and what is inside it is `toc`'s
  question. Accepting a knob and silently ignoring it would be a lie in the interface; accepting it
  and honouring it would need an engine change, which is out of scope. `expand` has no path target at
  all.
- **Rejected:** silently ignoring the flags (lies); making them valid (engine change, out of scope).

### D12 — the parser's shape and its messages

- **Decision:** `parseArgs` keeps its hand-rolled structure and its `request` struct. Changes:
  - the verb gate accepts `toc`, `resolve` and `expand`; the no-verb message becomes
    `no verb given; expected: toc, resolve, or expand`; `unknown verb: <v>` is unchanged.
  - flag validity is checked per verb, producing `<flag> is not valid for <verb>`.
  - the arity message is spelled per verb: `<verb> takes exactly one target, got <n>`.
  - `expand` additionally rejects a target containing no `#` (D10) here, with
    `strings.Contains(target, "#")`. It belongs in `parseArgs` because that keeps the rejection pure
    over the argument slice — no root discovery, no engine call — which is the property the parser's
    fixture-free table test rests on.
  - `request` gains nothing: `verb` already exists and `depth`/`symbols` simply stay at their zero
    values for the two new verbs.
- **Rationale:** `parseArgs` is already pure over its argument slice, which is what lets its table
  test run with no fixtures; that property must survive. One parser for three verbs beats three
  parsers, because the shared flags (`--text`, `--root`, `--help`) are the majority.

### D13 — `usageText`

- **Decision:** extended to list all three verbs with their own flag lines, ASCII only, and with the
  exit-code block reworded so it is true of all three:

  ```
  usage:
    quarry toc <target> [--depth N|all] [--symbols|--no-symbols] [--text] [--root <path>]
    quarry resolve <glyph|path> [--text] [--root <path>]
    quarry expand <glyph> [--text] [--root <path>]
  ```

  with `1  negative answer: not found, outside the repository, ambiguous, not a type, or not a
  well-formed glyph` replacing
  the current line 1 wording, and the existing note that JSON is the default and there is no `--json`
  flag retained.
- **Rationale:** the current text names only `toc` and would be wrong the moment a second verb
  exists. `usage.go`'s ASCII-only and byte-comparable-in-tests constraints are kept.

### D14 — the `after/` evidence set

- **Decision:** eight new files, all real in-process invocations against the Loomyard checkout pinned
  at `72c23d9`, produced and compared by the existing `TestAfterGoldens` table:

  | golden | invocation | exit |
  |---|---|---|
  | `resolve-glyph.txt` | `resolve internal/logger#stderrHandlerSnapshot` | 0 |
  | `resolve-glyph-text.txt` | `resolve --text internal/logger#stderrHandlerSnapshot` | 0 |
  | `resolve-method.txt` | `resolve internal/logger#dualHandler.Handle` | 0 |
  | `resolve-not-found.txt` | `resolve internal/logger#noSuchSymbol` | 1 |
  | `resolve-path.txt` | `resolve internal/logger/logger.go` | 0 |
  | `expand-type.txt` | `expand internal/logger#dualHandler` | 0 |
  | `expand-type-text.txt` | `expand --text internal/logger#dualHandler` | 0 |
  | `expand-not-a-type.txt` | `expand internal/logger#newDualHandler` | 1 |

  Each file keeps T5a's exact format: the invocation line `$ quarry <verb> <args>`, a blank line, and
  stdout verbatim, **with no exit-code trailer**. The expected exit code moves into the test table
  and into `INDEX.md`'s own column. `afterGoldenCase` therefore gains a `verb` field as well as an
  exit-code field: today both the argv it builds and the invocation line it records hardcode `toc`.
- **Rationale:** the format is a T5a decision, stated in `after/INDEX.md` and contrasted there
  against the before side's untrue claim about its own trailers; changing it now would regenerate
  four committed goldens for a cosmetic reason. The exit code is still recorded in a tracked file
  (`INDEX.md`) and still asserted by the gate (the test table), which is what the done-when needs.
  The symbols chosen are the ones the before-side files actually queried, so the two sides answer the
  same questions about the same code.
- **Rejected:** adding `(exit code: N)` trailers (regenerates T5a's goldens, two formats or a
  conditional format in one directory); generating the `ambiguous` and `multipart` goldens from
  Loomyard (see D15 — Loomyard is not known to contain a build-tag-duplicated declaration or a
  several-`init` unit, and a golden that silently is not the case it claims to be is worse than none).

### D15 — `INDEX.md` is rewritten to make "nothing missing" checkable

- **Decision:** `after/INDEX.md`'s mapping table gains a row for **every** before-side file, each
  either naming a successor or saying "no successor, by design" with its reason, plus an exit-code
  column for the after files. The rows this task adds:

  | before | after | note |
  |---|---|---|
  | `../definition.txt` | `resolve-glyph.txt` (+ `-text`) | `definition --in-file internal/logger/logger.go stderrHandlerSnapshot` becomes `resolve internal/logger#stderrHandlerSnapshot`: no `--in-file`, no absolute path, a glyph instead of a bare name |
  | `../definition-ambiguous.txt` | `resolve-method.txt` | V1's ambiguity does not survive the glyph grammar. `definition … Handle` was a fuzzy name query that matched two receivers' methods; `internal/logger#dualHandler.Handle` names exactly one and answers `found`. The before side's `ok: true` with exit 2 was the addressing defect, not a fact about the repository |
  | `../symbol.txt` | `expand-type.txt` (+ `-text`) | `symbol dualHandler` was a fuzzy `workspace/symbol` passthrough returning an unrelated package's test function and undecoded LSP kind integers; `expand internal/logger#dualHandler` answers the same question exactly — the head plus every member, named kinds, glyph ids, no cross-package noise |
  | `../impact.txt`, `../impact-file-scope.txt` | *no successor, phase 2* | `impact` needs a type checker (plan §5); it is T8 |
  | `../assert-no-callers.txt` | *no successor, phase 2* | same, plan §5 and §12 T8 |
  | `../refs.txt` | *no successor, by design* | plan §5: "No reference query in phase 1" — dropped after measurement, not deferred |
  | *(none)* | `resolve-not-found.txt` | new: the `unit: found` miss plan §5 and §8.1 name as the validator's reason for the key. V1 had no equivalent — `definition` on a missing name returned an empty list |
  | *(none)* | `resolve-path.txt` | new: a repository-relative path as a target (plan §4, §5), which makes a non-code deliverable a checkable plan target. V1 had no path-target form at all |
  | *(none)* | `expand-not-a-type.txt` | new: plan §5's "the glyph must name a type; on any other kind the answer is `ok: false` naming the kind" |

  Three of the eight new goldens answer no before-side file, so they take `*(none)*` rows rather than
  being left out — the same device T5a used for `toc-dir-text.txt` and `toc-file-text.txt`. That is
  what makes the exit-code column total: **every** after-side file has a row, so every one of them
  records its exit code in a tracked file, which is D14's whole rationale for not writing trailers
  into the goldens themselves.

  The six rows T5a wrote stay exactly as they are: two `toc` before→after rows (`../toc-dir.txt`,
  `../toc-file.txt`), two compact-view "no successor, by design" rows (`../toc-dir-compact.txt`,
  `../toc-file-compact.txt`), and two `*(none)*` text-view rows (`toc-dir-text.txt`,
  `toc-file-text.txt`). They gain an exit-code cell (all four after files exit 0) and nothing else.
  A short "what changed" paragraph is added for the resolve/expand side, matching the one T5a wrote
  for `toc`.
- **Rationale:** the done-when is "`after/` covers the same command set as the before side — nothing
  missing". That is only checkable if every before-side file has an explicit row, and only true if
  the four phase-2 and dropped verbs are stated as deliberate absences with their plan citation.
  T5a set this precedent for the two compact files.

### D16 — the `docs/glyph.md` §5 statuses are proved through the CLI on a self-built fixture tree

- **Decision:** a new `internal/cli/glyph5_test.go` builds its own repository under
  `.scratch/cli-tests/` with the existing `writeScratchTree` helper and drives `Run` end to end,
  asserting the exit code and the decoded payload for all four statuses:

  | status | fixture |
  |---|---|
  | `found` | a package with one free function; `resolve <unit>#Fn` |
  | `not_found` with `unit: found` | the same package, `resolve <unit>#NoSuch` |
  | `not_found` with `unit: not_found` | `resolve nosuchdir#Fn` |
  | `ambiguous` with `candidates` | two files in one package, each declaring the same function under a different build tag |
  | `multipart` | two `func init()` in one package; `resolve <unit>#init` |
  | non-type | `expand <unit>#Fn` returns the D4 envelope and exit 1 |

- **Rationale:** the done-when requires these to answer correctly *through the CLI*, and the `after/`
  goldens cannot carry them: they skip on any machine without `LADDER_LOOMYARD_REPO`, and Loomyard is
  not known to contain a build-tag-duplicated declaration or a several-`init` unit. A self-built tree
  runs everywhere, including CI, which is what a done-criterion needs.
- **Rejected:** pointing `--root` at `internal/engine/testdata/tags` etc. — it couples `internal/cli`
  tests to another package's fixture paths, which the overview's per-package-copy convention (see
  `scratchtree_test.go`'s and `loomyard_test.go`'s own comments) already rejects for helpers.

## Technical context

**Module:** `github.com/Knatte18/quarry`. Go only; no Python (CLAUDE.md). No tracked file may carry a
machine-specific path.

**Layering, unchanged by this task:** `internal/engine` (all behaviour, no cwd, no git discovery) →
`quarry/` (public facade: aliases + delegating methods + renderers, no behaviour) → `internal/cli`
(the only layer with a working directory: flags, root discovery, target arithmetic, exit codes,
renderer choice) → `cmd/quarry` (one line).

**Files this task reads and must not change:**

- `internal/engine/answer.go` — `ResolveResult`, `ExpandAnswer`, `Status` and its four constants,
  `Symbol` (note `File` is populated by `resolve`/`expand` and empty inside `toc`).
- `internal/engine/resolve.go` — `Resolve`, `resolveGlyphTarget` (grammar rejection lands in the
  payload with `Error` + `Reason`, `Status` left empty; `Unit` set on `not_found` only),
  `resolvePathTarget` (`Depth: 0`, symbols off; `ErrTargetNotFound` → `status: not_found`;
  `ErrTargetOutsideRepo` → payload `Error`; anything else fails the whole call),
  `statusForMatches` (the one place a match set becomes a status).
- `internal/engine/expand.go` — `Expand`, `NotATypeError{ID, Kind}`, and the five-row disposition
  its doc comment enumerates.
- `internal/engine/repo.go` — `resolveTarget`: an absolute target or one that cleans to a leading
  `..` returns `ErrTargetOutsideRepo`. This is what D8 relies on.
- `glyph/errors.go` — `ParseError`, `Reason`, and `ReasonNoSeparator` (needed by D10).

**Files this task changes:**

- `quarry/quarry.go` (aliases + constants), `quarry/repo.go` (two methods), `quarry/render.go`
  (extract `renderJSON`, add two JSON renderers — or a new `quarry/resolve.go` / `quarry/expand.go`
  pair, mill-plan's call), `quarry/text.go` (the `File` prefix on `writeSymbolLine`; the two new text
  renderers), `quarry/doc.go`.
- `internal/cli/flags.go` (three verbs, per-verb flag validity, per-verb arity messages),
  `internal/cli/target.go` (split into `repoRelPath` + `repoRelTarget`), `internal/cli/cli.go` (two
  new pipelines beside `Run`'s existing one; `codeForTOCError` stays as-is and gains siblings for the
  two new verbs so each mapping stays table-testable), `internal/cli/usage.go`, `internal/cli/doc.go`.
  `internal/cli` gains one import: `glyph/` (D10a).
- `internal/cli/after_test.go` — eight new rows, and `afterGoldenCase` gains **two** fields: a `verb`
  (today `after_test.go` hardcodes `"toc"` into argv and `"$ quarry toc "` into the recorded
  invocation line, so both sites become `tc.verb`) and an expected exit code. `TestAfterGoldens`'s
  two blanket assertions — exit 0 and empty stderr — become per-case expectations at the same time
  (gotcha 1 below).
- **Existing test files this task edits**, all named by the Testing section and each an edit rather
  than a new file: `internal/cli/flags_test.go` (the parser table gains the three-verb gate, the
  per-verb flag-validity rejections, both arity messages, and D10's `#` check),
  `internal/cli/target_test.go` (the `repoRelPath`/`repoRelTarget` split must leave every existing
  row passing, plus new leading-`..` rows), `internal/cli/cli_test.go` (the two new pipelines and the
  new exit-code mappings), `quarry/render_test.go` (the two new JSON renderers and the shared-encoder
  extraction), `quarry/text_test.go` (the two new text renderers plus the `writeSymbolLine`
  no-regression golden), `quarry/repo_test.go` (the two new delegating methods and the
  `errors.As`-through-the-facade test).
- **New test files:** `internal/cli/glyph5_test.go` (D16).
- `docs/research/output-formats/after/INDEX.md` + eight new `.txt` files.

**Gotchas found during exploration:**

1. `TestAfterGoldens` currently asserts `code != exitOK` is a fatal failure and `stderr` is empty.
   Both must become per-case expectations, because `resolve-not-found.txt` exits 1 and
   `expand-not-a-type.txt` exits 1 *and writes to stderr* (`fail` always does).
2. The regeneration command in `after/INDEX.md` is
   `go test ./internal/cli/ -run TestAfter -update`, and `after_test.go`'s header comment says the
   function name and that command are load-bearing on each other. Keep the name.
3. `-update` is `internal/cli`'s own `flag.Bool`, deliberately distinct from `internal/engine`'s.
4. `quarry/text.go`'s `joinRel` is a deliberate re-declaration of the engine's unexported one. D7
   needs no new copy of anything like it.
5. `RenderText`'s `targetIsFile` parameter is authoritative and never inferred. D7's decision to use
   the directory form for path results is what keeps it out of `resolve`'s pipeline.
6. `fail()` writes the JSON envelope to stdout on *every* failure path including under `--text`; that
   rule is unchanged and applies to D4 and to every exit-2/exit-3 path here.
7. `.golangci.*` is absent from the repository root, so `golangci-lint run` uses its defaults. The
   `errcheck` default is on — commit `e45c649` exists because of it; every new `io.WriteString` /
   `Write` must handle or explicitly discard its error the way `cli.go` already does.
8. `docs/research/output-formats/` before-side files record absolute machine paths. The after side
   must not, which is why every golden target is repository-relative and `--root` is stripped from
   the recorded invocation line (`afterGoldenCase.invocation` is spelled literally per row for
   exactly this reason).

## Constraints

- No `CONSTRAINTS.md` at the hub root.
- **Go only, no Python** (CLAUDE.md).
- **No machine path in any tracked file** (task body; HANDOFF §1).
- **`internal/engine/` is not modified** (task scope, Out).
- **Nothing new invented:** every envelope or view question is to be answered from plan §4 or T5a's
  merged code; where one genuinely is not, stop and report rather than decide locally. Three such
  questions were found and are flagged in D2, D3 and D10 for the round-1 reviewer.
- **MCP is out of scope** (T6 owns `cmd/quarry-mcp`).
- **Done gate:** `go test ./... && golangci-lint run` green.
- **The four `toc` goldens must still compare equal without regeneration** — the proof that the
  shared-helper refactors of D6 and D7 changed nothing.
- Tests never use the system temp directory; `.scratch/` is the sanctioned location
  (`mill:conversation`, and `scratchtree_test.go`'s own comment).

## Testing

**`quarry/` (facade).**

- `RenderResolveJSON` / `RenderExpandJSON`: table over the payload shapes — `found`, `multipart`,
  `ambiguous`, `not_found` with each `unit` value, a grammar rejection carrying `error` + `reason`, a
  path result carrying `dir`. Assert the byte contract T5a's `render_test.go` already asserts for
  `RenderJSON`: two-space indent, one trailing newline, no HTML escaping, no `ok` key, key order =
  struct declaration order, and — for `not_found` — that `symbols` and `candidates` are absent rather
  than `null` or `[]`.
- `RenderResolveText` / `RenderExpandText`: table over every branch of D7's grammar, asserted as
  exact strings. **These are the TDD candidates**: the grammar is fully specified above, so the tests
  can be written before the renderer.
- A regression test that `RenderText` over a `toc` answer is byte-identical before and after the
  `writeSymbolLine` change — cheapest as a fixed golden string in `quarry/text_test.go`, which
  already exists.
- `(*Repo).Resolve` / `(*Repo).Expand`: that they return the engine's value and error unchanged, and
  that `errors.As(err, &notType)` against `*quarry.NotATypeError` succeeds — the transitivity test,
  the analogue of the existing `errors.Is` sentinel tests in `quarry/repo_test.go`.

**`internal/cli/`.**

- `parseArgs`: extend the existing pure table with the three-verb gate, every per-verb flag-validity
  rejection, both new arity messages, the new no-verb message, and `--help` still winning from any
  position for every verb. Pure, no fixtures. **TDD candidate.**
- `repoRelPath` / `repoRelTarget`: the split must leave `toc`'s existing `target_test.go` table
  passing unchanged, plus new cases asserting `repoRelPath` returns the leading-`..` form rather than
  rejecting it, and that a glyph target never reaches either function.
- The exit-code mappings: one named, table-testable function per verb, mirroring
  `codeForTOCError`'s existing rationale — a table from (payload, error) to code, covering every row
  of D3.
- `Run` end to end for both verbs against a scratch fixture tree: JSON and `--text`, success and every
  negative, asserting exit code, stdout bytes and stderr bytes together (the "stdout gets the
  envelope, stderr gets the same sentence" rule).
- `glyph5_test.go` per D16 — the done-when's own gate. **TDD candidate**, since the fixtures and
  expected statuses are enumerated above.
- `after_test.go` per D14, gated on `LADDER_LOOMYARD_REPO` exactly as today.

**Scenarios that must be covered and are easy to miss:**

- `resolve` of a path that escapes the root answers with the payload's `error` and exits 1, not with
  the error envelope (D2 + D8).
- `resolve` of a nonexistent path answers `status: not_found` with no `unit` key (a path belongs to
  no unit) and exits 1.
- `resolve` of a glyph whose unit does not exist answers `not_found` with `unit: not_found`; of a
  glyph whose unit exists but whose member does not, `not_found` with `unit: found`. These two are
  the validator's whole reason for the key (plan §5, §8.1).
- `expand` of a type with no members answers `found` with a head and no `members` key — not
  `not_found`, and not an error.
- `expand internal/logger` (no `#`) is rejected by `parseArgs` with exit 2 before any engine call
  (D10); `expand '#x'` — a `#`-bearing target the grammar still rejects — takes D4's exit-1 envelope
  path instead. Both belong in the tables.
- Every failure path writes the JSON envelope to stdout even under `--text`.
- `--root` rebases path-target interpretation but has no effect on a glyph target.
- A path target given relative to a subdirectory echoes the repository-relative form in the payload's
  `target` and on the text view's line 1, while a glyph target echoes argv verbatim (D8). Cover both
  in one test so the asymmetry is pinned rather than discovered.

## Q&A log

- **Q:** Does the CLI take one target per call or plan §5's variadic `resolve <glyph|path>...`? **A:** [auto-pick] One target per call for both verbs; the facade keeps the engine's multi-target `Resolve([]string)`. **Why:** the task body states it, and a variadic CLI would require a new aggregate-exit-code rule the task forbids inventing; plan §5's batching survives on the facade, which plan §7 names the primary surface. Flagged for review — plan §5's `resolve` bullet is not amended anywhere.
- **Q:** On a negative answer, does stdout carry the payload or T5a's `{"ok":false,"error":…}` envelope? **A:** [auto-pick] The payload, rendered by the selected renderer; the envelope is used only when there is no payload. **Why:** the envelope would destroy `unit`, `candidates` and `reason` — the three fields plan §5 and §8.1 name as the reason the validator asks. `toc` uses the envelope on exit 1 only because `toc` has no negative payload.
- **Q:** Which exit code does `ambiguous` get? **A:** [auto-pick] 1. **Why:** exit 2 means "the caller asked wrong about the CLI"; an ambiguous glyph is a well-formed invocation run to a definite conclusion, which is exit 1's stated meaning. Exit 2 here would repeat V1's `definition` defect in mirror image.
- **Q:** Which exit code does a glyph grammar rejection get under `resolve`? **A:** [auto-pick] 1, with the payload carrying `error` and `reason`. **Why:** exit 2 prints `usageText` and routes through `fail()`, which discards the `reason` word the engine deliberately put in the payload. Flagged for review.
- **Q:** What does `expand` do with a non-type glyph? **A:** [auto-pick] The error envelope, exit 1, sentence `expand <id>: not a type, kind <kind>` spelled by the CLI from `*quarry.NotATypeError`'s fields. **Why:** plan §5 says the answer is `ok: false` naming the kind, and `Expand` returns no payload on that path; T5a forbids leaking the engine's `engine: ` prefix.
- **Q:** What does `expand` do with a target containing no `#`? **A:** [auto-pick, revised at round-1 review] Usage error, exit 2, message `expand takes a glyph (a target containing "#"), got: <target>`, decided by the CLI's own `strings.Contains(target, "#")` check at argument-handling time, before any engine call. **Why:** `expand` accepts one argument class, so a target with no `#` is the wrong verb, which is exit 2's meaning; and routing it post-engine via `parseErr.Reason == glyph.ReasonNoSeparator` (the original auto-pick) would contradict exit 2's own documented "TOC is never called on this path" and duplicate a check D8 already gives the CLI.
- **Q:** Do `resolve` and `expand` accept `--depth` and `--symbols`? **A:** [auto-pick] No; both are usage errors naming the verb. **Why:** `resolvePathTarget` hardcodes `Depth: 0` and symbols off by design, so accepting the knobs would either lie or require an out-of-scope engine change.
- **Q:** How is the `resolve`/`expand` symbol line rendered in the text view, given those symbols carry `file` and `toc`'s do not? **A:** [auto-pick] `writeSymbolLine` gains a `<file>:` prefix emitted only when `File != ""`. **Why:** `File` is always empty inside a `toc` answer, so every existing `toc` golden stays byte-identical and there is one grammar with one implementation.
- **Q:** Does a `resolve` path result render `RenderText`'s file form or directory form? **A:** [auto-pick] The directory form, always. **Why:** `resolvePathTarget` uses `Depth: 0` with symbols off, so a file target's answer is its enclosing directory's answer with one entry, which the directory form renders losslessly — and this is what lets `resolve`'s pipeline drop `toc`'s `os.Lstat` step entirely.
- **Q:** How are head and members told apart in `expand`'s text view without a key? **A:** [auto-pick] One blank line between the head line and the member lines. **Why:** it is the same block separator `dirBlocks` already uses, so no new marker is invented.
- **Q:** Does the CLI stat a `resolve` target before calling the engine? **A:** [auto-pick] No. **Why:** "does not exist" is the engine's own payload answer; pre-empting it with `fail()` would destroy it.
- **Q:** How does a path target that escapes the root reach the engine, given `repoRelTarget` rejects it? **A:** [auto-pick] Split `repoRelTarget` into `repoRelPath` (arithmetic only, returns the leading-`..` form) and the existing rejecting wrapper `toc` keeps using; `resolve` calls `repoRelPath`. **Why:** the engine's `resolveTarget` already owns the outside-repo rule, and synthesising the rejection in the CLI would be a second implementation of it.
- **Q:** Do the new `after/` goldens record an exit-code trailer? **A:** [auto-pick] No — T5a's format is kept, the expected code moves into the test table and into `INDEX.md`'s own column. **Why:** T5a decided the format explicitly and criticised the before side over it; adding trailers would regenerate four committed goldens for a cosmetic reason, or leave two formats in one directory.
- **Q:** Are `ambiguous` and `multipart` covered by `after/` goldens? **A:** [auto-pick] No — by a self-built fixture tree in `internal/cli/glyph5_test.go`. **Why:** the `after/` goldens skip wherever `LADDER_LOOMYARD_REPO` is unset, and Loomyard is not known to contain a build-tag-duplicated declaration or a several-`init` unit; a golden that silently is not the case it claims is worse than none.
- **Q:** How is "`after/` covers the same command set — nothing missing" made checkable, given `impact`, `refs` and `assert-no-callers` have no phase-1 successor? **A:** [auto-pick] `INDEX.md`'s mapping table gets a row for every before-side file, each naming a successor or stating "no successor, by design" with its plan citation. **Why:** T5a set exactly this precedent for the two compact-view files.
- **Q:** Is `RenderJSON` generalised to `any`, or are there per-type renderers? **A:** [auto-pick] Per-type exported renderers over one shared unexported `renderJSON(v any)`. **Why:** T6 is being written against `RenderJSON`'s current signature in parallel, and a generic exported renderer would let any type reach a function contracted to emit plan §4's key set.
- **Q:** How does the CLI obtain the `<reason>` word for `expand`'s exit-1 grammar rejections, given `Expand` wraps the parse error behind an `engine:` prefix D4 forbids passing through? **A:** [round-2 gap resolution] `internal/cli` imports `glyph/` directly and reaches `parseErr.Reason` with `errors.As` against `*glyph.ParseError`; no facade alias is added (D10a). **Why:** `glyph/` is a public, cgo-free, dependency-free package, so the import costs nothing; the facade's aliases exist only because `internal/engine` is unreachable, and aliasing a reachable public type would be a second spelling of the grammar's own error type, which `docs/glyph.md` §6 forbids. Dropping `<reason>` was rejected because it is the only actionable part of the rejection and is the same word `resolve` puts in its payload's `reason` key.
