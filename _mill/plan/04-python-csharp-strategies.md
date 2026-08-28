# Batch: python-csharp-strategies

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "python-csharp-strategies"
number: 4
cards: 6
verify: go test ./internal/quarryengine/toc
depends-on: [3]
```

## Batch Scope

This batch adds the two remaining implemented strategies, chosen because their docstring placements
are structurally unlike Go's and unlike each other: Python's docstring is a string literal *inside*
the body, and C#'s is a sibling comment block carrying XML markup, nested two container levels deep.
Together with Go they are what proves the `Strategy` interface generalizes rather than describing Go.

No new shared helper is introduced here — if either strategy needs one, that is a signal the batch-3
helpers were wrong, and the helper is fixed in `nodes.go` rather than duplicated.

Batch-local decisions, established by dumping real parse trees from the pinned grammars:

- Python's module docstring is an `expression_statement` wrapping a `string` node, whose text lives
  in a `string_content` child between `string_start` and `string_end`. The same shape is the first
  statement of a definition's `block` for a function or class docstring.
- A decorated Python definition is wrapped in a `decorated_definition` node. Unwrapping it is
  required, not optional: without it, every decorated function in a file is silently dropped.
- C# `record_declaration` for a positional record has no body at all and ends in a `;`, which must be
  trimmed off the signature.
- C# members reach their body through the `body` field, which is a `block` for a block-bodied member
  and an `arrow_expression_clause` for an expression-bodied one — so one `ChildByFieldName("body")`
  call covers both, and the expression is excluded either way.

## Cards

### Card 22: Python symbol extraction

- **Context:**
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/golang.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/python.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create the Python strategy as an unexported `pythonStrategy` registered from this
  file's `init`, with `Language()` returning `"python"`.
  `Symbols` walks the `module` root's children and, for each `class_definition`, descends into its
  `body` field (a `block`) and walks that block's children too — that is the container descent the
  design requires, since in Python every method is nested inside a class block. It never descends into
  a `function_definition`'s body, so a nested helper is not listed.
  Kinds:
  - `class_definition` — `KindType`, `Owner` empty. Its body-bearing child is
    `ChildByFieldName("body")`.
  - `function_definition` at module level — `KindFunction`, `Owner` empty.
  - `function_definition` inside a class block — `KindMethod`, `Owner` is the enclosing class's name.
  Add `unwrapDecorated(n *ts.Node) (decl *ts.Node, outer *ts.Node)`: when `n.Kind()` is
  `decorated_definition`, return its `definition` field child as `decl` and `n` itself as `outer`;
  otherwise return `n` for both. `decl` is what the kind, name, signature and docstring are read
  from; `outer` is what the range is measured from, so a decorator line is inside the emitted range.
  Without this unwrap every decorated definition is dropped, which is the failure mode to guard
  against.
  `Name` is the `name` field child's text. `Signature` is `SignatureCut(decl, decl.ChildByFieldName("body"), src)`,
  which yields the whole `def ...:` header including a multi-line parameter list.
  `SigEnd` is `SigEnd(decl, decl.ChildByFieldName("body"), false)` — the `false` flag is what makes
  the helper subtract a line, because Python's `block` starts on the line *after* the `def ...:`
  header. Passing `true` here would leak the body's first line into every Python signature range,
  which is exactly the cross-language trap the flag exists to prevent.
  `Docstring` is the first statement of `decl`'s `body` block when that statement is an
  `expression_statement` whose only named child is a `string`: take that string's `string_content`
  child's text, then normalise it exactly the way the Go and C# docstrings are normalised — **trim
  each line, join the lines with `\n`, and trim the whole result** — rather than a single
  `strings.TrimSpace` over the raw text. A PEP 257 docstring is indented to its `def`, so a whole-text
  trim alone would leave every line but the first carrying that indentation, while Go's `//` and C#'s
  `///` stripping removes it; the shared "docstrings keep the prose and drop the syntax" decision
  requires all three languages to produce the same shape.
  Implement it by calling `StripLineComment(text, "")`: with an empty prefix that function is exactly
  the per-line trim-join-trim rule and nothing else, so the normalisation is literally the same code
  the other two strategies run rather than a parallel reimplementation. Say that in the doc comment,
  since an empty-prefix call reads as odd until the reason is stated, and note the deliberate
  consequence: indentation inside a docstring's code example is not preserved, for Python no more and
  no less than for Go.
  A body whose first statement is anything else has no docstring.
  **The range needs no docstring adjustment.** `Start` is `outer`'s first line and `End` is `outer`'s
  last line, because the docstring is already inside the definition node's own span. State that in the
  method's doc comment: it is the one place Python's rule differs from Go's and C#'s, and a reader
  who has just written those two will reach for the sibling-comment adjustment by reflex.
