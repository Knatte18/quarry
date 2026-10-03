# Batch: parse-once tests and Loomyard measurement

```yaml
task: 'Resolve self-target path: per-call dirPackage memo (GH #34)'
batch: parse-once tests and Loomyard measurement
number: 2
cards: 2
verify: LADDER_LOOMYARD_REPO=/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo/.scratch/loomyard-pin go test -count=1 ./internal/engine/
depends-on: [1]
```

## Batch Scope

This batch pins the acceptance criterion in tests against batch 1's seam (`unitMemo.files`, `fileMemo.builds`) and then measures the fix on the real Loomyard repro.
The tests assert per-file build counts against expected path sets written out from each fixture, never against what the memo saw requested.
The measurement is a read-only, Commit: none card whose result goes to `.scratch/loomyard-measurement.md` for the handoff.

## Cards

### Card 6: Parse-once, equivalence, clause-less and gitignored-target tests

- **Context:**
  - `internal/engine/memo.go`
  - `internal/engine/resolve.go`
  - `internal/engine/toc.go`
  - `internal/engine/answer.go`
  - `internal/engine/toc_test.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/resolve_test.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/memo_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/engine/memo_test.go` in package `engine`, with a file header comment naming what it pins: each file is built at most once per Resolve call, whatever mix of self and member targets reaches its directory, and memo sharing never changes an answer.
  Use `openScratchRepo` and `boolPtr` (both in `toc_test.go`) for fixtures, `newUnitMemo` plus `r.resolve(targets, m)` to keep the memo, and `m.files.builds` as the seam.
  Write one helper, `assertBuilds(t *testing.T, builds map[string]int, want []string)`, that fails unless every path in `want` has count exactly 1 and `builds` has no key outside `want`; its doc comment says the expected set is written out from the fixture, and that the second half pins the lazy, no-eager-parse rule.
  Compare answers with `reflect.DeepEqual`.

  1. `TestResolve_BuildsEachFileOnce`. Fixture `memo-builds-once`, with these files and contents:
     - `pkg/a.go`: `"package pkg\n\nfunc A() {}\n"`
     - `pkg/b.go`: `"package pkg\n\nfunc B() {}\n"`
     - `pkg/b_test.go`: `"package pkg_test\n\nfunc TestB() {}\n"`
     - `pkg/README.md`: `"# pkg\n\nNotes.\n"`
     - `pkg/sub/s.go`: `"// Package sub is a subdirectory.\npackage sub\n"`
     - `pkg/sub/deeper/d.go`: `"package deeper\n"`
     - `other/o.go`: `"package other\n\nfunc O() {}\n"`
     - `untouched/u.go`: `"package untouched\n"`

     Targets, in one `r.resolve` call: `"pkg/a.go#"`, `"pkg/b.go#"`, `"pkg/README.md#"`, `"pkg#"`, `"pkg#A"`, `"pkg#B"`, `"pkg_test#TestB"`, `"other/o.go#"`.
     Assert each result's `Status` is `StatusFound`.
     Then `assertBuilds` with exactly the paths `pkg/a.go`, `pkg/b.go`, `pkg/b_test.go`, `pkg/README.md`, `pkg/sub/s.go`, `other/o.go` (the directory self target reaches `pkg/sub` for its identity and stops there).
     The doc comment explains the expected set: each self file target's directory, each member glyph's `unitDirs` directories, the directory self target's own directory and its direct subdirectories.
  2. `TestResolve_MemoAnswersMatchFreshCalls`, on the same fixture (its own `openScratchRepo` name, `memo-equivalence`) and the same targets:
     - For every self target, the result's `*Listing` equals `r.TOC(unit, TOCOptions{Symbols: boolPtr(false)})` from a fresh call, where `unit` is the target with its trailing `#` removed.
     - For every member target, the result's Symbols field equals `r.SpansOf(g)` for the glyph parsed from the target.
     - `r.Resolve` over the reversed target list answers each target exactly as the forward call did.
     The doc comment says these guard memo keying and sharing, not extraction: both sides run the same extraction code.
  3. `TestResolve_ClauselessFileKeepsPerConsumerRule`. Fixture `memo-clauseless`:
     - `cl/a.go`: `"package cl\n\nfunc A() {}\n"`
     - `cl/broken.go`: `"packag cl\n\nfunc Lost() {}\n"` — tree-sitter recovers `func Lost` but records no clause; the entry is lossy.

     First, a fresh `r.TOC("cl", TOCOptions{Symbols: boolPtr(true)})`: the `broken.go` entry has Lossy true and a non-nil Symbols field containing ID `"cl#Lost"` (the walk consumer includes the clause-less file under unit `cl`).
     Then one `r.resolve` over `"cl/broken.go#"`, `"cl#Lost"`, `"cl#A"` with a constructed memo: `cl#Lost` is `StatusNotFound` with `Unit == StatusFound` (the member consumer excludes the clause-less file); `cl#A` is `StatusFound`; the `cl/broken.go#` listing equals the fresh `r.TOC("cl/broken.go", TOCOptions{Symbols: boolPtr(false)})`.
     Finally `assertBuilds` with `cl/a.go` and `cl/broken.go`.
  4. `TestResolve_GitignoredExplicitTargetVote`. Fixture `memo-gitignored`:
     - `g/.gitignore`: `"secret.go\n"`
     - `g/a.go`: `"package g\n\nfunc A() {}\n"`
     - `g/secret.go`: `"package aaa\n\nfunc S() {}\n"`

     The named, gitignored `secret.go` joins its own target's vote, which ties `aaa` against `g` and picks `aaa`; every other target's vote excludes it and picks `g`.
     For both target orders, `["g/a.go#", "g/secret.go#", "g#A"]` and its reverse, run `r.resolve` with a fresh memo and assert: the `g/a.go#` listing equals the fresh `r.TOC("g/a.go", TOCOptions{Symbols: boolPtr(false)})` and has `Package == "g"`; the `g/secret.go#` listing equals the fresh `r.TOC("g/secret.go", TOCOptions{Symbols: boolPtr(false)})` and has `Package == "aaa"`; `g#A` is `StatusFound`; `assertBuilds` with `g/.gitignore`, `g/a.go`, `g/secret.go` (`.gitignore` is itself a listed file in its directory).
     The doc comment names the bug this catches: memoising the extended vote under the plain directory key.
