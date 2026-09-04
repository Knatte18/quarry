# quarry output formats — after, rewritten CLI

Every file in this directory is one real invocation of the rewritten `quarry` CLI's `toc` verb,
run in-process (`internal/cli/after_test.go`'s `TestAfterGoldens`) against a Loomyard checkout at
HEAD `72c23d9` — the same checkout and commit the "before" side and `docs/rewrite-plan.md` §4 were
taken from.
No absolute path is recorded here: the checkout is identified by name and pin only.

## Before-to-after mapping

| before | after | notes |
|---|---|---|
| `../toc-dir.txt` | `toc-dir.txt` | `toc dir internal/logger` becomes `toc internal/logger` |
| `../toc-file.txt` | `toc-file.txt` | `toc file internal/logger/logger.go` becomes `toc internal/logger/logger.go` |
| `../toc-dir-compact.txt` | *no successor, by design* | the compact view is not carried forward — see below |
| `../toc-file-compact.txt` | *no successor, by design* | the compact view is not carried forward — see below |
| *(none)* | `toc-dir-text.txt` | new: the lossless text view, `--text internal/logger` |
| *(none)* | `toc-file-text.txt` | new: the lossless text view, `--text internal/logger/logger.go` |

**The compact view is gone, not replaced.**
It was the lossy one-sentence-per-file view whose precision loss (0.96 to 0.82,
`docs/rewrite-plan.md` §1 lesson 4) is the rewrite's fourth measured lesson: extraction is
complete, a view filters, no view is ever forced.
The lossless text view (`--text`) is what replaces it in spirit — same intent, no lost precision —
but it is a new view over the same complete data,
not a continuation of the compact view's own lossy grammar.

## What changed between the two sides

Reading the generated files, not asserting from memory:

- **Key order.** The after side's JSON key order is the answer struct's declaration order
  (`dir`, `package`, `language`, `doc`, `files`, ...), stable and byte-comparable.
  The before side's key order is alphabetical (`dirs`, `files`, `ok`), which is what a marshalled
  Go map produces in V1, and `generated` sorts before `header` for the same reason.
- **No boilerplate fields.** `ok: true`, `test: false`, `generated: false`, and the empty
  `dirs: []` are gone: shared facts are stated once and defaults are never emitted.
- **No repeated directory prefix.** Every file path on the before side repeats the query's own
  directory (`"path": "internal/logger/logger.go"`); the after side gives a bare `name` under the
  directory answer's own `dir`, once.
- **One verb, not a split.** The before side is `quarry toc dir <target>` versus
  `quarry toc file <target>`; the after side is `quarry toc <target>` — the CLI, not the caller,
  tells directory from file.

## Regenerating and the regression gate

These four files are also the golden fixtures `internal/cli/after_test.go` compares against,
so the committed evidence and the regression gate cannot disagree.
Regenerating them is:

```
LADDER_LOOMYARD_REPO=<checkout pinned at 72c23d9> go test ./internal/cli/ -run TestAfter -update
```

Each `.txt` file is exactly the invocation line (`$ quarry toc ...`), a blank line,
and the command's stdout verbatim, with no exit-code trailer.
All four commands exit 0, which is asserted by the test, not recorded in the file — unlike the
before side's own `INDEX.md`, whose claim that "the exit code is at the bottom of each file" is
untrue of the four before-side `toc` files it describes.
