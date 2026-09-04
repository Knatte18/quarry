# Batch: goldens-and-after

```yaml
task: "Facade + CLI, toc (T5a)"
batch: "goldens-and-after"
number: 5
cards: 5
verify: go test ./internal/cli/...
depends-on: [4]
```

## Batch Scope

This batch produces the task's committed evidence and its regression gate as one artifact: the four
`docs/research/output-formats/after/` outputs *are* the golden fixtures the tests compare against,
gated on a Loomyard checkout at `72c23d9` exactly as `internal/engine/loomyard_test.go` already
gates T3's goldens. It also adds the `docs/rewrite-plan.md` §4 byte-for-byte check at the CLI level,
which is what proves the rendering layer added nothing and dropped nothing, and the `after/INDEX.md`
that records the before-to-after mapping.

It is one batch because all five cards share the same environment gate and the same pinned
checkout, and because the goldens are the one part of this task that cannot be written before the
code — a golden can only be produced by running the code, never hand-written.

Batch-local decision: the gate helpers are copied rather than imported. `loomyardRepo`,
`loomyardPin` and the `-update` flag live in `package engine`'s test files and Go does not export
test helpers across packages; card 19 declares this package's own copy, and the copy is the reason
this batch does not depend on any engine change.

## Cards

### Card 19: the Loomyard environment gate for `internal/cli`

- **Context:**
  - `internal/engine/loomyard_test.go`
  - `internal/cli/cli.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/loomyard_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/loomyard_test.go` in `package cli`, mirroring
  `internal/engine/loomyard_test.go`'s structure. Declare
  `var updateGoldens = flag.Bool("update", false, ...)` — this package's own `-update` flag, since
  `flag.Bool` panics on a duplicate name only within one binary and each package's tests build their
  own; declare `const loomyardPin = "72c23d9"`; declare `func loomyardRepo(t *testing.T) string`
  reading `LADDER_LOOMYARD_REPO`, skipping when it is unset or names a path that is not an existing
  directory, skipping when `git -C <repo> rev-parse HEAD` cannot run, and **failing** with
  `t.Fatalf` when git succeeds but `HEAD` does not start with `loomyardPin`.
  The file-level comment records why the asymmetry is deliberate — "this machine has no Loomyard" is
  a normal state every other machine and CI are entitled to be in, while "this machine has the wrong
  Loomyard" is a checkout silently answering for the wrong commit, and a skip there would let this
  task's own done-criterion pass without the byte-for-byte comparison it exists to make ever
  running. It also records that this is a deliberate copy of the engine package's gate rather than a
  shared helper, because Go test helpers are not importable across packages and this task does not
  modify `internal/engine`.
  Read `LADDER_LOOMYARD_REPO` from the environment only — never from a tracked file, because no
  tracked file in this repository may carry a machine-specific path.
- **Commit:** `test(cli): add the Loomyard environment gate for the CLI goldens`

### Card 20: the four `after/` golden cases

- **Context:**
  - `internal/cli/loomyard_test.go`
  - `internal/cli/cli.go`
  - `internal/engine/golden_test.go`
  - `docs/research/output-formats/toc-dir.txt`
  - `docs/research/output-formats/toc-file.txt`
