MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5
reviewed_file: plan/
date: 2026-08-28
```

## Findings

### [BLOCKING:design] LeadingBlocks cannot supply the block text three cards use
**Location:** batch 3 card 17 (helper), consumed by cards 20, 23, 26
**Issue:** `LeadingBlocks(root, src) []*ts.Node` returns only "the first node of each block", yet card 20 says `Header` "strips each block with `StripLineComment(raw, "//")`", card 23 the same with `"#"`, card 26 with `"///"` — `raw` (the joined text of a whole block) is not derivable from a single first node, and no other helper returns it; `CommentBlockAbove` only walks *upward* from a declaration.
**Fix:** state the block-text contract in card 17 — either return per-block `(first *ts.Node, raw string, startLine int)` triples, or `[][]*ts.Node` plus a named joining helper — so the three strategies do not each reinvent the downward sibling walk that card 4's "no new shared helper" rule forbids.

### [BLOCKING:scope] Context omits the file defining an identifier the card names
**Location:** batch 1 card 4; batch 6 card 34
**Issue:** card 4's Requirements assert `errors.Is(err, quarryengine.ErrNoLanguage)` is false but list only `treesitter.go` and `registry_test.go` in `Context:`; card 34 implements `TOCLanguages` as `return registry.ExtensionLanguages()` but lists `registry/registry.go`, not `registry/extension.go`, where card 5 creates that function. Card 3 already lists `errors.go` for the same sentinel, so the omission is also internally inconsistent.
**Fix:** add `internal/quarryengine/errors.go` to card 4's `Context:` and `internal/quarryengine/registry/extension.go` to card 34's.

### [NIT:consistency] Card 52 rests on an edit batch 2 never prescribes
**Location:** batch 8 card 52; batch 2 card 16
**Issue:** card 7 updates layering_test.go's constant-block comment to say "seventh"; card 16 adds `tocPkg` (an eighth path) but never instructs the corresponding count bump, so card 52's premise that "batches 1 and 2 updated both" is false and its "leaving them alone is the correct outcome" framing invites leaving a stale count.
**Fix:** have card 16 own the constant-block comment update alongside the `layeringTable` doc comment, or drop card 52's "already updated" premise.

### [NIT:consistency] Sweep grep misses the site card 49 exists to fix
**Location:** batch 8 card 54
**Issue:** the alternation contains `import all four`, but `internal/quarryengine/doc.go:66` reads "It imports all four packages above" — the pattern does not match, so the one count card 49 is chartered to correct is invisible to the verification aid.
**Fix:** use `imports? all four` (and confirm the other alternatives against their real strings before the card runs).

### [NIT:consistency] Two-phase-flow decision names the wrong batches
**Location:** overview `### Decision: the two-phase read flow is documented in the verb's help text`
**Issue:** `Applies to:` lists facade-and-cli and docs-and-sweep, but batch 6 card 36 explicitly defers the flow to batch 7, and card 44 (doc-sentences-config) is where it lands.
**Fix:** replace facade-and-cli with doc-sentences-config in that decision's `Applies to:` line.

### [NIT:consistency] Python docstring trimming diverges from the shared rule
**Location:** batch 4 card 22; overview `### Decision: docstrings keep the prose and drop the syntax`
**Issue:** the decision says "each stripped line is then trimmed and the lines joined with `\n`, and the whole result is trimmed", but card 22 specifies only `strings.TrimSpace` over the whole `string_content` text, so an indented Python docstring keeps per-line indentation that Go and C# lose.
**Fix:** state in card 22 which of the two behaviours is intended, and align the decision or the card to it.

### [NIT:scope] `path` composition needs a correlation step the cards do not name
**Location:** batch 6 cards 36 and 37
**Issue:** card 36 mandates producing the envelope by re-marshalling the typed struct into `map[string]any`; `DirEntry.Name` is `json:"-"`, so after that marshal the base names are gone and card 37's `filepath.Join(arg, entry.Name)` requires zipping the decoded `files` array back to the original `[]DirEntry` by index — a step neither card states.
**Fix:** say in card 37 that the emitted `files` maps are correlated with `result.Files` positionally before the `path` key is injected.

### [NIT:scope] Block-comment forms are unaddressed for Go and C#
**Location:** batch 2 card 11; batch 3 card 20; batch 4 cards 25, 26
**Issue:** every stripping path is `StripLineComment` with a line prefix, but tree-sitter emits `/* ... */` as a `comment` node too, so a block-form Go header or C# doc comment reaches `header`/`docstring` with its delimiters intact.
**Fix:** state the disposition explicitly — strip `/*`/`*/` in `StripLineComment`, or record block comments as out of scope in the toc package doc and card 47's survey.

## Verdict

REQUEST_CHANGES
One helper contract cannot supply what three strategy cards consume; two Context lists are incomplete.
MILL_REVIEW_END
