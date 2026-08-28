# Batch: docs-and-sweep

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "docs-and-sweep"
number: 8
cards: 8
verify: go test ./internal/quarryengine ./quarry ./internal/cli
depends-on: [7]
```

## Batch Scope

This batch writes the per-language survey document, updates the README for the new verbs and the new
build dependency, and executes the stale-prose sweep — the deliverable that keeps every count,
enumeration and capability claim in the tree from silently describing the quarry that existed before
this task.

It lands last because the sweep is only correct once the final shape exists: the facade's identifier
count, the engine's package count, and the verb list all depend on what batches 1–7 actually built.

The sweep is defined by an invariant, not by a grep. After this batch, no prose anywhere in the tree
may still assert any of:

- a count or enumeration of the engine's packages;
- a count or enumeration of quarry's verbs or capabilities — including prose that lists what quarry
  does without counting it;
- a count of the facade's re-exports;
- that quarry is LSP-only, or that it does not parse source itself;
- that a batch entry is keyed on a symbol;
- that quarry builds without a C toolchain.

Anything matching that description is in scope whether or not the searches in card 54 find it. The
last three clauses are not count-shaped and no grep reliably reaches them.

One site is explicitly **out of scope**: the historical research document at `docs/scout-vs-grep.md`
uses "LSP-backed" to describe a past measurement of a differently-named predecessor tool, not a
current claim about quarry. Historical research documents are not updated when the product changes.

## Cards

### Card 47: the per-language docstring-association survey

- **Context:**
  - `docs/scout-multilang.md`
  - `internal/quarryengine/toc/golang.go`
  - `internal/quarryengine/toc/python.go`
  - `internal/quarryengine/toc/csharp.go`
  - `internal/quarryengine/toc/nodes.go`
  - `internal/quarryengine/toc/classify.go`
- **Edits:** none
- **Creates:**
  - `docs/toc-docstring-association.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write the per-language survey as a standalone reference, in the spirit of the
  existing multilang research document — a reader must be able to add a sixth language from it
  without reading the strategies.
  Cover all five languages, including the two designed but not implemented, with one section each
  stating: the declaration node kinds that produce a symbol; how a docstring is associated with its
  declaration; how the file header is found; how the signature's body-bearing child is resolved; how
  `sigend` is derived; how the package or namespace name is found; and what the `test` and
  `generated` rules are, including which languages have none.
  Record the three implemented shapes as confirmed facts, since they were established by dumping real
  parse trees from the pinned grammars rather than inferred: Go's contiguous `comment` prev-siblings
  with a blank-line stop, plus the `type_declaration` / `type_spec` / `type_alias` split and the
  `field_declaration_list`-or-`{` body rule; Python's in-body first-statement `string` with its
  `string_content` child, plus the `decorated_definition` wrapper; C#'s `///` prev-siblings with XML
  tags, plus the two-level namespace-and-declaration-list descent and the
  `block`-versus-`arrow_expression_clause` body split.
  For the two unimplemented languages, record the shapes the design accounts for: TypeScript's
  `/** ... */` JSDoc block comments as prev-siblings, and Rust's `///` and `//!` line doc comments —
  calling out that `//!` is an *inner* doc comment documenting the enclosing item rather than the
  following one, which is how Rust file headers are written and a genuine trap for the header rule.
  State plainly which languages are implemented and which are designed only, so the document is not
  read as describing shipped behaviour for all five.
- **Commit:** `docs: add the per-language toc docstring-association survey`

### Card 48: README

- **Context:**
  - `internal/cli/toc.go`
  - `internal/cli/cli.go`
  - `docs/toc-docstring-association.md`
  - `go.mod`
