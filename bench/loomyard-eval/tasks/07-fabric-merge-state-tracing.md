# Task 07 — fabric merge-state tracing

Type: exploration
Verb under test: pre-resolved glyph pack (push mode) vs a plain file list vs neither
Status: runnable now

## Setup

Pinned to `72c23d9eecc1fa55add567622093a8bbbfba8c1d` ("Surface merge-in-progress in fabric status"):

```
git -C "$LADDER_LOOMYARD_REPO" worktree add /tmp/loomyard-eval-07 72c23d9eecc1fa55add567622093a8bbbfba8c1d
```

`<TARGET_DIR>` for this task is `/tmp/loomyard-eval-07`. Remove the worktree
when done (`git -C "$LADDER_LOOMYARD_REPO" worktree remove /tmp/loomyard-eval-07`).

## Scope

Four packages, seven files: the weft merge lifecycle -- how the repository
distinguishes a merge its own fabric layer recorded from one that is merely
present in the underlying git checkout because someone ran a merge there by
hand, and what each layer does with that distinction.

## `<TASK TEXT>` (identical for all three arms)

> This repository distinguishes two different kinds of "a merge is in
> progress" on a weft: one the fabric layer itself recorded, and one that is
> merely present in the underlying git checkout because someone ran a merge
> there by hand. Using the symbols listed below, explain how that distinction
> is drawn and enforced, end to end. Your explanation must cover:
>
> 1. Which predicate computes each of the two kinds, what on-disk evidence
>    each one reads, and why the read-only probe cannot be substituted for
>    the guard the mutating verbs consult.
> 2. What a sibling mutating verb does while a fabric-recorded merge is in
>    progress, which typed error carries that refusal, and how that outcome
>    differs from the one produced when only the foreign git merge state is
>    present.
> 3. How the command-line layer surfaces the fabric-recorded state, and
>    which of the two predicates it calls to do so.
> 4. Where the automated conflict resolver sits in this picture: what it is
>    handed, what it does when the merge cannot be finished, and whether it
>    participates in the in-progress bookkeeping at all.

## Output schema (exploration tasks)

This schema was recovered from the V1 benchmark protocol document after that document was deleted.

```json
{
  "relevant_files": ["path/to/file.go", "..."],
  "key_symbols": [
    {"name": "FuncOrTypeName", "file": "path/to/file.go", "role": "one sentence"}
  ],
  "summary": "3-6 sentences explaining how the mechanism works end to end",
  "confidence": "high|medium|low",
  "open_questions": ["anything left uncertain, if any"]
}
```

## Notes for whoever prepares C's fasit / scores this

The chosen subject is the weft merge lifecycle: how the repository tells a
fabric-recorded merge in progress apart from foreign git-level merge state,
spanning `internal/fabriccli` (the CLI verb wiring, `addWeftVerbs`),
`internal/fabricengine` (the lifecycle itself --
`Fabric.MergeInProgress`/`Fabric.MergeContinue`, the guard predicate
`mergeInProgressReason`, the two state probes
`Fabric.mergeRecordExists`/`Fabric.foreignMergeStatePresent`, and the typed
error `ErrMergeInProgress`), `internal/gitrepo` (the low-level probe
`Repo.MergeHeadPresent`) and `internal/mergeresolve` (the automated conflict
resolver's own entry point, `Resolver.Resolve`). The pin is literally the
commit that surfaced merge-in-progress in fabric status, so the mechanism is
present and coherent there; the answer requires holding several predicates
and one typed error together rather than looking one thing up; it spans four
packages and seven files; and it is untouched by tasks 01 through 06, so no
existing fasit overlaps it.

Substitution rule, verbatim, for card 27/28 if a glyph does not resolve
`found` through the facade: do not weaken the gate and do not edit the
engine under test. Replace the offending glyph with another symbol from the
same package and the same mechanism (candidates in reserve:
`internal/fabricengine#Fabric.MergeAbort`,
`internal/fabricengine#Fabric.mergeStateOrForeignErr`,
`internal/gitrepo#Repo.ConflictedFiles`), then, in this order: (1) edit
`pack_targets:` in the ladder file; (2) hand-edit the `Uses:` list in all
three cards, since `ladder pack` rewrites only the pack cell's sentinel
block; (3) re-derive e2's `Files:` list from the new glyph set, deduplicated;
(4) re-run `ladder pack`, which rewrites the pack block and the provenance
record; (5) record the substitution and its reason in this section; (6)
re-run the fasit's cross-check.
