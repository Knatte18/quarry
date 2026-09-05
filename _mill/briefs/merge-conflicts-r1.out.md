All conflicts fixed, staged. The only conflicting file was `internal/cli/loomyard_test.go`, which had one conflict block (the `updateGoldens` var declaration and its doc comment). Both sides made real, independent edits — origin/main updated the golden path from `docs/research/output-formats/after` to `internal/cli/testdata` (the goldens-move commit), while HEAD widened the description to also cover the new `name/` goldens — so I combined both into a single doc comment/description rather than picking one side. I verified no conflict markers or stale path references (`docs/research/output-formats/after`) remain anywhere in the file, and re-read it in full for consistency.

File resolved: `/home/knatte/Code/quarry/wts/glyph-maker/internal/cli/loomyard_test.go`

{"status":"success"}