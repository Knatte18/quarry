# Batch: facade-renderers

```yaml
task: "Facade + CLI, toc (T5a)"
batch: "facade-renderers"
number: 2
cards: 4
verify: go test ./quarry/...
depends-on: [1]
```

## Batch Scope

This batch adds the two renderers the facade exports so that T6's MCP can be a transport shim and
the CLI can be a flag parser: `RenderJSON` and `RenderErrorJSON` (the wire contract) and
`RenderText` (the lossless text view whose grammar `discussion.md`'s `text-view-grammar` fixes to
the character). It is one batch because both renderers consume the same `DirAnswer` and are tested
the same way — over hand-built values with no filesystem at all — and because the text view's tag
order and prose normalisation are the kind of detail that drifts if split across two sessions.

The external interface batch 4 consumes: `quarry.RenderJSON`, `quarry.RenderErrorJSON`,
`quarry.RenderText`.

Batch-local decision: `RenderText` needs the engine's `joinRel` rule and cannot call it (see the
overview's `engine-unexported-helpers-are-not-reachable`), so card 6 declares an unexported
`joinRel` in `quarry/text.go` with the same three-line body.

## Cards

### Card 5: RenderJSON and RenderErrorJSON

- **Context:**
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `internal/engine/answer.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `quarry/render.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/render.go` in `package quarry`.
  Declare `func RenderJSON(a DirAnswer) ([]byte, error)`: build a `bytes.Buffer`, wrap it in a
  `json.NewEncoder`, call `SetEscapeHTML(false)` and `SetIndent("", "  ")`, `Encode(a)`, and return
  the buffer's bytes. `json.Encoder.Encode` already appends exactly one trailing newline — do not
  append another. On an encode error return `nil` and the error.
  Declare an unexported `type errorEnvelope struct` with exactly two fields:
  `OK bool` tagged `json:"ok"` and `Error string` tagged `json:"error"`. `OK` is never set, so it
  marshals as `false`; the tag carries no `omitempty`, which is what makes `ok` present on the
  failure path and absent on the success path, since `DirAnswer` has no `ok` field at all.
  Declare `func RenderErrorJSON(msg string) []byte`: encode `errorEnvelope{Error: msg}` through a
  `json.NewEncoder` with `SetEscapeHTML(false)` and **no** `SetIndent`, returning the buffer's
  bytes. The emitted bytes are therefore `{"ok":false,"error":"<msg>"}` followed by one `\n`, with
  no space after either colon — see the overview's `json-encoder-spacing-is-the-byte-contract`.
  The signature returns no error, so handle the impossible encode failure by returning a hand-built
  constant envelope with the message `internal error: failed to render error envelope`, and comment
  that the branch is unreachable because a struct of one bool and one string cannot fail to marshal
  and a `bytes.Buffer` write cannot fail — the fallback exists so stdout always carries a parseable
  object rather than nothing.
  Document on `RenderJSON` why HTML escaping is disabled: headers and package docs are real prose
  containing `<`, `>` and `&`, and the default encoder would emit `<` and make the output both
  unreadable and unequal to `docs/rewrite-plan.md` §4's examples. Document that key order is the
  struct field declaration order of `internal/engine/answer.go`, which is already §4's order, and
  that this is why no hand-written marshaller is needed.
  Document on `RenderErrorJSON` that `ok` is present only on failure, so it can never disagree with
  the exit code, and that the payload carries `ok` and `error` only — no `kind`, no `status`,
  because the exit code already discriminates.
- **Commit:** `feat(quarry): add RenderJSON and RenderErrorJSON`

### Card 6: RenderText — the lossless text view

- **Context:**
  - `quarry/quarry.go`
  - `quarry/render.go`
  - `internal/engine/answer.go`
  - `internal/engine/walk.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `quarry/text.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/text.go` in `package quarry`, declaring
  `func RenderText(a DirAnswer, targetIsFile bool) string` plus the unexported helpers below. The
  returned string has no trailing whitespace on any line and ends with exactly one `\n`. It cannot
  fail and returns no error.

  Declare `func normalizeProse(s string) string` returning `strings.Join(strings.Fields(s), " ")`.
  Apply it to every `Doc`, `Header`, `Signature` and `FileEntry.Error` value before printing.
  Its doc comment records that `FileEntry.Error` is in the list because it is an arbitrary `os` or
  UTF-8 message quarry does not author, emitted inside a bracketed tag, so a multi-line value would
  break the one-record-per-line property the whole format rests on — and that "prose intact" means
  nothing is truncated or dropped, not that source line breaks survive.

  Declare `func joinRel(parent, child string) string` returning `child` when `parent == "."` and
  `parent + "/" + child` otherwise. Its doc comment records that this is the same rule as
  `internal/engine/walk.go`'s own unexported `joinRel`, re-declared here because that one is
  unexported and this task does not modify the engine.

  **Directory form** (`targetIsFile` false). Emit one block per `DirAnswer` in depth-first order —
  `a` itself, then each entry of `a.Dirs` in order, recursively — with exactly one blank line
  between consecutive blocks and no blank line before the first or after the last. Each block:
  - Line 1: `a.Dir`, then ` (package <Package>, <Language>)` only when `Package` is non-empty, with
    `, <Language>` inside those parentheses only when `Language` is non-empty (so a package with no
    language emits ` (package <Package>)`), then `, <N> files` only when `len(a.Files) > 0`, using
    the singular `, 1 file` when `len(a.Files) == 1`.
  - Line 2: `normalizeProse(a.Doc)` on one line, emitted only when `a.Doc` is non-empty. When it is
    empty, emit nothing — no blank line, no placeholder.
  - One line per entry of `a.Files`, in the slice's own order: the bare `Name` (not a path — the
    directory heads the block), then the tags, then `: ` + `normalizeProse(Header)` when `Header` is
    non-empty. A file with no header emits the name and tags and no colon.
  - After each file's line, its symbol lines when `Symbols` is non-nil, one per element in order,
    immediately before the next file's line.

  **Tags.** `func fileTags(fe FileEntry) string` returns a space-separated run of bracketed markers,
  each emitted only when its underlying field is present, in this fixed order:
  `[test]` when `Test`; `[generated]` when `Generated`; `[package <Package>]` when `Package` is
  non-empty; `[language <Language>]` when `Language` is non-empty; `[lossy]` when `Lossy`;
  `[error <normalizeProse(Error)>]` when `Error` is non-empty. When at least one tag is emitted the
  run is preceded by a single space; when none is, nothing is emitted.

  **File form** (`targetIsFile` true). Emit exactly one block, no directory line and no directory
  doc: `joinRel(a.Dir, fe.Name)`, then ` (package <Package>, <Language>)` under the same two
  presence rules as the directory form's line 1 — these are the enclosing directory's facts, taken
  from `a`, not from the entry — then the entry's tags, then `: ` + `normalizeProse(fe.Header)` when
  the header is non-empty; then the entry's symbol lines. `fe` is `a.Files[0]`. When `a.Files` is
  empty — which the engine never produces for a file target — emit the directory form's line 1 alone
  rather than panicking, and say so in the doc comment.

  **Symbol lines**, identical in both forms, one per symbol in the answer's order:
  `<Start>-<End>`, then ` (sig <Start>-<SigEnd>)` only when `SigEnd != 0`, then ` <ID>: ` +
  `normalizeProse(Signature)`. When `Doc` is non-empty, a following line of exactly four spaces then
  `normalizeProse(Doc)`; when it is empty, no line at all — no blank line, no placeholder.

  Document on `RenderText` that `targetIsFile` is authoritative and is never inferred from the
  answer's shape, because a directory holding exactly one file and no subdirectories is
  indistinguishable from a file target by shape alone, so the caller — which knows what it asked
  for — must say. Document that `SigEnd == 0` is the engine's documented marker for a symbol with no
  body (a Go type alias), never line zero, since every real line number is 1-based.
  Do not write a parser, a round-trip helper, or any escaping: the text view is a rendering and JSON
  remains the contract.
