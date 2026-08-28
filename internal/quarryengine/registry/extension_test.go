// extension_test.go table-drives LanguageForExtension over the whole extension set plus its edge
// cases, and pins ExtensionLanguages/ExtensionsForLanguage against the same map.

package registry

import (
	"reflect"
	"testing"
)

func TestLanguageForExtension(t *testing.T) {
	tests := []struct {
		name   string
		ext    string
		want   string
		wantOK bool
	}{
		{name: "go", ext: ".go", want: "go", wantOK: true},
		{name: "python", ext: ".py", want: "python", wantOK: true},
		{name: "csharp", ext: ".cs", want: "csharp", wantOK: true},
		{name: "typescript", ext: ".ts", want: "typescript", wantOK: true},
		{name: "tsx", ext: ".tsx", want: "typescript", wantOK: true},
		{name: "rust", ext: ".rs", want: "rust", wantOK: true},
		{name: "unknown", ext: ".java", want: "", wantOK: false},
		{name: "dotless", ext: "go", want: "go", wantOK: true},
		{name: "uppercase", ext: ".GO", want: "go", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := LanguageForExtension(tt.ext)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("LanguageForExtension(%q) = (%q, %v); want (%q, %v)", tt.ext, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestLanguageForExtension_IgnoresDirectoryContext pins the reason this map exists separately
// from DetectLanguage: resolving ".ts" is unconditional, unlike marker-based detection which
// would resolve a .ts file inside a Go module to "go".
func TestLanguageForExtension_IgnoresDirectoryContext(t *testing.T) {
	got, ok := LanguageForExtension(".ts")
	if !ok || got != "typescript" {
		t.Errorf("LanguageForExtension(\".ts\") = (%q, %v); want (\"typescript\", true) regardless of directory context", got, ok)
	}
}

func TestExtensionLanguages(t *testing.T) {
	want := []string{"csharp", "go", "python", "rust", "typescript"}
	got := ExtensionLanguages()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtensionLanguages() = %v; want %v", got, want)
	}
}

func TestExtensionsForLanguage(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want []string
	}{
		{name: "typescript has two extensions", lang: "typescript", want: []string{".ts", ".tsx"}},
		{name: "go has one extension", lang: "go", want: []string{".go"}},
		{name: "unknown language returns nil", lang: "cobol", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtensionsForLanguage(tt.lang)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtensionsForLanguage(%q) = %v; want %v", tt.lang, got, tt.want)
			}
		})
	}
}
