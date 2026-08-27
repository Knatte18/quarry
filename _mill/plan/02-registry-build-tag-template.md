# Batch: registry-build-tag-template

```yaml
task: "Improve gopls query precision (build tags + scoping)"
batch: "registry-build-tag-template"
number: 2
cards: 4
verify: go test ./internal/quarryengine/registry/ ./internal/quarryengine/
depends-on: []
```

## Batch Scope

This batch delivers the whole build-tag vocabulary as pure, transport-free functions in `registry` plus the one new typed error in the engine's root leaf package: tag-set normalization, the `Entry.InitializationOptions` field with Go's built-in template, and the template renderer that turns a normalized tag set into the `initializationOptions` map the `initialize` request carries. Nothing here talks to a language server, so every card is TDD-shaped.

The external interface batch 3 consumes is `registry.NormalizeBuildTags` and `registry.RenderInitializationOptions`, plus `quarryengine.ErrBuildTagsUnsupported`. Batch-local decision: the renderer returns `(nil, nil)` for an empty tag set rather than an empty map, because that nil is the single signal every downstream consumer uses to answer "is this query tagged?" (see the overview's `rendered-options-non-nil-means-tagged` Shared Decision).

## Cards

### Card 5: NormalizeBuildTags

- **Context:**
  - `internal/quarryengine/registry/registry.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/registry/buildtags.go`
  - `internal/quarryengine/registry/buildtags_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create `internal/quarryengine/registry/buildtags.go` in package `registry` declaring `func NormalizeBuildTags(tags ...string) []string`. Each argument is split on `,`, every resulting element is trimmed of surrounding whitespace, empty elements are dropped, duplicates are removed, and the remainder is sorted lexicographically. A call whose arguments all normalize away returns nil, not an empty non-nil slice.
  - The variadic shape is what lets both callers share one function without a second entry point: `internal/cli` calls it with the single raw `--build-tags`/`$QUARRY_BUILD_TAGS` string, and the engine calls it with an already-split `[]string` expanded via `...`. Say so in the function's doc comment.
  - The function must be idempotent: `NormalizeBuildTags(NormalizeBuildTags("b,a")...)` equals `NormalizeBuildTags("b,a")`. The engine re-normalizes defensively what the CLI already normalized, so idempotence is a contract, not an incidental property.
  - Write `internal/quarryengine/registry/buildtags_test.go` covering, as a table: `"b,a"` and `"a,b"` normalize identically; `""`, `","` and `" , "` each normalize to nil; `"a,,b, a "` normalizes to `[a b]`; the multi-argument form `NormalizeBuildTags("b", "a")` matches the single-string form; and idempotence over each of those inputs.
- **Commit:** `feat(registry): add NormalizeBuildTags shared by the CLI and the engine`

### Card 6: ErrBuildTagsUnsupported typed error

- **Context:**
  - `internal/quarryengine/doc.go`
- **Edits:**
  - `internal/quarryengine/errors.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add to `internal/quarryengine/errors.go`, following the `ErrResolverUnsupported` pattern already in that file byte for byte in shape: a package-level `var ErrBuildTagsUnsupportedSentinel = errors.New("quarry: build tags unsupported")`, a `type ErrBuildTagsUnsupported struct { Language string }`, its `Error() string` method, and its `Is(target error) bool` method comparing against the sentinel.
  - `Error()` must produce exactly this message shape, with the language name quoted: `quarry: --build-tags is not supported for language "python" (its registry entry's initialization_options has no {{tags}} placeholder)`. The wording names the missing placeholder rather than the missing field, so an operator whose `servers.yaml` overlay whole-replaced the `go:` entry and dropped the placeholder is told what to put back.
  - Document on the type that this error is raised inside the engine's own detect step, before any language server is spawned, and that `internal/cli` maps it to the standard error envelope with exit 1 like every other typed engine error.
- **Commit:** `feat(engine): add ErrBuildTagsUnsupported typed error and sentinel`

### Card 7: Entry.InitializationOptions and Go's built-in template

- **Context:**
  - `internal/quarryengine/registry/load.go`
- **Edits:**
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/registry/registry_test.go`
  - `internal/quarryengine/registry/load_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add the field `InitializationOptions map[string]any` with yaml tag `initialization_options` to `Entry` in `internal/quarryengine/registry/registry.go`, placed after `HasNativeDaemon`. Extend `Entry`'s doc comment to describe it as a per-language, build-tag-only template whose string values may carry the `{{tags}}` placeholder, and to state that it is optional and Go-only in V1, exactly as that comment already describes `PinnedVersion` and `HasNativeDaemon`.
  - Populate it on the `"go"` entry inside `builtins()` as `map[string]any{"buildFlags": []any{"-tags={{tags}}"}}`. Use `[]any` rather than `[]string` for the list so a decoded yaml overlay entry and the built-in entry have the same dynamic shape and the renderer needs only one list case. The other four entries leave the field at its nil zero value.
  - Do not add the field to `validateEntry`'s required checks. It is optional, and an entry without it is valid — that is exactly the case `ErrBuildTagsUnsupported` covers at query time.
  - Extend `internal/quarryengine/registry/registry_test.go` to assert that the built-in `"go"` entry carries a template containing the literal `{{tags}}` and that each of the other four built-in entries has a nil `InitializationOptions`.
  - Extend `internal/quarryengine/registry/load_test.go` to assert that a `servers.yaml` carrying an `initialization_options` block decodes without tripping the decoder's `KnownFields(true)` setting, that an entry omitting the key still validates, and that an overlay `go:` block which whole-replaces the built-in and omits `initialization_options` yields an entry whose `InitializationOptions` is nil — the overlay hazard the hard error exists for.
- **Commit:** `feat(registry): add optional initialization_options entry field with Go's build-tag template`

### Card 8: RenderInitializationOptions

- **Context:**
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/registry/buildtags.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/registry/initoptions.go`
  - `internal/quarryengine/registry/initoptions_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create `internal/quarryengine/registry/initoptions.go` in package `registry` declaring `func RenderInitializationOptions(lang string, entry Entry, tags []string) (map[string]any, error)`.
  - When `len(tags) == 0` it returns `(nil, nil)` unconditionally, whatever `entry.InitializationOptions` holds. This is the back-compat path: the caller sends no `initializationOptions` key at all and the `initialize` request stays byte-identical to today's.
  - When `len(tags) > 0` and no string value anywhere in `entry.InitializationOptions` contains the literal `{{tags}}`, it returns `(nil, &quarryengine.ErrBuildTagsUnsupported{Language: lang})`. An absent template, an empty template, and a non-empty placeholder-free template all take this branch identically — the predicate is the placeholder's absence, never the map's emptiness.
  - When `len(tags) > 0` and the placeholder is present, it deep-copies `entry.InitializationOptions` and replaces every occurrence of `{{tags}}` in every string value with `strings.Join(tags, ",")`, returning the copy. The source entry's own map must never be mutated. Map keys are never substituted. Recurse through nested `map[string]any` and slice values; a decoded yaml map may arrive as `map[string]any` and a decoded yaml list as `[]any`, so handle both `[]any` and `[]string` list element types.
  - No per-tag list expansion: a single comma-joined string replaces the placeholder in place, and a list element containing the placeholder stays one element.
  - Write `internal/quarryengine/registry/initoptions_test.go` covering: a nested map with `{{tags}}` inside a list element renders to the comma-joined value; the source entry map is unchanged afterwards (assert by rendering twice and comparing the entry's own map against a pre-render copy); a map key spelled `{{tags}}` is left alone; a non-empty tag set against a placeholder-free entry returns an error satisfying `errors.Is(err, quarryengine.ErrBuildTagsUnsupportedSentinel)` whose message names the language; a non-empty tag set against an entry with a nil map returns the same error; and an empty tag set against the template-bearing built-in `"go"` entry returns `(nil, nil)` — the back-compat assertion, which should be written first.
- **Commit:** `feat(registry): render initialization_options templates for a normalized build-tag set`

## Batch Tests

`verify:` runs the hermetic tests for the two packages this batch touches: `internal/quarryengine/registry/` covers cards 5, 7 and 8 (normalization, entry decode and validation, and template rendering), and `internal/quarryengine/` covers card 6's typed error alongside the existing layering and seam guards, which must keep passing now that `registry` gains an import of the engine's root leaf package. Every card here is a pure function over in-memory values, so no language server, subprocess, or filesystem fixture is involved.
