MILL_REVIEW_BEGIN
# Review: Facade + CLI, resolve + expand (T5b) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: "Anthropic Claude Opus-class model; runtime reports 'Opus 5' — self-assessment uncertain beyond 'Claude Opus'"
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:design] RenderExpandText has no catch-all branch
**Location:** batch 1 / card 5
**Issue:** `RenderResolveText` is specified with an explicit `otherwise` third branch, and card 10 insists every mapping spell an unreachable `default`; `RenderExpandText` is given exactly three branches (not-found, found, ambiguous) with no disposition for `Status == ""` or `StatusMultipart`, yet the same card promises the return value "ends with exactly one newline" — an exported renderer any external caller can hand a zero `ExpandAnswer`.
**Fix:** state the fall-through branch's output and add it to the card's test table, matching card 10's unreachable-default posture.

### [BLOCKING:scope] Resolve on a file path target has no machine-independent test
**Location:** batch 1 card 5, batch 3 card 11, batch 4 card 14
**Issue:** card 5 makes a distinctive claim — a path result naming a *file* is rendered with the directory form, no file-vs-directory flag plumbed — but card 11's end-to-end list has only a directory target and a non-existent path, card 14's status table has no path-file row, and the only file-path exercise is `resolve-path.txt` (card 15), which skips wherever `LADDER_LOOMYARD_REPO` is unset.
**Fix:** add a file path target to card 11's end-to-end list (or card 5's text table) so the claim is pinned on every machine.

### [NIT:scope] Expand + ambiguous is unreachable in every fixture the plan builds
**Location:** batch 3 card 12, batch 4 card 14
**Issue:** `codeForExpandAnswer` maps `StatusAmbiguous` to exit 1 and `RenderExpandText` has an ambiguous branch, but card 14's fixture duplicates a *function* under build tags — which the kind gate turns into `NotATypeError`, not ambiguous — and card 12's e2e list omits the case, so the expand-ambiguous path is only ever table-tested.
**Fix:** either add a build-tag-duplicated *type* to card 14's fixture, or state explicitly that expand-ambiguous is table-tested only and why.

### [NIT:consistency] Shared Decision's Applies-to omits the batch its own text binds
**Location:** overview / "the payload's error field is engine text, emitted verbatim"
**Issue:** the decision's Applies-to lists only cli-pipelines and evidence-and-status-gate, yet its body prescribes facade-surface behaviour ("the text view renders that same string as prose after normalisation") and its consequence says "anywhere in this plan".
**Fix:** add facade-surface to the Applies-to line.

### [NIT:consistency] Scratch-tree decision names the wrong directory for quarry
**Location:** overview / "tests build fixtures under .scratch/"
**Issue:** the decision applies to facade-surface and states the helper "writes under `.scratch/cli-tests/`", but `quarry/scratchtree_test.go` writes under `.scratch/quarry-tests/`; only `internal/cli/scratchtree_test.go` uses `cli-tests`.
**Fix:** name both per-package locations, or drop the literal path and say "its own package's subdirectory".

### [NIT:consistency] Plan verify runs none of the plan's new tests
**Location:** overview yaml `verify: go build ./...`
**Issue:** the plan-level verify is a build only, while the plan's own done-gate decision names `go test ./... && golangci-lint run` and the errcheck decision depends on lint actually running; nothing under `verify:` at plan or batch level runs `golangci-lint` or the repo-wide test.
**Fix:** make the plan-level `verify:` the stated done gate, or note explicitly that the hub's `pipeline.done_gate` is the only place it runs.

### [NIT:scope] Card 17 must read five files it is not given
**Location:** batch 4 / card 17
**Issue:** the card requires a "what changed" paragraph "drawn from reading the produced files", but `Context:` lists only `resolve-glyph.txt`, `resolve-method.txt` and `expand-type.txt` — the other five card-16 outputs (`resolve-glyph-text`, `resolve-not-found`, `resolve-path`, `expand-type-text`, `expand-not-a-type`) are absent.
**Fix:** list all eight produced files in card 17's `Context:`.

## Verdict

REQUEST_CHANGES
One renderer branch is unspecified and one design claim is untested off-Loomyard.
MILL_REVIEW_END