- **Commit:** `feat(toc): add Python symbol extraction`

### Card 23: Python header, generated and test-file rules

- **Context:**
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/classify.go`
  - `internal/quarryengine/toc/strategy.go`
- **Edits:**
  - `internal/quarryengine/toc/python.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** implement the remaining four `Strategy` methods on `pythonStrategy`.
  `Package` returns `""` unconditionally. Python has no package clause inside a source file — its
  package identity comes from the directory layout and `__init__.py`, which is a filesystem fact
  rather than something the file declares. State that in the method's doc comment, and do **not**
  synthesize a name from the filename or the parent directory: the field reports what the file
  itself declares, and a synthesized value would be indistinguishable from a declared one.
  `Header` prefers the module docstring: the first `expression_statement` child of the `module` root
  whose only named child is a `string`, read through its `string_content` child and normalised with
  the same `StripLineComment(text, "")` call card 22 uses, so a module docstring and a function
  docstring come out shaped identically. This is the same node shape as a function docstring, one
  level up.
  When the module has no docstring, fall back to the leading `comment` blocks: walk `LeadingBlocks`,
  strip each with `StripLineComment(b.Raw, "#")`, and return the first block for which
  `IsDirectiveBlock("python", b.StartLine, stripped)` is false. That fallback is what makes the
  shebang and PEP 263 coding-line cases of `IsDirectiveBlock` reachable, and it means a file with a
  shebang and a module docstring returns the docstring rather than the shebang.
  `Header` returns the text untruncated; `FirstParagraph` is applied by the entry points.
  `Generated` returns `(false, false)` — Python has no generated-file rule, so the key is omitted
  rather than emitted as `false`.
  `TestFile` delegates to `TestFileByName("python", base)`.
- **Commit:** `feat(toc): add the Python header and classification rules`

### Card 24: Python strategy tests

- **Context:**
  - `internal/quarryengine/toc/python.go`
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/golang_test.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/python_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** table-driven tests over inline Python fixtures, following the helper shape
  `golang_test.go` established.
  Cases:
  - a module-level function with a docstring — `Docstring` is the stripped prose and the range is the
    definition's own span, asserting explicitly that `Start` is the `def` line and **not** adjusted
    upward, because the docstring is in-body;
  - a function with no docstring — `Docstring` is empty;
  - a class — `KindType`, with its own docstring taken from the first statement of its block;
  - a method inside a class — `KindMethod`, `Owner` is the class name, `Name` is the bare method
    name, and it is found at all (the container descent);
  - a nested function inside a function body — **not** listed;
  - a decorated function — listed, with the range starting at the decorator line;
  - a multi-line `def` signature — the whole signature through the closing `):` is returned, and
    `SigEnd` is the line carrying that closing `):`, not the `def` line;
  - a single-line `def` with a docstring on the next line — `SigEnd` is the `def` line and the body is
    **not** included, which is the case the subtract-a-line rule exists for;
  - a single-line `def f(): return 1` — `SigEnd` equals the `def` line rather than the line above it,
    proving the clamp;
  - a class — `SigEnd` is the `class ...:` line;
  - several symbols — ascending by `Start`.
  `Package` case: any fixture — the empty string, asserted explicitly rather than left unchecked, so
  a later "improvement" that derives a module name from the filename fails this test.
  Header cases:
  - a file with a shebang, then a coding line, then a module docstring — the docstring is the header;
  - a file with a module docstring only — that is the header;
  - a file with no docstring and a prose comment block — the comment block is the header;
  - a file with no docstring and only a shebang — no header;
  - a file with neither — no header, and symbols still returned.
  Classification: assert `Generated` reports `known == false` for every Python fixture, and that
  `TestFile` reports `known == true` for both pytest-default name shapes.
- **Commit:** `test(toc): cover Python symbol, header and range extraction`

### Card 25: C# symbol extraction

