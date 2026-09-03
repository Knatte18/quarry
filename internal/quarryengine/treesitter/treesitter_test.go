// treesitter_test.go covers grammar loading for every wired language and the parser/tree release
// behaviour WithTree guarantees on every route out of the function.

package treesitter

import (
	"errors"
	"strings"
	"testing"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// TestWithTree_ParsesEachWiredLanguage is the canary for a grammar-module version bump that
// renames or removes an API: each case parses a trivial valid source and asserts the resolved
// root node's kind matches the language's documented root production.
func TestWithTree_ParsesEachWiredLanguage(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		src      string
		wantKind string
	}{
		{"Go", "go", "package main\n", "source_file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotKind string
			var gotPartial bool
			var sawRoot bool
			err := WithTree(tt.lang, []byte(tt.src), func(root *ts.Node, partial bool) error {
				sawRoot = root != nil
				gotKind = root.Kind()
				gotPartial = partial
				return nil
			})
			if err != nil {
				t.Fatalf("WithTree(%q, ...) returned error: %v", tt.lang, err)
			}
			if !sawRoot {
				t.Fatalf("WithTree(%q, ...) handed the callback a nil root node", tt.lang)
			}
			if gotPartial {
				t.Errorf("WithTree(%q, ...) partial = true; want false for valid source", tt.lang)
			}
			if gotKind != tt.wantKind {
				t.Errorf("WithTree(%q, ...) root.Kind() = %q; want %q", tt.lang, gotKind, tt.wantKind)
			}
		})
	}
}

// TestWithTree_ReleasesParserAndTreeOnEveryRoute uses the onRelease test seam to assert the
// release counter advances exactly once on each of the three routes out of WithTree: success,
// the route where the callback itself returns an error, and the route where the parse is only
// partial. It also asserts the callback's own error is returned unchanged on the middle route.
func TestWithTree_ReleasesParserAndTreeOnEveryRoute(t *testing.T) {
	t.Cleanup(func() { onRelease = nil })

	t.Run("Success", func(t *testing.T) {
		releases := 0
		onRelease = func() { releases++ }
		t.Cleanup(func() { onRelease = nil })

		err := WithTree("go", []byte("package main\n"), func(root *ts.Node, partial bool) error {
			return nil
		})
		if err != nil {
			t.Fatalf("WithTree returned error: %v", err)
		}
		if releases != 1 {
			t.Errorf("release count = %d; want 1", releases)
		}
	})

	t.Run("CallbackError", func(t *testing.T) {
		releases := 0
		onRelease = func() { releases++ }
		t.Cleanup(func() { onRelease = nil })

		wantErr := errors.New("callback failed")
		err := WithTree("go", []byte("package main\n"), func(root *ts.Node, partial bool) error {
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("WithTree returned %v; want %v unchanged", err, wantErr)
		}
		if releases != 1 {
			t.Errorf("release count = %d; want 1", releases)
		}
	})

	t.Run("Partial", func(t *testing.T) {
		releases := 0
		onRelease = func() { releases++ }
		t.Cleanup(func() { onRelease = nil })

		var gotPartial bool
		// A deliberately broken fixture: an unclosed brace forces tree-sitter's error recovery,
		// so the resulting tree's root node reports HasError() == true.
		err := WithTree("go", []byte("package main\nfunc f() {\n"), func(root *ts.Node, partial bool) error {
			gotPartial = partial
			return nil
		})
		if err != nil {
			t.Fatalf("WithTree returned error: %v", err)
		}
		if !gotPartial {
			t.Fatal("partial = false for a deliberately broken fixture; want true")
		}
		if releases != 1 {
			t.Errorf("release count = %d; want 1", releases)
		}
	})
}

// TestWithTree_UnknownLanguage asserts an unresolvable language name returns a non-nil error
// naming the language.
func TestWithTree_UnknownLanguage(t *testing.T) {
	err := WithTree("cobol", []byte("IDENTIFICATION DIVISION.\n"), func(root *ts.Node, partial bool) error {
		return nil
	})
	if err == nil {
		t.Fatal("WithTree(\"cobol\", ...) returned nil error; want a non-nil error")
	}
	if got := err.Error(); !strings.Contains(got, "cobol") {
		t.Errorf("WithTree(\"cobol\", ...) error = %q; want it to name the unresolved language", got)
	}
}
