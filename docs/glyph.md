# Glyph — a stable name for a source symbol

A glyph names one symbol in a repository so that a plan, a document, or a program can refer to it,
and quarry can turn it back into exactly where that symbol is right now. It is the identity; quarry
supplies the location. The word is from Greek *glyphein*, to carve: a glyph is a mark cut into
stone, which is where quarry's names come from.

This document is the contract. `docs/rewrite-plan.md` says how quarry implements it; Loomyard's plan
format adopts it for card targets. Anything not stated here is not part of the contract.

Go is implemented; nothing else is, until a language is wanted. The Python and C# alphabets are
specified here so the form is known to hold for them, and are implemented one at a time, later
(`docs/rewrite-plan.md` §9).

## 1. One form, three alphabets

```
glyph = unit "#" member
```

The form is the same for every language. What may appear in `unit` and in `member` is defined per
language (§2, §3), because each language already has its own compiler-guaranteed way of naming a
namespace and a symbol uniquely, and those ways differ. A glyph borrows the language's own
uniqueness instead of inventing one. `#` is the only separator with a fixed meaning; `.`, `/`, `(`
and `,` mean whatever the language's alphabet says.

Structurally, any glyph splits at its first `#`. That split needs no language.
Whether the two halves are well-formed is checked against one language's alphabet by `Parse` (§6), and the language is the caller's to say:
the string alone cannot tell a Python module path from a C# namespace, and a symbol a plan is about to create has no source to look at yet.
For an existing symbol the language is what `map` reported for its file; for a new one, the plan card carries it.

The alphabet is chosen per file, not per repository.
A repository that holds Python parsers inside a C# tree has Python glyphs and C# glyphs side by side;
nothing in the form says which is which, and nothing needs to (§5 says what happens on the rare textual collision).

Glyphs are case-sensitive and each symbol has exactly one glyph. No short form, no alias.

| glyph | language | names |
|---|---|---|
| `internal/logger#stderrHandlerSnapshot` | Go | package-level function in `internal/logger` |
| `internal/logger#dualHandler.stderr` | Go | method `stderr` on `dualHandler` |
| `internal/reedengine/render#Renderer.Draw` | Go | method `Draw` on `Renderer` |
| `cmd/lyx#run` | Go | function `run` in a `package main` |
| `loomyard.engine.layout#Beta.Inner.handle` | Python | method on class `Inner` nested in `Beta`, module `loomyard/engine/layout.py` |
| `Loomyard.Engine.Layout#Renderer.Draw(int)` | C# | the `Draw` overload taking one `int` |
| `Loomyard.Engine.Layout#Renderer.Title` | C# | property `Title` |

## 2. The unit, per language

| language | unit | spelled as |
|---|---|---|
| Go | the package directory | its path relative to the repository root: `internal/logger`, `cmd/lyx`. For a single-module repository this is the import path without the module prefix, which is what `go doc` accepts |
| Python | the module | its dotted path from the source root, which is what `__module__` holds: `loomyard.engine.layout`; a package's own `__init__.py` symbols use the package: `loomyard.engine` |
| C# | the namespace | as declared: `Loomyard.Engine.Layout` |

Unique by construction in each language: two Go directories cannot share a path, two Python
modules cannot share a dotted path, and C# types are unique within a namespace by the compiler's
rules. Nothing is enforced by quarry and nothing depends on a repository's naming discipline.

Unit corner cases:

- **Go, external test package.** `package logger_test` shares the directory with `package logger`
  and is a second unit. Its unit name is the directory path with `_test` appended,
  `internal/logger_test`, the pseudo-path `go doc` and gopls already use. `_test.go` files in the
  package itself belong to the package's own unit.
- **Python, the source root.** The nearest ancestor directory that is not itself a package (has no
  `__init__.py`); `src/`-layout and flat-layout repositories both satisfy this. *Open:* namespace
  packages without `__init__.py` (PEP 420) defeat the rule; a repository using them needs the root
  declared, which is the one case where a glyph would depend on something outside the source.
- **C#, the global namespace.** A type declared outside any namespace has the unit `global`, as in
  C#'s own `global::`.

The price is that a Go glyph carries `internal/` where Go's own name does. Shorter schemes were
considered and rejected, each for the same reason — the glyph would change without the symbol
changing:

- *directory basename with a uniqueness invariant* — fails on any repository with `internal/logger`
  and `loomyard/logger`, which large repositories have;
- *shortest unique suffix of the path* — a glyph's spelling changes when an unrelated package is
  added elsewhere;
- *numbered or tagged collisions, `logger[1]`, `logger[loomyard]`* — the same drift, plus an
  ordering nothing defines;
- *source roots declared in a repository config file* — makes a glyph's meaning depend on a file
  beside the code, and a config edit silently renames every glyph.

