# Batch: docs-inventory

```yaml
task: 'The glyph-maker: declaration to glyph (P1, roadmap 2b)'
batch: 'docs-inventory'
number: 5
cards: 4
verify: go build ./... && go vet ./...
depends-on: [3]
```

## Batch Scope

This batch closes the prose inventory: every in-tree statement a fourth verb or a payload shape
carrying no status word falsifies, in the files no earlier batch already edits. It is one batch
because it is one predicate applied four times, and because a reviewer checking the inventory for
completeness wants to read every disposition in one place. It depends on batch 3 so the prose lands
after the surface it describes exists.

Batch-local decision: prose sites inside files an earlier batch already edits stay with that batch,
committed together with the code they describe — the file header in the JSON renderer file, the flag
parser's own doc comment, the help text's layout comment, the exit-code mappers' comments, and the
update flag's description. Splitting a file's comment from the code in the same file across two
batches would leave the file half-true for the length of the DAG.

The predicate this batch applies is: *any statement a new verb or a new envelope shape falsifies,
counted or not.* It is deliberately wider than a count-based search, which is structurally blind to a
sentence like "a negative answer is a payload with a status word" — true of the two resolution verbs,
false of the maker, and carrying no number to find.

## Cards

### Card 17: the facade's own surface counts

- **Context:**
  - `quarry/render.go`
  - `quarry/text.go`
  - `quarry/name.go`
  - `quarry/quarry.go`
- **Edits:**
  - `quarry/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `quarry/doc.go`, four sentences:

  `the package exposes three query methods, not one: TOC, Resolve and Expand` becomes four, naming
  `Name`. The sentence's rhetorical point — "not one" — survives; only the count and the list change.
  Note in passing that `Name` is a package-level function rather than a method, so the phrase "query
  methods" needs rewording to cover both shapes.

  `The package owns seven renderers` becomes nine. The sentence goes on to enumerate them in three
  groups; the two new renderers join the JSON-success group and the text group respectively.

  `the three text renderers, RenderText, RenderResolveText and RenderExpandText` becomes four; and
  `The three JSON success renderers ... share one encoder configuration` becomes four. The second is
  load-bearing rather than cosmetic: it is the sentence stating that the byte contract cannot drift,
  and the new JSON renderer is inside that guarantee because it delegates to the same unexported
  encoder.

  The paragraph beginning `The failure envelope's "ok" key` says a negative outcome `is a payload
  with a status word, rendered by the ordinary renderer, not the failure envelope`. The maker's
  rejection is a payload rendered by the ordinary renderer that carries no status word at all — only
  an error and a reason. Extend the sentence to cover that shape explicitly. This is the sentence the
  no-`ok`-key decision rests on, so leaving it half-true would undercut that decision's own citation.
- **Commit:** `docs(quarry): four queries, nine renderers, and a payload with no status word`

### Card 18: the command's own package documentation

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/flags.go`
  - `internal/cli/usage.go`
- **Edits:**
  - `internal/cli/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/cli/doc.go`, three paragraphs changed and one left alone:

  `The command has three verbs` becomes four, and the per-verb sentences gain the maker's: it takes a
  declaration head, which is neither a path nor a glyph.

  The paragraph stating that a target is handed to the facade verbatim `whenever the verb takes a
  glyph` is true of the new verb for the same reason — no path arithmetic, no stat — but its stated
  condition excludes it. Widen the condition from "whenever the verb takes a glyph" to "whenever the
  verb does not take a path". The paragraph's closing sentence, that "toc" is the only verb that
  still takes a path, stays true as written and needs no edit; leave it, so the next reader does not
  change it twice.

  The negative-answer paragraph says such an answer is `a payload carrying a status word (or, for the
  pre-resolution case, an error field of its own)`. The maker's negative payload carries an error and
  a reason and no status at all, which the parenthetical anticipates in shape but attributes only to
  the resolve verb's pre-resolution rejection; name the maker there explicitly. This paragraph also
  states what the `ok` key does and does not mean, which is the rule the no-`ok` decision rests on,
  so extending it keeps that rule's own statement true.

  The classification paragraph, which says the glyph parser is the single classifier and that no
  surface tests a target for the separator, stays true and needs no edit: the new verb adds no
  classification of any kind — its target is a fragment handed to the extractor, never tested for a
  separator. This is a disposition, recorded because a reader auditing the file will ask.