- **Context:**
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/golang.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/csharp.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create the C# strategy as an unexported `csharpStrategy` registered from this
  file's `init`, with `Language()` returning `"csharp"`.
  `Symbols` walks the `compilation_unit` root and descends through container nodes, in this order:
  - `file_scoped_namespace_declaration` — a `namespace X;` statement whose following declarations are
    its **siblings**, not its children, so the walk simply continues over the root's remaining
    children rather than descending;
  - `namespace_declaration` — braced, with a `body` field holding a `declaration_list`; descend into
    it;
  - a type declaration (`class_declaration`, `interface_declaration`, `record_declaration`,
    `struct_declaration`) — emit it as `KindType`, then descend into its `body` field
    (a `declaration_list`) when it has one, to reach its members.
  Members inside a `declaration_list` are emitted as `KindMethod` with `Owner` set to the enclosing
  type declaration's name. The emitted set is **closed** and consists of exactly these five node
  kinds — the named callables:
  - `method_declaration`
  - `constructor_declaration`
  - `destructor_declaration`

  All three carry a `name` field in this grammar (verified against the pinned
  tree-sitter-c-sharp v0.23.5, not assumed), so the single `ChildByFieldName("name")` rule below
  covers the whole set with no per-kind naming branch.

  Every other member kind is **deliberately excluded** and produces no symbol. Name the exclusions
  explicitly in a comment at the switch, so a later reader sees they are decided rather than
  forgotten: `property_declaration`, `indexer_declaration`, `event_declaration`,
  `event_field_declaration`, `field_declaration`, `delegate_declaration`, `enum_member_declaration`,
  `operator_declaration`, and `conversion_operator_declaration`. Two distinct reasons drive the cut,
  and the comment must give both:
  - a property, indexer, event, or field is **state, not a callable** — and the accessor-bearing
    forms would additionally need a `SigEnd` rule for `accessor_list` that nothing else in this
    design requires;
  - `operator_declaration` and `conversion_operator_declaration` are callables, but neither has a
    `name` field — an operator's identity lives in an `operator:` field holding the symbol, and a
    conversion operator has no name node at all, only a `type:` field. Emitting them would require a
    bespoke name-synthesis rule for a construct the `kind` vocabulary (`function`/`method`/`type`)
    has no room for, so they are omitted rather than half-named.

  A nested type declaration inside a `declaration_list` is not a member: it is matched by the type
  branch above and emitted as `KindType`, then descended into like any other type.

  A declaration inside a member's body is never reached, because the walk descends only into
  `declaration_list` and namespace bodies, never into a `block`.
  `Name` is the `name` field child's text.
  **The body-bearing child is always `ChildByFieldName("body")`** — the grammar sets that field to a
  `declaration_list` for a type, a `block` for a block-bodied member, and an
  `arrow_expression_clause` for an expression-bodied (`=>`) member. One call therefore covers all
  three, and the expression body is excluded from the signature exactly as a block body is.
  When `body` is nil — a positional `record` declaration, or an interface method with no
  implementation — the signature is the whole declaration text with any trailing `;` trimmed. Add
  that trim; without it every bodyless C# signature ends in a stray semicolon.
  `SigEnd` depends on which body shape was found, and this is the one language where the flag is not
  constant: pass `true` when the body node's kind is `block` or `declaration_list` (the `{` sits on
  the signature's last line), and `false` when it is `arrow_expression_clause` (the signature ends
  before the `=>`). A nil body yields `0`, so a bodyless declaration omits the key.
  Read the flag off the body node's `Kind()` rather than off the declaration's, and give it its own
  named helper so the branch is stated once for every C# kind.
  `Docstring` is `StripXMLDocTags(StripComment(raw, "///"))` over `CommentBlockAbove`'s raw text,
  in that order — strip the line prefix first, then the tags, since the tags are inside the prose the
  prefix strip exposes. `Start` is the comment block's first line when a block was found and the
  declaration's first line otherwise; `End` is the declaration's last line.
- **Commit:** `feat(toc): add C# symbol extraction`

### Card 26: C# header, generated and test-file rules