## 3. The member, per language

The member is the symbol's own name, preceded by the names of the types that enclose it, outermost
first, joined with `.`.

**Go.** `Name` for a package-level `func`, `type`, `const` or `var`; `Type.Name` for a method or an
interface method. The receiver type is half the key — `dualHandler.Handle` and
`durableHandler.Handle` are two glyphs — and whether it is a pointer receiver is not part of it.
Go has no nesting and no overloading, so a member never has more than one `.` and never has
parentheses. Type parameters are not part of a glyph: `Box[T]` is `Box` (Go does not allow `Box`
and `Box[T]` to coexist). `func init()` may occur many times per package and Go gives them no
individual identity, only an order: `internal/logger#init` is one glyph, and with several `init`
functions it resolves `multipart`, every one returned in run order (file order, then line).

**Python.** `name` for a module-level `def` or `class`; `Class.name` for a method; nested classes
chain, `Beta.Inner.handle`. No overloading, so no parentheses. A `def` decorated with
`@typing.overload` is a typing declaration of the implementation that follows it, not a second
symbol: the stubs have no glyph and the undecorated implementation is the symbol. Any other
redefinition of a name in one scope is `ambiguous`.

**C#.** `Type.Name` for members, nested types chain, `Outer.Inner.Name`. **A method or
constructor always carries its parameter types**, in parentheses, comma-separated, no spaces, as
written in the declaration: `Renderer.Draw(int)`, `Renderer.Draw(int,string)`, `Renderer.Draw()`,
`Renderer.Renderer(Box)` for a constructor. Parameter *names* and default values are never
included; parameter modifiers are, since they distinguish overloads: `Draw(ref int)`,
`Draw(out Box)`, `Draw(params string[])`. Type names are never qualified beyond how the source
writes them (`int`, not `System.Int32`; `List<Pane>`, not `System.Collections.Generic.List<Pane>`).
Properties, fields, events and types have no parentheses, since they cannot be overloaded.

Generic arity is part of the name, because `Foo`, `Foo<T>` and `Foo<T,U>` are three types in one
namespace. It is written the way C#'s own `typeof` writes an open generic type — commas only, no
parameter names: `List<>`, `Dictionary<,>`, and `Draw<>(T)` for a generic method, whose parameter
types are written as declared. The other members C# can declare are spelled as C# spells them:
an indexer `Renderer.this[int]`, an operator `Box.operator +(Box,Box)`, a finalizer
`Renderer.~Renderer()`. Built-in type aliases are canonicalised to the keyword by a fixed table
(`Int32` → `int`, `String` → `string`), so the two files of a partial method cannot spell one
signature two ways. *Known:* an explicit interface implementation, `void ILayout.Draw(int)` inside
`Renderer`, yields `Renderer.ILayout.Draw(int)`, which reads like a nested type; the grammar has a
distinct node for it, so quarry produces it unambiguously, but a reader of the string cannot tell.

The parentheses are always present, not only when an overload exists, because a name-only glyph
would stop being unique the day someone else adds an overload — and then nothing says which of the
two the old glyph meant. With the parameter types always present, `Draw(int)` is the same glyph
before and after `Draw(double)` appears.

*Open, C# only:* a method with many parameters yields a long glyph. Measured against a real C#
repository's parameter-count distribution before anything is done about it. If a cap is ever
adopted it is the first N parameter types plus a short hash of the complete canonical list,
with N fixed by this document — never abbreviations of type names, which are unreadable and not
unique. Since the full list is what any hash would be computed from, adopting a cap later changes
nothing already written.

Which declarations get a glyph:

| language | glyphed | not glyphed (today) |
|---|---|---|
| Go | package-level `func`, `type`, `const`, `var`; methods; interface methods | struct fields, local declarations |
| Python | module-level `def` and `class`; methods; nested classes | attributes, local functions |
| C# | types (incl. nested), methods, constructors, properties, fields, events, indexers, operators | locals, lambdas |

## 4. What a glyph does not carry

File, line, column, kind, return type, visibility, build tags, doc. All of those are answers, not
addresses: quarry returns them for a glyph, and they change while the glyph does not.

