# Batch: docs-and-live-tier

```yaml
task: "Add `impact` verb for caller-context lookup"
batch: "docs-and-live-tier"
number: 4
cards: 4
verify: go build ./... && go vet -tags lsp ./internal/cli/ && go test ./internal/quarryengine/ ./internal/cli/...
depends-on: [3]
```

## Batch Scope

This batch closes the two things that can only be done once the verb exists end to end: the
remaining documentation sites that enumerate the engine package set or the verb set, and the
live-tier test that proves the feature's central requirement against a real gopls.
It is one batch because the doc-audit sweep is only meaningful after every other file has settled,
and because the live-tier test is the single place the docstring-inclusive range claim is actually
proved.

Batch-local decision beyond `## Shared Decisions`: the live-tier test is guarded by a `lsp` build tag
and skipped when gopls is absent, so it is not part of the default verify.
This batch's `verify:` therefore adds `go vet -tags lsp ./internal/cli/` — a type-check of the tagged
tier that costs nothing and prevents a tagged file that does not even compile from being committed,
which is the realistic failure mode for a test the default tier never builds.

## Cards

### Card 14: Engine package doc updates

- **Context:**
  - `internal/quarryengine/impact/impact.go`
  - `internal/quarryengine/impact/types.go`
  - `internal/quarryengine/layering_test.go`
- **Edits:**
  - `internal/quarryengine/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Four distinct edits, not two. Each is a separate enumeration of the package set and all four go
  stale without this card.

  First, the "# The package layout" section's opening line calls the engine "a seven-package DAG";
  correct it to eight.

  Second, the engine/CLI-split paragraph enumerates the packages the seam-enforcement walk covers —
  "this package, lsp, registry, daemon, daemon/daemontest, query, treesitter, and toc"; add the new
  package to that enumeration.

  Third, the same section carries an *earlier*, separate enumeration in its opening sentence —
  "this package and its lsp, registry, daemon, query, treesitter, and toc subpackages" — which
  carries no "seven-package" phrase and is the site the original phrase-grep missed.
  Add the new package there too.

  Fourth, add a new bullet to the package-layout bullet list, in the same shape as the existing
  ones: name the package and its files, state that it composes the verified caller set with the
  declaration ranges for the `impact` verb, and state its allowed imports — the root, `query`, and
  `toc`. Place it after the `query` bullet, since it sits above `query` in the DAG.

  Also add `impact` to the opening paragraph's list of questions the engine answers, alongside the
  existing refs/definition/symbol/toc phrasing.
  Do not add a "what this engine deliberately does not do" entry for transitive impact: that is a
  CLI-facing scope decision recorded in the verb's own help, not an engine capability the engine
  declines to have.
- **Commit:** `docs(engine): document the impact package in the engine package doc`

### Card 15: Seam-guard comment enumerations

- **Context:**
  - `internal/quarryengine/impact/impact.go`
- **Edits:**
  - `internal/quarryengine/seam_enforcement_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Comment-only edits: this file's `minPackageDirs` constant needs **no** bump.
  Its own comment records that the floor is deliberately kept one below the real count — 8, while
  the two-tree walk actually visits 9 — so adding a package keeps the assertion satisfied.
  Changing the constant would silently convert a deliberate slack into a tight coupling.

  Update the header comment's package enumeration, which lists "the root leaf, lsp, registry,
  daemon, daemon/daemontest, query, treesitter, and toc", to include the new package.

  Correct the header comment's "seven-package DAG" phrase to "eight-package DAG". This is the third
  and last occurrence of that phrase in the repo; the other two were corrected in batches 2 and 4's
  card 14.

  Update the `minPackageDirs` comment's own directory enumeration to name the new package, while
  keeping its "deliberately kept one below the real count" reasoning intact and correcting the two
  counts it cites so they stay consistent with the new directory total.

  `TestEngineSeamInvariant_BannedImports`' own doc comment carries no enumeration and needs no edit.
  Change no logic in this file.
- **Commit:** `docs(engine): update the seam guard's package enumerations for impact`

### Card 16: README verb list

- **Context:**
  - `internal/cli/impact.go`
  - `internal/cli/cli.go`
