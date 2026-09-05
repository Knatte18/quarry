All conflicts are resolved and staged (all 6 files now show as "modified" under "Changes to be committed" with no "Unmerged paths" section remaining). No content was discarded — both sides' additions (the glyphs view and the delta/git-delta feature) were combined in every conflicting doc comment and test file, with verb counts, renderer counts, and message strings updated consistently across all six files. Build, vet, and the full test suite (`go build ./...`, `go test ./internal/... ./quarry/...`) pass.

Summary of resolved files:
- /home/knatte/Code/quarry/wts/glyphs-verb/quarry/doc.go — merged package doc comment: now describes six queries (TOC, Resolve, Expand, Delta, Name, Glyphs) and thirteen renderers, reconciling the glyphs-view and delta-git paragraphs.
- /home/knatte/Code/quarry/wts/glyphs-verb/quarry/render.go — merged file header describing five JSON renderers here plus the sixth (RenderGlyphsJSON) declared in view.go.
- /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/cli.go — merged Run's doc comment (four repository verbs post-rewrite, plus the glyphs-rewrite note).
- /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/doc.go — merged package doc comment describing six verbs/five pipelines, including both glyphs and delta.
- /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/flags.go — merged request struct doc, parseArgs doc comment, and the verb-gate/"no verb given" messages to include all six verbs (toc, glyphs, resolve, expand, delta, name).
- /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/flags_test.go — reconstructed the interleaved TestParseArgs_Glyphs, TestParseArgs_GlyphsIsTheFrozenTOCExpansion, TestParseArgs_Delta, and TestParseArgs_DeltaFlagValidity functions in full (git's 3-way diff had woven them together due to structurally similar table-test scaffolding); updated the shared usage-error message literals and doc comments.

No files fell under any batch's Deletes: list, and no UD (modify/delete) conflicts occurred, so no deletions were staged.

{"status":"success"}
