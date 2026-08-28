MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: plan/
date: 2026-08-28
```

## Findings

### [BLOCKING:design] `go mod tidy` will delete card 1's new requires
**Location:** batch 1 card 1 **Issue:** No package in the module imports any tree-sitter module until card 3 creates `treesitter.go`, so `go mod tidy` prunes all six requires plus `go-pointer` and leaves `go.sum` unpopulated; card 1's own `go build ./...` check passes trivially and card 3 then fails with "no required module provides package". **Fix:** pin with `go get <module>@<exact version>` (still never `@latest`), or move the dependency add into the card that creates the first importing file and run `tidy` only after it.

### [BLOCKING:decision] `classifyTOCError` has no source for the implemented set
**Location:** batch 6 card 36 (+ card 34) **Issue:** The Shared Decision requires the unsupported-language message to name "the implemented set", but card 36 says to build it from `quarry.TOCLanguages()`, which card 34 defines as `registry.ExtensionLanguages()` — all five *designed* names including `rust` and `typescript`; `toc.Implemented()` is never re-exported and card 36 forbids `internal/cli` importing engine subpackages. **Fix:** decide and state which set the message names, and if it is the implemented set, add a `TOCImplemented()` facade re-export in card 34.

### [BLOCKING:consistency] Card 15's `Register`-panic test poisons card 33's guard
**Location:** batch 2 card 15 vs batch 5 card 33 **Issue:** Asserting `Register` panics on a duplicate requires one successful registration of a fake language into the package-level map, which has no unregister path; `classify_test.go` sorts before `toc_test.go`, so `TestImplemented_MatchesRegisteredStrategies`' "exactly csharp, go, python" assertion sees the fake and fails. **Fix:** give card 15 a snapshot/restore seam (or a test-only reset) in `t.Cleanup`, and say so in card 10's `Register` requirements.

### [BLOCKING:scope] `toc dir` batch mode never composes per-file `path`
**Location:** batch 6 cards 37, 38, 40 **Issue:** Card 37 specifies the `json:"-"` `Name` → `path` zip-by-index only inside `tocDirCommand`'s single-argument `RunE`; card 38 wires `runPathBatch` for both subcommands without restating it, and card 40's batch cases assert only the entry-level `"path"` key, so a multi-argument `toc dir` can silently emit `files[]` with no `path`. **Fix:** name the composition as a shared helper both branches call, and add a batch-mode `files[].path` assertion to card 40.

### [NIT:consistency] `output.Err` alone does not set exit 1
**Location:** batch 6 cards 36, 37; batch 7 card 43 **Issue:** The cards say "emits `output.Err` and returns nil" while requiring exit 1, but every existing verb writes `SetExit(ctx, output.Err(out, ...))` — `output.Err`'s return value is the exit code and is otherwise discarded. **Fix:** state the `SetExit(ctx, output.Err(...))` shape explicitly in the three error-path requirements.

### [NIT:consistency] `SigEnd(decl, ...)` is undefined for a grouped type spec
**Location:** batch 3 card 19 **Issue:** The card says `SigEnd` is `SigEnd(decl, goTypeBody(spec), true)`, but in the grouped branch the symbol unit is the spec while `decl` names the enclosing `type_declaration`; the clamp floor would then be the group's start line rather than the spec's. **Fix:** name the spec node as the first argument in the grouped branch (the `true` flag makes it inert today, but the ambiguity is what a later change breaks).

## Verdict

REQUEST_CHANGES
Four blocking issues: dependency mechanic, sentinel message source, test-registry pollution, and batch-mode path composition.
MILL_REVIEW_END
