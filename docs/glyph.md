# Glyph — a stable name for a source symbol

A glyph names one symbol in a repository so that a plan, a document, or a program can refer to it,
and quarry can turn it back into exactly where that symbol is right now. It is the identity; quarry
supplies the location. The word is from Greek *glyphein*, to carve: a glyph is a mark cut into
stone, which is where quarry's names come from.

This document is the contract. `docs/rewrite-plan.md` says how quarry implements it; Loomyard's plan
format will adopt it for card targets. Anything not stated here is not part of the contract.

## 1. Form

```
glyph  = unit "#" member
member = name *( "." name )
```

- **unit** — the name of the namespace unit the symbol is declared in (§2). It contains no `#`,
  no `/` or `\`, and no whitespace. Everything else, including `.` and `-`, is allowed, since `#`
  is the only separator that matters.
- **member** — the symbol's own name, preceded by the names of the types that enclose it,
  outermost first, joined with `.`. Each `name` is an identifier as the language spells it.
- Glyphs are case-sensitive, and a glyph has exactly one spelling. There is no short form, no long
  form, no alias.

Examples, from Loomyard:

| glyph | what it names |
|---|---|
| `logger#stderrHandlerSnapshot` | package-level function `stderrHandlerSnapshot` in `internal/logger` |
| `logger#dualHandler` | type `dualHandler` |
| `logger#dualHandler.stderr` | method `stderr` on `dualHandler` |
| `render#Renderer.Draw` | method `Draw` on `Renderer` in `internal/reedengine/render` |
| `lyx#run` | function `run` in `cmd/lyx` (a `package main`; the unit is still the directory name) |
| `models#Beta.Inner.handle` | Python method `handle` on class `Inner` nested in class `Beta`, in module `models` |

## 2. The unit

The unit is the language's own namespace boundary, named by its basename:

| language | unit | unit name |
|---|---|---|
| Go | the package directory | the directory's basename — never the package clause (`main` is many directories), never the import path |
| Python | the module file, or the package for symbols declared in `__init__.py` | the file's basename without `.py`, or the package directory's basename |
| C# | the directory (folder = namespace, the usual convention) | the directory's basename |

**Unit names are unique across the repository.** This is the invariant that lets a glyph be short.
quarry checks it on every resolution and refuses to resolve any glyph in a unit whose name occurs
twice, naming both paths, until one is renamed. It is a convention the repository keeps and quarry
enforces, not something quarry assumes. Loomyard, 2026-09-03: 83 Go packages, 0 duplicate basenames.

Why the basename and not the path: the path would put `internal/` in front of nearly every glyph in
every plan file, and a glyph is a name, not a place. Why not the package clause: it is not unique
(`main`), and it is a local alias a Go file can even override at import.

*Open for Python:* a repository that cannot keep module basenames unique (several `models.py`) would
need the dotted module path from the source root as its unit name instead. Decided when a Python
repository first adopts glyphs, not before.

## 3. The member

- A package-level symbol is its name: `logger#newDualHandler`.
- A method or member is `Owner.Name`. The owner is the declaring type's bare name. For Go the
  receiver is half the key — `logger#dualHandler.Handle` and `logger#durableHandler.Handle` are two
  glyphs — and whether the receiver is a pointer is not part of it.
- Nested types chain: `models#Beta.Inner`, `models#Beta.Inner.handle`.
- Type parameters are not part of a glyph: `Box[T]` is `Box`; `List<T>` is `List`.
- An interface's methods are members of the interface: `shedengine#ShedProducer.Call`.

Which declarations get a glyph, by language:

| language | glyphed | not glyphed (today) |
|---|---|---|
| Go | package-level `func`, `type`, `const`, `var`; methods; interface methods | struct fields, local declarations |
| Python | module-level `def` and `class`; methods; nested classes | attributes, local functions |
| C# | types (incl. nested), methods, properties, fields | locals, lambdas |

## 4. What a glyph does not carry

File, line, column, signature, kind, build tags, overload parameters, visibility. All of those are
answers, not addresses: quarry returns them for a glyph, and they change while the glyph does not.

A glyph survives every edit that does not rename the symbol or move it out of its unit. In Go that
includes moving a method to another file in the same package. A rename is a new glyph, by design:
the glyph is the name, and a plan's `Rename` card is a pair of glyphs, old and new. Moving a
symbol to another unit is likewise a new glyph.

A glyph may name a symbol that does not exist yet. `resolve` answers `not_found` until it is
written, then `found`; a `Create` card's done-check is exactly that transition.

## 5. Resolution

Given a glyph, quarry parses the unit's source as it is at that moment — no index, no cache, no
daemon — and returns every declaration whose unit, owner chain and name match. Results are ordered
by file and then by start line, so the answer is deterministic.

| status | meaning |
|---|---|
| `found` | exactly one declaration |
| `multipart` | one symbol the language lets be declared in several places: a C# `partial` type or partial method. Every part is returned. Go and Python never produce this |
| `ambiguous` | several *different* declarations match: Go build-tag duplicates, C# overloads, a Python name defined twice in one module, a unit name that occurs in two directories. The candidates are returned, with their files; nothing is chosen |
| `not_found` | no declaration matches |

Resolution never guesses. There is no fuzzy matching, no case folding, no "did you mean". A glyph
that does not resolve is `not_found`, and a caller that wants suggestions asks `map`.

Measured cost (Loomyard, 4-core WSL2, in-process, source read fresh every time): 5–8 ms for a
5-file unit, 24–65 ms for a 35-file unit, ~100 ms for twenty glyphs across five units.

## 6. Writing glyphs down

- In prose: the glyph `render#Renderer.Draw`.
- In JSON: the key is `id`; the value is the glyph. Short on purpose, since it repeats per symbol.
- In quarry's text view: the glyph stands alone, no key.
- On a command line: positional arguments. `quarry resolve logger#dualHandler.stderr render#Renderer`.
- In YAML: safe unquoted inside a token (`- render#Renderer.Draw`); a YAML comment needs whitespace
  before its `#`. In Markdown: safe anywhere but the first column of a line.
- In Go: `type Glyph struct { Unit string; Owner []string; Name string }`, `ParseGlyph(string)
  (Glyph, error)`, `Glyph.String()`. The string form is canonical; the struct is a view of it.

## 7. Relationship to other names

- **Loomyard plan cards** today write `shedrecipe.Lookup` — a glyph with `.` where the `#` goes.
  Adopting glyphs means writing `shedrecipe#Lookup`; the separator alone then splits unit from
  member, and `pkg.Type.Method` needs no package list to parse.
- **Go's own names** (`render.Renderer` in source, `internal/reedengine/render.Renderer` in `go doc`)
  are not glyphs and are not accepted where a glyph is expected.
- **LSP** addresses by file + position. A translator from an LSP location to a glyph is: find the
  enclosing declaration by `map` on that file, take its glyph. From a glyph to a position: `resolve`.
  Both directions are mechanical; neither is lossy in the direction that matters, since the glyph
  never depended on the position.
