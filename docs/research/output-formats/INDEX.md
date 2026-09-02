# quarry output formats — actual runs, 2026-09-02

Every file is one real invocation against the Loomyard checkout
(`/home/knatte/Code/loomyard/wts/loomyard`, HEAD 72c23d9), run from that directory with the binary
built from quarry `d464334`. JSON is pretty-printed; the exit code is at the bottom of each file.

| file | command | shape |
|---|---|---|
| `toc-file.txt` | `toc file internal/logger/logger.go` | decoded, self-describing |
| `toc-file-compact.txt` | `toc file --compact …` | plain text |
| `toc-dir.txt` | `toc dir internal/logger` | decoded, self-describing |
| `toc-dir-compact.txt` | `toc dir --compact …` | plain text |
| `impact.txt` | `impact --in-file … --within … stderrHandlerSnapshot` | decoded, richest envelope |
| `impact-file-scope.txt` | same, `newDualHandler` | a caller at file scope — no enclosing declaration |
| `refs.txt` | `refs --in-file … newDualHandler` | bare positions |
| `definition.txt` | `definition --in-file … stderrHandlerSnapshot` | bare position |
| `definition-ambiguous.txt` | `definition --in-file … Handle` | refuses, lists candidates |
| `symbol.txt` | `symbol dualHandler` | raw LSP passthrough |
| `assert-no-callers.txt` | `assert-no-callers --within … --except … ` | gate envelope |

## What the outputs actually show

**Three different design generations are visible in one CLI.**

*Decoded and self-describing* — `toc` and `impact`. Named kinds (`"kind": "function"`), `owner`,
`signature`, docstrings, and line numbers under names that say what they mean (`start`, `sigend`,
`end`, `enclosing_range`). A reader needs no protocol knowledge. `impact` is the richest: each caller
carries the call site *and* the enclosing declaration's identity and span.

*Thin position lists* — `refs` and `definition`. `{file, line, character}` and nothing else. This is
the shape that duplicates what a grep already returned (see HANDOFF §6, row 1).

*Raw LSP passthrough* — `symbol`. Three problems visible in one response:

- **`"kind": 12`, `23`, `8`, `6`** — undecoded LSP `SymbolKind` integers. Nothing in the output says
  12 is a function and 23 is a struct. Every other command in the same CLI emits named kinds.
- **Inconsistent qualification** — one entry is
  `github.com/Knatte18/loomyard/internal/clihelp.TestExecute_FailHandlerReturnsOne` (full import path),
  the next is bare `dualHandler`, the next is `dualHandler.transform`. Three naming conventions in one
  array.
- **Fuzzy-match noise** — querying `dualHandler` returns `TestExecute_FailHandlerReturnsOne` from an
  unrelated package. `workspace/symbol` is a fuzzy query by protocol definition; the envelope does not
  say so, and gives the caller nothing to filter on.

## Two inconsistencies worth fixing regardless of any strategy decision

**`definition` returns `"ok": true` while exiting 2.** On the ambiguous `Handle` it correctly refuses
and lists both candidates — good behaviour — but the envelope says `ok: true` on what the exit code
calls a failure. A consumer checking `ok` reads success; a consumer checking the exit code reads
failure. See `definition-ambiguous.txt`.

**Addressing differs per command.** `refs`, `definition` and `impact` take `--in-file <path> <name>`.
`assert-no-callers` does not — it rejects `--in-file` as an unknown flag and takes only a bare name or
`file:line:col`. Same engine, same resolution, three commands sharing a flag and a fourth not.
