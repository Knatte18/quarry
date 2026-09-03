MILL_REVIEW_BEGIN
# Review: Engine core (T3)

```yaml
duration_s: 306.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/engine-core/_mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Generic receiver owner keeps its type parameters
**Section:** D5 / D3 / "Helpers to reuse rather than rewrite"
**Issue:** `docs/glyph.md` §3 line 95 states "Type parameters are not part of a glyph: `Box[T]` is `Box`", but the owner is read by `golang.go:goReceiverTypeName`, which returns the receiver type node's verbatim text (`NodeText(typeNode)`, or the pointee's for a pointer receiver) — for `func (b *Box[T]) M()` that is `Box[T]`, so `ID` becomes `unit#Box[T].M`, which `glyph.Parse` rejects with `ReasonMemberTypeParams` (`glyph/golang.go:100`) and Testing 15's parse assertion fails. The type *name* is safe (`spec.ChildByFieldName("name")` is bare); only the method owner is not, and no decision covers it.
**Fix:** State in D5 that a generic receiver's owner is the bare type name with the type-argument list stripped, and add a fixture case (method on `Box[T]`, pointer and value receiver) to Testing 4/5.

### [BLOCKING:design] `SpansOf`'s both-directories collision has no carrier
**Section:** D16
**Issue:** The stated signature is `func (r *Repo) SpansOf(g glyph.Glyph) ([]Symbol, error)`, yet the same decision says that when both `U/` and `U_test`'s stripped directory exist "the collision is recorded on the result for T4 to report as `ambiguous`" — neither return value nor `Symbol` has a field for it, so Testing 11's "collision recorded" assertion cannot be written as specified.
**Fix:** Name the mechanism (a result struct, a second return value, or a typed sentinel error) or state that T3 returns only the union and the collision is T4's to rediscover.

### [NIT:decision] Blank-identifier declarations have no disposition
**Demoted-from:** BLOCKING
**Section:** D5 / D6
**Issue:** D5's rule "one symbol per declared *name*" for file-scope `const`/`var` mechanically emits `_` for the `var _ = …` / `var _ Iface = (*T)(nil)` shape, giving an id `unit#_` that parses but names nothing addressable, and many such declarations in one package collapse to one glyph the way `init` does; the case is live in-tree at `internal/quarryengine/cgoguard_nocgo.go:20` (`internal/cgoguard/` after D2), which Testing 14's self-round-trip walks, and glyph.md is silent on it.
**Fix:** Decide whether the walk skips blank-identifier declarations or lists them under `unit#_`, and if listed say it inherits D6's set-equality treatment.

### [NIT:design] Where mutable test fixtures live is unstated
**Section:** Testing 11g / 11b, Constraints
**Issue:** Fixtures are described as "small Go trees committed in-repo", but 11g rewrites a fixture's `.gitignore` between two `TOC` calls and 11b needs a symlink cycle; a committed tree cannot be rewritten without dirtying the worktree, and the Constraints ban `/tmp` and any system temp directory, which is what `t.TempDir()` uses.
**Fix:** Say whether these two cases build their tree at runtime (and where) or are committed with the mutation applied to a copy.

### [NIT:design] `symbols,omitempty` conflates "not requested" with "none found"
**Section:** D12
**Issue:** Today's `FileTOC.Symbols` is `json:"symbols"` and deliberately non-nil so the key is `[]` rather than absent (`types.go:63-66`); D12 makes it `omitempty`, so a file queried with `symbols` on but holding no declaration is byte-identical to a directory query, and §4's "defaults never" list names `test`, `generated` and `dirs`, not `symbols`.
**Fix:** State the intended reading of an absent `symbols` key, or exempt it from `omitempty` as it is today.

### [NIT:scope] `depth` on a file target is undefined
**Section:** D13
**Issue:** A file target is defined as a one-entry directory answer with no `dirs`, but nothing says what a non-zero `Depth` does with a file target — ignored, or an error.
**Fix:** One sentence: `Depth` is ignored for a file target.

## Verdict

REQUEST_CHANGES
Generic receiver owners, the collision carrier, and blank identifiers need decisions before planning.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
