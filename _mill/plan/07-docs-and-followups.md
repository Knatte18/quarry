# Batch: docs-and-followups

```yaml
task: "Improve gopls query precision (build tags + scoping)"
batch: "docs-and-followups"
number: 7
cards: 4
verify: null
depends-on: [5, 6]
```

## Batch Scope

This batch brings the operator-facing documentation in line with what now ships — the new flags, the new environment variable, the new state-directory keying, and the native-path cost including its batch-mode amplification — corrects one arithmetic error in the document this task cites as its primary evidence, and files the two follow-up issues the design deliberately deferred rather than fixed.

It runs last because every statement it makes is a statement about shipped behaviour, and it is one batch because all four cards are prose or issue text with no code surface. Batch-local decision: the dated findings records under `docs/` are left as they are; only the one arithmetic error is corrected, for the reason card 28 states.

## Cards

### Card 26: README — flags, environment, state keying, native cost

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/paths.go`
  - `internal/quarryengine/daemon/ensureserver.go`
  - `docs/servers.yaml.example`
- **Edits:**
  - `README.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - In the Verbs section, mention `--build-tags` as available on all four verbs and `--no-verify` as `assert-no-callers`-only. Keep the section's one-line-per-verb shape.
  - In the Configuration section, document `--build-tags <a,b>` and its `$QUARRY_BUILD_TAGS` fallback with the same numbered-precedence shape the section already uses for `servers.yaml`. State that the tag set is normalized — split on commas, trimmed, deduplicated, sorted — so two spellings of one set behave identically. State that passing tags for a language whose registry entry carries no build-tag template is a hard error rather than a silent no-op, and call out the consequence an operator will actually hit: `$QUARRY_BUILD_TAGS` exported globally in a shell and then a query run against a Python project fails loudly. Say that this is deliberate, because a silently-ignored precision flag is the failure mode the flag exists to remove.
  - In the State section, document that a non-empty tag set appends a `tags-<hex>` segment to the resolved leaf directory at all three precedence tiers, so each distinct tag set gets its own daemon, socket, state file and lock, and that an empty tag set leaves the resolved path exactly as it is today. Note the accepted cost in one sentence: alternating tagged and untagged queries leaves two gopls daemons resident until each idles out.
  - In the "Windows: native strategy only" section, document that a non-empty tag set makes the native strategy spawn a private gopls rather than joining the shared `-remote=auto` daemon, so a tagged query on that path pays a cold start with no cross-invocation reuse, and that on windows this is the normal path for tagged queries rather than a fallback. Add the batch-mode amplification: `refs`, `definition` and `symbol` make one independent engine call per positional argument, so an N-symbol tagged batch on the native path pays N cold starts within one invocation.
  - Add a short subsection describing what `assert-no-callers` now does by default: it resolves each candidate reference's own definition and keeps only those that resolve back to the queried symbol's declaration, so an interface-method check is precise without a scoping flag. Say that verification is fail-closed — a reference it cannot verify stays a violation — and that `--no-verify` reinstates the older, noisier behaviour.
- **Commit:** `docs(readme): document --build-tags, tag-keyed state, native private spawn, and verification`

### Card 27: servers.yaml.example — the initialization_options template

- **Context:**
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/registry/load.go`
- **Edits:**
  - `docs/servers.yaml.example`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add `initialization_options` to the `go:` block only, at the built-in template's exact value, written in the file's existing compact yaml style. Do not add it to the `python:`, `csharp:`, `typescript:` or `rust:` blocks: the file's convention is to omit a field that is zero for a language, and the other two Go-only optional fields, `pinned_version` and `has_native_daemon`, already appear on `go:` alone.
  - Extend the header comment with a short paragraph explaining that `initialization_options` is a build-tag template rather than a general static-configuration channel: its string values must contain the `{{tags}}` placeholder, which is replaced with the comma-joined tag set, and only Go carries one today.
  - State the overlay hazard the header already warns about generally, made specific here: because a block whole-replaces its built-in counterpart, a `go:` block that edits the entry and drops `initialization_options` loses the template, and a subsequent `--build-tags` query then fails with an error naming the missing placeholder rather than silently ignoring the flag.
- **Commit:** `docs(servers): document the initialization_options build-tag template on the go entry`

### Card 28: correct the scout-multilang headline arithmetic

- **Context:**
  - `docs/scout-agent-usage-findings.md`
  - `docs/scout-vs-grep.md`
- **Edits:**
  - `docs/scout-multilang.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - The "Verdict up front" bullet in `docs/scout-multilang.md` states that 40% of the repo's true call sites to the two most heavily-tested benchmark symbols live behind the build-tag boundary. Its own per-symbol counts later in the same document do not support that figure: 42 of 68 for one symbol and 17 of 38 for the other is 59 of 106 combined, which is 56%, not 40%.
  - Correct that headline to the figure its own counts support, and state the two per-symbol counts inline so the headline is self-checking rather than a number a reader has to go and verify.
  - Change nothing else in that document. It is a dated measurement record, and everything else in it is an accurate account of how the tool behaved when it was written.
  - Do not edit `docs/scout-agent-usage-findings.md` or `docs/scout-vs-grep.md`. Both are dated findings records whose open question this task answers rather than falsifies, and their value is as a record of the behaviour at the time. Only the arithmetic error above is corrected, and only because it sits in the document this task cites as its primary evidence.
- **Commit:** `docs(scout-multilang): correct the build-tag gap headline to match its own counts`

### Card 29: file the two deferred follow-up issues

- **Context:**
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/cli/cli.go`
  - `internal/quarryengine/daemon/ensureserver.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - File one GitHub issue on this repository recording `lsp.Client`'s single-flight transport limitation, so the constraint is written down rather than rediscovered. Name the three specific causes: `Call` increments the unexported `nextID` field with no synchronization, `writeMessage` takes no write lock so two concurrent frames could interleave on the wire, and the response loop selects on one shared channel and silently discards any message whose id is not its own, with no pending-request registry to route a response back to the right caller. State that any change introducing concurrent `Call`s must add that registry and a write mutex first, and that this is why per-reference verification is sequential.
  - File a second GitHub issue recording batch-mode connection reuse. State that `refs`, `definition` and `symbol` make one independent engine call per positional argument, each running a full connection acquire-and-tear-down cycle, so an N-symbol batch pays N cycles on every path — merely wasteful normally, and pathological on the native path with a non-empty tag set, where each cycle is a cold gopls start that indexes the whole module. State that a fix means an engine-level batch entry point serving the supervised path too, not something bolted onto the native fallback.
  - Before each `gh issue create`, run `gh issue list --state all --search <the issue's own title>` against this repository and skip the create if an issue with that exact title already exists, reporting the existing URL instead. This card produces no commit and no file, so a re-run of the batch would otherwise file both issues a second time with nothing in the repository to detect the duplicate.
  - Use `gh issue create` against this repository's own origin remote. Do not reopen issues #1 or #2: both are already closed, and this task is their follow-through.
  - This card changes no file in the repository, so it makes no commit. Report both issue URLs in the batch's completion output so they are traceable.
- **Commit:** none

## Batch Tests

`verify:` is `null` because this batch has no runnable surface: three cards edit prose, and the fourth files GitHub issues and touches no file at all. Correctness here is a review property — every statement must match what batches 1 through 6 actually shipped — and the module-wide `go vet` that runs at the batch boundary is enough to confirm no code was disturbed. The behaviour these documents describe is already pinned by the hermetic tests in batches 2 through 5 and the live tests in batch 6.
