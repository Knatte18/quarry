# Batch: evidence-and-status-gate

```yaml
task: "Facade + CLI, resolve + expand (T5b)"
batch: "evidence-and-status-gate"
number: 4
cards: 4
verify: go test ./internal/cli/
depends-on: [3]
```

## Batch Scope

This batch ships the task's committed evidence and its machine-independent done-criterion gate: a new end-to-end test proving all four resolution statuses through the command line on a self-built fixture tree, the golden table widened to three verbs and per-case exit codes, the eight new evidence files themselves, and the rewritten index that makes "the after side covers the same command set — nothing missing" a checkable claim rather than an assertion.

It is one batch because the golden table, the golden files and the index are one artifact in three parts — the committed evidence and the regression gate are the same files — and because the status gate belongs beside them: it exists precisely to cover the two statuses the evidence files cannot carry.

Batch-local decisions beyond the overview's Shared Decisions:

- The evidence files keep the existing format exactly: the invocation line, a blank line, and stdout verbatim, with no exit-code trailer. The expected exit code moves into the test table and into the index's own column instead. Adding trailers would regenerate four already-committed files for a cosmetic reason, or leave two formats in one directory.
- The ambiguous and multipart statuses get no evidence file. The evidence test skips on any machine without a Loomyard checkout, and that repository is not known to contain a build-tag-duplicated declaration or a several-declaration initialiser; a golden that silently is not the case it claims to be is worse than none. Card 14's self-built tree covers them instead, and runs everywhere.
- Card 15 opens a deliberate red window: it compares against eight golden files before card 16 creates them, so on a machine with the pinned checkout the package's tests fail between those two commits. That window is expected, is bounded to within this batch, and is exactly the window the existing file's own header comment already documents for the same reason. Do not paper it over by skipping on a missing golden: after card 16 a missing golden is a real regression, and a skip there would hide the failure the gate exists to catch.

## Cards

### Card 14: prove all four statuses through the command line on a self-built tree

- **Context:**
  - `internal/cli/scratchtree_test.go`
  - `internal/cli/cli_test.go`
  - `internal/cli/cli.go`
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
  - `internal/engine/expand.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/glyph5_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/cli/glyph5_test.go`. It builds its own repository with the package's existing `writeScratchTree` helper — never the system temp directory — and drives `Run` end to end with buffer sinks and an explicit root flag, asserting both the exit code and the decoded payload for each case below. Decode the payload with the standard library's JSON decoder into a local struct or a generic map; do not import the engine.

  The fixture tree needs, at minimum: a package directory holding one free function and one type with at least one method; a second file in that same package declaring the same function name under a different build tag from the first, so one glyph matches two different declarations; and a file declaring two package-level initialiser functions, so one glyph matches several parts of one symbol.

  The cases, each named for the status it proves:

  | case | invocation | expected |
  |---|---|---|
  | found | the resolve verb on a glyph naming the free function | exit 0, status found, one symbol |
  | not_found, unit found | the resolve verb on a glyph whose unit exists and whose member does not | exit 1, status not-found, unit found |
  | not_found, unit not_found | the resolve verb on a glyph whose unit directory does not exist | exit 1, status not-found, unit not-found |
  | ambiguous | the resolve verb on the build-tag-duplicated function's glyph | exit 1, status ambiguous, candidates present, symbols absent |
  | multipart | the resolve verb on the initialiser glyph | exit 0, status multipart, every part present in the symbols list |
  | not a type | the expand verb on the free function's glyph | exit 1, the failure envelope on stdout, and the exact sentence on stderr with no usage text |

  Assert on the candidates and symbols keys' presence and absence, not only on the status word: the separate key is the signal that nothing was chosen, and a test that reads only the status would pass against an implementation that emitted the wrong one.

  Do not point the root flag at another package's fixture directory. Each package keeps its own copy of its fixtures, which is the convention the existing helper's own comment already states.

  The file's header comment states what it is: the machine-independent proof that all four statuses answer correctly through the command line, written because the evidence goldens cannot carry two of them — they skip wherever the checkout environment variable is unset, and that repository is not known to contain either case.
- **Commit:** `test(cli): prove all four resolution statuses end to end`

### Card 15: the golden table gains a verb, an expected exit code, and eight rows

