MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic), high reasoning effort
reviewed_file: /home/knatte/Code/quarry/wts/diff-to-symbols/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Interface types have no body-bearing child
**Section:** "The body token stream, and the exact-tier identity test"
**Issue:** `internal/engine/golang.go`'s `goTypeBody` returns the bare `"{"` leaf as an `interface_type`'s body-bearing child (its own doc comment says the grammar exposes "no named body node and no 'body' field at all"), so under the stated definition every interface's body token stream is the single pair `("{","{")` — method-set and embedded-interface changes are invisible to `body`, and two unrelated interfaces (`type Reader interface{…}` vs `type Closer interface{…}`) satisfy the exact-tier identity test and would be **asserted** as a rename in the tier the contract says quarry asserts.
**Fix:** State what the body stream is for a declaration whose body-bearing child is a bare delimiter token, and whether interface members are inside the body stream, the signature stream, or neither.

### [BLOCKING:design] "Signature token stream" has two incompatible readings
**Section:** "The body token stream…" (signature stream) and "What `modified` means"
**Issue:** "the declaration node with its body-bearing child excluded" is unambiguous only when the body child spans to the end of the declaration; for an interface, whose body child is the `"{"` leaf, excluding that one node still leaves every method element in the head stream, while `Symbol.Signature` (`SignatureCut`, `internal/engine/nodes.go`) cuts at `body.StartByte()` and excludes them — so `changed:["signature"]` (verbatim text) and `signature_identical_modulo_name` (token stream) are computed over different spans of the same declaration.
**Fix:** Define the signature stream by byte range relative to the body child rather than by node exclusion, and state that it must agree with `SignatureCut`'s cut point.

### [BLOCKING:design] Untracked working-tree files have no stated disposition
**Section:** "Git plumbing: exactly which calls" / "Entry dispositions…"
**Issue:** With `--to` absent — the working-tree path the Problem section names as Loomyard's card-done binding — `git diff --name-status --no-renames <from>` reports tracked files only, so a file a card created but never `git add`ed contributes no entry at all and its new symbols are silently absent from `created`; the discussion never mentions untracked files, and the `files` echo (which exists precisely so a caller can tell "no symbol changes" from "never read") cannot record what git never listed.
**Fix:** Decide and state whether untracked files are enumerated (e.g. `git ls-files --others --exclude-standard`) or explicitly excluded with the consequence documented.

### [BLOCKING:design] No decided seam produces the per-side clause maps
**Section:** "Layering: pure core, thin git layer" / Technical context
**Issue:** `DeltaGit` is said to use `internal/gitsrc` to build the batch "including the per-directory clause maps", while `internal/gitsrc` is simultaneously specified as holding "no quarry types" and (Technical context) "no tree-sitter" — but a clause can only come from `Strategy.Package(root, src)` inside `treesitter.WithTree`; and the stated working-tree fallback, "reuses the engine's existing on-disk `dirPackage` path", names an unexported method on `*engine.Repo` that package `quarry` cannot call. `UnitsForDir` is the only new exported helper named, and it takes an already-built `map[base]clause`.
**Fix:** Name the exported engine entry points that (a) extract a clause from bytes and (b) build a clause map for an on-disk directory, and say which layer calls each.

### [BLOCKING:scope] The gitignore filter is never applied on the git path
**Section:** Scope / "Git plumbing" / Technical context
**Issue:** `dirPackage`'s doc comment records that a gitignored `.go` file "never votes in the tie-break and never contributes a clause", and `symbolsOfUnit` filters through the same `ignoreSet` so it cannot "contribute spans the walk never listed" — but `git diff` and `git ls-tree` enumerate tracked files, which is a different set (a tracked-but-gitignored `.go` file exists in both), so the delta can emit symbols `toc` never lists and the two sides' clause votes can be taken over different file sets. The discussion applies the analogous `unitSpellable` consistency rule explicitly but never mentions `ignoreSet`.
**Fix:** State whether the git-sourced batch is ignore-filtered, and if so which side's `.gitignore` chain governs.

### [BLOCKING:consistency] "`.` already means the repository root" is false
**Section:** "CLI shape, and the exit-code contract" (rationale)
**Issue:** `internal/cli/cli.go` sets `base = cwd` unless `--root` is given and `runTOC` converts the target with `repopath.RepoRelTarget(root, base, target)`, so `quarry toc .` run from a subdirectory names that subdirectory, not the root; the rationale's premise is wrong, and the discussion never says whether `delta`'s target is put through `RepoRelTarget` before being used as a git pathspec (`git -C <root>` interprets a pathspec against the root, giving two different meanings for the same argument).
**Fix:** Decide whether `delta`'s target goes through `RepoRelTarget` like `toc`'s, and restate what `.` means from a subdirectory.

### [NIT:consistency] "No fuzzy matching … anywhere in the query" vs a similarity float
**Section:** Constraints / "Evidence-tier gating and signals"
**Issue:** The Constraints bullet forbids fuzzy matching "anywhere in the query" while the design emits `body_token_similarity` (a Jaccard coefficient) and sorts by it; `docs/rewrite-plan.md` §9 ("Fuzzy matching of any kind") is cited nowhere against it.
**Fix:** Narrow the constraint's wording to "nothing quarry decides on", and cite §9 in the evidence-tier rationale.

### [NIT:design] An unknown revision lands on exit 3
**Section:** "CLI shape, and the exit-code contract" / Testing (CLI)
**Issue:** `--from bogus` fails inside `git`, and the stated rule ("a git command that failed" → exit 3) reports the most likely user error for this verb as `internal error: ` with git's raw message.
**Fix:** State whether an unresolvable revision is exit 2 with quarry's own sentence, or deliberately exit 3.

## Verdict

REQUEST_CHANGES
Token-stream definition, git enumeration and clause-map layering are underspecified against the source.
MILL_REVIEW_END
