# Batch: docs-and-live-tier

```yaml
task: "Add `impact` verb for caller-context lookup"
batch: "docs-and-live-tier"
number: 4
cards: 4
verify: go build ./... && go test ./internal/quarryengine/ && go test -tags lsp ./internal/cli/
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
and by an `exec.LookPath("gopls")` skip, so no gate anywhere else in the pipeline ever builds or runs
it — `pipeline.done_gate` is an untagged `go test ./...`, and every other batch's `verify:` is
untagged too.
This batch's `verify:` therefore **runs** the tagged tier itself, via `go test -tags lsp
./internal/cli/`, rather than merely type-checking it.
A `-tags lsp` run is a strict superset of the untagged one for that package — build tags only add
files — so it replaces the plain `./internal/cli/...` run rather than sitting alongside it.
The `exec.LookPath("gopls")` skip keeps it green on a machine without gopls (which is the case in
this worktree today, so the assertions will skip here and prove themselves on a machine that has
it); what the gate buys unconditionally is that the tagged file must compile, which no other gate in
the pipeline checks at all.

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
  Five distinct edits, not two. Each is a separate enumeration of the package set or of the
  questions the engine answers, and all five go stale without this card.

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

  Fifth, add `impact` to the opening paragraph's list of questions the engine answers, alongside the
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
  Comment-only edits: this file's `minPackageDirs` constant needs **no** bump, and must not be
  changed.
  Its floor of 8 is already below the directory count the two-tree walk visits today (nine, once
  `quarry/` is counted), and batch 1's new package only widens that gap to two, so the assertion
  stays satisfied without any change.
  Leaving the constant alone is also what `discussion.md` decides explicitly: raising it would
  convert a deliberate slack into a tight coupling that every future package addition has to
  service.

  Update the header comment's package enumeration, which lists "the root leaf, lsp, registry,
  daemon, daemon/daemontest, query, treesitter, and toc", to include the new package.

  Correct the header comment's "seven-package DAG" phrase to "eight-package DAG". This is the third
  and last occurrence of that phrase in the repo; batch 2's card 8 corrected the one in the facade
  header, and card 14 above corrected the one in the engine package doc.

  Update the `minPackageDirs` comment's own directory enumeration to name the new package, and
  restate its slack **without a count**: the floor is deliberately kept below the real count, full
  stop.
  Do not preserve the "kept one below" phrasing and do not re-derive a new gap figure.
  The reason is arithmetic: the comment currently reads "…and toc is eight, quarry/ makes nine —
  …its floor is deliberately kept one below the real count (8, not 9)", but after batch 1 the
  two-tree walk visits nine engine directories plus `quarry/`, so the floor of 8 is two below, not
  one.
  Keeping the phrasing and correcting the cited counts are mutually exclusive; naming a specific gap
  at all is what made this comment go stale in the first place, so the fix is to stop naming one.
  Correct the enumeration's own two totals (nine engine directories, ten across both trees) since
  those are statements of fact about the walk, not about the slack.

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

  Finally, perform the doc-audit sweep as the last step of this card, once cards 14 and 15 have
  landed and this card's own edit is made.
  Re-read every site the `doc-site-ownership-by-touching-batch` Shared Decision names — across all
  four batches, not just this one — and confirm each now includes the new package or the new verb.
  The sweep is over that rule, not over the phrase "seven-package DAG": grepping only for the phrase
  is what originally missed two of the sites, so a phrase grep is a cross-check, never the sweep
  itself.
  Record the sweep's outcome in this card's commit body, naming each site checked.
  If the sweep finds a stale site that no card covers, fix it here and say so in the commit body
  rather than leaving it for a later task.
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

`verify: go build ./... && go test ./internal/quarryengine/ && go test -tags lsp ./internal/cli/`
has three parts, each earning its place.

`go build ./...` is the compile gate over the whole module.

`go test ./internal/quarryengine/` runs the engine **root** package, where card 15's edited
`internal/quarryengine/seam_enforcement_test.go` lives alongside the layering guard.
The pattern is deliberately not `./internal/quarryengine/...`: this batch touches no engine
subpackage, and both guards live in the root package.

`go test -tags lsp ./internal/cli/` runs the CLI package **including** the `lsp`-tagged tier card 17
creates. This is the only gate anywhere in the pipeline that builds or runs that tier — every other
batch's `verify:` and `pipeline.done_gate` are untagged — so without it the new tagged file could be
committed in a state that does not even compile, the realistic failure mode a tag-gated test invites.
Because build tags only add files, this run is a strict superset of the untagged
`./internal/cli/...` run and replaces it rather than sitting alongside it.
On a machine without gopls the live assertions skip via `exec.LookPath`, leaving the compile check as
the unconditional benefit; on a machine with gopls the brief's central claim is actually proved.

New test file: `internal/cli/impact_lsp_test.go` (card 17), in the CLI package under the `lsp` tag.
Edited test file: `internal/quarryengine/seam_enforcement_test.go` (card 15), comments only.
Card 16 edits `README.md`, a documentation file with no runnable surface — it is covered by the
doc-audit sweep below rather than by any test.

The doc-audit sweep is card 16's own last requirement, not a free-floating instruction here: see that
card's `Requirements:` for what it covers and where its outcome is recorded.
