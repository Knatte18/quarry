MILL_REVIEW_BEGIN
# Review: Add `impact` verb for caller-context lookup

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/impact-verb/_mill/discussion.md
date: 2026-08-28
```

## Findings

### [BLOCKING:consistency] `--within` counterpart drops refs' path normalization
**Section:** Technical context → "`--within` is applied CLI-side" (and Decision `cli-shape-mirrors-refs`, which claims "flag parity with `refs`" and "the same `--within` filter")
**Issue:** `filterWithin` (`internal/cli/cli.go:716-740`) does three normalization steps on the flag value before comparing — join onto `baseDir` when relative, `filepath.Abs`, `filepath.Clean` — and its own comment records the failure mode: "`Reference.File` … is always absolute, so `w` must be too, or every comparison below silently fails"; the discussion reduces impact's counterpart to "filtering on each entry's `file` with the existing `isWithinDir` helper", which for a relative `--within internal/foo` makes `filepath.Rel` error and silently filter every caller out.
**Fix:** State that impact's `--within` counterpart reproduces `filterWithin`'s within-value normalization (relative→`baseDir` join, `Abs`, `Clean`) before the per-entry `isWithinDir` call, and add a CLI test asserting a *relative* `--within` value matches, not just an absolute one.

### [BLOCKING:scope] Doc/guard enumeration inventory is not exhaustive as claimed
**Section:** Scope "In" → "Doc updates, enumerated precisely (each verified to exist at the stated site)"; Technical context → "Guards that will fail…"
**Issue:** The inventory was evidently built by grepping the phrase "seven-package DAG" plus nearby lists, and misses enumerations that carry no such phrase: `internal/quarryengine/doc.go:29-30` holds a **third** package enumeration ("this package and its lsp, registry, daemon, query, treesitter, and toc subpackages") while the discussion asserts doc.go has "two separate stale enumerations"; doc.go's own package-layout bullet list (lines 53-92) needs a new `impact` bullet, which is never stated; and `internal/cli/cli_test.go:945-974`'s `TestBuildTagsFlag_RegisteredOnAllFourVerbs` is a hand-enumerated four-verb table that both silently under-covers impact's `--build-tags` and encodes the "`--no-verify` is assert-no-callers-only" invariant the Scope "Out" list asserts for impact — the exact silent-under-coverage category the discussion already flagged for `quarry/facade_test.go`.
**Fix:** Replace the phrase-grep inventory with an enumeration rule stated in the discussion (e.g. "every comment or test table that lists the engine packages or the verb set"), and re-derive the list under it so the plan writer inherits a complete, checkable set rather than three verified sites plus unknown misses.

### [NIT:consistency] `ok`/`resolution` listed as struct-tag-fixed keys
**Section:** Decision `json-key-set` **Issue:** "the emitted key set is fixed by json tags on the `impact` package's result structs" plus an "**Envelope:** `ok`, `resolution`, `callers` always present" bullet reads as putting `resolution` on `impact.Result`, but `ok` comes from `output.Ok` and `refs` adds `resolution` CLI-side (`emitLookupResult`, `internal/cli/cli.go:790`); an engine-side `resolution` field would leak a CLI trust marker into `quarry.Impact`'s SDK contract. **Fix:** Say explicitly that `ok` and `resolution` are envelope keys added CLI-side and that only `target`/`definition`/`callers` are struct-tag-fixed.

### [NIT:design] Greatest-`Start` rationale rests on the wrong overlap example
**Section:** Decision `enclosing-range-reuses-toc-symbols`; Technical context gotcha on grouped types **Issue:** `toc/golang.go:176-205` computes each grouped-spec Symbol's range "from spec itself, not from the enclosing type_declaration", so grouped Go specs do **not** overlap; the guaranteed overlap is Python/C# class↔method nesting (`toc/python.go:70-85` emits a class Symbol spanning its methods), which the discussion never names and whose fixture coverage the file-level test list (Go, package-scope, unreadable, `.ts`) omits. **Fix:** Restate the tie-break's justification on class↔method nesting and add a Python or C# fixture case for it.

### [NIT:design] Parse-cache assertion has no committed observable
**Section:** Testing → "file-level tests against fixtures" **Issue:** "the observable proxy is that the result is identical and the test does not need a parse counter if the design exposes one; otherwise inject a counting seam" — identical results do not prove one parse, so the primary proxy cannot prove the `per-call-parse-cache` decision's claim, leaving two alternatives without a choice. **Fix:** Commit to the counting seam as the assertion mechanism and drop the identical-result proxy.

### [NIT:consistency] `buildQuery` described as an existing shared seam
**Section:** Decision `cli-shape-mirrors-refs` **Issue:** "the same `buildQuery` seam" implies a reusable helper, but `buildQuery` is a per-command local closure duplicated at `internal/cli/cli.go:176` and `:316`; the Technical context reuse list correctly names only `parseQuery`/`inFileQuery`. **Fix:** Say impact re-creates the same closure shape, so it is not mistaken for a helper to call.

## Verdict

REQUEST_CHANGES
Two blockers: `--within` normalization silently dropped, and an incomplete doc/guard inventory.
MILL_REVIEW_END
