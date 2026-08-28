// initoptions_test.go covers RenderInitializationOptions's back-compat empty-tag-set path, the
// placeholder-substitution paths (map, list, key-untouched), the never-mutate-the-source guarantee,
// and the ErrBuildTagsUnsupported error paths.

package registry

import (
	"errors"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/quarryengine"
)

// TestRenderInitializationOptions_EmptyTagSetIsBackCompatNoOp is the back-compat assertion: an
// empty tag set against the template-bearing built-in "go" entry returns (nil, nil), whatever the
// entry's template holds. Written first, per the empty-tag-set-is-a-uniform-no-op Shared Decision.
func TestRenderInitializationOptions_EmptyTagSetIsBackCompatNoOp(t *testing.T) {
	goEntry := builtins()["go"]

	got, err := RenderInitializationOptions("go", goEntry, nil)
	if err != nil {
		t.Fatalf("RenderInitializationOptions(empty tags) returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("RenderInitializationOptions(empty tags) = %v; want nil", got)
	}
}

func TestRenderInitializationOptions_NestedMapWithListPlaceholderRenders(t *testing.T) {
	entry := Entry{
		InitializationOptions: map[string]any{
			"buildFlags": []any{"-tags={{tags}}"},
		},
	}

	got, err := RenderInitializationOptions("go", entry, []string{"b", "a"})
	if err != nil {
		t.Fatalf("RenderInitializationOptions() returned unexpected error: %v", err)
	}
	buildFlags, ok := got["buildFlags"].([]any)
	if !ok || len(buildFlags) != 1 {
		t.Fatalf("RenderInitializationOptions()[\"buildFlags\"] = %#v; want a one-element []any", got["buildFlags"])
	}
	want := "-tags=b,a"
	if buildFlags[0] != want {
		t.Errorf("RenderInitializationOptions()[\"buildFlags\"][0] = %v; want %q", buildFlags[0], want)
	}
}

func TestRenderInitializationOptions_SourceEntryMapUnchanged(t *testing.T) {
	entry := Entry{
		InitializationOptions: map[string]any{
			"buildFlags": []any{"-tags={{tags}}"},
		},
	}

	// Render twice, then compare the entry's own map against what it held
	// before either render -- if either render mutated entry.InitializationOptions in place, the
	// second render's placeholder would already be gone.
	before := "-tags={{tags}}"

	if _, err := RenderInitializationOptions("go", entry, []string{"a"}); err != nil {
		t.Fatalf("first RenderInitializationOptions() returned unexpected error: %v", err)
	}
	afterFirst := entry.InitializationOptions["buildFlags"].([]any)[0]
	if afterFirst != before {
		t.Fatalf("entry.InitializationOptions mutated after first render: got %v; want unchanged %q", afterFirst, before)
	}

	got2, err := RenderInitializationOptions("go", entry, []string{"b"})
	if err != nil {
		t.Fatalf("second RenderInitializationOptions() returned unexpected error: %v", err)
	}
	afterSecond := entry.InitializationOptions["buildFlags"].([]any)[0]
	if afterSecond != before {
		t.Fatalf("entry.InitializationOptions mutated after second render: got %v; want unchanged %q", afterSecond, before)
	}
	want2 := "-tags=b"
	if got2["buildFlags"].([]any)[0] != want2 {
		t.Errorf("second render buildFlags[0] = %v; want %q", got2["buildFlags"].([]any)[0], want2)
	}
}

func TestRenderInitializationOptions_MapKeyPlaceholderLeftAlone(t *testing.T) {
	entry := Entry{
		InitializationOptions: map[string]any{
			"{{tags}}": "unrelated-value",
			"real":     "{{tags}}",
		},
	}

	got, err := RenderInitializationOptions("go", entry, []string{"x"})
	if err != nil {
		t.Fatalf("RenderInitializationOptions() returned unexpected error: %v", err)
	}
	if _, ok := got["{{tags}}"]; !ok {
		t.Fatalf("RenderInitializationOptions() dropped the literal %q key; got %v", "{{tags}}", got)
	}
	if got["{{tags}}"] != "unrelated-value" {
		t.Errorf(`RenderInitializationOptions()["{{tags}}"] = %v; want "unrelated-value" (map keys are never substituted)`, got["{{tags}}"])
	}
	if got["real"] != "x" {
		t.Errorf(`RenderInitializationOptions()["real"] = %v; want "x"`, got["real"])
	}
}

func TestRenderInitializationOptions_PlaceholderFreeTemplateErrors(t *testing.T) {
	entry := Entry{
		InitializationOptions: map[string]any{
			"buildFlags": []any{"-tags=static"},
		},
	}

	_, err := RenderInitializationOptions("python", entry, []string{"x"})
	if err == nil {
		t.Fatal("RenderInitializationOptions(placeholder-free template) returned nil error; want ErrBuildTagsUnsupported")
	}
	if !errors.Is(err, quarryengine.ErrBuildTagsUnsupportedSentinel) {
		t.Errorf("RenderInitializationOptions() error = %v; want errors.Is(err, ErrBuildTagsUnsupportedSentinel)", err)
	}
	if !strings.Contains(err.Error(), `"python"`) {
		t.Errorf("RenderInitializationOptions() error = %q; want it to name the language %q", err.Error(), "python")
	}
}

func TestRenderInitializationOptions_NilTemplateErrors(t *testing.T) {
	var entry Entry // InitializationOptions is nil.

	_, err := RenderInitializationOptions("python", entry, []string{"x"})
	if err == nil {
		t.Fatal("RenderInitializationOptions(nil template) returned nil error; want ErrBuildTagsUnsupported")
	}
	if !errors.Is(err, quarryengine.ErrBuildTagsUnsupportedSentinel) {
		t.Errorf("RenderInitializationOptions() error = %v; want errors.Is(err, ErrBuildTagsUnsupportedSentinel)", err)
	}
	if !strings.Contains(err.Error(), `"python"`) {
		t.Errorf("RenderInitializationOptions() error = %q; want it to name the language %q", err.Error(), "python")
	}
}