- **Commit:** `feat(quarry): add the lossless RenderText view`

### Card 7: JSON renderer tests

- **Context:**
  - `quarry/render.go`
  - `quarry/quarry.go`
  - `internal/engine/answer.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `quarry/render_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/render_test.go` in `package quarry`, over hand-built `DirAnswer`
  values only — no filesystem, no `Open`, no `TOC`.
  Cover, per `discussion.md`'s "JSON view" test list: key order matches
  `internal/engine/answer.go`'s struct field order (`dir`, `package`, `language`, `doc`, `files`,
  `dirs`; and within a file entry `name`, `header`, `test`, `generated`, `package`, `language`,
  `lossy`, `error`, `symbols`), asserted on the rendered bytes rather than on a decoded map, since a
  map loses order; absent fields are absent — a `DirAnswer` with `Test` false and no `Dirs` renders
  with no `"test"` key and no `"dirs"` key; no `ok` key appears on the success path; a `Doc`
  containing `<`, `>` and `&` renders those characters literally and not as `<` and friends;
  indentation is two spaces; the output ends with exactly one `\n` and does not end with two.
  Cover `RenderErrorJSON`: the exact bytes for a plain message are
  `{"ok":false,"error":"boom"}` + `"\n"`, asserted as a byte-for-byte string comparison; a message
  containing `<` and `&` is not escaped; a message containing a double quote is JSON-escaped
  normally.
  Cover the `Symbols` distinction that matters at the wire level: a `FileEntry` with a nil
  `Symbols` and one with a pointer to an empty slice both render with no `"symbols"` key, since
  `omitempty` drops both — the distinction exists for Go callers only.
- **Commit:** `test(quarry): pin the JSON view's key order, escaping, and defaults`

