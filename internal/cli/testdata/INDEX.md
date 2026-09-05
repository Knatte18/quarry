# quarry output formats — after, rewritten CLI

Every `.txt` file in this directory is one real invocation of the rewritten `quarry` CLI, run
in-process (`internal/cli/after_test.go`'s `TestAfterGoldens`) against a Loomyard checkout at HEAD
`72c23d9`
— the same checkout and commit the "before" side and `docs/rewrite-plan.md` §4 were taken from.
The table now spans three verbs — `toc`, `resolve`, and `expand` — not `toc` alone.
No absolute path is recorded here: the checkout is identified by name and pin only.

## Before-to-after mapping

This table is total: every before-side file has its own row, naming either its successor or
stating "no successor" with a reason, and every after-side file has its own exit-code cell. That
totality is what makes "the after side covers the same command set — nothing missing" a checkable
claim rather than an assertion.

| before | after | exit | notes |
|---|---|---|---|
| `docs/research/output-formats/toc-dir.txt` | `toc-dir.txt` | 0 | `toc dir internal/logger` becomes `toc internal/logger` |
| `docs/research/output-formats/toc-file.txt` | `toc-file.txt` | 0 | `toc file internal/logger/logger.go` becomes `toc internal/logger/logger.go` |
| `docs/research/output-formats/toc-dir-compact.txt` | *no successor, by design* | — | the compact view is not carried forward — see below |
| `docs/research/output-formats/toc-file-compact.txt` | *no successor, by design* | — | the compact view is not carried forward — see below |
| *(none)* | `toc-dir-text.txt` | 0 | new: the lossless text view, `--text internal/logger` |
| *(none)* | `toc-file-text.txt` | 0 | new: the lossless text view, `--text internal/logger/logger.go` |
| `docs/research/output-formats/definition.txt` | `resolve-glyph.txt` | 0 | the in-file flag and the absolute path are gone; a glyph replaces a bare name |
| `docs/research/output-formats/definition-ambiguous.txt` | `resolve-method.txt` | 0 | the old ambiguity does not survive the glyph grammar. The old query was a fuzzy name match across two receivers' methods; the glyph names exactly one and answers found. The old side's success-with-a-usage-exit-code was the addressing defect, not a fact about the repository |
| `docs/research/output-formats/symbol.txt` | `expand-type.txt` | 0 | the old query was a fuzzy workspace passthrough returning an unrelated package's test function and undecoded kind integers; the new one answers the same question exactly — the head plus every member, named kinds, glyph identifiers, no cross-package noise |
| `docs/research/output-formats/impact.txt` and `docs/research/output-formats/impact-file-scope.txt` | *no successor, phase 2* | — | that query needs a type checker; it is a later wave's task |
| `docs/research/output-formats/assert-no-callers.txt` | *no successor, phase 2* | — | same, and cited to the same plan section |
| `docs/research/output-formats/refs.txt` | *no successor, by design* | — | the plan states there is no reference query in phase 1: dropped after measurement, not deferred |
| *(none)* | `resolve-glyph-text.txt` | 0 | new: the lossless text view of the same query |
| *(none)* | `expand-type-text.txt` | 0 | new: the lossless text view of the same query |
| *(none)* | `resolve-not-found.txt` | 1 | new: the unit-found miss the plan names as the validator's whole reason for that key. The old side had no equivalent — its definition query on a missing name returned an empty list |
| *(none)* | `resolve-self-file.txt` | 0 | new: a file addressed as its own glyph — the same repository-relative path with a trailing `#` — which makes a non-code deliverable a checkable plan target. The old side had no path-target form at all |
| *(none)* | `resolve-self-dir.txt` | 0 | new: a directory addressed as its own glyph, answering with the same listing `toc` would produce for that directory |
| *(none)* | `resolve-self-file-text.txt` | 0 | new: the lossless text view of `resolve-self-file.txt`'s query |
| *(none)* | `resolve-self-dir-text.txt` | 0 | new: the lossless text view of `resolve-self-dir.txt`'s query |
| *(none)* | `expand-not-a-type.txt` | 1 | new: the plan's rule that the glyph must name a type, and that on any other kind the answer names the kind |

**The compact view is gone, not replaced.**
It was the lossy one-sentence-per-file view whose precision loss (0.96 to 0.82,
`docs/rewrite-plan.md` §1 lesson 4) is the rewrite's fourth measured lesson: extraction is
complete, a view filters, no view is ever forced.
The lossless text view (`--text`) is what replaces it in spirit — same intent, no lost precision —
but it is a new view over the same complete data,
not a continuation of the compact view's own lossy grammar.

## What changed between the two sides

### toc

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

### resolve and expand

Reading `resolve-glyph.txt`, `resolve-method.txt`, `expand-type.txt`, `resolve-not-found.txt`,
`resolve-self-file.txt` and `expand-not-a-type.txt` against their before-side counterparts:

- **A glyph, not a bare name plus `--in-file`.** `definition.txt`'s
  `--in-file internal/logger/logger.go stderrHandlerSnapshot` becomes
  `resolve internal/logger#stderrHandlerSnapshot`: the unit and the name are one addressable
  string, and there is no flag naming the file the caller already had to know to type.
- **No absolute path, anywhere.** Every before-side `file` value is
  `/home/knatte/Code/loomyard/wts/loomyard/internal/logger/logger.go`; every after-side `file`
  value is `internal/logger/logger.go`, repository-relative, because the checkout's own location
  is never part of the answer.
- **A full symbol, not a point.** `definition.txt`'s answer is a `{character, file, line}` triple;
  `resolve-glyph.txt`'s is a complete symbol — `id`, `kind`, `file`, `start`, `sigend`, `end`,
  `signature`, and `doc` — the same shape `toc`'s own `--symbols` entries carry, because all three
  verbs answer with one symbol shape rather than each inventing its own.
- **Ambiguity is a payload, not an exit-2 usage failure.** `definition-ambiguous.txt` answers
  `Handle` with two absolute-path candidates and exits 2, the same code an unparseable flag gets.
  The glyph in `resolve-method.txt` names exactly one receiver's method and answers found at exit
  0; nothing here is still ambiguous under the new grammar, because the old ambiguity was an
  addressing defect the glyph closes, not a fact about the repository the CLI is answering about.
- **`expand` answers the whole type, not a filtered symbol search.** `symbol.txt`'s
  `dualHandler` query is a fuzzy, cross-package name match: 24 hits including an unrelated
  `internal/clihelp` test function and a `internal/loomrecipe` test function, each carrying an
  undecoded integer `kind` (6, 8, 12, 23, ...). `expand-type.txt`'s `internal/logger#dualHandler`
  answers with exactly the type's head plus its five own members — `stderr`, `Enabled`, `Handle`,
  `WithAttrs`, `WithGroup` — each with a named `kind` ("type", "method") and a glyph `id`, and
  nothing from another package.
- **Two answers with no before-side analogue at all.** `resolve-not-found.txt`'s `unit: "found"`
  on a miss is the validator's whole reason for that key, per the plan; the old `definition` query
  on a missing name simply returned an empty `definitions` list, with no way to say whether the
  file itself was even found. `expand-not-a-type.txt`'s failure — naming the kind a glyph actually
  resolved to when it names something other than a type — has no old-side equivalent either: the
  old queries never asked "is this a type" in the first place.

## Regenerating and the regression gate

These fifteen files are also the golden fixtures `internal/cli/after_test.go` compares against,
so the committed evidence and the regression gate cannot disagree. That is why they live here,
beside the test that owns them, rather than under `docs/research/output-formats/`, which is a
frozen record — the before-side files this table's left column names are still there.
Regenerating them is:

```
LADDER_LOOMYARD_REPO=<checkout pinned at 72c23d9> go test ./internal/cli/ -run TestAfter -update
```

Each `.txt` file is exactly the invocation line (`$ quarry <verb> ...`), a blank line,
and the command's stdout verbatim, with no exit-code trailer.
The expected exit code for each invocation lives in the table above and in the golden table
inside `internal/cli/after_test.go` itself — not in the file — unlike the before side's own
`docs/research/output-formats/INDEX.md`, whose claim that "the exit code is at the bottom of each
file" is untrue of the four before-side `toc` files it describes.
