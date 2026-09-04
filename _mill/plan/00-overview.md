# Plan: Facade + CLI, resolve + expand (T5b)

```yaml
task: "Facade + CLI, resolve + expand (T5b)"
slug: "facade-cli-resolve-expand"
approved: true
started: "20260904-085953"
parent: "main"
root: ""
verify: go build ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: facade-surface
    file: 01-facade-surface.md
    depends-on: []
    verify: go test ./quarry/
  - number: 2
    name: cli-argument-layer
    file: 02-cli-argument-layer.md
    depends-on: [1]
    verify: go test ./internal/cli/
  - number: 3
    name: cli-pipelines
    file: 03-cli-pipelines.md
    depends-on: [2]
    verify: go test ./internal/cli/
  - number: 4
    name: evidence-and-status-gate
    file: 04-evidence-and-status-gate.md
    depends-on: [3]
    verify: go test ./internal/cli/
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: internal/engine is never modified

- **Decision:** no file under `internal/engine/` is edited, created, deleted, or moved by any card in this plan. Both verbs are already complete in the engine; this task ships only the surface above it. The same applies to `glyph/` — it is read and imported, never changed.
- **Rationale:** the task's scope states it outright. An engine change would mean the merged T4 shipped incomplete, and it would put the same behaviour rule in two layers.
- **Applies to:** all batches

### Decision: a negative answer renders the payload, never the error envelope

- **Decision:** when the facade returns a value with a nil error, the CLI renders that value with the selected renderer and exits with the code the exit-code table below assigns — including `status: not_found`, `status: ambiguous`, and a result carrying a pre-resolution `error` string. The `{"ok":false,"error":"..."}` envelope, written through `fail`, is used only where there is no payload at all.
- **Rationale:** the engine deliberately moved these dispositions into the payload. Collapsing them into the envelope would destroy the `unit`, `candidates` and `reason` fields that are the whole reason a validator asks. The absence of an `ok` key is not a claim of success; `status` is the discriminator.
- **Applies to:** all batches

### Decision: the exit-code table both new verbs are written from

- **Decision:** the four existing exit codes keep their current meanings and the two verbs map onto them as follows. The rows are checked in this order and the final row is the catch-all: a mapping function must test the named conditions first and fall through to 3, never the reverse.

  | verb | condition | code | stdout |
  |---|---|---|---|
  | `resolve` | `status: found` | 0 | the result payload |
  | `resolve` | `status: multipart` | 0 | the result payload |
  | `resolve` | `status: not_found` (glyph or path) | 1 | the result payload |
  | `resolve` | `status: ambiguous` | 1 | the result payload |
  | `resolve` | pre-resolution rejection: `error` set, `status` absent | 1 | the result payload |
  | `expand` | `status: found` | 0 | the answer payload |
  | `expand` | `status: not_found` | 1 | the answer payload |
  | `expand` | `status: ambiguous` | 1 | the answer payload |
  | `expand` | a not-a-type failure | 1 | the error envelope |
  | both | unparseable flag, wrong target count, unknown verb, a toc-only flag on a new verb, a `--root` that is not a directory, no repository root discovered | 2 | the error envelope, plus the usage text on stderr |
  | `resolve` | the path arithmetic itself erroring | 1 | the error envelope, message `target outside repository: <target>` |
  | `expand` | a target containing no `#` | 2 | the error envelope, plus the usage text on stderr |
  | `expand` | a grammar rejection of a target that does contain `#` | 1 | the error envelope, message `expand <target>: <reason>` |
  | `expand` | the missing-head-span invariant failure | 3 | the error envelope |
  | both | any remaining error: engine read failure, wrong result count, render failure, stdout write failure | 3 | the error envelope |

- **Rationale:** exit 2 keeps its existing meaning, "the caller asked wrong about the CLI"; a well-formed invocation naming an unspellable glyph is not that, because the CLI ran it to a definite conclusion and has a payload with a reason word to show for it. Exit 1 keeps its existing meaning, "the invocation was well formed and ran to a definite, negative conclusion", and `ambiguous` is negative in that sense: nothing was chosen.
- **Applies to:** all batches

### Decision: the payload's error field is engine text, emitted verbatim

- **Decision:** the existing rule that forbids leaking the engine's package-name prefix through an exit-1 or exit-2 message binds the sentences quarry itself authors — the failure path's message, and therefore both the error envelope on stdout and the same sentence on stderr. It does not bind a result payload's own error field, which is a data field of the answer, populated by the engine, and carried to stdout unchanged. So a resolve of a path that escapes the root prints the engine's own doubled-prefix string inside its payload and exits 1, and the text view renders that same string as prose after normalisation. The end-to-end tests pin that exact string; no evidence golden covers it, because no golden invocation escapes the pinned checkout's root.
- **Rationale:** the alternative is for the command line to overwrite a payload field the engine authored, which would be a second implementation of the outside-repository disposition — the very thing routing that case through the engine avoids. The rule's purpose is that quarry's own prose not name an internal package; a data field echoing the producer's text is a different thing, and rewriting it would make the command line's payload disagree with the facade's, which returns the engine value untouched.
- **Consequence for the implementer:** the doubled prefix in that string is the engine's own wording and is a known defect handed to the operator as a follow-up, not something any card here tightens. Do not rewrite, trim, or re-prefix a payload error field anywhere in this plan.
- **Applies to:** facade-surface, cli-pipelines, evidence-and-status-gate

### Decision: no new file is added to quarry/

