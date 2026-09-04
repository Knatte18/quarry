All 4 cards (31, 32, 33, 34) have matching commits in the log, matching the batch's declared 4 cards exactly. 4 of 4 cards committed — this is a genuinely complete batch, verified against the commit log by matching each `Commit:` message.

{"status":"success","commit_sha":"ff40b59f1c08c036bbcc6e5a9b120696f2e0839a","session_id":"91fee318-329b-4fdc-be4e-64c1930c0045","cards_done":[31,32,33,34]}

Summary of work:
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/resolve.go` (new) — `unitDirs` (literal-first unit->directory lookup with collision flag), `dirExists`, `dirChainBelowRoot`, `symbolsOfUnit`/`symbolsOfDir` (per-unit parse-once primitive with per-directory ignore-chain extend/trim), `sameOwner`, and the public `SpansOf` wrapper.
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/errors.go` — rewrote `ErrLanguageUnsupported`'s doc comment for its sole remaining trigger (`SpansOf` called with a non-Go `Lang`).
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/testdata/foo_test/literal.go` (new fixture) and `/home/knatte/Code/quarry/wts/engine-core/internal/engine/resolve_test.go` (new) — full coverage: hit/miss/missing-unit, external-test unit split both directions, literal-first, collision (via `.scratch/`), ignore filter (via `.scratch/`), `ErrLanguageUnsupported` via `errors.Is`, and argument/unspellable-unit validation surfacing `*glyph.ParseError` via `errors.As`.

{"status":"success","commit_sha":"ff40b59f1c08c036bbcc6e5a9b120696f2e0839a","session_id":"91fee318-329b-4fdc-be4e-64c1930c0045","cards_done":[31,32,33,34]}
