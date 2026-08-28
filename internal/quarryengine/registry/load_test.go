// load_test.go table-drives LoadRegistry against t.TempDir fixtures, each writing its overlay file
// at a path of its own choosing and passing that path directly to LoadRegistry — the told-path
// signature card 15 introduced.

package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeServersYAML writes contents to path, creating any missing parent directories.
func writeServersYAML(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func TestLoadRegistry_AbsentFileYieldsBuiltins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")

	got, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry(absent) returned unexpected error: %v", err)
	}

	want := builtins()
	if len(got) != len(want) {
		t.Fatalf("LoadRegistry(absent) has %d entries; want %d", len(got), len(want))
	}
	for lang, wantEntry := range want {
		gotEntry, ok := got[lang]
		if !ok || gotEntry.Match != wantEntry.Match || len(gotEntry.Markers) != len(wantEntry.Markers) {
			t.Errorf("LoadRegistry(absent)[%q] = %+v; want %+v", lang, gotEntry, wantEntry)
		}
	}
}

func TestLoadRegistry_EmptyFileYieldsBuiltins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	writeServersYAML(t, path, "# comments only, no entries\n")

	got, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry(comments-only) returned unexpected error: %v", err)
	}
	if len(got) != len(builtins()) {
		t.Fatalf("LoadRegistry(comments-only) has %d entries; want %d (builtins unchanged)", len(got), len(builtins()))
	}
}

func TestLoadRegistry_FileOverridesWholeEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	writeServersYAML(t, path, `
go:
  markers: [go.mod, go.work]
  match: any
  command: [gopls, serve]
  install_hint: "brew install gopls"
`)

	got, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry(override) returned unexpected error: %v", err)
	}

	// The four other built-ins are still present alongside the override.
	for _, lang := range []string{"python", "csharp", "typescript", "rust"} {
		if _, ok := got[lang]; !ok {
			t.Errorf("LoadRegistry(override) missing built-in language %q", lang)
		}
	}

	goEntry, ok := got["go"]
	if !ok {
		t.Fatal("LoadRegistry(override) missing overridden language \"go\"")
	}
	if !stringSlicesEqual(goEntry.Markers, []string{"go.mod", "go.work"}) {
		t.Errorf("LoadRegistry(override)[\"go\"].Markers = %v; want [go.mod go.work]", goEntry.Markers)
	}
	if !stringSlicesEqual(goEntry.Command, []string{"gopls", "serve"}) {
		t.Errorf("LoadRegistry(override)[\"go\"].Command = %v; want [gopls serve]", goEntry.Command)
	}
	if goEntry.InstallHint != "brew install gopls" {
		t.Errorf("LoadRegistry(override)[\"go\"].InstallHint = %q; want %q", goEntry.InstallHint, "brew install gopls")
	}
}

func TestLoadRegistry_FileExtendsWithNewLanguage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	writeServersYAML(t, path, `
zig:
  markers: [build.zig]
  match: any
  command: [zls]
  install_hint: "install zls"
`)

	got, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry(extends) returned unexpected error: %v", err)
	}
	for lang := range builtins() {
		if _, ok := got[lang]; !ok {
			t.Errorf("LoadRegistry(extends) missing built-in language %q", lang)
		}
	}
	zig, ok := got["zig"]
	if !ok {
		t.Fatal("LoadRegistry(extends) missing new language \"zig\"")
	}
	if zig.Match != "any" || zig.Command[0] != "zls" {
		t.Errorf("LoadRegistry(extends)[\"zig\"] = %+v; want match any, command starting with zls", zig)
	}
}

func TestLoadRegistry_RejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name       string
		contents   string
		wantSubstr string
	}{
		{
			name: "unknown yaml field",
			contents: `
go:
  markers: [go.mod]
  match: any
  command: [gopls]
  install_hint: "go install gopls"
  weight: 5
`,
			wantSubstr: "field",
		},
		{
			name: "out-of-vocab match",
			contents: `
go:
  markers: [go.mod]
  match: sometimes
  command: [gopls]
  install_hint: "go install gopls"
`,
			wantSubstr: "invalid match",
		},
		{
			name:       "malformed yaml",
			contents:   "go: [this is not a map",
			wantSubstr: "parse",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "servers.yaml")
			writeServersYAML(t, path, tt.contents)

			_, err := LoadRegistry(path)
			if err == nil {
				t.Fatalf("LoadRegistry(%s) returned nil error; want error containing %q", tt.name, tt.wantSubstr)
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("LoadRegistry(%s) error = %q; want substring %q", tt.name, err.Error(), tt.wantSubstr)
			}
		})
	}
}