- **Decision:** the two JSON renderers and the shared unexported encoder go into `quarry/render.go`; the two text renderers go into `quarry/text.go` beside the grammar helpers they reuse. The aliases and constants go into `quarry/quarry.go`, the two methods into `quarry/repo.go`.
- **Rationale:** splitting a renderer away from the encoder configuration or the grammar it shares would put the two halves of one byte contract in two files.
- **Applies to:** facade-surface

### Decision: one implementation of each byte contract

- **Decision:** the three exported JSON renderers share one unexported encoder helper. The symbol line has one implementation, extended in place, rather than a second copy for the new verbs. The CLI's path arithmetic is split into two named functions rather than gaining a mode flag.
- **Rationale:** three functions that each configure an encoder identically are three places the byte contract can drift; one is one. Extending the existing symbol-line writer in place is what keeps every committed `toc` golden byte-identical without regeneration, which is this task's own proof that the shared-helper refactors changed nothing.
- **Applies to:** facade-surface, cli-argument-layer

### Decision: the CLI classifies a target with a "#" containment test and nothing else

- **Decision:** the CLI tells a glyph target from a path target with `strings.Contains(target, "#")`. A glyph target is handed to the facade verbatim — no path arithmetic, no rebasing, no stat. A path target for `resolve` goes through the arithmetic half of the path helper only, and a form that escapes the root is passed to the engine as a leading-`..` relative path so the engine's own outside-repo rule produces the answer.
- **Rationale:** a glyph's unit is repository-relative by the grammar's own definition, so cwd arithmetic on it would corrupt it. Letting the engine own the outside-repo disposition avoids a second implementation of that rule in the CLI.
- **Applies to:** cli-argument-layer, cli-pipelines

### Decision: errcheck is on, so every write's error is handled or explicitly discarded

- **Decision:** `.golangci.*` is absent from the repository root, so `golangci-lint run` uses its defaults and the `errcheck` default is on. Every new `io.WriteString` and `Write` call either handles its error or discards it with an explicit `_, _ =`, matching what `internal/cli/cli.go` already does.
- **Rationale:** the done gate is `go test ./... && golangci-lint run`; an unchecked write error fails it.
- **Applies to:** all batches

### Decision: tests build fixtures under .scratch/, never in a system temp directory

- **Decision:** every new test that needs a tree on disk uses its own package's existing `writeScratchTree` helper, which writes under that package's own subdirectory of `.scratch/` — `.scratch/quarry-tests/` for the facade package and `.scratch/cli-tests/` for the command-line package. The two helpers are a deliberate per-package copy, because Go test helpers are not importable across packages; neither is made shared. No test calls `t.TempDir()`, and no test changes the process working directory.
- **Rationale:** the system temp directory is banned for this repository's tests and `.scratch/` is the sanctioned location; the helper already registers its own cleanup.
- **Applies to:** facade-surface, cli-pipelines, evidence-and-status-gate

### Decision: no tracked file carries a machine-specific path

- **Decision:** the Loomyard checkout the evidence goldens are produced against is named only by the `LADDER_LOOMYARD_REPO` environment variable, sourced from the gitignored `.scratch/ladder.env`. Every golden target is repository-relative, and each golden's recorded invocation line is spelled literally per row so `--root` cannot leak into a committed file.
- **Rationale:** the task body and the repository's handoff notes both forbid a machine path in a tracked file, and the existing golden table already spells its invocation lines literally for exactly this reason.
- **Applies to:** evidence-and-status-gate

### Decision: the done gate is the repo-wide test and lint pair

- **Decision:** `go test ./... && golangci-lint run`, already configured as this hub's `pipeline.done_gate`. It was run against this worktree's tip before planning and exits 0, so it carries no pre-existing debt into this task.
- **Rationale:** the batch verify scopes are per-package, so a repo-wide gate is what catches a cross-package regression from the shared-helper edits.
- **Where each gate actually runs:** the per-batch `verify:` commands run one package's test binary after every implementer and fixer round. The plan-level `verify:` is `go build ./...`, run at each batch boundary, and is deliberately a compile check only — it is what catches a cross-package break at the batch that introduced it, without paying the repository-wide suite's cost on every round. Neither of those runs the linter or the repository-wide test: those run once, at the end, as the hub's configured done gate. An implementer who wants to check the errcheck rule before then runs `golangci-lint run` by hand; nothing in the per-round loop will catch an unchecked write error for them.
- **Applies to:** all batches

## All Files Touched

_Full union of every `Creates:` / `Edits:` / `Moves:` **target** path across every batch, sorted alphabetically (Move **source** paths are excluded — they disappear, like `Deletes:` tokens).
Cards are the source of truth;
this section is the input `_plan_validate.py`'s `all-files-touched-mismatch` check cross-references against the derived union of every card's `Edits:`/`Creates:`/Move-target paths, to catch drift between the hand/agent-maintained list here and that derived union._

- `docs/research/output-formats/after/INDEX.md`
- `docs/research/output-formats/after/expand-not-a-type.txt`
- `docs/research/output-formats/after/expand-type-text.txt`
- `docs/research/output-formats/after/expand-type.txt`
- `docs/research/output-formats/after/resolve-glyph-text.txt`
- `docs/research/output-formats/after/resolve-glyph.txt`
- `docs/research/output-formats/after/resolve-method.txt`
- `docs/research/output-formats/after/resolve-not-found.txt`
- `docs/research/output-formats/after/resolve-path.txt`
- `internal/cli/after_test.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/doc.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/glyph5_test.go`
- `internal/cli/target.go`
- `internal/cli/target_test.go`
- `internal/cli/usage.go`
- `quarry/doc.go`
- `quarry/quarry.go`
- `quarry/render.go`
- `quarry/render_test.go`
- `quarry/repo.go`
- `quarry/repo_test.go`
- `quarry/text.go`
- `quarry/text_test.go`
