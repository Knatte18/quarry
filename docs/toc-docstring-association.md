# Per-language docstring-association survey — `quarry toc`

**Task:** `toc-verbs`.
This is a standalone reference to the tree-sitter node shapes the toc extraction strategies read.
A reader adding a sixth language should be able to work from this document alone, without reading any
existing strategy's source.

## Status by language

Three languages are **implemented** and ship a concrete `Strategy`: Go, Python, and C#.
Two are **designed but not implemented**: TypeScript and Rust.
An unimplemented language still resolves to a real language name through the extension map, then to
`ErrLanguageUnsupported` — never to a silent empty result.

The three implemented shapes below are recorded as **confirmed facts**: they were established by
dumping real parse trees from the pinned grammars (`tree-sitter-go` v0.25.0, `tree-sitter-python`
v0.25.0, `tree-sitter-c-sharp` v0.23.5), not inferred from documentation.
The two unimplemented shapes are recorded as **design intent**: the shapes a future implementer is
expected to confirm the same way before writing the strategy, not facts already dumped from a tree.

## Go — implemented

- **Declaration node kinds that produce a symbol:** `function_declaration`, `method_declaration`, and
  `type_declaration`, matched as direct children of the `source_file` root.
  A `type_declaration` can hold more than one `type_spec` (or `type_alias`) child in its grouped
  `type ( ... )` form, so it can produce more than one symbol.
  The walk never descends into a `block`, so a func literal or a nested type declared inside a
  function body is never listed.
- **Docstring association:** a Go doc comment is a contiguous run of `comment` prev-siblings of the
  declaration, stopped at the first non-comment sibling or the first blank line — detected as the
  previous comment's last line not being immediately followed by the current node's first line.
  This is `CommentBlockAbove`, walked upward from the declaration.
  For the grouped `type ( ... )` form, the grammar makes each spec's own doc comment a comment child of
  the `type_declaration` itself, interleaved with the spec siblings, so the identical prev-sibling walk
  works one level down, starting from the `type_spec`/`type_alias` node rather than the
  `type_declaration`.
- **File header:** the first leading comment block of the `source_file` root — grouped by the same
  blank-line rule (`LeadingBlocks`) — that is not a directive block.
  A directive block is a build constraint (`go:build`, `+build`), a `go:generate` or `go:embed` line, a
  `nolint` line, or a generated-file banner.
  This deliberately differs from docstring association in two ways: it takes the first *non-directive*
  block rather than the block adjacent to `package`, since a build-constraint block can sit between the
  file header and `package` with no comment of its own on the `package` line;
  and it tolerates the blank line `CommentBlockAbove` would treat as a boundary, since a file can carry
  both a file header and a separate package doc comment immediately above `package`, and the header
  rule must return the earlier block.
