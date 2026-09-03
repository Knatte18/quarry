# Plan: The glyph package (T1)

```yaml
task: "The glyph package (T1)"
slug: "glyph-package"
approved: false
started: "20260903-163743"
parent: "main"
root: ""
verify: go vet ./... && golangci-lint run
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: types-and-printer
    file: 01-types-and-printer.md
    depends-on: []
    verify: go test ./glyph/
  - number: 2
    name: parser-and-go-alphabet
    file: 02-parser-and-go-alphabet.md
    depends-on: [1]
    verify: go test ./glyph/
```

## Shared Decisions

_Cross-cutting decisions every batch inherits._

### Decision: the discussion's Decisions section is the specification, verbatim

- **Decision:** `_mill/discussion.md`'s `## Decisions` section fixes the alphabet, the sixteen
  `Reason` constants, the two precedence orders, the `Params` nil/empty rule and the print form. This
  plan restates each of them at the card that implements it, in implementable terms. Where this plan
  and the discussion disagree in wording, the discussion's Decisions section wins; where this plan
  names a Go identifier the discussion left unnamed, this plan wins.
- **Rationale:** the discussion is the reviewed artefact; the plan is the executable transcription of
  it. Restating rather than cross-referencing keeps each card readable cold, which is what a Sonnet
  implementer needs.
- **Applies to:** all batches

### Decision: standard library only, in package and test files alike

- **Decision:** no file in the new package — implementation or `_test.go` — imports anything outside
  the Go standard library. The whole package needs only `fmt`, `strings`, `unicode` and
  `unicode/utf8`; the tests add only `errors`, `reflect` and `testing`. `go.mod` gains no
  `require` line from this task.
- **Rationale:** `docs/glyph.md` §6 promises any program can import this package without the engine.
  `go list -deps` cannot see test imports, so the test half of the rule is enforced by review of the
  import lines. `github.com/google/go-cmp` is therefore unavailable and `reflect.DeepEqual` is the
  comparison, matching the repository's existing precedent in the toc tests.
- **Applies to:** all batches

### Decision: whole-`Glyph` comparison uses reflect.DeepEqual

- **Decision:** tests compare whole `Glyph` values with `reflect.DeepEqual`, never `==`, including
  when the expected value is the zero `Glyph`.
- **Rationale:** `Glyph` holds two slice fields and is not comparable with `==`.
  `reflect.DeepEqual` distinguishes a nil slice from an empty one, which the `Owner`-is-nil,
  `Params`-is-nil and `Params`-printing assertions all depend on.
- **Applies to:** all batches

### Decision: all three test files are `package glyph`

- **Decision:** `glyph/parse_test.go`, `glyph/golang_test.go` and `glyph/string_test.go` all declare
  `package glyph`, not `package glyph_test`.
- **Rationale:** the split helper under test is unexported, the round-trip test is driven from the
  accept table declared in another file, and the `Reasons` completeness test must see reject rows
  declared in more than one file. All three need one package for any of that to compile.
- **Applies to:** all batches

### Decision: on any error Parse returns the zero Glyph

- **Decision:** every error path in `Parse` returns `Glyph{}` — every field zero — alongside the
  `*ParseError`. No partially populated struct is ever returned, for any of the sixteen reasons.
  Every reject test row asserts this alongside the `Reason`.
- **Rationale:** a caller that logs the error and carries on must not hold a value that looks half
  usable and names nothing.
- **Applies to:** all batches

### Decision: file-level comments and godoc match the toc package

- **Decision:** every new file — implementation and test alike — opens with a file-level comment
  naming what the file holds and why. In every file except `glyph/doc.go` that comment is
  **separated from the `package glyph` clause by a blank line**, as
  `internal/quarryengine/toc/types.go` separates its own from `package toc` and
  `internal/quarryengine/toc/toc_test.go` does for a test file. `glyph/doc.go` is the sole
  exception: its comment abuts the package clause, because there it *is* the package doc comment,
  and `doc.go` is the only file that owns it. Every exported identifier carries a godoc
  comment; the
  closed `Reason` vocabulary is a `string`-based named type with a grouped `const` block and a doc
  comment on each constant.
- **Rationale:** this is the existing convention in `internal/quarryengine/toc` and in
  `internal/quarryengine`, and the `golang-comments` skill's rules.
- **Applies to:** all batches

### Decision: verification is cgo-enabled, and the no-cgo claim is about this package only

- **Decision:** every verify command runs in the ordinary cgo-enabled configuration.
  A `CGO_ENABLED=0` build of the repository is expected to fail by design and is never attempted.
  The pure-Go claim is proved for the new package alone, with `go list -deps ./glyph`, whose pass
  condition is that every package it prints is standard library and no non-stdlib module appears.
- **Rationale:** the engine ships a compile-time guard that fails a `CGO_ENABLED=0` build on purpose,
  so a repository-wide no-cgo build proves nothing about this package.
- **Applies to:** all batches

### Decision: golangci-lint is green at the parent tip and stays in the module-wide verify

- **Decision:** the module-wide `verify:` above is `go vet ./... && golangci-lint run`, run at each
  batch boundary after the batch's own `verify:` passes. `pipeline.done_gate` in the hub config is
  left at its existing `go test ./...` and is not changed by this task.
- **Rationale:** `golangci-lint run` exits 0 at the parent tip, so adding it costs nothing and
  catches lint regressions in the new package at the batch that introduces them. Changing the hub
  config file mid-task would make every future task in this hub depend on an edit this task has no
  reason to make.
- **Applies to:** all batches

### Decision: the spec is not edited, and two questions are left open for the hub

- **Decision:** no card edits `docs/glyph.md`. The two open questions the discussion records — how a
  Go package in the repository root is spelled, and whether whitespace is banned in a unit segment —
  stay open. This plan implements the discussion's answers: the root package has no glyph (`""` is
  `unit_empty`, `.` is `unit_dot_segment`), and whitespace in a unit is `unit_bad_rune`.
- **Rationale:** the task's own constraint — where the spec is unclear, ask; the hub fixes the spec,
  the code does not fix itself around it.
- **Applies to:** all batches

## All Files Touched

- `glyph/doc.go`
- `glyph/errors.go`
- `glyph/glyph.go`
- `glyph/golang.go`
- `glyph/golang_test.go`
- `glyph/parse.go`
- `glyph/parse_test.go`
- `glyph/string_test.go`