- **Context:**
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/comments.go`
  - `internal/quarryengine/toc/classify.go`
  - `internal/quarryengine/toc/strategy.go`
- **Edits:**
  - `internal/quarryengine/toc/csharp.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** implement the remaining four `Strategy` methods on `csharpStrategy`.
  `Package` returns the namespace name: the `name` field of the root's
  `file_scoped_namespace_declaration` when there is one, otherwise the `name` field of the outermost
  `namespace_declaration`, otherwise `""` for a file in the global namespace. A qualified name such
  as a dotted namespace is returned whole, exactly as written.
  When a file declares more than one braced namespace at the root, take the first in source order and
  say in the doc comment that this is a deliberate simplification: the field is one per file, so a
  multi-namespace file gets the first one rather than a synthesized list.
  `Header` walks `LeadingBlocks`, strips each block with `StripComment(b.Raw, "///")` followed by
  `StripComment(result, "//")` so both the XML-doc and the plain comment forms are handled, then
  applies `StripXMLDocTags`, and returns the first block for which
  `IsDirectiveBlock("csharp", b.StartLine, stripped)` is false. An auto-generated block is
  therefore skipped as a header and the next block is taken.
  `Header` returns the text untruncated; `FirstParagraph` is applied by the entry points.
  `Generated` reads the raw text of the first leading block and delegates to
  `GeneratedByBanner("csharp", raw)`, so the same block that was skipped as a header is still
  consumed as a marker.
  `TestFile` returns `(false, false)` — C# has no reliable rule. Test-ness lives in attributes or in a
  project file referencing a test SDK, and a `Tests.cs`-shaped filename is style, not a rule. The key
  must therefore be omitted, never emitted as `false`; say so in the method's doc comment, because
  this is the single rule most likely to rot into a best-effort `false`.
- **Commit:** `feat(toc): add the C# header and classification rules`

### Card 27: C# strategy tests

- **Context:**
  - `internal/quarryengine/toc/csharp.go`
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/strategy.go`
  - `internal/quarryengine/toc/types.go`
  - `internal/quarryengine/toc/golang_test.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/toc/csharp_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** table-driven tests over inline C# fixtures, following the helper shape
  `golang_test.go` established.
  Cases:
  - a class with an XML doc comment — the docstring has both the `///` prefix and the `<summary>` and
    `<param>` tags removed while their text is kept, and the range starts at the comment's first line;
  - a method inside a file-scoped namespace, inside a class — found, `KindMethod`, `Owner` is the
    class name (this is the two-level container descent);
  - the same inside a braced `namespace { ... }` — also found, proving both namespace forms are
    descended;
  - an expression-bodied member (`=>`) — the signature stops before the `=>` and the expression is
    excluded, and `SigEnd` is the signature's last line;
  - a multi-line expression-bodied member whose `=>` sits on its own line — `SigEnd` is the line
    before the `=>`;
  - a single-line expression-bodied member — `SigEnd` equals the declaration's line rather than the
    line above it, proving the clamp;
  - a block-bodied member with a multi-line parameter list — the whole signature is returned, and
    `SigEnd` is the line carrying the `{`, not the first signature line;
  - a positional `record` declaration with no body — the signature is the declaration with no trailing
    semicolon, and `SigEnd` is `0` so the key is omitted;
  - an `interface` declaration and its method — both listed, with the interface as `KindType`;
  - a declaration inside a method body — **not** listed;
  - a class carrying one of each emitted member kind — a method, a constructor, and a destructor —
    all three listed as `KindMethod` with `Owner` set to the class and with their names resolved
    (`C` for the constructor and the destructor);
  - a class carrying an auto-property with `{ get; set; }`, a `field_declaration`, an
    `event` field, a `delegate_declaration`, an `operator +`, and an `explicit operator` conversion —
    **none** of them listed. This is the closed-set assertion: it is what fails if someone later
    widens the member switch without revisiting this design;
  - a nested `class` inside a class — listed as `KindType`, not as a member;
  - a member with no doc comment — `Docstring` is empty and the range starts at the declaration;
  - several symbols — ascending by `Start`.
  Header cases:
  - a file whose first block is an `<auto-generated>` block followed by a real `///` header — the
    header is the second block **and** `Generated` still reports true from the first;
  - a file with only a `///` header — that is the header;
  - a file with no leading comment — no header, symbols still returned.
  `Package` cases: a file-scoped `namespace X.Y;` — `X.Y`; a braced `namespace X { }` — `X`; a file
  with no namespace at all — the empty string.
  Classification: assert `TestFile` reports `known == false` for a `Tests.cs`-shaped name, which is
  the explicit omission case, not merely a false answer.
- **Commit:** `test(toc): cover C# symbol, header and range extraction`

## Batch Tests

`verify: go test ./internal/quarryengine/toc` runs the whole toc package. Both new strategies live
there, and re-running the batch-2 and batch-3 tests alongside them is the point: these two strategies
are the first consumers of `nodes.go` other than Go, so a helper that turned out to be Go-shaped
surfaces here rather than later.

New test files: `internal/quarryengine/toc/python_test.go`,
`internal/quarryengine/toc/csharp_test.go`.

Every fixture is an inline string constant; no test reads or writes a file.
