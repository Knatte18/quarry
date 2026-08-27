# Batch: doc-sentences-config

```yaml
task: "Add file/dir toc verbs (Tree-sitter-backed)"
batch: "doc-sentences-config"
number: 7
cards: 6
verify: go test ./internal/quarryengine/toc ./internal/cli
depends-on: [6]
```

## Batch Scope

This batch replaces the hard-coded `DocSentences: 1` batch 6 left in `toc file` with the real
four-tier precedence chain: the `--doc-sentences` flag, then `$QUARRY_TOC_CONFIG`, then a
`.quarry.yaml` in the target directory itself, then the built-in default of `1`. It also finishes
`toc file`'s help text with the two-phase discovery flow, which only becomes describable once the
flag exists.

All of it lives in `internal/cli`. The engine already takes a `DocSentences` value and has no opinion
about where it came from; adding a YAML loader to the engine would put configuration parsing behind
the seam for no gain, and would invite reusing the registry loader — which this design explicitly
forbids.

Batch-local decision: the config file's `doc_sentences` value is decoded as a `*string`, not an
`int`. `gopkg.in/yaml.v3` decodes both an unquoted integer scalar and the bare word `all` into a
string field, which is exactly the union the setting needs, and a nil pointer cleanly distinguishes
"the file said nothing" from "the file said 0". This was confirmed against the pinned yaml.v3
version, not assumed.

## Cards

### Card 41: the toc config path resolver

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/cwdcontext.go`
- **Edits:**
  - `internal/cli/paths.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `resolveTOCConfigPath(targetDir string) string`, resolving the toc config
  file path in precedence order: `$QUARRY_TOC_CONFIG` when set, otherwise
  `filepath.Join(targetDir, ".quarry.yaml")`.
  It resolves a path only and never reads the file — an absent file is not an error at this layer,
  mirroring what `resolveConfigPath` already does for the servers.yaml overlay. It returns no error,
  because unlike `resolveConfigPath` it never consults a machine-global directory and so has nothing
  to fail at.
  Its doc comment must state the deliberate limits, since they are the part a later change would
  erode:
  - the lookup happens in `targetDir` **and nowhere else** — no walk up the directory tree, no
    repository-root search, no project detection. toc has no repository-root concept and never detects
    a project the way the LSP verbs do, and an upward search would introduce exactly that concept;
  - `targetDir` is the file's parent directory for `toc file`, so the resolution is per-directory
    rather than per-invocation. `toc dir` never calls this function — it emits headers only, never
    docstrings, so no setting in this file applies to it. The signature is directory-shaped rather
    than file-shaped anyway, because that is the natural shape of a per-directory lookup; the doc
    comment must say plainly that the `toc dir` case is **reserved but currently unused**, so nobody
    reading the signature assumes a caller exists;
  - this file is **not** `servers.yaml`. It has its own name and its own environment variable
    precisely so toc never touches the language-server registry, which it needs no part of.
  Update the file's own header comment, which currently says this file resolves "the two path axes
  internal/cli owns", to name the third.
- **Commit:** `feat(cli): resolve the per-directory toc config path`

### Card 42: the toc config loader

