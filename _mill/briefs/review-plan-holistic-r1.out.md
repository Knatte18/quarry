MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus-class model (identifies as claude-opus-5 per harness metadata)
reviewed_file: plan/
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Python shebang directive rule can never match
**Location:** batch 2 card 13 + batch 4 card 23
**Issue:** `IsDirectiveBlock` receives an already-stripped block, but card 23 strips with `StripLineComment(raw, "#")`, so `#!/usr/bin/env python` arrives as `!/usr/bin/env python` and card 13's "`#!` shebang" test never fires; Go's `//go:build` → `go:build` survives stripping, which is why only Python breaks.
**Fix:** state the matched form on the *stripped* text (leading `!`, plus `coding[:=]`), or pass the raw block for the Python case.

### [BLOCKING:design] Card 43 contradicts itself on config resolution frequency
**Location:** batch 7 card 43
**Issue:** One paragraph requires "Resolve the value once per invocation, before the single-argument branch and the batch branch diverge, so a batch call cannot re-read the config once per argument"; the next requires "the file tier per argument" because a batch may span directories.
**Fix:** pick one — per-argument file-tier resolution with the flag tier hoisted, or a single per-invocation resolution — and delete the other sentence.

### [BLOCKING:design] C# member node kinds are never enumerated
**Location:** batch 4 card 25 (and card 27's tests)
**Issue:** Requirements say members are "`method_declaration`, and the constructor/property-free set this design lists" — no such set is listed anywhere in the plan or overview, so whether `constructor_declaration`, `property_declaration`, `operator_declaration`, etc. produce symbols is undecided; card 27 tests only methods.
**Fix:** name the exact closed set of C# member kinds emitted as `KindMethod`, and state explicitly which kinds are deliberately excluded.

### [BLOCKING:design] Grouped Go type branch keyed on spec count misclassifies one-spec groups
**Location:** batch 3 card 19
**Issue:** "Ungrouped — the `type_declaration` holds exactly one spec child" vs "Grouped — two or more spec children"; a legal single-spec `type ( X int )` takes the ungrouped branch, so `Signature` is cut from the `type_declaration` and emits `type (\n X int\n)` with a wrong range.
**Fix:** branch on the presence of the `(` child of `type_declaration`, not on the spec count, and add that fixture to card 21's table.

### [BLOCKING:consistency] CLI is told to import an engine subpackage directly
**Location:** batch 6 cards 36 and 37 (`registry.ExtensionLanguages()` for `--lang` validation)
**Issue:** `internal/cli` today imports nothing under `internal/quarryengine` — every engine identifier reaches it through `quarry/facade.go` (verified: zero `quarryengine` imports in `internal/cli`). Card 34 does not re-export `ExtensionLanguages`, so these cards would make `toc.go` the first direct engine-internal import in the CLI.
**Fix:** re-export the extension-language vocabulary through the facade in card 34, or state in card 36 why the facade convention is deliberately broken here.

### [BLOCKING:scope] Two cards name identifiers from files absent from their Context
**Location:** batch 7 cards 42 and 45
**Issue:** Card 42's `parseDocSentences` returns `quarry.TOCAllSentences`, but `quarry/facade.go` is not in its `Context:`/`Edits:`; card 45 tests `resolveDocSentences`, which card 43 declares in `internal/cli/toc.go`, yet that file is in neither `Context:` nor `Edits:` (and it also asserts on `quarry.TOCAllSentences`).
**Fix:** add `quarry/facade.go` to card 42's Context, and `internal/cli/toc.go` plus `quarry/facade.go` to card 45's.

### [NIT:consistency] treesitter reuses ErrNoLanguage outside its documented meaning
**Location:** batch 1 card 3; batch 8 card 49
**Issue:** `WithTree` returns an error wrapping `quarryengine.ErrNoLanguage`, whose own doc comment in `errors.go` and whose entry in `internal/quarryengine/doc.go` both define it as "no registry entry's markers matched under the target directory"; card 49's sentinel-section rewrite only adds `ErrLanguageUnsupported` and never revisits that now-inaccurate definition.
**Fix:** either widen `ErrNoLanguage`'s documented meaning in card 49, or have card 3 return a plain unwrapped error for an unwired grammar name.

### [NIT:consistency] The toc-dir half of the config decision is unreachable
**Location:** overview "per-directory configuration" decision; batch 7 cards 41 and 43
**Issue:** The decision and card 41's required doc comment both say `targetDir` is "the directory argument for `toc dir`", but card 43 registers `--doc-sentences` on `toc file` only and no `toc dir` path ever calls `resolveTOCConfigPath`, so that half never executes.
**Fix:** say plainly that the toc-dir case is reserved-but-unused, or drop it from the decision and from card 41's doc-comment requirement.

## Verdict

REQUEST_CHANGES
Four extraction/config rules are wrong or undecided; two Context lists and one import convention need fixing.
MILL_REVIEW_END
