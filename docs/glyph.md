# Glyph — a stable name for a source symbol

A glyph names one symbol in a repository so that a plan, a document, or a program can refer to it,
and quarry can turn it back into exactly where that symbol is right now. It is the identity; quarry
supplies the location. The word is from Greek *glyphein*, to carve: a glyph is a mark cut into
stone, which is where quarry's names come from.

This document is the contract. `docs/rewrite-plan.md` says how quarry implements it; Loomyard's plan
format adopts it for card targets. Anything not stated here is not part of the contract.

Go and C# are implemented; Python is not. The Python alphabet is specified here so the form is
known to hold for it, and is implemented when it is wanted (`docs/rewrite-plan.md` §9).

## 1. One form, three alphabets

```
glyph = unit "#" member
```

The form is the same for every language. What may appear in `unit` and in `member` is defined per
language (§2, §3), because each language already has its own compiler-guaranteed way of naming a
namespace and a symbol uniquely, and those ways differ. A glyph borrows the language's own
uniqueness instead of inventing one. `#` is the only separator with a fixed meaning; `.`, `/`, `(`
and `,` mean whatever the language's alphabet says. `member` may be empty: a glyph with nothing
after its `#` is the self form, naming the unit itself rather than a symbol inside it (§3).

Structurally, a glyph contains exactly one `#` and splits there.
Whether the two halves are well-formed is checked against one language's alphabet by `Parse` (§6), and the language is the caller's to say:
the string alone cannot tell a Python module path from a C# namespace, and a symbol a plan is about to create has no source to look at yet.
For an existing symbol the language is what `toc` reported for its file; for a new one, the plan card carries it.

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
| `internal/reedengine/render#` | Go | the package `internal/reedengine/render` itself |
| `internal/reedengine/render/focus.go#` | Go | the file `internal/reedengine/render/focus.go` itself |
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
The C# guarantee is per assembly, not per repository: a repository holding several assemblies can
declare one namespace, and one type name in it, more than once. Such a glyph resolves `ambiguous`
(§5), and nothing is chosen.

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
- **C#, nested and file-scoped namespaces.** `namespace A { namespace B {} }` and `namespace A.B;`
  both give the unit `A.B`, the enclosing names joined with `.`. A unit segment that is a reserved
  keyword is written with the C# escape, `@class`, and a `@` before any other identifier is
  rejected: `@Foo` is `Foo`.
- **C#, no directory rule.** C# does not tie a namespace to a directory, so a namespace may span
  directories and a directory may hold several. The unit comes from the declarations in the
  source, never from the path.

In the self form (§3) the left half is the thing's own repository-relative path or unit name,
whichever the language spells, and what a trailing `#` means differs per language:

| language | a trailing `#` means |
|---|---|
| Go | a package directory or a file, both spelled as their repository-relative path — Go's unit already is one |
| Python | the module or package itself, which is the file; Python has no separate file self glyph |
| C# | the namespace itself; the C# alphabet has no file-level unit, so it has no file self glyph |

A `.cs` file is still addressable as a whole: its path self glyph, such as `Norce.Models/Foo.cs#`,
is spelled by the Go unit alphabet, which spells any repository-relative path. The C# alphabet
itself never produces it.

The external test unit's self glyph is well-formed and parses like any other. It always resolves
`not_found`, with the unit reported as found: the pseudo-unit, `internal/logger_test`, has no
directory of its own, while `internal/logger` — the directory it borrows its symbols from — does
exist.

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

An empty member is the self form (§1), in every alphabet: that much is shared. For Go, and only for
Go, removing the trailing `#` yields the plain repository-relative path in both directions with no
other conversion — this holds only because Go's unit is itself spelled as a repository-relative
path (§2). It does not hold unscoped: a Python dotted module and a C# namespace are not
repository-relative paths, so stripping the trailing `#` from a Python or C# self glyph does not
yield one. This direction is exposed as `UnitPath()` on the parsed `Glyph`, which answers the path
from the unit half and reports whether the glyph's alphabet spells units as paths at all, so no
consumer performs the strip itself.

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

**C#.** `Type.Name` for members, nested types chain, `Outer.Inner.Name`. **A method, constructor,
finalizer, operator or indexer always carries its parameter types**, in parentheses (brackets for an
indexer), comma-separated, no spaces: `Renderer.Draw(int)`, `Renderer.Draw(int,string)`,
`Renderer.Draw()`, `Renderer.Renderer(Box)` for a constructor. Parameter *names*, default values
and attributes are never included; parameter modifiers are, since they distinguish overloads:
`Draw(ref int)`, `Draw(out Box)`, `Draw(params string[])`, `Draw(scoped ref Span<int>)`.
Properties, fields, events and types have no parentheses, since they cannot be overloaded.