- **Context:**
  - `internal/quarryengine/registry/load.go`
  - `internal/cli/paths.go`
  - `internal/quarryengine/toc/types.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/tocconfig.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `internal/cli/tocconfig.go` with a file header comment stating that it
  holds the optional toc config file's shape and loader, and that it deliberately does not reuse the
  registry's `LoadRegistry` — the two files have nothing in common but their YAML encoding, and
  sharing a loader would couple toc to the language-server registry it exists without.
  Declare the file shape as two structs: an outer type with a single `TOC *tocSection` field tagged
  `yaml:"toc"`, and `tocSection` with a single `DocSentences *string` field tagged
  `yaml:"doc_sentences"`.
  Both are pointers so an absent section and an absent key are distinguishable from a present zero,
  and `DocSentences` is a `*string` rather than a `*int` because the value is a union of an integer
  and the word `all`: yaml.v3 decodes an unquoted integer scalar into a string field as its literal
  text, which is what makes one field serve both forms. Say that in its doc comment, so nobody
  "fixes" the type to `*int` and silently breaks `all`.
  Add `loadTOCConfig(path string) (*string, error)`, returning the raw `doc_sentences` value when the
  file supplied one and `nil` when it did not:
  - a file that does not exist returns `(nil, nil)` — an absent file is not an error, the built-in
    default applies;
  - any other read error is returned wrapped;
  - decode with `yaml.NewDecoder` and `decoder.KnownFields(true)`, exactly as `LoadRegistry` does, so
    a misspelled key is a loud error naming the file rather than a silent no-op;
  - an empty or comments-only file yields `io.EOF` from `Decode` with nothing set, which is a valid
    "no settings" file and must return `(nil, nil)` rather than an error — `LoadRegistry` already
    handles that case the same way, and it is easy to get wrong;
  - any other decode error is returned wrapped with the file path prepended.
  Add `parseDocSentences(raw string) (int, error)`, the one place the value's grammar is defined:
  `all` (case-sensitive, matching the documented form) yields `quarry.TOCAllSentences`; a
  non-negative integer yields itself; a negative integer or any other string is an error whose
  message names the valid forms — a non-negative integer, or `all`. This function is shared by the
  flag and the config file, so both reject the same values with the same message.
- **Commit:** `feat(cli): add the optional toc config file loader`

### Card 43: the `--doc-sentences` flag and precedence chain

- **Context:**
  - `internal/cli/tocconfig.go`
  - `internal/cli/paths.go`
  - `internal/output/output.go`
  - `quarry/facade.go`
  - `internal/cli/cwdcontext.go`
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add a `--doc-sentences` string flag to `toc file` only — `toc dir` emits headers
  and never docstrings, so the setting does not apply there; say so in a comment at the registration
  site rather than leaving the asymmetry unexplained.
  The flag is a **string**, not an int, because `all` is one of its values. Its usage text names both
  forms and the default.
  Add `resolveDocSentences(flagValue string, targetDir string) (int, error)` implementing the whole
  chain, highest precedence first:
  1. a non-empty `flagValue` — parsed through `parseDocSentences`;
  2. the file at `resolveTOCConfigPath(targetDir)`, loaded through `loadTOCConfig`; when it supplied
     a value, parsed through `parseDocSentences`;
  3. the built-in default, `1`.
  Note that tiers 2 and 3 of the design's four-tier chain — `$QUARRY_TOC_CONFIG` and the target
  directory's own `.quarry.yaml` — are both resolved inside `resolveTOCConfigPath`, which is why this
  function reads as three steps rather than four. State that in its doc comment so the two
  descriptions are not read as disagreeing.
  Replace the hard-coded `quarry.TOCOptions{DocSentences: 1}` in `toc file`'s `RunE` with the
  resolved value, and delete the `// TODO` comment batch 6 left there. `targetDir` is the resolved
  file's **parent** directory, computed with `filepath.Dir` after the seam-cwd join.
  Any error from the chain — an invalid flag value, an unknown key in the config file, a malformed
  config file — is an `output.Err` and exit 1, emitted before the engine is called at all.
  Resolution is **per argument, not per invocation**: the setting is per-directory and a batch may
  span directories, so each argument resolves the file tier against its own parent directory. The
  flag tier is hoisted — parse `flagValue` once, before the single-argument branch and the batch
  branch diverge — and a non-empty flag value short-circuits the per-argument file work entirely,
  which is what keeps the common case from re-reading a config file once per argument. An invalid
  flag value therefore fails once, up front, before any argument is processed.
- **Commit:** `feat(cli): add the --doc-sentences flag and its precedence chain`

### Card 44: the two-phase flow in the help text

- **Context:**
  - `internal/cli/cli.go`
- **Edits:**
  - `internal/cli/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** extend `toc file`'s `Long` string with the two-phase discovery flow. Without it no
  agent finds the cheap path, and the verb ships with its most valuable usage undiscoverable — this
  card is the difference between the feature existing and the feature being used.
  Document the flow concretely, with its measured cost on a real 1186-line file holding 40 functions
  and methods:
  1. `quarry toc file --doc-sentences 0 <path>` — the map, 8.4 KB for that file;
  2. read `start`–`sigend` for the few candidates that look relevant, roughly 6 lines each,
     **directly from the source file**;
  3. read `start`–`end` for the one that matched, roughly 16 lines;
  against reading all 1186 lines.
  State the decisive point in the text, not just the numbers: the prose is read **from the source
  file**, never from quarry's rendering of it, so the agent never has to trust quarry's `//` and `///`
  stripping, its C# XML tag removal, or its sentence splitting — it sees the actual bytes.
  `--doc-sentences 0` is therefore the recommended **discovery** mode, not a frugality mode. That
  framing is the point; a help text that presents it as a way to save bytes will be read as optional
  and skipped.
  Also state the `sigend` imprecision here: in a single-line function the signature and the body share
  a line, so `start`–`sigend` includes the body, and no line-based range can separate them.
  Document the config precedence in the same `Long`: the flag, then `$QUARRY_TOC_CONFIG`, then
  `.quarry.yaml` in the target directory, then the default of `1` — including the explicit statement
  that the file is looked up in the target directory alone, with no upward search.
- **Commit:** `docs(cli): document the two-phase toc read flow in the verb help`

### Card 45: config resolution tests