- **Commit:** `test(engine): pin one record build per file per Resolve call`

### Card 7: Loomyard before/after measurement

- **Context:**
  - `internal/engine/memo.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Measure the fix on the agreed Loomyard repro and write the result to `.scratch/loomyard-measurement.md`.
  Every step is read-only towards the Loomyard prime `/home/knatte/Code/loomyard-LYXHUB/loomyard`: never edit, create or delete a file inside it, never install a binary on `PATH` or into the Go bin dir, and never `cd`; run each command from the quarry worktree root `/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo` and use `go -C <dir>` or `env -C <dir>` to set a working directory for one command.
  Let `W` be that worktree root and `P` the prime.
  1. Baseline binary, built against the quarry v0.2.0 that `P/go.mod` requires: `GOWORK=off go -C P build -o W/.scratch/lyx-v0.2.0 ./cmd/lyx`.
  2. Write `W/.scratch/lyx-work/go.work` containing `go 1.26` and a `use` block listing `P` and `W` (absolute paths); then `GOWORK=W/.scratch/lyx-work/go.work go -C P build -o W/.scratch/lyx-branch ./cmd/lyx`.
  3. Time each binary on the repro, capturing stdout as the findings: `time env -C P W/.scratch/lyx-v0.2.0 webster validate --plan-dir P/.scratch/hub/quarry34-plan > W/.scratch/findings-v0.2.0.json`, then the same with `W/.scratch/lyx-branch` into `W/.scratch/findings-branch.json`.
     A non-zero exit from `validate` is expected when it reports findings; it is not a failure of this card.
     Give the baseline run a Bash timeout of 600000 ms (it took 74 s before the fix).
  4. `cmp W/.scratch/findings-v0.2.0.json W/.scratch/findings-branch.json`.
  5. Write `.scratch/loomyard-measurement.md` with both wall times, the `cmp` result, and the two binaries' paths.

  Acceptance: the branch wall time is at most 5 s and `cmp` reports the files identical.
  If either fails, report the numbers and the first differing bytes; do not change code in this card.
  Re-running is safe: each step overwrites only files under `W/.scratch/`.
- **Commit:** none

## Batch Tests

`verify:` runs the whole `internal/engine` package with the pinned Loomyard clone, as batch 1 does: card 6's tests live in that package, and running them beside the existing suite confirms batch 1's refactor and the new assertions hold together.
Card 7's acceptance is checked by the card itself, since it needs two built binaries and an external checkout no `go test` run can own.