A glyph survives every edit that does not rename the symbol, move it out of its unit, or (C#)
change its parameter types. In Go that includes moving a method to another file in the same
package. A rename is a new glyph, by design: the glyph is the name, and a plan's `Rename` card is a
pair of glyphs, old and new. Moving a symbol to another unit is likewise a new glyph.

A glyph may name a symbol that does not exist yet. `resolve` answers `not_found` until it is
written, then `found`; a `Create` card's done-check is exactly that transition.

Every element of every alphabet is chosen to be syntactic: a directory, a `package` or
`namespace` declaration, a receiver or enclosing type as declared, a parameter type as written.
Nothing requires a type checker, which is what lets a tree-sitter parse produce every glyph in a
repository. That is a tested claim, not a design hope: `docs/rewrite-plan.md` §12 requires a
round trip over a whole repository per language — every declaration `map` lists resolves back to
exactly its own span.

## 5. Resolution

Given a glyph, quarry parses the unit's source as it is at that moment — no index, no cache, no
daemon — and returns every declaration whose unit, owner chain, name and (C#) parameter types
match. Results are ordered by file and then by start line, so the answer is deterministic.

| status | meaning |
|---|---|
| `found` | exactly one declaration |
| `multipart` | one symbol the language lets be declared in several places: Go `init`, a C# `partial` type or partial method. Every part is returned. Python never produces this |
| `ambiguous` | several *different* declarations match: Go build-tag duplicates, a Python name defined twice in one module. The candidates are returned, with their files; nothing is chosen |
| `not_found` | no declaration matches. A C# method glyph without parentheses is `not_found`, not "all overloads". The answer also says whether the unit exists: `unit: found` when the directory, module or namespace is there and only the member is missing, `unit: not_found` otherwise. A Create card needs the first; a misspelled unit gives the second |

Resolution never guesses. There is no fuzzy matching, no case folding, no "did you mean". A glyph
that does not resolve is `not_found`; a caller that wants to see what exists asks `map` or
`members`.

In a repository with more than one language the glyph is tried against each alphabet present.
Go units contain `/` and collide with nothing.
A Python module path and a C# namespace can spell the same string, and when a declaration matches in both the answer is `ambiguous` with the candidates marked by language.
There is no language prefix in the glyph to prevent this: the case is rare, and a prefix would be a second spelling.

`resolve` also accepts what is not a glyph: a repository-relative path, file or directory, told apart by having no `#`.
It answers with the file entry or directory answer and `found` or `not_found`.
That is how a whole file — an HTML viewer, a Markdown page — is a plan target with the same checks as a symbol.

Measured cost (Loomyard, 4-core WSL2, in-process, source read fresh every time): 5–8 ms for a
5-file unit, 24–65 ms for a 35-file unit, ~100 ms for twenty glyphs across five units.

## 6. Writing glyphs down, and the one implementation

- In prose: the glyph `internal/reedengine/render#Renderer.Draw`.
- In JSON: the key is `id`; the value is the glyph. Short on purpose, since it repeats per symbol.
- In quarry's text view: the glyph stands alone, no key.
- On a command line: positional arguments.
- In YAML: safe unquoted inside a token (`- cmd/lyx#run`); a YAML comment needs whitespace before
  its `#`. In Markdown: safe anywhere but the first column of a line. C# glyphs contain `(`, `,`
  and `<`; quote them where a format cares.
- In Go: package `github.com/Knatte18/quarry/glyph` — pure Go, no cgo, no dependencies, so that
  any program can import it without the engine. `type Language`, with `Go` alone until a second
  language is added (`Python` and `CSharp` are the names reserved for the alphabets below);
  `type Glyph struct { Lang Language; Unit string; Owner []string; Name string; Params []string }`;
  `Parse(lang Language, s string) (Glyph, error)`; `Glyph.String()`. `Parse` is the syntactic
  check: it reads no source and accepts exactly the alphabet of `lang`. The string form is
  canonical; the struct is a view of it.

**There is one implementation of the glyph grammar, and it is this package.** Loomyard's plan
parser imports it; it does not re-implement parsing, printing or canonicalisation. That is what
makes any later refinement of the alphabet — a C# parameter cap, say — a change in one place.

## 7. Relationship to other names

- **Loomyard plan cards** today write `shedrecipe.Lookup` — package clause, dot, name. As a glyph
  that is `internal/shedrecipe#Lookup`: the unit is the directory path, and `#` splits unit from
  member so no parser needs a package list to tell `pkg.Type.Method` from `pkg.Name`.
- **Go's own names** (`render.Renderer` in source, `internal/reedengine/render.Renderer` in
  `go doc`) are not glyphs and are not accepted where a glyph is expected: the `.` there is
  ambiguous between path, package, type and member, and quarry does not try alternatives.
- **C# XML documentation IDs** (`M:Loomyard.Engine.Layout.Renderer.Draw(System.Int32)`) solve the
  same problem for the same reason and differ only in spelling: a kind prefix, fully qualified
  types, and no unit separator. A C# glyph is the readable form of the same key.
- **File paths** are not glyphs. A repository-relative path has no `#`, and that is how `resolve`
  tells the two apart (§5); a plan card whose deliverable is a whole file names the path.
- **LSP** addresses by file + position. From an LSP location to a glyph: `map` on that file, take
  the enclosing declaration's glyph. From a glyph to a position: `resolve`. Both directions are
  mechanical, and the glyph never depended on the position.