- **Context:**
  - `internal/cli/loomyard_test.go`
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/cli/after_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `internal/cli/after_test.go`, give `afterGoldenCase` two new fields: a verb, and an expected exit code. Today the table's own type hardcodes the table-of-contents verb in two places — the argument slice the test builds and the invocation line each golden records — and both become the row's own verb field. Keep every field's existing comment accurate after the change, and keep each row spelling its invocation-line suffix literally rather than deriving it from the argument slice, so a machine-specific root flag cannot leak into a committed file by construction.

  Turn the test's two blanket assertions into per-case expectations: the exit code is compared against the row's expected code rather than against success, and the standard error stream is asserted empty only for rows that expect success. Two of the new rows exit 1, and one of those also writes to standard error, because the failure path always does.

  Set the four existing rows' verb to the table-of-contents verb and their expected exit code to 0. Add these eight rows, each against the same pinned checkout:

  | golden | verb and arguments | exit |
  |---|---|---|
  | `resolve-glyph.txt` | resolve, `internal/logger#stderrHandlerSnapshot` | 0 |
  | `resolve-glyph-text.txt` | resolve, the text flag and the same glyph | 0 |
  | `resolve-method.txt` | resolve, `internal/logger#dualHandler.Handle` | 0 |
  | `resolve-not-found.txt` | resolve, `internal/logger#noSuchSymbol` | 1 |
  | `resolve-path.txt` | resolve, `internal/logger/logger.go` | 0 |
  | `expand-type.txt` | expand, `internal/logger#dualHandler` | 0 |
  | `expand-type-text.txt` | expand, the text flag and the same glyph | 0 |
  | `expand-not-a-type.txt` | expand, `internal/logger#newDualHandler` | 1 |

  Keep the test function's name exactly as it is. The regeneration instructions the evidence directory records match it by prefix, so a rename would make that regeneration a silent no-op producing no files at all — the name and the command are load-bearing on each other.

  Extend the file's header comment to state that the table now spans three verbs, that the expected exit code lives in the table and in the evidence index rather than in a trailer inside each file, and that the red window this card opens closes at the next card.
- **Commit:** `test(cli): widen the golden table to three verbs and per-case exit codes`

### Card 16: generate the eight evidence files

- **Context:**
  - `internal/cli/after_test.go`
  - `internal/cli/loomyard_test.go`
  - `docs/research/output-formats/after/toc-dir.txt`
  - `docs/research/output-formats/after/toc-file.txt`