- **Edits:** none
- **Creates:**
  - `internal/cli/after_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/after_test.go` in `package cli`, table-driven over four
  cases, each gated by `loomyardRepo(t)`.

  | golden file | args after the verb |
  |---|---|
  | `toc-dir.txt` | `internal/logger` |
  | `toc-file.txt` | `internal/logger/logger.go` |
  | `toc-dir-text.txt` | `--text internal/logger` |
  | `toc-file-text.txt` | `--text internal/logger/logger.go` |

  Each case calls `Run` in-process — `Run([]string{"toc", "--root", repo, <flags...>, <target>}, &stdout, &stderr)`
  — with the target given **repository-relative**, asserts the returned code is `exitOK`, and asserts
  `stderr` is empty. No `go build`, no `os/exec`, no `t.Chdir`. A repository-relative target is
  correct here precisely because `--root` rebases target interpretation; without that rule the
  target would resolve against the test process's own working directory, land outside the Loomyard
  root, and return `exitNegative` instead of a golden. Note this dependency in the file-level
  comment: the two decisions are load-bearing on each other and must not be changed independently.

  Assemble the file content as the invocation line `$ quarry toc ` followed by the same flags and
  target the case passed, then `"\n\n"`, then `stdout` verbatim. There is **no** exit-code trailer:
  the four before-side files these pair with carry none, and since these files are byte-compared
  goldens the trailer would decide their bytes, so the exit code is asserted by the test instead.
  Compare byte for byte against `docs/research/output-formats/after/<golden file>`, resolved through
  `filepath.Join("..", "..", "docs", "research", "output-formats", "after", name)` since the test's
  working directory is its own package directory. Under `*updateGoldens`, write the assembled bytes
  to that path instead of comparing, creating the `after/` directory with `os.MkdirAll` if needed.
  On mismatch report with a want/got diff in the style `internal/engine/golden_test.go`'s
  `compareGolden` uses.
- **Commit:** `test(cli): add the four after/ golden cases`

### Card 21: generate the four `after/` outputs

- **Context:**
  - `internal/cli/after_test.go`
  - `internal/cli/loomyard_test.go`
  - `internal/engine/golden_test.go`
  - `docs/research/output-formats/toc-dir.txt`
  - `docs/research/output-formats/toc-dir-compact.txt`
- **Edits:** none
- **Creates:**
  - `docs/research/output-formats/after/toc-dir.txt`
  - `docs/research/output-formats/after/toc-file.txt`
  - `docs/research/output-formats/after/toc-dir-text.txt`
  - `docs/research/output-formats/after/toc-file-text.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Produce the four files by running card 20's test under `-update` against a
  Loomyard checkout pinned at `72c23d9` — never by hand. A hand-written golden pins the wrong bytes
  and passes forever, which is exactly the failure `internal/engine/golden_test.go`'s own comment
  warns the next `-update` run against.

  Locate the checkout without recording a machine path anywhere: use `LADDER_LOOMYARD_REPO` when it
  is already set in the environment; otherwise look for a sibling Loomyard container beside this
  repository's own, at the mill container layout's `../../../loomyard/wts/` relative to this
  worktree, and confirm the candidate with `git -C <candidate> rev-parse HEAD` starting with
  `72c23d9`. Then run
  `LADDER_LOOMYARD_REPO=<checkout> go test ./internal/cli/ -run TestAfter -update`.
  If no checkout at that pin can be found, stop and report it — the four files cannot be produced
  without one, and inventing them would defeat their entire purpose. Do not weaken the gate, do not
  change `loomyardPin`, and do not commit placeholder files.

  After generating, re-run the same test **without** `-update` and confirm it passes, so the
  committed bytes are known to compare equal to themselves.
  Verify by reading the generated files that none contains an absolute filesystem path: the
  invocation lines must read `$ quarry toc internal/logger` and the like, and every path inside the
  output must be repository-root relative with forward slashes. The before side violates this in
  `definition-ambiguous.txt`; the after side must not.
- **Commit:** `docs(research): capture the after/ toc outputs as goldens`

### Card 22: the plan §4 byte-for-byte check

- **Context:**
  - `internal/cli/loomyard_test.go`
  - `internal/cli/after_test.go`
  - `internal/cli/cli.go`
  - `internal/engine/testdata/loomyard/render-layout-file.json`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `internal/cli/plan4_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/cli/plan4_test.go` in `package cli`, one gated case asserting
  that `Run([]string{"toc", "--root", repo, "internal/reedengine/render/layout.go"}, ...)` produces
  the JSON of `docs/rewrite-plan.md` §4's file example, modulo the prose §4 abridges with `...`.
  Decode the emitted stdout into a `quarry.DirAnswer` and assert on it, rather than string-matching
  the plan's own hand-formatted example, which is line-wrapped for reading:
  `Dir` is `internal/reedengine/render`, `Package` is `render`, `Language` is `go`; `Files` holds
  exactly one entry whose `Name` is `layout.go`; that entry's `Symbols` is non-nil and its first
  four elements carry, in order, the `ID`, `Kind`, `Start`, `SigEnd` and `End` values §4 lists —
  `internal/reedengine/render#placement` (`type`, 16, 20, 29),
  `internal/reedengine/render#buildStackBody` (`function`, 31, 34, 50),
  `internal/reedengine/render#wrapLayout` (`function`, 52, 54, 56), and
  `internal/reedengine/render#bandHeader` (`function`, 58, 63, 76) — and the exact `Signature`
  strings §4 gives in full for the first three. Do not assert on `bandHeader`'s signature or on any
  `Doc` value §4 abridges.
  Additionally assert on the raw stdout bytes that there is no `"ok"` key, that the indentation is
  two spaces, and that the output ends with exactly one newline.
  The file-level comment records that T3 already pins this answer at the engine level
  (`internal/engine/testdata/loomyard/render-layout-file.json`), and that repeating it at the CLI
  level is what proves the rendering layer added nothing and dropped nothing.