- **Edits:**
  - `README.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** update the README for what quarry now is.
  - The opening two sentences call quarry LSP-backed and say it works "rather than reimplementing a
    parser per language". After this task quarry parses source itself for the toc verbs while still
    speaking LSP for the other four. Rewrite both sentences to describe the two backends honestly,
    without turning the summary into a changelog.
  - The verb section states a count and lists four verbs. Add `toc file` and `toc dir` with one line
    each, and rewrite the count so it is correct — or drop the count, which is the more durable fix.
  - The building section must state the new build dependency: `CGO_ENABLED=1` and a C toolchain are
    required to **build**, and nothing extra is required to **run** the built binary. Include both
    supported windows routes: natively with mingw-w64 (MSYS2) or TDM-GCC, and cross-compiled from
    linux or WSL2 with
    `CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags '-extldflags "-static"' -o quarry.exe ./cmd/quarry`,
    noting that this needs `gcc-mingw-w64-x86-64` and that `-static` is deliberate so the produced
    executable does not depend on mingw runtime DLLs sitting beside it.
    **Run that cross-compile command before documenting it.** It has never been executed. If it
    succeeds, document it plainly. If it fails, adjust it until it works and document what actually
    worked. If no mingw-w64 toolchain is installed on the build machine, this does **not** block the
    task: document the recipe marked explicitly as unverified, and say so in the commit message so a
    follow-up can verify it on a machine that has the toolchain. Nothing in this task is
    platform-specific and the windows process implementation is untouched, so an unverified build
    recipe is a documentation gap rather than a regression.
  - Add a section documenting `--doc-sentences`, `$QUARRY_TOC_CONFIG`, and the `.quarry.yaml` target
    directory lookup, with the same four-tier precedence the verb's help text states — including the
    explicit "target directory only, no upward search" limit, which is the part a reader will
    otherwise assume works like every other config file they have met.
  - The testing section states the tiers; confirm it is still accurate now that a C toolchain is
    needed to build the tests at all, and say so if it is not.
- **Commit:** `docs(readme): document the toc verbs, cgo build dependency and toc config`

### Card 49: the engine package-layout documentation

- **Context:**
  - `internal/quarryengine/toc/doc.go`
  - `internal/quarryengine/treesitter/treesitter.go`
  - `internal/quarryengine/errors.go`
- **Edits:**
  - `internal/quarryengine/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** bring the engine's own package documentation in line with the DAG that now
  exists. This is the largest single documentation change in the task and the one most likely to be
  skipped, because none of it fails a build.
  - The opening paragraph describes the engine as answering three "where is this symbol" questions
    over a uniform LSP path. It now also answers "what is in this file" and "what is in this
    directory" over a second, non-LSP backend. Rewrite it to cover both without turning it into a
    history of the change.
  - The engine/CLI split section enumerates the subpackages twice — once in prose and once when
    describing what the seam enforcement test walks. Both enumerations gain `treesitter` and `toc`.
  - The package layout section calls the engine a five-package DAG. It is now seven. Fix the count.
  - That same section is a bulleted list with one bullet per package, each naming the package's files
    and its allowed imports. It needs **two new bullets**, not merely a count edit: one for
    `treesitter` (the parsing backend; imports the root only) and one for `toc` (the orchestration
    layer holding the per-language strategies and the two entry points; imports the root, `registry`,
    and `treesitter`). Place them so the list still reads bottom-up, and describe each package by
    what it is for rather than by which task added it.
  - The `query` bullet says it "imports all four packages above". That count is now wrong; restate it
    in terms of which packages rather than how many, since a count in that position will go stale
    again the next time a package is added.
  - The typed error vocabulary section enumerates every sentinel. Add `ErrLanguageUnsupported`,
    stating what it means and that it is the toc path's only sentinel.
  - The "what this engine deliberately does not do" section is a good place to record two of this
    task's decisions, and both are the kind a future reader would otherwise reopen: exactly one
    parsing backend ships rather than a cgo-and-pure-Go pair selected by build tag, because two
    implementations do not produce identical trees and the same file would then yield different
    answers depending on how the binary was compiled; and toc spawns no daemon and caches nothing,
    because tree-sitter has no project index and no cross-file state for a daemon to keep warm.