- **Edits:** none
- **Creates:**
  - `docs/research/output-formats/after/expand-not-a-type.txt`
  - `docs/research/output-formats/after/expand-type-text.txt`
  - `docs/research/output-formats/after/expand-type.txt`
  - `docs/research/output-formats/after/resolve-glyph-text.txt`
  - `docs/research/output-formats/after/resolve-glyph.txt`
  - `docs/research/output-formats/after/resolve-method.txt`
  - `docs/research/output-formats/after/resolve-not-found.txt`
  - `docs/research/output-formats/after/resolve-path.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Produce the eight evidence files by running the golden table's own regeneration mode against a Loomyard checkout pinned at the commit the previous card's table names. Take the checkout's path from the `LADDER_LOOMYARD_REPO` environment variable, sourced from the gitignored scratch environment file this repository already uses for it — never from a tracked file, and never hardcoded into any file this card touches. Concretely, from the repository root:

  ```
  set -a; . ./.scratch/ladder.env; set +a; go test ./internal/cli/ -run TestAfter -update
  ```

  If that environment file is absent on the machine running this card, stop and report rather than inventing a path: the eight files cannot be produced without the pinned checkout, and a hand-written approximation of them would be evidence of nothing.

  Then re-run the same command without the regeneration flag and confirm every row passes, including the four already-committed table-of-contents files, which must compare equal without having been regenerated. That equality is this task's own proof that the shared-encoder and symbol-line refactors changed nothing.

  Before committing, inspect each of the eight produced files and confirm: no absolute path appears anywhere in it; the first line is the invocation line naming the verb and the repository-relative target with no root flag; the second line is blank; and there is no exit-code trailer at the end.

  Do not hand-edit any produced file. If one is wrong, the implementation or the table row that produced it is wrong, and that is what gets fixed.
- **Commit:** `docs(evidence): add the eight resolve and expand after-side outputs`

### Card 17: rewrite the evidence index so nothing missing is checkable

- **Context:**
  - `docs/research/output-formats/INDEX.md`
  - `docs/research/output-formats/after/resolve-glyph.txt`
  - `docs/research/output-formats/after/resolve-method.txt`
  - `docs/research/output-formats/after/expand-type.txt`
  - `internal/cli/after_test.go`
- **Edits:**
  - `docs/research/output-formats/after/INDEX.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Rewrite `docs/research/output-formats/after/INDEX.md` so its mapping table carries a row for every before-side file — each either naming a successor or stating "no successor, by design" with the reason — plus an exit-code column for every after-side file. That is what makes the done-criterion checkable: the claim is that the after side covers the same command set with nothing missing, and it is only verifiable if every before-side file has an explicit row and every deliberate absence is stated with its reason.

  Keep the six existing rows exactly as they are, adding only an exit-code cell to each (all four after-side files there exit 0): the two table-of-contents before-to-after rows, the two compact-view no-successor rows, and the two new-text-view rows.

  Add these rows:

  | before | after | note |
  |---|---|---|
  | the before-side definition output | `resolve-glyph.txt` | the in-file flag and the absolute path are gone; a glyph replaces a bare name |
  | the before-side ambiguous-definition output | `resolve-method.txt` | the old ambiguity does not survive the glyph grammar. The old query was a fuzzy name match across two receivers' methods; the glyph names exactly one and answers found. The old side's success-with-a-usage-exit-code was the addressing defect, not a fact about the repository |
  | the before-side symbol output | `expand-type.txt` | the old query was a fuzzy workspace passthrough returning an unrelated package's test function and undecoded kind integers; the new one answers the same question exactly — the head plus every member, named kinds, glyph identifiers, no cross-package noise |
  | the two before-side impact outputs | no successor, phase 2 | that query needs a type checker; it is a later wave's task |
  | the before-side assert-no-callers output | no successor, phase 2 | same, and cited to the same plan section |
  | the before-side refs output | no successor, by design | the plan states there is no reference query in phase 1: dropped after measurement, not deferred |
  | none | `resolve-glyph-text.txt` | new: the lossless text view of the same query |
  | none | `expand-type-text.txt` | new: the lossless text view of the same query |
  | none | `resolve-not-found.txt` | new: the unit-found miss the plan names as the validator's whole reason for that key. The old side had no equivalent — its definition query on a missing name returned an empty list |
  | none | `resolve-path.txt` | new: a repository-relative path as a target, which makes a non-code deliverable a checkable plan target. The old side had no path-target form at all |
  | none | `expand-not-a-type.txt` | new: the plan's rule that the glyph must name a type, and that on any other kind the answer names the kind |

  Reference each before-side file by its actual relative path in the table, as the existing rows already do. The three new after-side files that answer no before-side question take a none row rather than being left out, which is the same device the existing text-view rows already use; that is what makes the exit-code column total, so every after-side file records its exit code in a tracked file.

  Add a short "what changed" paragraph for the resolve and expand side, matching the one already written for the table-of-contents side, drawn from reading the produced files rather than asserted from memory.

  Update the two places the existing text is now wrong: the opening paragraph, which says every file in the directory is one invocation of the table-of-contents verb, and the closing paragraph, which says these four files are the goldens and that all four exit 0. The closing paragraph must now state that the expected exit code lives in the table above and in the test's own table, and that the files themselves still carry no trailer. Keep the regeneration command block and the statement that the committed evidence and the regression gate are the same files.
- **Commit:** `docs(evidence): rewrite the after-side index with a total before-to-after mapping`

## Batch Tests

`verify: go test ./internal/cli/` runs the whole `internal/cli` test binary, which is the scope this batch needs: it covers card 14's new status gate and card 15's widened golden table together, and the batch touches nothing outside that package.

The golden comparison inside that run is conditional by design. On a machine whose checkout environment variable is unset, the golden test skips and the batch's verify still proves card 14's status gate, which is the part that must run everywhere including on a build machine. On a machine that has the pinned checkout, the same command additionally compares all twelve golden files byte for byte, including the four already-committed table-of-contents files that must still compare equal without regeneration.

Card 16 is the one card whose work is not proved by the batch verify alone: it is verified by re-running the same command without the regeneration flag on the machine that produced the files, which the card requires explicitly.
