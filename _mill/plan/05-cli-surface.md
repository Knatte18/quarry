# Batch: cli-surface

```yaml
task: "Improve gopls query precision (build tags + scoping)"
batch: "cli-surface"
number: 5
cards: 5
verify: go test ./internal/cli/
depends-on: [3, 4]
```

## Batch Scope

This batch delivers everything an operator sees: the `--build-tags` flag on all four verbs with its `$QUARRY_BUILD_TAGS` fallback, the `tags-<hex>` state-directory segment that gives each distinct tag set its own supervised daemon, the rewiring of `assert-no-callers` onto `quarry.Callers` with its `--no-verify` escape hatch, and the help text that is currently wrong once verification is on by default.

It is one batch because every card edits `internal/cli/cli.go` and they share one flag-plumbing shape. Batch-local decision: `resolveBuildTags` lives in `internal/cli/paths.go` beside `resolveConfigPath` and `resolveStateDir` rather than in `internal/cli/cli.go`, because it is the same flag-then-environment-then-default resolution those two already implement and that file already imports `os`.

## Cards

### Card 18: --build-tags on all four verbs

- **Context:**
  - `quarry/facade.go`
  - `internal/cli/exec.go`
- **Edits:**
  - `internal/cli/paths.go`
  - `internal/cli/cli.go`
  - `internal/cli/paths_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add `func resolveBuildTags(flagValue string) []string` to `internal/cli/paths.go`, resolving in the precedence `--build-tags` value, then `$QUARRY_BUILD_TAGS`, then empty, and returning `quarry.NormalizeBuildTags` applied to whichever won. A raw value that normalizes away — `""`, `","`, `" , "` — yields nil, which every downstream consumer treats as "no build tags".
  - Add a `--build-tags` string flag to each of the four subcommands built in `internal/cli/cli.go`: `refsCommand`, `definitionCommand`, `symbolCommand`, and `assertNoCallersCommand`. Declare it per-verb alongside each command's existing `--target-dir` / `--lang` / `--timeout` locals, not as a root persistent flag: build tags are a query-scoping concern, not process infrastructure.
  - Flag help, identical on all four: name the comma-separated shape, name `$QUARRY_BUILD_TAGS` as the fallback, and state that passing tags for a language whose registry entry carries no build-tag template is an error rather than a silent no-op.
  - Add a `buildTags []string` parameter to `buildOptions` in `internal/cli/cli.go` and set `Options.BuildTags` from it. Update every existing `buildOptions` call site to pass the resolved value; there are seven, spread across the single-argument and batch-mode paths of all four verbs.
  - Extend `internal/cli/paths_test.go` with a table over `resolveBuildTags`: the flag value wins over the environment variable; the environment variable is used when the flag is empty; neither set yields nil; and each of `""`, `","` and `" , "` yields nil from either source. Set and unset the environment variable with `t.Setenv` so the tests stay isolated.
- **Commit:** `feat(cli): add --build-tags and $QUARRY_BUILD_TAGS to all four verbs`

### Card 19: fold the tag set into the resolved state directory

- **Context:**
  - `internal/quarryengine/daemon/daemonstate.go`
  - `quarry/facade.go`
- **Edits:**
  - `internal/cli/paths.go`
  - `internal/cli/cli.go`
  - `internal/cli/paths_test.go`
  - `internal/cli/resolve_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add a `buildTags []string` parameter to `resolveStateDir` in `internal/cli/paths.go`. When the slice is empty the function returns exactly what it returns today at all three precedence tiers, byte for byte. When it is non-empty, append one further path segment to the resolved leaf, at every tier including the explicit `--state-dir` and `$QUARRY_STATE_DIR` tiers, which bypass `workspaceKey` entirely today.
  - The segment is the literal `tags-` followed by the first 12 hex characters of the SHA-256 digest of the normalized tags joined with `,` — the same hash and the same 6-byte truncation `workspaceKey` in that file already uses, so the file carries one hashing convention rather than two. Factor it as its own small function beside `workspaceKey` and document the shared convention on it.
  - Explain in the function's doc comment why the segment is applied to the resolved leaf rather than folded into `workspaceKey`: the first two tiers never call `workspaceKey`, so folding it in there would silently collide two tag sets on one socket for an operator who pins `--state-dir`, which is exactly the case this keying exists to prevent.
  - Add a `buildTags []string` parameter to `resolveContext` in `internal/cli/cli.go` and pass it to `resolveStateDir`. Update all four `resolveContext` call sites to resolve the tag set first and pass it.
  - Extend `internal/cli/paths_test.go`: an empty tag set yields paths identical to today at all three tiers — write this back-compat assertion first, comparing against the values the existing table already pins; a non-empty tag set yields a distinct path at all three tiers, explicit `--state-dir` included; `[]string{"a","b"}` and `[]string{"b","a"}` yield the same path once normalized; and the appended segment is the literal `tags-` followed by exactly 12 hexadecimal characters.
  - Update the existing `resolveContext` call sites in `internal/cli/resolve_test.go` for the new parameter, and add one case asserting a non-empty tag set changes the returned state directory while leaving the returned registry and absolute target directory unchanged.
- **Commit:** `feat(cli): key the daemon state directory by the normalized build-tag set`

### Card 20: assert-no-callers runs on the verified one-connection path