- **Signature's body-bearing child:** for a function or method, `decl.ChildByFieldName("body")`.
  For a type spec, the body is resolved from the spec's `"type"` field child: the first direct child of
  that type node whose kind is `field_declaration_list` (a struct's body) or the literal `{` (an
  interface's body, which the grammar exposes with no named body node and no `"body"` field at all).
  A type with neither — `type ID string`, `type Alias = T` — has no body-bearing child, and the whole
  spec's text is the signature.
- **`sigend` derivation:** the line the body's opening `{` sits on, i.e. `bodyOnSignatureLine = true` —
  Go always puts the body-opening token on the signature's own last line, for every kind including a
  body-bearing type.
- **Package name:** the `package_identifier` text of the root's `package_clause` child, or `""` when
  the root has no `package_clause` (a partial parse bad enough to lose it).
- **`test` rule:** known; a `_test.go` filename suffix, the toolchain's own convention.
- **`generated` rule:** known; the first leading comment block matches the "Code generated ... DO NOT
  EDIT." banner, checked directive-or-not since a generated banner is a directive block for the header
  rule and a marker here, independently.

## Python — implemented

- **Declaration node kinds that produce a symbol:** `function_definition` and `class_definition`,
  matched as direct children of the module root, each optionally wrapped in a `decorated_definition`.
  A `class_definition` additionally contributes one symbol per `function_definition` found as a direct
  child of its body `block` — the one level of container descent Python needs, since every method is
  nested inside a class block.
  The walk never descends into a `function_definition`'s own body, so a nested helper is never listed.
- **Decorator handling:** `unwrapDecorated` peels a `decorated_definition` into `decl` (its
  `"definition"` field child, read by every extraction rule for kind, name, signature, and docstring)
  and `outer` (the `decorated_definition` node itself, which the emitted range is measured from so the
  decorator line stays inside `start`–`end`).
  For an undecorated node both values are the node itself.
- **Docstring association:** the docstring is **inside** the definition's body, not a sibling of it —
  the one shape that differs structurally from Go and C#.
  It is the definition body's first statement, when that statement is an `expression_statement` whose
  only named child is a `string` node, read through that string's `string_content` child (the text
  between the `string_start`/`string_end` delimiters).
  A leading `comment` child of the body (unusual, but legal) is skipped when looking for the first
  statement — it is not a statement and must not be mistaken for the docstring candidate.
  Because the docstring lives inside the body, Python needs **no `start` adjustment**: `start` is
  simply the declaration's (or, when decorated, the `decorated_definition`'s) own first line, since the
  docstring already falls within that span.