func TestLoadRegistry_InvalidEntryErrorNamesAliasAndPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	writeServersYAML(t, path, `
go:
  markers: [go.mod]
  match: sometimes
  command: [gopls]
  install_hint: "go install gopls"
`)

	_, err := LoadRegistry(path)
	if err == nil {
		t.Fatal("LoadRegistry(invalid match) returned nil error; want error naming the alias and path")
	}
	if !strings.Contains(err.Error(), `"go"`) {
		t.Errorf("LoadRegistry error = %q; want it to name the alias %q", err.Error(), "go")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("LoadRegistry error = %q; want it to name the file path %q", err.Error(), path)
	}
}

// TestLoadRegistry_PathIsDirectoryReturnsError proves the told-path signature does not silently
// fall back to built-ins when path exists but is a directory rather than a file: the absent-file
// fallback keys specifically on os.ErrNotExist, which a directory read does not produce.
func TestLoadRegistry_PathIsDirectoryReturnsError(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadRegistry(dir)
	if err == nil {
		t.Fatalf("LoadRegistry(%q) (a directory) returned nil error; want an error", dir)
	}
}

// TestLoadRegistry_DecodesInitializationOptionsBlock proves an initialization_options block decodes
// without tripping the decoder's KnownFields(true) setting.
func TestLoadRegistry_DecodesInitializationOptionsBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	writeServersYAML(t, path, `
go:
  markers: [go.mod]
  match: any
  command: [gopls]
  install_hint: "go install gopls"
  initialization_options:
    buildFlags: ["-tags={{tags}}"]
`)

	got, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry(initialization_options) returned unexpected error: %v", err)
	}
	goEntry, ok := got["go"]
	if !ok {
		t.Fatal(`LoadRegistry(initialization_options) missing language "go"`)
	}
	if goEntry.InitializationOptions == nil {
		t.Fatal(`LoadRegistry(initialization_options)["go"].InitializationOptions = nil; want a decoded template`)
	}
}

// TestLoadRegistry_EntryOmittingInitializationOptionsValidates proves an entry omitting the
// initialization_options key still validates -- the field is optional.
func TestLoadRegistry_EntryOmittingInitializationOptionsValidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	writeServersYAML(t, path, `
go:
  markers: [go.mod]
  match: any
  command: [gopls]
  install_hint: "go install gopls"
`)

	got, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry(no initialization_options) returned unexpected error: %v", err)
	}
	goEntry, ok := got["go"]
	if !ok {
		t.Fatal(`LoadRegistry(no initialization_options) missing language "go"`)
	}
	if goEntry.InitializationOptions != nil {
		t.Errorf(`LoadRegistry(no initialization_options)["go"].InitializationOptions = %v; want nil`, goEntry.InitializationOptions)
	}
}

// TestLoadRegistry_OverlayWholeReplacingGoDropsInitializationOptions proves the overlay hazard the
// ErrBuildTagsUnsupported hard error exists for: an overlay "go:" block that whole-replaces the
// built-in and omits initialization_options yields an entry whose InitializationOptions is nil, even
// though the built-in "go" entry itself carries a template.
func TestLoadRegistry_OverlayWholeReplacingGoDropsInitializationOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	writeServersYAML(t, path, `
go:
  markers: [go.mod, go.work]
  match: any
  command: [gopls, serve]
  install_hint: "brew install gopls"
`)

	got, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry(overlay drops initialization_options) returned unexpected error: %v", err)
	}
	goEntry, ok := got["go"]
	if !ok {
		t.Fatal(`LoadRegistry(overlay drops initialization_options) missing language "go"`)
	}
	if goEntry.InitializationOptions != nil {
		t.Errorf(`LoadRegistry(overlay drops initialization_options)["go"].InitializationOptions = %v; want nil (whole-replace, not merged with the built-in template)`, goEntry.InitializationOptions)
	}
}
