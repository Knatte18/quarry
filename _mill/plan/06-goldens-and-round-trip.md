# Batch: goldens-and-round-trip

```yaml
task: "Engine core (T3)"
batch: "goldens-and-round-trip"
number: 6
cards: 6
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go vet ./... && CGO_ENABLED=1 go test ./internal/...
depends-on: [5]
```

## Batch Scope

This batch is the task's done-criterion made runnable: the byte-for-byte goldens against the real
repository plan §4's examples were taken from (D17), and the round trip that asserts, per glyph,
that what `toc` listed and what `SpansOf` returns are the same set of spans (D6, D16). It adds no
production rule; if a golden or a round trip fails, the defect is in an earlier batch and is fixed
there.

It also closes out the module hygiene the task's gate names: `go vet` clean and `go mod tidy`
leaving no diff, which drops the two unused indirect grammar requirements.

## Cards

### Card 35: The Loomyard environment gate

- **Context:**
  - `internal/engine/repo.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `internal/engine/loomyard_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** New test-only file holding one helper,
  `loomyardRepo(t *testing.T) string`. It reads the checkout path from the `LADDER_LOOMYARD_REPO`
  environment variable — never from a tracked file, because no tracked file may carry a machine
  path — and:

  - **skips** when the variable is unset or names a path that is not an existing directory;
  - runs `exec.Command("git", "-C", <repo>, "rev-parse", "HEAD")` and **fails** when the output's
    prefix is not `72c23d9`, the commit the plan's examples were taken at;
  - **skips**, with the reason, when the `git` binary is missing or the command errors — that is "no
    usable checkout", not a mismatch.

  The comment must say why the pin is read through `git` rather than by reading `.git/HEAD`: the
  checkout is a git *worktree*, whose `.git` is a file pointing into a worktrees directory, so the
  naive read gets the wrong ref or none. It must also say why the skip-versus-fail split is not
  symmetric: "this machine has no Loomyard" is normal, while "this machine has the wrong Loomyard"
  would let the task's own done-criterion pass without ever being checked. Note that the per-machine
  path is kept in a gitignored `.scratch/ladder.env`, recreated per machine, and is not this
  repository's to track.

  Also declare the `-update` flag this batch's goldens use, as a package-level
  `flag.Bool("update", false, ...)`, so exactly one file owns it.
- **Commit:** `test(engine): add the Loomyard checkout gate and pin check`

### Card 36: The directory and file goldens

- **Context:**
  - `internal/engine/loomyard_test.go`
  - `internal/engine/toc.go`
  - `internal/engine/repo.go`
  - `internal/engine/answer.go`
  - `docs/rewrite-plan.md`
- **Edits:** none
- **Creates:**
  - `internal/engine/golden_test.go`
  - `internal/engine/testdata/loomyard/render-dir.json`
  - `internal/engine/testdata/loomyard/render-layout-file.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:** `golden_test.go` runs two cases through `Open` on the Loomyard checkout and
  `TOC`, marshals the answer to indented JSON, and compares it byte for byte against the committed
  golden: the render directory as a directory target with default options, and that directory's
  layout source file as a file target, whose one entry therefore carries `symbols`. Under the
  `-update` flag each case rewrites its own golden instead of comparing.

  The goldens are the plan's own examples with its prose elisions filled in from the real source:
  "byte for byte, apart from prose" means the structure and the key set are exact and the prose is
  whatever the file actually says. Generate them once with `-update` and read the diff against the
  plan's examples by hand before committing — a golden generated from a wrong walk is a wrong golden,
  and nothing downstream would catch it. Assert in the review of that diff that the directory answer
  carries `dir`, `package`, `language`, `doc` and its files and no other key; that no file entry
  carries `test: false`, `generated: false` or an empty `dirs`; that the file entry's symbols carry
  `id`, `kind`, `start`, `sigend`, `end`, `signature` and `doc` and no `file`; and that the type
  symbol the plan names has the start, sigend and end the plan names.
- **Commit:** `test(engine): pin the Loomyard directory and file goldens`

### Card 37: The depth-zero subdirectory golden

- **Context:**
  - `internal/engine/loomyard_test.go`
  - `internal/engine/toc.go`
  - `internal/engine/answer.go`
  - `docs/rewrite-plan.md`
- **Edits:**
  - `internal/engine/golden_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add the third case: the parent of the render directory as a directory target at
  depth zero. It asserts on the answer directly rather than against a committed golden, because the
  point is a shape rather than prose — the subdirectory entry in `dirs` carries `dir`, `package` and
  `doc` and **no other key**, which is what the plan's second example shows. Assert it on the
  marshalled JSON of that one entry so an accidentally-populated `files` or `language` fails.
