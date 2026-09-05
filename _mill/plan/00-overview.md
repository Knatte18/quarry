# Plan: Glyph self-form and the resolve contract (C1)

```yaml
task: "Glyph self-form and the resolve contract (C1)"
slug: "glyph-self-form"
approved: false
discussion_sha: "70a9ac5cd076e5bd8ac27a426a43b2283fe73717"
started: "20260905-113711"
parent: "main"
root: ""
verify: go vet ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: glyph-grammar
    file: 01-glyph-grammar.md
    depends-on: []
    verify: go test ./glyph/...
  - number: 2
    name: engine-resolve-contract
    file: 02-engine-resolve-contract.md
    depends-on: [1]
    verify: go test ./internal/engine/... ./quarry/...
  - number: 3
    name: expand-gate-and-sentinel
    file: 03-expand-gate-and-sentinel.md
    depends-on: [2]
    verify: go test ./internal/engine/... ./quarry/...
  - number: 4
    name: cli-repopath-mcp
    file: 04-cli-repopath-mcp.md
    depends-on: [3]
    verify: go test ./internal/cli/... ./internal/repopath/... ./internal/mcpserver/...
  - number: 5
    name: docs-and-goldens
    file: 05-docs-and-goldens.md
    depends-on: [4]
    verify: LADDER_LOOMYARD_REPO=/home/knatte/Code/loomyard/wts/loomyard go test ./glyph/... ./internal/cli/...
```

## Shared Decisions

### Decision: the grammar is the only classifier

- **Decision:** after this task no surface tests a target for `#` containment. `glyph.Parse` is the
  single classifier. The three existing tests — `internal/engine`'s `isGlyphTarget`,
  `internal/cli`'s `runResolve` branch, and `parseArgs`' `expand` usage gate — are all deleted, in
  batches 2 and 4 respectively. No replacement check is written anywhere.
- **Rationale:** discussion D4, D6, D7, D19. D7's mandated `internal/cli/doc.go` sentence
  ("classification happens exactly once and it is `glyph.Parse` doing it") is false the day it is
  written unless all three go, so their deletion is a single acceptance condition rather than three
  independent cleanups.
- **Applies to:** all batches

### Decision: doc comments are part of every edit

- **Decision:** every card that changes behaviour also rewrites the doc comments that describe that
  behaviour, in the same commit. The discussion enumerates the falsified comments with line ranges —
  five in `glyph/` (D15/Scope-In), seven paragraphs in `internal/cli` (D7) — and each is named on the
  card that owns it. A card that lands code without its comment edit is incomplete, not partially
  done.
- **Rationale:** this repository's comment density is deliberate: doc comments carry the rationale
  and the rejected alternatives, and `docs/rewrite-plan.md` plus the package docs are treated as part
  of the contract surface (discussion, Technical context).
- **Applies to:** all batches

### Decision: the module-wide compile stays green at every batch boundary

- **Decision:** the `ResolveResult.Dir` → `ResolveResult.Listing` rename (D13) breaks every caller of
  that field at once, across three packages. Batch 2 therefore carries the rename *and* every
  compile-level consequence of it — `quarry/text.go`, `quarry/repo_test.go`,
  `internal/cli/cli_test.go` — even though the behavioural rewrite of `internal/cli`'s own tests
  belongs to batch 4. The module compiles, test files included, after every batch in this plan, and
  the overview's module-wide `verify: go vet ./...` is what gates that at each batch boundary rather
  than leaving the claim unchecked. `go vet` is the right gate and `go build ./...` is not: `go
  build` skips `_test.go` files entirely, and every compile break this plan can cause — card 10's
  rename reaching `quarry/render_test.go`, `quarry/repo_test.go` and `internal/cli/cli_test.go` —
  is in a test file. `go vet ./...` type-checks them, is seconds rather than minutes, and exits 0 on
  the tree as it stands, so it starts clean and any failure it reports belongs to this plan.
- **Rationale:** a broken module-wide build between batches would make the `pipeline.done_gate`
  (`go test ./... && golangci-lint run`) meaningless as an intermediate signal, and would leave a
  bisect landing on a batch that cannot compile. Test *failures* between batches are accepted and
  bounded (see the next decision); compile failures are not. A decision with no gate is a wish, which
  is why the module-wide `verify:` is set rather than left null.
- **Applies to:** all batches

### Decision: batch 2 and batch 3 open a bounded red window in `internal/cli`