Every parameter type is written in one canonical spelling, so that the files of a partial method
and two authors of one call cannot spell one signature two ways:

- whitespace is removed, except one space between two adjacent word tokens (`ref int`, `this Box`);
- every modifier is kept (`ref`, `out`, `in`, `params`, `this`, `scoped`, `ref readonly`);
- the `global::` qualifier is removed;
- a built-in type alias is replaced by its keyword, bare or `System.`-qualified, at any depth
  (`Int32` and `System.Int32` become `int`, `String` becomes `string`, `List<System.Int32>`
  becomes `List<int>`); only a whole dotted name is replaced, so `MyInt32` and `Foo.Int32` stay;
- tuple element names are dropped: `(int a, string b)` is `(int,string)`;
- everything else is as the source writes it: no qualification is added or removed, and a nullable
  `?` is kept (`List<Pane>`, not `System.Collections.Generic.List<Pane>`).

`Parse` rejects a type that is not in this spelling, `Draw(Int32)`, and names the canonical
spelling as the fix; a parameter that still carries its name or default value, `Draw(int a)`, is
rejected the same way. A spelling no strategy mints is a syntax error with its fix, never a silent
`not_found`.

Generic arity is part of the name, because `Foo`, `Foo<T>` and `Foo<T,U>` are three types in one
namespace. It is written the way C#'s own `typeof` writes an open generic type — commas only, no
parameter names: `List<>`, `Dictionary<,>`, and `Draw<>(T)` for a generic method, whose parameter
types are written as declared. The other members C# can declare are spelled as C# spells them:

- an indexer, `Renderer.this[int]`, with its parameter list in brackets;
- an operator, `Box.operator +(Box,Box)`, the operator token taken verbatim (`operator <`,
  `operator >>`, `operator ==`), and `Box.operator checked +(Box,Box)` for a checked one;
- a conversion operator, `Box.implicit operator int(Box)` or `Box.explicit operator int(Box)`,
  with `checked` after the keyword for a checked one and the target type canonical, qualified or
  not (`implicit operator System.Text.Rune(Box)`);
- a finalizer, `Renderer.~Renderer()`;
- a static constructor, `Renderer.static Renderer()`, beside the instance constructors
  `Renderer.Renderer()`, so the two parameterless constructors are two glyphs.

A `@` is written only where C# needs it: before a reserved keyword (`Renderer.@class`). A `@`
before any other identifier is rejected, and the bare spelling is the glyph.

An **explicit interface implementation** carries its interface as a qualifier. The interface type,
canonicalised with its type arguments, is appended to the owner chain as its dotted components:
`int IProvider<A>.Value { get; }` inside `Renderer` is `Renderer.IProvider<A>.Value`, and
`void System.IDisposable.Dispose()` is `Renderer.System.IDisposable.Dispose()`. The type arguments
keep `IProvider<A>.Value` and `IProvider<B>.Value` apart.
*Known:* the qualifier reads like a nested type, so the two can collide. When `Renderer` declares a
nested interface `IHandler` with a method `Handle()` and also implements `IHandler.Handle()`
explicitly, both spell `Renderer.IHandler.Handle()`. The glyph resolves `ambiguous` and nothing is
chosen; `expand` of `Renderer.IHandler` lists both. quarry produces each string unambiguously
from the grammar, but a reader of the string cannot tell which is meant.

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
| C# | types at any nesting depth (class, struct, interface, enum, record, record struct, delegate); methods, finalizers, instance, static and primary constructors, properties and a record's positional parameters, fields (one per declarator), constants and enum members, events, indexers, operators and conversion operators | locals, lambdas, local functions, accessors, members of a C# 14 `extension` block, a class or struct primary constructor's parameters, and any member declared in a namespace body or the compilation unit outside a type |

A C# glyph's kind is one of `type`, `method`, `constructor`, `property`, `field`, `const`, `event`,
`indexer` or `operator`; Go never emits the last five of those words. A record's positional
parameter is a `property` unless the declaring part of the record declares a member of the same
name, and one inherited from a base record is still listed. A declaration whose minted glyph does
not parse, such as a type named with a combining mark, which C# admits and the alphabet's letter,
digit and `_` rule does not, gets no glyph, and its file is reported `lossy`.

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
`namespace` declaration, a receiver or enclosing type as declared, a parameter type in its
canonical spelling.
Nothing requires a type checker, which is what lets a tree-sitter parse produce every glyph in a
repository. That is a tested claim, not a design hope: `docs/rewrite-plan.md` §12 requires a
round trip over a whole repository per language — every declaration `toc` lists resolves back to
exactly its own span.

## 5. Resolution