- **Context:**
  - `quarry/facade.go`
  - `internal/quarryengine/query/callers.go`
  - `internal/output/output.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Replace the two separate `quarry.Definition` and `quarry.References` calls in `assertNoCallersCommand`'s `RunE` in `internal/cli/cli.go` with a single `refs, declRefs, err := quarry.Callers(ctx, opts)`. Route its error through the existing `emitAmbiguousOrError` helper exactly as the two calls it replaces already do, so the ambiguity-to-exit-2 and everything-else-to-exit-1 mapping is unchanged.
  - Add a `--no-verify` boolean flag, declared only on this command, and set `opts.SkipVerification` from it after `buildOptions` returns. The flag's polarity matches the engine field: absent means verify.
  - Keep the filter composition after the call exactly as it is — `--within` via `filterWithin`, then `filterUnexpectedCallers` applying `--except` and the declaration exclusion in its single existing pass. Verification now happens inside `quarry.Callers`, ahead of all three, so the documented "`--within` before `--except`" relationship is preserved and no restructuring is implied. Do not modify `filterWithin`, `isWithinDir`, or `filterUnexpectedCallers`.
  - Keep the emitted envelope and exit codes byte-identical: exit 0 with an empty `callers` list when no violations remain, and `violation: true` with the `callers` list plus exit 1 when any do.
  - Two further comments in `internal/cli/cli.go` go stale with this card and must be corrected in it, not left for card 21. The comment above the `filterWithin` call inside `assertNoCallersCommand` says `--within` exists because an unscoped interface-method check can otherwise report a false violation — that is the claim card 21 rewrites the help text to retract, so rewrite it here to say `--within` scopes the candidate set before `--except` runs and nothing more. And `emitAmbiguousOrError`'s own doc comment says it maps "References/Definition errors" to the output envelope, which stops being the whole truth once this card routes `quarry.Callers`'s error through it; name `Callers` there too.
  - Delete the now-stale inline comment above the removed `quarry.Definition` call and replace it with one explaining what still holds: the declaration set must come from a real `textDocument/definition` rather than from the caller-supplied position, because `quarryengine.Position.Character` is a 1-based byte column and `quarry.Reference.Character` is a 1-based UTF-16 column, so they coincide only on a pure-ASCII line. `quarry.Callers` returns the definition-only declaration set precisely so this exclusion keeps working.
- **Commit:** `feat(cli): verify assert-no-callers callers by default, with --no-verify to opt out`

### Card 21: rewrite the assert-no-callers help for default-on verification

- **Context:**
  - `internal/quarryengine/query/callers.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Rewrite the `--within` paragraph in `assertNoCallersCommand`'s `Long` help in `internal/cli/cli.go`. It currently tells the caller to use `--within` *because* an unscoped interface-method check reports false violations, which stops being true once verification is on by default. The new paragraph describes `--within` as an ordinary optional scoping filter that restricts the caller search to references whose file lies within a directory, and keeps its existing claim that it applies before `--except` — that ordering claim stays true and is cited elsewhere as authoritative, so it must survive the rewrite verbatim in meaning.
  - Rewrite the interface-method conflation paragraph to describe what happens now: gopls' references for an interface method conservatively include every structurally-compatible interface's matching method across the workspace, and `assert-no-callers` now filters that set by resolving each candidate reference's own definition and keeping only those that resolve back to the queried symbol's declaration — so the default answer is precise without a scoping flag.
  - Add a paragraph documenting `--no-verify` as the escape hatch that reinstates the older, noisier behaviour, and stating that it is `assert-no-callers`-only: `refs`, `definition` and `symbol` are unchanged and gain no verification flag.
  - Change the `--within` flag's own one-line help, which currently calls it "required for a correct check on an interface method". It is no longer required for correctness; describe it as an optional scoping filter.
  - Add the `--no-verify` flag's one-line help, naming the fail-closed posture in a clause: an unverifiable reference is kept as a violation rather than dropped.
  - Leave the `refs` and `definition` `Long` help unchanged. Those verbs' behaviour is untouched by this task, and their own `--within` documentation is still accurate for them.
- **Commit:** `docs(cli): rewrite assert-no-callers help for default-on definition verification`

### Card 22: internal/cli behaviour tests

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/exec.go`
  - `internal/cli/paths.go`
  - `internal/output/output.go`
  - `quarry/facade.go`
- **Edits:**
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Update the existing `buildOptions` test in `internal/cli/cli_test.go` for the new parameter, and add an assertion that the resolved tag set lands on `Options.BuildTags`.
  - Add a table test over `filterUnexpectedCallers` and `filterWithin` composed in the order `assertNoCallersCommand` applies them — `--within`, then `--except`, then the declaration exclusion — over a hand-built reference slice. This is the only part of the ordering `internal/cli` can observe: `quarry.Callers` is a direct package-level call with no injection seam, so no test here can see the verification step. State that limitation in the test's comment and name the engine test that covers the rest.
  - Assert that a reference in the declaration set is excluded, that a reference under an `--except` path is excluded, that a reference outside `--within` is excluded, and that a reference surviving all three is reported as a violation.
  - Add an assertion that the four verbs each accept a `--build-tags` flag and `assert-no-callers` additionally accepts `--no-verify`, by looking the flags up on the built command tree rather than by executing a query. The existing tests in this file already build the command tree this way.
  - Do not add any test here that needs a language server. The live-tier coverage for `assert-no-callers` is batch 6's job.
- **Commit:** `test(cli): cover build-tag plumbing and the assert-no-callers filter ordering`

## Batch Tests

`verify:` runs `internal/cli/`, which is the only package this batch touches. Cards 18 and 19 are covered by the extended tables in `internal/cli/paths_test.go` and `internal/cli/resolve_test.go` — in particular the empty-tag-set back-compat assertions, which pin that today's resolved state directories are unchanged at all three precedence tiers. Cards 20, 21 and 22 are covered by `internal/cli/cli_test.go`: the filter-ordering table, the flag-registration assertions, and the existing envelope tests that must keep passing to prove the exit-code contract did not move. Verification itself is deliberately not asserted here — it happens inside `quarry.Callers`, which this package calls directly with no seam to fake — so the engine tests from batch 4 and the live tests in batch 6 carry that.
