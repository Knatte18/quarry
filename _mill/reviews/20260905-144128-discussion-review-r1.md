# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Named-leaf-only token stream drops operators and keywords
**Section:** "The body token stream, and the exact-tier identity test"
**Issue:** The stream is defined as the pairs of "every *named, leaf* tree-sitter node under
the declaration's body-bearing child" — but in tree-sitter, operators and keywords are
*anonymous* nodes (`+`, `-`, `++`, `--`, `return`, `:=` are grammar string literals, not named
rules), so they contribute nothing to the stream. Two consequences, both correctness failures
in asserted answers: (a) a body change from `x + y` to `x - y` (or `x++` to `x--`) produces an
identical stream, so the symbol is **not** reported `modified` — the delta says "nothing
changed" for a real semantic change, which breaks exactly the card done-check the Problem
section names as a consumer; (b) two symbols differing only in such tokens satisfy "identical
modulo the renamed identifier" and can be **asserted** as an exact-tier rename that is false —
the one tier the contract says quarry asserts.
**Suggested fix:** Define the stream over *all* leaf nodes (anonymous included), keeping the
`(kind, text)` pair shape — the substitution rule already keys on `identifier` nodes carrying
the two names, so it is unaffected; add `x++`→`x--` (not modified today, must be) and
`+`→`-` (must demote exact to evidence) to the TDD cases.

### [NIT:design] Signature comparison "modulo the name" is textual and substring-hazardous
**Section:** "Exact-tier rename scope" condition 5 / evidence signal `signature_identical_modulo_name`
**Issue:** The body test's substitution rule is node-based (`identifier` nodes only), but the
signature is verbatim text, and "identical modulo the renamed identifier, under the same
substitution rule" has no nodes to key on — a textual substitution of `Run`→`Execute` also
hits the `Runner` receiver in `func (r *Runner) Run() error`, yielding a false mismatch (or
with naive replace, a false match).
**Suggested fix:** State the mechanism: compare the signatures' own token streams under the
same node-based rule (the declaration head is part of the same parse), or word-boundary-bounded
substitution — one sentence in the decision either way.

### [NIT:consistency] Constraint pins a goldens path a parallel task is relocating
**Section:** Constraints — "committed goldens under `docs/research/output-formats/after/` ... must stay byte-identical"
**Issue:** A parallel task (`goldens-move`) is moving exactly those files to
`internal/cli/testdata/`; if it merges first, this constraint and the Testing section's
`compareAfterGolden` reference name a location that no longer exists, inviting a false
"constraint violated" reading at merge time.
**Suggested fix:** Phrase the constraint as "existing committed goldens, wherever they live at
merge time, stay byte-identical" and note the in-flight move.

## Verdict

REQUEST_CHANGES
Exceptionally thorough and verified throughout, but the named-leaf-only stream definition makes asserted answers wrong on operator-only changes.
