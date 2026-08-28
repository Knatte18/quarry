MILL_REVIEW_BEGIN
# Review: Add `impact` verb for caller-context lookup

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (self-assessed; Anthropic Claude, Opus tier)
reviewed_file: /home/knatte/Code/quarry/wts/impact-verb/_mill/discussion.md
date: 2026-08-28
```

## Findings

### [BLOCKING:consistency] omitempty list contradicts degradation shapes
**Section:** `### json-key-set` vs `### no-enclosing-declaration-is-not-an-error` and `### resolved-symbol-definition-range`
**Issue:** The key set is declared "fixed by json tags" and the omitempty enumeration is `target, definition, owner, package, signature, enclosing_range, sigend_line, per-caller error` — but `no-enclosing-declaration-is-not-an-error` requires `kind` and `name` to be *omitted* on a file-scope entry, and `resolved-symbol-definition-range` requires `definition.start_line`/`end_line` to be omitempty and `definition.error` to exist; none of those four keys is on the omitempty list, so the two required degraded shapes emit `"kind":"","name":""` and cannot emit `definition.error` at all.
**Fix:** Extend the json-key-set enumeration to state the disposition of `kind`, `name`, `definition.start_line`, `definition.end_line`, `definition.error`, and whether a file-scope caller entry keeps `package` (which is known, since the file parsed).

### [BLOCKING:consistency] Reference-typed CLI helpers cannot be reused
**Section:** `## Technical context` ("Reuse all of these; add nothing parallel to them") vs `### callers-come-from-quarry-callers-verified` and `### cli-shape-mirrors-refs`
**Issue:** `filterUnexpectedCallers` and `filterWithin` (`internal/cli/cli.go:696`, `:716`) take and return `[]quarry.Reference`, while `impact`'s callers are its own struct; worse, declaration-set exclusion must run inside `internal/quarryengine/impact` (that package cannot import `internal/cli` — `seam_enforcement_test.go` bans it), so the instruction to reuse them without adding anything parallel is unimplementable, and whether the SDK entry point `quarry.Impact` returns callers already excluding declaration sites and already `--within`-filtered is left undecided.
**Fix:** State explicitly that declaration exclusion happens engine-side inside `impact` (duplicating the six-line set-membership rule) and that `--within` is applied CLI-side over `impact`'s own entry type via `isWithinDir`, and drop those two names from the "reuse, do not reimplement" list.

### [NIT:consistency] "every one of the silent-skip cases" overstates
**Section:** `### resolved-symbol-definition-range`, third bullet
**Issue:** `callersFromClient` (`internal/quarryengine/query/callers.go:63-105`) returns `declaration = toSortedReferences(defLocs)`; it is empty only in the `defErr != nil` and `len(defLocs) == 0` cases — the `!SupportsImplementation()` and `implErr != nil` cases reach the skip only when `len(defLocs) > 0`, so declaration is non-empty there.
**Fix:** Narrow the claim to the two definition-side cases; the decision itself (both keys omitted on an empty set) is unaffected.

### [NIT:decision] `Impact`'s options type has no stated disposition
**Section:** `### impact-lives-in-its-own-engine-package`, `## Scope` ("In")
**Issue:** The signature is given as `Impact(ctx, opts) (Result, error)` with `opts` unnamed; the facade re-export list names `Impact` "plus the result types" and no options alias, implying `query.Options` is reused, but that is never said, and `query.Options` carries no `Within` field.
**Fix:** Name the parameter type (`query.Options`, re-exported as the existing `quarry.Options`) and confirm no new options type is introduced.

## Verdict

REQUEST_CHANGES
Two contract-level contradictions: degraded-entry key set, and unusable cross-seam helper reuse.
MILL_REVIEW_END
