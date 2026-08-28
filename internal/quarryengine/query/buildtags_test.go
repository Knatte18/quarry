// buildtags_test.go is the untagged, spawn-free counterpart to any future
// buildtags_lsp_test.go: it covers the build-tag hard error detectAndRender raises (refs.go) and
// the empty-tag-set no-op, exercised against a synthetic registry.Registry built in-test rather
// than the built-in one, mirroring definition_test.go's style — a nonexistent server binary makes
// every assertion here observable with no real language server and no //go:build tag.

package query

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
)

// buildTagFixtureDir creates a target directory containing only a go.mod marker file, so
// registry.DetectLanguage succeeds against it without any server ever being consulted.
func buildTagFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/buildtagfixture\n"), 0o644); err != nil {
		t.Fatalf("write go.mod marker: %v", err)
	}
	return dir
}

// templateFreeRegistry returns a synthetic registry.Registry whose sole "go" entry carries no
// InitializationOptions template (the nil zero value) and points Command at a binary name that
// cannot exist on $PATH, so any call that gets past the build-tag check fails with
// quarryengine.ErrServerNotFoundSentinel rather than actually spawning anything.
func templateFreeRegistry() registry.Registry {
	return registry.Registry{
		"go": {
			Markers:     []string{"go.mod"},
			Match:       "any",
			Command:     []string{"quarry-nonexistent-binary-buildtags-xyz"},
			InstallHint: "this binary is intentionally fake for the test",
		},
	}
}

// buildTagOptions returns the Options common to every test in this file, varying only Query and
// BuildTags per call site.
func buildTagOptions(dir string, buildTags []string, query Query) Options {
	return Options{
		Registry:  templateFreeRegistry(),
		TargetDir: dir,
		Lang:      "go",
		Query:     query,
		Timeout:   5 * time.Second,
		BuildTags: buildTags,
	}
}

// TestBuildTags_HardErrorAcrossAllThreeEntryPoints asserts that References, Definition, and
// Symbol all raise quarryengine.ErrBuildTagsUnsupportedSentinel — sharing detectAndRender's one
// hard-error step — when BuildTags is non-empty and the detected entry's InitializationOptions has
// no {{tags}} placeholder to render it into.
func TestBuildTags_HardErrorAcrossAllThreeEntryPoints(t *testing.T) {
	dir := buildTagFixtureDir(t)

	tests := []struct {
		name string
		call func(opts Options) error
	}{
		{
			name: "References",
			call: func(opts Options) error {
				_, err := References(t.Context(), opts)
				return err
			},
		},
		{
			name: "Definition",
			call: func(opts Options) error {
				_, err := Definition(t.Context(), opts)
				return err
			},
		},
		{
			name: "Symbol",
			call: func(opts Options) error {
				_, err := Symbol(t.Context(), opts)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := buildTagOptions(dir, []string{"integration"}, Query{Symbol: "Resolve"})
			err := tt.call(opts)
			if !errors.Is(err, quarryengine.ErrBuildTagsUnsupportedSentinel) {
				t.Fatalf("%s() err = %v; want errors.Is(err, ErrBuildTagsUnsupportedSentinel)", tt.name, err)
			}
			if !strings.Contains(err.Error(), "go") {
				t.Errorf("%s() err message = %q; want it to name the language %q", tt.name, err.Error(), "go")
			}
		})
	}
}

// TestBuildTags_HardErrorPrecedesConnectionAttempt asserts the build-tag error is raised before
// any connection attempt: the synthetic entry's Command names a binary that cannot exist on
// $PATH, so a returned quarryengine.ErrServerNotFoundSentinel (rather than the build-tag error)
// would prove the check ran too late — after acquireConnection had already tried to spawn.
func TestBuildTags_HardErrorPrecedesConnectionAttempt(t *testing.T) {
	dir := buildTagFixtureDir(t)
	opts := buildTagOptions(dir, []string{"integration"}, Query{Symbol: "Resolve"})

	_, err := References(t.Context(), opts)
	if !errors.Is(err, quarryengine.ErrBuildTagsUnsupportedSentinel) {
		t.Fatalf("References() err = %v; want errors.Is(err, ErrBuildTagsUnsupportedSentinel)", err)
	}
	if errors.Is(err, quarryengine.ErrServerNotFoundSentinel) {
		t.Errorf("References() err = %v; want it to NOT also satisfy errors.Is(err, ErrServerNotFoundSentinel) — the build-tag check must short-circuit before acquireConnection ever runs", err)
	}
}

// TestBuildTags_EmptySetIsANoOp asserts that every input which normalizes to the empty tag set —
// nil, a slice of one empty string, and a slice of one whitespace-and-comma-only string — takes
// detectAndRender's back-compat branch rather than the build-tag error: each call instead reaches
// (and fails at) the fake binary lookup, which is the correct signal that it got past the
// build-tag check.
func TestBuildTags_EmptySetIsANoOp(t *testing.T) {
	dir := buildTagFixtureDir(t)

	tests := []struct {
		name      string
		buildTags []string
	}{
		{name: "Nil", buildTags: nil},
		{name: "SingleEmptyString", buildTags: []string{""}},
		{name: "WhitespaceAndCommaOnly", buildTags: []string{" , "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := buildTagOptions(dir, tt.buildTags, Query{Symbol: "Resolve"})
			_, err := References(t.Context(), opts)
			if errors.Is(err, quarryengine.ErrBuildTagsUnsupportedSentinel) {
				t.Fatalf("References() with BuildTags = %#v err = %v; want it to NOT satisfy errors.Is(err, ErrBuildTagsUnsupportedSentinel) — an empty normalized tag set is a no-op", tt.buildTags, err)
			}
			if !errors.Is(err, quarryengine.ErrServerNotFoundSentinel) {
				t.Errorf("References() with BuildTags = %#v err = %v; want errors.Is(err, ErrServerNotFoundSentinel) (proving it got past the build-tag check)", tt.buildTags, err)
			}
		})
	}
}

// TestBuildTags_DefensiveNormalizationMakesOrderIrrelevant asserts that BuildTags supplied in
// different orders ({"b","a"} vs {"a","b"}) produce the identical build-tag error message,
// proving detectAndRender's defensive re-normalization (registry.NormalizeBuildTags) actually
// runs rather than trusting an already-sorted caller.
func TestBuildTags_DefensiveNormalizationMakesOrderIrrelevant(t *testing.T) {
	dir := buildTagFixtureDir(t)

	_, errBA := References(t.Context(), buildTagOptions(dir, []string{"b", "a"}, Query{Symbol: "Resolve"}))
	_, errAB := References(t.Context(), buildTagOptions(dir, []string{"a", "b"}, Query{Symbol: "Resolve"}))

	if !errors.Is(errBA, quarryengine.ErrBuildTagsUnsupportedSentinel) || !errors.Is(errAB, quarryengine.ErrBuildTagsUnsupportedSentinel) {
		t.Fatalf("References() errs = %v, %v; want both errors.Is(err, ErrBuildTagsUnsupportedSentinel)", errBA, errAB)
	}
	if errBA.Error() != errAB.Error() {
		t.Errorf("References() error messages differ by BuildTags order: %q (b,a) vs %q (a,b); want identical messages", errBA.Error(), errAB.Error())
	}
}
