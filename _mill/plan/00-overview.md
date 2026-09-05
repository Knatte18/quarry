# Plan: The glyph-maker: declaration to glyph (P1, roadmap 2b)

```yaml
task: 'The glyph-maker: declaration to glyph (P1, roadmap 2b)'
slug: 'glyph-maker'
approved: true
started: '20260905-142520'
parent: 'main'
root: ""
verify: null
discussion_sha: db3e6ccd476743985de56d1b19b77c6e395a656c
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: engine-maker
    file: 01-engine-maker.md
    depends-on: []
    verify: go test ./internal/engine/ -run 'TestName'
  - number: 2
    name: facade-and-renderers
    file: 02-facade-and-renderers.md
    depends-on: [1]
    verify: go test ./quarry/
  - number: 3
    name: cli-verb
    file: 03-cli-verb.md
    depends-on: [2]
    verify: go test ./internal/cli/
  - number: 4
    name: naming-round-trip
    file: 04-naming-round-trip.md
    depends-on: [1]
    verify: go test ./internal/engine/ -run 'TestRoundTrip'
  - number: 5
    name: docs-inventory
    file: 05-docs-inventory.md
    depends-on: [3]
    verify: go build ./... && go vet ./...
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: the maker owns no naming logic

- **Decision:** the maker wraps the supplied declaration head in a synthetic in-memory Go file, parses it through `treesitter.WithTree`, and calls the registered Go `Strategy`'s `Symbols` method. It contains no receiver-type derivation, no owner-chain construction, no blank-name rule, and no id formatting: the only value it reads off the extracted `Symbol` is the `ID` and `Kind` the extractor already computed.
- **Rationale:** prediction and eventual extraction must be the same function by construction. Any naming decision reimplemented in the maker is a second rulebook that can drift from the Go strategy, and the whole value of the query is that it cannot.
- **Applies to:** all batches. A reviewer reading the maker should see only the wrap, the parse, the count, and the round trip between input and output.

### Decision: nothing is written to disk

- **Decision:** the synthetic file exists only as a `[]byte` handed to `treesitter.WithTree`. No temp file, no scratch directory, on any code path — including the maker's own tests.
- **Rationale:** the query performs no I/O at all; that is the property the package-level facade shape rests on.
- **Applies to:** all batches.

### Decision: additive only

- **Decision:** no existing envelope, verb, exit code, status value, or renderer behaviour changes. The one behaviour edit to existing code is `Run`'s dispatch order, which changes nothing for the three existing verbs.
- **Rationale:** stated constraint of the task.
- **Applies to:** all batches. The one carve-out is test-helper signatures: batch 4 widens `compareGolden`'s third parameter and extends `roundTripSymbol`. Those govern no product envelope, verb, or exit code, so the additive constraint does not reach them.

### Decision: the reason vocabulary and its error sentences

- **Decision:** the maker owns four reason words — `parse`, `no_declaration`, `several_declarations`, `internal` — and propagates, verbatim, any `glyph.Reason` word the validation step produces. Each maker-owned reason has one fixed single-line `Error` sentence that does not repeat the target:

  | reason | `Error` sentence |
  | --- | --- |
  | `parse` | `declaration does not parse` |
  | `no_declaration` | `declaration declares no symbol` |
  | `several_declarations` | `declaration declares N symbols; exactly one is required` |
  | `internal` | `internal error: ` followed by the underlying error's own text |
  | *(propagated)* | the `*glyph.ParseError`'s own `Error()` text, whole |

- **Rationale:** the caller must be able to tell "your fragment is malformed" from "your fragment declared two things" — different fixes. The closed-set-plus-enumerating-slice shape follows the sixteen-reason vocabulary the glyph package already uses.
- **Applies to:** all batches. The text renderer prints the sentence after the target, which is why none of them repeats it; the propagated row is the one exception, because the glyph package composes its own message that way.

### Decision: the `internal` reason diverges between the facade and the CLI

- **Decision:** the facade carries `internal` as an ordinary per-entry reason, so one entry's internal failure never costs a batch its other good answers. The CLI, which renders exactly one entry, checks for it first and takes the compact-error-envelope path — exit 3, no payload — instead of rendering a negative-answer payload.
- **Rationale:** an unwired grammar or a nil tree says nothing about the caller's declaration. The help text already promises exit 3 for internal errors, and the error envelope is reserved for usage and internal errors with no payload.
- **Applies to:** facade-and-renderers, cli-verb.

### Decision: a failed id round trip that is not a parse rejection is `internal`

- **Decision:** validation parses the extracted `Symbol.ID` back through `glyph.Parse` and asserts `String()` returns the same bytes. A `*glyph.ParseError` yields the propagated reason word. A successful parse whose `String()` differs from the input is an extractor invariant violation and takes the `internal` reason, with the `internal` sentence carrying a constructed error naming both spellings.
- **Rationale:** the discussion fixes the propagated-reason disposition for a parse rejection but leaves the byte-inequality branch unstated. It cannot be a caller-facing rejection — the caller supplied a well-formed unit and the grammar accepted the id — so it belongs with the other unreachable-but-spelled conditions, exactly as this repository spells its unreachable branches rather than letting one fall through to a success shape.
- **Applies to:** engine-maker, naming-round-trip.

### Decision: the counts golden ships absent, and the first Loomyard-equipped run produces it

- **Decision:** `internal/engine/testdata/loomyard/` gains no committed counts file in this plan. The naming round trip reads it through `compareGolden`, which fails with the golden's path when it is missing; the batch adds a failure message naming the exact regeneration command alongside it.
- **Rationale:** the machine this plan is being implemented on has no `LADDER_LOOMYARD_REPO` checkout and no `.scratch/ladder.env`, so every Loomyard-gated test skips here and the file cannot be produced. Committing invented numbers would pin the wrong bytes and fail loudly on the first real machine anyway — with a worse message. An explicit missing-golden failure naming the `-update` command is the honest state, and it matches the deliberate red window `after_test.go` already documents for its own goldens.
- **Applies to:** naming-round-trip. Every other test in this plan is machine-independent and runs everywhere.

### Decision: verify scope is one package per batch

- **Decision:** each code batch's `verify:` runs `go test` over the single package it touches, narrowed with `-run` where the package holds a large unrelated suite. The docs batch is the one carve-out: it changes prose only, spans four packages and two Markdown files, and asserts nothing a package test could run, so it is gated by `go build ./... && go vet ./...` instead.
- **Rationale:** `verify:` runs after every implementer and fixer round. The repo-wide gate already exists as `pipeline.done_gate` (`go test ./... && golangci-lint run`), which mill-go runs once before marking the task done, so a repo-wide per-batch command would buy nothing and cost minutes per round. Build and vet are the whole of what is mechanically checkable for a comment-only change, and they are cheap enough to run repo-wide.
- **Applies to:** batches 1 through 4 in the one-package form; batch 5 in the build-and-vet form.

### Decision: goldens are payload bytes only

- **Decision:** each file under `internal/cli/testdata/name/` holds exactly the bytes the command wrote to stdout — no `$ quarry ...` invocation header. The expected exit code is pinned in the test table, never inside a golden file.
- **Rationale:** a header would make each `.json` golden invalid JSON, and the engine's own goldens under `internal/engine/testdata/loomyard/` are pure payload. The header on the frozen research goldens exists because those files are read as evidence documents; these are regression fixtures.
- **Applies to:** cli-verb.

## All Files Touched

_Full union of every `Creates:` / `Edits:` / `Moves:` **target** path across every batch, sorted alphabetically (Move **source** paths are excluded — they disappear, like `Deletes:` tokens).
Cards are the source of truth;
this section is the input `_plan_validate.py`'s `all-files-touched-mismatch` check cross-references against the derived union of every card's `Edits:`/`Creates:`/Move-target paths, to catch drift between the hand/agent-maintained list here and that derived union._

- `README.md`
- `docs/rewrite-plan.md`
- `internal/cli/cli.go`
- `internal/cli/doc.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/loomyard_test.go`
- `internal/cli/name_golden_test.go`
- `internal/cli/name_test.go`
- `internal/cli/testdata/name/function.json`
- `internal/cli/testdata/name/function.txt`
- `internal/cli/testdata/name/malformed.json`
- `internal/cli/testdata/name/malformed.txt`
- `internal/cli/testdata/name/method.json`
- `internal/cli/testdata/name/method.txt`
- `internal/cli/testdata/name/multi-symbol.json`
- `internal/cli/testdata/name/multi-symbol.txt`
- `internal/cli/testdata/name/type.json`
- `internal/cli/testdata/name/type.txt`
- `internal/cli/testdata/name/unknown-receiver.json`
- `internal/cli/testdata/name/unknown-receiver.txt`
- `internal/cli/usage.go`
- `internal/engine/answer.go`
- `internal/engine/expand.go`
- `internal/engine/golden_test.go`
- `internal/engine/name.go`
- `internal/engine/name_test.go`
- `internal/engine/naming_roundtrip_test.go`
- `internal/engine/roundtrip_test.go`
- `quarry/doc.go`
- `quarry/name.go`
- `quarry/name_test.go`
- `quarry/quarry.go`
- `quarry/render.go`
- `quarry/text.go`