### Card 8: text view tests

- **Context:**
  - `quarry/text.go`
  - `quarry/quarry.go`
  - `internal/engine/answer.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `quarry/text_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `quarry/text_test.go` in `package quarry`, table-driven, over hand-built
  `DirAnswer` values only — no filesystem.
  Cover, per `discussion.md`'s text-view test list:
  *Directory form* — a directory with a package and a doc; one with neither; one with a package but
  no language; `1 file` singular versus `N files` plural; a directory with no files emitting no
  count clause at all; a depth-cut subdirectory carrying only `Dir` and `Doc` emitting exactly two
  lines; nested blocks in depth-first order separated by exactly one blank line, with no leading or
  trailing blank line.
  *Tags* — each of `test`, `generated`, `package`, `language`, `lossy`, `error` alone, and all six
  together in the fixed order `[test] [generated] [package p] [language go] [lossy] [error msg]`;
  plus a multi-line `Error` value collapsing to one line.
  *File form* — an answer whose `Dir` is `"."` emitting the entry's bare base name alone, with no
  leading dot-slash prefix, which is what the `joinRel` rule buys over a naive dir-slash-name
  template; a file form with the enclosing directory's package facts on the same line and the directory's doc
  absent; the same one-file answer rendered both ways, asserting the form is chosen by the
  `targetIsFile` argument and not by the answer's shape.
  *Symbols* — a symbol with a doc and one without; one with `SigEnd` zero emitting no `(sig …)`
  group and one with a non-zero `SigEnd` emitting `(sig <Start>-<SigEnd>)`; a file entry with nil
  `Symbols` and one with a pointer to an empty slice rendering identically.
  *Prose normalisation* — use `docs/rewrite-plan.md` §4's own `placement` example as the fixture,
  since it is the one case the plan shows both sides of: the `doc` value
  `placement is one resolved pane: its tmux pane id and the row height it\nhas been assigned.`
  renders as
  `placement is one resolved pane: its tmux pane id and the row height it has been assigned.`
  Also assert a run of spaces and a tab collapse to one space and that nothing is truncated.
  *Whole-output invariants* — for every case, no line has trailing whitespace and the output ends
  with exactly one `\n`.
- **Commit:** `test(quarry): pin the text view grammar to the character`

## Batch Tests

`verify: go test ./quarry/...` runs `quarry/render_test.go` and `quarry/text_test.go` alongside
batch 1's `quarry/repo_test.go`. Scoped to the `quarry` package because that is the only package
this batch touches. Every test in this batch builds its `DirAnswer` values by hand and touches no
filesystem, so the batch is fully verifiable on a machine with no Loomyard checkout — the goldens
that do need one are batch 5's.