- **Commit:** `docs(quarryengine): document the seven-package DAG and the toc sentinel`

### Card 50: the facade's own documentation

- **Context:**
  - `quarry/facade_test.go`
  - `internal/quarryengine/doc.go`
- **Edits:**
  - `quarry/facade.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** rewrite the two stale claims in the facade's package doc comment.
  It calls the engine a five-package DAG and enumerates its members; both become seven, matching the
  wording card 49 settled on so the two documents agree rather than merely both being correct.
  It then claims to re-export "exactly the N identifiers this package exported before the
  engine-repackage move: no more, no less" — a sentence this task's additions falsify outright.
  Recount and rewrite it. Do not leave a stale number, and do not simply delete the number without
  replacing the guarantee it was expressing: the sentence exists to say the facade adds nothing of its
  own, and that claim is still true and still worth stating. Restate it as the property rather than
  as an arithmetic fact, so it stops going stale on every addition.
- **Commit:** `docs(quarry): correct the facade's package and re-export claims`

### Card 51: the CLI package documentation and root command summary

- **Context:**
  - `internal/cli/toc.go`
  - `internal/cli/exec.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** three stale-prose sites in this file, all in or beside the package doc comment.
  - The package doc says the package exposes four verbs and enumerates them. Add the toc group with a
    one-clause description, and correct the count — or restate it without a count.
  - The package doc's batch-mode paragraph says a batch returns "one JSON entry per symbol under a
    top-level results array". toc's batch driver keys its entries on a path instead. Restate the
    contract so it covers both shapes: the shared parts — the results array, the per-entry status
    vocabulary, the worst-status exit code and its ranking — are common, and only the identity key
    differs between the symbol-addressed verbs and the path-addressed ones.
  - The root cobra command's `Short` reads as a capability enumeration and currently leaves
    `quarry --help` describing a tool without toc. Rewrite it to cover what quarry does now. Keep it
    to one line: it is the first thing a user sees.
  Change no code in this card. The `AddCommand` call landed in batch 6; this is prose only.
- **Commit:** `docs(cli): update the package doc, batch contract and root summary`

### Card 52: guard-test prose enumerations

- **Context:**
  - `internal/quarryengine/doc.go`
  - `internal/quarryengine/toc/doc.go`
  - `internal/quarryengine/treesitter/treesitter.go`
- **Edits:**
  - `internal/quarryengine/seam_enforcement_test.go`
  - `internal/quarryengine/layering_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** both guard files open with a header comment enumerating the packages their walk
  covers, and those enumerations are now short by two. Update each to name every package the walk
  actually visits.
  In the layering guard, also check the `layeringTable` doc comment and the import-path constant
  block's comment. Batch 2 card 16 owns both — it is the card that adds the eighth import path — so
  the expected finding here is that they already name eight paths and both new packages. Verify;
  edit only if the expectation does not hold. Do not assume they are stale, and do not restate a
  count this card has no authority over.
  Change no assertion, no constant, and no table row in this card — batches 1 and 2 own those, and
  the numbers they set are correct. This card is prose only, and a diff that touches a `const` or a
  `pathSet` call is a mistake.
- **Commit:** `docs(quarryengine): update the guard tests' package enumerations`

### Card 53: enable the repo-wide done gate

- **Context:**
  - `internal/quarryengine/layering_test.go`
  - `internal/quarryengine/seam_enforcement_test.go`