Given a glyph, quarry parses the unit's source as it is at that moment — no index, no cache, no
daemon — and returns every declaration whose unit, owner chain, name and (C#) parameter types
match. Results are ordered by file and then by start line, so the answer is deterministic.

A C# glyph needs one more step, because C# does not tie a namespace to a directory. Each call that
holds a target parsing as a C# glyph scans the repository's `.cs` files once, under the same ignore
set the walk applies and never following a symlink, so a gitignored file's namespace and symbols
are left out. The scan is a local of that call: each file is read at most once and nothing is kept
afterwards. A call with no C# target scans nothing. A C# unit is found when some file declares it,
and a declared namespace declares every prefix of itself: `namespace A.B;` declares `A.B` and `A`.

| status | meaning |
|---|---|
| `found` | exactly one declaration |
| `multipart` | one symbol the language lets be declared in several places: Go `init`, a C# `partial` type or partial method. Every part is returned. For C# it applies when every match carries the `partial` modifier and all are of one kind; Python never produces this |
| `ambiguous` | several *different* declarations match: Go build-tag duplicates, a C# non-partial duplicate such as the two branches of an `#if`/`#else`, a C# member glyph that an explicit interface qualifier and a nested type both spell (§3), a Python name defined twice in one module, or a match in two alphabets (below). The candidates are returned, with their files; nothing is chosen |
| `not_found` | no declaration matches. A C# method glyph without parentheses is `not_found`, not "all overloads". The answer also says whether the unit exists: `unit: found` when the directory, module or namespace is there and only the member is missing, `unit: not_found` otherwise. A Create card needs the first; a misspelled unit gives the second |

C# parameters are part of the match. A glyph without parentheses matches only a declaration that
has no parameter list (a type, property, field or event), so `Renderer.Draw` matches no method; a
glyph with empty parentheses, `Renderer.Draw()`, matches only a parameterless declaration; and a
parameter list matches element by element. Go glyphs carry no parameters, so Go matching is
unchanged.

The four statuses above are a closed vocabulary a caller can check, not a set of strings to compare
against ad hoc. An absent status names a pre-resolution rejection of the target string itself,
never a resolution outcome.

Resolution never guesses. There is no fuzzy matching, no case folding, no "did you mean". A glyph
that does not resolve is `not_found`; a caller that wants to see what exists asks `toc` or
`expand`.

In a repository with more than one language the glyph is tried against every alphabet that parses it, and each accepting alphabet answers on its own.
A Go unit that is a single path segment can spell the same string as a C# namespace: the directory `Norce.Models` and the namespace `Norce.Models` both give `Norce.Models#Foo`.
If one alphabet matches, its answer stands, exactly as it would in a repository of one language.
If two match a member glyph, the answer is `ambiguous` with the candidates of both, each marked by `language`.
If two match a self glyph, the answer is `ambiguous` with no candidates and a `languages` list naming both in name order.
If none matches, the answer is `not_found`, with `unit: found` when any accepting alphabet found the unit.
There is no language prefix in the glyph to prevent this: the case is rare, and a prefix would be a second spelling.
`expand` applies the same rule to its target, with the raw match sets of both alphabets as the candidates.

A target that no alphabet parses is rejected with one parse error.
It is C#'s when the unit half is a valid C# unit and the member holds a character outside Go's member alphabet (`(`, `[`, `<`, `~`, `@` or a space), so `Foo.Bar#Draw(Int32)` is told its canonical spelling.
It is Go's otherwise, so `a/b#Draw(int)` keeps Go's reason.

`toc` takes paths; `resolve` takes glyphs. A bare repository-relative path handed to `resolve` is
rejected pre-resolution, with a message naming the fix — append the trailing `#` to make it a self
glyph. A self glyph answers with the same listing block `toc` would produce for that path and the
same four statuses above: `found` with the listing, or `not_found`. That is how a whole file — an
HTML viewer, a Markdown page — is a plan target with the same checks as a symbol.

A C# self glyph names a namespace, which spans directories, so its listing is a list.
`Ns#` is `found` exactly when some file declares `Ns`, and the answer carries `listings`: one directory answer per directory with a file holding a symbol in `Ns`, in path order, each reduced to those files and listed without symbols.
A declared namespace with no symbol directly in it, such as `A` when only `A.B` holds symbols, is `found` with `listings: []`.
The `listing` key is absent.
An undeclared namespace is `not_found` with `unit: not_found`.

A directory named like a namespace makes the self glyph collide: `Norce.Models#` is both the Go path self glyph of the directory `Norce.Models` and the C# self glyph of the namespace, so it resolves `ambiguous`.
The way out is to name the file or a member: a path self glyph spelled with the file, `Norce.Models/Foo.cs#`, which only the Go unit alphabet spells, or a member glyph such as `Norce.Models#Foo`.

`name` is the maker verb: given a unit plus a declaration head, it predicts the id and kind that
declaration will get once it is actually written, without reading the repository. On a call to
`resolve` or `name` that returns answers at all, there is one answer per input, in argument order,
with the input echoed verbatim on every answer — including a rejection — and a repeated input is
answered once per occurrence. `resolve` can fail the entire call instead of answering: an engine
failure then returns no answers at all, never a partial or padded slice, while a malformed target
taints only its own answer; `name` has no such path and never fails batch-wide.

A `#` in any path segment is an explicit error at both verbs — `toc` and `resolve` alike reject it,
never reclassifying the target as the other kind. This sits beside an asymmetry that is not a
contradiction: the contract above governs what a caller may *name*, while the walk's own
spellability rule (§2's unit corner cases; `internal/engine/walk.go`'s `unitSpellable`) governs
what it may *mint* — so a directory whose own name happens to carry a `#` is still listed, without
symbols, when it is encountered below a listed target, even though naming it directly as a resolve
or expand target is rejected.

The repository root cannot be addressed by `resolve` at all: a lone `#` fails as an empty unit, and
a `.` segment is rejected as a dot segment. `toc` on the root remains the way to ask what is there.

`expand` of a type lists its members: every symbol of the unit whose owner chain starts with the type's own chain followed by its name, so `Ns#Outer.Inner` lists the members of the nested type and never those of a top-level `Inner`.
A C# type's head is its declaration through its base list and constraints, a strict subset of its span.
A C# `partial` type resolves `multipart`, and its `expand` answers `parts`, one head per part, instead of `head`, with the members of every part sorted by file and line.

`enclose` and `delta` read every implemented language.
A `.cs` file is enclosed on the working tree and at a revision like a Go file, and a line outside every member answers `not_found` with the file's path self glyph as `unit` when the Go unit alphabet spells the path, and with no `unit` when it does not.
`delta` extracts a `.cs` file's symbols with their own namespaces as units, so a unit the caller supplies for a `.cs` entry is ignored, and a side holding a declaration that got no glyph is reported lossy.
Its exact rename tier substitutes the identifier without its arity suffix (`Draw<>` as `Draw`) and never pairs an operator, conversion operator, indexer, finalizer or static constructor, since no identifier names them.
Its evidence tier reports a pair with the same unit, owner chain and kind and either a different name or, for C#, the same name with different parameters, since a parameter change is a new glyph (§4).

Measured cost (Loomyard, 4-core WSL2, in-process, source read fresh every time): 5–8 ms for a
5-file unit, 24–65 ms for a 35-file unit, ~100 ms for twenty glyphs across five units.

## 6. Writing glyphs down, and the one implementation

- In prose: the glyph `internal/reedengine/render#Renderer.Draw`.
- In JSON: the key is `id`; the value is the glyph. Short on purpose, since it repeats per symbol.
- In quarry's text view: the glyph stands alone, no key.
- On a command line: positional arguments.
- In YAML: safe unquoted inside a token (`- cmd/lyx#run`); a YAML comment needs whitespace before
  its `#`. In Markdown: safe anywhere but the first column of a line. C# glyphs contain `(`, `,`
  and `<`; quote them where a format cares. A trailing `#` is safe in every format already listed
  here, and it is never optional: the canonical form keeps it, so a self glyph is never printed or
  accepted with it stripped.
- In Go: package `github.com/Knatte18/quarry/glyph` — pure Go, no cgo, no dependencies, so that
  any program can import it without the engine. `type Language`, with `Go` and `CSharp`
  (`Python` is the name reserved for the alphabet still to come);
  `type Glyph struct { Lang Language; Unit string; Owner []string; Name string; Params []string }`,
  where `Params` is nil for a glyph without parentheses and non-nil, possibly empty, for one with;
  `Parse(lang Language, s string) (Glyph, error)`; `Glyph.String()`;
  `Self(lang Language, path string) (Glyph, error)`, the compose constructor for the self form;
  `Glyph.IsSelf() bool`, reporting whether a Glyph is one; and
  `CanonicalCSharpType(s string) string`, the canonical spelling of §3 for one parameter type,
  total and idempotent. `Parse` is the syntactic
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
- **File paths** are not glyphs, but every path has one: `toc` takes the plain path, and `resolve`
  takes that path's self glyph — the same path with a trailing `#` appended (§1, §3). A plan card
  whose deliverable is a whole file names the path; the self glyph is how `resolve` checks it.
- **LSP** addresses by file + position. From an LSP location to a glyph: `toc` on that file, take
  the enclosing declaration's glyph. From a glyph to a position: `resolve`. Both directions are
  mechanical, and the glyph never depended on the position.
