MILL_REVIEW_BEGIN
# Review: Thin quarry/ facade over internal/quarryengine

```yaml
verdict: APPROVE
reviewer_model: sonnetmax
reviewer_self_id: Claude Sonnet 5 (model id claude-sonnet-5)
reviewed_file: _mill/discussion.md
date: 2026-08-27
```

## Findings

### [NIT:consistency] doc.go rationale overclaims uniform section→package mapping
**Section:** Decisions → `doc.go` splits three ways **Issue:** rationale states "every section already maps cleanly onto exactly one of the new packages," but doc.go's "The generalized LSP client" section (L38–98) embeds the resolver/ambiguity paragraphs (L65–87) that this same Decision assigns to `query`, not `lsp` — one section splits mid-body across two packages, not cleanly onto one. **Fix:** soften the uniformity claim or state explicitly that the LSP-client section needs a paragraph-level (not section-level) split.

### [NIT:design] Package-doc host file unnamed for multi-file subpackages
**Section:** Decisions → `doc.go` splits three ways **Issue:** "a package doc comment at the head of each subpackage's primary file" names no file for `daemon` (4 files: ensureserver.go/toolchain.go/daemonstate.go/probe.go) or `query` (3 files: definition.go/refs.go/symbol.go), so "primary file" is undefined for exactly the two packages that need it most. **Fix:** name the host file per subpackage, or state that a lightweight per-package doc.go is an acceptable alternative.

### [NIT:design] Two divergent names left open for the relocated log handler
**Section:** Technical context → identifiers that must become exported **Issue:** "defaultLogHandler ... exported as quarryengine.Logger (or kept as an exported package-level *slog.Logger)" reads as two candidate identifier names, not a single decision, for a symbol with 7 call sites across `lsp`/`daemon`. **Fix:** pick one exported name (e.g. `quarryengine.Logger`) and drop the parenthetical alternative.

## Verdict

APPROVE
Extensively self-consistent and source-verified; only minor doc-comment/naming loose ends remain.
MILL_REVIEW_END
