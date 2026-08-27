# Behavioural equivalence: quarry vs `lyx scout`

This document records the batch-5 comparison that proves the ported `quarry`
binary answers exactly like `lyx scout` did, for a query set resolved fresh
against this task's own worktree at the batch-5 commit — not against the
stale, never-committed positions in the multi-language research document
(see `docs/scout-multilang.md`'s benchmark:
one package it queries was since renamed, two symbols moved five to ten
lines, and the ground-truth file it was graded against was never
committed). Feeding those stale positions to both binaries returns a
matching `ErrSymbolNotFound` envelope from each side, which compares equal
while proving nothing — the rule this document's methodology exists to
avoid.

## Method

- **Binaries:** `quarry` built at
  `/home/knatte/Code/quarry/wts/quarry/.scratch/quarry` (commit `2c54254`,
  batch 5 card 34) and `lyx` built at `.scratch/lyx` in this worktree
  (commit `95f041b3`, the plan-edit commit immediately preceding this
  document).
- **Target directory:** both binaries were pointed at this task worktree,
  `/home/knatte/Code/loomyard/wts/scout-extract-standalone-repo`, via
  `--target-dir` — the same absolute path on both sides, so no legitimate
  absolute-path difference is possible in this comparison (see "Permitted
  path differences" below).
- **Language server:** both binaries resolve their own toolchain-managed
  `gopls` (pinned `v0.23.0`) independently — `quarry` via its own
  `resolveGoToolchain`, `lyx scout` via `internal/scoutengine`'s copy of
  the same logic — so this comparison also exercises that each side's
  toolchain manager converges on the identical pinned server.
- **State isolation:** `lyx scout` anchors its supervised daemon relative
  to the target directory (this worktree is outside a lyx hub, so
  `AnchorRoot` falls back to the target directory itself, under the
  gitignored `.lyx/scout/go/`); `quarry` was pointed at an explicit
  `--state-dir` under `/tmp`. Different state directories are expected and
  irrelevant to the comparison — daemon state paths are never part of a
  lookup envelope.
- **Query resolution:** for every named-symbol query below, the position
  actually queried by `refs`, `definition`, and `assert-no-callers` was
  first resolved with `lyx scout symbol <name> --target-dir <this
  worktree>` against the current tree at this commit, then re-expressed as
  an explicit `file:line:col` position — this sidesteps `workspace/symbol`
  fuzzy-matching a bare method/function name against many unrelated
  candidates (several of the names below match 20-100+ fuzzy candidates
  across this module's own code and its vendored dependencies), which
  would otherwise make `refs`/`definition` report `"ambiguous"` (exit 2)
  instead of exercising the actual reference/definition lookup. The
  `symbol` verb itself is still tested with the bare name, unresolved,
  since `symbol` never has an ambiguous state to route around.

## Query set

| # | Symbol | Category | Resolved position |
|---|---|---|---|
| 1 | `Err` | high-fan-in plain function (215 references) | `internal/output/output.go:27:6` |
| 2 | `CurrentSHA` | method with many call sites (30 references, `(*gitrepo.Repo).CurrentSHA`) | `internal/gitrepo/gitrepo.go:72:16` |
| 3 | `CurrentBranch` | method (`(*gitrepo.Repo).CurrentBranch`) | `internal/gitrepo/gitrepo.go:237:16` |
| 4 | `ReadJSON` | generic function (`func ReadJSON[T any](...)`) | `internal/state/state.go:59:6` |
| 5 | `WriteJSON` | generic function (`func WriteJSON[T any](...)`) | `internal/state/state.go:28:6` |
| 6 | `ShedProducer.Call` | interface method (100 references across every producer implementation) | `internal/shedengine/producer.go:31:2` |
| 7 | `sortedLanguages` | plain function, single unambiguous `workspace/symbol` match (control case) | `internal/scoutengine/detect.go:75:6` |
| 8 | `markerExists` | plain function, `assert-no-callers` pass case (`--except internal/scoutengine/detect.go`) | `internal/scoutengine/detect.go:68:6` |
| 9 | `QuarryNoSuchSymbolXYZ123` | not-found / error path — **the deliberate exception**: a matching `ErrSymbolNotFound` envelope on both sides *is* the assertion here, not a proof-of-nothing false pass | n/a (never resolves) |
| 10 | `ReadJSON` + `WriteJSON` | batch mode (2+ positional args), `symbol` verb | as #4/#5 |

## Per-query per-verb verdict

Every row below is `ok:true` with a non-empty result on the `lyx scout`
side (verified first, per this batch's rule) before being compared to
`quarry`'s envelope. All 27 comparisons matched **byte for byte**,
including the exit code.

| Query | Verb | `lyx scout` exit | `quarry` exit | Envelope | Exit code |
|---|---|---|---|---|---|
| `Err` | `symbol` | 0 | 0 | MATCH | MATCH |
| `Err` | `refs` (215 refs) | 0 | 0 | MATCH | MATCH |
| `Err` | `definition` | 0 | 0 | MATCH | MATCH |
| `Err` | `assert-no-callers` | 1 (violation) | 1 (violation) | MATCH | MATCH |
| `CurrentSHA` | `symbol` | 0 | 0 | MATCH | MATCH |
| `CurrentSHA` | `refs` (30 refs) | 0 | 0 | MATCH | MATCH |
| `CurrentSHA` | `definition` | 0 | 0 | MATCH | MATCH |
| `CurrentSHA` | `assert-no-callers` | 1 (violation) | 1 (violation) | MATCH | MATCH |
| `CurrentBranch` | `symbol` | 0 | 0 | MATCH | MATCH |
| `CurrentBranch` | `refs` (4 refs) | 0 | 0 | MATCH | MATCH |
| `ReadJSON` | `symbol` | 0 | 0 | MATCH | MATCH |
| `ReadJSON` | `refs` (22 refs) | 0 | 0 | MATCH | MATCH |
| `ReadJSON` | `definition` | 0 | 0 | MATCH | MATCH |
| `WriteJSON` | `symbol` | 0 | 0 | MATCH | MATCH |
| `WriteJSON` | `refs` (22 refs) | 0 | 0 | MATCH | MATCH |
| `ShedProducer.Call` | `symbol` | 0 | 0 | MATCH | MATCH |
| `ShedProducer.Call` | `refs` (100 refs) | 0 | 0 | MATCH | MATCH |
| `ShedProducer.Call` | `definition` | 0 | 0 | MATCH | MATCH |
| `sortedLanguages` | `symbol` (1 match) | 0 | 0 | MATCH | MATCH |
| `sortedLanguages` | `refs` (2 refs) | 0 | 0 | MATCH | MATCH |
| `sortedLanguages` | `definition` | 0 | 0 | MATCH | MATCH |
| `sortedLanguages` | `assert-no-callers` | 1 (violation) | 1 (violation) | MATCH | MATCH |
| `markerExists` | `assert-no-callers --except detect.go` | 0 (pass, `"callers":[]`) | 0 (pass, `"callers":[]`) | MATCH | MATCH |
| `QuarryNoSuchSymbolXYZ123` | `symbol` | 1 (`ErrSymbolNotFound`) | 1 (`ErrSymbolNotFound`) | MATCH — the deliberate exception | MATCH |
| `QuarryNoSuchSymbolXYZ123` | `refs` | 1 (`ErrSymbolNotFound`) | 1 (`ErrSymbolNotFound`) | MATCH — the deliberate exception | MATCH |
| `QuarryNoSuchSymbolXYZ123` | `definition` | 1 (`ErrSymbolNotFound`) | 1 (`ErrSymbolNotFound`) | MATCH — the deliberate exception | MATCH |
| `ReadJSON`,`WriteJSON` (batch) | `symbol` | 0 | 0 | MATCH | MATCH |

Every error-message string compared equal verbatim, including the
`"quarry: "` prefix quarry's own errors now carry (e.g.
`{"error":"quarry: symbol \"QuarryNoSuchSymbolXYZ123\" not found
under <target-dir>","ok":false}` from both binaries) — the prefixes were
deliberately left unrenamed through the port so this exact comparison
could be strict (see the plan's "behavioural equivalence is the acceptance
criterion" Shared Decision); card 36 files the follow-up issue to rename
them now that this proof exists.

## Permitted path differences

None. Both binaries were invoked with the identical `--target-dir`
(this worktree's own absolute path), so every file path in every envelope
is byte-identical by construction — there is no legitimate path difference
to enumerate for this run.

## Verdict

**The port is proven behaviourally equivalent to `lyx scout` for this
query set.** All 27 query/verb comparisons — spanning a high-fan-in plain
function, a method with many call sites, two generic functions, an
interface method whose references span every implementing producer, a
single-match control case, both `assert-no-callers` outcomes (violation
and pass, with exit-code comparison), the not-found error path, and batch
mode — matched byte for byte on both the JSON envelope and the process
exit code. Batch 6's deletion of `internal/scoutengine` and
`internal/scoutcli` is authorized to proceed on the strength of this
result, once card 37 confirms the live tier (card 34) is also green.