- **Edits:**
  - `README.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add an `impact <symbol|file:line:col>` bullet to the Verbs list, placed after the
  `assert-no-callers` bullet and before the two `toc` bullets, matching the command registration
  order. Describe it in one line as every caller of the symbol, each with its enclosing
  declaration's full line range including the preceding docstring — the answer to "what do I have to
  rewrite" rather than merely "where is it mentioned".

  Correct the sentence immediately below the list that begins "All four verbs accept
  `--build-tags`". It is already wrong today — the list above it holds six verbs — and becomes
  wronger with a seventh. Restate it as a rule rather than a count: `--build-tags` is accepted by
  every LSP-backed verb, naming them, while the tree-sitter-backed `toc` verbs do not take it;
  keep the existing clause stating `--no-verify` is `assert-no-callers`-only.

  Do not restate the JSON shape here — the verb's own `--help` and the plan's Shared Decisions own
  it, and a second copy in the README would go stale independently.
- **Commit:** `docs(readme): add the impact verb and fix the stale verb-count sentence`

### Card 17: Live-tier end-to-end test

- **Context:**
  - `internal/cli/assertnocallers_lsp_test.go`
  - `internal/cli/impact.go`
  - `internal/cli/exec.go`
  - `testdata/impactfixture/billing/invoice.go`
  - `testdata/impactfixture/refund/refund.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/impact_lsp_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Follow `internal/cli/assertnocallers_lsp_test.go` exactly: a `//go:build lsp` constraint on the
  first line, a file header comment stating why the claim is not reproducible against a fake server,
  and a `t.Skip` guarded on `exec.LookPath("gopls")` with the same install hint the existing test
  prints.
  Resolve the repo root from the test file's own location and reuse the fixture tree batch 1 created,
  rooting the query at it.
  Go's registry entry has a native daemon, so use a `t.TempDir()` state directory and reap the
  recorded daemon in a `t.Cleanup`, exactly as the existing live-tier test does through the sanctioned
  facade rather than through any engine-internal package.

  Do not add a second `repoRoot` or daemon-reaping helper: both already exist in this package's
  `lsp`-tagged tier and are reachable from a new file in the same package and under the same build
  tag.

  Drive `impact` against the fixture's `ApplyDiscount` method through the CLI seam and decode the
  JSON envelope. Assert: the envelope carries `ok:true` and `"resolution":"complete"`; `target`
  names the method by its bare name with its owner and package; `definition` carries a `start_line`
  strictly less than its `line`, proving the definition range reaches back over the docstring;
  the `callers` array contains one entry per call site in the calling package, including **two**
  entries for the enclosing function that calls it twice, and those two entries share an identical
  `enclosing_range` while carrying distinct `call_site_line` values; and — the brief's central
  requirement — every caller entry's `enclosing_range.start_line` lands on that caller's docstring
  line rather than its `func` line, located by scanning the fixture source rather than by a
  hard-coded line number.
  Assert also that no entry's file is the declaration site itself, proving the declaration exclusion
  survives the real round trip.
- **Commit:** `test(cli): add the lsp-tier end-to-end test for the impact verb`

## Batch Tests

`verify: go build ./... && go vet -tags lsp ./internal/cli/ && go test ./internal/quarryengine/ ./internal/cli/...`
has three parts, each earning its place.
`go build ./...` is the compile gate.
`go vet -tags lsp ./internal/cli/` type-checks the `lsp`-tagged tier this batch creates — that tier is
never built by the default `go test` run, so without this the new tagged file could be committed in a
state that does not compile, which is exactly the failure mode a tag-gated test invites.
`go test ./internal/quarryengine/ ./internal/cli/...` runs the two packages this batch edits test
files in: the engine root package, whose `seam_enforcement_test.go` card 15 edits, and the CLI
package, whose behaviour cards 16 and 17 describe.
The engine-root pattern is deliberately not `./internal/quarryengine/...` here: this batch touches no
subpackage, and the root package is where both guards live.

New test file: `internal/cli/impact_lsp_test.go` (card 17), which runs only under the `lsp` tag on a
machine with gopls and is not part of the default verify.
Edited test file: `internal/quarryengine/seam_enforcement_test.go` (card 15), comments only.

Doc-audit sweep, performed by the implementer as part of card 16 and reported in its commit body:
after the three doc cards land, re-read every site the `doc-site-ownership-by-touching-batch` Shared
Decision names and confirm each now includes the new package or the new verb.
The sweep is over that rule, not over the phrase "seven-package DAG" — grepping only for the phrase
is what originally missed two of the sites.
