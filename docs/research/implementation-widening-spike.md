# Implementation-widening measurement spike

This document records the output of
`internal/quarryengine/query/implementation_spike_lsp_test.go`'s
`TestImplementationWidening_Spike`, run once against the pinned `gopls`
binary, and the widening-mode decision derived from that measurement per
the plan's decision rule (batch 1, card 4).

## Measurement

- **gopls version:** `golang.org/x/tools/gopls v0.23.0` (`$HOME/.cache/quarry/tools/go/v0.23.0/gopls version`)
- **Fixture:** `testdata/clockfixture` (self-contained module; packages `builder`, `runner`, `sched`, each declaring a structurally identical, otherwise unrelated `clock` interface — see batch 1 card 2)
- **Command:** `PATH="$HOME/.cache/quarry/tools/go/v0.23.0:$PATH" go test -tags lsp -run TestImplementationWidening_Spike -v ./internal/quarryengine/query/`

### Position (a): interface-method `clock.Now` in `builder/poll.go`

`textDocument/definition` (1 location):

```
testdata/clockfixture/builder/poll.go:13:2
```

`textDocument/implementation` (5 locations):

```
testdata/clockfixture/builder/poll.go:21:18
testdata/clockfixture/runner/tick.go:12:2
testdata/clockfixture/runner/tick.go:20:18
testdata/clockfixture/sched/wait.go:12:2
testdata/clockfixture/sched/wait.go:20:18
```

`textDocument/references` (7 locations, `includeDeclaration: true`):

```
testdata/clockfixture/builder/poll.go:13:2
testdata/clockfixture/builder/poll.go:33:11
testdata/clockfixture/builder/poll.go:41:11
testdata/clockfixture/runner/tick.go:32:11
testdata/clockfixture/runner/tick.go:39:11
testdata/clockfixture/sched/wait.go:32:11
testdata/clockfixture/sched/wait.go:39:11
```

Per-reference `textDocument/definition` (one call per reference above, logged alongside the reference it resolves):

```
testdata/clockfixture/builder/poll.go:13:2 -> testdata/clockfixture/builder/poll.go:13:2   (interface clock.Now decl)
testdata/clockfixture/builder/poll.go:33:11 -> testdata/clockfixture/builder/poll.go:13:2  (interface clock.Now decl)
testdata/clockfixture/builder/poll.go:41:11 -> testdata/clockfixture/builder/poll.go:21:18 (concrete realClock.Now)
testdata/clockfixture/runner/tick.go:32:11 -> testdata/clockfixture/runner/tick.go:12:2    (runner's own interface clock.Now decl)
testdata/clockfixture/runner/tick.go:39:11 -> testdata/clockfixture/runner/tick.go:20:18   (runner's own concrete realClock.Now)
testdata/clockfixture/sched/wait.go:32:11 -> testdata/clockfixture/sched/wait.go:12:2      (sched's own interface clock.Now decl)
testdata/clockfixture/sched/wait.go:39:11 -> testdata/clockfixture/sched/wait.go:20:18     (sched's own concrete realClock.Now)
```

### Position (b): concrete method `realClock.Now` in `builder/poll.go`

`textDocument/definition` (1 location):

```
testdata/clockfixture/builder/poll.go:21:18
```

`textDocument/implementation` (3 locations):

```
testdata/clockfixture/builder/poll.go:13:2
testdata/clockfixture/runner/tick.go:12:2
testdata/clockfixture/sched/wait.go:12:2
```

`textDocument/references` (5 locations, `includeDeclaration: true`):

```
testdata/clockfixture/builder/poll.go:21:18
testdata/clockfixture/builder/poll.go:33:11
testdata/clockfixture/builder/poll.go:41:11
testdata/clockfixture/runner/tick.go:32:11
testdata/clockfixture/sched/wait.go:32:11
```

## Decision

`textDocument/implementation` at the interface-method position returned not
only `builder`'s own concrete satisfier (`poll.go:21:18`), but also the
structurally identical, unrelated satisfiers declared independently in
`runner` and `sched` (`tick.go:12:2`, `tick.go:20:18`, `wait.go:12:2`,
`wait.go:20:18`). Per the decision rule this fires the directional branch.

`mode: directional`

## Counts (interface-method position, `builder/poll.go:13:2`)

`references-unfiltered:` 7

The seven raw `textDocument/references` locations listed above.

`references-verified:` 3

Applying `declarationMatchSet`'s package-scoped filter by hand: the
declaration match set is the position-level `textDocument/definition`
result (`poll.go:13:2`, directory `builder`) unioned with only the
`textDocument/implementation` locations whose own directory is also
`builder` (`poll.go:21:18` — `tick.go:12:2`, `tick.go:20:18`, `wait.go:12:2`,
and `wait.go:20:18` are excluded, their directories being `runner` and
`sched`). Of the seven references' own logged per-reference definition
results, three land on a location in that two-location set
(`poll.go:13:2 -> poll.go:13:2`, `poll.go:33:11 -> poll.go:13:2`, and
`poll.go:41:11 -> poll.go:21:18`); the remaining four resolve to `runner`'s
or `sched`'s own interface or concrete declaration, neither of which is in
the match set, so all four are dropped.

`callers-verified:` 2

`references-verified` (3) minus the declaration site the position-level
`textDocument/definition` result names (`poll.go:13:2`, which is present in
the reference list only because `includeDeclaration: true` puts it there):
`3 - 1 = 2`. This is the figure batch 6's live test asserts against —
`assert-no-callers`'s reported `callers` list excludes every returned
declaration by construction, so its count is `references-verified` minus
the declaration count, not the raw `references-unfiltered` figure.

This measurement supersedes issue #1's pre-fix "31 → 2" figure, which was
taken against a different repo entirely; it is not carried forward here.
