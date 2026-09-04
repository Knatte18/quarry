# Batch: fixtures

```yaml
task: "resolve + expand (T4)"
batch: "fixtures"
number: 2
cards: 2
verify: CGO_ENABLED=1 go test ./internal/engine/
depends-on: []
```

## Batch Scope

This batch commits the two new fixture packages the status and expand tests read: a type whose
methods span two files, and a build-tag pair declaring the same names twice. Both are new sibling
directories under `internal/engine/testdata/`. Every other fixture this plan needs is either an
existing committed package read without modification, or a run-time tree built under `.scratch/` by
the tests themselves; both are batch 3's and batch 4's business, not this one's.

It is one batch because both cards are pure fixture data with one shared risk — perturbing an
existing assertion — and one shared gate: the whole engine test suite must still pass untouched after
they land. Two existing tests see these new packages and both are argued through rather than assumed
safe. `TestRoundTrip_QuarryItself` walks this module's own root with `DepthAll` and symbols on, so it
does enumerate both new packages; it compares the walk's spans against `symbolsOfUnit`'s through
`tupleSetDiff`, a multiset diff that counts duplicates rather than collapsing them, so the build-tag
pair's two identical-named declarations round-trip cleanly — both sides list both, and neither side
evaluates a build constraint. The assertions that enumerate a *specific* directory's children are the
walk's, over `internal/engine/testdata/tree/`, and the symbol-list assertions over
`internal/engine/testdata/glyphs/`; nothing in this batch adds a file to either.

## Cards

### Card 3: testdata/methods, a type with methods across two files

- **Context:**
  - `internal/engine/testdata/tree/pkg/alpha.go`
  - `internal/engine/walk.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/testdata/methods/aardvark.go`
  - `internal/engine/testdata/methods/widget.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create the package `methods` in two files, so a single type's members are split
  across files whose names sort opposite to the file that declares the type. `Widget` is declared in
  `internal/engine/testdata/methods/widget.go` together with one method; its other two methods live
  in `internal/engine/testdata/methods/aardvark.go`, which sorts first. A test asserting members come
  back in file-then-line order therefore fails if the implementation returns the declaring file's
  member first.

  Write exactly this content, with a package-level doc comment on neither file's package clause —
  `dirPackage` votes on the clause and needs no doc, and a package doc here would change nothing this
  plan asserts.

`internal/engine/testdata/methods/widget.go`:

```go
// widget.go declares the type whose members the expand verb's cross-file member test collects. It
// sorts after aardvark.go deliberately: the members declared there must still come back first.

package methods

// Widget is a fixture type with three methods, two of them in a sibling file.
type Widget struct {
	// Name is a fixture field. Struct fields are not part of the identifier contract and
	// contribute no symbol.
	Name string
}

// Zeta is Widget's method declared in the same file as the type itself.
func (w *Widget) Zeta() string {
	return w.Name
}
```

`internal/engine/testdata/methods/aardvark.go`:

```go
// aardvark.go declares two of Widget's methods, in the file that sorts first in this package.

package methods

// Alpha is Widget's value-receiver method.
func (w Widget) Alpha() int {
	return len(w.Name)
}

// Beta is Widget's pointer-receiver method, declared after Alpha in this same file so the
// within-file start-line order is asserted alongside the across-file order.
func (w *Widget) Beta() {
	w.Name = ""
}
```
- **Commit:** `test(engine): add the methods fixture package for cross-file member coverage`

### Card 4: testdata/tags, a build-tag pair declaring the same names twice

- **Context:**
  - `internal/engine/testdata/glyphs/decls.go`
  - `internal/engine/walk.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/testdata/tags/linux.go`
  - `internal/engine/testdata/tags/other.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create the package `tags` in two files carrying mutually exclusive build
  constraints, each declaring the same three names, so one glyph matches two different declarations.
  This is the `ambiguous` case docs/glyph.md §5 calls "Go build-tag duplicates". The engine never
  evaluates a build constraint — tree-sitter does not, and the walk has no `GOOS` to evaluate it
  against — so both declarations are extracted and both are reported, which is the point of the
  fixture. A directory under `internal/engine/testdata/` is invisible to the go tool, so the pair
  never reaches a build and cannot break compilation or `go vet`.

  The three names cover the three shapes the tests need: `Dup` is a function declared twice, giving a
  type-free ambiguous match set; `DupType` is a type declared twice, giving the type-only ambiguous
  match set; `Mixed` is a type in one file and a function in the other, giving the mixed-kind match
  set that must answer `ambiguous` rather than naming a kind.

  Write exactly this content.

`internal/engine/testdata/tags/linux.go`:

```go
//go:build linux

// linux.go is one half of the build-tag duplicate pair. Its sibling other.go declares the same
// three names under the negated constraint, so every name here matches twice. Nothing under
// testdata/ is ever built, so the pair is data, not code.

package tags

// Dup is declared in both files of this package, as a function in each.
func Dup() string {
	return "linux"
}

// DupType is declared in both files of this package, as a type in each.
type DupType struct {
	// Which names the constraint this declaration sits under.
	Which string
}

// Mixed is a type here and a function in other.go, so one glyph names two different kinds.
type Mixed struct{}
```

`internal/engine/testdata/tags/other.go`:

```go
//go:build !linux

// other.go is the negated half of the build-tag duplicate pair described in linux.go.

package tags

// Dup is declared in both files of this package, as a function in each.
func Dup() string {
	return "other"
}

// DupType is declared in both files of this package, as a type in each.
type DupType struct {
	// Which names the constraint this declaration sits under.
	Which string
}

// Mixed is a function here and a type in linux.go, so one glyph names two different kinds.
func Mixed() {}
```
- **Commit:** `test(engine): add the tags fixture package for build-tag ambiguity coverage`

## Batch Tests

`verify:` runs the whole `internal/engine` package test suite — under half a second on this host,
so there is no reason to scope it narrower — and its job here is regression, not new coverage: these
two cards add committed fixture packages, and the two things that could go wrong are perturbing an
assertion that enumerates a directory or a package's symbol list, and perturbing the whole-repository
round trip. Both are argued through in the Batch Scope above: the enumerating assertions cover
`internal/engine/testdata/tree/` and `internal/engine/testdata/glyphs/`, and neither gains a file
here; `TestRoundTrip_QuarryItself` does walk both new packages, and `tupleSetDiff`'s multiset
semantics are what let the build-tag pair's duplicate declarations pass it. Both arguments are
predictions, and `verify:` is what checks them — a round-trip failure on the `tags` fixture would mean
the multiset reading is wrong, and the implementer must report that rather than deleting the fixture.
The fixtures' own content is asserted by batch 3's cards 11 and 13 and batch 4's cards 16 through 18.