- **Edits:**
  - `mill-config.yaml`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** set `pipeline.done_gate` to `go test ./...`.
  This is a **bootstrap card**, and the justification for changing the pipeline's own configuration
  mid-task is that `pipeline.done_gate` is read only by mill-go's Handoff step, after every batch has
  already been implemented and verified. It is never read by the planner or by any batch's own verify
  command, so changing it while this task is in flight cannot affect the execution of this task's
  remaining work — only the final gate that runs after it.
  The gate is worth enabling here specifically because this plan's batch verify scopes are
  deliberately narrow: no single batch runs the whole suite, and the two guard tests that this task
  weakens-then-restrengthens live in a package most batches do not verify. A repo-wide run is what
  catches a regression in a package outside every batch's scope.
  `go test ./...` was measured against the current worktree tip before this plan was written: it
  exits 0 in about 3 seconds, so it is cheap enough to run unconditionally.
  Do **not** append a lint command. `golangci-lint` is not installed on this machine, so a
  `done_gate` naming it would fail for a reason unrelated to this task's changes. Record that finding
  in the commit message rather than silently omitting it.
  Expect the first post-change run to be slower than 3 seconds, because it compiles the tree-sitter C
  sources; subsequent runs hit the build cache.
- **Commit:** `chore(mill): enable the repo-wide done gate`

### Card 54: stale-prose sweep verification

- **Context:**
  - `README.md`
  - `quarry/facade.go`
  - `quarry/facade_test.go`
  - `internal/cli/cli.go`
  - `internal/quarryengine/doc.go`
  - `internal/quarryengine/layering_test.go`
  - `internal/quarryengine/seam_enforcement_test.go`
  - `docs/scout-vs-grep.md`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** verify the sweep invariant stated in this batch's scope actually holds, and report
  the result. This card writes nothing; if it finds a stale site, the fix belongs in whichever earlier
  card owns that file, and that card is amended rather than a new commit being made here.
  Run both searches below and **read their output**, treating them as search aids rather than as the
  specification:

  ```
  grep -rnE 'five-package|four verbs|29 identifiers|eight blank-identifier|LSP-backed|minPackageDirs|The six internal|imports? all four|seven re-exported' \
    --include='*.go' --include='*.md' . | grep -v '^\./_mill/' | grep -v '^\./\.scratch/'

  grep -rnE 'lsp, registry|registry, daemon|root leaf' \
    --include='*.go' --include='*.md' . | grep -v '^\./_mill/' | grep -v '^\./\.scratch/'
  ```

  The second search exists because package-set enumerations use no count word and the first search
  cannot reach them.
  Note that the first alternation deliberately reads `imports? all four`, not `import all four`: the
  two sites it must reach are `internal/quarryengine/doc.go`'s `query` bullet ("It **imports** all
  four packages above", which card 49 rewrites) and `layering_test.go`'s `layeringTable` doc comment
  ("query's production files **import** all four", which batch 2 card 16 rewrites). A bare `import all four` matches only
  the second and hides the one this batch is chartered to fix — verified by running both forms
  against the tree before this plan was approved.
  Then check the three clauses no grep reaches, by reading rather than searching: that no prose still
  claims quarry is LSP-only or does not parse source itself; that no prose still claims a batch entry
  is keyed on a symbol; and that no prose still implies quarry builds without a C toolchain.
  Two hits are expected and must be left alone: the historical research document's two uses of
  "LSP-backed", which describe a past measurement of a predecessor tool rather than a current claim
  about quarry. Confirm both hits are that document and no other.
  Report the outcome in the batch's summary: every search's hit count, which hits were confirmed
  out-of-scope, and an explicit statement that the three non-grep-reachable clauses were checked by
  reading.
- **Commit:** none

## Batch Tests

`verify: go test ./internal/quarryengine ./quarry ./internal/cli` covers the three packages whose
files this batch edits. Nearly every change here is a comment, which no test asserts — the run is
there to prove the comment edits did not disturb the code around them, and to re-run the two guard
tests whose header comments card 52 rewrites.

No new test file. Modified: `internal/quarryengine/layering_test.go`,
`internal/quarryengine/seam_enforcement_test.go` — both prose-only in this batch.

The facade's stale identifier count is the one thing here a compiler cannot catch: batch 6 already
made the re-exports self-checking through the blank-identifier block, but a wrong *number* in the
sentence above it stays green forever. That is why card 50 exists as its own card and why card 54
re-reads it rather than trusting a passing build.
