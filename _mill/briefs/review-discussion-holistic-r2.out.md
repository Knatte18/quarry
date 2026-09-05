MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (runtime-reported; not independently verifiable from my own knowledge)
reviewed_file: /home/knatte/Code/quarry/wts/glyph-self-form/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:scope] A third `#`-containment classifier is unenumerated
**Section:** Problem / Scope (`internal/cli`) / D4 / D7
**Issue:** `internal/cli/flags.go:153` holds a third classification — `verb == "expand" && !strings.Contains(req.target, "#")` → usage error, exit 2, "expand takes a glyph (a target containing \"#\")" — so the Problem's "implemented twice" is wrong, D4's "`expand` already routes a bare path through this same reason, so the two verbs agree" is false at the CLI (`expand <path>` never reaches `glyph.Parse`; it exits 2 while `resolve <path>` exits 1), and D7's mandated doc.go sentence "classification happens exactly once and it is `glyph.Parse` doing it" would be false the day it is written.
**Fix:** State `parseArgs`' gate's disposition (keep, or delete so `no_separator` and exit 1 cover both verbs) and correct D4's and D7's premises accordingly.

### [BLOCKING:design] D8's "MCP needs no other edit" rests on a false premise
**Section:** D8 / Technical context (`internal/mcpserver/`)
**Issue:** `internal/mcpserver/toc.go:107-112` maps `RepoRelTarget` errors with one `errors.Is(ErrTargetOutsideRepo)` branch and an `else` of `"internal error: " + err.Error()`, so `ErrTargetHasSeparator` would surface on the MCP tool as an internal error carrying the sentinel's package-namespaced text — a malformed user target reported as a server fault, and the Testing item for `toc_errors_test.go` would pass on that wrong shape.
**Fix:** Decide the MCP wording and sentinel text explicitly and record that `tocResult` gains a branch, rather than stating the surface inherits D8 unedited.

### [BLOCKING:design] The self form is not language-free as stated
**Section:** Scope (Out: Python and C#) / D1 rationale / D15 (§3)
**Issue:** `docs/glyph.md` §2 defines the unit as the Python *dotted module* and the C# *namespace*, so the §3 paragraph D15 mandates — an empty member is the self form "in every alphabet" and "removing the trailing `#` yields the plain repository-relative path" — is true only for Go, and a file self glyph (`…/focus.go#`) has no meaning under either non-Go unit definition; §2 is also absent from D15's five-site edit inventory while it is the section the self form contradicts.
**Fix:** Scope the path-conversion sentence to Go (or state the per-alphabet self form explicitly) and add §2 to the edit list, or record that §2 stays and why.

### [BLOCKING:consistency] `docs/rewrite-plan.md` §4's round-trip sentence is left standing
**Section:** D16
**Issue:** §4's bullet at line 81 — "**Glyphs as keys in every output**, under `id`. What `toc` lists is what `resolve` takes." — becomes false once `resolve` takes glyphs only, because `toc`'s file and directory entries carry no `id`, so a consumer (the Loomyard adoption this task is timed for) must hand-concatenate `dir + "/" + name + "#"`, which is the printing the Constraints' one-implementation rule forbids outside package `glyph`; D16 enumerates the §4 edit and names only the *other* neighbouring sentence as untouched.
**Fix:** Say in D16 whether that bullet is rewritten, and state how a consumer obtains a listed file's self glyph without re-implementing `String()`.

### [NIT:consistency] Module map misplaces two sentinels
**Section:** D8 / Technical context (`internal/engine/errors.go`)
**Issue:** `ErrTargetNotFound` and `ErrTargetOutsideRepo` are declared in `internal/engine/repo.go:46,50`; `errors.go` holds only `ErrLanguageUnsupported` and its header says "the one error sentinel the engine's subpackages share", so `ErrTargetHasSeparator` cannot be declared "beside" them there without also rewriting that header.
**Fix:** Correct the location in the module map and name the header edit.

### [NIT:consistency] "Delete the `RepoRelPath` tests" deletes tests of retained code
**Section:** Testing (`internal/repopath/target_test.go`)
**Issue:** `TestRepoRelPath_LeadingDotDotNotRejected` and `TestRepoRelPath_AgreesWithRepoRelTarget` call the *unexported* `repoRelPath`, which D6 keeps, so deleting them drops coverage of behaviour this task preserves — and the second test's "agree on every input that does not escape" claim narrows once `repoRelTarget` gains the separator reject.
**Fix:** Delete only tests of the exported wrapper and say how the agreement test is amended.

## Verdict

REQUEST_CHANGES
Four premises contradict the source: a third classifier, MCP error mapping, §2's per-language unit, §4's round-trip sentence.
MILL_REVIEW_END