- **Commit:** `test(engine): assert the depth-zero subdirectory shape`

### Card 38: The round trip over quarry itself

- **Context:**
  - `internal/engine/toc.go`
  - `internal/engine/resolve.go`
  - `internal/engine/repo.go`
  - `internal/engine/answer.go`
  - `internal/engine/walk.go`
  - `glyph/glyph.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/roundtrip_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** The headline criterion, over this repository, with no environment needed so it
  always runs. Resolve the module root from `runtime.Caller(0)`, `Open` it, and call `TOC` on the
  root with `DepthAll` and symbols on. Collect every listed symbol, **group the listed glyphs by
  unit**, and for each unit call `symbolsOfUnit` **once**; then assert, per glyph, that the set of
  `(File, Start, SigEnd, End)` tuples the lookup returned equals the set the walk listed. Zero
  misses, zero extras.

  Grouping by unit is required rather than an optimisation, and the comment must say so: a per-glyph
  lookup re-parses the whole unit directory for every glyph in it, nothing is cached, and the
  Loomyard run in the next card would land in the minutes and inside reach of the default test
  timeout. Set equality rather than a one-to-one match is likewise required: several `func init()` in
  one package are one glyph with several declarations, and so are build-tag duplicates, so a
  one-to-one check is unsatisfiable by construction.
- **Commit:** `test(engine): round-trip every glyph quarry lists against its spans`

### Card 39: The round trip over Loomyard

- **Context:**
  - `internal/engine/loomyard_test.go`
  - `internal/engine/roundtrip_test.go`
  - `internal/engine/toc.go`
  - `internal/engine/resolve.go`
  - `internal/engine/answer.go`
  - `glyph/glyph.go`
  - `glyph/parse.go`
- **Edits:**
  - `internal/engine/roundtrip_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** The same assertion over the whole Loomyard checkout, gated on the environment
  helper and skipped under `-short`. Factor the assertion the previous card wrote into a helper both
  cases call, rather than copying it. This case additionally asserts that **every** listed symbol's
  `ID` round-trips through `glyph.Parse` and back through `String` unchanged, which is what "every
  declaration `toc` lists has a glyph" means operationally — and which is the assertion that catches
  an id the grammar would reject.
- **Commit:** `test(engine): round-trip Loomyard's glyphs and assert every id parses`

### Card 40: Module hygiene

- **Context:**
  - `internal/engine/doc.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Run `go mod tidy` and commit the result. The two indirect grammar requirements
  for languages this module does not use are unused and should drop out; if `go mod tidy` keeps
  them, do not hand-edit the file — report it instead, because a requirement tidy keeps is one
  something still imports. Then run `go vet ./...` and fix anything it reports in code this task
  wrote. Do not add, upgrade or downgrade any requirement: the engine's only dependency is
  tree-sitter and its Go grammar, and this task introduces none.
- **Commit:** `chore(deps): tidy the module and drop the unused grammar requirements`

## Batch Tests

`verify:` adds `go vet ./...` to the build-and-test pair the earlier batches use, because card 40's
own subject is vet-cleanliness and a batch whose verify does not run it cannot check it.

New test files: `loomyard_test.go`, `golden_test.go` and `roundtrip_test.go`, plus the two committed
goldens under `internal/engine/testdata/loomyard/`.

Two of the three Loomyard-dependent cases skip cleanly on a machine with no checkout, which is the
normal case, and fail loudly on a checkout at the wrong commit, which is the case that would
otherwise make the done-criterion unverifiable. The quarry round trip has no environment dependency
and always runs, so this batch's headline assertion is checked on every machine.

`go mod tidy` leaving no diff is not expressible as a Go test; it is part of the task's gate and is
checked by card 40 running it and committing whatever it produces, after which a second run leaves
the tree clean.