- **Commit:** `docs(cli): four verbs, the widened verbatim-target rule, and the statusless payload`

### Card 19: the engine's own count and its stale pointers

- **Context:**
  - `internal/engine/name.go`
  - `docs/rewrite-plan.md`
- **Edits:**
  - `internal/engine/expand.go`
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `internal/engine/expand.go`, two sites:

  The sentence `one symbol entry is what all three verbs return for a symbol` keeps its count of
  three but must be reworded: the maker returns no symbol entry at all — its answer carries an id and
  a kind, not a symbol — so the claim is still true of exactly three verbs. Reword it to name *which*
  three, "the three verbs that return symbols", rather than leaving "all three verbs" to read as
  "every verb", which stops being true.

  The two citations of `docs/rewrite-plan.md's three-queries section` in this file: repoint each to
  the section by number and title, section 5, The queries, rather than by its query count, which this
  task makes four. A pointer that names a count goes stale the moment the count changes; one that
  names the section does not.

  Edit `internal/engine/answer.go`, one site: the same three-queries-section citation, repointed the
  same way.

  Change nothing else in either file. Both are product code and this card touches comments only.
- **Commit:** `docs(engine): name the three symbol-returning verbs and repoint the plan citations`

### Card 20: the repository's front door and the rewrite plan

- **Context:**
  - `internal/cli/usage.go`
  - `quarry/name.go`
  - `internal/engine/name.go`
- **Edits:**
  - `README.md`
  - `docs/rewrite-plan.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Edit `README.md`: the opening sentence names three queries and lists them; it becomes four, with
  the maker's verb added to the list in the same backticked style.

  Edit `docs/rewrite-plan.md`, two sites:

  Section 5, The queries, gains one paragraph for the new verb, in the style of the resolve and
  expand paragraphs beside it: the contract (unit plus declaration head in, glyph id and kind out),
  the batch-versus-CLI split, the same-extractor property that makes prediction and extraction one
  function, and the two non-goals — interface methods and multi-name const or var specs. Place it
  after the toc paragraph so the three existing paragraphs keep their order.

  The opening framing sentence at the top of section 1 reads "three queries over one tree-sitter
  parse". This is a phrasing decision rather than a number swap: the new verb is a query over a
  *supplied fragment*, not over the tree, so incrementing the count alone would quietly falsify the
  original claim. Keep "three queries over one tree-sitter parse" intact and add the maker as a
  fourth query over the same extractor.

  Change nothing else in either file. The performance paragraph, the phase-2 parking note, and the
  no-reference-query paragraph are all unaffected.
- **Commit:** `docs: a fourth query in the README and the rewrite plan`

## Batch Tests

`verify: go build ./... && go vet ./...` is the right gate for this batch: every card changes Go doc
comments or Markdown prose, and no card touches a statement any test asserts — the help text, the
flag messages and the verb-gate test name all live in files batch 3 already edited and committed,
together with the assertions that pin them. Build and vet confirm the comment edits left every file
compiling and well-formed, which is the whole of what is mechanically checkable here.

The prose itself is checked by review against the three inventory passes the batch scope names:
search the verb names together, search the number words with the renderer names beside them, and read
the doc comment of every package this task touches asking whether each sentence survives a fourth
verb whose payload carries an error and a reason and no status. The repo-wide `pipeline.done_gate`
runs the full suite and the linter before the task is marked done.
