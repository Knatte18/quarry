Good — the SHA is exactly 40 characters: `de6a2a868ee86789376e02410dae1e598210324d`.

All 9 cards (8 through 16) in batch `02-toc-scaffolding.md` were committed this turn, matching the 9 declared in the batch file's frontmatter. The verify command `go test ./internal/quarryengine ./internal/quarryengine/toc` passed, and the working tree is clean.

Summary of files touched:
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/errors.go` — added `ErrLanguageUnsupported`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/doc.go` — new package doc
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/types.go` — `Kind`, `Symbol`, `FileTOC`, `DirEntry`, `DirTOC`, `Options`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/strategy.go` — `Strategy`, `Register`/`StrategyFor`/`Implemented`, `swapRegistry`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/comments.go` — `StripLineComment`, `StripComment`, `StripXMLDocTags`, `FirstParagraph`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/sentences.go` — `FirstSentences`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/classify.go` — `IsDirectiveBlock`, `TestFileByName`, `GeneratedByBanner`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/toc/comments_test.go`, `sentences_test.go`, `classify_test.go` — new tests
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/layering_test.go` and `seam_enforcement_test.go` — toc layering rows and raised `minPackageDirs` floors (6→8)

{"status":"success","commit_sha":"de6a2a868ee86789376e02410dae1e598210324d","session_id":"7ac2c536-be61-4a74-baec-c982562bbf8c","cards_done":[8,9,10,11,12,13,14,15,16]}