- **Decision:** after batch 2, `internal/cli`'s behavioural resolve tests fail: a bare path given to
  `resolve` is now rejected by the grammar, and `internal/cli/cli_test.go` still asserts the old
  path-target answers. Batch 4 retargets them. The window is one batch wide, is not papered over with
  a skip, and each batch's own `verify:` is scoped to the packages that batch owns, so the window
  never blocks the pipeline.
- **Rationale:** the alternative is one batch spanning engine, facade and CLI, which exceeds
  `pipeline.max_cards_per_batch` and would exceed the context budget a Sonnet implementer can hold.
  This repository already carries the same pattern deliberately —
  `internal/cli/after_test.go`'s own header documents an identical bounded red window.
- **Applies to:** engine-resolve-contract, cli-repopath-mcp

### Decision: the `after/` goldens are regenerated, never hand-written

- **Decision:** batch 5's `docs/research/output-formats/after/` files are produced by running
  `LADDER_LOOMYARD_REPO=/home/knatte/Code/loomyard/wts/loomyard go test ./internal/cli/ -run TestAfter -update`
  against the pinned checkout, never by editing golden bytes by hand. That checkout exists on this
  machine at HEAD `72c23d9eecc1fa55add567622093a8bbbfba8c1d`, which is `loomyardPin`, so batch 5's
  `verify:` sets the variable and the golden comparison genuinely runs rather than skipping.
- **Rationale:** `after/INDEX.md` and `internal/cli/after_test.go` both state that these twelve (now
  fifteen) files are one real invocation each; a hand-derived byte is evidence of nothing. Every
  other batch leaves `LADDER_LOOMYARD_REPO` unset, so the Loomyard-gated tests skip there and no
  batch depends on the checkout except the one that regenerates from it.
- **Applies to:** docs-and-goldens

### Decision: the squash message records the free-breakage window

- **Decision:** the squash-merge message for this task states that the resolve envelope had no
  external consumers at the time of the change, and names the five breaking pieces: `resolve` no
  longer accepts a bare path; the `dir` key is now `listing`; `expand <bare-path>` moved from exit 2
  to exit 1; `member_empty` left the `Reason` vocabulary; `repopath.RepoRelPath` is gone. This is a
  merge-time obligation (discussion D18), not a card — no plan file carries it, and `mill-merge` is
  where it lands.
- **Rationale:** D18. It is the record that makes the breakage defensible when Loomyard is wired up.
  Recorded here so it is not lost between the plan and the merge.
- **Applies to:** all batches

### Decision: `verify:` scope is per-package, and the repo-wide gate is the config's `done_gate`

- **Decision:** each batch's `verify:` names only the packages that batch touches. Two wider gates
  sit above them: the overview's module-wide `verify: go vet ./...`, which mill-go runs at every
  batch boundary after the batch's own verify passes, and `mill-config.yaml`'s `pipeline.done_gate`,
  already set to `go test ./... && golangci-lint run`, which mill-go runs from `git_root` before
  marking the task done. No batch runs the unbounded suite itself.
- **Rationale:** `verify:` runs after every implementer and fixer round; the full suite plus
  tree-sitter parsing is far more than the batch needs. The cross-package regressions this plan can
  produce — chiefly the `Dir` → `Listing` rename — are caught at the introducing batch by the
  previous decision's compile rule and at the end by `done_gate`.
- **Applies to:** all batches

## All Files Touched

- `docs/glyph.md`
- `docs/research/output-formats/after/INDEX.md`
- `docs/research/output-formats/after/resolve-self-dir-text.txt`
- `docs/research/output-formats/after/resolve-self-dir.txt`
- `docs/research/output-formats/after/resolve-self-file-text.txt`
- `docs/research/output-formats/after/resolve-self-file.txt`
- `docs/rewrite-plan.md`
- `glyph/doc.go`
- `glyph/docs_test.go`
- `glyph/errors.go`
- `glyph/glyph.go`
- `glyph/golang.go`
- `glyph/golang_test.go`
- `glyph/parse.go`
- `glyph/parse_test.go`
- `glyph/self.go`
- `glyph/self_test.go`
- `glyph/string_test.go`
- `internal/cli/after_test.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/doc.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/usage.go`
- `internal/engine/answer.go`
- `internal/engine/expand.go`
- `internal/engine/expand_test.go`
- `internal/engine/repo.go`
- `internal/engine/resolve.go`
- `internal/engine/resolve_test.go`
- `internal/engine/walk.go`
- `internal/mcpserver/toc.go`
- `internal/mcpserver/toc_errors_test.go`
- `internal/repopath/target.go`
- `internal/repopath/target_test.go`
- `quarry/quarry.go`
- `quarry/render_test.go`
- `quarry/repo_test.go`
- `quarry/text.go`
- `quarry/text_test.go`