- **Context:**
  - `internal/cli/tocconfig.go`
  - `internal/cli/paths.go`
  - `internal/cli/toc.go`
  - `internal/cli/paths_test.go`
  - `internal/cli/cli_test.go`
  - `quarry/facade.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/tocconfig_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** tests for `resolveTOCConfigPath`, `loadTOCConfig`, `parseDocSentences`, and
  `resolveDocSentences`, over files written into a `t.TempDir()`. Use `t.Setenv` for
  `$QUARRY_TOC_CONFIG` so the variable is restored automatically and the tests stay isolated; do not
  mark those subtests `t.Parallel()`, since `t.Setenv` and parallelism are incompatible.
  Cases:
  - no file, no environment variable, no flag — the resolved value is `1`;
  - a `.quarry.yaml` in the target directory holding `doc_sentences: 3` — the resolved value is `3`;
  - the same file holding `doc_sentences: all` — the resolved value is `quarry.TOCAllSentences`;
  - the same file holding `doc_sentences: 0` — the resolved value is `0`, distinguishable from "the
    file said nothing";
  - `$QUARRY_TOC_CONFIG` pointing at a different file — it wins over a `.quarry.yaml` present in the
    target directory;
  - a flag value of `0` — it wins over both the environment variable and the file;
  - a `.quarry.yaml` in the **parent** of the target directory — it is **ignored**, and the resolved
    value is the default. This is the no-upward-search assertion, and it is the single most important
    case in this file: without it, a later "helpful" change that walks upward passes every other test
    here;
  - a file with an unknown key — a loud error naming the file, from `KnownFields`;
  - a file with an unknown key inside the `toc` section — the same;
  - a file that does not exist — no error, and the default applies;
  - an empty file, and a comments-only file — no error, and the default applies;
  - malformed YAML — an error;
  - `parseDocSentences` over `all`, `0`, `1`, `7`, `-1`, `All`, and `sentence` — the last three are
    errors whose messages name both valid forms.
- **Commit:** `test(cli): cover the toc config precedence and value grammar`

### Card 46: `--doc-sentences` end-to-end tests

- **Context:**
  - `internal/cli/toc.go`
  - `internal/cli/toc_test.go`
  - `internal/cli/cli_test.go`
  - `internal/output/output.go`
- **Edits:**
  - `internal/cli/toc_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** extend the CLI toc tests to drive the flag through the `RunCLIIn` seam against
  fixtures in a `t.TempDir()`, asserting on the decoded JSON rather than on Go values — the omission
  behaviour is only observable there.
  Cases:
  - `--doc-sentences 0` — the `docstring` key is **absent** from every symbol object, not present and
    empty. Assert absence by checking the decoded map, since an empty string and a missing key are
    indistinguishable once decoded into a struct;
  - no flag — each symbol's `docstring` is exactly one sentence;
  - `--doc-sentences all` — the whole docstring;
  - `--doc-sentences 9` against a two-sentence docstring — the whole docstring, exit 0, no error;
  - a fixture whose first sentence contains `e.g.` — under the default, the docstring is not split at
    `e.g.`;
  - a fixture whose docstring contains a backtick-quoted dotted identifier — not split inside the
    backticks;
  - `--doc-sentences -1`, and `--doc-sentences sentence` — `output.Err`, exit 1, and the message names
    both valid forms;
  - `--doc-sentences 0` combined with batch mode — every entry's symbols omit `docstring`, and the
    exit code is unaffected;
  - a `.quarry.yaml` beside the fixture holding `doc_sentences: all`, with no flag — the whole
    docstring, proving the chain reaches the engine and not merely the resolver;
  - `start`, `sigend`, and `end` are byte-identical across `--doc-sentences 0`, no flag, and
    `--doc-sentences all` for the same fixture — the ranges never move with the emitted text, which is
    the property the two-phase flow depends on;
  - `toc dir` has no `--doc-sentences` flag — assert the flag is unregistered there, mirroring how
    `TestInFileFlag_RegisteredOnRefsAndDefinitionOnlyNotSymbol` pins the equivalent asymmetry for the
    existing verbs.
- **Commit:** `test(cli): cover the --doc-sentences flag end to end`

## Batch Tests

`verify: go test ./internal/quarryengine/toc ./internal/cli` covers the package the flag ultimately
drives and the package every change in this batch lands in. `quarry` is unchanged here — the facade
already re-exports `TOCOptions` and `TOCAllSentences` from batch 6 — so it is not re-run.

New test file: `internal/cli/tocconfig_test.go`. Modified: `internal/cli/toc_test.go`.

The parent-directory case in `tocconfig_test.go` is the load-bearing assertion of this batch. Every
other precedence case would still pass if someone added an upward search; only that one fails.
