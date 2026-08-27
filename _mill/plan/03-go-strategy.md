# Batch: go-strategy

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "go-strategy"
number: 3
cards: 5
verify: go test ./internal/quarryengine/toc
depends-on: [2]
```

## Batch Scope

This batch adds the shared tree-walking helpers every strategy uses, and the first concrete
`Strategy` — Go. Go is first because it is the language whose declarations are flat top-level
children of the root, so the walk is simple, while its `type_declaration` shape is the one that makes
the signature rule non-trivial and forces the helpers to be right before two more languages depend
on them.

The external interface batch 4 consumes is `nodes.go`'s helpers: `NodeText`, `SignatureCut`,
`CommentBlockAbove`, and `LeadingBlocks`.

Batch-local decisions, all established by dumping real parse trees from the pinned grammar rather
than assumed:

- A Go **type alias** (`type Alias = T`) parses as a `type_alias` node, **not** a `type_spec`. Both
  shapes must be handled; a walk that only looks for `type_spec` silently drops every alias in the
  file.
- A **grouped** `type ( ... )` declaration holds its per-spec doc comments as `comment` children
  *inside* the `type_declaration`, interleaved with the `type_spec` children — so the same
  prev-sibling comment walk works there, one level down.
- `struct_type`'s body child is `field_declaration_list`; `interface_type` has **no** body field and
  no named body node at all — its body begins at its literal `{` child. One rule covers both: the
  first direct child of the type node whose kind is `field_declaration_list` or `{`.

## Cards

### Card 17: shared node helpers

- **Context:**
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/classify.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/nodes.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add the language-independent node helpers every strategy shares. Import the
  runtime as `ts "github.com/tree-sitter/go-tree-sitter"`.
  - `NodeText(n *ts.Node, src []byte) string` — `string(src[n.StartByte():n.EndByte()])`, the
    verbatim source span, untrimmed.
  - `Line(n *ts.Node) (start, end int)` — the node's 1-based inclusive line range, computed as
    `int(n.StartPosition().Row) + 1` and `int(n.EndPosition().Row) + 1`. Every line number in the
    package goes through this one function so the 0-based-to-1-based conversion has exactly one
    implementation.
  - `SignatureCut(decl *ts.Node, body *ts.Node, src []byte) string` — when `body` is non-nil, the
    trimmed source from `decl.StartByte()` to `body.StartByte()`; when `body` is nil, the trimmed
    whole of `NodeText(decl, src)`. It must never truncate at a newline: a multi-line parameter list
    is part of the signature.
  - `SigEnd(decl *ts.Node, body *ts.Node, bodyOnSignatureLine bool) int` — the last line of the
    signature, the single implementation of the per-language derivation. When `body` is nil it returns
    `0`, the absent marker that omits the field. When `bodyOnSignatureLine` is true (Go, and a
    block-bodied C# member, where the `{` sits on the signature's own last line) it returns the body
    node's start line. When false (Python, where the `block` starts on the line after the `def`, and
    an expression-bodied C# member, where the signature ends before the `=>`) it returns the body's
    start line minus one, **clamped so it is never below the declaration's own start line**.
    The clamp is load-bearing rather than defensive: a single-line `def f(): return 1` or
    `void F() => 1;` puts the body on the declaration's own line, and an unclamped subtraction would
    emit a `sigend` above `start`. Say so in the doc comment.
    Document the other direction too: using "where the body begins" directly for every language does
    not work, which is why this helper takes the flag at all — in Go the `{` is on the signature's
    last line, in Python the body starts a line later, so one uniform rule would leak a line of
    implementation into every Python signature range.
    Note the known imprecision as well: a single-line Go function shares one line between signature
    and body, so `start`–`sigend` includes the body there. No line-based range can separate them, and
    the fix is help text, not columns.
  - `CommentBlockAbove(n *ts.Node, src []byte) (first *ts.Node, raw string)` — walks `PrevSibling`
    backwards over contiguous `comment` nodes, stopping at the first non-comment sibling **or** at the
    first blank line, detected as `prev.EndPosition().Row + 1 != cur.StartPosition().Row`. Returns the
    topmost comment node of the block and the raw joined source of its lines, or `(nil, "")` when the
    node has no adjacent comment. The strict-adjacency stop is what keeps a trailing comment on the
    previous declaration from being misattributed to this one; say so in the doc comment, and note
    that the file-header rule deliberately differs by tolerating one blank line.
  - `LeadingBlocks(root *ts.Node, src []byte) []*ts.Node` — the root's leading `comment` children
    grouped into blocks by the same blank-line rule, returned in source order as the first node of
    each block, stopping at the first non-comment child of the root. This is what the header rule
    iterates to skip directive blocks.
  Every helper takes `src` explicitly rather than closing over it, and none of them retains a node
  beyond its own return — the tree is closed by `treesitter.WithTree` as soon as extraction ends.
- **Commit:** `feat(toc): add the shared tree-sitter node helpers`

### Card 18: Go functions and methods

- **Context:**
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/comments.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/golang.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create the Go strategy as an unexported `goStrategy` struct registered from this
  file's `init` via `Register`, with `Language()` returning `"go"`.
  `Symbols` iterates the direct children of the `source_file` root in order and handles exactly three
  kinds, ignoring every other child — in particular it never descends into a `block`, so a
  `type_declaration` or `func` literal inside a function body is not listed:
  - `function_declaration` — `Kind` is `KindFunction`; `Name` is the `name` field child's text;
    `Owner` is empty; the body-bearing child is `ChildByFieldName("body")`, a `block`.
  - `method_declaration` — `Kind` is `KindMethod`; `Name` is the `name` field child's text (a
    `field_identifier`); `Owner` is the receiver's type name, read from the `receiver` field child (a
    `parameter_list`) by taking its `parameter_declaration`'s `type` field and stripping a leading
    `*` when it is a `pointer_type`, so a `*T` receiver yields `T`; the body-bearing child is
    `ChildByFieldName("body")`.
  - `type_declaration` — handled by card 19.
  For each symbol, `Docstring` is `StripLineComment(raw, "//")` over `CommentBlockAbove`'s raw text,
  and `Start` is the comment block's first line when a block was found and the declaration's own
  first line otherwise; `End` is always the declaration's last line. Emit the **full** docstring —
  sentence trimming happens in the entry point.
  `Signature` is `SignatureCut(decl, body, src)`, and `SigEnd` is
  `SigEnd(decl, body, true)` — Go puts the `{` on the signature's own last line, which is what the
  `true` flag means.
- **Commit:** `feat(toc): add Go function and method extraction`

### Card 19: Go type declarations, grouped and ungrouped

- **Context:**
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/comments.go`
- **Edits:**
  - `internal/quarryengine/toc/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** extend `goStrategy.Symbols` to emit `KindType` symbols from a
  `type_declaration`, with one rule that covers both source shapes rather than a branch per shape.
  The symbol unit is always the spec node — either a `type_spec` or, for `type X = Y`, a `type_alias`
  node, which the pinned grammar produces instead of a `type_spec`. Handle both kinds; a walk that
  matches only `type_spec` drops every alias.
  The two shapes are distinguished by the **presence of a `(` child of the `type_declaration`**, never
  by counting spec children. `type ( X int )` is legal Go with a single spec, and a spec-count test
  would route it down the ungrouped path, cutting `Signature` from the `type_declaration` and emitting
  the whole `type (\n\tX int\n)` block with the group's range instead of the spec's. Verified against
  the pinned tree-sitter-go v0.25.0: a grouped declaration always has a literal `(` child and an
  ungrouped one never does, whatever the spec count. State the reason in a comment at the branch.
  - **Ungrouped** — the `type_declaration` has no `(` child. It holds exactly one spec. Emit one
    symbol whose
    `Signature` and range are computed from the enclosing `type_declaration`, not from the spec:
    `Start` is the `type_declaration`'s doc-comment block's first line (or the declaration's own),
    `End` is the declaration's last line, and `Signature` is `SignatureCut` from the *declaration's*
    first byte, so the emitted signature includes the `type` keyword. A bare `FileLock struct` would
    be invalid Go and useless to paste anywhere.
  - **Grouped** — the `type_declaration` has a `(` child, and holds one or more spec children between
    `(` and `)`. Emit one symbol per spec. Each spec's docstring is its own `CommentBlockAbove` walked from the spec node,
    which works unchanged because the grammar makes the group's per-spec doc comments `comment`
    children of the `type_declaration`, interleaved with the specs. Each spec's range is the spec's
    own lines, extended upward to its comment block when it has one. Each spec's signature is rendered
    by prepending `"type "` to the spec's own signature text, so the grouped and ungrouped forms
    produce identical output for identical types.
    A comment attached to the `type (` line itself documents the group rather than any one spec and is
    dropped: it is a prev-sibling of the `type_declaration`, not of any spec, so this falls out of the
    rule rather than needing a special case — but assert it in the tests, because it is the part a
    later refactor would break silently.
  - **Name** is the spec's `name` field child's text, for both spec kinds.
  - **The body-bearing child of a spec** is resolved from the spec's `type` field child: the first
    direct child of that type node whose kind is `field_declaration_list` (a `struct_type`'s body) or
    `{` (an `interface_type`'s body, which the grammar exposes with no named body node and no `body`
    field). When neither exists — `type ID string`, `type Alias = T` — there is no body-bearing child
    and the whole spec text is the signature, which is short by construction.
  - **`SigEnd`** is `SigEnd(decl, goTypeBody(spec), true)` for a spec that has a body-bearing child,
    and `0` — omitted — for one that does not. A `type ID string` or `type Alias = T` has no body at
    all, so there is no separate "signature end" to report: `start`–`end` already is the signature.
  Add a helper `goTypeBody(spec *ts.Node) *ts.Node` implementing that resolution once, and document
  in its comment why a naive `ChildByFieldName("body")` is wrong here: a Go `type_declaration` has no
  `body` field, so the naive call returns nil and the signature silently becomes the entire struct
  body — the exact token blowup this verb exists to prevent.
- **Commit:** `feat(toc): add Go type declaration extraction`

### Card 20: Go header, generated and test-file rules

- **Context:**
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/classify.go`
  - `internal/quarryengine/toc/strategy.go`
- **Edits:**
  - `internal/quarryengine/toc/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** implement the remaining four `Strategy` methods on `goStrategy`.
  `Package` returns the text of the `package_clause` child's `package_identifier`, or `""` when the
  root has no `package_clause` — which is what a file broken badly enough to lose its package clause
  under a `partial` parse looks like.
  `Header` walks `LeadingBlocks` in order, strips each block with `StripLineComment(raw, "//")`, and
  returns the first block for which `IsDirectiveBlock("go", blockStartLine, stripped)` is false. When
  every leading block is a directive block, or the file has no leading comment, it returns `""`.
  This rule deliberately differs from docstring association in two ways, both of which must be stated
  in the method's doc comment: it takes the **first** non-directive block rather than the block
  adjacent to `package`, and it tolerates a blank line between that block and whatever follows. Both
  matter in this very repository — a file can carry a build constraint, a blank line, then its real
  header, and a file can carry both a file header and a separate package doc comment, where the file
  header is the one that describes the file.
  `Header` returns the block untruncated; `FirstParagraph` is applied by the entry points.
  `Generated` reads the raw text of the **first** leading block (directive or not, since the banner is
  a directive block for header purposes and a marker here — the two readings are independent) and
  delegates to `GeneratedByBanner("go", raw)`.
  `TestFile` delegates to `TestFileByName("go", base)`.
- **Commit:** `feat(toc): add the Go header, generated and test-file rules`

### Card 21: Go strategy tests

- **Context:**
  - `internal/quarryengine/toc/golang.go`
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/golang_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** table-driven tests over small inline Go source fixtures, each parsed through
  `treesitter.WithTree` and passed to the registered Go strategy. Add a small test helper in this file
  that takes a fixture string and returns the extracted `[]Symbol` and header, so each table row stays
  one fixture plus its expectation.
  Symbol cases, each asserting the full `Symbol` value rather than one field:
  - a function with a docstring — the range starts at the docstring's first line;
  - a function with no docstring — `Docstring` is empty and the range starts at the declaration;
  - two declarations where a blank line separates the first one's trailing comment from the second
    declaration — the comment is not misattributed to the second symbol;
  - a comment block separated from its declaration by a blank line — it is **not** a docstring, which
    is the rule that differs from the header rule;
  - a method — `Owner` is the receiver type with any `*` stripped, and `Name` stays the bare
    identifier;
  - a declaration inside a function body — it is **not** listed;
  - a multi-line function signature — the whole signature is returned, not just its first line;
  - `type X struct` with fields — the signature is the declaration up to the opening brace and the
    field body is **not** in the signature;
  - a type with no body (`type ID string`) — the whole spec is the signature;
  - a type alias (`type Alias = T`) — it is listed, with the whole spec as the signature;
  - an interface type — the signature stops at the opening brace and the method set is excluded;
  - a grouped `type ( ... )` block — one symbol per spec, each with its own range, each signature
    carrying the `type` keyword, and a comment on the `type (` line itself attributed to no spec;
  - a **single-spec** grouped block — `type (\n\tX int\n)` — one symbol whose range is the spec's own
    line and whose signature is `type X int`, not the whole parenthesised block. This is the
    presence-of-`(` assertion: a spec-count branch passes every other grouped case here and fails
    only this one;
  - several symbols in one file — `Symbols` is ascending by `Start`.
  `SigEnd` cases:
  - a docstring plus a single-line signature with a block body — `SigEnd` is the signature's line;
  - a multi-line signature — `SigEnd` is the **last** signature line, the one carrying the `{`, not
    the first;
  - a type with a struct body — `SigEnd` is the line the opening brace sits on;
  - a type alias and a bodyless defined type — `SigEnd` is `0`, so the key is omitted;
  - a single-line function (`func f() int { return 1 }`) — `SigEnd` equals `Start`-adjusted
    declaration line and therefore includes the body. Assert the documented behaviour explicitly
    rather than treating it as a bug, so a later reader finds the intent asserted rather than
    inferred.
  `Package` cases: a normal file — the package clause's identifier; a fixture with no package clause
  — the empty string.
  Header cases:
  - a header separated from `package` by a blank line — still found;
  - a file with both a file header and a package doc comment — the *first* block wins;
  - a file with no header at all — the header is empty and symbols are still returned;
  - a file starting with a build constraint, a blank line, then the header — the header is returned
    and the build constraint is not;
  - a file whose only leading block is a build constraint — no header;
  - a block mixing a `go:generate` line with prose — treated as a header, not skipped;
  - a file starting with a generated-code banner then a real header — the banner is skipped as a
    header **and** `Generated` still reports true from it.
  Error tolerance: a fixture whose broken declaration swallows a later valid one — assert the callback
  observes `partial == true` and that the surviving symbols are returned. This test documents that
  recovery is lossy rather than merely incomplete.
- **Commit:** `test(toc): cover Go symbol, header and range extraction`

## Batch Tests

`verify: go test ./internal/quarryengine/toc` runs the whole toc package, which is exactly what this
batch touches — the new `nodes.go` and `golang.go` plus the batch-2 helpers they call. Nothing
outside that package changes.

New test file: `internal/quarryengine/toc/golang_test.go`. The batch-2 test files still run in the
same package and act as a regression check on the helpers this batch is the first real consumer of.

Every fixture is an inline string constant. No fixture is read from disk and no test writes a file,
so the package stays hermetic and parallel-safe.