- **Commit:** `test(cli): pin the plan section 4 example at the CLI level`

### Card 23: `after/INDEX.md`

- **Context:**
  - `docs/research/output-formats/INDEX.md`
  - `internal/cli/after_test.go`
  - `docs/research/output-formats/after/toc-dir.txt`
  - `docs/research/output-formats/after/toc-dir-text.txt`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `docs/research/output-formats/after/INDEX.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `docs/research/output-formats/after/INDEX.md`. It states that every file
  in the directory is one real invocation of the rewritten `quarry` CLI against the Loomyard
  checkout at HEAD `72c23d9`, identifying the checkout by name and pin only — **no absolute path**,
  unlike the before side's own `INDEX.md`, which records one.
  It carries a table mapping each before file to its successor:
  `../toc-dir.txt` to `toc-dir.txt`, `../toc-file.txt` to `toc-file.txt`,
  `../toc-dir-compact.txt` and `../toc-file-compact.txt` to **no successor, by design**, and the
  two new text-view files `toc-dir-text.txt` and `toc-file-text.txt` as having no predecessor.
  It records that the compact view was the lossy one-sentence-per-file view whose precision loss
  (0.96 to 0.82) is `docs/rewrite-plan.md` §1's lesson 4, and that the lossless text view replaces
  it rather than continuing it.
  It records what changed between the two sides, reading the generated files rather than asserting
  from memory: key order is now the answer struct's declaration order rather than alphabetical (a
  map was marshalled somewhere in V1, which is why the before side reads `dirs`, `files`, `ok` and
  puts `generated` before `header`); `ok: true`, `test: false`, `generated: false` and the empty
  `dirs: []` are gone, because shared facts are stated once and defaults never; the repeated
  directory prefix on every file path is gone, replaced by a bare `name` under the directory
  answer's own `dir`; and the verb is now `quarry toc <target>` rather than the V1
  `quarry toc dir|file <target>` split.
  It records that these four files are also the golden fixtures `internal/cli/after_test.go`
  compares against, so the committed evidence and the regression gate cannot disagree, and that
  regenerating them is `go test ./internal/cli/ -run TestAfter -update` with
  `LADDER_LOOMYARD_REPO` pointing at a checkout at that pin.
  It states that each `.txt` is exactly the invocation line, a blank line, and the output verbatim,
  with no exit-code trailer, and that all four commands exit 0 — asserted by the test rather than by
  a line in the file. Do not repeat the before side's `INDEX.md` claim that "the exit code is at the
  bottom of each file": that claim is untrue of the four before-side `toc` files.
- **Commit:** `docs(research): add after/INDEX.md mapping the before side to the after side`

## Batch Tests

`verify: go test ./internal/cli/...` runs every test in the package: batches 3 and 4's Loomyard-free
suite plus this batch's three gated files. On a machine with no Loomyard checkout the gated cases
skip with a reason and the rest still run, which is the intended asymmetry — a machine at the
**wrong** pin fails loudly instead. Card 21 is the one card that genuinely requires the checkout;
its own requirements say to stop and report rather than invent fixtures if none is present.
The scope stays `./internal/cli/...` because that is the only Go package this batch touches; the
`docs/research/output-formats/after/` files are data the tests in that package read.