- **File header:** the module docstring, read the same way as a function or class docstring one level
  up — the module root's direct children serve as its own "body" for this purpose.
  When the module has no docstring, the header falls back to the leading comment blocks
  (`LeadingBlocks`), stripped with `#`, returning the first block that is not a directive block (a
  shebang or a PEP 263 coding line, both directives only on the file's first or second physical line).
  A file with both a shebang and a module docstring returns the docstring.
- **Signature's body-bearing child:** `decl.ChildByFieldName("body")` — the `block` following the
  `def ...:` or `class ...:` header.
- **`sigend` derivation:** the body `block`'s start line minus one, i.e.
  `bodyOnSignatureLine = false` — Python's block starts on the line *after* the header line, unlike
  Go's brace.
  Clamped to never fall below the declaration's own first line, so a single-line `def f(): return 1`
  does not emit a `sigend` above `start`.
- **Package or namespace name:** always `""`.
  Python has no package clause inside a source file — package identity is a directory-layout fact
  (`__init__.py`), not something the file itself declares, and this field deliberately does not
  synthesize a name from the filename or parent directory.
- **`test` rule:** known; a `test_` filename prefix or a `_test.py` suffix, pytest's defaults.
- **`generated` rule:** unknown (`known == false`).
  Python has no generated-file banner convention, so the `generated` key is omitted entirely rather
  than emitted as a guessed `false`.

## C# — implemented

- **Declaration node kinds that produce a symbol:** `class_declaration`, `interface_declaration`,
  `record_declaration`, and `struct_declaration` produce a `KindType` symbol each; `method_declaration`,
  `constructor_declaration`, and `destructor_declaration` produce a `KindMethod` symbol each.
  The walk descends through C#'s container nodes: a `file_scoped_namespace_declaration` ("namespace X;")
  is matched but not descended into — the declarations it governs are its *siblings* in the same
  container, which the same walk already visits — a braced `namespace_declaration`'s `"body"` field
  (a `declaration_list`), and a type declaration's own `"body"` field (also a `declaration_list`) when
  descending into a nested type or its members.
  The emitted member set deliberately excludes `property_declaration`, `indexer_declaration`,
  `event_declaration`, `event_field_declaration`, `field_declaration` (state, not a callable),
  `delegate_declaration` and `enum_member_declaration` (neither state nor a callable this package's
  `Kind` vocabulary covers), and `operator_declaration`/`conversion_operator_declaration` (callables,
  but with no `"name"` field this design's name-based extraction can read).
  A nested type inside a `declaration_list` is matched by the type branch and descended into like any
  other type; a declaration inside a member's own `block` is never reached, since the walk only
  descends into a `declaration_list` or a namespace body, never a `block`.
- **Docstring association:** `///` XML-doc comment lines are a contiguous run of `comment` prev-siblings
  of the declaration — the identical `CommentBlockAbove` walk Go uses — with the XML tags then stripped
  from the joined text (`StripXMLDocTags`), keeping the tags' text content.
- **File header:** the first leading comment block of the root (`LeadingBlocks`) that is not a directive
  block, stripped with `///` then `//` so both the XML-doc and plain comment forms are handled.
  A directive block is one containing `<auto-generated`, checked **before** XML-tag stripping runs:
  C#'s directive rule matches the literal `<auto-generated` tag text, so checking after tag removal
  would make the check permanently unreachable.
  The block returned as the header has `StripXMLDocTags` applied only after the directive check passes.
- **Signature's body-bearing child:** `decl.ChildByFieldName("body")`.
  For a bodyless declaration (a positional record, or an interface method with no implementation) the
  whole declaration's text ends in a trailing `;` that the shared signature-cut helper does not know
  about, so C#'s own signature builder trims it.
- **`sigend` derivation:** read off the body node's own `Kind()`, per member: `"block"` and
  `"declaration_list"` hold the body-opening `{` on the signature's own last line
  (`bodyOnSignatureLine = true`); `"arrow_expression_clause"` (an expression-bodied `=>` member) starts
  on the line after the signature ends, before the `=>` (`bodyOnSignatureLine = false`); a `nil` body
  has no body-opening token, and `SigEnd` short-circuits to `0` before ever consulting the flag.
  The `false` branch is clamped the same way Python's is.
- **Package or namespace name:** the `"name"` field of root's `file_scoped_namespace_declaration` when
  there is one, otherwise the `"name"` field of the outermost `namespace_declaration`, otherwise `""`
  for a file in the global namespace.
  A dotted namespace is returned whole, exactly as written.
  When a file declares more than one braced namespace at the root, the first one in source order is
  returned — a deliberate simplification, since the field is one per file.
- **`test` rule:** unknown (`known == false`).
  C# test-ness lives in attributes or a project file referencing a test SDK; a `Tests.cs`-shaped
  filename is style, not a rule the toolchain enforces.
- **`generated` rule:** known; whether the first leading comment block's raw text contains
  `<auto-generated`, the same substring the header rule's directive check matches, read from the raw
  (not yet delimiter-stripped) text since the marker check and the header-skip check are independent
  readings of the same block.

## TypeScript — designed, not implemented

- **Declaration node kinds that produce a symbol:** not yet confirmed against a real parse tree.
  The design expects `function_declaration`, `method_definition` (inside a `class_declaration`'s body),
  and `class_declaration`/`interface_declaration` to be the analogous set to Go's and C#'s, but this has
  not been verified the way the three implemented languages' shapes were.
- **Docstring association:** `/** ... */` JSDoc block comments as `comment` prev-siblings of the
  declaration — the same sibling-adjacency shape Go and C# use, not the in-body shape Python uses.
  A future implementer should expect to reuse `CommentBlockAbove` largely unchanged, once the JSDoc
  delimiter-stripping rule (`/**`, leading `*` per line, `*/`) is written.
- **File header:** expected to follow the same leading-block-walk shape as Go and C#: the first
  non-directive leading comment block, stripped of JSDoc delimiters.
  No TypeScript-specific directive convention has been designed yet; a future implementer will need to
  decide what a TypeScript directive block is (a shebang for a `.ts` script, a `// @ts-nocheck` pragma,
  or a generated-file banner convention such as `// GENERATED FILE`) before this rule can be written for
  real.
- **Signature's body-bearing child:** expected to be a `statement_block` field, analogous to Go's and
  C#'s `"body"` field — not yet confirmed.
- **`sigend` derivation:** not yet designed.
  The design intent is `bodyOnSignatureLine = true` by analogy with Go and C#'s block-bodied case
  (TypeScript's `{` sits on the signature's own last line the same way), but this has not been checked
  against a real tree.
- **Package or namespace name:** not designed.
  TypeScript has no package clause the way Go and C# do; a future implementer should decide whether
  this returns `""` unconditionally, the way Python does, or reads something else (a `namespace`
  declaration, an ES module's own path) before implementing `Package`.
- **`test` and `generated` rules:** the extension map's registered `TestFileByName` case for
  `"typescript"` already exists (`known == true`; `.test.ts`, `.test.tsx`, `.spec.ts`, `.spec.tsx`
  suffixes, the jest/vitest defaults) even though no strategy is registered to call it yet.
  `GeneratedByBanner` has no `"typescript"` case; a future implementer must add one, or Generated
  returns `known == false` for TypeScript by default, same as Python and Rust.

## Rust — designed, not implemented

- **Declaration node kinds that produce a symbol:** not yet confirmed against a real parse tree.
  The design expects `function_item` and `impl_item` (whose block holds the `fn` items that become
  methods, analogous to C#'s type-then-member descent) to be the working set, plus `struct_item`,
  `enum_item`, and `trait_item` for `KindType`.
- **Docstring association — the genuine trap:** Rust has two distinct doc-comment forms, and confusing
  them is the specific mistake this section exists to prevent.
  `///` is an **outer** doc comment: like Go's and C#'s, it documents the item that *follows* it, and
  is expected to behave as a `comment` prev-sibling the same way.
  `//!` is an **inner** doc comment: it documents the *enclosing* item, not the item that follows it —
  the form Rust file headers and module-level documentation are written with.
  A prev-sibling walk that treats `//!` the same as `///` would misattribute a file's own header
  comment to the first declaration inside it, or vice versa.
  A future implementer must branch on the comment text's own prefix (`///` vs `//!`) before deciding
  which node the comment documents, not just on adjacency.
- **File header:** expected to be built from leading `//!` lines specifically, not from `///` lines —
  the inverse of every other implemented language's header rule, which reads whichever comment sits
  immediately before the first declaration.
  A leading run of `///` comments immediately followed by a declaration is that declaration's
  docstring, not the file's header, even though it sits at the top of the file.
- **Signature's body-bearing child:** expected to be a `block` field on `function_item`, analogous to
  Go's `"body"` — not yet confirmed.
- **`sigend` derivation:** design intent is `bodyOnSignatureLine = true` by analogy with Go (Rust's `{`
  sits on the signature's own last line), not yet checked against a real tree.
- **Package or namespace name:** not designed.
  Rust's module system (`mod`, the crate root) does not map onto a single per-file "package" the way Go
  or C# does; a future implementer should decide whether this returns `""` unconditionally or attempts
  to read a `mod` declaration before implementing `Package`.
- **`test` and `generated` rules:** neither `TestFileByName` nor `GeneratedByBanner` has a `"rust"`
  case.
  Rust test-ness is normally an inline `#[test]` attribute on a function inside the same file rather
  than a naming convention, so `TestFileByName` may never gain a meaningful Rust case the way Go's does
  — `known == false` may be the permanent, correct answer rather than a placeholder.
  Rust has no standard generated-file banner convention either; `GeneratedByBanner` is expected to stay
  `known == false` for `"rust"`.

## Adding a sixth language

Confirm every shape above against a real dumped parse tree before writing the strategy — do not infer
node kinds or field names from a grammar's documentation or from another language's shape.
The two unimplemented sections above are design intent, not confirmed fact, and the first thing their
eventual implementer must do is replace "expected" and "not yet confirmed" with the same kind of
concrete node-kind and field-name statements the three implemented sections make.
